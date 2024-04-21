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
	AccountNumbers []string
	Type           string
	MsgType        string
}

// NewRequest returns a new distributed mutual lock message.
func NewRequest(accountNumbers []string, Type string) *Message {
	return &Message{
		AccountNumbers: accountNumbers,
		Type:           Type,
		MsgType:        "Request",
	}
}

func NewRelease(accountNumbers []string, Type string) *Message {
	return &Message{
		AccountNumbers: accountNumbers,
		Type:           Type,
		MsgType:        "Release",
	}
}

func NewGrant(ano []string, Type, msg string) *Message {
	return &Message{
		AccountNumbers: ano,
		Type:           Type,
		MsgType:        msg,
	}
}
