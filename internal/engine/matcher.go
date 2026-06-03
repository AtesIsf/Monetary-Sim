package engine

import (
	"github.com/AtesIsf/monetary-simulator/internal/agents"
	"github.com/AtesIsf/monetary-simulator/internal/core"
)

/*
 * matcher.go
 *
 * This file implements the Matcher struct, which facilitates economic
 * transactions and matching between agents, specifically handling
 * households buying goods from firms and firms hiring workers.
 */

type Matcher struct {
	sim *Simulation
}

func NewMatcher(sim *Simulation) *Matcher {
	return &Matcher{ sim: sim }
}

func (m *Matcher) BuyGoods(buyer *agents.Household, ld *core.Ledger) {
	maxAmount := buyer.CalculateConsumption(ld)
	if maxAmount <= 0 {
		return
	}

	// Pick random firm using existing logic
	randFirm, err := m.sim.GetRandom(core.Firm)
	if err != nil {
		return
	}

	selected, err := m.sim.AgentWhere(randFirm.Id)
	if err != nil {
		return
	}
	firmAgent, _ := (*selected).(*agents.Firm)

	price := firmAgent.GetPrice()
	amount := maxAmount - maxAmount % int64(price)
	if amount <= 0 {
		return
	}

	ld.Transfer(buyer.GetId(), randFirm.Id, amount)
	firmAgent.PerformSale(amount)
}

func (m *Matcher) HireWorker(firm *agents.Firm) {
	m.sim.agentsMutex.Lock()
	defer m.sim.agentsMutex.Unlock()

	for _, ag := range m.sim.agents {
		if ag.GetType() != core.Household {
			continue
		}

		house, _ := ag.(*agents.Household)
		if !house.IsEmployed() {
			house.SetEmployer(firm.GetId())
			firm.AddEmployee(ag.GetId())
			break
		}
	}
}
