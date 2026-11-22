// Package embed runs a small sentence-embedding model in plain Go: no cgo, no
// assembly, no server. It reads model.safetensors (float32) and tokenizer.json
// of a BERT model that lower-cases (all-MiniLM-L6-v2), and turns a text into a
// vector of unit length.
package embed

import (
	"errors"
	"fmt"
	"math"
	"path/filepath"
	"runtime"
	"sync"
)

// Config is the shape of the model. The loader checks every tensor of the
// weight file against it.
type Config struct {
	Vocab, Hidden, Layers, Heads, Inter, MaxPos int
}

// MiniLM is all-MiniLM-L6-v2: 6 layers of 384 values.
var MiniLM = Config{Vocab: 30522, Hidden: 384, Layers: 6, Heads: 12, Inter: 1536, MaxPos: 512}

// MaxTokens is where a text is cut, [CLS] and [SEP] included.
const MaxTokens = 256

// ErrClosed is returned by Embed after Close.
var ErrClosed = errors.New("the model is closed")

type layer struct {
	qw, qb, kw, kb, vw, vb, ow, ob, ln1w, ln1b, iw, ib, fw, fb, ln2w, ln2b []float32
}

// Model is a loaded model. It is safe for use by one goroutine at a time.
type Model struct {
	cfg                      Config
	word, pos, typ, lnw, lnb []float32
	layers                   []layer
	tok                      *wordPiece
	cls, sep                 int
	file                     *tensorFile
	mu                       sync.Mutex
	closed                   bool
}

// Load opens the model in dir: model.safetensors and tokenizer.json. The
// weight file is mapped where the system allows it. A malformed, cut or
// oversized file is an error that names the check, never a crash or an
// allocation the size of what the file claims.
func Load(dir string, cfg Config) (*Model, error) {
	if cfg.Hidden%cfg.Heads != 0 || cfg.MaxPos < MaxTokens {
		return nil, fmt.Errorf("model shape %+v is not usable", cfg)
	}
	f, err := openTensors(filepath.Join(dir, "model.safetensors"))
	if err != nil {
		return nil, err
	}
	m, err := build(f, dir, cfg)
	if err != nil {
		_ = f.close()
		return nil, err
	}
	return m, nil
}

func build(f *tensorFile, dir string, cfg Config) (*Model, error) {
	m := &Model{cfg: cfg, file: f}
	var err error
	get := func(name string, shape ...int) []float32 {
		if err != nil {
			return nil
		}
		var v []float32
		v, err = f.get(name, shape...)
		return v
	}
	h := cfg.Hidden
	m.word = get("embeddings.word_embeddings.weight", cfg.Vocab, h)
	m.pos = get("embeddings.position_embeddings.weight", cfg.MaxPos, h)
	m.typ = get("embeddings.token_type_embeddings.weight", 2, h)
	m.lnw, m.lnb = get("embeddings.LayerNorm.weight", h), get("embeddings.LayerNorm.bias", h)
	for i := 0; i < cfg.Layers; i++ {
		p := fmt.Sprintf("encoder.layer.%d.", i)
		m.layers = append(m.layers, layer{
			qw: get(p+"attention.self.query.weight", h, h), qb: get(p+"attention.self.query.bias", h),
			kw: get(p+"attention.self.key.weight", h, h), kb: get(p+"attention.self.key.bias", h),
			vw: get(p+"attention.self.value.weight", h, h), vb: get(p+"attention.self.value.bias", h),
			ow: get(p+"attention.output.dense.weight", h, h), ob: get(p+"attention.output.dense.bias", h),
			ln1w: get(p+"attention.output.LayerNorm.weight", h), ln1b: get(p+"attention.output.LayerNorm.bias", h),
			iw: get(p+"intermediate.dense.weight", cfg.Inter, h), ib: get(p+"intermediate.dense.bias", cfg.Inter),
			fw: get(p+"output.dense.weight", h, cfg.Inter), fb: get(p+"output.dense.bias", h),
			ln2w: get(p+"output.LayerNorm.weight", h), ln2b: get(p+"output.LayerNorm.bias", h),
		})
	}
	if err != nil {
		return nil, err
	}
	if m.tok, err = loadWordPiece(filepath.Join(dir, "tokenizer.json")); err != nil {
		return nil, err
	}
	for _, id := range m.tok.vocab {
		if id < 0 || id >= cfg.Vocab {
			return nil, fmt.Errorf("tokenizer file: id %d is outside the %d words of the model", id, cfg.Vocab)
		}
	}
	var ok1, ok2 bool
	m.cls, ok1 = m.tok.vocab["[CLS]"]
	m.sep, ok2 = m.tok.vocab["[SEP]"]
	if !ok1 || !ok2 {
		return nil, errors.New("tokenizer file: [CLS] or [SEP] is not in the vocabulary")
	}
	return m, nil
}

// Dim is the length of the vectors.
func (m *Model) Dim() int { return m.cfg.Hidden }

// Close unmaps the weight file. The model cannot be used afterwards.
func (m *Model) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return nil
	}
	m.closed = true
	return m.file.close()
}

var workers = runtime.GOMAXPROCS(0)

func dot(a, c []float32) float32 {
	c = c[:len(a)]
	var s0, s1, s2, s3 float32
	i := 0
	for ; i+4 <= len(a); i += 4 {
		s0 += a[i] * c[i]
		s1 += a[i+1] * c[i+1]
		s2 += a[i+2] * c[i+2]
		s3 += a[i+3] * c[i+3]
	}
	for ; i < len(a); i++ {
		s0 += a[i] * c[i]
	}
	return s0 + s1 + s2 + s3
}

// dot4 returns the dot products of x with four rows at once, reading x once.
func dot4(x, w0, w1, w2, w3 []float32) (float32, float32, float32, float32) {
	w0, w1, w2, w3 = w0[:len(x)], w1[:len(x)], w2[:len(x)], w3[:len(x)]
	var a0, a1, a2, a3 float32
	for i, v := range x {
		a0 += v * w0[i]
		a1 += v * w1[i]
		a2 += v * w2[i]
		a3 += v * w3[i]
	}
	return a0, a1, a2, a3
}

// parallel runs fn on contiguous ranges of [0, n) on all cores.
func parallel(n int, fn func(lo, hi int)) {
	if n <= 0 {
		return
	}
	var wg sync.WaitGroup
	chunk := (n + workers - 1) / workers
	for lo := 0; lo < n; lo += chunk {
		hi := min(lo+chunk, n)
		wg.Add(1)
		go func() {
			defer wg.Done()
			fn(lo, hi)
		}()
	}
	wg.Wait()
}

// linear computes dst[t][o] = x[t]·w[o] + bias[o] for n tokens; w is [out][in].
// out is a multiple of 4: the rows are read four at a time.
func linear(dst, x []float32, n, in, out int, w, bias []float32) {
	parallel(out/4, func(lo, hi int) {
		for g := lo; g < hi; g++ {
			o := g * 4
			r0, r1, r2, r3 := w[o*in:(o+1)*in], w[(o+1)*in:(o+2)*in], w[(o+2)*in:(o+3)*in], w[(o+3)*in:(o+4)*in]
			for t := 0; t < n; t++ {
				a0, a1, a2, a3 := dot4(x[t*in:(t+1)*in], r0, r1, r2, r3)
				d := dst[t*out+o : t*out+o+4]
				d[0], d[1], d[2], d[3] = a0+bias[o], a1+bias[o+1], a2+bias[o+2], a3+bias[o+3]
			}
		}
	})
}

func layerNorm(x []float32, n, d int, w, b []float32) {
	parallel(n, func(lo, hi int) {
		for t := lo; t < hi; t++ {
			row := x[t*d : (t+1)*d]
			var mean float64
			for _, v := range row {
				mean += float64(v)
			}
			mean /= float64(d)
			var variance float64
			for _, v := range row {
				c := float64(v) - mean
				variance += c * c
			}
			inv := 1 / math.Sqrt(variance/float64(d)+1e-12)
			for i, v := range row {
				row[i] = float32((float64(v)-mean)*inv)*w[i] + b[i]
			}
		}
	})
}

// Embed returns the vector of unit length of text: the mean of the vectors of
// its tokens. A text is cut at MaxTokens tokens.
func (m *Model) Embed(text string) ([]float32, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return nil, ErrClosed
	}
	ids := m.tok.encode(text)
	if len(ids) > MaxTokens-2 {
		ids = ids[:MaxTokens-2]
	}
	ids = append(append([]int{m.cls}, ids...), m.sep)
	n, d, inter := len(ids), m.cfg.Hidden, m.cfg.Inter
	h := make([]float32, n*d)
	for t, id := range ids {
		for i := 0; i < d; i++ {
			h[t*d+i] = m.word[id*d+i] + m.pos[t*d+i] + m.typ[i]
		}
	}
	layerNorm(h, n, d, m.lnw, m.lnb)
	q, k, v := make([]float32, n*d), make([]float32, n*d), make([]float32, n*d)
	ctx, tmp := make([]float32, n*d), make([]float32, n*d)
	ff := make([]float32, n*inter)
	dh := d / m.cfg.Heads
	scale := float32(1 / math.Sqrt(float64(dh)))
	for _, l := range m.layers {
		linear(q, h, n, d, d, l.qw, l.qb)
		linear(k, h, n, d, d, l.kw, l.kb)
		linear(v, h, n, d, d, l.vw, l.vb)
		parallel(m.cfg.Heads, func(h0, h1 int) {
			scores := make([]float32, n)
			for hd := h0; hd < h1; hd++ {
				off := hd * dh
				for i := 0; i < n; i++ {
					qi := q[i*d+off : i*d+off+dh]
					mx := float32(-1e30)
					for j := 0; j < n; j++ {
						s := dot(qi, k[j*d+off:j*d+off+dh]) * scale
						scores[j] = s
						mx = max(mx, s)
					}
					var sum float32
					for j := 0; j < n; j++ {
						scores[j] = float32(math.Exp(float64(scores[j] - mx)))
						sum += scores[j]
					}
					out := ctx[i*d+off : i*d+off+dh]
					clear(out)
					for j := 0; j < n; j++ {
						p := scores[j] / sum
						vj := v[j*d+off : j*d+off+dh]
						for x := range out {
							out[x] += p * vj[x]
						}
					}
				}
			}
		})
		linear(tmp, ctx, n, d, d, l.ow, l.ob)
		for i := range h {
			h[i] += tmp[i]
		}
		layerNorm(h, n, d, l.ln1w, l.ln1b)
		linear(ff, h, n, d, inter, l.iw, l.ib)
		parallel(len(ff), func(lo, hi int) {
			for i := lo; i < hi; i++ {
				x := float64(ff[i])
				ff[i] = float32(0.5 * x * (1 + math.Erf(x/math.Sqrt2)))
			}
		})
		linear(tmp, ff, n, inter, d, l.fw, l.fb)
		for i := range h {
			h[i] += tmp[i]
		}
		layerNorm(h, n, d, l.ln2w, l.ln2b)
	}
	out := make([]float32, d)
	for t := 0; t < n; t++ {
		for i := 0; i < d; i++ {
			out[i] += h[t*d+i]
		}
	}
	var sum float64
	for _, x := range out {
		sum += float64(x) * float64(x)
	}
	if sum == 0 || math.IsNaN(sum) {
		return nil, errors.New("the model gave a vector of no length")
	}
	inv := float32(1 / math.Sqrt(sum))
	for i := range out {
		out[i] *= inv
	}
	return out, nil
}

// Cosine is the cosine similarity of two vectors of unit length.
func Cosine(a, b []float32) float32 { return dot(a, b) }
