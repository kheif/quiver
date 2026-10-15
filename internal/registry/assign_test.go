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
