package main

import (
	"fmt"
	"net"
	"net/rpc"
	"os"
	"./shared"
	"time"
	"log"
	"io/ioutil"
)

const ClientTimeoutThreshold = 2.5
const ClientMonitorPeriod = 2
const FirstClientId = 1
const FirstChunkVer = 0
const LoggingOn = true

// Contains filename.
type AllChunksOfflineError uint8


type ClientRegistrationInfo struct {
	ClientId int
	ClientAddress string
	LatestHeartbeat time.Time
	RPCConnection *rpc.Client
}

func (e AllChunksOfflineError) Error() string {
	return fmt.Sprintf("All clients are offline for chunk [%d]\n", e)
}

type ChunkInfo struct {
	CurrentVersion int
	// ChunkOwners maps a chunk version to owners by Client ID
	ChunkOwners map[int][]int
}

type FileInfo struct {
	// ChunkInfo represents chunk ownership. Maps chunk # to client ID.
	ChunkInfo map[uint8]*ChunkInfo
	// LockHolder represents the Client ID of the client who is currently
	// holding the write lock for the file.
	LockHolder int
}
type Server struct {
	ConnectedClients, DisconnectedClients map[int]*ClientRegistrationInfo
	Files map[string]*FileInfo
	NextClientId int
}



func main() {
	if !LoggingOn {
		log.SetOutput(ioutil.Discard)
	}

	clientIncomingAddr := os.Args[1]

	server := &Server{
		make(map[int]*ClientRegistrationInfo),
		make(map[int]*ClientRegistrationInfo),
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
func (s *Server) RegisterClient(args *shared.ClientRegistrationRequest, reply *int) error {
	var assignedClientId int

	if args.ClientId == -1 {
		// Case: new client
		s.ConnectedClients[s.NextClientId] = &ClientRegistrationInfo{
			ClientId:        args.ClientId,
			ClientAddress:   args.ClientAddress,
			LatestHeartbeat: args.LatestHeartbeat,
		}
		s.NextClientId = s.NextClientId + 1
		assignedClientId = s.NextClientId - 1
		*reply = s.NextClientId - 1
		log.Printf("Client [%d] connected\n", assignedClientId)
	} else {
		// Case: reconnecting client
		// remove from DisconnectedClients and add to ConnectedClients
		s.ConnectedClients[args.ClientId] = &ClientRegistrationInfo{
			ClientId:        args.ClientId,
			ClientAddress:   args.ClientAddress,
			LatestHeartbeat: args.LatestHeartbeat,
		}
		delete(s.DisconnectedClients, args.ClientId)
		assignedClientId = args.ClientId
		*reply = args.ClientId
		log.Printf("Client [%d] reconnected\n", assignedClientId)
	}

	s.ConnectedClients[assignedClientId].ClientId = assignedClientId

	err := s.establishRPCConnection(assignedClientId)
	if err != nil {return err}

	return nil
}

// DisconnectClient removes the client from online clients. Called by unmounting.
func (s *Server) DisconnectClient(args *shared.ClientRegistrationRequest, reply *int) error {
	s.disconnectClient(args.ClientId)
	*reply = args.ClientId
	return nil
}

func (s *Server) establishRPCConnection(clientId int) error {
	addr := s.ConnectedClients[clientId].ClientAddress
	client, err := rpc.Dial("tcp", addr)
	if err != nil {
		log.Printf("Error establishing RPC connection to [%s]\n", addr)
		return err
	} else {
		log.Printf("Established RPC connection to client [%d] at [%s]\n", clientId, addr)
	}

	s.ConnectedClients[clientId].RPCConnection = client
	return nil
}

// RPC call target. Checks if a file by some name has ever been created.
// Does not care if any or all of that file is offline.
func (s *Server) CheckFileExists(args *shared.FileExistsRequest, reply *bool) error {
	log.Printf("CheckFileExists: [%s]\n", args.Filename)
	*reply = s.doesFileExist(args.Filename)
	return nil
}

// doesFileExist checks if the filename has been seen by the server.
// It does NOT check whether all the chunks are online.
func (s *Server) doesFileExist(filename string) bool {
	_, exists := s.Files[filename]
	return exists
}

// PingServer is called remotely (RPC) by each connected client periodically
// to tell the server that its connection is being maintained.
func (s *Server) PingServer(args *shared.ClientHeartbeat, reply *int) error {
	s.ConnectedClients[args.ClientId].LatestHeartbeat = args.Timestamp
	*reply = args.ClientId
	return nil
}


// OpenFile is an RPC target. If the mode is WRITE, if the file lock is available, it is
// assigned to the calling client. If the file is already locked, the file is not opened.
// Upon opening a file, it returns chunks of the file that are most recent AND online
// (best effort) without guarantee that they are the most recent versions.
func (s *Server) OpenFile(req *shared.OpenFileRequest, reply *shared.OpenFileResponse) error {
	log.Printf("Open: client [%d], file [%s]", req.ClientId, req.Filename)

	if !s.doesFileExist(req.Filename) {
		// Filename has never been seen by server. Create new file.
		s.createNewFile(req)
		*reply = shared.OpenFileResponse{Chunks: nil, Success: true}
		return nil
	} else {
		if req.Mode == shared.WRITE {
			if !s.isFileLockAvailable(req.Filename, req.ClientId) {
				// Write access conflict occurs
				log.Printf("Error: Write conflict for file [%s]\n", req.Filename)
				*reply = shared.OpenFileResponse{
					Chunks: nil, Success: false, ConflictError: true, UnavailableError: false,
				}
				return nil
			} else {
				s.Files[req.Filename].LockHolder = req.ClientId
			}
		}

		fileInfo := s.Files[req.Filename]
		if len(fileInfo.ChunkInfo) == 0 {
			// File exists but it was never written to
			*reply = shared.OpenFileResponse{Success: true}
			return nil
		}

		// Best-effort file fetch from online clients
		var chunks []shared.Chunk
		for chunkNum := 0; chunkNum < shared.ChunksPerFile; chunkNum++ {
			_, exists := fileInfo.ChunkInfo[uint8(chunkNum)]
			if exists {
				chunk, err := s.getChunkBestEffort(req.Filename, uint8(chunkNum))
				if err == nil {chunks = append(chunks, chunk)}
			}
		}

		if len(fileInfo.ChunkInfo) > 0 && len(chunks) == 0 {
			log.Printf("Error: file [%s] is non-trivial but no chunks are reachable\n", req.Filename)
			*reply = shared.OpenFileResponse{
				Chunks: nil, Success: false, ConflictError: false, UnavailableError: true,
			}
			return nil
		}

		*reply = shared.OpenFileResponse{Chunks: chunks, Success: true}

		// For each chunk fetched, the client is now included as an owner
		for _, ci := range chunks {
			chunkInfo := fileInfo.ChunkInfo[ci.ChunkNum]
			chunkInfo.ChunkOwners[ci.Version] =
				append(fileInfo.ChunkInfo[ci.ChunkNum].ChunkOwners[ci.Version], req.ClientId)
		}
		return nil
	}
}

// ReadChunk: in READ or WRITE mode, fetches the newest version of the chunk or returns an error.
// In DREAD mode, returns the 'best effort' version of the chunk.
func (s *Server) ReadChunk(req *shared.GetLatestChunkRequest, resp *shared.GetLatestChunkResponse) error {
	log.Printf("Read: ClientId: [%d], Filename [%s], Chunk [%d]",
		req.ClientId, req.Filename, req.ChunkNum)

	if req.Mode == shared.DREAD {
		log.Println("DREAD Read is currently unsupported")
		*resp = shared.GetLatestChunkResponse{Success: false}
		return nil
	}

	fileInfo := s.Files[req.Filename]

	chunkInfo, exists := fileInfo.ChunkInfo[req.ChunkNum]

	if !exists {
		// File exists but chunk has never been written to
		*resp = shared.GetLatestChunkResponse{Success: true}
		return nil
	}

	currentVersion := chunkInfo.CurrentVersion
	chunk, e := s.getChunkByVersion(req.Filename, req.ChunkNum, currentVersion)

	if e != nil {
		log.Printf("Error: all owners offline for file [%s], chunk [%d]\n", req.Filename, req.ChunkNum)
		*resp = shared.GetLatestChunkResponse{Success: false}
	} else {
		*resp = shared.GetLatestChunkResponse{ChunkData: chunk, Success: true}
		// Add client to owners
		fileInfo.ChunkInfo[chunk.ChunkNum].ChunkOwners[chunk.Version] =
			append(fileInfo.ChunkInfo[chunk.ChunkNum].ChunkOwners[chunk.Version], req.ClientId)
	}
	return nil
}

func (s *Server) getChunkBestEffort(filename string, chunkNum uint8) (chunk shared.Chunk, err error) {
	chunkInfo, exists := s.Files[filename].ChunkInfo[chunkNum]

	// If chunk has never been written, return empty data
	if !exists {return shared.Chunk{}, nil}

	// Find online client with the latest version reachable
	for ver := chunkInfo.CurrentVersion; ver >= FirstChunkVer; ver-- {
		chunk, e := s.getChunkByVersion(filename, chunkNum, ver)
		if e == nil {return chunk, nil}
	}
	log.Printf("Error: all owners offline for file [%s], chunk [%d]\n", filename, chunkNum)
	// All owners are offline for every version
	return shared.Chunk{}, AllChunksOfflineError(chunkNum)
}

func (s *Server) getChunkByVersion(filename string, chunkNum uint8, ver int) (chunk shared.Chunk, err error) {
	chunkInfo := s.Files[filename].ChunkInfo[chunkNum]

	versionOwners := chunkInfo.ChunkOwners[ver]
	for _, owner := range versionOwners {
		if s.isClientConnected(owner) {
			log.Printf("Fetch: owner ClientId: [%d], Filename [%s], Chunk [%d], Ver: [%d]\n",
				owner, filename, chunkNum, ver)
			req := shared.FetchChunkRequest{
				Filename: filename,
				ChunkNum: chunkNum,
			}
			var resp shared.FetchChunkResponse
			err = s.ConnectedClients[owner].RPCConnection.Call("DiskService.FetchChunk", req, &resp)
			if err != nil {
				log.Print(err)
			}

			resp.ChunkData.Version = ver
			return resp.ChunkData, nil
		}
	}

	return shared.Chunk{}, AllChunksOfflineError(chunkNum)
}


// WriteChunk records a Write event in the file's metadata.
// Assumes that the writer has the write lock.
func (s *Server) WriteChunk(args *shared.WriteChunkRequest, reply *shared.WriteChunkResponse) error {
	// Add client as newest chunk version owner and increment chunk version
	fileInfo := s.Files[args.Filename]
	if fileInfo.ChunkInfo[args.ChunkNum] == nil {
		// Chunk has never been written to
		owners := make(map[int][]int)
		owners[0] = []int{args.ClientId}
		fileInfo.ChunkInfo[args.ChunkNum] = &ChunkInfo{FirstChunkVer, owners}
	} else {
		nv := fileInfo.ChunkInfo[args.ChunkNum].CurrentVersion + 1
		fileInfo.ChunkInfo[args.ChunkNum].CurrentVersion = nv
		fileInfo.ChunkInfo[args.ChunkNum].ChunkOwners[nv] = []int{args.ClientId}
	}

	log.Printf("Write: ClientId: [%d], Filename [%s], Chunk [%d], Ver: [%d]\n",
		args.ClientId, args.Filename, args.ChunkNum, fileInfo.ChunkInfo[args.ChunkNum].CurrentVersion)

	*reply = shared.WriteChunkResponse{Success: true}

	return nil
}

// createNewFile adds a new file to the server's file metadata.
// There is no initial information about any chunk.
func (s *Server) createNewFile(args *shared.OpenFileRequest) {
	fileInfo := FileInfo{make(map[uint8]*ChunkInfo), shared.UnsetClientId}
	// Lock file if opened in WRITE mode
	if args.Mode == shared.WRITE {fileInfo.LockHolder = args.ClientId}
	s.Files[args.Filename] = &fileInfo
	log.Printf("Created file: [%s]\n", args.Filename)
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
	log.Printf("Client [%d] disconnected\n", clientId)
	s.DisconnectedClients[clientId] = s.ConnectedClients[clientId]
	delete(s.ConnectedClients, clientId)
	s.unlockByClientId(clientId)
	// todo - close file
	// todo - return error if client was disconnected when called (or not? what about late pings)
}

func (s *Server) unlockByClientId(clientId int) {
	for fn, fi := range s.Files {
		if fi.LockHolder == clientId {
			fi.LockHolder = shared.UnsetClientId
			log.Printf("Unlocked [%s]\n", fn)
			}
	}
}

func (s *Server) isClientConnected(clientId int) bool {
	_, exists := s.ConnectedClients[clientId]
	return exists
}

func (s *Server) isFileLockAvailable(filename string, clientId int) bool {
	lockHolder := s.Files[filename].LockHolder
	return lockHolder == shared.UnsetClientId || lockHolder == clientId
}