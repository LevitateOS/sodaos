# Upstream ownership audit

Reviewed on 10 September 2026 against installer candidate `8320f9c`.
**Source ownership review complete; findings are not implemented fixes.** This audit covers every current `internal/` package
and follows the actual application, native, frontend and build callers. It asks
whether Soda needs each responsibility and whether a mature upstream mechanism
can own more of it. It is separate from installer acceptance, a penetration test,
or permission for a broad rewrite/deployment.

The [architecture](architecture.md) and [upstream-first instructions](../AGENTS.md#human-maintainable-engineering)
remain the product boundary. A small adapter is justified when it binds native
operations to Soda's operator, project, actor or artifact contract. Reimplementing
the underlying forge, package manager, authentication protocol, container engine
or service supervisor is not.

The result supports keeping the architecture. It does **not** establish that Soda
has recreated those upstream systems wholesale. The clearest findings are unused
credential retention, inconsistent cancellation checks, file-update correctness,
missing schema checks and a provider service-entrypoint mismatch. Smaller reuse
candidates exist, but replacing working boundaries requires behavioral equivalence.

## Coverage and selected baselines

The source inventory contains **16 internal packages**. Package/file count is
coverage bookkeeping, not a design score. Reviews trace effectful callers and
existing tests rather than treating directory boundaries as separate authorities.
Production source versions are selected by the existing manifests and recipes:

- [Go modules](../go.mod): Go 1.26.7, coder/websocket 1.8.15,
  x/crypto 0.55.0, x/sys 0.47.0, modernc SQLite 1.58.0,
  DiceBear Go 10.7.0 and schema 1.5.1.
- [Frontend workspace](../package.json): Bun 1.4.2, TypeScript 7.0.2,
  Lit 3.3.3 and xterm 6.0.0; [Cockpit](../cockpit/package.json) retains
  React 18.3.1 and PatternFly 6.6.1. [Lit analysis](../tools/lit-check/package.json)
  separately pins lit-analyzer 2.0.3 and its classic TypeScript 5.9.3 runtime.
- [Forgejo service](../appliance/services/forgejo.container): stock Forgejo
  15.0.7. [Caddy service](../appliance/services/soda-proxy.container): 2.10.2.
  [Project OS](../project-os/Containerfile): Rocky 10.2 userspace; native RPMs
  resolve during an authorized build and actual versions belong in build metadata.
- [CoreOS ISO](../appliance/locks/coreos-iso.json): 44.20260817.3.2.
  [Installer builder](../scripts/build-installer.py) selects CoreOS Installer
  0.26.0; selected FCOS metadata identifies systemd 259.8 and
  OpenSSH 10.2p1. Those base versions are not proof of downstream configuration,
  SELinux behavior or any retained appliance's installed package state.
- [GitHub runner](../appliance/locks/github-runner-source.toml): 2.337.0.
  Forgejo Runner/Tailscale/Podman are native package inputs, not newly pinned by
  this report. Relevant upstream APIs and observed package metadata are discussed
  with their callers below.

| Internal package | Actual callers and upstream owner | Soda's remaining responsibility / verdict |
| --- | --- | --- |
| `acceptance` | [Support command](../tools/soda-acceptance/main.go), SSH/QEMU/QMP, existing build/check scripts | Exact requested phase/target, owned process cleanup, private evidence and transport. Keep bounded support; no independent product test scenarios or release scheduler. |
| `avatar` | [Web avatar adapter](../internal/web/avatars.go), dashboard startup and [preview](../tools/soda-avatars/main.go); DiceBear renderer | Original robot artwork, stable public seed and version/size contract. Keep; it calls the upstream renderer. |
| `config` | [Dashboard](../cmd/soda-dashboard/main.go), [setup](../cmd/soda-setup/main.go), runners; Go URL/JSON/filesystem APIs | Soda origins, socket/credential paths and operator binding. Keep validation; audit unused bootstrap-token retention below. |
| `filelock` | [Runner state admission](../internal/runners/native.go); kernel flock | Keep cancellable waiting around advisory locking. The kernel owns the lock; a Go-context wait is Soda's small addition. |
| `forgejo` | Web/setup/Tailnet commands; native Forgejo HTTP/configuration | Keep bounded acting-user APIs, actual consent inspection and sanitized transport. Forgejo owns passwords, repository permissions and Git; Soda owns its OAuth return/context binding. |
| `host` | [Host daemon](../cmd/soda-host/main.go), project image and web helper client; Podman, OpenSSH, Linux accounts, tmux/systemd | Keep fixed privileged operations and exact actor/project/account binding. Correct key-file concurrency, queued cancellation and output bounds below. |
| `installer` | [Media command](../appliance/installer/main.go), [builder](../scripts/build-installer.py); CoreOS Installer, NetworkManager, OpenSSL, Ignition, systemd/OpenSSH, Caddy | Text choices, irreversible-effect boundary, verified Soda payload and explicit first-use integration. Keep the adapter; native validation remains deferred. Custom TCP connection supervision was removed in this candidate. |
| `linuxhost` | Runner CLI/helper; NSS and pkexec | Keep the independently enforced root-only operator policy. Moving this small single-feature adapter into runners is optional cleanup, not an upstream replacement. |
| `nativebuild` | [Artifact command](../tools/soda-artifacts/main.go), build/stage/installer; Go archive/ELF/hash/filesystem, gpgv/xz, Podman | Bind source/platform and fixed public payload without runtime secrets. Keep these policies; compare the narrow OCI parser with upstream image readers before extending it. |
| `process` | [Tailnet status reader](../internal/tailnet/tailnet.go); Go os/exec | Only production consumer uses Output. Reduce unused generic command/trace surface during Tailnet work; retain a small test seam and add capture bounds. |
| `projectos` | Store/web/helper image inspection; Podman/OCI image identity | Keep immutable creation-profile binding. OCI owns image identity; Soda owns the selected profile/interface and its association with the created project. |
| `runners` | Native CLI/helper, Cockpit and global operator settings; provider runners and systemd | Keep local account/capacity/service integration, fixed operations and private registration input. Correct GitHub's service entrypoint; provider scheduling, workflow execution and cache remain upstream-owned. |
| `store` | Web/setup; SQLite and Go AEAD | Keep Soda-only associations, original account memberships, cancellation transactions and encrypted grants. Complete v8 startup validation; consider native random-nonce packing with persisted-format proof. |
| `strictjson` | Web/helper/runner requests and profiles; Go encoding/json | Bounded body, valid UTF-8, one object, top-level duplicate-name rejection and destination-struct unknown-field rejection. Keep on the selected Go baseline; not a general recursive schema engine. |
| `tailnet` | [Tailnet CLI](../cmd/soda-tailnet/command.go), [Forgejo advertisement](../cmd/soda-forgejo-tailnet/main.go), Cockpit; native tailscale CLI/LocalAPI | Keep native identity/address projection and separate Git advertisement. Review unstable LocalAPI dependence and native device UI overlap; preserve operator access and current preferences. |
| `web` | Dashboard command, native Forgejo hooks and Spaces frontend; Go HTTP/templates, Forgejo OAuth/API, coder/websocket | Keep Soda actor/context/access admission and exact terminal leases/receipts. Complete post-provider session checks; retain native Forgejo and Lit/xterm owners. |

## Findings

This order prioritizes authority and data correctness over optional code reduction.
It is a maintenance recommendation, not a new feature roadmap or deployment grant.

| Order | Finding | Disposition |
| --- | --- | --- |
| First | Unused bootstrap token retained and service-readable | Stop future copy/access; preserve existing operator files and compatibility. |
| First | Join/Start/Stop/key Apply can use a session captured before delayed provider I/O | Add original-session checks and precise admission tests. |
| First | Project-key replacement can overwrite a later native edit | Define and implement an honest conflict/update contract with race fixtures. |
| Next | Current schema validation omits v8 columns/trigger | Extend native SQLite startup checks and malformed-schema fixtures. |
| Next | GitHub service uses interactive launcher | Use the upstream service entrypoint after validating the selected runner layout and shutdown behavior. |
| Next | Queued host cancellation and unbounded command/stream capture | Fix at the existing admission/capture boundaries; no scheduler or process framework. |
| Next | Aggregate preparation and browser fixture invocation gaps | Wire existing test owners and fresh outputs once. |
| Conditional | Tailnet LocalAPI/native UI, OCI reader, nonce packing, uncalled terminal mode | Compare actual contracts before replacing/removing; preserve state and user journeys. |

### Remove unused bootstrap credential retention

[Setup](../cmd/soda-setup/main.go) copies the operator's bootstrap token into an
`admin-token` file; [configuration](../internal/config/config.go) requires its path.
[Activation](../appliance/bin/soda-activate) grants the dashboard service read
access to that file. A repository-wide production-consumer search found no runtime
credential reader: provider operations use acting-user encrypted grants. Activation
adjusts the unused file's permissions. This is **confirmed unnecessary retained
authority**, not evidence that the dashboard is making privileged provider calls.

Stop creating the copy for new setup and stop granting future service readability.
Keep old configuration parse compatibility without requiring the retired path.
Preserve existing operator-owned files; revocation and cleanup require separate
scope. The [setup guide](operator-setup.md) also requests admin/repository token
scopes although the client calls only `/user` and `/user/applications/oauth2`.
The selected [Forgejo API routes](https://codeberg.org/forgejo/forgejo/src/tag/v15.0.7/routers/api/v1/api.go)
place those operations under user scope. Narrow token guidance with focused
verification of the actual bootstrap operations; the selected operator's eligibility
check is separate from a need to call an administrator API.

### Recheck the original session before native mutation

[Join](../internal/web/environments_api.go), [Lifecycle and key Apply](../internal/web/management.go)
capture a session, wait for provider identity/repository authorization and then
dispatch the native action without checking whether that same login context still
exists. Logout can complete during the provider wait. An ordinary HTTP context is
not cancelled by deleting its session. This is a **source-confirmed admission gap**;
the interleaving was not executed in this audit.

Use the original-session comparison already present in Create and
[runner authorization](../internal/web/runners.go), with delayed-provider/logout
fixtures at the actual mutation handlers. Preserve stable actor ID, login context
and CSRF comparison. A recheck closes the stale-provider-result window; it is not
atomic cancellation of a native operation or rollback after dispatch. Define the
exact admission boundary before promising stronger logout semantics. Do not hold
terminal registry locks across provider/native I/O or add a provider-role cache.

### Complete current database validation

[Migration v8](../internal/store/migrations.go) introduces `creation_profile`,
`repository_settings_return` and the `immutable_creation_profile` trigger. The
startup completeness queries omit all three. An incomplete database that reports
the current version can therefore pass the intended startup gate. No damage to a
retained database was observed or alleged.

Extend the existing zero-row queries and inspect SQLite's catalog for the expected
trigger, with malformed-current-version fixtures. [SQLite's integrity and foreign-key
checks](https://www.sqlite.org/pragma.html) validate storage and declared constraints;
they cannot infer Soda's intended schema. Keep transactional append-only migrations,
pre-migration grant-key checks and native SQL. No ORM, schema-repair engine or
version bump is needed merely to validate what the current version requires.

### Correct project-key concurrency before claiming compare-and-swap

[Project key maintenance](../internal/host/project_keys.py) reads and hashes the
current managed key file, writes a temporary replacement, rechecks the current
pathname and then calls `os.replace`. A native administrator can change the file
between that last check and replacement; Soda can overwrite the later change.
Directory `flock` coordinates cooperating Soda operations, not arbitrary native
editors. The [kernel contract](https://man7.org/linux/man-pages/man2/flock.2.html)
is advisory locking. This is a **source-confirmed correctness defect**, not a
runtime reproduction in this audit.

Keep native OpenSSH key files. Define the precise update/conflict contract before
changing the write: existing-inode append is suitable for enrollment but does not
by itself implement revocation or replacement of a managed key set. Preserve later
native edits and report uncertainty instead of restoring a snapshot. Add deterministic
replacement/in-place-edit and partial-write cases. Do not substitute a live
`AuthorizedKeysCommand` database or claim general atomic filesystem compare-and-swap.
The installer enrollment append correction does not fix this separate project path.

### Make helper admission cancellable and bound command capture

[Buffered host dispatch](../internal/host/daemon.go) holds one mutex across native
operations, including creation's readiness wait. A cancelled waiter cannot leave
that mutex wait. Preserve serialization, but use a local context-selectable admission
primitive and recheck cancellation before effects. Verify with a delayed executor;
do not introduce per-project scheduling or a general queue.

The same executor uses unbounded output buffers before callers enforce response
limits. Add limits while capturing native output, preserving redacted diagnostics
and injected tests. [Go os/exec](https://pkg.go.dev/os/exec) remains the process
owner. This correction is more useful than replacing small wrappers with a general
process framework.

### Preserve native project and terminal ownership

The [existing-project service](../appliance/services/soda-project@.service) starts
and stops the already-created persistent Podman container. Podman's
[existing-container service example](https://docs.podman.io/en/latest/markdown/podman-generate-systemd.1.html)
supports that shape. Although [Quadlet](https://docs.podman.io/en/latest/markdown/podman-systemd.unit.5.html)
is already used for appliance services and supports templates, changing retained
projects requires proof that their original container IDs, writable roots and
start/stop behavior survive. A declarative rewrite is not automatically safer.

[Project account provisioning](../project-os/rootfs/usr/libexec/soda/project-account)
uses native account tools and adds only Soda's identity association, shared path
and key installation. Persistent people/accounts are not a `DynamicUser` use case.
[Terminal integration](../internal/host/project_terminal.py) uses tmux for sessions
and systemd for lifetime; Soda binds exact accounts, PTYs and cleanup receipts.
The [nested Podman socket](../project-os/rootfs/etc/systemd/system/soda-podman.socket)
already delegates activation to systemd and the API/runtime to Podman. Preserve
these boundaries; an unrestricted host engine socket would remove necessary policy.

[OS observation](../internal/host/project_os.py) reads one bounded regular file from
the existing project. Python's [standard convenience reader](https://docs.python.org/3.10/library/platform.html#platform.freedesktop_os_release)
adds fallback behavior and does not itself establish Soda's file-size/type policy.
Keep the bounded observation unless a concrete native-parser replacement preserves
those constraints. It must neither start a stopped project nor infer installed
packages from a distro label.

### Delegate runner execution through the provider's service contract

[GitHub launch selection](../internal/runners/launch.go) invokes `run.sh` from the
[Soda systemd service](../appliance/services/soda-runner@.service). GitHub's
[custom-service documentation](https://docs.github.com/en/actions/how-tos/manage-runners/self-hosted-runners/configure-the-application)
requires `runsvc.sh`. This is a confirmed **documented contract mismatch**, not an
executed job failure. Use the provider's service entrypoint and verify its selected
2.337.0 packaged location, registration-created files, signal/stop handling and
update-disabled behavior before paired delivery. Fresh exact-tag source retrieval
failed during this slice; the source pin plus official service contract supports
the finding, not a claim of inspecting that script or proving runtime behavior.

The remainder of [runners](../internal/runners/) delegates local accounts to native
Linux, locking to flock, services to systemd, and jobs/registration to the provider.
The root CLI gate and web configured-operator gate protect distinct transports;
neither replaces provider authority. The no-echo native PTY passes registration
input to the upstream command without putting it in argv. Keep bounded private input
and sanitization; replacing it needs proof of an equally restricted native channel.
Local removal is distinct from provider deregistration; uncertain registration and
local cleanup outcomes must remain explicit.

Current [Forgejo configuration](../internal/runners/native_create.go) selects host
labels, disables container-engine access and disables cache. The selected Rocky
ephemeral job image and native cache integration are **unfinished feature work**,
not existing isolation or a reason to build a Soda scheduler/cache server.
[Forgejo's guide](https://forgejo.org/docs/latest/admin/actions/) assigns jobs to its
runner, logs/artifacts to Forgejo and cache storage to the runner. Validate the actual
installed runner version and its configuration before implementing those choices;
the layered package recipe does not pin the same version as the Forgejo server.

Cockpit and the [native operator page](../frontend/runners/soda-runners-page.ts)
reuse the runner backend and [shared response decoder](../frontend/runners/soda-runner-response.ts).
Retain Cockpit's backing logic/tests until the selected replacement meets its whole
journey. Two presentation entrypoints during that migration are not evidence of two
CI schedulers. Keep provider registration, permissions, results and workflow UI upstream.

### Review Tailnet's unstable API seam and native UI overlap

[Cockpit's Tailnet adapter](../cockpit/src/tailscale/native.ts) uses native CLI
status/up/set and accesses LocalAPI preferences and interactive login through the
privileged Unix socket. Upstream's [LocalAPI source](https://github.com/tailscale/tailscale/blob/main/ipn/localapi/localapi.go)
explicitly treats the v0 routes as internal and not necessarily stable. Exact
installed Tailscale version is unresolved by the layered package recipe. Record and
validate that baseline; do not describe the endpoints as a stable public API.

Tailscale's [device web interface](https://tailscale.com/docs/features/client/device-web-interface)
already offers exit-node selection/advertisement and device settings. This is a
concrete reuse candidate before expanding Soda's Tailnet forms. It has its own
reachability, device identity and management-access requirements, so it is not a
drop-in replacement for root-operator Cockpit or a first-enrollment solution without
further validation. Preserve current controls/tests until an equivalent journey
works. Never reset unrelated preferences merely to simplify reauthentication.

The [authentication stream](../cockpit/src/tailscale/stream.ts) frames successive
JSON objects and delegates parsing to `JSON.parse`; it is not another JSON engine.
Its accumulated partial object currently has no size bound. Add a bound at that
seam and to native runner/process output capture, with malformed/oversized fixtures.
No new stream framework is necessary.

The [Tailnet page observer](../cockpit/src/tailscale/store.ts) can call
[Forgejo address refresh](../cmd/soda-forgejo-tailnet/main.go), which edits the native
Git SSH advertisement and may restart Forgejo. Preserve and expose that effect when
changing the UI; the complete caller chain is not read-only status. It keeps browser
and OAuth origins separate and refuses an unpublished Git listener. Tailnet presence
does not prove approved subnet routing or client reachability.

### Simplify local boundaries only where it removes real machinery

[Key maintenance](../internal/host/management.go) preloads the whole terminal module
to reuse `account_for`. A small shared identity source could reduce that coupling,
but it must work in both immutable installed and embedded paths. Keep the current
reuse if extraction adds more deployment/import machinery than it removes.

`internal/process` has one production consumer and unused Run/trace machinery.
A Tailnet-local output seam can remove that surface. `internal/linuxhost` is already
a small legitimate root/NSS adapter; co-locating it with its runner owner is optional.
Retain tested cancellation/root checks. These are low-priority local simplifications,
not evidence of copied upstream subsystems or prerequisites for finishing features.

### Keep artifact policy; review general OCI parsing before expanding it

[InspectOCI](../internal/nativebuild/oci.go) implements a bounded single-platform
archive reader: it rejects unsafe/duplicate paths, hashes every blob entry and
validates referenced descriptor sizes/digests, resolves the local manifest/config
and binds Soda's architecture/revision
labels. It does not unpack or execute layers. The [build](../scripts/build-native.sh)
and [installation](../scripts/install-native.sh) already delegate image construction,
export and import to Podman using actual image identities.

OCI is upstream's format, and [Skopeo exposes manifest/config inspection](https://github.com/podman-container-tools/skopeo/blob/main/docs/skopeo-inspect.1.md).
Compare a pinned upstream reader or verified-copy operation before adding format
support. A successful metadata inspection alone is not an equivalent replacement
for this whole-archive verification. Keep Soda's private-payload exclusion, identity
binding and no-import preflight; use malformed/tampered fixtures to establish
equivalence. This is a **reuse candidate**, not evidence that deleting the current
validator is safe.

The [OCI v1.1.1 layout specification](https://github.com/opencontainers/image-spec/blob/v1.1.1/image-layout.md)
permits content beyond Soda's sealed, local, single-image subset. Document that
subset rather than presenting its refusals as universal OCI requirements.

### Keep the existing test runners; remove repeated preparation

The [root test scripts](../package.json) invoke `build:forgejo` separately through
multiple branches of the aggregate test command. The [native checker](../scripts/check-native.sh)
uses those existing Go/Bun/Python owners, and the
[remote support executor](../internal/acceptance/remote_executor.py) invokes the
same build/check scripts. The repeated preparation is a confirmed consolidation
opportunity already covered by the [maintenance plan](refactoring-plan.md).
Prepare browser outputs once for an aggregate run while retaining independently
usable feature commands. Do not introduce a second test scheduler or duplicate
product scenarios in support tools.

The [page fixture orchestrator](../scripts/test-spaces-page.ts) produces only Spaces
HTML; runner/repository-settings browser consumers need their separate producers
and output paths. The [settings-link browser test](../tests/forgejo/settings-link.test.ts)
is gated and absent from the explicit Lit browser list. A Go producer writes a fresh
artifact directory without disabling result caching. Wire existing owners, require
fresh fixture outputs and preserve independent entrypoints. These are source
invocation findings, not newly executed failures or measured timing improvements.

### Preserve native web authority and narrow frontend cleanup

Selected [Forgejo OAuth source](https://codeberg.org/forgejo/forgejo/src/tag/v15.0.7/routers/web/auth/oauth.go)
omits scopes from the token response, can reuse an existing grant and exposes actual
consent through introspection. Soda's bounded grant inspection, refresh serialization,
transaction-bound callback/logout cancellation and encrypted actor/session binding
therefore remain necessary. A general OAuth client cannot replace those policies.
Similarly, Go's cross-origin middleware does not supply Soda's required exact
origin, CSRF token, actor context or WebSocket first-frame authorization. Keep those
checks rather than treating them as duplicate authentication.

SQLite owns persistence; [Go's existing random-nonce GCM API](https://github.com/golang/go/blob/go1.26.7/src/crypto/cipher/gcm.go)
could replace the manual nonce-prefix mechanics in [grant encryption](../internal/store/grants.go).
This is a small **reuse candidate**, not a cryptographic defect. Require old/new
ciphertext compatibility, wrong-key and actor/session-binding fixtures; retain the
current key and associated-data contract without rewriting stored grants blindly.

[Lit controllers](https://lit.dev/docs/composition/controllers/) already own reactive
lifecycle composition. [Task](https://lit.dev/docs/data/task/) is an optional read-only
request candidate; cancellation/latest-result handling does not undo mutations or
resolve uncertain outcomes. Xterm/FitAddon own terminal rendering/fitting. Its
[AttachAddon](https://github.com/xtermjs/xterm.js/blob/6.0.0/addons/addon-attach/src/AttachAddon.ts)
is raw transport and does not implement Soda's authenticated control frames, exact
IDs or cleanup receipts. Preserve the existing bounded transport and flat terminal
host identity; no new terminal manager or frontend state framework is warranted.

One concrete removal candidate is [terminal standalone mode](../frontend/spaces/sodaspaces-terminal.ts).
Every production caller supplies a workspace locator; the no-locator caller is an
older fixture. Make the production contract explicit and port its useful adversarial
tests before removing the alternate storage/action/rendering branches. Preserve
[workspace legacy locator import](../frontend/spaces/sodaspaces-workspace.ts) and
unknown/pending evidence. Removing the uncalled mode must not erase legacy sessions.

All **253 Forgejo template overrides** were inventoried and compared for identical
copies against the retained native export; none was byte-identical. Representative
login, settings, notification and shared navigation callers preserve native handlers
and permission authority. This is ownership coverage, not independent proof of every
field/gate in all 253 templates. The [official Forgejo customization mechanism](https://forgejo.org/docs/v15.0/admin/advanced/customization/)
explicitly lacks backward-compatibility support. Keep version-specific template and
route checks; documented customization is not a stable API, and it does not require
or authorize a downstream executable fork.

### Keep bounded JSON policy on the selected toolchain

[strictjson](../internal/strictjson/decode.go) uses Go's parser, adding request
constraints that legacy `encoding/json` does not supply by default. Its duplicate
check applies to top-level field names; unknown-field refusal applies while decoding
the destination struct shape, including nested structs.
It should not claim recursive duplicate rejection or general schema validation.

Current [Go JSON documentation](https://pkg.go.dev/encoding/json) describes v2 in
Go 1.27. Inspection of the selected Go 1.26.7 source in the offline Linux checker
found v2 still behind `GOEXPERIMENT=jsonv2`, outside that version's compatibility
promise. Do not incidentally change the toolchain or enable an experiment for this
audit. Reconsider parsing/duplicate handling during an intentional baseline migration,
preserving body limits, object-only input, explicit unknown-member refusal and current
protocol behavior. Native v2 does not itself remove those application policies.

### Keep upstream-rendered artwork and native support protocols

[The avatar adapter](../internal/avatar/avatar.go) calls the installed DiceBear
10.7.0 `NewStyle`/`NewAvatar`/`SVG` implementation. Soda supplies original artwork
and deterministic seed/version policy. It is not a second avatar renderer.

The [QMP client](../internal/acceptance/qmp.go) implements capability negotiation,
request ID correlation and event skipping against the
[upstream protocol](https://www.qemu.org/docs/master/interop/qmp-spec.html).
QEMU owns virtualization. The support process wrapper owns only its launched child
groups and evidence; it does not replace the appliance's systemd supervisor.
Replacing these bounded clients with a broad orchestration framework has no
demonstrated benefit in this review.

The [Lit analysis adapter](../tools/lit-check/README.md) likewise uses the actual
upstream analyzer. Its separate classic compiler addresses documented package
resolution failures under the product's TypeScript 7 workspace. Keep it until
supported upstream versions remove that concrete mismatch; do not patch dependency
source or recreate the analyzer.

## Evidence and limits

The working reports and file inventory are retained under ignored
`.artifacts/upstream-audit-20260910/`; they are supporting evidence, not a build
dependency. Selected source, actual callers, manifests and primary upstream
contracts establish these recommendations. Current upstream pages identify
capabilities; they do not silently select a new dependency or establish installed
compatibility. Failed page retrievals were not treated as missing features.

No product refactor, native/provider action or deployment is performed by this
audit. Installer source checks and their actual results remain in the separate
[handoff](implementation-status.md). The x86 media build and full install remain
deferred. Findings outside the installer need their own bounded implementation,
regression checks and any required paired native delivery before being called fixed.

The interrupted runner review did not leave a completed artifact; the root reviewer
finished its source/caller review directly. No audit subagents remained active at
that continuation. Native/build/web reports and an independent build-support review
were retained; the additional planned independent web recheck was interrupted and
is not claimed complete. Root directly corroborated the credential consumers,
mutation paths and schema checks. Coverage includes all internal packages and
caller families, not exhaustive security proof or line-by-line parity of every
template, native dependency and UI branch.
