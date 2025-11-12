package main

import (
	"fmt"
	"os"

	"github.com/go-delve/delve/service/api"
	"github.com/go-delve/delve/service/rpc2"
)

// This version halts the program first before querying state
func main() {
	fmt.Println("Delve API POC - Halt First, Then Query")
	fmt.Println("=" + string(make([]byte, 60)))

	// Connect using Delve's official RPC2 client
	fmt.Println("\nConnecting to Delve server at localhost:2345...")
	client := rpc2.NewClient("localhost:2345")
	defer client.Detach(true)

	fmt.Println("Connected successfully!")

	//  Check initial state
	fmt.Println("\n--- Checking if program is running ---")
	state, err := client.GetState()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting initial state: %v\n", err)
	} else {
		fmt.Printf("Program running: %v\n", state.Running)
		if state.Running {
			fmt.Println("Program is running, halting it now...")
			_, err = client.Halt()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error halting: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("Program halted successfully!")
		}
	}

	// Now get state again
	fmt.Println("\n--- TEST 1: Get Debugger State (after halt) ---")
	state, err = client.GetState()
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

	// Test 4: Set a breakpoint
	fmt.Println("\n--- TEST 4: Create Breakpoint ---")
	bp, err := client.CreateBreakpoint(&api.Breakpoint{
		File: "/src/cmd/payto/main.go",
		Line: 100,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating breakpoint: %v\n", err)
	} else {
		fmt.Printf("SUCCESS! Breakpoint created: [%d] %s:%d\n", bp.ID, bp.File, bp.Line)
	}

	// Continue execution
	fmt.Println("\n--- TEST 5: Continue Execution ---")
	contState := client.Continue()
	fmt.Printf("SUCCESS! Program resumed\n")
	_ = contState

	fmt.Println("\n" + string(make([]byte, 60)))
	fmt.Println("POC completed successfully!")
}
