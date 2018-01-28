package dfslib

import "net/rpc"
import (
	"../shared"
	"os"
	"log"
)

// todo - clean up this struct, most stuff is in c already
type File struct {
	filename string
	clientId int
	data []shared.Chunk
	localPath string
	rpcClient *rpc.Client
	c *DFSConnection
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
// NOTE - assumes file exists locally as a result of Open().
func (f File) Write(chunkNum uint8, chunk *Chunk) (err error) {
	if f.c.currentMode != WRITE {return BadFileModeError(f.c.currentMode)}

	if !f.c.isConnected() {return DisconnectedError(f.c.serverAddr.String())}

	request := shared.WriteChunkRequest{
		ClientId:  f.c.clientId,
		Filename:  f.filename,
		ChunkNum:  chunkNum,
	}
	var response shared.WriteChunkResponse
	err = f.c.rpcClient.Call("Server.WriteChunk", request, &response)
	if err != nil {return err}

	if !response.Success {
		// todo - maybe it should wait until lock is available? ATM this is not even possible
		return WriteModeTimeoutError("???")
	}
	// Commit write locally
	diskFile, err := os.OpenFile(f.getFilePath(), os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	if err != nil {
		log.Printf("Error: cannot open file [%s]\n", f.filename)
		return err
	}

	_, err = diskFile.WriteAt(chunk[:], getByteOffsetFromChunkNum(chunkNum))
	if err != nil {
		log.Printf("Error: cannot write to file [%s]\n", f.filename)
		return err
	}

	diskFile.Sync()
	diskFile.Close()

	return err
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
func convertChunkToChunk(chunk *Chunk) shared.Chunk {
	var d [32]byte
	copy(d[:], chunk[:])
	return shared.Chunk{Data: d}

}
