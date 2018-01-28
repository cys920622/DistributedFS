package dfslib

import (
	"net"
	"regexp"
	"os"
	"net/rpc"
	"../shared"
	"time"
	"log"
	"strconv"
)

// todo - remove

type DFSConnection struct {
	// Struct fields
	clientId int
	serverAddr *net.TCPAddr
	localAddr *net.TCPAddr
	localPath string
	rpcClient *rpc.Client
	currentMode FileMode
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

	args := shared.FileExistsRequest{Filename: fname}
	var fileExistsReply bool
	err = c.rpcClient.Call("Server.CheckFileExists", args, &fileExistsReply)
	log.Println(fileExistsReply)
	return fileExistsReply, nil
}

func (c DFSConnection) Open(fname string, mode FileMode) (f DFSFile, err error) {
	if !isFileNameValid(fname) {return nil, BadFilenameError(fname)}

	c.currentMode = mode

	if mode == READ || mode == WRITE {
		if !c.isConnected() {return nil, DisconnectedError(c.serverAddr.String())}

		openArgs := shared.OpenFileRequest{
			ClientId: c.clientId,
			Filename: fname,
			Mode:     convertMode(mode),
		}
		var openFileResponse shared.OpenFileResponse
		err = c.rpcClient.Call("Server.OpenFile", openArgs, &openFileResponse)


		if !openFileResponse.Success {
			err = OpenWriteConflictError(fname)
			return nil, err
			// todo - this might not be the only error possible
		}

		if openFileResponse.FileData == nil {
			log.Printf("Creating file on disk: %s\n", fname)
			c.createLocalEmptyFile(fname)
		} else {
			// File has been retrieved from server
			// todo - download file from server
		}

		f := File{
			fname,
			c.clientId,
			openFileResponse.FileData, // todo - is this necessary?
			c.localPath,
			c.rpcClient,
			&c,
			}

		return f, err

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
	if err != nil {
		log.Println("Error connecting to server")
		return err
	}

	freePortNum := getFreeLocalPort(c.localAddr.IP.String())
	rpcReceivingAddr := c.localAddr.IP.String() + ":" + strconv.Itoa(freePortNum)

	// Establish bi-directional RPC connection
	go c.acceptServerRPC(rpcReceivingAddr)

	args := shared.ClientRegistrationRequest{
		ClientId: c.clientId,
		ClientAddress: rpcReceivingAddr,
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


// acceptServerRPC listens for RPC calls from server
func (c *DFSConnection) acceptServerRPC(addr string) error {
	a, e :=net.ResolveTCPAddr("tcp", addr)
	if e != nil {return e}
	tcpListener, err := net.ListenTCP("tcp", a)
	if err == nil {
		log.Printf("Listening for server RPC calls at client address [%s]\n", addr)
		rpc.Accept(tcpListener)
	} else {
		log.Println("Failed to start server")
		log.Println(err)
	}

	return err
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

// PingServer sends heartbeats to the server to keep the connection alive
func (c *DFSConnection) PingServer() int {
	args := shared.ClientHeartbeat{ClientId: c.clientId, Timestamp: time.Now().UTC()}
	var pingReply int
	err := c.rpcClient.Call("Server.PingServer", args, &pingReply)
	if err != nil {
		// todo - remove, or keep state
		log.Println("Server stopped responding")
	} else if pingReply != c.clientId {
		log.Printf("Unexpected response '%d' from server for client %d", pingReply, c.clientId)
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

// Create a file on disk filled with zeros.
func (c *DFSConnection) createLocalEmptyFile(fname string) {
	filePath := c.localPath + fname + shared.FileExtension
	_, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		f, err := os.Create(filePath)
		if err != nil {
			log.Printf("Problem creating file %s\n", fname)
		}
		emptyArr := make([]byte, shared.BytesPerChunk * shared.ChunksPerFile)
		_, err = f.WriteAt(emptyArr, 0)
		if err != nil {
			log.Printf("Problem writing to file %s\n", fname)
		}
		f.Close()
	} else {
		log.Printf("Problem creating file %s - file already exists at %s\n", fname, c.localPath)
	}
}