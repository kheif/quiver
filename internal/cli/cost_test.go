package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kheif/quiver/internal/server"
)

func statusAt(t *testing.T, dir string, args ...string) string {
	t.Helper()
	t.Setenv(server.ProjectEnv, "")
	t.Chdir(dir)
	t.Setenv("PATH", t.TempDir())
	return mustRun(t, "", append([]string{"status"}, args...)...)
}

func TestStatusSaysWhatTheListingCosts(t *testing.T) {
	f := listEnv(t)
	out := statusAt(t, f.app)
	// acme/review (global) and tdd (project) are listed; pdf is always-on.
	wantOut(t, out, "Session start: 2 skills listed, 137 characters (about 35 tokens, an estimate); 1 always-on, loaded in full:", "0 Searchable, which add nothing to the listing.")
	mustRun(t, "", "add", "notes", "--searchable", "--project", f.app)
	out = statusAt(t, f.app)
	wantOut(t, out, "2 skills listed, 137 characters", "1 Searchable, which add nothing to the listing.")

	var st statusJSON
	if err := json.Unmarshal([]byte(statusAt(t, f.app, "--json")), &st); err != nil {
		t.Fatal(err)
	}
	if c := st.Catalog; c.Listed != 2 || c.ListedChars != 137 || c.ListedTokens != 35 || c.AlwaysOn != 1 || c.AlwaysOnChars == 0 || c.Searchable != 1 {
		t.Errorf("catalog %+v", c)
	}
	raw := statusAt(t, f.app, "--json")
	for _, key := range []string{`"listed_chars"`, `"listed_tokens_estimate"`, `"always_on_chars"`, `"always_on_tokens_estimate"`, `"searchable"`} {
		if !strings.Contains(raw, key) {
			t.Errorf("json lacks %s", key)
		}
	}
}

func TestStatusVerboseNamesTheLongestEntries(t *testing.T) {
	f := listEnv(t)
	// Fewer than ten listed skills: all of them, longest first, each with the
	// command that makes it Searchable.
	out := statusAt(t, f.app, "-v")
	wantOut(t, out, "Longest listed entries", "quiver add tdd --searchable\n", "quiver add acme/review --searchable --global\n")
	_, listing, _ := strings.Cut(out, "Longest listed entries")
	if strings.Index(listing, "  tdd ") > strings.Index(listing, "  acme/review ") {
		t.Errorf("not longest first:\n%s", out)
	}
	if out := statusAt(t, f.app); strings.Contains(out, "Longest listed entries") {
		t.Errorf("a status without -v lists entries:\n%s", out)
	}

	// More than ten: ten, the longest first.
	for i := range 13 {
		name := fmt.Sprintf("long-%02d", i)
		dir := filepath.Join(f.home, "local", name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		desc := strings.Repeat("word ", 10+i)
		if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\nname: "+name+"\ndescription: "+desc+"\n---\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		mustRun(t, "", "add", name, "--project", f.app)
	}
	out = statusAt(t, f.app, "-v")
	_, after, _ := strings.Cut(out, "Longest listed entries")
	entries := 0
	for _, l := range strings.Split(after, "\n") {
		if strings.Contains(l, " characters  quiver add ") {
			entries++
		}
	}
	if entries != 10 || !strings.Contains(after, "long-12") || strings.Contains(after, "long-00") {
		t.Errorf("%d entries:\n%s", entries, after)
	}
}

// The hint after the upgrade: skills at no level and no Searchable skill.
func TestStatusHintsThatSearchFindsOnlySearchableSkills(t *testing.T) {
	f := listEnv(t)
	out := statusAt(t, f.app)
	// acme/deploy, notes are at no level here; the source acme has the most (1,
	// against the local skill notes, which is in no source).
	wantOut(t, out, "Search finds only Searchable skills: 2 skills of the library are at no level here", "(1 of them in source acme)", "quiver add --source acme --searchable --global", "quiver add <id> --searchable")

	// Not in --json.
	if raw := statusAt(t, f.app, "--json"); strings.Contains(raw, "Search finds only") {
		t.Errorf("the hint is in the JSON:\n%s", raw)
	}

	// The first Searchable skill takes it away.
	mustRun(t, "", "add", "notes", "--searchable", "--project", f.app)
	if out := statusAt(t, f.app); strings.Contains(out, "Search finds only") {
		t.Errorf("the hint stays after a Searchable skill exists:\n%s", out)
	}

	// Nothing unassigned: no hint, even without a Searchable skill.
	mustRun(t, "", "remove", "notes", "--project", f.app)
	mustRun(t, "", "add", "notes", "acme/deploy", "--global")
	if out := statusAt(t, f.app); strings.Contains(out, "Search finds only") {
		t.Errorf("a hint with nothing unassigned:\n%s", out)
	}
}

func TestPanelCarriesTheSameCostForTheManagedProject(t *testing.T) {
	f := listEnv(t)
	t.Setenv(server.ProjectEnv, "")
	t.Chdir(f.api) // the folder of quiver is api; the panel manages app
	b := &tuiBackend{}
	d, err := b.Load("app-000001")
	if err != nil {
		t.Fatal(err)
	}
	if d.Cost.Listed != 2 || d.Cost.ListedChars != 137 || d.Cost.AlwaysOn != 1 {
		t.Errorf("cost of app %+v", d.Cost)
	}
	d, err = b.Load("")
	if err != nil {
		t.Fatal(err)
	}
	// api lists only tdd, and the global acme/review.
	if d.Cost.Listed != 2 {
		t.Errorf("cost of api %+v", d.Cost)
	}
	mustRun(t, "", "add", "notes", "--searchable", "--global")
	if d, _ = b.Load("app-000001"); d.Cost.Searchable != 1 {
		t.Errorf("searchable %+v", d.Cost)
	}
}

// A source made Searchable at once holds hundreds of skills: the listing names
// the first ten and says how many more there are, and -v names them all.
func TestSearchableListIsShortenedUnlessVerbose(t *testing.T) {
	f := listEnv(t)
	var ids []string
	for i := range 14 {
		name := fmt.Sprintf("bulk-%02d", i)
		dir := filepath.Join(f.home, "local", name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\nname: "+name+"\ndescription: Bulk skill.\n---\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		ids = append(ids, name)
	}
	out := mustRun(t, "", append(append([]string{"add"}, ids...), "--searchable", "--global")...)
	wantOut(t, out, "bulk-09", ", and 4 more (-v names them all)")
	if strings.Contains(out, "bulk-10") {
		t.Errorf("the add listing names more than ten:\n%s", out)
	}
	out = statusAt(t, f.app)
	wantOut(t, out, "Searchable in app (not listed; found by searching): ", ", and 4 more (-v names them all)")
	if strings.Contains(out, "bulk-13") {
		t.Errorf("status names more than ten:\n%s", out)
	}
	out = statusAt(t, f.app, "-v")
	wantOut(t, out, "bulk-13")
	if strings.Contains(out, "more (-v names them all)") {
		t.Errorf("-v still shortens the list:\n%s", out)
	}
	// Ten or fewer are named in full, as before.
	if got := searchableList([]string{"a", "b"}, false); got != "a, b" {
		t.Errorf("short list %q", got)
	}
}
