package library

import (
	"context"
	"encoding/json"
	"os"
	"slices"
	"strings"
	"testing"
	"unicode"

	"github.com/kheif/quiver/internal/skill"
)

// The queries and the library of TASK62.1 (spike/discovery). The library is
// the author's own set of third-party skills and is not in the repository;
// corpus.py writes it to corpus.json from `quiver list --json`. Without it
// the test skips.
const spikeDir = "../../spike/discovery/"

type spikeQuery struct {
	Kind string   `json:"kind"`
	Text string   `json:"text"`
	Gold []string `json:"gold"`
}

func loadSpike(t *testing.T) ([]*skill.Skill, []spikeQuery) {
	t.Helper()
	data, err := os.ReadFile(spikeDir + "corpus.json")
	if err != nil {
		t.Skip("spike/discovery/corpus.json is not there: run corpus.py to make it")
	}
	var docs []struct{ ID, Name, Description string }
	if err := json.Unmarshal(data, &docs); err != nil {
		t.Fatal(err)
	}
	var all []*skill.Skill
	for _, d := range docs {
		all = append(all, &skill.Skill{ID: d.ID, Name: d.Name, Description: d.Description})
	}
	data, err = os.ReadFile(spikeDir + "queries.json")
	if err != nil {
		t.Fatal(err)
	}
	var qs []spikeQuery
	if err := json.Unmarshal(data, &qs); err != nil {
		t.Fatal(err)
	}
	return all, qs
}

// englishHalves keeps the English queries and splits each kind down the file:
// even-numbered ones calibrate, odd-numbered ones test, as in the spike.
func englishHalves(qs []spikeQuery) (calibration, test []spikeQuery) {
	n := map[string]int{}
	for _, q := range qs {
		if q.Kind == "paraphrase_tr" || strings.IndexFunc(q.Text, func(r rune) bool { return unicode.IsLetter(r) && r > unicode.MaxASCII }) >= 0 {
			continue
		}
		if n[q.Kind]%2 == 0 {
			calibration = append(calibration, q)
		} else {
			test = append(test, q)
		}
		n[q.Kind]++
	}
	return
}

func fitsNothing(q spikeQuery) bool { return strings.HasPrefix(q.Kind, "nofit") }

// topScore is the best BM25 score any skill gets for the query.
func topScore(w *Words, all []*skill.Skill, query string) float64 {
	var top float64
	qt := terms(query)
	for _, s := range all {
		top = max(top, w.score(newWordDoc(s), qt))
	}
	return top
}

// The cutoff keeps the queries that fit nothing from returning anything: on
// the whole library it is within reach of the value that the queries of the
// calibration half give, and the test half returns the rates of the spike.
func TestSpikeQueriesOnTheWholeLibrary(t *testing.T) {
	all, qs := loadSpike(t)
	w := NewWords(all)
	calibration, test := englishHalves(qs)

	var fitted float64
	for _, q := range calibration {
		if fitsNothing(q) {
			fitted = max(fitted, topScore(w, all, q.Text))
		}
	}
	t.Logf("the cutoff that the calibration queries give on %d skills: %.5f (in use: %.3f)", len(all), fitted, descriptionCutoff)
	if fitted > descriptionCutoff {
		t.Errorf("a calibration query that fits nothing scores %.3f, above the cutoff %.3f: fit the cutoff again", fitted, descriptionCutoff)
	}

	var fit, hit, nofit, empty int
	for _, q := range test {
		got, _ := w.Search(context.Background(), q.Text, all)
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
	t.Logf("test half: %d of %d fitting queries have a fitting skill in the first five; %d of %d queries that fit nothing return nothing", hit, fit, empty, nofit)
	// The spike measured 44 of 60 and 26 of 27 with the matching of the
	// names added; the rates must not fall below what it measured.
	if rate := float64(hit) / float64(fit); rate < 0.70 {
		t.Errorf("top-5 rate %.0f%%, the spike measured 73%%", rate*100)
	}
	if empty < nofit-2 {
		t.Errorf("%d of %d queries that fit nothing return nothing, the spike measured 26 of 27", empty, nofit)
	}
}
