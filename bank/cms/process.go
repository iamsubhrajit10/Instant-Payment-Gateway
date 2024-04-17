package cms

import (
	"bank/config"
	"database/sql"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type RequestData struct {
	TransactionID string
	AccountNumber string
	Amount        int
	Type          string
}

const (
	localTaskIntervalLow     = 100
	localTaskIntervalHigh    = 300
	criticalTaskIntervalLow  = 100
	criticalTaskIntervalHigh = 200
)

type process struct {
	dl            *dislock
	pid           int // not really PID but identify itself , used in send message
	port          int
	lockManagerID int // centralized lock manager server pid.
	logger        *log.Logger

	// stat info
	latency int64 //elapsed time between making a request and being able to enter the critical section
}

var validateLogger *log.Logger = CreateLog("log/validateGlobalCount.log", "")

// just for facilitating our testing cases.
var globalCnt int
var globalCntArray []int

func CreateLog(fileName, header string) *log.Logger {
	newpath := filepath.Join(".", "log")
	os.MkdirAll(newpath, os.ModePerm)
	serverLogFile, _ := os.OpenFile(fileName, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	return log.New(serverLogFile, header, log.Lmicroseconds|log.Lshortfile)
}

func NewProcess(port, pid, lockManagerID int) (*process, error) {
	p := &process{port: port, pid: pid, lockManagerID: lockManagerID}
	p.logger = CreateLog("log/process_"+strconv.Itoa(pid)+".log", "[process]")
	dl, err := NewDislock(port, pid)
	if err != nil {
		p.logger.Printf("process(%v) create error: %v.\n", p.pid, err.Error())
		return nil, err
	}
	p.dl = dl
	return p, nil
}

// func connectWithSql() (*sql.DB, string, error) {
// 	db, err := sql.Open("mysql", "root:root@tcp(127.0.0.1:3306)/upi")
// 	if err != nil {
// 		log.Fatal(err)
// 		return nil, "", err
// 	}
// 	//defer DB.Close()
// 	if db != nil {
// 		log.Printf("DB is not nil")
// 		err = db.Ping()
// 		log.Printf("DB is not nil")
// 		if err != nil {
// 			log.Fatal(err)
// 			return nil, "", err
// 		}

// 		log.Println("Successfully connected to MySQL database")
// 		return db, "Success", nil
// 	}
// 	return nil, "", nil
// }

func (p *process) work(data RequestData, db *sql.DB) (string, error) {
	switch data.Type {
	case "debit":
		{

			// DB, _, err := connectWithSql()
			// log.Printf("Processing debit request1: %v", data.AccountNumber)
			// err_ := DB.Ping()
			// log.Printf("Processing debit request1: %v", err_)
			// log.Printf("Processing debit request2: %v", data)
			// // handle error
			// if err != nil {
			// 	log.Printf("Error connecting to the database: %v", err)
			// 	panic(err)
			// }

			log.Printf("Pong\n")
			results, err := db.Query("SELECT Amount FROM bank_details WHERE Account_number = ?", data.AccountNumber)
			if err != nil {
				//log.Fatal(err)
				return "", err
			}

			for results.Next() {
				var amount int
				// for each row, scan the result into our tag composite object
				err = results.Scan(&amount)
				if err != nil {
					log.Fatal(err)
					// proper error handling instead of panic in your app
					return "", err
				}
				log.Printf("Processing debit request3: %v", amount)
				if amount < data.Amount {
					log.Printf("Insufficient balance")
					return "Insufficient balance", nil
				} else {
					amount = amount - data.Amount
					_, err := db.Exec("UPDATE bank_details SET Amount = ? WHERE Account_number = ?", amount, data.AccountNumber)
					if err != nil {
						log.Fatal(err)
						return "", err
					}
					return "", nil
				}
			}

			return "No Records Found", nil
		}

	case "credit":
		{
			// DB, _, err := connectWithSql()
			// log.Printf("Processing debit request1: %v", data.AccountNumber)
			// err_ := DB.Ping()
			// log.Printf("Processing debit request1: %v", err_)
			// log.Printf("Processing debit request2: %v", data)
			// // handle error
			// if err != nil {
			// 	log.Printf("Error connecting to the database: %v", err)
			// 	panic(err)
			// }

			// log.Printf("Pong\n")
			results, err := config.DB.Query("SELECT Amount FROM bank_details WHERE Account_number = ?", data.AccountNumber)
			if err != nil {
				log.Fatal(err)
				return "", err
			}

			for results.Next() {
				var amount int
				// for each row, scan the result into our tag composite object
				err = results.Scan(&amount)
				if err != nil {
					log.Fatal(err)
					// proper error handling instead of panic in your app
					return "", err
				}
				amount = amount + data.Amount
				_, err := config.DB.Exec("UPDATE bank_details SET Amount = ? WHERE Account_number = ?", amount, data.AccountNumber)
				if err != nil {
					log.Fatal(err)
					return "", err
				}
				return "", nil
			}
			return "No Records Found", nil
		}
	}
	return "", nil
}

func (p *process) Run(msgContent string, data RequestData, db *sql.DB) (string, error) {
	// if w != nil {
	// 	p.work = w
	// } else {
	// 	p.work = p.defaultWork
	// }
	// do lock task
	log.Printf("request data: %v\n", data)
	var err error
	//startTime := time.Now().Unix()
	// begin to enter critical section, acquire lock first.
	err = p.dl.Acquire(p.lockManagerID, msgContent) // if any process still in critical section, it will block.
	// request failure
	if err != nil {
		log.Printf("process(%v) fail to acquire lock: %v.\n", p.pid, err.Error())
		return "", err
	}
	// endTime := time.Now().Unix()
	// p.latency = endTime - startTime
	log.Printf("process(%v) entered the critical section at %v.\n", p.pid, time.Unix(time.Now().Unix(), 0).Format("2006-01-02 15:04:05"))
	// success, excute critical section code, operate shared resources
	msg, err := p.work(data, db) // ignore any failure occurs in this stage temporaily.
	// exit critical section, release lock first.
	err_ := p.dl.Release(p.lockManagerID, msgContent+"[Release]")
	if err_ != nil {
		p.logger.Printf("process(%v) fail to release lock: %v.\n", p.pid, err.Error())
		return "", err_
	}

	if err != nil {
		return "", err
	}
	if data.Type == "debit" {
		if msg == "Insufficient balance" {
			return msg, nil
		}
		if msg == "No Records Found" {
			return msg, nil
		}
		return "Debit Success", nil
	}
	if data.Type == "credit" {
		if msg == "No Records Found" {
			return msg, nil
		}
		return "Credit Success", nil
	}

	log.Printf("process(%v) exited the critical section.\n", p.pid)
	return "", nil
}

// the method is not good in usage logical, because the lock will automatically close when process called Release.
// so it just mainly facilitate our testing cases.
func (p *process) Close() error {
	if err := p.dl.Close(); err != nil {
		return err
	}
	return nil
}
