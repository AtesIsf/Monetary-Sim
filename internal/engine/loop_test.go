package engine

import (
	"sync"
	"testing"

	"github.com/AtesIsf/monetary-simulator/internal/agents"
	"github.com/AtesIsf/monetary-simulator/internal/core"
)

func TestRun_SingleTick(t *testing.T) {
	var sim Simulation
	sim.Populate(5, 2)
	defer sim.Close()

	sim.Run(1)

	if sim.tick != 1 {
		t.Errorf("expected tick to be 1, got %d", sim.tick)
	}

	if len(sim.priceHistory) != 1 {
		t.Errorf("expected price history length 1, got %d", len(sim.priceHistory))
	}
}

func TestRun_HireAndPayCycle(t *testing.T) {
	var sim Simulation
	sim.Populate(3, 1)
	defer sim.Close()

	// Set tick to 1 to avoid tick%12 == 0 yearly resets
	sim.tick = 1

	var firm *agents.Firm
	for _, ag := range sim.agents {
		if f, ok := ag.(*agents.Firm); ok {
			firm = f
			break
		}
	}
	if firm == nil {
		t.Fatal("expected a firm in simulation")
	}

	for firm.GetNEmployees() > 0 {
		firm.PopEmployee()
	}

	firm.PerformSale(0)

	for _, ag := range sim.agents {
		if h, ok := ag.(*agents.Household); ok {
			h.SetEmployer(h.GetId())
		}
	}

	sim.Run(1)

	if firm.GetNEmployees() == 0 {
		t.Error("expected firm to have hired a worker")
	}

	var hiredHousehold *agents.Household
	for _, ag := range sim.agents {
		if h, ok := ag.(*agents.Household); ok {
			if h.GetEmployer() == firm.GetId() {
				hiredHousehold = h
				break
			}
		}
	}

	if hiredHousehold == nil {
		t.Fatal("expected one household to be hired")
	}

	bal := sim.ld.GetBalance(hiredHousehold.GetId())
	if bal == 100 {
		t.Errorf("expected hired household ledger balance to change, got %d", bal)
	}
}

func TestRun_FireCycle(t *testing.T) {
	var sim Simulation
	sim.Populate(2, 1)
	defer sim.Close()

	// Set tick to 1 to avoid yearly resets
	sim.tick = 1

	var firm *agents.Firm
	for _, ag := range sim.agents {
		if f, ok := ag.(*agents.Firm); ok {
			firm = f
			break
		}
	}
	if firm == nil {
		t.Fatal("expected a firm")
	}

	var house *agents.Household
	for _, ag := range sim.agents {
		if h, ok := ag.(*agents.Household); ok {
			house = h
			break
		}
	}
	if house == nil {
		t.Fatal("expected a household")
	}

	firm.AddEmployee(house.GetId())
	house.SetEmployer(firm.GetId())

	firm.PerformSale(1000)

	sim.Run(1)

	if firm.GetNEmployees() != 0 {
		t.Errorf("expected employee count to be 0, got %d", firm.GetNEmployees())
	}
	if house.IsEmployed() {
		t.Error("expected house to be unemployed")
	}
}

func TestRun_ConsumeCycle(t *testing.T) {
	var sim Simulation
	sim.Populate(1, 1)
	defer sim.Close()

	// Set tick to 1 to avoid yearly resets
	sim.tick = 1

	var house *agents.Household
	var firm *agents.Firm
	for _, ag := range sim.agents {
		if h, ok := ag.(*agents.Household); ok {
			house = h
		}
		if f, ok := ag.(*agents.Firm); ok {
			firm = f
		}
	}

	sim.ld.AddToBalance(house.GetId(), 1000)
	// Clear firm balance and set to 50 so it is below MaxLiquidity and doesn't deposit extra cash
	sim.ld.Transfer(firm.GetId(), sim.bank.GetId(), sim.ld.GetBalance(firm.GetId()))
	sim.ld.AddToBalance(firm.GetId(), 50)
	firmInitialBal := int64(50)
	firm.AddInventory(10)

	sim.Run(1)

	if sim.ld.GetBalance(firm.GetId()) <= firmInitialBal {
		t.Error("expected firm ledger balance to increase after consumption")
	}
}

func TestRun_DrawSavings_NoConsumption(t *testing.T) {
	var sim Simulation
	sim.Populate(1, 0)
	defer sim.Close()

	// Set tick to 1 to avoid yearly resets
	sim.tick = 1

	var house *agents.Household
	for _, ag := range sim.agents {
		if h, ok := ag.(*agents.Household); ok {
			house = h
			break
		}
	}

	sim.ld.AddToBalance(house.GetId(), 500)
	sim.bank.AddDemandDeposit(house.GetId(), 200, &sim.ld)
	house.SetSavings(200)
	sim.ld.Transfer(house.GetId(), sim.bank.GetId(), sim.ld.GetBalance(house.GetId())) // house cash becomes 0

	sim.Run(1)

	expectedMaxLiq := int64(core.MaxLiquidity * (100 - sim.pol.GetInterestRate()) / 100)
	expectedDeposits := int64(200) - expectedMaxLiq

	if got := sim.ld.GetBalance(house.GetId()); got != expectedMaxLiq {
		t.Errorf("expected household ledger balance to be %d, got %d", expectedMaxLiq, got)
	}
	if got := sim.bank.QueryDeposits(house.GetId()); got != expectedDeposits {
		t.Errorf("expected bank deposits to be %d, got %d", expectedDeposits, got)
	}
}

func findNothingFirm(id uint32) *agents.Firm {
	f := agents.NewFirm(id)
	f.AddEmployee(999)
	var pol core.Policies
	pol.Populate()
	var ld core.Ledger
	ld.Init()
	ld.AddToBalance(id, 10000) // seed rich balance to pass funding check
	for range 1000 {
		f.PerformSale(1)
		ret := f.Update(&pol, &ld, mockMacroTracker{}, 1)
		if ret.Action == core.Nothing {
			return f
		}
	}
	return nil
}

func TestRun_FirmNothing_WagesNotPaid(t *testing.T) {
	var sim Simulation
	sim.Populate(0, 0)
	sim.ld.Init()
	sim.pol.Populate()
	sim.tick = 1

	firm := findNothingFirm(2)
	if firm == nil {
		t.Fatal("could not find a firm returning Nothing")
	}
	for firm.GetNEmployees() > 0 {
		firm.PopEmployee()
	}

	house := agents.NewHousehold(1)
	firm.AddEmployee(house.GetId())
	house.SetEmployer(firm.GetId())

	sim.ld.AddToBalance(firm.GetId(), 50)
	sim.ld.SetWageExpectation(house.GetId(), 20)

	sim.agentsMutex.Lock()
	sim.agents = append(sim.agents, house, firm)
	sim.agentsMutex.Unlock()

	initialHouseBal := sim.ld.GetBalance(house.GetId())

	sim.Run(1)

	if got := sim.ld.GetBalance(house.GetId()); got > initialHouseBal {
		t.Errorf("expected household balance not to increase, got initial %d -> final %d", initialHouseBal, got)
	}
}

func TestRun_RequestLoan_FirmGetsLoan(t *testing.T) {
	var sim Simulation
	sim.Populate(0, 0)
	sim.ld.Init()
	sim.pol.Populate()
	sim.tick = 1

	sim.bank = *agents.NewBank(0)
	sim.ld.AddToBalance(0, 10000)

	firm := agents.NewFirm(1)
	firm.SetSavings(0)
	sim.ld.AddToBalance(1, -100)
	firm.AddEmployee(999)
	sim.ld.SetWageExpectation(999, 500)

	sim.agentsMutex.Lock()
	sim.agents = append(sim.agents, &sim.bank, firm)
	sim.agentsMutex.Unlock()

	initialBankBal := sim.ld.GetBalance(0)

	sim.Run(1)

	if got := sim.ld.GetBalance(0); got >= initialBankBal {
		t.Errorf("expected bank balance to decrease after loan, got initial %d, final %d", initialBankBal, got)
	}
}

func TestRun_RequestLoan_BankBroke(t *testing.T) {
	var sim Simulation
	sim.Populate(0, 0)
	sim.ld.Init()
	sim.pol.Populate()
	sim.tick = 1

	sim.bank = *agents.NewBank(0)
	sim.ld.AddToBalance(0, 50)

	firm := agents.NewFirm(1)
	firm.SetSavings(0)
	sim.ld.AddToBalance(1, -100)
	firm.AddEmployee(999)
	sim.ld.SetWageExpectation(999, 500)

	sim.agentsMutex.Lock()
	sim.agents = append(sim.agents, &sim.bank, firm)
	sim.agentsMutex.Unlock()

	initialBankBal := sim.ld.GetBalance(0)

	sim.Run(1)

	if got := sim.ld.GetBalance(0); got != initialBankBal {
		t.Errorf("expected bank balance to remain same, got initial %d, final %d", initialBankBal, got)
	}
}

func TestRun_DepositExtra(t *testing.T) {
	var sim Simulation
	sim.Populate(0, 0)
	sim.ld.Init()
	sim.pol.Populate()
	sim.tick = 0

	sim.bank = *agents.NewBank(0)
	sim.ld.AddToBalance(0, 10000)

	house := agents.NewHousehold(1)
	sim.ld.AddToBalance(1, 1000)

	sim.agentsMutex.Lock()
	sim.agents = append(sim.agents, &sim.bank, house)
	sim.agentsMutex.Unlock()

	sim.Run(1)

	// After depositing excess, household balance should be near MaxLiquidity.
	// It may be slightly above due to deposit interest paid back in the same
	// concurrent tick (bank.Update transfers interest to depositor's ledger).
	got := sim.ld.GetBalance(house.GetId())
	if got > core.MaxLiquidity + 50 {
		t.Errorf("expected household balance near core.MaxLiquidity, got %d", got)
	}
	if got := sim.bank.QueryDeposits(house.GetId()); got <= 0 {
		t.Errorf("expected demand deposits, got %d", got)
	}
}

func TestGetUnemploymentRate(t *testing.T) {
	var sim Simulation
	sim.Populate(4, 1)
	defer sim.Close()

	rate := sim.GetUnemploymentRate()
	if rate != 1.0 {
		t.Errorf("expected unemployment rate 1.0, got %f", rate)
	}

	count := 0
	for _, ag := range sim.agents {
		if h, ok := ag.(*agents.Household); ok {
			if count%2 == 0 {
				h.SetEmployer(5)
			}
			count++
		}
	}

	rate = sim.GetUnemploymentRate()
	if rate != 0.5 {
		t.Errorf("expected unemployment rate 0.5, got %f", rate)
	}
}

func TestGetInflationRate_Before12Ticks(t *testing.T) {
	var sim Simulation
	sim.Populate(1, 1)
	defer sim.Close()

	rate := sim.GetInflationRate()
	if rate != 1.0 {
		t.Errorf("expected 1.0, got %f", rate)
	}
}

func TestGetInflationRate_After12Ticks(t *testing.T) {
	var sim Simulation
	sim.Populate(1, 1)
	defer sim.Close()

	sim.priceHistory = []float64{10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 15}
	rate := sim.GetInflationRate()
	if rate != 1.5 {
		t.Errorf("expected 1.5, got %f", rate)
	}
}

func TestSyncPass(t *testing.T) {
	var sim Simulation
	sim.Populate(0, 0)
	sim.ld.Init()
	sim.pol.Populate()
	sim.tick = 1 // avoid yearly resets on tick 0

	sim.bank = *agents.NewBank(0)

	house := agents.NewHousehold(1)
	sim.ld.AddToBalance(1, 500)
	sim.ld.SetWageExpectation(1, 0)

	sim.bank.AddDemandDeposit(1, 50, &sim.ld)
	sim.ld.Transfer(1, 999, sim.ld.GetBalance(1)) // transfer remaining household cash to 999
	sim.ld.Transfer(0, 999, sim.ld.GetBalance(0)) // set bank balance to exactly 0 to force loan failure

	sim.agentsMutex.Lock()
	sim.agents = append(sim.agents, &sim.bank, house)
	sim.agentsMutex.Unlock()

	// Run 2 ticks. Tick 1 syncs the balance. Tick 2 draws it because balance is 0 and savings is 50.
	sim.Run(2)

	if got := sim.bank.QueryDeposits(1); got != 0 {
		t.Errorf("expected deposits to be drawn to 0 after sync pass and DrawSavings, got %d", got)
	}
}

func TestRace_FullSimulation(t *testing.T) {
	var sim Simulation
	sim.Populate(10, 3)
	defer sim.Close()

	sim.Run(24)
}

func TestRace_ConcurrentHiring(t *testing.T) {
	var sim Simulation
	sim.Populate(20, 5)
	defer sim.Close()

	var firms []*agents.Firm
	for _, ag := range sim.agents {
		if f, ok := ag.(*agents.Firm); ok {
			firms = append(firms, f)
		}
	}

	var wg sync.WaitGroup
	for _, firm := range firms {
		wg.Add(1)
		go func(f *agents.Firm) {
			defer wg.Done()
			for range 5 {
				sim.matcher.HireWorker(f)
			}
		}(firm)
	}
	wg.Wait()
}

func TestRace_ConcurrentBuyGoods(t *testing.T) {
	var sim Simulation
	sim.Populate(20, 5)
	defer sim.Close()

	var households []*agents.Household
	for _, ag := range sim.agents {
		if h, ok := ag.(*agents.Household); ok {
			households = append(households, h)
		}
	}

	var wg sync.WaitGroup
	for _, house := range households {
		wg.Add(1)
		go func(h *agents.Household) {
			defer wg.Done()
			for range 5 {
				sim.matcher.BuyGoods(h, &sim.ld)
			}
		}(house)
	}
	wg.Wait()
}

func TestRace_ConcurrentPayWages(t *testing.T) {
	var sim Simulation
	sim.Populate(20, 5)
	defer sim.Close()

	var firms []*agents.Firm
	for _, ag := range sim.agents {
		if f, ok := ag.(*agents.Firm); ok {
			firms = append(firms, f)
		}
	}

	var wg sync.WaitGroup
	for _, firm := range firms {
		wg.Add(1)
		go func(f *agents.Firm) {
			defer wg.Done()
			for range 5 {
				f.PayWages(&sim.ld)
			}
		}(firm)
	}
	wg.Wait()
}

func TestRace_ConcurrentLoanIssuance(t *testing.T) {
	var sim Simulation
	sim.Populate(20, 5)
	defer sim.Close()

	var wg sync.WaitGroup
	for i := range 10 {
		wg.Add(1)
		go func(id uint32) {
			defer wg.Done()
			target := core.AgentId{AType: core.Household, Id: id}
			_ = sim.bank.IssueLoan(&sim.ld, target, 10, 5)
		}(uint32(i))
	}
	wg.Wait()
}

func TestRace_ConcurrentDepositAndWithdraw(t *testing.T) {
	var sim Simulation
	sim.Populate(10, 2)
	defer sim.Close()

	var wg sync.WaitGroup
	wg.Go(func() {
		for range 5 {
			sim.bank.AddDemandDeposit(1, 10, &sim.ld)
		}
	})

	wg.Go(func() {
		for range 50 {
			sim.bank.WithdrawAll(1, &sim.ld)
		}
	})

	wg.Wait()
}
