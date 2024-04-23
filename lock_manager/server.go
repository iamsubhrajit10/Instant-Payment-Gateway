package main

import (
	"net/http"
	"sync"

	"github.com/labstack/echo/v4"
)

type Request struct {
	RequestType string   `json:"requestType"`
	Accounts    []string `json:"accounts"`
}

var LockStats = make(map[string]bool)

func GetLocksOnAvailableAccounts(accounts []string) []string {
	mu := sync.Mutex{}
	mu.Lock()

	var availableAccounts []string

	for _, account := range accounts {
		if contains(LockStats, account) {
			if LockStats[account] == false {
				availableAccounts = append(availableAccounts, account)
				LockStats[account] = true
			}
		} else {
			LockStats[account] = true
			availableAccounts = append(availableAccounts, account)
		}
	}

	mu.Unlock()
	return availableAccounts

}

func contains(s map[string]bool, account string) bool {
	_, ok := s[account]
	return ok
}

func ReleaseLocksOnAccounts(accounts []string) bool {
	mu := sync.Mutex{}
	mu.Lock()
	for _, account := range accounts {
		LockStats[account] = false
	}
	mu.Unlock()
	return true

}

func main() {

	e := echo.New()
	e.POST("/get-lock", func(c echo.Context) error {
		var req Request
		if err := c.Bind(req); err != nil {
			return err
		}
		if req.RequestType == "request" {
			accounts := GetLocksOnAvailableAccounts(req.Accounts)
			return c.JSON(http.StatusOK, accounts)
		} else if req.RequestType == "release" {
			ReleaseLocksOnAccounts(req.Accounts)
			return c.JSON(http.StatusOK, "Locks released")
		} else {
			return c.JSON(http.StatusBadRequest, "Invalid request type")
		}
	})
	e.Logger.Fatal(e.Start(":1323"))
}
