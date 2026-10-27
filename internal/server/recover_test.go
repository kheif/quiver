package server

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestPanicBecomesToolError(t *testing.T) {
	var logs bytes.Buffer
	s := New("dev", slog.New(slog.NewTextHandler(&logs, nil)))
	mcp.AddTool(s, &mcp.Tool{Name: "boom"}, func(context.Context, *mcp.CallToolRequest, struct{}) (*mcp.CallToolResult, any, error) {
		panic("kaboom")
	})
	cs := connect(t, s)
	out, isError := call(t, cs, "boom", map[string]any{})
	if !isError || !strings.Contains(out, "internal error in quiver: kaboom") {
		t.Errorf("got %q, isError %v", out, isError)
	}
	if !strings.Contains(logs.String(), "kaboom") || !strings.Contains(logs.String(), "stack=") {
		t.Errorf("logs:\n%s", logs.String())
	}
	// The session survives.
	if out, isError := call(t, cs, "ping", map[string]any{}); out != "pong" || isError {
		t.Errorf("ping after panic = %q, isError %v", out, isError)
	}
}
