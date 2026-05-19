package core

import (
	"sync"
)

/*
 * ledger.go
 *
 * This file simulates a ledger, which records transactions in the economy.
 * In general terms, it records the exchange of goods, services, and funds.
 */

// the amount households are paid per turn by firms
const Wage = 20

type Ledger struct {
	mutex sync.RWMutex
	balances map[uint32]int64 // maps agent id to balance
	// banks map[uint32]uint32 // maps agent id to bank id
}

func (ld *Ledger) Init() {
	ld.balances = make(map[uint32]int64)
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

