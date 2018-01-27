package dfslib

import "net/rpc"
import (
	"../shared"
)

type File struct {
	filename string
	clientId int
	data []byte
	localPath string
	rpcClient *rpc.Client
	c *DFSConnection
	// Struct fields
}

// Reads chunk number chunkNum into storage pointed to by
// chunk. Returns a non-nil error if the read was unsuccessful.
//
// Can return the following errors:
// - DisconnectedError (in READ,WRITE modes)
// - ChunkUnavailableError (in READ,WRITE modes)
func (f File) Read(chunkNum uint8, chunk *Chunk) (err error) {
	if f.c.currentMode == DREAD {
		// todo - implement
		return nil
	} else {
		if !f.c.isConnected() {return DisconnectedError(f.c.serverAddr.String())}

		// todo - implement

		return nil
	}
}

// Writes chunk number chunkNum from storage pointed to by
// chunk. Returns a non-nil error if the write was unsuccessful.
//
// Can return the following errors:
// - BadFileModeError (in READ,DREAD modes)
// - DisconnectedError (in WRITE mode)
// - WriteModeTimeoutError (in WRITE mode)
func (f File) Write(chunkNum uint8, chunk *Chunk) (err error) {
	if f.c.currentMode != WRITE {return BadFileModeError(f.c.currentMode)}

	if !f.c.isConnected() {return DisconnectedError(f.c.serverAddr.String())}

	request := shared.WriteChunkRequest{
		ClientId:  f.c.clientId,
		Filename:  f.filename,
		ChunkNum:  chunkNum,
		ChunkData: convertChunk(chunk),
	}
	var response shared.WriteChunkResponse
	err = f.c.rpcClient.Call("Server.WriteChunk", request, &response)

	if !response.Success {
		// todo - maybe it should wait until lock is available
		return WriteModeTimeoutError("???")
	} else {
		// Commit write locally
	}
	// todo - implement

	return nil

}

// Closes the file/cleans up. Can return the following errors:
// - DisconnectedError
func (f File) Close() (err error) {
	// todo - set filemode to nil?
	return nil
}

// Returns the absolute path for the file
func (f File) getFilePath() string {
	return f.c.localPath + f.filename + shared.FileExtension
}

// Convert dfslib.Chunk into a type shareable between server and client
func convertChunk(chunk *Chunk) shared.Chunk {
	var c shared.Chunk
	copy(c[:], chunk[:])
	return c

}
