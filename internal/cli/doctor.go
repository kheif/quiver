package cli

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/kheif/quiver/internal/assistant"
	"github.com/kheif/quiver/internal/config"
	"github.com/kheif/quiver/internal/fsutil"
	"github.com/kheif/quiver/internal/git"
	"github.com/kheif/quiver/internal/library"
	"github.com/kheif/quiver/internal/registry"
	"github.com/kheif/quiver/internal/server"
	"github.com/kheif/quiver/internal/setup"
	"github.com/kheif/quiver/internal/skill"
	"github.com/spf13/cobra"
)

func newDoctorCmd() *cobra.Command {
	var fix bool
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Check the Quiver home, the library and the Claude Code setup",
		Long: `Check the Quiver home, the library, the assignments and the Claude Code
setup, and report each check as ok, warn or error.

With --fix it applies the safe fixes: it creates missing parts of the Quiver
home, commits uncommitted changes in local/, deletes folders left over from
an interrupted source add or remove, and, after asking, unassigns skills
that are not in the library and deletes copies of served skills that
Claude Code also loads by itself. Copies committed in git and copies in
plugins are only reported.`,
		Example: `  quiver doctor
  quiver doctor --fix`,
		ValidArgsFunction: noArgs,
		Args:              cobra.NoArgs,
		// Check the home as it is; the root command would repair it first.
		PersistentPreRunE: func(*cobra.Command, []string) error { return nil },
		RunE: func(cmd *cobra.Command, _ []string) error {
			return doctor(cmd.InOrStdin(), cmd.OutOrStdout(), fix)
		},
	}
	cmd.Flags().BoolVar(&fix, "fix", false, "apply the safe fixes")
	return cmd
}

// doc runs the checks. Later checks use what earlier ones loaded and are
// skipped when it could not be loaded.
type doc struct {
	in       *bufio.Reader
	out      io.Writer
	fix      bool
	home     string
	cfg      *config.Config
	reg      *registry.Registry
	lib      *library.Library
	errors   int
	warnings int
}

func doctor(in io.Reader, out io.Writer, fix bool) error {
	home, err := fsutil.QuiverHome()
	if err != nil {
		return err
	}
	d := &doc{in: bufio.NewReader(in), out: out, fix: fix, home: home}
	for _, check := range []func(){d.checkHome, d.checkLocalRepo, d.checkSkills, d.checkAssigned, d.checkSources, d.checkProjects, d.checkClashes, d.checkNative, d.checkMCP} {
		check()
	}
	fmt.Fprintln(out)
	switch {
	case d.errors > 0:
		return fmt.Errorf("%s, %s", plural(d.errors, "error"), plural(d.warnings, "warning"))
	case d.warnings > 0:
		fmt.Fprintf(out, "%s, no errors.\n", plural(d.warnings, "warning"))
	default:
		fmt.Fprintln(out, "Everything is fine.")
	}
	return nil
}

// report prints one result. level is "ok", "warn" or "error"; details are
// printed indented below it.
func (d *doc) report(level, title string, details ...string) {
	switch level {
	case "warn":
		d.warnings++
	case "error":
		d.errors++
	}
	fmt.Fprintf(d.out, "%-5s  %s\n", level, title)
	for _, line := range details {
		fmt.Fprintf(d.out, "       %s\n", line)
	}
}

// fixed prints the result of a fix and takes back the problem it fixed.
func (d *doc) fixed(level string, err error, done string) {
	if err != nil {
		fmt.Fprintf(d.out, "       fix failed: %v\n", err)
		return
	}
	fmt.Fprintf(d.out, "       fixed: %s\n", done)
	if level == "error" {
		d.errors--
	} else {
		d.warnings--
	}
}

func (d *doc) skip(title, reason string) {
	fmt.Fprintf(d.out, "%-5s  %s (skipped: %s)\n", "-", title, reason)
}

// 1. The layout of the Quiver home and its JSON files.
func (d *doc) checkHome() {
	const title = "Quiver home layout"
	var missing []string
	for _, p := range []string{"", "local", "sources", "config.json", "registry.json", filepath.Join("local", ".git")} {
		if _, err := os.Stat(filepath.Join(d.home, p)); errors.Is(err, os.ErrNotExist) {
			missing = append(missing, filepath.Join(d.home, p)+" is missing")
		}
	}
	if len(missing) > 0 {
		d.report("error", title, missing...)
		if d.fix {
			_, err := setup.Init()
			d.fixed("error", err, "created the missing parts")
		}
	} else {
		d.report("ok", title+": "+d.home)
	}
	cfg, err := config.Load()
	if err != nil {
		d.report("error", "config.json", err.Error())
	} else {
		d.cfg = cfg
		d.report("ok", "config.json")
	}
	reg, err := registry.Load()
	if err != nil {
		d.report("error", "registry.json", err.Error())
	} else {
		d.reg = reg
		d.report("ok", "registry.json")
	}
}

// 2. local/ is a git repository without uncommitted changes.
func (d *doc) checkLocalRepo() {
	const title = "local/ is committed"
	local := filepath.Join(d.home, "local")
	if !git.HasCommit(local) {
		d.report("error", "local/ is not a git repository with a commit; run quiver doctor --fix")
		return
	}
	status, err := git.Status(local)
	if err != nil {
		d.report("error", title, err.Error())
		return
	}
	if status == "" {
		d.report("ok", title)
		return
	}
	d.report("warn", "local/ has uncommitted changes", strings.Split(status, "\n")...)
	if d.fix {
		err := fsutil.WithLock(func() error { return git.Commit(local, "Commit changes found by quiver doctor") })
		d.fixed("warn", err, "committed them")
	}
}

// 3. Every library skill loads and validates.
func (d *doc) checkSkills() {
	const title = "library skills are valid"
	if d.cfg == nil {
		d.skip(title, "config.json is invalid")
		return
	}
	lib, err := library.Load()
	if err != nil {
		d.report("error", title, err.Error())
		return
	}
	d.lib = lib
	var skipped, soft []string
	for _, w := range lib.Warnings {
		switch {
		case strings.HasSuffix(w, "is not downloaded; add it again"):
			// Reported by checkSources.
		case strings.HasPrefix(w, "skipped "):
			skipped = append(skipped, w)
		default:
			soft = append(soft, w)
		}
	}
	switch {
	case len(skipped) > 0:
		d.report("error", "library skills that cannot be used", append(skipped, soft...)...)
	case len(soft) > 0:
		d.report("warn", "library skills with warnings", soft...)
	default:
		d.report("ok", fmt.Sprintf("%s: %s", title, plural(len(lib.All()), "skill")))
	}
}

// 4. Every assigned ID is in the library.
func (d *doc) checkAssigned() {
	const title = "assigned skills are in the library"
	if d.reg == nil || d.lib == nil {
		d.skip(title, "the registry or the library could not be loaded")
		return
	}
	inLib := func(id string) bool { _, err := d.lib.Get(id); return err == nil }
	// A source that is not downloaded loses all its skills; adding it again
	// is the fix, not unassigning them.
	notDownloaded := func(id string) string {
		name, _, ok := strings.Cut(id, "/")
		if !ok {
			return ""
		}
		if _, configured := d.cfg.Sources[name]; !configured {
			return ""
		}
		if _, err := os.Stat(filepath.Join(d.home, "sources", name)); errors.Is(err, os.ErrNotExist) {
			return name
		}
		return ""
	}
	var details []string
	global := []string{}
	projects := map[string][]string{}
	for _, id := range d.reg.Global {
		if inLib(id) {
			continue
		}
		if src := notDownloaded(id); src != "" {
			details = append(details, fmt.Sprintf("global: %s (source %s is not downloaded)", id, src))
			continue
		}
		details = append(details, "global: "+id)
		global = append(global, id)
	}
	ids := make([]string, 0, len(d.reg.Projects))
	for id := range d.reg.Projects {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, pid := range ids {
		p := d.reg.Projects[pid]
		for _, id := range p.Skills {
			if inLib(id) {
				continue
			}
			if src := notDownloaded(id); src != "" {
				details = append(details, fmt.Sprintf("project %s: %s (source %s is not downloaded)", p.Name, id, src))
				continue
			}
			details = append(details, fmt.Sprintf("project %s: %s", p.Name, id))
			projects[pid] = append(projects[pid], id)
		}
	}
	if len(details) == 0 {
		d.report("ok", title)
		return
	}
	d.report("error", "assigned skills that are not in the library, so they are not served", details...)
	n := len(global)
	for _, skills := range projects {
		n += len(skills)
	}
	if !d.fix || n == 0 {
		return
	}
	if !confirm(d.in, d.out, fmt.Sprintf("       Unassign %s?", plural(n, "missing skill"))) {
		fmt.Fprintln(d.out, "       Not unassigned.")
		return
	}
	err := registry.Update(func(r *registry.Registry) error {
		r.UnassignGlobal(global...)
		for pid, skills := range projects {
			if _, err := r.Unassign(pid, skills...); err != nil {
				return err
			}
		}
		return nil
	})
	if n < len(details) {
		// Skills of a source that is not downloaded are still assigned.
		if err == nil {
			fmt.Fprintf(d.out, "       fixed: unassigned %s\n", plural(n, "skill"))
		} else {
			fmt.Fprintf(d.out, "       fix failed: %v\n", err)
		}
		return
	}
	d.fixed("error", err, "unassigned "+plural(n, "skill"))
}

// 5. Every configured source is downloaded, and no temp folder is left.
func (d *doc) checkSources() {
	const title = "sources are downloaded"
	if d.cfg == nil {
		d.skip(title, "config.json is invalid")
		return
	}
	dir := filepath.Join(d.home, "sources")
	var missing []string
	names := make([]string, 0, len(d.cfg.Sources))
	for name := range d.cfg.Sources {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if _, err := os.Stat(filepath.Join(dir, name)); errors.Is(err, os.ErrNotExist) {
			missing = append(missing, fmt.Sprintf("%s (%s): remove it and add it again", name, d.cfg.Sources[name].Repo))
		}
	}
	if len(missing) > 0 {
		d.report("error", "sources that are not downloaded", missing...)
	} else {
		d.report("ok", fmt.Sprintf("%s: %s", title, plural(len(names), "source")))
	}
	entries, err := os.ReadDir(dir)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		d.report("error", "sources/ cannot be read", err.Error())
		return
	}
	var leftovers []string
	for _, e := range entries {
		if e.IsDir() && (strings.HasPrefix(e.Name(), ".add-") || strings.HasPrefix(e.Name(), ".remove-")) {
			leftovers = append(leftovers, filepath.Join(dir, e.Name()))
		}
	}
	if len(leftovers) == 0 {
		return
	}
	d.report("warn", "folders left over from an interrupted source add or remove", leftovers...)
	if d.fix {
		err := fsutil.WithLock(func() error {
			for _, p := range leftovers {
				if err := os.RemoveAll(p); err != nil {
					return err
				}
			}
			return nil
		})
		d.fixed("warn", err, "deleted them")
	}
}

// 6. Projects with no folder on this machine.
func (d *doc) checkProjects() {
	const title = "project folders exist"
	if d.reg == nil {
		d.skip(title, "registry.json is invalid")
		return
	}
	var gone []string
	for id, p := range d.reg.Projects {
		if len(p.Paths) > 0 && !slices.ContainsFunc(p.Paths, exists) {
			gone = append(gone, fmt.Sprintf("%s [%s]: %s", p.Name, id, strings.Join(p.Paths, ", ")))
		}
	}
	if len(gone) == 0 {
		d.report("ok", title)
		return
	}
	sort.Strings(gone)
	gone = append(gone, "They may exist on another machine. If not: quiver projects forget <id>")
	d.report("warn", "projects with no folder on this machine", gone...)
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// 7. Two served skills with the same name, in any folder.
func (d *doc) checkClashes() {
	const title = "no two served skills share a name"
	if d.reg == nil || d.lib == nil {
		d.skip(title, "the registry or the library could not be loaded")
		return
	}
	seen := map[string]bool{}
	var found []string
	check := func(served []string, where string) {
		byName := map[string][]string{}
		for _, id := range served {
			if s, err := d.lib.Get(id); err == nil {
				byName[s.Name] = append(byName[s.Name], id)
			}
		}
		for name, ids := range byName {
			key := strings.Join(ids, " ")
			if len(ids) < 2 || seen[key] {
				continue
			}
			seen[key] = true
			found = append(found, fmt.Sprintf("%s: %s share the name %q", where, strings.Join(ids, ", "), name))
		}
	}
	// Global clashes first, so they are reported once, not per project.
	check(d.reg.Served(""), "every folder")
	ids := make([]string, 0, len(d.reg.Projects))
	for id := range d.reg.Projects {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		check(d.reg.Served(id), "project "+d.reg.Projects[id].Name)
	}
	if len(found) == 0 {
		d.report("ok", title)
		return
	}
	sort.Strings(found)
	found = append(found, "The assistant uses only one of them; unassign or rename the others.")
	d.report("warn", "served skills that share a name", found...)
}

// 8. Skills served here that Claude Code also loads by itself.
func (d *doc) checkNative() {
	const title = "no served skill is also installed in Claude Code"
	if d.reg == nil || d.lib == nil {
		d.skip(title, "the registry or the library could not be loaded")
		return
	}
	copies, err := servedCopies(d.reg, d.lib)
	if err != nil {
		d.report("error", title, err.Error())
		return
	}
	if len(copies) == 0 {
		d.report("ok", title)
		return
	}
	var details []string
	var deletable []library.NativeCopy
	for _, c := range copies {
		line := c.ID + ": " + describeCopy(c)
		switch {
		case c.Plugin != "":
			line += "; disable the plugin in Claude Code with /plugin"
		case c.Tracked:
			line += "; it is committed in git, delete it there if no one needs it"
		default:
			deletable = append(deletable, c)
		}
		details = append(details, line)
	}
	details = append(details, "The assistant may load these skills twice.")
	d.report("warn", "served skills that are also installed in Claude Code", details...)
	if !d.fix || len(deletable) == 0 {
		return
	}
	if !confirm(d.in, d.out, fmt.Sprintf("       Delete %s?", plural(len(deletable), "copy"))) {
		fmt.Fprintln(d.out, "       Not deleted.")
		return
	}
	for _, c := range deletable {
		if err = os.RemoveAll(c.Dir); err != nil {
			break
		}
	}
	if len(deletable) < len(copies) {
		// Plugin and committed copies are still there.
		if err == nil {
			fmt.Fprintf(d.out, "       fixed: deleted %s\n", plural(len(deletable), "copy"))
		} else {
			fmt.Fprintf(d.out, "       fix failed: %v\n", err)
		}
		return
	}
	d.fixed("warn", err, "deleted "+plural(len(deletable), "copy"))
}

// servedCopies returns the Claude Code skills that are copies of a skill
// served in the folder the server would use.
func servedCopies(r *registry.Registry, l *library.Library) ([]library.NativeCopy, error) {
	w, err := server.Resolve()
	if err != nil {
		return nil, err
	}
	var served []*skill.Skill
	for _, id := range r.Served(w.ProjectID()) {
		if s, err := l.Get(id); err == nil {
			served = append(served, s)
		}
	}
	root := ""
	if w.Project != nil {
		root = w.Project.Root
	}
	return library.NativeCopies(served, root)
}

// describeCopy says where a copy is and why it counts as one.
func describeCopy(c library.NativeCopy) string {
	why := "same name"
	switch {
	case c.SameName && c.SameText:
		why = "same name and content"
	case c.SameText:
		why = "same content"
	}
	where := c.Dir
	if c.Plugin != "" {
		where = "plugin " + c.Plugin
	}
	return fmt.Sprintf("%s (%s)", where, why)
}

// 9. Claude Code starts this quiver as its MCP server.
func (d *doc) checkMCP() {
	const title = "registered with Claude Code"
	reg, err := assistant.Get("")
	switch {
	case errors.Is(err, assistant.ErrNoClaude):
		d.report("warn", "the claude command was not found, so the Claude Code setup was not checked")
		return
	case err != nil:
		d.report("error", title, err.Error())
		return
	case reg == nil:
		d.report("error", "not registered with Claude Code; run: quiver mcp install")
		return
	}
	bin, _, err := command()
	if err == nil && reg.Command != bin {
		d.report("warn", "Claude Code runs another quiver", fmt.Sprintf("it runs %s %s, but this is %s; run: quiver mcp install", reg.Command, reg.Args, bin))
		return
	}
	if !reg.User() {
		d.report("ok", fmt.Sprintf("%s (%s)", title, reg.Scope))
		return
	}
	d.report("ok", title)
}
