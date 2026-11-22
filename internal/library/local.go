package library

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/kheif/quiver/internal/embed"
	"github.com/kheif/quiver/internal/skill"
)

// EngineLocal is the engine that searches by meaning with a small embedding
// model that Quiver runs itself, fused with the words engine.
const EngineLocal = "local"

// RankMeaning is the rank of a skill that only the vectors found, after the
// last tier of the words engine.
const RankMeaning = RankDescription + 1

const (
	// maxVectorCandidates is how many vector matches go into the fusion.
	maxVectorCandidates = 50
	// fusionK is the constant of the reciprocal rank fusion.
	fusionK = 60
	// DefaultBuildLimit is how long a search builds the missing part of the
	// index before it answers with what it has.
	DefaultBuildLimit = 60 * time.Second
)

// Refresh says how a search brings the index up to date.
type Refresh int

const (
	// RefreshSync builds the missing vectors before it answers, for up to
	// Options.Limit. It is the zero value: the CLI.
	RefreshSync Refresh = iota
	// RefreshBackground builds them in a goroutine and answers at once with
	// the vectors it has: the server and the panel.
	RefreshBackground
	// RefreshNone embeds nothing.
	RefreshNone
)

// Options say how a search brings its index up to date.
type Options struct {
	Refresh Refresh
	// Limit is the time RefreshSync may build for; zero is DefaultBuildLimit.
	Limit time.Duration
	// Progress is told how many skills are embedded of how many are missing.
	Progress func(done, total int)
}

// localEngine is the Searcher of the engine "local".
type localEngine struct {
	spec     ModelSpec
	minScore float64
	all      []*skill.Skill
	words    *Words
	opts     Options

	mu   sync.Mutex
	note string
}

func newLocal(l *Library, env engineEnv) (Searcher, error) {
	name := env.settings.Local.Model
	if name == "" {
		name = DefaultModel
	}
	spec, ok := models[name]
	if !ok {
		return nil, fmt.Errorf("unknown model %q", name)
	}
	if err := spec.CheckModelFile(); err != nil {
		return nil, err
	}
	e := &localEngine{spec: spec, minScore: spec.MinScore, all: l.All(), words: env.words, opts: env.opts}
	if v := env.settings.Local.MinScore; v != nil {
		e.minScore = *v
	}
	return e, nil
}

// Note is a line to say after the last search: the index is incomplete.
func (e *localEngine) Note() string {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.note
}

func (e *localEngine) setNote(s string) {
	e.mu.Lock()
	e.note = s
	e.mu.Unlock()
}

type vectorHit struct {
	skill *skill.Skill
	score float32
}

func (e *localEngine) Search(ctx context.Context, query string, skills []*skill.Skill) ([]Match, error) {
	e.setNote("")
	byWords, err := e.words.Search(ctx, query, skills)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(query) == "" {
		return byWords, nil
	}
	model, err := openModel(e.spec)
	if err != nil {
		return nil, err
	}
	x := loadVectors(e.spec)
	st, missing := vectorStatus(x, e.all)
	if len(missing) > 0 {
		switch e.opts.Refresh {
		case RefreshSync:
			if err := e.buildNow(ctx, model, missing); err != nil {
				return nil, err
			}
			x = loadVectors(e.spec)
			st, _ = vectorStatus(x, e.all)
			if st.Stale > 0 {
				e.setNote(fmt.Sprintf("the search index is incomplete: %d of %d skills have a vector; the next search builds the rest, and quiver engine reindex builds it all", st.Indexed, st.Total))
			}
		case RefreshBackground:
			buildInBackground(e.spec, e.all, missing)
		}
	}
	q, err := embedder(model, query)
	if err != nil {
		return nil, fmt.Errorf("embedding the query: %w", err)
	}
	return fuse(byWords, e.vectorHits(x, q, skills)), nil
}

// buildNow embeds the missing skills for up to the limit. Running out of time
// is not an error: the search answers with the vectors it has.
func (e *localEngine) buildNow(ctx context.Context, model *embed.Model, missing []*skill.Skill) error {
	limit := cmp.Or(e.opts.Limit, DefaultBuildLimit)
	bctx, cancel := context.WithTimeout(ctx, limit)
	defer cancel()
	err := refreshIndex(bctx, e.spec, model, e.all, missing, e.opts.Progress)
	if err != nil && !errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	return nil
}

// vectorHits scores the skills of the set that have a current vector, drops
// those below minScore and keeps the best maxVectorCandidates.
func (e *localEngine) vectorHits(x *vectorIndex, q []float32, skills []*skill.Skill) []vectorHit {
	have := x.byID()
	var hits []vectorHit
	for _, s := range skills {
		ent, ok := have[s.ID]
		if !ok || ent.Hash != textHash(embeddedText(s)) {
			continue
		}
		if score := cosine(q, ent.Vector); float64(score) >= e.minScore {
			hits = append(hits, vectorHit{s, score})
		}
	}
	slices.SortFunc(hits, func(a, b vectorHit) int {
		return cmp.Or(cmp.Compare(b.score, a.score), cmp.Compare(a.skill.ID, b.skill.ID))
	})
	if len(hits) > maxVectorCandidates {
		hits = hits[:maxVectorCandidates]
	}
	return hits
}

func cosine(a, b []float32) float32 {
	var s float32
	for i := range a {
		s += a[i] * b[i]
	}
	return s
}

// fuse combines the matches of the words engine with the vector matches by
// reciprocal rank fusion over two orders: the order of the words matches
// (matches of one rank and score share a rank) and the order of the vector
// scores, where a skill without a vector score comes after those with one.
// Ties break by vector score, then by ID. Matches of the exact-name tier stay
// above everything the fusion produces. A skill that only the vectors found
// has the rank RankMeaning; the others keep the tier of the words engine.
func fuse(byWords []Match, vec []vectorHit) []Match {
	var exact []Match
	wordsRank := map[string]int{}
	var rest []Match
	rank, prev := 0, -1
	for i, m := range byWords {
		if i == 0 || m.Rank != byWords[prev].Rank || m.Score != byWords[prev].Score {
			rank = i + 1
			prev = i
		}
		wordsRank[m.ID] = rank
		if m.Rank == RankExactName {
			exact = append(exact, m)
		} else {
			rest = append(rest, m)
		}
	}
	vecRank := map[string]int{}
	vecScore := map[string]float32{}
	for i, h := range vec {
		vecRank[h.skill.ID] = i + 1
		vecScore[h.skill.ID] = h.score
	}
	type cand struct {
		m     Match
		fused float64
		v     float32
		hasV  bool
	}
	var cands []cand
	for _, m := range rest {
		c := cand{m: m}
		c.fused = 1 / float64(fusionK+wordsRank[m.ID])
		vr, ok := vecRank[m.ID]
		if !ok {
			vr = len(vec) + 1
		}
		c.fused += 1 / float64(fusionK+vr)
		c.v, c.hasV = vecScore[m.ID], ok
		cands = append(cands, c)
	}
	inWords := map[string]bool{}
	for _, m := range byWords {
		inWords[m.ID] = true
	}
	for _, h := range vec {
		if inWords[h.skill.ID] {
			continue
		}
		cands = append(cands, cand{
			m:     Match{Skill: h.skill, Rank: RankMeaning, Score: float64(h.score)},
			fused: 1/float64(fusionK+len(byWords)+1) + 1/float64(fusionK+vecRank[h.skill.ID]),
			v:     h.score, hasV: true,
		})
	}
	slices.SortFunc(cands, func(a, b cand) int {
		return cmp.Or(cmp.Compare(b.fused, a.fused), cmp.Compare(b.v, a.v), cmp.Compare(a.m.ID, b.m.ID))
	})
	out := append([]Match(nil), exact...)
	for _, c := range cands {
		out = append(out, c.m)
	}
	return out
}

// background is the build that a long-lived process runs for the index; one
// at a time, whichever search started it.
var background struct {
	sync.Mutex
	running bool
}

// buildInBackground embeds the missing skills in a goroutine, unless one is
// running. A failed embedding stops it: the next search starts it again.
func buildInBackground(spec ModelSpec, all, missing []*skill.Skill) {
	background.Lock()
	if background.running {
		background.Unlock()
		return
	}
	background.running = true
	background.Unlock()
	go func() {
		defer func() {
			background.Lock()
			background.running = false
			background.Unlock()
		}()
		m, err := openModel(spec)
		if err != nil {
			return
		}
		_ = refreshIndex(context.Background(), spec, m, all, missing, nil)
	}()
}
