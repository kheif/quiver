package library

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// settingsHome points the Quiver home at a new folder and returns the path
// of the settings file in it.
func settingsHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("QUIVER_HOME", home)
	return filepath.Join(home, "state", "search.json")
}

func writeSettings(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadEngine(t *testing.T) {
	path := settingsHome(t)
	if got := LoadEngine(); got != (EngineSetting{Engine: "words"}) {
		t.Errorf("missing file: %+v", got)
	}
	for _, content := range []string{`{"version": 1}`, `{"version": 1, "engine": ""}`, `{"version": 1, "engine": "words"}`} {
		writeSettings(t, path, content)
		if got := LoadEngine(); got != (EngineSetting{Engine: "words"}) {
			t.Errorf("%s: %+v", content, got)
		}
	}
}

func TestLoadEngineUnknownValue(t *testing.T) {
	path := settingsHome(t)
	writeSettings(t, path, `{"version": 1, "engine": "telepathy"}`)
	got := LoadEngine()
	if got.Engine != "words" || got.Refused || !strings.Contains(got.Problem, `unknown engine "telepathy"`) {
		t.Errorf("%+v", got)
	}
}

func TestLoadEngineNewerVersion(t *testing.T) {
	path := settingsHome(t)
	const content = `{"version": 2, "engine": "local"}`
	writeSettings(t, path, content)
	got := LoadEngine()
	want := "search.json has version 2, but this quiver supports only up to 1: update quiver"
	if got.Engine != "words" || !got.Refused || !strings.HasPrefix(got.Problem, want) {
		t.Errorf("%+v", got)
	}
	err := UseEngine("words")
	if err == nil || err.Error() != want {
		t.Errorf("use: %v", err)
	}
	if data, _ := os.ReadFile(path); string(data) != content {
		t.Errorf("the file was rewritten: %s", data)
	}
}

func TestLoadEngineBrokenFile(t *testing.T) {
	path := settingsHome(t)
	writeSettings(t, path, `{not json`)
	if got := LoadEngine(); got.Engine != "words" || got.Problem == "" {
		t.Errorf("%+v", got)
	}
	if err := UseEngine("words"); err == nil {
		t.Error("use replaced a broken file")
	}
}

func TestUseEngine(t *testing.T) {
	path := settingsHome(t)
	if err := UseEngine("words"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil || !strings.Contains(string(data), `"version": 1`) || !strings.Contains(string(data), `"engine": "words"`) {
		t.Errorf("%s, %v", data, err)
	}
	// Other keys stay.
	writeSettings(t, path, `{"version": 1, "engine": "words", "later": {"a": 1}}`)
	if err := UseEngine("words"); err != nil {
		t.Fatal(err)
	}
	if data, _ := os.ReadFile(path); !strings.Contains(string(data), `"later"`) {
		t.Errorf("a key was dropped: %s", data)
	}
	err = UseEngine("telepathy")
	if err == nil || !strings.Contains(err.Error(), `unknown search engine "telepathy": use local or words`) {
		t.Errorf("unknown: %v", err)
	}
}

func TestUseEngineWritesNothingForAnUnknownValue(t *testing.T) {
	path := settingsHome(t)
	if err := UseEngine("telepathy"); err == nil {
		t.Fatal("no error")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("a file was written: %v", err)
	}
}

func TestNewSearchReadsTheSettings(t *testing.T) {
	path := settingsHome(t)
	l := libraryOf(nil)
	s, err := l.NewSearch("")
	if err != nil || s.Engine != "words" || s.Problem != "" {
		t.Errorf("%+v, %v", s, err)
	}
	writeSettings(t, path, `{"version": 9}`)
	if s, err = l.NewSearch(""); err != nil || s.Engine != "words" || !strings.Contains(s.Problem, "update quiver") {
		t.Errorf("%+v, %v", s, err)
	}
}
