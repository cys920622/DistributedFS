package shared

import (
	"time"
)

const UnsetClientId = -1
const FileExtension = ".dfs"
const ChunksPerFile = 256
const BytesPerChunk = 32

type FileMode int

type Chunk struct {
	Data [32]byte
}
const (
	// Read mode.
	READ FileMode = iota

	// Read/Write mode.
	WRITE

	// Disconnected read mode.
	DREAD
)

type FileExistsRequest struct {
	Filename string
}

type ClientRegistrationRequest struct {
	ClientId int
	ClientAddress string
	LatestHeartbeat time.Time
}

type ClientHeartbeat struct {
	ClientId int
	Timestamp time.Time
}

type OpenFileRequest struct {
	ClientId int
	Filename string
	Mode FileMode
}

type OpenFileResponse struct {
	FileData []byte
	Success bool
}

type GetLatestChunkRequest struct {
	ClientId int
	Filename string
	ChunkNum uint8
	Mode FileMode
}

type GetLatestChunkResponse struct {
	ChunkData Chunk
	Error error
}

type WriteChunkRequest struct {
	ClientId int
	Filename string
	ChunkNum uint8
	ChunkData Chunk
}

type WriteChunkResponse struct {
	Success bool
}
