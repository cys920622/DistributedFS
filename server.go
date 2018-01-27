package main

import (
	"fmt"
	"net"
	"net/rpc"
	"os"
	"./shared"
	"time"
	"log"
)

const ClientTimeoutThreshold = 2.5
const ClientMonitorPeriod = 2
const FirstClientId = 1

type ChunkInfo struct {
	CurrentVersion int
	// ChunkOwners maps a chunk version to owners by Client ID
	ChunkOwners map[int][]int
}

type FileInfo struct {
	// ChunkInfo represents chunk ownership. Maps chunk # to client ID.
	// todo - need lock
	ChunkInfo map[int]*ChunkInfo
	// LockHolder represents the Client ID of the client who is currently
	// holding the write lock for the file.
	LockHolder int
}
type Server struct {
	ConnectedClients, DisconnectedClients map[int]*shared.ClientRegistrationInfo
	Files map[string]*FileInfo
	NextClientId int
}



func main() {
	clientIncomingAddr := os.Args[1]

	server := &Server{
		make(map[int]*shared.ClientRegistrationInfo),
		make(map[int]*shared.ClientRegistrationInfo),
		make(map[string]*FileInfo),
		FirstClientId,
	}
	rpc.Register(server)

	addr, err := net.ResolveTCPAddr("tcp", clientIncomingAddr)
	if err != nil {
		log.Println("Failed to resolve address: " + clientIncomingAddr)
		log.Println(err)
	}

	tcpListener, err := net.ListenTCP("tcp", addr)

	if err == nil {
		fmt.Println("Accepting clients at " + addr.String())
		go server.monitorClientConnections()
		rpc.Accept(tcpListener)
	} else {
		fmt.Println("Failed to start server")
		log.Println(err)
	}
}

// Adds clients to the connected clients list.
// When a new client connects, assign a unique ClientID.
// Restore client metadata if one reconnects.
func (s *Server) RegisterClient(args *shared.ClientRegistrationInfo, reply *int) error {

	if args.ClientId == -1 {
		// Case: new client
		s.ConnectedClients[s.NextClientId] = args
		s.NextClientId = s.NextClientId + 1
		*reply = s.NextClientId - 1
	} else {
		// Case: reconnecting client
		// remove from DisconnectedClients and add to ConnectedClients
		s.ConnectedClients[args.ClientId] = args
		delete(s.DisconnectedClients, args.ClientId)
		*reply = args.ClientId
	}

	log.Printf("Client %d connected\n", *reply)
	// todo - establish bidirectional RPC connection

	return nil
}

// RPC call target
func (s *Server) CheckFileExists(args *shared.FileExistsRequest, reply *bool) error {
	log.Printf("CheckFileExists: %s\n", args.Filename)
	*reply = s.doesFileExist(args.Filename)
	return nil
}

// doesFileExist checks if the filename has been seen by the server.
// It does NOT check whether all the chunks are online.
func (s *Server) doesFileExist(filename string) bool {
	_, exists := s.Files[filename]
	return exists
}

// todo - document, rename function?
func (s *Server) PingServer(args *shared.ClientHeartbeat, reply *int) error {
	s.ConnectedClients[args.ClientId].LatestHeartbeat = args.Timestamp
	*reply = args.ClientId
	return nil
}


// todo - document
// todo - should return array of bytes
func (s *Server) OpenFile(req *shared.OpenFileRequest, reply *shared.OpenFileResponse) error {
	if !s.doesFileExist(req.Filename) {
		// Filename has never been seen by server. Create new file.
		s.createNewFile(req)
		*reply = shared.OpenFileResponse{FileData: nil, Success: true}
		return nil
	} else {
		if req.Mode == shared.WRITE {
			if !s.isFileLockAvailable(req.Filename, req.ClientId) {
				// Write access conflict occurs
				fmt.Println("Write conflict")
				*reply = shared.OpenFileResponse{FileData: nil, Success: false}
				return nil
			} else {
				s.Files[req.Filename].LockHolder = req.ClientId
			}
		}
		fileInfo := s.Files[req.Filename]
		if len(fileInfo.ChunkInfo) == 0 {
			// File exists but it was never written to
			*reply = shared.OpenFileResponse{FileData: nil, Success: true}
			return nil
		} else {
			// File is registered and has been written to
			// todo - get latest chunks that are reachable
			return nil

		}
	}
}

func (s *Server) GetLatestChunk(args *shared.GetLatestChunkRequest, reply *shared.GetLatestChunkResponse) error {

	// todo - implement

	time.Sleep(1 * time.Minute)
	return nil
}

func (s *Server) WriteChunk(args *shared.WriteChunkRequest, reply *shared.WriteChunkResponse) error {
	// todo - implement
	fmt.Printf("Chunk: %s\n", string(args.ChunkData[:]))

	time.Sleep(1 * time.Minute)
	// todo
	*reply = shared.WriteChunkResponse{Success: true}

	return nil
}

// createNewFile adds a new file to the server's file metadata.
// There is no initial information about any chunk.
func (s *Server) createNewFile(args *shared.OpenFileRequest) {
	fileInfo := FileInfo{make(map[int]*ChunkInfo), shared.UnsetClientId}
	// Lock file if opened in WRITE mode
	if args.Mode == shared.WRITE {fileInfo.LockHolder = args.ClientId}
	s.Files[args.Filename] = &fileInfo
	log.Printf("Created file: %s\n", args.Filename)
}



// Periodically can ConnectedClients to remove clients that have timed out
func (s *Server) monitorClientConnections() {
	for {
		time.Sleep(ClientMonitorPeriod * time.Second)
		timeNow := time.Now().UTC()
		for c, v := range s.ConnectedClients {
			timeDiff := timeNow.Sub(v.LatestHeartbeat).Seconds()
			if timeDiff > ClientTimeoutThreshold {
				s.disconnectClient(c)
			}
		}
	}
}

func (s *Server) disconnectClient(clientId int) {
	log.Printf("Client %d disconnected\n", clientId)
	s.DisconnectedClients[clientId] = s.ConnectedClients[clientId]
	delete(s.ConnectedClients, clientId)
	s.unlockByClientId(clientId)
	// todo - close file
}

func (s *Server) unlockByClientId(clientId int) {
	for _, file := range s.Files {
		if file.LockHolder == clientId {file.LockHolder = shared.UnsetClientId}
	}
}

func (s *Server) isFileLockAvailable(filename string, clientId int) bool {
	lockHolder := s.Files[filename].LockHolder
	return lockHolder == shared.UnsetClientId || lockHolder == clientId
}