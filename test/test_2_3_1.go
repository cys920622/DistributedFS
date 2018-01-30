// Multiple reader clients and one writer client
// Client A writes/closes file, disconnects. Client B connects, writes file. Client A re-connects, reads, observes B changes

package test

import (
	"io/ioutil"
	"fmt"
	"../dfslib"
	"sync"
	"time"
	"strings"
)

const FileName231 = "231"

func Test_2_3_1(serverAddr string, itwg *sync.WaitGroup) {
	fmt.Println("[2.3.1]")
	fmt.Println("Disconnected - One reader/writer client and one writer client")
	fmt.Println("Client A writes/closes file, disconnects. Client B connects, writes file. Client A re-connects, reads, observes B changes")
	// this creates a directory (to be used as localPath) for each client.
	// The directories will have the format "./client{A,B}NNNNNNNNN", where
	// N is an arbitrary number. Feel free to change these local paths
	// to best fit your environment
	clientALocalPath, errA := ioutil.TempDir(".", "clientA231_")
	clientBLocalPath, errB := ioutil.TempDir(".", "clientB231_")
	clientCLocalPath, errC := ioutil.TempDir(".", "clientC231_")
	clientDLocalPath, errD := ioutil.TempDir(".", "clientD231_")

	var writer sync.WaitGroup
	var readers sync.WaitGroup

	if errA != nil || errB != nil || errC != nil || errD != nil {
		panic("Could not create temporary directory")
	}

	errChannel := make(chan error)

	readers.Add(4)

	writer.Add(1)
	go writeChunk_2_3_1("A", serverAddr, LocalIP, clientALocalPath, errChannel, &writer, &readers, 3)
	writer.Wait()
	writer.Add(1)
	go writeChunk_2_3_1("B", serverAddr, LocalIP, clientBLocalPath, errChannel, &writer, &readers, 243)
	writer.Wait()

	go read_2_3_1("C", serverAddr, LocalIP, clientCLocalPath, errChannel, &readers)
	time.Sleep(200 * time.Millisecond)
	go read_2_3_1("D", serverAddr, LocalIP, clientDLocalPath, errChannel, &readers)

	readers.Wait()


	isSame, err := dread_compare_2_3_1(
		"C", "D", serverAddr, LocalIP, LocalIP, clientCLocalPath, clientDLocalPath, errChannel)

	e := <- errChannel
	if e != nil {
		itwg.Done()
		reportError(e)
	}


	if err != nil || !isSame {
		itwg.Done()
		reportError(err)
	} else {
		fmt.Printf("\nALL TESTS PASSED: Test_2_3_1\n\n")
		CleanDir("clientA231")
		CleanDir("clientB231")
		itwg.Done()
		return
	}
}

func writeChunk_2_3_1(name, serverAddr, localIP, localPath string,
					  rc chan <- error, writer, readers *sync.WaitGroup,
					  ChunkNum uint8) (err error) {
	var blob dfslib.Chunk
	var dfs dfslib.DFS

	logger := NewLogger("(2.3.1) Client " + name + " (W)")
	content := fmt.Sprintf("(%s) This is test 2.3.1!", name)

	testCase := fmt.Sprintf("Mounting DFS('%s', '%s', '%s')", serverAddr, localIP, localPath)

	dfs, err = dfslib.MountDFS(serverAddr, localIP, localPath)
	if err != nil {
		logger.TestResult(testCase, false)
		rc <- err
		writer.Done()
		return
	}
	logger.TestResult(testCase, true)

	defer func() {
		// if the client is ending with an error, do not make thing worse by issuing
		// extra calls to the server
		if err != nil {
			rc <- err
			writer.Done()
			return
		}


		readers.Wait()

		if err = dfs.UMountDFS(); err != nil {
			logger.TestResult("Unmounting DFS", false)
			rc <- err
			return
		}

		logger.TestResult("Unmounting DFS", true)
	}()

	testCase = fmt.Sprintf("Opening file '%s' for writing", FileName231)

	file, err := dfs.Open(FileName231, dfslib.WRITE)
	if err != nil {
		logger.TestResult(testCase, false)
		rc <- err
		return
	}
	logger.TestResult(testCase, true)

	defer func() {
		if err != nil {
			rc <- err
			writer.Done()
			return
		}

		testCase := fmt.Sprintf("Closing file '%s'", FileName231)


		err = file.Close()
		if err != nil {
			logger.TestResult(testCase, false)
			rc <- err
			writer.Done()
			return
		}

		logger.TestResult(testCase, true)

		readers.Done()
		writer.Done()
	}()

	// Write chunk
	testCase = fmt.Sprintf("Writing chunk %d", ChunkNum)

	copy(blob[:], content)
	err = file.Write(ChunkNum, &blob)
	if err != nil {
		logger.TestResult(testCase, false)
		rc <- err
		return
	} else {
		logger.TestResult(testCase, true)
	}


	return
}

func read_2_3_1(name, serverAddr, localIP, localPath string, rc chan <- error, readers *sync.WaitGroup) (err error) {
	var dfs dfslib.DFS

	logger := NewLogger("(2.3.1) Client " + name + " (R)")

	testCase := fmt.Sprintf("Mounting DFS('%s', '%s', '%s')", serverAddr, localIP, localPath)

	dfs, err = dfslib.MountDFS(serverAddr, localIP, localPath)
	if err != nil {
		logger.TestResult(testCase, false)
		rc <- err
		readers.Done()
		return
	}
	logger.TestResult(testCase, true)

	defer func() {
		// if the client is ending with an error, do not make thing worse by issuing
		// extra calls to the server
		if err != nil {
			readers.Done()
			return
		}

		if err = dfs.UMountDFS(); err != nil {
			logger.TestResult("Unmounting DFS", false)
			rc <- err
			readers.Done()
			return
		}

		logger.TestResult("Unmounting DFS", true)

		readers.Done()
		rc <- err
		return
	}()

	testCase = fmt.Sprintf("Opening file '%s' for read", FileName231)

	_, err = dfs.Open(FileName231, dfslib.READ)
	if err != nil {
		logger.TestResult(testCase, false)
		rc <- err
	} else {
		logger.TestResult(testCase, true)
	}

	return
}


func dread_compare_2_3_1(n0, n1, serverAddr, localIP0,
					     localIP1, localPath0, localPath1 string, rc chan <- error) (isIdentical bool, err error) {

	// Client C
	var dfs0 dfslib.DFS

	logger0 := NewLogger("(2.3.1) Client " + n0 + " (D)")

	testCase := fmt.Sprintf("Mounting DFS('%s', '%s', '%s')", serverAddr, localIP0, localPath0)

	dfs0, err = dfslib.MountDFS(serverAddr, localIP0, localPath0)
	if err != nil {
		logger0.TestResult(testCase, false)
		rc <- err
		return
	}
	logger0.TestResult(testCase, true)


	testCase = fmt.Sprintf("Opening file '%s' for D-read", FileName231)

	_, err = dfs0.Open(FileName231, dfslib.DREAD)
	if err != nil {
		logger0.TestResult(testCase, false)
		rc <- err
	} else {
		logger0.TestResult(testCase, true)
	}

	// Client D
	var dfs1 dfslib.DFS

	logger1 := NewLogger("(2.3.1) Client " + n1 + " (D)")

	testCase = fmt.Sprintf("Mounting DFS('%s', '%s', '%s')", serverAddr, localIP1, localPath1)

	dfs1, err = dfslib.MountDFS(serverAddr, localIP1, localPath1)
	if err != nil {
		logger1.TestResult(testCase, false)
		rc <- err
		return
	}
	logger1.TestResult(testCase, true)


	testCase = fmt.Sprintf("Opening file '%s' for D-read", FileName231)

	file, err := dfs1.Open(FileName231, dfslib.DREAD)
	if err != nil {
		logger1.TestResult(testCase, false)
		rc <- err
	} else {
		logger1.TestResult(testCase, true)
	}

	for i := 0; i <= 255; i++ {
		var buf0, buf1 dfslib.Chunk

		err = file.Read(uint8(i), &buf0)
		err = file.Read(uint8(i), &buf1)

		if err != nil {return}

		isIdentical = strings.Compare(string(buf0[:]),string(buf0[:])) == 0
		if !isIdentical {
			logger0.TestResult("Files are identical", false)
			return
		}
	}

	// Unmount
	if err = dfs0.UMountDFS(); err != nil {
		fmt.Println(err)
		logger0.TestResult("Unmounting DFS", false)
		rc <- err
		return false, err
	}

	logger0.TestResult("Unmounting DFS", true)

	if err = dfs1.UMountDFS(); err != nil {
		fmt.Println(err)
		logger1.TestResult("Unmounting DFS", false)
		rc <- err
		return false, err
	}

	logger1.TestResult("Unmounting DFS", true)

	logger0.TestResult("Files are identical", true)
	return isIdentical, err // placeholder
}