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
	name   string // only for one skill
}

func newImportCmd() *cobra.Command {
	var o importOptions
	cmd := &cobra.Command{
		Use:   "import [<dir> | <skill>]",
		Short: "Copy skills from a project, or one skill, into the library",
		Long: `Copy the skills in a project's .claude/skills into the library and assign
them, to the project or to every folder. With --deep, every folder with a
SKILL.md counts. The default is the project of the current folder.

To import one skill, give its SKILL.md, its folder, or any markdown file
with name and description frontmatter. --name sets its name in the library.
It is assigned to the project of the current folder.

For each skill it asks where to assign it, a new name if the name is taken,
and whether to delete the original folder. --global makes every folder the
default. With --yes it asks nothing: taken names are skipped and the
originals are kept.`,
		Example: `  # Import the .claude/skills of the project in this folder
  quiver import

  # Search every folder of another project, and ask nothing
  quiver import ~/code/api --deep --yes

  # Import one skill under another name
  quiver import ~/Downloads/greeter/SKILL.md --name demo`,
		ValidArgsFunction: pathArg,
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
	cmd.Flags().StringVar(&o.name, "name", "", "name in the library, when importing one skill")
	return cmd
}

// importer holds what importSkills needs across skills.
type importer struct {
	in         *bufio.Reader
	out        io.Writer
	o          importOptions
	root       string           // folder searched, or the folder of one skill
	projectDir string           // the folder of the project to assign to
	proj       *project.Project // registered on the first project import
	from       string           // project name, for the commit message
	local      string
	original   string // one skill: the file or folder the user gave
}

func importSkills(in io.Reader, out io.Writer, o importOptions, dir string) error {
	if dir != "" {
		done, err := importOne(in, out, o, dir)
		if done || err != nil {
			return err
		}
	}
	if o.name != "" {
		return fmt.Errorf("--name works only when importing one skill: give its SKILL.md or its folder")
	}
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
	im := &importer{in: bufio.NewReader(in), out: out, o: o, root: root, projectDir: root, from: filepath.Base(root), local: local}
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
			p, created, copied, err := registry.RegisterProject(im.projectDir)
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
	original, what := f.Dir, "folder"
	if im.original != "" {
		original = im.original
		if info, err := os.Stat(original); err == nil && !info.IsDir() {
			what = "file"
		}
	}
	if !im.o.yes && confirm(im.in, im.out, "Delete the original "+what+" "+im.rel(original)+"?") {
		if err := os.RemoveAll(original); err != nil {
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

// importOne imports path when it is one skill: a SKILL.md, another markdown
// file, or a folder with a SKILL.md. done is false for a folder to search.
// The skill is assigned to the project of the current folder.
func importOne(in io.Reader, out io.Writer, o importOptions, path string) (done bool, err error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return false, err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return false, err
	}
	dir, fallback := abs, filepath.Base(abs)
	switch {
	case info.IsDir():
		if _, err := os.Stat(filepath.Join(abs, skill.FileName)); err != nil {
			return false, nil
		}
	case filepath.Base(abs) == skill.FileName:
		dir = filepath.Dir(abs)
		fallback = filepath.Base(dir)
	case strings.EqualFold(filepath.Ext(abs), ".md"):
		// A lone markdown file: import a copy of it as SKILL.md, without
		// the other files of its folder.
		tmp, err := os.MkdirTemp("", "quiver-import-")
		if err != nil {
			return false, err
		}
		defer func() { _ = os.RemoveAll(tmp) }()
		data, err := os.ReadFile(abs)
		if err != nil {
			return false, err
		}
		if err := os.WriteFile(filepath.Join(tmp, skill.FileName), data, 0o644); err != nil {
			return false, err
		}
		dir = tmp
		fallback = strings.TrimSuffix(filepath.Base(abs), filepath.Ext(abs))
	default:
		return false, fmt.Errorf("%s is not a SKILL.md, a markdown file or a folder", abs)
	}
	f := library.ReadFound(dir, fallback)
	if f.Err != nil {
		return true, fmt.Errorf("%w; a skill needs frontmatter with name and description", f.Err)
	}
	if o.name != "" {
		f.Name = o.name
	}
	wd, err := os.Getwd()
	if err != nil {
		return true, err
	}
	projectDir := wd
	if p, err := project.Find(wd); err == nil {
		projectDir = p.Root
	}
	local, err := library.LocalDir()
	if err != nil {
		return true, err
	}
	// The original is the skill folder, or the lone markdown file.
	original := dir
	if dir != abs && filepath.Base(abs) != skill.FileName {
		original = abs
	}
	parent := filepath.Dir(original)
	im := &importer{
		in: bufio.NewReader(in), out: out, o: o,
		root: parent, projectDir: projectDir,
		from: filepath.Base(parent), local: local, original: original,
	}
	return true, im.one(f)
}
