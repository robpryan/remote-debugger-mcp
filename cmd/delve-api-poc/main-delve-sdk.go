package main

import (
	"fmt"
	"os"

	"github.com/go-delve/delve/service/rpc2"
)

// This version uses Delve's official RPC2 client SDK
func main() {
	fmt.Println("Delve API POC - Using Official Delve SDK")
	fmt.Println("=" + string(make([]byte, 60)))

	// Connect using Delve's official RPC2 client
	fmt.Println("\nConnecting to Delve server at localhost:2345...")
	client := rpc2.NewClient("localhost:2345")
	defer client.Detach(true)

	fmt.Println("Connected successfully!")

	// Test 1: Get debugger state
	fmt.Println("\n--- TEST 1: Get Debugger State ---")
	state, err := client.GetState()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting state: %v\n", err)
	} else {
		fmt.Printf("SUCCESS! State retrieved:\n")
		fmt.Printf("  Running: %v\n", state.Running)
		fmt.Printf("  PID: %d\n", state.Pid)
		fmt.Printf("  Exited: %v\n", state.Exited)
		if state.CurrentThread != nil {
			fmt.Printf("  Current Thread: %d\n", state.CurrentThread.ID)
			fmt.Printf("  Location: %s:%d\n", state.CurrentThread.File, state.CurrentThread.Line)
			if state.CurrentThread.Function != nil {
				fmt.Printf("  Function: %s\n", state.CurrentThread.Function.Name)
			}
		}
	}

	// Test 2: List breakpoints
	fmt.Println("\n--- TEST 2: List Breakpoints ---")
	breakpoints, err := client.ListBreakpoints(false)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listing breakpoints: %v\n", err)
	} else {
		fmt.Printf("SUCCESS! Found %d breakpoints\n", len(breakpoints))
		for _, bp := range breakpoints {
			fmt.Printf("  [%d] %s:%d", bp.ID, bp.File, bp.Line)
			if bp.Cond != "" {
				fmt.Printf(" (condition: %s)", bp.Cond)
			}
			fmt.Printf(" hits: %d\n", bp.TotalHitCount)
		}
	}

	// Test 3: List goroutines
	fmt.Println("\n--- TEST 3: List Goroutines ---")
	goroutines, _, err := client.ListGoroutines(0, 10)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listing goroutines: %v\n", err)
	} else {
		fmt.Printf("SUCCESS! Found %d goroutines (showing first 10)\n", len(goroutines))
		for i, gr := range goroutines {
			if i >= 10 {
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
	fmt.Println("POC completed successfully!")
}
