package assistant

import (
	"errors"
	"slices"
	"testing"

	"github.com/kheif/quiver/internal/assistant/assistanttest"
)

func TestInstallNew(t *testing.T) {
	f := assistanttest.Install(t)
	res, hidden, err := Install("/opt/bin/quiver")
	if err != nil || res != Installed || hidden != nil {
		t.Fatalf("got %v, %v, %v", res, hidden, err)
	}
	if f.User() != "/opt/bin/quiver|mcp" {
		t.Errorf("user %q", f.User())
	}
}

func TestInstallUnchanged(t *testing.T) {
	f := assistanttest.Install(t)
	f.SetUser(t, "/opt/bin/quiver", "mcp")
	res, _, err := Install("/opt/bin/quiver")
	if err != nil || res != Unchanged {
		t.Fatalf("got %v, %v", res, err)
	}
	if want := []string{"mcp get quiver", "mcp get quiver"}; !slices.Equal(f.Calls(), want) {
		t.Errorf("calls %q", f.Calls())
	}
}

func TestInstallUpdatesPath(t *testing.T) {
	f := assistanttest.Install(t)
	f.SetUser(t, "/opt/homebrew/Cellar/quiver/0.1.0/bin/quiver", "mcp")
	res, _, err := Install("/opt/homebrew/bin/quiver")
	if err != nil || res != Updated {
		t.Fatalf("got %v, %v", res, err)
	}
	if f.User() != "/opt/homebrew/bin/quiver|mcp" {
		t.Errorf("user %q", f.User())
	}
}

// A local registration in the current folder hides the user one from
// "claude mcp get" there, but not from Install.
func TestInstallBehindLocal(t *testing.T) {
	f := assistanttest.Install(t)
	t.Chdir(t.TempDir())
	f.SetLocal(t, "/dev/quiver", "mcp")
	f.SetUser(t, "/opt/bin/quiver", "mcp")
	res, hidden, err := Install("/opt/bin/quiver")
	if err != nil || res != Unchanged {
		t.Fatalf("got %v, %v", res, err)
	}
	if hidden == nil || hidden.Command != "/dev/quiver" || hidden.User() {
		t.Errorf("hidden %+v", hidden)
	}

	f.SetLocal(t, "/opt/bin/quiver", "mcp")
	if _, hidden, err = Install("/opt/bin/quiver"); err != nil || hidden != nil {
		t.Errorf("same binary: hidden %+v, %v", hidden, err)
	}
}

func TestUninstall(t *testing.T) {
	f := assistanttest.Install(t)
	f.SetUser(t, "/opt/bin/quiver", "mcp")
	if res, err := Uninstall(); err != nil || res != Removed {
		t.Fatalf("got %v, %v", res, err)
	}
	if f.User() != "" {
		t.Errorf("user %q", f.User())
	}
	if res, err := Uninstall(); err != nil || res != Unchanged {
		t.Errorf("second uninstall: %v, %v", res, err)
	}
}

func TestNoClaude(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	if _, _, err := Install("/opt/bin/quiver"); !errors.Is(err, ErrNoClaude) {
		t.Errorf("install: %v", err)
	}
	if _, err := Uninstall(); !errors.Is(err, ErrNoClaude) {
		t.Errorf("uninstall: %v", err)
	}
}

func TestCommands(t *testing.T) {
	if got := AddCommand("/opt/bin/quiver"); got != "claude mcp add --scope user quiver -- /opt/bin/quiver mcp" {
		t.Errorf("got %q", got)
	}
	if got := AddCommand("/Users/me/my tools/it's/quiver"); got != `claude mcp add --scope user quiver -- '/Users/me/my tools/it'\''s/quiver' mcp` {
		t.Errorf("quoted: got %q", got)
	}
	want := "{\n  \"mcpServers\": {\n    \"quiver\": {\n      \"command\": \"/opt/bin/quiver\",\n      \"args\": [\n        \"mcp\"\n      ]\n    }\n  }\n}"
	if got := ConfigSnippet("/opt/bin/quiver"); got != want {
		t.Errorf("snippet %s", got)
	}
}
