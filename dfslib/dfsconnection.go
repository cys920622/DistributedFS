package dfslib

import (
	"net"
	"regexp"
	"os"
	"net/rpc"
	"fmt"
	"../shared"
	"time"
	"log"
)

// todo - remove

type DFSConnection struct {
	// Struct fields
	clientId int
	serverAddr *net.TCPAddr
	localAddr *net.TCPAddr
	localPath string
	rpcClient *rpc.Client
}

func (c DFSConnection) LocalFileExists(fname string) (exists bool, err error) {
	if !isFileNameValid(fname) {return false, BadFilenameError(fname)}

	filePath := c.localPath + fname + ".dfs"

	_, e := os.Stat(filePath)

	return e == nil, nil
}

func (c DFSConnection) GlobalFileExists(fname string) (exists bool, err error) {
	if !isFileNameValid(fname) {return false, BadFilenameError(fname)}
	if !c.isConnected() {return false, DisconnectedError(c.serverAddr.String())}

	args := shared.FileExistsArgs{Filename: fname}
	var fileExistsReply bool
	err = c.rpcClient.Call("Server.CheckFileExists", args, &fileExistsReply)
	fmt.Println(fileExistsReply)
	return fileExistsReply, nil
}

func (c DFSConnection) Open(fname string, mode FileMode) (f DFSFile, err error) {
	if !isFileNameValid(fname) {return nil, BadFilenameError(fname)}

	if mode == READ || mode == WRITE {
		if !c.isConnected() {return nil, DisconnectedError(c.serverAddr.String())}

		openArgs := shared.OpenFileArgs{
			ClientId: c.clientId,
			Filename: fname,
			Mode:     convertMode(mode),
		}
		var openFileResponse shared.OpenFileResponse
		err = c.rpcClient.Call("Server.OpenFile", openArgs, &openFileResponse)
		if openFileResponse.Error != nil {
			// todo - respond to error
		} else {
			if openFileResponse.FileData == nil {
				// File is new or has never been written to
				log.Println("File is new or has never been written to")
				// todo - create file locally
			} else {
				// File has been retrieved from server
				// todo - download file from server
			}
		}

		f := File{openFileResponse.FileData}

		return f, openFileResponse.Error

	} else {
		// todo - need to do DREAD
		return nil, nil
	}
}

func (c DFSConnection) UMountDFS() (err error) {
	// todo
	return nil
}


// Connect creates an RPC connection to the DFS server.
// Returns an error if there was an issue connecting the server.
func (c *DFSConnection) Connect() error {
	server, err := rpc.Dial("tcp", c.serverAddr.String())
	if err != nil {return err}

	args := shared.ClientRegistrationInfo{
		ClientId: c.clientId,
		ClientAddress: c.localAddr.String(),
		LatestHeartbeat: time.Now().UTC(),
		}

	var cid int
	err = server.Call("Server.RegisterClient", args, &cid)
	if cid == shared.UnsetClientId || err != nil {return err}

	c.rpcClient = server
	c.clientId = cid
	// todo - store this ClientID on disk

	// Start sending heartbeat to server
	go c.sendHeartbeat()

	return nil
}

func (c *DFSConnection) sendHeartbeat() {
	for {
		time.Sleep(2 * time.Second)
		c.PingServer()
	}
}

func (c *DFSConnection) isConnected() bool {
	// todo - reconsider this approach? check connected state instead from periodic ping?
	return c.PingServer() > 0
}

func (c *DFSConnection) PingServer() int {
	args := shared.ClientHeartbeat{ClientId: c.clientId, Timestamp: time.Now().UTC()}
	var pingReply int
	err := c.rpcClient.Call("Server.PingServer", args, &pingReply)
	if err != nil {
		// todo - remove, or keep state
		fmt.Println("Server stopped responding")
	} else if pingReply != c.clientId {
		fmt.Printf("Unexpected response '%d' from server for client %d", pingReply, c.clientId)
	}
	return pingReply
}



// isFileNameValid returns true if these requirements are met:
// - fname is 1-16 chars long
// - fname only contains characters from a-z or 0-9
//
// Note: ".dfs" is not considered to be a part of fname.
func isFileNameValid(fname string) bool {
	if len(fname) > 16 || len(fname) < 1 {return false}

	matched, _ := regexp.MatchString("^[a-z0-9]+$", fname)
	return matched
}

// Convert dfslib.FileMode into a type shareable between server and client
func convertMode(mode FileMode) shared.FileMode {
	if mode == READ {
		return shared.READ
	}
	if mode == WRITE {
		return shared.WRITE
	} else {
		return shared.DREAD
	}
}

func (c DFSConnection) OpenReadMode(fname string) (f DFSFile, err error) {
	// Check for connection
	return nil, nil

}

func (c DFSConnection) OpenWriteMode(fname string) (f DFSFile, err error) {
	// Check for connection
	return nil, nil

}
func (c DFSConnection) OpenDReadMode(fname string) (f DFSFile, err error) {
	return nil, nil

}