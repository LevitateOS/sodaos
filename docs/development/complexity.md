# Production function complexity

## Integrated source debt — 18 October 2026

One scan of source `51ba6d2f` with the existing pinned analyzers found **58 Go functions above 20**, maximum **94** in `internal/factory/dispatch_prompt.go::BuildDispatchPrompt`; Rust and TypeScript have **zero above 20**, maxima 20 and 19. Preferred-target debt (>10) is Go 631, Rust 638, TypeScript 10. This is source complexity, not a behavior/quality acceptance score. The unchanged baseline remains historical; no allowance is regenerated. Exact rows are retained in the existing ignored `.artifacts/source-complexity/current-report.txt`.

Completed complexity batches, including `6bf6cecf` and subsequent focused repairs, are retained. The residual BROAD-COMPLEXITY-037 outcome is unassigned in the [canonical remaining record](factory-remaining-tasks.md#other-distinct-outcomes-and-shared-consumers).

Completed 18 October 2026: all 58 residuals split to ≤20 with behavior preserved (one small admission check per duty + one renderer per section; refusal order, error identities and lock/effect order unchanged). Slices: `BuildDispatchPrompt` 94→13 (`52b8a95f`); domain validators (`0797b90b`); store transactions (`65a77c3c`); intent adoption/reconcile/correction (`827b84be`); dispatch retry (`e7f66a71`); merge reconcile (`8cdcf54c`); intent delta (`7a60ee35`); delivery acceptance (`64d70dc0`); deciding drive (`635e3ff7`); task events (`4d8f3af8`); coder preparation (`ed46542f`); prompt stage (`861a5315`); correction admission (`2ce945c8`); priority/judgment/time-only (`e29e909e`); reassessment (`a58b6de0`); inputs/stall/recovery (`0a6c4045`); acceptance pair (`0e4e0b12`); MCP submission/repair grants (`d14cad13`); read/result/planning/retry (`b74df5d7`). Repo-wide gocyclo over-20 gate is clean; `internal/factory`, `internal/store`, `internal/archcheck` pass. Sandbox-only failures (PG-fixture lanes, unix-socket binds) are environmental and proven pre-existing on the untouched base; host PG proofs remain host-side.

Go, Rust and TypeScript share this policy: **prefer cyclomatic complexity ≤10;
normal maximum 20**. Above 20, a function needs a narrow, specific justification
and an explicit score cap (for example, an exhaustive protocol dispatcher whose
cases select existing handlers). Do not split functions mechanically to game the
score. Review cohesion and readability alongside the metric.

### Executable residual — BROAD-COMPLEXITY-037

Ready for one cohesive source slice after implementation is authorized. Use the
retained `current-report.txt` at `51ba6d2f` to select affected functions, then read
their current body/callers; do not rescan the repository just to choose work.
Its 58 Go residuals comprise 33 controller, 11 domain/prompt and 14 Store functions.
Begin with `internal/factory/dispatch_prompt.go::BuildDispatchPrompt` and
`dispatch_prompt_stage.go` if their ownership is free: separate actual prompt
sections/admission duties while preserving field order, bounds and refusal order.
An executor may instead take one coherent Store transaction or controller branch
whose callers are understood; do not combine unrelated functions merely to fill
a batch or split code mechanically to lower a score.

Keep lock/effect order, authority, error/wait classification and rollback behavior
unchanged. Check the changed functions with the existing complexity gate against
their real preimage and run the affected behavioral tests (real PG for changed
transaction/locking paths). New helpers must meet the normal cap. Existing
baseline and exceptions cannot be regenerated to accept an increase. Record the
remaining function identities in the existing report/change context as each
slice finishes; all 58 retain the same single outcome until <=20 or a substantive
narrow exception resolves each. Do not broaden this into the >10 preferred debt
or repeat accepted Rust/TypeScript reductions. No functional factory parent must
finish before a behavior-preserving simplification of an available owner.

Existing excess is recorded in `tools/source-complexity/baseline.json`, from its
exact `revision`. A changed function cannot exceed either that baseline or its
score in the comparison revision. Reductions therefore cannot be regained in a
later commit. New/renamed functions have no inherited allowance. Scores 11–20
are reported against the preferred target, not rejected. Unchanged historical
excess does not require a mass refactor.

## Running the gate

After `bun install --frozen-lockfile`, install the small Rust parser helper's
locked dependencies once:

```sh
cargo fetch --locked --manifest-path tools/source-complexity/rust/Cargo.toml
bash scripts/check-complexity.sh                    # working changes versus HEAD
bash scripts/check-complexity.sh --staged           # actual index, versus HEAD
bash scripts/check-complexity.sh --base origin/main # branch/worktree changes
bash scripts/check-complexity.sh --all --report     # source inventory and maxima
bash scripts/check-complexity.sh --language rust    # one language, changed files
bash scripts/check-ts-complexity.sh frontend/spaces/workspace.ts
bun run test:complexity
```

Explicit source-file paths can follow `--`. `bun run check:complexity` is the
shared command; the existing TypeScript command selects the same policy for TS.
The pre-commit hook reads source **and policy records from the index**. CI checks
all production files against the event's base commit (the previous commit for a
manual run), and its result is required by the aggregate CI job. Missing tools,
parse errors and invalid references fail closed. The helper builds only itself,
not Soda; Go uses the already pinned gocyclo dependency. Temporary output stays
under `.artifacts/source-complexity/`.

Baseline entries are historical debt, not waivers to enlarge functions. Do not
regenerate the baseline to bless a change. The maintenance-only
`bun tools/source-complexity/check.ts --snapshot <commit>` emits an initial
baseline for explicit review. Add exceptional new/increased complexity to
`tools/source-complexity/exceptions.json` under the exact `path::function` key,
with `max` and a substantive `reason`. Review that small policy diff with the
source. Caps are enforced; generic reasons such as “large function” are not
sufficient. Remove obsolete exceptions when simplifying.

## Scope and counting

The gate parses authored `.go`, `.rs`, `.ts` and `.tsx` files, including authored
operational/development tools. It excludes test/spec files, test/fixture/testdata
directories, declaration-only TS, vendor/dependency trees, generated files and
recognized generated headers, build output, and upstream provenance copies.
An application package named `build` remains production code. Rust inline
`#[cfg(test)]` modules, test functions and cfg expressions that provably require
`test` are excluded. It does not type-check or execute the application.

- **Go:** existing gocyclo over the Go AST. Starts at 1; adds `if`, `for`/`range`,
  non-default switch/select clauses and `&&`/`||`. Its established behavior
  includes nested function-literal decisions in the enclosing function and
  measures named top-level function literals. Build tags are not evaluated.
- **Rust:** `syn` source AST. Starts at 1; adds `if`, `for`, `while`, `?`, `&&`/`||`,
  non-wildcard match alternatives and match guards. Unconditional `loop` adds
  no decision. Closures contribute to their enclosing function; nested named
  functions are measured separately. Inline modules, impl methods and trait
  default methods are visited. Macros are not expanded or token-counted; macro-
  generated control flow needs review. Other cfg feature/target alternatives
  remain included, so this is not a binary-specific measurement.
- **TypeScript:** the already locked TypeScript 5.9 compiler API used by the Lit
  checker; the root TS7 compiler remains responsible for application type checks.
  Starts at 1; adds conditionals, loops, non-default cases, catches, ternaries,
  logical/nullish operators and their assignment forms, parameter/destructuring
  defaults, and optional-chain segments. Nested functions are separate; named
  functions, arrows, methods, accessors and constructors are measured. Decorator
  or transpiler expansion is not included.

These are documented source-level conventions, not interchangeable measurements
of generated machine code. Conditional duplicate function names receive stable
occurrence suffixes; anonymous functions use enclosing names and occurrence order.
Moving/renaming them can require reviewing a new identity rather than inheriting
an unrelated baseline entry. No regex approximation is used to count branches.

Initial measured maxima (historical baseline, not desired limits): Go **117**
(`internal/factory/intent.go`, `IntentRevision.Validate`); Rust **90**
(`lib/host/src/dbackend/websocket.rs`, `pump_attached_admitted`); TypeScript **60**
(`scripts/screenshot.ts`, `main`). Test, vendor and generated code is excluded.
