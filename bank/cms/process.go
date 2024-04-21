package cms

import (
	"bank/config"
	"bytes"
	"encoding/json"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type RequestDataBank struct {
	TransactionID string
	AccountNumber string
	IFSCCode      string
	HolderName    string
	Amount        int
	Type          string
}

type process struct {
	dl             *dislock
	port           int
	Type           string
	accountNumbers string
}

type transactionLog struct {
	TransactionID string
	AccountNumber string
	Amount        int
	Type          string
}

type creditData struct {
	transactionID string
	accountNumber string
	amount        int
}

var creditMap map[string][]creditData = make(map[string][]creditData)
var creditLock map[string]*sync.Mutex = make(map[string]*sync.Mutex)
var globalLock *sync.Mutex = &sync.Mutex{}
var creditAccountNumbers []string

func NewProcess(port int, accountNumber string, Type string) (*process, error) {
	p := &process{port: port, accountNumbers: accountNumber, Type: Type}
	dl, err := NewDislock(port)
	if err != nil {
		config.Logger.Printf(" Account Number :- (%v) create error: %v.\n", accountNumber, err.Error())
		return nil, err
	}
	p.dl = dl
	return p, nil
}

func (p *process) checkRequest(data RequestDataBank) (string, error) {
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
		config.Logger.Printf("Error marshaling query: %v", err)
		return "", err
	}

	// Search in ElasticSearch
	res, err := config.Client.Search(
		config.Client.Search.WithIndex(config.IndexName),
		config.Client.Search.WithBody(bytes.NewReader(queryJSON)),
	)
	if err != nil {
		config.Logger.Printf("Error searching in ElasticSearch: %v", err)
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
	config.Logger.Printf("Length of the response is %v", len)

	if len > 0 {
		config.Logger.Printf("Found existing document with the same transaction ID and type")
		return "Existing document found", nil
	}

	return "", nil
}

func addCreditRequest(data creditData) (string, error) {

	config.Logger.Printf("Inside addCreditRequest")
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

func CreditProcessing(port int) {
	for {

		globalLock.Lock()
		creditAccountNumbers = make([]string, 0)
		for key, _ := range creditMap {
			creditAccountNumbers = append(creditAccountNumbers, key)
		}
		globalLock.Unlock()

		dl, err := NewDislock(port)
		accountNumbers, err := dl.Acquire(creditAccountNumbers, "credit")
		if err != nil {
			config.Logger.Printf(" Account Number :- (%v) create error: %v.\n", creditAccountNumbers, err.Error())
			continue
			//return nil, err
		}

		err_ := config.DB.Ping()
		if err_ != nil {
			config.Logger.Printf("Error connecting to the database: %v", err_)
			_, err := config.ConnectWithSql()
			if err != nil {
				continue
			}
		}

		for _, accountNumber := range accountNumbers {

			creditLock[accountNumber].Lock()
			Failed_Transaction := make([]creditData, 0)
			for index, data := range creditMap[accountNumber] {

				//config.Logger.Printf("Processing credit request: %v", data.AccountNumber)
				// handle error
				results, err := config.DB.Query("SELECT Amount FROM bank_details WHERE AccountNumber = ?", data.accountNumber)
				if err != nil {
					config.Logger.Fatal(err)
					Failed_Transaction = append(Failed_Transaction, data)
					continue
				}
				for results.Next() {
					var amount int
					err = results.Scan(&amount)
					if err != nil {
						config.Logger.Fatal(err)
						Failed_Transaction = append(Failed_Transaction, data)
						continue
						// proper error handling instead of panic in your app
						//return "", err
					}
					amount = amount + data.amount
					_, err := config.DB.Exec("UPDATE bank_details SET Amount = ? WHERE AccountNumber = ?", amount, data.accountNumber)
					if err != nil {
						config.Logger.Fatal(err)
						Failed_Transaction = append(Failed_Transaction, data)
						continue
					}
					creditMap[accountNumber] = append(creditMap[accountNumber][:index], creditMap[accountNumber][index+1:]...)
					transactionLog := transactionLog{TransactionID: data.transactionID, AccountNumber: data.accountNumber, Amount: data.amount, Type: "credit"}
					transactionString, _ := json.Marshal(transactionLog)
					config.Client.Index(config.IndexName, bytes.NewReader(transactionString))
					//return "", nil
				}
				//return "No Records Found", nil
			}
			if len(Failed_Transaction) == 0 {
				delete(creditMap, accountNumber)
			} else {
				creditMap[accountNumber] = Failed_Transaction
			}

			creditLock[accountNumber].Unlock()
		}

		err = dl.Release(accountNumbers, "credit")
		if err != nil {
			config.Logger.Printf(" Account Number :- (%v) create error: %v.\n", creditAccountNumbers, err.Error())
			continue
			//return nil, err
		}
		time.Sleep(3 * time.Second)
	}
}

func (p *process) work(data RequestDataBank) (string, error) {
	switch data.Type {
	case "debit":
		{
			config.Logger.Printf("Processing debit  request with DB operation: %v", data.AccountNumber)
			err_ := config.DB.Ping()
			if err_ != nil {
				config.Logger.Printf("Error connecting to the database: %v", err_)
				msg, err := config.ConnectWithSql()
				if err != nil {
					return msg, err
				}
			}
			results, err := config.DB.Query("SELECT Amount FROM bank_details WHERE AccountNumber = ?", data.AccountNumber)
			if err != nil {
				config.Logger.Fatal(err)
				return "", err
			}
			for results.Next() {
				var amount int
				// for each row, scan the result into our tag composite object
				err = results.Scan(&amount)
				if err != nil {
					config.Logger.Fatal(err)
					return "", err
				}
				if amount < data.Amount {
					config.Logger.Printf("Insufficient balance")
					return "Insufficient balance", nil
				} else {
					amount = amount - data.Amount
					_, err := config.DB.Exec("UPDATE bank_details SET Amount = ? WHERE AccountNumber = ?", amount, data.AccountNumber)
					if err != nil {
						config.Logger.Fatal(err)
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
			config.Logger.Printf("Processing debit reversal with DB operation: %v", data.AccountNumber)
			err_ := config.DB.Ping()
			if err_ != nil {
				config.Logger.Printf("Error connecting to the database: %v", err_)
				msg, err := config.ConnectWithSql()
				if err != nil {
					return msg, err
				}
			}
			results, err := config.DB.Query("SELECT Amount FROM bank_details WHERE AccountNumber = ?", data.AccountNumber)
			if err != nil {
				config.Logger.Fatal(err)
				return "", err
			}
			for results.Next() {
				var amount int
				// for each row, scan the result into our tag composite object
				err = results.Scan(&amount)
				if err != nil {
					config.Logger.Fatal(err)
					return "", err
				}
				amount = amount + data.Amount
				_, err := config.DB.Exec("UPDATE bank_details SET Amount = ? WHERE AccountNumber = ?", amount, data.AccountNumber)
				if err != nil {
					config.Logger.Fatal(err)
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

func (p *process) Run(accountNumber string, Type string, data RequestDataBank) (string, error) {

	config.Logger.Printf("Transaction  data: %v\n", data)
	var err error

	res, err := p.checkRequest(data)
	if err != nil {
		config.Logger.Print("Elastic Search is Down")
		return "Elastic Search is Down", err
	}
	if res == "Existing document found" {
		config.Logger.Printf("Found existing document with the same transaction ID and type")
		if Type != "reverse" {
			return "Found existing document with the same transaction ID and type", nil
		} else {

			aNo := make([]string, 0)
			aNo = append(aNo, data.AccountNumber)
			_, err = p.dl.Acquire(aNo, Type) // if any process still in critical section, it will block.
			if err != nil {
				config.Logger.Printf("(%v) fail to acquire lock for type (%v): %v.\n", accountNumber, Type, err.Error())
				return "", err
			}
			config.Logger.Printf("Account Number (%v) entered the critical section for (%v).\n", accountNumber, Type)
			msg, err := p.work(data) // ignore any failure occurs in this stage temporaily.
			err_ := p.dl.Release(aNo, Type)
			if err_ != nil {
				config.Logger.Printf("(%v) fail to release lock of type  %v: %v.\n", accountNumber, Type, err.Error())
				return "", err_
			}

			if err != nil {
				return "", err
			}
			return msg, nil
		}
	}

	if Type == "debit" {

		aNo := make([]string, 0)
		aNo = append(aNo, accountNumber)
		_, err = p.dl.Acquire(aNo, Type) // if any process still in critical section, it will block.
		if err != nil {
			config.Logger.Printf("(%v) fail to acquire lock for type (%v): %v.\n", accountNumber, Type, err.Error())
			return "", err
		}

		config.Logger.Printf("Account Number (%v) entered the critical section for (%v).\n", accountNumber, Type)
		msg, err := p.work(data) // ignore any failure occurs in this stage temporaily.
		err_ := p.dl.Release(aNo, Type)
		if err_ != nil {
			config.Logger.Printf("(%v) fail to release lock of type  %v: %v.\n", accountNumber, Type, err.Error())
			return "", err_
		}

		if err != nil {
			return "", err
		}

		if msg == "Insufficient balance" {
			return msg, nil
		}
		if msg == "No Records Found" {
			return msg, nil
		}
		return "Debit Success", nil
	} else {
		data := creditData{transactionID: data.TransactionID, accountNumber: data.AccountNumber, amount: data.Amount}
		_, err := addCreditRequest(data)
		if err != nil {
			config.Logger.Printf("Credit request for transaction id %v failed with error: %v.\n", data.transactionID, err.Error())
		}
		return "Credit Success", nil
	}
}

// the method is not good in usage logical, because the lock will automatically close when process called Release.
// so it just mainly facilitate our testing cases.
func (p *process) Close() error {
	if err := p.dl.Close(); err != nil {
		return err
	}
	return nil
}
