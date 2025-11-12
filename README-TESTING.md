# Delve MCP Testing Guide

Quick guide to test the Delve MCP implementation.

## What Was Built

This is an MCP (Model Context Protocol) server that provides remote debugging capabilities using the Delve debugger API. It migrated from CLI-based interaction to the official Delve JSON-RPC API for better performance and reliability.

## What You Need

1. **Go 1.24+** installed
2. **Delve** installed: `go install github.com/go-delve/delve/cmd/dlv@latest`
3. **jq** (optional, for prettier output): `brew install jq`

## Quick Start (3 Easy Steps)

### Step 1: Start the Test Application with Delve

Open Terminal 1:
```bash
./scripts/start-delve-test.sh
```

You should see:
```
Building test application...
Starting Delve server on port 2345...

=== Delve MCP Test Application ===
This app runs indefinitely for debugging practice
...
```

### Step 2: Start the MCP Server

Open Terminal 2:
```bash
./scripts/start-mcp-server.sh
```

You should see:
```
Building MCP server...
Starting MCP server on localhost:8899...
MCP endpoint: http://localhost:8899/mcp

{"level":"info","time":"...","message":"Remote Debugger MCP Connector starting..."}
```

### Step 3: Run the Quick Test

Open Terminal 3:
```bash
./scripts/quick-test.sh
```

This will test:
1. ✓ Connecting to Delve
2. ✓ Halting the program
3. ✓ Listing goroutines
4. ✓ Setting breakpoints
5. ✓ Continuing execution
6. ✓ Inspecting stack traces
7. ✓ Disconnecting

## Test Application Features

The test app (`cmd/test-app/main.go`) includes:

- **3 Concurrent Goroutines** for testing goroutine inspection
- **Multiple Data Types**: structs, slices, maps, primitives
- **Recursive Functions** for stack trace testing
- **50 Iterations** for repeated breakpoint testing
- **Rich Local Variables** for inspection testing

## Manual Testing

You can also test manually using curl:

```bash
# Connect
curl -X POST http://localhost:8899/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "tools/call",
    "params": {
      "name": "delve",
      "arguments": {
        "session_id": "manual-test",
        "action": "connect",
        "host": "localhost",
        "port": 2345
      }
    }
  }' | jq

# Halt
curl -X POST http://localhost:8899/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 2,
    "method": "tools/call",
    "params": {
      "name": "delve",
      "arguments": {
        "session_id": "manual-test",
        "action": "command",
        "command": "halt"
      }
    }
  }' | jq

# List goroutines
curl -X POST http://localhost:8899/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 3,
    "method": "tools/call",
    "params": {
      "name": "delve",
      "arguments": {
        "session_id": "manual-test",
        "action": "command",
        "command": "goroutines"
      }
    }
  }' | jq
```

## Available Commands

**Session Management:**
- `action: "connect"` - Connect to Delve server
- `action: "disconnect"` - Disconnect from server
- `action: "command"` - Execute debugging command

**Execution Control:**
- `continue` (c) - Resume execution
- `next` (n) - Step over
- `step` (s) - Step into
- `stepout` (so) - Step out
- `halt` - Pause execution

**Breakpoints:**
- `break <location>` (b) - Set breakpoint (e.g., "break main.main")
- `breakpoints` (bp) - List all breakpoints
- `clear <id>` - Remove breakpoint by ID

**Inspection:**
- `print <expr>` (p) - Print variable or expression
- `locals` - List local variables
- `stack` (bt) - Show stack trace
- `goroutines` (grs) - List goroutines
- `funcs <regex>` - List functions matching pattern

## Good Test Locations

Try setting breakpoints at these functions:

```bash
# Entry point
break main.main

# Simple function
break main.calculateSum

# Recursive function (good for stack traces)
break main.calculateFactorial

# Function with many local variables
break main.demonstrateLocalVars

# Goroutine function
break main.processNumbers

# Method
break main.(*Counter).Increment
```

## Architecture

The implementation uses:
- **Delve SDK** (`github.com/go-delve/delve/service/rpc2`) for JSON-RPC communication
- **Session Management** with auto-cleanup (30-minute timeout)
- **AutoHalt Mode** for safe inspection with `--continue` flag
- **Command Routing** that maps CLI-style commands to API calls

## Troubleshooting

**"Connection refused"**
- Make sure Delve is running on port 2345
- Run: `lsof -i :2345` to check

**"MCP server not responding"**
- Make sure MCP server is running on port 8899
- Run: `lsof -i :8899` to check

**"Session not found"**
- Make sure you called `action: "connect"` first
- Use the same `session_id` for all commands

**Breakpoint never hits**
- Make sure you called `continue` after setting the breakpoint
- Try `halt` first, then check `breakpoints` to see if it was set

## Next Steps

For more detailed testing information, see [TESTING.md](TESTING.md)

For the POC validation results, see [cmd/delve-api-poc/POC_SUMMARY.md](cmd/delve-api-poc/POC_SUMMARY.md)
