#!/usr/bin/env python3
"""Runs the cells of TASK62.1 item 1 on Claude Code. Not product code.
usage: runs.py <workdir> <cell>[,<cell>...] [--n 20] [--par 6]
<workdir> holds lib/ (mklib.py) and the discovery binary (go build). Each run
gets its own project folder with project-level settings, reads no other
settings (--setting-sources project) and sees only the prototype server (or the
real quiver binary for the baseline cell) through --strict-mcp-config."""
import concurrent.futures as cf, json, os, shutil, subprocess, sys, time

CELLS = {
  "a": "primary: candidate wording, no skill listed",
  "b": "baseline: today's texts and server, skills not assigned",
  "c": "candidate wording, 16 skills listed, none fits",
  "d": "two turns, the second does not name the topic",
  "e": "primary, prompt in Turkish",
  "f": "control: a plain question about the project",
  "g": "primary with the local engine",
  "h": "reference: both fitting skills listed on demand",
}
FIT = {"spike/fenwick-announcements", "spike/tone-ladder"}
PROMPT = {
  "en": "Write a short draft of the announcement for Fenwick's new recurring bookings feature, for our customer newsletter.",
  "tr": "Fenwick'in yeni tekrarlayan rezervasyon özelliği için müşteri bültenine gidecek kısa bir duyuru taslağı yaz.",
  "d1": "The announcement of Fenwick's new recurring bookings feature is overdue, and the #launch-comms Slack channel keeps asking about it. What should I tell them?",
  "d2": "Write a draft about the last topic we discussed and send it.",
  "f": "Which programming language is this project written in? Answer in one sentence.",
}
TOOLS = ["mcp__quiver__*", "Read", "Glob", "Grep"]

def cue_no(cell):
    return "2" if cell.endswith("2") else "1"

def sh(*a, **k):
    return subprocess.run(a, capture_output=True, text=True, **k)

def block(entries):
    lines = ["<quiver-skills>", "The other skills served here, as id: description. Before starting a task, check whether one fits. If it does, call the get_skill tool of the quiver MCP server with its id and follow it."]
    lines += [f"- {e['id']}: {e['description']}" for e in entries]
    lines.append("</quiver-skills>")
    return "\n".join(lines)

def setup(work, cell):
    """Writes the cell's files into <work>/cells/<cell>: hook text, searchable list, mcp config pieces."""
    d = os.path.join(work, "cells", cell)
    os.makedirs(d, exist_ok=True)
    base = cell[0]
    idx = json.load(open(os.path.join(work, "lib", "index.json")))
    byid = {e["id"]: e for e in idx}
    cue = sh(os.path.join(work, "discovery-new"), "cue", cue_no(cell)).stdout.strip()
    sixteen = [e for e in idx if e["id"].startswith("obra/")] + [e for e in idx if e["id"].startswith("codeswithroh/")]
    sixteen = sixteen[:16]
    ids = lambda es: [e["id"] for e in es]
    if base in "adeg" or base == "f":
        hook = "<quiver-search>\nNo skills are listed in this folder. " + cue + "\n</quiver-search>"
        searchable = ids(idx)
    elif base == "c":
        hook = block(sixteen) + "\n\n<quiver-search>\n" + cue + "\n</quiver-search>"
        searchable = [i for i in ids(idx) if i not in ids(sixteen)]
    elif base == "h":
        hook = block([byid[i] for i in sorted(FIT)])
        searchable = None
    else:
        hook = None
        searchable = None
    if hook is not None:
        open(os.path.join(d, "hook.txt"), "w").write(hook + "\n")
    if searchable is not None:
        open(os.path.join(d, "searchable.txt"), "w").write("\n".join(searchable) + "\n")
    return d

def base_home(work):
    """b: a throwaway QUIVER_HOME with every skill of the library in local/, nothing assigned, served by the real quiver binary."""
    home = os.path.join(work, "quiver-home-b")
    if os.path.exists(os.path.join(home, "local")) and len(os.listdir(os.path.join(home, "local"))) > 200:
        return home
    shutil.rmtree(home, ignore_errors=True)
    env = dict(os.environ, QUIVER_HOME=home)
    subprocess.run(["quiver", "init"], env=env, input="n\n", capture_output=True, text=True)
    idx = json.load(open(os.path.join(work, "lib", "index.json")))
    seen = set()
    for e in idx:
        if e["name"] in seen:
            continue
        seen.add(e["name"])
        shutil.copytree(os.path.join(work, "lib", e["dir"]), os.path.join(home, "local", e["name"]), symlinks=True)
    return home

def project(run_dir, work, cell):
    proj = os.path.join(run_dir, "proj")
    os.makedirs(os.path.join(proj, ".claude"), exist_ok=True)
    open(os.path.join(proj, "README.md"), "w").write("# Fenwick Bookings API\n\nA small Go service for the recurring bookings backend.\n")
    open(os.path.join(proj, "main.go"), "w").write("package main\n\nfunc main() {}\n")
    open(os.path.join(proj, "go.mod"), "w").write("module fenwick/bookings\n\ngo 1.22\n")
    cdir = os.path.join(work, "cells", cell)
    disc = os.path.join(work, "discovery")
    log = os.path.join(run_dir, "server.log")
    settings = {}
    if cell == "b":
        home = base_home(work)
        mcp = {"mcpServers": {"quiver": {"command": shutil.which("quiver"), "args": ["mcp"], "env": {"QUIVER_HOME": home}}}}
        settings = {"hooks": {"SessionStart": [{"hooks": [{"type": "command", "command": f"QUIVER_HOME='{home}' '{shutil.which('quiver')}' hook session-start claude-code"}]}]}}
    else:
        args = ["serve", "--lib", os.path.join(work, "lib"), "--log", log, "--cue", cue_no(cell)]
        disc = os.path.join(work, "discovery-new")
        if cell == "h":
            args += ["--variant", "today"]
        else:
            args += ["--searchable", os.path.join(cdir, "searchable.txt")]
        if cell == "g":
            args += ["--engine", "local", "--model", os.path.join(work, "model-minilm"), "--vecs", os.path.join(work, "vecs-minilm.json")]
        mcp = {"mcpServers": {"quiver": {"command": disc, "args": args}}}
        settings = {"hooks": {"SessionStart": [{"hooks": [{"type": "command", "command": f"cat '{os.path.join(cdir, 'hook.txt')}'"}]}]}}
    json.dump(settings, open(os.path.join(proj, ".claude", "settings.json"), "w"))
    mcp_path = os.path.join(run_dir, "mcp.json")
    json.dump(mcp, open(mcp_path, "w"))
    return proj, mcp_path

def claude(proj, mcp_path, prompt, out, resume=None):
    cmd = ["claude", "-p", prompt, "--mcp-config", mcp_path, "--strict-mcp-config", "--setting-sources", "project",
           "--disable-slash-commands", "--allowedTools", *TOOLS, "--max-turns", "14", "--output-format", "stream-json", "--verbose"]
    if resume:
        cmd += ["--resume", resume]
    with open(out, "w") as f:
        try:
            subprocess.run(cmd, cwd=proj, stdin=subprocess.DEVNULL, stdout=f, stderr=subprocess.STDOUT, timeout=420)
        except subprocess.TimeoutExpired:
            f.write('\n{"type":"spike_timeout"}\n')

def session_id(path):
    for l in open(path):
        if l.startswith("{"):
            try:
                j = json.loads(l)
            except ValueError:
                continue
            if j.get("session_id"):
                return j["session_id"]

def one(work, cell, i):
    run_dir = os.path.join(work, "runs", cell, f"{i:02d}")
    if os.path.exists(os.path.join(run_dir, "done")):
        return
    shutil.rmtree(run_dir, ignore_errors=True)
    os.makedirs(run_dir)
    proj, mcp_path = project(run_dir, work, cell)
    if cell[0] == "d":
        claude(proj, mcp_path, PROMPT["d1"], os.path.join(run_dir, "out1.jsonl"))
        sid = session_id(os.path.join(run_dir, "out1.jsonl"))
        if sid:
            claude(proj, mcp_path, PROMPT["d2"], os.path.join(run_dir, "out2.jsonl"), resume=sid)
    else:
        p = PROMPT["tr"] if cell == "e" else PROMPT["f"] if cell == "f" else PROMPT["en"]
        claude(proj, mcp_path, p, os.path.join(run_dir, "out1.jsonl"))
    open(os.path.join(run_dir, "done"), "w").write(time.strftime("%F %T"))

if __name__ == "__main__":
    work = os.path.abspath(sys.argv[1])
    cells = sys.argv[2].split(",")
    n = int(sys.argv[sys.argv.index("--n") + 1]) if "--n" in sys.argv else 20
    par = int(sys.argv[sys.argv.index("--par") + 1]) if "--par" in sys.argv else 6
    for c in cells:
        setup(work, c)
    jobs = [(c, i) for i in range(1, n + 1) for c in cells]
    with cf.ThreadPoolExecutor(par) as ex:
        for f in cf.as_completed([ex.submit(one, work, c, i) for c, i in jobs]):
            f.result()
    print("done", cells, n)
