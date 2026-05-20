package engine

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"os"

	"github.com/AtesIsf/monetary-simulator/internal/agents"
	"github.com/AtesIsf/monetary-simulator/internal/core"
)

/*
 * recorder.go
 *
 * Records the current state of the simulation to files on each tick.
 */

type Recorder struct {
	file *os.File
	writer *csv.Writer
	frequency uint64 // 1 write per how many ticks
}

func InitRecorder(dest string, frequency uint64) Recorder {
	file, _ := os.Create(dest)
	bufWriter := bufio.NewWriterSize(file, 4 * 1024 * 1024) // 4MB
	writer := csv.NewWriter(bufWriter)
	writer.Write([]string{ "Tick", "Id", "Type", "Balance", "Employer" })

	return Recorder{ file: file, writer: writer, frequency: frequency }
}

func (r *Recorder) GetFrequency() uint64 {
	return r.frequency
}

func (r *Recorder) Close() {
	r.writer.Flush()
	r.file.Close()
}

func (r *Recorder) Write(arr []core.Agent, tick uint64, ld *core.Ledger) {
	for _, ag := range arr {

		// Tick, Id, Type, Balance, Employer
		line := make([]string, 5)
		line[0] = fmt.Sprintf("%d", tick)
		line[1] = fmt.Sprintf("%d", ag.GetId())
		line[3] = fmt.Sprintf("%d", ld.GetBalance(ag.GetId()))

		switch ag.GetType() {
		case core.Household:
			house, _ := ag.(*agents.Household)
			line[2] = "Household"
			line[4] = fmt.Sprintf("%d", house.GetEmployer())

		case core.Firm:
			firm, _ := ag.(*agents.Firm)
			line[2] = "Firm"
			line[4] = fmt.Sprintf("%d", firm.GetId())

		case core.Bank:
			bank, _ := ag.(*agents.Bank)
			line[2] = "Bank"
			line[4] = fmt.Sprintf("%d", bank.GetId())

		}

		r.writer.Write(line)
	}
}

