package fsutil

import (
	"path/filepath"
	"sync"
	"testing"
)

func TestConcurrentWritersDoNotCorrupt(t *testing.T) {
	home := t.TempDir()
	t.Setenv("QUIVER_HOME", home)
	path := filepath.Join(home, "counter.json")
	if err := WriteJSONAtomic(path, doc{}); err != nil {
		t.Fatal(err)
	}

	const writers, rounds = 4, 10
	var wg sync.WaitGroup
	for range writers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range rounds {
				err := WithLock(func() error {
					var d doc
					if err := ReadJSON(path, &d); err != nil {
						return err
					}
					d.N++
					return WriteJSONAtomic(path, d)
				})
				if err != nil {
					t.Error(err)
					return
				}
			}
		}()
	}
	wg.Wait()

	var got doc
	if err := ReadJSON(path, &got); err != nil {
		t.Fatal(err)
	}
	if got.N != writers*rounds {
		t.Errorf("got %d increments, want %d", got.N, writers*rounds)
	}
}
