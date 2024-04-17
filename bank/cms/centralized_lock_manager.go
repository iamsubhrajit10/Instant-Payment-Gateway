// a centralized mutex server implementation
// a single server that acts as a lock manager. It maintains queue Q containing lock requests that have not yet been granted.
package cms

import (
	msgp "bank/cms/msg"
	netq "bank/cms/netq"
	"container/list"
	"encoding/json"
	"fmt"
	"log"
	// "sync"
)

const (
	MSG_BUFFERED_SIZE = 100
)

type CentLockMang struct {
	managerID    int        // regrad it as manager process id.
	processQueue *list.List // FIFO
	granted      bool
	srv          netq.Server
	port         int
	chanRecvMsg  chan msgCompStruct
	logger       *log.Logger

	// sata info
	readCnt  int
	writeCnt int
}

type msgCompStruct struct {
	msg    msgp.Message
	connID int
}

func NewCentLockMang(port, managerID int) (*CentLockMang, error) {
	clm := &CentLockMang{
		managerID:    managerID,
		port:         port,
		chanRecvMsg:  make(chan msgCompStruct, MSG_BUFFERED_SIZE),
		granted:      false,
		processQueue: list.New(),
	}
	//clm.logger = CreateLog("log/centLockMang.log", "[centLockMang]")
	srv, err := netq.NewServer(clm.port)
	if err != nil {
		log.Printf("centLockMang create error: %v.\n", err.Error())
		return nil, err
	}
	clm.srv = srv
	log.Printf("centLockMang create successfully.\n")
	return clm, nil
}

func (clm *CentLockMang) Start() error {
	go clm.handleLockMsg()
	for {
		connID, readBytes, err := clm.srv.ReadData()
		log.Printf("centLockManlllg receive message(%v) from process(%v).\n", string(readBytes), connID)
		if err != nil {
			clm.logger.Printf("centLockMang receive message error: %v.\n", err.Error())
			// continue
			return err
		}
		clm.readCnt++
		var msg msgp.Message
		json.Unmarshal(readBytes, &msg)
		log.Printf("centLockMang receive message(%v) from process(%v).\n", msg.String(), msg.Sender)
		clm.chanRecvMsg <- msgCompStruct{connID: connID, msg: msg}
	}
}

// you may not need the granted flag.
func (clm *CentLockMang) handleLockMsg() {
	for {
		select {
		case msgComp := <-clm.chanRecvMsg:
			message := msgComp.msg
			log.Printf("centLockMang receive message(%v) from process(%v).\n", message.String(), message.Sender)
			switch message.MsgType {
			case msgp.Request:
				if clm.processQueue.Len() == 0 && !clm.granted {
					if err := clm.sendGrantMsg(message.Receiver, message.Sender, msgComp.connID, message.MsgContent.(string)); err != nil {
						// return // TODO: handle error
						fmt.Printf(err.Error())
						continue
					}
					clm.granted = true
				} else {
					clm.processQueue.PushBack(msgComp) // store the connection anyway
					clm.logger.Printf("centLockMang defer response to process(%v).\n", message.Sender)
				}
			case msgp.Release:
				clm.granted = false
				if clm.processQueue.Len() > 0 {
					mc := clm.processQueue.Remove(clm.processQueue.Front()).(msgCompStruct)
					// clm.managerID
					if err := clm.sendGrantMsg(mc.msg.Receiver, mc.msg.Sender, mc.connID, ""); err != nil {
						// return // TODO: handle error
						continue
					}
					clm.granted = true
				}
			case msgp.Grant:
				clm.logger.Printf("Error message(%v) type Grant.\n", message.String())
			}
		}

	}
}

func (clm *CentLockMang) sendGrantMsg(sender, receiver, connID int, content interface{}) error {
	// write
	lg := msgp.NewGrant(sender, receiver, content.(string)+"[Grant]")
	log.Printf("centLockMang send message(%v) to process(%v).\n", lg.String(), lg.Receiver)
	lgBytes, _ := json.Marshal(lg)
	//log.Printf("centLockMang send message(%v) to process(%v).\n", lg.String(), lg.Receiver)
	if err := clm.srv.WriteData(connID, lgBytes); err != nil {
		clm.logger.Printf("centLockMang send message(%v) to process(%v) error: %v.\n", lg.String(), lg.Receiver, err.Error())
		return err
	}
	log.Printf("centLockMang send message(%v) to process(%v) successfully.\n", lg.String(), lg.Receiver)

	clm.writeCnt++
	clm.logger.Printf("centLockMang send message(%v) to process(%v) successfully.\n", lg.String(), lg.Receiver)
	return nil
}

// @see process.Close
func (clm *CentLockMang) Close() error {
	if err := clm.srv.Close(); err != nil {
		return err
	}
	return nil
}
