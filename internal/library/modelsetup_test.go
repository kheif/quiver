package library

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/kheif/quiver/internal/embed"
	"github.com/kheif/quiver/internal/embed/embedtest"
)

// hostEnv starts a fake model host, registers the model "tiny" with its
// pinned sizes and hashes, and makes downloads use the host's transport.
func hostEnv(t *testing.T) (*embedtest.Host, ModelSpec) {
	t.Helper()
	t.Setenv("QUIVER_HOME", t.TempDir())
	h := embedtest.NewHost(t, 1)
	spec := ModelSpec{
		Name: "tiny", Config: embed.Config(embedtest.Tiny), MinScore: -1, BaseURL: h.URL,
		Size: int64(len(h.Weights)), SHA256: embedtest.Hash(h.Weights),
		TokenizerSize: int64(len(h.Tokenizer)), TokenizerSHA256: embedtest.Hash(h.Tokenizer),
	}
	t.Cleanup(RegisterModel(spec))
	t.Cleanup(SetModelTransport(h.Transport()))
	return h, spec
}

func noPartFiles(t *testing.T, dir string) {
	t.Helper()
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if strings.Contains(e.Name(), ".part-") {
			t.Errorf("temporary file left: %s", e.Name())
		}
	}
}

func TestEnsureModelFilesDownloadsAndChecks(t *testing.T) {
	h, spec := hostEnv(t)
	var last = map[string]int64{}
	res, err := spec.EnsureModelFiles(context.Background(), func(file string, done, total int64) { last[file] = done })
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Downloaded) != 2 || res.Downloaded["model.safetensors"] != spec.SHA256 || res.Downloaded["tokenizer.json"] != spec.TokenizerSHA256 {
		t.Errorf("downloaded %v", res.Downloaded)
	}
	if last["model.safetensors"] != spec.Size {
		t.Errorf("progress %v", last)
	}
	if err := spec.VerifyModelFile(); err != nil {
		t.Error(err)
	}
	dir, _ := ModelDir("tiny")
	noPartFiles(t, dir)
	if got := h.Requests(); got != 2 {
		t.Errorf("%d requests", got)
	}
	// Again: nothing is downloaded.
	res, err = spec.EnsureModelFiles(context.Background(), nil)
	if err != nil || len(res.Downloaded) != 0 || len(res.Present) != 2 || h.Requests() != 2 {
		t.Errorf("second run: %+v, %v, %d requests", res, err, h.Requests())
	}
	if s := h.Stray(); len(s) != 0 {
		t.Errorf("stray dials %v", s)
	}
}

func TestEnsureModelFilesRejectsABadDownload(t *testing.T) {
	for name, tc := range map[string]struct {
		handler func(w http.ResponseWriter, r *http.Request, h *embedtest.Host) bool
		want    string
	}{
		"wrong hash": {func(w http.ResponseWriter, r *http.Request, h *embedtest.Host) bool {
			b := append([]byte(nil), h.Weights...)
			b[len(b)-1] ^= 0xff
			_, _ = w.Write(b)
			return true
		}, "pinned SHA-256"},
		"cut": {func(w http.ResponseWriter, r *http.Request, h *embedtest.Host) bool {
			w.Header().Set("Content-Length", "100000000")
			_, _ = w.Write(h.Weights[:len(h.Weights)/2])
			return true
		}, "larger than announced"},
		"truncated": {func(w http.ResponseWriter, r *http.Request, h *embedtest.Host) bool {
			_, _ = w.Write(h.Weights[:len(h.Weights)/2])
			return true
		}, "cut after"},
		"larger": {func(w http.ResponseWriter, r *http.Request, h *embedtest.Host) bool {
			_, _ = w.Write(append(append([]byte(nil), h.Weights...), 1, 2, 3))
			return true
		}, "larger than announced"},
		"not found": {func(w http.ResponseWriter, r *http.Request, h *embedtest.Host) bool {
			http.NotFound(w, r)
			return true
		}, "404"},
	} {
		t.Run(name, func(t *testing.T) {
			h, spec := hostEnv(t)
			h.Handler = func(w http.ResponseWriter, r *http.Request) bool {
				if r.URL.Path != "/model.safetensors" {
					return false
				}
				return tc.handler(w, r, h)
			}
			_, err := spec.EnsureModelFiles(context.Background(), nil)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err = %v, want %q", err, tc.want)
			}
			dir, _ := ModelDir("tiny")
			if _, err := os.Stat(filepath.Join(dir, "model.safetensors")); !os.IsNotExist(err) {
				t.Errorf("the bad file was kept: %v", err)
			}
			noPartFiles(t, dir)
		})
	}
}

func TestEnsureModelFilesRedirects(t *testing.T) {
	t.Run("to another host over HTTPS", func(t *testing.T) {
		h, spec := hostEnv(t)
		other := embedtest.NewHost(t, 1)
		h.Allow(other.Listener.Addr().String())
		h.Handler = func(w http.ResponseWriter, r *http.Request) bool {
			http.Redirect(w, r, other.URL+r.URL.Path, http.StatusFound)
			return true
		}
		if _, err := spec.EnsureModelFiles(context.Background(), nil); err != nil {
			t.Fatal(err)
		}
		if other.Requests() != 2 {
			t.Errorf("the second host answered %d requests", other.Requests())
		}
	})
	t.Run("to plain HTTP", func(t *testing.T) {
		h, spec := hostEnv(t)
		plain := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write(h.Weights) }))
		t.Cleanup(plain.Close)
		h.Handler = func(w http.ResponseWriter, r *http.Request) bool {
			http.Redirect(w, r, plain.URL+r.URL.Path, http.StatusFound)
			return true
		}
		_, err := spec.EnsureModelFiles(context.Background(), nil)
		if err == nil || !strings.Contains(err.Error(), "not HTTPS") {
			t.Errorf("err = %v", err)
		}
	})
	t.Run("a fourth redirect", func(t *testing.T) {
		h, spec := hostEnv(t)
		hops := 0
		h.Handler = func(w http.ResponseWriter, r *http.Request) bool {
			hops++
			http.Redirect(w, r, h.URL+"/hop", http.StatusFound)
			return true
		}
		_, err := spec.EnsureModelFiles(context.Background(), nil)
		if err == nil || !strings.Contains(err.Error(), "redirects") {
			t.Errorf("err = %v", err)
		}
		if hops != maxRedirects+1 {
			t.Errorf("%d requests before it stopped", hops)
		}
	})
	t.Run("three redirects are followed", func(t *testing.T) {
		h, spec := hostEnv(t)
		h.Handler = func(w http.ResponseWriter, r *http.Request) bool {
			n := strings.Count(r.URL.Path, "/hop")
			switch {
			case strings.HasSuffix(r.URL.Path, "/model.safetensors") && n < 3:
				http.Redirect(w, r, h.URL+"/hop"+r.URL.Path+"", http.StatusFound)
				return true
			case strings.Contains(r.URL.Path, "/hop"):
				_, _ = w.Write(h.Weights)
				return true
			}
			return false
		}
		if _, err := spec.EnsureModelFiles(context.Background(), nil); err != nil {
			t.Fatal(err)
		}
	})
}

func TestEnsureModelFilesRefusesAnHTTPAddress(t *testing.T) {
	_, spec := hostEnv(t)
	spec.BaseURL = "http://example.invalid"
	_, err := spec.EnsureModelFiles(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "not an HTTPS address") {
		t.Errorf("err = %v", err)
	}
}

func TestEnsureModelFilesByHand(t *testing.T) {
	h, spec := hostEnv(t)
	dir, _ := ModelDir("tiny")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	embedtest.Write(t, dir, "model.safetensors", h.Weights)
	embedtest.Write(t, dir, "tokenizer.json", h.Tokenizer)
	res, err := spec.EnsureModelFiles(context.Background(), nil)
	if err != nil || len(res.Downloaded) != 0 || len(res.Present) != 2 || h.Requests() != 0 {
		t.Fatalf("%+v, %v, %d requests", res, err, h.Requests())
	}
	// Another hash is refused, with the pinned one in the message, and kept.
	embedtest.Write(t, dir, "model.safetensors", []byte("not the model"))
	_, err = spec.EnsureModelFiles(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), spec.SHA256) {
		t.Errorf("err = %v", err)
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "model.safetensors")); string(b) != "not the model" {
		t.Error("the file of the user was replaced")
	}
	if h.Requests() != 0 {
		t.Error("a download was made")
	}
}

func TestEnsureModelFilesNeedsPinnedFiles(t *testing.T) {
	t.Setenv("QUIVER_HOME", t.TempDir())
	_, err := (ModelSpec{Name: "x"}).EnsureModelFiles(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "no pinned") {
		t.Errorf("err = %v", err)
	}
}

func TestRemoveDownloadedKeepsFilesOfTheUser(t *testing.T) {
	h, spec := hostEnv(t)
	dir, _ := ModelDir("tiny")
	if _, err := spec.EnsureModelFiles(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
	// The user replaces the tokenizer by hand: it no longer has the recorded hash.
	embedtest.Write(t, dir, "tokenizer.json", []byte("mine"))
	installed := map[string]string{"model.safetensors": spec.SHA256, "tokenizer.json": spec.TokenizerSHA256}
	removed, err := RemoveDownloaded("tiny", installed)
	if err != nil || len(removed) != 1 || !strings.HasSuffix(removed[0], "model.safetensors") {
		t.Fatalf("removed %v, %v", removed, err)
	}
	if _, err := os.Stat(filepath.Join(dir, "tokenizer.json")); err != nil {
		t.Error("the file of the user was deleted")
	}
	// A file that is not recorded is never deleted.
	embedtest.Write(t, dir, "model.safetensors", h.Weights)
	if removed, _ := RemoveDownloaded("tiny", nil); len(removed) != 0 {
		t.Errorf("removed %v", removed)
	}
	if _, err := os.Stat(filepath.Join(dir, "model.safetensors")); err != nil {
		t.Error(err)
	}
}

func TestSetupRecordKeepsTheOtherKeys(t *testing.T) {
	path := settingsHome(t)
	writeSettings(t, path, `{"version": 1, "engine": "words", "local": {"minScore": 0.4}, "future": [1]}`)
	if err := RecordInstalled("tiny", map[string]string{"model.safetensors": "abc"}); err != nil {
		t.Fatal(err)
	}
	if err := UseLocal("tiny"); err != nil {
		t.Fatal(err)
	}
	if err := MarkOffered(); err != nil {
		t.Fatal(err)
	}
	cur := LoadEngine()
	if cur.Engine != EngineLocal || cur.Local.Model != "tiny" || cur.Local.MinScore == nil || *cur.Local.MinScore != 0.4 || !cur.Offered {
		t.Errorf("settings %+v", cur)
	}
	if got := Installed("tiny")["model.safetensors"]; got != "abc" {
		t.Errorf("installed %q", got)
	}
	if b, _ := os.ReadFile(path); !strings.Contains(string(b), `"future"`) {
		t.Errorf("a key was lost: %s", b)
	}
	if err := ForgetInstalled("tiny"); err != nil || Installed("tiny") != nil {
		t.Errorf("forget: %v, %v", err, Installed("tiny"))
	}
	// A file of a newer version is left as it is.
	writeSettings(t, path, `{"version": 2}`)
	if err := MarkOffered(); err == nil {
		t.Error("a newer file was changed")
	}
	if b, _ := os.ReadFile(path); string(b) != `{"version": 2}` {
		t.Errorf("file %s", b)
	}
}

func TestCheckVectorPath(t *testing.T) {
	localEnv(t)
	l := libraryOf(map[string][2]string{
		"a-slack": {"a-slack", "Send messages to a Slack channel."},
		"pdf":     {"pdf", "Read and fill in PDF forms."},
		"notes":   {"notes", "Keep meeting notes."},
		"tdd":     {"tdd", "Write the test first."},
		"sql":     {"sql", "Tune slow database queries."},
		"deploy":  {"deploy", "Ship a release."},
		"mail":    {"mail", "Write email."},
		"maps":    {"maps", "Draw maps."},
	})
	if _, err := l.Reindex(context.Background(), "tiny", nil); err != nil {
		t.Fatal(err)
	}
	note, err := l.CheckVectorPath("tiny")
	if err != nil || !strings.Contains(note, "found at place") {
		t.Fatalf("%q, %v", note, err)
	}
	// A broken vector path: the vector of the skill points the other way.
	x := loadVectors(models["tiny"])
	for i := range x.Entries {
		if x.Entries[i].ID == "a-slack" {
			for k := range x.Entries[i].Vector {
				x.Entries[i].Vector[k] = -x.Entries[i].Vector[k]
			}
		}
	}
	path, _ := vectorPath()
	if err := os.WriteFile(path, mustJSON(t, x), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := l.CheckVectorPath("tiny"); err == nil || !strings.Contains(err.Error(), "first five") {
		t.Errorf("err = %v", err)
	}
	if note, err := libraryOf(nil).CheckVectorPath("tiny"); err != nil || !strings.Contains(note, "no skills") {
		t.Errorf("empty library: %q, %v", note, err)
	}
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestPinnedModelMatchesTheSpikeResults(t *testing.T) {
	doc, err := os.ReadFile(filepath.Join("..", "..", "docs", "spike-discovery-results.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(doc)
	spec := models[DefaultModel]
	for _, want := range []string{
		spec.SHA256, spec.TokenizerSHA256,
		"90,868,376", "466,247",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("docs/spike-discovery-results.md does not record %q", want)
		}
	}
	m := regexp.MustCompile(`/resolve/([0-9a-f]{40})$`).FindStringSubmatch(spec.BaseURL)
	if m == nil || !strings.Contains(text, m[1]) {
		t.Errorf("the pinned revision of %s is not the full commit of the results", spec.BaseURL)
	}
	if !strings.HasPrefix(spec.BaseURL, "https://") {
		t.Error("the model address is not HTTPS")
	}
	if spec.Size != 90868376 || spec.TokenizerSize != 466247 {
		t.Errorf("sizes %d, %d", spec.Size, spec.TokenizerSize)
	}
}
