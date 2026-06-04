package agents

import (
	"testing"

	"github.com/AtesIsf/monetary-simulator/internal/core"
)

func TestNewHousehold(t *testing.T) {
	h := NewHousehold(1)
	if h.GetId() != 1 {
		t.Errorf("expected ID 1, got %d", h.GetId())
	}
	if h.GetType() != core.Household {
		t.Errorf("expected type Household, got %d", h.GetType())
	}
	if h.IsEmployed() {
		t.Error("new household should be unemployed initially")
	}
	if h.GetEmployer() != 1 {
		t.Errorf("expected employer to be self (1), got %d", h.GetEmployer())
	}
	if h.mpcY < 0.6 || h.mpcY > 0.9 {
		t.Errorf("mpcY out of range: %f", h.mpcY)
	}
	if h.mpcB < 0.02 || h.mpcB > 0.05 {
		t.Errorf("mpcB out of range: %f", h.mpcB)
	}
	if h.c0 < 1 || h.c0 > 5 {
		t.Errorf("c0 out of range: %d", h.c0)
	}
	if h.wageExpectation != int64(core.StandardWage) {
		t.Errorf("expected wageExpectation to be %d, got %d", core.StandardWage, h.wageExpectation)
	}
}

func TestHousehold_IsEmployed(t *testing.T) {
	h := NewHousehold(1)
	if h.IsEmployed() {
		t.Error("expected unemployed")
	}
	h.SetEmployer(2)
	if !h.IsEmployed() {
		t.Error("expected employed")
	}
}

func TestHousehold_SetEmployer(t *testing.T) {
	h := NewHousehold(1)
	h.SetEmployer(5)
	if got := h.GetEmployer(); got != 5 {
		t.Errorf("expected employer 5, got %d", got)
	}
}

func TestHousehold_SetSavings(t *testing.T) {
	h := NewHousehold(1)
	h.SetSavings(300)
	if h.bankBalance != 300 {
		t.Errorf("expected bankBalance 300, got %d", h.bankBalance)
	}
}

func TestConsumption_Unemployed(t *testing.T) {
	h := NewHousehold(1)
	h.c0 = 5
	h.mpcB = 0.05

	var ld core.Ledger
	ld.Init()
	ld.AddToBalance(1, 100)

	var mt core.MacroTracker = dummyMacroTracker{}

	// Unemployed consumption: c0 * inflationRate (1.0) + mpcB * balance => 5 * 1.0 + 0.05 * 100 = 10
	got := h.CalculateConsumption(&ld, &mt)
	if got != 10 {
		t.Errorf("expected consumption to be 10, got %d", got)
	}
}

func TestConsumption_Employed(t *testing.T) {
	h := NewHousehold(1)
	h.c0 = 5
	h.mpcB = 0.05
	h.mpcY = 0.8
	h.SetEmployer(2) // Employed

	var ld core.Ledger
	ld.Init()
	ld.AddToBalance(1, 100)

	var mt core.MacroTracker = dummyMacroTracker{}

	// Employed consumption: c0 * 1.0 + mpcB * 100 + mpcY * StandardWage => 5 + 5 + 0.8 * 20 = 26
	got := h.CalculateConsumption(&ld, &mt)
	if got != 26 {
		t.Errorf("expected consumption to be 26, got %d", got)
	}
}

func TestConsumption_NegativeBalance(t *testing.T) {
	h := NewHousehold(1)
	h.c0 = 5
	h.mpcB = 0.05

	var ld core.Ledger
	ld.Init()
	ld.AddToBalance(1, -100)

	var mt core.MacroTracker = dummyMacroTracker{}

	// Negative balance clamped to 0: 5 + 0 = 5
	got := h.CalculateConsumption(&ld, &mt)
	if got != 5 {
		t.Errorf("expected consumption to be 5, got %d", got)
	}
}

func TestConsumption_MutatesC0(t *testing.T) {
	h := NewHousehold(1)
	h.c0 = 5

	var ld core.Ledger
	ld.Init()

	var mt core.MacroTracker = dummyMacroTracker{}

	// The current codebase mutates h.c0. Let's verify it mutates:
	_ = h.CalculateConsumption(&ld, &mt)
	// If it mutates, h.c0 is updated. If inflation is 1.0, it remains 5, but what if inflation is 0.5?
	// Let's use a tracker with 0.5 inflation:
	mtHalf := mockMacro{inflation: 0.5}
	var mtHalfTracker core.MacroTracker = mtHalf
	_ = h.CalculateConsumption(&ld, &mtHalfTracker)
	if h.c0 != 2 { // 5 * 0.5 = 2.5 floored to 2
		t.Errorf("expected c0 to be mutated to 2, got %d", h.c0)
	}
}

func TestConsumption_C0_DeflationCollapse(t *testing.T) {
	h := NewHousehold(1)
	h.c0 = 5

	var ld core.Ledger
	ld.Init()

	mtDeflation := mockMacro{inflation: 0.8}
	var mtDeflationTracker core.MacroTracker = mtDeflation
	// Call CalculateConsumption repeatedly:
	// 5 * 0.8 = 4
	// 4 * 0.8 = 3
	// 3 * 0.8 = 2
	// 2 * 0.8 = 1
	// 1 * 0.8 = 0
	for range 5 {
		_ = h.CalculateConsumption(&ld, &mtDeflationTracker)
	}
	if h.c0 != 0 {
		t.Errorf("expected c0 to collapse to 0 under deflation mutation, got %d", h.c0)
	}
}

type mockMacro struct {
	unemployment float64
	inflation    float64
}

func (m mockMacro) GetUnemploymentRate() float64 { return m.unemployment }
func (m mockMacro) GetInflationRate() float64    { return m.inflation }

func TestUpdate_EmployedTightLabor(t *testing.T) {
	h := NewHousehold(1)
	h.SetEmployer(2) // Employed
	h.wageExpectation = 20

	var ld core.Ledger
	ld.Init()

	var pol core.Policies
	pol.Populate()

	mt := mockMacro{unemployment: 0.04, inflation: 1.0}
	h.Update(&pol, &ld, mt, 1)

	if h.wageExpectation != 21 {
		t.Errorf("expected wageExpectation to increase to 21 under tight labor, got %d", h.wageExpectation)
	}
}

func TestUpdate_EmployedLooseLabor(t *testing.T) {
	h := NewHousehold(1)
	h.SetEmployer(2)
	h.wageExpectation = 20

	var ld core.Ledger
	ld.Init()

	var pol core.Policies
	pol.Populate()

	mt := mockMacro{unemployment: 0.06, inflation: 1.0}
	h.Update(&pol, &ld, mt, 1)

	if h.wageExpectation != 20 {
		t.Errorf("expected wageExpectation to remain 20 under loose labor, got %d", h.wageExpectation)
	}
}

func TestUpdate_Unemployed(t *testing.T) {
	h := NewHousehold(1) // Unemployed
	h.wageExpectation = 20

	var ld core.Ledger
	ld.Init()

	var pol core.Policies
	pol.Populate()

	mt := mockMacro{unemployment: 0.06, inflation: 1.0}
	h.Update(&pol, &ld, mt, 1)

	if h.wageExpectation != 19 {
		t.Errorf("expected wageExpectation to decrease to 19 when unemployed, got %d", h.wageExpectation)
	}
}

func TestUpdate_Unemployed_MinWageFloor(t *testing.T) {
	h := NewHousehold(1)
	h.wageExpectation = core.MinWage // 1

	var ld core.Ledger
	ld.Init()

	var pol core.Policies
	pol.Populate()

	mt := mockMacro{unemployment: 0.06, inflation: 1.0}
	h.Update(&pol, &ld, mt, 1)

	if h.wageExpectation != core.MinWage {
		t.Errorf("expected wageExpectation to stay at MinWage (%d), got %d", core.MinWage, h.wageExpectation)
	}
}

func TestUpdate_InflationAdjustment_Tick12(t *testing.T) {
	h := NewHousehold(1)
	h.SetEmployer(2)
	h.wageExpectation = 20

	var ld core.Ledger
	ld.Init()

	var pol core.Policies
	pol.Populate()

	// inflation is 1.10, so scaled wage expectation should adjust on tick 12
	mt := mockMacro{unemployment: 0.06, inflation: 1.10}
	h.Update(&pol, &ld, mt, 12)

	// Since employed and unemployment = 0.06 (loose), tick-based adjustment does not change wage.
	// Annual inflation rate (1.10) scales the wage expectation: 20 * 1.10 = 22.
	if h.wageExpectation != 22 {
		t.Errorf("expected wageExpectation to adjust to 22 on annual inflation adjustment, got %d", h.wageExpectation)
	}
}

func TestUpdate_DeflationAdjustment_Tick12(t *testing.T) {
	h := NewHousehold(1)
	h.SetEmployer(2)
	h.wageExpectation = 20

	var ld core.Ledger
	ld.Init()

	var pol core.Policies
	pol.Populate()

	mt := mockMacro{unemployment: 0.06, inflation: 0.90}
	h.Update(&pol, &ld, mt, 12)

	// 20 * 0.90 = 18
	if h.wageExpectation != 18 {
		t.Errorf("expected wageExpectation to adjust to 18 on annual deflation adjustment, got %d", h.wageExpectation)
	}
}

func TestUpdate_ReturnsConsume(t *testing.T) {
	h := NewHousehold(1)
	h.c0 = 5
	h.mpcB = 0.0
	h.mpcY = 0.0

	var ld core.Ledger
	ld.Init()
	ld.AddToBalance(1, 10) // balance (10) >= consumption (5)

	var pol core.Policies
	pol.Populate()

	mt := mockMacro{unemployment: 0.06, inflation: 1.0}
	got := h.Update(&pol, &ld, mt, 1)

	if got != core.Consume {
		t.Errorf("expected Update to return core.Consume, got %d", got)
	}
}

func TestUpdate_ReturnsDrawSavings(t *testing.T) {
	h := NewHousehold(1)
	h.c0 = 15
	h.mpcB = 0.0
	h.mpcY = 0.0
	h.bankBalance = 20

	var ld core.Ledger
	ld.Init()
	ld.AddToBalance(1, 5) // balance (5) < consumption (15), but balance + savings (25) >= consumption

	var pol core.Policies
	pol.Populate()

	mt := mockMacro{unemployment: 0.06, inflation: 1.0}
	got := h.Update(&pol, &ld, mt, 1)

	if got != core.DrawSavings {
		t.Errorf("expected Update to return core.DrawSavings, got %d", got)
	}
}

func TestUpdate_ReturnsRequestLoan(t *testing.T) {
	h := NewHousehold(1)
	h.c0 = 30
	h.mpcB = 0.0
	h.mpcY = 0.0
	h.bankBalance = 5

	var ld core.Ledger
	ld.Init()
	ld.AddToBalance(1, 5) // balance + savings = 10 < consumption (30)

	var pol core.Policies
	pol.Populate()

	mt := mockMacro{unemployment: 0.06, inflation: 1.0}
	got := h.Update(&pol, &ld, mt, 1)

	if got != core.RequestLoan {
		t.Errorf("expected Update to return core.RequestLoan, got %d", got)
	}
}

func TestUpdate_SyncsWageToLedger(t *testing.T) {
	h := NewHousehold(1)
	h.wageExpectation = 25

	var ld core.Ledger
	ld.Init()

	var pol core.Policies
	pol.Populate()

	mt := mockMacro{unemployment: 0.06, inflation: 1.0}
	h.Update(&pol, &ld, mt, 1)

	if got := ld.GetWageExpectation(1); got != 24 { // Unemployed wage decreases by 1 to 24
		t.Errorf("expected ledger wage expectation to be synced to 24, got %d", got)
	}
}

func TestHouse_RepayLoans(t *testing.T) {
	b := NewBank(0)

	tests := []struct {
		name         string
		initialCash  int64
		initialBank  int64
		loanRem      uint64
		expectedCash int64
		expectedBank int64
		expectedRem  uint64
	}{
		{
			name:         "pay with demand deposits",
			initialCash:  0,
			initialBank:  100,
			loanRem:      50,
			expectedBank: 95, // installment = 5
			expectedRem:  45,
		},
		{
			name:         "pay with cash",
			initialCash:  100,
			initialBank:  0,
			loanRem:      50,
			expectedCash: 95,
			expectedRem:  45,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHousehold(1)
			var ld core.Ledger
			ld.Init()
			ld.AddToBalance(1, tt.initialCash)
			ld.AddToBalance(0, 1000)

			loan := &Loan{
				from:            core.AgentId{AType: core.Bank, Id: 0},
				to:              core.AgentId{AType: core.Household, Id: 1},
				initialAmount:   50,
				remainingAmount: tt.loanRem,
				installment:     10,
				interest:        5,
			}
			h.loans = []*Loan{loan}
			h.bankBalance = tt.initialBank
			b.demandDeposits[1] = tt.initialBank

			h.RepayLoans(b, &ld)

			if h.bankBalance != tt.expectedBank {
				t.Errorf("expected bankBalance to be %d, got %d", tt.expectedBank, h.bankBalance)
			}
			if loan.remainingAmount != tt.expectedRem {
				t.Errorf("expected loan remainingAmount to be %d, got %d", tt.expectedRem, loan.remainingAmount)
			}
		})
	}
}

func TestHouse_DepositExtra(t *testing.T) {
	b := NewBank(0)
	var ld core.Ledger
	ld.Init()
	ld.AddToBalance(1, core.MaxLiquidity+100)
	ld.AddToBalance(0, 1000)

	h := NewHousehold(1)
	h.DepositExtra(b, &ld)

	if got := ld.GetBalance(1); got != core.MaxLiquidity {
		t.Errorf("expected ledger balance to be core.MaxLiquidity, got %d", got)
	}
	if h.bankBalance != 100 {
		t.Errorf("expected bankBalance to be 100, got %d", h.bankBalance)
	}
}
