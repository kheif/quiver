package library

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/kheif/quiver/internal/embed"
	"github.com/kheif/quiver/internal/fsutil"
)

// DefaultModel is the model of the local engine: all-MiniLM-L6-v2, English.
const DefaultModel = "all-minilm-l6-v2"

// defaultModelBase is the pinned revision of the model on the hub: the full
// commit, never a branch. docs/spike-discovery-results.md records it.
const defaultModelBase = "https://huggingface.co/sentence-transformers/all-MiniLM-L6-v2/resolve/1110a243fdf4706b3f48f1d95db1a4f5529b4d41"

// ModelSpec is a model Quiver knows: the shape its loader checks, and the
// pinned size and hash of its weight file. TASK62.7 downloads the files; the
// hash of a file is checked there and by quiver doctor, not at every search,
// where only the size is.
type ModelSpec struct {
	Name   string
	Config embed.Config
	// Size and SHA256 are of model.safetensors; TokenizerSHA256 is of
	// tokenizer.json.
	Size            int64
	SHA256          string
	TokenizerSize   int64
	TokenizerSHA256 string
	// BaseURL is the folder of the pinned revision that the two files are
	// downloaded from. Only the models of tests have another one.
	BaseURL string
	// MinScore is the cosine below which a vector match is dropped, from the
	// calibration of TASK62.1: the highest top score of the queries that fit
	// nothing.
	MinScore float64
}

// models are the models Quiver knows, by the name of their folder under
// cache/models/. Tests add their own.
var models = map[string]ModelSpec{
	DefaultModel: {
		Name:            DefaultModel,
		Config:          embed.MiniLM,
		Size:            90868376,
		SHA256:          "53aa51172d142c89d9012cce15ae4d6cc0ca6895895114379cacb4fab128d9db",
		TokenizerSize:   466247,
		TokenizerSHA256: "be50c3628f2bf5bb5e3a7f17b1f74611b2561a3a27eeab05e5aa30f411572037",
		BaseURL:         defaultModelBase,
		MinScore:        0.287,
	},
}

// RegisterModel adds a model to the known ones, for tests of other packages,
// and returns the function that takes it out again.
func RegisterModel(spec ModelSpec) (unregister func()) {
	models[spec.Name] = spec
	return func() { delete(models, spec.Name) }
}

// KnownModel returns the spec of a model by name.
func KnownModel(name string) (ModelSpec, bool) {
	s, ok := models[name]
	return s, ok
}

// ModelDir is the folder of a model, cache/models/<name> in the Quiver home.
func ModelDir(name string) (string, error) {
	home, err := fsutil.QuiverHome()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "cache", "models", name), nil
}

// ModelFile is the weight file of a model.
func ModelFile(name string) (string, error) {
	dir, err := ModelDir(name)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "model.safetensors"), nil
}

// CheckModelFile says what is missing or wrong with the model's files, in one
// line, without reading the weights: a search checks only that the weight
// file has the pinned size. quiver doctor and the setup of TASK62.7 hash it.
func (s ModelSpec) CheckModelFile() error {
	dir, err := ModelDir(s.Name)
	if err != nil {
		return err
	}
	weights := filepath.Join(dir, "model.safetensors")
	st, err := os.Stat(weights)
	switch {
	case os.IsNotExist(err):
		return fmt.Errorf("the model file %s is missing", weights)
	case err != nil:
		return err
	case s.Size != 0 && st.Size() != s.Size:
		return fmt.Errorf("the model file %s has %d bytes, not the %d of the model", weights, st.Size(), s.Size)
	}
	if _, err := os.Stat(filepath.Join(dir, "tokenizer.json")); err != nil {
		return fmt.Errorf("the tokenizer file %s is missing", filepath.Join(dir, "tokenizer.json"))
	}
	return nil
}

// VerifyModelFile hashes the model's files and compares them with the pinned
// hashes. It reads the whole weight file, so a search never does it; quiver
// doctor and the setup of TASK62.7 do.
func (s ModelSpec) VerifyModelFile() error {
	if err := s.CheckModelFile(); err != nil {
		return err
	}
	dir, err := ModelDir(s.Name)
	if err != nil {
		return err
	}
	for _, file := range []string{"model.safetensors", "tokenizer.json"} {
		want := s.SHA256
		if file == "tokenizer.json" {
			want = s.TokenizerSHA256
		}
		if want == "" {
			continue
		}
		got, err := fileSHA256(filepath.Join(dir, file))
		if err != nil {
			return err
		}
		if got != want {
			return fmt.Errorf("%s does not have the pinned SHA-256 (%s, not %s)", filepath.Join(dir, file), got, want)
		}
	}
	return nil
}

func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
