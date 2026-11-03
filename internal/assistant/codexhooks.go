package assistant

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
)

// Codex runs a new or changed hook only once it is trusted. Trust lives in
// hooks.state of config.toml, keyed by the hook's hash. Quiver trusts its own
// hooks through codex app-server, the two requests Codex's review dialog sends
// (hooks/list, then config/batchWrite). See docs/spike-hooks.md.

// codexHookStatus is one hook from codex app-server's hooks/list.
type codexHookStatus struct {
	Key         string `json:"key"`
	Command     string `json:"command"`
	CurrentHash string `json:"currentHash"`
	TrustStatus string `json:"trustStatus"`
}

// isQuiver reports whether the hook runs Quiver's codex session-start or
// subagent-start command.
func (h codexHookStatus) isQuiver() bool {
	t := hookTarget{id: "codex"}
	return t.isQuiverHook(h.Command, "session-start") || t.isQuiverHook(h.Command, "subagent-start")
}

// CodexTrustHooks trusts Quiver's hooks that are new or changed, and only
// those, through codex app-server. It returns the number it trusted. A hook's
// hash changes with its command or fields, so install trusts again after it
// updates a hook.
func CodexTrustHooks() (int, error) {
	srv, err := startCodexAppServer()
	if err != nil {
		return 0, err
	}
	defer srv.close()
	hooks, err := srv.quiverHooks()
	if err != nil {
		return 0, err
	}
	review := map[string]map[string]string{}
	for _, h := range hooks {
		if h.TrustStatus == "untrusted" || h.TrustStatus == "modified" {
			review[h.Key] = map[string]string{"trusted_hash": h.CurrentHash}
		}
	}
	if len(review) == 0 {
		return 0, nil
	}
	if err := srv.call(3, "config/batchWrite", map[string]any{
		"edits":            []any{map[string]any{"keyPath": "hooks.state", "value": review, "mergeStrategy": "upsert"}},
		"reloadUserConfig": true,
	}, nil); err != nil {
		return 0, err
	}
	return len(review), nil
}

// CodexHooksNeedTrust returns the commands of Quiver's codex hooks that are
// new or changed, for doctor. It is empty when every Quiver hook is trusted.
func CodexHooksNeedTrust() ([]string, error) {
	srv, err := startCodexAppServer()
	if err != nil {
		return nil, err
	}
	defer srv.close()
	hooks, err := srv.quiverHooks()
	if err != nil {
		return nil, err
	}
	var need []string
	for _, h := range hooks {
		if h.TrustStatus != "trusted" {
			need = append(need, h.Command)
		}
	}
	return need, nil
}

// codexAppServer is a running codex app-server and its JSON-RPC stream.
type codexAppServer struct {
	cmd *exec.Cmd
	in  io.WriteCloser
	out *bufio.Reader
}

// startCodexAppServer starts codex app-server and does the initialize
// handshake.
func startCodexAppServer() (*codexAppServer, error) {
	path, err := exec.LookPath("codex")
	if err != nil {
		return nil, ErrNoCodex
	}
	cmd := exec.Command(path, "app-server")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	srv := &codexAppServer{cmd: cmd, in: stdin, out: bufio.NewReader(stdout)}
	if err := srv.call(1, "initialize", map[string]any{"clientInfo": map[string]string{"name": "quiver", "version": "1"}}, nil); err != nil {
		srv.close()
		return nil, err
	}
	if err := srv.notify("initialized"); err != nil {
		srv.close()
		return nil, err
	}
	return srv, nil
}

// quiverHooks lists Quiver's hooks as codex app-server sees them for a scratch
// folder. The folder only scopes the list; trust is global.
func (s *codexAppServer) quiverHooks() ([]codexHookStatus, error) {
	cwd, err := os.Getwd()
	if err != nil {
		cwd = os.TempDir()
	}
	var res struct {
		Data []struct {
			Hooks []codexHookStatus `json:"hooks"`
		} `json:"data"`
	}
	if err := s.call(2, "hooks/list", map[string]any{"cwds": []string{cwd}}, &res); err != nil {
		return nil, err
	}
	var quiver []codexHookStatus
	for _, e := range res.Data {
		for _, h := range e.Hooks {
			if h.isQuiver() {
				quiver = append(quiver, h)
			}
		}
	}
	return quiver, nil
}

// call sends a request and decodes its result into out (nil to ignore).
func (s *codexAppServer) call(id int, method string, params, out any) error {
	if err := s.send(map[string]any{"id": id, "method": method, "params": params}); err != nil {
		return err
	}
	for {
		var m struct {
			ID     *int            `json:"id"`
			Result json.RawMessage `json:"result"`
			Error  json.RawMessage `json:"error"`
		}
		line, err := s.out.ReadBytes('\n')
		if err != nil {
			return fmt.Errorf("codex app-server: %w", err)
		}
		if err := json.Unmarshal(line, &m); err != nil {
			continue // a log line, not a JSON-RPC message
		}
		if m.ID == nil || *m.ID != id {
			continue
		}
		if len(m.Error) > 0 {
			return fmt.Errorf("codex app-server %s: %s", method, m.Error)
		}
		if out != nil && len(m.Result) > 0 {
			return json.Unmarshal(m.Result, out)
		}
		return nil
	}
}

func (s *codexAppServer) notify(method string) error {
	return s.send(map[string]any{"method": method})
}

func (s *codexAppServer) send(m map[string]any) error {
	m["jsonrpc"] = "2.0"
	b, err := json.Marshal(m)
	if err != nil {
		return err
	}
	_, err = s.in.Write(append(b, '\n'))
	return err
}

func (s *codexAppServer) close() {
	_ = s.in.Close()
	_ = s.cmd.Process.Kill()
	_ = s.cmd.Wait()
}
