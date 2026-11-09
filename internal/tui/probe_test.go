package tui

import (
	"errors"
	"slices"
	"testing"
)

func TestBackgroundCheckAsksEverySourceOnceAndMarksTheOnesWithNews(t *testing.T) {
	f := fixture()
	f.updates = map[string]bool{"team": true}
	m := start(t, f)
	if !slices.Equal(f.probed, []string{"team", "anthropic"}) {
		t.Fatalf("asked %v, want each source once in list order", f.probed)
	}
	contains(t, m, "team has updates — press u to review")
	m = keys(t, m, "3")
	contains(t, m, "Updates available")
	if len(f.applied) != 0 || len(f.calls) != 0 || m.preview != nil {
		t.Errorf("the background check changed something: applied %v, calls %v", f.applied, f.calls)
	}
}

func TestSeveralSourcesWithNewsShowACount(t *testing.T) {
	f := fixture()
	f.updates = map[string]bool{"team": true, "anthropic": true}
	contains(t, start(t, f), "2 sources have updates — press u")
}

func TestNoNoticeWhenNothingIsNew(t *testing.T) {
	m := start(t, fixture())
	lacks(t, m, "has updates", "sources have updates")
	if m.status != "" {
		t.Errorf("status %q", m.status)
	}
}

func TestUOnAnySectionReviewsTheSourceBeforeItPulls(t *testing.T) {
	f := fixture()
	f.updates = map[string]bool{"team": true}
	f.checks = map[string]Preview{"team": {Source: "team", From: "abc1234", To: "9999999", Ask: true, Lines: []string{"team/migrate-check changed"}}}
	m := keys(t, start(t, f), "u")
	if f.callList() != "check team" {
		t.Fatalf("calls %q", f.callList())
	}
	contains(t, m, "team/migrate-check changed")
	if len(f.applied) != 0 {
		t.Fatal("pulled before the review was answered")
	}
	m = keys(t, m, "y")
	if !slices.Equal(f.applied, []string{"team"}) {
		t.Fatalf("applied %v", f.applied)
	}
	lacks(t, m, "has updates")
}

func TestUDoesNothingOnOtherSectionsWithoutNews(t *testing.T) {
	f := fixture()
	m := keys(t, start(t, f), "u")
	if len(f.calls) != 0 || m.sec != secSkills {
		t.Errorf("calls %v, section %v", f.calls, m.sec)
	}
}

func TestBackgroundCheckWithoutNetworkShowsNothingAndStopsEarly(t *testing.T) {
	f := fixture()
	f.sources = append(f.sources, Source{Name: "third", Repo: "x/y", Skills: 1, Commit: "1234567"})
	f.probeErr = errors.New("could not resolve host")
	m := start(t, f)
	if len(f.probed) != probeFailuresStop {
		t.Errorf("asked %v, want the run to stop after %d failures", f.probed, probeFailuresStop)
	}
	lacks(t, m, "has updates", "could not resolve")
	if m.status != "" || m.failed {
		t.Errorf("a failed background check was reported: %q", m.status)
	}
	// The panel works as if nothing happened.
	m = keys(t, m, "3")
	if m.sec != secSources {
		t.Fatal("panel did not react to keys")
	}
}

func TestBackgroundCheckSkipsSourcesWithoutAClone(t *testing.T) {
	f := fixture()
	f.sources[0].Commit = ""
	start(t, f)
	if !slices.Equal(f.probed, []string{"anthropic"}) {
		t.Errorf("asked %v", f.probed)
	}
}

func TestBackgroundCheckKeepsWhatAPersonAlreadyChecked(t *testing.T) {
	f := fixture()
	f.updates = map[string]bool{"team": true}
	m := New(f)
	m.checks["team"] = "Up to date (abc1234)"
	next, _ := m.probed(probedMsg{source: "team", behind: true})
	m = next.(Model)
	if m.checks["team"] != "Up to date (abc1234)" {
		t.Errorf("the background check overwrote %q", m.checks["team"])
	}
}
