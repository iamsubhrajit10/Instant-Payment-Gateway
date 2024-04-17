package netq

// Server defines the interface for a server.
type Server interface {
	ReadData() (int, []byte, error)

	WriteData(connID int, data []byte) error

	CloseConn(connID int) error

	Close() error
}
