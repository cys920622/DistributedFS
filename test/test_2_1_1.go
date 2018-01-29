// One client
// Client writes file(s), disconnects, and can use DREAD and LocalFileExists while disconnected

package test

import (
	"io/ioutil"
	"fmt"
	"../dfslib"
	"sync"
	"time"
)

const FileName211 = "211"

func Test_2_1_1(serverAddr string, itwg *sync.WaitGroup) {
	fmt.Println("[2.1.1]")
	fmt.Println("Disconnected - One client")
	fmt.Println("Client writes file(s), disconnects, and can use DREAD and LocalFileExists while disconnected")
	// this creates a directory (to be used as localPath) for each client.
	// The directories will have the format "./client{A,B}NNNNNNNNN", where
	// N is an arbitrary number. Feel free to change these local paths
	// to best fit your environment
	clientALocalPath, errA := ioutil.TempDir(".", "clientA211_")

	var writer sync.WaitGroup
	var readers sync.WaitGroup
	writer.Add(1)
	readers.Add(3)

	if errA != nil {
		panic("Could not create temporary directory")
	}

	err := clientA_2_1_1(serverAddr, LocalIP, clientALocalPath)

	if err != nil {
		reportError(err)
	}

	fmt.Printf("\nALL TESTS PASSED: Test_2_1_1\n\n")
	CleanDir("clientA211")
	itwg.Done()
	return
}

func clientA_2_1_1(serverAddr, localIP, localPath string) (err error) {
	var blob dfslib.Chunk
	var dfs dfslib.DFS

	logger := NewLogger("(2.1.1) Client A")
	content := "This is test 2.1.1!"

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
			return
		}

		logger.TestResult("Unmounting DFS", true)

	}()

	testCase = fmt.Sprintf("Opening file '%s' for writing", FileName211)

	file, err := dfs.Open(FileName211, dfslib.WRITE)
	if err != nil {
		logger.TestResult(testCase, false)
		return
	}
	logger.TestResult(testCase, true)

	//defer func() {
	//	if err != nil {
	//		return
	//	}
	//
	//	testCase := fmt.Sprintf("Closing file '%s'", FileName211)
	//
	//	err = file.Close()
	//	if err != nil {
	//		logger.TestResult(testCase, false)
	//		return
	//	}
	//
	//	logger.TestResult(testCase, true)
	//}()

	// Write chunk
	testCase = fmt.Sprintf("Writing chunk %d", CHUNKNUM)

	copy(blob[:], content)
	err = file.Write(CHUNKNUM, &blob)
	if err != nil {
		logger.TestResult(testCase, false)
		return
	} else {
		logger.TestResult(testCase, true)
	}

	err = file.Close()
	if err != nil {
		logger.TestResult(testCase, false)
		return
	}

	logger.TestResult(testCase, true)

	for i := ServerShutdownTimer; i > 0; i-- {
		fmt.Printf("SHUT DOWN SERVER NOW: [%d]\n", i)
		time.Sleep(1 * time.Second)
	}

	testCase = fmt.Sprintf("File '%s' exists locally", FileName211)

	exists, err := dfs.LocalFileExists(FileName211)
	if err != nil {
		logger.TestResult(testCase, false)
		return
	}

	if !exists {
		err = fmt.Errorf("Expected file '%s' to exist locally", FileName211)
		logger.TestResult(testCase, false)
		return
	}

	logger.TestResult(testCase, true)


	testCase = fmt.Sprintf("Opening file '%s' in DREAD", FileName211)

	file, err = dfs.Open(FileName211, dfslib.DREAD)
	if err != nil {
		logger.TestResult(testCase, false)
		return
	}
	logger.TestResult(testCase, true)



	var readBuf dfslib.Chunk
	testCase = fmt.Sprintf("Able to read '%s' back from chunk %d while disconnected", content, CHUNKNUM)

	err = file.Read(CHUNKNUM, &readBuf)

	if err != nil {
		logger.TestResult(testCase, false)
		return
	}

	str := string(readBuf[:len(content)])
	if str != content {
		logger.TestResult(testCase, false)
		return fmt.Errorf("Reading from chunk %d. Expected: '%s'; got: '%s'", CHUNKNUM, content, str)
	} else {
		logger.TestResult(testCase, true)
	}

	return
}