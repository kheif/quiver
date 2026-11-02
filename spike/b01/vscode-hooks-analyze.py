#!/usr/bin/env python3
"""usage: vscode-hooks-analyze.py <dir> [VSC folder, default /tmp/qvsc]

Summarizes vscode-hooks-run.sh. For the runs of the 10 prompts: always-on,
fitting and other skills, and get_skill calls for reply-style. For single
runs: hook firings, whether the hook's text reached the model (the context VS
Code saves with each request) and which tokens of a long text it kept, the
request's time, tool calls and the answer. For user-mcp: the folder the server
started in, from Quiver's mcp.log.
"""
import glob
import importlib.util
import json
import pathlib
import re
import sys

root = pathlib.Path(sys.argv[1])
vsc = pathlib.Path(sys.argv[2] if len(sys.argv) > 2 else "/tmp/qvsc")
spec = importlib.util.spec_from_file_location("session", pathlib.Path(__file__).with_name("vscode-session.py"))
session = importlib.util.module_from_spec(spec)
spec.loader.exec_module(session)
RULE = "House reply style"


def storage(w):
    for f in glob.glob(str(vsc / "data/User/workspaceStorage/*/workspace.json")):
        if json.load(open(f)).get("folder") == "file://" + str(w):
            return pathlib.Path(f).parent


IDS = {p.strip(): i for i, p in (l.split("|", 1) for l in open(pathlib.Path(__file__).with_name("prompts.txt")))}


def asked(w):
    """(id, request) for every request in the workspace's chats, matched to the
    fixture prompts by their text; unanswered ones too, with done False."""
    s = storage(w)
    if not s:
        return
    for f in sorted((s / "chatSessions").glob("*.jsonl"), key=lambda p: p.stat().st_mtime):
        for r in session.requests(f):
            yield IDS.get(r["prompt"].strip(), "cat" if r["prompt"].startswith("Suggest") else "sub"), r


def firings(w):
    log = w / "hook.log"
    return [json.loads(l) for l in log.read_text().splitlines() if l.strip()] if log.exists() else []


def prompts(w):
    always = fitting = other = pointer = 0
    stuck = 0
    for id_, r in asked(w):
        if not r["done"]:
            stuck += 1
            continue
        always += "[always-on]" in r["answer"]
        pointer += any("get_skill(reply-style)" in c for c in r["calls"])
        if id_.startswith("fit"):
            fitting += "[used:" in r["answer"]
        else:
            other += any("get_skill(" in c and "reply-style" not in c for c in r["calls"])
    print(f"{w.name:18} always-on {always}/10  fitting {fitting}/6  other {other}/4  "
          f"get_skill(reply-style) {pointer}/10  firings {len(firings(w))}  unanswered {stuck}")


def single(w):
    events = [e.get("hook_event_name") for e in firings(w)]
    for id_, r in asked(w):
        if not r["done"]:
            print(f"{w.name:18} {id_} unanswered")
            continue
        kept = sorted(set(re.findall(r"(?:START|MID|END)-TOKEN-\w+", r["context"])))
        print(f"{w.name:18} {id_} firings={events} rule-in-context={RULE in r['context']} kept={kept} "
              f"elapsed={r['elapsed']}ms calls={r['calls']} marker={'[always-on]' in r['answer']} {r['answer'][-60:]!r}")
        for sub in r["subagents"]:
            print(f"{'':18} sub-agent marker={'[always-on]' in sub} {sub[-60:]!r}")
    log = w / "mcp.log"
    if log.exists():
        for line in log.read_text().splitlines():
            if "server started" in line:
                started = re.search(r'dir=(\S+) from=("[^"]*")', line).groups()
                print(f"{'':18} server started in {started}")


def copilot(w):
    """A Copilot harness run: its session is an events.jsonl in COPILOT_HOME."""
    for f in sorted(glob.glob(str(vsc / "copilot-home/session-state/*/events.jsonl"))):
        lines = [json.loads(l) for l in open(f) if l.strip()]
        start = next((e for e in lines if e["type"] == "session.start"), {})
        if start.get("data", {}).get("context", {}).get("cwd") != str(w):
            continue
        hooks = [e["data"]["hookType"] for e in lines if e["type"] == "hook.start"]
        answer = "\n".join(e["data"].get("content", "") for e in lines if e["type"] == "assistant.message")
        rule = any(RULE in json.dumps(e) for e in lines if e["type"] not in ("hook.start", "hook.end"))
        print(f"{w.name:18} hooks={hooks} firings={len(firings(w))} rule-in-session={rule} "
              f"marker={'[always-on]' in answer} {answer[-60:]!r}")


for w in sorted(p for p in root.iterdir() if (p / "asked.log").exists()):
    if w.name in ("mcp", "prompts", "both", "instructions"):
        prompts(w)
    elif w.name.startswith("copilot"):
        copilot(w)
    else:
        single(w)
