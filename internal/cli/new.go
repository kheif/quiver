package cli

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/kheif/quiver/internal/library"
	"github.com/kheif/quiver/internal/registry"
	"github.com/kheif/quiver/internal/skill"
	"github.com/spf13/cobra"
)

// todoDescription is the description of a skill created without one. edit
// warns while it is still there.
const todoDescription = "TODO — one sentence on when the assistant should use this skill."

func newNewCmd() *cobra.Command {
	var (
		t           target
		none        bool
		description string
	)
	cmd := &cobra.Command{
		Use:   "new <name>",
		Short: "Create a local skill and assign it to this project",
		Long: `Create a local skill in the Quiver home and assign it to this project, to
every folder with --global, or nowhere with --none.

Without --description the new SKILL.md opens in $VISUAL or $EDITOR.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return newSkill(cmd.OutOrStdout(), t, none, description, args[0])
		},
	}
	t.flags(cmd)
	cmd.Flags().BoolVar(&none, "none", false, "create the skill without assigning it")
	cmd.Flags().StringVar(&description, "description", "", "one sentence on when the assistant should use the skill; skips the editor")
	cmd.MarkFlagsMutuallyExclusive("global", "none")
	cmd.MarkFlagsMutuallyExclusive("project", "none")
	return cmd
}

func newSkill(out io.Writer, t target, none bool, description, name string) error {
	if err := skill.ValidateName(name); err != nil {
		return err
	}
	local, err := library.LocalDir()
	if err != nil {
		return err
	}
	// Check before a project is registered for nothing. CreateLocal checks
	// again under the lock.
	if _, err := os.Lstat(filepath.Join(local, name)); err == nil {
		return fmt.Errorf("skill %s already exists; change it with: quiver edit %s", name, name)
	}
	scope, projectID := library.ScopeNone, ""
	switch {
	case none:
	case t.global:
		scope = library.ScopeGlobal
	default:
		dir, err := t.dir()
		if err != nil {
			return err
		}
		p, created, copied, err := registry.RegisterProject(dir)
		if err != nil {
			return err
		}
		reportProject(out, p, created, copied)
		scope, projectID = library.ScopeProject, p.ID
	}
	edit := description == ""
	if edit {
		description = todoDescription
	}
	warnings, err := library.CreateLocal(name, description, template(name), nil, scope, projectID)
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "Created skill %s in %s\n", name, filepath.Join(local, name))
	if err := reportAssigned(out, name, scope, projectID); err != nil {
		return err
	}
	if edit {
		return editSkill(out, name)
	}
	for _, w := range warnings {
		fmt.Fprintln(out, "warning:", w)
	}
	return nil
}

// template is the body of a new skill: a heading from the name.
func template(name string) string {
	title := strings.ReplaceAll(name, "-", " ")
	return "# " + strings.ToUpper(title[:1]) + title[1:] + "\n\nInstructions go here."
}

// reportAssigned prints where the new skill is served, and warns when it
// shares its name with another served skill.
func reportAssigned(out io.Writer, name string, scope library.Scope, projectID string) error {
	if scope == library.ScopeNone {
		fmt.Fprintf(out, "Not assigned. Assign it with: quiver add %s\n", name)
		return nil
	}
	r, err := registry.Load()
	if err != nil {
		return err
	}
	if scope == library.ScopeGlobal {
		fmt.Fprintf(out, "Assigned globally: %s\n", name)
	} else {
		fmt.Fprintf(out, "Assigned to %s: %s\n", r.Projects[projectID].Name, name)
	}
	l, err := library.Load()
	if err != nil {
		return err
	}
	for _, c := range clashes(l, r.Served(projectID), []string{name}) {
		fmt.Fprintln(out, "warning:", c)
	}
	return nil
}

func newEditCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "edit <id>",
		Short: "Open a local skill in your editor and commit the change",
		Long: `Open a local skill's SKILL.md in $VISUAL or $EDITOR. When the editor closes,
the skill is checked and the change is committed in the Quiver home.

Skills from sources are read-only: "quiver update" would overwrite edits.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return editSkill(cmd.OutOrStdout(), args[0])
		},
	}
}

// editSkill opens the skill in the editor, then validates and commits it.
// A local skill is opened from its folder, not through the library, so a
// skill that an earlier edit broke can still be fixed.
func editSkill(out io.Writer, id string) error {
	local, err := library.LocalDir()
	if err != nil {
		return err
	}
	path := filepath.Join(local, id, skill.FileName)
	if !filepath.IsLocal(id) || strings.ContainsAny(id, `/\`) {
		path = ""
	} else if _, err := os.Stat(path); err != nil {
		path = ""
	}
	if path == "" {
		l, err := library.Load()
		if err != nil {
			return err
		}
		s, err := l.Get(id)
		if err != nil {
			return err
		}
		if err := library.CheckEditable(s); err != nil {
			return fmt.Errorf("%w; to change it, make a local copy with: quiver fork %s", err, id)
		}
		path = s.Path()
	}
	before, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := runEditor(path); err != nil {
		return fmt.Errorf("editor: %w", err)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if bytes.Equal(before, after) {
		fmt.Fprintln(out, "No changes.")
	} else {
		warnings, err := library.CommitEdit(id)
		if err != nil {
			return fmt.Errorf("%w\nthe change is saved but not committed; fix it with: quiver edit %s", err, id)
		}
		for _, w := range warnings {
			fmt.Fprintln(out, "warning:", w)
		}
		fmt.Fprintf(out, "Saved %s.\n", id)
	}
	if s, err := skill.Load(filepath.Dir(path)); err == nil && strings.HasPrefix(s.Description, "TODO") {
		fmt.Fprintf(out, "warning: the description of %s still starts with TODO; assistants pick skills by their description\n", id)
	}
	return nil
}

// runEditor opens path in $VISUAL or $EDITOR (default vi) and waits until it
// closes. Tests replace it.
var runEditor = func(path string) error {
	editor := "vi"
	for _, v := range []string{"VISUAL", "EDITOR"} {
		if e := strings.TrimSpace(os.Getenv(v)); e != "" {
			editor = e
			break
		}
	}
	// Split, so editors with flags such as "code --wait" work.
	args := strings.Fields(editor)
	c := exec.Command(args[0], append(args[1:], path)...)
	c.Stdin, c.Stdout, c.Stderr = os.Stdin, os.Stdout, os.Stderr
	return c.Run()
}
