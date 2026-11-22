package cli

import (
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kheif/quiver/internal/embed"
	"github.com/kheif/quiver/internal/embed/embedtest"
	"github.com/kheif/quiver/internal/library"
)

// setupHost starts a fake model host, makes it the home of the model "tiny",
// and chooses that model for the local engine. The engine stays words.
func setupHost(t *testing.T) *embedtest.Host {
	t.Helper()
	h := embedtest.NewHost(t, 1)
	t.Cleanup(library.RegisterModel(library.ModelSpec{
		Name: "tiny", Config: embed.Config(embedtest.Tiny), MinScore: -1, BaseURL: h.URL,
		Size: int64(len(h.Weights)), SHA256: embedtest.Hash(h.Weights),
		TokenizerSize: int64(len(h.Tokenizer)), TokenizerSHA256: embedtest.Hash(h.Tokenizer),
	}))
	t.Cleanup(library.SetModelTransport(h.Transport()))
	writeSearchFile(t, `{"version": 1, "engine": "words", "local": {"model": "tiny", "minScore": -1}}`)
	return h
}

func asTerminal(t *testing.T) {
	t.Helper()
	old := interactive
	interactive = func(io.Reader) bool { return true }
	t.Cleanup(func() { interactive = old })
}

func modelFile(t *testing.T) string {
	t.Helper()
	dir, err := library.ModelDir("tiny")
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Join(dir, "model.safetensors")
}

func TestEngineSetupEndToEnd(t *testing.T) {
	searchableEnv(t)
	h := setupHost(t)
	out, errOut, err := runSplit(t, "engine", "setup", "--yes")
	if err != nil {
		t.Fatalf("%v\n%s\n%s", err, out, errOut)
	}
	wantOut(t, out, "Search by meaning is on.", "found at place", "quiver engine use words", "quiver engine remove",
		"the only thing that leaves this", "a search itself makes no network call")
	wantOut(t, errOut, "Downloading model.safetensors: 100%", "Indexing skills for the search:")
	if cur := library.LoadEngine(); cur.Engine != library.EngineLocal || cur.Local.Model != "tiny" {
		t.Errorf("settings %+v", cur)
	}
	if got := library.Installed("tiny"); len(got) != 2 || got["model.safetensors"] != embedtest.Hash(h.Weights) {
		t.Errorf("installed %v", got)
	}
	if !exists(filepath.Join(os.Getenv("QUIVER_HOME"), "cache", "embeddings.json")) {
		t.Error("no index")
	}
	if h.Requests() != 2 || len(h.Stray()) != 0 {
		t.Errorf("%d requests, stray dials %v", h.Requests(), h.Stray())
	}
	// A search makes no network call: the host stays at two requests, and a
	// dial of any address would have failed the transport.
	if out, _, err := runSplit(t, "search", "deploy"); err != nil || !strings.Contains(out, "acme/deploy") {
		t.Errorf("search: %q, %v", out, err)
	}
	if h.Requests() != 2 || len(h.Stray()) != 0 {
		t.Errorf("a search used the network: %d requests, %v", h.Requests(), h.Stray())
	}
}

func TestEngineSetupAsksOnceAndDefaultsToNo(t *testing.T) {
	searchableEnv(t)
	h := setupHost(t)
	asTerminal(t)
	for _, answer := range []string{"\n", "n\n", ""} {
		out := mustRun(t, answer, "engine", "setup")
		wantOut(t, out, "Download the model and switch to search by meaning? [y/N]", "Nothing changed.", "91 MB"[:0]+"tiny (English only)", "about 85 MB")
		if strings.Count(out, "[y/N]") != 1 {
			t.Errorf("the question was asked more than once: %q", out)
		}
	}
	if h.Requests() != 0 || library.LoadEngine().Engine != library.EngineWords || exists(modelFile(t)) {
		t.Error("something changed without a yes")
	}
	out := mustRun(t, "y\n", "engine", "setup")
	wantOut(t, out, "Search by meaning is on.")
	if strings.Count(out, "[y/N]") != 1 {
		t.Errorf("asked again: %q", out)
	}
}

func TestEngineSetupWithoutATerminalChangesNothing(t *testing.T) {
	searchableEnv(t)
	h := setupHost(t)
	out := mustRun(t, "y\n", "engine", "setup")
	wantOut(t, out, "Nothing changed", "quiver engine setup", "--yes")
	if h.Requests() != 0 || exists(modelFile(t)) || library.LoadEngine().Engine != library.EngineWords {
		t.Error("something changed")
	}
}

func TestEngineSetupRejectsABadDownload(t *testing.T) {
	searchableEnv(t)
	h := setupHost(t)
	h.Handler = func(w http.ResponseWriter, r *http.Request) bool {
		_, _ = w.Write([]byte("a different file"))
		return true
	}
	_, err := runIn(t, "", "engine", "setup", "--yes")
	if err == nil || !strings.Contains(err.Error(), "step 1") || !strings.Contains(err.Error(), "engine stays words") {
		t.Fatalf("err = %v", err)
	}
	if exists(modelFile(t)) || library.LoadEngine().Engine != library.EngineWords {
		t.Error("the bad file was kept or the engine changed")
	}
	// Searching keeps working, as words.
	if out, _, err := runSplit(t, "search", "deploy"); err != nil || !strings.Contains(out, "acme/deploy") {
		t.Errorf("search: %q, %v", out, err)
	}
}

func TestEngineSetupAcceptsAFileByHand(t *testing.T) {
	searchableEnv(t)
	h := setupHost(t)
	dir := filepath.Dir(modelFile(t))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	embedtest.Write(t, dir, "model.safetensors", h.Weights)
	embedtest.Write(t, dir, "tokenizer.json", h.Tokenizer)
	out := mustRun(t, "", "engine", "setup", "--yes")
	wantOut(t, out, "already there with the pinned SHA-256", "Search by meaning is on.")
	if h.Requests() != 0 {
		t.Error("a file that was there was downloaded")
	}
	if got := library.Installed("tiny"); len(got) != 0 {
		t.Errorf("a file of the user is recorded as downloaded: %v", got)
	}
	// A file with another hash is refused with the pinned hash.
	embedtest.Write(t, dir, "model.safetensors", []byte("other"))
	_, err := runIn(t, "", "engine", "setup", "--yes")
	if err == nil || !strings.Contains(err.Error(), embedtest.Hash(h.Weights)) {
		t.Errorf("err = %v", err)
	}
}

func TestEngineSetupCheckThatFailsLeavesWords(t *testing.T) {
	searchableEnv(t)
	setupHost(t)
	old := checkVectorPath
	checkVectorPath = func(*library.Library, string) (string, error) {
		return "", errors.New("the vector search found nothing")
	}
	t.Cleanup(func() { checkVectorPath = old })
	_, err := runIn(t, "", "engine", "setup", "--yes")
	if err == nil || !strings.Contains(err.Error(), "step 3, the check") {
		t.Fatalf("err = %v", err)
	}
	if library.LoadEngine().Engine != library.EngineWords {
		t.Error("the engine changed")
	}
	// What was downloaded is recorded, so remove can delete it.
	if len(library.Installed("tiny")) != 2 {
		t.Error("the download is not recorded")
	}
	// Words still finds the skill.
	if out, _, err := runSplit(t, "search", "deploy"); err != nil || !strings.Contains(out, "acme/deploy") {
		t.Errorf("search: %q, %v", out, err)
	}
}

func TestEngineRemoveDeletesOnlyWhatQuiverDownloaded(t *testing.T) {
	searchableEnv(t)
	h := setupHost(t)
	mustRun(t, "", "engine", "setup", "--yes")
	asTerminal(t)

	// No: the index goes, the model stays.
	out := mustRun(t, "n\n", "engine", "remove")
	wantOut(t, out, "Deleted the search index.", "Search engine: words", "Delete the model file that Quiver downloaded? [y/N]", "The model file stays.")
	if !exists(modelFile(t)) || exists(filepath.Join(os.Getenv("QUIVER_HOME"), "cache", "embeddings.json")) {
		t.Error("remove deleted the wrong thing")
	}
	if library.LoadEngine().Engine != library.EngineWords {
		t.Error("the engine is not words")
	}
	// Yes: the model goes; a file the user replaced stays.
	dir := filepath.Dir(modelFile(t))
	embedtest.Write(t, dir, "tokenizer.json", []byte("mine"))
	out = mustRun(t, "y\n", "engine", "remove")
	wantOut(t, out, "model.safetensors")
	if exists(modelFile(t)) || !exists(filepath.Join(dir, "tokenizer.json")) {
		t.Error("remove deleted the wrong file")
	}
	if len(library.Installed("tiny")) != 0 {
		t.Error("the record stays")
	}
	_ = h
}

func TestEngineRemoveNeverDeletesAFileByHand(t *testing.T) {
	searchableEnv(t)
	h := setupHost(t)
	dir := filepath.Dir(modelFile(t))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	embedtest.Write(t, dir, "model.safetensors", h.Weights)
	embedtest.Write(t, dir, "tokenizer.json", h.Tokenizer)
	mustRun(t, "", "engine", "setup", "--yes")
	out := mustRun(t, "", "engine", "remove", "--yes")
	wantOut(t, out, "downloaded no model file")
	if !exists(modelFile(t)) {
		t.Error("a file of the user was deleted")
	}
}

func TestEngineRemoveWithoutATerminalKeepsTheModel(t *testing.T) {
	searchableEnv(t)
	setupHost(t)
	mustRun(t, "", "engine", "setup", "--yes")
	out := mustRun(t, "y\n", "engine", "remove")
	wantOut(t, out, "The model file stays", "--yes")
	if !exists(modelFile(t)) {
		t.Error("the model was deleted")
	}
}

func TestEngineUseWordsKeepsEverything(t *testing.T) {
	searchableEnv(t)
	setupHost(t)
	mustRun(t, "", "engine", "setup", "--yes")
	mustRun(t, "", "engine", "use", "words")
	if !exists(modelFile(t)) || !exists(filepath.Join(os.Getenv("QUIVER_HOME"), "cache", "embeddings.json")) || len(library.Installed("tiny")) != 2 {
		t.Error("use words deleted something")
	}
}

func TestOfferAfterTheFirstSearchableAssignment(t *testing.T) {
	f := searchableEnv(t)
	h := setupHost(t)
	asTerminal(t)

	// The first assignment offers once; the default is no, and it is recorded.
	out := mustRun(t, "\n", "add", "pdf", "--searchable", "--global")
	wantOut(t, out, "Set up search by meaning now?", "[y/N]", "quiver engine setup")
	if !library.LoadEngine().Offered || h.Requests() != 0 || library.LoadEngine().Engine != library.EngineWords {
		t.Errorf("after no: %+v, %d requests", library.LoadEngine(), h.Requests())
	}
	// Never again.
	out = mustRun(t, "y\n", "add", "tdd", "--searchable", "--global")
	if strings.Contains(out, "Set up search by meaning") || h.Requests() != 0 {
		t.Errorf("the offer was repeated: %s", out)
	}
	_ = f
}

func TestOfferWithAYesSetsUp(t *testing.T) {
	searchableEnv(t)
	setupHost(t)
	asTerminal(t)
	out := mustRun(t, "y\n", "add", "pdf", "--searchable", "--global")
	wantOut(t, out, "Search by meaning is on.")
	if library.LoadEngine().Engine != library.EngineLocal {
		t.Error("the engine is not local")
	}
}

func TestNoOfferWithoutATerminalOrUnderLocal(t *testing.T) {
	searchableEnv(t)
	setupHost(t)
	out := mustRun(t, "y\n", "add", "pdf", "--searchable", "--global")
	if strings.Contains(out, "Set up search") || library.LoadEngine().Offered {
		t.Errorf("offered without a terminal: %s", out)
	}
	asTerminal(t)
	writeSearchFile(t, `{"version": 1, "engine": "local", "local": {"model": "tiny", "minScore": -1}}`)
	out = mustRun(t, "n\n", "add", "tdd", "--searchable", "--global")
	if strings.Contains(out, "Set up search") {
		t.Errorf("offered under the local engine: %s", out)
	}
}

func TestStatusAdvisesTheSetupWhileSearchableSkillsExist(t *testing.T) {
	searchableEnv(t)
	h := setupHost(t)
	out := mustRun(t, "", "status")
	wantOut(t, out, "search by meaning: quiver engine setup")
	if h.Requests() != 0 || library.LoadEngine().Offered {
		t.Error("status started something")
	}
	writeSearchFile(t, `{"version": 1, "engine": "local", "local": {"model": "tiny"}}`)
	if out := mustRun(t, "", "status"); strings.Contains(out, "search by meaning") {
		t.Errorf("advice under local: %s", out)
	}
}

func TestDoctorReportsAndRepairsTheChain(t *testing.T) {
	searchableEnv(t)
	h := setupHost(t)
	mustRun(t, "", "engine", "setup", "--yes")
	if err := os.Remove(modelFile(t)); err != nil {
		t.Fatal(err)
	}

	out, _ := runIn(t, "", "doctor")
	wantOut(t, out, "warn   search engine: local, model tiny", "is missing", "Repair: quiver engine setup")

	// --fix without a terminal downloads nothing.
	before := h.Requests()
	out, _ = runIn(t, "y\n", "doctor", "--fix")
	if h.Requests() != before || exists(modelFile(t)) {
		t.Errorf("a fix without a terminal downloaded: %s", out)
	}
	// The panel's Fix downloads nothing either, and points to the command.
	if _, err := (&tuiBackend{}).Fix(); err != nil {
		t.Fatal(err)
	}
	if h.Requests() != before || exists(modelFile(t)) {
		t.Error("the panel's Fix downloaded")
	}
	var found bool
	r, _ := (&tuiBackend{}).Health()
	for _, it := range r.Items {
		if strings.HasPrefix(it.Title, "search engine: local") && it.Level == "warn" && strings.Contains(strings.Join(it.Details, " "), "quiver engine setup") {
			found = true
		}
	}
	if !found {
		t.Errorf("the panel does not point to the setup: %+v", r.Items)
	}

	// --fix in a terminal repeats the step with the same question.
	asTerminal(t)
	out, _ = runIn(t, "y\n", "doctor", "--fix")
	wantOut(t, out, "Download the model and switch to search by meaning? [y/N]", "fixed:")
	if !exists(modelFile(t)) || h.Requests() != before+1 {
		t.Errorf("the file was not repaired: %d requests\n%s", h.Requests(), out)
	}
	out, _ = runIn(t, "", "doctor")
	wantOut(t, out, "ok     search engine: local, model tiny")
}

func TestDoctorRebuildsAStaleIndex(t *testing.T) {
	searchableEnv(t)
	setupHost(t)
	mustRun(t, "", "engine", "setup", "--yes")
	if err := os.Remove(filepath.Join(os.Getenv("QUIVER_HOME"), "cache", "embeddings.json")); err != nil {
		t.Fatal(err)
	}
	asTerminal(t)
	out, _ := runIn(t, "", "doctor", "--fix")
	wantOut(t, out, "rebuilt the index")
	out, _ = runIn(t, "", "doctor")
	wantOut(t, out, "ok     search engine: local, model tiny")
}

func TestSetupRunsNoCommand(t *testing.T) {
	// The setup is a download and the model runs in the process: the source
	// of the setup starts no command.
	for _, name := range []string{"enginesetup.go", "../library/modelsetup.go"} {
		b, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		for _, banned := range []string{"os/exec", "exec.Command"} {
			if strings.Contains(string(b), banned) {
				t.Errorf("%s uses %s", name, banned)
			}
		}
	}
}
