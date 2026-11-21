#!/usr/bin/env python3
"""Builds the throwaway library of TASK62.1 item 1: usage: mklib.py <outdir>.
Copies every skill that the local Quiver serves (sources of four origins) into
<outdir>/lib/<source>/<name>/, adds the two fixture skills as source "spike",
and writes <outdir>/lib/index.json (id, name, description, dir)."""
import json, os, re, shutil, subprocess, sys
out = os.path.abspath(sys.argv[1])
here = os.path.dirname(os.path.abspath(__file__))
lib = os.path.join(out, "lib")
shutil.rmtree(lib, ignore_errors=True)
os.makedirs(lib)
skills = json.loads(subprocess.run(["quiver", "list", "--json"], capture_output=True, text=True, check=True).stdout)["skills"]
src_root = os.path.expanduser("~/.quiver/sources")
index, missing = [], []
by_name = {}
for source in os.listdir(src_root):
    for root, dirs, files in os.walk(os.path.join(src_root, source)):
        if "SKILL.md" in files:
            m = re.search(r"^name:\s*(.+)$", open(os.path.join(root, "SKILL.md")).read(), re.M)
            if m:
                by_name[(source, m.group(1).strip().strip("\"'"))] = root
for s in skills:
    d = by_name.get((s["source"], s["name"]))
    if not d:
        missing.append(s["id"])
        continue
    dest = os.path.join(lib, s["source"], s["name"])
    shutil.copytree(d, dest, dirs_exist_ok=True, symlinks=True)
    index.append({"id": s["id"], "name": s["name"], "description": " ".join(s["description"].split()), "dir": os.path.join(s["source"], s["name"])})
for name in os.listdir(os.path.join(here, "fixtures")):
    d = os.path.join(here, "fixtures", name)
    text = open(os.path.join(d, "SKILL.md")).read()
    desc = re.search(r"^description:\s*(.+)$", text, re.M).group(1)
    shutil.copytree(d, os.path.join(lib, "spike", name))
    index.append({"id": "spike/" + name, "name": name, "description": desc, "dir": "spike/" + name})
json.dump(index, open(os.path.join(lib, "index.json"), "w"), indent=1)
print(len(index), "skills in the library;", len(missing), "not found on disk:", missing)
