#!/usr/bin/env python3
"""usage: antigravity-analyze.py <out dir>

Summarizes the sessions of antigravity-run.sh: for each prompt, the Quiver
tool calls in order (and the reads of Antigravity's cached instructions),
whether the answer ends with the always-on marker, and the skill markers.
"""
import json
import pathlib
import re
import sys

out = pathlib.Path(sys.argv[1])
for f in sorted(out.glob("*.jsonl")):
    calls, answer = [], ""
    for line in f.read_text().splitlines():
        try:
            e = json.loads(line)
        except ValueError:
            continue
        if e.get("event") == "result":
            answer = e["result"].get("response", "")
            continue
        step = e.get("step_update") or {}
        if step.get("step_type") != "tool" or step.get("state") not in ("DONE", "ERROR"):
            continue
        info = step.get("tool_info") or {}
        params = info.get("parameters") or {}
        if info.get("name") == "call_mcp_tool" and params.get("ServerName") == "quiver":
            args = params.get("Arguments") or {}
            arg = args.get("id") or args.get("path") or ""
            status = "" if step["state"] == "DONE" else "!error"
            calls.append(f"{params['ToolName']}({arg}){status}")
        elif info.get("name") == "read_file" and "/mcp/quiver/" in params.get("Path", ""):
            calls.append("read:" + pathlib.Path(params["Path"]).name)
    markers = sorted(set(re.findall(r"\[used:[^\]]+\]", answer)))
    always = "yes" if answer.rstrip().endswith("[always-on]") else "no"
    print(f"{f.stem:10} always-on={always:3} markers={','.join(markers) or '-':30} calls={' '.join(calls) or '-'}")
