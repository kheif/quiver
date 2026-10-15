package source

import (
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
