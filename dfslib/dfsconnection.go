package dfslib

import (
	"net"
	"regexp"
	"os"
	"net/rpc"
	"../shared"
	"time"
	"log"
	"strings"
)

type DFSConnection struct {
	// Struct fields
	clientId       int
	serverAddr     *net.TCPAddr
	localAddr      *net.TCPAddr
	localPath      string
	rpcClient      *rpc.Client
	currentMode    FileMode
	shouldSendPing bool
	files 		   map[string]*File
}

func (c DFSConnection) LocalFileExists(fname string) (exists bool, err error) {
	if !isFileNameValid(fname) {return false, BadFilenameError(fname)}

	filePath := getFilePath(c.localPath, fname)

	_, e := os.Stat(filePath)

	return e == nil, nil
}

func (c DFSConnection) GlobalFileExists(fname string) (exists bool, err error) {
	if !isFileNameValid(fname) {return false, BadFilenameError(fname)}
	if !c.isConnected() {
		c.closeFile(fname)
		return false, DisconnectedError(c.serverAddr.String())
	}

	args := shared.FileExistsRequest{Filename: fname}
	var fileExistsReply bool
	err = c.rpcClient.Call("Server.CheckFileExists", args, &fileExistsReply)
	return fileExistsReply, nil
}

func (c DFSConnection) Open(fname string, mode FileMode) (f DFSFile, err error) {
	if !isFileNameValid(fname) {return nil, BadFilenameError(fname)}

	c.currentMode = mode

	if !c.isConnected() {
		if mode == READ || mode == WRITE {
			c.closeFile(fname)
			return nil, DisconnectedError(c.serverAddr.String())
		} else {
			exists, _ := c.LocalFileExists(fname)
			if exists {
				return c.createFileInstance(fname, true)
			} else {
				return nil, FileDoesNotExistError(fname)
			}
		}
	}

	openFileReq := shared.OpenFileRequest{
		ClientId: c.clientId,
		Filename: fname,
		Mode:     convertMode(mode),
	}
	var resp shared.OpenFileResponse
	err = c.rpcClient.Call("Server.OpenFile", openFileReq, &resp)

	if err != nil {
		if mode == READ || mode == WRITE {
			c.closeFile(fname)
			return nil, DisconnectedError(c.serverAddr.String())
		} else {
			return c.createFileInstance(fname, true)
		}
	}

	if resp.UnavailableError {
		log.Printf("Error: File is unavailable: [%s]\n", fname)
		return nil, FileUnavailableError(fname)
	}
	if resp.ConflictError {
		log.Printf("Error: Write conflict: [%s]\n", fname)
		return nil, OpenWriteConflictError(fname)
	}

	c.createLocalEmptyFile(fname)

	WriteChunksToDisk(resp.Chunks, getFilePath(c.localPath, fname))

	return c.createFileInstance(fname, true)
}

func (c DFSConnection) UMountDFS() (err error) {

	c.closeAllFiles()

	if !c.isConnected() {
		log.Println("UMountDFS called but client is disconnected.")
		return nil
	}

	req := shared.ClientRegistrationRequest{
		ClientId: c.clientId,
		ClientAddress: c.localAddr.String(),
		LatestHeartbeat: time.Now().UTC(),
	}
	var resp int
	err = c.rpcClient.Call("Server.DisconnectClient", req, &resp)

	if resp == c.clientId {
		log.Printf("Client [%d] unmounting\n", c.clientId)
		c.shouldSendPing = false
		c.rpcClient.Close()
		return nil
	} else {
		log.Printf("Client [%d] cannot unmount, already disconnected from server\n", c.clientId)
		c.shouldSendPing = false
		c.rpcClient.Close()
		return DisconnectedError(c.serverAddr.String())
	}
}

func (c *DFSConnection) closeAllFiles() {
	log.Println("Closing all files")
	for _, file := range c.files {
		file.isOpen = false
	}
}

func (c *DFSConnection) closeFile(filename string) error {
	_, exists := c.files[filename]
	if !exists {
		log.Printf("Error: close file that does not exist [%s]\n", filename)
		return FileDoesNotExistError(filename)
	} else {
		log.Printf("Closing file [%s]\n", filename)
		c.files[filename].isOpen = false
		return nil
	}
}


// Connect creates an RPC connection to the DFS server.
// Returns an error if there was an issue connecting the server.
// ^ todo - it should not fail on connection issue
func (c *DFSConnection) Connect() error {

	server, err := rpc.Dial("tcp", c.serverAddr.String())
	if err != nil {
		if c.currentMode == DREAD {
			log.Printf("Cannot reach server, continuing in disconnected mode.\n")
			return nil
		}
		log.Println("Error connecting to server")
		return err
	}

	// Establish bi-directional RPC connection
	tcpAddr := c.acceptServerRPC()
	c.localAddr, err = net.ResolveTCPAddr("tcp", tcpAddr)
	if err != nil {
		log.Println("Error ")
		return err
	}


	cidFromDisk, err := c.getClientIdFromDisk()
	if err != nil {
		log.Println("Error retrieving client ID from disk")
		return err
	}

	args := shared.ClientRegistrationRequest{
		ClientId: cidFromDisk,
		ClientAddress: c.localAddr.String(),
		LatestHeartbeat: time.Now().UTC(),
		}

	var cidResponse int

	err = server.Call("Server.RegisterClient", args, &cidResponse)
	if cidResponse == shared.UnsetClientId || err != nil {return err}

	if cidFromDisk == UnsetClientID {
		c.storeClientIdToDisk(cidResponse)
	}

	c.rpcClient = server
	c.clientId = cidResponse


	// Start sending heartbeat to server
	c.shouldSendPing = true
	go c.sendHeartbeat()

	return nil
}


// acceptServerRPC listens for RPC calls from server
func (c *DFSConnection) acceptServerRPC() (ipAddr string) {
	diskService := DiskService{c: *c}

	server := rpc.NewServer()
	server.Register(&diskService)

	log.Printf("LocalAddr: %s\n", c.localAddr.String())

	a, e :=net.ResolveTCPAddr("tcp", c.localAddr.String())

	if e != nil {
		log.Println("Error resolving IP address")
		log.Println(e)
		return
	}

	tcpListener, err := net.ListenTCP("tcp", a)

	ipAddr = tcpListener.Addr().String()

	log.Printf("tcpListner addr: [%s]\n", ipAddr)

	go func() {
		if err == nil {
			log.Printf("Listening for server RPC calls at client address [%s]\n", ipAddr)
			for {
				conn, err := tcpListener.Accept()
				if err != nil {
					log.Printf("Error: failed to start server at [%s]\n", ipAddr)
					return
				}
				log.Printf("Client at [%s] accepted connection from [%s]", ipAddr, conn.RemoteAddr())
				go server.ServeConn(conn)
			}
		} else {
			log.Println("Failed to start server")
			log.Println(err)
		}
	}()

	if err != nil {
		log.Println("Error accepting server RPC")
		log.Println(err)
		return
	} else {
		return
	}
}

func (c *DFSConnection) sendHeartbeat() {
	for {
		if c.shouldSendPing {
			c.PingServer()
		}
		time.Sleep(2 * time.Second)
	}
}

func (c *DFSConnection) isConnected() bool {
	return c.shouldSendPing && c.PingServer() > 0
}

// PingServer sends heartbeats to the server to keep the connection alive
func (c *DFSConnection) PingServer() int {

	args := shared.ClientHeartbeat{ClientId: c.clientId, Timestamp: time.Now().UTC()}
	var pingReply int
	err := c.rpcClient.Call("Server.PingServer", args, &pingReply)
	if err != nil {
		log.Println("Server stopped responding")
		c.shouldSendPing = false
		c.rpcClient.Close()
		return 0
	} else if pingReply != c.clientId {
		log.Printf("Server rejected ping for client %d", c.clientId)
		c.shouldSendPing = false
		c.rpcClient.Close()
		return 0
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

// Create a file on disk filled with zeros, if a file does not exist already.
// Does nothing if the file already exists.
func (c *DFSConnection) createLocalEmptyFile(filename string) {
	filePath := getFilePath(c.localPath, filename)
	_, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		f, err := os.Create(filePath)
		if err != nil {
			log.Printf("Error: cannot create file %s\n", filename)
		}
		emptyArr := make([]byte, shared.BytesPerChunk * shared.ChunksPerFile)
		_, err = f.WriteAt(emptyArr, 0)
		if err != nil {
			log.Printf("Error: cannot write to file %s\n", filename)
		}
		f.Close()
	}
}

func (c DFSConnection) createFileInstance(filename string, isConnected bool) (f *File, err error) {
	if !isFileNameValid(filename) {return nil, BadFilenameError(filename)}

	f = &File{filename, &c, true}
	c.files[filename] = f
	return
}

// Returns the absolute path for the file
func getFilePath(localPath string, filename string) string {
	if strings.HasSuffix(localPath, "/") {
		return localPath + filename + shared.FileExtension
	} else {
		return localPath + "/" + filename + shared.FileExtension
	}
}

func getByteOffsetFromChunkNum(chunkNum uint8) int64 {
	return int64(chunkNum) * int64(shared.BytesPerChunk)
}