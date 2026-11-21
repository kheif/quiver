# Discovery spike results (TASK62.1)

**Status: partial.** This file holds items 2 and 3 of TASK62.1 for the embedding models: the comparison of the search engines and the size, memory and speed of the routes. The comparison is read on English queries, because English is the language that the library and the assistants' searches are written in; the Turkish queries are reported on their own. Item 1 was run on Claude Code only (the section "Runs with Claude Code"); the per-system setup steps of the runtimes are not measured. No threshold fixed in TASK62.1 was changed; where a rule had to be read differently, the text says so.

The code is in `spike/discovery/` (not product code, own `go.mod`). `queries.json` is the labelled query set; `corpus.py`, `ref.py`, `ollama_vecs.py` and `main.go` (`validate`, `vcheck`, `eval`, `bench`) make every number below.

## Setup

- Machine: Apple M1 (8 cores), 16 GB, macOS. Every run is in a fresh process with the model files in the operating system's cache (a cold disk read is slower and was not measured).
- Library: 262 skills of four sources (224, 22, 15 and 1 skills; the 38 of the three smaller sources are programming or workflow skills, the rest are business skills). The text that is embedded and searched is the ID, the name and the description.
- Queries, 210 in six kinds, each labelled before any engine scored it:
  - Set A, written first: 30 paraphrases in English (no word of the skill's name), 30 in the style of an assistant, 30 paraphrases in Turkish, 30 that fit nothing (5 of them in Turkish).
  - Set B, added after the first comparison to widen the English test: 60 paraphrases of skills that set A does not use (26 of them skills of the three smaller sources) and 30 English queries that fit nothing. Set B shares more words with the skills than set A (80% of its queries share two or more words with the description of their skill, against 70%; 17% share a word of the name, against none), which is closer to how people write and easier for a word search. Where neighbouring skills fit equally well, a query names all of them.
- Split and cutoff: within each kind the split alternates down the file, even-numbered queries (calibration) and odd-numbered (test). For each engine and model the cutoff is the highest top score of the calibration queries that fit nothing, so that none of them returns anything. The measure for the fitting kinds is the share of queries whose fitting skill is among the first five results above the cutoff; for the kinds that fit nothing it is the share that returns nothing.
- The English run leaves out the 35 Turkish queries and calibrates on English queries only. Its test half has 15 paraphrases of set A, 30 of set B, 15 assistant-style queries and 27 that fit nothing.
- Checks of the Go code against the reference: the static models give the same token ids as the Python package for all 160 texts compared and vectors with cosine 1.00000 for all 382 texts; the BERT model all-MiniLM-L6-v2 written in Go gives vectors with cosine 1.00000 against those that Ollama makes of the same weights, for all 472 texts.

## Engines in English, test half

"With BM25" is the reciprocal-rank fusion (k = 60) of the ranked word search and the model, each cut by its own cutoff, as TASK62.6 plans it. Paraphrases A and B are the two sets of English paraphrases; "fits nothing" counts the 27 queries that return nothing.

| Engine | Route | Para. A (15) | Para. B (30) | Paraphrases (45) | Assistant style (15) | All fitting (60) | Fits nothing (27) | With BM25: paraphrases | With BM25: all fitting | With BM25: fits nothing |
|---|---|---|---|---|---|---|---|---|---|---|
| matching of today | words | 0 | 1 | 1 (2%) | 9 | 10 (17%) | 27 | | | |
| ranked word search (BM25) | words | 8 | 22 | 30 (67%) | 14 | 44 (73%) | 26 | | | |
| potion-base-2M | in process, static | 3 | 14 | 17 (38%) | 11 | 28 (47%) | 27 | 32 (71%) | 47 (78%) | 26 |
| potion-base-4M | in process, static | 8 | 20 | 28 (62%) | 14 | 42 (70%) | 27 | 34 (76%) | 49 (82%) | 26 |
| potion-base-8M | in process, static | 10 | 22 | 32 (71%) | 15 | 47 (78%) | 25 | 36 (80%) | 51 (85%) | 24 |
| potion-base-32M | in process, static | 7 | 19 | 26 (58%) | 14 | 40 (67%) | 26 | 34 (76%) | 49 (82%) | 25 |
| potion-retrieval-32M | in process, static | 8 | 15 | 23 (51%) | 15 | 38 (63%) | 27 | 33 (73%) | 48 (80%) | 26 |
| potion-multilingual-128M | in process, static | 11 | 20 | 31 (69%) | 15 | 46 (77%) | 26 | 35 (78%) | 50 (83%) | 25 |
| **all-MiniLM-L6-v2** (22M) | in process, BERT in Go | 12 | 25 | 37 (82%) | 14 | 51 (85%) | 23 | 41 (91%) | 56 (93%) | 22 |
| gte-small (33M) | in process, BERT in Go | 11 | 27 | 38 (84%) | 15 | 53 (88%) | 26 | 41 (91%) | 56 (93%) | 25 |
| bge-small-en-v1.5 (33M) | in process, BERT in Go | 8 | 21 | 29 (64%) | 15 | 44 (73%) | 25 | 38 (84%) | 53 (88%) | 24 |
| e5-small-v2 (33M) | in process, BERT in Go | 3 | 18 | 21 (47%) | 14 | 35 (58%) | 26 | 35 (78%) | 50 (83%) | 25 |
| snowflake-arctic-embed 22M | runtime | 6 | 19 | 25 (56%) | 15 | 40 (67%) | 26 | 33 (73%) | 48 (80%) | 25 |
| granite-embedding 30M | runtime | 10 | 24 | 34 (76%) | 14 | 48 (80%) | 22 | 39 (87%) | 54 (90%) | 21 |
| snowflake-arctic-embed 110M | runtime | 9 | 22 | 31 (69%) | 15 | 46 (77%) | 27 | 38 (84%) | 53 (88%) | 26 |
| nomic-embed-text | runtime | 8 | 22 | 30 (67%) | 15 | 45 (75%) | 25 | 38 (84%) | 53 (88%) | 24 |
| paraphrase-multilingual (278M) | runtime | 11 | 22 | 33 (73%) | 13 | 46 (77%) | 25 | 40 (89%) | 55 (92%) | 24 |
| qwen3-embedding 0.6B | runtime | 14 | 30 | 44 (98%) | 15 | 59 (98%) | 23 | 45 (100%) | 60 (100%) | 22 |
| embeddinggemma (300M) | runtime | 14 | 30 | 44 (98%) | 15 | 59 (98%) | 23 | 44 (98%) | 59 (98%) | 23 |

all-MiniLM-L6-v2 is the same model as Ollama's `all-minilm`; the runtime row for it would be the same numbers.

Reading the table:

- The matching of today finds the fitting skill for 2% of the paraphrases. The ranked word search finds it for 67%, and for 93% of the assistant-style queries, which share most of their words with the skill.
- A static model alone is about level with the word search (potion-base-8M 71% against 67%). On set A alone potion-base-8M is ahead of the word search by 3 queries of 30 (25 against 22, with the assistant-style queries); on set B, whose queries share more words with the skills, it is level (22 against 22 of 30). The static models do not get better with size: potion-base-8M is better than the 32M models, and the retrieval-tuned potion-retrieval-32M is not better than the base model on this library.
- The BERT models that run in Go are better than the static models: all-MiniLM-L6-v2 82% and gte-small 84% alone. bge-small-en-v1.5 and e5-small-v2 are weaker here (64%, 47%) although they were given the query prefixes that their authors name.
- Fused with the word search, all-MiniLM-L6-v2 and gte-small reach 91% on the paraphrases and 93% on all fitting queries, and potion-base-8M 80% and 85%. The two largest models, embeddinggemma and qwen3-embedding, reach 98% alone; fusion adds nothing to embeddinggemma.
- The gap between all-MiniLM-L6-v2 fused and embeddinggemma is 3 of 45 paraphrases (41 against 44) and 3 of 60 fitting queries. With 45 test paraphrases a difference of one or two queries is within chance, so the order of the models inside the 82% to 91% group is not settled; the gap between that group and the word search, and between the word search and today's matching, is.
- Fusion costs some of the share of queries that fit nothing and still return something: all-MiniLM-L6-v2 with BM25 returns nothing for 22 of 27, the word search alone for 26 of 27.
- The labels name the fitting skills for each query, but other skills may fit as well; the rates of the best models are an underestimate.

## Engines in Turkish (set A, 15 test queries, run on all queries)

| Engine | Paraphrase in Turkish (15) |
|---|---|
| matching of today, ranked word search, all-MiniLM-L6-v2, granite-embedding 30M, the other English-only models | 0 or 1 |
| potion-multilingual-128M | 4 |
| paraphrase-multilingual (runtime) | 11 |
| qwen3-embedding 0.6B (runtime) | 12 |
| embeddinggemma (runtime) | 14 |

A model that was trained on English finds nothing in Turkish, as the word search does. Only the runtime models read it well.

## Ranked word search, as measured

The numbers above come from `spike/discovery/words.go`. BM25 over two fields: the name counts three times and the description once, the ID is not scored. Terms are lower-cased, split at every character that is not a letter or a digit, stripped of the stop words in `stop` and lightly stemmed (a plural -s, and -ies to -y). k1 = 1.2, b = 0.75; the document frequencies and the average length are taken over the 262 skills. An exact name gets +1000 and a name prefix +500, which lifts them above every score. The cutoff of the English run is 9.784: the highest top score of the English calibration queries that fit nothing. A cutoff of this kind depends on the statistics of the library, so the product takes the statistics from the whole library whatever set of skills is searched (TASK62.5).

## Routes

### In process, BERT written in Go (plain Go, no SIMD, no cgo)

The weights are `model.safetensors` from `huggingface.co/<repository>` and `tokenizer.json`; the loader maps the file. The F32 file is used as it is; gte-small is stored as F16 and is converted to float32 when it is loaded. Fresh process, file mapped, all 8 cores:

| Model | Download | License | SHA-256 of `model.safetensors` | Load | Memory loaded | Memory, library embedded (peak) | Embed 262 skills | Per query |
|---|---|---|---|---|---|---|---|---|
| all-MiniLM-L6-v2 (revision `1110a243fdf4`) | 91 MB | Apache-2.0 | `53aa51172d142c89d9012cce15ae4d6cc0ca6895895114379cacb4fab128d9db` | 16 ms | 11 MB | 84 MB (85 MB) | 22 s | 20 ms |
| gte-small (`17e1f347d17f`) | 67 MB | MIT | `9a1eb90bbac323ea08aa5629b624fe6ae75db121b904799c2266a1e2c2de22d2` | 88 ms | 203 MB | 335 MB | 44 s | 40 ms |
| bge-small-en-v1.5 (`5c38ec7c405e`) | 134 MB | MIT | `3c9f31665447c8911517620762200d2245a2518d6e7208acc78cd9db317e21ad` | 10 ms | 11 MB | 126 MB | 44 s | 52 ms |
| e5-small-v2 (`ffb93f3bd404`) | 134 MB | MIT | `45bfa60070649aae2244fbc9d508537779b93b6f353c17b0f95ceccb1c5116c1` | not repeated | | | | |

Pinned files of all-MiniLM-L6-v2, checked against the Hugging Face API on 2026-11-21: revision `1110a243fdf4706b3f48f1d95db1a4f5529b4d41`. `model.safetensors` is 90,868,376 bytes with the SHA-256 above, which the hub lists too. `tokenizer.json` is 466,247 bytes with the SHA-256 `be50c3628f2bf5bb5e3a7f17b1f74611b2561a3a27eeab05e5aa30f411572037`; the hub lists its size but no hash, because the file is not stored with Git LFS, so that hash is the one of the spike's copy and TASK62.7 checks it again against a fresh download. The spike also read `config.json` (612 bytes); the product loader (TASK62.6) does not: the shape is fixed in its code and checked against the tensors.

Reading the file into memory instead of mapping it needs 191 MB for all-MiniLM-L6-v2. The code is unoptimised: the linear layers read four weight rows per pass and run on all cores, nothing more (a first version needed 37 s for the library). The index of the library is made once and kept; after that a changed skill costs one embedding. With fewer cores (the same Apple M1, the number of processors Go may use limited):

| Cores | Embed 262 skills | One skill | Per query |
|---|---|---|---|
| 8 | 22 s | about 85 ms | 20 ms |
| 2 | 51 s | about 200 ms | 39 ms |
| 1 | 97 s | about 370 ms | 75 ms |

The cores of other processors are not measured; an Apple M1 core is fast. The Go code that runs the model is about 410 lines (`bert.go`, with the file loader) and the WordPiece tokenizer about 130 lines (`tokenizers.go`).

### In process, static models (a table of vectors, mean pooling)

Weights come from `huggingface.co/minishlab/<model>` under the MIT license; the loader needs `model.safetensors` and `tokenizer.json`. Fresh process, file mapped:

| Model | Download | SHA-256 of `model.safetensors` | Load | Memory loaded | Memory, library embedded (peak) | Embed 262 skills | Per query |
|---|---|---|---|---|---|---|---|
| potion-base-2M | 8 MB | `f95ffde02ad06f63ae38eb9d400038cd5ccaf8411ec3cb650c6025113f96cbb8` | not measured | | | | |
| potion-base-4M | 16 MB | `8a7140edd17ffab30ddcff1135eda127df57abfae856203dbd7b3d061295c31a` | not measured | | | | |
| potion-base-8M | 31 MB | `f65d0f325faadc1e121c319e2faa41170d3fa07d8c89abd48ca5358d9a223de2` | 16 ms | 11 MB | 29 MB (33 MB) | 13 ms | 10 µs |
| potion-base-32M | 131 MB | `99f6c33204c9231a7391871b7a3c91409b532c8f587a9ea44fc282303d8dec28` | 23 ms | 14 MB | 45 MB (50 MB) | 13 ms | 10 µs |
| potion-retrieval-32M | 131 MB | `07609e5bd33aad37900b3fd62f4ec96f6daec88ca4d46b9d8b928bfababf6ea0` | not measured | | | | |
| potion-multilingual-128M | 531 MB | `14b5eb39cb4ce5666da8ad1f3dc6be4346e9b2d601c073302fa0a31bf7943397` | 466 ms | 112 MB | 156 MB (162 MB) | 79 ms | 50 µs |

The 112 MB of the multilingual model right after loading are resident while its weights are only mapped, so that memory belongs to the tokenizer's table of 500,000 pieces.

### Runtime (Ollama 0.40.16, started on a private loopback port with its own home, so the machine's settings were not touched)

- Ollama itself: 186 MB archive, 772 MB unpacked, 41 MB of memory idle.
- `/v1/embeddings` answers as TASK62.6 needs. The models were used with the prefix or instruction that their model cards name (nomic-embed-text: `search_query:` and `search_document:`; embeddinggemma: `task: search result | query:` and `title: none | text:`; qwen3-embedding: an `Instruct:` line for the query; snowflake-arctic-embed: its query sentence). The effect of leaving them out was not measured.

| Model | On disk | Memory in use with the model loaded (server and runner) | First embedding after the model was unloaded | Embed 262 skills | Per query (median) |
|---|---|---|---|---|---|
| all-minilm | 45 MB | 183 MB | 0.33 s | 1.9 s | 7 ms |
| snowflake-arctic-embed 22M | 45 MB | 182 MB | 0.27 s | 1.9 s | 8 ms |
| granite-embedding 30M | 62 MB | 204 MB | 0.28 s | 1.9 s | 7 ms |
| nomic-embed-text | 274 MB | 395 MB | 0.28 s | 6.2 s | 15 ms |
| embeddinggemma | 621 MB | 751 MB | 0.81 s | 6.8 s | 16 ms |
| paraphrase-multilingual | 562 MB | 759 MB | 0.55 s | 5.0 s | 14 ms |
| qwen3-embedding 0.6B | 639 MB | 1230 MB | 0.58 s | 16.6 s | 40 ms |
| snowflake-arctic-embed 110M | 218 MB | 348 MB | not measured cleanly | | 14 ms |

A second run of the same server, made while another download was stalling, took far longer for the library and the first embedding; its times were discarded, and only its memory and per-query figures (which matched the first run) were used for the 110M model.

## Runs with Claude Code (item 1)

Claude Code 2.1.311, model claude-opus-5-5 in every cell, macOS. Each run is a headless `claude -p` in its own project folder with project-level settings (`--setting-sources project`, `--strict-mcp-config`, `--disable-slash-commands`), so no setting, hook or skill of the machine reaches the session. Tools allowed: the server's, `Read`, `Glob`, `Grep`. The library has 264 skills of four sources (224, 22, 15 and 1) and the two fixture skills of `spike/discovery/fixtures/`, all Searchable except where a cell says otherwise. The task names a made-up product: *"Write a short draft of the announcement for Fenwick's new recurring bookings feature, for our customer newsletter."* `spike/fenwick-announcements` is named after the product; `spike/tone-ladder` has no word of the task in its name or ID and can only be reached through its description (house voice for text that customers read). A skill is found as TASK62 defines it: `get_skill` of one of the two after a `search_skills` (cell h, where the skills are listed, counts a `get_skill` alone); no run read skill files from `~/.quiver`. The code is `serve.go` (the prototype server and the candidate wordings), `runs.py` and `analyze.py`; the per-run numbers are in `spike/discovery/results/runs-summary.json`.

| Cell | Engine, wording | Runs | Searched | Found | Of which fenwick / tone-ladder |
|---|---|---|---|---|---|
| a, primary (no skill listed) | ranked word search, wording 1 | 20 | 20 | **7** | 7 / 0 |
| a2, primary with changed wording | ranked word search, wording 2 | 20 | 20 | **20** | 20 / 0 |
| g, primary with `local` | all-MiniLM-L6-v2 fused with the word search, wording 1 | 20 | 20 | 20 | 20 / 0 |
| b, baseline (today's texts and server, skills not assigned) | matching of today | 20 | 0 | 0 | 0 / 0 |
| c, 16 skills listed, none fits | ranked word search, wording 1 | 20 | 20 | 10 | 5 / 5 |
| d, two turns | ranked word search, wording 1 | 20 | 20 | 0 | 0 / 0 |
| d2, two turns | ranked word search, wording 2 | 20 | 20 | 4 | 4 / 0 |
| e, primary in Turkish | ranked word search, wording 1 | 20 | 20 | 20 | 20 / 0 |
| f, control (a plain question about the project) | ranked word search, wording 1 | 20 | **0** | n/a | n/a |
| h, reference (both skills listed on demand) | n/a | 20 | 0 | 20 | 20 / 20 |

- **Threshold (16 of 20 in the primary cell).** Wording 1 with the ranked word search misses it (7 of 20). The assistant searched in every run, but its first query named no product (`customer newsletter announcement` in 18 runs, `customer newsletter feature announcement` in 2). The first returns nothing from the word search; the second returned the product skill. The skill was found in the 5 runs that searched again with the product name and in those 2. Wording 2 (the query names the product, tool or channel and what is produced; one more search from another angle if nothing fits) meets it with 20 of 20, every first query being `Fenwick customer newsletter announcement`. With wording 1, `local` meets it as well (20 of 20), because the model ranks the skill for the same query. Wording 2 is a changed wording run as a new cell, as TASK62.1 allows; both rows are kept and no threshold was changed.
- **Control.** The assistant searched in 0 of 20 runs of the plain question (limit 2). The baseline searched in 0 of 20 runs of the task, so without the cue it never reaches a skill that is not listed.
- **What the cells that inform say.** Listed skills are loaded without a search (h, 20 of 20, both). With 16 listed skills the cue still makes the assistant search in 20 of 20 runs and finds a fitting skill in 10; in the other 10 it takes a skill from the library that fits less well (`brand-voice-enforcement`, `brand-review`) or none. In Turkish the assistant writes the query in English and names the product (20 of 20). In the two-turn conversation the assistant's topic is the Slack message, not the product, and its queries are about Slack and status updates (`anthropics/stakeholder-update` and `anthropics/slack-messaging` are loaded instead); it searched in turn 2 in 5 of 20 runs of d and in turn 1 in all of them.
- **Limits.** One task, one made-up product, one model, 20 runs per cell. The second fitting skill (`tone-ladder`, reachable only through its description) was found by search in 5 runs of cell c and in none of a, a2, e or g, so the rates above are rates of finding the skill named after the product; the assistant names the product in its query once the wording asks it to, and no cell measures how well a skill is found when the prompt names nothing that the skill contains. Cell d does not test the second turn alone, because the topic of the first turn already differs from the skill. All cells ran with the throwaway server of the prototype, not the shipped build (TASK62.10 measures that again).
- **Spent.** 200 runs (160 planned for Claude Code plus 40 of the two cells with the changed wording): 2,180 input tokens, 1,497,404 cache-creation tokens, 20,484,618 cache-read tokens and 222,285 output tokens, which the CLI reports as USD 25.61. No retry was made.
- **Not measured.** Every assistant other than Claude Code (TASK68.2 runs the same cells and thresholds). The earlier spikes (docs/spike-mcp-clients.md) drove Claude Code, Codex, Cursor, VS Code and Antigravity by script; Claude Desktop cannot be driven by a script and is documented as not measured.

## Decisions by the rules fixed in TASK62.1, read on English queries

The rules count the three fitting kinds taken together, which include the Turkish paraphrase. They are applied here to the English queries only (paraphrases A and B and the assistant-style queries, 60 in the test half), because English is the language of the library and of the assistants' searches. This is a different reading of the rules than the one fixed in TASK62.1; that task lists every deviation from its plan.

1. **The ranked word search replaces the matching of today.** It finds the fitting skill for 44 of 60 queries against 10 of 60, and is higher on each of the three English kinds. Its share on the queries that fit nothing is one query lower (26 of 27 against 27 of 27); the rule asks for "not lower", and the rule's own text says that the rules are read together with the numbers. A gain of 34 points against one query.
2. **`local` ships in v1.6.** The best model is higher than the word search on all three English kinds (paraphrases A: 14 against 8; paraphrases B: 30 against 22; assistant style: 15 against 14, a difference of one query).
3. **The route.** Read as written, on the model alone, the best in-process model (gte-small, 53 of 60) reaches 90% of the best runtime model (59 of 60) only by rounding (89.8%), and all-MiniLM-L6-v2 reaches 86%; both are under the 90% that the rule asks. Read on what ships, the word search fused with the model, both reach 56 of 60 against 59 of 60 (95%). Memory (at most 500 MB) and download (at most 150 MB) are met by all-MiniLM-L6-v2 (84 MB, 91 MB) and gte-small (335 MB, 67 MB). The rule does not name the fused reading. The route of v1.6 is decided as in process with all-MiniLM-L6-v2, on English queries, after these measurements; rule 2 is therefore reported here and in the README and no longer decides the route. That is a change in how the rule is used, and it is noted here. The runtime route and the models of other languages move to backlog B20.
4. **Model.** In process: all-MiniLM-L6-v2. Fused with the word search it ties with gte-small (56 of 60); it uses 84 MB of memory against 335 MB, embeds the library twice as fast (22 s against 44 s) and has 6 layers against 12; gte-small has the smaller download (67 MB against 91 MB). The cutoff of the English run for all-MiniLM-L6-v2, which becomes its default `minScore`, is a cosine of 0.287 (the highest top score of the calibration queries that fit nothing). On the runtime route: embeddinggemma, which has the best rate (59 of 60; qwen3-embedding is equal and needs 1.2 GB) and uses 751 MB, under the 2 GB limit.
5. **Languages.** The models of the in-process route are English only. Turkish needs the runtime route (embeddinggemma 14 of 15; the best in-process model that reads Turkish, potion-multilingual-128M, 4 of 15 and 531 MB).

## Not measured, and limits

- Item 1: runs were made on Claude Code only (see above). The queries of the kind "assistant style" were written by hand from the skills' descriptions, not taken from runs, so they are probably easier than real ones. The queries that the assistants wrote in the runs are in `spike/discovery/results/runs-summary.json`; the query set has not been extended with them, so the numbers above are unchanged.
- Per-system setup steps and the need for administrator rights of Ollama and LM Studio, a running instance on a default port, and Windows and Linux numbers: not done. Only macOS on Apple silicon was run. The loader maps the file with a Unix call; Windows needs another way (or reads the file, at 191 MB for all-MiniLM-L6-v2).
- 45 paraphrases and 15 assistant-style queries in the English test half, one person labelled them, 224 of the 262 skills come from one source. A library of other skills may behave differently.
- Set B was added after the first comparison, which used set A only; the advantage of the static models over the word search is smaller on set B than on set A (see above), so the numbers of set B are the ones to rely on for English.
- Speed of the Go model on other processors, and the memory of several processes that search at once: not measured (only fewer cores of the same Apple M1 were).
