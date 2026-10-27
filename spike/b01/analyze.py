import glob, json, os, re, sys
root = sys.argv[1]
for v in sorted(os.listdir(root)):
    print(f"== {v}")
    for id in sorted(os.listdir(os.path.join(root, v))):
        d = os.path.join(root, v, id)
        # One out*.jsonl per turn; the last turn's result is the one shown.
        tools, result, deferred = [], "", None
        for f in sorted(glob.glob(os.path.join(d, "out*.jsonl"))):
            for l in open(f):
                try: m = json.loads(l)
                except Exception: continue
                if m.get("type") == "system" and m.get("subtype") == "init":
                    deferred = "ToolSearch" in m.get("tools", [])
                elif m.get("type") == "assistant":
                    for c in m["message"]["content"]:
                        if c["type"] == "tool_use":
                            n = re.sub(r"^mcp__quiver(-spike)?__", "", c["name"])
                            arg = c["input"].get("id") or c["input"].get("name") or c["input"].get("query") or ""
                            tools.append(f"{n}({arg})" if arg else n)
                elif m.get("type") == "result":
                    result = m.get("result") or ""
        marks = [x for x in ["[used:", "[read:checklist]", "[always-on]"] if x in result]
        print(f"{id:9} toolsearch={deferred!s:5} tools={tools} marks={marks}")
