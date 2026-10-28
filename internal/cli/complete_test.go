package cli

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// Every command has the help that quiver help <cmd> shows.
func TestHelpText(t *testing.T) {
	var walk func(c *cobra.Command)
	walk = func(c *cobra.Command) {
		if c.Hidden || c.Name() == "help" || c.Name() == "completion" {
			return
		}
		if c.Short == "" || c.Long == "" || c.Example == "" {
			t.Errorf("%q: Short, Long and Example must be set", c.CommandPath())
		}
		for _, sub := range c.Commands() {
			walk(sub)
		}
	}
	walk(newRootCmd())
}

// complete runs the shell completion request for args and returns the
// suggestions, one per line, without the directive line.
func complete(t *testing.T, args ...string) []string {
	t.Helper()
	out, err := runIn(t, "", append([]string{cobra.ShellCompRequestCmd}, args...)...)
	if err != nil {
		t.Fatalf("%v: %v", args, err)
	}
	var lines []string
	for _, l := range strings.Split(strings.TrimSpace(out), "\n") {
		if l != "" && !strings.HasPrefix(l, ":") && !strings.HasPrefix(l, "Completion ended") {
			lines = append(lines, l)
		}
	}
	return lines
}

func TestCompletion(t *testing.T) {
	f := listEnv(t)
	t.Chdir(f.app)
	for _, c := range []struct {
		args []string
		want string
	}{
		{[]string{"add", ""}, "acme/deploy\tTest skill.|acme/review\tTest skill.|notes\tKeep notes.|pdf\tFill in and read PDF forms.|tdd\tWrite the test first. Then write the code that makes it pas…"},
		{[]string{"add", "pdf", "a"}, "acme/deploy\tTest skill.|acme/review\tTest skill."},
		{[]string{"where", "pdf", ""}, ""},
		{[]string{"edit", ""}, "notes\tKeep notes.|pdf\tFill in and read PDF forms.|tdd\tWrite the test first. Then write the code that makes it pas…"},
		{[]string{"remove", ""}, "acme/review|tdd"},
		{[]string{"remove", "--global", ""}, "acme/review|pdf"},
		{[]string{"remove", "--project", f.api, ""}, "tdd"},
		{[]string{"update", ""}, "acme\t" + f.tmp},
		{[]string{"list", "--source", "l"}, "local\tyour own skills"},
		{[]string{"projects", "forget", "a"}, "api\tapi-000002|api-000002\tapi|app\tapp-000001|app-000001\tapp"},
		{[]string{"projects", "rename", "app", ""}, ""},
		{[]string{"status", ""}, ""},
	} {
		got := strings.Join(complete(t, c.args...), "|")
		if c.args[0] == "update" {
			// The repository path differs per run; keep only its start.
			got, _, _ = strings.Cut(got, "\t")
			c.want, _, _ = strings.Cut(c.want, "\t")
		}
		if got != c.want {
			t.Errorf("complete %q:\ngot  %q\nwant %q", c.args, got, c.want)
		}
	}
}

// Completion never creates the Quiver home.
func TestCompletionNoHome(t *testing.T) {
	home := sourceEnv(t)
	if got := complete(t, "add", ""); len(got) != 0 {
		t.Errorf("got %q", got)
	}
	if _, err := os.Stat(home); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("home created: %v", err)
	}
}
