package source

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	cases := []struct{ in, url, name string }{
		{"anthropics/skills", "https://github.com/anthropics/skills.git", "anthropics"},
		{"anthropics/skills.git", "https://github.com/anthropics/skills.git", "anthropics"},
		{" JuliusBrussee/caveman ", "https://github.com/JuliusBrussee/caveman.git", "juliusbrussee"},
		{"github.com/a/b", "https://github.com/a/b.git", "a"},
		{"gitlab.com/group/sub/b", "https://gitlab.com/group/sub/b.git", "group"},
		{"https://gitlab.com/a/b", "https://gitlab.com/a/b.git", "a"},
		{"https://gitlab.com/a/b/", "https://gitlab.com/a/b.git", "a"},
		{"https://github.com/a/b.git", "https://github.com/a/b.git", "a"},
		{"https://git.example.com/skills", "https://git.example.com/skills.git", "skills"},
		{"git@github.com:a/b.git", "git@github.com:a/b.git", "a"},
		{"git@github.com:My_Org/b.git", "git@github.com:My_Org/b.git", "my-org"},
		{"ssh://git@host:22/team/skills.git", "ssh://git@host:22/team/skills.git", "team"},
		{"file:///srv/repos/skills.git", "file:///srv/repos/skills.git", "skills"},
	}
	for _, c := range cases {
		got, err := Parse(c.in)
		if err != nil || got.URL != c.url || got.Name != c.name {
			t.Errorf("Parse(%q) = %+v, %v; want %s, %s", c.in, got, err, c.url, c.name)
		}
	}
}

func TestParseLocalPath(t *testing.T) {
	home, err := filepath.EvalSymlinks(t.TempDir()) // macOS: /var is /private/var
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home) // os.UserHomeDir on Windows
	chdir(t, home)
	// On Windows /srv is on the current drive.
	abs := func(p string) string { a, _ := filepath.Abs(p); return a }
	cases := []struct{ in, url, name string }{
		{"/srv/repos/Team_Skills.git", abs("/srv/repos/Team_Skills.git"), "team-skills"},
		{"/srv/repos/skills/", abs("/srv/repos/skills"), "skills"},
		{"./work/skills", filepath.Join(home, "work", "skills"), "skills"},
		{"../up", filepath.Join(filepath.Dir(home), "up"), "up"},
		{"~/code/my-skills", filepath.Join(home, "code", "my-skills"), "my-skills"},
	}
	if runtime.GOOS == "windows" {
		cases = append(cases,
			struct{ in, url, name string }{`C:\repos\Team_Skills.git`, `C:\repos\Team_Skills.git`, "team-skills"},
			struct{ in, url, name string }{`.\work\skills`, filepath.Join(home, "work", "skills"), "skills"},
		)
	}
	for _, c := range cases {
		got, err := Parse(c.in)
		if err != nil || got.URL != c.url || got.Name != c.name {
			t.Errorf("Parse(%q) = %+v, %v; want %s, %s", c.in, got, err, c.url, c.name)
		}
	}
	if _, err := Parse("/"); err == nil || !strings.Contains(err.Error(), "--name") {
		t.Errorf("root: %v", err)
	}
}

// chdir changes the working folder for one test. t.Chdir needs Go 1.24.
func chdir(t *testing.T, dir string) {
	t.Helper()
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(old) })
}

func TestParseRejects(t *testing.T) {
	cases := map[string]string{
		"":                             "empty",
		"--upload-pack=touch x":        "must not start with '-'",
		"ext::sh -c touch% /tmp/pwned": "transport helper",
		"fd::17":                       "transport helper",
		"ftp://host/a/b":               `scheme "ftp"`,
		"https://github.com":           "cannot read",
		"skills":                       "cannot read",
		"a/b/c":                        "cannot read",
		"a b/c":                        "cannot read",
		"https://github.com/---/x":     "--name",
	}
	for in, want := range cases {
		if _, err := Parse(in); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("Parse(%q) = %v, want %q", in, err, want)
		}
	}
}
