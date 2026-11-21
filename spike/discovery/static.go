package main

// A static embedding model (Model2Vec): a tokenizer, a table with one vector per
// token, mean pooling and an L2 normalisation. No neural network runs, so the
// whole model is one weight file and a loop. Pure Go, no cgo.

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"syscall"
	"unsafe"
)

type Tokenizer interface{ Encode(text string) []int }

type Static struct {
	Dim, Rows int
	emb       []float32
	raw       []byte
	tok       Tokenizer
	unk       int
}

// LoadStatic reads dir/model.safetensors and dir/tokenizer.json. With mmap the
// weight file is mapped and only the rows that are used become resident.
func LoadStatic(dir string, mmap bool) (*Static, error) {
	path := filepath.Join(dir, "model.safetensors")
	var raw []byte
	if mmap {
		f, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		st, _ := f.Stat()
		raw, err = syscall.Mmap(int(f.Fd()), 0, int(st.Size()), syscall.PROT_READ, syscall.MAP_SHARED)
		f.Close()
		if err != nil {
			return nil, err
		}
	} else {
		var err error
		if raw, err = os.ReadFile(path); err != nil {
			return nil, err
		}
	}
	n := int(binary.LittleEndian.Uint64(raw[:8]))
	var hdr map[string]json.RawMessage
	if err := json.Unmarshal(raw[8:8+n], &hdr); err != nil {
		return nil, err
	}
	var t struct {
		Dtype   string `json:"dtype"`
		Shape   []int  `json:"shape"`
		Offsets [2]int `json:"data_offsets"`
	}
	if err := json.Unmarshal(hdr["embeddings"], &t); err != nil {
		return nil, fmt.Errorf("no embeddings tensor: %w", err)
	}
	if t.Dtype != "F32" || len(t.Shape) != 2 {
		return nil, fmt.Errorf("unsupported tensor %s %v", t.Dtype, t.Shape)
	}
	data := raw[8+n+t.Offsets[0] : 8+n+t.Offsets[1]]
	s := &Static{Rows: t.Shape[0], Dim: t.Shape[1], raw: raw, unk: 1}
	s.emb = unsafe.Slice((*float32)(unsafe.Pointer(&data[0])), len(data)/4)

	tj, err := os.ReadFile(filepath.Join(dir, "tokenizer.json"))
	if err != nil {
		return nil, err
	}
	var tk struct {
		Normalizer json.RawMessage `json:"normalizer"`
		Model      json.RawMessage `json:"model"`
	}
	if err := json.Unmarshal(tj, &tk); err != nil {
		return nil, err
	}
	var kind struct {
		Type string `json:"type"`
	}
	json.Unmarshal(tk.Model, &kind)
	switch kind.Type {
	case "WordPiece":
		s.tok, err = newWordPiece(tk.Model)
	case "Unigram":
		s.tok, err = newUnigram(tk.Model)
	default:
		err = fmt.Errorf("tokenizer %q not supported", kind.Type)
	}
	return s, err
}

func (s *Static) Close() {
	if s.raw != nil {
		syscall.Munmap(s.raw)
	}
}

// Embed returns the normalised mean of the vectors of the text's tokens.
func (s *Static) Embed(text string) []float32 {
	out := make([]float32, s.Dim)
	n := 0
	for _, id := range s.tok.Encode(text) {
		if id == s.unk || id < 0 || id >= s.Rows {
			continue
		}
		row := s.emb[id*s.Dim : (id+1)*s.Dim]
		for i, v := range row {
			out[i] += v
		}
		n++
	}
	if n == 0 {
		return out
	}
	var sum float64
	for _, v := range out {
		sum += float64(v) * float64(v)
	}
	if sum == 0 {
		return out
	}
	inv := float32(1 / math.Sqrt(sum))
	for i := range out {
		out[i] *= inv
	}
	return out
}

func cosine(a, b []float32) float64 {
	var dot, na, nb float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		na += float64(a[i]) * float64(a[i])
		nb += float64(b[i]) * float64(b[i])
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / math.Sqrt(na*nb)
}
