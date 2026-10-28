package server

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/kheif/quiver/internal/library"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// maxArgs is how many words of a task a prompt takes. Claude Code splits
// what the user types after the command on spaces, gives one word to each
// declared argument in order, and drops the rest; quotes do not help. So
// the task is declared as args, args2 … args32 and joined again.
const maxArgs = 32

// promptArgs declares the task arguments. Only the first is described.
func promptArgs() []*mcp.PromptArgument {
	out := []*mcp.PromptArgument{{Name: "args", Description: "what to use the skill for (optional)"}}
	for i := 2; i <= maxArgs; i++ {
		out = append(out, &mcp.PromptArgument{Name: fmt.Sprintf("args%d", i)})
	}
	return out
}

// joinArgs puts the task back together from the declared arguments.
func joinArgs(a map[string]string) string {
	words := []string{a["args"]}
	for i := 2; i <= maxArgs; i++ {
		words = append(words, a[fmt.Sprintf("args%d", i)])
	}
	return strings.Join(strings.Fields(strings.Join(words, " ")), " ")
}

// prompts keeps one prompt per served skill, named after the skill ID, so
// the user can run a skill as a command: Claude Code lists it as
// /quiver:<id>.
type prompts struct {
	mu sync.Mutex
	// desc maps the ID of each registered prompt to its description.
	desc map[string]string
}

// sync registers a prompt for each skill served in the project and removes
// the prompts of skills no longer served. The SDK then sends
// prompts/list_changed, once for a batch of changes. It reports whether the
// list changed. On error the prompts stay as they were.
func (p *prompts) sync(s *mcp.Server, projectID string) (bool, error) {
	_, skills, err := loadServed(projectID)
	if err != nil {
		return false, err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	next := make(map[string]string, len(skills))
	for _, sk := range skills {
		next[sk.ID] = sk.Description
	}
	var gone []string
	for id := range p.desc {
		if _, ok := next[id]; !ok {
			gone = append(gone, id)
		}
	}
	if len(gone) > 0 {
		s.RemovePrompts(gone...)
	}
	changed := len(gone) > 0
	for _, sk := range skills {
		if d, ok := p.desc[sk.ID]; ok && d == sk.Description {
			continue
		}
		id := sk.ID
		s.AddPrompt(&mcp.Prompt{
			Name:        id,
			Description: sk.Description,
			Arguments:   promptArgs(),
		}, func(_ context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
			return skillPrompt(id, joinArgs(req.Params.Arguments))
		})
		changed = true
	}
	p.desc = next
	return changed, nil
}

// skillPrompt returns the skill as one user message, with args appended as
// the task. The skill is read again, so an edit since startup is seen.
func skillPrompt(id, args string) (*mcp.GetPromptResult, error) {
	l, err := library.Load()
	if err != nil {
		return nil, err
	}
	sk, err := l.Get(id)
	if err != nil {
		return nil, err
	}
	content, err := skillText(sk)
	if err != nil {
		return nil, err
	}
	if args = strings.TrimSpace(args); args != "" {
		content += "\nTask: " + args + "\n"
	}
	return &mcp.GetPromptResult{
		Description: sk.Description,
		Messages:    []*mcp.PromptMessage{{Role: "user", Content: &mcp.TextContent{Text: content}}},
	}, nil
}
