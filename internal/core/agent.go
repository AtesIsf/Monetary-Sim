package core

/*
 * agent.go
 *
 * This file defines and implements a generic agent for this economy.
 * All more specialized agents implement this Agent interface.
 */

type MacroTracker interface {
	GetUnemploymentRate() float64
	GetInflationRate() float64
}

type Agent interface {
	GetId() uint32
	Update(pol *Policies, ld *Ledger, macro MacroTracker, tick uint64) UpdateReturn
	GetType() AgentType
	Log()
}

type ActionType uint8
const (
	Nothing ActionType = iota
	// Household Update() Returns
	Consume
	// Firm Update() Returns
	HireWorkers
	FireWorkers
	RequestLoan
	DrawSavings
	// Bank Update() Returns
)

type UpdateReturn struct {
	Action  ActionType
	Count   uint32
	Finance ActionType
}

// These help identify what type of agent
type AgentType uint8
const (
	Household AgentType = iota
	Bank
	Firm
)
type AgentId struct {
	AType AgentType
	Id uint32
}

