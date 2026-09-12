# Upstream-first refactoring review and implementation plan

**This is the single maintenance implementation plan:** actions, order, decisions,
exit criteria and current source status live here. The
[upstream ownership audit](upstream-ownership-audit.md) owns findings and research;
[the handoff](implementation-status.md) owns actual execution/delivery evidence.
Do not create a second backlog in the audit or repeat completed work under a new label.

The original planning review used `c20abc3` on 10 September 2026. A–D and the selected
E admission slice now have local source checks; native delivery remains separate.

The [remaining audit implementation plan](#7-audit-remediation-implementation-plan)
is based on the rebased `1f1a9d3` tree. It is the current maintenance order and
accounts for already implemented schema checks, source-check wiring and shared
session/page mechanics. The earlier sections retain their design/evidence context.

## Requested full upstream-ownership audit — source review complete

The [audit](upstream-ownership-audit.md) is the evidence source for the remediation
phases below. Its coverage, selected upstream versions and review limitations live
there, not in a second inventory here. The [repository instructions](../AGENTS.md#working-style)
and feature-owner contracts continue to govern implementation; neither document
grants native execution or deployment permission.

## Decision

Keep the selected architecture. Improve Soda's integration and development feedback;
do not build smaller replacements for Forgejo, Lit, tmux, systemd, OpenSSH, Podman,
CoreOS Installer or the existing test runners.

Use the [current status and order](#current-status-and-order), not the historical
A–G labels, to select remaining work. Test wiring, v8 completeness, bounded frontend
cleanup and cancellation-aware admission must not be scheduled again. Broad
frontend decomposition, a generic terminal manager, per-project scheduling and a
new installation engine are **not** prerequisites or selected rewrites.

The [Sodaspaces plan](sodaspaces-plan.md) still owns feature order. This document owns
maintenance recommendations, not another product roadmap. The [architecture](architecture.md),
[Project OS](project-os.md), [frontend improvement guide](frontend-improvement-plan.md),
[installer plan](coreos-installer-plan.md) and [deferred scope](deferred.md) retain
their responsibilities. Actual execution belongs in the [handoff](implementation-status.md).

## 1. Upstream-first review

### The test for adding Soda code

For each proposed adapter or extraction:

1. Identify the actual owner and inspect the selected upstream version/configuration,
   plus Soda's existing caller. Missing JSON or an inconvenient interface is not
   evidence that upstream lacks a workflow.
2. Prefer the supported native workflow/configuration/API. Use its existing renderer,
   protocol, lifecycle and persistence rather than reconstructing them.
3. State the remaining Soda requirement precisely: for example, binding an authorized
   Forgejo actor to an original project account, not implementing Linux identity.
4. Add only the code needed at that boundary. Keep one source of authority and explicit
   callers; no generic controller, scheduler, schema engine or provider abstraction
   just to move branches out of a large file.
5. If the native mechanism cannot meet the requirement, record the exact limitation and
   return for a product decision. Do not fork upstream, weaken security, silently change
   the product or abandon an unfinished feature to make an adapter easier.

Using upstream does not mean leaving users to assemble missing integration. Real
Create/Join/access, working installer continuation and usable application onboarding
remain Soda obligations.

### Responsibility and reuse findings

Use the audit's [package ownership inventory](upstream-ownership-audit.md#coverage-and-selected-baselines)
and [findings](upstream-ownership-audit.md#findings). This plan does not maintain a
second upstream responsibility matrix. The original review's detailed comparison
remains in Git at `22f5c20:docs/refactoring-plan.md`; research provenance is linked
[below](#6-research-provenance-and-references). Implementation constraints remain
with the concrete slices and feature owners.

## 2. Disposition of the original findings

Numbers refer to the original audit, not new product milestones.

This is a routing index, not a second findings/status table. Each slice below owns
its implementation record; the audit owns the source rationale.

| Original finding | Single implementation owner |
| --- | --- |
| 1–2. Test invocation and repeated preparation | A |
| 3. Schema completeness | B |
| 4. Current versus historical guidance | G |
| 5. Frontend readability | D |
| 6. Session/page plumbing | C; missing mutation checks are separate phase 2 work |
| 7–9. Registry, validator and helper admission | E; phase 5a points to the same admission fix |
| 10–11. Installer and profiles | F and its feature-owner plans |

Canonical tokens, enforced Lit diagnostics, typed views, shared runner response
contracts and one authored Spaces source directory are **already implemented**.
Do not count redoing them as progress. File length and repeated checks across trust
boundaries are not sufficient reasons to extract or delete code.

## 3. Recommended implementation slices

Keep commits reviewable. Do not combine formatting, security behavior, native
concurrency and feature changes in one patch. Each slice below extends the existing
product-owned tests; it does not create a second acceptance gate.

### A. Reliable, single-preparation source checks — first

**Implemented with local source/browser checks.** See the
[commands and prerequisites](typescript.md#local-source-checks) and
[revision-specific results and failures](implementation-history.md#refactoring-step-1--local-source-check-wiring).
This does not complete native-stage or installed acceptance, or the later slices.

**Owners:** [root scripts](../package.json), [Spaces fixture orchestrator](../scripts/test-spaces-page.ts),
[frontend tests](../tests/frontend/), [Forgejo tests](../tests/forgejo/),
[native check](../scripts/check-native.sh).

1. The native integration step 4 supersedes the original Go HTML producers:
   `scripts/test-spaces-page.ts` invokes the existing `TestNativeConnectionFixture`
   in a fresh retained directory. It prepares canonical assets, authenticates with
   the authorized native account and runs all three existing browser consumers.
2. Missing native host, credentials, assets or failing consumers fail the page gate.
   Consumers retain synthetic operation responses while using actual native HTML;
   no duplicate shell/host/auth harness replaces the retired Go templates.
3. Include `tests/forgejo/settings-link.test.ts` in the explicitly enabled local
   browser group. Currently it requires `SODA_LIT_BROWSER=1` but is absent from the
   `test:lit` file list. Set local fixture flags only on the relevant commands;
   do not enable unrelated installed/provider tests globally.
4. Make aggregate `test` prepare locked terminal assets and emitted Forgejo modules
   once, then run the existing frontend/page/layout/Forgejo/Cockpit suites. Keep
   focused commands convenient without duplicating their test bodies. Explicit
   preparation ordering is enough; do not add caching infrastructure or a daemon.
5. Offer one source-check aggregate of existing Go, TypeScript/Lit, browser/Cockpit
   and local Python checks without requiring a sealed native stage. Keep native
   architecture, clean-revision and staging verification in `check-native.sh`;
   reuse the same source commands there. List prerequisites and effects accurately.

**Exit:** every requested page journey actually executes against fresh Go HTML/CSP
and emitted assets; selected fixture/tool absence fails; the aggregate prepares
assets once; focused entrypoints still work. Existing failure and authorization
cases remain. Measure elapsed preparation/suite time when implementation is tested,
without promising an unmeasured speedup or treating prior counts as new results.

### B. Complete schema-v8 validation — independent small fix

**Implemented with local source checks.** The
[handoff](implementation-history.md#refactoring-step-b--schema-v8-completeness)
records the reproduced failures, independent historical fixtures and passing checks.
No migration history was rewritten and no retained database was accessed.

**Owners:** [migrations](../internal/store/migrations.go),
[migration tests](../internal/store/migrations_test.go),
[profile migration tests](../internal/store/project_profile_test.go).

- Include `projects.creation_profile` and `oauth.repository_settings_return` in the
  existing zero-row column checks.
- Check the required `immutable_creation_profile` trigger in SQLite's schema catalog;
  preserve functional tests that an update is rejected. Keep this a named Soda
  schema requirement, not a SQL parser, generalized schema-diff engine or repair loop.
- Add malformed-current-version fixtures missing each required addition and retain
  fresh/populated upgrade, rollback-on-failure, legacy-unknown and OAuth single-use
  cases. Prefer an independently preserved historical SQL fixture where changing
  migration source would otherwise silently change both implementation and test input.
- Retain SQLite's native integrity/foreign-key checks in their existing validation
  callers. They complement required-schema checks; they are not substitutes for them.

**Exit:** malformed current schemas fail closed without being silently reconstructed;
valid fresh and populated upgrades retain records/constraints. No migration-history
rewrite, schema-version bump solely for this check, ORM or dependency change is needed.
Do not inspect or modify a retained database to manufacture a failing fixture.

### C. Current-session and page mechanics — bounded cleanup

**Implemented with local source/browser and web race checks.** See the
[handoff](implementation-history.md#refactoring-step-c--current-session-and-page-mechanics)
for the bounded extraction, explicit Spaces stored-user consistency hardening,
preserved caller policy and actual successful/failed checks. Cookie parsing already
has a shared owner; differing page policies remain explicit rather than generalized.

**Owners:** [API guard](../internal/web/api.go),
[repository authority](../internal/web/environment_authority.go),
[provider grants](../internal/web/provider.go),
[Spaces page](../internal/web/spaces_page.go),
[runner page](../internal/web/settings_page.go),
[repository settings](../internal/web/repository_settings.go).

1. Give the repeated current-session comparison a concrete helper with explicit
   original-session input and an error result. Check stable user ID, `ContextID`
   and CSRF. Leave response encoding/status selection with each HTML/API caller.
2. Preserve call ordering: authorize before metadata/native effects; recheck at the
   existing post-external-I/O boundaries. A request-local result is not a cached
   provider role. Do not mechanically replace stronger terminal membership checks
   or move their store checks outside the lock that coordinates logout/admission.
3. Share genuinely identical origin/cookie/buffered-template mechanics only where
   useful. Keep page data and feature authorization explicit. Spaces alone needs
   its xterm inline-style CSP exception; settings must not inherit it.
4. Keep Forgejo as identity/permission authority and reuse its acting-user APIs.
   Do not replace Soda's bounded adapter session with borrowed Forgejo cookies,
   or replace native authorization with a generic role/policy framework.

**Exit:** current-actor, duplicate-cookie, CSRF/origin, denied/unavailable provider,
logout-during-I/O and rendering-failure coverage remains at real handlers. No route,
OAuth scope, cookie or operation-authority change is concealed as extraction.

**Later, only with a new settings destination:** consider one validated in-memory
OAuth destination value mapped to the existing persisted fields. Preserve fixed,
transaction-bound returns and append-only schema evolution. Do not migrate every
stored login merely to replace booleans or introduce arbitrary return URLs.

### D. Readable frontend ownership using Lit — incremental, not a rewrite

**Implemented as a bounded local slice.** See the
[handoff](implementation-history.md#refactoring-step-d--frontend-readability-and-measurement-ownership)
for formatting equivalence checks, the private measurement controller, named owner
commands and emitted-browser/source results. Existing terminal/project authority,
payload ownership and unfinished features remain; no async-task dependency was needed.
This is not a claim that every template needs—or received—a wholesale restyle.

**Owners:** [workspace](../frontend/spaces/sodaspaces-workspace.ts),
[project controls](../frontend/spaces/sodaspaces-project.ts),
[terminal](../frontend/spaces/sodaspaces-terminal.ts),
[authoring guide](lit.md), [existing composition guidance](frontend-improvement-plan.md#6-component-composition-and-preserved-owners).

1. Reformat dense methods/templates in owner-scoped commits. Move multi-step inline
   command bodies into named methods of the same owner where that exposes ordering.
   Keep existing typed stateless views and pure layout/attention functions.
2. Start with measurement/observer subscriptions if they demonstrably obscure the
   workspace lifecycle. Use Lit's `ReactiveController`/host hooks for an extracted
   lifecycle, with browser `ResizeObserver`, font events and AbortController. No
   new controller base class, event bus, duplicate store or extra custom element.
   Account for the first rendered node: connection alone does not mean it exists.
3. Keep command admission, original bindings and uncertain outcomes together. Do not
   split terminal transport, controls and retention into synchronized authorities
   merely to reduce file size. A subcontroller may own subscriptions/resources while
   the existing component remains their policy owner; stop if the result is only
   forwarding methods and parameter bags.
4. If a subsequent ordinary read needs a reusable async abstraction, compare a
   bounded `@lit/task` adoption with the existing direct fetch first. Inspect/pin
   the exact package and wire the shared runtime/build/analyzer if selected. Do not
   add it for mutations, streaming terminals or speculative future consumers.
5. Carry source imports, compiler/analyzer discovery, emitted module mapping,
   preview/staging and test callers together. The current flat basename mapping is
   a constraint, not a defect requiring a new bundler. Change it only for an actual
   move; keep `forgejo-payload.json` authoritative and public URLs compatible.

**Exit:** existing page/drawer and project draft tests pass; host/screen/xterm/socket
identity survives reactive updates, selection, splits, resize, Hide and same-target
Refresh. Late callbacks cannot publish/reopen after retirement. Storage/locators,
finite retain/Return, HTTP End and uncertain outcomes are preserved. Disposal detaches;
it does not End. Native navigation/forms remain upstream-owned. No new runtime or
claim of native CLI compatibility follows from this source refactor.

### E. Native and terminal coupling — separate conditional slices

**Reviewed; only cancellation-aware helper admission was justified and implemented.**
The [handoff](implementation-history.md#refactoring-e--cancellation-aware-helper-admission-only)
records the reproduced contention, new regressions, serial gate and passing local
source/race checks. Registry extraction is deferred: moving fields alone would not
improve its atomic logout/OAuth/Stop coordination. Account validation already has one
source; changing embedded/installed packaging and exact-program verification is not
justified by a measured benefit. Neither deferred slice blocks feature work.

Do not make these prerequisites for runner parity, the installer or independent
feature work. Native effects and later paired delivery need their existing scope.

**Web terminal registry:** [server fields](../internal/web/server.go) and
[session operations](../internal/web/terminal_sessions.go) may move into one concrete
internal owner if that hides map/lock details from logout/OAuth/Stop callers without
weakening their coordination. Specify admission, attach, receipt expiry and shutdown
invariants first. Retain the exact ID-keyed protocol, one writer, slot limits,
uncertain reservations and cleanup receipts. No durable session database, generic
terminal/desktop/AI manager or replacement tmux/systemd supervisor.

**Native account validator:** [key operation](../internal/host/project_keys.py) imports
`account_for` from [the terminal program](../internal/host/project_terminal.py), which
[the Go caller](../internal/host/management.go) preloads in full. Consider a small
shared authored identity source for both fixed programs. Preserve native `pwd`,
root-owned marker/ancestor checks, isolated execution and exact-account binding.
The terminal program is also installed by [the image recipe](../project-os/Containerfile):
any extraction must work in both embedded and installed paths without project-writable
imports. If it adds more deployment/module machinery than it removes, retain the
current reuse. Account creation/key mutation/credential dropping remain separate
operations; do not create a user-management framework.

**Helper admission (implemented):** [daemon dispatch](../internal/host/daemon.go)
formerly held one mutex across buffered reads/mutations and creation's readiness wait.
A delayed-Exec contention test reproduced blocked cancelled waiters. Ordinary Go
context/channel admission now preserves serialization while allowing those waiters
to leave, with cancellation rechecked after admission and decoding before dispatch. Existing
[provider gates](../internal/web/provider.go) demonstrate the primitive, and
[filelock](../internal/filelock/filelock.go) already handles cancellation for native
cross-process locks. Do not route host policy through the web package or add a generic
lock manager. Keep runner cross-surface file locking unchanged.

**Exit:** focused Go/race and Python filesystem/protocol/PTY coverage preserves
logout/Stop-versus-open ordering, exact cleanup and account/file safety. Cancellation
must not execute the queued operation or leak admission. Per-project parallelism
requires demonstrated need and an explicit shared-network/account/unit race analysis;
it is not selected by this audit. Marketplace's long operations belong to its planned
native install unit, not a wider version of this request mutex.

### F. Installer and profiles — implement through their feature owners

**Installer:** follow the [selected password-only correction](coreos-installer-plan.md).
Keep `executeDisk`, disk reinspection, the attempt marker and stock
`coreos-installer install`. Refine input/review with Back, correctable validation and
explicit pre-write restart; keep hidden password input/flush safety specialized.
Reuse NetworkManager's editor, native password hashing and the existing
Ignition/Butane and trusted-bundle paths. Do not build a form toolkit, storage engine,
custom network UI or install/recovery daemon. If richer interaction genuinely needs
an established TUI library, review its exact API, maintenance, accessibility and secret
handling before adding it; no package is selected by this document.

Post-boot enrollment is Soda's integration around OpenSSH, not an upstream
enrollment workflow. The [replacement source](coreos-installer.md#import-a-laptop-key-after-local-password-login)
now delegates connection activation/lifetime to systemd and password authentication
to a separate stock sshd configuration. Its temporary listener deliberately excludes
PAM session migration so authenticated children stay in the bounded unit lifetime;
ordinary sshd/PAM policy remains unchanged. Native account-policy, SELinux,
arming/timeout/success closure and actual access still require validation.
In OpenSSH 10.2p1, `PermitRootLogin forced-commands-only` permits **public-key** forced
commands and disables other root authentication; it does not implement password-based
enrollment. `ForceCommand` alone also does not disable forwarding. Do not silently
loosen ordinary SSH policy to make the flow work. Complete keyboard-only installation,
continuation, payload delivery and real separate-client access before claiming success.

**Profiles:** follow [Project OS creation profiles](project-os.md#selected-environment-profiles).
Keep OCI identity, distro RPM/signature mechanisms, existing persistent containers and
shared authored integration. When adding the next complete profile, deliberately
extend build metadata/staging/install/helper image slots and decouple project bases
from the dashboard base. Use one bounded shipped-profile definition with reviewed
Go/browser contract cases, not a schema generator, image registry or runtime plugin
system. Persist original creation identity; installed availability and current OS
observations remain separate. Never backfill or convert retained roots.

**Exit:** feature-owned source checks and applicable exact native creation/access/
persistence or fresh-disk journeys pass. Compiler/package metadata is not that proof.
Neither feature waits for all conditional refactors above.

### G. Keep current guidance readable — small parallel documentation work

Update the leading current-work sections and label superseded instructions, notably
[sodaspaces-plan.md](sodaspaces-plan.md)'s older “Immediate next step” cleanup sequence.
Link completed frontend work and current gaps rather than copying all details. Keep
one current feature order, detailed feature contracts and revision-bound historical
evidence. Avoid mass-moving history/anchors or creating a second status register.
This review does not itself rewrite those historical entries.

## 4. Preserve unfinished features and native ownership

| Selected unfinished work | Integration direction retained; what this plan must not invent |
| --- | --- |
| Sodarunners parity | Existing native runner lifecycle and shared protocol, Forgejo registration and scheduling. Keep Cockpit Runners and backing tests until the separately required parity/cutover; Tailnet stays in Cockpit. No Soda CI scheduler or provider-role copy. |
| Rocky/Fedora headless and KDE | Existing Project OS foundation, native package/session mechanisms and exact native investigation. No live distro conversion, separate desktop machine or speculative VM/host-runtime fallback. |
| Desktop transport/Lock | Investigate the selected KDE/private native transport and established browser client under the desktop guide. No homegrown remote-desktop protocol/compositor, second reusable password authority or fake computer-use support. Native Lock/unlock integration and exact package/session compatibility remain open. |
| Services marketplace | Reviewed app recipes over Podman/Quadlet/systemd, Caddy ingress and each app's native accounts/settings. Do not reproduce Vaultwarden, Adminer or Homepage, use SQLite as service-running truth, or build a generic registry/update platform/service supervisor. Preserve the selected catalog and per-app upgrade design work. |
| Issue/PR AI automation | Forgejo Actions/workflow/secret/result authority, native Git publication and actual isolated command execution. Do not reproduce Actions or substitute a new scheduler. The documented task-token/runner isolation gap still needs exact upstream investigation; a rootless-engine label or GitHub-style `permissions:` stanza is not proof. |
| Credentials/onboarding | Real native account provisioning and supported explicit public-key selection. Personal outbound-Git consent, scope, at-rest trust and host-key verification remain decisions before automated registration; no borrowed grants or duplicate Git permission engine. |
| Installer/media and validation | Existing CoreOS tools, verified Soda payload and native setup/continuation. Payload inclusion and keyboard-only setup now have replacement source/local checks; rebuilt media, full first install, prepared QCOW2, provider/CLI/client and independent aarch64 proof remain work. No release/update platform or manufactured acceptance. |

These features are preserved, not silently deprioritized because refactoring is easier.
The [Services/AI guide](services-and-ai-plan.md), [runner guide](runners-port.md),
[Project OS](project-os.md) and [native validation](native-validation.md) own their
actual remaining implementation and proof. This review does not claim to have freshly
validated every prospective desktop/app/provider integration.

## 5. Order, checks and stop conditions

Use the [remaining audit sequence](#7-audit-remediation-implementation-plan) for
current maintenance. A, B and the bounded C extraction now have local evidence;
do not repeat them as unfinished work. D and optional E extractions follow the
remaining correctness fixes. F proceeds through its selected feature owner.

- Source changes use focused existing tests first, then the affected combined checks
  and strict TypeScript/Lit. Security/concurrency changes include Go race checks.
- Public source moves preserve exact build/import/payload/license and template
  compatibility tests. Source assertions guarding native commands/permissions are
  not “brittle tests” to delete. Replace only incidental implementation assertions
  with stronger behavior coverage when a real refactor exposes them.
- Native/helper/program changes require their applicable build and paired compatibility
  proof before delivery. Preserve current credentials, roots, files and evidence;
  follow [same-root maintenance](project-os.md#deliver-required-additions-without-replacing-roots)
  rather than replacing projects or replaying first-install.
- Stop an extraction that creates synchronized state, a generalized platform,
  weaker authority, automatic replay, arbitrary commands/URLs or an upstream fork.
  Explain the concrete problem and revise the recommendation instead.
- Measure reduced duplicated preparation and clearer change scope. Do not impose
  file-size/class-count targets or forecast unmeasured productivity gains.

Standing local implementation/testing scope remains as recorded in the handoff.
This document grants no new deployment, disk-write, provider, host-network, retained
project or cleanup action, and adds no repeated-permission requirement for already
authorized ordinary local tests. Source-ready, native-validated and delivered remain
different claims.

## 6. Research provenance and references

The audit owns [research evidence and limits](upstream-ownership-audit.md#evidence-and-limits),
including [the earlier review's provenance](upstream-ownership-audit.md#earlier-upstream-review-provenance).
The handoff owns [actual implementation checks](implementation-status.md).
Research artifacts are optional evidence, never build prerequisites or acceptance.

## 7. Audit remediation implementation plan

**Planning baseline: `1f1a9d3`, 10 September 2026.** This section implements the
follow-up to the [ownership audit](upstream-ownership-audit.md), not a second product
roadmap. It specifies future source changes and checks; writing the plan does not
execute those changes or authorize deployment. Keep each behavior change in a
separate reviewable commit. Complete one implementation slice and its checks before
starting another; a focused independent review may run alongside it.

### Current status and order

| Phase | Work | Current state / dependency |
| --- | --- | --- |
| 0 | Correct stale audit status and macOS test fixture | Source fixed; normal/symlink-path Linux checks passed. Native macOS rerun remains pending. |
| 1 | Remove unused bootstrap-token retention/access | Source complete with local checks; retained credential maintenance and native/provider proof remain unrun. |
| 2 | Reject stale-session project mutations | Source complete with local handler/race/source checks; native delivery remains separate. |
| 3 | Make the managed-key writer contract safe and explicit | Cooperative-writer rule accepted; source complete with local checks. Paired native delivery/SSH proof remain unrun. |
| 4 | Retired GitHub runner recommendation | Superseded by the user-selected removal of GitHub runner support. |
| 5 | Cancellable host admission and bounded capture | Admission completed locally in `22f5c20` (same work as E); capture and Tailnet stream bounds remain open. |
| 6 | Combined regression checks and scoped native delivery | After the relevant fixes; native target/effect authorization remains separate. |
| 7 | Optional upstream reuse/dead-path cleanup | Later; only candidates with demonstrated benefit and equivalent behavior. |

Database completeness is already source-fixed in `bd74b5f`. Aggregate preparation,
fresh Go HTML fixtures and local browser invocation were addressed in `4e5c4ff`,
`e249c10` and `6e3b6e0`. Preserve those implementations and their regression tests.
The session/page extraction in `4b7fc3b` is complete, but did not add missing checks
to Join, Start/Stop or key Apply. These source facts do not establish appliance
delivery or native acceptance. The x86 ISO build and full install remain deferred.

### Phase 0 — reliable baseline

**Source fixed; native macOS rerun pending.** The
[handoff](implementation-history.md#phase-0--physical-temporary-fixture-path)
records the reproduced symlink-path failure and passing local checks.

**Owner:** [source-command tests](../tests/build/test_source_checks.py).
Normalize the temporary fixture root with `Path(...).resolve()` before comparing it
with the child process's physical working directory. Do not weaken the cwd assertion
or alter the real source-check command to accommodate the fixture.

**Check/exit:** all three command-double tests pass on macOS and Linux, including
each suite's fail-fast behavior and the preserved native gates. The preceding review
reproduced one macOS failure because `/var/...` resolves to `/private/var/...`;
this is a test portability defect, not a failed database or installer operation.

### Phase 1 — retire the unused bootstrap token

**Source complete, locally checked.** See the
[handoff](implementation-history.md#phase-1--retire-bootstrap-token-retention)
for regressions, scope research and executed checks. The separate
[existing-install maintenance recipe](dashboard-credentials.md#retired-bootstrap-token--existing-install-maintenance)
is authored, not executed; no installed credential has been remediated by this work.

**Finding:** [unused credential retention](upstream-ownership-audit.md#remove-unused-bootstrap-credential-retention).

**Owners:** [setup command](../cmd/soda-setup/main.go),
[configuration](../internal/config/config.go), [activation](../appliance/bin/soda-activate),
[operator guide](operator-setup.md) and their config/setup/activation/installed checks.

1. Keep reading the operator-supplied token through its private input file for the
   existing native setup calls. Stop copying it into the installed `admin-token` file.
   Retain OAuth client secret and grant-key generation, operator eligibility and
   exclusive publication/refusal after uncertain provider setup.
2. Omit `admin_token_file` from new output. Accept the legacy field for parsing
   compatibility without requiring, reading or using its path. Do not use a legacy
   configuration value as authority to chmod, delete or revoke a credential.
3. Remove that file from activation's service-readable credential list and update
   all checks that currently require it. Narrow bootstrap token-scope guidance only
   after inspecting the selected Forgejo endpoints and exercising synthetic API
   fixtures. Do not change the configured Soda-operator authority.
4. Author a separate existing-install maintenance recipe: identify the exact old
   canonical file and current permissions, preserve it, and remove unnecessary
   service-group access under an authorized target action. Token revocation/deletion
   remains an explicit operator/provider action, not an automatic migration.

**Check/exit:** new setup emits no copy/reference; old configs still load when the
retired file is missing/unreadable; activation touches only required files. Tests
assert that token bytes never appear in generated configuration, argv, errors or
logs. Existing setup failures retain their uncertain external outcome. Source-ready
does not mean already installed tokens have been remediated.

### Phase 2 — finish session checks at mutation admission

**Source complete, locally checked.** The existing helper now gates Join, Start/Stop
and key Apply at their first-effect boundaries. See the
[handoff](implementation-history.md#phase-2--fresh-session-mutation-admission)
for reproduced failures, actual checks and the explicit admission—not rollback—limit.

**Finding:** [stale-session mutation admission](upstream-ownership-audit.md#recheck-the-original-session-before-native-mutation).

**Owners:** [Join](../internal/web/environments_api.go),
[lifecycle/key handlers](../internal/web/management.go), existing
[session helper](../internal/web/api.go) and real-handler web tests.

Use `requireCurrentSession` after request decoding, provider authorization and
relevant state reads, immediately before the first effect: account provisioning for
Join, terminal cancellation/Stop admission for Stop, native Start, and native key
Apply. Preserve exact cookie disambiguation, stable user/context/CSRF comparison,
original membership/login and current saved-key confirmation. Failure must not
provision, cancel terminals, dispatch the helper or record a successful membership.

The contract is a fresh admission decision: logout/rotation completed before this
final check denies the action. It does not undo an admitted native operation or
promise atomic cancellation of a remote command after dispatch. Keep the stronger
terminal admission/logout synchronization intact; no global lock over provider I/O,
session cache or new operation scheduler.

**Check/exit:** pause the actual provider path, complete real Soda logout or rotate
the context, then release the request. Join, Start, Stop and Apply each deny with
zero first-effect calls. Include changed actor/CSRF, store failure/cancellation,
provider denial, unchanged-session success and Stop preserving live terminals on
denial. Run the affected handler suite and Go race checks. The helper extraction
alone is not completion of this phase.

### Phase 3 — managed-key concurrency contract

**Approved and source-complete, locally checked.** The user accepted the cooperative
writer rule and requested the bounded fix, not a new subsystem. See the
[handoff](implementation-history.md#phase-3--cooperative-managed-key-writers)
for actual checks and the explicit advisory-lock/native-delivery limits.

**Finding:** [key concurrency and durability](upstream-ownership-audit.md#correct-project-key-concurrency-before-claiming-compare-and-swap).

**Owners:** [key program](../internal/host/project_keys.py),
[native caller](../internal/host/management.go),
[filesystem tests](../tests/build/test_project_keys.py),
[account provisioning](../project-os/rootfs/usr/libexec/soda/project-account) and
[Project OS access guidance](project-os.md#access-credentials-and-connectivity).

**Accepted decision:** use one cooperative writer contract for Soda's
root-owned managed key directory. All Soda writers and native administrators editing
these managed files must hold the same directory `flock`, acquired before opening
the file and held through the complete update. The directory itself remains stable.
This uses the existing native lock and OpenSSH file format; it introduces no key
database, SSH broker or `AuthorizedKeysCommand` service.

The native-writer requirement is now accepted. Standard check-then-rename does not
provide expected-inode or content compare-and-swap against arbitrary root editors. A post-write reread or an
extra hash cannot close that gap. If unrestricted simultaneous edits remain required,
stop this approach and revisit ownership with the user; do not mark the finding fixed
by merely changing the wording or preserving a backup after an overwritten revocation.

Retain whole-set replacement and empty-set revocation:
under the shared lock, validate the original account/directory/file, compare the
preview revision, write, explicitly flush and then sync the exclusive temporary
file, atomically publish,
sync the directory and verify the result. Every managed-key writer must follow the
same contract. Keep content, inode/path safety and no automatic replay/restoration
checks; refuse stale previews and busy writers and preserve uncertainty without
exposing keys in diagnostics. Normal SSH reads do not need to participate in the lock.

The reviewed sync-before-flush and unlocked account-provisioning defects are now
corrected. Both writers hold the directory lock before account/key observations and
effects, retaining exclusive missing-file creation and repeat-Join refusal on drift.
The [API contract](dashboard-api.md) and code now specify content hashes rather than
history/generation counters. Existing conservative public failure responses are
unchanged; the native key program uses the same single sanitized failure.

**Check/exit:** deterministic separate-process fixtures cover competing Soda/native
cooperating writers, stale preview, lock contention, append/edit/replacement before
admission, empty-set revocation, mode/symlink/hardlink refusal, write/rename/fsync
failure and post-publication uncertainty. Preserve later cooperating writes and
clean only the exact unpublished temporary file. Include an explicit demonstration
that a noncooperating root writer bypasses the advisory contract; do not turn it into
a passing universal-safety claim. Existing-root delivery must preserve accounts,
key bytes and unrelated SSH sessions under the same-root maintenance rules.
Extend [account tests](../tests/build/test_project_account.py) for lock refusal
before account commands. Use event barriers instead of timing sleeps, and preserve
native nanosecond stat fields when constructing metadata-sensitive test fixtures.

### Phase 4 — GitHub runner service compatibility

**Retired:** the user selected removal of GitHub runner support throughout Soda.
The registration, launch, UI and packaged client paths have been removed; this
recommendation is no longer implementation work. Forgejo's daemon and local
lifecycle remain. Historical findings are preserved in Git at `216db47`.

### Phase 5 — cancellation and capture bounds

**Findings:** [host admission/capture](upstream-ownership-audit.md#make-helper-admission-cancellable-and-bound-command-capture)
and [Tailnet framing](upstream-ownership-audit.md#review-tailnets-unstable-api-seam-and-native-ui-overlap).

**5a, host admission — completed locally:** this is the same work as
[slice E](#e-native-and-terminal-coupling--separate-conditional-slices), not another
implementation phase. `22f5c20` supplies cancellation-aware serial admission and
its regression/race checks; preserve those tests. Native delivery is not implied.

**5b, native capture:** inspect the actual output needs of host/native runner/pkexec
and [Tailnet CLI](../internal/tailnet/tailnet.go) callers. Reuse native filtered
output where available; cap bytes while collecting, not after allocating an entire
response. Specify per-command stdout/stderr limits from the current protocol and
fixtures before implementation. Keep status JSON complete-or-error and errors
sanitized; never parse a truncated buffer as success.

Define overflow behavior with each command's effects. Status readers can cancel
their owned read command. Mutating commands should retain at most the bound, drain
excess under a finite command deadline and report an unconfirmed result without
replay or rollback. Preserve existing owned-child cleanup; timeout/pipe closure is
not proof that arbitrary descendants or remote mutations stopped. Test oversized
stdout/stderr, exact boundary sizes, nonzero exit, cancellation, cleanup and secret
redaction using synthetic children. Avoid a general command framework.

**5c, Tailnet stream:** bound the partial-object accumulator in the existing
[authentication framer](../cockpit/src/tailscale/stream.ts). Keep native `JSON.parse`,
split-object/string-escape behavior and native preference preservation. Test one
oversized unterminated object, many valid objects, malformed/truncated input,
multibyte accounting and cancellation. Error/cancel must not reset native preferences
or start a second sign-in attempt.

### Phase 6 — integration, evidence and delivery

After each behavioral commit, run its focused existing tests; use race checks for
changed concurrency/security paths. After the required fixes are integrated, run
the existing combined source command with the documented tool prerequisites and
record failures/skips honestly. Preserve schema-v8 and three-page browser regression
coverage; no new all-purpose test framework or duplicate acceptance gate.

Keep **source-ready**, **native-validated** and **delivered** separate. New-install
token behavior, key permissions/locking and runner signal behavior require native
proof on an authorized fixture. Existing appliances need a fresh backup and a
reviewed affected-component/same-root recipe. Do not replay activation/first-install,
delete old credentials, replace project roots or mutate provider registrations.
Current target/action grants must be checked at that time. The x86 ISO build and
complete first-install journey remain deferred until the machine is available;
this plan does not reactivate them or authorize publication.

### Overengineering review follow-through

The user requested fixes for the [three-pass review](overengineering-review.md).
Implement its supported removal boundaries, not the withdrawn inventory/hash deletion
or an evidence-system redesign. One bounded change per commit, with existing focused
checks followed by the aggregate source check; no deployment or retained-state work.

- Managed keys: remove unused error classes/diagnostic distinctions; keep writer
  exclusion, durability and preservation tests. Source simplified.
- Tailnet: remove test-only classification and generic process configuration; keep
  the real status/endpoint callers and their cancellation/error behavior. Source
  complete with focused local checks; no replacement process abstraction.
- Runners: connect the public CLI directly to native operations, preserving the
  combined root/`PKEXEC_UID`/native identity admission and cross-process lock. Retire
  the helper source and packaging together, not installed files. Source complete;
  focused runner/CLI/identity/host race checks passed. Native-stage proof is unrun.
- Templates: read authored source in behavior tests; make historical normalization
  explicit in parity checks. Source complete with local Go checks; preserved hashes
  and permission fixtures, removed reconstruction entries with no parity caller.
- Terminal: explicitly retire no-locator mounting and port useful tests to managed
  locators. Retain legacy storage import and live pending events; this is a source
  interface change, not proof of compatibility with unknown external consumers.
  Source complete; ported terminal and shared workspace/Lit tests passed locally.
- Support labels and broad presentation snapshots: no demonstrated defect requiring
  replacement; retain their documented contracts and unresolved evidence reporting.

Actual checks and remaining native limits belong in the
[handoff](implementation-history.md#overengineering-review-follow-through).

### Phase 7 — optional cleanup after correctness

The supported review removals above are source-complete. Other candidates remain
conditional: native GCM nonce packing (prove old/new ciphertext compatibility),
narrower OCI parsing through a
verified upstream reader, or Tailnet native UI reuse. Adopt only when the existing
authority, persistence, file-size/provenance and complete-user-journey contracts
remain intact. Preserve Cockpit until its replacement is validated. These candidates
do not block the required fixes, installer native proof or ordinary product work.

**Completion:** phases 0–5 have focused regression evidence and the accepted
key-writer contract; phase 6 records the actual native/delivery outcome separately.
No open high-priority item is relabelled complete because a helper exists, docs
changed, optional cleanup was skipped or a source test passed.
