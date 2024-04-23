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
	"log"
	"net"
	"resolver/config"
	pb "resolver/resolverproto"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"google.golang.org/grpc"
	_ "google.golang.org/grpc/encoding/gzip" // import gzip
	"google.golang.org/grpc/keepalive"
)

var port *int
var db *sql.DB

// server is used to implement helloworld.GreeterServer.
type server struct {
	pb.UnimplementedDetailsServer
}

type Request struct {
	TransactionID string
	PaymentID     string
	Type          string
}

type Reply struct {
	TransactionID string `json:"TransactionID"`
	PaymentID     string `json:"PaymentID"`
	Status        string `json:"Status"`
	AccountNumber string `json:"AccountNumber"`
	IFSCCode      string `json:"IFSCCode"`
	HolderName    string `json:"HolderName"`
}

type RequestData struct {
	Requests []Request
}

type ReplyDataResolver struct {
	Responses []Reply
}

func processResolveRequest(data RequestData) ([]Reply, error) {
	var Responses []Reply
	for _, request := range data.Requests {
		var reply Reply
		reply.TransactionID = request.TransactionID
		reply.PaymentID = request.PaymentID

		// Query the database for the bank details
		row := db.QueryRow("SELECT AccountNumber, IFSCCode, HolderName FROM bank_details WHERE PaymentID = ?", request.PaymentID)
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
				return nil, err
			}
		} else {
			// Results found, set status to "found"
			reply.Status = "found"
		}
		Responses = append(Responses, reply)
	}
	return Responses, nil
}

func processGRPCMessage(msg string) (string, error) {
	var data RequestData
	err := json.Unmarshal([]byte(msg), &data)
	if err != nil {
		return "", err
	}
	if data.Requests[0].Type == "resolve" {
		var Responses []Reply
		Responses, err := processResolveRequest(data)
		if err != nil {
			return "", err
		}
		replyJSON, err := json.Marshal(Responses)
		if err != nil {
			return "", err
		}
		return string(replyJSON), nil
	}
	return "", nil
}

// UnarryCall implements helloworld.GreeterServer
// UnarryCall implements helloworld.GreeterServer
func (s *server) UnarryCall(ctx context.Context, in *pb.Clientmsg) (*pb.Servermsg, error) {
	// Create channels to communicate the result and errors from goroutines
	resultChan := make(chan *pb.Servermsg)
	errorChan := make(chan error)

	// Start a new goroutine to process each gRPC message concurrently
	go func(msg string) {
		log.Printf("Received: %v", in.GetName())
		msg, err := processGRPCMessage(msg)
		if err != nil {
			errorChan <- err
			return
		}
		log.Printf("Sending: %v", msg)
		resultChan <- &pb.Servermsg{Message: msg}
	}(in.GetName())

	// Wait for the result from the goroutines
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
		log.Fatalf("Error opening database: %v", err)
	}
	port = flag.Int("port", config.RESOLVER_SERVER_PORT, "The resovler server port")
	flag.Parse()
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
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

	pb.RegisterDetailsServer(s, &server{})
	log.Printf("server listening at %v", lis.Addr())
	log.Printf("server going to listen at %v", *port)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
