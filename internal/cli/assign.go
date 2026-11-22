package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"text/tabwriter"

	"github.com/kheif/quiver/internal/config"
	"github.com/kheif/quiver/internal/library"
	"github.com/kheif/quiver/internal/project"
	"github.com/kheif/quiver/internal/registry"
	"github.com/spf13/cobra"
)

// target is where add and remove change assignments: the global list or one
// project.
type target struct {
	global  bool
	project string // --project path; empty means the current folder
	always  bool   // also mark (add) or only unmark (remove) always-on
	yes     bool   // register a new project without asking
}

func (t *target) flags(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&t.global, "global", false, "change the skills served in every folder")
	cmd.Flags().StringVar(&t.project, "project", "", "project folder (default: the current folder)")
	cmd.MarkFlagsMutuallyExclusive("global", "project")
	projectFlag(cmd)
}

func (t *target) yesFlag(cmd *cobra.Command) {
	cmd.Flags().BoolVarP(&t.yes, "yes", "y", false, "register a new project without asking")
}

func (t *target) alwaysFlag(cmd *cobra.Command, usage string) {
	cmd.Flags().BoolVar(&t.always, "always", false, usage)
}

// dir returns the folder to look for the project in. A --project path must
// exist, or Find would walk up to a parent project.
func (t target) dir() (string, error) {
	if t.project == "" {
		return os.Getwd()
	}
	info, err := os.Stat(t.project)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("%s is not a folder", t.project)
	}
	return t.project, nil
}

// addOptions are the flags of add that are not part of the target.
type addOptions struct {
	searchable bool   // --searchable: the Searchable level instead of On demand
	source     string // --source: every unassigned skill of this source instead of IDs
}

func newAddCmd() *cobra.Command {
	var (
		t target
		o addOptions
	)
	cmd := &cobra.Command{
		Use:   "add <id>...",
		Short: "Assign skills to this project, or to every folder with --global",
		Long: `Assign skills to this project, or to every folder with --global.

A folder without .quiver.json is registered first; in a terminal it asks
before, unless --yes is given. Commit the new file, so the project keeps
its ID.

With --always the skills are also always-on: the assistant loads them before
its first reply, from the next session on.

With --searchable the skills are neither listed nor loaded at the start of a
session: the assistant finds them by searching and loads them when one fits.
A skill moves: --searchable takes it off the served list of the same scope,
and without the flag it leaves the Searchable list of that scope.

With --source <name> and --searchable, every skill of that source that is not
assigned yet in the scope becomes Searchable. A skill that is listed or
always-on there keeps its level.`,
		Example: `  # Assign two skills to the project in this folder
  quiver add pdf anthropic/docx

  # Let the assistant find a skill by searching, in every folder
  quiver add slack-style --searchable --global

  # Make the skills of a source Searchable, except those already assigned
  quiver add --source anthropic --searchable --global

  # Serve a skill in every folder, and load it before the first reply
  quiver add caveman --global --always

  # Assign to another project folder
  quiver add tdd --project ~/code/api`,
		ValidArgsFunction: skillIDs(false, 0),
		Args:              idsOrSource("source"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return add(cmd.InOrStdin(), cmd.OutOrStdout(), t, o, args)
		},
	}
	t.flags(cmd)
	t.yesFlag(cmd)
	t.alwaysFlag(cmd, "also make the skills always-on: loaded before the first reply")
	cmd.Flags().BoolVar(&o.searchable, "searchable", false, "make the skills Searchable: found by searching, not listed or loaded at session start")
	cmd.Flags().StringVar(&o.source, "source", "", "every skill of this source that is not assigned yet, instead of IDs (needs --searchable)")
	_ = cmd.RegisterFlagCompletionFunc("source", completeSources(false, 0))
	cmd.MarkFlagsMutuallyExclusive("searchable", "always")
	return cmd
}

// idsOrSource is the Args rule of add and remove: skill IDs, or none when
// --source takes their place.
func idsOrSource(flag string) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if name, _ := cmd.Flags().GetString(flag); name != "" {
			if len(args) > 0 {
				return fmt.Errorf("--%s takes the place of the skill IDs; give one or the other", flag)
			}
			return nil
		}
		return cobra.MinimumNArgs(1)(cmd, args)
	}
}

// sourceIDs returns the IDs of the library skills of the source, sorted. It
// is an error when no such source is configured.
func sourceIDs(l *library.Library, name string) ([]string, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	if _, ok := cfg.Sources[name]; !ok {
		return nil, fmt.Errorf("source %q not found; see quiver source list", name)
	}
	var ids []string
	for _, s := range l.All() {
		if s.Source == name {
			ids = append(ids, s.ID)
		}
	}
	return ids, nil
}

func add(in io.Reader, out io.Writer, t target, o addOptions, ids []string) error {
	if o.searchable && t.always {
		return fmt.Errorf("--searchable and --always cannot be combined: an always-on skill is loaded at session start")
	}
	if o.source != "" && !o.searchable {
		return fmt.Errorf("--source needs --searchable: it makes the skills of the source Searchable")
	}
	l, err := library.Load()
	if err != nil {
		return err
	}
	if o.source != "" {
		if ids, err = sourceIDs(l, o.source); err != nil {
			return err
		}
	}
	// Check every ID first, so one typo changes nothing.
	var errs []error
	for _, id := range ids {
		if _, err := l.Get(id); err != nil {
			errs = append(errs, err)
		}
	}
	if err := errors.Join(errs...); err != nil {
		return err
	}
	var dir string
	if !t.global {
		if dir, err = t.dir(); err != nil {
			return err
		}
		if ok, err := confirmProject(in, out, dir, !t.yes && interactive(in)); err != nil || !ok {
			return err
		}
	}
	var (
		reg     *registry.Registry
		p       *project.Project
		created bool
		copied  *registry.PossibleCopy
		before  map[string]registry.Level // the level of each ID in the scope, before the change
	)
	// assign changes the scope: the global one, or the project's. Under
	// --source it first drops the skills that are assigned already.
	assign := func(r *registry.Registry, projectID string) error {
		level := func(id string) registry.Level {
			if t.global {
				return r.GlobalLevel(id)
			}
			return r.ProjectLevel(projectID, id)
		}
		if o.source != "" {
			ids = slices.DeleteFunc(slices.Clone(ids), func(id string) bool { return level(id) != registry.NotAssigned })
		}
		before = map[string]registry.Level{}
		for _, id := range ids {
			before[id] = level(id)
		}
		switch {
		case o.searchable && t.global:
			r.AssignSearchableGlobal(ids...)
		case o.searchable:
			return r.AssignSearchable(projectID, ids...)
		case t.global:
			r.AssignGlobal(ids...)
			if t.always {
				return r.MarkAlwaysOnGlobal(ids...)
			}
		default:
			if err := r.Assign(projectID, ids...); err != nil {
				return err
			}
			if t.always {
				return r.MarkAlwaysOn(projectID, ids...)
			}
		}
		return nil
	}
	// Register runs under the registry lock too, so two processes cannot
	// write different IDs into the same new folder.
	err = registry.Update(func(r *registry.Registry) error {
		reg = r
		if t.global {
			return assign(r, "")
		}
		var err error
		p, err = project.Find(dir)
		if errors.Is(err, project.ErrNotRegistered) {
			created = true
			p, err = project.Register(dir)
		}
		if err != nil {
			return err
		}
		copied = r.EnsureProject(p.ID, p.Root)
		return assign(r, p.ID)
	})
	if err != nil {
		return err
	}
	projectID := ""
	verb := "Assigned"
	switch {
	case o.searchable:
		verb = "Made Searchable"
	case t.always:
		verb = "Assigned as always-on"
	}
	where := "every folder"
	if !t.global {
		projectID = p.ID
		where = reg.Projects[p.ID].Name
		reportProject(out, p, created, copied)
	}
	switch {
	case len(ids) == 0:
		fmt.Fprintf(out, "Every skill of source %s is assigned in %s already; nothing changed.\n", o.source, where)
	case t.global:
		fmt.Fprintf(out, "%s globally: %s\n", verb, strings.Join(ids, ", "))
	default:
		fmt.Fprintf(out, "%s to %s: %s\n", verb, where, strings.Join(ids, ", "))
	}
	for _, id := range ids {
		moved(out, id, before[id], o.searchable)
	}
	if !o.searchable {
		for _, c := range clashes(l, reg.Served(projectID), ids) {
			fmt.Fprintln(out, "warning:", c)
		}
	}
	if err := printServed(out, reg, projectID, where); err != nil {
		return err
	}
	sessionNote(out)
	if o.searchable && len(ids) > 0 {
		return offerEngineSetup(context.Background(), in, out, out)
	}
	return nil
}

// moved says when a change took the skill from another level of the scope.
func moved(out io.Writer, id string, before registry.Level, searchable bool) {
	switch {
	case searchable && before == registry.OnDemand:
		fmt.Fprintf(out, "note: %s was on demand there; it is Searchable now\n", id)
	case searchable && before == registry.AlwaysOn:
		fmt.Fprintf(out, "note: %s was always-on there; it is Searchable now, so it is no longer loaded at session start\n", id)
	case !searchable && before == registry.Searchable:
		fmt.Fprintf(out, "note: %s was Searchable there; it is on demand now\n", id)
	}
}

func newRemoveCmd() *cobra.Command {
	var (
		t      target
		source string
	)
	cmd := &cobra.Command{
		Use:   "remove <id>...",
		Short: "Unassign skills from this project, or from every folder with --global",
		Long: `Unassign skills from this project, or from every folder with --global. The
skills stay in the library. A skill is removed from every level of the scope:
always-on, on demand and Searchable.

With --always only the always-on mark is removed; the skills stay assigned.

With --source <name> every skill of that source is removed from the scope,
including those that the source no longer has.`,
		Example: `  quiver remove pdf
  quiver remove caveman --global

  # Undo quiver add --source acme --searchable --global
  quiver remove --source acme --global

  # Keep the skill assigned, but stop loading it before the first reply
  quiver remove caveman --global --always`,
		ValidArgsFunction: assignedIDs,
		Args:              idsOrSource("source"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return remove(cmd.OutOrStdout(), t, source, args)
		},
	}
	t.flags(cmd)
	t.alwaysFlag(cmd, "only remove the always-on mark and keep the skills assigned")
	cmd.Flags().StringVar(&source, "source", "", "every skill of this source, instead of IDs")
	_ = cmd.RegisterFlagCompletionFunc("source", completeSources(false, 0))
	cmd.MarkFlagsMutuallyExclusive("source", "always")
	return cmd
}

// remove does not look IDs up in the library, so a skill that was deleted or
// renamed in its source can still be unassigned.
func remove(out io.Writer, t target, source string, ids []string) error {
	var p *project.Project
	if !t.global {
		dir, err := t.dir()
		if err != nil {
			return err
		}
		p, err = project.Find(dir)
		if errors.Is(err, project.ErrNotRegistered) {
			return fmt.Errorf("%s is not a Quiver project, so it has no skills of its own; use --global for global skills", dir)
		}
		if err != nil {
			return err
		}
	}
	if source != "" {
		var err error
		if ids, err = assignedOfSource(t, p, source); err != nil {
			return err
		}
		if len(ids) == 0 {
			where := "every folder"
			if p != nil {
				where = filepath.Base(p.Root)
			}
			fmt.Fprintf(out, "No skill of source %s is assigned in %s.\n", source, where)
			return nil
		}
	}
	var (
		reg         *registry.Registry
		notAssigned []string
	)
	err := registry.Update(func(r *registry.Registry) error {
		reg = r
		switch {
		case t.global && t.always:
			notAssigned = r.UnmarkAlwaysOnGlobal(ids...)
			return nil
		case t.global:
			notAssigned = r.UnassignGlobal(ids...)
			return nil
		}
		if _, ok := r.Projects[p.ID]; !ok {
			notAssigned = ids
			return nil
		}
		var err error
		if t.always {
			notAssigned, err = r.UnmarkAlwaysOn(p.ID, ids...)
		} else {
			notAssigned, err = r.Unassign(p.ID, ids...)
		}
		return err
	})
	if err != nil {
		return err
	}
	projectID, where := "", "every folder"
	if !t.global {
		projectID, where = p.ID, filepath.Base(p.Root)
		if rp, ok := reg.Projects[p.ID]; ok {
			where = rp.Name
		}
	}
	skip := map[string]bool{}
	for _, id := range notAssigned {
		skip[id] = true
		switch {
		case t.always && t.global:
			fmt.Fprintf(out, "note: %s was not always-on globally\n", id)
		case t.always && slices.Contains(reg.AlwaysOn, id):
			fmt.Fprintf(out, "note: %s is always-on globally; remove the mark with --global\n", id)
		case t.always:
			fmt.Fprintf(out, "note: %s was not always-on in %s\n", id, where)
		case t.global:
			fmt.Fprintf(out, "note: %s was not global\n", id)
		case slices.Contains(reg.Global, id):
			fmt.Fprintf(out, "note: %s is global; remove it with --global\n", id)
		case slices.Contains(reg.Searchable, id):
			fmt.Fprintf(out, "note: %s is Searchable globally; remove it with --global\n", id)
		default:
			fmt.Fprintf(out, "note: %s was not assigned to %s\n", id, where)
		}
	}
	var removed []string
	for _, id := range ids {
		if !skip[id] {
			removed = append(removed, id)
		}
	}
	switch {
	case len(removed) > 0 && t.always:
		fmt.Fprintf(out, "No longer always-on in %s: %s\n", where, strings.Join(removed, ", "))
	case len(removed) > 0:
		fmt.Fprintf(out, "Removed from %s: %s\n", where, strings.Join(removed, ", "))
	}
	if err := printServed(out, reg, projectID, where); err != nil {
		return err
	}
	if len(removed) > 0 {
		sessionNote(out)
	}
	return nil
}

// assignedOfSource returns the IDs of the source's skills that the scope has
// at any level, whether the library still has them or not.
func assignedOfSource(t target, p *project.Project, source string) ([]string, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	if _, ok := cfg.Sources[source]; !ok {
		return nil, fmt.Errorf("source %q not found; see quiver source list", source)
	}
	r, err := registry.Load()
	if err != nil {
		return nil, err
	}
	var all []string
	switch {
	case t.global:
		all = slices.Concat(r.Global, r.Searchable)
	case r.Projects[p.ID] != nil:
		all = slices.Concat(r.Projects[p.ID].Skills, r.Projects[p.ID].Searchable)
	}
	var ids []string
	for _, id := range all {
		if strings.HasPrefix(id, source+"/") {
			ids = append(ids, id)
		}
	}
	slices.Sort(ids)
	return slices.Compact(ids), nil
}

// sessionNote says when assistants see a changed assignment.
func sessionNote(out io.Writer) {
	fmt.Fprintln(out, "Open assistant sessions see the change through list_skills; the catalog and always-on skills change from the next session.")
}

// confirmProject asks before a command registers a new project for dir, so
// a command run in the wrong folder does not create one there. It asks only
// when ask is set: a person is at a terminal and --yes was not given. ok is
// false when the answer is no.
func confirmProject(in io.Reader, out io.Writer, dir string, ask bool) (ok bool, err error) {
	if !ask {
		return true, nil
	}
	if _, err := project.Find(dir); !errors.Is(err, project.ErrNotRegistered) {
		return true, nil
	}
	root, err := project.NewRoot(dir)
	if err != nil {
		return false, err
	}
	if !confirmYes(in, out, "Register "+root+" as a project?") {
		fmt.Fprintln(out, "Not registered.")
		return false, nil
	}
	return true, nil
}

// reportProject tells the user about a new .quiver.json and a project
// that is in more than one folder.
func reportProject(out io.Writer, p *project.Project, created bool, copied *registry.PossibleCopy) {
	if created {
		fmt.Fprintf(out, "Created %s. Commit it, so the project keeps its ID.\n", filepath.Join(p.Root, project.FileName))
	}
	if copied != nil {
		fmt.Fprintln(out, "warning:", copied)
	}
}

// clashes describes added skills that share their name with another served
// skill. Assistants pick skills by name, so only one of them would be used.
func clashes(l *library.Library, served, added []string) []string {
	byName := map[string][]string{}
	for _, id := range served {
		if s, err := l.Get(id); err == nil {
			byName[s.Name] = append(byName[s.Name], id)
		}
	}
	var out []string
	seen := map[string]bool{}
	for _, id := range added {
		s, err := l.Get(id)
		if err != nil || seen[s.Name] || len(byName[s.Name]) < 2 {
			continue
		}
		seen[s.Name] = true
		out = append(out, fmt.Sprintf("skills %s share the name %q; keep only one", strings.Join(byName[s.Name], ", "), s.Name))
	}
	return out
}

// printServed lists the skills served in the project, each marked global or
// project, and always-on where it is. An empty projectID lists only the
// global skills. where names the place in the heading.
func printServed(out io.Writer, r *registry.Registry, projectID, where string) error {
	fmt.Fprintf(out, "Served in %s:\n", where)
	served := r.Served(projectID)
	searchable := r.SearchableServed(projectID)
	defer func() {
		if len(searchable) > 0 {
			fmt.Fprintf(out, "Searchable in %s (not listed; found by searching): %s\n", where, strings.Join(searchable, ", "))
		}
	}()
	if len(served) == 0 {
		fmt.Fprintln(out, "  (none)")
		return nil
	}
	global := map[string]bool{}
	for _, id := range r.Global {
		global[id] = true
	}
	always := r.AlwaysOnServed(projectID)
	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	for _, id := range served {
		scope := "project"
		if global[id] {
			scope = "global"
		}
		if slices.Contains(always, id) {
			scope += "\talways-on"
		}
		fmt.Fprintf(w, "  %s\t%s\n", id, scope)
	}
	return w.Flush()
}
