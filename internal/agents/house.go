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
	employerMutex sync.RWMutex

	// Stored here, not in the bank for convenience
	bankBalance int64
	loans []*Loan
	loanLock sync.RWMutex
	wageExpectation int64
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
	house.wageExpectation = int64(core.StandardWage)

	return &house
}

func (h *Household) AddLoan(loan *Loan) {
	if loan == nil {
		return
	}
	h.loans = append(h.loans, loan)
}

// If self.id = employer.id, then there is no employer
func (h *Household) GetEmployer() uint32 {
	h.employerMutex.RLock()
	defer h.employerMutex.RUnlock()
	return h.employer
}

func (h *Household) SetEmployer(id uint32) {
	h.employerMutex.Lock()
	defer h.employerMutex.Unlock()
	h.employer = id
}

func (h *Household) GetId() uint32 {
	return h.id.Id
}

func (h *Household) Update(pol *core.Policies, ld *core.Ledger,
								macro core.MacroTracker, tick uint64) core.UpdateReturn {

	// 1. Unemployment-driven adjustment (every tick)
	if h.IsEmployed()  && macro.GetUnemploymentRate() < 0.05 {
			h.wageExpectation++
	} else if !h.IsEmployed() {
		h.wageExpectation = max(core.MinWage, h.wageExpectation - 1)
	}

	// 2. Price-driven adjustment (every 12 ticks)
	if tick % 12 == 0 && tick > 0 {
		rate := macro.GetInflationRate()
		scaled := float64(h.wageExpectation) * rate
		h.wageExpectation = max(core.MinWage, int64(scaled))
	}

	// 3. Register expectation on the ledger
	ld.SetWageExpectation(h.GetId(), h.wageExpectation)

	consumption := h.CalculateConsumption(ld, &macro)
	balance := ld.GetBalance(h.GetId())

	if balance >= consumption {
		return core.Consume
	}

	if balance + int64(h.bankBalance) >= consumption {
		return core.DrawSavings
	}
	// the case where balance + savings don't add up to the consumption
	return core.RequestLoan
}

func (h *Household) GetType() core.AgentType {
	return core.Household
}

func (h *Household) CalculateConsumption(ld *core.Ledger, 
																				macro *core.MacroTracker) int64 {
	h.c0 = uint32(float64(h.c0) * (*macro).GetInflationRate())
	balanceConsumption := h.mpcB * max(float64(ld.GetBalance(h.GetId())), 0)
	totalC := float64(h.c0) + balanceConsumption

	if h.IsEmployed() {
		totalC += h.mpcY * float64(core.StandardWage)
	}
	return int64(totalC)
}

// I'm assuming that this both mutates the bank and this agent's loans
func (h *Household) RepayLoans(bank *Bank, ld *core.Ledger) {
	h.loanLock.Lock()
	defer h.loanLock.Unlock()

	for _, loan := range h.loans {
		if loan == nil || loan.remainingAmount == 0 {
			continue
		}

		installment := loan.initialAmount / uint64(loan.installment)
		toBePaid := installment
		toBePaid = min(toBePaid, loan.remainingAmount)

		// Pay with demand deposits
		if toBePaid <= uint64(h.bankBalance) {
			h.bankBalance -= int64(toBePaid)
			bank.DecreaseDemandDeposit(h.GetId(), int64(toBePaid))
			loan.remainingAmount -= toBePaid

		// Pay with available cash
		} else if toBePaid <= uint64(max(0, ld.GetBalance(h.GetId()))) {
			ld.Transfer(h.GetId(), bank.GetId(), int64(toBePaid))
			loan.remainingAmount -= toBePaid

		// Combine both
		} else if toBePaid <= uint64(h.bankBalance) +
													uint64(max(0, ld.GetBalance(h.GetId()))) {
			cashNeeded := toBePaid - uint64(h.bankBalance)
			bank.DecreaseDemandDeposit(h.GetId(), int64(h.bankBalance))
			h.bankBalance = 0
			ld.Transfer(h.GetId(), bank.GetId(), int64(cashNeeded))
			loan.remainingAmount -= toBePaid

		// Cannot pay debt -> default
		} else {
			// TODO: Maybe take out another loan?
			loan.remainingAmount = 0
		}
	}

	// Filter out finished loans
	newLoans := make([]*Loan, 0, len(h.loans))
	for _, loan := range h.loans {
		if loan.remainingAmount > 0 {
			newLoans = append(newLoans, loan)
		}
	}
	h.loans = newLoans
}

func (h *Household) DepositExtra(bank *Bank, ld *core.Ledger) {
	if ld.GetBalance(h.GetId()) <= core.MaxLiquidity {
		return
	}
	difference := ld.GetBalance(h.GetId()) - core.MaxLiquidity
	bank.AddDemandDeposit(h.GetId(), difference, ld)
	h.bankBalance += difference
}

func (h *Household) SetSavings(amount int64) {
	h.bankBalance = amount
}

func (h *Household) ReceiveLoan(loan *Loan) {
	h.loanLock.Lock()
	defer h.loanLock.Unlock()

	h.loans = append(h.loans, loan)
}

func (h *Household) IsEmployed() bool {
	return h.GetId() != h.GetEmployer()
}

func (h *Household) Log() {
	fmt.Printf("House <%d> -- Employer Id: %d\n",
							h.GetId(), h.GetEmployer())
}

