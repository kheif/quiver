// Package server is Quiver's MCP server. Assistants start it with
// "quiver mcp" and talk to it over stdin and stdout.
package server

import (
	"context"
	"log/slog"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Options configure a server.
type Options struct {
	Version string
	// Log gets the server's own logs and the SDK's.
	Log *slog.Logger
	// Where is the project to serve, from Resolve.
	Where Where
}

// New returns the MCP server with every tool registered.
func New(opts Options) *mcp.Server {
	s := mcp.NewServer(&mcp.Implementation{Name: "quiver", Version: opts.Version}, &mcp.ServerOptions{
		Logger:       opts.Log,
		Instructions: instructions(opts),
	})
	s.AddReceivingMiddleware(recoverPanics(opts.Log))
	addSkillTools(s, opts.Where)
	return s
}

// instructions returns the text the assistant adds to its system prompt:
// the catalog of skills served in the project. It is built once, so skills
// assigned later reach it in the next session.
func instructions(opts Options) string {
	_, skills, err := loadServed(opts.Where.ProjectID())
	if err != nil {
		opts.Log.Warn("no skill catalog in the instructions", "err", err)
		return ""
	}
	return catalog(skills, maxInstructions)
}

// Run serves MCP over stdin and stdout until the client disconnects.
// Nothing else may write to stdout, or the client sees broken messages, so
// logs go to stderr and LogFile.
func Run(ctx context.Context, version string) error {
	log, closeLog := openLog()
	defer closeLog()
	w, err := Resolve()
	if err != nil {
		log.Warn("no project found, serving global skills only", "err", err)
	}
	if w.Project != nil {
		copied, err := recordPath(w.Project)
		if err != nil {
			log.Warn("could not record the project folder", "err", err)
		}
		if copied != nil {
			log.Warn(copied.Error())
		}
	}
	log.Info("server started", "version", version, "dir", w.Dir, "from", w.From, "project", w.ProjectID())
	err = New(Options{Version: version, Log: log, Where: w}).Run(ctx, &mcp.StdioTransport{})
	if err != nil {
		log.Error("server stopped", "err", err)
	} else {
		log.Info("server stopped")
	}
	return err
}

// text returns a tool result with one text block.
func text(s string) *mcp.CallToolResult {
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: s}}}
}
