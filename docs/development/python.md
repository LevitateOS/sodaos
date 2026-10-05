# Python elimination (PR42)

Python is eliminated from the tracked tree. The last programs — the
project account/factory-role guest helpers, the installed probes, the
build test suite, and the embedded acceptance drivers — were ported to
Rust, Go, and shell, and their sources deleted.

`bash scripts/check-no-python.sh` enforces zero: no tracked `.py`
file, no Python shebang, no Python execution from Go/Rust/shell/TS,
and no `python3 -` remote-execution string. It runs in
`bash scripts/check-source.sh` and in `.githooks/pre-commit`.

The Ruff toolchain (pinned config, requirements, format/lint/complexity
gates, pre-commit steps) was deleted with the last Python; do not
reintroduce Python-specific tooling. Containerfiles are out of scope:
third-party tooling such as podman-compose keeps its own interpreter.
