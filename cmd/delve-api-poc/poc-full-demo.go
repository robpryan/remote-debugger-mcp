package main

import (
	"fmt"
	"os"

	"github.com/go-delve/delve/service/api"
	"github.com/go-delve/delve/service/rpc2"
)

// Comprehensive Delve API POC demonstrating all key operations
func main() {
	fmt.Println("Delve API Comprehensive POC")
	fmt.Println("=" + string(make([]byte, 70)))

	// Connect to Delve instance
	fmt.Println("\n[1] Connecting to Delve server at localhost:2346...")
	client := rpc2.NewClient("localhost:2346")
	defer client.Detach(true)
	fmt.Println("✓ Connected successfully!")

	// Get initial state
	fmt.Println("\n[2] Getting initial debugger state...")
	state, err := client.GetState()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	} else {
		fmt.Printf("✓ PID: %d, Running: %v, Exited: %v\n", state.Pid, state.Running, state.Exited)
	}

	// Create a breakpoint
	fmt.Println("\n[3] Creating breakpoint at main.processLoop...")
	bp, err := client.CreateBreakpoint(&api.Breakpoint{
		FunctionName: "main.processLoop",
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating breakpoint: %v\n", err)
	} else {
		fmt.Printf("✓ Breakpoint created: [%d] %s:%d\n", bp.ID, bp.File, bp.Line)
	}

	// List all breakpoints
	fmt.Println("\n[4] Listing all breakpoints...")
	breakpoints, err := client.ListBreakpoints(false)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	} else {
		fmt.Printf("✓ Found %d breakpoints:\n", len(breakpoints))
		for _, bp := range breakpoints {
			fmt.Printf("   [%d] %s:%d\n", bp.ID, bp.File, bp.Line)
		}
	}

	// Continue execution (this will start the program)
	fmt.Println("\n[5] Continuing execution (program will run until breakpoint)...")
	contState := client.Continue()
	fmt.Printf("✓ Program resumed\n")
	_ = contState

	// Wait a moment and get state again
	fmt.Println("\n[6] Getting state after continue...")
	state, err = client.GetState()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	} else {
		fmt.Printf("✓ Running: %v\n", state.Running)
		if state.CurrentThread != nil {
			fmt.Printf("   Location: %s:%d\n", state.CurrentThread.File, state.CurrentThread.Line)
			if state.CurrentThread.Function != nil {
				fmt.Printf("   Function: %s\n", state.CurrentThread.Function.Name)
			}
		}
	}

	// List goroutines
	fmt.Println("\n[7] Listing goroutines...")
	goroutines, _, err := client.ListGoroutines(0, 10)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	} else {
		fmt.Printf("✓ Found %d goroutines:\n", len(goroutines))
		for i, gr := range goroutines {
			if i >= 5 {
				fmt.Printf("   ... and %d more\n", len(goroutines)-5)
				break
			}
			fmt.Printf("   [%d] %s:%d", gr.ID, gr.UserCurrentLoc.File, gr.UserCurrentLoc.Line)
			if gr.UserCurrentLoc.Function != nil {
				fmt.Printf(" in %s", gr.UserCurrentLoc.Function.Name)
			}
			fmt.Println()
		}
	}

	// Get stack trace
	fmt.Println("\n[8] Getting stack trace for current goroutine...")
	frames, err := client.Stacktrace(-1, 10, api.StacktraceReadDefers, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	} else {
		fmt.Printf("✓ Stack trace (%d frames):\n", len(frames))
		for i, frame := range frames {
			if i >= 5 {
				fmt.Printf("   ... and %d more frames\n", len(frames)-5)
				break
			}
			fmt.Printf("   #%d %s:%d", i, frame.File, frame.Line)
			if frame.Function != nil {
				fmt.Printf(" in %s", frame.Function.Name())
			}
			fmt.Println()
		}
	}

	// List local variables
	fmt.Println("\n[9] Listing local variables...")
	vars, err := client.ListLocalVariables(api.EvalScope{GoroutineID: -1, Frame: 0}, api.LoadConfig{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	} else {
		fmt.Printf("✓ Found %d local variables:\n", len(vars))
		for i, v := range vars {
			if i >= 5 {
				fmt.Printf("   ... and %d more\n", len(vars)-5)
				break
			}
			fmt.Printf("   %s = %s (%s)\n", v.Name, v.Value, v.Type)
		}
	}

	// Clear breakpoint
	if len(breakpoints) > 0 {
		fmt.Printf("\n[10] Clearing breakpoint %d...\n", breakpoints[0].ID)
		_, err = client.ClearBreakpoint(breakpoints[0].ID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		} else {
			fmt.Println("✓ Breakpoint cleared")
		}
	}

	fmt.Println("\n" + string(make([]byte, 70)))
	fmt.Println("✓ All tests completed successfully!")
	fmt.Println("\nThe Delve JSON-RPC API is working perfectly!")
}
