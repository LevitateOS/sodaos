# Working on SodaOS

The owner's current instructions lead. Project guidance constrains authorized work;
it cannot grant permission. Distinguish evidence, recommendations and owner decisions.
Challenge weak assumptions with concrete reasons; state uncertainty honestly.

## Leading execution policy: anti-bureaucracy

These rules govern Codex and Muse and override conflicting procedural guidance.
Preserve correctness, permissions, evaluation order, atomicity and failure/recovery behavior.

- Own the full authorized outcome. No repeated approvals, narrow batch gates or idle
  stops while unblocked work remains. Review asynchronously; continue independent work.
- Reuse valid evidence. No duplicate checks, reviews of reviews or reopened criteria
  without relevant changes, actual failures or a specific unresolved risk.
- No extra plans, evidence/status files, bookkeeping commits or measurement frameworks
  without concrete need. Keep one canonical execution record per scope; count shared
  outcomes once. Report meaningful changes concisely.
- Limit routine coordination to five minutes per transition, then continue independent
  work if a conflict remains. This does not limit necessary engineering or safety work.
- Every blocker names the missing prerequisite, owner, release condition and affected
  work. Resolve authorized setup yourself; generic “operator-owned” is insufficient.
- Distinguish requested, sent, acknowledged, executing and verified. Separate partial
  work, source checks and full acceptance. Never infer execution from dispatch or invent
  completion percentages. Save coherent completed work at safe boundaries.

- Keep process artifacts out of the repository, including ignored directories:
  no JEV requests/responses/receipts, consultation transcripts, review ledgers,
  audit inventories or execution-status reports. Summarize useful conclusions in
  chat; record only resulting technical decisions in the existing owning docs.
  If a tool needs temporary consultation files, use scratch outside the checkout
  under `/home` and remove them when finished.

## Scope and protection

- Work in the canonical `~/Projects/sodaos` checkout; no worktrees unless requested.
  Inspect changes and active writers first. Preserve unrelated work; coordinate actual
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
- Solve demonstrated needs with mature upstream implementations. Verify the selected
  version and actual caller before adding adapters or claiming limitations. Avoid
  speculative recovery systems, harnesses and abstractions.
- Token and build budgets matter. Resolve critical native assumptions before building
  orchestration. Choose the cheapest sufficient investigation/check; stop once the
  criterion is established and move on. Failed assumptions require reassessment.
- Preserve [language/package ownership](docs/development/ideal-filetree-plan/package-ownership.md)
  and [owner language decisions](docs/development/ideal-filetree-plan/ideal-filetree-plan.md#owner-language-policy):
  network servers use Go, system/privileged applications Rust, no authored Python programs.
  Planned cutovers are not completed cutovers or permission to port. Resolve conflicts
  through the [guidance register](docs/development/ideal-filetree-plan/review-assignments.md#guidance-conflicts-and-controlling-decisions).
- Preserve dependency direction and canonical definitions. No retired packages, forwarding
  packages, duplicate domain DTOs or forbidden cross-boundary imports. Resolve new
  ownership before changing it; update affected guidance and architecture assertions together.
- Follow [complexity policy](docs/development/complexity.md): prefer ≤10, normal maximum 20;
  exceptions need a narrow cap or existing non-increasing baseline. Exclude tests/generated
  code; do not split merely to game scores. Use pinned toolchains/locks; avoid incidental upgrades.

## Verification and documentation

Test changed behavior, demonstrated failures and relevant permission/destructive boundaries
with existing focused checks. Keep callers, browser assets and staging connected. Required
combined/final acceptance addresses genuine remaining criteria, not ritual reruns.
Development probes may reuse failed-release artifacts but cannot qualify that release.
Do not rebuild shipping bytes for driver-only debugging without a concrete need.
SodaOS targets native x86_64; cross-compilation/emulation is not native or installed proof.

Use [docs/README.md](docs/README.md) to find the owning guide; read relevant sections only.
Keep one owner per requirement, link rather than duplicate, and update affected plan facts
when responsibilities, paths, dependencies or status change. No unchanged-plan audit.
Product docs are not status trackers; completed plans are absorbed and removed.

Use [testing](docs/development/testing.md), [native support](docs/development/native-support.md)
and [installation](docs/guides/installation.md) for execution commands and effects;
[Go](docs/development/go.md), [TypeScript](docs/development/typescript.md) and
`internal/archcheck` for language/boundary checks. Read the relevant operational guide
before provisioning, activation, VM or installed checks; these can change real state.

Only when explicitly requested, consult Jev/SystemOne with complete supplied context:
it cannot research. Make three fresh, semantically equivalent consultations with rewritten
prose and unchanged technical facts; investigate disagreements and treat agreement as
advice rather than proof. Keep the three consultations; do not create receipts
or a wording/equivalence report. This grants no external-call permission.
