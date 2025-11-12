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
	ID      *int        `json:"id,omitempty"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

type MCPResponse struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Result  *struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	} `json:"result,omitempty"`
	Error *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

type SSEClient struct {
	httpClient *http.Client
	url        string
}

func NewSSEClient(url string) *SSEClient {
	return &SSEClient{
		httpClient: &http.Client{},
		url:        url,
	}
}

func (c *SSEClient) SendRequest(req MCPRequest) (*MCPResponse, error) {
	reqBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", c.url, bytes.NewReader(reqBytes))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json, text/event-stream")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("http post: %w", err)
	}
	defer resp.Body.Close()

	// Read SSE stream
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	// Debug: print raw response
	bodyStr := string(body)
	fmt.Printf("DEBUG: Raw response:\n%s\n---\n", bodyStr)

	// Parse SSE format
	var lastData string
	lines := strings.Split(bodyStr, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "data: ") {
			lastData = strings.TrimPrefix(line, "data: ")
		}
	}

	if lastData == "" {
		return nil, fmt.Errorf("no data in SSE response")
	}

	var mcpResp MCPResponse
	if err := json.Unmarshal([]byte(lastData), &mcpResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w (data: %s)", err, lastData)
	}

	return &mcpResp, nil
}

func (c *SSEClient) SendNotification(req MCPRequest) error {
	reqBytes, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal notification: %w", err)
	}

	httpReq, err := http.NewRequest("POST", c.url, bytes.NewReader(reqBytes))
	if err != nil {
		return fmt.Errorf("create notification: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json, text/event-stream")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("http post notification: %w", err)
	}
	defer resp.Body.Close()

	// Consume and discard the response
	_, _ = io.Copy(io.Discard, resp.Body)

	return nil
}

func main() {
	fmt.Println("=== MCP Delve Test Client v2 ===\n")

	client := NewSSEClient(mcpURL)
	sessionID := "test-client-v2"

	// Step 1: Initialize
	fmt.Println("[0/9] Initializing MCP session...")
	id := 0
	initResp, err := client.SendRequest(MCPRequest{
		JSONRPC: "2.0",
		ID:      &id,
		Method:  "initialize",
		Params: map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]interface{}{},
			"clientInfo": map[string]string{
				"name":    "test-client-v2",
				"version": "1.0.0",
			},
		},
	})
	if err != nil {
		log.Fatalf("Initialize failed: %v", err)
	}
	if initResp.Error != nil {
		log.Fatalf("Initialize error: %s", initResp.Error.Message)
	}
	fmt.Println("✓ Session initialized")

	// Step 2: Send initialized notification
	fmt.Println("[1/9] Sending initialized notification...")
	id = 1
	_, err = client.SendRequest(MCPRequest{
		JSONRPC: "2.0",
		ID:      &id,
		Method:  "notifications/initialized",
	})
	if err != nil {
		log.Fatalf("Initialized notification failed: %v", err)
	}
	fmt.Println("✓ Initialized notification sent\n")

	// Helper function for tool calls
	callTool := func(id int, args map[string]interface{}, desc string) string {
		fmt.Printf("[%d/9] %s...\n", id, desc)
		resp, err := client.SendRequest(MCPRequest{
			JSONRPC: "2.0",
			ID:      &id,
			Method:  "tools/call",
			Params: map[string]interface{}{
				"name":      "delve",
				"arguments": args,
			},
		})
		if err != nil {
			log.Fatalf("%s failed: %v", desc, err)
		}
		if resp.Error != nil {
			log.Fatalf("%s error: %s", desc, resp.Error.Message)
		}
		if resp.Result == nil || len(resp.Result.Content) == 0 {
			log.Fatalf("%s: no content in response", desc)
		}
		result := resp.Result.Content[0].Text
		fmt.Printf("✓ %s\n\n", result)
		return result
	}

	// Test 1: Connect
	callTool(2, map[string]interface{}{
		"session_id": sessionID,
		"action":     "connect",
		"host":       "localhost",
		"port":       2345,
	}, "Connecting to Delve")

	// Test 2: Halt
	callTool(3, map[string]interface{}{
		"session_id": sessionID,
		"action":     "command",
		"command":    "halt",
	}, "Halting program")

	// Test 3: List goroutines
	result := callTool(4, map[string]interface{}{
		"session_id": sessionID,
		"action":     "command",
		"command":    "goroutines",
	}, "Listing goroutines")
	fmt.Printf("   Found goroutines: %s\n\n", truncate(result, 100))

	// Test 4: Set breakpoint
	callTool(5, map[string]interface{}{
		"session_id": sessionID,
		"action":     "command",
		"command":    "break main.calculateSum",
	}, "Setting breakpoint at main.calculateSum")

	// Test 5: List breakpoints
	callTool(6, map[string]interface{}{
		"session_id": sessionID,
		"action":     "command",
		"command":    "breakpoints",
	}, "Listing breakpoints")

	// Test 6: Continue
	callTool(7, map[string]interface{}{
		"session_id": sessionID,
		"action":     "command",
		"command":    "continue",
	}, "Continuing execution")

	// Test 7: Stack trace
	result = callTool(8, map[string]interface{}{
		"session_id": sessionID,
		"action":     "command",
		"command":    "stack",
	}, "Getting stack trace")
	fmt.Printf("   Stack: %s\n\n", truncate(result, 150))

	// Test 8: Disconnect
	callTool(9, map[string]interface{}{
		"session_id": sessionID,
		"action":     "disconnect",
	}, "Disconnecting")

	fmt.Println("=== All Tests Passed! ✓ ===")
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
