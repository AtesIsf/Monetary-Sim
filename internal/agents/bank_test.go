package agents

import (
	"testing"

	"github.com/AtesIsf/monetary-simulator/internal/core"
)

type dummyMacroTracker struct{}

func (d dummyMacroTracker) GetUnemploymentRate() float64 { return 0.0 }
func (d dummyMacroTracker) GetInflationRate() float64    { return 1.0 }

func TestNewBank(t *testing.T) {
	b := NewBank(10)
	if b.GetId() != 10 {
		t.Errorf("expected bank ID to be 10, got %d", b.GetId())
	}
	if b.GetType() != core.Bank {
		t.Errorf("expected type to be Bank, got %d", b.GetType())
	}
	if b.demandDeposits == nil {
		t.Error("demandDeposits map is nil")
	}
	if b.debt != 0 {
		t.Errorf("expected initial debt to be 0, got %d", b.debt)
	}
}

func TestBank_AddDemandDeposit(t *testing.T) {
	var ld core.Ledger
	ld.Init()
	b := NewBank(0)
	ld.AddToBalance(0, 1000) // Bank initial balance
	ld.AddToBalance(1, 500)  // User initial balance

	b.AddDemandDeposit(1, 100, &ld)

	if got := b.QueryDeposits(1); got != 100 {
		t.Errorf("expected deposit of 100, got %d", got)
	}
	// Ledger transfer check: 1 should lose 100, bank should gain 100
	if got := ld.GetBalance(1); got != 400 {
		t.Errorf("expected user ledger balance to be 400, got %d", got)
	}
	if got := ld.GetBalance(0); got != 1100 {
		t.Errorf("expected bank ledger balance to be 1100, got %d", got)
	}
}

func TestBank_AddDemandDeposit_Multiple(t *testing.T) {
	var ld core.Ledger
	ld.Init()
	b := NewBank(0)
	ld.AddToBalance(0, 1000)
	ld.AddToBalance(1, 500)

	b.AddDemandDeposit(1, 100, &ld)
	b.AddDemandDeposit(1, 200, &ld)

	if got := b.QueryDeposits(1); got != 300 {
		t.Errorf("expected deposit of 300, got %d", got)
	}
	if got := ld.GetBalance(1); got != 200 {
		t.Errorf("expected user ledger balance to be 200, got %d", got)
	}
}

func TestBank_QueryDeposits(t *testing.T) {
	b := NewBank(0)
	if got := b.QueryDeposits(999); got != 0 {
		t.Errorf("expected default query deposit to be 0, got %d", got)
	}
}

func TestBank_DecreaseDemandDeposit(t *testing.T) {
	b := NewBank(0)
	b.demandDeposits[1] = 500
	b.DecreaseDemandDeposit(1, 200)
	if got := b.QueryDeposits(1); got != 300 {
		t.Errorf("expected deposit to be 300, got %d", got)
	}
}

func TestBank_WithdrawAll(t *testing.T) {
	var ld core.Ledger
	ld.Init()
	b := NewBank(0)
	ld.AddToBalance(0, 1000)
	b.demandDeposits[1] = 300

	b.WithdrawAll(1, &ld)

	if got := b.QueryDeposits(1); got != 0 {
		t.Errorf("expected deposit after withdraw to be 0, got %d", got)
	}
	if got := ld.GetBalance(1); got != 300 {
		t.Errorf("expected user ledger balance to be 300, got %d", got)
	}
	if got := ld.GetBalance(0); got != 700 {
		t.Errorf("expected bank ledger balance to be 700, got %d", got)
	}
}

func TestBank_WithdrawAll_ZeroDeposit(t *testing.T) {
	var ld core.Ledger
	ld.Init()
	b := NewBank(0)
	ld.AddToBalance(0, 1000)

	b.WithdrawAll(1, &ld)

	if got := b.QueryDeposits(1); got != 0 {
		t.Errorf("expected deposit to be 0, got %d", got)
	}
	if got := ld.GetBalance(1); got != 0 {
		t.Errorf("expected user ledger balance to be 0, got %d", got)
	}
	if got := ld.GetBalance(0); got != 1000 {
		t.Errorf("expected bank ledger balance to be 1000, got %d", got)
	}
}

func TestBank_IssueLoan(t *testing.T) {
	var ld core.Ledger
	ld.Init()
	b := NewBank(0)
	ld.AddToBalance(0, 500)

	target := core.AgentId{AType: core.Household, Id: 1}
	loan := b.IssueLoan(&ld, target, 200, 5)

	if loan == nil {
		t.Fatal("expected loan to be issued, got nil")
	}
	if loan.from.Id != 0 {
		t.Errorf("expected loan from 0, got %d", loan.from.Id)
	}
	if loan.to.Id != 1 {
		t.Errorf("expected loan to 1, got %d", loan.to.Id)
	}
	// interest is 5, amount is 200. amountWithInterest is 200 * 5 / 100 = 10? Wait.
	// Oh, wait, in bank.go line 55: amountWithInterest := amount * int64(interest) / 100
	// Wait, is it amount * int64(interest) / 100, or amount + amount * int64(interest) / 100?
	// Let's check bank.go: amountWithInterest := amount * int64(interest) / 100.
	// That means it's 200 * 5 / 100 = 10! Wait! If it's 10, then the remaining amount is 10.
	// Let's check:
	expectedAmount := uint64(200 * 5 / 100) // 10
	if loan.initialAmount != expectedAmount {
		t.Errorf("expected initialAmount to be %d, got %d", expectedAmount, loan.initialAmount)
	}
	if loan.remainingAmount != expectedAmount {
		t.Errorf("expected remainingAmount to be %d, got %d", expectedAmount, loan.remainingAmount)
	}
	if loan.installment != 12 {
		t.Errorf("expected installment to be 12, got %d", loan.installment)
	}
	if loan.interest != 5 {
		t.Errorf("expected interest to be 5, got %d", loan.interest)
	}

	// Ledger check: bank lost 200, user gained 200
	if got := ld.GetBalance(0); got != 300 {
		t.Errorf("expected bank ledger balance to be 300, got %d", got)
	}
	if got := ld.GetBalance(1); got != 200 {
		t.Errorf("expected user ledger balance to be 200, got %d", got)
	}
}

func TestBank_IssueLoan_InsufficientFunds(t *testing.T) {
	var ld core.Ledger
	ld.Init()
	b := NewBank(0)
	ld.AddToBalance(0, 100)

	target := core.AgentId{AType: core.Household, Id: 1}
	loan := b.IssueLoan(&ld, target, 200, 5)

	if loan != nil {
		t.Error("expected loan to be nil due to insufficient funds")
	}
	if got := ld.GetBalance(0); got != 100 {
		t.Errorf("expected bank ledger balance to remain 100, got %d", got)
	}
	if got := ld.GetBalance(1); got != 0 {
		t.Errorf("expected user ledger balance to remain 0, got %d", got)
	}
}

func TestBank_Update_InterestCalculation(t *testing.T) {
	// Interest is updated every 12 ticks
	b := NewBank(0)
	b.demandDeposits[1] = 1000

	var pol core.Policies
	pol.Populate() // interestRate is 4

	// When ticks % 12 != 0, nothing changes
	b.Update(&pol, nil, dummyMacroTracker{}, 1)
	if got := b.QueryDeposits(1); got != 1000 {
		t.Errorf("expected deposits to remain 1000 on tick 1, got %d", got)
	}

	// When ticks % 12 == 0, interest is added
	// rate = policy interest rate (4) - 2 = 2.
	// interest added = 1000 * 2 / 100 = 20.
	// New balance = 1020.
	b.Update(&pol, nil, dummyMacroTracker{}, 12)
	if got := b.QueryDeposits(1); got != 1020 {
		t.Errorf("expected deposits to be 1020 after interest update, got %d", got)
	}
}
