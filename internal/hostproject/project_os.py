"""Fixed read-only OS observation inside an existing running project.

Never source os-release as shell code or dump arbitrary files/environment.
"""
import json
import os
import re
import shlex
import stat


def observe(path="/etc/os-release"):
    fd = os.open(path, os.O_RDONLY | os.O_NONBLOCK | os.O_CLOEXEC)
    try:
        if not stat.S_ISREG(os.fstat(fd).st_mode):
            raise ValueError("not a regular OS release file")
        raw = os.read(fd, 4097)
        if len(raw) > 4096:
            raise ValueError("oversized OS release file")
    finally:
        os.close(fd)
    values = {}
    for line in raw.decode("utf-8", errors="strict").splitlines():
        if not line.strip() or line.lstrip().startswith("#"):
            continue
        key, sep, value = line.partition("=")
        if key not in ("ID", "VERSION_ID", "PRETTY_NAME"):
            continue
        if not sep or key in values:
            raise ValueError("ambiguous OS release field")
        words = shlex.split(value, comments=False, posix=True)
        if len(words) != 1 or not words[0] or len(words[0].encode("utf-8")) > 256:
            raise ValueError("invalid OS release field")
        if any(ord(c) < 32 or ord(c) == 127 for c in words[0]):
            raise ValueError("invalid OS release text")
        values[key] = words[0]
    if not re.fullmatch(r"[a-z0-9][a-z0-9._-]{0,63}", values.get("ID", "")):
        raise ValueError("missing OS identity")
    if not re.fullmatch(r"[a-zA-Z0-9][a-zA-Z0-9._-]{0,63}", values.get("VERSION_ID", "")):
        raise ValueError("missing OS version")
    return {"id": values["ID"], "version": values["VERSION_ID"], "name": values.get("PRETTY_NAME", values["ID"])}


if __name__ == "__main__":
    try:
        print(json.dumps(observe(), ensure_ascii=True))
    except (OSError, UnicodeError, ValueError):
        raise SystemExit("OS observation unavailable")
