#!/bin/bash

# Script to start a simple Go program with Delve in JSON-RPC mode

echo "Starting Delve with simple test app..."
echo "======================================"
echo ""

cd testapp

# Start Delve in headless mode with JSON-RPC API
# Using --headless without --listen uses JSON-RPC on default port 2345
echo "Command: dlv exec --headless --api-version=2 --listen=localhost:2345 --accept-multiclient ./simple"
echo ""

# Build the program first
echo "Building simple.go..."
go build -gcflags='all=-N -l' -o simple simple.go

if [ $? -ne 0 ]; then
    echo "Build failed!"
    exit 1
fi

echo "Build successful!"
echo ""
echo "Starting Delve server on localhost:2345..."
echo "Press Ctrl+C to stop"
echo ""

# Start Delve
dlv exec --headless --api-version=2 --listen=localhost:2345 --accept-multiclient ./simple
