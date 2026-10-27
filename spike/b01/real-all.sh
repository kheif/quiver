#!/bin/bash
# usage: real-all.sh   (make spike)
# Repeats the B01 checks against the real `quiver mcp`: the ten prompts in
# prompts.txt, a five-turn session, and a session compacted after turn two.
# Every session is a real `claude -p` run, so this costs tokens.
set -e
SPIKE=$(cd "$(dirname "$0")" && pwd)
export QUIVER=${QUIVER:-$SPIKE/../../bin/quiver}
export OUT=${OUT:-$(mktemp -d)}
rm -rf "$OUT/real"

sid() {
  python3 -c '
import json, sys
for l in open(sys.argv[1]):
    try: m = json.loads(l)
    except Exception: continue
    if m.get("type") == "result": print(m["session_id"])' "$1"
}

# turns <id> <prompt>... runs the prompts as one resumed session.
turns() {
  local id=$1 s="" f
  shift
  for p in "$@"; do
    RESUME=$s "$SPIKE/run-real.sh" "$id" "$p"
    f=$(ls "$OUT/real/$id"/out*.jsonl | sort | tail -1)
    s=$(sid "$f")
  done
}

while IFS='|' read -r id p; do
  "$SPIKE/run-real.sh" "$id" "$p" &
done < "$SPIKE/prompts.txt"
turns multi "Explain what a CRDT is in two sentences." "Give one example of a CRDT." \
  "What is a vector clock? Two sentences." "Suggest three names for a cat." "Write a haiku about Go." &
turns compact "Explain what a CRDT is in two sentences." "Give one example of a CRDT." "/compact" \
  "What is a vector clock? Two sentences." "Suggest three names for a cat." &
wait

python3 "$SPIKE/analyze.py" "$OUT"
echo "sessions in $OUT/real"
