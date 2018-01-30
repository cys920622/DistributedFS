// Multiple reader clients and one writer client
// Client A writes file, client B,C,D reads same file and observe A's write

package test

import (
	"io/ioutil"
	"fmt"
	"../dfslib"
	"sync"
	"time"
)

const FileName132 = "132"

func Test_1_3_2(serverAddr string, itwg *sync.WaitGroup) {
	fmt.Println("[1.3.2]")
	fmt.Println("One reader/writer client and one writer client")
	fmt.Println("Client A writes file, client B reads same file and observes A's write")
	// this creates a directory (to be used as localPath) for each client.
	// The directories will have the format "./client{A,B}NNNNNNNNN", where
	// N is an arbitrary number. Feel free to change these local paths
	// to best fit your environment
	clientALocalPath, errA := ioutil.TempDir(".", "clientA132_")
	clientBLocalPath, errB := ioutil.TempDir(".", "clientB132_")
	clientCLocalPath, errB := ioutil.TempDir(".", "clientC132_")
	clientDLocalPath, errB := ioutil.TempDir(".", "clientD132_")

	var writer sync.WaitGroup
	var readers sync.WaitGroup
	writer.Add(1)
	readers.Add(3)

	if errA != nil || errB != nil {
		panic("Could not create temporary directory")
	}

	errChannel := make(chan error)

	go clientA_1_3_2(serverAddr, LocalIP, clientALocalPath, errChannel, &writer, &readers)

	writer.Wait()
	time.Sleep(200 * time.Millisecond)
	go reader_1_3_2("B", serverAddr, LocalIP, clientBLocalPath, errChannel, &readers)
	time.Sleep(200 * time.Millisecond)
	go reader_1_3_2("C", serverAddr, LocalIP, clientCLocalPath, errChannel, &readers)
	time.Sleep(200 * time.Millisecond)
	go reader_1_3_2("D", serverAddr, LocalIP, clientDLocalPath, errChannel, &readers)

	readers.Wait()

	e := <- errChannel
	if e != nil {
		itwg.Done()
		reportError(e)
	}

	fmt.Printf("\nALL TESTS PASSED: Test_1_3_2\n\n")
	CleanDir("clientA132")
	CleanDir("clientB132")
	itwg.Done()
	return
}

func clientA_1_3_2(serverAddr, localIP, localPath string, rc chan <- error, writer, readers *sync.WaitGroup) (err error) {
	var blob dfslib.Chunk
	var dfs dfslib.DFS

	logger := NewLogger("(1.3.2) Client A")
	content := "This is test 1.3.2!"

	testCase := fmt.Sprintf("Mounting DFS('%s', '%s', '%s')", serverAddr, localIP, localPath)

	dfs, err = dfslib.MountDFS(serverAddr, localIP, localPath)
	if err != nil {
		logger.TestResult(testCase, false)
		rc <- err
		return
	}
	logger.TestResult(testCase, true)

	defer func() {
		// if the client is ending with an error, do not make thing worse by issuing
		// extra calls to the server
		if err != nil {
			rc <- err
			return
		}

		if err = dfs.UMountDFS(); err != nil {
			logger.TestResult("Unmounting DFS", false)
			rc <- err
			return
		}

		logger.TestResult("Unmounting DFS", true)

	}()

	testCase = fmt.Sprintf("Opening file '%s' for writing", FileName132)

	file, err := dfs.Open(FileName132, dfslib.WRITE)
	if err != nil {
		logger.TestResult(testCase, false)
		rc <- err
		return
	}
	logger.TestResult(testCase, true)

	defer func() {
		if err != nil {
			rc <- err
			return
		}

		testCase := fmt.Sprintf("Closing file '%s'", FileName132)

		writer.Done()
		readers.Wait()

		err = file.Close()
		if err != nil {
			logger.TestResult(testCase, false)
			rc <- err
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

func reader_1_3_2(name, serverAddr, localIP, localPath string, rc chan <- error, readers *sync.WaitGroup) (err error) {
	var dfs dfslib.DFS

	logger := NewLogger("(1.3.2) Client " + name)

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
			return
		}

		logger.TestResult("Unmounting DFS", true)

		rc <- err
		return
	}()

	testCase = fmt.Sprintf("Opening file '%s' for read", FileName132)

	file, err := dfs.Open(FileName132, dfslib.READ)
	if err != nil {
		logger.TestResult(testCase, false)
		rc <- err
	} else {
		logger.TestResult(testCase, true)
	}

	//

	content := "This is test 1.3.2!"
	var blob dfslib.Chunk
	testCase = fmt.Sprintf("Able to read '%s' back from chunk %d", content, CHUNKNUM)

	err = file.Read(CHUNKNUM, &blob)

	if err != nil {
		logger.TestResult(testCase, false)
		fmt.Println(err)
		rc <- err
		readers.Done()
		return
	}

	str := string(blob[:len(content)])
	if str != content {
		logger.TestResult(testCase, false)
		rc <- err
		readers.Done()
		return fmt.Errorf("Reading from chunk %d. Expected: '%s'; got: '%s'", CHUNKNUM, content, str)
	} else {
		logger.TestResult(testCase, true)
		readers.Done()
	}

	return
}
