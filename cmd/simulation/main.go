package main

import "github.com/AtesIsf/monetary-simulator/internal/engine"

func main() {
	var sim engine.Simulation
	sim.Populate(10, 2)
	sim.Run(3)
}
