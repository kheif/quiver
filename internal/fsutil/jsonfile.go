package fsutil

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// ErrNotExist is returned (wrapped) by ReadJSON when the file is missing.
var ErrNotExist = fs.ErrNotExist

// rename is replaced in tests to simulate a crash before the final rename.
var rename = os.Rename

// WriteJSONAtomic writes v as indented JSON to path. It writes a temp file in
// the same folder, syncs it, then renames it over path, so a crash leaves
// either the old or the new file, never a partial one.
func WriteJSONAtomic(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("encode %s: %w", path, err)
	}
	return WriteFileAtomic(path, append(data, '\n'), 0o644)
}

// WriteFileAtomic writes data to path with perm the way WriteJSONAtomic
// does: through a synced temp file in the same folder that is renamed over
// path.
func WriteFileAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(tmp.Name()) }() // no-op after a successful rename

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmp.Name(), perm); err != nil {
		return err
	}
	return rename(tmp.Name(), path)
}

// ReadJSON decodes the JSON file at path into v. A missing file returns an
// error that matches errors.Is(err, ErrNotExist).
func ReadJSON(path string, v any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%w: %s", ErrNotExist, path)
		}
		return err
	}
	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}
	return nil
}
