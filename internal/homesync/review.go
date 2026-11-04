package homesync

import (
	"encoding/json"
	"slices"
	"sort"
	"strings"

	"github.com/kheif/quiver/internal/git"
)

// Incoming is what a pull would change in the text that sessions start with.
// What another machine pushed becomes live here as it is, so whoever can
// write to the remote decides it; "quiver update" shows such changes of
// source skills first, and sync shows these.
type Incoming struct {
	// AlwaysOn lists skills that become always-on, as "<skill>" for global
	// ones and "<skill> (project <name>)" for a project's.
	AlwaysOn []string
	// Changed lists always-on local skills whose files change.
	Changed []string
	// Sources lists sources that are added or point to another repository.
	Sources []string
}

// Empty reports whether nothing that sessions start with changes.
func (i Incoming) Empty() bool {
	return len(i.AlwaysOn) == 0 && len(i.Changed) == 0 && len(i.Sources) == 0
}

// registryFile and configFile are the parts of the synced files that matter
// here.
type registryFile struct {
	AlwaysOn []string `json:"alwaysOn"`
	Projects map[string]struct {
		Name     string   `json:"name"`
		AlwaysOn []string `json:"alwaysOn"`
	} `json:"projects"`
}

type configFile struct {
	Sources map[string]struct {
		Repo string `json:"repo"`
	} `json:"sources"`
}

// review compares what the repository in home has now (HEAD) with the
// upstream commit it is about to take.
func review(home, upstream string) (Incoming, error) {
	var in Incoming
	ours, theirs, err := readRegistries(home, upstream)
	if err != nil {
		return in, err
	}
	// Always-on skills, in either version, whose files may change.
	watch := map[string]bool{}
	for _, id := range ours.AlwaysOn {
		watch[id] = true
	}
	for _, id := range theirs.AlwaysOn {
		watch[id] = true
		if !slices.Contains(ours.AlwaysOn, id) {
			in.AlwaysOn = append(in.AlwaysOn, id)
		}
	}
	for _, p := range ours.Projects {
		for _, id := range p.AlwaysOn {
			watch[id] = true
		}
	}
	for pid, p := range theirs.Projects {
		name := p.Name
		if name == "" {
			name = pid
		}
		for _, id := range p.AlwaysOn {
			watch[id] = true
			if o, ok := ours.Projects[pid]; !ok || !slices.Contains(o.AlwaysOn, id) {
				in.AlwaysOn = append(in.AlwaysOn, id+" (project "+name+")")
			}
		}
	}
	base := "HEAD"
	if b, err := git.MergeBase(home, "HEAD", upstream); err == nil && b != "" {
		base = b
	}
	files, err := git.DiffNames(home, base, upstream)
	if err != nil {
		return in, err
	}
	changed := map[string]bool{}
	for _, f := range files {
		if rest, ok := strings.CutPrefix(f, "local/"); ok {
			if id, _, found := strings.Cut(rest, "/"); found && watch[id] {
				changed[id] = true
			}
		}
	}
	for id := range changed {
		in.Changed = append(in.Changed, id)
	}
	if in.Sources, err = changedSources(home, upstream); err != nil {
		return in, err
	}
	sort.Strings(in.AlwaysOn)
	sort.Strings(in.Changed)
	return in, nil
}

func readRegistries(home, upstream string) (ours, theirs registryFile, err error) {
	for rev, into := range map[string]*registryFile{"HEAD": &ours, upstream: &theirs} {
		data, err := git.Show(home, rev, "registry.json")
		if err != nil {
			return ours, theirs, err
		}
		if len(data) == 0 {
			continue
		}
		if err := json.Unmarshal(data, into); err != nil {
			// A file this version cannot read cannot be told apart from
			// a change: show everything that is in it.
			*into = registryFile{}
		}
	}
	return ours, theirs, nil
}

func changedSources(home, upstream string) ([]string, error) {
	var ours, theirs configFile
	for rev, into := range map[string]*configFile{"HEAD": &ours, upstream: &theirs} {
		data, err := git.Show(home, rev, "config.json")
		if err != nil {
			return nil, err
		}
		if len(data) > 0 {
			_ = json.Unmarshal(data, into)
		}
	}
	var out []string
	for name, s := range theirs.Sources {
		if o, ok := ours.Sources[name]; !ok || o.Repo != s.Repo {
			out = append(out, name+" ("+s.Repo+")")
		}
	}
	sort.Strings(out)
	return out, nil
}
