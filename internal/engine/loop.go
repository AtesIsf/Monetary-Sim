package engine

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
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
	rec Recorder
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

	// Records once every 5 ticks
	sim.rec = InitRecorder(filepath.Join(".", "data", "sim.csv"), 5)

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

	// Shuffle agents except the bank at index 0
	if len(sim.agents) > 2 {
		rand.Shuffle(len(sim.agents) - 1, func(i int, j int) {
			temp := sim.agents[i + 1]
			sim.agents[i + 1] = sim.agents[j + 1]
			sim.agents[j + 1] = temp
		})
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

func (sim *Simulation) Close() {
	sim.rec.Close()
}

func (sim *Simulation) Run(ticks uint64) {
	for range ticks {
		var wg sync.WaitGroup

		for _, ag := range sim.agents {
			wg.Go(func() {
				result := ag.Update(&sim.pol, &sim.ld, sim.tick)

				// Handling update return here

				switch result {
				// Firm
				case core.HireWorkers:
					if ag.GetType() == core.Firm {
						firm, _ := ag.(*agents.Firm)
						sim.HireWorker(firm)
					}

				case core.FireWorkers:
					firm, _ := ag.(*agents.Firm)
					id := firm.PopEmployee()
					sim.agentsMutex.Lock()
					for _, a := range sim.agents {
						if a.GetType() == core.Household && a.GetId() == id {
							house, _ := a.(*agents.Household)
							house.SetEmployer(0) // 0 is the bank, so no employer
							break
						}
					}
					sim.agentsMutex.Unlock()

				case core.RequestLoan:

				// Household
				case core.Consume:
					house, _ := ag.(*agents.Household)
					amount := house.CalculateConsumption(&sim.ld)
					// Diversify consumption later
					randFirm, err := sim.GetRandom(core.Firm)
					if err == nil {
						sim.ld.Transfer(house.GetId(), randFirm.Id, amount)
						selected, err := sim.AgentWhere(randFirm.Id)
						if err != nil {
							fmt.Println("Fatal Error! No firms found.")
							os.Exit(1)
						}
						firmAgent, _ := (*selected).(*agents.Firm)
						firmAgent.PerformSale(amount)

						// TODO: Now, if the balance is less than 0, request a loan

					} // else, no firm was valid so no consumption took place

				default: // result == core.Nothing, so do nothing
				}

				// For debug purposes
				/*if sim.tick == ticks - 1 {
					ag.Log()
				}*/
			})
		}
		wg.Wait()
		sim.tick += 1
		if sim.tick % sim.rec.GetFrequency() == 0 {
			sim.rec.Write(sim.agents, sim.tick, &sim.ld)
		}
	}

	// sim.ld.PrintBalances()
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
		if !house.IsEmployed() {
			house.SetEmployer(f.GetId())
			f.AddEmployee(ag.GetId())
			break
		}
	}
}

// This function is to be used in phase 1, where exchanges are random
// It fails if there is no agents of the specified type
func (sim *Simulation) GetRandom(target core.AgentType) (core.AgentId, error) {
	sim.agentsMutex.RLock()
	defer sim.agentsMutex.RUnlock()

	if target == core.Firm {
		totalEmployees := 0
		var eligibleFirms []*agents.Firm
		for _, ag := range sim.agents {
			if ag.GetType() == core.Firm {
				firm, ok := ag.(*agents.Firm)
				if ok && firm.GetNEmployees() > 0 {
					totalEmployees += firm.GetNEmployees()
					eligibleFirms = append(eligibleFirms, firm)
				}
			}
		}

		if totalEmployees == 0 {
			return core.AgentId{}, errors.New("No valid firm with employees exists!")
		}

		r := rand.IntN(totalEmployees)
		cumulative := 0
		for _, firm := range eligibleFirms {
			cumulative += firm.GetNEmployees()
			if r < cumulative {
				return core.AgentId{AType: core.Firm, Id: firm.GetId()}, nil
			}
		}
	}

	// If the target is a Household or Bank
	length := len(sim.agents)
	if length == 0 {
		return core.AgentId{}, errors.New("No valid agent exists!")
	}

	index := rand.IntN(length)

	for increment := range sim.agents {
		selected := sim.agents[(index + increment)%length]
		if selected.GetType() == target {
			return core.AgentId{AType: target, Id: selected.GetId()}, nil
		}
	}

	fmt.Printf("GetRandom failed for target %d\n", target)
	return core.AgentId{}, errors.New("No valid agent exists!")
}
// Returns the agent where the id equals the parameter
func (sim *Simulation) AgentWhere(id uint32) (*core.Agent, error) {
	for _, ag := range sim.agents {
		if ag.GetId() == id {
			return &ag, nil
		}
	}
	return nil, errors.New("No matching agent found.")
}

