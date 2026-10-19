# Working on SodaOS

The owner's current instructions lead. Project guidance constrains authorized work;
it cannot grant permission. Distinguish evidence, recommendations and owner decisions.
Challenge weak assumptions with concrete reasons; state uncertainty honestly.

## Leading execution policy: anti-bureaucracy

These rules govern Codex and Muse and override conflicting procedural guidance.
Preserve correctness, permissions, evaluation order, atomicity and failure/recovery behavior.

- Development speed is the priority. SodaOS has zero users. Implement the smallest
  useful change, run the smallest check that establishes it works, and keep moving.
  Every procedural step must help deliver the current task or prevent a concrete
  failure; skip steps that merely satisfy process. Do not add reproducibility,
  release-process or compatibility work for hypothetical future users. Stale plans
  and procedural docs must not block authorized implementation or testing.
- Own the full authorized outcome. No repeated approvals, narrow batch gates or idle
  stops while unblocked work remains. Review asynchronously; continue independent work.
- Never look at the version number. If you have to look at a version number, you are
  already harming the project. Do not inspect, compare, pin or gate work on tool or
  dependency version numbers. Test the installed tool's required behavior directly;
  resolve actual failures and update stale guidance from observed behavior. A version
  mismatch must never block testing, force a downgrade or invalidate working evidence.
- Reuse valid evidence. No duplicate checks, reviews of reviews or reopened criteria
  without relevant changes, actual failures or a specific unresolved risk.
- No extra plans, evidence/status files, bookkeeping commits or measurement frameworks
  without concrete need. Update an existing record when useful; ordinary work does not
  require an execution record. Report meaningful changes concisely.
- Keep process artifacts out of the repository, including ignored directories:
  no JEV requests/responses/receipts, consultation transcripts, review ledgers,
  audit inventories or execution-status reports. Summarize useful conclusions in
  chat; record only resulting technical decisions in the existing owning docs.
  If a tool needs temporary consultation files, use scratch outside the checkout
  under `/home` and remove them when finished.
- Coordinate actual overlaps briefly and continue independent work. Do not turn
  coordination into a timed ceremony or a prerequisite for unrelated changes.
- Resolve authorized setup yourself. If genuinely blocked, explain the concrete obstacle
  and what is needed to proceed; continue everything that does not depend on it.
- Distinguish requested, sent, acknowledged, executing and verified. Separate partial
  work, source checks and full acceptance. Never infer execution from dispatch or invent
  completion percentages. Save coherent completed work at safe boundaries.

## Scope and protection

- Work in the canonical `~/Projects/sodaos` checkout; no worktrees unless requested.
  Inspect changes in files you will touch. Preserve unrelated work; coordinate actual
  overlaps without stopping independent work. Leave the predecessor/Updates platform alone.
- Implementation includes matching code, tests, docs and operational configuration.
  Installation/deployment requests include necessary builds, transfers, scoped restarts
  and verification. Read-only review does not authorize mutation. Do not infer permission
  to publish, push, open PRs, merge, contact others or perform unrelated destructive work.
  Never amend or rewrite history without permission.
- **All writes, builds, caches, state and scratch belong under `/home`. The root disk
  is off-limits, including as a fallback.** Approved privilege use does not change this.
- Protect credentials and explicitly preserved data. Use restricted secret files; never
  expose secrets in argv, logs, screenshots or evidence, or request private SSH keys.
  Back up protected state consistently; never restore over newer protected writes or
  replay mutations to repair an observer. Use neutral fixture identities, never the
  owner's personal information or identities derived from local accounts or paths.
- Clean only identified task-owned resources. Before release cleanup, inspect the active
  `request.json` and [release reservations](docs/development/release.md#guided-release-operator-flow).
  Retain reserved outputs/assets/caches until promotion or explicit abandonment; idle
  resources are not automatically disposable. Preserve canonical assets, licenses and
  attribution; put generated outputs in ignored `.artifacts/` and keep private inputs untracked.

## Engineering

- This is unreleased software: maintain one current implementation. Replace obsolete
  formats, callers and fixtures together; no compatibility layers or migrations unless
  explicitly required. Removals should reduce code, not add replacement scaffolding.
- Solve demonstrated needs with mature upstream implementations. Verify actual behavior
  and the actual caller before adding adapters or claiming limitations. Avoid
  speculative recovery systems, harnesses and abstractions.
- Token and build budgets matter. Test a critical assumption before investing in work
  that depends on it. Choose the cheapest sufficient check and move on once it answers
  the question. A failed assumption changes the affected approach, not the whole plan.
- Preserve [language and package ownership](docs/development/ownership.md):
  network servers use Go, system/privileged applications Rust, no authored Python programs.
  Planned cutovers are not completed cutovers or permission to port. Resolve routine
  implementation choices directly from the owner's instructions and existing code;
  no register entry or separate approval is needed.
- Preserve dependency direction and canonical definitions. No retired packages, forwarding
  packages, duplicate domain DTOs or forbidden cross-boundary imports. Put new code in
  the appropriate existing package; update guidance or architecture assertions only
  when the change actually affects them.
- Keep code cohesive and readable. Use [complexity reports](docs/development/complexity.md)
  when they help a concrete refactor; scores do not block work or require exception
  paperwork. Avoid unrelated dependency changes.

## Verification and documentation

Test changed behavior, demonstrated failures and relevant permission/destructive boundaries
with existing focused checks. Keep callers, browser assets and staging connected. Required
combined/final acceptance addresses genuine remaining criteria, not ritual reruns.
Reuse artifacts and passing checks when they still test the relevant behavior. A failed
run requires fixing and retesting the actual failure, not discarding unrelated evidence.
Claim release qualification only when its relevant checks pass.
Do not rebuild shipping bytes for driver-only debugging without a concrete need.
SodaOS targets native x86_64; cross-compilation/emulation is not native or installed proof.

Consult [docs/README.md](docs/README.md) when you need to locate guidance; read only what
helps the task. Correct docs made misleading by your change and link rather than
duplicate. Do not turn implementation into a plan audit, status reconciliation or
documentation cleanup unless requested or necessary to complete the task.

Use [testing](docs/development/testing.md), [native support](docs/development/native-support.md)
and [installation](docs/guides/installation.md) for execution commands and effects;
[Go](docs/development/go.md), [TypeScript](docs/development/typescript.md) and
`internal/archcheck` for language/boundary checks. Before changing real state, understand
the affected resources and recovery path; consult the operational guide where needed.

Only when explicitly requested, consult Jev/SystemOne with complete supplied context:
it cannot research. Make three fresh, semantically equivalent consultations with rewritten
prose and unchanged technical facts; investigate disagreements and treat agreement as
advice rather than proof. Keep the three consultations; do not create receipts
or a wording/equivalence report. This grants no external-call permission.
