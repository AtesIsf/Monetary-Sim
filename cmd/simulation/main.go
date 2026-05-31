package main

import "github.com/AtesIsf/monetary-simulator/internal/engine"

func main() {
	var sim engine.Simulation
	/*
	In the US, there are approximately 33.2M businesses and 350M people.
	So, approximately using the ratio 1/10 firms to households should be ok
	*/
	sim.Populate(1000, 100)
	defer sim.Close()

	sim.Run(300)
}
