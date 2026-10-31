# TASK40 — `install.ps1` for Windows

**Phase:** Phase 4c — v1.2

**Status:** ✅ Done

**Depends on:** TASK33

1. `install.ps1`, used as `irm https://raw.githubusercontent.com/kheif/quiver/main/install.ps1 | iex`: the same steps as `install.sh` for the Windows zip (`quiver_windows_<arch>.zip`), with `QUIVER_VERSION`, `QUIVER_INSTALL_DIR` (default `%LOCALAPPDATA%\Programs\quiver`) and `QUIVER_DOWNLOAD_URL`.
2. Check the SHA-256 against `checksums.txt`, and add the folder to the user's `PATH` (user scope, no admin).
3. CI's `release-dry-run` and the release workflow's install checks run it on Windows.
4. README: the Windows install line; `install.sh`'s Windows error mentions it.

**Done when:** a clean Windows runner installs with the script and `quiver --version` works in a new shell.

**Notes:**
- `irm | iex` runs the script in the caller's shell, so everything is in one `& { }` block and errors `throw`: `exit` would close the shell. It works in Windows PowerShell 5.1 (TLS 1.2 is turned on, the progress bar off) and in PowerShell 7.
- `QUIVER_DOWNLOAD_URL` also takes a local folder or a `file://` URL, which are copied, because `Invoke-WebRequest` does not read files; CI installs from `dist/` this way.
- A running `quiver.exe` cannot be overwritten but can be renamed: the old one becomes `quiver.exe.old` and is deleted on the next install.
- The `PATH` is changed in `HKCU:\Environment` with the raw value and its registry type, so entries such as `%USERPROFILE%in` stay unexpanded; setting and clearing a variable through `[Environment]` then tells Explorer, so new shells get the new `PATH`.
- CI's `release-dry-run-windows` installs the snapshot archive in Windows PowerShell 5.1, again over it in PowerShell 7, and runs `quiver --version` with the `PATH` read from the registry, as a new shell gets it; a step keeps the runner's `PATH`. The release workflow's `verify-powershell` does the same with `irm | iex` from the tag.
- The new-version line on Windows is the `irm` line, with `$env:QUIVER_INSTALL_DIR` when `quiver.exe` is not in its default folder.
