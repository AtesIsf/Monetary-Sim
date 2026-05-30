package agents

import (
	"fmt"
	"math/rand/v2"
	"sync"

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

	// Stored here, not in the bank for convenience
	bankBalance uint32
	loans []*Loan
	loanLock sync.RWMutex
}

// Total consumption: mpcY * Y + mpcB * B + c0

func NewHousehold(id uint32) *Household {
	var house Household
	house.id.AType = core.Household
	house.id.Id = id
	house.employer = id

	// TODO: you may want to change these later idk
	house.mpcY = float64(rand.IntN(4)) / 10
	house.mpcY +=  0.6 // 0.6 <= mpcY <= 0.9

	house.mpcB = float64(rand.IntN(4)) / 100
	house.mpcB +=  0.02 // 0.02 <= mpcB <= 0.05

	house.c0 = uint32(rand.IntN(5)) + 1

	house.bankBalance = 0

	return &house
}

// If self.id = employer.id, then there is no employer
func (f *Household) GetEmployer() uint32 {
	return f.employer
}

func (f *Household) SetEmployer(id uint32) {
	f.employer = id
}

func (f *Household) GetId() uint32 {
	return f.id.Id
}

func (f *Household) Update(pol *core.Policies, ld *core.Ledger,
																							 tick uint64) core.UpdateReturn {
	consumption := f.CalculateConsumption(ld)
	exitStatus := core.Consume
	if ld.GetBalance(f.GetId()) - consumption < 0 {
		exitStatus = core.RequestLoan
	}
	return exitStatus
}

func (f *Household) GetType() core.AgentType {
	return core.Household
}

func (f *Household) CalculateConsumption(ld *core.Ledger) int64 {
	balanceConsumption := f.mpcB * max(float64(ld.GetBalance(f.GetId())), 0)
	totalC := float64(f.c0) + balanceConsumption
	
	if f.IsEmployed() {
		totalC += f.mpcY * core.Wage
	}
	return int64(totalC)
}

// I'm assuming that this both mutates the bank and this agent's loans
func (h *Household) RepayLoans(bank *Bank, ld *core.Ledger) {
	h.loanLock.Lock()
	defer h.loanLock.Unlock()
	var toBeDeleted []int

	for index, loan := range h.loans {
		if loan.remainingAmount == 0 {
			toBeDeleted = append(toBeDeleted, index)	
		}

		toBePaid := loan.remainingAmount - 
								loan.initialAmount / uint64(loan.installment)
		// Pay with demand deposits
		if toBePaid < uint64(h.bankBalance) {
			h.bankBalance -= uint32(toBePaid)				
			bank.DecreaseDemandDeposit(h.GetId(), int64(toBePaid))
			loan.remainingAmount -= toBePaid

		// Pay with available cash
		} else if toBePaid < uint64(ld.GetBalance(h.GetId())) {
			ld.AddToBalance(h.GetId(), -int64(toBePaid))
			loan.remainingAmount -= toBePaid

		// Combine both
		} else if toBePaid < uint64(h.bankBalance) +
												 uint64(ld.GetBalance(h.GetId())) {
			difference := uint64(h.bankBalance) +
										uint64(ld.GetBalance(h.GetId())) - toBePaid
			bank.DecreaseDemandDeposit(h.GetId(), int64(h.bankBalance))
			ld.AddToBalance(h.GetId(), -(int64(difference) - 
																	 int64(h.bankBalance)))
			loan.remainingAmount -= toBePaid

		// Cannot pay debt -> default
		} else {
			// TODO: Maybe take out another loan?
			loan.remainingAmount = 0
		}
	}

	for index := range toBeDeleted {
		h.loans = append(h.loans[:index], h.loans[index + 1:]...)
	}
}

func (h *Household) DepositExtra(bank *Bank, ld *core.Ledger) {
	if ld.GetBalance(h.GetId()) <= core.MaxLiquidity {
		return
	}
	difference := ld.GetBalance(h.GetId()) - core.MaxLiquidity
	bank.AddDemandDeposit(h.GetId(), difference, ld)
	h.bankBalance += uint32(difference)
}

func (f *Household) RecieveLoan(loan *Loan) {
	f.loanLock.Lock()
	defer f.loanLock.Unlock()

	f.loans = append(f.loans, loan)
}

func (f *Household) IsEmployed() bool {
	return f.GetId() != f.GetEmployer()
}

func (f *Household) Log() {
	fmt.Printf("House <%d> -- Employer Id: %d\n",
							f.GetId(), f.GetEmployer())
}

