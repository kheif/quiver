package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
)

// projectNames turns project IDs into the names the panel shows.
func (m Model) projectNames(ids []string) []string {
	names := make([]string, 0, len(ids))
	for _, id := range ids {
		name := id
		for _, p := range m.data.Projects {
			if p.ID == id {
				name = p.Name
			}
		}
		names = append(names, name)
	}
	return names
}

// askFork asks for the name of the local copy of a source skill. The copy
// takes the original's place wherever it is assigned.
func (m Model) askFork() (Model, tea.Cmd) {
	s, ok := m.selectedSkill()
	if !ok {
		return m, nil
	}
	if s.Local {
		m.say(s.ID+" is a local skill; press e to edit it.", false, true)
		return m, nil
	}
	return m.ask(&asking{
		title: "FORK · " + s.ID, prompt: "Name: ", verb: "fork", value: s.Name,
		hint: "A local copy you can edit. It takes the original's place wherever " + s.ID + " is assigned. Keep the name, or give it another.",
		submit: func(m Model, name string) (Model, tea.Cmd) {
			if name == "" {
				return m.fail("A skill name cannot be empty."), nil
			}
			if err := m.checkNewName(name); err != nil {
				return m.fail(errText(err)), nil
			}
			key := "fork:" + s.ID
			if !m.start(key, "forking "+s.ID) {
				return m, nil
			}
			be := m.be
			return m, func() tea.Msg {
				id, note, err := be.Fork(s.ID, name)
				return doneMsg{key: key, text: note, err: err, keep: id}
			}
		},
	})
}

// checkNewName tells at once that a name cannot be used for a new local
// skill, before the person has typed the rest.
func (m Model) checkNewName(name string) error {
	if err := m.be.CheckName(name); err != nil {
		return err
	}
	if _, taken := m.skillByID(name); taken {
		return fmt.Errorf("a skill named %s already exists; choose another name", name)
	}
	return nil
}

// askNewSkill asks for a name, a description and where the skill goes, then
// creates it. Without a description the editor opens.
func (m Model) askNewSkill() (Model, tea.Cmd) {
	where := ""
	if m.data.Project != nil {
		where = m.data.Project.Path
		if where == "" && len(m.data.Project.Paths) > 0 {
			where = m.data.Project.Paths[0]
		}
	}
	scope := "none"
	if where != "" {
		scope = "project"
	}
	return m.ask(&asking{
		title: "NEW SKILL", prompt: "Name: ", verb: "next",
		hint: "A short name with letters, digits and dashes, such as release-notes.",
		submit: func(m Model, name string) (Model, tea.Cmd) {
			if name == "" {
				return m.fail("A skill name cannot be empty."), nil
			}
			if err := m.checkNewName(name); err != nil {
				return m.fail(errText(err)), nil
			}
			return m.ask(&asking{
				title: "NEW SKILL · " + name, prompt: "Description: ", verb: "next",
				hint: "One sentence on when the assistant should use it. Leave it empty to write the skill in your editor.",
				submit: func(m Model, description string) (Model, tea.Cmd) {
					return m.ask(&asking{
						title: "NEW SKILL · " + name, prompt: "Assign to: ", verb: "create", value: scope,
						hint: m.scopeHint(where),
						submit: func(m Model, answer string) (Model, tea.Cmd) {
							return m.createSkill(name, description, answer, where)
						},
					})
				},
			})
		},
	})
}

// scopeHint explains the choices of the last question of a new skill.
func (m Model) scopeHint(where string) string {
	if where == "" {
		return "none: only create it, assign it later. global: on demand in every folder."
	}
	return fmt.Sprintf("project: on demand in %s. global: on demand in every folder. none: only create it, assign it later.", m.data.Project.Name)
}

// createSkill runs the creation once the three answers are in.
func (m Model) createSkill(name, description, answer, where string) (Model, tea.Cmd) {
	var global bool
	switch strings.ToLower(answer) {
	case "project", "p", "":
		if where == "" {
			return m.fail("There is no project folder to assign it to. Type global or none."), nil
		}
	case "global", "g":
		global, where = true, ""
	case "none", "n":
		where = ""
	default:
		return m.fail(fmt.Sprintf("Type project, global or none, not %q.", answer)), nil
	}
	be := m.be
	if description == "" {
		var text string
		return m, execProcess(&execFunc{func() (err error) {
			text, err = be.NewSkill(name, "", where, global)
			return err
		}}, func(err error) tea.Msg { return doneMsg{text: text, err: err, keep: name} })
	}
	key := "new:" + name
	if !m.start(key, "creating "+name) {
		return m, nil
	}
	return m, func() tea.Msg {
		text, err := be.NewSkill(name, description, where, global)
		return doneMsg{key: key, text: text, err: err, keep: name}
	}
}

// askDelete confirms the deletion of a local skill and lists where it is
// assigned, since the deletion unassigns it there.
func (m Model) askDelete() (Model, tea.Cmd) {
	s, ok := m.selectedSkill()
	if !ok {
		return m, nil
	}
	if !s.Local {
		m.say(fmt.Sprintf("%s comes from source %s and cannot be deleted on its own. Removing the source (3, then x) deletes all of its skills.", s.ID, s.Source), false, true)
		return m, nil
	}
	var where []string
	if s.Global {
		where = append(where, "every folder")
	}
	where = append(where, m.projectNames(s.UsedIn)...)
	prompt := fmt.Sprintf("Delete skill %s from the library?", s.ID)
	if len(where) > 0 {
		prompt += "\nIt is assigned in: " + strings.Join(where, ", ") + ". The deletion unassigns it there."
	} else {
		prompt += "\nIt is not assigned anywhere."
	}
	prompt += "\nThe removal is committed in your Quiver home, so git history still has the skill."
	force := len(where) > 0
	id := s.ID
	m.confirm = &confirmation{prompt: prompt, run: func(m Model) (Model, tea.Cmd) {
		return m.runProject("delete:"+id, "deleting "+id, func() (string, error) { return m.be.Delete(id, force) })
	}}
	return m, nil
}

// askPin keeps a source skill at its current commit, or lets it follow the
// source again.
func (m Model) askPin() (Model, tea.Cmd) {
	s, ok := m.selectedSkill()
	if !ok {
		return m, nil
	}
	if s.Local {
		m.say(s.ID+" is a local skill. Only skills of a source can be pinned.", false, true)
		return m, nil
	}
	id := s.ID
	if s.Pin != "" {
		m.confirm = &confirmation{
			prompt: fmt.Sprintf("Serve %s from the current version of source %s again? It is kept at %s now, and updates change it again.", id, s.Source, s.Pin),
			run: func(m Model) (Model, tea.Cmd) {
				return m.runProject("unpin:"+id, "unpinning "+id, func() (string, error) { return m.be.Unpin(id) })
			},
		}
		return m, nil
	}
	m.confirm = &confirmation{
		prompt: fmt.Sprintf("Keep %s at its current commit of source %s? Updates then leave it alone, everywhere it is assigned, until you press p on it again.", id, s.Source),
		run: func(m Model) (Model, tea.Cmd) {
			return m.runProject("pin:"+id, "pinning "+id, func() (string, error) { return m.be.Pin(id) })
		},
	}
	return m, nil
}
