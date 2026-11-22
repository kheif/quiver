package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/kheif/quiver/internal/embed"
	"github.com/kheif/quiver/internal/embed/embedtest"
	"github.com/kheif/quiver/internal/library"
)

// runSplit runs quiver and returns stdout and stderr apart.
func runSplit(t *testing.T, args ...string) (out, errOut string, err error) {
	t.Helper()
	var o, e bytes.Buffer
	cmd := newRootCmd()
	cmd.SetOut(&o)
	cmd.SetErr(&e)
	cmd.SetIn(strings.NewReader(""))
	cmd.SetArgs(args)
	err = cmd.Execute()
	return o.String(), e.String(), err
}

func searchFile(t *testing.T) string {
	t.Helper()
	return filepath.Join(os.Getenv("QUIVER_HOME"), "state", "search.json")
}

func writeSearchFile(t *testing.T, content string) {
	t.Helper()
	path := searchFile(t)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

const newerSearchFile = `{"version": 2, "engine": "local"}`
const newerSearchMessage = "search.json has version 2, but this quiver supports only up to 1: update quiver"

func TestEngineShowsWordsByDefault(t *testing.T) {
	searchableEnv(t)
	if out := mustRun(t, "", "engine"); out != "Search engine: words\n" {
		t.Errorf("out %q", out)
	}
	var got statusSearch
	if err := json.Unmarshal([]byte(mustRun(t, "", "engine", "--json")), &got); err != nil || got != (statusSearch{Engine: "words"}) {
		t.Errorf("json %+v, %v", got, err)
	}
}

func TestEngineUse(t *testing.T) {
	searchableEnv(t)
	if out := mustRun(t, "", "engine", "use", "words"); out != "Search engine: words\n" {
		t.Errorf("out %q", out)
	}
	data, err := os.ReadFile(searchFile(t))
	if err != nil || !strings.Contains(string(data), `"engine": "words"`) || !strings.Contains(string(data), `"version": 1`) {
		t.Errorf("file %s, %v", data, err)
	}
	if out := mustRun(t, "", "engine"); out != "Search engine: words\n" {
		t.Errorf("out %q", out)
	}
}

func TestEngineUseUnknownWritesNothing(t *testing.T) {
	searchableEnv(t)
	_, err := runIn(t, "", "engine", "use", "telepathy")
	if err == nil || !strings.Contains(err.Error(), `unknown search engine "telepathy": use local or words`) {
		t.Errorf("err = %v", err)
	}
	if _, err := os.Stat(searchFile(t)); !os.IsNotExist(err) {
		t.Errorf("a file was written: %v", err)
	}
	if _, err := runIn(t, "", "engine", "use"); err == nil {
		t.Error("use without an engine did not fail")
	}
	if _, err := runIn(t, "", "engine", "extra"); err == nil {
		t.Error("an argument that is not a subcommand did not fail")
	}
}

func TestEngineShowsAProblemInTheFile(t *testing.T) {
	searchableEnv(t)
	writeSearchFile(t, `{"version": 1, "engine": "telepathy"}`)
	wantOut(t, mustRun(t, "", "engine"), "Search engine: words", `warning: search.json: unknown engine "telepathy", searching with words`)

	writeSearchFile(t, newerSearchFile)
	wantOut(t, mustRun(t, "", "engine"), "Search engine: words", "warning: "+newerSearchMessage)
	_, err := runIn(t, "", "engine", "use", "words")
	if err == nil || err.Error() != newerSearchMessage {
		t.Errorf("use: %v", err)
	}
	if data, _ := os.ReadFile(searchFile(t)); string(data) != newerSearchFile {
		t.Errorf("the file was rewritten: %s", data)
	}
}

func TestSearchEngineFlag(t *testing.T) {
	searchableEnv(t)
	wantOut(t, mustRun(t, "", "search", "--engine", "words", "deploy"), "acme/deploy")
	_, err := runIn(t, "", "search", "--engine", "telepathy", "deploy")
	if err == nil || !strings.Contains(err.Error(), `unknown search engine "telepathy"`) {
		t.Errorf("err = %v", err)
	}
}

func TestSearchStillNeedsText(t *testing.T) {
	searchableEnv(t)
	for _, args := range [][]string{{"search"}, {"search", "--engine", "words"}, {"search", "  "}} {
		if _, err := runIn(t, "", args...); err == nil {
			t.Errorf("%v did not fail", args)
		}
	}
}

// A settings file that cannot be used is said once on stderr, and the search
// runs as words.
func TestSearchWithAProblemInTheFile(t *testing.T) {
	searchableEnv(t)
	writeSearchFile(t, newerSearchFile)
	out, errOut, err := runSplit(t, "search", "deploy")
	if err != nil || !strings.Contains(out, "acme/deploy") {
		t.Fatalf("out %q, err %v", out, err)
	}
	if strings.Count(errOut, newerSearchMessage) != 1 || strings.Contains(out, "update quiver") {
		t.Errorf("stderr %q, stdout %q", errOut, out)
	}
	// --engine does not read the file.
	if _, errOut, _ := runSplit(t, "search", "--engine", "words", "deploy"); errOut != "" {
		t.Errorf("stderr %q", errOut)
	}
}

func TestStatusAndDoctorShowTheEngine(t *testing.T) {
	searchableEnv(t)
	wantOut(t, mustRun(t, "", "status"), "Search engine: words")
	out, _ := runIn(t, "", "doctor")
	wantOut(t, out, "ok     search engine: words")

	writeSearchFile(t, newerSearchFile)
	out = mustRun(t, "", "status")
	wantOut(t, out, "Search engine: words", "warning: "+newerSearchMessage)
	var st statusJSON
	if err := json.Unmarshal([]byte(mustRun(t, "", "status", "--json")), &st); err != nil || st.Search.Engine != "words" || !strings.Contains(st.Search.Problem, "update quiver") {
		t.Errorf("status json %+v, %v", st.Search, err)
	}
	out, _ = runIn(t, "", "doctor")
	wantOut(t, out, "warn   search engine: words", newerSearchMessage)
}

// The name of a command or an alias is used once, so "quiver engine" never
// takes the place of another command.
func TestCommandNamesAreUnique(t *testing.T) {
	seen := map[string]string{}
	for _, c := range newRootCmd().Commands() {
		for _, n := range append([]string{c.Name()}, c.Aliases...) {
			if other, dup := seen[n]; dup {
				t.Errorf("%q is a name of %s and %s", n, other, c.Name())
			}
			seen[n] = c.Name()
		}
	}
	if seen["engine"] != "engine" {
		t.Error("no engine command")
	}
}

// The panel searches the whole library: a skill that is assigned nowhere and
// one that is Searchable are found, and nothing is capped.
func TestTUIBackendSearchesTheWholeLibrary(t *testing.T) {
	searchableEnv(t)
	b := &tuiBackend{}
	if _, err := b.Load(""); err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{"deploy", "notes", "pdf"} {
		if ids, err := b.SearchWords(q); err != nil || len(ids) == 0 {
			t.Errorf("%s: %v, %v", q, ids, err)
		}
		if ids, err := b.Search(t.Context(), q); err != nil || len(ids) == 0 {
			t.Errorf("%s with the engine: %v, %v", q, ids, err)
		}
	}
}

// tinyModel installs the tiny model of embedtest as the model "tiny" of the
// local engine and chooses it.
func tinyModel(t *testing.T) {
	t.Helper()
	dir, err := library.ModelDir("tiny")
	if err != nil {
		t.Fatal(err)
	}
	embedtest.WriteModel(t, dir, 1)
	st, err := os.Stat(filepath.Join(dir, "model.safetensors"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(library.RegisterModel(library.ModelSpec{Name: "tiny", Config: embed.Config(embedtest.Tiny), Size: st.Size(), MinScore: -1}))
	writeSearchFile(t, `{"version": 1, "engine": "local", "local": {"model": "tiny", "minScore": -1}}`)
}

// A search that finds the index cold builds it and says so on stderr; the
// next one does not.
func TestSearchBuildsAColdIndexWithAProgressLine(t *testing.T) {
	searchableEnv(t)
	tinyModel(t)
	out, errOut, err := runSplit(t, "search", "deploy")
	if err != nil || !strings.Contains(out, "acme/deploy") {
		t.Fatalf("out %q, err %v", out, err)
	}
	if !regexp.MustCompile(`Indexing skills for the search: \d+ of \d+\n`).MatchString(errOut) {
		t.Errorf("stderr %q", errOut)
	}
	if strings.Contains(out, "Indexing") {
		t.Errorf("progress on stdout: %q", out)
	}
	if _, errOut, _ := runSplit(t, "search", "deploy"); errOut != "" {
		t.Errorf("the second search said %q", errOut)
	}
}

func TestEngineReindex(t *testing.T) {
	searchableEnv(t)
	tinyModel(t)
	out, errOut, err := runSplit(t, "engine", "reindex")
	if err != nil || !regexp.MustCompile(`^Indexed \d+ skills with tiny\.\n$`).MatchString(out) {
		t.Fatalf("out %q, err %v", out, err)
	}
	if !strings.Contains(errOut, "Indexing skills for the search:") {
		t.Errorf("stderr %q", errOut)
	}
	if _, err := os.Stat(filepath.Join(os.Getenv("QUIVER_HOME"), "cache", "embeddings.json")); err != nil {
		t.Error(err)
	}
	if _, _, err := runSplit(t, "engine", "reindex", "extra"); err == nil {
		t.Error("an argument was accepted")
	}
}

func TestEngineReindexWithoutTheModelFile(t *testing.T) {
	searchableEnv(t)
	writeSearchFile(t, `{"version": 1, "engine": "local"}`)
	_, err := runIn(t, "", "engine", "reindex")
	if err == nil || !strings.Contains(err.Error(), "model file") || !strings.Contains(err.Error(), "is missing") {
		t.Errorf("err = %v", err)
	}
	// A search still works, as words, with one line.
	out, errOut, err := runSplit(t, "search", "deploy")
	if err != nil || !strings.Contains(out, "acme/deploy") || strings.Count(errOut, "searching with words") != 1 {
		t.Errorf("out %q, stderr %q, err %v", out, errOut, err)
	}
}

func TestDoctorChecksTheLocalEngine(t *testing.T) {
	searchableEnv(t)
	writeSearchFile(t, `{"version": 1, "engine": "local"}`)
	out, _ := runIn(t, "", "doctor")
	wantOut(t, out, "warn   search engine: local, model all-minilm-l6-v2", "is missing", "Search runs as words")

	tinyModel(t)
	out, _ = runIn(t, "", "doctor")
	wantOut(t, out, "warn   search engine: local, model tiny", "model file intact, 8 values per vector, 0 of 5 skills indexed", "5 skills are stale")
	if _, _, err := runSplit(t, "engine", "reindex"); err != nil {
		t.Fatal(err)
	}
	out, _ = runIn(t, "", "doctor")
	wantOut(t, out, "ok     search engine: local, model tiny", "model file intact, 8 values per vector, 5 of 5 skills indexed")

	writeSearchFile(t, `{"version": 1, "engine": "local", "local": {"model": "nope"}}`)
	out, _ = runIn(t, "", "doctor")
	wantOut(t, out, `unknown model "nope"`)
}
