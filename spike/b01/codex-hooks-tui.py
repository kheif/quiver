#!/usr/bin/env python3
"""usage: codex-hooks-tui.py <trust|skip|folder|show|session|compact> <dir>

Runs one interactive Codex session in <dir> through a pseudo-terminal and
keeps its screen with pyte (pip install pyte). Codex asks the terminal for the
cursor position and colors at start and waits for the answers, so this script
gives them. Screens go to <dir>/screen-*.txt and the raw output to
<dir>/tui.raw.

  trust    answers "Hooks need review" with "Trust all and continue", then one prompt
  skip     answers it with "Continue without trusting", then one prompt
  folder   answers "Trust this folder?" and then "Hooks need review", then one prompt
  show     one prompt, for a session that shows no dialog
  session  a prompt, /clear, a prompt, /compact, a prompt
  compact  a prompt, /compact, a prompt

Writes <dir>/first-prompt.txt, which says whether a hook ran before the first
prompt. See docs/spike-hooks.md.
"""
import fcntl
import os
import pty
import re
import select
import struct
import sys
import termios
import time

import pyte

PROMPT = "Suggest three names for a {}."


class TUI:
    def __init__(self, argv, cwd, log, rows=50, cols=160):
        self.screen = pyte.Screen(cols, rows)
        self.stream = pyte.ByteStream(self.screen)
        self.raw = b""
        self.log = open(log, "wb")
        env = dict(os.environ, TERM="xterm-256color")
        pid, fd = pty.fork()
        if pid == 0:
            os.chdir(cwd)
            os.execvpe(argv[0], argv, env)
        fcntl.ioctl(fd, termios.TIOCSWINSZ, struct.pack("HHHH", rows, cols, 0, 0))
        self.pid, self.fd, self.alive = pid, fd, True

    def pump(self, secs):
        end = time.time() + secs
        while self.alive and time.time() < end:
            ready, _, _ = select.select([self.fd], [], [], 0.1)
            if not ready:
                continue
            try:
                data = os.read(self.fd, 65536)
            except OSError:
                data = b""
            if not data:
                self.alive = False
                break
            self.raw += data
            self.log.write(data)
            self.log.flush()
            self.stream.feed(data)
            if b"\x1b[6n" in data:
                c = self.screen.cursor
                self.send(f"\x1b[{c.y + 1};{c.x + 1}R")
            if b"\x1b[c" in data:
                self.send("\x1b[?62;22c")
            if b"\x1b]10;?" in data:
                self.send("\x1b]10;rgb:0000/0000/0000\x1b\\")
            if b"\x1b]11;?" in data:
                self.send("\x1b]11;rgb:ffff/ffff/ffff\x1b\\")

    def text(self):
        return "\n".join(line.rstrip() for line in self.screen.display)

    def wait(self, needle, secs):
        """Pumps until needle is on the screen; returns whether it appeared."""
        end = time.time() + secs
        while time.time() < end and self.alive and needle not in self.text():
            self.pump(0.5)
        return needle in self.text()

    def send(self, s):
        os.write(self.fd, s.encode())

    def type(self, s):
        self.send(s)
        self.pump(1)
        self.send("\r")

    def ask(self, prompt, secs=120):
        """Types a prompt and pumps until the reply's "Worked for" line appears."""
        done = self.raw.count(b"Worked for")
        self.type(prompt)
        end = time.time() + secs
        while time.time() < end and self.alive and self.raw.count(b"Worked for") == done:
            self.pump(0.5)
        self.pump(2)

    def save(self, d, name):
        open(f"{d}/screen-{name}.txt", "w").write(self.text() + "\n")
        # Inline mode scrolls finished output out of the screen, so keep
        # everything printed so far as text too.
        text = re.sub(rb"\x1b\[[0-9;?<>= ]*[A-Za-z]|\x1b\][^\x07\x1b]*(\x07|\x1b\\)|\x1b[()][A-Z0-9]|\x1b[=>78M]|\r", b"", self.raw)
        open(f"{d}/output-{name}.txt", "w").write(text.decode(errors="replace"))

    def quit(self):
        self.type("/quit")
        self.pump(3)
        try:
            os.kill(self.pid, 9)
        except ProcessLookupError:
            pass
        os.waitpid(self.pid, 0)


def choose(t, index):
    """Picks item <index> (0-based) of the dialog on the screen."""
    for _ in range(index):
        t.send("\x1b[B")
        t.pump(0.5)
    t.send("\r")
    t.pump(5)


def main():
    scenario, d = sys.argv[1], sys.argv[2]
    t = TUI(["codex", "--no-alt-screen", "-s", "read-only"], d, f"{d}/tui.raw")
    if scenario == "folder":
        t.wait("Trust this folder", 20)
        t.save(d, "folder")
        choose(t, 0)
    if scenario in ("trust", "skip", "folder"):
        t.wait("Hooks need review", 20)
        t.save(d, "review")
        choose(t, 1 if scenario != "skip" else 2)
    else:
        t.pump(10)
        if "Hooks need review" in t.text():
            t.save(d, "unexpected")
            t.quit()
            sys.exit(f"{d}: unexpected hook review")
    t.wait("Ask Codex", 10)
    t.pump(5)
    with open(f"{d}/first-prompt.txt", "w") as f:
        f.write(f"hook ran before the first prompt: {os.path.exists(d + '/hook.log')}\n")
    t.ask(PROMPT.format("cat"))
    if scenario == "session":
        t.type("/clear")
        t.pump(8)
        t.ask(PROMPT.format("dog"))
    if scenario in ("session", "compact"):
        t.type("/compact")
        t.pump(60)
        t.ask(PROMPT.format("fish"))
    t.save(d, "end")
    t.quit()


if __name__ == "__main__":
    main()
