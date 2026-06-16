package agents

import (
	"fmt"
	"sync"

	"github.com/AtesIsf/monetary-simulator/internal/core"
)

/*
 * bank.go
 *
 * This file defines and implements a bank as an agent for this economy.
 */

type Bank struct {
	id core.AgentId
	debt uint32 // for simplicity, banks can only borrow from the central bank
	loans []*Loan
	loanLock sync.RWMutex

	demandDeposits map[uint32]int64
	ddLock sync.RWMutex
}

type Loan struct {
	from core.AgentId
	to core.AgentId
	initialAmount uint64
	remainingAmount uint64
	installment uint32
	interest uint32
}

func NewBank(id uint32) *Bank {
	var b Bank
	b.id.AType = core.Bank
	b.id.Id = id
	b.debt = 0
	b.demandDeposits = make(map[uint32]int64)
	return &b
}

// Issues a loan and transfers funds from the bank to the target
func (b *Bank) IssueLoan(ld *core.Ledger, target core.AgentId, 
												 amount int64, interest uint32) *Loan {

	b.loanLock.Lock()
	defer b.loanLock.Unlock()

	if ld.GetBalance(b.GetId()) - amount < 0 {
		return nil // TODO: you may want to return an error in the future
	}

	amountWithInterest := amount + (amount * int64(interest) / 100)

	loan := Loan {
		from: core.AgentId{ AType: core.Bank, Id: b.GetId() },
		to: target,
		initialAmount: uint64(amountWithInterest),
		remainingAmount: uint64(amountWithInterest),
		installment: 12,
		interest: interest,
	}
	loanPtr := &loan

	b.loans = append(b.loans, loanPtr)
	ld.Transfer(b.GetId(), target.Id, amount)
	return loanPtr
}

// To be used when repaying debt with demand deposits
func (b *Bank) DecreaseDemandDeposit(id uint32, amount int64) {
	b.ddLock.Lock()
	defer b.ddLock.Unlock()

	b.demandDeposits[id] -= amount
}

func (b *Bank) AddDemandDeposit(id uint32, amount int64, ld *core.Ledger)  {
	ld.Transfer(id, b.GetId(), amount)

	b.ddLock.Lock()
	defer b.ddLock.Unlock()

	b.demandDeposits[id] += amount
}

func (b *Bank) QueryDeposits(id uint32) int64 {
	b.ddLock.RLock()
	defer b.ddLock.RUnlock()

	return b.demandDeposits[id]
}

func (b *Bank) WithdrawAll(id uint32, ld *core.Ledger) {
	b.ddLock.Lock()
	defer b.ddLock.Unlock()

	deposits := b.demandDeposits[id]
	ld.Transfer(b.GetId(), id, deposits)
	b.demandDeposits[id] -= deposits
}

func (b *Bank) GetId() uint32 {
	return b.id.Id
}

func (b *Bank) Update(pol *core.Policies, ld *core.Ledger,
									tr core.MacroTracker, ticks uint64) core.UpdateReturn {
	// Annualize loans -> not really realistic
	if ticks % 12 != 0 {
		return core.Nothing
	}

	b.ddLock.Lock()
	defer b.ddLock.Unlock()

	rate := pol.GetInterestRate() - 2
	for id, amount := range b.demandDeposits {
		interest := amount * int64(rate) / 100
		b.demandDeposits[id] += interest
		// Pay interest from bank's ledger to depositor's ledger,
		// keeping the total money supply conserved.
		ld.Transfer(b.GetId(), id, interest)
	}

	return core.Nothing
}

func (b *Bank) GetType() core.AgentType {
	return core.Bank
}

func (b *Bank) Log() {
	fmt.Printf("Bank\n")
}
