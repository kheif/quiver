//go:build !unix

package embed

import "os"

// mapFile reads path into memory where the system has no mapping call used
// here (TASK65 settles Windows).
func mapFile(path string) ([]byte, func() error, error) {
	st, err := os.Stat(path)
	if err != nil {
		return nil, nil, err
	}
	if st.Size() < headerLenSize || st.Size() > maxFileSize {
		return nil, nil, errFileSize(st.Size())
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}
	return raw, func() error { return nil }, nil
}
