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
	invTarget uint32 // targeted inventory, measured in units of goods
	invCurr uint32 // current sold inventory, in units
	inventory uint32 // current unsold stock
	prevInventory uint32 // snapshot of unsold stock before production
	employees []uint32 // ids of employees
	stockPrice int

	// Stored here, not in the bank for convenience
	bankBalance int64
	loans []*Loan
	loanLock sync.RWMutex
	firmStateMutex sync.Mutex // protects firm's state
}

func NewFirm(id uint32) *Firm {
	var f Firm
	f.id.AType = core.Firm
	f.id.Id = id

	f.invTarget = uint32(rand.IntN(6)) + 25 // quantity target: 25 <= target <= 30
	f.invCurr = 0
	f.inventory = 0

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
	f.firmStateMutex.Lock()
	defer f.firmStateMutex.Unlock()
	f.employees = append(f.employees, id)
}

// Removes a random employee and returns its id
func (f *Firm) PopEmployee() uint32 {
	f.firmStateMutex.Lock()
	defer f.firmStateMutex.Unlock()
	randomIndex := rand.IntN(len(f.employees))
	id := f.employees[randomIndex]
	f.employees = slices.Delete(f.employees, randomIndex, randomIndex + 1)
	return id
}

func (f *Firm) GetNEmployees() int {
	f.firmStateMutex.Lock()
	defer f.firmStateMutex.Unlock()
	return len(f.employees)
}

func (f *Firm) GetInventory() uint32 {
	f.firmStateMutex.Lock()
	defer f.firmStateMutex.Unlock()
	return f.inventory
}

// Call this after a household consumes to update invCur and inventory
func (f *Firm) PerformSale(units uint32) {
	f.firmStateMutex.Lock()
	defer f.firmStateMutex.Unlock()
	if f.inventory >= units {
		f.inventory -= units
	} else {
		f.inventory = 0
	}
	f.invCurr += units
}

func (f *Firm) GetId() uint32 {
	return f.id.Id
}

// delta -> target - curr -> helps us modify the price
func (f *Firm) adaptPrice(delta int64) {
	f.firmStateMutex.Lock()
	defer f.firmStateMutex.Unlock()
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
										_ core.MacroTracker, tick uint64) core.UpdateReturn {
	var action core.ActionType = core.Nothing
	var count uint32 = 0

	f.firmStateMutex.Lock()
	f.prevInventory = f.inventory
	if tick % core.TicksPerYear == 0 {
		f.invCurr = 0 // reset tracked value since it is a new year
	}

	// 1 unit produced per employee per tick
	f.inventory += uint32(len(f.employees))

	if f.invCurr < f.invTarget && f.prevInventory == 0 { // Underproducing and no carry-over inventory, so hire
		action = core.HireWorkers
		delta := int64(f.invTarget) - int64(f.invCurr)
		ticksRemaining := 12 - (tick % 12)
		if ticksRemaining > 0 {
			count = uint32(delta / int64(ticksRemaining))
		}
		if count == 0 {
			count = 1
		}
	} else if (f.invCurr > f.invTarget || (f.invCurr < f.invTarget && f.prevInventory > 0)) && len(f.employees) > 0 { // Overproducing or weak demand (carry-over inventory)
		action = core.FireWorkers
		var delta int64
		if f.invCurr > f.invTarget {
			delta = int64(f.invCurr) - int64(f.invTarget)
		} else {
			delta = int64(f.invTarget) - int64(f.invCurr)
		}
		ticksRemaining := 12 - (tick % 12)
		if ticksRemaining > 0 {
			count = uint32(delta / int64(ticksRemaining))
		}
		if count == 0 {
			count = 1
		}
		if count > uint32(len(f.employees)) {
			count = uint32(len(f.employees))
		}
	}

	deltaAdapt := int64(f.invTarget) - int64(f.invCurr)
	employeesCopy := make([]uint32, len(f.employees))
	copy(employeesCopy, f.employees)
	f.firmStateMutex.Unlock()

	f.adaptPrice(deltaAdapt)

	// Sum actual employee wage expectations
	var totalSpending int64 = 0
	for _, empId := range employeesCopy {
		wage := ld.GetWageExpectation(empId)
		if wage <= 0 {
			wage = core.MinWage
		}
		totalSpending += wage
	}

	// Balance is enough to pay
	balance := ld.GetBalance(f.GetId())
	var finance core.ActionType = core.Nothing

	if balance < totalSpending {
		if balance + f.bankBalance >= totalSpending {
			finance = core.DrawSavings
		} else if pol.GetInterestRate() <= core.MaxBorrowRate { // take out a loan
			finance = core.RequestLoan
		}
	}

	return core.UpdateReturn{
		Action:  action,
		Count:   count,
		Finance: finance,
	}
}

func (f *Firm) AddLoan(loan *Loan) {
	if loan == nil {
		return
	}
	f.loanLock.Lock()
	defer f.loanLock.Unlock()
	f.loans = append(f.loans, loan)
}

func (f *Firm) DepositExtra(bank *Bank, ld *core.Ledger, 
														polRate uint32) {
	scaledMaxLiquidity := core.MaxLiquidity * (100 - polRate) / 100
	if ld.GetBalance(f.GetId()) <= int64(scaledMaxLiquidity) {
		return
	}
	difference := ld.GetBalance(f.GetId()) - int64(scaledMaxLiquidity)
	bank.AddDemandDeposit(f.GetId(), difference, ld)
	f.firmStateMutex.Lock()
	f.bankBalance += difference
	f.firmStateMutex.Unlock()
}

func (f *Firm) PayWages(ld *core.Ledger) {
	f.firmStateMutex.Lock()
	employeesCopy := make([]uint32, len(f.employees))
	copy(employeesCopy, f.employees)
	f.firmStateMutex.Unlock()

	for _, id := range employeesCopy {
		wage := ld.GetWageExpectation(id)
		if wage <= 0 {
			wage = core.MinWage
		}
		ld.Transfer(f.GetId(), id, wage)
	}
}

func (f *Firm) SetSavings(amount int64) {
	f.firmStateMutex.Lock()
	defer f.firmStateMutex.Unlock()
	f.bankBalance = amount
}

func (f *Firm) GetPrice() int {
	f.firmStateMutex.Lock()
	defer f.firmStateMutex.Unlock()
	return f.stockPrice
}

func (f *Firm) GetType() core.AgentType {
	return core.Firm
}

func (f *Firm) AddInventory(units uint32) {
	f.firmStateMutex.Lock()
	defer f.firmStateMutex.Unlock()
	f.inventory += units
}

func (f *Firm) Log() {
	f.firmStateMutex.Lock()
	target := f.invTarget
	curr := f.invCurr
	price := f.stockPrice
	nEmp := len(f.employees)
	f.firmStateMutex.Unlock()
	fmt.Printf("Firm <%d> -- Target: %d -- Actual: %d -- Price: %d -- %d Employees\n", 
							f.GetId(), target, curr, price, nEmp)
}

