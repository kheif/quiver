package server

import (
	"context"
	"fmt"
	"log/slog"
	"runtime/debug"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// recoverPanics turns a panic in a handler into an error, so one bad call
// does not end the session. In a tool call it becomes a tool error the
// assistant can read; other methods return a protocol error.
func recoverPanics(log *slog.Logger) mcp.Middleware {
	return func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, req mcp.Request) (res mcp.Result, err error) {
			defer func() {
				p := recover()
				if p == nil {
					return
				}
				log.Error("panic", "method", method, "panic", p, "stack", string(debug.Stack()))
				msg := fmt.Sprintf("internal error in quiver: %v", p)
				if method == "tools/call" {
					r := text(msg)
					r.IsError = true
					res, err = r, nil
					return
				}
				res, err = nil, fmt.Errorf("%s", msg)
			}()
			return next(ctx, method, req)
		}
	}
}
