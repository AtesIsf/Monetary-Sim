package agents

import (
	"testing"

	"github.com/AtesIsf/monetary-simulator/internal/core"
)

func TestNewFirm(t *testing.T) {
	f := NewFirm(1)
	if f.GetId() != 1 {
		t.Errorf("expected ID to be 1, got %d", f.GetId())
	}
	if f.GetType() != core.Firm {
		t.Errorf("expected type to be Firm, got %d", f.GetType())
	}
	if f.GetPrice() != core.Price {
		t.Errorf("expected price to be core.Price (%d), got %d", core.Price, f.GetPrice())
	}
	if f.invCurr != 0 {
		t.Errorf("expected invCurr to be 0, got %d", f.invCurr)
	}
	if f.invTarget < 25 || f.invTarget > 30 {
		t.Errorf("expected invTarget to be in [25, 30], got %d", f.invTarget)
	}
}

func TestFirm_AddEmployee(t *testing.T) {
	f := NewFirm(1)
	f.AddEmployee(2)
	if got := f.GetNEmployees(); got != 1 {
		t.Errorf("expected employee count 1, got %d", got)
	}
	if f.employees[0] != 2 {
		t.Errorf("expected employee ID 2, got %d", f.employees[0])
	}
}

func TestFirm_PopEmployee(t *testing.T) {
	f := NewFirm(1)
	f.AddEmployee(2)
	f.AddEmployee(3)

	id := f.PopEmployee()
	if id != 2 && id != 3 {
		t.Errorf("expected popped ID to be 2 or 3, got %d", id)
	}
	if got := f.GetNEmployees(); got != 1 {
		t.Errorf("expected employee count to be 1, got %d", got)
	}
}

func TestFirm_PopEmployee_PanicsOnEmpty(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected PopEmployee on empty list to panic, but it did not")
		}
	}()
	f := NewFirm(1)
	f.PopEmployee()
}

func TestFirm_PerformSale(t *testing.T) {
	f := NewFirm(1)
	f.PerformSale(100)
	if f.invCurr != 100 {
		t.Errorf("expected invCurr to be 100, got %d", f.invCurr)
	}
}

func TestFirm_GetPrice(t *testing.T) {
	f := NewFirm(1)
	f.stockPrice = 30
	if got := f.GetPrice(); got != 30 {
		t.Errorf("expected GetPrice to return 30, got %d", got)
	}
}

func TestFirm_AdaptPrice_Underselling(t *testing.T) {
	f := NewFirm(1)
	f.stockPrice = 20
	// target > current => delta > 0. adaptPrice decrements price by rand.IntN(3) + 1 (1 to 3)
	f.adaptPrice(10)
	if f.stockPrice >= 20 || f.stockPrice < 17 {
		t.Errorf("expected price to decrease to range [17, 19], got %d", f.stockPrice)
	}
}

func TestFirm_AdaptPrice_Overselling(t *testing.T) {
	f := NewFirm(1)
	f.stockPrice = 20
	// target < current => delta < 0. adaptPrice increments price by 1 to 3
	f.adaptPrice(-10)
	if f.stockPrice <= 20 || f.stockPrice > 23 {
		t.Errorf("expected price to increase to range [21, 23], got %d", f.stockPrice)
	}
}

func TestFirm_AdaptPrice_OnTarget(t *testing.T) {
	f := NewFirm(1)
	f.stockPrice = 20
	f.adaptPrice(0)
	if f.stockPrice != 20 {
		t.Errorf("expected price to remain 20, got %d", f.stockPrice)
	}
}

func TestFirm_AdaptPrice_MinPriceFloor(t *testing.T) {
	f := NewFirm(1)
	f.stockPrice = core.MinPrice // 1
	f.adaptPrice(10)             // should try to decrease
	if f.stockPrice < core.MinPrice {
		t.Errorf("expected price to floor at core.MinPrice (%d), got %d", core.MinPrice, f.stockPrice)
	}
}

func TestFirm_Update_HireWhenUnderTarget(t *testing.T) {
	f := NewFirm(1)
	f.invTarget = 25
	f.invCurr = 20
	var ld core.Ledger
	ld.Init()
	ld.AddToBalance(1, 1000)

	var pol core.Policies
	pol.Populate()

	got := f.Update(&pol, &ld, dummyMacroTracker{}, 1)
	if got.Action != core.HireWorkers {
		t.Errorf("expected HireWorkers, got %d", got.Action)
	}
}

func TestFirm_Update_FireWhenOverTarget(t *testing.T) {
	f := NewFirm(1)
	f.invTarget = 25
	f.invCurr = 30
	f.AddEmployee(2)
	var ld core.Ledger
	ld.Init()
	ld.AddToBalance(1, 1000)

	var pol core.Policies
	pol.Populate()

	got := f.Update(&pol, &ld, dummyMacroTracker{}, 1)
	if got.Action != core.FireWorkers {
		t.Errorf("expected FireWorkers, got %d", got.Action)
	}
}

func TestFirm_Update_YearlyReset(t *testing.T) {
	f := NewFirm(1)
	f.invCurr = 20
	var ld core.Ledger
	ld.Init()
	ld.AddToBalance(1, 1000)
	var pol core.Policies
	pol.Populate()

	f.Update(&pol, &ld, dummyMacroTracker{}, 12) // tick % 12 == 0 resets invCurr
	if f.invCurr != 0 {
		t.Errorf("expected invCurr to reset to 0, got %d", f.invCurr)
	}
}

func TestFirm_Update_FundingCheck(t *testing.T) {
	var pol core.Policies
	pol.Populate()

	tests := []struct {
		name            string
		ledgerBal       int64
		bankBal         int64
		employees       []uint32
		expectedAction  core.ActionType
		expectedFinance core.ActionType
	}{
		{
			name:            "sufficient ledger balance",
			ledgerBal:       100,
			bankBal:         0,
			employees:       []uint32{2}, // expects 20
			expectedAction:  core.HireWorkers,
			expectedFinance: core.Nothing,
		},
		{
			name:            "insufficient ledger, sufficient bank balance",
			ledgerBal:       10,
			bankBal:         50,
			employees:       []uint32{2}, // expects 20
			expectedAction:  core.HireWorkers,
			expectedFinance: core.DrawSavings,
		},
		{
			name:            "insufficient ledger and bank balance",
			ledgerBal:       0,
			bankBal:         0,
			employees:       []uint32{2},
			expectedAction:  core.HireWorkers,
			expectedFinance: core.RequestLoan,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewFirm(1)
			f.invTarget = 25
			f.invCurr = 0
			for _, emp := range tt.employees {
				f.AddEmployee(emp)
			}
			f.bankBalance = tt.bankBal

			var ld core.Ledger
			ld.Init()
			ld.AddToBalance(1, tt.ledgerBal)
			for _, emp := range tt.employees {
				ld.SetWageExpectation(emp, 20)
			}

			got := f.Update(&pol, &ld, dummyMacroTracker{}, 1)
			if got.Action != tt.expectedAction {
				t.Errorf("got Action %d, want %d", got.Action, tt.expectedAction)
			}
			if got.Finance != tt.expectedFinance {
				t.Errorf("got Finance %d, want %d", got.Finance, tt.expectedFinance)
			}
		})
	}
}

func TestFirm_PayWages_UsesWageExpectations(t *testing.T) {
	f := NewFirm(1)
	f.AddEmployee(2)
	f.AddEmployee(3)

	var ld core.Ledger
	ld.Init()
	ld.AddToBalance(1, 100)
	ld.SetWageExpectation(2, 25)
	ld.SetWageExpectation(3, 35)

	f.PayWages(&ld)

	if got := ld.GetBalance(2); got != 25 {
		t.Errorf("expected employee 2 to get 25, got %d", got)
	}
	if got := ld.GetBalance(3); got != 35 {
		t.Errorf("expected employee 3 to get 35, got %d", got)
	}
	if got := ld.GetBalance(1); got != 40 {
		t.Errorf("expected firm balance to be 40, got %d", got)
	}
}

func TestFirm_PayWages_MinWageFallback(t *testing.T) {
	f := NewFirm(1)
	f.AddEmployee(2)

	var ld core.Ledger
	ld.Init()
	ld.AddToBalance(1, 100)
	// wage expectation is 0 (default) or negative
	ld.SetWageExpectation(2, 0)

	f.PayWages(&ld)

	if got := ld.GetBalance(2); got != core.MinWage {
		t.Errorf("expected employee 2 to get core.MinWage (%d), got %d", core.MinWage, got)
	}
}

func TestFirm_PayWages_NoEmployees(t *testing.T) {
	f := NewFirm(1)
	var ld core.Ledger
	ld.Init()
	ld.AddToBalance(1, 100)

	f.PayWages(&ld)

	if got := ld.GetBalance(1); got != 100 {
		t.Errorf("expected firm balance to remain 100, got %d", got)
	}
}

func TestFirm_RepayLoans(t *testing.T) {
	// RepayLoans has multiple branches. Let's test them.
	// Initializing helper:
	b := NewBank(0)

	tests := []struct {
		name          string
		initialCash   int64
		initialBank   int64
		loanRem       uint64
		expectedCash  int64
		expectedBank  int64
		expectedRem   uint64
	}{
		{
			name:         "pay with demand deposits",
			initialCash:  0,
			initialBank:  100,
			loanRem:      50,
			expectedCash: 0,
			expectedBank: 95, // installment = initial/10 = 50/10 = 5
			expectedRem:  45,
		},
		{
			name:         "pay with cash",
			initialCash:  100,
			initialBank:  0,
			loanRem:      50,
			expectedCash: 95, // transfers 5 to bank
			expectedBank: 0,
			expectedRem:  45,
		},
		{
			name:         "pay combined",
			initialCash:  2,
			initialBank:  3,
			loanRem:      50,
			expectedCash: 0, // needs 5. Takes all 3 from bank, remaining 2 from cash
			expectedBank: 0,
			expectedRem:  45,
		},
		{
			name:         "cannot pay -> default",
			initialCash:  0,
			initialBank:  0,
			loanRem:      50,
			expectedCash: 0,
			expectedBank: 0,
			expectedRem:  0, // loan is zeroed out
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewFirm(1)
			var ld core.Ledger
			ld.Init()
			ld.AddToBalance(1, tt.initialCash)
			ld.AddToBalance(0, 1000) // bank initial ledger balance

			loan := &Loan{
				from:            core.AgentId{AType: core.Bank, Id: 0},
				to:              core.AgentId{AType: core.Firm, Id: 1},
				initialAmount:   50,
				remainingAmount: tt.loanRem,
				installment:     10,
				interest:        5,
			}
			f.loans = []*Loan{loan}
			f.bankBalance = tt.initialBank
			b.demandDeposits[1] = tt.initialBank

			f.RepayLoans(b, &ld)

			if f.bankBalance != tt.expectedBank {
				t.Errorf("expected firm bankBalance to be %d, got %d", tt.expectedBank, f.bankBalance)
			}
			if b.demandDeposits[1] != tt.expectedBank {
				t.Errorf("expected bank demand deposit to be %d, got %d", tt.expectedBank, b.demandDeposits[1])
			}
			if got := ld.GetBalance(1); got != tt.expectedCash {
				t.Errorf("expected firm ledger balance to be %d, got %d", tt.expectedCash, got)
			}
			if loan.remainingAmount != tt.expectedRem {
				t.Errorf("expected loan remainingAmount to be %d, got %d", tt.expectedRem, loan.remainingAmount)
			}
		})
	}
}

func TestFirm_DepositExtra_AboveThreshold(t *testing.T) {
	b := NewBank(0)
	var ld core.Ledger
	ld.Init()
	ld.AddToBalance(1, core.MaxLiquidity+100) // 80 + 100 = 180
	ld.AddToBalance(0, 1000)                 // bank ledger balance

	f := NewFirm(1)
	f.DepositExtra(b, &ld, 0)

	if got := ld.GetBalance(1); got != core.MaxLiquidity {
		t.Errorf("expected firm ledger balance to be core.MaxLiquidity (%d), got %d", core.MaxLiquidity, got)
	}
	if got := b.QueryDeposits(1); got != 100 {
		t.Errorf("expected demand deposits to be 100, got %d", got)
	}
	if f.bankBalance != 100 {
		t.Errorf("expected firm bankBalance to be 100, got %d", f.bankBalance)
	}
}

func TestFirm_DepositExtra_BelowThreshold(t *testing.T) {
	b := NewBank(0)
	var ld core.Ledger
	ld.Init()
	ld.AddToBalance(1, core.MaxLiquidity-50)
	ld.AddToBalance(0, 1000)

	f := NewFirm(1)
	f.DepositExtra(b, &ld, 0)

	if got := ld.GetBalance(1); got != core.MaxLiquidity-50 {
		t.Errorf("expected firm ledger balance to remain same, got %d", got)
	}
	if got := b.QueryDeposits(1); got != 0 {
		t.Errorf("expected demand deposits to be 0, got %d", got)
	}
}

func TestFirm_Update_RateSensitiveBorrowing(t *testing.T) {
	tests := []struct {
		name            string
		policyRate      uint32
		expectedFinance core.ActionType
	}{
		{
			name:            "interest rate below threshold",
			policyRate:      10,
			expectedFinance: core.RequestLoan,
		},
		{
			name:            "interest rate above threshold",
			policyRate:      20,
			expectedFinance: core.Nothing,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewFirm(1)
			f.invTarget = 25
			f.invCurr = 0
			f.AddEmployee(2) // expects wage 20
			f.bankBalance = 0

			var ld core.Ledger
			ld.Init()
			ld.AddToBalance(1, 0) // short of cash
			ld.SetWageExpectation(2, 20)

			var pol core.Policies
			pol.Populate()
			pol.SetInterestRate(tt.policyRate)

			got := f.Update(&pol, &ld, dummyMacroTracker{}, 1)
			if got.Finance != tt.expectedFinance {
				t.Errorf("got Finance %d, want %d", got.Finance, tt.expectedFinance)
			}
		})
	}
}
