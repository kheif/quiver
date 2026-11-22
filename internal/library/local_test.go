package library

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kheif/quiver/internal/embed"
	"github.com/kheif/quiver/internal/embed/embedtest"
	"github.com/kheif/quiver/internal/fsutil"
	"github.com/kheif/quiver/internal/skill"
)

// localEnv makes a Quiver home with the tiny model of embedtest under the name
// "tiny", chooses the local engine with it, and counts the embeddings.
func localEnv(t *testing.T) *atomic.Int64 {
	t.Helper()
	t.Setenv("QUIVER_HOME", t.TempDir())
	dir, err := ModelDir("tiny")
	if err != nil {
		t.Fatal(err)
	}
	embedtest.WriteModel(t, dir, 1)
	st, err := os.Stat(filepath.Join(dir, "model.safetensors"))
	if err != nil {
		t.Fatal(err)
	}
	models["tiny"] = ModelSpec{Name: "tiny", Config: embed.Config(embedtest.Tiny), Size: st.Size(), SHA256: "tiny-sha", MinScore: -1}
	var calls atomic.Int64
	orig := embedder
	embedder = func(m *embed.Model, text string) ([]float32, error) {
		calls.Add(1)
		return orig(m, text)
	}
	t.Cleanup(func() {
		delete(models, "tiny")
		embedder = orig
		loaded.Lock()
		loaded.m = nil
		loaded.Unlock()
	})
	useLocal(t, `{"model": "tiny", "minScore": -1}`)
	return &calls
}

func useLocal(t *testing.T, block string) {
	t.Helper()
	path, _ := searchSettingsPath()
	writeSettings(t, path, `{"version": 1, "engine": "local", "local": `+block+`}`)
}

func fiveSkills() *Library {
	return libraryOf(map[string][2]string{
		"slack": {"slack", "Send messages to a Slack channel."},
		"pdf":   {"pdf", "Read and fill in PDF forms."},
		"notes": {"notes", "Keep meeting notes."},
		"tdd":   {"tdd", "Write the test first."},
		"sql":   {"sql", "Tune slow database queries."},
	})
}

func searchLocal(t *testing.T, l *Library, opts Options, query string) ([]Match, string, *Search) {
	t.Helper()
	s, err := l.NewSearchWith("", opts)
	if err != nil {
		t.Fatal(err)
	}
	m, note := s.Find(context.Background(), query, l.All())
	return m, note, s
}

func readIndex(t *testing.T) *vectorIndex {
	t.Helper()
	path, _ := vectorPath()
	var x vectorIndex
	if err := fsutil.ReadJSON(path, &x); err != nil {
		t.Fatal(err)
	}
	return &x
}

func TestLocalEngineIndexesWhenASearchNeedsIt(t *testing.T) {
	calls := localEnv(t)
	l := fiveSkills()
	if _, err := os.Stat(mustVectorPath(t)); !os.IsNotExist(err) {
		t.Fatal("an index before any search")
	}
	_, note, s := searchLocal(t, l, Options{}, "send a message")
	if s.Engine != EngineLocal || s.Problem != "" || note != "" {
		t.Errorf("engine %s, problem %q, note %q", s.Engine, s.Problem, note)
	}
	if calls.Load() != 6 { // five skills and the query
		t.Errorf("%d embeddings, want 6", calls.Load())
	}
	x := readIndex(t)
	if x.Version != 1 || x.Model != "tiny" || x.SHA256 != "tiny-sha" || x.Dim != 8 || len(x.Entries) != 5 {
		t.Errorf("index %+v", x)
	}
	// The second search embeds the query only.
	calls.Store(0)
	searchLocal(t, l, Options{}, "send a message")
	if calls.Load() != 1 {
		t.Errorf("%d embeddings, want 1", calls.Load())
	}
}

func mustVectorPath(t *testing.T) string {
	t.Helper()
	p, err := vectorPath()
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// A changed skill is embedded again, a deleted one is dropped when the file is
// next written, and the others are left alone.
func TestLocalEngineRefreshesWhatChanged(t *testing.T) {
	calls := localEnv(t)
	l := fiveSkills()
	searchLocal(t, l, Options{}, "x")
	calls.Store(0)

	l.skills["pdf"].Description = "Merge and split PDF files." // a changed text, as an edit or a pull gives
	searchLocal(t, l, Options{}, "x")
	if calls.Load() != 2 { // the skill and the query
		t.Errorf("%d embeddings after a change, want 2", calls.Load())
	}

	delete(l.skills, "tdd")
	l.skills["new"] = &skill.Skill{ID: "new", Name: "new", Description: "Brand new."}
	calls.Store(0)
	searchLocal(t, l, Options{}, "x")
	x := readIndex(t)
	var ids []string
	for _, e := range x.Entries {
		ids = append(ids, e.ID)
	}
	if want := []string{"new", "notes", "pdf", "slack", "sql"}; !slices.Equal(ids, want) {
		t.Errorf("entries %v, want %v", ids, want)
	}
	if calls.Load() != 2 {
		t.Errorf("%d embeddings for a new skill, want 2", calls.Load())
	}
}

func TestLocalEngineEmbedsNothingUnderWords(t *testing.T) {
	calls := localEnv(t)
	writeSettings(t, mustSettingsPath(t), `{"version": 1, "engine": "words"}`)
	l := fiveSkills()
	if _, note, s := searchLocal(t, l, Options{}, "slack"); s.Engine != EngineWords || note != "" || s.Problem != "" {
		t.Errorf("engine %s, note %q, problem %q", s.Engine, note, s.Problem)
	}
	if calls.Load() != 0 {
		t.Errorf("%d embeddings under words", calls.Load())
	}
	if _, err := os.Stat(mustVectorPath(t)); !os.IsNotExist(err) {
		t.Error("an index under words")
	}
}

func mustSettingsPath(t *testing.T) string {
	t.Helper()
	p, err := searchSettingsPath()
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// A missing or a changed model file gives the words matches with one line,
// and embeds nothing.
func TestLocalEngineWithAnUnusableModelFile(t *testing.T) {
	calls := localEnv(t)
	l := fiveSkills()
	dir, _ := ModelDir("tiny")
	weights := filepath.Join(dir, "model.safetensors")
	data, _ := os.ReadFile(weights)

	for name, change := range map[string]func(){
		"missing":   func() { _ = os.Remove(weights) },
		"a size":    func() { _ = os.WriteFile(weights, append(data, 0), 0o644) },
		"tokenizer": func() { _ = os.Remove(filepath.Join(dir, "tokenizer.json")) },
	} {
		_ = os.WriteFile(weights, data, 0o644)
		embedtest.Write(t, dir, "tokenizer.json", embedtest.Tokenizer(nil))
		change()
		got, _, s := searchLocal(t, l, Options{}, "slack")
		if s.Engine != EngineWords || !strings.Contains(s.Problem, "search engine local cannot run") || !strings.Contains(s.Problem, "searching with words") {
			t.Errorf("%s: engine %s, problem %q", name, s.Engine, s.Problem)
		}
		if len(got) != 1 || got[0].ID != "slack" {
			t.Errorf("%s: %v", name, ids(got))
		}
	}
	if calls.Load() != 0 {
		t.Errorf("%d embeddings with an unusable model file", calls.Load())
	}
	if _, err := os.Stat(mustVectorPath(t)); !os.IsNotExist(err) {
		t.Error("an index was made")
	}
}

// A failed embedding falls back to words with one line and leaves the index as
// it was; a kill or a failure after a batch keeps that batch.
func TestLocalEngineFailedEmbeddingFallsBack(t *testing.T) {
	calls := localEnv(t)
	l := fiveSkills()
	searchLocal(t, l, Options{}, "x")
	before, _ := os.ReadFile(mustVectorPath(t))

	l.skills["pdf"].Description = "Changed text."
	orig := embedder
	embedder = func(*embed.Model, string) ([]float32, error) { return nil, errors.New("out of memory") }
	t.Cleanup(func() { embedder = orig })
	got, note, _ := searchLocal(t, l, Options{}, "slack")
	if !strings.Contains(note, "search engine local cannot run (embedding pdf: out of memory); searched with words") || strings.Count(note, "\n") != 0 {
		t.Errorf("note %q", note)
	}
	if len(got) != 1 || got[0].ID != "slack" {
		t.Errorf("words matches: %v", ids(got))
	}
	after, _ := os.ReadFile(mustVectorPath(t))
	if string(before) != string(after) {
		t.Error("a failed embedding changed the index")
	}
	_ = calls
}

func TestLocalEngineKeepsTheBatchesBeforeAFailure(t *testing.T) {
	calls := localEnv(t)
	files := map[string][2]string{}
	for i := range 12 {
		files[fmt.Sprintf("s%02d", i)] = [2]string{fmt.Sprintf("s%02d", i), fmt.Sprintf("Skill number %d.", i)}
	}
	l := libraryOf(files)
	orig := embedder
	var n atomic.Int64
	embedder = func(m *embed.Model, text string) ([]float32, error) {
		if n.Add(1) == 11 { // the third skill of the second batch
			return nil, errors.New("killed")
		}
		return orig(m, text)
	}
	t.Cleanup(func() { embedder = orig })
	_, note, _ := searchLocal(t, l, Options{}, "skill")
	if !strings.Contains(note, "killed") {
		t.Errorf("note %q", note)
	}
	if got := len(readIndex(t).Entries); got != batchSize {
		t.Errorf("%d entries after a failure in the second batch, want %d", got, batchSize)
	}
	_ = calls
}

func TestLocalEngineRebuildsAnIndexThatDoesNotFit(t *testing.T) {
	calls := localEnv(t)
	l := fiveSkills()
	searchLocal(t, l, Options{}, "x")
	path := mustVectorPath(t)
	for name, change := range map[string]func(x *vectorIndex){
		"another format": func(x *vectorIndex) { x.Version = 99 },
		"another model":  func(x *vectorIndex) { x.SHA256 = "another-file" },
		"another name":   func(x *vectorIndex) { x.Model = "other" },
		"another dim":    func(x *vectorIndex) { x.Dim = 384 },
	} {
		x := readIndex(t)
		change(x)
		if err := fsutil.WriteJSONAtomic(path, x); err != nil {
			t.Fatal(err)
		}
		calls.Store(0)
		searchLocal(t, l, Options{}, "x")
		if calls.Load() != 6 {
			t.Errorf("%s: %d embeddings, want 6 (all skills and the query)", name, calls.Load())
		}
		if got := readIndex(t); got.Version != 1 || got.Dim != 8 || len(got.Entries) != 5 {
			t.Errorf("%s: index %+v", name, got)
		}
	}
	// A file that is not JSON is rebuilt too.
	if err := os.WriteFile(path, []byte("{broken"), 0o644); err != nil {
		t.Fatal(err)
	}
	searchLocal(t, l, Options{}, "x")
	if len(readIndex(t).Entries) != 5 {
		t.Error("a broken index was not rebuilt")
	}
}

// A search builds the missing part for the limit it is given, then answers
// with what it has and says that the index is incomplete.
func TestLocalEngineColdIndexInTheCLI(t *testing.T) {
	calls := localEnv(t)
	files := map[string][2]string{}
	for i := range 40 {
		files[fmt.Sprintf("s%02d", i)] = [2]string{fmt.Sprintf("s%02d", i), fmt.Sprintf("Skill number %d.", i)}
	}
	l := libraryOf(files)
	orig := embedder
	embedder = func(m *embed.Model, text string) ([]float32, error) {
		time.Sleep(15 * time.Millisecond)
		return orig(m, text)
	}
	t.Cleanup(func() { embedder = orig })
	var progress []int
	_, note, _ := searchLocal(t, l, Options{Limit: 100 * time.Millisecond, Progress: func(done, total int) { progress = append(progress, done) }}, "skill")
	if !strings.Contains(note, "the search index is incomplete") || !strings.Contains(note, "quiver engine reindex") {
		t.Errorf("note %q", note)
	}
	if len(progress) == 0 || progress[len(progress)-1] >= 40 {
		t.Errorf("progress %v", progress)
	}
	got := len(readIndex(t).Entries)
	if got == 0 || got >= 40 {
		t.Errorf("%d entries saved", got)
	}
	// The next search goes on from there.
	embedder = orig
	calls.Store(0)
	_, note, _ = searchLocal(t, l, Options{}, "skill")
	if note != "" || len(readIndex(t).Entries) != 40 || calls.Load() != int64(40-got)+1 {
		t.Errorf("note %q, entries %d, %d embeddings", note, len(readIndex(t).Entries), calls.Load())
	}
}

// A long-lived process answers at once with what it has, and builds the rest
// in the background.
func TestLocalEngineBackgroundBuild(t *testing.T) {
	localEnv(t)
	l := fiveSkills()
	got, note, _ := searchLocal(t, l, Options{Refresh: RefreshBackground}, "slack")
	if note != "" || len(got) == 0 || got[0].ID != "slack" {
		t.Errorf("answer %v, note %q", ids(got), note)
	}
	deadline := time.Now().Add(10 * time.Second)
	for {
		background.Lock()
		running := background.running
		background.Unlock()
		if _, err := os.Stat(mustVectorPath(t)); err == nil && !running && len(readIndex(t).Entries) == 5 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("the background build did not finish")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestLocalEngineRefreshNoneEmbedsNoSkill(t *testing.T) {
	calls := localEnv(t)
	l := fiveSkills()
	searchLocal(t, l, Options{Refresh: RefreshNone}, "slack")
	if calls.Load() != 1 { // the query
		t.Errorf("%d embeddings", calls.Load())
	}
}

// The embedding runs outside the lock: a lock taken inside the embedder would
// block for ever if the lock were held.
func TestRefreshEmbedsOutsideTheLockAndMergesAfterEachBatch(t *testing.T) {
	localEnv(t)
	l := fiveSkills()
	spec := models["tiny"]
	m, err := openModel(spec)
	if err != nil {
		t.Fatal(err)
	}
	orig := embedder
	embedder = func(m *embed.Model, text string) ([]float32, error) {
		done := make(chan error, 1)
		go func() { done <- fsutil.WithLock(func() error { return nil }) }()
		select {
		case err := <-done:
			if err != nil {
				return nil, err
			}
		case <-time.After(5 * time.Second):
			return nil, errors.New("the lock was held during the embedding")
		}
		return orig(m, text)
	}
	t.Cleanup(func() { embedder = orig })
	all := l.All()
	if err := refreshIndex(context.Background(), spec, m, all, all, nil); err != nil {
		t.Fatal(err)
	}
	if len(readIndex(t).Entries) != 5 {
		t.Error("entries missing")
	}
}

// Two processes that embed at the same time do some work twice, and neither
// loses the other's entries.
func TestTwoRefreshesAtOnceLoseNothing(t *testing.T) {
	localEnv(t)
	files := map[string][2]string{}
	for i := range 30 {
		files[fmt.Sprintf("s%02d", i)] = [2]string{fmt.Sprintf("s%02d", i), fmt.Sprintf("Skill number %d.", i)}
	}
	l := libraryOf(files)
	spec := models["tiny"]
	m, err := openModel(spec)
	if err != nil {
		t.Fatal(err)
	}
	all := l.All()
	var wg sync.WaitGroup
	for _, part := range [][]*skill.Skill{all[:20], all[10:]} { // overlapping
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := refreshIndex(context.Background(), spec, m, all, part, nil); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if got := len(readIndex(t).Entries); got != 30 {
		t.Errorf("%d entries, want 30", got)
	}
}

func TestReindexRebuildsTheWholeIndex(t *testing.T) {
	calls := localEnv(t)
	l := fiveSkills()
	searchLocal(t, l, Options{}, "x")
	calls.Store(0)
	var last int
	n, err := l.Reindex(context.Background(), "tiny", func(done, total int) { last = done })
	if err != nil || n != 5 || calls.Load() != 5 || last != 5 {
		t.Errorf("n %d, err %v, %d embeddings, progress %d", n, err, calls.Load(), last)
	}
	if st, _ := l.IndexStatus("tiny"); st != (VectorStatus{Indexed: 5, Total: 5}) {
		t.Errorf("status %+v", st)
	}
	l.skills["sql"].Description = "Changed."
	if st, _ := l.IndexStatus("tiny"); st != (VectorStatus{Indexed: 4, Stale: 1, Total: 5}) {
		t.Errorf("status after a change %+v", st)
	}
	if _, err := l.Reindex(context.Background(), "nope", nil); err == nil {
		t.Error("an unknown model was reindexed")
	}
}

func TestReindexWithoutTheModelFile(t *testing.T) {
	localEnv(t)
	dir, _ := ModelDir("tiny")
	_ = os.Remove(filepath.Join(dir, "model.safetensors"))
	if _, err := fiveSkills().Reindex(context.Background(), "tiny", nil); err == nil || !strings.Contains(err.Error(), "is missing") {
		t.Errorf("err = %v", err)
	}
}

func TestLocalSettings(t *testing.T) {
	localEnv(t)
	useLocal(t, `{"model": "tiny", "minScore": 0.5}`)
	cur := LoadEngine()
	if cur.Engine != "local" || cur.Local.Model != "tiny" || cur.Local.MinScore == nil || *cur.Local.MinScore != 0.5 {
		t.Errorf("%+v", cur)
	}
	useLocal(t, `{"minScore": 7}`)
	if cur := LoadEngine(); cur.Engine != "words" || !strings.Contains(cur.Problem, "minScore") {
		t.Errorf("%+v", cur)
	}
	useLocal(t, `{"model": "nope"}`)
	s, err := fiveSkills().NewSearch("")
	if err != nil || s.Engine != "words" || !strings.Contains(s.Problem, `unknown model "nope"`) {
		t.Errorf("%+v, %v", s, err)
	}
	if err := UseEngine("local"); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(mustSettingsPath(t))
	if !strings.Contains(string(data), `"model": "nope"`) || !strings.Contains(string(data), `"engine": "local"`) {
		t.Errorf("use rewrote the block: %s", data)
	}
}

// minScore drops a vector match, and a query that fits nothing leaves only the
// words matches.
func TestVectorHitsHonourMinScore(t *testing.T) {
	a := &skill.Skill{ID: "a", Name: "a", Description: "d"}
	b := &skill.Skill{ID: "b", Name: "b", Description: "d"}
	c := &skill.Skill{ID: "c", Name: "c", Description: "d"}
	x := &vectorIndex{Entries: []vectorEntry{
		{ID: "a", Hash: textHash(embeddedText(a)), Vector: []float32{1, 0}},
		{ID: "b", Hash: textHash(embeddedText(b)), Vector: []float32{0.6, 0.8}},
		{ID: "c", Hash: "stale", Vector: []float32{1, 0}},
	}}
	e := &localEngine{minScore: 0.5}
	got := e.vectorHits(x, []float32{1, 0}, []*skill.Skill{a, b, c})
	if len(got) != 2 || got[0].skill.ID != "a" || got[1].skill.ID != "b" {
		t.Errorf("%+v", got)
	}
	e.minScore = 0.7
	if got := e.vectorHits(x, []float32{1, 0}, []*skill.Skill{a, b, c}); len(got) != 1 || got[0].skill.ID != "a" {
		t.Errorf("minScore 0.7: %+v", got)
	}
	e.minScore = 1.01
	if got := e.vectorHits(x, []float32{1, 0}, []*skill.Skill{a, b, c}); len(got) != 0 {
		t.Errorf("a query that fits nothing: %+v", got)
	}
}

func TestVectorHitsAreCapped(t *testing.T) {
	var skills []*skill.Skill
	x := &vectorIndex{}
	for i := range 80 {
		s := &skill.Skill{ID: fmt.Sprintf("s%02d", i), Name: "n", Description: "d"}
		skills = append(skills, s)
		x.Entries = append(x.Entries, vectorEntry{ID: s.ID, Hash: textHash(embeddedText(s)), Vector: []float32{1}})
	}
	e := &localEngine{minScore: 0}
	got := e.vectorHits(x, []float32{1}, skills)
	if len(got) != maxVectorCandidates || got[0].skill.ID != "s00" {
		t.Errorf("%d hits, first %s", len(got), got[0].skill.ID)
	}
}

func skillOf(id string) *skill.Skill { return &skill.Skill{ID: id, Name: id, Description: id} }

func TestFusion(t *testing.T) {
	m := func(id string, rank int, score float64) Match {
		return Match{Skill: skillOf(id), Rank: rank, Score: score}
	}
	h := func(id string, score float32) vectorHit { return vectorHit{skillOf(id), score} }
	order := func(ms []Match) string {
		var out []string
		for _, x := range ms {
			out = append(out, x.ID)
		}
		return strings.Join(out, ",")
	}

	// An exact name stays first, though its vector rank is low or missing.
	got := fuse(
		[]Match{m("exact", RankExactName, 0), m("w1", RankDescription, 12), m("w2", RankDescription, 11)},
		[]vectorHit{h("w2", 0.9), h("w1", 0.8), h("v1", 0.7)})
	if got[0].ID != "exact" {
		t.Errorf("exact name not first: %s", order(got))
	}
	// The others: w2 is high in both orders; v1 is only in the vectors.
	if order(got) != "exact,w2,w1,v1" {
		t.Errorf("order %s", order(got))
	}
	// A vector-only match has RankMeaning and its vector score; the others
	// keep the tier of the words engine.
	for _, x := range got {
		switch x.ID {
		case "v1":
			if x.Rank != RankMeaning || x.Score != float64(float32(0.7)) {
				t.Errorf("v1: %+v", x)
			}
		case "w1", "w2":
			if x.Rank != RankDescription {
				t.Errorf("%s: rank %d", x.ID, x.Rank)
			}
		}
	}

	// A query that fits nothing returns only the words matches.
	got = fuse([]Match{m("a", RankNameContains, 0), m("b", RankDescription, 10)}, nil)
	if order(got) != "a,b" {
		t.Errorf("no vector candidates: %s", order(got))
	}
	// Nothing at all.
	if got := fuse(nil, nil); len(got) != 0 {
		t.Errorf("%v", got)
	}

	// Equal ranks and scores share a rank: the tie goes to the vector score.
	got = fuse(
		[]Match{m("a", RankDescription, 10), m("b", RankDescription, 10), m("c", RankDescription, 10)},
		[]vectorHit{h("c", 0.9), h("b", 0.8)})
	if order(got) != "c,b,a" {
		t.Errorf("shared rank: %s", order(got))
	}
	// Without a vector score a skill comes after those with one, and ties
	// there break by ID.
	got = fuse(
		[]Match{m("a", RankDescription, 10), m("b", RankDescription, 10)}, []vectorHit{h("b", 0.5)})
	if order(got) != "b,a" {
		t.Errorf("a skill without a vector score: %s", order(got))
	}
	got = fuse([]Match{m("z", RankDescription, 10), m("a", RankDescription, 10)}, nil)
	if order(got) != "a,z" {
		t.Errorf("tie by ID: %s", order(got))
	}
}

// The words matches of a vector-only candidate are not duplicated.
func TestFusionDoesNotDuplicate(t *testing.T) {
	got := fuse([]Match{{Skill: skillOf("a"), Rank: RankNamePrefix}}, []vectorHit{{skillOf("a"), 0.9}, {skillOf("b"), 0.8}})
	if len(got) != 2 || got[0].ID != "a" || got[1].Rank != RankMeaning {
		t.Errorf("%+v", got)
	}
}
