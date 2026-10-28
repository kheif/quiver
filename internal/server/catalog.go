package server

import (
	"fmt"
	"strings"
)

// maxInstructions is the size limit of the server's instructions, in
// bytes. Claude Code cuts them at about 4096 characters; bytes are never
// fewer than characters, so the limit holds for any text.
const maxInstructions = 3800

// maxDescription is the most characters of a description in the catalog.
const maxDescription = 150

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

// catalog lists the skills for the server's instructions, in at most
// budget bytes. Descriptions are shortened to maxDescription characters.
// If the list is still too long, descriptions are dropped from the end,
// then IDs, and the list ends with a pointer to list_skills. It returns ""
// when there are no skills.
func catalog(skills []servedSkill, budget int) string {
	if len(skills) == 0 {
		return ""
	}
	lines := make([]string, len(skills))
	size := len(catalogHeader)
	for i, s := range skills {
		lines[i] = fmt.Sprintf("- %s: %s\n", s.ID, shorten(s.Description))
		size += len(lines[i])
	}
	for i := len(lines) - 1; i >= 0 && size > budget; i-- {
		short := "- " + skills[i].ID + "\n"
		size += len(short) - len(lines[i])
		lines[i] = short
	}
	if size <= budget {
		return catalogHeader + strings.Join(lines, "")
	}
	var b strings.Builder
	b.WriteString(catalogHeader)
	for i, line := range lines {
		// Line i fits only if the pointer to the lines after it fits too.
		if b.Len()+len(line)+len(more(len(lines)-i-1)) > budget {
			b.WriteString(more(len(lines) - i))
			break
		}
		b.WriteString(line)
	}
	return b.String()
}

// more is the last catalog line when n skills are left out.
func more(n int) string {
	return fmt.Sprintf("…and %d more: call list_skills\n", n)
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
