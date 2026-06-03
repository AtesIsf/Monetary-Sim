package engine

import (
	"math"

	"github.com/AtesIsf/monetary-simulator/internal/agents"
	"github.com/AtesIsf/monetary-simulator/internal/core"
)

type Matcher struct {
	sim *Simulation
}

func NewMatcher(sim *Simulation) *Matcher {
	return &Matcher{sim: sim}
}

func (m *Matcher) BuyGoods(buyer *agents.Household, ld *core.Ledger) {
	var tracker core.MacroTracker = m.sim
	maxAmount := buyer.CalculateConsumption(ld, &tracker)
	if maxAmount <= 0 {
		return
	}

	m.sim.agentsMutex.RLock()
	var cheapestFirm *agents.Firm
	var lowestPrice int = -1

	for _, ag := range m.sim.agents {
		if ag.GetType() == core.Firm {
			firm, ok := ag.(*agents.Firm)
			if !ok {
				continue
			}

			price := firm.GetPrice()
			if lowestPrice == -1 || price < lowestPrice {
				lowestPrice = price
				cheapestFirm = firm
			}
		}
	}
	m.sim.agentsMutex.RUnlock()

	if cheapestFirm == nil {
		return
	}

	amount := maxAmount - maxAmount % int64(lowestPrice)
	if amount <= 0 {
		return
	}

	ld.Transfer(buyer.GetId(), cheapestFirm.GetId(), amount)
	cheapestFirm.PerformSale(amount)
}

func (m *Matcher) HireWorker(firm *agents.Firm) {
	m.sim.agentsMutex.RLock()
	var cheapest *agents.Household
	var lowestWage int64 = math.MaxInt64

	for _, ag := range m.sim.agents {
		if ag.GetType() != core.Household {
			continue
		}
		house, ok := ag.(*agents.Household)
		if ok && !house.IsEmployed() {
			wage := m.sim.ld.GetWageExpectation(house.GetId())
			if wage < lowestWage {
				lowestWage = wage
				cheapest = house
			}
		}
	}
	m.sim.agentsMutex.RUnlock()

	if cheapest == nil {
		return
	}

	// Safely apply changes under write lock
	m.sim.agentsMutex.Lock()
	defer m.sim.agentsMutex.Unlock()

	// Double-check employment status under write lock to prevent race double-hiring
	if !cheapest.IsEmployed() {
		cheapest.SetEmployer(firm.GetId())
		firm.AddEmployee(cheapest.GetId())
	}
}
