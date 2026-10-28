package cli

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/kheif/quiver/internal/fsutil"
	"github.com/kheif/quiver/internal/library"
	"github.com/kheif/quiver/internal/project"
	"github.com/kheif/quiver/internal/registry"
	"github.com/kheif/quiver/internal/skill"
	"github.com/spf13/cobra"
)

type importOptions struct {
	deep   bool
	yes    bool
	global bool
}

func newImportCmd() *cobra.Command {
	var o importOptions
	cmd := &cobra.Command{
		Use:   "import [<dir>]",
		Short: "Copy skills from a project into the library",
		Long: `Copy the skills in a project's .claude/skills into the library and assign
them, to the project or to every folder. With --deep, every folder with a
SKILL.md counts. The default is the project of the current folder.

For each skill it asks where to assign it, a new name if the name is taken,
and whether to delete the original folder. --global makes every folder the
default. With --yes it asks nothing: taken names are skipped and the
originals are kept.`,
		Example: `  # Import the .claude/skills of the project in this folder
  quiver import

  # Search every folder of another project, and ask nothing
  quiver import ~/code/api --deep --yes`,
		ValidArgsFunction: dirArg,
		Args:              cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := ""
			if len(args) == 1 {
				dir = args[0]
			}
			return importSkills(cmd.InOrStdin(), cmd.OutOrStdout(), o, dir)
		},
	}
	cmd.Flags().BoolVar(&o.deep, "deep", false, "also import every other folder with a SKILL.md")
	cmd.Flags().BoolVarP(&o.yes, "yes", "y", false, "ask nothing; keep the originals")
	cmd.Flags().BoolVar(&o.global, "global", false, "assign to every folder by default, instead of the project")
	return cmd
}

// importer holds what importSkills needs across skills.
type importer struct {
	in    *bufio.Reader
	out   io.Writer
	o     importOptions
	root  string           // folder searched
	proj  *project.Project // registered on the first project import
	from  string           // project name, for the commit message
	local string
}

func importSkills(in io.Reader, out io.Writer, o importOptions, dir string) error {
	if dir == "" {
		wd, err := os.Getwd()
		if err != nil {
			return err
		}
		dir = wd
		if p, err := project.Find(wd); err == nil {
			dir = p.Root
		}
	}
	root, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	if info, err := os.Stat(root); err != nil {
		return err
	} else if !info.IsDir() {
		return fmt.Errorf("%s is not a folder", root)
	}
	home, err := fsutil.QuiverHome()
	if err != nil {
		return err
	}
	found, err := library.FindSkills(root, o.deep, home)
	if err != nil {
		return err
	}
	if len(found) == 0 {
		fmt.Fprintf(out, "No skills found in %s.\n", root)
		if !o.deep {
			fmt.Fprintln(out, "Only .claude/skills is searched; use --deep to search every folder.")
		}
		return nil
	}
	local, err := library.LocalDir()
	if err != nil {
		return err
	}
	im := &importer{in: bufio.NewReader(in), out: out, o: o, root: root, from: filepath.Base(root), local: local}
	if p, err := project.Find(root); err == nil {
		if r, err := registry.Load(); err == nil && r.Projects[p.ID] != nil {
			im.from = r.Projects[p.ID].Name
		}
	}
	fmt.Fprintf(out, "Found %s in %s:\n", plural(len(found), "skill"), root)
	for _, f := range found {
		line := fmt.Sprintf("  %s  %s", f.Name, im.rel(f.Dir))
		if f.Err != nil {
			line += "  (skipped: " + f.Err.Error() + ")"
		}
		fmt.Fprintln(out, line)
	}
	var failed int
	for _, f := range found {
		if f.Err != nil {
			continue
		}
		fmt.Fprintln(out)
		if err := im.one(f); err != nil {
			fmt.Fprintf(out, "error: %s: %v\n", f.Name, err)
			failed++
		}
	}
	if failed > 0 {
		return fmt.Errorf("%s not imported", plural(failed, "skill"))
	}
	return nil
}

// one imports one found skill.
func (im *importer) one(f library.Found) error {
	fmt.Fprintf(im.out, "%s: %s\n", f.Name, cut(f.Description, 100))
	scope := library.ScopeProject
	if im.o.global {
		scope = library.ScopeGlobal
	}
	if !im.o.yes {
		choices := "[P/g/s]"
		if im.o.global {
			choices = "[p/G/s]"
		}
		switch im.ask("Assign it to this project, to every folder, or skip it? " + choices) {
		case "":
		case "p", "project":
			scope = library.ScopeProject
		case "g", "global":
			scope = library.ScopeGlobal
		default:
			fmt.Fprintln(im.out, "Skipped.")
			return nil
		}
	}
	name, ok, err := im.name(f)
	if err != nil || !ok {
		return err
	}
	projectID := ""
	if scope == library.ScopeProject {
		if im.proj == nil {
			p, created, copied, err := registry.RegisterProject(im.root)
			if err != nil {
				return err
			}
			reportProject(im.out, p, created, copied)
			im.proj = p
		}
		projectID = im.proj.ID
	}
	res, err := library.ImportLocal(f.Dir, name, scope, projectID, im.from)
	if err != nil {
		return err
	}
	where := "this project"
	if scope == library.ScopeGlobal {
		where = "every folder"
	}
	if res.Reused {
		fmt.Fprintf(im.out, "%s is already in the library with the same content; assigned it to %s.\n", name, where)
	} else {
		fmt.Fprintf(im.out, "Imported %s into the library and assigned it to %s.\n", name, where)
	}
	for _, w := range res.Warnings {
		fmt.Fprintln(im.out, "warning:", w)
	}
	if !im.o.yes && confirm(im.in, im.out, "Delete the original folder "+im.rel(f.Dir)+"?") {
		if err := os.RemoveAll(f.Dir); err != nil {
			return err
		}
		fmt.Fprintln(im.out, "Deleted.")
	}
	return nil
}

// name returns the library name for f. A taken or invalid name is asked
// for again; ok is false when the skill is skipped.
func (im *importer) name(f library.Found) (name string, ok bool, err error) {
	name = f.Name
	for {
		problem := ""
		if err := skill.ValidateName(name); err != nil {
			problem = err.Error()
		} else if _, err := os.Lstat(filepath.Join(im.local, name)); err == nil {
			// Same content under the same name is assigned, not copied.
			if name == f.Name {
				if same, err := library.SameAsLocal(f.Dir, name); err != nil {
					return "", false, err
				} else if same {
					return name, true, nil
				}
			}
			problem = fmt.Sprintf("the library already has a different skill named %s", name)
		} else if !errors.Is(err, os.ErrNotExist) {
			return "", false, err
		}
		if problem == "" {
			return name, true, nil
		}
		if im.o.yes {
			fmt.Fprintf(im.out, "Skipped: %s.\n", problem)
			return "", false, nil
		}
		fmt.Fprintf(im.out, "%s.\n", problem)
		if name = im.ask("New name (empty to skip):"); name == "" {
			fmt.Fprintln(im.out, "Skipped.")
			return "", false, nil
		}
	}
}

// ask prints question and returns the answer, trimmed and in lower case.
// No input gives "".
func (im *importer) ask(question string) string {
	fmt.Fprintf(im.out, "%s ", question)
	line, _ := im.in.ReadString('\n')
	return strings.ToLower(strings.TrimSpace(line))
}

func (im *importer) rel(dir string) string {
	if rel, err := filepath.Rel(im.root, dir); err == nil {
		return rel
	}
	return dir
}

// plural returns "1 skill" or "<n> skills".
func plural(n int, word string) string {
	if n == 1 {
		return "1 " + word
	}
	return fmt.Sprintf("%d %ss", n, word)
}
