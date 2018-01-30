// One reader/writer client and one writer client
// Client A opens file F for writing, disconnects. Client B connects and opens F for writing, succeeds

package test

import (
	"io/ioutil"
	"fmt"
	"../dfslib"
	"sync"
)

const FileName221 = "221"

func Test_2_2_1(serverAddr string, wg *sync.WaitGroup) {
	fmt.Println("[2.2.1]")
	fmt.Println("Disconnected - One reader/writer client and one writer client")
	fmt.Println("Client A opens file F for writing, disconnects. Client B connects and opens F for writing, succeeds")
	// this creates a directory (to be used as localPath) for each client.
	// The directories will have the format "./client{A,B}NNNNNNNNN", where
	// N is an arbitrary number. Feel free to change these local paths
	// to best fit your environment
	clientALocalPath, errA := ioutil.TempDir(".", "clientA221_")
	clientBLocalPath, errB := ioutil.TempDir(".", "clientB221_")


	if errA != nil || errB != nil {
		panic("Could not create temporary directory")
	}


	err := clientA_2_2_1(serverAddr, LocalIP, clientALocalPath)
	if err != nil {
		wg.Done()
		reportError(err)
	}
	//time.Sleep(500*time.Millisecond)
	err = clientB_2_2_1(serverAddr, LocalIP, clientBLocalPath)
	if err != nil {
		wg.Done()
		reportError(err)
	}

	fmt.Printf("\nALL TESTS PASSED: Test_2_2_1\n\n")
	CleanDir("clientA221")
	CleanDir("clientB221")
	wg.Done()
	return
}

func clientA_2_2_1(serverAddr, localIP, localPath string) (err error) {
	var dfs dfslib.DFS

	logger := NewLogger("(2.2.1) Client A (writer)")

	testCase := fmt.Sprintf("Mounting DFS('%s', '%s', '%s')", serverAddr, localIP, localPath)

	dfs, err = dfslib.MountDFS(serverAddr, localIP, localPath)
	if err != nil {
		logger.TestResult(testCase, false)
		return
	}
	logger.TestResult(testCase, true)

	defer func() {
		// if the client is ending with an error, do not make thing worse by issuing
		// extra calls to the server
		if err != nil {
			return
		}

		if err = dfs.UMountDFS(); err != nil {
			logger.TestResult("Unmounting DFS", false)
		} else {
			logger.TestResult("Unmounting DFS", true)
		}
		return

	}()

	testCase = fmt.Sprintf("Opening file '%s' for writing", FileName221)

	_, err = dfs.Open(FileName221, dfslib.WRITE)
	if err != nil {
		logger.TestResult(testCase, false)
		return
	}
	logger.TestResult(testCase, true)
	return
}

func clientA__2_2_1_Read(serverAddr, localIP, localPath string) (err error) {
	// READ PHASE
	var blob dfslib.Chunk
	newContent := "Client B overwrote this file!"

	logger := NewLogger("(2.2.1) Client A (reader)")

	testCase := fmt.Sprintf("Mounting DFS('%s', '%s', '%s')", serverAddr, localIP, localPath)

	dfs, err := dfslib.MountDFS(serverAddr, localIP, localPath)
	if err != nil {
		logger.TestResult(testCase, false)
		return
	}
	logger.TestResult(testCase, true)

	testCase = fmt.Sprintf("Opening file '%s' for read", FileName221)

	file, err := dfs.Open(FileName221, dfslib.READ)
	if err != nil {
		logger.TestResult(testCase, false)
	} else {
		logger.TestResult(testCase, true)
	}

	//

	testCase = fmt.Sprintf("Able to read '%s' back from chunk %d", newContent, CHUNKNUM)

	err = file.Read(CHUNKNUM, &blob)

	if err != nil {
		logger.TestResult(testCase, false)
		fmt.Println(err)
		return
	}

	str := string(blob[:len(newContent)])
	if str != newContent {
		logger.TestResult(testCase, false)
		return fmt.Errorf("Reading from chunk %d. Expected: '%s'; got: '%s'", CHUNKNUM, newContent, str)
	} else {
		logger.TestResult(testCase, true)
	}

	return
}

func clientB_2_2_1(serverAddr, localIP, localPath string) (err error) {
	var dfs dfslib.DFS

	logger := NewLogger("(2.2.1) Client A (writer)")

	testCase := fmt.Sprintf("Mounting DFS('%s', '%s', '%s')", serverAddr, localIP, localPath)

	dfs, err = dfslib.MountDFS(serverAddr, localIP, localPath)
	if err != nil {
		logger.TestResult(testCase, false)
		return
	}
	logger.TestResult(testCase, true)

	defer func() {
		// if the client is ending with an error, do not make thing worse by issuing
		// extra calls to the server
		if err != nil {
			return
		}

		if err = dfs.UMountDFS(); err != nil {
			logger.TestResult("Unmounting DFS", false)
		} else {
			logger.TestResult("Unmounting DFS", true)
		}
		return

	}()

	testCase = fmt.Sprintf("Opening file '%s' for writing", FileName221)

	_, err = dfs.Open(FileName221, dfslib.WRITE)
	if err != nil {
		logger.TestResult(testCase, false)
		return
	}
	logger.TestResult(testCase, true)
	return
}
