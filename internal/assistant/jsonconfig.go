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
	ID   string // the value of mcp install --for
	Name string
	Dir  string // the assistant's own folder; it exists once the assistant ran
	Path string // the config file, inside Dir
	// Global means the assistant starts servers outside any project, so
	// Quiver serves only the global skills there.
	Global bool
	// NoInstructions means the assistant does not pass the server's
	// instructions to the model, so it never sees the skill list or the
	// always-on skills (docs/spike-mcp-clients.md). mcp install sets it up
	// only when asked by name.
	NoInstructions bool
}

// Cursor is Cursor's user-wide MCP config.
func Cursor() (*JSONAssistant, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	dir := filepath.Join(home, ".cursor")
	return &JSONAssistant{ID: "cursor", Name: "Cursor", Dir: dir, Path: filepath.Join(dir, "mcp.json")}, nil
}

// ClaudeDesktop is Claude Desktop's config: in ~/Library/Application Support
// on macOS and in %APPDATA% on Windows.
func ClaudeDesktop() (*JSONAssistant, error) {
	config, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	dir := filepath.Join(config, "Claude")
	return &JSONAssistant{ID: "claude-desktop", Name: "Claude Desktop", Dir: dir, Path: filepath.Join(dir, "claude_desktop_config.json"), Global: true, NoInstructions: true}, nil
}

// JSONAssistants returns every assistant configured by JSON.
func JSONAssistants() ([]*JSONAssistant, error) {
	var all []*JSONAssistant
	for _, f := range []func() (*JSONAssistant, error){Cursor, ClaudeDesktop} {
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
	_, servers, err := readConfig(a.Path)
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
		i := index(servers, ServerName)
		if i < 0 {
			value, _ := json.Marshal(serverEntry{bin, []string{"mcp"}})
			return append(servers, member{ServerName, value}), Installed, nil
		}
		entry, err := parseObject(servers[i].value)
		if err != nil {
			entry = nil // not an object: replace it
		}
		command, _ := json.Marshal(bin)
		args, _ := json.Marshal([]string{"mcp"})
		if j := index(entry, "command"); j >= 0 && same(entry[j].value, command) {
			if k := index(entry, "args"); k >= 0 && same(entry[k].value, args) {
				return servers, Unchanged, nil
			}
		}
		entry = set(set(entry, "command", command), "args", args)
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
// the file when f changed it. The old file is copied to Path+".quiver-backup"
// first. The file keeps its key order and indentation; a file that is not
// valid JSON is never written.
func (a *JSONAssistant) change(f func([]member) ([]member, Result, error)) (Result, error) {
	top, servers, err := readConfig(a.Path)
	if err != nil {
		return Unchanged, err
	}
	old, err := os.ReadFile(a.Path)
	exists := err == nil
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return Unchanged, err
	}
	servers, res, err := f(servers)
	if err != nil || res == Unchanged {
		return Unchanged, err
	}
	top = set(top, "mcpServers", encodeObject(servers))

	perm := os.FileMode(0o644)
	if exists {
		if info, err := os.Stat(a.Path); err == nil {
			perm = info.Mode().Perm()
		}
		if err := fsutil.WriteFileAtomic(a.Path+".quiver-backup", old, perm); err != nil {
			return Unchanged, fmt.Errorf("back up %s: %w", a.Path, err)
		}
	}
	var out bytes.Buffer
	if err := json.Indent(&out, encodeObject(top), "", indentOf(old)); err != nil {
		return Unchanged, err
	}
	out.WriteByte('\n')
	if err := fsutil.WriteFileAtomic(a.Path, out.Bytes(), perm); err != nil {
		return Unchanged, err
	}
	return res, nil
}

// readConfig reads the config file's top-level object and its mcpServers
// object. A missing or empty file is an empty object; servers is nil when
// there is no mcpServers key.
func readConfig(path string) (top, servers []member, err error) {
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
	i := index(top, "mcpServers")
	if i < 0 {
		return top, nil, nil
	}
	servers, err = parseObject(top[i].value)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: mcpServers is not a JSON object, so the file was left as it is", path)
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
