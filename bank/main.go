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
	"math/rand"
	"net"

	//"strconv"
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
}

type RequestDataBank struct {
	TransactionID string
	AccountNumber string
	IFSCCode      string
	HolderName    string
	Amount        int
	Type          string
}

func generateProcessId() string {

	rand.Seed(time.Now().UnixNano())
	return fmt.Sprintf("%06d", rand.Intn(1000000))
}

func processDebitRequest(RequestData RequestDataBank) (string, error) {

	config.Logger.Printf("Processing debit request: %v", RequestData)

	p, err := cms.NewProcess(config.LeaderPort, RequestData.AccountNumber, RequestData.Type)
	if err != nil {
		config.Logger.Printf("client create error: %v.\n", err.Error())
		return "", err
	}

	msg, err_ := p.Run(RequestData.AccountNumber, RequestData.Type, cms.RequestDataBank(RequestData))
	if err_ != nil {
		config.Logger.Printf("Debit request for transaction id %v failed with error: %v.\n", RequestData.TransactionID, err_.Error())
		return "", err_
	}

	if msg != "Debit Success" {
		config.Logger.Printf("Debit request for transaction id %v failed with error: %v.\n", RequestData.TransactionID, msg)
		return msg, nil
	}

	return "Debit request processed", nil
}

func processCreditRequest(RequestData RequestDataBank) (string, error) {
	config.Logger.Printf("Processing credit request: %v", RequestData)
	p, err := cms.NewProcess(config.LeaderPort, RequestData.AccountNumber, RequestData.Type)
	if err != nil {
		config.Logger.Printf("client create error: %v.\n", err.Error())
		return "", err
	}

	msg, err_ := p.Run(RequestData.AccountNumber, RequestData.Type, cms.RequestDataBank(RequestData))
	if err_ != nil {
		config.Logger.Printf("Credit request for transaction id %v failed with error: %v.\n", RequestData.TransactionID, err_.Error())
		return "", err_
	}

	if msg != "Credit Success" {
		config.Logger.Printf("Credit request for transaction id %v failed with error: %v.\n", RequestData.TransactionID, msg)
		return msg, nil
	}

	return "Credit request processed", nil
}

func processReverseRequest(RequestData RequestDataBank) (string, error) {
	config.Logger.Printf("Processing reverse request: %v", RequestData)
	return "Reverse request processed", nil
}

func processGRPCMessage(msg string) (string, error) {

	config.Logger.Printf("Processing message: %v", msg)
	var data RequestDataBank
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
	config.Logger.Printf("Received: %v", in.GetName())
	msg, err := processGRPCMessage(in.GetName())
	if err != nil {
		return &pb.Servermsg{Message: "Error processing message"}, err
	}
	return &pb.Servermsg{Message: msg}, nil
}
func initializeLearderServer() {
	// initialize the leader server
	clm, err := cms.NewCentLockMang(config.LeaderPort)
	if err != nil {
		config.Logger.Printf("Start centralized server manager(%v) error: %v.\n", config.ServerID, err.Error())
		return
	}
	clms := CentLockMangStruct{clm: clm}
	go clms.clm.Start()
}

func main() {

	config.LoadEnvData()
	if config.IsLeader == "TRUE" {
		config.Logger.Printf("Bank server is the leader")
		initializeLearderServer()
	}
	port = flag.Int("port", config.BANKSERVERPORT, "The server port")
	flag.Parse()
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		config.Logger.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterDetailsServer(s, &server{})
	config.Logger.Printf("server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		config.Logger.Fatalf("failed to serve: %v", err)
	}
}
