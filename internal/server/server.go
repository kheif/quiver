// Package server is Quiver's MCP server. Assistants start it with
// "quiver mcp" and talk to it over stdin and stdout.
package server

import (
	"context"
	"log/slog"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// New returns the MCP server with every tool registered. log gets the
// server's own logs and the SDK's.
func New(version string, log *slog.Logger) *mcp.Server {
	s := mcp.NewServer(&mcp.Implementation{Name: "quiver", Version: version}, &mcp.ServerOptions{Logger: log})
	addPing(s)
	return s
}

// Run serves MCP over stdin and stdout until the client disconnects.
// Nothing else may write to stdout, or the client sees broken messages, so
// logs go to stderr and LogFile.
func Run(ctx context.Context, version string) error {
	log, closeLog := openLog()
	defer closeLog()
	wd, _ := os.Getwd()
	log.Info("server started", "version", version, "dir", wd)
	err := New(version, log).Run(ctx, &mcp.StdioTransport{})
	if err != nil {
		log.Error("server stopped", "err", err)
	} else {
		log.Info("server stopped")
	}
	return err
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
