package engine

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"testing"

	"github.com/AtesIsf/monetary-simulator/internal/agents"
	"github.com/AtesIsf/monetary-simulator/internal/core"
)

func TestRecorder_InitCreatesFile(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "sim.csv")

	rec := InitRecorder(filePath, 5)
	defer rec.Close()

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Errorf("expected CSV file to be created at %s, but it was not", filePath)
	}

	if rec.GetFrequency() != 5 {
		t.Errorf("expected frequency to be 5, got %d", rec.GetFrequency())
	}
}

func TestRecorder_WriteAndClose(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "sim.csv")

	rec := InitRecorder(filePath, 2)

	var ld core.Ledger
	ld.Init()

	// Add some agents
	b := agents.NewBank(0)
	ld.AddToBalance(0, 1000)

	h := agents.NewHousehold(1)
	ld.AddToBalance(1, 100)
	ld.SetWageExpectation(1, 20) // seed wage expectation

	f := agents.NewFirm(2)
	ld.AddToBalance(2, 500)

	agentList := []core.Agent{b, h, f}

	rec.Write(agentList, 10, &ld)
	rec.Close()

	// Read file and verify content
	file, err := os.Open(filePath)
	if err != nil {
		t.Fatalf("failed to open generated CSV: %v", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("failed to read generated CSV: %v", err)
	}

	// Expected headers + 3 rows = 4 records total
	if len(records) != 4 {
		t.Errorf("expected 4 CSV records, got %d", len(records))
	}

	// Verify headers
	expectedHeaders := []string{"Tick", "Id", "Type", "Balance", "Employer", "Price", "WageExpectation"}
	for i, h := range expectedHeaders {
		if records[0][i] != h {
			t.Errorf("expected header %s at col %d, got %s", h, i, records[0][i])
		}
	}

	// Verify Bank row (id 0)
	bankRow := records[1]
	if bankRow[0] != "10" || bankRow[1] != "0" || bankRow[2] != "Bank" || bankRow[3] != "1000" || bankRow[6] != "0" {
		t.Errorf("unexpected bank row content: %v", bankRow)
	}

	// Verify Household row (id 1)
	houseRow := records[2]
	if houseRow[0] != "10" || houseRow[1] != "1" || houseRow[2] != "Household" || houseRow[3] != "100" || houseRow[6] != "20" {
		t.Errorf("unexpected household row content: %v", houseRow)
	}

	// Verify Firm row (id 2)
	firmRow := records[3]
	if firmRow[0] != "10" || firmRow[1] != "2" || firmRow[2] != "Firm" || firmRow[3] != "500" || firmRow[6] != "0" {
		t.Errorf("unexpected firm row content: %v", firmRow)
	}
}
