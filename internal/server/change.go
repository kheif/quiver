package server

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kheif/quiver/internal/library"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type updateIn struct {
	ID          string            `json:"id" jsonschema:"the ID of the local skill to change, as list_skills shows it"`
	Description *string           `json:"description,omitempty" jsonschema:"the new description; leave it out to keep the current one"`
	Body        *string           `json:"body,omitempty" jsonschema:"the new instructions, as markdown without frontmatter, replacing the whole body; leave it out to keep the current one"`
	Files       map[string]string `json:"files,omitempty" jsonschema:"extra files to change: a relative path inside the skill folder mapped to its new content; an empty content deletes the file"`
}

// UpdateResult is the update_skill result.
type UpdateResult struct {
	ID       string   `json:"id"`
	Path     string   `json:"path"`
	Warnings []string `json:"warnings,omitempty"`
}

// addUpdateTool registers update_skill.
func addUpdateTool(s *mcp.Server) {
	notDestructive := false
	mcp.AddTool(s, &mcp.Tool{
		Name: "update_skill",
		Description: `Change a local skill in the user's Quiver library: its description, its body, or its extra files. ` +
			`Only the given parts change. Skills from remote sources are read-only. ` +
			`Before you call this tool, show the user the change you want to make and ask them to confirm it. ` +
			`The change is committed in the library, and get_skill returns the new version right away.`,
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: false, DestructiveHint: &notDestructive},
	}, func(_ context.Context, _ *mcp.CallToolRequest, in updateIn) (*mcp.CallToolResult, UpdateResult, error) {
		return updateSkill(in)
	})
}

func updateSkill(in updateIn) (*mcp.CallToolResult, UpdateResult, error) {
	local, err := library.LocalDir()
	if err != nil {
		return nil, UpdateResult{}, err
	}
	l, err := library.Load()
	if err != nil {
		return nil, UpdateResult{}, err
	}
	s, err := l.Get(in.ID)
	if err == nil {
		if err := library.CheckEditable(s); err != nil {
			return nil, UpdateResult{}, fmt.Errorf("%w. To change it, ask the user to fork it into a local skill with \"quiver fork %s\", then update the fork", err, in.ID)
		}
	} else if _, serr := os.Stat(filepath.Join(local, in.ID)); strings.ContainsAny(in.ID, `/\`) || serr != nil {
		// An invalid local skill is not in the library, but can be fixed.
		return nil, UpdateResult{}, err
	}
	out := UpdateResult{ID: in.ID, Path: filepath.Join(local, in.ID)}
	out.Warnings, err = library.UpdateLocal(in.ID, in.Description, in.Body, in.Files)
	if err != nil {
		if errors.Is(err, library.ErrNothingToChange) {
			return nil, UpdateResult{}, err
		}
		return nil, UpdateResult{}, fmt.Errorf("skill %s was not changed: %w", in.ID, err)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Updated skill %s in %s.\n", out.ID, out.Path)
	for _, w := range out.Warnings {
		fmt.Fprintf(&b, "Warning: %s\n", w)
	}
	b.WriteString("get_skill returns the new version. A new description shows in the instructions from the next session on.")
	return text(b.String()), out, nil
}
