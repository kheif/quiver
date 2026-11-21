// Throwaway prototype for TASK62.1: compares the matching of today, a ranked
// word search and embedding models on a labelled query set. Not product code.
package main

import (
	"cmp"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type query struct {
	Kind string   `json:"kind"`
	Text string   `json:"text"`
	Gold []string `json:"gold"`
}

type vecFile struct {
	Docs    [][]float32 `json:"docs"`
	Queries [][]float32 `json:"queries"`
	Ids     [][]int     `json:"ids"`
	Sample  []string    `json:"sample"`
}

func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func readJSON(path string, v any) {
	b, err := os.ReadFile(path)
	must(err)
	must(json.Unmarshal(b, v))
}

func docText(d skillDoc) string { return d.ID + " " + d.Name + ": " + d.Description }

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: discovery validate|eval|bench [flags]")
		os.Exit(2)
	}
	cmd := os.Args[1]
	if cmd == "cue" {
		if len(os.Args) > 2 && os.Args[2] == "2" {
			fmt.Println(searchableCue2)
			return
		}
		fmt.Println(searchableCue)
		return
	}
	if cmd == "embedlib" {
		runEmbedLib(os.Args[2:])
		return
	}
	if cmd == "serve" {
		runServe(os.Args[2:])
		return
	}
	fs := flag.NewFlagSet(cmd, flag.ExitOnError)
	corpus := fs.String("corpus", "corpus.json", "skill library")
	queries := fs.String("queries", "queries.json", "labelled queries")
	models := fs.String("models", "models", "directory with the model folders")
	vecs := fs.String("vecs", "", "directory with vec_<name>.json files from a runtime (docs and queries)")
	ref := fs.String("ref", "", "directory with ref_<name>.json files from the Python package")
	mode := fs.String("mode", "mmap", "mmap or read")
	miss := fs.String("miss", "", "print the misses of this engine")
	lang := fs.String("lang", "all", "all, or en to leave out the Turkish queries")
	fs.Parse(os.Args[2:])

	var docs []skillDoc
	var raw []struct{ ID, Name, Description string }
	readJSON(*corpus, &raw)
	for _, r := range raw {
		docs = append(docs, skillDoc{r.ID, r.Name, r.Description})
	}
	var qs []query
	readJSON(*queries, &qs)

	switch cmd {
	case "validate":
		validate(docs, qs, *models, *ref)
	case "vcheck":
		vcheck(docs, qs, fs.Arg(0), fs.Arg(1))
	case "bench":
		bench(docs, qs, fs.Arg(0), *mode == "mmap")
	case "eval":
		eval(docs, qs, *models, *vecs, *miss, *lang == "en")
	}
}

// ---- validate: the Go loader against the Python package ----

func validate(docs []skillDoc, qs []query, models, ref string) {
	for _, name := range []string{"potion-base-8M", "potion-base-32M", "potion-multilingual-128M"} {
		s, err := LoadStatic(filepath.Join(models, name), true)
		must(err)
		var r vecFile
		readJSON(filepath.Join(ref, "ref_"+name+".json"), &r)
		idSame := 0
		for i, t := range r.Sample {
			var got []int
			for _, id := range s.tok.Encode(t) {
				got = append(got, id)
			}
			want := r.Ids[i]
			if slices.Equal(got, want) {
				idSame++
			} else if idSame < 1000 && os.Getenv("SHOWDIFF") != "" {
				fmt.Printf("  ids differ for %q\n    go  %v\n    py  %v\n", t, got, want)
			}
		}
		var sum, min float64 = 0, 2
		n := 0
		check := func(text string, want []float32) {
			c := cosine(s.Embed(text), want)
			sum += c
			if c < min {
				min = c
			}
			n++
		}
		for i, d := range docs {
			check(docText(d), r.Docs[i])
		}
		for i, q := range qs {
			check(q.Text, r.Queries[i])
		}
		fmt.Printf("%-26s dim %3d  token ids equal for %d/%d texts  vector cosine to Python: mean %.5f min %.5f (%d texts)\n",
			name, s.Dim, idSame, len(r.Sample), sum/float64(n), min, n)
		s.Close()
	}
}

// ---- bench: one model in one fresh process ----

func rssMB() float64 {
	out, err := exec.Command("ps", "-o", "rss=", "-p", strconv.Itoa(os.Getpid())).Output()
	if err != nil {
		return 0
	}
	kb, _ := strconv.ParseFloat(strings.TrimSpace(string(out)), 64)
	return kb / 1024
}

func bench(docs []skillDoc, qs []query, dir string, mmap bool) {
	r0 := rssMB()
	t0 := time.Now()
	s, err := LoadAny(dir, mmap)
	must(err)
	load := time.Since(t0)
	debug.FreeOSMemory() // give back what parsing the tokenizer file needed once
	r1 := rssMB()
	t1 := time.Now()
	for _, d := range docs {
		s.EmbedDoc(docText(d))
	}
	lib := time.Since(t1)
	r2 := rssMB()
	t2 := time.Now()
	n := 0
	for time.Since(t2) < 1500*time.Millisecond {
		for _, q := range qs[:30] {
			s.EmbedQuery(q.Text)
			n++
		}
	}
	perQ := time.Since(t2) / time.Duration(n)
	var ru syscall.Rusage
	syscall.Getrusage(syscall.RUSAGE_SELF, &ru)
	fmt.Printf("%-26s %-5s load %7.1f ms | rss start %5.1f -> loaded %6.1f -> library embedded %6.1f MB | peak %6.1f MB | library %d skills %8.1f ms | query %9.1f us\n",
		filepath.Base(dir), map[bool]string{true: "mmap", false: "read"}[mmap], float64(load.Microseconds())/1000, r0, r1, r2,
		float64(ru.Maxrss)/1048576, len(docs), float64(lib.Microseconds())/1000, float64(perQ.Nanoseconds())/1000)
}

// vcheck compares the vectors of a model with the vectors a runtime made of the same texts.
func vcheck(docs []skillDoc, qs []query, dir, file string) {
	s, err := LoadAny(dir, true)
	must(err)
	var v vecFile
	readJSON(file, &v)
	var sum float64
	min := 2.0
	n := 0
	for i, d := range docs {
		c := cosine(s.EmbedDoc(docText(d)), v.Docs[i])
		sum += c
		if c < min {
			min = c
		}
		n++
	}
	for i, q := range qs {
		c := cosine(s.EmbedQuery(q.Text), v.Queries[i])
		sum += c
		if c < min {
			min = c
		}
		n++
	}
	fmt.Printf("%s against %s: cosine mean %.5f min %.5f over %d texts\n", filepath.Base(dir), filepath.Base(file), sum/float64(n), min, n)
}

// ---- eval ----

type engine struct {
	name   string
	raw    [][]int // ranked documents per query, before any cutoff
	final  [][]int // after the cutoff
	cutoff string
}

func topN(h []hit, n int) []int {
	var out []int
	for _, x := range h {
		out = append(out, x.doc)
		if len(out) == n {
			break
		}
	}
	return out
}

func ids(h []hit) []int { return topN(h, len(h)) }

func isTurkish(s string) bool { return strings.ContainsAny(s, "çğıöşüÇĞİÖŞÜ") }

func isNofit(k string) bool { return strings.HasPrefix(k, "nofit") }

var relax = func() int { n, _ := strconv.Atoi(os.Getenv("RELAX")); return n }()

// skip marks the queries that a run leaves out (the Turkish ones with -lang en).
var skip []bool

// half is 0 for the calibration half and 1 for the test half: the split
// alternates down the file within each kind, in the order the queries were
// written, counting only the queries that are not left out.
func half(qs []query, i int) int {
	n := 0
	for j := 0; j < i; j++ {
		if qs[j].Kind == qs[i].Kind && !skip[j] {
			n++
		}
	}
	return n % 2
}

// calibrated builds an engine from scored hits. The cutoff is the highest top
// score of the queries that fit nothing in the calibration half, so that none
// of them returns anything.
func calibrated(name string, qs []query, hits [][]hit) engine {
	var tops []float64
	for i, q := range qs {
		if isNofit(q.Kind) && !skip[i] && half(qs, i) == 0 {
			if len(hits[i]) > 0 {
				tops = append(tops, hits[i][0].score)
			} else {
				tops = append(tops, -1e18)
			}
		}
	}
	slices.SortFunc(tops, func(a, b float64) int { return cmp.Compare(b, a) })
	cut := -1e18
	if len(tops) > relax {
		cut = tops[relax]
	}
	e := engine{name: name, cutoff: fmt.Sprintf("%.3f", cut)}
	for i := range qs {
		e.raw = append(e.raw, ids(hits[i]))
		var f []int
		for _, h := range hits[i] {
			if h.score > cut {
				f = append(f, h.doc)
			}
		}
		e.final = append(e.final, f)
	}
	return e
}

func fuse(name string, qs []query, a, b engine) engine {
	e := engine{name: name, cutoff: "fused"}
	for i := range qs {
		score := map[int]float64{}
		for _, list := range [][]int{a.final[i], b.final[i]} {
			for r, d := range list {
				score[d] += 1 / float64(60+r+1)
			}
		}
		var h []hit
		for d, s := range score {
			h = append(h, hit{d, s})
		}
		slices.SortFunc(h, func(x, y hit) int { return cmp.Or(cmp.Compare(y.score, x.score), cmp.Compare(x.doc, y.doc)) })
		e.raw = append(e.raw, ids(h))
		e.final = append(e.final, ids(h))
	}
	return e
}

func hitsFrom(ids []int) []hit {
	var h []hit
	for _, d := range ids {
		h = append(h, hit{doc: d})
	}
	return h
}

func eval(docs []skillDoc, qs []query, models, vecs, miss string, englishOnly bool) {
	skip = make([]bool, len(qs))
	if englishOnly {
		for i, q := range qs {
			skip[i] = q.Kind == "paraphrase_tr" || isTurkish(q.Text)
		}
	}
	goldIdx := func(q query) map[int]bool {
		m := map[int]bool{}
		for _, g := range q.Gold {
			for i, d := range docs {
				if d.ID == g {
					m[i] = true
				}
			}
		}
		return m
	}
	var engines []engine

	// the matching of today: no score, so no cutoff
	today := engine{name: "today", cutoff: "-"}
	for _, q := range qs {
		h := ids(searchToday(docs, q.Text))
		today.raw = append(today.raw, h)
		today.final = append(today.final, h)
	}
	engines = append(engines, today)

	bm := newBM25(docs)
	var bh [][]hit
	for _, q := range qs {
		bh = append(bh, bm.Rank(q.Text))
	}
	bm25 := calibrated("bm25", qs, bh)
	engines = append(engines, bm25)

	vectorEngine := func(name string, dv, qv [][]float32) engine {
		var hh [][]hit
		for i := range qs {
			var h []hit
			for d := range docs {
				h = append(h, hit{d, cosine(qv[i], dv[d])})
			}
			slices.SortFunc(h, func(a, b hit) int { return cmp.Or(cmp.Compare(b.score, a.score), cmp.Compare(a.doc, b.doc)) })
			hh = append(hh, h)
		}
		return calibrated(name, qs, hh)
	}
	var vec []engine
	dirs, _ := filepath.Glob(filepath.Join(models, "*"))
	slices.Sort(dirs)
	for _, dir := range dirs {
		if _, err := os.Stat(filepath.Join(dir, "model.safetensors")); err != nil {
			continue
		}
		s, err := LoadAny(dir, true)
		must(err)
		var dv, qv [][]float32
		for _, d := range docs {
			dv = append(dv, s.EmbedDoc(docText(d)))
		}
		for _, q := range qs {
			qv = append(qv, s.EmbedQuery(q.Text))
		}
		vec = append(vec, vectorEngine(filepath.Base(dir), dv, qv))
		s.Close()
	}
	if vecs != "" {
		files, _ := filepath.Glob(filepath.Join(vecs, "vec_*.json"))
		slices.Sort(files)
		for _, f := range files {
			var v vecFile
			readJSON(f, &v)
			name := strings.TrimSuffix(strings.TrimPrefix(filepath.Base(f), "vec_"), ".json")
			if len(v.Queries) != len(qs) {
				fmt.Fprintf(os.Stderr, "skipping %s: %d query vectors for %d queries\n", name, len(v.Queries), len(qs))
				continue
			}
			vec = append(vec, vectorEngine(name, v.Docs, v.Queries))
		}
	}
	engines = append(engines, vec...)
	for _, v := range vec {
		engines = append(engines, fuse("bm25+"+v.name, qs, bm25, v))
	}

	// kinds present, in file order
	var kinds []string
	for i, q := range qs {
		if !skip[i] && !slices.Contains(kinds, q.Kind) {
			kinds = append(kinds, q.Kind)
		}
	}
	type cnt struct{ ok, n, rawOK int }
	scoreEngine := func(e engine, h int) (map[string]*cnt, *cnt, *cnt) {
		per := map[string]*cnt{}
		paraphrase, fit := &cnt{}, &cnt{}
		for _, k := range kinds {
			per[k] = &cnt{}
		}
		for i, q := range qs {
			if skip[i] || half(qs, i) != h {
				continue
			}
			x := per[q.Kind]
			x.n++
			if isNofit(q.Kind) {
				if len(e.final[i]) == 0 {
					x.ok++
				}
				continue
			}
			g := goldIdx(q)
			found, rawFound := false, false
			for _, d := range topN(hitsFrom(e.final[i]), 5) {
				found = found || g[d]
			}
			for _, d := range topN(hitsFrom(e.raw[i]), 5) {
				rawFound = rawFound || g[d]
			}
			if found {
				x.ok++
			}
			if rawFound {
				x.rawOK++
			}
			fit.n++
			if found {
				fit.ok++
			}
			if rawFound {
				fit.rawOK++
			}
			if strings.HasPrefix(q.Kind, "paraphrase") {
				paraphrase.n++
				if found {
					paraphrase.ok++
				}
				if rawFound {
					paraphrase.rawOK++
				}
			}
			if miss == e.name && !found {
				var top []string
				for _, d := range topN(hitsFrom(e.final[i]), 3) {
					top = append(top, docs[d].ID)
				}
				fmt.Printf("  MISS [%s/%d] %q want %v got %v\n", q.Kind, h, q.Text, q.Gold, top)
			}
		}
		return per, paraphrase, fit
	}

	for h, title := range []string{"calibration half (the cutoff was fitted here)", "TEST HALF"} {
		if h == 0 && miss == "" {
			continue
		}
		fmt.Printf("\n%s, k of n (queries left out: %d)\n", title, countTrue(skip))
		fmt.Printf("%-30s", "engine")
		for _, k := range kinds {
			fmt.Printf(" | %-13s", k)
		}
		fmt.Printf(" | %-12s | %-12s | %-9s | %s\n", "paraphrases", "all fitting", "no cutoff", "cutoff")
		for _, e := range engines {
			per, para, fit := scoreEngine(e, h)
			fmt.Printf("%-30s", e.name)
			for _, k := range kinds {
				fmt.Printf(" | %2d/%-2d %4.0f%%  ", per[k].ok, per[k].n, pct(per[k].ok, per[k].n))
			}
			fmt.Printf(" | %2d/%-2d %4.0f%% | %2d/%-2d %4.0f%% | %3.0f%%      | %s\n", para.ok, para.n, pct(para.ok, para.n), fit.ok, fit.n, pct(fit.ok, fit.n), pct(fit.rawOK, fit.n), e.cutoff)
		}
	}
}

func pct(a, b int) float64 {
	if b == 0 {
		return 0
	}
	return 100 * float64(a) / float64(b)
}

func countTrue(b []bool) int {
	n := 0
	for _, x := range b {
		if x {
			n++
		}
	}
	return n
}
