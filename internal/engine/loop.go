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
	bank agents.Bank
	tick uint64
}

func (sim *Simulation) Populate(nHouses uint32, nFirms uint32) {
	var totalAgents uint32 = nFirms + nHouses + 1

	sim.tick = 0
	sim.ld.Populate(totalAgents)
	sim.pol.Populate()
	sim.agents = make([]core.Agent, 0, totalAgents)

	var idCounter uint32 = 0
	sim.bank = *agents.NewBank(idCounter)
	idCounter += 1
	for range nHouses {
		sim.agents = append(sim.agents, agents.NewHousehold(idCounter))
		idCounter += 1
	}
	for range nFirms {
		sim.agents = append(sim.agents, agents.NewFirm(idCounter))
		idCounter += 1
	}
	sim.agents = append(sim.agents, &sim.bank)

	// Add all agents' balances as loanable funds
	for i := range len(sim.agents) - 1 {
		id := sim.agents[i].GetId()
		amount := sim.ld.GetBalance(id)
		sim.ld.AddToBalance(sim.bank.GetId(), amount)
	}
}

func (sim *Simulation) Run(ticks uint32) {
	for i := range ticks {
		var wg sync.WaitGroup

		fmt.Printf("\n---- Step %d ----\n", i)
		for _, ag := range sim.agents {
			wg.Go(func() {
				if ag.GetType() == core.Firm {
					val, err := sim.GetRandom(core.Household)
					if err == nil {
						sim.ld.Transfer(ag.GetId(), val.Id, 10)
					}
				}
				ag.Update(&sim.pol, &sim.ld)
				// For debug purposes
				fmt.Printf("Agent %d, Type %d, Bal: %d\n", 
								ag.GetId(), ag.GetType(), sim.ld.GetBalance(ag.GetId()))
			})
		}
		wg.Wait()
		sim.tick += 1
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

