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
	"bank/config"
	pb "bank/protos"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

var port *int

// server is used to implement helloworld.GreeterServer.
type server struct {
	pb.UnimplementedDetailsServer
}

type RequestData struct {
	TransactionID string
	AccountNumber string
	Amount        int
	Type          string
}

func processDebitRequest(RequestData RequestData) (string, error) {

	log.Printf("Processing debit request: %v", RequestData)
	return "Debit request processed", nil
}

func processCreditRequest(RequestData RequestData) (string, error) {
	log.Printf("Processing credit request: %v", RequestData)
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

func main() {
	config.LoadEnvData()
	port = flag.Int("port", config.BANKSERVERPORT, "The server port")
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
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
