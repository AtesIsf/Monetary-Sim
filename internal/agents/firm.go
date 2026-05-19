package agents

import (
	"fmt"
	"math/rand/v2"

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
}

func NewFirm(id uint32) *Firm {
	var f Firm
	f.id.AType = core.Firm
	f.id.Id = id

	// TODO: You may want to change how the target is set in the future
	f.invTarget = uint32(rand.IntN(100)) + 500 // 500 <= target <= 600
	f.invCurr = 0

	return &f
}

func (f *Firm) AddEmployee(id uint32) {
	f.employees = append(f.employees, id)
}

func (f *Firm) GetId() uint32 {
	return f.id.Id
}

// TODO: Finish this
func (f *Firm) Update(pol *core.Policies, ld *core.Ledger) core.UpdateReturn {
	var returnVal core.UpdateReturn = core.Nothing

	if f.invCurr < f.invTarget { // Underproducing, so hire
		returnVal = core.HireWorkers
	} else if f.invCurr > f.invTarget { // Overproducing, so fire
		returnVal = core.FireWorkers
	}

	selfId := f.GetId()
	for id := range f.employees {
		ld.Transfer(selfId, uint32(id), 20) // assume flat wage of 20 for now
	}

	currBal := ld.GetBalance(f.GetId())
	if currBal < 0 { // take out a loan
		returnVal = core.RequestLoan
	}

	return returnVal
}

func (f *Firm) GetType() core.AgentType {
	return core.Firm
}

func (f *Firm) Log() {
	fmt.Printf("Firm <%d> -- Target: %d -- Actual: %d\n", 
							f.GetId(), f.invTarget, f.invCurr)
}

