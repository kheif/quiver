package registry

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func readRegistryFile(t *testing.T, home string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(home, "registry.json"))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestVersion2FileReadsAsBefore(t *testing.T) {
	setHome(t)
	writeFile(t, `{"version": 2, "global": ["a"], "alwaysOn": ["a"], "projects": {"p": {"name": "p", "skills": ["b"]}}}`)
	r, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if r.Version != 2 || len(r.Searchable) != 0 || r.LevelIn("p", "a") != AlwaysOn || r.LevelIn("p", "b") != OnDemand {
		t.Errorf("got %+v", r)
	}
}

func TestWriteVersionFollowsSearchable(t *testing.T) {
	home := setHome(t)
	if err := Update(func(r *Registry) error {
		r.Projects["p"] = &Project{Name: "p"}
		return r.AssignSearchable("p", "x")
	}); err != nil {
		t.Fatal(err)
	}
	data := readRegistryFile(t, home)
	if !strings.Contains(data, `"version": 3`) || !strings.Contains(data, `"searchable": [`) {
		t.Errorf("project searchable not written as version 3:\n%s", data)
	}
	if err := Update(func(r *Registry) error {
		r.AssignSearchableGlobal("y")
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := Update(func(r *Registry) error {
		_, err := r.Unassign("p", "x")
		r.UnassignGlobal("y")
		return err
	}); err != nil {
		t.Fatal(err)
	}
	data = readRegistryFile(t, home)
	if !strings.Contains(data, `"version": 2`) || strings.Contains(data, "searchable") {
		t.Errorf("last searchable entry removed, want version 2 without the list:\n%s", data)
	}
}

func TestEmptySearchableListsOmitted(t *testing.T) {
	home := setHome(t)
	if err := Update(func(r *Registry) error {
		r.Projects["p"] = &Project{Name: "p"}
		r.AssignGlobal("a")
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if data := readRegistryFile(t, home); strings.Contains(data, "searchable") {
		t.Errorf("empty list written:\n%s", data)
	}
}

func TestNewerVersionRefused(t *testing.T) {
	setHome(t)
	writeFile(t, `{"version": 4}`)
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "update quiver") {
		t.Errorf("got %v", err)
	}
}

func TestSkillInTwoListsNormalisedOnRead(t *testing.T) {
	setHome(t)
	writeFile(t, `{"version": 3, "global": ["g"], "alwaysOn": ["g"], "searchable": ["g", "s"],
		"projects": {"p": {"name": "p", "skills": ["a"], "alwaysOn": ["a"], "searchable": ["a", "b"]}}}`)
	r, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	p := r.Projects["p"]
	if !reflect.DeepEqual(r.Searchable, []string{"s"}) || !reflect.DeepEqual(p.Searchable, []string{"b"}) {
		t.Errorf("got %v and %v", r.Searchable, p.Searchable)
	}
	if r.LevelIn("p", "g") != AlwaysOn || r.LevelIn("p", "a") != AlwaysOn {
		t.Error("a served skill lost its level")
	}
}

func TestAssigningCallsMoveTheSkill(t *testing.T) {
	r := Default()
	r.Projects["p"] = &Project{Name: "p"}
	_ = r.Assign("p", "a")
	_ = r.MarkAlwaysOn("p", "a")
	_ = r.AssignSearchable("p", "a")
	p := r.Projects["p"]
	if len(p.Skills) != 0 || len(p.AlwaysOn) != 0 || !reflect.DeepEqual(p.Searchable, []string{"a"}) {
		t.Errorf("Searchable did not move: %+v", p)
	}
	_ = r.Assign("p", "a")
	if len(p.Searchable) != 0 || !reflect.DeepEqual(p.Skills, []string{"a"}) {
		t.Errorf("Assign did not move back: %+v", p)
	}
	r.AssignGlobal("g")
	_ = r.MarkAlwaysOnGlobal("g")
	r.AssignSearchableGlobal("g")
	if len(r.Global) != 0 || len(r.AlwaysOn) != 0 || !reflect.DeepEqual(r.Searchable, []string{"g"}) {
		t.Errorf("global did not move: %+v", r)
	}
	r.AssignGlobal("g")
	if len(r.Searchable) != 0 {
		t.Errorf("AssignGlobal left it Searchable: %+v", r)
	}
}

func TestLevelAcrossScopes(t *testing.T) {
	r := Default()
	r.Projects["p"] = &Project{Name: "p"}
	r.Projects["q"] = &Project{Name: "q"}
	r.AssignSearchableGlobal("a", "b", "c")
	_ = r.Assign("p", "a", "c")
	_ = r.MarkAlwaysOn("p", "c")
	cases := []struct {
		project, skill string
		want           Level
	}{
		{"p", "a", OnDemand},   // Searchable global, listed in the project
		{"p", "c", AlwaysOn},   // always on in the project
		{"p", "b", Searchable}, // Searchable global only
		{"q", "a", Searchable},
		{"", "a", Searchable},
		{"p", "z", NotAssigned},
		{"none", "b", Searchable}, // a project without a folder, or unknown
	}
	for _, c := range cases {
		if got := r.LevelIn(c.project, c.skill); got != c.want {
			t.Errorf("LevelIn(%q, %q) = %v, want %v", c.project, c.skill, got, c.want)
		}
	}
	r.Global = []string{"g"}
	_ = r.AssignSearchable("q", "g")
	if r.LevelIn("q", "g") != OnDemand {
		t.Error("global served skill must stay served when Searchable in the project")
	}
}

func mergeJSON(t *testing.T, base, ours, theirs string) *Registry {
	t.Helper()
	r, err := Merge([]byte(base), []byte(ours), []byte(theirs))
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestMergeSearchableAgainstServed(t *testing.T) {
	base := `{"version": 2, "global": [], "alwaysOn": [], "projects": {"p": {"name": "p", "skills": []}}}`
	cases := map[string]struct{ ours, theirs string }{
		"on demand": {
			`{"version": 3, "global": [], "alwaysOn": [], "searchable": ["x"], "projects": {"p": {"name": "p", "skills": [], "searchable": ["y"]}}}`,
			`{"version": 2, "global": ["x"], "alwaysOn": [], "projects": {"p": {"name": "p", "skills": ["y"]}}}`,
		},
		"always on": {
			`{"version": 3, "global": [], "alwaysOn": [], "searchable": ["x"], "projects": {"p": {"name": "p", "skills": [], "searchable": ["y"]}}}`,
			`{"version": 2, "global": ["x"], "alwaysOn": ["x"], "projects": {"p": {"name": "p", "skills": ["y"], "alwaysOn": ["y"]}}}`,
		},
	}
	for label, c := range cases {
		t.Run(label, func(t *testing.T) {
			r := mergeJSON(t, base, c.ours, c.theirs)
			p := r.Projects["p"]
			if len(r.Searchable) != 0 || len(p.Searchable) != 0 {
				t.Errorf("still Searchable: %v %v", r.Searchable, p.Searchable)
			}
			if !reflect.DeepEqual(r.Global, []string{"x"}) || !reflect.DeepEqual(p.Skills, []string{"y"}) {
				t.Errorf("served level lost: %+v", r)
			}
			if label == "always on" && (!reflect.DeepEqual(r.AlwaysOn, []string{"x"}) || !reflect.DeepEqual(p.AlwaysOn, []string{"y"})) {
				t.Errorf("always-on mark lost: %v %v", r.AlwaysOn, p.AlwaysOn)
			}
			if r.Version != 2 {
				t.Errorf("version %d with no searchable list", r.Version)
			}
		})
	}
}

func TestMergeVersionFollowsSearchable(t *testing.T) {
	base := `{"version": 2, "global": [], "alwaysOn": [], "projects": {}}`
	ours := `{"version": 3, "global": [], "alwaysOn": [], "searchable": ["x"], "projects": {}}`
	theirs := `{"version": 2, "global": ["a"], "alwaysOn": [], "projects": {}}`
	r := mergeJSON(t, base, ours, theirs)
	if r.Version != 3 || !reflect.DeepEqual(r.Searchable, []string{"x"}) || !reflect.DeepEqual(r.Global, []string{"a"}) {
		t.Errorf("got %+v", r)
	}
	// Both lists merge as sets.
	other := `{"version": 3, "global": [], "alwaysOn": [], "searchable": ["z"], "projects": {}}`
	r = mergeJSON(t, base, ours, other)
	if !reflect.DeepEqual(r.Searchable, []string{"x", "z"}) {
		t.Errorf("got %v", r.Searchable)
	}
	// A result without a searchable list is version 2.
	r = mergeJSON(t, base, base, theirs)
	if r.Version != 2 {
		t.Errorf("version %d", r.Version)
	}
}

func TestReplaceKeepsSearchable(t *testing.T) {
	r := Default()
	r.Projects["p"] = &Project{Name: "p"}
	r.AssignSearchableGlobal("old")
	_ = r.AssignSearchable("p", "old")
	r.Replace("old", "new")
	if !reflect.DeepEqual(r.Searchable, []string{"new"}) || !reflect.DeepEqual(r.Projects["p"].Searchable, []string{"new"}) {
		t.Errorf("got %+v", r)
	}
}

func TestMergeRefusesNewerVersion(t *testing.T) {
	ok := `{"version": 2, "global": [], "alwaysOn": [], "projects": {}}`
	newer := `{"version": 4, "global": [], "alwaysOn": [], "projects": {}}`
	if _, err := Merge([]byte(ok), []byte(ok), []byte(newer)); err == nil || !strings.Contains(err.Error(), "update quiver") {
		t.Errorf("got %v", err)
	}
}

func TestMergeIgnoresVersionDifference(t *testing.T) {
	base := `{"version": 1, "global": [], "alwaysOn": [], "projects": {}}`
	ours := `{"version": 2, "global": ["a"], "alwaysOn": [], "projects": {}}`
	theirs := `{"version": 3, "global": [], "alwaysOn": [], "searchable": ["b"], "projects": {}}`
	r := mergeJSON(t, base, ours, theirs)
	if r.Version != 3 || !reflect.DeepEqual(r.Global, []string{"a"}) || !reflect.DeepEqual(r.Searchable, []string{"b"}) {
		t.Errorf("got %+v", r)
	}
}
