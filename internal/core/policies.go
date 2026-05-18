package core

import "sync"

/*
 * policies.go
 *
 * This file implements the functionality for the policymaker, which the user
 * controls. It sets variables such as the reserve rate and the policy rate.
 */

type Policies struct {
	mutex sync.RWMutex
	// both values below are integers between 0 and 100
	interestRate uint8
	reserveReq uint8
}

func (p *Policies) Populate() {
	p.interestRate = 4
	p.reserveReq = 0
}

func (p *Policies) GetInterestRate() uint8 {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return p.interestRate
}

func (p *Policies) GetReserveReq() uint8 {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return p.reserveReq
}
