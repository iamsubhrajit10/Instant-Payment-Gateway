// package main

// import (
// 	"log"
// 	"net/http"
// 	"sync"

// 	"github.com/labstack/echo/v4"
// )

// type Request struct {
// 	RequestType string   `json:"requestType"`
// 	Accounts    []string `json:"accounts"`
// }

// var LockStats = make(map[string]bool)

// func GetLocksOnAvailableAccounts(accounts []string) []string {
// 	mu := sync.Mutex{}
// 	log.Printf("hello 3")
// 	mu.Lock()

// 	var availableAccounts []string
// 	log.Printf("accounts: %v", accounts)
// 	for _, account := range accounts {
// 		log.Printf("account1: %v", account)
// 		if contains(LockStats, account) {
// 			if LockStats[account] == false {
// 				log.Printf("account2: %v", account)
// 				availableAccounts = append(availableAccounts, account)
// 				LockStats[account] = true
// 			}
// 		} else {
// 			LockStats[account] = true
// 			log.Printf("account3: %v", account)
// 			availableAccounts = append(availableAccounts, account)
// 		}
// 	}

// 	mu.Unlock()
// 	return availableAccounts

// }

// func contains(s map[string]bool, account string) bool {
// 	_, ok := s[account]
// 	return ok
// }

// func ReleaseLocksOnAccounts(accounts []string) bool {
// 	mu := sync.Mutex{}
// 	mu.Lock()
// 	for _, account := range accounts {
// 		LockStats[account] = false
// 	}
// 	mu.Unlock()
// 	return true

// }

// func main() {

// 	e := echo.New()
// 	e.POST("/get-lock", func(c echo.Context) error {
// 		log.Printf("hello 1")
// 		var req Request
// 		if err := c.Bind(req); err != nil {
// 			return err
// 		}
// 		if req.RequestType == "request" {
// 			log.Printf("hello 2")
// 			accounts := GetLocksOnAvailableAccounts(req.Accounts)
// 			return c.JSON(http.StatusOK, accounts)
// 		} else if req.RequestType == "release" {
// 			ReleaseLocksOnAccounts(req.Accounts)
// 			return c.JSON(http.StatusOK, "Locks released")
// 		} else {
// 			return c.JSON(http.StatusBadRequest, "Invalid request type")
// 		}
// 	})
// 	e.Logger.Fatal(e.Start(":1323"))
// }

package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

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
	e.POST("/get-lock", func(c echo.Context) error {
		var req Request
		if err := c.Bind(&req); err != nil {
			return err
		}
		if req.RequestType == "request" {
			accounts := GetLocksOnAvailableAccounts(req.Accounts)
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
	e.Logger.Fatal(e.Start(":1323"))
}
