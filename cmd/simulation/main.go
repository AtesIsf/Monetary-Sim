package main

import "github.com/AtesIsf/monetary-simulator/internal/engine"

func main() {
	var sim engine.Simulation
	sim.Populate(100, 5)
	defer sim.Close()

	sim.Run(30)
}
