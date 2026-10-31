#!/bin/bash
# usage: catalog-all.sh   (make spike-catalog)
# Checks the list_skills pointer (TASK36): with 40 extra skills the catalog
# does not fit in the instructions. Runs the six fitting prompts plus four
# of them again, two non-fitting ones, and a session compacted after turn
# two that ends with a fitting prompt. Every session is a real `claude -p`
# run, so this costs tokens.
set -e
SPIKE=$(cd "$(dirname "$0")" && pwd)
export QUIVER=${QUIVER:-$SPIKE/../../bin/quiver}
export OUT=${OUT:-$(mktemp -d)}
export FILLER=40
rm -rf "$OUT/real"

sid() {
  python3 -c '
import json, sys
for l in open(sys.argv[1]):
    try: m = json.loads(l)
    except Exception: continue
    if m.get("type") == "result": print(m["session_id"])' "$1"
}

turns() {
  local id=$1 s="" f
  shift
  for p in "$@"; do
    RESUME=$s "$SPIKE/run-real.sh" "$id" "$p"
    f=$(ls "$OUT/real/$id"/out*.jsonl | sort | tail -1)
    s=$(sid "$f")
  done
}

{ grep '^fit-' "$SPIKE/prompts.txt"; grep '^fit-' "$SPIKE/prompts.txt" | head -4 | sed 's/^fit-/again-/'; grep '^none-[12]' "$SPIKE/prompts.txt"; } |
while IFS='|' read -r id p; do
  "$SPIKE/run-real.sh" "$id" "$p" &
done
turns compact "Explain what a CRDT is in two sentences." "Give one example of a CRDT." "/compact" \
  "My pod api-7f9 is stuck in CrashLoopBackOff. How do I find out why?" &
wait

python3 "$SPIKE/analyze.py" "$OUT"
echo "sessions in $OUT/real"
