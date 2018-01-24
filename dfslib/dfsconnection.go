package dfslib

import "net"

type DFSConnection struct {
	// Struct fields
	serverAddr *net.TCPAddr
	localAddr *net.TCPAddr
	localPath string
}

func (c DFSConnection) LocalFileExists(fname string) (exists bool, err error) {
	return false, nil
}

func (c DFSConnection) GlobalFileExists(fname string) (exists bool, err error) {
	return false, nil
}

func (c DFSConnection) Open(fname string, mode FileMode) (f DFSFile, err error) {
	// Establish TCP connection to server
	return nil, nil
}

func (c DFSConnection) UMountDFS() (err error) {
	return nil
}