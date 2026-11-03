#!/usr/bin/env python3
"""usage: antigravity-hooks-analyze.py <dir>

Summarizes antigravity-hooks-run.sh. For the runs of the 10 prompts:
always-on, fitting and other skills, and get_skill calls for reply-style.
For the other runs, per session: seconds, the Quiver tools and sub-agents
called, the rule markers in the answer, and for long hook texts the tokens
the session's transcript kept.
"""
import json
import pathlib
import re
import sys

root = pathlib.Path(sys.argv[1])


def session(path):
    calls, answer, conversation = [], "", ""
    for line in path.read_text().splitlines():
        try:
            e = json.loads(line)
        except ValueError:
            continue
        if e.get("event") == "init":
            conversation = e.get("conversation_id", "")
        if e.get("event") == "result":
            answer = e["result"].get("response", "")
        step = e.get("step_update") or {}
        if step.get("step_type") == "tool" and step.get("state") == "DONE":
            info = step.get("tool_info") or {}
            params = info.get("parameters") or {}
            if info.get("name") == "call_mcp_tool" and params.get("ServerName") == "quiver":
                calls.append(f"{params['ToolName']}({(params.get('Arguments') or {}).get('id', '')})")
            elif "subagent" in info.get("name", ""):
                calls.append(info["name"])
    return conversation, calls, answer


def markers(answer):
    return [m for m in ("[always-on]", "[global-rule]", "[project-rule]") if m in answer]


for w in sorted(p for p in root.iterdir() if (p / "out").is_dir()):
    log = w / "hook.log"
    firings = [json.loads(l) for l in log.read_text().splitlines() if l.strip()] if log.exists() else []
    secs = dict(l.split() for l in (w / "seconds.log").read_text().splitlines()) if (w / "seconds.log").exists() else {}
    files = sorted((w / "out").glob("*.jsonl"))
    if len(files) == 10:
        always = fitting = other = pointer = 0
        for f in files:
            _, calls, answer = session(f)
            always += "[always-on]" in answer
            pointer += "get_skill(reply-style)" in calls
            if f.stem.startswith("fit"):
                fitting += "[used:" in answer
            else:
                other += any(c.startswith("get_skill(") and "reply-style" not in c for c in calls)
        print(f"{w.name:20} always-on {always}/10  fitting {fitting}/6  other {other}/4  "
              f"get_skill(reply-style) {pointer}/10  firings {len(firings)}")
        continue
    print(f"{w.name:20} firings={len(firings)}")
    transcripts = {e.get("conversationId"): e.get("transcriptPath") for e in firings}
    for f in files:
        conversation, calls, answer = session(f)
        kept = ""
        t = transcripts.get(conversation)
        if t and pathlib.Path(t).exists():
            kept = sorted(set(re.findall(r"(?:START|MID|END)-TOKEN-\w+", pathlib.Path(t).read_text(errors="replace"))))
        print(f"{'':20} {f.stem:6} {secs.get(f.stem, '?'):>3}s markers={markers(answer)} calls={calls} "
              f"{'kept=' + str(kept) + ' ' if kept else ''}{answer[-60:]!r}")
