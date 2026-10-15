package library

import (
	"errors"
	"reflect"
	"testing"

	"github.com/kheif/quiver/internal/skill"
)

func TestGet(t *testing.T) {
	fixture(t)
	l, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	s, err := l.Get("pdf")
	if err != nil || s.Name != "pdf" || s.Body != "Body" {
		t.Errorf("got %+v, %v", s, err)
	}
}

func TestGetNotFoundSuggests(t *testing.T) {
	l := &Library{skills: map[string]*skill.Skill{}}
	for _, id := range []string{"pdf", "db-migration", "anthropic/pdf", "anthropic/docx"} {
		l.skills[id] = &skill.Skill{ID: id}
	}
	cases := map[string][]string{
		"pfd":          {"anthropic/pdf", "pdf"},
		"db-migratoin": {"db-migration"},
		"anthropic/pd": {"anthropic/pdf"},
		"zzzzzz":       nil,
	}
	for id, want := range cases {
		_, err := l.Get(id)
		var nf *NotFoundError
		if !errors.Is(err, ErrSkillNotFound) || !errors.As(err, &nf) {
			t.Fatalf("%s: got %v", id, err)
		}
		if !reflect.DeepEqual(nf.Suggestions, want) {
			t.Errorf("%s: suggestions %v, want %v (%v)", id, nf.Suggestions, want, err)
		}
	}
}

func TestDistance(t *testing.T) {
	for _, c := range []struct {
		a, b string
		want int
	}{{"", "abc", 3}, {"kitten", "sitting", 3}, {"pdf", "pdf", 0}, {"pfd", "pdf", 2}} {
		if got := distance(c.a, c.b); got != c.want {
			t.Errorf("distance(%q, %q) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}
