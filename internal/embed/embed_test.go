package embed

import (
	"encoding/binary"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kheif/quiver/internal/embed/embedtest"
)

// tinyConfig is the shape of the model that embedtest writes.
var tinyConfig = Config(embedtest.Tiny)

func writeTiny(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	embedtest.WriteModel(t, dir, 1)
	return dir
}

func loadTiny(t *testing.T) *Model {
	t.Helper()
	m, err := Load(writeTiny(t), tinyConfig)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = m.Close() })
	return m
}

func TestEmbedGivesVectorsOfUnitLength(t *testing.T) {
	m := loadTiny(t)
	if m.Dim() != 8 {
		t.Errorf("dim %d", m.Dim())
	}
	var vecs [][]float32
	for _, text := range []string{"send a note", "send a note", "pull-request review", "", "!!!", "zzzz qqqq 12345"} {
		v, err := m.Embed(text)
		if err != nil || len(v) != 8 {
			t.Fatalf("%q: %v, %v", text, v, err)
		}
		var sum float64
		for _, x := range v {
			sum += float64(x) * float64(x)
		}
		if math.Abs(sum-1) > 1e-5 {
			t.Errorf("%q: length %v", text, sum)
		}
		vecs = append(vecs, v)
	}
	if c := Cosine(vecs[0], vecs[1]); c < 0.99999 {
		t.Errorf("the same text twice: cosine %v", c)
	}
	if c := Cosine(vecs[0], vecs[2]); c > 0.999999 {
		t.Errorf("different texts: cosine %v", c)
	}
}

func TestEmbedStripsAccentsAndCase(t *testing.T) {
	m := loadTiny(t)
	a, _ := m.Embed("Café NOTE")
	b, _ := m.Embed("cafe note")
	if Cosine(a, b) < 0.99999 {
		t.Errorf("cosine %v", Cosine(a, b))
	}
}

func TestEmbedCutsALongText(t *testing.T) {
	m := loadTiny(t)
	cut := strings.TrimSpace(strings.Repeat("a ", MaxTokens-2))
	long := strings.TrimSpace(strings.Repeat("a ", 3*MaxTokens))
	a, err1 := m.Embed(cut)
	b, err2 := m.Embed(long)
	if err1 != nil || err2 != nil || Cosine(a, b) < 0.99999 {
		t.Errorf("errors %v %v, cosine %v", err1, err2, Cosine(a, b))
	}
}

func TestEmbedAfterClose(t *testing.T) {
	m := loadTiny(t)
	if err := m.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Embed("x"); err != ErrClosed {
		t.Errorf("err = %v", err)
	}
	if err := m.Close(); err != nil {
		t.Errorf("second close: %v", err)
	}
}

// Each check of the loader names itself, and none crashes or allocates by the
// size the file claims.
func TestLoadChecks(t *testing.T) {
	names, shapes := embedtest.Tensors(embedtest.Tiny)
	good := embedtest.Safetensors(names, shapes, 1, nil)
	header := func(raw []byte) (int, map[string]any) {
		n := int(binary.LittleEndian.Uint64(raw))
		var h map[string]any
		_ = json.Unmarshal(raw[8:8+n], &h)
		return n, h
	}
	rewrite := func(f func(h map[string]any)) []byte {
		n, h := header(good)
		f(h)
		hj, _ := json.Marshal(h)
		out := binary.LittleEndian.AppendUint64(nil, uint64(len(hj)))
		return append(append(out, hj...), good[8+n:]...)
	}
	word := "embeddings.word_embeddings.weight"
	cases := map[string]struct {
		file []byte
		want string
	}{
		"empty":             {nil, "size 0"},
		"cut in the header": {good[:20], "header length"},
		"cut in the data":   {good[:len(good)-100], "outside the"},
		"header too long":   {append(binary.LittleEndian.AppendUint64(nil, 1<<40), good[8:]...), "header length"},
		"header not JSON":   {append(binary.LittleEndian.AppendUint64(nil, 5), []byte("{nope rest")...), "not JSON"},
		"wrong type":        {rewrite(func(h map[string]any) { h[word].(map[string]any)["dtype"] = "F16" }), "type F16"},
		"wrong shape": {rewrite(func(h map[string]any) {
			h[word].(map[string]any)["shape"] = []int{tinyConfig.Hidden, tinyConfig.Vocab}
		}), "has shape [8 64], want [64 8]"},
		"shape and bytes disagree": {rewrite(func(h map[string]any) { h[word].(map[string]any)["shape"] = []int{64, 9} }), "its shape [64 9] needs"},
		"absurd shape": {rewrite(func(h map[string]any) {
			h[word].(map[string]any)["shape"] = []int{1 << 40, 1 << 40}
		}), "has shape"},
		"negative offsets": {rewrite(func(h map[string]any) { h[word].(map[string]any)["data_offsets"] = []int{-8, 8} }), "offsets"},
		"offsets past the data": {rewrite(func(h map[string]any) {
			h[word].(map[string]any)["data_offsets"] = []int{0, 1 << 40}
		}), "outside the"},
		"missing tensor": {rewrite(func(h map[string]any) { delete(h, "embeddings.LayerNorm.bias") }), "tensor embeddings.LayerNorm.bias is missing"},
	}
	for name, c := range cases {
		dir := t.TempDir()
		embedtest.Write(t, dir, "model.safetensors", c.file)
		embedtest.Write(t, dir, "tokenizer.json", embedtest.Tokenizer(nil))
		m, err := Load(dir, tinyConfig)
		if err == nil {
			_ = m.Close()
			t.Errorf("%s: no error", name)
			continue
		}
		if !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: %q does not name %q", name, err, c.want)
		}
	}
}

func TestLoadRefusesAnOversizedFile(t *testing.T) {
	dir := writeTiny(t)
	f, err := os.OpenFile(filepath.Join(dir, "model.safetensors"), os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	// A sparse file: it costs no disk, and its size is over the limit.
	if err := f.Truncate(maxFileSize + 1); err != nil {
		t.Skip("cannot make a sparse file:", err)
	}
	_ = f.Close()
	if _, err := Load(dir, tinyConfig); err == nil || !strings.Contains(err.Error(), "size") {
		t.Errorf("err = %v", err)
	}
}

func TestLoadTokenizerChecks(t *testing.T) {
	names, shapes := embedtest.Tensors(embedtest.Tiny)
	model := embedtest.Safetensors(names, shapes, 1, nil)
	for name, tk := range map[string]struct {
		file []byte
		want string
	}{
		"not JSON":          {[]byte("{"), "tokenizer file"},
		"not WordPiece":     {[]byte(`{"model":{"type":"BPE","vocab":{"a":0}}}`), "not a WordPiece"},
		"id out of range":   {embedtest.Tokenizer(map[string]int{"zz": 9999}), "outside the 64 words"},
		"no special tokens": {[]byte(`{"model":{"type":"WordPiece","vocab":{"[UNK]":0,"a":1},"unk_token":"[UNK]"}}`), "[CLS] or [SEP]"},
		"no unknown token":  {[]byte(`{"model":{"type":"WordPiece","vocab":{"a":0},"unk_token":"[UNK]"}}`), "unknown token"},
	} {
		dir := t.TempDir()
		embedtest.Write(t, dir, "model.safetensors", model)
		embedtest.Write(t, dir, "tokenizer.json", tk.file)
		if _, err := Load(dir, tinyConfig); err == nil || !strings.Contains(err.Error(), tk.want) {
			t.Errorf("%s: err = %v, want %q", name, err, tk.want)
		}
	}
	dir := t.TempDir()
	embedtest.Write(t, dir, "model.safetensors", model)
	if _, err := Load(dir, tinyConfig); err == nil {
		t.Error("no tokenizer file: no error")
	}
}

// An unaligned tensor is copied, not viewed.
func TestFloatsOfUnalignedBytes(t *testing.T) {
	raw := make([]byte, 17)
	for i, v := range []float32{1.5, -2, 0.25, 8} {
		binary.LittleEndian.PutUint32(raw[1+4*i:], math.Float32bits(v))
	}
	got := floats(raw[1:17])
	want := []float32{1.5, -2, 0.25, 8}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v", got)
		}
	}
	if floats(nil) != nil {
		t.Error("floats of nothing")
	}
}

type golden struct {
	Text   string    `json:"text"`
	Repeat int       `json:"repeat"`
	IDs    []int     `json:"ids"`
	Vector []float32 `json:"vector"`
}

// QUIVER_TEST_MODEL names a folder with the real all-MiniLM-L6-v2 files. The
// golden token ids and vectors were made by the spike's own implementation,
// which TASK62.1 checked against Ollama's all-minilm (cosine 1.00000 for all
// 472 texts), for texts with accents, punctuation, an empty text and a text
// over 256 tokens.
func TestRealModelMatchesTheGoldenVectors(t *testing.T) {
	dir := os.Getenv("QUIVER_TEST_MODEL")
	if dir == "" {
		t.Skip("QUIVER_TEST_MODEL does not name the all-MiniLM-L6-v2 folder")
	}
	data, err := os.ReadFile("testdata/minilm-golden.json")
	if err != nil {
		t.Fatal(err)
	}
	var gold []golden
	if err := json.Unmarshal(data, &gold); err != nil {
		t.Fatal(err)
	}
	m, err := Load(dir, MiniLM)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = m.Close() }()
	for _, g := range gold {
		text := g.Text
		if g.Repeat > 0 {
			text = strings.Repeat(text, g.Repeat)
		}
		ids := m.tok.encode(text)
		if len(ids) != len(g.IDs) {
			t.Errorf("%.30q: %d token ids, want %d", g.Text, len(ids), len(g.IDs))
		}
		for i := range ids {
			if i < len(g.IDs) && ids[i] != g.IDs[i] {
				t.Errorf("%.30q: token %d is %d, want %d", g.Text, i, ids[i], g.IDs[i])
				break
			}
		}
		v, err := m.Embed(text)
		if err != nil {
			t.Fatal(err)
		}
		if c := Cosine(v, g.Vector); c < 0.9999 {
			t.Errorf("%.30q: cosine %v against the reference", g.Text, c)
		}
	}
	// A few known queries: the skill that fits is nearer than one that does not.
	q, _ := m.Embed("send a note to the team channel")
	fit, _ := m.Embed("slack-messaging: Send messages to a Slack channel or a colleague.")
	other, _ := m.Embed("pdf-forms: Fill in and read PDF forms.")
	if Cosine(q, fit) <= Cosine(q, other) {
		t.Errorf("fitting %v, other %v", Cosine(q, fit), Cosine(q, other))
	}
}
