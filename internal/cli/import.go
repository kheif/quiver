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
		Use:   "import [<folder> | <skill>]",
		Short: "Copy skills into the library",
		Long: `Copy skills into the library and assign them to the project of the current
folder, or to every folder.

  quiver import             the .claude/skills of this project (--deep: every folder of it)
  quiver import <folder>    every skill in the folder and its subfolders
  quiver import <skill>     one skill: its SKILL.md, its folder or a markdown file

A skill is a folder with a SKILL.md, or a markdown file of any name whose
frontmatter has a description. Its name is the one in the frontmatter, or
the folder or file name; --name sets another one when importing one skill.
Markdown files without frontmatter, such as READMEs, are not skills. Skills
without a description are skipped and listed at the end.

For each skill it asks where to assign it, a new name if the name is taken,
and whether to delete the original. --global makes every folder the default.
With --yes it asks nothing: taken names are skipped and the originals are
kept.`,
		Example: `  # Import the .claude/skills of the project in this folder
  quiver import

  # Import every skill in a folder and its subfolders, and ask nothing
  quiver import ~/Downloads/skill-pack --yes

  # Import one skill under another name
  quiver import ~/Downloads/greeter/SKILL.md --name demo
  quiver import notes/review.md`,
		ValidArgsFunction: pathArg,
		Args:              cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := ""
			if len(args) == 1 {
				path = args[0]
			}
			return importSkills(cmd.InOrStdin(), cmd.OutOrStdout(), o, path)
		},
	}
	cmd.Flags().BoolVar(&o.deep, "deep", false, "without a folder: search every folder of this project, not only .claude/skills")
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
	root       string           // shown paths are relative to it
	projectDir string           // the folder of the project to assign to
	proj       *project.Project // registered on the first project import
	from       string           // where the skills come from, for the commit message
	local      string
}

func importSkills(in io.Reader, out io.Writer, o importOptions, path string) error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	projectDir := wd
	if p, err := project.Find(wd); err == nil {
		projectDir = p.Root
	}
	local, err := library.LocalDir()
	if err != nil {
		return err
	}
	im := &importer{in: bufio.NewReader(in), out: out, o: o, projectDir: projectDir, local: local}

	// Without a path: this project's skills.
	root, deep := projectDir, o.deep
	if path != "" {
		if root, err = filepath.Abs(path); err != nil {
			return err
		}
		info, err := os.Stat(root)
		if err != nil {
			return err
		}
		if one, err := oneSkill(root, info); err != nil || one != nil {
			if err != nil {
				return err
			}
			return im.importOne(*one)
		}
		if !info.IsDir() {
			return fmt.Errorf("%s is not a SKILL.md, a markdown file or a folder", root)
		}
		deep = true
	}
	if o.name != "" {
		return fmt.Errorf("--name works only when importing one skill: give its SKILL.md, its folder or its markdown file")
	}
	home, err := fsutil.QuiverHome()
	if err != nil {
		return err
	}
	found, err := library.FindSkills(root, deep, home)
	if err != nil {
		return err
	}
	im.root, im.from = root, filepath.Base(root)
	if p, err := project.Find(root); err == nil {
		if r, err := registry.Load(); err == nil && r.Projects[p.ID] != nil {
			im.from = r.Projects[p.ID].Name
		}
	}
	var skills, skipped []library.Found
	for _, f := range found {
		if f.Err != nil {
			skipped = append(skipped, f)
		} else {
			skills = append(skills, f)
		}
	}
	if len(skills) == 0 {
		fmt.Fprintf(out, "No skills found in %s.\n", root)
		if !deep {
			fmt.Fprintln(out, "Only .claude/skills is searched; use --deep to search every folder.")
		}
	} else {
		fmt.Fprintf(out, "Found %s in %s:\n", plural(len(skills), "skill"), root)
		for _, f := range skills {
			fmt.Fprintf(out, "  %s  %s\n", f.Name, im.rel(f.Path()))
		}
	}
	var imported, failed int
	for _, f := range skills {
		fmt.Fprintln(out)
		ok, err := im.one(f)
		switch {
		case err != nil:
			fmt.Fprintf(out, "error: %s: %v\n", f.Name, err)
			failed++
		case ok:
			imported++
		}
	}
	if len(skills) > 0 {
		fmt.Fprintf(out, "\nImported %s.\n", plural(imported, "skill"))
	}
	im.printSkipped(skipped)
	if failed > 0 {
		return fmt.Errorf("%s failed", plural(failed, "skill"))
	}
	return nil
}

// oneSkill returns path as one skill when it is a SKILL.md, a markdown
// file, or a folder with a SKILL.md; nil for a folder to search.
func oneSkill(path string, info os.FileInfo) (*library.Found, error) {
	var f library.Found
	switch {
	case info.IsDir():
		if _, err := os.Stat(filepath.Join(path, skill.FileName)); err != nil {
			return nil, nil
		}
		f = library.ReadFound(path, filepath.Base(path))
	case filepath.Base(path) == skill.FileName:
		dir := filepath.Dir(path)
		f = library.ReadFound(dir, filepath.Base(dir))
	case strings.EqualFold(filepath.Ext(path), ".md"):
		f = library.ReadFoundFile(path)
	default:
		return nil, nil
	}
	if f.Err != nil {
		return nil, fmt.Errorf("%w; a skill needs frontmatter with a description", f.Err)
	}
	return &f, nil
}

// importOne imports a skill the user named, under --name if given.
func (im *importer) importOne(f library.Found) error {
	if im.o.name != "" {
		f.Name = im.o.name
	}
	im.root = filepath.Dir(f.Path())
	im.from = filepath.Base(im.root)
	_, err := im.one(f)
	return err
}

// printSkipped lists what looked like a skill but is not usable.
func (im *importer) printSkipped(skipped []library.Found) {
	if len(skipped) == 0 {
		return
	}
	fmt.Fprintf(im.out, "\nSkipped %d, not valid skills:\n", len(skipped))
	for _, f := range skipped {
		// The error starts with the absolute path; show it relative.
		msg := f.Err.Error()
		if rel := im.rel(f.Path()); rel != f.Path() {
			msg = strings.ReplaceAll(msg, f.Path()+string(filepath.Separator), rel+string(filepath.Separator))
			msg = strings.ReplaceAll(msg, f.Path(), rel)
		}
		fmt.Fprintf(im.out, "  %s\n", msg)
	}
}

// one imports one found skill. ok is false when the user skipped it.
func (im *importer) one(f library.Found) (ok bool, err error) {
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
			return false, nil
		}
	}
	src, cleanup, err := f.Stage()
	if err != nil {
		return false, err
	}
	defer cleanup()
	name, ok, err := im.name(f, src)
	if err != nil || !ok {
		return false, err
	}
	projectID := ""
	if scope == library.ScopeProject {
		if im.proj == nil {
			p, created, copied, err := registry.RegisterProject(im.projectDir)
			if err != nil {
				return false, err
			}
			reportProject(im.out, p, created, copied)
			im.proj = p
		}
		projectID = im.proj.ID
	}
	res, err := library.ImportLocal(src, name, scope, projectID, im.from)
	if err != nil {
		return false, err
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
	what := "folder"
	if f.File != "" {
		what = "file"
	}
	if !im.o.yes && confirm(im.in, im.out, "Delete the original "+what+" "+im.rel(f.Path())+"?") {
		if err := os.RemoveAll(f.Path()); err != nil {
			return true, err
		}
		fmt.Fprintln(im.out, "Deleted.")
	}
	return true, nil
}

// name returns the library name for f, whose files are in src. A taken or
// invalid name is asked for again; ok is false when the skill is skipped.
func (im *importer) name(f library.Found, src string) (name string, ok bool, err error) {
	name = f.Name
	for {
		problem := ""
		if err := skill.ValidateName(name); err != nil {
			problem = err.Error()
		} else if _, err := os.Lstat(filepath.Join(im.local, name)); err == nil {
			// Same content under the same name is assigned, not copied.
			if name == f.Name {
				if same, err := library.SameAsLocal(src, name); err != nil {
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

func (im *importer) rel(path string) string {
	if rel, err := filepath.Rel(im.root, path); err == nil && !strings.HasPrefix(rel, "..") {
		return rel
	}
	return path
}

// plural returns "1 skill" or "<n> skills".
func plural(n int, word string) string {
	if n == 1 {
		return "1 " + word
	}
	return fmt.Sprintf("%d %ss", n, word)
}
