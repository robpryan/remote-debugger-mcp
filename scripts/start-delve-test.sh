#!/bin/bash
# Start Delve server for test application

set -e

# Build test app first
echo "Building test application..."
cd "$(dirname "$0")/.."
go build -o build/test-app ./cmd/test-app/

# Start Delve in JSON-RPC mode
echo "Starting Delve server on port 2345..."
echo "Use Ctrl+C to stop"
echo ""

dlv exec \
  --headless \
  --api-version=2 \
  --listen=:2345 \
  --accept-multiclient \
  --continue \
  ./build/test-app
