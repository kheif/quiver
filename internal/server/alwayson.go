package server

import (
	"errors"
	"fmt"
	"io/fs"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/kheif/quiver/internal/library"
	"github.com/kheif/quiver/internal/registry"
	"github.com/kheif/quiver/internal/skill"
)

// Limit is how much text one hook command can pass to an assistant. Max is
// counted in characters, or in bytes when Bytes is set (docs/spike-hooks.md).
type Limit struct {
	Max   int
	Bytes bool
}

func (l Limit) size(s string) int {
	if l.Bytes {
		return len(s)
	}
	return utf8.RuneCountInString(s)
}

const (
	alwaysOnTag = "quiver-always-on"
	skillsTag   = "quiver-skills"
	noticeTag   = "quiver-notice"

	alwaysOnIntro   = "Always-on skills from Quiver. Follow each one for the whole conversation.\n"
	alwaysOnOutro   = "\nExtra files of these skills are loaded with the get_skill_file tool of the quiver MCP server.\n"
	skillsIntro     = "The other skills served here, as id: description. Before starting a task, check whether one fits. If it does, call the get_skill tool of the quiver MCP server with its id and follow it.\n"
	continuedNotice = "(This continues the section of the previous part.)\n"
	noticeIntro     = "Quiver could not load everything for this session. Tell the user, and suggest running quiver doctor:\n"

	// fragmentReserve is the room a fragment keeps for its tags with the
	// longest part label and the notice that it continues.
	fragmentReserve = len(`<quiver-always-on part="999/999">`) + 1 + len(continuedNotice) + len("</quiver-always-on>\n")

	// minSplit is the room left in a part below which a text that does not
	// fit starts a new part instead of being cut.
	minSplit = 200
)

// SessionContext is what a session-start hook passes for a project.
type SessionContext struct {
	// Parts are the texts of the hook commands, in order: one part for each.
	// Part 1 comes first; assistants that run hooks in parallel may add the
	// parts in any order, so each part says which one it is.
	Parts []string
	// IDs are the always-on skills whose text is in the parts, in order.
	IDs []string
	// Count is the number of parts, also when only the count was asked for.
	Count int
	// Problems say which skills are missing from the parts, and why. They are
	// also in part 1, in a <quiver-notice> block, so the assistant tells the
	// user instead of working without them unseen.
	Problems []string
}

// built is what is known about one skill's text.
type built struct {
	text string
	err  error
	// size is the size of text under each kind of limit, counted when asked.
	size [2]int
	// line is the skill's line of the catalog.
	line string
	// ascii is 1 when text, as a unit, is ASCII only, 2 when it is not, 0 when
	// it was not checked yet.
	ascii int8
}

// textCache holds the text of each skill once it is built, so that it is
// built, counted and written once however many projects use it.
type textCache map[string]*built

func (c textCache) get(sk *skill.Skill) *built {
	if b, ok := c[sk.ID]; ok {
		return b
	}
	b := &built{line: fmt.Sprintf("- %s: %s\n", oneLine(sk.ID), shorten(sk.Description))}
	b.text, b.err = skillText(sk)
	c[sk.ID] = b
	return b
}

// unit returns the skill's text as a unit of the always-on section.
func (b *built) unit(id string, limit Limit) unit {
	text := "\n## " + oneLine(id) + "\n" + b.text
	k := 0
	if limit.Bytes {
		k = 1
	}
	if b.size[k] == 0 {
		b.size[k] = limit.size(text)
	}
	if b.ascii == 0 {
		b.ascii = 1
		for i := 0; i < len(text); i++ {
			if text[i] >= utf8.RuneSelf {
				b.ascii = 2
				break
			}
		}
	}
	return unit{alwaysOnTag, text, b.size[k], b.ascii == 1}
}

// unit is a piece of the text that is kept whole when a part has room for it.
type unit struct {
	tag, text string
	// size is the size of text under the limit, when it is known; 0 means
	// pack counts it. Texts of skills are counted once, however many
	// projects use them.
	size int
	// ascii is set when text holds only ASCII, so a byte is a character and
	// cutting it needs no scan.
	ascii bool
}

// fragment is the run of a section's units inside one part.
type fragment struct {
	tag     string
	text    strings.Builder
	first   bool // starts the section, so it has no continued notice
	reserve int
}

// BuildSessionContext builds what the hook passes for the project, or for the
// global skills alone when projectID is "": the always-on skills in full,
// global ones first and then the project's, each with the text get_skill
// returns, and then the id and description of every other skill served there.
// The text is split into parts of at most limit, at the end of a unit when it
// can, so that no skill and no line is left out.
//
// notes are further problems the caller found, such as a .quiver.json that
// could not be read; they go in the notice with the missing skills.
func BuildSessionContext(projectID string, limit Limit, notes ...string) (SessionContext, error) {
	r, err := registry.Load()
	if err != nil {
		return SessionContext{}, err
	}
	// A hook runs at the start of every session: it reads only the skills it
	// passes on, not the whole library.
	l, err := library.LoadFor(append(r.Served(projectID), r.AlwaysOnInOrder(projectID)...))
	if err != nil {
		return SessionContext{}, err
	}
	return buildSessionContext(r, l, projectID, limit, textCache{}, notes, true), nil
}

// problem says why a skill that is assigned is not loaded.
func problem(l *library.Library, id string, err error) string {
	if err == nil {
		for _, w := range l.Warnings {
			if strings.HasPrefix(w, "skipped "+id+":") {
				return oneLine(id) + ": " + oneLine(strings.TrimPrefix(w, "skipped "+id+": "))
			}
		}
		return oneLine(id) + ": it is assigned, but not in the library"
	}
	return oneLine(id) + ": " + oneLine(err.Error())
}

// buildSessionContext never fails for a skill: one that cannot be read is
// left out and named in Problems, so the others still reach the session.
func buildSessionContext(r *registry.Registry, l *library.Library, projectID string, limit Limit, cache textCache, notes []string, build bool) SessionContext {
	var ids, problems []string
	reported, loaded := map[string]bool{}, map[string]bool{}
	problems = append(problems, notes...)
	var units []unit
	for _, id := range r.AlwaysOnInOrder(projectID) {
		sk, err := l.Get(id)
		if err != nil {
			problems = append(problems, problem(l, id, nil))
			reported[id] = true
			continue
		}
		b := cache.get(sk)
		if b.err != nil {
			problems = append(problems, problem(l, id, b.err))
			reported[id] = true
			continue
		}
		ids = append(ids, id)
		loaded[id] = true
		units = append(units, b.unit(id, limit))
	}
	if len(ids) > 0 {
		units = append([]unit{{alwaysOnTag, alwaysOnIntro, 0, false}}, units...)
		units = append(units, unit{alwaysOnTag, alwaysOnOutro, 0, false})
	}
	var lines []unit
	for _, id := range r.Served(projectID) {
		if loaded[id] || reported[id] {
			continue
		}
		sk, err := l.Get(id)
		if err != nil {
			problems = append(problems, problem(l, id, nil))
			continue
		}
		lines = append(lines, unit{skillsTag, cache.get(sk).line, 0, false})
	}
	if len(lines) > 0 {
		units = append(units, unit{skillsTag, skillsIntro, 0, false})
		units = append(units, lines...)
	}
	if len(problems) > 0 {
		var b strings.Builder
		b.WriteString(noticeIntro)
		for _, p := range problems {
			b.WriteString("- " + oneLine(p) + "\n")
		}
		units = append([]unit{{noticeTag, b.String(), 0, false}}, units...)
	}
	parts, n := pack(units, limit, build)
	return SessionContext{Parts: parts, Count: n, IDs: ids, Problems: problems}
}

// PartsNeeded returns the most parts any project's text, or the global text
// alone, takes: the number of hook commands an assistant needs so that every
// folder gets all of it. It is at least 1.
func PartsNeeded(limit Limit) (int, error) {
	r, err := registry.Load()
	if err != nil {
		return 1, err
	}
	l, err := library.Load()
	if err != nil {
		return 1, err
	}
	most := 1
	cache := textCache{}
	for _, id := range append([]string{""}, projectIDs(r)...) {
		most = max(most, buildSessionContext(r, l, id, limit, cache, nil, false).Count)
	}
	return most, nil
}

func projectIDs(r *registry.Registry) []string {
	ids := make([]string, 0, len(r.Projects))
	for id := range r.Projects {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	return ids
}

// pack puts the units, in order, into as few parts as fit under limit. A unit
// that is longer than a part is cut at line ends, and at a character when a
// single line is longer than a part.
//
// With build unset it only counts: it returns no text, only how many parts
// there would be, which is all PartsNeeded asks for and takes far less time.
func pack(units []unit, limit Limit, build bool) (parts []string, n int) {
	if len(units) == 0 {
		return nil, 0
	}
	type part struct{ frags []*fragment }
	var made []*part
	var cur *part
	used := 0
	newPart := func() {
		cur = &part{}
		made = append(made, cur)
		used = 0
	}
	// section tells whether a tag has been opened before, so that a later
	// fragment of it says it continues.
	opened := map[string]bool{}
	add := func(tag, text string, size int) {
		if cur == nil {
			newPart()
		}
		last := (*fragment)(nil)
		if n := len(cur.frags); n > 0 {
			last = cur.frags[n-1]
		}
		if last == nil || last.tag != tag {
			last = &fragment{tag: tag, first: !opened[tag], reserve: fragmentReserve}
			opened[tag] = true
			cur.frags = append(cur.frags, last)
			used += last.reserve
		}
		if build {
			last.text.WriteString(text)
		}
		used += size
	}
	room := func(tag string) int {
		if cur == nil {
			return limit.Max - fragmentReserve
		}
		if n := len(cur.frags); n > 0 && cur.frags[n-1].tag == tag {
			return limit.Max - used
		}
		return limit.Max - used - fragmentReserve
	}
	for _, u := range units {
		size := u.size
		if size == 0 {
			size = limit.size(u.text)
		}
		if size <= room(u.tag) {
			add(u.tag, u.text, size)
			continue
		}
		if size <= limit.Max-fragmentReserve {
			newPart()
			add(u.tag, u.text, size)
			continue
		}
		// Longer than a part: cut it. The first piece fills what is left of
		// this part, unless that is little.
		rest := u.text
		if room(u.tag) < minSplit {
			newPart()
		}
		for rest != "" {
			n := cutAt(rest, room(u.tag), limit, u.ascii)
			piece := n
			if !u.ascii {
				piece = limit.size(rest[:n])
			}
			add(u.tag, rest[:n], piece)
			rest = rest[n:]
			if rest != "" {
				newPart()
			}
		}
	}
	if !build {
		return nil, len(made)
	}
	out := make([]string, len(made))
	for i, p := range made {
		var b strings.Builder
		for _, f := range p.frags {
			open := "<" + f.tag + ">\n"
			if len(made) > 1 {
				open = fmt.Sprintf("<%s part=\"%d/%d\">\n", f.tag, i+1, len(made))
			}
			b.WriteString(open)
			if !f.first {
				b.WriteString(continuedNotice)
			}
			b.WriteString(f.text.String())
			b.WriteString("</" + f.tag + ">\n")
		}
		out[i] = b.String()
	}
	return out, len(out)
}

// cutAt returns how many bytes of s to take so that they fit in room: up to
// the last line end that fits, else as many whole characters as fit. It always
// takes at least one character.
func cutAt(s string, room int, limit Limit, ascii bool) int {
	if room < 1 {
		room = 1
	}
	// n is the longest prefix, in whole characters, that fits.
	n, size := 0, 0
	if ascii {
		n = min(room, len(s))
	} else {
		for i, r := range s {
			w := 1
			if limit.Bytes {
				w = utf8.RuneLen(r)
			}
			if size+w > room {
				break
			}
			size += w
			n = i + utf8.RuneLen(r)
		}
	}
	if n == 0 {
		_, w := utf8.DecodeRuneInString(s)
		return w
	}
	if n == len(s) {
		return n
	}
	if i := strings.LastIndexByte(s[:n], '\n'); i >= 0 {
		return i + 1
	}
	return n
}

// FailedContext is what a hook passes when the skills could not be read at
// all: a notice, so that the assistant tells the user that this session has
// no Quiver skills instead of the session looking like one without any. It
// is empty when the Quiver home does not exist, as on a machine that does
// not use Quiver.
func FailedContext(limit Limit, err error) SessionContext {
	if errors.Is(err, fs.ErrNotExist) {
		return SessionContext{}
	}
	msg := "Quiver could not read your skills, so no always-on skills and no skill list are loaded in this session: " + oneLine(err.Error())
	parts, n := pack([]unit{{noticeTag, noticeIntro + "- " + msg + "\n", 0, false}}, limit, true)
	return SessionContext{Parts: parts, Count: n, Problems: []string{msg}}
}
