package cli

import (
	"io"
	"os"

	"golang.org/x/term"
)

// terminal returns the width of out when it is a terminal. Tests replace it.
var terminal = func(out io.Writer) (width int, ok bool) {
	f, isFile := out.(*os.File)
	if !isFile || !term.IsTerminal(int(f.Fd())) {
		return 0, false
	}
	w, _, err := term.GetSize(int(f.Fd()))
	if err != nil {
		return 0, true
	}
	return w, true
}

// dim returns s in dim text when out is a terminal and NO_COLOR is not set.
func dim(out io.Writer, s string) string {
	if _, ok := terminal(out); !ok || os.Getenv("NO_COLOR") != "" {
		return s
	}
	return "\x1b[2m" + s + "\x1b[0m"
}
