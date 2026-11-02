---
title: Environment variables
description: Environment variables that change how quiver and its install scripts behave.
---

## quiver

| Variable | Effect |
| --- | --- |
| `QUIVER_HOME` | The Quiver home. Default: `~/.quiver`. |
| `QUIVER_PROJECT` | The project folder the MCP server and `quiver status` serve, instead of the current folder. |
| `QUIVER_NO_UPDATE_CHECK` | Set to any value to turn off the daily check for a new release. |
| `VISUAL`, `EDITOR` | The editor that `quiver new` and `quiver edit` open, `VISUAL` first. |
| `NO_COLOR` | Set to any value to print without colors. |
| `CLAUDE_CONFIG_DIR` | Where Claude Code keeps its config, when not `~/.claude`. Quiver looks there for copies of served skills. |

## Install scripts

`install.sh` and `install.ps1` read:

| Variable | Effect |
| --- | --- |
| `QUIVER_VERSION` | The release to install, such as `v1.2.0`. Default: the latest. |
| `QUIVER_INSTALL_DIR` | Where to put the binary. Default: `~/.local/bin`, or `%LOCALAPPDATA%\Programs\quiver` on Windows. |
| `QUIVER_DOWNLOAD_URL` | A folder with the archives and `checksums.txt`, for mirrors and tests. Default: the GitHub release. |
