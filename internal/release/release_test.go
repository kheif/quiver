package release

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

// fakeGitHub serves tag as the latest release, or fails with status. It
// returns the number of requests.
func fakeGitHub(t *testing.T, tag string, status int) *atomic.Int32 {
	t.Helper()
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		if r.Header.Get("User-Agent") != "quiver/test" {
			t.Errorf("User-Agent %q", r.Header.Get("User-Agent"))
		}
		if status != http.StatusOK {
			w.WriteHeader(status)
			return
		}
		_, _ = w.Write([]byte(`{"tag_name": "` + tag + `", "name": "Quiver"}`))
	}))
	t.Cleanup(srv.Close)
	old := LatestURL
	LatestURL = srv.URL
	t.Cleanup(func() { LatestURL = old })
	return &hits
}

func home(t *testing.T) string {
	t.Helper()
	h := t.TempDir()
	t.Setenv("QUIVER_HOME", h)
	return h
}

func TestCheckOnceADay(t *testing.T) {
	h := home(t)
	hits := fakeGitHub(t, "v1.3.0", http.StatusOK)
	now := time.Date(2026, 10, 31, 12, 0, 0, 0, time.UTC)
	for _, c := range []struct {
		at   time.Time
		hits int32
	}{
		{now, 1},
		{now.Add(23 * time.Hour), 1},
		{now.Add(25 * time.Hour), 2},
		{now.Add(-time.Hour), 3}, // the clock went back
	} {
		s, err := Check(context.Background(), c.at, "quiver/test")
		if err != nil || s.Latest != "1.3.0" {
			t.Errorf("at %v: %+v %v", c.at, s, err)
		}
		if got := hits.Load(); got != c.hits {
			t.Errorf("at %v: %d requests, want %d", c.at, got, c.hits)
		}
	}
	if _, err := os.Stat(filepath.Join(h, StateFile)); err != nil {
		t.Error(err)
	}
}

func TestCheckFailed(t *testing.T) {
	home(t)
	now := time.Now()
	fakeGitHub(t, "v1.3.0", http.StatusOK)
	if _, err := Check(context.Background(), now, "quiver/test"); err != nil {
		t.Fatal(err)
	}
	// A day later GitHub fails: the version found before stays, and it is
	// not asked again the same day.
	hits := fakeGitHub(t, "", http.StatusInternalServerError)
	later := now.Add(Interval)
	s, err := Check(context.Background(), later, "quiver/test")
	if err == nil || s.Latest != "1.3.0" || !s.Checked.Equal(later.UTC()) {
		t.Errorf("failed check: %+v %v", s, err)
	}
	if _, err := Check(context.Background(), later.Add(time.Hour), "quiver/test"); err != nil || hits.Load() != 1 {
		t.Errorf("asked again: %d requests, %v", hits.Load(), err)
	}
}

func TestCheckBadAnswer(t *testing.T) {
	home(t)
	fakeGitHub(t, "nightly", http.StatusOK)
	if s, err := Check(context.Background(), time.Now(), "quiver/test"); err == nil || s.Latest != "" {
		t.Errorf("got %+v %v", s, err)
	}
}

func TestCheckNoHome(t *testing.T) {
	t.Setenv("QUIVER_HOME", filepath.Join(t.TempDir(), "missing"))
	hits := fakeGitHub(t, "v1.3.0", http.StatusOK)
	if _, err := Check(context.Background(), time.Now(), "quiver/test"); err == nil || hits.Load() != 0 {
		t.Errorf("%d requests, %v", hits.Load(), err)
	}
}

func TestCheckBrokenState(t *testing.T) {
	h := home(t)
	path := filepath.Join(h, StateFile)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	fakeGitHub(t, "v1.3.0", http.StatusOK)
	if s, err := Check(context.Background(), time.Now(), "quiver/test"); err != nil || s.Latest != "1.3.0" {
		t.Errorf("got %+v %v", s, err)
	}
	if s, err := Load(); err != nil || s.Latest != "1.3.0" {
		t.Errorf("not replaced: %+v %v", s, err)
	}
}

func TestNewer(t *testing.T) {
	for _, c := range []struct {
		latest, current string
		want            bool
	}{
		{"1.3.0", "1.2.0", true},
		{"1.2.1", "1.2.0", true},
		{"2.0.0", "1.10.0", true},
		{"1.10.0", "1.9.0", true},
		{"1.2.0", "1.2.0", false},
		{"1.1.0", "1.2.0", false},
		{"1.2.0", "1.2.0-rc.1", true},
		{"1.2.0-rc.2", "1.2.0", false},
		{"", "1.2.0", false},
		{"1.3.0", "dev", false},
		{"1.3", "1.2.0", false},
	} {
		if got := Newer(c.latest, c.current); got != c.want {
			t.Errorf("Newer(%q, %q) = %v", c.latest, c.current, got)
		}
	}
}
