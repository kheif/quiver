package git

// Clone makes a shallow clone (--depth 1) of url into dir. "--" stops a url
// that starts with "-" from being read as an option.
func Clone(url, dir string) error {
	_, err := run("", "clone", "--quiet", "--depth", "1", "--", url, dir)
	return err
}

// Fetch downloads new commits from origin without changing the files.
func Fetch(dir string) error {
	_, err := run(dir, "fetch", "--quiet")
	return err
}

// Pull fast-forwards dir to origin. It fails instead of merging when the
// history has diverged.
func Pull(dir string) error {
	_, err := run(dir, "pull", "--quiet", "--ff-only")
	return err
}
