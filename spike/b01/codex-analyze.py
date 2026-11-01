#!/usr/bin/env python3
"""usage: codex-analyze.py <out dir>

Summarizes the sessions of codex-run.sh: for each prompt, the Quiver tool
calls in order, whether the always-on marker ends the first reply, and the
skill markers in the answer.
"""
import json
import pathlib
import re
import sys

out = pathlib.Path(sys.argv[1])
for f in sorted(out.glob("*.jsonl")):
    calls, msgs = [], []
    for line in f.read_text().splitlines():
        try:
            e = json.loads(line)
        except ValueError:
            continue
        item = e.get("item") or {}
        if e.get("type") != "item.completed":
            continue
        if item.get("type") == "mcp_tool_call" and item.get("server") == "quiver":
            args = item.get("arguments") or {}
            arg = args.get("id") or args.get("path") or ""
            status = "" if item.get("status") == "completed" else "!" + item.get("status", "")
            calls.append(f"{item['tool']}({arg}){status}")
        elif item.get("type") == "agent_message":
            msgs.append(item["text"])
    answer = "\n".join(msgs)
    markers = sorted(set(re.findall(r"\[used:[^\]]+\]", answer)))
    always = "yes" if msgs and msgs[-1].rstrip().endswith("[always-on]") else "no"
    print(f"{f.stem:10} always-on={always:3} markers={','.join(markers) or '-':30} calls={' '.join(calls) or '-'}")
