package main

import (
	"encoding/json"
	"slices"
	"sort"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// ---- WordPiece (BERT uncased), as in potion-base-8M and potion-base-32M ----

type WordPiece struct {
	vocab    map[string]int
	unk      int
	maxChars int
	prefix   string
}

func newWordPiece(model json.RawMessage) (*WordPiece, error) {
	var m struct {
		Vocab    map[string]int `json:"vocab"`
		Unk      string         `json:"unk_token"`
		Prefix   string         `json:"continuing_subword_prefix"`
		MaxChars int            `json:"max_input_chars_per_word"`
	}
	if err := json.Unmarshal(model, &m); err != nil {
		return nil, err
	}
	return &WordPiece{vocab: m.Vocab, unk: m.Vocab[m.Unk], maxChars: m.MaxChars, prefix: m.Prefix}, nil
}

func isCJK(r rune) bool {
	return unicode.Is(unicode.Han, r) && (r >= 0x4E00 && r <= 0x9FFF || r >= 0x3400 && r <= 0x4DBF ||
		r >= 0x20000 && r <= 0x2A6DF || r >= 0xF900 && r <= 0xFAFF)
}

func bertNormalize(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r == 0 || r == 0xFFFD:
			continue
		case r == '\t' || r == '\n' || r == '\r' || unicode.IsSpace(r):
			b.WriteRune(' ')
		case unicode.IsControl(r) || unicode.Is(unicode.Cf, r):
			continue
		case isCJK(r):
			b.WriteString(" " + string(r) + " ")
		default:
			b.WriteRune(r)
		}
	}
	var c strings.Builder
	for _, r := range norm.NFD.String(b.String()) {
		if !unicode.Is(unicode.Mn, r) {
			c.WriteRune(r)
		}
	}
	return strings.ToLower(c.String())
}

func isBertPunct(r rune) bool {
	return r < 128 && (r >= 33 && r <= 47 || r >= 58 && r <= 64 || r >= 91 && r <= 96 || r >= 123 && r <= 126) || unicode.IsPunct(r)
}

func (w *WordPiece) Encode(text string) []int {
	var words []string
	var cur []rune
	flush := func() {
		if len(cur) > 0 {
			words = append(words, string(cur))
			cur = cur[:0]
		}
	}
	for _, r := range bertNormalize(text) {
		switch {
		case unicode.IsSpace(r):
			flush()
		case isBertPunct(r):
			flush()
			words = append(words, string(r))
		default:
			cur = append(cur, r)
		}
	}
	flush()
	var ids []int
	for _, word := range words {
		rs := []rune(word)
		if len(rs) > w.maxChars {
			ids = append(ids, w.unk)
			continue
		}
		var piece []int
		start := 0
		ok := true
		for start < len(rs) {
			end := len(rs)
			found := -1
			for end > start {
				sub := string(rs[start:end])
				if start > 0 {
					sub = w.prefix + sub
				}
				if id, hit := w.vocab[sub]; hit {
					found = id
					break
				}
				end--
			}
			if found < 0 {
				ok = false
				break
			}
			piece = append(piece, found)
			start = end
		}
		if ok {
			ids = append(ids, piece...)
		} else {
			ids = append(ids, w.unk)
		}
	}
	return ids
}

// ---- Unigram + Metaspace (XLM-R / BGE-M3), as in potion-multilingual-128M ----
//
// The tokenizer's own normaliser is a precompiled SentencePiece character map.
// This prototype approximates it with NFKC; the validation against the Python
// tokenizer says how many texts come out with different ids.

type Unigram struct {
	pieces   []string // sorted, so that all pieces with one prefix are next to each other
	pid      []int32
	scores   []float64
	unk      int
	unkScore float64
}

func newUnigram(model json.RawMessage) (*Unigram, error) {
	var m struct {
		Unk   int             `json:"unk_id"`
		Vocab [][]interface{} `json:"vocab"`
	}
	if err := json.Unmarshal(model, &m); err != nil {
		return nil, err
	}
	u := &Unigram{scores: make([]float64, len(m.Vocab)), unk: m.Unk}
	order := make([]int32, len(m.Vocab))
	names := make([]string, len(m.Vocab))
	min := 0.0
	for i, e := range m.Vocab {
		names[i] = e[0].(string)
		sc := e[1].(float64)
		u.scores[i] = sc
		if sc < min {
			min = sc
		}
		order[i] = int32(i)
	}
	slices.SortFunc(order, func(a, b int32) int { return strings.Compare(names[a], names[b]) })
	u.pieces = make([]string, len(order))
	u.pid = order
	for i, id := range order {
		u.pieces[i] = names[id]
	}
	u.unkScore = min - 10
	return u, nil
}

func uniNormalize(s string) string {
	s = norm.NFKC.String(s)
	var b strings.Builder
	for _, r := range s {
		switch {
		case r == '\t' || r == '\n' || r == '\r':
			b.WriteRune(' ')
		case unicode.IsControl(r) || unicode.Is(unicode.Cf, r):
			continue
		case r < 128 && isBertPunct(r):
			b.WriteString(" " + string(r) + " ")
		default:
			b.WriteRune(r)
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

func (u *Unigram) Encode(text string) []int {
	s := uniNormalize(text)
	if s == "" {
		return nil
	}
	rs := []rune("▁" + strings.ReplaceAll(s, " ", "▁"))
	n := len(rs)
	best := make([]float64, n+1)
	from := make([]int, n+1)
	piece := make([]int, n+1)
	for i := 1; i <= n; i++ {
		best[i] = -1e18
	}
	for i := 0; i < n; i++ {
		if best[i] <= -1e17 {
			continue
		}
		single := false
		for l := 1; i+l <= n; l++ {
			p := string(rs[i : i+l])
			lo := sort.SearchStrings(u.pieces, p)
			if lo == len(u.pieces) || !strings.HasPrefix(u.pieces[lo], p) {
				break
			}
			if u.pieces[lo] != p {
				continue
			}
			id := int(u.pid[lo])
			if l == 1 {
				single = true
			}
			if c := best[i] + u.scores[id]; c > best[i+l] {
				best[i+l], from[i+l], piece[i+l] = c, i, id
			}
		}
		if !single {
			if c := best[i] + u.unkScore; c > best[i+1] {
				best[i+1], from[i+1], piece[i+1] = c, i, u.unk
			}
		}
	}
	var out []int
	for i := n; i > 0; i = from[i] {
		out = append(out, piece[i])
	}
	for l, r := 0, len(out)-1; l < r; l, r = l+1, r-1 {
		out[l], out[r] = out[r], out[l]
	}
	return out
}
