package library

import (
	"path/filepath"

	"github.com/kheif/quiver/internal/fsutil"
	"github.com/kheif/quiver/internal/git"
	"github.com/kheif/quiver/internal/source"
)

// IndexFile caches discovery results inside each source's .git folder, so
// it never shows in git status or clashes with a file from upstream.
const IndexFile = "quiver-index.json"

// indexVersion must change whenever discovery rules change, so old caches
// are rebuilt.
const indexVersion = 1

type index struct {
	Version int    `json:"version"`
	Head    string `json:"head"`
	source.Discovery
}

// scan runs discovery. Tests replace it to count scans.
var scan = source.Discover

// discover returns the skills in a source repository. It reuses the cached
// index while HEAD is unchanged. A folder that is not a git repository is
// scanned every time.
func discover(dir string) (*source.Discovery, error) {
	head, err := git.Head(dir)
	if err != nil {
		return scan(dir)
	}
	path := filepath.Join(dir, ".git", IndexFile)
	var cached index
	if fsutil.ReadJSON(path, &cached) == nil && cached.Version == indexVersion && cached.Head == head {
		return &cached.Discovery, nil
	}
	d, err := scan(dir)
	if err != nil {
		return nil, err
	}
	// A failed write only means the next load scans again.
	_ = fsutil.WriteJSONAtomic(path, index{Version: indexVersion, Head: head, Discovery: *d})
	return d, nil
}
