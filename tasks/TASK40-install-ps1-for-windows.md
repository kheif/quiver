# TASK40 — `install.ps1` for Windows

**Phase:** Phase 4c — v1.2

**Status:** Todo

**Depends on:** TASK33

1. `install.ps1`, used as `irm https://raw.githubusercontent.com/kheif/quiver/main/install.ps1 | iex`: the same steps as `install.sh` for the Windows zip (`quiver_windows_<arch>.zip`), with `QUIVER_VERSION`, `QUIVER_INSTALL_DIR` (default `%LOCALAPPDATA%\Programs\quiver`) and `QUIVER_DOWNLOAD_URL`.
2. Check the SHA-256 against `checksums.txt`, and add the folder to the user's `PATH` (user scope, no admin).
3. CI's `release-dry-run` and the release workflow's install checks run it on Windows.
4. README: the Windows install line; `install.sh`'s Windows error mentions it.

**Done when:** a clean Windows runner installs with the script and `quiver --version` works in a new shell.
