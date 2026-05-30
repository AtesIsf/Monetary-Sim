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
	loans []Loan
	liabilities uint32
	loanLock sync.RWMutex
}

type Loan struct {
	from core.AgentId
	to core.AgentId
	initialAmount uint64
	installment uint32
	interest uint32
}

func NewBank(id uint32) *Bank {
	var b Bank
	b.id.AType = core.Bank
	b.id.Id = id
	b.debt = 0
	return &b
}

// Issues a loan and transfers funds from the bank to the target
func (b *Bank) IssueLoan(ld *core.Ledger, target core.AgentId, 
												 amount int64, interest uint32) {

	b.loanLock.Lock()
	defer b.loanLock.Unlock()

	if ld.GetBalance(b.GetId()) - amount < 0 {
		return // TODO: you may want to return an error in the future
	}

	loan := Loan {
		from: core.AgentId{ AType: core.Bank, Id: b.GetId()},
		to: target,
		initialAmount: uint64(amount),
		installment: 10,
		interest: interest,
	}

	b.loans = append(b.loans, loan)
	ld.Transfer(b.GetId(), target.Id, amount)
}

func (b *Bank) GetId() uint32 {
	return b.id.Id
}

func (b *Bank) Update(pol *core.Policies, ld *core.Ledger,
																					ticks uint64) core.UpdateReturn {
	return core.Nothing
}

func (b *Bank) GetType() core.AgentType {
	return core.Bank
}

func (b *Bank) Log() {
	fmt.Printf("Bank\n")
}
