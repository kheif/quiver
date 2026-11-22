#!/usr/bin/env python3
"""Runs the cells of TASK62.1 on the shipped build (TASK62.10 item 4). Not product code.
usage: runs_shipped.py <workdir> <cell>[,<cell>...] [--n 20] [--par 4] [--model claude-opus-5-5]
<workdir> holds the quiver binary built from the release commit, and the throwaway
homes home-a (engine words) and home-g (engine local, model downloaded by
`quiver engine setup`), made by mkhome.sh from the library of mklib.py.
Unlike runs.py there is no prototype: the session-start text comes from
`quiver hook session-start claude-code` and the tools from `quiver mcp`, both of
the shipped build. Cells: a primary, f control (a plain question), g primary with
the local engine. Each run gets its own project folder with project-level
settings, reads no other settings (--setting-sources project) and sees only this
server (--strict-mcp-config)."""
import concurrent.futures as cf, json, os, shutil, subprocess, sys, time

HOME = {"a": "home-a", "f": "home-a", "g": "home-g"}
PROMPT = {
    "a": "Write a short draft of the announcement for Fenwick's new recurring bookings feature, for our customer newsletter.",
    "g": "Write a short draft of the announcement for Fenwick's new recurring bookings feature, for our customer newsletter.",
    "f": "Which programming language is this project written in? Answer in one sentence.",
}
TOOLS = ["mcp__quiver__*", "Read", "Glob", "Grep"]


def project(run_dir, work, cell):
    proj = os.path.join(run_dir, "proj")
    os.makedirs(os.path.join(proj, ".claude"), exist_ok=True)
    open(os.path.join(proj, "README.md"), "w").write("# Fenwick Bookings API\n\nA small Go service for the recurring bookings backend.\n")
    open(os.path.join(proj, "main.go"), "w").write("package main\n\nfunc main() {}\n")
    open(os.path.join(proj, "go.mod"), "w").write("module fenwick/bookings\n\ngo 1.22\n")
    home = os.path.join(work, HOME[cell])
    quiver = os.path.join(work, "quiver")
    hook = f"QUIVER_NO_UPDATE_CHECK=1 QUIVER_HOME='{home}' '{quiver}' hook session-start claude-code"
    settings = {"hooks": {"SessionStart": [{"hooks": [{"type": "command", "command": hook}]}]}}
    json.dump(settings, open(os.path.join(proj, ".claude", "settings.json"), "w"))
    mcp = {"mcpServers": {"quiver": {"command": quiver, "args": ["mcp"], "env": {"QUIVER_HOME": home, "QUIVER_NO_UPDATE_CHECK": "1"}}}}
    mcp_path = os.path.join(run_dir, "mcp.json")
    json.dump(mcp, open(mcp_path, "w"))
    return proj, mcp_path


def claude(proj, mcp_path, prompt, out, model):
    cmd = ["claude", "-p", prompt, "--model", model, "--mcp-config", mcp_path, "--strict-mcp-config", "--setting-sources", "project",
           "--disable-slash-commands", "--allowedTools", *TOOLS, "--max-turns", "14", "--output-format", "stream-json", "--verbose"]
    with open(out, "w") as f:
        try:
            subprocess.run(cmd, cwd=proj, stdin=subprocess.DEVNULL, stdout=f, stderr=subprocess.STDOUT, timeout=420)
        except subprocess.TimeoutExpired:
            f.write('\n{"type":"spike_timeout"}\n')


def one(work, cell, i, model):
    run_dir = os.path.join(work, "runs", cell, f"{i:02d}")
    if os.path.exists(os.path.join(run_dir, "done")):
        return
    shutil.rmtree(run_dir, ignore_errors=True)
    os.makedirs(run_dir)
    proj, mcp_path = project(run_dir, work, cell)
    claude(proj, mcp_path, PROMPT[cell], os.path.join(run_dir, "out1.jsonl"), model)
    open(os.path.join(run_dir, "done"), "w").write(time.strftime("%F %T"))


if __name__ == "__main__":
    work = os.path.abspath(sys.argv[1])
    cells = sys.argv[2].split(",")
    arg = lambda name, default: sys.argv[sys.argv.index(name) + 1] if name in sys.argv else default
    n, par, model = int(arg("--n", 20)), int(arg("--par", 4)), arg("--model", "claude-opus-5-5")
    jobs = [(c, i) for i in range(1, n + 1) for c in cells]
    with cf.ThreadPoolExecutor(par) as ex:
        for f in cf.as_completed([ex.submit(one, work, c, i, model) for c, i in jobs]):
            f.result()
    print("done", cells, n)
