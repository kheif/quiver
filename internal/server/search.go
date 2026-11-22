package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/kheif/quiver/internal/library"
	"github.com/kheif/quiver/internal/registry"
	"github.com/kheif/quiver/internal/skill"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// maxFound is the most skills search_skills returns: the assistant needs few,
// and every match carries its description.
const maxFound = 10

// FoundSkill is one skill in the search_skills result.
type FoundSkill struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Source      string `json:"source"`
	// Assigned is true: search_skills returns only Searchable skills of the
	// folder. Scope names the scope that holds the skill as Searchable, the
	// project before the global one.
	Assigned bool   `json:"assigned"`
	Scope    string `json:"scope,omitempty"`
}

// SearchResult is the search_skills result.
type SearchResult struct {
	Skills []FoundSkill `json:"skills"`
	// More is the number of matches left out after the first maxFound.
	More int `json:"more,omitempty"`
}

// onceLog logs each message once per server process.
type onceLog struct {
	log  *slog.Logger
	seen sync.Map
}

func (o *onceLog) warn(msg string) {
	if msg == "" || o.log == nil {
		return
	}
	if _, dup := o.seen.LoadOrStore(msg, true); !dup {
		o.log.Warn(msg)
	}
}

// addSearchTool registers search_skills.
func addSearchTool(s *mcp.Server, sess *session, log *slog.Logger) {
	type searchIn struct {
		Query string `json:"query" jsonschema:"text to find in skill names and descriptions, such as pdf or pull request"`
	}
	warned := &onceLog{log: log}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "search_skills",
		Annotations: readOnly,
		Description: `Search the skills that are searchable in this project: assigned here, but not listed. ` +
			`Each match is assigned here, so load the one that fits the task with get_skill and follow it. ` +
			`Skills that are listed already appear in the skill list, and skills that are not assigned are not searched.`,
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in searchIn) (*mcp.CallToolResult, SearchResult, error) {
		return searchSkills(ctx, warned, sess.projectID(), in.Query)
	})
}

// searchableSet returns the skills that are Searchable in the folder, with
// the scope that holds each: the project before the global one.
func searchableSet(l *library.Library, r *registry.Registry, projectID string) ([]*skill.Skill, map[string]string) {
	scope := map[string]string{}
	for _, id := range r.Searchable {
		scope[id] = "global"
	}
	if p, ok := r.Projects[projectID]; ok {
		for _, id := range p.Searchable {
			scope[id] = "project"
		}
	}
	var set []*skill.Skill
	for _, id := range r.SearchableServed(projectID) {
		if s, err := l.Get(id); err == nil {
			set = append(set, s)
		}
	}
	return set, scope
}

func searchSkills(ctx context.Context, warned *onceLog, projectID, query string) (*mcp.CallToolResult, SearchResult, error) {
	if strings.TrimSpace(query) == "" {
		return nil, SearchResult{}, errors.New("give the text to search for in query")
	}
	r, err := registry.Load()
	if err != nil {
		return nil, SearchResult{}, err
	}
	l, err := library.Load()
	if err != nil {
		return nil, SearchResult{}, err
	}
	set, scope := searchableSet(l, r, projectID)
	out := SearchResult{Skills: []FoundSkill{}}
	// The engine ranks and caps within the Searchable set, but takes its
	// statistics from the whole library.
	search, err := l.NewSearchWith("", library.Options{Refresh: library.RefreshBackground})
	if err != nil {
		return nil, SearchResult{}, err
	}
	warned.warn(search.Problem)
	matches, note := search.Find(ctx, query, set)
	warned.warn(note)
	if len(matches) > maxFound {
		out.More = len(matches) - maxFound
		matches = matches[:maxFound]
	}
	var b strings.Builder
	if len(matches) == 0 {
		fmt.Fprintf(&b, "No searchable skill matches %q.", query)
	}
	for _, m := range matches {
		f := FoundSkill{ID: m.ID, Name: m.Name, Description: m.Description, Source: m.Source, Assigned: true, Scope: scope[m.ID]}
		out.Skills = append(out.Skills, f)
		fmt.Fprintf(&b, "- %s (%s): %s\n", f.ID, f.Scope, f.Description)
	}
	if out.More > 0 {
		fmt.Fprintf(&b, "%d more matches; search for something more specific.\n", out.More)
	}
	return text(strings.TrimRight(b.String(), "\n")), out, nil
}
