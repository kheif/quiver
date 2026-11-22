package library

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"

	"github.com/kheif/quiver/internal/fsutil"
)

// maxRedirects is how many redirects a model download follows: the host that
// serves the large file is not the host of the pinned URL.
const maxRedirects = 3

// PinnedFile is one file of a model that Quiver downloads, with its pinned size
// and SHA-256.
type PinnedFile struct {
	Name   string
	Size   int64
	SHA256 string
}

// Files returns the two files the loader reads.
func (s ModelSpec) Files() []PinnedFile {
	return []PinnedFile{
		{"model.safetensors", s.Size, s.SHA256},
		{"tokenizer.json", s.TokenizerSize, s.TokenizerSHA256},
	}
}

// modelTransport carries the downloads. Only a test replaces it.
var modelTransport http.RoundTripper = http.DefaultTransport

// SetModelTransport makes downloads use rt, for tests, and returns the
// function that restores the real one.
func SetModelTransport(rt http.RoundTripper) (restore func()) {
	old := modelTransport
	modelTransport = rt
	return func() { modelTransport = old }
}

func modelClient() *http.Client {
	return &http.Client{
		Transport: modelTransport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			switch {
			case len(via) > maxRedirects:
				return fmt.Errorf("more than %d redirects", maxRedirects)
			case req.URL.Scheme != "https":
				return fmt.Errorf("redirected to %s, which is not HTTPS", req.URL.Redacted())
			}
			return nil
		},
	}
}

// DownloadProgress is told how many bytes of a file are downloaded of its
// pinned size.
type DownloadProgress func(file string, done, total int64)

// SetupResult says what EnsureModelFiles did with each file.
type SetupResult struct {
	// Downloaded maps the files Quiver downloaded to their SHA-256.
	Downloaded map[string]string
	// Present lists the files that were already there with the pinned hash,
	// a file the user placed by hand or an earlier setup.
	Present []string
}

// EnsureModelFiles makes the model's files present with their pinned hashes.
// A file that is there with the pinned hash is kept and not downloaded again;
// a file with another hash is refused and left as it is; a missing file is
// downloaded over HTTPS to a temporary file, checked and moved into place.
func (s ModelSpec) EnsureModelFiles(ctx context.Context, progress DownloadProgress) (SetupResult, error) {
	res := SetupResult{Downloaded: map[string]string{}}
	dir, err := ModelDir(s.Name)
	if err != nil {
		return res, err
	}
	for _, f := range s.Files() {
		if f.Size == 0 || f.SHA256 == "" {
			return res, fmt.Errorf("model %s has no pinned %s, so it cannot be downloaded", s.Name, f.Name)
		}
		path := filepath.Join(dir, f.Name)
		got, err := fileSHA256(path)
		switch {
		case err == nil && got == f.SHA256:
			res.Present = append(res.Present, f.Name)
			continue
		case err == nil:
			return res, fmt.Errorf("%s does not have the pinned SHA-256 (%s, not %s): move it away and run the setup again", path, got, f.SHA256)
		case !errors.Is(err, os.ErrNotExist):
			return res, err
		}
		if s.BaseURL == "" {
			return res, fmt.Errorf("model %s has no download address", s.Name)
		}
		if err := downloadModelFile(ctx, s.BaseURL+"/"+f.Name, dir, f, progress); err != nil {
			return res, fmt.Errorf("downloading %s: %w", f.Name, err)
		}
		res.Downloaded[f.Name] = f.SHA256
	}
	return res, nil
}

func downloadModelFile(ctx context.Context, url, dir string, f PinnedFile, progress DownloadProgress) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	if req.URL.Scheme != "https" {
		return fmt.Errorf("%s is not an HTTPS address", req.URL.Redacted())
	}
	resp, err := modelClient().Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s answered %s", req.URL.Host, resp.Status)
	}
	if resp.ContentLength > f.Size {
		return fmt.Errorf("the file is larger than announced (%d bytes, not %d)", resp.ContentLength, f.Size)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, f.Name+".part-*")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(tmp.Name()) }()
	h := sha256.New()
	n, err := io.Copy(io.MultiWriter(tmp, h, &countWriter{file: f.Name, total: f.Size, progress: progress}), io.LimitReader(resp.Body, f.Size+1))
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	switch {
	case err != nil:
		return err
	case n > f.Size:
		return fmt.Errorf("the file is larger than announced (more than %d bytes)", f.Size)
	case n < f.Size:
		return fmt.Errorf("the download was cut after %d of %d bytes", n, f.Size)
	}
	if got := hex.EncodeToString(h.Sum(nil)); got != f.SHA256 {
		return fmt.Errorf("the file does not have the pinned SHA-256 (%s, not %s)", got, f.SHA256)
	}
	return os.Rename(tmp.Name(), filepath.Join(dir, f.Name))
}

type countWriter struct {
	file     string
	done     int64
	total    int64
	progress DownloadProgress
}

func (c *countWriter) Write(p []byte) (int, error) {
	c.done += int64(len(p))
	if c.progress != nil {
		c.progress(c.file, c.done, c.total)
	}
	return len(p), nil
}

// CheckVectorPath proves that search by meaning works: it embeds the name of
// one skill of the library and the skill must be among the first five by
// vector score alone, with no help from the words. The index must be built.
// An empty library has nothing to find; the check then passes with a note.
func (l *Library) CheckVectorPath(modelName string) (note string, err error) {
	spec, ok := models[modelName]
	if !ok {
		return "", fmt.Errorf("unknown model %q", modelName)
	}
	all := l.All()
	if len(all) == 0 {
		return "the library has no skills yet, so there was nothing to find", nil
	}
	target := all[0]
	for _, s := range all {
		if s.ID < target.ID {
			target = s
		}
	}
	m, err := openModel(spec)
	if err != nil {
		return "", err
	}
	q, err := embedder(m, target.Name)
	if err != nil {
		return "", fmt.Errorf("embedding %q: %w", target.Name, err)
	}
	x := loadVectors(spec)
	type scored struct {
		id    string
		score float32
	}
	var ranked []scored
	for _, e := range x.Entries {
		ranked = append(ranked, scored{e.ID, cosine(q, e.Vector)})
	}
	sort.Slice(ranked, func(i, j int) bool {
		if ranked[i].score != ranked[j].score {
			return ranked[i].score > ranked[j].score
		}
		return ranked[i].id < ranked[j].id
	})
	for i, r := range ranked {
		if i >= 5 {
			break
		}
		if r.id == target.ID {
			return fmt.Sprintf("%q found at place %d", target.Name, i+1), nil
		}
	}
	return "", fmt.Errorf("the vector search did not find %q among the first five skills", target.Name)
}

// RemoveIndex deletes the index of vectors.
func RemoveIndex() error {
	path, err := vectorPath()
	if err != nil {
		return err
	}
	err = fsutil.WithLock(func() error { return os.Remove(path) })
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

// RemoveDownloaded deletes the files of the model that the settings file
// records as downloaded by Quiver and that still have the recorded hash. A
// file the user put there, or replaced, is never deleted. It returns what it
// deleted.
func RemoveDownloaded(model string, installed map[string]string) (removed []string, err error) {
	dir, err := ModelDir(model)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(installed))
	for n := range installed {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, name := range names {
		path := filepath.Join(dir, name)
		got, herr := fileSHA256(path)
		if herr != nil || got != installed[name] {
			continue
		}
		if err := os.Remove(path); err != nil {
			return removed, err
		}
		removed = append(removed, path)
	}
	_ = os.Remove(dir) // only when empty
	return removed, nil
}
