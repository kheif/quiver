// Package project identifies projects by the .quiver.json file at their root.
package project

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kheif/quiver/internal/fsutil"
)

// FileName is the file that marks a project root.
const FileName = ".quiver.json"

// ErrNotRegistered is returned by Find when no .quiver.json is found.
var ErrNotRegistered = errors.New("not a Quiver project")

// Project is a folder with a .quiver.json.
type Project struct {
	ID   string
	Root string
}

// Find walks up from dir to the first folder containing .quiver.json.
// It stops before $HOME and at the file system root, so a stray
// ~/.quiver.json never turns every folder into one project.
func Find(dir string) (*Project, error) {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	home, _ := os.UserHomeDir()
	for d := range parents(dir, home) {
		path := filepath.Join(d, FileName)
		if _, err := os.Stat(path); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return nil, err
		}
		id, err := readID(path)
		if err != nil {
			return nil, err
		}
		return &Project{ID: id, Root: d}, nil
	}
	return nil, ErrNotRegistered
}

// Register returns the project that dir belongs to, creating it if needed.
// An existing .quiver.json found by Find wins. Otherwise the root is the
// nearest folder with .git, or dir itself, and a new ID is written there.
func Register(dir string) (*Project, error) {
	p, err := Find(dir)
	if err == nil || !errors.Is(err, ErrNotRegistered) {
		return p, err
	}
	dir, err = filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	home, _ := os.UserHomeDir()
	if home != "" && dir == filepath.Clean(home) {
		return nil, errors.New("cannot register the home directory as a project")
	}
	root := dir
	for d := range parents(dir, home) {
		if _, err := os.Stat(filepath.Join(d, ".git")); err == nil {
			root = d
			break
		}
	}
	id := newID(filepath.Base(root))
	if err := SetID(root, id); err != nil {
		return nil, err
	}
	return &Project{ID: id, Root: root}, nil
}

// SetID writes id to root/.quiver.json. Other keys in an existing file are
// kept.
func SetID(root, id string) error {
	if !validID(id) {
		return fmt.Errorf("invalid project id %q", id)
	}
	path := filepath.Join(root, FileName)
	fields, err := readFile(path)
	if errors.Is(err, os.ErrNotExist) {
		fields = map[string]json.RawMessage{}
	} else if err != nil {
		return err
	}
	raw, err := json.Marshal(id)
	if err != nil {
		return err
	}
	fields["id"] = raw
	return fsutil.WriteJSONAtomic(path, fields)
}

// idChars is Crockford base32: 32 chars, no i/l/o/u, so 0 and 1 are unambiguous.
const idChars = "0123456789abcdefghjkmnpqrstvwxyz"

// newID returns "<name>-<6 random chars>", with name reduced to lowercase
// letters, digits and dashes.
func newID(folder string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(folder) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else {
			b.WriteRune('-')
		}
	}
	name := strings.Trim(b.String(), "-")
	if name == "" {
		name = "project"
	}
	suffix := make([]byte, 6)
	_, _ = rand.Read(suffix) // never fails on supported platforms
	for i, c := range suffix {
		suffix[i] = idChars[int(c)%len(idChars)]
	}
	return name + "-" + string(suffix)
}

// parents yields dir and each parent folder, stopping before home and
// after the file system root.
func parents(dir, home string) func(yield func(string) bool) {
	return func(yield func(string) bool) {
		for {
			if home != "" && dir == filepath.Clean(home) {
				return
			}
			if !yield(dir) {
				return
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				return
			}
			dir = parent
		}
	}
}

// readFile reads .quiver.json with all its keys.
func readFile(path string) (map[string]json.RawMessage, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return nil, fmt.Errorf("%s: invalid JSON: %w", path, err)
	}
	if fields == nil {
		return nil, fmt.Errorf("%s: invalid JSON: want an object", path)
	}
	return fields, nil
}

// readID reads the id from .quiver.json. A missing or invalid id is an error
// that names the file.
func readID(path string) (string, error) {
	fields, err := readFile(path)
	if err != nil {
		return "", err
	}
	raw, ok := fields["id"]
	if !ok {
		return "", fmt.Errorf("%s: id is missing", path)
	}
	var id string
	if err := json.Unmarshal(raw, &id); err != nil || !validID(id) {
		return "", fmt.Errorf("%s: id must be a non-empty string without spaces or slashes", path)
	}
	return id, nil
}

func validID(id string) bool {
	return id != "" && !strings.ContainsAny(id, " \t\r\n/\\")
}
