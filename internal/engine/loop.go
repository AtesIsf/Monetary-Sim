package engine

import (
	"errors"
	"fmt"
	"math/rand/v2"
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
	matcher *Matcher
	priceHistory []float64
}

func (sim *Simulation) Populate(nHouses uint32, nFirms uint32) {
	var totalAgents uint32 = nFirms + nHouses + 1

	sim.tick = 0
	sim.ld.Init()
	sim.pol.Populate()
	sim.agents = make([]core.Agent, 0, totalAgents)
	sim.priceHistory = make([]float64, 0, 1200)

	// Records once every 5 ticks
	sim.rec = InitRecorder(filepath.Join(".", "data", "sim.csv"), 5)
	sim.matcher = NewMatcher(sim)

	var idCounter uint32 = 0

	sim.bank = *agents.NewBank(idCounter)
	// Giving the bank a balance for now
	sim.ld.AddToBalance(sim.bank.GetId(), 1000)
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
	/* for i := range len(sim.agents) {
		if i == 0 {
			continue
		}
		id := sim.agents[i].GetId()
		amount := sim.ld.GetBalance(id)
		sim.ld.AddToBalance(sim.bank.GetId(), amount)
	} */
}

func (sim *Simulation) Close() {
	sim.rec.Close()
}

func (sim *Simulation) UpdatePolicyRate(rate uint32) {
	sim.pol.SetInterestRate(rate)
}

func (sim *Simulation) Run(ticks uint64) {
	for range ticks {
		var wg sync.WaitGroup

		for _, ag := range sim.agents {
			wg.Go(func() {
				result := ag.Update(&sim.pol, &sim.ld, sim, sim.tick)

				// Handling update return here

				switch result.Action {
				// Firm
				case core.HireWorkers:
					if ag.GetType() == core.Firm {
						firm, _ := ag.(*agents.Firm)
						sim.matcher.HireWorkers(firm, result.Count)
						firm.PayWages(&sim.ld)
					}

				case core.FireWorkers:
					firm, _ := ag.(*agents.Firm)
					for range result.Count {
						if firm.GetNEmployees() == 0 {
							break
						}
						id := firm.PopEmployee()
						sim.agentsMutex.Lock()
						for _, a := range sim.agents {
							if a.GetType() == core.Household && a.GetId() == id {
								house, _ := a.(*agents.Household)
								// Set employer to self to indicate unemployment
								house.SetEmployer(house.GetId())
								break
							}
						}
						sim.agentsMutex.Unlock()
					}
					firm.PayWages(&sim.ld)

				// TODO: Make this logic more complex
				case core.RequestLoan:
					agentId := core.AgentId {
						Id: ag.GetId(),
						AType: ag.GetType(),
					}
					// TODO: Change these placeholder values later
					loanInterest := sim.pol.GetInterestRate() + 2
					loan := sim.bank.IssueLoan(&sim.ld, agentId, 200, loanInterest)
					if loan != nil {
						if ag.GetType() == core.Firm {
							f, _ := ag.(*agents.Firm)
							f.AddLoan(loan)
							f.PayWages(&sim.ld)

						} else if ag.GetType() == core.Household {
							h, _ := ag.(*agents.Household)
							h.AddLoan(loan)
						}
					}

				// Household
				case core.Consume:
					house, _ := ag.(*agents.Household)
					sim.matcher.BuyGoods(house, &sim.ld)

				// TODO: this pushes back meaningful activity by a tick!
				// May want to recurse in the future
				case core.DrawSavings:
					if ag.GetType() == core.Household {
						h, _ := ag.(*agents.Household)
						sim.bank.WithdrawAll(h.GetId(), &sim.ld)
						h.SetSavings(0)
					} else if ag.GetType() == core.Firm {
						f, _ := ag.(*agents.Firm)
						sim.bank.WithdrawAll(f.GetId(), &sim.ld)
						f.SetSavings(0)
						f.PayWages(&sim.ld)
					}

				default: // result.Action == core.Nothing, so do nothing
				}

				// Annualize loans -> not really realistic
				if sim.tick % 12 != 0 {
					return	
				}

				if ag.GetType() == core.Firm {
					f, _ := ag.(*agents.Firm)
					f.RepayLoans(&sim.bank, &sim.ld)
					f.DepositExtra(&sim.bank, &sim.ld)
				} else if ag.GetType() == core.Household {
					h, _ := ag.(*agents.Household)
					h.RepayLoans(&sim.bank, &sim.ld)
					h.DepositExtra(&sim.bank, &sim.ld)
				}

				// For debug purposes
				/* if ag.GetType() == core.Firm {
					ag.Log()
				} */
			})
		}
		wg.Wait()

		// sync work -> helps sync the bank balances with agents
		for _, ag := range sim.agents {
			wg.Go(func() {
				if ag.GetType() == core.Bank {
					return
				}	

				deposits := sim.bank.QueryDeposits(ag.GetId())
				if (ag.GetType() == core.Household) {
					h, _ := ag.(*agents.Household)
					h.SetSavings(deposits)
				} else if (ag.GetType() == core.Firm) {
					f, _ := ag.(*agents.Firm)
					f.SetSavings(deposits)
				}
			})
		} 
		wg.Wait()

		// Pricelevel tracking
		var totalPrice int = 0
		var firmCount int = 0
		for _, ag := range sim.agents {
			if ag.GetType() != core.Firm {
				continue
			}
			f, _ := ag.(*agents.Firm)
			totalPrice += f.GetPrice()
			firmCount++
		}
		avgPrice := 1.0
		if firmCount > 0 {
			avgPrice = float64(totalPrice) / float64(firmCount)
		}
		sim.priceHistory = append(sim.priceHistory, avgPrice)

		sim.tick += 1
		if sim.tick % sim.rec.GetFrequency() == 0 {
			sim.rec.Write(sim.agents, sim.tick, &sim.ld)
		}
	}

	// sim.ld.PrintBalances()
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

func (sim *Simulation) GetUnemploymentRate() float64 {
	sim.agentsMutex.RLock()
	defer sim.agentsMutex.RUnlock()

	var totalHouses, unemployed float64
	for _, ag := range sim.agents {
		if ag.GetType() != core.Household {
			continue
		}
		totalHouses++
		house, _ := ag.(*agents.Household)
		if !house.IsEmployed() {
			unemployed++
		}
	}
	if totalHouses == 0 {
		return 0
	}
	return unemployed / totalHouses
}

func (sim *Simulation) GetInflationRate() float64 {
	length := len(sim.priceHistory)
	if length < 12 {
		return 1.0
	}
	currentPrice := sim.priceHistory[length - 1]
	prevPrice := sim.priceHistory[length - 12]
	if prevPrice == 0 {
		return 1.0
	}
	return currentPrice / prevPrice
}

