#!/bin/sh
# usage: mkhome.sh <work> <home>: a throwaway Quiver home with the library as sources, every skill Searchable globally
set -e
W=$1; H=$2
export QUIVER_HOME=$H QUIVER_NO_UPDATE_CHECK=1 GIT_CONFIG_NOSYSTEM=1 GIT_AUTHOR_NAME=T GIT_AUTHOR_EMAIL=t@e.x GIT_COMMITTER_NAME=T GIT_COMMITTER_EMAIL=t@e.x
rm -rf "$H"
"$W/quiver" init --yes >/dev/null 2>&1 || true
for s in anthropics cathrynlavery codeswithroh juliusbrussee obra spike; do
  R="$W/repos/$s"; rm -rf "$R"; mkdir -p "$W/repos"; cp -R "$W/lib/$s" "$R"
  (cd "$R" && git init -q -b main && git add -A && git commit -q -m v1)
  "$W/quiver" source add "$R" --name "$s" --searchable </dev/null 2>&1 | tail -2
done
"$W/quiver" list --json | python3 -c "import json,sys; d=json.load(sys.stdin)['skills']; print(len(d),'skills;',sum(1 for s in d if s.get('searchable',{}).get('global')),'searchable global')"
