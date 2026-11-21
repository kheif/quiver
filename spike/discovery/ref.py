"""Reference vectors and token ids from the Python model2vec package, to check the Go loader against."""
import json
import os
import sys
import time

import numpy as np
from model2vec import StaticModel

root = os.environ.get("SPIKE_ROOT", ".")
corpus = json.load(open(f"{root}/corpus.json"))
queries = json.load(open("queries.json"))
docs = [f"{s['id']} {s['name']}: {s['description']}" for s in corpus]
qtexts = [q["text"] for q in queries]

for name in sys.argv[1:]:
    m = StaticModel.from_pretrained(f"{root}/models/{name}")
    t0 = time.time()
    dv = m.encode(docs)
    t1 = time.time()
    qv = m.encode(qtexts)
    t2 = time.time()
    sample = docs[:40] + qtexts
    ids = [m.tokenizer.encode(t, add_special_tokens=False).ids for t in sample]
    out = {"name": name, "docs": dv.tolist(), "queries": qv.tolist(), "sample": sample, "ids": ids}
    with open(f"{root}/ref_{name}.json", "w") as f:
        json.dump(out, f)
    print(name, "dim", dv.shape[1], f"docs {t1 - t0:.2f}s queries {t2 - t1:.3f}s", "normalize", m.normalize, "unk", getattr(m, "unk_token_id", None))
