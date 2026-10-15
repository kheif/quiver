// Package project identifies projects by the .quiver.json file at their root.
package project

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
