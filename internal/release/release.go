// Package release finds out whether a newer Quiver release is out. It asks
// GitHub at most once a day and keeps the answer in the Quiver home.
package release

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/kheif/quiver/internal/fsutil"
)

// StateFile is where the last check is kept, relative to the Quiver home.
const StateFile = "state/version-check.json"

// Interval is how often GitHub is asked.
const Interval = 24 * time.Hour

// Timeout limits one request to GitHub.
const Timeout = 2 * time.Second

// LatestURL returns the latest release. Tests replace it.
var LatestURL = "https://api.github.com/repos/kheif/quiver/releases/latest"

// State is the result of the last check.
type State struct {
	Checked time.Time `json:"checked"`
	// Latest is the version of the latest release, without the leading v.
	// A failed check keeps the one found before.
	Latest string `json:"latest,omitempty"`
}

// Load returns the saved state; the zero State when there is none.
func Load() (State, error) {
	var s State
	path, err := statePath()
	if err != nil {
		return s, err
	}
	if err := fsutil.ReadJSON(path, &s); err != nil && !errors.Is(err, os.ErrNotExist) {
		return State{}, err
	}
	return s, nil
}

// Due reports whether the last check is older than Interval. A time in the
// future, from a clock that was wrong, counts as old.
func (s State) Due(now time.Time) bool {
	return s.Checked.IsZero() || now.Sub(s.Checked) >= Interval || s.Checked.After(now)
}

// Check asks GitHub for the latest release when the last check is due, and
// saves the result. A failed request is saved too, so it is tried again
// only the next day; its error is returned with the state found before.
// Without a Quiver home it saves nothing.
func Check(ctx context.Context, now time.Time, userAgent string) (State, error) {
	// A broken state file is replaced.
	s, err := Load()
	if err == nil && !s.Due(now) {
		return s, nil
	}
	if err != nil {
		s = State{}
	}
	home, err := fsutil.QuiverHome()
	if err != nil {
		return s, err
	}
	if _, err := os.Stat(home); err != nil {
		return s, err
	}
	latest, fetchErr := Fetch(ctx, userAgent)
	s.Checked = now.UTC()
	if fetchErr == nil {
		s.Latest = latest
	}
	path := filepath.Join(home, StateFile)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return s, err
	}
	if err := fsutil.WriteJSONAtomic(path, s); err != nil {
		return s, err
	}
	return s, fetchErr
}

// Fetch returns the version of the latest release, without the leading v.
func Fetch(ctx context.Context, userAgent string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, Timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, LatestURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", userAgent)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("latest release: %s", resp.Status)
	}
	var body struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", fmt.Errorf("latest release: %w", err)
	}
	v := strings.TrimPrefix(body.TagName, "v")
	if _, ok := parse(v); !ok {
		return "", fmt.Errorf("latest release: unknown version %q", body.TagName)
	}
	return v, nil
}

// Newer reports whether version latest is newer than current. A version
// that is not major.minor.patch is never newer, and nothing is newer than
// it.
func Newer(latest, current string) bool {
	l, ok1 := parse(latest)
	c, ok2 := parse(current)
	if !ok1 || !ok2 {
		return false
	}
	for i := range 3 {
		if l.core[i] != c.core[i] {
			return l.core[i] > c.core[i]
		}
	}
	// 1.2.0 is newer than 1.2.0-rc.1.
	return l.pre == "" && c.pre != ""
}

type semver struct {
	core [3]int
	pre  string
}

// parse reads major.minor.patch with an optional -prerelease and +build.
func parse(v string) (semver, bool) {
	var s semver
	v, _, _ = strings.Cut(v, "+")
	v, s.pre, _ = strings.Cut(v, "-")
	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return s, false
	}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return s, false
		}
		s.core[i] = n
	}
	return s, true
}

func statePath() (string, error) {
	home, err := fsutil.QuiverHome()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, StateFile), nil
}
