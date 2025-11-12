# Delve API POC

This is a proof-of-concept program to test the Delve JSON-RPC API.

## Components

1. **testapp/** - Simple Go program to debug
2. **main.go** - POC client that connects to Delve and tests various API calls

## How to Run

### Terminal 1: Start the test application with Delve

```bash
cd cmd/delve-api-poc/testapp
dlv debug --headless --api-version=2 --listen=127.0.0.1:2346 --accept-multiclient
```

This will:
- Start the debugger in headless mode
- Listen on port 2346
- Use API version 2
- Allow multiple clients to connect

### Terminal 2: Run the POC client

```bash
# Build and run the POC
cd cmd/delve-api-poc
go run main.go
```

## What the POC Tests

The POC client will:
1. **Connect** to the Delve server on localhost:2346
2. **Get debugger state** - Check current execution state
3. **List breakpoints** - Show all active breakpoints
4. **List goroutines** - Show running goroutines
5. **Find location** - Resolve `main.main` function location
6. **List local variables** - Show variables in current scope
7. **List functions** - Find functions matching pattern `main.*`
8. **Get stack trace** - Show current call stack

## Example Output

```
Delve API POC - Testing JSON-RPC Connection
====================================================

Connecting to Delve server at localhost:2345...
Connected successfully!

--- TEST 1: Get Debugger State ---
==> Calling RPCServer.GetState (ID: 1)
    Params: map[]
<== Response received (ID: 1)
State:
{
  "Running": false,
  "currentThread": {...},
  ...
}

--- TEST 2: List Breakpoints ---
...
```

## Troubleshooting

### "Connection refused"
Make sure Delve is running on port 2346. Check with:
```bash
lsof -i :2346
```

### "No such file or directory"
Make sure you're in the correct directory when running dlv debug.

### "Cannot find main module"
Run `go mod init` in the testapp directory if needed.

## Next Steps

Once the POC works, we'll:
1. Extract the client code into `pkg/tools/delve/client` package
2. Add proper error handling and types
3. Implement command wrappers for each operation
4. Integrate into the MCP tool
