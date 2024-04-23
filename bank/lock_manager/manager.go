package lock_manager

import (
	"log"
	"sync"
)

var LockStats = make(map[string]bool)

func GetLocksOnAvailableAccounts(accounts []string) []string {
	mu := sync.Mutex{}
	mu.Lock()

	var availableAccounts []string
	log.Print("accounts: %v", accounts)
	for _, account := range accounts {
		log.Print("account1: %v", account)
		if contains(LockStats, account) {
			if LockStats[account] == false {
				log.Print("account2: %v", account)
				availableAccounts = append(availableAccounts, account)
				LockStats[account] = true
			}
		} else {
			LockStats[account] = true
			log.Print("account3: %v", account)
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
