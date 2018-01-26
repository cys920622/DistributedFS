package main

import (
	"fmt"
	"net"
	"net/rpc"
	"os"
	"./shared"
)

type Server struct {
	ConnectedClients, DisconnectedClients map[int]*shared.ClientRegistrationInfo
	NextClientId int
}


func main() {
	clientIncomingAddr := os.Args[1]

	server := &Server{
		make(map[int]*shared.ClientRegistrationInfo),
		make(map[int]*shared.ClientRegistrationInfo),
		0,
	}
	rpc.Register(server)

	addr, err := net.ResolveTCPAddr("tcp", clientIncomingAddr)
	if err != nil {
		fmt.Println("Failed to resolve address: " + clientIncomingAddr)
		fmt.Println(err)
	}

	tcpListener, err := net.ListenTCP("tcp", addr)

	if err == nil {
		fmt.Println("Accepting clients at " + addr.String())
		rpc.Accept(tcpListener)
	} else {
		fmt.Println("Failed to start server")
		fmt.Println(err)
	}
}


func (s *Server) RegisterClient(args *shared.ClientRegistrationInfo, reply *int) error {
	// todo - do this in a goroutine
	fmt.Printf("Registering client...\n")
	fmt.Printf("ID: %d\n", args.ClientId)
	fmt.Printf("Address: %s\n", args.ClientAddress)

	if args.ClientId == -1 {
		// Case: new client
		s.ConnectedClients[s.NextClientId] = args
		s.NextClientId = s.NextClientId + 1
		*reply = s.NextClientId - 1
	} else {
		// Case: reconnecting client
		// remove from DisconnectedClients and add to ConnectedClients
		s.ConnectedClients[args.ClientId] = args
		delete(s.DisconnectedClients, args.ClientId)
		*reply = args.ClientId
	}

	// todo - establish bidirectional RPC connection
	// todo - start listening for heartbeat

	return nil
}
func (s *Server) CheckFileExists(args *shared.Args, reply *string) error {
	fmt.Println("CheckFileExists called")
	*reply = args.Filename + args.B

	return nil
}
