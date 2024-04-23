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
	"log"
	"math/rand"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	//"strconv"

	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

var port *int
var msg string
var err error
var DB *sql.DB

var creditMap map[string][]creditData = make(map[string][]creditData)
var creditLock map[string]*sync.Mutex = make(map[string]*sync.Mutex)
var globalLock *sync.Mutex = &sync.Mutex{}
var creditAccountNumbers []string

// server is used to implement helloworld.GreeterServer.
type server struct {
	pb.UnimplementedDetailsServer
}

type CentLockMangStruct struct {
	clm *cms.CentLockMang
}

type creditData struct {
	transactionID string
	accountNumber string
	amount        int
}

type transactionLog struct {
	TransactionID string
	AccountNumber string
	Amount        int
	Type          string
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

func work(data RequestDataBank) (string, error) {
	db1 := config.DB
	db2 := config.DB2
	db3 := config.DB3
	db := db1

	acc, err := strconv.Atoi(data.AccountNumber)
	if err != nil {
		log.Printf("Error converting account number to integer: %v", err)
		return "", err
	}
	remains := acc % 3
	switch remains {
	case 0:
		db = db1
	case 1:
		db = db2
	case 2:
		db = db3
	}

	switch data.Type {
	case "debit":
		{
			log.Printf("Processing debit  request with DB operation: %v", data.AccountNumber)
			err_ := db.Ping()
			if err_ != nil {
				log.Printf("Error connecting to the database: %v", err_)
				msg, err := config.ConnectWithSql()
				if err != nil {
					return msg, err
				}
			}
			results, err := db.Query("SELECT Amount FROM bank_details WHERE AccountNumber = ?", data.AccountNumber)
			if err != nil {
				log.Fatalf(err.Error())
				return "", err
			}
			for results.Next() {
				var amount int
				// for each row, scan the result into our tag composite object
				err = results.Scan(&amount)
				if err != nil {
					log.Fatalf(err.Error())
					return "", err
				}
				if amount < data.Amount {
					log.Printf("Insufficient balance")
					return "Insufficient balance", nil
				} else {
					amount = amount - data.Amount
					_, err := db.Exec("UPDATE bank_details SET Amount = ? WHERE AccountNumber = ?", amount, data.AccountNumber)
					if err != nil {
						log.Fatalf(err.Error())
						return "", err
					}
					transactionLog := transactionLog{TransactionID: data.TransactionID, AccountNumber: data.AccountNumber, Amount: data.Amount, Type: data.Type}
					transactionString, _ := json.Marshal(transactionLog)
					config.Client.Index(config.IndexName, bytes.NewReader(transactionString))
					return "", nil
				}
			}

			return "No Records Found", nil
		}
	case "reverse":
		{
			log.Printf("Processing debit reversal with DB operation: %v", data.AccountNumber)
			db1 := config.DB
			db2 := config.DB2
			db3 := config.DB3
			db := db1

			acc, err := strconv.Atoi(data.AccountNumber)
			if err != nil {
				log.Printf("Error converting account number to integer: %v", err)
				return "", err
			}
			remains := acc % 3
			switch remains {
			case 0:
				db = db1
			case 1:
				db = db2
			case 2:
				db = db3
			}
			err_ := db.Ping()
			if err_ != nil {
				log.Printf("Error connecting to the database: %v", err_)
				msg, err := config.ConnectWithSql()
				if err != nil {
					return msg, err
				}
			}
			results, err := db.Query("SELECT Amount FROM bank_details WHERE AccountNumber = ?", data.AccountNumber)
			if err != nil {
				log.Fatalf(err.Error())
				return "", err
			}
			for results.Next() {
				var amount int
				// for each row, scan the result into our tag composite object
				err = results.Scan(&amount)
				if err != nil {
					log.Fatalf(err.Error())
					return "", err
				}
				amount = amount + data.Amount
				_, err := db.Exec("UPDATE bank_details SET Amount = ? WHERE AccountNumber = ?", amount, data.AccountNumber)
				if err != nil {
					log.Fatalf(err.Error())
					return "", err
				}
				transactionLog := transactionLog{TransactionID: data.TransactionID, AccountNumber: data.AccountNumber, Amount: data.Amount, Type: data.Type}
				transactionString, _ := json.Marshal(transactionLog)
				config.Client.Index(config.IndexName, bytes.NewReader(transactionString))
				return "Transaction Reversed", nil

			}

			return "No Records Found", nil
		}

	}
	return "", nil
}

func checkRequest(data RequestDataBank) (string, error) {
	// Search in ElasticSearch for documents with the same transaction ID and type

	Type := data.Type
	if data.Type == "reverse" {
		Type = "debit"
	}

	query := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": []interface{}{
					map[string]interface{}{"term": map[string]interface{}{"TransactionID": data.TransactionID}},
					map[string]interface{}{"term": map[string]interface{}{"Type": Type}},
				},
			},
		},
	}

	// Convert the query to JSON
	queryJSON, err := json.Marshal(query)
	if err != nil {
		log.Printf("Error marshaling query: %v", err)
		return "", err
	}

	// Search in ElasticSearch
	res, err := config.Client.Search(
		config.Client.Search.WithIndex(config.IndexName),
		config.Client.Search.WithBody(bytes.NewReader(queryJSON)),
	)
	if err != nil {
		log.Printf("Error searching in ElasticSearch: %v", err)
		return "", err
	}

	// Check if any documents were found
	response := map[string]interface{}{}
	err_ := json.NewDecoder(res.Body).Decode(&response)
	if err_ != nil {
		config.Logger.Print("Error Decoding the response")
		return "", err_
	}

	len := len(response["hits"].(map[string]interface{})["hits"].([]interface{}))
	log.Printf("Length of the response is %v", len)

	if len > 0 {
		log.Printf("Found existing document with the same transaction ID and type")
		return "Existing document found", nil
	}

	return "", nil
}

func processDebitRequest(RequestData RequestDataBank) (string, error) {

	log.Printf("Processing debit request: %v", RequestData)

	// p, err := cms.NewProcess(config.LeaderPort, RequestData.AccountNumber, RequestData.Type)
	// if err != nil {
	// 	log.Printf("client create error: %v.\n", err.Error())
	// 	return "", err
	// }
	//if config.IsLeader == "TRUE" {
	// accounts := lock_manager.GetLocksOnAvailableAccounts([]string{RequestData.AccountNumber})
	// fmt.Println("I have locks on:%v", accounts)
	// //perform the bank thing here

	// if len(accounts) != 0 {
	// 	able := lock_manager.ReleaseLocksOnAccounts(accounts)
	// 	if !able {
	// 		return "Error releasing locks", fmt.Errorf("Error releasing locks")
	// 	}
	// }
	//} else {
	// send request to leader
	res, err := checkRequest(RequestData)
	if err != nil {
		config.Logger.Print("Elastic Search is Down")
		return "Elastic Search is Down", err
	}
	if res == "Existing document found" {
		config.Logger.Print("Existing document found")
		return "Existing document found", nil
	}
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
		log.Fatalf(error.Error())
		return "Error sending request", fmt.Errorf("Error sending request")
	}
	//parse response into the  array of accounts
	var accounts []string
	err = json.NewDecoder(response.Body).Decode(&accounts)
	fmt.Println("%v", response.Body)

	if len(accounts) != 0 {
		msg, err := work(RequestData)

		req = lock_manager.Request{
			RequestType: "release",
			Accounts:    []string{RequestData.AccountNumber},
		}
		jsonData, err_ := json.Marshal(req)
		if err_ != nil {
			return "Error marshalling request", fmt.Errorf("Error marshalling request")
		}
		request, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
		request.Header.Set("Content-Type", "application/json; charset=UTF-8")

		client := &http.Client{}
		_, error_ := client.Do(request)
		if error_ != nil {
			log.Fatalf(error_.Error())
			return "Error sending request", fmt.Errorf("Error sending request")
		}

		if err != nil {
			return msg, err
		}
		if msg == "Insufficient balance" {
			return msg, nil
		}
		if msg == "No Records Found" {
			return msg, nil
		}
		return "Debit request processed", nil
	}
	return "Failed to get lock", nil
	// accounts := lock_manager.GetLocksOnAvailableAccounts([]string{RequestData.AccountNumber})

	// fmt.Println("%v", response.Body)
	// if err != nil {
	// 	fmt.Println("Error decoding response:", err)
	// 	return "Error decoding response", fmt.Errorf("Error decoding response")
	// }
	// fmt.Println("I have locks on:%v", accounts)
	//defer response.Body.Close()

}

// msg, err_ := p.Run(RequestData.AccountNumber, RequestData.Type, cms.RequestDataBank(RequestData))
// if err_ != nil {
// 	log.Printf("Debit request for transaction id %v failed with error: %v.\n", RequestData.TransactionID, err_.Error())
// 	return "", err_
// }

// if msg != "Debit Success" {
// 	log.Printf("Debit request for transaction id %v failed with error: %v.\n", RequestData.TransactionID, msg)
// 	return msg, nil
// }

//return "Debit request processed", nil

func addCreditRequest(data creditData) (string, error) {

	log.Printf("Inside addCreditRequest")
	if _, ok := creditLock[data.accountNumber]; !ok {
		creditLock[data.accountNumber] = &sync.Mutex{}
	}

	if _, ok := creditMap[data.accountNumber]; !ok {
		creditMap[data.accountNumber] = make([]creditData, 0)
	}
	creditLock[data.accountNumber].Lock()
	globalLock.Lock()
	creditMap[data.accountNumber] = append(creditMap[data.accountNumber], data)
	globalLock.Unlock()
	creditLock[data.accountNumber].Unlock()

	return "", nil
}

func processCreditRequest(RequestData RequestDataBank) (string, error) {
	log.Printf("Processing credit request: %v", RequestData)
	// p, err := cms.NewProcess(config.LeaderPort, RequestData.AccountNumber, RequestData.Type)
	// if err != nil {
	// 	log.Printf("client create error: %v.\n", err.Error())
	// 	return "", err
	// }

	data := creditData{transactionID: RequestData.TransactionID, accountNumber: RequestData.AccountNumber, amount: RequestData.Amount}
	_, err := addCreditRequest(data)
	if err != nil {
		log.Printf("Credit request for transaction id %v failed with error: %v.\n", data.transactionID, err.Error())
	}
	//return "Credit Success", nil

	// msg, err_ := p.Run(RequestData.AccountNumber, RequestData.Type, cms.RequestDataBank(RequestData))
	// if err_ != nil {
	// 	log.Printf("Credit request for transaction id %v failed with error: %v.\n", RequestData.TransactionID, err_.Error())
	// 	return "", err_
	// }

	// if msg != "Credit Success" {
	// 	log.Printf("Credit request for transaction id %v failed with error: %v.\n", RequestData.TransactionID, msg)
	// 	return msg, nil
	// }
	return "Credit request processed", nil
}

func processReverseRequest(RequestData RequestDataBank) (string, error) {
	log.Printf("Processing reverse request: %v", RequestData)
	// p, err := cms.NewProcess(config.LeaderPort, RequestData.AccountNumber, RequestData.Type)
	// if err != nil {
	// 	log.Printf("client create error: %v.\n", err.Error())
	// 	return "", err
	// }

	res, err := checkRequest(RequestData)
	if err != nil {
		config.Logger.Print("Elastic Search is Down")
		return "Elastic Search is Down", err
	}

	if res == "Existing document found" {

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
			log.Fatalf(error.Error())
			return "Error sending request", fmt.Errorf("Error sending request")
		}
		//parse response into the  array of accounts
		var accounts []string
		err = json.NewDecoder(response.Body).Decode(&accounts)
		fmt.Println("%v", response.Body)

		if len(accounts) != 0 {
			msg, err := work(RequestData)

			req = lock_manager.Request{
				RequestType: "release",
				Accounts:    []string{RequestData.AccountNumber},
			}
			jsonData, err_ := json.Marshal(req)
			if err_ != nil {
				return "Error marshalling request", fmt.Errorf("Error marshalling request")
			}
			request, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
			request.Header.Set("Content-Type", "application/json; charset=UTF-8")

			client := &http.Client{}
			_, error_ := client.Do(request)
			if error_ != nil {
				log.Fatalf(error_.Error())
				return "Error sending request", fmt.Errorf("Error sending request")
			}
			if err != nil {
				return msg, err
			}
			if msg != "Transaction Reversed" {
				log.Printf("Debit reverse for transaction id %v failed with error: %v.\n", RequestData.TransactionID, msg)
				return msg, nil
			}
			return "Transaction Reversed", nil
		}
	}

	return "Transaction Reversed", nil
}

func processGRPCMessage(msg string) (string, error) {

	log.Printf("Processing message: %v", msg)
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
	log.Printf("Received: %v", in.GetName())
	msg, err := processGRPCMessage(in.GetName())
	if err != nil {
		return &pb.Servermsg{Message: "Error processing message"}, err
	}
	return &pb.Servermsg{Message: msg}, nil
}

// func initializeLearderServer() {
// 	// initialize the leader server
// 	clm, err := cms.NewCentLockMang(config.LeaderPort)
// 	if err != nil {
// 		log.Printf("Start centralized server manager(%v) error: %v.\n", config.ServerID, err.Error())
// 		return
// 	}
// 	clms := CentLockMangStruct{clm: clm}
// 	go clms.clm.Start()
// }

func main() {
	config.LoadEnvData()
	// if config.IsLeader == "TRUE" {
	// 	go func() {
	// 		lock_manager.StartServer()
	// 	}()
	// }
	// go cms.CreditProcessing(config.LeaderPort)
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
