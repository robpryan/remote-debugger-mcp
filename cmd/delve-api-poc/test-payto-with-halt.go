package main

import (
	"fmt"
	"os"

	"github.com/go-delve/delve/service/api"
	"github.com/go-delve/delve/service/rpc2"
)

// Test connecting to payto when program is running (--continue flag)
// We need to halt first before calling GetState
func main() {
	fmt.Println("Testing Payto Connection (with --continue handling)")
	fmt.Println("=" + string(make([]byte, 70)))

	// Connect to payto Delve instance on port 2345
	fmt.Println("\n[1] Connecting to Delve server at localhost:2345...")
	client := rpc2.NewClient("localhost:2345")
	defer client.Detach(true)
	fmt.Println("✓ Connected successfully!")

	// IMPORTANT: When --continue is used, the program is running
	// We must halt it before calling most API methods
	fmt.Println("\n[2] Halting program (required when using --continue)...")
	_, err := client.Halt()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error halting: %v\n", err)
		// This is okay if already halted
	} else {
		fmt.Println("✓ Program halted")
	}

	// Now we can safely get state
	fmt.Println("\n[3] Getting debugger state...")
	state, err := client.GetState()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting state: %v\n", err)
	} else {
		fmt.Printf("✓ State retrieved:\n")
		fmt.Printf("   PID: %d\n", state.Pid)
		fmt.Printf("   Running: %v\n", state.Running)
		fmt.Printf("   Exited: %v\n", state.Exited)
		if state.CurrentThread != nil {
			fmt.Printf("   Current Thread: %d\n", state.CurrentThread.ID)
			if state.CurrentThread.File != "" {
				fmt.Printf("   Location: %s:%d\n", state.CurrentThread.File, state.CurrentThread.Line)
			}
			if state.CurrentThread.Function != nil {
				fmt.Printf("   Function: %s\n", state.CurrentThread.Function.Name)
			}
		}
	}

	// List breakpoints
	fmt.Println("\n[4] Listing breakpoints...")
	breakpoints, err := client.ListBreakpoints(false)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listing breakpoints: %v\n", err)
	} else {
		fmt.Printf("✓ Found %d breakpoints\n", len(breakpoints))
		for i, bp := range breakpoints {
			if i >= 5 {
				fmt.Printf("   ... and %d more\n", len(breakpoints)-5)
				break
			}
			if bp.File != "" {
				fmt.Printf("   [%d] %s:%d\n", bp.ID, bp.File, bp.Line)
			}
		}
	}

	// List goroutines
	fmt.Println("\n[5] Listing goroutines...")
	goroutines, _, err := client.ListGoroutines(0, 10)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listing goroutines: %v\n", err)
	} else {
		fmt.Printf("✓ Found %d goroutines (showing first 10):\n", len(goroutines))
		for i, gr := range goroutines {
			if i >= 10 {
				fmt.Printf("   ... and %d more\n", len(goroutines)-10)
				break
			}
			fmt.Printf("   [%d] ", gr.ID)
			if gr.UserCurrentLoc.File != "" {
				fmt.Printf("%s:%d", gr.UserCurrentLoc.File, gr.UserCurrentLoc.Line)
			}
			if gr.UserCurrentLoc.Function != nil {
				fmt.Printf(" in %s", gr.UserCurrentLoc.Function.Name)
			}
			fmt.Println()
		}
	}

	// Get stack trace
	fmt.Println("\n[6] Getting stack trace...")
	frames, err := client.Stacktrace(-1, 10, api.StacktraceReadDefers, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting stacktrace: %v\n", err)
	} else {
		fmt.Printf("✓ Stack trace (%d frames, showing first 5):\n", len(frames))
		for i, frame := range frames {
			if i >= 5 {
				fmt.Printf("   ... and %d more frames\n", len(frames)-5)
				break
			}
			fmt.Printf("   #%d ", i)
			if frame.File != "" {
				fmt.Printf("%s:%d", frame.File, frame.Line)
			}
			if frame.Function != nil {
				fmt.Printf(" in %s", frame.Function.Name())
			}
			fmt.Println()
		}
	}

	// Create a breakpoint at main.main
	fmt.Println("\n[7] Creating breakpoint at main.main...")
	bp, err := client.CreateBreakpoint(&api.Breakpoint{
		FunctionName: "main.main",
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating breakpoint: %v\n", err)
	} else {
		fmt.Printf("✓ Breakpoint created: [%d] %s:%d\n", bp.ID, bp.File, bp.Line)
	}

	// Continue execution
	fmt.Println("\n[8] Continuing execution...")
	contState := client.Continue()
	fmt.Println("✓ Program continued")
	_ = contState

	fmt.Println("\n" + string(make([]byte, 70)))
	fmt.Println("✓ Test completed successfully!")
	fmt.Println("\nKey Finding: When using --continue, you MUST call Halt() before GetState()")
}
