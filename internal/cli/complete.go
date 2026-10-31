package cli

import (
	"os"
	"slices"
	"sort"
	"strings"

	"github.com/kheif/quiver/internal/config"
	"github.com/kheif/quiver/internal/library"
	"github.com/kheif/quiver/internal/project"
	"github.com/kheif/quiver/internal/registry"
	"github.com/spf13/cobra"
)

// Completion functions read the Quiver home without the lock and never
// write. Any error gives no suggestions instead of a message in the shell.

type completeFunc = func(cmd *cobra.Command, args []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective)

const noFiles = cobra.ShellCompDirectiveNoFileComp

// suggest keeps the candidates that start with toComplete and are not in
// args yet. Each candidate may carry a description after a tab.
func suggest(candidates []string, args []string, toComplete string) []cobra.Completion {
	var out []cobra.Completion
	for _, c := range candidates {
		value, _, _ := strings.Cut(c, "\t")
		if strings.HasPrefix(value, toComplete) && !slices.Contains(args, value) {
			out = append(out, c)
		}
	}
	return out
}

// skillIDs completes library skill IDs, with their descriptions. With
// localOnly, only local skills. max is the number of arguments the command
// takes; 0 means any.
func skillIDs(localOnly bool, max int) completeFunc {
	return func(_ *cobra.Command, args []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
		if max > 0 && len(args) >= max {
			return nil, noFiles
		}
		l, err := library.Load()
		if err != nil {
			return nil, noFiles
		}
		var ids []string
		for _, s := range l.All() {
			if !localOnly || s.Source == library.LocalSource {
				ids = append(ids, s.ID+"\t"+cut(s.Description, 60))
			}
		}
		return suggest(ids, args, toComplete), noFiles
	}
}

// sourceSkillIDs completes the IDs of skills from sources, for fork.
func sourceSkillIDs(_ *cobra.Command, args []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
	if len(args) >= 1 {
		return nil, noFiles
	}
	l, err := library.Load()
	if err != nil {
		return nil, noFiles
	}
	var ids []string
	for _, s := range l.All() {
		if s.Source != library.LocalSource {
			ids = append(ids, s.ID+"\t"+cut(s.Description, 60))
		}
	}
	return suggest(ids, args, toComplete), noFiles
}

// assignedIDs completes the skills that remove can take away: the global
// ones with --global, otherwise those of the project in --project or the
// current folder.
func assignedIDs(cmd *cobra.Command, args []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
	r, err := registry.Load()
	if err != nil {
		return nil, noFiles
	}
	if global, _ := cmd.Flags().GetBool("global"); global {
		return suggest(r.Global, args, toComplete), noFiles
	}
	dir, _ := cmd.Flags().GetString("project")
	if dir == "" {
		if dir, err = os.Getwd(); err != nil {
			return nil, noFiles
		}
	}
	p, err := project.Find(dir)
	if err != nil || r.Projects[p.ID] == nil {
		return nil, noFiles
	}
	return suggest(r.Projects[p.ID].Skills, args, toComplete), noFiles
}

// completeSources completes configured source names; withLocal adds "local".
// max is the number of arguments the command takes; 0 means any.
func completeSources(withLocal bool, max int) completeFunc {
	return func(_ *cobra.Command, args []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
		if max > 0 && len(args) >= max {
			return nil, noFiles
		}
		c, err := config.Load()
		if err != nil {
			return nil, noFiles
		}
		var names []string
		for name, s := range c.Sources {
			names = append(names, name+"\t"+s.Repo)
		}
		sort.Strings(names)
		if withLocal {
			names = append([]string{library.LocalSource + "\tyour own skills"}, names...)
		}
		return suggest(names, args, toComplete), noFiles
	}
}

// projectRefs completes the first argument with project names and IDs.
func projectRefs(_ *cobra.Command, args []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, noFiles
	}
	r, err := registry.Load()
	if err != nil {
		return nil, noFiles
	}
	var refs []string
	names := map[string]int{}
	for _, p := range r.Projects {
		names[p.Name]++
	}
	for id, p := range r.Projects {
		refs = append(refs, id+"\t"+p.Name)
		// A name shared by several projects is not a valid reference.
		if names[p.Name] == 1 && p.Name != id {
			refs = append(refs, p.Name+"\t"+id)
		}
	}
	sort.Strings(refs)
	return suggest(refs, args, toComplete), noFiles
}

// noArgs completes nothing and does not fall back to file names.
func noArgs(*cobra.Command, []string, string) ([]cobra.Completion, cobra.ShellCompDirective) {
	return nil, noFiles
}

// pathArg completes one file or folder name.
func pathArg(_ *cobra.Command, args []string, _ string) ([]cobra.Completion, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, noFiles
	}
	return nil, cobra.ShellCompDirectiveDefault
}

// projectFlag makes --project complete folders.
func projectFlag(cmd *cobra.Command) {
	_ = cmd.RegisterFlagCompletionFunc("project", func(*cobra.Command, []string, string) ([]cobra.Completion, cobra.ShellCompDirective) {
		return nil, cobra.ShellCompDirectiveFilterDirs
	})
}
