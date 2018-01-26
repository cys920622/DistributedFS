package dfslib

import (
	"net"
	"regexp"
	"os"
	"net/rpc"
	"fmt"
	"../shared"
)

// todo - remove

type DFSConnection struct {
	// Struct fields
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
	// todo - if file exists locally, does it exist globally?
	if !isFileNameValid(fname) {return false, BadFilenameError(fname)}

	existsLocal, _ := c.LocalFileExists(fname)

	if existsLocal {
		return true, nil
	} else if c.rpcClient != nil {
		// todo - implement
		//args := shared.Args{Filename: "string1", B: "string2"}
		//var reply string
		//err = c.rpcClient.Call("Server.CheckFileExists", args, &reply)
		//fmt.Println(reply)
		// todo - remove dummy return values
		return false, nil
	} else {
		return false, DisconnectedError(c.serverAddr.String())
	}
}

func (c DFSConnection) Open(fname string, mode FileMode) (f DFSFile, err error) {
	// Establish TCP connection to server
	if mode == READ {
		return c.OpenReadMode(fname)
	} else if mode == WRITE {
		return c.OpenWriteMode(fname)
	} else {
		return c.OpenDReadMode(fname)
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
	if err == nil {
		args := shared.ClientRegistrationInfo{ClientId: -1, ClientAddress: c.localAddr.String()}
		var cid int
		err = server.Call("Server.RegisterClient", args, &cid)
		if cid != -1 && err == nil {
			fmt.Printf("Server accepted connection for ClientID: %d\n", cid)
			// todo - store this ClientID on disk
			// todo - start sending heartbeat
			c.rpcClient = server
			return nil
		} else {
			return err
		}
	}
	return err
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