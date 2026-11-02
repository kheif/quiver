---
title: Layers
description: The layers a folder gets its skills from, from the project up to the company, and which ones work today.
---

Every folder gets the skills of its layers. A layer adds to the ones above it, each a step closer to the work, so nobody copies skills between projects or teams.

| Layer | Whose skills | Served in | Today |
| --- | --- | --- | --- |
| Company | Set once for every developer | Every folder of everyone in the company | Planned, with Quiver for teams |
| Team | Set for one team, such as UI or backend | Every folder of that team's members | Planned, with Quiver for teams |
| You | Yours | Every folder you work in | Yes: global skills |
| Project | One codebase's | The folders of that project | Yes: project skills |

## The layers that work today

The **you** layer is your global skills, and the **project** layer is a project's skills. You put a skill on either with an assignment:

```sh
quiver add commit-style --global   # you: every folder
quiver add release-notes           # the project of this folder
```

A folder that is not a project gets only the global skills. See [Assignments](/docs/concepts/assignments/) for the commands, and [Projects](/docs/concepts/projects/) for how a folder becomes a project.

Assignments live in your `~/.quiver`, not in the project. A teammate who clones the project gets its `.quiver.json`, but not your assignments; to share skills today, put them in a git repository that everyone adds as a [source](/docs/guides/sources/).

## The team and company layers

Quiver for teams adds two layers above yours: one for the whole company, and one per team. They are managed in one place, so a security review set by the company, or the UI team's design tokens, reach every developer they apply to without anyone copying them. Each team sees only its own skills. See [Quiver for teams](/teams/) for the plan.

## Always on, in any layer

A skill on any layer can also be always on: it is in the conversation before the first reply, instead of waiting for the assistant to load it. See [Always-on skills](/docs/concepts/always-on/).
