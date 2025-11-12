#!/bin/bash
# Start the MCP server

set -e

# Build MCP server first
echo "Building MCP server..."
cd "$(dirname "$0")/.."
make build

# Start MCP server
echo "Starting MCP server on localhost:8899..."
echo "MCP endpoint: http://localhost:8899/mcp"
echo "Use Ctrl+C to stop"
echo ""

./build/remote-debugger-mcp --debug
