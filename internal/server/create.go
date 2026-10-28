package server

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/kheif/quiver/internal/library"
	"github.com/kheif/quiver/internal/project"
	"github.com/kheif/quiver/internal/registry"
	"github.com/kheif/quiver/internal/skill"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type createIn struct {
	Name        string            `json:"name" jsonschema:"skill name: lowercase letters, digits and dashes, such as db-migration"`
	Description string            `json:"description" jsonschema:"one sentence on when an assistant should use the skill"`
	Body        string            `json:"body" jsonschema:"the instructions, as markdown without frontmatter"`
	Files       map[string]string `json:"files,omitempty" jsonschema:"extra text files: a relative path inside the skill folder, such as references/checklist.md, mapped to its content"`
	Scope       string            `json:"scope" jsonschema:"project: serve the skill only in this project; global: serve it in every folder"`
}

// CreateResult is the create_skill result.
type CreateResult struct {
	ID    string `json:"id"`
	Path  string `json:"path"`
	Scope string `json:"scope"`
	// ProjectFile is the .quiver.json created for the project, if any.
	ProjectFile string   `json:"projectFile,omitempty"`
	Warnings    []string `json:"warnings,omitempty"`
}

// addCreateTool registers create_skill.
func addCreateTool(s *mcp.Server, sess *session) {
	schema, err := jsonschema.For[createIn](nil)
	if err != nil {
		panic(err) // createIn is fixed, so this fails only while developing
	}
	schema.Properties["scope"].Enum = []any{string(library.ScopeProject), string(library.ScopeGlobal)}
	notDestructive := false
	mcp.AddTool(s, &mcp.Tool{
		Name: "create_skill",
		Description: `Create a new skill in the user's Quiver library, for example to keep a workflow from this conversation for later sessions. ` +
			`Scope "project" serves it only in this project; "global" serves it in every folder. ` +
			`If the user has not said which scope they want, ask them before you call this tool. ` +
			`The new skill can be loaded with get_skill right away.`,
		InputSchema: schema,
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: false, DestructiveHint: &notDestructive},
	}, func(_ context.Context, _ *mcp.CallToolRequest, in createIn) (*mcp.CallToolResult, CreateResult, error) {
		return createSkill(sess, in)
	})
}

func createSkill(sess *session, in createIn) (*mcp.CallToolResult, CreateResult, error) {
	scope := library.Scope(in.Scope)
	if scope != library.ScopeProject && scope != library.ScopeGlobal {
		return nil, CreateResult{}, errors.New(`scope must be "project" or "global"; ask the user which one they want`)
	}
	local, err := library.LocalDir()
	if err != nil {
		return nil, CreateResult{}, err
	}
	if err := skill.ValidateName(in.Name); err != nil {
		if alt := slug(in.Name); alt != "" {
			err = fmt.Errorf("%w; use a name such as %q", err, freeName(local, alt))
		}
		return nil, CreateResult{}, err
	}
	// Check before a project is registered for nothing. CreateLocal checks
	// again under the lock.
	if _, err := os.Lstat(filepath.Join(local, in.Name)); err == nil {
		return nil, CreateResult{}, taken(local, in.Name)
	}
	out := CreateResult{ID: in.Name, Path: filepath.Join(local, in.Name), Scope: in.Scope}
	where, projectID := "every folder", ""
	if scope == library.ScopeProject {
		p, created, err := sess.project()
		if err != nil {
			return nil, CreateResult{}, err
		}
		if created {
			out.ProjectFile = filepath.Join(p.Root, project.FileName)
		}
		projectID, where = p.ID, filepath.Base(p.Root)
	}
	out.Warnings, err = library.CreateLocal(in.Name, in.Description, in.Body, in.Files, scope, projectID)
	if errors.Is(err, library.ErrSkillExists) {
		return nil, CreateResult{}, taken(local, in.Name)
	}
	if err != nil {
		return nil, CreateResult{}, err
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Created skill %s in %s. It is served in %s.\n", out.ID, out.Path, where)
	if out.ProjectFile != "" {
		fmt.Fprintf(&b, "Created %s for this project. Tell the user to commit it, so the project keeps its ID.\n", out.ProjectFile)
	}
	for _, w := range out.Warnings {
		fmt.Fprintf(&b, "Warning: %s\n", w)
	}
	b.WriteString("Load it with get_skill when it applies. The instructions list it from the next session on.")
	return text(b.String()), out, nil
}

// project returns the served project. When the session has none, it
// registers the session's folder, and later requests serve that project.
// A project missing from the registry is added to it.
func (s *session) project() (p *project.Project, created bool, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	dir := s.where.Dir
	if s.where.Project != nil {
		dir = s.where.Project.Root
	}
	if dir == "" {
		return nil, false, errors.New(`the server has no project folder, so it cannot create a project skill; use scope "global", or start the server in the project`)
	}
	p, created, _, err = registry.RegisterProject(dir)
	if err != nil {
		return nil, false, err
	}
	s.where.Project = p
	return p, created, nil
}

// taken is the error for a name that is in use, with a free one to use.
func taken(local, name string) error {
	return fmt.Errorf("a skill named %q already exists; use another name, such as %q, or ask the user to change the existing skill with \"quiver edit %s\"",
		name, freeName(local, name), name)
}

// slug turns s into a valid skill name: lowercase letters, digits and
// single dashes. It returns "" when nothing is left.
func slug(s string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(s) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			if dash && b.Len() > 0 {
				b.WriteByte('-')
			}
			b.WriteRune(r)
			dash = false
		} else {
			dash = true
		}
	}
	out := b.String()
	if len(out) > 60 {
		out = strings.TrimRight(out[:60], "-")
	}
	return out
}

// freeName returns name if no folder in local/ has it, or else name with
// the lowest free number appended: name-2, name-3 and so on.
func freeName(local, name string) string {
	candidate := name
	for i := 2; ; i++ {
		if _, err := os.Lstat(filepath.Join(local, candidate)); err != nil {
			return candidate
		}
		candidate = fmt.Sprintf("%s-%d", name, i)
	}
}
