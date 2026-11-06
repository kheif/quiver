package tui

import (
	"regexp"
	"strings"
	"unicode"
)

// Skills come from sources the person may not have written, so nothing that
// a skill contains reaches the screen as it is: a control character could
// move the cursor, recolor the screen or break the frame, and a character
// that reorders text could make a line read differently from what it says.

// dropped reports whether r is removed: the bidirectional controls and the
// invisible characters that reorder or hide text.
func dropped(r rune) bool {
	switch {
	case r == 0x061C, r == 0x200E, r == 0x200F:
		return true
	case r >= 0x202A && r <= 0x202E, r >= 0x2066 && r <= 0x2069:
		return true
	case r == 0xFEFF, r == 0xFFFD:
		return true
	}
	return false
}

// oneLine returns s on one line: every control character and line break
// becomes a space, runs of spaces collapse and the ends are trimmed.
func oneLine(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case dropped(r):
		case unicode.IsControl(r), r == 0x2028, r == 0x2029:
			b.WriteRune(' ')
		default:
			b.WriteRune(r)
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

// lines returns s with only its line breaks kept: tabs become four spaces
// and every other control character is removed.
func lines(s string) string {
	s = strings.NewReplacer("\r\n", "\n", "\r", "\n", " ", "\n", " ", "\n", "\t", "    ").Replace(s)
	var b strings.Builder
	for _, r := range s {
		if dropped(r) || (unicode.IsControl(r) && r != '\n') {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// reasonFirst turns "open /path/file: permission denied" into
// "permission denied (open /path/file)": the cause comes first, so it is
// still visible when a long path fills the line.
var reasonFirst = regexp.MustCompile(`^(open|stat|lstat|read|write|rename|remove|mkdir|chmod|decode|readdir|readlink) (\S+): (.+)$`)

// errText is err as one clean line, cause first.
func errText(err error) string {
	s := oneLine(err.Error())
	if m := reasonFirst.FindStringSubmatch(s); m != nil {
		return m[3] + " (" + m[1] + " " + m[2] + ")"
	}
	return s
}
