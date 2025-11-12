package client

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/go-delve/delve/service/api"
)

// LoadConfig returns a default LoadConfig for variable inspection.
func DefaultLoadConfig() api.LoadConfig {
	return api.LoadConfig{
		FollowPointers:     true,
		MaxVariableRecurse: 1,
		MaxStringLen:       64,
		MaxArrayValues:     64,
		MaxStructFields:    -1,
	}
}

// ExecuteCommand executes a Delve CLI-style command via the API.
// This provides a bridge between CLI-style commands and API calls.
func (c *Client) ExecuteCommand(command string, args []string) (string, error) {
	switch command {
	case "continue", "c":
		return c.executeContinue()
	case "next", "n":
		return c.executeNext()
	case "step", "s":
		return c.executeStep()
	case "stepout", "so":
		return c.executeStepOut()
	case "restart", "r":
		return c.executeRestart()
	case "halt":
		return c.executeHalt()
	case "break", "b":
		return c.executeBreak(args)
	case "breakpoints", "bp":
		return c.executeBreakpoints()
	case "clear":
		return c.executeClear(args)
	case "clearall":
		return c.executeClearAll()
	case "print", "p":
		return c.executePrint(args)
	case "locals":
		return c.executeLocals()
	case "args":
		return c.executeArgs()
	case "vars":
		return c.executeVars(args)
	case "whatis":
		return c.executeWhatis(args)
	case "set":
		return c.executeSet(args)
	case "stack", "bt":
		return c.executeStack(args)
	case "frame":
		return c.executeFrame(args)
	case "goroutines", "grs":
		return c.executeGoroutines(args)
	case "goroutine", "gr":
		return c.executeGoroutine(args)
	case "funcs":
		return c.executeFuncs(args)
	case "types":
		return c.executeTypes(args)
	case "sources":
		return c.executeSources(args)
	case "list", "ls", "l":
		return c.executeList(args)
	default:
		return "", fmt.Errorf("unknown command: %s", command)
	}
}

func (c *Client) executeContinue() (string, error) {
	state, err := c.Continue()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("Continuing... (program is now running)\nState: %s", formatState(state)), nil
}

func (c *Client) executeNext() (string, error) {
	state, err := c.Next()
	if err != nil {
		return "", err
	}
	return formatState(state), nil
}

func (c *Client) executeStep() (string, error) {
	state, err := c.Step()
	if err != nil {
		return "", err
	}
	return formatState(state), nil
}

func (c *Client) executeStepOut() (string, error) {
	state, err := c.StepOut()
	if err != nil {
		return "", err
	}
	return formatState(state), nil
}

func (c *Client) executeRestart() (string, error) {
	state, err := c.Restart()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("Restarted\n%s", formatState(state)), nil
}

func (c *Client) executeHalt() (string, error) {
	state, err := c.Halt()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("Halted\n%s", formatState(state)), nil
}

func (c *Client) executeBreak(args []string) (string, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("break requires a location argument")
	}

	location := strings.Join(args, " ")

	// Create breakpoint - the API accepts various location formats
	bp := &api.Breakpoint{
		// Try to parse as file:line first
		File: "",
		Line: 0,
		// Or as function name
		FunctionName: location,
	}

	// Check if it's file:line format
	if strings.Contains(location, ":") {
		parts := strings.Split(location, ":")
		if len(parts) == 2 {
			if line, err := strconv.Atoi(parts[1]); err == nil {
				bp.File = parts[0]
				bp.Line = line
				bp.FunctionName = ""
			}
		}
	}

	createdBp, err := c.CreateBreakpoint(bp)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("Breakpoint %d set at %s:%d", createdBp.ID, createdBp.File, createdBp.Line), nil
}

func (c *Client) executeBreakpoints() (string, error) {
	bps, err := c.ListBreakpoints()
	if err != nil {
		return "", err
	}

	if len(bps) == 0 {
		return "No breakpoints set", nil
	}

	var result strings.Builder
	for _, bp := range bps {
		result.WriteString(fmt.Sprintf("[%d] %s:%d", bp.ID, bp.File, bp.Line))
		if bp.FunctionName != "" {
			result.WriteString(fmt.Sprintf(" in %s", bp.FunctionName))
		}
		if bp.Cond != "" {
			result.WriteString(fmt.Sprintf(" (condition: %s)", bp.Cond))
		}
		if bp.TotalHitCount > 0 {
			result.WriteString(fmt.Sprintf(" [hit count: %d]", bp.TotalHitCount))
		}
		result.WriteString("\n")
	}

	return result.String(), nil
}

func (c *Client) executeClear(args []string) (string, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("clear requires a breakpoint ID")
	}

	id, err := strconv.Atoi(args[0])
	if err != nil {
		return "", fmt.Errorf("invalid breakpoint ID: %s", args[0])
	}

	bp, err := c.ClearBreakpoint(id)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("Breakpoint %d cleared at %s:%d", bp.ID, bp.File, bp.Line), nil
}

func (c *Client) executeClearAll() (string, error) {
	bps, err := c.ListBreakpoints()
	if err != nil {
		return "", err
	}

	var result strings.Builder
	for _, bp := range bps {
		_, err := c.ClearBreakpoint(bp.ID)
		if err != nil {
			result.WriteString(fmt.Sprintf("Failed to clear breakpoint %d: %v\n", bp.ID, err))
		} else {
			result.WriteString(fmt.Sprintf("Breakpoint %d cleared\n", bp.ID))
		}
	}

	if result.Len() == 0 {
		return "No breakpoints to clear", nil
	}

	return result.String(), nil
}

func (c *Client) executePrint(args []string) (string, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("print requires an expression")
	}

	expr := strings.Join(args, " ")
	scope := c.GetScope()
	cfg := DefaultLoadConfig()

	variable, err := c.Eval(scope, expr, &cfg)
	if err != nil {
		return "", err
	}

	return formatVariable(variable), nil
}

func (c *Client) executeLocals() (string, error) {
	scope := c.GetScope()
	cfg := DefaultLoadConfig()

	vars, err := c.ListLocalVariables(scope, cfg)
	if err != nil {
		return "", err
	}

	if len(vars) == 0 {
		return "No local variables", nil
	}

	var result strings.Builder
	for _, v := range vars {
		result.WriteString(formatVariable(&v))
		result.WriteString("\n")
	}

	return result.String(), nil
}

func (c *Client) executeArgs() (string, error) {
	scope := c.GetScope()
	cfg := DefaultLoadConfig()

	vars, err := c.ListFunctionArgs(scope, cfg)
	if err != nil {
		return "", err
	}

	if len(vars) == 0 {
		return "No function arguments", nil
	}

	var result strings.Builder
	for _, v := range vars {
		result.WriteString(formatVariable(&v))
		result.WriteString("\n")
	}

	return result.String(), nil
}

func (c *Client) executeVars(args []string) (string, error) {
	filter := ".*"
	if len(args) > 0 {
		filter = args[0]
	}

	// Note: The API doesn't have a direct ListPackageVars equivalent in rpc2.RPCClient
	// We'll need to use ListLocalVars as a fallback
	scope := c.GetScope()
	cfg := DefaultLoadConfig()

	vars, err := c.ListLocalVariables(scope, cfg)
	if err != nil {
		return "", err
	}

	if len(vars) == 0 {
		return fmt.Sprintf("No variables matching '%s'", filter), nil
	}

	var result strings.Builder
	for _, v := range vars {
		result.WriteString(formatVariable(&v))
		result.WriteString("\n")
	}

	return result.String(), nil
}

func (c *Client) executeWhatis(args []string) (string, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("whatis requires an expression")
	}

	expr := strings.Join(args, " ")
	scope := c.GetScope()
	cfg := DefaultLoadConfig()

	variable, err := c.Eval(scope, expr, &cfg)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s: %s", variable.Name, variable.Type), nil
}

func (c *Client) executeSet(args []string) (string, error) {
	if len(args) < 3 || args[1] != "=" {
		return "", fmt.Errorf("set requires format: set <var> = <value>")
	}

	symbol := args[0]
	value := strings.Join(args[2:], " ")
	scope := c.GetScope()

	err := c.SetVariable(scope, symbol, value)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("Set %s = %s", symbol, value), nil
}

func (c *Client) executeStack(args []string) (string, error) {
	depth := 50
	if len(args) > 0 {
		if d, err := strconv.Atoi(args[0]); err == nil {
			depth = d
		}
	}

	goroutineID := c.currentGoroutine
	frames, err := c.Stacktrace(goroutineID, depth, api.StacktraceReadDefers)
	if err != nil {
		return "", err
	}

	if len(frames) == 0 {
		return "No stack frames", nil
	}

	var result strings.Builder
	for i, frame := range frames {
		result.WriteString(fmt.Sprintf("#%d ", i))
		if frame.Function != nil {
			result.WriteString(frame.Function.Name())
		} else {
			result.WriteString("???")
		}
		result.WriteString(fmt.Sprintf(" at %s:%d\n", frame.File, frame.Line))
	}

	return result.String(), nil
}

func (c *Client) executeFrame(args []string) (string, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("frame requires an index")
	}

	idx, err := strconv.Atoi(args[0])
	if err != nil {
		return "", fmt.Errorf("invalid frame index: %s", args[0])
	}

	c.SetCurrentFrame(idx)
	return fmt.Sprintf("Switched to frame %d", idx), nil
}

func (c *Client) executeGoroutines(args []string) (string, error) {
	count := 100
	if len(args) > 0 {
		if c, err := strconv.Atoi(args[0]); err == nil {
			count = c
		}
	}

	grs, _, err := c.ListGoroutines(0, count)
	if err != nil {
		return "", err
	}

	if len(grs) == 0 {
		return "No goroutines", nil
	}

	var result strings.Builder
	for _, gr := range grs {
		result.WriteString(fmt.Sprintf("[%d] ", gr.ID))
		if gr.UserCurrentLoc.Function != nil {
			result.WriteString(gr.UserCurrentLoc.Function.Name())
		}
		result.WriteString(fmt.Sprintf(" at %s:%d\n", gr.UserCurrentLoc.File, gr.UserCurrentLoc.Line))
	}

	return result.String(), nil
}

func (c *Client) executeGoroutine(args []string) (string, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("goroutine requires an ID")
	}

	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return "", fmt.Errorf("invalid goroutine ID: %s", args[0])
	}

	c.SetCurrentGoroutine(id)
	return fmt.Sprintf("Switched to goroutine %d", id), nil
}

func (c *Client) executeFuncs(args []string) (string, error) {
	filter := ".*"
	if len(args) > 0 {
		filter = args[0]
	}

	funcs, err := c.ListFunctions(filter, 0)
	if err != nil {
		return "", err
	}

	if len(funcs) == 0 {
		return fmt.Sprintf("No functions matching '%s'", filter), nil
	}

	var result strings.Builder
	for _, fn := range funcs {
		result.WriteString(fn)
		result.WriteString("\n")
	}

	return result.String(), nil
}

func (c *Client) executeTypes(args []string) (string, error) {
	filter := ".*"
	if len(args) > 0 {
		filter = args[0]
	}

	types, err := c.ListTypes(filter)
	if err != nil {
		return "", err
	}

	if len(types) == 0 {
		return fmt.Sprintf("No types matching '%s'", filter), nil
	}

	var result strings.Builder
	for _, t := range types {
		result.WriteString(t)
		result.WriteString("\n")
	}

	return result.String(), nil
}

func (c *Client) executeSources(args []string) (string, error) {
	filter := ".*"
	if len(args) > 0 {
		filter = args[0]
	}

	sources, err := c.ListSources(filter)
	if err != nil {
		return "", err
	}

	if len(sources) == 0 {
		return fmt.Sprintf("No sources matching '%s'", filter), nil
	}

	var result strings.Builder
	for _, s := range sources {
		result.WriteString(s)
		result.WriteString("\n")
	}

	return result.String(), nil
}

func (c *Client) executeList(args []string) (string, error) {
	// List source code around current location
	// This is more complex and requires the ListSources API
	// For now, return a placeholder
	return "list command not yet implemented", nil
}

// formatState formats a DebuggerState for display.
func formatState(state *api.DebuggerState) string {
	if state == nil {
		return "No state information"
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("Running: %v\n", state.Running))

	if state.CurrentThread != nil {
		result.WriteString(fmt.Sprintf("Thread: %d\n", state.CurrentThread.ID))
		if state.CurrentThread.File != "" {
			result.WriteString(fmt.Sprintf("Location: %s:%d\n", state.CurrentThread.File, state.CurrentThread.Line))
		}
		if state.CurrentThread.Function != nil {
			result.WriteString(fmt.Sprintf("Function: %s\n", state.CurrentThread.Function.Name()))
		}
		if state.CurrentThread.Breakpoint != nil {
			result.WriteString(fmt.Sprintf("Breakpoint: %d\n", state.CurrentThread.Breakpoint.ID))
		}
	}

	if state.Exited {
		result.WriteString(fmt.Sprintf("Exited: true (exit status: %d)\n", state.ExitStatus))
	}

	return result.String()
}

// formatVariable formats a Variable for display.
func formatVariable(v *api.Variable) string {
	if v == nil {
		return ""
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("%s (%s) = %s", v.Name, v.Type, v.Value))

	if len(v.Children) > 0 {
		result.WriteString(fmt.Sprintf(" [%d children]", len(v.Children)))
	}

	return result.String()
}
