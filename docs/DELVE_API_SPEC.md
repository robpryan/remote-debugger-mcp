# Delve API Integration Specification

## Overview

This document specifies the requirements and implementation plan for migrating the Delve MCP tool from CLI-based interaction to direct API-based interaction with the Delve debugger.

**Current State:** The Delve tool spawns `dlv connect` processes and communicates via stdin/stdout.

**Target State:** The Delve tool will communicate directly with the Delve DAP/JSON-RPC API server.

**Benefits:**
- More reliable communication (no CLI parsing)
- Better error handling and state management
- Access to structured responses
- Reduced overhead (no process spawning per command)
- Better support for concurrent operations

## Version Information

- **Document Version:** 1.0.0
- **Target Delve Version:** 1.20.0+
- **MCP Server Version:** 1.1.0+

## Requirements

### 1. Core Execution Control (Must Have - Priority 1)

| Command | CLI Syntax | Description | Use Case |
|---------|------------|-------------|----------|
| `continue` | `c`, `continue` | Continue execution until next breakpoint | Resume program flow |
| `next` | `n`, `next` | Step over to next source line | Step through code |
| `step` | `s`, `step` | Step into function calls | Dive into function implementation |
| `stepout` | `so`, `stepout` | Step out of current function | Exit current function |
| `restart` | `r`, `restart` | Restart the program | Reset debugging session |

**API Requirements:**
- Must support async execution state changes
- Must provide execution state feedback (running, stopped, exited)
- Must handle goroutine context for step commands

### 2. Breakpoint Management (Must Have - Priority 1)

| Command | CLI Syntax | Description | Use Case |
|---------|------------|-------------|----------|
| `break` | `b <location>`, `break <location>` | Set breakpoint at location | Stop at specific code location |
| `breakpoints` | `bp`, `breakpoints` | List all breakpoints | View active breakpoints |
| `clear` | `clear <id>` | Remove breakpoint by ID | Remove specific breakpoint |
| `clearall` | `clearall` | Remove all breakpoints | Clean slate debugging |
| `condition` | `condition <id> <expr>` | Add condition to breakpoint | Conditional stopping |

**Location Formats:**
- Function name: `main.main`, `pkg.FuncName`
- File:line: `file.go:42`
- Address: `*0xdeadbeef`

**API Requirements:**
- Breakpoint creation with location parsing
- Conditional breakpoint support
- List breakpoints with full metadata (ID, location, hit count, condition)
- Delete individual or all breakpoints

### 3. Variable Inspection (Must Have - Priority 1)

| Command | CLI Syntax | Description | Use Case |
|---------|------------|-------------|----------|
| `print` | `p <expr>`, `print <expr>` | Print value of expression | Inspect variable value |
| `locals` | `locals` | Show all local variables | View local scope |
| `args` | `args` | Show function arguments | View function parameters |
| `vars` | `vars <regex>` | Show package variables | View package-level state |
| `whatis` | `whatis <expr>` | Show type of expression | Determine variable type |
| `set` | `set <var> = <value>` | Modify variable value | Change runtime state |

**API Requirements:**
- Expression evaluation in current context
- Structured variable representation (type, value, children)
- Support for complex types (structs, slices, maps, interfaces)
- Variable modification support
- Scope-aware variable listing (local, args, package)

### 4. Stack & Context (Must Have - Priority 1)

| Command | CLI Syntax | Description | Use Case |
|---------|------------|-------------|----------|
| `stack` | `bt`, `stack` | Show stack trace | View call stack |
| `frame` | `frame <n>` | Select stack frame | Switch context |
| `list` | `ls`, `list` | Show source code around current line | View code context |

**API Requirements:**
- Full stack trace retrieval
- Frame selection and context switching
- Source code retrieval with line numbers
- Support for listing at arbitrary locations

### 5. Goroutine Management (Must Have - Priority 2)

| Command | CLI Syntax | Description | Use Case |
|---------|------------|-------------|----------|
| `goroutines` | `goroutines` | List all goroutines | View concurrent execution |
| `goroutine` | `goroutine <id>` | Switch to specific goroutine | Change goroutine context |
| `goroutine cmd` | `goroutine <id> <cmd>` | Run command in goroutine context | Execute in specific goroutine |

**API Requirements:**
- List all goroutines with state (running, waiting, etc.)
- Goroutine context switching
- Per-goroutine stack traces
- Goroutine filtering by location or state

### 6. Information & Help (Must Have - Priority 2)

| Command | CLI Syntax | Description | Use Case |
|---------|------------|-------------|----------|
| `help` | `help`, `help <cmd>` | List commands or show help | User guidance |
| `funcs` | `funcs <regex>` | List functions | Find function by name |
| `types` | `types <regex>` | List types | Find type definitions |
| `sources` | `sources <regex>` | List source files | Find source files |

**API Requirements:**
- Symbol table access for functions, types, sources
- Regular expression filtering
- Complete symbol metadata

### 7. Advanced Breakpoint Features (Nice to Have - Priority 3)

| Command | CLI Syntax | Description | Use Case |
|---------|------------|-------------|----------|
| `on` | `on <bp-id> <cmd>` | Execute command when breakpoint hits | Automated inspection |
| `trace` | `t <location>`, `trace <location>` | Tracepoint - log without stopping | Non-intrusive monitoring |
| `watch` | `watch <expr>` | Break when expression changes | Data change detection |

**API Requirements:**
- Tracepoint support (breakpoint with continue)
- Watchpoint/data breakpoint support
- Breakpoint command execution (callback commands)

### 8. Memory & Low-Level Inspection (Nice to Have - Priority 4)

| Command | CLI Syntax | Description | Use Case |
|---------|------------|-------------|----------|
| `examinemem` | `x <address>` | Examine raw memory | Low-level debugging |
| `disassemble` | `disass` | Show assembly code | Performance analysis |
| `regs` | `regs` | Show CPU registers | Architecture-specific debug |

**API Requirements:**
- Memory read at arbitrary addresses
- Disassembly retrieval
- Register state access

### 9. Deferred Function Inspection (Nice to Have - Priority 3)

| Command | CLI Syntax | Description | Use Case |
|---------|------------|-------------|----------|
| `deferred` | `deferred` | Show deferred functions | Go defer debugging |

**API Requirements:**
- Deferred function call list for current goroutine
- Multiple defer stacks if nested

### 10. Call Injection (Nice to Have - Priority 4)

| Command | CLI Syntax | Description | Use Case |
|---------|------------|-------------|----------|
| `call` | `call <expr>` | Call function from debugger | Runtime state modification |

**API Requirements:**
- Function call injection
- Return value capture
- Side effect handling

### 11. State Capture & Analysis (Nice to Have - Priority 4)

| Command | CLI Syntax | Description | Use Case |
|---------|------------|-------------|----------|
| `dump` | `dump <var>` | Deep dump of complex structures | Complex data inspection |
| `config` | `config -list`, `config <key> <val>` | View/modify debugger config | Customize behavior |

**API Requirements:**
- Deep variable traversal
- Configuration get/set operations
- Config persistence (if supported)

## API Protocol Selection

### Options

1. **JSON-RPC 2.0** (Delve native protocol)
   - Pros: Full feature support, mature, well-documented
   - Cons: More complex than DAP, custom protocol

2. **DAP (Debug Adapter Protocol)**
   - Pros: Standard protocol, VS Code compatibility
   - Cons: May not expose all Delve features, newer

3. **gRPC** (if available in future versions)
   - Pros: Performance, type safety
   - Cons: Not currently available in Delve

**Recommendation:** Use JSON-RPC 2.0 for full feature access.

## MCP Tool Interface Design

### Input Parameters (Extended)

```go
type DelveRequest struct {
    // Connection
    Host      string `json:"host" validate:"omitempty,hostname|ip"`
    Port      int    `json:"port" validate:"omitempty,min=1,max=65535"`

    // Session Management
    SessionID string `json:"session_id" validate:"required"`
    Action    string `json:"action" validate:"omitempty,oneof=connect disconnect command"`

    // Command Execution (for action=command)
    Command   string   `json:"command" validate:"required_if=Action command"`
    Args      []string `json:"args" validate:"omitempty"`

    // Context
    GoroutineID int64  `json:"goroutine_id" validate:"omitempty,min=1"`
    FrameIndex  int    `json:"frame_index" validate:"omitempty,min=0"`

    // Pagination
    MaxLines  int `json:"max_lines" validate:"omitempty,min=1,max=10000"`
    Offset    int `json:"offset" validate:"omitempty,min=0"`
}
```

### Command Categories

The tool will support commands grouped by functionality:

1. **Execution Control:** `continue`, `next`, `step`, `stepout`, `restart`
2. **Breakpoints:** `break`, `breakpoints`, `clear`, `clearall`, `condition`, `on`, `trace`, `watch`
3. **Variables:** `print`, `locals`, `args`, `vars`, `whatis`, `set`, `dump`
4. **Stack:** `stack`, `frame`, `list`
5. **Goroutines:** `goroutines`, `goroutine`
6. **Symbols:** `funcs`, `types`, `sources`
7. **Low-level:** `examinemem`, `disassemble`, `regs`
8. **Go-specific:** `deferred`, `call`
9. **Config:** `config`, `help`

### Session Management

Sessions will maintain:
- Active API connection
- Current goroutine context
- Current frame context
- Breakpoint state
- Configuration state

### Response Format

```go
type DelveResponse struct {
    Success   bool              `json:"success"`
    SessionID string            `json:"session_id"`
    State     string            `json:"state"` // running, stopped, exited
    Output    string            `json:"output"`
    Data      map[string]any    `json:"data,omitempty"` // Structured response
    Error     string            `json:"error,omitempty"`

    // Context Info
    CurrentGoroutine int64  `json:"current_goroutine,omitempty"`
    CurrentFrame     int    `json:"current_frame,omitempty"`
    Location         string `json:"location,omitempty"` // file:line
}
```

## Implementation Phases

### Phase 1: Core API Integration (Priority 1)
- [ ] Establish JSON-RPC client connection
- [ ] Implement session management with API connections
- [ ] Core execution control (continue, next, step, stepout, restart)
- [ ] Basic breakpoint management (break, breakpoints, clear, clearall)
- [ ] Variable inspection (print, locals, args, vars, whatis)
- [ ] Stack operations (stack, frame, list)

### Phase 2: Enhanced Debugging (Priority 2)
- [ ] Goroutine management (goroutines, goroutine switching)
- [ ] Symbol navigation (funcs, types, sources)
- [ ] Conditional breakpoints (condition)
- [ ] Variable modification (set)
- [ ] Help system

### Phase 3: Advanced Features (Priority 3)
- [ ] Tracepoints (trace)
- [ ] Watchpoints (watch)
- [ ] Breakpoint commands (on)
- [ ] Deferred function inspection (deferred)
- [ ] Deep variable dumps (dump)

### Phase 4: Power User Features (Priority 4)
- [ ] Call injection (call)
- [ ] Memory examination (examinemem)
- [ ] Disassembly (disassemble)
- [ ] Register access (regs)
- [ ] Configuration management (config)

## Success Criteria

1. **Functional Parity:** All Priority 1 & 2 commands work via API
2. **Performance:** API calls complete in <100ms for typical operations
3. **Reliability:** Session management handles disconnections gracefully
4. **Usability:** Structured responses are more useful than CLI text parsing
5. **Compatibility:** Works with Delve 1.20.0+ servers

## Testing Strategy

1. **Unit Tests:** API client, request/response parsing
2. **Integration Tests:** Full debugging workflows
3. **Compatibility Tests:** Multiple Delve versions
4. **Error Handling Tests:** Connection failures, invalid commands
5. **Performance Tests:** High-frequency command execution

## Documentation Updates Required

- [ ] Update PROJECT_NOTES.md with API architecture
- [ ] Update README.md with new capabilities
- [ ] Create API usage examples
- [ ] Document migration guide from CLI to API mode
- [ ] Add troubleshooting guide

## Open Questions

1. Should we maintain CLI mode as a fallback?
2. How to handle API version differences across Delve versions?
3. Should we implement DAP in addition to JSON-RPC for broader compatibility?
4. How to handle concurrent command execution in a single session?
5. Should we support multiple simultaneous debugging sessions per MCP server instance?

## References

- Delve API Client Guide: https://github.com/go-delve/delve/blob/master/Documentation/api/ClientHowto.md
- Delve JSON-RPC API: https://github.com/go-delve/delve/tree/master/Documentation/api
- DAP Specification: https://microsoft.github.io/debug-adapter-protocol/
- Delve Command Documentation: https://github.com/go-delve/delve/tree/master/Documentation/cli

## Revision History

| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 1.0.0 | 2025-01-12 | Initial | Initial specification |
