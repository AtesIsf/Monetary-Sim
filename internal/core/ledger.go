package core

import (
	"math/rand/v2"
	"sync"
)

/*
 * ledger.go
 *
 * This file simulates a ledger, which records transactions in the economy.
 * In general terms, it records the exchange of goods, services, and funds.
 */

type Ledger struct {
	mutex sync.RWMutex
	balances map[uint32]int64 // maps agent id to balance
	// banks map[uint32]uint32 // maps agent id to bank id
}

// default balance is 1000-1500 for now
func (ld *Ledger) Populate(nAgents uint32) {
	ld.balances = make(map[uint32]int64)
	for i := range nAgents {
		ld.balances[i] = rand.Int64N(500) + 1000
	}
}

func (ld *Ledger) GetBalance(id uint32) int64 {
	ld.mutex.RLock()
	defer ld.mutex.RUnlock()
	return ld.balances[id]
}

func (ld *Ledger) AddToBalance(id uint32, amount int64) {
	ld.mutex.Lock()
	defer ld.mutex.Unlock()
	ld.balances[id] += amount
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

