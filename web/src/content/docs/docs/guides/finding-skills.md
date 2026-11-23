---
title: Finding skills
description: Keep skills out of the session-start list and let the assistant find them by searching, with the Searchable level.
sidebar:
  badge:
    text: Next
    variant: default
---

:::note[Next release]
The Searchable level is on the main branch and ships after v1.5.0.
:::

Listed skills cost tokens in every session: each one adds its ID and description to the text the assistant starts with. A skill that is needed only when its topic comes up, such as a style guide for Slack messages or a guide to writing like a person, should cost nothing until then. Make it **Searchable**.

## The four levels

Each scope, every folder or one project, holds a skill at one of four levels:

| Level | At the start of a session | Set with |
| --- | --- | --- |
| Not assigned | Nothing | `quiver remove <id>` |
| Searchable | Nothing: the assistant finds it by searching | `quiver add <id> --searchable` |
| On demand | Its ID and description are listed; the assistant loads it when a task fits | `quiver add <id>` |
| Always on | Loaded in full | `quiver add <id> --always` |

A skill moves between them: `--searchable` takes it off the listed skills of that scope, always-on mark included, and `quiver add` without the flag takes it off the Searchable ones. Across the global and the project scope, the level in a folder is the most served of the two. The [panel](/docs/guides/panel/) sets the same four levels in the detail of a skill.

## How the assistant finds a skill

A Searchable skill is not an MCP prompt and is not in `list_skills`. The assistant calls `search_skills` with a query that names what it needs, and gets the ID and description of the best matches, not their text. If one fits, it calls `get_skill` and follows it. It does not ask you first: assigning a skill is your statement that you trust it, which is why you should read a skill with `quiver show <id>` before you make it Searchable, and why the assistant cannot change one through `update_skill`.

With the default engine, search is ranked by words (the `local` engine adds meaning, below). A name that is, starts with or contains your text comes first; after that, skills whose name and description fit the words of the text, best first. A query that fits nothing returns nothing. `search_skills` searches only the Searchable skills of the folder and returns at most 10, so a few skills are never crowded out by the rest of the library; `quiver search` and the panel search the whole library and show the level of each match.

## Make skills Searchable

```sh
quiver add slack-style --searchable             # in this project
quiver add slack-style --searchable --global    # in every folder
quiver add --source anthropic --searchable --global
```

`--source <name>` makes every skill of a source Searchable that is not assigned yet in that scope. A skill that is listed or always on there keeps its level. `--searchable` cannot be combined with `--always`.

When you add a source, Quiver asks once, in a terminal, whether to make its skills Searchable in every folder. The default is no, and `--searchable` answers yes:

```sh
quiver source add anthropics/skills --name anthropic --searchable
```

The output prints the command that undoes it, `quiver remove --source anthropic --global`. `quiver remove <id>` takes a skill off every level of the scope, and `--source <name>` does it for a whole source.

## See where a skill is Searchable

```sh
quiver list         # the ASSIGNED column shows searchable (global, 2 projects)
quiver where slack-style
quiver status       # the Searchable skills of this folder
```

`quiver list --unassigned` leaves Searchable skills out, because they are assigned. `quiver update` previews changes to a Searchable skill like those of any assigned skill, and `quiver sync` lists skills that another machine made Searchable before it pulls them.

## What a listing costs

Every listed skill adds a line (its ID and description) to the text that starts each session. `quiver status` says how much: how many skills are listed, the size of the listing in characters and in estimated tokens (characters divided by four), the always-on skills counted apart because their whole text is loaded, and how many skills are Searchable, which add nothing. `quiver status -v` lists the ten listed skills with the longest descriptions, each with the command that makes it Searchable. The panel's Skills header shows the same count for the project it manages, and the detail of a Searchable skill says that it costs nothing in the listing. The numbers inform; Quiver never refuses an assignment or trims the listing.

When the library has skills that are at no level in a folder and the folder has no Searchable skill, `quiver status` adds a line: `search_skills` finds only Searchable skills, where v1.5 searched the whole library, so those skills are not found. It names the source with the most of them and the command that makes them Searchable. The line goes away with the first Searchable skill.

## The search engine

`quiver engine` shows the engine that `quiver search`, `search_skills` and the panel use, and `quiver engine use <engine>` writes the choice to `state/search.json`, a file of this machine that is never synced. `quiver search --engine words <text>` uses an engine for one search only.

- **`words`** is the default. It ranks skills by the words of their names and descriptions.
- **`local`** adds meaning: "send a note to the team channel" finds the Slack skill, though the skill shares no word with the query. A small model, all-MiniLM-L6-v2, runs inside the Quiver process, with no server and no network call. It understands English only; a query in another language gets the `words` matches. It needs the model file, which `quiver engine setup` downloads. On the English test queries of the discovery spike, a fitting skill was among the first five matches for 55 of 60 queries, against 44 of 60 for `words`.

### Setting up `local`

`quiver engine setup` does it in one step. It first says what will happen and asks once, with the default "no": the model (all-MiniLM-L6-v2, English only), the download of 91 MB, the disk space (the file and an index of about 1 MB), the memory a search uses while the model is loaded (up to about 85 MB, mostly the mapped weight file that the system shares between processes), and that the download of the model file is the only thing that leaves the machine. Then it downloads the file, builds the index and checks that the vector search finds one of your skills among the first five; only then does it switch the engine to `local`. A failed step leaves the engine at `words` and says which step failed. `--yes` answers the question; without a terminal and without `--yes` nothing changes and the command is printed.

After the first time you make a skill Searchable in a terminal, Quiver offers the setup once, and never asks again. `quiver status` and `quiver doctor` point to it while Searchable skills exist and the engine is `words`. Nothing starts a setup without your yes.

- **The download.** Two files, `model.safetensors` and `tokenizer.json`, come over HTTPS from huggingface.co at one fixed revision. Each is checked against its pinned size and SHA-256 before it is moved into `~/.quiver/cache/models/all-minilm-l6-v2/`; a wrong hash, a cut download or a file larger than announced is deleted. Redirects are followed only over HTTPS, at most three. No program is installed or run.
- **By hand.** If the address is blocked or you use a company mirror, put the two files in that folder yourself. `quiver engine setup` accepts them when their hash matches the pinned one and skips the download; a file with another hash is refused, and the message names the pinned hash.
- **If the address changes** or the file is removed there, the setup stops with the reason, the engine stays `words`, and searching keeps working.
- **Undo.** `quiver engine use words` switches back and keeps everything. `quiver engine remove` deletes the index and, after asking, the model files that Quiver downloaded. A file you put there is never deleted.
- **Repair.** `quiver doctor` reports each link: the model files and their hashes, the index. In a terminal, `quiver doctor --fix` repeats the missing step with the same question. The panel's `Fix` never starts a download; it points to the command line.

The first search after you add or change skills embeds them, about 0.1 s a skill. `quiver search` builds the missing part for up to a minute and then answers with what it has; the server and the panel build in the background and answer with `words` meanwhile. `quiver engine reindex` rebuilds all of it, and `quiver doctor` shows the model file and how many skills are indexed.

A missing or unusable settings file or model file never stops a search: it runs as `words`, and `quiver engine`, `quiver status` and `quiver doctor` show what is wrong. See [the files of the Quiver home](/docs/reference/files/).
