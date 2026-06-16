package core

import (
	"sync"
	"testing"
)

func TestLedger_Init(t *testing.T) {
	var ld Ledger
	ld.Init()
	if ld.balances == nil {
		t.Error("balances map was not initialized")
	}
	if ld.wageExpectations == nil {
		t.Error("wageExpectations map was not initialized")
	}
}

func TestLedger_GetBalance_Default(t *testing.T) {
	var ld Ledger
	ld.Init()
	if balance := ld.GetBalance(999); balance != 0 {
		t.Errorf("expected balance of unseeded agent to be 0, got %d", balance)
	}
}

func TestLedger_AddToBalance(t *testing.T) {
	tests := []struct {
		name    string
		id      uint32
		adds    []int64
		wantBal int64
	}{
		{
			name:    "single add",
			id:      1,
			adds:    []int64{100},
			wantBal: 100,
		},
		{
			name:    "multiple adds",
			id:      2,
			adds:    []int64{100, 200, -50},
			wantBal: 250,
		},
		{
			name:    "negative starting add",
			id:      3,
			adds:    []int64{-100},
			wantBal: -100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ld Ledger
			ld.Init()
			for _, add := range tt.adds {
				ld.AddToBalance(tt.id, add)
			}
			if got := ld.GetBalance(tt.id); got != tt.wantBal {
				t.Errorf("GetBalance() = %d, want %d", got, tt.wantBal)
			}
		})
	}
}

func TestLedger_Transfer(t *testing.T) {
	tests := []struct {
		name        string
		from        uint32
		to          uint32
		fromStart   int64
		toStart     int64
		amount      int64
		wantFromEnd int64
		wantToEnd   int64
	}{
		{
			name:        "simple transfer",
			from:        1,
			to:          2,
			fromStart:   100,
			toStart:     50,
			amount:      30,
			wantFromEnd: 70,
			wantToEnd:   80,
		},
		{
			name:        "transfer to self",
			from:        1,
			to:          1,
			fromStart:   100,
			toStart:     100,
			amount:      30,
			wantFromEnd: 100,
			wantToEnd:   100,
		},
		{
			name:        "negative amount transfer",
			from:        1,
			to:          2,
			fromStart:   100,
			toStart:     50,
			amount:      -30,
			wantFromEnd: 130,
			wantToEnd:   20,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ld Ledger
			ld.Init()
			ld.AddToBalance(tt.from, tt.fromStart)
			if tt.from != tt.to {
				ld.AddToBalance(tt.to, tt.toStart)
			}
			_ = ld.Transfer(tt.from, tt.to, tt.amount)
			if got := ld.GetBalance(tt.from); got != tt.wantFromEnd {
				t.Errorf("from GetBalance() = %d, want %d", got, tt.wantFromEnd)
			}
			if got := ld.GetBalance(tt.to); got != tt.wantToEnd {
				t.Errorf("to GetBalance() = %d, want %d", got, tt.wantToEnd)
			}
		})
	}
}

func TestLedger_Transfer_DisallowsNegativeBalance(t *testing.T) {
	var ld Ledger
	ld.Init()
	ld.AddToBalance(1, 10)
	ld.AddToBalance(2, 20)
	err := ld.Transfer(1, 2, 50)
	if err == nil {
		t.Error("expected error due to insufficient balance, got nil")
	}
	if bal1 := ld.GetBalance(1); bal1 != 10 {
		t.Errorf("expected balance of 1 to remain 10, got %d", bal1)
	}
	if bal2 := ld.GetBalance(2); bal2 != 20 {
		t.Errorf("expected balance of 2 to remain 20, got %d", bal2)
	}
}

func TestLedger_WageExpectation(t *testing.T) {
	var ld Ledger
	ld.Init()

	if defaultWage := ld.GetWageExpectation(1); defaultWage != 0 {
		t.Errorf("expected default wage expectation to be 0, got %d", defaultWage)
	}

	ld.SetWageExpectation(1, 25)
	if wage := ld.GetWageExpectation(1); wage != 25 {
		t.Errorf("expected wage expectation to be 25, got %d", wage)
	}

	ld.SetWageExpectation(1, 30)
	if wage := ld.GetWageExpectation(1); wage != 30 {
		t.Errorf("expected updated wage expectation to be 30, got %d", wage)
	}
}

func TestLedger_ConcurrentAccess(t *testing.T) {
	var ld Ledger
	ld.Init()

	var wg sync.WaitGroup
	workers := 100
	ops := 100

	wg.Add(workers)
	for i := range workers {
		go func(id uint32) {
			defer wg.Done()
			for j := range ops {
				ld.AddToBalance(id, 1)
				ld.Transfer(id, (id + 1)%uint32(workers), 1)
				ld.SetWageExpectation(id, int64(j))
				_ = ld.GetWageExpectation(id)
			}
		} (uint32(i))
	}
	wg.Wait()

	// Total money added should be equal to total balances if we just check sums
	// AddToBalance puts workers * ops into the ledger.
	// Transfer is zero-sum, so total sum of balances should be workers * ops.
	var sum int64
	for i := range workers {
		sum += ld.GetBalance(uint32(i))
	}
	expectedSum := int64(workers * ops)
	if sum != expectedSum {
		t.Errorf("expected total balance sum %d, got %d", expectedSum, sum)
	}
}
