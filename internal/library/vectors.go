package library

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"sync"
	"time"

	"github.com/kheif/quiver/internal/embed"
	"github.com/kheif/quiver/internal/fsutil"
	"github.com/kheif/quiver/internal/skill"
)

// VectorFile is the index of vectors, relative to the Quiver home. It is
// derived data: not synced, not committed, and rebuilt when it does not fit.
const VectorFile = "cache/embeddings.json"

// vectorVersion is the format version of the index. An index of another
// version is ignored and rebuilt.
const vectorVersion = 1

// batchSize is how many skills are embedded before the index is merged and
// written, so a killed process loses at most one batch.
const batchSize = 8

// vectorEntry is one skill: a hash of the embedded text, and its vector.
type vectorEntry struct {
	ID     string    `json:"id"`
	Hash   string    `json:"hash"`
	Vector []float32 `json:"vector"`
}

// vectorIndex is the file: the model it was made with, and the vectors.
type vectorIndex struct {
	Version int           `json:"version"`
	Model   string        `json:"model"`
	SHA256  string        `json:"sha256"`
	Dim     int           `json:"dim"`
	Entries []vectorEntry `json:"entries"`
}

func vectorPath() (string, error) {
	home, err := fsutil.QuiverHome()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, filepath.FromSlash(VectorFile)), nil
}

// embeddedText is what is embedded of a skill: its ID, name and description.
func embeddedText(s *skill.Skill) string {
	return s.ID + " " + s.Name + ": " + s.Description
}

func textHash(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:])
}

// emptyIndex is an index with no entries, for the model.
func emptyIndex(spec ModelSpec) *vectorIndex {
	return &vectorIndex{Version: vectorVersion, Model: spec.Name, SHA256: spec.SHA256, Dim: spec.Config.Hidden}
}

// fits reports whether the index was made by this format and this model.
func (x *vectorIndex) fits(spec ModelSpec) bool {
	return x.Version == vectorVersion && x.Model == spec.Name && x.SHA256 == spec.SHA256 && x.Dim == spec.Config.Hidden
}

// loadVectors reads the index. A missing, unreadable or foreign index is an
// empty one for the model: it is rebuilt, not refused, because it is derived
// data.
func loadVectors(spec ModelSpec) *vectorIndex {
	path, err := vectorPath()
	if err != nil {
		return emptyIndex(spec)
	}
	var x vectorIndex
	if err := fsutil.ReadJSON(path, &x); err != nil || !x.fits(spec) {
		return emptyIndex(spec)
	}
	x.Entries = slices.DeleteFunc(x.Entries, func(e vectorEntry) bool { return len(e.Vector) != spec.Config.Hidden })
	return &x
}

func (x *vectorIndex) byID() map[string]vectorEntry {
	m := make(map[string]vectorEntry, len(x.Entries))
	for _, e := range x.Entries {
		m[e.ID] = e
	}
	return m
}

// VectorStatus says how much of the library the index covers.
type VectorStatus struct {
	Indexed int // skills with a current vector
	Stale   int // skills with none, or with a vector of an older text
	Total   int
}

// vectorStatus compares the library with the index.
func vectorStatus(x *vectorIndex, all []*skill.Skill) (VectorStatus, []*skill.Skill) {
	have := x.byID()
	st := VectorStatus{Total: len(all)}
	var missing []*skill.Skill
	for _, s := range all {
		if e, ok := have[s.ID]; ok && e.Hash == textHash(embeddedText(s)) {
			st.Indexed++
		} else {
			missing = append(missing, s)
		}
	}
	st.Stale = len(missing)
	return st, missing
}

// IndexStatus reports how many skills of the library the index covers with a
// current vector, for the model. It embeds nothing.
func (l *Library) IndexStatus(modelName string) (VectorStatus, error) {
	spec, ok := models[modelName]
	if !ok {
		return VectorStatus{}, fmt.Errorf("unknown model %q", modelName)
	}
	st, _ := vectorStatus(loadVectors(spec), l.All())
	return st, nil
}

// loaded is the model a process has mapped. A process maps the file once and
// the operating system shares the pages between processes.
var loaded struct {
	sync.Mutex
	path string
	size int64
	mod  time.Time
	m    *embed.Model
}

// openModel returns the mapped model, loading it at the first use, and again
// when the file was replaced.
func openModel(spec ModelSpec) (*embed.Model, error) {
	if err := spec.CheckModelFile(); err != nil {
		return nil, err
	}
	dir, err := ModelDir(spec.Name)
	if err != nil {
		return nil, err
	}
	path := filepath.Join(dir, "model.safetensors")
	st, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	loaded.Lock()
	defer loaded.Unlock()
	if loaded.m != nil && loaded.path == path && loaded.size == st.Size() && loaded.mod.Equal(st.ModTime()) {
		return loaded.m, nil
	}
	m, err := embed.Load(dir, spec.Config)
	if err != nil {
		return nil, err
	}
	// The old mapping is left to the process: another goroutine may still
	// be embedding with it.
	loaded.path, loaded.size, loaded.mod, loaded.m = path, st.Size(), st.ModTime(), m
	return m, nil
}

// embedder turns a text into a vector. Tests replace it to count calls and to
// fail.
var embedder = func(m *embed.Model, text string) ([]float32, error) { return m.Embed(text) }

// refreshIndex embeds the skills of missing in batches. After each batch it
// takes the home lock, merges the batch into the file as it is then, drops
// the entries of skills that are not in all any more and writes the file
// atomically. The embedding runs outside the lock, so two processes may do
// some work twice and neither loses the other's entries. It stops when ctx is
// done, with what has been saved. progress is called after every batch with
// the number of skills embedded of len(missing).
func refreshIndex(ctx context.Context, spec ModelSpec, m *embed.Model, all, missing []*skill.Skill, progress func(done, total int)) error {
	path, err := vectorPath()
	if err != nil {
		return err
	}
	live := map[string]bool{}
	for _, s := range all {
		live[s.ID] = true
	}
	done := 0
	for len(missing) > 0 {
		n := min(batchSize, len(missing))
		batch := make([]vectorEntry, 0, n)
		for _, s := range missing[:n] {
			if err := ctx.Err(); err != nil {
				// What is embedded of this batch is saved first.
				if len(batch) > 0 && saveBatch(path, spec, batch, live) == nil {
					done += len(batch)
					if progress != nil {
						progress(done, done+len(missing)-len(batch))
					}
				}
				return err
			}
			text := embeddedText(s)
			v, err := embedder(m, text)
			if err != nil {
				return fmt.Errorf("embedding %s: %w", s.ID, err)
			}
			batch = append(batch, vectorEntry{ID: s.ID, Hash: textHash(text), Vector: v})
		}
		if err := saveBatch(path, spec, batch, live); err != nil {
			return err
		}
		done += n
		missing = missing[n:]
		if progress != nil {
			progress(done, done+len(missing))
		}
	}
	return nil
}

// saveBatch merges a batch into the index on disk under the lock.
func saveBatch(path string, spec ModelSpec, batch []vectorEntry, live map[string]bool) error {
	return fsutil.WithLock(func() error {
		x := loadVectors(spec)
		have := map[string]vectorEntry{}
		for _, e := range x.Entries {
			if live[e.ID] {
				have[e.ID] = e
			}
		}
		for _, e := range batch {
			have[e.ID] = e
		}
		x.Entries = x.Entries[:0]
		for _, e := range have {
			x.Entries = append(x.Entries, e)
		}
		sort.Slice(x.Entries, func(i, j int) bool { return x.Entries[i].ID < x.Entries[j].ID })
		return fsutil.WriteJSONAtomic(path, x)
	})
}

// Reindex rebuilds the whole index of the model without a time limit. It
// returns the number of skills embedded.
func (l *Library) Reindex(ctx context.Context, modelName string, progress func(done, total int)) (int, error) {
	spec, ok := models[modelName]
	if !ok {
		return 0, fmt.Errorf("unknown model %q", modelName)
	}
	m, err := openModel(spec)
	if err != nil {
		return 0, err
	}
	path, err := vectorPath()
	if err != nil {
		return 0, err
	}
	// Start from an empty index: every skill is embedded again.
	if err := fsutil.WithLock(func() error { return os.Remove(path) }); err != nil && !errors.Is(err, os.ErrNotExist) {
		return 0, err
	}
	all := l.All()
	return len(all), refreshIndex(ctx, spec, m, all, all, progress)
}
