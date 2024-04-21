// a centralized mutex server implementation
// a single server that acts as a lock manager. It maintains queue Q containing lock requests that have not yet been granted.
package cms

import (
	msgp "bank/cms/msg"
	netq "bank/cms/netq"
	"bank/config"

	//"container/list"
	"encoding/json"
	"fmt"
	// "sync"
)

const (
	MSG_BUFFERED_SIZE = 100
)

type CentLockMang struct {
	//processQueue *list.List // FIFO
	// processDebitMap  map[string]*list.List
	// processCreditMap map[string]*list.List
	grantMap map[string]bool
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
		port:        port,
		chanRecvMsg: make(chan msgCompStruct, MSG_BUFFERED_SIZE),
		grantMap:    make(map[string]bool),
		//processCreditMap: make(map[string]*list.List),
		//processDebitMap:  make(map[string]*list.List),
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
					config.Logger.Print("Lock Request recieved")
					switch message.Type {
					case "debit", "reverse":
						if !clm.grantMap[message.AccountNumbers[0]] {
							clm.grantMap[message.AccountNumbers[0]] = true
							message.MsgType = "Grant"
							config.Logger.Print("Granting lock to Account %v for type %v", message.AccountNumbers[0], "debit")
							if err := clm.sendMsg(msgComp.connID, message); err != nil {
								// return // TODO: handle error
								fmt.Printf(err.Error())
								continue
							}

						} else {
							message.MsgType = "Reject"
							config.Logger.Print("Reject Lock request for  Account %v for type %v", message.AccountNumbers[0], "debit")
							if err := clm.sendMsg(msgComp.connID, message); err != nil {
								// return // TODO: handle error
								fmt.Printf(err.Error())
								continue
							}
						}
					case "credit":
						result := make([]string, 0)
						for _, data := range message.AccountNumbers {
							if !clm.grantMap[data] {
								clm.grantMap[data] = true
								result = append(result, data)
							}
						}

						message.MsgType = "Grant"
						message.AccountNumbers = result
						config.Logger.Print("grant lock to account %v for credit", message.AccountNumbers)
						if err := clm.sendMsg(msgComp.connID, message); err != nil {
							// return // TODO: handle error
							fmt.Printf(err.Error())
							continue
						}
					}
				}
			case "Release":
				{
					config.Logger.Print("Relese lock")
					for _, data := range message.AccountNumbers {

						clm.grantMap[data] = false

					}
				}
			}
		}

	}
}

func (clm *CentLockMang) sendMsg(connID int, content msgp.Message) error {
	// write
	lg := msgp.NewGrant(content.AccountNumbers, content.Type, "Grant")
	lgBytes, _ := json.Marshal(lg)
	//config.Logger.Printf("centLockMang send message to %v.\n", content.AccountNumbers)
	if err := clm.srv.WriteData(connID, lgBytes); err != nil {
		//	config.Logger.Printf("centLockMang send message to process(%v) error: %v.\n", content.AccountNumber, err.Error())
		return err
	}

	//config.Logger.Printf("centLockMang send message to process(%v) successfully.\n", content.AccountNumber)
	return nil
}

// @see process.Close
func (clm *CentLockMang) Close() error {
	if err := clm.srv.Close(); err != nil {
		return err
	}
	return nil
}
