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

## Current source and observed evidence

The existing implementation is the Cockpit page plus
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
removal. There is no Global SodaOS settings page or web-to-root runner bridge in
current source.

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
UI; the current lock covers create/remove but not start/stop/restart. A Lit busy
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
units, provider clients, and focused non-UI tests. No build, test, provider action,
service mutation, deployment, or Cockpit removal was performed by this documentation
revision.
