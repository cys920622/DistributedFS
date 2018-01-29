// One reader/writer client and one writer client
// Client A creates file, client B checks file with GlobalFileExists

package test

import (
	"io/ioutil"
	"fmt"
	"../dfslib"
	"time"
	"sync"
)

const FileName122 = "122"

func Test_1_2_2(serverAddr string, wg *sync.WaitGroup) {
	fmt.Println("[1.2.2]")
	fmt.Println("One reader/writer client and one writer client")
	fmt.Println("Client A creates file, client B checks file with GlobalFileExists")
	// this creates a directory (to be used as localPath) for each client.
	// The directories will have the format "./client{A,B}NNNNNNNNN", where
	// N is an arbitrary number. Feel free to change these local paths
	// to best fit your environment
	clientALocalPath, errA := ioutil.TempDir(".", "clientA122_")
	clientBLocalPath, errB := ioutil.TempDir(".", "clientB122_")

	if errA != nil || errB != nil {
		panic("Could not create temporary directory")
	}

	errChannel := make(chan error)

	go clientA_1_2_2(serverAddr, LocalIP, clientALocalPath, errChannel)
	time.Sleep(500*time.Millisecond)
	go clientB_1_2_2(serverAddr, LocalIP, clientBLocalPath, errChannel)

	e := <- errChannel
	if e != nil {
		reportError(e)
	}
	if e == nil {
		fmt.Printf("\nALL TESTS PASSED: Test_1_2_2\n\n")
		CleanDir("clientA122")
		CleanDir("clientB122")
		wg.Done()
	}
}

func clientA_1_2_2(serverAddr, localIP, localPath string, rc chan <- error) (err error) {
	var dfs dfslib.DFS

	logger := NewLogger("(1.2.2) Client A")

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

	testCase = fmt.Sprintf("Opening file '%s' for writing", FileName122)

	file, err := dfs.Open(FileName122, dfslib.WRITE)
	if err != nil {
		logger.TestResult(testCase, false)
		rc <- err
		return
	}
	defer func() {
		if err != nil {
			return
		}

		testCase := fmt.Sprintf("Closing file '%s'", FileName122)

		err = file.Close()
		if err != nil {
			logger.TestResult(testCase, false)
			rc <- err
			return
		}

		logger.TestResult(testCase, true)
	}()

	logger.TestResult(testCase, true)

	time.Sleep(1500 * time.Millisecond)

	return
}

func clientB_1_2_2(serverAddr, localIP, localPath string, rc chan <- error) (err error) {
	var dfs dfslib.DFS

	logger := NewLogger("(1.2.2) Client B")

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

		rc <- nil
	}()


	testCase = fmt.Sprintf("Opening file '%s' for READ", FileName122)
	_, err = dfs.Open(FileName122, dfslib.READ)
	if err == nil {
		logger.TestResult(testCase, true)
	} else {
		logger.TestResult(testCase, false)
		rc <- err
	}

	testCase = fmt.Sprintf("Check file '%s' GlobalFileExists", FileName122)
	exists, err := dfs.GlobalFileExists(FileName122)
	if err == nil && exists {
		logger.TestResult(testCase, true)
	} else {
		logger.TestResult(testCase, false)
		rc <- err
	}

	return
}
