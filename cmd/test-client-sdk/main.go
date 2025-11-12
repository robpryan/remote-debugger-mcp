package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const mcpURL = "http://localhost:8899/mcp"

func main() {
	fmt.Println("=== MCP Delve Test Client (Using Official SDK) ===\n")

	ctx := context.Background()

	// Create MCP client
	client := mcp.NewClient(&mcp.Implementation{
		Name:    "test-client-sdk",
		Version: "1.0.0",
	}, nil)

	// Create SSE transport
	transport := &mcp.SSEClientTransport{
		Endpoint: mcpURL,
	}

	// Connect to server
	fmt.Println("[1/9] Connecting to MCP server...")
	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer session.Close()
	fmt.Println("✓ Connected and initialized\n")

	sessionID := "test-sdk-session"

	// Helper function for tool calls
	callTool := func(step int, action string, args map[string]interface{}, desc string) {
		fmt.Printf("[%d/9] %s...\n", step, desc)

		params := &mcp.CallToolParams{
			Name:      "delve",
			Arguments: args,
		}

		result, err := session.CallTool(ctx, params)
		if err != nil {
			log.Fatalf("%s failed: %v", desc, err)
		}

		// Print result
		if len(result.Content) > 0 {
			if textContent, ok := result.Content[0].(*mcp.TextContent); ok {
				fmt.Printf("✓ %s\n", truncate(textContent.Text, 100))
			} else {
				fmt.Printf("✓ Success\n")
			}
		} else {
			fmt.Printf("✓ Success (no content)\n")
		}
		fmt.Println()
	}

	// Test 1: Connect to Delve
	callTool(2, "connect", map[string]interface{}{
		"session_id": sessionID,
		"action":     "connect",
		"host":       "localhost",
		"port":       2345,
	}, "Connecting to Delve")

	// Test 2: Halt
	callTool(3, "command", map[string]interface{}{
		"session_id": sessionID,
		"action":     "command",
		"command":    "halt",
	}, "Halting program")

	// Test 3: List goroutines
	callTool(4, "command", map[string]interface{}{
		"session_id": sessionID,
		"action":     "command",
		"command":    "goroutines",
	}, "Listing goroutines")

	// Test 4: Set breakpoint
	callTool(5, "command", map[string]interface{}{
		"session_id": sessionID,
		"action":     "command",
		"command":    "break main.calculateSum",
	}, "Setting breakpoint at main.calculateSum")

	// Test 5: List breakpoints
	callTool(6, "command", map[string]interface{}{
		"session_id": sessionID,
		"action":     "command",
		"command":    "breakpoints",
	}, "Listing breakpoints")

	// Test 6: Continue
	callTool(7, "command", map[string]interface{}{
		"session_id": sessionID,
		"action":     "command",
		"command":    "continue",
	}, "Continuing execution")

	// Wait a bit for breakpoint to hit
	time.Sleep(500 * time.Millisecond)

	// Test 7: Stack trace
	callTool(8, "command", map[string]interface{}{
		"session_id": sessionID,
		"action":     "command",
		"command":    "stack",
	}, "Getting stack trace")

	// Test 8: Disconnect
	callTool(9, "disconnect", map[string]interface{}{
		"session_id": sessionID,
		"action":     "disconnect",
	}, "Disconnecting from Delve")

	fmt.Println("=== All Tests Passed! ✓ ===")
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func prettyPrint(v interface{}) {
	b, _ := json.MarshalIndent(v, "", "  ")
	fmt.Println(string(b))
}
