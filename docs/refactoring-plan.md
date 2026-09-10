# Upstream-first refactoring review and implementation plan

Reviewed on 10 September 2026 against `c20abc3ec35bf416c4f55154373da7f29a3f2e28`.
**Status: reviewed recommendations, not implemented refactors or new runtime proof.**
The user requested a maintainability audit, preservation of unfinished features,
and an upstream-reuse review before turning the findings into an implementation plan.
This document records that review and **narrows the initial audit** where it was
premature or overlooked an existing mechanism.

## Decision

Keep the selected architecture. Improve Soda's integration and development feedback;
do not build smaller replacements for Forgejo, Lit, tmux, systemd, OpenSSH, Podman,
CoreOS Installer or the existing test runners.

**Do first:** reliable invocation of existing integration tests, one browser build
per aggregate test run, complete schema-v8 startup checks, and readable current
instructions. Then make small, feature-aligned changes to existing owners. Broad
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

| Area | Established mechanism and source reviewed | Soda's legitimate remainder / decision |
| --- | --- | --- |
| Forgejo pages, identity, permissions, Git and Actions | Selected 15.0.7 template lookup, native navbar/settings templates and own-user key handlers; official customization | Preserve native workflows, including administrator views. Use hooks/necessary reviewed overrides and supported acting-user APIs. Soda pages authorize only Soda features; no copied forge backend, borrowed sessions or downstream executable patch. |
| UI lifecycle and rendering | Installed Lit 3.3.3 / reactive-element 2.1.2 controller interfaces and implementation; official controller documentation | Lit already supplies controller registration, lifecycle callbacks and reactive updates. Use those if a subcomponent lifecycle really needs extraction; do not invent a parallel controller/event framework. Existing typed views and pure layout functions remain useful. |
| Async request state | Browser fetch/AbortController and official `@lit/task` documentation | Task already offers pending/result/error, latest-result handling and cancellation signalling, including manual mode. Do not author a generic Task clone. Keep current direct requests unless a concrete read-only slice benefits; Task is not currently a dependency, and no dependency addition is selected here. |
| Browser terminal | Locked xterm renderer; retained upstream tmux 3.2a attach/no-start source; Soda's current terminal program | Xterm owns terminal rendering; tmux owns live shell/screen/history. Soda binds exact IDs, actor/account, access deadlines and cleanup receipts. Do not implement an emulator, multiplexer or new public terminal server. |
| Native service lifetime | systemd service/kill documentation; existing `project_terminal.py` uses transient services, `KillMode=control-group`, watchdog, runtime cap and `Restart=no` | The native supervision delegation is already implemented. Refactoring the web registry must not duplicate process supervision or remove the Soda access lease. Host, project, terminal and workload scopes remain distinct. |
| Host installation | CoreOS Installer 0.26.0 CLI source, ISO/install documentation, existing `executeDisk`, Ignition/Butane and NetworkManager `nmtui` caller | Improve Soda's input/review/continuation flow, not the disk writer, partitioner, network editor or provisioning language. The irreversible execution boundary already exists. |
| Accounts and remote access | Python `pwd`/OS APIs and existing account/key scripts; OpenSSH 10.2p1 configuration documentation | Native accounts, authentication and SSH protocol stay native. Soda retains original-account association and bounded enrollment policy. No custom SSH server, password verifier, identity remapper or credential broker. |
| Database | Existing SQLite driver/transactions/migrations; official integrity and foreign-key pragmas | SQLite validates its own storage and existing constraints, not which application columns/triggers Soda intended to ship. A small required-schema check is warranted; a new ORM, migration platform or schema-repair engine is not. |
| Tests and builds | Existing Go tests, Bun package scripts/test runner, Playwright fixtures and production payload inventory; official Bun run documentation | Fix invocation and producer/consumer ordering. Do not create another test runner, scenario DSL, readiness service, build DAG/cache engine or parallel payload registry. |
| Runners, services and profiles | Existing runner/native code and feature guides; Podman/Quadlet/systemd and distro package mechanisms | Finish the selected integrations over their native owners. Do not become a CI scheduler, package manager, container runtime, app-authentication backend or whole-host update platform. New capabilities still require exact-version investigation in their feature slice. |

**Important limitations:** a Lit task can run manually; rejecting it on the claim that
it must auto-run would be wrong. However, cancelling a request does not undo a native
mutation, and latest-result handling does not resolve uncertain Create/End outcomes.
Do not put terminal creation, key changes, lifecycle or credential operations into an
automatically rerun task. Task documentation itself distinguishes request/response
work from open-ended streams. Existing generation and exact-target checks are not
all redundant promise boilerplate.

Similarly, systemd's watchdog supervises the guard's health. It does not know whether
a Soda actor still has access. Tmux attachment does not supply web authorization.
Keep the small integration that connects these independent native mechanisms.

## 2. Disposition of the original findings

Numbers refer to the original audit, not new product milestones.

| Finding | Review outcome | Recommended disposition |
| --- | --- | --- |
| 1. Conditional integration tests | Confirmed invocation gap; tests already exist and have historical explicit runs | Fix ordinary suite wiring, not rewrite scenarios. |
| 2. Repeated browser builds | Confirmed: root `test` reaches `build:forgejo` four times | Prepare once and invoke existing suites. Timing benefit is unmeasured. |
| 3. Schema completeness | Confirmed: v8 columns are absent from startup completeness queries | Small startup-validation fix and focused fixtures; no evidence of damaged retained databases. |
| 4. Current versus historical docs | Confirmed overlapping/stale next-step instructions | Repair the current index and label history, preserving evidence and links. |
| 5. Frontend decomposition | Dense coordination is real, but the initial prescription to split several resource owners was too broad | Format first; use existing Lit composition/lifecycle. Extract only one demonstrated independent responsibility at a time. No forced relocation of xterm/command authority. |
| 6. Session/page plumbing | Repeated mechanics exist; `apiProtected`, `visibleRepository`, `userGrant`, `authorizeOperator` and specialized terminal checks already exist | Reuse those owners; extract only repeated current-context/page mechanics. OAuth destination redesign waits for an actual new destination. |
| 7. Backend terminal registry | State and lock responsibilities span callers; existing registry already implements the feature | Conditional encapsulation within `internal/web`, not a replacement session service. Preserve admission/logout/Stop atomicity. |
| 8. Python identity dependency | Keys preload the entire terminal program to use `account_for` | Small shared-source extraction only if delivery remains simpler and equally confined; not a new deployed Python framework. |
| 9. Global helper mutex | Cross-project blocking and uncancellable mutex waiting follow from source; production latency has not been measured | Characterize and fix cancellation first. Do not prescribe per-project parallelism or a queue yet. |
| 10. Installer split | Input/validation correction is needed, but `executeDisk` already separates native execution | Reuse it and implement the selected input/review flow. Withdraw the implication that an installation engine is missing. |
| 11. Profile expansion | Real one-image/base coupling, already documented by the Project OS owner | Complete it with the next profile, not a speculative runtime selector or second profile plan. |

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
[revision-specific results and failures](implementation-status.md#refactoring-step-1--local-source-check-wiring).
This does not complete native-stage or installed acceptance, or the later slices.

**Owners:** [root scripts](../package.json), [Spaces fixture orchestrator](../scripts/test-spaces-page.ts),
[frontend tests](../tests/frontend/), [Forgejo tests](../tests/forgejo/),
[native check](../scripts/check-native.sh).

1. Extend/rename the existing small page-fixture orchestrator, updating its callers,
   rather than writing separate runner/settings harnesses. Run the existing Go
   producers with `-count=1` into a fresh retained artifact directory:
   - `TestSpacesHTMLSessionAuthorityAndBoundedException` → `SODA_SPACES_PAGE_HTML`.
   - `TestRunnerOperatorGatesBeforeNativeAndDecode` → `SODA_RUNNERS_PAGE_HTML`.
   - `TestRepositorySettingsUsesFreshStableIdentityAndSharedControls` →
     `SODA_REPOSITORY_SETTINGS_HTML`.
2. Check that all requested fixture files were produced before running their browser
   consumers. Missing production HTML/CSP must fail this integration command rather
   than become a skip or handwritten replacement fixture.
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
[handoff](implementation-status.md#refactoring-step-b--schema-v8-completeness)
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

**Helper admission:** [daemon dispatch](../internal/host/daemon.go) holds one mutex
across buffered reads/mutations and creation's readiness wait. Author a delayed-Exec
contention test before changing it. Use ordinary Go context/channel admission if
needed to preserve serialization while allowing cancelled waiters to leave; recheck
cancellation after admission and before native effects. Existing
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

Post-boot enrollment is **not** already implemented by OpenSSH. Use native SSH/PAM
and restricted command configuration for its transport/authentication rather than
inventing a password server. Resolve the exact listener/configuration and local
arming/timeout/success closure through the feature review before implementation.
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
| Sodarunners parity | Existing native runner lifecycle and shared protocol, Forgejo/GitHub registration and scheduling. Keep Cockpit Runners and backing tests until the separately required parity/cutover; Tailnet stays in Cockpit. No Soda CI scheduler or provider-role copy. |
| Rocky/Fedora headless and KDE | Existing Project OS foundation, native package/session mechanisms and exact native investigation. No live distro conversion, separate desktop machine or speculative VM/host-runtime fallback. |
| Desktop transport/Lock | Investigate the selected KDE/private native transport and established browser client under the desktop guide. No homegrown remote-desktop protocol/compositor, second reusable password authority or fake computer-use support. Native Lock/unlock integration and exact package/session compatibility remain open. |
| Services marketplace | Reviewed app recipes over Podman/Quadlet/systemd, Caddy ingress and each app's native accounts/settings. Do not reproduce Vaultwarden, Adminer or Homepage, use SQLite as service-running truth, or build a generic registry/update platform/service supervisor. Preserve the selected catalog and per-app upgrade design work. |
| Issue/PR AI automation | Forgejo Actions/workflow/secret/result authority, native Git publication and actual isolated command execution. Do not reproduce Actions or substitute a new scheduler. The documented task-token/runner isolation gap still needs exact upstream investigation; a rootless-engine label or GitHub-style `permissions:` stanza is not proof. |
| Credentials/onboarding | Real native account provisioning and supported explicit public-key selection. Personal outbound-Git consent, scope, at-rest trust and host-key verification remain decisions before automated registration; no borrowed grants or duplicate Git permission engine. |
| Installer/media and validation | Existing CoreOS tools, verified Soda payload and native setup/continuation. ISO payload inclusion, prepared QCOW2, keyboard-only first install, provider/CLI/client and independent aarch64 proof remain work. No release/update platform or manufactured acceptance. |

These features are preserved, not silently deprioritized because refactoring is easier.
The [Services/AI guide](services-and-ai-plan.md), [runner guide](runners-port.md),
[Project OS](project-os.md) and [native validation](native-validation.md) own their
actual remaining implementation and proof. This review does not claim to have freshly
validated every prospective desktop/app/provider integration.

## 5. Order, checks and stop conditions

Recommended maintenance order: **A and B first; G in parallel; C next; D one small
slice at a time.** E is conditional, split by boundary. F proceeds with the selected
feature work, not after a repository-wide cleanup barrier.

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

This review inspected source/configuration and existing tests; it did **not** run
product builds/tests, install dependencies, execute downloaded code, access retained
VMs or change provider/project state. Public research responses, source hashes and
failed retrievals are retained under `.artifacts/refactoring-upstream-review-GVDpXf/`.
That ignored directory is optional evidence, not a build prerequisite.

Fresh Forgejo 15.0.7 `modules/templates/base.go` and CoreOS Installer 0.26.0
`src/cmdline/install.rs` downloads matched the retained source bytes. Other Forgejo
navbar/key and tmux findings used the already retained selected-version source;
Lit controller findings used the installed locked package source. Current web docs
are guidance, not exact-version/native proof. The reviewed systemd v259 documentation
and OpenSSH 10.2p1 source align with the selected CoreOS metadata's 259.8/10.2p1 bases;
this is not a survey of retained project RPMs or proof of downstream PAM/configuration.
An initial Bun `.mdx` URL returned 404; the official run page succeeded. No capability
was declared missing on the basis of that failed URL.

Primary references:

- Forgejo [official customization](https://forgejo.org/docs/v15.0/admin/advanced/customization/),
  [15.0.7 template lookup](https://codeberg.org/forgejo/forgejo/src/tag/v15.0.7/modules/templates/base.go),
  [repository settings navigation](https://codeberg.org/forgejo/forgejo/src/tag/v15.0.7/templates/repo/settings/navbar.tmpl),
  [own-user keys](https://codeberg.org/forgejo/forgejo/src/tag/v15.0.7/routers/api/v1/user/key.go).
- Lit [reactive controllers](https://lit.dev/docs/composition/controllers/),
  [tasks/manual execution/cancellation](https://lit.dev/docs/data/task/),
  and repository [runtime/build rules](lit.md). Installed declarations and host
  implementation are in locked `@lit/reactive-element` 2.1.2, not a new dependency.
- Tmux [3.2a manual](https://github.com/tmux/tmux/blob/3.2a/tmux.1) and
  [client no-start behavior](https://github.com/tmux/tmux/blob/3.2a/client.c);
  Soda's [terminal contract](terminal-integration.md).
- Systemd v259 [service watchdog/runtime behavior](https://github.com/systemd/systemd/blob/v259/man/systemd.service.xml)
  and [control-group cleanup](https://github.com/systemd/systemd/blob/v259/man/systemd.kill.xml).
- CoreOS Installer [0.26.0 CLI](https://github.com/coreos/coreos-installer/blob/v0.26.0/src/cmdline/install.rs),
  [install interface](https://coreos.github.io/coreos-installer/cmd/install/),
  [ISO customization](https://coreos.github.io/coreos-installer/cmd/iso/).
- OpenSSH [10.2p1 server configuration](https://github.com/openssh/openssh-portable/blob/V_10_2_P1/sshd_config.5).
- SQLite [integrity checking](https://www.sqlite.org/pragma.html#pragma_integrity_check)
  and [foreign-key checking](https://www.sqlite.org/pragma.html#pragma_foreign_key_check).
- Bun [script execution](https://bun.com/docs/cli/run), Go [context cancellation](https://pkg.go.dev/context),
  and existing [local tooling boundaries](typescript.md).
