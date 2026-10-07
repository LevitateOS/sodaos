# Prepared slice audit review input baseline

The preparation sections below retain the historical 2026-10-05 review scope.
Use the [current input baseline](#current-input-baseline-2026-10-07) for the
2026-10-07 correctness and maintenance audit preparation.

## Use during the authorized audit

The user started connected slice reviews on 2026-10-05 and directed continuation
through all 80 slices. Source stays pinned to the commit below. The original
working-guidance packet and this manifest were retained before review at
`.artifacts/slice-audit/f7e9cf9d-816989ab02e6/guidance/`, preserving their
repository-relative paths. The retained manifest SHA-256 is
`816989ab02e66c8c6803eb15547420e5ca8c6d58348a4817f55ed2f891dd8e7b`.
Verify its table against those retained bytes. Live records/indexes/plan documents
now evolve under the audit command; they are not substituted for the retained
input identities. Reconcile source drift against the pin before claiming newer
coverage. Later user instructions remain authoritative over archived prose.

## Preparation scope

Pinned on **2026-10-05** for the user-requested audit preparation. This identifies
the committed source and working guidance that reviewers must verify before
use. Pinning inputs does not launch reviewers, create slice review records,
establish intended models or authorize implementation or Git operations.

Working-guidance identities are explicitly reconciled on **2026-10-05** for
the later user-authorized reviewer binding/briefing step. Only assignment
guidance and this manifest changed in that preparation step; the source commit,
earlier coverage scopes and other working guidance remained as pinned. That
instruction authorized reviewer briefing. The later audit instruction recorded
above started the connected reviews; it did not authorize implementation.

## Distinct baseline identities

| Input or earlier evidence scope | Pinned identity |
| --- | --- |
| Repository | Canonical `sodaos` checkout; the separate `soda-os` checkout is excluded. |
| Review source commit | `f7e9cf9db616f93de351dd44c9bb7456feb608b9` |
| Committed source tree | `5dc7f4ff7312409a87a37eac4fdddcd5ed9939a1` |
| Committed tracked paths | 1,719 |
| Existing catalog/coverage evidence | `0d8d3b8ebb5a7df36759685529f27f417d96fac8`; 80 candidate slices, with its original responsibility-coverage scope. |
| Historical structural evidence | `d7e565aa1019753997a99fd430ba103d7a472b48`; its original structural scope remains separate. |
| Working guidance | The exact paths, statuses, byte sizes and SHA-256 values below. |

The checked committed delta is
`0d8d3b8ebb5a7df36759685529f27f417d96fac8` →
`f7e9cf9db616f93de351dd44c9bb7456feb608b9`: **22 modified Markdown paths**, zero added or
deleted paths, and **zero implementation changes**. The modifications are
within the living plan. This establishes the delta's source scope; it does not
turn earlier structural/coverage inspections into code-validity reviews or
refresh behavioral evidence.

At the pin, the working changes are documentation only: nine modified tracked
files, two untracked guidance documents listed below, and this new untracked
manifest. Tracked non-Markdown source has no working delta against the pinned
commit. Existing product/reference/development contracts without working changes
are identified by their content in that commit; their authority, applicability
and possible stale assumptions must still be assessed per slice.

## Uncommitted working guidance

The hashes cover each file's complete raw bytes, including final newlines,
captured after the guidance/link/count edits for this preparation step. This
table includes every other modified or untracked path reported by Git at the
pin. These identities supplement the committed source; a dirty checkout is not
silently treated as commit-only evidence.

| Repository-relative path | Git status at pin | Bytes | SHA-256 of raw bytes |
| --- | --- | ---: | --- |
| [AGENTS.md](../../../AGENTS.md) | modified tracked | 19154 | `7f83b9f98f4c41f44b7f3bf0b6235944bee4fdf68c2b9d9a3b42f1171a2b9f3f` |
| [docs/development/ideal-filetree-plan/coverage/README.md](coverage/README.md) | modified tracked | 46819 | `6d6123977e5a5264edc618d08923b3b5cfe1fe87045cc416337dc42b3f13c3f8` |
| [docs/development/ideal-filetree-plan/coverage/inventory/docs.md](coverage/inventory/docs.md) | modified tracked | 54514 | `97c98d380351370df2dd5593232764352461ae425b460d676cce6c344c0983fb` |
| [docs/development/ideal-filetree-plan/coverage/inventory/root-files.md](coverage/inventory/root-files.md) | modified tracked | 5012 | `e35f99b2c7ca2749a64f3b500151829bc3be3765d479482cb05bcc378c978ba9` |
| [docs/development/ideal-filetree-plan/ideal-filetree-plan.md](ideal-filetree-plan.md) | modified tracked | 8241 | `f4479d2d4740c72917967e36a2f531adf43bb501029c34adc08fdb79909525df` |
| [docs/development/ideal-filetree-plan/integration.md](integration.md) | modified tracked | 8242 | `a168a20075b4a80de5da29d3e1f1bf2965a3755c62e5e39db778e126f9fbb718` |
| [docs/development/ideal-filetree-plan/maintenance.md](maintenance.md) | modified tracked | 6256 | `0bf66ec1b57077a5148b26049586fd842e6c9238277e50a4b68610cf8b4b2225` |
| [docs/development/ideal-filetree-plan/proposed-tree.md](proposed-tree.md) | modified tracked | 124021 | `a871d690af5e6260fa9b306d07109bf397effe914b2962332fc0237ad8a87bb4` |
| [docs/development/ideal-filetree-plan/review-assignments.md](review-assignments.md) | untracked | 36685 | `d99c501c5bbfec49de38d689a02f5e13771ead79d8748812ea4585283da14f61` |
| [docs/development/ideal-filetree-plan/review-format.md](review-format.md) | untracked | 21516 | `ae533cb7445ed85a6d97ae75e5cda87ecd1ab3d96ef854660c783244bf46295a` |
| [docs/development/ideal-filetree-plan/slices/README.md](slices/README.md) | modified tracked | 4162 | `108c3b104f0707c65fb4514b19426b40435630f5d78189be5b0d243765127d0f` |

This manifest, `docs/development/ideal-filetree-plan/review-baseline.md`, is
explicitly excluded from its own hash table to avoid recursive hashing. Its
only role is recording the pin and verification rules; it adds no product or
language decision. The target tree and H06 planning coverage account for it.
The table identifies working contents; it is not a retained copy of those files.

## Controlling user instructions

These descriptive references identify the applicable instructions in this
conversation; hashes of repository files do not capture or override them.

- **OWNER-PIN:** pin `f7e9cf9d` plus the uncommitted guidance documents and
  verify the documentation-only committed delta.
- **OWNER-AUTHORITY:** the supplied Codex Collaboration System Prompt preserves
  human ownership, normal-mode authorship/operational limits, and the requirement
  for a direct scoped instruction. The AGENTS refresh reflects those instructions.
- **OWNER-PREPARE:** the user kept this phase in planning and separately requested
  baseline, review-format, ownership/collaboration, conflict and completion
  preparation. This pin does not authorize launching the full audit.
- **OWNER-REVIEW:** establish the intended model per slice, review every
  established responsibility and complete workflow, independently challenge
  findings, specify target ownership and retain blocking unresolved questions.
- **OWNER-REVIEWERS:** use GPT-6.1 Sol for the three reviewers, with actual
  runtime identities recorded in [assignments](review-assignments.md).
- **OWNER-BRIEF:** bind and brief A/B/C with their assignments, contracts,
  coverage maps and collaboration rules. This later preparation instruction
authorizes worker briefing, not slice audits or implementation.
- **OWNER-START:** begin connected identity, factory and persistence/private IPC
  reviews, create slice records as reviews begin, then work through all 80 slices.
  This subsequent instruction authorizes source review and review/plan records;
  it does not authorize implementation, Git mutation or runtime operations.

The [recorded language policy](ideal-filetree-plan.md#owner-language-policy)
and [guidance conflict register](review-assignments.md#guidance-conflicts-and-controlling-decisions)
remain inputs to applicability checks, not a source of additional authority.

## Verifying and using the pin

Before using this baseline, confirm the source commit and tree identities,
then verify the retained manifest and every retained guidance byte hash. The
statuses below describe the preparation pin, not later authored review records.
Check for changed implementation source and identify live guidance changes
separately. Without retained inputs, compare current files to the table and
reconcile any drift before use; never silently claim the original pin still matches.
Commit/status changes require scope reconciliation even if file bytes match;
this pin must not silently become a later commit's review baseline.

If inputs drift, record the new source/guidance identities and affected questions
under an authorized baseline refresh or explicit review-scope reconciliation.
Preserve which earlier evidence still applies; do not automatically advance
slice conclusions, catalog coverage or structural evidence. Current user
instructions remain authoritative if they amend an earlier decision.

Per-slice records cite this input baseline and separately record their actually
inspected source, controlling contract/user decisions, findings, performed
checks and limits under the [shared format](review-format.md). Ignored artifacts,
private inputs, remote services and installed behavior were not inspected by
this pin. No tests, builds, runtime probes or source-validity audit ran.

## Current input baseline (2026-10-07)

This refresh executes only the owner's baseline step: identify current source,
working guidance, dependencies and previous evidence scopes. It does not launch
the remaining audit steps, fix findings or establish integration simplicity.
The historical preparation and retained guidance above remain unchanged in scope.

| Input | Captured identity |
| --- | --- |
| Canonical source commit | `0b0734398f0350d75cc6fdb4dc8129251d8ab308` |
| Committed source tree | `00e7015175c1a4d7805b5dca409fc3956dba5bf6` |
| Committed tracked paths | 2,544; this count is not responsibility-review coverage. |
| Initial working changes | 12 modified tracked Markdown files; empty index; no uncommitted application source, manifest or lock changes. |
| Local SDK replacement | `../forgejo-ext/sdk`, clean sibling commit `c92db11c14b773c9cc20ccfa4b853b4c017e8717`, selected by `go.mod`; the placeholder module version is not its source identity. |
| Retained input packet | `.artifacts/audit-baseline-20261007-0b073439/`; `baseline.json` records exact identities, dirty paths, byte hashes and delta scopes. |

The 12 pre-existing changes are the plan entrypoint, task list, lane schedule,
adoption chapter, package ownership, placement and reviews G04/G07/H04/O01/O06/O07.
Their exact pre-edit bytes and diff are retained under `working-guidance/` and
`working-guidance.patch`; none is included in this baseline change. They reconcile
task/ownership status and named source corrections while preserving the original
review scopes. They are working inputs, not newly verified findings or runtime
evidence. This document's baseline update follows the captured source identity;
it is excluded from the current guidance hash inventory to avoid self-reference.

### Dependency and toolchain inputs

`dependency-inventory.json` retains every locked Rust/Bun package version, Go
module selection, manifest constraint and input-file hash. Raw manifests, locks,
toolchain selectors and asset locks are retained under `dependency-inputs/`;
the SDK module inputs are retained separately under `external-sdk/`.

| Family | Captured versions and selection scope |
| --- | --- |
| Rust | 28 workspace members; Cargo.lock v4 has 292 package entries, including local packages. Key resolved versions: serde_json 1.0.151, Tokio 1.53.2, tokio-postgres 0.7.18, Hyper 1.12.0, tungstenite 0.30.0, ssh-key 0.7.0-rc.11, x509-cert 0.2.5, url 2.5.8, time 0.3.55, clap 4.6.7, rustix 1.1.5, tempfile 3.27.0, flate2 1.1.10 and tar 0.4.46. Manifest requirements remain distinct from lock resolution. |
| Go | `go.mod` declares Go 1.26.7 and 47 requirements. An offline read-only module-graph query with Go 1.26.7 selects 90 modules, including the main module and local SDK replacement; selection does not prove package reachability. The 192 go.sum entries are integrity records. Selected integrations include pgx/v5 5.10.0 and coder/websocket 1.8.15. |
| Bun/JS | packageManager is bun 1.4.2; bun.lock v2 has 176 package entries. Root TypeScript is 7.0.2, Lit 3.3.3, Playwright 1.63.0, oxlint 1.81.0 and oxfmt 0.68.0; tools/lit-check separately selects TypeScript 5.9.3 and lit-analyzer 2.0.3. Root package name `sodaos` differs from the lock's `soda-cockpit`; record the naming drift without editing either input. |
| Observed executables | rustc 1.99.0 (`b940084d7`, 2026-09-28), Cargo 1.99.0 (`5f94df478`, 2026-08-27), explicitly selected Go 1.26.7 linux/amd64, Bun 1.4.2. Rust's `stable` selector floats; the Go directive is not an exact executable identity. These observations do not reconstruct earlier build environments. |

No dependency was installed or upgraded, and no manifest or lock was modified.
Native image/package inputs, installed dependency inventories and release-worker
environments are outside this capture; source dependency records do not qualify them.

### Previous scopes and evidence reuse

| Earlier evidence | Current disposition |
| --- | --- |
| Structural `d7e565aa` and catalog `0d8d3b8e` | Preserve their original scopes. Current path count does not advance their responsibility maps or close R02 inventory/tree/joins. |
| 80-slice review at `f7e9cf9db616f93de351dd44c9bb7456feb608b9` | Preserve the original source and contract conclusions at their recorded scope. Rename-aware delta: 425 exact renames, 119 modified renames, 260 modified paths, 994 additions and 169 deletions. Rename similarity is navigation evidence, not semantic equivalence; changed bindings, callers and contracts require reconciliation. |
| Adoption packets L00–L18 | Keep each recorded source revision, producer profile, independent review and performed-check scope in the [adoption chapter](library-adoption.md). Completed source packets do not close unrelated findings, held replacements or installed workflows. |
| R03.L at `d7eca882b5ce18f25ea3287dce578366d769deda` | Current committed delta is four Markdown files only. Retained focused evidence 16/16 and all-target offline Rust workspace compile checks remain usable for unchanged exercised source; packet-specific Go/Bun checks retain their original identities. No fresh behavioral checks were run here. |
| Working G04/G07/H04/O01/O06/O07 reconciliations | Named structural moves or source corrections only; they do not replace full slice review, repeat tests or establish installed/native qualification. |

The original retained manifest and all 11 listed guidance hashes were verified.
The three R03.L receipt hashes match their identities in commit `3bd86114`;
its source-pin transcription correction remains recorded in `9266d6de`.
`prior-input-integrity.json` records these checks. Raw and rename-aware source
deltas are retained under `deltas/`; unchanged contracts and working guidance
are identified separately in `guidance-identities.json`.
Separate delta segments use `72e4bb9015b6d6a622b45638104c74851a137473`
to distinguish earlier restructuring from subsequent adoption changes.

Reuse requires the same inspected or exercised source unit, applicable contract
and review question, with caller/lifecycle/dependency changes assessed. An exact
move alone cannot establish integration equivalence. Existing acceptance checks
do not answer the new maintenance-cost question for runtime bridges, duplicated
JSON mechanics or retained compatibility. Those questions remain for subsequent
audit steps. R02, broader R03, R04, B03.C, L16.G, L10.N4 and CFG01 retain their
existing scopes and gates; this input refresh closes none of them.
