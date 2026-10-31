package skill

import (
	"bytes"
	"errors"
	"strings"

	"gopkg.in/yaml.v3"
)

// Rewrite returns the content of a SKILL.md with a new description, body,
// or both; nil leaves that part as it is. Other frontmatter fields are
// kept. The body is written the way new skills get it: trimmed, after an
// empty line.
func Rewrite(data []byte, description, body *string) ([]byte, error) {
	fm, oldBody, err := splitFrontmatter(data)
	if err != nil {
		return nil, err
	}
	if description != nil {
		if fm, err = setField(fm, "description", strings.TrimSpace(*description)); err != nil {
			return nil, err
		}
	}
	out := "---\n" + string(fm) + "---\n"
	if body == nil {
		return append([]byte(out), oldBody...), nil
	}
	if b := strings.TrimSpace(*body); b != "" {
		out += "\n" + b + "\n"
	}
	return []byte(out), nil
}

// setField sets a string field in YAML frontmatter, keeping the order and
// the other fields.
func setField(fm []byte, key, value string) ([]byte, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(fm, &doc); err != nil {
		return nil, err
	}
	if doc.Kind == 0 {
		doc = yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{{Kind: yaml.MappingNode, Tag: "!!map"}}}
	}
	m := doc.Content[0]
	if m.Kind != yaml.MappingNode {
		return nil, errors.New("invalid frontmatter: not a mapping")
	}
	v := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: value}
	found := false
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			m.Content[i+1] = v
			found = true
		}
	}
	if !found {
		m.Content = append(m.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key}, v)
	}
	var b bytes.Buffer
	enc := yaml.NewEncoder(&b)
	enc.SetIndent(2)
	if err := enc.Encode(&doc); err != nil {
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}
