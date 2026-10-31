"""Summarizes cursor-run.sh output: the Quiver tools each session called and
the markers its answer ends with. usage: cursor-analyze.py <dir>/out"""
import json
import pathlib
import sys

for path in sorted(pathlib.Path(sys.argv[1]).glob("*.jsonl")):
    calls, answer = [], ""
    for line in path.read_text().splitlines():
        try:
            msg = json.loads(line)
        except json.JSONDecodeError:
            continue
        call = msg.get("tool_call", {}).get("mcpToolCall", {})
        if "result" in call:
            args = call["args"]
            ok = "success" in call["result"]
            arg = args["args"].get("id", "") + (":" + args["args"]["path"] if "path" in args["args"] else "")
            calls.append(f"{args['toolName']}({arg}){'' if ok else '!'}")
        if msg.get("type") == "result":
            answer = msg.get("result", "")
    markers = [w for w in ("[always-on]", "[used:") if w in answer]
    used = [l.strip() for l in answer.splitlines() if l.strip().startswith("[used:")]
    print(f"{path.stem:10} calls={' '.join(calls) or '-':70} used={','.join(used) or '-':28} always-on={'[always-on]' in answer}")
