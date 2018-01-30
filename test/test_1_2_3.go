// One reader/writer client and one writer client
// Client A writes file, client B reads same file and observes A's write

package test

import (
	"io/ioutil"
	"fmt"
	"../dfslib"
	"time"
	"sync"
)

const FileName123 = "123"

func Test_1_2_3(serverAddr string, wg *sync.WaitGroup) {
	fmt.Println("[1.2.3]")
	fmt.Println("One reader/writer client and one writer client")
	fmt.Println("Client A writes file, client B reads same file and observes A's write")
	// this creates a directory (to be used as localPath) for each client.
	// The directories will have the format "./client{A,B}NNNNNNNNN", where
	// N is an arbitrary number. Feel free to change these local paths
	// to best fit your environment
	clientALocalPath, errA := ioutil.TempDir(".", "clientA123_")
	clientBLocalPath, errB := ioutil.TempDir(".", "clientB123_")


	if errA != nil || errB != nil {
		panic("Could not create temporary directory")
	}

	errChannel := make(chan error)

	go clientA_1_2_3(serverAddr, LocalIP, clientALocalPath, errChannel)
	time.Sleep(500*time.Millisecond)
	go clientB_1_2_3(serverAddr, LocalIP, clientBLocalPath, errChannel)

	e := <- errChannel
	if e != nil {
		wg.Done()
		reportError(e)
	} else {
		fmt.Printf("\nALL TESTS PASSED: Test_1_2_3\n\n")
		CleanDir("clientA123")
		CleanDir("clientB123")
		wg.Done()
		return
	}

}

func clientA_1_2_3(serverAddr, localIP, localPath string, rc chan <- error) (err error) {
	var blob dfslib.Chunk
	var dfs dfslib.DFS

	logger := NewLogger("(1.2.3) Client A")
	content := "This is test 1.2.3!"

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

	testCase = fmt.Sprintf("Opening file '%s' for writing", FileName123)

	file, err := dfs.Open(FileName123, dfslib.WRITE)
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

		testCase := fmt.Sprintf("Closing file '%s'", FileName123)

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


	time.Sleep(1500 * time.Millisecond)

	return
}

func clientB_1_2_3(serverAddr, localIP, localPath string, rc chan <- error) (err error) {
	var dfs dfslib.DFS

	logger := NewLogger("(1.2.3) Client B")

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
			return
		}

		if err = dfs.UMountDFS(); err != nil {
			logger.TestResult("Unmounting DFS", false)
			rc <- err
			return
		}

		logger.TestResult("Unmounting DFS", true)

		time.Sleep(1500 * time.Millisecond)

		rc <- err
	}()

	testCase = fmt.Sprintf("Opening file '%s' for read", FileName123)

	file, err := dfs.Open(FileName123, dfslib.READ)
	if err != nil {
		logger.TestResult(testCase, false)
		rc <- err
	} else {
		logger.TestResult(testCase, true)
	}

	//

	content := "This is test 1.2.3!"
	var blob dfslib.Chunk
	testCase = fmt.Sprintf("Able to read '%s' back from chunk %d", content, CHUNKNUM)

	err = file.Read(CHUNKNUM, &blob)

	if err != nil {
		logger.TestResult(testCase, false)
		fmt.Println(err)
		rc <- err
		return
	}

	str := string(blob[:len(content)])
	if str != content {
		logger.TestResult(testCase, false)
		rc <- err
		return fmt.Errorf("Reading from chunk %d. Expected: '%s'; got: '%s'", CHUNKNUM, content, str)
	} else {
		logger.TestResult(testCase, true)
	}

	return
}
