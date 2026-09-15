# Python formatting, linting, and complexity

Pinned [Ruff](https://docs.astral.sh/ruff/) 0.16.6 is the Python analogue of
oxfmt/oxlint and of gofumpt plus gocyclo: one parser family for format, correctness
lint, and McCabe complexity. Do not add Black, isort, flake8, pylint, or radon on
top. Config lives in `pyproject.toml`; the install pin is `requirements-ruff.txt`.
Install with `python3 -m pip install -r requirements-ruff.txt`, or run via `uvx`
(`scripts/ruff-env.sh` will use `uvx ruff@0.16.6` when Ruff is not on `PATH`).

Whole-tree scripts live beside the Go and TypeScript gates; the pre-commit hook is
staged-only so an existing backlog cannot block unrelated commits. Go ownership
remains in [the slop audit](slop-audit.md#go-quality-gates-githookspre-commit-staged-scope-only);
TypeScript ownership remains in [the TypeScript guide](typescript.md#formatting-linting-and-complexity).

| Command | Scope |
| --- | --- |
| `bash scripts/check-ruff-format.sh` | Zero-tolerance format on the given `.py` paths, or every tracked Python file. Fix with `ruff format <files>`. |
| `bash scripts/check-ruff.sh` | Correctness lint (Ruff `E4`/`E7`/`E9`/`F`). Complexity is excluded here. |
| `bash scripts/check-py-complexity.sh` | Shipping Python only (`internal/`, `cmd/`, `appliance/`, `project-os/`). Cyclomatic complexity strictly below 10, matching shipping Go. `scripts/` and `tests/` are out of scope. |

`.githooks/pre-commit` runs those three on staged `.py` files after the TypeScript
checks. Formatting and lint apply to staged tests as well; complexity does not.

First measurement (Ruff 0.16.6): 42 of 45 tracked files fail formatting; 127
correctness findings, all in `tests/` and `scripts/` (shipping `internal/` is
clean); 8 shipping functions at cyclomatic 10+. No mass rewrite — the hook
ratchets staged files instead.
