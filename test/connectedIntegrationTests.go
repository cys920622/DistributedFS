// This tests a simple, single client DFS scenario. Two clients (A and B) connect
// to the same server in turns:
//
// - Client A mounts DFS at a certain local path (see main function)
// - Client A tries to create a file with an invalid name -- this should return an error
// - Client A creates a new file for writing
// - Client A writes some content on an arbitrary chunk
// - Client A reads back the information just written
// - Client A closes the file and umounts
// - Client B later connects to the same server, but wiht a different local path
// - Client B attempts to check if the file created by Client A exists globally -- this should be true
// - Client B tries to open that file, but it fails -- client A is no longer connected.
package test

import (
	"fmt"
	"io/ioutil"
	"../dfslib"
)

func RunTests(serverAddr string) {
	localIP := LocalIP

	// this creates a directory (to be used as localPath) for each client.
	// The directories will have the format "./client{A,B}NNNNNNNNN", where
	// N is an arbitrary number. Feel free to change these local paths
	// to best fit your environment
	clientALocalPath, errA := ioutil.TempDir(".", "clientA")
	clientBLocalPath, errB := ioutil.TempDir(".", "clientB")
	if errA != nil || errB != nil {
		panic("Could not create temporary directory")
	}

	if err := clientA(serverAddr, localIP, clientALocalPath); err != nil {
		reportError(err)
	}

	if err := clientB(serverAddr, localIP, clientBLocalPath); err != nil {
		reportError(err)
	}

	fmt.Printf("\nALL TESTS PASSED: connectedIntegrationTests\n")

}

func clientA(serverAddr, localIP, localPath string) (err error) {
	var blob dfslib.Chunk
	var dfs dfslib.DFS

	logger := NewLogger("Client A")
	content := "CPSC 416: Hello World!"

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

	testCase = fmt.Sprintf("Attempt to open file '%s' fails", INVALID_FILE_NAME)

	_, err = dfs.Open(INVALID_FILE_NAME, dfslib.WRITE)
	if err == nil {
		logger.TestResult(testCase, false)
		return fmt.Errorf("Opening invalid file name '%s' did not cause an error", INVALID_FILE_NAME)
	}

	logger.TestResult(testCase, true)

	testCase = fmt.Sprintf("Opening file '%s' for writing", VALID_FILE_NAME)

	file, err := dfs.Open(VALID_FILE_NAME, dfslib.WRITE)
	if err != nil {
		logger.TestResult(testCase, false)
		return
	}
	defer func() {
		if err != nil {
			return
		}

		testCase := fmt.Sprintf("Closing file '%s'", VALID_FILE_NAME)

		err = file.Close()
		if err != nil {
			logger.TestResult(testCase, false)
			return
		}

		logger.TestResult(testCase, true)
	}()

	logger.TestResult(testCase, true)

	testCase = fmt.Sprintf("Reading empty chunk %d", CHUNKNUM)

	err = file.Read(CHUNKNUM, &blob)
	if err != nil {
		logger.TestResult(testCase, false)
		return
	}

	for i := 0; i < 32; i++ {
		if blob[i] != 0 {
			logger.TestResult(testCase, false)
			return fmt.Errorf("Byte %d at chunk %d expected to be zero, but is %v", i, CHUNKNUM, blob[i])
		}
	}
	logger.TestResult(testCase, true)

	testCase = fmt.Sprintf("Writing chunk %d", CHUNKNUM)

	copy(blob[:], content)
	err = file.Write(CHUNKNUM, &blob)
	if err != nil {
		logger.TestResult(testCase, false)
		return
	}
	logger.TestResult(testCase, true)

	testCase = fmt.Sprintf("Able to read '%s' back from chunk %d", content, CHUNKNUM)

	err = file.Read(CHUNKNUM, &blob)
	if err != nil {
		logger.TestResult(testCase, false)
		return
	}

	str := string(blob[:len(content)])

	if str != content {
		logger.TestResult(testCase, false)
		return fmt.Errorf("Reading from chunk %d. Expected: '%s'; got: '%s'", CHUNKNUM, content, str)
	}
	logger.TestResult(testCase, true)

	return
}

func clientB(serverAddr, localIP, localPath string) (err error) {
	var dfs dfslib.DFS

	logger := NewLogger("Client B")

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

	testCase = fmt.Sprintf("File '%s' exists globally", VALID_FILE_NAME)

	exists, err := dfs.GlobalFileExists(VALID_FILE_NAME)
	if err != nil {
		logger.TestResult(testCase, false)
		return
	}

	if !exists {
		err = fmt.Errorf("Expected file '%s' to exist globally", VALID_FILE_NAME)
		logger.TestResult(testCase, false)
		return
	}

	logger.TestResult(testCase, true)

	testCase = fmt.Sprintf("Opening file '%s' for reading fails", VALID_FILE_NAME)

	_, err = dfs.Open(VALID_FILE_NAME, dfslib.READ)
	if err == nil {
		logger.TestResult(testCase, false)
		err = fmt.Errorf("Expected opening file '%s' to fail, but it succeded", VALID_FILE_NAME)
		return
	}

	logger.TestResult(testCase, true)
	err = nil // so that the main function won't report the above (expected) error

	return
}
