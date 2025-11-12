# Go Debugger MCP - Project Notes

## Project Overview

The Go Debugger MCP is a Model Context Protocol (MCP) server that provides interactive Go debugging capabilities using the Delve debugger API. It acts as a bridge between AI coding assistants (like Claude Code) and the Delve debugger.

**Current Version:** 1.0
**Language:** Go 1.25
**License:** BSD 3-Clause

## Architecture

### Core Components

1. **MCP Server** (`cmd/debugger/main.go`)
   - HTTP server running on port 8899 (default)
   - Exposes MCP endpoint at `/mcp`
   - Uses StreamableHTTPHandler for Claude Code compatibility
   - Supports debug mode with structured logging (zerolog)

2. **Delve Tool** (`pkg/tools/delve/delve.go`)
   - Session-based persistent connections to Delve API
   - AutoHalt mode for safe inspection operations
   - Automatic session cleanup (30-minute timeout)
   - Full Delve command set support (20+ commands)

3. **Delve Client** (`pkg/tools/delve/client/`)
   - Wrapper around Delve RPC2 client
   - Session-aware context management
   - Command execution and formatting

### Dependencies

- `github.com/modelcontextprotocol/go-sdk v1.1.0` - MCP SDK
- `github.com/go-delve/delve v1.25.2` - Delve debugger
- `github.com/rs/zerolog v1.34.0` - Structured logging
- `github.com/go-playground/validator/v10 v10.27.0` - Input validation

## Delve Tool Features

**Session-Based Debugging:**
- Connect once, execute multiple commands
- Persistent connections with 30-minute idle timeout
- Three operation modes: connect, disconnect, command

**AutoHalt Pattern:**
- Automatically halts program before inspection
- Works with `--continue` flag for running programs
- Non-blocking connection test using `GetStateNonBlocking()`

**Full Command Set:**
- Execution control: continue, next, step, stepout, halt, restart
- Breakpoints: break, breakpoints, clear, clearall
- Variable inspection: print, locals, args, vars, whatis, set
- Stack & Context: stack, frame, list
- Goroutines: goroutines, goroutine
- Information: funcs, types, sources

**Input Parameters:**
- `session_id` (required): Unique session identifier
- `action`: connect, disconnect, or command (default: command)
- `host`: Delve server host (default: localhost)
- `port`: Delve server port (default: 2345)
- `command`: Delve command to execute
- `max_lines`: Pagination limit (default: 1000)
- `offset`: Line offset for pagination

## Build System

**Makefile targets:**
- `make build` - Build binary to `build/go-debugger-mcp`
- `make test` - Run test suites with coverage

## Integration

### Claude Code
```bash
claude mcp add go-debugger-mcp http://localhost:8899/mcp --transport http
```

## Usage Example

```bash
# Start your Go application with Delve
dlv debug --headless --listen=:2345 --api-version=2 --accept-multiclient --continue

# Connect from MCP client
delve SessionID=debug1 Action=connect Port=2345
delve SessionID=debug1 Action=command Command="break main.main"
delve SessionID=debug1 Action=command Command=continue
delve SessionID=debug1 Action=command Command=locals
delve SessionID=debug1 Action=disconnect
```

## Development Status

### Completed Features
- ✅ MCP server with StreamableHTTPHandler (Claude Code compatible)
- ✅ Delve API integration (rpc2 client)
- ✅ Session-based persistent connections
- ✅ AutoHalt mode for safe inspection
- ✅ Automatic session cleanup (30-minute timeout)
- ✅ Full Delve command set (20+ commands)
- ✅ Connection error handling with timeout
- ✅ Non-blocking connection test
- ✅ Variable formatting with nested display
- ✅ List command implementation
- ✅ Comprehensive input validation
- ✅ Complete test coverage
- ✅ Structured logging with debug mode

### Recent Changes
- Renamed from remote-debugger-mcp to go-debugger-mcp
- Removed server wrapper and tool interface abstractions
- Simplified to Delve-only implementation
- Fixed connection timeout using GetStateNonBlocking()
- Enhanced variable formatting (nested, recursive)
- Implemented list command
- Updated to StreamableHTTPHandler for Claude Code

### Technical Details

**Critical Delve API Behavior:**
- `GetState()` blocks when program is running (waits for halt/breakpoint/exit)
- `Continue()` returns immediately (non-blocking)
- `GetStateNonBlocking()` always returns immediately (use for connection tests)
- AutoHalt pattern required when using `--continue` flag
- Close() can block on failed connections (run in background goroutine)

**MCP Transport:**
- Uses `mcp.NewStreamableHTTPHandler()` for HTTP transport with SSE streaming
- Compatible with Claude Code's HTTP transport
- Sessions created via POST requests
- Responses streamed via Server-Sent Events (SSE)

## Security Considerations

This is a debugging tool that:
- Connects to Delve debugger instances
- Provides full access to program memory and execution
- Executes debugging commands

**Defensive Use Only:** Designed for legitimate debugging in development/testing environments.

**Input Validation:** All inputs validated using go-playground/validator to protect against injection.

**Network Security:** Only run Delve on trusted networks. Use firewalls/VPNs to restrict access.

## Testing

See `docs/TESTING.md` for complete testing guide.

**Quick test:**
```bash
# Terminal 1: Start test app with Delve
./scripts/start-delve-test.sh

# Terminal 2: Start MCP server
./scripts/start-mcp-server.sh

# Terminal 3: Run SDK test client
./scripts/test-mcp-client.sh
```
