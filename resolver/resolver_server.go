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
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"resolver/config"
	pb "resolver/resolverproto"

	_ "github.com/mattn/go-sqlite3"
	"google.golang.org/grpc"
)

var port *int
var db *sql.DB

// server is used to implement helloworld.GreeterServer.
type server struct {
	pb.UnimplementedDetailsServer
}

type RequestData struct {
	TransactionID string
	PaymentID     string
	Type          string
}

type ReplyData struct {
	TransactionID string `json:"TransactionID"`
	PaymentID     string `json:"PaymentID"`
	Status        string `json:"Status"`
	AccountNumber string `json:"AccountNumber"`
	IFSCCode      string `json:"IFSCCode"`
	HolderName    string `json:"HolderName"`
}

func processResolveRequest(data RequestData) (ReplyData, error) {
	var reply ReplyData
	reply.TransactionID = data.TransactionID
	reply.PaymentID = data.PaymentID

	// Query the database for the bank details
	row := db.QueryRow("SELECT AccountNumber, IFSCCode, HolderName FROM bank_details WHERE PaymentID = ?", data.PaymentID)
	err := row.Scan(&reply.AccountNumber, &reply.IFSCCode, &reply.HolderName)
	if err != nil {
		if err == sql.ErrNoRows {
			// No results, set status to "not found" and fill other fields with identifiable string
			reply.Status = "not found"
			reply.AccountNumber = "N/A"
			reply.IFSCCode = "N/A"
			reply.HolderName = "N/A"
		} else {
			// Some other error occurred
			return ReplyData{}, err
		}
	} else {
		// Results found, set status to "found"
		reply.Status = "found"
	}
	return reply, nil
}

func processGRPCMessage(msg string) (string, error) {
	config.Logger.Printf("Processing message: %v", msg)
	var data RequestData
	err := json.Unmarshal([]byte(msg), &data)
	if err != nil {
		return "", err
	}
	if data.Type == "resolve" {
		var reply ReplyData
		reply, err := processResolveRequest(data)
		if err != nil {
			return "", err
		}
		replyJSON, err := json.Marshal(reply)
		if err != nil {
			return "", err
		}
		return string(replyJSON), nil
	}
	return "", nil
}

// SayHello implements helloworld.GreeterServer
func (s *server) UnarryCall(ctx context.Context, in *pb.Clientmsg) (*pb.Servermsg, error) {
	// Create a channel to communicate the result from the goroutine
	resultChan := make(chan *pb.Servermsg)
	errorChan := make(chan error)

	// Start a new goroutine to process the gRPC message
	go func() {
		config.Logger.Printf("Received: %v", in.GetName())
		msg, err := processGRPCMessage(in.GetName())
		if err != nil {
			errorChan <- err
			return
		}
		config.Logger.Printf("Sending: %v", msg)
		resultChan <- &pb.Servermsg{Message: msg}
	}()

	// Wait for the result from the goroutine
	select {
	case result := <-resultChan:
		return result, nil
	case err := <-errorChan:
		return &pb.Servermsg{Message: "Error processing message"}, err
	}
}

func main() {
	config.LoadEnvData()

	// open the bank_details database
	var err error
	db, err = sql.Open("sqlite3", config.DB_PATH)
	if err != nil {
		config.Logger.Fatalf("Error opening database: %v", err)
	}
	port = flag.Int("port", config.RESOLVER_SERVER_PORT, "The resovler server port")
	flag.Parse()
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		config.Logger.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterDetailsServer(s, &server{})
	config.Logger.Printf("server listening at %v", lis.Addr())
	config.Logger.Printf("server going to listen at %v", *port)
	if err := s.Serve(lis); err != nil {
		config.Logger.Fatalf("failed to serve: %v", err)
	}
}