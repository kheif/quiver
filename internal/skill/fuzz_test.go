package skill

import "testing"

func FuzzParse(f *testing.F) {
	for _, seed := range []string{
		"---\nname: pdf\ndescription: Merge PDF files.\n---\nUse qpdf.\n",
		"\xef\xbb\xbf---\r\nname: a\r\n---\r\n",
		"---\n---\n",
		"---\nname: [a, b\n---\n",
		"---\ndescription: |\n  two\n  lines\n---\n",
		"# No frontmatter\n",
		"---\nname: x",
	} {
		f.Add([]byte(seed))
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		s, err := parse(data)
		if err == nil && s.Frontmatter == nil {
			t.Error("nil frontmatter without an error")
		}
	})
}
