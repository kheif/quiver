#!/usr/bin/env python3
"""usage: claude-hooks-analyze.py <dir>

Summarizes claude-hooks-run.sh. For the prompt runs: always-on, fitting and
other skills, and get_skill calls for reply-style. For single runs: seconds,
hook outcomes, tools and the start of the answer. For sub-agents: the marker in
the sub-agent's reply. For the chains and the interactive runs: the sources the
hook fired on and the marker in each answer.
"""
import json
import pathlib
import re
import sys

root = pathlib.Path(sys.argv[1])


def events(path):
    for line in path.read_text(errors="replace").splitlines():
        try:
            yield json.loads(line)
        except ValueError:
            continue


def session(path):
    """Tool calls, final answer and hook outcomes of a stream-json file."""
    tools, answer, hooks = [], "", []
    for e in events(path):
        if e.get("type") == "assistant":
            for c in e["message"]["content"]:
                if c["type"] == "tool_use":
                    name = c["name"].replace("mcp__quiver__", "")
                    arg = c["input"].get("id") or c["input"].get("name") or ""
                    tools.append(f"{name}({arg})" if arg else name)
        elif e.get("type") == "result":
            answer = e.get("result") or ""
        elif e.get("subtype") == "hook_response":
            hooks.append(f"{e.get('outcome')}:{e.get('exit_code')}")
    return tools, answer, hooks


def hook_log(d):
    log = d / "hook.log"
    return [e for e in events(log)] if log.exists() else []


def prompts(d):
    always = fitting = other = pointer = 0
    for f in sorted((d / "out").glob("*.jsonl")):
        tools, answer, _ = session(f)
        always += "[always-on]" in answer
        pointer += "get_skill(reply-style)" in tools
        if f.stem.startswith("fit"):
            fitting += "[used:" in answer
        else:
            other += any(t.startswith("get_skill(") and "reply-style" not in t for t in tools)
    print(f"{d.name:24} always-on {always}/10  fitting {fitting}/6  other {other}/4  get_skill(reply-style) {pointer}/10")


def single(d):
    tools, answer, hooks = session(d / "out.jsonl")
    secs = (d / "seconds.txt").read_text().strip()
    marker = "[always-on]" in answer
    print(f"{d.name:24} {secs:>3}s hooks={hooks} tools={tools} marker={marker} {answer[:90]!r}")


def subagent(d):
    single(d)
    log = hook_log(d)
    transcript = pathlib.Path(log[0]["transcript_path"])
    for f in sorted(transcript.with_suffix("").glob("subagents/*.jsonl")):
        text = ""
        for e in events(f):
            if e.get("type") == "assistant":
                for c in e["message"]["content"]:
                    if c.get("type") == "text":
                        text = c["text"]
        print(f"{'':24} sub-agent marker={'[always-on]' in text}")


def chain(d):
    sources = [e["source"] for e in hook_log(d) if "source" in e]
    _, answer, _ = session(d / "3.jsonl")
    tokens = re.findall(r"CONTEXT-TOKEN-\w+", session(d / "2.jsonl")[1]) if (d / "2.jsonl").exists() else []
    print(f"{d.name:24} sources={sources} marker={'[always-on]' in answer} tokens={tokens}")


def interactive(d):
    log = hook_log(d)
    print(f"{d.name:24} sources={[e['source'] for e in log]} {(d / 'pretrust.txt').read_text().strip()}")
    for f in sorted(pathlib.Path(log[0]["transcript_path"]).parent.glob("*.jsonl"), key=lambda p: p.stat().st_mtime):
        for e in events(f):
            content = (e.get("message") or {}).get("content")
            if e.get("isCompactSummary"):
                print(f"{'':24} compact summary has the rule: {'always-on' in json.dumps(e)}")
            elif e.get("type") == "user" and isinstance(content, str) and content.startswith("Suggest"):
                print(f"{'':24} {f.stem[:8]} {content[:30]}")
            elif e.get("type") == "assistant":
                for c in content:
                    if c.get("type") == "text" and len(c["text"]) > 30:
                        print(f"{'':24} {f.stem[:8]} marker={'[always-on]' in c['text']}")


for d in sorted(p for p in root.iterdir() if p.is_dir()):
    if d.name.startswith("prompts-"):
        prompts(d)
    elif d.name.startswith("sub-"):
        subagent(d)
    elif d.name.startswith("chain-"):
        chain(d)
    elif d.name.startswith("tui-"):
        interactive(d)
    elif (d / "out.jsonl").exists():
        single(d)
