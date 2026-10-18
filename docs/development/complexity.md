# Production function complexity

Go, Rust and TypeScript share this policy: **prefer cyclomatic complexity ≤10;
normal maximum 20**. Above 20, a function needs a narrow, specific justification
and an explicit score cap (for example, an exhaustive protocol dispatcher whose
cases select existing handlers). Do not split functions mechanically to game the
score. Review cohesion and readability alongside the metric.

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
