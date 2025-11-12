# Testing the Delve MCP Implementation

This guide shows how to test the Delve MCP tool with the included test application.

## Prerequisites

- [Delve](https://github.com/go-delve/delve) installed (`go install github.com/go-delve/delve/cmd/dlv@latest`)
- Go 1.24+

## Quick Start

### Terminal 1: Start the Delve Server

```bash
./scripts/start-delve-test.sh
```

This will:
1. Build the test application
2. Start Delve in JSON-RPC mode on port 2345
3. Run the test app with `--continue` flag (starts running immediately)

### Terminal 2: Start the MCP Server

```bash
./scripts/start-mcp-server.sh
```

This will:
1. Build the MCP server
2. Start it on http://localhost:8899
3. Enable debug logging

### Terminal 3: Test with MCP Tool

Now you can test the MCP tool. Here are example commands:

#### 1. Connect to Delve Session

```bash
# Using curl to call MCP endpoint
curl -X POST http://localhost:8899/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "tools/call",
    "params": {
      "name": "delve",
      "arguments": {
        "session_id": "test1",
        "action": "connect",
        "host": "localhost",
        "port": 2345
      }
    }
  }'
```

#### 2. Halt the Running Program

```bash
curl -X POST http://localhost:8899/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 2,
    "method": "tools/call",
    "params": {
      "name": "delve",
      "arguments": {
        "session_id": "test1",
        "action": "command",
        "command": "halt"
      }
    }
  }'
```

#### 3. List Goroutines

```bash
curl -X POST http://localhost:8899/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 3,
    "method": "tools/call",
    "params": {
      "name": "delve",
      "arguments": {
        "session_id": "test1",
        "action": "command",
        "command": "goroutines"
      }
    }
  }'
```

#### 4. Set a Breakpoint

```bash
curl -X POST http://localhost:8899/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 4,
    "method": "tools/call",
    "params": {
      "name": "delve",
      "arguments": {
        "session_id": "test1",
        "action": "command",
        "command": "break main.calculateSum"
      }
    }
  }'
```

#### 5. List Breakpoints

```bash
curl -X POST http://localhost:8899/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 5,
    "method": "tools/call",
    "params": {
      "name": "delve",
      "arguments": {
        "session_id": "test1",
        "action": "command",
        "command": "breakpoints"
      }
    }
  }'
```

#### 6. Continue Execution

```bash
curl -X POST http://localhost:8899/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 6,
    "method": "tools/call",
    "params": {
      "name": "delve",
      "arguments": {
        "session_id": "test1",
        "action": "command",
        "command": "continue"
      }
    }
  }'
```

#### 7. Get Stack Trace (when halted at breakpoint)

```bash
curl -X POST http://localhost:8899/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 7,
    "method": "tools/call",
    "params": {
      "name": "delve",
      "arguments": {
        "session_id": "test1",
        "action": "command",
        "command": "stack"
      }
    }
  }'
```

#### 8. List Local Variables

```bash
curl -X POST http://localhost:8899/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 8,
    "method": "tools/call",
    "params": {
      "name": "delve",
      "arguments": {
        "session_id": "test1",
        "action": "command",
        "command": "locals"
      }
    }
  }'
```

#### 9. Print a Variable

```bash
curl -X POST http://localhost:8899/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 9,
    "method": "tools/call",
    "params": {
      "name": "delve",
      "arguments": {
        "session_id": "test1",
        "action": "command",
        "command": "print numbers"
      }
    }
  }'
```

#### 10. Disconnect

```bash
curl -X POST http://localhost:8899/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 10,
    "method": "tools/call",
    "params": {
      "name": "delve",
      "arguments": {
        "session_id": "test1",
        "action": "disconnect"
      }
    }
  }'
```

## Test Application Features

The test application (`cmd/test-app/main.go`) includes:

- **Multiple Goroutines**: 3 concurrent goroutines for testing goroutine inspection
- **Various Data Types**: structs, slices, maps, primitives
- **Recursive Functions**: `calculateFactorial()` for stack trace testing
- **Loop Iterations**: Main loop runs 50 times for repeated breakpoint testing
- **Local Variables**: `demonstrateLocalVars()` has many variable types
- **Shared State**: `Counter` type with mutex for concurrency testing

## Good Breakpoint Locations

Try setting breakpoints at:
- `main.main` - Entry point
- `main.calculateSum` - Simple function
- `main.calculateFactorial` - Recursive function
- `main.demonstrateLocalVars` - Many local variables to inspect
- `main.(*Counter).Increment` - Method with mutex
- `main.processNumbers` - Goroutine function

## Available Commands

All standard Delve commands are supported:

**Execution Control:**
- `continue` (c) - Resume execution
- `next` (n) - Step over
- `step` (s) - Step into
- `stepout` (so) - Step out
- `halt` - Pause execution
- `restart` (r) - Restart debugging

**Breakpoints:**
- `break <location>` (b) - Set breakpoint
- `breakpoints` (bp) - List breakpoints
- `clear <id>` - Remove breakpoint
- `clearall` - Remove all breakpoints

**Inspection:**
- `print <expr>` (p) - Print expression
- `locals` - List local variables
- `args` - List function arguments
- `stack` (bt) - Show stack trace
- `goroutines` (grs) - List goroutines
- `goroutine <id>` (gr) - Switch goroutine
- `frame <n>` - Switch stack frame

**Navigation:**
- `funcs <regex>` - List functions
- `types <regex>` - List types
- `sources <regex>` - List source files

## Troubleshooting

### Connection Refused
- Make sure Delve server is running on port 2345
- Check that you're using JSON-RPC mode (not DAP)

### "Program is running" Errors
- Call `halt` command first before inspection commands
- The API client has `AutoHalt` enabled, so this should be automatic

### Tests Fail to Connect
- Verify `dlv` is installed: `dlv version`
- Check port 2345 is not in use: `lsof -i :2345`
- Ensure test app is built: `ls -la build/test-app`

## Manual Delve Testing

You can also test directly with Delve CLI:

```bash
# Start Delve server
dlv exec --headless --api-version=2 --listen=:2345 --accept-multiclient --continue ./build/test-app

# In another terminal, connect with CLI
dlv connect :2345
```
