package dfslib

type File struct {
	// Struct fields
}

// Reads chunk number chunkNum into storage pointed to by
// chunk. Returns a non-nil error if the read was unsuccessful.
//
// Can return the following errors:
// - DisconnectedError (in READ,WRITE modes)
// - ChunkUnavailableError (in READ,WRITE modes)
func (f File) Read(chunkNum uint8, chunk *Chunk) (err error) {
	return nil
}

// Writes chunk number chunkNum from storage pointed to by
// chunk. Returns a non-nil error if the write was unsuccessful.
//
// Can return the following errors:
// - BadFileModeError (in READ,DREAD modes)
// - DisconnectedError (in WRITE mode)
// - WriteModeTimeoutError (in WRITE mode)
func (f File) Write(chunkNum uint8, chunk *Chunk) (err error) {
	return nil
}

// Closes the file/cleans up. Can return the following errors:
// - DisconnectedError
func (f File) Close() (err error) {
	return nil
}
