package netq

import (
	// msgp "bank/cms/msg"
	// "encoding/json"
	// "fmt"
	"log"
	// "math/rand"
	"os"
	// "sync"
	// "testing"
	// "time"
)

var (
	logger *log.Logger
)

func init() {
	serverLogFile, _ := os.OpenFile("log_tcp_netq.log", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	logger = log.New(serverLogFile, "[tcp_netq_test]", log.Lmicroseconds|log.Lshortfile)
}

type msgData struct {
	content string
}

type ClientStruct struct {
	cli         Client
	cliReadCnt  int
	cliWriteCnt int
}

type ServerStruct struct {
	srv         Server
	srvReadCnt  int
	srvWriteCnt int
}

type runSystem struct {
	clis       []ClientStruct
	srv        ServerStruct
	cliCnt     int
	managerID  int
	msgCnt     int
	timeout    int
	clientDone chan bool
}

// type runSystem struct {
// 	clis []ClientStruct
// 	srv  ServerStruct
// 	// stats info
// 	srvReadCnt   int
// 	cliReadCnt   []int
// 	srvWriteCnt  int
// 	cliReadWrite []int
// }

// func (rs *runSystem) runClient(wg *sync.WaitGroup, index int, managerID int) {
// 	defer wg.Done()
// 	c := rs.clis[index].cli
// 	// write
// 	for i := 0; i < rs.msgCnt; i++ {
// 		s := msgp.NewRequest(index, managerID, fmt.Sprintf("This is content of [%d,%d]", index, i))
// 		mdBytes, _ := json.Marshal(s)
// 		if err := c.WriteData(mdBytes); err != nil {
// 			logger.Printf("client write error: %v.\n", err.Error())
// 			rs.clientDone <- false
// 			return
// 		}
// 		rs.clis[index].cliWriteCnt++
// 		// read
// 		readBytes, err := c.ReadData()
// 		if err != nil {
// 			logger.Printf("client read error: %v.\n", err.Error())
// 			rs.clientDone <- false
// 			return
// 		}
// 		var r msgp.Message
// 		json.Unmarshal(readBytes, &r)
// 		// logger.Printf("Client read data from server %v.\n", readBytes)
// 		logger.Printf("client read message from server: %v.\n", r.String())
// 		rs.clis[index].cliReadCnt++
// 	}
// 	// c.Close()
// 	rs.clientDone <- true
// 	return
// }

// func (rs *runSystem) runServer(wg *sync.WaitGroup, managerID int) {
// 	defer wg.Done()
// 	s := rs.srv.srv
// 	// read
// 	for {
// 		connID, readBytes, err := s.ReadData()
// 		if err != nil {
// 			logger.Printf("server read error: %v.\n", err.Error())
// 			return
// 		}
// 		var r msgp.Message
// 		json.Unmarshal(readBytes, &r)
// 		logger.Printf("server read message from connection %v: %v.\n", connID, r.String())
// 		rs.srv.srvReadCnt++
// 		// write #managerID
// 		g := msgp.NewGrant(managerID, r.Sender, r.MsgContent.(string)+"_grant")
// 		mdBytes, _ := json.Marshal(g)
// 		if err := s.WriteData(connID, mdBytes); err != nil {
// 			logger.Printf("client write error: %v.\n", err.Error())
// 			return
// 		}
// 		logger.Printf("server write message to connection %v: %v.\n", connID, g.String())
// 		rs.srv.srvWriteCnt++
// 	}
// }

// func testBasic1(t *testing.T, cliCnt, managerID, msgCnt, timeout int) {
// 	port := 3000 + rand.Intn(50000)
// 	s, err := NewServer(port)
// 	if err != nil {
// 		logger.Printf("server create error: %v.\n", err.Error())
// 		return
// 	}
// 	ss := ServerStruct{srv: s}
// 	var clis []ClientStruct
// 	for i := 0; i < cliCnt; i++ {
// 		c, err := NewClient(port)
// 		if err != nil {
// 			logger.Printf("client create error: %v.\n", err.Error())
// 			return
// 		}
// 		clis = append(clis, ClientStruct{cli: c})
// 	}
// 	rs := &runSystem{
// 		clis:       clis,
// 		srv:        ss,
// 		cliCnt:     cliCnt,
// 		managerID:  managerID,
// 		msgCnt:     msgCnt,
// 		timeout:    timeout,
// 		clientDone: make(chan bool, len(clis)),
// 	}
// 	var wg sync.WaitGroup
// 	wg.Add(cliCnt + 1)
// 	for i := 0; i < cliCnt; i++ {
// 		go rs.runClient(&wg, i, managerID)
// 	}

// 	go rs.runServer(&wg, managerID)

// 	timeoutChan := time.After(time.Duration(rs.timeout) * time.Millisecond)
// 	go func() {
// 		clientDoneCnt := 0
// 		for {
// 			select {
// 			case <-rs.clientDone:
// 				clientDoneCnt++
// 				if clientDoneCnt == len(rs.clis) {
// 					time.Sleep(500 * time.Millisecond)
// 					rs.Close()
// 					return
// 				}
// 			case <-timeoutChan:
// 				rs.Close()
// 				logger.Fatalf("Test time out after %v secs.\n", rs.timeout)
// 				return
// 			}
// 		}
// 	}()

// 	wg.Wait()

// 	// print stats info
// 	rs.printStats()
// }

func (rs *runSystem) Close() {
	for i := 0; i < rs.cliCnt; i++ {
		if rs.clis[i].cli != nil {
			rs.clis[i].cli.Close()
		}
	}
	if rs.srv.srv != nil {
		rs.srv.srv.Close()
	}
	logger.Printf("close all connections successfully.\n")
}

func (rs *runSystem) printStats() {
	totalReadCnt := 0
	totalWriteCnt := 0
	for i := 0; i < rs.cliCnt; i++ {
		logger.Printf("client(%v) readCount=%v, writeCount=%v.\n", i, rs.clis[i].cliReadCnt, rs.clis[i].cliWriteCnt)
		totalReadCnt += rs.clis[i].cliReadCnt
		totalWriteCnt += rs.clis[i].cliWriteCnt
	}
	logger.Printf("server(%v) readCount=%v, writeCount=%v.\n", rs.managerID, rs.srv.srvReadCnt, rs.srv.srvWriteCnt)
	totalReadCnt += rs.srv.srvReadCnt
	totalWriteCnt += rs.srv.srvWriteCnt
	logger.Printf("totalReadCount=%v, totalWriteCount=%v.\n", totalReadCnt, totalWriteCnt)
}
