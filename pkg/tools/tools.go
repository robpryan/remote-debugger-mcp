package tools

import (
	"github.com/robpryan/remote-debugger-mcp/pkg/server"
)

type Tool interface {
	Register(srv *server.Server)
}
