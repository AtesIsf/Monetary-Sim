package main

import (
	"fmt"
	"github.com/AtesIsf/monetary-simulator/internal/engine"
	"github.com/AtesIsf/monetary-simulator/internal/agents"
)

func main() {
	eng := engine.Engine { }
	agent := agents.Agent { }

	fmt.Println("Hello, World!")
	fmt.Printf("Engine: %p\n", &eng)
	fmt.Printf("Agent: %p\n", &agent)
}
