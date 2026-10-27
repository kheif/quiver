// Package server is Quiver's MCP server. Assistants start it with
// "quiver mcp" and talk to it over stdin and stdout.
package server

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// New returns the MCP server with every tool registered.
func New(version string) *mcp.Server {
	s := mcp.NewServer(&mcp.Implementation{Name: "quiver", Version: version}, nil)
	addPing(s)
	return s
}

// Run serves MCP over stdin and stdout until the client disconnects.
// Nothing else may write to stdout, or the client sees broken messages.
func Run(ctx context.Context, version string) error {
	return New(version).Run(ctx, &mcp.StdioTransport{})
}

// addPing registers a placeholder tool that proves the wiring. Remove it
// once the real tools exist.
func addPing(s *mcp.Server) {
	mcp.AddTool(s, &mcp.Tool{Name: "ping", Description: "Check that the Quiver server responds."},
		func(context.Context, *mcp.CallToolRequest, struct{}) (*mcp.CallToolResult, any, error) {
			return text("pong"), nil, nil
		})
}

// text returns a tool result with one text block.
func text(s string) *mcp.CallToolResult {
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: s}}}
}
