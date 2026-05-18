package main

import "github.com/AtesIsf/monetary-simulator/internal/engine"

func main() {
	var sim engine.Simulation
	// 10 households, 2 firms, 0 banks
	sim.Populate(10, 2, 0)
	sim.Run(10)
}
