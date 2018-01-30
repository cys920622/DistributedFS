// One reader/writer client and one writer client
// Both clients attempt to open same file for writing

package test

import (
	"io/ioutil"
	"fmt"
	"../dfslib"
	"sync"
)

const FileName121 = "121"

func Test_1_2_1(serverAddr string, wg *sync.WaitGroup) {
	fmt.Println("[1.2.1]")
	fmt.Println("One reader/writer client and one writer client")
	fmt.Println("Both clients attempt to open same file for writing")
	// this creates a directory (to be used as localPath) for each client.
	// The directories will have the format "./client{A,B}NNNNNNNNN", where
	// N is an arbitrary number. Feel free to change these local paths
	// to best fit your environment
	clientALocalPath, errA := ioutil.TempDir(".", "clientA121_")
	clientBLocalPath, errB := ioutil.TempDir(".", "clientB121_")
	if errA != nil || errB != nil {
		panic("Could not create temporary directory")
	}

	errChannel := make(chan error)

	var writerAWait sync.WaitGroup
	var writerBWait sync.WaitGroup

	writerAWait.Add(1)
	writerBWait.Add(1)

	go clientA_1_2_1(serverAddr, LocalIP, clientALocalPath, errChannel, &writerAWait, &writerBWait)
	writerAWait.Wait()

	go clientB_1_2_1(serverAddr, LocalIP, clientBLocalPath, errChannel, &writerBWait)
	writerBWait.Wait()

	e := <- errChannel
	if e != nil {
		wg.Done()
		reportError(e)
	} else {
		fmt.Printf("\nALL TESTS PASSED: Test_1_2_1\n\n")
		CleanDir("clientA121")
		CleanDir("clientB121")
		wg.Done()
	}

}

func clientA_1_2_1(serverAddr, localIP, localPath string, rc chan <- error, wg, next *sync.WaitGroup) (err error) {
	var dfs dfslib.DFS

	logger := NewLogger("(1.2.1) Client A")

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

	testCase = fmt.Sprintf("Opening file '%s' for writing", FileName121)

	file, err := dfs.Open(FileName121, dfslib.WRITE)
	if err != nil {
		logger.TestResult(testCase, false)
		rc <- err
		return
	}
	defer func() {
		if err != nil {
			rc <- err
			wg.Done()
			return
		}

		wg.Done()
		next.Wait()

		testCase := fmt.Sprintf("Closing file '%s'", FileName121)

		err = file.Close()
		if err != nil {
			logger.TestResult(testCase, false)
			rc <- err
			return
		}

		logger.TestResult(testCase, true)
	}()

	logger.TestResult(testCase, true)

	return
}

func clientB_1_2_1(serverAddr, localIP, localPath string, rc chan <- error, wg *sync.WaitGroup) (err error) {
	var dfs dfslib.DFS

	logger := NewLogger("(1.2.1) Client B")

	testCase := fmt.Sprintf("Mounting DFS('%s', '%s', '%s')", serverAddr, localIP, localPath)

	dfs, err = dfslib.MountDFS(serverAddr, localIP, localPath)
	if err != nil {
		logger.TestResult(testCase, false)
		rc <- err
		return
	}
	logger.TestResult(testCase, true)

	defer func() {
		if err = dfs.UMountDFS(); err != nil {
			logger.TestResult("Unmounting DFS", false)
			rc <- err
			return
		}

		logger.TestResult("Unmounting DFS", true)

		wg.Done()
		rc <- nil
	}()

	testCase = fmt.Sprintf("Opening file '%s' for writing fails", FileName121)

	_, err = dfs.Open(FileName121, dfslib.WRITE)
	if err != nil {
		logger.TestResult(testCase, true)
	} else {
		logger.TestResult(testCase, false)
		rc <- err
	}

	return
}
