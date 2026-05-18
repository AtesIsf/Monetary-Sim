package agents

import "github.com/AtesIsf/monetary-simulator/internal/core"

/*
 * firm.go
 *
 * This file defines and implements a firm as an agent for this economy.
 */

type Firm struct {
	id core.AgentId
}

func NewFirm(id uint32) *Firm {
	var f Firm
	f.id.AType = core.Firm
	f.id.Id = id
	return &f
}

func (f *Firm) GetId() uint32 {
	return f.id.Id
}

func (f *Firm) Update(pol *core.Policies, ld *core.Ledger) error {
	return nil
}

func (f *Firm) GetType() core.AgentType {
	return core.Firm
}

