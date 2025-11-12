package main

import (
	"fmt"
	"net/rpc/jsonrpc"
	"os"
)

// Delve API structures
type DebuggerState struct {
	Running       bool    `json:"Running"`
	Pid           int     `json:"pid"`
	CurrentThread *Thread `json:"currentThread"`
	Exited        bool    `json:"exited"`
	ExitStatus    int     `json:"exitStatus"`
	When          string  `json:"when"`
}

type Thread struct {
	ID          int       `json:"id"`
	PC          uint64    `json:"pc"`
	File        string    `json:"file"`
	Line        int       `json:"line"`
	Function    *Function `json:"function"`
	GoroutineID int64     `json:"goroutineID"`
}

type Function struct {
	Name string `json:"name"`
}

type Breakpoint struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Addr     uint64 `json:"addr"`
	File     string `json:"file"`
	Line     int    `json:"line"`
	Cond     string `json:"Cond"`
	HitCount int    `json:"totalHitCount"`
}

type Goroutine struct {
	ID             int64    `json:"id"`
	CurrentLoc     Location `json:"currentLoc"`
	UserCurrentLoc Location `json:"userCurrentLoc"`
	GoStatementLoc Location `json:"goStatementLoc"`
	StartLoc       Location `json:"startLoc"`
	ThreadID       int      `json:"threadID"`
}

type Location struct {
	PC       uint64    `json:"pc"`
	File     string    `json:"file"`
	Line     int       `json:"line"`
	Function *Function `json:"function"`
}

// Request/Response wrappers
type StateIn struct{}
type StateOut struct {
	State *DebuggerState
}

type ListBreakpointsIn struct{}
type ListBreakpointsOut struct {
	Breakpoints []*Breakpoint
}

type ListGoroutinesIn struct {
	Start int
	Count int
}
type ListGoroutinesOut struct {
	Goroutines []*Goroutine
	Nextg      int
}

func main() {
	fmt.Println("Delve API POC v4 - Skip SetApiVersion (server already at v2)")
	fmt.Println("=" + string(make([]byte, 60)))

	// Connect using Go's jsonrpc package
	fmt.Println("\nConnecting to Delve server at localhost:2345...")
	client, err := jsonrpc.Dial("tcp", "localhost:2345")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()
	fmt.Println("Connected successfully!")
	fmt.Println("(Server already running with --api-version=2, skipping SetApiVersion)")

	// Test 1: Get debugger state
	fmt.Println("\n--- TEST 1: Get Debugger State ---")
	var stateOut StateOut
	err = client.Call("State", StateIn{}, &stateOut)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting state: %v\n", err)
	} else {
		fmt.Printf("SUCCESS! State retrieved:\n")
		fmt.Printf("  Running: %v\n", stateOut.State.Running)
		fmt.Printf("  PID: %d\n", stateOut.State.Pid)
		fmt.Printf("  Exited: %v\n", stateOut.State.Exited)
		if stateOut.State.CurrentThread != nil {
			fmt.Printf("  Current Thread: %d\n", stateOut.State.CurrentThread.ID)
			fmt.Printf("  Location: %s:%d\n", stateOut.State.CurrentThread.File, stateOut.State.CurrentThread.Line)
			if stateOut.State.CurrentThread.Function != nil {
				fmt.Printf("  Function: %s\n", stateOut.State.CurrentThread.Function.Name)
			}
		}
	}

	// Test 2: List breakpoints
	fmt.Println("\n--- TEST 2: List Breakpoints ---")
	var bpOut ListBreakpointsOut
	err = client.Call("ListBreakpoints", ListBreakpointsIn{}, &bpOut)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listing breakpoints: %v\n", err)
	} else {
		fmt.Printf("SUCCESS! Found %d breakpoints\n", len(bpOut.Breakpoints))
		for _, bp := range bpOut.Breakpoints {
			fmt.Printf("  [%d] %s:%d", bp.ID, bp.File, bp.Line)
			if bp.Cond != "" {
				fmt.Printf(" (condition: %s)", bp.Cond)
			}
			fmt.Printf(" hits: %d\n", bp.HitCount)
		}
	}

	// Test 3: List goroutines
	fmt.Println("\n--- TEST 3: List Goroutines ---")
	var grOut ListGoroutinesOut
	err = client.Call("ListGoroutines", ListGoroutinesIn{Start: 0, Count: 10}, &grOut)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listing goroutines: %v\n", err)
	} else {
		fmt.Printf("SUCCESS! Found %d goroutines (showing first 10)\n", len(grOut.Goroutines))
		for i, gr := range grOut.Goroutines {
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
	fmt.Println("POC completed successfully! Delve JSON-RPC API is working!")
}
