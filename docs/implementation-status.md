# Implementation handoff

## Authority-boundary correction — documentation only

At the user's request, corrected the leading plan, H05 contract, source guide and
related scope documents after `1275c88`. **Forgejo owns its authentication, external
identity integration, native logout/revocation, account policy, ownership/transfer
rules and upstream security fixes. Soda extends and delegates; it maintains its
own adapters/carried patches without becoming a replacement authority.** Missing
native interfaces remain engineering work, not new policy questions for the user.

- Removed the proposed alternative IdP/logout/transfer policies and generic demand
  to name a new owner of upstream security as U01 blockers. Replaced the H05 draft's
  mandatory custom transaction/schema/account-attempt rules with a native-service
  delegation contract. Retained browser/session/CSRF/PKCE/secret/bounds protections,
  native challenge enforcement, origin/RP-ID preservation and working-login hold.
- Made three inspected code corrections explicit with U owners and regression
  requirements: ordinary shared-admin substitution (U04/U05/U06/U16), independent
  Linux/password rules in native account creation (U05/U16), and stale creator
  authority in environment handlers (U07/U04/U12). Fix active paths before their
  later U18 removal; the correctly delegated JSON admin handler does not excuse
  legacy callers. This is not an exhaustive codebase audit or a claim of fixes.
- Clarified that native organization ownership/repository transfers are required;
  generalized Linux remapping and separate environment lifecycle machinery stay
  deferred. Preserve legitimate Soda memberships, all roots/accounts/keys/workloads
  and distinct native/operator/host authorities.
- Recorded the user's **Apache-2.0 selection for original SodaOS code**. A license/
  notice boundary still needs authoring; all third-party terms remain intact and
  transitive/native/font compliance is not yet established.

Executed the existing read-only document checker: 20 ordered U milestones, three
conditional E tracks, 179 action groups assigned once, 30 local new/plan links and
37 incoming plan links/anchors passed; `git diff --check` passed. Log retained at
`.artifacts/research/authority-docs-1275c88/document-check.log`. These are document
checks, not native evidence. No Go/frontend/native tests or builds, source
remediation, license-file creation, deployment, provider mutation or environment
change is claimed. The selected native input remains 16.0.3; installed Forgejo stays 15.0.7.
All four environments are untouched. Only bounded U08 is accepted (1/20).

The following historical entry records the original U01 delta work/evidence.
Its human-policy blockers and prescriptive auth candidate are superseded by this
correction; preserved release/license/Git experiment evidence is not relabeled.

## U01 delta execution — concrete dispositions, acceptance still held

Implemented the revised closure work starting from clean `d5b5065`, reusing the
existing H01 register, source lock/preparer and auth/read drafts. No repeat route
inventory, archive preparation or replacement design document. Current results
and the consolidated human questions are in the leading plan's **Closure results
— execution after `d5b5065`** subsection; those supersede the blank research tasks
in historical handoffs below.

Completed/advanced:

- Recovered and read complete 16.0.0–16.0.3 release notes through the contents API,
  verifying decoded size and Git blob hash. The formerly truncated 16.0.3 note is
  9,799 bytes, not 8,192. Preserved the old response; recorded migration, proxy,
  import/mirror/CGNAT, hook, OAuth/session and security implications in the existing
  source guide. Retained the single 16.0.3 implementation input/stable-track
  direction; no live upgrade or human maintenance commitment was assumed.
- Inspected actual native npm lock/direct Soda frontend metadata and upstream
  license-generation behavior. Resolved all three absent native npm license fields
  from exact SRI-verified tarballs (MIT). Recorded the 260 upstream Go notice records'
  lack of version/build-tag binding and the generator's tolerated collection errors.
  Selected corresponding-source/notices delivery through existing U02 staging.
  This is not complete transitive Go/native/font license clearance. The absent
  SodaOS top-level license is a concrete copyright-owner decision, not a guessed
  grant or permission to relicense third-party code.
- Complete release notes contradicted one part of the earlier v16 delta coverage:
  verified `is_2fa_enabled` directly in v16 `admin/user.go::SearchUsers` and updated
  AD22 in the same 179-group register. The remaining account/MFA/reset gaps and
  native site-admin authority are unchanged; no full v16 re-audit was started.
- Extended the existing read draft with exact initial routes/DTOs, byte-preserving
  content/identity, native predicates/errors, safety ceilings, cancellation and
  minimal pre-auth-readable compatibility/freshness behavior. Extended the existing
  auth draft with concrete operation/state/binding/expiry/attempt-limit choices,
  explicit native helper/atomicity requirements and review holds. No endpoint,
  native patch, compatibility advertisement or adapter was fabricated.
- Resolved origin handling as per-installation operator configuration with stable
  RP-ID preservation, not a required global production hostname. Recorded stable-ID
  rename/unavailable/invalidation/data-preservation consequences. Source confirms
  project administration uses stored `Project.OwnerID`; transfer succession,
  especially organization recipients, still needs an explicit product decision.

**Executed evidence:** public source/license metadata retrieval and integrity checks;
a bounded synthetic Git/worker experiment on local x86_64/Git 2.52.0. A 24-commit
run-owned repository passed 65,536-byte-line blame, literal shell-metacharacter
path and net diff under the candidate launcher limits. An over-limit allocation
failed with `MemoryError`. A focused follow-up tested a one-second CPU limit
(worker killed) and explicit ready/PID binding before SIGTERM (worker exited).
This proves only the host primitive, not Forgejo parser/API correctness, packaged
Alpine behavior, hostile/worst-case capacity or aarch64. The second probe closes
missing observations without rerunning the first repository experiment.

Original responses/provenance, exact tarballs/license texts, derived metadata,
scripts/logs and both local fixture directories remain in ignored
`.artifacts/research/u01-d5b5065/`. No cleanup was performed. No production source,
manifest/lock pin, installed service, origin, credential or retained environment
was changed; no installed provider/account/repository fixture was created. No full
Go/frontend/native build or product test suite was rerun for these document changes.
Read-only document checks passed: 20 ordered U milestones, three conditional E
tracks, all 179 action groups assigned once, 30 local new/plan links and 37 incoming
plan links/anchors; `git diff --check` passed. Script/log:
`check-documents.py` / `document-check.log` in that same evidence directory. These
checks are separate from the executed primitive experiment and do not validate
native protocol semantics.

**Not accepted (corrected):** U01 still needs concrete native-interface/security
and license/build readiness review. The original license-selection and alternative
IdP/logout/transfer-policy questions are resolved/withdrawn as described above. Exact artifact/native tests
remain with U02/U03/U04/U09/U17/U20; their ownership is not a PASS. Only bounded U08
is accepted (1/20). Continue from these concrete candidates/remaining questions,
not another research or proposal pass. Working login and all four environments
remain untouched.

## U01 plan corrected to avoid duplicate R&D

At the user's request, replaced the leading plan's broad U01 research tasks with
an explicit completed-input / remaining-delta closure checklist. Reuse H01 at
`c832901`, source preparation at `c9a9be0`, and auth/read drafts at `2cf9127`.
Do not repeat the 179-group inventory, v16 API comparison, source lock/preparer or
authentication proposal. The headless work package now follows that same checklist.

Remaining work is decision closure and narrowly identified missing evidence:
complete the truncated release-note/migration material; close uncovered dependency/
license/build-design obligations; assign human maintenance/security ownership;
resolve the existing authentication and linked-identity questions; finish the first
wire/authority/budget/compatibility contracts. Present human decisions together,
and distinguish questions awaiting a decision from questions requiring research.
Any new investigation must name the evidence gap and smallest useful result;
reopening settled findings requires changed inputs, stale decision-critical facts
or contradictory evidence. Actual image assembly/pins, native patches, full
conformance and installed journeys remain with their implementation owners.

This is a plan correction, not U01 acceptance or new product evidence. Only bounded
U08 remains accepted (1/20). No product source, dependencies, runtime state or
retained resources changed. Read-only documentation checks passed: 20 ordered U
milestones, three conditional E tracks, all 179 groups assigned once, 31 local
new/plan links and 37 incoming plan links/anchors; `git diff --check` passed.
No build, test suite, deployment, provider action or cleanup was needed. Earlier handoff sections describe their historical stages; the new
U01 checklist controls next work rather than restarting their research steps.

## Early auth/read contract review drafts — native endpoints not implemented

After source-preparation commit `c9a9be0`, recorded the requested parallel U01/
U04/U05/U16 review in [the authentication design](forgejo-authentication-design.md)
and [first read-contract draft](forgejo-read-contracts.md). Updated the leading U
plan/headless work package to distinguish implemented source preparation from the
still-missing image/native API/compatibility spine. Go owns the source preparer and
its tests through the existing aggregate; no duplicate Python preparer is planned.

The draft native-owned pre-auth transaction covers credential verification,
forced password change, MFA/enrollment/recovery, consent, exchange, expiry/replay,
refresh and explicit logout/revocation ownership. It is **not an implemented or
approved wire protocol**, second password authority or grant for gated accounts.
Concrete decisions remain: production Soda origin and preservation of enrolled
WebAuthn RP-ID, external IdP browser interaction, local logout versus explicit
native-grant revocation, and a named native-auth/security/update maintainer.
Retained identity/project association consequences require the U owners' review;
no Linux remapping/offboarding was selected. Working OAuth/ingress is untouched.

For blame/net comparison, recorded required immutable identities, attribution/
original paths/lines, direct/merge-base and net-file semantics, content/limits and
native authority/cancellation tests. Exact schemas, numeric work/output/concurrency
budgets and broader fork comparisons remain unfinished. Source inspection found
that `BlameReader.NextPart` discards long-line continuation fragments and original
porcelain line/path positions; stopping early also requires careful native command/
pipe cancellation, and ignore-revs copying needs a bound. These are **source
findings, not reproduced native runtime evidence or authored upstream fixes**.
Do not expose a new API by wrapping the current reader or advertise fake extension
compatibility. H03/H04/H05 implementations remain pending the relevant review.

Release/support research, bounded source download and Soda Go tests are the
executed work below. Full release/migration qualification is not complete: the
retained `release-16.txt` response is 8,192 bytes and ends mid-entry, and must not
be cited as a complete v16 release-note review. No second H01 inventory, production
Forgejo compilation, native patch/deployment/provider fixture or frontend change
occurred. Local document link/anchor/179-group checks and `git diff --check` were
run for these drafts; they establish consistency, not protocol feasibility.

Next implementation after review: finish pinned native build/runtime/license/
source-delivery inputs and existing image consumers; write shared native fixes/
contracts/tests, minimal real compatibility and the Go→React U09 slice. The
requested batch is **partially implemented**, not complete; only U08 is accepted.

## H02 source preparation started — development candidate only

The user authorized implementation of the next-work plan after `2ff9e44`, including
local builds/tests but **not deployment, retained-project changes, lifecycle or
cleanup**. Started from that clean tree. The first implementation is core-owned
Forgejo source preparation, not completion of the planned batch or of H02.

- Added `appliance/forgejo/source.lock.json`, its README, the Go build-only
  `tools/soda-forgejo-source` entrypoint and `internal/forgejobuild/` with focused
  tests. No new Go module/frontend dependency, provider API, application service,
  patch, native image recipe or aggregate-build/staging change.
- Locked **16.0.3 as a development candidate only**. A fresh commit-addressed
  Codeberg archive download matched H01's SHA-256 and Git PAX commit metadata.
  Public support metadata inspected on 2026-09-07 reports stable support through
  **29 October 2026**, versus installed 15.0.7 LTS through **15 July 2027**. The
  useful v16 human Actions APIs favor avoiding backports, but release adoption
  needs a named supported-release/security owner, complete dependency/license/
  migration review and installed proof. This is not approval of an upgrade.
- The preparer verifies bounded lock/archive/patch snapshots, extracts only safe
  source entries, applies a contiguous reviewed patch series with native Git in
  an isolated fresh output, and writes a content/type/mode/link/lock receipt only
  after success. HTTP is public locked-commit-only, finite and no-redirect/no-retry;
  malformed inputs and partial failures cannot become prepared output. Failed
  attempts are preserved. The empty patch list advertises no native extension.
- Existing `forgejo.iid`/OCI/staging/sealing/install paths are not rewired yet.
  Native build-image input pins, complete upstream frontend/Go/runtime build,
  corresponding source/notices payload and downstream tree/identity verification
  remain H02 work. A prepared source receipt is not a native binary, image,
  bit-for-bit reproducibility claim or provider conformance result.

**Executed local evidence:** Go **1.26.7** focused tests (15 top-level tests), focused
race checks and `go test -mod=readonly ./...` passed. The aggregate reused cached
unchanged-package results; the new source tests ran. The real v16 commit archive
passed both the local-file and public-HTTPS preparer paths; their source/lock
receipts match. These `go run` invocations compiled/executed the build-only Soda
preparer, not Forgejo. No frontend or full native build/check pipeline was run.

The first real preparation failed with `unexpected archive root`: Go's tar reader
exposes Git's global PAX header. Kept that output/log, then explicitly validated
its commit metadata and reran into fresh outputs successfully. No evidence was
replaced or removed. Logs, the archive/provenance and failed/successful preparations
are in ignored `.artifacts/research/headless-2ff9e44/` (`source-tests-first.log`,
`source-tests-pax.log`, `source-tests-complete.log`, `source-race.log`,
`prepare-local.log`, `prepare-local-pax.log`, `prepare-download.log`, `all-go.log`,
`source-tests-final.log`, `source-race-final.log`, `all-go-final.log`).
Source hash for matching successful preparations:
`373a15421f722aede9e9399cb135be169bbd71e2facedeed342f62c9a4d4dd3f`;
lock hash `1639438adc2de03edc7545e023b7b62c97f6ed93c93a17ae05b0174cf77ec259`.
Those identify this source preparation, not installed bytes.

Installed affected components remain last recorded at `8b823db`; Forgejo service
source remains 15.0.7. All four environments, logins, private credentials, keys,
checkouts, workloads and failed native evidence were untouched. No installed
provider/fixture/account/repository action, live network/origin/service change,
reboot, retained-resource cleanup or publication occurred. H01 remains complete source discovery; **only bounded U08 is
accepted (1/20)**. H02 is partial; H03–H08 product implementation remains pending.

## Dashboard implementation plan revised after H01

At the user's request, revised `docs/dashboard-implementation-plan.md` against the
**179-action-group audit at `c832901`**, starting from that clean tree. Retained
U01–U20 and the single action register; no new milestone count, implementation
stack, provider authority or acceptance claim. The leading plan now assigns every
H01 group a primary U owner and names its cross-feature callers.

The remaining sequence is now explicit:

1. Review supported Forgejo baseline, concrete contracts/resource limits,
   linked-resource effects, license/build obligations and named security/update
   maintainers. Do not repeat the broad H01 inventory or infer an approved v16
   upgrade from its usable human Actions additions.
2. Start U04/U05/U16 native authentication/challenge/consent/security design early,
   alongside U02/H02 source-build and U03/H03 compatibility. First-password/MFA
   gates and WebAuthn origins are architectural risks, not late account forms.
3. Deliver the first U09/H04 blame/net-diff slice and reviewed authentication path;
   keep suitable stock APIs, bounded adapters and existing-adapter/UI-only fixes.
   U09 now covers the audited code/ref/restore/notes/patch/copy/task workflows, not
   only its earlier personal-fork/basic-import subset. Its eventual rollout must
   account for the reviewed Forgejo candidate, not assume dashboard-only changes.
4. Complete feature-owned batches: U10 boards/time/dependencies/history, U11 native
   review/viewed/merge state, U12 settings/hooks/invitations, U13 search/graphs,
   U14 baseline-aware Actions, U15 wiki/packages and early/complete U16 admin.
   U17 integrates evidence rather than implementing everyone's postponed gaps.
5. Demonstrate update/rebase/native contract compatibility as patches grow. U17
   verifies complete candidate workflows and end-state ingress rehearsal; U18
   performs the separately approved retained-installation cutover. This avoids a
   circular prerequisite or closing live Forgejo login before its replacement.
   U19 improves measured tasks; U20 requires final native proof on both targets.

Assigned the separately requested **existing-workspace browser terminal to U07**,
with U02/U03/U04 build/security review and U17/U20 coverage. Transport, native
account binding and lifetime/security/stopped behavior still need review; no
mechanism, implicit create/join/start or native shell execution was selected.
U08 remains accepted only for its earlier bounded x86_64 scope (**1/20**). Retained
its evidence-link headings and concise revision/criterion reconciliation; the
original detailed execution checklist remains in Git history, not a new run order.
E01–E03 and optional media remain unselected.

Aligned the page/dependency inventory and headless work package with the revised
order and actual resolved/installed subsets. Clarified in the deferred guide that
Soda cross-system/Linux offboarding deferrals do not waive native Forgejo security,
invitation or identity workflows. Updated subordinate support ownership/artifact
contracts for the planned core-owned Forgejo build and terminal, and repaired two
historical incoming snapshot links. No second P implementation or readiness gate.

Checks: read-only documentation validation confirmed **20 unique U milestones,
3 conditional E tracks, all 179 audited groups assigned exactly once**, local
new/rewritten-plan links/anchors and all tracked incoming plan links; `git diff
--check` passed. Script/logs are ignored research artifacts under
`.artifacts/research/dashboard-replan-c832901/`, not a product test suite. These
checks establish document consistency, not native semantics or feasibility.
No Go/frontend/native builds/tests, dependency changes, native patches, deployment,
provider/account/repository actions, fixture creation, network changes, reboot,
cleanup or push occurred. Installed affected components remain last recorded at
`8b823db`; all four environments, private inputs, workloads and failed evidence
were untouched. Existing local build/test authorization remains valid.

Next: the concrete baseline/contracts/maintenance and early H05 review above;
H02–H08 remain unimplemented and every U milestone except bounded U08 remains
unaccepted. This revision is the implementation plan, not proof that those
architecture decisions or workflows are complete.

## H01 — full source workflow audit recorded

At the user's request, replaced the family-only `docs/forgejo-api-coverage.md`
inventory with **179 action groups**, including all selected frontend families
and additional native web surfaces: authentication/consent/security, profiles and
keys, code/copy/search, issues/reviews/boards, settings/hooks/organizations, work,
Actions/runners, releases/wiki/packages, site administration and browser closure.
Each group names interfaces/native owners, authority/configuration constraints,
classification, owning U milestone and required native/adapter/browser coverage.
This remains the single register, not another roadmap or readiness counter.

Started from clean `0f43b9f`. Retrieved complete public sources at v15.0.7 commit
`d4de9eb2a87c26b402fdd0259e079957f8cd2b4b` and v16.0.3 commit
`eccddb2d17c93b42b2c8995725e03e549ac9ec0c`; exact archive hashes/source provenance
are in the register, with research under `.artifacts/research/h01-0f43b9f/`.
The v15 source-surface audit includes routers/middleware, native handlers/services,
DTOs, configuration and alternate protocols; v16 is a targeted delta/authority
comparison, not a second full qualification. Schema indexing found 314 paths/491
operations versus 326/506, **not** that many proven workflows. No source archive
hash was promoted to an image identity or a new dependency lock.

Key findings that supersede earlier coarse gap descriptions:

- Native API middleware blocks first-password-change and required-MFA enrollment;
  WebAuthn JSON is native-session/origin-bound. Basic auth rejects security-key
  users. Headless authentication needs reviewed native challenge/consent/security
  contracts, not a password-to-token or administrator-token workaround.
- V16 adds human Actions jobs/logs/artifacts/cancel APIs with native repository/
  unit gates; it still lacks complete step/attempt/rerun/workflow/trust coverage
  and the U09 blame/net-diff interfaces. No upgrade was selected.
- PAT listing is usable without the creation/deletion Basic-auth gate. Repo
  Actions configuration/runners are owner-gated in REST versus admin-gated web
  settings; that actor mismatch needs native review, never Soda elevation.
  Repository deletion uses owner user/organization scope; package scope needs its
  own consent. Existing review replies and advanced Soda protection/team fields
  must not be mislabeled as missing native APIs.
- Native gaps also include boards, resolve/viewed review state, merge-panel data,
  selected settings/invitations/hooks, historical wiki content/search, package
  descriptor/settings and substantial admin/security operations. Wiki has no
  native CAS in either web/API edit; release notes clearing is a definite PATCH
  semantic gap, not permission to assume every empty field should be accepted.
- Current issue `native_url` downloads, account onboarding instructions, OAuth,
  legacy/content links and full Forgejo proxy ingress remain Soda-only closure
  gaps. The TSX `forgejo_url` guard alone does not prove closure. No live login or
  ingress was changed. Conditional settings/lifecycle questions were recorded,
  not silently disabled, waived or implemented as identity remapping.

Updated the linked architecture/headless/core plans, page inventory and AGENTS
with the audit result and review-first next step. Corrected stale pre-React
unbuilt/auth wording without relabeling historical native evidence.

Checks: source/schema comparison, documentation row/unique-ID/local-link checks,
182 explicitly prefixed upstream file references and 112 named native functions/
types checked, and `git diff --check`. Early reference checking caught abbreviated
paths and an incorrect blame symbol; corrected to exact paths/`RefBlame` before
final checks. Research/check scripts and logs are ignored local artifacts, not a
new production test suite. **No Go/frontend/native build or test suite, dependency
resolution, upstream contribution, provider mutation, deployment, fixture,
reboot, network change, cleanup or push** occurred. Existing local build/test
authorization remains valid. Installed affected components remain last recorded
at `8b823db`; the four retained environments and U08 evidence were not touched.

H01 source coverage is now recorded, but field-complete contracts, measured work
limits, authentication threat model/feasibility, baseline/security/support and
maintenance ownership, linked-resource lifecycle decisions and installed
conformance remain open. H02–H08 are unimplemented; U01/U09/U17 are not accepted.
U08 remains accepted in its bounded scope (1/20). Next: review those concrete
baseline/contracts/H05 gates before broad UI expansion; retain working stock
adapters and no-Forgejo-frontend requirement, not a codebase/system rewrite.

## Headless implementation and upstream-change resilience plan authored

Added `docs/forgejo-headless-implementation-plan.md` at the user's request, linked
from the architecture revision, leading core plan, coverage register and AGENTS.
It turns the proposed boundary into H01–H08 work packages assigned to existing
U owners: workflow audit, verified source/patch build, minimal explicit interface
compatibility, first blame/aggregate-diff slice, parallel authentication/security
design, remaining coverage, recurring upstream-update tests and preserved rollout.
These are tasks, not a second milestone count or P-owned readiness gate.

Inspected the actual Forgejo service reference and native build caller: Forgejo
is currently pulled alongside Caddy. The plan specifies a reviewed source/patch
build feeding the existing image/IID/staging contracts, without changing Caddy or
adding a permanent stock/patched selector. Proposed tracked lock/patch/build/test
paths do not yet exist. No hashes, API endpoints, maintainers, version selection
or successful compatibility evidence were fabricated.

“Future-proofing” is bounded: detect missing interfaces before screens, explicit
contract revisions with safe failures, shared native authority/implementation,
real contract tests on every proposed update and patch retirement when upstream
supplies an equivalent interface. No plugin framework, discovery platform, clone
cache, role mirror or automatic CI/update mechanism. Authentication remains its
own design gate rather than being inferred from the two read APIs.

Documentation only: source inspection and `git diff --check`; no tests/builds,
dependency resolution, patch implementation, deployment, provider mutations or
infrastructure changes ran in this turn. Prior local build/test authorization
remains valid. Current installation, four projects and U08 evidence are unchanged.
Next: H01 workflow audit, concrete contract/baseline/maintenance review, then the
specified implementation commits; no renewed native-frontend fallback decision.

## Headless Forgejo architecture revision plan authored

At the user's request, added `docs/forgejo-architecture-revision-plan.md` and linked
it from architecture, the leading dashboard plan, inventory and prior integration
research. Clarified AGENTS/core guidance: preserve Forgejo business-rule ownership;
review bounded Forgejo-side API additions rather than assuming the stock REST API
can supply complete UI parity. This is a proposed architectural revision, not an
implemented patch, approved upstream version change or new authentication design.

The plan defines concrete ownership, a single workflow/action audit, shared native
functionality behind web/API handlers, endpoint contracts and work limits, a
separate full authentication/security gate, first U09 blame/aggregate-diff slice,
patch/build/security-update responsibilities, preservation and Soda-only ingress
acceptance. It assigns work to existing U01–U20 owners; no parallel milestone count,
P product suite, new release framework or codebase/VM reset. Stock APIs and working
Soda source remain reusable. No missing-feature/native-page waiver is introduced.

Documentation only: source inspection and `git diff --check`; no builds/tests,
dependency changes, deployment, provider operations or infrastructure mutations in
this turn. Existing local build/test authorization remains in force; previous
passing checks are not new execution evidence. Installed `8b823db` and all four
U08 environments/evidence remain untouched. Next: review the proposed boundary,
complete the workflow audit and select concrete contracts/maintenance ownership
before implementing a patch; design authentication in parallel, not as a later
surprise.

## Complete Soda frontend — requirement enforced and newer APIs investigated

The user rejected **all Forgejo frontend fallbacks**, including temporary ones.
Every Forgejo-backed developer/admin workflow must remain in Soda; this includes
login/password changes, MFA, consent and account security. Missing APIs require
integration work, not omitted features. Updated `AGENTS.md`, architecture, the
page inventory, leading plan and coverage/API contracts. The earlier `1d74a08`
native-link proposal below is superseded, not an outstanding product choice.

Removed explicit Forgejo-page links from React history/comparison, shell,
account/consent, Actions, hooks, PRs and releases. Retained truthful unavailable
states where integration is missing; these are **not implemented replacements**.
Added Soda-only history/compare DOM assertions and a production-component source
regression guard. Existing OAuth redirects, legacy HTMX, provider/content URLs
and direct browser ingress still need closure; removing links alone does not
fulfil the no-Forgejo-frontend requirement. No live authentication or ingress
configuration was changed, and no password/permission authority moved into Soda.

Inspected newest returned stable **Forgejo v16.0.3** and development commit
`bdc33af0c11568873c336137d404fc327ce0a40e`: API router/schema and comparison/commit
handlers still do not supply blame data or aggregate comparison patches. Audited
v16 web blame/compare engines as candidates for upstream API exposure. Its auth
implementation has changed and needs its own full review, not assumed v15 gates.
No upgrade selected. `docs/forgejo-frontend-integration.md` records public source
provenance, proposed Forgejo-owned bounded read interfaces, maintenance/build
costs and remaining authentication/security/Actions/admin coverage. No upstream
issue, patch, contribution or replacement Git backend was implemented/published.

**Local builds/tests are now explicitly authorized** on this development
machine. Performed on the working source based on `1d74a08` plus this follow-up:

- Go 1.26.7 `go test -mod=readonly ./...`: passed, including prior U09 cases.
- Dashboard TypeScript `tsc --noEmit`: passed.
- Dashboard Vite+ tests: **31 passed across 15 files**. First attempt had 30 pass
  and one source-guard failure because jsdom rewrote `import.meta.url` to HTTP;
  selected Node environment for that filesystem test and reran successfully.
  Both failed and final logs remain; no application guard was weakened.
- Dashboard production bundle and `soda-dashboard` Go compilation: passed.
  Fresh local outputs under `.artifacts/checks/soda-only-1d74a08/`, preserving
  prior `dashboard/dist` and native deployment stage. These are local build
  outputs, not a sealed/staged appliance or deployed candidate. Vite retained
  its >500 kB chunk warning and outside-project output-directory warning.
- `git diff --check`: passed. Logs under `.artifacts/logs/soda-only-*`.
  Existing pinned Node 24.20.0/pnpm 11.25.0/dependency cache used; no upgrades or
  dependency manifest/lock changes.

No full native image/staging pipeline, installed browser verification, deployment,
repository/provider mutation, project/lifecycle/network change, cleanup or push.
Installed affected components remain `8b823db`; all four U08 roots/evidence remain
unchanged. U09 and complete-frontend acceptance remain open. Next: review the
concrete API integration proposal, continue feature/test work and design the
headless authentication/security boundary; no native-page waiver is available.

## U09 implementation started — upstream contracts and source corrections

Implemented a first source/test batch from clean `5d9dfe6`, following the user's
implementation request. Existing installed bytes remain `8b823db`; U08's recorded
acceptance is unchanged. **No builds, compilation/type checks, tests, dependency
installation, deployment or native/provider mutations ran in this batch.** Only
upstream source/metadata retrieval, source inspection/edits, Go formatting and
Git whitespace review occurred. The remaining native actions require their exact
repository/import-source/build/rollout approval.

- Audited selected Forgejo 15.0.7 comparison/commit/blame/fork/import code. Native
  comparison returns the whole bounded commit list and concatenated per-commit
  file entries, not net changed files or a paginated aggregate patch. Native fork
  returns 202 after synchronous clone; migration also runs synchronously, applies
  native allowlist/authority first and owns its failed-destination cleanup.
- Comparison now resolves refs to validated full native SHAs (same ref resolved
  once), compares only those snapshots, verifies total/list consistency and
  rejects malformed identities without another request/retry. UI exposes pinned
  identities and honestly labels duplicate/reverted per-commit file entries.
  Branch/tag GETs use read scope while writes retain write scope and native gates.
  Commit/ref mutation results are validated instead of trusting empty metadata.
- Fork/import results must identify a valid personal destination; forks require
  native fork/non-mirror flags. Wrong/unknown destination results remain
  unconfirmed writes, not optimistic success or a mutation retry. Import failure
  keeps non-secret form inputs while immediately clearing credential inputs.
- File saves, forks and ref mutations track their target lifetime: late replies
  cannot navigate/reset a different repository/file/account. History/ref paging
  resets stale next-page state, and diff/ref-write expiry invalidates the session.
- Added Go comparison/scope/result/copy/credential-error cases, dedicated
  history/ref/compare/copy DOM suites and file-editor route-race/UTF-8-bound tests.
  These tests are **authored, unexecuted**, not passing evidence.

**Concrete upstream decisions:** `U17-BLAME` and `U17-COMPARE-DIFF` now record
native HTML views without supported OAuth API equivalents in the inspected
version. Full-SHA/configured-origin native links are present and explicitly
labeled interim; no HTML scraping, cookie borrowing, local Git backend or API
endpoint guess was added. Acceptance of native views for this version versus
waiting for upstream capability/revising scope is unresolved. U09 is not complete.

Remaining source work includes the rest of the planned mutation/read edge-case
matrix and the installed two-user UI/Git/fork/import fixture entrypoints, followed
by approved execution and real native ref/byte/permission/readiness evidence.
The current work does not claim full custom blame/aggregate-diff parity or accept
those gaps silently. Contract details and evidence provenance are in
`docs/forgejo-api-coverage.md`; changed DTO/scope semantics in
`docs/dashboard-api.md`. Browser terminal work remains separate.

## U09 completion plan recorded

Added a source-backed completion sequence under U09 in the leading dashboard
plan: exact upstream capability audit; history/blame/commit/compare read coverage;
SHA-bound writes and route/account-safe forms; personal fork/one-time import;
focused Go/DOM/installed tests; gated matching build/backed-up dashboard rollout;
real two-user native Git verification; explicit final acceptance/dispositions.
Inspected current history/file-write web/provider code and editor/fork/import UI
plus focused tests. Blame, aggregate comparison/paging and dedicated read/copy
DOM coverage remain gaps, not newly implemented features.

The proposed native run needs separately approved repository-only fixtures and
an allowed HTTPS import source. It preserves all U08 projects/data and requires
no new project environment, lifecycle operation, provider job or cleanup. The
browser workspace terminal stays outside U09. This was documentation/source
inspection only: no builds, tests, dependency changes, deployment or native/provider
operations ran. U08 remains accepted; U09 remains incomplete.

## Release-day public handbook authored

Added `docs/public/`: five sections, 22 published pages plus the unpublished
structure/ingestion README. Studied the predecessor handbook and the website's
actual discovery, Markdown validation, snapshot/provenance, asset and rendering
contracts. All 17 former page slugs remain, with new dashboard, operator setup,
collaboration, shared-tool/file and project-service guides. The old checkout's
unrelated dirty/unmerged work was not modified.

The handbook describes the release-day Forgejo identity/unified dashboard,
explicit project-local joining, direct-IP SSH, shared installations and native
workload model, with root-only Cockpit/Tailnet/Runners. It includes the requested
existing-workspace browser terminal as product intent, not implemented evidence.
It does not restore predecessor host developer accounts, managed clones,
Anaconda/cloud-init/bootc recipes, deletion workflows or reserved Updates work.

Retained ISO/QCOW2/Scaleway and equal-architecture release-day deployment paths
without claiming current media or inventing signing identities. Exact media,
platform Ignition delivery, verification trust and real-interface capture work
are recorded only in `docs/public-docs-review.md`. This is editorial guidance,
not authorization or proof of installation, release delivery, terminal execution,
new U/E/P acceptance or a backup/update platform. Broader website marketing
alignment remains separate; no product concept was removed as a readiness caveat.

Validation: the website's real `snapshotFromDirectory` parsed the draft in memory
(5 sections, 22 pages), validating Markdown shape and all relative page/heading
links without writing a fictitious revision-bound snapshot. `git diff --check`
passed. Source committed at `ba2c5d6`; website `65fb5db` ingests that exact source
through the CLI, adds source-link/retained-route regressions and points Dashboard
guide to `/docs/dashboard` instead of operator Cockpit. Source freshness and
snapshot integrity passed. Website `vp run test` passed 49 Node ingestion/
architecture tests and 48 component/page tests; `vp run typecheck` and
`vp run build` passed. No browser/visual or real-interface screenshot review ran.
Neither repository was pushed or deployed. No appliance native build/test,
VM/service/network, provider, image-generation or destructive operation ran for
this work. The original source revision remains the snapshot's valid provenance;
this subsequent internal evidence entry changes no public handbook bytes.

## New requirement recorded — browser workspace terminal

The user requested browser terminal access after selecting a project, when their
workspace already exists there. Recorded in the dashboard page inventory and
leading plan, replacing the prior terminal exclusion. The terminal targets the
user's existing project-local account/home, not a new environment or host shell;
server-side access checks and explicit joining remain required. Mechanism,
session/security design, stopped-project behavior and milestone placement are
undecided. No implementation, dependency change, test, build or native operation
was performed. Existing U08 acceptance does not cover this new requirement.

## U08 accepted — bounded native x86_64 first product proof

**U08 is complete for the recorded infra → `soda-test` first-product scope.** This
is not U20/release acceptance, a fresh appliance installation, SPA cutover or
independent aarch64 proof. The criterion-by-criterion reconciliation is in the
[leading plan](dashboard-implementation-plan.md#u08-closure-reconciliation--merged-candidate).
Other U milestones remain unfinished; acceptance of this narrower installed
journey does not pass their broader feature/security matrices.

Built/sealed and ran the full native aggregate at clean **`8b823db`**: Go 1.26.7,
Cockpit 60 tests, dashboard 21 tests, 30 Python build tests and 9 staging tests
passed. Current installed dashboard image, host helper, **runner companion** and
default new-project image match that candidate. Restricted populated-v3
DB/config/key/helper/runner/unit/prior-image backups and isolated startup rehearsal
preceded rollout; all identity/key/project/membership/session/encrypted-grant rows
matched. Only affected components were deployed; other retained appliance bytes
are not claimed as a whole-host upgrade. `/app/` remains preview; HTMX is default.

After rollout, complete `host.sh` substrate/permission/listener checks passed;
artifact binding was checked separately for the exact four changed components,
not through a false full-host `verify-installed`. Trusted TLS/asset-missing/API-401
checks, independent operator/Alice/Bob OAuth/navigation/logout, authenticated
connection authorization/public keys, native root Cockpit login/navigation/logout
and root-versus-existing-nobody PAM account checks passed. Alice's first repeated
connection observation refused an existing output (`EEXIST`); it was repeated in
a fresh private **observation directory**, without resetting fixtures or pins.

Direct own-key SSH/PTY/SCP/SFTP and cross-project/sudo denials passed again.
Different-UID/default-user/SQL/PTY exec and Bob's engine denial passed; explicit
`workloads.sh check` passed without another up/build. Infra and both users read
the exact retained committed PostgreSQL rows and live HTTP content. Both users'
own project-local Git agents authenticated and returned their original native
remote refs; shared executable device/inode/ownership/version matched. All four
projects' preexisting declared state and Soda records survived; the only additions
were the explicitly run SSH/file-transfer probe files in the newest project.
The three older roots matched exactly. Boot ID did not change.

**Lifecycle evidence disposition:** reuse the real f233a4a stop/start/reboot
results for unchanged persistence mechanisms, not as a newly executed c96c108 or
8b823db reboot. f233a4a→c96c108 changes only the fixed create-time SYS_PTRACE
capability; c96c108 fresh boot and different-UID tests exercised that change.
The merge changes no project runtime/rootfs/start/stop source. A strict image
layer-ID equality probe failed and is retained: rebuilding changes Tea and tar
metadata. Per-layer content/mode/owner/link/capability comparison then established
only `/usr/local/bin/tea` has changed content; runtime dependency layers and
project configuration remain identical. The new immutable image's native Tea
0.15.1 version check passed; its stopped, network-disabled diagnostic container
is retained on infra. It is not a new project. No lifecycle rerun or additional
project is needed to close unchanged U08 assertions; U20 owns final repetition.

**Operator gap disposition:** native RPM capability queries prove installed
nodejs22/zlib-ng-compat satisfy nodejs/zlib; missing-provider regression tests keep
failure strict. Updating the stale runner command restored native list output:
zero runners/listeners/capacity. `NeedsLogin` is the observed Tailscale state, not
enrollment proof. The initial operator script incorrectly printed completion
when its console hook was missing. Added `-e` to its inner shell and a regression
for missing/failing/noisy/quiet hooks. The corrected script now **fails** on absent
`/etc/profile.d/soda-console-welcome.sh`; do not count the earlier apparent pass.
That preexisting console delivery/interactive-review gap remains P11/U20, outside
U08's preserved Cockpit-service criterion. No console files, providers, networks
or runner services were changed to hide it. Full P11 acceptance is not claimed.

**Evidence:** `.artifacts/logs/u08-closure-*`, including build/check `.exit` files,
`rollout-8b823db`, `host-regression`, `operator-corrected`, `browser-*`,
`cockpit-{browser,pam}`, `static-api-boundary`, `developer-access`,
`different-uid-exec`, `retained-client-reads`, `workload-check`,
`{original-three,four-project}-preservation`, `project-image-{delta,content}`,
`tea-image-version` and `final-byte-binding`. Private backups/rehearsal are under
`/var/lib/soda/u08-completion-8b823db/`; browser observations under
`.artifacts/test-vm/u08-closure-8b823db/`; both completion fixture directories retain
before/after snapshots. Earlier c96c108 artifacts moved intact to
`.artifacts/retained-native-u08-c96c108/` before the build. No evidence was resealed
or failed result overwritten. Post-build operator-test correction/documents do
not alter deployed production bytes; all **31 current Python build tests** passed,
including the new failure regression. No reboot, fixture creation, enrollment,
provider job, project replacement, cleanup or publication occurred.

## U08 closure implementation — merged-candidate preparation

User authorized the closure plan on infra and `soda-test`, using retained fixtures
without another project, lifecycle action or provider enrollment/job. Added the
criterion/evidence/delta table in the leading plan. Runtime source comparison
supports reuse of recorded f233a4a persistence for unchanged start/stop/rootfs
mechanisms, alongside c96c108 fresh-boot/different-UID exec proof; no new reboot
result is claimed. Current merged frontend/build/support changes still need their
own build/check and affected installed verification.

Read-only native diagnosis found real providers `nodejs22` and `zlib-ng-compat`;
installer/host preflights now query RPM capabilities for nodejs/zlib, with focused
replacement/missing-provider tests. Installed `soda-runners` lacks the new config
field `grant_key_file`; it needs matching companion deployment, not weaker config
validation, enrollment or a second configuration. Other installed config.Load
consumers were inspected; the dashboard and runner command are its callers.
The pinned Go 1.26.7 merged suite passed before these shell/test changes. Evidence:
`.artifacts/logs/u08-closure-{merged-go,package-observation,rpm-provides,runner-config-observation}.log`
and runtime/merge delta diffs. Build/rollout results will follow, not inferred.

## Requested core/support integration

Merged the native support branch through `3d7ca2e` with the local U08 lifecycle
record and different-UID exec preparation through `c96c108`. Preserved both
handoffs below; their evidence applies only to their named candidates. This
integration performed conflict/whitespace review, not builds, product tests or
native operations. The combined candidate remains unvalidated. The pre-existing
uncommitted `tests/installed/workloads.sh` edit is retained separately.

## U08 different-UID exec verified — c96c108

The user approved project-namespace-scoped SYS_PTRACE and **one further fresh
fixture** on `soda-test`. Current installed dashboard/helper/default project image
are **`c96c108`**. Full native x86_64 build/seal and aggregate check passed (Go,
Cockpit 60 tests, dashboard 21 tests, 20 Python build and 9 staging tests). A
consistent populated-v3 backup, isolated startup/preservation rehearsal and
matching rollout preserved all existing rows and project/Forgejo/proxy identities.
No first-install/bootstrap, U18 cutover or whole-appliance reinstall occurred.

The existing Alice/Bob identities created/joined private repository
`u08-alice-8417/u08-completion-c96c108`, provider repository ID `4`, environment
`p7b41edaf83f10a6fd7e579bf`, currently `10.89.0.5`. It was created directly from the
exact new image, with no manual initializer/unit/rootfs integration patch. Native
inspection proved private project-owned user/network/PID namespaces, default
seccomp, nonprivileged parent, mapped root and absent human host accounts. The
fixed creation profile adds SYS_PTRACE only in that project user namespace; no
caller-selected capability, host namespace/socket, process-debugging supervisor
or retrofit of older containers was introduced.

**The exec blocker is resolved in this fresh fixture.** PostgreSQL PID 1 actually
runs as UID/GID 999. `tests/installed/workload-exec.py` passed default-root exec,
explicit PostgreSQL-user exec, real SQL and PTY exec; Bob still receives native
socket permission denial. Ordinary Compose bridge creation/build/start succeeded
without a network override. Its first immediate HTTP check raced the newly
started server (connection refused). The test now bounds HTTP/PG readiness reads
and offers explicit `check` mode, so completing reads never replays up/build.
Those reads and Compose SQL exec passed against the same retained workloads.
A missing personal `.config` parent was corrected before secret creation; no
credential or partially created resource was reset.

Direct SSH/PTY/SCP/SFTP, cross-project authentication denial, sudo boundaries,
personal encrypted Git keys/acting-user registration/clone/commit/push/readback,
shared Node installation/files and ordinary-member tool-write denial passed.
The real client, Alice and Bob also passed live bind-mounted HTTP and committed
PostgreSQL read/write/readback. A new loopback Git transport uses client port
24424; prior routes/transports were left intact. Git passphrase inputs remain
restricted and private; keys stay in the two personal project homes.

Before/after comparison proved **all three older roots' declared state unchanged**,
including accounts, keys, tools, Git, workloads and database rows. Original Soda
records matched; exactly one project and its two explicit memberships were added.
The VM boot ID is unchanged. No project lifecycle or VM reboot was repeated for
this capability follow-up; earlier `f233a4a` lifecycle evidence remains scoped to
its recorded bytes. No extra fixture, provider CI, publication or cleanup ran.

Evidence: `.artifacts/logs/u08-ptrace-*` (especially `build-c96c108`,
`check-c96c108`, `fresh-image-boundary`, `different-uid-exec`,
`workloads-ready-check`, `client-member-workloads`, `git-exercise`, `shared-tools`
and `preserved-comparison`). Private inputs/current bindings/new-root snapshot:
`.artifacts/test-vm/u08-completion-c96c108/`. VM payload/backups/rehearsal:
`/var/lib/soda/u08-completion-c96c108/`. Readiness regression tests and documentation
follow the built revision; they do not change its production bytes. All 24 current
Python build tests passed, including start-once, bounded failed readiness,
check-without-mutation and invalid/empty-mode cases; shell syntax and Git whitespace
checks passed.

**U08 is still not marked accepted:** reconcile the separately recorded
host/operator regression limits and exact-revision lifecycle coverage against the
core acceptance criteria. Full provider/runner and fresh-appliance support exits
remain U20/P work, not a new independent U08 gate. The
specific different-UID exec and final-image fresh-creation gaps are now closed
for native x86_64, not inferred from TCP success. Unselected aarch64/U20/media and
other U milestones are unchanged.

## Core/support merge — source only

Merged native support remediation `41fb6d3`/`9acbda5` with core follow-ups through
`f233a4a`, preserving both handoffs and all core runtime/lifecycle corrections.
The incoming `952f3b3` and `935dbdf` aggregate build/check passes below supersede
older blanket aggregate-pending statements **for those candidates only**. Neither
includes the support remediation, so the merged candidate still requires its own
build/check. The socket-ordering correction and remaining U08/runtime gaps retain
their recorded limits. This merge ran no builds, tests or native operations and
changed no installed state.

## Native support audit remediation — source only

Implemented the next active support-source pass from clean baseline `15e49b1`.
No current native milestone exit is claimed. The [audit follow-up](native-porting-audit.md#source-remediation-follow-up) maps the changes and remaining work; the original audit remains a historical snapshot.

- **P02/P03:** structured JSON is sanitized before encoding; observations publish under their final name only after write/close/leak checks, with the pending file retained. Evidence scanning/hashing and report reads use open-directory capabilities; reports hash the exact decoded metadata and expose cleanup/exit/invocation/artifact context. Transfers retain the actual manifest digest and detect changed streamed bytes. Failed VM launches retain cleanup facts separately. Linux child groups use non-reaping `waitid(WNOWAIT)` before group cleanup, preventing leader exit/PID reuse from dropping or misdirecting cleanup. Non-Linux owned execution now fails closed; portable metadata/report/direct-key-probe operations remain available. This is not a sandbox for processes that deliberately leave their group or change privileges.
- **U02/P04 shared contract:** moved the existing core frontend validation unchanged into `internal/frontend`; web startup retains its entrypoint and P04 now consumes the same validator. No second frontend build or manifest policy was added. Record the real dashboard package/lock inputs and cross-check public build revision/platform/image IDs. `/etc` export admits only the current public staging paths, excluding actual OAuth/admin/grant credentials and unknown files. Directory-relative bundle copies verify copied hashes; OCI checks now include schema/media types, local descriptors and rootfs/diff-ID structure. Compressed layer contents still require native import proof.
- **P05/P06/P11:** preflight SSH availability, record qemu-img version, bound signature/decompression phases and tighten public download URLs. Build parents are checked before descendant creation. First-install preflight checks existing host/container networks, destination ancestors and one booted deployment before writes; the existing installer/setup/activation ownership is unchanged. Installed checks assert configured credential/TLS modes and native bindings, accounting for rootful DNAT publication rather than assuming every published port appears in `ss`. Cockpit browser observations require SELinux enforcing. Installed-byte verification has explicit filesystem/command test inputs and preserves CoreOS's writable-prefix mapping.

Authored focused tests for escaped/structured redaction, finalization failures,
renamed evidence roots, Linux leader-first/TERM-resistant group cleanup, CLI
cancellation, changed transfer inputs, credential contamination, missing/stale
React payloads, metadata/OCI failures, bounded trusted-CA HTTP downloads, installed
identity/mode/image failures and explicit remote phase ordering/refusal. These
new tests have **not run**. Formatting, shell syntax, Python AST inspection and
whitespace review are source checks only, not compilation or behavioral proof.

The tightened verifier requires a freshly built bundle with the new public
inputs. Preserve old stages/bundles and use their matching historical verifier;
never edit/reseal retained metadata to satisfy new checks. Current installed
component revisions and the existing VM/projects/tunnels/data are untouched.
Remaining work includes executing/fixing the new and aggregate suites on an
explicitly authorized exact native candidate; additional external-tool/VM and
installer failure fixtures; selecting actual trusted native inputs; fresh
fixture/install/operator and independent aarch64 evidence. P07/P08 still belong
to U08/U20, and P09/P10 media remain unselected. No build, dependency resolution,
test execution, native target/provider action, publication or cleanup occurred
in this pass.

## U08 completion execution — candidate and remaining runtime correction

**Preceding installed checkpoint: `f233a4a`.** Its
entire native x86_64 build/seal and aggregate `check-native.sh` passed: full Go,
60 Cockpit tests, 21 dashboard tests, 19 build tests and 9 staging tests. Earlier
`952f3b3` and `935dbdf` candidates were also built/checked and rolled out during
diagnosis. Before each matching rollout, a private populated-v3 backup and
isolated startup rehearsal preserved users, keys, projects, memberships, sessions
and encrypted grants. No whole-core reinstall or U18 cutover occurred; `/app/`
remains the preview. This is populated v3 preservation, not a populated
version-changing migration or a live rollback.

The approved private `u08-completion-952f3b3` repository/environment now exists:
`ped30b9d6932974b14feb2278`, initially at `10.89.0.4`, with explicit Alice/Bob joins.
Addresses changed during lifecycle checks; current verified values are in private
`target.json` and per-user connection observations, not assumed from old ports/IPs.
Its exact image, project-owned user/network namespaces, NET_ADMIN, default seccomp
and lack of privileged-parent/host accounts were verified. Direct SSH/PTY/SCP/SFTP,
personal native Git and shared Node/files passed. New Git passphrases are retained
only in restricted client files for post-reboot agent unlock. Browser fixture and
transport test issues were corrected without recreating users/projects: an access
test variable shadowed its scenario, and SSH control socket paths needed shortening.

Default bridge startup exposed read-only per-interface network sysctls. A narrow
project-local proc bind makes only `/proc/sys/net` writable, leaving the rest of
`/proc/sys` read-only. The same failed workload containers/volumes were retained
and started after the correction; an intermediate stale DNS/interface failure is
also retained. HTTP source edits and committed PostgreSQL operations now passed
from the real client and both users using the ordinary bridge, not host mode.
The initializer performs that narrow setup before publishing readiness. The
initializer and engine service/socket corrections are now packaged in `f233a4a`
and applied to the fresh fixture's retained writable root, with prior source
inside `/var/lib/u08-proc-net-fix/`. Its outer container/base-image identity remains
`952f3b3`; final-image fresh creation is not proven by this in-place source update.

A separate native exec failure was traced through restricted process diagnostics
to OCI `openat /proc/<pid>/ns/mnt: Permission denied`. The API service's wheel GID
prevented same-UID access; keeping its native root GID while retaining the socket's
root:wheel 0660 permissions fixes same-UID exec. Different-UID PostgreSQL exec
still fails: the engine is root, PostgreSQL uses UID/GID 999 in host-mode userns
(the project's namespace), and the project lacks SYS_PTRACE. This is consistent
with Linux process-namespace access checks; a fixed namespaced-capability
correction still needs native validation. No SYS_PTRACE, privileged parent,
host-engine mount or Podman-state edit has been applied. This remains a runtime
coverage gap, not a passing Compose-exec case. Review/approve one further fresh
fixture for the corrected profile; the approved one additional project was used. State snapshots use the real native PostgreSQL TCP client and
private pgpass input, not a fabricated or empty exec result. A complete preflight
snapshot of all three projects and Soda associations succeeded.

**Persistence passed after correction.** The first approved project stop/start
preserved all declared stable data except the subsequent deliberate socket-unit
correction: comparison identified exactly that file's SHA256, no other changes.
The corrected repeat started init/socket/SSH automatically, retained only the
network sysctl subtree writable, and matched the entire three-project snapshot.
Both users unlocked their own retained encrypted Git keys in new project-local
agents and matched actual native remote refs.

Cold startup exposed an ordering cycle: socket -> sockets.target -> basic.target
-> init -> socket. The enabled init/socket jobs were not started; readiness was
absent. The correction removes the socket's init dependency; the activated service
still requires init, and init still precedes SSH. The native repeat verified this
startup, not just a static unit assertion.

**Only `soda-test` then rebooted.** Pinned SSH returned with a different boot ID;
all three stable snapshots matched before/after, including accounts, groups,
permissions, public host keys, roots, dirty/untracked Git work/refs, shared tools,
workload/volume identities and committed PostgreSQL rows. Existing workloads
required explicit native starts; no recreation, reseeding or automatic workload
resurrection is claimed. The exact private route/firewall/browser/Cockpit/both Git
transports were restored. Rediscovery checked prior public host-key pins before
updating only private connection/pgpass inputs. No LAN/Tailnet exposure changed.
Post-reboot direct SSH/PTY/SCP/SFTP, sudo/engine denials, personal Git unlock/native
ref readback, shared Node/files and real bridge HTTP/PostgreSQL passed again.
Native Compose `up --no-recreate --no-build` also passed with unchanged workloads.
Operator and both developer OAuth/read/navigation/logout passed; real connection
APIs returned new addresses, matching public keys and appropriate access denials.

Read-only regression attempts also retained preexisting fixture limits: `host.sh`
stops at RPM name assumptions (`nodejs`, `zlib` absent under those names), and
`operator.sh` cannot list runners without its integration configuration. Neither
is a passing full host/operator regression. Native Cockpit PAM root admission /
existing non-root denial and configured-origin TLS did pass after reboot. A real
root Cockpit login/target/navigation/logout also passed without opening the
Tailnet page and invoking advertisement refresh. Full runner/provider jobs and
complete host/operator integration are not implied.

**Evidence and limits:** logs are `.artifacts/logs/u08-completion-*`; the final
`build-f233a4a` and `check-f233a4a` logs have recorded exit 0. Candidate payloads,
backups and rehearsals are `/var/lib/soda/u08-completion-{952f3b3,935dbdf,f233a4a}/`
on the VM. Private fixture inputs, before/after JSON, and restored transports are
`.artifacts/test-vm/u08-completion-952f3b3/`; earlier roots/evidence remain intact.
Key result logs include `corrected-project-persistence`, `vm-persistence`,
`vm-developer-access`, `vm-workload-client-members`, `vm-git-tools`,
`vm-*-browser-connections`, `vm-dashboard-browser-operator` and `vm-cockpit-browser`.
A failed source-test driver accidentally overwrote the earlier `935dbdf` build
log with a clean-tree refusal; that refusal is retained as `935dbdf-dirty-restart`.
Its aggregate log/sealed bytes remain, and the final `f233a4a` full logs are intact;
no missing PASS record was reconstructed. Other diagnosis/argument/selector and
transport failures remain recorded rather than erased.

The latest post-reboot shared-tools check now takes the rediscovered isolation
address explicitly; that harness-only correction follows the built `f233a4a`
revision. Its focused source contract and all 20 current Python build tests
passed; shell/Python syntax and `git diff --check` passed. Final post-reboot byte
checks matched `f233a4a` dashboard/helper/default-image hashes and preserved the
unprivileged dashboard, restricted helper socket/credential modes and enforcing
host SELinux (`final-bytes-and-boundary.log`). **U08 remains unaccepted** pending different-UID exec/final-profile proof
and host/operator regression reconciliation. Earlier fixture Git passphrases
were not retained: those old agents died at reboot; their encrypted keys remain,
but only the new fixture proves durable unlock. No extra fixture, infrastructure
reboot, data reset, provider job, CI publication or media delivery was performed.

## U08 completion execution authorized — preparation

The user requested execution of the entire recorded U08 completion plan, including
the exact `soda-test` rollout, one additional fixture, its stop/start and the VM
reboot. This does not authorize unrelated targets or destructive cleanup.
Prepared fresh-fixture browser coverage, parameterized existing Git/access/shared-
tool/workload cases, retained private passphrase inputs for new Git agents, and
bounded all-project lifecycle snapshots/comparisons. Exact private transport
restoration is authored under `.artifacts/tools/u08-completion-transports.py`.
Preparation is not a new native PASS; the following build/rollout/runtime stages
must record their actual outcomes before acceptance.

## U08 completion plan recorded

Added the [U08 completion checklist](dashboard-implementation-plan.md#u08-completion-execution-plan--baseline-0d4c4eb)
against baseline `0d4c4eb`: prepare missing proof, build/check one candidate,
backed-up matching preview/helper/image rollout, one additional approved project
for default bridge verification, then separately authorized project stop/start
and `soda-test` reboot persistence. It explicitly preserves current fixtures,
handles ephemeral Git agents and transport restoration, and distinguishes stable
state comparisons from volatile runtime observations. U08 closes only on actual
criterion-by-criterion evidence; no U20/ISO requirement was added. This update is
planning/documentation only; no build, deployment, fixture creation or lifecycle
operation ran and no new approval was inferred.

## Native porting remaining-work audit

Source/documentation audit at `58ddc0d0707b9b2c72f97363e8daa1b53853f341`, with a clean starting tree: [full findings, P01–P13 status, missing coverage and next steps](native-porting-audit.md). The earlier unconditional active-source-complete description was too strong. Concrete remaining work includes the public bundle accepting actual Soda credential filenames, incomplete descendant cleanup, structured redaction/finalization defects, path confinement, U02 payload/input alignment and evidence binding. Existing x86_64 `8417a90` build/seal and subsequent component/Go-suite evidence is credited; neither the aggregate check nor fresh support fixture/install/operator/aarch64 exits is newly passed. Core U08/U20 ownership and unselected P09/P10 media are unchanged.

Only audit documentation/current native-guide summaries changed. No implementation fixes, tests, builds, artifact re-verification, private-input inspection, SSH/VM/provider actions or cleanup were performed. Existing private evidence was not reread; historical execution claims are taken from the recorded handoff, with retained-worktree mapping noted. No commit or push was performed for the audit.

## Personal Git, shared tools and nested workload evidence

Continued the user's selected U08 work on the retained Alice/Bob projects from
infra. No project container was stopped/replaced, no VM reboot occurred and no
Forgejo listener/advertisement or appliance network configuration was changed.

**Personal Git passed.** `tests/installed/personal-git.py` generated separate
passphrase-encrypted outbound Git keys inside each user's home in Alice's project
and loaded project-local agents. Only public parts reached the acting-user
Forgejo Git-key UI/API through `personal-git.mjs`. No development key was reused,
agent forwarded, provider token borrowed or private SSH key exported. Both users
cloned their actual native private-repository URL, committed/pushed independent
branches and matched remote ref readback; Bob separately fetched/read Alice's
commit. This is native Git authority, separate from Soda membership. The generated
passphrases were temporary and removed after agent loading: these fixture Git
identities currently depend on the live agents, **not durable post-reboot Git
login proof**. Encrypted keys, agent PID/socket records and ordinary checkouts are
retained in each user's home; do not assume an agent will survive reboot.

Forgejo actually advertises `ssh://git@127.0.0.1:2222/...` and listens only on VM
loopback. The exact-target private `.artifacts/tools/u08-git-transport.py` carries
that endpoint through infra loopback 24422 to project loopback 2222 using normal
SSH forwards. The project-wide transport is shared; each Git operation still uses
its own native identity and an independently verified Forgejo public host key.
The two transport control sockets and public results are retained under
`.artifacts/test-vm/u08-8417a90/personal-git/`. These forwards are test-origin
transport, not a production Git gateway or a changed advertised endpoint.

**Shared tools/files passed.** Alice used project sudo with umask 022 to install
Node 24.20.0 through native global mise. `shared-tools.sh` verified both users run
the same root-owned `/opt/mise` executable/device/inode through noninteractive
SSH. Bob cannot write the binary or replace its parent/config/shim paths. Both
users changed/read the same group-shared file/inode, absent in Bob's separate
project. Probe files remain under `/srv/project/shared/u08-shared-*`.

**Two native runtime defects corrected:** the project service's empty
`CONTAINER_HOST` enabled Podman's remote mode; then its directly created socket
was mode 0600, denying wheel clients. Current source uses `UnsetEnvironment`,
native systemd socket activation, root:wheel socket 0660 and directory 0750.
The units/init changes were applied only to Alice's existing project with prior
files retained at `/var/lib/u08-podman-env-fix/` **inside that project**. Owner
engine access succeeded and ordinary-member access was denied. Only the failed
project-local API service was stopped/reconfigured; the outer project stayed up.
Bob's second project still has its earlier image/service configuration.

**Default nested networking is still blocked in the existing fixture.** Compose
pulled postgres:17, built the Python HTTP image and created its resources, then
both starts failed with a netavark Netlink permission error. A native NS_GET_USERNS
inspection proved the project network namespace belongs to its own user namespace,
not the host's; its capability set lacks NET_ADMIN. Current helper source adds
NET_ADMIN to the fixed SYS_ADMIN/MKNOD set for future project creation. That change
is source-tested, **not built/deployed or proved in a fresh project**. The installed
host's Podman 5.8.4 has no capability-update option. Do not recreate current
projects to apply it or call the bridge fixed on the installed target.

**Real workload diagnostic passed using project-network mode.** Through the same
nested engine, `u08-projectnet-files` and `u08-projectnet-database` use native
`--network host`, meaning the project's namespace, never the appliance network.
The HTTP image built by Compose is used with Alice's checkout bind-mounted;
PostgreSQL uses a new native `u08_projectnet_database` volume. The original failed
Compose containers/network/volume remain intact. `workload-access.py` verified a
live dirty source edit through HTTP from infra/Alice/Bob; the real client inserted
and committed PostgreSQL data, both users read it through TCP, Bob updated it and
a fresh client connection read the committed value. This does not prove the
unmodified Compose bridge path, complete isolation or restart/reboot persistence.

The fixture now uses an external native `soda-example-db` secret with
POSTGRES_PASSWORD_FILE instead of a password environment value that Compose
would put into Podman argv. The server secret and client pgpass inputs are private,
not tracked or logged. PostgreSQL's native client was installed inside Alice's
project; infra uses a private venv with psycopg[binary] 3.2.9, not host package
installation. Git/DB credentials remain outside checkouts/shared directories.
Alice's checkout has a committed workload fixture plus a deliberately dirty public
HTML edit. SQL rows and private evidence/pgpass files are retained under
`.artifacts/test-vm/u08-8417a90/u08-workload-access-*` and the users' private homes.

**Checks:** full Go suite passed with Go 1.26.7/CGO disabled; 15 Python build tests
passed, including new service/socket/secret contracts. New Go creation-argument
coverage preserves fixed namespace/privilege boundaries. Native Git/shared-tool/
HTTP/PostgreSQL tests passed as described; shell/JS syntax checks passed. No full
image rebuild, new default-bridge fixture, dashboard cutover, restart/reboot or
provider CI job was executed. Installed dashboard/helper/image baselines remain
35df189/8417a90/8417a90, with the explicit project-local unit patch above.

Logs under `.artifacts/logs/`: `u08-personal-git-*`, `u08-*-git-registration.log`,
`u08-git-transport.log`, `u08-shared-node-install.log`, `u08-shared-tools-files.log`,
`u08-podman-*-fix.log`, `u08-project-engine-access.log`, `u08-workload-first-run.log`,
`u08-nested-network-namespace.log`, `u08-project-network-workload.log`,
`u08-workload-client-and-members.log`, `u08-postgres*-*.log`,
`u08-host-capability-tests.log`, `u08-project-runtime-source-tests.log` and
`u08-git-workload-all-go.log`. Failed attempts are retained, not rewritten as passes.

Next: build the corrected helper/image and obtain an approved fresh fixture to
validate native default bridge networking without replacing current projects.
Complete remaining runtime/isolation checks and separately authorized project
stop/start and VM reboot persistence. U08/U20 remain incomplete; source work on
U15/U16 and coverage closure remains independent.

## Temporary worktree cleanup

At the user's request, removed all six clean detached U08 worktrees after
checking for uncommitted/untracked source and active users, then pruned Git's
worktree metadata. Only the main checkout remains. Before removal, moved each
worktree's `.artifacts/` and compiled dashboard/Cockpit `dist/` directories
unchanged to `.artifacts/retained-worktree-builds/<former-worktree-name>/`.
Each retained directory has `retention.json` recording its original path, source
commit and retained paths. Historical logs/manifests still refer to the original
checkout locations; use this mapping rather than rewriting evidence. Dependency
directories and disposable Python caches were removed with the worktrees.
Main-checkout artifacts/logs, VM/backing image, live tunnel and project fixtures
were not changed. No builds or tests ran during cleanup.

## Approved private routing and direct developer access

The user explicitly approved the private SSH tunnel after the `6a1f129` plan
update. Infra is now the routed developer client for the existing `soda-test`
projects. The exact-target private recipe is
`.artifacts/tools/u08-project-tunnel.py`; evidence is
`.artifacts/logs/u08-project-tunnel.log`. It creates `tun8417` at both ends,
point-to-point `169.254.84.1` (infra) / `169.254.84.2` (guest), MTU 1400, and only
the project route `10.89.0.0/24` on infra. Preflight checked target names,
interface/route/table/config collisions and existing VM forwarding (already 1).
No global forwarding sysctl, LAN route, Tailnet enrollment or existing firewall
chain was changed.

Dedicated `inet soda_u08_tunnel` tables restrict tunnel traffic: infra cannot
forward LAN/container traffic through it; VM ingress is limited to forwarding
from the client address to `soda0`/the project subnet, with established return
traffic only. Tunnel access to VM-local services and project-initiated access to
infra are dropped. SSH `PermitTunnel point-to-point` is scoped to root from the
QEMU management address `10.0.2.2`, port 22, in the new
`/etc/ssh/sshd_config.d/00-soda-u08-tunnel.conf`. Configuration validation preceded
SSH reload; no service restart, VM reboot or project replacement occurred.

`tests/installed/developer-access.py` passed for Alice in her project and Bob in
both projects: own-key direct-IP SSH, exact username/home/non-root identity,
interactive PTY, bidirectional SCP and SFTP with byte comparison, mapped
non-host-root UID namespace, administrator sudo success and ordinary-member sudo
denial. Alice's attempt to authenticate to Bob's unjoined project failed with
actual public-key authentication denial, not a transport or host-key error.
The final run disables personal SSH configuration and agent/port forwarding;
strict independently verified host keys remain required. Private keys were used
only by the client and never transferred. Logs:
`u08-first-direct-project-ssh.log`, `u08-direct-developer-access.log` and
`u08-direct-developer-access-isolated-client.log` in `.artifacts/logs/`.

New uniquely named probe directories/files are retained in the three project
homes and under `.artifacts/test-vm/u08-8417a90/u08-access-*`. The tunnel control
socket and rule inputs are in that fixture root's `tunnel/` directory. The exact
teardown recipe `.artifacts/tools/u08-project-tunnel-stop.py` is authored but
**not executed**; it removes only this route/tunnel/tables/SSH drop-in and reloads
SSH, preserving project data and evidence. The route is not persistent across
host reboots and does not provide access from the user's laptop automatically.

This supersedes earlier routing-approval/direct-SSH-pending statements below.
U08 remains incomplete: personal Git, genuinely shared tools/files, nested
application/database work, remaining runtime isolation and stop/start/reboot
persistence are still unverified. Installed application/helper/image revisions
remain unchanged; no build, provider job or lifecycle test ran in this follow-up.

## Current local verification after execution approval

The user's “go until finished”, following the listed execution gates, authorized
local build/test work. On the x86_64 workstation (Go 1.27.0, Node 24.20.0,
pnpm 11.25.0), executed `go test ./...`, dashboard `vp run check`, `vp test --run`
and `vp build`: all passed. The dashboard suite ran 16 tests across 10 files.
The first UI run exposed two unavailable Jest-style assertions; these now inspect
native DOM properties. Restored the already-selected Markdown dependencies missing
from the manifest and resolved the real lockfile with lifecycle scripts disabled.
Logs are in ignored `.artifacts/logs/{all-go-tests,dashboard-typecheck,
dashboard-ui-tests,dashboard-build,dashboard-install}.log`.

This supersedes earlier **local** unbuilt/unexecuted statements below for the
checked source. It is not a full native artifact build, installed verification,
Go 1.26 baseline run, aarch64 proof or U milestone completion. No VM, service,
provider resource, project state or networking was changed. Deployment/migration,
developer routing and installed workload/persistence acceptance remain pending.

## Native first developer fixtures and complete core build

Full matching-native x86_64 build/staging/sealing succeeded at `8417a90`, after
correcting a payload filter that mistook npm's `installed-deep.js` for appliance
installation state. The signed pinned gh RPM installed successfully in Rocky.
Cockpit type checks plus 60 tests, dashboard type checks plus 21 tests, 12 build
fixture tests and 9 staging tests passed against that checkout/stage. The aggregate
`check-native.sh` stopped on two Go test-fixture directories that assumed a private
umask; `eca7673` makes their private parents explicit, and the full Go suite passed
with Go 1.26.7, CGO disabled and umask 022. Do not call the earlier aggregate
entrypoint a pass; its remaining components were invoked separately.

The `8417a90` helper and project image were transferred with verified hashes and
installed on `soda-test`, retaining prior artifacts in
`/var/lib/soda/u08-core-8417a90/`. Host configuration stayed unchanged; the image
is used for new projects, never to replace existing writable roots. Stopping the
helper socket also stopped the dashboard through its native Requires dependency;
the same preview dashboard was explicitly restarted. No Forgejo, proxy, Cockpit,
Tailnet or VM restart occurred.

Core-owned installed browser tests created `u08-alice-8417` and `u08-bob-8417`
through Soda's acting-admin People API. Native first-login password changes,
independent OAuth sessions, development public-key registration, two private
repositories, two persistent environments and explicit owner joins succeeded.
Bob joined Alice's environment as a non-administrator. Alice separately granted
native repository write collaboration; that did not grant Linux membership.
Before collaboration Bob could not read the private repository; afterward he
still could not create its environment. Bob's explicitly admin-scoped OAuth grant
was denied by native site-admin authorization. Both users remain absent from the
appliance's host passwd database.

The run was resumed explicitly after two PatternFly alert selectors timed out
after successful writes. Existing fixture identities/keys were checked rather
than recreated; private inputs, bindings and failed-attempt logs were preserved.
The new tests do not automatically retry uncertain provisioning or clean up
resources. Their resumed result is not a fresh-install acceptance run.

Both users' authenticated connection responses and join-required denial were
checked. Advertised public host keys matched an independent pinned operator SSH
read of each project's public host-key file; private developer keys remain only
on infra. Known-host entries and connection metadata are retained privately under
`.artifacts/test-vm/u08-8417a90/`. Project-local home/public-key/Tea/gh checks passed
through operator `podman exec` as each local administrator, explicitly **not** as
client SSH proof. No developer private key was copied into the VM or projects.

Logs: `u08-core-native-build-payload-fix.log`, `u08-core-native-check-8417a90.log`,
`u08-native-go-umask022.log`, `u08-*-check-8417a90.log`,
`u08-*-tests-8417a90.log`, `u08-core-runtime-rollout.log`,
`u08-developer-browser*.log`, `u08-{alice,bob}-connections.log`, and
`u08-project-local-observations.log`. The application remains `35df189` at `/app/`;
helper/project image are `8417a90`. U08 is still incomplete: direct client
SSH/SCP/SFTP, personal project Git, shared installs, nested workload/data and
persistence are not proven. A narrow Layer-3 SSH tunnel/route was proposed for
infra→project access and awaits approval; no infra routing/Tailnet change was made.
Independent arm64, U15 wiki/packages, U16 and U17–U20 remain unfinished.

**Preservation:** real developer identities, keys, repositories, memberships and
project writable state now exist. The old pre-migration backup cannot be restored
without an explicit decision preserving those later writes. Do not reset fixtures,
replace project containers or roll back the Soda DB as a repair shortcut.

## Native React preview migration and operator browser proof

Executed on infra and its existing x86_64 `soda-test` guest, with strict pinned
SSH and browser certificate verification. Candidate application revision
`35df189` was built with Go 1.26.7, Node 24.20.0 and pnpm 11.25.0 in a fresh
checkout. Candidate archive SHA-256:
`5c6e660829350b48d139bcc73802e159f43c7420a9dfed6630586643b1a5f276`.
The full Go suite also passed with the pinned Go compiler.

A private backup under `/var/lib/soda/u08-preview-35df189/` preserves the old
SQLite database (backup API plus integrity check), configuration, credentials,
unit metadata and immutable prior image archive. An isolated network-disabled
candidate migrated a copy from schema 1 to 3 and served `/app/`; one user and two
sessions were preserved, with zero existing keys/projects/memberships. Missing
key, wrong key and missing frontend cases refused startup without modifying the
copied DB. The prior image started against its matching schema-1 rollback copy.
This is not evidence of populated project persistence or a live rollback.

A second quiesced snapshot preceded the live dashboard-only rollout. The existing
OAuth application/origins and Soda identity/project records were preserved; a new
restricted grant key was provisioned. Only the dashboard was restarted; Forgejo
and proxy container IDs stayed unchanged. The installed candidate now serves the
React preview at `/app/`; default HTMX remains, and U18 cutover has not occurred.

`tests/installed/dashboard-react.mjs` exercised native login/consent/callback,
secure session attributes, acting-user account/repository/notification/admin
reads, React navigation/direct-link reload and CSRF-protected logout. The initial
old-consent attempt correctly failed with `consent_required`, without escalation.
The operator's uniquely named Soda grant was explicitly revoked through its own
native Applications UI, then reauthorized including administrator consent; the
journey passed. Browser requests use the browser's trusted CA path rather than
Playwright's separately untrusted Node request context. No TLS bypass was added.
Logs: `u08-preview-*`, `u08-dashboard-*`, `u08-pinned-go-tests.log` and
`u08-react-browser*.log` under `.artifacts/logs/`. The initial failed attempts are
retained. Exact-target private operator recipes are retained in `.artifacts/tools/`.

Full-core native build attempts then exposed Tea Make GOFLAGS export and ANSI
version-matching defects, fixed in `9768dd1`/`5ab427f` with passing focused fixture
tests. The next attempt reached an unavailable pinned gh RPM in the rolling
vendor repository; `e4c1173` selects its signed retained release RPM without
changing the version or disabling signature checking. Full image/staging checks,
current helper/project-image installation, two-user/project/workload proof,
client routing, independent arm64 and the remaining core work are not passed.

## U15 release/asset source and local verification

Connected native release list/detail/create/changed-field edit and bounded asset
upload/download. Creation requires an explicit draft state (the UI defaults to
draft); duplicate tags and upstream permission errors are not retried. The pinned
release PATCH ignores empty title/body replacements, so Soda rejects those rather
than claiming a clear. Asset downloads first resolve repository/release-scoped
metadata and then use only the fixed native attachment UUID path with the acting
grant. External URLs and redirects are never credential-forwarding targets.
Uploads are bounded to 32 KiB; inert downloads to 8 MiB before browser headers.
No Soda artifact store or release records were added.

Full local Go suite, dashboard type check, existing 21 UI tests and production
build passed; `.artifacts/logs/u15-releases-*.log`. Focused Go tests cover duplicate
creation, changed-field edits, ignored clears, scoped download denial, redirects,
lossless attachment IDs and invalid uploads. Release-specific DOM and installed
provider proof remain pending, as do wiki/packages and the rest of U15–U20.
No release, tag or asset was created on a real provider.

## U14 source and local verification; U08 access observation

Connected native Actions runs/detail, repository tasks, recursive workflow-file
selection and explicit dispatch; repository/organization secret and variable
configuration is native, with write-only secret values and no Soda persistence.
Run/event payloads and runner credentials are excluded from DTOs. Run polling
retains one response and stops on navigation, logout, terminal status or failure.
Run/task pagination reads the native public API cap because those endpoints
return body totals without Link headers. Workflow discovery retains the pinned
first-directory rule, native recursive tree pagination and a 30-second/10,000-entry
bound. Provider denial is not retried with another authority.

Pinned web run routes exist for logs/artifacts/cancel/rerun, but OAuth2.Verify
explicitly excludes those web paths. No runner protocol, password, or borrowed
cookie adapter was introduced. Native-login fallback remains an explicit U17
disposition, not completed parity or user acceptance of the gap.

Executed local full Go suite, dashboard type checking, 21 tests in 12 UI files
and production build successfully; logs `.artifacts/logs/u14-*.log`. No real run,
secret, variable, repository or VM mutation was executed.

Read-only U08 builder observation attempts failed before execution: the recorded
`vince@192.168.2.253` address rejected public-key authentication, and
`vince@linux-infra.dimensionlab.net` had no trusted ED25519 host-key entry here.
Strict host-key checking was retained. Logs `u08-builder-observation.log` and
`u08-builder-hostname-observation.log` contain only those failures. Those were
mistaken SSH-to-self attempts, not a builder access blocker: subsequent local
`hostname`/address inspection confirmed this workspace is already on
`linux-infra.dimensionlab.net` at `192.168.2.253`. Local
`scripts/test-vm.sh status` reported the existing guest running, and the pinned
`scripts/test-vm.sh ssh 'hostname; uname -m'` succeeded, reporting `soda-test`
and `x86_64`. No SSH repair is required. These read-only observations did not
change VM state or establish installed product acceptance.

## U13 source and local verification

Connected My work, native permission-filtered repository/issue/PR search,
notification read/unread/pin updates and native user profiles/activity. Personal
work filters use the acting grant, not a caller-selected user ID. Profile-owned
repository search resolves the user through Forgejo and never widens a missing
owner into an unfiltered search. Notification navigation extracts only a trusted
repository subject number, never provider URL hosts/query credentials. Native
notification permission and 205 acknowledgement semantics were inspected, as were
native profile/activity visibility rules. Added verified `write:notification`
consent; no work/activity/notification inventory is stored in Soda.

Executed `go test ./...`, dashboard type checking, 18 DOM/unit tests in 11 files,
and production build: all passed locally. Logs: `.artifacts/logs/u13-*.log`.
Focused cases cover personal-actor binding, large IDs, notification denial/links,
missing-owner scope, superseded search and independently failing overview panels.
Installed native-user verification, advanced activity/coverage decisions and the
remaining U14–U20 work are still pending. No provider or VM mutation was executed.

## Core/support rebase integration — source only

Resolved the build/context/installation-guide overlap in favor of U02's real React payload and missing-lockfile guard. The full native build skips the dashboard in its generic command loop and calls the core `build-dashboard.sh` with `--payload-only`; support then builds/exports the dashboard image exactly once with native identity metadata. The standalone dashboard entrypoint retains its existing image build and does not depend on support tooling. Container context rules admit the generated dashboard assets while retaining dashboard dependency/dist exclusions and denying other artifact trees by default. Staging and the core Containerfile continue to consume that same payload. This is source integration, not executed build or runtime evidence; builds/tests and dependency resolution remain held.

## Native support source implementation — not executed

Implemented the active outside-support source against the coordinated native plan. The [support guide](native-support.md) documents inputs, effects, recipes, shared-file ownership and retention; [notices](native-support-notices.md) identify selected predecessor reuse. Core U08/U20 still own product journeys/readiness. P07/P08 are not new work queues; P09/P10 media remains unselected and unimplemented.

- **P01/P02:** separate `tools/soda-{artifacts,acceptance}`, restricted/exclusive evidence, bounded streaming redaction, literal pinned-SSH/stdin transport, separate native/evidence outcomes, and one exact-source remote prepare/build/check/bundle phase at a time. No release account, publisher, scenario registry or copied product flow.
- **P03:** fresh matching-native KVM ownership, QMP negotiation/deadlines, validated host trust/hostname, isolated overlay/NVRAM, bounded shutdown/restart and retained private work. The existing `soda-test` helper, VM and backing files are untouched.
- **P04:** explicit native Go/local-engine guards, checkout/output locking, ELF/OCI/blob/source/base checks, resolved service-image IDs, public dependency/package/CLI metadata, allowlisted bundles/notices/checksums and verifier exclusion from installed payloads. Existing build/stage paths and core service references remain authoritative; React packaging is not implemented here.
- **P05:** independently pinned CoreOS architecture inputs, explicit trusted signer/keyring verification, immutable fresh cache output, public/bootstrap separation, restricted strict Butane conversion, verified transfer and first-install preflight/partial-state refusal. No app setup/OAuth/migration implementation is duplicated.
- **P06/P11:** read-only immutable installed-byte/service-image binding and authored host/order/PAM-account/TLS and retained Cockpit/Tailnet/Runners/advertisement observations, plus an opt-in trusted provider-job fixture outside CI discovery. Provider mutations/interactive reviews still require their exact grants and actual observations.
- **P12/P13:** ordinary support handoff reporting with source/platform/file-identity checks, separate missing/failure/cleanup states and core-owned result references. Reporting source exists; **no x86_64 or aarch64 native exit has been completed by this work**.

Authored Go and Python fixture/regression checks, preserving existing core/packaging/Cockpit checks. Per the execution boundary, **no build, test suite, dependency installation, VM/SSH/provider operation, installation, network change, commit/push or publication was run for this implementation**. Work performed is source/research/editing and formatting/static review only. Public CoreOS stream metadata was inspected; no CoreOS image/signature was downloaded or booted. Current tool/installer/firmware/runtime compatibility remains unverified.

Shared changes are limited to P04/P05 format/identity/provisioning concerns in build/check/stage/install and the existing Containerfile base argument, plus the P06 host check. Source review is the next step; executing the authored checks/builds requires a named matching-native builder/revision and a separate grant. Fresh installation/VM/operator work additionally needs its own named targets/actions. Do not attribute the historical successes below to these new bytes or delete private/persistent artifacts to obtain a fresh attempt.

## Current execution — 2026-09-06

**Recorded M15 x86_64 build/source checks passed; the local dashboard is activated and its operator browser journey is verified.** The isolated `soda-test` CoreOS KVM host runs Forgejo, the Soda dashboard, Caddy and native operator services. See [local testing](local-testing.md) for URLs, private credential locations, exact scope and logs. Initial native startup fixes and resolved Go metadata are in `88be176`; navigation changes are in `95a194d`, and dashboard-access/repository-picker changes are in `c96530c`. The combined tree with branding, console and project-CLI follow-ups has not been rebuilt or retested. No artifact publication was performed.

The full M16–M17 developer/project/workload journeys remain pending. AArch64 M18 remains unverified. A first install recovered from the discovered copy/label defects is not a fresh-disk proof of the final installer. Nested Podman and direct client routing remain the highest native risks. Additional installations, network changes and provider/lifecycle operations still require named targets and explicit permission.

### Dashboard activation and browser access

Completed Forgejo's native installer on the isolated VM with a new local `operator` administrator, created a scoped native token, and ran the real `soda-setup` OAuth bootstrap and `soda-activate`. Soda is at `https://localhost:24443`, Forgejo at `https://localhost:24444`, through loopback-only SSH tunnels; Cockpit remains at port 29090. The existing localhost test certificate/CA is used, not public production TLS. Added `scripts/test-vm.sh web-tunnel` and documented the separate laptop forwards and private credentials.

Added and executed `tests/installed/dashboard.mjs`: real browser TLS verification, Forgejo password authentication, consent/S256 OAuth callback into Soda, secure HTTP-only session, authenticated Projects/Profile/operator People navigation, and CSRF-protected sign-out. Chromium trusts the CA in an isolated NSS home; no certificate bypass or normal browser/system trust changes were used. The native dashboard process is UID/GID 2000 with zero effective capabilities, services are active, and no failed systemd units were observed. Evidence is in `dashboard-browser-check.log` and `dashboard-services.log`.

No developer users, repositories, project containers or provider runners were created. This is operator authentication/navigation evidence, not the full product journey. Corrected operator command examples to use `/usr/local/sbin` explicitly because the CoreOS root SSH PATH omits it. Dashboard bootstrap intentionally restarted only the VM's Forgejo service, not the host or VM itself.

### Forgejo repository picker

Replaced manual `owner/repository` entry on Projects with an accessible native select populated from Forgejo's `GET /users/{username}/repos`, verified against the installed 15.0.7 API schema. Requests target the signed-in user, not `/user/repos` for the privileged operator token. Pagination continues until an empty page; the dashboard filters by stable owner ID and excludes every existing project reservation, including incomplete provisioning, then sorts names. Creation still re-fetches the selected repository and enforces ownership server-side.

Added native Forgejo creation/refresh links and distinct empty/provider-error states that retain the existing Projects table. Go tests cover pagination (including server page-size caps), malformed/failed responses, cancellation, ownership/privacy filtering, existing reservations, empty states and forged selection rejection. All source/staging checks pass. Rebuilt and loaded only the dashboard image, updated its staged/native binary, and restarted only `soda-dashboard.service`; application data and the other services were preserved. The old image archive was retained after Podman's refusal to overwrite it directly.

The real browser check now verifies the picker or its empty state, native create/refresh links, and the existing OAuth/navigation/logout journey. The test operator currently owns zero repositories, so the live run verified the empty state; populated/filtering cases are covered by Go tests, not claimed as an installed populated-repository journey. No repository or project fixture was created. Evidence: `repository-picker-tests.log`, `repository-picker-build.log`, `repository-picker-deploy.log`, and `dashboard-browser-check.log`. Changes are committed in `c96530c`.

### Cockpit/Tailnet native correction

The first interactive Tailnet read exposed a missing SELinux PAM session transition: root authenticated successfully but its bridge remained in `cockpit_session_t`, where Tailscale socket access and stock systemd operations were denied. Restored the native Fedora Cockpit PAM stack while retaining the required UID-0 account gate. A new authenticated Cockpit WebSocket session now runs in the native operator context and successfully reads `tailscale status --json` and LocalAPI preferences. SELinux remains enforcing; no socket permission changes, daemon restart or Tailnet enrollment were performed. Native PAM account checks allow root and deny the existing non-operator `core` account.

Added a staging regression for the root-only gate and ordered SELinux session rules. The old staged config fails it; the corrected stage passes all five packaging checks, along with Go, TypeScript, 60 Cockpit tests and installed host checks. Existing Cockpit users must log out and back in to receive the correction. This correction is included in `88be176`.

### Accounts navigation

At the operator's request, hide only Cockpit's stock Accounts menu entry through the native `/etc/cockpit/users.override.json` merge patch. The source config is staged for future installations and applied to `soda-test`; no packages, host accounts or native account tools were removed, and no services restarted. Native `cockpit-bridge --packages` before/after output confirms that `users` loses only its Accounts label and every other menu entry is unchanged (`cockpit-packages-before.log` / `cockpit-packages-after.log`). Added a packaging regression; all six packaging checks, Go tests, TypeScript checks and 60 Cockpit tests pass. Browser sessions may need logout/login to discard cached manifests. This navigation change is committed in `95a194d`.

## Core implementation continued — protected grants and first workflow

Resumed from clean `13bd49a`. U03/U04 now have AES-256-GCM session-bound grants,
actual-scope introspection, bounded code/refresh exchange, serialized local
refresh and logout-safe update-only persistence. Production validates the
restricted external key before migrations; schema v3 preserves existing product
records and refuses wrong existing encryption keys. Setup/activation source
provides the new key only for first installs; existing-state migration and
rollback are documented separately, not executed.

U05–U07 now have individually registered acting-user account/settings/Git-key,
Forgejo People/create-person, repository list/create/detail/content and Soda
environment list/create/detail/join/member/connection APIs, with connected React
routes/forms. New APIs never read the bootstrap token. Forgejo People uses actual
native admin authority, not Soda operator ID. Creation reserves once, does not
join the creator, and reports retained incomplete native results. Join calls the
real fixed account helper before saving membership. The new core-owned helper
`/connection` reads only a fixed public Ed25519 host key and observed address;
stopped containers stay stopped, and routing is explicitly unverified.

Follow-up source adds actual Link/total-header pagination (without following
provider URLs), safe GFM/relative README rendering with raw HTML and automatic
image loading disabled, bounded 8-MiB authenticated attachment downloads and a
frontend render-error boundary. Markdown pins were selected from public npm
metadata; no dependencies were resolved or installed. Pagination, download
headers/size/traversal and Markdown safety tests are authored, not executed.

U01 inspected matching upstream v15.0.7 OAuth, route middleware and repository
search source. Important limits: confidential-client consent can remain at old
scopes; token responses omit scope, so the new flow introspects actual consent.
Native refresh counters are per user/application: another session may invalidate
older refresh material, which requires reauthentication rather than credential
sharing or changing upstream settings. See [API coverage](forgejo-api-coverage.md).

Authored Go encrypted-storage/binding/legacy-session/key-restart tests, acting-user
admin denial/nonoperator-admin tests, refresh/logout race, JSON environment
create/explicit-join/failure tests and fixed-public-key helper tests. Authored DOM
repository-create/inert-content/ref and failed-join tests. Go formatting and diff
checks only have run. No dependency resolution, build, type check, product test,
provider mutation, native service/network operation or deployment has run. The
VM, builder infrastructure and backing image are unchanged.

U09 follow-up source connects commit/file history, pinned commit detail/bounded
unified diff, branch/tag list/create, comparisons, SHA-preconditioned file
create/edit/upload, personal forks and one-time HTTPS Git imports. These operations
use native permissions, do not execute Git in Go or copy/mutate environments, and
keep import credentials transient. Authored stale-file/protected-branch/import
ownership and editor-draft tests. Added a real-handler OAuth callback/grant test
and conservative token expiry measured before the exchange, not after later
identity/introspection calls. All checks remain authored, unexecuted. Blame and
full native diff/import capability coverage remain U09/U17 audit work.

U10 source now adds issue list/filter/create/detail/edit, comments/edit,
assignments/state/labels/milestones, reactions, own subscription and bounded issue
attachment upload, with connected React views and sanitized lossless IDs. The
selected native router requires `issue` scopes separately from `repository`;
default consent now includes `write:issue`. Native permission checks remain on
every operation, no issue data is stored in Soda, and these operations never call
the environment helper. Authored native-denial, actor-binding, multipart-boundary
and create-issue DOM tests; none executed. Structured native issue templates,
comment-attachment/reaction details and full stale/permission/native journey
coverage remain source work. Attachment downloads explicitly use configured
native Forgejo links, not borrowed cookies or a fabricated custom binary proxy.

U11 source adds connected PR list/create/detail, a shared native issue/PR
conversation component, commit/file/diff/status inspection, reviewer requests,
explicit-head reviews/new-side inline comments and protected native merge. Native
review/merge source was inspected at v15.0.7: review comments use file line numbers,
reviews receive explicit `commit_id`, and merges receive `head_commit_id` with
force/auto-merge/branch deletion fixed false. File/diff reads check displayed
head/base/merge-base before and after retrieval; stale snapshots fail closed.
No merge operation calls the project helper. Authored changed-diff, stale-review,
merge-protection/no-force and retained-draft DOM cases; none executed. Existing
inline-thread rendering, old-side positions, team reviewers and full native
permission/conflict evidence remain pending. Native pending comments can remain
after a failed review submission; no automatic retry or invented recovery occurs.

U12 hook source now exposes native metadata/list/create/edit with URLs and
credentials write-only to the browser. Inspected v15.0.7 hook implementation:
responses contain decrypted authorization headers; PATCH resets omitted
headers/events/filter, does not rotate signing secrets, and cannot change existing
package/action event flags. The adapter redacts outputs, preserves omitted values
within the request and rejects unsupported changes rather than claiming success.
Forgejo owns target delivery/host policy; Soda never calls a hook URL. Its pinned
default resolves an empty allowed-host setting to external hosts and verifies TLS;
no appliance webhook override was found by source inspection. Authored redaction,
preservation, unsafe-input and native-denial tests remain unexecuted. U12 now also has connected repository metadata/feature/merge settings, native
collaborator permission inspection/set/remove, Git deploy-key list/add/remove,
and branch/tag protection list/create/edit. Repository settings submit only
changed fields; native rename/transfer/archive/delete are not exposed. Deploy
keys remain separate from Soda development keys, and provider access changes do
not propagate to Linux. Pinned branch/tag PATCH implementations were inspected;
omitted protection values are retained, otherwise-ignored nested push flags are
rejected, and IDs remain lossless. Added settings-only-patch, privilege-denial,
public-key rejection and protection dependency/opaque-ID cases plus a focused DOM
test. Only gofmt/diff inspection ran. Advanced branch-policy form details,
the full native permission matrix and advanced form details remain pending.

Organization/team source now adds native directory/create/profile, members/team
lists, scoped team detail/description, explicit member and repository assignment
changes and bounded policy APIs. Default consent adds verified `write:organization`.
Pinned router, organization/team mutation and unit definitions were inspected:
organization PATCH clears omitted profile strings (retained in-request), team
policy has parent-permission dependencies and administrator units cannot be
pretended to accept limited overrides. Forgejo resolves actual organizations,
teams, repositories and all permissions; no organization/member/role rows are
stored in Soda. Org-owned environments remain unsupported under the existing
human-owner rule. Authored preservation, opaque-ID, direct-denial and ignored-field
cases are unexecuted. Advanced team policy/protection forms and native matrix
acceptance remain source/execution work.

### Current milestone ledger

| Milestones | Source | Built / tested / installed | Remaining |
| --- | --- | --- | --- |
| U01 | Partial first-workflow scope/token/visibility audit | No new execution | Full action inventory, dependency closure/licenses, remaining native capability research |
| U02 | Partial preview/build packaging and new routes | None | Real lockfile/resolution, full error boundary/dev arrangement and frontend verification |
| U03/U04 | Connected encrypted-grant/config/API source and focused tests | None | Executed migration/key/refresh/consent/permission suite, more callback/scope race cases, controlled upgrade rehearsal |
| U05 | Native profile/Git SSH keys/People + Soda preferences/development keys | None | Native onboarding proof, remaining account/key coverage and focused UI cases |
| U06 | Repository list/create/detail, ref-aware text/GFM/README, bounded downloads and native pagination metadata | None | Full visibility/ref/content tests, dependency resolution, browser/native-write proof |
| U07 | Connected persistent create/join/inspect/member/connect paths | None | Additional partial-result/authorization/connection cases and native evidence |
| U08 | Pending | None | Explicit build/deployment/fixture/lifecycle permissions, approved direct client route and two-project/workload/persistence proof |
| U09 | Connected history/ref/compare/file-write/fork/basic-import source | None | Expanded focused tests, blame/native diff/import capability audit and real native Git verification |
| U10 | Connected issue/comment/label/milestone/reaction/subscription/issue-upload source | None | Native templates/comment-asset detail, expanded focused cases and two-user native proof |
| U11 | Connected bounded PR/review/merge source and revision-bound mutation tests authored | None | Inline-thread/old-side/team detail and real reviewer/merger/native conflict journeys |
| U12 | Connected repository settings/access/protection/hooks and organization/team source | None | Advanced team/protection forms, sub-action coverage and full native permission matrix |
| U13 | Connected overview/search/notifications/native profiles/activity | Local Go/UI/type/build checks passed; no installed proof | Native two-user visibility/update journeys and coverage closure |
| U14 | Connected runs/tasks/workflows/dispatch and repository/organization Actions configuration | Local Go/UI/type/build checks passed; no provider mutation | Real approved provider/runner journeys and explicit native run-control gap disposition |
| U15 | Connected releases and bounded assets | Local Go/type/build and existing UI suite passed; no installed proof | Wiki/packages, release DOM cases and native permission/transfer journeys |
| U16–U20 | Pending | None | Admin expansion, coverage decisions, gated cutover, polish and final native architectures/fresh-install/upgrade proof |
| E01–E03 | Unselected | None | Explicit selection required; no dormant controls added |

No U milestone is complete. GFM rendering is not parity with all Forgejo Markdown extensions. Native
account-security links and consent recovery are labeled upstream dependencies;
no runtime fallback or conditional environment extension was implemented.
[Credential migration/rollback](dashboard-credentials.md) remains an authored
procedure, not installation evidence. Execution gates do not block continued
independent source implementation.

## Core implementation started (first batch, historical)

The user explicitly requested implementation of the complete leading core plan. Began from clean `b7241b2`; the P plan remains outside support and E01–E03 remain conditional. This is the first connected source batch, **not completion of U01–U20 or new native evidence**.

- **U01 audit in progress:** added [Forgejo API coverage](forgejo-api-coverage.md) from the saved 15.0.7 schema with its hash, endpoint/option families, authority ownership and explicit unresolved scope/refresh/API-gap work. Inspected public npm metadata for React Router 7.18.3 and React-plugin peers; selected only the router addition. No broad admin-token proxy or invented upstream endpoint was added.
- **U02 source in progress:** added `dashboard/` with client React/PatternFly/Vite+/Zustand, browser routing, canonical assets, session state, actual profile/key forms and explicit legacy Projects/People links. Go mounts compiled output at `/app/`, verifies manifest/entry/assets/licenses before opening the database, keeps legacy routes/default intact and returns 404 for missing asset files. Added a dashboard-only native build entrypoint and integrated its assets into existing full build, image, staging and check entrypoints. Infrastructure/P tools were not ported.
- **U03 source in progress:** added JSON session discovery/logout, Soda preference and development-key APIs; method/content/body/unknown-field validation; mandatory JSON origin/header CSRF checks for every unsafe method; non-redirecting JSON 401/403/404/405/413/415/503 failures and no-store responses. API identities are decimal strings, and public-key parsing is shared with the legacy form. No new API uses an operator provider credential.
- **Schema v2:** append-only transactional migrations recognize existing v1, preserve product/session rows and add the constrained OAuth return path. `/login?return_to=/app/` binds the return destination to single-use state without changing the native callback or weakening PKCE. Unknown/newer/invalid schemas are refused. The old binary's positional OAuth inserts are not v2-compatible; preserve a consistent pre-deployment DB/config/artifact set. No live database was opened or migrated.
- **Tests authored:** Go API/method/CSRF/privacy/key/session tests, legacy migration/preservation/refusal tests, OAuth return binding, frontend filesystem/manifest/routing tests, TypeScript fetch/session-race/form tests and staging checks. No tests or type checks have been executed for this batch.

**Remaining immediate work:** U01 exact selected-version OAuth scope/middleware/refresh audit; U02 dependency resolution/real lockfile, full frontend verification, error boundaries and trusted HTTPS development arrangement; U03/U04 protected session-bound grants/configuration/refresh/logout concurrency; full Forgejo account/People/repository/environment APIs and subsequent collaboration/admin milestones. The current OAuth still requests `read:user` and discards the identity access token; the preview exposes only Soda-owned operations and preserves native/legacy links. Do not mistake it for expanded Forgejo authorization or U05 completion.

**Execution:** source/metadata inspection, source edits, formatting and diff/documentation review only. No dependency installation/resolution, compilation/build, product tests, generated frontend assets, provider mutations, VM/network/service actions or deployment. The new dashboard lockfile is deliberately absent rather than fabricated; frozen build scripts fail with an explicit preflight message until authorized resolution/review. Go formatting completed. Frontend formatter attempts failed at the wrapper/workspace/config boundary because the new dashboard dependencies are not installed; no successful frontend format/type/test result is claimed, and no dependency installation was used to bypass the gate. The running test VM remains unchanged. See [dashboard source instructions](../dashboard/README.md) and [API/migration contracts](dashboard-api.md).

### Provider response boundary follow-up

Rechecked clean HEAD `70fe1da` and read the complete leading plan, handoff,
architecture, inventory, deferred scope and native installation/validation guides.
U03/U04 provider transport now returns sanitized typed HTTP status errors,
preserves request cancellation, and rejects oversized, null, malformed or
trailing JSON responses, including OAuth token responses. It does not retain raw
provider errors or retry with another credential. Existing callers use this
boundary; OAuth scopes and grant retention are unchanged.

Authored focused response-size/format, status/denial/no-retry, cancellation and
OAuth-response tests. Only Go formatting and `git diff --check` ran; no tests,
builds, dependency resolution, provider requests or native changes ran. Requested
specific build/test, backed-up dashboard deployment, fixture and lifecycle
permissions; routing/client authority still needs an explicit selection.

Milestone ledger remains: U01/U02/U03 and Soda-local U05 partial source;
U04 has only this transport preparation, not per-user grant integration;
U06–U20 pending. No new milestone is source-complete, built, source-tested or
installed-verified. No new native fallback was implemented. E01–E03 remain
unselected. Execution permissions do not block independent source work, which
also remains unfinished; this follow-up does not complete the assignment.

## Core/native plan coordination merge

Pulled `origin/main` (`9c8d672`) into local `55ce5cb` with `git pull --no-rebase --no-commit origin main`, preserving both documentation histories. The only textual conflict was the introduction to `docs/implementation-plan.md`; resolved it by retaining the historical M01–M18 context, the leading U plan and subordinate native-support reference. No application-source conflict or application change was involved.

At the user's direction, [dashboard-implementation-plan.md](dashboard-implementation-plan.md) leads the core—including Go/API/auth/data, production `internal/host`/`project-os`, shared build/config contracts and U08/U20 product acceptance. Reworked [native-porting-plan.md](native-porting-plan.md) around outside VM/QMP/SSH/evidence/artifact tools, provisioning transport and retained host-operator integrations. Former P07/P08 redirect to U08/U20; their useful direct-IP, shared-installation, workload and bounded-persistence test details are retained in the core plan. P06 owns host/service observations, P11 outside integrations, and P12/P13 scoped support evidence—not parallel browser/product suites or readiness verdicts. P09/P10 media remains conditional and is not a core gate.

Added explicit shared-file/input/output ownership, evidence reuse and non-circular ordering to both plans; aligned AGENTS, README, architecture, inventory, deferred scope, historical plan, installation, native validation and reuse guidance. Existing authorized tools may support U08 without waiting for the P port. No U/E/P implementation milestone is completed by this coordination, and existing native limitations/evidence are unchanged.

This merge/coordination performs documentation/diff, conflict-marker, local link/anchor, milestone-reference and whitespace review only. No builds, product tests, dependency installation/resolution, artifact generation, VM/service/network/provider operations, push or publication ran. The merged application remains unrebuilt/unretested.

## Unified React dashboard planning

Recorded the user's next frontend direction and a proposed full page/direct-dependency inventory in [dashboard planning](dashboard-plan.md): client-rendered TypeScript/React, PatternFly, Vite+ and Zustand, without SSR, Tailwind or TanStack. Go remains the application API and Forgejo the identity/Git/collaboration authority. The plan separates the first working repository-to-environment flow, later functional coverage and native screens/API gaps; it does not add a host-administration frontend or a second password authority.

Inspected current source/manifests and the previously saved installed Forgejo 15.0.7 API schema. The current OAuth flow requests only `read:user` and does not retain access/refresh tokens for subsequent user-scoped operations; the plan explicitly requires extending that lifecycle rather than using the operator credential as a universal proxy. Updated guidance to distinguish the selected next direction from the still-existing HTMX implementation. Only documentation was changed and whitespace/diff review performed. No dependencies were installed or changed; no builds, product tests, VM operations or live provider requests ran for this planning work. Page scope and new dependency pins remain proposals, not implemented capabilities.

Clarified the upstream boundary at the user's request: Soda extends Forgejo with development environments, project-local access and public-key provisioning; frontend ownership does not transfer Forgejo's backend/data/permission responsibilities. Expanded the plan to explicitly include upstream-authorized Forgejo site-administrator views instead of permanently relegating them to the native UI. Native screens remain fallbacks for unverified interfaces; Cockpit remains host administration. Removed the illustrative backend ratio from the plan. Reviewed the saved `/admin` API schema and updated guidance only; no application, dependency or runtime changes were made.

### Multi-milestone frontend/backend plan

Added [the U01–U20 implementation plan](dashboard-implementation-plan.md) at the user's request. It sequences source-backed capability research, React/static packaging, JSON/schema migration, per-session Forgejo credentials, accounts/admin onboarding, repositories, native environments and early installed proof before broader collaboration/admin coverage, cutover, polish and final verification. It includes source ownership, API/state/security contracts, data/config migration and rollback constraints, page-to-milestone coverage, dependencies, acceptance cases and evidence levels. Forgejo remains upstream; the Go layer does not acquire its business or permission ownership.

Rocky/Fedora profiles, existing-environment lifecycle controls and basic resource caps are conditional E01–E03 tracks, not silently approved scope. Other previously deferred lifecycle/ownership/recovery features remain explicit decisions. Linked the current plan from guidance, architecture, inventory and README and marked the original M01–M18 plan historical. This is documentation only; no implementation milestone, dependency resolution, compilation, product test, provider call, VM action or deployment ran. Documentation paths/anchors, milestone coverage references and whitespace were reviewed; the planning work was committed as `55ce5cb`.

## Source merge — 2026-09-06

Merged the local predecessor follow-ups (`0f25570`, `4e751ac`, `f2523ac`) with remote history through `95a194d`. Resolved conflicts in agent guidance, README navigation/status, operator setup and the project image. The image retains native `curl-minimal` and mise checksum fixes alongside Tea/GitHub CLI inputs; combined staging tests retain both the PAM regression and the branding/console checks. The real Go metadata and native installation fixes are preserved.

The earlier native evidence does **not** validate this merged tree or its additional checks. This merge performs source/diff, conflict-marker, whitespace and local documentation-link review only. No dependency resolution, builds, product tests, VM/service operations or provider actions were run. The current request authorizes Git merge/commit/push, not additional native execution.

### Dashboard follow-up merge

Merged `origin/main` through `f4fe066` into `c96530c`, preserving both histories. Resolved five documentation conflicts by retaining the newer dashboard activation/browser and repository-picker evidence alongside the predecessor follow-ups and their unvalidated status. Reviewed the automatic staging/configuration merge; Accounts navigation, PAM checks and branding/console checks are retained. Updated stale uncommitted-change references.

Only source/diff, conflict-marker and whitespace checks were performed for this merge. No builds, tests, dependency resolution, VM/service operations or provider actions were run; the combined tree remains unvalidated.

## Original native artifact and acceptance proposal (scope superseded above)

Upstream commit `9c8d672` added the original [porting proposal](native-porting-plan.md), based on current source `6f7b51e` and predecessor `bc1d3e0`. It maps selected VM/QMP, process/cleanup, SSH/evidence, artifact-inspection and scenario source/tests to proposed destinations, with P01–P13 dependencies, source/native exits and exact-target execution gates. CoreOS remains the proposed host path; bootc/Anaconda, the old account model, Updates and release publication/qualification machinery remain excluded.

The plan explicitly distinguishes application OCI from a bootable host image, public media from private provisioning, a QCOW2 deployment kit from a preinstalled image, and forwarded access from a real client route. It prioritizes fresh x86_64 installation/developer/persistence evidence and preserves independent aarch64 follow-up. Existing `soda-test` state and its backing disk must remain untouched.

Planning/documentation only. Reviewed source/documentation, diffs, local documentation link targets, plan anchors and whitespace. No port implementation, generated artifacts, builds, product tests, dependency resolution, VM/service/network/provider operations or publication ran. At that proposal's creation P01–P13 were not started. The coordination merge above transfers P07/P08 into core U08/U20 and narrows the remaining P scope; active P milestones were still unimplemented at that coordination point. The subsequent source-only support implementation is recorded above; native exits remain unverified. Prior native results and the merged tree's unvalidated status are unchanged.

## Original source handoff (historical)

The M01–M14 entries below describe the original source-only handoff, when builds, tests, type checks, dependency resolution, installation and publication had not run. Their original “not run” statements are historical; current evidence is recorded above and in [local testing](local-testing.md).

## Baseline

- New backend and privileged integration: Go; no Rust subsystem without a concrete need. Retained Cockpit frontend: TypeScript/React.
- Go 1.26.7 (predecessor source baseline); HTMX 2.0.10 vendored from its npm distribution with license.
- Host candidate: Fedora CoreOS stable 44.20260817.3.2, reported for x86_64/aarch64 by the upstream stable stream metadata. Native compatibility remains unverified.
- Predecessor source: local `soda-os` commit `bc1d3e0dbec48dfaa6a20f9d0453ad3e4cdf353c`. No changes are made in that repository; Updates remain excluded.

## M01 — source implemented

Go server/configuration, embedded canonical branding, HTMX source, initial templates, narrow process runner reused from predecessor, and authored configuration/server/process tests. Application features follow in subsequent milestones; this foundation is not a completed product.

Build: not run. Validation: not run.

## M02 — source implemented

SQLite via modernc.org/sqlite v1.58.0 (upstream module metadata), schema and concrete store, hashed session tokens and single-use OAuth state storage. Startup opens the persistent database. Tests authored, not run; go.sum/dependency resolution remains a later build input step.

## M03 — source implemented

Forgejo 15.0.7 API client, confidential OAuth setup CLI, persistent Forgejo container definition and operator-native installation instructions. Native API/schema source inspected; no live accounts or OAuth applications created.

## M04 — source implemented

Forgejo OAuth authorization code + S256, session rotation/logout, CSRF/origin checks, profile/public-key forms, operator user creation, and real provider/database wiring. Tests authored; not run. Browser and native OAuth behavior remain unverified.

## M05 — source implemented

Rocky 9.6 + mise project image recipe, systemd/OpenSSH initialization, native project-account setup, root-owned SSH keys and retained writable-rootfs strategy. Native image/account/SSH behavior remains unbuilt and unvalidated.

## M06 — source implemented

Fixed-operation root helper over a root:soda systemd Unix socket; dashboard client wired. Native Podman create-once/start/inspect/account commands, user-namespace mapping, retained project units and routed bridge IP readback. Configure a real route to the chosen project subnet before client SSH; bridge allocation alone is not remote reachability proof. Native helper/network/persistence tests authored, not executed.

## M07 — source implemented

Project discovery/create/detail and explicit join routes now reach the provider, native helper and database. Repository ownership is checked server-side; membership records follow native account success. Real IP inspection and non-ready errors are displayed. Authored a provider/native-double create-flow test, not run.

## M08 — source implemented

Shared mise install/shim path and native global config are wired into login profiles and SSH non-interactive environments. Project-owner installation instructions and authored two-user checks accompany ordinary Git/shared-file guidance. No private resource selection machinery. Build/validation not run.

## M09 — source implemented, high native risk

Nested project-local Podman service, owner-only shared-engine socket, Compose 1.6.0 provider and build/mount/database example. Parent user namespace, capabilities, fuse, SELinux and cgroup combination requires native proof; no privileged host shortcut or dormant fallback added. Build/validation not run.

## M10 — source implemented

Selected native CoreOS package layering for Cockpit, root-only PAM, loopback-first operator access and native /etc/cockpit/branding delivery. Shared frontend dependency closure/build source and licenses ported without old Projects/Updates entrypoints. Feature entrypoints follow in M11/M12. No builds or tests run.

## M11 — source implemented

Tailnet page/native state/store/components and focused tests ported. Native host refresh now updates Forgejo Git SSH advertisement while preserving configured browser/OAuth origins, inspects live environment and restarts the actual container unit when needed. No Tailnet enrollment or native execution performed. M14 additionally verifies the actual Git SSH listener before advertising a Tailnet address, avoiding unreachable clone guidance.

## M12 — source implemented

Runners page, native provider/lifecycle/helper/launcher code, service wiring and focused tests ported. Root-only authorization replaces the incompatible predecessor human-host-admin rule. Configured Forgejo origins now feed native registration and browser links. Provider-client source locks retained; no binary download, registration, build or validation performed.

## M13 — source implemented

Native build/staging, first-install and private Butane provisioning recipes authored; dashboard/proxy Quadlets, TLS activation, service identity/ownership, native extension package requests, provider-client checksum fetch and branding staging included. Routes require explicit deployment configuration; no project DNS/gateway added. Native staging tests authored. No build, package install, provisioning render, provider download, activation or validation executed.

## M14 — source implemented

Integrated authenticated navigation, visible HTMX error responses, pending native-operation states, basic accessibility and direct SSH/SCP/SFTP guidance that is withheld for stopped/unavailable endpoints. Added server-side compatible username validation, HTTPS/loopback configuration guards, provider/native journey and privilege test source, and the missing retained Cockpit process-test support. Removed predecessor-guessed Tailnet service URLs; actual listener inspection now guards Forgejo advertisement. Corrected native Forgejo SVG/favicon destinations and relocated theme imports from inspected upstream template paths without modifying canonical branding assets.

Authored explicit matching-native source/staging check entrypoint and the full later Alice/Bob, persistence, Tailnet and provider-runner journey. Architecture documentation now reflects implemented choices rather than describing them as unresolved. x/sys v0.47.0 source metadata was inspected; dependency resolution was not run.

**Source-complete, unbuilt, unvalidated.** No builds, compilation/type checks, tests, dependency installations, provisioning renders, native activations, enrollments, registrations or CI jobs were executed. M15–M18 remain held.

## Repository guidance update

Customized `AGENTS.md` from the predecessor's engineering guidance: requirement-versus-choice classification, human-maintainable design, coherent refactoring/reuse, source ownership, actual script side effects, scoped commit authorization and separate evidence reporting. Retained SodaOS's execution hold, project-local authority, private networking and persistent-container boundaries; excluded obsolete predecessor test/release commands and UI assumptions.

Documentation-only change. Reviewed the predecessor/current guidance, owning documentation, script source and full diff; local documentation links and whitespace checked. No product behavior changed and no builds, tests, dependency resolution or deployment operations ran.

## Compatible predecessor follow-up ports

Source reference remains `soda-os` commit `bc1d3e0dbec48dfaa6a20f9d0453ad3e4cdf353c`; that repository is unchanged.

- **Branding/configuration:** ported Forgejo browser checks, PNG renderer/pixel comparator and focused test source. Added fresh-evidence and explicit native-review boundaries; actual renderer verification is opt-in with the `branding` test tag. Adapted app metadata, native/accessibility theme choices and cache revalidation into a staged Forgejo environment source file, using the selected upstream environment-to-INI encoding. The disposable component sheet is not an appliance payload. No artwork regeneration, browser checks, tests, builds or native configuration changes ran.
- **Console/handbook:** adapted the native operator welcome, main-table and connected-uplink discovery, interactive-only hook and CLI delivery. It reads configured public origins without printing secrets or inventing reachable ports. Added command-double/staging test source. Reworked developer instructions for project-IP SSH, editors, separate Git credentials, real shared mise installations and project-local service ports; adapted the screenshot brief without adding fabricated captures. No native console, tests, account operations or screenshots were executed.
- **Project CLIs:** retained Tea 0.15.1 source lock/license/fetch behavior and adapted its native Makefile build into project-image inputs, with fetch/build-boundary test source. Added the predecessor's GitHub CLI 2.97.0 baseline through GitHub's signed RPM repository inside Rocky rather than copying a Fedora host package. Developer auth remains native and personal. Source/release metadata was inspected; no archives, binaries, RPMs, dependencies, builds or logins were fetched/executed. The reuse inventory and later validation guide separate these ports from the excluded workspace/release/Updates systems. Source review also tightened console origin parsing and removed stale predecessor RPM/page/renderer instructions and a broken link from the Cockpit asset guide. Formatting, diff/whitespace inspection and local documentation path checks are the only checks executed.
