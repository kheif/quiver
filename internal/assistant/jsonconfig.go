package assistant

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/kheif/quiver/internal/fsutil"
)

// JSONAssistant is an assistant that reads its MCP servers from the
// mcpServers object of a JSON file.
type JSONAssistant struct {
	ID   string // the install argument
	Name string
	Dir  string // the assistant's own folder; it exists once the assistant ran
	Path string // the config file
	// Global means the assistant starts servers outside any project, so
	// Quiver serves only the global skills there.
	Global bool
	// Delivery is how always-on skills reach this assistant
	// (docs/spike-hooks.md). install, doctor and status report it.
	Delivery Delivery
	// ServersKey is the top-level key that holds the servers object. It is
	// "mcpServers" for most assistants and "servers" for VS Code; an empty
	// value means "mcpServers".
	ServersKey string
	// Extra are members added to Quiver's entry, after command and args, such
	// as VS Code's "type": "stdio". They are kept up to date on install.
	Extra []member
}

// serversKey is the top-level key that holds the servers object.
func (a *JSONAssistant) serversKey() string {
	if a.ServersKey != "" {
		return a.ServersKey
	}
	return "mcpServers"
}

// entry is Quiver's server entry for the config file: command, args and the
// assistant's extra members.
func (a *JSONAssistant) entry(bin string) []member {
	e := []member{
		{"command", mustMarshal(bin)},
		{"args", mustMarshal([]string{"mcp"})},
	}
	return append(e, a.Extra...)
}

// Cursor is Cursor's user-wide MCP config.
func Cursor() (*JSONAssistant, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	dir := filepath.Join(home, ".cursor")
	return &JSONAssistant{ID: "cursor", Name: "Cursor", Dir: dir, Path: filepath.Join(dir, "mcp.json"), Delivery: DeliveryHook}, nil
}

// ClaudeDesktop is Claude Desktop's config: in ~/Library/Application Support
// on macOS and in %APPDATA% on Windows.
func ClaudeDesktop() (*JSONAssistant, error) {
	config, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	dir := filepath.Join(config, "Claude")
	return &JSONAssistant{ID: "claude-desktop", Name: "Claude Desktop", Dir: dir, Path: filepath.Join(dir, "claude_desktop_config.json"), Global: true, Delivery: DeliveryOnRequest}, nil
}

// Antigravity is the MCP config of the Antigravity CLI (agy). It keeps the
// server's instructions in a file that the model rarely reads.
func Antigravity() (*JSONAssistant, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	dir := filepath.Join(home, ".gemini")
	return &JSONAssistant{ID: "antigravity", Name: "Antigravity", Dir: filepath.Join(dir, "antigravity-cli"), Path: filepath.Join(dir, "config", "mcp_config.json"), Delivery: DeliveryHook}, nil
}

// VSCode is VS Code's user-wide MCP config. The server lives under the key
// "servers" with "type": "stdio", in the user folder's mcp.json:
// ~/Library/Application Support/Code/User on macOS, %APPDATA%\Code\User on
// Windows, ~/.config/Code/User on Linux (docs/spike-hooks.md). VS Code starts
// the server in the home folder, so Global would serve only global skills;
// instead the server asks for the workspace through MCP roots (server.Resolve),
// so Global stays off. VS Code Insiders keeps its files under "Code - Insiders";
// when only Insiders is installed, that folder is used. Always-on skills reach
// VS Code through the ~/.copilot hook (vscodehooks.go), so Delivery is a hook.
func VSCode() (*JSONAssistant, error) {
	config, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	user := filepath.Join(config, "Code", "User")
	if !isDir(user) {
		if ins := filepath.Join(config, "Code - Insiders", "User"); isDir(ins) {
			user = ins
		}
	}
	return &JSONAssistant{ID: "vscode", Name: "VS Code", Dir: user, Path: filepath.Join(user, "mcp.json"), ServersKey: "servers", Extra: []member{{"type", mustMarshal("stdio")}}, Delivery: DeliveryHook}, nil
}

// isDir reports whether path is a directory.
func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// JSONAssistants returns every assistant configured by JSON.
func JSONAssistants() ([]*JSONAssistant, error) {
	var all []*JSONAssistant
	for _, f := range []func() (*JSONAssistant, error){Cursor, ClaudeDesktop, Antigravity, VSCode} {
		a, err := f()
		if err != nil {
			return nil, err
		}
		all = append(all, a)
	}
	return all, nil
}

// Installed reports whether the assistant's folder exists.
func (a *JSONAssistant) Installed() bool {
	info, err := os.Stat(a.Dir)
	return err == nil && info.IsDir()
}

// Get returns the quiver entry of the config file, or nil if there is none.
// Scope is the file's path.
func (a *JSONAssistant) Get() (*Registration, error) {
	_, servers, err := readConfig(a.Path, a.serversKey())
	if err != nil || servers == nil {
		return nil, err
	}
	i := index(servers, ServerName)
	if i < 0 {
		return nil, nil
	}
	var entry serverEntry
	if err := json.Unmarshal(servers[i].value, &entry); err != nil {
		return nil, fmt.Errorf("%s: the %s entry: %w", a.Path, ServerName, err)
	}
	return &Registration{Scope: a.Path, Command: entry.Command, Args: strings.Join(entry.Args, " ")}, nil
}

// Install adds the quiver entry, which runs bin, to the config file, or
// points an existing one to bin. Other servers, other keys and the other
// fields of the entry, such as env, are kept.
func (a *JSONAssistant) Install(bin string) (Result, error) {
	return a.change(func(servers []member) ([]member, Result, error) {
		want := a.entry(bin)
		i := index(servers, ServerName)
		if i < 0 {
			return append(servers, member{ServerName, encodeObject(want)}), Installed, nil
		}
		entry, err := parseObject(servers[i].value)
		if err != nil {
			entry = nil // not an object: replace it
		}
		changed := false
		for _, m := range want {
			if j := index(entry, m.key); j < 0 || !same(entry[j].value, m.value) {
				changed = true
			}
			entry = set(entry, m.key, m.value)
		}
		if !changed {
			return servers, Unchanged, nil
		}
		servers[i].value = encodeObject(entry)
		return servers, Updated, nil
	})
}

// Uninstall removes the quiver entry from the config file.
func (a *JSONAssistant) Uninstall() (Result, error) {
	return a.change(func(servers []member) ([]member, Result, error) {
		i := index(servers, ServerName)
		if i < 0 {
			return servers, Unchanged, nil
		}
		return slices.Delete(servers, i, i+1), Removed, nil
	})
}

// change applies f to the mcpServers object of the config file and writes
// the file when f changed it; see editJSONFile.
func (a *JSONAssistant) change(f func([]member) ([]member, Result, error)) (Result, error) {
	key := a.serversKey()
	return editJSONFile(a.Path, func(top []member) ([]member, Result, error) {
		var servers []member
		if i := index(top, key); i >= 0 {
			s, err := parseObject(top[i].value)
			if err != nil {
				return nil, Unchanged, fmt.Errorf("%s: %s is not a JSON object, so the file was left as it is", a.Path, key)
			}
			servers = s
			if servers == nil {
				servers = []member{}
			}
		}
		servers, res, err := f(servers)
		if err != nil || res == Unchanged {
			return top, res, err
		}
		return set(top, key, encodeObject(servers)), res, nil
	})
}

// editJSONFile applies f to the top-level object of the JSON file at path and
// writes the file when f changed it. The old file is copied to
// path+".quiver-backup" first. The file keeps its key order, indentation and
// mode; a missing file starts from an empty object, and a file that is not
// valid JSON is never written.
func editJSONFile(path string, f func(top []member) ([]member, Result, error)) (Result, error) {
	old, err := os.ReadFile(path)
	exists := err == nil
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return Unchanged, err
	}
	var top []member
	if exists && len(bytes.TrimSpace(old)) > 0 {
		top, err = parseObject(old)
		if err != nil {
			return Unchanged, fmt.Errorf("%s is not valid JSON, so it was left as it is: %w", path, err)
		}
	}
	top, res, err := f(top)
	if err != nil || res == Unchanged {
		return Unchanged, err
	}

	perm := os.FileMode(0o644)
	if exists {
		if info, err := os.Stat(path); err == nil {
			perm = info.Mode().Perm()
		}
		if err := fsutil.WriteFileAtomic(path+".quiver-backup", old, perm); err != nil {
			return Unchanged, fmt.Errorf("back up %s: %w", path, err)
		}
	}
	var out bytes.Buffer
	if err := json.Indent(&out, encodeObject(top), "", indentOf(old)); err != nil {
		return Unchanged, err
	}
	out.WriteByte('\n')
	if err := fsutil.WriteFileAtomic(path, out.Bytes(), perm); err != nil {
		return Unchanged, err
	}
	return res, nil
}

// readConfig reads the config file's top-level object and its servers object
// under key. A missing or empty file is an empty object; servers is nil when
// there is no such key.
func readConfig(path, key string) (top, servers []member, err error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) || (err == nil && len(bytes.TrimSpace(data)) == 0) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	top, err = parseObject(data)
	if err != nil {
		return nil, nil, fmt.Errorf("%s is not valid JSON, so it was left as it is: %w", path, err)
	}
	i := index(top, key)
	if i < 0 {
		return top, nil, nil
	}
	servers, err = parseObject(top[i].value)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %s is not a JSON object, so the file was left as it is", path, key)
	}
	if servers == nil {
		servers = []member{}
	}
	return top, servers, nil
}

// serverEntry is an entry of mcpServers.
type serverEntry struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
}

// member is one key of a JSON object, in file order.
type member struct {
	key   string
	value json.RawMessage
}

// parseObject decodes a JSON object into its members, in order.
func parseObject(data []byte) ([]member, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	if d, ok := tok.(json.Delim); !ok || d != '{' {
		return nil, errors.New("not a JSON object")
	}
	var members []member
	for dec.More() {
		tok, err := dec.Token()
		if err != nil {
			return nil, err
		}
		key, _ := tok.(string)
		var value json.RawMessage
		if err := dec.Decode(&value); err != nil {
			return nil, err
		}
		members = append(members, member{key, value})
	}
	if _, err := dec.Token(); err != nil {
		return nil, err
	}
	if _, err := dec.Token(); err != io.EOF {
		return nil, errors.New("more data after the object")
	}
	return members, nil
}

// encodeObject encodes members as a compact JSON object.
func encodeObject(members []member) []byte {
	var b bytes.Buffer
	b.WriteByte('{')
	for i, m := range members {
		if i > 0 {
			b.WriteByte(',')
		}
		key, _ := json.Marshal(m.key)
		b.Write(key)
		b.WriteByte(':')
		_ = json.Compact(&b, m.value)
	}
	b.WriteByte('}')
	return b.Bytes()
}

func index(members []member, key string) int {
	return slices.IndexFunc(members, func(m member) bool { return m.key == key })
}

// set replaces the value of key, or appends key.
func set(members []member, key string, value json.RawMessage) []member {
	if i := index(members, key); i >= 0 {
		members[i].value = value
		return members
	}
	return append(members, member{key, value})
}

// same reports whether two JSON values are equal apart from whitespace.
func same(a, b json.RawMessage) bool {
	var ca, cb bytes.Buffer
	return json.Compact(&ca, a) == nil && json.Compact(&cb, b) == nil && bytes.Equal(ca.Bytes(), cb.Bytes())
}

// indentOf returns the indentation of the first indented line of data, or
// two spaces.
func indentOf(data []byte) string {
	for _, line := range strings.Split(string(data), "\n")[1:] {
		trimmed := strings.TrimLeft(line, " \t")
		if n := len(line) - len(trimmed); n > 0 && trimmed != "" {
			return line[:n]
		}
	}
	return "  "
}
