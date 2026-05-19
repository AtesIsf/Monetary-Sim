package engine

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"sync"

	"github.com/AtesIsf/monetary-simulator/internal/agents"
	"github.com/AtesIsf/monetary-simulator/internal/core"
)

/*
 * loop.go
 *
 * This file implements the simulation's main loop. All economic activities
 * occur in discrete units of time called ticks.
 * The engine's singleton struct is defined in this file too.
 */

type Simulation struct {
	ld core.Ledger
	pol core.Policies
	agents []core.Agent
	agentsMutex sync.RWMutex // used when modifying the list
	bank agents.Bank
	tick uint64
}

func (sim *Simulation) Populate(nHouses uint32, nFirms uint32) {
	var totalAgents uint32 = nFirms + nHouses + 1

	sim.tick = 0
	sim.ld.Init()
	sim.pol.Populate()
	sim.agents = make([]core.Agent, 0, totalAgents)

	var idCounter uint32 = 0

	sim.bank = *agents.NewBank(idCounter)
	sim.ld.AddToBalance(sim.bank.GetId(), 0)
	sim.agents = append(sim.agents, &sim.bank)

	idCounter += 1
	for range nHouses {
		sim.agents = append(sim.agents, agents.NewHousehold(idCounter))
		sim.ld.AddToBalance(idCounter, 100)
		idCounter += 1
	}
	for range nFirms {
		sim.agents = append(sim.agents, agents.NewFirm(idCounter))
		sim.ld.AddToBalance(idCounter, 500)
		idCounter += 1
	}


	// Add all agents' balances as loanable funds
	for i := range len(sim.agents) {
		if i == 0 {
			continue
		}
		id := sim.agents[i].GetId()
		amount := sim.ld.GetBalance(id)
		sim.ld.AddToBalance(sim.bank.GetId(), amount)
	}
}

func (sim *Simulation) Run(ticks uint32) {
	for i := range ticks {
		var wg sync.WaitGroup

		fmt.Printf("\n---- Step %d ----\n", i + 1)
		for _, ag := range sim.agents {
			wg.Go(func() {
				result := ag.Update(&sim.pol, &sim.ld)

					
				switch result {
				case core.HireWorkers:
					if ag.GetType() == core.Firm {
						firm, _ := ag.(*agents.Firm)
						sim.HireWorker(firm)
					}
				case core.FireWorkers:

				case core.RequestLoan:

				default: // result == core.Nothing, so do nothing
				}

				// For debug purposes
				ag.Log()
				fmt.Printf("\t>> <%d> Balance: %d\n", 
										ag.GetId(), sim.ld.GetBalance(ag.GetId()))
			})
		}
		wg.Wait()
		sim.tick += 1
	}
}

// TODO: You may want to parameterize number of workers to hire
func (sim *Simulation) HireWorker(f *agents.Firm) {
	sim.agentsMutex.Lock()
	defer sim.agentsMutex.Unlock()

	for _, ag := range sim.agents {
		if ag.GetType() != core.Household {
			continue
		}

		house, _ := ag.(*agents.Household)
		if house.GetEmployer() == 0 {
			house.SetEmployer(f.GetId())
			f.AddEmployee(ag.GetId())
			break
		}
	}
}

// This function is to be used in phase 1, where exchanges are random
// It fails if there is no agents of the specified type
func (sim *Simulation) GetRandom(target core.AgentType) (core.AgentId, error) {
	length := len(sim.agents)
	index := rand.IntN(length)

	for increment := range sim.agents {
		selected := &sim.agents[(index + increment) % length]
		if (*selected).GetType() == target {
			return core.AgentId{ AType: target, Id: (*selected).GetId() }, nil
		}
	}
	
	fmt.Printf("GetRandom failed for target %d\n", target)
	return core.AgentId{}, errors.New("No valid agent exists!")
}

