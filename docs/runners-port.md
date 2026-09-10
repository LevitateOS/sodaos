# Local CI runners

**Current source:** local CI runners support only Forgejo. The user selected
removal of GitHub runner registration, execution and packaging. Unsupported saved
descriptors are rejected without changing their accounts, files or credentials;
this source change performs no installed cleanup.

**Migration status:** the dashboard
replacement is implemented in source and has recorded local test coverage. It is
not yet a delivered, provider-validated replacement for Cockpit. Finish the
[implementation and completion plan](#implementation-and-completion-gate) below;
reuse the existing controls, API and native bridge.

**Subsequent integration request:** the [native page integration plan](forgejo-soda-pages-plan.md)
now owns moving this page body into Forgejo's real header/profile shell, naming its
operator navigation link Runners, automatic connection on initial entry and
coordinated normal logout. Native page hosts and shared connection/logout now have local implementation
and browser proof; the existing runner controls now mount in that native host.
Normal navigation reaches it, and the old Go URL is a fixed bookmark bridge.
The plan supersedes
the separate HTML placement and explicit-only connection guidance below; the
completed presentation/lifetime slice remains historical evidence. Native/provider
parity and coordinated Cockpit retirement remain governed by this runner plan.

## Implementation lane boundary

The [Soda-pages ownership table and handoff](forgejo-soda-pages-plan.md#implementation-lanes-and-handoff)
is authoritative for dividing these two implementation lanes. **This lane owns
runner operation correctness, native/provider parity, paired management delivery
and Cockpit retirement—not native page hosting, navigation, OAuth/connection,
coordinated logout, shell migration or their browser/payload fixture ports.**
The Soda-pages lane has implemented step 2 with local proof; its shared files
remain reserved until the explicit source handoff. Do not edit its shared
session/OAuth/store work or implement a competing runner-specific flow.

Step 1 below is a completed historical slice, not an active UI assignment.
Proceed independently with runner backend/CLI/socket regressions and installed
runner/provider test preparation. The Soda-pages lane reserves shared runner UI,
page-handler, browser-fixture and page-packaging files through its source handoff;
request an exact operation fix through that owner or serialize it afterward.
Reuse its entry/actor/logout/retirement tests and committed interface rather than
copying them. Final native-shell runner UI parity depends on that handoff, while
native backend preparation does not. Runner OS/OCI/cache remain separate work,
not part of the Soda-pages lane or prerequisites for this migration.

For shared candidate builds and target maintenance, follow the linked
single-executor delivery rule. Consume shell/schema evidence; own the additional
runner management/provider/preservation checks. Neither plan authorizes duplicate
deployment, changes to the other agent's files, or provider/lifecycle effects.

## Selected product boundary

Move Soda's local runner capacity and service controls to the operator-only
**Runners** destination selected by the Soda-pages lane (previously named
**Global SodaOS settings → Sodarunners**). Current source uses a Soda-owned Go page
and Lit component under the existing `/-/soda/` namespace. The subsequent integration plan
hosts that component within Forgejo's native dashboard through its documented
template customization, while keeping Soda's protected API. It is not a Forgejo administrator page, a
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
| Operator page | [`settings_page.go`](../internal/web/settings_page.go), [`runners.html`](../internal/web/templates/runners.html), [`soda-runners-page.ts`](../frontend/runners/soda-runners-page.ts): separate Go shell and Lit inventory, registration, lifecycle, provider links, HTML denial, keyboard/responsive presentation and guarded browser restoration, with local coverage. | Soda-pages owns migration and browser proof; this lane verifies real runner operations on its handed-off page. |
| Navigation | [`extra_links.tmpl`](../appliance/forgejo/templates/custom/extra_links.tmpl), [`footer.tmpl`](../appliance/forgejo/templates/custom/footer.tmpl), [`soda-settings-link.ts`](../assets/branding/forgejo/soda-settings-link.ts), and the Spaces page. | Soda-pages owns discoverability/connection proof, including a nonadmin operator; consume its entry contract. |
| Web authority and API | [`internal/web/runners.go`](../internal/web/runners.go): fresh provider subject, configured `operator_id`, original Soda session, guarded list/create/lifecycle routes; existing API middleware supplies actor/origin/CSRF checks. | Preserve denial-before-decode/native-read behavior and exercise the actual proxy/OAuth/helper chain. |
| OAuth and persistence | Fixed `destination=runners` return was added in schema v7; source at the page-host handoff is schema v8 and preserves that return. | Soda-pages owns subsequent shared return/logout/schema changes and rehearsal. Consume its exact candidate schema; do not add a runner inventory migration or duplicate the OAuth work. |
| Root bridge | [`internal/host/runners.go`](../internal/host/runners.go), wired by [`cmd/soda-host`](../cmd/soda-host/main.go), dispatches six fixed operations to the existing runner implementation. | Deliver compatible binaries together and prove socket/group/service confinement natively. |
| Native runner owner | [`internal/runners/`](../internal/runners/): provider configuration, dedicated accounts/state, one-slot listeners, lifecycle and cross-process file locking for reads and all mutations. | Real provider lifecycle/jobs and retained-state preservation remain unproved. Only Forgejo is supported; unsupported saved providers are refused before native effects. |
| Shared frontend protocol | [`soda-runner-types.d.ts`](../frontend/runners/soda-runner-types.d.ts) and [`soda-runner-response.ts`](../frontend/runners/soda-runner-response.ts) serve Lit and Cockpit. | Keep one response contract while both surfaces coexist; preserve backend/CLI tests after Cockpit removal. |
| Packaging | [`forgejo-payload.json`](../internal/nativebuild/forgejo-payload.json), [`build-forgejo.ts`](../scripts/build-forgejo.ts), [`stage.py`](../scripts/stage.py) include settings assets, native hooks and both Cockpit pages. | Consume Soda-pages assets/hooks; this lane owns runner executable/service/client compatibility and Cockpit coexistence/retirement. Use the single-executor candidate handoff. |
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
unreadable/ambiguous descriptor, missing or ambiguous required unit observation,
or unavailable/blank client version makes the whole list fail;
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

**Status: completed historical slice, not an active implementation assignment.**
The following records what the separate Go/Lit page implemented and tested. Its
shell, explicit entry and standalone logout are superseded by Soda-pages steps
2–5, which alone own porting this UI and its browser tests. Do not repeat this
checklist as parallel runner work. Preserve its operation invariants through that
handoff. The [handoff](implementation-status.md) records the affected checks,
earlier unsuccessful aggregate attempts and later revision-specific source passes;
none establishes native/provider parity.

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
  Do not call native site-admin markup an operator check. This recorded explicit
  connection flow is superseded by the bounded automatic-entry contract in the
  native page integration plan; first consent and failure recovery remain visible.
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

**Status: runner-owned backend gaps implemented; focused/race and aggregate
source checks passed for `7cb316b`. Soda-pages integration handoff remains open.**
The Native fixtures now use separate test processes sharing one temporary lock,
cover Restart's enable/restart admission and failed-command release, and require
cancelled contenders to exit without dispatch. New fixtures cover Remove failures
at stop/account/state stages, malformed/duplicate/trailing descriptors, missing or
ambiguous systemd observations, blank client versions, whole-list failure and
unchanged legacy files/credentials. Every runner API has operation-specific
admission/strict-body/target/sanitized-failure checks; socket tests retain the
same fixed protocol. These are synthetic peers and command doubles, not real
provider registrations, Linux account deletion or installed process proof.

Native/CLI Remove errors now identify the reached stage without command output;
the existing web/socket protocol deliberately still returns generic unconfirmed
outcomes. Any stage-specific web presentation belongs to a subsequent explicit
shared-file handoff, not an edit to the Soda-pages agent's reserved files. The
[handoff](implementation-status.md) records exact checks and remaining limits.

Keep tests with their current owners and use the ordinary source gate. Do not
create a second runner test orchestrator or infer that previously passing fixtures
cover new behavior. This step owns runner operation/backend gaps only. The Soda-pages
lane ports the shared page/browser tests and owns connection, logout, navigation
and restoration integration; consume that evidence rather than adding a second
matrix. Reserve shared UI/test files until its explicit commit handoff.

| Boundary | Required coverage and outcome |
| --- | --- |
| Runner API authority | Configured operator without site-admin rights succeeds; unrelated site/org/repository admins and ordinary users cannot read metadata or mutate. Exercise every runner API's rejection before body decoding/native dispatch for invalid actors, scopes or sessions, including logout during pending authorization. Consume shared cookie/session protection tests. Soda-pages alone owns page/deep-link, OAuth and coordinated-logout journeys and any shared auth changes. |
| Request and secret boundary | Own runner endpoint CSRF/origin/method/query/strict-body refusals, fixed internal destination, unsupported providers and rejection of browser-selected unit/account/path/command. Own sanitized API responses and public-origin projection. Soda-pages ports existing browser link, token clearing/storage and departure/logout/no-replay assertions; request missing operation-specific cases through its file handoff. Synthetic probes never use real credentials. |
| Provider and UI parity | Consume the Soda-pages port of the existing form, confirmation and uncertainty assertions. Add only missing runner-operation behavior after its source handoff; no duplicate host/bootstrap/logout fixture. Real provider parity belongs to step 5 below, not either lane's synthetic browser checks. |
| Native/overlap semantics | Keep reads/create/start/stop/restart/remove under the same cross-process lock. Extend separate-process `internal/runners.Native` fixtures with a shared temporary lock path and command doubles; Restart must hold admission across enable/restart. Test CLI and socket adapters through their existing seams; actual CLI → Native and web → soda-host overlap belongs to step 5, not production test flags or another helper. A cancelled waiter must not dispatch. Invalid/unreadable descriptors and unavailable unit/version observations must not become an empty list. |
| Partial outcomes and preservation | Cover provider registration followed by descriptor/start failure, account-delete failure before local state removal, and state-removal failure after account deletion. Use owned temporary data and native-command doubles. Preserve legacy descriptors without adding label metadata or rewriting credentials. |
| Payload and coexistence | Consume Soda-pages canonical assets/hooks/runtime/notices checks. Own runner executable/service/client payload, obsolete helper absence, Forgejo-only build inputs and Cockpit/Tailnet coexistence. Coordinate edits to shared staging files; do not independently port page packaging. |

Run focused Go/race and browser cases during edits, then `bun run check:source`
with the existing toolchain/dependencies. That command owns Go/module, strict
TypeScript/Lit, emitted-page/browser, Cockpit and Python source checks. Supplement
with affected runner/host/web race tests where the ordinary gate is not a race run.
Record exact revision, actual tool versions, failures, skips and new evidence paths.
Do not repeat full suites without a new change or unresolved concern.

**Exit:** the candidate's local checks pass, both Cockpit and dashboard still work
against one protocol, and every test result is labelled synthetic/local. The
runner-owned backend exit passed for `7cb316b`, including all 11 existing Go-HTML/
runner browser cases and retained Cockpit tests. This does not close the Soda-pages
lane's migrated native-shell browser exit or any stage-specific web feedback still
requiring its shared-file handoff. No native service or provider acceptance follows
from these local checks.

### 4. Prepare product-owned native journeys and a paired candidate

**Status: partially implemented, not blocked as a whole.** Runner-owned inputs,
callable phases and observers are implemented. Native connection/logout and all
three page bodies have now landed in Soda-pages (`6097564`, `a77dea1`). Its step 4
still owns navigation/legacy-shell retirement and caller/test migration. Do not
wait for page bodies that already exist or treat them as the final source handoff.
The earlier `9476858` source-check receipt is historical; the pulled combined
checkout needs its own applicable checks and exact-candidate evidence. The
[native runner preparation guide](runners-native-validation.md) owns exact source
entrypoints, private input/effect gates, candidate artifact checklist and blockers.
`tests/installed/runners.ts` supplies callable list/denial, registration and exact
lifecycle/removal cases; companion fixed native/provider observers use the real CLI,
state/credential hashes and official exact-run Forgejo APIs. The manual-only trusted
workflow records native job identity and two-step workspace proof. Local tests use
synthetic command/HTTP outputs and owned files, not native/provider execution.

Complete the following preparation in order, without turning execution approvals
into a hold on independent local implementation:

1. **Review the landed contracts.** Consume the native entry URLs, original-actor
   binding, connection/logout/retirement behavior and request guard. Record any
   runner-specific mismatch against its actual owner; do not rebuild the shell or
   authentication integration.
2. **Integrate the installed caller after a file-level handoff.**
   `loadRunnerInput` and `exerciseRunners` currently have no installed callers.
   Soda-pages retains ownership of `tests/installed/sodaspaces.ts` and shared
   browser/authentication files until an explicit committed handoff names the
   editor and files. Wire the phases into that existing driver/one-shot guard,
   then test admission, credential clearing, failure evidence and no replay.
   Runner-owned module/test improvements can proceed before this handoff; do not
   edit reserved files in parallel or add a second login/test orchestrator.
3. **Finish scenario preparation.** Use step 5's coverage table below to distinguish
   callable phases from missing integrated scenarios. Author missing scoped cases
   with the existing product-owned journeys, or specify exact approved native-tool
   procedures where automation is not selected. Every required case needs explicit
   inputs, effects, observations and failure/evidence retention before execution;
   do not leave it as an unspecified manual checklist or a new support-tool gate.
4. **Validate the combined source and prepare the builder.** Run applicable focused
   checks and the ordinary source gate on the final combined source. Prepare the
   pinned native Go 1.26.7/Bun 1.4.2 toolchain locally. Go 1.26.7 is now downloaded
   and checksum-verified under `.artifacts/runners-step4-continued/toolchain/go/`;
   select it through command-local PATH, leaving the host default/pins unchanged.
   Toolchain availability is no longer a blocker; candidate build/export is still
   outstanding.
   Reuse applicable exact-candidate receipts; do not transfer historical PASS
   records across changed source merely because commits were pulled/rebased.
5. **Prepare the candidate and execution proposal.** Produce/inspect the matching
   native export and affected-artifact set below. Select and document exact
   target/provider resources and the compatibility/effect list for authorization.
   Preparing inputs does not create a provider record, fixture or permission grant.

The distinction is explicit:

| Category | Remaining requirement |
| --- | --- |
| Local unfinished work | Runner scenario/caller integration, combined checks, pinned builder, native export and compatibility preparation. Routine authorized local implementation/testing may continue. |
| Cross-lane dependency | Committed file-level driver handoff and final Soda-pages navigation/assets/schema contract. Only dependent shared edits/final candidate readiness wait for these. |
| Execution approval/input | Exact native target, actors, provider resources, credentials and allowed actions. Real registration/jobs/lifecycle, reboot, cleanup and deployment remain separately scoped. |

The original Cockpit/Tailnet probes stay unchanged and read-only within their
existing scope. Support tools supply transport and observations, not copied
scenarios or a second readiness gate.

The test inputs must identify the exact target, browser origin/trust, configured
operator and denied test actor, each approved provider scope/repository/registration,
trusted workflow and run-owned runner IDs. Secret inputs remain restricted files
or protected browser fields; evidence records only non-secret IDs and selected
observations. Separate list/auth checks, registration/job/lifecycle checks, host
reboot and destructive local/provider cleanup. Nothing opts into all of them by
opening the page or enabling a read-only test.

For the chosen native architecture, obtain one exact clean candidate build/check/
export through `scripts/build-native.sh`, `scripts/check-native.sh` and the existing
sealed-bundle producer, following the single-executor delivery handoff. Reuse a
matching combined-candidate receipt from Soda-pages; build again only for changed
bytes or missing applicable evidence, not to satisfy a duplicate plan checkbox.
Inspect the real selected Forgejo runner package/version, host dependencies and
service confinement; the package version on
the development machine is not proof of the retained target's version.

Prepare one matching affected-artifact set. The running backend is the dashboard
image containing `soda-dashboard`; its separately staged host binary is not that
service. Pair **`soda-host` and `soda-runners`**, since both now embed the native
state lock. Include the Forgejo-only `soda-runner-launch`, the handed-off Soda-pages
assets/hooks and only necessary service changes; the launcher is not a
management-lock owner.
An approved delivery must also identify and retire the obsolete installed
`soda-runner-helper` executable after all earlier management processes finish;
source/package removal alone does not disable that independently callable old writer. Preserve the old Cockpit
payload during this phase. No whole-appliance installation recipe is an updater.

**Exit:** reproducible product-owned test inputs, a verified candidate manifest,
rehearsable compatibility changes and a concrete target/action/effect list are
ready for the applicable native/provider authorization. Preparing the plan itself
does not grant new VM, provider, reboot or cleanup actions.

### 5. Prove the delivered replacement on an isolated native fixture

**Status: native/provider execution not run; scenario preparation is incomplete.**
Finish each case's preparation under step 4 before its separately approved run.
Use an explicitly authorized fixture and trusted provider resources. Retain failed
attempts and partial states. Existing local fixtures are not installed acceptance.

| Required proof | Current preparation and remaining work |
| --- | --- |
| Native page admission and exact runner operations | Callable list/denial, register/start/stop/restart/remove phases exist; shared driver/guard wiring and its integration checks remain. Native proxy/socket/systemd proof is unrun. |
| Real trusted job and exact result | Dispatch/exact-run reads and a two-step identity/workspace fixture exist. Workflow publication, approved registration/unique label, driver sequencing and real provider execution are not done. |
| Idle/active-job lifecycle and termination | Individual lifecycle actions and PID/start/cgroup observations exist, not a complete interruption/process-tree/provider-outcome scenario. Prepare the exact sequence and observations; do not infer cleanup from listener state or one PID. |
| Cockpit/CLI/web overlap and faults | Local lock/command-double regressions exist. Installed concurrent callers, scoped fault procedures and partial-outcome receipts still need preparation and execution. |
| Existing-state activation and reboot | Selected state/hash observers exist, not a quiesced backup or prior-version activation/reboot journey. Define the baseline, activation comparison and separately gated reboot procedure without cloning live listeners. |
| Local Remove and provider aftermath | Local removal phase exists. Exact provider record/history inspection and any separately authorized provider cleanup procedure remain to be prepared; local deletion is not provider deletion. |

This table identifies remaining preparation, not optional acceptance criteria.
Keep these scenarios at their existing owners; do not duplicate Soda-pages shell,
OAuth, logout or common installed-artifact verification suites.

Before candidate activation, inventory the preservation baseline. Saved unsupported
providers require an explicit operator decision before deployment; do not reinterpret
or delete them. New disposable Forgejo runners own the job-interruption, lifecycle
and fault cases below; the preservation baseline is not the destructive test target.

1. Enter the handed-off native Runners page using the Soda-pages connection
   contract and cite its navigation/OAuth/logout/restoration evidence. This lane
   proves runner-specific UI/API admission through the real proxy, backend, root
   socket and systemd: a nonadmin operator can inspect/control local runners;
   a nonoperator site administrator cannot. Do not repeat the common shell/auth
   suite or restore the retired explicit-only connection flow. If that prerequisite
   fails, hand it back to Soda-pages; runner native admission remains independently
   required.
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

**Status: execution requires a later explicit delivery grant; recipe preparation
may proceed locally.** Entry requires step 5's applicable exact-candidate native/
provider proof plus the Soda-pages committed final navigation/assets/schema contract,
applicable integration receipts and reviewed affected-component delivery recipe.
Landed page bodies alone do not satisfy those prerequisites. Identify one executor,
one exact candidate manifest and the shared versus runner-only effects before any
maintenance window. Do not require a duplicate Soda-pages rollout if its applicable
delivery is already complete.

This step owns runner-management delivery only. Consume the Soda-pages shell/schema handoff;
common phases of an approved combined rollout have one executor and receipt,
not two competing maintenance recipes. A later runner-only rollout preserves the
already delivered shell/schema and rehearses only its actual compatibility delta,
with fresh target-state assessment and backups as required below. Follow the current
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

**Status: unchanged, gated on delivered parity and a coordinated removal change.**
This is downstream work, not a blocker to step 4 preparation. Applicable steps 5–6
proof must precede retirement; a local source pass or completed page mount is not
permission to remove the fallback. Remove
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
AGENTS guidance to name the Soda-pages lane's handed-off native **Runners**
destination as the normal runner entrypoint. Do not redesign its navigation.
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
