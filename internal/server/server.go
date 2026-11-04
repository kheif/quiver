// Package server is Quiver's MCP server. Assistants start it with
// "quiver mcp" and talk to it over stdin and stdout.
package server

import (
	"context"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/kheif/quiver/internal/project"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// rootsTimeout bounds the roots request Quiver makes to a client after
// initialization.
const rootsTimeout = 10 * time.Second

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
	sess := &session{where: opts.Where}
	var p *prompts
	var s *mcp.Server
	s = mcp.NewServer(&mcp.Implementation{Name: "quiver", Version: opts.Version}, &mcp.ServerOptions{
		Logger:       opts.Log,
		Instructions: instructions(opts),
		// resolveRoots sends a server-to-client request, so it runs off the
		// session's receiving goroutine to avoid blocking it.
		InitializedHandler: func(_ context.Context, req *mcp.InitializedRequest) {
			go sess.resolveRoots(req, s, p, opts.Log)
		},
	})
	s.AddReceivingMiddleware(recoverPanics(opts.Log))
	addSkillTools(s, sess)
	addCreateTool(s, sess)
	addUpdateTool(s, sess)
	addSearchTool(s, sess)
	p = &prompts{}
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

// resolveRoots serves the project of an MCP root when the server started
// outside a project, for a client that sends the workspace as a root (VS
// Code). $QUIVER_PROJECT wins, and a working directory already in a project is
// kept, so roots are read only when neither names a project. It never fails
// the session: on any trouble the global skills are still served.
func (s *session) resolveRoots(req *mcp.InitializedRequest, srv *mcp.Server, p *prompts, log *slog.Logger) {
	if os.Getenv(ProjectEnv) != "" {
		return
	}
	s.mu.Lock()
	has := s.where.Project != nil
	s.mu.Unlock()
	if has {
		return
	}
	caps := req.Session.InitializeParams().Capabilities
	// Roots are deprecated upstream (SEP-2577) but stay functional through a
	// year-plus window, and they are how VS Code hands the server its
	// workspace (docs/spike-hooks.md), so Quiver still reads them.
	if caps == nil || (caps.RootsV2 == nil && !caps.Roots.ListChanged) { //nolint:staticcheck // SA1019: roots is the only workspace signal VS Code sends
		return // the client has no roots to ask for
	}
	// The notification's context ends when the handler returns, so use a
	// fresh, bounded one for the server-to-client request.
	ctx, cancel := context.WithTimeout(context.Background(), rootsTimeout)
	defer cancel()
	res, err := req.Session.ListRoots(ctx, nil) //nolint:staticcheck // SA1019: see above
	if err != nil {
		log.Warn("could not read the client's roots", "err", err)
		return
	}
	uris := make([]string, 0, len(res.Roots))
	for _, r := range res.Roots {
		uris = append(uris, r.URI)
	}
	s.applyRoots(srv, p, uris, log)
}

// applyRoots serves the project of the first root URI that is in one: it sets
// the session's folder, records the project's path and resyncs the prompts.
// It returns whether a project was served.
func (s *session) applyRoots(srv *mcp.Server, p *prompts, uris []string, log *slog.Logger) bool {
	w, ok := ResolveRoots(uris)
	if !ok {
		return false
	}
	s.mu.Lock()
	s.where = w
	s.mu.Unlock()
	if _, err := recordPath(w.Project); err != nil {
		log.Warn("could not record the project folder", "err", err)
	}
	if p != nil && srv != nil {
		if _, err := p.sync(srv, w.ProjectID()); err != nil {
			log.Warn("no skill prompts for the root project", "err", err)
		}
	}
	log.Info("served the project from an MCP root", "dir", w.Dir, "project", w.ProjectID())
	return true
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

// instructions returns the text the assistant adds to its system prompt: how
// to use the server, the pointer to the always-on skills, then the pointer to
// create_skill. The skills themselves come from the session-start hook. It is
// built once, so skills assigned later reach it in the next session.
func instructions(opts Options) string {
	_, skills, err := loadServed(opts.Where.ProjectID())
	if err != nil {
		opts.Log.Warn("no always-on pointer in the instructions", "err", err)
		return usage + createHint
	}
	return usage + pointer(skills) + createHint
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
