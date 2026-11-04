package server

import (
	"strings"
	"testing"
)

func TestServerInfo(t *testing.T) {
	t.Setenv("QUIVER_HOME", t.TempDir())
	info := connect(t, New(Options{Version: "1.2.3", Log: discard})).InitializeResult().ServerInfo
	if info.Name != "quiver" || info.Version != "1.2.3" {
		t.Errorf("got %+v", info)
	}
}

// A server that started outside a project serves the project of an MCP root,
// as VS Code sends after initialization. The server-to-client roots request is
// driven by applyRoots directly: the in-memory transport does not deliver the
// initialized notification that would call it in production.
func TestApplyRootsServesProject(t *testing.T) {
	w, _ := skillsEnv(t)
	root := w.Project.Root
	outside := t.TempDir() // the server starts here, in no project

	s, sess, p := newServer(Options{Version: "dev", Log: discard, Where: Where{Dir: outside, From: "working directory"}})
	cs := connect(t, s)

	// Before the roots are applied, the project's skill is not served.
	before, _ := call(t, cs, "list_skills", nil)
	if strings.Contains(before, "db-migration") {
		t.Fatalf("the project was served before any root:\n%s", before)
	}

	uri := fileURI(root)
	if !sess.applyRoots(s, p, []string{"http://x", uri}, discard) {
		t.Fatal("applyRoots did not serve a project")
	}
	out, isErr := call(t, cs, "list_skills", nil)
	if isErr {
		t.Fatalf("list_skills: %s", out)
	}
	if !strings.Contains(out, "db-migration") {
		t.Errorf("the root project's skill was not served:\n%s", out)
	}
}
