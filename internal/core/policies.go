package core

import "sync"

/*
 * policies.go
 *
 * This file implements the functionality for the policymaker, which the user
 * controls. It sets variables such as the reserve rate and the policy rate.
 */

// Defines how many ticks there are per year
const TicksPerYear = 12

type Policies struct {
	interestMutex sync.RWMutex
	reserveMutex sync.RWMutex
	// both values below are integers between 0 and 100
	interestRate uint32
	reserveReq uint32
}

func (p *Policies) Populate() {
	p.interestRate = 4
	p.reserveReq = 0
}

func (p *Policies) GetInterestRate() uint32 {
	p.interestMutex.RLock()
	defer p.interestMutex.RUnlock()
	return p.interestRate
}

func (p *Policies) SetInterestRate(rate uint32) {
	p.interestMutex.Lock()
	defer p.interestMutex.Unlock()
	p.interestRate = rate
}

func (p *Policies) GetReserveReq() uint32 {
	p.reserveMutex.RLock()
	defer p.reserveMutex.RUnlock()
	return p.reserveReq
}

func (p *Policies) SetReserveReq(rate uint32) {
	p.reserveMutex.Lock()
	defer p.reserveMutex.Unlock()
	p.reserveReq = rate
}
