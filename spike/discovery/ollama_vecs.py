"""Embed the library and the queries through a running Ollama (/v1/embeddings), one model at a time,
and record memory, cold start and speed. Writes vecs/vec_<model>.json for the Go comparison."""
import json
import os
import subprocess
import time
import urllib.request

root = os.environ.get("SPIKE_ROOT", ".")
host = "http://127.0.0.1:11999"
corpus = json.load(open(f"{root}/corpus.json"))
queries = json.load(open("queries.json"))
docs = [f"{s['id']} {s['name']}: {s['description']}" for s in corpus]
qtexts = [q["text"] for q in queries]

models = {
    "nomic-embed-text": ("search_query: ", "search_document: "),
    "embeddinggemma": ("task: search result | query: ", "title: none | text: "),
    "qwen3-embedding:0.6b": ("Instruct: Given a web search query, retrieve relevant passages that answer the query\nQuery: ", ""),
    "all-minilm": ("", ""),
    "snowflake-arctic-embed:22m": ("Represent this sentence for searching relevant passages: ", ""),
    "granite-embedding:30m": ("", ""),
    "paraphrase-multilingual": ("", ""),
    "snowflake-arctic-embed:110m": ("Represent this sentence for searching relevant passages: ", ""),
    "mxbai-embed-large": ("Represent this sentence for searching relevant passages: ", ""),
}
import sys

if len(sys.argv) > 1:
    models = {m: models[m] for m in sys.argv[1:]}


def post(path, body):
    req = urllib.request.Request(host + path, json.dumps(body).encode(), {"Content-Type": "application/json"})
    with urllib.request.urlopen(req, timeout=600) as r:
        return json.load(r)


def embed(model, texts):
    return [d["embedding"] for d in post("/v1/embeddings", {"model": model, "input": texts})["data"]]


def rss_mb():
    out = subprocess.run(["ps", "-axo", "rss=,command="], capture_output=True, text=True).stdout
    tot = 0
    for line in out.splitlines():
        kb, _, cmd = line.strip().partition(" ")
        if "ollama serve" in cmd or "ollama-bin/llama-server" in cmd:
            tot += int(kb)
    return tot / 1024


def unload(model):
    post("/api/generate", {"model": model, "keep_alive": 0})
    time.sleep(2)


os.makedirs(f"{root}/vecs", exist_ok=True)
for model, (qp, dp) in models.items():
    for m in models:
        try:
            unload(m)
        except Exception:
            pass
    base = rss_mb()
    t0 = time.time()
    first = embed(model, [qp + qtexts[0]])
    cold = time.time() - t0
    loaded = rss_mb()
    t0 = time.time()
    dv = []
    for i in range(0, len(docs), 32):
        dv += embed(model, [dp + d for d in docs[i : i + 32]])
    lib = time.time() - t0
    qv = []
    lat = []
    for q in qtexts:
        t0 = time.time()
        qv += embed(model, [qp + q])
        lat.append(time.time() - t0)
    lat.sort()
    after = rss_mb()
    safe = model.replace(":", "-")
    with open(f"{root}/vecs/vec_{safe}.json", "w") as f:
        json.dump({"docs": dv, "queries": qv}, f)
    print(
        f"{model:22} dim {len(dv[0])} | rss idle {base:6.0f} -> loaded {loaded:6.0f} -> after library {after:6.0f} MB"
        f" | cold first embedding {cold:5.2f} s | library {len(docs)} skills {lib:5.2f} s"
        f" | query median {lat[len(lat)//2]*1000:5.0f} ms p95 {lat[int(len(lat)*.95)]*1000:5.0f} ms"
    )
