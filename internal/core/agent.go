package core

/*
 * agent.go
 *
 * This file defines and implements a generic agent for this economy.
 * All more specialized agents implement this Agent interface.
 */

type Agent interface {
	GetId() uint32
	Update(pol *Policies, ld *Ledger) error
	GetType() AgentType
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

