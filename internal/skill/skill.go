// Package skill reads skills in the Agent Skills format: a folder with a
// SKILL.md file that starts with YAML frontmatter.
package skill

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// FileName is the file that makes a folder a skill.
const FileName = "SKILL.md"

// Skill is one parsed skill folder.
type Skill struct {
	Name        string
	Description string
	Dir         string // absolute or caller-given folder path
	Source      string // "local" or a source name; set by the caller
	ID          string // set by the caller
	Body        string // markdown after the frontmatter
	// Frontmatter holds every frontmatter field, including unknown ones.
	Frontmatter map[string]any
}

// Path returns the path of the skill's SKILL.md.
func (s *Skill) Path() string {
	return filepath.Join(s.Dir, FileName)
}

// Load reads dir/SKILL.md and parses its frontmatter.
// It does not check the fields; use Validate for that.
func Load(dir string) (*Skill, error) {
	return LoadFile(filepath.Join(dir, FileName))
}

// LoadFile is Load for a skill file with any name, such as greeter.md.
// Dir is the folder the file is in.
func LoadFile(path string) (*Skill, error) {
	dir := filepath.Dir(path)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("%s: not found", path)
		}
		return nil, err
	}
	fm, body, err := splitFrontmatter(data)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	meta := map[string]any{}
	if err := yaml.Unmarshal(fm, &meta); err != nil {
		return nil, fmt.Errorf("%s: invalid frontmatter: %w", path, err)
	}
	if meta == nil {
		meta = map[string]any{}
	}
	s := &Skill{Dir: dir, Body: string(body), Frontmatter: meta}
	s.Name, _ = meta["name"].(string)
	s.Description, _ = meta["description"].(string)
	return s, nil
}

// splitFrontmatter splits data into the YAML between the leading "---"
// lines and the body after them.
func splitFrontmatter(data []byte) (fm, body []byte, err error) {
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))
	data = bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n"))
	if !bytes.HasPrefix(data, []byte("---\n")) {
		return nil, nil, errors.New("missing frontmatter: file must start with ---")
	}
	rest := data[len("---\n"):]
	for i := 0; i <= len(rest); {
		end := bytes.IndexByte(rest[i:], '\n')
		line := rest[i:]
		next := len(rest)
		if end >= 0 {
			line = rest[i : i+end]
			next = i + end + 1
		}
		if string(bytes.TrimRight(line, " \t")) == "---" {
			return rest[:i], rest[next:], nil
		}
		if end < 0 {
			break
		}
		i = next
	}
	return nil, nil, errors.New("missing frontmatter: no closing ---")
}

// HasFrontmatter reports whether the file at path starts with a "---"
// line, as a skill file does. Markdown files without it, such as READMEs,
// are not skills at all.
func HasFrontmatter(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer func() { _ = f.Close() }()
	head := make([]byte, 8)
	n, _ := io.ReadFull(f, head)
	head = bytes.TrimPrefix(head[:n], []byte("\xef\xbb\xbf")) // UTF-8 BOM
	return bytes.HasPrefix(head, []byte("---\n")) || bytes.HasPrefix(head, []byte("---\r\n"))
}
