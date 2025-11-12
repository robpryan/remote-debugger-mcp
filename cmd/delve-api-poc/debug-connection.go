package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"time"
)

// Simple diagnostic tool to see raw JSON-RPC communication
func main() {
	fmt.Println("Delve Connection Debugger")
	fmt.Println("=" + string(make([]byte, 50)))

	// Connect
	fmt.Println("\nConnecting to localhost:2345...")
	conn, err := net.DialTimeout("tcp", "localhost:2345", 5*time.Second)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()
	fmt.Println("Connected!")

	// Set read deadline
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))

	// Send a simple State request
	request := map[string]interface{}{
		"method": "RPCServer.State",
		"params": []interface{}{map[string]interface{}{}},
		"id":     1,
	}

	requestJSON, _ := json.MarshalIndent(request, "", "  ")
	fmt.Printf("\n==> Sending request:\n%s\n", string(requestJSON))

	encoder := json.NewEncoder(conn)
	if err := encoder.Encode(request); err != nil {
		fmt.Fprintf(os.Stderr, "Error sending: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\n<== Waiting for response (5 second timeout)...")

	// Read response
	reader := bufio.NewReader(conn)

	// Try to read first few bytes to see what we get
	firstByte, err := reader.ReadByte()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading first byte: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("First byte received: %c (0x%02x)\n", firstByte, firstByte)

	// Put it back and read full line/object
	reader.UnreadByte()

	var response map[string]interface{}
	decoder := json.NewDecoder(reader)
	if err := decoder.Decode(&response); err != nil {
		fmt.Fprintf(os.Stderr, "Error decoding response: %v\n", err)
		os.Exit(1)
	}

	responseJSON, _ := json.MarshalIndent(response, "", "  ")
	fmt.Printf("\nResponse received:\n%s\n", string(responseJSON))

	fmt.Println("\n" + string(make([]byte, 50)))
	fmt.Println("Connection test completed!")
}
