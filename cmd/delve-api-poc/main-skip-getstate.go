package main

import (
	"fmt"
	"os"

	"github.com/go-delve/delve/service/api"
	"github.com/go-delve/delve/service/rpc2"
)

// Skip GetState, try other commands directly
func main() {
	fmt.Println("Delve API POC - Skip GetState, Try Other Commands")
	fmt.Println("=" + string(make([]byte, 60)))

	// Connect using Delve's official RPC2 client
	fmt.Println("\nConnecting to Delve server at localhost:2345...")
	client := rpc2.NewClient("localhost:2345")
	defer client.Detach(true)

	fmt.Println("Connected successfully!")

	// Skip GetState, go straight to list breakpoints
	fmt.Println("\n--- TEST 1: List Breakpoints (without GetState) ---")
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

	// Try creating a breakpoint
	fmt.Println("\n--- TEST 2: Create Breakpoint ---")
	bp, err := client.CreateBreakpoint(&api.Breakpoint{
		File: "/src/cmd/payto/main.go",
		Line: 100,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating breakpoint: %v\n", err)
	} else {
		fmt.Printf("SUCCESS! Breakpoint created: [%d] %s:%d\n", bp.ID, bp.File, bp.Line)
	}

	// List functions
	fmt.Println("\n--- TEST 3: List Functions (main.*) ---")
	funcs, err := client.ListFunctions("main\\.", 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listing functions: %v\n", err)
	} else {
		fmt.Printf("SUCCESS! Found %d functions matching 'main.'\n", len(funcs))
		for i, fn := range funcs {
			if i >= 5 {
				fmt.Printf("  ... and %d more\n", len(funcs)-5)
				break
			}
			fmt.Printf("  %s\n", fn)
		}
	}

	// Try list goroutines
	fmt.Println("\n--- TEST 4: List Goroutines ---")
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

	// Now try GetState last
	fmt.Println("\n--- TEST 5: Get State (last test) ---")
	state, err := client.GetState()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting state: %v\n", err)
	} else {
		fmt.Printf("SUCCESS! State retrieved:\n")
		fmt.Printf("  Running: %v\n", state.Running)
		fmt.Printf("  PID: %d\n", state.Pid)
		fmt.Printf("  Exited: %v\n", state.Exited)
	}

	fmt.Println("\n" + string(make([]byte, 60)))
	fmt.Println("Tests completed!")
}
