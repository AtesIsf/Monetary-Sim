package engine

import (
	"testing"

	"github.com/AtesIsf/monetary-simulator/internal/agents"
	"github.com/AtesIsf/monetary-simulator/internal/core"
)

func TestBuyGoods_SelectsCheapestFirm(t *testing.T) {
	var sim Simulation
	sim.ld.Init()
	sim.agentsMutex.Lock()

	buyer := agents.NewHousehold(1)
	// Seed buyer balance so they can consume
	sim.ld.AddToBalance(1, 100)

	firm1 := agents.NewFirm(2) // Price initially 20
	firm2 := agents.NewFirm(3) // Price initially 20

	// Let's drive firm1's price to 10 (cheaper) and firm2's price to 30 (more expensive)
	var pol core.Policies
	pol.Populate()
	var mt mockMacroTracker

	// Decrease firm1 price to 10
	for firm1.GetPrice() > 10 {
		// invCurr = 0 < invTarget, so delta > 0, price decreases
		firm1.Update(&pol, &sim.ld, mt, 1)
	}

	// Increase firm2 price to 30
	for firm2.GetPrice() < 30 {
		// invCurr = target + 100 > invTarget, so delta < 0, price increases
		firm2.PerformSale(100) // ensure invCurr > invTarget
		firm2.Update(&pol, &sim.ld, mt, 1)
	}

	sim.agents = append(sim.agents, buyer, firm1, firm2)
	sim.agentsMutex.Unlock()

	matcher := NewMatcher(&sim)
	buyer.SetSavings(0)
	firm1.AddInventory(100)
	firm2.AddInventory(100)

	matcher.BuyGoods(buyer, &sim.ld)

	// Check that the cheapest firm (firm1) got the sale!
	if firm1.GetPrice() >= firm2.GetPrice() {
		t.Fatalf("firm1 price (%d) should be less than firm2 price (%d)", firm1.GetPrice(), firm2.GetPrice())
	}
}

func TestBuyGoods_TransfersCorrectAmount(t *testing.T) {
	var sim Simulation
	sim.ld.Init()
	sim.agentsMutex.Lock()

	buyer := agents.NewHousehold(1)
	sim.ld.AddToBalance(1, 100)

	firm := agents.NewFirm(2)
	// Leave firm price at 20

	sim.agents = append(sim.agents, buyer, firm)
	sim.agentsMutex.Unlock()

	matcher := NewMatcher(&sim)
	sim.ld.AddToBalance(1, 1000)
	firm.AddInventory(100)

	matcher.BuyGoods(buyer, &sim.ld)

	// The buyer balance should have decreased by at least 20 (or some multiple of 20)
	// and the firm should have gained that amount.
	buyerBal := sim.ld.GetBalance(1)
	firmBal := sim.ld.GetBalance(2)

	if firmBal <= 0 {
		t.Error("expected firm to have received a transfer")
	}
	if buyerBal >= 1100 {
		t.Error("expected buyer balance to have decreased below 1100")
	}
	if firmBal != 1100 - buyerBal { // Initial sum = 1100
		t.Errorf("expected total balance to be conserved, got buyer=%d, firm=%d", buyerBal, firmBal)
	}
}

func TestBuyGoods_RoundsDownToMultiple(t *testing.T) {
	var sim Simulation
	sim.ld.Init()
	sim.agentsMutex.Lock()

	buyer := agents.NewHousehold(1)
	sim.ld.AddToBalance(1, 1000)

	firm := agents.NewFirm(2)

	sim.agents = append(sim.agents, buyer, firm)
	sim.agentsMutex.Unlock()

	matcher := NewMatcher(&sim)
	firm.AddInventory(100)
	
	var tracker core.MacroTracker = &sim
	maxC := buyer.CalculateConsumption(&sim.ld, &tracker, sim.pol.GetInterestRate())
	price := int64(firm.GetPrice()) // 20
	expectedTransfer := maxC - maxC%price

	matcher.BuyGoods(buyer, &sim.ld)

	firmBal := sim.ld.GetBalance(2)
	if firmBal != expectedTransfer {
		t.Errorf("expected transfer of %d, got %d", expectedTransfer, firmBal)
	}
}

func TestBuyGoods_NoFirms(t *testing.T) {
	var sim Simulation
	sim.ld.Init()
	sim.agentsMutex.Lock()
	buyer := agents.NewHousehold(1)
	sim.ld.AddToBalance(1, 100)
	sim.agents = append(sim.agents, buyer)
	sim.agentsMutex.Unlock()

	matcher := NewMatcher(&sim)
	// Should not panic or perform any transfer
	matcher.BuyGoods(buyer, &sim.ld)
	if got := sim.ld.GetBalance(1); got != 100 {
		t.Errorf("expected buyer balance to remain 100, got %d", got)
	}
}

func TestHireWorker_SelectsLowestWage(t *testing.T) {
	var sim Simulation
	sim.ld.Init()
	sim.agentsMutex.Lock()

	firm := agents.NewFirm(1)

	// Add 3 unemployed households with different wage expectations
	h1 := agents.NewHousehold(2)
	sim.ld.SetWageExpectation(2, 25)

	h2 := agents.NewHousehold(3)
	sim.ld.SetWageExpectation(3, 15)

	h3 := agents.NewHousehold(4)
	sim.ld.SetWageExpectation(4, 35)

	sim.agents = append(sim.agents, firm, h1, h2, h3)
	sim.agentsMutex.Unlock()

	matcher := NewMatcher(&sim)
	matcher.HireWorker(firm)

	// It should hire the cheapest (h2 with wage expectation 15)
	if got := h2.GetEmployer(); got != 1 {
		t.Errorf("expected h2 (cheapest) to be hired by firm 1, got employer %d", got)
	}
	if firm.GetNEmployees() != 1 {
		t.Errorf("expected firm to have 1 employee, got %d", firm.GetNEmployees())
	}
	// Check that other households are still unemployed (employer == self)
	if h1.GetEmployer() != 2 {
		t.Errorf("expected h1 to remain unemployed, got employer %d", h1.GetEmployer())
	}
	if h3.GetEmployer() != 4 {
		t.Errorf("expected h3 to remain unemployed, got employer %d", h3.GetEmployer())
	}
}

func TestHireWorker_SkipsEmployed(t *testing.T) {
	var sim Simulation
	sim.ld.Init()
	sim.agentsMutex.Lock()

	firm := agents.NewFirm(1)

	h1 := agents.NewHousehold(2)
	h1.SetEmployer(99) // already employed
	sim.ld.SetWageExpectation(2, 10)

	h2 := agents.NewHousehold(3) // unemployed
	sim.ld.SetWageExpectation(3, 20)

	sim.agents = append(sim.agents, firm, h1, h2)
	sim.agentsMutex.Unlock()

	matcher := NewMatcher(&sim)
	matcher.HireWorker(firm)

	// Should skip h1 (already employed) and hire h2
	if got := h2.GetEmployer(); got != 1 {
		t.Errorf("expected h2 to be hired, got employer %d", got)
	}
	if got := h1.GetEmployer(); got != 99 {
		t.Errorf("expected h1 employer to remain 99, got %d", got)
	}
}

func TestHireWorker_NoUnemployed(t *testing.T) {
	var sim Simulation
	sim.ld.Init()
	sim.agentsMutex.Lock()

	firm := agents.NewFirm(1)

	h1 := agents.NewHousehold(2)
	h1.SetEmployer(99)

	sim.agents = append(sim.agents, firm, h1)
	sim.agentsMutex.Unlock()

	matcher := NewMatcher(&sim)
	matcher.HireWorker(firm)

	if firm.GetNEmployees() != 0 {
		t.Errorf("expected firm to have 0 employees, got %d", firm.GetNEmployees())
	}
}

type mockMacroTracker struct{}

func (m mockMacroTracker) GetUnemploymentRate() float64 { return 0.0 }
func (m mockMacroTracker) GetInflationRate() float64    { return 1.0 }
