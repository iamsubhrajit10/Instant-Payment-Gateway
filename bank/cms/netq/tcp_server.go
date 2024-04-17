package netq

import (
	"errors"
	"fmt"
	"net"
	"sync/atomic"
)

const (
	MSG_BUFFERED_SIZE  = 100
	CHAN_SIZE          = 1
	CONN_BUFFERED_SIZE = 1
	SERVER_HOST        = "localhost"
)

type server struct {
	clientNum     int           // number of clients.
	chanConnList  chan net.Conn // channel to notice a coming connection.
	chanStop      chan bool     // channel to indicate stop server.
	chanRespMap   map[int]*compConn
	chanConnClose chan int
	ln            net.Listener
	readChannel   chan *ReadDataComp
	hostport      string
	chanOnClose   chan bool
	readError     chan int
}

type compConn struct {
	connID         int
	conn           net.Conn
	chanDataBuffer chan []byte
}

type ReadDataComp struct {
	data   []byte
	connID int
}

// New creates and returns (but does not start) a new server.
func NewServer(port int) (Server, error) {
	s := &server{
		clientNum:     0,
		chanStop:      make(chan bool, CHAN_SIZE),
		readError:     make(chan int, CHAN_SIZE),
		chanConnList:  make(chan net.Conn, CONN_BUFFERED_SIZE),
		chanRespMap:   make(map[int]*compConn),
		chanConnClose: make(chan int, CONN_BUFFERED_SIZE),
		readChannel:   make(chan *ReadDataComp, MSG_BUFFERED_SIZE),
		hostport:      fmt.Sprintf(":%d", port),
	}
	if err := s.start(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *server) start() error {
	ln, err := net.Listen("tcp", s.hostport)
	if err != nil {
		fmt.Printf("%v\n", err)
		return err
	}
	s.ln = ln
	go s.handleStuff()
	go func() {
		for {
			// fmt.Println("Server waiting for a connection.")
			if s.ln == nil {
				return
			}
			conn, err := s.ln.Accept()
			if err != nil {
				fmt.Println("Server error on accept: ", err)
				return
			}
			s.chanConnList <- conn
		}
	}()
	return nil
}

func (s *server) ReadData() (int, []byte, error) {
	for {
		select {
		case rdc := <-s.readChannel:
			return rdc.connID, rdc.data, nil
		case connID := <-s.readError:
			// fmt.Printf("Server error read.\n")
			return connID, nil, errors.New(fmt.Sprintf("server read from connection(%v) error.\n", connID))
		}
	}
}

func (s *server) WriteData(connID int, data []byte) error {
	cc := s.chanRespMap[connID]
	_, err := cc.conn.Write(Packet(data))
	// fmt.Printf("server write write data to  connection %v: %v\n", connID, data)
	if err != nil {
		// TODO: handle error.
		return err
	}
	return nil
}

func (s *server) Close() error {
	// fmt.Printf("server Close called..\n")
	s.chanStop <- true
	return nil
}

func (s *server) CloseConn(connID int) error {
	s.chanConnClose <- connID
	return nil
}

func (s *server) Count() int {
	return s.clientNum
}

func (s *server) handleConn(cc *compConn) {
	// fmt.Println("Server Reading from connection..")
	tmpBuffer := make([]byte, 0)

	buffer := make([]byte, 1024)
	for {
		n, err := cc.conn.Read(buffer)
		if err != nil {
			// fmt.Printf("Server error read: %v.\n", err.Error())
			s.readError <- cc.connID
			s.chanConnClose <- cc.connID
			return
		}
		// TODO: read error handle
		tmpBuffer = Unpack(append(tmpBuffer, buffer[:n]...), cc.connID, s.readChannel)
	}
}

func (s *server) handleStuff() {
	for {
		select {
		case <-s.chanStop:
			for _, cc := range s.chanRespMap {
				s.chanConnClose <- cc.connID
			}
			s.ln.Close()
			return
		case con := <-s.chanConnList:
			if con != nil {
				s.clientNum++
				connID := s.nextServerConnID()
				cc := &compConn{
					connID:         connID,
					conn:           con,
					chanDataBuffer: make(chan []byte, MSG_BUFFERED_SIZE),
				}
				s.chanRespMap[connID] = cc
				go s.handleConn(cc)
			}
		case connID := <-s.chanConnClose:
			cc, exist := s.chanRespMap[connID]
			if exist && cc.conn != nil {
				s.clientNum--
				delete(s.chanRespMap, connID)
				cc.conn.Close()
			}
		}
	}
}

var nextServerConnID int32 = 0

func (s *server) nextServerConnID() int {
	return int(atomic.AddInt32(&nextServerConnID, 1))
}
