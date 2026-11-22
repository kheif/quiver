//go:build unix

package embed

import (
	"os"
	"syscall"
)

// mapFile maps path read-only and returns its bytes and the function that
// unmaps them. The file is only ever replaced by a rename (TASK62.7), never
// changed in place, so the mapping cannot shrink under a running process.
func mapFile(path string) ([]byte, func() error, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = f.Close() }()
	st, err := f.Stat()
	if err != nil {
		return nil, nil, err
	}
	size := st.Size()
	if size < headerLenSize || size > maxFileSize {
		return nil, nil, errFileSize(size)
	}
	raw, err := syscall.Mmap(int(f.Fd()), 0, int(size), syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil {
		return nil, nil, err
	}
	return raw, func() error { return syscall.Munmap(raw) }, nil
}
