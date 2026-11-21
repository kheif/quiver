# TASK62: Finding unassigned skills

**Phase:** Phase 7b: Skill discovery (v1.6)

**Status:** 🚧 In progress

**Depends on:** TASK61

Today a skill is *Not assigned*, *On demand* or *Always on*, in each of two scopes (every folder, one project). Always-on skills are delivered in full at the start of a session. On-demand skills are listed with ID and description, so the assistant knows what exists. That listing costs tokens for every skill, in every session: as session-start lines (`- id: description`), the 261 skills of a library measured on 2026-11-19 come to about 100 KB, roughly 25k tokens by the usual four characters per token (an estimate, not a token count), and even IDs alone would grow without limit with the library. A skill that is only needed when its topic comes up (a style guide for Slack messages, a guide to writing like a person) should cost nothing until then. `search_skills` could serve that, but it matches only skills that contain every word of the query in their name, ID or description, it searches skills the user never chose, and the only cue to call it is one line in the server instructions ("search_skills finds skills by words", `usage` in `internal/server/catalog.go`); the session-start hook never mentions it and several assistants show the instructions late or not at all (docs/spike-mcp-clients.md).

This task adds a level between *Not assigned* and *On demand*:

| Level | What the session gets | When the skill reaches the assistant |
|---|---|---|
| Always on | the full text | from the first message |
| On demand | ID and description | when the assistant decides to load it |
| **Searchable** (new) | one short cue that searchable skills exist, nothing about the skills | when the assistant searches and finds it, and loads it without asking |
| Not assigned | nothing | never by itself; found only in the CLI and the panel |

- **The level** (TASK62.2, TASK62.3, TASK62.4, TASK62.8): a skill the user made searchable, or that the user created or imported themselves, is Searchable in a scope. Skills of a source added with `quiver source add` stay *Not assigned* until the user says yes when adding it.
- **Search** (TASK62.5, TASK62.6, TASK62.7): `search_skills` searches the Searchable skills of the folder, ranks matches better, and can find a skill by meaning. The `words` engine becomes a ranked word search. The `local` engine uses a small embedding model that Quiver runs itself, inside its own process, in plain Go: all-MiniLM-L6-v2, a file of 91 MB that Quiver downloads once after the user says yes (TASK62.6, TASK62.7). It needs no runtime, no server and no background process, and understands English only. The user's own models, models of other languages and a server on a loopback address come later (backlog B20). The choice is kept per machine in `~/.quiver/state/search.json` (never synced; see TASK62.5). `quiver search` and the panel keep searching the whole library.
- **Cue** (TASK62.8): the session-start text tells the assistant that searchable skills exist and when to search: when the topic of the conversation calls for something a skill could cover, not on every message. The assistant forms the query from the conversation, which is why the search is not run on every message by Quiver: Quiver sees only the message, and the assistant knows the topic. A simple question about the project needs no search.

The catalog of listed skills is still delivered in full; nothing here caps it. Documentation on the website follows `web/README.md`: what is not released carries the "Next release" marks, and the pages are committed only after the maintainer confirms. Out of scope: embedding services on another machine and a vector store on another machine, and in this release any embedding model other than the one above. The `engine` setting is built so that another engine can be added later.

**A skill is found** in a run when the assistant loads the fitting Searchable skill with `get_skill` after a search, with no word in the prompt naming Quiver or the skill, and after the server instructions and the session-start text as shipped. A run in which the assistant reads the skill's files straight from `~/.quiver` is not found. TASK62.1 measures this for each assistant; the threshold is fixed in that task, before any run.

**Done when:** every subtask is done; the rate of finding the fitting skill in a library of 250+ skills is recorded on Claude Code, the only assistant measured for v1.6, and the threshold is met there; every other assistant is documented as not yet measured for finding Searchable skills until TASK68.2 measures it; the threshold is met on Claude Code on the spike's prototypes and again on the shipped build (TASK62.10). If it is not, the phase is planned again before anything ships. An assistant below the threshold is documented as not supported for finding Searchable skills. Claude Desktop cannot be driven by a script, so it is not measured: it drops the server instructions, defers the tools and needs the user to name Quiver, and it is documented so.

**Subtasks:**
- [TASK62.1](TASK62.1-discovery-spike.md): Discovery spike
- [TASK62.2](TASK62.2-searchable-level-in-the-registry.md): Searchable level in the registry
- [TASK62.3](TASK62.3-assigning-the-searchable-level.md): Assigning the Searchable level
- [TASK62.4](TASK62.4-searchable-delivery-readers-and-panel.md): Searchable skills in delivery, readers and the panel
- [TASK62.5](TASK62.5-search-settings-and-word-search.md): Search settings and word search
- [TASK62.6](TASK62.6-local-vector-search.md): Local vector search
- [TASK62.7](TASK62.7-guided-local-model-setup.md): Guided setup of the local model
- [TASK62.8](TASK62.8-search-cue-and-instructions.md): Search cue and default level of new skills
- [TASK62.9](TASK62.9-catalog-cost.md): Catalog cost
- [TASK62.10](TASK62.10-release-v1-6-0.md): Release v1.6.0
