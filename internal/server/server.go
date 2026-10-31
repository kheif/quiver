// Package server is Quiver's MCP server. Assistants start it with
// "quiver mcp" and talk to it over stdin and stdout.
package server

import (
	"context"
	"log/slog"
	"sync"

	"github.com/kheif/quiver/internal/project"
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

// New returns the MCP server with every tool and prompt registered.
func New(opts Options) *mcp.Server {
	s, _, _ := newServer(opts)
	return s
}

// newServer is New, plus what watch needs to keep the server up to date.
func newServer(opts Options) (*mcp.Server, *session, *prompts) {
	s := mcp.NewServer(&mcp.Implementation{Name: "quiver", Version: opts.Version}, &mcp.ServerOptions{
		Logger:       opts.Log,
		Instructions: instructions(opts),
	})
	s.AddReceivingMiddleware(recoverPanics(opts.Log))
	sess := &session{where: opts.Where}
	addSkillTools(s, sess)
	addCreateTool(s, sess)
	addUpdateTool(s)
	p := &prompts{}
	if _, err := p.sync(s, sess.projectID()); err != nil {
		opts.Log.Warn("no skill prompts", "err", err)
	}
	return s, sess, p
}

// session holds the project the tools serve. create_skill can register the
// folder during the session, and requests may run at the same time, so it
// is read and changed under a lock.
type session struct {
	mu    sync.Mutex
	where Where
}

// projectID returns the served project's ID. While there is none, it looks
// for one again: "quiver add" may have registered the folder since the
// server started.
func (s *session) projectID() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.where.Project == nil && s.where.Dir != "" {
		if p, err := project.Find(s.where.Dir); err == nil {
			s.where.Project = p
		}
	}
	return s.where.ProjectID()
}

// instructions returns the text the assistant adds to its system prompt:
// the pointer to the always-on skills, the catalog of skills served in the
// project, then the pointer to create_skill. It is built once, so skills
// assigned later reach it in the next session.
func instructions(opts Options) string {
	_, skills, err := loadServed(opts.Where.ProjectID())
	if err != nil {
		opts.Log.Warn("no skill catalog in the instructions", "err", err)
		return createHint
	}
	return buildInstructions(skills)
}

// buildInstructions joins the parts of the instructions within
// maxInstructions.
func buildInstructions(skills []servedSkill) string {
	p := pointer(skills)
	c := catalog(skills, maxInstructions-len(p)-len(createHint)-1)
	if c != "" {
		c += "\n"
	}
	return p + c + createHint
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
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	s, sess, p := newServer(Options{Version: version, Log: log, Where: w})
	if err := watch(ctx, s, sess, p, log); err != nil {
		log.Warn("no live updates; new skills appear in the next session", "err", err)
	}
	err = s.Run(ctx, &mcp.StdioTransport{})
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
