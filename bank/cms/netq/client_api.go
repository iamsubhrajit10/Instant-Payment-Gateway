package netq

// Client defines the interface for a client.
type Client interface {
	ConnID() int

	ReadData() ([]byte, error)

	WriteData(data []byte) error

	Close() error
}
