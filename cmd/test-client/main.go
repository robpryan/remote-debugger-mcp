package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
)

const mcpURL = "http://localhost:8899/mcp"

type MCPRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
}

type ToolCallParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

func main() {
	fmt.Println("=== MCP Delve Test Client ===\n")

	// Initialize MCP session
	fmt.Println("[0/8] Initializing MCP session...")
	if err := initializeMCPSession(); err != nil {
		log.Fatalf("Failed to initialize MCP session: %v", err)
	}
	fmt.Println("✓ MCP session initialized\n")

	sessionID := "test-client-session"

	// Test 1: Connect
	fmt.Println("[1/8] Connecting to Delve...")
	result, err := callMCPTool(1, "delve", map[string]interface{}{
		"session_id": sessionID,
		"action":     "connect",
		"host":       "localhost",
		"port":       2345,
	})
	if err != nil {
		log.Fatalf("Connect failed: %v", err)
	}
	fmt.Printf("✓ Connected: %s\n\n", result)

	// Test 2: Halt
	fmt.Println("[2/8] Halting program...")
	result, err = callMCPCommand(2, sessionID, "halt")
	if err != nil {
		log.Fatalf("Halt failed: %v", err)
	}
	fmt.Printf("✓ Halted: %s\n\n", truncate(result, 200))

	// Test 3: List goroutines
	fmt.Println("[3/8] Listing goroutines...")
	result, err = callMCPCommand(3, sessionID, "goroutines")
	if err != nil {
		log.Fatalf("Goroutines failed: %v", err)
	}
	fmt.Printf("✓ Goroutines listed: %s\n\n", truncate(result, 300))

	// Test 4: Set breakpoint
	fmt.Println("[4/8] Setting breakpoint at main.calculateSum...")
	result, err = callMCPCommand(4, sessionID, "break main.calculateSum")
	if err != nil {
		log.Fatalf("Break failed: %v", err)
	}
	fmt.Printf("✓ Breakpoint set: %s\n\n", result)

	// Test 5: List breakpoints
	fmt.Println("[5/8] Listing breakpoints...")
	result, err = callMCPCommand(5, sessionID, "breakpoints")
	if err != nil {
		log.Fatalf("Breakpoints failed: %v", err)
	}
	fmt.Printf("✓ Breakpoints: %s\n\n", result)

	// Test 6: Continue
	fmt.Println("[6/8] Continuing execution...")
	result, err = callMCPCommand(6, sessionID, "continue")
	if err != nil {
		log.Fatalf("Continue failed: %v", err)
	}
	fmt.Printf("✓ Continued: %s\n\n", truncate(result, 200))

	// Test 7: Stack trace (after waiting for breakpoint)
	fmt.Println("[7/8] Getting stack trace...")
	result, err = callMCPCommand(7, sessionID, "stack")
	if err != nil {
		log.Fatalf("Stack failed: %v", err)
	}
	fmt.Printf("✓ Stack: %s\n\n", truncate(result, 400))

	// Test 8: Disconnect
	fmt.Println("[8/8] Disconnecting...")
	result, err = callMCPTool(8, "delve", map[string]interface{}{
		"session_id": sessionID,
		"action":     "disconnect",
	})
	if err != nil {
		log.Fatalf("Disconnect failed: %v", err)
	}
	fmt.Printf("✓ Disconnected: %s\n\n", result)

	fmt.Println("=== All Tests Passed! ✓ ===")
}

func callMCPCommand(id int, sessionID, command string) (string, error) {
	return callMCPTool(id, "delve", map[string]interface{}{
		"session_id": sessionID,
		"action":     "command",
		"command":    command,
	})
}

func callMCPTool(id int, toolName string, args map[string]interface{}) (string, error) {
	req := MCPRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  "tools/call",
		Params: ToolCallParams{
			Name:      toolName,
			Arguments: args,
		},
	}

	reqBytes, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", mcpURL, bytes.NewReader(reqBytes))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json, text/event-stream")

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("http post: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	// Handle SSE format
	bodyStr := string(body)
	lines := strings.Split(bodyStr, "\n")
	var jsonData string
	for _, line := range lines {
		if strings.HasPrefix(line, "data: ") {
			jsonData = strings.TrimPrefix(line, "data: ")
			break
		}
	}

	if jsonData == "" {
		return "", fmt.Errorf("no data in SSE response: %s", bodyStr)
	}

	// Parse JSON response
	var mcpResp struct {
		JSONRPC string `json:"jsonrpc"`
		ID      int    `json:"id"`
		Result  struct {
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"result"`
		Error struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal([]byte(jsonData), &mcpResp); err != nil {
		return "", fmt.Errorf("unmarshal response: %w (data: %s)", err, jsonData)
	}

	if mcpResp.Error.Message != "" {
		return "", fmt.Errorf("MCP error: %s", mcpResp.Error.Message)
	}

	if len(mcpResp.Result.Content) == 0 {
		return "", fmt.Errorf("no content in response")
	}

	return mcpResp.Result.Content[0].Text, nil
}

func initializeMCPSession() error {
	req := MCPRequest{
		JSONRPC: "2.0",
		ID:      0,
		Method:  "initialize",
		Params: map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]interface{}{},
			"clientInfo": map[string]string{
				"name":    "test-client",
				"version": "1.0.0",
			},
		},
	}

	reqBytes, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal init request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", mcpURL, bytes.NewReader(reqBytes))
	if err != nil {
		return fmt.Errorf("create init request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json, text/event-stream")

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("http post init: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read init response: %w", err)
	}

	// Parse SSE response
	bodyStr := string(body)
	lines := strings.Split(bodyStr, "\n")
	foundData := false
	for _, line := range lines {
		if strings.HasPrefix(line, "data: ") {
			foundData = true
			break
		}
	}

	if !foundData {
		return fmt.Errorf("no data in init response")
	}

	// Send initialized notification (required by MCP protocol)
	notif := MCPRequest{
		JSONRPC: "2.0",
		Method:  "notifications/initialized",
		Params:  map[string]interface{}{},
	}

	notifBytes, err := json.Marshal(notif)
	if err != nil {
		return fmt.Errorf("marshal initialized notification: %w", err)
	}

	httpReq, err = http.NewRequest("POST", mcpURL, bytes.NewReader(notifBytes))
	if err != nil {
		return fmt.Errorf("create initialized notification: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json, text/event-stream")

	resp, err = client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("send initialized notification: %w", err)
	}
	defer resp.Body.Close()

	// Read and discard the response
	_, _ = io.ReadAll(resp.Body)

	return nil
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
