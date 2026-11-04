package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/kheif/quiver/internal/fsutil"
	"github.com/kheif/quiver/internal/server"
)

// A session start runs one hook command per part, often all at once. Each
// would read the whole library to build the same text, and a skill edited in
// between would give the parts of one session different texts. So part 1
// builds the text and leaves its parts in the cache folder; the parts after
// it take theirs from there, and build the text themselves only when nothing
// is there. Part 1 always builds anew, so a new session never gets an old
// text.

const (
	// hookCacheDir is the cache folder, inside the Quiver home.
	hookCacheDir = "cache/hook"
	// cacheFresh is how long after part 1 wrote it the other parts may use
	// its text.
	cacheFresh = 30 * time.Second
	// cacheWait is how long a part waits for part 1's text to appear.
	cacheWait = 3 * time.Second
	// cacheKeep is how long part 1 keeps old entries.
	cacheKeep = 2 * time.Minute
)

// partCache is the text of one session start, as part 1 built it.
type partCache struct {
	dir, key string
}

// newPartCache returns the cache for a session start, or nil when the Quiver
// home does not exist: a hook never creates it.
func newPartCache(assistant, event, project, session, folder string) *partCache {
	home, err := fsutil.QuiverHome()
	if err != nil {
		return nil
	}
	if info, err := os.Stat(home); err != nil || !info.IsDir() {
		return nil
	}
	sum := sha256.Sum256([]byte(strings.Join([]string{assistant, event, project, session, folder}, "\x00")))
	return &partCache{dir: filepath.Join(home, filepath.FromSlash(hookCacheDir)), key: hex.EncodeToString(sum[:12])}
}

func (c *partCache) file(name string) string {
	return filepath.Join(c.dir, c.key+"."+name)
}

// meta is what the parts after the first need to know besides their text.
type partMeta struct {
	Count    int      `json:"count"`
	Problems []string `json:"problems,omitempty"`
}

// write leaves the parts of sc in the cache, the meta file last: a part that
// finds it finds every part.
func (c *partCache) write(sc server.SessionContext) {
	if err := os.MkdirAll(c.dir, 0o700); err != nil {
		return
	}
	c.sweep()
	for i, text := range sc.Parts {
		if err := fsutil.WriteFileAtomic(c.file(strconv.Itoa(i+1)), []byte(text), 0o600); err != nil {
			return
		}
	}
	data, _ := json.Marshal(partMeta{Count: len(sc.Parts), Problems: sc.Problems})
	_ = fsutil.WriteFileAtomic(c.file("meta"), data, 0o600)
}

// read returns part n of the text that part 1 built, waiting up to wait for
// it. ok is false when there is none, or it is not fresh.
func (c *partCache) read(n int, wait time.Duration) (text string, meta partMeta, ok bool) {
	deadline := time.Now().Add(wait)
	for {
		if info, err := os.Stat(c.file("meta")); err == nil && time.Since(info.ModTime()) < cacheFresh {
			data, err := os.ReadFile(c.file("meta"))
			if err == nil && json.Unmarshal(data, &meta) == nil {
				if n > meta.Count {
					return "", meta, true // no such part: nothing to print
				}
				if b, err := os.ReadFile(c.file(strconv.Itoa(n))); err == nil {
					return string(b), meta, true
				}
			}
		}
		if time.Now().After(deadline) {
			return "", partMeta{}, false
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// sweep removes entries older than cacheKeep.
func (c *partCache) sweep() {
	entries, err := os.ReadDir(c.dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if info, err := e.Info(); err == nil && time.Since(info.ModTime()) > cacheKeep {
			_ = os.Remove(filepath.Join(c.dir, e.Name()))
		}
	}
}
