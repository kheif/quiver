#!/usr/bin/env python3
"""Summarises the runs of runs.py. usage: analyze.py <workdir> [cell...]
A skill is found when a get_skill of a fitting skill follows a search_skills in
the same run (cell h, where the skill is listed, counts a get_skill alone) and
the run did not read skill files from ~/.quiver instead."""
import collections, glob, json, os, sys

FIT = {"fenwick-announcements", "tone-ladder"}

def events(path):
    """(kind, detail) per tool call and the result message of one stream-json file."""
    ev, res, model = [], None, None
    for l in open(path, errors="replace"):
        if not l.startswith("{"):
            continue
        try:
            j = json.loads(l)
        except ValueError:
            continue
        t = j.get("type")
        if t == "system" and j.get("subtype") == "init":
            model = j.get("model")
        elif t == "assistant":
            for c in j["message"]["content"]:
                if c["type"] == "tool_use":
                    ev.append((c["name"], c["input"]))
        elif t == "result":
            res = j
        elif t == "spike_timeout":
            res = {"subtype": "timeout"}
    return ev, res, model

def run_summary(d):
    ev, usage, cost, model, status, turn = [], collections.Counter(), 0.0, None, [], {}
    for k, f in enumerate(sorted(glob.glob(os.path.join(d, "out*.jsonl"))), 1):
        e, res, m = events(f)
        model = model or m
        status.append(res.get("subtype") if res else "no-result")
        if res:
            cost += res.get("total_cost_usd") or 0
            for key in ("input_tokens", "output_tokens", "cache_creation_input_tokens", "cache_read_input_tokens"):
                usage[key] += (res.get("usage") or {}).get(key, 0)
        ev += [(k, n, i) for n, i in e]
    searches = [i.get("query") for k, n, i in ev if n.endswith("search_skills")]
    gets = [i.get("id") for k, n, i in ev if n.endswith("get_skill")]
    first_search = next((x for x, (k, n, i) in enumerate(ev) if n.endswith("search_skills")), None)
    fit_gets = [(x, i.get("id")) for x, (k, n, i) in enumerate(ev) if n.endswith("get_skill") and (i.get("id") or "").split("/")[-1] in FIT]
    after = [(x, g) for x, g in fit_gets if first_search is not None and x > first_search]
    reads = [i.get("file_path") or i.get("path") or i.get("pattern") or "" for k, n, i in ev if n in ("Read", "Glob", "Grep")]
    bypass = any(".quiver" in str(r) for r in reads)
    which = sorted({g.split("/")[-1] for x, g in (fit_gets if os.path.basename(os.path.dirname(d)) == "h" else after)})
    return {
        "status": status, "model": model, "cost": cost, "usage": dict(usage),
        "searches": searches, "gets": gets, "fit": which, "bypass": bypass,
        "fit_turn": sorted({ev[x][0] for x, g in (fit_gets if os.path.basename(os.path.dirname(d)) == "h" else after)}),
        "searched_turn": sorted({k for k, n, i in ev if n.endswith("search_skills")}),
        "found": bool(which) and not bypass,
    }

def main():
    work = os.path.abspath(sys.argv[1])
    cells = sys.argv[2:] or sorted(os.listdir(os.path.join(work, "runs")))
    out = {}
    for c in cells:
        rs = {os.path.basename(d): run_summary(d) for d in sorted(glob.glob(os.path.join(work, "runs", c, "*"))) if os.path.exists(os.path.join(d, "done"))}
        n = len(rs)
        s = {
            "runs": n,
            "found": sum(r["found"] for r in rs.values()),
            "found_A": sum("fenwick-announcements" in r["fit"] for r in rs.values()),
            "found_B": sum("tone-ladder" in r["fit"] for r in rs.values()),
            "searched": sum(bool(r["searches"]) for r in rs.values()),
            "bypass": sum(r["bypass"] for r in rs.values()),
            "not_success": sum(any(x != "success" for x in r["status"]) for r in rs.values()),
            "models": sorted({r["model"] or "?" for r in rs.values()}),
            "cost_usd": round(sum(r["cost"] for r in rs.values()), 2),
            "tokens": {k: sum(r["usage"].get(k, 0) for r in rs.values()) for k in ("input_tokens", "output_tokens", "cache_creation_input_tokens", "cache_read_input_tokens")},
            "per_run": rs,
        }
        out[c] = s
        print(f"{c}: runs {n}  found {s['found']}  (A {s['found_A']}, B {s['found_B']})  searched {s['searched']}  bypass {s['bypass']}  not-success {s['not_success']}  ${s['cost_usd']}  {s['models']}")
    json.dump(out, open(os.path.join(work, "summary.json"), "w"), indent=1)

main()
