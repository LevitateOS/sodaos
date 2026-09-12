# Tailnet in the native dashboard — implementation plan

**Status: stage-1 local investigation and stage-2 backend source contracts complete;
native UI, project runtime, installed proof and retirement remain pending.** The user requested native dashboard ownership
of host and project Tailnet configuration, automatic enrollment without a login per
project, and eventual stock Cockpit administration without Soda extension pages.
This document owns that feature's implementation order and acceptance criteria.
Planning is not permission to enroll devices, change networking/capabilities, stop
services, remove installed packages or clean up retained state.

## 1. Outcome, scope and owners

Projects inherit an **appliance-managed enrollment policy**, never the host's
Tailscale device identity, state directory or node keys. The host remains a
persistent device; each enabled project gets its own automatically enrolled,
ephemeral network identity while running. Browser lifetime does not own networking.
Persistent project roots, accounts, SSH identities and work remain persistent.

Recommended first-release decisions from the brainstorm:

- Keep ordinary browser sign-in for the host, with its existing persistent state.
  A one-use, non-ephemeral auth key is an optional unattended host-enrollment input;
  the host does not need a reusable credential to stay enrolled.
- Use one operator-configured **OAuth client credential**, restricted to auth-key
  creation for the selected project tag(s), for automatic project enrollment.
  Use upstream OAuth and Tailscale Go API clients in the existing host helper to
  create single-use ephemeral keys. The [stage-1 design](#stage-1-design-decisions)
  explains why the initially preferred host CLI/socket shortcut is not selected.
  No separate provider/token service. Host sign-in alone does not grant device-creation rights.
- Offer a managed-network default for new projects, explicitly enabled by the
  operator. Existing projects and old Create callers remain off until selected.
  Start re-enrolls an enabled project automatically; no repeated human consent.
- First release has one managed project network and an Off option, not arbitrary
  per-user networks, a credential vault UI or a network-profile framework. Normally
  the operator chooses the host's Tailnet; a label is not proof the credentials
  target it, and changing host login must not silently change project enrollment.
- Reusable + ephemeral auth keys are an **optional follow-up**, not a prerequisite
  for the OAuth path. They require the same credential-containment proof and an
  expiry/rotation UX; do not pass them to developer-controlled daemons.
- “Stock Cockpit” means no Soda Runners/Tailnet extension pages. Preserve upstream
  administration, root-only PAM/SELinux transition, private socket access, existing
  branding and the Accounts navigation policy. Resetting those is separate scope.

Requirements remain with their owners:

| Owner | Responsibility |
| --- | --- |
| [Architecture](architecture.md) | Product, identity and host/project security boundaries |
| [Native pages](forgejo-soda-pages-plan.md) | Forgejo host, enumerated entry/OAuth returns, logout, navigation and asset lifecycle |
| [Sodaspaces settings](sodaspaces-plan.md#settings-pages-and-os-selection) | Placement and existing Create/project administration |
| This plan | Tailnet UX, enrollment/state/lifecycle design, delivery stages and parity |
| [API](dashboard-api.md#tailnet-backend) | Implemented backend request/response/error models and stage boundaries |
| [Credentials](dashboard-credentials.md) | Shared OAuth/schema migration, consistent backups and compatible restoration |
| [Project OS](project-os.md), [project services](project-services.md) | Persistent roots, native accounts, runtime and networking confinement |
| [Cockpit](cockpit-port.md) | Retained stock administration/security configuration |
| [Runner combined plan](native-pages-runners-plan.md) | Separate runner retirement; this work neither reopens step 5 nor grants runner removal |
| [Installation](installation.md#retained-sodaspaces-cutover), [native support](native-support.md#build-and-artifact-contract) | Maintenance and exact-revision build/check/export |
| [Handoff](implementation-status.md), [AGENTS.md](../AGENTS.md) | Actual state, grants, preservation and reporting |

No public ingress/Funnel, Tailscale SSH replacement, automatic ACL synchronization,
project deletion, idle-stop policy, Runner OS work, general network controller or
new release/update mechanism is added. Subnet-router configuration remains an
explicit operator/native workflow; moving the host page does not advertise routes.

## 2. Actual starting point and upstream findings

| Inspected source | Consequence for implementation |
| --- | --- |
| `cockpit/src/tailscale/native.ts`, `store.ts`, `status.ts`, `stream.ts`, `cockpit/src/pages/TailscalePage.tsx` and their tests | Existing host-only CLI/LocalAPI adapter: status/peers/preferences, initial login and reauthentication, exit-node selection/LAN preference, exit-node advertisement and Forgejo refresh. The adapter hardcodes the host socket. Port behavior, not React/Cockpit transport. |
| `internal/tailnet/`, `cmd/soda-tailnet/`, `cmd/soda-forgejo-tailnet/` | Existing identity/diagnostic/advertisement callers remain unchanged. Stage 2 adds a separate opt-in management adapter and native policy owner, not a change to the read-only CLI contract. |
| `internal/web/runners.go`, `auth.go`, `pages.go`, `frontend/spaces/soda-connection.ts`, `assets/branding/forgejo/soda-native-page.ts` | Reuse native operator admission, fixed return handling and Lit page lifetime; no new password authority or dashboard shell. |
| `internal/web/environment_authority.go`, `management.go`, `environments_api.go` | Current repository/organization ownership and explicit Soda operator authority govern lifecycle. Creation remains human repository-owner-only. Mere repository visibility is broader than membership and must not expose private network metadata. |
| `internal/host/daemon.go`, `management.go`, `appliance/services/soda-project@.service` | The helper owns fixed native operations and cancellable admission; systemd owns Create/Start/Stop/boot and failure restart of the existing container. A browser-only enrollment hook would miss boot/native starts. |
| `project-os/Containerfile`, `project-os/rootfs/`, project creation flags | Tailscale is not installed in the project image. Existing projects have their own network/user namespaces; selected new profiles include namespace-scoped NET_ADMIN but no explicit `/dev/net/tun` grant. Retained roots are not interchangeable with today's profile. |
| `internal/store/migrations.go` | Stage 2 appends schema v10 for Tailnet OAuth returns, with local preservation/refusal tests. Installed targets remain v9; see the credential owner for migration/backup scope. |
| `cockpit/`, root `package.json`, `scripts/stage.py`, `internal/nativebuild/bundle.go`, installed operator tests | Tailnet still owns the remaining Cockpit workspace/build dependencies and a required bundle page. Eventual removal must update actual callers/inventories, not just delete its directory. |

Public research from the preceding brainstorm is retained in
`.artifacts/tailnet-enrollment-TDPVW1/`. It is upstream research, not installed proof:

- [Auth keys](https://tailscale.com/kb/1085/auth-keys): reusable and ephemeral are
  independent properties. Auth keys expire within 90 days; expiry/revocation of an
  enrollment key does not itself revoke already enrolled devices.
- [OAuth clients](https://tailscale.com/kb/1215/oauth-clients): scoped client
  credentials can create tagged auth keys without repeated human login. Do not
  request general device-write, policy-write or DNS-write merely for enrollment.
- Inspected [`up.go` at v1.102.4](https://github.com/tailscale/tailscale/blob/v1.102.4/cmd/tailscale/cli/up.go)
  and [`feature/oauthkey/oauthkey.go`](https://github.com/tailscale/tailscale/blob/v1.102.4/feature/oauthkey/oauthkey.go):
  the CLI accepts secret-file inputs, resolves an OAuth secret to a non-reusable
  auth key, and sends the resulting key to LocalAPI. Ephemeral defaults true;
  preauthorization is a distinct choice, not implied permission to bypass approval.
- [Ephemeral nodes](https://tailscale.com/kb/1111/ephemeral-nodes): fresh identities
  can have new addresses; orderly logout and eventual inactive-device expiry differ.
  In-memory state has native shutdown behavior that must be checked in the selected
  runtime. Do not promise exact crash-expiry timing or stable names/IPs.
- [Container parameters](https://tailscale.com/docs/features/containers/docker/docker-params):
  upstream has separate state/socket, OAuth/file-input and userspace/kernel modes.
  SOCKS/HTTP or ingress-only forwarding is not transparent whole-project networking.
- [OAuth-app device provisioning](https://tailscale.com/docs/features/oauth-apps/device-provisioning)
  was documented as alpha, same-tailnet and one-consent-per-device without a refresh
  token. It is not the unattended enrollment mechanism selected here.

Soda currently provisions Tailscale from its stable Fedora repository; this research
**does not pin or upgrade the installed client**. The
[stage-1 receipt](implementation-history.md#tailnet-stage-1--local-runtime-and-enrollment-investigation)
separates builder observations, retained package evidence, upstream source and local
synthetic checks. Investigated versions are not a shipping compatibility claim;
adopted Go dependencies and companion image identities must land in actual source
manifests/build inputs when implementation selects them.

## 3. User experience

### Global Tailnet page

Add operator-only **Tailnet** beside Runners, rendered at the proposed native
`/?soda-view=tailnet` destination. Extend the existing fixed bookmark bridge,
OAuth return and native-page registry through the page owner. A raw rendering URL
is not a universal signed-out entry. Keep native Forgejo navigation/sign-out.

Two separately headed sections, never a scope selector that can accidentally apply
a host action to a project:

```text
Tailnet

Appliance
Connected · soda-office · [observed Tailnet]
Persistent connection
[Manage connection]

Automatic project access
Enabled · [managed network]
Enrollment: OAuth client
Devices: Ephemeral while projects run
New projects: Connect automatically
[Configure]
```

Host details retain status, addresses, relevant health, peers/eligible exit nodes,
exit-node/LAN preferences and advertisement approval guidance. Sign-in opens the
validated native Tailscale authentication URL; it is not another Soda login.
Reauthentication preserves native preferences and unrelated advertised routes.
Add explicit host logout only with a disruptive-action confirmation and a known
alternative management path, or an honest warning when none is verified.
An unenrolled host must remain manageable through its existing approved private
route or console. Keep native CLI bootstrap/recovery available; do not require an
already connected Tailnet, a public listener or a new browser origin to configure it.

Forgejo Git SSH advertisement refresh becomes an **explicit action**, with its
potential Forgejo restart warning. Current Cockpit `syncForgejo()` can mutate on
observation; do not reproduce that side effect in GET, polling, focus or page entry.
Keep successful host enrollment and failed advertisement refresh as separate facts.
Never rewrite browser/OAuth origins or listeners from an observed Tailnet name.

Automatic-enrollment setup:

1. Explain private-network exposure, the project tag and who owns Tailnet policy.
2. Guide the operator to native Tailscale credential setup with minimum permissions;
   collect the credential once through a protected field, never display it again.
3. Check credential usability without registering a hidden test device. Report
   `configured`, `credential check passed`, and `enrollment verified` separately.
   Show unknown scope/expiry/target facts honestly; do not infer them from a label.
4. Confirm the new-project default. Existing projects and connections do not change.
   Enabling an existing project is its exact-ID action, not a background bulk sweep.

Rotation within the same verified network/tag binding affects future enrollment,
not existing node identities. Replacing the network or broadening tags is a new
explicit selection, not ordinary credential rotation. Disabling new enrollment,
changing the default and disconnecting existing nodes must be separate controls.
No automatic provider credential revocation or device deletion accompanies Save.

### Create and project Network panel

The existing Create form offers **Use appliance-managed Tailnet** or **Off**. Its
initial choice reflects the current operator default, with a visible network label
and revision. Submit the choice explicitly; an old caller omitting it retains Off.
Resolve supported runtime availability before offering it. Changing a selector does
not enroll or create anything, and a default change between preview/submit requires
refresh rather than silently selecting another network.

Reuse one project-scoped Network panel in the existing repository Spaces settings
and project workspace details, not another settings application or repository tab:

```text
Tailscale                         Connected
Network        [verified observed network]
Managed by     This appliance
Connection     Automatic · while project runs
Device/address [current observed values]
[Copy SSH command] [Disable Tailscale]
```

Spaces cards show a compact status linking to that panel, not full peer inventories
or credential forms. Ordinary members see only their permitted project's metadata;
configuration controls use current environment administration. Copy SSH uses the
member's original Linux login and existing trusted project SSH identity. Tailnet
membership does not join a Soda project or install SSH keys. Conversely, Soda
membership does not grant Tailnet connectivity or authenticate HTTP/database apps.

States: Off; waiting for project start; enrolling; approval/signature required;
connected; expired/credential attention required; runtime unsupported; unavailable;
operation unconfirmed. Keep project-running state separate from Tailnet status.
Show last observations as stale and suppress usable-address claims when unavailable.
An address or healthy daemon is not a client reachability receipt.

Enable on a stopped project records policy only; it must not start the project.
Disable means both “do not auto-enroll on the next start” and an explicit attempt to
end this project's current connection. Warn about interrupted SSH/service sessions.
Preserve failed-disconnect uncertainty separately from the successfully saved Off
setting. Retry is an explicit exact-project attempt after observation, never a
browser reload or polling side effect.

## 4. Authority, credentials and native state

### Admission and data minimization

- Global reads/writes: configured stable Soda operator, not Forgejo site-admin or
  Linux wheel status. Root CLI/console authority remains a separate native path.
- Project configuration/retry: current `environmentAdministrator` semantics or the
  explicit Soda operator, with fresh acting-user/repository/session checks. Do not
  authorize from `projects.owner_id` or the creation-time container owner label.
- Project network reads: own existing membership with applicable fresh access, or
  current environment administration/operator authority. Public repository visibility
  alone is not authority to inspect Tailnet addresses, peers or credential status.
  Copy-connection remains own-membership-only. Add focused transfer/rename/denial tests.
- Reuse expected-actor, Origin/CSRF, strict bounded JSON and current-session admission.
  Fixed helper operations resolve original project/container identity server-side;
  no caller-selected socket, namespace, PID, UID, unit, executable or host flag.
  Connection/lifecycle actions resolve tags from the saved enrollment binding.
- Only the dedicated operator configuration flow chooses the validated enrollment
  binding/tags. Project admins may enable/disable that binding; they cannot enlarge
  its permissions. Provider policy
  remains Tailscale-owned, not a copied ACL database. `tag:soda-project` alone is not
  isolation: accepted Tailnet grants must also limit project access to host/peers.

Project root/wheel still has the existing trusted-team power over project processes,
files and networking. No claim of hostile-tenant isolation or atomic synchronization
of web ownership, native wheel/SSH rights and Tailnet grants is introduced.

### One owner for each kind of state

| State | Proposed owner and behavior |
| --- | --- |
| Host device identity and preferences | Existing native `tailscaled` persistent state; never clone, reset or move it into the dashboard/project database |
| Enrollment secret | Restricted root-only host storage, outside all project mounts and browser-readable config; one canonical owner, no duplicate saved secret in SQLite or environment variables |
| Global enrollment/default policy | Small versioned root-owned configuration with an opaque binding/revision, allowed tags and safe credential reference; dashboard receives only a sanitized projection |
| Per-project Tailnet setting | Root-owned, versioned policy keyed by validated project ID and original native identity; canonical source for boot/native startup, not a duplicate desired-state flag in the Soda DB |
| Active project node, socket and command attempt | Exact run-owned runtime state and native supervision, separate from the persistent project root; bind to the current container/network-namespace incarnation |
| Soda sessions and repository associations | Existing store remains authoritative; Tailnet does not become another identity database |

Prefer extending `internal/tailnet` with concrete Go operations and the existing
host client/daemon routes. Reuse native CLI/LocalAPI and systemd/Podman mechanisms;
no generic credential broker, job queue, reconciler or network-provider abstraction.
Missing unconfigured policy means Off; malformed or ambiguous configured state means
unavailable, not permission to regenerate it or enroll with defaults.

The existing host helper resolves the root-only OAuth credential through upstream
OAuth/Tailscale Go clients, addressing the explicit saved Tailnet rather than `-`.
Only the resulting single-use key goes to an appliance-owned companion's CLI and
daemon. Neither command executes in the developer-controlled Project OS root.
The [stage-1 design](#stage-1-design-decisions) owns that transport decision. Reject
credential-input URL options such as `baseURL`; derive options from validated
server-owned policy and keep provider/control endpoints fixed. No reusable enrollment
credential crosses into either container. Never copy host Tailscale state into an image/root.

Secret entry is transient protected request data with no-store responses, cleared
on serialization, failure, page departure and authorization loss. Native auth URLs
are credential-bearing too: validated HTTPS origins, no referrer leakage, browser
storage, telemetry, screenshots or logs. Bound captured stdout/stderr while reading;
complete-or-error JSON, sanitized public errors and no raw provider response dumps.
Use secret-file inputs, not argv or container environments; preserve owned-child
cleanup without treating killed observers as proof of native cancellation.

Credential/config updates use checked ancestry, restrictive modes, atomic publication
and a revision check under the owning lock. Preserve prior inputs under authorized
maintenance rather than implicitly revoking/deleting them. Native state readback,
metadata publication and provider effects can fail independently; expose those
outcomes without rollback or blind retry. Restoration of host identity/credentials
requires the existing maintenance/backup contract; never run a cloned node during
copied-state rehearsal.

### Project runtime and lifetime

**Selected design for native proof:** an appliance-owned upstream Tailscale
companion joining the exact project's **user and network** namespaces through
Podman. It has its own filesystem, PID/IPC/UTS/cgroup namespaces and lifecycle
identity; no project root, host engine socket or reusable credential mount. Native
execution remains unproven, including TUN, SELinux/seccomp and older retained profiles.
The [stage-1 decisions](#stage-1-design-decisions) specify the supported upstream
mechanisms and the remaining proof, not an installed capability grant.

Provide ordinary inbound OpenSSH/service access and intended outbound Tailnet access;
do not substitute SOCKS-only or ingress-only proxying while claiming full networking.
No host-network mode, privileged parent or unrelated namespace access. Do not alter
all project capabilities or recreate roots. An incompatible retained project stays
unchanged and reports unsupported until a separately approved compatible path exists.

Use systemd's native project startup/stop/boot ownership, not browser presence or a
background scan of every project. The per-project policy survives restart; ephemeral
node identity does not need to. No second copy of the project's running/boot-enabled
state is added. Inspect actual unit dependencies and test these transitions:

| Event | Required effect |
| --- | --- |
| Explicit Create with managed access | Bind the reviewed enrollment policy to the new reservation/native identity; provision once. Project creation and Tailnet outcome are separate results. |
| Explicit Start, enabled host boot, native failure restart | Start the same container; initiate one bounded enrollment for its new runtime incarnation if policy permits, without a Forgejo browser session. |
| Already connected, refresh, focus, visibility or repeated Start | Observe/reuse the existing node; never mint another key or force reauthentication. |
| Enable while running | One explicit enrollment; reject duplicate/concurrent attempts for the same incarnation. |
| Disable, project Stop or normal host shutdown | Prevent new enrollment, perform bounded native logout/stop for that exact runtime, then release only its owned runtime resources. Never block project shutdown indefinitely on provider reachability. |
| Crash/abrupt disappearance | Native supervisor ties the daemon to the original namespace/process identity. Report unconfirmed provider removal; Tailscale may expire the offline device later. |
| Failed enrollment or failed readback | Retain the project, policy and outcome evidence; report network failure/unconfirmed state. Do not recreate or roll back the project. |
| New credentials or host login/logout | No implicit logout, migration or reauthentication of existing project nodes. Project enrollment policy is separate from host identity. |
| Dashboard/helper restart, or project daemon restart within the same live project | Re-observe the actual owned runtime. Preserve/reuse its available ephemeral state; lost or ambiguous enrollment state does not authorize another key automatically. No network lifetime is tied to the dashboard process. |

Soda-issued operations need cancellable serialization and incarnation/revision checks
across enable/disable/Start/Stop/rotation. Preserve the existing helper admission and
terminal/runner paths; do not hold its global project gate while waiting for a human
login or a provider network round trip. Tie bounded work to native supervision and
retain enough safe attempt state to inspect an uncertain result before another key
is minted. Automatic enrollment on a *new* runtime is intended; automatically
replaying an uncertain attempt in the *same* runtime is not.

The host's normal Tailscale reconnect and control-plane behavior remains native.
External root CLI changes are not prevented by a Soda UI lock: use field-specific
preference updates, preserve unrelated fields and observe the resulting state.
Do not advertise a global compare-and-swap guarantee against noncooperating writers.

## 5. Implementation stages and exits

Stages are coherent delivery slices, not a command sequence to repeat per edit.
Host UI work can proceed while project runtime feasibility is investigated. Do not
make unfinished desktops/runner jobs/installer roadmaps new Tailnet prerequisites.

### Stage 1 — resolve the native integration and security decisions

**Local investigation completed.** The design and native proof proposal below
replace the earlier open runtime/CLI assumptions. No production handler, unit,
image, policy writer or enrollment integration has been implemented.

Inspect selected installed-package evidence and matching upstream source; record
compatible versions in source manifests/recipes when selected, not just this plan.
Reuse `.artifacts/tailnet-enrollment-TDPVW1/` research where applicable. Resolve:

1. Host native login initiation/status/auth-URL and preference API behavior, including
   bounded human-wait handling without a new durable OAuth-login job subsystem.
2. Host-side OAuth resolution and project socket targeting; proof that neither
   project root nor a substituted daemon/socket can obtain the reusable credential.
3. Project daemon placement, namespace/TUN/DNS/egress behavior, exact systemd lifetime
   and original-identity checks, including existing roots with older creation flags.
4. How minimum-scope credentials and native observations establish the intended
   Tailnet/tag binding. A successful token request is not proof of matching Tailnet,
   tags or permission to enroll. If additional read scope or a first real enrollment
   is necessary, specify it explicitly; no hidden test node or invented verification.
5. Device-approval and Tailnet Lock behavior. Automatic preapproval is an explicit
   provider-authorized setting; unsupported signing requirements fail closed with
   actionable guidance, not copied signing keys or widened scopes.
6. Managed-network replacement/rotation semantics and unsupported-version behavior.
   Reject silent cross-Tailnet rebinding or broader tag policy on future starts.

**Exit:** reviewed concrete runtime/credential design, testable interfaces and exact
remaining native proof/effect proposal. Source research alone is not permission for
a new VM, real enrollment or a capability/network change. A concrete unsupported
mechanism comes back for a decision rather than a dormant alternate backend.

#### Stage-1 design decisions

**Host login:** retain the existing socket and persistent daemon state. For an
existing node, use the current masked `WantRunning` update and, when required,
`POST /localapi/v0/login-interactive`; preserve unrelated preferences. For first
login, retain upstream `tailscale up --json`, but set a short native wait timeout
and an enclosing process deadline. Decode bounded JSON notifications with Go,
then observe the daemon's status/AuthURL. A wait timeout with a pending native
login is not cancellation of that login. Never hold the helper's global project
gate or a browser request until a human finishes. No new durable login worker.

**Companion rather than a custom namespace launcher:** Podman supports both
`--userns=container:FULL_CID` and `--network=container:FULL_CID`. Sharing only the
network namespace is insufficient for the proposed capability boundary. Resolve
FULL_CID with Soda's existing project validator, then add an incarnation observation
(PID/start identity, user/net namespace identities, mappings and native unit).
Recheck both project and companion before dispatch; labels/CID alone do not detect
a restart of the same persistent container. An unowned `tailscale0`, socket,
companion occupant or resolver owner is a conflict, not an attach/reset target.
Use Podman's namespace joining, not browser-supplied `/proc` paths, a bespoke
`setns` supervisor or `NetworkNamespacePath` without the owning user namespace.

The native proof recipe is an immutable upstream Tailscale image, direct
`/usr/local/bin/tailscaled` entrypoint, kernel `tailscale0`, `--cap-drop=ALL` plus
project-user-namespace `NET_ADMIN`, and only `/dev/net/tun` as the added device.
Keep private PID/IPC/UTS/cgroup namespaces, default seccomp and the existing selected
outer-profile SELinux-label policy; no `--privileged`, SYS_MODULE or host namespace.
Do not inherit containerboot enrollment/environment defaults. No image digest or
installed compatibility is asserted yet; native staging must resolve/record the
selected image, CLI/daemon versions and actual image contents. The published image
owner is upstream `build_docker.sh`/mkctr, not its explicitly non-production Dockerfile.
New companions use upstream `--no-logs-no-support` (including its loss of upstream
technical support), no container log capture and no raw attached daemon output in
the journal. Upstream daemon diagnostics can contain auth URLs. Emit only sanitized
helper outcome/health projections; do not change the existing host daemon's logging
or treat suppressed log uploads alone as stdout redaction.

Only new, exact-run control/state directories are mounted into the companion.
Their mapped ownership is derived from verified project mappings; never apply
recursive `:U` chown or mount a project root. A root-only host ancestor prevents
ordinary host access. Keep the companion's small writable runtime root for native
DNS backup/recovery; a blanket read-only root would break the selected DNS manager.
Project root retains its existing trusted-team ability to alter its own network,
but must not see companion processes, filesystem, control socket or enrollment
credentials. Prove this with a substituted socket/daemon and project-root attempts,
not just file-mode inspection.

**Why the original host CLI shortcut is not selected:** Linux translates Unix
socket peer credentials into the receiving user namespace. Appliance UID 0 is
outside the project's shifted mapping; Tailscale's LocalAPI write admission checks
root/operator identity in the receiving namespace. A host-root CLI pointed at the
companion socket therefore is not a valid write-authority design. The local CLI
probe proves file input and socket selection only, not cross-user-namespace access.
Do not repair this with host-user-namespace containers, an overflow-UID operator,
a project-readable OAuth file or a new privileged socket proxy.

Use the official `tailscale.com/client/tailscale/v2` key API in the existing Go
helper instead. Upstream's older exported client is deprecated; its internal OAuth
resolver is not an importable integration API. The concrete sequence is:

1. Admit the saved binding/revision and exact project incarnation under its owning
   lock; record the attempt before any provider call. Read the restricted host
   client ID/secret only in that helper.
2. Use upstream `clientcredentials.Config` with the operation context, bounded
   HTTP transport, fixed token endpoint, `auth_keys` scope and selected `tags`
   parameter. The v2 SDK's convenience OAuth wrapper uses a background token
   context; do not use that wrapper for a bounded operation. No token cache/service
   or reusable token in project/runtime metadata.
3. Call `Keys().CreateAuthKey` with the explicit saved Tailnet identifier (never
   empty or `-`), non-reusable, ephemeral, the exact tags, short key expiry and the
   separately chosen preauthorization bit. This prevents a replacement credential
   from silently selecting its own different Tailnet. Check returned key metadata;
   missing/mismatched capabilities or an uncertain response do not authorize another
   key or local enrollment. Do not add automatic POST retries.
4. Publish only that single-use key to a new restrictive file in the companion's
   exact-run input directory. Invoke its fixed upstream CLI as container UID 0
   through host-owned Podman exec, targeting its own fixed socket and using `file:`
   input. The companion is appliance-owned, **not** the developer's Project OS;
   neither container receives the OAuth secret or provider access token. Validate
   native exec completion, not just termination of its host-side observer.
5. Observe Self/device ID, applied tags, expected native Tailnet identity, approval,
   Lock and daemon state through the same trusted companion. Only then publish the
   connected projection. A successful key response is not proof of registration,
   reachability or provider-policy isolation.

This is a small existing-helper operation using supported upstream clients, not a
new key-minting service or copied provider implementation. The SDK reads response
bodies without a size cap and exposes provider error text; its selected transport
must enforce streaming byte/deadline limits, reject redirects/unexpected status
and endpoints, and map errors without logging raw bodies. Local SDK fixture passes
are not tests of that still-unimplemented Soda boundary.

**Scope and binding:** baseline credentials need only `auth_keys` for the chosen
tags. A token request can check usability/requested scope, but cannot prove the
Tailnet binding. Save remains `configured`/`credential check passed`; the first
explicitly enabled real project verifies enrollment. Every key request names the
expected Tailnet before any device registration, so no hidden test node or new DNS,
policy or device-write scope is needed. Provider refusal leaves the project intact.
Use the provider's Tailnet identifier, not a friendly UI label; reject path syntax
and implicit-default selectors. Credential rotation must preserve that identifier,
tags and policy revision; a different network/tag set is an explicit new binding,
never transparently adopted by already-enabled projects. Existing connections do
not change when credentials rotate or the appliance logs into a different network.

Device approval is separate: `preauthorized=false` remains the default. An
approval-required Tailnet cannot promise unattended connectivity unless its operator
explicitly selects provider-permitted preauthorization. Tailnet Lock is not an
automatically supported signing path: report signature-required/unavailable, never
copy host signing keys or expand credential scope. Native proof must establish the
actual approval/Lock observations; do not infer them from token success.

**DNS:** Podman's container-network join normally bind-mounts the original project's
`/etc/resolv.conf`, not merely a copy. Tailscale's native direct DNS manager supports
bind-mounted resolver files, backup and restore. This gives a plausible ordinary
project MagicDNS/outbound path without a Soda DNS server or modifying old roots to
install Tailscale. Use `--no-hosts` for the companion so its creation does not append
its hostname to the project's shared `/etc/hosts`.

Before offering managed access, validate the actual resolver mount/owner and that
native Tailscale may manage it. Custom project resolver mounts/managers are not
silently overwritten. Native DNS recovery must precede loss of the companion's
backup/root. Its generated file declares native ownership; hand edits retaining the
Tailscale marker are not protected by an upstream compare-and-swap. Do not claim
arbitrary-edit preservation or repair DNS by blindly restoring an old snapshot.
Prove normal/public DNS, MagicDNS, nested-workload behavior, conflicting owners and
stop/crash recovery. If the selected profile cannot meet that contract, report it
unsupported rather than claim SOCKS-only or IP-only behavior is full integration.

**Ephemeral state:** use a native state file in a restricted host `/run` directory
bound to the project incarnation. The key makes the device ephemeral; diskless
`--state=mem:` is not required. This preserves the node across a companion-daemon
restart during the same project run. A new project run uses a fresh directory/node,
not an old retained copy. A missing/ambiguous state file in the same run requires
inspection or explicit retry, never another automatic key.

Unlike `mem:`, this file-backed mode does not set upstream's `LoginEphemeral` flag
that triggers best-effort shutdown logout. The native stop path must explicitly
attempt logout when ending project access; stopping a daemon alone only makes the
provider device offline. Do not logout merely to restart the companion within the
same project run. Native logout failure and eventual provider expiry stay distinct.

**Systemd lifetime:** keep `soda-project@` as the only project running/boot owner.
Its start-post hook queues the exact companion unit with `--no-block`; it must not
call back into the helper's held global Create/lifecycle gate or wait for enrollment.
That hook runs for a new unit activation/failure restart, not an already-active
Start. The companion uses `BindsTo=` and `After=` on that exact project unit; its
bounded preparation waits for and binds the actual container incarnation, not
just Type=simple's early active indication. Provider work remains in the companion
unit's fixed native phase, outside the helper's global gate.

Stop ordering must end the companion before the project's network disappears.
Logout is for Off or the end of the project run, not a same-run daemon restart;
use saved policy and the actual owning systemd stop job, not `ActiveState` alone
(which may still be active while an ordered stop is queued). Preserve per-run state
on daemon failure/restart, inspect incomplete attempts, and do not mint twice from
unit restart hooks. Exact stop-job detection and failure-restart ordering are
native proof items, not established by reading the unit manual. No `Upholds=` loop,
browser polling mutation or scan/re-enrollment of every project.

**Testable implementation seams:** concrete Go operations for host
observe/login/preference/refresh; credential check and atomic binding update;
project policy selection; exact-run prepare/enroll/observe/disconnect. Keep provider
key creation separately observable from LocalAPI enrollment and readback. Inputs
are fixed IDs/revisions/actions; only root-owned state resolves units, CIDs, sockets,
files and native arguments. Stage 2 adds their DTOs/failure tests to the API owner;
there is no reason to create a provider interface hierarchy or new task queue.

#### Remaining native proof proposal

No target or action below is authorized yet. Use one **new isolated native x86_64
fixture**, not either retained appliance. Before execution, bind its exact name,
base/candidate, package/image identities, two newly created project IDs/roots,
client route and preservation inventory in the existing support/installed inputs.
Refuse occupied names/state. No sibling-architecture gate or retained-root retrofit.

| Phase | Exact proposed effects and assertions |
| --- | --- |
| A — no provider credentials | Approved fixture provisioning/start, two disposable projects and their companion units; TUN/NET_ADMIN only in the specified companion profile. Prove actual UID/net namespace ownership, private PID/mount/IPC boundaries, denied project-root access using synthetic canaries, host-root LocalAPI admission versus companion-local admission, fixed binaries/CIDs and no host engine exposure. |
| A — native runtime lifecycle | On those fixtures only: project Start/Stop, same-run companion restart/failure, project failure restart, duplicate Start and Off. Validate queued systemd stop jobs, reverse stop ordering, changed namespace/PID rejection and retained state. No auth key/device may be created. Resolver mount ownership and safe failure behavior are observed without claiming registered networking. |
| B — separately approved real enrollment | A restricted client-ID/secret input for one explicitly named Tailnet/tag policy, approved client/peer endpoints and approval/Lock policy. At most four single-use keys/devices: initial A and B, then A after one clean project restart and one project failure restart. No hidden test device, same-run daemon-restart enrollment or retry after an uncertain key outcome. |
| B — connectivity and isolation | Ordinary project SSH/PTY/SCP/SFTP plus one project service, intended outbound Tailnet traffic, public DNS/MagicDNS and permitted/denied policy paths. Preserve ordinary LAN/terminal paths. Verify A/B independent identities, actual tags, approval/signature states, and unchanged appliance identity/preferences. Do not enable Tailscale SSH, exit-node/subnet advertisement or route acceptance. |
| B — ending access | Explicit logout/stop of those exact project nodes, one Disable with no subsequent re-enable, bounded failure handling and fresh readback. Record confirmed logout versus uncertain removal/delayed expiry. Preserve project roots, credentials and attempt evidence. |

No host logout, host route/advertisement change, host reboot, ACL/tag creation,
provider device-delete, retained-target contact, fixture teardown or resource
cleanup is included. A negative real-credential or Lock case requiring another
Tailnet/credential/policy change needs its own input/grant; the synthetic refusal
tests do not justify such a mutation. If a namespace, resolver or supervisor
mechanism fails, stop that case and report the concrete limitation; do not add
host privilege, rebuild roots or replay registration to make the observer pass.

### Stage 2 — Go contracts, state and authorization

**Completed source-only slice.** `internal/tailnet` now owns host management and
versioned native policy/credential storage; `internal/host/tailnet.go` exposes fixed
methods and `internal/web/tailnet.go` supplies protected models/handlers. Concrete
requests, outcomes and limits belong to [the API owner](dashboard-api.md#tailnet-backend).
The existing `soda-tailnet` and advertisement callers/error semantics are preserved.
One native policy lock serializes cooperating writers without entering the project
lifecycle gate; no raw preference, state or credential proxy was added.

The root helper's optional `tailnet_management` configuration defaults false.
Host LocalAPI and CLI-dependent effects check the reviewed release recorded in
`ManagementCLIRelease`. Root-only credential checking uses pinned upstream
`clientcredentials` with bounded fixed-endpoint transport; it creates no auth key or
device. The Tailscale key-creation SDK/runtime remains Stage 4, not a synthetic
success path in this slice. Enabling managed defaults or project enable/retry refuses;
project Off saves intent but cannot claim disconnection.

Schema v10 appends the enumerated bookmark/OAuth return without dropping pending
contexts/grants or loosening mixed-destination checks. Storage, completeness probing,
backup and refusal requirements belong to [the credential owner](dashboard-credentials.md#tailnet-credentials-and-schema-v10-return).
Project policy is not a second SQLite lifecycle authority; native run-incarnation
and supervisor state is still Stage 4 work. The bookmark/callback enum does not
register a Lit page or advertise unfinished navigation.

Focused Go tests cover authority/context loss, current ownership/membership, strict
input/secret projection, preserved native preference operations, provider bounds,
concurrent CAS/cancelled locks, unsafe files, replacement CID refusal and uncertain
publication/readback. See [the source receipt](implementation-history.md#tailnet-stage-2--backend-state-and-authorization)
for checks actually run and limitations.

**Exit met at source scope:** disabled-by-default backend contracts, no real
enrollment, UI, runtime hooks, target contact or installed acceptance.

### Stage 3 — native host Tailnet UI and parity

Add Lit Tailnet modules and the native page/entry/bookmark/navigation registrations.
Update canonical templates, `forgejo-payload.json`, browser build inputs, localization
and the graph-wide presentation epoch through their existing owners. No React mount,
iframe, copied login harness or replacement Forgejo header.

Port Cockpit behavior tests before removing their owner. Required coverage map:

| Existing behavior | Replacement checks |
| --- | --- |
| Host status, peers, exit-node availability/selection | Typed complete-or-unavailable Go projection and actual emitted-page tests; stale/missing selected node is visible |
| Initial sign-in, HaveNodeKey resume/reauthentication, split auth JSON | Native operation tests and browser pending/approval/error UX; use Go/native JSON decoding rather than copying a custom framing engine unnecessarily |
| Preserve unrelated native preferences/routes | Field-specific operation tests plus installed before/after comparison |
| Draft edits versus polling/readback, duplicate saves, retired callbacks | Native Lit generation/busy/draft tests, including responses ignoring abort and BFCache/actor loss |
| Successful write plus failed refresh, provider diagnostics | Independent saved/unconfirmed/read-error presentation, bounded sanitized diagnostics, no automatic replay |
| Forgejo advertisement success/failure/retry | Explicit action tests and preserved origins; passive page/GET reads dispatch zero refresh operations |
| Exit-node approval and safe authentication links | Provider-owned approval guidance and safe link/secret handling; neither online status nor advertisement claims routed traffic |

New host-disconnect confirmation, credential-entry and operator-only navigation
need their own tests; they are not inherited Cockpit proof. Keep Cockpit Tailnet
installed/source-present through replacement acceptance.

**Exit:** working native host controls with reviewed source parity, keyboard/focus,
light/dark/narrow/wide presentation and the existing connection/logout contracts.

### Stage 4 — automatic enrollment and project integration

Implement the accepted scoped runtime and root policy from stages 1–2. Wire existing
Create/Start/Stop/native boot paths and the shared project Network panel; extend
connection metadata without changing existing LAN/SSH behavior or account authority.
A global default must be reviewed in the Create request and recorded once by the
native policy owner; missing legacy fields remain Off. Provisioning failures retain
the original reservation and do not leave an unbound auto-enrollment task.

Add the operator setup/rotation/default UX and project summary/actions. Support the
OAuth baseline first. Native enrollment failure does not undo successful project
provisioning or change the meaning of `ready`; expose separate Tailnet state.
Ensure project-root changes cannot select host targets, credentials, tags or binaries.

**Exit:** source-tested automatic lifecycle, secret isolation and multi-project
independence, with no browser login needed for configured project starts. Optional
reusable-key support remains out until equivalent containment/rotation is accepted.

### Stage 5 — native candidate and isolated proof

Build/check/export a clean candidate under the existing native contract. Reuse
`TestNativeConnectionFixture`, the page consumers, and the installed
`tests/installed/sodaspaces.ts` driver. Add a Tailnet scenario module and strict
phase inputs there; do not copy native login/consent, actor guards or evidence tools.
Keep root/stock Cockpit observations in the existing operator journey. Pure reads
must not retain the old advertisement-refresh effect as an unnecessary gate.

Native input/effect records bind exact candidate/architecture/target, trusted client
route, operator/member/denied actors, existing host identity/preferences, project
IDs and every preserved root, enrollment credential reference, expected Tailnet/tags,
approval/signing policy and exact allowed actions. Host logout/route changes,
project enrollment/disconnect, provider cleanup and reboot are distinct grants.
Never log tokens, auth URLs, node state, whole inspections or provider bodies.

Exercise the matrix below using explicitly authorized fixtures and actual network
clients. Do not mutate retained project capacity merely to fill a table. Reuse valid
unchanged mechanism evidence, but do not relabel runner or Docker documentation as
project-Tailnet proof. Validate x86_64 and aarch64 natively when selected; one is not
a prerequisite for an unrelated sibling build and neither proves the other.

**Exit:** candidate-bound source/build/export and bounded native functional/security
receipts, with failures, skips, topology and unsupported retained profiles explicit.

### Stage 6 — affected retained delivery, then Cockpit retirement

Use [installation maintenance](installation.md#retained-sodaspaces-cutover), not
first-install/setup replay. Select the minimal paired dashboard/helper/custom-assets/
unit/package delta; inspect effective units, installed clients and occupants, take
fresh appropriate backups and declare actual interruptions. Preserve host enrollment,
project roots/accounts/keys/work, existing runner state and all later writes.
No migration auto-enrolls existing projects or changes their capabilities/defaults.
A necessary runtime addition to an old root follows Project OS same-root maintenance;
if creation-time changes cannot be delivered safely, leave that root unsupported.

Host UI and automatic-enrollment delivery are separately observable outcomes. They
may share an approved candidate/window, but do not deploy incidental image changes
or claim source parity as installed acceptance. Retained fallback remains until
that target's host replacement and ordinary administration checks pass.

Then prepare one coordinated **Tailnet source retirement** candidate:

- Remove `cockpit/soda-tailscale/` and presentation/store/transport only after their
  applicable tests are at the retained Go/native-page/installed owners.
- Audit every remaining import. If this is the last custom Cockpit consumer, remove
  the unused workspace, Vite/build tooling and React/PatternFly/Zustand dependencies
  from actual manifests/lock. Preserve any still-used root test dependency and
  required attribution/notices; deleting a workspace is not a license-cleanup grant.
- Update root scripts/workspace inventory, tsconfigs, native build/metadata/staging,
  bundle required/forbidden payload rules, source/packaging tests and public journeys.
  Reject stale `soda-tailscale` output instead of copying an ignored old dist tree.
- Port `tests/installed/operator.ts` to a stock Cockpit bridge/page and retain minimal
  used test types outside the deleted workspace. Keep root/non-root PAM, SELinux,
  Services/Logs, private access and logout checks; native Tailnet uses its own shared
  scenario. Do not drop ordinary operator coverage because the custom page vanished.
- Preserve `internal/tailnet`, native CLI/advertisement logic, `tailscaled`, native
  services, host device/credential state and all unrelated Cockpit packages/config.

Build/check/export and rehearse the actual inventoried obsolete-package delta.
Obtain explicit per-target removal-delivery approval; remove only accepted obsolete
assets/links with unchanged occupants. Preserve custom/unexpected files for a decision.
Runner package removal still has the [combined plan](native-pages-runners-plan.md#6-retire-only-the-cockpit-runner-presentation)
owner; an explicitly reviewed window may include both, but this plan is no grant
for either. Unaccepted targets retain their respective fallbacks.

**Exit per target:** no Soda Cockpit navigation/package remains for the approved
scope; native host/project management and stock administration work; unchanged host,
runner, provider and persistent project state is verified. Record source removal
and each installed removal separately in the handoff/history.

## 6. Focused acceptance matrix

| Boundary | Required cases |
| --- | --- |
| Authorization | Configured operator/nonoperator site admin; current owner/org owner, member, unrelated/public-repository reader; rename/transfer, missing scopes, stale session and expected-actor mismatch. No unauthorized native read or dispatch. |
| Migration/state | Fresh/current v9 migration, pending OAuth/context/grant preservation, malformed/newer/incomplete schema; root policy missing versus corrupt; stale revisions; retained projects Off; no source-bundle/private-input contamination. |
| Host parity | Existing enrolled identity survives UI/delivery/reboot when approved; initial sign-in and expired reauthentication; drafts/poll races, exit-node/advertisement failures, explicit Forgejo refresh, no passive mutations; disruptive host actions and alternative access. |
| Enrollment credentials | Correct/wrong/revoked credential, unsupported version, invalid tag/scope, approval/Tailnet Lock requirement, rotation and network-replacement refusal. No reusable secret in project state/env/argv, API responses or captured output. Credential check creates no node. |
| Project lifecycle | New enabled/Off/legacy Create; enable stopped/running; Start/Stop and approved host boot/failure restart; current-incarnation deduplication; two concurrent projects; disable during enrollment; unrelated project's process/policy/credentials unchanged. |
| Partial/cancelled outcomes | Cancel before admission; timeout after key creation/registration/connection; response/readback/publication failure; tab departure/logout; no second key/node or automatic mutation replay. Explicit inspection/retry and retained reservation/state. |
| Confinement | Original CID/PID-start/namespace and socket binding, changed/unexpected occupant, project-root tampering, inaccessible host enrollment secret/control socket, namespace-scoped device/capabilities, no host network/engine exposure or unapproved routes. |
| Real connectivity | From an intended authorized client: ordinary own-account SSH/PTY/SCP/SFTP and representative project HTTP/database service; intended outbound Tailnet access and DNS; permitted/denied Tailnet policy paths; existing LAN/terminal paths preserved. No Tailscale SSH substitution. |
| Ephemeral aftermath | Observed native logout versus delayed crash expiry, exact provider device identity, no old-node resurrection/reuse across root copies; next start auto-enrolls, addresses shown freshly. No promise of free/unlimited ephemeral device usage. |
| Browser/payload/stock Cockpit | Native HTML/CSP, shared actor/login/logout and BFCache/no replay, inaccessible secret captures, responsive keyboard forms; exact staged assets/notices; stale custom packages rejected; remaining root/PAM/stock pages work. |

Use existing Go/package/race tests for changed concurrency, TypeScript/Lit/browser
checks and Python packaging tests at their affected scope. Full native checking
already invokes the source suite; a duplicate broad pre-native run is not a new
requirement. Local fixtures and authored failure tests are not installed/network
proof. Update the current handoff in place; detailed execution receipts belong in
history, not extra mandatory sequences in this plan.

## 7. Completion and remaining decisions

Implementation starts with stage 1's concrete runtime/security choices, then host
contracts/UI and automatic project integration. Do not claim the requested feature
complete after merely moving the host screen. Do not claim Cockpit is stock on a
target until both obsolete Soda extensions are actually retired there.

Stage 1 selected a companion/upstream-Go-client design, bounded host-login seam,
explicit Tailnet targeting and approval/rotation behavior. Remaining native proof
covers actual package/image compatibility, namespace/TUN/LocalAPI isolation, DNS
ownership/recovery, systemd stop/restart ordering and provider responses. These
are not proven by the local SDK or CLI fixtures. Optional reusable-key enrollment
and arbitrary additional project Tailnets are not hidden prerequisites. No retained
target, native test fixture, provider credential or maintenance window is
selected/authorized by this document.
