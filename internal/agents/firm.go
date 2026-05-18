package agents

import (
	"math/rand/v2"

	"github.com/AtesIsf/monetary-simulator/internal/core"
)

/*
 * firm.go
 *
 * This file defines and implements a firm as an agent for this economy.
 */

type Firm struct {
	id core.AgentId
	invTarget uint32 // targeted inventory, measured in monetary units
	invCurr uint32 // current sold inventory
}

func NewFirm(id uint32) *Firm {
	var f Firm
	f.id.AType = core.Firm
	f.id.Id = id

	// TODO: You may want to change how the target is set in the future
	f.invTarget = uint32(rand.IntN(100)) + 500 // 500 <= target <= 600
	f.invCurr = 0

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

