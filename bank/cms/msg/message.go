// a centralized mutex server implementation
package message

import (
	//"fmt"
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
	AccountNumber string
	Type          string
	MsgType       string
}

// NewRequest returns a new distributed mutual lock message.
func NewRequest(accountNumber, Type string) *Message {
	return &Message{
		AccountNumber: accountNumber,
		Type:          Type,
		MsgType:       "Request",
	}
}

func NewRelease(accountNumber, Type string) *Message {
	return &Message{
		AccountNumber: accountNumber,
		Type:          Type,
		MsgType:       "Release",
	}
}

func NewGrant(ano, Type, msg string) *Message {
	return &Message{
		AccountNumber: ano,
		Type:          Type,
		MsgType:       msg,
	}
}

// String returns a string representation of this message. To pretty-print a
// message, you can pass it to a format string like so:
//
//	msg := NewRequest()
//	fmt.Printf("Request message: %s\n", msg)
// func (m *Message) String() string {
// 	var name string
// 	switch m.MsgType {
// 	case Request:
// 		name = "Request"
// 	case Release:
// 		name = "Release"
// 	case Grant:
// 		name = "Grant"
// 	}
// 	return fmt.Sprintf("[%s %s %d %d %v]", name, m.MsgID, m.Sender, m.Receiver, m.MsgContent)
// }
