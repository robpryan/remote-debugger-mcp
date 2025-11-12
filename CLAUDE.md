# Claude Instructions

## On Startup
Always read `docs/PROJECT_NOTES.md` first to understand the project context and current state.

## Project Commands
- Build: `make build`
- Test: `make test`

## Documentation Structure

### Core Documentation (docs/)
- **PROJECT_NOTES.md** - Project overview, goals, and implementation status
- **ARCHITECTURE.md** - System architecture and implementation overview
- **DELVE_API_SPEC.md** - Delve API specification and command reference
- **DELVE_POC_FINDINGS.md** - Critical technical findings (blocking behavior, AutoHalt pattern, etc.)
- **TESTING.md** - Complete testing guide with examples
- **VERIFICATION_RESULTS.md** - Verification test results

### Important Implementation Details

#### Delve API Behavior (see docs/DELVE_POC_FINDINGS.md)
- `GetState()` **blocks** when program is running (waits for breakpoint/halt/exit)
- `Continue()` returns immediately (non-blocking)
- Always use AutoHalt pattern when using `--continue` flag
- Protocol: JSON-RPC mode requires flags before executable path

#### MCP Transport
- Server uses `mcp.NewStreamableHTTPHandler()` for streamable HTTP transport
- Compatible with Claude Code's HTTP transport
- Clients should use `mcp.StreamableClientTransport` from official SDK
- Sessions created via POST requests, responses streamed via SSE
- Tool descriptions include comprehensive usage instructions for LLMs

## Testing
See `docs/TESTING.md` for complete testing guide.

Quick test:
```bash
# Start servers
./scripts/start-delve-test.sh    # Delve on :2345
./scripts/start-mcp-server.sh    # MCP on :8899

# Run test client
./scripts/test-mcp-client.sh
```
