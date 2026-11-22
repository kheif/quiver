package library

import (
	"cmp"
	"context"
	"fmt"
	"math"
	"slices"
	"strings"
	"unicode"

	"github.com/kheif/quiver/internal/skill"
)

// How well a skill matches a search; lower is better.
const (
	RankExactName = iota
	RankNamePrefix
	RankNameContains
	// RankDescription is a match by the ranked word search of names and
	// descriptions; the Score tells how good it is.
	RankDescription
)

// Match is a skill found by a search.
type Match struct {
	*skill.Skill
	Rank int
	// Score is the BM25 score of the words engine. Other engines may leave it
	// zero.
	Score float64
}

// Searcher finds skills by words. skills is the set that can match: the
// callers choose it, and an engine ranks and caps within it. An engine takes
// whatever statistics it needs from the whole library, so the same skill gets
// the same score in any set.
type Searcher interface {
	Search(ctx context.Context, query string, skills []*skill.Skill) ([]Match, error)
}

// fold lower-cases s and treats hyphens and underscores like spaces, so
// "pull request" finds pull-request.
func fold(s string) string {
	return strings.NewReplacer("-", " ", "_", " ").Replace(strings.ToLower(s))
}

// descriptionCutoff is the score a match by name and description words must
// exceed. It is the highest top score of the English queries that fit nothing
// in the calibration half of TASK62.1, so that none of them returns anything:
// 9.784 on the 262 skills of the spike, 9.811 on the 263 of the library when
// TASK62.5 checked it again (TestSpikeQueriesOnTheWholeLibrary).
const descriptionCutoff = 9.811

// BM25 parameters and the weight of a word of the name, as TASK62.1 measured
// them.
const (
	bm25K1     = 1.2
	bm25B      = 0.75
	nameWeight = 3.0
)

var stopWords = func() map[string]bool {
	m := map[string]bool{}
	for _, w := range strings.Fields("a an the of to for and or in on at by with from my our your is are was be it its that this these those what which who how do does did i we you they me us can could should would will about into so as if than then not no up out over per via use used using when where why") {
		m[w] = true
	}
	return m
}()

// stem takes off a plural -s, and turns -ies into -y.
func stem(w string) string {
	switch {
	case len(w) > 4 && strings.HasSuffix(w, "ies"):
		return w[:len(w)-3] + "y"
	case len(w) > 3 && strings.HasSuffix(w, "s") && !strings.HasSuffix(w, "ss"):
		return w[:len(w)-1]
	}
	return w
}

// terms lower-cases s, splits it at every character that is not a letter or
// a digit, drops stop words and stems the rest.
func terms(s string) []string {
	var out []string
	for _, f := range strings.FieldsFunc(strings.ToLower(s), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) }) {
		if !stopWords[f] {
			out = append(out, stem(f))
		}
	}
	return out
}

// wordDoc is a skill as the ranked word search sees it: the name counts three
// times and the description once; the ID is not scored.
type wordDoc struct {
	tf  map[string]float64
	len float64
}

func newWordDoc(s *skill.Skill) wordDoc {
	d := wordDoc{tf: map[string]float64{}}
	for _, t := range terms(s.Name) {
		d.tf[t] += nameWeight
		d.len += nameWeight
	}
	for _, t := range terms(s.Description) {
		d.tf[t]++
		d.len++
	}
	return d
}

// Words is the words engine: the matching of names in three tiers, then a
// BM25 ranking of names and descriptions. It is a Searcher.
type Words struct {
	docs map[string]wordDoc // by skill ID
	df   map[string]int
	n    float64
	avg  float64
}

// NewWords builds the engine with the statistics of all, which should be the
// whole library.
func NewWords(all []*skill.Skill) *Words {
	w := &Words{docs: map[string]wordDoc{}, df: map[string]int{}, n: float64(len(all))}
	var total float64
	for _, s := range all {
		d := newWordDoc(s)
		w.docs[s.ID] = d
		for t := range d.tf {
			w.df[t]++
		}
		total += d.len
	}
	if len(all) > 0 {
		w.avg = total / w.n
	}
	return w
}

// score is the BM25 score of d for the query terms.
func (w *Words) score(d wordDoc, q []string) float64 {
	var s float64
	for _, t := range q {
		f := d.tf[t]
		if f == 0 || w.avg == 0 {
			continue
		}
		df := float64(w.df[t])
		idf := math.Log(1 + (w.n-df+0.5)/(df+0.5))
		s += idf * f * (bm25K1 + 1) / (f + bm25K1*(1-bm25B+bm25B*d.len/w.avg))
	}
	return s
}

// Search returns the skills of the set that match text. Exact names come
// first, then names that start with text, then names or IDs that contain it,
// ignoring case, hyphens and underscores; when the whole text is not found, a
// skill still matches in the third tier if every word of it is in the name or
// the ID. The rest are the skills whose BM25 score over name and description
// exceeds the cutoff, best first. Matches of one name tier are sorted by ID,
// ties of the score too. Empty text matches nothing.
func (w *Words) Search(_ context.Context, text string, skills []*skill.Skill) ([]Match, error) {
	q := fold(strings.TrimSpace(text))
	if q == "" {
		return nil, nil
	}
	words := strings.Fields(q)
	containsAll := func(s string) bool {
		for _, w := range words {
			if !strings.Contains(s, w) {
				return false
			}
		}
		return true
	}
	qt := terms(text)
	var out []Match
	for _, s := range skills {
		name, id := fold(s.Name), fold(s.ID)
		rank := -1
		switch {
		case name == q || id == q:
			rank = RankExactName
		case strings.HasPrefix(name, q):
			rank = RankNamePrefix
		case strings.Contains(name, q) || strings.Contains(id, q):
			rank = RankNameContains
		case containsAll(name) || containsAll(id):
			rank = RankNameContains
		}
		d, ok := w.docs[s.ID]
		if !ok {
			d = newWordDoc(s)
		}
		score := w.score(d, qt)
		if rank < 0 && score > descriptionCutoff {
			rank = RankDescription
		}
		if rank >= 0 {
			out = append(out, Match{Skill: s, Rank: rank, Score: score})
		}
	}
	slices.SortFunc(out, func(a, b Match) int {
		c := cmp.Compare(a.Rank, b.Rank)
		if c == 0 && a.Rank == RankDescription {
			c = cmp.Compare(b.Score, a.Score)
		}
		return cmp.Or(c, cmp.Compare(a.ID, b.ID))
	})
	return out, nil
}

// engineEnv is what an engine is built from.
type engineEnv struct {
	opts     Options
	settings EngineSetting
	words    *Words
}

// engines are the engines Quiver knows, by the name of the settings file. An
// engine is built from the whole library.
var engines = map[string]func(*Library, engineEnv) (Searcher, error){
	EngineWords: func(_ *Library, env engineEnv) (Searcher, error) { return env.words, nil },
	EngineLocal: newLocal,
}

// Search is the search of one run: the engine that was chosen and the words
// engine it falls back to.
type Search struct {
	// Engine is the engine in use.
	Engine string
	// Problem is what was wrong with the settings file, empty when nothing
	// was. The caller says it once.
	Problem string

	engine Searcher
	words  *Words
}

// NewSearch chooses the engine as NewSearchWith does, with the default
// options: a search builds a missing index for up to a minute.
func (l *Library) NewSearch(override string) (*Search, error) {
	return l.NewSearchWith(override, Options{})
}

// NewSearchWith chooses the engine: override when it is not empty, otherwise
// the one of the settings file. An unknown override is an error; a settings
// file that cannot be used, or an engine that cannot run, is a Problem, and
// the words engine runs.
func (l *Library) NewSearchWith(override string, opts Options) (*Search, error) {
	s := &Search{Engine: EngineWords, words: NewWords(l.All())}
	cur := LoadEngine()
	name := cur.Engine
	if override != "" {
		name = override
		if _, ok := engines[name]; !ok {
			return nil, unknownEngine(name)
		}
	} else {
		s.Problem = cur.Problem
	}
	s.Engine = name
	if name != EngineWords {
		var err error
		if s.engine, err = engines[name](l, engineEnv{opts: opts, settings: cur, words: s.words}); err != nil {
			s.Problem = joinProblems(s.Problem, fmt.Sprintf("search engine %s cannot run (%v); searching with words", name, err))
			s.Engine = EngineWords
		}
	}
	return s, nil
}

// Find searches the set with the engine. It never fails because of the engine:
// an engine that cannot run gives way to words, and note says so in one line.
// An engine may also have a line to say, such as an incomplete index.
func (s *Search) Find(ctx context.Context, query string, skills []*skill.Skill) (matches []Match, note string) {
	if s.engine != nil {
		out, err := s.engine.Search(ctx, query, skills)
		if err == nil {
			if n, ok := s.engine.(interface{ Note() string }); ok {
				note = n.Note()
			}
			return out, note
		}
		note = fmt.Sprintf("search engine %s cannot run (%v); searched with words", s.Engine, err)
	}
	out, _ := s.words.Search(ctx, query, skills)
	return out, note
}

func joinProblems(a, b string) string {
	if a == "" {
		return b
	}
	return a + "; " + b
}
