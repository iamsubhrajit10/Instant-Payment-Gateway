package netq

// handler for packing and unpacking message.

import (
	"bytes"
	"encoding/binary"
	// "fmt"
)

const (
	packactHeader = "Centralized Mutex Server"
	headerLength  = 24
	dataLength    = 4 // store message data size.
)

// pack message
func Packet(msgByte []byte) []byte {
	return append(append([]byte(packactHeader), intToBytes(len(msgByte))...), msgByte...)
}

// unpack message
func Unpack(buffer []byte, connID int, readChannel chan *ReadDataComp) []byte {
	bufSize := len(buffer)

	var i int
	for i = 0; i < bufSize; i++ {
		// incomplete packet, read continue.
		if bufSize < i+headerLength+dataLength {
			break
		}
		if string(buffer[i:i+headerLength]) == packactHeader {
			msgLen := bytesToInt(buffer[i+headerLength : i+headerLength+dataLength])
			// incomplete packet, read continue.
			if bufSize < i+headerLength+dataLength+msgLen {
				break
			}
			// read a message bytes from buffer.
			data := buffer[i+headerLength+dataLength : i+headerLength+dataLength+msgLen]
			readChannel <- &ReadDataComp{connID: connID, data: data}
			// fmt.Printf("readChannel <- %v", data)
			// continue read.
			i += headerLength + dataLength + msgLen - 1
		}
	}
	if i == bufSize {
		return make([]byte, 0)
	}
	// return unprocessed bytes, incomplete message.
	return buffer[i:]
}

func intToBytes(n int) []byte {
	x := int32(n)

	bytesBuffer := bytes.NewBuffer([]byte{})
	binary.Write(bytesBuffer, binary.BigEndian, x)
	return bytesBuffer.Bytes()
}

func bytesToInt(b []byte) int {
	bytesBuffer := bytes.NewBuffer(b)

	var x int32
	binary.Read(bytesBuffer, binary.BigEndian, &x)

	return int(x)
}
