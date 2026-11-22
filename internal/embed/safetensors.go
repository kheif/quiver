package embed

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"slices"
	"unsafe"
)

const (
	headerLenSize = 8
	// maxHeader and maxFileSize bound what a malformed file can make the
	// loader do before any tensor is checked against its expected shape.
	maxHeader   = 1 << 20
	maxFileSize = 1 << 30
)

func errFileSize(n int64) error {
	return fmt.Errorf("weight file: size %d is outside what a model of this kind has", n)
}

type tensorInfo struct {
	Dtype   string `json:"dtype"`
	Shape   []int  `json:"shape"`
	Offsets [2]int `json:"data_offsets"`
}

// tensorFile is a safetensors file with its header checked.
type tensorFile struct {
	data    []byte // the bytes after the header
	tensors map[string]tensorInfo
	unmap   func() error
}

// openTensors maps the file and checks its header: the length, the JSON and,
// for every tensor, the type, the shape, and the offsets against the size of
// the file. A failed check is an error that names it.
func openTensors(path string) (*tensorFile, error) {
	raw, unmap, err := mapFile(path)
	if err != nil {
		return nil, err
	}
	f, err := parseTensors(raw)
	if err != nil {
		_ = unmap()
		return nil, err
	}
	f.unmap = unmap
	return f, nil
}

func parseTensors(raw []byte) (*tensorFile, error) {
	if len(raw) < headerLenSize {
		return nil, fmt.Errorf("weight file: cut: %d bytes, no header", len(raw))
	}
	n := binary.LittleEndian.Uint64(raw[:headerLenSize])
	if n > maxHeader || n > uint64(len(raw)-headerLenSize) {
		return nil, fmt.Errorf("weight file: header length %d does not fit the file of %d bytes", n, len(raw))
	}
	var hdr map[string]json.RawMessage
	if err := json.Unmarshal(raw[headerLenSize:headerLenSize+int(n)], &hdr); err != nil {
		return nil, fmt.Errorf("weight file: header is not JSON: %w", err)
	}
	data := raw[headerLenSize+int(n):]
	f := &tensorFile{data: data, tensors: map[string]tensorInfo{}}
	for name, v := range hdr {
		if name == "__metadata__" {
			continue
		}
		var ti tensorInfo
		if err := json.Unmarshal(v, &ti); err != nil {
			return nil, fmt.Errorf("weight file: tensor %s: %w", name, err)
		}
		count := 1
		for _, d := range ti.Shape {
			if d <= 0 || count > maxFileSize/4/d {
				return nil, fmt.Errorf("weight file: tensor %s has shape %v", name, ti.Shape)
			}
			count *= d
		}
		lo, hi := ti.Offsets[0], ti.Offsets[1]
		if lo < 0 || hi < lo || hi > len(data) {
			return nil, fmt.Errorf("weight file: tensor %s has offsets %d..%d outside the %d bytes of data (cut file?)", name, lo, hi, len(data))
		}
		// Tensors the model does not use (position_ids is stored as I64) are
		// only checked for their offsets.
		if ti.Dtype == "F32" && hi-lo != count*4 {
			return nil, fmt.Errorf("weight file: tensor %s has %d bytes, its shape %v needs %d", name, hi-lo, ti.Shape, count*4)
		}
		f.tensors[name] = ti
	}
	return f, nil
}

func (f *tensorFile) close() error {
	if f.unmap == nil {
		return nil
	}
	return f.unmap()
}

// get returns a tensor of the given shape as float32 values.
func (f *tensorFile) get(name string, shape ...int) ([]float32, error) {
	ti, ok := f.tensors[name]
	if !ok {
		return nil, fmt.Errorf("weight file: tensor %s is missing", name)
	}
	if ti.Dtype != "F32" {
		return nil, fmt.Errorf("weight file: tensor %s has type %s, want F32", name, ti.Dtype)
	}
	if !slices.Equal(ti.Shape, shape) {
		return nil, fmt.Errorf("weight file: tensor %s has shape %v, want %v", name, ti.Shape, shape)
	}
	return floats(f.data[ti.Offsets[0]:ti.Offsets[1]]), nil
}

var littleEndian = binary.NativeEndian.Uint16([]byte{1, 0}) == 1

// floats turns the little-endian bytes of a tensor into float32 values. It is
// the one conversion of bytes to float32 and the only use of unsafe: a view
// of the bytes when they are aligned and the machine is little-endian, a copy
// otherwise.
func floats(b []byte) []float32 {
	n := len(b) / 4
	if n == 0 {
		return nil
	}
	if littleEndian && uintptr(unsafe.Pointer(&b[0]))%4 == 0 {
		return unsafe.Slice((*float32)(unsafe.Pointer(&b[0])), n)
	}
	out := make([]float32, n)
	for i := range out {
		out[i] = math.Float32frombits(binary.LittleEndian.Uint32(b[4*i:]))
	}
	return out
}
