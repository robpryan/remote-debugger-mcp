package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"time"
)

// Raw TCP debug - see what the server actually sends back
func main() {
	fmt.Println("Delve TCP Debug - See Raw Communication")
	fmt.Println("=" + string(make([]byte, 60)))

	// Raw TCP connection
	fmt.Println("\nConnecting to localhost:2345...")
	conn, err := net.DialTimeout("tcp", "localhost:2345", 5*time.Second)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()
	fmt.Println("Connected!")

	// Send a JSON-RPC request
	request := map[string]interface{}{
		"method": "RPCServer.ListBreakpoints",
		"params": []interface{}{map[string]interface{}{}},
		"id":     1,
	}

	requestJSON, _ := json.Marshal(request)
	fmt.Printf("\nSending JSON-RPC request:\n%s\n", string(requestJSON))

	// Write request with newline
	_, err = conn.Write(append(requestJSON, '\n'))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error writing: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\nWaiting for response (3 second timeout)...")
	conn.SetReadDeadline(time.Now().Add(3 * time.Second))

	// Try to read any response
	reader := bufio.NewReader(conn)

	// Read first line
	line, err := reader.ReadBytes('\n')
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading response: %v\n", err)
		fmt.Println("\nNO RESPONSE RECEIVED - This confirms the server is not responding to JSON-RPC")
	} else {
		fmt.Printf("\nResponse received (%d bytes):\n%s\n", len(line), string(line))

		// Try to parse as JSON
		var response map[string]interface{}
		if err := json.Unmarshal(line, &response); err == nil {
			pretty, _ := json.MarshalIndent(response, "", "  ")
			fmt.Printf("\nParsed JSON:\n%s\n", string(pretty))
		}
	}

	fmt.Println("\n" + string(make([]byte, 60)))
}
