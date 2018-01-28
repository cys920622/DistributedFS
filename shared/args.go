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
	ChunkNum uint8
	Version int
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
	Chunks []Chunk
	Success bool
	ConflictError bool
	UnavailableError bool
}

type GetLatestChunkRequest struct {
	ClientId int
	Filename string
	ChunkNum uint8
	Mode FileMode
}

type GetLatestChunkResponse struct {
	ChunkData Chunk
	Success bool
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

type FetchChunkRequest struct {
	Filename string
	ChunkNum uint8
}

type FetchChunkResponse struct {
	ChunkData Chunk
}