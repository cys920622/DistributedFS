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
	"flag"
)

const ClientTimeoutThreshold = 2.5
const ClientMonitorPeriod = 2
const FirstClientId = 1
const FirstChunkVer = 0
const LoggingOn = true

// Contains filename.
type AllChunksOfflineError uint8
func (e AllChunksOfflineError) Error() string {
	return fmt.Sprintf("All clients are offline for chunk [%d]\n", e)
}

type ChunkIsTrivialError uint8
func (e ChunkIsTrivialError) Error() string {
	return fmt.Sprintf("Chunk [%d] has never been written to\n", e)
}

type ClientRegistrationInfo struct {
	ClientId int
	ClientAddress string
	LatestHeartbeat time.Time
	RPCConnection *rpc.Client
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
	isLoggingOn := flag.Bool("log", false, "a bool")
	flag.Parse()
	if len(flag.Args()) != 1 {
		fmt.Fprintln(os.Stderr, "./server [-log] [server-address]")
		os.Exit(1)
	}
	clientIncomingAddr := flag.Arg(0)

	if !*isLoggingOn {
		log.SetOutput(ioutil.Discard)
	}

	newServer := rpc.NewServer()
	server := &Server{
		make(map[int]*ClientRegistrationInfo),
		make(map[int]*ClientRegistrationInfo),
		make(map[string]*FileInfo),
		FirstClientId,
	}
	newServer.Register(server)

	addr, err := net.ResolveTCPAddr("tcp", clientIncomingAddr)
	if err != nil {
		log.Println("Failed to resolve address: " + clientIncomingAddr)
		log.Println(err)
	}

	tcpListener, err := net.ListenTCP("tcp", addr)

	if err == nil {
		log.Printf("Accepting clients at [%s]\n", addr)
		for {
			conn, err := tcpListener.Accept()
			if err != nil {
				log.Printf("Error: failed to start server at [%s]\n", addr)
				return
			}
			log.Printf("Client at [%s] accepted connection from [%s]", addr, conn.RemoteAddr())
			go newServer.ServeConn(conn)
		}
	} else {
		log.Println("Failed to start server")
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
	_, isClientConnected := s.ConnectedClients[args.ClientId]
	if isClientConnected {
		s.ConnectedClients[args.ClientId].LatestHeartbeat = args.Timestamp
		*reply = args.ClientId
	} else {
		// A Ping may arrive after a client has already disconnected
		*reply = shared.UnsetClientId
	}
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
				if err == nil {
					chunks = append(chunks, chunk)
				} else {
					log.Println(err)
				}
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

// RPC target
// CloseFile unlocks the file if the mode was WRITE
func (s *Server) CloseFile(req *shared.CloseFileRequest, res *shared.CloseFileResponse) error {
	log.Printf("CloseFile: client [%d], filename [%s]\n", req.ClientId, req.Filename)

	if req.Mode != shared.WRITE {
		*res = shared.CloseFileResponse{Success: true}
		return nil
	}

	lockHolder := s.Files[req.Filename].LockHolder
	if lockHolder == req.ClientId {
		s.Files[req.Filename].LockHolder = shared.UnsetClientId
		log.Printf("Unlocked [%s.dfs]\n", req.Filename)
		*res = shared.CloseFileResponse{Success: true}
	} else {
		log.Printf("Error: cannot unlock file [%s] as client [%d] does not have the lock\n",
			req.Filename, req.ClientId)
		*res = shared.CloseFileResponse{Success: false}
	}
	return nil
}

// ReadChunk: in READ or WRITE mode, fetches the newest version of the chunk or returns an error.
// In DREAD mode, returns the 'best effort' version of the chunk.
func (s *Server) ReadChunk(req *shared.GetLatestChunkRequest, resp *shared.GetLatestChunkResponse) error {
	log.Printf("Read: ClientId: [%d], Filename [%s], Chunk [%d]",
		req.ClientId, req.Filename, req.ChunkNum)

	if req.Mode == shared.DREAD {
		chunk, err := s.getChunkBestEffort(req.Filename, req.ChunkNum)
		if err != nil {
			*resp = shared.GetLatestChunkResponse{Success: false}
		} else {
			*resp = shared.GetLatestChunkResponse{ChunkData: chunk, Success: true}
		}
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

// Returns the latest reachable version of a chunk.
// Returns an error if chunk has never been written, or all owners are offline.
func (s *Server) getChunkBestEffort(filename string, chunkNum uint8) (chunk shared.Chunk, err error) {
	chunkInfo, exists := s.Files[filename].ChunkInfo[chunkNum]

	// If chunk has never been written, return empty data
	if !exists {return shared.Chunk{}, ChunkIsTrivialError(chunkNum)}

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
// Cannot assume the client has the lock because they may have timed out.
func (s *Server) WriteChunk(args *shared.WriteChunkRequest, reply *shared.WriteChunkResponse) error {
	// Add client as newest chunk version owner and increment chunk version
	fileInfo, exists := s.Files[args.Filename]

	// File open failed, or write mode has timed out
	if !exists || fileInfo.LockHolder != args.ClientId {
		*reply = shared.WriteChunkResponse{Success: false}
		return nil
	}

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
	_, clientExists := s.ConnectedClients[clientId]
	if clientExists {
		s.DisconnectedClients[clientId] = s.ConnectedClients[clientId]
		delete(s.ConnectedClients, clientId)
	}
	s.unlockByClientId(clientId)
}

func (s *Server) unlockByClientId(clientId int) {
	for fn, fi := range s.Files {
		if fi.LockHolder == clientId {
			fi.LockHolder = shared.UnsetClientId
			log.Printf("Unlocked [%s.dfs]\n", fn)
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