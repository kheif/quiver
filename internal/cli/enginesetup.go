package cli

import (
	"bufio"
	"cmp"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/kheif/quiver/internal/library"
	"github.com/kheif/quiver/internal/registry"
	"github.com/spf13/cobra"
)

// setupCommand is the command that sets the local engine up, printed where
// the setup cannot ask.
const setupCommand = "quiver engine setup"

func newEngineSetupCmd() *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Download the model of the local engine and switch to it",
		Long: `Set up search by meaning: download the model file of the local engine
(all-MiniLM-L6-v2, English, 91 MB) over HTTPS from one pinned address, check
its SHA-256, build the index of the library, check that the vector search
finds a skill of yours, and make local the engine of this machine.

It asks once before it changes anything. The only thing that leaves the
machine is the download of the model file; a search makes no network call.
Quiver installs and starts no program: the model runs inside Quiver.

A model file that you put in cache/models/all-minilm-l6-v2/ in the Quiver
home is accepted when its SHA-256 matches, and is not downloaded again.
Without a terminal and without --yes nothing changes.`,
		Example: `  quiver engine setup
  quiver engine setup --yes`,
		ValidArgsFunction: noArgs,
		Args:              cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runEngineSetup(cmd, yes)
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "answer the question with yes")
	return cmd
}

// checkVectorPath is the check of the setup. Tests replace it to break the
// vector path.
var checkVectorPath = (*library.Library).CheckVectorPath

func megabytes(n int64) string { return fmt.Sprintf("%d MB", (n+1<<20-1)>>20) }

// setupQuestion says what the setup will do, as the question asks for it.
func setupQuestion(out io.Writer, spec library.ModelSpec) {
	var size int64
	for _, f := range spec.Files() {
		size += f.Size
	}
	fmt.Fprintf(out, `Search by meaning needs a model file, which Quiver downloads once:
  model   %s (English only)
  size    %s to download, and about 1 MB for the index
  memory  a search uses up to about 85 MB while the model is loaded, mostly the
          mapped file, which the system shares between processes
  network the download of the model file is the only thing that leaves this
          machine; a search itself makes no network call
`, spec.Name, megabytes(size))
}

func runEngineSetup(cmd *cobra.Command, yes bool) error {
	out := cmd.OutOrStdout()
	if !yes && !interactive(cmd.InOrStdin()) {
		fmt.Fprintf(out, "Nothing changed: this needs your answer. Run %s in a terminal, or add --yes.\n", setupCommand)
		return nil
	}
	return engineSetup(cmd.Context(), bufio.NewReader(cmd.InOrStdin()), out, cmd.ErrOrStderr(), yes)
}

// engineSetup asks the question unless yes, and runs the setup. The caller
// has checked that a person can answer.
func engineSetup(ctx context.Context, in *bufio.Reader, out, errOut io.Writer, yes bool) error {
	name := cmp.Or(library.LoadEngine().Local.Model, library.DefaultModel)
	spec, ok := library.KnownModel(name)
	if !ok {
		return fmt.Errorf("unknown model %q", name)
	}
	setupQuestion(out, spec)
	if !yes && !confirmLine(in, out, "Download the model and switch to search by meaning?") {
		fmt.Fprintln(out, "Nothing changed.")
		return nil
	}
	l, err := library.Load()
	if err != nil {
		return err
	}

	res, derr := spec.EnsureModelFiles(ctx, downloadProgress(errOut))
	// Record what was downloaded even if a later step fails, so that remove
	// deletes it.
	prior := library.Installed(name)
	installed := map[string]string{}
	for file, hash := range res.Downloaded {
		installed[file] = hash
	}
	for _, file := range res.Present {
		if h, ok := prior[file]; ok {
			installed[file] = h
		}
	}
	if err := library.RecordInstalled(name, installed); err != nil {
		return err
	}
	if derr != nil {
		return fmt.Errorf("step 1, the model file: %w; the engine stays words", derr)
	}
	if len(res.Present) > 0 && len(res.Downloaded) == 0 {
		fmt.Fprintf(out, "The model file is already there with the pinned SHA-256; nothing downloaded.\n")
	}

	if _, err := l.Reindex(ctx, name, indexProgress(errOut)); err != nil {
		return fmt.Errorf("step 2, the index: %w; the engine stays words", err)
	}
	note, err := checkVectorPath(l, name)
	if err != nil {
		return fmt.Errorf("step 3, the check: %w; the engine stays words", err)
	}
	if err := library.UseLocal(name); err != nil {
		return err
	}
	fmt.Fprintf(out, "\nSearch by meaning is on. Check: %s.\n", note)
	dir, _ := library.ModelDir(name)
	fmt.Fprintf(out, "  model file  %s (%s)\n", dir, megabytes(spec.Size))
	fmt.Fprintln(out, "  index       cache/embeddings.json in the Quiver home")
	fmt.Fprintln(out, "  undo        quiver engine use words keeps everything; quiver engine remove deletes the index and the downloaded model")
	return nil
}

// downloadProgress prints one line per file when out is not a terminal, and a
// line that updates when it is.
func downloadProgress(out io.Writer) library.DownloadProgress {
	_, tty := terminal(out)
	last := map[string]int{}
	return func(file string, done, total int64) {
		pct := int(done * 100 / max(total, 1))
		if pct == last[file] && done < total {
			return
		}
		last[file] = pct
		switch {
		case tty && done >= total:
			fmt.Fprintf(out, "\rDownloading %s: %d%%\n", file, pct)
		case tty:
			fmt.Fprintf(out, "\rDownloading %s: %d%%", file, pct)
		case done >= total || pct%25 == 0:
			fmt.Fprintf(out, "Downloading %s: %d%%\n", file, pct)
		}
	}
}

func newEngineRemoveCmd() *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   "remove",
		Short: "Delete the search index and the model Quiver downloaded",
		Long: `Delete the index of the local engine, cache/embeddings.json, and, after
asking, the model files that quiver engine setup downloaded. A file that you
put there by hand is never deleted. The engine goes back to words.`,
		Example:           `  quiver engine remove`,
		ValidArgsFunction: noArgs,
		Args:              cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			in, out := bufio.NewReader(cmd.InOrStdin()), cmd.OutOrStdout()
			name := cmp.Or(library.LoadEngine().Local.Model, library.DefaultModel)
			installed := library.Installed(name)
			if err := library.RemoveIndex(); err != nil {
				return err
			}
			fmt.Fprintln(out, "Deleted the search index.")
			if library.LoadEngine().Engine == library.EngineLocal {
				if err := library.UseEngine(library.EngineWords); err != nil {
					return err
				}
				fmt.Fprintln(out, "Search engine: words")
			}
			if len(installed) == 0 {
				fmt.Fprintln(out, "Quiver downloaded no model file, so none is deleted.")
				return nil
			}
			if !yes {
				if !interactive(cmd.InOrStdin()) {
					fmt.Fprintf(out, "The model file stays: add --yes to delete what %s downloaded.\n", setupCommand)
					return nil
				}
				if !confirmLine(in, out, "Delete the model file that Quiver downloaded?") {
					fmt.Fprintln(out, "The model file stays.")
					return nil
				}
			}
			removed, err := library.RemoveDownloaded(name, installed)
			if err != nil {
				return err
			}
			if err := library.ForgetInstalled(name); err != nil {
				return err
			}
			fmt.Fprintf(out, "Deleted %s.\n", strings.Join(removed, ", "))
			return nil
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "delete the downloaded model file without asking")
	return cmd
}

// offerEngineSetup offers the setup once, after the first Searchable
// assignment a person makes in a terminal while the engine is words. The offer
// is recorded before it is made, so it is never repeated, and the default is
// no. Without a terminal it does nothing and records nothing.
func offerEngineSetup(ctx context.Context, in io.Reader, out, errOut io.Writer) error {
	cur := library.LoadEngine()
	if cur.Engine != library.EngineWords || cur.Offered || cur.Refused || !interactive(in) {
		return nil
	}
	spec, ok := library.KnownModel(cmp.Or(cur.Local.Model, library.DefaultModel))
	if !ok {
		return nil
	}
	if err := library.MarkOffered(); err != nil {
		return nil // a settings file that cannot be written: do not ask
	}
	fmt.Fprintln(out)
	setupQuestion(out, spec)
	r := bufio.NewReader(in)
	if !confirmLine(r, out, "Set up search by meaning now? (You can do it later with "+setupCommand+".)") {
		fmt.Fprintf(out, "Not now. Searching keeps using words; run %s to change that.\n", setupCommand)
		return nil
	}
	return engineSetup(ctx, r, out, errOut, true)
}

// searchableCount is the number of Searchable assignments in the registry.
func searchableCount() int {
	r, err := registry.Load()
	if err != nil {
		return 0
	}
	n := len(r.Searchable)
	for _, p := range r.Projects {
		n += len(p.Searchable)
	}
	return n
}

// offerAfterSearchable runs a command that may make skills Searchable, as new
// and import do, and offers the setup once when it did.
func offerAfterSearchable(cmd *cobra.Command, run func() error) error {
	before := searchableCount()
	if err := run(); err != nil {
		return err
	}
	if searchableCount() > before {
		return offerEngineSetup(cmd.Context(), cmd.InOrStdin(), cmd.OutOrStdout(), cmd.ErrOrStderr())
	}
	return nil
}
