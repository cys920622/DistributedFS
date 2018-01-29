// Multiple reader clients and one writer client
// Client A writes file, client B,C,D checks file with GlobalFileExists

package test

import (
	"io/ioutil"
	"fmt"
	"../dfslib"
	"time"
	"sync"
)

const FileName131 = "131"


func Test_1_3_1(serverAddr string, itwg *sync.WaitGroup) {
	fmt.Println("[1.3.1]")
	fmt.Println("One reader/writer client and one writer client")
	fmt.Println("Client A creates file, client B checks file with GlobalFileExists")
	// this creates a directory (to be used as localPath) for each client.
	// The directories will have the format "./client{A,B}NNNNNNNNN", where
	// N is an arbitrary number. Feel free to change these local paths
	// to best fit your environment
	clientALocalPath, errA := ioutil.TempDir(".", "clientA131_")
	clientBLocalPath, errB := ioutil.TempDir(".", "clientB131_")
	clientCLocalPath, errC := ioutil.TempDir(".", "clientB131_")
	clientDLocalPath, errD := ioutil.TempDir(".", "clientB131_")

	if errA != nil || errB != nil || errC != nil || errD != nil {
		panic("Could not create temporary directory")
	}

	errChannel := make(chan error)

	var wg sync.WaitGroup
	wg.Add(1)

	go clientA_1_3_1(serverAddr, LocalIP, clientALocalPath, errChannel)
	time.Sleep(500*time.Millisecond)

	go check_1_3_1("B", serverAddr, LocalIP, clientBLocalPath, errChannel, &wg)
	go check_1_3_1("C", serverAddr, LocalIP, clientCLocalPath, errChannel, &wg)
	go check_1_3_1("D", serverAddr, LocalIP, clientDLocalPath, errChannel, &wg)

	e := <- errChannel

	if e != nil {
		reportError(e)
		return
	}

	// Wait for all goroutines to complete
	wg.Wait()
	fmt.Printf("\nALL TESTS PASSED: Test_1_3_1\n\n")
	CleanDir("clientA131")
	CleanDir("clientB131")
	itwg.Done()
	return
}

func clientA_1_3_1(serverAddr, localIP, localPath string, rc chan <- error) (err error) {
	var dfs dfslib.DFS

	logger := NewLogger("(1.3.1) Client A")

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

	testCase = fmt.Sprintf("Opening file '%s' for writing", FileName131)

	file, err := dfs.Open(FileName131, dfslib.WRITE)
	if err != nil {
		logger.TestResult(testCase, false)
		rc <- err
		return
	}
	defer func() {
		if err != nil {
			return
		}

		testCase := fmt.Sprintf("Closing file '%s'", FileName131)

		err = file.Close()
		if err != nil {
			logger.TestResult(testCase, false)
			rc <- err
			return
		}

		logger.TestResult(testCase, true)
	}()

	logger.TestResult(testCase, true)

	// Wait for clients B C D to call GlobalFileExists
	time.Sleep(1500 * time.Millisecond)

	return
}

func check_1_3_1(name string, serverAddr, localIP, localPath string, rc chan <- error, wg *sync.WaitGroup) (err error) {
	defer wg.Done()
	var dfs dfslib.DFS

	logger := NewLogger("(1.3.1) Client " + name)

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


	testCase = fmt.Sprintf("Opening file '%s' for READ", FileName131)
	_, err = dfs.Open(FileName131, dfslib.READ)
	if err == nil {
		logger.TestResult(testCase, true)
	} else {
		logger.TestResult(testCase, false)
		rc <- err
	}

	testCase = fmt.Sprintf("Check file '%s' GlobalFileExists", FileName131)
	exists, err := dfs.GlobalFileExists(FileName131)
	if err == nil && exists {
		logger.TestResult(testCase, true)
	} else {
		logger.TestResult(testCase, false)
		rc <- err
	}

	return
}
