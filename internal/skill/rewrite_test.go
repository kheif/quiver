package skill

import "testing"

func TestRewrite(t *testing.T) {
	desc, body, empty := "New: text.", "  New body.  \n", ""
	src := "---\nname: pdf # the name\ndescription: Old.\nmetadata:\n  tags: [a, b]\n---\r\nOld body.\n"
	for _, c := range []struct {
		label       string
		src         string
		description *string
		body        *string
		want        string
	}{
		{"nothing", src, nil, nil, "---\nname: pdf # the name\ndescription: Old.\nmetadata:\n  tags: [a, b]\n---\nOld body.\n"},
		{"body", src, nil, &body, "---\nname: pdf # the name\ndescription: Old.\nmetadata:\n  tags: [a, b]\n---\n\nNew body.\n"},
		{"empty body", src, nil, &empty, "---\nname: pdf # the name\ndescription: Old.\nmetadata:\n  tags: [a, b]\n---\n"},
		{"description", src, &desc, nil, "---\nname: pdf # the name\ndescription: 'New: text.'\nmetadata:\n  tags: [a, b]\n---\nOld body.\n"},
		{"added description", "---\nname: pdf\n---\nBody.\n", &desc, nil, "---\nname: pdf\ndescription: 'New: text.'\n---\nBody.\n"},
		{"empty frontmatter", "---\n---\n", &desc, &body, "---\ndescription: 'New: text.'\n---\n\nNew body.\n"},
	} {
		got, err := Rewrite([]byte(c.src), c.description, c.body)
		if err != nil || string(got) != c.want {
			t.Errorf("%s: got %v\n%s\nwant\n%s", c.label, err, got, c.want)
		}
	}
	if _, err := Rewrite([]byte("no frontmatter"), &desc, nil); err == nil {
		t.Error("no frontmatter: no error")
	}
	if _, err := Rewrite([]byte("---\n- a\n---\n"), &desc, nil); err == nil {
		t.Error("list frontmatter: no error")
	}
}

func TestSetField(t *testing.T) {
	got, err := SetField([]byte("---\nname: pdf\ndescription: Read PDFs.\n---\nBody.\n"), "forkedFrom", "acme/pdf@abc123")
	if want := "---\nname: pdf\ndescription: Read PDFs.\nforkedFrom: acme/pdf@abc123\n---\nBody.\n"; err != nil || string(got) != want {
		t.Errorf("got %v\n%s", err, got)
	}
}
