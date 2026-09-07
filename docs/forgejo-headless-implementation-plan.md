# Headless Forgejo implementation plan

**Status: H01 source audit recorded; H02 source preparation implemented at `c9a9be0`; native image/extensions remain unimplemented.**
The [single workflow register](forgejo-api-coverage.md) now records the v15.0.7
surface audit and targeted v16.0.3 comparison. U01's subsequent baseline/first-read/
native-auth/build-license source review is accepted. The audit alone did not
supply that review, and neither establishes native patch/build/deployment proof. Implements the proposed
[architecture revision](forgejo-architecture-revision-plan.md) through the existing
[U01–U20 owners](dashboard-implementation-plan.md). This is a detailed work package,
not another product roadmap or a reset of accepted U08 evidence. The leading
plan's [post-H01 execution order](dashboard-implementation-plan.md#6-milestone-map-and-execution-order)
and [action ownership](dashboard-implementation-plan.md#9-inventory-coverage-cross-reference)
now govern sequencing: early authentication/admin review, feature-owned gaps,
candidate verification before retained-installation cutover. U01 planning/contract
readiness and bounded U08 native x86_64 proof are accepted (2/20). The [source preparer](../appliance/forgejo/README.md) locks 16.0.3 only as a
development candidate, not an approved deployment baseline. The
[authentication design](forgejo-authentication-design.md) and
[first read contracts](forgejo-read-contracts.md) record reviewed implementation
dispositions, not existing endpoints.

**Goal:** make missing or changing upstream interfaces a supported engineering
path, not a recurring reason to expose Forgejo's frontend or abandon a feature.
“Future-proof” means early detection, stable bounded contracts and tested updates;
it cannot promise compatibility with arbitrary future Forgejo releases.

All Forgejo user workflows stay in Soda. Forgejo owns authentication, permissions,
Git, collaboration and administration. Preserve React, Go, existing suitable
adapters, Soda extension records and project integration. No scraping, embedded
HTML, copied provider roles, local repository clones/indexes, unrestricted proxy,
privileged sidecar or second password authority.

## 1. Concrete source and delivery arrangement

Tracked layout and remaining build work (the source preparer exists; most native paths remain proposed):

| Path | Responsibility / caller |
| --- | --- |
| `docs/forgejo-api-coverage.md` | The single action/workflow coverage register; expanded by owning U milestones, reconciled in U17 |
| `appliance/forgejo/README.md` | Implemented source-preparation procedure, development baseline review and remaining native build/license/maintenance work; extend with patch provenance and real upgrade/removal instructions |
| `appliance/forgejo/source.lock.json` | Implemented exact development commit/archive integrity and ordered patch digests (currently empty); required build-image identities still pending; sole authoritative new source lock |
| `tools/soda-forgejo-source/`, `internal/forgejobuild/` | Implemented core-owned Go source retrieval/verification/extraction/patch preparation and integrity receipt; called by a build-only command, not installed |
| `appliance/forgejo/patches/` | Reviewable upstream-compatible patches, including native tests and API specification changes; no copied full Forgejo tree |
| `appliance/forgejo/Containerfile` | Build/package the selected patched Forgejo, preserving required upstream entrypoint/runtime behavior and notices |
| `scripts/build-forgejo.sh` | Narrow Forgejo build entrypoint called by `scripts/build-native.sh`; verified source and fresh run-owned outputs only |
| `internal/forgejobuild/*_test.go` | Implemented focused source/patch/identity failures through the existing Go aggregate; extend with real native build-boundary checks rather than duplicate a Python source preparer |
| `internal/forgejo/` | Existing explicit stock clients plus feature-owned extension clients and compatibility checks |
| `internal/web/`, `dashboard/src/` | Existing Soda API and React features; no second adapter/router/frontend stack |
| `tests/installed/` | Core-owned real Forgejo/Soda workflow tests, including new interface and preservation proof |

Only the entries marked implemented and the existing Soda owners exist. Native
patches, Containerfile/build caller and extension/compatibility routes do not.
Full upstream checkout, generated patches/build products and test repositories belong under a
fresh ignored `.artifacts/` attempt directory. Preserve attribution and required
corresponding source; do not fabricate hashes or upstream contribution references.

Current `scripts/build-native.sh` pulls Forgejo from the service unit's image
reference alongside Caddy. Replace **only the Forgejo build input** when the new
candidate is approved; continue producing the existing `forgejo.iid` and
`images/forgejo.oci` outputs. Caddy is unchanged. Coordinate image classification,
identity/sealing, installer retagging and staging with their existing owners.
Do not add an independent build/stage implementation or a permanent stock/patched
runtime selector. No image/service reference changes are made by this plan.

## 2. Work packages, dependencies and completion

H numbers below identify tasks within existing U milestones, not new acceptance
milestones. Each implementation commit includes callers, focused tests and an
honest handoff; authoring a contract does not pass its workflow.

| Task | Existing owner | Depends on | Deliverable / exit |
| --- | --- | --- | --- |
| H01 — Action-level contract audit | U01/U17 with feature owners | Architecture review | Every required action classified; exact upstream evidence, authorization, existing or missing interface, implementation owner and test requirement recorded |
| H02 — Source/patch build spine | U01/U02/U03 shared build owners | U01 baseline/build/license/maintenance review | Verified native Forgejo build from pinned source/patches, compatible output/install identity and negative build tests |
| H03 — Bounded compatibility contract | U03/U04/U17 | H01, reviewed first API and pre-auth compatibility contracts | Soda distinguishes supported, incompatible and unavailable provider interfaces without HTML/privileged fallback |
| H04 — Blame and aggregate diff | U09 | Relevant H02/H03 contracts and early H05 feasibility review | First complete native read slice → explicit API → Soda adapter → React, with authorization/resource/native Git tests; not all U09 acceptance |
| H05 — Headless authentication/security | U04/U05/U16 | H01; design starts before broad expansion, alongside H02/H03 | Reviewed threat model and full native-backed login/challenge/security flow; account-security and admin design do not wait for all collaboration screens |
| H06 — Remaining coverage | U05/U06/U09–U16 features, with authentication/security in H05 | Relevant H02/H03/H05 and shared feature contracts, not whole-milestone completion | Remaining code, boards/reviews/settings/Actions/wiki/packages/account/admin actions completed by their owners, not postponed to U17 |
| H07 — Upgrade/rebase verification | U01/U02/U03/U17/U20 | H02–H04 first slice; expanded with H05/H06 | Candidate update procedure demonstrated against native contracts and approved populated fixtures |
| H08 — Installed integration and UI exposure closure | U17/U18/U20 and feature owners | Required H04–H07 work | Matching candidate, preserved state, complete Soda-only browser journeys and deliberate API/Git/package/Cockpit separation |

**Order from the audited tree:** review the recorded H01 baseline/contracts and
early H05 feasibility rather than repeat the broad inventory. H02/H03 implementation
and reviewed H05 work proceed alongside the first H04 read slice; H06 follows
feature contracts, not a rigid U-number sequence. H07 starts with that slice and
grows with coverage; H08 requires complete candidate workflows before live cutover.
U17 integrates evidence, not everyone's postponed native gaps. Independent stock
adapter/UI fixes and already authorized local checks can continue; no new parallel
P product suite or re-execution of accepted U08.

## 3. H01 — Audit before writing more screens

The first full source-coverage pass is now in the existing
[coverage register](forgejo-api-coverage.md), including authentication/admin,
settings authority differences, boards/review/wiki/package sub-actions and v16's
usable Actions additions. Continue that register rather than maintaining a second
matrix. The following rules apply to contract review and every later feature:

- Enumerate required workflows and every action, including defaults, pagination,
  attachments, validation, denied operations, expiry and reauthentication.
- Trace selected upstream router → middleware → handler → native implementation
  and configuration. Inspect alternate supported protocols and maintainer/design
  history before assuming a new endpoint is necessary.
- Use the register's precise classes: **1** suitable stock API; **2** bounded
  adaptation/composition of an existing interface; **3** reviewed native addition
  or extension; **4** specified UI fields whose Soda adapters already exist.
  Class 4 is not a general label for all missing Soda code. A schema omission is
  not proof of absence, and an unfinished screen is not an API limitation.
- State whether web behavior mixes rendering with computation/authorization.
  Name the smallest extraction needed for shared native functionality, its
  callers, effects, authority and tests. No blanket internal-function exposure.
- Select the initial upstream baseline with security/support compatibility in
  mind. 15.0.7 is installed; inspected 16.0.3 does not solve U09 by itself.
  Neither an upgrade nor an old-version support commitment is selected here.

**Recurring rule:** every new feature starts with this same contract trace before
its UI implementation. Missing coverage is surfaced early to the owning milestone.
H01 does not need every UI detail implemented before the first bounded slice, but
must identify the authentication/admin design risks and ownership up front.

## 4. H02 — Make the patched dependency maintainable

1. Review source baseline, patch scope, tracked paths, runtime/build dependencies
   and named maintainers. Use Forgejo's actual build and packaging contracts,
   not guessed Go-only compilation that omits embedded assets/migrations/features.
   Build tools for Forgejo may differ from Soda's pins; keep them source-owned.
2. Implement verified source retrieval/extraction into a fresh directory. Refuse
   wrong digests, unsafe archive paths, occupied output, missing/reordered patches,
   unexpected patch base and partial patch application. Never patch a live tree.
3. Keep each patch cohesive: shared native implementation extraction, API handler,
   schema and tests together where needed. Add a reason, upstream reference if one
   actually exists and removal condition; no running fork of duplicated rules.
4. Compile/package natively with actual platform/tool/input identities. Preserve
   Forgejo UID/data layout, entrypoint, Git SSH behavior, migrations and configured
   protocols. Check both architectures independently; cross-compilation is not
   installed aarch64 evidence.
5. Wire existing build/staging/installer consumers and notices as one change.
   Record Soda revision, upstream commit, patch-set identity and image digest
   separately. No floating tag, filename or advertised API version proves bytes.

**Tests:** bad integrity, reordered/conflicting patch, source mismatch, missing
build dependency/output/license, incorrect architecture, stale archive and image
identity mismatch. Real build metadata only; unit fixture success is not a native
build or a compatible installation.

## 5. H03 — Small explicit compatibility boundary

Define a minimal read-only interface description in the Forgejo extension and
its documented schema. Its exact route is reviewed with the first patches, not
asserted to exist today. It reports only the implemented extension contract
revision and required feature contract revisions (initially blame/comparison),
not native roles, installation secrets or a dynamic command catalog.

Soda requests it through its configured Forgejo client with the appropriate
native authentication. Review pre-authentication compatibility with H05 so the
login contract does not depend on obtaining an ordinary grant it cannot yet issue.
No browser-supplied provider origin or token. Keep requirements next to their
concrete feature clients; no plug-in registry, arbitrary
capability dispatch, interchangeable providers or multiple dormant backends.

- Missing endpoint/contract: clear incompatible-provider result for that feature;
  no guessed request, HTML fallback, clone, operator token or mutation retry.
- Transport failure: unavailable, not “unsupported.” 401/403: authentication/access
  error, not an excuse to try another interface. Malformed metadata fails closed.
- Unknown additive metadata may be ignored; missing required semantics or an
  unsupported breaking revision must not be guessed compatible.
- Do not turn one missing optional-to-that-request feature into a global outage
  of working login, stock API reads or environment access. Final delivery still
  requires all selected workflows; graceful development failure is not acceptance.
- Avoid durable capability rows. Read at a bounded connection/session boundary;
  if reused in memory, define finite freshness and invalidate on reconnection or
  mismatch so a provider replacement cannot leave a permanent stale assumption.

**Compatibility is not authorization:** each operation still uses native resource
checks. **Advertisement is not proof:** native contract tests verify responses and
semantics against exact built bytes. Preserve stable Soda-facing DTOs across
upstream updates through small explicit field translations, not copied business
rules. Change a mapping deliberately and test it; do not silently select runtime
fallback implementations based on response shape.

**Tests:** supported, missing, obsolete, malformed, unauthorized, forbidden,
network-failed and changed-provider cases; zero writes or privilege escalation
from any probe or incompatibility path.

## 6. H04 — First complete native-backed read slice

### Forgejo implementation

Refactor only the needed computation out of HTML handling, retaining its existing
web caller and tests. Add handlers under Forgejo's normal native API authority.
Reuse the existing blame/comparison/diff engines; no Soda database changes, clone
cache, temporary PR, copied Git algorithm or direct filesystem access from Soda.

Review contracts covering:

- Blame: full commit SHA, validated file path, attribution spans and original
  lines/paths, native commit identities, ignore-revs behavior and explicit limits.
- Compare: immutable base/head/merge-base SHAs, direct versus merge-base semantics,
  net changed files, renamed/deleted/binary entries and bounded hunks. Existing
  concatenated per-commit entries are not an aggregate diff.
- Bounds on input/file size, underlying work, wall time, output and concurrent
  expensive operations. Line paging alone does not bound blame computation.
  Cancellation must stop/reap native work appropriately, not only stop writing
  the HTTP response. Select actual limits from upstream behavior and measurements.
- Native access to every target and any disclosed historical path/commit;
  no visibility expansion through cross-repository comparisons or caches.

### Soda integration and proof

Extend the concrete clients and web handlers, then add full React attribution,
commit navigation and comparison views. Keep session/route race guards and inert
rendering. Display meaningful binary/large/partial states; do not imply complete
results when limited. Preserve existing history/file-write/fork/import functions.

Test real native repositories for private denial, scope separation, rename,
delete, binary/invalid-encoding, empty/large files, ignore-revs, divergent history,
ref movement, cancellation and contract mismatch. Compare attribution and net
diffs with ordinary native Git. Test Soda DTO/CSRF/session boundaries and React
navigation, errors and stale responses separately. Native fixture mutation still
requires its exact approval; authoring these tests does not run them.

## 7. H05/H06 — Extend the pattern, not the exception list

### Authentication/security

Follow the corrected [authentication delegation contract](forgejo-authentication-design.md): Forgejo owns authentication/IdP/consent/logout/revocation rules, Soda presents and securely adapts them. Do not implement the withdrawn draft account/factor policies or treat missing interfaces as missing policy. Soda maintains its own adapters/patches; upstream security authority is not reassigned to a newly named human.

First produce a reviewed sequence and threat model covering signup/onboarding
under native configuration, first-password change, sign-in, MFA/security keys,
consent, refresh, logout, recovery, email verification and sensitive account/admin
operations. Conditional native features remain required when enabled; disabling
them is not a scope waiver. U05 owns self-account/security views, U16 their
administrator counterparts, and U04 the shared authentication/challenge boundary.
Do not equate inspected second-factor WebAuthn with proven passwordless passkeys.
Identify native challenge issuance/verification, replay/expiry, session fixation,
login-CSRF, rate limits, enumeration resistance and credential redaction. Resolve
WebAuthn RP-ID/origin compatibility and any supported external-provider redirects.

Do not manufacture a password-to-token endpoint, remove native security checks,
borrow sessions or introduce a second credential authority. Where an upstream
mechanism cannot meet the experience, propose that exact native change for review.
Keep the current OAuth flow functioning until its replacement passes isolated
proof; its existing UI exposure remains a delivery blocker, not an exception.

### Other features

Follow H01–H03 for each proven gap. Reuse suitable stock operations without
wrapping them unnecessarily. Prioritize human-authorized Actions data/controls,
webhook update semantics, release clear/download cases and account/admin security;
then close every remaining inventory action. Audit newer interfaces before
patching based on old-version assumptions. Do not mistake empty-field PATCH
semantics for a missing transport, or missing wiki/package UI for missing APIs.

Each added contract ships with native tests, Soda adapter tests, React workflow
coverage and assigned update ownership. No generic “call internal method” endpoint,
provider role mirror or privileged configuration editor.

## 8. H07 — Make upstream changes routine and testable

Run this procedure for each proposed security/feature update; it is a manual,
authorized development procedure initially, not automatic CI or a new updater:

1. Preserve the accepted source/patch/image references and evidence. Fetch the
   candidate to a new workspace; review upstream release/security/migration notes,
   dependency requirements and changes to touched native services/middleware.
2. Check whether upstream now supplies a required interface. If equivalent, adopt
   its stock interface with contract tests and retire the downstream patch in one
   coherent change. Matching endpoint names alone do not establish equivalence.
3. Rebase remaining patches explicitly; review cleanly applying patches too.
   Re-run native functionality, authority and extension-contract tests against
   the new implementation, plus Soda consumer/security/DOM regressions.
4. Build exact native outputs and run compatibility checks. Exercise supported
   old/new Soda–Forgejo combinations needed for the chosen rollout order. An
   incompatible pair must fail clearly before user writes, never silently bypass
   permissions or expose a native page. Do not promise arbitrary mixed versions.
5. With separate permission, rehearse startup/migration on restricted copies of
   populated state and verify identity, grants, repository outcomes and retained
   integration. Record any irreversible schema/config/auth changes and safe
   rollout ordering. Never restore a stale live database over subsequent work.
6. Approve the candidate only with affected evidence; otherwise retain the failure
   and investigate. Assign an owner to security-update blockers instead of silently
   freezing a vulnerable release. Do not weaken checks merely to unblock shipping.

**Exit:** demonstrate the procedure on the first selected upstream update or
reviewed contract-changing candidate. Until performed, upgrade resilience remains
an authored mechanism, not proven evidence. Upstream contributions/publication
are separately approved; proposed patches are not claimed accepted upstream.

## 9. H08 — Delivery, preservation and browser boundary

Use existing core build, staging and installation owners. Before approved rollout,
back up/rehearse consistent populated Forgejo/Soda/config/key inputs and account
for all strict config consumers. Preserve the four U08 environments, existing
repositories, credentials, memberships, dirty work, workloads and prior evidence.
No project image change, new environment, reboot, reset or cleanup is implied.

Verify all required workflows through Soda, including real authentication/MFA/
consent and acting-user versus admin/operator separation. Observe authoritative
native state independently; no mock or metadata row substitutes for outcome proof.
Audit app-owned links, provider URLs, Markdown object links, redirects, downloads,
raw/archive protocols and direct browser access to the upstream origin. Plan the
proxy/listener split so Forgejo frontend routes are inaccessible to end users
without breaking API/Git/package/SSH or separate Cockpit. Do not deploy that split
while current login still depends on provider pages. Update installed browser
fixtures coherently; preserve historical OAuth evidence rather than relabel it.

U17 verifies complete workflows and end-state ingress on the matching approved
candidate/rehearsal target; U18 then performs the preserved default-route/live
ingress cutover. U17 must not depend on a completed U18, nor require blocking live
login before its replacement passes. U20 owns final matching revision/upgrade/
native architecture, terminal and retained-service acceptance. This work package
does not create another readiness certificate or erase prior U08 scope.

## 10. Ready-to-implement review and first commits

U01's baseline/first-read/native-auth/build-license source review is complete.
Use its concrete dispositions; review actual native patch diffs and runtime/build
results at their implementation stage. The review does not prove implemented
compatibility, complete distribution compliance or headless authentication.

Remaining coherent commit sequence — follow the leading plan's
[U01 delta-only closure checklist](dashboard-implementation-plan.md#u01--capability-authority-and-baseline-audit), not a fresh R&D pass:

1. Implement from the accepted U01 source/design review in the existing source
   guide and contracts, without repeating its baseline/auth/read/license review. The user selected Apache-2.0
   for original Soda code; native auth/IdP/logout/ownership rules remain Forgejo's,
   not unanswered Soda policy choices. Fix the leading plan's explicit acting-user,
   account-validation and stale-owner defects under their U owners. Reuse H01 authority findings. Investigate
   only an identified missing or contradictory fact; do not repeat the inventory,
   v16 API comparison or authentication proposal.
2. Retain the implemented source lock/preparer/tests at `c9a9be0`. Extend them with
   the reviewed real build inputs, patch provenance and build-boundary tests;
   do not recreate source preparation as a milestone gate.
3. Wire the native Forgejo build into existing packaging consumers.
4. Add reviewed compatibility and first native read contracts with upstream tests.
5. Connect and test Soda blame/comparison end to end.
6. Implement the reviewed authentication/security boundary and remaining features.
7. Add update/rebase tests and retain exact authorized execution evidence.
8. Perform separately approved rehearsal/rollout/exposure closure and acceptance.

Already authorized local builds/tests do not need another generic permission
request. New patch implementation follows contract review; deployment, real
repository/account/provider mutations, new fixtures, upstream submission and
publication retain separate scopes. H01 ran source/research/document checks only.
H02 now has tested source preparation and a development input lock, not the native
build/packaging spine. H03–H08 product implementation and native validation remain
pending; U01's accepted review supplies their initial implementation contracts,
not a missing-API waiver or native completion claim.
