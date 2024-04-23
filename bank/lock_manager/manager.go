package lock_manager

import (
	"sync"
)

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
