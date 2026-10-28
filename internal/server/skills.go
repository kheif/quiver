package server

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/kheif/quiver/internal/library"
	"github.com/kheif/quiver/internal/registry"
	"github.com/kheif/quiver/internal/skill"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// SkillInfo is one skill in the list_skills result.
type SkillInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	// Scope is "global" or "project".
	Scope string `json:"scope"`
}

// servedSkill is a skill served in the project.
type servedSkill struct {
	*skill.Skill
	Scope    string
	AlwaysOn bool
}

// loadServed reads the library and the registry, and returns the library
// with the skills served in the project, sorted by ID. Assigned skills that
// are no longer in the library are left out. Tools call it on every
// request, so they see skills assigned or updated during the session.
func loadServed(projectID string) (*library.Library, []servedSkill, error) {
	r, err := registry.Load()
	if err != nil {
		return nil, nil, err
	}
	l, err := library.Load()
	if err != nil {
		return nil, nil, err
	}
	var out []servedSkill
	always := r.AlwaysOnServed(projectID)
	for _, id := range r.Served(projectID) {
		s, err := l.Get(id)
		if err != nil {
			continue
		}
		scope := "project"
		if slices.Contains(r.Global, id) {
			scope = "global"
		}
		out = append(out, servedSkill{s, scope, slices.Contains(always, id)})
	}
	return l, out, nil
}

// addSkillTools registers the tools that read skills.
func addSkillTools(s *mcp.Server, sess *session) {
	type listOut struct {
		Skills []SkillInfo `json:"skills"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_skills",
		Description: "List the skills available in this project: the global skills and the project's own, including skills assigned during this session.",
	}, func(context.Context, *mcp.CallToolRequest, struct{}) (*mcp.CallToolResult, listOut, error) {
		_, skills, err := loadServed(sess.projectID())
		if err != nil {
			return nil, listOut{}, err
		}
		out := listOut{Skills: []SkillInfo{}}
		for _, s := range skills {
			out.Skills = append(out.Skills, SkillInfo{ID: s.ID, Name: s.Name, Description: s.Description, Scope: s.Scope})
		}
		return nil, out, nil
	})

	type getIn struct {
		ID string `json:"id" jsonschema:"skill ID, as in the instructions or list_skills"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_skill",
		Description: "Load a skill: its instructions (SKILL.md), its extra files and its folder. Follow the instructions.",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in getIn) (*mcp.CallToolResult, any, error) {
		l, skills, err := loadServed(sess.projectID())
		if err != nil {
			return nil, nil, err
		}
		sk, err := l.Get(in.ID)
		if err != nil {
			return nil, nil, err
		}
		files, err := extraFiles(sk)
		if err != nil {
			return nil, nil, err
		}
		var b strings.Builder
		if !slices.ContainsFunc(skills, func(s servedSkill) bool { return s.ID == sk.ID }) {
			fmt.Fprintf(&b, "Note: %s is not assigned in this project. The user can assign it with \"quiver add %s\".\n\n", sk.ID, sk.ID)
		}
		b.WriteString(strings.TrimSpace(sk.Body))
		b.WriteString("\n\n---\n")
		if len(files) > 0 {
			fmt.Fprintf(&b, "Extra files (load with get_skill_file): %s\n", strings.Join(files, ", "))
		}
		fmt.Fprintf(&b, "Scripts and files referenced by this skill are in %s. Use absolute paths.\n", sk.Dir)
		return text(b.String()), nil, nil
	})

	type fileIn struct {
		ID   string `json:"id" jsonschema:"skill ID"`
		Path string `json:"path" jsonschema:"file path inside the skill folder, as listed by get_skill"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_skill_file",
		Description: "Load an extra file of a skill, such as references/checklist.md. Large and binary files are not returned; you get their absolute path instead.",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in fileIn) (*mcp.CallToolResult, any, error) {
		l, err := library.Load()
		if err != nil {
			return nil, nil, err
		}
		sk, err := l.Get(in.ID)
		if err != nil {
			return nil, nil, err
		}
		root, err := filepath.EvalSymlinks(sk.Dir)
		if err != nil {
			return nil, nil, err
		}
		path, err := resolveIn(root, in.Path)
		if err != nil {
			return nil, nil, err
		}
		info, err := os.Stat(path)
		if err != nil {
			return nil, nil, err
		}
		if info.Size() > maxText {
			return text(fmt.Sprintf("%s is %d bytes, more than %d KB, so it is not returned as text. Use it from that path.", path, info.Size(), maxText/1024)), nil, nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, nil, err
		}
		if !utf8.Valid(data) || bytes.IndexByte(data, 0) >= 0 {
			return text(fmt.Sprintf("%s is a binary file (%d bytes), so it is not returned as text. Use it from that path.", path, len(data))), nil, nil
		}
		return text(string(data)), nil, nil
	})
}

// maxText is the largest file get_skill_file returns as text.
const maxText = 256 * 1024

// extraFiles lists the skill's extra files. skill.Files lists symlinks as
// they are, so entries that get_skill_file would refuse are dropped.
func extraFiles(s *skill.Skill) ([]string, error) {
	files, err := skill.Files(s)
	if err != nil {
		return nil, err
	}
	root, err := filepath.EvalSymlinks(s.Dir)
	if err != nil {
		return nil, err
	}
	out := files[:0]
	for _, f := range files {
		if _, err := resolveIn(root, f); err == nil {
			out = append(out, f)
		}
	}
	return out, nil
}

// resolveIn returns the real path of rel, a slash-separated path inside
// root. root must have no symlinks in its path. It fails unless rel names a
// regular file inside root, also after following symlinks. Hidden files
// are refused too: get_skill does not list them, and a skill at the root of
// a source repository has .git inside its folder.
func resolveIn(root, rel string) (string, error) {
	outside := fmt.Errorf("%s is outside the skill folder", rel)
	if !filepath.IsLocal(filepath.FromSlash(rel)) {
		return "", outside
	}
	real, err := filepath.EvalSymlinks(filepath.Join(root, filepath.FromSlash(rel)))
	if errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("%s not found", rel)
	}
	if err != nil {
		return "", err
	}
	in, err := filepath.Rel(root, real)
	if err != nil || !filepath.IsLocal(in) {
		return "", outside
	}
	info, err := os.Stat(real)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("%s is not a file", rel)
	}
	for _, part := range strings.Split(in, string(filepath.Separator)) {
		if strings.HasPrefix(part, ".") {
			return "", fmt.Errorf("%s is hidden", rel)
		}
	}
	return real, nil
}
