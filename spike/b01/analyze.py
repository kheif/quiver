import json, os, sys
root = sys.argv[1]
for v in sorted(os.listdir(root)):
    print(f"== {v}")
    for id in sorted(os.listdir(os.path.join(root, v))):
        d = os.path.join(root, v, id)
        tools, result, deferred = [], "", None
        try:
            for l in open(os.path.join(d, "out.jsonl")):
                try: m = json.loads(l)
                except Exception: continue
                if m.get("type") == "system" and m.get("subtype") == "init":
                    deferred = "ToolSearch" in m.get("tools", [])
                elif m.get("type") == "assistant":
                    for c in m["message"]["content"]:
                        if c["type"] == "tool_use":
                            n = c["name"].replace("mcp__quiver-spike__", "")
                            arg = c["input"].get("name") or c["input"].get("query") or ""
                            tools.append(f"{n}({arg})" if arg else n)
                elif m.get("type") == "result":
                    result = m.get("result") or ""
        except FileNotFoundError:
            pass
        marks = [x for x in ["[used:", "[read:checklist]", "[always-on]"] if x in result]
        print(f"{id:9} toolsearch={deferred!s:5} tools={tools} marks={marks}")
