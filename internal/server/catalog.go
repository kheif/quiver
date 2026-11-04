package server

import "strings"

// maxDescription guards the catalog against huge descriptions. The Agent
// Skills format allows at most 1024 characters.
const maxDescription = 1024

// usage teaches the assistant to use the server. The catalog and the
// always-on skills are not in the instructions: Claude Code cuts them at about
// 4096 characters, so the session-start hook carries them (BuildSessionContext).
const usage = "Quiver serves skills: reusable instructions for specific tasks. At the start of a session Quiver adds a <quiver-skills> block with the id and description of the skills served here, and a <quiver-always-on> block with the always-on skills in full, to the conversation. If there is no <quiver-skills> block, call the list_skills tool of this server before your first reply.\nBefore starting a task, check whether a skill fits. If it does, call the get_skill tool of this server with its id and follow it. The get_skill_file tool loads an extra file of a skill, and search_skills finds skills by words.\n\n"

// createHint points to create_skill. Claude Code defers MCP tools, so
// without it the assistant does not know the tool exists, and saves a
// skill some other way.
const createHint = "To save a new skill, for example from this conversation, call the create_skill tool of this server; do not write skill files yourself. Unless the user already said, ask whether the skill is for this project only or for every folder.\n"

// pointer tells the assistant to load the always-on skills before its first
// reply, for an assistant whose session-start hook has not added them. It
// returns "" when no skill is always-on.
func pointer(skills []servedSkill) string {
	var ids []string
	for _, s := range skills {
		if s.AlwaysOn {
			ids = append(ids, oneLine(s.ID))
		}
	}
	if len(ids) == 0 {
		return ""
	}
	return "Always-on skills: " + strings.Join(ids, ", ") + ". If the conversation already has a <quiver-always-on> block, these skills are loaded: follow them. Otherwise call the get_skill tool of this server for each before your first reply and follow it for the whole conversation.\n\n"
}

// shorten puts a description on one line and cuts it to maxDescription
// characters.
func shorten(desc string) string {
	desc = oneLine(desc)
	if r := []rune(desc); len(r) > maxDescription {
		return string(r[:maxDescription-1]) + "…"
	}
	return desc
}
