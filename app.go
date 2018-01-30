/*

A trivial application to illustrate how the dfslib library can be used
from an application in assignment 2 for UBC CS 416 2017W2.

Usage:
go run app.go
*/

package main

// Expects dfslib.go to be in the ./dfslib/ dir, relative to
// this app.go file
import "./dfslib"

import "fmt"
import (
	"os"
	"time"
	"./test"
	"sync"
)

const ServerAddr = "127.0.0.1:8081"
const FileName = "helloworld"
const LocalPath1 = "/tmp/dfs-dev/"
const LocalPath2 = "/tmp/dfs-dev1/"
const RunConnectedTests = true

func main() {
	test.CleanDir("client")
	if len(os.Args) != 2 {
		showUsage()
	}
	serverAddr := os.Args[1]

	var wg sync.WaitGroup

	if RunConnectedTests {
		wg.Add(1)
		go test.Test_1_2_1(serverAddr, &wg)
		wg.Wait()

		wg.Add(1)
		go test.Test_1_2_2(serverAddr, &wg)
		wg.Wait()

		wg.Add(1)
		go test.Test_1_2_3(serverAddr, &wg)
		wg.Wait()

		wg.Add(1)
		go test.Test_1_2_4(serverAddr, &wg)
		wg.Wait()

		wg.Add(1)
		go test.Test_1_3_1(serverAddr, &wg)
		wg.Wait()

		wg.Add(1)
		go test.Test_1_3_2(serverAddr, &wg)
		wg.Wait()
	}


	// Below tests (disconnected) must be run manually
	// ----------------------------------------------
	//wg.Add(1)
	//go test.Test_2_1_1(serverAddr, &wg)
	//wg.Wait()
	//wg.Add(1)
	//go test.Test_2_3_1(serverAddr, &wg)
	//wg.Wait()
    // ----------------------------------------------

	wg.Add(1)
	go test.Test_2_2_1(serverAddr, &wg)
	wg.Wait()

	wg.Add(1)
	go test.Test_2_2_2(serverAddr, &wg)
	wg.Wait()

	time.Sleep(2 * time.Second)
	test.CleanDir("client")
	return
}

func showUsage() {
	fmt.Fprintf(os.Stderr, "%s [server-address]\n", os.Args[0])
	os.Exit(1)
}

func runWriterClient() {
	fmt.Println("runWriterClient")
	localIP := "127.0.0.1"

	// Connect to DFS.
	dfs, err := dfslib.MountDFS(ServerAddr, localIP, LocalPath1)
	if checkError(err) != nil {
		return
	}

	// Close the DFS on exit.
	// Defers are really cool, check out: https://blog.golang.org/defer-panic-and-recover
	defer dfs.UMountDFS()

	// Check if hello.txt file exists in the global DFS.
	//exists, err := dfs.GlobalFileExists(FileName)
	//if checkError(err) != nil {
	//	return
	//}
	//
	//if exists {
	//	fmt.Println("File already exists, mission accomplished")
	//	return
	//}

	// Open the file (and create it if it does not exist) for writing.
	f, err := dfs.Open(FileName, dfslib.WRITE)
	if checkError(err) != nil {
		return
	}

	// Close the file on exit.
	defer f.Close()

	// Create a chunk with a string message.
	var chunk dfslib.Chunk
	const str = "Hello friends!"
	copy(chunk[:], str)

	// Write the 0th chunk of the file.
	err = f.Write(0, &chunk)
	if checkError(err) != nil {
		return
	}

	// Read the 0th chunk of the file.
	err = f.Read(0, &chunk)

	checkError(err)
}

func runReaderClient() {
	fmt.Printf("\nrunReaderClient\n")
	localIP := "127.0.0.1"

	// Connect to DFS.
	dfs, err := dfslib.MountDFS(ServerAddr, localIP, LocalPath2)
	if checkError(err) != nil {
		return
	}

	// Close the DFS on exit.
	// Defers are really cool, check out: https://blog.golang.org/defer-panic-and-recover
	defer dfs.UMountDFS()

	// Check if hello.txt file exists in the global DFS.
	//exists, err := dfs.GlobalFileExists(FileName)
	//if checkError(err) != nil {
	//	return
	//}
	//
	//if exists {
	//	fmt.Println("File already exists, mission accomplished")
	//	return
	//}

	// Open the file (and create it if it does not exist) for writing.
	f, err := dfs.Open(FileName, dfslib.READ)
	if checkError(err) != nil {
		return
	}

	// Close the file on exit.
	defer f.Close()

	// Create a chunk with a string message.
	var chunk dfslib.Chunk

	// Read the 0th chunk of the file.
	err = f.Read(0, &chunk)
	checkError(err)
}

func runDReaderClient() {
	fmt.Println("Kill server now")
	time.Sleep(3 * time.Second)
	fmt.Printf("\nrun[D]ReaderClient\n")

	localIP := "127.0.0.1"

	// Connect to DFS.
	dfs, err := dfslib.MountDFS(ServerAddr, localIP, LocalPath1)
	if checkError(err) != nil {
		return
	}

	// Close the DFS on exit.
	// Defers are really cool, check out: https://blog.golang.org/defer-panic-and-recover
	defer dfs.UMountDFS()

	// Check if hello.txt file exists in the global DFS.
	exists, err := dfs.LocalFileExists(FileName)
	if checkError(err) != nil {
		return
	}

	if exists {
		fmt.Println("File already exists, mission accomplished")
		return
	}

	// Open the file (and create it if it does not exist) for writing.
	f, err := dfs.Open(FileName, dfslib.DREAD)
	if checkError(err) != nil {
		return
	}

	// Close the file on exit.
	defer f.Close()

	// Create a chunk with a string message.
	var chunk dfslib.Chunk

	// Read the 0th chunk of the file.
	err = f.Read(0, &chunk)
	checkError(err)
}

// If error is non-nil, print it out and return it.
func checkError(err error) error {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error ", err.Error())
		return err
	}
	return nil
}
