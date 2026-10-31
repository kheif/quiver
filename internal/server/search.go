package server

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// maxFound is the most skills search_skills returns.
const maxFound = 50

// FoundSkill is one skill in the search_skills result.
type FoundSkill struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Source      string `json:"source"`
	// Assigned reports whether the skill is served in this project, and
	// Scope says how: "global" or "project".
	Assigned bool   `json:"assigned"`
	Scope    string `json:"scope,omitempty"`
}

// SearchResult is the search_skills result.
type SearchResult struct {
	Skills []FoundSkill `json:"skills"`
	// More is the number of matches left out after the first maxFound.
	More int `json:"more,omitempty"`
}

// addSearchTool registers search_skills.
func addSearchTool(s *mcp.Server, sess *session) {
	type searchIn struct {
		Query string `json:"query" jsonschema:"text to find in skill IDs, names and descriptions, such as pdf or pull request"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name: "search_skills",
		Description: `Search the user's whole Quiver library, including skills not assigned in this project. ` +
			`Each match says whether it is assigned here. ` +
			`You may suggest an unassigned skill to the user when it fits the task, but do not assume it applies: ` +
			`ask before you load it with get_skill. The user assigns it with "quiver add <id>".`,
	}, func(_ context.Context, _ *mcp.CallToolRequest, in searchIn) (*mcp.CallToolResult, SearchResult, error) {
		return searchSkills(sess.projectID(), in.Query)
	})
}

func searchSkills(projectID, query string) (*mcp.CallToolResult, SearchResult, error) {
	if strings.TrimSpace(query) == "" {
		return nil, SearchResult{}, errors.New("give the text to search for in query")
	}
	l, served, err := loadServed(projectID)
	if err != nil {
		return nil, SearchResult{}, err
	}
	scope := map[string]string{}
	for _, s := range served {
		scope[s.ID] = s.Scope
	}
	out := SearchResult{Skills: []FoundSkill{}}
	matches := l.Search(query)
	if len(matches) > maxFound {
		out.More = len(matches) - maxFound
		matches = matches[:maxFound]
	}
	var b strings.Builder
	if len(matches) == 0 {
		fmt.Fprintf(&b, "No skill matches %q.", query)
	}
	for _, m := range matches {
		f := FoundSkill{ID: m.ID, Name: m.Name, Description: m.Description, Source: m.Source, Scope: scope[m.ID]}
		f.Assigned = f.Scope != ""
		out.Skills = append(out.Skills, f)
		here := "not assigned here"
		if f.Assigned {
			here = "assigned here (" + f.Scope + ")"
		}
		fmt.Fprintf(&b, "- %s, %s: %s\n", f.ID, here, f.Description)
	}
	if out.More > 0 {
		fmt.Fprintf(&b, "%d more matches; search for something more specific.\n", out.More)
	}
	return text(strings.TrimRight(b.String(), "\n")), out, nil
}
