package main

import (
	"cmp"
	"math"
	"slices"
	"strings"
	"unicode"
)

type skillDoc struct{ ID, Name, Description string }

type hit struct {
	doc   int
	score float64
}

// ---- the matching of today, ported from internal/library/search.go ----

func fold(s string) string {
	return strings.NewReplacer("-", " ", "_", " ").Replace(strings.ToLower(s))
}

// searchToday returns matches ordered by tier, then ID. The score is minus the
// tier so that a larger score is better.
func searchToday(docs []skillDoc, text string) []hit {
	q := fold(strings.TrimSpace(text))
	if q == "" {
		return nil
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
	type m struct {
		doc  int
		rank int
	}
	var out []m
	for i, s := range docs {
		name, id, desc := fold(s.Name), fold(s.ID), fold(s.Description)
		rank := -1
		switch {
		case name == q || id == q:
			rank = 0
		case strings.HasPrefix(name, q):
			rank = 1
		case strings.Contains(name, q) || strings.Contains(id, q):
			rank = 2
		case strings.Contains(desc, q):
			rank = 3
		case containsAll(name) || containsAll(id):
			rank = 2
		case containsAll(name + " " + id + " " + desc):
			rank = 3
		}
		if rank >= 0 {
			out = append(out, m{i, rank})
		}
	}
	slices.SortFunc(out, func(a, b m) int {
		return cmp.Or(cmp.Compare(a.rank, b.rank), cmp.Compare(docs[a.doc].ID, docs[b.doc].ID))
	})
	var hits []hit
	for _, x := range out {
		hits = append(hits, hit{x.doc, -float64(x.rank)})
	}
	return hits
}

// ---- ranked word search: BM25 over name and description ----

var stop = map[string]bool{}

func init() {
	for _, w := range strings.Fields("a an the of to for and or in on at by with from my our your is are was be it its that this these those what which who how do does did i we you they me us can could should would will about into so as if than then not no up out over per via use used using when where why") {
		stop[w] = true
	}
}

func stem(w string) string {
	switch {
	case len(w) > 4 && strings.HasSuffix(w, "ies"):
		return w[:len(w)-3] + "y"
	case len(w) > 3 && strings.HasSuffix(w, "s") && !strings.HasSuffix(w, "ss"):
		return w[:len(w)-1]
	}
	return w
}

func terms(s string) []string {
	var out []string
	for _, f := range strings.FieldsFunc(strings.ToLower(s), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) }) {
		if !stop[f] {
			out = append(out, stem(f))
		}
	}
	return out
}

type BM25 struct {
	docs  []skillDoc
	tf    []map[string]float64
	len   []float64
	df    map[string]int
	avg   float64
	names []string
}

const nameWeight = 3.0

func newBM25(docs []skillDoc) *BM25 {
	b := &BM25{docs: docs, df: map[string]int{}}
	var total float64
	for _, d := range docs {
		tf := map[string]float64{}
		var l float64
		for _, t := range terms(d.Name) {
			tf[t] += nameWeight
			l += nameWeight
		}
		for _, t := range terms(d.Description) {
			tf[t]++
			l++
		}
		for t := range tf {
			b.df[t]++
		}
		b.tf = append(b.tf, tf)
		b.len = append(b.len, l)
		b.names = append(b.names, strings.Join(strings.Fields(fold(d.Name)), " "))
		total += l
	}
	b.avg = total / float64(len(docs))
	return b
}

// Rank returns the documents with a score above zero, best first. An exact name
// and a name prefix are lifted above every BM25 score.
func (b *BM25) Rank(text string) []hit {
	const k1, bb = 1.2, 0.75
	q := terms(text)
	nq := strings.Join(strings.Fields(fold(strings.TrimSpace(text))), " ")
	var out []hit
	for i := range b.docs {
		var s float64
		for _, t := range q {
			f := b.tf[i][t]
			if f == 0 {
				continue
			}
			n := float64(len(b.docs))
			idf := math.Log(1 + (n-float64(b.df[t])+0.5)/(float64(b.df[t])+0.5))
			s += idf * f * (k1 + 1) / (f + k1*(1-bb+bb*b.len[i]/b.avg))
		}
		switch {
		case nq != "" && b.names[i] == nq:
			s += 1000
		case nq != "" && strings.HasPrefix(b.names[i], nq):
			s += 500
		}
		if s > 0 {
			out = append(out, hit{i, s})
		}
	}
	slices.SortFunc(out, func(a, c hit) int {
		return cmp.Or(cmp.Compare(c.score, a.score), cmp.Compare(b.docs[a.doc].ID, b.docs[c.doc].ID))
	})
	return out
}
