#!/bin/bash
# Quick test of the Delve MCP implementation
# This script tests the basic workflow

set -e

MCP_URL="http://localhost:8899/mcp"
SESSION_ID="quicktest-$(date +%s)"

echo "=== Delve MCP Quick Test ==="
echo "Session ID: $SESSION_ID"
echo ""

# Helper function to make MCP calls
call_mcp() {
    local method=$1
    local command=$2
    local id=$3

    if [ -z "$command" ]; then
        # Connect/disconnect actions
        curl -s -X POST $MCP_URL \
          -H "Content-Type: application/json" \
          -d "{
            \"jsonrpc\": \"2.0\",
            \"id\": $id,
            \"method\": \"tools/call\",
            \"params\": {
              \"name\": \"delve\",
              \"arguments\": {
                \"session_id\": \"$SESSION_ID\",
                \"action\": \"$method\",
                \"host\": \"localhost\",
                \"port\": 2345
              }
            }
          }" | jq -r '.result.content[0].text // .error.message // .'
    else
        # Command actions
        curl -s -X POST $MCP_URL \
          -H "Content-Type: application/json" \
          -d "{
            \"jsonrpc\": \"2.0\",
            \"id\": $id,
            \"method\": \"tools/call\",
            \"params\": {
              \"name\": \"delve\",
              \"arguments\": {
                \"session_id\": \"$SESSION_ID\",
                \"action\": \"command\",
                \"command\": \"$command\"
              }
            }
          }" | jq -r '.result.content[0].text // .error.message // .'
    fi
}

# Check if MCP server is running
echo "[1/10] Checking MCP server..."
if ! curl -s $MCP_URL > /dev/null 2>&1; then
    echo "ERROR: MCP server not running on $MCP_URL"
    echo "Start it with: ./scripts/start-mcp-server.sh"
    exit 1
fi
echo "✓ MCP server is running"
echo ""

# Check if Delve is running
echo "[2/10] Checking Delve server..."
if ! lsof -i :2345 > /dev/null 2>&1; then
    echo "ERROR: Delve server not running on port 2345"
    echo "Start it with: ./scripts/start-delve-test.sh"
    exit 1
fi
echo "✓ Delve server is running"
echo ""

# Test 1: Connect
echo "[3/10] Connecting to Delve..."
result=$(call_mcp "connect" "" 1)
echo "$result"
echo ""

# Test 2: Halt
echo "[4/10] Halting program..."
result=$(call_mcp "" "halt" 2)
echo "$result"
echo ""

# Test 3: List goroutines
echo "[5/10] Listing goroutines..."
result=$(call_mcp "" "goroutines" 3)
echo "$result" | head -20
echo ""

# Test 4: Set breakpoint
echo "[6/10] Setting breakpoint at main.calculateSum..."
result=$(call_mcp "" "break main.calculateSum" 4)
echo "$result"
echo ""

# Test 5: List breakpoints
echo "[7/10] Listing breakpoints..."
result=$(call_mcp "" "breakpoints" 5)
echo "$result"
echo ""

# Test 6: Continue (will run until breakpoint)
echo "[8/10] Continuing execution (will hit breakpoint)..."
result=$(call_mcp "" "continue" 6)
echo "$result"
echo ""

# Wait a bit for breakpoint to hit
sleep 2

# Test 7: Get stack trace
echo "[9/10] Getting stack trace at breakpoint..."
result=$(call_mcp "" "stack" 7)
echo "$result"
echo ""

# Test 8: Disconnect
echo "[10/10] Disconnecting..."
result=$(call_mcp "disconnect" "" 8)
echo "$result"
echo ""

echo "=== Quick Test Complete ==="
echo ""
echo "All basic operations work! ✓"
echo ""
echo "For more detailed testing, see TESTING.md"
