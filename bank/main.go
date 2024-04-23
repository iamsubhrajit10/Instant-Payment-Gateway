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
	lock_manager "bank/lock_manager"
	pb "bank/protos"
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"time"

	//"strconv"

	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
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

type ReverseData struct {
	TransactionID string
	AccountNumber string
	Amount        int
	Type          string
}

func generateProcessId() string {

	rand.Seed(time.Now().UnixNano())
	return fmt.Sprintf("%06d", rand.Intn(1000000))
}

func processDebitRequest(RequestData RequestDataBank) (string, error) {

	config.Logger.Printf("Processing debit request: %v", RequestData)

	// p, err := cms.NewProcess(config.LeaderPort, RequestData.AccountNumber, RequestData.Type)
	// if err != nil {
	// 	config.Logger.Printf("client create error: %v.\n", err.Error())
	// 	return "", err
	// }
	if config.IsLeader == "TRUE" {
		accounts := lock_manager.GetLocksOnAvailableAccounts([]string{RequestData.AccountNumber})
		fmt.Println("I have locks on:%v", accounts)
		//perform the bank thing here

		if len(accounts) != 0 {
			able := lock_manager.ReleaseLocksOnAccounts(accounts)
			if !able {
				return "Error releasing locks", fmt.Errorf("Error releasing locks")
			}
		}
	} else {
		// send request to leader
		url := fmt.Sprintf("http://%v:1323/get-lock", config.LeaderIPV4)

		var req lock_manager.Request
		req = lock_manager.Request{
			RequestType: "request",
			Accounts:    []string{RequestData.AccountNumber},
		}
		jsonData, err := json.Marshal(req)
		if err != nil {
			return "Error marshalling request", fmt.Errorf("Error marshalling request")
		}
		request, error := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
		request.Header.Set("Content-Type", "application/json; charset=UTF-8")

		client := &http.Client{}
		response, error := client.Do(request)

		if error != nil {
			panic(error)
		}
		//parse response into the  array of accounts
		var accounts []string
		err = json.NewDecoder(response.Body).Decode(&accounts)
		if err != nil {
			fmt.Println("Error decoding response:", err)
			return "Error decoding response", fmt.Errorf("Error decoding response")
		}
		fmt.Println("I have locks on:%v", accounts)
		defer response.Body.Close()

	}

	// msg, err_ := p.Run(RequestData.AccountNumber, RequestData.Type, cms.RequestDataBank(RequestData))
	// if err_ != nil {
	// 	config.Logger.Printf("Debit request for transaction id %v failed with error: %v.\n", RequestData.TransactionID, err_.Error())
	// 	return "", err_
	// }

	// if msg != "Debit Success" {
	// 	config.Logger.Printf("Debit request for transaction id %v failed with error: %v.\n", RequestData.TransactionID, msg)
	// 	return msg, nil
	// }

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
	p, err := cms.NewProcess(config.LeaderPort, RequestData.AccountNumber, RequestData.Type)
	if err != nil {
		config.Logger.Printf("client create error: %v.\n", err.Error())
		return "", err
	}

	msg, err_ := p.Run(RequestData.AccountNumber, RequestData.Type, cms.RequestDataBank(RequestData))
	if err_ != nil {
		config.Logger.Printf("Debit reverse for transaction id %v failed with error: %v.\n", RequestData.TransactionID, err_.Error())
		return "", err_
	}

	if msg != "Transaction Reversed" {
		config.Logger.Printf("Debit reverse for transaction id %v failed with error: %v.\n", RequestData.TransactionID, msg)
		return msg, nil
	}

	return "Transaction Reversed", nil
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
			msg, err := processDebitRequest(data)
			if err != nil {
				return "", err
			}
			return msg, nil
		}

	case "reverse":
		{
			msg, err := processDebitRequest(data)
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
		go func() {
			lock_manager.StartServer()
		}()
	}
	// go cms.CreditProcessing(config.LeaderPort)
	port = flag.Int("port", config.BANKSERVERPORT, "The server port")
	flag.Parse()
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		config.Logger.Fatalf("failed to listen: %v", err)
	}

	// Set keepalive server parameters
	ka := keepalive.ServerParameters{
		MaxConnectionAge:      time.Second * 30,
		MaxConnectionAgeGrace: time.Second * 10,
		Time:                  time.Second * 10,
		Timeout:               time.Second * 5,
	}

	// Set keepalive enforcement policy
	kep := keepalive.EnforcementPolicy{
		MinTime:             time.Second * 10, // Minimum amount of time a client should wait before sending a keepalive
		PermitWithoutStream: true,             // Allow pings even when there are no active streams
	}

	// Create server options to set the keepalive policy
	kaOption := grpc.KeepaliveParams(ka)
	kepOption := grpc.KeepaliveEnforcementPolicy(kep)

	// Create the gRPC server with the keepalive options
	s := grpc.NewServer(kaOption, kepOption)

	pb.RegisterDetawilsServer(s, &server{})
	config.Logger.Printf("server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		config.Logger.Fatalf("failed to serve: %v", err)
	}
}
