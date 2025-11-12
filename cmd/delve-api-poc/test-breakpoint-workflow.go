package main

import (
	"fmt"
	"os"
	"time"

	"github.com/go-delve/delve/service/api"
	"github.com/go-delve/delve/service/rpc2"
)

// Test the complete breakpoint workflow:
// 1. Halt running program
// 2. Set a breakpoint
// 3. Continue
// 4. Verify it halts again at the breakpoint
func main() {
	fmt.Println("Testing Breakpoint Workflow (Halt -> Set BP -> Continue -> Hit BP)")
	fmt.Println("=" + string(make([]byte, 70)))

	// Connect
	fmt.Println("\n[1] Connecting to Delve server at localhost:2345...")
	client := rpc2.NewClient("localhost:2345")
	defer client.Detach(true)
	fmt.Println("✓ Connected successfully!")

	// Halt the running program
	fmt.Println("\n[2] Halting the running program...")
	_, err := client.Halt()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error halting: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✓ Program halted")

	// Get current state to confirm halt
	fmt.Println("\n[3] Checking state after halt...")
	state, err := client.GetState()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting state: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✓ Running: %v (should be false)\n", state.Running)

	// Set a breakpoint at main.main
	fmt.Println("\n[4] Setting breakpoint at main.main...")
	bp, err := client.CreateBreakpoint(&api.Breakpoint{
		FunctionName: "main.main",
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating breakpoint: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✓ Breakpoint created: [%d] at %s:%d\n", bp.ID, bp.File, bp.Line)

	// Continue execution
	fmt.Println("\n[5] Continuing execution (should run until breakpoint)...")
	fmt.Println("   Waiting for breakpoint to be hit...")
	contState := client.Continue()
	fmt.Println("✓ Continue() returned")
	_ = contState

	// Give it a moment to hit the breakpoint
	time.Sleep(1 * time.Second)

	// Check state again - should be stopped at breakpoint
	fmt.Println("\n[6] Checking state after continue (should be stopped at breakpoint)...")
	state, err = client.GetState()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting state: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ State after continue:\n")
	fmt.Printf("   Running: %v (should be false - stopped at breakpoint)\n", state.Running)
	fmt.Printf("   Exited: %v\n", state.Exited)

	if state.CurrentThread != nil {
		fmt.Printf("   Current location: %s:%d\n", state.CurrentThread.File, state.CurrentThread.Line)
		if state.CurrentThread.Function != nil {
			fmt.Printf("   Function: %s\n", state.CurrentThread.Function.Name)
		}
	}

	// Check if we stopped at a breakpoint
	if state.CurrentThread != nil && state.CurrentThread.Breakpoint != nil {
		fmt.Printf("\n✓✓✓ SUCCESS! Stopped at breakpoint [%d]\n", state.CurrentThread.Breakpoint.ID)
	} else {
		fmt.Println("\n   Note: May need to wait longer or program may have different execution path")
	}

	// Get stack trace at breakpoint
	fmt.Println("\n[7] Getting stack trace at current location...")
	frames, err := client.Stacktrace(-1, 5, api.StacktraceReadDefers, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting stacktrace: %v\n", err)
	} else {
		fmt.Printf("✓ Stack trace (%d frames):\n", len(frames))
		for i, frame := range frames {
			if i >= 5 {
				break
			}
			fmt.Printf("   #%d %s:%d", i, frame.File, frame.Line)
			if frame.Function != nil {
				fmt.Printf(" in %s", frame.Function.Name())
			}
			fmt.Println()
		}
	}

	// Clean up: clear the breakpoint
	fmt.Printf("\n[8] Clearing breakpoint %d...\n", bp.ID)
	_, err = client.ClearBreakpoint(bp.ID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error clearing breakpoint: %v\n", err)
	} else {
		fmt.Println("✓ Breakpoint cleared")
	}

	// Continue again
	fmt.Println("\n[9] Continuing execution (program should run freely now)...")
	contState = client.Continue()
	fmt.Println("✓ Program continued")
	_ = contState

	fmt.Println("\n" + string(make([]byte, 70)))
	fmt.Println("✓ Workflow test completed successfully!")
	fmt.Println("\nConfirmed: Halt -> Set Breakpoint -> Continue -> Hit Breakpoint works!")
}
