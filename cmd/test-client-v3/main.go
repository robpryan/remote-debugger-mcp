package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"sync"
)

const mcpURL = "http://localhost:8899/mcp"

type MCPMessage struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      *int        `json:"id,omitempty"`
	Method  string      `json:"method,omitempty"`
	Params  interface{} `json:"params,omitempty"`
	Result  *struct {
		ProtocolVersion string `json:"protocolVersion,omitempty"`
		Capabilities    interface{} `json:"capabilities,omitempty"`
		ServerInfo      interface{} `json:"serverInfo,omitempty"`
		Content         []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content,omitempty"`
	} `json:"result,omitempty"`
	Error *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

type MCPClient struct {
	url        string
	httpClient *http.Client
	responses  chan *MCPMessage
	errors     chan error
	writer     io.Writer
	mu         sync.Mutex
	nextID     int
}

func NewMCPClient(url string) *MCPClient {
	jar, _ := cookiejar.New(nil)
	return &MCPClient{
		url:        url,
		httpClient: &http.Client{Jar: jar},
		responses:  make(chan *MCPMessage, 10),
		errors:     make(chan error, 10),
		nextID:     0,
	}
}

func (c *MCPClient) sendMessage(msg *MCPMessage) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}

	req, err := http.NewRequest("POST", c.url, bytes.NewReader(msgBytes))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("http post: %w", err)
	}
	defer resp.Body.Close()

	// Read SSE response
	scanner := bufio.NewScanner(resp.Body)
	var lastData string

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			lastData = strings.TrimPrefix(line, "data: ")
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read SSE: %w", err)
	}

	if lastData == "" {
		// No response for notifications
		return nil
	}

	var response MCPMessage
	if err := json.Unmarshal([]byte(lastData), &response); err != nil {
		return fmt.Errorf("unmarshal response: %w", err)
	}

	c.responses <- &response
	return nil
}

func (c *MCPClient) SendRequest(method string, params interface{}) (*MCPMessage, error) {
	id := c.nextID
	c.nextID++

	msg := &MCPMessage{
		JSONRPC: "2.0",
		ID:      &id,
		Method:  method,
		Params:  params,
	}

	if err := c.sendMessage(msg); err != nil {
		return nil, err
	}

	// Wait for response
	select {
	case resp := <-c.responses:
		if resp.Error != nil {
			return nil, fmt.Errorf("MCP error: %s", resp.Error.Message)
		}
		return resp, nil
	case err := <-c.errors:
		return nil, err
	}
}

func (c *MCPClient) SendNotification(method string, params interface{}) error {
	msg := &MCPMessage{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
	}

	return c.sendMessage(msg)
}

func main() {
	fmt.Println("=== MCP Delve Test Client v3 ===\n")

	client := NewMCPClient(mcpURL)
	sessionID := "test-client-v3"

	// Initialize
	fmt.Println("[0/9] Initializing MCP session...")
	resp, err := client.SendRequest("initialize", map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]interface{}{},
		"clientInfo": map[string]string{
			"name":    "test-client-v3",
			"version": "1.0.0",
		},
	})
	if err != nil {
		log.Fatalf("Initialize failed: %v", err)
	}
	fmt.Printf("✓ Initialized: %v\n\n", resp.Result.ServerInfo)

	// Send initialized notification
	fmt.Println("[1/9] Sending initialized notification...")
	if err := client.SendNotification("notifications/initialized", nil); err != nil {
		log.Fatalf("Initialized notification failed: %v", err)
	}
	fmt.Println("✓ Notification sent\n")

	// Helper for tool calls
	callTool := func(step int, args map[string]interface{}, desc string) string {
		fmt.Printf("[%d/9] %s...\n", step, desc)
		resp, err := client.SendRequest("tools/call", map[string]interface{}{
			"name":      "delve",
			"arguments": args,
		})
		if err != nil {
			log.Fatalf("%s failed: %v", desc, err)
		}
		if resp.Result == nil || len(resp.Result.Content) == 0 {
			log.Fatalf("%s: no content", desc)
		}
		result := resp.Result.Content[0].Text
		fmt.Printf("✓ Success\n\n")
		return result
	}

	// Tests
	callTool(2, map[string]interface{}{
		"session_id": sessionID,
		"action":     "connect",
		"host":       "localhost",
		"port":       2345,
	}, "Connecting to Delve")

	callTool(3, map[string]interface{}{
		"session_id": sessionID,
		"action":     "command",
		"command":    "halt",
	}, "Halting program")

	result := callTool(4, map[string]interface{}{
		"session_id": sessionID,
		"action":     "command",
		"command":    "goroutines",
	}, "Listing goroutines")
	fmt.Printf("   %s\n\n", truncate(result, 100))

	callTool(5, map[string]interface{}{
		"session_id": sessionID,
		"action":     "command",
		"command":    "break main.calculateSum",
	}, "Setting breakpoint")

	callTool(6, map[string]interface{}{
		"session_id": sessionID,
		"action":     "command",
		"command":    "breakpoints",
	}, "Listing breakpoints")

	callTool(7, map[string]interface{}{
		"session_id": sessionID,
		"action":     "command",
		"command":    "continue",
	}, "Continuing execution")

	result = callTool(8, map[string]interface{}{
		"session_id": sessionID,
		"action":     "command",
		"command":    "stack",
	}, "Getting stack trace")
	fmt.Printf("   %s\n\n", truncate(result, 150))

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
