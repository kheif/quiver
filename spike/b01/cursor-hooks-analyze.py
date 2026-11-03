#!/usr/bin/env python3
"""usage: cursor-hooks-analyze.py <dir>

Summarizes cursor-hooks-run.sh: for each workspace, the hook events that
fired (from hook.log) and, per session, the Quiver tools called, the seconds
it took and whether the answer ends with the always-on marker.
"""
import json
import pathlib
import sys

root = pathlib.Path(sys.argv[1])
for w in sorted(p for p in root.iterdir() if (p / "out").is_dir()):
    log = w / "hook.log"
    fired = [json.loads(l).get("hook_event_name") for l in log.read_text().splitlines() if l.strip()] if log.exists() else []
    secs = dict(l.split() for l in (w / "seconds.log").read_text().splitlines()) if (w / "seconds.log").exists() else {}
    always = fitting = other = pointer = 0
    rows = []
    for f in sorted((w / "out").glob("*.jsonl")):
        calls, answer = [], ""
        for line in f.read_text().splitlines():
            try:
                msg = json.loads(line)
            except json.JSONDecodeError:
                continue
            call = msg.get("tool_call", {}).get("mcpToolCall", {})
            if "result" in call:
                calls.append(f"{call['args']['toolName']}({call['args']['args'].get('id', '')})")
            if msg.get("type") == "result":
                answer = msg.get("result", "")
        always += "[always-on]" in answer
        pointer += any("reply-style" in c for c in calls)
        if f.stem.startswith("fit"):
            fitting += "[used:" in answer
        elif f.stem.startswith("none"):
            other += any("get_skill(" in c and "reply-style" not in c for c in calls)
        rows.append(f"{'':16} {f.stem:8} {secs.get(f.stem, '?'):>3}s marker={'[always-on]' in answer} calls={calls} {answer[-50:]!r}")
    print(f"{w.name:16} fired={fired}")
    if w.name == "prompts":
        print(f"{'':16} always-on {always}/10  fitting {fitting}/6  other {other}/4  get_skill(reply-style) {pointer}/10")
    else:
        print("\n".join(rows))
