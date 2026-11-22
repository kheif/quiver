#!/bin/sh
# Makes the golden Quiver home of one release: runs that release's binary
# through a fixed script of commands and copies the files that sync carries
# into <out>. Usage: make-home.sh <quiver-binary> <out-dir>
#
# Left out of the copy: sources/ (clones, which hold a .git), logs/ and
# state/ (machine-local), and local/.git (a repository cannot be committed
# inside this one; the test makes a fresh one in its copy of the home).
set -eu

bin=$1
out=$2
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT

export HOME="$work/home" QUIVER_HOME="$work/home/.quiver" QUIVER_PROJECT=
export GIT_AUTHOR_NAME=quiver GIT_AUTHOR_EMAIL=quiver@example.invalid
export GIT_COMMITTER_NAME=quiver GIT_COMMITTER_EMAIL=quiver@example.invalid
mkdir -p "$HOME"

# A source with two skills, as a local repository.
src="$work/acme"
mkdir -p "$src/one" "$src/two"
printf -- '---\nname: one\ndescription: First skill of the source.\n---\n' >"$src/one/SKILL.md"
printf -- '---\nname: two\ndescription: Second skill of the source.\n---\n' >"$src/two/SKILL.md"
git -C "$src" init --quiet
git -C "$src" add .
git -C "$src" commit --quiet -m skills

proj="$work/app"
mkdir -p "$proj"

"$bin" init --yes
"$bin" source add "$src" --name acme
"$bin" new notes --global --description "Take notes."
"$bin" add acme/one --global
"$bin" add notes --global --always
(
	cd "$proj"
	"$bin" new deploy --description "Deploy the app." --yes
	"$bin" add acme/two --yes
	"$bin" add deploy --always --yes
)

rm -rf "$out"
mkdir -p "$out"
cp -R "$QUIVER_HOME/." "$out/"
rm -rf "$out/sources" "$out/logs" "$out/state" "$out/local/.git" "$out/.lock"
# The source lives in a temporary folder; give it a stable address.
sed "s|$src|https://example.invalid/acme.git|" "$out/config.json" >"$out/config.json.new"
mv "$out/config.json.new" "$out/config.json"
