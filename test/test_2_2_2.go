// Multiple reader clients and one writer client
// Client A writes/closes file, disconnects. Client B connects, writes file. Client A re-connects, reads, observes B changes

package test

import (
	"io/ioutil"
	"fmt"
	"../dfslib"
	"sync"
	"time"
)

const FileName222 = "222"

func Test_2_2_2(serverAddr string, itwg *sync.WaitGroup) {
	fmt.Println("[2.2.2]")
	fmt.Println("Disconnected - One reader/writerA client and one writerA client")
	fmt.Println("Client A writes/closes file, disconnects. Client B connects, writes file. Client A re-connects, reads, observes B changes")
	// this creates a directory (to be used as localPath) for each client.
	// The directories will have the format "./client{A,B}NNNNNNNNN", where
	// N is an arbitrary number. Feel free to change these local paths
	// to best fit your environment
	clientALocalPath, errA := ioutil.TempDir(".", "clientA222_")
	clientBLocalPath, errB := ioutil.TempDir(".", "clientB222_")

	var writerA sync.WaitGroup
	var writerB sync.WaitGroup
	writerA.Add(1)
	writerB.Add(1)

	if errA != nil || errB != nil {
		panic("Could not create temporary directory")
	}

	errChannel := make(chan error)

	go clientA_2_2_2(serverAddr, LocalIP, clientALocalPath, errChannel, &writerA, &writerB)
	writerA.Wait()
	go clientBOpenRead_2_2_2(serverAddr, LocalIP, clientBLocalPath, errChannel, &writerB)
	writerB.Wait()
	time.Sleep(200 * time.Millisecond)
	writerA.Add(1)
	writerB.Add(1)
	go clientB_2_2_2(serverAddr, LocalIP, clientBLocalPath, errChannel, &writerA, &writerB)
	time.Sleep(200 * time.Millisecond)
	writerB.Wait()
	go clientAVerifyBWrite_2_2_2(serverAddr, LocalIP, clientBLocalPath, errChannel, &writerA)


	e := <- errChannel
	if e != nil {
		itwg.Done()
		reportError(e)
	}

	writerA.Wait()
	//writerB.Wait()

	fmt.Printf("\nALL TESTS PASSED: Test_2_2_2\n\n")
	CleanDir("clientA222")
	CleanDir("clientB222")
	itwg.Done()
	return
}

func clientA_2_2_2(serverAddr, localIP, localPath string, rc chan <- error, writerA, writerB *sync.WaitGroup) (err error) {
	var blob dfslib.Chunk
	var dfs dfslib.DFS

	logger := NewLogger("(2.2.2) Client A (W)")
	content := "(A) This is test 2.2.2!"

	testCase := fmt.Sprintf("Mounting DFS('%s', '%s', '%s')", serverAddr, localIP, localPath)

	dfs, err = dfslib.MountDFS(serverAddr, localIP, localPath)
	if err != nil {
		logger.TestResult(testCase, false)
		rc <- err
		writerA.Done()
		return
	}
	logger.TestResult(testCase, true)

	defer func() {
		// if the client is ending with an error, do not make thing worse by issuing
		// extra calls to the server
		if err != nil {
			rc <- err
			writerA.Done()
			return
		}



		if err = dfs.UMountDFS(); err != nil {
			logger.TestResult("Unmounting DFS", false)
			rc <- err
			return
		}

		logger.TestResult("Unmounting DFS", true)
	}()

	testCase = fmt.Sprintf("Opening file '%s' for writing", FileName222)

	file, err := dfs.Open(FileName222, dfslib.WRITE)
	if err != nil {
		logger.TestResult(testCase, false)
		rc <- err
		return
	}
	logger.TestResult(testCase, true)

	defer func() {
		if err != nil {
			rc <- err
			writerA.Done()
			return
		}

		testCase := fmt.Sprintf("Closing file '%s'", FileName222)

		writerA.Done()
		writerB.Wait()

		err = file.Close()
		if err != nil {
			logger.TestResult(testCase, false)
			rc <- err
			writerA.Done()
			return
		}

		logger.TestResult(testCase, true)
	}()

	// Write chunk
	testCase = fmt.Sprintf("Writing chunk %d", CHUNKNUM)

	copy(blob[:], content)
	err = file.Write(CHUNKNUM, &blob)
	if err != nil {
		logger.TestResult(testCase, false)
		rc <- err
		return
	} else {
		logger.TestResult(testCase, true)
	}


	return
}

func clientAVerifyBWrite_2_2_2(serverAddr, localIP, localPath string, rc chan <- error, writerA *sync.WaitGroup) (err error) {
	var dfs dfslib.DFS

	logger := NewLogger("(2.2.2) Client A (R)")

	testCase := fmt.Sprintf("Mounting DFS('%s', '%s', '%s')", serverAddr, localIP, localPath)

	dfs, err = dfslib.MountDFS(serverAddr, localIP, localPath)
	if err != nil {
		logger.TestResult(testCase, false)
		rc <- err
		writerA.Done()
		return
	}
	logger.TestResult(testCase, true)

	defer func() {
		// if the client is ending with an error, do not make thing worse by issuing
		// extra calls to the server
		if err != nil {
			writerA.Done()
			return
		}

		if err = dfs.UMountDFS(); err != nil {
			logger.TestResult("Unmounting DFS", false)
			rc <- err
			writerA.Done()
			return
		}

		logger.TestResult("Unmounting DFS", true)

		writerA.Done()
		rc <- err
		return
	}()

	testCase = fmt.Sprintf("Opening file '%s' for read", FileName222)

	file, err := dfs.Open(FileName222, dfslib.READ)
	if err != nil {
		logger.TestResult(testCase, false)
		rc <- err
	} else {
		logger.TestResult(testCase, true)
	}

	content := "Client B overwrote this file!"
	var blob dfslib.Chunk
	testCase = fmt.Sprintf("Able to read '%s' back from chunk %d", content, CHUNKNUM)

	err = file.Read(CHUNKNUM, &blob)

	if err != nil {
		logger.TestResult(testCase, false)
		rc <- err
		writerA.Done()
		return
	}

	str := string(blob[:len(content)])
	if str != content {
		logger.TestResult(testCase, false)
		rc <- err
		writerA.Done()
		return fmt.Errorf("Reading from chunk %d. Expected: '%s'; got: '%s'", CHUNKNUM, content, str)
	} else {
		logger.TestResult(testCase, true)
	}

	return

}


func clientBOpenRead_2_2_2(serverAddr, localIP, localPath string, rc chan <- error, writerB *sync.WaitGroup) (err error) {
	var dfs dfslib.DFS

	logger := NewLogger("(2.2.2) Client B (R)")

	testCase := fmt.Sprintf("Mounting DFS('%s', '%s', '%s')", serverAddr, localIP, localPath)

	dfs, err = dfslib.MountDFS(serverAddr, localIP, localPath)
	if err != nil {
		logger.TestResult(testCase, false)
		rc <- err
		writerB.Done()
		return
	}
	logger.TestResult(testCase, true)

	defer func() {
		// if the client is ending with an error, do not make thing worse by issuing
		// extra calls to the server
		if err != nil {
			writerB.Done()
			return
		}

		if err = dfs.UMountDFS(); err != nil {
			logger.TestResult("Unmounting DFS", false)
			rc <- err
			writerB.Done()
			return
		}

		logger.TestResult("Unmounting DFS", true)

		writerB.Done()
		rc <- err
		return
	}()

	testCase = fmt.Sprintf("Opening file '%s' for read", FileName222)

	_, err = dfs.Open(FileName222, dfslib.READ)
	if err != nil {
		logger.TestResult(testCase, false)
		rc <- err
	} else {
		logger.TestResult(testCase, true)
	}
	return
}

func clientB_2_2_2(serverAddr, localIP, localPath string, rc chan <- error, writerA, writerB *sync.WaitGroup) (err error) {
	var dfs dfslib.DFS

	logger := NewLogger("(2.2.2) Client B (W)")

	testCase := fmt.Sprintf("Mounting DFS('%s', '%s', '%s')", serverAddr, localIP, localPath)

	dfs, err = dfslib.MountDFS(serverAddr, localIP, localPath)
	if err != nil {
		logger.TestResult(testCase, false)
		rc <- err
		writerB.Done()
		return
	}
	logger.TestResult(testCase, true)

	defer func() {
		// if the client is ending with an error, do not make thing worse by issuing
		// extra calls to the server
		if err != nil {
			rc <- err
			writerB.Done()
			return
		}

		if err = dfs.UMountDFS(); err != nil {
			logger.TestResult("Unmounting DFS", false)
			rc <- err
			return
		}

		logger.TestResult("Unmounting DFS", true)
	}()

	testCase = fmt.Sprintf("Opening file '%s' for writing", FileName222)

	file, err := dfs.Open(FileName222, dfslib.WRITE)
	if err != nil {
		logger.TestResult(testCase, false)
		rc <- err
		writerB.Done()
		return
	}
	logger.TestResult(testCase, true)

	defer func() {
		if err != nil {
			rc <- err
			writerB.Done()
			return
		}

		testCase := fmt.Sprintf("Closing file '%s'", FileName222)


		err = file.Close()
		if err != nil {
			logger.TestResult(testCase, false)
			rc <- err
			writerB.Done()
			return
		}

		logger.TestResult(testCase, true)

		writerB.Done() // Start readerA
		writerA.Wait() // Wait until readerA is done
	}()

	// Write chunk
	var blob dfslib.Chunk
	content := "Client B overwrote this file!"

	testCase = fmt.Sprintf("Writing chunk %d", CHUNKNUM)

	copy(blob[:], content)
	err = file.Write(CHUNKNUM, &blob)
	if err != nil {
		logger.TestResult(testCase, false)
		rc <- err
		return
	} else {
		logger.TestResult(testCase, true)
	}


	return
}
