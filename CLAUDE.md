# Claude Instructions

## On Startup
Always read `docs/PROJECT_NOTES.md` first to understand the project context and current state.

## Project Commands
- Lint: `make lint`
- Test: `make test`
- Build: `make build`

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

#### MCP Transport (updated)
- Server uses `mcp.NewSSEHandler()` for proper SSE transport
- Clients should use `mcp.SSEClientTransport` from official SDK
- Sessions maintained via persistent HTTP connections with SSE

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
