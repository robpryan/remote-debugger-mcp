package main

import (
	"fmt"
	"net/rpc/jsonrpc"
	"os"
	"time"
)

// Simpler approach - try calling State directly
func main() {
	fmt.Println("Delve API POC v3 - Simplified Test")
	fmt.Println("=" + string(make([]byte, 60)))

	// Connect using Go's jsonrpc package
	fmt.Println("\nConnecting to Delve server at localhost:2345...")
	conn, err := jsonrpc.Dial("tcp", "localhost:2345")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()
	fmt.Println("Connected successfully!")

	// Try calling State directly (might work without SetApiVersion)
	fmt.Println("\n--- Calling State ---")

	// Create channels for timeout
	done := make(chan bool)
	var result interface{}

	go func() {
		err = conn.Call("RPCServer.State", struct{}{}, &result)
		done <- true
	}()

	// Wait with timeout
	select {
	case <-done:
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error calling State: %v\n", err)
		} else {
			fmt.Printf("Success! Result type: %T\n", result)
			fmt.Printf("Result: %+v\n", result)
		}
	case <-time.After(3 * time.Second):
		fmt.Println("Timeout waiting for response (3s)")
		fmt.Println("\nThis suggests the server might be:")
		fmt.Println("1. Waiting for SetApiVersion first")
		fmt.Println("2. Using a different protocol format")
		fmt.Println("3. Not responding to JSON-RPC calls")
	}

	fmt.Println("\n" + string(make([]byte, 60)))
}
