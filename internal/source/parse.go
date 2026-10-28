// Package source turns what the user types after `quiver source add` into a
// git clone URL and a default source name.
package source

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Parsed is the result of Parse.
type Parsed struct {
	URL  string // passed to git clone
	Name string // default source name; valid for config.ValidateSourceName
}

var (
	segmentRE = regexp.MustCompile(`^[A-Za-z0-9_.-]+$`)
	scpRE     = regexp.MustCompile(`^[A-Za-z0-9_.-]+@[A-Za-z0-9_.-]+:(.+)$`)
	schemes   = map[string]bool{"https": true, "http": true, "ssh": true, "git": true, "file": true}
)

// Parse reads input as owner/repo (GitHub), host/owner/repo, a git URL, an
// scp-style address such as git@github.com:a/b.git, or a local path: an
// absolute one (C:\ on Windows), or one that starts with ./, ../ or ~/.
func Parse(input string) (Parsed, error) {
	in := strings.TrimSpace(input)
	if err := checkSafe(in); err != nil {
		return Parsed{}, err
	}
	switch {
	case isLocalPath(in):
		return parseLocal(in)
	case strings.Contains(in, "://"):
		return parseURL(in)
	case scpRE.MatchString(in):
		return withName(in, scpRE.FindStringSubmatch(in)[1])
	}
	parts := strings.Split(strings.Trim(in, "/"), "/")
	for _, p := range parts {
		if !segmentRE.MatchString(p) {
			return Parsed{}, unreadable(in)
		}
	}
	switch {
	case len(parts) == 2 && !strings.Contains(parts[0], "."):
		return withName("https://github.com/"+parts[0]+"/"+withGit(parts[1]), parts[0]+"/"+parts[1])
	case len(parts) >= 3 && strings.Contains(parts[0], "."):
		path := strings.Join(parts[1:len(parts)-1], "/") + "/" + withGit(parts[len(parts)-1])
		return withName("https://"+parts[0]+"/"+path, path)
	}
	return Parsed{}, unreadable(in)
}

// checkSafe rejects input that git could read as an option or as a
// transport helper (ext::, fd::), both of which can run commands.
func checkSafe(in string) error {
	switch {
	case in == "":
		return fmt.Errorf("source is empty")
	case strings.HasPrefix(in, "-"):
		return fmt.Errorf("source %q must not start with '-'", in)
	case strings.Contains(in, "::"):
		return fmt.Errorf("source %q uses a git transport helper, which quiver does not allow", in)
	}
	return nil
}

func isLocalPath(in string) bool {
	if filepath.IsAbs(in) {
		return true
	}
	prefixes := []string{"/", "./", "../", "~/"}
	if filepath.Separator == '\\' {
		prefixes = append(prefixes, `.\`, `..\`, `~\`)
	}
	for _, prefix := range prefixes {
		if strings.HasPrefix(in, prefix) {
			return true
		}
	}
	return in == "." || in == ".." || in == "~"
}

// parseLocal makes the path absolute, because git clone runs from another
// folder. The default name is the folder name.
func parseLocal(in string) (Parsed, error) {
	path := in
	if path == "~" || strings.HasPrefix(path, "~/") || strings.HasPrefix(path, "~"+string(filepath.Separator)) {
		home, err := os.UserHomeDir()
		if err != nil {
			return Parsed{}, err
		}
		path = filepath.Join(home, path[1:])
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return Parsed{}, err
	}
	return withName(abs, filepath.Base(abs))
}

func parseURL(in string) (Parsed, error) {
	u, err := url.Parse(in)
	if err != nil {
		return Parsed{}, fmt.Errorf("source %q: %w", in, err)
	}
	if !schemes[u.Scheme] {
		return Parsed{}, fmt.Errorf("source %q: scheme %q is not supported (use https, ssh, git or file)", in, u.Scheme)
	}
	path := strings.Trim(u.Path, "/")
	if path == "" {
		return Parsed{}, unreadable(in)
	}
	switch u.Scheme {
	case "https", "http":
		u.Path = "/" + withGit(path)
		return withName(u.String(), path)
	case "file":
		return withName(in, path[strings.LastIndex(path, "/")+1:])
	}
	return withName(in, path)
}

// withName returns url with the default name taken from repoPath: the owner
// when there is one, otherwise the repository name.
func withName(url, repoPath string) (Parsed, error) {
	parts := strings.Split(strings.Trim(repoPath, "/"), "/")
	name := parts[0]
	if len(parts) == 1 {
		name = strings.TrimSuffix(name, ".git")
	}
	clean := sanitize(name)
	if clean == "" {
		return Parsed{}, fmt.Errorf("cannot pick a name for %q: pass one with --name", url)
	}
	return Parsed{URL: url, Name: clean}, nil
}

func withGit(repo string) string {
	if strings.HasSuffix(repo, ".git") {
		return repo
	}
	return repo + ".git"
}

var invalidRE = regexp.MustCompile(`[^a-z0-9]+`)

// sanitize lowercases s and replaces runs of other characters with "-".
func sanitize(s string) string {
	return strings.Trim(invalidRE.ReplaceAllString(strings.ToLower(s), "-"), "-")
}

func unreadable(in string) error {
	return fmt.Errorf("cannot read %q as a source: use owner/repo, a git URL or a local path", in)
}
