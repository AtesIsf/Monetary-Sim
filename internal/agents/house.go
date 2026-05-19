package agents

import (
	"fmt"
	"math/rand/v2"

	"github.com/AtesIsf/monetary-simulator/internal/core"
)

/*
 * house.go
 *
 * This file defines and implements a household as an agent for this economy.
 * For simplicity, a household is one person.
 */

type Household struct {
	id core.AgentId
	// MPC = dC/dY
	mpcY float64 // from recent income -> high
	mpcB float64 // from balance -> low
	c0 uint32 // autonomoues consumption
	employer uint32 // id of employer
}

// Total consumption: mpcY * Y + mpcB * B + c0

func NewHousehold(id uint32) *Household {
	var house Household
	house.id.AType = core.Household
	house.id.Id = id

	house.employer = 0

	// TODO: you may want to change these later idk
	house.mpcY = float64(rand.IntN(4)) / 10
	house.mpcY +=  0.6 // 0.6 <= mpcY <= 0.9

	house.mpcB = float64(rand.IntN(4)) / 100
	house.mpcB +=  0.02 // 0.02 <= mpcB <= 0.05

	house.c0 = uint32(rand.IntN(5)) + 1

	return &house
}

// If 0, then there is no employer --> Thats the bank
func (h *Household) GetEmployer() uint32 {
	return h.employer
}

func (h *Household) SetEmployer(id uint32) {
	h.employer = id
}

func (h *Household) GetId() uint32 {
	return h.id.Id
}

func (h *Household) Update(pol *core.Policies, ld *core.Ledger, tick uint64) core.UpdateReturn {
	// For now, spend all in one random firm. Diversify consumption later
	return core.Consume
}

func (h *Household) GetType() core.AgentType {
	return core.Household
}

func (h *Household) CalculateConsumption(ld *core.Ledger) int64 {
	balanceConsumption := h.mpcB * max(float64(ld.GetBalance(h.GetId())), 0)
	totalC := float64(h.c0) + balanceConsumption
	
	if h.GetEmployer() != 0 { // 0 -> unemployed
		totalC += h.mpcY * core.Wage
	}
	return int64(totalC)
}

func (h *Household) Log() {
	fmt.Printf("House <%d> -- Employer Id: %d\n",
							h.GetId(), h.GetEmployer())
}

