#!/bin/bash
# Test MCP client against running servers

set -e

echo "=== MCP Client Test ==="
echo ""

# Check if servers are running
echo "[1/3] Checking servers..."
if ! lsof -i :8899 > /dev/null 2>&1; then
    echo "✗ MCP server not running on port 8899"
    echo "  Start it with: ./scripts/start-mcp-server.sh"
    exit 1
fi
echo "✓ MCP server running on port 8899"

if ! lsof -i :2345 > /dev/null 2>&1; then
    echo "⚠ Warning: Delve server not running on port 2345"
    echo "  Start it with: ./scripts/start-delve-test.sh"
    echo "  (Continuing anyway to test MCP connection)"
fi
echo ""

# Run test client
echo "[2/3] Running test client..."
echo ""
go run cmd/test-client-sdk/main.go

echo ""
echo "[3/3] Test complete!"
