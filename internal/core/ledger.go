package core

import "sync"

/*
 * ledger.go
 *
 * This file simulates a ledger, which records transactions in the economy.
 * In general terms, it records the exchange of goods, services, and funds.
 */

type Ledger struct {
	mutex sync.RWMutex
	balances map[uint32]int64
}

// default balance is 100 for now
func (ld *Ledger) Populate(nAgents uint32) {
	ld.balances = make(map[uint32]int64)
	for i := range nAgents {
		ld.balances[i] = 100
	}
}

func (ld *Ledger) GetBalance(id uint32) int64 {
	ld.mutex.RLock()
	defer ld.mutex.RUnlock()
	return ld.balances[id]
}

// Amount is in cents! This prevents floating point errors.
func (ld *Ledger) Transfer(from uint32, to uint32, amount int64) {
	ld.mutex.Lock()
	defer ld.mutex.Unlock()

	// Allowing negative balances
	// I guess this would also work for borrowing, given a negative amount
	ld.balances[from] -= amount
	ld.balances[to] += amount
}
