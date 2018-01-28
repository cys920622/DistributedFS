package dfslib

import (
	"../shared"
	"log"
	"os"
)

type DiskService struct {
	c DFSConnection
}

func (service *DiskService) FetchChunk(req *shared.FetchChunkRequest, reply *shared.FetchChunkResponse) error {
	log.Printf("Server requested file [%s] chunk [%d]\n", req.Filename, req.ChunkNum)

	diskFile, err := os.Open(getFilePath(service.c.localPath, req.Filename))
	if err != nil {
		log.Printf("Error: cannot open file [%s]\n", req.Filename)
		return err
	}

	buffer := make([]byte, 32)

	_, err = diskFile.Seek(getByteOffsetFromChunkNum(req.ChunkNum),0)
	if err != nil {
		log.Printf("Error: cannot read file [%s]\n", req.Filename)
		return err
	}

	log.Printf("Disk read: file [%s], chunk [%d] (offset = %d bytes)\n",
		req.Filename, req.ChunkNum, getByteOffsetFromChunkNum(req.ChunkNum))
	_, err = diskFile.Read(buffer)

	if err != nil {
		log.Printf("Error: cannot read file [%s]\n", req.Filename)
		return err
	}

	diskFile.Close()
	var d [32]byte
	copy(d[:], buffer[:])
	chunk := shared.Chunk{ChunkNum: req.ChunkNum, Data: d}
	*reply = shared.FetchChunkResponse{ChunkData: chunk}
	return nil
}

func WriteChunksToDisk(chunks []shared.Chunk, filePath string) error {
	// todo - implement
	diskFile, err := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	if err != nil {
		log.Printf("Error: cannot open file [%s]\n", filePath)
		return err
	}

	for _, chunk := range chunks {
		_, err = diskFile.WriteAt(chunk.Data[:], getByteOffsetFromChunkNum(chunk.ChunkNum))
		if err != nil {
			log.Printf("Error: cannot write to file [%s]\n", filePath)
			return err
		}
	}

	diskFile.Sync()
	diskFile.Close()

	return nil
}