# Over-engineering audit

<!-- Generated from the audit run's journal; edit the summary by hand, regenerate the ranked lists. -->

## How this was made

One finder agent read each subsystem in full (30 subsystems: 20 planned, plus 10 that two completeness-critic rounds asked for). Every finding then went to a separate agent that tried to refute it against the code. That agent could correct the title, locations, simpler version and line estimate. Corrected values are used below. Minor items got one batched check per subsystem; refuted ones are dropped.

Result: **317 findings** (133 confirmed, 184 partially confirmed, none refuted). By impact: 92 high, 153 medium, 72 low. Their estimates add up to about 284k removable lines. Findings overlap, so treat that as an upper bound.

Each finding records:

- **Visible change**: whether simplifying it changes what users see (UI), the model-facing MCP tool contract, or the DB schema.
- **Behaviour-preserving**: whether a version exists that changes none of those.
- **Status**:
  - _done_: a commit on `simplify/over-engineering`;
  - _owner decision_: changes a visible feature, tool contract or schema, so it waits for the owner;
  - _coordinate with prod-hardening_: touches files the `prod-hardening` branch edits;
  - _open_: behaviour-preserving and not yet done.

Ranking: impact first, then removable lines.

## Summary

SodaOS has zero users, but much of its code is built for adversarial, multi-tenant, long-lived operation. Two kinds of over-engineering account for most of the size:

- Machinery that guards against conditions that cannot occur yet: digests, receipts, revision chains and strict re-validation at every boundary.
- Machinery that rations things that are free or abundant: JEV calls, coder minutes, review rounds.

Both kinds caused the founder-demo failures. The themes below cover almost all of the high-impact findings.

### 1. One run, one clock: collapse the budgets

**Findings:** BUD-1..12, DSP-5, FEF-3, WEB-4, DLV-2.

A run's deadline is the minimum of up to six budgets:

- the sponsorship's lifetime minutes;
- the connection's 24-hour rolling minutes;
- the attempt's active-time clock;
- the 179-minute host bound;
- Issue-zero funded minutes;
- repair grants.

The deadline is fixed when the packet is recorded, before the JEV context-bundle choice runs. A nearly spent budget therefore produces a 1-5 minute run. A slow or failed choice burns that time and ends as `stop-before-start`, and the run is still charged. This is the 4-minute death from the demo.

**Simpler:**

- one constant per-run wall clock, started when the host actually launches;
- one soft spend cap, checked at admission;
- charge only runs that reached the host;
- one loop cap (correction cycles).

Budget numbers also live inside revisioned authority grants. Raising an allowance therefore withdraws the repository's open publications and merges (BUD-2).

### 2. JEV classifiers: keep them, strip the machinery around them

**Findings:**

- Machinery around the calls: JDG-2, BUD-5..7, PLN-2, RTY-4, DSP-1, MCP-4, CTX-3, CTX-4.
- Classifications whose answer reaches nothing: DSP-3 and PLN-1 (planning freshness) and PLN-8 are removed. Tracing the consumers showed the answers of PLN-4 (walkthrough consistency), DSP-2 (priority tournament), JDG-1 (post-merge reassessment fence) and FED-4 (role contributions) do reach a gate, a dispatch choice, a worker inbox or the UI, so their calls stay; what remains open there is the machinery around them, which is a product decision for the owner.

System-one classifiers such as JEV simplify the code. A classification call replaces branching code that would otherwise imitate judgment, and AGENTS.md says the calls are free. The over-engineering is around the calls:

- each call is "funded" once from rationed budgets (per-intent planning calls, per-attempt delivery calls, a global ceiling that is never checked);
- a failed call stays recorded, and replay rejects it, so the run holds until its deadline and only Stop+Retry recovers it;
- the paid selection is fingerprinted and re-checked at every finish, task start and consult;
- some classifications gate work, but their answer is never shown to the worker or used.

**Simpler:**

- keep the classifiers;
- make each one a plain call that retries on failure and falls back to a sensible default (for the context bundle, `current`) instead of holding the run;
- delete the funding ledgers, call budgets and reservation receipts;
- for a classification whose answer goes nowhere, either route the answer to where it was meant to go or drop the call.

### 3. Model-facing tools make the model echo what the server knows

**Findings:** MCP-1..5, BND-1..3, VRF-4, RUN-1, RUN-3, CTX-3, CTX-7, PLN-6.

Finish and submit tools require the model to copy back server-known values and then refuse on any mismatch:

- `capability_id` and `expected_input_digest`;
- attempt root, source commit and preparation;
- the plan digest and plan id;
- tuple and coverage digests;
- delivery ids and receipts.

The finish tools also freeze their input before validating it, so 16 park paths wedge the assignment silently (MCP-2). The payload shape is defined in three places, and the definitions disagree (MCP-3). Every phase also has two result channels: a fenced stdout block and an MCP finish tool (MCP-5, RUN-3).

**Simpler:** tools take only the model's judgment, and the bound server fills in every binding. These are model-facing contract changes, so they are owner decisions.

### 4. Feed, settings and frontend parsers

**Findings:**

- Feed: FED-1..7.
- Settings: FEF-1, FEF-2, GOM-4, BND-6.
- Response parsers: FEF-4, FED-3, BND-8.

The Feed carries:

- a 122-entry component catalog, with the component chosen by the worker;
- about 57 card renderers over three card models;
- a client-side thread cache with draft lineage and N+1 polling.

Four card types and one enriched refresh would cover what the Feed shows.

Settings ask the owner to pick broker grants, see credential generations, type harness versions and set four concurrency caps, none of which take effect.

About 6.8k lines of response parsers re-validate SodaOS's own JSON (exact key sets, closed enums, 32-hex ids). They fail the whole view on any drift.

### 5. Speculative subsystems nobody turns on

**Findings:**

| Subsystem | Findings |
|---|---|
| Optional lifecycle effects (~2.5k lines; needs a credential no setup creates) | DLV-5, WEB-2, DSP-12 |
| Tailnet (~10k lines across build, host and UI) | HPL-5, RBA-11, EXT-3 |
| Codex factory harness (implement runs cannot start by default) | HFW-2 |
| Library Admin and Researcher roles (reachable only from a hand-written block) | CTX-1, PLN-9 |
| Departments | PLN-11 |
| Restore crash-recovery journal | APP-3 |
| Stall investigator | DSP-10 |
| The human "Start Codex" broker | PTH-2 |

### 6. Duplicate schemas, release machinery and docs

**Findings:**

- Schemas: STO-1, STO-2, IDN-4.
- Release machinery: REL-1..3, TST-2, HPL-6, EXT-4, DOC-3.
- Docs: DOC-1, DOC-2.
- Forgejo template overrides: FEP-4.

The Rust identity broker carries a mechanical copy of the whole Go schema: 85 tables, a version gate and migrations. The store runs three schema-evolution mechanisms.

Release qualification is built for a product with users:

- sealed evidence, journeys, a candidate sandbox with its own SELinux policy, and hand-written enrollment manifests;
- an installed qualification harness of about 12.5k lines that no CI runs.

The docs include:

- a 4,000-line Factory execution tracker;
- seven overlapping Factory specs that freeze the budget and digest machinery;
- 264 Forgejo template overrides (about 17k lines) that re-skin native pages.

### What this branch did

The branch is `simplify/over-engineering`, cut from `main`, with `prod-hardening` merged in. The first eleven commits are behaviour-preserving. The later ones, made on the owner's go-ahead, change visible behaviour or model-facing tool contracts; none adds a DB migration (unused tables stay in the schema). Across the branch about 92,000 lines are deleted and 12,000 added, about 80,000 net.

| Commit | Change | Lines removed |
|---|---|---|
| `0e59bf58` | Go code nothing reaches (deadcode + staticcheck to a fixed point), including the test-only `modernc.org/sqlite` dependency | ~5,150 |
| `a9047a8a` | Unused frontend exports | ~95 |
| `0711047a` | Dead decomposition-judgment path (PLN-7) | ~490 |
| `61fbce18` | Test-only merge-ledger and replacement-generation code (DLV-3, first step) | ~1,300 |
| `f03e8b4e` | P6 MCP transport prototype | ~480 |
| `c6d141c6` | Dead Go prompt-entry record and project constants (PRJ-8) | ~70 |
| `8764b259` | Test-only Go copy of the identity broker, grant cipher and keyed store entrypoint (IDN-9) | ~900 net |
| `2775af41` | Advisory complexity reporter (TST-11) | ~950 |
| `d0435632` | Dead Rust host runtime code (HFW-3, HPL-12, PTH-5, HFW-12, TRM-7) | ~2,400 |
| `762e0dc4` | Unused provider-store review probe binary (REL-10) | ~790 |
| `3e6b5b31` | Dead `can_embed` field left from the removed in-Forge app frame (FEP-8) | ~70 |
| `b1d4c63e` | One clock per run: the per-run limit; soft spend cap; no rolling budget or short-allowance hold; Retry once runs settle (BUD-1/3/4, DSP-5, DLV-2) | ~1,500 |
| `72c5afe7` | The context classifier advises instead of holding the run (DSP-1, MCP-4) | ~210 |
| `42a051e9` | MCP tools take only the payload: no expected_input_digest, optional submission ids (BND-1, MCP-1) | ~90 |
| `291451f6` | SodaOS fills role bindings (review tuple and coverage, finding digest, manifest digest, delivery and journey bindings) (MCP-1, BND-3, VRF-4, CTX-7) | +70 |
| `fcc7b413` | Delete optional lifecycle effects (DLV-5, WEB-2, DSP-12) | ~4,700 |
| `8ab34803` | Delete the Feed component catalog and worker-chosen component_id (FED-1) | ~1,950 |
| `c62f6ec1` | JEV calls retry instead of staying spent; planning-call allowance and delivery-call budget removed (JDG-2, BUD-5/6/7, PLN-2, STO-6) | ~2,700 |
| `3b2273ff` | Muse-only factory: the Codex factory harness is gone (HFW-2) | ~2,250 |
| `034f553c`, `ab431714` | Delete the Library Admin and Researcher roles and the non-code assessment path (CTX-1, PLN-9) | ~4,400 |
| `92b906a6` | Installed qualification harness (TST-2) | ~12,600 |
| `cbc4918b` | soda-project-maintain release/recover/rollback/reconcile system (HPL-6) | ~3,650 |
| `c529efb5` | Release qualification, journey and upgrade receipts; promote after a smoke install (REL-2) | ~5,900 |
| `9b348cd0` | Media self-signing, Trust/Permit models, fixture authority, lab-credentials (REL-3 first step) | ~3,050 |
| `f6e409c5` | Extension asset inventory and version recheck (EXT-4 first step) | ~80 |
| `91248f8a` | Release docs cut to the operator how-to (DOC-3) | ~3,070 |
| `590f35f2` | Factory tracker packets and body; founder-demo checklist kept (DOC-1) | ~3,180 |
| `9f76b333` | Stale Feed component specs (DOC-2 first step) | ~1,930 |

### Owner decisions: done and still open

On 2026-10-28 the owner asked for complexity to come down overall, not just dead code, so these changes now go ahead in this order on the same branch. JEV and system-one classifiers stay: they replace code that would imitate judgment. What goes is the funding, one-shot and receipt machinery around them.

Each of these changes a visible feature, a model-facing tool contract or the DB schema. They are listed roughly by payoff.

Done on this branch: 1 (budgets, including the connection usage budget and the dead concurrency settings), 2 (JEV calls plain and retryable; the assumption and planning-freshness calls whose answers reached nothing are gone, and the four whose answers do reach a gate or a person stay), 3 (bindings and one result channel per phase: MCP finish tools only), 4 (component catalog, one card render path, plain JSON reads and one Feed refresh), the settings and frontend parser sprawl, and from 6 the lifecycle effects, Codex factory harness and Library Admin/Researcher roles, and most of 8 (qualification, the project-maintain release path, media self-signing, release docs and the tracker body). Still open: 5, Departments (with the Workspace removal) and Tailnet (in the installed product; a product decision) in 6, 7, and from 8 the build sandbox (REL-3), the remaining Factory specs (DOC-2) and the founder-demo checklist (DOC-1).

1. **Budgets:** replace the budget stack with one per-run clock started at launch, one soft spend cap and charging only for runs that reached the host (theme 1). Delete the JEV rationing: planning-call allowance, delivery-call budget, funding journal.
2. **JEV classifiers:** keep them as plain calls that retry and fall back to a default instead of holding the run. Delete their funding ledgers, call budgets and receipt re-checks. Route or drop the classifications whose answer reaches nothing (planning freshness, walkthrough consistency, reassessment fence, priority tournament, role contributions).
3. **Finish tools:** reduce them to judgment-only payloads; the server fills bindings and digests. Use one result channel per phase.
4. **Feed:** delete the component catalog and collapse it to four card types with one refresh.
5. **Settings:** reduce them to what a single owner needs, dropping the dead concurrency and queue knobs and grant pickers. Loosen the frontend parsers to the fields each view reads.
6. **Unused subsystems:** delete optional lifecycle effects, the Codex factory harness, the Library Admin and Researcher roles, Departments, and possibly Tailnet.
7. **Schema copy:** drop the Rust broker's copy of the Go schema and the extra schema-evolution mechanisms (DB-facing).
8. **Release, qualification and docs:** delete the qualification, sandbox and journey machinery and the Factory tracker docs. Nothing user-facing changes, but these remove processes that AGENTS.md or release docs still name.

### Deferred

- **Coordinate with `prod-hardening`:** findings that touch files that branch edits are marked _coordinate_. That branch adds symptom fixes, such as a short-allowance hold and `MinUsefulRunMinutes`, on top of the budget stack that theme 1 would remove.
- **Owned by `repo-first-ui`:** Workspace findings (FEP-1, FEF-5 and the Workspace test suites in TST-10).

## Ranked findings: high impact

### FEP-4: 264 Forgejo template overrides (17k lines) re-skin native pages via soda-* classes; a 10k-line unread inventory and a pinned locale merge pipeline serve 6 used strings

- **Subsystem:** Frontend: Workspace, project, applications, terminals, layout, Forgejo and Tailnet pages. **Category:** process-machinery. **Verdict:** partially-confirmed.
- **Removable:** ~15000 lines. **Risk:** medium. **Visible change:** UI. **Behaviour-preserving:** yes.
- **Status:** owner decision.
- **Where:** `frontend/forgejo/templates/** (264 files, 17,091 lines; admin 25/2,126, repo/migrate 12/1,403, repo/settings 38/2,435, user/settings 37/2,000, org 16/793, package 8/506, shared 10/837)`, `frontend/forgejo/README.md:1-827`, `frontend/forgejo/locale.lock.json:1-4`, `frontend/forgejo/locales/en-US.ini:1-11`, `frontend/forgejo/locales/README.md:1-42`, `frontend/forgejo/payload.json:2,48`, `tests/forgejo/presentation/inventory.json (10,013 lines, read only by README)`, `tools/release-assets/src/locales/merge.rs:1-368`, `tools/release-assets/src/bin/soda-forgejo-locales.rs`, `tools/release-assets/tests/locales_*.rs`, `tests/forgejo/presentation/locales.test.ts`, `assets/branding/forgejo/*.css (about 5,990 lines that depend on the override classes)`

**What.** The appliance ships 264 template overrides (17,091 lines). 243 of them replace native Forgejo templates for branding: admin LDAP/SMTP/PAM/OAuth editors, migration forms for eight forges, LFS, runners, package cleanup rules, webhooks, wiki and org teams. Each carries an 'Embedded source SHA-256' header for the 15.0.9 original while the target is 15.0.10. A 10k-line inventory.json accounts for every override. The README is an 827-line journal of local checks. To add 11 English strings, the build pins upstream locale_en-US.ini by URL and sha256, merges it with a Rust tool and stages a full replacement catalog. 4 of the 11 keys are unused.

**Simpler.** Delete inventory.json, which nothing reads. Delete the 14 Embedded-source-SHA headers. Delete the 4 unused locale keys. Write the 6 used strings as literal English in the templates, then delete locales/, locale.lock.json, the payload locale entries (lines 2 and 48), the Rust locale merge tool with its bin and tests, and locales.test.ts. Cut the README down to the asset mapping and the reload command. Separately, and only with owner sign-off: keep the overrides that carry Soda hooks plus sign-in and home branding, drop the rest, and strip from the branding CSS the soda-* rules that only those dropped templates used.

**Risk.** Many native pages change visually. The owner directed the September redesign, so confirm which pages must stay branded. Theme CSS may assume classes that only the overrides add; spot-check stock pages. ui/repo-places edits repo/header.tmpl and the settings navbars and adds places templates; keep those. No hardening overlap.

**Visibility.** Native admin, settings and migration pages revert to stock Forgejo layout under the Soda theme. Inlining the strings and trimming docs/inventory has no visible effect.

**Safe first step.** Delete tests/forgejo/presentation/inventory.json (only README.md mentions it) and the README paragraphs about it. Delete the 4 unused keys (profile_portrait, tailnet_title, tailnet_appliance, tailnet_projects) from frontend/forgejo/locales/en-US.ini. Remove the 14 "Embedded source SHA-256" comment lines from the templates. Then write the 6 used soda.* strings as literal English where the templates call them, and delete frontend/forgejo/locales/, frontend/forgejo/locale.lock.json, payload.json entries at lines 2 and 48, tools/release-assets/src/locales/ with bin/soda-forgejo-locales.rs and tests/locales_*.rs, and tests/forgejo/presentation/locales.test.ts. The rendered output stays identical, but first check that the stock Forgejo catalog shipped with Fountain is used once the replacement is gone.

**Related.** The *-native-contracts.json files, form-presentation-deltas.json and the gallery templates in tests/forgejo/presentation/ follow the same per-override accounting pattern. The branding CSS in assets/branding/forgejo/ is tied to the override classes.

### TST-2: Installed 'qualification' harness: about 12.5k lines of receipt, seal and custody release machinery that no CI runs (CI only type-checks it)

- **Subsystem:** Test harnesses and source guards. **Category:** process-machinery. **Verdict:** confirmed.
- **Removable:** ~12600 lines. **Risk:** medium. **Visible change:** none. **Behaviour-preserving:** yes.
- **Status:** done (`92b906a6`): tests/installed/qualification and scripts/build-qualification.ts deleted (the stale `rolling_minutes`/`max_concurrent` mentions went with them).
- **Where:** `tests/installed/qualification/** (55 files, 12,451 lines)`, `tests/installed/qualification/main.ts:1-92`, `tests/installed/qualification/report.ts:1-479`, `tests/installed/qualification/collector.ts:1-319`, `tests/installed/qualification/closure.ts:1-268`, `tests/installed/qualification/check-retirement.ts:1-172`, `tests/installed/qualification/phases/factory-phase.ts:1-882`, `tests/installed/qualification/phases/docker-phases.ts:1-671`, `scripts/build-qualification.ts:1-149`, `lib/soda-release-tools/README.md:100-200 (build-qualification instructions)`, `lib/soda-release-tools/src/qualification.rs + src/qualification/ + src/worker/ [outside scope]`, `docs/development/release.md:290-800`

**What.** This is a release-qualification framework. A Rust side owns an 'immutable request/scenario envelope', 'authority, credential delivery, native custody and sealing'. A TS 'protected worker adapter' collector writes scope receipts and sha256-bound observation documents with wx/0600 files. Around them sit admission, retained/interrupted admission recovery states, readiness, closure ('Stable retained identities must not drift'), retirement guards and a 'receipt projection' report. The build script refuses dirty trees and records sha256 of package.json, bun.lock, go.mod and go.sum.

**Simpler.** Delete tests/installed/qualification/ and scripts/build-qualification.ts. Delete the matching sections of docs/development/release.md and lib/soda-release-tools/README.md. Keep one founder-journey e2e script as the installed check. Hand the Rust qualification and worker modules in lib/soda-release-tools to the release-tooling finder so they are removed together. Leave alone the separate qualification-tag and receipt code in push.sh, ci-reuse.sh and soda-release-deliver, unless that finder removes it too.

**Risk.** The Rust release tooling (soda-release-tools qualification, tools/acceptance) is coupled to this harness, so remove them together. Make sure no installer path depends on a 'qualification seal' first.

**Visibility.** Test and release tooling only. The product does not change.

**Safe first step.** Delete the tests/installed/qualification/ directory (55 files) and scripts/build-qualification.ts. Remove the build-qualification and harness sections from docs/development/release.md (about lines 290-800) and from lib/soda-release-tools/README.md (about lines 100-200). CI does not run these files and no product or installer code imports them. The only CI effect is that the tsconfig.tests.json typecheck has fewer files.

**Related.** lib/soda-release-tools/src/qualification.rs, src/qualification/target.rs, src/worker/*, src/journey.rs (soda-qualification-journey) and tools/acceptance consume this harness's build output. The separate qualification-receipt and tag machinery lives in lib/soda-release-deliver/src/publish/qualification.rs, upgrade.rs, scripts/push.sh and scripts/ci-reuse.sh.

### RBA-11: Optional Tailnet product feature (in documented scope) spans build, payload, importer, units, host runtime and UI, about 9-10k lines

- **Subsystem:** Release build primitives, asset fetch/render tools, small install binaries and system units. **Category:** too-many-states-or-budgets. **Verdict:** partially-confirmed.
- **Removable:** ~9500 lines. **Risk:** medium. **Visible change:** UI. **Behaviour-preserving:** no.
- **Status:** owner decision; left. Tailnet ships in the payload (OptionalExtensions, sixth image role), the image importer and the host units, so it is not unused; removing it is the owner's product call.
- **Where:** `system/host/services/soda-tailnet-host.service`, `system/host/services/soda-tailnet@.service`, `system/containers/extension/tailnet-package/`, `system/licenses/tailscale-LICENSE`, `lib/soda-release-build/src/production_assets.rs:97-105`, `lib/soda-release-build/src/production_inputs.rs:16-18`, `lib/soda-release-build/src/production.rs:75-95`, `cmd/soda-image-import/src/import.rs:10-40`, `lib/release-inputs/src/payload.rs (TAILNET_* constants, OptionalExtensions, validation ~460-480)`, `lib/host/src/tailnet/ (6298 lines)`, `frontend/tailnet/ (~1.6k lines)`, `frontend/spaces/sodaspaces-network.ts`, `internal/tailnet/`, `cmd/soda-tailnet/`, `internal/web/api/tailnet.go`, `internal/web/extension.go`, `tests/build/tailnet_image_test.go`, `docs/product/scope.md:62`

**What.** Tailnet is an 'optional' extension. It needs a second extension package build (build-soda-extension --package=tailnet), a hand-pinned official Tailscale image digest frozen into every build as a sixth role, an `ImageSet::Tailnet` with an `ensure --set tailnet` verb in soda-image-import, two systemd units, payload v4 OptionalExtensions/TailnetDependency validation ('exactly soda then tailnet'), and lib/host/src/tailnet at about 6.3k lines. The README itself says 'host Tailnet enrollment alone does not provide' project routes. For an unreleased product with zero users this is a whole optional feature path on the install and build critical path.

**Simpler.** This is an owner product decision, not a mechanical simplification. Tailnet is listed as in scope in docs/product/scope.md. If the owner drops it, remove the whole thing: units, the tailnet extension package and its build step, the Tailscale frozen input (five roles instead of six), ImageSet::Tailnet, the payload OptionalExtensions block, lib/host/src/tailnet, internal/tailnet, cmd/soda-tailnet, the API handlers, frontend/tailnet, and the related tests and docs. If the owner keeps it, there is nothing to cut wholesale under this finding.

**Risk.** This is a product decision rather than a refactor. It touches the payload format, the extension catalog and host code outside this scope.

**Visibility.** The Tailnet extension and its settings disappear from the forge.

**Related.** The Tailnet UI is in frontend/tailnet/ and frontend/spaces/sodaspaces-network.ts. The Go side is internal/tailnet/, cmd/soda-tailnet/, internal/web/api/tailnet.go and the extension wiring in internal/web/extension.go and internal/web/auth/extension.go. Tests are tests/build/tailnet_image_test.go and internal/web/tailnet_test.go. Docs: docs/architecture/networking.md#tailnet-model, docs/product/scope.md and docs/product/overview.md.

### TST-7: factory-feed-browser.test.ts (10.7k lines) pins a client-side thread cache and a draft-lineage system

- **Subsystem:** Test harnesses and source guards. **Category:** too-many-states-or-budgets. **Verdict:** partially-confirmed.
- **Removable:** ~9000 lines. **Risk:** medium. **Visible change:** UI. **Behaviour-preserving:** no.
- **Status:** done on `simplify/feed-render`: the file keeps 12 end-to-end tests; the cache, lineage and race tests went with the code.
- **Where:** `tests/frontend/factory-feed-browser.test.ts:1116-3119 (revision draft lineage/ancestor/unverified, 17 tests, ~2,000 lines)`, `tests/frontend/factory-feed-browser.test.ts:3580-10703 (thread paging/tombstone/resurrection/body-retry races, ~7,100 lines; 3119-3580 are source-GET stale guards, a few worth keeping)`, `tests/frontend/factory-feed-communication-browser.test.ts:1846-2291 (Check progress / unavailable recovery races only)`, `frontend/spaces/sodaspaces-factory-feed.ts:130-450 (draft lineage/ancestor persistence and retirement)`, `frontend/spaces/sodaspaces-factory-feed.ts:486-600,794-930,1090-1215,1589-1670,1809-1830 (thread window merge, authoritative removal, resurrection guards)`, `frontend/spaces/sodaspaces-factory-feed.ts:2985-3200 (unverified revision / refreshRevision)`, `tests/frontend/fixtures/drawer-fixture.ts, project-controls-driver.ts`

**What.** About 80 of the file's 90 tests cover two pieces of client-side machinery. The first is an incremental client thread cache that merges truncated, disjoint and overlapping pages, tombstones, 'authoritative older completion purges', and 'never resurrects' rules. The second is a draft-persistence system with lineage keys, ancestor-chain retirement, adopted-key collisions, cross-tab divergence and an 'unverified revision' CAS state. The suite drives these through a mocked backend fixture in a real browser.

**Simpler.** On open and refresh, re-read the newest N thread replies and render what the server returns. Keep older paging append-only. Do no client merge, tombstone reconciliation or resurrection guards; keep only a request-serial check that drops stale responses after a project, view or source switch. Store one revision draft per (post, account) in localStorage and send it with the server's current revision. On conflict, show the server text next to the draft. Drop lineage tokens, ancestor chains and the 'unverified' state. Keep about 10 e2e tests: read, comment, reply, revision submit and conflict, failure/retry, and the stale-switch guard. Delete the remaining race tests together with the code.

**Risk.** Requires simplifying the feed frontend at the same time; coordinate with the frontend finder and with the hardening branch, which edits sodaspaces-factory-feed.ts. Very long threads would refetch more data, which is fine at local scale.

**Visibility.** Draft-conflict notices and 'unverified revision' states would change or disappear. Ordinary feed reading looks the same.

**Related.** Similar race and tombstone handling sits in frontend/spaces/sodaspaces-factory-progress-view.ts (13 matches) and in sodaspaces-factory-feed-response.ts / progress-response.ts / stream-response.ts (truncated and authoritative parsing). There are also 3 Check progress race tests in tests/frontend/factory-feed-communication-browser.test.ts. The prod-hardening branch edits sodaspaces-factory-feed.ts (allowance read, plus a reformat in the shield/pipeline code), so the changes need coordinating.

### DOC-2: Seven overlapping Factory specs (~7,350 lines) restate code contracts and freeze the budget and digest machinery

- **Subsystem:** Documentation. **Category:** duplicated-layer. **Verdict:** partially-confirmed.
- **Removable:** ~6800 lines. **Risk:** low. **Visible change:** none. **Behaviour-preserving:** yes.
- **Status:** partly done (`9f76b333`): factory-feed-components.md (describing the catalog `8ab34803` removed) and factory-feed-and-ralph-integration.md are deleted. Open: merging factory-worker-mcp.md, factory-workflow-contracts.md, factory-interfaces.md, factory-demo-journey.md and the capability map (trust.md and overview.md cite its source traces) into one factory.md; that is an editorial rewrite that needs the owner's design intent.
- **Where:** `docs/development/factory-workflow-contracts.md:1-2486`, `docs/design/factory-worker-mcp.md:1-1536`, `docs/design/factory-feed-components.md:1-1149`, `docs/design/factory-feed-and-ralph-integration.md:1-765`, `docs/research/factory-capability-map.md:1-514`, `docs/architecture/factory-interfaces.md:1-497`, `docs/design/factory-demo-journey.md:1-407`, `internal/factory/README.md:56 (and 8 other package READMEs linking factory-interfaces.md)`, `docs/README.md, docs/development/README.md, docs/architecture/trust.md, docs/architecture/overview.md (inbound links)`

**What.** The same Factory is specified seven times, plus product/overview.md (778 lines) and trust.md's Factory section. The cross-ownership disclaimers point at each other: feed-components 'expands the worker MCP redesign', feed-ralph says 'workflow contracts, factory interfaces and trust model continue to own execution', and factory-interfaces says 'the implementation plan records verification scope'. Contracts are restated in detail: D14 (lines 1257-1460) lists every MCP tool with its input schema, the 'submission_id is a 32-hex identifier' rule (line 1396), digest-bound page tokens, 60 calls/minute/run and the 96 KiB envelope. D04 (lines 250-298) is a 'Frozen value' table that codifies the overlapping budgets from demo evidence #2: 120 active minutes, 3 corrections, ReviewRounds=6, DeliveryCalls=8, ReassessmentMinutes=30, 32 merge generations, and 'Reserve calls in the same transaction as the existing global budget'. D14 also specifies the paid context-bundle Jev choice ('one fixed bundle foregrounded by a saved Jev choice... Missing/uncertain selection → basic snapshot plus hold'), which is demo evidence #3. factory-feed-components.md has a 121-row prose catalog alongside the executable internal/factory/feed_components.json, which has 122 id entries.

**Simpler.** Replace the seven docs with one docs/architecture/factory.md of about 300 lines. It covers the roles and what each produces, the request→plan→build→review→merge→verify flow, one line per MCP tool (pointing at internal/web/api/factory_mcp*.go for schemas), the budget the code keeps, and a pointer to feed_components.json for the card list. Delete all seven, including factory-interfaces.md. Merge any demo-journey persona text into guides/founder-demo.md. In the same change, repoint the 9 package READMEs and about 20 docs that link to these files at the new doc.

**Risk.** Docs only, and code does not read them. Some prose carries owner design intent (feed role colours, library-admin scope), so skim for decisions worth one line in the new doc. Doing this together with the code simplification of budgets and MCP digests avoids the docs going stale twice.

**Visibility.** These docs are not fed to workers. The MCP descriptions live in Go. They describe the model contract but do not define it.

**Safe first step.** Delete docs/design/factory-feed-components.md's per-variant catalog tables. internal/factory/feed_components.json is the source the code uses, so keep only the intro paragraph that points to it. Also delete docs/research/factory-capability-map.md and docs/design/factory-feed-and-ralph-integration.md, and fix the inbound links in docs/README.md, docs/development/README.md and docs/architecture/*.md. No code reads any of these files.

**Related.** docs/development/factory-remaining-tasks/*.md and docs/development/public-handbook/*audit*.md also restate these contracts. docs/product/overview.md (778 lines) and the Factory section of docs/architecture/trust.md overlap them as well.

### HPL-5: Tailnet: about 11.5k Rust lines, a per-project companion feature, an epoch-keyed grant store, and lifecycle calls validated again on the daemon side

- **Subsystem:** Rust host platform: daemon, terminals, project, application, tailnet (lib/host non-factory, soda-project-maintain, forgejo/pg/console helpers, soda-tailnet, unix-http, wire-time). **Category:** speculative-abstraction. **Verdict:** partially-confirmed.
- **Removable:** ~5500 lines. **Risk:** high. **Visible change:** UI. **Behaviour-preserving:** yes.
- **Status:** owner decision; left. Tailnet is in scope (docs/product/scope.md) and used by the installed product: the tailnet package is built into every release, the host units ship in the image and the Network settings UI calls these paths. Only clearly unused code was to be removed here.
- **Where:** `lib/host/src/tailnet/companion/ (1815 lines)`, `lib/host/src/tailnet/runtime/ (1081)`, `lib/host/src/tailnet/files/ (819)`, `lib/host/src/tailnet/domain/ (921)`, `lib/host/src/tailnet/admission.rs:74-231`, `lib/host/src/tailnet/lifecycle.rs`, `lib/host/src/tailnet/lifecycle/runtime.rs`, `lib/host/src/tcontrol*.rs (5161)`, `lib/host/src/daemon/routes.rs:397-450`, `lib/host/src/dbackend/tailnet.rs:1-63`, `lib/host/src/dbackend/tailnet.rs:270-292`, `lib/host/src/dbackend/backend.rs:46-82`, `lib/host/src/main.rs:200-370`, `system/host/services/soda-tailnet@.service`, `internal/host/tailnet.go:52-161`, `frontend/spaces/sodaspaces-project-journey-view.ts:333-341`, `frontend/spaces/sodaspaces-project-settings-view.ts`

**What.** An optional Tailscale extension carries: a Tailscale companion container per project (soda-tailnet@%i.service run through soda-host --tailnet-action) as well as a host container; a policy file with random 128-bit revisions, compare-and-swap and confirm_id; provider HTTPS through host curl for key creation; an enrollment state machine (save/rotate/check/disable/default with admission/default flags); and an 'ephemeral applied authority' GrantStore. The grant is valid only when nine fields match the Fountain manager state plus a ServiceEpoch (pid start time and boot id read from /proc). Lifecycle calls from the Go web server are fully re-validated in routes.rs validate_lifecycle_call: expiry, controller 'soda'/package 'tailnet' scope, operation-to-endpoint binding and release pin. lifecycle_deadline in dbackend/tailnet.rs checks expiry again, commented 'defense in depth'. /tailnet/options rewrites available/default after a GrantStore::admit. Errors are classified by substring ('conflict', 'unsupported') and mapped onto unrelated BackendError variants (ExportStale for 409, ExportCandidate for 422). On the Go side, internal/host/tailnet.go re-checks the daemon's own replies (admitEnrollmentSave, tailnetProjectConfirmed).

**Simpler.** Cleanup that keeps behaviour: drop the repeated expiry check in dbackend lifecycle_deadline (keep only the remaining-budget clip). Replace map_tailnet_err's substring matching with a small typed error enum mapped to the same statuses. Drop the Go-side re-checks of the daemon's own replies (admitEnrollmentSave*, admitEnrollmentAction, tailnetProjectConfirmed). Drop ServiceEpoch and keep the grant as the plain manager-state match. As a separate product decision: drop per-project companions (companion/runtime modules, soda-tailnet@ unit, run_tailnet_action, project Tailnet UI) and collapse the GrantStore and revision CAS/confirm_id into "extension enabled, so the host node runs with the saved key".

**Risk.** Removing per-project nodes and enrollment options is a product decision, and it touches the Network settings UI and the Go tailnet package. The validation and grant parts can be removed alone at lower risk, about 800 lines.

**Visibility.** Dropping per-project Tailnet options changes the Network settings and project-creation Tailnet choice. Removing only the grant store and re-validation is invisible.

**Safe first step.** In lib/host/src/dbackend/tailnet.rs, remove the expired-call rejection in lifecycle_deadline, keeping only the min(remaining, cap) clip, because routes.rs validate_lifecycle_call already rejects expired calls. Replace map_tailnet_err's substring matching with a typed TailnetError enum returned by tcontrol (same 400/409/422/503/502 mapping). In internal/host/tailnet.go, delete admitEnrollmentSave/admitEnrollmentSaveRotate/admitEnrollmentAction and tailnetProjectConfirmed and return the daemon reply directly.

**Related.** The Go side mirrors this: internal/tailnet/control_validation.go, control_types.go, project_runtime.go, identity.go, and internal/host/tailnet_lifecycle.go with extension_lifecycle*.go. The UI is frontend/tailnet/* and the frontend/spaces project network files.

### REL-2: Release qualification is a protected collector, sealed-evidence and journey system for a product with zero users

- **Subsystem:** Release, install, packaging, CI and freshness machinery. **Category:** unneeded-validation-or-digest. **Verdict:** confirmed.
- **Removable:** ~5500 lines. **Risk:** low. **Visible change:** none. **Behaviour-preserving:** no.
- **Status:** done (`c529efb5`): collect/finalize-evidence, journey and cold capture, qualify/qualify-upgrade, sealed evidence, the upgrade receipt module and the soda-qualifier credential path are gone. Release is prepare → stage → smoke-install → promote; promote keeps the `soda-docker-qualified` marker that install-docker.sh checks.
- **Where:** `lib/soda-release-tools/src/qualification.rs:1-991`, `lib/soda-release-tools/src/qualification/input.rs:1-66`, `lib/soda-release-tools/src/qualification/retirement.rs:1-373`, `lib/soda-release-tools/src/qualification/target.rs:1-158`, `lib/soda-release-tools/src/journey.rs:1-571`, `lib/soda-release-tools/src/journey/cold.rs:1-775`, `lib/soda-release-tools/src/journey/cold/retirement.rs:1-536`, `lib/soda-release-tools/src/release_cli.rs:60-115`, `lib/soda-release-tools/src/worker/execution.rs:12-150`, `lib/soda-release-deliver/src/publish/qualification.rs:1-637`, `lib/soda-release-deliver/src/publish/mod.rs:288-555`, `lib/soda-release-deliver/src/publish/mod.rs:627-1130`, `lib/soda-release-deliver/src/publish/mod.rs:1214-1280`, `lib/soda-release-deliver/src/upgrade.rs:1-300`, `scripts/build-qualification.ts:1-149`, `lib/soda-release-tools/README.md:20-200`

**What.** Qualifying a release runs a root-admitted 'collector' as a separate `soda-qualifier` UID under systemd. It takes a root-only authority JSON that hash-pins both executables, build/candidate/payload hashes and a public-input inventory. It also takes a root-owned runtime manifest of per-file SHA-256s and two credentials with private sha256 pins. Ancestor-ownership checks run on every path. The collector produces request.json, a resolved-input snapshot, executed.json and a sealed release-evidence.json, followed by finalize-evidence recovery and consumed-work retirement. On top of that sit `soda-release journey check/status/run/report` (scenario.json/request.json bindings, readiness, closure diagnostics) and `journey cold capture/restore`, which copies the full /etc, /home and /var Docker volumes with xattrs and timestamps. Promotion then re-reads the protected evidence, renders a JSON summary block into the release notes and classifies publication state through next_action/observe_recommendation. scripts/build-qualification.ts builds a pinned driver with its own build.json receipt. The adjacent TS driver in tests/installed/qualification adds ~12.4k lines.

**Simpler.** Keep `stage` (upload a prerelease and compare the asset inventory) and `promote` (`gh release edit --latest`, notes). Replace collect, finalize-evidence, qualify, qualify-upgrade, journey and cold with one smoke script. It installs the downloaded ISO on a VM (or runs the Docker image) and prints pass/fail with the installed revision, which the operator pastes into the notes. Delete the collector UID, authority and runtime manifest, the sealed evidence types, retirement, cold capture/restore, the soda-qualifier credential path in worker/execution.rs, the upgrade summary block that upgrade.rs reads back, and build-qualification.ts.

**Risk.** Only the release operator uses this, and no release has been promoted for users. The loss is a cryptographic paper trail nobody consumes.

**Visibility.** Developer and operator tooling only. Release notes lose the embedded `soda-qualification-summary-v1` JSON block.

**Related.** The TS driver in tests/installed/qualification is about 12.4k lines (collector.ts, main.ts, status.ts, check-*.ts, phases/). Other related places: the soda-qualifier user handling in lib/soda-release-tools/src/worker/execution.rs, the upgrade qualification and summary block in lib/soda-release-deliver/src/upgrade.rs, and the docs in docs/development/release.md, docs/development/native-support.md, docs/development/rc6-qualification-2026-10-25.md and docs/development/factory-remaining-tasks/06-candidate-release-closure.md.

### FEP-1: Workspace page and panel (dual-mode SodaSpaces) is the catch-all the owner removed

- **Subsystem:** Frontend: Workspace, project, applications, terminals, layout, Forgejo and Tailnet pages. **Category:** duplicated-layer. **Verdict:** partially-confirmed.
- **Removable:** ~5200 lines. **Risk:** medium. **Visible change:** UI. **Behaviour-preserving:** no.
- **Status:** deferred to repo-first-ui (Workspace removal).
- **Where:** `frontend/spaces/sodaspaces-workspace.ts:155-2284`, `frontend/spaces/sodaspaces-workspace-setup.ts:1-815`, `frontend/spaces/sodaspaces-workspace-navigation.ts:1-299`, `frontend/spaces/sodaspaces-workspace-inventory.ts:1-391`, `frontend/spaces/sodaspaces-workspace-shell-view.ts:1-187`, `frontend/spaces/sodaspaces-workspace-toolbar-view.ts:1-141`, `frontend/spaces/sodaspaces-workspace-drawer.ts:1-107`, `frontend/spaces/sodaspaces-workspace-factory.ts:1-196`, `frontend/spaces/sodaspaces-workspace-focus.ts:1-101`, `frontend/spaces/soda-spaces-entry.ts:1-81`, `frontend/spaces/sodaspaces-page.ts:1-36`, `frontend/spaces/sodaspaces-workspace.css:1-1113`, `system/containers/extension/extension.json:9-35,52`, `docs/product/spaces.md:1-80`

**What.** The global /workspace page and the persistent 'Workspace' panel are one retained Lit owner (SodaSpaces) with two kinds ('page' and 'native') and 7 views (feed/app/work/plans/terminal/sessions/project). It holds a 3-step request/new-project setup flow (intent, repository picker, configure) with its own router (/workspace/new, /workspace/requests/{id}, ?repository=), a cross-project project directory and 'Saved plans', a sessions list with attention filters, 'Open in drawer', the project-settings host, and per-project view/delivery/scope persistence in localStorage. This is the catch-all the owner decided to remove: everything belongs on the repository page.

**Simpler.** After the repository places land: - Remove the `spaces` page, its new and requests subroutes, and the `workspace` panel from extension.json. - Delete SodaSpaces and the modules only it imports: setup, navigation, inventory, toolbar view, shell view, drawer, focus, workspace-factory, soda-spaces-entry, sodaspaces-page.ts and most of workspace.css. - Keep sodaspaces-attention.ts and session-navigation-view where the surviving terminal modules still use them. - Build the repository Terminals place as a small single-environment component (about 300-500 lines) instead of a `terminalsOnly` mode of SodaSpaces. - Delete the shell-only tests and re-point workspace-intent tests at the repository Request place.

**Risk.** Large deletion that must follow ui/repo-places, which edits sodaspaces-workspace.ts, -setup, -types and -css and depends on SodaSpaces for Terminals. Deleting first would break that branch. If terminals must stay attached across navigation, the repository Terminals entry has to keep its owner alive (ui/repo-places SPEC M2 already plans this). No hardening-branch overlap.

**Visibility.** /workspace, its New request/Request subroutes and the Workspace panel disappear. The same features are reached from each repository's places (owner decision of 28 Oct).

**Related.** ui/repo-places keeps the spaces page and workspace panel in extension.json and adds a terminalsOnly mode to sodaspaces-workspace.ts (+163 lines). Its Terminals place would then mount the whole SodaSpaces owner. docs/product/spaces.md and docs/reference/forgejo.md describe the shared page and panel owner and would need updating. tests/frontend/workspace-*.test.ts covers this shell.

### REL-3: Candidate builds run in a sandbox: a separate UID, an admitted root-owned controller, an SELinux policy, and self-signed media with permits and trust epochs

- **Subsystem:** Release, install, packaging, CI and freshness machinery. **Category:** unneeded-validation-or-digest. **Verdict:** confirmed.
- **Removable:** ~5000 lines. **Risk:** medium. **Visible change:** none. **Behaviour-preserving:** yes.
- **Status:** partly done (`9b348cd0`, the safe first step): media self-signing, deliver/image Trust and Permit, the OCI document writer, the trust-key feature, MediaAuthorityDirectory/--media-authority, candidate-setup's fixture authority, release-trust.json and lab-credentials are gone; packaging inputs are checked against their sha256 inventory only. Open: the worker UID, admitted controller, SELinux worker policy and pinned Go root (the non-identical step), and the rest of candidate-setup.
- **Where:** `lib/soda-release-tools/src/worker/config.rs:1-385`, `lib/soda-release-tools/src/worker/execution.rs:1-712`, `lib/soda-release-tools/src/worker/runtime.rs:1-718`, `tools/candidate-setup/src/:1-1977`, `tools/lab-credentials/src/:1-717`, `system/host/selinux/soda-build-worker.te:1-121`, `lib/soda-release-image/src/media_authentication.rs:1-146`, `lib/soda-release-image/src/media.rs:140-240`, `lib/soda-release-image/src/model/trust.rs:1-138`, `lib/soda-release-deliver/src/model/trust.rs:1-100`, `lib/soda-release-deliver/src/native/sign.rs:1-127`, `lib/soda-release-deliver/src/native/policy.rs:1-335`, `lib/soda-release-tools/src/pipeline.rs:330-368`, `system/host/trust/release-trust.json`

**What.** `soda-build` will not build in the developer's shell. It dispatches to a systemd worker running as `soda-build-worker`, requires the controller binary installed root-owned at /usr/local/lib/soda/soda-build and bind-remapped to /run/soda-build-controller, checks root-owned, symlink-free ancestor chains, requires a pinned Go root and a confining SELinux module, and requires a clean committed checkout. candidate-setup (a Rust port of a shell script, ~2,466 lines with tests) exists only to install all of that, plus a 'fixture media authority' of four P-256 role keys. The build then signs its own host OCI archive and its own packaging-input inventory with those local keys, using Permit{expires: now+3600}, Trust{epoch, not_before, max_age_seconds 60..7d, clock_skew_seconds}, and generates a containers policy.json. It then verifies those signatures in the same pipeline run (authenticate_packaging_inputs then verify_media_inventory). lab-credentials re-implements fixture-authority rotation a second time.

**Simpler.** Let `soda-build` run the pipeline directly as the invoking developer (rootless podman/buildah, with sudo only for the steps that truly need root) from the working checkout. Drop the worker UID, controller admission, SELinux worker policy and pinned Go root. Drop media self-signing, the Trust/Permit models, policy.json generation and the fixture authority, and keep only the sha256 inventory check plus the sha256s already in candidate.json and payload.json. Delete candidate-setup and lab-credentials. The manual runbook classes in lab-credentials become a docs paragraph.

**Risk.** The build pipeline is large. Removing the worker boundary touches the path plumbing (WORKER_SOURCE/WORKER_TOOLS constants) used throughout soda-release-image. The installed host must not depend on the media signatures; it appears not to, since release-trust.json is unreferenced, but that should be confirmed.

**Visibility.** Developer build tooling only. Shipped bytes are the same, minus signature sidecar files.

**Safe first step.** Remove media self-signing, which nothing downstream reads. (a) In lib/soda-release-image/src/media_authentication.rs, delete sign_media_input. Reduce authenticate_packaging_inputs to collect_media_inventory followed by verify_media_inventory, dropping the input-document write and both signing calls. (b) Remove sign_media and verify_copy from the Production trait (lib/soda-release-image/src/foreign.rs) and from its implementation (lib/soda-release-tools/src/pipeline.rs:330-368). (c) Delete lib/soda-release-deliver/src/native/sign.rs and policy.rs, plus verify_copy in native/mod.rs and the merge_policy oracle tests. (d) Delete the deliver and image Trust/Permit models along with MediaAuthority admission in media.rs. (e) Delete system/host/trust/release-trust.json. (f) Remove the MediaAuthorityDirectory config field (worker/config.rs, candidate-setup config.rs) and both fixture_authority.rs files, along with lab-credentials if nothing else remains in it. After this, build outcomes are unchanged, since the sha256 inventory check still runs. The worker UID, controller and SELinux removal is a separate, non-identical step.

**Related.** docs/development/native-support.md (MediaAuthorityDirectory and the worker setup runbook), tools/candidate-setup/README.md, docs/architecture/release.md (packaging-key section), lib/release-inputs (strict DER SPKI admission adapter used for these keys)

### DOC-1: Delete the 4,000-line Factory execution tracker (factory-remaining-tasks.md + six packets), AGENTS.md's binding to it, and its ~35 inbound doc links

- **Subsystem:** Documentation. **Category:** process-machinery. **Verdict:** partially-confirmed.
- **Removable:** ~3900 lines. **Risk:** low. **Visible change:** none. **Behaviour-preserving:** yes.
- **Status:** partly done (`590f35f2`): the six packets and the tracker's R01-R55 index, task map, Canlang recipes and audit/acceptance indexes are deleted (~3,200 lines). The founder-demo checklist stays, including the `#current-product-ux-tasks` section AGENTS.md binds to; AGENTS.md is unchanged. Open: replacing that checklist, which needs the owner (and prod-hardening's production-gaps.md).
- **Where:** `docs/development/factory-remaining-tasks.md:1-1889`, `docs/development/factory-remaining-tasks/01-boundaries.md:1-216`, `docs/development/factory-remaining-tasks/02-prompt-to-pr.md:1-237`, `docs/development/factory-remaining-tasks/03-review-merge-reassessment.md:1-213`, `docs/development/factory-remaining-tasks/04-controls-repair-verification.md:1-612`, `docs/development/factory-remaining-tasks/05-experience-and-development.md:1-571`, `docs/development/factory-remaining-tasks/06-candidate-release-closure.md:1-249`, `AGENTS.md:35`, `docs/development/README.md:5,49-53`, `docs/README.md:69-70`, `docs/development/testing.md:361,409`, `docs/development/rc6-qualification-2026-10-25.md:7`, `docs/guides/founder-demo.md:88`, `docs/development/public-handbook/cutover.md:431,478,567,708,811,814,906`, `docs/guides/canlang-applications.md:347`, `docs/development/public-handbook/coverage-audit.md:321`, `docs/development/ownership.md:37`, `docs/development/library-boundaries.md:5`, `docs/development/factory-workflow-contracts.md:5,1464,2337`, `docs/design/factory-feed-and-ralph-integration.md:70,710,712`, `docs/development/canlang-soda-os-implementation-plan/production-workerd.md:4`, `docs/design/factory-demo-journey.md:7,388`, `docs/design/factory-feed-components.md:19,23`, `docs/design/spaces-ux.md:86`, `docs/design/factory-worker-mcp.md:1164,1196,1202,1527`

**What.** There are three overlapping identifier schemes and five tracker layers: 106 executable tasks (Rxx.y) refining 53 parent outcomes (R01-R55), 12 Canlang outcomes, FJ, FUX and J ids. The six packets each open with the same 9-line disclaimer ('This packet preserves the parent contracts... This documentation restructuring starts no runtime work'). The tracker also carries a parent acceptance index, tool-to-task and visual-coverage maps, owner lanes A/B/C ('historical responsibility labels'), 'Immediate priority: tomorrow's manual testing' (line 907), and acceptance prose like 'Bind the actual executor and next file/resource lease... A Muse assignment still requires its supported continuation and actual receipt' (01-boundaries R01). AGENTS.md:35 makes this tracker the completion boundary for agent work.

**Simpler.** Delete factory-remaining-tasks.md and the factory-remaining-tasks/ directory. Keep one short current list: production-gaps.md from prod-hardening once it merges, or move the still-open FUX items into guides/founder-demo.md as a short checklist. Reword AGENTS.md:35 to point at that list, or drop it now that the 28 Oct demo has happened. Remove or retarget the ~35 inbound links in the other docs, plus the rows in docs/README.md:69-70 and docs/development/README.md:5,49-53.

**Risk.** Pure docs; Git keeps history. Coordinate with prod-hardening, which edits docs/development/README.md and adds production-gaps.md, so pick that file as the survivor rather than writing a third. AGENTS.md:35 changes agent supervision behaviour for the founder-demo waker, so the owner should agree the replacement checklist.

**Visibility.** Docs and coding-agent instructions only. Factory worker prompts are unaffected.

**Safe first step.** Delete the six packet files in docs/development/factory-remaining-tasks/ (2098 lines). Retarget their four direct inbound links (cutover.md:567,708,811,814,906 and factory-workflow-contracts.md:1464) to the relevant workflow-contract or design sections, or drop them. AGENTS.md:35 and the FUX checklist in factory-remaining-tasks.md stay untouched until the owner agrees on a replacement checklist. Runtime behaviour is unchanged.

**Related.** The prod-hardening branch adds another tracker, docs/development/production-gaps.md. docs/development/rc6-qualification-2026-10-25.md and docs/development/public-handbook/coverage-audit.md are also tracker-like and link here. docs/design/factory-demo-journey.md:388 repeats the founder-demo checklist pointer.

### JDG-1: After every merge, the repository is fenced behind a JEV reassessment gate

- **Subsystem:** Completion, reassessment, review and JEV judgments. **Category:** too-many-states-or-budgets. **Verdict:** confirmed.
- **Removable:** ~3800 lines. **Risk:** medium. **Visible change:** UI, model tools, DB schema. **Behaviour-preserving:** no.
- **Status:** owner decision; JEV call kept: the reassess/continue answer decides whether a coder run opens or the fence releases, so it is not a dead answer. The fence, double inventory collection and cycle machinery around it stay an owner decision.
- **Where:** `internal/factory/control/completion_drive.go:14-134,253-643`, `internal/factory/control/completion_decision.go:1-320`, `internal/factory/control/completion_fence.go:1-124`, `internal/factory/control/completion_coder.go:1-61`, `internal/factory/control/completion_proposal.go:13-232`, `internal/factory/control/completion_retry.go:1-68`, `internal/factory/control/completion_adoption.go:67-254`, `internal/factory/control/reassessment_adoption.go:941-1078`, `internal/factory/reassessment.go:1-100,300-720,1184-1238`, `internal/factory/completion_retry.go:1-39`, `internal/store/factory_reassessments.go:1-767`, `internal/store/factory_reassessment_allowances.go:1-308`, `internal/store/factory_completion_retry.go:1-60`, `internal/store/factory_reassessment_activation.go:1-82`, `internal/web/api/factory_journey_readback.go:581-660`, `frontend/spaces/sodaspaces-factory-completion-retry.ts:1-205`, `internal/factory/control/dispatch.go:339-348,435`, `internal/factory/control/dispatch_attempt.go:198-204`

**What.** Every verified merge opens a reassessment cycle (factory_reassessments.go:186-228). The repository's new code admissions stay fenced until that cycle is released or superseded (checkCompletionFenceTx :237-250, checkDispatchFence, excludeFencedCandidates). The cycle runs like this. (1) Collect the entire open-issue inventory through a revision-bracketed collector and seal it with digests. (2) Ask JEV post-completion-reassessment-v1 whether to reassess or continue. It asks even when the inventory is empty (reassessment.go:3-5). (3) On continue, collect the whole inventory again to prove nothing changed, then release. (4) On reassess, admit a coder run, ask JEV again (post-completion-proposal-v1), then apply the result through a serial adoption journal that re-collects the inventory at every step. The machinery around it: 8 cycle states, a 2-call/1-run ledger, a separate time-only attempt allowance, a retry command that creates successor cycles, and a UI retry button.

**Simpler.** Keep recording the completion fact in factory_completions. Delete the reassessment cycle, the fence and both post-completion JEV purposes. On a verified merge, call DeletePriorityScan and EnqueueReadinessRootEvent right away; the continue release already does exactly this at completion_drive.go:500-504. If replanning after a merge is wanted, start an ordinary non-blocking planning run through the intent-revision path. Keep the pieces that stall repair and delivery repair share: the bracket collector, CompletionLocal and the graph-repair journal.

**Risk.** This changes product behaviour: there is no automatic post-merge reassessment any more. Several pieces are shared with stall repair and delivery repair (repair_*.go, delivery_repair_*.go) and must stay until those flows are simplified: the bracket collector, CompletionLocal/CompletionMaterial, graph-repair preflight and adoption (ReassessmentJournal, graphRepairInputs, matchGraphRepairEffects) and the reassessment_scope MCP tool. Feed SQL joins factory_reassessments (factory_feed.go:97-119, factory_feed_contribution.go:90). Dispatch packet admission checks the fence (factory_dispatch_packet.go:221 and 660-720).

**Visibility.** Gone: the completion gates and judgments in the work journey, the 'Retry completion review' control, reassessment_pending waits and reassessment Feed records. Completions no longer issue the reassess-phase prompt or expect the typed reassessment report. The factory_reassessments table can be dropped.

**Related.** The same receipt pattern (a reserved factory_priority_judgments row with NULL data that later reservations refuse) appears in internal/store/factory_intents.go:389-398, factory_attempt_allowances.go:1223-1240 and factory_context_selection.go:50, so a crash can strand those reservations the same way. The fence is also checked in factory_dispatch_packet.go. Feed SQL joins factory_reassessments in factory_feed.go and factory_feed_contribution.go.

### HPL-6: soda-project-maintain has its own release, recover, rollback and reconcile system with journals and qualification modes

- **Subsystem:** Rust host platform: daemon, terminals, project, application, tailnet (lib/host non-factory, soda-project-maintain, forgejo/pg/console helpers, soda-tailnet, unix-http, wire-time). **Category:** process-machinery. **Verdict:** confirmed.
- **Removable:** ~3400 lines. **Risk:** medium. **Visible change:** none. **Behaviour-preserving:** no.
- **Status:** done (`cbc4918b`): the release/release-reconcile subcommands, their six modules, the reconcile unit and its candidate.json/prepare.rs entries and the operation.json marker checks are gone; operator docs now say upgrade by reinstall or the native image update.
- **Where:** `cmd/soda-project-maintain/src/release_operation.rs:1-903`, `cmd/soda-project-maintain/src/release_stage.rs:1-593`, `cmd/soda-project-maintain/src/release_target.rs:1-531`, `cmd/soda-project-maintain/src/release_extensions.rs:1-649`, `cmd/soda-project-maintain/src/release_inventory.rs:1-230`, `cmd/soda-project-maintain/src/importer_custody.rs:1-412`, `cmd/soda-project-maintain/src/main.rs:43-57`, `cmd/soda-project-maintain/README.md:26-31,180-191`, `system/host/services/soda-project-release-reconcile.service`, `system/host/provisioning/candidate.json:21`, `lib/soda-release-image/src/prepare.rs:257`, `lib/host/src/project/admission.rs:55`, `cmd/soda-project-maintain/src/terminal_support.rs:333`, `docs/public/50-Operate/70-updates-and-fallback.md`, `docs/public/50-Operate/80-recovery.md`

**What.** An in-place host upgrade path. It takes a downloaded GitHub release, requires 'its exact native qualification receipt', and writes a 'format-2 journal' (operation.json/last-operation) that freezes Fountain intent and component pins. It supports --recover and --rollback, plus --qualification-fixture/--qualification-public modes that cannot change during recovery. A boot service (release-reconcile) verifies the active pair, retires recovery copies and reopens admission, using exit code 75 for 'core committed but extension readiness unknown'. It also checks extension cache receipts against catalog rows.

**Simpler.** Delete the release and release-reconcile subcommands in main.rs. Delete the release_operation, release_stage, release_target, release_extensions, release_inventory and importer_custody modules. Delete the reconcile service, its entries in candidate.json and soda-release-image prepare.rs, and the operation.json marker checks in host admission and terminal_support. Keep release.rs and release_wire.rs, which startup config and canlang use. Upgrade by reinstalling or by the plain bootc image update, and update the operator docs to match.

**Risk.** Coordinate with the release-tooling finder (lib/soda-release-*). The docs sections on explicit release maintenance go with it. Check that project::admission's operation.json marker logic has no other writer.

**Visibility.** Operator CLI and docs only. Nothing in the web UI.

**Related.** lib/soda-release-image/src/prepare.rs installs the reconcile unit. lib/host/src/project/admission.rs reads operation.json. lib/soda-release-* and tools/release-assets are the release-tooling finding. tools/acceptance has qualification reporting. docs/public/50-Operate/{70-updates-and-fallback,80-recovery}.md, docs/guides/operator-setup.md#explicit-release-maintenance and docs/development/release.md describe this procedure.

### DOC-3: Release and qualification docs (~4,800 lines) describe heavy release machinery, including a 1,950-line design and J01-J26 implementation plan inside the workflow doc

- **Subsystem:** Documentation. **Category:** process-machinery. **Verdict:** confirmed.
- **Removable:** ~3200 lines. **Risk:** low. **Visible change:** none. **Behaviour-preserving:** yes.
- **Status:** done (`91248f8a`, with the native-support historical audit removed in `9b348cd0`): release.md is the operator how-to (2,529 → ~400 lines), the RC6/RC7 record and freshness.md's run log are deleted, architecture/release.md lost the trust-key, OCI admission and Docker qualification detail, and lib/soda-release-tools/README.md is a command list.
- **Where:** `docs/development/release.md:34-1983`, `docs/development/release.md:1984-2529`, `docs/development/rc6-qualification-2026-10-25.md:1-845`, `docs/architecture/release.md:1-334`, `docs/development/native-support.md:547-573`, `lib/soda-release-tools/README.md:1-358`, `tools/release-assets/README.md:1-142`, `docs/development/README.md:43`, `docs/design/factory-demo-journey.md:47`

**What.** release.md is 2,529 lines. Lines 34-1983 are 'Normalize the qualification procedure', a design with alternatives, a resource lifecycle and a '26 tasks... living implementation task list' (J01-J26) for `soda-release journey check/status/run/report`. That covers protected collectors, sealing, receipt reports, resource retirement, cold-recovery baselines, journey build.json hashes, 'Each source-to-target pair requires its own native qualification receipt', and 'Promote the unchanged qualified digest'. The rc6/rc7 record (845 lines) is one attempt's run log ('It exists so another agent can find and resume this work'). architecture/release.md adds trust-key admission, OCI metadata admission and registry recommendation pointers. native-support.md ends with a 'Historical audit reference... not a current checklist'.

**Simpler.** Delete release.md lines 34-1983 (the design and the J01-J26 plan), rc6-qualification-2026-10-25.md and the native-support historical audit (547-573). Remove the links to the rc6 record in docs/development/README.md, docs/design/factory-demo-journey.md and release.md:9. Condense the guided operator flow and the rest of release.md to about 150 lines. Keep architecture/release.md, because many docs link to its anchors. Shrink it to what ships (architectures, public distribution, host-update ownership) and drop the trust-key and OCI admission detail. Reduce lib/soda-release-tools/README.md to a command list. If the journey/qualification code is cut, delete its docs with it.

**Risk.** Docs only. The Guided release operator flow (lines 1984-2259) may be the only procedure the owner actually runs, so keep it, condensed. The release code itself is another finder's call.

**Visibility.** Developer docs only.

**Safe first step.** Delete docs/development/rc6-qualification-2026-10-25.md and remove its links at docs/development/README.md:43, docs/design/factory-demo-journey.md:47 and docs/development/release.md:9. Delete docs/development/release.md lines 34-1983 (the section 'Normalize the qualification procedure' through the end of the J01-J26 list) and docs/development/native-support.md lines 547-573 (the historical audit). Fix any anchors that pointed into the deleted sections. Code is untouched.

**Related.** docs/development/public-handbook/coverage-audit.md and source-audit.md are similar audit and process docs that cite architecture/release.md line ranges. The underlying code is lib/soda-release-tools/src/journey*, qualification.rs and scripts/build-qualification.ts.

### FEF-5: The work overview, plan, direction-revision and detail views exist only in the Workspace, duplicate the Feed and project-work, and their repository-only branch is unreachable

- **Subsystem:** Frontend: Factory settings, work, progress, request, intent screens. **Category:** duplicated-layer. **Verdict:** confirmed.
- **Removable:** ~3000 lines. **Risk:** medium. **Visible change:** UI. **Behaviour-preserving:** yes.
- **Status:** deferred to repo-first-ui (Workspace removal).
- **Where:** `frontend/spaces/sodaspaces-factory-work.ts:166-753,1046-1425`, `frontend/spaces/sodaspaces-factory-progress-view.ts:79-780,1004-1297`, `frontend/spaces/sodaspaces-factory-work-view.ts:33-565`, `frontend/spaces/sodaspaces-factory-progress-response.ts:62-437,670-693,820-1136`, `frontend/spaces/sodaspaces-factory-work-response.ts:561-822`, `frontend/spaces/sodaspaces-workspace-setup.ts:312-319,333-397,465-490,531`, `frontend/spaces/sodaspaces-project-work.ts:410-426`, `system/containers/extension/extension.json:7-38`, `tests/frontend/factory-work-controls.test.ts`, `tests/frontend/fixtures/factory-work-fixture.ts`

**What.** Four elements (soda-factory-work-overview, -work-plan, -direction-revision and -work-detail with its intervention controls), each with model builders, renderers and parsers. They are mounted only by the Workspace setup panel and coordinated through a 'soda-factory-overview-scope' CustomEvent, which the Workspace validates again even though the same process produced it (overviewScopeDetail, sameScope, childScopeFromReading). The detail view embeds the same four children (work-journey, requirements-acceptance, completion-retry, context-resolution) that SodaProjectWork.renderRequirements also composes.

**Simpler.** Do not port these views to the repository page. The single work view is the Feed: status per issue, links to the Forgejo issue and PR, and Stop and Retry on the run card. Pause/Resume becomes the Factory On/Off switch. Delete the four elements, the overview-scope event bus, their describe/render/parse code, factory-work-fixture and factory-work-controls.test.ts. First move nativeIssueLink and nativePullLink, which the Feed uses, to sodaspaces-api.ts.

**Risk.** Takeover and project-stop lose their only UI, and Stop/Retry must land on the Feed run card before the detail view goes. This line count overlaps those of FEF-4, FEF-8 and FEF-9.

**Visibility.** The Workspace work panels disappear, consistent with the no-Workspace decision. The unreachable repository-only overview can be deleted with no visible change.

**Safe first step.** Delete the unreachable repository-only overview path. In frontend/spaces/sodaspaces-factory-work.ts, remove applyRepositoryView, readOverviewSummary and readOverview (about lines 229-236 and 282-322), make intentID required on WorkOverviewContext, and drop the renderWorkOverview fallback in render(). In frontend/spaces/sodaspaces-factory-progress-view.ts, remove WorkOverviewModel/Input, describeWorkOverview, renderWorkOverview, countIssues and dependencyRows plus their helpers (about lines 79-259). Then remove factoryRepositorySummary from the progress-response imports if nothing else uses it. setupOverviewContext and the fixture always pass intentID, so nothing observable changes.

**Related.** The prod-hardening branch (the real name of the 'hardening/production-gaps' branch) adds allowanceText to describeWorkOverview and renderWorkOverview in sodaspaces-factory-progress-view.ts, and that copy can never show. SodaProjectWork (sodaspaces-project-work.ts) is also mounted only by the Workspace and composes the same four child elements. The line count overlaps FEF-4, FEF-8 and FEF-9.

### PLN-3: Explore builds a template 'Proposed work' list that first generation never builds; the planner candidate is what gets built. The frontend keeps a large shelf/history layer to keep the template list consistent

- **Subsystem:** Intent to plan (Product Lead, Architect, research, decomposition). **Category:** duplicated-layer. **Verdict:** partially-confirmed.
- **Removable:** ~3000 lines. **Risk:** medium. **Visible change:** UI, model tools. **Behaviour-preserving:** no.
- **Status:** owner decision; coordinate with prod-hardening.
- **Where:** `internal/factory/intent_proposal.go:183-221`, `internal/web/api/factory_intent_explore.go:62-80,145-204,273-300`, `internal/factory/control/intent_decomposition.go:21-46,130-147`, `internal/factory/control/intent_generation.go:361,502-503`, `internal/factory/control/intent_bound.go:83-94`, `internal/store/factory_context_current.go:229-238`, `internal/web/api/factory_request_view.go:89-112,265-277`, `internal/web/api/factory_revision_view.go:283-312`, `frontend/spaces/sodaspaces-intent.ts:312-352,404-412,487-573,912-2072`, `frontend/spaces/sodaspaces-intent-view.ts:719-730`, `frontend/spaces/sodaspaces-intent-response.ts:720-741`, `tests/frontend/workspace-intent.test.ts`

**What.** Explore derives a template task list on the IntentRevision. ProposeIntent produces one or two boilerplate tasks per accepted idea, and a JEV 'product-approaches' call chooses complete_first, incremental or clarify. That list is what the founder sees as 'Proposed work', what accepted-revision validation requires (len(Tasks)>0), and what the planner prompt and factory.context carry. Generate, however, builds issues only from the Muse planner's chosen candidate. On top of that, the frontend keeps the template list consistent across declines: shelved tasks, pre-prune dependency-edge records, lineage remapping, 8-hop cross-device saved-history recovery and narrowed-scope gates, persisted in localStorage.

**Simpler.** Explore proposes ideas and questions only, and Plan produces the tasks. Show the chosen candidate's tasks in the plan status, and on acceptance store them on the revision, the way completion and repair successors already store their graph in Revision.Tasks. Remove ProposeIntent's task derivation, Approach, the product-approaches JEV call, the 104-ID minting and the frontend shelf/lineage/history/narrowed-scope machinery. Keep IntentRevision.Tasks, filled with real (submitted) tasks. Then requestMappingDTO and revisionGenerationDTO become correct automatically.

**Risk.** Several places read revision.Tasks: completion_material.go:346, completion_adoption.go:274, store/factory_feed_product_change.go:249 and the app-verification fallbacks. Point them at the submitted tasks. The Feed product-change successor flow (FJ33) must keep working. The line count includes about 1,900 test lines.

**Visibility.** 'Proposed work' becomes the planner's real task list. Declined-work and saved-history recovery panels and the narrowed-scope gate disappear. The planner prompt and factory.context stop carrying template tasks and approach.

**Related.** Completion and repair successor revisions use Revision.Tasks as the real submitted graph (internal/factory/control/completion_adoption.go:167,229; repair_adoption.go:88,152; completion_material.go:376). App verification falls back between revision tasks and submitted tasks (internal/store/factory_app_verification.go:128-133). The Feed product-change flow reads revision tasks (store/factory_feed_product_change.go).

### BND-8: About 6.8k lines of frontend response parsers re-validate SodaOS's own JSON and fail the whole view on any mismatch

- **Subsystem:** Boundary machinery: strict JSON, digests, receipts, revision checks (cross-cutting). **Category:** duplicated-layer. **Verdict:** partially-confirmed.
- **Removable:** ~2800 lines. **Risk:** medium. **Visible change:** UI. **Behaviour-preserving:** yes.
- **Status:** owner decision; coordinate with prod-hardening.
- **Where:** `frontend/spaces/sodaspaces-api.ts:36-46`, `frontend/spaces/sodaspaces-factory-work-response.ts`, `frontend/spaces/sodaspaces-factory-progress-response.ts`, `frontend/spaces/sodaspaces-factory-feed-response.ts`, `frontend/spaces/sodaspaces-intent-response.ts`, `frontend/spaces/sodaspaces-factory-response.ts`, `frontend/spaces/sodaspaces-factory-acceptance-response.ts`, `frontend/spaces/sodaspaces-factory-work-journey-response.ts`, `frontend/spaces/sodaspaces-factory-repair.ts:141`

**What.** Sixteen *-response.ts files re-check every field the Go API returns: 32-, 40- and 64-hex regexes, bounded lengths, reason-code regexes, exact identity echoes such as `receipt.command_id === change.command_id && receipt.revision === change.expected_revision + 1`. Any deviation calls check(), which throws 'Invalid or mismatched Soda response', and the whole panel fails. These parsers are a second, hand-written copy of the Go DTOs. Every backend change has to be mirrored here. The hardening branch had to add allowance parsing to work-response.ts and null-array tolerance in the store for the same reason.

**Simpler.** Keep the exported TS interfaces and replace parse functions with typed casts after readSodaJSON, keeping `?? []`/defaults only where views branch on a field. Delete regex/length/allow-list checks and echo assertions (including factory-repair.ts:141). Each parser that renames or defaults fields needs its own pass, not a blind delete. Optionally generate the TS types from Go DTOs later.

**Risk.** The parsers also reshape some data (renaming and defaulting), so each file needs a pass rather than a blind delete. Frontend tests that cover parser refusals would be deleted along with the parsers.

**Visibility.** For valid data the UI renders the same. Malformed or unexpected backend data shows partially instead of as an error panel.

**Safe first step.** Merge the identical helpers into sodaspaces-api.ts and import them: digest (/^[a-f0-9]{64}$/), text(value, limit), counter and decimalID. They are currently redefined in sodaspaces-factory-work-response.ts, -progress-response.ts, -work-journey-response.ts, -intent-response.ts, -acceptance-response.ts and sodaspaces-application-response.ts (sha). Behaviour is identical. Removing the actual checks changes how malformed data is shown, so that step is not strictly behaviour-preserving.

**Related.** frontend/spaces/sodaspaces-factory-repair.ts (receipt echo check). About 28 frontend files carry hex-shape regexes, including non-response component files. The hardening branch (prod-hardening) adds frontend/spaces/sodaspaces-factory-allowance.ts and +4 lines in work-response.ts in the same style. tests/frontend/*.test.ts exercise the parser refusals.

### DLV-5: Optional lifecycle effects (issue label/assignee, PR reviewer request, completion note) are a ~2,500-line subsystem the installer never enables

- **Subsystem:** Publication, merge, correction, repair, acceptance, verification, lifecycle. **Category:** speculative-abstraction. **Verdict:** confirmed.
- **Removable:** ~2500 lines. **Risk:** low. **Visible change:** UI, DB schema. **Behaviour-preserving:** yes.
- **Status:** done (`fcc7b413`).
- **Where:** `internal/factory/lifecycle_action.go:1-430`, `internal/factory/lifecycle_action_config.go:1-98`, `internal/factory/control/lifecycle_consume.go:1-606`, `internal/factory/control/lifecycle_capabilities.go:1-79`, `internal/factory/control/coordinator.go:90`, `internal/store/factory_lifecycle_actions.go:1-404`, `internal/web/api/factory_lifecycle_actions.go:1-291`, `internal/web/api/extension_native.go:222-224`, `internal/factory/control/publication.go:103-107`, `internal/factory/control/publication_withdraw.go:289-293`, `internal/factory/control/merge.go:124-127`, `internal/factory/control/merge_evidence.go:237-241`, `internal/factory/control/intent_generation.go:238-239`, `internal/web/server.go:41,80-82,109-111,197-219`, `internal/config/config.go:65-68,213-219`, `internal/store/schema.go:75-78,382-384`, `cmd/soda-identity/src/schema.rs:70-73,349-351`, `cmd/soda-setup/src/config.rs:33,69,101,126-127`, `cmd/soda-setup/src/factory_enrollment.rs:211,1455`, `internal/forgejo/lifecycle.go`, `internal/forgejo/lifecycle_capabilities.go`, `frontend/spaces/sodaspaces-factory-progress-view.ts:782-935`, `frontend/spaces/sodaspaces-factory-progress-response.ts:496-670`, `frontend/spaces/sodaspaces-factory-work.ts (lifecycle effect state/load/save, ~:891,:965)`

**What.** Three cosmetic effects carry a lot of machinery: a capability handshake registry with per-installation markers refreshed at startup; a revisioned per-event config whose enable step is gated on that capability; a durable action journal with per-target results (pending, delivered, failed, uncertain, denied); a two-attempt budget; claim and enroll steps; deny-reason classification; lookup reconciliation; an 'unavailable executor' journaling path; and a drain that runs in every Publish and Merge pass.

**Simpler.** Delete the subsystem: its three tables (Go schema and the Rust soda-identity mirror), the config field (Go and soda-setup), the capability handshake source and its startup/bootstrap refresh, the Coordinator.Lifecycle hooks in the Publish, Merge and intent-generation passes, the three lifecycle routes, and the 'Optional effects' UI in the progress view and work space, together with their tests. If one effect is wanted later, such as a PR reviewer request, make one Forgejo REST call from finishPublication with no journal.

**Risk.** Default installs never execute any of it. The handshake refresh at startup becomes unnecessary.

**Visibility.** The 'Optional effects' section in the factory progress/settings UI and its GET/PUT lifecycle API endpoints go away.

**Safe first step.** Remove the executor path, which is unreachable on every soda-setup install because the credential is never created. In order: delete the five nil-guarded `if c.Lifecycle != nil {...}` blocks (publication.go:105-107, publication_withdraw.go:291-293, merge.go:125-127, merge_evidence.go:239-241, intent_generation.go:238-240). Then delete the Coordinator.Lifecycle field and LifecycleExecutor interface usage, the server.go:109-111 wiring, internal/forgejo/lifecycle.go, and internal/factory/control/lifecycle_consume.go with its test. Then make lifecycleExecutorReady always return false. On a default install the API, UI and Publish/Merge outcomes stay identical. The UI, routes, tables and handshake refresh go in a second, user-visible step.

**Related.** The table DDL is duplicated in cmd/soda-identity/src/schema.rs. The config field is plumbed through cmd/soda-setup (config.rs, factory_enrollment.rs) and scripts/fixtures/portcontracts/dashboard_config_vectors.json. Frontend tests in tests/frontend/factory-work-controls.test.ts and factory-settings-controls.test.ts, plus docs (docs/reference/factory.md, api.md, forgejo.md, guides/operator-setup.md), reference it. The other optional credential-gated actors (Reviews, Resolutions, Merges, Graph) follow the same 'only if a credential file exists' wiring in server.go and may deserve the same check.

### WEB-2: Dormant 'optional lifecycle effects': needs a credential no setup path creates, yet ships ~2,500 lines and a startup capability handshake

- **Subsystem:** Web API and view models (non-MCP, non-Feed, non-extension): internal/web/*.go, internal/web/auth/**, internal/web/api/*.go excluding factory_mcp*, factory_feed*, extension*. **Category:** unused-configuration. **Verdict:** confirmed.
- **Removable:** ~2500 lines. **Risk:** low. **Visible change:** UI. **Behaviour-preserving:** yes.
- **Status:** done (`fcc7b413`).
- **Where:** `internal/web/api/factory_lifecycle_actions.go:1-291`, `internal/web/api/extension_native.go:222-224`, `internal/web/server.go:41-43,80-83,109-111,197,201-221`, `internal/config/config.go:68,213-219`, `cmd/soda-setup/src/config.rs:33,69,101,126-127`, `cmd/soda-setup/src/factory_enrollment.rs:211,1455`, `cmd/soda-setup/README.md:219`, `internal/factory/control/lifecycle_consume.go:1-606`, `internal/factory/control/lifecycle_capabilities.go:1-79`, `internal/factory/control/intent_generation.go:238-239`, `internal/factory/control/merge.go:125-126`, `internal/factory/control/merge_evidence.go:239-240`, `internal/factory/control/publication.go:105-106`, `internal/factory/control/publication_withdraw.go:291-292`, `internal/factory/lifecycle_action.go:1-430`, `internal/factory/lifecycle_action_config.go:1-98`, `internal/store/factory_lifecycle_actions.go:1-404`, `internal/store/factory_priority_inventory.go:319,639-667`, `internal/forgejo/lifecycle.go:1-317`, `internal/forgejo/lifecycle_capabilities.go:1-72`, `frontend/spaces/sodaspaces-factory-work.ts:755-1045`, `frontend/spaces/sodaspaces-factory-progress-response.ts:500-600`, `frontend/spaces/sodaspaces-factory-progress-view.ts:~800-860 (lifecycle describe/submitter text)`

**What.** Optional native effects per event: label and assignee on issue_created, reviewers on pr_created, a completion note on verified_merge. The feature has stored configs, durable action rows with attempt budgets, capability markers proved by a native handshake, an enrolled-submitter lookup, three routes and a Work-page UI.

**Simpler.** Delete the feature: - The three lifecycle routes and their handlers. - Coordinator.Lifecycle and its five nil-guarded calls. - Server.capabilities, the refreshing flag, the OnBootstrap hook, refreshLifecycleCapabilities, CapabilityHandshakeSource and ReadOperations. - The forgejo_lifecycle_credential_file config key, its validation and the soda-setup plumbing and README line. - The factory, store, forgejo and control lifecycle-action files and their tests. - priorityBatchLifecycle in the priority inventory. - The Work-page effects section and the lifecycle parsers and text in progress-response.ts and progress-view.ts. Leave the factory_lifecycle_actions and factory_lifecycle_capabilities tables and their schema probes in place, so no migration is needed.

**Risk.** First confirm the founder box has no hand-added forgejo_lifecycle_credential_file. Nothing else reads lifecycle configs, lifecycle actions or capability markers.

**Visibility.** The Work page loses the effects section, which is inert today ('No lifecycle executor is configured yet'). Settings sprawl shrinks.

**Safe first step.** Remove the startup and bootstrap handshake, which has no observable effect beyond log lines: 1. In internal/web/server.go, delete the capabilities and refreshing fields, the s.capabilities assignment and the background.OnBootstrap hook (lines 80-83), the s.refreshLifecycleCapabilities(ctx) call in Start (line 197), and refreshLifecycleCapabilities itself (lines 201-221). 2. Delete internal/forgejo/lifecycle_capabilities.go and its test, and internal/factory/control/lifecycle_capabilities.go and its test. 3. In factory_lifecycle_actions.go, have the two handlers that read CapabilityProved (lines 153 and 254) treat the capability as not proved. In practice nothing changes: no credential exists, so lifecycleExecutorReady is false, the GET still reports executor_ready=false and enabling still returns 409 executor_unavailable. Before any of this, confirm the founder box config has no hand-added forgejo_lifecycle_credential_file.

**Related.** The same pattern recurs in the other optional per-role credentials that soda-setup only preserves and never creates. Each is a credential file in internal/config/config.go that turns a Coordinator field on in internal/web/server.go: ForgejoReviewCredentialFile, ForgejoResolveCredentialFile, ForgejoMergeCredentialFile, and ForgejoGraphCredentialFile with ForgejoGraphActorID. Each is worth checking for the same 'no setup path creates it' dormancy. factory_priority_inventory.go also folds lifecycle rows into its digest.

### DSP-1: JEV context-bundle-v1 call gates every launch, and a failure holds the run until its deadline

- **Subsystem:** Factory dispatch and run start. **Category:** speculative-abstraction. **Verdict:** confirmed.
- **Removable:** ~2000 lines. **Risk:** medium. **Visible change:** UI, model tools. **Behaviour-preserving:** yes.
- **Status:** done (`72c5afe7`, `c62f6ec1`): the context classifier advises and retries; it never holds the run.
- **Where:** `internal/factory/control/dispatch_launch.go:690-764`, `internal/factory/control/dispatch_launch.go:250`, `internal/factory/control/context_bundle_prepare.go:11-69`, `internal/factory/control/context_bundle_advice.go:1-149`, `internal/factory/control/context_bundle_state.go:1-238`, `internal/factory/control/context_resolution.go:1-109`, `internal/factory/control/dispatch_context_recovery.go:1-200`, `internal/factory/control/dispatch_recovery.go:72`, `internal/factory/control/settle.go:220`, `internal/factory/control/judgment_advice.go:49-116`, `internal/factory/context_bundle.go:1-75`, `internal/factory/context_resolution.go:1-67`, `internal/factory/run.go:21-23`, `internal/factory/mcp_submission.go:37-40`, `internal/store/factory_context_selection.go:15-60`, `internal/store/factory_context_resolution.go:1-101`, `internal/store/factory_implementation_retry.go:1-309`, `internal/factory/control/lifecycle_retry.go:37-53`, `internal/factory/control/dispatch_attempt.go:112-125`, `internal/web/api/factory_mcp.go:761-782`, `internal/web/api/factory_mcp.go:844-941`, `internal/web/api/factory_mcp_context_gate.go:10-35`, `internal/web/api/factory_views.go:111,171-178`, `internal/web/api/factory_lifecycle.go:106,158-172`, `frontend/spaces/sodaspaces-factory-context-resolution.ts:1-249`, `internal/acceptance/factory_context_status.go:1-338`

**What.** MCPKey is always loaded in production (web/server.go:189-193). Before every host launch, launchTail therefore marks the run ContextPending, stages the MCP capability and calls PrepareContextBundleSelection. That makes one JEV choice call (context-bundle-v1) over {current, attempt, lineage, snapshot_hold}. Anything except a confident eligible answer (probability and confidence both at least 0.80) returns a context_selection_hold wait, and FactoryLaunch is never called. The choice only decides whether factory.read includes attempt.run_history (opaque run IDs) and lineage.parent_assignment/parent_outcome/parent_reason (buildReadBody, factory_mcp.go:882-941). Eligibility is already decided in code (BundleEligibility). Built around it are: a reservation that writes context_selection_snapshot into the capability; a receipt that is replayed and re-validated on each pass and on every MCP read and submission; a pending-launch recovery path that rebuilds the launch from the packet; a startup preservation path; an owner 'use current context' command with a semantic digest; and a separate CommandRetryImplementation retry kind. That retry kind exists only to replace an implementation child that was stopped while ContextPending (stop-before-start).

**Simpler.** Delete the selection. launchTail produces the capability, derives the secret and launches; there is no ContextPending and no PrepareContextBundleSelection. factory.read always serves attempt.run_history when RunHistory is non-empty and lineage when ContextParentVisible is true; drop the context_selection and bundle fields. requireCurrentWork keeps only the deterministic current-context completeness check; optionally run the same check before launch and return a named wait. Remove context_bundle_{advice,prepare,state}.go, control/context_resolution.go, dispatch_context_recovery.go, Run.ContextPending and its branches, factory/context_resolution.go, MCPCapability.ContextSelectionSnapshot/ContextCurrentResolution, store/factory_context_selection.go and factory_context_resolution.go, the use_current_context action/API/frontend file, CommandRetryImplementation (store/factory_implementation_retry.go, the lifecycle_retry.go branch, PendingImplementationRetry, RecordImplementationRetryPacket, validateImplementationRetryChildTx) and the pending-context installed probe, plus their tests. Update the D14 docs.

**Risk.** The deletion is broad (control, store, api, frontend, acceptance probe) but mechanical. The behaviour change is the point: no LLM gate, eligible history fields always served, runs never held for context. No migration is needed because the fields live inside JSON data columns. Remove the context_pending SQL in the implementation-retry query together with the retry kind.

**Visibility.** The run page loses the context hold, the 'use current context' control and context_selection status, and the special implementation Retry becomes the ordinary Retry. factory.read/context drops context_selection{bundle,action,reason_code,fingerprint} and bundle, always includes eligible attempt/lineage, and stops refusing submissions with 'context selection holds new work'.

**Safe first step.** In internal/factory/control/context_bundle_advice.go, delete SelectContextBundle, ContextBundleFingerprint and ContextBundleRequest. Their only references are inside that file and in tests. Delete the tests that cover only them. Keep selectContextBundleState and ContextBundleFromReceipt, which production uses. The full gate removal changes behaviour and is the real fix.

**Related.** The same pattern of a single reserved, paid JEV purpose call whose failed receipt is replayed forever appears wherever AdvisePurpose/replayPurposeJudgment is used (judgment_advice.go callers). Each of those purposes may also hold permanently after one failed consultation.

### HPL-1: Hand-written JSON wire layer copying Go quirks, though the Go host daemon has been deleted

- **Subsystem:** Rust host platform: daemon, terminals, project, application, tailnet (lib/host non-factory, soda-project-maintain, forgejo/pg/console helpers, soda-tailnet, unix-http, wire-time). **Category:** duplicated-layer. **Verdict:** confirmed.
- **Removable:** ~2000 lines. **Risk:** medium. **Visible change:** none. **Behaviour-preserving:** yes.
- **Status:** open (behaviour-preserving).
- **Where:** `lib/host/src/lib.rs:1-5`, `lib/host/src/json/mod.rs:1-276`, `lib/host/src/terminal/lease.rs:1-449`, `lib/host/src/terminal/factory/run.rs:93-350`, `lib/host/src/terminal/wire.rs`, `lib/host/src/terminal/binding.rs:35-208`, `lib/host/src/terminal/identity.rs`, `lib/host/src/domain/mod.rs:143-168`, `lib/host/src/domain/account.rs`, `lib/host/src/domain/profile.rs`, `lib/host/src/daemon/config.rs`, `lib/host/src/dbackend/mod.rs:30-110`, `lib/host/src/dbackend/tailnet.rs:75-200`, `lib/host/src/account/mod.rs`, `lib/host/src/tcontrol_wire.rs`, `lib/host/src/tcontrol_policy.rs:107-345`, `cmd/soda-project-maintain/src/config_wire.rs:1-197`, `lib/host/tests/tcontrol_oracle.rs`, `lib/host/tests/muse_serve_oracle.rs`, `lib/host/tests/tcodex_ops_oracle.rs`

**What.** Most request and response types have a hand-written serde Visitor. These copy Go encoding/json quirks: case-insensitive keys (156 uses of to_ascii_lowercase/eq_ignore_ascii_case outside factory), JSON null treated as absent, `,string` int64 (parse_string_i64), `-0` integers (SignedInteger), base64-or-array byte fields (BytesField), and *_seen flags. On top of that, a strict pre-pass in decode_strict_as (UniqueSeed/UniqueVisitor) checks duplicate keys and depth and then decodes a second time. Outputs are built by hand as strings with json::quote and a GoFormatter that HTML-escapes <, >, & (`fn encode(` appears 30+ times; Connection::encode, IdentityRequest::encode and others push_str field by field). Lease keeps `deadline_raw` so it can re-encode the exact bytes it received.

**Simpler.** Use #[derive(Serialize, Deserialize)] with #[serde(deny_unknown_fields)], skip_serializing_if to get omitempty, and serde_json::to_vec for output. Delete GoFormatter/quote, SignedInteger, BytesField, the UniqueSeed pre-pass (derive already rejects duplicate fields), the `,string` adapter and deadline_raw in both lease.rs and factory/run.rs. Keep only real validation: IDs, paths and digests checked by the root daemon. Delete or rewrite the oracle tests that pin exact bytes along with the code.

**Risk.** Go clients must keep decoding the output. They will, because Go's decoder is order-insensitive and case-insensitive. Hand-encoded response field names and omitempty behaviour need a mechanical check, and the oracle tests that pin exact bytes (lib/host/tests/*_oracle.rs, gmux_smoke.rs) would be deleted or rewritten along with the code.

**Visibility.** Only the wire bytes between soda-host and the Go web server change (escaping, field order). Nothing the frontend renders changes.

**Safe first step.** In cmd/soda-project-maintain/src/config_wire.rs, replace the hand-written ConfigVisitor and its helpers (HOST_FIELDS, host_field_slot, set_early_field, set_late_field, set_string) with #[derive(Deserialize)] #[serde(default, deny_unknown_fields)] on Config. Keep decode_host_config's null-or-object handling. The host config is written with exact snake_case keys, so the observable result is identical. Also fix the false "byte for byte" claim in the lib/host/src/lib.rs header.

**Related.** The same Go-mirroring style (deadline_raw, Go-exact encoders, oracle tests) is in lib/host/src/terminal/factory/run.rs and in the lib/host/tests/*_oracle.rs and gmux_smoke.rs suites. The other Rust services under cmd/*/src that were ported from Go likely follow the same pattern.

### FED-1: Delete the 122-entry Feed component catalog and the worker-chosen component_id

- **Subsystem:** The Feed. **Category:** speculative-abstraction. **Verdict:** confirmed.
- **Removable:** ~1900 lines. **Risk:** medium. **Visible change:** UI, model tools. **Behaviour-preserving:** yes.
- **Status:** done (`8ab34803`).
- **Where:** `internal/factory/feed_components.json:1-1714`, `internal/factory/feed_components.go:1-83`, `internal/web/api/factory_mcp_bound.go:62-94`, `internal/web/api/factory_mcp.go:504-508,597-607,909`, `internal/web/api/factory_mcp_role_schema.go:21,37`, `internal/web/api/factory_mcp_communication.go:18`, `internal/web/api/factory_mcp_library.go:109`, `internal/factory/control/intent_bound.go:123-126`, `internal/store/factory_mcp_communication.go:30-32,154-197`, `internal/store/factory_feed_events.go:21-29`, `internal/store/factory_feed_record.go:140-142,786-790`, `internal/store/factory_feed.go:99-100`, `internal/store/factory_feed_contribution.go:55,165`, `internal/store/factory_library.go:402,465-466`, `internal/store/factory_repair_draft.go:80`, `internal/store/factory_repair_diagnosis.go:53`, `internal/store/factory_verification_journey.go:83`, `internal/factory/library.go:145-166,181-183,202,216-250`, `frontend/spaces/sodaspaces-factory-feed-communication.ts:3-40,52-58`, `frontend/spaces/sodaspaces-factory-feed-thread.ts:95-118,229-262`, `frontend/spaces/sodaspaces-factory-feed-families.ts:170-213`, `internal/store/factory_feed_catalog_test.go:1-98`

**What.** A 1,714-line JSON catalog defines 122 presentation 'components', each with roles, outlets, authority and availability: S01-16, PL01-11, AR01-11, BU01-11, IN01-11, FX01-11, ST01-11, TP01-12, K01-08, LA01-09 and F01-11. Every communication and mutation tool, finish tools included, offers an optional component_id. Its schema is rewritten per capability into an enum plus an 'ID: Label; ...' description. The read body also carries a feed_components list. The server validates the value twice (admitMutation, then recordMCPCommunicationTx) and refuses the submission if it is wrong. On every feed read, feedSubmissionComponents re-reads all MCP submissions of the window just to recover a finish tool's component. The browser bundles the whole JSON but uses only id, label, family and presentation (4 of 8 fields), mapping 122 IDs onto 19 presentations.

**Simpler.** Remove feed_components.json and feed_components.go, the component_id schema property, the enum building, workerFeedComponents and the feed_components read field, the ValidateFeedComponent calls, feedSubmissionComponents and the catalog test. Presentation on the frontend comes from communication kind. Controller events use a short Go list of event kinds, which replaces the authority check in recordFactoryFeedEventTx. Keep writing the existing F03, F04, F10 and F11 values, or rewrite the SQL in factory_feed.go, factory_feed_contribution.go and the F10 verification queries. Library operations switch on the tool name. Thread level is derived from kind. A stray component_id in old payloads is ignored, not refused.

**Risk.** The model tool schemas lose one optional field, and the refusals for it go away. Five SQL queries treat component_id='F10' as the durable app-verified fact (see FED-9), so keep that value or migrate those queries first. ui/repo-places edits sodaspaces-factory-feed-communication.ts. Browser tests assert per-component captions.

**Visibility.** Card captions such as 'Task lane board' or 'Plan handoff brief' become kind labels such as 'Progress update' or 'Plan'. Tool schemas lose component_id and the read body loses feed_components. Dropping the one-value enum on the 13 tools, feedSubmissionComponents' re-read and the DE IDs changes nothing visible except record headline wording.

**Safe first step.** In frontend/spaces/sodaspaces-factory-feed-thread.ts, delete DE01, DE07 and DE10 from the 'post' levelList string, and DE02-06, DE08 and DE09 from the 'comment' string. These IDs exist in neither the catalog nor any backend writer, so no record can carry them and the change is invisible.

**Related.** Durable SQL that keys on component_id values: F10 in factory_app_verification.go:58, application_factory_delivery.go:61,137, factory_delivery_admission.go:183 and factory_journey_readback.go:297; F03, F04 and F11 in factory_feed.go:99-100 and factory_feed_contribution.go:55,165. Controller writers across store/ hard-code F01-F11. FinishProductPlanArgsGuidance in control/intent_bound.go names component_id in a worker-facing prompt string.

### FED-7: Collapse the Feed element's draft lineage, reply-omission cache, per-comment clocks and N+1 per-post polling into one enriched feed refresh

- **Subsystem:** The Feed. **Category:** speculative-abstraction. **Verdict:** partially-confirmed.
- **Removable:** ~1900 lines. **Risk:** medium. **Visible change:** UI. **Behaviour-preserving:** yes.
- **Status:** done on `simplify/feed-render`: GET factory/feed serves each human post with its comment and outcome; the element keeps one serial, plain per-request drafts and a newest-page thread refresh.
- **Where:** `frontend/spaces/sodaspaces-factory-feed.ts:126-492,494-702,704-856,935-937,1054-1180,1981-2200,2206-2330,2984-3231,3596-3903`, `internal/web/api/factory_feed_comment.go:354-379,409-458`, `internal/store/factory_feed_product_change.go:128-246`, `internal/store/factory_feed_record.go:45`, `tests/frontend/factory-feed-browser.test.ts`

**What.** The 4,045-line Feed element carries a lot of client-side state: - unsent revision drafts in localStorage with lineage tokens, ancestor snapshot chains and compare-and-swap against other tabs (~330 lines) - stored intake records - a cursor-windowed reply cache that infers deleted replies from page coverage (sweepThreadPage, coveredOlderOmissions, spliceThreadPage, ~210 lines) - separate write-order clocks per comment for progress, body and removal - queued role-reply scope tokens - per-filter cached views that restore scroll position - about 15 serial counters. Every 10 seconds it reloads the page. It then GETs every human post's comment detail one at a time, every stored intake record, and every open thread.

**Simpler.** Let GET factory/feed return each post's full body and pipeline outcome. Re-render from that response on the existing 10-second refresh, and drop refreshPostBodies and the per-record refreshOutcomes loops. For an open thread, re-fetch its first reply page and append 'Show more' pages without inferring omissions. Keep one serial to drop stale responses. Store unsent drafts as plain text per key, with no lineage, ancestors or intake records.

**Risk.** This is a large rewrite of the most-edited frontend file. ui/repo-places (a 322-line diff) and the hardening branch (the allowance notice) both edit it, so sequence it after they land. factory-feed-browser.test.ts (10,703 lines) encodes many of these races.

**Visibility.** The 'Another tab saved a different draft' and stale-revision banners disappear. Otherwise the content is the same, with far fewer requests.

**Safe first step.** In frontend/spaces/sodaspaces-factory-feed.ts, refreshOutcomes (about lines 1981-2200) repeats the same 'stamp order, GET factory/feed/comments/{id}, handle 404/403 and tombstone, check requested-ID binding, check order, apply' block three times: for intake records, for threadWatched rows and for cached unsettled thread rows. Merge them into one private helper that takes the comment id and an apply callback, keeping the same checks in the same order and the same requests. This removes roughly 150-200 lines with identical behaviour. The draft lineage, omission inference and N+1 replacement change behaviour and must be handled as a deliberate visible change.

**Related.** apiFactoryFeedReplies (internal/web/api/factory_feed_comment.go:409-458) has the same N+1 pattern on the server: for each reply id it runs FeedComment, CommentPipeline, CommentProgress and feedCommentWithProduct. reloadPostBody (around line 2335) and pollOutcome repeat the body and progress clock checks. The prod-hardening branch adds readAllowance into the same readLatest refresh fan-out.

### FED-2: Collapse about 57 card renderers and three overlapping card models into one render path with four card types

- **Subsystem:** The Feed. **Category:** duplicated-layer. **Verdict:** confirmed.
- **Removable:** ~1800 lines. **Risk:** medium. **Visible change:** UI. **Behaviour-preserving:** yes.
- **Status:** done on `simplify/feed-render`: one card shell with four card types; the family model, status blocks, payload blocks and side models are deleted. The unused soda-feed CSS in sodaspaces-workspace.css is still there (another branch owns that file).
- **Where:** `frontend/spaces/sodaspaces-factory-feed-families.ts:38-1501`, `frontend/spaces/sodaspaces-factory-feed-view.ts:107-174,224-303,722-740,870-964,1070-1113,1166-1259`, `frontend/spaces/sodaspaces-factory-feed-record-actions.ts:1-86`, `frontend/spaces/sodaspaces-factory-feed-actions.ts:1-153`, `frontend/spaces/sodaspaces-factory-feed-communication.ts:52-77`, `frontend/spaces/sodaspaces-factory-feed-thread.ts:13-93,492-590`, `frontend/spaces/sodaspaces-factory-feed.ts:3508-3517`, `frontend/spaces/sodaspaces-workspace.css (≈576 soda-feed/soda-action lines)`

**What.** describeFeedPost builds several overlapping models for every entry: - a v1 'family' model: 23 FeedFamily values, 4 FeedStatusClass values, familyText detail/next boilerplate, and FeedBlocks from either payloadBlocks or familyBlocks - a v2 model: the 7-state FeedState plus describePostLine's headline and next line - a header status from readableStatus (35 labels plus systemStatusLabels) - PlanIssue, WorkEvidence and DecisionAttention models. Then renderFeedPost calls renderFeedFamily, which goes either through renderFeedCommunication → communicationAction → renderFeedAction (18 bodies) or through renderRecordAction (9 scenes) plus renderFeedBlock (8 primitives), plus renderJudgment bars. The latest commit, 'Feed v2' (b2597858, +1,015 lines), was added on top of v1 instead of replacing it. Layers between a stored worker report and the DOM: decode → JSONB → union query → enrichFeedPosts → JSON → parseFeedPayload → threadFeed → describeFeedFamily/describePostLine/describePostHeader → renderFeedPost → renderFeedFamily → renderFeedCommunication → renderFeedAction.

**Simpler.** One renderCard(entry) with four card types (work record, worker message, factory event, comment) sharing the v2 shell, one 7-state FeedState model, the typed result rendered once, and no payloadBlocks/familyBlocks/FeedFamily/familyText/PlanIssue/WorkEvidence/DecisionAttention or dead status vocabulary.

**Risk.** Every card changes visually. tests/frontend/factory-feed-browser.test.ts (10,703 lines) and factory-feed-communication-browser.test.ts (2,291 lines) assert the current anatomy and would be rewritten or deleted. ui/repo-places edits feed-communication.ts and feed.ts, so sequence after it lands. About 300 more CSS lines in sodaspaces-workspace.css go with the bespoke layouts.

**Visibility.** Removing the duplicate 'Recorded details and evidence' disclosure, the dead PlanIssue, denied and status branches, and the placeholder generation is invisible or nearly so. The bespoke per-family layouts become four consistent cards that show each fact once.

**Safe first step.** Delete the dead code in this order. Steps 1-2 change nothing observable; step 3 changes only an unreachable branch. 1. In sodaspaces-factory-feed-view.ts: - Delete PlanIssueModel and describePlanIssue (around lines 224-245) and the `plan` field of FeedPostModel (line 873), with its assignment (line 956). - Drop the `denied` parameter from describeFeedPost and describeDecisionAttention, together with the `if (denied)` branch at line 285. - Remove the literal `false` argument at sodaspaces-factory-feed.ts:3514. 2. Delete the dead status-word handling: - the 'judgment:applied|draft:valid|effect:created' regex at families.ts:137 - the equivalent ternaries at families.ts:1292-1296 (keep the entry.activity==='judgment' check) - view.ts:166-168 - the `finish_incomplete: 'incomplete'` mapping at families.ts:185 - the matching fixture in tests/frontend/factory-feed-browser.test.ts:23 3. Stop generating placeholder rows inside familyBlocks so withoutPlaceholders can be removed. Some rows also lose their 'not recorded' fallback, but those rows are filtered out today anyway.

**Related.** sodaspaces-factory-feed-response.ts (1,936 lines) and the large browser tests (tests/frontend/factory-feed-browser.test.ts, factory-feed-communication-browser.test.ts) lock in the current anatomy. In the Go store, internal/store/factory_feed.go classifies the same entries for the decisions, knowledge, attention and evidence filters with SQL predicates that repeat the family classification.

### HFW-3: ~1,800 lines of unreferenced host runtime code

- **Subsystem:** Rust host: factory worker runtime. **Category:** dead-code. **Verdict:** confirmed.
- **Removable:** ~1800 lines. **Risk:** low. **Visible change:** none. **Behaviour-preserving:** yes.
- **Status:** done (`d0435632`).
- **Where:** `lib/host/src/factory/confirmation.rs:1-253`, `lib/host/src/factory/mod.rs:70-77,109`, `lib/host/src/factory/tasks.rs:1-359`, `lib/host/src/terminal/factory/children.rs:1-457`, `lib/host/src/factory/task_events.rs:66-125`, `lib/host/src/terminal/factory/prompt_entry.rs:1-145`, `lib/host/src/terminal/factory/mcp_relay.rs:14-26,29-58,86-97,104-131,187-220`, `lib/host/src/terminal/factory/run.rs:109-292,339-365`, `lib/host/src/terminal/factory/muse/reserve.rs:14-26`, `lib/host/src/terminal/factory/muse/commands.rs:51-110,141-189`, `lib/host/src/terminal/factory/tcodex.rs:112-137,180-215,263-272`, `lib/host/src/terminal/factory/msp/session.rs:153-162,280-293,309-333`, `lib/host/src/terminal/factory/msp/codex.rs:45-55,102-127`, `lib/host/src/muse/execution.rs:13-32`

**What.** Code with no production callers: - confirmation.rs: 18 client-side confirm_* and status-mapping functions ('Pure client-side logic from internal/host/...go'). - factory/tasks.rs TaskRunner and terminal/factory/children.rs ChildTable: a host-spawned child-process table superseded by native children. - Factory::record_task_event. - prompt_entry.rs ('Terminal-side mirror of project.FactoryPromptEntry'). - Most of mcp_relay.rs: UserPromptDelivery, TRUSTED_INSTRUCTION_CHANNEL (a const None), instruction_route, StdioConfig/stdio_config, relay_stream_is_mcp, mcp_route, the FACTORY_MCP_RELAY/IPC_DIR/MCP_SOCKET and MAX_MCP_CONFIG constants, and the Codex TOML route. - The hand decoder for terminal FactoryRun (FactoryRunWire, the feed_* helpers and FactoryRun::decode). - The non-controlled factory_muse_reserve and factory_codex_reserve, plus the 'muse exec' and 'codex exec' supervisor scripts and channels:None branches. muse_controlled_argv and codex_control::exec_argv build that argv and then overwrite its last element with the serve/app-server script. - msp start_session, read_session, log_streams, start_codex_thread and read_codex_thread. - MuseRuntime::verify_guest_binary ('Retained for explicit digest re-verification').

**Simpler.** Delete these. Build the controlled worker argv directly (env list plus serve script) instead of building the exec argv and patching its last element. Keep instruction_argv, PerRunInstructionRoute and MAX_INSTRUCTION in mcp_relay.rs, which are still used.

**Risk.** None of it is reachable. Only re-exports and tests that import it need removing.

**Visibility.** No runtime effect.

**Safe first step.** 1. Delete lib/host/src/factory/confirmation.rs, lib/host/src/factory/tasks.rs, lib/host/src/terminal/factory/children.rs and lib/host/src/terminal/factory/prompt_entry.rs. 2. Remove their mod declarations, and the re-exports at factory/mod.rs:70-77 and :109. 3. Delete Factory::record_task_event (task_events.rs:66-125), MuseRuntime::verify_guest_binary, and FactoryRun::decode with FactoryRunWire and the feed_* helpers in terminal/factory/run.rs. 4. Delete the dead mcp_relay items, keeping instruction_argv, PerRunInstructionRoute and MAX_INSTRUCTION. 5. Delete msp start_session, read_session, log_streams, start_codex_thread and read_codex_thread. 6. Delete the non-controlled factory_muse_reserve and factory_codex_reserve wrappers.

**Related.** The same exec-argv-then-patch pattern appears in lib/host/src/terminal/factory/codex_control.rs (exec_argv over tcodex::reserve_exec_argv) and in muse/commands.rs (muse_controlled_argv over muse_exec_argv).

### HFW-4: Go/CPython byte-compatibility emulation on host-private files and Rust↔Rust wires

- **Subsystem:** Rust host: factory worker runtime. **Category:** duplicated-layer. **Verdict:** confirmed.
- **Removable:** ~1800 lines. **Risk:** medium. **Visible change:** none. **Behaviour-preserving:** yes.
- **Status:** open (behaviour-preserving).
- **Where:** `lib/host/src/factory/receipt.rs:63-290`, `lib/host/src/factory/run.rs:394-648`, `lib/host/src/factory/identity.rs:1-247`, `lib/host/src/factory/manifest.rs:44-227`, `lib/host/src/factory/events.rs:95-160,300-345`, `lib/host/src/factory/state.rs:43-150`, `lib/host/src/factory/requests.rs:285-384`, `lib/host/src/preparation/mod.rs:116-260`, `lib/host/src/preparation/state.rs`, `lib/host/src/preparation/decisions.rs`, `lib/host/src/prepare/mod.rs:1-120`, `lib/host/src/muse/wire.rs:1-425`, `lib/host/src/muse/validate.rs:120-430`, `cmd/soda-project-terminal/src/factory_roles/emit.rs:1-101`, `cmd/soda-project-terminal/src/pyemit.rs`, `cmd/soda-muse/src/launch_wire.rs:1-95`, `cmd/soda-factory/src/main.rs:43-105,149-197`

**What.** The host was ported 'byte for byte' from a Go host that no longer exists (lib.rs header; internal/host now holds only the Go client), and it kept Go's encoding/json quirks everywhere. In scope (excluding tcontrol, covered in HFW-12) there are: - 52 hand-written serde Deserialize impls and 127 case-insensitive key matches (eq_ignore_ascii_case). - Six near-identical 'Go strict object' macros: typed_run_fields!, typed_event_fields!, typed_strict_object!, typed_state_object!, typed_preparation_object!, typed_decision!. - 32 hand-written field-ordered JSON encoders, among them FactoryReceipt::encode, FactoryRun::encode_into, FactoryState::encode and Lease::encode_into. - Go quirk emulation: SignedInteger for JSON '-0', base64-or-byte-array PromptBytes/PlanBytes, GoStringList (null element becomes ""), Go zero-time tombstones, and SoftString/SoftBool tolerant podman-inspect decoders. - In the in-container helper: CPython json.dumps output and Python == semantics (1 == 1.0, True == 1). - In soda-factory: a hand-rolled Go flag parser and a GoFormatter that HTML-escapes <, > and & in a 3-key envelope.

**Simpler.** Use #[derive(Serialize, Deserialize)] with #[serde(deny_unknown_fields)] where strictness matters, skip_serializing_if for omitempty, and a base64 helper for byte fields. Share one LaunchRequest type between soda-muse and the host, and use plain serde_json for helper I/O and soda-factory's envelope. Delete the Go-strict macros, hand-written visitors and field-ordered encoders, GoFormatter, the hand-rolled Go flag parser, the Python json.dumps and `==` emulation, and the oracle vectors that pin them. Keep the tolerant decoding of podman-inspect output, because podman is a real external peer.

**Risk.** Field names stay identical, so the Go client and any existing receipt files still decode. Canonical TaskEvent encoding is needed only while the hash chain exists.

**Visibility.** Only the acceptance of exotic inputs no real peer sends changes (mixed-case keys, -0, byte-array prompts).

**Safe first step.** In cmd/soda-factory/src/main.rs, delete GoFormatter and the BTreeMap logic in encode_envelope (lines ~150-200). Replace them with a plain serde_json::to_vec of the envelope: the Rust host decodes both outputs to the same values. In cmd/soda-muse/src/launch_wire.rs, drop the "exact Go field order" framing and reuse the host's request type. Both changes keep observable behaviour identical.

**Related.** The same Go-quirk machinery appears outside the listed scope: lib/host/src/tcontrol_*.rs (HFW-12), lib/host/src/terminal/{wire,lease,binding,stream}.rs, lib/host/src/tailnet/runtime/*, lib/host/src/domain/{account,profile}.rs (string_field! macros), lib/host/src/project/inspect.rs (tolerant_object!), lib/host/src/json/mod.rs, and the CPython emitters in cmd/soda-project-terminal/src/{pyemit.rs,broker.rs,bin/project-account.rs}.

### PLN-4: Walkthrough consolidation + consistency JEV pair runs twice, gates Generate, and its verdict feeds nothing that is built

- **Subsystem:** Intent to plan (Product Lead, Architect, research, decomposition). **Category:** duplicated-layer. **Verdict:** confirmed.
- **Removable:** ~1800 lines. **Risk:** medium. **Visible change:** UI, model tools, DB schema. **Behaviour-preserving:** no.
- **Status:** owner decision; JEV calls kept: the consistency verdict gates Generate (admitRecordedWalkthroughReview) and the recommendation and consistency notes render in explore and Check walkthrough, so the answer reaches a gate and the UI. Dropping the gate or the explore-time pair is a product decision.
- **Where:** `internal/factory/intent_alternatives.go:8-187`, `internal/factory/intent_proposal_content.go:53-84`, `internal/web/api/factory_intent_explore.go:211-228`, `internal/web/api/factory_intent_generation.go:247-325`, `internal/factory/control/intent_plan_choice.go:31-37,119-123,335-363`, `internal/factory/control/consolidation_advice.go:1-66`, `internal/factory/control/consistency_advice.go:1-112`, `internal/factory/control/judgment_advice.go:49-53,86-93`, `internal/factory/consistency.go:38,67-128`, `internal/factory/consolidation.go:1-132`, `internal/factory/decomposition_options.go:370-372`, `internal/factory/intent_review.go:1-163`, `internal/store/factory_intent_reviews.go:1-70`, `internal/web/api/factory_intents.go:121-167`, `internal/web/api/extension_native.go:209`, `internal/factory/control/intent_bound.go:97-99`, `frontend/spaces/sodaspaces-intent-walkthrough.ts:67-140`

**What.** Consolidation is a JEV call that picks one of two walkthroughs; consistency is a second call that checks it. The pair runs twice. First during explore, over hand-written alternatives chosen by keyword match (DescribeAlternatives: single_person/shared_review/self_service/managed_intake). Then again in 'Check walkthrough', over the two walkthroughs the planner authored. Generate refuses until that second pair replays clean. The chosen walkthrough and its report never reach issues, prompts or verification.

**Simpler.** Delete DescribeAlternatives and the hand-written alternatives, the explore-time consolidation/consistency calls, the effect regex, the Generate gate (admitRecordedWalkthroughReview) and POST plan/review. Ask the planner for one walkthrough and show it next to the candidate tasks; the founder's Accept is the check. If a machine check is still wanted, keep one advisory JEV consistency call that never blocks and is not poisoned by a failed receipt. SavedReview shrinks to the list of challenges.

**Risk.** This is a product decision: the owner may want a JEV comparison. If so, keep a single advisory call rather than a gate. The decompose prompt and finish schema change (one walkthrough, no copied facts).

**Visibility.** The 'Check walkthrough' step, the compared-approaches and consistency notes in explore, and intent_review_required refusals disappear. The factory_intent_reviews table becomes unneeded.

**Related.** The same receipt-poisoning (a failed receipt becomes permanently unavailable) in judgment_advice.go applies to every AdvisePurpose user (assumption review, other purpose judgments). The prod-hardening branch edits intent_bound.go and intent_proposal_content.go, sharpening the walkthrough prompt and validation errors, so it extends rather than removes this path.

### PTH-2: Human 'Start Codex' terminal broker is a separate credential-delivery path beside the Muse and factory identity paths

- **Subsystem:** In-project helpers (soda-project-terminal) and the host Muse launch runtime. **Category:** duplicated-layer. **Verdict:** partially-confirmed.
- **Removable:** ~1600 lines. **Risk:** medium. **Visible change:** UI. **Behaviour-preserving:** no.
- **Status:** owner decision.
- **Where:** `cmd/soda-project-terminal/src/broker.rs:1-165`, `cmd/soda-project-terminal/src/subscription_wire.rs:1-186`, `cmd/soda-project-terminal/src/subscription_profile.rs:1-268`, `cmd/soda-project-terminal/src/subscription_prepare.rs:1-153`, `cmd/soda-project-terminal/src/subscription_retire.rs:1-146`, `cmd/soda-project-terminal/src/subscription_credentials.rs:1-119`, `cmd/soda-project-terminal/src/subscription_start.rs:1-106`, `cmd/soda-project-terminal/src/subscription_cgroup.rs:1-99`, `lib/host/src/terminal/identity.rs:1-422`, `lib/host/src/terminal/broker.rs:51-146`, `lib/host/src/terminal/native.rs:73-93`, `lib/host/src/dbackend/backend.rs:532-583`, `lib/host/src/dbackend/websocket.rs:112`, `frontend/spaces/soda-identity.ts:190-270`

**What.** Codex in a browser terminal has its own end-to-end credential pipeline. The frontend 'Start Codex' form POSTs /identity/launch. The host acquires a codex lease, then project-terminal `broker` 'prepare' reserves a terminal, writes a 'subscription' profile file, mounts two tmpfs (model/harness, model/auth) and creates a tmux session. The host registers the lease, stages the harness, and 'start' root-seals the harness, re-hashes it, rechecks the binding and respawns the pane. 'stage' seeds auth.json; validate/finish/stop capture the credential back and retire it. On every call the stored lease is deep-compared to the incoming one with Python dict equality. Muse does the same job (lease, tmpfs credential root, auth.json staging, retirement) in lib/host/src/muse/{stage,operate,stop}.rs, and factory runs have a third path (factory_identity_operation).

**Simpler.** Owner decision. If humans only need Muse in project terminals, delete the Codex terminal path: the broker route and subscription_*.rs, the broker cases in tests/cli.rs, host identity.rs's Codex staging and identity_call, terminal::identity_launch, the KIND_TERMINAL fallthrough in dbackend identity_action, the managed_end Codex hook, stream_identity_harness, and the 'Start Codex' form in soda-identity.ts. Keep the codex_harness/codex_harness_sha256 host config, because factory Codex (factory/model_catalog.rs, tcodex.rs) and soda-project-maintain still read it. If Codex-in-terminal has to stay, move it onto the host-side staging that Muse uses instead of keeping the in-container broker protocol.

**Risk.** This removes a user-visible feature (Start Codex in Spaces identity panel) and the managed Codex end hook in dbackend/websocket.rs managed_end. The soda-identity Codex provider and the factory Codex harness are in other scopes and may still need the Codex connection itself.

**Visibility.** The 'Start Codex' launch form in the project identity panel disappears, or changes if merged into the Muse path.

**Related.** Factory Codex harness path (lib/host/src/terminal/factory/tcodex.rs, model_catalog.rs) shares codex_harness config; Muse staging in lib/host/src/muse/stage.rs is the parallel credential path; soda-project-maintain config_wire.rs/support_root.rs carry codex_harness fields.

### CTX-1: Library Admin and Researcher stacks (unreachable roles) can be deleted outright

- **Subsystem:** Worker context assembly, project library and research. **Category:** dead-code. **Verdict:** confirmed.
- **Removable:** ~1500 lines. **Risk:** low. **Visible change:** UI, model tools, DB schema. **Behaviour-preserving:** yes.
- **Status:** done (`034f553c`, `ab431714`).
- **Where:** `internal/factory/library.go:89-252`, `internal/factory/research.go:1-94`, `internal/factory/research_record.go:1-238`, `internal/factory/library_result.go:1-11`, `internal/factory/control/library_maintenance.go:1-61`, `internal/factory/control/research_settlement.go:1-26`, `internal/factory/control/planning.go:86-91`, `internal/factory/control/dispatch_attempt.go:517`, `internal/store/factory_library.go:83-254`, `internal/store/factory_library.go:387-714`, `internal/store/factory_research.go:1-83`, `internal/store/factory_context_library.go:1-50`, `internal/store/factory_context_current.go:242-258`, `internal/web/api/factory_mcp_library.go:24-150,180-200`, `internal/web/api/factory_mcp_research.go:1-77`, `internal/factory/mcp_profile.go:26,65,270-281,309-311`, `internal/factory/feed_components.go:71`, `internal/factory/feed_components.json:1448-1574 (LA01-LA09)`, `internal/factory/soda_issue_block.go:20-21,66,105-130`, `internal/store/factory_feed.go:104 and factory_feed_contribution.go:89 (CASE arms)`, `frontend/spaces/sodaspaces-factory-feed-families.ts:75,92; sodaspaces-factory-feed.ts:1238`, `lib/host/src/factory/run.rs:63`, `internal/store/factory_library_test.go`

**What.** There are two whole worker roles, Library Admin (10 mutation tools, preview, inventory with duplicate/contradiction/orphan analysis, LA01-LA09 Feed cards) and Researcher (research_scope, record_research with question-indexed progressive logs and coverage, finish_research, its own factory_research_records table). Each comes with its own execution phase (maintain_library, research), prompt branches, settlement, CurrentContext role header, feed SQL CASE arms and MCP schemas.

**Simpler.** Delete both roles: the maintain_library/research phases, the library_admin/researcher profiles and their tools (library mutations, preview, inventory, research_scope/record_research/finish_research), LibraryChange/DecodeLibraryChange/LibraryInventory, research.go, research_record.go, factory_research.go (the factory_research_records table), factory_context_library.go, the inventory and maintenance halves of store/factory_library.go, the maintenance schemas, research MCP, LA01-LA09 cards, the role values in the soda block, the planning guard and the prompt and feed-SQL branches. selectInitialIssuePhase then always returns investigate/coder. Keep factory_library_items/factory_library_revisions, LibrarySearch/LibraryRead, the read tools and the communication library writes, because the project library page and the other roles use them. Delete the tests that cover the removed code.

**Risk.** Nothing in the product creates these assignments. The only thing to watch is mechanical: CASE arms in feed SQL, Feed role labels (feed-families.ts, portraits) and the result/assignment unions need trimming together so nothing references a removed constant.

**Visibility.** Library Admin and Researcher Feed role labels, the LA cards and the research_scope/record_research/finish_research/library maintenance tools disappear. Nobody can currently reach them, so nothing a user does today changes. factory_research_records and the library revisions table stop being created (no migration needed with zero users).

**Safe first step.** Delete the Researcher stack first, since it is self-contained: internal/factory/research.go, internal/factory/research_record.go, internal/factory/control/research_settlement.go, internal/store/factory_research.go, internal/web/api/factory_mcp_research.go, the SodaRoleResearch case in soda_issue_block.go and library_maintenance.go:45-46, and the research arm in planning.go:90. Then remove ExecutionResearch/MCPProfileResearcher and their tool entries from mcp_profile.go, project/factory.go, the dispatch prompt and the result unions, along with the 'research' CASE arms in the feed SQL. No reachable path changes behaviour, because no producer writes a soda block with role research.

**Related.** The SodaIssueBlock parser itself (soda_issue_block.go: Labels/Workflow have no outside callers) only exists to route these two roles plus coder, so it can collapse once they are gone. The same unreachable role names also appear in the Rust host (lib/host/src/factory/run.rs EXECUTION_RESEARCH), cmd/soda-project-terminal/tests/factory_roles_oracle.rs, tests/frontend/factory-feed-browser.test.ts and docs/design/factory-worker-mcp.md §11.

### DSP-2: A durable priority scan and JEV tournament replace the simple oldest-first queue, which is dead in production

- **Subsystem:** Factory dispatch and run start. **Category:** speculative-abstraction. **Verdict:** partially-confirmed.
- **Removable:** ~1500 lines. **Risk:** low. **Visible change:** UI, DB schema. **Behaviour-preserving:** no.
- **Status:** owner decision; JEV call kept: the tournament winner is the issue dispatchOne launches and the progress view renders the comparisons, so the answer reaches a decision. Replacing the durable scan with one in-memory choice stays open.
- **Where:** `internal/factory/control/priority_dispatch.go:1-431`, `internal/factory/control/priority.go:40-225`, `internal/store/factory_priority_inventory.go:1-814`, `internal/store/schema.go:124-130`, `cmd/soda-identity/src/schema.rs:119`, `internal/factory/control/completion_fence.go:51-100`, `internal/factory/control/dispatch.go:332-351`, `internal/factory/control/dispatch.go:382-424`, `internal/web/server.go:74`, `internal/factory/control/readiness_work.go:168-215`, `internal/factory/control/dispatch_attempt.go:190-197`, `internal/factory/control/completion_proposal.go:217`, `internal/factory/control/completion_drive.go:500`, `internal/factory/control/delivery_repair_adoption.go:60`, `internal/factory/control/repair_adoption.go:59`, `frontend/spaces/sodaspaces-factory-progress-view.ts:282-303`, `frontend/spaces/sodaspaces-factory-progress-response.ts:252-291`

**What.** server.go:74 wires Priority unconditionally, so DispatchPass always returns through dispatchPriorityPass and never reaches its oldest-first loop over DispatchControlsAfter (dispatch.go:390-423). Instead each pass does the following. It observes the native revision and captures a 'complete inventory' plus a local digest batched across assignments, allowances, repository inputs, budgets and lifecycle state (factory_priority_inventory.go:91-670). It re-observes the revision and persists a scan with a cursor. It 'observes' each member by running the full dispatchOne planner in collect mode and persisting candidate JSON. It runs a JEV next-work tournament in groups of 255, sized for 4,096 issues (PriorityDefaultMaxCalls=18). It re-observes the revision and re-captures the whole inventory digest. Finally it runs dispatchOne again for the winner, with a checkPlan hook that compares a sha256 of the re-derived candidate.

**Simpler.** Delete dispatchPriorityPass and stop wiring Coordinator.Priority. Dispatch with the existing loop: DispatchControlsAfter oldest-first, with DispatchQueueCursor rotation (keep traversal.go), calling dispatchOne for each issue. Remove the following: priority_dispatch.go; selectPriority and the tournament in priority.go; the tournament fence helpers in completion_fence.go; factory_priority_inventory.go; and the factory_priority_scans and factory_priority_candidates tables in schema.go and schema.rs. Also drop the ObserveRevision, RefreshCandidate, collectPlan and checkPlan deps and all four outside DeletePriorityScan calls (completion_proposal, completion_drive, delivery_repair_adoption, repair_adoption). Keep the PriorityJournal interface, the factory_priority_judgments table and TypeSafePriority, which other judgments share. Remove or simplify the progress view's priority-comparison and "next" display to match. If LLM ordering is still wanted, make one in-memory choice call over this pass's planned candidates and fall back to oldest-first.

**Risk.** The replacement loop exists and is what Priority-less runs (tests) use. The only loss is LLM ordering among several queued issues of one repository. The fallback is oldest-first, which the tournament already applies for fairness and low confidence.

**Visibility.** The 'priority' journey judgment and the priority_incomplete/priority_unavailable/priority_snapshot_changing observation stages disappear. If an in-memory JEV choice is kept, selection semantics stay the same and only the durable scan, digests and double planning go. factory_priority_scans and factory_priority_candidates become unused.

**Related.** The same durable-reservation pattern on factory_priority_judgments is reused by comment judgments (factory_feed_comment.go:442), repair judgments (factory_repair_judgment.go:54), contribution review (factory_feed_contribution.go:46), intent planning (factory_intents.go:429) and SystemOne MCP replay (factory_mcp_systemone.go:263). The store comment at factory_intents.go:270-272 ties a cap to PriorityDefaultMaxCalls, and schema.go:37 checks max_calls BETWEEN 0 AND 18; both would loosen once the tournament is gone.

### PRP-1: Approval chain before any dispatch: requirement chain, approval chain and per-role standing preparation

- **Subsystem:** Project preparation pipeline below the API (standing and candidate preparations, decisions, hold, host and root helper). **Category:** process-machinery. **Verdict:** confirmed.
- **Removable:** ~1500 lines. **Risk:** high. **Visible change:** UI, DB schema. **Behaviour-preserving:** no.
- **Status:** owner decision; coordinate with prod-hardening.
- **Where:** `internal/factory/control/preparation_decisions.go:1-160`, `internal/factory/control/preparation_start.go:1-170`, `internal/factory/control/dispatch_selection.go:177-213`, `internal/store/project_grants.go:30-160`, `internal/store/preparation.go:93-240`, `internal/project/grants.go:36-97`, `internal/project/preparation.go:125-183`, `internal/web/api/preparation_source.go:170-303`, `internal/web/api/preparation_decisions.go:88-222`, `internal/web/api/preparation_decision_preview.go`, `internal/store/factory_merges.go:144-145`, `internal/store/factory_publications.go:168-169`, `internal/store/factory_delivery_admission.go:111`, `internal/store/factory_priority_inventory.go:499-502`, `internal/store/factory_completion_material.go`, `internal/store/factory_app_verification.go`, `cmd/soda-identity/src/schema.rs:191-194,303-306,359-362`, `lib/host/src/preparation/decisions.rs`, `docs/public/40-Configure/30-prepare-project.md`

**What.** No issue can be dispatched until a founder has (1) previewed the source, (2) recorded a RequirementDecision in a chain keyed by predecessor, (3) recorded an ApprovalDecision in a second chain that binds the requirement head, then (4) started a separate standing preparation for each of soda-coder and soda-reviewer. Only then does selectPreparationFor find a ready preparation whose Requirements.ID and Approval.ID equal both current heads. Each decision goes through factory-command idempotency (command_id, SettingsDigest, replayDecision), then decision-id replay (sameJSONDocument), then predecessor compare-and-set on a heads table. Each preparation stores the decision refs together with a chain 'depth' (count(*)), and every later admission recomputes and compares that depth.

**Simpler.** Same as the finder's proposal, with one addition: the merge, publication, delivery-admission and priority-inventory guards that currently check EXISTS on project_requirement_heads and project_approval_heads must switch to checking the per-project setup row, or drop the check. The Rust soda-identity schema and the host decisions.rs refs must go with them.

**Risk.** RequirementDecision and ApprovalDecision are also read by factory_completion_material.go:1100-1160 and factory_app_verification.go:282 (delivery-preparation digests), so those readers must switch to the setup row. The frontend settings section, the API routes and the docs go away with them. A migration (or a fresh database, given zero users) is needed.

**Visibility.** The Factory settings preparation section (source preview, accept, approve-initial, approve, prepare with client-chosen identities, replace-terminal checkbox) is replaced by an automatic status line. The docs page 30-prepare-project.md needs a rewrite. Feed F01 cards stay.

**Related.** The heads tables are also guarded in the merge, publication, delivery-admission, priority-inventory, dispatch-packet, delivery-repair, candidate-preparation and intent-generation store code. Other places: the Rust identity schema with its immutability triggers, the host preparation decision refs (lib/host/src/preparation/decisions.rs), and tests/installed/qualification/native.ts.

### VRF-2: check.sh runs twice under two separate supervisors; the run-scoped re-execution exists only to keep the bundle

- **Subsystem:** Test Pilot / app verification (Go factory+control+store+MCP, Rust host, in-project helper). **Category:** duplicated-layer. **Verdict:** confirmed.
- **Removable:** ~1500 lines. **Risk:** medium. **Visible change:** model tools. **Behaviour-preserving:** no.
- **Status:** owner decision.
- **Where:** `cmd/soda-project-terminal/src/factory_roles/verification.rs:1-540`, `cmd/soda-project-terminal/src/factory_roles/verification_execution.rs:1-438`, `cmd/soda-project-terminal/src/factory_roles/execution.rs:795-878`, `lib/host/src/factory/verification_execution.rs:1-395`, `lib/host/src/factory/verification.rs:1-229`, `lib/host/src/factory/finish.rs:125-160`, `lib/host/src/factory/run.rs:660-675`, `internal/project/factory_verification.go:1-181`, `internal/factory/control/verification_execution.go:40-163`, `internal/store/factory_verification_manifest.go:1-86`, `internal/factory/run.go:88-104`, `internal/project/factory.go:349-354`, `internal/store/application_factory_delivery.go:85-177`

**What.** The reviewer preparation already runs setup.sh and then check.sh in its checkout at the delivered commit (execution.rs run_preparation). Delivery admission requires CheckExit==0 from that run. When an adopted manifest exists, a second system runs setup.sh and check.sh again before the Test Pilot CLI may start. That system is a Go VerificationManifest (marshal, parse and validate), a Rust host copy (decode, validate, verify_for_run, stored and reloaded for every call), a run-scoped helper Request, a fork-and-pipe gated supervisor with pidfd signalling, fixture inode records, not-started, stopped and finished tombstones, retirement confirmation, and a FactoryVerificationChecksState mirrored in Go and Rust. The state is validated in the host helper call, in saved_verification_checks, in Go adoptVerificationChecks, populateVerificationExecution, Run.Validate, CurrentVerificationEvidence.ValidateFor, and checkVerifiedManifestExecutionTx, which also re-runs the 15-field packet comparison.

**Simpler.** Have run_preparation call verification_bundle::retain on its own checkout after a passing check, and expose the bundle digests on the preparation state. Bundle export then reads them from the preparation. Delete the run-scoped verify, verify_inspect and verify_stop helper ops, the host verification_execution.rs, the VerificationManifest types in Go and Rust, FactoryVerificationChecksState, the pre-CLI gate in finish.rs and stop.rs, and the run.VerificationManifest/VerificationChecks fields. The Test Pilot then starts on a passing preparation, and a failing check is reported by delivery admission (F09) or a worker 'failed' report, not by an opaque stop.

**Risk.** Application delivery and packaging (application_factory_delivery.go, host application.rs, terminal export_bundle) must switch from run-bound to preparation-bound bundle digests. The preparation checkout is longer-lived than a disposable clone, so the bundle must be retained right after the check passes. Old runs carry the verification_manifest jsonb field, which can be ignored.

**Visibility.** The verification_evidence tool output loses the verification_checks and manifest_check_receipt fields and the 'native_manifest_passed' outcome string. Bundle delivery cards keep working but cite the preparation instead of the run.

**Related.** The 30-minute cap is duplicated in cmd/soda-project-terminal/src/factory_roles/verification.rs:25 and internal/factory/verification.go:26. Bundle digest consumers are internal/store/application_factory_delivery.go:135 and the host application.rs. docs/development/factory-workflow-contracts.md:2059 documents native_manifest_passed.

### EXT-3: Optional Tailnet is a second Fountain package run by a generic lifecycle controller, a host grant store and Fountain round trips

- **Subsystem:** Fountain extension and its backend. **Category:** speculative-abstraction. **Verdict:** confirmed.
- **Removable:** ~1400 lines. **Risk:** medium. **Visible change:** UI. **Behaviour-preserving:** no.
- **Status:** owner decision; left. Moving the Tailnet switch from Fountain's lifecycle to a Soda setting changes how the installed product enables Tailnet and touches the privileged host runtime; nothing here is unused.
- **Where:** `system/containers/extension/tailnet-package/extension.json:1-22`, `system/containers/extension/tailnet-package/run:1-2`, `cmd/soda-extension/main.go:35-47`, `internal/web/extension.go:13-22`, `internal/web/api/extension_lifecycle.go:1-201`, `internal/web/api/extension_lifecycle_controller.go:1-36`, `internal/host/tailnet_lifecycle.go:1-100`, `lib/host/src/dbackend/tailnet.rs:1-36,270-365`, `lib/host/src/dbackend/backend.rs:46-85,597-598`, `lib/host/src/daemon/routes.rs:363-447`, `lib/host/src/tailnet/lifecycle.rs:1-148`, `lib/host/src/tailnet/lifecycle/runtime.rs:1-322`, `lib/host/src/tailnet/admission.rs:1-231`, `lib/host/src/tcontrol_wire.rs:1345-1647`, `lib/host/src/main.rs:242-247,295-298,357-367`, `lib/soda-release-image/src/native_package.rs:37,401-452,502-555,599-626`, `lib/release-inputs/src/payload.rs:31-47`, `frontend/tailnet/soda-tailnet-entry.ts:1-62`, `frontend/spaces/soda-extension.ts:36-48`, `internal/web/api/extension_lifecycle_test.go`, `internal/web/api/extension_lifecycle_controller_test.go`

**What.** Whether Tailnet runs is owned by Fountain's generic package lifecycle (desired state, pending operation, generation, installation UUID, controller instance) and mirrored in a host 'operational grant' file. Enabling Tailnet in Fountain's extension admin makes the manager call the soda package's lifecycle handler, which forwards to the dashboard, which calls the Rust host twice, which calls back through the dashboard's operator socket to read Fountain manager state. Every Tailnet host/enrollment/project start and every options read re-reads that state.

**Simpler.** Make Tailnet a plain Soda setting. Serve the Tailnet settings page from the soda package as an admin-scope page and add Enable/Disable to the existing /api/settings/tailnet/host handler, gated by authority.Actor.SiteAdmin. The host keeps one 'enabled' flag file; enable = soda-image-import ensure --set tailnet, then start soda-tailnet-host and running project companions; disable = stop the units. Delete the tailnet package, LifecycleController, /api/extensions/lifecycle, host /tailnet/prepare and /tailnet/lifecycle, the tcontrol_wire lifecycle types, ReleasePin, GrantStore/ManagerState/ServiceEpoch, native.extension.lifecycle, the catalog/payload Tailnet entries and the lifecycle tests.

**Risk.** Moves where Tailnet is switched on and touches the privileged host runtime; the image-import ensure and unit start/stop composition must be kept. Fountain's generic lifecycle (other repo) simply goes unused. Catalog/payload pieces belong to release tooling and may also be raised by that audit; tests (~720 Go lines) go with the code.

**Visibility.** An admin enables Tailnet from Soda's Tailnet settings instead of Fountain's extension admin; host/enrollment/project Tailnet controls are unchanged. Payload v4 no longer lists a tailnet package. Not touched by the hardening branch.

**Related.** Release tooling also carries Tailnet catalog and payload pieces: lib/release-inputs payload.rs lifecycle_controller and OptionalExtensions, lib/soda-release-image native_package.rs and tests/oracle/staging.rs, lib/release-inputs/fixtures/payload-v4.json, lib/soda-release-deliver/tests/goldens/deliver.json, and lib/host/tests/data/iconfig/release.json. Docs: docs/development/factory-remaining-tasks/05-experience-and-development.md (R45/R46).

### HFW-2: Codex factory harness kept as a full parallel path (implement runs cannot even start by default)

- **Subsystem:** Rust host: factory worker runtime. **Category:** duplicated-layer. **Verdict:** confirmed.
- **Removable:** ~1300 lines. **Risk:** medium. **Visible change:** UI, model tools. **Behaviour-preserving:** yes.
- **Status:** done (`3b2273ff`): the factory is Muse-only; the human Start Codex terminal stays.
- **Where:** `lib/host/src/dbackend/factory_terminal.rs:44-397`, `lib/host/src/terminal/factory/tcodex.rs:1-684`, `lib/host/src/terminal/factory/muse/reserve.rs:47-240`, `lib/host/src/terminal/factory/muse/lifecycle.rs:8-109`, `lib/host/src/terminal/factory/muse/start.rs:9-70`, `lib/host/src/terminal/factory/model_catalog.rs:1-95`, `lib/host/src/terminal/factory/msp/codex.rs:1-160`, `lib/host/src/terminal/factory/codex_control.rs:23-76`, `lib/host/src/factory/native_control.rs:38-67,496-679,693-740`, `lib/host/src/factory/native_tasks.rs:456-474,806-952`, `lib/host/src/terminal/factory/mcp_stage.rs:12-14,46-61`, `lib/host/tests/tcodex_ops_oracle.rs:1-661`, `cmd/soda-project-maintain/src/config_wire.rs:87-88,121-122`, `cmd/soda-project-maintain/src/config_validation.rs:33-36`

**What.** Every FactoryTerminal method in TerminalSeam branches Codex vs Muse: reserve, start, wait, stop, stop_unbound, capture, live, output and native_output. Each branch lands in a near-identical factory_codex_*/factory_muse_* pair. factory_codex_reserve_with_channels and factory_muse_reserve_with_channels are copies. factory_codex_setup and factory_muse_setup run the same podman setup and git rev-parse with different error strings. factory_codex_stage and factory_muse_stage are identical except for the guest path. The start, wait, validate, stop, stop_unbound, capture, live and output pairs differ only in the paths struct and the scope constant. On top of that, Codex alone has: model-catalog staging with 'native profile' retention, its own app-server driver (drive_codex), a dynamic-tool task correlator (NativeTasks::item/tools), per-item evidence files, and a TOML MCP config.

**Simpler.** Muse-only factory: delete tcodex.rs (keep the few shared helpers), codex_control's exec_argv and supervisor, model_catalog.rs, msp/codex.rs and Connection::codex, drive_codex, native_evidence and retain_native_profile, NativeTasks::item/tools, the Codex arm of mcp_stage and every Codex arm in TerminalSeam. Also delete the codex_model_catalog config keys in daemon/config.rs and soda-project-maintain, and tcodex_ops_oracle.rs. valid_harness_family (Rust and Go) then accepts only muse, and the Settings default becomes muse. If Codex must stay, merge each factory_codex_*/factory_muse_* pair into one function that takes a family descriptor (guest path, config dir, scope, setup script, error strings, optional extra staging step). That saves roughly 300-400 lines.

**Risk.** Product decision: Codex is still a listed provider (frontend/spaces/soda-providers.ts) and the default in the policy form, which would need to become muse. The merge-only variant is low risk.

**Visibility.** Factory roles lose Codex in the Settings harness field and default. The Codex-only dynamic tools (factory_start_task, factory_release_output) and the Codex developer instructions are deleted; the Muse worker contract is unchanged.

**Safe first step.** Merge the duplicated pairs without changing behaviour: 1. In lib/host/src/terminal/factory/tcodex.rs and muse/reserve.rs, merge factory_codex_reserve_with_channels with factory_muse_reserve_with_channels, factory_codex_setup with factory_muse_setup, and factory_codex_stage with factory_muse_stage. Each becomes one function taking a family descriptor: paths, scope constant, guest helper, setup script, the two family-specific error strings, and an optional extra staging hook (Codex catalog or Muse agent definition). 2. Do the same for stop, stop_unbound, capture and live in tcodex.rs:600-684 and muse/lifecycle.rs. 3. In lib/host/src/dbackend/factory_terminal.rs, collapse the codex/muse if-else arms so they dispatch on the descriptor. Keep every error string and the Codex implement-profile refusal exactly as they are.

**Related.** Go keeps two harness families too: internal/project/factory.go (FactoryHarnessCodex, valid family check) and internal/web/api/factory_mcp_p6.go:28. On the frontend, soda-providers.ts lists Codex and sodaspaces-factory-settings.ts:1492,1495 defaults to 'codex'. lib/host/src/terminal/factory/mcp_relay.rs and bound.rs have Codex arms (CodexDeveloperInstructions, Provider::Codex). cmd/soda-project-maintain parses and validates the codex_model_catalog keys.

### PRP-2: Each run creates durable candidate preparations that re-check the whole dispatch authority

- **Subsystem:** Project preparation pipeline below the API (standing and candidate preparations, decisions, hold, host and root helper). **Category:** duplicated-layer. **Verdict:** confirmed.
- **Removable:** ~1300 lines. **Risk:** medium. **Visible change:** UI, DB schema. **Behaviour-preserving:** no.
- **Status:** owner decision; coordinate with prod-hardening.
- **Where:** `internal/factory/control/dispatch_preparation.go:1-264`, `internal/store/factory_candidate_preparations.go:1-751`, `internal/store/factory_source_preparations.go:1-161`, `internal/factory/control/publication_children.go:772-960`, `internal/factory/control/lifecycle.go:126-160`, `internal/factory/control/dispatch_attempt.go:410-436`, `internal/factory/control/dispatch.go:329-331,431,454`, `internal/store/factory_dispatch_packet.go:236-246`, `lib/host/src/prepare/candidate.rs:74-300`, `lib/host/src/prepare/mod.rs:230-250`

**What.** Once the target branch has moved past the standing preparation's commit (always true after the first merge), each coder dispatch runs prepareSourceCoder. Each explicit Retry runs prepareRetryCoder, and each reviewer child runs prepareReviewCandidate. Each one creates a new f-prefixed preparation: a full clone plus setup.sh and check.sh in a fresh checkout, with its own durable admission envelope (factory_admission JSON holding 16 fields: gate revision, IssueControl, AuthorityRef, connection, attempt root, deadline, Stopped/Retirement). AdmitCandidatePreparation and AuthorizeCandidatePreparation each re-run inside a transaction: the dispatch gate revision, the issue-control compare, validateAttemptPacketTx, loadAdmissionGrantsTx, checkAssignmentAuthorityTx, both decision documents with chain depth, retry-command facts, and the merged-source export. Dispatch had just computed all of these and checks them again when it creates the assignment. reconcileRetryPreparations then re-admits every pending coder preparation on every progress pass, and retires it through a native stop with a 'confirmed/uncertain' retirement receipt.

**Simpler.** Make 'fresh role checkout at commit X, run setup.sh and check.sh' part of launching the run. Pass the run's assignment ID as the preparation ID, let the host prepare and launch, and record the outcome on the run. Then delete CandidatePreparationRegistration and the admission envelope, AdmitCandidatePreparation and AuthorizeCandidatePreparation with their authorize/owner/source checks, PendingRetryPreparations with reconcileRetryPreparations, and the separate reviewer-preparation admit/verify/reconcile/launch functions. The run's normal stop or settlement cleans up the checkout. If the run's own diff changed setup.sh, run the new setup instead of refusing (drop confirm_unchanged_setup_inputs on this path).

**Risk.** The run launch path, attempt-allowance deadline locking (attemptDeadlineUnix) and lifecycle stop all touch these rows, so the change needs a careful pass through dispatch_launch.go and the publication-children flow. The host must still get the source export for the reviewer (exportCandidate stays).

**Visibility.** Fewer F01 'Worker environment ...' Feed cards. The WaitPreparation texts for source/retry preparation disappear.

**Related.** The packet admission path (internal/store/factory_dispatch_packet.go:236-246) and context resolution (internal/store/factory_context_resolution.go:75, factory_context_library.go:15) run the same validateAttemptPacketTx and checkAssignmentAuthorityTx checks. Project-wide candidate stop sweeps are in lifecycle.go stopCandidatePreparationsForRoot. The host-side unchanged-input refusal is called from lib/host/src/prepare/mod.rs:241.

### BUD-8: Each Issue-zero run kind has its own funded-minutes budget (intent 0/30, reassessment 30, verification 2 runs/60 min/30 per run), stored as five time-only allowance variants

- **Subsystem:** Budgets, allowances and limits (cross-cutting). **Category:** too-many-states-or-budgets. **Verdict:** confirmed.
- **Removable:** ~1200 lines. **Risk:** medium. **Visible change:** UI, model tools, DB schema. **Behaviour-preserving:** no.
- **Status:** owner decision.
- **Where:** `internal/factory/grants.go:175-199`, `internal/factory/grants.go:249-257`, `internal/factory/control/dispatch_timeonly.go:336-349`, `internal/factory/control/dispatch_timeonly.go:452-548`, `internal/store/factory_dispatch_packet.go:154-192`, `internal/factory/allowance.go:95-117`, `internal/factory/allowance.go:150-166`, `internal/factory/allowance.go:177-240`, `internal/factory/allowance.go:313-375`, `internal/store/factory_attempt_allowances.go:288-378`, `internal/store/factory_attempt_allowances.go:834-1070`, `internal/store/factory_reassessment_allowances.go:1-308`, `internal/store/factory_delivery_admission.go:123-170`, `internal/factory/verification.go:20-27`, `internal/factory/verification_allowance.go:1-69`, `internal/store/factory_verification_allowance.go:1-246`, `internal/web/api/factory_verification_allowance.go:1-81`, `internal/web/api/factory_policy.go:24-25`, `internal/web/api/factory_policy.go:76-79`, `frontend/spaces/sodaspaces-factory-verification-allowance.ts:1-263`, `frontend/spaces/sodaspaces-project-work-view.ts:115-117`, `internal/store/schema.go:81`, `cmd/soda-identity/src/schema.rs:76`, `frontend/spaces/sodaspaces-factory-settings.ts:1026`, `frontend/spaces/sodaspaces-factory-settings.ts:1497-1499`

**What.** Intent planning, completion reassessment, delivery verification, stall repair and delivery repair each size their run from a separate funding source. Each has its own attempt-allowance variant, constructor, lock and transition branch, and all of them also pass the shared caps.

**Simpler.** Give every run the single per-run limit from BUD-1, so Issue-zero roots need no allowance row. Keep only structural counts, as constants derived from assignments (for example, at most 2 verification runs per accepted revision, and one reassessment coder run per cycle). Delete IntentMinutes, ReassessmentMinutes and the verification allowance (types, command chain, API, component), along with the five time-only variants and their constructors, lock/transition branches and CHECK branches.

**Risk.** Touches packet admission and finish transitions for five subject kinds. Each currently relies on its allowance row for 'closed' and replay detection, which must move to the assignment.

**Visibility.** Settings loses 'Intent active minutes' and the verification-limits component goes. Product Lead planning no longer needs funding. The factory.context allowance block disappears for Issue-zero roots.

**Safe first step.** none. The only behaviour-identical piece is tiny: compute the delivery verification run minutes min(ActiveMinutes, MaxAppVerificationRunMinutes, remaining/60) in one helper shared by dispatch_timeonly.go:481 and factory_delivery_admission.go:129, and drop the redundant EffectiveReassessmentMinutes re-application at dispatch_timeonly.go:547 (policy Validate already guarantees 1..120).

**Related.** Repair charges (RepairChargeID, DefaultRepairCycleActiveMinutes) fund the stall and delivery-repair runs through a separate charge ledger (dispatch_timeonly.go:486-529). The verification allowance is also wired through internal/web/api/extension.go, extension_native.go and factory_lifecycle.go, and is documented in docs/development/factory-workflow-contracts.md.

### FED-5: Collapse the comment pipeline's redundant states (intake, applied/verified, resume, receipt replay), its single-implementation seams and the display-only product lineage reads

- **Subsystem:** The Feed. **Category:** too-many-states-or-budgets. **Verdict:** partially-confirmed.
- **Removable:** ~1200 lines. **Risk:** medium. **Visible change:** UI, DB schema. **Behaviour-preserving:** yes.
- **Status:** partly done on `simplify/feed-render`: no judgment reservation or receipt replay, no resume flag, one effect advance to verified, no GET-time lineage or linked-work reads, one outcome line. Still open: the intake/judgment/draft/effect columns (need a migration), PostStatus, and the single-implementation drafter/creator seams (the pipeline tests drive through them).
- **Where:** `internal/factory/feed_communication.go:24-63,256-422`, `internal/store/factory_feed_comment.go:333-520,650-889`, `internal/web/api/factory_feed_comment.go:30-265,495-925,990-1033`, `internal/web/api/factory_feed_comment_context.go:1-104`, `internal/web/api/factory_feed_comment_draft.go:1-368`, `internal/web/api/factory_feed_comment_create.go:1-229`, `internal/store/factory_feed_product_change.go:128-375`, `internal/web/api/factory_feed_product_change.go:60-114`, `internal/store/factory_feed_work_trace.go:1-38`, `frontend/spaces/sodaspaces-factory-feed-view.ts:304-684`, `frontend/spaces/sodaspaces-factory-feed-response.ts:1333-1866`

**What.** Each founder comment becomes a request row with: - columns intake, judgment, route, draft, effect, issue and attempts (≤6) - JSON keys resume, resumed, feedback_revision, history, detail and last_checked_at. Every advance is a compare-and-swap over 9 columns plus resume (AdvanceCommentPipeline). JEV routing is reserved in the global judgment counter, and its receipt is replayed on restart. Route new_issue launches a whole Product Lead planning worker run to author one issue: a draft intent is saved, then an IssueDraft is persisted, validated and wiped if invalid. The issue is created through a graph operation and then looked up again to move it from applied to verified. A comment whose source is an accepted product takes a different 'product change' lane. Every GET of such a comment reconstructs adoption lineage for display: recursive ancestry, generation integrity, F10 verification and per-task completion. PostStatus derives 14 status strings. The browser turns those into 5-6 'Processing details' rows and about 20 outcome sentences.

**Simpler.** Replace the separate intake, judgment, draft and effect states with one state column: pending, then discussion, clarify, linked or creating, then created or failed. Keep issue, attempts and detail, updated with plain UPDATEs. Make routing one JEV choice with the answer stored on the row. Drop the reservation and receipt replay, and drop the bare clarify and existing_work replay cases. Merge applied and verified into created. The graph op is already idempotent. Drop the resume, resumed and resume-branch logic in RetryFeedComment. Inline the single-implementation interfaces and the anonymous assertions. For product-change comments, keep ReviewCommentProductChange (a real founder action). Drop only the GET-time lineage reconstruction and link to the intent instead. Bound or cheapen the 250-issue BracketedRead candidate gathering. Note that this changes what the judge sees. In the browser, show one outcome line.

**Risk.** This is the founder-demo 'submit a change as a feed comment' path, so whichever new-work lane survives must be re-checked end to end. Existing request rows need their state columns mapped or migrated.

**Visibility.** Routing results can stay identical when only intake, the applied/verified split, compatibility replay, the interfaces and the resume flag are removed. The 'Processing details' stages and outcome wording get simpler, and the product-change evidence stages go away.

**Safe first step.** In internal/web/api/factory_feed_comment.go: - Delete the dead bare-clarify choice description, the `case factory.CommentRouteClarify:` branch in the JudgeComment key switch (lines 231-232). - Delete the old-receipt cases for bare existing_work and clarify in commentJudgmentFromSelection (lines 132-139). Keep new_issue and discussion. - Inline the single-implementation interfaces CommentDrafter and CommentIssueCreator, plus the anonymous PrepareCommentInput assertion at line 764, by calling the concrete *TypeSafeCommentJudge, *IntentCommentDrafter and creator types directly. - Delete the matching fakes and tests in factory_feed_comment_test.go. There are no state, schema or UI changes.

**Related.** The same pattern of reservation plus receipt replay for funded JEV calls appears in internal/factory/control (TypeSafePriority and the global judgment counter). The product-change lineage duplicates intent adoption and F10 verification reads in internal/store intent and generation code. Frontend Processing details stage mapping lives in sodaspaces-factory-feed-response.ts and sodaspaces-factory-feed-view.ts. On the prod-hardening branch, internal/store/factory_feed_comment.go (RetryFeedComment history JSON) and frontend/spaces/sodaspaces-factory-feed.ts (allowance display) overlap these files but do not address the finding.

### FEF-4: Response parsers reject a whole 200 reply on harmless drift: exact key sets, closed enum lists, 32-hex ids, re-derived server values and echo checks

- **Subsystem:** Frontend: Factory settings, work, progress, request, intent screens. **Category:** unneeded-validation-or-digest. **Verdict:** confirmed.
- **Removable:** ~1200 lines. **Risk:** low. **Visible change:** none. **Behaviour-preserving:** yes.
- **Status:** coordinate with prod-hardening.
- **Where:** `frontend/spaces/sodaspaces-factory-response.ts:455-464`, `frontend/spaces/sodaspaces-factory-stream-response.ts:24-38,51-58`, `frontend/spaces/sodaspaces-factory-response.ts:303-317,363-374`, `frontend/spaces/sodaspaces-factory-progress-response.ts:773-797,972-985,1033-1050,1085-1096,1124-1136`, `frontend/spaces/sodaspaces-factory-work-response.ts:11-13,187-228,390-406,435-450,539-559`, `frontend/spaces/sodaspaces-intent-response.ts:509-567`, `frontend/spaces/sodaspaces-factory-access-response.ts:155-194`, `frontend/spaces/sodaspaces-factory-repair.ts:33-76,141`, `frontend/spaces/sodaspaces-factory-activity.ts:32-45,90-100`, `frontend/spaces/sodaspaces-factory-settings.ts:517-523,609-615,636-661`, `frontend/spaces/sodaspaces-api.ts:4-35`, `frontend/spaces/sodaspaces-factory-request.ts:155-169`

**What.** About 3,900 lines of response parsers (work-response 822, progress-response 1136, intent-response 761, factory-response 464, work-journey-response 419, access-response 194, acceptance-response 89, stream-response 58) check every field against regexes, closed value lists and cross-field rules, and throw on any mismatch. The view then shows 'unavailable' or 'The reply could not be understood'.

**Simpler.** Read leniently: a TypeScript type plus optional chaining and defaults for the fields a view actually renders. Show unknown enum values as text, ignore extra fields, and never reject a 2xx reply over a field the view does not show. Drop the id echo checks and the recomputation of server values. Remove the arbitrary count caps, starting with the 8-charge repair cap, which the unbounded backend list already breaks. Raise or drop the 64 KiB default. HTTP status and JSON.parse failure become the only error paths.

**Risk.** If the backend sends garbage, a view may show an odd label instead of going blank. That is acceptable with one trusted backend.

**Visibility.** No change for valid replies. The failure states — 'unavailable', 'reply could not be understood', read-only settings — stop appearing for harmless drift.

**Safe first step.** In sodaspaces-factory-response.ts, delete the Object.keys exact-key checks in factoryStatusFrame (460-462) and the input_sha regex check (372). Do the same for the key checks in sodaspaces-factory-stream-response.ts (26, 53). In sodaspaces-factory-progress-response.ts, delete the recomputed available/status equality checks (777-782) and the unchanged_count === reused.length check (976-977). In sodaspaces-factory-work-response.ts:541, drop the repository/issue echo check, and in sodaspaces-factory-activity.ts:94, drop the item.url === recomputed url check. For valid replies the rendered output stays the same.

**Related.** sodaspaces-factory-feed-response.ts (1936 lines, the same style, not counted in the finding). The hardening branch prod-hardening adds sodaspaces-factory-allowance.ts and wires allowanceHolds into factoryRepositorySummary in work-response.ts, which extends the strict-parser pattern. sodaspaces-factory-work-journey-response.ts and acceptance-response.ts are the same style.

### FEF-6: Request review carries about 1,200 lines of client-side task shelving, dependency-edge repair and cross-device history recovery to work around one server refusal

- **Subsystem:** Frontend: Factory settings, work, progress, request, intent screens. **Category:** speculative-abstraction. **Verdict:** confirmed.
- **Removable:** ~1200 lines. **Risk:** medium. **Visible change:** UI. **Behaviour-preserving:** no.
- **Status:** owner decision.
- **Where:** `frontend/spaces/sodaspaces-intent.ts:331-352`, `frontend/spaces/sodaspaces-intent.ts:487-572`, `frontend/spaces/sodaspaces-intent.ts:912-1007`, `frontend/spaces/sodaspaces-intent.ts:1108-2072`, `frontend/spaces/sodaspaces-intent.ts:2073-2108`, `frontend/spaces/sodaspaces-intent.ts:2908-3094`, `internal/factory/intent.go:252-262`, `internal/web/api/extension_native.go:198`

**What.** Declining an idea makes the client drop every task linked to it. It keeps the dropped tasks on a 'shelf' (in memory and localStorage) with their original dependency edges and idea lineage, re-attaches the edges if the idea is accepted again, and remaps edges onto regenerated task keys by matching idea sets. It shows 'unresolved repair' reviews. On another device it walks GET /factory/intents/{id}/history up to 8 predecessors, with a 'CAS-like' stale-head guard, to rebuild the shelf.

**Simpler.** Let a revision keep tasks linked to declined ideas, and have generation skip tasks whose ideas are all declined (or re-run the planner after an idea is accepted again). Then delete the split/repair/prune helpers, the shelf, the lineage remap, the unresolved-repair review, the localStorage shelf and the saved-history recovery, and drop the history endpoint if nothing else uses it. Keep the plain draft restore and scopeRecovery.

**Risk.** Needs a small backend validation change. The only behaviour lost is keeping hand-edited dependency edges when an idea is declined and then accepted again, a rare path.

**Visibility.** The review screen loses the 'Re-accept to restore', unresolved-repair and saved-history recovery prompts.

**Related.** Backend GET /api/factory/intents/{intentID}/history (internal/web/api/extension_native.go:198, the apiIntentHistory handler, and the route-table entry in extension_test.go:300) exists only for this recovery path. The declined-reference check in IntentRevision.Validate is in internal/factory/intent.go:257-258.

### REL-1: Factory enrollment needs a hand-written JSON manifest (IDs gathered by hand with psql and the Forgejo CLI) applied by a 1,906-line validate/receipt/rollback engine; three setup binaries each carry their own origin/address check

- **Subsystem:** Release, install, packaging, CI and freshness machinery. **Category:** process-machinery. **Verdict:** partially-confirmed.
- **Removable:** ~1200 lines. **Risk:** medium. **Visible change:** UI. **Behaviour-preserving:** no.
- **Status:** owner decision; left on simplify/release-docs. Replacing the enrollment manifest changes real credential provisioning and Forgejo bindings and needs one end-to-end install to verify; the shared origin check touches three setup binaries that install-docker.sh runs directly.
- **Where:** `cmd/soda-setup/src/factory_enrollment.rs:1-1906`, `cmd/soda-setup/src/cli.rs:1-248`, `cmd/soda-setup/src/origin.rs:1-158`, `cmd/soda-activate/src/origin.rs:1-239`, `cmd/soda-activate/src/activation.rs:1-361`, `cmd/soda-install/src/setup/configure.rs:163-230`, `cmd/soda-install/src/setup/address.rs:1-150`, `cmd/soda-install/src/netip/`, `system/docker/install-docker.sh:200-212`, `docs/guides/operator-setup.md:165-275`, `docs/public/40-Configure/20-factory-authorization.md:136-270`

**What.** Turning the Factory on requires a long manual root procedure. The operator generates five service-account PATs one by one with `podman exec ... gitea admin user generate-access-token`, plus a separate hooks PAT. They then run SQL against the forgejo database to read token and actor IDs, run `forgejo extensions installation soda` to get the installation UUID, and hand-write a `soda-factory-enrollment/v1` JSON manifest with numeric repository/actor/token IDs, provider pins and paths. Only then does `soda-setup --factory-enrollment-file` apply it. The applier is 1,906 lines: a deny_unknown_fields manifest, ~10 validate_* passes, a hook receipt file with consistency checks (receipt_path/read_receipt_bytes/check_receipt_consistency/matching_receipt), snapshot and rollback of dashboard.json and identity.json (Snapshot/rollback/restore_snapshot/restore_services_after_rollback/recover_enrolled_configs), and maintenance gates. Separately, `soda-install configure` shells out to two more binaries, soda-setup and soda-activate. Each of the three has its own origin/address validator: soda-setup/origin.rs, soda-activate/origin.rs (hand-rolled v4/v6 private-range math) and soda-install setup/address.rs plus netip/.

**Simpler.** Add one `soda-setup factory` step that creates the service accounts and tokens through the Forgejo CLI it already calls (PodmanForgejo). It reads back the token and actor IDs and the installation UUID, runs extensions bind, creates the intake hook and writes dashboard.json and identity.json directly. Identity and provider choices come from defaults or a small operator config. Drop the manifest schema and its validate_* passes, the hook receipt file, the snapshot/rollback/recovery layer and the maintenance gates; on failure, report what happened and let the operator rerun. Separately, merge the origin/private-address checks into one std::net::IpAddr-based helper with one chosen policy. Keep the soda-setup and soda-activate entry points, or update install-docker.sh and the release-image checks, since Docker runs them directly.

**Risk.** This touches real credential provisioning and Forgejo bindings. It needs one end-to-end install to confirm tokens, bindings and the hook are created correctly. The pg-provision first-boot path (`soda-setup --provision-db-only`) must keep working.

**Visibility.** The operator procedure and CLI flags change: the manifest file and `--factory-enrollment-file`/`--maintenance` go away, as do the separate soda-setup/soda-activate commands. The resulting services and config files are the same.

**Related.** system/docker/install-docker.sh:207-210 runs soda-setup and soda-activate directly. lib/soda-release-image/src/host.rs:64 and tests/installed/host.sh:33 assert the binaries exist. tools/candidate-setup/src/preflight.rs references them. Development docs (docs/development/factory-remaining-tasks/05-experience-and-development.md, public-handbook cutover/source-audit) describe the manifest.

### FED-4: Delete JEV 'role contribution' invitations

- **Subsystem:** The Feed. **Category:** speculative-abstraction. **Verdict:** confirmed.
- **Removable:** ~1150 lines. **Risk:** low. **Visible change:** UI, model tools. **Behaviour-preserving:** no.
- **Status:** owner decision; JEV calls kept: an invited verdict puts a contribution invitation in the role's inbox and lets it reply in the thread, so the answer reaches a worker and the UI. Deleting the feature is a product decision.
- **Where:** `internal/factory/feed_contribution.go:1-132`, `internal/store/factory_feed_contribution.go:1-621 (keep isOwnDiscussionReplyTx ~536-550)`, `internal/web/api/factory_feed_contribution.go:1-120`, `internal/web/api/factory_feed_comment.go:1032`, `internal/store/factory_mcp_communication.go:43-57,96-99`, `internal/store/factory_mcp_inbox.go:18-22,43-60,131-139,164-195`, `internal/factory/mcp_feed.go:6,21`, `internal/store/factory_feed_comment.go:32`, `internal/web/api/factory_mcp_communication.go:76`, `internal/web/api/factory_mcp.go:911`, `frontend/spaces/sodaspaces-factory-consultation.ts:10-28`, `frontend/spaces/sodaspaces-factory-feed.ts:2467-2645,3998-4007`, `frontend/spaces/sodaspaces-factory-feed-response.ts:1683-1731`

**What.** Each coordinator progress pass looks for sources from the last 24 hours while another role is running: human posts, top-level worker posts, F11 posts, and finished assignment records. For one source per pass it asks JEV 3 fixed formulations for each eligible role (up to 4 roles, so up to 12 calls, with a 15-second timeout) whether that role should comment. Around that sit: - a 'contribution' reply post with 7 states - a digest of the thread (a recursive CTE) and snapshot re-checks under share locks - claim, settle and expire passes - three SQL legs that consume an invitation when the invited worker replies through factory.reply. The browser renders the assessment with thresholds and per-call answers, and has 180 lines of race-guarded paging just to scroll to the role's reply.

**Simpler.** Delete: - driveContributions, assessContribution and judgeContributions - ContributionSources, ContributionContext, ClaimContributionReview, SettleContributionReview and ExpireContributionReviews - contributionReplyTx - the three invitation legs in admitWorkerDiscussionReplyTx, which reduces it to isOwnDiscussionReplyTx plus a short refusal - the contribution legs in the inbox and discussion CTE and the record-invitation path in MCPDiscussion - the MCPFeedMessage.Contribution projection and trimming - the discussion_guidance text - renderContribution, FeedContribution and openRoleReply/runRoleReply. Workers reply only in their own threads. A human who wants a role's input replies on that role's post, which already reaches its inbox through reply_to.

**Risk.** Not in the demo checklist. Existing contribution posts need a one-line filter (data ? 'contribution'), or can render as plain text.

**Visibility.** Threads lose the JEV contribution-assessment replies and the 'Read the role's reply' link. The worker read body loses the invitation guidance and inbox items lose contribution data.

**Related.** The same pattern of several JEV calls with fixed formulations behind a gate also appears in consultation_guidance (factory_mcp.go:914) and in the priority-judgment budget and receipt machinery (ReservePriorityJudgment/RecordPriorityJudgment). Many docs under docs/design and docs/architecture describe role contributions and would need trimming, specifically factory-feed-components.md, factory-worker-mcp.md and trust.md.

### STO-1: Three schema-evolution mechanisms plus per-open shape verification, duplicated in Go and the Rust identity broker, for a zero-user product

- **Subsystem:** Store (PostgreSQL layer) structure. **Category:** dead-code. **Verdict:** partially-confirmed.
- **Removable:** ~1100 lines. **Risk:** medium. **Visible change:** DB schema. **Behaviour-preserving:** yes.
- **Status:** owner decision.
- **Where:** `internal/store/schema.go:280-307`, `internal/store/schema.go:309-415`, `internal/store/schema.go:453-558`, `internal/store/store.go:83-125`, `internal/store/factory_library.go:17-25`, `internal/store/factory_research.go:12-18`, `internal/store/factory_post_resolution.go:12-20`, `internal/store/factory_artifacts.go:15-35`, `internal/store/observe.go:22-95`, `internal/store/application_setup.go:17-30`, `internal/store/application_setup.go:201-266`, `internal/web/api/application_setup.go:193-196`, `internal/store/schema_test.go`, `internal/store/application_schema_test.go`, `internal/store/observe_test.go`, `cmd/soda-identity/src/schema.rs:1-30`, `cmd/soda-identity/src/schema.rs:317`, `cmd/soda-identity/src/store_schema.rs:78-215`, `cmd/soda-identity/tests/broker.rs:781-811`, `lib/soda-release-image/src/model.rs:43-44`, `internal/store/factory_delivery_schema.go:12-20`, `internal/store/factory_repair_schema.go:12-20`, `internal/store/factory_delivery_repair_schema.go:30-38`

**What.** The schema evolves in three different ways. (1) A versioned CREATE plus a hand-written 68->69 upgrade, with an in-process mutex (schemaOpenMu) and a cross-process 'lost upgrade race' re-read (reopenAfterLostUpgradeRace). (2) Four tables (library items/revisions, research records, post resolutions, feed artifacts) created lazily with CREATE TABLE IF NOT EXISTS outside the versioned script; one of them also carries an ALTER TABLE ... ADD COLUMN IF NOT EXISTS 'convergent shape' migration. (3) Three CHECK-rewriting migrations (STO-3). On top of that, every dashboard open runs 58 'SELECT ... LIMIT 0' column probes and 6 trigger-existence probes inside the open transaction, and refuses anything else. configureStore chains 8 initialize* steps.

**Simpler.** At open, run one idempotent current script in both Go and the Rust broker: CREATE ... IF NOT EXISTS for every table and index, with the 4 lazy tables folded in, CREATE OR REPLACE for functions, and seed rows with ON CONFLICT DO NOTHING. The cleaner option is a single owner: the Rust broker would just require the Go-created tables, with no mirrored DDL and no verification. Delete the 68->69 upgrade in both languages, along with reopenAfterLostUpgradeRace / reopen_after_lost_upgrade_race, schemaOpenMu, refusal of unversioned databases, verifyRequiredColumns / verify_required_columns, the trigger probes, ReadSchemaVersion, IntegrityCheck, RenewApplicationSetup, MaxSetupRenews and the renewals table. Delete schema_version only together with STO-3, whose CHECK rewrites lock it. Also drop the stale release-image SCHEMA_VERSION or point it at the right source. An incompatible shape change means recreating the dev database.

**Risk.** A fresh database gets the identical shape, and retained 69 databases open unchanged. The only behaviour lost is refusing foreign or partial databases at open, a guard no zero-user deployment needs. About 610 of the lines are tests that cover only this machinery.

**Visibility.** No UI or API change. Drops schema_version and the never-written application_setup_renewals table; no data migration. The 'Open the production store' paragraph of internal/store/README.md needs a one-line update.

**Safe first step.** Delete ReadSchemaVersion and (*Store).IntegrityCheck from internal/store/observe.go. Neither has a production caller. Trim internal/store/observe_test.go (TestReadSchemaVersionAndOpenObserve) to cover only OpenObserve, or delete it. Leave OpenObserve in place, since it is used by internal/acceptance/factory_context_status.go. Separately, delete RenewApplicationSetup and MaxSetupRenews from internal/store/application_setup.go. Nothing outside tests calls them, but keep the application_setup_renewals table for now, because dropping it is a schema change that must happen in Go and Rust at once.

**Related.** The Rust identity broker mirrors the full schema and open logic: cmd/soda-identity/src/schema.rs and store_schema.rs, with tests in cmd/soda-identity/tests/broker.rs. lib/soda-release-image/src/model.rs holds a stale SCHEMA_VERSION=68 that is stamped into the release payload. STO-3's CHECK-rewriting initializers (factory_delivery_schema.go, factory_repair_schema.go, factory_delivery_repair_schema.go) depend on schema_version. internal/store/ephemeral.go:131 keys test databases by SchemaVersion(). internal/store/README.md describes the refuse-on-open behaviour.

### FEF-3: Seven budget controls across five screens; the planning-call budget alone appears in four places and rations JEV calls, which the owner says are free

- **Subsystem:** Frontend: Factory settings, work, progress, request, intent screens. **Category:** too-many-states-or-budgets. **Verdict:** confirmed.
- **Removable:** ~1000 lines. **Risk:** medium. **Visible change:** UI, DB schema. **Behaviour-preserving:** no.
- **Status:** owner decision; coordinate with prod-hardening.
- **Where:** `frontend/spaces/sodaspaces-factory-repair.ts:1-291`, `frontend/spaces/sodaspaces-factory-verification-allowance.ts:1-263`, `frontend/spaces/sodaspaces-intent-view.ts:94-120,400-436`, `frontend/spaces/sodaspaces-intent.ts:2122-2321`, `frontend/spaces/sodaspaces-factory-work.ts:581-690,704-717`, `frontend/spaces/sodaspaces-factory-progress-view.ts:408-441,1004-1059`, `frontend/spaces/sodaspaces-factory-progress-response.ts:438-494,773-797`, `frontend/spaces/sodaspaces-intent-response.ts:124-142`, `frontend/spaces/sodaspaces-factory-settings.ts:1498-1501,1575-1576,1595,1601-1629`, `internal/store/factory_intents.go:328-333`, `internal/store/factory_intent_allowances.go`, `internal/store/factory_verification_allowance.go`, `internal/store/factory_delivery_repair_budget.go`, `internal/store/schema.go:44`

**What.** Limits the owner faces. Settings: sponsorship allowance minutes and concurrent sessions, connection rolling minutes, minutes per attempt, correction cycles, intent minutes. Repair funding: investigator runs (max 4), proposal judgments (max 4) and active minutes (max 120), with a mandatory 'Effect of this change' text before raising any of them. Verification: runs and minutes per request, increase only. Planning calls per request appear four times: set at request time under 'More options', raised in the review ('Planning budget … Raise allowance'), adjusted again in the direction view ('New amount (0–100 planning calls)' with an effect note), and listed read-only in settings with reserved, consumed and ceiling.

**Simpler.** Keep one minutes allowance per account plus minutes per attempt in the Factory form. Remove the planning-call allowance (its UI and the ReserveIntentJudgment gate, so JEV consultations are never rationed), the repair funding panel and the verification allowance panel. Make the backend gates in factory_intent_allowances.go, factory_verification_allowance.go and factory_delivery_repair_budget.go go away at the same time. Keep the hardening branch's Feed allowance notice as the only budget readout.

**Risk.** The backend must stop gating on these budgets first, or fund them automatically. Otherwise work waits with no UI left to raise the limit.

**Visibility.** Removes the repair panel, the verification panel, the planning budget in the request review, the direction-view allowance form and the settings planning readout. A schema change is needed only if the backing allowance and grant tables are dropped too; removing the UI alone needs none.

**Related.** Backend gating in internal/store/factory_intent_allowances.go (ReserveIntentJudgment, ErrPlanningAllowanceExhausted), internal/factory/verification_allowance.go, internal/store/factory_verification_allowance.go, internal/store/factory_delivery_repair_budget.go, internal/store/factory_repair_scope.go and internal/factory/stall.go. Tables are in internal/store/schema.go (factory_intent_planning_calls and the allowance and grant tables). The prod-hardening branch adds frontend/spaces/sodaspaces-factory-allowance.ts (a Feed hold notice) but removes none of these budgets.

### FEP-3: About 2,300 lines of read*/set* closure bags exist only to split four components across files

- **Subsystem:** Frontend: Workspace, project, applications, terminals, layout, Forgejo and Tailnet pages. **Category:** pass-through-wrapper. **Verdict:** confirmed.
- **Removable:** ~1000 lines. **Risk:** low. **Visible change:** none. **Behaviour-preserving:** yes.
- **Status:** open (behaviour-preserving).
- **Where:** `frontend/spaces/sodaspaces-workspace.ts:892-913,1296-1414,1560-1615,1679-1950,1990-2028,2063-2094`, `frontend/spaces/sodaspaces-project.ts:313-355,424-492,501-605,661-820`, `frontend/spaces/sodaspaces-terminal.ts:160-214,235-420`, `frontend/tailnet/soda-tailnet-page.ts:161-293,395-447`, `frontend/spaces/sodaspaces-project-refresh.ts:31-88`, `frontend/spaces/sodaspaces-project-settings-view.ts:87-149`, `frontend/spaces/sodaspaces-project-journey-view.ts:8-44`, `frontend/spaces/sodaspaces-terminal-attachment.ts:10-50`, `frontend/spaces/sodaspaces-terminal-actions.ts:7-34`, `frontend/spaces/sodaspaces-terminal-screen.ts:10-35`, `frontend/tailnet/soda-tailnet-observation.ts:21-49`, `frontend/tailnet/soda-tailnet-actions.ts:44-80`, `frontend/spaces/sodaspaces-project-network.ts:23-33`, `frontend/spaces/sodaspaces-project-runtime.ts:30-32`, `frontend/spaces/sodaspaces-project-request.ts:95-117`

**What.** SodaSpaces, SodaProjectControls, SodaTerminal and SodaTailnet have 33 builder methods. Each builds a fresh object of readX/setX/isX closures on every call, so functions in sibling files can read and write the component's private fields, and each sibling file declares a matching *Input/*Reading/*Actions interface. Several module functions only forward to the closure they were handed: setStopConfirmed, setCreateNetworkEnabled, setNetworkConfirmed, beginEpoch, announceChanged, and announceObserved, which copies five fields 1:1.

**Simpler.** Turn the module functions into methods of their component, or have them take the component instance through `import type`. Pass plain values to the stateless view functions. Build any view input at most once per render. Delete the *Input/*Reading/*Actions interfaces, the 33 builders, and the forwarding helpers setStopConfirmed, setCreateNetworkEnabled, setNetworkConfirmed, beginEpoch, announceChanged and announceObserved. Behaviour stays identical.

**Risk.** Mechanical but wide. The 1,300 lines count project/terminal/tailnet. About 1,000 more sit in Workspace files that FEP-1/FEP-5 delete. Coordinate with ui/repo-places edits to sodaspaces-project.ts.

**Visibility.** None; internal structure only.

**Safe first step.** Delete the six forwarding helpers and call the component directly: - setStopConfirmed in frontend/spaces/sodaspaces-project-runtime.ts - setCreateNetworkEnabled and setNetworkConfirmed in frontend/spaces/sodaspaces-project-network.ts - beginEpoch, announceChanged and announceObserved in frontend/spaces/sodaspaces-project-request.ts The call sites are settingsViewInput in sodaspaces-project.ts, where `this.stopConfirmed = checked` etc. replace the helper calls, and the refresh/request callers. Then compute settingsViewInput() once per render in SodaProjectControls.render().

**Related.** The Workspace files (sodaspaces-workspace*.ts) use the same pattern on a larger scale, for example inventoryInput. Those files are slated for deletion under FEP-1/FEP-5.

### PLN-1: Per-issue 'planning-freshness' JEV call gates every first dispatch, but no worker ever sees its answer

- **Subsystem:** Intent to plan (Product Lead, Architect, research, decomposition). **Category:** other. **Verdict:** confirmed.
- **Removable:** ~1000 lines. **Risk:** low. **Visible change:** UI, DB schema. **Behaviour-preserving:** no.
- **Status:** done (`9ac11119`): PlanningPass, the plan_not_current gate, the planning record store, the F11 strategy card and the plan-view plan/freshness section are gone; the factory_planning table stays in the schema.
- **Where:** `internal/factory/control/planning.go:18-23,31-299`, `internal/factory/control/planning_admission.go:129-150`, `internal/factory/control/dispatch.go:351,436,454-461`, `internal/factory/control/dispatch_attempt.go:185-189`, `internal/factory/control/coordinator.go:108`, `internal/factory/planning.go:193-259`, `internal/store/factory_planning.go:1-66`, `internal/store/factory_feed_plan.go:15-37`, `internal/store/factory_intents.go:420-439`, `internal/factory/mcp_communication.go:38,98-108`, `internal/web/api/factory_plan_view.go:15-154,344-361`, `internal/store/schema.go:38`, `frontend/spaces/sodaspaces-factory-progress-view.ts:678-720,980`, `frontend/spaces/sodaspaces-factory-progress-response.ts:237-370,676-690`, `frontend/spaces/sodaspaces-factory-feed-communication.ts:74`, `frontend/spaces/sodaspaces-factory-feed-response.ts:349,1123`

**What.** Every Dispatch pass first runs PlanningPass. For each accepted issue that is queued or blocked, it rebuilds a full dispatch plan (authority, sponsorship, preparation, gate) and asks JEV (purpose planning-freshness) to pick a label: direct_small_change, incremental_acceptance or prerequisite_first, or plan_stable_scope / wait_for_outputs while the issue is blocked. The result is stored in factory_planning and posted as an F11 Feed card. launchDispatchPlan then refuses any code issue unless planningCurrent() finds a matching non-provisional record with Outcome 'planned'; otherwise the issue waits with reason plan_not_current.

**Simpler.** Delete PlanningPass, planIssue, planningInput, admitPlanningReadiness/Authority, planningCallFailed, planningCurrent and planningRecordCurrent, together with the PlanCurrent dependency, the 'planning' stage and the dispatch gate. Also delete PlanningInput, PlanningRecord, PlanningSnapshot and ReadyToPlan, store/factory_planning.go and its table, ReserveIssuePlanning, recordPlanningFeedTx and FeedPlanningSource, investigationCovered, planningQueue, the JudgmentPlanningFreshness purpose, and the plan/freshness half of the issue plan view (keep its priority comparisons). PlanningJudgments keeps only Chooser, which many other JEV purposes still use. The Architect's investigation plan already decides the per-issue approach.

**Risk.** Only the dispatch gate, the F11 card and the issue plan view consume these records. The Architect investigation path does not depend on them. The frontend progress view and Feed renderers need their planning branches removed alongside.

**Visibility.** The F11 Feed cards 'Implementation strategy recorded / Provisional ... / unavailable' and the issue plan view's plan and freshness section disappear. Issues no longer wait in stage 'planning' (plan_not_current). The factory_planning table becomes unused; drop it or leave it.

**Related.** The same AdvisePurpose rule (a failed receipt is never retried while inputs stay unchanged) applies to the other JEV purposes in judgment_advice.go: decomposition, completion, repair proposals and delivery repair proposals. c.Planning == nil checks elsewhere (completion_decision.go, delivery_judgment_producer.go, repair_proposal.go) need only the Chooser.

### BUD-7: Per-intent planning-call allowance: a second JEV budget with its own revision chain, three tables and UI in four views

- **Subsystem:** Budgets, allowances and limits (cross-cutting). **Category:** too-many-states-or-budgets. **Verdict:** confirmed.
- **Removable:** ~900 lines. **Risk:** low. **Visible change:** UI, DB schema. **Behaviour-preserving:** no.
- **Status:** done (`c62f6ec1`).
- **Where:** `internal/factory/intent_allowance.go:1-63`, `internal/store/factory_intent_allowances.go:1-161`, `internal/store/factory_intents.go:325-438`, `internal/store/factory_access_reads.go:12-60`, `internal/store/schema.go:44-46`, `cmd/soda-identity/src/schema.rs:39-41`, `cmd/soda-identity/src/schema.rs:352-353`, `internal/web/api/extension_native.go:199`, `internal/web/api/extension_native.go:205`, `internal/web/api/factory_intents.go:169-212`, `internal/web/api/factory_intent_explore.go:109-120`, `internal/web/api/factory_intent_explore.go:163-170`, `internal/web/api/factory_access_view.go:216-254`, `internal/web/api/factory_access_view.go:396-404`, `internal/factory/intent.go:81`, `internal/factory/intent.go:126`, `internal/factory/control/planning.go:205-215`, `frontend/spaces/sodaspaces-intent.ts:2109-2200`, `frontend/spaces/sodaspaces-factory-work.ts:604-640`, `frontend/spaces/sodaspaces-factory-settings.ts:1600-1633`, `frontend/spaces/sodaspaces-factory-progress-view.ts:364-460`, `frontend/spaces/sodaspaces-factory-progress-view.ts:1010-1060`, `frontend/spaces/sodaspaces-factory-progress-response.ts:438-500`

**What.** Product Lead exploration and planning must be funded with 'planning calls'. Each allowance is a CAS-chained record with amount, ceiling and effect text, and consumption is derived from joins.

**Simpler.** Delete planning allowances: - the three tables and their Rust mirror; - PlanningAllowance, with its store, API routes and DTOs; - the allowance half of reserveIntentJudgmentTx, which keeps only the fingerprint dedupe and, for now, the global judgment budget; - the 409 planning_funding_required and the PlanningAllowanceAvailable pre-check; - the intent-page allowance poster, the factory-work allowance editor, the Settings 'Planning calls' section, and the progress-view allowance sections and parsers; - IntentRevision.PlanningCalls.

**Risk.** Removes a gate in front of a free service; exploration still makes one judge call per distinct fingerprint.

**Visibility.** The 'Planning calls' input, the allowance editors and the allowance sections disappear, and exploration never returns 409 planning_funding_required.

**Related.** ReserveIntentJudgment also charges the global factory_judgment_budget (spent+1), a separate JEV ceiling. Other per-thing allowances follow the same pattern: factory_attempt_allowances and the verification and Strategist allowances (sodaspaces-factory-verification-allowance.ts, sodaspaces-factory-repair.ts:209).

### FED-3: Stop re-validating the server's own Feed JSON field by field in the browser; send status as fields, not a composed string

- **Subsystem:** The Feed. **Category:** unneeded-validation-or-digest. **Verdict:** confirmed.
- **Removable:** ~900 lines. **Risk:** low. **Visible change:** UI. **Behaviour-preserving:** yes.
- **Status:** partly done on `simplify/feed-render`: the browser maps snake_case once and skips a malformed entry instead of failing the page. Record status is still one composed string (the judge and MCP snapshots read it).
- **Where:** `frontend/spaces/sodaspaces-factory-feed-response.ts:435-1290,1872-1925`, `frontend/spaces/sodaspaces-factory-feed-families.ts:139,1353`, `internal/factory/feed_contract.go:97-392`, `internal/store/factory_feed_record.go:41-147,273-298,300-655`, `internal/store/factory_feed_plan.go:41-75`, `internal/store/factory_feed_events.go:20-45`, `internal/store/factory_library.go:465-470`, `internal/factory/mcp_communication.go:175-250`

**What.** Go projects each canonical result into a parallel tree of 34 Feed* structs, with every list capped at 24 and paired with a *_total field. The browser mirrors these as 43 TS interfaces and re-parses every field with 120 check() calls. Those checks duplicate the Go bounds: title ≤160, text ≤2048, list ≤24, entry ≤512, plus commit, digest and artifact-href regexes. Along the way the parser copies snake_case into camelCase. - A mismatch inside a payload turns that card's payload into {invalid:true}, which renders as 'Recorded detail unavailable'. - A mismatch in the entry envelope (unknown activity kind, status over 64 characters) throws and blanks the whole page, because parseFeedPageView has no per-entry catch. Record status works the same way. Go builds a string like 'phase:stage:outcome:verdict>findings:N>checks:N>failures:N>published>merged>completed', truncates it to 64 characters (RecordStatus), and parseFeedStatusSegments splits it back apart only to feed familyBlocks.

**Simpler.** - Replace the check()-based parsers with TS interfaces that mirror the Go JSON, using a plain cast or one mechanical snake_case-to-camelCase mapper. - Add a per-entry try/catch in parseFeedPageView. - Send record status as structured fields and delete RecordStatus's segment dropping and parseFeedStatusSegments. - Keep text bounds only in the MCP decoders and in FeedCommunication.Validate for worker posts.

**Risk.** The only thing dropped is trust-nothing parsing of same-origin JSON produced by this server; Lit already escapes text. Field names change if status becomes structured. Overlaps with the comment-type parsing removed in FED-5.

**Visibility.** The only visible effect: cards that today degrade to 'Recorded detail unavailable', such as long-approach F11 posts and long-titled library posts, start rendering.

**Safe first step.** In frontend/spaces/sodaspaces-factory-feed-response.ts, remove the length, regex and count check() calls inside parseCommunicationPayload and the other per-family payload parsers. Keep only the snake_case-to-camelCase field copies and the presence checks for required objects. For well-formed server data the parsed output is identical; the only change is that cards which exceeded browser-only limits stop degrading. Strictly identical behaviour holds only for payloads that already pass. Next, wrap each entry in parseFeedPageView in a try/catch.

**Related.** FED-5: the comment-type parsing is the same pattern. frontend/spaces/sodaspaces-factory-feed-families.ts consumes parseFeedStatusSegments. Other *-response.ts parsers in frontend/spaces likely re-validate server JSON the same way.

### PLN-2: Per-intent planning-call allowance (revisioned CAS chain, ledger, global counter) rations a free service and stalled the demo

- **Subsystem:** Intent to plan (Product Lead, Architect, research, decomposition). **Category:** too-many-states-or-budgets. **Verdict:** confirmed.
- **Removable:** ~900 lines. **Risk:** low. **Visible change:** UI, DB schema. **Behaviour-preserving:** yes.
- **Status:** done (`c62f6ec1`).
- **Where:** `internal/factory/intent_allowance.go:1-63`, `internal/store/factory_intent_allowances.go:1-161`, `internal/store/factory_intents.go:267-300,328-407`, `internal/store/factory_reassessments.go:654-671`, `internal/store/factory_attempt_allowances.go:1232-1245`, `internal/store/factory_context_selection.go:24-58`, `internal/store/factory_delivery_repair_adoption.go:70-86`, `internal/factory/control/planning.go:212`, `internal/web/api/factory_intents.go:169-212`, `internal/web/api/factory_intent_explore.go:109-119,163-171`, `internal/factory/intent.go:81,126`, `internal/factory/control/reassessment_adoption.go:950`, `internal/web/api/factory_access_view.go:211-270,378-404`, `internal/store/factory_access_reads.go`, `internal/store/schema.go:35-36,44-46,385-386`, `cmd/soda-identity/src/schema.rs:30-31,39-41,352-353`, `frontend/spaces/sodaspaces-intent.ts:306-310,380,2109-2284`, `frontend/spaces/sodaspaces-intent-view.ts:409-436`, `frontend/spaces/sodaspaces-intent-response.ts:131-158`, `frontend/spaces/sodaspaces-factory-work.ts:566-633`, `frontend/spaces/sodaspaces-factory-access-response.ts`

**What.** Every intent JEV consultation must reserve against a per-intent PlanningAllowance: explore design, assumptions, consolidation, consistency, planned choice, and the per-issue planning calls for generated issues (PLN-1). The allowance is its own revisioned record: a predecessor CAS chain plus head table, amount/ceiling, the unit 'planning_calls' (the only value), mandatory 'effect' prose for any increase, a reservation ledger (factory_intent_planning_calls) and a global factory_judgment_budget counter. The frontend creates and raises it from two different screens and mirrors the amount into IntentRevision.PlanningCalls.

**Simpler.** Delete PlanningAllowance end to end: the type, the store, the POST/GET API, both UIs and the access-view rows. Also delete IntentRevision.PlanningCalls, the allowance heads/revisions tables and factory_intent_planning_calls, in both the Go and Rust schemas. Remove factory_judgment_budget and its read/increment in all seven reservation paths. Keep the per-snapshot de-duplication: insert factory_priority_judgments with ON CONFLICT DO NOTHING and count nothing.

**Risk.** Keep ReservePriorityJudgment's per-snapshot de-duplication for priority tournaments. Remove the 'planning_funding_required' branches in the explore UI and e2e scripts.

**Visibility.** The 'Planning budget' section, the 'Planning calls' input, the raise flows on the intent and Work screens, the access-view planning rows and the 409 planning_funding_required error disappear.

**Safe first step.** Remove the factory_judgment_budget accounting that nothing reads. Delete the `SELECT spent FROM factory_judgment_budget ... FOR UPDATE` scan into the unused `spent` variable, and the `UPDATE factory_judgment_budget SET spent=spent+...`. Do this in internal/store/factory_intents.go (ReservePriorityJudgment ~283/296 and reserveIntentJudgmentTx ~383/404), factory_reassessments.go ~655/669, factory_attempt_allowances.go ~1233/1243, factory_context_selection.go ~25/56 and factory_delivery_repair_adoption.go ~71/84. Delete the tests that read `spent`. The table can be dropped later in schema.go and cmd/soda-identity/src/schema.rs.

**Related.** The same unchecked global factory_judgment_budget counter is incremented by reassessment, attempt-allowance, context-selection and delivery-repair reservations. Per-attempt allowances (factory_attempt_allowances.go) look like a similar rationing pattern for free JEV calls. The schema is mirrored in cmd/soda-identity/src/schema.rs.

### CTX-4: Context-bundle selection support machinery (owner 'use current context' override, semantic digest, coverage facts, history preview) exists only to serve the paid JEV bundle pick

- **Subsystem:** Worker context assembly, project library and research. **Category:** speculative-abstraction. **Verdict:** partially-confirmed.
- **Removable:** ~800 lines. **Risk:** low. **Visible change:** UI, model tools. **Behaviour-preserving:** no.
- **Status:** owner decision; coordinate with prod-hardening.
- **Where:** `internal/factory/control/context_bundle_state.go:1-238`, `internal/factory/control/context_bundle_advice.go:1-149`, `internal/factory/control/context_resolution.go:1-109`, `internal/factory/context_resolution.go:1-67`, `internal/store/factory_context_resolution.go:1-101`, `internal/factory/control/context_bundle_prepare.go:1-69`, `internal/factory/control/dispatch_launch.go:745-756`, `internal/web/api/factory_mcp.go:844-879`, `internal/web/api/factory_lifecycle.go:159-177`, `internal/web/api/factory_views.go:172`, `internal/factory/mcp_submission.go:40`, `internal/acceptance/factory_context_status.go:221,296`, `frontend/spaces/sodaspaces-factory-context-resolution.ts:1-249`, `frontend/spaces/sodaspaces-factory-response.ts:187,413`, `frontend/spaces/sodaspaces-project-work.ts:12-13,151,420`, `frontend/spaces/sodaspaces-factory-work.ts:3,1301`, `tests/installed/qualification/phases/factory-phase.ts`, `tests/installed/qualification/phases/factory-command.ts`, `tests/installed/qualification/check-browser.ts`

**What.** loadContextBundleState computes the full CurrentContext, a SHA-256 of the role JSON, a 'semantic digest' that blanks fields, and coverage facts (field names, source/resolution/check counts, plan task/receipt/acceptance-ref/owned-path counts). It also builds a history preview that checks MCPRunWasObserved per run. All of this exists only to feed the context-bundle-v1 JEV call choosing current/attempt/lineage/hold. When that call stalls, an owner 'use-current-context' command, with its own Command type, fingerprint/scope/required-digest triple check, capability jsonb_set, run/assignment FOR UPDATE locks, grant reauthorisation and a 249-line frontend panel, records a ContextCurrentResolution. foregroundReadBundle then recomputes the context and digest on every read to honour it.

**Simpler.** Pick the bundle deterministically from factory.BundleEligibility: serve current, add attempt.run_history when attempt-eligible and the lineage fields when lineage-eligible. Delete context_bundle_state.go (move the required-current completeness check, with the Test Pilot verification injection, into a small launch gate), context_bundle_prepare.go, context_bundle_advice.go, both context_resolution.go files, store/factory_context_resolution.go, the use_current_context action branch in factory_lifecycle.go, the ContextSelectionStatus field in factory_views.go/acceptance, ContextCurrentResolution on the capability, the resolution branch and receipt replay in foregroundReadBundle, and the frontend context-resolution panel and its wiring. Keep markRequiredContextHold and the serveRead 'hold' action for incomplete required current context, which is independent of JEV. Update the installed qualification scripts that drive use_current_context.

**Risk.** Removes a paid call and its escape hatch together. The only behaviour lost is a model choosing to hide own-attempt history, which a worker can simply ignore.

**Visibility.** The run detail's context-selection panel and 'Use current context' button disappear. factory.context drops context_selection action/reason_code hold states.

**Related.** The JEV call itself (context_bundle_advice.go, the context-bundle judgment purpose) overlaps DSP-1/RTY-4. MCPRunWasObserved is shared with dispatch_context_recovery.go and must stay. The installed qualification harness (tests/installed/qualification) and docs/development/rc6-qualification-2026-10-25.md reference use_current_context. prod-hardening (the local name of the hardening branch) touches factory_mcp.go and mcp_submission.go only for refusal wording, so there is a textual-conflict risk but no semantic overlap.

### JDG-3: One yes/no delivery judgment carries receipts, five effective actions and a full rebuild at merge time

- **Subsystem:** Completion, reassessment, review and JEV judgments. **Category:** unneeded-validation-or-digest. **Verdict:** confirmed.
- **Removable:** ~800 lines. **Risk:** medium. **Visible change:** UI. **Behaviour-preserving:** yes.
- **Status:** owner decision.
- **Where:** `internal/factory/control/delivery_judgment_material.go:16-209`, `internal/factory/control/delivery_judgment_producer.go:13-16,127-227,262-307`, `internal/factory/control/judgment_advice_delivery.go:12-137`, `internal/factory/control/delivery_typesafe.go:1-42`, `internal/factory/judgment.go:270-576`, `internal/factory/judgment_delivery.go:1-202`, `internal/factory/control/delivery_acceptance.go:267-357`, `internal/factory/control/delivery_producer.go:79-101`, `internal/store/factory_attempt_allowances.go:1189-1302`, `internal/factory/control/publication_children.go:168-196`, `internal/factory/control/merge_judgment_gate.go:265-305`

**What.** When the Inspector requests changes with optional-only, disputed or repetitive feedback (routeDeliveryFeedback, publication_children_routing.go:57-102), SodaOS asks JEV delivery-review-v1 to choose ready_to_merge or review_again. Around that single choice sit: a 'complete material' snapshot of up to 256 KiB; a 16-field fingerprint input with 5 content digests plus a whole-material digest; a funded request ID; a receipt whose Calls list always has one entry; a decision triple; five effective actions; an acceptance step that recomputes the fingerprint; and a merge-time path that rebuilds all of it to find the receipt again.

**Simpler.** Use a single judged review step. If the Inspector clears and checks pass, merge. If there are actionable unfixed findings, run the Fixer, counted against one ReviewRounds limit. Only for disputes or optional-only feedback, ask JEV once with the accepted outcomes, diff, findings, dispositions and check results. Store round, head, selected option, confidence, probability and model on the publication's DeliveryAcceptance. At merge admission, compare the stored head and round with the current ones instead of rebuilding the material. Treat low confidence as review_again while rounds remain, otherwise show one 'needs owner' wait. If the call fails, retry on the next pass. Delete judgment_delivery.go, the DeliveryFingerprintInput/DeliveryAnswer/DeliveryDecision machinery in judgment.go, judgment_advice_delivery.go, delivery_typesafe.go, currentFundedDeliveryAcceptance, checkJevFingerprint, the test-only AdviseMergeJudgment, and the store's reserve/record/read trio.

**Risk.** Merge authorization (merge_authorization.go, and CheckMergeDecisionGate in merge_operation.go) consumes the DeliveryDecision and Fingerprint fields, and the journey renders delivery receipts. Those readers must switch to the stored acceptance fields in the same change.

**Visibility.** Journey delivery judgments keep rendering, now from the acceptance. The waits judgment_attention, review_allowance_exhausted and delivery_judgment_unavailable collapse into fewer states that can be retried. Assembling the material once per decision and dropping the repeated fingerprint and validation passes is behaviour-identical.

**Safe first step.** Delete the production-dead code along with the tests that cover it. - internal/factory/control/merge_judgment_gate.go: remove AdviseMergeJudgment (lines ~265-305); its only callers are in merge_judgment_gate_test.go. - internal/factory/judgment_delivery.go: remove DeliveryReceipt.ConfirmExpectedResolution (~134-160); it is called only from that test. Then remove the redundant re-validation passes. - Drop the duck-typed ValidateDeliveryState hook in AdviseDelivery (judgment_advice_delivery.go:80-84), because the producer has already run material.validate(). - Drop the re-validate and re-fingerprint in TypeSafePriority.SelectDelivery (delivery_typesafe.go:19-36) by passing the producer's fingerprint through. - Delete the `var _ DeliveryFunding` assertion. Observable behaviour stays identical.

**Related.** The same receipt/fingerprint/reservation pattern for funded judgments probably exists in the other JEV judgment paths (plan/intent judgments in internal/factory/judgment.go, and allowance reservations in internal/store/factory_attempt_allowances.go). MergeAuthorization.Decision in internal/factory/merge_operation.go embeds the full DeliveryDecision too.

### TRM-6: Guest keeps one systemd unit + tmux server + privileged prepare hook + per-terminal record directory for each terminal, where sessions on one per-account tmux server would do

- **Subsystem:** Human terminal stack end to end (browser, Go API, host, in-project helper). **Category:** speculative-abstraction. **Verdict:** partially-confirmed.
- **Removable:** ~800 lines. **Risk:** high. **Visible change:** UI. **Behaviour-preserving:** no.
- **Status:** owner decision.
- **Where:** `cmd/soda-project-terminal/src/term_create.rs:76-287`, `cmd/soda-project-terminal/src/term_prepare.rs:1-243`, `cmd/soda-project-terminal/src/term_binding.rs:1-305`, `cmd/soda-project-terminal/src/term_status.rs:1-225`, `cmd/soda-project-terminal/src/term_collect.rs:1-176`, `cmd/soda-project-terminal/src/svc.rs:1-166`, `cmd/soda-project-terminal/src/tmux.rs:1-83`, `cmd/soda-project-terminal/src/term_paths.rs:1-103`, `cmd/soda-project-terminal/src/term.rs:1-145`, `cmd/soda-project-terminal/src/subscription_prepare.rs`, `cmd/soda-project-terminal/src/subscription_profile.rs`, `internal/host/terminal.go:161`, `frontend/spaces/sodaspaces-terminal-response.ts:21-42`

**What.** Each terminal is a /run/soda-terminals/<id> directory holding root-owned binding, reservation, name (with name.next staging-file recovery), writer, tmux.conf and screen/ files. It runs as a systemd-run transient unit with its own 'tmux -D' server. The privileged ExecStartPost '+project-terminal prepare <id>' hook proves its own cgroup membership by reading cgroup.procs, waits for the socket, chowns and seals screen/, records the socket dev/ino and MAINPID in a 'ready' record, and only then creates the tmux session. Status is derived from systemctl show plus cgroup emptiness plus the reservation plus a socket dev/ino comparison plus a writer flock probe. The code re-checks the binding account tuple against getpwnam on every operation, and its comments say this replicates the Python implementation's comparison exactly ('Exact .py comparison').

**Simpler.** Give each project account one tmux server on a private socket, run as that user. Each terminal is a session named by id, and the display name is a session option. - list/inspect: `tmux list-sessions -F` with #{session_attached} - end: kill-session (optionally followed by a pkill of the session's process group) - rename: set-option - attach: `tmux attach -t id` inside the existing PTY relay Keep a systemd unit only for the Codex subscription terminal, which needs RuntimeMaxSec and an invocation id. Collapse the status vocabulary to ready/ended. The frontend and Go host then drop opening and ending, or keep them only for the subscription path.

**Risk.** The Codex subscription path (subscription_*.rs, managed_end) reuses create_terminal, the unit naming and RuntimeMaxSec. That coupling would need to be split carefully. Killing a session no longer kills processes that escaped its PTY (the cgroup does today).

**Visibility.** The user still sees named terminals with ready/attached/ended states. Some transitional states ('opening', 'ending') would mostly disappear.

**Related.** The subscription terminal path reuses create_terminal, terminal_status, unit naming and the RuntimeMaxSec lifetime: subscription_prepare.rs, subscription_profile.rs, subscription_retire.rs, subscription_cgroup.rs and term_attach.rs. The opening/ready/ending/ended state vocabulary is validated in internal/host/terminal.go and rendered in frontend/spaces/sodaspaces-terminal-*.ts. Throughout the crate, `.py` parity comments (pyemit.rs, "PR25 brief" argument shapes) keep the code bound to the retired Python implementation.

### WEB-3: Project preparation is a five-request digest-echo ceremony with two decision ledgers, and it gates every dispatch

- **Subsystem:** Web API and view models (non-MCP, non-Feed, non-extension): internal/web/*.go, internal/web/auth/**, internal/web/api/*.go excluding factory_mcp*, factory_feed*, extension*. **Category:** process-machinery. **Verdict:** confirmed.
- **Removable:** ~800 lines. **Risk:** medium. **Visible change:** UI, model tools, DB schema. **Behaviour-preserving:** no.
- **Status:** owner decision.
- **Where:** `internal/web/api/environments_preparation.go:1-251`, `internal/web/api/preparation_decisions.go:13-222`, `internal/web/api/preparation_decision_preview.go:1-241`, `internal/web/api/preparation_source.go:1-303`, `internal/factory/effective_authority.go:62-64`, `internal/factory/control/preparation_decisions.go:28-60`, `internal/factory/control/preparation_start.go:24`, `internal/factory/control/intent_plan_assembly.go:147-194`, `internal/factory/control/dispatch_preparation.go:131-142`, `frontend/spaces/sodaspaces-factory-settings.ts:616-633,680-870`

**What.** Until preparation is ready (missing code preparation_not_ready), nothing can dispatch. To get there an owner must make five requests: (1) action=source-preview with hand-picked source_paths, setup_files and required_tools; (2) POST /preparation/acceptances, echoing source_commit, setup_digest, inputs_digest and a source_inputs path→digest map, plus a client-minted decision_id, predecessor and command_id; (3) action=approve-initial with free-text effects_evidence and readiness_evidence (up to 8 KiB each), which the server hashes into digests; (4) action=prepare, naming the requirement and approval IDs and two distinct, client-chosen preparation IDs; (5) action=approve, echoing effects_digest and readiness_digest. There are also hold and inspect actions, and a decision preview that re-reads native inputs under a 30 s budget.

**Simpler.** Replace the steps with one owner/admin 'Prepare environment' POST. The server reads setup.sh and check.sh at the target-branch head through the native source API, starts coder and reviewer preparation with server-minted IDs, and records one row (commit, setup digest, result). A GET shows state and logs. Drop the requirement and approval ledgers, source-preview, the approve-initial evidence text, the digest echo and the decision preview's native re-reads. Pick the planning context server-side, for example tracked files at head, bounded.

**Risk.** Preparation runs repository scripts on the host. Keeping 'current repository owner/admin only' preserves the real boundary. Planner context and retry re-binding currently come from the accepted source_inputs, so control and store must change too (other finders' scope).

**Visibility.** Six preparation forms become one button. The planner's staged source context would come from a server rule instead of the human's selection.

**Related.** Store tables for requirement and approval decisions (RequirementDecision, ApprovalDecision, RequirementDepth, ApprovalDepth), project.RequirementDecision/ApprovalDecision validation, control/preparation_start.go and its 580-line test, the planner context (intent_plan_assembly.go), retry rebinding (dispatch_preparation.go, Store.MergedPreparationSource), tests/frontend/factory-settings-controls.test.ts

### APP-3: Restore is a crash-recovery system: an on-disk journal with 4 phases, partial-output retention, a restore-operation table with 3 states plus a DB trigger, an abort action, and pending gating on actions and snapshots

- **Subsystem:** Application runtime: build, home, snapshots, backups, restore, factory delivery. **Category:** too-many-states-or-budgets. **Verdict:** confirmed.
- **Removable:** ~650 lines. **Risk:** low. **Visible change:** UI, DB schema. **Behaviour-preserving:** no.
- **Status:** owner decision.
- **Where:** `lib/host/src/application/restoration.rs:1-309`, `lib/host/src/application/wire.rs:118-195`, `lib/host/src/application/runtime.rs:388-390`, `internal/application/restore.go:1-39`, `internal/application/administration.go:35-74`, `internal/host/application.go:69`, `internal/web/api/application_restore.go:1-148`, `internal/store/application_restore.go:1-133`, `internal/store/schema.go:30,185,348`, `cmd/soda-identity/src/schema.rs:21,25,218,315`, `internal/web/api/application_actions.go:132`, `internal/web/api/application_snapshots.go:53-55`, `internal/web/api/applications.go:22,86-88`, `frontend/spaces/sodaspaces-applications.ts:388-512`, `frontend/spaces/sodaspaces-application-response.ts:47-49,200`, `tests/installed/qualification/native.ts:42`, `docs/public/50-Operate/50-backups-and-restoration.md:147-178`

**What.** Restore reserves an immutable RestoreOperation (Go and Rust each validate it), inserts an application_restores row (pending/complete/aborted), and nulls the observation. Rust keeps a root-only JSON journal under restores/ with phases prepared/copied/installed/aborted, fsyncs it after every step, and on retry reconciles four different resume cases. retain_partial renames half-copied output aside using renameat2 NOREPLACE. 'restore_abort' is a second action with its own rules. Every lifecycle action, backup and view first checks PendingApplicationRestore and refuses with 'Reconcile or abort it'. All of this exists so an interrupted restore can be resumed exactly. That is the 'speculative recovery system' AGENTS.md forbids, for an unreleased product with zero users and zero app data.

**Simpler.** Keep restore as one synchronous Rust 'restore' op. It requires the app to be stopped, copies the snapshot into a fresh data dir with snapshot::restore, removes the container and recreates it with the new data, then returns the observation. Go then updates data_id and revision in one UPDATE. Drop the journal, retain_partial, restore_abort, the application_restores table (both DDL copies), the pending gate and the pending UI, and loosen soda_guard_application so a data_id change is allowed together with a revision+1 bump. Fully deleting Restore is not enough: the existing CLI only makes unbound copies, and the trigger blocks rebinding.

**Risk.** There is no real data to protect yet. If restore is kept in its simple form, a crash mid-copy leaves an orphan directory that is never selected, which is acceptable.

**Visibility.** The restore and abort UI and the 'restore pending' refusals are removed or simplified.

**Related.** The same pending-operation-table-plus-trigger pattern appears in application_setups (pending/running) and project_preparations, both counted in the same qualification 'unsettled' checks (tests/installed/qualification/native.ts, phases/docker-native.ts). The schema DDL is duplicated between internal/store/schema.go and cmd/soda-identity/src/schema.rs.

### DSP-3: The planning-freshness JEV call gates dispatch, but its answer never reaches the worker

- **Subsystem:** Factory dispatch and run start. **Category:** other. **Verdict:** confirmed.
- **Removable:** ~600 lines. **Risk:** low. **Visible change:** UI, DB schema. **Behaviour-preserving:** no.
- **Status:** done (`9ac11119`), with PLN-1. Acceptance status already refuses superseded generated intents, so no replacement gate was added.
- **Where:** `internal/factory/control/planning.go:33-299`, `internal/factory/control/dispatch.go:266,351,436,455-461`, `internal/factory/control/dispatch_attempt.go:185-189`, `internal/factory/control/planning_admission.go:133-150`, `internal/factory/control/judgment_advice.go:49-53`, `internal/factory/planning.go:195-259`, `internal/factory/judgment.go (JudgmentPlanningFreshness)`, `internal/store/factory_planning.go:1-65`, `internal/store/schema.go:38`, `internal/store/factory_feed_plan.go:15-37`, `internal/store/factory_feed.go:110-116`, `internal/store/factory_intents.go:340-439`, `internal/web/api/factory_plan_view.go:84-95,346`, `internal/web/server.go:72`, `frontend/spaces/sodaspaces-factory-feed-communication.ts:74`, `frontend/spaces/sodaspaces-factory-feed-response.ts:352,1114-1126`, `frontend/spaces/sodaspaces-factory-progress-view.ts:680,706,984`, `frontend/spaces/sodaspaces-factory-progress-response.ts:241,351`, `internal/factory/dispatch_prompt.go:62-63`

**What.** Every Dispatch runs PlanningPass first. For each accepted issue without an assignment it re-reads acceptance evidence, native inputs, sponsorship, preparation and the gate, builds a PlanningInput snapshot, and asks JEV (planning-freshness) to pick direct_small_change / incremental_acceptance / prerequisite_first, or plan_stable_scope / wait_for_outputs. The call is charged to a per-intent planning allowance. Dispatch then refuses to launch any code issue ('plan_not_current') unless planningCurrent re-derives the same snapshot (another round of native reads) and the stored record says planned and not provisional.

**Simpler.** Delete PlanningPass, planIssue, planningInput (and its admit helpers), planningCallFailed, planningRecordCurrent and the PlanCurrent dispatch gate together with the plan_not_current and planning_deferred waits. Code issues then go straight to their Architect investigation run. Drop PlanningRecord, PlanningInput, PlanningSnapshot, ReadyToPlan, PlanningControlsAfter, the factory_planning table, the F11 strategy Feed card and its Feed-query clause, the plan-view Strategy field and the frontend planningSource/strategy rendering. Also drop ReserveIssuePlanning and the issue part of the intent planning-call allowance. Keep the PlanningJudgments struct with Chooser and Funding, which other JEV callers use; remove only its Journal field. If any gate is still wanted, keep the GeneratedIntentCurrent staleness check.

**Risk.** This removes a gate, not a producer: no worker input changes. Confirm the strategy card is not a demo checklist item, and keep the generated-intent currency check.

**Visibility.** The 'Implementation strategy recorded/unavailable' Feed card, the plan-view strategy and the plan_not_current/planning_deferred waits disappear. Issues stop silently stalling after a failed or unfunded planning call.

**Related.** The same pattern appears in other places that store a JEV failure. AdvisePurpose/replayPurposeJudgment turns a stored failed receipt into ErrPriorityUnavailable, which is unavailable forever for unchanged input, and every PurposeAdvice caller shares this (completion_*, repair_proposal, delivery_judgment_producer). The intent planning allowance (factory_intent_allowances / factory_intent_planning_calls) and the global factory_judgment_budget are the overlapping budgets shown in factory_access_view.go Planning allowances. The prod-hardening branch adds allowanceLowWait in dispatch code nearby, but leaves the planning gate alone.

### JDG-2: Each JEV call is 'funded' once and never retried automatically, and the global budget it charges is never read

- **Subsystem:** Completion, reassessment, review and JEV judgments. **Category:** too-many-states-or-budgets. **Verdict:** confirmed.
- **Removable:** ~600 lines. **Risk:** low. **Visible change:** UI, model tools, DB schema. **Behaviour-preserving:** yes.
- **Status:** done (`c62f6ec1`): failed JEV calls retry; the global counter is no longer touched.
- **Where:** `internal/factory/control/judgment_advice.go:13-116`, `internal/factory/control/judgment_advice_delivery.go:62-137`, `internal/web/api/factory_mcp_systemone.go:225-287`, `internal/store/factory_intents.go:266-325,410-418`, `internal/store/factory_attempt_allowances.go:1189-1251`, `internal/store/factory_reassessments.go:22-26,580-672`, `internal/store/factory_context_selection.go:25-56`, `internal/store/factory_delivery_repair_adoption.go:71-84`, `internal/store/schema.go:35-36`, `cmd/soda-identity/src/schema.rs:30-31`, `internal/factory/control/completion_decision.go:159-260,276-279`, `internal/factory/control/completion_drive.go:420-436`, `internal/factory/control/planning.go:186-259`, `internal/factory/reassessment.go:63-70,383-449`

**What.** Before each call, every JEV purpose reserves three things in one transaction: a fingerprint row, a purpose-specific ledger entry and a global counter. The purpose ledgers are: the planning-call allowance, the attempt's DeliveryCalls, the reassessment 2-call ledger, the per-capability context selection, the repair cycle and the comment revision. Any failure records a failed receipt, and that receipt is replayed as 'unavailable' forever for the same inputs (AdvisePurpose replayPurposeJudgment :49-54, AdviseDelivery :92-98, worker SystemOne :263-272).

**Simpler.** Drop the 'funding' around JEV and use one ask path. Call JEV, retry once quickly on a transport error, and on failure return the error and record nothing, so the next progress pass simply asks again. Successful answers can optionally be cached by fingerprint. Delete: - factory_judgment_budget, in both the Go schema and the Rust soda-identity schema, along with every spent update - ErrReassessmentBudgetExhausted, errReassessmentExhausted and the 'funding_exhausted' branches - the Reserve closure on PurposeAdvice and the fingerprint-row reservation logic - the DecisionCalls ledger and ReserveReassessmentCall - the DeliveryCalls subledger and ReserveDeliveryJudgment - errReassessmentDeferred, planningCallFailed and recoverReassessmentDecision - the tests that assert on spent or on the exhaustion error Keep the planning-call allowance only if the owner wants it as a product setting.

**Risk.** Removing the unread budget and the dead exhaustion branches is behaviour-identical. Retrying changes outcomes only in failure cases, which stop being permanent. The planning-call allowance (ReserveIntentJudgment) is a user setting owned by the intent subsystem. The context-bundle case is documented, but not fixed, in docs/development/production-gaps.md on prod-hardening.

**Visibility.** Waits that are permanent today become transient: delivery_judgment_unavailable, decision_unavailable/funding_exhausted and context_selection_unavailable. The SystemOne tool stops refusing with 'reservation retained without another provider call'. The factory_judgment_budget table goes.

**Safe first step.** Delete the unread global counter and the dead exhaustion path; none of this changes behaviour. - Remove the `SELECT spent FROM factory_judgment_budget ... FOR UPDATE` and `UPDATE factory_judgment_budget SET spent=...` statements from factory_intents.go (ReservePriorityJudgment and ReserveIntentJudgment), factory_attempt_allowances.go (ReserveDeliveryJudgment), factory_reassessments.go (ReserveReassessmentCall), factory_context_selection.go and factory_delivery_repair_adoption.go. - Drop the CREATE/INSERT lines for factory_judgment_budget from internal/store/schema.go:35-36 and cmd/soda-identity/src/schema.rs:30-31. - Delete ErrReassessmentBudgetExhausted (factory_reassessments.go:22-26) and errReassessmentExhausted (completion_decision.go:158-161), plus the branches that test for them: completion_decision.go:217-219 and 276-279, and completion_drive.go:425-428. - Delete or trim the test assertions on spent and on ErrReassessmentBudgetExhausted.

**Related.** The same reserve-before-call, failure-sticks pattern is in factory_context_selection.go (context-bundle-v1, documented as a gap in prod-hardening docs/development/production-gaps.md:62-68), factory_delivery_repair_adoption.go (delivery-repair-v1), ReserveIntentJudgment/ReserveIssuePlanning (intent planning and planning freshness) and the worker SystemOne MCP tools. The Rust soda-identity schema (cmd/soda-identity/src/schema.rs) mirrors the budget table. lib/host/src/project/create.rs:42 has a similar 'reservation retained' idiom.

### WEB-4: Six separate grant and budget records, each with its own settings route, plus a derived preparation check, must all be in place before one run; four concurrency caps overlap (two are clamped away), and max_queued is never enforced

- **Subsystem:** Web API and view models (non-MCP, non-Feed, non-extension): internal/web/*.go, internal/web/auth/**, internal/web/api/*.go excluding factory_mcp*, factory_feed*, extension*. **Category:** too-many-states-or-budgets. **Verdict:** confirmed.
- **Removable:** ~600 lines. **Risk:** high. **Visible change:** UI, DB schema. **Behaviour-preserving:** yes.
- **Status:** owner decision; coordinate with prod-hardening.
- **Where:** `internal/web/api/extension_native.go:217-230`, `internal/web/api/factory_policy.go:11-158`, `internal/web/api/factory_sponsorship.go:13-192`, `internal/web/api/factory_connection_usage_budget.go:12-74`, `internal/web/api/factory_repair.go:14-127`, `internal/web/api/factory_verification_allowance.go:30-81`, `internal/web/api/factory_intents.go:166-209`, `internal/web/api/factory_access_view.go:132-215`, `internal/web/api/factory_status.go:15-18,49-55,99-108`, `internal/factory/effective_authority.go:6-72`, `internal/factory/grants.go:246,315-323,372`, `internal/factory/control/dispatch_selection.go:79-107`, `frontend/spaces/sodaspaces-factory-settings.ts:74-80,876,899-929,1150-1158,1440-1600`

**What.** Factory authority is split into a repository policy, an operator grant, an appliance capacity, a sponsorship, a connection usage budget, an environment grant and preparation. Each has its own PUT, revision and settings form. Budgets are spread across even more routes: repair grant, verification allowance and planning allowance.

**Simpler.** Use one per-repository Factory record: enabled/paused, a coder-minutes allowance and one concurrency cap, with the appliance cap as a config default. The server fills the native actor bindings from config. It also picks the broker grant for the repository's project on the chosen connection and reads that grant's generation itself. An existing project counts as the environment grant, and the connection usage budget folds into the allowance. Delete the capacity, operator-grant, environment-grant and usage-budget PUTs, max_queued, and the policy/operator MaxConcurrent fields (dispatch hard-clamps them to 1 anyway), along with their required-action and usage mapping. Fold the repair, verification and planning allowances into the one allowance, or drop them along with their features.

**Risk.** This changes dispatch admission and the AuthorityRef bound into assignments, so it needs the control and store finders. The hardening branch is adding a coder-time allowance hold in dispatch_selection.go. Deleting max_queued and filling actor bindings and the broker grant server-side can be done first, independently.

**Visibility.** Settings shrinks from about 8 permission and budget forms to one, the sponsorship grant no longer has to be re-picked, and the always-zero 'active' count disappears or becomes real.

**Safe first step.** In internal/factory/control/dispatch_selection.go, replace the repoLimit computation (min of plan.policy.MaxConcurrent and grant.MaxConcurrent, then clamp to 1) with repoLimit := 1. Validation keeps both values >= 1, so this changes nothing observable. The OperatorGrant read stays for now because it confirms the grant still exists. This file is on the hardening branch, so coordinate with that session first. Every other part (removing max_queued, server-filled actor bindings, filling the broker grant server-side, removing the PUTs) changes the API or Settings UI.

**Related.** The Store layer (SaveCapacity, OperatorGrant, EnvironmentGrant, usage budget tables in internal/store) and AuthorityRef, which is bound into assignments in internal/factory/control. Test fixtures in tests/frontend/factory-settings-controls.test.ts and the internal/factory/control *_test.go files seed capacity with MaxQueued. The hardening branch adds frontend/spaces/sodaspaces-factory-allowance.ts and a coder-time hold in dispatch_selection.go, which adds yet another budget.

### DLV-3: The merge authorization 'generation ledger' and the legacy decision gate are unreachable; the one live binding is re-verified three times; withdrawn or refused merges can never be redone

- **Subsystem:** Publication, merge, correction, repair, acceptance, verification, lifecycle. **Category:** speculative-abstraction. **Verdict:** confirmed.
- **Removable:** ~550 lines. **Risk:** low. **Visible change:** none. **Behaviour-preserving:** yes.
- **Status:** first step done (`61fbce18`): test-only ledger, replacement generations and AdviseMergeJudgment deleted; History/snapshot types and the legacy gate branch remain.
- **Where:** `internal/factory/merge_operation.go:295-701`, `internal/factory/merge_operation.go:707-819`, `internal/factory/merge_operation.go:821-929`, `internal/factory/merge.go:95-135`, `internal/factory/merge.go:202-237`, `internal/factory/merge.go:318-338`, `internal/factory/control/merge_authorization.go:29-56`, `internal/factory/control/merge_authorization.go:112-154`, `internal/factory/control/merge_authorization.go:178-234`, `internal/factory/control/delivery_producer.go:79-142`, `internal/factory/control/merge_evidence.go:175-191`, `internal/store/factory_merges.go:97-108`, `internal/store/factory_merges.go:193-421`

**What.** Each merge row carries MergeAuthorization: a MergeGeneration with 17 fields, a DeliveryDecision and a ResolutionsConfirmed flag. It also carries an append-only History of MergeGenerationSnapshot, a MergeLedger view with register/supersede/adopt transitions and a 32-generation ceiling, MergePredecessorProof replacement rules, and a separate legacy 'funded decision only' gate for rows without an AcceptanceID. The store enforces all of this with reflect.DeepEqual history checks and replacement rules.

**Simpler.** Keep a single authorization on the merge row: acceptance id, head, base and check assessment revision. Set it once in authorizeCleanDeliveryMerge and re-check it once before submit. Delete History, the snapshots, predecessor proofs, MergeLedger, the union-versus-legacy split, ResolutionsConfirmed, NativeOperationID and the store replacement rules. If a withdrawn merge or a no-effect refusal should be able to merge again, reset the same row to open with the next op id instead of keeping a ledger.

**Risk.** The deletions only remove unreachable or test-only code. Reopening a withdrawn merge is optional and is the only behaviour change.

**Visibility.** None for the deletions. With the reopen rule, a merge withdrawn by Pause would complete after Resume.

**Safe first step.** Delete the code that only tests reach, along with its tests: - In internal/factory/merge_operation.go: MergeLedger, NewMergeLedger, Validate, MergeLedgerUpdate, Apply, register, supersede and adopt (around lines 536-701), SupersededByBase (around 503-510) and MergeUnionReadyInput (merge_authorization.go:219-234). - In internal/factory/control/merge_authorization.go: authorizeReplacementGeneration (112-154). Have AuthorizeMerge refuse anything other than generation 1 on an unbound open row. - In internal/store/factory_merges.go: mergeReplacementAllowed and its helpers (324-421), mergeHistoryPrefixStable (239-250) and the history-length branch in mergeUpdateAllowed. Treat any change in History length as ErrMergeConflict. - Drop Merge.Ledger() from Merge.Validate (merge.go:227, 318-338). - Delete merge_generations_test.go, factory_merges_generations_test.go and the ledger, AdviseMergeJudgment and legacy-gate cases in merge_judgment_gate_test.go. Production only ever writes generation 1 with an AcceptanceID, so observable behaviour stays identical. In a second step, remove the History, MergeGenerationSnapshot and MergePredecessorProof types and the non-union CheckMergeDecisionGate branches.

**Related.** internal/factory/control/merge_judgment_gate.go (AdviseMergeJudgment, MergeJudgmentGateInput) is only reached from tests and goes with the legacy gate. The explicit retry path (internal/store/factory_explicit_retry.go retryMergeSettled) is today's only way out of a terminal withdrawn or failed merge. Stale doc comments in merge_authorization.go still claim there is no production binder.

### IDN-4: Broker carries a mechanical mirror of the whole Soda schema (85 tables, version gate, 68->69 Applications migration)

- **Subsystem:** Identity service and provider credentials. **Category:** duplicated-layer. **Verdict:** confirmed.
- **Removable:** ~530 lines. **Risk:** low. **Visible change:** none. **Behaviour-preserving:** yes.
- **Status:** open (behaviour-preserving).
- **Where:** `cmd/soda-identity/src/schema.rs:1-387`, `cmd/soda-identity/src/store_schema.rs:10-16`, `cmd/soda-identity/src/store_schema.rs:78-215`, `cmd/soda-identity/src/store.rs:53-58`, `internal/store/schema.go:309-312`

**What.** schema.rs is 'Mechanically mirrored from internal/store/schema.go'. On startup the broker creates all 85 tables when the database is empty, refuses unless the version marker is exactly 69, upgrades 68->69 by creating application_setup_renewals (an Applications table), re-verifies 60 'SELECT ... LIMIT 0' column lists for factory, feed, application and project tables plus 6 triggers, and serialises opens with a process-wide SCHEMA_OPEN mutex and a lost-race reread. The broker uses six identity tables.

**Simpler.** Let Go own the schema. The broker checks that its six tables exist (an information_schema query) and refuses with a clear message otherwise; order soda-identity.service after the dashboard so first boot works. A broker-only CREATE TABLE IF NOT EXISTS is also possible, but it would need the trigger functions as well. Delete STATEMENTS, VERIFY_QUERIES, VERIFY_TRIGGERS, UPGRADE_68_TO_69, load_schema_version, migrate_schema_68_to_69, reopen_after_lost_upgrade_race and SCHEMA_OPEN.

**Risk.** First-boot ordering only: today either process can bootstrap an empty database; afterwards the dashboard (or the broker's own IF NOT EXISTS) creates the identity tables. No data or schema change.

**Visibility.** None; removes recurring mirror-sync commits on every Go schema change.

**Safe first step.** In cmd/soda-identity/src/store_schema.rs, delete the static SCHEMA_OPEN mutex (lines 10-15) and its lock in initialize_schema. main.rs opens exactly one Store per process and no test references it, so nothing observable changes. Then the larger step: replace open_current_schema and verify_current_schema with a check that the six identity tables exist, and delete schema.rs, UPGRADE_68_TO_69, migrate_schema_68_to_69, load_schema_version and reopen_after_lost_upgrade_race. This step changes first-boot ordering, so add an After= on the dashboard in system/host/services/soda-identity.service.

**Related.** lib/soda-release-image/src/model.rs:44 has another stale schema-version mirror (SCHEMA_VERSION = 68 while Go is at 69), used in payload_stage.rs:372. The Go 68->69 upgrade (internal/store/schema.go, schemaUpgrade68To69) is the same kind of migration for retained databases that do not exist with zero users.

### EXT-1: Extension backend re-gates every request: second route table, second authority callback, 64 KiB response cap, no-op contribution authorizer

- **Subsystem:** Fountain extension and its backend. **Category:** duplicated-layer. **Verdict:** confirmed.
- **Removable:** ~520 lines. **Risk:** low. **Visible change:** UI. **Behaviour-preserving:** yes.
- **Status:** owner decision.
- **Where:** `internal/web/api/extension.go:22-538`, `internal/web/api/application_routes.go:1-22`, `cmd/soda-extension/main.go:35-71`, `system/containers/extension/extension.json:44`, `lib/release-inputs/src/payload.rs:35-45`, `lib/release-inputs/fixtures/payload-v4.json`, `lib/host/tests/data/iconfig/release.json`, `lib/soda-release-deliver/tests/goldens/deliver.json`, `internal/web/api/extension_test.go:23-408`

**What.** The soda-extension process should only carry Fountain requests to /ipc/soda.sock. Instead it re-implements the service gate: a ~400-line hand-written path/method/query allowlist (extensionRoute plus ~25 helpers) mirroring the service mux, a full auth.ExtensionAuthority (CurrentActor callback to Fountain) plus contribution checks, a buffered response cap, and its own request/response header allowlists. On top, AuthorizeContribution (capability native.contribution.authorize) makes Fountain RPC the backend before every page render and API call, and it always answers Allowed.

**Simpler.** Make api.Extension a forwarder of about 20 lines: an httputil.ReverseProxy over the unix dialer, whose Rewrite sets a fixed scheme and host, sets Path to '/api'+In.Path and copies the In headers. ReverseProxy already handles WebSocket upgrades. Delete extensionRoute and all its helpers, application_routes.go, boundExtensionResponse and its error types, the header allowlists and the long header timeout. The service mux, which already does the clean-path check, the body cap and ExtensionAuthority, becomes the only route table and the only gate. Delete contributionAuthorizer and allowedContribution, and drop native.contribution.authorize from extension.json, SODA_CAPABILITIES (array length 9 becomes 8), payload-v4.json, iconfig/release.json and the deliver.json golden. Delete the proxy allowlist and bounds tests along with the code.

**Risk.** Not a security boundary (world-connectable socket, service re-verifies). The capability must be removed from manifest, payload constants and goldens together because Fountain checks registration against declared capabilities at start (../forgejo-ext/services/extensions/manager.go:648). Unknown paths return the service's 404 instead of the proxy's.

**Visibility.** Identical responses for every route the proxy admits today. Visible only as fixes: App delivery (application-builds, ?delivery_id=) stops returning 404 and responses over 64 KiB stop failing; proxy-level error text becomes Fountain's. cmd/soda-extension/main.go is also edited on the ui/repo-places* branches.

**Safe first step.** Behaviour-identical removals first: (a) In cmd/soda-extension/main.go, delete contributionAuthorizer and allowedContribution and the AuthorizeContribution assignment. In the same change, remove "native.contribution.authorize" from system/containers/extension/extension.json, lib/release-inputs/src/payload.rs SODA_CAPABILITIES (resize to [&str; 8]), lib/release-inputs/fixtures/payload-v4.json, lib/host/tests/data/iconfig/release.json and lib/soda-release-deliver/tests/goldens/deliver.json. The authorizer always answers Allowed, and without the capability Fountain skips the RPC. (b) In internal/web/api/extension.go, remove "Sec-Fetch-Mode", "Sec-Fetch-Dest" and "X-CSRF-Token" from the forwarded-header list, and remove the Set-Cookie and context/admission header deletions in ModifyResponse. Fountain never sends the first three and copies back only Content-Type. (c) In the proxy handler in extension.go, drop the redundant second auth.ExtensionContribution call, which ExtensionAuthority already performs. Then replace extensionRoute and application_routes.go with the plain forwarder. This step changes only 404 text for unknown paths and fixes the application-builds routes.

**Related.** The service-side ExtensionHandler (internal/web/api/extension_terminal.go:19-43) already has its own clean-path check and body cap, and extensionProductContribution re-checks paths per contribution, so the gates are layered three deep. The tailnet package (cmd/soda-extension/main.go:42) uses the same api.Extension proxy and benefits from the same simplification.

### RBA-2: Muse reaches projects three ways, and the build pin must equal live upstream anyway

- **Subsystem:** Release build primitives, asset fetch/render tools, small install binaries and system units. **Category:** duplicated-layer. **Verdict:** confirmed.
- **Removable:** ~500 lines. **Risk:** medium. **Visible change:** none. **Behaviour-preserving:** yes.
- **Status:** open (behaviour-preserving).
- **Where:** `tools/release-assets/src/fetch/muse.rs:357-565`, `tools/release-assets/src/bin/soda-fetch-muse.rs:1-165`, `system/project/muse-release.json`, `lib/soda-release-build/src/production_assets.rs:120-135`, `system/project/Containerfile:32-34`, `tools/release-assets/src/render/stage/mod.rs:65-69`, `system/host/services/soda-provider-update.service`, `system/host/services/soda-provider-update.timer`, `system/host/services/soda-project@.service:19-20`, `cmd/soda-project-maintain/src/main.rs:96-118`, `cmd/soda-project-maintain/src/provider_stage.rs:1-60`, `cmd/soda-muse/src/store_resolve.rs:1-30`

**What.** 1) The build pins Muse by version, sha256 and size in system/project/muse-release.json. `admit_current` then resolves the live official channel on every build and refuses unless pin == live ('stale baseline; run soda-fetch-muse --refresh'). The pin therefore adds nothing except a forced hand commit each time upstream ships. Supporting it are a check mode, a refresh mode with an atomic rewrite of the JSON, a Baseline/FetchRefusal state set and a hand-written RawObject serde visitor with last-key-wins semantics. 2) The pinned bytes are baked into the project image (/usr/local/libexec/soda/muse) and into the host image (/usr/share/soda/muse-tools/muse-native). 3) Separately, soda-provider-update refreshes Muse from the same channel on the host every 5 minutes. soda-project-maintain --provider-store stages that selection into projects with a muse-selection.json receipt and says it 'never falls back to a pinned descriptor'. soda-project-maintain keeps both modes (load_tools vs resolve_native_staging).

**Simpler.** Pick one delivery path. Either keep the runtime provider store, and drop the baked muse-native, soda-fetch-muse, muse-release.json, the pre-updater fallback in soda-muse store_resolve and soda-project-maintain's load_tools mode. Or keep the build-time fetch, have it download the live release and verify it against the live manifest checksum, with no committed pin and no check/refresh modes. Either way, delete fetch_muse now.

**Risk.** Which path projects actually run Muse from needs a decision: the receipt enforcement in soda-muse and lib/host/src/muse depends on the provider-store path. Removing the pin alone is low risk.

**Visibility.** Operator and build behaviour only. Builds stop failing on 'stale baseline'.

**Safe first step.** Delete the uncalled `pub fn fetch_muse` and its doc comment in tools/release-assets/src/fetch/muse.rs (lines ~550-565). Its only body is a call to admit_current, and nothing references it.

**Related.** The same hand-written last-key-wins RawObject serde visitor is repeated in tools/release-assets/src/fetch/terminal.rs:32-70 and cmd/soda-project-terminal/src/term_binding.rs:36-60. The live-vs-pin check is exercised by tools/release-assets/checks/fetch-muse-baseline.sh. Docs still describe the pinned/manual path: docs/reference/project-os.md:204-213, docs/reference/credentials.md:234, docs/public/50-Operate/60-project-maintenance.md:40, cmd/soda-project-maintain/README.md.

### STO-2: The Rust identity broker carries a full second copy of the Go schema and its bootstrap, migration and verification logic

- **Subsystem:** Store (PostgreSQL layer) structure. **Category:** duplicated-layer. **Verdict:** confirmed.
- **Removable:** ~500 lines. **Risk:** low. **Visible change:** none. **Behaviour-preserving:** yes.
- **Status:** open (behaviour-preserving).
- **Where:** `cmd/soda-identity/src/schema.rs:1-387`, `cmd/soda-identity/src/store_schema.rs:9-15`, `cmd/soda-identity/src/store_schema.rs:78-133`, `cmd/soda-identity/src/store_schema.rs:143-215`, `cmd/soda-identity/src/store.rs:54-57`, `cmd/soda-identity/tests/common/mod.rs:103-119`, `internal/store/schema.go:18-278`, `system/host/services/soda-identity.service`, `docs/development/go.md:215`

**What.** The identity broker creates the entire appliance schema: 147 statements covering all versioned tables, the 33 triggers and the guard functions. It also re-implements the 68->69 upgrade, the SCHEMA_OPEN mutex, the lost-race re-read, the 58 column probes and the trigger probes. All of this exists so that whichever service opens first can create the database.

**Simpler.** Go owns all DDL. Preferred: the broker creates nothing and checks that schema_version equals 69 and that its identity tables exist. Otherwise it refuses, and systemd's Restart=on-failure retries every 3s until the dashboard has created the database (in the real flow soda-setup enrollment runs after the dashboard, so this does not happen). Alternative: the broker runs CREATE TABLE IF NOT EXISTS for its own identity tables and grant_key_check only (needs STO-1 so Go accepts that database). Either way, delete schema.rs and about 140 lines of store_schema.rs: the mutex, the 68->69 upgrade, the lost-race reread and the full column and trigger probes. Keep check_grant_key and initialize_grant_key. Change the Rust test fixture to have Go create the schema first, or rely on the own-tables alternative.

**Risk.** Identity table DDL does not change. Only the first-start behaviour of the socket-activated broker changes: it now needs the dashboard to have opened the database once.

**Visibility.** Invisible to users and models. Removes a hand-maintained 387-line mirror that every Go schema change must track.

**Safe first step.** Delete VERIFY_QUERIES and VERIFY_TRIGGERS from cmd/soda-identity/src/schema.rs, and verify_required_columns, verify_trigger and the calls to them in cmd/soda-identity/src/store_schema.rs. In the deployed flow Go has already created and verified the same schema, so the broker opens identically. Then delete STATEMENTS, UPGRADE_68_TO_69, SCHEMA_OPEN and reopen_after_lost_upgrade_race, and make open_current_schema require version==69 (refuse otherwise). Update cmd/soda-identity/tests/common/mod.rs so the test database gets its schema from Go's DDL, or switch to the broker creating only its own identity tables.

**Related.** Go's own bootstrap and migration logic in internal/store/schema.go (STO-1/STO-3) is what this copy mirrors. The broker_compat_test.go fixture already opens the database with Go store.OpenEphemeral before starting the broker.

### VRF-1: Verification-manifest proposal/catalog/adoption chain re-encodes "run approved check.sh for ResolvedScope"; its presence only toggles native execution vs a per-revision check.sh declaration, and the planning-time check digest pin blocks delivery after check.sh re-approval

- **Subsystem:** Test Pilot / app verification (Go factory+control+store+MCP, Rust host, in-project helper). **Category:** duplicated-layer. **Verdict:** partially-confirmed.
- **Removable:** ~500 lines. **Risk:** medium. **Visible change:** UI, model tools. **Behaviour-preserving:** yes.
- **Status:** owner decision; coordinate with prod-hardening.
- **Where:** `internal/factory/mcp_manifest.go:1-316`, `internal/factory/control/verification_catalog.go:1-79`, `internal/web/api/factory_mcp_verification_catalog.go:1-47`, `internal/web/api/factory_mcp.go:536-575`, `internal/factory/control/intent_plan_choice.go:129-139`, `internal/factory/control/intent_plan_choice.go:418-452`, `internal/factory/control/completion_adoption.go:31-65`, `internal/factory/control/verification_execution.go:17-38`, `internal/factory/control/verification_execution.go:125-163`, `internal/factory/control/delivery_admission.go:55-66`, `internal/factory/verification_approved_check.go:14-82`, `internal/factory/context_verification.go:117-147`, `internal/store/factory_verification_manifest.go:13-47`, `internal/store/factory_intent_generation.go:115-135`, `internal/factory/intent_generation.go:44-47`, `internal/factory/control/mcp_fw40.go:300`, `internal/web/api/factory_mcp_role_schema.go:51`, `internal/factory/control/intent_bound.go:126`

**What.** The same fact, 'run the approved check.sh for every accepted outcome', is carried by five types. (1) The Product Lead's VerificationManifestProposal maps outcomes to catalog IDs, with params, artifacts, fixtures and fixture_grants. (2) ApprovedProjectCheckCatalog snapshots the planning preparation and the check.sh digest. (3) AdoptedVerificationManifest (proposal plus catalog plus intent binding) is stored on IntentGeneration and copied forward to each new revision in completion_adoption. (4) ApprovedAppCheck {intent, revision, outcomes} is digested into DeliverySubject.VerificationManifestDigest. (5) project.VerificationManifest is the native run spec. Behind this sits a generic ApprovedCatalog interface with RequiredParams, AllowedParams and ParamValueChars.

**Simpler.** Use one delivery path. At delivery admission, read the approved check.sh from the ready reviewer preparation at the integrated tip. Always stage the native project.VerificationManifest for ResolvedScope(revision), using the CheckDigest read at that moment. Set DeliverySubject.VerificationManifestDigest from that manifest, or drop it. Delete VerificationManifestProposal, ApprovedProjectCheckCatalog, AdoptedVerificationManifest, the ApprovedCatalog interface, the catalog endpoint, checkManifestProposal, the IntentGeneration fields and their store checks, the plan-choice binding and settle checks, and inheritGraphVerification. Also drop the "# soda-app-verification:" declaration path, which needs per-revision IDs written into check.sh. Remove verification_manifest_proposal from the finish_product_plan schema and from the guidance texts.

**Risk.** The generation JSON fields become unused, but they live in jsonb, so no migration is needed. The hardening branch's product-plan settle code parses verification_manifest_proposal and would need the matching deletion. The only behaviour lost is the planning-time digest pin, which is the bug described above.

**Visibility.** The Product Lead finish tool loses the optional verification_manifest_proposal field and the long ApplicationBundleGuidance text about it. The Feed and settings are unchanged.

**Safe first step.** In internal/factory/mcp_manifest.go, delete the ApprovedCatalog interface and make ValidateManifestProposal and validateCatalogRef take the concrete factory.ApprovedProjectCheckCatalog. Change the Catalog dep type in internal/web/api/factory_mcp.go:83 and the return type in internal/web/api/factory_mcp_verification_catalog.go to match. Remove the always-empty RequiredParams, AllowedParams and ParamValueChars fields from ApprovedCheck, along with the bound, allowed-name and required-param loops in validateCatalogRef. Replace them with "if len(params) != 0 { return errors.New(\"invalid check parameter name\") }" so refusal outcomes and text stay the same. MCPMaxCatalogParams and MCPMaxCatalogParamChars then become unused and can go too.

**Related.** Overlaps MCP-11 on Product Lead finish-schema fields. The hardening branch parses verification_manifest_proposal in mcp_fw40.go, factory_mcp.go and factory_mcp_product_plan.go and would need the matching deletion. docs/development/factory-workflow-contracts.md around line 2197 describes the proposal contract. InvestigationPlanGuidance in dispatch_prompt.go:482 and the argv description in factory_mcp_plan_schema.go:61 mention verification_manifest_proposal and verification_catalog.

### BUD-1: A run's deadline is whatever is left of 4-6 overlapping budgets. It is fixed at dispatch and the pre-launch JEV hold is charged against it (the 4-minute stop-before-start)

- **Subsystem:** Budgets, allowances and limits (cross-cutting). **Category:** too-many-states-or-budgets. **Verdict:** partially-confirmed.
- **Removable:** ~400 lines. **Risk:** medium. **Visible change:** UI, model tools. **Behaviour-preserving:** no.
- **Status:** done (`b1d4c63e`): one per-run limit; leftover budgets never shorten a run.
- **Where:** `internal/factory/control/dispatch_selection.go:48-175`, `internal/factory/control/dispatch_launch.go:243-260`, `internal/factory/control/dispatch_launch.go:690-761`, `internal/factory/control/dispatch_timeonly.go:336-349`, `internal/factory/control/dispatch_context_recovery.go:39-61`, `internal/factory/control/progress.go:118-131`, `lib/host/src/factory/stop.rs:94-117`, `internal/factory/control/settle.go:240-320`, `internal/factory/control/dispatch_result.go:175-184`, `internal/factory/control/dispatch_recovery.go:263-279`, `internal/factory/control/lifecycle_retry.go:37-53`, `internal/store/factory_implementation_retry.go:1-309`, `internal/factory/grants.go:59,94`, `internal/factory/dispatch_prompt.go:358`

**What.** Each run's absolute deadline is computed once, when the dispatch packet is recorded. It is the minimum of what is left in: the sponsorship's lifetime minutes (minus held plans), the connection's 24h rolling minutes, the attempt's wall-clock seconds, the 179-minute host bound and, for Issue-zero roots, the subject's funded minutes. Dispatch refuses only when less than one minute is left, so a nearly spent budget produces a 1-5 minute run. Both the clock and the usage charge start at record time (run.Started). That is before the capability is staged and before the context-bundle JEV choice, and a failed choice is held without retry until the deadline. The evidence lists every distinct budget.

**Simpler.** (a) One constant per-run wall-clock limit (e.g. 120 minutes, within the 179-minute host bound). Set Started and Deadline when launchTail clears ContextPending, just before FactoryLaunch, and never derive the deadline from what budgets have left. (b) One spend cap, checked only at admission as a soft cap. (c) Charge usage only for runs that reached the host. Once those hold, recoverUnused becomes the single expiry path for never-launched runs: stop settling ContextPending runs in the progress pass. Then delete the Cancelled 'stop-before-start' state, CommandRetryImplementation, its branch in lifecycle_retry.go and dispatch_launch.go, and factory_implementation_retry.go. The hardening branch's holdShortAllowance and MinUsefulRunMinutes become unnecessary. Reword the prompt line at dispatch_prompt.go:358.

**Risk.** Changes how deadlines and charges work. The record-before-call design and confirmUnused's fencing of lost launch replies must stay. A soft cap lets spend overrun by one run.

**Visibility.** Runs no longer end as 'stop-before-start', and waits for a nearly spent budget read differently. factory.context deadline_remaining_seconds keeps its meaning (only the value grows). The prompt sentence about deadlines 'within the remaining attempt and provider allowances' (dispatch_prompt.go:358) would be reworded.

**Related.** The prod-hardening branch adds holdShortAllowance, MinUsefulRunMinutes and an allowance-low notice in dispatch_selection.go plus the frontend; it treats the symptom, not the cause. The sponsorship, rolling, attempt, intent, verification and repair budget inventory overlaps the sibling findings BUD-3..BUD-11.

### EXT-4: Extension packaging needs hand-pinned artifacts for every build, plus a four-way Fountain pin chain

- **Subsystem:** Fountain extension and its backend. **Category:** process-machinery. **Verdict:** confirmed.
- **Removable:** ~400 lines. **Risk:** medium. **Visible change:** none. **Behaviour-preserving:** yes.
- **Status:** partly done (`f6e409c5`, the safe first step): extension.rs, files.json and the always-true version recheck are gone; assets are hard-linked directly. Open: a local package+catalog command, consuming the Fountain SDK as a normal Go module (drops the sibling replace, verify_fountain_sdk and check-fountain-pin), and the Fountain-side directory option; these need a cross-repo Fountain change.
- **Where:** `lib/soda-release-image/src/native_package.rs:120-178,359-388,420-452,502-555`, `lib/soda-release-image/src/extension.rs:1-64`, `scripts/build-soda-extension.ts:41-74,275-312`, `fountain-source.json:1-4`, `go.mod:6,61`, `lib/soda-release-tools/src/fountain_pin.rs:1-225`, `lib/soda-release-tools/src/build_cli.rs:337-352`, `lib/soda-release-tools/src/worker/runtime.rs:569-597`, `lib/soda-release-tools/src/release_start.rs:186-210`, `lib/soda-release-image/src/forgejo.rs:9-43`, `frontend/spaces/fountain-browser-contract.d.ts:1-6`, `lib/release-inputs/src/payload.rs:35-47`, `lib/soda-release-tools/tests/cli/soda_artifacts.rs`, `system/host/services/forgejo.container:33-38`, `docs/development/frontend-vm-update.md:1-263`, `docs/development/release.md:2474-2516`

**What.** Fountain installs the required soda package only from a trusted catalog entry (url, size, sha256, version, protocol, capabilities) and bootstraps from receipts, a verified cache or an HTTPS download. Soda's only producer of such packages is the full release pipeline (version stamped from the release tag, deterministic USTAR seal, GitHub release URL, catalog hash bound into Payload v4). With no local path, every extension change on a VM is assembled and pinned by hand. On the source side the SDK comes from an unversioned sibling checkout while three other places pin Fountain by hand.

**Simpler.** (1) Add one local command that builds the backend and assets, calls the existing stage_soda_package/seal_package, and writes soda.tar.gz plus a catalog.json with the computed sha256 and size. Add one install step that copies both to the VM and restarts Forgejo. This removes the hand pins and the derived image for front-end edits. Later, on the Fountain side, let an image-bundled required package load from a directory without a catalog hash. (2) Consume the SDK as a normal Go module (vendored, or a real module path with a pseudo-version) so go.sum is the pin. Then drop the sibling replace, verify_fountain_sdk, check-fountain-pin (fountain_pin.rs and its CLI tests) and the fork extraction in forgejo.rs. Derive the release URL from the one pin, and import the browser contract types from the SDK instead of keeping a hashed mirror. (3) Delete files.json and extension.rs and copy the fresh assets directory directly. Read capabilities from extension.json only. Drop the always-true version re-check and the extra manifest validators. Replace or delete the stale frontend-vm-update.md.

**Risk.** Release pipeline and a cross-repo Fountain change for the directory option; the GPL source obligation for the shipped Fountain image must still be met (Fountain releases already publish source archives in release.json). Parts overlap whoever audits lib/soda-release-* and soda-artifacts. The hardening branch's docs/development/production-gaps.md 'Release and versioning' item proposes refusing unstamped 0.1.0 builds, i.e. more checks in the opposite direction.

**Visibility.** Installed bytes and runtime behaviour unchanged; only build/install procedure and release tooling change. The hardening branch touches only a doc that discusses extension versioning, not these files.

**Safe first step.** In lib/soda-release-image/src/native_package.rs, delete the version-equals-release-tag check at the top of finish_staged_package (lines ~365-370). It is always true because read_package_manifest stamped the version from the same tag. Then delete lib/soda-release-image/src/extension.rs and the files.json write in scripts/build-soda-extension.ts:310, plus its assertion in build-soda-extension.test.ts:56. In stage_package, replace the stage_extension_assets call (native_package.rs:485) with a plain recursive copy of <native>/soda-extension-assets/assets into the staged assets directory. collect_package_files already walks and sorts that directory, so the sealed archive bytes are identical.

**Related.** soda-artifacts CLI (lib/soda-release-tools/tests/cli/soda_artifacts.rs check-fountain-pin tests); worker config FountainReleaseURL in lib/soda-release-tools/src/worker/config.rs:317-342; tests/build/forgejo_payload_test.go manifest checks; docs/development/release.md Fountain tag prose; overlaps any audit of lib/soda-release-* release machinery.

### TRM-2: Opening one terminal takes reserve + create + attach + inspect: four host round trips, about 12 podman spawns and about 20 Forgejo API calls

- **Subsystem:** Human terminal stack end to end (browser, Go API, host, in-project helper). **Category:** too-many-states-or-budgets. **Verdict:** confirmed.
- **Removable:** ~400 lines. **Risk:** medium. **Visible change:** UI. **Behaviour-preserving:** yes.
- **Status:** owner decision.
- **Where:** `frontend/spaces/sodaspaces-terminal-attachment.ts:132-176,213-228,339-376`, `internal/web/api/extension_terminal.go:50-75,99-129`, `internal/web/api/extension_terminal_stream.go:14-111`, `internal/web/api/extension_terminal_authority.go:55-123`, `internal/host/terminal_client.go:59-98`, `lib/host/src/dbackend/websocket.rs:71-130`, `cmd/soda-project-terminal/src/term_create.rs:155-287`, `cmd/soda-project-terminal/src/term_binding.rs:191-270`, `cmd/soda-project-terminal/src/term_status.rs:22-47,150-200`, `cmd/soda-project-terminal/src/term_collect.rs:100-176`

**What.** A new terminal starts with a POST /terminal-sessions reserve: a host op, then a guest 'reserve' that writes binding, reservation (expires in 120s, scope), name, writer and tmux.conf. The browser then opens a WebSocket with handshake 'create'. Go makes a separate host TerminalStates('create') call, and the guest consumes the reservation and runs systemd-run. Go then makes a third host op, OpenTerminal('attach'), and the guest spawns the PTY relay. After 'ready' the browser makes a fourth call, an inspect refresh. Each host op runs podman inspect, project admission, support_in_container (a podman exec sha256sum plus a host hash of the agent binary), managed_end (for end), and NativeAttach (hashes the agent binary again, then podman exec). Go wraps each op in extensionTerminalAccount and/or extensionTerminalCurrent before and after, and each of those costs CurrentActor + Repository Forgejo calls plus two DB reads. The reservation also needs an extra terminal state: 'opening' with an unconsumed permit, and the classify_bare_state error 'unconsumed reservation has a native unit'.

**Simpler.** Use one WS handshake: {action:'open', id?, name, cols, rows}. Go authorizes once and picks the id, which it returns in the 'ready' frame. The host runs one guest exec that creates the terminal if the id is new and then attaches in the same process. Delete the reserve endpoint, the guest reservation file with its validators and GC, permit_live, the Opening-by-permit state, the separate TerminalStates('create') call and the post-ready inspect refresh.

**Risk.** The create-then-attach handoff inside one guest process has to wait for the systemd/tmux readiness that the separate calls get today. Lost-reply duplicates become possible but harmless.

**Visibility.** The POST /terminal-sessions endpoint and its {id} response, which the frontend calls, go away. The terminal UI and messages can stay the same.

**Safe first step.** In internal/web/api/extension_terminal_stream.go, delete the leading extensionTerminalCurrent check in extensionOpenHostTerminal. extensionClaimTerminal ran the same check immediately before it. For the attach path this removes one full check (Forgejo authority, Repository call and 2 DB reads) per open with no observable change.

**Related.** extensionTerminalOperation (extension_terminal.go:50-75) checks current both before and after every host op, right after extensionTerminalAccount has already authorized the request. The same before-and-after pattern repeats for list, end and rename. The 15s heartbeat (extension_terminal_stream.go:136) also runs a full current check every tick.

### WEB-1: Hand-copied proxy route allowlist has drifted from the private mux: Application build/setup and lifecycle-effect history are blocked in the browser

- **Subsystem:** Web API and view models (non-MCP, non-Feed, non-extension): internal/web/*.go, internal/web/auth/**, internal/web/api/*.go excluding factory_mcp*, factory_feed*, extension*. **Category:** duplicated-layer. **Verdict:** partially-confirmed.
- **Removable:** ~370 lines. **Risk:** medium. **Visible change:** UI. **Behaviour-preserving:** no.
- **Status:** owner decision.
- **Where:** `internal/web/api/extension.go:63-67,113-497 (proxy gate and route predicates)`, `internal/web/api/application_routes.go:1-22`, `internal/web/api/extension_native.go:176-287 (private mux table)`, `internal/web/api/extension_terminal.go:19-42 (private handler; also serves manager-only /api/extensions/lifecycle)`, `internal/web/auth/extension_service.go:137-161`, `internal/web/api/extension_test.go:176-311`, `frontend/spaces/sodaspaces-application-delivery.ts:191,320,385`, `frontend/spaces/sodaspaces-factory-work.ts:767-768,1015-1030`, `frontend/spaces/soda-feed-entry.ts:57-58`

**What.** The soda-extension proxy declares every product route again (path shape, method, and a separate list of paths that may carry a query string) before forwarding to the private listener. The listener registers the same routes on its ServeMux, the auth fallback has its own method table, and the proxy test has a fourth copy. The copies have drifted apart.

**Simpler.** Make the private ServeMux the only product route table. The proxy keeps extensionCleanRoute's path checks (without the query allowlist), the ExtensionAuthority and contribution check, the body and response bounds (including extensionArtifactRoute for the artifact limit), and the two stream predicates for upgrade headers. It adds an explicit refusal of /extensions/lifecycle and forwards everything else. Delete extensionRoute's product, session, me and terminal predicates, extensionQueryRoute, application_routes.go and the allowlist test table.

**Risk.** The private listener rejects unknown routes and methods on its own and re-checks native authority. The only behaviour change is that registered routes that are blocked today start working. extension.go belongs to the extension finder's scope, so coordinate with them.

**Visibility.** Two broken UI paths start working: Applications build and setup, and the lifecycle-effect history on Work. Today both fail with a non-JSON 404. Nothing changes for routes that already work.

**Related.** The auth fallback in internal/web/auth/extension_service.go has its own method table for /session and /me, and the private handler in extension_terminal.go repeats the clean-path check that the proxy's extensionCleanRoute already does. sodaspaces-workspace.ts:951-996 also sends Applications requests through this.api, so it hits the same blocked application-builds route.

### DSP-5: A run's deadline is the minimum of six budgets, and the attempt clock starts at packet time

- **Subsystem:** Factory dispatch and run start. **Category:** too-many-states-or-budgets. **Verdict:** confirmed.
- **Removable:** ~350 lines. **Risk:** medium. **Visible change:** UI, DB schema. **Behaviour-preserving:** yes.
- **Status:** done (`b1d4c63e`).
- **Where:** `internal/factory/control/dispatch_selection.go:104-175`, `internal/factory/control/dispatch_timeonly.go:336-349,458-548`, `internal/factory/control/dispatch.go:301-312`, `internal/factory/control/dispatch_occupancy.go:47-55`, `internal/store/factory_dispatch_packet.go:162-198,1126-1218`, `internal/store/factory_attempt_allowances.go:1168-1186`, `internal/factory/allowance.go:413-425`, `internal/factory/control/readiness_work.go:13-23,84-166`, `internal/factory/control/dispatch_preparation.go:177,226,251`, `internal/store/factory_intents.go:282-297,383-405`, `internal/store/factory_context_selection.go:24-27,56`, `internal/store/factory_attempt_allowances.go:1233-1243`, `internal/store/factory_reassessments.go:655-669`, `internal/store/factory_delivery_repair_adoption.go:71-84`, `internal/store/schema.go:35-36`, `cmd/soda-identity/src/schema.rs:30-31`

**What.** sizeAttemptDeadline sets the deadline to min(sponsorship AllowanceMinutes − confirmed usage − held planned minutes, connection rolling-usage budget, attempt active-time remaining, 179). Time-only roots also take the min with IntentMinutes, ReassessmentMinutes, the app-verification allowance (2 runs / 60 minutes) or repair-charge minutes. The packet transaction recomputes the sponsorship and rolling-usage checks. Separately, every pass carries five native-RPC counters, and every judgment reservation increments a global factory_judgment_budget.spent under a row lock.

**Simpler.** Keep one coder-time allowance per repository connection (sponsorship AllowanceMinutes or the rolling budget, not both), the 179-minute per-run cap and the attempt limit. Start the attempt clock when the host confirms launch. Fold the per-subject minute settings into that allowance. Drop factory_judgment_budget and the readiness RPC counters, keeping the pass deadline. The hardening hold then reduces to one rule: if fewer than 20 minutes remain, wait with allowance_exhausted.

**Risk.** This changes funding semantics and settings; the allowance/budget finder may report the allowance.go side too. Starting the clock at launch means runs that never start no longer cost attempt time, which is the intended change.

**Visibility.** Settings would show one coder-time allowance instead of sponsorship minutes plus a rolling connection budget plus per-subject minutes. Deleting the write-only judgment counter and the dead planned-minutes arithmetic (always 0 under the one-session clamp) changes no behaviour.

**Safe first step.** Delete the write-only global judgment counter. Remove the `SELECT spent FROM factory_judgment_budget WHERE id=1 FOR UPDATE` scan and the `UPDATE factory_judgment_budget SET spent=spent+...` statement from internal/store/factory_context_selection.go (lines 24-27, 56-58), factory_attempt_allowances.go (1232-1235, 1243-1245), factory_delivery_repair_adoption.go (71, 84), factory_intents.go (282-285, 296-298 and 383, 404), and factory_reassessments.go (655, 669). Fix the ReserveDeliveryJudgment comment, which mentions a "global judgment ceiling". Delete the test assertions that read spent (factory_mcp_live_test.go, factory_mcp_bound_test.go, factory_journey_readback_test.go, factory_feed_comment_test.go, factory_reassessments_test.go). The CREATE/INSERT in internal/store/schema.go:35-36 and cmd/soda-identity/src/schema.rs:30-31 can be removed in a follow-up schema change. Observable behaviour stays the same, because spent is never compared or displayed.

**Related.** The prod-hardening branch (the task names it hardening/production-gaps, which does not exist) adds MinUsefulRunMinutes, allowanceTotal/allowanceBound and an Sscanf detail read-back to dispatch_selection.go. That is a seventh rule layered on the same budgets. The allowance side is in internal/factory/allowance.go and internal/factory/stall.go (a second CheckpointUnix accounting struct).

### GOM-4: Six Forgejo actor credentials plus a native-binding inventory and per-repository actor pickers that record the same identities three times

- **Subsystem:** Other Go packages (forgejo, acceptance, project, host, tailnet, application, config, strictjson, filelock, avatar, cmd/soda-dashboard, cmd/soda-tailnet, cmd/soda-extension). **Category:** unused-configuration. **Verdict:** partially-confirmed.
- **Removable:** ~350 lines. **Risk:** medium. **Visible change:** UI. **Behaviour-preserving:** yes.
- **Status:** owner decision.
- **Where:** `internal/config/config.go:24-80,135-235`, `internal/web/server.go:76-111`, `internal/web/api/factory_status.go:50-55`, `internal/web/api/factory_settings.go:90-110`, `internal/factory/grants.go:109-125,205-215,279-310`, `internal/factory/control/acceptance_initial.go:108`, `internal/factory/control/merge_evidence.go:159,189`, `internal/factory/control/delivery_judgment_producer.go:107-108`, `frontend/spaces/sodaspaces-factory-settings.ts:74-80,975-990,1516-1545`, `cmd/soda-setup/src/config.rs`, `cmd/soda-setup/src/factory_enrollment.rs:137-211,470-520,1375,1612-1720`, `scripts/fixtures/portcontracts/dashboard_config_vectors.json`

**What.** Config has separate credential files for background/publisher, review, resolve, merge (also reused for CheckAssessor), lifecycle and graph. Graph alone has an ActorID in config; every other actor is resolved through /user. On top of that, config carries factory_native_bindings, a list of {token_id, actor_id, repository_id, kind}. Its only reader is factory_status.go, which passes it to the settings page so the founder picks Publisher, PR creator, Reviewer, Thread resolver and Merger from five dropdowns. The comments say the configured credential 'must belong to the repository policy's distinct review-only operation binding', so the same identity is recorded three times and has to agree. validateReviewCredentials repeats the 'absolute path + requires background service inputs' block four times, and validateGraphConfig adds a fifth. soda-setup (Rust) mirrors the schema and enforces fixed file names (factory_enrollment.rs:470-475), and a port-contract fixture pins it.

**Simpler.** Use at most two Forgejo identities: a publisher, and a reviewer distinct from it, because Forgejo will not let an author approve their own PR. Resolve, merge, lifecycle and graph run as one of those two. soda-setup writes both credential files. The dashboard looks up each actor ID once through /user on startup and hands it to the coordinator. Every control-plane reader of policy.Publish/Create/Review/Merge/Resolve.ActorID then reads the coordinator's resolved IDs instead. After that, delete factory_native_bindings (config, status API, Rust inspect/reconcile, fixture), ForgejoGraphActorID, the unused TokenID, the policy actor fields and the five settings dropdowns. Fold the repeated credential validation into one loop.

**Risk.** Spans Go config, Rust soda-setup enrollment and the settings UI. Confirm which Forgejo actions really need a distinct actor (self-review ban, and possibly branch protection on merge). Repository policy is stored as a JSON blob (store.loadGrant 'factory_policies'), so dropping fields needs no migration, but old rows will carry unused keys.

**Visibility.** The five actor dropdowns and the 'Native actors need host administrator enrollment' gate disappear from repository policy settings.

**Safe first step.** In internal/config/config.go, replace the four copy-pasted blocks in validateReviewCredentials (resolve, review, merge, lifecycle) with one loop over name/value pairs that keeps the error strings identical ("%s must be an absolute path", "%s requires background service inputs"). Leave validateGraphConfig's message ("must be absolute") alone, or keep its exact wording. No other behaviour changes.

**Related.** The policy actor IDs are consumed across internal/factory/control: acceptance_initial.go, merge.go, merge_evidence.go, delivery_judgment_material.go, delivery_judgment_producer.go, publication_authority.go, publication_children.go and correction.go. They are also used in internal/factory/review_native.go (validateResolveBinding). The Rust native-binding enrollment is in cmd/soda-setup/src/factory_enrollment.rs (inspect_native_bindings and reconcile_native_bindings). The Go and TS tests that cover the inventory are internal/web/factory_settings_test.go and tests/frontend/factory-settings-controls.test.ts.

### RTY-4: A failed context-bundle JEV call leaves a recorded failure receipt that replay always rejects, so the run holds until its deadline; only Stop+Retry recovers it, with a dedicated implementation-retry path for implement-phase runs

- **Subsystem:** Retry kinds across store, control and UI. **Category:** too-many-states-or-budgets. **Verdict:** partially-confirmed.
- **Removable:** ~330 lines. **Risk:** medium. **Visible change:** UI. **Behaviour-preserving:** no.
- **Status:** done (`72c5afe7`, `c62f6ec1`).
- **Where:** `internal/factory/control/judgment_advice.go:49-80`, `internal/factory/control/context_bundle_advice.go:121-149`, `internal/factory/control/context_bundle_prepare.go:53-60`, `internal/store/factory_context_selection.go:15-60`, `internal/factory/control/dispatch_launch.go:690-760`, `internal/factory/control/dispatch_context_recovery.go:42-62`, `internal/factory/control/lifecycle_retry.go:38-55,145-153`, `internal/factory/control/context_resolution.go:64`, `internal/store/factory_implementation_retry.go:1-309`, `internal/store/factory_attempt_allowances.go:797-830`

**What.** Every launch goes through launchTail, which sets run.ContextPending and calls PrepareContextBundleSelection before FactoryLaunch. ReserveContextBundleSelection writes context_selection_snapshot onto the capability before the JEV call. If the call fails, the snapshot stays without a receipt. Every later pass takes the 'ValidDigest(snapshot)' branch, PriorityJudgment returns not-found, and the run reports context_selection_hold/'context_selection_unavailable' until its deadline. None of the retry kinds apply. RetryRun refuses because the run is not reconciled (ErrPendingRuns, 'stop or reconcile them first'). UseCurrentContext requires reason=='context_selection_attention'. The implementation retry needs the owner to Stop first (yielding Cancelled/'stop-before-start'), applies only to implement-phase children, and requires RemainingSeconds>0. Meanwhile the packet started the attempt active-time allowance (admitAttemptAllowanceTx Checkpoint(now, true)), so the hold burns the root's time. Once the deadline passes, recoverUnused marks the run Failed 'dispatch launch never reached the host'. Automatic relaunch then hits the exhausted allowance and returns quietly (RTY-5), and admitDispatchAttempt reports 'attempt is closed; maintainer intervention is required'. Explicit Retry then refuses because the prior assignment is still 'assigned', not 'finished' (explicit_retry.go:283).

**Simpler.** When the context-bundle receipt holds a failure ("consultation_failed"), treat the choice as unspent: clear the snapshot and ask JEV again on the next pass (JEV is free), or fall back to the 'current' bundle. A held run then never outlives one failed call. Once implement-phase Stop+Retry no longer needs its own revalidation path, delete CommandRetryImplementation and factory_implementation_retry.go and let ordinary Retry handle it. Optionally start the allowance clock at FactoryLaunch instead of at packet admission.

**Risk.** This touches the context-bundle selection that other finders probably own. Re-asking JEV after a failure is cheap because JEV is free per AGENTS.md. Starting the allowance at launch changes the accounting for time the run spends held.

**Visibility.** Runs no longer sit on context_selection_hold until the deadline, and the implementation-retry queued reason text disappears.

**Related.** The same single-shot reserve-then-record pattern, where a failed receipt can never be replayed, also covers the other paid JEV purposes that use AdvisePurpose/ReservePriorityJudgment, such as issue planning in internal/store/factory_intents.go:420-439. Any of them could stick after a single failed call. RTY-5 covers the quiet automatic-relaunch exit when the allowance is exhausted. RTY-1 covers deleting CommandRetryImplementation.

### CTX-5: Required inputs are delivered three times: rendered prompt, current.role inline, and current_role_page as HMAC-cursored 4 KB JSON fragments

- **Subsystem:** Worker context assembly, project library and research. **Category:** duplicated-layer. **Verdict:** confirmed.
- **Removable:** ~300 lines. **Risk:** low. **Visible change:** model tools. **Behaviour-preserving:** no.
- **Status:** owner decision.
- **Where:** `internal/factory/context_inputs.go:12-91`, `internal/factory/control/dispatch_attempt.go:497`, `internal/factory/control/publication_children.go:1117`, `internal/factory/dispatch_prompt.go:122-133,380-390`, `internal/web/api/factory_mcp_context.go:63-90`, `internal/web/api/factory_mcp_context_page.go:1-181`, `internal/factory/context_current.go:169-180`, `internal/store/factory_context_current.go:184-196`, `internal/factory/dispatch_prompt_stage.go:137`, `internal/web/api/factory_mcp_bound.go:158`, `docs/design/factory-worker-mcp.md:345`

**What.** The dispatch prompt already renders the accepted title, body, sources, resolutions, prerequisites, repository context files and plan. CaptureAcceptedContext copies the same PromptInputs into Assignment.AcceptedContext, which factory.context returns as current.role.accepted. CurrentAcceptanceContext repeats it again as title/content/source digests. The whole role is then serialised a second time into current_role_page: sorted fields cut into 4096-byte 'json-fragment' records that the model must concatenate and JSON-decode, continued by HMAC-signed cursors that expire after 5 minutes. When the inline copy exceeds 8192 bytes, current.role is silently replaced by {kind, record_source:'required_current'}, so only the fragment stream remains.

**Simpler.** Return the role once as plain JSON. If it is large, split it by named field with a plain integer offset; it needs no HMAC, snapshot or expiry because the capability already authenticates the caller. Drop CurrentAcceptanceContext, which only holds digests of text already present, and drop the ValidateAcceptance re-hash on read. Better still, since the prompt already carries the objective, sources and plan, factory.context could carry only what changes during the run (task receipts, review facts, CI assessment).

**Risk.** The model reads the same facts through a simpler shape. The prompt wording for the Inspector changes.

**Visibility.** The factory.context response shape (current, current_role_page, cursor input schema) changes. The frontend does not render it.

**Related.** internal/store/factory_repair_scope.go:144 uses a similar 'utf8-json-fragment' offset paging for repair scope. docs/development/factory-workflow-contracts.md:1668 and docs/design/factory-worker-mcp.md:345 document the current_role_page fragment protocol.

### FEF-1: Sponsorship form makes the owner pick a broker grant, see the credential generation and tick role boxes; the grant select drops its saved value (the re-pick bug)

- **Subsystem:** Frontend: Factory settings, work, progress, request, intent screens. **Category:** unused-configuration. **Verdict:** confirmed.
- **Removable:** ~250 lines. **Risk:** low. **Visible change:** UI. **Behaviour-preserving:** yes.
- **Status:** owner decision; coordinate with prod-hardening.
- **Where:** `frontend/spaces/sodaspaces-factory-settings.ts:1546-1599`, `frontend/spaces/sodaspaces-factory-settings.ts:880-929`, `frontend/spaces/sodaspaces-factory-settings.ts:551-585`, `frontend/spaces/sodaspaces-factory-settings.ts:1631-1690`, `frontend/spaces/sodaspaces-factory-settings.ts:95-142`, `frontend/spaces/sodaspaces-factory-access-response.ts:24-30,83-112`, `tests/installed/qualification/prepare-funding.ts:44-63,89-98`, `internal/web/api/factory_sponsorship.go:35-140`

**What.** The 'Provider allowance and sponsorship' fieldset asks the owner to: choose a connection; pick a 'Project access' broker grant, shown as raw `project_id · grant_id`; read 'Credential generation N'; tick Builder/Inspector role boxes; set allowance minutes and concurrent sessions; tick an active box. A second form in the same fieldset sets the connection's rolling usage budget. Below that the page lists execution usage, a permission table and required owner actions. The grant, generation and roles go back to the server, which validates them against data it already holds.

**Simpler.** Quick fix (about 2 lines): render the options with `?selected=${grant.id === sponsorship.grant_id}` and `?selected=${row.id === connection}`, the way renderPolicyBindings does, instead of putting `.value` on the select. Real fix: one 'Factory account' form with a connection select, one minutes field and a 'Use this account' toggle, saved as PUT {connection, minutes, active}. The server: - resolves the single live grant for the repository's project, or creates it, with the sponsor action taking over the explicit credential-exposure consent; - reads the current generation itself; - fixes roles to soda-coder and soda-reviewer (this removes the owner's ability to sponsor only one role, which completion and dispatch respect today); - writes the one minutes value as both the allowance and the rolling budget, coordinated with prod-hardening's allowance-hold notice. Then delete the grant select, the generation line, the role checkboxes, concurrent sessions, the separate budget form, and the usage, permission and required-action tables.

**Risk.** The server must handle several live grants for one connection+project, which the qualification script currently asserts cannot happen: pick the newest or refuse with a clear message. Sponsoring stays limited to the connection owner; the server already checks that.

**Visibility.** The Factory tab loses about six inputs and three readout tables, and saving no longer needs a re-pick. The ?selected fix alone is invisible apart from the bug going away.

**Safe first step.** In frontend/spaces/sodaspaces-factory-settings.ts renderProviderControls: 1. Remove `.value=${safeText(sponsorship.grant_id)}` from the grant_id select. Add `?selected=${!sponsorship.grant_id}` to the placeholder option and `?selected=${safeText(grant.id) === safeText(sponsorship.grant_id)}` to each grant option. 2. Do the same for the selected_connection select: remove `.value=${connection}` and use `?selected=${safeText(row.id) === connection}` on each option. Nothing else changes except that the saved grant and connection now display, so saving no longer needs a re-pick.

**Related.** Other `.value=` bindings on a <select> whose options are mapped in the same render have the same bug pattern; grep frontend/spaces for `<select .* .value=`. The prod-hardening branch's frontend/spaces/sodaspaces-factory-allowance.ts depends on sponsorship allowance_minutes, so merging the two budgets has to be coordinated with it. tests/installed/qualification/prepare-funding.ts holds the same grant-and-roles derivation that the server would take over.

### MCP-2: Finish tools freeze first and validate later; 16 park paths wedge the assignment silently

- **Subsystem:** Model-facing MCP tools. **Category:** duplicated-layer. **Verdict:** confirmed.
- **Removable:** ~250 lines. **Risk:** medium. **Visible change:** UI, model tools. **Behaviour-preserving:** yes.
- **Status:** done (`27319fac`, branch simplify/result-channel): CheckMCPTerminalFinish runs the settler mapping before recording; no park paths remain. The Builder/Fixer final-checkpoint wait stays.
- **Where:** `internal/web/api/factory_mcp.go:1063-1085`, `internal/factory/control/mcp_admission.go:138-153`, `internal/store/factory_mcp_submission.go:318-336`, `internal/factory/control/mcp_fw40.go:46-124`, `internal/factory/control/mcp_fw40.go:263-603`, `internal/factory/control/mcp_correction_finish.go:21-23`, `internal/factory/control/mcp_settle.go:118-143`, `internal/factory/control/mcp_settle.go:368-401`, `internal/factory/control/dispatch_result.go:38-52`, `internal/web/api/factory_mcp_issue_plan.go:16-69`, `internal/web/api/factory_mcp_product_plan.go:13-52`, `internal/web/api/factory_mcp.go:535-578`, `internal/web/api/factory_mcp.go:625-644`

**What.** serveMutation records a terminal proposal after only loose key and enum checks. From then on, any new submission is refused as frozen (ErrMCPSubmissionFrozen). Typed decoding and binding run later, in the background settler (driveMCPTerminals, then mapFW40Terminal). There a proposal either settles, fences the assignment as NeedsHuman with a summary the worker never sees, or parks: nothing happens and the receipt stays 'recorded'. A parked terminal also blocks the run-output path, so the assignment stays 'assigned' with capacity held until stall handling or a human notices. To work around this, pre-freeze validators were added per tool, each duplicating its settler and each repeating the replay and freeze lookups.

**Simpler.** Write one synchronous function per finish tool: result(assigned, current, payload) (factory.Assignment, error). Call it in the tool handler before recording anything. An error goes straight back to the model as the refusal; nothing is recorded, so the worker can correct and resubmit. On success, record the receipt and finish the assignment in the same request. Keep the deferred drive only for Builder and Fixer finishes that must wait for the native final checkpoint (checkpointTerminalReady). Delete parkFW40, the carrier sniffers (isObjectRaw, isObjectArrayRaw, isArrayRaw), validateIssuePlanSubmission, checkProductPlanWalkthroughs, CheckProductPlanFinish (on prod-hardening), the product-plan replay branch in admitMutation and the parked case in settleRecordedMCPBeforeRun.

**Risk.** Settlement side effects (decompose semantic settle, repair chain, artifact resolution) must move into the synchronous path or run right after the insert in the same request. The checkpoint-gated Builder/Fixer finish keeps its wait.

**Visibility.** Workers get the exact defect as a tool error and can retry, instead of a frozen proposal. The Feed shows fewer 'needs human attention' fences and no silently stuck assignments. Merging the duplicate validators into the settlers is behaviour-preserving on its own.

**Safe first step.** Take the strict issue-plan decode, phase, binding and approach checks out of settleIssuePlan (internal/factory/control/mcp_fw40.go) and put them in one exported control function that returns an error. Have validateIssuePlanSubmission (internal/web/api/factory_mcp_issue_plan.go) call that function in place of its own copy, keeping its expected_plan_digest refusal wording. Have settleIssuePlan wrap the same error as its fence detail. Then route the existing receipt-replay and freeze lookups in validateIssuePlanSubmission and admitMutation's product-plan branch (factory_mcp.go:625-644) through the single check AdmitMCPSubmission already performs, so they are not repeated. Refusal and fence texts stay the same.

**Related.** The prod-hardening branch (the real name of hardening/production-gaps) adds CheckProductPlanFinish in internal/factory/control/mcp_fw40.go and calls it from internal/web/api/factory_mcp_product_plan.go. That is a third pre-freeze copy of the product-plan settlement checks, and it touches the same files: mcp_admission.go, factory_mcp.go, factory_mcp_issue_plan.go, dispatch_result.go and factory_mcp_submission.go.

### MCP-3: Payload shape is defined three times and the layers disagree, making whole paths unreachable

- **Subsystem:** Model-facing MCP tools. **Category:** duplicated-layer. **Verdict:** confirmed.
- **Removable:** ~250 lines. **Risk:** medium. **Visible change:** UI, model tools. **Behaviour-preserving:** yes.
- **Status:** partly done (`27319fac`): one required/optional key table drives schema and admission; unread keys and the hex checks are gone. The communication, library and repair payload schemas still pair a hand-written schema with a check function.
- **Where:** `internal/web/api/factory_mcp_role_schema.go:12-114`, `internal/web/api/factory_mcp_communication.go:13-51`, `internal/web/api/factory_mcp.go:296-533`, `internal/factory/mcp_communication.go:66-72,253-281,302-304`, `internal/store/factory_mcp_communication.go:19-35`, `internal/store/factory_library.go:306-385`, `internal/factory/control/mcp_fw40.go:366-372,436-495`, `internal/factory/control/task_checkpoint.go:36-50,108-112`, `internal/factory/plan_task.go:22`, `internal/factory/planning.go:161`

**What.** Each tool's payload has (1) an advertised JSON schema that the SDK enforces (rolePayloadSchema is built from mcpMutationPayloads plus a hard-coded optional-key list, alongside communicationPayloadSchema and others), (2) Go key, enum and hex specs (mcpPayloadSpec, mcpOptionalPayloadKey, checkPayload*, checkReviewPayload), and (3) strict decoders at settle or store time (strictDecodeField, DecodeMCPCommunication again inside the transaction, decodeTaskOutput). They disagree, so some features cannot be reached and some admitted values break later.

**Simpler.** Give each finish or record tool one Go payload struct. Generate its schema with jsonschema.For[T], with descriptions in struct tags, and let the SDK validate. Decode once with DisallowUnknownFields into T. Delete mcpPayloadSpec, mcpMutationPayloads, the checkPayload* family, mcpOptionalPayloadKey and the hand-written key lists. Choose per feature: either delete the worker amendment, challenge and decision_owner paths, or add those fields to the struct. Keep the DecisionOwner field itself, because the internal intent-revision feed writer sets it. Use ValidDigest at admission so submit_task_output cannot admit values that later fail decoding. Drop the unread top-level finish_issue_plan alternatives and unresolved_facts, and the unread finish_incomplete affected_task_ids and requested_resolution.

**Risk.** Advertised schemas change shape slightly; required fields nobody reads must be dropped deliberately.

**Visibility.** Advertised tool schemas become exactly what the server accepts. Non-code review and knowledge amendment either start working or are removed, both of which show in the Feed.

**Safe first step.** Delete the worker knowledge-amendment and challenge path. The SDK schema rejects these keys before they can reach it, so observable behaviour stays identical. Five deletions: (1) the RevisesItem, ExpectedRevision and Challenge fields in internal/factory/mcp_communication.go (lines 70-72); (2) their validation in the same file (lines 253-264 and 302-304); (3) the `if content.RevisesItem != ""` branch in createCommunicationLibraryTx (internal/store/factory_library.go:311-313); (4) amendCommunicationLibraryTx (factory_library.go:332-385); (5) TestLearnedAmendAndChallengeCAS in internal/store/factory_library_test.go. Keep the DecisionOwner field, because internal/store/factory_feed_intent.go:29 sets it.

**Related.** Other tools follow the same parallel-schema pattern: libraryPayloadSchema/checkLibraryPayload, researchPayloadSchema/checkResearchPayload, verificationJourneyPayloadSchema/checkVerificationJourneyPayload, and the repair diagnosis/draft schema and check pairs, all wired in factory_mcp.go mcpEnvelopeSchema and checkPayload. The prod-hardening branch adds another pre-settle validator, CheckProductPlanFinish in mcp_fw40.go. It also rewrites the checkPayload* refusal texts; its new hex message says "64 lowercase hex" while isHex64 still accepts uppercase.

### RUN-1: The task-output and finish round trip makes the model copy back what SodaOS already observed; one advertised option always fences and one tool has no consumer

- **Subsystem:** Run end: task events, checkpoints, results and settlement (RUN). **Category:** unneeded-validation-or-digest. **Verdict:** confirmed.
- **Removable:** ~250 lines. **Risk:** medium. **Visible change:** model tools. **Behaviour-preserving:** yes.
- **Status:** partly done (`27319fac`, `4e6dc8af`): request_handoff and `unchanged` deleted; SodaOS fills the task output digest, native child and final candidate. submit_task_output still exists as the parent acceptance step; the progress loop does not checkpoint by itself.
- **Where:** `internal/web/api/factory_mcp_checkpoints.go:19-100`, `internal/factory/control/task_checkpoint.go:28-57`, `internal/factory/control/task_checkpoint.go:151-204`, `internal/web/api/factory_mcp.go:337-344`, `internal/web/api/factory_mcp.go:413-421`, `internal/web/api/factory_mcp.go:505-530`, `internal/web/api/factory_mcp_role_schema.go:59-60`, `internal/web/api/factory_mcp_role_schema.go:89-104`, `internal/factory/control/mcp_settle.go:239-256`, `internal/factory/control/mcp_settle.go:345-366`, `internal/factory/control/mcp_checkpoint_finish.go:15-43`, `internal/factory/control/mcp_admission.go:46-49,107-109,162-169`, `internal/factory/mcp_profile.go:131,176,202,318`, `internal/store/factory_mcp_submission.go:142,338-354`, `internal/store/factory_mcp_submission.go:461-470`, `internal/store/factory_task_outputs.go:12-35`, `lib/host/src/terminal/factory/muse/commands.rs:23`, `lib/host/src/factory/native_control.rs:526`

**What.** For each finished Builder child, the trusted Rust host records child_reported with the output digest. Go imports it into a plan-task receipt (output_ready). factory.context then shows the model ready_outputs {task_id, output_digest, native_child_ref} plus manifest_digest. The model must call factory.submit_task_output with task_id, manifest_digest, output_digest, native_child_ref and evidence_refs, all copied from that page. driveRunCheckpoints then matches the submission back against the receipt field by field (task_checkpoint.go:180-181), re-proves the capability binding, moves the MCP receipt recorded→verifying, and only then calls the host checkpoint. After every task commits and the final candidate commits, the guidance tells the model to 'Finish implementation only after integration.status is committed, using its candidate and receipt_digest'. The model has to copy the candidate commit out of run.FinalCheckpoint into finish_implementation, and checkpointTerminalReady compares it with run.FinalCheckpoint again. finish_implementation does not even accept receipt_digest (mcpOptionalPayloadKey allows only unchanged and candidate), so the guidance names a field the tool would refuse. The advertised `unchanged:true` passes MCP admission (factory_mcp.go:415) but always lands in settleCodeFinish with an empty candidate and is fenced as NeedsHuman/harness_no_report. factory.request_handoff (a Builder tool) is admitted, counted and capped at 256 (MCPMaxHandoffs), but no control code ever reads a handoff submission.

**Simpler.** Have the progress loop checkpoint output_ready tasks itself, in report order, using the receipt it already uses to build the request. If parent acceptance is wanted, reduce submit_task_output to {task_id}. Make finish_implementation take no binding fields and have the settler read the candidate from the committed run.FinalCheckpoint. Delete `unchanged`, or settle it as completed-without-candidate. Delete factory.request_handoff with its profile entry, schema, payload spec, both cap checks, MCPMaxHandoffs and the store count. Rewrite the context guidance and both Rust parent prompts to match.

**Risk.** Changes the Builder/Fixer tool contract and prompts. The automatic checkpoint path must keep report-order serialization and the 'busy' retry. The host still enforces ownership and expected head, so no trust boundary is lost.

**Visibility.** Only tool schemas, descriptions, refusals and the context guidance change. The Feed and UI do not render these submissions.

**Safe first step.** In internal/factory/control/mcp_settle.go, delete the unreachable ValidCommit branch of settleCodeFinish (Builder's finish_implementation with a valid candidate always goes to settleTypedImplementation), so it only fences. Also delete the unreachable inner `default:` arm in the FinishIncomplete/FinishImplement switch. Observable behaviour stays identical.

**Related.** The Rust parent prompts in lib/host/src/terminal/factory/muse/commands.rs and lib/host/src/factory/native_control.rs tell the parent to copy the digest and child ref. Docs still describe request_handoff as if it works: docs/development/factory-workflow-contracts.md:1338,1393,1922, docs/design/factory-worker-mcp.md:516, docs/architecture/factory-interfaces.md:99 and docs/design/factory-feed-components.md:338,968. The handoff cap is checked twice, in control/mcp_admission.go and in store checkMCPHandoffCapTx.

### STO-3: JSON-content CHECK constraints on assignments and allowances duplicate Go validation and force a pg_constraint-rewriting migration for every new subject kind

- **Subsystem:** Store (PostgreSQL layer) structure. **Category:** unneeded-validation-or-digest. **Verdict:** confirmed.
- **Removable:** ~235 lines. **Risk:** low. **Visible change:** DB schema. **Behaviour-preserving:** yes.
- **Status:** owner decision.
- **Where:** `internal/store/schema.go:80-81`, `internal/store/factory_delivery_schema.go:1-61`, `internal/store/factory_repair_schema.go:1-61`, `internal/store/factory_delivery_repair_schema.go:1-103`, `internal/store/store.go:106-114`, `internal/store/schema_test.go:25-90`, `internal/factory/assignment.go:503-548`, `internal/factory/allowance.go:177`, `cmd/soda-identity/src/schema.rs:75-76`

**What.** The issue column on factory_assignments and factory_attempt_allowances carries 6.7 KB of CHECK SQL: five issue-zero alternatives per table, regexes on IDs and digests, data->>'id'=id, data->>'repository'=repository::TEXT, and role/phase literals. It re-validates the JSON document the store just marshalled after calling Assignment.Validate() or AttemptAllowance.Validate(). Each new issue-zero subject kind (delivery, stall, delivery_repair) came with its own migration. The migration finds the constraint by pg_get_constraintdef LIKE '%completion%' AND LIKE '%intent%', splices an 'OR (...)' branch into the definition text, and drops and re-adds the constraint.

**Simpler.** Use 'issue INTEGER NOT NULL CHECK(issue>=0)' on both tables in internal/store/schema.go and in the mirrored cmd/soda-identity/src/schema.rs, and keep validation in Go. Delete the three migration files, including the deliveryRepair*Scope constants, delete their calls in store.go and the schema_test.go code that exercises them. For retained databases, add one idempotent 'ALTER TABLE ... DROP CONSTRAINT IF EXISTS <t>_issue_check, DROP CONSTRAINT IF EXISTS <t>_issue_scope_check' per table. Re-adding CHECK(issue>=0) is optional.

**Risk.** The Go validators already reject the same rows. Only a hand-written raw-SQL insert loses its database-level guard.

**Visibility.** Constraint change only; rows are untouched. Removes a real development tax: no more per-subject-kind DDL surgery, and no startup refusals after a CHECK edit.

**Safe first step.** 1. Delete internal/store/factory_delivery_schema.go, internal/store/factory_repair_schema.go and internal/store/factory_delivery_repair_schema.go, and remove their three calls in internal/store/store.go:106-114. 2. In internal/store/schema.go:80-81 and cmd/soda-identity/src/schema.rs:75-76, replace the issue CHECK expressions with CHECK(issue>=0). 3. For retained databases, add one idempotent ALTER TABLE per table: DROP CONSTRAINT IF EXISTS <t>_issue_check, DROP CONSTRAINT IF EXISTS <t>_issue_scope_check. 4. Delete the schema_test.go code that references deliveryRepair*Scope and issue_scope_check. Rows the Go validators accept are stored exactly as before.

**Related.** cmd/soda-identity/src/schema.rs is a mechanically mirrored copy of the Go schema and carries the same CHECK text, so both copies must change together. Other tables in internal/store/schema.go also use JSON-content regex CHECKs (82 `~ '` occurrences across internal/store/*.go). Those are the same pattern but not covered by this finding.

### BUD-4: Issue-attempt active-time clock: a 120-minute wall clock that keeps running through CI and judge waits, shortens later runs, gates native operations and withdraws pending merges

- **Subsystem:** Budgets, allowances and limits (cross-cutting). **Category:** too-many-states-or-budgets. **Verdict:** partially-confirmed.
- **Removable:** ~200 lines. **Risk:** medium. **Visible change:** UI, model tools. **Behaviour-preserving:** no.
- **Status:** done (`b1d4c63e`): the attempt clock is a per-run limit.
- **Where:** `internal/factory/allowance.go:19-39`, `internal/factory/allowance.go:411-448`, `internal/factory/control/dispatch_selection.go:142-175`, `internal/store/factory_attempt_allowances.go:51-175`, `internal/store/factory_attempt_allowances.go:423-455`, `internal/store/factory_attempt_allowances.go:641-795`, `internal/store/factory_assignments.go:185-245`, `internal/factory/control/merge_reconcile.go:42-52`, `internal/factory/control/dispatch_attempt.go:80-95`, `internal/factory/control/publication_children.go:88-95`, `internal/factory/control/lifecycle_retry.go:79`, `internal/factory/control/review_cycle.go:694`, `internal/factory/control/correction.go:327`, `internal/factory/control/merge_evidence.go:115`, `internal/factory/control/publication_reconcile.go:273`, `internal/factory/control/delivery_producer.go:111`, `internal/factory/control/delivery_judgment_material.go:121`, `internal/factory/control/delivery_judgment_producer.go:209`, `internal/store/factory_publications.go:120-128`, `internal/store/factory_merges.go:110-116`, `internal/factory/control/lifecycle.go:50-60`, `internal/factory/dispatch_prompt.go:346-359`, `internal/store/factory_context_current.go:140-160`, `frontend/spaces/sodaspaces-factory-settings.ts:1500`

**What.** A per-attempt wall clock with checkpoints, pause/freeze and closure. It caps every run's deadline and every native operation's NotAfter, on top of the run deadline, the spend cap and the correction-cycle cap.

**Simpler.** Stop time-bounding ordinary issue attempts only. Drop ActiveMinutes from the issue AttemptLimits and Settings ('Active minutes per attempt'). sizeAttemptDeadline stops capping issue runs by the attempt's remaining seconds and keeps the run deadline. The seven CapAttemptDeadline callers use their requested NotAfter, and ensureAttemptDeadlineForRegistrationTx no longer applies to issue roots. Remove the time branch of CloseExpiredAttempt and the merge withdrawal it drives, keeping Closed for terminal results. Remove the issue prompt sentence (dispatch_prompt.go:359) and active_seconds_remaining for issue contexts. Keep the ActiveSeconds/CheckpointUnix/Active/RemainingSeconds/Checkpoint machinery and the freeze for the time-only roots (intent, reassessment, stall repair, delivery repair and delivery admission), where it is the only time budget.

**Risk.** Removes the only wall-clock bound on a whole attempt; loops stay bounded by correction cycles, launch attempts and the spend cap. Many call sites change: 7 NotAfter callers and 4 CloseExpiredAttempt callers.

**Visibility.** Settings loses 'Active minutes per attempt', and attempts no longer close or withdraw merges because of time. factory.context drops active_seconds_remaining, and the prompt loses the active-time sentences. The factory_active_attempts index becomes optional.

**Related.** The same clock record backs the time-only roots: intent (stall.go IntentMinutes, insertIntentAttemptAllowanceTx), completion reassessment (factory_reassessment_allowances.go), stall repair (factory_repair_dispatch.go), delivery repair (factory_delivery_repair_dispatch.go) and delivery admission (factory_delivery_admission.go). There are also parallel time budgets in verification_allowance.go (60-minute app verification), the stall.go repair grant (120/30 minutes) and IntentExecutionAllowance.Checkpoint (stall.go:539). On prod-hardening, holdShortAllowance in dispatch_attempt.go adds yet another layer that holds runs on the coder-time allowance and explicitly excludes runs bounded by the attempt.

### FEF-2: Policy, capacity and grant forms carry dead and overlapping settings: hand-typed harness versions, an unused queue limit, four concurrency caps (policy/operator clamped to 1 anyway), three pause switches, and intent minutes that default to unfunded

- **Subsystem:** Frontend: Factory settings, work, progress, request, intent screens. **Category:** unused-configuration. **Verdict:** confirmed.
- **Removable:** ~200 lines. **Risk:** medium. **Visible change:** UI. **Behaviour-preserving:** yes.
- **Status:** owner decision; coordinate with prod-hardening.
- **Where:** `frontend/spaces/sodaspaces-factory-settings.ts:1479-1544`, `frontend/spaces/sodaspaces-factory-settings.ts:978-1029`, `frontend/spaces/sodaspaces-factory-settings.ts:1437-1477`, `frontend/spaces/sodaspaces-factory-settings.ts:867-878,931-965,1158`, `frontend/spaces/sodaspaces-factory-settings.ts:1096-1109,1168-1183`, `internal/factory/control/dispatch_selection.go:79-90,217-236`, `internal/factory/control/dispatch_launch.go:705-712`, `internal/factory/grants.go:138-141,199,226-262,316-323`, `internal/web/api/factory_policy.go:95-115`, `internal/factory/control/dispatch_timeonly.go:467-470`

**What.** The policy form asks for: coder and reviewer harness, harness version and model; target branch; required checks; five native actor bindings; intent minutes; minutes per attempt; correction cycles; a repository concurrency limit; enable and pause. Capacity asks for concurrent and queued runs. The operator grant has its own concurrency limit and active flag, the environment grant its own active flag, and sponsorship its own concurrency. Each is a separate form with its own command_id and expected_revision.

**Simpler.** One 'Factory' form. Its On/Off switch sets the policy enabled and unpaused and both grants active (one owner on a local appliance). It also holds coder model, reviewer model, minutes per attempt and correction rounds. Take the harness version from the staged pin: drop the two fields and the Muse equality check. Drop Queued runs (UI, API field, struct field, validation). Keep one concurrency number, the appliance capacity; policy and operator caps are clamped to 1 anyway. Default intent minutes to 30, or remove the dead DefaultIntentMinutes constant and fold intent minutes into minutes per attempt. Native bindings already auto-select when only one actor is enrolled for a kind. Keep target branch and required checks as optional advanced fields. Keep one pause control.

**Risk.** Folding the operator and environment grants into the owner's switch removes the operator-versus-owner separation, which matters only on a multi-user appliance (none exist). Dropping the Muse version check needs a small backend edit.

**Visibility.** About 12 inputs and 3 save buttons leave the Factory tab. Removing Queued runs and pre-filling the harness version from the pin change nothing at dispatch.

**Safe first step.** Remove Queued runs end to end. Delete the 'Queued runs' numberInput (settings.ts:1444), the max_queued line in submitCapacity (876) and the queued part of the capacity summary (1158). Delete MaxQueued from factoryCapacityRequest and the Capacity literal (factory_policy.go:97,114), and from factory.Capacity and its Validate (grants.go:316,323). Dispatch never reads it, so no dispatch outcome changes. Also delete the unreferenced DefaultIntentMinutes constant (grants.go:199).

**Related.** Sponsorship max_concurrent (store/factory_dispatch_packet.go) is a fourth cap on the same path. The dispatch_closed and policy_paused handling repeats across delivery_repair.go:216, store/factory_grant_commands.go:225, web/api/factory_issue_view.go:505 and frontend factory-work-view.ts/work-response.ts.

### RUN-3: Every phase has two finish channels (stdout result-json and MCP finish tool), and the stdout path has three near-identical derivers

- **Subsystem:** Run end: task events, checkpoints, results and settlement (RUN). **Category:** duplicated-layer. **Verdict:** confirmed.
- **Removable:** ~200 lines. **Risk:** medium. **Visible change:** UI, model tools. **Behaviour-preserving:** yes.
- **Status:** done (`27319fac`): the stdout derivers and result-json prompts are gone; a run without a finish call settles incomplete.
- **Where:** `internal/factory/control/dispatch_result.go:186-434`, `internal/factory/control/dispatch_attempt.go:752-775`, `internal/factory/dispatch_prompt_stage.go:135-140`, `internal/factory/dispatch_prompt.go:398-416`, `internal/factory/control/mcp_settle.go:100-179`, `internal/factory/mcp_profile.go:299-309`

**What.** derivePhasedAttemptResult, derivePhasedReviewAttemptResult and derivePhasedVerifyAttemptResult each repeat: the run/role/source identity check (already done by the caller at dispatch_result.go:202 and again in settleRecordedMCPBeforeRun), the run-outcome switch (mapPhasedRunOutcome is duplicated inline twice), ParsePhaseResult+BindAssignment, and the same completed/blocked/failed/cancelled status switch building an AssignmentResult. Separately, the Architect, Product Lead and Builder prompts (phasedReportContract) tell the worker to 'Close your work with exactly one fenced result-json block', while finish_issue_plan, finish_product_plan and finish_implementation also settle the same phases through the MCP settler. For review, the prompt says 'Finish through factory.finish_review ... Do not substitute a legacy review-json block', yet the stdout deriver still accepts a v2 result-json for review and verify.

**Simpler.** Merge the three stdout derivers into one: identity check once, mapPhasedRunOutcome once, ParsePhaseResult+BindAssignment once, and one status switch. ValidateForAssignment and the Candidate assignment run only for coder roles, and the refusal text depends on the phase. Delete derivePhasedReviewAttemptResult and derivePhasedVerifyAttemptResult. Then pick one channel per phase, which means MCP, since the prompts for review, verify, research and maintain_library already use only MCP. Remove the stdout acceptance for those phases and the result-json prompt branches for the MCP-finishing planner and builder phases. A run that ends without a recorded finish settles as one synthesized blocked result.

**Risk.** Removing a channel changes the worker prompts, and the Architect demo failures came from the MCP finish, so remove a channel only together with RUN-1 and the hardening fixes. Merging the derivers alone is safe.

**Visibility.** Merging the derivers keeps the same results. Dropping a channel changes prompts and refusal text, and may change the blocked-result summaries shown on Feed cards.

**Safe first step.** In internal/factory/control/dispatch_result.go, delete derivePhasedReviewAttemptResult (lines 305-369) and derivePhasedVerifyAttemptResult (lines 371-434). In derivePhasedAttemptResult, after mapPhasedRunOutcome, handle the maintain_library and research refusals inline. For the reviewer role, run the same ParsePhaseResult/BindAssignment path with refusal text by phase ("review completed..." / "verification completed..." / "run completed..."), then go through one status switch. That switch skips ValidateForAssignment and the Candidate assignment when the role is reviewer, so results and refusal texts stay identical. Delete reportPhasedReviewerResult once it is empty. Leave the prompts and the MCP settle path untouched in this step.

**Related.** The legacy review-json prompt branch in internal/factory/dispatch_prompt.go:407-413 and writePromptTypedReviewContract (dispatch_prompt_stage.go:142+) are still stdout finish channels for reviewer and correction phases. The correction phase (Fixer) has the same split: its prompt asks for two fenced blocks, while finish_correction also exists (mcp_profile.go:303). The stdout path also skips checkSettledArtifactRefs and checkpointTerminalReady, which mcp_settle.go applies.

### BUD-3: Two spend caps on coder minutes over the same usage rows: lifetime sponsorship minutes and the 24h connection budget

- **Subsystem:** Budgets, allowances and limits (cross-cutting). **Category:** too-many-states-or-budgets. **Verdict:** confirmed.
- **Removable:** ~150 lines. **Risk:** low. **Visible change:** UI. **Behaviour-preserving:** no.
- **Status:** done (`b1d4c63e`): the 24h rolling budget no longer gates dispatch; its settings record remains.
- **Where:** `internal/factory/sponsorship.go:15-35`, `internal/factory/sponsorship.go:58-60`, `internal/factory/grants.go:329-356`, `internal/factory/control/dispatch_selection.go:39-42`, `internal/factory/control/dispatch_selection.go:104-140`, `internal/store/factory_reservations.go:175-293`, `internal/store/factory_dispatch_packet.go:1076-1081`, `internal/store/factory_dispatch_packet.go:1133-1140`, `internal/store/factory_dispatch_packet.go:1204-1218`, `internal/store/factory_grant_commands.go:430-449`, `internal/store/factory_priority_inventory.go:620-636`, `internal/factory/effective_authority.go:47-53`, `internal/web/api/factory_sponsorship.go:20-48`, `internal/web/api/factory_connection_usage_budget.go:1-74`, `internal/web/api/factory_access_view.go:319-345`, `frontend/spaces/sodaspaces-factory-settings.ts:880-897`, `frontend/spaces/sodaspaces-factory-settings.ts:924`, `frontend/spaces/sodaspaces-factory-settings.ts:1550-1598`, `frontend/spaces/sodaspaces-factory-settings.ts:1651-1661`, `tests/installed/qualification/prepare-funding.ts:95-110`, `tests/installed/qualification/phases/factory-phase.ts:553`, `tests/installed/qualification/phases/factory-native.ts:127`, `tests/installed/qualification/native.ts:23`

**What.** Coder session minutes are capped twice from the same factory_usage table. One cap is per repository+connection for all time; the other is per connection over a rolling 24h. Each has its own grant record, API, Settings form and checks in both control and store.

**Simpler.** Keep one cap: the rolling 24h connection budget, shown as "coder minutes per day". It refills by itself, has a default and already has a form. Delete Sponsorship.AllowanceMinutes everywhere it is used: the field and its validation, UsageTotal, the control check (dispatch_selection.go:110-118), the packet check (factory_dispatch_packet.go:1204-1218), the priority-inventory allowRoom use (factory_priority_inventory.go:620-636), the API field and the Settings input. Update the qualification scripts that read allowance_minutes. In Settings Execution usage, show the rolling budget next to rolling 24h usage (ConnectionUsageCharge) instead of lifetime UsageTotal. The alternative is to keep only the sponsorship number and drop the budget table, API, command, missing-code, AuthorityRef field, held-reservation projection and Rust schema mirror. That is about 400 lines plus a schema change.

**Risk.** Removes a second limit while the remaining cap still bounds spend. With the rolling cap kept, spend is bounded per day rather than for all time.

**Visibility.** Settings loses 'Allowance minutes' (or 'Rolling minutes' under the alternative). Execution usage shows one consistent pair, and dispatch waits name one cap. The alternative would drop factory_connection_usage_budgets and its Rust mirror.

**Related.** factory_priority_inventory.go reads AllowanceMinutes for priority ordering (allowRoom). The installed qualification scripts under tests/installed/qualification read both allowance_minutes and rolling_minutes. The hardening branch (prod-hardening) deepens the coupling: allowanceTotal/allowanceBound and holdShortAllowance in dispatch_selection.go, and the new frontend/spaces/sodaspaces-factory-allowance.ts that renders allowance waits. The deadline sizing in BUD-1 and the AuthorityRef revision in BUD-2 share this code.

### MCP-1: Every role still copies server-known bindings into tool calls; only the Architect plan bindings were fixed

- **Subsystem:** Model-facing MCP tools. **Category:** unneeded-validation-or-digest. **Verdict:** confirmed.
- **Removable:** ~150 lines. **Risk:** medium. **Visible change:** model tools. **Behaviour-preserving:** yes.
- **Status:** done (`42a051e9`, `291451f6`); task output digests and native child refs still come from the model.
- **Where:** `internal/web/api/factory_mcp.go:91-100`, `internal/web/api/factory_mcp.go:305-345`, `internal/web/api/factory_mcp.go:608-610`, `internal/web/api/factory_mcp_bound.go:54-60`, `internal/factory/control/mcp_admission.go:154-161`, `internal/store/factory_mcp_submission.go:257-261`, `internal/web/api/factory_mcp_native_task.go:26-28`, `internal/web/api/factory_mcp_repair_preview.go:26-28`, `internal/web/api/factory_mcp_repair_scope.go:53-55`, `internal/factory/dispatch_prompt_stage.go:136-140`, `internal/factory/control/mcp_fw40.go:377-411`, `internal/factory/control/mcp_correction_finish.go:28-33`, `internal/factory/control/task_checkpoint.go:174-181`, `internal/web/api/factory_mcp_checkpoints.go:89-94`, `internal/web/api/factory_mcp_verification.go:31`, `internal/factory/control/app_verification.go:128`, `internal/factory/verification_journey.go:126-135`, `internal/factory/repair_diagnosis.go:91-110`, `internal/factory/intent_proposal_content.go:180-190`, `internal/factory/control/intent_bound.go:99`, `internal/factory/dispatch_prompt.go:489-494`, `internal/store/factory_planning_revisions.go:74-79`, `internal/web/api/factory_mcp_role_schema.go:37-104`, `internal/web/api/factory_mcp_systemone.go:111,132-134`, `internal/web/api/factory_mcp_artifact.go:19-22`

**What.** The model must mint or echo values the server already holds. Envelope: submission_id is required on every mutation, progress posts included, and must be 32-hex (hardening also accepts UUIDs, nothing else). expected_input_digest is optional, but when sent it must equal capability.ScopeDigest(), a value that never changes for a capability. Per role, after the hardening branch: Inspector finish_review needs tuple{cycle_id, head, base_oid} plus a coverage digest; coverage is required even for findings or incomplete, where it is ignored. Fixer finish_correction needs finding_digest. Builder and Fixer submit_task_output need manifest_digest, output_digest and native_child_ref. Test Pilot finish_app_verification needs delivery_id, input_digest and a check_receipts copy of the native receipt; record_journey_result needs delivery_id, input_digest, source_commit, preparation and manifest_digest. Strategist record_diagnosis needs subject, input_digest, source_commit and inventory_digest, and its reports carry stall_id/subject, inventory_digest, source_commit, preparation_digest and local_input_digest. Product Lead walkthroughs need producer, supported=true, intent, revision, the objective (up to 16000 chars, byte-equal), facts, excluded and exclusions, plus outcomes[0] and idea_ids for every alternative. Architect: the plan id is still model-minted, prerequisite_digests must be echoed, and generation is the constant 0. Also: consultation_group (32-hex), artifact upload_id plus offset, and artifact_refs digests echoed from publish responses.

**Simpler.** The authenticated capability plus the assignment becomes the only source of bindings. (1) submission_id becomes optional. Mint one when it is absent. When it is present, hash any string down to 32 hex. (2) Drop expected_input_digest and ErrMCPSubmissionStale, and always record capability.ScopeDigest(). (3) Generalize prod-hardening's bindIssuePlan into one bindPayload. It overwrites the review tuple and coverage, finding_digest, task output digests and child (keyed by task_id), app-verification delivery, input digest and receipt, journey and diagnosis bindings, and the copied walkthrough fields. Each value comes from the same canonical source the settler compares against. (4) Remove those properties from the advertised schemas and from the prompt guidance. Fix the checkpoint guidance that tells the Builder to send receipt_digest, which finish_implementation refuses.

**Risk.** Fill values from the same canonical source the settlers compare against, in one helper. During the transition, accept and overwrite echoed values so in-flight workers do not break.

**Visibility.** Tool schemas, prompt guidance and refusal texts lose the binding fields. Feed content and stored rows stay the same, because the same values are recorded, now server-filled. Calls that echo correct values keep working if overwrite-on-receive is used.

**Safe first step.** In internal/store/factory_mcp_submission.go (~line 259), delete the `capability.ScopeDigest() != submission.ExpectedInputDigest` clause from the in-transaction capability check. It repeats the admission check at internal/factory/control/mcp_admission.go:159 against the same immutable digest, so no outcome changes. Fixing the checkpoint guidance string at internal/web/api/factory_mcp_checkpoints.go:93 is a separate step: it is a model-contract change, though it only corrects text that contradicts what the server accepts.

**Related.** The prod-hardening branch (the requested name hardening/production-gaps does not exist) already adds bindIssuePlan in internal/web/api/factory_mcp*.go and UUID submission_id normalization. Any generalization should build on that helper rather than run parallel to it. Prompt guidance repeating the echo instructions also lives in internal/factory/dispatch_prompt.go (InvestigationPlanGuidance) and internal/factory/control/intent_bound.go.

### MCP-4: Every finish, task start and JEV consult re-checks the paid context-bundle selection and refuses on a hold

- **Subsystem:** Model-facing MCP tools. **Category:** too-many-states-or-budgets. **Verdict:** confirmed.
- **Removable:** ~150 lines. **Risk:** low. **Visible change:** UI, model tools. **Behaviour-preserving:** yes.
- **Status:** done (`72c5afe7`).
- **Where:** `internal/web/api/factory_mcp_context_gate.go:10-53`, `internal/web/api/factory_mcp.go:645`, `internal/web/api/factory_mcp.go:761-782`, `internal/web/api/factory_mcp.go:844-879`, `internal/web/api/factory_mcp.go:885-940`, `internal/web/api/factory_mcp_repair_scope.go:64-71`, `internal/web/api/factory_mcp_native_task.go:41`, `internal/web/api/factory_mcp_systemone.go:245`, `internal/factory/mcp_submission.go:37-40`, `internal/factory/control/context_bundle_advice.go:121-149`, `internal/factory/context_resolution.go:41-67`, `internal/factory/control/dispatch_launch.go:745-757`

**What.** requireNewSubmissionContext calls requireCurrentWork, which runs foregroundReadBundle on every call. That reloads the context-bundle-v1 JEV receipt and replays it through ContextBundleFromReceipt: it rebuilds the purpose request, re-fingerprints it and re-applies the 0.80 confidence thresholds. It then builds the full FactoryCurrentContext (a repeatable-read transaction, run twice when a resolution exists) and refuses unless availability is 'complete'. The gate covers all finish_* except incomplete, submit_task_output, publish_artifact, record_research, library mutations, start_task/release_output and every systemone call. The refusal tells the worker to give up: 'context selection holds new work; read factory.context and report incomplete or blocked'. What is being selected is tiny: attempt = run-history IDs; lineage = parent outcome and reason.

**Simpler.** MCP side: always serve the attempt and lineage sections when BundleEligibility says they exist. Delete foregroundReadBundle, requireCurrentWork/requireNewSubmissionContext and their calls in admitMutation, native_task and systemone. Delete the context_selection, bundle and lineage_incomplete body fields in serveRead, buildReadBody and the copied block in repair_scope, and ContextSelectionSnapshot/ContextCurrentResolution on MCPCapability. Liveness stays enforced by capability admission (CheckMCPWorkActive) and the deadline. Dispatch side (other finder): drop the context-bundle-v1 judgment and the launch hold.

**Risk.** The hidden sections only add bounded run-ID and parent-outcome data from the same actor, project and attempt (ContextParentVisible stays), so always serving them grants no new access.

**Visibility.** factory.context loses context_selection and bundle fields, and workers stop seeing context-hold refusals. Owner-facing context-hold states and the 'use current context' action disappear once the dispatch side is removed too.

**Safe first step.** Merge the copied context_selection/action block in internal/web/api/factory_mcp_repair_scope.go:64-71 and internal/web/api/factory_mcp.go:762-781 into one helper that serveRead and serveReassessmentScope both call. In requireCurrentWork (factory_mcp_context_gate.go), stop doing a second FactoryCurrentContext build when a resolution exists: foregroundReadBundle already computed it, so return it and reuse it rather than calling addCurrentContext again. Both keep the same outputs and refusals.

**Related.** Dispatch side: control.PrepareContextBundleSelection and the context_selection_hold wait in internal/factory/control/dispatch_launch.go:745-757. The owner 'use current context' resolution in internal/factory/control/context_resolution.go and frontend/spaces/sodaspaces-factory-context-resolution.ts. context_selection rendering in frontend/spaces/sodaspaces-factory-response.ts.

### BUD-2: Budget numbers live inside revisioned authority grants, so saving a higher allowance withdraws the repository's open publications and merges

- **Subsystem:** Budgets, allowances and limits (cross-cutting). **Category:** other. **Verdict:** confirmed.
- **Removable:** ~120 lines. **Risk:** medium. **Visible change:** UI. **Behaviour-preserving:** no.
- **Status:** owner decision; coordinate with prod-hardening.
- **Where:** `internal/store/factory_grant_commands.go:303-324`, `internal/store/factory_grant_commands.go:356-372`, `internal/factory/control/grants.go:81-131`, `internal/store/factory_publications.go:130-175`, `internal/store/factory_publications.go:534-603`, `internal/store/factory_merges.go:130-150`, `internal/store/factory_merges.go:484-530`, `internal/factory/control/publication_withdraw.go:55-76`, `internal/factory/control/publication_withdraw.go:190-239`, `internal/factory/control/merge_reconcile.go:54-62`, `internal/factory/grants.go:378-392`, `internal/factory/control/dispatch_selection.go:13-46`, `internal/factory/control/dispatch_context_recovery.go:70-84`, `internal/factory/readiness.go:355-375`, `internal/web/api/factory_sponsorship.go:24-69`, `internal/web/api/factory_sponsorship.go:101-108`, `frontend/spaces/sodaspaces-factory-settings.ts:899-930`, `frontend/spaces/sodaspaces-factory-settings.ts:1547-1585`

**What.** Allowance minutes, concurrency, attempt limits, intent/reassessment minutes and the rolling budget are fields of the same revisioned grant records that carry authorization (policy enabled/paused, operator active, sponsorship active, environment). Every save goes through the grant-command ledger, so changing a number is processed as an authority change.

**Simpler.** (1) In settleGrantWithdrawalTx, request publication and merge withdrawals only when withdraw is true, and in applyGrant cancel only when receipt.Withdrawn. That is about 5 lines and stops the cancellations. (2) Stop treating numbers as authority: AuthorityRef keeps identities and active flags, not revisions of rows whose numbers changed, and numeric limits are read live at admission (or move them out of the grant ledger into a plain settings row). (3) Saving a sponsorship keeps the stored grant when it is unchanged (default the select to it, or to the only grant) and drops the generation compare-and-swap.

**Risk.** The withdraw-only fix is low risk. Dropping revision binding changes the withdrawal design for real authority edits; disable and withdraw still close the dispatch gate, which is the actual fence.

**Visibility.** Saving a number no longer cancels publications or merges. The sponsorship form saves without re-picking the grant. Grant receipts set publications_pending/merges_pending only on withdrawals.

**Related.** The prod-hardening branch's new notice in frontend/spaces/sodaspaces-factory-allowance.ts:54 sends the owner to 'Raise Allowance minutes', which triggers this path. That branch also edits the same withdrawal SELECT in internal/store/factory_publications.go (jsonb_typeof guards only), so a fix would conflict with it textually. The revision pinning of AuthorityRef is also compared in factory_dispatch_packet.go ('grants changed during dispatch') and in readiness_work.go's AuthorityFingerprint.

### MCP-5: Two result channels (fenced v2 block and finish_* MCP tools) with different shapes, plus unread or unreachable report contracts in the prompts

- **Subsystem:** Model-facing MCP tools. **Category:** duplicated-layer. **Verdict:** partially-confirmed.
- **Removable:** ~120 lines. **Risk:** medium. **Visible change:** model tools. **Behaviour-preserving:** yes.
- **Status:** done (`27319fac`): the finish tools are the only result channel.
- **Where:** `internal/factory/control/dispatch_attempt.go:671-677`, `internal/factory/control/dispatch_attempt.go:752-785`, `internal/factory/control/intent_bound.go:95`, `internal/factory/control/intent_bound.go:126`, `internal/factory/dispatch_prompt.go:86-91`, `internal/factory/dispatch_prompt.go:396-441`, `internal/factory/dispatch_prompt_stage.go:142-155`, `internal/factory/review_native.go:218-221,820-827`, `internal/factory/publication_correction.go:81-85`, `internal/factory/result.go:579-600`, `internal/factory/control/dispatch_result.go:13-52,200-225,325-345,390-410`

**What.** Workers are told to 'Close your work with exactly one fenced block' holding a v2 PhaseResult, which the run-output path parses (ParsePhaseResult). The same workers also get finish_* MCP tools with different payload shapes. A parked MCP terminal blocks the fenced path, and the fenced path is used when no MCP terminal exists. Several prompt branches are dead or ask for blocks nothing reads.

**Simpler.** Keep the finish_* MCP tools as the only result channel. Replace each fenced '## Report' template with the role's finish tool plus finish_incomplete. Then delete the fence constants, fenceCollision/promptFenceGuard and the ParsePhaseResult settle paths. First, delete the unreachable and cut-off branches, the unused ClearReviewReportFence and the unread correction-json block.

**Risk.** Removing the run-output parser means a worker that never calls a finish tool ends with no report. Make the prompt say so clearly.

**Visibility.** Worker prompts change: one consistent finish instruction. Feed rendering of results is unchanged.

**Safe first step.** These deletions leave every prompt a worker receives unchanged: - internal/factory/dispatch_prompt_stage.go:142-155: delete writePromptTypedReviewContract. - internal/factory/dispatch_prompt.go writePromptReport: delete the `else if TypedReview/TypedClear` call (405-406) and the review-json else branch (407-413). Delete the v1 branch inside the correction case (420-422) and keep only the v2 line. Delete the final coding v1 else (430-435), because renderPhasedDispatchPrompt always replaces it. - internal/factory/review_native.go:820-827: delete the unused ClearReviewReportFence. Keep ReviewReportFence in the fenceCollision/promptFenceGuard replacers for now so prompt escaping stays byte-identical. Dropping the correction-json block (dispatch_prompt.go:423-426, plus the 'two fenced blocks' wording) is a small, intentional prompt change; leave it for a follow-up.

**Related.** ParsePhaseResult also settles reviewer and verification runs (dispatch_result.go:334, 399). mcp_fw40.go phaseResultFor builds a PhaseResult from MCP submissions, which is the other half of the duplicated channel.

### VRF-4: Test Pilot must hand-copy delivery bindings, receipts and canonical coverage the controller already knows, then is refused unless they match exactly

- **Subsystem:** Test Pilot / app verification (Go factory+control+store+MCP, Rust host, in-project helper). **Category:** unneeded-validation-or-digest. **Verdict:** confirmed.
- **Removable:** ~120 lines. **Risk:** low. **Visible change:** model tools. **Behaviour-preserving:** no.
- **Status:** done (`291451f6`).
- **Where:** `internal/web/api/factory_mcp.go:333-336`, `internal/web/api/factory_mcp.go:523`, `internal/web/api/factory_mcp_role_schema.go:82-88`, `internal/web/api/factory_mcp_verification.go:10-33`, `internal/factory/control/mcp_fw40.go:545-603`, `internal/factory/control/app_verification.go:101`, `internal/factory/control/app_verification.go:109-147`, `internal/store/factory_app_verification.go:226-271`, `internal/store/factory_verification_manifest.go:78-85`, `internal/factory/verification_journey.go:23-38`, `internal/factory/verification_journey.go:126-147`, `internal/factory/verification.go:319-351`, `internal/factory/control/delivery_admission.go:82`

**What.** This is the same model-hostile pattern as finding 1 of the demo evidence. finish_app_verification requires delivery_id and input_digest, plus check_receipts with check_id 'project.check', outcome 'passed', and artifact_digest equal to the native output digest. Its coverage entries must list task_keys and delivered_acceptance_ids that exactly equal the canonical delivered tasks. record_journey_result requires delivery_id, input_digest, source_commit, preparation and manifest_digest. The controller then compares each copied value with its own records.

**Simpler.** The controller fills delivery_id, input_digest, the single native check receipt and the canonical coverage when it builds the AppVerificationReport, and the five journey bindings when it records a journey. Test Pilot submits only outcome, failures (for failed), outcome_ids/steps/observations for journeys, and evidence refs. Delete the receipt and coverage equality checks in verifyApplicationReport, RecordVerifiedApplication and checkVerifiedManifestExecutionTx (they become tautological), delete the journey binding comparisons, and drop the copy guidance from the prompt and the evidence tool. Note that the finish's input_digest is already only format-checked and never compared, so it is pure overhead.

**Risk.** The values come from the server's own records, so filling them server-side is strictly more correct.

**Visibility.** The finish_app_verification and record_journey_result input schemas, descriptions and the Test Pilot prompt shrink.

**Related.** The issue-plan input_digest copy pattern (internal/web/api/factory_mcp_plan_schema.go:22, 'Copy prompt_sha from factory.context') was addressed on the hardening branch. Similar copied input_digest and subject binding checks exist for delivery repair (internal/factory/repair_draft.go:126-140, internal/factory/control/repair_preview.go:118), repair diagnosis (internal/factory/repair_diagnosis.go:95) and finish_reassessment's input_digest (factory_mcp.go:525).

### BUD-5: JEV 'funding' journal: a global ceiling that is never checked, and one-shot reservations that make every failed judge call permanent

- **Subsystem:** Budgets, allowances and limits (cross-cutting). **Category:** too-many-states-or-budgets. **Verdict:** confirmed.
- **Removable:** ~110 lines. **Risk:** low. **Visible change:** UI, model tools, DB schema. **Behaviour-preserving:** yes.
- **Status:** done (`c62f6ec1`).
- **Where:** `internal/store/schema.go:35-37`, `cmd/soda-identity/src/schema.rs:30-32`, `internal/store/factory_intents.go:267-325`, `internal/store/factory_intents.go:383-404`, `internal/store/factory_context_selection.go:12-58`, `internal/store/factory_attempt_allowances.go:1228-1246`, `internal/store/factory_reassessments.go:18-26`, `internal/store/factory_reassessments.go:579-674`, `internal/store/factory_delivery_repair_adoption.go:60-90`, `internal/store/factory_feed_comment.go:430-443`, `internal/factory/control/judgment_advice.go:49-116`, `internal/factory/control/judgment_advice_delivery.go:62-109`, `internal/factory/control/context_bundle_prepare.go:12-68`, `internal/factory/control/completion_decision.go:159-162`, `internal/factory/control/completion_decision.go:214-230`, `internal/factory/control/completion_decision.go:270-280`, `internal/factory/control/completion_drive.go:420-430`, `internal/web/api/factory_mcp_systemone.go:242-275`

**What.** Every JEV call is treated as a paid purchase. A fingerprint row is reserved before the call, a global spent counter is bumped under a table lock, and the reservation is kept whatever happens, so a question that once failed is never asked again.

**Simpler.** Drop factory_judgment_budget (Go schema and the Rust mirror) and every SELECT ... FOR UPDATE / UPDATE of `spent`. Drop the max_calls column and parameter. Delete ErrReassessmentBudgetExhausted, errReassessmentExhausted and the funding_exhausted branches. Keep factory_priority_judgments only as a cache of successful answers keyed by fingerprint, and stop recording failure receipts (or delete the reservation row on failure), so the next progress pass asks again. Clear the capability's context_selection_snapshot stamp when its call fails. The SystemOne tool retries the same way, so its 'reservation retained' refusal goes. The planning-call and delivery-call allowances go with BUD-7 and BUD-6.

**Risk.** A single coordinator owns the database (file lock), and the fingerprint primary key already prevents duplicate rows, so the global lock is not needed for concurrency. Retrying once per progress tick avoids hammering the judge.

**Visibility.** Holds such as context_selection_hold and 'JEV gave no usable answer' clear on the next pass instead of waiting for a manual Retry. The SystemOne tool's refusal text changes. The factory_judgment_budget table and the max_calls column go, in the Rust schema mirror too. Removing the global counter alone changes no behaviour.

**Safe first step.** These deletions leave every outcome the same, because nothing reads the counter and the error is never returned: - In factory_intents.go (ReservePriorityJudgment, ReserveIntentJudgment), factory_context_selection.go, factory_attempt_allowances.go (around 1232-1245), factory_reassessments.go (ReserveReassessmentCall) and factory_delivery_repair_adoption.go, delete the `var spent int`, the `SELECT spent FROM factory_judgment_budget ... FOR UPDATE` and the `UPDATE factory_judgment_budget SET spent=...`. - Delete ErrReassessmentBudgetExhausted (factory_reassessments.go:22-26) and errReassessmentExhausted with its branches (completion_decision.go:159-162, 217-219, 276-279; completion_drive.go:425-428). - Fix the false 'global ceiling' doc comments. - Delete or adjust the test assertions on `spent`. Dropping the table and the column from schema.go/schema.rs is a separate step because it changes the schema. Changing failure retention is a separate step because users see it.

**Related.** The same "reserve before the call, keep the reservation forever" pattern is behind the per-intent planning allowance (factory_intent_planning_calls, ErrPlanningAllowanceExhausted, which is BUD-7), the per-attempt delivery subledger and the repair judgment charge (factory_attempt_allowances.go, admitRepairChargeTx, which is BUD-6), and the reassessment cycle ledger (MaxReassessmentDecisionCalls). The priority tournament's up-front reservation of max_calls (up to 18) is in control priority selection.

### BND-2: Issue plan makes the Architect echo six server-known bindings, a plan id and a Go-encoding SHA-256, plus a duplicated approach; with prod-hardening the server re-checks what it filled itself

- **Subsystem:** Boundary machinery: strict JSON, digests, receipts, revision checks (cross-cutting). **Category:** unneeded-validation-or-digest. **Verdict:** partially-confirmed.
- **Removable:** ~100 lines. **Risk:** medium. **Visible change:** model tools. **Behaviour-preserving:** yes.
- **Status:** owner decision; coordinate with prod-hardening.
- **Where:** `internal/factory/dispatch_prompt.go:482`, `internal/web/api/factory_mcp_plan_schema.go:17-24`, `internal/factory/planning.go:55-125`, `internal/web/api/factory_mcp_issue_plan.go:19-70`, `internal/web/api/factory_mcp.go:314`, `internal/web/api/factory_mcp_role_schema.go:55`, `internal/factory/control/mcp_fw40.go:329`, `internal/factory/control/correction_plan.go:154-158`, `prod-hardening:internal/web/api/factory_mcp_issue_plan.go (bindIssuePlan)`

**What.** InvestigationPlan contains ID, AttemptRoot, ScopeDigest, InputDigest, SourceCommit, IntegrationCommit, Preparation and Digest. Digest is SHA-256 over Go encoding/json of {Version, Plan}. InvestigationPlanGuidance tells the Architect to copy these values from factory.context and to work out the Go-encoding digest itself. The finish tool also needs 'approach' both at top level and inside tasks, and refuses unless the two match (issue_plan.go:64-67). The hardening branch adds bindIssuePlan, which overwrites these fields from the assignment and computes the digest before validation. Validate then recomputes PlanDigest and compares it with the digest the server just wrote. The attempt_root/source_commit/preparation compare then checks values the server just copied in. Plan.Validate(), including the digest recompute, also runs again on every read of the stored plan, in five store and api paths. Downstream, only plan.ScopeDigest (delivery_producer.go:59, delivery_judgment_producer.go:160) and plan.Digest are used, and only as Feed evidence and plan-revision keys.

**Simpler.** Remove these from the model-facing issue-plan schema and guidance: attempt_root, scope_digest, input_digest, source_commit, integration_commit, preparation and digest, and optionally id. Also remove the duplicated top-level approach. Have one server function build the stored InvestigationPlan from the payload plus the assignment, set the generated id and the digest, and then call Validate once. Do not add this alongside bindIssuePlan; replace bindIssuePlan with it. Keep these fields in the stored plan: AttemptRoot, ID, SourceCommit, Preparation, InputDigest, IntegrationCommit and ScopeDigest. They are used downstream as keys, checkpoint bases, gate inputs and context comparisons. Optionally stop re-running Validate on stored-plan reads. There are about 16 call sites, not 5.

**Risk.** factory_plan_revisions keys on (attempt_root, plan_id) and stores digest. The server keeps filling them, so no schema change is needed. Stored-plan reads would stop rejecting rows the system wrote itself.

**Visibility.** Architect schema, guidance prompt and refusal texts all shrink. The Feed plan card still shows the plan digest, which the server computes.

**Safe first step.** Only once prod-hardening's bindIssuePlan is merged: in internal/web/api/factory_mcp_issue_plan.go validateIssuePlanSubmission, delete the attempt_root/source_commit/preparation compare (lines 61-63). After bindIssuePlan it can never fire, because a plan that fails to decode is already refused by the strictjson.Decode at line 48. Keep the refusal-text hint unchanged so the model contract stays the same. On main as it stands, every part of this changes the Architect contract, so nothing is purely behaviour-preserving there.

**Related.** The same pattern of echoing server-known bindings and re-checking them appears in Product Lead product-plan bindings (factory_mcp_product_plan.go, where prod-hardening adds checkProductPlanSemantics) and in mcp_fw40.go:329-332, which re-checks approach, attempt_root and source_commit at settlement. Stored plans are re-validated in completion_material.go:1312, factory_planning_revisions.go:267/363, dispatch_attempt.go:560/721, dispatch_launch.go:126, task_execute.go:169, publication_plan_gate.go:86, delivery_producer.go:46 and delivery_judgment_material.go:134.

### BND-3: Inspector and Test Pilot finishes make the model echo server-known values (tuple, coverage digest, delivery_id, input_digest), and a finish that leaves them out silently parks and freezes

- **Subsystem:** Boundary machinery: strict JSON, digests, receipts, revision checks (cross-cutting). **Category:** unneeded-validation-or-digest. **Verdict:** confirmed.
- **Removable:** ~100 lines. **Risk:** medium. **Visible change:** model tools. **Behaviour-preserving:** no.
- **Status:** done (`291451f6`).
- **Where:** `internal/factory/control/mcp_fw40.go:36-44`, `internal/factory/control/mcp_fw40.go:350-441`, `internal/factory/control/mcp_fw40.go:545-608`, `internal/web/api/factory_mcp.go:319-336`, `internal/web/api/factory_mcp.go:516-523`, `internal/web/api/factory_mcp_role_schema.go:40`, `internal/web/api/factory_mcp_role_schema.go:62-63`, `internal/web/api/factory_mcp_role_schema.go:83`, `internal/factory/review_native.go:931-946`, `internal/factory/context_review.go:17`, `internal/store/factory_context_review.go:69`, `internal/factory/control/review_cycle.go:372-386`, `internal/factory/control/delivery_acceptance.go:150-172`, `internal/web/api/factory_mcp_verification.go:32`, `internal/store/factory_verification_journey.go:127-134`, `internal/factory/control/mcp_admission.go:138-160`

**What.** Review finish (Inspector): a clear or findings verdict settles only if the payload contains tuple{cycle_id, head, base_oid} and, for clear, 'coverage', which must equal ClearCoverageDigest(head, base, contentDigest). The store computes that digest into the model's context (factory_context_review.go:69). The model copies it back, and the server recomputes and compares it in review_cycle.go and twice in delivery_acceptance.go. The cycle, head and base are already recorded on the review assignment. App-verify finish (Test Pilot): it needs delivery_id plus input_digest. The input_digest is the capability ScopeDigest the server handed out at factory_mcp_verification.go:32. When any of these echo fields is missing or malformed, mapFW40Terminal returns parkFW40(). The submission is recorded and frozen as the terminal proposal, the receipt says accepted/recorded, and nothing settles. The assignment then waits until its deadline. This is the same failure as the Architect's, in two more roles, and it is invisible to the model.

**Simpler.** Fill cycle_id, head and base_oid from the assignment's recorded review cycle, and delivery_id from assigned.Subject.Delivery. input_digest is already carried and checked as the submission's ExpectedInputDigest. Remove these fields from the role schemas. Drop the model-echoed coverage digest. If a binding to the carried set is still wanted, compute ClearCoverageDigest once on the server when the clear verdict is recorded, not in the context. For anything the model must still supply (findings, check_receipts), refuse synchronously at the tool call before the terminal freezes, as CheckProductPlanFinish now does for the Product Lead on prod-hardening, instead of parking.

**Risk.** Review clear-coverage exists so a clear verdict covers the exact carried finding set. Computing it on the server from the same recorded cycle keeps that meaning. The settler has to read the assignment's review context, which it already loads elsewhere.

**Visibility.** Inspector and Test Pilot schemas lose the echo fields. Parked finishes become explicit refusals.

**Safe first step.** none. The only identical-behaviour piece is the repeated coverage-validity check in settleReviewVerdict (mcp_fw40.go:402-406), which is unreachable after the same check at 381-385. Deleting it is trivial, not meaningful. Every substantive part changes the Inspector and Test Pilot tool contract.

**Related.** The same parked-and-frozen pattern exists for the Strategist (settleReassessmentFinish parks without a 'report' object, mcp_fw40.go:443-447) and for non-code assessments without a 'deliverable' (mcp_fw40.go:399ish/settleNonCodeAssessment). The Correction finish requires the model to echo finding_digest (factory_mcp.go:323-326). The Architect/Product Lead version is partly handled on prod-hardening by CheckProductPlanFinish. Every submission's ExpectedInputDigest already carries the scope digest, which makes payload input_digest echoes redundant (also on finish_reassessment, factory_mcp.go:525).

### BUD-6: Per-attempt JEV delivery-call budget rations a free judge, and a policy first saved from Settings gets 0 calls, so it can never auto-merge

- **Subsystem:** Budgets, allowances and limits (cross-cutting). **Category:** too-many-states-or-budgets. **Verdict:** confirmed.
- **Removable:** ~90 lines. **Risk:** low. **Visible change:** UI, model tools. **Behaviour-preserving:** no.
- **Status:** done (`c62f6ec1`).
- **Where:** `internal/factory/allowance.go:16-32`, `internal/factory/allowance.go:57-76`, `internal/factory/allowance.go:130-137`, `internal/factory/allowance.go:258-272`, `internal/factory/allowance.go:377-409`, `internal/store/factory_attempt_allowances.go:609`, `internal/store/factory_attempt_allowances.go:1189-1247`, `internal/factory/control/judgment_advice_delivery.go:26-49`, `internal/factory/control/delivery_judgment_producer.go:146-152`, `internal/web/api/factory_policy.go:73`, `frontend/spaces/sodaspaces-factory-settings.ts:1015-1028`, `internal/store/factory_context_current.go:155-159`, `internal/factory/context_current.go:56`

**What.** DeliveryCalls is a per-attempt count of funded JEV delivery judgments, kept as a second ledger beside the fingerprint journal. Zero is a deliberate 'unfunded, never auto-merge' value, and there is no control for it.

**Simpler.** Delete DeliveryCalls: the limit, the ledger and its validation, ReserveDeliveryCall, ErrDeliveryExhausted, the attempt half of ReserveDeliveryJudgment (keep the fingerprint journal row and the global judgment counter) and the DeliveryCallsRemaining context field. The review/correction loop is already bounded by correction cycles and review rounds.

**Risk.** Judge calls are free. Each call is still tied to a distinct evidence fingerprint, and the loop that produces new evidence is bounded by correction cycles.

**Visibility.** Repositories whose policy was created in Settings start merging automatically. factory.context drops delivery_calls_remaining.

**Related.** Intent minutes use the same pattern, where zero is kept as an explicit "unfunded" posture (factory_policy.go IntentMinutes, settings.ts "Zero leaves intent planning unfunded"). The global factory_judgment_budget counter is charged on the same path in factory_intents.go and factory_delivery_repair_adoption.go.

### IDN-1: Lease acquisition pins (and may refresh/download) a provider bundle that no lease ever runs

- **Subsystem:** Identity service and provider credentials. **Category:** other. **Verdict:** confirmed.
- **Removable:** ~90 lines. **Risk:** low. **Visible change:** none. **Behaviour-preserving:** yes.
- **Status:** open (behaviour-preserving).
- **Where:** `cmd/soda-identity/src/acquisition.rs:21-42`, `cmd/soda-identity/src/control.rs:16-22`, `cmd/soda-identity/src/control.rs:76-92`, `cmd/soda-identity/src/control.rs:103-115`, `cmd/soda-identity/src/control.rs:150-178`, `cmd/soda-identity/src/registration.rs:47-86`, `cmd/soda-identity/src/providers/codex/mod.rs:28-36`, `cmd/soda-identity/src/providers/muse/mod.rs:41-49`, `cmd/soda-identity/src/providers/resolution.rs:83-116`, `cmd/soda-identity/src/providers/resolution.rs:161-191`, `cmd/soda-identity/src/main.rs:36-51`, `lib/provider-tools/src/config.rs:17`, `docs/development/provider-freshness-contract.md:56-64`, `cmd/soda-identity/tests/http.rs:354`, `cmd/soda-identity/tests/common/mod.rs:225`, `cmd/soda-identity/tests/provider_store.rs:193-252`

**What.** Every acquire (and every replayed acquire) calls pin_lease -> Provider::bind_new -> resolution::resolve_new, which reads the broker's private provider store, runs an upstream release check (and possibly download + probe) whenever the 300 s TTL is due, refuses when the selection is Stale/Unknown, and stores a RefToken in lease_refs until the lease is forgotten. The bound executable is never used for a lease: soda-host runs leases from its own root-owned store with its own resolve_current/bind_recorded freshness gate. All of this runs inside the broker's single controller Mutex and after reserve_and_link_execution has already committed the lease, so a stale or failed refresh returns 500 'unavailable' (host: 'identity operation failed') while leaving the reserved lease (for Codex, the connection's exclusive slot) behind. Replays overwrite the lease_refs entry without releasing the old token (RefToken has no Drop; refs are files under <root>/refs), and a broker restart empties the in-memory map, so ref files leak.

**Simpler.** Delete pin_lease, unpin_lease and State.lease_refs (a Mutex nested inside the controller Mutex). Remove Provider::bind_new from the trait, along with both wrappers and both provider implementations. Remove the call sites in acquire, observe_terminal and release_execution_lease, and the matching test stubs. Keep resolve_new only in enrollment session creation (session_binary), where the broker really spawns the CLI. Optional follow-up: let enrollment read the host's provider store read-only instead of keeping a second private store with its own refresh.

**Risk.** Intended behaviour change: acquisition stops depending on the broker's enrollment-binary freshness and on network access; new runs stay gated by the host's own provider freshness. Enrollment still binds a fresh bundle per session.

**Visibility.** No UI or API shape change; removes a source of opaque 'identity operation failed' acquisition refusals (identical behaviour on the success path).

**Safe first step.** This step keeps behaviour identical. In cmd/soda-identity/src/control.rs, delete the State.lease_refs field and its initialiser. Change pin_lease to call provider.bind_new()? and, if it returns Some(mut b), call resolution::release_bound(&mut b) straight away; the freshness gate stays. Delete unpin_lease and its two calls in cmd/soda-identity/src/registration.rs (observe_terminal, release_execution_lease). This removes the ref-file leak and the nested Mutex. The next step, which does change behaviour, deletes pin_lease, bind_new (trait, wrappers, codex/muse) and the test stubs entirely.

**Related.** Enrollment keeps its own copy of this logic: session_binary in codex/mod.rs:76-84 and muse/mod.rs call resolve_new against the broker's second private provider store (/home/soda-identity/providers), which runs its own upstream refresh in parallel with the host's /var/lib/soda/providers store. resolution.rs fresh_selection duplicates soda_provider_tools::resolve::resolve_new (lib/provider-tools/src/resolve.rs:20+).

### DLV-1: Every native op binds the instance-wide native revision into a one-shot immutable intent, so an unrelated Forgejo write before submit becomes a terminal refusal that closes the attempt

- **Subsystem:** Publication, merge, correction, repair, acceptance, verification, lifecycle. **Category:** unneeded-validation-or-digest. **Verdict:** confirmed.
- **Removable:** ~80 lines. **Risk:** medium. **Visible change:** none. **Behaviour-preserving:** no.
- **Status:** coordinate with prod-hardening.
- **Where:** `internal/factory/control/publication_reconcile.go:255-308`, `internal/factory/control/publication_reconcile.go:377-395`, `internal/factory/control/merge_evidence.go:50-62`, `internal/factory/control/merge_evidence.go:113-126`, `internal/factory/control/merge_reconcile.go:141-163`, `internal/factory/control/merge_reconcile.go:272-283`, `internal/factory/control/correction.go:185-211`, `internal/factory/control/correction.go:317-363`, `internal/factory/publication.go:33`, `internal/factory/publication.go:136,239`, `internal/factory/merge.go:42,127,193`, `internal/factory/merge_operation.go:171-273,430-471`, `internal/store/factory_publications.go:186-191`, `internal/store/factory_merges.go:169-175,414`, `../forgejo-ext/models/nativeoperation/store.go:191-212,254-255,334-335,513-529`, `../forgejo-ext/services/nativeoperation/guard.go:151-200`, `../forgejo-ext/services/nativeoperation/merge.go:108-117`, `../forgejo-ext/services/nativeoperation/publish.go:243-244`

**What.** Branch publish, PR create, correction push and merge each persist an immutable intent before submit. The intent includes ExpectedNativeRevision, taken from a bracketed read of Forgejo's global, instance-wide revision counter, and each op has exactly one attempt (MaxPublicationAttempts = MaxMergeAttempts = 1). The native side compares that number with the global counter, which every ordinary writer increments. On a mismatch it records the op as not_committed with reason stale_native_revision. Soda treats not_committed as final: the publication fails as native_refused and closes the attempt root (closeRoot); the merge fails as native_refused and closes the root, and no replacement path is reachable (see DLV-3); a correction waits on 'correction_refused' forever, because the same candidate can never open a new op identity (correction.go:190-201). Separately, Soda waits 'native_inputs_changed' whenever the revision from the acceptance-status read differs from the op observation made later in the same pass.

**Simpler.** Stop binding the global revision for ops that already carry exact OID compare-and-swap conditions. Either the native side treats ExpectedNativeRevision=0 as unbound, or Soda reads the revision immediately before submit and does not persist it. Remove the three native_inputs_changed equality waits. Treat a confirmed no-effect refusal (not_committed with stale_native_revision or expired_before_admission) as retryable under a new op identity, instead of failing the publication or merge and closing the root. This also removes the NativeRev/ObservedUnix plumbing and its validation from PublicationIntent, MergeIntent, Publication and Merge.

**Risk.** Needs either a small native-side change or a Soda retry rule that mints a new op identity only after a confirmed no-effect outcome. The per-op OID conditions keep double effects impossible.

**Visibility.** The UI keeps its shape. Feed and Work would show fewer terminal 'Publication failed' and 'native_refused' results and fewer closed attempts.

**Related.** The same ExpectedNativeRevision binding and the same stale_native_revision refusal apply to the other conditional native ops in ../forgejo-ext/services/nativeoperation: issuegraph.go:74-88, lifecycle_issue.go:115-129, lifecycle_review.go:118-132, lifecycle_note.go:120, review.go:248 and prcreate.go:156-170. Their Soda-side callers likely have the same one-shot terminal handling. The merge guard ObservedNativeRev check is at internal/factory/merge_operation.go:430-471.

### CTX-3: New finish/submit, SystemOne and native-task calls are refused unless CurrentContext is 'complete' (19 missing-reason codes) and the bundle selection is not on hold

- **Subsystem:** Worker context assembly, project library and research. **Category:** unneeded-validation-or-digest. **Verdict:** partially-confirmed.
- **Removable:** ~60 lines. **Risk:** medium. **Visible change:** model tools. **Behaviour-preserving:** no.
- **Status:** owner decision; coordinate with prod-hardening.
- **Where:** `internal/web/api/factory_mcp_context_gate.go:10-53`, `internal/web/api/factory_mcp.go:645`, `internal/web/api/factory_mcp.go:846-879`, `internal/web/api/factory_mcp_systemone.go:245`, `internal/web/api/factory_mcp_native_task.go:41`, `internal/store/factory_context_current.go:58-129`, `internal/web/api/factory_mcp_context.go:64-69,88-97`, `internal/web/api/factory_mcp_bound_test.go`

**What.** requireCurrentWork rebuilds the whole CurrentContext in a REPEATABLE READ transaction: capability, assignment, run, allowance, every submission receipt, acceptance plus heads/withdrawals, plan, task receipts, plan head, review/publication facts, CI assessment. It also re-runs the context-bundle selection lookup. It then refuses with 'required current context holds new work; read factory.context and report incomplete or blocked' unless Evidence.Availability == "complete". It runs before every new finish/submit tool, every SystemOne consultation and every native task tool.

**Simpler.** Delete requireCurrentWork and requireNewSubmissionContext (the whole factory_mcp_context_gate.go file) and their three call sites. Keep factory.context's required-current projection and its hold hint as information for the worker. Admission already validates the capability, the submission store validates assignment stage and terminal state, and review settlement re-verifies head and base. If a specific input is truly needed to finish (for example the plan head for a Builder), check that one input in that tool's validator, with a refusal that names it. If the owner wants to keep a guard before paid SystemOne calls, keep a narrow check of the one input the consultation needs.

**Risk.** Removes a blanket precondition. A worker could then submit while, say, the allowance row is missing. The submission store and settlement still validate their own inputs, so the result would be a specific refusal later instead of a generic one now.

**Visibility.** The 'required current context holds new work' refusal text disappears from MCP responses.

**Related.** foregroundReadBundle (internal/web/api/factory_mcp.go:846-879) applies the same Availability=="complete" rule a second time on the ContextCurrentResolution path and rebuilds FactoryCurrentContext. markRequiredContextHold (factory_mcp_context.go:94-97) and control/context_bundle_state.go:97 carry the same complete-or-hold logic into context_selection. Review settlement (control/review_cycle.go:488,513,767) already re-verifies the candidate head and base that review_candidate_inputs_changed protects.

### DLV-2: The delivery tail runs on the coder's attempt clock: every native op, review/correction child and merge is capped by the issue's active-minute allowance, which keeps ticking while waiting on CI and reviews, and expiry withdraws the merge

- **Subsystem:** Publication, merge, correction, repair, acceptance, verification, lifecycle. **Category:** too-many-states-or-budgets. **Verdict:** confirmed.
- **Removable:** ~60 lines. **Risk:** medium. **Visible change:** none. **Behaviour-preserving:** no.
- **Status:** done (`b1d4c63e`).
- **Where:** `internal/factory/control/publication_reconcile.go:272-277`, `internal/factory/control/correction.go:326-331`, `internal/factory/control/merge_evidence.go:113-119`, `internal/factory/control/delivery_producer.go:111-114`, `internal/factory/control/merge_reconcile.go:42-53`, `internal/factory/control/publication_children.go:58-107`, `internal/factory/control/publication_children.go:617-624`, `internal/store/factory_publications.go:109-128`, `internal/store/factory_merges.go:109-117`, `internal/store/factory_attempt_allowances.go:51-91,99-152,153-175,423-455`, `internal/store/factory.go:215-243,381-402`, `docs/development/factory-workflow-contracts.md:274-276`

**What.** An issue attempt's allowance (default 120 active minutes, allowance.go:36) is meant to cover worker time. After the coder finishes, the allowance stays Active until the publication or merge reaches a terminal state, so RemainingSeconds counts wall-clock time spent waiting on CI, up to 6 review rounds, JEV judgments and 3 corrections. Every native op registration must fit inside that budget: CapAttemptDeadline at observe time, and ensureAttemptDeadlineForRegistrationTx inside the store write. The child producer stops with 'attempt is closed; maintainer intervention is required'. admitMergeRow calls CloseExpiredAttempt and withdraws the merge, which is terminal (see DLV-3).

**Simpler.** Give native ops a fixed NotAfter of now+10m. Delete CapAttemptDeadline and ensureAttemptDeadlineForRegistrationTx along with their call sites, because these require the allowance to be Active. Drop the CloseExpiredAttempt/withdrawMerge expiry branch from admitMergeRow, since a merge spends no worker time. After that, checkpoint the attempt allowance inactive when the coder run settles and active again only while a review or correction child is live. That also means dropping the !Active early returns in the child gate, or reactivating at child launch. Update contracts doc line 274 to match.

**Risk.** This changes what the allowance means. Run accounting already checkpoints at launch and settle, so the change is local to the settle transition and the deadline caps.

**Visibility.** Attempts would stop closing, and merges stop being withdrawn, during long CI or review waits. No UI shape change.

**Related.** Typed allowances use the same shared joint, transitionLockedAttemptRootTx, and stay Active until their recorded effects settle: completion (settleTypedCompletionRootTx), delivery, intent and stall roots. The delivery-repair and verification allowances (factory_delivery_repair_budget.go, factory_verification_allowance.go) use the same Active/Checkpoint pattern. On prod-hardening, production-gaps.md describes the same 120-minute pressure only for sponsorship and dispatch.

### CTX-7: Inspector must copy server-computed tuple and clear_coverage_digest back into finish_review (same failure class as demo item 1)

- **Subsystem:** Worker context assembly, project library and research. **Category:** unneeded-validation-or-digest. **Verdict:** confirmed.
- **Removable:** ~22 lines. **Risk:** low. **Visible change:** model tools. **Behaviour-preserving:** no.
- **Status:** done (`291451f6`).
- **Where:** `internal/factory/dispatch_prompt_stage.go:137-138`, `internal/store/factory_context_review.go:63-69`, `internal/factory/context_review.go:17`, `internal/web/api/factory_mcp.go:319-321`, `internal/web/api/factory_mcp_role_schema.go:61-63`, `internal/factory/control/mcp_fw40.go:377-411`, `internal/factory/control/delivery_acceptance.go:151-172`

**What.** currentReviewTx computes ClearCoverageDigest = digest(head, base_oid, FindingContentDigest(carried findings)) and exposes it, together with cycle/head/base, inside the fragment-paged role. The prompt then requires the Inspector to 'Supply tuple {cycle_id,head,base_oid} from review.cycle, coverage equal to review.clear_coverage_digest'. The server already holds every one of these values for this assignment (a.ReviewContext.Cycle).

**Simpler.** Have finish_review fill tuple and coverage from the assignment's recorded ReviewContext (cycle, plus the coverage digest computed from the carried findings), the way prod-hardening's bindIssuePlan does for the issue plan. Drop clear_coverage_digest from the context, the prompt and the required schema keys, and drop tuple from the schema. checkClearCoverage stays as the server-side check. The model supplies only conclusion, findings, carried_assessments and evidence_refs.

**Risk.** Changes the finish_review schema and description. Validation stays server-side against the same recorded values.

**Visibility.** The finish_review input schema and Inspector prompt text change.

**Related.** The same hand-copy pattern appears in the finish_correction finding_digest (required hex key, factory_mcp.go mcpMutationPayloads). It also appears in the submit_task_output and request_handoff manifest_digest fields, and in input_digest on several finish tools (factory_mcp_role_schema.go). The legacy fenced phase-result review contract (writePromptTypedReviewContract) also asks for the coverage digest.

## Ranked findings: medium impact

### TST-5: About 12.5k lines of Forgejo presentation inventory, contract JSON and gallery templates that nothing reads

Test harnesses and source guards. dead-code. confirmed. ~12535 lines, risk low. Visible change: none. Behaviour-preserving: yes. **Status:** open (behaviour-preserving).

Where: `tests/forgejo/presentation/inventory.json:1-10013`, `tests/forgejo/presentation/form-native-contracts.json:1-750`, `tests/forgejo/presentation/settings-native-contracts.json:1-611`, `tests/forgejo/presentation/repository-settings-native-contracts.json:1-886`, `tests/forgejo/presentation/form-presentation-deltas.json:1-174`, `tests/forgejo/milestone-page-boundaries.json:1-4`, and 3 more.

**Simpler.** Delete all of these files, plus the unused settings-contracts.ts, and the README sentences that describe them.

**Risk.** Nothing reads these files, so deleting them cannot change behaviour.

**Safe first step.** Delete these files, which no code reads: - tests/forgejo/presentation/inventory.json - form-native-contracts.json - settings-native-contracts.json - repository-settings-native-contracts.json - form-presentation-deltas.json - gallery.tmpl, refinement-gallery.tmpl, repository-settings-gallery.tmpl - settings-contracts.ts - tests/forgejo/milestone-page-boundaries.json Then remove the sentences in frontend/forgejo/README.md that describe them: lines 437-438, 496-500, 572-577 and 715-720.

### TST-10: Dead installed journeys, plus unit tests of the harness's own HTTP/matrix parsers (and their test-only modules); the Workspace suites go with the Workspace removal

Test harnesses and source guards. dead-code. partially-confirmed. ~8100 lines, risk low. Visible change: none. Behaviour-preserving: yes. **Status:** deferred to repo-first-ui (Workspace removal).

Where: `tests/installed/sodaspaces-first-use-journey.ts:1-713`, `tests/installed/sodaspaces-workspace-journey.ts:1-293`, `tests/frontend/workspace-journey.test.ts:1-226`, `tests/frontend/sodaspaces-http.test.ts:1-109`, `tests/installed/sodaspaces-http.ts:1-85 (only imported by its own test)`, `tests/installed/sodaspaces-cli.ts:1-199 (only imported by workspace-journey.test.ts)`, and 3 more.

**Simpler.** Now: delete the two unreferenced installed journeys, workspace-journey.test.ts, sodaspaces-http.test.ts, sodaspaces-http.ts (it has no other users, so nothing replaces it) and sodaspaces-cli.ts. Later, in the same change that removes the Workspace frontend: delete the Workspace suites and fixtures, move any terminal/setup/intent coverage that still matters to repository-page tests, and drop the workspace files from test:lit (update tests/build/terminal_assets_test.go if test:lit is removed entirely).

**Risk.** The Workspace suites are tied to the Workspace removal; the dead journeys and parser unit tests can go now. Check which Workspace tests actually cover features that move to the repository page, such as terminals or setup, before deleting them.

**Safe first step.** Delete tests/installed/sodaspaces-first-use-journey.ts, tests/installed/sodaspaces-workspace-journey.ts, tests/frontend/workspace-journey.test.ts, tests/frontend/sodaspaces-http.test.ts, tests/installed/sodaspaces-http.ts and tests/installed/sodaspaces-cli.ts (about 1,625 lines). None of them is imported by anything that remains, and no script runs them.

### TST-8: About 5-6k of ~10.5k lines in the budget-named Go/TS test suites pin the overlapping budgets; delete those parts with the budget collapse and keep the non-budget cases

Test harnesses and source guards. too-many-states-or-budgets. partially-confirmed. ~5500 lines, risk medium. Visible change: UI, DB schema. Behaviour-preserving: no. **Status:** owner decision; coordinate with prod-hardening.

Where: `internal/factory/control/dispatch_timeonly_test.go (time-only root charge/unfunded/replay tests; not the plan-head/preparation refusals)`, `internal/factory/control/dispatch_accounting_test.go (usage/release accounting tests)`, `internal/factory/control/dispatch_result_timeonly_test.go:1-533`, `internal/factory/control/dispatch_expiry_test.go:1-274`, `internal/factory/control/repair_admit_test.go:1-141`, `internal/store/factory_reassessment_allowances_test.go:1-1125`, and 12 more.

**Simpler.** When the budgets collapse into one allowance, delete the test functions that only cover budget, allowance, grant or sponsorship arithmetic, with their check-sql-locality exemptions (expiry, accounting, restore_helpers, completion_budget). Keep or port the non-budget cases mixed into the same files: plan-head and preparation refusals, reassessment record validation and CAS, policy CAS, withdrawal ordering, and settings replay. Keep one integration test that a run is held with a clear reason once the single allowance is used up.

**Risk.** Depends on the budget-collapse finding in the control/store subsystem. Do not delete these suites before that code changes, or the remaining budgets lose coverage.

### REL-4: tools/acceptance driver (exec/native/vm/probe-ssh/report, QMP, evidence store, owner-ID gates) has no in-tree callers; the newer TS collector writes observations directly, reusing only the report format

Release, install, packaging, CI and freshness machinery. duplicated-layer. partially-confirmed. ~5000 lines, risk low. Visible change: none. Behaviour-preserving: yes. **Status:** open (behaviour-preserving).

Where: `tools/acceptance/src/driver/`, `tools/acceptance/src/vm/`, `tools/acceptance/src/qmp.rs`, `tools/acceptance/src/evidence/`, `tools/acceptance/src/native_phase.rs`, `tools/acceptance/src/coreos.rs`, and 14 more.

**Simpler.** Delete the soda-acceptance driver binary and the modules it alone uses: driver, vm, qmp, evidence, report/handoff, native_phase, coreos, provisioning, probe, trust and remote.rs. Also drop the native-phase arm in soda-acceptance-remote. Remove tools/soda-acceptance from candidate packaging (build_compile.rs, soda-release-tools qualification.rs file list) and from the name exclusions, and update the docs. Keep soda-host-probes and the personal-git, cockpit-account and project-state remote phases, with the command/process/files helpers they still use. Keep report::Observation/read_observation/valid_owner until deliver's qualification check is simplified or moved, together with REL-2.

**Risk.** First confirm which soda-acceptance-remote subcommands (native-phase, cockpit-account, project-state) the TS collector and Go internal/acceptance still invoke.

**Safe first step.** Delete the following: - tools/acceptance/src/main.rs and the [[bin]] soda-acceptance entry in tools/acceptance/Cargo.toml. - These modules and their lib.rs entries: src/driver/, src/vm/, src/qmp.rs, src/evidence/, src/report/handoff.rs, src/coreos.rs, src/provisioning.rs, src/probe.rs, src/trust/, src/remote.rs. - src/native_phase.rs and the "native-phase" arm in src/bin/soda-acceptance-remote.rs. Then remove the ("soda-acceptance", "soda-acceptance") build and staging entries in lib/soda-release-image/src/build_compile.rs (around lines 223 and 259-261), and "tools/soda-acceptance" from lib/soda-release-tools/src/qualification.rs:402. Delete the driver sections of docs/development/native-support.md. Keep report/observation.rs, valid_owner and read_observation, which deliver qualification.rs uses. Keep host_probes, project_state, cockpit and personal_git, plus the command/process/files helpers they import.

### TST-6: Most tests/forgejo browser suites need a hand-run Forgejo at localhost:3300 and one-off review flags, so CI never runs them

Test harnesses and source guards. dead-code. partially-confirmed. ~3300 lines, risk low. Visible change: none. Behaviour-preserving: yes. **Status:** open (behaviour-preserving).

Where: `tests/forgejo/work-item-lists.test.ts`, `tests/forgejo/repository-container.test.ts`, `tests/forgejo/repository-switcher.test.ts`, `tests/forgejo/explore-overflow.test.ts`, `tests/forgejo/milestones-layout.test.ts`, `tests/forgejo/notification-preview.test.ts`, and 9 more.

**Simpler.** Delete the origin-gated and review-flag-gated browser suites, the component-browser fixture together with its lifecycle probe and test, and the SODA_IDENTITY_BROWSER test block in identity.test.ts (keep that file's three unconditional tests). Delete scripts/check-forgejo-branding.ts. Update the README and docs that mention these commands. Do not build a replacement suite unless a real need comes up.

**Risk.** Since none of these suites run today, deleting them loses no coverage CI currently has.

**Safe first step.** Delete scripts/check-forgejo-branding.ts and its row in docs/research/predecessor-reuse.md. Delete the suites gated on *_REVIEW flags: tests/forgejo/repository-switcher.test.ts and tests/forgejo/presentation/{dashboard-sidebar,form,migration,notification-layout,repository-actionbar}-browser.test.ts. Also delete the origin-gated tests/forgejo/presentation/{gallery.test.ts,refinement-browser.test.ts,home-background-browser.test.ts,login-station-browser.test.ts,profile-browser.test.ts}. Update frontend/forgejo/README.md and docs/design/screenshot-capture.md to match. None of these run in check:source or CI today.

### TST-4: tcontrol_oracle compiles production tailnet sources via #[path] to drive dead production test seams; stale port-era comments and in-process strictjson tests

Test harnesses and source guards. dead-code. partially-confirmed. ~3200 lines, risk low. Visible change: none. Behaviour-preserving: yes. **Status:** open (behaviour-preserving).

Where: `lib/host/tests/tcontrol_oracle.rs:1-2942`, `lib/host/src/tcontrol.rs:78-82,96-98,112,125,140`, `lib/host/src/tcontrol_policy.rs:445-447,456,563`, `lib/host/src/project/mod.rs:42-47`, `lib/host/src/terminal/factory/tcodex.rs:31-35`, `scripts/wire_contracts_test.go:1-5,50-160`, and 1 more.

**Simpler.** Delete lib/host/tests/tcontrol_oracle.rs. Remove the cli_stub, local_stub and provider_stub fields and branches from Control (tcontrol.rs), and the sync_hook field and branch from PolicyStore (tcontrol_policy.rs). Production always leaves them None. Drop the stale 'muse_serve_oracle compiles this through a #[path] copy' comments in project/mod.rs and tcodex.rs, and the #[allow(unused_imports)] annotations they justify. In scripts/wire_contracts_test.go, delete TestStrictjsonWireVectors and TestStrictjsonWireLimits (in-process unit tests) together with strictjson_vectors.json, and remove the 'frozen shared contracts / port lanes' wording. Keep the black-box binary tests (TestOperatorWireBytes, TestCLISurface, TestSystemdWiring, the dashboard config tests), the public-API and binary oracles (muse_serve, tcodex_ops, factory_roles), the release-crate behaviour tests, and the tcontrol_wire error text, which is the live daemon error contract.

**Risk.** Most of these lines are in Rust crates outside the narrow scope, so coordinate with the Rust/host finder. Removing the seams touches production Rust (small, mechanical). Error-text changes in tcontrol could alter messages shown to operators.

**Safe first step.** Delete lib/host/tests/tcontrol_oracle.rs. Then remove the cli_stub, local_stub and provider_stub fields, their None initialisers and their if-let/match branches from lib/host/src/tcontrol.rs (lines 78-82, 96-98, 112, 125, 140), and the sync_hook field, initialiser and branch from lib/host/src/tcontrol_policy.rs (lines 445-447, 456, 563). Production always leaves them None, so observable behaviour does not change. Also delete the stale #[path]-copy comments at lib/host/src/project/mod.rs:42-47 and lib/host/src/terminal/factory/tcodex.rs:31-33.

### GOM-8: internal/acceptance plus soda-installed-probes: 3.3k lines of fixture-bound installed probes that reproduce retired Python output

Other Go packages (forgejo, acceptance, project, host, tailnet, application, config, strictjson, filelock, avatar, cmd/soda-dashboard, cmd/soda-tailnet, cmd/soda-extension). process-machinery. confirmed. ~2800 lines, risk medium. Visible change: none. Behaviour-preserving: no. **Status:** open (behaviour-preserving).

Where: `internal/acceptance/*.go (3,220 lines)`, `internal/acceptance/README.md`, `tools/soda-installed-probes/main.go`, `lib/soda-release-image/src/build_compile.rs:270-275`, `lib/soda-release-tools/src/qualification.rs:405`, `scripts/build-qualification.ts:87,106,116`, and 5 more.

**Simpler.** Delete personal-git, workload-access, workload-exec, developer-access and service-https, along with the retired-Python failure-line and argparse exit-code compatibility. This also covers the Rust personal_git half in tools/acceptance and the sodaspaces_test developer-access test. factory-context-status is still invoked by the installed qualification phases (factory-phase.ts, docker-phases.ts). Either reduce it to a short plain check without the pinned tool revision and scope-file ceremony, or drop it from qualification. Remove the binary from the release image and the qualification file list.

**Risk.** Release/qualification scripts (build-qualification.ts, qualification.rs, build_compile.rs, tests/build/sodaspaces_test.go) reference it and must be trimmed at the same time. The owner must confirm which installed checks they still run by hand.

### DSP-12: Optional lifecycle effects with capability handshakes are an unneeded speculative feature

Factory dispatch and run start. speculative-abstraction. confirmed. ~2700 lines, risk low. Visible change: UI, DB schema. Behaviour-preserving: no. **Status:** done (`fcc7b413`).

Where: `internal/factory/lifecycle_action.go:1-430`, `internal/factory/lifecycle_action_config.go:1-98`, `internal/factory/control/lifecycle_consume.go:1-606`, `internal/factory/control/lifecycle_capabilities.go:1-79`, `internal/store/factory_lifecycle_actions.go:1-404`, `internal/web/api/factory_lifecycle_actions.go:1-291`, and 16 more.

**Simpler.** Delete the feature in full: the config, action and capability types; the consume and reconcile hooks; the store and the three tables; the API routes; the Forgejo executor and the handshake source; the dashboard credential field; and the Optional effects UI. Setup never creates the lifecycle credential, so the executor is already never wired on a default install. The owner can label issues or request reviewers in Forgejo directly.

**Risk.** Off by default, so removing it changes nothing for a default install. The settings finder may report the same thing.

### REL-6: `soda-install enroll-key` runs a temporary sshd and listener (2,226 lines) for something the installer and the password-enabled sshd already allow, and an x509 verifier exists only to self-check a local CA

Release, install, packaging, CI and freshness machinery. speculative-abstraction. confirmed. ~2700 lines, risk low. Visible change: UI. Behaviour-preserving: yes. **Status:** owner decision.

Where: `cmd/soda-install/src/enroll/:1-2226`, `cmd/soda-install/src/enroll/mod.rs:24-32`, `cmd/soda-install/src/run.rs:158-232`, `cmd/soda-install/src/main.rs:9-44`, `cmd/soda-install/src/execute.rs:127-131`, `cmd/soda-install/src/inputs.rs:185-229`, and 4 more.

**Simpler.** Delete the enroll-key, enrollment-serve and enrollment-receive actions and the enroll module, along with docs/operator/enroll-key.md and the references to it. Change the post-install hint to: paste your key during install, or use ssh-copy-id over the password-enabled sshd. Make local_ca_fingerprint return sha256_hex of the DER from pemx::decode_certificate. Delete x509/ and the p224, rsa, ecdsa, signature and ed25519-dalek deps. Keep p256, p384, p521, elliptic-curve and x509-cert, which sshkey and pemx use.

**Risk.** Removes an optional operator convenience. The fingerprint output stays identical.

**Safe first step.** In cmd/soda-install/src/setup/local_ca.rs, replace the parse and self-signature check with `Ok(crate::buildx::sha256_hex(&der))`. Then delete cmd/soda-install/src/x509/ and its `mod x509` line, and drop p224, rsa, ecdsa, signature and ed25519-dalek from Cargo.toml. The fingerprint printed for the host's real Caddy CA stays the same.

### REL-5: Release docs carry ~1,950 lines of redesign plan, J01-J26 task tracker and attempt receipts; the freshness doc carries a run log

Release, install, packaging, CI and freshness machinery. process-machinery. partially-confirmed. ~2200 lines, risk low. Visible change: none. Behaviour-preserving: yes. **Status:** open (behaviour-preserving).

Where: `docs/development/release.md:34-1983`, `docs/development/freshness.md:74-125`, `lib/soda-release-tools/README.md:103-302`, `lib/soda-release-deliver/README.md:1-149`, `docs/development/rc6-qualification-2026-10-25.md`.

**Simpler.** Cut release.md to its operator how-to (current lines 1-33 plus 1984-2529, tightened): build, stage, smoke-install, promote, Fountain maintenance. Delete the 34-1983 redesign plan and J01-J26 tracker with its receipts and paid-trial narratives; git history keeps them. Delete freshness.md's Implementation status run log (74-125) and keep the policy. Update the inbound anchors (#implementation-plan in tools/soda-installed-probes/README.md and lib/soda-release-tools/README.md; #normalize-the-qualification-procedure in the rc6 qualification doc). Trim the soda-release-tools README readiness and admission sections (103-302) to command usage once REL-2/REL-3 land.

**Risk.** Docs only. Agents read these files and follow them as procedure, so removing them also removes pressure to rebuild the machinery.

**Safe first step.** Delete docs/development/release.md lines 34-1983 (sections "Normalize the qualification procedure" through the end of "Final retained state and cold recovery", including the J01-J26 task list). Delete docs/development/freshness.md lines 74-125 ("Implementation status"). Repoint the links to release.md#implementation-plan (tools/soda-installed-probes/README.md:44, lib/soda-release-tools/README.md:204 and 244) and release.md#normalize-the-qualification-procedure (docs/development/rc6-qualification-2026-10-25.md:249 and 261) to #guided-release-operator-flow or remove them.

### REL-7: Two build front ends (soda-candidate wizard over soda-build) and two rootfs file servers

Release, install, packaging, CI and freshness machinery. duplicated-layer. partially-confirmed. ~2200 lines, risk low. Visible change: none. Behaviour-preserving: no. **Status:** open (behaviour-preserving).

Where: `lib/soda-release-tools/src/bin/soda-candidate.rs`, `lib/soda-release-tools/src/candidate/:1-470`, `lib/soda-release-tools/src/candidate_prompts/:1-485`, `lib/soda-release-tools/src/candidate_display/:1-487`, `lib/soda-release-tools/src/candidate_controller.rs:1-250`, `lib/soda-release-tools/src/candidate_fixture.rs:1-444`, and 7 more.

**Simpler.** Use `soda-build` with plain flags (the command candidate-setup already prints) and drop the soda-candidate wizard, its prompts, display, controller relay and hints. Keep one way to serve a rootfs to VMs: either the Go bridge service or a static server, not a second per-build hyper server. Narrow the boolean flags to true/false. Leave `soda-release start` out of this finding; whether it stays is a separate release-machinery question.

**Risk.** Operator UX only. Scripts invoking soda-candidate need the new flags.

### IDN-3: Identity wire records duplicated across broker and soda-host, with Go encoding/json emulation in Rust and Go-oracle byte tests

Identity service and provider credentials. duplicated-layer. partially-confirmed. ~1500 lines, risk medium. Visible change: none. Behaviour-preserving: yes. **Status:** open (behaviour-preserving).

Where: `cmd/soda-identity/src/wire.rs:1-95`, `cmd/soda-identity/src/wire_scalars.rs:1-136`, `cmd/soda-identity/src/wire_grants.rs:1-155`, `cmd/soda-identity/src/wire_execution.rs:1-268`, `cmd/soda-identity/src/providers/types.rs:1-54`, `cmd/soda-identity/src/strict.rs:1-144`, and 12 more.

**Simpler.** Move the derive-based identity wire structs (one Lease, one Binding, one Execution, one Delivery) into one module or crate shared by the broker and soda-host. Delete soda-host's case-insensitive visitors, its hand-built JSON string encoders, factory::Lease and the cv_* lease converters. Delete `null_tolerant`, plus strict.rs's duplicate-key pass for request bodies, and rely on derive plus deny_unknown_fields. Keep the Go structs and the Go string form of owner_id/actor_id/user_id, because the Go dashboard and factory control remain admin-socket clients (execution get/close, connections, grants, leases). Delete the unused Go runtime client methods (Acquire, Register, Return, Reject, ReconcileLease) and the Go-oracle byte tests.

**Risk.** Mechanical but spans three processes that ship together (Go dashboard, broker, host). Keep field names and the string form of owner_id/actor_id/user_id so existing JSONB rows still decode and dashboard responses stay identical. About 1,300 of the lines are in the listed scope; the rest are soda-host terminal/factory wire files.

**Safe first step.** 1. Delete the Go-oracle byte tests: lib/host/tests/identity_transport/requests.rs and responses.rs, plus the common.rs and main.rs wiring. 2. Delete the Go runtime-only client methods in internal/identity/client/client.go (Acquire, Register, Return, Reject, ReconcileLease, which have no production callers) and the parts of their tests that cover them. 3. Inside soda-host, replace factory::Lease (lib/host/src/factory/identity.rs, including its LeaseVisitor) with terminal::Lease and remove cv_lease_to_texec (lib/host/src/daemon/broker.rs) and its uses in dbackend/factory_terminal.rs. The broker sees identical bytes on the socket throughout.

### JDG-10: CompletionMaterial wraps repository state in 12 separately validated, digested projection types, with a freshness-reconstruction layer on top

Completion, reassessment, review and JEV judgments. duplicated-layer. partially-confirmed. ~1500 lines, risk high. Visible change: model tools. Behaviour-preserving: yes. **Status:** owner decision.

Where: `internal/factory/completion_material.go:1-1606`, `internal/store/factory_completion_material.go:1-1170`, `internal/store/factory_completion_source.go:1-304`, `internal/factory/reassessment.go:99-306`, `internal/factory/control/completion_proposal_inputs.go:26-168`, `internal/factory/control/completion_adoption_freshness.go:56-232`, and 5 more.

**Simpler.** Keep one plain read projection of current repository state for the reassessment, stall-repair and delivery-repair contexts. Drop the per-type and cross-reference validators, the canonical Digest/Equal freshness comparisons and matchGraphRepairEffects' reconstruction of the expected material. Have graph repair re-read the live rows it needs (bindings, merges, assignments, publications) when it computes protection and when it applies. Do this after JDG-1 removes the D13 consumer.

**Risk.** Shared with stall and delivery repair, which are outside this scope. Protected-task computation and adoption freshness depend on it. Best done after JDG-1 removes the D13 consumer.

**Safe first step.** In internal/factory/completion_material.go, delete the structural validators that only re-check data the store just captured: the Validate methods of CompletionWithdrawal, CompletionPreparation, CompletionPlanHead, CompletionPlanTask, CompletionRunState, CompletionReservation, CompletionAssignment (and its validateAddress/Subject/Stage/History helpers), CompletionPublication, CompletionMerge, CompletionCheck and CompletionAuthority, plus the CompletionMaterial validateAcceptances/Controls/Plans/Assignments/Publications/ChecksIntents index checks. Keep the nil-section check and the MaxCompletionMaterialBytes bound in CompletionMaterial.Validate. Drop the matching Validate() calls in internal/store/factory_completion_material.go. For valid captured data the outcome is identical, because these checks only refuse on internal bugs.

### PLN-9: Researcher and Library Admin roles are only reachable from a hand-written ```soda block, yet every prompt carries library guidance

Intent to plan (Product Lead, Architect, research, decomposition). speculative-abstraction. confirmed. ~1500 lines, risk medium. Visible change: UI, model tools. Behaviour-preserving: no. **Status:** done (`034f553c`, `ab431714`).

Where: `internal/factory/control/library_maintenance.go:11-61`, `internal/factory/control/planning.go:86-91`, `internal/factory/soda_issue_block.go:1-128`, `internal/factory/library.go:11-13,100-252`, `internal/factory/mcp_profile.go:26,82-93,117-119,177-187`, `internal/store/factory_library.go:83-254,389-705`, and 10 more.

**Simpler.** Delete the Researcher and Library Admin roles: their phases, the ```soda block parser and its routing, the 13 maintenance tools and 3 research tools, the inventory/preview/apply store code, and their Feed variants. Keep record_decision/learned creating library items, and keep search_library/read_library_item plus the founder Library view (apiFactoryLibrary). Cut LibraryReadGuidance down to a one-line hint, or drop it together with the library_guidance MCP field.

**Risk.** The owner's docs (R10.1, R32.7-R32.9) list these as intended profiles; confirm before deleting. The library item tables stay.

### VRF-3: Delivery repair only triggers on a model-reported outcome failure with a passing native check; a failing check.sh can never start a repair

Test Pilot / app verification (Go factory+control+store+MCP, Rust host, in-project helper). other. partially-confirmed. ~1500 lines, risk medium. Visible change: UI, model tools. Behaviour-preserving: no. **Status:** owner decision.

Where: `internal/factory/control/delivery_repair.go:200-265`, `internal/factory/control/app_verification.go:47-48,96-100`, `internal/factory/control/verification_execution.go:109-117`, `internal/factory/context_verification.go:135`, `lib/host/src/factory/finish.rs:134-160`, `internal/factory/verification.go`, and 4 more.

**Simpler.** The owner must make a product decision: either keep repair as the path for Test Pilot-observed outcome failures (as it works today), or cut the whole delivery-repair subsystem and route every failed verification through the existing Feed and Retry path. Turning a native check failure into a repair cause would be a feature addition, not a simplification.

**Risk.** Removing it drops a user-requestable repair action and its funding settings. Rewiring it touches dispatch and the budget code. Other finders covering repair and budgets should coordinate.

### DOC-4: About 1,300 lines of finished handbook-writing audits, plans and history; the live HB task queue in cutover.md stays (condensed)

Documentation. process-machinery. partially-confirmed. ~1400 lines, risk low. Visible change: none. Behaviour-preserving: yes. **Status:** open (behaviour-preserving).

Where: `docs/development/public-handbook/coverage-audit.md:1-385`, `docs/development/public-handbook/source-audit.md:1-370`, `docs/development/public-handbook/page-contracts.md:1-105`, `docs/development/public-handbook/proposed-tree.md:1-69`, `docs/development/public-handbook/ideal-filetree-plan.md:1-130`, `docs/development/public-handbook/cutover.md:16-39`, and 6 more.

**Simpler.** 1. Delete coverage-audit.md, source-audit.md, page-contracts.md, proposed-tree.md and ideal-filetree-plan.md. 2. In cutover.md, delete "Completed baseline" and everything from "Completed source cutover" to the end (disposition, packet allocation, shared writers, verification). Keep it as a short queue: HB01-HB18 plus the pending-notice id map, condensed. Better still, fold that queue into factory-remaining-tasks/05-experience-and-development.md and delete the folder. 3. Trim docs/public/README.md: drop the links to the deleted files, shorten the doc-pending rendering contract to a brief format note, and replace "Sources for maintainers" with one line. 4. Fix the links in docs/README.md:71, docs/development/README.md:74-79 and factory-remaining-tasks.md:1375. 5. Optionally, move the doc-visual production instructions out of the published pages.

**Risk.** The external soda-os-website sync consumes docs/public, so docs/public/README.md and the fence formats must stay in a form the website still accepts. Nothing outside docs/ reads public-handbook/.

**Safe first step.** 1. Delete docs/development/public-handbook/coverage-audit.md, source-audit.md, proposed-tree.md, ideal-filetree-plan.md and page-contracts.md. 2. Delete cutover.md lines 16-39 (Completed baseline) and 1039-1220 (Completed source cutover through Verification), which are history. 3. Remove the links to the deleted files from docs/public/README.md (lines 21-23 and 239) and docs/development/README.md:74-75. 4. Repoint factory-remaining-tasks.md:1375 from coverage-audit to the relevant HB task. Leave the HB01-HB18 queue and the pending-notice map in cutover.md alone. Leave docs/public pages untouched.

### FEP-5: VS Code-style tiling terminal layout with strictly re-validated sessionStorage persistence

Frontend: Workspace, project, applications, terminals, layout, Forgejo and Tailnet pages. speculative-abstraction. confirmed. ~1400 lines, risk low. Visible change: UI. Behaviour-preserving: yes. **Status:** owner decision.

Where: `frontend/spaces/sodaspaces-layout.ts:1-347`, `frontend/spaces/sodaspaces-workspace-layout.ts:1-300`, `frontend/spaces/sodaspaces-workspace-pane-view.ts:1-339`, `frontend/spaces/sodaspaces-workspace-measurement.ts:1-46`, `frontend/spaces/sodaspaces-workspace.ts:628-662,1560-1597,1990-2059`, `frontend/spaces/sodaspaces-workspace-terminal-hosts.ts:166-181,266-291`, and 7 more.

**Simpler.** Show a tab strip and one terminal at a time in the repository Terminals view (the server already lists the environment's terminals). Keep New terminal, rename and end. Drop the split tree, drag and drop, dividers, maximize/consolidate, the minimum projection, cell measurement and the minimum event. If restoring on reload is wanted, keep only the selected terminal id as one storage string, with no versioned tree and no strict parser.

**Risk.** Split panes go away. If FEP-1 lands as proposed this disappears with the Workspace. The finding matters on its own while ui/repo-places reuses SodaSpaces for its Terminals place. Geometry tests (workspace-layout, empty-pane, parts of zoom/dialog geometry) go with it.

**Safe first step.** Delete publishMinimum(), the lastMinimum field and the publishMinimum() call in measure() in frontend/spaces/sodaspaces-workspace.ts (around lines 302 and 639-661). Delete the 'soda-workspace-minimum' listener and the minimumNotifications counter in tests/frontend/fixtures/workspace-measurement-probe.ts, and the assertions that use them in tests/frontend/workspace-layout.test.ts (around lines 138, 167 and 181). Keep paneMinimum, because sodaspaces-workspace-layout.ts uses it to fit panes. Leave the parseLayout call inside serializeLayout for now: removing it changes the oversized-layout save notice.

### JDG-4: The merge-judgment gate, accepted-resolution reconciliation and MergeUnionReadyInput are reachable only from an allowlisted unit test

Completion, reassessment, review and JEV judgments. dead-code. partially-confirmed. ~1340 lines, risk low. Visible change: none. Behaviour-preserving: yes. **Status:** partly done (`61fbce18`): AdviseMergeJudgment and its tests deleted; the rest is open.

Where: `internal/factory/control/merge_judgment_gate.go:1-304`, `internal/factory/control/merge_judgment_gate_test.go:1-836`, `internal/factory/control/correction_resolution.go:22-181`, `internal/factory/control/merge_authorization.go:219-234`, `internal/factory/judgment_delivery.go:134-155`, `tools/unitban/allow-go.txt:43`.

**Simpler.** Delete merge_judgment_gate.go, merge_judgment_gate_test.go and the allowlist line for that test. Delete correction_resolution.go lines 22-181, keeping only AcceptedResolveOperationID. Delete MergeUnionReadyInput in merge_authorization.go, the last reference to MergeNativeGate. Delete DeliveryReceipt.ConfirmExpectedResolution. Optionally, in a later change, drop the write-only ObservedNativeRevision plumbing (judgment_delivery.go, judgment_advice_delivery.go, delivery_judgment_producer.go).

**Risk.** Nothing in production calls it. CheckMergeDecisionGate, the live merge gate in merge_operation.go, is unaffected.

**Safe first step.** Delete these together: internal/factory/control/merge_judgment_gate.go, internal/factory/control/merge_judgment_gate_test.go, and the line for merge_judgment_gate_test.go in tools/unitban/allow-go.txt (line 43). In the same change, delete MergeUnionReadyInput (with its comment) from internal/factory/control/merge_authorization.go:219-234, or the build breaks on MergeNativeGate. Delete everything after AcceptedResolveOperationID in internal/factory/control/correction_resolution.go (lines 22-181), then trim the unused imports. Delete DeliveryReceipt.ConfirmExpectedResolution in internal/factory/judgment_delivery.go:134-155. Leave the ObservedNativeRevision field alone for now.

### BUD-9: Repair funding: a repository repair grant, a charge ledger, a delivery-repair run ceiling and a 480-minute intent aggregate, all stored as command-ledger chains and partly dead

Budgets, allowances and limits (cross-cutting). too-many-states-or-budgets. confirmed. ~1300 lines, risk medium. Visible change: UI, model tools. Behaviour-preserving: yes. **Status:** owner decision.

Where: `internal/factory/stall.go:145-215`, `internal/factory/stall.go:216-405`, `internal/factory/stall.go:405-560`, `internal/factory/grants.go:67-82`, `internal/store/factory_repair_grants.go:14-547`, `internal/store/factory_repair_grants.go:590-690`, and 10 more.

**Simpler.** Replace all of this with constants: at most 2 automatic repair cycles per episode and 2 delivery repairs per accepted intent, counted from existing assignments, with each run using the per-run limit. Delete RepairGrant, RepairCharge and RepairAccount, IntentExecutionAllowance, the three command types, the grant/charge/limit code in factory_repair_grants.go, factory_delivery_repair_budget.go, the repair-funding API and component, the sponsor-equality gate and CurrentRepairBudget. The dead parts (about 170 lines) can go now with no behaviour change.

**Risk.** Repair is an automatic loop; keep the per-episode cycle constant so it cannot spin.

**Safe first step.** Delete the intentInterval type, IntentExecutionAllowance.ChargeInterval and IntentExecutionAllowance.Checkpoint (the 3-argument one) from internal/factory/stall.go (about lines 476-560), together with ErrIntentChargeOverlap and ErrIntentTimeExhausted if nothing else references them. Delete AttemptAllowanceHistory, IntentExecutionAggregate and IntentExecutionAggregateForIntent from internal/store/factory_repair_grants.go (about lines 590-690), along with the test that exercises them.

### TST-1: The unit-test ban is enforced by four mechanisms (shell gate, go vet analyzer, Rust build script, TS AST guard) that mostly duplicate one shell gate

Test harnesses and source guards. duplicated-layer. partially-confirmed. ~1300 lines, risk low. Visible change: none. Behaviour-preserving: yes. **Status:** open (behaviour-preserving).

Where: `tools/unitban/main.go:1-107`, `tools/unitban/testfixtures/unit.go:1-12`, `scripts/check-go-unitban.sh:1-37`, `tools/unitban/build_impl.rs:1-65`, `cmd/*/build.rs, lib/*/build.rs, tools/{candidate-setup,lab-credentials,pg-fixture,release-assets,test-vm}/build.rs (26 include-only 7-line files)`, `tools/acceptance/build.rs:5-6,54`, and 11 more.

**Simpler.** Keep check-no-unit-tests.sh as the single gate: the Rust src/ grep, the *_test.go and *.test.ts allowlists (or a directory rule), and the banned filename shapes, with *_test.ts, *_spec.ts, *.test.tsx and *_test.tsx added to that list so the coverage the TS guard gave is not lost. Delete these: - tools/unitban/main.go, testfixtures/ and check-go-unitban.sh, along with the golang.org/x/tools require in go.mod - build_impl.rs, the 26 include-only build.rs files, and the include! plus deny_unit_tests() call in tools/acceptance/build.rs and lib/soda-release-tools/build.rs - check-ts-unitban.ts and its two package.json scripts (and drop them from typecheck) Also drop check-go-unitban.sh from check-source.sh, check-push.sh and the expectedSourceCommands list in source_checks_test.go. Cut no-unit-tests.md to the rule and the one gate. Change AGENTS.md:27 to name the single gate, with owner approval.

**Risk.** AGENTS.md:27 is owner-authored and names the analyzer, the TS guard and the Rust build ban, so the owner needs to approve a one-line edit there. No product behaviour changes. Cargo builds get slightly faster without the build scripts.

**Safe first step.** Delete the Go vet analyzer layer: - tools/unitban/main.go, tools/unitban/testfixtures/unit.go and scripts/check-go-unitban.sh - the `check bash scripts/check-go-unitban.sh` line at scripts/check-source.sh:28 and the matching line at scripts/check-push.sh:25 - the {"bash","scripts/check-go-unitban.sh"} entry in tests/build/source_checks_test.go expectedSourceCommands - the golang.org/x/tools require in go.mod (it becomes unused) This keeps every outcome the same: any *_test.go outside allow-go.txt still fails check-no-unit-tests.sh, and Test functions in non-_test.go files never run. Second step, also behaviour-preserving for tracked files: delete tools/unitban/build_impl.rs, the 26 include-only build.rs files, and the include!/deny_unit_tests() lines in tools/acceptance/build.rs and lib/soda-release-tools/build.rs. The shell gate's Rust section covers the same markers.

### TST-3: tests/build and scripts: Python-era Go tests that pin source text and check that retired code stays absent (plus fake-tool tests of the dev gate scripts)

Test harnesses and source guards. unneeded-validation-or-digest. partially-confirmed. ~1300 lines, risk low. Visible change: none. Behaviour-preserving: yes. **Status:** open (behaviour-preserving).

Where: `tests/build/candidate_gate_test.go:1-69`, `tests/build/tailnet_image_test.go:1-44`, `tests/build/terminal_assets_test.go:1-81`, `tests/build/proxy_image_test.go:1-21`, `tests/build/project_runtime_test.go:1-132`, `tests/build/project_foundation_test.go:1-70`, and 10 more.

**Simpler.** Delete the pure text and absence pins: candidate_gate, tailnet_image, terminal_assets, proxy_image, project_runtime, project_foundation, forgejo_payload's text-only tests (ExtensionPages, OperatorSettings), pg_runtime_test lines 31-203, TestSystemdWiring, the SourceFlags loop of TestCLISurface (keep the part that runs binaries), TestConsoleWelcomeInstallWiring and TestForgejoMigrateUnitWiring. Separately, if the owner agrees that dev-gate scripts need no tests, delete source_checks and complexity_scope, which run the gate scripts with fake tools. Then drop TestSource from the check-push.sh -run regex and keep TestCI, TestPush and TestDockerInstaller. Keep helpers_test, or at least note that it covers the CargoBinary cache that the kept integration tests use. Keep forgejo_payload's template-closure test.

**Risk.** These tests catch only textual drift. Real regressions surface in the kept integration tests and the installed e2e.

**Safe first step.** Delete tests/build/candidate_gate_test.go, tests/build/tailnet_image_test.go, tests/build/terminal_assets_test.go, tests/build/proxy_image_test.go, tests/build/project_runtime_test.go and tests/build/project_foundation_test.go. Remove TestPostgresUnitTopology, TestForgejoUnitPostgresWiring, TestPostgresProvisionUnitWiring, TestDashboardUnitDatabaseWiring and TestPostgresBackupScheduleAndStaging from scripts/pg_runtime_test.go, and TestConsoleWelcomeInstallWiring and TestForgejoMigrateUnitWiring from their files. Before deleting, check that no remaining file uses helpers defined only in the deleted files (e.g. requireContains, readRuntimeFile, packageScripts). No product code changes.

### GOM-7: Frozen 'language port' contract fixtures still pin strictjson, the dashboard config schema, CLI help text and systemd lines after the port finished

Other Go packages (forgejo, acceptance, project, host, tailnet, application, config, strictjson, filelock, avatar, cmd/soda-dashboard, cmd/soda-tailnet, cmd/soda-extension). process-machinery. confirmed. ~1217 lines, risk low. Visible change: none. Behaviour-preserving: yes. **Status:** open (behaviour-preserving).

Where: `scripts/fixtures/portcontracts/*.json (676 lines)`, `scripts/system_formats_test.go:1-268`, `scripts/wire_contracts_test.go:1-273`.

**Simpler.** Delete scripts/fixtures/portcontracts and the two harness files (system_formats_test.go, wire_contracts_test.go). They are tests that check byte-for-byte compatibility for a language port that is already finished. If a systemd-wiring smoke check is wanted, keep it as a few greps in an existing check script.

**Risk.** These are tests and fixtures only. Deleting them removes a guard against accidental CLI/config drift, which matters to no external users.

**Safe first step.** Delete scripts/fixtures/portcontracts/ (cli_surface.json, dashboard_config_vectors.json, operator_wire.json, strictjson_vectors.json, systemd_wiring.json), scripts/system_formats_test.go and scripts/wire_contracts_test.go. First confirm that no other file in package scripts uses buildRustPortBinary, buildRustPortBinaryAs, runPortBinary or wireStubServer from these files. Leave internal/config and internal/strictjson as they are.

### DOC-8: Dated audits, run logs and lane-coordination contracts committed as docs

Documentation. process-machinery. confirmed. ~1200 lines, risk low. Visible change: none. Behaviour-preserving: yes. **Status:** open (behaviour-preserving).

Where: `docs/development/ux-audit-2026-10-19.md:1-606`, `docs/development/provider-freshness-contract.md:1-108`, `docs/development/frontend-vm-update.md:1-263`, `docs/development/factory-remaining-tasks.md:907-1093`, `frontend/forgejo/README.md:300-330,783-827`, `docs/design/factory-feed-components.md:27-37`, and 5 more.

**Simpler.** Delete ux-audit-2026-10-19.md, dropping its links in factory-demo-journey.md:53 and factory-remaining-tasks.md:909. Delete provider-freshness-contract.md and point freshness.md and credentials.md at lib/provider-tools/. Move frontend-vm-update.md to the host's VM notes, outside the repo, and drop its index rows in docs/README.md and docs/development/README.md, or cut it to a short generic dev-VM recipe. Strip the dated verification paragraphs and census diaries from frontend/forgejo/README.md, factory-feed-components.md ("Implementation update · 19 October 2026") and factory-remaining-tasks.md, keeping only what each file or contract is.

**Risk.** frontend-vm-update.md may be the owner's working deploy recipe, so relocate it rather than delete it outright.

**Safe first step.** Delete docs/development/provider-freshness-contract.md and change docs/development/freshness.md:82 and docs/reference/credentials.md:65 to point at lib/provider-tools/. Delete docs/development/ux-audit-2026-10-19.md and remove its links at docs/design/factory-demo-journey.md:53 and docs/development/factory-remaining-tasks.md:909. Both are docs only, with no code references.

### TST-11: Advisory complexity reporter is ungated dead weight; the lit-check gate carries a second TypeScript that unitban also depends on

Test harnesses and source guards. process-machinery. partially-confirmed. ~1100 lines, risk low. Visible change: none. Behaviour-preserving: yes. **Status:** done (`2775af41`).

Where: `tools/source-complexity/check.ts:1-151`, `tools/source-complexity/typescript.ts:1-113`, `tools/source-complexity/go/main.go:1-49`, `tools/source-complexity/rust/ (src/main.rs 264, Cargo.toml, Cargo.lock)`, `tools/source-complexity/tsconfig.json`, `scripts/check-complexity.sh:1-5`, and 12 more.

**Simpler.** Delete tools/source-complexity, scripts/check-complexity.sh, scripts/check-ts-complexity.sh, tests/build/complexity_scope_test.go and docs/development/complexity.md. Drop the check:complexity and check:ts-complexity scripts and the final source-complexity tsc step from typecheck, and remove the doc mentions. As a separate owner decision, lit-check could also go. That removes an enforced gate, and the classic TS 5.9 must be kept for check-ts-unitban.ts (for example as an aliased root devDependency). It also means updating worker_caches.rs, the build-forgejo.ts guard, bun.lock and the docs.

**Risk.** Lit attribute and property type mistakes would no longer be caught statically, only by the browser tests.

**Safe first step.** Delete tools/source-complexity/ (check.ts, typescript.ts, tsconfig.json, go/, rust/), scripts/check-complexity.sh, scripts/check-ts-complexity.sh, tests/build/complexity_scope_test.go and docs/development/complexity.md. In package.json, remove the check:complexity and check:ts-complexity scripts and the trailing `&& bun --bun tools/lit-check/node_modules/typescript/bin/tsc -p tools/source-complexity/tsconfig.json` from typecheck. Drop the references in docs/development/go.md:287, docs/development/typescript.md:86 and the check-oxlint.sh comment. No gate outcome changes. Leave tools/lit-check alone for now, since check-ts-unitban.ts needs its classic TypeScript.

### REL-10: Provider CLI store: an unused 787-line regression probe binary, per-launch re-hashing of every bundle file (twice per pin), and the gate logic copied in soda-identity and soda-muse

Release, install, packaging, CI and freshness machinery. dead-code. partially-confirmed. ~1050 lines, risk low. Visible change: none. Behaviour-preserving: yes. **Status:** partly done (`762e0dc4`): review_probe bin deleted; the per-launch re-hashing is open.

Where: `lib/provider-tools/src/bin/review_probe.rs:1-787`, `lib/provider-tools/src/store.rs:130-131,268-277,488-503,700-946`, `lib/provider-tools/src/resolve.rs:1-208`, `lib/provider-tools/src/refresh.rs:242-320,477`, `lib/provider-tools/src/lib.rs:1-60`, `cmd/soda-identity/src/providers/resolution.rs:1-155`, and 3 more.

**Simpler.** Delete review_probe.rs. Verify the archive sha256 once at download and admission, then trust the directory: drop inventory.json, scan_tree re-hashing and record_inventory, and keep only the release.json identity check and the executable-exists check. Remove the duplicate verify_bundle call in acquire_locked/resolve_pinned. Drop previous-*.json and keep the current selection plus refs. Have the library expose one gated new-work helper that returns the Selection with the pinned path and token, and use it from soda-identity resolution.rs and soda-muse store_resolve.rs instead of their copied gates. Update or delete the integration tests that assert tamper refusal.

**Risk.** Freshness and blocking semantics are unchanged. The only behaviour lost is detecting on-disk tampering of root-owned files.

**Safe first step.** Delete lib/provider-tools/src/bin/review_probe.rs (787 lines): it is an auto-discovered bin with no references, and no release build or script builds it. Also remove the redundant first verify_bundle call in resolve::resolve_pinned (acquire_locked already verifies under the same lock). Behaviour is identical.

### IDN-12: soda-identity-compose and nested-container Muse registration: a human-developer feature with its own frozen CLI parity fixture

Identity service and provider credentials. speculative-abstraction. confirmed. ~1000 lines, risk medium. Visible change: UI. Behaviour-preserving: yes. **Status:** owner decision.

Where: `cmd/soda-identity-compose/src/main.rs:1-42`, `cmd/soda-identity-compose/src/options.rs:1-158`, `cmd/soda-identity-compose/src/compose.rs:1-144`, `cmd/soda-identity-compose/src/registration.rs:1-150`, `cmd/soda-identity-compose/src/launch_wire.rs:1-88`, `cmd/soda-identity-compose/README.md`, and 22 more.

**Simpler.** If nested Compose Muse is not a current requirement, delete: - the binary and its README - the guide section and the root README row - the workspace member - the Containerfile, release-build, release-image, release-assets and soda-project-maintain staging entries - the tests/build entries and the parity fixture entry - soda-host's whole nested path: nested.rs, register_nested, nested_authorize, nested_caller, copy_nested_config, nested_pid, and the /run/soda-muse/nested argv, execution and validate branches. If it stays, replace options.rs with a short --login/--service/--file/--muse parser and drop the frozen usage-text fixture.

**Risk.** Product decision: removes a documented human-developer capability. The options.rs/fixture part alone is behaviour-preserving apart from usage text.

**Safe first step.** Delete the soda-identity-compose entry from scripts/fixtures/portcontracts/cli_surface.json (the --help case) and the matching `else if b.Package == "./cmd/soda-identity-compose"` branch in scripts/system_formats_test.go:216-218. This removes test-only parity machinery and changes no runtime behaviour. After that, options.rs can be simplified (bind_claude_flag renamed, Go-style bool spellings dropped). That second step changes only the help and error text.

### FEF-10: Settings embeds optional lifecycle effects that cannot run on a standard install, plus a second activity feed with display toggles saved on the server

Frontend: Factory settings, work, progress, request, intent screens. dead-code. confirmed. ~950 lines, risk low. Visible change: UI. Behaviour-preserving: no. **Status:** owner decision; coordinate with prod-hardening.

Where: `frontend/spaces/sodaspaces-factory-work.ts:755-1044`, `frontend/spaces/sodaspaces-factory-progress-view.ts:782-978,1061-1143`, `frontend/spaces/sodaspaces-factory-progress-response.ts:496-668`, `frontend/spaces/sodaspaces-factory-activity.ts:1-242`, `frontend/spaces/sodaspaces-factory-settings.ts:26-34,1048-1063,1082-1083`, `internal/web/server.go:109-111`, and 4 more.

**Simpler.** Remove both sections from Settings, along with the lifecycle-effects and activity-settings endpoints. First move nativePath and activityTimestamp out of activity.ts; 10 modules import them, including the Feed. Delete the Playwright tests that cover these elements. Optionally, also remove the nil-guarded Coordinator.Lifecycle consumers in factory/control and the forgejo lifecycle executor.

**Risk.** Removes a dormant feature that someone could enable by hand-editing the config. Add it back if a lifecycle credential is ever provisioned.

**Safe first step.** Move nativePath and activityTimestamp from frontend/spaces/sodaspaces-factory-activity.ts into a shared helper module, and update their 10 importers. That step alone is behaviour-neutral. Next, remove the soda-factory-activity and soda-factory-optional-effects mounts (settings.ts:1082-1083), the elements, and their endpoints, which is a user-visible change.

### IDN-9: Go keeps a second, test-only copy of the broker: runtime client calls, host callbacks, AES-GCM credential cipher and seed fixtures

Identity service and provider credentials. dead-code. confirmed. ~950 lines, risk low. Visible change: none. Behaviour-preserving: yes. **Status:** done (`8764b259`).

Where: `internal/identity/client/client.go:86-88`, `internal/identity/client/client.go:154-172`, `internal/host/identity.go:15-34`, `internal/identity/transport.go:18-22`, `internal/identity/types.go:116-119`, `internal/identity/launch.go:1-38`, and 12 more.

**Simpler.** Delete these pieces together with their tests. In Go, keep only the admin-socket client (connections, available, enrollments, grants, leases list/end, revoke, execution get/close), `host.IdentityLaunch`, and the record types they serialise. `store.Open` and `OpenDashboard` then become one key-free open. `OpenEphemeral` loses its key parameter, and `factory_settings_test.go` passes `nil` instead of a key. Keep the `grant_key_check` table in the shared schema, because the Rust broker uses it.

**Risk.** About two thirds of the lines are tests; no production path uses the removed code.

**Safe first step.** Delete these whole files: `internal/identity/launch.go`, `internal/identity/enrollment.go` and `internal/store/identity_fixture.go`. Also delete these tests: `internal/store/identity_test.go`, `internal/store/grants_test.go` and `internal/identity/client/broker_compat_test.go`. Then make these edits: - In `internal/identity/client/client.go`, remove `ReconcileLease`, `Acquire`, `Register`, `Return` and `Reject`. - In `internal/host/identity.go`, remove `IdentityValidate`, `IdentityStop`, `IdentityStart` and `IdentityFinish`, keeping `IdentityLaunch`. - Remove `DeliveryWire` from `internal/identity/transport.go` and `Delivery` from `internal/identity/types.go`. - Trim `internal/identity/client/client_test.go` to the `Connections` check, or delete it. - Delete `internal/store/grants.go` and remove the cipher and key-check code from `store.go`: `OpenEncrypted`, the `checkKey`/`initializeGrantKey` branches and the `grants` field. - Drop the key argument from `OpenEphemeral` and update its callers, which are tests only. Production uses `OpenDashboard` (no key, no key check), so its behaviour does not change.

### DOC-9: Product, design and handbook docs still make the Workspace the centre of Spaces, against the owner's no-Workspace decision

Documentation. other. partially-confirmed. ~900 lines, risk low. Visible change: UI. Behaviour-preserving: yes. **Status:** deferred to repo-first-ui (Workspace removal).

Where: `docs/product/spaces.md:3-60,109-137`, `docs/public/10-Start-here/30-spaces.md:1-50`, `docs/public/30-Projects-and-development/20-workspace-terminals.md:1-92`, `docs/public/30-Projects-and-development/10-projects-and-workspaces.md (filename only)`, `docs/design/spaces-ux.md:586-610,702-744`, `docs/design/spaces/ (README.md, render-sheets.ts, tsconfig.tools.json, sheets.css, *.svg)`.

**Simpler.** Delete docs/design/spaces/ entirely now: SVG sheets, sheets.css, README, render-sheets.ts and tsconfig.tools.json. Nothing outside docs references it. Also drop the 'Sheets and tooling' section of spaces-ux.md. When the Workspace frontend goes, rewrite product/spaces.md and public 30-spaces.md around the repository page, fold 20-workspace-terminals.md into a Terminal section of the repository page doc, rename 10-projects-and-workspaces.md to 10-projects.md, and remove the Full-page workspace and Persistent panel chapters of spaces-ux.md.

**Risk.** Must land with or after the frontend removal, or the handbook will describe UI that is not there. render-sheets.ts is compiled by its own tsconfig, so check that no TS check glob includes docs/design/spaces.

**Safe first step.** Delete the docs/design/spaces/ directory (drawer-compact.svg, drawer-sessions.svg, drawer-work.svg, focus.svg, mobile-states.svg, split.svg, sheets.css, README.md, render-sheets.ts, tsconfig.tools.json) and the '## Sheets and tooling' section at docs/design/spaces-ux.md:739-744. No code, build config or published handbook page references them.

### DSP-9: Durable paged readiness cascade, creation-recovery scan and per-pass RPC budgets for intake

Factory dispatch and run start. speculative-abstraction. confirmed. ~900 lines, risk medium. Visible change: DB schema. Behaviour-preserving: no. **Status:** owner decision.

Where: `internal/factory/control/readiness_work.go:13-166,260-486`, `internal/store/readiness_work.go:1-412`, `internal/store/readiness_creations.go:1-195`, `internal/factory/control/creation_recovery.go:1-166`, `internal/factory/control/readiness.go:102-149`, `internal/factory/control/dispatch.go:465-485`, and 3 more.

**Simpler.** On an intake hint or settlement, assess the changed issue and then its recorded dependants directly, in memory, within the pass deadline. A local appliance has few issues. Drop the durable source/node graph, pages, generations, capacity errors and RPC counters (factory_readiness_* tables). Bound completion paging with the plain context deadline, not the shared RPC budget. Replace creation recovery with periodic re-assessment of issues whose control is missing or stale, or drop it.

**Risk.** Crash-resume of a partially walked cascade goes away; with small local graphs the next pass simply re-walks. Keep the rule that intake is acknowledged only after assessment if webhook delivery depends on it.

### PLN-5: Generate re-proves the recorded plan about N+4 times, and a local 5-minute expiry or any stale check cancels the generation into terminal 'refused'

Intent to plan (Product Lead, Architect, research, decomposition). unneeded-validation-or-digest. partially-confirmed. ~900 lines, risk medium. Visible change: UI. Behaviour-preserving: yes. **Status:** owner decision; coordinate with prod-hardening.

Where: `internal/factory/control/intent_plan_choice.go:84-147,170-252,279-333,365-460`, `internal/factory/control/intent_generation.go:206-246,306-335,338-377,386-432,663-681,808-815,858-877`, `internal/factory/control/intent_bound.go:128-152`, `internal/web/api/factory_intent_generation.go:44-59`, `frontend/spaces/sodaspaces-intent-view.ts:440-449`, `internal/forgejo/issue_graph.go:97-100`.

**Simpler.** Validate the recorded plan once in ResolveAcceptedPlanChoice, before the funded choice: staged input, scope freeze, coverage and source. Drop the post-comparison recheck. Run one validateGenerationBinding immediately before SubmitGraph. Drop the before-adopt and per-task revalidations in bindValidatedGenerationTask and the repeated per-task CheckIntentOwner in readAdoptEvidence, and rely on native receipt matching during adoption. Keep NotAfter, since the native operation protocol requires it, but stop cancelling into 'refused' on local expiry or on a stale validation. Instead return a retryable error and refresh the window when the next generation begins. Keep 'refused' for genuine native EffectNotCommitted outcomes.

**Risk.** Generation and adoption are the critical path. Keep native receipt matching and the delivered-issue rules for deltas. Reassessment and repair generations share validateGenerationBinding (intent_plan_choice.go:374-379). Roughly 1,000 of the lines are intent_plan_choice_test.go and intent_generation_binding_test.go.

**Safe first step.** In internal/factory/control/intent_generation.go, make bindValidatedGenerationTask (:810-815) call c.Store.BindIntentTaskAcceptance directly, without revalidateGenerationBeforeAdopt, because revalidateGenerationBeforeAdopt already ran once at :229 before adoptIntentGraph. In internal/factory/control/intent_plan_choice.go, remove recheckPlanChoiceAfterComparison (:325-333) and its call at :140; the pre-submit validateGenerationBinding repeats the same source and head checks before any native effect. Together these remove N+1 redundant pure-read validations, and nothing observable changes when no concurrent edit happens mid-adoption.

### TRM-9: Terminal hosting is coupled to the Workspace tiling layout and should be replaced by a single-terminal host when the Workspace is removed

Human terminal stack end to end (browser, Go API, host, in-project helper). dead-code. partially-confirmed. ~900 lines, risk medium. Visible change: UI. Behaviour-preserving: no. **Status:** deferred to repo-first-ui (Workspace removal).

Where: `frontend/spaces/sodaspaces-workspace-terminals.ts:1-520`, `frontend/spaces/sodaspaces-workspace-terminal-hosts.ts:1-291`, `frontend/spaces/sodaspaces-layout.ts:1-347`, `frontend/spaces/sodaspaces-terminal-view.ts:115-119`, `frontend/spaces/sodaspaces-terminal-screen.ts:87-125`, `frontend/spaces/sodaspaces-workspace.ts`, and 1 more.

**Simpler.** As part of the Workspace removal, list terminals on the repository page from the existing inventory response. Open each one as a single full-size soda-terminal in its own place or tab, keyed by terminal id in the URL. Delete sodaspaces-workspace-terminals.ts, sodaspaces-workspace-terminal-hosts.ts and sodaspaces-layout.ts. Delete the tiling hooks (paneMinimum/publishMinimum geometry publication and the project/rename/hide workspaceCommand). Keep sodaspaces-terminal.ts, its response/vendor modules and the CSS classes, because the Factory view also uses them.

**Risk.** Depends on where the repository page puts terminals. The workspace-setup and inventory files are FEP's scope.

### DOC-6: Canlang runtime plan (890 lines, still the live contract for open P14-P32 packets) plus overlapping Canlang how-tos

Documentation. process-machinery. partially-confirmed. ~800 lines, risk low. Visible change: none. Behaviour-preserving: yes. **Status:** open (behaviour-preserving).

Where: `docs/development/canlang-soda-os-implementation-plan/production-workerd.md:1-728`, `docs/development/canlang-soda-os-implementation-plan/proposed-tree.md:1-162`, `docs/guides/canlang-applications.md:63-377`, `docs/public/30-Projects-and-development/70-canlang-development.md:238-308`, `docs/public/50-Operate/40-private-applications.md:1-226`, `docs/development/factory-remaining-tasks.md:1385,1647-1653,1661,1691`, and 4 more.

**Simpler.** Delete proposed-tree.md, which is mostly a file tree of the sibling canlang repo. Shrink production-workerd.md to a short runtime section in guides/canlang-applications.md, or link Canlang's docs/soda-runtime.md. Drop the 23 October verification diary, Alternatives considered, Independent challenge and readiness, the packet narrative and the source map. Keep only the still-true runtime facts: the 1 GiB / 2 CPU / 128-process envelope, UID 10000, where data lives, and the setup/restore behaviour. At the same time, repoint the 10 or so inbound links: factory-remaining-tasks P14-P28 rows, the development README, docs/README, testing.md, public-handbook/cutover.md and the guide. Then cut the guide's bundle-assembly and packaging steps that repeat the public pages, keeping only what is specific to developers and operators.

**Risk.** Docs only. Make sure no public page links into the plan dir before deleting.

**Safe first step.** Delete docs/development/canlang-soda-os-implementation-plan/proposed-tree.md and remove the sentence that links it in docs/development/README.md (lines 65-66). Nothing else links to it. Then cut the '23 October implementation' diary (production-workerd.md lines 7-30), 'Alternatives considered', 'Independent challenge and readiness' and 'Defining source map and target allocation' from production-workerd.md. No inbound anchor targets those sections; the inbound links use #implementation-packets-and-dependencies, #guided-application-setup-and-recovery and #source-baseline-scope-and-evidence.

### FED-8: Delete the second 'Factory activity' feed and its visibility settings

The Feed. duplicated-layer. partially-confirmed. ~800 lines, risk low. Visible change: UI, DB schema. Behaviour-preserving: no. **Status:** owner decision; coordinate with prod-hardening.

Where: `internal/store/factory_activity.go:1-260`, `internal/factory/activity.go:1-42`, `internal/web/api/factory_activity.go:1-67`, `internal/web/api/extension_native.go:215-216`, `internal/web/api/factory_access_view.go:62,91`, `internal/store/schema.go:19,354`, and 10 more.

**Simpler.** Delete the soda-factory-activity component, the activity API routes and handler, the store code and factory.Activity* types, the activity_settings access-view gate (backend and frontend), the handbook section, and the tests for these. Drop factory_activity_settings from both the Go and Rust schema lists and probes, or leave the table unused. Move nativePath and activityTimestamp, which nine modules import, into an existing helper module.

**Risk.** Factory settings lose a section, and the settings browser tests reference it. The table can be dropped, or left unused to avoid a schema change.

**Safe first step.** none (the only change that keeps behaviour identical is preparation: move nativePath and activityTimestamp out of frontend/spaces/sodaspaces-factory-activity.ts into an existing helper module and update the nine importers; the deletion itself removes a visible settings section)

### PTH-4: Hand-written Go encoding/json emulation in the host muse/terminal wire types

In-project helpers (soda-project-terminal) and the host Muse launch runtime. duplicated-layer. confirmed. ~750 lines, risk low. Visible change: none. Behaviour-preserving: yes. **Status:** open (behaviour-preserving).

Where: `lib/host/src/muse/wire.rs:1-425`, `lib/host/src/muse/validate.rs:126-430`, `lib/host/src/muse/inspect.rs:10-106`, `lib/host/src/terminal/wire.rs:1-470`, `cmd/soda-muse/src/launch_wire.rs:44-95`, `internal/identity/launch.go:1-38`.

**Simpler.** Use #[derive(Deserialize, Serialize)] with exact field names. Add #[serde(deny_unknown_fields)] on the launch-socket types, which are a trust boundary, and #[serde(default, skip_serializing_if = ...)] for omitempty. Give actor_id one small helper for its ',string' encoding. For podman Mounts, use a derived struct with Option fields, or serde_json::Value, and match on it. Replace the soda-muse LaunchExit visitor with a derive plus #[serde(default)]. Delete internal/identity/launch.go. Leave terminal/stream.rs's TerminalStart ',string' handling in place, because Go really does send that message.

**Risk.** Mixed-case keys and null-as-zero would stop being accepted, and only Go clients ever sent those. The launch socket is a real trust boundary, so keep deny_unknown_fields and the existing semantic validators (LaunchRequest::validate, registration_valid).

**Safe first step.** Delete internal/identity/launch.go; no Go code references any of its names. In cmd/soda-muse/src/launch_wire.rs, replace the hand-written LaunchExit Deserialize impl with #[derive(Deserialize)] #[serde(default)] struct LaunchExit { code: i64, error: String }. Unknown fields are already ignored by default, so the only difference is that repeated keys would be rejected. Then convert muse/wire.rs NestedRegistration, LaunchRequest and LaunchControl to derived types with deny_unknown_fields. The only sender, soda-muse, already emits exact lowercase keys.

### REL-9: Release crates duplicate models and helpers, and keep single-impl seam traits that no test uses

Release, install, packaging, CI and freshness machinery. duplicated-layer. partially-confirmed. ~750 lines, risk low. Visible change: none. Behaviour-preserving: yes. **Status:** open (behaviour-preserving).

Where: `lib/soda-release-tools/src/pipeline.rs:100-420`, `lib/soda-release-image/src/foreign.rs:1-140`, `lib/soda-release-image/src/model/trust.rs:12-138`, `lib/soda-release-deliver/src/model/trust.rs:13-100`, `lib/soda-release-deliver/src/model/mod.rs:27`, `lib/soda-release-image/src/model/candidate.rs:118`, and 16 more.

**Simpler.** Have the image crate use the Trust/Permit/Candidate/ProducedImage types from deliver and build (or one shared model module), and delete the pipeline.rs *_of conversions. Call BuildProduction directly and delete the foreign::Production/Progress traits and RealProduction. Replace the single-impl HttpTransport, Registry, GitHub and Runner traits with their concrete types. Use the release-inputs oci_architecture, is_digest, is_revision and hash_file everywhere, deleting the independent copies in build files.rs, tools digest.rs and soda-image-import, and the thin wrappers in image model.rs and acceptance. Keep a single MANIFEST_TYPE in deliver. Only touch the json_serde null shims if accepting different JSON is acceptable.

**Risk.** Mechanical refactor. Oracle tests that script the traits would need rewriting or deleting along with them.

**Safe first step.** Delete the 4 single-impl traits and use their concrete types directly: Registry/Skopeo (lib/soda-release-deliver/src/publish/registry.rs), GitHub/Gh (publish/mod.rs), Runner/Native (lib/soda-release-deliver/src/native/mod.rs) and HttpTransport/UreqTransport (lib/soda-release-build/src/http.rs). No test implements any of them. Separately, delete the duplicate MANIFEST_TYPE in lib/soda-release-deliver/src/oci/mod.rs:15 in favour of document.rs:11. Then replace the independent oci_architecture/is_digest/is_revision bodies in lib/soda-release-tools/src/digest.rs, lib/soda-release-build/src/files.rs:93 and cmd/soda-image-import/src/platform.rs:3 with calls to soda_build_tools::reader. All three already depend on it, and the error text is the same.

### DOC-5: no-unit-tests.md copies the enforcement scripts byte for byte (754 lines) for 'other projects'

Documentation. duplicated-layer. confirmed. ~700 lines, risk low. Visible change: none. Behaviour-preserving: yes. **Status:** open (behaviour-preserving).

Where: `docs/development/no-unit-tests.md:42-754`, `scripts/check-no-unit-tests.sh`, `scripts/check-go-unitban.sh`, `scripts/check-ts-unitban.ts`, `tools/unitban/main.go`, `tools/unitban/build_impl.rs`.

**Simpler.** Shrink no-unit-tests.md to about 40-50 lines. Keep the rule (what counts as a unit test), the 5-layer table with a path to each script or tool, how to extend the tools/unitban/allow-{go,ts}.txt allowlists, and a short limits note. Remove these sections: the embedded copies of the scripts, analyzer, linter, build_impl.rs and fixture; 'Adopting this in another project'; 'Removing the tests in the first place'; 'Verifying a layer'. Keep the file path (four docs link to it) and keep why-no-unit-tests.md, which the gates cite.

**Risk.** Check that no gate prints a link to no-unit-tests.md anchors that would disappear. Gates cite why-no-unit-tests.md.

**Safe first step.** In docs/development/no-unit-tests.md, remove these parts: - the five byte-identical code blocks: lines 83-160 (check-no-unit-tests.sh), 175-283 (main.go), 308-346 (check-go-unitban.sh), 379-585 (check-ts-unitban.ts) and 606-672 (build_impl.rs), plus the inlined unit.go fixture block - the 'Adopting this in another project' section (lines 42-63) - the 'Removing the tests in the first place' section (lines 692-720) - the 'Verifying a layer' section (lines 721-733) Then point each layer section at its script or tool path. Keep the file path and why-no-unit-tests.md, because no gate output refers to anything in no-unit-tests.md.

### DSP-10: Idle-pass stall classifier and opt-in stall investigator are a speculative recovery system; stall.go holds dead intent-allowance code

Factory dispatch and run start. speculative-abstraction. partially-confirmed. ~700 lines, risk medium. Visible change: UI, model tools, DB schema. Behaviour-preserving: yes. **Status:** owner decision; coordinate with prod-hardening.

Where: `internal/factory/control/stall_detect.go:1-517`, `internal/factory/control/stall_serve.go:21-88`, `internal/factory/control/progress.go:216-224`, `internal/factory/control/repair_admit.go:30-71`, `internal/factory/stall.go:151-185`, `internal/factory/stall.go:470-566`, and 4 more.

**Simpler.** Remove automatic stall investigation. When nothing is ready, show the named waits that dispatch already produces; the classifier can stay as a pure function for the UI if wanted. Delete stall events and episodes, the stall half of repair admission and the stall-investigation half of the repair-grant UI, coordinating with the repair/funding finder. Spending already requires an explicit repository repair grant, so this is about removing an opt-in subsystem, not about stopping an auto-spend. Delete the dead IntentExecutionAllowance methods now.

**Risk.** Types in stall.go (RepairCharge, StallRepairReport, IntentExecutionAllowance limits) are shared with delivery repair. Delete only what the detector owns unless the repair pipeline goes too.

**Safe first step.** In internal/factory/stall.go, delete ErrIntentChargeOverlap (keep ErrIntentTimeExhausted), the intentInterval type, IntentExecutionAllowance.ChargeInterval and the three-argument IntentExecutionAllowance.Checkpoint (about lines 473-566), and drop the math import if it becomes unused. Delete IntentExecutionAllowance.ApplyIncrease (lines 734-749) together with its only use, the ApplyIncrease block in internal/store/factory_repair_grants_test.go near line 413. No production code calls any of these.

### DSP-6: Automatic relaunch re-plans the run and validates lineage in both control and store; missing-run recovery is unreachable

Factory dispatch and run start. duplicated-layer. partially-confirmed. ~700 lines, risk medium. Visible change: UI. Behaviour-preserving: yes. **Status:** owner decision; coordinate with prod-hardening.

Where: `internal/factory/control/dispatch_launch.go:320-681`, `internal/factory/control/dispatch_launch.go:806-1074`, `internal/store/factory_dispatch_packet.go:444-935`, `internal/factory/control/dispatch_attempt.go:546-568`, `internal/factory/control/dispatch_recovery.go:86-112,191-264`, `internal/factory/control/dispatch_result.go:16-36,524-570`, and 2 more.

**Simpler.** On a refused, confirmed-unused launch, finish the assignment as failed with the host's reason and rely on the existing explicit Retry, or relaunch the recorded packet unchanged under a fresh run id without re-planning or control-side lineage revalidation. Keep lineage rules in one layer (the store's check* functions, returning a typed error that control maps to a superseded finish). Share one newestCommittedReviewRun helper. Make recoverSettled's held branch call AccountSettledRun (it already does usage + consume + finishFromRun-equivalent). Delete recoverMissingRun and the nil-run handling in dispatchHistoryRuns/activeHistoryRun.

**Risk.** Retry semantics for phased children change; keep the explicit Retry path. Host refusals are rare, so ending the attempt and asking for Retry is acceptable for local use.

**Safe first step.** In internal/factory/control/dispatch_recovery.go delete recoverMissingRun (lines ~254-264) and the ErrNotFound->nil branch in dispatchHistoryRuns (plus the nil guard in activeHistoryRun). Treat a missing run as a store error. Replace the ReservationHeld branch of recoverSettled with a call to AccountSettledRun(ctx, deps.Store, latest, recoverOutput(...), time.Now()). Replace newestCommittedCorrectionRunID in dispatch_launch.go with one exported helper shared with store's newestCommittedReviewRun, e.g. a method on factory.Publication. Leave the control-vs-store lineage dedup for later: it changes the outcome (superseded finish vs refusal) unless the store error is mapped.

### MCP-6: Dead, duplicate and test-only tools inflate a 24-tool shared surface

Model-facing MCP tools. dead-code. partially-confirmed. ~700 lines, risk low. Visible change: UI, model tools. Behaviour-preserving: yes. **Status:** partly done (`f03e8b4e`): P6 prototype server deleted; the live tool-surface trims are an owner decision.

Where: `internal/factory/mcp_profile.go:78-197`, `internal/factory/mcp_profile.go:264-325`, `internal/web/api/factory_mcp.go:731-732`, `internal/web/api/factory_mcp.go:954-1061`, `internal/web/api/factory_mcp.go:1087-1109`, `internal/web/api/factory_mcp.go:95-125`, and 16 more.

**Simpler.** 1. Delete status, evidence and research_scope; factory.context already serves the same data. 2. Delete request_handoff with its cap: the admission branch, checkMCPHandoffCapTx, MCPHandoffCount and both Err*HandoffCap. Keep a plain submissions-list limit constant for factory_context_current.go in place of MCPMaxHandoffs. 3. Fold the journey records with their IDs and the strategy inventory into current.role, then delete verification_scope and reassessment_scope. 4. After MCP-2 makes finish_reassessment validate before freezing, delete preview_repair and draft_repair together with the reassessmentWorkGuidance text that names them. 5. Optionally delete receipt, and update the hardening branch's frozen-refusal text that points to it. 6. Move FactoryMCPRoute beside factoryBoundCatalogHandler, then delete factory_mcp_p6.go and its test, factoryCatalogMux, the unbound capability_id branches (envelope property and required lists, the systemone BoundCapability==nil branch) and the tests that cover only them.

**Risk.** Fold the unique parts of the scope reads (journey records with their IDs, strategy inventory) into current.role before deleting those tools. About 400 test lines go with the removed code.

**Safe first step.** 1. Move `const FactoryMCPRoute = "/mcp"` from internal/web/api/factory_mcp_p6.go into internal/web/api/factory_mcp_bound.go. 2. Delete internal/web/api/factory_mcp_p6.go and internal/web/api/factory_mcp_p6_test.go. 3. Delete factoryCatalogMux (internal/web/api/factory_mcp.go:1087-1109) and its single test use at factory_mcp_test.go:1385. 4. Drop the `if s.deps.BoundCapability == nil` capability_id branch in internal/web/api/factory_mcp_systemone.go:122-125. All of these are test-only, because production always builds through factoryBoundCatalogHandler with BoundCapability set.

### CTX-2: Project knowledge library (items+revisions tables, CAS amendments, read guidance in every prompt) is a mutable projection of learned/decision Feed posts

Worker context assembly, project library and research. duplicated-layer. partially-confirmed. ~650 lines, risk low. Visible change: UI, model tools, DB schema. Behaviour-preserving: no. **Status:** owner decision; coordinate with prod-hardening.

Where: `internal/store/factory_library.go:17-85,255-385`, `internal/factory/library.go:10-130`, `internal/web/api/factory_mcp_library.go:151-252`, `internal/web/api/factory_mcp.go:913-915`, `internal/factory/control/dispatch_attempt.go:753-761`, `internal/factory/control/intent_bound.go:95`, and 10 more.

**Simpler.** Drop the library tables, revisions, amendment/challenge fields, LibraryReadGuidance and the library_admin maintenance phase. learned and record_decision stay ordinary Feed posts. Rewrite the Feed list query's decisions, knowledge, search and current filters (factory_feed.go:96-106) to use only post communication fields: kind and title/decision/rationale text. Delete the 'current' handling of removed/superseded items, or drop the current filter for these posts. If cross-run memory is wanted, keep it in the repo (NOTES.md). If workers still need a search tool, factory.search_library can become a single query over learned/decision Feed posts. Delete the library view and routes.

**Risk.** Workers lose a search tool whose content is a copy of Feed posts; any decision text lives on in the Feed and/or repo. Prompt text changes, so the model behaves differently: fewer startup calls.

### EXT-6: xterm is pinned twice: bun.lock plus a hand-kept per-file sha lock, a Rust fetcher, a hash-checking copy step and a dead payload-stage check

Fountain extension and its backend. duplicated-layer. confirmed. ~650 lines, risk low. Visible change: none. Behaviour-preserving: yes. **Status:** open (behaviour-preserving).

Where: `package.json:29,44-45`, `tools/release-assets/terminal-assets.lock.json:1-43`, `tools/release-assets/src/fetch/terminal.rs:1-345`, `tools/release-assets/src/bin/soda-fetch-terminal.rs`, `tools/release-assets/Cargo.toml:23-24`, `tools/release-assets/tests/fetchers_cli.rs`, and 18 more.

**Simpler.** Import '@xterm/xterm' and '@xterm/addon-fit' dynamically from node_modules and let Bun's splitting emit a lazy chunk. Bundle or copy xterm.css from node_modules into the terminal stylesheet, and copy the two LICENSE files in copyNotices. Delete the lock, soda-fetch-terminal (bin, fetcher, CLI test), the externals plugin, copyLockedTerminal and the --terminal-assets flag, terminal-vendor.d.ts, the payload-stage lock reader and its dead branch, the fetcher and --terminal-assets steps in production_assets.rs, the vendor symlink in build_context, and the lock tests. Point the frontend test fixtures at the built output. build:forgejo becomes bun-only.

**Risk.** Same upstream bytes (identical integrity). Confirm chunk URLs resolve relative to Fountain's assetBase (entries are already ESM with splitting and chunk-[hash].js naming). tools/release-assets may also be covered by the release-machinery audit.

**Safe first step.** Remove the dead payload-stage terminal check. Delete locked_terminal_assets, LockedItem and LockedAsset in tools/release-assets/src/render/stage/payload.rs:54-163. Delete the `@build/terminal-assets/` branch in stage/mod.rs:219-235 along with the `locked` parameter and the call at mod.rs:259. Drop the `@build/terminal-assets/` allowance in tests/build/forgejo_payload_test.go:64. No payload.json entry uses that origin, so output is identical.

### FEP-9: Project controls carry two Create/Join/Start renderings (first-use journey and settings view) and embed a second provider-accounts UI (soda-identity)

Frontend: Workspace, project, applications, terminals, layout, Forgejo and Tailnet pages. duplicated-layer. partially-confirmed. ~650 lines, risk medium. Visible change: UI. Behaviour-preserving: yes. **Status:** owner decision.

Where: `frontend/spaces/sodaspaces-project.ts:68,163,307-312,391-428`, `frontend/spaces/sodaspaces-project-journey-view.ts:1-451`, `frontend/spaces/sodaspaces-project-settings-view.ts:28-77,162-413`, `frontend/spaces/sodaspaces-workspace.ts:774,1479,2115-2126`, `frontend/spaces/soda-identity.ts:1-291`, `frontend/spaces/soda-identity-response.ts:24-38,92-114`, and 2 more.

**Simpler.** Keep one environment section, the existing settings-view path: status, Create for owners, Join, Start/Stop for admins, keys and network. Drop the separate journey renderer and fold its useful first-use copy into that section. Remove the 'standard' versus 'settings' flag distinction, along with the panel. Remove soda-identity from project controls and keep provider accounts in the header dialog; move delegation and leases there only if they are still wanted. Drop the hidden non-admin fieldset and the test-only ProjectContext.settings flag.

**Risk.** First-use copy and layout change. Delegation/lease management needs a home or is dropped; confirm with the owner. Coordinate with ui/repo-places (M19 Settings > SodaOS, EnvironmentStrip).

**Safe first step.** Remove `settings?: boolean` from ProjectContext (sodaspaces-project.ts:68) and the `!!this.binding?.settings ||` term in readBindingSettings (line 426). Change tests/frontend/fixtures/drawer-fixture.ts:242 to call api.setPresentation('settings') instead of passing settings: true. Product callers never pass the flag, so observable behaviour stays the same.

### STO-7: About 35 exported Store methods (~650 lines) have no production caller, including second write paths, unreachable states and unreachable features

Store (PostgreSQL layer) structure. dead-code. confirmed. ~650 lines, risk medium. Visible change: none. Behaviour-preserving: yes. **Status:** coordinate with prod-hardening.

Where: `internal/store/factory_grants.go:44-126`, `internal/store/factory_grant_commands.go:405-466`, `internal/store/project_grants.go:15-22`, `internal/store/preparation.go:18-53`, `internal/store/preparation.go:98-127`, `internal/store/factory_mcp_capability.go:103-193`, and 15 more.

**Simpler.** Delete the dead methods and their private helpers (markFeedComment, blockCommentEffectTx, casMCPCapabilityState). Drop the revoked and retired capability states and the 44 deleted/revoked predicates. Make tests seed through the production paths: grant commands, dispatch packet, live activation and SaveFeedCommentWithSource. Delete tests that only cover the removed methods. Keep saveFactoryGrantChangeTx and factory_mcp_live.go: they are the production paths, listed only for comparison.

**Risk.** Production is unaffected. About 150 test call sites need re-seeding or deletion (SaveRepositoryPolicy is used 37 times, SaveSponsorship 34).

**Safe first step.** Delete the two methods that nothing calls, not even tests: Store.SpaceProjects in internal/store/store.go:264-285 and Store.RecordRepairCharge in internal/store/factory_repair_grants.go:257-271, plus any private helper that becomes unused. Also update the doc comments in factory_inventory.go:102 and members.go:11 that mention SpaceProjects. Next, delete the reader-only methods Sponsorship and LifecycleGrant, and RetireMCPCapability (1 test ref), together with their tests.

### FEP-2: Own-server replies are re-validated field by field (key counts, echo checks, duplicate server invariants), so additive backend changes break whole views

Frontend: Workspace, project, applications, terminals, layout, Forgejo and Tailnet pages. unneeded-validation-or-digest. partially-confirmed. ~600 lines, risk low. Visible change: none. Behaviour-preserving: yes. **Status:** open (behaviour-preserving).

Where: `frontend/spaces/sodaspaces-application-response.ts:65-397`, `frontend/spaces/sodaspaces-inventory-response.ts:24-121`, `frontend/spaces/sodaspaces-project-response.ts:15-143`, `frontend/spaces/sodaspaces-project-refresh.ts:16-30,141-153,219,376-401`, `frontend/spaces/sodaspaces-project-mutations.ts:96-187`, `frontend/spaces/sodaspaces-terminal-response.ts:20-65`, and 5 more.

**Simpler.** Describe replies with TypeScript interfaces. Keep one shallow object() guard plus the few fields that drive control flow or status text (for example tailnet_outcome, provisioned, applied). Delete the exact key-count checks, regex re-checks of IDs and digests, echo-equality checks and JSON.stringify invariants, and the 64-terminal and origin rules that duplicate server-side checks. Keep href/origin checks only for external URLs that end up in an href. The same-bundle event parsers in workspace-setup go away with the Workspace removal (FEP-1/FEP-5).

**Risk.** Identical behaviour for valid replies. Malformed replies would render partially instead of failing. Browser tests that assert refusals (application-controls, identity, tailnet) shrink with it. About 250 more lines sit in Workspace files (layout parse, setup event parsers) that FEP-1/FEP-5 delete.

**Safe first step.** In frontend/spaces/sodaspaces-application-response.ts, drop the `Object.keys(s).length === 12` check in identityFields and the `Object.keys(a).length === 4` / `Object.keys(row).length === 6` checks. Drop the JSON.stringify restore-equality checks (213-217) and the regex re-checks of IDs, digests and the origin; the server enforces those in Spec.Validate. In frontend/spaces/sodaspaces-project-mutations.ts, delete the echo-only checks (installed_fingerprints JSON equality, lifecycle running/boot_enabled echo, delete existing_project_access_changed) and keep the branches that pick status text. Valid replies render identically. Delete the browser tests that assert these refusals along with the checks.

### HPL-9: Terminal helper digest check (Spaces field + attach gate) and standalone terminal-support repair subcommand for retained projects

Rust host platform: daemon, terminals, project, application, tailnet (lib/host non-factory, soda-project-maintain, forgejo/pg/console helpers, soda-tailnet, unix-http, wire-time). speculative-abstraction. partially-confirmed. ~600 lines, risk low. Visible change: UI. Behaviour-preserving: no. **Status:** owner decision.

Where: `lib/host/src/terminal/support.rs:1-159`, `lib/host/src/terminal/support_root.rs:60-115 (in_stopped_root; stopped_namespace_argv is still used by release maintenance)`, `lib/host/src/daemon/routes.rs:222`, `lib/host/src/dbackend/backend.rs:218-220`, `lib/host/src/daemon/backend.rs:153`, `lib/host/src/dbackend/websocket.rs:101-105`, and 12 more.

**Simpler.** Drop the /terminal-support observation route, the terminal_support field on Spaces/environment responses and its settings-view sentence, the pre-attach digest gate and its error codes, and the `terminal-support` maintain subcommand flow. Keep terminal_support.sh, Root, guest and the helpers while release maintenance still uses them, or delete them together with release maintenance in a separate change. An outdated retained Project is recreated.

**Risk.** Remove the Go TerminalSupport field and any Spaces UI badge with it.

### PRJ-2: Hand-written serde Visitors and JSON encoders emulating Go encoding/json for every project DTO

Project environment runtime on the host (create/join/start, executor, domain, ssh, accounts). duplicated-layer. confirmed. ~600 lines, risk low. Visible change: none. Behaviour-preserving: yes. **Status:** open (behaviour-preserving).

Where: `lib/host/src/project/inspect.rs:13-330`, `lib/host/src/project/profile.rs:11-64`, `lib/host/src/project/mod.rs:64-127`, `lib/host/src/domain/profile.rs:18-131`, `lib/host/src/domain/profile.rs:148-203`, `lib/host/src/domain/account.rs:64-204`, and 6 more.

**Simpler.** Use #[derive(Deserialize, Serialize)] with exact lowercase keys (rename_all = "PascalCase" for the podman structs). Add skip_serializing_if for omitempty fields. Use Option<T> or one small null_as_default helper wherever Go or podman can send null (Account/AccessKeys keys, IDMappings, Labels). Keep json::SignedInteger and decode_strict_as through #[serde(deserialize_with)]/types. Delete the tolerant_object! macro, the take_inspection_* helpers, every manual encode_into, and the GoStringList copies, and use serde_json::to_string.

**Risk.** The real inputs all use exact-case keys, so derive accepts the same documents. The output only differs in escaping of <, > and & (Go's < form versus literal characters), which the Go json decoder treats identically. The Profile label written into containers must decode the same way, and it does with derive.

**Safe first step.** In lib/host/src/project/mod.rs, replace the hand-written Lifecycle Visitor (about lines 70-110) with `#[derive(Deserialize)] #[serde(deny_unknown_fields)]` and Option-aware fields: either `#[serde(default, deserialize_with = null_as_default)]` or Option<String> mapped to String. Do the same for AccountConfirmation in lib/host/src/account/mod.rs:46-87, keeping json::SignedInteger for identity. Both are only fed exact lowercase keys by the Go host client (internal/project/types.go) and the native helper. Then replace LifecycleState::encode and Connection::encode with serde Serialize. Their outputs match Go except for the <>& escape spelling, which never occurs in these hex, IP and bool values.

### PRP-5: Rust host preparation domain mirrors Go validation and decodes with hand-written, Go-compatible serde visitors

Project preparation pipeline below the API (standing and candidate preparations, decisions, hold, host and root helper). duplicated-layer. confirmed. ~570 lines, risk low. Visible change: none. Behaviour-preserving: yes. **Status:** open (behaviour-preserving).

Where: `lib/host/src/preparation/mod.rs:116-259`, `lib/host/src/preparation/decisions.rs:1-103`, `lib/host/src/preparation/setup.rs:13-147`, `lib/host/src/preparation/candidate.rs:55-120`, `lib/host/src/preparation/state.rs:28-60,364-423`, `lib/host/src/prepare/state.rs:25-290`, and 5 more.

**Simpler.** Use #[derive(Deserialize)] with #[serde(deny_unknown_fields)] on plain structs. Use #[serde(default)] or Option for fields Go may send as null (nil slices and maps). Keep one small base64 deserialize_with helper for byte fields. Drop the case-insensitive key matching. Remove the AdminApproval validation: the approval is never read. Reduce RequirementAcceptance validation to source_commit and source_inputs, which are the only fields used. Optionally keep only one of the Go client reply-identity check (host/prepare.go) and the coordinator's adoptStandingPreparation check.

**Risk.** The wire bytes do not change if serde field names match. Go always sends lower-case keys, so case-insensitive matching is never exercised.

**Safe first step.** Two changes, both in lib/host/src. (1) In preparation/mod.rs, delete the `self.approval.validate()?;` call in Preparation::validate. Delete AdminApproval::validate and the typed_decision!(AdminApproval ...) invocation in preparation/decisions.rs, and decode approval as an ignored field, either serde::de::IgnoredAny or a plain derived struct. The host never reads the approval, and Go always sends a valid one, so outcomes do not change. (2) Replace the typed_decision! and typed_preparation_object! macros (decisions.rs:69-103, mod.rs:214-259) with #[derive(Deserialize)] #[serde(deny_unknown_fields)] plus #[serde(default)] on the field structs. Keep the json::SignedInteger and GoStringList field adapters through deserialize_with, so input from the Go client internal/host decodes the same way.

### IDN-5: Identity store re-validates rows it wrote itself, pages 64 rows at a time, and wraps tokio-postgres in a try-lock/deadline engine whose lock is never contended

Identity service and provider credentials. unneeded-validation-or-digest. confirmed. ~550 lines, risk low. Visible change: none. Behaviour-preserving: yes. **Status:** open (behaviour-preserving).

Where: `cmd/soda-identity/src/store_connections.rs:8-15`, `cmd/soda-identity/src/store_connections.rs:87-173`, `cmd/soda-identity/src/store_connections.rs:209-271`, `cmd/soda-identity/src/store_grants.rs:48-118`, `cmd/soda-identity/src/store_leases.rs:13-103`, `cmd/soda-identity/src/store_leases.rs:217-234`, and 11 more.

**Simpler.** Use plain 'SELECT data FROM ... WHERE ...' into a Vec<T> and serde_json::to_vec for list responses. Use one loop over a Vec for each retirement sweep. Replace the try_lock spin with Mutex::lock and make Store.grants a plain GrantCipher. Replace the per-call deadline/cancel machinery with a statement_timeout, keeping lazy reconnect when the session is closed. Delete ListJson/BoundedListWriter, LIST_* constants, the screening SQL, decode_lease_page, LeaseFailures, pg_query.rs, the Tx failed-flag plumbing where client.transaction() covers it, and the four screening tests. Keep the two atomic transactions.

**Risk.** Keep the two transactions that provide real atomicity (reserve lease + link execution; store returned credential + delete lease). Very large lists are not a real case (rows are connections and leases of one operator).

**Safe first step.** In cmd/soda-identity/src/store.rs, replace the begin_operation try_lock/sleep loop with self.client.lock() and remove the operation_budget field (use OPERATION_BUDGET directly). Make Store.grants a GrantCipher instead of Option<GrantCipher>: fold open() into open_encrypted and drop the is_none/is_some checks at store_connections.rs:19 and store.rs:56. Behaviour stays identical because Controller's Mutex already serializes every Store call and open is only ever called with Some(cipher).

### VRF-8: Separate application-verification allowance overlapping attempt limits, with a duplicated cap

Test Pilot / app verification (Go factory+control+store+MCP, Rust host, in-project helper). too-many-states-or-budgets. confirmed. ~550 lines, risk low. Visible change: UI. Behaviour-preserving: yes. **Status:** owner decision.

Where: `internal/factory/verification.go:21-28`, `internal/factory/verification_allowance.go:1-69`, `internal/store/factory_verification_allowance.go:1-246`, `internal/web/api/factory_verification_allowance.go:1-81`, `internal/web/api/factory_lifecycle.go:225-230`, `internal/web/api/extension_native.go:200-201`, and 7 more.

**Simpler.** Use the ordinary attempt limit with one fixed per-run cap for verify runs. Delete AppVerificationLimits and Usage/Allowance/Change, the verification-allowance command chain, its API routes and Lit view, and the Rust MAX_CHECK_ACTIVE copy. Apply the cap in a single place, at admission, and keep the repair-run count separate. Coordinate with the budgets finder.

**Risk.** Removes an owner setting. The product has no users, so nothing depends on custom values.

**Safe first step.** Delete the Rust duplicate cap: remove MAX_CHECK_ACTIVE and its comment (verification.rs:23-25) and the deadline-exceeds-one-run check (verification.rs:310-312). The Go admission (factory_delivery_admission.go:129-138) already rejects any deadline beyond min(policy, 30 min, remaining), so that Rust check can never fire.

### WEB-12: The /api/spaces inventory carries a bounding framework (slots, caps, timeouts, cursor, marshal loop) for a handful of projects, and its only uses are the Workspace and a project picker on Applications

Web API and view models (non-MCP, non-Feed, non-extension): internal/web/*.go, internal/web/auth/**, internal/web/api/*.go excluding factory_mcp*, factory_feed*, extension*. too-many-states-or-budgets. partially-confirmed. ~550 lines, risk low. Visible change: UI. Behaviour-preserving: no. **Status:** deferred to repo-first-ui (Workspace removal).

Where: `internal/web/api/spaces.go:17-178`, `internal/web/api/spaces_inspection.go:1-171`, `internal/web/api/spaces_inventory.go:1-143`, `internal/web/api/spaces_authority.go:1-80`, `internal/web/api/api.go:42-43,60`, `internal/web/api/repositories.go:116-122`, and 6 more.

**Simpler.** Point the Applications project picker (sodaspaces-applications.ts loadProjects) at a plain project list: one store query returning repository_id and name, with no slots, caps, cursor, Tailnet or factory inspection. Then delete /api/spaces (spaces*.go, SpacesSlots) together with the Workspace frontend (workspace-inventory, inventory-response). Also drop RepositorySlots and /api/repositories along with sodaspaces-workspace-setup.ts, which is its only caller.

**Risk.** This is a product change tied to removing the Workspace frontend; do the two together.

### WEB-6: Issue journey read: lineage, chain paging, 24 reason codes and row re-validation that no client reads

Web API and view models (non-MCP, non-Feed, non-extension): internal/web/*.go, internal/web/auth/**, internal/web/api/*.go excluding factory_mcp*, factory_feed*, extension*. unneeded-validation-or-digest. confirmed. ~550 lines, risk low. Visible change: none. Behaviour-preserving: yes. **Status:** open (behaviour-preserving).

Where: `internal/web/api/factory_plan_view_work.go:12-25,83-519,529-640`, `internal/web/api/factory_journey_readback.go:15-21,35-113,343-445`, `internal/web/api/factory_issue_view.go:36,719`, `frontend/spaces/sodaspaces-factory-work-journey-response.ts:1-419`, `frontend/spaces/sodaspaces-factory-context-resolution.ts:102-104`.

**Simpler.** Build each section straight from store rows as plain struct literals. Reduce lineage to latest_run, or move latest_run onto the issue view's attempt. Delete journeyChain, the 24 reason codes, addJourneyReason, available/complete, applyJourneyPublicationLineage and the fixture-era DTO validators. Trim the frontend parser so it only checks the types it renders.

**Risk.** Rows that are hidden today as 'invalid' would appear. AcceptanceAssignmentsAfter keeps its other caller in control/acceptance_status.go.

**Safe first step.** In internal/web/api/factory_journey_readback.go, delete journeyChain, the journeyChainPageLimit/MaxPages constants and the chain block at the top of journeyPhaseSection. Also delete applyJourneyPublicationLineage and its call. In factory_plan_view_work.go, drop every journeyLineageView field except LatestRun, plus Reasons, Available, Complete, the journeyReason* constants and addJourneyReason, and remove every addJourneyReason call site. Delete the Go tests that assert those fields. Leave the section DTO validators in place for a second step, because removing them could reveal rows that are hidden today.

### WEB-7: Routes and fields nothing calls or reads (one of them drives a dead withdrawal subsystem), plus one field that is never set

Web API and view models (non-MCP, non-Feed, non-extension): internal/web/*.go, internal/web/auth/**, internal/web/api/*.go excluding factory_mcp*, factory_feed*, extension*. dead-code. confirmed. ~550 lines, risk low. Visible change: UI, DB schema. Behaviour-preserving: yes. **Status:** owner decision; coordinate with prod-hardening.

Where: `internal/web/api/factory_assignments.go:13-35,59-118,142-177 (keep factoryAssignmentPublication for workPublicationDTO)`, `internal/web/api/extension_native.go:238-239 (route registrations)`, `internal/web/api/issue_acceptances.go:142-177`, `internal/factory/control/acceptance_status.go:17-28,65,219-430`, `internal/factory/control/prerequisites.go:154-160,266`, `internal/factory/control/settle.go:179-180`, and 10 more.

**Simpler.** Delete the assignment and withdrawal routes and handlers, the whole withdrawal coordinator path (WithdrawAcceptance and helpers, CommandWithdrawal, the settle.go recovery case), the store writer and reader, and every issue_acceptance_withdrawals check, including the AcceptanceWithdrawn reads in acceptance_status.go and prerequisites.go. Delete the POST /api/applications registration. Remove can_embed and applicationEmbeddable from the backend and the frontend parsers together. Drop the plan cursor and keep only limit. Remove routing_verified. Compute queue.active or drop it. Add Enrollments to the IdentityClient interface and delete the fallback. Delete SetForgejo and SetHost, and switch the test helper to set the field directly. Update docs/reference/api.md to match.

**Risk.** Go unit tests covering these can be deleted with them. If queue.active is computed in the backend, the frontend parser the hardening branch edits stays untouched; removing the withdrawal NOT EXISTS clauses touches store/factory_publications.go, which the hardening branch also edits.

**Safe first step.** In internal/web/api/extension_native.go, remove the withdrawal and assignment route lines (238-239). Delete apiIssueWithdrawal and issueWithdrawalRequest from internal/web/api/issue_acceptances.go. Delete apiFactoryAssignment and the reservation part of the view from internal/web/api/factory_assignments.go, keeping factoryAssignmentPublication for workPublicationDTO. Delete Coordinator.WithdrawAcceptance and its resume, recover, finish, settle and replay helpers plus WithdrawalReceipt from internal/factory/control/acceptance_status.go, and the CommandWithdrawal case from settle.go:179-180. Delete Store.WithdrawAcceptanceDecision. In internal/web/api/applications.go, delete apiApplicationRegister and its POST dispatch. Delete SetForgejo/SetHost in internal/web/server.go. Delete the identity.go fallback after adding Enrollments to the interface. Delete the Go unit tests covering these. Leave the store withdrawal NOT EXISTS clauses, which touch factory_publications.go (hardening branch), the table itself, and queue.active for a later step. Nothing a user sees changes.

### DLV-8: Repair funding stacks about seven budgets and a revisioned grant ceremony on one Strategist run; no default grant exists, so repair does nothing until someone funds it by hand

Publication, merge, correction, repair, acceptance, verification, lifecycle. too-many-states-or-budgets. confirmed. ~500 lines, risk medium. Visible change: UI. Behaviour-preserving: no. **Status:** owner decision.

Where: `internal/store/factory_repair_grants.go:14-423`, `internal/store/factory_delivery_repair_budget.go:16-145`, `internal/store/factory_delivery_repair_request.go:13-83`, `internal/store/factory_delivery_repair_adoption.go:26-94`, `internal/factory/control/delivery_repair.go:127-157`, `internal/factory/control/delivery_repair.go:292-322`, and 6 more.

**Simpler.** Use one per-repository repair allowance in minutes, on by default, consumed through the repair root's normal attempt allowance. Drop charges-as-commands, the grant lineage and effect text, the two-run ceiling and the intent aggregate, and let the existing judgment journal dedupe JEV calls. Delete the read-only eligibility copy, because admission's error already answers the question.

**Risk.** This changes spending policy. It overlaps whichever finder covers allowance.go and stall.go budgets.

### FEF-7: Project preparation is a six-step ceremony: digests copied back to the server, ids minted in the browser, and free-text 'evidence' the server never checks

Frontend: Factory settings, work, progress, request, intent screens. process-machinery. confirmed. ~500 lines, risk medium. Visible change: UI. Behaviour-preserving: yes. **Status:** owner decision.

Where: `frontend/spaces/sodaspaces-factory-settings.ts:161-293,328-344`, `frontend/spaces/sodaspaces-factory-settings.ts:663-865`, `frontend/spaces/sodaspaces-factory-settings.ts:1168-1435`, `frontend/spaces/sodaspaces-factory-settings.ts:148-154`, `internal/web/api/preparation_source.go:170-197,213-245`, `internal/web/api/preparation_decisions.go:59-77`, and 1 more.

**Simpler.** One 'Prepare project' panel: show the server's preview of setup.sh, check.sh and the selected paths, and one 'Approve and prepare' button. The server ties the approval to the digests it computed for that preview (it already recomputes and compares them in admitPreparationStart), mints the coder and reviewer preparation ids, replaces failed preparations itself, and reports ready, setup exit code and check exit code. Drop the evidence boxes and their hashed 'digests' in apiPreparationInitialApproval, the second 'approve' step, the digest copying, the browser-made ids and the client re-checks of the preview.

**Risk.** This is the owner's check before repository scripts run with privileged effects, and agents can edit setup.sh. Keep one explicit approval, tied to digests the server computed, that shows the file contents.

**Safe first step.** All edits are in frontend/spaces/sodaspaces-factory-settings.ts. 1. Delete the method approvalMatchesRequirement (1330-1332) and call the module function approvalMatches at 1335 and 1375. 2. Delete matchingPreparationRows (1362-1372). At 1377, call matchingPreparations(prep, safeText(prep.requirements), safeText(prep.approval), safeText(requirements.source_commit), safeText(requirements.setup_digest)) instead, using sourceReview's requirements. 3. Delete validSourcePreview, validSourceDigests and exactReturnedPaths (164-200) and the check at 691-692, which only re-validates the server's own response. Steps 1 and 2 behave identically at their call sites. Step 3 behaves identically whenever the server returns a well-formed preview.

### HPL-2: Daemon routing repeats each path across several route lists and goes through a 40-method trait with one real implementation and pass-through pops Req/Ops wrappers

Rust host platform: daemon, terminals, project, application, tailnet (lib/host non-factory, soda-project-maintain, forgejo/pg/console helpers, soda-tailnet, unix-http, wire-time). pass-through-wrapper. partially-confirmed. ~500 lines, risk low. Visible change: none. Behaviour-preserving: yes. **Status:** open (behaviour-preserving).

Where: `lib/host/src/daemon/routes.rs:16-133`, `lib/host/src/daemon/routes.rs:174-316`, `lib/host/src/daemon/admission.rs:16-89`, `lib/host/src/daemon/backend.rs:62-278`, `lib/host/src/dbackend/backend.rs:111-600`, `lib/host/src/dbackend/mod.rs:172-220`, and 3 more.

**Simpler.** Use one route table of (path, body limit, needs writer gate, requires clean path, handler fn), with handlers as DaemonBackend methods that decode domain types and call project::Runtime directly. Delete the ExecBackend trait, StubBackend, pops.rs (Req newtypes and Ops), ROUTE_TABLE, DaemonConfig::all_enabled and RouteOutcome::into_response. Keep the clean-path flag per route, or knowingly extend the check to /inspect, /connection and /account.

**Risk.** A mechanical refactor. The gmux_smoke test, which relies on the trait seam, would be deleted or rewritten.

**Safe first step.** Delete lib/host/src/pops.rs. In lib/host/src/dbackend/backend.rs, have DaemonBackend call domain::AccessKeys::decode, preparation::*::decode and project::Runtime { exec: project::Native, config } directly instead of crate::pops::*Req::decode and self.pops. Drop the `pops` field from dbackend/mod.rs. Also delete StubBackend (daemon/backend.rs:177-278), ROUTE_TABLE (routes.rs:16-63), DaemonConfig::all_enabled and RouteOutcome::into_response. Delete or trim lib/host/tests/gmux_smoke.rs and lib/host/tests/project_operations/, which depend on them. HTTP behaviour stays identical. Collapsing the trait and route lists into one table can follow as a second step, keeping a clean-path flag per route.

### RTY-1: Three owner-triggered issue/delivery retry kinds, plus completion retry, each with its own record function, pending selector and eligibility SQL; the implementation-retry kind covers only one narrow Stop-during-context-hold case

Retry kinds across store, control and UI. duplicated-layer. partially-confirmed. ~500 lines, risk medium. Visible change: UI. Behaviour-preserving: yes. **Status:** owner decision; coordinate with prod-hardening.

Where: `internal/factory/control/lifecycle_retry.go:18-95`, `internal/store/factory_explicit_retry.go:39-259`, `internal/store/factory_implementation_retry.go:1-309`, `internal/store/factory_delivery_retry.go:1-221`, `internal/store/factory_completion_retry.go:1-60`, `internal/factory/control/completion_retry.go:1-68`, and 8 more.

**Simpler.** Delete CommandRetryImplementation, factory_implementation_retry.go, the implementation branch in RetryRun, the PendingImplementationRetry branch in resolveDispatchRetry and priority_inventory, and the validateImplementationRetryChildTx call in the dispatch packet. A stopped held child is then retried through ordinary explicit Retry once its root closes, or as a fresh root. Fold the delivery-retry record/replay/selector into the explicit-retry functions and keep the delivery-specific eligibility predicate. Keep completion retry for its reassessment cycle, but trim its duplicate types (RTY-6).

**Risk.** Today a retried implement child keeps the finished investigation and its remaining time. Retrying it as a fresh root re-runs the Architect investigation, which costs one extra planning run. Explicit Retry also hands its command ID to the preparation custody path (preparationOwner/RetireRetry), so that hand-off must be kept or replaced. Tests that cover the removed paths (delivery_admission_test.go, factory_dispatch_phase_test.go, etc.) would be deleted with them.

**Safe first step.** Merge the command-insert, receipt-replay and finished-command lookup code that RecordDeliveryRetryCommand/PendingDeliveryRetry in internal/store/factory_delivery_retry.go duplicate from RecordExplicitRetryCommand/pendingExplicitRetry in internal/store/factory_explicit_retry.go into shared helpers. Keep each kind's eligibility SQL (delivery version vs issue/acceptance head) unchanged so observable behaviour stays identical. Removing the implementation-retry kind is a behaviour change (it re-runs the investigation) and should be a separate, owner-approved step.

### TRM-1: Terminal wire hand-coded to copy Go/Python byte formatting and checked again on the trusted side of each hop

Human terminal stack end to end (browser, Go API, host, in-project helper). duplicated-layer. partially-confirmed. ~500 lines, risk medium. Visible change: none. Behaviour-preserving: yes. **Status:** open (behaviour-preserving).

Where: `lib/host/src/terminal/wire.rs:1-470`, `lib/host/src/terminal/stream.rs:90-120`, `lib/host/src/terminal/text.rs:1-70`, `lib/host/src/terminal/native.rs:256`, `lib/host/src/dbackend/websocket.rs:157-160`, `internal/host/terminal.go:72,120-244`, and 5 more.

**Simpler.** Keep checks where trust changes: the Go API on browser frames (pumpTerminalInput plus InputValid), the root host on requests and input from the web process (request.valid, input_valid), the host on guest output (output_valid), and the guest on the argv it runs. Remove the copies on the trusted side: Go TerminalRequest.Valid in OpenTerminal, Go OutputValid in Receive, the second request.valid call in native.rs, and validHandshakeDimensions (use one dimensions helper). Replace the hand Visitors and encode_into in wire.rs and stream.rs with #[derive(Serialize, Deserialize)], #[serde(deny_unknown_fields)] and skip_serializing_if for omitempty, and drop the Go case-folding and duplicate-key emulation. Collapse the TS key-set re-check to what the UI actually needs.

**Risk.** Integration oracles in lib/host/tests (tcontrol_oracle) may assert byte-exact encodings and would need updating. A mismatch would show up as a stream that closes at once, so it is easy to see in a local open-terminal check.

**Safe first step.** In internal/web/api/terminal_registry.go, delete validHandshakeDimensions and call one shared dimensions helper (export host.terminalDimensions) from extension_terminal_stream.go:77, and make validTerminalGeometry use the same helper. In internal/host/terminal_client.go, drop the !f.OutputValid() clause in Receive, since the root host already ran output_valid on every frame it forwards. Then replace the hand TerminalFrame/TerminalState/TerminalRequest visitors and encode_into in lib/host/src/terminal/wire.rs with serde derives (deny_unknown_fields, skip_serializing_if for empty or zero fields), keeping the predicate functions, and adjust the tcontrol_oracle expectations if they are byte-exact.

### HFW-12: tcontrol keeps test stubs, test-only Go-mirror validators, a dead lifecycle validator and hand encoders in production code

Rust host: factory worker runtime. speculative-abstraction. partially-confirmed. ~480 lines, risk medium. Visible change: none. Behaviour-preserving: yes. **Status:** partly done (`d0435632`): validate_lifecycle_observation and helpers deleted; oracle-only validators remain with TST-4.

Where: `lib/host/src/tcontrol.rs:75-145`, `lib/host/src/tcontrol_enroll.rs:16-75`, `lib/host/src/tcontrol_provider.rs:27-75`, `lib/host/src/tcontrol_native.rs:29-42 (Transport/CliTransport only; RoundTrip/CliRoundTrip are used by production functions)`, `lib/host/src/tcontrol_policy.rs:445-456,560-566`, `lib/host/src/tcontrol_wire.rs:636-645 (decode_project_selection), 873-918 and 981-1112 (test-only view validators; HostView.validate at 919-980 is live), 1115-1350 (encoders), 1525-1578 (dead validate_lifecycle_observation)`, and 1 more.

**Simpler.** Delete validate_lifecycle_observation, decode_project_selection, validate_project_view, and the EnrollmentView/SettingsView/HostResult/EnrollmentResult/ProjectOptions validate methods, together with their oracle assertions. Keep HostView.validate, which runs in production. Remove cli_stub/local_stub/provider_stub, sync_hook, Provider::Stub, ProviderRequest and the Transport/CliTransport aliases, and delete the stub-driven oracle tests with them. Optionally replace the hand encoders with the serde derives the crate already depends on, keeping field order and case-insensitive decoding.

**Risk.** Most of tcontrol_oracle.rs (2,942 lines) depends on these seams. Policy allows deleting tests together with the code they test, but this drops coverage of real decision logic until an integration check replaces it.

**Safe first step.** In lib/host/src/tcontrol_wire.rs, delete validate_lifecycle_observation (around 1525-1578, zero callers), decode_project_selection (636+), validate_project_view (1099-1112), and the validate methods on EnrollmentView (875-918), SettingsView, HostResult, EnrollmentResult and ProjectOptions (981-1098). Keep HostView.validate (used at tcontrol_native.rs:750). Then delete the oracle assertions in lib/host/tests/tcontrol_oracle.rs that call them. Production behaviour stays identical.

### PLN-7: Dead decomposition-judgment path: 3-call 'triplet', options call, split/combine candidates, ScopedDecomposition

Intent to plan (Product Lead, Architect, research, decomposition). dead-code. confirmed. ~480 lines, risk low. Visible change: none. Behaviour-preserving: yes. **Status:** done (`0711047a`).

Where: `internal/factory/control/intent_decomposition_judgment.go:10-31,73-189`, `internal/factory/decomposition_options.go:13-303`, `internal/factory/control/intent_decomposition.go:48-68,79-83,133-137,162-168`, `internal/web/api/factory_settings.go:31-32`.

**Simpler.** Delete ErrIntentDecompositionReview, decompositionQuestions/Choices, decompositionEvidence, judgeDecompositionTriplet, AdviseIntentDecomposition, decompositionOptionsRubric and RecomposeDecomposition. Delete decomposition_options.go lines 13-303 (DecompositionCandidate, ScopedDecomposition, the candidate generators and the scoped validators). Delete scopedIntentTasks, and in prepareFirstIntentGraph and loadIntentDeltaBuild call scopedIntentTasksFromChosen directly, making chosen a required argument. Drop the dead ErrIntentDecompositionReview case in factoryCommandError. Keep effectiveDecompositionScope, JudgmentDecomposition, the Semantic* planner types and the intent_review_required code, which other paths still use.

**Risk.** No production caller. intent_generation_mechanics_test.go uses scopedIntentTasks and loses that case.

**Safe first step.** In internal/factory/control/intent_decomposition.go, delete scopedIntentTasks (lines 48-68). Replace the if/else at 133-137 and 163-167 with a direct call to c.scopedIntentTasksFromChosen, and remove the stale comment block at 79-83. Then delete from intent_decomposition_judgment.go everything except effectiveDecompositionScope and its imports: ErrIntentDecompositionReview, the questions/choices/evidence, judgeDecompositionTriplet, AdviseIntentDecomposition, decompositionOptionsRubric and RecomposeDecomposition. Delete lines 13-303 of internal/factory/decomposition_options.go, keeping IntentPlanningOptions and the Semantic* types from line 305 on. Remove the ErrIntentDecompositionReview case at internal/web/api/factory_settings.go:31-32. No tests reference the removed symbols.

### RBA-6: CoreOS stream and registry resolution implemented twice (release-build and acceptance)

Release build primitives, asset fetch/render tools, small install binaries and system units. duplicated-layer. confirmed. ~480 lines, risk low. Visible change: none. Behaviour-preserving: yes. **Status:** open (behaviour-preserving).

Where: `tools/acceptance/src/coreos.rs:1-494`, `tools/acceptance/src/vm/base.rs:61`, `lib/soda-release-build/src/coreos_stream.rs:1-287`, `lib/soda-release-build/src/coreos_registry.rs:1-195`, `tools/acceptance/Cargo.toml`.

**Simpler.** Add soda-release-build as a dependency of tools/acceptance. In vm/base.rs, call soda_release_build::coreos_stream::resolve_coreos_qemu(&config.architecture) instead of coreos::resolve_qemu, and map the error. Delete tools/acceptance/src/coreos.rs and its mod declaration. Keep the comparison against the live resolution in base.rs: reading only verified-base.json would drop the freshness check. Replacing release-build's raw_object! and Visitor code with derived structs is a separate, optional follow-up.

**Risk.** Tooling only. Acceptance would gain a crate dependency.

**Safe first step.** In tools/acceptance/Cargo.toml add `soda-release-build = { path = "../../lib/soda-release-build" }`. In tools/acceptance/src/vm/base.rs, replace `coreos::resolve_qemu(phase, &config.architecture)?` with `soda_release_build::coreos_stream::resolve_coreos_qemu(&config.architecture).map_err(|e| Error::msg(e.to_string()))?` and drop `use crate::coreos`. Delete tools/acceptance/src/coreos.rs and remove `mod coreos` from tools/acceptance/src/lib.rs. Release, image and the pass/fail outcome stay the same; only transport error wording and the curl logging go away.

### JDG-9: Inventory collection uses revision brackets, a double local capture and prepaid native-call accounting

Completion, reassessment, review and JEV judgments. unneeded-validation-or-digest. confirmed. ~450 lines, risk medium. Visible change: UI. Behaviour-preserving: no. **Status:** owner decision.

Where: `internal/factory/control/completion_inventory.go:1-464`, `internal/factory/control/completion_drive.go:136-251`, `internal/web/api/factory_completion.go:55-337`, `internal/factory/control/readiness_work.go:18-21,123-165`.

**Simpler.** List open issues with plain paged reads and read their bodies directly. Drop the revision bracket, the second local capture, the prepaid units and the cursor persistence. If drift must be detected before applying a change, compare issue content versions at apply time; the graph writer already conditions on the native revision.

**Risk.** Shared with stall and delivery repair; once JDG-1 lands, only those flows remain as users. Concurrent native edits during a collection would no longer force a restart.

### PTH-1: Python-emulation layer in soda-project-terminal (pyemit + StateValue + Python ==/int()/truthiness) copies retired .py byte for byte

In-project helpers (soda-project-terminal) and the host Muse launch runtime. duplicated-layer. confirmed. ~450 lines, risk low. Visible change: none. Behaviour-preserving: yes. **Status:** open (behaviour-preserving).

Where: `cmd/soda-project-terminal/src/pyemit.rs:1-117`, `cmd/soda-project-terminal/src/state_json.rs:1-202`, `cmd/soda-project-terminal/src/subscription_wire.rs:9-100`, `cmd/soda-project-terminal/src/factory_roles/emit.rs:1-101`, `cmd/soda-project-terminal/src/key_request.rs:8-40`, `cmd/soda-project-terminal/src/proto.rs:107-139`, and 9 more.

**Simpler.** Delete pyemit.rs and write output with serde_json::to_string / to_vec plus a newline. Replace StateValue with serde_json::Value, or with derived structs for the binding, reservation, subscription-profile and key-request records. Use one plain structural equality (Value ==) in place of both json_equal copies. Drop json_int and RevisionValue: take an i64 actor_id and a string revision. Use str::trim. Validate names with one char::is_control-based rule, or rely on the host's GO_CC/GO_CF check. Delete or loosen the byte-exact assertions in tests/cli.rs, tests/factory_roles_oracle.rs and the Go tests/build/project_account_test.go (and similar) together with the code. Fix the Cargo description.

**Risk.** Output bytes change (spacing, non-ASCII no longer escaped), but every reader is a JSON parser. State files already on disk still parse with serde_json. Duplicate-key 'last wins' becomes serde's default (error or last wins, depending on API), which no real writer produces. The factory_roles finder should coordinate, because emit.rs and inputs.rs also use pyemit.

**Safe first step.** Merge the two json_equal copies. Delete json_equal and its numbers_equal/unique_keys helpers from cmd/soda-project-terminal/src/factory_roles/emit.rs and have factory_roles call the one in subscription_wire.rs, or move one copy into state_json.rs. Both implement the same Python-== semantics, so output does not change. Also correct the stale ".py port" description in cmd/soda-project-terminal/Cargo.toml.

### PTH-3: Muse binary identity and freshness are re-checked across host and guest, with a receipt file, unreachable store-authoritative guest branches and a dead pinned-identity path

In-project helpers (soda-project-terminal) and the host Muse launch runtime. unneeded-validation-or-digest. partially-confirmed. ~450 lines, risk medium. Visible change: UI. Behaviour-preserving: yes. **Status:** owner decision.

Where: `lib/host/src/muse/provider.rs:21-140`, `lib/host/src/muse/provider.rs:179-245`, `lib/host/src/muse/execution.rs:10-32`, `lib/host/src/muse/execution.rs:47-95`, `lib/host/src/muse/stage.rs:1-90`, `cmd/soda-muse/src/store_resolve.rs:1-254`, and 5 more.

**Simpler.** Same as the finding. The host resolves and gates the selection once, with one shared gate function in provider-tools. It stages the bundle bytes when the guest sha256 differs, verifies once, and records the digest on the execution. Drop the receipt file and its byte comparison. Drop store_resolve.rs, so that --soda-exec execs /usr/local/libexec/soda/muse directly and --soda-check becomes a plain exec smoke test or goes away. Also delete the dead verify_guest_binary, expected_muse_identity and bundle_digest functions and the binary_version/binary_sha256 fields. Letting a Stale selection run is a separate policy decision.

**Risk.** soda-project-maintain also stages muse-selection.json (other scope), so remove it in step. Dropping the Stale block changes policy: an outdated Muse keeps running until refreshed. The digest pin and the ref-token retention for in-flight runs can stay.

**Safe first step.** These deletions change nothing a user or operator can observe: - In lib/host/src/muse/execution.rs, delete verify_guest_binary (lines 10-32). It has no callers. - In lib/host/src/muse/provider.rs, delete expected_muse_identity and bundle_digest. Remove the binary_version/binary_sha256 fields from MuseRuntime (runtime_types.rs) and its constructor. - In provider.rs, replace gate_snapshot with a call to the gate fresh_selection already uses. - In cmd/soda-muse/src/store_resolve.rs, delete the Ok(Some(_)) store-authoritative arm, the SODA_MUSE_STAGED_PATH/SODA_MUSE_RECEIPT_PATH overrides, gate_live, and the `Some(live)` arm. These cannot run in production because the guest never sees the provider store.

### DLV-7: Stall repair and delivery repair are two copy-pasted repair state machines that have already drifted apart

Publication, merge, correction, repair, acceptance, verification, lifecycle. duplicated-layer. confirmed. ~425 lines, risk medium. Visible change: none. Behaviour-preserving: yes. **Status:** open (behaviour-preserving).

Where: `internal/factory/control/repair_adoption.go:16-163`, `internal/factory/control/delivery_repair_adoption.go:16-179`, `internal/factory/control/repair_proposal.go:85-234`, `internal/factory/control/delivery_repair_proposal.go:21-171`, `internal/factory/control/repair_dispatch.go:25-135`, `internal/factory/control/delivery_repair.go:84-193`, and 7 more.

**Simpler.** Write one graph-repair adoption driver and one generation validator over a small subject interface: repository, progress generation, journal loader, sponsor check, hold/save and activate. Merge the two Activate*RepairIntent store functions around a shared transaction body. Use the atomic delivery-style judgment reservation for both. Keep per-subject prompts, rubrics and hold reasons as data. Unifying the stored record shape can come later.

**Risk.** Repair is rarely exercised. Merge the copies around the atomic delivery variant.

**Safe first step.** Merge driveStallRepairAdoption (internal/factory/control/repair_adoption.go:16-124) and driveDeliveryRepairAdoption (internal/factory/control/delivery_repair_adoption.go:16-124) into one driveGraphRepairAdoption. It takes a small subject struct or closures for repository, progress generation (record.ProgressGeneration or 0), journal loader, hold, save-cycle, sponsor check and activate. Do the same with validateStallRepairGeneration and validateDeliveryRepairGeneration, keeping the dispatch by cycle lookup. All hold reason codes are already the same, so behaviour does not change. Leave the proposal drivers and judgment reservations alone in this step, because their reasons and ordering differ.

### RBA-1: soda-release-build sits behind a one-impl trait, a thread-local bridge, optional hooks and three error types

Release build primitives, asset fetch/render tools, small install binaries and system units. duplicated-layer. confirmed. ~420 lines, risk low. Visible change: none. Behaviour-preserving: yes. **Status:** open (behaviour-preserving).

Where: `lib/soda-release-build/src/production.rs:12-50`, `lib/soda-release-build/src/lib.rs:31-98`, `lib/soda-release-tools/src/pipeline.rs:1-470`, `lib/soda-release-image/src/foreign.rs:29-125`.

**Simpler.** Merge soda-release-build into the crate that drives the pipeline. Nothing blocks this: soda-release-deliver depends on soda-release-image only as a dev-dependency, and cmd/soda-install never uses soda-release-image in its source. Call the build functions directly with a plain `&Runner` or command parameter instead of boxed Send+Sync hooks. Delete the foreign::Production trait, RealProduction, the thread-local, the conversion functions and the dead error fields. This is the soda-release-build side of REL-9 and should be done in the same change.

**Risk.** Build-time code only, with no runtime behaviour. The oracle tests that cover removed seams can be deleted with them.

**Safe first step.** In lib/soda-release-build/src/lib.rs, delete the dead `exit_code`/`signal`/`cancelled` fields and `exit_code()`, `signal()`, `is_cancelled()`, `with_exit_code`, `with_signal` and `Error::cancelled`. Error then holds only `message`, and nothing outside the crate calls any of these. Then, in production.rs, make `execute`/`capture` plain non-Option `Box<dyn Fn(..)>` with no Send+Sync bound. Drop the None branches in call_execute/call_capture and the `is_none()` checks in validate(). In lib/soda-release-tools/src/pipeline.rs, have make_production move a `runner.clone()` into each closure and delete the `CURRENT_RUNNER` thread-local and `with_current_runner`. Also replace `deliver_payload_of` with direct use of the payload. Update the closures in lib/soda-release-build/tests/oracle/production.rs to match. All of this is internal and gives identical observable behaviour.

### APP-8: Descriptor-walk-from-root layer in home.rs (FJ-847/855) guards root-owned ancestor directories against symlink races only root could cause

Application runtime: build, home, snapshots, backups, restore, factory delivery. unneeded-validation-or-digest. partially-confirmed. ~400 lines, risk medium. Visible change: none. Behaviour-preserving: yes. **Status:** open (behaviour-preserving).

Where: `lib/host/src/application/home.rs:30-116`, `lib/host/src/application/home.rs:129-378`, `lib/host/src/application/home.rs:693-703`, `lib/host/src/application/ingress.rs:29-107`, `lib/host/src/application/backups.rs:31-63`, `lib/host/src/application/runtime.rs:457-509`, and 1 more.

**Simpler.** Open root-owned directories (storage root, data/, setups/, snapshots/, routes/) with one open(O_DIRECTORY|O_NOFOLLOW) followed by an fstat owner/mode check, or with plain std::fs as restoration.rs already does. Drop the per-component walk from '/', the per-level dev/ino binds, the /proc/self/fd readback, the resolve_at/owned_directory double canonicalize, and the fd_matches_path rebinds. Write the Caddy route with fs::write (or a temp file plus rename) and compare the content. Keep the openat/O_NOFOLLOW/O_NONBLOCK, regular-file and owner checks wherever root reads or writes inside a container-writable directory: data/<id> (.writer-lock, snapshot copy) and setups/<id> (token.json, receipt.json).

**Risk.** setup.rs (HPL-8 scope) is the heaviest user, so coordinate with HPL. Do not loosen the checks on reads of container-owned data.

**Safe first step.** In lib/host/src/application/home.rs, replace walk_from_root/walk_components/descend_bound/open_trust_root/readlink_at/fd_proc_path (lines 129-332) with a single libc::open(path, O_RDONLY|O_DIRECTORY|O_NOFOLLOW|O_CLOEXEC) inside open_validated_dir and open_parent_dir, keeping the fstat owner/mode check. Delete fd_matches_path and its calls in owned_directory, open_parent_dir, open_validated_dir, ingress.rs anchor(), backups.rs root_anchored(), and the end of runtime.rs data_directory(). Leave open_child_opt, open_staged_file, make_child_dir, stat_opened and the setup.rs reads from the uid-10000 control_dir unchanged. Observable behaviour is the same for every non-racing (non-root-tampered) host layout.

### DLV-4: Publication, merge and correction repeat the same native-op helpers (adopt, outcome, call-error, gated UPDATE, authority) instead of sharing them

Publication, merge, correction, repair, acceptance, verification, lifecycle. duplicated-layer. partially-confirmed. ~400 lines, risk medium. Visible change: none. Behaviour-preserving: yes. **Status:** coordinate with prod-hardening.

Where: `internal/factory/control/publication_withdraw.go:242-299`, `internal/factory/control/merge_reconcile.go:88-333`, `internal/factory/control/publication_reconcile.go:130-395`, `internal/factory/control/publication.go:316-337`, `internal/factory/control/merge.go:172-210`, `internal/factory/control/correction.go:397-443,712-770`, and 7 more.

**Simpler.** Share the identical helpers. Write one adopt-observed and one outcome-of function over the common op fields, using a shared embedded struct or a generic. Replace the three *CallError functions with one that takes the wait and error sinks. Build the gate-checked registration UPDATE from one helper, parameterised by table and key column. Keep one stage constant set and one authority check, with merge's extra reviewer and BaseRef comparison passed in as a parameter. A full single driveOp is optional and should only be done if the per-kind hooks stay small. Corrections already reuse PublicationOperation.

**Risk.** This is a large refactor of the code that performs effects. The lost-reply lookup by persisted op id and the dispatch-gate registration check must be preserved exactly.

**Safe first step.** Merge publicationCallError (control/publication.go:318-329), mergeCallError (control/merge.go:174-185) and correctionCallError (control/correction.go:759-770) into one function that takes wait and error callbacks. Merge adoptObserved and adoptMergeObserved, and operationOutcomeOf and mergeOperationOutcomeOf, into one implementation over the shared op fields (Effect, Cancellation, Completion, Reason, Receipt, OperationID, InstallationID, Kind, ActorID, RepositoryID, Work.ActorID, Work.Repository). In the store, pull the shared gate-check SQL out of factory_publications.go:155-175 and factory_merges.go:132-151 into one builder that takes the table and key column. None of these change any reason string or outcome.

### FEF-9: Several parsers per endpoint and repeated fetches by sibling elements: /factory/access, the planning allowance, intent generation and the issue read are each fetched or parsed more than one way

Frontend: Factory settings, work, progress, request, intent screens. duplicated-layer. partially-confirmed. ~400 lines, risk low. Visible change: UI. Behaviour-preserving: yes. **Status:** owner decision; coordinate with prod-hardening.

Where: `frontend/spaces/sodaspaces-factory-access-response.ts:1-194`, `frontend/spaces/sodaspaces-factory-progress-response.ts:14-23,438-494,695-818`, `frontend/spaces/sodaspaces-intent-response.ts:124-160,279-313`, `frontend/spaces/sodaspaces-factory-progress-view.ts:45-77`, `frontend/spaces/sodaspaces-factory-work.ts:719-737,827-895,1285-1301,1328-1341`, `frontend/spaces/sodaspaces-factory-work-journey.ts:53-82`, and 7 more.

**Simpler.** Keep one reader per endpoint (one FactoryAccess parser, one planningAllowance, one intentGeneration) and delete the unused intentGeneration and generationBinding in progress-view. Move the identical digest, counter, text, decimalID and readinessStates helpers into sodaspaces-api.ts. Have the issue-detail parent fetch the issue and access once and pass them to the journey, acceptance and completion-retry children as properties, with one refresh on the parent. Have settings pass its access object to optional-effects. Inline single-use view models only where that clearly shortens the code.

**Risk.** Mechanical. Children lose their own refresh buttons in favour of one refresh on the parent.

**Safe first step.** Delete the uncalled `intentGeneration` function and its sole helper `generationBinding` from frontend/spaces/sodaspaces-factory-progress-view.ts (about lines 55-77), keeping the `IntentGeneration` interface that WorkOverviewInput uses. The prod-hardening branch also edits this file, so expect a small conflict. Then export the byte-identical `counter` (`Number.isSafeInteger`, >= 0) and the 64-hex `digest` type guard from sodaspaces-api.ts. Replace the local copies in sodaspaces-factory-progress-response.ts, sodaspaces-factory-work-response.ts and sodaspaces-factory-work-journey-response.ts, and `counter` in sodaspaces-factory-acceptance-response.ts. Leave the stricter `text` copy in intent-response alone.

### FEP-7: Lost-reply reconciliation reads, error-code notice tables and completion coalescing in App controls; command-record reconcile and per-draft plan replay pool in Work

Frontend: Workspace, project, applications, terminals, layout, Forgejo and Tailnet pages. speculative-abstraction. partially-confirmed. ~400 lines, risk medium. Visible change: UI. Behaviour-preserving: no. **Status:** owner decision.

Where: `frontend/spaces/sodaspaces-application-delivery.ts:7-62,225-316,332-380,431-498`, `frontend/spaces/sodaspaces-application-setup.ts:12-30,180-227,253-321`, `frontend/spaces/sodaspaces-applications.ts:869-985`, `frontend/spaces/sodaspaces-project-work.ts:242-322,560-596`.

**Simpler.** If a reply fails or is lost, show one 'Not confirmed - refresh to see the current state' message and call refresh() once. - Drop readDeliveryBuild, readApplyState and reconcileExpiredSetup, and replace the four code-set tables with a few status-based messages (401, 403, 409, other). - Replace the four setupComplete* fields and the settle loop with a single refresh() on application-setup-complete, followed by one plain notice. - In Work, drop the factory/commands/{id} reconcile branch from requestAction. Leave 'Retry same request', which reuses the same command id, and let server-side idempotency handle duplicates. - Keep the 4 s repair-progress polling: it shows real progress, not reconciliation. - Serve plan status in the intents list (small Go change), then drop the per-draft GET pool and the timeout-hole handling.

**Risk.** In real lost-reply cases the user sees a generic message and refreshes. Server-side idempotency is unchanged. Browser tests asserting the specific reconciliation copy go with it. Listing plan status inline needs a Go list-field addition.

### HFW-10: Run, lease and output types and validators defined twice in the Rust host, bridged by cv_* converters and single-production-impl trait seams

Rust host: factory worker runtime. duplicated-layer. partially-confirmed. ~400 lines, risk medium. Visible change: none. Behaviour-preserving: yes. **Status:** open (behaviour-preserving).

Where: `lib/host/src/factory/run.rs:14-21,96-191,262-391`, `lib/host/src/terminal/factory/run.rs:9-107,296-338`, `lib/host/src/preparation/mod.rs:28-29,53-90`, `lib/host/src/factory/identity.rs:1-247`, `lib/host/src/terminal/lease.rs:26,438`, `lib/host/src/daemon/broker.rs:6-58,95`, and 10 more.

**Simpler.** Keep one FactoryRun (the factory one; the terminal copy is a subset), one Lease/AcquireRequest (terminal::Lease), one OutputSlice and one validator/constant set (preparation:: and factory::run), and have terminal code import them. Delete the cv_* converters, the empty FactoryBroker::reconcile_lease and its 7 calls, and the gmux_smoke check that the two prompt bounds agree. Optionally make Factory concrete over TerminalSeam/BrokerClient (Executor stays generic), turning TerminalSeam's reservation logic into plain methods and rewriting or deleting the factory_roles_oracle.rs trait fixtures.

**Risk.** Mechanical; the test fixtures that implement the traits would go too.

**Safe first step.** In lib/host/src/terminal/factory/run.rs, delete FACTORY_HARNESS_CODEX/MUSE, valid_harness_family, valid_factory_role, valid_preparation_id, valid_digest, valid_commit, valid_factory_run_id, valid_harness_version, MAX_FACTORY_PROMPT/MAX_REASSESS_PROMPT/MAX_DECOMPOSE_PROMPT and max_staging_prompt_for_phase, and re-export or import them from crate::preparation and crate::factory (msp/wire.rs uses MAX_REASSESS_PROMPT). Delete MAX_FACTORY_OUTPUT_* in terminal/factory/output.rs in favour of the factory::run constants. Delete the gmux_smoke.rs:2218-2234 check that the two copies agree. Remove the empty `fn reconcile_lease` from the FactoryBroker trait (factory/mod.rs:221), from daemon/broker.rs:95 and from the oracle fixture, plus its 7 calls in factory/finish.rs and factory/stop.rs. All values and error strings are identical, so behaviour does not change.

### HPL-7: soda-project-maintain re-implements host config parsing/validation, the release-payload loader, IP-prefix parsing and hex encoding, despite depending on soda-host and soda-release-deliver; the copy has already drifted (it rejects tailnet_image)

Rust host platform: daemon, terminals, project, application, tailnet (lib/host non-factory, soda-project-maintain, forgejo/pg/console helpers, soda-tailnet, unix-http, wire-time). duplicated-layer. confirmed. ~400 lines, risk low. Visible change: none. Behaviour-preserving: yes. **Status:** open (behaviour-preserving).

Where: `cmd/soda-project-maintain/src/config.rs:1-34`, `cmd/soda-project-maintain/src/config_wire.rs:1-142`, `cmd/soda-project-maintain/src/config_validation.rs:1-120`, `cmd/soda-project-maintain/src/release.rs:1-37,44-113`, `cmd/soda-project-maintain/src/release_wire.rs:1-10`, `cmd/soda-project-maintain/src/network.rs:1-35`, and 4 more.

**Simpler.** In main.rs, call soda_host::daemon::config::load_config(&o.config, RELEASE_PATH) and use soda_host::daemon::config::Config. Replace the two other load_release_payload callers with soda_release_deliver::payload::load plus Role::ProjectOs.binding(&p.images). Replace hex_encode with soda_host::sha256::hex_lower and import sha2::Sha256 directly. Delete config.rs, config_validation.rs, network.rs, sha256.rs, release_wire.rs, the Visitor part of config_wire.rs and the loader/apply_release_images part of release.rs. Keep go_quoted (move it into options.rs) and is_hex_string.

**Risk.** load_config also sets canlang_runtime and the tailnet pins from the release. Maintain would simply ignore those.

**Safe first step.** In cmd/soda-project-maintain/src/filesystem.rs and provider_stage.rs, replace super::sha256::hex_encode with soda_host::sha256::hex_lower (identical code) and import sha2::Sha256 directly, then delete cmd/soda-project-maintain/src/sha256.rs and its mod line in main.rs. Next, replace release.rs load_release_payload and release_wire.rs with soda_release_deliver::payload::load(path) plus Role::ProjectOs.binding(&payload.images).config for the three callers (release.rs apply_release_images, terminal_support.rs:87, release_target.rs:197). Every failure already collapses to a fixed message, so observable output stays the same.

### VRF-6: Journey-observation subsystem: two MCP tools, paging, a Feed card and six re-validations for notes that barely affect verification

Test Pilot / app verification (Go factory+control+store+MCP, Rust host, in-project helper). speculative-abstraction. confirmed. ~380 lines, risk low. Visible change: UI, model tools. Behaviour-preserving: yes. **Status:** owner decision; coordinate with prod-hardening.

Where: `internal/factory/verification_journey.go:1-166`, `internal/store/factory_verification_journey.go:1-220`, `internal/web/api/factory_mcp_verification_journey.go:1-67`, `internal/web/api/factory_mcp.go:457-459`, `internal/web/api/factory_mcp.go:735-736`, `internal/web/api/factory_mcp_bound.go:109-114`, and 8 more.

**Simpler.** Drop verification_scope, because the Test Pilot prompt already carries accepted_scope and the delivery subject. Also drop the journey:<id> references and their re-validation at finish, adoption and delivery repair. Then either let the Test Pilot put journey steps and expected/observed text in the finish report, or keep record_journey_result as a plain TP03 communication post with no bindings, no back-references and no re-validation.

**Risk.** Nothing downstream reads journeys except the cite check and TP03 rendering.

**Safe first step.** In internal/store/factory_verification_journey.go, make these deletions: - In VerificationScope, remove the per-row DecodeVerificationJourney/ValidateFor re-validation. Decode the payload without re-checking scope. - In VerificationScope, remove the 'projection is incomplete' total-vs-records comparison. - In validateVerificationJourneyRefsTx, remove the `digest != factory.MCPInputDigest(raw)` self-digest comparison and the payload_hash column from its SELECT. The rows are immutable and were written atomically after validation, so these checks fire only on database corruption. Outcomes stay identical. Removing the tools themselves changes the contract, so that step needs an owner decision.

### BND-10: lib/host hand-encodes its JSON wire and state to match Go encoding/json byte for byte instead of using serde Serialize

Boundary machinery: strict JSON, digests, receipts, revision checks (cross-cutting). other. partially-confirmed. ~350 lines, risk medium. Visible change: none. Behaviour-preserving: yes. **Status:** open (behaviour-preserving).

Where: `lib/host/src/json/mod.rs:241-276`, `lib/host/src/json/mod.rs:31-66`, `lib/host/src/factory/receipt.rs:63-108`, `lib/host/src/factory/state.rs:45`, `lib/host/src/factory/events.rs:138-170`, `lib/host/src/tcontrol_wire.rs:11`, and 8 more.

**Simpler.** In lib/host, replace the hand-written encode/encode_into functions with #[derive(Serialize)] plus skip_serializing_if for omitempty fields, and plain serde_json output. The Go and TS consumers decode by field name. The one exception is TaskEvent in factory/events.rs, whose sha256 Go recomputes in internal/project/factory_events.go. Keep that encoder until the chain digest is dropped together with BND-4. Delete or update the byte-exact oracle vectors. Separately, drop the CPython json.dumps emulation in soda-project-terminal (pyemit.rs) in favour of plain serde_json. Relax deny_unknown_fields only on true component-to-component DTOs, not on release-manifest readers.

**Risk.** I need to confirm which encoders feed a hash compared on the Go side (receipts and run records). Change those together with BND-4.

**Safe first step.** In lib/host/src/factory/receipt.rs, replace the hand-written FactoryReceipt::encode (lines ~63-108) and its nested FactoryRun/Lease/binding encode_into helpers with #[derive(Serialize)]. Use skip_serializing_if for exit_code, generation, output, retirement and reason, then encode with serde_json::to_string. Nothing hashes these bytes; they are written as a run record and read back by name-keyed decoders on the Rust and Go sides (internal/factory/run.go). Do the same for terminal/wire.rs TerminalFrame::encode_into. Leave factory/events.rs TaskEvent::encode untouched: Go recomputes its digest.

### PRJ-1: Executor: 4 trait methods x 3 hand-forwarded impls, a status-only twin, typed and legacy error paths, plus a second process runner in companion

Project environment runtime on the host (create/join/start, executor, domain, ssh, accounts). duplicated-layer. confirmed. ~350 lines, risk medium. Visible change: none. Behaviour-preserving: yes. **Status:** open (behaviour-preserving).

Where: `lib/host/src/project/executor.rs:1-372`, `lib/host/src/project/executor.rs:515-558`, `lib/host/src/project/executor.rs:580-877`, `lib/host/src/tailnet/companion/execute.rs:15-110`, `lib/host/src/tailnet/forgejo.rs:30-55`, `lib/host/src/tcontrol_native.rs:760-775`, and 2 more.

**Simpler.** One method: `fn run(&self, stdin, cmd, args, deadline, limits: Limits) -> Result<Vec<u8>, ExecError>`, where `Limits { stdout: Option<usize>, stderr: Option<usize>, grace: Option<Duration> }` implements Default and `enum ExecError { StdoutLimit, Failed(String) }`. Callers that only want a String use `.map_err(ExecError::into_string)`. Drop NativeStatusOnly and the status_only parameter: forgejo.rs formats its own error, or drops it. Drop is_host_native and companion's run_native_command, and call Native with a 64 KiB stdout limit and stderr 0, mapping any error to ERR_UNCONFIRMED. Drop cleanup_confirmed so that any stdout overflow is StdoutLimit. Keep the waitid/killpg core of the loop.

**Risk.** Process lifecycle code is subtle: keep the existing spawn, drain, poll and killpg core and only collapse the API surface and the error-mapping branches. The gmux_smoke integration tests reference run_bounded_typed and NativeStatusOnly and need updating.

**Safe first step.** In lib/host/src/project/executor.rs, replace the five trait methods with one `run(stdin, cmd, args, deadline, policy: CapturePolicy)`, with CapturePolicy made pub and Default. Have that method return RunError mapped into BoundedError, and leave String conversion to a helper. Delete the per-method forwarding in Native, NativeStatusOnly and the `&E` blanket impl. Make NativeStatusOnly a `Native { status_only: bool }` field, or keep it as a one-line impl that sets the flag. Delete completion_text and call refusal_text directly. Update the roughly 30 call sites (source.rs, tcontrol_native.rs, terminal/*, factory/*, application/*, account, tailnet/host.rs) and the test fakes and gmux_smoke callers. The same deadline and limits produce identical error strings and outcomes. Leave the companion run_native_command, the cleanup_confirmed precedence and the status_only removal for a second step: they change lifecycle details and wording.

### REL-11: CI has a tag-qualify-then-fast-forward push gate, a reuse admission job, and two overlapping workflows

Release, install, packaging, CI and freshness machinery. process-machinery. confirmed. ~350 lines, risk low. Visible change: none. Behaviour-preserving: no. **Status:** open (behaviour-preserving).

Where: `scripts/push.sh:1-75`, `scripts/ci-reuse.sh:1-24`, `scripts/check-push.sh:1-40`, `.github/workflows/ci.yml:31-186`, `.github/workflows/deep-ci.yml:1-267`, `tests/build/ci_admission_test.go:1-149`, and 2 more.

**Simpler.** One workflow on push and PR that runs check-source plus the cargo tests. Change the GitHub branch-protection settings so main can be pushed directly, or require checks on PRs instead. Delete push.sh, ci-reuse.sh, ci_admission_test.go, the reuse and aggregator jobs, and the ci/soda-main-* and ci/soda-full-* tag triggers. Merge deep-ci.yml into that single workflow or delete it, and update testing.md and release.md.

**Risk.** Allows a red main for a zero-user product, which AGENTS.md treats as acceptable.

### REL-8: Fountain revision pin needs a release-scanning helper (check-fountain-pin) and four jq shape checks because the matching release URL is supplied by hand

Release, install, packaging, CI and freshness machinery. unused-configuration. partially-confirmed. ~350 lines, risk low. Visible change: none. Behaviour-preserving: yes. **Status:** open (behaviour-preserving).

Where: `fountain-source.json:1-4`, `lib/soda-release-tools/src/fountain_pin.rs:1-225`, `lib/soda-release-tools/src/artifacts.rs:13-30,75-80,424-436`, `lib/soda-release-tools/src/lib.rs:20`, `lib/soda-release-tools/tests/cli/soda_artifacts.rs:123-325`, `lib/soda-release-tools/README.md:334-341`, and 9 more.

**Simpler.** Pin the Fountain release, either as a tag in fountain-source.json or as the fork's latest release per the freshness policy. Derive the release.json URL from the tag and the revision from its source_commit, so operators no longer pass SODA_FOUNTAIN_RELEASE_URL by hand. Then delete check-fountain-pin with its wiring, tests and README text, and remove the four jq shape-check lines (the checkout step and the strict Rust parsing already reject a bad pin). Keep verify_fountain_sdk and the docker and publish candidate cross-checks, pointed at the derived revision.

**Risk.** The go.mod SDK replace still needs a local checkout at the matching commit, so CI keeps the checkout step.

**Safe first step.** Delete the `jq -e 'keys == [...]' fountain-source.json >/dev/null` line from .github/workflows/ci.yml (line 79) and from .github/workflows/deep-ci.yml (lines 60, 120, 185), keeping the jq -r output lines. A malformed pin still fails at checkout and in the strict Rust parsers, so only the CI error text changes.

### EXT-2: Native authority is re-verified many times per request (pre- and post-effect, terminal heartbeat) and /api/session and /api/me/* use a parallel admission chain

Fountain extension and its backend. unneeded-validation-or-digest. partially-confirmed. ~330 lines, risk medium. Visible change: UI. Behaviour-preserving: yes. **Status:** owner decision.

Where: `internal/web/api/extension_native.go:22-175`, `internal/web/api/extension_native.go:289-292`, `internal/web/auth/extension_service.go:30-281`, `internal/web/auth/extension.go:17-53`, `internal/web/api/extension_terminal_authority.go:30-126`, `internal/web/api/extension_terminal.go:50-93`, and 12 more.

**Simpler.** Use one admission function for every extension route: ExtensionAuthority once (keep its method==action and actor-equality checks), the contribution binding, a user upsert, and the authority stored in ctx. Register /api/session and /api/me/* on the same mux and remove the duplicated identity, origin and generation checks in auth.serveExtension; keep its JSON Content-Type check for mutations. Delete the generation-header compares, validSecFetchHeaders, extensionActor and nativeTerminalSessionActor. Drop post-effect re-checks: tailnetSession after the host call, extensionTerminalCurrent after TerminalStates, and revalidateExtensionAuthority in the GET and forgejo-keys paths. Remove session_current from the access view, or leave it constant. Terminals check membership and write permission once at open and rely on Fountain's monitorStream for liveness, plus at most a membership check in the heartbeat. Keep the handshake generation compare that Fountain delegates.

**Risk.** Security-sensitive: exactly one verified callback must remain at the service because the socket is world-connectable, and each handler's business authorization (repository role, membership, SiteAdmin) stays. A logout racing a request would get the completed result instead of a 401.

**Safe first step.** Delete the checks that cannot change any outcome, keeping all callback re-checks for now. (1) extensionGenerationCurrent and its call in extension_native.go:32,93-100. (2) extensionSessionGenerationMatches and the 409 branch in auth/extension_service.go:67-70,201-204. (3) extensionTerminalGeneration and the generationHeader parameter in extension_terminal_authority.go:30-33,62-65 and its callers. (4) validSecFetchHeaders in terminal_registry.go:105-117 and its uses at terminal_registry.go:126 and factory_output.go:34. (5) extensionActor in extension_native.go:289-292: make tailnetAdminAuthorization read requestExtensionAuthority directly. (6) nativeTerminalSessionActor in extension_terminal_authority.go:124-126 and its call at extension_terminal.go:80. (7) Make extensionTerminalOperation call registerExtensionTerminalPeer instead of duplicating it. Steps 1-4 compare values that Fountain sets itself or that the caller controls; steps 5 and 6 compare the authority with a session built from that same authority; step 7 merges duplicate code. Observable behaviour stays the same.

### HFW-7: Harness bytes hashed four times per launch, the pin round-trips through Go for a race-prone check, provider selection is duplicated, the sidecar recovery path is unreachable, and the muse_harness config is unused

Rust host: factory worker runtime. duplicated-layer. partially-confirmed. ~330 lines, risk medium. Visible change: none. Behaviour-preserving: yes. **Status:** coordinate with prod-hardening.

Where: `lib/host/src/factory/launch.rs:40-55,420-436`, `lib/host/src/dbackend/factory_terminal.rs:69-145,215-258,442-460`, `lib/host/src/terminal/factory/bound.rs:25-307`, `lib/host/src/muse/provider.rs:21-110`, `lib/host/src/terminal/factory/muse/reserve.rs:185-240`, `lib/host/src/daemon/config.rs:27-38,85-116,296-331`, and 9 more.

**Simpler.** reserve resolves the current bundle once (one hash), stages it, and releases the store reference right after staging. It returns the harness, version and sha, and Go records those as AdoptedHarness instead of pre-reading and echoing a pin. That removes current_pin, the launch() pin check and the race refusal. Drop the HarnessBind 'previous' recovery and restore branches, bind_recorded and reserve_failure_is_definitive, since reserve runs once per run. Keep one fresh_selection helper, shared by bound.rs and muse/provider.rs, or add a provider-tools variant of resolve_new that returns the Selection. Delete the muse_harness, muse_harness_sha256 and muse_harness_version config plumbing and the soda-muse pre-updater staged-binary fallback. Gate Muse on muse_socket rather than muse_sha256. Keep codex_harness_sha256, which the identity harness uses, but stop stamping it into factory catalog metadata.

**Risk.** Go reads the pin in checkHarnessFor (internal/factory/control/dispatch_selection.go) and records AdoptedHarness from it; that must move to the launch result. The operator config docs change.

**Safe first step.** Delete fresh_selection() in lib/host/src/muse/provider.rs (lines ~21-53) and call a shared bound::fresh_selection_for(Provider::Muse) from fresh_provider_snapshot instead. With Provider::Muse it gives the same gate and the same messages, so nothing observable changes. Then delete the unused muse_harness / muse_harness_sha256 / muse_harness_version fields: the struct fields, parsing and validation in lib/host/src/daemon/config.rs; the threading in lib/host/src/main.rs, lib/host/src/dbackend/mod.rs, lib/host/src/dbackend/factory_terminal.rs:495-497, lib/host/src/terminal/agent.rs:26-28 and lib/host/src/muse/resolve.rs; the cmd/soda-project-maintain config_wire.rs and config_validation.rs entries; and the matching test initialisers in lib/host/tests and cmd/soda-project-maintain/src/support_root.rs. No installer sets these keys.

### RBA-4: Confined-fd filesystem layer and strict JSON reader exist only to read live_inputs.json; files::File decoder is dead

Release build primitives, asset fetch/render tools, small install binaries and system units. unneeded-validation-or-digest. confirmed. ~330 lines, risk low. Visible change: none. Behaviour-preserving: yes. **Status:** open (behaviour-preserving).

Where: `lib/soda-release-build/src/confined_files.rs:1-190`, `lib/soda-release-build/src/json_input.rs:1-62`, `lib/soda-release-build/src/live_inputs.rs:1-23`, `lib/soda-release-build/src/files.rs:12-90`, `lib/soda-release-build/src/lib.rs:10,19`, `lib/soda-release-build/tests/oracle/inputs.rs`.

**Simpler.** Make read_live_inputs = serde_json::from_slice(&std::fs::read(path)?) followed by valid_live_inputs. Delete confined_files.rs, json_input.rs (including parse_integer_token), the files::File struct and its hand-written Deserialize impl, the two `pub mod` lines in lib.rs, and the oracle tests that exercise them.

**Risk.** Single local reader of a file the same pipeline wrote. No concurrent or untrusted writer.

**Safe first step.** Delete the files::File struct and its Deserialize impl (lib/soda-release-build/src/files.rs lines 12-90, plus the now-unused serde imports), parse_integer_token in json_input.rs, and the File-based tests in tests/oracle/inputs.rs. Nothing outside tests uses them, so behaviour is identical. Then make live_inputs::read_live_inputs use std::fs::read + serde_json::from_slice + valid_live_inputs, delete confined_files.rs and json_input.rs, remove their `pub mod` lines in lib.rs, and drop the matching oracle tests.

### GOM-1: Forgejo snapshot reads re-hash host digests, validate each request two or three times, and bracket write-path reads that conditional submit already guards

Other Go packages (forgejo, acceptance, project, host, tailnet, application, config, strictjson, filelock, avatar, cmd/soda-dashboard, cmd/soda-tailnet, cmd/soda-extension). unneeded-validation-or-digest. partially-confirmed. ~320 lines, risk medium. Visible change: none. Behaviour-preserving: yes. **Status:** open (behaviour-preserving).

Where: `internal/forgejo/snapshot.go:59-95`, `internal/forgejo/snapshot.go:101-195`, `internal/forgejo/snapshot_issue.go:31-90`, `internal/forgejo/snapshot_pull.go:27-109`, `internal/forgejo/snapshot_review_threads.go:22-152`, `internal/forgejo/snapshot_merge_eligibility.go:16-72`, and 6 more.

**Simpler.** Drop the Soda-side digest recomputation in the valid* functions. Keep the Visible/redaction, Complete and Total checks, because Complete=false is a real host outcome that is surfaced as incomplete_evidence or an inventory hold. Validate each request once, in the transport, and drop the extra ValidateRequest call in BracketedRead. On write paths (review, merge, merge_completion, checks, lifecycle, issue_graph, delivery), drop the second ReadNativeRevision call and rely on ExpectedNativeRevision at submit. Optionally, have the host return the revision inside the snapshot. Keep the bracket on read-only paths (acceptance evidence, dispatch inputs, completion inventory) unless their stale/busy refusals are also dropped on purpose.

**Risk.** Some callers compare the revisions of two reads (review.go:72 snapshot.Revision != issue.Revision). Those comparisons become 'both reads returned a revision' and rely on submit-time ExpectedNativeRevision. Getting the revision from the snapshot itself needs a small change to the sibling SDK and host. Dropping digest checks removes detection of a host bug, not of an adversary.

**Safe first step.** In internal/forgejo/snapshot_issue.go (validIssue, validCommentPage), snapshot_pull.go, snapshot_review_threads.go and snapshot_repository_issues.go, delete only the `X.Digest != ContentDigest(X.Content)` recomputation comparisons. Keep the presence, redaction, Complete and Total checks. The host computes these digests with the same function over the same bytes, so observable behaviour stays identical. Also delete the repeated ValidateRequest call at the top of BracketedRead (snapshot.go:62-64). BackgroundSnapshotReader.ReadSnapshot runs the same check on every bracketed read, and the bracket's own nil-reader guard stays.

### RUN-2: Go re-checks the host's task journal (chain, binding, lifecycle) a second and third time in the import fold and the store; the fold's release/retire decisions are never read

Run end: task events, checkpoints, results and settlement (RUN). duplicated-layer. partially-confirmed. ~320 lines, risk medium. Visible change: none. Behaviour-preserving: yes. **Status:** safe first step done (`8b33da82`): the fold returns one Stop flag. The Go chain/binding re-checks and the store re-validation remain.

Where: `internal/factory/control/task_execute.go:14-785`, `internal/factory/control/task_progress.go:16-79`, `internal/factory/control/task_checkpoint.go:80-175,275-300`, `internal/store/factory_planning_revisions.go:100-260`, `internal/store/factory_planning_revisions.go:478-655`, `internal/factory/control/publication_plan_gate.go:100-170`.

**Simpler.** Delete the dead outputs: NewlyReady, RetireChildren, Contained, Stopped, TaskExecutionDecisions, readyPlanTasks and activeChildren; return stop = contained||stopped||overLimit. Drop the Go re-check of prev_digest, invocation and plan digest per event (host append already enforces these) and keep a plain sequence check against the cursor. Fold lifecycle errors into one error. Stop the store from re-loading the plan, re-checking validPlanTaskTransition and the Parent check, and running CAS on a page the importer has just produced: a plain upsert inside one transaction will do. Check plan check-key membership once, at import, not in the store and again in the publication gate. Keep the per-receipt 'type@seq:digest' Events refs, or replace them in task_checkpoint.go too, because checkpoint ordering reads them. Remove the unused singular SavePlanTaskReceipt, the PlanTaskReceipt getter and the never-written InputDigest field.

**Risk.** A corrupt or forged journal would no longer be caught in Go. But the journal comes from our own host over the local host socket, and that host already validates on append. Task-failure containment and over-limit stop must keep working.

**Safe first step.** In internal/factory/control/task_execute.go, delete the NewlyReady and RetireChildren fields (collapse TaskExecutionDecisions to StopRun and LimitViolation, or to a single stop bool read by task_progress.go:78), delete TaskExecutionResult.Stopped and Contained, delete readyPlanTasks (lines 718-752) and activeChildren (754-764), and simplify finishExecution to match. None of these has a reader anywhere in the repo, so behaviour is identical. Then in internal/store/factory_planning_revisions.go, delete the singular SavePlanTaskReceipt (lines 478-485) and the PlanTaskReceipt getter (642-655). Point the tests that call them (factory_planning_revisions_test.go, factory_completion_material_test.go, planning_admission_test.go) at SavePlanTaskReceipts and PlanTaskReceipts, or delete those test cases.

### DLV-6: Four per-pass report type families carry 234 wait/error reasons that every production caller throws away, so delivery waits are invisible

Publication, merge, correction, repair, acceptance, verification, lifecycle. dead-code. confirmed. ~300 lines, risk low. Visible change: none. Behaviour-preserving: yes. **Status:** open (behaviour-preserving).

Where: `internal/factory/control/publication.go:35-70`, `internal/factory/control/publication.go:147-159,214-220,318-329`, `internal/factory/control/merge.go:33-70,131-137,174-185`, `internal/factory/control/correction.go:15-56,239-250,759-770`, `internal/factory/control/checks_pass.go:19-60`, `internal/factory/control/publication_reconcile.go:91-103`, and 4 more.

**Simpler.** Replace the four families with one small sink: each driver returns (progressed bool, reason string). Store the latest wait reason on the publication or merge row, or post one Feed step when it changes, and log real errors with slog. Delete the Link and Report types and the converters between them.

**Risk.** Pure plumbing. The few length checks become bool returns.

**Safe first step.** Delete the write-only parts in internal/factory/control: - the PublishLink, MergeLink, CheckLink and CorrectionLink types and the Published/Merged/Checked/Corrected link slices, keeping a counter or bool only where control flow reads one (publication.go:157 and correction.go:170 read Corrected) - the Detail fields and the Unavailable flags (publication.go:82, merge.go:82, checks_pass.go:72, review_cycle.go:38), which nothing reads - all JSON tags on these report types Then change PublishPass, CheckPass and MergePass to return nothing, since coordinator.go:196-204 and publication_withdraw.go:95-97 ignore the results. Delete the test assertions that read the removed fields (integration_helpers_test.go:537, publication_fixture_test.go:463, merge_fixture_test.go:163, review_tuple_test.go).

### DLV-9: Publication/merge writes re-validate the whole blob, diff it field by field against the stored row, and re-check authority heads in SQL; the 'outstanding' predicate is pasted three times

Publication, merge, correction, repair, acceptance, verification, lifecycle. unneeded-validation-or-digest. partially-confirmed. ~300 lines, risk medium. Visible change: none. Behaviour-preserving: yes. **Status:** coordinate with prod-hardening.

Where: `internal/store/factory_publications.go:59-209`, `internal/store/factory_publications.go:221-482`, `internal/store/factory_publications.go:495-514,538-557,613-629`, `internal/store/factory_publications.go:589-591`, `internal/store/factory_merges.go:78-152`, `internal/store/factory_merges.go:239-451`, and 5 more.

**Simpler.** On write: marshal, then UPDATE ... WHERE revision=$prev. Keep the dispatch-gate EXISTS on registering writes. Keep the acceptance-withdrawal, requirement-head and approval-head EXISTS unless those changes start actively withdrawing work, because nothing else fences them atomically. Validate when the value is built, not on every write. Remove publicationUpdateAllowed and the merge non-replacement diffs. Put the publication 'outstanding' predicate in one shared SQL fragment. Remove CandidateTuple.Advance's observedRevision argument, but keep Tuple.Revision itself, since it sits in the reviewer context JSON.

**Risk.** Revision CAS and the dispatch-gate check must stay. Grant and acceptance changes already withdraw outstanding work actively (grants.go:127-128, acceptance.go:200-202), so the authority-head re-checks are a third layer.

**Safe first step.** In internal/store/factory_publications.go, hoist the identical publication 'outstanding' WHERE fragment, used in OutstandingPublications, requestPublicationWithdrawalsTx and OpenPublications (about lines 495-514, 538-557 and 613-629), into one Go const and build the three queries from it. Do this after prod-hardening lands, or on top of its jsonb_typeof version, to avoid a conflict. Also remove the observedRevision parameter and check from CandidateTuple.Advance (internal/factory/publication.go:566-578) and update its single caller in internal/factory/control/review_cycle.go:775. That check can never fail.

### FEF-11: The 'Use current required context' card exists only to override a low-confidence JEV context-bundle pick. It echoes server digests back and tracks 11 fields to guard against races.

Frontend: Factory settings, work, progress, request, intent screens. other. partially-confirmed. ~300 lines, risk low. Visible change: UI, model tools. Behaviour-preserving: no. **Status:** owner decision.

Where: `frontend/spaces/sodaspaces-factory-context-resolution.ts:1-249`, `frontend/spaces/sodaspaces-factory-response.ts:244-275`, `frontend/spaces/sodaspaces-factory-work.ts:1301`, `frontend/spaces/sodaspaces-project-work.ts:139-151,420-422`, `internal/factory/control/context_resolution.go:1-109`, `internal/factory/control/context_bundle_advice.go:1-149`, and 3 more.

**Simpler.** Have the backend always foreground the current required context, with no JEV context-bundle judgment and no hold. Then delete this element, the context_selection parser, both places that compose it, the use_current_context action, ContextSelectionStatus/UseCurrentContext and the store resolution path. If some hold must stay, use a plain 'Continue' POST by run id with no digests, and make it work for failed JEV calls too (today the button is disabled in that case).

**Risk.** Remove the frontend only after the backend stops producing the 'hold' action; otherwise held runs lose their only release button.

### FEF-8: Frontend writes check echoed receipt fields, read the command back after a network error, and keep 'Retry same' pending states (the backend still needs command_id and expected_revision)

Frontend: Factory settings, work, progress, request, intent screens. unneeded-validation-or-digest. partially-confirmed. ~300 lines, risk low. Visible change: UI. Behaviour-preserving: yes. **Status:** owner decision; coordinate with prod-hardening.

Where: `frontend/spaces/sodaspaces-factory-response.ts:42-47`, `frontend/spaces/sodaspaces-factory-work-response.ts:604-787,811-822`, `frontend/spaces/sodaspaces-factory-work.ts:1194-1250,1362-1425`, `frontend/spaces/sodaspaces-factory-work-view.ts:429-526`, `frontend/spaces/sodaspaces-factory-context-resolution.ts:132-147,212-224`, `frontend/spaces/sodaspaces-factory-completion-retry.ts:104-177`, and 7 more.

**Simpler.** Keep sending a random command_id and the expected_revision just read, since the backend requires both. On 2xx, show the substantive receipt fields (per-run pause/stop outcomes, reopened, takeover destination) and refresh. On an error, show the server's message and leave the button enabled for a fresh attempt. Delete the identity-echo checks (command_id, repository, event and cycle equality, revision === expected+1), the factory/commands/{id} read-back recovery paths, the pending 'Retry same ... change' states, and the validate-only factoryIssueDetail call in completion-retry.

**Risk.** Two tabs saving the same setting become last-write-wins, which is fine for one owner. If the backend still requires expected_revision, send the value just read.

**Safe first step.** In frontend/spaces/sodaspaces-factory-completion-retry.ts, delete the factoryIssueDetail(issueValue, ...) validate-only call (line 81) and its import. The journey is still read from issueValue, so success-path behaviour stays the same. Then remove the echo-equality checks from verify() (lines 107-117) except row.queued, and the receipt.command_id/revision check at repair.ts:141. On a well-formed 2xx reply both produce the same UI.

### GOM-5: Go host client re-validates requests and echo-checks responses that the Rust soda-host has already validated

Other Go packages (forgejo, acceptance, project, host, tailnet, application, config, strictjson, filelock, avatar, cmd/soda-dashboard, cmd/soda-tailnet, cmd/soda-extension). duplicated-layer. partially-confirmed. ~300 lines, risk low. Visible change: none. Behaviour-preserving: yes. **Status:** open (behaviour-preserving).

Where: `internal/host/factory_client.go`, `internal/host/client.go:140-198`, `internal/host/tailnet.go:45-105`, `internal/host/tailnet_lifecycle.go`, `internal/host/application_setup.go`, `internal/host/application_snapshots.go`, and 7 more.

**Simpler.** Keep the wire structs and the shared project validators (API, MCP and store paths still use them). In internal/host, delete the in.Validate() pre-checks on requests the host validates, and delete the response echo/confirm conditions. Keep JSON decoding, HTTP status mapping (ErrRunNotFound/ErrRunStale) and derived-value checks such as Architecture == GOARCH. Delete the tailnet admitEnrollment* cross-checks. Remove the dummy-id Setup(...).Validate() tail from BuildArtifact.ValidateFor and the dummy-team bind in SetupSpec.Validate. Replace strictjson plus Validate in project.Decode with json.Unmarshal. Leave the ValidateFor checks in factory/control/store alone; they are a separate question.

**Risk.** The host already refuses bad input, so the main change is that host bugs surface as later errors instead of immediate 'did not match' refusals. A few checks guard derived values (e.g. Architecture == runtime.GOARCH in profiles.go) and should be kept.

**Safe first step.** In internal/host/tailnet.go, delete admitEnrollmentSave, admitEnrollmentSaveRotate and admitEnrollmentAction and their two calls in TailnetEnrollment. In internal/host/client.go, reduce the post-call checks in Create, Inspect, ProjectAccess and Connection to plain decoding. In internal/host/factory_client.go, delete factoryStateConfirmed and the out.ID/out.Project/phase echo conditions in FactoryLaunch, FactoryInspect, FactoryStop and FactoryOutput, keeping factoryNotFound and the 404/409 mapping. With a correct host, behaviour is identical.

### HPL-10: host.json knobs no producer writes: Codex/Muse harness and catalog pins are hand-set only, image knobs matter only without a release, and the repo host.example.json is dead

Rust host platform: daemon, terminals, project, application, tailnet (lib/host non-factory, soda-project-maintain, forgejo/pg/console helpers, soda-tailnet, unix-http, wire-time). unused-configuration. partially-confirmed. ~300 lines, risk medium. Visible change: UI. Behaviour-preserving: yes. **Status:** owner decision.

Where: `lib/host/src/daemon/config.rs:22-128`, `lib/host/src/daemon/config.rs:137-180`, `lib/host/src/daemon/config.rs:285-345`, `lib/host/src/terminal/broker.rs:50-108`, `lib/host/src/dbackend/backend.rs:532-546`, `lib/host/src/main.rs:325-352`, and 4 more.

**Simpler.** Delete the unreferenced system/host/config/host.example.json now. Derive the Muse runtime and harness pins from the release payload, the way canlang_runtime already is, and make the socket paths constants. Drop `image` and `tailnet_image` as JSON knobs if the no-release dev mode is not needed. Drop the Codex harness and catalog knobs and the human Codex terminal launch only if the owner decides Codex is no longer a wanted provider.

**Risk.** The Codex provider runs through soda-identity, the factory (tcodex.rs, msp/codex.rs) and the UI, so dropping it needs a product decision. Deriving Muse pins from the release needs the payload to carry them.

**Safe first step.** Delete system/host/config/host.example.json (5 lines). Nothing in the repo refers to it: the shipped default is generated by lib/soda-release-image/src/complete.rs write_factory_defaults, and soda-install's deny_unknown_fields MachineDefaults would reject this file's image field anyway.

### IDN-10: Two provider-account UIs plus a third connection loader; soda-identity duplicates soda-providers and re-validates the server's own data

Identity service and provider credentials. duplicated-layer. partially-confirmed. ~300 lines, risk low. Visible change: UI. Behaviour-preserving: yes. **Status:** owner decision.

Where: `frontend/spaces/soda-identity.ts:1-291`, `frontend/spaces/soda-providers.ts:133-177`, `frontend/spaces/soda-identity-response.ts:36-114`, `frontend/spaces/sodaspaces-project.ts:406-420`, `frontend/spaces/sodaspaces-project-work.ts:10`, `frontend/spaces/soda-feed-entry.ts:53`, and 5 more.

**Simpler.** Keep one provider surface, the Providers dialog, with connect, reconnect, cancel and disconnect, plus the active-use list with End if wanted. Delete soda-identity; its grant parts go with IDN-2, and whether the Codex launcher moves to the terminal page is a separate decision. Factory settings should use the dialog's change event or a shared loader instead of its own inventory fetch. Drop confirm_credential_exposure and the client-supplied label from the enrollment request; the broker can default the label to the provider name. Reduce soda-identity-response.ts to plain typing without the origin, cap and owner re-checks.

**Risk.** User-visible consolidation; keep disconnect and lease-end reachable from the remaining surface.

**Safe first step.** In frontend/spaces/soda-identity-response.ts, delete enrollmentOrigin and the URL-origin check in enrollmentView, the 128-item cap in items(), and the 2048-length and id-regex checks. Fold connectionView's owner_id === actor check into availableConnectionView, since /api/identity/connections already returns only the caller's own connections. Also remove the side-effect import './soda-providers.js' from frontend/spaces/sodaspaces-project-work.ts:10; it is never rendered there, and the element is registered by soda-feed-entry.ts and sodaspaces-factory-settings.ts. Both changes are identical for any valid server response.

### PRP-6: Root factory-roles helper keeps CPython json.dumps bytes and Python == semantics that only a byte-exact test oracle relies on; preparation spends 5-7 helper execs

Project preparation pipeline below the API (standing and candidate preparations, decisions, hold, host and root helper). other. partially-confirmed. ~300 lines, risk medium. Visible change: none. Behaviour-preserving: yes. **Status:** open (behaviour-preserving).

Where: `cmd/soda-project-terminal/src/factory_roles/emit.rs:1-101`, `cmd/soda-project-terminal/src/factory_roles/records.rs:102-130`, `cmd/soda-project-terminal/src/factory_roles/validate.rs:107-130`, `cmd/soda-project-terminal/src/factory_roles/b64.rs:1-18`, `cmd/soda-project-terminal/src/factory_roles/mod.rs:1-12`, `cmd/soda-project-terminal/src/factory_roles/inputs.rs:460-480`, and 4 more.

**Simpler.** Drop the Python-parity layer. In records.rs, compare the tools.json evidence with plain serde_json::Value equality or a typed struct ==, and delete emit::json_equal, unique_keys, norm_int and numbers_equal. Remove the CPython wording from the comments in emit.rs, validate.rs, b64.rs and mod.rs. Then, optionally, move helper requests and state files to serde structs with compact serde_json output, and cut the oracle test's byte-exact stdout assertions down to parsed-field checks or delete them. Optionally merge ensure, approve, record and start into fewer ops. Keep the custody code (O_NOFOLLOW, renameat2 publication, process-group signalling).

**Risk.** pyemit and state_json are also used by subscription_*, term_*, key_request and broker in the same crate, so either keep them for those modules or convert them together. Existing on-disk state files from dev machines may need a reset.

**Safe first step.** In cmd/soda-project-terminal/src/factory_roles/records.rs:111-113, replace emit::json_equal(&saved, &fresh) with equality of the two documents parsed as serde_json::Value. Both are tools.json documents the helper itself writes with the same key set. Then delete json_equal, unique_keys, is_int_literal, norm_int and numbers_equal from factory_roles/emit.rs (about 80 lines), and remove the CPython and "byte-identical to the original" wording in emit.rs, mod.rs:11, validate.rs:107-110 and b64.rs (rename python_validate). Leave pyemit's output format unchanged so factory_roles_oracle.rs still passes. The only behaviour difference is Python-numeric edge cases such as 1 vs 1.0 or true vs 1 inside host-supplied tool evidence, which the host never sends.

### RTY-5: Automatic launch relaunch is a separate in-place mutation path that silently parks on admission refusals (including harness-version bumps it claims to tolerate) and keeps a legacy-pin compatibility branch

Retry kinds across store, control and UI. too-many-states-or-budgets. confirmed. ~300 lines, risk medium. Visible change: UI. Behaviour-preserving: yes. **Status:** owner decision.

Where: `internal/store/factory_retry_packet.go:16-273`, `internal/factory/control/dispatch_launch.go:325-425,505-542,593-641`, `internal/factory/control/dispatch_recovery.go:286-314`, `internal/factory/control/dispatch_attempt.go:220-243`, `internal/factory/assignment.go:31,585-590`, `internal/factory/run.go:118-128,158`, and 1 more.

**Simpler.** Settle an unlaunched attempt as finished (failed, 'launch did not start') and let the ordinary fresh-dispatch path start a new root on the next pass, with a small attempt counter on the issue if a cap is wanted. That deletes RecordRetryPacket and the three relaunch functions. At minimum: (a) append the admissionWait / plan-time wait to report.Waits in recordRetryRun and routeAttemptWait instead of returning silently, or finish terminally on ErrAdmissionChanged as routeAttemptWait already does for WaitAuthority; (b) delete the legacy no-pin branch everywhere it appears (checkRetryPinAdvance, line 67, AdoptedToolIdentity's zero-pin return, Assignment.Validate's optional pin, the dispatch_context_recovery legacy comment path) and make AdoptedHarness required.

**Risk.** In-place relaunch keeps the same assignment ID, root and recorded immutable child prompts. Children (review, correct, implement) need lineage-preserving relaunch, so a fresh-root replacement must re-derive their prompts. This overlaps DSP-6 and should be coordinated with it.

**Safe first step.** Delete the legacy pin-less path, which no current row can reach. In internal/store/factory_retry_packet.go, remove the empty-pin early return in checkRetryPinAdvance (lines 89-91) and the `a.AdoptedHarness != (project.FactoryHarnessPin{}) &&` guard on line 67. In internal/factory/assignment.go:585-590, require AdoptedHarness to be valid instead of optional. In internal/factory/run.go:121-125, remove AdoptedToolIdentity's zero-pin return and in the same file the legacy branch at ~158. Remove the matching legacy fallback in internal/factory/control/dispatch_context_recovery.go:120-135. Every persisted assignment is built with AdoptedHarness set (dispatch_launch.go:56/223, dispatch_timeonly.go:410, publication_children.go:999), so observable behaviour stays the same. Leave the silent-park fix and the removal of the relaunch path for DSP-6 coordination, because both change visible behaviour.

### WEB-9: Settings writes use four different idempotency and concurrency schemes, with inconsistent revision rules and checks that can never fail

Web API and view models (non-MCP, non-Feed, non-extension): internal/web/*.go, internal/web/auth/**, internal/web/api/*.go excluding factory_mcp*, factory_feed*, extension*. duplicated-layer. confirmed. ~300 lines, risk medium. Visible change: UI. Behaviour-preserving: yes. **Status:** owner decision.

Where: `internal/web/api/factory_settings.go:21-37,85-93`, `internal/web/api/factory_policy.go:30-91,105-124,134-158`, `internal/web/api/factory_sponsorship.go:35-71,153-192`, `internal/web/api/factory_connection_usage_budget.go:54-74`, `internal/web/api/factory_repair.go:25-127`, `internal/web/api/factory_verification_allowance.go:30-81`, and 8 more.

**Simpler.** For settings, the PUT carries the full value. The server saves it (last write wins) and returns the saved record. Drop command_id, expected_revision, digest and receipt from the settings writes, and drop the client ID and predecessor compare-and-swap from the planning allowance. Delete the inline replay in factory_repair and in the verification allowance, and simplify the matching frontend checks. Keep the command ledger only for run and repository actions that can return 202 running (stop, retry, takeover, pause, resume). commandRepository then shrinks to run and project targets.

**Risk.** Settings forms and their responses change. Existing settings command rows become harmless leftovers. Two people editing the same setting at once would get last-write-wins, which is fine on a single-founder box.

**Safe first step.** In internal/factory/control/grants.go, delete the `Revision != expected` comparisons and the `expected int64` parameter from ApplyPolicy, ApplyCapacity, ApplyOperatorGrant, ApplySponsorship, ApplyConnectionUsageBudget and ApplyEnvironmentGrant. Keep the `Repository <= 0` guards. Update the six call sites in internal/web/api/factory_policy.go, factory_sponsorship.go and factory_connection_usage_budget.go to stop passing in.ExpectedRevision. Every caller passes the same value it put into record.Revision, so the store compare-and-swap still makes every decision and behaviour is identical. Adjust or delete internal/factory/control/integration_helpers_test.go usages as needed.

### FEP-6: App view keeps manual 'paste five sha256 digests' setup/replace forms beside the prepared-build path, plus persisted company/delivery filters

Frontend: Workspace, project, applications, terminals, layout, Forgejo and Tailnet pages. duplicated-layer. confirmed. ~280 lines, risk medium. Visible change: UI. Behaviour-preserving: no. **Status:** owner decision.

Where: `frontend/spaces/sodaspaces-application-setup.ts:322-385`, `frontend/spaces/sodaspaces-applications.ts:634-661,756-819`, `frontend/spaces/sodaspaces-application-delivery.ts:332-498`, `frontend/spaces/sodaspaces-applications.ts:107-125,986-995,1018-1042`, `frontend/spaces/sodaspaces-workspace.ts:460-485,725-767,955-956`, `frontend/spaces/sodaspaces-applications.ts:270-300,587-598`, and 1 more.

**Simpler.** Keep one setup path and one replace path, both keyed by the prepared build id. Delete the manual JSON setup form (setup.ts create() and its form; keep beginPreparedSetup) and the replace form (replaceView/replaceApplication). Make the Feed's open-app action link to the repository App place with ?delivery=... instead of persisting delivery and company selections in localStorage. Drop the company-scope filter and its two events. Remove the 'Verify build' button and fold provenance checking into prepare/apply on the server. Optionally, also drop the backend's raw replacement Spec and the verify_build action afterwards.

**Risk.** Operators lose the escape hatch for apps not built by Factory. Backups/restore (applications.ts:308-512) are documented Canlang scope (P25/P26) and are left in place here, though AGENTS.md discourages such recovery systems. ui/repo-places rewrites sodaspaces-applications.ts (+276 lines); coordinate.

### STO-9: Store reads re-validate rows that the store validated on write and cross-check JSON copies against their own key columns

Store (PostgreSQL layer) structure. unneeded-validation-or-digest. partially-confirmed. ~280 lines, risk low. Visible change: none. Behaviour-preserving: yes. **Status:** coordinate with prod-hardening.

Where: `internal/store/factory_attempt_allowances.go:177-185`, `internal/store/factory_grants.go:128-140`, `internal/store/factory_grant_commands.go:71-80`, `internal/store/factory_grants.go:167-186`, `internal/store/factory_grants.go:437-460`, `internal/store/factory_completion_material.go:241-249`, and 6 more.

**Simpler.** On read, unmarshal without re-checking. Drop validateAttemptAllowanceRow, the duplicated budget key-equality checks, the allowed-map scope re-check in SubscribedFeed, the dead SpaceProjects, and the octet_length CASE projections in SpaceProjectsAfter. In the dispatch packet, drop the ID/repository equality re-checks of structs the coordinator built, but keep the locked control/head/gate reads and the Authority comparison that returns ErrAdmissionChanged. Keep read-side Validate only on tables that SQL jsonb_set updates, or drop it there too once those updates are checked.

**Risk.** There are zero users and no other writers. The one behaviour lost is the read-time refusal of rows that current code cannot produce.

**Safe first step.** Delete the uncalled SpaceProjects function (internal/store/store.go:264-285) and the comment references to it in factory_inventory.go and members.go. Remove the duplicated `budget.Connection != connection` / "connection usage budget key mismatch" checks in internal/store/factory_grants.go:134-136 and internal/store/factory_grant_commands.go:73-75; loadGrant fetches by that same connection key, so the check can never fail. Remove the allowed-map scope re-check in SubscribedFeed (internal/store/factory_feed.go:157-160 and 170-172), since its SQL already filters on query.Repositories.

### APP-7: Hand-typed reference paths (register endpoint, install action, paste-JSON replace and setup forms) duplicate the Factory build path

Application runtime: build, home, snapshots, backups, restore, factory delivery. dead-code. partially-confirmed. ~260 lines, risk medium. Visible change: UI. Behaviour-preserving: yes. **Status:** owner decision.

Where: `internal/web/api/applications.go:237,280-283,315-338`, `internal/store/applications.go:13-43`, `internal/web/api/application_actions.go:51-54,148`, `internal/application/administration.go:48,74-76`, `lib/host/src/application/runtime.rs:299-315,393`, `frontend/spaces/sodaspaces-applications.ts:270-272,577,634-661,756-818`, and 1 more.

**Simpler.** After APP-1's route fix, delete POST /api/applications (apiApplicationRegister, applicationRegistration, CreateApplication and its tests), the install action (admission in application_actions.go/administration.go, the Install button, Runtime::install and its check_release branch), the paste-JSON replace form (keep replacement_build), and the paste-JSON manual setup form (keep prepared-build setup). Keep admit_image's label checks and the Spec digests: the Factory build, setup and replacement_build paths all depend on them.

**Risk.** Until the extension route admits application-builds, these manual paths are the only working ones. Do APP-1's route fix first.

**Safe first step.** Delete the unused register endpoint. Remove the POST case at internal/web/api/applications.go:237, apiApplicationRegister and the applicationRegistration type (applications.go:280-283, 315-338), and Store.CreateApplication (internal/store/applications.go:13-43, keeping insertApplicationRecord, which setup uses). Then delete or adjust the CreateApplication-based tests in internal/store/applications_test.go and internal/web/api/applications_test.go. No frontend code calls this endpoint, so the UI is unchanged.

### FED-10: Trim the Feed filters and subscription machinery; remove a setting that does nothing

The Feed. unused-configuration. confirmed. ~250 lines, risk low. Visible change: UI. Behaviour-preserving: yes. **Status:** owner decision; coordinate with prod-hardening.

Where: `internal/store/factory_feed.go:15-35,108-176`, `internal/factory/feed_contract.go:395-509`, `internal/factory/feed_subscription.go:1-64`, `internal/store/factory_feed_subscriptions.go:1-65`, `internal/web/api/factory_feed_subscription.go:1-79`, `internal/web/api/factory_feed_serve.go:1-73`, and 5 more.

**Simpler.** - Remove kinds end to end (Show select, PUT/GET field, FeedSubscription.Kinds, FeedQuery.Kinds, the SQL wantsRecords/wantsPosts parameters, the handbook sentence). It never took effect. - Keep All, Needs me, search and role. Drop the Decisions, Knowledge and Evidence views and the $9 current-only predicates. - Save the repository list with a plain upsert instead of the revision CAS. - Validate the query once, at parseFeedQueryParams. - Drop the store-side scope-mismatch check and the second entry validation. - Memoize visibleRepository per distinct repository within a request, and skip it in project scope. - Use a plain integer cursor.

**Risk.** Keep a per-repository visibility check on the global page, because a subscription can name repositories the user has since lost access to. ui/repo-places also edits the Feed element.

**Safe first step.** Backend only, same observable behaviour: - internal/web/api/factory_feed_serve.go: delete both query.Validate calls (lines 39 and 55; route.go already validated, and narrowing to a subset cannot invalidate the query) and the per-entry entry.Validate loop check. - internal/store/factory_feed.go: delete the leading query.Validate and the 'allowed' map with its 'feed entry scope mismatch' check (the SQL already filters on repository = ANY($2)). - internal/web/api/factory_feed_route.go: memoize the visible() closure's result per entry.Repository in a map, so each distinct repository is checked once per request. - internal/factory/feed_subscription.go: in ApplyFeedSubscription, drop the comment wording about 'preserving kinds' (kinds removal itself changes the UI, so leave it for a later step).

### FED-9: Write merge, completion and judgment cards as feed events instead of deriving them in the feed query; stop using the F10 card as the app-verified record

The Feed. duplicated-layer. partially-confirmed. ~250 lines, risk medium. Visible change: UI, DB schema. Behaviour-preserving: no. **Status:** owner decision; coordinate with prod-hardening.

Where: `internal/store/factory_feed.go:50-92`, `internal/store/factory_feed_record.go:656-855`, `internal/web/api/factory_feed_route.go:105-162`, `internal/store/schema.go:37,102,133-150,356,398-399`, `internal/store/factory_completions.go:26`, `cmd/soda-identity/src/schema.rs:32,97,128,137-147,323,365-366`, and 9 more.

**Simpler.** At the merge, completion and Jev-judgment write sites, call recordFactoryFeedEventTx in the same transaction with F06/F03/F04 communication content. Then delete: - the four UNION legs - FeedMilestoneDetails and its helpers - the milestone branch in enrichFeedRecords - the FeedActivity* constants - the activity 'judgment' filter terms (F03/F04 are already matched by component_id) - the feed_seq columns, triggers and shape checks, in both the Go schema and the soda-identity Rust schema Move the frontend rendering for merged, completed and judgment cards from entry.activity and payload.merge/completion/judgment to payload.communication with component F06/F03/F04, and drop 'merged'/'completed'/'judgment' from feedActivities. Optionally record app verification on the verify assignment's result and point the five F10 gate queries at it, so the F10 post becomes display only.

**Risk.** Historic merge and judgment cards disappear unless backfilled, which is acceptable with zero users. The app-delivery queries must be re-run end to end after the change.

### GOM-3: Duplicated actor checks and submit/lookup/cancel code across the native Forgejo workers, a no-op mapRevisionError, and a second set of publish error types

Other Go packages (forgejo, acceptance, project, host, tailnet, application, config, strictjson, filelock, avatar, cmd/soda-dashboard, cmd/soda-tailnet, cmd/soda-extension). duplicated-layer. partially-confirmed. ~250 lines, risk low. Visible change: none. Behaviour-preserving: yes. **Status:** open (behaviour-preserving).

Where: `internal/forgejo/review.go:22-48,81-89,170-230`, `internal/forgejo/merge.go:22-45,81-89,110-170`, `internal/forgejo/checks.go:25-52,85-99`, `internal/forgejo/issue_graph.go:18-45,105-150`, `internal/forgejo/delivery_resolve.go:14-122`, `internal/forgejo/lifecycle.go:105-129,157-189`, and 4 more.

**Simpler.** Extract one nativeActor{background, rest, tokenFile} with a single checkActor(ctx, actor) and embed it in Reviewer, Merger and CheckAssessor. Merge reviewReadError/mergeReadError into readWait(err, fallback) and keep checkReadError's 503 branch as an option. Add one submitOrAdopt helper for the SubmitOperation, then GetOperation on error, then adopt sequence, and one lookup/cancel helper parameterised by the outcome mapper. Delete mapRevisionError. Have publish return factory.PublicationRefusal/Wait directly and delete Refusal/Wait/Adopted and flattenError, keeping the candidate_invalid/operations_unavailable rewrite. Decide separately whether the per-operation REST /user check and the HTTP/1 loadActor transport are still needed; removing them changes which wait reason appears.

**Risk.** Wait/refusal reason strings must stay the same if anything matches on them. Caching the actor means a rotated token takes effect only at restart, which Publisher already accepts.

**Safe first step.** Delete mapRevisionError in internal/forgejo/publish/operation_observation.go and return err directly at its call site. Replace the identical checkActor bodies in review.go, merge.go and checks.go with one shared helper over {background, rest, tokenFile}, with the same reasons: operations_unavailable, credential_invalid, authority_lost. Merge reviewReadError and mergeReadError into one function that takes the fallback reason. Leave loadActor and the REST /user check unchanged in this step.

### HPL-4: Host keeps two copies of Lease/AcquireRequest (terminal and factory) with field-by-field converters, plus a one-implementation IdentityBroker trait

Rust host platform: daemon, terminals, project, application, tailnet (lib/host non-factory, soda-project-maintain, forgejo/pg/console helpers, soda-tailnet, unix-http, wire-time). duplicated-layer. partially-confirmed. ~250 lines, risk low. Visible change: none. Behaviour-preserving: yes. **Status:** open (behaviour-preserving).

Where: `lib/host/src/terminal/lease.rs:24-449`, `lib/host/src/factory/identity.rs:7-247`, `lib/host/src/daemon/broker.rs:6-59`, `lib/host/src/dbackend/factory_terminal.rs:3-310`, `lib/host/src/terminal/broker.rs:27-37`, `lib/host/src/dbackend/mod.rs:289`.

**Simpler.** Inside lib/host, have the factory code use terminal::Lease (or one host-level Lease) directly and drop factory::Lease, its visitor and encoder, and factory::AcquireRequest. Give the factory path a constructor that sets the deadline from RFC3339. Delete cv_lease_to_texec, cv_acquire_to_texec and cv_lease_to_factory. Make identity_launch take the concrete DaemonBackend, or a closure, and delete the IdentityBroker trait. Leave soda-identity's server-side serde structs alone, since sharing them needs a new crate and a reconciled decoding model.

**Risk.** factory/ and cmd/soda-identity are other finders' scope, so coordinate. The change is mechanical.

**Safe first step.** Delete the IdentityBroker trait in lib/host/src/terminal/broker.rs:27-37 and its re-export in terminal/mod.rs:96. Change identity_launch to take &DaemonBackend, and turn the methods in the `impl terminal::IdentityBroker for DaemonBackend` block (dbackend/mod.rs:289) into inherent methods. Then replace factory::Lease with terminal::Lease in the factory module. To keep behaviour identical, keep factory's empty-deadline encoding: today factory writes "" and terminal writes the Go zero time. After that, delete the cv_* converters in lib/host/src/daemon/broker.rs and their uses in dbackend/factory_terminal.rs.

### IDN-6: One global controller mutex is held across 4-minute host calls, behind an HTTP admission/drain layer that cannot add concurrency

Identity service and provider credentials. other. confirmed. ~250 lines, risk medium. Visible change: none. Behaviour-preserving: yes. **Status:** open (behaviour-preserving).

Where: `cmd/soda-identity/src/control.rs:117-148`, `cmd/soda-identity/src/control.rs:202-213`, `cmd/soda-identity/src/runtime.rs:8-10`, `cmd/soda-identity/src/retirement.rs:17-68`, `cmd/soda-identity/src/retirement.rs:112-181`, `cmd/soda-identity/src/registration.rs:19`, and 7 more.

**Simpler.** Keep the two activated sockets and serve each with a plain hyper http1 loop (or a thread per connection) and a shutdown flag. Drop the semaphores, drain map, active flags, inflight, percent-decoding, Origin/chunked checks, the manual bind path, the extra flag spellings and the unused Controller::close. Separately, stop holding the controller lock across host calls: decide under the lock, call the host outside it, then re-lock to record the outcome.

**Risk.** Removing the HTTP machinery is low risk (trusted local callers on 0600/0660 sockets). Narrowing the lock needs care around Codex single-stream reservation and retirement ordering.

**Safe first step.** 1. Delete the inflight counter: the field, constructor parameter and InflightGuard in cmd/soda-identity/src/http.rs, the AtomicUsize wiring in cmd/soda-identity/src/main.rs:355-366, and the matching arguments in cmd/soda-identity/tests/http.rs:30-41. 2. Delete the uncalled Controller::close in cmd/soda-identity/src/control.rs:202-213. 3. In main.rs parse_args, keep only the documented `--config PATH` (drop -config, --config= and -config=). 4. In main.rs, delete the manual listen() fallback in service_listeners, since the shipped unit always socket-activates.

### IDN-8: Hand-pinned provider binary/version/sha256 are required by setup and accepted by the broker, then ignored (dead explicit mode)

Identity service and provider credentials. unused-configuration. confirmed. ~250 lines, risk low. Visible change: none. Behaviour-preserving: yes. **Status:** open (behaviour-preserving).

Where: `cmd/soda-identity/src/main.rs:15-51`, `cmd/soda-identity/src/main.rs:269-301`, `cmd/soda-identity/src/providers/codex/config.rs:6-19`, `cmd/soda-identity/src/providers/codex/config.rs:45-71`, `cmd/soda-identity/src/providers/codex/mod.rs:31-36`, `cmd/soda-identity/src/providers/codex/mod.rs:73-83`, and 15 more.

**Simpler.** Provider settings become {root}, and a provider is enabled when root is set. In soda-identity, delete the store flag, the explicit-mode branches in validate_config/check_binary and session_binary, the fallback_version/fallback_digest fields, Session::provider_version/provider_digest, soda-identity's own resolution::resolve_existing, digest_of_executable, via_store, and the Option in bind_new. Keep the lib/provider-tools resolve_existing. In soda-setup, drop check_provider_config/validate_provider_binary/digest_provider_binary and the binary/version/sha256 manifest fields, keeping validate_provider_root. Fix configuration.md, credentials.md and the public factory-authorization example.

**Risk.** Existing identity.json files carry the old keys and deny_unknown_fields would refuse them: accept-and-ignore once, or let soda-setup --maintenance rewrite the file.

**Safe first step.** This step changes nothing observable and needs no config change. In cmd/soda-identity/src/providers/codex/mod.rs and muse/mod.rs, delete Session::provider_version/provider_digest, the fallback_version/fallback_digest fields and their initialisers, the `if !self.config.store` early returns in bind_new and session_binary, and the explicit branches in validate_config/check_binary (codex/config.rs:50-70, muse/mod.rs:334-363). In cmd/soda-identity/src/providers/resolution.rs, delete resolve_existing, digest_of_executable and the debug_assert that uses it, plus the resolve_existing assertion in tests/provider_store.rs:240-250. Leave the config keys, the store flag and soda-setup validation alone for now.

### MCP-8: factory.context pages its own JSON in HMAC-signed fragments, duplicates the role inline and in pages, and pads every read

Model-facing MCP tools. other. partially-confirmed. ~250 lines, risk low. Visible change: model tools. Behaviour-preserving: no. **Status:** owner decision; coordinate with prod-hardening.

Where: `internal/web/api/factory_mcp_context_page.go:1-181`, `internal/web/api/factory_mcp_context.go:47-91`, `internal/web/api/factory_mcp.go:883-952`, `internal/web/api/factory_mcp.go:764-774`, `internal/web/api/factory_mcp_bound.go:158`, `internal/web/api/factory_mcp_repair_scope.go:21-28,73-90`, and 3 more.

**Simpler.** Return current.role inline up to the 64 KiB result limit, always and exactly once, and drop result_schema; the tools list already carries the finish schema. Delete factory_mcp_context_page.go, current_role_page, ContextCursorKey, the cursor schema and its handling, and the 8192-byte stub swap. For reassessment_scope, return the referenced diagnosis or draft payload whole (it is already bounded by MCPMaxSubmissionPayload) instead of record/offset fragments, and delete ReadRepairObservation's fragmenting. Drop the plumbing fields (identity_source, route, limits, generation, capability, source_digest, scope_digest, prompt_sha) and the repeated guidance strings, keeping guidance in the phase prompt.

**Risk.** Check the largest real role contexts (review with carried findings, plan) against the 64 KiB result bound.

### RUN-4: AssignmentResult carries never-set legacy fields (Review, ReviewPassed, Findings) plus dead ResultSchema/CompletedCandidate, mirrors PhaseResult status/summary, and re-validates PhaseResult many times per finish

Run end: task events, checkpoints, results and settlement (RUN). dead-code. partially-confirmed. ~250 lines, risk low. Visible change: UI, model tools. Behaviour-preserving: yes. **Status:** owner decision; coordinate with prod-hardening.

Where: `internal/factory/result.go:89-95`, `internal/factory/result.go:355-420`, `internal/factory/result.go:612-656`, `internal/factory/library_result.go:9`, `internal/factory/assignment_result.go:12-339`, `internal/factory/review_native.go:218-289`, and 10 more.

**Simpler.** Delete ResultSchema and Result.CompletedCandidate outright. Drop AssignmentResult.Review and ReviewPassed, and the always-empty Findings, along with every check on them: the r.Review guards, the line-177 legacy reviewer tail, the feed Review/Findings branches and the review_cycle nil checks. Read Status and Summary from PhaseResult, or keep thin accessors. Then remove the Summary==PhaseResult.Summary mirror checks. Keep the role/phase gating and candidate binding that ValidateForAssignment performs; that part is not duplicated. Collapse the ten nilTypedOutputExcept* methods into one count-the-set-slots helper. Separately, fix or remove the legacy review-json prompt branch in dispatch_prompt.go, which nothing parses; that is a worker-prompt change. Validate PhaseResult once at decode and once at FinishAssignment.

**Risk.** Stored rows are JSONB, so the dropped fields are simply ignored on read. Readers of Status and Summary must switch to PhaseResult, or keep thin accessors.

**Safe first step.** In internal/factory/result.go, delete ResultSchema (line 622) and Result.CompletedCandidate (lines 624-656); both have zero references. Collapse the nine nilTypedOutputExcept* methods in result.go (lines 89-95, 355-420) and nilTypedOutputExceptMaintenance in library_result.go into one helper that checks exactly one PhaseOutput slot is set; behaviour stays the same. In internal/store/factory_feed_record.go:329-340, delete the result.Review and result.Findings branches, which can never fire because Review is never assigned and Findings is always empty.

### WEB-11: Acceptance and context-hold flows make the browser copy back server-computed digests; sponsorship echoes the credential generation

Web API and view models (non-MCP, non-Feed, non-extension): internal/web/*.go, internal/web/auth/**, internal/web/api/*.go excluding factory_mcp*, factory_feed*, extension*. unneeded-validation-or-digest. partially-confirmed. ~250 lines, risk medium. Visible change: UI. Behaviour-preserving: no. **Status:** owner decision.

Where: `internal/web/api/factory_issue_view.go:45-245`, `internal/web/api/issue_acceptances.go:29-140,200-241`, `internal/web/api/factory_views.go:113-118,175-182`, `internal/web/api/factory_lifecycle.go:108-162`, `internal/web/api/factory_sponsorship.go:24-33,104-107`, `internal/factory/context_resolution.go:26-30`, and 5 more.

**Simpler.** Acceptance POST sends {command_id, comment_ids split into sources/resolutions, prerequisites as endpoint repo/issue/outcome}. The server snapshots native state, computes the digests, revisions, predecessor and prerequisite acceptance ids itself, and records what it saw. Drop the preview digest echo, the client decision_id and predecessor, and use the common error envelope. use_current_context takes only the run, or goes away with the per-run context-bundle hold. Sponsorship keeps the user's grant choice (or drops it only if one grant per connection is enforced) but reads the connection generation server-side.

**Risk.** This gives up 'accept exactly the revision I looked at' protection against several editors racing, which a single-founder box does not need. The acceptance and context-resolution cores live in control.

### WEB-8: auth package re-implements the protected request pipeline for four /api/me routes, two of which have no caller

Web API and view models (non-MCP, non-Feed, non-extension): internal/web/*.go, internal/web/auth/**, internal/web/api/*.go excluding factory_mcp*, factory_feed*, extension*. duplicated-layer. confirmed. ~250 lines, risk low. Visible change: none. Behaviour-preserving: yes. **Status:** open (behaviour-preserving).

Where: `internal/web/auth/extension_service.go:16-281`, `internal/web/auth/session.go:12-38`, `internal/web/auth/forgejo_keys.go:7-23`, `internal/web/auth/extension_test.go`, `internal/store/store.go:158-164`, `internal/web/api/extension_terminal.go:31`, and 3 more.

**Simpler.** Register GET/POST /api/me/development-keys, DELETE /api/me/development-keys/{key} and GET /api/me/forgejo-keys on the API mux with extensionProtected. Move apiKeys, apiRemoveDevelopmentKey and apiExtensionForgejoKeys into api; they only need store.Session and the request authority. Delete /api/session, /api/me/preferences, Store.RenameProfile, the proxy's extensionSessionRoute and its /me/preferences case, and the whole auth extension pipeline. Make the '/' fallback a plain JSON 404. If the Content-Type check is wanted, add it to extensionProductMutation once rather than keeping a second pipeline.

**Risk.** auth also requires Content-Type: application/json on mutations; keep that check in protect() if wanted.

**Safe first step.** Delete the two routes nobody calls: - In internal/web/auth/extension_service.go: the /api/session branch in serveExtension, serveExtensionSession, writeExtensionSession, sessionUserView, the /api/me/preferences case in serveExtensionAccount, serveExtensionPreferences, and the /api/session and /api/me/preferences cases in allowExtensionMethod. - In internal/web/auth/session.go: preferences(). - In internal/store/store.go: Store.RenameProfile. - In internal/web/api/extension.go: extensionSessionRoute and the "/me/preferences" case in extensionMeRoute. - In docs/reference/api.md: lines 29-30. - Tests: the matching cases in internal/web/auth/extension_test.go and internal/web/api/extension_test.go, and drop "/api/session" from internal/web/retired_frontend_test.go. No frontend code calls these routes, so behaviour for real callers stays identical. Then move development-keys and forgejo-keys onto the mux with protect() as a second step.

### DSP-4: Three concurrency settings and MaxQueued are dead; admission rechecks partly duplicate the planner, but still guard the two HTTP-driven intent-plan dispatches

Factory dispatch and run start. duplicated-layer. partially-confirmed. ~230 lines, risk medium. Visible change: UI, model tools. Behaviour-preserving: yes. **Status:** owner decision; coordinate with prod-hardening.

Where: `internal/factory/control/dispatch_selection.go:62-140`, `internal/factory/control/dispatch_occupancy.go:1-55`, `internal/factory/control/dispatch_attempt.go:217-256,471-473`, `internal/factory/control/dispatch_launch.go:291-318`, `internal/factory/control/publication_children.go:1106-1107`, `internal/store/factory_dispatch_packet.go:76-95`, and 11 more.

**Simpler.** First, behaviour-preserving: set repoLimit to the constant 1 and drop the policy/operator min computation. Delete the unreachable sponsorship-slot check in checkSponsorshipBudget and the slot count in checkAdmissionLimitsTx (heldConn then only feeds `planned`, which is always 0). Drop the redundant byProject term (ProjectUnsettledRuns already covers it). Drop the duplicate Validate() calls in checkDispatchPacketInputs. Then, with a visible and contract change: remove the MaxConcurrent fields from policy, operator and sponsorship, plus MaxQueued, from the structs, API, Settings UI and the prompt's 'Concurrency limits' line (or reduce it to the appliance limit). Finally, once both HTTP DispatchIntentPlan callers run under progressPass and grant writes are serialized: move the preparation and attempt-allowance counts into the planner, collapse the transaction's grant re-lock and recount, and delete passOccupancy/refreshOccupancy and the lost-race admissionWait mappings.

**Risk.** A grant the owner changes mid-pass would take effect one pass later instead of being refused inside the transaction; pause already stops outstanding runs. The HTTP DispatchIntentPlan path must be serialized first.

**Safe first step.** In internal/factory/control/dispatch_selection.go checkConcurrencyLimits, replace the policy/operator min-and-clamp (lines 83-89) with repoLimit := 1, and drop the `+occupancy.byProject[plan.projectID]` term, since ProjectUnsettledRuns already refuses on any unsettled project run. In checkSponsorshipBudget, delete the unreachable `slots >= plan.sponsorship.MaxConcurrent` WaitSponsorship branch. In internal/store/factory_dispatch_packet.go checkAdmissionLimitsTx, delete the unreachable sponsorship-slot count and ErrSponsorshipFull return (lines 1197-1203). Keep the stored fields, prompt line and Settings inputs in this step, so nothing observable changes.

### STO-8: 25 never-firing immutable-row triggers plus redundant creation-profile/preparation/application guards and their open-time checks and Rust mirrors; execution-identity and command guards still protect current code paths

Store (PostgreSQL layer) structure. unneeded-validation-or-digest. partially-confirmed. ~230 lines, risk low. Visible change: DB schema. Behaviour-preserving: yes. **Status:** owner decision; coordinate with prod-hardening.

Where: `internal/store/schema.go:172-277`, `internal/store/factory_completions.go:63-70`, `internal/store/factory_mcp_capability.go:44-46`, `internal/store/factory_mcp_submission.go:51-53`, `internal/store/schema.go:417-451`, `internal/store/schema.go:545-556`, and 5 more.

**Simpler.** In both the Go and Rust schemas, drop soda_reject_immutable and its 25 triggers, and drop soda_guard_creation_profile, soda_guard_preparation_refs and soda_guard_application, which Go already enforces at application_restore.go:105. Remove the outcome/reconciled clauses from soda_guard_run_binding, or drop the whole function if pinning the identity fields is not wanted. Delete verifyTrigger and the verifyImmutable* checks, plus VERIFY_TRIGGERS and its Rust users, and the trigger assertions in application_schema_test.go. Keep soda_guard_execution_identity until Rust observe_execution gets a compare-and-set on state and digest. Keep soda_guard_command's finished/outcome clause until the two delivery-repair outcome UPDATEs add finished=''. Keep the feed-order triggers.

**Risk.** These triggers only turn a future bug into an exception. No current statement can fire them.

**Safe first step.** In internal/store/schema.go, delete the 18 inline soda_reject_immutable trigger statements (lines 252-259, 268-277) and the FactoryCompletionTriggers, FactoryCompletionDeleteTrigger, FactoryReassessmentDeleteTrigger, FactoryMCPCapabilityDeleteTrigger and FactoryMCPSubmissionTriggers constants with their uses. Also delete soda_reject_immutable, soda_guard_creation_profile and soda_guard_preparation_refs with their triggers. Remove verifyImmutableCreationProfile, verifyImmutablePreparationRefs and the revision entries of verifyImmutableApplications, and their calls in schema.go and observe.go. Make the same deletions in cmd/soda-identity/src/schema.rs and VERIFY_TRIGGERS. Delete the trigger-exception cases in application_schema_test.go and factory_completions_test.go. Leave soda_guard_execution_identity and soda_guard_command in place. No current statement can fire anything removed, so observable behaviour does not change.

### BUD-10: Three caps on one build-review-fix loop (correction cycles, review rounds, a per-head extra pass), tracked in ledgers that duplicate the publication's review-cycle history

Budgets, allowances and limits (cross-cutting). too-many-states-or-budgets. partially-confirmed. ~220 lines, risk low. Visible change: UI, model tools. Behaviour-preserving: no. **Status:** owner decision.

Where: `internal/factory/allowance.go:11-32`, `internal/factory/allowance.go:64-76`, `internal/factory/allowance.go:84-94`, `internal/factory/allowance.go:134-149`, `internal/factory/allowance.go:274-311`, `internal/factory/allowance.go:450-570`, and 12 more.

**Simpler.** Keep CorrectionCycles as the one loop cap: it bounds the number of heads. For each head allow one first review, the policy-required review (already deduplicated per head and actor by policyReviewRecorded) and one discretionary re-review, found by checking p.ReviewCycles for that head. Delete ReviewRounds and the Reviews, ReviewedHeads and UnchangedHeadExtra ledgers. Delete ReviewFits, ConsumeReview, ChargeReviewDispatch, ErrReviewRoundsExhausted and ErrUnchangedHeadExtraExhausted, along with the matching no-refill and time-only validation clauses. Replace the delivery 'review_allowance_exhausted' outcome with the same per-head check. Show correction exhaustion as a named attention state instead of the generic 'fenced'.

**Risk.** The review count stays bounded by corrections + 1, plus one discretionary pass per head.

### FED-6: Remove dead comment edit/delete/revoke store paths and always-false deleted/revoked tombstone checks

The Feed. dead-code. partially-confirmed. ~220 lines, risk low. Visible change: none. Behaviour-preserving: yes. **Status:** coordinate with prod-hardening.

Where: `internal/store/factory_feed_comment.go:79-85,147-288 (SaveFeedComment wrapper, EditFeedComment, DeleteFeedComment, MarkFeedCommentRevoked, markFeedComment, blockCommentEffectTx)`, `internal/store/factory_feed_comment.go:51,55,114,299-300,529-530,665-666,723 (deleted/revoked flags and predicates)`, `internal/factory/feed_communication.go:229-230 (Deleted/Revoked fields; keep CommentEffectBlocked at :54)`, `internal/store/factory_feed.go:48-49`, `internal/store/factory_feed_contribution.go:53,367,431-432,498-506,579-609`, `internal/store/factory_mcp_inbox.go:16,26`, and 7 more.

**Simpler.** Delete SaveFeedComment, EditFeedComment, DeleteFeedComment, MarkFeedCommentRevoked, markFeedComment and blockCommentEffectTx along with their test call sites (TestFeedCommentEditDeleteRevoke and others). Drop the deleted/revoked predicates, which are always false, from feed, contribution, inbox, MCP and post-resolution SQL. Drop the Record.Deleted/Revoked fields and checks, and commentBodyFor. In the browser, delete commentTombstoneOf and the bodyless 'removed' branch of parseCommentDetail. Keep CommentEffectBlocked, which an invalid Product Lead draft still sets. Also keep the browser's 403/404 access-loss removal (removalOrder, stamp/applyAuthoritativeRemoval). Simplifying that would be a separate change that users can see.

**Risk.** Old rows keep harmless 'deleted':false and 'revoked':false keys. If a reader loses repository access, the 403/404 can show as an ordinary read error.

**Safe first step.** In internal/store/factory_feed_comment.go, delete SaveFeedComment (lines 79-85) and lines 147-288 (EditFeedComment, DeleteFeedComment, MarkFeedCommentRevoked, markFeedComment, blockCommentEffectTx). Point test callers of SaveFeedComment at SaveFeedCommentWithSource(..., nil), and delete the edit/delete/revoke tests: TestFeedCommentEditDeleteRevoke in internal/store/factory_feed_comment_test.go and the call sites in internal/store/factory_mcp_communication_test.go, internal/web/api/factory_feed_comment_test.go and factory_mcp_live_test.go. Leave CommentEffectBlocked and the frontend 403/404 removal code alone.

### PRP-4: Maintenance hold stored twice (database row plus native marker, each with a revision), synced in three places and never released by lifecycle start

Project preparation pipeline below the API (standing and candidate preparations, decisions, hold, host and root helper). duplicated-layer. partially-confirmed. ~220 lines, risk medium. Visible change: UI, DB schema. Behaviour-preserving: yes. **Status:** owner decision.

Where: `internal/store/schema.go:168`, `internal/store/preparation.go:55-91`, `internal/store/preparation.go:228-235`, `internal/store/factory_grant_commands.go:88-121`, `internal/project/preparation.go:391-451`, `internal/web/api/environments_preparation.go:104-198,242-250`, and 7 more.

**Simpler.** Preferred: drop the maintenance hold entirely and let the lifecycle-stopped state bar preparation. That removes project_maintenance, the MaintenanceHold/PrepareHold revision protocol, syncPreparationHold, the hold endpoints and actions, the hold checks in authorizeStandingPreparationTx and loadAuthorityProjectReadiness, and the native hold marker, while keeping the native check that refuses to stop or retire running preparations. If the hold stays, keep one copy. Because factory authority reads the row in SQL, keeping the database row as the authority and making the native marker a plain flag with no revision of its own, set and cleared idempotently, is easier than making the marker the only copy. Also either clear the hold on lifecycle start, or keep it and show it clearly instead of the opaque preparation wait. Fold the three copies of the marker-first sequence into one helper.

**Risk.** The helper's 'running preparations bar hold release' check is a real native guard and should stay if the hold stays.

**Safe first step.** Merge the copied marker-first hold logic. Make apiPreparationActionHold (internal/web/api/preparation_decisions.go:117-139) and the hold branch of apiPreparationHold (internal/web/api/environments_preparation.go:148-174) call one shared helper that returns (revision, ok): it either resyncs at the recorded revision or calls syncPreparationHold. Have lifecycleStopHold (internal/web/api/lifecycle.go:207-227) call syncPreparationHold-style logic without an http.ResponseWriter rather than re-coding the sequence. Keep each endpoint's response fields exactly as they are (hold, hold_revision, marker_synced; hold, hold_synced).

### BND-4: Stored self-digests beside the objects they digest (manifest plan, command, verification manifest, round-tripped manifest digest)

Boundary machinery: strict JSON, digests, receipts, revision checks (cross-cutting). unneeded-validation-or-digest. partially-confirmed. ~200 lines, risk medium. Visible change: UI, model tools, DB schema. Behaviour-preserving: yes. **Status:** owner decision.

Where: `internal/project/factory_manifest.go:62-165`, `internal/project/factory.go:309-315`, `internal/factory/control/dispatch_launch.go:167`, `internal/factory/types.go:63-117`, `internal/factory/grants.go:101-106`, `internal/store/factory.go:459-483`, and 10 more.

**Simpler.** Behaviour-preserving now: - Merge project.PlanDigest into project.ExecutionManifestDigest. They are the same sha256 hex over bytes. - In RecordFactoryCommand, compare the stored type, target and payload instead of the digest. Larger change, needs Go, Rust and frontend edited together: - Drop ExecutionManifest.PlanDigest in Go and in lib/host manifest.rs and events. - Drop Command.Digest along with its column, its trigger clause and the frontend check. - Stop storing VerificationManifestDigest and compute it where needed. It must stay on the wire to the host. - Drop manifest_digest from the submit_task_output and request_handoff MCP schemas and from the host checkpoint and native-task requests, and bind by run ID. This is a model tool contract change. - Use plain json.Unmarshal for internally written manifests.

**Risk.** No external party writes these rows. The verification manifest digest is also passed to the Rust host and the Forgejo side, so that one has to stay on the wire unless those readers change together.

**Safe first step.** Delete project.PlanDigest (internal/project/factory_manifest.go:162-166). Call project.ExecutionManifestDigest instead in factory_manifest.go:109 and internal/factory/control/dispatch_launch.go:167, since both compute the same sha256 hex over bytes. In internal/store/factory.go:478, replace `stored.Digest != c.Digest` with a comparison of stored.Type, Target and Payload against c's. Leave the column and the wire fields untouched for now.

### HFW-5: Muse native-child correlation pins Muse's internal raw-log payload versions and runs redundant per-child cross-checks

Rust host: factory worker runtime. unneeded-validation-or-digest. partially-confirmed. ~200 lines, risk medium. Visible change: none. Behaviour-preserving: no. **Status:** open (behaviour-preserving).

Where: `lib/host/src/factory/native_tasks.rs:137-412`, `lib/host/src/factory/native_tasks.rs:89-98`, `lib/host/src/factory/native_control.rs:299-318,416-494`.

**Simpler.** Keep reading the parent's session log, but only for spawn_accepted (match objective to the persisted start_task prompt and get subagent_id), child_session_bound (child id) and result_ready (completion). Do not compare payload_schema_version or lineage/admitted_execution schema versions. Drop the child-session paging, the inspect_session call, muse_session, the run.model.configured, run started and terminal checks, and MuseChild's observed flags. Treat child_session_bound plus result_ready as the gate for child_started and output observation (fill profiles from the spawn_accepted model and effort). Let the owned-path git checkpoint decide what gets committed. Keep Ledger.profiles because the Codex path uses it, and keep the factory-task agent definition in reserve.rs because the worker prompt names it. If the owner wants the contributor model verified, do that once at preparation, not from the raw log.

**Risk.** Removes a guard against the parent model misdirecting or reusing children; the owned-path checkpoint and the checks remain. Needs one real Muse run to confirm.

### HPL-8: Application setup: string-matched claim-refusal chain reusing factory 409 variants, plus heavy renewal/reconcile-receipt binding

Rust host platform: daemon, terminals, project, application, tailnet (lib/host non-factory, soda-project-maintain, forgejo/pg/console helpers, soda-tailnet, unix-http, wire-time). unneeded-validation-or-digest. partially-confirmed. ~200 lines, risk medium. Visible change: UI. Behaviour-preserving: yes. **Status:** owner decision.

Where: `lib/host/src/application/setup.rs:110-280`, `lib/host/src/application/setup.rs:500-545`, `lib/host/src/application/setup.rs:586-718`, `lib/host/src/application/setup.rs:893-1005`, `lib/host/src/dbackend/backend.rs:87-130`, `lib/host/src/daemon/routes.rs:275-276`, and 4 more.

**Simpler.** Give setup refusals their own typed BackendError variant (or a JSON body such as {"code":"committed"}) instead of reusing ExportStale/OutputStale and matching exact strings. Have Go read that code through the normal callLimit path, and delete callSetupLimit, readNativeSetupResponseLimit and the redundant mapSetupClaimError. Optionally, cut the reconcile stdout binding down to ok plus operation, and drop the renewal tamper predicate on the root-only journal. Keep the fd checks on token.json and receipt.json: the app container can write those files.

**Risk.** The Canlang can-soda CLI contract (schema canlang.soda.setup.v1) is shared with Canlang. Simplifying the host side may need matching Canlang changes. Setup refusal wording in the UI may change.

**Safe first step.** Delete mapSetupClaimError in internal/host/application_setup.go and return the callSetupLimit error directly. setupClaimError.Error() already produces the same three literals, so this has no effect. Then replace the string chain. In lib/host/src/dbackend/backend.rs, swap map_setup_claim_err's ExportStale/OutputStale for a dedicated BackendError::SetupClaim(&'static str) that routes.rs sends as 409 with a fixed code body. In internal/host/client.go, merge callSetupLimit into callLimit and have readNativeSetupResponseLimit read the dedicated code. The API-level codes and messages in internal/web/api/application_setup.go stay the same.

### MCP-7: MCP capability is re-checked several times per call; the Observed hook, revoke/retire transitions and the bound scope re-compare are dead

Model-facing MCP tools. too-many-states-or-budgets. partially-confirmed. ~200 lines, risk low. Visible change: model tools. Behaviour-preserving: yes. **Status:** owner decision; coordinate with prod-hardening.

Where: `internal/web/api/factory_mcp_bound.go:19-46`, `internal/web/api/factory_mcp_bound.go:154-221`, `internal/web/api/factory_mcp.go:65-82`, `internal/web/api/factory_mcp.go:239-278`, `internal/web/api/factory_mcp.go:580-650`, `internal/web/api/factory_mcp_context.go:72-78`, and 9 more.

**Simpler.** First, with no behaviour change: delete MCPRunObservation, FactoryMCPDeps.Observed, factoryCatalogServer.observed, the observed checks in AdmitMCPCapability (keep the Validate, active, expiry and profile/tool checks), the unreachable ObservedState branch in factory_mcp_context.go, admitCall's second GetMCPCapability and ScopeDigest compare when BoundCapability is set (use the bound copy), the handler's second AdmitMCPCapability call, and store ActivateMCPCapability, RevokeMCPCapability, RetireMCPCapability and casMCPCapabilityState, plus the revoked and retired constants (move the tests to ActivateObservedMCPCapability). Then, accepting new refusal text: give the real Prepare reason, make one replay and freeze decision inside RecordMCPSubmission's transaction (keep the guard lock and CheckMCPWorkActive), and drop Generation from validation and the digest (the column can stay at 1).

**Risk.** Keep the stop and withdrawal check (CheckMCPWorkActive) and the transactional capability lock; those are the real guards.

**Safe first step.** internal/factory/control/mcp_admission.go: delete the MCPRunObservation type and the observed parameter and checks from AdmitMCPCapability. internal/web/api/factory_mcp.go: delete the FactoryMCPDeps.Observed field and the factoryCatalogServer.observed method; in admitCall, when BoundCapability is set, use *bound directly and drop the GetMCPCapability reload and the ScopeDigest compare. internal/web/api/factory_mcp_bound.go: drop the Observed block. internal/web/api/factory_mcp_context.go: drop the Observed/ObservedState block at lines 72-78, keeping the field. internal/store/factory_mcp_capability.go: delete ActivateMCPCapability, RevokeMCPCapability, RetireMCPCapability and casMCPCapabilityState, and update or delete the tests that call them. Leave the refusal strings untouched; prod-hardening rewrites them.

### PLN-6: The Architect still has to echo prerequisite digests and mint a 32-hex plan id that SodaOS already knows or could generate; ReplacePlanHead is test-only

Intent to plan (Product Lead, Architect, research, decomposition). unneeded-validation-or-digest. partially-confirmed. ~200 lines, risk low. Visible change: model tools. Behaviour-preserving: yes. **Status:** owner decision; coordinate with prod-hardening.

Where: `internal/factory/control/planning_admission.go:14-127`, `internal/factory/planning.go:25-60,119-140`, `internal/store/factory_planning_revisions.go:13-17,48-98,294-309,359-401,449-476`, `internal/factory/dispatch_prompt.go:65-68,170-185,480-492`, `internal/factory/context_inputs.go:31,56`, `internal/web/api/factory_mcp_plan_schema.go:16-27`, and 2 more.

**Simpler.** Extend prod-hardening's bindIssuePlan so the server stamps the plan id (a fresh random 32-hex) and prerequisite_digests. Take the prerequisite digests from the assignment's prompt-time accepted context (plan_prerequisites), not from a fresh recompute. Drop the "Echo ... prerequisite digests" text from the schema, the prompt and InvestigationPlanGuidance, and stop printing the digests in the investigation prompt. Delete ReplacePlanHead and its tests. Simplify admitInvestigationPlan to save, set the head, then admit. Keep the revision and head tables: the Feed staleness SQL, completion material, retry and readback read them.

**Risk.** The Feed 'Architecture plan recorded' card cites the plan id and digest; generate both server-side. Collapsing the revision and head tables is optional.

**Safe first step.** Delete Store.ReplacePlanHead (internal/store/factory_planning_revisions.go:359-401) together with TestReplacePlanHeadAtomic in internal/store/factory_planning_revisions_test.go. In internal/factory/control/planning_admission_test.go:370 and review_tuple_test.go:154,361, replace the ReplacePlanHead calls with SaveInvestigationPlan+AdvancePlanHead, or delete those tests. There are no production callers, so observable behaviour does not change.

### RBA-9: Duplicated origin/percent-escape validators and a copy of CPython's ipaddress tables in soda-activate and console-welcome

Release build primitives, asset fetch/render tools, small install binaries and system units. duplicated-layer. partially-confirmed. ~200 lines, risk low. Visible change: UI. Behaviour-preserving: yes. **Status:** owner decision.

Where: `cmd/soda-activate/src/origin.rs:6-107`, `cmd/soda-activate/src/origin.rs:109-239`, `cmd/soda-console-welcome/src/origin.rs:34-97`, `cmd/soda-setup/src/origin.rs:7-112`, `cmd/soda-install/src/setup/access.rs:10-61`, `cmd/soda-activate/src/activation.rs:43`.

**Simpler.** Write one small https-origin check on url::Url: scheme https, a host, no userinfo, empty query/fragment, path "" or "/", plus a raw scan for control characters and bad % escapes. Share it from a small crate used by soda-activate, console-welcome and soda-install. soda-setup can keep its http variant or take a flag. Replace the CPython tables with a short rule: reject unspecified, loopback, multicast, and any address that is not RFC1918, link-local, ULA or 100.64/10. 100.64/10 (CGNAT/Tailscale) must stay accepted. Do it together with REL-1.

**Risk.** Corner-case IP verdicts (special ranges such as 198.18/15 or 2001:db8) may differ slightly from CPython. Operators type a LAN IP.

**Safe first step.** In cmd/soda-activate/src/origin.rs, delete the dead checks in check_browser_origin. The `value.contains(['\r','\n'])` check (lines 178-180) can never fire because split_origin already rejects every byte <= 0x20. The `parsed.path()` check (lines 175-177) repeats the raw-path check in split_origin. Then move the byte-identical valid_percent_escapes copies (soda-activate:222, soda-console-welcome:80, soda-setup:95, soda-install access.rs:45) into one shared helper. Leave the IP classifier and the differing query/userinfo rules for a separate change that is allowed to change behaviour.

### RTY-2: Retry eligibility is fully re-validated at the command, preparation, pending-pass and packet steps, plus payload and digest echoes the store recomputes from values the server just built

Retry kinds across store, control and UI. unneeded-validation-or-digest. partially-confirmed. ~200 lines, risk medium. Visible change: none. Behaviour-preserving: yes. **Status:** open (behaviour-preserving).

Where: `internal/factory/control/lifecycle_retry.go:143-206`, `internal/store/factory_explicit_retry.go:43-117`, `internal/store/factory_explicit_retry.go:132-180`, `internal/store/factory_explicit_retry.go:185-259`, `internal/store/factory_explicit_retry.go:261-407`, `internal/store/factory_implementation_retry.go:25-39,162-198,275-309`, and 3 more.

**Simpler.** Keep two eligibility checks: one at request time (immediate refusal) and one in the packet transaction under the dispatch gate (the real serialization point). Drop the store-side SettingsDigest recompute and payload re-decoding in admitRetryCommand, implementationRetryPayload and RecordDeliveryRetryCommand. Keep the digest column and recordFactoryCommandTx's conflict check, because that is the shared command-idempotency key. Instead of echoing prior_outcome in the payload, read the run's outcome when validating. Make PendingImplementationRetry a selection query only (drop its per-pass implementationRetryFactsTx). Make checkRetryPreparationOwnerTx only confirm that the command is pending, and leave the fact checks to the packet. Keep one replay layer. Simplifying PendingExplicitRetry's SQL (including the preparation-stop clause) is a separate behaviour change, because the issue view and planning admission depend on it.

**Risk.** The rechecks guard against state changing between the click and the dispatch pass (a new acceptance, a publication reopening). The ordinary packet and the LatestIssueAssignment/acceptance-head checks it already makes cover most of that, but a short list of 'still latest, still same acceptance' checks should stay at dispatch.

**Safe first step.** Delete the store-side digest recompute and payload re-decoding of server-built commands. In internal/store/factory_explicit_retry.go admitRetryCommand, remove the json.Unmarshal of cmd.Payload and the SettingsDigest comparison (lines ~55-66). Do the same in internal/store/factory_implementation_retry.go implementationRetryPayload (lines ~29-37) and in the RecordDeliveryRetryCommand guard (internal/store/factory_delivery_retry.go:17-24). Pass the prior outcome as a plain argument from control, or read it from the run row in validateExplicitRetryFactsTx. Keep recordFactoryCommandTx's digest-conflict check and the digest field itself. Then drop the implementationRetryFactsTx call inside PendingImplementationRetry (factory_implementation_retry.go:194), because RecordImplementationRetryPacket reruns it under the same gate lock.

### WEB-5: Native authority and repository access are re-fetched 2-7 times per request

Web API and view models (non-MCP, non-Feed, non-extension): internal/web/*.go, internal/web/auth/**, internal/web/api/*.go excluding factory_mcp*, factory_feed*, extension*. duplicated-layer. confirmed. ~200 lines, risk low. Visible change: none. Behaviour-preserving: yes. **Status:** open (behaviour-preserving).

Where: `internal/web/api/lifecycle.go:62-68`, `internal/web/api/environments_api.go:63-69,204-207,234-237`, `internal/web/api/tailnet.go:55-61,179-208,232-279`, `internal/web/api/environments_create.go:115-137,187-204`, `internal/web/api/identity.go:42-62`, `internal/web/api/identity_launch.go:14-45`, and 10 more.

**Simpler.** Verify once, in extensionProtected. Have factoryRepository/visibleRepository memoize repositoryAccess in the request context (or return it), and have factoryOwner, executionRepository and environmentAdministrator take it as input. Delete checkLifecycleSession, requireListedSession, tailnetSession, verifyCurrentSessionMatch, reconfirmRepositoryAndSession, confirmOSSession, admit/confirmProjectTailnetDispatch and auth's revalidateExtensionAuthority. Report session_current as a constant or drop it. Keep periodic re-checks only on long-lived terminal and run-output streams.

**Risk.** This drops checks against the actor changing during a single request, for example after a 3-minute provisioning call. The real boundary, verification at request start on both ends, stays.

**Safe first step.** In internal/web/api/factory_settings.go, make factoryRepository return the repositoryAccess from visibleRepository, and make factoryOwner accept that repositoryAccess instead of re-calling visibleRepository. Update apiFactoryPolicy (factory_policy.go:32-33) and apiFactoryActivity (factory_activity.go:17-33) to pass it through, which removes 1-2 redundant Native().Repository round trips per request. In the same two handlers, drop the checkLifecycleSession call. It runs right after extensionProtected and before any slow work, so the outcome is the same.

### GOM-6: strictjson's duplicate-key scanner and 1 MiB/UTF-8/depth admission for 9 call sites, most with smaller limits of their own, including model tool input

Other Go packages (forgejo, acceptance, project, host, tailnet, application, config, strictjson, filelock, avatar, cmd/soda-dashboard, cmd/soda-tailnet, cmd/soda-extension). unneeded-validation-or-digest. partially-confirmed. ~190 lines, risk low. Visible change: model tools. Behaviour-preserving: no. **Status:** owner decision; coordinate with prod-hardening.

Where: `internal/strictjson/decode.go:1-144`, `internal/strictjson/README.md`, `internal/web/api/factory_mcp_issue_plan.go:48`, `internal/web/api/factory_mcp_issue_plan_transport.go:35`, `internal/web/auth/http.go:73`, `internal/factory/control/operator.go:69`, and 9 more.

**Simpler.** Delete the Go package, its README, its rows in README.md and docs/development/go.md, the strictjson tests in scripts/wire_contracts_test.go, and the strictjson_vectors.json fixture. DecodeAPIObject and operator.go already limit body size, so they only need json.NewDecoder with DisallowUnknownFields and a check that nothing follows the object. Terminal frames and the stored profile need only json.Unmarshal. The MCP transport (map[string]any) can use json.Unmarshal, because the JSON schema check already refuses unknown fields. issue_plan.go:48 should decode leniently or report the expected fields. Coordinate both MCP files with the prod-hardening branch. The Rust ports (soda-identity strict.rs, lib/host json and dbackend) are a separate, related simplification.

**Risk.** Browser API handlers lose their duplicate-key refusal. No untrusted party relies on duplicate keys, and auth still runs. The MCP part must be done together with the hardening branch.

### JDG-7: Two TypeSafe clients for one endpoint, repeated distribution and state validation on the delivery path, and strict refusal of slightly-off provider answers

Completion, reassessment, review and JEV judgments. duplicated-layer. partially-confirmed. ~180 lines, risk medium. Visible change: model tools. Behaviour-preserving: yes. **Status:** owner decision; coordinate with prod-hardening.

Where: `internal/factory/control/systemone_typesafe.go:17-294`, `internal/factory/control/priority_typesafe.go:17-141`, `internal/factory/control/delivery_typesafe.go:11-42`, `internal/factory/control/judgment_advice_delivery.go:80-83,111-114`, `internal/factory/control/delivery_judgment_producer.go:13-16,135,274`, `internal/factory/judgment.go:158-183,327-382`, and 6 more.

**Simpler.** Use one small JEV transport: Ask(ctx, state, questions) returning the raw answers, model and usage. It owns the key file, the endpoint, the model default, the client default and one response cap. TypeSafeSystemOne and TypeSafePriority become thin answer projections over it. Validate each answer once, where it enters, and remove the repeated Validate and ValidateDeliveryState calls on the delivery path. Inline DeliveryModel/SelectDelivery into the priority client, but keep binding the model into the delivery fingerprint, and keep DeliveryAnswer.Validate for loaded records. Construct the client once in server.go and share it. Optionally replace the strict score, legend and token checks with a lenient normaliser (renormalise, argmax, clamp). That last step is a separate change in behaviour.

**Risk.** Lenient normalisation accepts answers that are refused today. The worker tool returns the provider's answer as-is in both cases.

**Safe first step.** In internal/factory/control, add one private helper that reads the key, defaults the model to jev-latest, builds the default client, POSTs to the endpoint and returns the body. Use it from both TypeSafeSystemOne.evaluateQuestionMap (systemone_typesafe.go:120-156) and postPriorityRequest (priority_typesafe.go:63-89), keeping each caller's error text and response cap so behaviour does not change. Delete the second ValidateDeliveryState call in SelectDelivery (delivery_typesafe.go:28-30). AdviseDelivery already runs it at judgment_advice_delivery.go:80-83, and it is the only caller of SelectDelivery. Then make DeliveryModel the one source of the jev-latest default for Select and Evaluate.

### MCP-9: Posting surface: ten near-identical communication outlets (plus resolve_post), each needing a minted id plus a Feed card choice, validated and decoded twice

Model-facing MCP tools. duplicated-layer. partially-confirmed. ~180 lines, risk medium. Visible change: UI, model tools. Behaviour-preserving: yes. **Status:** owner decision; coordinate with prod-hardening.

Where: `internal/factory/mcp_profile.go:104-113,288-297`, `internal/factory/mcp_communication.go:14-31,285-326`, `internal/web/api/factory_mcp_communication.go:13-60`, `internal/web/api/factory_mcp_bound.go:62-94`, `internal/web/api/factory_mcp.go:472-479,506-508,597-607,909`, `internal/store/factory_mcp_communication.go:19-35`, and 2 more.

**Simpler.** Make one factory.post tool with a kind enum, title, summary, optional fields and reply_to, and have the server mint the ids. resolve_post becomes a kind. The Feed derives presentation from kind and content, with at most one optional variant where a role has a real alternative. Keep one required-field table and decode and validate once. Communications are already checked only twice, not three times: the read-time feedSubmissionComponents check applies only to terminal finish tools.

**Risk.** The Feed catalog's per-outlet mapping must key on kind; coordinate with the Feed catalog finder.

**Safe first step.** In internal/store/factory_mcp_communication.go recordMCPCommunicationTx, delete the repeated MCPToolAdmitted check (lines 27-29) and the repeated ValidateFeedComponent call (31-33). Both are already enforced before the transaction: AdmitMCPCapability at control/mcp_admission.go:90, and admitMutation at api/factory_mcp.go:597-607. Also have checkCommunicationPayload (api/factory_mcp_communication.go:53-60) and the store share a single decode. Only do this after confirming that every path into recordMCPCommunicationTx goes through admitMutation, and coordinate with the hardening branch, which edits factory_mcp.go.

### CTX-8: Context reads re-validate self-written rows (accepted context digests, review cycles/journals/sources, receipts) and hard-fail on mismatch

Worker context assembly, project library and research. unneeded-validation-or-digest. partially-confirmed. ~170 lines, risk low. Visible change: none. Behaviour-preserving: yes. **Status:** open (behaviour-preserving).

Where: `internal/factory/context_inputs.go:73-91`, `internal/store/factory_context_current.go:166-178,185-190,262-311,321-327`, `internal/store/factory_context_review.go:17,26-46`, `internal/store/factory_context_review_facts.go:12-124`, `internal/web/api/factory_mcp_context.go:124-136`, `internal/factory/control/context_bundle_state.go:92-102`.

**Simpler.** Read the rows and project them. Keep the scoping joins (assignment ID, publication assignment), the capability state/expiry check, the acceptance head/withdrawal EXISTS check and the CheckAssessment selection filter. Delete ValidateAcceptance entirely. Drop the read-side ValidateFor calls (the functions stay for Assignment.Validate), the cycle equality check, the DeepEqual journal check, currentReviewSourceTx's field-equality checks and the per-row Validate calls. Share one helper for the verification missing-reason filter.

**Risk.** A corrupted row would surface later or as odd context, not as a refusal. No adversarial writer exists.

**Safe first step.** In internal/store/factory_context_current.go, remove the a.AcceptedContext.ValidateFor(a) and ValidateAcceptance(accepted) calls at lines 185-190. Delete AcceptedContext.ValidateAcceptance from internal/factory/context_inputs.go:73-91, along with its test-only uses in dispatch_test.go and factory_mcp_live_test.go. Remove the cycle struct-equality branch in internal/store/factory_context_review.go:41-43, which can never fire because publicationJournalsStable makes cycles immutable. Leave the acceptance head/withdrawal EXISTS check and the CheckAssessment filter untouched.

### STO-5: factory_dispatch_regs is a write-only shadow of assignment IDs, and its never-pruned history makes Pause fail after 1024 dispatches

Store (PostgreSQL layer) structure. duplicated-layer. partially-confirmed. ~160 lines, risk low. Visible change: UI, DB schema. Behaviour-preserving: yes. **Status:** owner decision.

Where: `internal/store/schema.go:72-73`, `internal/store/schema.go:253-254`, `internal/store/schema.go:377`, `cmd/soda-identity/src/schema.rs:67-68`, `cmd/soda-identity/src/schema.rs:286-287`, `cmd/soda-identity/src/schema.rs:344`, and 12 more.

**Simpler.** Keep the factory_dispatch gate row: lock it FOR UPDATE inside the packet and refuse when it is closed, as the first half of registerDispatchTx already does. Drop factory_dispatch_regs (in both the Go schema and the Rust soda-identity schema mirror) with its triggers and probe, Store.RegisterDispatch and the dead Coordinator.RegisterDispatch wrapper, ErrDispatchConflict, Withdrawal.Captured, GrantReceipt.Captured, MaxCapturedDispatch and the extra EXISTS collision probes. Either drop the F08 "Outstanding work" item or derive its count from factory_assignments WHERE repository=$1 AND stage='assigned'.

**Risk.** The gate semantics stay. The only observable difference is the missing 'captured' array in receipts nobody renders, plus the fix for the latent 1024-dispatch Pause failure.

**Safe first step.** Delete the uncalled Coordinator.RegisterDispatch in internal/factory/control/dispatch_registration.go (lines 14-33) and Store.RegisterDispatch in internal/store/factory_grants.go (lines 191-207; only tests call it), along with the tests that call them. Production paths reach registerDispatchTx only through the packet and retry transactions, so observable behaviour stays identical.

### APP-1: Factory build-to-app delivery re-derives the verified-delivery chain several times per request. The application-builds routes are blocked at the extension proxy

Application runtime: build, home, snapshots, backups, restore, factory delivery. duplicated-layer. partially-confirmed. ~150 lines, risk medium. Visible change: UI. Behaviour-preserving: yes. **Status:** owner decision.

Where: `internal/store/application_factory_delivery.go:35-182`, `internal/store/application_builds.go:61-171`, `internal/web/api/application_builds.go:31-146`, `internal/web/api/application_build_replacement.go:11-44`, `internal/web/api/applications.go:79-90,141-164`, `internal/store/applications.go:291-312`, and 2 more.

**Simpler.** First, admit application-builds (GET with ?delivery_id, POST, and {build}/setup POST) in extensionRepositoryRoute and extensionQueryRoute (WEB-1). Second, compute VerifiedApplicationDelivery once per application in applicationView and launchableApplications and pass it into applicationEnvironment. Third, have the POST build handler pass its already-verified request and delivery into SaveApplicationBuild instead of re-querying. Keep lockBuildIntent as the head check. Next, keep only one of the Rust PrivateState artifact cache and the Go application_builds row. Only after accepting the behaviour change: drop the stale-head re-checks at setup and replacement, and derive delivery and environment from the newest application_builds row matching Spec.Image instead of the VerifiedApplicationDelivery join and the contradiction logic.

**Risk.** The delivery_id shown on launch rows and the environment label must still come out the same. An image-digest join gives the same answer while builds are unique per (delivery, environment) and images are content-addressed.

**Safe first step.** In internal/web/api/applications.go, call s.Store.VerifiedApplicationDelivery once per record in applicationView (line 83) and in launchableApplications (line 143). Change applicationEnvironment(r, id) to take that already-fetched delivery and error as parameters instead of calling it again at line 159. This removes two of the four identical join transactions per application row with the same outputs.

### APP-2: Redundant image-mount readbacks of bundle.json (post-build, cached build, verify_build) plus Observation carry-forward and custom decoder

Application runtime: build, home, snapshots, backups, restore, factory delivery. unneeded-validation-or-digest. partially-confirmed. ~150 lines, risk low. Visible change: UI. Behaviour-preserving: yes. **Status:** owner decision.

Where: `lib/host/src/application/build.rs:1-64`, `lib/host/src/application/package.rs:263-292`, `lib/host/src/application/runtime.rs:419-430`, `lib/host/src/application/wire.rs:161,208`, `internal/web/api/application_actions.go:174-181`, `internal/store/application_builds.go:173-179`, and 8 more.

**Simpler.** Keep BuildArtifact.bundle_file_sha256: it is hashed in memory at package.rs:178 and ties the build to the factory verification checks. Delete the mount-and-rehash readbacks: the one after the build in package_application, and the one in check_build_artifact, which should keep only its admit_image call plus its runtime and tools checks. Then, together with APP-1, re-key VerifiedApplicationDelivery on the observation's image through application_builds (image to artifact, which carries run and bundle_file_sha256). After that, remove bundle_file_sha256 from Observation, the verify_build action (Rust, Go, frontend, docs, oracle test), the carry-forward block in performApplicationOperation, and applicationResponse's custom UnmarshalJSON.

**Risk.** Do this together with APP-1 so the delivery identity on app cards keeps resolving.

**Safe first step.** In lib/host/src/application/package.rs, delete the post-build readback (lines 267-270, the `if self.bundle_file_sha256(...) != artifact.bundle_file_sha256 { return Err(...) }`). In check_build_artifact, replace the `self.bundle_file_sha256(&artifact.references(), deadline)? != artifact.bundle_file_sha256` comparison with a plain `self.admit_image(&artifact.references(), deadline)?` call, so the label and existence check stays. The image is content-addressed and built from bytes already hashed at package.rs:178, so outcomes are identical. build.rs then serves only verify_build, and is removed in the APP-1 step together with verify_build.

### APP-4: Backups: dev/ino anchor-rebinding around snapshot.rs for lane-ownership reasons, 64-cap/shape validation repeated in Rust, Go and frontend, and repeated full-tree hashing on the restore path

Application runtime: build, home, snapshots, backups, restore, factory delivery. unneeded-validation-or-digest. partially-confirmed. ~150 lines, risk low. Visible change: UI. Behaviour-preserving: yes. **Status:** owner decision.

Where: `lib/host/src/application/backups.rs:27-264`, `lib/host/src/application/snapshot.rs:180-357`, `lib/host/src/application/restoration.rs:230-290`, `internal/host/application_snapshots.go:17-30`, `internal/application/snapshot.go:1-62`, `frontend/spaces/sodaspaces-application-response.ts:368-397`, and 1 more.

**Simpler.** In backups.rs, drop root_anchored and all the open_child_dir_opt/fd_matches_path rebinding. Validate the storage root and snapshots/ once with an ownership and mode check, then call snapshot.rs with plain paths. The writer flock and the stopped-container check stay. Drop the Go re-validation loop in host/application_snapshots.go; keep only the confirmation that the backup or inspect id is present. Remove the frontend duplicate check. Run verified_snapshot on the source once per restore instead of up to three times. Keep the client-supplied id, because it is what makes retries idempotent. Removing the 64 cap and the complete flag is optional and requires raising or removing the 128 KiB response limit.

**Risk.** The real requirement, a stopped writer (flock on .writer-lock), stays.

**Safe first step.** 1. In lib/host/src/application/backups.rs, delete the post-call rebind blocks, where each leaf is re-opened and fd_matches_path is called again after snapshot::application_info or backup_application (inspect at about lines 112-121; backup "fresh" at about 175-186). Delete the per-entry open_child_dir_opt/fd_matches_path skip in the list loop, along with the FJ-847/FJ-855 comments. Under a root-owned 0700 tree, none of these can change the result. 2. In internal/host/application_snapshots.go, delete the duplicate `seen` and `snapshot.Validate()` re-checks; keep the presence check for the backup or inspect id. Leave the 64 cap, the complete flag and the client id unchanged in this step.

### BND-1: MCP tool envelope makes the model carry capability_id and expected_input_digest that the bound server already knows; payload hash is checked against itself

Boundary machinery: strict JSON, digests, receipts, revision checks (cross-cutting). unneeded-validation-or-digest. confirmed. ~150 lines, risk low. Visible change: model tools. Behaviour-preserving: yes. **Status:** done (`42a051e9`).

Where: `internal/web/api/factory_mcp.go:91-135`, `internal/web/api/factory_mcp.go:248-270`, `internal/web/api/factory_mcp.go:604-620`, `internal/web/api/factory_mcp_bound.go:50-61`, `internal/web/api/factory_mcp_bound.go:154-218`, `internal/factory/mcp_submission.go:111-125`, and 16 more.

**Simpler.** Remove capability_id and expected_input_digest from mcpCallArgs and from every tool schema. - admitCall uses deps.BoundCapability directly. Keep the active, expiry and stop admission, and the existing store-side lock and recheck at insert time. - Drop MCPSubmission.ExpectedInputDigest, ErrMCPSubmissionStale, the task_checkpoint ScopeDigest rechecks, the store ScopeDigest rechecks, and the three per-tool compares. - Drop the expected_input_digest field from the current-context receipt projection. - Keep the input_digest column filled with the server ScopeDigest, or drop it later. Remove the redundant s.input_digest filters in repair_scope, since the capability ID already pins the scope. - Drop the PayloadHash recompute in Validate. Keep one payload hash for idempotent replay. - Delete the unbound-transport branches and their test fixtures.

**Risk.** The real security boundary is the HMAC bearer on the bound handler, and it stays. The digest echo protects against nothing an attacker can do once the bearer is checked. The DB column input_digest is NOT NULL with a CHECK, so it either needs a fill value or a column drop.

**Safe first step.** In internal/web/api/factory_mcp.go admitCall, delete the re-read ScopeDigest-vs-bound compare at lines 263-265, the "worker context changed" refusal. It cannot fail, because the scope fields of a capability row are never updated. Also drop the PayloadHash recompute from the API path, since PayloadHash is computed at line 616 from the same raw bytes; this means moving that check out of MCPSubmission.Validate or skipping it for freshly built submissions. Neither change alters observable behaviour, and the tool schemas stay as they are.

### BND-6: The six grant settings forms (policy, capacity, operator grant, sponsorship, usage budget, environment grant) go through command_id idempotency, a payload digest, a factory_commands row, a triple revision check and a GrantReceipt that the UI never reads; the client sends a fresh id each time

Boundary machinery: strict JSON, digests, receipts, revision checks (cross-cutting). process-machinery. partially-confirmed. ~150 lines, risk medium. Visible change: none. Behaviour-preserving: yes. **Status:** open (behaviour-preserving).

Where: `internal/factory/control/grants.go:19-151`, `internal/store/factory_grant_commands.go:180-539`, `internal/web/api/factory_sponsorship.go:24-66`, `internal/web/api/factory_sponsorship.go:149-190`, `internal/web/api/factory_policy.go:30-158`, `internal/web/api/factory_connection_usage_budget.go:14-74`, and 4 more.

**Simpler.** For the six grant records, the server validates the record and does a single revision-guarded upsert (saveRevisionedGrantQuery already exists) inside the same transaction that does the dispatch, publication and merge withdrawal. No factory_commands row, no SettingsDigest, no GrantReceipt outcome, and no command_id from the client. Collapse the six Coordinator.Apply* wrappers and the six Store.Apply*Command one-liners into one save path. Drop the redundant record.Revision != expected pre-check and the unreachable stale_revision replay branch. Leave repair grants, the verification allowance, acceptance, retry and delivery repair on the journal: there the journal holds the state, or the client really does reuse the id. Activity settings already follow the simple pattern. Update the installed probe so it no longer requires a policy command row.

**Risk.** Withdrawal closes dispatch through the same path (applyGrant), so the dispatch-closing side effect has to stay. factory_commands is also reused as a journal by delivery repair and retry, so only the settings types would leave it.

**Safe first step.** Three changes that keep observable behaviour identical: 1. In internal/factory/control/grants.go, delete the `record.Revision != expected` pre-checks in ApplyPolicy, ApplyCapacity, ApplyOperatorGrant, ApplySponsorship, ApplyConnectionUsageBudget and ApplyEnvironmentGrant. The handlers build each record with Revision = ExpectedRevision, so these checks never fire; keep the Repository > 0 checks. Delete the unreachable `failure.Error == "stale_revision"` branch in replayGrant, because a stale CAS rolls back before the command row persists. 2. In internal/store/factory_grant_commands.go, replace the six identical Apply*Command one-liners (lines 180-202) with direct calls to applyFactoryGrantCommand from applyGrant. That removes the type switch in grants.go:100-115. 3. Dropping the command row, digest and receipt for the six grants is the larger step. Leave the client's expected_revision and the SQL CAS in place, because removing them changes the 412 stale_revision behaviour. That step also needs internal/acceptance/factory_context_status.go:236-241 updated, since it reads the policy command row.

### DSP-8: Write-only dispatch registration table that makes pause fail after 1,024 dispatches

Factory dispatch and run start. dead-code. partially-confirmed. ~150 lines, risk low. Visible change: UI, DB schema. Behaviour-preserving: yes. **Status:** owner decision.

Where: `internal/store/factory_grants.go:191-260`, `internal/store/factory_grants.go:356-405`, `internal/factory/control/dispatch_registration.go:14-34`, `internal/store/factory_dispatch_packet.go:56-69,98-104`, `internal/store/factory_grant_commands.go:311-315`, `internal/factory/grants.go:429,450-468,482-515`, and 8 more.

**Simpler.** Keep only the gate-open check inside the packet transaction (ensureDispatchGateTx / SELECT open FROM factory_dispatch FOR UPDATE, refusing when closed). Delete DispatchRegistration and its parameter threading, the factory_dispatch_regs table and triggers (both Go and Rust schema), Withdrawal.Captured and GrantReceipt.Captured, MaxCapturedDispatch, both RegisterDispatch functions and the four EXISTS(factory_dispatch_regs) collision clauses. Drop the misleading 'Outstanding work: N recorded dispatches' item from the F08 pause card, or reword it.

**Risk.** Nothing reads the ledger except redundant collision checks. The latent pause failure goes away as a side effect.

**Safe first step.** Delete Coordinator.RegisterDispatch (internal/factory/control/dispatch_registration.go:14-34), which has no callers. Delete Store.RegisterDispatch (internal/store/factory_grants.go:191-207), which only tests call, along with the factory_grants_test.go cases that exercise it. Also remove the redundant 'OR EXISTS(SELECT 1 FROM factory_dispatch_regs WHERE id=$1)' clauses at factory_explicit_retry.go:95,195, factory_implementation_retry.go:142 and factory_delivery_retry.go:48. Registration ids always equal assignment ids written in the same transaction, and production never deletes factory_assignments, so behaviour stays identical.

### JDG-6: Carried findings are rebuilt and fully re-validated from publication history at five call sites

Completion, reassessment, review and JEV judgments. unneeded-validation-or-digest. partially-confirmed. ~150 lines, risk medium. Visible change: none. Behaviour-preserving: yes. **Status:** open (behaviour-preserving).

Where: `internal/factory/control/review_cycle.go:164-469`, `internal/factory/control/review_cycle.go:1124-1138`, `internal/factory/control/review_cycle.go:516`, `internal/factory/control/delivery_judgment_material.go:137-150`, `internal/factory/control/delivery_producer.go:53`, `internal/factory/control/correction_result.go:227`, and 1 more.

**Simpler.** Keep the replay (oldest content wins; a key retires only on a verified assessment), but stop re-proving history that was already admitted. Delete checkPriorCycleBindings, checkPriorReviewerEvidence (except reading the typed output), checkPriorRunJoins (FactoryRun plus the publicationAssignmentsForRun join), and the per-round Review/Clear Validate and coverage-digest recomputation in processPriorTypedReview and processPriorClearReview. Those checks already ran when the verdict was bound in bindPhasedReviewVerdict. Each prior round then becomes 'load reviewer assignment, merge NewFindings, retire verified keys'. Optionally, later, cache the computed set on the publication at admission. That needs a new persisted publication field and store append-only rules, so it is not free.

**Risk.** Correction admission depends on carried-set semantics (oldest content wins; a key retires only on a verified assessment). Computing the set at the same admission points keeps those semantics identical.

**Safe first step.** In internal/factory/control/review_cycle.go, delete checkPriorRunJoins (steps 18-20, the FactoryRun and publicationAssignmentsForRun calls) and checkPriorCycleBindings (steps 01-04). Drop the Review.Validate and Clear.Validate, FindingContentDigest and ClearCoverageDigest re-checks (steps 10, 12-15) in processPriorTypedReview and processPriorClearReview. Keep the carriedCollector add/retire merge and the journal ordering unchanged. Delete the refusal cases in review_carried_history_test.go that test only these removed checks. For history that was validly admitted, the carried set returned to all five callers is identical.

### VRF-5: 'App verified' is an F10 Feed card read as state, with native evidence re-derived at 6 call sites and re-checked again on record

Test Pilot / app verification (Go factory+control+store+MCP, Rust host, in-project helper). duplicated-layer. partially-confirmed. ~150 lines, risk medium. Visible change: model tools. Behaviour-preserving: yes. **Status:** owner decision.

Where: `internal/store/factory_app_verification.go:51-78`, `internal/store/factory_app_verification.go:199-332`, `internal/factory/control/app_verification.go:23-175`, `internal/factory/control/delivery_admission.go:70-80`, `internal/factory/control/delivery_admission.go:152-175`, `internal/factory/control/delivery_repair.go:238`, and 8 more.

**Simpler.** In RecordVerifiedApplication, write one verified marker (assignment jsonb) and make F10 a pure projection. Switch the 5 F10 queries to read the marker together. In RecordVerifiedApplication, trust the evidence verifyApplicationReport just built and drop the repeated preparation, state and approval re-checks. Keep the tip and readiness check, but make PendingAppVerifications skip superseded deliveries so stale reports stop being re-polled. Give the worker the evidence through one channel (the tool or factory.context), not also embedded in the prompt. Remove the second ValidateFor in context_bundle_state.go. Merge the duplicated first and retry prompt payload builders in delivery_admission.go. Keep the outcome strings, because they name different states.

**Risk.** The 5 queries feed delivery/packaging and journey readback and must switch together. FED-9 changes only the F10 card text, so the two do not conflict.

**Safe first step.** In internal/factory/control/context_bundle_state.go:93, delete the redundant `&& evidence.ValidateFor(*current.Role.Delivery) == nil`. verify is c.ApplicationVerificationEvidence (dispatch.go:429), which already returns that same ValidateFor result as its error, so behaviour is identical. Then, in internal/factory/control/delivery_admission.go, merge the identical anonymous payload struct, marshal and prompt-building code in the first admission path (~lines 73-80) and deliveryRetryPrompt (~lines 161-175) into one helper, keeping the two prompt prefixes byte-for-byte.

### APP-5: Lifecycle actions carry container-generation fences, revision echo, observation CAS with pre-call nulling and post-call generation re-checks, on top of a per-app mutex

Application runtime: build, home, snapshots, backups, restore, factory delivery. unneeded-validation-or-digest. partially-confirmed. ~130 lines, risk medium. Visible change: UI. Behaviour-preserving: no. **Status:** owner decision.

Where: `internal/web/api/application_actions.go:13-189`, `internal/store/applications.go:172-201`, `internal/application/administration.go:35-119`, `internal/host/application.go:34-76`, `lib/host/src/application/wire.rs:157-201`, `lib/host/src/application/runtime.rs:95-127,385-424`, and 4 more.

**Simpler.** For the lifecycle actions (install, inspect, start, stop, verify_build, replace), the body becomes {action} plus replacement. Confirm stays in the UI only. The server holds the per-app mutex. Rust resolves the container by name and acts. Go writes the returned observation without a CAS and drops the pre-call nulling. On a native error it runs inspect automatically instead of storing unknown. Delete revision and confirm_stop from the action body, delete admitApplicationGeneration, and delete applicationGenerationConfirmed/applicationActionConfirmed. Make validGeneration apply to the restore actions only. Keep SameObservation and the expected_container field, because restore (store/application_restore.go) and snapshot backup still use them. Update the delivery replace call, the qualification test body assertions and api.md to match.

**Risk.** Setup and restore also use expected_container (setup is HPL scope). Simplify lifecycle actions first and leave setup alone.

### BUD-12: Most admission limits are checked twice: control pre-checks repeat the packet transaction's SQL checks (but also gate side effects and the unsettled-run rule)

Budgets, allowances and limits (cross-cutting). duplicated-layer. partially-confirmed. ~120 lines, risk medium. Visible change: UI. Behaviour-preserving: yes. **Status:** owner decision; coordinate with prod-hardening.

Where: `internal/factory/control/dispatch_selection.go:13-140`, `internal/factory/control/dispatch_occupancy.go:1-55`, `internal/store/factory_dispatch_packet.go:1060-1209`, `internal/factory/control/dispatch_attempt.go:220-256`, `internal/factory/control/dispatch_attempt.go:389-391`, `internal/factory/control/dispatch_launch.go:293-318`, and 3 more.

**Simpler.** Read the connection usage budget once in selectSponsorship and pass it on through the plan. Then, after BUD-1 (fixed per-run deadline) and a soft spend cap: move the not-settled-run rule into checkAdmissionLimitsTx and take applianceConcurrent/repositoryConcurrent from the locked grants in the transaction. Then delete checkConcurrencyLimits, checkSponsorshipBudget and passOccupancy/refreshOccupancy, and rely on admissionWait. Accept that limit waits will be reported after the preparation, harness and input waits rather than before them.

**Risk.** Depends on BUD-1. The packet transaction already rolls back refused admissions cleanly. A pass may build a prompt it then cannot admit, which is a little wasted work.

**Safe first step.** In internal/factory/control/dispatch_selection.go, keep the factory budget value read in selectSponsorship (line 39) on attemptPlan, for example as plan.usageBudget. Then delete the second deps.Store.ConnectionUsageBudget read and its revision check in checkSponsorshipBudget (lines ~122-128), using plan.usageBudget.RollingMinutes instead. The first read already confirmed the same revision, and checkAssignmentAuthorityTx re-checks it under lock. Observable behaviour is the same except in a race window of microseconds, where the detail text would read "grants changed during dispatch" instead of "connection usage budget changed during dispatch"; the wait reason (WaitAuthority) stays the same. Removing the rest has to wait for BUD-1.

### HFW-9: Run finish round-trips output and a Muse credential nobody uses, and retires the unit twice

Rust host: factory worker runtime. duplicated-layer. confirmed. ~120 lines, risk low. Visible change: none. Behaviour-preserving: yes. **Status:** open (behaviour-preserving).

Where: `lib/host/src/factory/native_control.rs:396-407`, `lib/host/src/terminal/factory/muse/start.rs:9-55`, `lib/host/src/factory/launch.rs:302-360`, `lib/host/src/terminal/factory/lifecycle.rs:116-161`, `lib/host/src/terminal/factory/muse/lifecycle.rs:64-83`, `lib/host/src/factory/finish.rs:120-320`, and 5 more.

**Simpler.** Return the native result directly, and make terminal.wait only wait for the unit to go inactive. Drop the exit-file cat and the head -c read. Drop the last-message.txt staging too, but keep a liveness check where one is needed. Stage the Muse credential once (auth.json only). For Muse leases, skip capture and just return or forget the lease. Make the host's stop_native the only retirement, and have the broker callback skip factory leases the host has already retired. Keep capture and return for Codex.

**Risk.** Codex leases rotate credentials through capture/return, so keep that path while Codex exists. The broker-callback change lives in cmd/soda-identity, outside this scope.

**Safe first step.** In lib/host/src/terminal/factory/lifecycle.rs factory_wait_result, delete the podman exec cat of {run_dir}/exit and the head -c 65537 read of the output file, and keep only the systemctl inactive poll. In lib/host/src/factory/launch.rs, use `native` directly where it now does native.unwrap_or(wait result). native is always Some in production (dbackend/factory_terminal.rs inserts channels for both families), and cat/head failures are already swallowed, so observable behaviour stays the same.

### MCP-11: Product Lead hand-copies a verification manifest whose content the server fully determines (only opting in is a real choice)

Model-facing MCP tools. speculative-abstraction. partially-confirmed. ~120 lines, risk medium. Visible change: model tools. Behaviour-preserving: yes. **Status:** owner decision; coordinate with prod-hardening.

Where: `internal/factory/mcp_manifest.go:27-51`, `internal/factory/mcp_manifest.go:76-81`, `internal/factory/mcp_manifest.go:109-113`, `internal/factory/mcp_manifest.go:236-316`, `internal/web/api/factory_mcp.go:535-578`, `internal/web/api/factory_mcp.go:764-774`, and 7 more.

**Simpler.** Replace verification_manifest_proposal in the Product Lead contract with a boolean opt-in, or drop the opt-in only if the owner decides every plan gets project.check verification. Build the proposal on the server at adoption as {Outcomes: ResolvedScope(head), Checks: one ApprovedAppCheckID ref per outcome with the catalog's receipt artifact and prepared fixture}, so the persisted AdoptedVerificationManifest keeps its shape. Delete the parameter, grant and per-check artifact/fixture validation, the ApprovedCatalog interface, checkManifestProposal, the Product Lead verification_catalog view and the related guidance text. Update the store's proposal-digest cross-check to compare against the server-built proposal. There are no ManifestCoverage or ManifestUncovered helpers to delete.

**Risk.** The adoption and Test Pilot paths consume AdoptedVerificationManifest; build it in one place. Coordinate with the verification finder.

**Safe first step.** In internal/factory/mcp_manifest.go, delete ApprovedCheck.RequiredParams, ParamValueChars and AllowedParams, the MCPMaxCatalogParams and MCPMaxCatalogParamChars constants, and the bound/AllowedParams/RequiredParams loop in validateCatalogRef. Replace that loop with `if len(params) != 0 { return errors.New("invalid check parameter name") }`, keeping the ManifestCheckRef.Params field so schema decoding is unchanged. Also replace the ApprovedCatalog interface with the concrete ApprovedProjectCheckCatalog in ValidateManifestProposal, FactoryMCPDeps.Catalog (internal/web/api/factory_mcp.go:83) and factory_mcp_verification_catalog.go. Refusal outcomes stay identical.

### PRJ-5: Creation profile with one allowed value is re-derived, compared and validated about nine times, and keeps a legacy no-profile path

Project environment runtime on the host (create/join/start, executor, domain, ssh, accounts). unneeded-validation-or-digest. confirmed. ~120 lines, risk low. Visible change: UI. Behaviour-preserving: yes. **Status:** owner decision.

Where: `lib/host/src/domain/profile.rs:81-139`, `lib/host/src/project/profile.rs:66-146`, `lib/host/src/project/create.rs:36-45`, `lib/host/src/project/create.rs:116-126`, `lib/host/src/project/inspect.rs:371`, `internal/host/profiles.go:13-23`, and 9 more.

**Simpler.** Resolve and validate the profile once at create, in Rust, and write the label. On inspect, decode the label without re-validating it and without cross-checking it against org.soda.profile and the image. Go stores the profile it received and reads it back with plain json.Unmarshal. Drop environmentProfileMismatch (three call sites) and the profile re-check in os.go. Make the profile non-optional in Rust and Go and delete the 'legacy / unknown' branch in Rust, Go and the UI. Leave the nullable DB column and its trigger as they are, so the schema does not change.

**Risk.** Only the containers this host creates carry the label. A mismatch could only come from someone hand-editing podman labels as root.

**Safe first step.** Three changes are safe to make first: - In internal/store/store.go CreateProject, delete the unreachable p.Profile==nil insert branch (the only caller, environments_create.go:258, always sets Profile) and the redundant p.Profile.Validate() (the profile was already validated by host.ResolveProfile). - In internal/host/os.go validOSEnvironment, drop the `env.Profile == nil || env.Profile.Validate() == nil` re-check (the Rust inspect already validated the label). - In internal/host/profiles.go, drop the duplicate Validate and arch re-check (Rust resolve_profile already enforces both). For a valid profile the observable behaviour is identical. Then delete the tests that cover the removed code.

### TRM-3: The creation 'scope' digest has two derivations, is re-validated at five layers, and in the identity path is compared with itself

Human terminal stack end to end (browser, Go API, host, in-project helper). unneeded-validation-or-digest. confirmed. ~120 lines, risk low. Visible change: none. Behaviour-preserving: yes. **Status:** open (behaviour-preserving).

Where: `internal/web/api/terminal_registry.go:32-36`, `internal/web/api/extension_terminal_authority.go:25-28`, `internal/web/api/extension_terminal.go:125`, `internal/web/api/extension_terminal_stream.go:99`, `internal/web/api/identity_launch.go:33`, `internal/identity/terminal.go:10`, and 10 more.

**Simpler.** Delete the scope field from TerminalRequest/TerminalStart, the Rust wire/stream/identity structs and the guest argv. Delete it from the reservation file, but keep the file's expiry and one-use consumption. Delete it from the subscription profile. Delete TerminalCreationScope, nativeTerminalScope and all five validators. The identity-path check is a guaranteed tautology. In the human path, Go already checks that the session is current; the only check lost is that create runs in the same session generation as reserve, which matters only for the same user within the 120-second reservation window. This goes naturally with TRM-2.

**Risk.** No real adversary: the value comes from Go, travels through the root host and is compared in the guest. The argv arity (11) changes in both host and guest together.

**Safe first step.** Remove the scope from the identity path only, where it never changes the outcome. Delete TerminalCreationScope (internal/web/api/terminal_registry.go:32-36) and the Scope field of identity.TerminalStart (internal/identity/terminal.go:10) and its use in identity_launch.go:33. Remove the scope check in lib/host/src/terminal/identity.rs terminal_preparation and the scope field in its struct/serializer (lines 24, 42-44, 108-117, 170-188). In cmd/soda-project-terminal/src/subscription_prepare.rs, stop reading request "scope" and drop the scope argument from profile_object (subscription_wire.rs:175-184), which nothing reads. Reserve and create still need a value until the human path is changed, so pass a fixed constant (or remove the scope from reservation and create at the same time). Update internal/web/identity_native_test.go:179 and terminal_test.go:81 accordingly.

### RUN-7: The checkpoint driver loads the plan and receipts and derives the integration head in three places with three different parsers, re-proves capability bindings, and checks a request against itself

Run end: task events, checkpoints, results and settlement (RUN). duplicated-layer. confirmed. ~110 lines, risk low. Visible change: none. Behaviour-preserving: yes. **Status:** open (behaviour-preserving).

Where: `internal/factory/control/task_checkpoint.go:62-147`, `internal/factory/control/task_checkpoint.go:233-315`, `internal/web/api/factory_mcp_checkpoints.go:41-67`, `internal/factory/control/mcp_checkpoint_finish.go:26-42`, `internal/factory/control/publication_plan_gate.go:261-286`.

**Simpler.** Write one loader, runExecution(ctx, run) → {manifest, plan, receipts, head}, used by the importer, the checkpoint driver, context and publication. Drop the capability re-binding loops and decode each submission once. In checkpointTerminalReady, compare finished.Result.Candidate with run.FinalCheckpoint.Candidate and check the required integration checks. With RUN-1 most of this file goes away anyway.

**Risk.** Pure restructuring. The 'busy' and lost-reply recovery of the reserved request must stay.

**Safe first step.** Add one helper in internal/factory/control (e.g. integrationHead(plan, receipts) built on parseTaskEventRef) and replace the two inline head loops in task_checkpoint.go (93-103, 294-301) with it. Export or duplicate-free reuse it in internal/web/api/factory_mcp_checkpoints.go:49-67 in place of the CutPrefix/ParseInt loop. Then delete the capability re-binding conditions at task_checkpoint.go:116-120 and 278-283, keeping only the GetMCPCapability call where its State is used for the active check.

### HFW-1: Model must copy host-observed output_digest/native_child_ref back in submit_task_output; Go/host only compare to what they stored; stale receipt_digest guidance

Rust host: factory worker runtime. unneeded-validation-or-digest. partially-confirmed. ~100 lines, risk medium. Visible change: model tools. Behaviour-preserving: no. **Status:** owner decision; coordinate with prod-hardening.

Where: `lib/host/src/factory/checkpoint/mod.rs:39-56`, `lib/host/src/factory/checkpoint/mod.rs:254-309`, `lib/host/src/factory/checkpoint/mod.rs:686-697`, `lib/host/src/factory/native_tasks.rs:601-665`, `lib/host/src/factory/native_actions.rs:119-160`, `lib/host/src/terminal/factory/muse/commands.rs:23`, and 5 more.

**Simpler.** Have submit_task_output take {task_id, evidence_refs}. Go reads task.OutputDigest and task.Child from its own store instead of matching copied values, and the host already checks against its stored OutputObservation. release_output stops telling the model to forward the digest. Fix or remove the "receipt_digest" finish guidance, since finish never takes it, and drop receipt_digest from the checkpoint reply. Keep the internal output digest as an identifier, or replace it with the tree, because it is used in commit messages, task events and verification receipts. For Muse start_task, optionally split NativeTasks::tool into a Codex params wrapper and an inner admit(tool, task_id, call) that native_actions::poll calls directly. This drops the synthetic threadId/turnId object without moving NativeTasks behind a mutex.

**Risk.** Changes the worker MCP schema, the tool descriptions and the Muse developer prompt. Go proposal matching has to switch to lookup by task id. NativeTasks moves from drive-thread ownership to a shared mutex.

### HFW-11: Hash-chained task journal (prev_digest and hand-written canonical encoding) guards a single-writer private file

Rust host: factory worker runtime. duplicated-layer. partially-confirmed. ~100 lines, risk low. Visible change: none. Behaviour-preserving: yes. **Status:** open (behaviour-preserving).

Where: `lib/host/src/factory/events.rs:85-206,375-400`, `lib/host/src/factory/task_events.rs:21-44`, `lib/host/src/factory/tasks.rs:125-140,260-270`, `lib/host/src/factory/checkpoint/mod.rs:152-178,312-351,585-590`, `internal/project/factory_events.go:56-112`, `internal/factory/control/task_execute.go:80-94,270-309,585-600`.

**Simpler.** Drop prev_digest, digest() and the chain checks in Rust append, the Go TaskEventDigest/lastDigest comparison and the digest suffix in taskEventRef. Keep the contiguous sequence check, which both sides already enforce. Exact replay detection can compare the whole event. Optionally put commit and tree in typed fields instead of parsing "commit=" from detail. Keep host checkpoint operations and Go receipts as they are, since they serve different purposes.

**Risk.** The Go fold has to drop its digest check in the same change.

**Safe first step.** Remove the prev_digest field in one change across both sides. Rust: remove prev_digest from TaskEvent (events.rs:85-95), from the deserializer table (events.rs:125) and from encode() (events.rs:144-145); delete the prev_digest checks in validate() (events.rs:178-183) and append() (events.rs:392-398); delete the prev_digest construction in checkpoint/mod.rs:337-341 and tasks.rs:131-139; delete digest() if nothing else uses it. Go: remove PrevDigest and validateChain's digest rules from factory_events.go:64,79,87,105-110; delete the PrevDigest != lastDigest check and lastDigest tracking in task_execute.go:292-309. Leave taskEventRef's digest suffix computed from the event (or switch it to type@seq) in a follow-up. Update or delete the chain-building tests in internal/factory/control/planning_admission_test.go.

### IDN-2: Factory sponsorship forces the owner to self-delegate a broker grant and re-pick it, though the broker skips grants for the owner

Identity service and provider credentials. speculative-abstraction. partially-confirmed. ~100 lines, risk low. Visible change: UI. Behaviour-preserving: no. **Status:** owner decision.

Where: `internal/web/api/factory_sponsorship.go:47-58`, `internal/web/api/factory_sponsorship.go:109-142`, `internal/factory/sponsorship.go:16-24`, `internal/factory/sponsorship.go:45`, `frontend/spaces/sodaspaces-factory-settings.ts:551-565`, `frontend/spaces/sodaspaces-factory-settings.ts:899-929`, and 2 more.

**Simpler.** Keep broker grants, which run documented named-member delegation (terminal use, the Available list, revocation). Make sponsorship stop going through them: the actor is the saving connection owner (already enforced by checkSponsorshipBroker), and the project comes from the repository's Project. Remove Sponsorship.GrantID and its check in Validate, the Identity.Grants lookup in checkSponsorshipBroker, the grants fetch in connectionFacts, and the 'Project access' grant select. Sponsoring as a delegated non-owner member goes away, which is a small product decision.

**Risk.** Product decision: removes 'let a named teammate use my subscription'. Needs a schema change (drop identity_grants, or leave it unused) and a sponsorship API/form change; coordinate with whoever simplifies Factory settings (evidence 4).

### MCP-10: Worker JEV consultations use spend reservations that block retries

Model-facing MCP tools. too-many-states-or-budgets. confirmed. ~100 lines, risk low. Visible change: UI, model tools. Behaviour-preserving: no. **Status:** owner decision.

Where: `internal/web/api/factory_mcp_systemone.go:104-287`, `internal/store/factory_intents.go:267-323`, `internal/factory/judgment.go:96-130`, `internal/web/api/factory_mcp_context_gate.go:10-36`, `internal/web/api/factory_mcp.go:915`.

**Simpler.** Call JEV, record request and result in the work post, and return the answer. A failed call can simply be retried. Drop ReservePriorityJudgment, the replay through PriorityJudgment, RecordPriorityJudgment and the requireCurrentWork gate for consultations. Keep the snapshot hash only to give the Feed post its id (adding a nonce or timestamp so retries get new posts). Accept any consultation_group string (hash it) or drop it. Optionally use one systemone.judge tool with a kind field.

**Risk.** Feed consultation posts still record each call; duplicates on retry are acceptable.

### PRJ-7: Lifecycle and key operations over-fence: identity fence around systemctl, a unit check that rejects any drop-in, a duplicate key preview, and redundant container inspections

Project environment runtime on the host (create/join/start, executor, domain, ssh, accounts). unneeded-validation-or-digest. partially-confirmed. ~100 lines, risk low. Visible change: none. Behaviour-preserving: yes. **Status:** open (behaviour-preserving).

Where: `lib/host/src/project/mod.rs:155-298`, `lib/host/src/project/inspect.rs:347-462`, `lib/host/src/account/member.rs:11-62`, `lib/host/src/account/mod.rs:330-386`, `cmd/soda-project-terminal/src/keys.rs:120-135`, `system/host/services/soda-project@.service`.

**Simpler.** Lifecycle: one project_container call, then `systemctl enable|disable --now`, then one inspect and one `systemctl is-enabled` (or keep the single show, reading only UnitFileState). Drop the identity fence, the FragmentPath/DropInPaths/LoadState validation and the start-time member-access precheck, which ExecStartPre already enforces. access_keys sends the apply request once and lets the agent's locked revision check refuse. In member.rs, reuse one inspection instead of project_container followed immediately by inspect. Merging the two inspectors into one format/decoder is optional follow-up work. project_container's isolation checks (userns, ID maps, privileged) must stay because about 27 call sites depend on them.

**Risk.** There is no concurrent writer. The unit check only guards against a root admin editing systemd, which is not an adversary here.

**Safe first step.** Three changes: 1. In lib/host/src/project/mod.rs, inside Runtime::lifecycle, delete the second project_container call and the 'project identity changed during operation' comparison. The unit only does `podman start`/`podman stop`, so the container ID cannot change and the outcome is identical. 2. In lib/host/src/account/mod.rs, inside access_keys, delete the `if input.apply { ... preview ... }` block. The agent in cmd/soda-project-terminal/src/keys.rs already refuses a revision mismatch under its flock. Update the expected error string in lib/host/tests/project_operations/access_keys.rs:70 to match. 3. Then reduce validate_unit_properties to reading UnitFileState only, dropping the FragmentPath and DropInPaths checks.

### BUD-11: Concurrency knobs that never take effect (queued runs; policy, operator and sponsorship concurrency), plus a redundant repo-limit min/clamp

Budgets, allowances and limits (cross-cutting). unused-configuration. partially-confirmed. ~90 lines, risk low. Visible change: UI, model tools. Behaviour-preserving: yes. **Status:** owner decision; coordinate with prod-hardening.

Where: `internal/factory/grants.go:155-162`, `internal/factory/grants.go:222`, `internal/factory/grants.go:246-248`, `internal/factory/grants.go:310-327`, `internal/factory/grants.go:358-376`, `internal/factory/sponsorship.go:26`, and 23 more.

**Simpler.** Delete MaxQueued, plus the policy, operator and sponsorship MaxConcurrent fields: their validation, API fields, Settings inputs and summary text, the prompt concurrency sentence and its PromptInputs checks, the control and store sponsorship-slot checks, and the repoLimit min/clamp (use a constant 1). Update the installed qualification scripts that assert these fields. Keep Capacity.MaxConcurrentRuns and the existing per-repository guards (unsettled runs, repoSessions, repoAttempts), because each guards a different state.

**Risk.** No observable limit changes; only inputs that never took effect go.

**Safe first step.** Backend-only, with identical observable behaviour. In internal/factory/control/dispatch_selection.go, replace the repoLimit min/clamp (lines 83-89) with repoLimit := 1, and delete the unreachable sponsorship-slot check (lines 105-109; planned stays 0 only if heldConn's other use is kept, so keep the planned sum). In internal/store/factory_dispatch_packet.go, delete the unreachable sponsorship slots count (lines 1197-1203), since repoSessions > 1 has already rejected the case. Delete the MaxQueued field and its validation (grants.go:316, 323) together with the API field (factory_policy.go:97, 114). This last step does change Settings, so treat it as part of the user-visible follow-up.

### PTH-5: Dead Muse pin path: configured muse_sha256/muse_version never compared; digest only acts as the Muse on/off switch

In-project helpers (soda-project-terminal) and the host Muse launch runtime. unused-configuration. partially-confirmed. ~90 lines, risk low. Visible change: none. Behaviour-preserving: yes. **Status:** partly done (`d0435632`): expected_muse_identity and verify_guest_binary deleted; the unread pin fields and config are open.

Where: `lib/host/src/muse/execution.rs:13-32`, `lib/host/src/muse/provider.rs:5-6,78,178-217`, `lib/host/src/muse/mod.rs:66-69`, `lib/host/src/muse/runtime_types.rs:101-112`, `lib/host/src/main.rs:344-351`, `lib/host/src/main.rs:405-411`, and 4 more.

**Simpler.** 1. Delete verify_guest_binary, expected_muse_identity (and its re-export), the binary_version/binary_sha256 fields and the matching MuseRuntime::new arguments. This is purely dead code. 2. Then enable the backend's Muse runtime when muse_socket is set, and drop muse_sha256/muse_version from the lib/host config, its validation and MuseConfig. Keep the socket-path checks. Update credentials.md to match. 3. soda-project-maintain's own muse_sha256 (used by load_tools) is separate and live; leave it alone.

**Risk.** soda-project-maintain reads the same config key for tool loading. Coordinate before removing the key from the shared config file, or keep it only there.

**Safe first step.** 1. In lib/host/src/muse/execution.rs, delete verify_guest_binary (lines 13-32 including its doc comment). 2. In lib/host/src/muse/provider.rs, delete expected_muse_identity (lines 178-217) and the stale mention in the comment at line 78. 3. In lib/host/src/muse/mod.rs:67, remove `expected_muse_identity` from the re-export. 4. In lib/host/src/muse/runtime_types.rs, remove the binary_version/binary_sha256 fields and the matching MuseRuntime::new parameters. 5. Update the three new() callers (main.rs:406, dbackend/mod.rs:222, and lib/host/tests/muse_serve_oracle.rs:142 and 1072) to stop passing those arguments. Leave the muse_sha256-based enable gate (main.rs:344) and the config validation alone in this first step. Changing them alters behaviour for configs that set the socket without a digest.

### PRP-3: StartPreparation re-runs authorization and the admission transaction, and has extra recovery and race-adoption paths

Project preparation pipeline below the API (standing and candidate preparations, decisions, hold, host and root helper). unneeded-validation-or-digest. partially-confirmed. ~80 lines, risk low. Visible change: UI. Behaviour-preserving: yes. **Status:** owner decision.

Where: `internal/factory/control/preparation_start.go:24-133`, `internal/store/preparation.go:180-206`, `internal/web/api/preparation_source.go:244-266`.

**Simpler.** Do the authorize callback once, since the API's admitPreparationStart already did the same checks. Admit the row in one transaction. Keep the native pre-inspect so a retry does not restart a running worker, then call host.Prepare and adopt its reply once. Drop the second authorize, Store.AuthorizeStandingPreparation and the detached recovery inspect. Make adoptStandingPreparation return the stored row on a CAS conflict instead of DeepEqual-matching it (sameStandingPreparationObservation goes). The progress-loop observer stays: it is the only path that records running->ready/failed.

**Risk.** The visible result is the same: the row shows running, then ready or failed. The only change is that the button returns sooner.

**Safe first step.** In internal/factory/control/preparation_start.go delete the second `authorize(bounded)` call and the `c.Store.AuthorizeStandingPreparation` call (lines 66-73). Then delete Store.AuthorizeStandingPreparation in internal/store/preparation.go (lines 180-206) and its tests. Since AdmitStandingPreparation just ran the identical authorizeStandingPreparationTx check, observable behaviour only differs if the heads or the hold change during the ~5s native inspect.

### STO-4: Publication and merge registration repeat an authority-revision fence pasted into two UPDATEs, and publications pin authority by revision with no way to re-authorize

Store (PostgreSQL layer) structure. unneeded-validation-or-digest. partially-confirmed. ~70 lines, risk medium. Visible change: UI. Behaviour-preserving: yes. **Status:** owner decision; coordinate with prod-hardening.

Where: `internal/store/factory_publications.go:155-174`, `internal/store/factory_merges.go:124-151`, `internal/factory/control/publication_authority.go:12-55`, `internal/factory/control/merge_evidence.go:128-170`, `internal/store/factory_dispatch_packet.go:1094-1124`, `internal/factory/grants.go:382-392`, and 1 more.

**Simpler.** First, behaviour-preserving: move the shared WHERE fence (gate, acceptance and authority arms, plus their argument list) into one store helper used by both UPDATEs, and fold publicationAuthority and mergeAuthority into one coordinator helper. The merge-only extra comparisons (BaseRef, ReviewerID) and the report type become parameters. Then, as a separate decision that changes behaviour: authorize on effective state (EffectiveAuthority.Effective, dispatch gate, acceptance head not withdrawn) and on the fields that decide who acts (target branch, reviewer, sponsorship connection and actor) rather than on equality of every revision, and drop the revision arms from the fence.

**Risk.** This deliberately relaxes the 'authority pinned at admission' rule. A grant that is actually revoked or paused mid-run is still caught by EffectiveAuthority and the dispatch gate. Removing only the SQL half is behaviour-preserving apart from a race window.

**Safe first step.** Extract the duplicated fence SQL and its 20 trailing arguments from UpdatePublication (internal/store/factory_publications.go:155-174) and applyMergeUpdateTx (internal/store/factory_merges.go:131-151) into one shared helper in internal/store that builds the authority/gate predicate and its args for (repository, issue, acceptance, assignmentID, projectID, AuthorityRef). Merge publicationAuthority (internal/factory/control/publication_authority.go) and mergeAuthority (internal/factory/control/merge_evidence.go:130-170) into one function that returns the reason code, with callers keeping their own report/wait wrappers and the merge-only BaseRef/ReviewerID check. Semantics stay identical.

### VRF-7: Legacy '# soda-app-verification:' check.sh fallback, used when Product Lead omits the optional manifest proposal, is practically unsatisfiable

Test Pilot / app verification (Go factory+control+store+MCP, Rust host, in-project helper). dead-code. partially-confirmed. ~60 lines, risk low. Visible change: UI, model tools. Behaviour-preserving: no. **Status:** owner decision.

Where: `internal/factory/verification_approved_check.go:25-56`, `internal/factory/control/verification_execution.go:17-31`, `internal/factory/context_verification.go:21,125,138-143`, `internal/factory/control/app_verification.go:99-104,122-127`, `internal/factory/control/delivery_repair.go:243`, `internal/factory/control/delivery_admission.go:89`, and 2 more.

**Simpler.** Make the adopted manifest the only path. Delete ParseApprovedAppCheck and ApprovedAppCheckPrefix. A nil VerificationManifest becomes a plain 'no adopted verification definitions' error in adoptedDeliveryCheck, and the else branch in CurrentVerificationEvidence.ValidateFor goes. ExecutionRequired collapses to always-true, so remove the field and its branches (keep Validate, Digest and BindsIntent, which the store and delivery admission use). Reduce the outcome strings to two. Rewrite delivery_admission.go:89, the verification_evidence next_step and the D16 doc paragraph to name the real missing input: an adopted verification_manifest_proposal plus a ready reviewer preparation with a passing check.sh. Optionally make the proposal required for application plans so the dead end shows up at plan time.

**Risk.** Only an administrator hand-editing check.sh with revision IDs could use this path today.

### PLN-8: JudgeAssumptions JEV call is redundant once material assumptions always become challenges: every assumption is material

Intent to plan (Product Lead, Architect, research, decomposition). other. partially-confirmed. ~55 lines, risk low. Visible change: UI, model tools. Behaviour-preserving: yes. **Status:** done (`653ef0e9`): every pending assumption becomes a challenge without a JEV call; the assumptions purpose is gone.

Where: `internal/factory/control/assumption_advice.go:10-84`, `internal/factory/intent_proposal.go:190-193`, `internal/web/api/factory_intent_explore.go:205-210,238-271`, `internal/factory/judgment.go:192,229`.

**Simpler.** Make JudgeAssumptions deterministic: a pending assumption (always material) becomes an AssumptionChallenge, and non-assumption ideas still go through settleChoice. Delete assumptionRubric, assumptionOptions, assumptionEvidence, the AdvisePurpose/ReserveIntentJudgment call, the dead non-material settle branch and the JudgmentAssumptions purpose. This also fixes main's stranded-assumption bug, which the hardening branch fixes another way.

**Risk.** Identical results to the hardening branch, without the call.

**Safe first step.** In internal/factory/control/assumption_advice.go, delete the unreachable `if !idea.Material { review.Settled = append(...) }` block inside the non-challenge branch (lines ~72-74). No producer creates a non-material assumption, so behaviour is identical. Removing the JEV call itself changes behaviour on main and changes allowance accounting even on prod-hardening.

### STO-6: A global factory_judgment_budget counter is incremented by six transactions, never read, and serialises every JEV reservation

Store (PostgreSQL layer) structure. too-many-states-or-budgets. confirmed. ~55 lines, risk low. Visible change: DB schema. Behaviour-preserving: yes. **Status:** done (`c62f6ec1`).

Where: `internal/store/schema.go:35-36`, `cmd/soda-identity/src/schema.rs:30-31`, `internal/store/factory_context_selection.go:24-27`, `internal/store/factory_context_selection.go:56-58`, `internal/store/factory_attempt_allowances.go:1232-1245`, `internal/store/factory_delivery_repair_adoption.go:70-86`, and 10 more.

**Simpler.** Delete the table and its seed row from both internal/store/schema.go and cmd/soda-identity/src/schema.rs. Delete the 12 SELECT ... FOR UPDATE and UPDATE statements and the spent variables in the six functions. Replace each count-then-insert on factory_priority_judgments with INSERT ... ON CONFLICT(snapshot) DO NOTHING and check RowsAffected, returning false or ErrCommandConflict exactly as today. Drop the spent assertions in the tests, or switch them to counting factory_priority_judgments rows. Keep the per-purpose gates (planning allowance, delivery calls, repair charges) as they are.

**Risk.** The counter gates nothing. Removing the global lock only stops unrelated reservations from serialising on one row.

**Safe first step.** Two steps. Step 1: in internal/store/factory_context_selection.go, factory_attempt_allowances.go, factory_delivery_repair_adoption.go, factory_reassessments.go and factory_intents.go (both functions), delete the `var spent` declaration, the `SELECT spent FROM factory_judgment_budget WHERE id=1 FOR UPDATE` scan and the `UPDATE factory_judgment_budget SET spent=...` exec. In the same change, turn each count/EXISTS-then-INSERT on factory_priority_judgments into INSERT ... ON CONFLICT(snapshot) DO NOTHING with a RowsAffected check that returns the same false/ErrCommandConflict as today. Step 2: drop the CREATE TABLE and seed INSERT from internal/store/schema.go:35-36 and cmd/soda-identity/src/schema.rs:30-31, and remove the spent reads and the spent=1000 setup from the listed tests.

### GOM-10: Live Workspace page and side panel remain against the no-Workspace decision; the soda contribution allow-list is copied three times

Other Go packages (forgejo, acceptance, project, host, tailnet, application, config, strictjson, filelock, avatar, cmd/soda-dashboard, cmd/soda-tailnet, cmd/soda-extension). dead-code. partially-confirmed. ~25 lines, risk low. Visible change: UI. Behaviour-preserving: yes. **Status:** deferred to repo-first-ui (Workspace removal).

Where: `cmd/soda-extension/main.go:58-71`, `internal/web/auth/extension.go:44-56`, `internal/web/api/extension_native.go:135-150`, `system/containers/extension/extension.json:7-36`, `frontend/spaces/soda-extension.ts:23-27,55-62`, `frontend/spaces/soda-spaces-entry.ts:14`, and 2 more.

**Simpler.** Remove the Workspace panel from extension.json, from the three allow-rules, and from the panel paths in soda-extension.ts and soda-spaces-entry.ts, along with its tests. Move the global 'Workspace' (spaces) page features to the repository page as part of the no-Workspace work. Separately, collapse the third allow-list copy now: in extension_native.go, make the non-tailnet branch of extensionProductContribution call auth.ExtensionContribution("soda", contribution), and delete extensionPageContribution and extensionWorkspacePanelContribution.

**Risk.** Must be coordinated with the frontend panel code (persistent-panel entry). Removing the panel removes UI.

**Safe first step.** In internal/web/api/extension_native.go, replace the non-tailnet return in extensionProductContribution with `return auth.ExtensionContribution("soda", contribution)`, and delete extensionPageContribution and extensionWorkspacePanelContribution. Behaviour stays identical because the caller has already checked auth.ExtensionContribution and the soda rule set is the same. Removing the panel itself changes UI and is a separate step.

### HPL-3: One process-wide try-lock mutation gate refuses unrelated projects' native mutations with 503 'service busy'

Rust host platform: daemon, terminals, project, application, tailnet (lib/host non-factory, soda-project-maintain, forgejo/pg/console helpers, soda-tailnet, unix-http, wire-time). too-many-states-or-budgets. partially-confirmed. ~15 lines, risk medium. Visible change: UI. Behaviour-preserving: no. **Status:** owner decision.

Where: `lib/host/src/daemon/admission.rs:55-69`, `lib/host/src/daemon/admission.rs:213-255`, `lib/host/src/daemon/routes.rs:176-191`, `lib/host/src/dbackend/backend.rs:112-300`, `lib/host/src/project/admission.rs:23-90`, `internal/host/client.go:86-98`.

**Simpler.** Keep serialization but stop refusing. In acquire_native_writer, replace gate.try_acquire() with the existing blocking gate.acquire() and delete the 503 'service busy' branch and AdmissionGate::try_acquire. If cross-project parallelism matters, key the lock by project id and keep one global lock for /create, /account and /prepare. Do NOT rely on project::admission: it is a shared (LOCK_SH) lock on one directory that only excludes maintenance, and several gated routes never take it.

**Risk.** Check that no two operations on different projects share host state (for example the podman network or the subnet allocator) that relied on the global gate. A narrow lock can protect that instead.

### RTY-3: Issue Retry capability ignores pending implementation retry; delivery Retry availability uses a separate locking + live-Forgejo check

Retry kinds across store, control and UI. duplicated-layer. partially-confirmed. ~10 lines, risk low. Visible change: UI. Behaviour-preserving: no. **Status:** owner decision.

Where: `internal/web/api/factory_issue_view.go:566-575,755-795`, `internal/web/api/factory_views.go:161,205-226`, `internal/store/factory_delivery_retry.go:102-128`, `internal/factory/control/lifecycle_retry.go:37-43,97-141`, `internal/store/factory_implementation_retry.go:122-160`, `internal/web/api/factory_lifecycle.go:29-30`.

**Simpler.** In factoryIssueCapabilities, set RetryQueued (and requested_retry) when either PendingExplicitRetry or PendingImplementationRetry finds a command. Once RTY-1 removes the implementation kind, this becomes a single selector. Have RecordImplementationRetryCommand refuse when a finished implementation-retry command already targets the same run. Optionally drop the generic per-view lock transaction for delivery by reading eligibility without FOR UPDATE, but keep the authoritative check in the POST. Do not build a parallel lock-free eligibility re-implementation.

**Risk.** Only the view computation changes. The POST still decides.

## Ranked findings: low impact

### FED-11: Delete the speculative Feed routing/acceptance and Ralph migration sections; fix the public Feed page

The Feed. process-machinery. partially-confirmed. ~700 lines, risk low. Visible change: none. Behaviour-preserving: yes. **Status:** open (behaviour-preserving).

Where: `docs/design/factory-feed-components.md:807-1149`, `docs/design/factory-feed-and-ralph-integration.md:400-645`, `docs/public/20-Factory/20-factory-feed.md:69-75`.

**Simpler.** Delete components-doc sections 17-20 (routing, precedence, per-role exceptions, acceptance contract) and the Ralph doc's frontend-stack, React-migration, MCP-proposal, state-machine and absorb sections. Keep the per-role catalog sections while feed_components.json still has 121 variants in 19 presentations; shrink them only if the catalog itself is collapsed. In the public page, remove the "Show: All activity / Worker activity / Comments" instruction (the saved kind is never applied), or fix the code. Update the anchor links in remaining-tasks and the READMEs.

**Risk.** Docs only. docs/README.md, factory-worker-mcp.md and the remaining-tasks docs link to anchors in factory-feed-components.md and would need updating.

**Safe first step.** Delete docs/design/factory-feed-components.md sections 17-20 (lines 807-1149) and docs/design/factory-feed-and-ralph-integration.md lines 400-645 (frontend stack, React migration assessment, MCP communication proposal, state machine, absorb table). Then fix anchor links to #failure-component-routing and similar in docs/development/factory-remaining-tasks.md, factory-remaining-tasks/04-*.md and 05-*.md, factory-worker-mcp.md and docs/README.md. Docs only; no runtime effect.

### RBA-10: soda-forgejo-domain is a 600-line Rust wrapper around a real Fountain offline-recovery procedure, replaceable by documented systemctl and marker commands

Release build primitives, asset fetch/render tools, small install binaries and system units. speculative-abstraction. partially-confirmed. ~650 lines, risk low. Visible change: none. Behaviour-preserving: yes. **Status:** open (behaviour-preserving).

Where: `cmd/soda-forgejo-domain/src/main.rs:1-58`, `cmd/soda-forgejo-domain/src/domain.rs:1-191`, `cmd/soda-forgejo-domain/src/marker.rs:1-189`, `cmd/soda-forgejo-domain/src/cli.rs:1-102`, `cmd/soda-forgejo-domain/src/system.rs:1-65`, `cmd/soda-forgejo-domain/README.md`, and 7 more.

**Simpler.** Delete the binary, its README, its Go CLI test, its Cargo workspace entry and its RUST_TOOLS entry. Then rewrite the recovery steps in operator-setup.md and 80-recovery.md as plain commands: `systemctl stop forgejo` (and check that `podman ps` shows no soda-forgejo container); `systemctl mask --runtime forgejo`; `touch /var/lib/soda/forgejo/gitea/nativeop-offline`; then `admin native-operation recover ...`; then `rm` the marker, `systemctl unmask --runtime forgejo` and `systemctl start forgejo`. Keep the procedure itself, because Fountain's models/nativeoperation honours the marker.

**Risk.** Check whether Fountain actually honours the marker before also dropping it from the docs.

**Safe first step.** In cmd/soda-forgejo-domain/src/system.rs, remove the `Sys` trait. Its only impl is RealSys, and it is described as "injectable for tests", but the crate has no tests. Make run/elapsed/sleep plain functions and update the callers in domain.rs and main.rs. This keeps the observable behaviour identical. Deleting the whole binary and rewriting the docs is the larger follow-up.

### DOC-10: trust.md spends about 700 lines on the Fountain conditional-mutation design; the mechanism is largely built, but the prose is out of date and much longer than needed

Documentation. speculative-abstraction. partially-confirmed. ~550 lines, risk low. Visible change: none. Behaviour-preserving: yes. **Status:** open (behaviour-preserving).

Where: `docs/architecture/trust.md:287-995`, `docs/architecture/trust.md:695-781`.

**Simpler.** Keep a short section of about 80-120 lines that describes what exists. (1) Fountain exposes an atomic native revision with an idle/busy reservation. (2) Soda brackets snapshot reads with it (ErrNativeBusy/ErrStaleSnapshot leading to PublicationWait). (3) Native operations carry actor, repository, authority revision, expiry and expected refs, and are reconciled through lookup or receipt. (4) Native permission, branch protection and fast-forward-only merge are preserved. Remove the "target capability, not an existing lock" wording, the reconciliation chronology, the comparison of alternatives, and the speculative quiescence and recovery-authority narrative. Link to internal/forgejo/publish/README.md for operation details.

**Risk.** Keep the real security boundaries: credential custody, UID isolation and worker capability scoping. Only the speculative coordination spec goes.

**Safe first step.** In docs/architecture/trust.md, rewrite lines 308-995 ("Conditional native mutations" through "Initial merge methods") as a short description of the mechanism as built: native revision plus idle reservation, bracketed reads in internal/forgejo/snapshot.go, persisted operations with authority revision and expected refs per internal/forgejo/publish/README.md, and fast-forward-only merge. Delete the "target capability, not an existing lock" text and the 2026-10-11 chronology. Keep lines 1-286 (identity split, UIDs, Docker, factory and project authority boundaries) unchanged. Docs only; no code changes.

### REL-12: Upgrade-qualification receipt machinery (qualify-upgrade, native evidence stamping, promoted "upgrade" summary key) for upgrades nobody performs

Release, install, packaging, CI and freshness machinery. speculative-abstraction. partially-confirmed. ~550 lines, risk low. Visible change: none. Behaviour-preserving: no. **Status:** open (behaviour-preserving).

Where: `lib/soda-release-deliver/src/upgrade.rs:1-391,442-469`, `lib/soda-release-deliver/src/publish/mod.rs:518-555`, `lib/soda-release-deliver/src/publish/mod.rs:659-664`, `lib/soda-release-deliver/src/publish/mod.rs:691-720`, `lib/soda-release-deliver/src/publish/mod.rs:807-819`, `lib/soda-release-deliver/src/publish/mod.rs:840-856`, and 7 more.

**Simpler.** Delete the qualify-upgrade command, stamp/collect/verify_collected/NativeEvidence/Qualification/PublishedQualification/published_from_notes, the "upgrade" key in the promotion summary and its re-checks, and maintain's receipt-required ordinary branch, along with their tests and docs. Keep the soda-qualification-summary-v1 block for the appliance and docker receipts. Keep admit_candidate_directory, and keep UpgradeFrom while the development S->T fixture maintenance and Docker provenance flow still uses it. When the first real upgrade happens, test it and write down the result.

**Risk.** cmd/soda-project-maintain/src/release_target.rs (outside my scope) reads the promoted summary for upgrade admission and would need the same removal.

### TST-9: Schema 68-to-69 migration, concurrent-open upgrade race and in-place CHECK-constraint 'scope adoption' (delivery, repair, delivery-repair) kept and tested for a product with zero users

Test harnesses and source guards. speculative-abstraction. partially-confirmed. ~550 lines, risk low. Visible change: none. Behaviour-preserving: yes. **Status:** open (behaviour-preserving).

Where: `internal/store/application_schema_test.go:104-338`, `internal/store/schema_test.go:25-83`, `internal/store/schema.go:280-311,464-536`, `internal/store/factory_delivery_repair_schema.go:1-103`, `internal/store/factory_repair_schema.go:1-61`, `internal/store/factory_delivery_schema.go:1-61`, and 1 more.

**Simpler.** Keep one current schema created fresh. On any other version, refuse with the existing 'database schema differs' error (TestOpenRejectsOldSchemaWithoutMutation already covers refusal). Delete schemaUpgrade68To69, migrateSchema68To69, the version==68 branch, reopenAfterLostUpgradeRace and schemaOpenMu. Delete the constraint-adoption functions initializeDeliveryRepairSchema, initializeRepairSchema and initializeDeliverySchema; the current CHECKs already live in schemaStatements. Delete the four migration tests and the scope-adoption block in TestOpenCreatesOnlyTheCurrentSchema.

**Risk.** Check whether the current demo VM database is still at version 68. If it is, recreate it once.

**Safe first step.** 1. Delete TestOpenMigratesRetained68To69, TestConcurrentOpensMigrateRetained68Once, TestLostUpgradeRaceRereadAcceptsCommitted69 and TestFailedMigrationLeaves68Intact (application_schema_test.go:104-338), and the scope-adoption block (schema_test.go:25-83). These are test-only, so runtime behaviour is unchanged. 2. In schema.go, delete schemaUpgrade68To69, migrateSchema68To69, the `version == 68` branch in initializeSchema, reopenAfterLostUpgradeRace and schemaOpenMu. This is identical for every database at version 69; a 68 database falls through to the existing refusal. Check the demo VM database version first and recreate it if it is still 68.

### RBA-8: Terminal-logo SVG geometry renderer regenerates two committed text files

Release build primitives, asset fetch/render tools, small install binaries and system units. process-machinery. confirmed. ~475 lines, risk low. Visible change: none. Behaviour-preserving: yes. **Status:** open (behaviour-preserving).

Where: `tools/release-assets/src/render/terminal_logo/mod.rs:1-70`, `tools/release-assets/src/render/terminal_logo/geometry.rs:1-178`, `tools/release-assets/src/render/terminal_logo/svg.rs:1-96`, `tools/release-assets/src/bin/soda-render-terminal-logo.rs:1-83`, `tools/release-assets/tests/render_terminal_logo.rs:1-109`, `tools/release-assets/tests/render_support/mod.rs:84-86`, and 7 more.

**Simpler.** Delete the terminal_logo module, the soda-render-terminal-logo bin and its [[bin]] entry, render_terminal_logo.rs and logo_bin(), the cargo --check call in TestTerminalBrandingMatchesCanonicalSymbol, and the README regeneration lines. Keep the committed sodaos.txt and motd.txt and hand-edit them on the rare emblem change. Keep the MOTD escape-code check and the fastfetch geometry test as plain file checks.

**Risk.** The outputs are committed, so nothing at runtime changes.

**Safe first step.** Delete tools/release-assets/src/render/terminal_logo/ (mod.rs, geometry.rs, svg.rs), tools/release-assets/src/bin/soda-render-terminal-logo.rs, the soda-render-terminal-logo [[bin]] block in tools/release-assets/Cargo.toml, tools/release-assets/tests/render_terminal_logo.rs and logo_bin() in tests/render_support/mod.rs. Remove `pub mod terminal_logo;` and the doc mention in src/render/mod.rs. Remove lines 12-15 (the cargo --check exec) from scripts/terminal_branding_test.go and keep the MOTD escape check. Remove the regeneration lines in assets/branding/terminal/README.md and tools/release-assets/README.md. Leave the committed assets/branding/terminal/*.txt untouched.

### DOC-7: docs/guides repeats the docs/public handbook for factory enrollment, Docker install and release-media install

Documentation. duplicated-layer. partially-confirmed. ~400 lines, risk low. Visible change: none. Behaviour-preserving: yes. **Status:** open (behaviour-preserving).

Where: `docs/guides/operator-setup.md:148-345`, `docs/public/40-Configure/20-factory-authorization.md:102-298`, `docs/guides/docker.md:12-98`, `docs/public/60-Install/50-docker.md:1-71`, `docs/guides/installation.md:47-73`, `docs/public/60-Install/40-install-on-premises.md:1-67`, and 5 more.

**Simpler.** Make docs/public the one owner of operator procedures. Delete the enrollment block from guides/operator-setup.md and point to public 20-factory-authorization.md#enroll-native-factory-operations. Keep operator-setup's release-maintenance and offline-recovery sections, or move them into public 50-Operate. Cut guides/docker.md down to the image build and maintainer-only qualification notes, and point to public 50-docker.md. Remove the release-media paragraphs from guides/installation.md. Delete guides/project-clis.md and guides/develop.md, after moving any unique lines into the public pages. Keep guides/project-services.md's Muse opt-in, or move it to public 60-project-services.md, and keep guides/media.md, which is developer material. Update docs/README.md, README.md, cmd/*/README.md links and the check-docker-packaging.sh message to the new owners.

**Risk.** Diff the guide and public versions before deleting, because the guides hold some developer-only flags. Keep the docker packaging script's pointer valid.

**Safe first step.** Delete docs/guides/operator-setup.md lines 148-345 ("Explicit factory enrollment" and subsections) and replace them with a link to docs/public/40-Configure/20-factory-authorization.md#enroll-native-factory-operations. Change cmd/soda-setup/README.md:80, which links #prepare-native-accounts-and-credentials, to the public anchor. Delete docs/guides/project-clis.md, which public 30-collaboration.md and 50-shared-tools-and-files.md already cover, and drop its rows from docs/README.md and README.md:93.

### RBA-7: release-assets keeps Python/Go-script CLI parity (Python exception names, argparse envelope, configparser emulation, exit 2/1 split) and duplicates sha256/flag helpers

Release build primitives, asset fetch/render tools, small install binaries and system units. other. partially-confirmed. ~400 lines, risk low. Visible change: none. Behaviour-preserving: yes. **Status:** open (behaviour-preserving).

Where: `tools/release-assets/src/render/provisioning/mod.rs:16-78`, `tools/release-assets/src/bin/soda-render-provisioning.rs:1-239`, `tools/release-assets/src/bin/soda-stage.rs:120-130`, `tools/release-assets/src/bin/soda-render-terminal-logo.rs:40-55`, `tools/release-assets/src/locales/merge.rs:1-30,110-130,240-290`, `tools/release-assets/src/locales/mod.rs:1-47`, and 9 more.

**Simpler.** Use one sha256_hex and one long-flag split_flag in a shared crate module. Replace the ProvKind Python class names with plain messages. Drop the argparse "ignored explicit argument" and usage-preface emulation and the exit 2/1 split in favour of `error: <msg>` with exit 1. Cut the configparser emulation down to a simple section/key check. Replace hand-written visitors with derived structs where rejecting duplicate keys is acceptable. Keep the Go-http-client User-Agent, because the Muse endpoint has only been observed with it.

**Risk.** Only CLI error texts and exit codes change, and the CLI tests that pin them would be deleted with that code.

**Safe first step.** Delete locales::split_flag and locales::sha256_hex (tools/release-assets/src/locales/mod.rs:15-47), plus the separate render::sha256_hex body, and point callers at one shared implementation. render::split_flag is byte-identical to locales::split_flag. render's sha256_hex can reuse fetch::sha256_hex/hex_lower, since the output is the same lowercase hex. Keep fetch::flag_token as it is, or make it a deliberate change, because it also accepts single-dash flags.

### TST-12: Push qualification gate: tag-and-poll push script, CI-result reuse via tags, two overlapping workflows, and a test that pins the workflow YAML

Test harnesses and source guards. process-machinery. confirmed. ~400 lines, risk medium. Visible change: none. Behaviour-preserving: yes. **Status:** open (behaviour-preserving).

Where: `scripts/push.sh:1-75`, `scripts/ci-reuse.sh:1-24`, `scripts/check-push.sh:1-40`, `tests/build/ci_admission_test.go:1-149`, `.github/workflows/ci.yml`, `.github/workflows/deep-ci.yml`, and 3 more.

**Simpler.** Use one workflow on push and PR that runs check-source.sh (plus the Rust and Docker packaging checks), and push to main normally (or through branch protection). Delete push.sh, ci-reuse.sh, check-push.sh, the reuse/admission jobs and ci_admission_test.go. Merge deep-ci.yml into ci.yml, drop the unused ci/soda-full-* tag trigger, remove the unitban allow-list entry and update the testing.md/release.md push instructions.

**Risk.** The owner may rely on push.sh day to day, so confirm the push workflow with them. Not product-facing.

**Safe first step.** Remove the CI-reuse layer. Delete scripts/ci-reuse.sh. In .github/workflows/ci.yml, delete the `reuse` job, the `needs: [reuse]`/`if: reused != 'true'` lines on go/browser/rust/packaging, and the REUSE_RESULT/REUSED_CHECKS handling in the `ci` admission step, leaving it to check each result for success. Delete tests/build/ci_admission_test.go and its line in tools/unitban/allow-go.txt. Product behaviour stays the same; the only difference is that a main push re-runs the push checks rather than reusing the tag run. Deleting push.sh and merging deep-ci.yml come after the owner confirms how they push.

### MCP-12: Host side: dead relay surface, dead prompt-entry mirrors, and the native-task protocol written five times

Model-facing MCP tools. dead-code. confirmed. ~360 lines, risk low. Visible change: model tools. Behaviour-preserving: yes. **Status:** owner decision.

Where: `lib/host/src/terminal/factory/mcp_relay.rs:14-58`, `lib/host/src/terminal/factory/mcp_relay.rs:87-128`, `lib/host/src/terminal/factory/mcp_relay.rs:187-220`, `lib/host/src/terminal/factory/mcp_stage.rs:9-10`, `cmd/soda-project-terminal/src/factory_mcp.rs:14`, `lib/host/src/terminal/factory/prompt_entry.rs:1-145`, and 7 more.

**Simpler.** Delete the dead relay items (UserPromptDelivery, TRUSTED_INSTRUCTION_CHANNEL, instruction_route, the CodexDeveloperInstructions variant with toml_string_body, StdioConfig/stdio_config, relay_stream_is_mcp, mcp_route, MAX_MCP_CONFIG, FACTORY_IPC_DIR, FACTORY_MCP_SOCKET, FACTORY_MCP_RELAY). Delete both prompt-entry files and the mod line. Keep one socket constant. State the native task protocol once, in the tool descriptions, and fix the receipt_digest drift in the checkpoint guidance. Decide separately whether the Codex dynamic-tool front-end stays.

**Risk.** The dead items are pub, so the compiler does not warn about them. Unifying the Codex front-end is a larger change than deleting dead code.

**Safe first step.** Delete lib/host/src/terminal/factory/prompt_entry.rs and `pub mod prompt_entry;` in lib/host/src/terminal/factory/mod.rs. Delete internal/project/factory_prompt_entry.go. In lib/host/src/terminal/factory/mcp_relay.rs, delete FACTORY_MCP_RELAY, FACTORY_IPC_DIR, FACTORY_MCP_SOCKET, MAX_MCP_CONFIG, UserPromptDelivery, TRUSTED_INSTRUCTION_CHANNEL, instruction_route, StdioConfig, stdio_config, relay_stream_is_mcp and mcp_route. Also remove the CodexDeveloperInstructions variant, toml_string_body and its match arms (along with the now-unused FACTORY_HARNESS_CODEX import), and drop the comment reference in internal/web/api/factory_mcp_p6_test.go:293 and the prompt_entry.rs mention in docs/development/factory-remaining-tasks/01-boundaries.md:182. Keep instruction_argv, instruction_config_key, json_string_body, MAX_INSTRUCTION and the MuseDeveloperPrompt variant.

### PRJ-8: Dead prompt-first entry in both Go and Rust, plus unused constants and re-exports

Project environment runtime on the host (create/join/start, executor, domain, ssh, accounts). dead-code. confirmed. ~330 lines, risk low. Visible change: none. Behaviour-preserving: yes. **Status:** done (`c6d141c6` Go, `d0435632` Rust; the dead ssh ALGO_* constants are still open).

Where: `internal/project/factory_prompt_entry.go:1-59`, `lib/host/src/terminal/factory/prompt_entry.rs:1-145`, `lib/host/src/terminal/factory/mod.rs:16`, `internal/project/factory.go:20-28`, `internal/project/factory_export.go:5-8`, `lib/host/src/ssh/mod.rs:5`, and 6 more.

**Simpler.** Delete both prompt-entry files and the `pub mod prompt_entry;` line. Remove FactoryScopeCodex/FactoryScopeMuse, MaxFactoryExportBundle and the seven unused ALGO_* constants. Remove b64_corrupt and drop it from the ssh/mod.rs re-export. Delete project/confirmation.rs along with its `mod confirmation;` line and `pub use` block in project/mod.rs. Drop the stale prompt_entry.rs mention from the remaining-tasks doc.

**Risk.** Nothing references any of it.

**Safe first step.** Go: delete internal/project/factory_prompt_entry.go, the FactoryScopeCodex/FactoryScopeMuse constants (factory.go:20-28, keeping the Harness constants) and MaxFactoryExportBundle (factory_export.go:5-8). Rust: delete lib/host/src/terminal/factory/prompt_entry.rs and its `pub mod` line in terminal/factory/mod.rs. Remove ALGO_RSA/DSS/ECDSA256/384/521/SKECDSA/SKED25519 from lib/host/src/ssh/mod.rs. Remove b64_corrupt from ssh/base64.rs and from the `pub use` on ssh/mod.rs:5. Delete lib/host/src/project/confirmation.rs and remove `mod confirmation;` and the `pub use self::confirmation::{...}` block from project/mod.rs.

### PLN-11: Departments: a speculative private org label with its own table, routes, owner check, unused list filter and creation dialog

Intent to plan (Product Lead, Architect, research, decomposition). speculative-abstraction. partially-confirmed. ~320 lines, risk low. Visible change: UI, DB schema. Behaviour-preserving: yes. **Status:** deferred to the Workspace removal (repo-first-ui). The routes and dialog live in the Workspace UI that session is replacing, so the table, routes and owner check go with it.

Where: `internal/factory/intent.go:14-28,74,113,341,357,368,379`, `internal/store/factory_intents.go:13-54,80-89,195`, `internal/web/api/factory_intents.go:15-42,77`, `internal/web/api/extension_native.go:192-193`, `internal/web/api/extension.go:231-232`, `internal/web/api/factory_request_view.go:68-74,238`, and 6 more.

**Simpler.** Delete Department: the type, the factory_departments table, the /api/factory/departments routes and admission entry, the ownership check on revision save, the backend-only list filter, the request-view context field and its unrendered frontend parse, the select and the New department dialog. Delete the tests that cover it.

**Risk.** Existing rows would simply be ignored.

**Safe first step.** No frontend code uses or renders two pieces, so remove them with no observable change: (a) the intent-list department filter, meaning IntentListFilter.Department in internal/factory/intent.go:368,379, `Department: q.Get("department")` in internal/web/api/factory_intents.go:77, and the `($5='' OR r.data->>'department'=$5)` clause with its parameter in internal/store/factory_intents.go:195; (b) requestContextView.Department in internal/web/api/factory_request_view.go:72,238 and its parse in frontend/spaces/sodaspaces-factory-progress-response.ts:74,137,140. Removing the rest (picker, dialog, table, routes) is visible to users.

### TRM-10: The soda-terminal element passes its state to its attachment, actions, screen and view modules through 22-34-member closure objects

Human terminal stack end to end (browser, Go API, host, in-project helper). pass-through-wrapper. partially-confirmed. ~320 lines, risk low. Visible change: none. Behaviour-preserving: yes. **Status:** open (behaviour-preserving).

Where: `frontend/spaces/sodaspaces-terminal.ts:160-420`, `frontend/spaces/sodaspaces-terminal-attachment.ts:10-50`, `frontend/spaces/sodaspaces-terminal-actions.ts:7-34`, `frontend/spaces/sodaspaces-terminal-screen.ts:10-35`, `frontend/spaces/sodaspaces-terminal-view.ts:32-55`.

**Simpler.** Move the attachment, actions and screen functions into SodaTerminal as private methods that use this.* directly. Delete the AttachmentInput, ActionsInput and ScreenInput interfaces and the attachmentInput/actionsInput/screenInput builders. Have terminalPresentation/terminalCommands read the element (or a small plain snapshot) instead of TerminalViewInput. renderTerminal can stay a pure template function.

**Risk.** Mechanical refactor that frontend e2e checks can verify.

**Safe first step.** Move the functions in frontend/spaces/sodaspaces-terminal-actions.ts (live, publish, detach, json, endTerminal and their helpers) into SodaTerminal as private methods using this.* fields. Then delete the ActionsInput interface and the actionsInput() builder (sodaspaces-terminal.ts:235-307), and replace calls like detach(this.actionsInput(), ...) with this.detach(...). Repeat for the screen module, then the attachment module. Keep event names, messages and the mountTerminal handle unchanged.

### IDN-11: identity_events is a write-only audit log with immutability triggers

Identity service and provider credentials. dead-code. confirmed. ~250 lines, risk low. Visible change: DB schema. Behaviour-preserving: yes. **Status:** owner decision.

Where: `cmd/soda-identity/src/store_events.rs:1-74`, `cmd/soda-identity/src/wire_execution.rs:173-228`, `cmd/soda-identity/src/store_connections.rs:48-61`, `cmd/soda-identity/src/store_connections.rs:191-204`, `cmd/soda-identity/src/store_grants.rs:23-36`, `cmd/soda-identity/src/store_grants.rs:136-149`, and 14 more.

**Simpler.** Stop writing events. Delete Store::events, append_event, lease_event and the Rust Event struct (plus its wire.rs re-export), the inline Event literals in store_connections.rs and store_grants.rs, and the four lease_event calls in store_leases.rs. Delete Go identity.Event, appendIdentityEvent, and the event parameter of seedIdentityInsert, which can fold into identityAtomic. Delete the tests that assert on the audit trail (the identity_events checks in broker.rs and the audit tests in identity_test.go). Optionally drop the table and its two triggers from both schema.rs and schema.go.

**Risk.** Dropping the table is optional (schema change); stopping the writes alone needs none.

**Safe first step.** Leave the schema as it is. Delete cmd/soda-identity/src/store_events.rs and its mod declaration. Remove the Event struct from wire_execution.rs and its re-export from wire.rs. Remove the tx.append_event(...) blocks in store_connections.rs (48-61, 191-204) and store_grants.rs (23-36, 136-149), and the four lease_event calls in store_leases.rs. Delete internal/identity/event.go. In internal/store/identity_fixture.go, remove appendIdentityEvent and the event argument of seedIdentityInsert. Delete the identity_events assertions in cmd/soda-identity/tests/broker.rs and the audit tests in internal/store/identity_test.go (TestIdentityAudit*). The table and its triggers stay for now and can be dropped later from schema.rs and schema.go.

### PRJ-3: Leftover emulation of retired Python/Go helpers in the host runtime: Python os-release parsing, Go base64 error offsets, Go-map-ordered helper stdin

Project environment runtime on the host (create/join/start, executor, domain, ssh, accounts). dead-code. partially-confirmed. ~250 lines, risk low. Visible change: none. Behaviour-preserving: yes. **Status:** open (behaviour-preserving).

Where: `lib/host/src/project/os.rs:61-271`, `lib/host/src/ssh/base64.rs:14-80`, `lib/host/src/ssh/base64.rs:96-115`, `lib/host/src/ssh/mod.rs:5`, `lib/host/src/account/mod.rs:313-330`, `lib/host/src/account/mod.rs:369-390`, and 3 more.

**Simpler.** Parse os-release as KEY=VALUE lines with plain str::lines, strip one pair of matching quotes, and keep the existing ID/VERSION_ID/control-character checks. Drop go_error_offset, source_len_for_incomplete_quantum and b64_corrupt. b64_decode_go returns a plain error, and json/mod.rs:87 uses a fixed message like the other four callers. Build the project-account and access-keys stdin with serde_json::to_vec on small structs, as member.rs already does. Keep the GO_CC/GO_CF name check, because Go's internal/host/os.go still applies the same rule, but keep only one copy (domain/mod.rs) and have terminal/text.rs import it. Keep go_arch, which podman's architecture comparison needs, and go_dir, a trivial Path::parent wrapper.

**Risk.** Odd os-release files (multiple shell words, unusual Unicode line breaks) may parse slightly differently. They only feed an informational OS label.

**Safe first step.** Three steps: 1. Delete b64_corrupt from lib/host/src/ssh/base64.rs and from the re-export in lib/host/src/ssh/mod.rs:5. It has no callers. 2. In lib/host/src/terminal/text.rs, delete the duplicate GO_CC/GO_CF/in_ranges (lines 5-35) and import them from crate::domain, made pub(crate). The tables are identical. 3. In lib/host/src/account/mod.rs, replace the two hand-built stdin strings with serde_json::to_vec on #[derive(Serialize)] structs whose fields are declared in the same alphabetical order (admin, identity, keys, login; and apply, identity, keys, login, revision). The output is byte-identical for normal login and key text, which the GO_*_BODY tests in lib/host/tests/project_operations/common.rs check. The helper reads fields by name, so it behaves the same whatever the key order.

### FEF-12: Dead exports, test-only mount functions, never-read inventory fields, a never-set generation path, and duplicate helpers in factory settings

Frontend: Factory settings, work, progress, request, intent screens. dead-code. partially-confirmed. ~240 lines, risk low. Visible change: none. Behaviour-preserving: yes. **Status:** partly done (`a9047a8a`): dead frontend exports deleted.

Where: `frontend/spaces/sodaspaces-factory-work.ts:1427-1503`, `frontend/spaces/sodaspaces-factory-work.ts:139-145`, `frontend/spaces/sodaspaces-factory-response.ts:4-41,48-89`, `frontend/spaces/sodaspaces-inventory-response.ts:6-7,12-13,79-86`, `frontend/spaces/sodaspaces-factory-progress-view.ts:45-77,82,170-178`, `frontend/spaces/sodaspaces-intent-view.ts:18-20`, and 5 more.

**Simpler.** Delete the following: - factoryAuthorityText and factoryControlText, plus spaceFactoryAuthority, spaceFactoryControl and their admit helpers, and the factory_authority/factory_control Space fields. - In progress-view.ts: intentGeneration, generationBinding, the never-set WorkOverviewInput.generation field, the issueForTask lookup and the two interfaces. - planningUnavailable. - sodaspaces-page.ts (point spaces-review-client.ts at mountSodaspaces instead). Then tidy settings.ts: merge approvalMatches with approvalMatchesRequirement and matchingPreparations with matchingPreparationRows, inline asObject and accessSpread, send verified:true, and call writeJSON directly in work.ts. Finally, change factory-work-fixture.ts to create the custom elements itself (or delete it together with its test), then delete the five mount functions and the seven admitWork* validators.

**Risk.** Only the test fixture and the dev preview script need updating.

**Safe first step.** Delete only code with zero callers; behaviour stays the same: - In frontend/spaces/sodaspaces-factory-response.ts, delete factoryAuthorityText and factoryControlText (lines 23-41). - In frontend/spaces/sodaspaces-factory-progress-view.ts, delete intentGeneration and generationBinding (lines 55-77). - Remove planningUnavailable from sodaspaces-intent-view.ts:18-20 and sodaspaces-intent.ts:2800. - In frontend/spaces/sodaspaces-factory-settings.ts: replace approvalMatchesRequirement with approvalMatches, replace matchingPreparationRows with matchingPreparations, inline asObject and accessSpread, and set verified:true in submitApproval. - In sodaspaces-factory-work.ts, replace postJSON and putJSON with direct writeJSON calls.

### DLV-12: Three startup schema initializers rewrite CHECK constraints of existing tables on every open, migration code for databases that cannot exist

Publication, merge, correction, repair, acceptance, verification, lifecycle. dead-code. confirmed. ~230 lines, risk low. Visible change: none. Behaviour-preserving: yes. **Status:** open (behaviour-preserving).

Where: `internal/store/factory_delivery_schema.go:1-61`, `internal/store/factory_repair_schema.go:1-61`, `internal/store/factory_delivery_repair_schema.go:1-103`, `internal/store/store.go:108-116`, `internal/store/schema_test.go:24-80`.

**Simpler.** Delete the three files and their calls in configureStore (store.go). Also delete the retained-scope adoption block in TestOpenCreatesOnlyTheCurrentSchema, which uses initializeDeliveryRepairSchema and deliveryRepair*Scope.

**Risk.** Only an old development database created before these branches existed would need recreating.

**Safe first step.** Delete internal/store/factory_delivery_schema.go, internal/store/factory_repair_schema.go and internal/store/factory_delivery_repair_schema.go. Remove the three `if err == nil { err = initialize…Schema(...) }` blocks at internal/store/store.go:108-116. Delete the retained-scope adoption section (about lines 24-80) of TestOpenCreatesOnlyTheCurrentSchema in internal/store/schema_test.go, since it references the removed functions and constants.

### HPL-12: Dead code and stale port leftovers in the host crate (excluding the test-used StubBackend)

Rust host platform: daemon, terminals, project, application, tailnet (lib/host non-factory, soda-project-maintain, forgejo/pg/console helpers, soda-tailnet, unix-http, wire-time). dead-code. partially-confirmed. ~210 lines, risk low. Visible change: none. Behaviour-preserving: yes. **Status:** done (`d0435632`).

Where: `lib/host/src/project/confirmation.rs:1-106`, `lib/host/src/project/mod.rs:31,38-41`, `lib/host/src/terminal/broker.rs:12-25`, `lib/host/src/terminal/broker.rs:112-146`, `lib/host/src/terminal/mod.rs:8-10,94-97`, `lib/host/src/tcontrol_wire.rs:1478 (valid_lifecycle_state), ~1526-1575 (validate_lifecycle_observation)`, and 1 more.

**Simpler.** Delete confirmation.rs and its mod/pub use. Delete the broker.rs valid_identity_request, IdentityRoute, identity_route and identity_action(path), and narrow the terminal/mod.rs re-export to identity_launch, rand_id and IdentityBroker. Delete validate_lifecycle_observation together with valid_lifecycle_state, and companion_stop. Fix the stale 'stays in Go' header. Keep StubBackend, which the gmux smoke tests use.

**Risk.** Pure deletion. A build check confirms it.

**Safe first step.** Delete lib/host/src/project/confirmation.rs and remove `mod confirmation;` and its `pub use` block from lib/host/src/project/mod.rs. In lib/host/src/terminal/broker.rs, remove the five-argument valid_identity_request (lines 12-25) and IdentityRoute, identity_route and identity_action (lines 112-146), and trim the re-export at terminal/mod.rs:94-97 to identity_launch, rand_id and IdentityBroker. Remove validate_lifecycle_observation and the private valid_lifecycle_state from lib/host/src/tcontrol_wire.rs, and companion_stop from lib/host/src/dbackend/mod.rs. Leave daemon/backend.rs StubBackend alone.

### EXT-7: dbackend keeps a Go-port compatibility layer: hand-rolled case-insensitive decoders, a hand-written JSON encoder and HTTP status by error-variant aliasing

Fountain extension and its backend. dead-code. confirmed. ~200 lines, risk low. Visible change: none. Behaviour-preserving: yes. **Status:** open (behaviour-preserving).

Where: `lib/host/src/dbackend/mod.rs:36-82`, `lib/host/src/dbackend/tailnet.rs:46-63,73-151,190-268`, `lib/host/src/dbackend/backend.rs:87-108`, `lib/host/src/daemon/backend.rs:16-42`, `lib/host/src/daemon/routes.rs:376-391`, `lib/host/src/dbackend/factory_terminal.rs:11-41`, and 1 more.

**Simpler.** For the request wire, use a serde derive with #[serde(deny_unknown_fields, default)]. For the empty body, use Option<Empty> with a derive and deny_unknown_fields. Derive Serialize on ProjectView, with skip_serializing_if on its four omitempty fields. Give BackendError explicit Conflict/Unsupported/SetupClaim* variants, or make it carry a status, and have the Tailnet and application runtimes return typed errors instead of strings. Keep the exact status and body pairs the Go routes rely on. Use one Lease/FactoryRun/OutputSlice type across factory and terminal.

**Risk.** Privileged host process but mechanical changes; keep validate_project_request, which is the real dashboard-to-root boundary. A host-runtime audit may raise the same converters.

**Safe first step.** Two steps in lib/host/src/dbackend/tailnet.rs. First, delete encode_project_view (lines 190-268). Add #[derive(Serialize)] to ProjectView in lib/host/src/tailnet/domain/mod.rs, with #[serde(skip_serializing_if = "String::is_empty")] on available_binding, available_network and dns_name, and "Vec::is_empty" on addresses. Return serde_json::to_vec(&out) at tailnet.rs:403. Field order already matches Go, so the bytes are identical. Second, replace the hand-written TailnetProjectWire Visitor (lines 81-137) with #[derive(Deserialize)] #[serde(deny_unknown_fields)] and keep its Option<String> fields. The only senders use exact lowercase keys, so the outcome is unchanged. Keep validate_project_request as it is.

### FEP-11: Host lifecycle passes through redundant wrapper layers: duplicate generation checks, unused Tailnet lifecycle methods, fixture-only sodaspaces-page.ts and an unchecked contract hash

Frontend: Workspace, project, applications, terminals, layout, Forgejo and Tailnet pages. pass-through-wrapper. partially-confirmed. ~200 lines, risk low. Visible change: none. Behaviour-preserving: yes. **Status:** open (behaviour-preserving).

Where: `frontend/spaces/soda-extension.ts:1-120`, `frontend/spaces/soda-native-paths.ts:1-16`, `frontend/spaces/soda-spaces-entry.ts:13-80`, `frontend/spaces/sodaspaces-workspace.ts:2164-2186,2235-2284`, `frontend/spaces/sodaspaces-terminal.ts:421-446,476-526`, `frontend/spaces/sodaspaces-project.ts:625-646,830-855`, and 8 more.

**Simpler.** Check the generation only in the entries and remove resumeGenerationRefused and its nested calls. Delete hostSuspend/hostResume/hostRetire, and fold applyPresentation/resumeHost/retireHost into a plain suspend/resume/retire on the Tailnet page. Delete sodaspaces-page.ts and point the review fixture at mountSodaspaces or the entry. Remove assertLive and requireHostTransport, and strip the sha256/re-pin header lines from the mirror. Treat removing the appSubUrl regex and the owner/repository validation as optional, because whether the host enforces them cannot be checked in this repo.

**Risk.** tests/frontend/workspace-owner.test.ts asserts substrings of the entry and the mirror; adjust it. Much of the Workspace part goes with FEP-1 anyway.

**Safe first step.** In frontend/tailnet/soda-tailnet-page.ts, delete the uncalled hostSuspend/hostResume/hostRetire methods (lines 137-145). In frontend/spaces/fountain-browser-contract.d.ts, delete the sha256(body) and 'lane C re-pins' header comment lines. Both changes have identical runtime behaviour.

### HFW-6: Drive loop re-reads the receipt each iteration; 100 ms polls redo podman/admission work; write-only protocol trace; unreachable native-parent fences

Rust host: factory worker runtime. unneeded-validation-or-digest. partially-confirmed. ~180 lines, risk medium. Visible change: none. Behaviour-preserving: yes. **Status:** open (behaviour-preserving).

Where: `lib/host/src/factory/native_control.rs:69-131,180-300,383-395,681-690`, `lib/host/src/factory/native_tasks.rs:414-454,513-537`, `lib/host/src/factory/checkpoint/mod.rs:59-125,353-620`, `lib/host/src/factory/checkpoint/state.rs:20-55`, `lib/host/src/factory/verification_execution.rs:263-348`, `lib/host/src/factory/receipt.rs:346-462`.

**Simpler.** Delete NativeTrace and its native-*-protocol.json writes. Delete the two unreachable existence fences: native-*.json in drive_muse and the native-tasks ledger in Ledger::new. Admit a checkpoint once per task_checkpoint call. Raise the final-candidate and verification polls from 100 ms to about 1 s. Optionally open the PrivateState fd once in open_factory and keep a lightweight in-memory stop flag next to the receipt, keeping the on-disk receipt as the crash-recovery truth. Keep the stop-before-start tombstone: it makes a launch that arrives after its stop refuse, and the Go retry logic keys on it.

**Risk.** The interleaving of stop with reserve and start must stay correct; the on-disk receipt remains for crash recovery. Changing the tombstone reason means updating the Go retry checks that match 'stop-before-start'.

**Safe first step.** In lib/host/src/factory/native_control.rs: - Delete the NativeTrace struct and impl (about lines 69-131). - Delete its construction and observe() calls that write native-{project}-{run}-protocol.json (around lines 213 and 508). - Delete the native-*.json existence check in drive_muse (lines 189-197). In lib/host/src/factory/native_tasks.rs: - Delete the native-tasks ledger existence check in Ledger::new (lines 427-434). None of these is read anywhere, and the fences cannot fire because drive runs only after launch_new has stored a fresh receipt for a run that had none.

### PRP-7: Dead preparation code: LifecycleGrant + table, unused path constants, never-set Credential binding; AdmitPreparation is a test-only fixture

Project preparation pipeline below the API (standing and candidate preparations, decisions, hold, host and root helper). dead-code. partially-confirmed. ~180 lines, risk low. Visible change: DB schema. Behaviour-preserving: yes. **Status:** owner decision.

Where: `internal/project/preparation.go:33-41,185-200,215,239-241`, `internal/store/preparation.go:15-53,93-127`, `internal/store/preparation_test.go:42-60`, `internal/store/schema.go:167,365`, `cmd/soda-identity/src/schema.rs:200,332`, `internal/factory/control/publication_children.go:777`, and 5 more.

**Simpler.** Delete LifecycleGrant, SaveLifecycleGrant, LifecycleGrant(), and TestLifecycleGrantRevisionRejectsStaleWriters. Drop project_lifecycle_grants from both schema lists (Go and soda-identity Rust) and from the doc row. Delete the five unused Go constants and Rust FACTORY_HOLD_FILE. Remove the never-set Preparation.Credential end to end: the Go field and validation, the lib/host field and mapping, the helper's request and stored key sets, check_credential_file, the credential_file response field, SODA_CREDENTIAL_FILE, and the oracle expectations. AdmitPreparation can go only if its roughly 25 test-fixture call sites are moved to AdmitStandingPreparation or a test insert. Otherwise leave it.

**Risk.** Dropping the table needs a schema edit. Leaving the table and deleting only the code is zero-risk.

**Safe first step.** In internal/project/preparation.go, delete the LifecycleGrant type and its Validate method (lines 185-200) and the constants FactoryDir, FactoryPreparationsDir, FactoryCredentialsDir, FactoryHoldFile and FactoryHelper (keep FactorySetupEntry and FactoryCheckEntry). In internal/store/preparation.go, delete SaveLifecycleGrant and LifecycleGrant (lines 15-53). In internal/store/preparation_test.go, delete TestLifecycleGrantRevisionRejectsStaleWriters. In lib/host/src/preparation/mod.rs, delete FACTORY_HOLD_FILE. Leave the project_lifecycle_grants table in place in internal/store/schema.go and cmd/soda-identity/src/schema.rs for a later schema edit.

### RBA-5: Hand-rolled HTTP redirect followers (two of them) over a one-impl transport trait, plus generic *_with twins of the fetches

Release build primitives, asset fetch/render tools, small install binaries and system units. speculative-abstraction. partially-confirmed. ~180 lines, risk low. Visible change: none. Behaviour-preserving: yes. **Status:** open (behaviour-preserving).

Where: `lib/soda-release-build/src/http.rs:1-236`, `lib/soda-release-build/src/coreos_stream.rs:228-287`, `lib/soda-release-build/src/coreos.rs:54-110,180-200`, `lib/soda-release-build/src/coreos_iso.rs:23-78`, `lib/soda-release-build/src/coreos_registry.rs:16-50`, `lib/soda-release-build/src/fountain_inputs.rs:27-34,100,143-160`, and 2 more.

**Simpler.** Drop the HttpTransport trait and every *_with/download_http generic twin. Call ureq directly through one ureq Agent built with `.redirects(5).https_only(true)` for soda-release-build fetches. Delete get_follow, resolve_location, split_url and redirect_url. Delete the unused resolve_coreos_iso/resolve_coreos_qemu wrappers. In fountain_pin.rs, replace the get_document redirect loop with a ureq agent with redirects enabled (not https_only, so its loopback http allowance is kept, or keep admit_url on the initial URL only).

**Risk.** The HTTPS-only redirect policy is kept through ureq's own setting.

**Safe first step.** Delete the uncalled pub wrappers resolve_coreos_iso and resolve_coreos_qemu in lib/soda-release-build/src/coreos_stream.rs (228-230, 245-247). Then remove the HttpTransport trait: make get_follow, fetch_capped_json, download_http, fetch_verified_iso, download_verified_archive, fetch_fountain_metadata, source_file and resolve_registry_digests_with take no transport and call UreqTransport's get logic as a plain function. Fold each *_with into its public name (fetch_coreos, fetch_coreos_iso, resolve_coreos, resolve_fountain_release, download), and update the README line. This keeps the redirect policy and error texts exactly as they are.

### STO-10: Store repeats the outstanding-publication and comment-visibility SQL predicates, the command-ledger record/finish, the decision-chain admission and small querier interfaces

Store (PostgreSQL layer) structure. duplicated-layer. partially-confirmed. ~180 lines, risk low. Visible change: none. Behaviour-preserving: yes. **Status:** coordinate with prod-hardening.

Where: `internal/store/factory_publications.go:494-512`, `internal/store/factory_publications.go:537-555`, `internal/store/factory_publications.go:612-630`, `internal/store/factory.go:459-532`, `internal/store/factory_grant_commands.go:503-539`, `internal/store/project_grants.go:97-160`, and 12 more.

**Simpler.** Put the outstanding-publication predicate in one const or helper used by all three queries. Make one shared decision-chain admission helper for project and acceptance decisions, with parameters for the head lock, the extra key column and a post-insert hook. Have RecordFactoryCommand/FinishFactoryCommand call the Tx versions in a short transaction, keeping their current error sentinels or updating callers. Use one sqlQuerier interface. Move the grant arms' Validate/Revision++ boilerplate into a small helper. Write the visible-comment predicate once wherever it is used as a filter.

**Risk.** Pure refactor. The integration tests that already exist exercise these paths.

**Safe first step.** In internal/store/factory_publications.go, move the identical outstanding-publication WHERE clause into a single const and splice it into the three queries: the repository listing at 494-512, requestPublicationWithdrawalsTx at 537-555 and OpenPublications at 612-630. OpenPublications leaves out the repository filter, so the const covers only the shared OR-expression. Also replace the five single-use querier interfaces (factory_library.go, factory_explicit_retry.go, factory_feed_contribution.go, issue_controls.go, factory_grant_commands.go) with one interface. Both changes are identical SQL and identical behaviour.

### STO-12: The test-only pg-fixture is hardened like a privileged service, the PostgreSQL image digest is hand-pinned in 7 places, and stale shell-parity comments remain

Store (PostgreSQL layer) structure. process-machinery. confirmed. ~180 lines, risk low. Visible change: none. Behaviour-preserving: yes. **Status:** open (behaviour-preserving).

Where: `tools/pg-fixture/src/main.rs:108-310`, `tools/pg-fixture/src/main.rs:23-25`, `tools/pg-fixture/src/main.rs:41-43`, `system/host/services/soda-postgres.container:13`, `scripts/pg_runtime_test.go:31-40`, `scripts/pg_runtime_test.go:186-202`, and 8 more.

**Simpler.** In the fixture, use tempfile's tempdir, or create_dir_all plus set_permissions(0700) for SODA_PG_FIXTURE_DIR. Write each password with OpenOptions mode(0o600) plus write_all. Drop the openat2 walk, the fstat checks and the readback. In scripts/pg_runtime_test.go, drop the hardcoded Image= digest from TestPostgresUnitTopology and keep only the existing check that the fixture matches the quadlet. Optionally have the fixture read Image= from the quadlet at runtime. Remove the "Rust port of ... matches the shell exactly" and "Std-only" claims from the comments in the three soda-pg-maintenance binaries, pg-fixture and pg_backup_test.go.

**Risk.** Developer tooling only. The fixture directory is caller-owned on a dev machine.

**Safe first step.** Delete the stale shell-parity and Std-only comment lines in cmd/soda-pg-maintenance/src/bin/soda-pg-backup.rs:13-14, soda-pg-restore.rs:13-14, soda-pg-init-roles.rs:13-14, tools/pg-fixture/src/main.rs:23-25 and scripts/pg_backup_test.go:23,62. Remove the hardcoded digest line from TestPostgresUnitTopology in scripts/pg_runtime_test.go:34; the existing fixture-matches-quadlet check covers it. Then in tools/pg-fixture/src/main.rs, replace split_absolute_parts/walk_components/check_private_dir/open_secret_dir with create_dir_all plus a 0700 chmod, and replace write_password's fstat checks and readback with an OpenOptions mode(0o600) write. Start/stop output and exit codes stay identical.

### GOM-2: ServiceBackground duplicates the SDK's background client (bootstrap, token and op-id validation, decoding) with an unused installation parameter and a dead observer fallback

Other Go packages (forgejo, acceptance, project, host, tailnet, application, config, strictjson, filelock, avatar, cmd/soda-dashboard, cmd/soda-tailnet, cmd/soda-extension). duplicated-layer. partially-confirmed. ~170 lines, risk medium. Visible change: none. Behaviour-preserving: yes. **Status:** open (behaviour-preserving).

Where: `internal/forgejo/background.go:1-172`, `internal/forgejo/background_admission.go:1-218`, `internal/forgejo/background_transport.go:1-121`, `internal/forgejo/observe.go:20-95`, `internal/forgejo/lifecycle.go:76-130,282-317`, `../forgejo-ext/sdk/background.go:212-480`.

**Simpler.** Behaviour-preserving now: drop NewServiceBackground's installation parameter and the `installation` field and checks; `pinned`, learned from bootstrap, stays. Pass the shared ServiceBackground into NewServiceObserver and delete its socket/hostUID fields, sharedBackgroundLocked and ShareBackground. Optionally fold LifecycleWriter into LifecycleExecutor. Bigger step: make the SDK's backgroundClient the single client. Move Soda's StatusError-with-refusal-body, per-request peer verification, rebind-on-401 with singleflight bootstrap and OnBootstrap, and admission access for PublishPushEnv into ../forgejo-ext/sdk. Then delete the copied validators, the bootstrap code and the post/decoding code in background_admission.go and background_transport.go.

**Risk.** Bootstrap concurrency (bootstrapCall singleflight, OnBootstrap hook) has to be preserved in whichever single client remains. The change touches the sibling SDK repo.

**Safe first step.** In internal/forgejo/background.go and background_admission.go: remove the `installation` parameter and field from NewServiceBackground/ServiceBackground, and remove the `installation != "" && ...` mismatch check and the installation_id request body field in performBootstrap; production always sends "", and omitempty sends no field. Update the three production call sites (internal/web/server.go:69,76) and the test callers; delete the "install-2" pinned test. In internal/forgejo/observe.go: change NewServiceObserver to take the *ServiceBackground, then delete the socket/hostUID fields, sharedBackgroundLocked and ShareBackground, and update server.go:77-79.

### HFW-8: Execution and verification manifests re-decoded and re-verified on every load; most VerificationManifest fields only shape-checked

Rust host: factory worker runtime. unneeded-validation-or-digest. partially-confirmed. ~170 lines, risk low. Visible change: none. Behaviour-preserving: yes. **Status:** open (behaviour-preserving).

Where: `lib/host/src/factory/run.rs:651-707`, `lib/host/src/factory/manifest.rs:228-305`, `lib/host/src/factory/execution_state.rs:18-99`, `lib/host/src/terminal/factory/execution_manifest.rs:12-23`, `lib/host/src/factory/native_control.rs:258-265,519`, `lib/host/src/factory/checkpoint/mod.rs:100-104,796-800`, and 3 more.

**Simpler.** Verify the execution manifest and verification manifest once, at launch and store time. Every later load from host-private state becomes a plain decode, with no repeat of verify_for_run or the digest checks. Remove the second digest comparison inside ExecutionManifest::verify_for_run. Remove the shape-only validation of VerificationManifest fields the host never reads (intent, accepted_revision, scope_digest, adopted_manifest_digest, requirements, approval, preparation_digest, outcomes, and the pinned constants on checks). Later, Go can stop sending them. Keep run.assignment and run.manifest_digest, because they are run identities used by Go and by the MCP wire. Keep the two plan projections; they read different fields. Keep the helper's invocation_id echo check, which catches stale state.

**Risk.** The host can stop requiring the extra fields before Go stops sending them.

**Safe first step.** In lib/host/src/factory/verification_execution.rs, make load_verification_manifest decode only and drop its verify_for_run call; the bytes were verified in store_verification_manifest before being written to PrivateState. In lib/host/src/factory/execution_state.rs:93-95, drop verify_for_run on reload. In lib/host/src/factory/manifest.rs, remove the second sha256 comparison in verify_for_run, since run.rs:686 already compares the same digest. In VerificationManifest::validate (verification.rs:53-96), drop the checks on the unread fields while keeping them as struct fields, so the strict decode still accepts what Go sends.

### APP-10: Optional preview environment doubles build/app lanes (company|preview|unknown) across Go, Rust, frontend and worker prompts

Application runtime: build, home, snapshots, backups, restore, factory delivery. too-many-states-or-budgets. partially-confirmed. ~150 lines, risk medium. Visible change: UI, model tools, DB schema. Behaviour-preserving: no. **Status:** owner decision.

Where: `internal/store/applications.go:291-312`, `internal/web/api/applications.go:140-164`, `internal/store/application_factory_delivery.go:88-91,151-177`, `internal/store/application_builds.go:16,82-85`, `internal/application/build.go:9-23`, `lib/host/src/factory/application.rs:47-56`, and 7 more.

**Simpler.** If the owner confirms preview is not a demo requirement, build and run one app per delivery. Drop the preview output from verification_bundle.rs and application_preview_bundle_sha256 from verification checks (Go and Rust). Drop Environment from BuildRequest/BuildArtifact, the environment column and the unique key part from application_builds, Store.ApplicationEnvironment, API.applicationEnvironment and the 'unknown' environment state. Remove the preview sentences from ApplicationBundleGuidance, from the factory settings text and docs, and the preview half of the founder-demo fixture.

**Risk.** The verifier (cmd/soda-project-terminal factory_roles/verification_bundle.rs) emits both bundles, so the change spans the verification code (other finders' scope). Confirm with the owner that preview is not a demo requirement.

### WEB-10: /factory/access duplicates /factory's authority and preparation reads and hand-restates the settings PUT gates; the frontend has two parsers for it

Web API and view models (non-MCP, non-Feed, non-extension): internal/web/*.go, internal/web/auth/**, internal/web/api/*.go excluding factory_mcp*, factory_feed*, extension*. duplicated-layer. partially-confirmed. ~150 lines, risk medium. Visible change: none. Behaviour-preserving: yes. **Status:** coordinate with prod-hardening.

Where: `internal/web/api/factory_access_view.go:15-114,232-407`, `internal/web/api/factory_status.go:89-99`, `internal/web/api/factory_settings.go:55-84`, `frontend/spaces/sodaspaces-factory-access-response.ts:155-194`, `frontend/spaces/sodaspaces-factory-progress-response.ts:695-822`.

**Simpler.** Use one frontend parser for /factory/access: progress-response.ts factoryAccess and its helpers become callers of factoryAccessResponse plus the repository check. Optionally fold the access fields (viewer, editable, required_actions, usage, planning) into GET /factory so EffectiveAuthority and preparation are computed once and Settings and Work make one request. Derive each editable flag from small boolean predicates that the factoryOwner, factoryOperator and environmentAdministrator gates also call. Keep session_current, the usage rows and the read-only reasons, because the UI renders them. workCapabilitiesDTO is mostly record state and can stay.

**Risk.** Several components read /factory/access, so consolidate the frontend parsers at the same time. On the hardening branch, sodaspaces-factory-work-response.ts parses /factory and progress-view.ts reads access.viewer.

**Safe first step.** In frontend/spaces/sodaspaces-factory-progress-response.ts, delete factoryAccess and its private helpers (accessViewer, accessEntry, accessRequiredAction, accessEffective, accessPlanningEntry; about lines 695-822) along with the duplicate interfaces. Make work.ts:724 call factoryAccessResponse from sodaspaces-factory-access-response.ts plus a repository-equality check, and keep the planning available/status arithmetic check in that single parser. Make no server change in this step. Coordinate with prod-hardening, which edits progress-view.ts.

### BND-7: About 38 Postgres immutability/guard triggers, plus their startup existence checks, protect SodaOS tables against SodaOS's own code

Boundary machinery: strict JSON, digests, receipts, revision checks (cross-cutting). unneeded-validation-or-digest. partially-confirmed. ~135 lines, risk low. Visible change: DB schema. Behaviour-preserving: yes. **Status:** owner decision.

Where: `internal/store/schema.go:172-277`, `internal/store/schema.go:417-450`, `internal/store/schema.go:539-557`, `internal/store/factory_completions.go:63-70`, `internal/store/factory_mcp_submission.go:50-53`, `internal/store/factory_mcp_capability.go:44-46`.

**Simpler.** Delete soda_reject_immutable, the six soda_guard_* functions, every *_immutable / *_no_delete trigger statement and the four trigger constants, along with verifyTrigger and the four verifyImmutable* functions and their calls in verifyCurrentSchema. Keep the feed-order triggers (soda_feed_merge_order and soda_feed_judgment_order), which compute feed_seq. If you want a bug to fail loudly, add `AND finished=''` to the two delivery-repair outcome UPDATEs. Do not replace the triggers with new machinery.

**Risk.** There are no users and no external writers. A buggy write becomes possible instead of raising an exception, which the owner policy accepts.

**Safe first step.** In internal/store/schema.go, delete verifyTrigger, verifyImmutableCreationProfile, verifyImmutablePreparationRefs, verifyImmutableExecutionIdentity and verifyImmutableApplications (lines 417-450) and their four calls in verifyCurrentSchema (lines 545-556). The triggers are still created, so behaviour is identical. Then remove the DELETE-only soda_reject_immutable triggers (the *_immutable_delete and *_no_delete statements in schema.go:240-275, plus FactoryCompletionDeleteTrigger, FactoryReassessmentDeleteTrigger, FactoryMCPCapabilityDeleteTrigger and FactoryMCPSubmissionTriggers and their schemaStatements entries). No code issues DELETE on those tables, so behaviour stays identical. Delete the tests in factory_reassessments_test.go, factory_mcp_test.go and factory_assignment_finish_test.go that assert the triggers fire.

### DSP-7: Ten hand-built launch structs, one redundant manifest re-parse, constant MCPKey/Store checks, and legacy no-tool-proof branches

Factory dispatch and run start. duplicated-layer. partially-confirmed. ~130 lines, risk low. Visible change: none. Behaviour-preserving: yes. **Status:** coordinate with prod-hardening.

Where: `internal/factory/control/dispatch_launch.go:99-114,266-287,360-372,622-636,975-989`, `internal/factory/control/dispatch_launch.go:160-182,706-756`, `internal/factory/control/dispatch_context_recovery.go:115-169`, `internal/factory/control/dispatch_timeonly.go:553-587`, `internal/factory/control/intent_bound.go:180-200`, `internal/factory/control/publication_children.go:1033-1052`, and 2 more.

**Simpler.** One launchFor(assignment, run, pin, manifest) helper used by every site, validated once in launchTail. Drop the Parse-after-Marshal in stageExecutionManifest; keep verifyStagedManifest (it does the run-binding check) and Run.Validate (store invariant). Delete the zero-tool-proof compatibility branches, the always-true MCPKey/Store checks in launchTail (updating test fixtures to supply a key), and the dead bundle/reason initializer.

**Risk.** Mechanical, and identical for rows the current code writes. Dev databases holding pre-proof rows would refuse those rows instead of applying the legacy rule (acceptable with zero users).

**Safe first step.** In internal/factory/control/dispatch_launch.go: delete the dead initializer `bundle, reason := factory.ContextBundleSnapshotHold, "context_selection_unavailable"` (line 748) and declare the variables at line 753. Remove the redundant `project.ParseExecutionManifest(data)` check in stageExecutionManifest (lines 176-178); verifyStagedManifest re-validates before any host call. Then add a launchFor helper that builds project.FactoryLaunch from the assignment, run, pin and manifest, and replace the five construction sites in dispatch_launch.go plus those in publication_children.go:1033 and dispatch_timeonly.go:567 with it, leaving field values identical.

### PRJ-4: OS release observation validated in four places across three languages, plus a dead fifth copy in confirmation.rs

Project environment runtime on the host (create/join/start, executor, domain, ssh, accounts). unneeded-validation-or-digest. partially-confirmed. ~130 lines, risk low. Visible change: none. Behaviour-preserving: yes. **Status:** partly done (`d0435632`): dead fifth copy deleted; the four live validators remain.

Where: `lib/host/src/project/os.rs:9-58`, `lib/host/src/project/os.rs:203-271`, `lib/host/src/domain/os.rs:45-83`, `internal/host/os.go:14-62`, `internal/web/api/environment_os.go:28-50`, `frontend/spaces/sodaspaces-project-response.ts:104-142`, and 2 more.

**Simpler.** Validate once, in the Rust parse_os_release. Fold in the Unicode Cc/Cf name check and delete valid_release_id/valid_release_version, or the domain copies, keeping one pair. Delete the dead OS confirmation functions in confirmation.rs and the confirm_observe_os re-export. Reduce Go ObserveOS to checking out.Environment.ID == id. Reduce the frontend to type checks. The `test -f` exec can optionally go: a missing file or directory still fails at once into 'unavailable'; only a FIFO would wait until the deadline.

**Risk.** The OS release is informational and does not gate anything.

**Safe first step.** In lib/host/src/project/confirmation.rs, delete valid_os_environment, valid_os_observation and confirm_observe_os (lines 53-89), and remove confirm_observe_os from the `pub use self::confirmation::{...}` in lib/host/src/project/mod.rs. Nothing calls them. Then, in lib/host/src/project/os.rs, delete valid_release_id and valid_release_version and rely on the domain::valid_os_release call already made in observe_os at os.rs:52. parse_os_release must still reject the missing-ID and missing-version cases it handles today; it can call the domain helpers, which would need to become pub(crate). Both steps leave observable behaviour identical.

### RBA-3: Canlang staging re-verifies an in-archive file inventory; locale lock is a third Forgejo-version pin; Tailscale digest in Rust source

Release build primitives, asset fetch/render tools, small install binaries and system units. process-machinery. partially-confirmed. ~130 lines, risk low. Visible change: none. Behaviour-preserving: yes. **Status:** open (behaviour-preserving).

Where: `tools/release-assets/src/fetch/canlang.rs:96-220`, `system/project/canlang-release.json`, `lib/soda-release-tools/src/release_start.rs:256-284`, `lib/soda-release-image/src/canlang.rs`, `frontend/forgejo/locale.lock.json`, `tools/release-assets/src/locales/merge.rs:217-240`, and 3 more.

**Simpler.** Canlang: after extraction, keep only the required-binaries check (bin/can-soda and the other tools) and the tools.json schema check. Delete verify_runtime and runtime_paths (the in-archive runtime.json per-file sha/bytes/mode re-verification, the 10-path list and the extra-file refusal). Keep the archive sha256, because the host uses it as the tools_sha256 runtime identity (io.canlang.tools label and change detection). Optionally compute it at build time instead of committing it, and stop rechecking it in release_start.rs and soda-release-image. Locales: derive the locale URL from FORGEJO_UPSTREAM_BASE, or commit the merged catalog, and collapse the two duplicated FORGEJO_UPSTREAM_BASE constants. Tailscale: drop the constant with Tailnet (RBA-11), or leave it, since it is a single line.

**Risk.** These are local build inputs with no adversary. Re-verifying an archive the owner just built protects nothing.

**Safe first step.** In tools/release-assets/src/fetch/canlang.rs, delete fn runtime_paths and fn verify_runtime (about lines 121-220) and the verify_runtime(root, &tools)? call in verify_staged_tree. Keep the required-files loop, the tools.json schema/sourceRevision check and the archive sha check unchanged. For any archive the owner builds, the staged output is identical.

### PTH-7: Test seams and injectable closures inside root production helpers

In-project helpers (soda-project-terminal) and the host Muse launch runtime. speculative-abstraction. confirmed. ~120 lines, risk low. Visible change: none. Behaviour-preserving: yes. **Status:** open (behaviour-preserving).

Where: `cmd/soda-project-terminal/src/project_account.rs:57-112`, `cmd/soda-project-terminal/src/project_account.rs:572`, `cmd/soda-project-terminal/src/project_account.rs:653-672`, `cmd/soda-project-terminal/src/bin/project-account.rs:75-85`, `cmd/soda-muse/src/store_resolve.rs:25-37`, `lib/host/src/terminal/native.rs:98-105`, and 4 more.

**Simpler.** Hard-code the production paths, uid/gid 0, the root gate and NSS home lookup. Call run_command_inherit and File::sync_all directly. Drop fail_sync, test_root, test_passwd_home and config_from_env. Replace staged_path()/receipt_path() with the constants MUSE_NATIVE/DEFAULT_RECEIPT and agent_program_path() with AGENT_PROGRAM. That makes the soda-project-maintain "!= DESTINATION" guards trivially true, so they can go too. Delete cmd/soda-project-terminal/tests/project_account.rs and tests/build/project_account_test.go, which exist only to drive the seams.

**Risk.** Loses the tests/project_account.rs and parts of soda-muse tests that drive the binary against a scratch root.

**Safe first step.** These two steps have no consumer at all. In cmd/soda-muse/src/store_resolve.rs, delete the SODA_MUSE_STAGED_PATH/SODA_MUSE_RECEIPT_PATH env reads and use MUSE_NATIVE and DEFAULT_RECEIPT directly. In lib/host/src/terminal/native.rs, make agent_program_path() return AGENT_PROGRAM (dropping the SODA_PROJECT_TERMINAL read). Next, in project_account.rs, remove the RunCommand/SyncFile boxed closures from Config and call run_command_inherit and file.sync_all() directly; behaviour stays identical.

### RBA-12: --arch knob threaded through every release tool, but only x86_64 is accepted

Release build primitives, asset fetch/render tools, small install binaries and system units. unused-configuration. confirmed. ~120 lines, risk low. Visible change: none. Behaviour-preserving: yes. **Status:** open (behaviour-preserving).

Where: `lib/soda-release-build/src/files.rs:92-118`, `lib/soda-release-build/src/production.rs:19-65`, `lib/soda-release-build/src/elf.rs:10-12`, `lib/soda-release-build/src/production_assets.rs:24-27,131-133,164-166`, `lib/soda-release-build/src/coreos_stream.rs:227-258`, `lib/soda-release-build/src/coreos.rs:41-42`, and 8 more.

**Simpler.** Hard-code x86_64/amd64 in one constant. Drop the arch parameters and the repeated architecture admissions inside the libraries. Then drop the --arch flags from the release CLIs and update their callers, tests and docs in the same change.

**Risk.** Only one value is ever accepted today.

**Safe first step.** Keep the CLI flags for now and remove only the redundant internal re-checks. Delete the arch parameter and its check from elf.rs inspect_elf, and update the callers in production_compile.rs. Delete the oci_architecture(arch)? calls in coreos_stream.rs resolve_coreos_iso_with/resolve_coreos_qemu_with and the one in muse.rs read_manifest_root. Replace the `oci_architecture(&self.arch).unwrap_or("amd64")` calls in production_images.rs/production_inputs.rs with an "amd64" constant. Entry-point validation (require_native, Production::validate) still rejects any other value, so observable behaviour stays the same.

### RTY-6: Completion retry: dead entry points, three types for one request, and a client-side receipt-recovery protocol duplicating server replay

Retry kinds across store, control and UI. dead-code. partially-confirmed. ~120 lines, risk low. Visible change: UI. Behaviour-preserving: yes. **Status:** owner decision.

Where: `internal/factory/control/completion_drive.go:564-583`, `internal/store/factory_reassessments.go:675-697`, `internal/factory/completion_retry.go:10-39`, `internal/factory/control/completion_retry.go:17-68`, `internal/store/factory_completion_retry.go:18-60`, `frontend/spaces/sodaspaces-factory-completion-retry.ts:104-180`.

**Simpler.** Delete Coordinator.StartCompletionRetry and Store.StartReassessmentRetry. Delete the tests that call them, and rewrite or delete the fixture use in factory_journey_readback_test.go:1151. Keep checkCompletionRetry and startReassessmentRetryTx. Optionally reduce the decision to {command_id, successor_cycle}. On the client, re-POST the same command_id on failure instead of the GET-and-verify recover() path. Treat 200 as queued and 409 as refused, and use a single success message.

**Risk.** The dead functions have no production callers. Removing the client recovery relies on server replay, which already exists.

**Safe first step.** In internal/factory/control/completion_drive.go, delete StartCompletionRetry and its doc comment (lines 564-583). In internal/store/factory_reassessments.go, delete StartReassessmentRetry and its doc comment (lines 675-697). Delete the tests that call them: completion_serving_pg_test.go around 320-340, dispatch_timeonly_test.go around 1015, factory_reassessments_test.go around 510-535, and the fixture call in web/api/factory_journey_readback_test.go:1151. Keep checkCompletionRetry and startReassessmentRetryTx, which the live RetryCompletion path uses. Leave the frontend recover() change for a separate step, because it changes UI text.

### APP-6: Shape validation is repeated across Go client, Go store reads and Rust; drop read-side re-validation, the custom observation decoder and Go pre-send request checks, and keep Go write-time validation

Application runtime: build, home, snapshots, backups, restore, factory delivery. duplicated-layer. partially-confirmed. ~110 lines, risk low. Visible change: none. Behaviour-preserving: yes. **Status:** open (behaviour-preserving).

Where: `internal/store/applications.go:78-93,129-140,262-289`, `internal/store/application_restore.go:13-25`, `internal/host/application.go:11-50`, `internal/application/administration.go:43-77`, `internal/application/reference.go:36-118`, `lib/host/src/application/wire.rs:1-195`, and 2 more.

**Simpler.** Keep Go Spec.Validate, Observation.Validate(spec) and Replacement at write time and on API input, because Rust validates only at execution time. Stop re-validating rows on scan in scanApplication, applicationInventory, ApplicationRevision and scanApplicationRestore. Replace applicationResponse.UnmarshalJSON with plain json.Unmarshal; Observation.Validate plus Rust's fixed serializer already cover it. Drop the pre-send in.Validate() in host.Application and the Request.Validate restore sub-checks that Rust repeats exactly. Optionally drop the duplicate Go ValidateFor in FactoryApplicationBuild, since SaveApplicationBuild re-checks the artifact before persisting it.

**Risk.** Error texts for malformed input would now come from Rust instead of Go. Nothing upstream generates malformed specs.

**Safe first step.** In internal/store/applications.go, change scanApplication to return r, nil instead of r, r.Validate(); delete the r.Validate() block in applicationInventory; change ApplicationRevision to return a, nil. In internal/store/application_restore.go, change scanApplicationRestore to return operation, nil. Every write path (insertApplicationRecord, ReplaceApplication, applicationObservationValue, BeginApplicationRestore) already validates before inserting. Then, in internal/host/application.go, delete the applicationResponse type and its UnmarshalJSON and decode straight into application.Observation; Observation.Validate(in.Application) stays. Delete any tests that cover only the removed decoder.

### EXT-5: The extension page/panel allowlist is re-coded in about six places beyond the manifest (one checked twice per request); the Workspace shared-owner plumbing goes with the in-flight Workspace removal

Fountain extension and its backend. duplicated-layer. partially-confirmed. ~110 lines, risk medium. Visible change: none. Behaviour-preserving: yes. **Status:** deferred to repo-first-ui (Workspace removal).

Where: `system/containers/extension/extension.json:7-37`, `cmd/soda-extension/main.go:58-71`, `internal/web/auth/extension.go:25,44-53`, `internal/web/api/extension_native.go:54-55,135-149`, `internal/web/api/extension_terminal_authority.go:42-44,98-104`, `internal/web/api/factory_output.go:53,99-114`, and 5 more.

**Simpler.** Keep the manifest as the only list. Server: auth.ExtensionAuthority keeps only the package binding (extension id is soda or tailnet, and tailnet means the admin page tailnet). extensionProductAuthority stops re-calling auth.ExtensionContribution. extensionProductContribution becomes "tailnet path requires a tailnet authority, every other path requires a soda authority". allowedContribution in soda-extension's main.go either always allows, or is dropped together with the native.contribution.authorize capability. Frontend: validateMountContext keeps the contract-version and session checks and drops isPageMount/isPanelMount; entries drop their page-id guards. nativeTerminalContribution stays as a policy until the Workspace removal (ui/repo-places*) moves terminals to a repository page. Then replace it with a repository-scoped contribution check against authority.Repository.Permission, and delete validateOwner, the ownerId field, the shared-entry handling in checkManifestEntries and the Workspace text in docs.

**Risk.** The dedupe alone is low risk. Removing the Workspace page/panel is a navigation change already in flight on ui/repo-places* (39 commits on installE; a track-2 manifest drops the Workspace page and panel but adds more allowlist copies), so coordinate rather than duplicate. The Workspace UI itself (~7,300 lines in frontend/spaces/sodaspaces-workspace*) is outside this scope.

**Safe first step.** In internal/web/api/extension_native.go, extensionProductAuthority: drop the second auth.ExtensionContribution call (line 54), which auth.ExtensionAuthority already makes at internal/web/auth/extension.go:25. Rewrite extensionProductContribution to take the authority's ExtensionID: tailnet paths require ExtensionID == "tailnet" (as now), other paths require ExtensionID == "soda". Since ExtensionAuthority already restricts soda authorities to exactly the manifest's soda contributions, this changes nothing observable. Then remove extensionPageContribution and extensionWorkspacePanelContribution, and inline the spaces-page/workspace-panel test into nativeTerminalContribution. Coordinate with ui/repo-places-installE, which edits these same lines.

### JDG-5: The legacy string review report, its fences, and the unreachable review-json prompt branch survive as dead code

Completion, reassessment, review and JEV judgments. dead-code. partially-confirmed. ~110 lines, risk low. Visible change: none. Behaviour-preserving: yes. **Status:** coordinate with prod-hardening.

Where: `internal/factory/review_native.go:218-260,283-289,820-827`, `internal/factory/assignment_result.go:19,41-46,158-159,165-177,211`, `internal/factory/control/review_cycle.go:158-160,275-278,858-885`, `internal/factory/control/publication_children.go:254-266`, `internal/factory/dispatch_prompt.go:90,407-412`, `internal/factory/control/dispatch_attempt.go:677`, and 1 more.

**Simpler.** Delete ReviewReport and its Validate, MaxReviewReportBytes, ReviewReportFence (and its entries in the dispatch_prompt.go:90 and dispatch_attempt.go:677 fence guards), ClearReviewReportFence, AssignmentResult.Review with its validation and r.Review==nil guards, the two string-report refusals in review_cycle.go, and the result.Review projection in factory_feed_record.go. Reduce reviewCycleInputDigest to a head/base-only function that still writes the same bytes (v1 domain plus the two trailing NULs). Delete the reviewer else-branch in writePromptReport, which renderPhasedDispatchPrompt always replaces. maintain_library needs no new reminder.

**Risk.** Deleting the fallback branch changes the maintain_library prompt text, so check that a library run still finishes through its tool.

**Safe first step.** In internal/factory/review_native.go, delete ClearReviewReportFence (lines 820-827), which nothing references. In internal/factory/control/review_cycle.go, change reviewCycleInputDigest to take only (head, base) and always write "review-cycle-input-v1\x00"+head+"\x00"+base+"\x00" followed by "\x00\x00", so the output bytes stay the same. Update the four calls in publication_children.go:254-266 and the test call at review_carried_history_test.go:67. Then delete the reviewer else-branch at dispatch_prompt.go:407-412 (renderPhasedDispatchPrompt always replaces that section, so no prompt changes). Remove AssignmentResult.Review and ReviewReport together with their guards in assignment_result.go, review_cycle.go and factory_feed_record.go:329-339.

### PLN-10: Two planning operations share one prompt and contract (only difference is a Supported-alternatives refusal); multi-candidate answers add a refusing JEV planned-choice at Generate

Intent to plan (Product Lead, Architect, research, decomposition). unused-configuration. partially-confirmed. ~110 lines, risk low. Visible change: UI, model tools. Behaviour-preserving: yes. **Status:** owner decision; coordinate with prod-hardening.

Where: `internal/factory/intent_proposal.go:9-21`, `internal/factory/decomposition_options.go:305-309,355-372,399,418,441`, `internal/factory/control/intent_planning.go:118-217`, `internal/factory/control/intent_bound.go:33-44,89`, `internal/factory/control/intent_plan_choice.go:103,124,408`, `internal/web/api/factory_intent_generation.go:56,89-115`, and 3 more.

**Simpler.** Keep one planning operation. Remove IntentPlanningOptions, ValidIntentPlanningOperation's two-value check, the Operation request field and the 'Planning operation' select. Ask the planner for exactly one candidate graph, or show several and let the founder pick. Then ChoosePlannedCandidate collapses to ChooseSingleCandidate, and its JEV planned-choice call and plannedChoiceRubric go away. Keep effectiveDecompositionScope, which the decomposition triplet judgment still uses.

**Risk.** The operation appears in the staged prompt and the output echo; update the decompose contract with it.

**Safe first step.** In internal/factory/control/intent_planning.go, ChoosePlannedCandidate's single-candidate branch (`return ChooseSingleCandidate(in, out)` at ~:176) re-runs AssessSemanticOutput. Build the ChosenIntentPlan directly there from out.Candidates[0], or have ChoosePlannedCandidate delegate before its own assess. Assessment is pure, so outcomes are identical. Everything else here (removing the operation, the select, or the JEV choice) changes the prompt or UI.

### PLN-12: Plan launch re-validates what the same request just built (pass-through admission wrappers)

Intent to plan (Product Lead, Architect, research, decomposition). unneeded-validation-or-digest. confirmed. ~110 lines, risk low. Visible change: none. Behaviour-preserving: yes. **Status:** coordinate with prod-hardening.

Where: `internal/factory/control/intent_bound.go:13-63`, `internal/factory/control/intent_planning.go:13-94,123-176`, `internal/factory/control/dispatch_timeonly.go:44-87`, `internal/factory/control/intent_plan_assembly.go:117-128`, `internal/factory/control/intent_plan_choice.go:118-124,343-350`, `internal/web/api/factory_intent_generation.go:243-256`.

**Simpler.** Inline the two real checks into DispatchIntentPlan: head.Repository>0, and non-empty in.Inspected returning ErrIntentPlanningInvestigate. Keep store.ErrStaleRevision on a head digest mismatch. Delete AdmitIntentBoundLaunch and AdmitIntentPlanning, ErrIntentPlanningExhausted/External/Cancelled and the SemanticPlanner interface. In ChoosePlannedCandidate, adopt the single candidate directly instead of re-assessing through ChooseSingleCandidate. Keep one AssessSemanticOutput at settlement. Stage the prompt once, either by passing the prompt from Assemble to Dispatch or by moving the friendly over-bound message to the dispatch path.

**Risk.** Pure internal re-checks of freshly built values.

**Safe first step.** internal/factory/control/intent_planning.go: delete the SemanticPlanner interface and the unreferenced ErrIntentPlanningExternal, ErrIntentPlanningCancelled and ErrIntentPlanningExhausted. Replace AdmitIntentPlanning and AdmitIntentBoundLaunch (intent_bound.go:21-44) with two inline checks in DispatchIntentPlan (dispatch_timeonly.go:67-72): if head.Repository<=0, return the same "intent planning needs its repository association" error; if len(in.Inspected)==0, return ErrIntentPlanningInvestigate. Keep the existing StageIntentPlanningPrompt call, which already runs in.Validate and the head digest compare.

### PRJ-6: Go host client re-checks requests and echoed responses that the Rust host already validates; only the client-side calls can go, not the shared validators

Project environment runtime on the host (create/join/start, executor, domain, ssh, accounts). unneeded-validation-or-digest. partially-confirmed. ~110 lines, risk low. Visible change: none. Behaviour-preserving: yes. **Status:** open (behaviour-preserving).

Where: `internal/host/factory_client.go:32-180`, `internal/host/lifecycle.go:10-38`, `internal/host/profiles.go:13-23`, `internal/host/client.go:144-198`, `internal/host/factory_context.go:31-54`, `internal/project/takeover.go:45-62`, and 4 more.

**Simpler.** Turn the internal/host client methods into plain calls. Remove the request pre-validation in FactoryLaunch, FactoryTaskEvents, FactoryExport, ReadPreparationContext and ProjectAccess, and remove the response echo and re-validation checks: factoryStateConfirmed, the identity checks in FactoryInspect, FactoryStop, factoryOutputError and factoryExportError (keep their HTTP status to sentinel-error mapping), factoryTaskEventsConfirmed, the out.Validate calls in FactoryHarness and FactoryTakeover, lifecycleOutcomeConfirmed/lifecycleActionConfirmed, ResolveProfile's Validate and GOARCH check, and the result checks in Create, Inspect, Connection and ProjectAccess (validAddress goes once nothing uses it). TakeoverResult.Validate is the only helper this leaves without a caller, so delete it. Keep FactoryPreparationContext.ValidateFor/ValidateContent, FactoryHarnessPin.Validate, Profile.Validate and the request Validate methods, because other code still uses them. Delete any internal/host tests that cover the removed checks.

**Risk.** The Rust host still refuses malformed requests. The only change is that a Go caller bug surfaces as a host 400 instead of a Go error.

**Safe first step.** In internal/host/factory_context.go, remove the in.Validate()/ValidFactoryRole pre-check and the out.ValidateFor(in, role) call from ReadPreparationContext. Every caller (factory/control dispatch_context.go, app_verification.go, delivery_admission.go, verification_execution.go, verification_catalog.go, intent_plan_assembly.go) already runs ValidateFor on the result. In internal/host/factory_client.go, delete factoryStateConfirmed, factoryTaskEventsConfirmed and the identity checks after FactoryInspect, FactoryStop and FactoryTakeover; keep the HTTP status to ErrRunNotFound/ErrRunStale mapping. Then delete project.TakeoverResult.Validate (internal/project/takeover.go:54-62), which then has no callers.

### RUN-8: Stop and Reconcile persist durable command receipts and replay them, although settleRun is already idempotent

Run end: task events, checkpoints, results and settlement (RUN). process-machinery. confirmed. ~110 lines, risk low. Visible change: none. Behaviour-preserving: yes. **Status:** open (behaviour-preserving).

Where: `internal/factory/control/settle.go:15-199`, `internal/factory/types.go:50-117`, `internal/factory/control/coordinator.go:182`, `internal/factory/control/repair_retire.go:39-40`, `internal/factory/control/operator.go:105-145`, `internal/web/api/factory_lifecycle.go:178-185`.

**Simpler.** Make Stop call settleRun(run) and Reconcile call reconcileRuns() directly, then return the receipt. Delete RecordFactoryCommand/FinishFactoryCommand use, replayStop, replayReconcile, and the stop/reconcile cases of settleAbandonedCommand. Also delete CommandDigest and the stop/reconcile branches of Command.Validate, which nothing else uses. In repair_retire, call settleRun when the run is not reconciled. Keep the command table for the other command types. The APIs can keep accepting command_id and ignore it.

**Risk.** A concurrent double stop would run settleRun twice. SaveFactoryRun's settled guard and the broker close are idempotent.

**Safe first step.** Remove the stop/reconcile handling from startup recovery first. In internal/factory/control/settle.go, delete the CommandStop and CommandReconcile cases of settleAbandonedCommand. Because each stop/reconcile then records and finishes in the same call, nothing user-visible changes beyond rows staying unfinished after a crash. Then, in settle.go, replace the record/replay/finish block in Stop and Reconcile with a direct settleRun/reconcileRuns call. Delete replayStop and replayReconcile. In repair_retire.go:39-40, call settleRun directly. Remove CommandDigest and the stop/reconcile Validate cases from internal/factory/types.go, and the Digest construction in operator.go:111,125 and factory_lifecycle.go:179. For every frontend flow (fresh ID per click) the result is identical.

### RUN-6: settleRepairChain re-proves a delivery-repair subject that SodaOS built itself from the failed verification

Run end: task events, checkpoints, results and settlement (RUN). unneeded-validation-or-digest. confirmed. ~105 lines, risk low. Visible change: none. Behaviour-preserving: yes. **Status:** coordinate with prod-hardening.

Where: `internal/factory/control/dispatch_result.go:54-149`, `internal/factory/control/dispatch_result.go:31`, `internal/factory/control/dispatch_result.go:559`, `internal/factory/control/mcp_settle.go:152-158`, `internal/factory/control/mcp_settle.go:54`.

**Simpler.** Delete repairChainSubject, checkRepairChain, settleRepairChain, refuseUnprovenRepair and repairChainStore, and their three call sites. Drop AssignmentByRun from MCPTerminalSettleStore. Admission (cause.Validate and validateDeliveryRepairCauseTx) and bindDeliveryRepairResult already cover these facts.

**Risk.** It only guards against SodaOS corrupting its own subject between admission and finish.

**Safe first step.** In internal/factory/control/dispatch_result.go, delete lines 54-149 (repairChainSubject, checkRepairChain, repairChainStore, settleRepairChain, refuseUnprovenRepair) and the settleRepairChain calls at lines 31-34 and 559-562. In internal/factory/control/mcp_settle.go, delete the comment and the call at lines 152-158 and the AssignmentByRun line (54) in MCPTerminalSettleStore. Then remove the "errors" and project imports only if nothing else in the file still uses them.

### DLV-10: Acceptance has three near-duplicate decision-vs-evidence comparators and a near-identical outcome replay decoder per command; these can be merged

Publication, merge, correction, repair, acceptance, verification, lifecycle. duplicated-layer. partially-confirmed. ~100 lines, risk low. Visible change: none. Behaviour-preserving: yes. **Status:** open (behaviour-preserving).

Where: `internal/factory/control/acceptance_evidence.go:57-145`, `internal/factory/control/acceptance_status.go:75-92`, `internal/factory/control/acceptance_status.go:139-213`, `internal/factory/control/acceptance_status.go:288-307`, `internal/factory/control/acceptance_status.go:413-434`, `internal/factory/control/acceptance.go:217-290`, and 4 more.

**Simpler.** Write one acceptanceReasons(decision, evidence, mode) comparator for the issue, source and edge checks. Admission takes the first reason after its explicit revision check, status takes all reasons, and replay uses the visibility-only mode. Fold checkCodePrereqCurrency into checkPrereqAcceptance with a collect/first switch. Replace the per-command replay decoders with one generic replayOutcome[T] helper that has an optional error-code shim. Keep the per-op AcceptanceStatus read in publicationAuthority and mergeAuthority: it is the only check that notices native edits, and its revision fences the publication, correction and merge observations. Dropping recoverAbandonedWithdrawal and the admission revision equality are separate decisions that change behaviour.

**Risk.** Merging the comparators and the replay helpers preserves behaviour. Dropping the global-revision equality only removes spurious refusals.

**Safe first step.** In internal/factory/control/acceptance_evidence.go, re-implement checkAcceptanceIssue and checkAcceptanceSources by calling assessAcceptanceObjective and assessAcceptanceSources (acceptance_status.go) and refusing with the first reason. Keep the revision check (acceptance_evidence.go:58-60) ahead of them. In acceptance.go, replace the source loop in verifyAcceptanceVisibility with a visibility-only path of the same helper. Then replace replayPause (lifecycle.go:74), replayResume (lifecycle.go:313), replayTakeover (lifecycle_takeover.go:87) and replayRetry (lifecycle_retry.go:252) with one generic replayOutcome[T]. Have replayAcceptance and replayWithdrawal use the same helper with the stale_revision/readiness_capacity shim. Leave publicationAuthority, mergeAuthority, recoverAbandonedWithdrawal and the admission revision equality as they are.

### DOC-11: AGENTS.md three-call JEV ritual, freshness policy repeated near the top of 11 docs, and a duplicated development index with drift-prone counts

Documentation. process-machinery. partially-confirmed. ~100 lines, risk low. Visible change: none. Behaviour-preserving: yes. **Status:** coordinate with prod-hardening.

Where: `AGENTS.md:7-11`, `AGENTS.md:55-56`, `docs/README.md:3`, `docs/README.md:63-87`, `docs/development/README.md:1-87`, `docs/development/release.md:3`, and 9 more.

**Simpler.** For the owner to decide: in AGENTS.md, replace the three-call rule with one well-framed JEV call, asking again only if the answer is surprising. Delete the one-line freshness restatements at the top of the docs other than AGENTS.md, freshness.md and README.md, since AGENTS.md already links freshness.md. Fold docs/development/README.md into the "Develop Soda" table of docs/README.md and delete it. Replace hard-coded counts (39 tasks, 41 pages, 53/12 outcomes, D01–D18) with plain links so they cannot drift. Leave the rest of the doc set to the other DOC findings.

**Risk.** AGENTS.md is the owner's instruction file, so these are proposals for the owner. Do not edit without approval.

**Safe first step.** Merge docs/development/README.md into docs/README.md: add its few unique rows (the RC6 qualification record, the Canlang/handbook pointers, the permissions note) to the "Develop Soda" table, delete docs/development/README.md, and update the links that point to it (docs/README.md row 67 and any rg hits for "development/README.md"). Also delete the one-line freshness banners at docs/development/README.md:3 and docs/development/release.md:3. Documentation only; no code behaviour changes. AGENTS.md edits (the three-call rule) wait for owner approval.

### STO-11: Go store duplicates the broker's credential key check and keeps OpenEncrypted/Open cipher plumbing used only by test fixtures

Store (PostgreSQL layer) structure. duplicated-layer. partially-confirmed. ~100 lines, risk low. Visible change: none. Behaviour-preserving: yes. **Status:** partly done (`8764b259`): Go grant cipher, OpenEncrypted and the keyed fixtures deleted; the remaining test-only entrypoints are open.

Where: `internal/store/store.go:27-30`, `internal/store/store.go:63-133`, `internal/store/grants.go:1-102`, `internal/store/identity_fixture.go:33-52`, `internal/store/ephemeral.go:42-75`, `internal/store/grants_test.go`, and 2 more.

**Simpler.** Keep one plain open path, with no checkKey flag and no checkGrantKey or initializeGrantKey. Keep OpenObserve as it is. Seal fixture credentials with a small helper inside SeedIdentityConnection that takes the key as an argument, instead of carrying a cipher on Store and stamping grant_key_check from Go. Delete OpenEncrypted, openWithKeyCheck, Store.grants and the key-check functions in grants.go, plus the grants_test.go tests that cover them. Optionally, drop the template-clone fallback in OpenEphemeral.

**Risk.** Only integration fixtures change. The broker keeps sole ownership of credential encryption.

**Safe first step.** In internal/store/grants.go, delete checkGrantKey, rejectUnkeyedIdentityCredentials, validateGrantKey, initializeGrantKey and keyBinding. In internal/store/store.go, delete the checkKey parameter and the cipher-driven initializeGrantKey call from configureStore, and merge open, openWithKeyCheck and OpenDashboard into one plain open (Open and OpenDashboard become the same function). Delete the key-check tests in internal/store/grants_test.go. Production dashboard behaviour stays identical, because OpenDashboard already skips all of this.

### IDN-7: Execution records carry a redundant acquisition digest (checked at three sites plus a trigger clause) and a pending-revert retry path nobody takes

Identity service and provider credentials. too-many-states-or-budgets. partially-confirmed. ~95 lines, risk low. Visible change: DB schema. Behaviour-preserving: yes. **Status:** owner decision.

Where: `cmd/soda-identity/src/wire_grants.rs:140-155`, `cmd/soda-identity/src/acquisition.rs:21-104`, `cmd/soda-identity/src/store_leases.rs:113-165`, `cmd/soda-identity/src/store_executions.rs:1-65`, `cmd/soda-identity/src/wire_execution.rs:141-171`, `cmd/soda-identity/src/wire_execution.rs:245-268`, and 4 more.

**Simpler.** Keep the execution row as a fence (kind, execution_id, state, lease_id, binding). Also keep the trigger's immutable-identity and terminal-stays-terminal clauses. Delete acquisition_digest and the Execution.digest field in Rust and Go. Remove the three digest comparisons and the digest predicate in the reserve UPDATE, and remove the digest clause from soda_guard_execution_identity. Make reconcile_lease call end() for unbound leases too (which tombstones the execution), and delete release_execution_lease's revert to pending. Optionally fold finish_lease and end into one function with a capture flag. Leave the reconcile, sweep and retire_connection scans alone, since each has its own scope.

**Risk.** Fencing between CloseExecution and acquire/register must stay; only the digest, trigger and unused retry semantics go. Dropping the trigger is a schema change; it can also be left in place harmlessly.

**Safe first step.** This step needs no schema change, since the trigger only compares the old and new JSON digest. In cmd/soda-identity: - Delete acquisition_digest from wire_grants.rs and its re-export in wire.rs. - Delete the digest field from Execution in wire_execution.rs, and the `state != terminal && digest.is_empty()` check in Execution::validate. - In acquisition.rs, drop the digest parameter and the `existing.digest != digest` comparisons in admit_execution and replay_acquisition. - In store_leases.rs, drop the `current.digest != expected.digest` check and the `AND data->>'digest'=$7` predicate from the reserve_and_link_execution UPDATE. - In internal/identity/types.go, drop the Digest field and its Validate check. Execution ids are unique per run or launch, so no observable outcome changes.

### DLV-11: VerifyChecks' adopted-definitions branch can never fire, and the clear-review cycle and child-check digest comparisons repeat each other

Publication, merge, correction, repair, acceptance, verification, lifecycle. unneeded-validation-or-digest. partially-confirmed. ~90 lines, risk low. Visible change: none. Behaviour-preserving: yes. **Status:** open (behaviour-preserving).

Where: `internal/factory/checks.go:59,70,104-105,152-185`, `internal/factory/checks_assessment.go:124,153-187`, `internal/factory/control/checks_pass.go:113-116,131`, `internal/factory/control/merge_evidence.go:93-95`, `internal/factory/control/delivery_acceptance.go:59-64,76-108`, `internal/factory/control/delivery_producer.go:22-25`, and 2 more.

**Simpler.** Drop AdoptedChecks. VerifyChecks should take the policy once and fill Checks, PolicyRevision and ChecksDigest from it. Delete CheckReasonDefinitionsChanged with its guidance text and allowlist entries. In AdmitClearReview, delete recordedClearCycle and replace recordedClearCycleMatches with a single check that cycle.PublicationID == published.ID. In currentChildCheckAssessment, drop the redundant ChecksDigest comparison next to slices.Equal. Keep the fresh native check re-score before submit and the union head-currency gate. checkUnionAcceptanceCapture could shrink to ID plus tuple, but that is optional.

**Risk.** Every removed check either cannot fail or duplicates a gate that runs later.

**Safe first step.** In internal/factory/checks.go, delete the AdoptedChecks type and its Validate. In internal/factory/checks_assessment.go, change VerifyChecks to take only (target, current policy, observed, now): fill Checks, PolicyRevision and ChecksDigest from the policy, delete the definitions_changed branch, and remove CheckReasonDefinitionsChanged from the constant, the guidance switch and the refused-reason allowlist. Update the two callers (internal/factory/control/checks_pass.go:113-116,131 and internal/factory/control/merge_evidence.go:94-95) and delete the tests covering AdoptedChecks or definitions_changed. Also delete recordedClearCycle from internal/factory/control/delivery_acceptance.go and reduce recordedClearCycleMatches to the PublicationID check.

### FEP-10: Seven hand-rolled request/JSON helpers duplicate sodaspaces-api.ts, with differing error policies

Frontend: Workspace, project, applications, terminals, layout, Forgejo and Tailnet pages. duplicated-layer. partially-confirmed. ~90 lines, risk low. Visible change: none. Behaviour-preserving: yes. **Status:** open (behaviour-preserving).

Where: `frontend/spaces/sodaspaces-workspace-inventory.ts:12-85`, `frontend/spaces/sodaspaces-project-request.ts:34-93`, `frontend/spaces/sodaspaces-terminal-actions.ts:85-110`, `frontend/spaces/soda-identity.ts:88-112`, `frontend/spaces/sodaspaces-project-members.ts:96-110`, `frontend/tailnet/soda-tailnet-observation.ts:51-80`, and 2 more.

**Simpler.** Add one sodaRequest(transport, path, {body, signal}) to sodaspaces-api.ts. It builds the init, reads the JSON and throws SodaRequestError(status, readSodaErrorCode). Callers keep their own 401/403 reaction and timeouts. Replace the copies in inventory, project-request, terminal-actions and project-members, and drop the requestApplication pass-through. Either keep the identity and tailnet differences (status-only messages, body cancel, no-referrer) as small wrappers, or align them on purpose. Decide whether Project Access keeps showing server messages.

**Risk.** Keep each caller's 401/403 handling. The only visible decision is whether project-members keeps showing server messages.

**Safe first step.** In frontend/spaces/sodaspaces-workspace-inventory.ts, delete errorCode (lines 52-60) and call readSodaErrorCode from sodaspaces-api.ts; it is identical. Then add a shared JSON RequestInit builder to sodaspaces-api.ts. Use it in place of sodaWorkspaceInit (inventory), sodaFetchInit+requestHeaders (project-request) and the inline init in terminal-actions json(), all of which produce the same init for the same inputs. Leave sodaErrorCode in project-request for now, because it accepts codes that do not match the regex. Leave the error handling in project-members, identity and tailnet unchanged.

### JDG-8: Reassessment and planning readers rebuild the full judged state just to find their own receipt

Completion, reassessment, review and JEV judgments. duplicated-layer. partially-confirmed. ~80 lines, risk low. Visible change: none. Behaviour-preserving: yes. **Status:** open (behaviour-preserving).

Where: `internal/web/api/factory_journey_readback.go:581-617`, `internal/factory/control/completion_decision.go:53-140,233-320`, `internal/factory/control/completion_drive.go:625-641`, `internal/factory/control/planning.go:243-259`, `internal/factory/judgment.go:70-82`, `internal/store/factory_reassessments.go:36-42`.

**Simpler.** When DecideReassessment records a receipt, save its fingerprint on the ReassessmentRecord (the way ProposalFingerprint is already saved), or save the answer itself. The journey, checkCompletionRetry and recoverReassessmentDecision then read that field instead of calling DecisionStateFromRecord plus fingerprintOf. planningCallFailed can be replaced by AdvisePurpose returning a distinct error for a recorded failure versus a refused reservation. Keep the delivery merge-admission rebuild: it is a deliberate check that the evidence has not changed. Keep fingerprint-keyed reservation unless funding is redesigned separately.

**Risk.** Readers switch to fields on records they already load. Replay-by-fingerprint deduplication goes away, which is fine once JDG-2 lands.

**Safe first step.** Add a DecisionFingerprint field next to ProposalFingerprint on store.ReassessmentRecord (it is JSON-encoded, so no migration) and set it where DecideReassessment's reserve callback receives the fingerprint. Then in factory_journey_readback.go:592, completion_drive.go:626-630 and completion_decision.go:242, read record.DecisionFingerprint instead of calling DecisionStateFromRecord/DecisionFingerprintFromRecord plus fingerprintOf. DecisionFingerprintFromRecord can then be deleted.

### PRJ-9: Go re-implements the Rust project-maintenance creation lock; both check directory owner and mode for no benefit

Project environment runtime on the host (create/join/start, executor, domain, ssh, accounts). duplicated-layer. partially-confirmed. ~70 lines, risk medium. Visible change: UI. Behaviour-preserving: yes. **Status:** owner decision.

Where: `internal/host/project_admission.go:1-69`, `lib/host/src/project/admission.rs:23-49`, `internal/web/api/environments_create.go:214-227`, `internal/web/api/api.go:31-33,61`, `lib/host/src/dbackend/backend.rs:191`, `lib/host/src/dbackend/mod.rs:31`.

**Simpler.** Drop the uid and mode checks in both lock functions and keep only the O_DIRECTORY open plus flock. To delete the Go copy as well, first make the Rust /create route return a 503 project_maintenance or project_admission_unavailable instead of mapping it through `internal`, and make Go relay that code. Accept that a reservation row will exist before the refusal, and that maintenance can then start between reservation and provisioning. Otherwise keep the Go gate. The simpler route is to keep it, with the extra checks removed.

**Risk.** A create attempted during maintenance would reserve a DB row before failing. The existing create-failure path handles that.

**Safe first step.** In lib/host/src/project/admission.rs lock(), delete the `info.uid() != owner || info.mode() & 0o7777 != 0o755` condition and the now-unused owner parameter (keep is_dir via O_DIRECTORY). In internal/host/project_admission.go, delete the `info.Uid != owner || ... 0o755` Fstat check and the owner parameter. On a host set up by tmpfiles (root:root 0755) the outcome is identical. Keep the Go gate itself until the Rust create route returns a typed project_maintenance 503.

### TRM-7: Dead Python int() emulation loop in guest argv parsing, duplicated Cc/Cf tables in lib/host, and stale '.py'/brief parity comments in the terminal guest

Human terminal stack end to end (browser, Go API, host, in-project helper). dead-code. partially-confirmed. ~70 lines, risk low. Visible change: none. Behaviour-preserving: yes. **Status:** partly done (`d0435632`): duplicate Cc/Cf table deleted; the Python int() loop is open.

Where: `cmd/soda-project-terminal/src/proto.rs:157-185`, `cmd/soda-project-terminal/src/proto.rs:107-139`, `lib/host/src/terminal/text.rs:3-46`, `lib/host/src/domain/mod.rs:24-52`, `cmd/soda-project-terminal/src/main.rs:99-160`, `cmd/soda-project-terminal/src/pty.rs:1-12,57-90`, and 1 more.

**Simpler.** Delete the whitespace and '+' re-check loop in parse_control_argv; the plain str::parse calls already decide the result. In lib/host/src/terminal/text.rs, remove GO_CC/GO_CF/in_ranges and reuse the identical table in lib/host/src/domain (make it pub(crate)). Rewrite or delete the '.py'/PR25/brief parity comments. Keep the SIGALRM timeouts and keep matching Go's ValidTerminalName, which is still live in internal/host/terminal.go. Optionally switch the terminal ready/output/closed/metadata lines to serde_json::to_vec. pyemit itself stays for the keys, broker, subscription and factory_roles paths.

**Risk.** pyemit and StateValue are also used by the subscription, keys and factory paths, so the terminal callers can switch first and the shared module can shrink later.

**Safe first step.** In cmd/soda-project-terminal/src/proto.rs, delete the `for (raw, parsed) in [...] { ... }` block (lines ~162-180) in parse_control_argv, which can never return None once the four parse() calls succeed. In lib/host/src/terminal/text.rs, delete GO_CC, GO_CF and in_ranges and call the identical domain::{GO_CC, GO_CF, in_ranges} (made pub(crate)) from valid_terminal_name. Remove the '.py'-parity wording from the comments in main.rs, pty.rs and term_status.rs.

### APP-9: application_revisions history is read only by tests and the restore freshness check; every current read joins it to get the spec

Application runtime: build, home, snapshots, backups, restore, factory delivery. speculative-abstraction. partially-confirmed. ~60 lines, risk low. Visible change: DB schema. Behaviour-preserving: yes. **Status:** owner decision.

Where: `internal/store/applications.go:45-76,127-142,150-174,205-260`, `internal/store/application_factory_delivery.go:116`, `internal/store/application_restore.go:60-67,114-119`, `internal/store/schema.go:24,31,249-250,353,443-444`, `cmd/soda-identity/src/schema.rs:19,26,282-283,320,381-385`, `internal/store/application_schema_test.go:17-21`, and 1 more.

**Simpler.** Add a spec JSONB column on applications that holds the current spec. Drop application_revisions with its FK, triggers and verify probes, in both the Go schema and the Rust mirror. Turn the seven joins into single-table reads and delete ApplicationRevision along with its test uses. Do this after APP-3 removes or rewrites the restore 'data already admitted' check, which is the only production read of historical revisions.

**Risk.** Needs a schema change, which is cheap with no users. The image digest of the previous revision is no longer kept, and nothing reads it.

**Safe first step.** Delete Store.ApplicationRevision (internal/store/applications.go:127-142), plus the test assertions that call it in internal/store/applications_test.go (around lines 77 and 208) and internal/web/api/applications_test.go (around lines 233 and 428). No production code calls it. Leave the table drop until APP-3 handles the restore freshness check.

### GOM-11: soda-dashboard duplicates its extension and factory-MCP Unix-socket listener and its serve goroutine; operator_other.go is a dead non-Linux stub

Other Go packages (forgejo, acceptance, project, host, tailnet, application, config, strictjson, filelock, avatar, cmd/soda-dashboard, cmd/soda-tailnet, cmd/soda-extension). duplicated-layer. partially-confirmed. ~60 lines, risk low. Visible change: none. Behaviour-preserving: yes. **Status:** open (behaviour-preserving).

Where: `cmd/soda-dashboard/extension.go:13-53`, `cmd/soda-dashboard/factory_mcp.go:30-82`, `cmd/soda-dashboard/main.go:105-113`, `cmd/soda-dashboard/main.go:146-154`, `cmd/soda-dashboard/operator_other.go`, `cmd/soda-dashboard/operator.go:19-55`.

**Simpler.** Write one listenUnix(socket, mode, mkdir) that does the reclaim-only-stale-socket logic and returns the listener plus the 10s/60s/16384 server. Use it for the extension and factory-MCP sockets and delete factoryMCPSocketLive. Leave operatorHTTPServer's unconditional unlink, peer gate and timeouts alone, or make that a separate decision because it changes behaviour. Merge the three serve goroutines into one serve(name, server, listener, stop, errs). Delete operator_other.go and the linux build tag on operator_linux.go.

**Risk.** Keep each socket's mode as it is.

**Safe first step.** Delete cmd/soda-dashboard/operator_other.go and drop the `//go:build linux` line from operator_linux.go. Then make extension.go hold a single unixSocketLive and listenUnix(socket, mode, mkdir, label) used by extensionHTTPServer and factoryMCPHTTPServer, keeping 0662 for extension and 0666 plus MkdirAll(0755) for factory MCP. Delete factoryMCPSocketLive and the body of factoryMCPListener, and rename the calls in extension_test.go. Leave operatorHTTPServer as it is.

### HPL-11: soda-project-maintain keeps an unreachable pinned-digest Muse staging mode beside the provider-store mode the service actually runs

Rust host platform: daemon, terminals, project, application, tailnet (lib/host non-factory, soda-project-maintain, forgejo/pg/console helpers, soda-tailnet, unix-http, wire-time). duplicated-layer. partially-confirmed. ~60 lines, risk low. Visible change: none. Behaviour-preserving: yes. **Status:** open (behaviour-preserving).

Where: `cmd/soda-project-maintain/src/main.rs:83-88,95-118`, `cmd/soda-project-maintain/src/filesystem.rs:70-85`, `cmd/soda-project-maintain/src/options.rs:7,93-135`, `cmd/soda-project-maintain/README.md:80-100`, `docs/public/50-Operate/60-project-maintenance.md:40`, `docs/reference/project-os.md:213`, and 2 more.

**Simpler.** Make provider-store staging the only `muse` mode. Delete load_tools and the muse_sha256-gated pinned branch in main.rs, and accept (or drop) the --provider-store flag so it becomes the default. Fold the --bind-only no-op into a single service line. Update the README and the operator docs that show `muse --project ID`. Keep verify_native, which is shared with canlang, terminal-support and release. Leave the soda-muse receipt gate alone: it is the provider-store freshness check, not a duplicate path. Its no-receipt grandfather branch can be reconsidered separately.

**Risk.** The receipt re-validation lives partly in the in-project launcher (cmd/soda-muse or soda-project-terminal factory code, other scopes).

**Safe first step.** In cmd/soda-project-maintain/src/main.rs, remove the `else { load_tools(&o.tools, &c.muse_sha256)? }` branch and the `c.muse_sha256.is_empty() && !o.bind_only && !o.provider_store` gate, and always use load_wrappers plus provider_stage::resolve_native_staging. Delete load_tools from filesystem.rs:80-85 and fix the comment at :70-71. Keep verify_native. Remove the "default pinned-directory path is unchanged until the provider store rollout completes" sentence from the README, and change the `muse --project` examples in docs/public/50-Operate/60-project-maintenance.md and docs/reference/project-os.md. On installed hosts muse_sha256 is always empty, so the pinned path can only produce "muse project runtime is disabled". Nothing observable changes except that error text, for a hand-run command.

### PTH-6: Three separate writers of /etc/ssh/authorized_keys/<login> with different trim/validation rules

In-project helpers (soda-project-terminal) and the host Muse launch runtime. duplicated-layer. partially-confirmed. ~60 lines, risk medium. Visible change: UI. Behaviour-preserving: no. **Status:** owner decision.

Where: `cmd/soda-project-terminal/src/project_account.rs:740-756`, `cmd/soda-project-terminal/src/project_account.rs:580-595`, `cmd/soda-project-terminal/src/project_member_access.rs:109-138`, `cmd/soda-project-terminal/src/project_member_access.rs:330`, `cmd/soda-project-terminal/src/project_member_access.rs:445-465`, `cmd/soda-project-terminal/src/keys.rs:109-175`.

**Simpler.** Use member_access::atomic (or keys.rs replace) as the one key-file writer, with one canonicalisation (canonical_lines). Provisioning and rejoin call it. Keep the keys route's sha256 revision compare, because the host API and UI depend on it. Provisioning either writes the requested keys (owner decision) or keeps its refuse-on-difference compare in front of the shared writer. key_request.rs, key_lines.rs and factory_roles/accounts.rs stay.

**Risk.** The 'refuse if different' behaviour on provisioning disappears, so join would overwrite keys changed by hand in the project. Confirm that is what the owner wants.

### BND-5: Reassessment inventory re-folds persisted bodies to re-check its own sealed digest (tamper check on SodaOS's own DB), plus a running chain that a single seal-time hash would replace

Boundary machinery: strict JSON, digests, receipts, revision checks (cross-cutting). unneeded-validation-or-digest. confirmed. ~55 lines, risk low. Visible change: none. Behaviour-preserving: yes. **Status:** open (behaviour-preserving).

Where: `internal/factory/reassessment.go:26-37`, `internal/factory/control/completion_inventory.go:95`, `internal/factory/control/completion_inventory.go:168-194`, `internal/factory/control/completion_inventory.go:332`, `internal/factory/control/completion_inventory.go:455-462`, `internal/factory/control/completion_decision.go:86-92`, and 2 more.

**Simpler.** Delete resealInventoryDigest and its comparison in completion_decision.go, and trust the persisted sealed record. Then replace the running chain with one sha256 over repository, generation, revision and the sorted canonical bodies, computed at seal time. That removes Running, foldInventoryPage and the sealInventoryDigest wrapper. Keep the digest as the identity key it already is. Adjust or drop the legacy empty-set compatibility check in factory_reassessments.go:123-129, which re-derives the empty digest.

**Risk.** The digest is also embedded in JEV request records. It can keep being computed once at seal time.

**Safe first step.** In internal/factory/control/completion_decision.go, delete the reseal block at lines 86-92 (the comment, the resealInventoryDigest call and the "differ from their sealed digest" refusal) and the resealInventoryDigest function at lines 111-140. Delete any _test.go cases that cover only the reseal or tamper refusal. Leave foldInventoryPage, sealInventoryDigest and ReassessmentInventoryDigest untouched, so digest values and every downstream comparison stay identical.

### PRP-9: Code prerequisites re-check the endpoint's acceptance one level deep on every pass, after the endpoint's own assessment already did

Project preparation pipeline below the API (standing and candidate preparations, decisions, hold, host and root helper). speculative-abstraction. partially-confirmed. ~55 lines, risk medium. Visible change: UI. Behaviour-preserving: no. **Status:** owner decision.

Where: `internal/factory/control/prerequisites.go:112-129`, `internal/factory/control/prerequisites.go:132-188`, `internal/factory/control/prerequisites.go:256-282`, `internal/factory/control/readiness.go:294-306`.

**Simpler.** Keep the endpoint visibility and closure read, and keep the stale-head check (head differs from PrereqAcceptance means acceptance_invalid). Replace the AcceptanceWithdrawn check plus endpointAcceptanceValid with one rule: a code prerequisite is satisfied when IssueMergeCompletion matches the bound acceptance and its original closing occurrence; otherwise it is pending. Read the endpoint issue only, without bracketing its selected source revisions. Keep result prerequisites: the acceptance UI offers them.

**Risk.** Blocker codes (acceptance_invalid vs code_pending) feed fingerprints and Feed and UI resolutions, so merging them changes stored fingerprints and texts. I did not trace every reader of the factory.Blocker codes.

### GOM-9: Setup-claim refusals are reshaped four times between Rust body text and the API code

Other Go packages (forgejo, acceptance, project, host, tailnet, application, config, strictjson, filelock, avatar, cmd/soda-dashboard, cmd/soda-tailnet, cmd/soda-extension). duplicated-layer. confirmed. ~50 lines, risk low. Visible change: none. Behaviour-preserving: yes. **Status:** open (behaviour-preserving).

Where: `internal/host/client.go:17-45,63-84,120-141`, `internal/host/application_setup.go:11-38`, `internal/web/api/application_setup.go:13-34,211-221`, `lib/host/src/dbackend/backend.rs:96-109 (related: Rust maps the setup sentences onto factory ExportStale/OutputStale variants)`.

**Simpler.** Export sentinels such as host.ErrSetupCommitted, ErrSetupConflict and ErrSetupUnconfirmed, and switch with errors.Is in apiApplicationSetupAction. Delete mapSetupClaimError and the ';' prefix parser in setupClaimRefusal. Give callLimit a reader argument, or check the status inside ApplicationSetup, and delete callSetupLimit. A separate, optional step: give the Rust setup route its own wire codes instead of reusing the factory 'incarnation changed' bodies. That would require editing the gmux_smoke tests.

**Risk.** Purely internal. The API codes and messages stay the same.

**Safe first step.** 1. In internal/host/client.go, delete callSetupLimit (lines 122-141). Add a reader parameter to callLimit, or have ApplicationSetup call a shared helper that passes readNativeSetupResponseLimit. 2. In internal/host/application_setup.go, delete mapSetupClaimError (lines 11-30) and return the setupClaimError directly. Its Error() already produces the same sentences, so setupClaimRefusal behaves identically. Then, optionally, export three sentinel errors and replace the string-prefix switch in internal/web/api/application_setup.go setupClaimRefusal with errors.Is, keeping the same code, name and message triples.

### TRM-5: Terminal API re-authorizes up to four times per request or WS open, and has dead peer fields and a triple-checking inventory path

Human terminal stack end to end (browser, Go API, host, in-project helper). unneeded-validation-or-digest. partially-confirmed. ~50 lines, risk low. Visible change: none. Behaviour-preserving: yes. **Status:** open (behaviour-preserving).

Where: `internal/web/api/extension_terminal_authority.go:54-120`, `internal/web/api/extension_terminal.go:50-93`, `internal/web/api/extension_terminal_stream.go:61-111,133-137`, `internal/web/api/terminal_registry.go:212-241`, `internal/web/api/api.go:44-54`, `internal/web/api/factory_output.go:213-217`, and 2 more.

**Simpler.** Authorize once with extensionTerminalAccount per HTTP request or WS open, and keep the 15 s heartbeat re-check. - Delete the pre-op extensionTerminalCurrent in extensionTerminalOperation; Account just ran. - Delete either the claim-time check or the first check in extensionOpenHostTerminal; they are back to back. - Optionally drop the post-op and post-create re-checks. - Delete TerminalPeer.Token and ContextID. - Collapse extensionTerminalStates' duplicate Project/MemberLogin reads, since Current already re-reads both. - Optionally drop peerIDInUse and rely on the guest writer flock. - Keep the peer map, TerminalStopping and terminalWG: lifecycle stop/pending and CloseTerminals use them, and factory output streams share the registry. - Keep or deliberately reconsider the 128 cap, since it also bounds factory output streams.

**Risk.** Revocation of a long-lived stream still happens within 15 seconds through the heartbeat. Lifecycle stop still cancels peers.

**Safe first step.** Three steps, all with identical observable behaviour: 1. In internal/web/api/api.go, change TerminalPeer to `Project, ID string; Cancel context.CancelFunc`, dropping Token and ContextID. Remove `ContextID: ...` from the TerminalPeer literals in extension_terminal.go:61 and extension_terminal_stream.go:70, and from internal/web/mutation_admission_test.go:51. 2. In extension_terminal.go, delete the pre-op `if !s.extensionTerminalCurrent(ctx, r, identity) {...}` (lines 66-68). 3. In extension_terminal_stream.go, delete the first check in extensionOpenHostTerminal (lines 95-97). extensionClaimTerminal re-verified the same thing immediately before.

### TRM-8: Terminal rows echo environment/repository/user/login that the browser only compares with its own binding and then discards; the TS layer also re-validates server-produced fields

Human terminal stack end to end (browser, Go API, host, in-project helper). unneeded-validation-or-digest. partially-confirmed. ~50 lines, risk low. Visible change: none. Behaviour-preserving: yes. **Status:** open (behaviour-preserving).

Where: `internal/web/api/terminal_registry.go:38-48`, `internal/web/api/extension_terminal.go:95-97`, `internal/web/api/spaces_inspection.go:62`, `frontend/spaces/sodaspaces-terminal-response.ts:4-65`, `frontend/spaces/sodaspaces-inventory-response.ts:29-60`, `frontend/spaces/sodaspaces-terminal-attachment.ts:178-189`, and 2 more.

**Simpler.** Have the API return host.TerminalState directly ({id, name, created_at, ready, attached, state}), deleting TerminalView and both constructors. In TS, drop admitTerminalBinding and the binding-copied fields from TerminalMetadata, keep a minimal shape guard (id format, known state), and stop threading identity into terminalMetadata/admitSpaceTerminal. Optionally drop repository_id from the WS handshake. Keep session_generation: it is the WebSocket replacement for the session-generation header that the other terminal routes enforce, and it is what refuses stale tabs after a re-login.

**Risk.** Pure echo removal.

**Safe first step.** In frontend/spaces/sodaspaces-terminal-response.ts, delete admitTerminalBinding (replace it with a terminalID check) and remove environment_id/repository_id/user_id/login from TerminalMetadata and terminalMetadata. Simplify the TerminalIdentity parameter threading in sodaspaces-inventory-response.ts (admitSpaceTerminal/spaceTerminals) and sodaspaces-workspace-terminal-hosts.ts accordingly. Then in internal/web/api/terminal_registry.go, delete the TerminalView struct and terminalDTO. Delete extensionTerminalView in extension_terminal.go and return host.TerminalState in spaces.go/spaces_inspection.go and extension_terminal.go:159. Update the frontend test fixtures that assert the echo fields. Leave the handshake session_generation check in place.

### FEP-8: Dead can_embed field left over from the removed in-Forge app frame

Frontend: Workspace, project, applications, terminals, layout, Forgejo and Tailnet pages. dead-code. partially-confirmed. ~45 lines, risk low. Visible change: none. Behaviour-preserving: yes. **Status:** done (`3e6b5b31`).

Where: `internal/web/api/applications.go:13,24,47,59,76,81,140,166-181`, `go.mod:41`, `frontend/spaces/sodaspaces-application-response.ts:45,178,185,262,277,299,309,338`, `tests/fixtures/founder-demo/verify-reference.ts:84`, `tests/frontend/application-controls.test.ts:844,988,2945-2962,3034,3872,4008,4104,4193,4296,4408`.

**Simpler.** Remove can_embed from the TS ApplicationView/ApplicationLaunch types, the key allowlists and the parsers, including the required-boolean check on launch records, and from the test fixtures. Then delete CanEmbed from both Go views, the canEmbed parameter of appLaunchView, applicationEmbeddable, and the publicsuffix import, and tidy go.mod. Keep the applicationOwner mount-and-hide logic: it preserves the app panel's drafts and in-flight setup, backup and restore state.

**Risk.** Order matters, because the TS launch parser currently rejects records without can_embed. Unmounting App when hidden re-fetches /api/applications on each visit, which is acceptable.

**Safe first step.** In frontend/spaces/sodaspaces-application-response.ts: drop can_embed from both interfaces (lines 45, 277), from both key allowlists (178, 299), from the optional check (185), from the required check (309) and from both spreads (262, 338). Drop the can_embed fixture fields in tests/frontend/application-controls.test.ts and tests/fixtures/founder-demo/verify-reference.ts. In the same change, in internal/web/api/applications.go: delete the CanEmbed fields (24, 47), the canEmbed parameter and assignment in appLaunchView (59, 76) and its caller (140), the line view.CanEmbed = ... (81), applicationEmbeddable (166-181) and the publicsuffix import, then go mod tidy.

### PRP-8: Setup/inputs digests re-checked about five times per standing preparation start, after admitPreparationStart already bound them to the stored decision

Project preparation pipeline below the API (standing and candidate preparations, decisions, hold, host and root helper). unneeded-validation-or-digest. partially-confirmed. ~40 lines, risk low. Visible change: none. Behaviour-preserving: yes. **Status:** open (behaviour-preserving).

Where: `internal/web/api/preparation_source.go:237,260,270-273`, `internal/project/preparation.go:289-300`, `internal/factory/control/preparation_start.go:28-35`, `internal/store/factory_candidate_preparations.go:582-605`, `internal/store/preparation.go:225`, `internal/store/project_grants.go:56-61,90-95`, and 2 more.

**Simpler.** Keep two checks: the one comparison in admitPreparationStart (the fresh repository read against the stored requirement decision), and the root helper's digest check where privileges change. Remove these repeats: the inputs-digest recomputation in StartPreparation, the setup-digest equality in Prepare.Validate (Go) and in Prepare::validate (Rust host), and the copied-field SetupDigest comparison in the standing path. As a separate refactor, replace the count(*) depth comparison with a current-head-ID check in candidate admission, then drop Revision from the refs. Keep EffectsDigest, because completion source and material consume it.

**Risk.** Keeping the helper's check preserves the one check at a privilege change.

**Safe first step.** Delete lines 28-35 of internal/factory/control/preparation_start.go (the setupFiles loop and the PreparationInputsDigestOf comparison). StartPreparation's only caller, startPreparationItems in internal/web/api/preparation_source.go, passes a draft whose InputsDigest admitPreparationStart already compared to requirement.InputsDigest. Also delete the SetupDigest equality in project.Prepare.Validate (internal/project/preparation.go:296-298) and in Rust Prepare::validate (lib/host/src/preparation/mod.rs:203-205). Leave admitPreparationStart, the authorize re-read and the helper's check in inputs.rs and verification.rs untouched.

### RUN-5: Settlement tail duplicated between finishFromRun and AccountSettledRun, plus a redundant second MCP settle in settleRun; FinishAssignment guards are partly needed because retries rewrite assigned rows

Run end: task events, checkpoints, results and settlement (RUN). duplicated-layer. partially-confirmed. ~40 lines, risk medium. Visible change: none. Behaviour-preserving: yes. **Status:** coordinate with prod-hardening.

Where: `internal/factory/control/dispatch_result.go:14-52`, `internal/factory/control/dispatch_result.go:517-571`, `internal/factory/control/settle.go:290-299`, `internal/factory/control/mcp_settle.go:155-178`, `internal/store/factory_assignments.go:92-184`, `internal/store/factory_retry_packet.go:140-150`.

**Simpler.** Have AccountSettledRun do accounting (usage, reservation) and then delegate to one shared finish helper, the body of finishFromRun, so the settleRecordedMCPBeforeRun -> finishAssignmentFromRun -> settleRepairChain -> FinishAssignment sequence exists once. Drop the extra SettleMCPTerminalProposal call in settleRun, because the progress drive already recovers it. Keep the revision/stage CAS in FinishAssignment. Retry packets rewrite run/revision/data on assigned rows. After that is confirmed, sameAssignmentFinishInput can be dropped in favour of the revision guard, and a distinct already-finished error from the locked stage check could replace the mcp_settle re-read.

**Risk.** Keep the attempt-allowance transition and readiness enqueue in the same transaction. They are the real side effects.

**Safe first step.** In internal/factory/control/dispatch_result.go, replace AccountSettledRun's tail (from the settleRecordedMCPBeforeRun call through FinishAssignment, about lines 548-570) with a call to a shared helper taken from finishFromRun's body (settleRecordedMCPBeforeRun, finishAssignmentFromRun, settleRepairChain, FinishAssignment). Keep AccountSettledRun's return mapping: ErrNotFound means accounted=true, and an MCP-handled finish means re-read and report the Finished stage. Do not touch FinishAssignment's revision/stage WHERE clause.

### TRM-4: The host agent binary is hashed twice per websocket terminal op (support check plus an attach argv the guest only reads on reserve), and the guest hashes again on reserve

Human terminal stack end to end (browser, Go API, host, in-project helper). unneeded-validation-or-digest. partially-confirmed. ~40 lines, risk low. Visible change: none. Behaviour-preserving: yes. **Status:** open (behaviour-preserving).

Where: `lib/host/src/dbackend/websocket.rs:101-122`, `lib/host/src/terminal/native.rs:110-145,250-272`, `lib/host/src/terminal/support.rs:67-75,118-141`, `lib/host/src/terminal/identity.rs:166-191`, `cmd/soda-project-terminal/src/main.rs:136-146`, `cmd/soda-project-terminal/src/term.rs:105-135`, and 3 more.

**Simpler.** In admit_terminal_session, compute agent_program_hash once and pass it to both the support check (support_against) and NativeAttach::attach, instead of hashing twice. Optionally send source_hash only for the reserve action. Keep the guest verify_program, or replace it with a host support_in_container call in prepare_identity, because the broker/Codex reserve path has no other version check. Caching the hash at daemon start or caching per container is optional and depends on how the binary gets replaced.

**Risk.** After an in-place binary swap without a restart, the stale hash is caught at the next restart. That is acceptable for local use.

**Safe first step.** In lib/host/src/dbackend/websocket.rs admit_terminal_session, call terminal::agent_program_hash(0) once. Pass the result to a public variant of support_against (lib/host/src/terminal/support.rs) and to NativeAttach::attach, which then takes the hash as a parameter (lib/host/src/terminal/native.rs:259 drops its own agent_program_hash call). This removes one 32MB read and hash per terminal op, and every outcome stays the same.

### CTX-6: factory.context shows the model unused allowance/run/binding/intent fields (but the gates and availability machinery are needed)

Worker context assembly, project library and research. too-many-states-or-budgets. partially-confirmed. ~35 lines, risk low. Visible change: model tools. Behaviour-preserving: no. **Status:** owner decision.

Where: `internal/factory/context_current.go:12-64`, `internal/store/factory_context_current.go:56-75,139-161`, `internal/web/api/factory_mcp_context.go:72-76`, `internal/web/api/factory_mcp_bound.go:101`, `internal/web/api/factory_mcp_plan_schema.go:17`, `internal/factory/dispatch_prompt.go:482`.

**Simpler.** Keep CurrentContext as the internal precondition struct that the gates read (three of the four gates need its Role anyway). Hide Allowance, Run, the Intent digests and the Binding digests/ids from the model, either with json:"-" or by marshalling only {version, stage, evidence, role} plus at most one 'active time left' line. Drop Run.ObservedState (the not_wired/unavailable/confirmed fill) and the unused Intent fields. Keep Evidence.Availability/MissingReasons, which are the real tool gate, and keep the role-page digests (payload_hash, plan digests) that tools and coverage use. Land this after prod-hardening removes the current.binding.attempt_root references, and reword the reassessment_scope description so it no longer points the model at "remaining allowance".

**Risk.** Internal gates need a direct read instead of the projection. The model loses fields no prompt tells it to use.

### DSP-11: Two TypeSafe clients repeat the SystemOne request code, model default and choice-distribution checks

Factory dispatch and run start. duplicated-layer. partially-confirmed. ~35 lines, risk low. Visible change: none. Behaviour-preserving: yes. **Status:** open (behaviour-preserving).

Where: `internal/factory/control/priority_typesafe.go:65-87,131-134`, `internal/factory/control/systemone_typesafe.go:30-33,79-82,122-153,165-229`, `internal/factory/control/delivery_typesafe.go:12-17`, `internal/factory/judgment.go:159-183`.

**Simpler.** Pull postPriorityRequest and the request half of evaluateQuestionMap into one function that POSTs to SystemOne. Add one model-default helper and use it in Select, Evaluate, EvaluateQuestions and DeliveryModel. Keep the two decoders and their error mappings separate: the MCP refusal text and the priority acceptance rules differ. Unifying the choice validator is optional and would change the maximal-choice tolerance and the error text.

**Risk.** The requests on the wire stay the same; only the response caps unify.

**Safe first step.** In internal/factory/control, merge postPriorityRequest (priority_typesafe.go) and the request-building and client-default half of TypeSafeSystemOne.evaluateQuestionMap (systemone_typesafe.go) into one helper, postSystemOne(ctx, client, model, state, questions, key) (*http.Response, error). Each caller keeps its own error mapping: ErrPriorityUnavailable for priority, the existing "System One ..." strings for the MCP path. Each caller also keeps its own size cap and decode. Replace the four "jev-latest" fallbacks with one systemOneModel(m string) string helper.

### GOM-12: Publication receipts are re-checked field by field against the persisted intent the host already enforced

Other Go packages (forgejo, acceptance, project, host, tailnet, application, config, strictjson, filelock, avatar, cmd/soda-dashboard, cmd/soda-tailnet, cmd/soda-extension). unneeded-validation-or-digest. partially-confirmed. ~30 lines, risk low. Visible change: none. Behaviour-preserving: yes. **Status:** open (behaviour-preserving).

Where: `internal/forgejo/publish/operation_receipts.go:40-58,90-100`, `internal/forgejo/review.go:188-190`, `internal/forgejo/merge.go:128-130`, `internal/forgejo/publish.go:184-210`.

**Simpler.** In DecodePublishReceipt and DecodePRCreateReceipt, keep the committed-effect and size checks and use a plain json.Unmarshal. Drop the trailing-token check and the Ref/NewOID/Comparison/Actor/Repository/OldOID/head/base echo comparisons, and keep only the PR identity (>0) check. publicationOutcomeMatches already checks the envelope scope. Drop the repeated outcome ActorID/RepositoryID checks in review.go and merge.go. Keep ServiceBackground's checkBackgroundRecord: it is Soda's only installation/operation-id check, because the SDK client is not used.

**Risk.** These checks only catch a host bug.

**Safe first step.** In internal/forgejo/publish/operation_receipts.go, delete the second decoder.Decode(new(any)) EOF checks and the "differs from its intent" comparison blocks in both decoders (lines ~44-58 and ~93-100), keeping the committed/size checks, the PR identity >0 check and the field assignment. Delete the `outcome.ActorID != w.ActorID || outcome.RepositoryID != w.Repository` checks at review.go:188-190 and merge.go:128-130. Leave checkBackgroundRecord and publicationOutcomeMatches in place.

### PRP-10: PrepareState.Ready is derived from Phase, and the intervention check is repeated inline at about five sites

Project preparation pipeline below the API (standing and candidate preparations, decisions, hold, host and root helper). too-many-states-or-budgets. partially-confirmed. ~30 lines, risk low. Visible change: none. Behaviour-preserving: yes. **Status:** open (behaviour-preserving).

Where: `internal/project/preparation.go:367`, `lib/host/src/preparation/state.rs:75,123-124`, `cmd/soda-project-terminal/src/factory_roles/records.rs:358`, `internal/factory/control/preparation_start.go:93-97,135-137`, `internal/store/preparation.go:202`, `internal/factory/control/dispatch_preparation.go:206`, and 1 more.

**Simpler.** Drop PrepareState.Ready from the Rust and Go structs and the native helper output, and read Phase == ready at every Ready call site, including the derived API views. Replace the inline failed/stopped/interrupted (+Stopped) checks with one helper. Keep Stopped, Retirement and the candidate meta stop receipt: Stopped marks a retired ready or failed preparation, and the meta is the only durable record of the stop receipt.

**Risk.** The wire field 'ready' is read by the frontend and Feed. It can be derived in the API view.

**Safe first step.** Add one function in internal/project/preparation.go, PrepareState.NeedsIntervention() (Stopped || Phase in failed/stopped/interrupted). Use it to replace preparationNeedsIntervention in internal/factory/control/preparation_start.go:135-137 and the inline check in internal/store/preparation.go:202. Leave the 3-way Phase-only checks in dispatch_preparation.go:206 and publication_children.go:868/929 as they are, or move them to a matching Phase-only helper, because they intentionally do not look at Stopped. Then remove the Ready field from internal/project/preparation.go PrepareState, lib/host/src/preparation/state.rs and the records.rs output, and replace each .State.Ready / state.ready read with Phase == ready.

### RUN-9: Four near-identical typed root settlers in settleClosedAttemptForRunTx can be merged; SaveFactoryRun's JSONB guard protects six concurrent snapshot writers and is not a pure repeat

Run end: task events, checkpoints, results and settlement (RUN). duplicated-layer. partially-confirmed. ~30 lines, risk low. Visible change: none. Behaviour-preserving: yes. **Status:** open (behaviour-preserving).

Where: `internal/store/factory.go:163-216`, `internal/store/factory.go:233-327`, `internal/store/factory.go:51-115`, `internal/factory/control/dispatch_launch.go:698`, `internal/factory/control/dispatch_launch.go:760`, `internal/factory/control/dispatch_launch.go:809`, and 3 more.

**Simpler.** Merge settleTypedCompletionRootTx, settleTypedStallRootTx, settleTypedIntentRootTx, settleTypedDeliveryRootTx and the inline NonCodeIssueWork branch into one function. A switch on Subject.Kind does the kind-specific scope and subject checks and returns the locked allowance; the shared Validate step and the `if !Closed && !finished return; transitionLockedAttemptRootTx(false)` tail are written once. Keep SaveFactoryRun's guard for now. Shrinking it means first changing the dispatch_launch, dispatch_recovery and verification_execution writers to targeted column or jsonb_set updates instead of whole-document snapshots. That is a separate redesign, not a removal of duplication.

**Risk.** Admission metadata must still not be dropped when settleRun saves a run loaded before admission wrote it. Use a targeted UPDATE of settle columns instead of a whole-document overwrite. This overlaps whichever finder owns attempt allowances.

**Safe first step.** In internal/store/factory.go, replace the four settleTyped*RootTx functions and the inline NonCodeIssueWork block in settleClosedAttemptForRunTx with one settleTypedRootTx. Inside it, a switch per Subject.Kind keeps each existing scope check, subject check and error, and calls that kind's lock helper; the shared tail (`if !allowance.Closed && assignment.Stage != factory.AssignmentFinished { return nil }; return transitionLockedAttemptRootTx(ctx, tx, allowance, false, now)`) is written once. Leave the SaveFactoryRun SQL unchanged.

### BND-9: Snapshot validation re-hashes issue and comment text to check digests sent alongside it in the same response; acceptance freshness tokens partly overlap

Boundary machinery: strict JSON, digests, receipts, revision checks (cross-cutting). unneeded-validation-or-digest. partially-confirmed. ~10 lines, risk low. Visible change: none. Behaviour-preserving: yes. **Status:** open (behaviour-preserving).

Where: `internal/forgejo/snapshot_issue.go:44-49`, `internal/forgejo/snapshot_issue.go:66-68`, `frontend/spaces/sodaspaces-factory-acceptance.ts:30-37`, `frontend/spaces/sodaspaces-factory-acceptance.ts:225-235`, `internal/web/api/issue_acceptances.go:44-120`, `internal/factory/control/acceptance_evidence.go:71`, and 2 more.

**Simpler.** Stop re-hashing Title, Content and comment Content in validIssue and validCommentPage. Keep the presence checks, or trust the extension. Keep TitleDigest and ContentDigest: acceptance records, AcceptedContext.ValidateAcceptance and readiness creation verification (whose webhook carries only digests) all rely on them. Collapsing the four acceptance tokens is optional and would change the API.

**Risk.** The stale-accept check is a real concurrent-writer concern, since issues can be edited in Forgejo. Keep one token for it.

**Safe first step.** In internal/forgejo/snapshot_issue.go, delete the recompute in validIssue (`issue.TitleDigest != ContentDigest(issue.Title) || issue.ContentDigest != ContentDigest(issue.Content)`, lines 47-49). In validCommentPage, delete the `item.ContentDigest != ContentDigest(item.Content)` comparison. Keep the non-empty checks. Leave the digest fields, the acceptance tokens and the DB columns unchanged.

## Minor items

These are small, local cleanups: each was confirmed, and refuted ones are left out. The note is the checker's. `bp` marks items that are behaviour-preserving.

### Completion, reassessment, review and JEV judgments

| Item | Where | bp | Note |
|---|---|---|---|
| Pass-through digest wrapper | `internal/factory/control/review_cycle.go:853-857` | yes | review_cycle.go:855 only forwards to reviewCycleInputDigestTyped(head, base, review.Review, review.Clear, checks). Its two callers are in publication_children.go:257 and :283, and both can call the typed function directly. |
| Decision-state Model is never set | `internal/factory/control/completion_decision.go:38,93-100` | yes | The only constructor (completion_decision.go:93-100) never sets Model, and nothing else assigns it, so JEV always receives "model":"". Dropping the field removes that key from the state and changes judgment fingerprints, which only invalidates cached receipts (there are no users). The comment saying the cache is bound to the model is wrong. |
| Hashes sent to JEV, and a redundant fingerprint | `internal/factory/control/completion_decision.go:34-46; internal/factory/control/completion_drive.go:41-48` | yes | InventoryDigest and EvidenceFingerprint go into the JEV state, although CompletionID, CycleID and the full bodies are already in it. EvidenceFingerprint is sha(digest+eventID+cycleID), which the state already covers. It can come out of the JEV state; only fingerprints and JEV input change. Removing the record field is a bigger change: the store, journey readback and the line-208 match use it. |
| Purpose registry for a closed constant set | `internal/factory/judgment.go:185-239` | yes | JudgmentPurposes() is called only by valid(), and valid() is used only in JudgmentFingerprint. The comment says twelve purposes and the list has 13. JudgmentNextWork is used only as a receipt label in priority.go:109. A switch or an inline set would behave the same. |
| Audit-only advice field | `internal/factory/control/judgment_advice.go:13-26,64-67` | no | Only intent_decomposition_judgment.go:84 sets Candidates, and they only replace the receipt's Candidates (judgment_advice.go:64-67). Removing the field changes what that receipt records but no decision. It is minor and changes receipt contents only. |
| Double 30-second timeouts | `internal/factory/control/systemone_typesafe.go:135; priority_typesafe.go:76; judgment_advice.go:57; judgment_advice_delivery.go:110` | yes | The default http.Client has a 30s Timeout (systemone_typesafe.go:135, priority_typesafe.go:76), and judgment_advice.go:57 and judgment_advice_delivery.go:110 wrap calls in a 30s context. Keep the client timeout, which also covers MCP calls that have no context bound, and drop the duplicate context timeouts. Effectively identical. |
| Test-only retry entry point | `internal/factory/control/completion_drive.go:565-583` | yes | StartCompletionRetry is called only by completion_serving_pg_test.go and dispatch_timeonly_test.go. Production RetryCompletion (completion_retry.go:64) calls checkCompletionRetry directly. The method and those test uses can be deleted. |
| Unreferenced methods missed by the dead-code pass | `internal/factory/control/reassessment_adoption.go:860; internal/factory/reassessment.go:356-360` | yes | rg finds no references at all, tests included, to ReassessmentJournal.ResumeOperation (reassessment_adoption.go:833) or CompletionEvent.UniqueKey (reassessment.go:358). Commit 0e59bf58 ('Delete Go code that nothing reaches') missed them. |
| Compatibility branch for an old empty-inventory encoding | `internal/store/factory_reassessments.go:122-128` | yes | The Evidence tag no longer has omitempty, and sealing builds Bodies with make(...,0,n), so an empty inventory now encodes as [] and never as null. The nil-Evidence exception at factory_reassessments.go:122-128 only serves legacy rows. Without it, a nil Evidence on a sealed record would simply be refused. |
| Three child-ID schemes in ensureReviewCycle | `internal/factory/control/review_cycle.go:965-1090` | no | There are head-based IDs for a first head or round, head-based IDs again for a new head after earlier rounds (identical branches), and round-derived IDs, plus replay that guesses the scheme from the stored ID. A single round-derived scheme is enough. Child and preparation IDs would change, which only affects existing data. The two identical reviewer firstHead branches can be merged with no behaviour change. |
| Freshness struct re-projects CompletionLocal | `internal/factory/control/completion_freshness.go:10-80` | yes | CompletionFreshness copies four CompletionLocal fields plus the digest and has exactly 2 call sites (completion_drive.go:495, completion_proposal.go:136). A function taking (CompletionLocal, digest) pairs could produce the same errors. |
| Strict decoding of resolve_post | `internal/factory/post_resolution.go:39-73` | no | DecodePostResolution uses DisallowUnknownFields plus a trailing-EOF check, so model payloads with extra fields are refused. Loosening it changes behaviour by accepting them, which is arguably desirable but not identical. |
| Optional sink found by type assertion | `internal/web/api/factory_mcp_systemone.go:42-47,59-62,79-82` | yes | There are three assertions on deps.PriorityJudgments.(consultationSink) (lines 59, 79, 253), and *store.Store implements both methods. Adding the methods to FactoryMCPPriorityJudgmentSource removes them, but test fakes that lack the methods would have to be updated or deleted. |
| Failure status decided by error substrings | `internal/web/api/factory_mcp_systemone.go:84-92` | no | Lines 84-92 choose unavailable or uncertain by matching 'not configured' and 'request failed' in error text (the latter from systemone_typesafe.go:139). Sentinel errors would keep the same statuses but need new error values. It is fragile rather than over-built, and the fix is small. |
| Duplicated check-then-merge sequence | `internal/factory/control/review_cycle.go:29-117` | yes | Both functions end with 'checks pass and currentPublicationApproved, so mergeOne into a throwaway MergeReport'. The surrounding order differs: CheckPass merges per assessed link before producing children, while progressAfterPublish refreshes after children. Only that tail can become one shared helper. The overlap is small. |
| Two local-capture entry points for the same material | `internal/store/factory_reassessments.go:327; internal/factory/control/completion_adoption.go:235` | no | store.CaptureCompletionLocal is used only at completion_adoption.go:235, which compares source, preparation and authority, and right after it bracketMaterialCollect recaptures through the serving CaptureLocal (the source-tip read) and runs matchReassessmentEffects. The store capture looks redundant, but it reads the store instead of the forge ref and returns a different error or wait, so removing it is not strictly identical. |

### Model-facing MCP tools

| Item | Where | bp | Note |
|---|---|---|---|
| Optional-interface test seams around *store.Store | `internal/web/api/factory_mcp.go:28-89,1142-1146 (plus 13 type assertions: _library.go:157, _checkpoints.go:26, _context.go:19, _verification_journey.go:44, _inbox.go:35, factory_mcp.go:848,864, _research.go:68, _repair_scope.go:56,74, _systemone.go:59,79,253)` | yes | FactoryMCPHandler always passes s.Store for all five deps, and the store implements every asserted interface. So the !ok/nil 'unavailable' branches never run in production, apart from test-only stubs, and the store can be held directly. |
| Builder guidance names a field the finish tool refuses | `internal/web/api/factory_mcp_checkpoints.go:93 vs internal/web/api/factory_mcp.go:510-511` | no | The guidance at factory_mcp_checkpoints.go:93 says 'using its candidate and receipt_digest'. But finish_implementation's required list is empty and its optional keys are only unchanged and candidate (a commit string), so receipt_digest gets 'unknown payload field'. The fix is to change the text, which changes the prompt slightly (intended). |
| Unreferenced MCP constants and validators | `internal/factory/mcp_profile.go:202,209-212,370-380; internal/factory/mcp_manifest.go:317-347` | yes | Partly out of date. ValidateMCPEnvelope, ManifestCoverage and ManifestUncovered were already deleted in 0e59bf58, and mcp_profile.go now has only 364 lines. The four constants MCPMaxRequestEnvelopeBytes, MCPMaxInstructionBytes, MCPMaxConfigBytes and MCPMaxLaunchEnvelopeBytes are still defined only, with no references in Go or Rust. |
| Overlapping byte bounds | `cmd/soda-project-terminal/src/factory_mcp.rs:15-16; internal/web/api/factory_mcp_bound.go:218; internal/factory/mcp_profile.go:201-216; internal/factory/plan_task.go:26; internal/web/api/factory_mcp_context.go:83` | no | The bounds listed are all there: relay 96K/256K, HTTP MaxBytesReader of MCPMaxSubmissionPayload+8192 (72K), payload/plan/result 64K, page 32K, current context 8192. The relay's 96K request cap is effectively shadowed by the 72K HTTP cap. Merging down to two limits is a design change and would alter some refusals. |
| Per-run rate limiter for a trusted local worker never forgets runs | `internal/web/api/factory_mcp.go:185-224` | no | One limiter is created per handler (bound.go:159). Its active, window and calls maps are keyed by run id and never pruned. Deleting the limiter removes the 4-concurrent and 60-per-minute refusals. Expiring stale entries would keep behaviour the same. |
| Transport refusals discard the cause | `internal/web/api/factory_mcp_bound.go:186-204,223-230` | yes | Both Prepare errors and Observed/Admit errors write a fixed 403 'worker context is not active'. unavailableFactoryMCP only nil-checks err and always writes 'factory MCP unavailable'. Dropping the err parameter keeps behaviour identical; surfacing the cause would change the messages. |
| Internal milestone text leaks into model output | `internal/web/api/factory_mcp.go:1000` | no | factory_mcp.go:1000 returns 'plan/review/inventory pages plus pagination arrive with FW40' as the model-facing reason. Rewording only changes that reason string. |
| Compatibility shim for pre-settled_at runs | `internal/store/factory_mcp_accounting.go:89-93,114-133` | yes | factory.go:64 stamps settled_at with clock_timestamp() when a run becomes settled, and accounting requires settled=true. So legacyFinishedRunEnd only serves old rows; with zero users, deleting it (treating a missing settled_at as an error) keeps behaviour the same for current data. |
| Stale comment on the submission guard table | `internal/store/factory_mcp_submission.go:55-62; internal/store/schema.go:156` | yes | The comment calls it a 'schema-registration proposal' and says tests apply it 'until registration lands', but schema.go:156 already registers FactoryMCPAssignmentGuardDDL. Only the comment needs to change. |
| finish.rs repeats the capture/return/close block three times and keeps unused deadline parameters | `lib/host/src/factory/finish.rs:114-240` | yes | The capture, return_lease/reconcile and close_execution block is copied verbatim in stop_failed_start and stop_timed_out, and likely in finish_run. 'let _ = deadline;' appears five times (lines 23, 58, 121, 203, 251), not four. |
| Artifact upload makes the model base64-chunk files | `internal/web/api/factory_mcp_artifact.go:19-50; internal/factory/mcp_artifact.go:11-50` | no | Chunks are capped at MaxFeedArtifactChunk=32K with an 8 MiB total. Each chunk needs upload_id, offset and final in the payload plus its own submission_id. Switching to a path read or a single call changes the tool contract. |
| Library Admin has 12 tools for one change pipeline | `internal/factory/mcp_profile.go:86-97; internal/web/api/factory_mcp_library.go:24-138` | no | There are 13 library tools (inventory, preview, begin, seven operations, report, finish), plus search and read elsewhere. Folding them into one change tool with an op enum is a redesign of the model-facing surface. |
| Worker prompt carries controller plumbing | `internal/factory/dispatch_prompt.go:317-338,359-360,461-478; internal/factory/dispatch_prompt_stage.go:191-322` | no | writePromptHeader and writePromptPolicy send template id, provider connection, requirements/approval ids, three concurrency limits and the readiness fingerprint to the model. Trimming changes the prompt. admitPromptText only partly duplicates the final MaxFactoryPrompt check: it uses rough overheads and gives a different error. Removing it is nearly, not exactly, identical. |
| Per-request host inspection and run read | `internal/web/api/factory_mcp_bound.go:29-38,207-215` | no | Every request calls FactoryInspect (5 s timeout) and Store.FactoryRun in prepareFactoryWorkerMCP, then reads FactoryRun again at bound.go:211 just to gate NativeTask. Reusing the first run read is identical. Dropping the per-request liveness check would weaken the revocation guard. |
| Capability id duplicates run id | `internal/factory/control/mcp_capability_producer.go:130-133` | yes | The producer sets ID: run.ID and RunID: run.ID. Both fields are used widely: the capabilities table id, submissions.capability and secret derivation. Merging them is mechanical but touches the schema and DTO. |
| Product Lead context exposes catalog binding digests | `internal/web/api/factory_mcp.go:764-774` | no | factory_mcp.go:769 puts the whole coordinator catalog object under 'binding' in verification_catalog, next to the one check entry the model uses. Removing it changes the context payload but nothing reads it back. |

### Identity service and provider credentials

| Item | Where | bp | Note |
|---|---|---|---|
| Keep the broker process; shrink it instead of porting again | `cmd/soda-identity` | yes | cmd/soda-identity/src is 6755 lines; the broker opens the store with the key file through Store::open_encrypted, so the separate process and key really are the custody boundary. This item is advice to keep the broker, not something to remove. |
| Host account wire mimics Go encoding and checks the key revision twice | `lib/host/src/account/mod.rs:47-100,301-400` | no | AccountConfirmation is a hand-written visitor that matches keys case-insensitively, sitting next to a derive(deny_unknown_fields) struct, and the JSON body is built by hand in Go map key order (mod.rs:374). The apply path first runs a preview to compare revisions, but keys.rs:130 checks the same revision. Dropping the preview gives the same outcome; only the error text changes ('key file changed since preview' replaces the preview mismatch error), and one helper call is saved. |
| Provider/EnrollmentSession traits with four newtype wrappers | `cmd/soda-identity/src/control.rs:10-93` | yes | control.rs:10-93 has two traits and four wrappers (Codex/Muse Provider and Session) whose only job is map_err(map_provider_error). An enum of the two providers would behave the same. |
| Entropy-injection seams left from banned unit tests | `cmd/soda-identity/src/control.rs:247-255; cmd/soda-identity/src/providers/mod.rs:84-98; cmd/soda-identity/src/crypto.rs:31-55` | yes | new_id_with, enrollment_tempdir_with and seal_with_random each have exactly one caller, and that caller passes getrandom::fill. Nothing in cmd/soda-identity/tests uses them. |
| Dead Rust items in the broker | `cmd/soda-identity/src/runtime.rs:9,93-96; crypto.rs:74-76; providers/codex/mod.rs:248-251; providers/codex/protocol.rs:56-59; control.rs:202-213; providers/mod.rs:54-59; wire_errors.rs:59-61; wire_time.rs:28-42,62-68` | yes | No callers in src or tests for any of these: default_limit/DEFAULT_LIMIT (a None limit actually falls back to FINISH_LIMIT), nonce_size, Codex Session::cancel (the cancelled flag is only read, never set), Controller::close (http.rs's backends.close() is a different object), providers and wire is_denied/is_uncertain, as_system_time, and parse_/format_rfc3339_nano with their re-export in wire.rs. |
| Store.grants is an Option that is always Some | `cmd/soda-identity/src/store.rs:24,35-58; store_schema.rs:53-69,137-141; store_connections.rs:19-23` | yes | Store::open is pub(crate) and only open_encrypted calls it, with Some (main.rs:223 and tests/common:119). The None branches in store_schema.rs and store_connections.rs:19 can never run. |
| Duplicate constants and codecs | `cmd/soda-identity/src/http_routes.rs:16; http_wire.rs:8; providers/types.rs:43-54; wire_scalars.rs:6-20` | no | MAX_BODY = 512<<10 appears in both http_routes.rs:16 and http_wire.rs:8; removing the duplicate is identical. The read_request comment at http_routes.rs:157 is stale. The two i64_string codecs differ: the providers one rejects null, wire_scalars turns null into 0. Merging them would make provider types accept null, so that part is not strictly identical. |
| Circular validate callback at registration | `cmd/soda-identity/src/registration.rs:19-22; cmd/soda-identity/src/runtime.rs:68-74` | no | register() calls runtime.validate, which goes to soda-host /identity/validate, and that re-attests a container or factory process that soda-host itself just bound before calling /register. It is mostly redundant, but it is a liveness re-check: removing it changes the result only in the rare case where the process dies in between. |
| Enrollment endpoint quirks in the Go API | `internal/web/api/identity.go:54-62,79-85` | yes | apiIdentityEnrollments type-asserts for Enrollments, which the only client (client.go:114) implements, so it can go on the IdentityClient interface. identityResult re-runs checkLifecycleSession after every broker call, on top of identityAdmission running it first; the only difference from dropping it is a session revoked during the call. |
| Hand-rolled mkdir -p and Go-flag argument parsing | `cmd/soda-identity/src/main.rs:98-120,307-341` | yes | mkdir_all_mode (main.rs:307-341) re-implements DirBuilder::recursive(true).mode(0o700). parse_args mimics Go flag spellings and help, but soda-identity.service runs ExecStart with no arguments. Only error strings would change. |
| Codex leases force a reconnect on every broker restart | `cmd/soda-identity/src/retirement.rs:17-31,88-110` | no | reconcile() ends every lease, and end() marks non-Muse leases uncertain (REAUTH) before stopping them. This is real, intended behaviour, a caution for simplifying retirement rather than something to remove. |
| Doc drift in the identity docs | `cmd/soda-identity/README.md:18,45; docs/reference/credentials.md:93-95,218-219,234-235; lib/host/src/iclient.rs:6` | yes | README.md:18 says /usr/libexec/..., but the unit uses /usr/local/libexec/soda/soda-identity. credentials.md:95, 219 and 235 still carry the pinned-bytes, 'retain native qualification evidence' and 'Muse factory runs stay unavailable' text. lib/host/tests has no iclient_oracle.rs. These are doc-only fixes. |
| Unused import of soda-providers | `frontend/spaces/sodaspaces-project-work.ts:10` | yes | Neither sodaspaces-project-work.ts nor its view renders <soda-providers>. The import only registers the element, and feed-entry, identity and factory-settings, which do render it, each import it themselves. |

### Frontend: Workspace, project, applications, terminals, layout, Forgejo and Tailnet pages

| Item | Where | bp | Note |
|---|---|---|---|
| renderSpacesLink() always returns '' but is still rendered | `frontend/spaces/sodaspaces-workspace-toolbar-view.ts:139-141; frontend/spaces/sodaspaces-workspace.ts:1290` | yes | toolbar-view.ts:139-141 returns ''; the only use is the interpolation at workspace.ts:1290 (plus its import at line 52). Delete the function, the import and the ${renderSpacesLink()} slot. |
| ProjectRowsInput.factory is always '' and gets rendered | `frontend/spaces/sodaspaces-workspace.ts:1028; frontend/spaces/sodaspaces-workspace-navigation.ts:272,298` | yes | It is typed `factory: unknown` (navigation.ts:272) and set to '' only at workspace.ts:1028, so `${input.factory}` at line 298 always renders nothing. |
| mountSodaspaces has a factory parameter nobody passes | `frontend/spaces/sodaspaces-workspace.ts:2256-2264,419,428` | yes | All 5 callers (soda-spaces-entry.ts, sodaspaces-page.ts and two test fixtures) pass only (root, context). configure(context, factory) has no other caller, so the field can just use mountTerminal. readFactory at line 1924 can read it directly. |
| soda-workspace-minimum event has no consumer | `frontend/spaces/sodaspaces-workspace.ts:639-644; tests/frontend/fixtures/workspace-measurement-probe.ts:66` | yes | The only listener is tests/frontend/fixtures/workspace-measurement-probe.ts:66. Remove publishMinimum/lastMinimum (workspace.ts:302,639-644,661) and that probe counter. Do not confuse it with the unrelated publishMinimum in terminal-screen.ts. |
| 'Older servers' compatibility in applicationsResponse | `frontend/spaces/sodaspaces-application-response.ts:351-366` | yes | The Go handler (internal/web/api/applications.go:261-266) always sends a non-nil launchable slice and a boolean can_setup, so the Array.isArray and can_setup-undefined fallbacks never fire with the real server. Any test fixtures that omit these fields would need them added. |
| Duck-typed error status/code helpers | `frontend/spaces/sodaspaces-application-delivery.ts:36-52; frontend/spaces/sodaspaces-application-setup.ts:12-19` | yes | The production transport throws SodaRequestError. The {status, code} duck typing is only exercised by tests/frontend/application-controls.test.ts, which throws Object.assign(new Error, {status, code}). Those tests would need to throw SodaRequestError, or be dropped. |
| Hidden Start/Stop fieldset rendered for non-admins | `frontend/spaces/sodaspaces-project-settings-view.ts:40-52` | yes | In project-settings-view.ts:44-50 the fieldset is always hidden for non-admins. Rendering only the two <p> lines looks the same. Tests may count on its presence (project-access-controls/drawer-controls tests mention hidden controls). |
| CSS files renamed at build time | `scripts/build-soda-extension.ts:15-21; frontend/spaces/soda-spaces-entry.ts:17-23` | yes | build-soda-extension.ts:17-18 renames workspace->page and project->drawer. Consumers to update together: soda-spaces-entry.ts:20-21, soda-feed-entry.ts:12, scripts/fixtures/spaces-review.html:3 and build-soda-extension.test.ts:62-63,88,319. |
| Four unused Soda locale keys | `frontend/forgejo/locales/en-US.ini:8-11` | yes | rg finds profile_portrait, tailnet_title, tailnet_appliance and tailnet_projects nowhere outside the locale file. |
| Secret-clearing ceremony on a local parameter | `frontend/tailnet/soda-tailnet-observation.ts:51-80` | yes | soda-tailnet-observation.ts:67 and 79 set `body = undefined` on a local parameter after use. This does nothing to the caller's string or to memory. |
| confirm_id echoes the project id back as a 'confirmation' | `frontend/spaces/sodaspaces-project-network.ts:57-63` | yes | The frontend sends confirm_id: network.project, and internal/tailnet/control_types.go:247 rejects any request unless ConfirmID == Project. Behaviour stays the same only if the server check (lines 243, 247 and 270) and tailnet_test.go are removed in the same change. |
| Login/environment-id regexes re-checked in four places | `frontend/spaces/sodaspaces-terminal.ts:476-494; frontend/spaces/sodaspaces-project-refresh.ts:27-29; frontend/spaces/sodaspaces-project-mutations.ts:129-132; frontend/spaces/sodaspaces-project-members.ts:37-41` | yes | The same /^[a-z][a-z0-9_-]{0,30}$/ check sits in terminal.ts:483-484, project-refresh.ts:28, project-mutations.ts:130 and project-members.ts:39 (the members one even leaves out the root exclusion). It only rejects malformed server replies, so normal operation is unchanged. |
| Stale spaces README | `frontend/spaces/README.md:7-23` | yes | README.md:7-23 still describes sodaspaces-workspace.ts as the shared page/panel workspace owner, soda-spaces-entry.ts as the 'shared Workspace owner' and sodaspaces-page.ts as the review fixture. This contradicts the no-Workspace decision. It is a docs-only change. |
| Workspace keeps saved requests from page mode for the native panel | `frontend/spaces/sodaspaces-workspace.ts:778-788,1406-1408` | yes | savedRequests is set only through setSavedRequests (setup.ts:585, inside renderSetup). renderSetup runs only when setupScreen is true, which requires binding.kind==='page'. renderDiscoveryRequests (workspace.ts:778-788) returns '' in page mode, so in native mode the list is only populated from earlier page use, and in practice it is empty. Can be removed with FEP-1. |

### Frontend: Factory settings, work, progress, request, intent screens

| Item | Where | bp | Note |
|---|---|---|---|
| Factory run watch is split across four files through bags of state closures | `frontend/spaces/sodaspaces-factory-request.ts:9-37; frontend/spaces/sodaspaces-factory-display.ts:5-25; frontend/spaces/sodaspaces-factory-screen.ts:8-25; frontend/spaces/sodaspaces-factory.ts:119-155,201-282` | yes | RequestInput declares about 28 closures. sodaspaces-factory.ts has about 50 arrow closures that rebuild these input bags. mountFactoryWatch is only called from sodaspaces-workspace-factory.ts:186, so the watch is Workspace-only. Folding the files into private methods is a pure refactor, and deleting the watch goes with removing the Workspace. |
| Hardening notice points to a settings path that does not exist | `prod-hardening:frontend/spaces/sodaspaces-factory-allowance.ts:14,54` | no | The file is not on main; it exists only on the prod-hardening branch, where settingsPath = 'Settings → SodaOS → Factory control'. On main, 'Allowance minutes' lives in the factory settings sponsorship form (sodaspaces-factory-settings.ts:1575). This is a copy fix on another branch, not removal of over-engineering. |
| 64 KiB default response cap | `frontend/spaces/sodaspaces-api.ts:14` | no | readSodaJSON(maxBytes = 65536) is at api.ts:14. There are about 45 call sites, and only feed.ts:504 raises the cap (4 MiB for conversations). Large reads would fail. This is a possible latent bug rather than over-engineering, and raising the cap changes behaviour only for oversized responses. |
| Shared helpers live in feature files that are slated for deletion | `frontend/spaces/sodaspaces-factory-activity.ts:24-30,104-109; frontend/spaces/sodaspaces-factory-work-view.ts:23-31; frontend/spaces/sodaspaces-intent-response.ts:155-156` | yes | nativePath and activityTimestamp are exported from factory-activity.ts and used by 10 files. nativeIssueLink, nativePullLink and decisionID are used by Feed files. This is a note about deletion order, not over-engineering. Moving the helpers changes nothing observable. |
| Requirements acceptance makes the owner type ids by hand and copies digests back | `frontend/spaces/sodaspaces-factory-acceptance.ts:226-238,290-320,362-372` | no | The owner types comment ids into comma-separated text fields and enters each prerequisite's repository ID and issue number by hand. The digests are not computed in the client: they are server preview values echoed back. Reachable through project-work.ts, which itself is only mounted from sodaspaces-workspace.ts. Simplifying needs server changes, so behaviour changes. |
| Departments are a speculative grouping feature | `frontend/spaces/sodaspaces-intent-view.ts:749-788; frontend/spaces/sodaspaces-intent.ts:2716-2756` | no | departmentDialog shows 'Private context for your ideas. It grants no access.' Backend: a factory.Department type plus an optional Department id on intents and filters, carried through reassessment_adoption. No grouping behaviour beyond the label and filter turned up. Removing it drops a visible UI feature and the department filter. |
| Sequential reads in the direction-revision view | `frontend/spaces/sodaspaces-factory-work.ts:564-568` | yes | Only partly accurate: readRevisionAccess needs revision.revision, so 3 of the 4 reads are independent, not 4. They can be parallelised with the same result; this is a minor latency gain. |
| Raw ids in user-facing labels | `frontend/spaces/sodaspaces-factory-settings.ts:1540,1569; frontend/spaces/sodaspaces-factory-work-journey-view.ts:113-117; frontend/spaces/sodaspaces-factory-progress-view.ts:826-830` | no | Verified: 'Actor X · token Y' (settings.ts:1540), 'project_id · grant id' (:1569), 'Completion … cycle … (generation n)' (journey-view.ts:116) and 'Native target N' with submitter/authorized (progress-view.ts:826-830). This is a copy/UX issue; changing it changes the visible text. |
| Tracker ids in code comments | `frontend/spaces/sodaspaces-factory-context-resolution.ts:64,108,151,172; frontend/spaces/sodaspaces-factory-work.ts:98` | yes | Found FJ-447 at context-resolution.ts:64 and V01–V07 at work.ts:98. About 33 matching references across the factory, intent, network and project-work files. These are comments only. |
| soda-providers imported but never rendered in project-work, and rendered in three hosts | `frontend/spaces/sodaspaces-project-work.ts:10; frontend/spaces/soda-identity.ts:279; frontend/spaces/soda-feed-entry.ts:53; frontend/spaces/sodaspaces-factory-settings.ts:1069` | yes | project-work.ts:10 is a side-effect import with no <soda-providers> in its template. Every host (identity, feed-entry, factory-settings) imports the element itself, so dropping that import is safe. Each of the three hosts creating its own instance is ordinary component use, not clear over-engineering. |
| Network panel copy describes internal mechanisms | `frontend/spaces/sodaspaces-network.ts:108-116` | no | network.ts:108-116 contains 'A connected node is not proof of client reachability' and 'never repeat an uncertain key request'. Fixing it means rewriting or removing text the user sees. |

### Factory dispatch and run start

| Item | Where | bp | Note |
|---|---|---|---|
| Assignment endpoint has no client | `internal/web/api/factory_assignments.go:1-177; internal/web/api/extension_native.go:239` | yes | The route is registered at extension_native.go:239 and the handler file is 177 lines. Nothing in frontend/, tools/, tests/ or cmd/ calls it. The only references are docs/reference/api.md:76 and the route-table rows in extension_test.go:257-302, which would be deleted along with it. |
| Store.RecordFactoryRun is test-only | `internal/store/factory.go:14-35` | yes | No non-test file calls RecordFactoryRun( apart from its definition at internal/store/factory.go:16. Production inserts runs through the dispatch packet. |
| Prompt-build failures reported as prompt_too_large | `internal/factory/control/dispatch_attempt.go:481-487; internal/factory/dispatch_prompt.go:233-237` | no | dispatch_attempt.go:481-487 maps every BuildDispatchPrompt error except ErrPromptPrerequisiteEvidence to WaitPrompt with 'accepted inputs exceed the prompt bound'. That includes the text, scope, phase, plan, reference, stage, context and controller-input checks. The fix is a mislabel correction, so the reported wait reason would change. |
| Prompt carries controller bookkeeping | `internal/factory/dispatch_prompt.go:321-353` | no | writePromptHeader and writePromptPolicy print the template name, native revision, preparation, requirements and approval IDs, provider connection and the three concurrency numbers. dispatch_prompt.go:225 refuses to build without the concurrency values, required checks and provider connection. Removing them changes the prompt bytes and therefore the prompt SHA. |
| Initial phase selection reads native inputs twice | `internal/factory/control/library_maintenance.go:25-49; internal/factory/control/dispatch_attempt.go:516-521` | yes | selectInitialIssuePhase calls readAttemptInputs only to parse the role block. planAttemptForRole then calls prepareAttemptInputs, which runs readAttemptInputs again, and issueLibraryMaintenance re-parses the body to cross-check. The second, authoritative read could select the phase instead; the only difference is one fewer native read. |
| Observation whitelist hides real wait reasons | `internal/factory/control/dispatch.go:248-295` | no | Code emits context_selection_hold (dispatch_launch.go:755, dispatch_context_recovery.go) and phase_not_admitted (dispatch_attempt.go:290-336). Neither is in the switch at dispatch.go:253-279, so both fall to the default and become reason/stage 'unavailable'. Removing the whitelist shows the real reasons in the UI, which is intended but observable. |
| Context bundle choice reuses the priority chooser | `internal/factory/control/dispatch_launch.go:748-753` | yes | In dispatch_launch.go the bundle/reason initialisation is overwritten at once by PrepareContextBundleSelection, and the chooser comes from deps.Priority.Chooser.(ProductJudgment). The dead initialisation can go with no behaviour change; untangling the chooser belongs with DSP-1. |
| Pass-through wrappers | `internal/factory/control/priority_dispatch.go:22-41; dispatch_attempt.go:595-597; dispatch_timeonly.go:99-119; dispatch_preparation.go:58-60; context_bundle_advice.go:96-98` | yes | These only forward: dispatchPriorityPass (one caller), the priorityObserveRevision/priorityRefreshCandidate nil shims, maybeAdvanceCompletedPlanning (one production caller), and SelectContextBundle (only callers are in context_bundle_gap_test.go). prepareRetryCoder is a method value used as the PrepareRetry adapter (dispatch.go:431) that passes nil; it can become a closure. The time-only chain is partly real: dispatchTimeOnlyRoot has two callers (dispatch_timeonly.go:87 and :100), but the middle level can still be folded. |
| SaveFactoryRun enforces monotonic fields in SQL for a single writer | `internal/store/factory.go:51-115; internal/factory/run.go:76-104` | no | The UPDATE has 11 guard clauses plus a jsonb merge that keeps admission, manifests and checkpoint when the new row omits them. Run.Validate re-parses the execution and verification manifests on every save, and there are 7 production callers. It can be simplified, but the merge-preserve behaviour and the ErrNotFound-on-conflict path are observable, so a plain UPDATE is not identical. |
| Legacy review-json report branch is effectively unreachable | `internal/factory/dispatch_prompt.go:405-412; internal/factory/control/dispatch_attempt.go:684-714` | yes | BuildDispatchPrompt has two callers. Publication-child reviewers (publication_children.go:1091-1094) always get phase review, which takes the MCP contract, and typed review takes the typed contract. renderAttemptPrompt passes every prompt through renderPhasedDispatchPrompt, which cuts at the '## Report' marker and replaces the section. The legacy review-json text therefore never reaches a model. |
| Second repository activity feed with its own settings | `internal/store/factory_activity.go:44-131; frontend/spaces/sodaspaces-factory-activity.ts` | no | RepositoryActivity builds issue_created, pr_created and verified_merge items behind three stored toggles with a revision compare-and-set, and is shown in factory-settings.ts:1082. The Feed already records native_issue:created, native_pr:created ('Pull request #N created', F05) and merged events. Removing it deletes a visible settings panel. Note that activityTimestamp and nativePath in that .ts file are imported by about 8 other files and must be moved, not deleted. |
| Priority membership predicate duplicated | `internal/factory/control/priority_dispatch.go:43-53` | no | priorityMemberEligible (priority_dispatch.go:47) repeats the SQL WHERE in factory_priority_inventory.go:123-124. It does real work, though: it re-checks a control re-read after the inventory snapshot, so removing it changes behaviour. It is a candidate for sharing one predicate, which belongs with DSP-2. |

### Intent to plan (Product Lead, Architect, research, decomposition)

| Item | Where | bp | Note |
|---|---|---|---|
| Dead intent states | `internal/factory/intent.go:119,376; frontend/spaces/sodaspaces-intent-view.ts:132-142; internal/store/factory_feed_intent.go:19` | yes | No non-test Go code or frontend path writes 'rejected' or 'abandoned'. Only one store test sets 'rejected'. The states are also accepted at internal/factory/completion_material.go:68 and typed in sodaspaces-intent-response.ts:76, so remove them there as well. |
| Unreferenced prompt-entry types | `internal/project/factory_prompt_entry.go:1-59` | yes | PromptEntryPhase, the PromptIntent* constants, FactoryPromptEntry and FactoryPromptRecord have zero references outside factory_prompt_entry.go. The whole file can be deleted. |
| Test-only plan store wrappers | `internal/store/factory_planning_revisions.go:359-404,478-485,642-654` | yes | ReplacePlanHead, SavePlanTaskReceipt and the singular PlanTaskReceipt are called only from store and control tests. Those tests would be deleted along with them. |
| Write-only global judgment counter | `internal/store/schema.go:35-36; internal/store/factory_intents.go:282-296,382-405` | yes | Five production files lock and increment spent (reassessments, attempt_allowances, context_selection, intents, delivery_repair_adoption), not six, and none compare the scanned value to a ceiling. A test sets spent=1000 and expects reservation to still succeed. Removing it also drops a global serialization point, but nothing depends on its value. |
| A Feed card per draft save | `internal/store/factory_feed_intent.go:15-48` | no | recordIntentRevisionFeedTx posts an F11 'Design proposal recorded' card for every repository-bound draft revision. I did not verify the count of 6-8 revisions per request. Dropping draft cards changes what the Feed shows. |
| Digests printed into issue bodies | `internal/factory/control/intent_generation.go:158-165` | no | writeTaskBodyTail appends the intent/revision IDs, an intent digest, the task digest and the product choices. Adoption at :707 compares the full body byte for byte and refuses with RefusalObjectiveChanged. Removing the tail changes issue bodies, though the adoption check would still compare consistently. |
| Journey view labelled 'fixture' re-validates its own records | `internal/web/api/factory_plan_view_work.go:11-24,48-55,364-384; internal/web/api/factory_journey_readback.go:488-491` | no | Judgments under thresholds are dropped as invalid by validJourneyScores (:364-384). The lane mapping at journey_readback.go:488-491 sends any lane other than a/b/c to 'unassigned', even though plans allow up to 4 arbitrarily named lanes (MaxPlanLanes). Fixing either one changes what is rendered, and the lane mapping is an actual bug. |
| Every unknown error reported as storage outage | `internal/web/api/factory_settings.go:20-35` | no | The default branch returns 503 store_unavailable. The walkthrough validation error at factory_intent_generation.go:310-312 and the dispatch-detail error at :148 both go through it. ErrStaleRevision maps to 'Grant changed'. Fixing this changes status codes and messages. |
| Gitea header fallbacks | `internal/web/api/factory_intake.go:40-45` | yes | intakeHeader falls back to X-Gitea-Delivery/Signature/Event, but Forgejo always sends the X-Forgejo-* headers. Any route tests that use the Gitea names would go with the fallback. |
| Frontend copy of the Go validators | `frontend/spaces/sodaspaces-intent-response.ts:131-158,229-314,727-760` | yes | planningAllowance checks unit==='planning_calls', the decisionID regex and the 0-100 bounds, and intentPlanStatus re-checks IDs and states. These checks only bite when the client and server disagree, so removing them is identical for responses the current server produces. |
| Self-validation of built or stored values | `internal/store/factory_intents.go:211-213; internal/web/api/factory_intents.go:155-163; internal/web/api/factory_intent_generation.go:142-148` | yes | The list runs r.Validate() on each stored row and fails the whole list on any error. The review handler validates the review and then the read it just built. The lost-launch path re-checks attempt fields that already match by key. All of these only matter if data is already corrupt. |
| 30-callback interface for one caller | `frontend/spaces/sodaspaces-factory-request.ts:9-37` | yes | RequestInput has 28 callbacks, and sodaspaces-factory.ts is the only importer of sodaspaces-factory-request.ts. Folding it back in is a pure refactor. |
| Compatibility comments with no old data to serve | `internal/web/api/factory_revision_view.go:41-43; internal/web/api/factory_request_view.go:99-101; internal/factory/control/intent_generation.go:499-501,539-541` | no | The comments are stale, but the code is not dead. unchanged_count is parsed and shown by the current frontend (factory-progress-view.ts:1227). The empty-effective branch is reachable when ResolvedScope returns nothing, for example when no ideas are accepted. The 'predates' checks act as guards against empty SubmittedTasks. Only the comments can go without a visible change, unless the frontend is changed too. |

### The Feed

| Item | Where | bp | Note |
|---|---|---|---|
| Global JEV judgment counter is a lock with no cap | `internal/store/factory_intents.go:282-299 (also factory_context_selection.go:24-56, factory_attempt_allowances.go:1232-1243, factory_reassessments.go:654-669, factory_delivery_repair_adoption.go:70-84)` | yes | All 6 reservation sites (factory_intents.go:283/383, context_selection:25, attempt_allowances:1233, reassessments:655, delivery_repair_adoption:71) do SELECT spent ... FOR UPDATE then spent+=n. They never compare it to a limit, and no non-test Go reads `spent`. Only tests read it (factory_reassessments_test even sets spent=1000 and still expects reservations to succeed). The table and both statements can go. The only observable change is that reservations stop being serialized globally. Tests that assert spent deltas would be deleted with it. |
| Contribution expiry runs even without TypeSafe | `internal/web/api/factory_feed_contribution.go:21-26` | yes | driveContributions calls ExpireContributionReviews (one UPDATE over factory_feed_comments) before the TypeSafeAPIKeyFile check. The only writer of pending rows is ClaimContributionReview, which runs after that check. Moving the expiry below the check keeps behaviour the same, except that rows left pending when TypeSafe was configured and then removed would stay pending. Very minor cost either way. |
| Phase→role mapping duplicated three times | `internal/store/factory_feed.go:104; internal/store/factory_feed_contribution.go:101; internal/store/factory_feed_record.go:218-236` | no | The SQL CASE at factory_feed.go:104 and factory_feed_contribution.go:89 duplicates Go recordProfile, and the copies have already diverged. The feed actor filter lacks WHEN 'repair' THEN 'strategist', while the contribution SQL and Go map repair to strategist. Merging them would change behaviour by fixing that bug: the strategist filter would then include repair records. |
| Post entries overload Activity with communication kinds | `internal/web/api/factory_feed_comment.go:959-961; frontend/spaces/sodaspaces-factory-feed-response.ts:45-46,1231-1232` | yes | factory_feed_comment.go:960 copies Communication.Kind into Activity for posts, and the TS feedActivities spreads all 18 communication kinds. No frontend reader uses activity on posts: the reads at families:202-204, view:69/166/239 and thread:86-87/563 check only merged, completed and judgment, or bail out when payload.communication is set. The Go route reads Activity only for records. Dropping both the assignment and the spread keeps behaviour the same. The kind list itself stays, because parseFeedPayload (response.ts:982) validates against it. Go's FeedEntry comment says Activity is 'empty on posts', which is already wrong. |
| Unused communication kind 'finish_incomplete' in the frontend | `frontend/spaces/sodaspaces-factory-feed-families.ts:183; sodaspaces-factory-feed-communication.ts:16; sodaspaces-factory-feed-thread.ts:46-48` | yes | finish_incomplete settles an assignment through mcp_settle.go settleIncomplete. It is not an MCPCommunicationTool, Go never builds a FeedCommunication with that kind, and no store code mentions it. The frontend references are dead and can be removed: response.ts:36, communication.ts:16, families.ts:185/1286, thread.ts:48, view.ts:147 and actions.ts:72. |
| Judge-input candidate merge is thrown away | `internal/web/api/factory_feed_comment.go:613-679; internal/web/api/factory_feed_comment_context.go:30-104` | no | Partly true. The production judge (TypeSafeCommentJudge.PrepareCommentInput) replaces Summary and Status with native issue data and adds every visible native issue in Forgejo order. Recent store candidates only add a WorkerStatus via the old.Status lookup. Some of the merge still survives: the bound/source/parent order and Origin, and Status as WorkerStatus, though only for the first 8 because of commentCandidateLimit. The Summary fill and the recent-work ordering are wasted. Simplifying keeps behaviour only if the WorkerStatus lookup is kept. |
| Re-validating values the store just built | `internal/factory/feed_communication.go:158-215; internal/store/factory_feed_comment.go:316-323,620,652,833` | yes | factory_feed_comment.go:616/635/645 validate CommentSource structs built from the store's own rows a line earlier. Line 316 validates a stored source and line 829 validates candidates scanned from rows. The API layer then validates candidates again (collectCommentJudgeInput, prepareNativeCommentInput). These checks are redundant for well-formed data. The only difference after removing them is that a corrupt row would no longer raise an error. |
| Output-viewer handshake echoes server-known identity | `internal/web/api/factory_output.go:118-145` | yes | validFactoryOutputHandshake compares the client's run_id, repository_id and session_generation against identity values the server already has. The clients are sodaspaces-factory-request.ts:89 and terminal-attachment.ts:185, which shares the pattern. Only cursor carries information. The one case that would change is a stale page whose binding generation no longer matches the cookie. factoryViewerCurrent already rechecks authority continuity, so the effect is practically identical. |
| Alias constant | `frontend/spaces/sodaspaces-factory-feed-response.ts:14-15` | yes | feedPageMaxEntries = feedPageLimit, and its only use is response.ts:1276. Inline it. |
| Process-tracker comments in code | `internal/factory/feed_contract.go:10-26,59-76; frontend/spaces/sodaspaces-factory-feed-response.ts:4-8,21-25` | yes | feed_contract.go:10-26 (R42 preparation, R32), :59-76 (Lane B2/B3/C) and response.ts:4-8 (Lane B release, R42.1) and :21-25 (Lane B3) cite planning lanes rather than behaviour. They are comment-only. Part of the :59-76 text is also stale: it says Activity is empty on posts. |

### Publication, merge, correction, repair, acceptance, verification, lifecycle

| Item | Where | bp | Note |
|---|---|---|---|
| SupersededByBase has no reference | `internal/factory/merge_operation.go:503-509` | yes | Only the definition at merge_operation.go:507 exists; no callers anywhere, tests included. |
| MergeUnionReadyInput left behind after CheckMergeUnionReady was deleted | `internal/factory/control/merge_authorization.go:219-234` | yes | The only hit is the type definition at merge_authorization.go:226. |
| AdmitAndJournalCorrection has no callers, not even in tests | `internal/factory/control/correction_result.go:269-305` | yes | The only hit is the definition at correction_result.go:274. |
| Resolution helpers survive only through test-only AdviseMergeJudgment | `internal/factory/control/correction_resolution.go:34-181` | yes | PlanAcceptedResolutions, JudgmentAcceptance, ReconcileAcceptedResolutions and AcceptedResolution are referenced only by merge_judgment_gate.go and its test. AdviseMergeJudgment is called only from merge_judgment_gate_test.go. They should be removed together with the gate file. |
| DefaultRepairGrant is never used in production | `internal/factory/stall.go:177-185` | yes | Only stall_serve_test.go:96 and :385 call it. Deleting it means rewriting those tests to build the grant inline or dropping them. Wiring it into production would be a fix (DLV-8), not a removal. |
| accumulateMergeSettleTx can never find its merge | `internal/store/factory.go:406-427` | yes | It is passed the owner assignment ID, the same value accumulatePublicationSettleTx matches against factory_publications.assignment. factory_merges.publication holds the publication ID (NewID, publication.go:287), and every other query joins on p.data->>'id'. So it always returns no rows and does nothing. This is a latent bug: either delete it (identical behaviour) or fix it by joining through publications (changes behaviour). |
| Unreachable fallback to PublishCorrection in publishCorrectionChild | `internal/factory/control/publication.go:181-191` | yes | PendingPublicationChildren requires reported and completed results with source_commit equal to the candidate. The producer only sets wire.phase to ExecutionCorrect for coder children. So the payload-less branch only ever reaches the correction_payload_missing error, and the PublishCorrection fallback is effectively unreachable. PublishCorrection itself is still used by correction.go:486 and :548. |
| adoptPublicationReceipts writes a new revision on every pass | `internal/factory/control/publication_reconcile.go:401-429` | no | storePublication (Revision++ then UpdatePublication) runs unconditionally at the end, even when nothing changed, and already-committed ops are re-adopted on every pass. Skipping the write when nothing changed keeps the logic the same but stops the revision churn and drops the implicit optimistic-concurrency check, so it is not strictly identical. |
| A fresh publication exports the candidate twice and validates the bundle three times | `internal/factory/control/publication_reconcile.go:155-161; internal/forgejo/publish.go:48-71,98-131` | yes | resolvePublicationExport calls exportCandidate on every pass whatever op.Kind is. ObservePublication runs PrepareValidated on the bundle for both the publish and PR-create ops, and the PushBranch path runs PrepareValidated again. PR creation does not use the bundle. |
| The Feed is used as the source of truth for state | `internal/store/factory_delivery_admission.go:163-173; internal/store/factory_repair_scope.go:81-131` | no | The lines are slightly off: the verified-scope check at factory_delivery_admission.go:178-185 joins factory_feed_comments on component_id 'F10'. factory_repair_scope.go:81, 87 and 131 join MCP submissions to feed posts through receipt feed_post_id. Real coupling, but fixing it is a redesign rather than a removal. |
| ApplicationVerificationEvidence is computed three times per verification | `internal/factory/control/delivery_admission.go:70,157; internal/factory/control/app_verification.go:117` | no | Called at delivery_admission.go:70 and :157 and at app_verification.go:117, plus delivery_repair.go:238 and the API at factory_mcp_verification.go:21. The calls happen at different times, so caching changes how fresh the evidence is. A reasonable simplification, but not strictly identical. |
| Six near-identical command replay decoders | `internal/factory/control/acceptance.go:217; acceptance_status.go:413; lifecycle.go:74,313; lifecycle_retry.go:252; lifecycle_takeover.go:87` | yes | All six open with the same Finished=="" check returning ErrCommandRunning, then decode the stored result into a different type. A generic replay[T] would remove the duplication. Low value. |
| control.CorrectionReport shadows factory.CorrectionReport | `internal/factory/control/correction.go:43` | yes | control/correction.go:43 is the pass outcome and factory/publication_correction.go:134 is the worker payload. publishCorrectionChild uses both in the same function. Renaming is cosmetic. |
| Journey readback is a second audit read model beside the Feed | `internal/web/api/factory_journey_readback.go; internal/store/factory_journey_readback.go` | no | 658 API lines and 406 store lines confirmed. I could not confirm the 795 TS lines because there is no frontend/src path. It is a separate read model, and removing it would drop a user-visible API and UI, so this is a product decision rather than a behaviour-preserving removal. |
| Children producer runs several times per cycle | `internal/factory/control/review_cycle.go:35-39,73-74; internal/factory/control/publication.go:101` | yes | producePublishedChildren is called from publication.go:101 (PublishPass) and review_cycle.go:37 (CheckPass when Checks is nil), :74 (CheckPass end) and :108 (progressAfterPublish). The producer is idempotent and adopts existing attempts, so removing the duplicate calls should not change what ends up recorded, only timing within a cycle. |
| Stale comments | `internal/factory/control/merge_authorization.go:25-28; internal/factory/lifecycle.go:114-118` | yes | The AuthorizeMerge comment says no production caller binds, but delivery_producer.go:133 (authorizeCleanDeliveryMerge) calls it. The RetryDecision comment says a later scheduler consumes it, but store PendingDeliveryRetry and factory_candidate_preparations.go:660 consume it now. Only a partial file subset was checked for planning codes: 40 FW/D/T code hits there, so 'about 80' across the full scope is plausible. |

### Fountain extension and its backend

| Item | Where | bp | Note |
|---|---|---|---|
| Unused socket env default | `cmd/soda-extension/main.go:20` | yes | main.go:20 is the only place that reads SODA_EXTENSION_SERVICE_SOCKET (README:19-20 also mentions it); both run wrappers pass --soda-socket=/ipc/soda.sock, so the env default does nothing. |
| Dead 15-minute proxy header timeout | `internal/web/api/extension.go:27,37-40` | yes | extensionResponseHeaderTimeout = stopOperationTimeout(14m)+1m. Fountain's extensionRequestContext sets a 30s WithTimeout on non-websocket requests (websocket.go:41-46), so this window can never be reached. The 10s-sleep test (extension_test.go:23-83) can be deleted with it. stopOperationTimeout is defined in provisioning.go and is used elsewhere, so keep it. |
| Operator extension-manager relay validates twice | `internal/web/api/extension_manager.go:20-38` | yes | The api handler checks ManagerControlBounds, MaxBytesReader(limit) and json.Valid, then the forgejo layer's only caller repeats the bounds, length and json.Valid checks (extension_manager.go:36-38). Keep the HTTP-facing checks (404, 405, content-type) in api and drop the repeat, or the reverse. |
| Resume generation checked three times | `frontend/spaces/soda-extension.ts:12-14` | yes | Fountain's checkSession only calls resume({sessionGeneration: generation}) after liveSessionMatches. Its owner.resume also retires on a mismatch (extension-mount.js), so the entry checks and the five resumeGenerationRefused component checks can never fire. |
| Pass-through transport wrapper | `frontend/spaces/soda-extension.ts:64-105,111-120` | yes | The host's mountContribution already throws on request/stream/navigate once retired or disposed (isTerminal) and always sets contractVersion=1, so the disposed flag and version check repeat host behaviour. Keep the Soda-specific pageId/ownerId/repository validation. websocket() is just stream().socket. |
| Hand-pinned browser contract mirror | `frontend/spaces/fountain-browser-contract.d.ts:1-6` | yes | The mirror body is currently byte-identical to ../forgejo-ext/sdk/browser/extension-contract.d.ts (diff is empty). Nothing checks the sha256; the only check is the includes('ownerId'/'navigate') test. go.mod already depends on ../forgejo-ext/sdk through replace, so importing the SDK .d.ts directly fits the existing setup. |
| Two hand-kept style lists cross-checked by regex | `scripts/build-soda-extension.ts:14-22,41-57` | yes | The styles map in the build script and the `const styles = [...]` arrays in all three entries are kept by hand, and checkStylesUsedByEntries compares them by regex over the entry source. A single declaration removes the check. |
| CSS post-processors around a shared stylesheet | `scripts/build-soda-extension.ts:102-196` | yes | Only half right. Splitting the :root tokens out of components.css does make stripDocumentTokens unnecessary. But restoreNativeSchemeCalls would still be needed: frontend/spaces/sodaspaces-workspace.css uses light-dark() outside :root (lines 446 and 1004-1007), so the Bun light-dark workaround stays unless those uses also change. |
| Hand-bumped cache epochs already inconsistent | `assets/branding/forgejo/components.css:1-2` | yes | palette.css is imported with ?v=2026-09-12.brutalist-2 in home.css and theme-soda-{light,dark}.css but with ?v=2026-10-19.role-colors-1 in components.css. More hand-kept epochs exist: ?v=17-square-tabs and build-forgejo.ts presentationVersion parsed from header.tmpl. Unifying them changes only cache URLs. |
| Duplicated terminal peer registration | `internal/web/api/extension_terminal.go:50-66` | yes | extension_terminal.go:53-62 repeats registerExtensionTerminalPeer with the same lock, closed/128-limit checks, map init, TerminalPeer insertion and wg.Add. The one difference is that the stopping check is skipped for action=='inspect', so a shared helper needs a flag for that. |
| User upsert plus re-read on every request | `internal/web/api/extension_native.go:122-133` | yes | store.UpsertUser (INSERT ... ON CONFLICT DO UPDATE SET login) is followed by store.User SELECT in both extension_native.go:122-133 and auth/extension_service.go:185-199. One INSERT ... ON CONFLICT ... RETURNING id,login,name would replace both queries. |
| Stale or contradictory extension docs | `docs/reference/forgejo.md:26-70,140-146` | yes | forgejo.md:148-150 ('builds its patched executable') contradicts the Containerfile header ('Soda never compiles Fountain'). frontend-vm-update.md refers to soda-extension-install.service, which tests assert is not-found, and to /usr/share/soda/packages/catalog.json, while forgejo.container uses /usr/share/soda/extensions/catalog.json. The /workspace page and Workspace panel are still declared in extension.json, so those docs match the code; they are stale only against the owner's no-Workspace decision. The README's lifecycle controller does exist (internal/web/extension.go:21). |

### Documentation

| Item | Where | bp | Note |
|---|---|---|---|
| go-packages.md overlaps go.md | `docs/development/go-packages.md:8-10` | yes | Lines 8-10 defer to go.md wherever the two overlap. It is 148 lines and linked only from development/README.md:21 and docs/README.md:75, so merging it means updating two index links. |
| Prose feed card count disagrees with the executable catalog | `docs/design/factory-feed-components.md:6; internal/factory/feed_components.json` | yes | The doc says 'All 121 numbered variants', but feed_components.json has 122 unique numbered ids (S16, PL11, AR11, BU11, IN11, FX11, F11, ST11, TP12, K8, LA9). The doc's number has drifted from the JSON. |
| Workflow contract still documents the 32-hex-only submission_id that the demo tripped on | `docs/development/factory-workflow-contracts.md:1396` | yes | On main the doc (line 1396) matches the code (factory_mcp.go:97 still says 32-hex). prod-hardening's mcpSubmissionID (factory_mcp.go:1156) accepts UUIDs, so the doc goes stale only when that branch merges. Fix it then. |
| research/factory-capability-map.md calls itself a non-normative snapshot yet trust.md links its 'mechanism comparison' as the rationale | `docs/research/factory-capability-map.md:10-14; docs/architecture/trust.md:701` | yes | Lines 10-14 call it non-normative, and trust.md:701 cites its #conditional-operation-mechanism-comparison anchor as the reason for a deferral. Other docs also link to it (architecture/overview.md, docs/README.md, development/testing.md), so deleting it means pruning those links too. |
| native-support.md keeps a 'Historical audit reference' table that is explicitly not a checklist | `docs/development/native-support.md:547-573` | yes | Lines 547-573 call the table historical, 'not a current checklist' and 'not a second product readiness gate', and point to git show 9f3baa7:docs/native-porting-audit.md. Delete it; the <a id=remaining-validation> anchor goes with it, so check that anchor for inbound links first. |
| docs/operator/enroll-key.md is a lone file in its own directory | `docs/operator/enroll-key.md` | yes | docs/operator holds only enroll-key.md. Three links point to it (guides/operator-setup.md:19, guides/media.md:56, cmd/soda-install/README.md:58); moving it into guides/ means updating those three. |
| Release-tool READMEs repeat release.md's commands | `lib/soda-release-tools/README.md (358 lines); lib/soda-release-deliver/README.md (149); tools/release-assets/README.md (142)` | yes | The soda-release-tools README has a short command table, then about 250 more lines of contract prose (collector config, qualification, admission) that overlap release.md (2529 lines, which names soda-release 40 times). It can shrink to the table plus links. The other two READMEs are similar but smaller. |
| Hardening branch adds another tracker doc | `prod-hardening: docs/development/production-gaps.md (+106), docs/development/README.md (+1)` | yes | prod-hardening adds production-gaps.md (+106) and one index row in development/README.md:44, while main still has factory-remaining-tasks.md and the factory-remaining-tasks/ directory. When DOC-1 runs, keep only one of the two. |

### Store (PostgreSQL layer) structure

| Item | Where | bp | Note |
|---|---|---|---|
| Exported DDL and table-name constants used only inside store | `internal/store/factory_completions.go:15-70, internal/store/factory_mcp_submission.go:27-62, internal/store/factory_mcp_capability.go:25-46` | yes | Nothing outside internal/store uses store.Factory*DDL/*Trigger(s) or TableFactory*. 11 const declarations in the three files plus the 2 TableFactory names; schema.go is the only real consumer. They can be unexported or inlined. |
| Stale process comments and dangling evidence references | `internal/store/factory_mcp_submission.go:13-14,55-62; internal/store/factory_mcp_capability.go:12-13` | yes | FactoryMCPAssignmentGuardDDL is in schemaStatements (schema.go:156), yet its comment still says tests apply it 'until registration lands'. evidence/fw38/ does not exist, and both files cite it. Ticket and fw tags are spread across many store files. Comment-only change. |
| Error sentinels with no errors.Is caller | `internal/store/*.go (e.g. ErrMCPSubmissionHandoffCap, ErrRunViewConflict, ErrPlanConflict, ErrDispatchConflict, ErrGrantKey)` | yes | Exactly 21 exported store Err* have no non-test caller outside internal/store, e.g. ErrDispatchConflict, ErrPlanConflict, ErrRunViewConflict, ErrMCPSubmissionHandoffCap, ErrGrantKey. A few are checked inside store or in tests (e.g. ErrGrantKey in grants.go), so unexport them instead of deleting them all. Error text stays the same. |
| Write-only projection columns | `internal/store/factory_mcp_capability.go:33-42; internal/store/schema.go:72` | yes | generation, project, preparation_digest and authority_digest show up only in the INSERT at factory_mcp_capability.go:83. factory_dispatch_regs.revision is only inserted (factory_grants.go:233), and the reads select data, id or seq. Renewals are only count(*)'d at application_setup.go:247. The schema validator's column probes (schema.go:350) must be changed along with them. |
| Duplicate run status flags | `internal/store/schema.go:54-59; internal/store/factory.go:64` | yes | SaveFactoryRun sets active=(r.Outcome=="") and settled=r.Reconciled (factory.go:64-76), so the columns mirror JSON. They are read in about 5 places: factory_context_current.go:65, factory_completion_material.go:431, factory_delivery_repair.go:132 and factory_feed_contribution.go:275. Each needs a JSON expression in its place. |
| Single-valued state column in the comment pipeline | `internal/store/schema.go:90; internal/factory/feed_communication.go:288-290; internal/store/factory_feed_comment.go:376-381` | yes | Validate rejects any Intake other than CommentIntakeRecorded (feed_communication.go:288). intake is a column, and it is a CAS key at factory_feed_comment.go:381. |
| Multi-row table that only ever has one row | `internal/store/factory_priority_inventory.go:30` | yes | Every read and write of factory_priority_scans/candidates uses PriorityInventoryScanID="current" (factory_priority_inventory.go:676-737). The scan key is a constant. |
| Feed ordering borrows another table's sequence | `internal/store/schema.go:88,102,133-150; internal/store/factory_completions.go:26` | no | These draw feed_seq from factory_assignments' sequence: comments (schema.go:88), completions (factory_completions.go:26), and merges and judgments via the soda_feed_*_order plpgsql triggers (schema.go:133-150). Replacing this changes how events are ordered across tables, so leaving it to the Feed rework is right. |
| Schema version literal written four times | `internal/store/schema.go:11,22; cmd/soda-identity/src/schema.rs:3,16; internal/store/application_schema_test.go:52-66` | yes | 69 appears in schemaFormatVersion, the INSERT seed in schema.go, SCHEMA_VERSION and its seed in schema.rs, and TestSchemaSeedMatchesVersionConstant, which checks that they agree. schema.rs also has an UPGRADE_68_TO_69 migration. |
| Coupled magic bound | `internal/store/factory_intents.go:270-276; internal/store/schema.go:37` | yes | The Go check maxCalls<1\|\|>18 (factory_intents.go:274) and CHECK(max_calls BETWEEN 0 AND 18) (schema.go:37) carry the 'move both bounds together' comment. The Go check already guards every write, so the bound can live in one place. |
| Misnamed lock row | `internal/store/schema.go:103-104; internal/store/readiness_work.go:68-398` | yes | factory_readiness_budget is a single row (id=1) that is locked FOR UPDATE and holds generation and next_turn, which is incremented (readiness_work.go:68-142). It works as a mutex plus a counter. Renaming it changes nothing observable. |
| JSON-null arrays fixed per call site | `prod-hardening docs/development/production-gaps.md; internal/store/factory_publications.go:494-630` | no | production-gaps.md exists only on the prod-hardening branch (lines 74-80 describe jsonb_typeof guards and a proposed lint). Main has no jsonb_typeof in factory_publications.go; it uses COALESCE(data->'corrections','[]') at :499. Normalising nil slices to [] fixes the bug once, but it changes stored JSON and error behaviour, so it is a fix, not a removal. |
| SQL round-trip just to compare two JSON documents | `internal/store/store.go:57-61` | yes | sameJSONDocument (store.go:57-61) runs SELECT $1::jsonb=$2::jsonb. It has 12 non-test call sites, not 10. Comparing the decoded Go values gives the same result for these payloads. |
| publicationAuthority and mergeAuthority are near-identical | `internal/factory/control/publication_authority.go:12-55; internal/factory/control/merge_evidence.go:130-170` | yes | The bodies are line-for-line the same apart from the report/wait/error helpers. mergeAuthority also compares BaseRef and ReviewerID, so a shared helper would need the extra policy comparison passed in. |
| Backup and restore machinery for a zero-user appliance | `cmd/soda-pg-maintenance/src/bin/soda-pg-backup.rs:28-143; system/host/services/soda-postgres-backup.timer` | no | The facts check out: nightly 02:00 timer with Persistent=true, SODA_PG_KEEP default 7, renameat2 RENAME_NOREPLACE (:149-154), and a globals CREATE ROLE check (:82-83). This is a scope freeze, not a removal; removing it would lose backups. |
| Docs to update alongside STO-1 and STO-11 | `internal/store/README.md:24-36` | yes | README.md:24-36 documents that opening refuses incompatible databases without migrating, plus OpenEncrypted and SchemaVersion(). Docs-only change, tied to the other items. |

### Rust host: factory worker runtime

| Item | Where | bp | Note |
|---|---|---|---|
| soda-factory emulates Go flags/JSON and requires a hand-made command ID | `cmd/soda-factory/src/main.rs:43-105,129-197` | no | main.rs:46-105 is a hand-rolled Go-style flag parser and encode_envelope uses a GoFormatter for <>&/U+2028 escapes that a JSON decoder ignores. Dropping the escaper and using plain serde_json gives the same result on the server. Auto-generating --command changes the CLI contract and drops operator-chosen idempotent retries, so that part is a small interface change. |
| Pass-through re-export modules in the factory role helper | `cmd/soda-project-terminal/src/factory_roles/mod.rs:13-39` | yes | ops_inspect, ops_record and proc (mod.rs:21-34) only `pub(crate) use` items from execution/records/fsx, and the #[path] renames keep Go-era names. Callers can import the real modules directly. This is a pure refactor. |
| Takeover re-checks a path it just built | `lib/host/src/factory/inspect.rs:56-84` | yes | inspect.rs:57-66 formats /home/{role}/checkouts/{prep} and passes it to takeover_source, which re-validates role/prep (already checked by receipt.run.validate(), run.rs:286-290) and compares the string with the same format, so it can never fail. takeover_source is also defined twice (factory/run.rs:200 and terminal/factory/artifacts.rs:26). Caveat: result.validate() also checks the destination returned by terminal.takeover_copy, so that part is not purely tautological. |
| Hand JSON escaper for a compile-time instruction string | `lib/host/src/terminal/factory/mcp_relay.rs:134-188` | yes | instruction_argv has one caller (muse/commands.rs:24) and it passes a constant ASCII string with .expect(). The Codex route and toml_string_body are never reached through it. For this constant, serde_json::to_string produces identical bytes. |
| Qualification ledger embedded in code | `lib/host/src/terminal/factory/mcp_relay.rs:1-80` | yes | mcp_relay.rs:1-80 cites P6/FW33/lane-b19/CHECKUP-012/D14/C11 and 'P6/P7 gates stay OPEN', tasks.rs:1-17 cites D05/FW14/FW24, and launch.rs:52 refuses with 'until its own proof passes'. Removing the comments changes nothing. Rewording the refusal text only changes an error string. |
| Root-only daemon keeps a systemd --user branch | `lib/host/src/terminal/factory/native.rs:33-73` | yes | factory_systemctl and factory_systemd_run (native.rs:173-185) pass current_euid(). serve_host_socket calls require_host_socket (main.rs:357/530), which refuses non-root, so the euid != 0 branch and factory_user_bus never run in the daemon. |
| Second private-file implementation | `lib/host/src/factory/execution_state.rs:18-99` | no | execution_state.rs:41-98 repeats the temp + fsync + rename + dir-sync write and the nlink/uid/mode read checks of PrivateState::write_bytes/read_bytes. Switching changes details: PrivateState adds the ancestor walk, requires state_dir mode &077 == 0, has a different size bound (journal bound vs MAX_EXECUTION_MANIFEST 64 KiB) and different error text. Close but not byte-identical. |
| Ancestor-walk check duplicated | `lib/host/src/muse_listener_state.rs:9-40` | yes | owned_directory (muse_listener_state.rs:9-40) is the same openat walk as PrivateState::open (checkpoint/state.rs:20-48), minus PrivateState's final service-private check on the leaf. A shared walk helper would preserve behaviour apart from error strings. Calling PrivateState::open directly would add the leaf check. |
| Run lifetime bounded five ways | `lib/host/src/terminal/factory/muse/reserve.rs:65` | no | reserve.rs:65 refuses max_secs outside 60..=10800 on top of FACTORY_DEADLINE_BOUND_NANOS (run.rs:135, launch.rs:58) and RuntimeMaxSec (native.rs:82). The marker-wait counter bounds a different wait (the marker before start). Removing the 60 s floor would let short deadlines that are refused today go through, so behaviour changes slightly. |
| Test-mode switch inside the root helper | `cmd/soda-project-terminal/src/factory_roles/mod.rs:8-11,56-127` | yes | Ctx::from_env (mod.rs:91-102) honors SODA_FACTORY_DIR/SODA_FACTORY_GIT and makes priv_uid/priv_gid return the euid/egid. Only cmd/soda-project-terminal/tests/factory_roles_oracle.rs and tests/build/project_factory_roles_fixture_test.go set them. With the variables unset, production behaviour is unchanged, and those tests could be deleted along with the seam. |
| Nested Compose Muse registration path | `lib/host/src/muse/nested.rs:1-252` | no | Neither the factory nor internal/ uses register_nested. However, it backs a shipped, documented manual feature: the soda-identity-compose binary is installed in system/project/Containerfile, listed in README.md:119 and described in docs/guides/project-services.md:31-41 'Muse opt-in'. Removing it drops that feature, so it is a product decision rather than a cleanup with no visible change. |

### Rust host platform: daemon, terminals, project, application, tailnet (lib/host non-factory, soda-project-maintain, forgejo/pg/console helpers, soda-tailnet, unix-http, wire-time)

| Item | Where | bp | Note |
|---|---|---|---|
| BackendError variants reused for unrelated meanings | `lib/host/src/daemon/routes.rs:262-280, lib/host/src/dbackend/tailnet.rs:49-63, lib/host/src/dbackend/backend.rs:84-106` | yes | map_tailnet_err picks a variant by substring (conflict->ExportStale 409, unsupported->ExportCandidate 422), and map_setup_claim_err matches exact strings (committed->ExportStale, conflict->OutputStale). Real variants with the same status and body would keep wire output identical. |
| Go client re-validates trusted daemon replies | `internal/host/tailnet.go:52-171` | yes | admitEnrollmentSave/SaveRotate/Action, tailnetProjectConfirmed and the TailnetPolicy project/saved check (internal/host/tailnet.go:52-171) re-check replies from the root daemon. With a correct daemon, nothing observable changes if they go. |
| soda-tailnet Go binary exists only to print status in the console welcome | `cmd/soda-tailnet/*.go, cmd/soda-console-welcome/src/welcome.rs:110-112` | yes | cmd/soda-tailnet (88 Go lines plus a README) calls TailnetSettings and prints status and guidance. The only consumer is welcome.rs:110-112, plus the cli_surface fixture and the tests/installed/host.sh command check. Folding it into soda-console-welcome keeps the output; dropping it changes the welcome text. |
| Offline Forgejo writer-domain recovery CLI | `cmd/soda-forgejo-domain/** (679 lines incl. README), docs/guides/operator-setup.md:360-386` | no | 679 lines and operator-only. No unit or service calls it. References: the release image staging (build_compile.rs), tests/build/forgejo_domain_test.go, operator-setup.md, 80-recovery.md and README. Removing it drops a documented operator command, so behaviour changes, though only for a manual recovery procedure. |
| Origin and clean-path checks on a root-owned private Unix socket | `lib/host/src/daemon/admission.rs:16-50,162-205` | no | NATIVE_CLEAN_PATHS (33 paths), plus no-Origin checks for identity/terminal and has_query/escaped checks, copy Go net/http edge cases. One universal check would change status codes for malformed requests (for example an unknown path with a query gets 400, not 404; Origin no longer refused). No real client sends these. |
| Config deserializer keeps Go's case folding, last-wins and trailing-garbage tolerance | `lib/host/src/daemon/config.rs:55-128,369-387` | yes | A hand-written Visitor does eq_ignore_ascii_case, treats null as a no-op and lets the last key win; load_config ignores trailing data on purpose. derive plus deny_unknown_fields behaves the same for well-formed configs written by setup, and only rejects malformed ones. I did not verify the '4 fields' count. |
| Executor trait run_bounded/run_bounded_typed defaults exist for test executors | `lib/host/src/project/executor.rs:16-90` | no | Four methods (run, run_bounded, run_bounded_typed, run_with_capture_grace). The only production impls are Native and NativeStatusOnly, which override them, so the defaults serve only test fakes in lib/host/tests and cmd tests. Dropping the defaults is safe. Collapsing to one bounded run would make about 17 unbounded run() call sites bounded unless they pass unlimited caps. |
| Two websocket helper files | `lib/host/src/daemon/websocket.rs:1-14, lib/host/src/dbackend/websocket.rs` | yes | daemon/websocket.rs is 14 lines wrapping tungstenite create_response; its only user is routes.rs:13. dbackend/websocket.rs is an unrelated 624-line terminal pump, so only the small file should be inlined. Trivial win. |
| application::home_path/storage_root are pass-through re-exports | `lib/host/src/application/mod.rs:10-15` | yes | These forwarders exist because factory/application.rs:121 calls crate::application::home_path; the other callers are application submodules. Replace with `pub(super) use home::{home_path, storage_root};` or make home pub(super). Purely cosmetic. |
| Body limits split into three constants plus a lifecycle-specific limit | `lib/host/src/daemon/admission.rs:5-10,163-174` | no | body_limit_for picks 1 MiB / 384 KiB / 16 KiB (tailnet prepare/lifecycle) / 64 KiB. tcontrol_wire also checks LIFECYCLE_BODY_LIMIT itself (line 1498), so the admission branch for it is redundant. One 1 MiB cap would change which oversize bodies get 413 at the edge, but only for bodies no real client sends. |

### Web API and view models (non-MCP, non-Feed, non-extension): internal/web/*.go, internal/web/auth/**, internal/web/api/*.go excluding factory_mcp*, factory_feed*, extension*

| Item | Where | bp | Note |
|---|---|---|---|
| Snapshot adapters are duplicated and live in the HTTP package | `internal/web/api/issue_acceptance_evidence.go:15-110; dispatch_inputs.go:12-96; factory_readiness.go:9-69; factory_completion.go:27-87` | yes | AcceptanceSnapshotSource and DispatchSnapshotSource are near-twin BracketedRead adapters. ServiceReadinessSource only nil-checks and forwards, and NewServiceReadinessSource builds both from the same observer. Merging them preserves behaviour. completionPageReader's per-call budget charge is real accounting, so it has to be kept or dropped on purpose. |
| Creation-profile picker with only one possible profile | `internal/web/api/project_profiles.go:14-43; environments_create.go:34-45,92-112` | yes | apiProjectProfiles always returns []Profile{ResolveProfile()}, and create accepts only '' or RockyHeadless. The frontend calls /profiles (sodaspaces-project-refresh.ts:216) and sends profile_id (project-request.ts:138), so those calls must be removed in the same change. |
| GET handlers that write (crash recovery on read) | `internal/web/api/environments_api.go:139-147; lifecycle.go:149-155; provisioning.go:54-69` | no | Environment GET and the lifecycle inspect call reconcileProvisioning, which runs Store.MarkReady. Without it, an interrupted creation would no longer be finished by a refresh. It is speculative recovery, but removing it changes behaviour. |
| Request body bounded four times and strict-decoded by two decoders | `internal/web/api/extension.go:79-81; extension_terminal.go:38-40; extension_native.go:111-119; auth/http.go:59-80; internal/strictjson/decode.go:17-37; factory_lifecycle_actions.go:128-139,220-228` | yes | MaxBytesReader(APIBodyLimit) is applied in extension.go:82, extension_terminal.go:39 and extension_native.go:117, and strictjson then adds a 1 MiB limit, a UTF-8 check and a duplicate-key scan. decodeLifecycleConfigSave is a second DisallowUnknownFields decoder. The duplicate limits can go with no visible change; dropping the duplicate-key scan only changes handling of malformed bodies. |
| HTTP method declared twice per route | `internal/web/api/extension_native.go:22-26,176-287` | yes | Every mux pattern names its method, so the mux already returns 405 and AllowAPIMethod in extensionProtected is redundant. The only difference: HEAD on a GET route gets a JSON 405 today. The r.Method branches inside handlers are real dispatch, because one handler serves both GET and POST; only the protect() method list is duplication. |
| User row upserted and re-read on every request | `internal/web/api/extension_native.go:122-133; internal/web/auth/extension_service.go:185-203` | yes | Both extensionProductUser (api) and extensionUser (auth) run UpsertUser and then Store.User on every request. One UPSERT ... RETURNING, or skipping the write when the login is unchanged, gives the same result. |
| Pass-through handler | `internal/web/api/repositories.go:101-108` | yes | apiRepositories only pulls out the authority and calls apiNativeRepositories, and that is its only caller, so the two can be merged. |
| apiError declared twice; four error envelopes | `internal/web/auth/http.go:17-20; internal/web/api/environments_create.go:18-21; issue_acceptances.go:200-208; factory_lifecycle.go:22-23; factory_settings.go:27-28` | no | apiError is defined in both auth/http.go and api/environments_create.go; removing the duplicate is safe. The envelopes really differ: acceptance refusals use {error:reason,message}, takeover uses {error:'pending_runs',runs,message}, a running command returns 202 {running:true}, and environment errors return {error:{...},environment}. Unifying them changes the wire format and needs matching frontend parser changes. |
| Intake webhook: Gitea header fallbacks and a one-implementation interface | `internal/web/api/factory_intake.go:25-45` | yes | intakeHeader falls back to X-Gitea-* for Delivery, Signature and Event. Forgejo sends X-Forgejo-* headers, so the fallback is dead. IntakeCoordinator is only ever given s.Coordinator in server.go:113. |
| Explore handler runs four paid judgments with a throwaway Coordinator | `internal/web/api/factory_intent_explore.go:62-80,121-243; api.go:26-28` | yes | The handler builds &control.Coordinator{Store, Planning} with an IntentExploreChooser override (production leaves it nil) and calls AdvisePurpose, JudgeAssumptions, JudgeConsolidation and JudgeConsistency. mintExplorationProposal mints 104 IDs per call. Moving this into control is a refactor; the judgments themselves are product behaviour. |
| Run status re-validates stored rows before rendering | `internal/web/api/factory_views.go:148-161,221-268` | yes | deliveryRepairProgressDTO compares about 7 cause/assignment fields and NaN/range-checks stored probability and confidence. The frontend checks input_sha against /^[0-9a-f]{40}$/ and then drops it (sodaspaces-factory-response.ts:372). Rows the system wrote itself would render the same without these checks. |
| Revision history knobs nobody uses | `internal/web/api/factory_revision_view.go:15-23,37-49,343-370` | yes | The only caller sends ?limit=2, which equals DefaultRevisionComparisons, and parseRevisionHistoryQuery exists only for that knob. unchanged_count always equals len(reused), and progress-view.ts:1227 prints both counts, so the frontend text must switch to reused.length. |
| Forgejo key list paged on the server | `internal/web/auth/extension_service.go:206-238; auth/forgejo_keys.go:7-18` | yes | The handler fetches all keys (refusing more than 100), slices 10 per page, and validates page=1..8 with parseForgejoKeysPage. The frontend calls /api/me/forgejo-keys?page=N, so that call must be updated in the same change. |
| Workspace contributions still admitted | `internal/web/auth/extension.go:45-53; internal/web/api/extension_native.go:139-150` | no | Both checks admit the 'workspace' panel and the 'spaces' page. extension.json still ships the 'spaces' page (titled Workspace, the main Soda entry with new and requests subroutes) and the 'workspace' panel. Removing them is the owner's product change: it needs the manifest and frontend changed too, and the spaces page content has to move to the repository page. |
| Test-only and stale bits in internal/web | `internal/web/server.go:225-235; internal/web/factory_mcp_listener.go:9-21,45-52` | yes | SetForgejo has no callers, and SetHost is called only from test_helpers_test.go:252. The 'fourteen-tool catalog' comment is still there. FactoryMCPHandler nil-guards s and s.API; this check did not confirm that they are never nil. |
| Crash-safety machinery for member access revoke/restore | `internal/web/api/member_access.go:84-140` | no | A process mutex, the MemberAccessFence startup fence, BeginMemberAccess with a revision CAS, the host call, FinishMemberAccess and a fence release wrap one host call. Removing them would remove the restart and interruption protection and the 409/202 pending responses. |
| Activity display filter stored with an optimistic revision | `internal/web/api/factory_activity.go:31-58` | yes | Three booleans that control only the local feed (factory/activity.go) are saved behind expected_revision with a 412 on stale_revision. Last-write-wins would look the same to a single owner. |
| GET /api/applications/{id} has no browser caller | `internal/web/api/applications.go:269-278; tests/installed/qualification/application.ts:164` | no | The frontend only calls the {id}/actions, snapshots, restores and project routes. The bare GET is used by several installed qualification scripts (application.ts:164, prepare-application.ts:188, docker-persistence.ts, prepare-final-recovery.ts), so removing it breaks those scripts unless they change too. |

### Boundary machinery: strict JSON, digests, receipts, revision checks (cross-cutting)

| Item | Where | bp | Note |
|---|---|---|---|
| Plan finish duplicates 'approach' at top level and inside tasks and refuses on mismatch | `internal/web/api/factory_mcp_issue_plan.go:64-67` | no | factory_mcp_issue_plan.go:64-67 only checks that top-level approach equals tasks.approach. The schema (factory_mcp.go:314 required, role_schema.go:55) asks the model to send it twice. Drop the top-level field and the check together. The only change is that the model can no longer hit this refusal. |
| foldInventoryPage takes repository and generation and discards them | `internal/factory/control/completion_inventory.go:176-177` | yes | completion_inventory.go:176-177 throws away `_, _ = repository, generation`. Both callers (completion_inventory.go:332, completion_decision.go:138) pass values that are never used. Removing the two parameters is pure dead code. |
| Two different PlanDigest functions with the same name in project and factory | `internal/project/factory_manifest.go:163; internal/factory/planning.go:57` | yes | project.PlanDigest([]byte) is a sha256 of the manifest plan bytes (manifest Validate, dispatch_launch.go:167). factory.PlanDigest(InvestigationPlan) hashes the versioned struct and is used by plan Validate, issue_plan and correction_plan. Renaming one is safe. Merging them would change the digest values. |
| MCPReceiptVerifying state used only by task checkpoints | `internal/factory/mcp_submission.go:136-141; internal/factory/control/task_checkpoint.go:134` | no | Only task_checkpoint.go uses it (set at line 197, read at 134 and 286), plus the validity switch in mcp_submission.go:238. It is a persisted 'pending' step while the checkpoint commit runs, so dropping it means redesigning how checkpoints settle. It is not a plain delete. |
| Context-current owner resolution re-binds ScopeDigest, Fingerprint and RequiredDigest to the capability | `internal/factory/context_resolution.go:15-38` | no | ValidateFor in context_resolution.go:33-38 matches ScopeDigest, Fingerprint (ContextSelectionSnapshot) and RequiredDigest against the capability. It exists only for the use-current-context owner command, so it goes when that feature goes. Removing it alone is not possible without dropping the feature. |
| Release crates are about 26k lines with ~820 sha256/digest hits for an unreleased product | `lib/release-inputs, lib/soda-release-build, lib/soda-release-deliver, lib/soda-release-tools` | no | Rust line counts: release-inputs 1410, release-build 3775, release-deliver 8887, release-tools 12548, about 26.6k total. lib/soda-release-image (8980) is not counted. The finder did not audit these, so leave them to the release/packaging finder. |
| soda-identity wire byte-identical to internal/identity Go records | `cmd/soda-identity/src/wire.rs:1; cmd/soda-identity/src/providers/resolution.rs:21` | no | wire.rs:1 says it is byte-identical to internal/identity, which still exists. resolution.rs:20-23 keeps explicit binary+sha256 legacy pins 'byte-for-byte' for fixtures and the broker compat test, though only store mode is routed now. The legacy-pin path is compat code. The Go/Rust wire match may still be needed if both sides exchange records. |
| Strict decoding of API requests from SodaOS's own frontend | `internal/web/auth/http.go:73` | no | auth/http.go:73 decodes with strictjson after the 64 KiB and object checks. The frontend should send no extra fields, so in practice nothing changes. A request with unknown or duplicate fields would no longer get a 400. Low value either way. |
| ParseExecutionManifest checks for a trailing token and unknown fields on bytes the same process marshaled | `internal/project/factory_manifest.go:79-98` | yes | factory_manifest.go:79-98. Every caller reads run.ExecutionManifest back from the store, and dispatch_launch.go:177 checks bytes it marshaled itself. Replacing DisallowUnknownFields and the trailing-token check with json.Unmarshal is safe. Keep the length bound and manifest.Validate(). |
| Delivery judgment material re-validates the plan and cross-checks ScopeDigest and AttemptRoot between two records built from the same plan | `internal/factory/control/delivery_judgment_material.go:66-67` | yes | validate() at delivery_judgment_material.go:66-67 runs on material just built in deliveryMaterial (producer.go:143 and 278, typesafe.go:24). The Plan.Validate, ScopeDigest and AttemptRoot conditions are self-consistency checks and can go. Keep the 256 KiB size bound, which is real behaviour. |

### Release, install, packaging, CI and freshness machinery

| Item | Where | bp | Note |
|---|---|---|---|
| Legacy worker home/run migration | `tools/candidate-setup/src/storage.rs:51-81, tools/candidate-setup/src/worker_tools.rs:292-295, tools/candidate-setup/src/main.rs:44-45` | no | storage.rs:50 migrate_candidate_home, which copies /var/lib/soda-candidate-home when it exists, is called from worker_tools.rs:275. The LEGACY_RUN check only prints a notice. Removing them changes behaviour only on a host that still has the old layout, which a zero-user product should not have (the owner's own lab host may). |
| Legacy separate-origin refusal | `cmd/soda-activate/src/activation.rs:97` | no | activation.rs:95 rejects any config that has a public_url key, and nothing else in the Go or Rust code mentions public_url. Removing the check means such a config is no longer rejected; fresh installs never write that key. |
| Exact '=' version pins and the reproducibility comment contradict freshness | `Cargo.toml:1-3, cmd/soda-install/Cargo.toml:12-30, lib/provider-tools/Cargo.toml:10-17` | no | Cargo.toml lines 2-3 say the lock is kept 'for reproducibility'. soda-install and provider-tools pin =1.0.229 serde, =3.27.0 tempfile, =0.2.190 libc and others, and mix them with loose ranges. Rewording the comment is free; loosening the pins lets resolution drift once the lock is updated. ssh-key =0.7.0-rc.11 is a pre-release, so it may genuinely need its exact pin. |
| Triplicated sha256_hex inside one crate | `tools/release-assets/src/render/mod.rs:76, tools/release-assets/src/locales/mod.rs:38, tools/release-assets/src/fetch/mod.rs:64` | yes | There are three pub fn sha256_hex in render, locales and fetch inside release-assets. The fetch copy uses hex_lower, but all three produce the same lowercase hex, so one shared helper gives identical output. |
| Copied process helpers | `tools/candidate-setup/src/process.rs, tools/lab-credentials/src/process.rs` | yes | The two process.rs files (182 and 204 lines) share status_code, flush_stdout, spawn_diag, Captured, capture, run, run_stdout_null and run_with_io with small differences. Captured::SpawnFailed carries a code in only one copy, and lab-credentials adds its JSON helpers. The two crates are separate binaries, so sharing needs a small common crate, and that may cost more than the duplication. |
| lab-credentials mostly prints runbooks | `tools/lab-credentials/src/runbooks.rs, tools/lab-credentials/src/main.rs` | no | In main.rs:64-97, the cloudflared-token, forgejo-runner and lab-vm-operator options only print constants from runbooks.rs (11 lines). Only fixture-authority --execute changes anything. Moving the text to docs changes CLI output but nothing that matters. |
| Two advisory complexity wrappers | `scripts/check-complexity.sh:1-5, scripts/check-ts-complexity.sh:1-5` | yes | check-ts-complexity.sh is check-complexity.sh plus '--language typescript'. Its only consumers are package.json check:ts-complexity, docs/development/typescript.md:86 and a comment in check-oxlint.sh. The package.json script can call check-complexity.sh --language typescript directly. |
| Rootfs server test reads another tool's source | `tools/soda-rootfs-server/main_test.go:1-20` | yes | main_test.go:50 runs a regex over tools/candidate-setup/src/main.rs to match ROOTFS_DIR against rootDir. TestUnitRunsTheSingleServerSource also checks unit text. TestExactFileServing is a real behaviour test and should stay. Deleting the text tests does not change runtime behaviour. |
| Unused re-export comment | `cmd/soda-install/src/sshkey/mod.rs:5` | yes | soda-install is a binary crate with no lib target, and nothing names sshkey::AuthorizedKey. The comment and #[allow(unused_imports)] keep an unused export. Dropping AuthorizedKey from the pub use, along with the allow, compiles to the same binary. |
| Fountain pin shape validated with jq in every CI job | `.github/workflows/ci.yml:76-79, .github/workflows/deep-ci.yml:57-60` | no | The same jq -e keys/regex check runs at ci.yml:79 and deep-ci.yml:60, 120 and 185. Removing the -e line leaves only the output line, so a malformed committed file would fail later at checkout instead of here. One shared check, or none, is enough. |
| Mixed checkout action versions | `.github/workflows/deep-ci.yml:113-117` | yes | deep-ci.yml:113 and 125 (the TypeScript and browser checks job, two checkouts) use actions/checkout@fbc6f39 # v5. Every other use in both workflows is v7.0.1 at 3d3c42e. Aligning them is a one-line fix each. |

### Budgets, allowances and limits (cross-cutting)

| Item | Where | bp | Note |
|---|---|---|---|
| Dead merge-judgment entry point | `internal/factory/control/merge_judgment_gate.go:266-304` | yes | AdviseMergeJudgment (merge_judgment_gate.go:274) is referenced only from merge_judgment_gate_test.go; there is no production caller, so the function and its tests can be deleted. |
| Connection-budget authority code is practically unreachable | `internal/factory/effective_authority.go:47-53` | yes | The budget row is inserted ON CONFLICT DO NOTHING whenever an active sponsorship is saved (internal/store/factory_grant_commands.go:437-447, not internal/factory/). No code deletes it, and saveSponsorshipQuery is the only sponsorship insert. So connection_usage_budget_missing only appears next to an inactive sponsorship. The missing check and its UI mappings (factory_access_view.go:146, settings.ts:1122) can go; keep the revision assignment into ref. |
| Identity service mirrors every budget table | `cmd/soda-identity/src/schema.rs:30-99` | no | cmd/soda-identity/src/schema.rs says it is 'Mechanically mirrored from internal/store/schema.go', and it holds factory_judgment_budget, the allowance tables, factory_connection_usage_budgets and factory_readiness_budget. This is a coordination cost, not something to delete: every budget-table change must be repeated there. Removing the mirror belongs to the identity subsystem. |
| Two planning-allowance posters | `frontend/spaces/sodaspaces-intent.ts:2122-2146` | yes | Both build {id, intent, predecessor, amount=ceiling=wanted, unit:'planning_calls', reserved, consumed, effect?} and POST it to factory/intent-allowances. They differ a little: factory-work sends reserved/consumed 0 when there is no current allowance, and the effect text is supplied by the caller. A shared helper is fine if it keeps each caller's effect wording. |
| Clock-regression and overflow guards in budget code | `internal/factory/allowance.go:42-44` | yes | Counts match: 8 ErrAllowanceClock references outside tests (1 is the definition, the rest are checks in allowance.go, stall.go, factory_attempt_allowances.go, factory_verification_allowance.go and factory_delivery_repair_budget.go) and 11 MaxInt64 references across the 4 named files. Behaviour only changes if the host clock goes backwards or minutes reach int64 limits. Where a guard is dropped, clamp negative elapsed time to 0. |
| Six per-pass readiness counters | `internal/factory/control/readiness_work.go:13-120` | no | readiness_work.go defines caps for pass time, pages (8), assessments (32), evidence reads (52), native RPCs (156) and selections (64), using CAS atomics. progress.go:126-131 breaks out of settling expired runs when reserveNativeRPC(5) fails. Collapsing the caps to just the time limit is a real simplification, but it changes per-tick behaviour under load. |
| Qualification script pins budget numbers | `tests/installed/qualification/prepare-funding.ts:64-124` | yes | prepare-funding.ts:69-71 asserts the existing budget is exactly 15 and the approved value exactly 25 before the PUT, then checks exact sponsor minutes and concurrency. The 15 does not match the product default of 360 (grants.go:332). This is test-harness ceremony only; product behaviour is unaffected. |
| Spend hidden per viewer on a single-owner appliance | `internal/web/api/factory_sponsorship.go:14-22` | no | sponsorshipView zeroes AllowanceMinutes and MaxConcurrent for non-owners, and accessUsageDTO returns connection_owner_only. Removing this changes what non-owner viewers see; it is only invisible if there really is a single owner, which I did not verify. |
| Settings shows internal revision numbers | `frontend/spaces/sodaspaces-factory-settings.ts:1155-1166` | no | Settings prints '(revision N)' in renderCapacitySummary (:1159), renderSponsorshipSummary (:1165) and the rolling-allowance row (:1593). Removing it changes the visible text only; the CAS revision is still kept in state for saves. |
| Active-attempt flag used as a second capacity count | `internal/store/factory_dispatch_packet.go:1183-1196` | no | checkAdmissionLimitsTx counts held reservations plus active runs, then counts again from the attempt allowances' active flag (Checkpoint true at factory_attempt_allowances.go:817). In normal operation this duplicates the run count. But the flag can stay set without a run (crash, missed checkpoint), and then this second count blocks admission. Removing it is not provably identical. |

### Other Go packages (forgejo, acceptance, project, host, tailnet, application, config, strictjson, filelock, avatar, cmd/soda-dashboard, cmd/soda-tailnet, cmd/soda-extension)

| Item | Where | bp | Note |
|---|---|---|---|
| mapRevisionError returns err on every branch, so it does nothing | `internal/forgejo/publish/operation_observation.go:70-81` | yes | Both errors.As branches and the fallthrough all return err unchanged; the two callers (lines 38, 60) can return err directly. |
| filelock package (plus a 46-line README) has one caller | `internal/filelock/filelock.go; internal/factory/control/coordinator.go:163` | no | Only coordinator.go:163 uses filelock.Acquire (36 lines plus a 46-line README). It polls LOCK_NB for up to 10s; a single LOCK_NB fails at once instead of waiting, so to keep identical behaviour, inline the small retry loop in coordinator.go. |
| FactoryPublicationRoot is optional and falls back to the private data dir | `internal/config/config.go:76-80; internal/web/server.go publicationRoot` | yes | The fallback is actually the same fixed path /var/lib/soda/dashboard/publication, hardcoded in three places: web/server.go:149, control/mcp_key.go:15 and soda-setup PUBLICATION_ROOT. soda-setup enforces that exact value (factory_enrollment.rs:475). Make it one constant and drop the config field, the fallbacks and the IsAbs check; the soda-setup config writer changes with it. |
| VerificationManifest.Checks always holds exactly one fixed check with fixed artifact and fixture names | `internal/project/factory_verification.go:46-72; internal/factory/control/verification_execution.go:161` | no | Validate requires len(Checks)==1 with constant CatalogID, Artifacts and Fixtures, and Check.Outcomes must equal m.Outcomes; the producer at verification_execution.go:161 always builds that one literal. The data is pure redundancy, but dropping it changes the manifest JSON and digest shared with lib/host, so lib/host must change in the same step. |
| Profile type with one allowed value of every identity field | `internal/project/profile.go:14-37` | no | Validate pins ID=rocky-headless, Distribution=rocky and Interface=headless; Version, Architecture, Image and Revision vary (Version is checked against a pattern too). Removing the fixed fields changes the stored and shared profile JSON (strict decode), so the consumers must change in step. |
| process_wait_other.go is a non-Linux stub that only refuses | `internal/acceptance/process_wait_other.go` | yes | A //go:build !linux file with three stubs that return an error. The Linux build compiles only process_wait_linux.go, so deleting the stub changes nothing on the product platform. |
| CapabilityHandshakeSource records the last handshake only for a log line | `internal/forgejo/lifecycle_capabilities.go:17-52; internal/web/server.go:218` | yes | The only LastHandshake caller is server.go:218, for the slog.Info line; the err and at fields are never read. Have ReconcileLifecycleCapabilities or the handshake return the installation and generation, or log inside Handshake, and drop the mutex and its state. Log output can stay the same. |
| forgejo.Client.decodeResponse rejects a literal JSON 'null' body and trailing data | `internal/forgejo/client.go:52-73` | no | Extra null and trailing-data checks on Forgejo's own API. Removing them changes behaviour only for malformed Forgejo responses (a zero value instead of ErrInvalidResponse). Keep the size limit, which guards against oversized responses. |
| soda-tailnet binary exists only to print guidance text from one host call | `cmd/soda-tailnet/*.go; lib/soda-release-image/src/prepare.rs:143` | no | 88 lines of Go that read one tailnet status call (2s timeout) and print a status and guidance message, linked into the image at prepare.rs:143. Small. Folding it into a script changes packaging and the command name, so it is low value. |

### Test harnesses and source guards

| Item | Where | bp | Note |
|---|---|---|---|
| scripts/build-soda-extension.test.ts is not run by any package script or CI job | `scripts/build-soda-extension.test.ts:1-365; tools/unitban/allow-ts.txt:2` | yes | No package script, check-*.sh or workflow globs scripts/*.test.ts (test:* only cover tests/frontend and tests/forgejo); the only reference is allow-ts.txt:2. Deleting it and that allowlist line changes nothing that runs. |
| allow-ts.txt comment is attached to the wrong entry | `tools/unitban/allow-ts.txt:1-2` | yes | Line 1 says 'e2e probe executed as a subprocess by component-browser-lifecycle.test.ts' but line 2 is scripts/build-soda-extension.test.ts. The comment does nothing at runtime. |
| Build-tagged branding test that never runs | `scripts/render_forgejo_native_test.go:1-18` | yes | The file is behind //go:build branding, no script or workflow passes -tags branding, and it only runs render-forgejo-branding.sh --check through require. testify stays in go.mod because 7 other files use it. |
| test:lit script duplicates test:frontend with a hand-maintained file list | `package.json:26` | yes | Every listed file already falls under the test:frontend:prepared or test:forgejo:prepared globs, and nothing in tests/frontend reads SODA_LIT_BROWSER. Removing it means also deleting the test:lit checks in tests/build/terminal_assets_test.go:48-51 and the doc mentions in typescript.md:53 and lit.md:28. |
| Founder-demo check split into modules to fit an 8 KiB per-file context limit | `tests/fixtures/founder-demo/README.md:31-37; northstar-check-*.mjs` | no | README:31-37 says the split exists for the native context reader's 8 KiB per-file limit. Merging the files would break that limit, so this can wait; it is not something to remove now. |
| check-no-unit-tests.sh stale-allowlist warnings will vanish once the allowlist is replaced by a directory rule | `scripts/check-no-unit-tests.sh:63-67` | no | Lines 63-67 only print stderr warnings for stale entries in allow-go.txt (264 entries) and allow-ts.txt (71). It can only go away together with a directory-rule rewrite, which is out of scope here; on its own it is tiny and harmless. |
| check-sql-locality.sh carries per-test-file exemptions for budget clock forging | `scripts/check-sql-locality.sh:20-40` | no | The exemptions exist (about 11 dispatch/traversal/review/budget test files), but they cover more than clock forging (traversal, carried history). Each one is still needed while its test file exists, so they can only go when those tests are removed. |
| Frontend fixture fake servers duplicate server logic | `tests/frontend/fixtures/intent-model.ts:1-319; workspace-model.ts:1-495; application-fixture.ts:1-316` | no | tests/frontend/fixtures totals 3130 lines (the three named files are 1130 of that). It is real upkeep, but it shrinks only as the surfaces it fakes shrink, so there is nothing to remove on its own now. |
| muse_exec_test is opt-in live CLI coverage that the default build suite skips | `tests/build/muse_exec_test.go:1-141` | yes | It calls t.Skip unless SODA_TEST_MUSE_CLI is set, and nothing in the repo sets that variable, so deleting it changes no default or CI run. |
| no-unit-tests.md is a 754-line copy-to-other-projects guide | `docs/development/no-unit-tests.md:1-754` | yes | The file is 754 lines. Docs only, so cutting it has no runtime effect, but the gate message points to why-no-unit-tests.md, and links to it would need checking. TST-1 is an audit ID, not anything in the repo. |

### Application runtime: build, home, snapshots, backups, restore, factory delivery

| Item | Where | bp | Note |
|---|---|---|---|
| LaunchApplicationRecord and OperatorApplication run the identical query | `internal/store/applications.go:202-216` | yes | applications.go:214 and :222 run the same SELECT string through scanApplication. Only the comments differ, so either can call the other or the two can merge. |
| Applications/OperatorApplications re-implement scanApplication inline | `internal/store/applications.go:250-281` | yes | applicationInventory's loop body (lines 271-285) copies scanApplication (lines 76-91). Change scanApplication to take interface{Scan(...any) error}, as scanApplicationBuild already does, and reuse it. |
| SetupRequest.Validate has three identical case branches | `internal/application/setup.go:85-96` | yes | The begin, inspect and abort cases in setup.go:76-87 are byte-identical. They can become one case "begin","inspect","abort". |
| SODA_APPLICATIONS_ROOT env exists only for a smoke test | `lib/host/src/application/home.rs:96-101` | yes | The only setter is lib/host/tests/gmux_smoke.rs:4571. Production always uses /home/soda-applications. Removing the variable changes nothing in production, but that smoke test's root redirection would have to go or be reworked. |
| Second entry to snapshot.rs via the standalone CLI | `lib/host/src/bin/soda-canlang-storage.rs:1-35` | no | Partly true. Only backup() (the None arms of backup_with_identity) is CLI-only. restore() is shared and is also called from restoration.rs:285/288. The CLI is a documented operator command (docs/public/50-Operate/50-backups-and-restoration.md) shipped by build_compile.rs, so removing it removes a visible feature. |
| Rust caches the BuildArtifact in private state as well as the Go application_builds row | `lib/host/src/factory/application.rs:61-81` | no | Go returns an existing application_builds row before calling the host (application_builds.go:92-98). The Rust cache is only reached when Rust succeeded but SaveApplicationBuild failed or crashed. Removing it means that retry rebuilds instead of re-checking. The result is nearly the same, but the retry path is not identical. |
| applicationView calls VerifiedApplicationDelivery twice per app | `internal/web/api/applications.go:83,159` | yes | It is called once directly (applications.go:83) and once in applicationEnvironment (:159). launchableApplications does the same (:140 env plus :143). Each call is its own FOR SHARE transaction (application_factory_delivery.go:110). One call per app is enough; the only difference is that the two separate reads could be slightly inconsistent today. |
| CanEmbed/applicationEmbeddable still computed on every view | `internal/web/api/applications.go:81,140,172-186` | yes | No frontend code reads can_embed beyond validating responses in sodaspaces-application-response.ts. That file requires can_embed to be a boolean on launch views (:309), so the parser must change along with the Go removal, and the matching frontend test fixtures go too. The publicsuffix import may then be unused. |
| Ingress install publishes a 4-line Caddy file via nonce, fsync, hard link and readback | `lib/host/src/application/ingress.rs:67-118` | yes | install_at stages a random-named pending file, fsyncs it, hard-links it to the final name and reads it back (ingress.rs:67-106), all inside a root-owned anchor dir. A plain create_new write followed by current_at gives the same observable result in local use. |
| extensionApplicationRoute admits snapshots/restores but the repository route omits application-builds | `internal/web/api/application_routes.go:5-22, internal/web/api/extension.go:259-267` | no | A 3-part /repositories/{id}/application-builds goes to extensionRepositoryProfileRoute, which admits only profiles/tailnet-options/factory, so it is rejected. The 4- and 5-part build/setup paths are not admitted either. The frontend calls these paths (sodaspaces-application-delivery.ts:191). Fixing this changes behaviour on purpose (it is the WEB-1 bug). |
| Snapshot inventory 64-cap plus 'complete' flag in Rust, Go and the frontend | `lib/host/src/application/backups.rs:256-263, internal/host/application_snapshots.go:17, frontend/spaces/sodaspaces-applications.ts:382` | yes | The 64 cap is in backups.rs:257-258 (truncate plus complete flag), application_snapshots.go:17 (rejects more than 64) and the frontend's 'newest 64' notice. Behaviour only differs past 64 backups, which nobody reaches. Go's 128KB response limit stays as a bound. |
| run_replace refuses 'migration is not qualified' | `lib/host/src/application/runtime.rs:345-356` | yes | runtime.rs:356 returns 'application migration is not qualified; replacement refused' when the storage label is missing. This is a wording-only fix: the gate stays and only the error text changes, so check whether any test or UI matches that string. |

### Test Pilot / app verification (Go factory+control+store+MCP, Rust host, in-project helper)

| Item | Where | bp | Note |
|---|---|---|---|
| Two doc comments stacked on DeliveryRepairReport and an AppVerificationReport doc comment misplaced above validateVerificationEvidence | `internal/factory/verification.go:236-241,354-365` | yes | verification.go:236-241 holds an AppVerificationReport comment sitting on validateVerificationEvidence. 354-365 has two stacked DeliveryRepairReport comments that point to R37/R38/R40 'consumers'. These are comments only, so deleting or merging them changes nothing. |
| Pass-through wrappers around the prerequisites join | `internal/store/factory_app_verification.go:80-114,186-197` | yes | applicationVerificationInputsTx and deliveredIntentTasksTx just return tasks[0].IssueIndex. They have 5 callers in total: delivery_admission x2, context_current, delivery_retry, and ApplicationVerificationInputs, which lifecycle_retry calls. Each caller could call the prerequisites/evidence function directly and index tasks[0]. The saving is small. |
| ApprovedProjectCheckCatalog reads requirement and approval heads twice around the host read | `internal/factory/control/verification_catalog.go:36-75` | yes | verification_catalog.go calls checkHeads() before InspectPreparation/ReadPreparationContext and again after them. The second call only catches an approval that changes during the host read. Outside that race the output is the same, so the second call can be removed. |
| verification_scope decodes its one-field cursor with DisallowUnknownFields after the JSON schema already forbids extra properties | `internal/web/api/factory_mcp_verification_journey.go:46-60` | yes | verificationJourneyPayloadSchema (line 23) declares {after:integer} with AdditionalProperties Not{}, and the handler decodes again with DisallowUnknownFields. Behaviour stays the same as long as the MCP input schema is enforced before the handler, which is the case for the payload schemas wired through factory_mcp_context. Keep the page.After<0 check. |
| PendingAppVerifications rescans stale verified reports forever | `internal/store/factory_app_verification.go:51-78; internal/factory/control/app_verification.go:152-175` | no | The query only excludes reports that have an F10 feed comment. driveAppVerifications pages with a cursor that wraps to 0, and RecordAppVerificationWait adds no terminal marker, so a report that can never be adopted is re-read on every pass. This is a real churn issue, but fixing it means adding a terminal state, which changes behaviour. Fold it into VRF-5. |

### Retry kinds across store, control and UI

| Item | Where | bp | Note |
|---|---|---|---|
| Two identical one-field structs with misplaced doc comments | `internal/store/factory_explicit_retry.go:39-45,182-187` | yes | factory_explicit_retry.go:39-45 and 182-187: admittedRetryPayload and admittedRetryCommand each hold only priorOutcome, and the RecordExplicitRetryCommand/RecordExplicitRetryPacket doc comments sit on them instead of the functions. They can become one struct or a plain string return, with the comments moved. |
| Near-duplicate publication/merge loaders with an artificial row limit refusal | `internal/store/factory_explicit_retry.go:459-513` | yes | loadRetryPublicationsTx and loadRetryMergesTx are line-for-line copies apart from table and type. Merging them is behaviour-preserving. Dropping the >1024-row refusal changes behaviour only past 1024 rows per issue, which never happens in practice. Keep FOR UPDATE because it serializes the work with Retry admission. |
| Single-use querier interface | `internal/store/factory_explicit_retry.go:128-130` | yes | explicitRetryQuerier only lets the code accept either *sql.DB or *sql.Tx. It is used by pendingExplicitRetry and also pendingImplementationRetry (factory_implementation_retry.go:202), so it has two users, not one. The only non-tx caller is PendingExplicitRetry via s.db. Minor. |
| Status GET takes row locks | `internal/store/factory_delivery_retry.go:104-128` | yes | The GET handler (factory_views.go:210) calls DeliveryRetryAvailable, which reuses the action's deliveryRetryPriorTx with FOR UPDATE on assignments, intent heads and the latest verify assignment, then rolls back. A lock-free read would give the same result, with less contention with dispatch writes. |
| Fixed reason strings stored and length-validated | `internal/factory/control/lifecycle_retry.go:47-53; internal/factory/lifecycle.go:125-133` | no | Reason is always one of two fixed strings but is still validated to 1-256 chars. Dropping that check is behaviour-preserving. Dropping the field is not: RetryDecision is returned as the JSON body of the Retry API (factory_lifecycle.go:187-191) and is stored in the command outcome. |
| Host-not-found detected by error substring | `internal/factory/control/dispatch_recovery.go:164-172` | yes | The comment says control cannot import the host package, but control already imports internal/host (verification_execution.go:12) and uses errors.Is(err, host.ErrRunNotFound) at line 91. The host client returns that sentinel directly, so errors.Is can replace isHostNotFound and its pinning test can be deleted. |
| Pass-through wrapper | `internal/factory/control/dispatch_preparation.go:58-60` | yes | prepareRetryCoder only calls prepareCoder(ctx, plan, command, nil). It is used as a function value for DispatchDeps.PrepareRetry (dispatch.go:431 and two test fixtures), so it adapts the signature. An inline closure would do the same job. Trivial. |
| Delivery retry has its own verification-time floor | `internal/store/factory_delivery_retry.go:154-160` | no | deliveryRetryPriorTx refuses when remainingAppVerificationSecondsTx < 60. This is an extra hard-coded threshold on top of the normal admission checks. Removing it would accept retries with 1-59 seconds of allowance left, so it is a behaviour change, though probably a harmless one. |

### Project preparation pipeline below the API (standing and candidate preparations, decisions, hold, host and root helper)

| Item | Where | bp | Note |
|---|---|---|---|
| Production code switches on a test-only environment variable | `cmd/soda-project-terminal/src/factory_roles/mod.rs:6-11,56-124` | yes | Ctx::from_env/priv_uid/priv_gid/git() in factory_roles/mod.rs take a test branch on SODA_FACTORY_DIR/SODA_FACTORY_GIT. The only consumers are tests: cmd/soda-project-terminal/tests/factory_roles_oracle.rs and tests/build/project_factory_roles_fixture_test.go. These are integration/fixture tests, not unit tests, so they would have to be deleted along with the seam. Production with the variables unset is unchanged. |
| PendingStandingPreparations fair-cursor paging | `internal/store/preparation.go:383-410` | no | The only caller is observeStandingPreparations (preparation_start.go:144). It uses a limit of 4 and an id cursor that rotates across Projects. A plain query for all non-terminal rows is simpler, but each tick would then inspect every row instead of at most 4 (still capped by readinessBudget), so per-tick RPC count changes. The end state is the same. |
| Client-chosen preparation and decision IDs | `frontend/spaces/sodaspaces-factory-settings.ts:780-817` | no | pendingPreparationIds/factoryPreparationId (sodaspaces-factory-settings.ts:257) mint coder/reviewer IDs in the browser. prepareIdentities requires the replace_terminal_preparations confirmation. Moving ID generation to the server changes the API contract (coder_id/reviewer_id) and removes the confirmation step. The simplification is valid but not identical. |
| Prepare path calls ensure on every request | `lib/host/src/prepare/mod.rs:229` | no | prepare_inner (lib/host/src/prepare/mod.rs:227) runs the factory_helper ensure op before every prepare. Running it only at enrollment loses the per-prepare self-repair of role accounts and changes behaviour when accounts drift. It does save a root exec per prepare. |
| Hard-coded 30s/25s/5s nested deadlines in StartPreparation | `internal/factory/control/preparation_start.go:43-90` | no | preparation_start.go has a 30s bounded context, a 5s inspect, a deadline-5s host.Prepare call, and a 5s recovery inspect under context.WithoutCancel. These could collapse into fewer deadlines. However, the recovery path deliberately outlives a cancelled request, so a simple flattening changes timeout and cancel semantics. |
| retryPreparationInvalidated lists eight unrelated errors | `internal/factory/control/dispatch_preparation.go:48-53` | no | The 8-sentinel errors.Is predicate is real (dispatch_preparation.go:48-53). It is a single small predicate, and removing it changes when a retry preparation is retired. It only goes away as part of the larger PRP-2 redesign, not as a standalone cut. |
| Two ways to approve | `internal/web/api/preparation_decisions.go:100-112; internal/web/api/preparation_source.go:170-198` | no | approve-initial (apiPreparationInitialApproval, preparation_source.go:170-198) checks the free-text evidence, then digests it into EffectsDigest/ReadinessDigest and calls apiPreparationActionApprove. The frontend keeps submitInitialApproval and submitApproval (ts:620-622). Merging them means changing the request contract and one form, so behaviour is not identical, though the merge is a reasonable simplification. |

### Run end: task events, checkpoints, results and settlement (RUN)

| Item | Where | bp | Note |
|---|---|---|---|
| Optional host interfaces with a single implementation | `internal/factory/control/task_checkpoint.go:17-24, internal/factory/control/task_progress.go:12-14` | yes | The only production Host is the *host.Client that web/server.go passes to NewCoordinator, and it implements all four methods (host/factory_checkpoint.go, factory_client.go). Only the test double progressTaskJournalHost also implements them. Moving the methods into HostFactory changes nothing in production, but every test HostFactory stub (stubHost, standingHost and others) would then need the methods or have to be deleted. |
| Store interfaces that exist only for (now banned) stubs | `internal/factory/control/mcp_settle.go:49-89, internal/factory/control/dispatch_result.go:116-121` | yes | MCPTerminalSettleStore, mcpDriveStore and repairChainStore are each implemented only by *store.Store, which is the type every production call site passes (settle.go:298, mcp_settle.go:398, dispatch_result.go). Tests are the only other users: finishRacingSettleStore in mcp_settle_pg_test.go and the stub stores in dispatch_restore_helpers_test and library_maintenance_test. Those would have to go along with the interfaces. |
| Unreachable inner default in mapMCPTerminalFinish | `internal/factory/control/mcp_settle.go:254-256` | yes | The outer case admits only MCPToolFinishIncomplete and MCPToolFinishImplement, and the inner switch handles both explicitly, so `default: settleCodeFinish` is dead. |
| Test-only store wrappers | `internal/store/factory_planning_revisions.go:478-485, internal/store/factory_planning_revisions.go:626-638` | yes | The singular SavePlanTaskReceipt (line 483) and the PlanTaskReceipt getter (line 643, not 626-638 as cited) are called only from _test.go files. Production uses SavePlanTaskReceipts and PlanTaskReceipts. |
| PlanTaskReceipt.InputDigest is never written | `internal/store/factory_planning_revisions.go:112, internal/store/factory_planning_revisions.go:183` | yes | No writer sets it: the seed at task_execute.go:240 and the copies at :436/:460 never touch it. It is only validated, compared at task_execute.go:355, and copied out as an always-empty omitempty field into CurrentTaskReceipt (context_current.go:310) and completion material (completion_material.go:977). Dropping it and those projections changes no output. |
| Usage.Validate re-derives minutes that the caller just computed | `internal/factory/assignment_resources.go:76-86, internal/factory/control/dispatch_result.go:175-184` | yes | Both producers compute minutes with the same rounding formula: recordConfirmedUsage in dispatch_result.go:176 and the accounting path at store/factory_mcp_accounting.go:97. Validate (called from RecordRunUsage and from the accounting path) recomputes it and can never disagree, so the check is redundant. |
| synthesizePhased has a legacy fallback | `internal/factory/control/dispatch_result.go:499-502` | no | The fallback exists, but it is not provably dead. ValidatePhaseBinding can fail on a role/phase mismatch or on an empty or oversized summary (PhaseResult.Validate), and then the result has no PhaseResult instead of an invalid one. Removing it changes behaviour in those edge cases unless they are ruled out first. |
| Legacy review-json and typed-review stdout prompt branches look unreachable | `internal/factory/dispatch_prompt.go:405-412, internal/factory/dispatch_prompt_stage.go:143-155` | no | Only half right. writePromptTypedReviewContract is dead: the only place that sets TypedReview/TypedClear is publication_children.go:1112, where a reviewer always gets phase review, which takes the MCP branch. Removing it preserves behaviour. The legacy review-json block, however, is reachable: verify and maintain_library reviewer prompts fall through to it, and no parser accepts review-json. That is a live stale instruction, and removing or fixing it changes the prompt text. Hand to the prompt finder. |
| Compatibility shim for rows admitted before harness proof | `internal/factory/assignment.go:585-591` | yes | Every production Assignment writer sets AdoptedHarness: dispatch_launch.go:56/223/406, dispatch_timeonly.go:410, publication_children.go:999. The empty-pin legacy allowance is repeated in assignment.go:585, run.go:123 (AdoptedToolIdentity) and factory_retry_packet.go:67/89. Making it required is safe with zero users. Test fixtures that leave it empty would break. |
| Prompt SHA re-hashed on every Assignment.Validate | `internal/factory/assignment.go:604-606` | yes | checkAssignmentExecution hashes the full prompt on every Validate. Valid rows always pass, so dropping the check is observably identical. The cost is about 1 ms of sha256 per call, so this is cosmetic rather than a real burden. |
| The broker-close retry's ctx.Done break exits only the select | `internal/factory/control/settle.go:255-266` | yes | The `break` in `case <-ctx.Done()` leaves only the select, so CloseExecution is retried with a cancelled ctx. It fails fast, closeErr stays non-nil, and the receipt ends up the same 'broker closure unconfirmed'. Simplifying the loop or returning on cancel gives the same result. |
| Another budget in the run-end loop | `internal/factory/control/progress.go:126-170` | no | progress.go:126-170 calls budget.reserveNativeRPC(1\|5) around inspect, import, recoverFinalCheckpoint, driveRunCheckpoints and settleRun. This is the shared readiness budget that the completion and dispatch paths also use. Removing it changes how much work one pass does. Hand to the budgets finder. |

### In-project helpers (soda-project-terminal) and the host Muse launch runtime

| Item | Where | bp | Note |
|---|---|---|---|
| internal/identity/launch.go has no Go references | `internal/identity/launch.go:1-38` | yes | rg over *.go finds no use of MuseLaunchSocket/NestedRegistration/LaunchRequest/LaunchControl/LaunchExit outside the file. The live protocol is lib/host/src/muse/wire.rs, so the file can be deleted. |
| soda-muse validates the launch request a second time on the client side | `cmd/soda-muse/src/shell.rs:62-110` | yes | validate_shell (shell.rs:63) is called once at shell.rs:53. It repeats LaunchRequest::validate (wire.rs:374) check for check, and the host runs that at request.rs:74 with the same 'identity authority denied' result. The only change if removed: an invalid request is refused by the host instead of before connecting. |
| soda-muse reproduces Go error and path text | `cmd/soda-muse/src/paths.rs:1-75` | no | Partly true. The go_strerror lowercasing and go_quote_rune (execution.rs:113) only copy Go's error text. go_base (execution ID from the root, execution.rs:108) and go_join (config_home, shell.rs:32) do real path work as thin std wrappers and must stay, though they can be renamed. Simplifying changes the wording of error messages. |
| Three copies of the stdout/stderr + EPIPE write helpers | `cmd/soda-project-terminal/src/main.rs:86-96, broker.rs:101-135, keys.rs:185-214` | yes | main.rs:86, broker.rs:117-135 and keys.rs:200-214 hold near-identical write_stdout_all/EPIPE/write_stderr_all code. All are modules of the same crate (main.rs declares mod broker/keys), so one shared helper works; only the error enum differs (BrokerFail/KeyFail). |
| Two Unicode Cc/Cf tables of different versions for terminal names | `cmd/soda-project-terminal/src/proto.rs:107-139; lib/host/src/terminal/text.rs:37-47` | yes | Duplication is real: proto.rs:108 is_cc_or_cf vs text.rs GO_CC/GO_CF, plus more copies in lib/host/src/domain/mod.rs and cmd/soda-project-maintain/src/config_wire.rs. But the 'different versions, can disagree' claim is wrong: the ranges are the same today. Also, term_status.rs:201 checks a name read from a guest disk record, not one the host just validated. |
| Terminal frames validated and re-encoded on both sides | `lib/host/src/terminal/native.rs:189-200, 334; cmd/soda-project-terminal/src/proto.rs:94-106` | yes | The host checks input_valid before sending (native.rs:334). pty_relay.rs:52 then decodes with proto::decode_frame, which repeats the strict-key, base64 1..16384 and dimension bounds. The host is the only producer, so the helper can do a plain decode. The helper still needs a JSON parse. |
| project-terminal hash checked three ways | `cmd/soda-project-terminal/src/term_create.rs:57-76; lib/host/src/terminal/support.rs:66-157; lib/host/src/terminal/identity.rs:180` | no | identity.rs:180 sends agent_program_hash, and term_create.rs verify_program re-hashes /usr/libexec/soda/project-terminal against source_hash on every create. The support probe (support.rs) hashes the guest copy separately. Removing the per-create check drops the 'terminal support version differs' refusal when the probe has not run or the copy is stale, so behaviour changes. |
| Host and identity broker use different provider stores | `system/host/services/soda-identity.service:12; lib/provider-tools/src/config.rs:13-23` | no | soda-identity.service sets SODA_PROVIDERS_ROOT=/home/soda-identity/providers, and soda-identity reads it through config::store_root (resolution.rs:87). soda-host runs as root on the default /var/lib/soda/providers. Merging them needs ownership and permission changes for the soda-identity user, so it is not a no-op. |
| Duplicated FACTORY_HARNESS_CODEX constant | `lib/host/src/factory/run.rs:14; lib/host/src/terminal/factory/run.rs:12` | yes | Defined in both lib/host/src/factory/run.rs:14 and lib/host/src/terminal/factory/run.rs:12, and both are used (bound.rs uses crate::factory, mcp_relay.rs uses super::run). Re-export one. FACTORY_HARNESS_MUSE and the family check look duplicated in the same way. |
| broker.rs header documents dead Python deltas | `cmd/soda-project-terminal/src/broker.rs:11-28; keys.rs:10-19` | yes | broker.rs:11-28 and keys.rs:10-19 are long notes on differences from the old .py versions, which no longer exist. Deleting the comments changes nothing. Removing the actor_id int/float/bool/string coercions they describe would be a separate code change, with no effect on output only if the host always sends ints. |

### Worker context assembly, project library and research

| Item | Where | bp | Note |
|---|---|---|---|
| Compatibility ALTER TABLE for pre-digest artifact tables | `internal/store/factory_artifacts.go:30-34` | yes | The CREATE TABLE at factory_artifacts.go:28 already has `digest TEXT` with the same CHECK, so the ALTER ADD COLUMN IF NOT EXISTS at :34 does nothing on any table that CREATE makes. It can be deleted. |
| Unwired CurrentContext fallback only reachable in tests | `internal/web/api/factory_mcp_context.go:19-27` | yes | The only production deps are built at factory_mcp.go:1129 with Submissions: s.Store, and *Store implements FactoryCurrentContext (store/factory_context_current.go:18). The bound handler copies those deps, so the !ok branch at factory_mcp_context.go:20-27 can only be reached from test sinks. |
| Duplicate pagination loops in library search vs inventory | `internal/store/factory_library.go:46-81,87-130` | yes | factory_library.go:57-80 and :106-127 are the same 50-item / MCPMaxPageBytes cursor loop over factory_library_items, differing only in the struct they fill. One shared helper would behave identically. |
| Run.ObservedState tri-state with 'not_wired' default | `internal/store/factory_context_current.go:74; internal/web/api/factory_mcp_context.go:72-78` | no | Worse than reported: deps.Observed is never set in production (factory_mcp.go:1128-1140), so the model always sees observed_state="not_wired" and the unavailable/confirmed branch (factory_mcp_context.go:72-78) never runs. The whole Observed dep is dead in production. Nothing in code reads the field, but dropping it removes a key from the JSON the model sees (docs/development/factory-workflow-contracts.md:1557 also mentions it). |
| Role.Tools/ResultTool/ResultSchema repeat MCP tools/list | `internal/web/api/factory_mcp_context.go:36-60; internal/factory/context_current.go:72-74` | no | ResultSchema comes from the same rolePayloadSchema/libraryPayloadSchema that tools/list uses for input schemas (factory_mcp.go:137-140), and Tools is MCPDiscovery again, so this is a second listing. Removing it changes what the model sees in factory.context and how it pages; context_bundle_state.go reads only Role.Submissions. |
| 8192-byte inline cap silently empties current.role | `internal/web/api/factory_mcp_context.go:80-85` | no | At factory_mcp_context.go:83-85, when the encoded context is over 8192 bytes, current.Role is replaced by {Kind, RecordSource:"required_current"}. No MissingReason is added and Availability is unchanged, so the model gets no explanation. Adding a reason or removing the cap changes model-visible output. |

### Human terminal stack end to end (browser, Go API, host, in-project helper)

| Item | Where | bp | Note |
|---|---|---|---|
| Duplicate dimension validator in the same file | `internal/web/api/terminal_registry.go:50-52,157-159` | yes | validHandshakeDimensions (terminal_registry.go:157) has the same 2..500/2..300 bounds as validTerminalGeometry (:50), and host.terminalDimensions (host/terminal.go:72) is a third copy. Export one host helper and call it from api. |
| Two 32-hex terminal id regexes in Go | `internal/web/api/terminal_registry.go:24; internal/host/terminal.go:243` | yes | api.TerminalID (terminal_registry.go:24) and host.terminalID (host/terminal.go:243) are the same ^[0-9a-f]{32}$. Export host's and drop api's. |
| Unused TerminalPeer fields | `internal/web/api/api.go:51-54` | yes | Token is never set or read anywhere. ContextID is set in extension_terminal.go:61 and extension_terminal_stream.go:70, but peer readers (lifecycle.go:82, terminal_registry.go:216, peerIDInUse) use only Project, ID and Cancel. |
| Every End makes a broker lookup podman exec, even for plain shells | `lib/host/src/terminal/identity.rs:257-293; lib/host/src/dbackend/websocket.rs:106-119` | no | managed_end always runs an identity 'lookup' call into the container for action=end and returns early when the lease is empty. Skipping it means moving lease detection into the guest end path. Outcomes could match, but the reconcile path changes, so this is not a drop-in removal. |
| Host terminalTargetUnchanged re-checks the reply of an op it just issued | `internal/host/terminal_client.go:69-74,95-97` | yes | This is a pure defensive check against the in-project helper returning a different id. reservedTerminal (create path, extension_terminal.go:126) and the stream's create check (len==1 && Ready) check the reply again. It can be removed while the guest behaves correctly. |
| Terminal support messages duplicated in Go and TS | `internal/web/api/terminal_support.go:14-18; frontend/spaces/sodaspaces-terminal-error.ts:3-8` | no | The same three English strings appear in terminal_support.go and sodaspaces-terminal-error.ts, and the frontend maps by code. The Go JSON 'message' field would still need some value, so the API response body changes slightly. The UI would look the same. |
| Output bytes base64-in-JSON across four hops | `cmd/soda-project-terminal/src/pty.rs:66-78; frontend/spaces/sodaspaces-terminal-attachment.ts:253-270` | no | output_line base64-encodes into JSON, and the frontend atob-decodes with a 4096-byte per-frame cap and a 262144-byte queue. Moving to binary frames is a wire-protocol rewrite across helper, host, Go and TS, and the caps change. It is a real simplification but not a quick drop-in. |
| Host hand-rolled serde visitors for podman inspect and broker binding | `lib/host/src/terminal/inspect.rs:20-170; lib/host/src/terminal/binding.rs:23-200` | yes | Hand-written Visitors (inspect.rs Mappings/TerminalInspection, binding.rs BindingWire) do case-insensitive key matching and turn null into an empty value. derive(Deserialize) with rename plus Option/default covers every real input, because podman and our own writers emit fixed key casing. Only odd-cased keys that are never produced would behave differently. |

### Project environment runtime on the host (create/join/start, executor, domain, ssh, accounts)

| Item | Where | bp | Note |
|---|---|---|---|
| Takeover is still reachable | `internal/project/takeover.go; internal/factory/control/lifecycle_takeover.go:24; frontend/spaces/sodaspaces-factory-work.ts:1377` | no | This is information, not a removal. TakeoverRun (lifecycle_takeover.go) is reached from the Factory work UI's 'takeover' intervention (sodaspaces-factory-work.ts:1377 posts to factory/runs/{id}/actions), and sodaspaces-workspace-setup.ts:321 also lists it. Removing takeover would remove a live feature. Only the workspace-setup wiring goes away with the no-Workspace removal. |
| wait_project_ready checks the deadline twice per loop | `lib/host/src/project/create.rs:131-156` | yes | create.rs:149-155 checks the deadline before and after the 1s sleep. Drop the check after the sleep: Native::run refuses to start a process once the deadline has passed (executor.rs:592), so the next pass returns the same 'deadline exceeded' with no podman launched. |
| impl Executor for &E is 50 lines of forwarding | `lib/host/src/project/executor.rs:87-136` | yes | executor.rs:87-136 forwards all five trait methods through (*self). It shrinks to nothing or almost nothing once the trait has fewer methods. Callers stay the same. |
| valid_key_revision duplicates domain::is_hex_lower | `lib/host/src/account/mod.rs:22-27` | yes | account/mod.rs:23-27 repeats the same byte predicate as domain::is_hex_lower (domain/mod.rs:54, pub(crate)). With len==64 the !is_empty check is already implied, so `s.len()==64 && domain::is_hex_lower(s)` is identical. preparation/mod.rs:64 has a third copy of is_hex_lower. |
| Codex harness family is still admitted | `internal/project/factory.go:17-22,133-135; internal/web/api/factory_mcp_p6.go:28` | no | FactoryHarnessCodex/FactoryScopeCodex (factory.go:19-22), ValidHarnessFamily (133-135) and p6AdmittedHarness (factory_mcp_p6.go:28-29) still accept 'codex'. Removing them would make codex requests fail, which is an observable change, but the owner runs Muse only. The rest of the harness subsystem (tcodex etc.) is outside this scope. |
| #[allow(unused_imports)] re-exports exist only for a test's private #[path] copy | `lib/host/src/project/mod.rs:42-55` | yes | The comments are stale. muse_serve_oracle.rs no longer contains any #[path] copy: it already uses the public API (`use soda_host::{..., project, ...}`, `use project::Executor`). The two #[allow(unused_imports)] and their comments (project/mod.rs:42-55) can simply be deleted, so the test needs no change. Re-exports that turn out unused could then be dropped. Only compiler warnings change. tcodex.rs:31 has the same stale comment. |
| ssh and base64 helpers are used as a general base64 utility across lib/host | `lib/host/src/ssh/mod.rs:5; lib/host/src/factory/artifacts.rs:63; lib/host/src/prepare/state.rs:280; lib/host/src/iclient.rs:484` | yes | b64_encode and b64_encode_raw are one-line wrappers over base64 STANDARD/STANDARD_NO_PAD and are called from about 10 places outside ssh (artifacts, prepare/state, preparation/state, iclient, terminal/lease), so calling the crate directly gives identical output. b64_decode and b64_decode_go use a custom Go-compatible engine (lenient trailing bits, CorruptInputError offsets). They are not plain base64-crate calls, so they cannot be swapped for identical behaviour, only moved out of ssh. |
| The key-type cross-check after re-serialisation is redundant | `lib/host/src/ssh/mod.rs:133-145,183-186` | no | ssh/mod.rs:128-145 parses certificates, checks the algorithm name and requires re-serialised bytes to equal the input. check_key_data (68-81) runs DSA p/q/g/y parameter checks. This is excess policy, but rejecting certificates and DSA outright changes behaviour, because valid certs and DSA keys are accepted today. It simplifies a lot. |

### Release build primitives, asset fetch/render tools, small install binaries and system units

| Item | Where | bp | Note |
|---|---|---|---|
| Units written with /usr/local paths and then rewritten at build | `system/host/services/soda-host.service, soda-identity.service, soda-tailnet*.service, soda-project-release-reconcile.service; lib/soda-release-image/src/prepare.rs:98-100` | yes | prepare.rs:98-100 rewrites /usr/local/libexec/soda/ to /usr/libexec/soda/ for every rootfs file copied with vendor=true (prepare.rs:160). The rewrite hits more than the units: it also covers soda-project@.service, soda-console.service and config/console-welcome.sh. Removing it means fixing all those sources plus scripts/fixtures/portcontracts/systemd_wiring.json, which a Go test checks against the unit text. Do not touch the unrelated /usr/local/libexec/soda/muse paths inside the project image. |
| Negative assertions for retired features in the host-image check | `lib/soda-release-image/src/host.rs:64` | yes | host.rs:64 has 8 `test ! -e` lines plus `! rpm -q tailscale`, all guarding against retired layouts. Dropping them changes nothing for a correct image. Keep `test ! -e /usr/local/libexec/soda/soda-host` only while item 1's rewrite exists. |
| Pass-through validators in files.rs and coreos.rs | `lib/soda-release-build/src/files.rs:101-110; lib/soda-release-build/src/coreos.rs:32-35` | yes | files.rs:102-109 and coreos.rs:33-35 are one-line forwards to soda_build_tools::reader. They are imported in about 8 files (production_inputs, production_images, coreos_registry, coreos_stream, http, fountain_inputs, production). Those imports can point straight at reader. |
| Dead public functions in soda-release-build | `lib/soda-release-build/src/coreos_stream.rs:228-250; lib/soda-release-build/src/lib.rs:70-80` | yes | resolve_coreos_iso and resolve_coreos_qemu have no callers; only the *_with variants are used, from coreos_iso.rs:60 and coreos.rs:197. Error::with_exit_code, with_signal and cancelled have no callers anywhere. The exit_code, signal and cancelled fields and their getters are therefore always None/false, so the Error struct could shrink to just the message. |
| Dead backward-compatibility wrapper fetch_muse | `tools/release-assets/src/fetch/muse.rs:549-565` | yes | fetch_muse (muse.rs:552) is referenced only at its own definition. Its doc comment says 'Backwards-compatible default fetch'. |
| Importer TOCTOU inode recheck on an image-owned file | `cmd/soda-image-import/src/payload.rs:19-38` | yes | The dev/ino recheck is at payload.rs:19-38. The note is slightly off: load() also serves the Ensure invocation, where the payload path comes from an argument, not only /usr/share/soda/release.json. That caller is still root/system-run, so a plain open plus take(max+1) is enough. The only difference is one error message on a race that does not happen in practice. |
| Go-parity comments and constants throughout | `cmd/soda-image-import/src/main.rs:35; lib/soda-release-build/src/lib.rs:31,101-131; tools/release-assets/src/fetch/muse.rs:18-20` | yes | The Go framing appears at main.rs:35, lib.rs:31/101/108/116/129 and muse.rs:18-20. Rewording the comments is behaviour-neutral. The USER_AGENT value 'Go-http-client/1.1' is actually sent on the wire, and the comment says the endpoint has only been seen with it. Reword its comment but keep the value, otherwise behaviour changes. |
| Crate name and path mismatch | `lib/release-inputs/Cargo.toml:2` | yes | lib/release-inputs/Cargo.toml declares name soda-build-tools with the description 'Foundation crate ... (lane R)'. It is cosmetic only. Renaming touches every dependent Cargo.toml and import, so it is churn with low value. |
| Second install target (Docker) beside the CoreOS ISO | `system/docker/install-docker.sh (403 lines); system/docker/Dockerfile; lib/soda-release-build/src/production_docker.rs; lib/soda-release-deliver/src/docker/` | no | install-docker.sh is 403 lines. The docker packaging runs through production_docker.rs, soda-release-deliver/src/docker/, and the distribution choice appliance\|docker\|all in release_cli, release_start, pipeline and qualification/target. Removing it is an owner product decision and changes what ships. |
| Fixture-only QEMU provisioning path | `tools/release-assets/src/render/provisioning/ (664 lines); lib/soda-release-build/src/coreos.rs fetch_coreos; tools/test-vm` | no | render/provisioning is 664 lines, and its only invokers are its own tests and the Go test tests/build/native_support_test.go. Nothing in tools/test-vm or tools/acceptance runs soda-render-provisioning. tools/acceptance/src/coreos.rs has its own qcow2 fetch, so soda-artifacts fetch-coreos / fetch_coreos look unused by the VM flow too. It is not on the install path. Removing it drops CLI commands, so it is not behaviour-identical, but it is likely dead. |

