package server

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/kheif/quiver/internal/library"
	"github.com/kheif/quiver/internal/registry"
)

// MaxAlwaysOn is the size limit of the always-on block, in characters.
// Claude Code passes a hook's text whole up to 10,000 characters and only a
// preview above that; the other assistants take at least as much
// (docs/spike-hooks.md).
const MaxAlwaysOn = 10000

const (
	alwaysOnStart = "<quiver-always-on>\nAlways-on skills from Quiver. Follow each one for the whole conversation.\n"
	alwaysOnEnd   = "\nExtra files of these skills are loaded with the get_skill_file tool of the quiver MCP server.\n</quiver-always-on>\n"
)

// AlwaysOn is what a session-start hook prints for a session.
type AlwaysOn struct {
	// Text is the marked block, or "" when no skill is always-on.
	Text string
	// IDs are the skills whose text is in the block, in its order.
	IDs []string
	// Listed are the skills that did not fit; the block names them so the
	// model loads them with get_skill.
	Listed []string
}

// AlwaysOnBlock builds the always-on block for the project, or for the
// global skills alone when projectID is "": global skills first, then the
// project's, each with the text get_skill returns. Skills no longer in the
// library are left out. When the block would pass MaxAlwaysOn, it keeps
// whole skills in order while they fit and names the rest.
func AlwaysOnBlock(projectID string) (AlwaysOn, error) {
	r, err := registry.Load()
	if err != nil {
		return AlwaysOn{}, err
	}
	l, err := library.Load()
	if err != nil {
		return AlwaysOn{}, err
	}
	var ids, sections []string
	for _, id := range r.AlwaysOnInOrder(projectID) {
		sk, err := l.Get(id)
		if err != nil {
			continue
		}
		text, err := skillText(sk)
		if err != nil {
			return AlwaysOn{}, fmt.Errorf("%s: %w", id, err)
		}
		ids = append(ids, id)
		sections = append(sections, "\n## "+id+"\n"+text)
	}
	if len(ids) == 0 {
		return AlwaysOn{}, nil
	}
	// The longest prefix of whole skills that fits, with a line for the rest.
	for n := len(ids); ; n-- {
		var b strings.Builder
		b.WriteString(alwaysOnStart)
		for _, s := range sections[:n] {
			b.WriteString(s)
		}
		if n < len(ids) {
			fmt.Fprintf(&b, "\nThese always-on skills did not fit here: %s. Before your first reply, load each one with the get_skill tool of the quiver MCP server and follow it for the whole conversation.\n", strings.Join(ids[n:], ", "))
		}
		b.WriteString(alwaysOnEnd)
		if n == 0 || utf8.RuneCountInString(b.String()) <= MaxAlwaysOn {
			return AlwaysOn{Text: b.String(), IDs: ids[:n:n], Listed: ids[n:]}, nil
		}
	}
}
