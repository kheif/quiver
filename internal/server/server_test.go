package server

import (
	"context"
	"testing"
)

func TestServerInfo(t *testing.T) {
	info := connect(t, New("1.2.3")).InitializeResult().ServerInfo
	if info.Name != "quiver" || info.Version != "1.2.3" {
		t.Errorf("got %+v", info)
	}
}

func TestPing(t *testing.T) {
	cs := connect(t, New("dev"))
	tools, err := cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(tools.Tools) != 1 || tools.Tools[0].Name != "ping" {
		t.Fatalf("tools = %+v", tools.Tools)
	}
	if out, isError := call(t, cs, "ping", map[string]any{}); out != "pong" || isError {
		t.Errorf("ping = %q, isError %v", out, isError)
	}
}
