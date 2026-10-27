package server

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// discard is the logger for tests that do not check logs.
var discard = slog.New(slog.NewTextHandler(io.Discard, nil))

// connect runs s on an in-memory transport and returns a client connected
// to it, with no subprocess. Every server test uses it.
func connect(t *testing.T, s *mcp.Server) *mcp.ClientSession {
	t.Helper()
	ctx := context.Background()
	serverEnd, clientEnd := mcp.NewInMemoryTransports()
	ss, err := s.Connect(ctx, serverEnd, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ss.Close() })
	cs, err := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "0"}, nil).Connect(ctx, clientEnd, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cs.Close() })
	return cs
}

// call calls a tool and returns its text, failing the test on a protocol
// error. A tool error is returned in isError.
func call(t *testing.T, cs *mcp.ClientSession, name string, args any) (out string, isError bool) {
	t.Helper()
	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range res.Content {
		if tc, ok := c.(*mcp.TextContent); ok {
			out += tc.Text
		}
	}
	return out, res.IsError
}
