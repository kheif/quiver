package server

import (
	"regexp"
	"strings"
	"unicode"
)

// ownTag matches the start of one of the tags Quiver wraps its text in. A
// skill that contains one could close a block early or open a fake one.
var ownTag = regexp.MustCompile(`(?i)<(/?quiver-(?:always-on|skills|notice))`)

// hidden reports whether r is a character that shows nothing or changes how
// the text reads, and that text for an assistant has no use for: controls
// other than line breaks and tabs, zero-width characters, direction
// overrides and the Unicode tag characters that can carry a hidden message.
// Zero-width joiners stay: scripts and emoji need them.
func hidden(r rune) bool {
	switch {
	case r == '\n' || r == '\t':
		return false
	case r == '\r':
		return true
	case r < 0x20 || (r >= 0x7f && r <= 0x9f):
		return true
	case r == 0x200b || r == 0x2060 || r == 0xfeff || r == 0x180e:
		return true
	case r >= 0x202a && r <= 0x202e, r >= 0x2066 && r <= 0x2069, r == 0x200e || r == 0x200f || r == 0x061c:
		return true
	case r >= 0xe0000 && r <= 0xe007f:
		return true
	}
	return false
}

// clean returns s as it may go into a session's context: invalid UTF-8 is
// replaced, hidden characters are dropped, and Quiver's own tags are written
// so that they no longer read as tags.
func clean(s string) string {
	s = strings.ToValidUTF8(s, "�")
	s = strings.Map(func(r rune) rune {
		if hidden(r) || !unicode.IsPrint(r) && !unicode.IsSpace(r) && r != unicode.ReplacementChar && !unicode.Is(unicode.Cf, r) {
			return -1
		}
		return r
	}, s)
	return ownTag.ReplaceAllString(s, "&lt;$1")
}

// oneLine is clean for an ID or a description: a single line.
func oneLine(s string) string {
	return strings.Join(strings.Fields(clean(s)), " ")
}
