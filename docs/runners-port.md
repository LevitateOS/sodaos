# Local CI runners

**Current source:** local CI runners support only Forgejo. The user selected
removal of GitHub runner registration, execution and packaging. Unsupported saved
descriptors are rejected without changing their accounts, files or credentials;
this source change performs no installed cleanup.

**Migration status:** the dashboard
replacement is implemented in source and has recorded local test coverage. It is
not yet a delivered, provider-validated replacement for Cockpit. Finish the
[implementation and completion plan](#implementation-and-completion-gate) below;
do not rebuild the existing page, API or native bridge.

## Selected product boundary

Move Soda's local runner capacity and service controls to **Global SodaOS
settings → Sodarunners**. The replacement is a Soda-owned Go page and Lit
component under the existing `/-/soda/` namespace, reached through supported
Forgejo template customization. It is not a Forgejo administrator page, a
repository drawer, or a revival of the removed standalone dashboard. The
[settings plan](sodaspaces-plan.md#settings-pages-and-os-selection) owns its
placement.

Only the stable Forgejo user ID recorded as `operator_id` during Soda setup may
read the page, inspect capacity, or mutate a runner. A Forgejo site, organization,
or repository administrator does not gain that appliance authority. Conversely,
the configured Soda operator need not be a Forgejo site administrator to inspect
and control existing local services. Provider registration can still require a
separate authorized provider administrator and credential.

Keep native repository/user/organization/administrator **Actions → Runners**,
secrets, variables, workflows, jobs, and results with Forgejo. Soda owns only this appliance's runner
accounts, installed clients, one-slot listeners, local state, and local service
lifecycle. Repository AI settings may select supported labels and report known
local availability, but they do not create host accounts, register capacity, or
become a scheduler.

Keep the Cockpit Runners page, its current protocol, backing logic, packaging,
dependencies, and focused tests until the new surface has passed the completion
gate below and its removal is coordinated. Tailnet stays in Cockpit.

## Selected Runner OS direction

Soda selects a dedicated **Rocky headless Runner OS** for future isolated jobs.
It shares the reviewed Rocky-family foundation and conventions with Project OS,
but it is a separate job artifact: it does not reuse a Project OS image, inherit
its persistent account/home/SSH/tmux contracts, or turn a development environment
into CI capacity. Ubuntu and Arch are not selected.

`mise` is required in the Runner OS. A repository may use the same ordinary
`mise.toml` that developers use, after the workflow makes the applicable trust
decision, but CI has its own installation and cache locations. It never borrows a
developer's `/opt/mise`, home, dotfiles, provider login, package-manager state or
credentials. Repository-selected language versions belong to mise; mise does not
replace the native compiler, headers, libraries and utilities the image must supply.

The intended execution model creates a fresh disposable job environment for each
job. Steps in that job share its checkout and workspace. Provider results and an
explicit cache/artifact mechanism may outlive the environment; arbitrary job
filesystem state does not. Planned cleanup is limited to the exact job-owned
disposable resources. This design is not authority to erase retained fixtures,
evidence, runner registration/state or unrelated caches. Production lifecycle work
must prove cleanup after success, failure and cancellation, including interrupted
or partially created environments.

The native runner agent remains a long-lived, dedicated service account and
listener. It launches provider-assigned job containers; it does not itself become the
job image. A workflow may request explicitly supported service containers alongside
the job on a job-owned network. That does not automatically make them a Podman pod,
permit a nested container engine, or expose the host/rootless engine socket inside
the command job. Exact network, volume, cancellation and cleanup behavior requires
native proof for the selected Forgejo runner and Podman versions. The current source
still executes Forgejo jobs directly on host runner accounts. The
dedicated rootless OCI candidate and its unresolved credential/publication boundary
remain owned by the [services and AI plan](services-and-ai-plan.md#trust-credentials-and-the-concrete-runtime-gap).

The recommended first image baseline, still subject to an accepted package inventory
and native validation, has five groups:

1. Shell and workflow utilities: Bash/coreutils, Git, OpenSSH client, CA trust,
   curl, find/process tools, `file`, patch/diff tools and `jq`.
2. Action runtimes: Node.js versions required by the supported JavaScript
   actions, plus OS Python for compatible workflow tooling. Repository language
   versions remain mise-owned.
3. Archive, artifact and cache tools: `tar`, gzip, zip/unzip, xz, bzip2 and zstd.
4. Native build prerequisites: GCC/G++, libc/C++ headers, Make, CMake, Ninja,
   pkg-config, binutils and the selected common development libraries.
5. The pinned, architecture-matched mise binary and its explicit CI directories.

That baseline is a recommendation, not an accepted exhaustive inventory or runtime
proof. Browser, mobile, container-build and cloud-provider workflows each need a
selected, tested capability and may use a different image or label. Do not grow one
generic image to cover every possible workload. Use
Forgejo's [action runtime requirements](https://forgejo.org/docs/latest/admin/actions/configuration/)
and the [mise CI guide](https://mise.jdx.dev/continuous-integration.html). Build and
validate the selected image separately on x86_64 and aarch64; image selection alone
does not establish architecture or CPU compatibility.

The current [Forgejo runner configuration](../internal/runners/native_create.go)
has disabled its cache since the initial port. Inspection of the configuration,
its introducing commit and documentation found no recorded rationale. Disabling
the cache is not a CoreOS requirement. Investigate the selected Forgejo runner's
[native cache mechanism](https://forgejo.org/docs/latest/admin/actions/configuration/#cache-configuration)
before enabling it, starting with archives
stored in runner-owned storage rather than caller-selected host paths. Verify exact
upstream behavior for repository/trust isolation, cache keys, version and native
architecture scoping; do not claim those boundaries are automatic or secure without
that evidence. No physical volume per repository is selected. Cache use should be
workflow opt-in and allocated on use. Retention, quotas, eviction, networking,
storage placement and cleanup remain unresolved product decisions. Providers retain
workflow semantics, scheduling and results. Caching follows the selected native
Forgejo runner contract and its actual behavior.

## Current source and observed evidence

### Implemented source inventory

| Area | Existing implementation | Remaining obligation |
| --- | --- | --- |
| Operator page | [`settings_page.go`](../internal/web/settings_page.go), [`runners.html`](../internal/web/templates/runners.html), [`soda-runners-page.ts`](../frontend/runners/soda-runners-page.ts): separate Go shell and Lit inventory, registration, lifecycle, provider links, HTML denial, keyboard/responsive presentation and guarded browser restoration, with local coverage. | Inspect the delivered page; this source does not embed the page in Forgejo's native shell. |
| Navigation | [`extra_links.tmpl`](../appliance/forgejo/templates/custom/extra_links.tmpl), [`footer.tmpl`](../appliance/forgejo/templates/custom/footer.tmpl), [`soda-settings-link.ts`](../assets/branding/forgejo/soda-settings-link.ts), and the Spaces page. | Prove discoverability with native Forgejo sessions, including a configured operator without site-admin rights. |
| Web authority and API | [`internal/web/runners.go`](../internal/web/runners.go): fresh provider subject, configured `operator_id`, original Soda session, guarded list/create/lifecycle routes; existing API middleware supplies actor/origin/CSRF checks. | Preserve denial-before-decode/native-read behavior and exercise the actual proxy/OAuth/helper chain. |
| OAuth and persistence | Fixed `destination=runners` return was added in schema v7; current source is schema v8 and preserves that return. | Rehearse the actual target schema to the chosen candidate schema; no new runner inventory database or migration is required merely to finish the UI. |
| Root bridge | [`internal/host/runners.go`](../internal/host/runners.go), wired by [`cmd/soda-host`](../cmd/soda-host/main.go), dispatches six fixed operations to the existing runner implementation. | Deliver compatible binaries together and prove socket/group/service confinement natively. |
| Native runner owner | [`internal/runners/`](../internal/runners/): provider configuration, dedicated accounts/state, one-slot listeners, lifecycle and cross-process file locking for reads and all mutations. | Real provider lifecycle/jobs and retained-state preservation remain unproved. Only Forgejo is supported; unsupported saved providers are refused before native effects. |
| Shared frontend protocol | [`soda-runner-types.d.ts`](../frontend/runners/soda-runner-types.d.ts) and [`soda-runner-response.ts`](../frontend/runners/soda-runner-response.ts) serve Lit and Cockpit. | Keep one response contract while both surfaces coexist; preserve backend/CLI tests after Cockpit removal. |
| Packaging | [`forgejo-payload.json`](../internal/nativebuild/forgejo-payload.json), [`build-forgejo.ts`](../scripts/build-forgejo.ts), [`stage.py`](../scripts/stage.py) include settings assets, native hooks and both Cockpit pages. | Produce and inspect a fresh matching-native export; local emitted assets are not installed proof. |
| Local checks | [`runners_test.go`](../internal/web/runners_test.go), [host bridge tests](../internal/host/runners_test.go), native runner tests, [actual Go HTML/Lit browser tests](../tests/frontend/runners.test.ts), [navigation tests](../tests/forgejo/settings-link.test.ts), and retained Cockpit tests. The ordinary page gate covers the Forgejo form and restored/retired pages using real Go HTML and synthetic peers. | Native/provider effects remain unproved. See the handoff for passing affected checks and broader platform/test-environment failures. |

### Evidence limits

The local source now includes the protected Go/Lit page and fixed root:soda runner
adapter introduced by `f60df0a`. Its local Go/emitted-browser proof is recorded in the
[handoff](implementation-status.md); native/provider parity remains pending and
Cockpit is not removed. The retained native implementation is the Cockpit page plus
`soda-runners` → `internal/runners.Native` and the `soda-runner@.service` launcher.
Cockpit runs the coordinator in its authenticated root session. The CLI receives
strict requests over stdin, requires real/effective root, rejects non-root original
`PKEXEC_UID` and verifies the native root account before any runner operation.
The redundant helper executable/subprocess is retired in source; installed old
files have not been removed. This path does not authorize a Forgejo web user.

Each runner has a stable, validated local ID, a dedicated noninteractive
`soda-runner-<id>` account, a private persistent state directory under
`/var/lib/soda/runners/`, one job slot, and a systemd instance. The service has no
sudo or Linux capabilities, has a read-only host view apart from its own state,
and can access the network. Jobs execute repository code and can change the
runner's persistent work files, so only trusted repositories and contributors
belong on this capacity. Current Forgejo labels select host execution. OCI AI-job isolation is
separate unimplemented work and must not appear because a label was entered.

Current local listing reads the descriptors, systemd `LoadState`, `ActiveState`,
`SubState` and `UnitFileState`, and the installed client version. It reports the
number of descriptors, active/running listener units, and one configured slot per
descriptor. It does **not** query provider online/offline or busy state, queued or
running jobs, available slots, labels, job history, machine load, or disk use. One
unreadable descriptor, unit, or client version currently makes the whole list fail;
the UI must report status as unavailable rather than render an empty inventory.

The original M12 source was later included in the recorded `8b823db` full native
x86_64 build/check, including the Go and 60-test Cockpit suites. The retained U08
operator journey observed the installed Cockpit page with zero runners, listeners,
and capacity. A later retained cutover delivered a strict-config `soda-runners`
binary and its native empty `list` succeeded; the helper and runner services were
unchanged from `8b823db`. No recorded evidence creates a Forgejo runner,
runs a provider-scheduled job, or exercises native runner start, stop, restart, or
removal. The new source supplies the Global SodaOS settings page and web-to-root bridge;
neither has been deployed or exercised with real provider registrations.

## Required web and native authority path

The Go server already authenticates a separate Soda session through Forgejo OAuth,
binds it to a stable provider user ID, exposes whether that ID equals
`operator_id`, checks an expected actor header, and applies origin/CSRF protection
to mutations. The fixed operator-settings OAuth return and protected
`/-/soda/settings/runners` page are implemented; do not accept a caller-supplied return URL. Check
`session.User.ID == Config.OperatorID` server-side before decoding a request or
reading runner state. Apply the same check to every page, API, deep link, refresh,
and mutation. Navigation visibility is only presentation and never authorization.

The dashboard container runs as UID 2000 with no capabilities, a read-only root,
and `NoNewPrivileges`; it cannot safely reuse the Cockpit coordinator by pretending
to be native root or by adding a web-triggered polkit prompt. Preserve the current
root-only `LinuxAuthorizer` for the CLI/Cockpit path. The fixed runner methods in the
existing root:soda Unix-socket service reuse `internal/runners.Native` behind
them. The web handler authorizes the configured Soda operator; the socket ACL admits
the existing root service and dedicated Soda service group to the fixed native
surface. Neither layer accepts a browser-selected UID, account name, state path,
unit name, executable, provider command, or host flag.

Keep the current strict JSON models, ID/URL/label validation, request size limits,
operation timeout, shared native lock, configured Forgejo origins, and sanitized
errors. For Forgejo creation, overwrite any browser value with the configured
internal Forgejo origin before the root call; expose only the configured browser
origin in links. Only `provider=forgejo` is accepted. Reject every other provider
before native dispatch; a saved unsupported provider makes inventory unavailable
rather than being omitted or interpreted as Forgejo.

During the overlap, Cockpit, the CLI, and the web page reach the same state. Native
serialization must cover **all** runner mutations before enabling the second
UI; the implemented source lock covers reads and all mutations, including the whole
restart enable/restart sequence. A Lit busy
button protects only one document and cannot prevent a Cockpit action, another tab,
or remove/start races. Reads during a partial mutation must return unavailable or a
specific uncertain outcome, not a fabricated previous state.

## Operation contract

| Action | Current native effect | Required UI and failure contract |
| --- | --- | --- |
| List / Refresh | Reads local descriptors, client versions, and exact systemd state; performs no provider query. | Show configured slots and listener state as local observations. Do not call a running listener online, idle, or provider-available. Preserve the previous view only as visibly stale if refresh fails. |
| Register Forgejo | Writes the provider-issued UUID/token and host labels into the runner's private state, records a local descriptor, enables and starts the listener. The provider record must already exist. | Explain that an authorized Forgejo administrator creates the **system** runner record and supplies its UUID/token. Do not grant site-admin rights, use Soda's setup token, or silently create/reset provider registration. Re-read local state after the operation. |
| Start | `systemctl enable --now` for the validated existing unit. | State that the listener starts now **and at host boot**. Do not create, repair, upgrade, or re-register it. Re-read actual state. |
| Stop | `systemctl disable --now` for the validated existing unit. | State that the listener is disabled for host boot and any active local job may be interrupted. Provider job outcome is not locally known. Re-read actual state. |
| Restart | Enables the unit, then restarts it. | Warn that it can interrupt a job and that it leaves boot start enabled, even if the listener was previously stopped. Do not call it a provider reconnect guarantee. |
| Remove | Stops/disables the unit, deletes the Linux account, then recursively removes that runner's local state directory. | Require the exact runner ID and a destructive warning covering provider credentials, work files, dependencies, and uncommitted job changes. Provider registration/history remain. Report partial outcomes: a failed account deletion leaves state; a failed state deletion occurs after account removal. No rollback or automatic provider cleanup. |

Creation also has partial states. Failure before retention normally attempts to
remove the new account and local directory, but cleanup can fail. A listener-start
failure occurs after the descriptor and provider client state are retained. The API
and Lit store must refresh after success and failure, keep the diagnostic free of
secrets, and make unconfirmed local/provider outcomes explicit before another try.

## Registration secrets

Treat registration tokens and all provider client state as credentials. The web
port may carry a token only in the protected HTTPS request body and the restricted
Unix-socket request body to the root operation. It must never put one in a URL,
query, redirect, HTML state, data attribute, browser storage, observable Lit store,
command argument, environment variable, journal field, trace, screenshot, or error.
Use `Cache-Control: no-store`, clear the password input and transient payload as
soon as the request is serialized, and never return the token.

The current Forgejo runner intentionally retains its connection token in a
mode-0600 file owned by the runner account and references it by file path. The
token file is the only retained registration credential channel. Do not expose,
download, inspect wholesale or migrate it through the settings page. Logs and evidence may contain runner IDs,
non-secret versions, and summarized results, never request bodies, provider
responses, tokens, or complete state directories.

## Compatibility and migration

Preserve existing IDs, descriptors, Linux accounts and UIDs, state directories,
working files, credentials, units, enabled/running state and Forgejo registrations. Opening the new page performs
no migration, service action, registration, repair, client update, or descriptor
rewrite. The `soda-dashboard` and `/var/lib/soda/dashboard` names remain backend and
persistent-data names; UI placement does not rename them.

The current descriptor does not store labels and the list protocol cannot report
them. Existing rows must therefore show labels as unavailable or link to the
provider's authorized view. Do not parse opaque provider credentials or infer labels
from workflow usage. If later source adds locally recorded label metadata, it needs
an explicitly backward-readable representation and an `unknown` legacy state;
never rewrite retained descriptors merely to make the table look complete. Deploy
any root protocol/helper and web consumer changes as a paired, version-compatible
set while the old Cockpit/CLI path continues to work.

Provider-side deletion or label edits can leave the local descriptor stale. That is
an honest provider/local mismatch, not permission for background reconciliation.
The operator uses native provider controls for provider cleanup and then an explicit
Soda action for local cleanup. No periodic sync, copied provider permission
inventory, runner adoption, or generalized recovery subsystem is introduced.

## Implementation and completion gate

This is the detailed plan for **finishing the Cockpit-to-dashboard move**, based on
the source inventory above. The [settings contract](sodaspaces-plan.md#settings-pages-and-os-selection)
owns placement; this guide owns runner parity and cutover; the
[handoff](implementation-status.md) owns executed evidence. Only step 1 was selected
for the prior presentation/lifetime turn. The user subsequently selected removing
GitHub runner support. The old service-compatibility recommendation is retired;
there is no GitHub client or compatibility phase to finish. Later validation and
delivery proposals below are not execution authorization.

The end state is a discoverable, configured-operator-only page that controls real
local Forgejo capacity, with working native provider links and preserved
runner state. Cockpit retains Tailnet and ordinary host administration. The removed
Cockpit runner presentation leaves the native runner owner, CLI and provider
mechanisms in place.

Runner OS/OCI isolation, AI automation, cache policy, metrics, drain/admission,
automatic reconciliation, client auto-upgrades and a general settings framework are
not prerequisites. Current host execution must remain explicit. No new provider
permission inventory, runner database, public root listener or Forgejo fork is needed.

### 1. Finish dashboard presentation and browser lifetime

**Status: implemented with local checks.** Owners remain the existing Lit page,
settings CSS, Go template and native navigation hook. This step retains the separate
Soda-rendered page and its fixed native links; it does not integrate its HTML shell
into Forgejo's dashboard or redesign Spaces. The [handoff](implementation-status.md)
records passing affected checks and the unsuccessful broader source-gate attempts.

The selected stock Forgejo 15.0.7 calls `custom/extra_links` from its navbar.
Its [official v15 customization contract](https://forgejo.org/docs/v15.0/contributor/customization/)
documents that hook. Keep the existing integration and version-specific checks;
no new Forgejo handler or whole-navbar override is required.

- Restore the useful provider navigation supplied by Cockpit. Add a configured
  Forgejo **Actions administration** link with its separate permission explanation,
  and each runner's validated provider destination. The provider may deny a
  nonadmin Soda operator; the convenience link grants no additional authority.
  Lit rows now render validated provider links from
  the configured public Forgejo origin. A local ID is not a provider
  runner ID; do not invent a deep link from it. Never expose the internal Forgejo
  origin, use a guessed port or attach credentials to a link. The API projection
  and both UIs use only the configured public Forgejo origin; stored registration
  URLs never become link destinations. Unsupported providers are refused at the
  descriptor, API and shared response-decoder boundaries without rewriting data.
- Complete first-use navigation. The native settings link currently requires an
  already connected, matching Soda operator session. Check the journey for an
  operator signed into Forgejo but not Soda: source guidance now exposes the existing
  Spaces → explicit Soda connection → settings route and retains the direct protected
  settings login. Native signed-in markup cannot establish Soda operator authority.
  Do not call native site-admin markup an operator check or connect OAuth silently.
- Complete form feedback and keyboard/responsive behavior using existing tokens
  and Lit conventions. Preserve exact-ID lifecycle confirmation, boot-policy and
  destructive effects, local-state uncertainty and the distinction between listeners
  and available job slots. Use field guidance for invalid UUID/labels without
  echoing provider diagnostics or a token. Keep Refresh and provider inspection
  usable when they are the safe next action. The page now renders a bounded HTML denial
  for a nonoperator opening the page, replacing the API-shaped 403 while
  preserving the status code, fresh authority checks and zero native reads.
- Preserve the browser-lifetime coverage and guarded request generation.
  The page now handles disconnect cleanup, `beforeunload` and explicit
  `pagehide`/`pageshow` restoration. On page departure, successful logout or
  authorization loss, clear unsent credential inputs. There is no provider selector.
  On BFCache return, discard pending confirmation, revalidate the original actor/session and
  refresh before enabling mutation. A departed page must not continue from a pending
  session fetch into a new mutation or apply late responses. Retire the original
  request generation on departure/disconnect so reconnecting the same element
  cannot revive earlier continuations. Cancelling a fetch
  after dispatch does not prove a native operation stopped; retain uncertainty and
  never replay it. Tests deliberately release old responses after reconnection,
  including responses that ignore abort. No real credential leak was observed.

**Exit:** the actual Go HTML with emitted Lit assets covers the Forgejo form,
validated links, disconnected/denied/expired states, confirmation, empty/populated/
stale inventory, navigation and restoration. Keyboard, focus, narrow viewports and
canonical light/dark styling receive browser inspection. Merely matching Cockpit's
layout is not the acceptance criterion. Assert that unsent tokens disappear at
`pagehide` and remain absent after `pageshow`; already dispatched mutations are
never replayed and remain unconfirmed until their outcome is established.

### 2. Retired GitHub runner work

The user selected removing GitHub runner support. The previous launcher repair
proposal is retired. Registration, launch, provider switching, archive fetching,
source locks and packaging support have been removed. Historical source and
research remain in Git at `216db47`; no provider or retained-state cleanup follows
from this source change.

### 3. Complete local parity and regression coverage

**Status: substantial existing coverage; extend specific missing cases.** Keep
tests with their current owners and use the ordinary source gate. Do not create a
second runner test orchestrator or infer that previously passing fixtures cover
new behavior.

| Boundary | Required coverage and outcome |
| --- | --- |
| Page/API authority | Configured operator without site-admin rights succeeds; unrelated site/org/repository admins and ordinary users cannot read metadata or mutate. Exercise page/deep links and every API, missing/ambiguous cookies, wrong actor, missing scope, changed provider subject, expired grant/session and logout while authorization is pending. Denial precedes body decoding/native dispatch. |
| Request and secret boundary | CSRF/origin/method/query/strict-body failures; fixed Forgejo internal destination and refusal of all other providers; no browser-selected unit/account/path/command. Link tests prove that saved URLs never override the configured Forgejo public origin. Token cleared on serialization, departure/logout and authorization loss, absent from reactive state/storage/HTML/errors; synthetic secret probes never use real credentials. |
| Provider and UI parity | The Go-HTML/Lit journey now exercises the Forgejo form, field/actor refusals, exact target confirmation, both logout outcomes and no automatic retry after an unconfirmed operation. These use synthetic peers, not real provider registration. |
| Native/overlap semantics | Keep reads/create/start/stop/restart/remove under the same cross-process lock. Extend separate-process `internal/runners.Native` fixtures with a shared temporary lock path and command doubles; Restart must hold admission across enable/restart. Test CLI and socket adapters through their existing seams; actual CLI → Native and web → soda-host overlap belongs to step 5, not production test flags or another helper. A cancelled waiter must not dispatch. Invalid/unreadable descriptors and unavailable unit/version observations must not become an empty list. |
| Partial outcomes and preservation | Cover provider registration followed by descriptor/start failure, account-delete failure before local state removal, and state-removal failure after account deletion. Use owned temporary data and native-command doubles. Preserve legacy descriptors without adding label metadata or rewriting credentials. |
| Payload and coexistence | Settings modules/styles/runtime, hooks and legal notices appear in the canonical stage. Preserve native Actions settings and all Cockpit/Tailnet entrypoints. Verify staging and build inputs no longer fetch or contain a GitHub runner client. |

Run focused Go/race and browser cases during edits, then `bun run check:source`
with the existing toolchain/dependencies. That command owns Go/module, strict
TypeScript/Lit, emitted-page/browser, Cockpit and Python source checks. Supplement
with affected runner/host/web race tests where the ordinary gate is not a race run.
Record exact revision, actual tool versions, failures, skips and new evidence paths.
Do not repeat full suites without a new change or unresolved concern.

**Exit:** the candidate's local checks pass, both Cockpit and dashboard still work
against one protocol, and every test result is labelled synthetic/local. No native
service or provider acceptance follows from this exit.

### 4. Prepare product-owned native journeys and a paired candidate

**Status: remaining test/delivery preparation.** Extend
[`tests/installed/`](../tests/installed/) and the
[operator Runners journey](native-validation.md#operator-runners-journey), keeping
the existing read-only Cockpit/Tailnet probe read-only. Add separately gated
dashboard/provider cases using existing browser, private-input and evidence
conventions. Support tools supply transport and observations, not copied scenarios.

The test inputs must identify the exact target, browser origin/trust, configured
operator and denied test actor, each approved provider scope/repository/registration,
trusted workflow and run-owned runner IDs. Secret inputs remain restricted files
or protected browser fields; evidence records only non-secret IDs and selected
observations. Separate list/auth checks, registration/job/lifecycle checks, host
reboot and destructive local/provider cleanup. Nothing opts into all of them by
opening the page or enabling a read-only test.

For the chosen native architecture, build/check/export the exact clean candidate
through `scripts/build-native.sh`, `scripts/check-native.sh` and the existing
sealed-bundle producer. Inspect the real selected Forgejo runner package/version,
host dependencies and service confinement; the package version on
the development machine is not proof of the retained target's version.

Prepare one matching affected-artifact set. The running backend is the dashboard
image containing `soda-dashboard`; its separately staged host binary is not that
service. Pair **`soda-host` and `soda-runners`**, since both now embed the native
state lock. Include the Forgejo-only `soda-runner-launch`, settings assets/hooks and
only necessary service changes; the launcher is not a management-lock owner.
An approved delivery must also identify and retire the obsolete installed
`soda-runner-helper` executable after all earlier management processes finish;
source/package removal alone does not disable that independently callable old writer. Preserve the old Cockpit
payload during this phase. No whole-appliance installation recipe is an updater.

**Exit:** reproducible product-owned test inputs, a verified candidate manifest,
rehearsable compatibility changes and a concrete target/action/effect list are
ready for the applicable native/provider authorization. Preparing the plan itself
does not grant new VM, provider, reboot or cleanup actions.

### 5. Prove the delivered replacement on an isolated native fixture

**Status: not run.** Use an explicitly authorized fixture and trusted provider
resources. Retain failed attempts and partial states.

Before candidate activation, inventory the preservation baseline. Saved unsupported
providers require an explicit operator decision before deployment; do not reinterpret
or delete them. New disposable Forgejo runners own the job-interruption, lifecycle
and fault cases below; the preservation baseline is not the destructive test target.

1. Exercise native Forgejo navigation → explicit Soda OAuth → Sodarunners through
   the real proxy, backend, root socket and systemd. Prove both sides of operator/
   site-admin separation through UI and direct APIs, including the nonadmin
   operator, logout, expired session, restored page and untouched native Actions.
2. Register one approved Forgejo runner from the
   page. Check exact account/UID, directory ownership/modes, descriptor, installed
   version, boot policy and one configured slot. Provider-native registration
   authority is exercised separately from Soda operator authority.
3. Run one real trusted provider-scheduled job, recording native identity,
   successful steps and the provider's actual result. Exercise idle and active-job
   Stop/Start/Restart and inspect process termination and provider outcome; a green
   local listener is insufficient. Observe provider offline/online state in its
   native interface without making that a new Soda inventory feature.
4. Compare Cockpit, CLI and dashboard local observations and bounded concurrent
   operations on the exact fixture IDs. Verify serialization and truthful refresh/
   uncertainty. Inject only explicitly scoped failures; no changes to unrelated
   provider/network policy or guessed repair commands.
5. Prove a pre-existing runner survives activating the new page without descriptor,
   account, work-data, credential, client-version or enabled/running-state changes.
   If the authorized fixture has no such runner, creating a prior-version baseline
   requires its own declared provider/fixture scope. Hash/compare private data
   without exposing it. Then, only with reboot scope, verify boot policy and
   retained state across that same fixture's reboot.
6. On an explicitly disposable registered runner, confirm exact-ID Remove, native
   unit/account/state outcomes and the remaining provider record/history. Provider
   cleanup is a separate explicit action. Keep the preservation baseline and all
   evidence; no blanket cleanup.

**Exit:** real UI-to-native/provider parity and preservation pass for the exact
candidate and architecture. Record x86_64 and aarch64 separately; neither
cross-compilation nor an x86 result establishes native aarch64 acceptance. Make
progress on available native hardware without a sibling-architecture build barrier.
An unavailable provider leaves that part open and Cockpit retirement pending.

### 6. Rehearse preserved-state delivery, then cut over the chosen target

**Status: not authorized or executed by this planning pass.** Follow the current
[installation constraints](installation.md#existing-state-dashboard-migration) and
[credential/schema contract](dashboard-credentials.md). Select the actual target;
previous `soda-test` or validation-VM permission is not a fresh deployment grant.

Inventory current binaries/images/effective units, schema, private inputs, native
customizations, runner accounts/state/work files, the host Forgejo runner version,
enabled/running state and active jobs. Include projects and current browser
terminals in the preservation/interruption assessment: restarting `soda-host` is
not an action confined to the new runner page. Prepare only affected changes and
their exact service interruptions; do not stop projects or runners implicitly.

Take fresh consistent application/config/key/artifact backups. Consistent runner
state backups require separately approved listener/job quiescence; otherwise
record bounded inventory/hashes and do not claim a recoverable state backup.
Rehearse the actual schema and paired artifacts on copied state. Do not start
cloned listeners with live copied
provider credentials; perform provider execution on the isolated approved
registrations from step 5. Rehearsal snapshots are not current rollback data.

After concrete delivery approval, stop admitting new Cockpit/CLI/web runner
management calls for the declared maintenance window and verify that prior
coordinator/helper management processes have finished. Replacing an executable
does not change an already running helper. Refuse an unreviewed rollout onto
retained unsupported providers; source removal is not permission to delete them.
Deliver compatible web and both native paths together, then verify exact running
bytes and reopen access. Keep listener units running except for separately
approved backup/compatibility interruptions. This prevents an old executing
Cockpit helper from overlapping new lifecycle semantics. Do not re-run setup,
regenerate credentials, replace project roots or rewrite runner descriptors.

Verify real dashboard access, native Actions coexistence, Cockpit fallback and
unchanged runner/project identities, work data, credentials and boot policy. Run
only the approved retained-target checks; do not repeat destructive fixture journeys
on retained capacity. On failure preserve new writes/partial state and assess a
compatible restoration; never restore an old database or runner snapshot blindly.

**Exit:** the approved target serves the exact verified replacement, preservation
checks pass, and the handoff records its bounded evidence and remaining platforms.
Cockpit Runners remains until the final coordinated removal.

### 7. Retire only Cockpit's runner presentation

**Status: gated on delivered parity and a coordinated removal change.** Remove
`cockpit/soda-runners/`, runner-only React presentation/store/transport and their
exclusive UI tests only after their applicable behavioral coverage exists at the
new owner. Adjust `cockpit/vite.config.ts`, package inventory, staging and payload
tests so new bundles stop shipping that page. Audit real imports before removing
dependencies: Tailnet still needs the Cockpit/React/PatternFly toolchain.

Keep `internal/runners`, the root CLI boundary, runner launch/service/client
inputs, sysusers/tmpfiles, shared protocol and focused native/CLI tests. Native
Forgejo Actions runner templates/styles are provider UI and must remain. Preserve
the existing Tailnet page, its backing logic/tests, root login and ordinary Cockpit
Services/Logs. Port the installed operator check's runner section; retain its
Tailnet and root-authority checks.

Update the public CI guide, installation/staging documentation, architecture and
AGENTS guidance to name Global SodaOS settings as the normal runner entrypoint.
Describe Cockpit Services/Logs for native diagnostics and remove the interim
"Cockpit Runners remains" copy only with this cutover. Installed removal is limited
to the verified obsolete page/assets on approved targets; never runner state,
accounts, units, provider records or the entire Cockpit installation. Keep the
installed fallback on any target that has not passed replacement parity.

**Final exit:** source and delivered UI agree, no removed Cockpit runner navigation
or stale package entry remains on the migrated target, Tailnet still works, local
capacity is managed from Sodarunners, and native provider workflows/jobs remain
upstream-owned. Record source completion, each architecture's native proof, target
delivery and Cockpit retirement separately. No percentage or whole-product
acceptance claim substitutes for those results.
