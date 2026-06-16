package core

import (
	"fmt"
	"sync"
)

/*
 * ledger.go
 *
 * This file simulates a ledger, which records transactions in the economy.
 * In general terms, it records the exchange of goods, services, and funds.
 */

// the amount households are paid per turn by firms
const StandardWage = 20
const MinWage = 1
// the base price of a good or service
const Price = 20
const MinPrice = 1
const MaxLiquidity = 150

type Ledger struct {
	balanceMutex sync.RWMutex
	balances map[uint32]int64 // maps agent id to balance
	// banks map[uint32]uint32 // maps agent id to bank id

	wageMutex sync.RWMutex
	wageExpectations map[uint32]int64
}

func (ld *Ledger) Init() {
	ld.balances = make(map[uint32]int64)
	ld.wageExpectations = make(map[uint32]int64)
}

func (ld *Ledger) GetBalance(id uint32) int64 {
	ld.balanceMutex.RLock()
	defer ld.balanceMutex.RUnlock()
	return ld.balances[id]
}

func (ld *Ledger) AddToBalance(id uint32, amount int64) {
	ld.balanceMutex.Lock()
	defer ld.balanceMutex.Unlock()
	ld.balances[id] += amount
}

// Amount is in cents! This prevents floating point errors.
func (ld *Ledger) Transfer(from uint32, to uint32, amount int64) error {
	ld.balanceMutex.Lock()
	defer ld.balanceMutex.Unlock()

	if ld.balances[from] < amount {
		return fmt.Errorf("insufficient balance: agent %d has %d, tried to transfer %d", from, ld.balances[from], amount)
	}

	ld.balances[from] -= amount
	ld.balances[to] += amount
	return nil
}

func (ld *Ledger) PrintBalances() {
	fmt.Println("\n---- Balances ----")
	for key, value := range ld.balances {
		fmt.Printf("Id: %d -- %d\n", key, value)
	}
}

func (ld *Ledger) GetWageExpectation(id uint32) int64 {
	ld.wageMutex.RLock()
	defer ld.wageMutex.RUnlock()
	return ld.wageExpectations[id]
}

func (ld *Ledger) SetWageExpectation(id uint32, wage int64) {
	ld.wageMutex.Lock()
	defer ld.wageMutex.Unlock()
	ld.wageExpectations[id] = wage
}

