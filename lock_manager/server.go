package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
)

type Request struct {
	RequestType string   `json:"requestType"`
	Accounts    []string `json:"accounts"`
}

type AccountLock struct {
	Locked bool
	Mutex  sync.Mutex
}

type ResponseStruct struct {
	Message []string `json:"Message"`
}

var LockStats = sync.Map{}

func GetLocksOnAvailableAccounts(accounts []string) []string {
	log.Printf("Getting locks on accounts: %v", accounts)
	var availableAccounts []string
	for _, account := range accounts {
		val, ok := LockStats.LoadOrStore(account, &AccountLock{})
		accountLock := val.(*AccountLock)
		accountLock.Mutex.Lock()
		if !ok || !accountLock.Locked {
			availableAccounts = append(availableAccounts, account)
			accountLock.Locked = true
		}
		accountLock.Mutex.Unlock()
	}
	log.Printf("Locks acquired on accounts: %v", availableAccounts)
	return availableAccounts
}

func ReleaseLocksOnAccounts(accounts []string) {
	log.Printf("Releasing locks on accounts: %v", accounts)
	for _, account := range accounts {
		val, ok := LockStats.Load(account)
		if ok {
			accountLock := val.(*AccountLock)
			accountLock.Mutex.Lock()
			accountLock.Locked = false
			accountLock.Mutex.Unlock()
		}
	}
	log.Printf("Locks released on accounts: %v", accounts)
}

func main() {
	e := echo.New()
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	e.POST("/get-lock", func(c echo.Context) error {
		var req Request
		if err := c.Bind(&req); err != nil {
			return err
		}
		if req.RequestType == "request" {
			accounts := GetLocksOnAvailableAccounts(req.Accounts)
			if accounts == nil {
				accounts = []string{}
			}
			acc := ResponseStruct{Message: accounts}
			a, err := json.Marshal(acc)
			if err != nil {
				return err
			}
			log.Printf("account %v", accounts)
			log.Printf("c.JSON %v", string(a))
			return c.JSON(http.StatusOK, acc)
		} else if req.RequestType == "release" {
			ReleaseLocksOnAccounts(req.Accounts)
			return c.JSON(http.StatusOK, "Locks released")
		} else {
			return c.JSON(http.StatusBadRequest, "Invalid request type")
		}
	})
	val, _ := strconv.Atoi(os.Getenv("LEADERPORT"))
	e.Logger.Fatal(e.Start(fmt.Sprintf(":%d", val)))
}
