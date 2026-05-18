package agents

import "github.com/AtesIsf/monetary-simulator/internal/core"

/*
 * bank.go
 *
 * This file defines and implements a bank as an agent for this economy.
 */

type Bank struct {
	id core.AgentId
}

func NewBank(id uint32) *Bank {
	var b Bank
	b.id.AType = core.Bank
	b.id.Id = id
	return &b
}

func (b *Bank) GetId() uint32 {
	return b.id.Id
}

func (b *Bank) Update(pol *core.Policies, ld *core.Ledger) error {
	return nil
}

func (b *Bank) GetType() core.AgentType {
	return core.Bank
}

