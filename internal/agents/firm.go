package agents

import (
	"fmt"
	"math/rand/v2"
	"slices"
	"sync"

	"github.com/AtesIsf/monetary-simulator/internal/core"
)

/*
 * firm.go
 *
 * This file defines and implements a firm as an agent for this economy.
 */

type Firm struct {
	id core.AgentId
	invTarget uint32 // targeted inventory, measured in monetary units
	invCurr uint32 // current sold inventory
	employees []uint32 // ids of employees
	stockPrice int

	// Stored here, not in the bank for convenience
	bankBalance int64
	loans []*Loan
	loanLock sync.RWMutex
}

func NewFirm(id uint32) *Firm {
	var f Firm
	f.id.AType = core.Firm
	f.id.Id = id

	// TODO: You may want to change how the target is set in the future
	f.invTarget = uint32(rand.IntN(100)) + 500 // 500 <= target <= 600
	f.invCurr = 0

	f.stockPrice = core.Price

	return &f
}

func (f *Firm) ReceiveLoan(loan *Loan) {
	f.loanLock.Lock()
	defer f.loanLock.Unlock()

	f.loans = append(f.loans, loan)
}

// I'm assuming that this both mutates the bank and this agent's loans
func (f *Firm) RepayLoans(bank *Bank, ld *core.Ledger) {
	f.loanLock.Lock()
	defer f.loanLock.Unlock()

	for _, loan := range f.loans {
		if loan == nil || loan.remainingAmount == 0 {
			continue
		}

		installment := loan.initialAmount / uint64(loan.installment)
		toBePaid := installment
		toBePaid = min(toBePaid, loan.remainingAmount)

		// Pay with demand deposits
		if toBePaid <= uint64(f.bankBalance) {
			f.bankBalance -= int64(toBePaid)
			bank.DecreaseDemandDeposit(f.GetId(), int64(toBePaid))
			loan.remainingAmount -= toBePaid

		// Pay with available cash
		} else if toBePaid <= uint64(max(0, ld.GetBalance(f.GetId()))) {
			ld.Transfer(f.GetId(), bank.GetId(), int64(toBePaid))
			loan.remainingAmount -= toBePaid

		// Combine both
		} else if toBePaid <= uint64(f.bankBalance) + 
													uint64(max(0, ld.GetBalance(f.GetId()))) {
			cashNeeded := toBePaid - uint64(f.bankBalance)
			bank.DecreaseDemandDeposit(f.GetId(), int64(f.bankBalance))
			f.bankBalance = 0
			ld.Transfer(f.GetId(), bank.GetId(), int64(cashNeeded))
			loan.remainingAmount -= toBePaid

		// Cannot pay debt -> default
		} else {
			// TODO: Maybe take out another loan?
			loan.remainingAmount = 0
		}
	}

	// Filter out finished loans
	newLoans := make([]*Loan, 0, len(f.loans))
	for _, loan := range f.loans {
		if loan.remainingAmount > 0 {
			newLoans = append(newLoans, loan)
		}
	}
	f.loans = newLoans
}

func (f *Firm) AddEmployee(id uint32) {
	f.employees = append(f.employees, id)
}

// Removes a random employee and returns its id
func (f *Firm) PopEmployee() uint32 {
	randomIndex := rand.IntN(len(f.employees))
	id := f.employees[randomIndex]
	f.employees = slices.Delete(f.employees, randomIndex, randomIndex + 1)
	return id
}

func (f *Firm) GetNEmployees() int {
	return len(f.employees)
}

// Call this after a household consumes to update invCur
func (f *Firm) PerformSale(amount int64) {
	f.invCurr += uint32(amount)
}

func (f *Firm) GetId() uint32 {
	return f.id.Id
}

// delta -> target - curr -> helps us modify the price
func (f *Firm) adaptPrice(delta int64) {
	var value int

	if delta > 0 { // below target -> decrease prices
		value = -(rand.IntN(3) + 1)
	} else if delta < 0 { // above target -> increase prices
		value = rand.IntN(3) + 1

	} // else -> value holds its zero value

	f.stockPrice += value
	f.stockPrice = max(core.MinPrice, f.stockPrice)
}

func (f *Firm) Update(pol *core.Policies, ld *core.Ledger,
																				  tick uint64) core.UpdateReturn {
	var returnVal core.UpdateReturn = core.Nothing
	if tick % core.TicksPerYear == 0 {
		f.invCurr = 0 // reset tracked value since it is a new year
	}

	if f.invCurr < f.invTarget { // Underproducing, so hire
		returnVal = core.HireWorkers
	} else if f.invCurr > f.invTarget && len(f.employees) > 0 { // Overproducing
		returnVal = core.FireWorkers
	}

	f.adaptPrice(int64(f.invTarget) - int64(f.invCurr))

	// TODO: adaptive wages later
	totalSpending := len(f.employees) * core.Wage

	if totalSpending >= 0 {
		return returnVal
	}

	// savings are enough to cover the deficit
	if totalSpending + int(f.bankBalance) >= 0 {
		returnVal = core.DrawSavings
	} else { // take out a loan
		returnVal = core.RequestLoan
	}

	return returnVal
}

func (f *Firm) AddLoan(loan *Loan) {
	if loan == nil {
		return
	}
	f.loans = append(f.loans, loan)
}

func (f *Firm) DepositExtra(bank *Bank, ld *core.Ledger) {
	if ld.GetBalance(f.GetId()) <= core.MaxLiquidity {
		return
	}
	difference := ld.GetBalance(f.GetId()) - core.MaxLiquidity
	bank.AddDemandDeposit(f.GetId(), difference, ld)
	f.bankBalance += difference
}

func (f *Firm) SetSavings(amount int64) {
	f.bankBalance = amount
}

func (f *Firm) GetPrice() int {
	return f.stockPrice
}

func (f *Firm) GetType() core.AgentType {
	return core.Firm
}

func (f *Firm) Log() {
	fmt.Printf("Firm <%d> -- Target: %d -- Actual: %d -- Price: %d -- %d Employees\n", 
							f.GetId(), f.invTarget, f.invCurr, f.stockPrice, len(f.employees))
}

