// One reader/writer client and one writer client
// Client A writes file, client B writes same file, client A observers B's write

package test

import (
	"io/ioutil"
	"fmt"
	"../dfslib"
	"time"
	"sync"
)

const FileName124 = "124"

func Test_1_2_4(serverAddr string, wg *sync.WaitGroup) {
	fmt.Println("[1.2.4]")
	fmt.Println("One reader/writer client and one writer client")
	fmt.Println("Client A writes file, client B writes same file, client A observers B's write")
	// this creates a directory (to be used as localPath) for each client.
	// The directories will have the format "./client{A,B}NNNNNNNNN", where
	// N is an arbitrary number. Feel free to change these local paths
	// to best fit your environment
	clientALocalPath, errA := ioutil.TempDir(".", "clientA124_")
	clientBLocalPath, errB := ioutil.TempDir(".", "clientB124_")


	if errA != nil || errB != nil {
		panic("Could not create temporary directory")
	}

	errChannel := make(chan error)

	go clientA_1_2_4(serverAddr, LocalIP, clientALocalPath, errChannel)
	time.Sleep(500*time.Millisecond)
	go clientB_1_2_4(serverAddr, LocalIP, clientBLocalPath, errChannel)

	e := <- errChannel
	if e != nil {
		wg.Done()
		reportError(e)
	} else {
		fmt.Printf("\nALL TESTS PASSED: Test_1_2_4\n\n")
		CleanDir("clientA124")
		CleanDir("clientB124")
		wg.Done()
		return
	}

}

func clientA_1_2_4(serverAddr, localIP, localPath string, rc chan <- error) (err error) {
	newContent := "Client B overwrote this file!"
	var blob dfslib.Chunk
	var dfs dfslib.DFS

	logger := NewLogger("(1.2.4) Client A")
	content := "This is test 1.2.4!"

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
		} else {
			logger.TestResult("Unmounting DFS", true)
		}
		rc <- err
		return

	}()

	testCase = fmt.Sprintf("Opening file '%s' for writing", FileName124)

	file, err := dfs.Open(FileName124, dfslib.WRITE)
	if err != nil {
		logger.TestResult(testCase, false)
		rc <- err
		return
	}
	logger.TestResult(testCase, true)


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

	testCase = fmt.Sprintf("Closing file '%s'", FileName124)

	err = file.Close()
	if err != nil {
		logger.TestResult(testCase, false)
		rc <- err
		return
	}

	logger.TestResult(testCase, true)

	// Wait for Client B to write
	time.Sleep(1000 * time.Millisecond)


	// READ PHASE


	testCase = fmt.Sprintf("Opening file '%s' for read", FileName124)

	file, err = dfs.Open(FileName124, dfslib.READ)
	if err != nil {
		logger.TestResult(testCase, false)
		rc <- err
	} else {
		logger.TestResult(testCase, true)
	}

	//

	testCase = fmt.Sprintf("Able to read '%s' back from chunk %d", newContent, CHUNKNUM)

	err = file.Read(CHUNKNUM, &blob)

	if err != nil {
		logger.TestResult(testCase, false)
		fmt.Println(err)
		rc <- err
		return
	}

	str := string(blob[:len(newContent)])
	if str != newContent {
		logger.TestResult(testCase, false)
		rc <- err
		return fmt.Errorf("Reading from chunk %d. Expected: '%s'; got: '%s'", CHUNKNUM, newContent, str)
	} else {
		logger.TestResult(testCase, true)
	}

	return

}

func clientB_1_2_4(serverAddr, localIP, localPath string, rc chan <- error) (err error) {
	var blob dfslib.Chunk
	var dfs dfslib.DFS

	logger := NewLogger("(1.2.4) Client B")
	newContent := "Client B overwrote this file!"

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

	testCase = fmt.Sprintf("Opening file '%s' for writing", FileName124)

	file, err := dfs.Open(FileName124, dfslib.WRITE)
	if err != nil {
		logger.TestResult(testCase, false)
		rc <- err
		return
	}
	logger.TestResult(testCase, true)


	// Write chunk
	testCase = fmt.Sprintf("Writing chunk %d", CHUNKNUM)

	copy(blob[:], newContent)
	err = file.Write(CHUNKNUM, &blob)
	if err != nil {
		logger.TestResult(testCase, false)
		rc <- err
		return
	} else {
		logger.TestResult(testCase, true)
	}

	testCase = fmt.Sprintf("Closing file '%s'", FileName124)

	err = file.Close()
	if err != nil {
		logger.TestResult(testCase, false)
		rc <- err
		return
	}

	logger.TestResult(testCase, true)

	// Wait for Client A to read after B's write
	time.Sleep(2000 * time.Millisecond)

	return
}
