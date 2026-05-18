package agents

import "github.com/AtesIsf/monetary-simulator/internal/core"

/*
 * house.go
 *
 * This file defines and implements a household as an agent for this economy.
 * For simplicity, a household is one person.
 */

type Household struct {
	id core.AgentId
}

func NewHousehold(id uint32) *Household {
	var house Household
	house.id.AType = core.Household
	house.id.Id = id
	return &house
}

func (h *Household) GetId() uint32 {
	return h.id.Id
}

func (h *Household) Update(pol *core.Policies, ld *core.Ledger) error {
	return nil
}

func (h *Household) GetType() core.AgentType {
	return core.Household
}

