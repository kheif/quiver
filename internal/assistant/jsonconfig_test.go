package assistant

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func testAssistant(t *testing.T, content string) *JSONAssistant {
	t.Helper()
	dir := t.TempDir()
	a := &JSONAssistant{ID: "test", Name: "Test", Dir: dir, Path: filepath.Join(dir, "mcp.json")}
	if content != "" {
		if err := os.WriteFile(a.Path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return a
}

func read(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestJSONInstallNewFile(t *testing.T) {
	a := testAssistant(t, "")
	if err := os.Remove(a.Dir); err != nil {
		t.Fatal(err)
	}
	if a.Installed() {
		t.Error("installed without its folder")
	}
	res, err := a.Install("/opt/bin/quiver")
	if err != nil || res != Installed {
		t.Fatalf("install: %v %v", res, err)
	}
	want := `{
  "mcpServers": {
    "quiver": {
      "command": "/opt/bin/quiver",
      "args": [
        "mcp"
      ]
    }
  }
}
`
	if got := read(t, a.Path); got != want {
		t.Errorf("file:\n%s", got)
	}
	if _, err := os.Stat(a.Path + ".quiver-backup"); !os.IsNotExist(err) {
		t.Errorf("backup of a new file: %v", err)
	}
	reg, err := a.Get()
	if err != nil || reg == nil || reg.Command != "/opt/bin/quiver" || reg.Args != "mcp" || reg.Scope != a.Path {
		t.Errorf("get: %+v %v", reg, err)
	}
}

const existing = `{
	"theme": "dark",
	"mcpServers": {
		"github": {"command": "gh-mcp", "args": ["--stdio"]},
		"quiver": {"command": "/old/quiver", "args": ["mcp"], "env": {"QUIVER_PROJECT": "/code/app"}}
	},
	"zeta": [1, 2]
}
`

func TestJSONInstallKeepsOtherKeys(t *testing.T) {
	a := testAssistant(t, existing)
	res, err := a.Install("/opt/bin/quiver")
	if err != nil || res != Updated {
		t.Fatalf("install: %v %v", res, err)
	}
	want := `{
	"theme": "dark",
	"mcpServers": {
		"github": {
			"command": "gh-mcp",
			"args": [
				"--stdio"
			]
		},
		"quiver": {
			"command": "/opt/bin/quiver",
			"args": [
				"mcp"
			],
			"env": {
				"QUIVER_PROJECT": "/code/app"
			}
		}
	},
	"zeta": [
		1,
		2
	]
}
`
	if got := read(t, a.Path); got != want {
		t.Errorf("file:\n%s", got)
	}
	if got := read(t, a.Path+".quiver-backup"); got != existing {
		t.Errorf("backup:\n%s", got)
	}
	// Windows has no Unix file modes.
	if info, err := os.Stat(a.Path); runtime.GOOS != "windows" && (err != nil || info.Mode().Perm() != 0o600) {
		t.Errorf("mode: %v %v", info.Mode(), err)
	}

	res, err = a.Install("/opt/bin/quiver")
	if err != nil || res != Unchanged {
		t.Errorf("second install: %v %v", res, err)
	}

	res, err = a.Uninstall()
	if err != nil || res != Removed {
		t.Fatalf("uninstall: %v %v", res, err)
	}
	got := read(t, a.Path)
	if strings.Contains(got, "quiver") || !strings.Contains(got, `"github"`) || !strings.Contains(got, `"theme": "dark"`) {
		t.Errorf("after uninstall:\n%s", got)
	}
	if reg, err := a.Get(); reg != nil || err != nil {
		t.Errorf("get after uninstall: %+v %v", reg, err)
	}
	res, err = a.Uninstall()
	if err != nil || res != Unchanged {
		t.Errorf("second uninstall: %v %v", res, err)
	}
}

func TestJSONInstallAddsToOtherServers(t *testing.T) {
	a := testAssistant(t, `{"mcpServers": {"github": {"command": "gh-mcp"}}}`)
	if res, err := a.Install("/opt/bin/quiver"); err != nil || res != Installed {
		t.Fatalf("install: %v %v", res, err)
	}
	got := read(t, a.Path)
	if !strings.Contains(got, `"github": {`) || strings.Index(got, "github") > strings.Index(got, "quiver") {
		t.Errorf("file:\n%s", got)
	}
}

func TestJSONInstallReplacesBadEntry(t *testing.T) {
	a := testAssistant(t, `{"mcpServers": {"quiver": "old"}}`)
	if _, err := a.Get(); err == nil {
		t.Error("get: no error for an entry that is not an object")
	}
	if res, err := a.Install("/opt/bin/quiver"); err != nil || res != Updated {
		t.Fatalf("install: %v %v", res, err)
	}
	if reg, err := a.Get(); err != nil || reg.Command != "/opt/bin/quiver" {
		t.Errorf("get: %+v %v", reg, err)
	}
}

func TestJSONUninstallMissingFile(t *testing.T) {
	a := testAssistant(t, "")
	if res, err := a.Uninstall(); err != nil || res != Unchanged {
		t.Errorf("uninstall: %v %v", res, err)
	}
	if _, err := os.Stat(a.Path); !os.IsNotExist(err) {
		t.Errorf("created the file: %v", err)
	}
}

func TestJSONBrokenFile(t *testing.T) {
	for name, content := range map[string]string{
		"syntax":  `{"mcpServers": {`,
		"comment": "{\n  // servers\n  \"mcpServers\": {}\n}",
		"array":   `[]`,
		"servers": `{"mcpServers": []}`,
		"second":  `{} {}`,
	} {
		t.Run(name, func(t *testing.T) {
			a := testAssistant(t, content)
			_, installErr := a.Install("/opt/bin/quiver")
			_, uninstallErr := a.Uninstall()
			_, getErr := a.Get()
			if installErr == nil || uninstallErr == nil || getErr == nil {
				t.Errorf("install %v, uninstall %v, get %v", installErr, uninstallErr, getErr)
			}
			if got := read(t, a.Path); got != content {
				t.Errorf("file changed:\n%s", got)
			}
			if _, err := os.Stat(a.Path + ".quiver-backup"); !os.IsNotExist(err) {
				t.Errorf("backup written: %v", err)
			}
		})
	}
}

func TestIndentOf(t *testing.T) {
	for in, want := range map[string]string{
		"":                     "  ",
		`{"a": 1}`:             "  ",
		"{\n    \"a\": 1\n}":   "    ",
		"{\n\t\"a\": 1\n}":     "\t",
		"{\n\n  \"a\": 1\n}\n": "  ",
	} {
		if got := indentOf([]byte(in)); got != want {
			t.Errorf("indentOf(%q) = %q, want %q", in, got, want)
		}
	}
}
