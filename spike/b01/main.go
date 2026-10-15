// Throwaway spike for backlog B01/B02: does the assistant load skills over MCP,
// and do always-on instructions arrive? Not production code.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type skill struct{ name, desc, dir string }

var logf *os.File

func logEvent(kind string, v any) {
	if logf == nil {
		return
	}
	b, _ := json.Marshal(map[string]any{"event": kind, "data": v})
	logf.Write(append(b, '\n'))
}

func loadSkills(root string) []skill {
	var out []skill
	dirs, _ := os.ReadDir(root)
	for _, d := range dirs {
		dir := filepath.Join(root, d.Name())
		b, err := os.ReadFile(filepath.Join(dir, "SKILL.md"))
		if err != nil {
			continue
		}
		s := skill{dir: dir}
		for _, line := range strings.Split(string(b), "\n") {
			if v, ok := strings.CutPrefix(line, "name: "); ok {
				s.name = v
			}
			if v, ok := strings.CutPrefix(line, "description: "); ok {
				s.desc = v
			}
		}
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].name < out[j].name })
	return out
}

func text(s string) *mcp.CallToolResult {
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: s}}}
}

func main() {
	exe, _ := os.Executable()
	base := filepath.Dir(exe)
	if p := os.Getenv("QV_LOG"); p != "" {
		logf, _ = os.OpenFile(p, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	}
	cwd, _ := os.Getwd()
	logEvent("start", map[string]any{"cwd": cwd, "pid": os.Getpid()})

	skills := loadSkills(filepath.Join(base, "skills"))
	byName := map[string]skill{}
	var list strings.Builder
	for _, s := range skills {
		byName[s.name] = s
		fmt.Fprintf(&list, "- %s: %s\n", s.name, s.desc)
	}

	instructions := ""
	if os.Getenv("QV_ALWAYS") != "" {
		instructions = "Always-on skill \"marker\": end every reply, including replies of subagents, with the line `[always-on]`.\n"
		if n := os.Getenv("QV_PAD"); n != "" {
			var k int
			fmt.Sscan(n, &k)
			instructions = strings.Repeat("Filler sentence to test the instructions length limit. ", k/55+1)[:k] + "\n" + instructions
		}
	}
	if os.Getenv("QV_ALWAYS") == "pointer" {
		rs := loadSkills(filepath.Join(base, "always"))[0]
		byName[rs.name] = rs
		instructions = "Always-on skills: " + rs.name + ". Before your first reply in this conversation, call the get_skill tool of this server for each always-on skill and follow it for the whole conversation.\n"
	}
	if os.Getenv("QV_CATALOG") != "" {
		instructions += "\nSkills (reusable instructions for specific tasks). Before starting a task, check whether one fits. " +
			"If it does, call the get_skill tool of this server with its name and follow it.\n" + list.String()
	}
	logEvent("instructions_len", len(instructions))

	var server *mcp.Server
	server = mcp.NewServer(&mcp.Implementation{Name: "quiver-spike", Version: "0.0.1"}, &mcp.ServerOptions{
		Instructions: instructions,
		InitializedHandler: func(ctx context.Context, req *mcp.InitializedRequest) {
			logEvent("initialized_enter", nil)
			defer func() {
				if r := recover(); r != nil {
					logEvent("initialized_panic", fmt.Sprint(r))
				}
			}()
			p := req.Session.InitializeParams()
			logEvent("initialized", map[string]any{"client": p.ClientInfo, "capabilities": p.Capabilities})
			go func() {
				r, err := req.Session.ListRoots(context.Background(), &mcp.ListRootsParams{})
				if err != nil {
					logEvent("roots_error", err.Error())
					return
				}
				logEvent("roots", r.Roots)
			}()
		},
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "list_skills",
		Description: "Skills are reusable instructions for specific tasks. Before starting a task, check whether one of these skills fits; " +
			"if it does, call get_skill with its name and follow it.\n\nAvailable skills:\n" + list.String(),
	}, func(ctx context.Context, req *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
		logEvent("call", map[string]any{"tool": "list_skills"})
		return text(list.String()), nil, nil
	})

	type getArgs struct {
		Name string `json:"name" jsonschema:"skill name from list_skills"`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_skill",
		Description: "Load a skill's instructions (SKILL.md) by name. Call this when a skill listed in list_skills fits the task, then follow the instructions.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, a getArgs) (*mcp.CallToolResult, any, error) {
		logEvent("call", map[string]any{"tool": "get_skill", "name": a.Name})
		rctx, cancel := context.WithTimeout(ctx, 3*time.Second)
		if r, err := req.Session.ListRoots(rctx, &mcp.ListRootsParams{}); err != nil {
			logEvent("roots_error", err.Error())
		} else {
			logEvent("roots", r.Roots)
		}
		cancel()
		s, ok := byName[a.Name]
		if !ok {
			return nil, nil, fmt.Errorf("unknown skill %q", a.Name)
		}
		b, _ := os.ReadFile(filepath.Join(s.dir, "SKILL.md"))
		var files []string
		filepath.WalkDir(s.dir, func(p string, d os.DirEntry, _ error) error {
			if !d.IsDir() && d.Name() != "SKILL.md" {
				rel, _ := filepath.Rel(s.dir, p)
				files = append(files, rel)
			}
			return nil
		})
		out := string(b) + "\n---\nSkill folder: " + s.dir + "\n"
		if len(files) > 0 {
			out += "Extra files (load with get_skill_file): " + strings.Join(files, ", ") + "\n"
		}
		return text(out), nil, nil
	})

	type fileArgs struct {
		Name string `json:"name" jsonschema:"skill name"`
		Path string `json:"path" jsonschema:"file path relative to the skill folder"`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_skill_file",
		Description: "Load an extra file of a skill, such as references/checklist.md.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, a fileArgs) (*mcp.CallToolResult, any, error) {
		logEvent("call", map[string]any{"tool": "get_skill_file", "name": a.Name, "path": a.Path})
		s, ok := byName[a.Name]
		if !ok {
			return nil, nil, fmt.Errorf("unknown skill %q", a.Name)
		}
		b, err := os.ReadFile(filepath.Join(s.dir, filepath.Clean("/"+a.Path)))
		if err != nil {
			return nil, nil, err
		}
		return text(string(b)), nil, nil
	})

	for _, s := range skills {
		s := s
		server.AddPrompt(&mcp.Prompt{Name: s.name, Description: s.desc}, func(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
			logEvent("prompt", s.name)
			b, _ := os.ReadFile(filepath.Join(s.dir, "SKILL.md"))
			return &mcp.GetPromptResult{Messages: []*mcp.PromptMessage{{Role: "user", Content: &mcp.TextContent{Text: string(b)}}}}, nil
		})
	}

	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Print(err)
	}
}
