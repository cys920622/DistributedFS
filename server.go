package main

import (
	"fmt"
	"os"
	"net"
	"log"
	"net/rpc"
)

func main() {
	clientIncomingAddr := os.Args[1]
	addr, err := net.ResolveTCPAddr("tcp", clientIncomingAddr)
	if err != nil {
		log.Fatal(err)
	}

	listener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}

	receiver := new(net.Listener)
	rpc.Register(receiver)
	rpc.Accept(listener)
}