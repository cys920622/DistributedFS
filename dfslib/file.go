package dfslib

import (
	"../shared"
	"os"
	"log"
	"strings"
)

type File struct {
	filename string
	c *DFSConnection
	isOpen bool
}


// RPC Target.
// Reads chunk number chunkNum into storage pointed to by
// chunk. Returns a non-nil error if the read was unsuccessful.
//
// Can return the following errors:
// - DisconnectedError (in READ,WRITE modes)
// - ChunkUnavailableError (in READ,WRITE modes)
func (f File) Read(chunkNum uint8, chunk *Chunk) (err error) {
	if f.c.currentMode == DREAD {
		// todo - implement
		// todo - chunk is never unavailable in DREAD mode
		return nil
	} else {
		if !f.isOpen || !f.c.isConnected() {return DisconnectedError(f.c.serverAddr.String())}

		req := shared.GetLatestChunkRequest{
			ClientId: f.c.clientId,
			Filename: f.filename,
			ChunkNum: chunkNum,
			Mode: convertMode(f.c.currentMode),
		}

		var resp shared.GetLatestChunkResponse
		err = f.c.rpcClient.Call("Server.ReadChunk", req, &resp)
		if err != nil {return err}

		if !resp.Success {
			log.Printf("Chunk [%d] of file [%s] is unavailable\n", chunkNum, f.filename)
			return ChunkUnavailableError(chunkNum)
		}

		c := []shared.Chunk{resp.ChunkData}

		copy(chunk[:], resp.ChunkData.Data[:])

		if c != nil {
			// Only update chunk locally if non-trivial data returned from server
			err = WriteChunksToDisk(c, f.getFilePath())
			if err != nil {return err}

		} else {
			return nil
		}
		return err
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
	if !f.isOpen || !f.c.isConnected() {return DisconnectedError(f.c.serverAddr.String())}

	request := shared.WriteChunkRequest{
		ClientId:  f.c.clientId,
		Filename:  f.filename,
		ChunkNum:  chunkNum,
	}
	var response shared.WriteChunkResponse
	err = f.c.rpcClient.Call("Server.WriteChunk", request, &response)
	if err != nil {
		// todo - some error here
		return err
	}

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
// - DisconnectedError (in READ/WRITE)
func (f File) Close() (err error) {
	if f.c.currentMode == DREAD {
		f.isOpen = false
		return nil
	}
	if f.isOpen {
		req := shared.CloseFileRequest{
			ClientId: f.c.clientId, Filename: f.filename, Mode: convertMode(f.c.currentMode)}
		var res shared.CloseFileResponse
		err := f.c.rpcClient.Call("Server.CloseFile", req, &res)
		if err != nil || !res.Success {
			log.Printf("Error: failed to close file [%s]\n", f.filename)
			log.Println(err)
			return DisconnectedError(f.c.serverAddr.String())
		} else {
			f.isOpen = false
			return nil
		}
	} else {
		return DisconnectedError(f.c.serverAddr.String())
	}
}

// Returns the absolute path for the file
func (f File) getFilePath() string {
	if strings.HasSuffix(f.c.localPath, "/") {
		return f.c.localPath + f.filename + shared.FileExtension
	} else {
		return f.c.localPath + "/" + f.filename + shared.FileExtension
	}
}

// Convert dfslib.Chunk into a type shareable between server and client
func convertChunkToChunk(chunk *Chunk) shared.Chunk {
	var d [32]byte
	copy(d[:], chunk[:])
	return shared.Chunk{Data: d}

}
