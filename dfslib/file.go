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


// Reads chunk number chunkNum into storage pointed to by
// chunk. Returns a non-nil error if the read was unsuccessful.
//
// Can return the following errors:
// - DisconnectedError (in READ,WRITE modes)
// - ChunkUnavailableError (in READ,WRITE modes)
func (f File) Read(chunkNum uint8, chunk *Chunk) (err error) {
	var resp shared.GetLatestChunkResponse
	req := shared.GetLatestChunkRequest{
		ClientId: f.c.clientId,
		Filename: f.filename,
		ChunkNum: chunkNum,
		Mode: convertMode(f.c.currentMode),
	}

	if f.c.currentMode == DREAD {
		chunkRetrieved := false

		if f.c.isConnected() {
			// Get best-effort version of chunk
			err = f.c.rpcClient.Call("Server.ReadChunk", req, &resp)
			if err == nil && resp.Success {
				copy(chunk[:], resp.ChunkData.Data[:])
				chunkRetrieved = true

				c := []shared.Chunk{resp.ChunkData}
				err = WriteChunksToDisk(c, f.getFilePath())
				if err != nil {return err}
				return nil
			}
		}

		if !chunkRetrieved {
			// Retrieve chunk from disk
			chunkFromDisk, err := ReadChunkFromDisk(f.getFilePath(), chunkNum)
			if err != nil {log.Println(err)}
			copy(chunk[:], chunkFromDisk.Data[:])
			return nil
		}
		return nil
	} else {
		if !f.isOpen || !f.c.isConnected() {
			f.isOpen = false
			return DisconnectedError(f.c.serverAddr.String())
		}

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
	if !f.isOpen || !f.c.isConnected() {
		f.isOpen = false
		return DisconnectedError(f.c.serverAddr.String())
	}

	request := shared.WriteChunkRequest{
		ClientId:  f.c.clientId,
		Filename:  f.filename,
		ChunkNum:  chunkNum,
	}
	var response shared.WriteChunkResponse
	err = f.c.rpcClient.Call("Server.WriteChunk", request, &response)
	if err != nil {
		log.Println("Error with RPC call to server")
		log.Println(err)
		return err
	}

	if !response.Success {
		// Possible transitory disconnection
		return WriteModeTimeoutError(f.filename)
	}
	// Commit write locally
	diskFile, err := os.OpenFile(f.getFilePath(), os.O_WRONLY, 0666)
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
			f.isOpen = false
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
