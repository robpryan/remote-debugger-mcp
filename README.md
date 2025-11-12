# Go Debugger MCP Server

A Model Context Protocol (MCP) server that provides interactive Go debugging capabilities using the [Delve](https://github.com/go-delve/delve) debugger API.

## Features

- **Persistent Sessions**: Connect once, execute multiple debugging commands
- **Full Delve Command Set**: 20+ commands including breakpoints, stepping, variable inspection, and goroutine management
- **AutoHalt Mode**: Automatically halts program before inspection operations
- **Non-blocking Connections**: Works with both running and stopped programs
- **Streamable HTTP Transport**: Compatible with Claude Code and other MCP clients

## Quick Start

### 1. Install

```bash
git clone https://github.com/robpryan/go-debugger-mcp.git
cd go-debugger-mcp
make build
```

### 2. Run the MCP Server

```bash
./build/go-debugger-mcp --debug
```

The server will start on `http://localhost:8899/mcp`

### 3. Start Your Go Application with Delve

```bash
# Debug from source
dlv debug --headless --listen=:2345 --api-version=2 --accept-multiclient --continue

# Or attach to running process
dlv attach <PID> --headless --listen=:2345 --api-version=2 --accept-multiclient --continue
```

### 4. Connect from Claude Code

```bash
claude mcp add go-debugger-mcp http://localhost:8899/mcp --transport http
```

## Usage

The Delve MCP tool uses a session-based workflow:

```javascript
// 1. Connect to establish a session
{
  "session_id": "my-debug-session",
  "action": "connect",
  "host": "localhost",
  "port": 2345
}

// 2. Execute debugging commands
{
  "session_id": "my-debug-session",
  "action": "command",
  "command": "break main.main"
}

// 3. Disconnect when done (optional - sessions auto-expire after 30min)
{
  "session_id": "my-debug-session",
  "action": "disconnect"
}
```

### Available Commands

**Execution Control:**
- `continue`, `c` - Resume execution
- `next`, `n` - Step over
- `step`, `s` - Step into
- `stepout`, `so` - Step out
- `halt` - Pause execution
- `restart`, `r` - Restart program

**Breakpoints:**
- `break <location>` - Set breakpoint (e.g., `break main.main` or `break file.go:42`)
- `breakpoints` - List all breakpoints
- `clear <id>` - Remove breakpoint
- `clearall` - Remove all breakpoints

**Variable Inspection:**
- `print <expr>` - Evaluate and print expression
- `locals` - Show local variables
- `args` - Show function arguments
- `vars [regex]` - Show package variables
- `whatis <expr>` - Show type information
- `set <var>=<value>` - Modify variable

**Stack & Context:**
- `stack [depth]` - Show stack trace
- `frame <n>` - Switch to stack frame
- `list [location]` - Show source code

**Goroutines:**
- `goroutines [filter]` - List goroutines
- `goroutine <id>` - Switch to goroutine
- `goroutine <id> <cmd>` - Run command in goroutine context

**Information:**
- `funcs [regex]` - List functions
- `types [regex]` - List types
- `sources [regex]` - List source files

## Development

### Build

```bash
make build
```

### Test

```bash
# Start test servers
./scripts/start-delve-test.sh   # Terminal 1: Starts test app with Delve on :2345
./scripts/start-mcp-server.sh   # Terminal 2: Starts MCP server on :8899

# Run test client
./scripts/test-mcp-client.sh    # Terminal 3: Runs SDK-based test
```

See [docs/TESTING.md](docs/TESTING.md) for complete testing guide.

### Lint

```bash
make lint
```

### Run Tests

```bash
make test
```

## Documentation

- **[docs/ARCHITECTURE.md](docs/ARCHITECTURE.md)** - System architecture and implementation overview
- **[docs/DELVE_API_SPEC.md](docs/DELVE_API_SPEC.md)** - Delve API specification and command reference
- **[docs/DELVE_POC_FINDINGS.md](docs/DELVE_POC_FINDINGS.md)** - Critical technical findings and patterns
- **[docs/TESTING.md](docs/TESTING.md)** - Complete testing guide
- **[docs/PROJECT_NOTES.md](docs/PROJECT_NOTES.md)** - Project context and history

## MCP Transport

This server uses `NewStreamableHTTPHandler` for HTTP transport with SSE streaming, making it compatible with Claude Code and other MCP clients that support HTTP transport.

Clients should use HTTP POST to create sessions and receive responses via Server-Sent Events (SSE).

## Security Considerations

**Remote Debugging Risks:**
- Delve provides full access to your application's memory and execution
- Only run Delve on trusted networks
- Use `--accept-multiclient` cautiously in production
- Consider firewalls or VPNs to restrict access

**Local Development:**
This tool is designed for local development and testing. For production debugging, ensure proper network isolation and authentication.
