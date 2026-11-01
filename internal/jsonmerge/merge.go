// Package jsonmerge merges two versions of a JSON document that changed
// apart from a common base, as git does with lines.
package jsonmerge

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
)

// missing stands for a value that is not there: a key that is not in an
// object, or a file that did not exist.
type missing struct{}

// Merge merges ours and theirs, two changed versions of base. A nil
// document stands for a file that did not exist. Objects are merged key by
// key and arrays as sets: an item one side added is kept, and an item one
// side removed is dropped. A change on one side wins over a removal on the
// other. Any other value changed differently on both sides is a conflict;
// the error names its path.
func Merge(base, ours, theirs []byte) ([]byte, error) {
	var docs [3]any
	for i, data := range [][]byte{base, ours, theirs} {
		docs[i] = missing{}
		if data == nil {
			continue
		}
		if err := json.Unmarshal(data, &docs[i]); err != nil {
			return nil, err
		}
	}
	v, err := merge("", docs[0], docs[1], docs[2])
	if err != nil {
		return nil, err
	}
	if _, ok := v.(missing); ok {
		return nil, nil
	}
	return json.Marshal(v)
}

func merge(path string, base, ours, theirs any) (any, error) {
	switch {
	case reflect.DeepEqual(ours, theirs), reflect.DeepEqual(base, theirs):
		return ours, nil
	case reflect.DeepEqual(base, ours):
		return theirs, nil
	}
	_, oursGone := ours.(missing)
	_, theirsGone := theirs.(missing)
	switch {
	case oursGone:
		return theirs, nil
	case theirsGone:
		return ours, nil
	}
	if o, ok := ours.(map[string]any); ok {
		if t, ok := theirs.(map[string]any); ok {
			b, _ := base.(map[string]any)
			return mergeObjects(path, b, o, t)
		}
	}
	if o, ok := ours.([]any); ok {
		if t, ok := theirs.([]any); ok {
			b, _ := base.([]any)
			return mergeSets(b, o, t), nil
		}
	}
	if path == "" {
		path = "the document"
	}
	return nil, fmt.Errorf("%s changed differently on both sides", path)
}

func mergeObjects(path string, base, ours, theirs map[string]any) (map[string]any, error) {
	keys := map[string]bool{}
	for _, m := range []map[string]any{base, ours, theirs} {
		for k := range m {
			keys[k] = true
		}
	}
	out := map[string]any{}
	for k := range keys {
		v, err := merge(join(path, k), value(base, k), value(ours, k), value(theirs, k))
		if err != nil {
			return nil, err
		}
		if _, gone := v.(missing); !gone {
			out[k] = v
		}
	}
	return out, nil
}

func value(m map[string]any, k string) any {
	if v, ok := m[k]; ok {
		return v
	}
	return missing{}
}

func join(path, k string) string {
	if path == "" {
		return k
	}
	return path + "." + k
}

// mergeSets keeps every item of ours and theirs except those in base that
// either side removed. The result is sorted by its JSON encoding.
func mergeSets(base, ours, theirs []any) []any {
	set := func(items []any) map[string]any {
		m := map[string]any{}
		for _, it := range items {
			key, _ := json.Marshal(it)
			m[string(key)] = it
		}
		return m
	}
	b, o, t := set(base), set(ours), set(theirs)
	keys := []string{}
	for _, m := range []map[string]any{o, t} {
		for k := range m {
			_, inBase := b[k]
			_, inOurs := o[k]
			_, inTheirs := t[k]
			if inBase && (!inOurs || !inTheirs) {
				continue
			}
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	out := []any{}
	for i, k := range keys {
		if i > 0 && keys[i-1] == k {
			continue
		}
		if v, ok := o[k]; ok {
			out = append(out, v)
		} else {
			out = append(out, t[k])
		}
	}
	return out
}
