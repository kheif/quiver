"""Writes corpus.json (id, name, description of every skill the local Quiver serves) from `quiver list --json`."""
import json
import subprocess

out = subprocess.run(["quiver", "list", "--json"], capture_output=True, text=True, check=True).stdout
skills = json.loads(out)["skills"]
corpus = [{"id": s["id"], "name": s["name"], "description": " ".join(s["description"].split())} for s in skills]
with open("corpus.json", "w") as f:
    json.dump(corpus, f, indent=1)
print(len(corpus), "skills")
