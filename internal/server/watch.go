package server

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/kheif/quiver/internal/fsutil"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// debounce is how long the watcher waits after the last change before it
// reloads. One quiver command or git pull writes many files.
var debounce = 300 * time.Millisecond

// watch reloads the skill prompts when the Quiver home changes, until ctx
// ends. Content changes need no reload, because get_skill and the prompts
// read skills from disk; a reload only matters when the served skills or
// their descriptions change. It returns once the watches are in place.
//
// fsnotify does not watch subfolders, so it watches: the home folder (for
// registry.json and config.json), local/ and each skill folder in it,
// sources/, and each source's .git folder, which a pull or reset changes.
func watch(ctx context.Context, s *mcp.Server, sess *session, p *prompts, log *slog.Logger) error {
	home, err := fsutil.QuiverHome()
	if err != nil {
		return err
	}
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	local, sources := filepath.Join(home, "local"), filepath.Join(home, "sources")
	dirs := []string{home, local, sources}
	dirs = append(dirs, subdirs(local, "")...)
	dirs = append(dirs, subdirs(sources, ".git")...)
	for _, d := range dirs {
		if err := w.Add(d); err != nil {
			_ = w.Close()
			return fmt.Errorf("watch %s: %w", d, err)
		}
	}
	reload := func() {
		// recoverPanics covers requests only, not this goroutine.
		defer func() {
			if r := recover(); r != nil {
				log.Error("reload panicked", "panic", r)
			}
		}()
		changed, err := p.sync(s, sess.projectID())
		if err != nil {
			log.Warn("reload failed; keeping the last skill list", "err", err)
			return
		}
		if changed {
			log.Info("skill list changed")
		}
	}
	go func() {
		defer func() { _ = w.Close() }()
		var tick <-chan time.Time
		for {
			select {
			case <-ctx.Done():
				return
			case ev, ok := <-w.Events:
				if !ok {
					return
				}
				if relevant(ev, home, local, sources) {
					follow(w, ev, local, sources, log)
					tick = time.After(debounce)
				}
			case err, ok := <-w.Errors:
				if !ok {
					return
				}
				log.Warn("watch error", "err", err)
			case <-tick:
				tick = nil
				reload()
			}
		}
	}()
	return nil
}

// relevant reports whether ev can change the served skills. Hidden names
// are skipped: temp files, the lock, and folders that are still being
// written, such as local/.new-*.
func relevant(ev fsnotify.Event, home, local, sources string) bool {
	dir, name := filepath.Split(ev.Name)
	dir = filepath.Clean(dir)
	switch {
	case dir == home:
		return name == "registry.json" || name == "config.json"
	case dir == sources, filepath.Base(dir) == ".git":
		return true
	}
	return !strings.HasPrefix(name, ".")
}

// follow starts watching a folder that was just created: a new skill in
// local/, or the .git folder of a new source.
func follow(w *fsnotify.Watcher, ev fsnotify.Event, local, sources string, log *slog.Logger) {
	if !ev.Has(fsnotify.Create) {
		return
	}
	dir := filepath.Dir(ev.Name)
	var target string
	switch dir {
	case local:
		target = ev.Name
	case sources:
		target = filepath.Join(ev.Name, ".git")
	default:
		return
	}
	if info, err := os.Stat(target); err != nil || !info.IsDir() || strings.HasPrefix(filepath.Base(ev.Name), ".") {
		return
	}
	if err := w.Add(target); err != nil {
		log.Warn("could not watch", "dir", target, "err", err)
	}
}

// subdirs returns each visible folder in dir, joined with sub. Folders
// where that path does not exist are left out.
func subdirs(dir, sub string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		p := filepath.Join(dir, e.Name(), sub)
		if info, err := os.Stat(p); err == nil && info.IsDir() {
			out = append(out, p)
		}
	}
	return out
}
