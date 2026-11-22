package library

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/kheif/quiver/internal/skill"
)

// QUIVER_TEST_MODEL names a folder with the real all-MiniLM-L6-v2 files. The
// test installs them in a new Quiver home, builds the index of the spike's
// library (spike/discovery/corpus.json) and runs the English queries of its
// test half: the rates of the README. The time it takes is the time of
// embedding the library with this machine's cores.
func TestRealModelOnTheSpikeQueries(t *testing.T) {
	src := os.Getenv("QUIVER_TEST_MODEL")
	if src == "" {
		t.Skip("QUIVER_TEST_MODEL does not name the all-MiniLM-L6-v2 folder")
	}
	all, qs := loadSpike(t)
	t.Setenv("QUIVER_HOME", t.TempDir())
	dir, _ := ModelDir(DefaultModel)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"model.safetensors", "tokenizer.json"} {
		if err := os.Symlink(filepath.Join(src, f), filepath.Join(dir, f)); err != nil {
			t.Fatal(err)
		}
	}
	spec := models[DefaultModel]
	if err := spec.CheckModelFile(); err != nil {
		t.Fatal(err)
	}
	path, _ := searchSettingsPath()
	writeSettings(t, path, `{"version": 1, "engine": "local"}`)
	l := &Library{skills: map[string]*skill.Skill{}}
	for _, s := range all {
		l.skills[s.ID] = s
	}
	start := time.Now()
	s, err := l.NewSearchWith("", Options{Limit: 10 * time.Minute})
	if err != nil || s.Engine != EngineLocal {
		t.Fatalf("engine %s, %v, problem %q", s.Engine, err, s.Problem)
	}
	_, test := englishHalves(qs)
	var fit, hit, nofit, empty int
	for _, q := range test {
		got, note := s.Find(context.Background(), q.Text, l.All())
		if note != "" {
			t.Fatal(note)
		}
		if fitsNothing(q) {
			nofit++
			if len(got) == 0 {
				empty++
			}
			continue
		}
		fit++
		var top []string
		for _, m := range got[:min(5, len(got))] {
			top = append(top, m.ID)
		}
		if slices.ContainsFunc(q.Gold, func(g string) bool { return slices.Contains(top, g) }) {
			hit++
		}
	}
	t.Logf("local: %d of %d fitting queries have a fitting skill in the first five; %d of %d queries that fit nothing return nothing (index and queries took %s)", hit, fit, empty, nofit, time.Since(start).Round(time.Second))
	if rate := float64(hit) / float64(fit); rate < 0.85 {
		t.Errorf("top-5 rate %.0f%%, the spike measured 93%% fused", rate*100)
	}
	if st, _ := l.IndexStatus(DefaultModel); st.Stale != 0 {
		t.Errorf("status %+v", st)
	}
}
