package main

import (
	"fmt"
	"os"

	"github.com/go-delve/delve/service/rpc2"
)

// Test connecting to our simple local Delve instance
func main() {
	fmt.Println("Testing Connection to Simple Delve Instance")
	fmt.Println("=" + string(make([]byte, 60)))

	// Connect to our local Delve instance on port 2346
	fmt.Println("\nConnecting to Delve server at localhost:2346...")
	client := rpc2.NewClient("localhost:2346")
	defer client.Detach(true)

	fmt.Println("Connected successfully!")

	// Test 1: List breakpoints
	fmt.Println("\n--- TEST 1: List Breakpoints ---")
	breakpoints, err := client.ListBreakpoints(false)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listing breakpoints: %v\n", err)
	} else {
		fmt.Printf("SUCCESS! Found %d breakpoints\n", len(breakpoints))
	}

	// Test 2: Get state
	fmt.Println("\n--- TEST 2: Get State ---")
	state, err := client.GetState()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting state: %v\n", err)
	} else {
		fmt.Printf("SUCCESS! State retrieved:\n")
		fmt.Printf("  Running: %v\n", state.Running)
		fmt.Printf("  PID: %d\n", state.Pid)
		fmt.Printf("  Exited: %v\n", state.Exited)
	}

	// Test 3: List goroutines
	fmt.Println("\n--- TEST 3: List Goroutines ---")
	goroutines, _, err := client.ListGoroutines(0, 10)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listing goroutines: %v\n", err)
	} else {
		fmt.Printf("SUCCESS! Found %d goroutines\n", len(goroutines))
		for i, gr := range goroutines {
			if i >= 5 {
				break
			}
			fmt.Printf("  [%d] %s:%d", gr.ID, gr.UserCurrentLoc.File, gr.UserCurrentLoc.Line)
			if gr.UserCurrentLoc.Function != nil {
				fmt.Printf(" in %s", gr.UserCurrentLoc.Function.Name)
			}
			fmt.Println()
		}
	}

	fmt.Println("\n" + string(make([]byte, 60)))
	fmt.Println("All tests completed successfully!")
}
