package engine

/*
 * loop.go
 *
 * This file implements the simulation's main loop. All economic activities
 * occur in discrete units of time called ticks.
 * The engine's singleton struct is defined in this file too.
 */

type Engine struct {
	tick uint64
}
