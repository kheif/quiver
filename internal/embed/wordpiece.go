package embed

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

const maxTokenizerFile = 16 << 20

// wordPiece is the tokenizer of BERT models that lower-case: it cleans the
// text, splits it at spaces and punctuation, strips accents and cuts every
// word into the longest pieces of the vocabulary.
type wordPiece struct {
	vocab    map[string]int
	unk      int
	maxChars int
	prefix   string
}

func loadWordPiece(path string) (*wordPiece, error) {
	st, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if st.Size() > maxTokenizerFile {
		return nil, fmt.Errorf("tokenizer file: size %d is outside what a model of this kind has", st.Size())
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var tk struct {
		Model struct {
			Type     string         `json:"type"`
			Vocab    map[string]int `json:"vocab"`
			Unk      string         `json:"unk_token"`
			Prefix   string         `json:"continuing_subword_prefix"`
			MaxChars int            `json:"max_input_chars_per_word"`
		} `json:"model"`
	}
	if err := json.Unmarshal(data, &tk); err != nil {
		return nil, fmt.Errorf("tokenizer file: %w", err)
	}
	m := tk.Model
	if m.Type != "WordPiece" || len(m.Vocab) == 0 {
		return nil, fmt.Errorf("tokenizer file: not a WordPiece vocabulary")
	}
	unk, ok := m.Vocab[m.Unk]
	if !ok {
		return nil, fmt.Errorf("tokenizer file: unknown token %q is not in the vocabulary", m.Unk)
	}
	if m.MaxChars <= 0 {
		m.MaxChars = 100
	}
	return &wordPiece{vocab: m.Vocab, unk: unk, maxChars: m.MaxChars, prefix: m.Prefix}, nil
}

func isCJK(r rune) bool {
	return r >= 0x4E00 && r <= 0x9FFF || r >= 0x3400 && r <= 0x4DBF || r >= 0x20000 && r <= 0x2A6DF || r >= 0xF900 && r <= 0xFAFF
}

// normalize cleans control characters, puts spaces around Chinese characters,
// strips accents (which needs Unicode decomposition) and lower-cases.
func normalize(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r == 0 || r == 0xFFFD:
		case r == '\t' || r == '\n' || r == '\r' || unicode.IsSpace(r):
			b.WriteRune(' ')
		case unicode.IsControl(r) || unicode.Is(unicode.Cf, r):
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

func isPunct(r rune) bool {
	return r < 128 && (r >= 33 && r <= 47 || r >= 58 && r <= 64 || r >= 91 && r <= 96 || r >= 123 && r <= 126) || unicode.IsPunct(r)
}

// encode returns the ids of the pieces of text, without [CLS] and [SEP].
func (w *wordPiece) encode(text string) []int {
	var words []string
	var cur []rune
	flush := func() {
		if len(cur) > 0 {
			words = append(words, string(cur))
			cur = cur[:0]
		}
	}
	for _, r := range normalize(text) {
		switch {
		case unicode.IsSpace(r):
			flush()
		case isPunct(r):
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
		var pieces []int
		start, ok := 0, true
		for start < len(rs) {
			end, found := len(rs), -1
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
			pieces = append(pieces, found)
			start = end
		}
		if ok {
			ids = append(ids, pieces...)
		} else {
			ids = append(ids, w.unk)
		}
	}
	return ids
}
