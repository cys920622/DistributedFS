package dfslib

import (
	"../shared"
	"log"
	"os"
	"strconv"
)

type DiskService struct {
	c DFSConnection
}

// FetchChunk gets a file chunk from local disk and sends it to the server.
func (service *DiskService) FetchChunk(req *shared.FetchChunkRequest, reply *shared.FetchChunkResponse) error {
	log.Printf("Server requested file [%s] chunk [%d]\n", req.Filename, req.ChunkNum)

	chunk, err := ReadChunkFromDisk(getFilePath(service.c.localPath, req.Filename), req.ChunkNum)

	if err != nil {return err}

	*reply = shared.FetchChunkResponse{ChunkData: chunk}
	return nil
}

func ReadChunkFromDisk(filePath string, chunkNum uint8) (shared.Chunk, error) {
	diskFile, err := os.Open(filePath)
	if err != nil {
		log.Printf("Error: cannot open file [%s]\n", filePath)
		return shared.Chunk{}, err
	}

	buffer := make([]byte, 32)

	_, err = diskFile.Seek(getByteOffsetFromChunkNum(chunkNum),0)
	if err != nil {
		log.Printf("Error: cannot read file [%s]\n", filePath)
		return shared.Chunk{}, err
	}

	log.Printf("Disk read: file [%s], chunk [%d] (offset = %d bytes)\n",
		filePath, chunkNum, getByteOffsetFromChunkNum(chunkNum))

	_, err = diskFile.Read(buffer)

	if err != nil {
		log.Printf("Error: cannot read file [%s]\n", filePath)
		return shared.Chunk{}, err
	}

	diskFile.Close()
	var d [32]byte
	copy(d[:], buffer[:])
	chunk := shared.Chunk{ChunkNum: chunkNum, Data: d}

	return chunk, nil
}

func WriteChunksToDisk(chunks []shared.Chunk, filePath string) error {
	diskFile, err := os.OpenFile(filePath, os.O_WRONLY, 0666)
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

// Gets the cached client ID from disk, if one exists.
// If the client was never assigned an ID, returns UnsetClientId.
func (c *DFSConnection) getClientIdFromDisk() (int, error) {
	cidFilePath := c.localPath + ClientIdFileName
	_, err := os.Stat(cidFilePath)
	if err != nil {
		// No previous Client ID
		return UnsetClientID, nil
	}

	idFile, err := os.Open(cidFilePath)
	if err != nil {
		log.Printf("Error: cannot open file [%s]\n", cidFilePath)
		return UnsetClientID, err
	}

	buffer := make([]byte, 16)
	n, err := idFile.Read(buffer)
	cidBytes := buffer[:n]

	if err != nil {
		log.Printf("Error: cannot read file [%s]\n", cidFilePath)
		return UnsetClientID, err
	}

	cid, err := strconv.Atoi(string(cidBytes))
	if err != nil {
		log.Printf("Error: cannot parse client ID file [%s]\n", cidFilePath)
		return UnsetClientID, err
	}

	log.Printf("Cliend ID retrieved from disk: [%d]\n", cid)

	idFile.Close()

	return cid, nil
}

func (c *DFSConnection) storeClientIdToDisk(cid int) error {
	cidFilePath := c.localPath + ClientIdFileName

	// Create the client ID file if it does not exist
	_, err := os.Stat(cidFilePath)
	if os.IsNotExist(err) {
		f, err := os.Create(cidFilePath)
		if err != nil {
			log.Printf("Error: cannot create file %s\n", cidFilePath)
		}
		f.Close()
	}

	cidFile, err := os.OpenFile(cidFilePath, os.O_WRONLY, 0666)
	if err != nil {
		log.Printf("Error: cannot open file [%s]\n", cidFilePath)
		return err
	}

	_, err = cidFile.WriteString(strconv.Itoa(cid))
	if err != nil {
		log.Printf("Error: cannot write to file [%s]\n", cidFilePath)
		return err
	}

	cidFile.Sync()
	cidFile.Close()

	return nil

}