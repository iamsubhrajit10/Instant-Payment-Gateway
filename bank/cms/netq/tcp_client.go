package netq

import (
	"fmt"
	"net"
	"sync/atomic"
)

type client struct {
	connID      int
	conn        net.Conn
	hostport    string
	readChannel chan *ReadDataComp
}

func NewClient(port int) (Client, error) {
	c := &client{
		readChannel: make(chan *ReadDataComp, MSG_BUFFERED_SIZE),
		hostport:    fmt.Sprintf(":%d", port),
	}
	if err := c.start(); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *client) start() error {
	addr, err := net.ResolveTCPAddr("tcp", c.hostport)
	if err != nil {
		return err
	}
	conn, err := net.DialTCP("tcp", nil, addr)
	if err != nil {
		return err
	}
	c.conn = conn
	c.connID = c.nextConnID()
	go c.handleConn()
	return nil
}

func (c *client) handleConn() {
	// fmt.Println("Client Reading from connection..")
	tmpBuffer := make([]byte, 0)

	buffer := make([]byte, 1024)
	for {
		n, err := c.conn.Read(buffer)
		if err != nil {
			break
		}
		// TODO: read error handle
		// fmt.Printf("Client read data from server %v.\n", buffer[:n])
		tmpBuffer = Unpack(append(tmpBuffer, buffer[:n]...), c.connID, c.readChannel)
	}
	c.Close()
}

func (c *client) ReadData() ([]byte, error) {
	for {
		select {
		case rdc := <-c.readChannel:
			return rdc.data, nil
		}
	}
}

func (c *client) WriteData(data []byte) error {
	// TODO: handle error
	_, err := c.conn.Write(Packet(data))
	if err != nil {
		return err
	}
	return nil
}

func (c *client) Close() error {
	if c.conn != nil {
		c.conn.Close()
	}
	return nil
}

func (c *client) ConnID() int {
	return c.connID
}

var nextConnID int32 = 0

func (c *client) nextConnID() int {
	return int(atomic.AddInt32(&nextConnID, 1))
}
