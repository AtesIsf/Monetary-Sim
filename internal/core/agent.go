package core

/*
 * agent.go
 *
 * This file defines and implements a generic agent for this economy.
 * All more specialized agents implement this Agent interface.
 */

type Agent interface {
	GetId() uint32
	Update(pol *Policies, ld *Ledger) UpdateReturn
	GetType() AgentType
	Log()
}

type UpdateReturn uint8
const (
	// Firm Update() Returns
	Nothing UpdateReturn = iota
	HireWorkers
	FireWorkers
	RequestLoan
	// Household Update() Returns
	// Bank Update() Returns
)

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

