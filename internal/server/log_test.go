package server

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOpenLogAppends(t *testing.T) {
	home := t.TempDir()
	t.Setenv("QUIVER_HOME", home)
	for _, msg := range []string{"first", "second"} {
		log, closeLog := openLog()
		log.Info(msg)
		closeLog()
	}
	data, err := os.ReadFile(filepath.Join(home, "logs", "mcp.log"))
	if err != nil {
		t.Fatal(err)
	}
	if s := string(data); !strings.Contains(s, "msg=first") || !strings.Contains(s, "msg=second") || !strings.Contains(s, "pid=") {
		t.Errorf("log:\n%s", s)
	}
}

func TestOpenLogFallsBackToStderr(t *testing.T) {
	home := t.TempDir()
	t.Setenv("QUIVER_HOME", home)
	// A file where the logs folder should be makes the log file fail.
	if err := os.WriteFile(filepath.Join(home, "logs"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	log, closeLog := openLog()
	defer closeLog()
	log.Info("still works")
}
