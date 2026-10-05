# Prepared slice audit review input baseline

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
