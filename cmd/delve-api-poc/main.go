package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
)

// JSON-RPC request structure
type JSONRPCRequest struct {
	Method string        `json:"method"`
	Params []interface{} `json:"params"`
	ID     int64         `json:"id"`
}

// JSON-RPC response structure
type JSONRPCResponse struct {
	ID     int64           `json:"id"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  json.RawMessage `json:"error,omitempty"`
}

// DelveClient manages connection to Delve JSON-RPC server
type DelveClient struct {
	conn      net.Conn
	requestID int64
	encoder   *json.Encoder
	decoder   *json.Decoder
}

// Connect to Delve server
func Connect(host string, port int) (*DelveClient, error) {
	addr := fmt.Sprintf("%s:%d", host, port)
	fmt.Printf("Connecting to Delve server at %s...\n", addr)

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	fmt.Println("Connected successfully!")

	return &DelveClient{
		conn:      conn,
		requestID: 0,
		encoder:   json.NewEncoder(conn),
		decoder:   json.NewDecoder(conn),
	}, nil
}

// Call makes a JSON-RPC call
func (c *DelveClient) Call(method string, params interface{}) (json.RawMessage, error) {
	c.requestID++

	request := JSONRPCRequest{
		Method: method,
		Params: []interface{}{params},
		ID:     c.requestID,
	}

	fmt.Printf("\n==> Calling %s (ID: %d)\n", method, c.requestID)
	fmt.Printf("    Params: %+v\n", params)

	// Send request
	if err := c.encoder.Encode(request); err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	// Read response
	var response JSONRPCResponse
	if err := c.decoder.Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if len(response.Error) > 0 {
		// Try to parse as string first
		var errorStr string
		if err := json.Unmarshal(response.Error, &errorStr); err == nil {
			return nil, fmt.Errorf("RPC error: %s", errorStr)
		}
		// Fall back to full error object
		return nil, fmt.Errorf("RPC error: %s", string(response.Error))
	}

	fmt.Printf("<== Response received (ID: %d)\n", response.ID)

	return response.Result, nil
}

// Close the connection
func (c *DelveClient) Close() error {
	fmt.Println("\nClosing connection...")
	return c.conn.Close()
}

// Pretty print JSON
func prettyPrint(label string, data json.RawMessage) {
	var formatted interface{}
	if err := json.Unmarshal(data, &formatted); err != nil {
		fmt.Printf("%s: %s\n", label, string(data))
		return
	}

	pretty, err := json.MarshalIndent(formatted, "", "  ")
	if err != nil {
		fmt.Printf("%s: %s\n", label, string(data))
		return
	}

	fmt.Printf("%s:\n%s\n", label, string(pretty))
}

func main() {
	fmt.Println("Delve API POC - Testing JSON-RPC Connection")
	fmt.Println("=" + string(make([]byte, 50)))

	// Connect to Delve server
	client, err := Connect("localhost", 2345)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	// Test 1: Get current state
	fmt.Println("\n--- TEST 1: Get Debugger State ---")
	result, err := client.Call("RPCServer.State", map[string]interface{}{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting state: %v\n", err)
	} else {
		prettyPrint("State", result)
	}

	// Test 2: List breakpoints
	fmt.Println("\n--- TEST 2: List Breakpoints ---")
	result, err = client.Call("RPCServer.ListBreakpoints", map[string]interface{}{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listing breakpoints: %v\n", err)
	} else {
		prettyPrint("Breakpoints", result)
	}

	// Test 3: List goroutines
	fmt.Println("\n--- TEST 3: List Goroutines ---")
	result, err = client.Call("RPCServer.ListGoroutines", map[string]interface{}{
		"Start": 0,
		"Count": 10,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listing goroutines: %v\n", err)
	} else {
		prettyPrint("Goroutines", result)
	}

	// Test 4: Find location
	fmt.Println("\n--- TEST 4: Find Location (main.main) ---")
	result, err = client.Call("RPCServer.FindLocation", map[string]interface{}{
		"Scope": map[string]interface{}{
			"GoroutineID": -1,
			"Frame":       0,
		},
		"Loc": "main.main",
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error finding location: %v\n", err)
	} else {
		prettyPrint("Location", result)
	}

	// Test 5: List local variables (if stopped at a breakpoint)
	fmt.Println("\n--- TEST 5: List Local Variables ---")
	result, err = client.Call("RPCServer.ListLocalVars", map[string]interface{}{
		"Scope": map[string]interface{}{
			"GoroutineID": -1,
			"Frame":       0,
		},
		"Cfg": map[string]interface{}{
			"MaxStringLen": 64,
			"MaxArrayValues": 64,
		},
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listing local vars: %v\n", err)
	} else {
		prettyPrint("Local Variables", result)
	}

	// Test 6: List functions matching pattern
	fmt.Println("\n--- TEST 6: List Functions (main.*) ---")
	result, err = client.Call("RPCServer.ListFunctions", map[string]interface{}{
		"Filter": "main.*",
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listing functions: %v\n", err)
	} else {
		prettyPrint("Functions", result)
	}

	// Test 7: Get stack trace
	fmt.Println("\n--- TEST 7: Get Stack Trace ---")
	result, err = client.Call("RPCServer.Stacktrace", map[string]interface{}{
		"Id":    -1, // Current goroutine
		"Depth": 10,
		"Opts": map[string]interface{}{
			"FollowPointers": true,
		},
		"Cfg": map[string]interface{}{
			"MaxStringLen": 64,
		},
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting stacktrace: %v\n", err)
	} else {
		prettyPrint("Stack Trace", result)
	}

	fmt.Println("\n" + string(make([]byte, 50)))
	fmt.Println("POC completed successfully!")
}
