// Package embedtest makes tiny models with random weights for tests, so they
// need no download. It does not import embed, so the tests of embed can use it.
package embedtest

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
)

// Shape is the shape of a model, as embed.Config has it.
type Shape struct{ Vocab, Hidden, Layers, Heads, Inter, MaxPos int }

// Tiny is the shape of the model that the tests generate.
var Tiny = Shape{Vocab: 64, Hidden: 8, Layers: 2, Heads: 2, Inter: 16, MaxPos: 512}

// Tensors returns the tensors of a model of the shape cfg, by name.
func Tensors(cfg Shape) (names []string, shapes map[string][]int) {
	h := cfg.Hidden
	shapes = map[string][]int{}
	add := func(name string, shape ...int) {
		names = append(names, name)
		shapes[name] = shape
	}
	add("embeddings.word_embeddings.weight", cfg.Vocab, h)
	add("embeddings.position_embeddings.weight", cfg.MaxPos, h)
	add("embeddings.token_type_embeddings.weight", 2, h)
	add("embeddings.LayerNorm.weight", h)
	add("embeddings.LayerNorm.bias", h)
	for i := 0; i < cfg.Layers; i++ {
		p := fmt.Sprintf("encoder.layer.%d.", i)
		for _, n := range []string{"query", "key", "value"} {
			add(p+"attention.self."+n+".weight", h, h)
			add(p+"attention.self."+n+".bias", h)
		}
		add(p+"attention.output.dense.weight", h, h)
		add(p+"attention.output.dense.bias", h)
		add(p+"attention.output.LayerNorm.weight", h)
		add(p+"attention.output.LayerNorm.bias", h)
		add(p+"intermediate.dense.weight", cfg.Inter, h)
		add(p+"intermediate.dense.bias", cfg.Inter)
		add(p+"output.dense.weight", h, cfg.Inter)
		add(p+"output.dense.bias", h)
		add(p+"output.LayerNorm.weight", h)
		add(p+"output.LayerNorm.bias", h)
	}
	return names, shapes
}

// Safetensors encodes tensors in the file format. mutate may change a header
// entry before it is written.
func Safetensors(names []string, shapes map[string][]int, seed int64, mutate func(name string, ti map[string]any)) []byte {
	rng := rand.New(rand.NewSource(seed))
	hdr := map[string]any{"__metadata__": map[string]string{"format": "pt"}}
	var data []byte
	for _, n := range names {
		count := 1
		for _, d := range shapes[n] {
			count *= d
		}
		start := len(data)
		for i := 0; i < count; i++ {
			v := float32(rng.NormFloat64() * 0.3)
			data = binary.LittleEndian.AppendUint32(data, math.Float32bits(v))
		}
		ti := map[string]any{"dtype": "F32", "shape": shapes[n], "data_offsets": []int{start, len(data)}}
		if mutate != nil {
			mutate(n, ti)
		}
		hdr[n] = ti
	}
	hj, _ := json.Marshal(hdr)
	out := binary.LittleEndian.AppendUint64(nil, uint64(len(hj)))
	out = append(out, hj...)
	return append(out, data...)
}

// Tokenizer is a WordPiece vocabulary of the letters, the digits, their
// "##" pieces, a few punctuation marks and the special tokens.
func Tokenizer(extra map[string]int) []byte {
	vocab := map[string]int{"[PAD]": 0, "[UNK]": 1, "[CLS]": 2, "[SEP]": 3}
	for i, r := range "abcdefghijklmnopqrstuvwxyz0123456789" {
		vocab[string(r)] = 4 + i
		vocab["##"+string(r)] = 4 + 36 + i%20
	}
	vocab["."], vocab["-"] = 62, 63
	for k, v := range extra {
		vocab[k] = v
	}
	b, _ := json.Marshal(map[string]any{"model": map[string]any{
		"type": "WordPiece", "vocab": vocab, "unk_token": "[UNK]", "continuing_subword_prefix": "##", "max_input_chars_per_word": 100,
	}})
	return b
}

// WriteModel writes the tiny model into dir: model.safetensors and
// tokenizer.json. The weights come from seed, so two seeds are two models.
func WriteModel(t testing.TB, dir string, seed int64) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	names, shapes := Tensors(Tiny)
	Write(t, dir, "model.safetensors", Safetensors(names, shapes, seed, nil))
	Write(t, dir, "tokenizer.json", Tokenizer(nil))
}

// Write writes a file of a model folder.
func Write(t testing.TB, dir, name string, data []byte) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), data, 0o644); err != nil {
		t.Fatal(err)
	}
}
