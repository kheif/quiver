package main

// A BERT-style sentence encoder (all-MiniLM, bge-small, gte-small, e5-small)
// written in plain Go: embeddings, post-LayerNorm transformer layers, pooling,
// L2 normalisation. float32, no SIMD, no cgo; the linear layers run on all cores.

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"unsafe"
)

type Embedder interface {
	EmbedQuery(text string) []float32
	EmbedDoc(text string) []float32
	Close()
}

func (s *Static) EmbedQuery(t string) []float32 { return s.Embed(t) }
func (s *Static) EmbedDoc(t string) []float32   { return s.Embed(t) }

// LoadAny opens a static model or a BERT model, by the model_type in config.json.
func LoadAny(dir string, mmap bool) (Embedder, error) {
	var cfg struct {
		ModelType string `json:"model_type"`
	}
	if b, err := os.ReadFile(filepath.Join(dir, "config.json")); err == nil {
		json.Unmarshal(b, &cfg)
	}
	if cfg.ModelType == "bert" {
		return LoadBert(dir, mmap)
	}
	return LoadStatic(dir, mmap)
}

type tensorInfo struct {
	Dtype   string `json:"dtype"`
	Shape   []int  `json:"shape"`
	Offsets [2]int `json:"data_offsets"`
}

type tensorFile struct {
	raw  []byte
	base int
	t    map[string]tensorInfo
}

func openSafetensors(path string, mmap bool) (*tensorFile, error) {
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
	tf := &tensorFile{raw: raw, base: 8 + n, t: map[string]tensorInfo{}}
	for k, v := range hdr {
		if k == "__metadata__" {
			continue
		}
		var ti tensorInfo
		if err := json.Unmarshal(v, &ti); err != nil {
			return nil, err
		}
		tf.t[k] = ti
	}
	return tf, nil
}

func half2float(h uint16) float32 {
	sign := uint32(h>>15) << 31
	exp := uint32(h>>10) & 0x1f
	man := uint32(h) & 0x3ff
	switch {
	case exp == 0 && man == 0:
		return math.Float32frombits(sign)
	case exp == 0: // subnormal
		e := uint32(127 - 15 + 1)
		for man&0x400 == 0 {
			man <<= 1
			e--
		}
		return math.Float32frombits(sign | e<<23 | (man&0x3ff)<<13)
	case exp == 31:
		return math.Float32frombits(sign | 0xff<<23 | man<<13)
	}
	return math.Float32frombits(sign | (exp+127-15)<<23 | man<<13)
}

// f32 returns a tensor as float32: a view of the file for F32, a converted copy for F16.
func (f *tensorFile) f32(name string) ([]float32, error) {
	ti, ok := f.t[name]
	if !ok {
		return nil, fmt.Errorf("tensor %s missing", name)
	}
	data := f.raw[f.base+ti.Offsets[0] : f.base+ti.Offsets[1]]
	switch ti.Dtype {
	case "F32":
		return unsafe.Slice((*float32)(unsafe.Pointer(&data[0])), len(data)/4), nil
	case "F16":
		out := make([]float32, len(data)/2)
		for i := range out {
			out[i] = half2float(binary.LittleEndian.Uint16(data[2*i:]))
		}
		return out, nil
	}
	return nil, fmt.Errorf("tensor %s: dtype %s", name, ti.Dtype)
}

type bertLayer struct {
	qw, qb, kw, kb, vw, vb, ow, ob, ln1w, ln1b, iw, ib, fw, fb, ln2w, ln2b []float32
}

type Bert struct {
	hidden, heads, inter, maxLen int
	word, pos, typ, lnw, lnb     []float32
	layers                       []bertLayer
	tok                          *WordPiece
	cls, sep                     int
	pooling                      string
	qPrefix, dPrefix             string
	file                         *tensorFile
}

// LoadBert reads model.safetensors, tokenizer.json, config.json and, if there,
// spike.json with the pooling ("mean" or "cls") and the query and document prefixes.
func LoadBert(dir string, mmap bool) (*Bert, error) {
	var cfg struct {
		Hidden int `json:"hidden_size"`
		Layers int `json:"num_hidden_layers"`
		Heads  int `json:"num_attention_heads"`
		Inter  int `json:"intermediate_size"`
	}
	cb, err := os.ReadFile(filepath.Join(dir, "config.json"))
	if err != nil {
		return nil, err
	}
	json.Unmarshal(cb, &cfg)
	b := &Bert{hidden: cfg.Hidden, heads: cfg.Heads, inter: cfg.Inter, maxLen: 256, pooling: "mean"}
	var side struct {
		Pooling     string `json:"pooling"`
		QueryPrefix string `json:"queryPrefix"`
		DocPrefix   string `json:"docPrefix"`
	}
	if sb, err := os.ReadFile(filepath.Join(dir, "spike.json")); err == nil {
		json.Unmarshal(sb, &side)
		if side.Pooling != "" {
			b.pooling = side.Pooling
		}
		b.qPrefix, b.dPrefix = side.QueryPrefix, side.DocPrefix
	}
	if b.file, err = openSafetensors(filepath.Join(dir, "model.safetensors"), mmap); err != nil {
		return nil, err
	}
	get := func(name string) []float32 {
		if err != nil {
			return nil
		}
		var v []float32
		v, err = b.file.f32(name)
		return v
	}
	b.word = get("embeddings.word_embeddings.weight")
	b.pos = get("embeddings.position_embeddings.weight")
	b.typ = get("embeddings.token_type_embeddings.weight")
	b.lnw, b.lnb = get("embeddings.LayerNorm.weight"), get("embeddings.LayerNorm.bias")
	for i := 0; i < cfg.Layers; i++ {
		p := fmt.Sprintf("encoder.layer.%d.", i)
		b.layers = append(b.layers, bertLayer{
			qw: get(p + "attention.self.query.weight"), qb: get(p + "attention.self.query.bias"),
			kw: get(p + "attention.self.key.weight"), kb: get(p + "attention.self.key.bias"),
			vw: get(p + "attention.self.value.weight"), vb: get(p + "attention.self.value.bias"),
			ow: get(p + "attention.output.dense.weight"), ob: get(p + "attention.output.dense.bias"),
			ln1w: get(p + "attention.output.LayerNorm.weight"), ln1b: get(p + "attention.output.LayerNorm.bias"),
			iw: get(p + "intermediate.dense.weight"), ib: get(p + "intermediate.dense.bias"),
			fw: get(p + "output.dense.weight"), fb: get(p + "output.dense.bias"),
			ln2w: get(p + "output.LayerNorm.weight"), ln2b: get(p + "output.LayerNorm.bias"),
		})
	}
	if err != nil {
		return nil, err
	}
	tj, err := os.ReadFile(filepath.Join(dir, "tokenizer.json"))
	if err != nil {
		return nil, err
	}
	var tk struct {
		Model json.RawMessage `json:"model"`
	}
	json.Unmarshal(tj, &tk)
	if b.tok, err = newWordPiece(tk.Model); err != nil {
		return nil, err
	}
	b.cls, b.sep = b.tok.vocab["[CLS]"], b.tok.vocab["[SEP]"]
	return b, nil
}

func (b *Bert) Close() {
	if b.file != nil && b.file.raw != nil {
		syscall.Munmap(b.file.raw)
	}
}

func (b *Bert) EmbedQuery(t string) []float32 { return b.embed(b.qPrefix + t) }
func (b *Bert) EmbedDoc(t string) []float32   { return b.embed(b.dPrefix + t) }

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

var workers = runtime.NumCPU()

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
	var wg sync.WaitGroup
	chunk := (n + workers - 1) / workers
	for lo := 0; lo < n; lo += chunk {
		hi := lo + chunk
		if hi > n {
			hi = n
		}
		wg.Add(1)
		go func(lo, hi int) {
			defer wg.Done()
			fn(lo, hi)
		}(lo, hi)
	}
	wg.Wait()
}

// linear computes dst[t][o] = x[t]·w[o] + bias[o] for n tokens; w is [out][in].
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
	parallel(n, func(lo, hi int) { layerNormRows(x, lo, hi, d, w, b) })
}

func layerNormRows(x []float32, lo, hi, d int, w, b []float32) {
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
}

func (b *Bert) embed(text string) []float32 {
	ids := b.tok.Encode(text)
	if len(ids) > b.maxLen-2 {
		ids = ids[:b.maxLen-2]
	}
	ids = append(append([]int{b.cls}, ids...), b.sep)
	n, d := len(ids), b.hidden
	h := make([]float32, n*d)
	for t, id := range ids {
		for i := 0; i < d; i++ {
			h[t*d+i] = b.word[id*d+i] + b.pos[t*d+i] + b.typ[i]
		}
	}
	layerNorm(h, n, d, b.lnw, b.lnb)
	q, k, v := make([]float32, n*d), make([]float32, n*d), make([]float32, n*d)
	ctx, tmp := make([]float32, n*d), make([]float32, n*d)
	ff := make([]float32, n*b.inter)
	dh := d / b.heads
	scale := float32(1 / math.Sqrt(float64(dh)))
	for _, l := range b.layers {
		linear(q, h, n, d, d, l.qw, l.qb)
		linear(k, h, n, d, d, l.kw, l.kb)
		linear(v, h, n, d, d, l.vw, l.vb)
		parallel(b.heads, func(h0, h1 int) {
			scores := make([]float32, n)
			for hd := h0; hd < h1; hd++ {
				off := hd * dh
				for i := 0; i < n; i++ {
					qi := q[i*d+off : i*d+off+dh]
					var max float32 = -1e30
					for j := 0; j < n; j++ {
						s := dot(qi, k[j*d+off:j*d+off+dh]) * scale
						scores[j] = s
						if s > max {
							max = s
						}
					}
					var sum float32
					for j := 0; j < n; j++ {
						scores[j] = float32(math.Exp(float64(scores[j] - max)))
						sum += scores[j]
					}
					out := ctx[i*d+off : i*d+off+dh]
					for x := range out {
						out[x] = 0
					}
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
		linear(ff, h, n, d, b.inter, l.iw, l.ib)
		parallel(len(ff), func(lo, hi int) {
			for i := lo; i < hi; i++ {
				x := float64(ff[i])
				ff[i] = float32(0.5 * x * (1 + math.Erf(x/math.Sqrt2)))
			}
		})
		linear(tmp, ff, n, b.inter, d, l.fw, l.fb)
		for i := range h {
			h[i] += tmp[i]
		}
		layerNorm(h, n, d, l.ln2w, l.ln2b)
	}
	out := make([]float32, d)
	if strings.EqualFold(b.pooling, "cls") {
		copy(out, h[:d])
	} else {
		for t := 0; t < n; t++ {
			for i := 0; i < d; i++ {
				out[i] += h[t*d+i]
			}
		}
	}
	var sum float64
	for _, x := range out {
		sum += float64(x) * float64(x)
	}
	inv := float32(1 / math.Sqrt(sum))
	for i := range out {
		out[i] *= inv
	}
	return out
}
