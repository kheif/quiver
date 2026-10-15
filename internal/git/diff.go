package git

import "strings"

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
