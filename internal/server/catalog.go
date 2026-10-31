package server

import (
	"fmt"
	"strings"
)

// maxInstructions is the size limit of the server's instructions, in
// bytes. Claude Code cuts them at about 4096 characters; bytes are never
// fewer than characters, so the limit holds for any text.
const maxInstructions = 3800

// maxDescription guards the catalog against huge descriptions. The Agent
// Skills format allows at most 1024 characters.
const maxDescription = 1024

const catalogHeader = "Skills (reusable instructions for specific tasks). Before starting a task, check whether one fits. If it does, call the get_skill tool of this server with its id and follow it.\n"

// createHint points to create_skill. Claude Code defers MCP tools, so
// without it the assistant does not know the tool exists, and saves a
// skill some other way.
const createHint = "To save a new skill, for example from this conversation, call the create_skill tool of this server; do not write skill files yourself. Unless the user already said, ask whether the skill is for this project only or for every folder.\n"

// pointer tells the assistant to load the always-on skills before its first
// reply. Only a pointer fits: Claude Code cuts the instructions at about 4096
// characters, and a whole SKILL.md can be longer. It returns "" when no
// skill is always-on.
func pointer(skills []servedSkill) string {
	var ids []string
	for _, s := range skills {
		if s.AlwaysOn {
			ids = append(ids, s.ID)
		}
	}
	if len(ids) == 0 {
		return ""
	}
	return "Always-on skills: " + strings.Join(ids, ", ") + ". Before your first reply in this conversation, call the get_skill tool of this server for each always-on skill and follow it for the whole conversation.\n\n"
}

// catalog lists the skills for the server's instructions, if the list fits
// in budget bytes. If it does not, it returns a pointer to list_skills
// instead, which returns every skill with its whole description. It
// returns "" when there are no skills.
func catalog(skills []servedSkill, budget int) string {
	if len(skills) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(catalogHeader)
	for _, s := range skills {
		fmt.Fprintf(&b, "- %s: %s\n", s.ID, shorten(s.Description))
	}
	if b.Len() <= budget {
		return b.String()
	}
	return listPointer(len(skills))
}

// listPointer replaces the catalog when it is too long for the
// instructions. Like the always-on pointer, it has the assistant call a
// tool before its first reply.
func listPointer(n int) string {
	return fmt.Sprintf("Skills (reusable instructions for specific tasks): this folder has %d, too many to list here. Before your first reply in this conversation, call the list_skills tool of this server; it lists the skills of this folder. Before starting a task, check whether one fits. If it does, call the get_skill tool of this server with its id and follow it.\n", n)
}

// shorten puts a description on one line and cuts it to maxDescription
// characters.
func shorten(desc string) string {
	desc = strings.Join(strings.Fields(desc), " ")
	if r := []rune(desc); len(r) > maxDescription {
		return string(r[:maxDescription-1]) + "…"
	}
	return desc
}
