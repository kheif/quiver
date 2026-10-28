package server

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

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

// addSkillPrompts registers every served skill as a prompt named after its
// ID, so the user can run it as a command: Claude Code lists it as
// /quiver:<id>. The list is read once, at startup.
func addSkillPrompts(s *mcp.Server, sess *session, log *slog.Logger) {
	_, skills, err := loadServed(sess.projectID())
	if err != nil {
		log.Warn("no skill prompts", "err", err)
		return
	}
	for _, sk := range skills {
		id := sk.ID
		s.AddPrompt(&mcp.Prompt{
			Name:        id,
			Description: sk.Description,
			Arguments:   promptArgs(),
		}, func(_ context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
			return skillPrompt(id, joinArgs(req.Params.Arguments))
		})
	}
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
