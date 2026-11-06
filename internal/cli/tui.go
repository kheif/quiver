package cli

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
	"sync"

	"github.com/kheif/quiver/internal/config"
	"github.com/kheif/quiver/internal/fsutil"
	"github.com/kheif/quiver/internal/library"
	"github.com/kheif/quiver/internal/project"
	"github.com/kheif/quiver/internal/registry"
	"github.com/kheif/quiver/internal/server"
	"github.com/kheif/quiver/internal/skill"
	"github.com/kheif/quiver/internal/source"
	"github.com/kheif/quiver/internal/tui"
)

// runTUI opens the interactive panel. Tests replace it.
var runTUI = tui.Run

// tuiBackend fills the panel with the functions the commands use. After a
// change it gives the assistants the hook commands the new skills need, as
// the root command does after the commands that change skills.
type tuiBackend struct {
	mu      sync.Mutex
	lib     *library.Library
	pending map[string]*library.Update // fetched sources waiting for Apply
}

// firstExisting returns the first of paths that is a folder, or "".
func firstExisting(paths []string) string {
	for _, p := range paths {
		if info, err := os.Stat(p); err == nil && info.IsDir() {
			return p
		}
	}
	return ""
}

func (b *tuiBackend) library() *library.Library {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.lib
}

func (b *tuiBackend) Load(projectID string) (tui.Data, error) {
	if !homeExists() {
		d := tui.Data{Version: currentVersion(), NoHome: true}
		d.Home, _ = fsutil.QuiverHome()
		return d, nil
	}
	l, err := library.Load()
	if err != nil {
		return tui.Data{}, err
	}
	b.mu.Lock()
	b.lib = l
	b.mu.Unlock()
	r, err := registry.Load()
	if err != nil {
		return tui.Data{}, err
	}
	cfg, err := config.Load()
	if err != nil {
		return tui.Data{}, err
	}
	w, err := server.Resolve()
	if err != nil {
		return tui.Data{}, err
	}
	here := w.ProjectID()
	d := tui.Data{Version: currentVersion()}
	d.Home, _ = fsutil.QuiverHome()
	for id, p := range r.Projects {
		path := firstExisting(p.Paths)
		skills := slices.Clone(p.Skills)
		sort.Strings(skills)
		always := slices.Clone(p.AlwaysOn)
		sort.Strings(always)
		folders := 0
		for _, f := range p.Paths {
			if firstExisting([]string{f}) != "" {
				folders++
			}
		}
		d.Projects = append(d.Projects, tui.Project{
			Name: p.Name, ID: id, Skills: slices.Compact(skills), Paths: p.Paths, Path: path,
			Current: id == here, Missing: len(p.Paths) > 0 && path == "",
			AlwaysOn: slices.Compact(always), Folders: folders,
		})
	}
	sort.Slice(d.Projects, func(i, j int) bool {
		a, c := d.Projects[i], d.Projects[j]
		return a.Name+"\x00"+a.ID < c.Name+"\x00"+c.ID
	})

	// The project the panel manages: the one that was chosen, or the one of
	// this folder. A chosen project that is gone falls back to this folder's.
	scope := projectID
	if !slices.ContainsFunc(d.Projects, func(p tui.Project) bool { return p.ID == scope }) {
		scope = here
	}
	var managed *registry.Project
	switch i := slices.IndexFunc(d.Projects, func(p tui.Project) bool { return p.ID == scope }); {
	case scope == "":
	case i >= 0:
		managed = r.Projects[scope]
		p := d.Projects[i]
		d.Project = &p
	case w.Project != nil:
		// A project of this folder that is not in the registry yet: its first
		// assignment registers it.
		d.Project = &tui.Project{Name: filepath.Base(w.Project.Root), ID: here, Paths: []string{w.Project.Root}, Path: firstExisting([]string{w.Project.Root}), Current: true}
	}
	if i := slices.IndexFunc(d.Projects, func(p tui.Project) bool { return p.ID == here }); i >= 0 {
		d.Opened = d.Projects[i].Name
	} else if w.Project != nil {
		d.Opened = filepath.Base(w.Project.Root)
	}

	for _, s := range l.All() {
		ts := tui.Skill{
			ID: s.ID, Name: s.Name, Source: s.Source, Description: s.Description,
			Local:  library.CheckEditable(s) == nil,
			Global: slices.Contains(r.Global, s.ID), GlobalAlways: slices.Contains(r.AlwaysOn, s.ID),
			UsedIn: r.ProjectsUsing(s.ID),
		}
		if pin, ok := cfg.Pins[s.ID]; ok {
			ts.Pin = pin.Commit[:min(7, len(pin.Commit))]
		}
		if managed != nil {
			ts.Project = slices.Contains(managed.Skills, s.ID)
			ts.ProjectAlways = ts.Project && slices.Contains(managed.AlwaysOn, s.ID)
		}
		d.Skills = append(d.Skills, ts)
	}
	sources, err := l.Sources()
	if err != nil {
		return tui.Data{}, err
	}
	for _, s := range sources {
		ts := tui.Source{Name: s.Name, Skills: s.Skills, Commit: s.Commit, Repo: s.Repo}
		if !s.Updated.IsZero() {
			ts.Updated = s.Updated.Format("2006-01-02")
		}
		for _, sk := range d.Skills {
			if sk.Source == s.Name && sk.Pin != "" {
				ts.Pinned = append(ts.Pinned, sk.ID+"@"+sk.Pin)
			}
		}
		d.Sources = append(d.Sources, ts)
	}
	return d, nil
}

// Health runs the checks of quiver doctor and says how always-on skills reach
// the assistants. It asks the assistants, so it is slow and the panel runs it
// on its own.
func (b *tuiBackend) Health() (tui.HealthReport, error) {
	if !homeExists() {
		return tui.HealthReport{}, nil
	}
	items, hookMissing := health()
	return tui.HealthReport{Items: items, HookMissing: hookMissing}, nil
}

// health runs the checks of quiver doctor that read only this machine, and
// adds how always-on skills reach each assistant. hookMissing is set when an
// installed assistant has no session-start hook, so always-on skills do not
// load there.
func health() (items []tui.HealthItem, hookMissing bool) {
	home, err := fsutil.QuiverHome()
	if err != nil {
		return nil, false
	}
	var out bytes.Buffer
	d := &doc{in: bufio.NewReader(strings.NewReader("")), out: &out, home: home}
	for _, check := range d.checks() {
		check()
	}
	items = parseChecks(out.String())
	ts, err := mcpTargets(nil)
	if err != nil {
		return items, false
	}
	var lines []string
	for _, t := range ts {
		del, missing := t.actualDelivery()
		hookMissing = hookMissing || missing
		line := t.name() + ": " + del.Describe()
		if missing {
			line += " (its session-start hook is missing; press i)"
		}
		lines = append(lines, line)
	}
	if len(lines) == 0 {
		lines = []string{"No supported assistant was found on this machine."}
	}
	return append(items, tui.HealthItem{Level: "ok", Title: "How always-on skills reach your assistants", Details: lines}), hookMissing
}

// parseChecks reads the lines doctor prints: "level  title", then the details
// of a result indented below it.
func parseChecks(text string) []tui.HealthItem {
	var items []tui.HealthItem
	for _, l := range strings.Split(text, "\n") {
		switch {
		case strings.TrimSpace(l) == "":
		case strings.HasPrefix(l, "       "):
			if n := len(items); n > 0 {
				items[n-1].Details = append(items[n-1].Details, strings.TrimSpace(l))
			}
		default:
			level, title, ok := strings.Cut(l, " ")
			if !ok {
				continue
			}
			if level == "-" {
				level = "ok"
			}
			items = append(items, tui.HealthItem{Level: level, Title: strings.TrimSpace(title)})
		}
	}
	return items
}

func (b *tuiBackend) Search(text string) ([]string, error) {
	l := b.library()
	if l == nil {
		return nil, errors.New("the library is not loaded")
	}
	var ids []string
	for _, m := range l.Search(text) {
		ids = append(ids, m.ID)
	}
	return ids, nil
}

func (b *tuiBackend) Details(id string) (tui.Details, error) {
	l := b.library()
	if l == nil {
		return tui.Details{}, errors.New("the library is not loaded")
	}
	s, err := l.Get(id)
	if err != nil {
		return tui.Details{}, err
	}
	raw, err := os.ReadFile(s.Path())
	return tui.Details{Body: s.Body, Raw: string(raw)}, err
}

// modeIn is how the registry has a skill in one scope: the global one, or the
// project in the folder at path.
func modeIn(r *registry.Registry, id string, global bool, path string) tui.Mode {
	assigned, always := slices.Contains(r.Global, id), slices.Contains(r.AlwaysOn, id)
	if !global {
		assigned, always = false, false
		if p, err := project.Find(path); err == nil {
			if rp, ok := r.Projects[p.ID]; ok {
				assigned, always = slices.Contains(rp.Skills, id), slices.Contains(rp.AlwaysOn, id)
			}
		}
	}
	switch {
	case !assigned:
		return tui.NotAssigned
	case always:
		return tui.AlwaysOn
	}
	return tui.OnDemand
}

// set moves a skill to a mode in one scope with the same functions as
// quiver add and quiver remove, and leaves the other scope alone. It returns
// the warnings those commands print.
func (b *tuiBackend) set(id string, t target, to tui.Mode) (string, error) {
	r, err := registry.Load()
	if err != nil {
		return "", err
	}
	from := modeIn(r, id, t.global, t.project)
	if from == to {
		return "", nil
	}
	var out bytes.Buffer
	ids := []string{id}
	switch {
	case to == tui.NotAssigned:
		err = remove(&out, t, ids)
	case to == tui.AlwaysOn:
		t.always = true
		err = add(strings.NewReader(""), &out, t, ids)
	case from == tui.AlwaysOn:
		// Keep it assigned and drop only the always-on mark.
		t.always = true
		err = remove(&out, t, ids)
	default:
		err = add(strings.NewReader(""), &out, t, ids)
	}
	syncHooks(io.Discard)
	if err != nil {
		return "", err
	}
	return warnings(out.String()), nil
}

func (b *tuiBackend) SetProject(id, projectPath string, to tui.Mode) (string, error) {
	return b.set(id, target{project: projectPath, yes: true}, to)
}

func (b *tuiBackend) SetGlobal(id string, to tui.Mode) (string, error) {
	return b.set(id, target{global: true, yes: true}, to)
}

func (b *tuiBackend) Edit(id string) (string, error) {
	var out bytes.Buffer
	err := editSkill(&out, id)
	syncHooks(io.Discard)
	return firstLine(out.String()), err
}

func (b *tuiBackend) Fork(id, name string) (string, string, error) {
	f, err := library.ForkRemote(id, name)
	if err != nil {
		return "", "", forkError(err)
	}
	syncHooks(io.Discard)
	note := fmt.Sprintf("Forked %s into %s. Change it with the e key.", id, f.Name)
	if f.Replaced.Empty() {
		note += " It was not assigned anywhere."
	} else {
		note += " The copy took the original's place where it was assigned."
	}
	for _, w := range f.Warnings {
		note += " warning: " + w
	}
	return f.Name, note, nil
}

// Check fetches the source like quiver update does and keeps the result for
// Apply.
func (b *tuiBackend) Check(name string, reset bool) (tui.Preview, error) {
	r, err := registry.Load()
	if err != nil {
		return tui.Preview{}, err
	}
	u, err := library.FetchSource(name, reset)
	if err != nil {
		if errors.Is(err, library.ErrDiverged) {
			return tui.Preview{}, fmt.Errorf("%w: %s", tui.ErrDiverged, name)
		}
		return tui.Preview{}, err
	}
	var out bytes.Buffer
	ask := printUpdate(&out, u, r)
	// The patches of the assigned skills that change, as quiver update --diff
	// prints them.
	var diff []string
	if ask {
		for _, id := range slices.Concat(u.Added, u.Changed) {
			if assignedWhere(r, id) == nil {
				continue
			}
			patch, err := u.Diff(id)
			if err != nil {
				return tui.Preview{}, err
			}
			diff = append(diff, strings.Split(strings.TrimRight(patch, "\n"), "\n")...)
			diff = append(diff, "")
		}
	}
	b.mu.Lock()
	if b.pending == nil {
		b.pending = map[string]*library.Update{}
	}
	b.pending[name] = u
	b.mu.Unlock()
	return tui.Preview{
		Source: name, From: short(u.From), To: short(u.To), UpToDate: u.From == u.To, Ask: ask,
		Lines: strings.Split(strings.TrimRight(out.String(), "\n"), "\n"), Diff: diff,
	}, nil
}

func (b *tuiBackend) Apply(name string) (string, error) {
	b.mu.Lock()
	u := b.pending[name]
	delete(b.pending, name)
	b.mu.Unlock()
	if u == nil {
		return "", fmt.Errorf("%s has no update to apply; check it again", name)
	}
	if err := u.Apply(); err != nil {
		return "", err
	}
	syncHooks(io.Discard)
	text := fmt.Sprintf("%s updated to %s.", name, short(u.To))
	for _, e := range u.PinErrors {
		text += " warning: pin of " + e
	}
	return text, nil
}

func (b *tuiBackend) RemoveSource(name string) (string, error) {
	u, err := library.RemoveSource(name, true)
	syncHooks(io.Discard)
	if err != nil {
		return "", err
	}
	text := fmt.Sprintf("Removed source %s.", name)
	if n := len(u.Global); n > 0 {
		text += fmt.Sprintf(" Unassigned %s globally.", plural(n, "skill"))
	}
	for _, p := range u.Projects {
		text += fmt.Sprintf(" Unassigned %s from %s.", plural(len(p.Skills), "skill"), p.Name)
	}
	return text, nil
}

func (b *tuiBackend) RenameProject(id, name string) error {
	return registry.Update(func(r *registry.Registry) error { return r.RenameProject(id, name) })
}

func (b *tuiBackend) ForgetProject(id string) error {
	err := registry.Update(func(r *registry.Registry) error { return r.ForgetProject(id) })
	syncHooks(io.Discard)
	return err
}

func (b *tuiBackend) SplitHere() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	oldID, p, err := registry.SplitProject(dir)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s is now project %s, with the skills of %s.", p.Root, p.ID, oldID), nil
}

func (b *tuiBackend) AddSource(repo, name string) (string, error) {
	p, err := source.Parse(repo)
	if err != nil {
		return "", err
	}
	if name == "" {
		name = p.Name
	}
	s, err := library.StageSource(name, p.URL)
	if err != nil {
		return "", err
	}
	if len(s.Discovery.Skills) == 0 {
		_ = s.Discard()
		return "", fmt.Errorf("no skills found in %s; the source was not added", repo)
	}
	if err := s.Keep(); err != nil {
		_ = s.Discard()
		return "", err
	}
	syncHooks(io.Discard)
	text := fmt.Sprintf("Added source %s with %s. Review them before you assign them.", name, plural(len(s.Discovery.Skills), "skill"))
	for _, w := range s.Discovery.Warnings {
		text += " warning: " + w
	}
	return text, nil
}

func (b *tuiBackend) NewSkill(name, description, projectPath string, global bool) (string, error) {
	t := target{project: projectPath, global: global, yes: true}
	var out bytes.Buffer
	err := newSkill(strings.NewReader(""), &out, t, projectPath == "" && !global, description, name)
	syncHooks(io.Discard)
	if err != nil {
		return "", err
	}
	// Created, where it is served, then the warnings.
	var lines []string
	for _, l := range strings.Split(strings.TrimSpace(out.String()), "\n") {
		switch {
		case strings.HasPrefix(l, "Created skill "):
			lines = append(lines, "Created skill "+name+".")
		case strings.HasPrefix(l, "Assigned"), strings.HasPrefix(l, "Not assigned"), strings.HasPrefix(l, "warning:"), strings.HasPrefix(l, "Saved"):
			lines = append(lines, sentence(l))
		}
	}
	return strings.Join(lines, " "), nil
}

func (b *tuiBackend) Import(path string) (string, error) {
	var out bytes.Buffer
	err := importSkills(strings.NewReader(""), &out, importOptions{yes: true}, expandHome(path))
	syncHooks(io.Discard)
	if err != nil {
		return "", err
	}
	// The report lists every skill it found; the panel keeps what it did.
	var parts []string
	for _, l := range strings.Split(out.String(), "\n") {
		l = strings.TrimSpace(l)
		switch {
		case importedLine.MatchString(l), strings.HasPrefix(l, "No skills found"), strings.HasPrefix(l, "Skipped"), strings.HasPrefix(l, "warning:"), strings.HasPrefix(l, "error:"):
			parts = append(parts, sentence(l))
		}
	}
	if len(parts) == 0 {
		return "Nothing to import.", nil
	}
	return strings.Join(parts, " "), nil
}

// importedLine is the summary of an import, not the line of each skill.
var importedLine = regexp.MustCompile(`^Imported \d+ skills?\.$`)

// expandHome turns a leading ~/ into the home folder, which a shell would do
// for a path typed on a command line.
func expandHome(path string) string {
	if rest, ok := strings.CutPrefix(path, "~/"); ok {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, rest)
		}
	}
	return path
}

// sentence ends a line with a full stop, so lines joined into one status
// message stay apart.
func sentence(s string) string {
	s = strings.TrimSpace(s)
	if s == "" || strings.HasSuffix(s, ".") || strings.HasSuffix(s, ":") {
		return s
	}
	return s + "."
}

// CheckName reports at once that a name cannot be a new skill.
func (b *tuiBackend) CheckName(name string) error { return skill.ValidateName(name) }

// CheckImport reports at once that nothing is at the path to import. An empty
// path is the .claude/skills of the project, which import itself checks.
func (b *tuiBackend) CheckImport(path string) error {
	if path == "" {
		return nil
	}
	if _, err := os.Stat(expandHome(path)); err != nil {
		return fmt.Errorf("nothing to import at %s: no such file or folder", path)
	}
	return nil
}

// Delete removes a local skill like quiver delete --yes; force unassigns it.
func (b *tuiBackend) Delete(id string, force bool) (string, error) {
	u, err := library.DeleteLocal(id, force)
	syncHooks(io.Discard)
	if err != nil {
		return "", err
	}
	text := "Deleted skill " + id + "."
	var from []string
	if len(u.Global) > 0 {
		from = append(from, "every folder")
	}
	for _, p := range u.Projects {
		from = append(from, p.Name)
	}
	if len(from) > 0 {
		text += " Unassigned from " + strings.Join(from, ", ") + "."
	}
	return text, nil
}

// Pin keeps a source skill at its current commit, like quiver pin.
func (b *tuiBackend) Pin(id string) (string, error) {
	var out bytes.Buffer
	err := pin(&out, id, "")
	syncHooks(io.Discard)
	if err != nil {
		return "", err
	}
	return firstLine(out.String()), nil
}

// Unpin serves a source skill from its source again, like quiver unpin.
func (b *tuiBackend) Unpin(id string) (string, error) {
	commit, err := library.UnpinSkill(id)
	syncHooks(io.Discard)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("Unpinned %s (was at %s); it follows its source again.", id, short(commit)), nil
}

// forkError words a fork error for the panel, which asks for the name.
func forkError(err error) error {
	msg := err.Error()
	msg = strings.ReplaceAll(msg, "; choose another name with --as", "; choose another name")
	msg = strings.ReplaceAll(msg, "; choose a name with --as", "; choose a name")
	return errors.New(msg)
}

// Setup creates the Quiver home and registers Quiver in the assistants, as
// the first steps of quiver init do.
func (b *tuiBackend) Setup() (string, error) {
	var out bytes.Buffer
	if err := createHome(&out); err != nil {
		return "", err
	}
	home, _ := fsutil.QuiverHome()
	if err := mcpInstall(&out, nil); err != nil {
		return "", fmt.Errorf("the library was created in %s, but setting up the assistants failed: %w", home, err)
	}
	syncHooks(io.Discard)
	text := "Quiver is set up in " + home + "."
	if strings.Contains(out.String(), "No assistant was found") {
		text += " No assistant was found; see Health for what to add by hand."
	} else {
		text += " Start a new assistant session to use it."
	}
	return text, nil
}

// Install registers the MCP server and the hooks like quiver install.
func (b *tuiBackend) Install() (string, error) {
	var out bytes.Buffer
	err := mcpInstall(&out, nil)
	syncHooks(io.Discard)
	if err != nil {
		return "", err
	}
	if strings.Contains(out.String(), "No assistant was found") {
		return "No assistant was found, so nothing was registered. Run quiver mcp config for the entry to add by hand.", nil
	}
	return "Registered Quiver in your assistants. Start a new session to use it.", nil
}

// Uninstall removes the MCP server and the hooks like quiver uninstall.
func (b *tuiBackend) Uninstall() (string, error) {
	var out bytes.Buffer
	if err := mcpUninstall(&out, nil); err != nil {
		return "", err
	}
	return "Removed Quiver from your assistants. The library is untouched.", nil
}

// Fix applies the safe fixes of quiver doctor. Fixes that need an answer are
// left, since nothing answers them here.
func (b *tuiBackend) Fix() (string, error) {
	home, err := fsutil.QuiverHome()
	if err != nil {
		return "", err
	}
	var out bytes.Buffer
	d := &doc{in: bufio.NewReader(strings.NewReader("")), out: &out, fix: true, home: home}
	for _, check := range d.checks() {
		check()
	}
	syncHooks(io.Discard)
	fixed, failed := 0, 0
	var reasons []string
	for _, l := range strings.Split(out.String(), "\n") {
		switch t := strings.TrimSpace(l); {
		case strings.HasPrefix(t, "fixed:"):
			fixed++
		case strings.HasPrefix(t, "fix failed:"):
			failed++
			reasons = append(reasons, t)
		}
	}
	text := fmt.Sprintf("Applied %d fixes.", fixed)
	switch fixed {
	case 0:
		text = "Nothing to fix here."
	case 1:
		text = "Applied 1 fix."
	}
	if failed > 0 {
		text += " " + strings.Join(reasons, " ")
		return "", errors.New(text)
	}
	return text, nil
}

// warnings returns the "warning:" lines of a command's output, as one line.
func warnings(s string) string {
	var out []string
	for _, l := range strings.Split(s, "\n") {
		if strings.HasPrefix(l, "warning:") {
			out = append(out, l)
		}
	}
	return strings.Join(out, " ")
}

func firstLine(s string) string {
	line, _, _ := strings.Cut(strings.TrimSpace(s), "\n")
	return line
}
