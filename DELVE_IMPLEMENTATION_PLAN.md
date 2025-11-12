# Delve API Implementation Plan

## Overview

This document outlines the detailed implementation plan for migrating the Delve MCP tool from CLI-based interaction to JSON-RPC API-based interaction.

**Related Documents:**
- [DELVE_API_SPEC.md](./DELVE_API_SPEC.md) - Requirements specification
- [PROJECT_NOTES.md](./PROJECT_NOTES.md) - Project overview

## API Protocol Analysis

### JSON-RPC API v2 Details

**Connection:**
- Protocol: JSON-RPC 1.0 over streaming socket (not HTTP)
- Default endpoint: `127.0.0.1:8181` (configurable)
- Launch command: `dlv debug --headless --api-version=2 --listen=127.0.0.1:8181`
- Multi-client support: `--accept-multiclient` flag

**Message Structure:**
```json
{
  "method": "RPCServer.MethodName",
  "params": [<single-parameter-object>],
  "id": <request-id>
}
```

**Response Structure:**
```json
{
  "id": <request-id>,
  "result": <response-object>,
  "error": <error-object-if-any>
}
```

### Available RPC Methods Mapping

Based on `service/rpc2.RPCServer`:

#### 1. Execution Control

| CLI Command | RPC Method | Parameters | Returns |
|-------------|------------|------------|---------|
| `continue` | `RPCServer.Command` | `{"Name": "continue"}` | `State` |
| `next` | `RPCServer.Command` | `{"Name": "next"}` | `State` |
| `step` | `RPCServer.Command` | `{"Name": "step"}` | `State` |
| `stepout` | `RPCServer.Command` | `{"Name": "stepOut"}` | `State` |
| `restart` | `RPCServer.Restart` | `{}` | `State` |
| `halt` | `RPCServer.Command` | `{"Name": "halt"}` | `State` |

#### 2. Breakpoint Management

| CLI Command | RPC Method | Parameters | Returns |
|-------------|------------|------------|---------|
| `break <loc>` | `RPCServer.CreateBreakpoint` | `{"Breakpoint": {"addr": ..., "file": ..., "line": ...}}` | `Breakpoint` |
| `breakpoints` | `RPCServer.ListBreakpoints` | `{}` | `[]Breakpoint` |
| `clear <id>` | `RPCServer.ClearBreakpoint` | `{"Id": <id>}` | `Breakpoint` |
| `clearall` | `RPCServer.ClearBreakpoint` | Loop through all IDs | Multiple calls |
| `condition <id> <expr>` | `RPCServer.AmendBreakpoint` | `{"Breakpoint": {"id": <id>, "Cond": "<expr>"}}` | `Breakpoint` |

**Note:** Need to use `RPCServer.FindLocation` to resolve location strings to addresses.

#### 3. Variable Inspection

| CLI Command | RPC Method | Parameters | Returns |
|-------------|------------|------------|---------|
| `print <expr>` | `RPCServer.Eval` | `{"Scope": {...}, "Expr": "<expr>"}` | `Variable` |
| `locals` | `RPCServer.ListLocalVars` | `{"Scope": {...}}` | `[]Variable` |
| `args` | `RPCServer.ListFunctionArgs` | `{"Scope": {...}}` | `[]Variable` |
| `vars <regex>` | `RPCServer.ListPackageVars` | `{"Filter": "<regex>"}` | `[]Variable` |
| `whatis <expr>` | `RPCServer.Eval` | `{"Scope": {...}, "Expr": "<expr>"}` | `Variable` (check `.Type`) |
| `set <var> = <val>` | `RPCServer.SetVariable` | `{"Scope": {...}, "Symbol": "<var>", "Value": "<val>"}` | `Variable` |

**Scope Structure:**
```go
type EvalScope struct {
    GoroutineID int64
    Frame       int
}
```

#### 4. Stack & Context

| CLI Command | RPC Method | Parameters | Returns |
|-------------|------------|------------|---------|
| `stack` | `RPCServer.Stacktrace` | `{"Id": <goroutine-id>, "Depth": <depth>}` | `[]Stackframe` |
| `frame <n>` | N/A (client-side state) | Update local frame context | Local state change |
| `list` | `RPCServer.ListSources` | `{"Filter": "<pattern>"}` | `[]string` (source files) |

**Note:** Frame switching is client-side state that affects the `Scope` parameter in subsequent calls.

#### 5. Goroutine Management

| CLI Command | RPC Method | Parameters | Returns |
|-------------|------------|------------|---------|
| `goroutines` | `RPCServer.ListGoroutines` | `{"Start": 0, "Count": <limit>}` | `[]Goroutine` |
| `goroutine <id>` | N/A (client-side state) | Update local goroutine context | Local state change |

**Note:** Goroutine switching is client-side state that affects the `GoroutineID` in `Scope` parameters.

#### 6. Symbol Navigation

| CLI Command | RPC Method | Parameters | Returns |
|-------------|------------|------------|---------|
| `funcs <regex>` | `RPCServer.ListFunctions` | `{"Filter": "<regex>"}` | `[]string` |
| `types <regex>` | `RPCServer.ListTypes` | `{"Filter": "<regex>"}` | `[]string` |
| `sources <regex>` | `RPCServer.ListSources` | `{"Filter": "<pattern>"}` | `[]string` |

#### 7. Advanced Features

| CLI Command | RPC Method | Parameters | Returns |
|-------------|------------|------------|---------|
| `watch <expr>` | `RPCServer.CreateWatchpoint` | `{"Scope": {...}, "Expr": "<expr>"}` | `Breakpoint` |
| `examinemem <addr>` | `RPCServer.ExamineMemory` | `{"Address": <addr>, "Length": <len>}` | `[]byte` |
| `disassemble` | `RPCServer.Disassemble` | `{"Scope": {...}, "StartPC": ..., "EndPC": ...}` | `AsmInstructions` |
| `regs` | `RPCServer.ListRegisters` | `{"ThreadID": <id>}` | `Registers` |

#### 8. State & Information

| CLI Command | RPC Method | Parameters | Returns |
|-------------|------------|------------|---------|
| N/A (get state) | `RPCServer.GetState` | `{}` | `State` |
| N/A (detach) | `RPCServer.Detach` | `{"Kill": false}` | `null` |

## Architecture Design

### Component Structure

```
pkg/tools/delve/
├── delve.go              # MCP tool interface (existing)
├── delve_test.go         # Tests (existing)
├── client/
│   ├── client.go         # JSON-RPC client implementation
│   ├── types.go          # Request/Response types
│   ├── commands.go       # High-level command wrappers
│   └── client_test.go    # Client tests
└── session/
    ├── session.go        # Session manager (enhanced)
    └── session_test.go   # Session tests
```

### New Package: `pkg/tools/delve/client`

#### `client.go` - JSON-RPC Client

```go
package client

import (
    "encoding/json"
    "fmt"
    "net"
    "sync"
)

// DelveClient manages JSON-RPC connection to Delve server
type DelveClient struct {
    conn      net.Conn
    mu        sync.Mutex
    requestID int64
}

// Connect establishes connection to Delve server
func Connect(host string, port int) (*DelveClient, error) {
    addr := fmt.Sprintf("%s:%d", host, port)
    conn, err := net.Dial("tcp", addr)
    if err != nil {
        return nil, fmt.Errorf("failed to connect to delve: %w", err)
    }

    return &DelveClient{
        conn:      conn,
        requestID: 0,
    }, nil
}

// Call makes a JSON-RPC call
func (c *DelveClient) Call(method string, params interface{}) (json.RawMessage, error) {
    c.mu.Lock()
    defer c.mu.Unlock()

    c.requestID++
    request := JSONRPCRequest{
        Method: method,
        Params: []interface{}{params},
        ID:     c.requestID,
    }

    // Send request
    if err := json.NewEncoder(c.conn).Encode(request); err != nil {
        return nil, fmt.Errorf("failed to send request: %w", err)
    }

    // Read response
    var response JSONRPCResponse
    if err := json.NewDecoder(c.conn).Decode(&response); err != nil {
        return nil, fmt.Errorf("failed to read response: %w", err)
    }

    if response.Error != nil {
        return nil, fmt.Errorf("rpc error: %s", response.Error.Message)
    }

    return response.Result, nil
}

// Close closes the connection
func (c *DelveClient) Close() error {
    return c.conn.Close()
}
```

#### `types.go` - Request/Response Types

```go
package client

import "encoding/json"

type JSONRPCRequest struct {
    Method string        `json:"method"`
    Params []interface{} `json:"params"`
    ID     int64         `json:"id"`
}

type JSONRPCResponse struct {
    ID     int64           `json:"id"`
    Result json.RawMessage `json:"result"`
    Error  *JSONRPCError   `json:"error"`
}

type JSONRPCError struct {
    Code    int    `json:"code"`
    Message string `json:"message"`
}

// Delve API types (from service/api package)
type DebuggerState struct {
    Running       bool        `json:"Running"`
    CurrentThread *Thread     `json:"currentThread"`
    Breakpoint    *Breakpoint `json:"Breakpoint"`
    // ... more fields
}

type Breakpoint struct {
    ID        int    `json:"id"`
    Addr      uint64 `json:"addr"`
    File      string `json:"file"`
    Line      int    `json:"line"`
    Cond      string `json:"Cond"`
    HitCount  int    `json:"totalHitCount"`
    // ... more fields
}

type Variable struct {
    Name     string      `json:"name"`
    Type     string      `json:"type"`
    Value    string      `json:"value"`
    Kind     int         `json:"kind"`
    Children []Variable  `json:"children"`
    // ... more fields
}

type Goroutine struct {
    ID            int64  `json:"id"`
    CurrentLoc    Location `json:"currentLoc"`
    UserCurrentLoc Location `json:"userCurrentLoc"`
    GoStatementLoc Location `json:"goStatementLoc"`
    // ... more fields
}

type Stackframe struct {
    PC       uint64    `json:"pc"`
    File     string    `json:"file"`
    Line     int       `json:"line"`
    Function *Function `json:"function"`
    // ... more fields
}

type EvalScope struct {
    GoroutineID int64 `json:"GoroutineID"`
    Frame       int   `json:"Frame"`
}
```

#### `commands.go` - High-Level Command Wrappers

```go
package client

import (
    "encoding/json"
    "fmt"
)

// ExecutionCommands provides execution control methods
type ExecutionCommands struct {
    client *DelveClient
}

func (e *ExecutionCommands) Continue() (*DebuggerState, error) {
    result, err := e.client.Call("RPCServer.Command", map[string]string{
        "Name": "continue",
    })
    if err != nil {
        return nil, err
    }

    var state DebuggerState
    if err := json.Unmarshal(result, &state); err != nil {
        return nil, err
    }
    return &state, nil
}

func (e *ExecutionCommands) Next() (*DebuggerState, error) {
    result, err := e.client.Call("RPCServer.Command", map[string]string{
        "Name": "next",
    })
    if err != nil {
        return nil, err
    }

    var state DebuggerState
    if err := json.Unmarshal(result, &state); err != nil {
        return nil, err
    }
    return &state, nil
}

// Similar methods for Step, StepOut, Halt, Restart...

// BreakpointCommands provides breakpoint management methods
type BreakpointCommands struct {
    client *DelveClient
}

func (b *BreakpointCommands) CreateBreakpoint(location string) (*Breakpoint, error) {
    // First, resolve the location
    locResult, err := b.client.Call("RPCServer.FindLocation", map[string]interface{}{
        "Scope": EvalScope{GoroutineID: -1, Frame: 0},
        "Loc":   location,
    })
    if err != nil {
        return nil, fmt.Errorf("failed to find location: %w", err)
    }

    var locations []Location
    if err := json.Unmarshal(locResult, &locations); err != nil {
        return nil, err
    }
    if len(locations) == 0 {
        return nil, fmt.Errorf("location not found: %s", location)
    }

    // Create breakpoint at resolved location
    bpResult, err := b.client.Call("RPCServer.CreateBreakpoint", map[string]interface{}{
        "Breakpoint": map[string]interface{}{
            "addr": locations[0].PC,
        },
    })
    if err != nil {
        return nil, err
    }

    var bp Breakpoint
    if err := json.Unmarshal(bpResult, &bp); err != nil {
        return nil, err
    }
    return &bp, nil
}

func (b *BreakpointCommands) ListBreakpoints() ([]Breakpoint, error) {
    result, err := b.client.Call("RPCServer.ListBreakpoints", map[string]interface{}{})
    if err != nil {
        return nil, err
    }

    var bps []Breakpoint
    if err := json.Unmarshal(result, &bps); err != nil {
        return nil, err
    }
    return bps, nil
}

// VariableCommands provides variable inspection methods
type VariableCommands struct {
    client *DelveClient
}

func (v *VariableCommands) Eval(expr string, scope EvalScope) (*Variable, error) {
    result, err := v.client.Call("RPCServer.Eval", map[string]interface{}{
        "Scope": scope,
        "Expr":  expr,
    })
    if err != nil {
        return nil, err
    }

    var variable Variable
    if err := json.Unmarshal(result, &variable); err != nil {
        return nil, err
    }
    return &variable, nil
}

func (v *VariableCommands) ListLocalVars(scope EvalScope) ([]Variable, error) {
    result, err := v.client.Call("RPCServer.ListLocalVars", map[string]interface{}{
        "Scope": scope,
    })
    if err != nil {
        return nil, err
    }

    var vars []Variable
    if err := json.Unmarshal(result, &vars); err != nil {
        return nil, err
    }
    return vars, nil
}

// GoroutineCommands provides goroutine management
type GoroutineCommands struct {
    client *DelveClient
}

func (g *GoroutineCommands) ListGoroutines(start, count int) ([]Goroutine, error) {
    result, err := g.client.Call("RPCServer.ListGoroutines", map[string]interface{}{
        "Start": start,
        "Count": count,
    })
    if err != nil {
        return nil, err
    }

    var goroutines []Goroutine
    if err := json.Unmarshal(result, &goroutines); err != nil {
        return nil, err
    }
    return goroutines, nil
}

// StackCommands provides stack inspection
type StackCommands struct {
    client *DelveClient
}

func (s *StackCommands) Stacktrace(goroutineID int64, depth int) ([]Stackframe, error) {
    result, err := s.client.Call("RPCServer.Stacktrace", map[string]interface{}{
        "Id":    goroutineID,
        "Depth": depth,
    })
    if err != nil {
        return nil, err
    }

    var frames []Stackframe
    if err := json.Unmarshal(result, &frames); err != nil {
        return nil, err
    }
    return frames, nil
}

// DelveCommands aggregates all command sets
type DelveCommands struct {
    Execution  *ExecutionCommands
    Breakpoint *BreakpointCommands
    Variable   *VariableCommands
    Goroutine  *GoroutineCommands
    Stack      *StackCommands
}

func NewDelveCommands(client *DelveClient) *DelveCommands {
    return &DelveCommands{
        Execution:  &ExecutionCommands{client: client},
        Breakpoint: &BreakpointCommands{client: client},
        Variable:   &VariableCommands{client: client},
        Goroutine:  &GoroutineCommands{client: client},
        Stack:      &StackCommands{client: client},
    }
}
```

### Enhanced Session Manager

Update `pkg/tools/delve/session/session.go`:

```go
package session

import (
    "fmt"
    "sync"
    "time"

    "github.com/tb0hdan/remote-debugger-mcp/pkg/tools/delve/client"
)

type Session struct {
    ID              string
    Client          *client.DelveClient
    Commands        *client.DelveCommands
    CurrentGoroutine int64
    CurrentFrame    int
    LastActivity    time.Time
    mu              sync.RWMutex
}

type Manager struct {
    sessions map[string]*Session
    mu       sync.RWMutex
    timeout  time.Duration
}

func NewManager(timeout time.Duration) *Manager {
    m := &Manager{
        sessions: make(map[string]*Session),
        timeout:  timeout,
    }
    go m.cleanupLoop()
    return m
}

func (m *Manager) Connect(sessionID, host string, port int) error {
    m.mu.Lock()
    defer m.mu.Unlock()

    if _, exists := m.sessions[sessionID]; exists {
        return fmt.Errorf("session already exists: %s", sessionID)
    }

    client, err := client.Connect(host, port)
    if err != nil {
        return err
    }

    m.sessions[sessionID] = &Session{
        ID:              sessionID,
        Client:          client,
        Commands:        client.NewDelveCommands(client),
        CurrentGoroutine: -1, // -1 means current goroutine
        CurrentFrame:    0,
        LastActivity:    time.Now(),
    }

    return nil
}

func (m *Manager) Get(sessionID string) (*Session, error) {
    m.mu.RLock()
    defer m.mu.RUnlock()

    session, exists := m.sessions[sessionID]
    if !exists {
        return nil, fmt.Errorf("session not found: %s", sessionID)
    }

    session.mu.Lock()
    session.LastActivity = time.Now()
    session.mu.Unlock()

    return session, nil
}

func (m *Manager) Disconnect(sessionID string) error {
    m.mu.Lock()
    defer m.mu.Unlock()

    session, exists := m.sessions[sessionID]
    if !exists {
        return fmt.Errorf("session not found: %s", sessionID)
    }

    // Detach from debugger
    _, _ = session.Client.Call("RPCServer.Detach", map[string]bool{"Kill": false})

    // Close connection
    if err := session.Client.Close(); err != nil {
        return err
    }

    delete(m.sessions, sessionID)
    return nil
}

// GetScope returns the current evaluation scope for the session
func (s *Session) GetScope() client.EvalScope {
    s.mu.RLock()
    defer s.mu.RUnlock()

    return client.EvalScope{
        GoroutineID: s.CurrentGoroutine,
        Frame:       s.CurrentFrame,
    }
}

// SetGoroutine changes the current goroutine context
func (s *Session) SetGoroutine(goroutineID int64) {
    s.mu.Lock()
    defer s.mu.Unlock()

    s.CurrentGoroutine = goroutineID
    s.CurrentFrame = 0 // Reset frame when switching goroutines
}

// SetFrame changes the current frame context
func (s *Session) SetFrame(frame int) {
    s.mu.Lock()
    defer s.mu.Unlock()

    s.CurrentFrame = frame
}
```

### Updated MCP Tool Interface

Update `pkg/tools/delve/delve.go`:

```go
// Execute command in session
case "command":
    session, err := sessionMgr.Get(sessionID)
    if err != nil {
        return formatError(err)
    }

    // Route command to appropriate handler
    output, err := d.executeCommand(session, req.Command, req.Args)
    if err != nil {
        return formatError(err)
    }

    return formatSuccess(sessionID, output)

// executeCommand routes commands to the appropriate API methods
func (d *DelveTool) executeCommand(session *Session, command string, args []string) (string, error) {
    scope := session.GetScope()

    switch command {
    case "continue", "c":
        state, err := session.Commands.Execution.Continue()
        return formatState(state), err

    case "next", "n":
        state, err := session.Commands.Execution.Next()
        return formatState(state), err

    case "step", "s":
        state, err := session.Commands.Execution.Step()
        return formatState(state), err

    case "break", "b":
        if len(args) == 0 {
            return "", fmt.Errorf("break requires a location")
        }
        bp, err := session.Commands.Breakpoint.CreateBreakpoint(args[0])
        return formatBreakpoint(bp), err

    case "breakpoints", "bp":
        bps, err := session.Commands.Breakpoint.ListBreakpoints()
        return formatBreakpoints(bps), err

    case "print", "p":
        if len(args) == 0 {
            return "", fmt.Errorf("print requires an expression")
        }
        variable, err := session.Commands.Variable.Eval(args[0], scope)
        return formatVariable(variable), err

    case "locals":
        vars, err := session.Commands.Variable.ListLocalVars(scope)
        return formatVariables(vars), err

    case "goroutines":
        grs, err := session.Commands.Goroutine.ListGoroutines(0, 100)
        return formatGoroutines(grs), err

    case "stack", "bt":
        frames, err := session.Commands.Stack.Stacktrace(scope.GoroutineID, 50)
        return formatStacktrace(frames), err

    case "goroutine":
        if len(args) == 0 {
            return "", fmt.Errorf("goroutine requires an ID")
        }
        id, err := strconv.ParseInt(args[0], 10, 64)
        if err != nil {
            return "", fmt.Errorf("invalid goroutine ID: %w", err)
        }
        session.SetGoroutine(id)
        return fmt.Sprintf("Switched to goroutine %d", id), nil

    case "frame":
        if len(args) == 0 {
            return "", fmt.Errorf("frame requires an index")
        }
        idx, err := strconv.Atoi(args[0])
        if err != nil {
            return "", fmt.Errorf("invalid frame index: %w", err)
        }
        session.SetFrame(idx)
        return fmt.Sprintf("Switched to frame %d", idx), nil

    default:
        return "", fmt.Errorf("unknown command: %s", command)
    }
}
```

## Implementation Phases

### Phase 1: Core Infrastructure (Week 1)
- [ ] Create `pkg/tools/delve/client` package structure
- [ ] Implement `client.go` - JSON-RPC client
- [ ] Implement `types.go` - Request/Response types
- [ ] Write unit tests for JSON-RPC communication
- [ ] Test connection to real Delve server

**Deliverable:** Working JSON-RPC client that can connect and make basic calls

### Phase 2: Core Commands (Week 2)
- [ ] Implement `ExecutionCommands` (continue, next, step, stepout, halt)
- [ ] Implement `BreakpointCommands` (create, list, clear)
- [ ] Implement `VariableCommands` (eval, locals, args, vars)
- [ ] Implement `StackCommands` (stacktrace)
- [ ] Write integration tests for each command set

**Deliverable:** Core debugging commands working via API

### Phase 3: Session Enhancement (Week 3)
- [ ] Update `Session` struct with client and context state
- [ ] Implement goroutine/frame context tracking
- [ ] Update `Manager` to use API connections
- [ ] Migrate session tests to API-based approach
- [ ] Add timeout and cleanup for API sessions

**Deliverable:** Enhanced session manager with API support

### Phase 4: MCP Tool Integration (Week 4)
- [ ] Update `delve.go` to use API client
- [ ] Implement command routing and parsing
- [ ] Add response formatting helpers
- [ ] Maintain backward compatibility (if needed)
- [ ] Update tool tests

**Deliverable:** MCP tool using API instead of CLI

### Phase 5: Advanced Features (Week 5-6)
- [ ] Implement `GoroutineCommands` (list, switch)
- [ ] Implement symbol navigation (funcs, types, sources)
- [ ] Implement watchpoints
- [ ] Implement memory examination
- [ ] Implement disassembly
- [ ] Add comprehensive error handling

**Deliverable:** Full feature parity with spec

### Phase 6: Testing & Documentation (Week 7)
- [ ] End-to-end integration tests
- [ ] Performance testing
- [ ] Error handling edge cases
- [ ] Update PROJECT_NOTES.md
- [ ] Update README.md
- [ ] Create usage examples
- [ ] Write migration guide

**Deliverable:** Production-ready implementation

## Testing Strategy

### Unit Tests
```go
// pkg/tools/delve/client/client_test.go
func TestJSONRPCCall(t *testing.T) {
    // Mock server
    server := startMockDelveServer(t)
    defer server.Close()

    client, err := Connect("localhost", server.Port)
    require.NoError(t, err)
    defer client.Close()

    // Test call
    result, err := client.Call("RPCServer.GetState", map[string]interface{}{})
    require.NoError(t, err)
    assert.NotNil(t, result)
}
```

### Integration Tests
```go
// pkg/tools/delve/delve_integration_test.go
func TestDelveAPIWorkflow(t *testing.T) {
    // Start real delve server
    dlv := startTestDelveServer(t, "./testdata/simple.go")
    defer dlv.Stop()

    // Connect via MCP tool
    tool := NewDelveTool()

    // Connect
    resp := tool.Execute(DelveRequest{
        SessionID: "test1",
        Action:    "connect",
        Host:      "localhost",
        Port:      dlv.Port,
    })
    require.True(t, resp.Success)

    // Set breakpoint
    resp = tool.Execute(DelveRequest{
        SessionID: "test1",
        Action:    "command",
        Command:   "break",
        Args:      []string{"main.main"},
    })
    require.True(t, resp.Success)

    // Continue
    resp = tool.Execute(DelveRequest{
        SessionID: "test1",
        Action:    "command",
        Command:   "continue",
    })
    require.True(t, resp.Success)

    // Inspect locals
    resp = tool.Execute(DelveRequest{
        SessionID: "test1",
        Action:    "command",
        Command:   "locals",
    })
    require.True(t, resp.Success)
    assert.Contains(t, resp.Output, "expected_var")
}
```

## Migration Path

### Option 1: Hard Cutover
- Remove CLI code entirely
- Replace with API implementation
- Bump major version (2.0.0)

**Pros:** Clean codebase, no maintenance burden
**Cons:** Breaking change for existing users

### Option 2: Gradual Migration
- Add API implementation alongside CLI
- Add configuration flag to choose mode
- Deprecate CLI mode after testing period
- Remove CLI in future version

**Pros:** Smooth transition, user testing
**Cons:** More code to maintain temporarily

### Recommendation: Option 2 (Gradual Migration)

Add config parameter:
```go
type DelveRequest struct {
    // ...
    UseAPI bool `json:"use_api" validate:"omitempty"` // default: true (API mode)
}
```

Transition plan:
1. v1.2.0: Add API support, default to API, CLI available with `use_api: false`
2. v1.3.0: Deprecate CLI mode, warn users
3. v2.0.0: Remove CLI mode entirely

## Performance Considerations

### Expected Improvements
- **Connection overhead:** Eliminated (persistent connection vs spawning processes)
- **Parsing overhead:** Eliminated (structured JSON vs text parsing)
- **Latency:** Reduced (~10-50ms per command saved)

### Potential Issues
- **Memory:** Persistent connections use more memory than CLI spawns
- **Concurrency:** Need mutex protection for shared connections

### Mitigation
- Implement connection pooling if needed
- Add configurable session limits
- Monitor memory usage in production

## Security Considerations

### Current CLI Approach
- Spawns external process (potential command injection)
- Text-based communication (parsing vulnerabilities)

### API Approach
- Direct TCP connection (network exposure)
- Structured protocol (less injection risk)
- Still requires input validation

### Recommendations
1. Maintain input validation for all parameters
2. Use TLS for production deployments (if Delve supports)
3. Implement connection authentication (if Delve supports)
4. Rate limiting on API calls
5. Timeout all network operations

## Open Questions & Decisions Needed

1. **CLI Backward Compatibility:** Keep CLI mode as fallback? → **Recommend: Yes, with deprecation path**

2. **Error Handling:** How verbose should API errors be? → **Recommend: Structured error objects with context**

3. **State Synchronization:** How to handle state changes from other clients (multi-client mode)? → **Recommend: Add state refresh command**

4. **Command Parsing:** Client-side or server-side? → **Recommend: Client-side for flexibility**

5. **Response Formatting:** Preserve CLI-like text output or use structured JSON? → **Recommend: Both, configurable**

## Success Metrics

1. **Functionality:** All Priority 1 & 2 commands working (13 core commands)
2. **Performance:** <50ms average latency for simple commands
3. **Reliability:** <0.1% error rate in normal operation
4. **Test Coverage:** >80% code coverage
5. **Documentation:** Complete API usage guide

## Timeline

| Week | Phase | Deliverable |
|------|-------|-------------|
| 1 | Infrastructure | JSON-RPC client |
| 2 | Core Commands | Execution, breakpoints, variables |
| 3 | Session | Enhanced session manager |
| 4 | Integration | MCP tool updated |
| 5-6 | Advanced | Full feature set |
| 7 | Testing & Docs | Production ready |

**Total Estimated Time:** 7 weeks

## Next Steps

1. Review and approve this implementation plan
2. Create feature branch: `feature/delve-api-client`
3. Begin Phase 1: Core Infrastructure
4. Set up CI/CD for automated testing
5. Create tracking issues for each phase

## References

- Delve API Documentation: https://github.com/go-delve/delve/tree/master/Documentation/api
- Delve service/rpc2 Package: https://pkg.go.dev/github.com/go-delve/delve/service/rpc2
- JSON-RPC 1.0 Spec: https://www.jsonrpc.org/specification_v1
- Current Implementation: [pkg/tools/delve/delve.go](./pkg/tools/delve/delve.go)

---

**Document Status:** Draft for Review
**Last Updated:** 2025-01-12
**Next Review:** After Phase 1 completion
