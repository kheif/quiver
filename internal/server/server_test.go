package server

import "testing"

func TestServerInfo(t *testing.T) {
	t.Setenv("QUIVER_HOME", t.TempDir())
	info := connect(t, New(Options{Version: "1.2.3", Log: discard})).InitializeResult().ServerInfo
	if info.Name != "quiver" || info.Version != "1.2.3" {
		t.Errorf("got %+v", info)
	}
}
