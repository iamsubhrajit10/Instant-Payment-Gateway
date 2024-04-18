// a centralized mutex server implementation
// a single server that acts as a lock manager. It maintains queue Q containing lock requests that have not yet been granted.
package cms

import (
	msgp "bank/cms/msg"
	netq "bank/cms/netq"
	"bank/config"
	"container/list"
	"encoding/json"
	"fmt"
	// "sync"
)

const (
	MSG_BUFFERED_SIZE = 100
)

type CentLockMang struct {
	//processQueue *list.List // FIFO
	processDebitMap  map[string]*list.List
	processCreditMap map[string]*list.List
	grantMap         map[string]bool
	//granted      bool
	srv         netq.Server
	port        int
	chanRecvMsg chan msgCompStruct
}

type msgCompStruct struct {
	msg    msgp.Message
	connID int
}

func NewCentLockMang(port int) (*CentLockMang, error) {
	clm := &CentLockMang{
		port:             port,
		chanRecvMsg:      make(chan msgCompStruct, MSG_BUFFERED_SIZE),
		grantMap:         make(map[string]bool),
		processCreditMap: make(map[string]*list.List),
		processDebitMap:  make(map[string]*list.List),
	}
	srv, err := netq.NewServer(clm.port)
	if err != nil {
		config.Logger.Printf("centLockMang create error: %v.\n", err.Error())
		return nil, err
	}
	clm.srv = srv
	config.Logger.Printf("centLockMang create successfully.\n")
	return clm, nil
}

func (clm *CentLockMang) Start() error {
	go clm.handleLockMsg()
	for {
		connID, readBytes, err := clm.srv.ReadData()
		config.Logger.Printf("centLockManlllg receive message(%v) from process(%v).\n", string(readBytes), connID)
		if err != nil {
			config.Logger.Printf("centLockMang receive message error: %v.\n", err.Error())
			// continue
			return err
		}
		//clm.readCnt++
		var msg msgp.Message
		json.Unmarshal(readBytes, &msg)
		clm.chanRecvMsg <- msgCompStruct{connID: connID, msg: msg}
	}
}

// you may not need the granted flag.
func (clm *CentLockMang) handleLockMsg() {
	for {
		select {
		case msgComp := <-clm.chanRecvMsg:
			message := msgComp.msg
			switch message.MsgType {
			case "Request":
				{
					config.Logger.Print("Request recieved")
					switch message.Type {
					case "debit":
						if clm.processDebitMap[message.AccountNumber] == nil {
							clm.processDebitMap[message.AccountNumber] = list.New()
						}
						// if no process is holding the lock, grant the lock to the process
						// else, add the process to the queue
						if clm.processDebitMap[message.AccountNumber].Len() == 0 && !clm.grantMap[message.AccountNumber] {
							clm.grantMap[message.AccountNumber] = true
							if err := clm.sendGrantMsg(msgComp.connID, message); err != nil {
								// return // TODO: handle error
								fmt.Printf(err.Error())
								continue
							}

						} else {
							clm.processDebitMap[message.AccountNumber].PushBack(msgComp) // store the connection anyway
							//	clm.logger.Printf("centLockMang defer response to process(%v).\n", message.Sender)
						}
					case "credit":
						if clm.processCreditMap[message.AccountNumber] == nil {
							clm.processCreditMap[message.AccountNumber] = list.New()
						}
						// if no process is holding the lock, grant the lock to the process
						// else, add the process to the queue
						if clm.processCreditMap[message.AccountNumber].Len() == 0 && !clm.grantMap[message.AccountNumber] {
							clm.grantMap[message.AccountNumber] = true
							if err := clm.sendGrantMsg(msgComp.connID, message); err != nil {
								// return // TODO: handle error
								fmt.Printf(err.Error())
								continue
							}

						} else {
							clm.processCreditMap[message.AccountNumber].PushBack(msgComp) // store the connection anyway
							//	clm.logger.Printf("centLockMang defer response to process(%v).\n", message.Sender)
						}
					}
				}
			case "Release":
				{
					clm.grantMap[message.AccountNumber] = false

					if (clm.processDebitMap[message.AccountNumber] != nil) && (clm.processDebitMap[message.AccountNumber].Len() > 0) {
						mc := clm.processDebitMap[message.AccountNumber].Remove(clm.processDebitMap[message.AccountNumber].Front()).(msgCompStruct)
						// clm.managerID
						clm.grantMap[message.AccountNumber] = true
						if err := clm.sendGrantMsg(mc.connID, message); err != nil {
							// return // TODO: handle error
							continue
						}
					}

					if (clm.processCreditMap[message.AccountNumber] != nil) && (clm.processCreditMap[message.AccountNumber].Len() > 0) {
						mc := clm.processCreditMap[message.AccountNumber].Remove(clm.processCreditMap[message.AccountNumber].Front()).(msgCompStruct)
						// clm.managerID
						clm.grantMap[message.AccountNumber] = true
						if err := clm.sendGrantMsg(mc.connID, message); err != nil {
							// return // TODO: handle error
							continue
						}
					}
				}
				// case msgp.Grant:
				// 	clm.logger.Printf("Error message(%v) type Grant.\n", message.String())
			}
		}

	}
}

func (clm *CentLockMang) sendGrantMsg(connID int, content msgp.Message) error {
	// write
	lg := msgp.NewGrant(content.AccountNumber, content.Type, "Grant")
	lgBytes, _ := json.Marshal(lg)
	config.Logger.Printf("centLockMang send message to %v.\n", content.AccountNumber)
	if err := clm.srv.WriteData(connID, lgBytes); err != nil {
		config.Logger.Printf("centLockMang send message to process(%v) error: %v.\n", content.AccountNumber, err.Error())
		return err
	}

	config.Logger.Printf("centLockMang send message to process(%v) successfully.\n", content.AccountNumber)
	return nil
}

// @see process.Close
func (clm *CentLockMang) Close() error {
	if err := clm.srv.Close(); err != nil {
		return err
	}
	return nil
}
