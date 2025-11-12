#!/bin/bash

# Script to help run the Delve API POC test

set -e

echo "Delve API POC Test Runner"
echo "=========================="
echo ""

# Check if delve is installed
if ! command -v dlv &> /dev/null; then
    echo "Error: dlv (Delve) is not installed"
    echo "Install with: go install github.com/go-delve/delve/cmd/dlv@latest"
    exit 1
fi

echo "Step 1: Checking if port 2346 is already in use..."
if lsof -Pi :2346 -sTCP:LISTEN -t >/dev/null 2>&1; then
    echo "Port 2346 is already in use. Please stop the existing process or choose a different port."
    echo "To see what's using the port: lsof -i :2346"
    exit 1
fi
echo "Port 2346 is available."
echo ""

echo "Step 2: Starting Delve server in background..."
cd testapp
dlv debug --headless --api-version=2 --listen=127.0.0.1:2346 --accept-multiclient > /tmp/delve-poc.log 2>&1 &
DELVE_PID=$!
cd ..

echo "Delve server started (PID: $DELVE_PID)"
echo "Log file: /tmp/delve-poc.log"
echo ""

# Wait for Delve to be ready
echo "Step 3: Waiting for Delve to be ready..."
for i in {1..10}; do
    if lsof -Pi :2346 -sTCP:LISTEN -t >/dev/null 2>&1; then
        echo "Delve is ready!"
        break
    fi
    if [ $i -eq 10 ]; then
        echo "Error: Delve failed to start. Check /tmp/delve-poc.log"
        kill $DELVE_PID 2>/dev/null || true
        exit 1
    fi
    sleep 1
done
echo ""

echo "Step 4: Running POC client..."
echo ""
go run main.go

# Cleanup
echo ""
echo "Step 5: Cleaning up..."
kill $DELVE_PID 2>/dev/null || true
echo "Delve server stopped."
echo ""
echo "Test completed successfully!"
