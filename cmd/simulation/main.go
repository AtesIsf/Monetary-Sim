package main

import "github.com/AtesIsf/monetary-simulator/internal/engine"

func main() {
	var sim engine.Simulation

	sim.Populate(500, 160)
	defer sim.Close()

	// 4 -> 20 -> 2
	sim.Run(360)
	sim.UpdatePolicyRate(20)
	sim.Run(360)
	sim.UpdatePolicyRate(2)
	sim.Run(360)
}
