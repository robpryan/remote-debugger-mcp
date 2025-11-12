# Delve MCP Architecture

## Overview

The Delve MCP tool provides remote debugging capabilities using the Delve JSON-RPC API.

## Implementation Status

✅ **COMPLETE** - All features implemented and tested.

## Architecture

```
MCP Client (Claude/Inspector)
    ↓ SSE Transport
MCP Server (cmd/debugger)
    ↓ mcp.NewSSEHandler
Delve Tool (pkg/tools/delve)
    ↓ Session Management
Delve Client (pkg/tools/delve/client)
    ↓ JSON-RPC
Delve Server (dlv headless)
    ↓ Debug API
Target Application
```

## Key Components

### Server (cmd/debugger/main.go)
- Uses `mcp.NewSSEHandler()` for SSE transport
- Registers Delve tool
- Serves on http://localhost:8899/mcp

### Delve Tool (pkg/tools/delve/)
- **delve.go** - MCP tool implementation with session management
- **client/client.go** - Session-aware wrapper around `rpc2.RPCClient`
- **client/commands.go** - Command routing (20+ commands supported)

### Session Management
- Persistent Delve API connections
- 30-minute idle timeout
- AutoHalt mode for safe inspection
- Goroutine and frame context tracking

## Supported Commands

**Execution**: continue, next, step, stepout, halt, restart  
**Breakpoints**: break, clear, clearall, breakpoints  
**Inspection**: print, locals, args, vars, stack, goroutines  
**Navigation**: frame, goroutine, threads  
**Info**: list, funcs, types, sources

See `docs/DELVE_API_SPEC.md` for complete command documentation.

## Critical Implementation Details

See `docs/DELVE_POC_FINDINGS.md` for important technical findings:
- `GetState()` blocking behavior
- `Continue()` non-blocking behavior
- AutoHalt pattern
- Protocol requirements

## Testing

See `docs/TESTING.md` for complete testing guide.

Quick start:
```bash
./scripts/start-delve-test.sh  # Start Delve server
./scripts/start-mcp-server.sh  # Start MCP server
./scripts/test-mcp-client.sh   # Run tests
```
