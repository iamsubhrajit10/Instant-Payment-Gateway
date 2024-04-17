// a centralized mutex server implementation
package message

import (
	"fmt"
	"math/rand"
)

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const msgIDCnt = 10

func RandStringBytes(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

type MessageType int

const (
	Request MessageType = iota + 1 // request mutual lock
	Release                        // release mutual lock
	Grant                          // grant mutual lock
)

type Message struct {
	MsgID      string
	MsgType    MessageType
	MsgContent interface{}
	Sender     int
	Receiver   int
}

// NewRequest returns a new distributed mutual lock message.
func NewRequest(sender int, receiver int, msgContent interface{}) *Message {
	return &Message{
		MsgID:      RandStringBytes(msgIDCnt),
		MsgType:    Request,
		Sender:     sender,
		Receiver:   receiver,
		MsgContent: msgContent,
	}
}

func NewRelease(sender int, receiver int, msgContent interface{}) *Message {
	return &Message{
		MsgID:      RandStringBytes(msgIDCnt),
		MsgType:    Release,
		Sender:     sender,
		Receiver:   receiver,
		MsgContent: msgContent,
	}
}

func NewGrant(sender int, receiver int, msgContent interface{}) *Message {
	return &Message{
		MsgID:      RandStringBytes(msgIDCnt),
		MsgType:    Grant,
		Sender:     sender,
		Receiver:   receiver,
		MsgContent: msgContent,
	}
}

// String returns a string representation of this message. To pretty-print a
// message, you can pass it to a format string like so:
//     msg := NewRequest()
//     fmt.Printf("Request message: %s\n", msg)
func (m *Message) String() string {
	var name string
	switch m.MsgType {
	case Request:
		name = "Request"
	case Release:
		name = "Release"
	case Grant:
		name = "Grant"
	}
	return fmt.Sprintf("[%s %s %d %d %v]", name, m.MsgID, m.Sender, m.Receiver, m.MsgContent)
}
