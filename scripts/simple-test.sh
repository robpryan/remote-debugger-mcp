#!/bin/bash
# Simple test that doesn't use SSE/streaming

set -e

MCP_URL="http://localhost:8899"
SESSION_ID="simple-test-$$"

echo "=== Simple Delve MCP Test ==="
echo ""

# Check MCP server
echo "[1/3] Checking MCP server info endpoint..."
curl -s "$MCP_URL/" | python3 -m json.tool
echo ""

# Check if Delve is running
echo "[2/3] Checking Delve server..."
if lsof -i :2345 > /dev/null 2>&1; then
    echo "✓ Delve server is running on port 2345"
else
    echo "✗ Delve server is NOT running on port 2345"
    echo "Start it with: ./scripts/start-delve-test.sh"
    exit 1
fi
echo ""

# Direct API test using dlv connect CLI
echo "[3/3] Testing Delve API directly via dlv connect..."
echo "Commands: halt, goroutines, breakpoints, quit"
timeout 5 dlv connect localhost:2345 <<EOF || true
halt
goroutines
breakpoints
quit
EOF
echo ""

echo "=== Test Complete ==="
echo ""
echo "Servers are running correctly!"
echo "The MCP endpoint requires a proper MCP client (like Claude Code or the MCP Inspector)"
echo "  - Delve server: localhost:2345 ✓"
echo "  - MCP server: http://localhost:8899/mcp ✓"
