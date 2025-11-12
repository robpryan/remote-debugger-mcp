package main

import (
	"fmt"
	"net/rpc/jsonrpc"
	"os"
)

// Manually create RPC client without going through rpc2.NewClient
// which tries to call SetApiVersion
func main() {
	fmt.Println("Delve API POC - Raw RPC Client (bypass NewClient)")
	fmt.Println("=" + string(make([]byte, 60)))

	// Connect directly with jsonrpc
	fmt.Println("\nConnecting to Delve server at localhost:2345...")
	conn, err := jsonrpc.Dial("tcp", "localhost:2345")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()
	fmt.Println("Connected successfully!")
	fmt.Println("(Using raw RPC client, not calling SetApiVersion)")

	// Manually call RPC methods using the raw client
	// Based on rpc2 package, the methods are on RPCServer

	// Test 1: List Breakpoints
	fmt.Println("\n--- TEST 1: List Breakpoints ---")
	type ListBreakpointsIn struct{}
	type Breakpoint struct {
		ID            int    `json:"id"`
		Name          string `json:"name"`
		Addr          uint64 `json:"addr"`
		File          string `json:"file"`
		Line          int    `json:"line"`
		Cond          string `json:"Cond"`
		TotalHitCount int    `json:"totalHitCount"`
	}
	type ListBreakpointsOut struct {
		Breakpoints []*Breakpoint
	}

	var bpIn ListBreakpointsIn
	var bpOut ListBreakpointsOut

	// Try "RPCServer.ListBreakpoints" first
	err = conn.Call("RPCServer.ListBreakpoints", bpIn, &bpOut)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error with RPCServer.ListBreakpoints: %v\n", err)

		// Try without prefix
		err = conn.Call("ListBreakpoints", bpIn, &bpOut)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error with ListBreakpoints: %v\n", err)
		} else {
			fmt.Printf("SUCCESS with 'ListBreakpoints'!\n")
		}
	} else {
		fmt.Printf("SUCCESS with 'RPCServer.ListBreakpoints'!\n")
	}

	if err == nil {
		fmt.Printf("Found %d breakpoints\n", len(bpOut.Breakpoints))
		for _, bp := range bpOut.Breakpoints {
			fmt.Printf("  [%d] %s:%d hits:%d\n", bp.ID, bp.File, bp.Line, bp.TotalHitCount)
		}
	}

	fmt.Println("\n" + string(make([]byte, 60)))
	fmt.Println("Test completed!")
}
