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
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
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

type ResponseStruct struct {
	Message []string `json:"Message"`
}
type Request struct {
	RequestType string   `json:"requestType"`
	Accounts    []string `json:"accounts"`
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
	db := config.DB1

	acc, err := strconv.Atoi(data.AccountNumber)
	if err != nil {
		log.Printf("Error converting account number to integer: %v", err)
		return "", err
	}
	remains := acc % 3
	switch remains {
	case 0:
		db = config.DB1
	case 1:
		db = config.DB2
	case 2:
		db = config.DB3
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

				remains = acc % 3
				switch remains {
				case 0:
					db = config.DB1
				case 1:
					db = config.DB2
				case 2:
					db = config.DB3
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
			// db1 := config.DB1
			// db2 := config.DB2
			// db3 := config.DB3
			db := config.DB1

			acc, err := strconv.Atoi(data.AccountNumber)
			if err != nil {
				log.Printf("Error converting account number to integer: %v", err)
				return "", err
			}
			remains := acc % 3
			switch remains {
			case 0:
				db = config.DB1
			case 1:
				db = config.DB2
			case 2:
				db = config.DB3
			}
			err_ := db.Ping()
			if err_ != nil {
				log.Printf("Error connecting to the database: %v", err_)
				msg, err := config.ConnectWithSql()
				if err != nil {
					return msg, err
				}
				remains = acc % 3
				switch remains {
				case 0:
					db = config.DB1
				case 1:
					db = config.DB2
				case 2:
					db = config.DB3
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
	res, err := checkRequest(RequestData)
	if err != nil {
		config.Logger.Print("Elastic Search is Down")
		return "Elastic Search is Down", err
	}
	if res == "Existing document found" {
		config.Logger.Print("Existing document found")
		return "Existing document found", nil
	}
	url := fmt.Sprintf("http://%v:%v/get-lock", config.LeaderIPV4, config.LeaderPort)

	var req Request
	req = Request{
		RequestType: "request",
		Accounts:    []string{RequestData.AccountNumber},
	}
	log.Print(req)
	jsonData, err := json.Marshal(req)
	if err != nil {
		return "Error marshalling request", fmt.Errorf("Error marshalling request")
	}
	request, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	request.Header.Set("Content-Type", "application/json; charset=UTF-8")

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		log.Fatalf(err.Error())
		return "Error sending request", fmt.Errorf("Error sending request")
	}

	// Read the response body
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Fatalf("Error reading response body: %v", err)
		return "Error reading response body", fmt.Errorf("Error reading response body")
	}
	var ress ResponseStruct
	err = json.Unmarshal(body, &ress)
	if err != nil {
		log.Fatalf("Error unmarshalling response: %v", err)
		return "Error unmarshalling response", fmt.Errorf("Error unmarshalling response")
	}

	log.Printf("%v", ress)

	if len(ress.Message) != 0 {
		msg, err := work(RequestData)

		req = Request{
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

}

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

	data := creditData{transactionID: RequestData.TransactionID, accountNumber: RequestData.AccountNumber, amount: RequestData.Amount}
	_, err := addCreditRequest(data)
	if err != nil {
		log.Printf("Credit request for transaction id %v failed with error: %v.\n", data.transactionID, err.Error())
	}
	return "Credit request processed", nil
}

func processReverseRequest(RequestData RequestDataBank) (string, error) {
	log.Printf("Processing reverse request: %v", RequestData)

	res, err := checkRequest(RequestData)
	if err != nil {
		config.Logger.Print("Elastic Search is Down")
		return "Elastic Search is Down", err
	}

	if res == "Existing document found" {

		url := fmt.Sprintf("http://%v:%v/get-lock", config.LeaderIPV4, config.LeaderPort)

		var req Request
		req = Request{
			RequestType: "request",
			Accounts:    []string{RequestData.AccountNumber},
		}
		jsonData, err := json.Marshal(req)
		if err != nil {
			return "Error marshalling request", fmt.Errorf("Error marshalling request")
		}
		request, error := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
		if error != nil {
			log.Printf("Error creating request: %v", error)
			return "Error creating request", fmt.Errorf("Error creating request")
		}
		request.Header.Set("Content-Type", "application/json; charset=UTF-8")

		client := &http.Client{}
		response, error := client.Do(request)
		log.Printf("Response: %v", response)
		if error != nil {
			log.Fatalf(error.Error())
			return "Error sending request", fmt.Errorf("Error sending request")
		}

		body, err := ioutil.ReadAll(response.Body)
		if err != nil {
			log.Fatalf("Error reading response body: %v", err)
			return "Error reading response body", fmt.Errorf("Error reading response body")
		}
		var ress ResponseStruct
		err = json.Unmarshal(body, &ress)
		if err != nil {
			log.Fatalf("Error unmarshalling response: %v", err)
			return "Error unmarshalling response", fmt.Errorf("Error unmarshalling response")
		}

		//parse response into the  array of account

		if len(ress.Message) != 0 {
			msg, err := work(RequestData)

			req = Request{
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

func CreditProcessing(port int) {
	for {

		globalLock.Lock()
		creditAccountNumbers = make([]string, 0)
		for key, _ := range creditMap {
			creditAccountNumbers = append(creditAccountNumbers, key)
		}
		globalLock.Unlock()

		url := fmt.Sprintf("http://%v:%v/get-lock", config.LeaderIPV4, config.LeaderPort)

		var req Request
		req = Request{
			RequestType: "request",
			Accounts:    creditAccountNumbers,
		}
		jsonData, err := json.Marshal(req)
		if err != nil {
			//return "Error marshalling request", fmt.Errorf("Error marshalling request")
			continue
		}
		request, error := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
		request.Header.Set("Content-Type", "application/json; charset=UTF-8")

		client := &http.Client{}
		response, error := client.Do(request)

		if error != nil {
			log.Fatalf(error.Error())
			//return "Error sending request", fmt.Errorf("Error sending request")
			continue
		}

		body, err := ioutil.ReadAll(response.Body)
		if err != nil {
			log.Fatalf("Error reading response body: %v", err)
			continue
			//return "Error reading response body", fmt.Errorf("Error reading response body")
		}

		var ress ResponseStruct
		err = json.Unmarshal(body, &ress)
		if err != nil {
			log.Fatalf("Error unmarshalling response: %v", err)
			continue
		}
		err1 := config.DB1.Ping()
		err2 := config.DB2.Ping()
		err3 := config.DB3.Ping()

		if err1 != nil || err2 != nil || err3 != nil {
			log.Printf("Error connecting to the database: %v", err1)
			_, err := config.ConnectWithSql()
			if err != nil {
				continue
			}
		}

		for _, accountNumber := range ress.Message {

			creditLock[accountNumber].Lock()
			Failed_Transaction := make([]creditData, 0)
			db := config.DB1
			acc, err := strconv.Atoi(accountNumber)
			if err != nil {
				log.Printf("Error converting account number to integer: %v", err)
				continue
			}
			remains := acc % 3
			switch remains {
			case 0:
				db = config.DB1
			case 1:
				db = config.DB2
			case 2:
				db = config.DB3
			}
			sum := 0
			//transactionLogs := make([]transactionLog, 0)
			for _, data := range creditMap[accountNumber] {
				sum = sum + data.amount
			}

			results, err := db.Query("SELECT Amount FROM bank_details WHERE AccountNumber = ?", accountNumber)
			if err != nil {
				log.Fatal(err)
				Failed_Transaction = creditMap[accountNumber]
				continue
			}

			for results.Next() {
				var amount int
				err = results.Scan(&amount)
				if err != nil {
					log.Fatal(err)
					Failed_Transaction = creditMap[accountNumber]
					continue
				}
				amount = amount + sum
				_, err := db.Exec("UPDATE bank_details SET Amount = ? WHERE AccountNumber = ?", amount, accountNumber)
				if err != nil {
					log.Fatal(err)
					Failed_Transaction = creditMap[accountNumber]
					continue
				}
				//creditMap[accountNumber] = append(creditMap[accountNumber][:index], creditMap[accountNumber][index+1:]...)

				//return "", nil
			}

			for _, data := range creditMap[accountNumber] {
				//sum=sum+data.amount
				transactionLog := transactionLog{TransactionID: data.transactionID, AccountNumber: data.accountNumber, Amount: data.amount, Type: "credit"}
				transactionString, _ := json.Marshal(transactionLog)
				config.Client.Index(config.IndexName, bytes.NewReader(transactionString))
				//transactionLogs = append(transactionLogs, transactionLog)
			}

			if len(Failed_Transaction) == 0 {
				delete(creditMap, accountNumber)
			} else {
				creditMap[accountNumber] = Failed_Transaction
			}
			creditLock[accountNumber].Unlock()
		}

		req = Request{
			RequestType: "release",
			Accounts:    creditAccountNumbers,
		}
		jsonData, err = json.Marshal(req)
		if err != nil {
			//return "Error marshalling request", fmt.Errorf("Error marshalling request")
			continue
		}
		request, error = http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
		request.Header.Set("Content-Type", "application/json; charset=UTF-8")

		client = &http.Client{}
		response, error = client.Do(request)

		if error != nil {
			log.Fatalf(error.Error())
			//return "Error sending request", fmt.Errorf("Error sending request")
			continue
		}
		time.Sleep(3 * time.Second)
	}
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
	err := config.LoadEnvData()
	if err != nil {
		log.Fatalf("Error loading .env file")
	}
	go CreditProcessing(config.LeaderPort)
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
	log.Printf("Server started on port %v", *port)
	pb.RegisterDetailsServer(s, &server{})
	log.Printf("server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
