package library

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/kheif/quiver/internal/fsutil"
)

// SearchSettingsFile is the search settings, relative to the Quiver home. They
// are machine-local and never synced: the engine and its model files may
// differ between machines.
const SearchSettingsFile = "state/search.json"

// SearchSettingsVersion is the newest version of the file this Quiver reads.
const SearchSettingsVersion = 1

// EngineWords is the default engine: names and descriptions, by words.
const EngineWords = "words"

// Engines returns the names of the engines Quiver knows, sorted.
func Engines() []string {
	var names []string
	for n := range engines {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

func unknownEngine(name string) error {
	return fmt.Errorf("unknown search engine %q: use %s", name, strings.Join(Engines(), " or "))
}

// LocalSettings is the "local" block of the settings file. Nothing else is
// configurable now.
type LocalSettings struct {
	// Model is the folder under cache/models/; empty means DefaultModel.
	Model string `json:"model,omitempty"`
	// MinScore is the cosine below which a vector match is dropped; nil means
	// the default of the model.
	MinScore *float64 `json:"minScore,omitempty"`
}

// EngineSetting is what the settings file says about the engine.
type EngineSetting struct {
	// Engine is the engine to use: words when the file is missing, has no
	// engine or cannot be used.
	Engine string
	// Problem says what is wrong with the file; empty when nothing is.
	Problem string
	// Local is the block "local" of the file, for the engine of that name.
	Local LocalSettings
	// Refused is true when the file has a newer version. Quiver then neither
	// reads nor rewrites it.
	Refused bool
}

func searchSettingsPath() (string, error) {
	home, err := fsutil.QuiverHome()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, filepath.FromSlash(SearchSettingsFile)), nil
}

func newerSettings(version int) error {
	return fmt.Errorf("search.json has version %d, but this quiver supports only up to %d: update quiver", version, SearchSettingsVersion)
}

// LoadEngine reads the settings file. Search is best effort, so a missing
// file, an unknown engine, a broken file and a newer version all give the
// words engine; the last three also give a Problem.
func LoadEngine() EngineSetting {
	cur := EngineSetting{Engine: EngineWords}
	path, err := searchSettingsPath()
	if err != nil {
		return cur
	}
	var f struct {
		Version int           `json:"version"`
		Engine  string        `json:"engine"`
		Local   LocalSettings `json:"local"`
	}
	if err := fsutil.ReadJSON(path, &f); err != nil {
		if !errors.Is(err, fsutil.ErrNotExist) {
			cur.Problem = "search.json cannot be read, searching with words: " + err.Error()
		}
		return cur
	}
	cur.Local = f.Local
	switch {
	case f.Version > SearchSettingsVersion:
		cur.Problem, cur.Refused, cur.Local = newerSettings(f.Version).Error()+"; searching with words", true, LocalSettings{}
	case f.Engine == "":
	case !slices.Contains(Engines(), f.Engine):
		cur.Problem = fmt.Sprintf("search.json: unknown engine %q, searching with words", f.Engine)
	case f.Local.MinScore != nil && (*f.Local.MinScore < -1 || *f.Local.MinScore > 1):
		cur.Problem = fmt.Sprintf("search.json: local.minScore %v is not a cosine between -1 and 1, searching with words", *f.Local.MinScore)
	default:
		cur.Engine = f.Engine
	}
	return cur
}

// UseEngine writes engine to the settings file, keeping any other key. An
// unknown engine is an error, and so is a file with a newer version, which is
// left as it is.
func UseEngine(engine string) error {
	if !slices.Contains(Engines(), engine) {
		return unknownEngine(engine)
	}
	path, err := searchSettingsPath()
	if err != nil {
		return err
	}
	return fsutil.WithLock(func() error {
		doc := map[string]json.RawMessage{}
		if err := fsutil.ReadJSON(path, &doc); err != nil && !errors.Is(err, fsutil.ErrNotExist) {
			return fmt.Errorf("search.json cannot be read, so it was not changed: %w", err)
		}
		if raw, ok := doc["version"]; ok {
			var v int
			if err := json.Unmarshal(raw, &v); err == nil && v > SearchSettingsVersion {
				return newerSettings(v)
			}
		}
		doc["version"] = json.RawMessage("1")
		name, _ := json.Marshal(engine)
		doc["engine"] = name
		return fsutil.WriteJSONAtomic(path, doc)
	})
}
