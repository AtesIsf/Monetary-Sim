package core

import "testing"

func TestPolicies_Populate(t *testing.T) {
	var p Policies
	p.Populate()
	if got := p.GetInterestRate(); got != 4 {
		t.Errorf("interestRate = %d, want 4", got)
	}
	if got := p.GetReserveReq(); got != 0 {
		t.Errorf("reserveReq = %d, want 0", got)
	}
}

func TestPolicies_GetInterestRate(t *testing.T) {
	var p Policies
	p.interestRate = 12
	if got := p.GetInterestRate(); got != 12 {
		t.Errorf("GetInterestRate() = %d, want 12", got)
	}
}

func TestPolicies_GetReserveReq(t *testing.T) {
	var p Policies
	p.reserveReq = 5
	if got := p.GetReserveReq(); got != 5 {
		t.Errorf("GetReserveReq() = %d, want 5", got)
	}
}
