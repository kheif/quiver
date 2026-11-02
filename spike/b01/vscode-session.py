#!/usr/bin/env python3
"""usage: vscode-session.py <chatSessions/*.jsonl>...
       vscode-session.py --count <chatSessions dir>
       vscode-session.py --done <chatSessions dir> <n>

Reads VS Code chat session files. With files, prints each session's prompt,
tool calls and answer. --count prints how many sessions have an answered
request; --done prints the newest such session and exits 0 once there are
more than <n>. A session file is a snapshot (kind 0) followed by patches:
kind 1 sets the value at path k, kind 2 appends to the list at path k.
"""
import json
import pathlib
import sys


def load(path):
    state = None
    for line in open(path):
        e = json.loads(line)
        if e["kind"] == 0:
            state = e["v"]
            continue
        target = state
        for key in e["k"][:-1]:
            target = target[key]
        last = e["k"][-1]
        if e["kind"] == 1:
            target[last] = e["v"]
        elif e["kind"] == 2:
            if e.get("i") is not None:
                del target[last][e["i"]:]
            target.setdefault(last, []).extend(e["v"])
    return state


def requests(path):
    for r in load(path).get("requests", []):
        meta = (r.get("result") or {}).get("metadata") or {}
        rounds = meta.get("toolCallRounds") or []
        calls = [f"{c['name']}({json.loads(c.get('arguments') or '{}').get('id', '')})" for rd in rounds for c in rd.get("toolCalls", [])]
        answer = rounds[-1]["response"] if rounds else ""
        context = json.dumps(meta.get("renderedGlobalContext")) + json.dumps(meta.get("renderedUserMessage"))
        elapsed = ((r.get("result") or {}).get("timings") or {}).get("totalElapsed")
        subagents = [p["toolSpecificData"].get("result") or "" for p in r.get("response", [])
                     if (p.get("toolSpecificData") or {}).get("kind") == "subagent"]
        yield {"prompt": r["message"]["text"], "calls": calls, "answer": answer, "context": context,
               "model": meta.get("resolvedModel"), "done": bool(r.get("result")), "elapsed": elapsed,
               "subagents": subagents}


def answered(folder):
    """Session files with an answered request, oldest first."""
    files = sorted(pathlib.Path(folder).glob("*.jsonl"), key=lambda p: p.stat().st_mtime)
    return [f for f in files if any(r["done"] for r in requests(f))]


if __name__ == "__main__" and sys.argv[1] == "--count":
    print(len(answered(sys.argv[2])) if pathlib.Path(sys.argv[2]).is_dir() else 0)
elif __name__ == "__main__" and sys.argv[1] == "--done":
    done = answered(sys.argv[2]) if pathlib.Path(sys.argv[2]).is_dir() else []
    if len(done) <= int(sys.argv[3]):
        sys.exit(1)
    print(done[-1].name)
elif __name__ == "__main__":
    for path in sys.argv[1:]:
        for r in requests(path):
            print(f"{r['prompt'][:40]!r} model={r['model']} calls={r['calls']} answer={r['answer'][-80:]!r}")
