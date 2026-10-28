package server

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/kheif/quiver/internal/config"
	"github.com/kheif/quiver/internal/library"
	"github.com/kheif/quiver/internal/registry"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// syncBuffer is a log buffer the watcher goroutine can write while the test
// reads it.
type syncBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.b.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.b.String()
}

// live starts a watched server for w and a client that reports
// prompts/list_changed on the returned channel.
func live(t *testing.T, w Where) (*mcp.ClientSession, <-chan struct{}, *syncBuffer) {
	t.Helper()
	old := debounce
	debounce = 20 * time.Millisecond
	t.Cleanup(func() { debounce = old })
	logs := &syncBuffer{}
	log := slog.New(slog.NewTextHandler(logs, nil))
	s, sess, p := newServer(Options{Version: "dev", Log: log, Where: w})
	if err := watch(t.Context(), s, sess, p, log); err != nil {
		t.Fatal(err)
	}
	changed := make(chan struct{}, 16)
	cs := connectWith(t, s, &mcp.ClientOptions{
		PromptListChangedHandler: func(context.Context, *mcp.PromptListChangedRequest) { changed <- struct{}{} },
	})
	return cs, changed, logs
}

func waitChange(t *testing.T, changed <-chan struct{}) {
	t.Helper()
	select {
	case <-changed:
	case <-time.After(5 * time.Second):
		t.Fatal("no prompts/list_changed notification")
	}
}

// prompts returns the prompt names and descriptions the client sees.
func promptList(t *testing.T, cs *mcp.ClientSession) map[string]string {
	t.Helper()
	res, err := cs.ListPrompts(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]string{}
	for _, p := range res.Prompts {
		out[p.Name] = p.Description
	}
	return out
}

func TestLiveAssign(t *testing.T) {
	w, _ := createEnv(t)
	cs, changed, _ := live(t, w)
	// What "quiver add draft" does, from another goroutine.
	go func() {
		_ = registry.Update(func(r *registry.Registry) error { return r.Assign(w.ProjectID(), "draft") })
	}()
	waitChange(t, changed)
	if _, ok := promptList(t, cs)["draft"]; !ok {
		t.Errorf("prompts = %v", promptList(t, cs))
	}
	if got, _ := call(t, cs, "list_skills", nil); !strings.Contains(got, `"id":"draft"`) {
		t.Errorf("list_skills: %s", got)
	}
}

func TestLiveNewSkillAndRemove(t *testing.T) {
	w, _ := createEnv(t)
	cs, changed, _ := live(t, w)
	if _, err := library.CreateLocal("fresh", "A new skill.", "Body.", nil, library.ScopeGlobal, ""); err != nil {
		t.Fatal(err)
	}
	waitChange(t, changed)
	if got := promptList(t, cs)["fresh"]; got != "A new skill." {
		t.Errorf("fresh = %q", got)
	}
	err := registry.Update(func(r *registry.Registry) error { r.UnassignGlobal("pdf"); return nil })
	if err != nil {
		t.Fatal(err)
	}
	waitChange(t, changed)
	got := promptList(t, cs)
	if _, ok := got["pdf"]; ok {
		t.Errorf("pdf still listed: %v", got)
	}
	names := make([]string, 0, len(got))
	for n := range got {
		names = append(names, n)
	}
	slices.Sort(names)
	if want := []string{"db-migration", "fresh"}; !slices.Equal(names, want) {
		t.Errorf("prompts = %v, want %v", names, want)
	}
}

func TestLiveDescriptionChange(t *testing.T) {
	w, local := createEnv(t)
	cs, changed, _ := live(t, w)
	writeFiles(t, local, map[string]string{"pdf/SKILL.md": "---\nname: pdf\ndescription: Only fill PDF forms.\n---\nBody.\n"})
	waitChange(t, changed)
	if got := promptList(t, cs)["pdf"]; got != "Only fill PDF forms." {
		t.Errorf("pdf = %q", got)
	}
}

// A pull changes files deep in the source, which is not watched, and the
// source's .git folder, which is.
func TestLiveSourcePull(t *testing.T) {
	w, _ := createEnv(t)
	home := os.Getenv("QUIVER_HOME")
	cfg := config.Default()
	cfg.Sources["acme"] = config.Source{Repo: "https://example.com/acme.git"}
	if err := config.Save(cfg); err != nil {
		t.Fatal(err)
	}
	repo := filepath.Join(home, "sources", "acme")
	writeFiles(t, repo, map[string]string{
		"skills/review/SKILL.md": "---\nname: review\ndescription: Review code.\n---\n",
		".git/HEAD":              "ref: refs/heads/main\n",
	})
	if err := registry.Update(func(r *registry.Registry) error { r.AssignGlobal("acme/review"); return nil }); err != nil {
		t.Fatal(err)
	}
	cs, changed, _ := live(t, w)
	writeFiles(t, repo, map[string]string{
		"skills/review/SKILL.md": "---\nname: review\ndescription: Review code and tests.\n---\n",
		".git/ORIG_HEAD":         "abc\n",
	})
	waitChange(t, changed)
	if got := promptList(t, cs)["acme/review"]; got != "Review code and tests." {
		t.Errorf("acme/review = %q", got)
	}
}

func TestLiveBadRegistryKeepsState(t *testing.T) {
	w, _ := createEnv(t)
	cs, changed, logs := live(t, w)
	path, _ := registry.Path()
	good, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("{broken"), 0o644); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for !strings.Contains(logs.String(), "reload failed; keeping the last skill list") {
		if time.Now().After(deadline) {
			t.Fatalf("no reload error logged:\n%s", logs.String())
		}
		time.Sleep(10 * time.Millisecond)
	}
	if got := promptList(t, cs); len(got) != 2 {
		t.Errorf("prompts after a bad reload = %v", got)
	}
	// The watcher keeps running.
	if err := os.WriteFile(path, good, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := registry.Update(func(r *registry.Registry) error { return r.Assign(w.ProjectID(), "draft") }); err != nil {
		t.Fatal(err)
	}
	waitChange(t, changed)
	if _, ok := promptList(t, cs)["draft"]; !ok {
		t.Error("draft not listed after the registry was fixed")
	}
}

func TestLiveIgnoresHidden(t *testing.T) {
	w, local := createEnv(t)
	_, changed, logs := live(t, w)
	home := os.Getenv("QUIVER_HOME")
	writeFiles(t, home, map[string]string{".lock": "", "notes.txt": "x"})
	writeFiles(t, local, map[string]string{".new-x-123/SKILL.md": "---\nname: x\ndescription: d\n---\n"})
	time.Sleep(200 * time.Millisecond)
	select {
	case <-changed:
		t.Error("notification for an ignored change")
	default:
	}
	if strings.Contains(logs.String(), "skill list changed") {
		t.Errorf("reloaded:\n%s", logs.String())
	}
}

// A server started in a folder that is not a project serves the project
// once "quiver add" registers the folder.
func TestLiveFolderRegisteredAfterStart(t *testing.T) {
	createEnv(t)
	dir := filepath.Join(t.TempDir(), "web")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	cs, changed, _ := live(t, Where{Dir: dir})
	// What "quiver add draft" does in that folder.
	go func() {
		p, _, _, err := registry.RegisterProject(dir)
		if err == nil {
			_ = registry.Update(func(r *registry.Registry) error { return r.Assign(p.ID, "draft") })
		}
	}()
	waitChange(t, changed)
	if _, ok := promptList(t, cs)["draft"]; !ok {
		t.Errorf("prompts = %v", promptList(t, cs))
	}
	if got, _ := call(t, cs, "list_skills", nil); !strings.Contains(got, `"id":"draft"`) {
		t.Errorf("list_skills: %s", got)
	}
}
