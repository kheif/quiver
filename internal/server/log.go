package server

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/kheif/quiver/internal/fsutil"
)

// LogFile is the server log inside the Quiver home. Every server process
// appends to it.
const LogFile = "logs/mcp.log"

// openLog returns a logger that writes to stderr and to LogFile, and a
// function that closes the file. If the file cannot be opened, the logger
// writes to stderr only and says why.
func openLog() (*slog.Logger, func()) {
	file, err := openLogFile()
	if err != nil {
		log := slog.New(slog.NewTextHandler(os.Stderr, nil))
		log.Warn("logging to stderr only", "err", err)
		return log, func() {}
	}
	// The file comes first: MultiWriter stops at the first failed write, and
	// a closed stderr must not stop the file log.
	w := io.MultiWriter(file, os.Stderr)
	log := slog.New(slog.NewTextHandler(w, nil)).With("pid", os.Getpid())
	return log, func() { _ = file.Close() }
}

func openLogFile() (*os.File, error) {
	home, err := fsutil.QuiverHome()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(home, filepath.FromSlash(LogFile))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	return os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
}
