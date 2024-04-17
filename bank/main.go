/*
 *
 * Copyright 2015 gRPC authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

// Package main implements a server for Greeter service.
package main

import (
	cms "bank/cms"
	"bank/config"
	pb "bank/protos"
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net"
	"strconv"
	"time"

	"google.golang.org/grpc"
)

var port *int
var msg string
var err error
var DB *sql.DB

// server is used to implement helloworld.GreeterServer.
type server struct {
	pb.UnimplementedDetailsServer
}

type CentLockMangStruct struct {
	clm *cms.CentLockMang
	// readCnt  int // total read count
	// writeCnt int // total write count
}

type RequestData struct {
	TransactionID string
	AccountNumber string
	Amount        int
	Type          string
}

func generateProcessId() string {

	rand.Seed(time.Now().UnixNano())
	return fmt.Sprintf("%06d", rand.Intn(1000000))
}

func processDebitRequest(RequestData RequestData) (string, error) {

	log.Printf("Processing debit request: %v", RequestData)
	processId, _ := strconv.Atoi(generateProcessId())
	p, err := cms.NewProcess(config.LeaderPort, processId, 123)
	if err != nil {
		log.Printf("client create error: %v.\n", err.Error())
		return "", err
	}

	msg, err_ := p.Run(fmt.Sprintf("message#%v", processId), cms.RequestData(RequestData), DB)
	if err_ != nil {
		log.Printf("Debit request for transaction id %v failed with error: %v.\n", RequestData.TransactionID, err_.Error())
		return "", err_
	}

	if msg != "Debit Success" {
		log.Printf("Debit request for transaction id %v failed with error: %v.\n", RequestData.TransactionID, msg)
		return msg, nil
	}

	return "Debit request processed", nil
}

func processCreditRequest(RequestData RequestData) (string, error) {
	log.Printf("Processing credit request: %v", RequestData)
	processId, _ := strconv.Atoi(generateProcessId())
	p, err := cms.NewProcess(config.LeaderPort, processId, 123)
	if err != nil {
		log.Printf("client create error: %v.\n", err.Error())
		return "", err
	}

	msg, err_ := p.Run(fmt.Sprintf("message#%v", processId), cms.RequestData(RequestData), DB)
	if err_ != nil {
		log.Printf("Debit request for transaction id %v failed with error: %v.\n", RequestData.TransactionID, err_.Error())
		return "", err_
	}

	if msg != "Credit Success" {
		log.Printf("Debit request for transaction id %v failed with error: %v.\n", RequestData.TransactionID, msg)
		return msg, nil
	}

	return "Credit request processed", nil
}

func processReverseRequest(RequestData RequestData) (string, error) {
	log.Printf("Processing reverse request: %v", RequestData)
	return "Reverse request processed", nil
}

func processGRPCMessage(msg string) (string, error) {

	log.Printf("Processing message: %v", msg)
	var data RequestData
	err := json.Unmarshal([]byte(msg), &data)
	if err != nil {
		return "", err
	}
	switch data.Type {
	case "debit":
		{
			msg, err := processDebitRequest(data)
			if err != nil {
				return "", err
			}
			return msg, nil
		}
	case "credit":
		{
			msg, err := processCreditRequest(data)
			if err != nil {
				return "", err
			}
			return msg, nil
		}

	case "reverse":
		{
			msg, err := processReverseRequest(data)
			if err != nil {
				return "", err
			}
			return msg, nil
		}

	}
	return "", nil
}

// SayHello implements helloworld.GreeterServer
func (s *server) UnarryCall(ctx context.Context, in *pb.Clientmsg) (*pb.Servermsg, error) {
	log.Printf("Received: %v", in.GetName())
	msg, err := processGRPCMessage(in.GetName())
	if err != nil {
		return &pb.Servermsg{Message: "Error processing message"}, err
	}
	return &pb.Servermsg{Message: msg}, nil
}
func initializeLearderServer() {
	// initialize the leader server
	clm, err := cms.NewCentLockMang(config.LeaderPort, config.ServerID)
	if err != nil {
		log.Printf("Start centralized server manager(%v) error: %v.\n", config.ServerID, err.Error())
		return
	}
	clms := CentLockMangStruct{clm: clm}
	go clms.clm.Start()
	// if err := clms.clm.Start(); err != nil {
	// 	return
	// }
}

func connectWithSql() (*sql.DB, string, error) {
	db, err := sql.Open("mysql", "root:root@tcp(127.0.0.1:3306)/upi")
	if err != nil {
		log.Fatal(err)
		return nil, "", err
	}
	//defer DB.Close()
	if db != nil {
		log.Printf("DB is not nil")
		err = db.Ping()
		log.Printf("DB is not nil")
		if err != nil {
			log.Fatal(err)
			return nil, "", err
		}

		log.Println("Successfully connected to MySQL database")
		return db, "Success", nil
	}
	return nil, "", nil
}

func main() {

	DB, msg, err = connectWithSql()
	if err != nil {
		log.Fatalf("Error connecting to sql: %v", err)
	}
	log.Printf(msg)
	err = DB.Ping()
	if err != nil {
		log.Fatal(err)
		//return nil, "", err
	}
	results, err := DB.Query("SELECT Amount FROM bank_details WHERE Account_number = ?", "1234")
	if err != nil {
		//log.Fatal(err)
		//return "", err
	}

	for results.Next() {
		var amount int
		// for each row, scan the result into our tag composite object
		err = results.Scan(&amount)
		if err != nil {
			log.Fatal(err)
			// proper error handling instead of panic in your app
			//	return "", err
		}
		log.Printf("Processing debit request3: %v", amount)

	}

	config.LoadEnvData()
	if config.IsLeader == "TRUE" {
		log.Printf("Bank server is the leader")
		initializeLearderServer()
	}
	port = flag.Int("port", config.BANKSERVERPORT, "The server port")
	flag.Parse()
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterDetailsServer(s, &server{})
	log.Printf("server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
