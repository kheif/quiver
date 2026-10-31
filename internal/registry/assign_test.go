package registry

import (
	"reflect"
	"testing"
)

func withProject() *Registry {
	r := Default()
	r.Projects["app"] = &Project{Name: "app"}
	return r
}

func TestAssignUnassign(t *testing.T) {
	r := withProject()
	if err := r.Assign("app", "b", "a", "b"); err != nil {
		t.Fatal(err)
	}
	if got := r.Projects["app"].Skills; !reflect.DeepEqual(got, []string{"a", "b"}) {
		t.Errorf("skills %v", got)
	}
	r.Projects["app"].AlwaysOn = []string{"a"}
	notAssigned, err := r.Unassign("app", "a", "zzz")
	if err != nil {
		t.Fatal(err)
	}
	p := r.Projects["app"]
	if !reflect.DeepEqual(notAssigned, []string{"zzz"}) || !reflect.DeepEqual(p.Skills, []string{"b"}) || len(p.AlwaysOn) != 0 {
		t.Errorf("got %v, %+v", notAssigned, p)
	}
}

func TestAssignUnknownProject(t *testing.T) {
	r := Default()
	if err := r.Assign("nope", "a"); err == nil {
		t.Error("want error")
	}
	if _, err := r.Unassign("nope", "a"); err == nil {
		t.Error("want error")
	}
}

func TestGlobal(t *testing.T) {
	r := Default()
	r.AssignGlobal("b", "a", "a")
	r.AlwaysOn = []string{"a"}
	if !reflect.DeepEqual(r.Global, []string{"a", "b"}) {
		t.Errorf("global %v", r.Global)
	}
	notAssigned := r.UnassignGlobal("a", "c")
	if !reflect.DeepEqual(notAssigned, []string{"c"}) || !reflect.DeepEqual(r.Global, []string{"b"}) || len(r.AlwaysOn) != 0 {
		t.Errorf("got %v, %+v", notAssigned, r)
	}
}

func TestServedOverlapping(t *testing.T) {
	r := withProject()
	r.AssignGlobal("pdf", "caveman")
	_ = r.Assign("app", "pdf", "db")
	if got := r.Served("app"); !reflect.DeepEqual(got, []string{"caveman", "db", "pdf"}) {
		t.Errorf("served %v", got)
	}
	if got := r.Served(""); !reflect.DeepEqual(got, []string{"caveman", "pdf"}) {
		t.Errorf("served without project %v", got)
	}
}

func TestProjectsUsing(t *testing.T) {
	r := withProject()
	r.Projects["tool"] = &Project{Name: "tool"}
	_ = r.Assign("tool", "db")
	_ = r.Assign("app", "db", "x")
	if got := r.ProjectsUsing("db"); !reflect.DeepEqual(got, []string{"app", "tool"}) {
		t.Errorf("got %v", got)
	}
	if got := r.ProjectsUsing("none"); len(got) != 0 {
		t.Errorf("got %v", got)
	}
}

func TestAlwaysOn(t *testing.T) {
	r := withProject()
	r.AssignGlobal("caveman", "pdf")
	_ = r.Assign("app", "db")
	if err := r.MarkAlwaysOnGlobal("caveman"); err != nil {
		t.Fatal(err)
	}
	if err := r.MarkAlwaysOn("app", "db"); err != nil {
		t.Fatal(err)
	}
	if err := r.MarkAlwaysOnGlobal("db"); err == nil {
		t.Error("marked a skill that is not global")
	}
	if err := r.MarkAlwaysOn("app", "pdf"); err == nil {
		t.Error("marked a skill not assigned to the project")
	}
	if err := r.MarkAlwaysOn("nope", "db"); err == nil {
		t.Error("marked in an unknown project")
	}
	if got := r.AlwaysOnServed("app"); !reflect.DeepEqual(got, []string{"caveman", "db"}) {
		t.Errorf("always-on in app %v", got)
	}
	if got := r.AlwaysOnServed(""); !reflect.DeepEqual(got, []string{"caveman"}) {
		t.Errorf("always-on without project %v", got)
	}

	notMarked, err := r.UnmarkAlwaysOn("app", "db", "pdf")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(notMarked, []string{"pdf"}) || len(r.Projects["app"].AlwaysOn) != 0 || !reflect.DeepEqual(r.Projects["app"].Skills, []string{"db"}) {
		t.Errorf("got %v, %+v", notMarked, r.Projects["app"])
	}
	notMarked = r.UnmarkAlwaysOnGlobal("caveman", "pdf")
	if !reflect.DeepEqual(notMarked, []string{"pdf"}) || len(r.AlwaysOn) != 0 || !reflect.DeepEqual(r.Global, []string{"caveman", "pdf"}) {
		t.Errorf("got %v, %+v", notMarked, r)
	}
}

func TestReplace(t *testing.T) {
	r := Default()
	r.Global, r.AlwaysOn = []string{"acme/pdf", "z"}, []string{"acme/pdf"}
	r.Projects["a"] = &Project{Name: "a", Skills: []string{"acme/pdf", "b"}, AlwaysOn: []string{"acme/pdf"}}
	r.Projects["b"] = &Project{Name: "b", Skills: []string{"b"}}
	r.Projects["c"] = &Project{Name: "c", Skills: []string{"acme/pdf", "pdf"}}
	r.Replace("acme/pdf", "pdf")
	if !reflect.DeepEqual(r.Global, []string{"pdf", "z"}) || !reflect.DeepEqual(r.AlwaysOn, []string{"pdf"}) {
		t.Errorf("global %v, always-on %v", r.Global, r.AlwaysOn)
	}
	for id, want := range map[string][2][]string{
		"a": {{"b", "pdf"}, {"pdf"}},
		"b": {{"b"}, nil},
		"c": {{"pdf"}, nil},
	} {
		if p := r.Projects[id]; !reflect.DeepEqual(p.Skills, want[0]) || !reflect.DeepEqual(p.AlwaysOn, want[1]) {
			t.Errorf("%s: skills %v, always-on %v", id, p.Skills, p.AlwaysOn)
		}
	}
}
