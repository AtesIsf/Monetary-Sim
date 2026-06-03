package engine

import (
	"math/rand/v2"

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
	maxAmount := buyer.CalculateConsumption(ld)
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
	var unemployed []*agents.Household

	for _, ag := range m.sim.agents {
		if ag.GetType() == core.Household {
			house, ok := ag.(*agents.Household)
			if ok && !house.IsEmployed() {
				unemployed = append(unemployed, house)
			}
		}
	}
	m.sim.agentsMutex.RUnlock()

	if len(unemployed) == 0 {
		return
	}

	// Pick a random unemployed household
	randomIndex := rand.IntN(len(unemployed))
	candidate := unemployed[randomIndex]

	// Safely apply changes under write lock
	m.sim.agentsMutex.Lock()
	defer m.sim.agentsMutex.Unlock()

	// Double-check employment status under write lock to prevent double-hiring
	if !candidate.IsEmployed() {
		candidate.SetEmployer(firm.GetId())
		firm.AddEmployee(candidate.GetId())
	}
}
