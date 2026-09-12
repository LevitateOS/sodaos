# Local CI runner contracts

This guide owns Soda's runner capacity, authority, operation, secret-handling and
compatibility requirements. The [native page guide](forgejo-soda-pages-plan.md)
owns shared entry/navigation/OAuth/logout; the [combined completion plan](native-pages-runners-plan.md)
alone owns coordination and Cockpit retirement. Current installations, grants and
results are recorded in the [handoff](implementation-status.md), not duplicated here.

<a id="implementation-lane-boundary"></a>
<a id="current-source-and-observed-evidence"></a>
<a id="implementation-and-completion-gate"></a>
Historical lane/progress/sequence links resolve here. Their original text is available
with `git show 3f80010:docs/runners-port.md`, not as current work or permission gates.

Local runner registration, execution and packaging are Forgejo-only. Unsupported
saved providers are rejected without changing accounts, files or credentials.
This does not remove project GitHub CLI tools or Forgejo's native repository import.

## Selected product boundary

Soda's local runner capacity and service controls belong to the operator-only
**Runners** destination. The source mounts the Lit component
at `/?soda-view=runners` within Forgejo's native dashboard through its documented
template customization, while keeping Soda's protected `/-/soda/api/` endpoints.
The old Go URL is a fixed bookmark bridge, not a separate page owner. It is not a Forgejo administrator page, a
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

Cockpit presentation retention/removal follows the [combined retirement gate](native-pages-runners-plan.md#6-retire-only-the-cockpit-runner-presentation).
Tailnet remains a separate Cockpit responsibility.

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

## Implemented source

| Responsibility | Owner |
| --- | --- |
| Native host and bookmark entry | `assets/branding/forgejo/soda-native-page.ts`, Forgejo dashboard templates and `internal/web/settings_page.go` |
| Runner controls | `frontend/runners/soda-runners-page.ts` |
| Web authority and fixed API | `internal/web/runners.go` |
| Shared OAuth/session persistence | Existing `internal/web` and `internal/store` owners; see the page and credential guides |
| Root bridge and native runner state | `internal/host/runners.go`, `cmd/soda-host`, `cmd/soda-runners`, `internal/runners/` |
| Shared Lit/Cockpit response protocol | `frontend/runners/soda-runner-types.d.ts`, `frontend/runners/soda-runner-response.ts` |
| Packaging | `internal/nativebuild/forgejo-payload.json`, `scripts/build-forgejo.ts`, `scripts/stage.py` |

The CLI/Cockpit path requires real/effective root, rejects non-root original
`PKEXEC_UID` and verifies the native root account before operations. The separate
web-to-helper authority path is specified below; a root CLI does not authorize a
Forgejo web user.

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

## Required web and native authority path

The Go server already authenticates a separate Soda session through Forgejo OAuth,
binds it to a stable provider user ID, exposes whether that ID equals
`operator_id`, checks an expected actor header, and applies origin/CSRF protection
to mutations. The page guide owns fixed native entry/OAuth returns and the
`/-/soda/settings/runners` bookmark bridge. Check
`session.User.ID == Config.OperatorID` server-side before decoding a request or
reading runner state on every runner API, refresh and mutation. A native shell
or bookmark response is not authorization to read capacity or dispatch an operation.

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

Cockpit, the CLI and the native page reach the same state. The native lock covers
reads and all mutations, including the whole restart enable/restart sequence.
A Lit busy button protects only one document and cannot prevent a Cockpit action, another tab,
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
| Remove | Stops/disables the unit, deletes the Linux account, then recursively removes that runner's local state directory. | Require the exact runner ID and a destructive warning covering provider credentials, work files, dependencies, and uncommitted job changes. Provider registration/history remain. Report partial outcomes: unconfirmed stop permits no account/state removal; failed account deletion leaves state; failed state deletion occurs after account removal. No rollback or automatic provider cleanup. |

Creation also has partial states. Failure before retention normally attempts to
remove the new account and local directory, but cleanup can fail. A listener-start
failure occurs after the descriptor and provider client state are retained. The API
and Lit store must refresh after success and failure, keep the diagnostic free of
secrets, and make unconfirmed local/provider outcomes explicit before another try.
Native/CLI errors identify the reached Remove stage; the web/socket response remains
sanitized and generic/unconfirmed rather than exposing native command output.

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

### Paired artifact compatibility

Review the affected set against the actual sealed inventory and target state.
The full producer also builds project/service images; their inclusion does not
select them for delivery or permit replacing project roots.

| Artifact | Delivery/compatibility responsibility |
| --- | --- |
| `images/dashboard.oci` | The running backend is in this image. Verify its source/config digest and update the target's actual image pin through the approved recipe. A separately staged `soda-dashboard` binary does not replace this service. |
| `rootfs/usr/local/libexec/soda/soda-host` and `soda-runners` | Paired management lock/protocol owners. Stop new management admission and wait for old CLI/helper operations before replacement. Restarting `soda-host` affects project/browser terminal integration too; declare that interruption. |
| `rootfs/usr/local/libexec/soda/soda-runner-launch` | Matching Forgejo-only launcher. It is not a management-lock owner. Do not restart existing listeners merely to replace this file. |
| Canonical Forgejo custom payload/assets/notices | Use `forgejo-payload.json` and the sealed export, not a second handwritten file list. Account for native template reload/service interruption in the maintenance scope. |
| Existing runner units, sysusers/tmpfiles and package requirements | Inspect actual effective confinement and `rpm` versions. Change only separately reviewed deltas; do not recreate accounts, assign new UIDs, rewrite descriptors or upgrade clients incidentally. |
| Obsolete installed `soda-runner-helper` | Explicit retirement only during approved maintenance after all old writers exit. Its absence in new source/bundles does not remove an installed independently callable writer. |
| Cockpit Runners and Tailnet | Apply the combined plan's retirement gate; ordinary runner maintenance does not remove either operator payload. |

## Browser behavior

Shared page entry, session bootstrap, coordinated logout and native history are
owned by the page integration guide. The runner component additionally preserves:

- Provider links built only from the configured public Forgejo origin. A local
  runner ID is not a provider runner ID; do not invent deep links or expose stored
  internal registration URLs. Provider administration links do not grant authority.
- Exact-ID confirmations, clear boot-policy/destructive effects, field guidance and
  local-state uncertainty. Keep Refresh/provider inspection usable as safe next
  actions; do not equate configured listeners with available provider job slots.
- Request-generation retirement on departure/disconnect. Clear unsent credentials
  on pagehide, logout or authorization loss. BFCache return discards confirmations,
  revalidates the original actor/session and refreshes before mutation. Late responses,
  including those ignoring abort, cannot revive an earlier request generation.
- Dispatched-operation uncertainty: cancelling a browser fetch does not prove native
  cancellation. Do not replay an operation or clear uncertainty merely on remount.

## Validation ownership

Local runner tests own API admission before decode/native reads, strict requests,
secret/public-origin projection, unsupported providers, cross-process serialization
and cancellation-before-dispatch, whole-list failure and staged Remove/creation
failures. Preserve legacy data and both consumers' response/confirmation behavior.
The existing Go, browser and Cockpit tests remain with those production owners;
use the [source-check guide](typescript.md#local-source-checks) for commands.

The [runner native guide](runners-native-validation.md) owns exact provider jobs,
process-tree/lifecycle/overlap observations, reboot preservation and Remove/provider
aftermath. Its private input and per-effect contracts are not a second browser or
build harness. [Installation](installation.md#retained-sodaspaces-cutover) owns
maintenance procedures. The combined plan owns the retirement parity map and exit;
there is no separate seven-step runner rollout or editor-handoff requirement.
