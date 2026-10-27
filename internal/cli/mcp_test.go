package cli

import (
	"bufio"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestMCPStdoutHasOnlyMessages runs "quiver mcp" in a child process (this
// test binary, re-run with QUIVER_MCP_CHILD=1) and checks that every line it
// writes to stdout is a JSON-RPC message.
func TestMCPStdoutHasOnlyMessages(t *testing.T) {
	if os.Getenv("QUIVER_MCP_CHILD") == "1" {
		os.Args = []string{"quiver", "mcp"}
		os.Exit(Execute())
	}
	home := sourceEnv(t)
	cmd := exec.Command(os.Args[0], "-test.run=^TestMCPStdoutHasOnlyMessages$")
	cmd.Env = append(os.Environ(), "QUIVER_MCP_CHILD=1")
	var stderr strings.Builder
	cmd.Stderr = &stderr
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	// Keep stdin open until both replies are read: the server stops at EOF.
	for _, msg := range []string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"t","version":"0"}}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"ping","arguments":{}}}`,
	} {
		if _, err := stdin.Write([]byte(msg + "\n")); err != nil {
			t.Fatal(err)
		}
	}
	var replies []string
	sc := bufio.NewScanner(stdout)
	for sc.Scan() {
		var msg struct {
			JSONRPC string `json:"jsonrpc"`
		}
		if err := json.Unmarshal(sc.Bytes(), &msg); err != nil || msg.JSONRPC != "2.0" {
			t.Errorf("stdout line is not a JSON-RPC message: %q", sc.Text())
		}
		if replies = append(replies, sc.Text()); len(replies) == 2 {
			_ = stdin.Close()
		}
	}
	if err := cmd.Wait(); err != nil {
		t.Fatalf("%v\nstderr:\n%s", err, stderr.String())
	}
	if len(replies) != 2 || !strings.Contains(replies[1], "pong") {
		t.Errorf("replies: %q", replies)
	}
	if !strings.Contains(stderr.String(), "server started") {
		t.Errorf("stderr:\n%s", stderr.String())
	}
	if data, err := os.ReadFile(filepath.Join(home, "logs", "mcp.log")); err != nil || !strings.Contains(string(data), "server started") {
		t.Errorf("log file: %v\n%s", err, data)
	}
}
