# Local CI runners

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
secrets, variables, workflows, jobs, and results with Forgejo. GitHub likewise
owns its registrations and Actions state. Soda owns only this appliance's runner
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
into CI capacity. Ubuntu and Arch are not selected. This choice does not claim
package-for-package or behavior parity with GitHub-hosted Ubuntu runners.

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
still executes Forgejo and GitHub jobs directly on host runner accounts. The
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
generic image until it resembles an undocumented GitHub runner image.
The comparison uses GitHub's [Ubuntu tool inventory](https://github.com/actions/runner-images/blob/main/images/ubuntu/Ubuntu2404-Readme.md),
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
provider/runner contract; GitHub behavior is comparison evidence, not proof of
Forgejo behavior.

## Current source and observed evidence

The local source now includes the protected Go/Lit page and fixed root:soda runner
adapter described below. Its local Go/emitted-browser proof is recorded in the
[handoff](implementation-status.md); native/provider parity remains pending and
Cockpit is not removed. The retained native implementation is the Cockpit page plus
`soda-runners` → `soda-runner-helper` → `internal/runners.Native` and the
`soda-runner@.service` launcher. Cockpit runs the coordinator in its authenticated
root session. Both coordinator and helper independently require the native caller
to be root; the helper receives requests over stdin and invokes only the fixed
runner operations. This path does not authorize a Forgejo web user.

Each runner has a stable, validated local ID, a dedicated noninteractive
`soda-runner-<id>` account, a private persistent state directory under
`/var/lib/soda/runners/`, one job slot, and a systemd instance. The service has no
sudo or Linux capabilities, has a read-only host view apart from its own state,
and can access the network. Jobs execute repository code and can change the
runner's persistent work files, so only trusted repositories and contributors
belong on this capacity. Current Forgejo labels are host-execution labels; the
GitHub runner also executes directly on the host account. OCI AI-job isolation is
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
unchanged from `8b823db`. No recorded evidence creates a Forgejo or GitHub runner,
runs a provider-scheduled job, or exercises native runner start, stop, restart, or
removal. The new source supplies the Global SodaOS settings page and web-to-root bridge;
neither has been deployed or exercised with real provider registrations.

## Required web and native authority path

The Go server already authenticates a separate Soda session through Forgejo OAuth,
binds it to a stable provider user ID, exposes whether that ID equals
`operator_id`, checks an expected actor header, and applies origin/CSRF protection
to mutations. Add a fixed operator-settings OAuth return and the protected
`/-/soda/settings/runners` page; do not accept a caller-supplied return URL. Check
`session.User.ID == Config.OperatorID` server-side before decoding a request or
reading runner state. Apply the same check to every page, API, deep link, refresh,
and mutation. Navigation visibility is only presentation and never authorization.

The dashboard container runs as UID 2000 with no capabilities, a read-only root,
and `NoNewPrivileges`; it cannot safely reuse the Cockpit coordinator by pretending
to be native root or by adding a web-triggered polkit prompt. Preserve the current
root-only `LinuxAuthorizer` for the CLI/Cockpit path. Add fixed runner methods to the
existing root:soda Unix-socket service and reuse `internal/runners.Native` behind
them. The web handler authorizes the configured Soda operator; the socket ACL admits
the existing root service and dedicated Soda service group to the fixed native
surface. Neither layer accepts a browser-selected UID, account name, state path,
unit name, executable, provider command, or host flag.

Keep the current strict JSON models, ID/URL/label validation, request size limits,
operation timeout, shared native lock, configured Forgejo origins, and sanitized
errors. For Forgejo creation, overwrite any browser value with the configured
internal Forgejo origin before the root call; expose only the configured browser
origin in links. GitHub remains limited by current source to HTTPS `github.com`
repository, organization, or enterprise-account paths, not arbitrary servers.

During the overlap, Cockpit, the CLI, and the web page reach the same state. Extend
native serialization to cover **all** runner mutations before enabling the second
UI; the new source lock covers reads and all mutations, including the whole
restart enable/restart sequence. A Lit busy
button protects only one document and cannot prevent a Cockpit action, another tab,
or remove/start races. Reads during a partial mutation must return unavailable or a
specific uncertain outcome, not a fabricated previous state.

## Operation contract

| Action | Current native effect | Required UI and failure contract |
| --- | --- | --- |
| List / Refresh | Reads local descriptors, client versions, and exact systemd state; performs no provider query. | Show configured slots and listener state as local observations. Do not call a running listener online, idle, or provider-available. Preserve the previous view only as visibly stale if refresh fails. |
| Register Forgejo | Writes the provider-issued UUID/token and host labels into the runner's private state, records a local descriptor, enables and starts the listener. The provider record must already exist. | Explain that an authorized Forgejo administrator creates the **system** runner record and supplies its UUID/token. Do not grant site-admin rights, use Soda's setup token, or silently create/reset provider registration. Re-read local state after the operation. |
| Register GitHub | Copies the pinned client, runs its native `config.sh` under the runner account with the short-lived token, records the descriptor, then enables/starts the listener. This can create provider state before a later local step fails. | Accept only an explicitly supplied authorized registration token. On timeout/failure, report that provider registration may remain and direct the operator to inspect it before retrying. Never claim compensation or retry automatically. |
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
GitHub client retains its own reconnect credential state after `config.sh`; its
short-lived registration input is sent through the existing no-echo PTY and not
argv or environment. Do not expose, download, inspect wholesale, or migrate these
files through the settings page. Logs and evidence may contain runner IDs,
non-secret versions, and summarized results, never request bodies, provider
responses, tokens, or complete state directories.

## Compatibility and migration

Preserve existing IDs, descriptors, Linux accounts and UIDs, state directories,
working files, credentials, units, enabled/running state, installed per-runner
GitHub client versions, and provider registrations. Opening the new page performs
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

1. Add operator-only Go page/API routes, fixed OAuth return, server-side deep-link
   checks, expected-actor/CSRF enforcement, no-store responses, and a complete Lit
   port of the current list/register/start/stop/restart/remove behavior. Keep the
   Soda-owned page shell and supported Forgejo navigation boundary.
2. Add the fixed root socket operations and shared mutation serialization. Reuse the
   native models/lifecycle and preserve the CLI/Cockpit contract; do not add an
   arbitrary command endpoint, web polkit grant, human host account, or second runner
   implementation.
3. Port the focused protocol/store/UI tests and add configured-operator versus site,
   organization and repository administrator denials; deep-link and stale-session
   cases; CSRF/expected-actor failures; secret non-retention; unavailable/partial
   reads; exact lifecycle semantics; cross-tab/Cockpit concurrency; and legacy
   descriptors/state preservation.
4. Build/stage and inspect the actual new page on an authorized target. Verify an
   operator who is not a site administrator can use it and a site administrator who
   is not the configured operator cannot read metadata or mutate through the page or
   direct APIs. Confirm native Runners pages and provider-owned workflows remain.
5. With separate provider and destructive-action authorization, use disposable
   Forgejo and GitHub registrations to prove create/list/start/stop/restart, one real
   trusted provider-scheduled job, provider-visible offline/online behavior, partial
   failure reporting, and exact local removal/provider leftovers. Also prove a
   retained legacy runner survives page activation and host reboot with unchanged
   account, state, credentials, client version, and boot policy. Native x86_64 and
   aarch64 evidence remain separate.

Only after those exits pass in the delivered interface may a separate coordinated
change remove the Cockpit Runners manifest/page/assets and adjust its build/staging
tests. Keep the native lifecycle, root boundary, CLI/support entrypoint, service
units, provider clients, and focused non-UI tests. Local Go, TypeScript/Lit and emitted-browser checks have now run for the new source;
see the handoff. No provider action, native service mutation, deployment or Cockpit
removal accompanied that source work.
