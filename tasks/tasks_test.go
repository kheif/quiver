// Package tasks holds the task files of the project. This test keeps the
// index in README.md and the files themselves consistent.
package tasks

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"testing"
)

var (
	// | ✅ | ↳ [TASK58.1](TASK58.1-x.md) | Title | TASK45, TASK14 |
	// A title may hold an escaped pipe (\|).
	rowRe       = regexp.MustCompile(`^\|\s*([^|]*?)\s*\|\s*(↳ )?\[(TASK[0-9.]+)\]\(([^)]+)\)\s*\|\s*((?:[^|\\]|\\.)+?)\s*\|\s*([^|]+?)\s*\|\s*$`)
	titleRe     = regexp.MustCompile(`(?m)\A# (TASK[0-9.]+)(?::| —) (.+)$`)
	statusRe    = regexp.MustCompile(`(?m)^\*\*Status:\*\* (.+)$`)
	dependsRe   = regexp.MustCompile(`(?m)^\*\*Depends on:\*\* (.+)$`)
	phaseRe     = regexp.MustCompile(`(?m)^\*\*Phase:\*\* (.+)$`)
	partOfRe    = regexp.MustCompile(`(?m)^\*\*Part of:\*\* \[(TASK[0-9.]+)\]\(([^)]+)\)`)
	subtaskRe   = regexp.MustCompile(`(?m)^- \[(TASK[0-9.]+)\]\(([^)]+)\): (.+)$`)
	linkRe      = regexp.MustCompile(`\]\(([^)#\s]+\.md)(?:#[^)]*)?\)`)
	idRe        = regexp.MustCompile(`TASK[0-9]+(?:\.[0-9]+)?`)
	statusMarks = []string{"✅", "🚧", "⏸"}
)

// forwardDependencies lists the dependencies that the index lists after the
// task that needs them. TASK55.7 was added to the subtasks of TASK55 after
// TASK56 was done, and needs it.
var forwardDependencies = map[string][]string{"TASK55.7": {"TASK56"}}

type row struct {
	status, id, file, title, depends, section string
	sub                                       bool // marked with ↳
}

type task struct {
	id, file, title, status, depends, phase, partOf, text string
}

// tree is what README.md and the task files hold.
type tree struct {
	rows   []row // in the order of the index
	byID   map[string]row
	tasks  map[string]task
	ids    []string // sorted IDs of the task files
	errors []string // problems found while reading
}

var (
	loadOnce sync.Once
	loaded   tree
)

// readText reads a file with Unix line endings, so a checkout that converted
// them (Windows) gives the same text.
func readText(name string) (string, error) {
	b, err := os.ReadFile(name)
	text := strings.ReplaceAll(string(b), "\r\n", "\n")
	// A mark such as ⏸ may be followed by the emoji variation selector.
	return strings.ReplaceAll(text, "\uFE0F", ""), err
}

// load reads README.md and every TASK*.md file in this folder, once.
func load(t *testing.T) *tree {
	t.Helper()
	loadOnce.Do(func() { loaded = read() })
	// Without rows or files every other check would pass without checking.
	if len(loaded.rows) == 0 || len(loaded.tasks) == 0 {
		t.Fatalf("found %d index rows and %d task files", len(loaded.rows), len(loaded.tasks))
	}
	return &loaded
}

func read() tree {
	tr := tree{byID: map[string]row{}, tasks: map[string]task{}}
	bad := func(format string, args ...any) { tr.errors = append(tr.errors, fmt.Sprintf(format, args...)) }

	readme, err := readText("README.md")
	if err != nil {
		bad("%v", err)
	}
	section := ""
	rowLines := 0
	for _, line := range strings.Split(readme, "\n") {
		if strings.HasPrefix(line, "|") && strings.Contains(line, "](TASK") {
			rowLines++
		}
		if strings.HasPrefix(line, "### ") {
			section = strings.TrimPrefix(line, "### ")
			continue
		}
		m := rowRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		r := row{status: m[1], sub: m[2] != "", id: m[3], file: m[4], title: m[5], depends: m[6], section: section}
		if _, dup := tr.byID[r.id]; dup {
			bad("README.md lists %s twice", r.id)
		}
		tr.byID[r.id] = r
		tr.rows = append(tr.rows, r)
	}

	if rowLines != len(tr.rows) {
		bad("README.md has %d index rows that mention a task and %d of them have the shape of a row", rowLines, len(tr.rows))
	}

	files, err := filepath.Glob("TASK*.md")
	if err != nil {
		bad("%v", err)
	}
	for _, f := range files {
		text, err := readText(f)
		if err != nil {
			bad("%v", err)
			continue
		}
		m := titleRe.FindStringSubmatch(text)
		if m == nil {
			bad("%s: the first line is not \"# TASKnn: Title\"", f)
			continue
		}
		tk := task{id: m[1], file: f, title: m[2], text: text}
		if !strings.HasPrefix(f, tk.id+"-") {
			bad("%s: the file name does not start with %s-", f, tk.id)
		}
		if m := statusRe.FindStringSubmatch(text); m != nil {
			tk.status = m[1]
		}
		if m := dependsRe.FindStringSubmatch(text); m != nil {
			tk.depends = m[1]
		}
		if m := phaseRe.FindStringSubmatch(text); m != nil {
			tk.phase = m[1]
		}
		if m := partOfRe.FindStringSubmatch(text); m != nil {
			tk.partOf = m[1]
		}
		tr.tasks[tk.id] = tk
		tr.ids = append(tr.ids, tk.id)
	}
	sort.Strings(tr.ids)
	return tr
}

// mark is the index mark a status line starts with, or empty for a task that
// is not started.
func mark(status string) string {
	for _, m := range statusMarks {
		if strings.HasPrefix(status, m) {
			return m
		}
	}
	return ""
}

func TestFilesParse(t *testing.T) {
	for _, e := range load(t).errors {
		t.Error(e)
	}
}

func TestIndexMatchesFiles(t *testing.T) {
	tr := load(t)
	for _, r := range tr.rows {
		tk, ok := tr.tasks[r.id]
		if !ok {
			t.Errorf("README.md lists %s, but no such file exists", r.id)
			continue
		}
		if r.file != tk.file {
			t.Errorf("%s: README.md links %s, the file is %s", r.id, r.file, tk.file)
		}
	}
	for _, id := range tr.ids {
		tk := tr.tasks[id]
		r, ok := tr.byID[id]
		if !ok {
			t.Errorf("%s: %s is not in the index of README.md", id, tk.file)
			continue
		}
		if title := strings.ReplaceAll(r.title, `\|`, "|"); title != tk.title {
			t.Errorf("%s: the index says %q, the file says %q", id, title, tk.title)
		}
		if got := mark(tk.status); got != r.status {
			t.Errorf("%s: the index mark is %q, the status line %q starts with %q", id, r.status, tk.status, got)
		}
		if r.status == "" && tk.status != "Todo" && !strings.HasPrefix(tk.status, "Todo ") {
			t.Errorf("%s: a task without a mark has the status line %q, not \"Todo\"", id, tk.status)
		}
		if r.status != "" && !strings.HasPrefix(tk.status, r.status+" ") {
			t.Errorf("%s: the status line %q does not start with %s and a space", id, tk.status, r.status)
		}
		if tk.depends != r.depends {
			t.Errorf("%s: the index says depends on %q, the file says %q", id, r.depends, tk.depends)
		}
		if r.sub != strings.Contains(id, ".") {
			t.Errorf("%s: the index marks a subtask with ↳, and only a subtask", id)
		}
		if tk.phase != r.section {
			t.Errorf("%s: the file says phase %q, the index lists it under %q", id, tk.phase, r.section)
		}
	}
}

func TestLinksAndReferencesResolve(t *testing.T) {
	tr := load(t)
	for _, id := range tr.ids {
		tk := tr.tasks[id]
		for _, m := range linkRe.FindAllStringSubmatch(tk.text, -1) {
			if strings.Contains(m[1], "://") {
				continue
			}
			if _, err := os.Stat(filepath.Join(filepath.Dir(tk.file), m[1])); err != nil {
				t.Errorf("%s: the link %s does not resolve", id, m[1])
			}
		}
		for _, ref := range idRe.FindAllString(tk.text, -1) {
			if _, ok := tr.tasks[ref]; !ok {
				t.Errorf("%s mentions %s, which does not exist", id, ref)
			}
		}
	}
}

func TestDependenciesAreKnownOrderedAndAcyclic(t *testing.T) {
	tr := load(t)
	position := map[string]int{}
	for i, r := range tr.rows {
		position[r.id] = i
	}
	deps := map[string][]string{}
	for _, id := range tr.ids {
		for _, d := range idRe.FindAllString(tr.tasks[id].depends, -1) {
			if _, ok := tr.tasks[d]; !ok {
				t.Errorf("%s depends on %s, which does not exist", id, d)
				continue
			}
			deps[id] = append(deps[id], d)
			// "Work through them top to bottom": a dependency comes first.
			allowed := false
			for _, f := range forwardDependencies[id] {
				allowed = allowed || f == d
			}
			if position[d] > position[id] && !allowed {
				t.Errorf("%s depends on %s, which the index lists after it", id, d)
			}
		}
	}
	const (
		visiting = 1
		done     = 2
	)
	state := map[string]int{}
	var visit func(id string, path []string)
	visit = func(id string, path []string) {
		switch state[id] {
		case done:
			return
		case visiting:
			t.Errorf("dependency cycle: %s", strings.Join(append(append([]string(nil), path...), id), " -> "))
			return
		}
		state[id] = visiting
		for _, d := range deps[id] {
			visit(d, append(append([]string(nil), path...), id))
		}
		state[id] = done
	}
	for _, id := range tr.ids {
		visit(id, nil)
	}
}

func TestSubtasksBelongToTheirParent(t *testing.T) {
	tr := load(t)
	subtasks := map[string][]string{}
	for _, id := range tr.ids {
		tk := tr.tasks[id]
		if !strings.Contains(id, ".") {
			continue
		}
		parentID := id[:strings.Index(id, ".")]
		parent, ok := tr.tasks[parentID]
		if !ok {
			t.Errorf("%s has no parent task %s", id, parentID)
			continue
		}
		subtasks[parentID] = append(subtasks[parentID], id)
		if tk.partOf != parentID {
			t.Errorf("%s: \"Part of\" names %q, the parent is %s", id, tk.partOf, parentID)
		}
		if tk.phase != parent.phase {
			t.Errorf("%s: phase %q differs from the phase of %s, %q", id, tk.phase, parentID, parent.phase)
		}
		listed := false
		for _, m := range subtaskRe.FindAllStringSubmatch(parent.text, -1) {
			if m[1] != id {
				continue
			}
			listed = true
			if m[2] != tk.file {
				t.Errorf("%s lists %s with the file %s, the file is %s", parentID, id, m[2], tk.file)
			}
			if m[3] != tk.title {
				t.Errorf("%s lists %s as %q, its title is %q", parentID, id, m[3], tk.title)
			}
		}
		if !listed {
			t.Errorf("%s does not list its subtask %s in a \"- [TASKnn](file): Title\" line", parentID, id)
		}
	}
	// A parent is not done before all its subtasks are. It may still be open
	// when they are, because its own "Done when" can ask for more.
	for _, parentID := range tr.ids {
		if tr.byID[parentID].status != "✅" {
			continue
		}
		for _, id := range subtasks[parentID] {
			if tr.byID[id].status != "✅" {
				t.Errorf("%s is done, but its subtask %s is not", parentID, id)
			}
		}
	}
	// A subtask row follows its parent or a sibling in the index.
	for i, r := range tr.rows {
		if !r.sub {
			continue
		}
		parentID := r.id[:strings.Index(r.id, ".")]
		if i == 0 {
			t.Errorf("%s: the index starts with a subtask", r.id)
			continue
		}
		prev := tr.rows[i-1].id
		if prev != parentID && !strings.HasPrefix(prev, parentID+".") {
			t.Errorf("%s: the index row follows %s, not %s or one of its subtasks", r.id, prev, parentID)
		}
	}
}
