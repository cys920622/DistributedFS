package main

import (
	"fmt"
	"net"
	"net/rpc"
	"os"
	"./shared"
	"time"
	"log"
)

const ClientTimeoutThreshold = 2.5
const ClientMonitorPeriod = 2

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
		go server.monitorClientConnections()
		rpc.Accept(tcpListener)
	} else {
		fmt.Println("Failed to start server")
		fmt.Println(err)
	}
}

// todo - document
func (s *Server) RegisterClient(args *shared.ClientRegistrationInfo, reply *int) error {

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

	log.Printf("Client %d connected\n", *reply)
	// todo - establish bidirectional RPC connection

	return nil
}

// todo - document
func (s *Server) CheckFileExists(args *shared.Args, reply *string) error {
	fmt.Println("CheckFileExists called")
	*reply = args.Filename + args.B

	return nil
}

// todo - document, rename function?
func (s *Server) PingServer(args *shared.ClientHeartbeat, reply *int) error {
	s.ConnectedClients[args.ClientId].LatestHeartbeat = args.Timestamp
	*reply = args.ClientId
	return nil
}

// Periodically can ConnectedClients to remove clients that have timed out
func (s *Server) monitorClientConnections() {
	for {
		time.Sleep(ClientMonitorPeriod * time.Second)
		timeNow := time.Now().UTC()
		for c, v := range s.ConnectedClients {
			timeDiff := timeNow.Sub(v.LatestHeartbeat).Seconds()
			if timeDiff > ClientTimeoutThreshold {
				s.disconnectClient(c)
			}
		}
	}
}

func (s *Server) disconnectClient(cid int) {
	log.Printf("Client %d disconnected\n", cid)
	s.DisconnectedClients[cid] = s.ConnectedClients[cid]
	delete(s.ConnectedClients, cid)
}