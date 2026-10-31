package git

import (
	"strconv"
	"strings"
)

// DiffNames returns the paths that changed between two commits.
func DiffNames(dir, from, to string) ([]string, error) {
	out, err := run(dir, "diff", "--name-only", "--no-renames", from, to, "--")
	if err != nil || out == "" {
		return nil, err
	}
	return strings.Split(out, "\n"), nil
}

// Diff returns the patch between two commits, limited to paths when given.
func Diff(dir, from, to string, paths ...string) (string, error) {
	args := append([]string{"diff", "--no-color", "--no-ext-diff", from, to, "--"}, paths...)
	return run(dir, args...)
}

// FileChange is one file that changed between two commits.
type FileChange struct {
	Path string
	// Status is A (added), M (modified), D (deleted) or T (type changed).
	Status string
	// Mode is the file mode after the change, such as 100644 or 100755;
	// the mode before it for a deleted file.
	Mode           string
	Added, Deleted int
	Binary         bool
}

// DiffFiles lists the files that changed between two commits, limited to
// paths when given, with their status, mode and line counts.
func DiffFiles(dir, from, to string, paths ...string) ([]FileChange, error) {
	args := append([]string{"diff", "--raw", "--numstat", "-z", "--no-renames", from, to, "--"}, paths...)
	out, err := run(dir, args...)
	if err != nil || out == "" {
		return nil, err
	}
	var files []FileChange
	byPath := map[string]int{}
	fields := strings.Split(out, "\x00")
	for i := 0; i < len(fields); i++ {
		f := fields[i]
		switch {
		case strings.HasPrefix(f, ":") && i+1 < len(fields):
			// :<old mode> <new mode> <old sha> <new sha> <status>, then the path.
			parts := strings.Fields(f[1:])
			if len(parts) < 5 {
				continue
			}
			i++
			c := FileChange{Path: fields[i], Status: parts[4][:1], Mode: parts[1]}
			if c.Status == "D" {
				c.Mode = parts[0]
			}
			byPath[c.Path] = len(files)
			files = append(files, c)
		case strings.Count(f, "\t") == 2:
			// <added>\t<deleted>\t<path>, "-" for binary files.
			parts := strings.SplitN(f, "\t", 3)
			j, ok := byPath[parts[2]]
			if !ok {
				continue
			}
			if parts[0] == "-" {
				files[j].Binary = true
				continue
			}
			files[j].Added, _ = strconv.Atoi(parts[0])
			files[j].Deleted, _ = strconv.Atoi(parts[1])
		}
	}
	return files, nil
}
