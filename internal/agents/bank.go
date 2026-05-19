package agents

import (
	"fmt"

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
}

type Loan struct {
	from core.AgentId
	to core.AgentId
	initialAmount uint32
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

func (b *Bank) GetId() uint32 {
	return b.id.Id
}

func (b *Bank) Update(pol *core.Policies, ld *core.Ledger, ticks uint64) core.UpdateReturn {
	return core.Nothing
}

func (b *Bank) GetType() core.AgentType {
	return core.Bank
}

func (b *Bank) Log() {
	fmt.Printf("Bank\n")
}

