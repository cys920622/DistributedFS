package shared

import "time"

const UnsetClientId = -1

type FileMode int

const (
	// Read mode.
	READ FileMode = iota

	// Read/Write mode.
	WRITE

	// Disconnected read mode.
	DREAD
)

type FileExistsArgs struct {
	Filename string
}

type ClientRegistrationInfo struct {
	ClientId int
	ClientAddress string
	LatestHeartbeat time.Time
}

type ClientHeartbeat struct {
	ClientId int
	Timestamp time.Time
}

type OpenFileArgs struct {
	ClientId int
	Filename string
	Mode FileMode
}



type OpenFileResponse struct {
	FileData []byte
	Error    error
}
