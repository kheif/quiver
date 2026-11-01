package jsonmerge

import (
	"strings"
	"testing"
)

func TestMerge(t *testing.T) {
	tests := []struct {
		name, base, ours, theirs, want string
	}{
		{"one side changed", `{"a":1}`, `{"a":1}`, `{"a":2}`, `{"a":2}`},
		{"different keys", `{"a":1}`, `{"a":1,"b":2}`, `{"a":1,"c":3}`, `{"a":1,"b":2,"c":3}`},
		{"set union", `{"l":["x"]}`, `{"l":["x","y"]}`, `{"l":["x","z"]}`, `{"l":["x","y","z"]}`},
		{"set removal", `{"l":["x","y"]}`, `{"l":["y","z"]}`, `{"l":["x","y"]}`, `{"l":["y","z"]}`},
		{"removed on both", `{"l":["x","y"]}`, `{"l":["y"]}`, `{"l":["y","w"]}`, `{"l":["w","y"]}`},
		{"no base", ``, `{"v":2,"l":["a"]}`, `{"v":2,"l":["b"]}`, `{"l":["a","b"],"v":2}`},
		{"key removed", `{"p":{"a":1},"q":1}`, `{"q":1}`, `{"p":{"a":1},"q":2}`, `{"q":2}`},
		{"change beats removal", `{"p":{"l":["a"]}}`, `{}`, `{"p":{"l":["a","b"]}}`, `{"p":{"l":["a","b"]}}`},
		{"nested objects", `{"p":{"x":{"l":[]}}}`, `{"p":{"x":{"l":["a"]}}}`, `{"p":{"x":{"l":[]},"y":{"l":["b"]}}}`, `{"p":{"x":{"l":["a"]},"y":{"l":["b"]}}}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var base []byte
			if tt.base != "" {
				base = []byte(tt.base)
			}
			got, err := Merge(base, []byte(tt.ours), []byte(tt.theirs))
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tt.want {
				t.Errorf("got %s, want %s", got, tt.want)
			}
		})
	}
}

func TestMergeConflict(t *testing.T) {
	_, err := Merge([]byte(`{"p":{"name":"a"}}`), []byte(`{"p":{"name":"b"}}`), []byte(`{"p":{"name":"c"}}`))
	if err == nil || !strings.Contains(err.Error(), "p.name changed differently on both sides") {
		t.Errorf("err = %v", err)
	}
}
