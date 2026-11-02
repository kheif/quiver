#!/usr/bin/env python3
"""usage: codex-hook-trust.py <hooks.json> [<config.toml>]
       codex-hook-trust.py --app-server <cwd>

Prints the hooks.state key and the trusted_hash that Codex 0.166 computes for
each command hook of a hooks.json: the SHA-256 of the hook's normalized
identity as compact JSON with sorted keys. With a config.toml, also writes the
hashes there, as the TUI's "Trust all and continue" does.

With --app-server, asks `codex app-server` instead: hooks/list for <cwd>, then
config/batchWrite of the current hash of every new or changed hook, the same
two requests the TUI sends. Prints the keys it trusted.

Only for the throwaway CODEX_HOME of codex-hooks-run.sh. See
docs/spike-hooks.md.
"""
import hashlib
import json
import re
import subprocess
import sys

LABEL = {"SessionStart": "session_start", "SubagentStart": "subagent_start"}


def hashes(path):
    for event, groups in json.load(open(path))["hooks"].items():
        for gi, group in enumerate(groups):
            for hi, h in enumerate(group["hooks"]):
                norm = {"type": "command", "command": h["command"], "timeout": h.get("timeout", 600),
                        "async": h.get("async", False)}
                if h.get("statusMessage"):
                    norm["statusMessage"] = h["statusMessage"]
                if h.get("additionalContextLimit", 2500) != 2500:
                    norm["additionalContextLimit"] = h["additionalContextLimit"]
                ident = {"event_name": LABEL[event], "hooks": [norm]}
                if group.get("matcher") is not None:
                    ident["matcher"] = group["matcher"]
                data = json.dumps(ident, sort_keys=True, separators=(",", ":"), ensure_ascii=False)
                yield f"{path}:{LABEL[event]}:{gi}:{hi}", "sha256:" + hashlib.sha256(data.encode()).hexdigest()


def write(config, trusted):
    try:
        text = open(config).read()
    except FileNotFoundError:
        text = ""
    for key, value in trusted:
        header = f'[hooks.state."{key}"]'
        line = f'trusted_hash = "{value}"'
        if header in text:
            text = re.sub(re.escape(header) + r"\ntrusted_hash = \"[^\"]*\"", lambda _: header + "\n" + line, text)
        else:
            text += f"\n{header}\n{line}\n"
    open(config, "w").write(text)


def app_server(cwd):
    p = subprocess.Popen(["codex", "app-server"], stdin=subprocess.PIPE, stdout=subprocess.PIPE, text=True)

    def send(message):
        p.stdin.write(json.dumps(dict(message, jsonrpc="2.0")) + "\n")
        p.stdin.flush()

    def call(i, method, params):
        send({"id": i, "method": method, "params": params})
        while True:
            m = json.loads(p.stdout.readline())
            if m.get("id") == i:
                if "error" in m:
                    sys.exit(f"{method}: {m['error']}")
                return m["result"]

    call(1, "initialize", {"clientInfo": {"name": "codex-hook-trust", "version": "0"}})
    send({"method": "initialized"})
    hooks = [h for e in call(2, "hooks/list", {"cwds": [cwd]})["data"] for h in e["hooks"]]
    review = {h["key"]: {"trusted_hash": h["currentHash"]} for h in hooks if h["trustStatus"] in ("untrusted", "modified")}
    if review:
        call(3, "config/batchWrite", {"edits": [{"keyPath": "hooks.state", "value": review, "mergeStrategy": "upsert"}],
                                      "reloadUserConfig": True})
    p.terminate()
    for key in review:
        print("trusted", key)


if sys.argv[1] == "--app-server":
    app_server(sys.argv[2])
else:
    trusted = list(hashes(sys.argv[1]))
    for key, value in trusted:
        print(key, value)
    if len(sys.argv) > 2:
        write(sys.argv[2], trusted)
