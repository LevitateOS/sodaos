# Production function complexity

Complexity reports help locate code that is hard to understand. They are advisory,
not commit or CI gates. Prefer cohesive functions; do not split code merely to hit
a score or create exception paperwork. Refactor when it helps the current task.

## Running a report

Use an existing report if it answers the question. For a focused check:

```sh
bash scripts/check-complexity.sh --language rust -- lib/host/src/muse/execution.rs
bash scripts/check-complexity.sh --all --report
```

The helper builds its parser, not Soda. Go uses gocyclo, Rust uses syn, and
TypeScript uses its existing parser. Reports cover authored production source;
tests, generated files, dependencies and upstream copies are excluded. Scores
are navigation hints, not proof of correctness or a reason to block working code.
Existing baseline/exception files describe historical measurements and do not
need updating for ordinary development. Tests for the reporting tool remain
available through `bun run test:complexity`.
