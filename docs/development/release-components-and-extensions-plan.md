# Release components and optional extensions implementation plan

Make Soda releases faster while retaining free GitHub hosting, and make Tailnet
an optional extension installed only when an administrator enables it. Fountain
owns the general extension-management experience; Soda declares its extension
contents and implements the required system operations.

This is the agreed implementation scope, not a claim that the changes have
landed. The three workstreams are **1: separate release components**,
**2a: optional Tailnet**, and **2b: Fountain extension management**. Fountain's
independent release track is a supporting dependency. Writing this plan does not
start a production build, publish a release, or modify an installed appliance.

This plan owns their coordinated implementation and acceptance sequence. Absorb
the completed contracts into [release architecture](../architecture/release.md),
[networking](../architecture/networking.md),
[Forgejo integration](../reference/forgejo.md) and the corresponding operator
guides, then remove this plan under the
[documentation authority rules](../README.md#authority-rules).

The [execution checklist](#execution-checklist) below prepares this work for Muse.
Preparation does not launch implementation. It is the single progress checklist;
runtime identities, prompts and detailed receipts belong in ignored `.artifacts/`.

## Current behavior and the problem

Soda builds its application images separately, exports their contents into a
shared OCI layout, and embeds that layout in the host root filesystem. The
release then compresses the combined OS and applications into one downloadable
rootfs. Docker copies the embedded application collection into its outer image.
The small installer ISO downloads the rootfs; it is the rootfs packaging that
causes the expensive compression work.

The current production profile uses LZMA with global deduplication to fit the
combined rootfs below GitHub's per-file limit. The new objective is the shortest
practical build, packaging and upload time, with correct installation and free
hosting. Smallest possible output is not an independent requirement.

Tailnet enrollment is optional today, but installation is not: appliance and
Docker packages include native Tailscale, enable its host service, and include
the Project Tailnet image. The Tailnet page is part of the required Soda
extension. Its enrollment and connection controls are not extension installation
or complete feature disablement. Private LAN setup already works without Tailnet
enrollment and supplies the baseline access path.

Fountain already has an extension package model, native admin authority and
extension page support. Its current CLI package activation is not the requested
browser-managed download, enable and disable experience. Extend that existing
model rather than implementing a separate extension manager in Soda.

## 1 Separate the release components

Keep the host OS and its native maintenance tools together. Remove application
image bodies from both the host rootfs and the outer Docker runtime image.
Publish complete components with meaningful identities; do not create numbered
rootfs chunks or divide arbitrary OS directories to meet hosting limits.

| Component | Owner and delivery | Default installation |
| --- | --- | --- |
| x86_64 installer ISO | Soda, GitHub Releases | Appliance entry point |
| Host rootfs with CoreOS and Soda host services | Soda, GitHub Releases | Appliance OS |
| Docker runtime image with Soda host services | Soda, public GHCR | Docker entry point |
| Compose file | Soda, GitHub Releases | Docker installation configuration |
| Dashboard, Project OS and Canlang runtime images | Soda, public GHCR | Required components, acquired automatically |
| Soda extension package and presentation resources | Soda, separate package on GitHub Releases using Fountain's supported format | Required Soda integration |
| Fountain | Its own repository and release track | Consume a tested released image by digest |
| Caddy and PostgreSQL | Official upstream images | Consume fixed upstream identities |
| Tailscale | Official upstream dependency, subject to the native compatibility check below | Acquired only when Tailnet is enabled |
| Release manifest, checksums, provenance, inventories and source | Release owner, GitHub Releases | Machine-readable release and source records |

Soda may need a small customization layer over Fountain for its existing branding
and startup integration. Fountain compilation belongs to Fountain's release
track. Use Fountain's native extension package format for extensions; do not
require a complete Fountain runtime inside every extension package solely for
validation or installation. Settle that format with workstream 2b before changing
the current extension-image producer. This package can share Soda's release
version and publication workflow; it does not need another independent pipeline.

One release binds the exact required component set and the compatible optional
extension versions. Use immutable digests and verified package checksums for
consumption. A release does not follow floating `latest` dependencies. Finalize
OCI compression before recording manifest digests, because changing compressed
blobs can change those digests.

For the ISO, installation or first initialization acquires and verifies every
required component before reporting readiness. Docker initializes from the same
component identities and retains its inner Podman storage in the existing
persistent state volume. Required acquisition uses the ordinary network connection
and must not depend on the optional Tailnet service or an application that has not
yet been downloaded. Existing cached content supports subsequent offline starts.

Appliance maintenance verifies and acquires the target's required images and the
dependencies of currently enabled extensions before activation. Disabled extensions
remain undownloaded. Retain images needed by rollback and existing Projects.
Preserve the native FCOS deployment and Project-support checks; adapt their content
inputs together with the new packaging. Optional extensions retain their explicit
enabled/disabled state across host updates.

Use standard registry and container tools for image transfer and layer reuse.
Do not add a custom download server, chunk protocol or second image cache. Keep
compiler caches and validated artifacts, reuse exact-source CI evidence where
admitted, and publish the same bytes that were validated.

### Size and hosting proof

[GitHub Releases](https://docs.github.com/en/repositories/releasing-projects-on-github/about-releases#storage-and-bandwidth-quotas)
requires each asset to be under 2 GiB; its documented limit is per file, not the
combined release. Public container packages and current GHCR storage/bandwidth
are covered by [GitHub's free hosting policy](https://docs.github.com/en/billing/concepts/product-billing/github-packages).

The host-only rootfs has not yet been measured. Before a broad packaging rewrite,
use retained build content to produce a representative host-only input and test
fast compression with the actual native tool version. Account for retained
OSTree content as well as the visible filesystem; hiding one image directory is
not proof that its bodies have left the packaged repository.

Measure the complete downloadable rootfs, including its installation metadata.
Record elapsed time, final bytes and remaining margin. An estimate, incomplete
file or timed-out pack is not a size proof. Choose the profile using measured
total release cost, and keep a per-release size gate. If the complete host asset
does not fit with reasonable headroom, resolve that result before implementation;
do not silently restore slow compression or introduce numbered parts.

## 2a Make Tailnet an optional extension

The same user journey applies to appliance and Docker installations:

1. Install Soda without the Tailscale runtime package, container image or running
   Tailnet service. The available-extension declaration may ship with Soda.
2. The administrator reaches Forge over the normal private LAN connection.
3. In **Settings → Extensions**, Tailnet is available and disabled.
4. The administrator enables it. Fountain coordinates activation; Soda downloads
   the declared, pinned dependency through its restricted host integration.
5. After successful acquisition and startup, Tailnet offers sign-in and its host
   and Project enrollment settings. Enabling the extension does not silently
   enroll devices or select a network.
6. The administrator can disable it. The system blocks new Tailnet connections,
   stops existing Project companions and the host Tailnet service, and retains
   the disabled state across restarts and upgrades.

Disable preserves Projects, files and saved configuration. It does not mean
uninstalling cached software or deleting credentials. Connection loss from
disabling the network must be clear to an administrator using it; private LAN
administration remains the baseline. A failed stop must not be reported as
successful disablement. Download/start failures must remain visible and must not
leave the extension falsely reported as enabled.

The Tailnet feature should be separately manageable without disabling the required
Soda extension or Forge itself. Its status and activation controls must remain
reachable while ordinary Tailnet management is disabled.

The preferred dependency is Tailscale's official image, with Soda retaining its
existing networking and enrollment policy. First prove it can support both the
host node and Project companions with the required socket, namespaces and
permissions. Soda already overrides the Project image entry point to run
`tailscaled` directly. Any remaining need for custom packaging must be justified
by a demonstrated compatibility failure, not by the existing recipe alone.

## 2b Fountain owns extension management

| Responsibility | Owner |
| --- | --- |
| Settings → Extensions, available/installed entries and administrator actions | Fountain |
| Native admin authentication and authorization | Fountain |
| Generic package identity, validation and lifecycle contract | Fountain |
| Enable/disable coordination, persistent requested state, progress and errors | Fountain |
| Extension contents, required versions and declared capabilities | Extension publisher; Soda for its extensions |
| Tailnet-specific dependency acquisition, startup and teardown | Soda extension integration |
| Privileged host and Project operations | Existing restricted Soda host service |

Fountain renders declared extensions and invokes their supported lifecycle
operations. Soda supplies the Tailnet declaration, dependencies and behavior;
it does not build another extension-management UI or general lifecycle manager.
The generic Fountain implementation must contain no Tailnet-specific behavior.

Extend Fountain's existing manifest/package model with only what the current
consumer needs. Publish a concrete contract for activation, deactivation,
operation results and recovery after interruption. Fountain records requested
state and presents actual lifecycle results; Soda reports the observed runtime
state. Avoid independent managers that can disagree about whether an extension
is enabled. Re-observe an interrupted operation before retrying its effects.

Use live native site-admin authority for the requested admin controls. Reconcile
the current configured-operator-only Tailnet checks deliberately; a displayed
button or claimed actor header is not authorization. The existing trust boundary
still applies: Fountain and arbitrary extension processes receive neither root
execution nor an unrestricted Podman or host-helper socket. Soda's authenticated
service bridge and restricted host operations enforce each declared capability.

Browser activation and disablement must have real runtime effects, not merely
change the next-start CLI setting. Define any necessary Fountain restart behavior
in its lifecycle contract before Soda integrates it. Installation and activation
must not require an administrator to finish the operation using hidden CLI steps.

Fountain's independent release track and extension management are separate
deliverables. Release its image, corresponding source and provenance from its own
repository. Soda pins a released Fountain version that provides the required
extension contract, adds its own integration, and tests the combination.

## Implementation order and ownership

| Step | Owner | Outcome needed before dependent work |
| --- | --- | --- |
| Prove host-only packaging | Soda release | Complete rootfs size and fast-profile timing, using retained content |
| Prove official Tailscale compatibility | Soda networking | Small native host/Project runtime checks without a full ISO build |
| Produce Fountain release contract | Fountain | Versioned consumable image, source and provenance; explicit downstream interface |
| Implement generic extension management | Fountain | Tested admin UI and lifecycle contract, including disable and failure outcomes |
| Separate required release components | Soda release | Thin host/Docker artifacts, component acquisition and coherent identity/maintenance checks |
| Implement optional Tailnet | Soda extension and host | Declaration and lifecycle connected to Fountain; default bundle/service dependencies removed |
| Qualify the composed result | Both repositories at fixed revisions | Native appliance and Docker journeys using the selected immutable component set |

The initial packaging and Tailscale experiments can proceed alongside Fountain
work. Final consumption depends on Fountain's tested release and extension
contract. Prove these interfaces before substantial orchestration or production
builds. Reuse existing test drivers and component artifacts. Full native release
qualification follows successful component checks; source tests do not substitute
for installed evidence.

## Acceptance checks

- A complete host-only rootfs fits below 2 GiB with recorded headroom and a
  measured fast compression profile. Every public asset passes its hosting limit.
- A fresh ISO installation and a fresh Docker deployment acquire the same
  required component identities anonymously from their public locations, without
  publisher credentials, and reach normal Forge/Project readiness over LAN.
- Default installation contains no Tailscale runtime payload or running Tailnet
  service and makes no Tailnet dependency download. Core operation requires no
  Tailnet enrollment.
- An administrator enables Tailnet entirely through Fountain settings; the
  declared dependency is downloaded and verified before usable status is shown.
- Non-admin requests are refused by native authority and the system boundary.
- Host and Project Tailnet behavior works with the admitted upstream dependency.
  Disable stops those connections, prevents new ones and preserves Project data.
- Disabled state survives restart and release maintenance. Re-enabling reuses
  valid cached content; download failures and interrupted operations show truthful
  state and do not require replaying unrelated setup.
- Enabled Tailnet survives a qualified update with its compatible dependencies
  verified before activation; the same update does not download a disabled
  extension's dependencies.
- Required cached components support subsequent starts without another download.
  Maintenance stages required content before activation and preserves rollback
  inputs and existing Project-support safeguards.
- Release measurements distinguish shared compilation, packaging, compression
  and upload. Claims of improvement use actual results rather than projected
  compression savings.

## Source starting points

For component production and delivery, start with `lib/soda-release-build`,
`lib/soda-release-image`, `lib/soda-release-deliver`, `cmd/soda-install`,
`cmd/soda-image-import`, `cmd/soda-project-maintain` and `system/docker`.

For Tailnet, start with `system/host/provisioning`,
`system/containers/tailnet`, `system/containers/extension/extension.json`,
`frontend/tailnet`, `internal/web/api/tailnet.go`, `lib/host/src/tailnet` and
`lib/host/src/daemon`.

Fountain owns its existing extension modules, native admin routes, SDK and package
lifecycle in its own repository. Coordinate changes there with its current owner;
the separate release work is not evidence that new extension management is ready.

## Execution contract

Plan version: **RCX-1**. The implementation coordinator owns this checklist;
Codex owns independent review, monitoring and closure. The maximum parallelism is
**three implementation lanes**, using **one Muse coordinator with native
subagents**. Queue within the actual native limit. Use the canonical Soda and
Fountain checkouts, disjoint file reservations and serialized Git index writes;
do not create worktrees or separate worker CLIs.

The owner selected **`muse-spark-1.3-contributor` with Contributor mode on**.
Use **XHIGH**, the highest documented effort for that variant; Meta's
[reasoning contract](https://dev.meta.ai/docs/reasoning) excludes MAX for Spark
1.3 Contributor. This selection matches the current skill. Verify the
installed profile and actual live settings/producer model before dispatch; a
profile command accepting a name is not proof of served capability. Do not
substitute another model. The prepared local roster records the exact CLI check.

### Requirements and evidence

| ID | Required outcome | Observable acceptance |
| --- | --- | --- |
| RQ01 | Complete components, fast free hosting | Finished host-only rootfs below 2 GiB with measured margin; named assets, no chunks; measured packaging time |
| RQ02 | Coherent independently delivered components | Pinned image/package identities; core absent from outer OS/Docker layers; verified acquisition before LAN readiness |
| RQ03 | Tailnet absent by default | Fresh appliance and Docker contain no Tailscale runtime, daemon or dependency pull |
| RQ04 | Admin browser enable and disable | Native admin activates, verified download/start, sign-in available; disable stops host and Project connections without deleting data |
| RQ05 | Fountain owns generic management | Native Extensions settings and lifecycle; Soda provides declarations and restricted effects; no Tailnet-specific Fountain logic |
| RQ06 | Accurate authority and failures | Non-admin/forged identity rejected; interrupted operations re-observed; failed start/stop remains visible and unconfirmed |
| RQ07 | Cached restart and maintenance | Offline cached core starts; enabled target dependencies prefetched; disabled ones not fetched; rollback and Project inputs retained |
| RQ08 | Lean, reviewable local delivery | Focused commits, owning docs updated, retained evidence reused, critical native boundaries proven before full builds |

Current source evidence: Soda payload format 3 requires seven images, a common
repository prefix, archive checksums and Soda revision checks for most roles.
Fountain's local independent release implementation now defines `release.json`,
an amd64 image, executable and corresponding source archives in its
`docs/fountain-release.md`. Its extension installation/enablement still requires
a stopped host. These are source findings, not public-release or native runtime
qualification receipts. Refresh both Git bases and active writer ownership at
launch; a clean checkout does not establish that another chat has released it.

### I1: release component contract

Replace the obsolete payload format and all strict readers together. No legacy
reader or compatibility branch is required for this unreleased format. Preserve
the existing release and `ProjectSupport` identity/verification owners.

| Record | Required semantic fields and rules |
| --- | --- |
| Release | Schema version, Soda version/source, architecture, required image roles, required packages and optional extension catalog |
| OCI component | Stable role, complete registry reference pinned by manifest digest, expected image configuration digest, platform and provenance owner/source |
| Extension package | ID, version, HTTPS asset URL, bytes, SHA-256, compatible Fountain release/SDK contract, required flag and manifest-declared capabilities |
| Optional binding | Extension ID plus its compatible package and exact dependency identities; presence in the catalog does not select it for acquisition |
| Provenance | Soda-built components bind Soda source; Fountain binds its own release/source; upstream Caddy/Postgres/Tailscale bind admitted upstream identities |

Registry references need not share Soda's repository prefix. Archive checksums
remain local build evidence where relevant, not invented runtime inputs for an
image obtained from a registry. Finalize registry blob compression before sealing
manifest digests. Reuse ordinary Podman transfer/cache and the existing registry
publisher's digest checks; consume independently published upstream images in
place. A small Soda Fountain presentation layer, if needed, is a Soda-built
component with the upstream base digest recorded.

The release lane owns the canonical payload serialization and matching producer,
installer, importer and maintenance readers. Publish its exact types and a
minimal fixture in R01 before Tailnet changes its payload readers. Required core
acquisition precedes readiness; failed acquisition is retryable without repeating
completed setup. Existing exact cached content must permit an offline restart.

### I2: Fountain package and lifecycle contract

Use one gzip-compressed archive of Fountain's native package directory (manifest,
executable and assets), with a stable root layout and SHA-256/size in the trusted
catalog. Extraction must remain inside the staging directory; validate the
existing manifest/executable rules before activation. The administrator chooses
among the release's declared packages; this scope does not add arbitrary URL
installation, a marketplace or a generic package solver.

Fountain owns the catalog reader, acquisition, installation identity and durable
requested enablement. Reuse its installation UUID, separate private data, manager
admission controls and package locking. Required Soda integration and optional
Tailnet are distinct package identities. Optional package failure must not prevent
Forge or required Soda integration from starting.

Extend the SDK with three logical lifecycle operations: **observe**, **activate**
and **deactivate**. Each request identifies the installed package/version and
operation; results identify the operation and report observed state, progress and
a bounded error. Fountain alone persists requested state. Distinguish disabled,
activating, enabled, deactivating and failed/unconfirmed outcomes. Enrollment is
separate: enabled and awaiting sign-in is a successful installation outcome.

Fountain can run a package's control/lifecycle handler while ordinary contribution
admission is closed. Activation opens ordinary admission only after successful
effects and observation. Disable first closes new admission, then invokes external
teardown and verifies it before stopping the backend and reporting completion.
Keep observation/cleanup reachable after a failed or interrupted operation. On
restart observe existing effects before retrying them; do not add an independent
Soda enablement authority or an elaborate general-purpose workflow engine.

Use live package activation through Fountain's manager; no whole-Fountain restart
or administrator CLI step is part of the enable/disable journey. F01 must prove
the package lock, required-package isolation and external teardown boundary with
the existing SDK fixture before T02 integrates. F01 publishes actual SDK types,
catalog shape, native admin route and a wire fixture; downstreams import those
definitions instead of creating parallel DTOs. These implementation identifiers
may follow existing conventions; the semantics above are fixed.

### I3: authority, host effects and maintenance

Native Fountain site-admin authentication and CSRF protections authorize lifecycle
actions initiated by a user. Background observation/reconciliation uses the
authenticated manager service identity and previously admitted installation
state, including after restart without an active administrator session. It must
not invent an admin actor or authorize new user requests through that service path.
Soda revalidates authenticated SDK actor authority and exact extension
identity/capability through its existing bridge. Do not trust browser-supplied
actor headers. Preserve existing Project-specific authorization; this work does
not broaden unrelated configured-operator operations.

Soda's Rust host service admits only declared Tailnet effects and reports observed
dependency, host-daemon and Project-companion state. Fountain/extension processes
receive no root shell, general container socket or arbitrary host command API.
The official-image probe fixes the digest, executable paths, sockets, namespaces,
devices and persistent state locations consumed by this restricted implementation.

Closing Tailnet admission must not disable stop/observation. The current
`run_tailnet_action` returns early for both run and stop when management is false;
remove that coupling as part of T02. Disable preserves credentials, configuration,
Project files and verified cached images. Clearly display network loss before an
admin disables the connection they are using; LAN remains available.

Fountain provides an authenticated snapshot of enabled extension IDs/versions to
maintenance. Before FCOS activation, acquire the target's required components and
compatible dependencies for those extensions. Serialize an enablement change
against this maintenance selection/activation interval using the existing
maintenance exclusion boundary; fail visibly if a stable selection cannot be
established. Never use a second desired-state file as authority. Retain inputs
needed by the previous deployment and existing Projects. R and T acknowledge this
handoff together in I01.

### Lanes and exclusive ownership

| Lane | Tasks and owned paths | Inputs, exclusions and join |
| --- | --- | --- |
| R — release | R00–R02; `lib/soda-release-build/`, `lib/soda-release-image/`, `lib/soda-release-deliver/`, `cmd/soda-install/`, `cmd/soda-image-import/`, release-facing `cmd/soda-project-maintain/`, `system/docker/`, `system/host/Containerfile`, `system/host/provisioning/`, `system/host/image/`, old `system/containers/tailnet/` | Own payload cutover, removal of bundled image bodies/RPMs, packaging and dependency acquisition. Consume F01 package contract and T00 official-image receipt. Do not edit Tailnet runtime/config readers owned by T. |
| F — Fountain | F00–F02; Fountain `modules/extensions/`, `services/extensions/`, `sdk/`, native admin routes/templates/navigation, related extension fixtures and `docs/fountain-release.md` | Use released ownership of Fountain checkout. Preserve the other chat's release work; change its smoke fixture/metadata only where the new extension contract requires it. No Soda edits. |
| T — optional Tailnet | T00–T02; `lib/host/src/tailnet/`, `tcontrol*.rs`, Tailnet portions of `dbackend/`, `daemon/` and `main.rs`; `internal/tailnet/`, Tailnet host/API/auth bridge callers, `cmd/soda-extension/`, `frontend/tailnet/`, extension manifests/assets and Tailnet service/start hooks | Own split of required Soda versus optional Tailnet package contents. Consume F01 SDK and R01 payload types. R owns removal of old image producer/Containerfile; transfer that exact file reservation before deletion. |
| Coordinator | P00, I01–I03, this checklist, shared root manifests/locks, final owning-doc updates and Git index operations | Reserve any newly discovered shared file before edits. Integrate native handoffs in place; no cherry-picking between shared-checkout workers. Codex alone owns V01/C01. |

R owns maintenance payload/default readers; T owns `lib/host/src/daemon/config.rs`
and the extension manifest. Changes to shared root manifests/locks are proposed
by lanes and applied by the coordinator after relevant writers release. Different
files do not remove semantic dependencies. Contract changes require affected
writer/command reconciliation, release and acknowledged reassignment.

## Execution checklist

Every task reports exact input/output revisions, checks actually run, retained
evidence path and residual limitations. A released handoff includes changed files,
active command handles and explicit writer release. The coordinator accepts that
exact handoff before the worker completes its native goal; parent integration and
Codex review remain separate. No task is complete merely because it was assigned.

- [ ] **P00 — Establish bases and reservations.** Coordinator; no prerequisites.
  Refresh both checkouts, existing goals/writers and preserved release requests;
  record actual bases and dirty inputs. Confirm Fountain ownership release before
  assigning F. Resolve tool/state/temp locations under `/home`, protected VM/state
  and exact disposable test targets. Freeze RCX-1 inputs and per-file reservations.
  Acceptance: acknowledged roster/inputs; no competing writers (RQ08).
- [ ] **R00 — Prove host-only packing.** R; P00. Inventory retained audit/native
  inputs, use the actual assembler/EROFS version and a development-only host tree.
  Remove application bodies from visible files and retained OSTree content; check
  metadata/boot compatibility. Complete the cheapest promising fast-profile pack,
  including installation metadata, and record elapsed time, bytes, headroom and
  integrity. Compare another profile only if the first result leaves a material
  decision unresolved. Gate the broad packaging rewrite on this evidence; an
  incomplete output or estimate fails acceptance (RQ01/RQ08).
- [ ] **T00 — Prove the official Tailscale image.** T; P00; parallel with R00/F00.
  Pin native amd64 upstream content; reuse cached inputs. Check actual host and
  Project entrypoints, LocalAPI, namespace/device/socket behavior, teardown and
  persistent host state, including nested Docker constraints. Use disposable
  fixtures; no ISO build. Record credential-free proof separately from later
  enrollment/reachability proof. Release exact invocation contract (RQ03/RQ04).
- [ ] **F00 — Bind the existing Fountain release interface.** F; P00 and Fountain
  writer release. Inspect its current release docs/metadata and SDK. Identify exact
  candidate source and reusable build inputs, package directory rules and image
  entrypoints. Record whether a public release exists; local source is insufficient.
  Acceptance: pinned source/SDK and downstream input contract, no rebuild of an
  unchanged Fountain artifact (RQ02/RQ08).
- [ ] **F01 — Implement and prove generic lifecycle.** F; F00. Implement I2 with
  native admin settings, trusted package acquisition, live manager operations,
  locking, persistent intent and failure/observation handling. Extend existing
  package/SDK/router fixtures for admin/CSRF, denied actors, download failure,
  live disable/re-enable, interrupted effects and required-package isolation.
  Publish the exact SDK/catalog/maintenance snapshot contracts and real fixture
  result before T02/R02 use them (RQ05/RQ06/RQ07).
- [ ] **R01 — Cut over component identities.** R; R00, F00. Implement I1 across
  producer/content verifier and all strict consumers. Keep Project support and
  provenance checks; remove seven-image/common-prefix assumptions. Acknowledge
  F01's final package fields before sealing shared types: **F01 is a completion
  prerequisite**. Verify digest-preserving
  registry round-trip and rejection of wrong digest/platform/source fixtures.
  Release canonical types for T02 (RQ01/RQ02/RQ07).
- [ ] **R02 — Make host and Docker thin.** R; R01, F01, T00 to start; **T02's
  package/runtime handoff is required to finish**. Prepare/build complete
  components, consume Fountain base, package Soda extensions, remove app bodies
  and default Tailscale packaging/services, wire core acquisition and update
  prefetch. Use R00's measured fast profile and per-asset size guard. Verify local
  component startup, exact cached restart and selected dependency acquisition;
  preserve rollback/Project guards. No full ISO until I01 (RQ01/RQ02/RQ03/RQ07).
- [ ] **T01 — Separate declaration and policy.** T; T00. Split optional Tailnet
  identity/page from required Soda; define restricted effects and replace
  configured-operator Tailnet authorization with verified native admin authority.
  Separate admission from cleanup; preserve Project policy. Author against I2/I3
  but wait for F01/R01 before binding concrete wire types (RQ03/RQ04/RQ06).
- [ ] **T02 — Connect lifecycle to actual runtimes.** T; T01, F01, R01. Implement
  observe/start/stop through admitted host operations; enable downloads exact
  upstream content and disable stops host/all Project companions. Handle failed
  and interrupted effects, retain data/cache, and expose truthful status through
  Fountain. Verify native effects and bridge authorization using existing focused
  drivers. Release package contents and runtime contracts to R02 (RQ04–RQ07).
- [ ] **F02 — Qualify the new Fountain contract.** F; F01. Extend its existing
  disposable image smoke fixture to exercise optional download/live lifecycle
  with the required extension still running. Bind source, SDK and tested image
  metadata. Local candidate may unblock local integration; public consumption
  requires the matching published immutable release. No publication is implied
  by this task (RQ02/RQ05/RQ08).
- [ ] **I01 — Join released lanes and run component acceptance.** Coordinator;
  R02/T02/F02 accepted with writers/commands released. Verify browser-to-host
  lifecycle, stable maintenance selection, failure results and artifact identities
  together. Run affected source checks using pinned tools; update owning docs.
  Commit coherent scopes serially and record exact candidate revisions. Only
  passing interfaces and component checks release the full-build gate (RQ01–RQ08).
- [ ] **I02 — Validate native appliance and Docker integration.** Coordinator;
  I01. Build a fixed native candidate, record rootfs/ISO/image sizes and timings, and use
  existing fresh-install/Docker/maintenance drivers. Prove LAN readiness, no default
  Tailnet payload or pulls, admin enable/sign-in, host and Project connectivity,
  complete disable, preserved data, offline cached restart, and enabled/disabled
  update plus rollback. Use exact isolated targets recorded by P00; preserve
  `sodaos-local`. A driver-only repair does not automatically require rebuilding
  unchanged shipping bytes. Unpublished fixture inputs make these development
  integration receipts, not public-release qualification. A later change to URL
  bindings or dependency bytes requires a new candidate; do not promise reuse
  of incompatible fixture bytes. Record each scenario's evidence (RQ01–RQ08).
- [ ] **I03 — Verify public-hosting qualification when available.** Coordinator;
  I02 and actual public immutable components. Check anonymous pulls/assets for
  the exact candidate, final per-file limits, source and manifest consistency.
  Perform the fresh native ISO install/boot and fresh Docker deployment with those
  actual public identities; bind readiness and lifecycle receipts to their exact
  bytes. Reuse I02 evidence only where its inputs are identical. Publication is a
  distinct authorized action; do not push,
  publish or change package visibility to manufacture this prerequisite. Local
  registry evidence does not complete this task. Report a precise external gate
  if public inputs are unavailable (RQ01/RQ02/RQ08).
- [ ] **V01 — Independent acceptance and repairs.** Codex; affected integrated
  scope and command/writer release. Independently inspect artifact contents and
  digests, actual default absence, native admin/forged-actor denial, failed-stop
  truthfulness and update selection. Reuse relevant native receipts and perform
  bounded independent probes where critical invariants lack evidence. Route
  defects through the coordinator with exact acceptance failures, verify repair
  receipt/progress and renewed handoff. Record I03's external gate separately;
  do not label overall qualification complete while it remains open.
- [ ] **C01 — Close exact owned resources.** Codex; accepted results/repairs and
  reported remaining gates. Preserve commits/evidence and any reserved candidate;
  reconcile commands/viewers, close only released
  run resources and retire the matching heartbeat when no local follow-up remains.
  If I03 is externally gated, record implementation delivered/qualification pending
  rather than completed qualification. Absorb completed design into owning docs
  and remove the plan only when its remaining work is resolved.

### Run records, supervision and cost controls

The prepared run directory is `.artifacts/muse-release-components/`. Codex owns
`monitor.json`, containing absolute checkout/checklist/prompt paths, planned role
roster, cleanup ownership and timestamped observations. No Muse identity, goal,
timer or pane is allocated by preparation. Each future worker owns its validated
startup/status record; startup facts are immutable and live observations retain
source timestamps. Use the invoked skill's atomic record helper. The coordinator
alone edits this checklist; runtime records are not competing progress checklists.

The saved coordinator prompt and lane assignments carry native goal handling,
ownership and exact handoff acceptance. Single-instance communication uses native
subagent tools; there is no separate peer handshake or multi-TUI human-ready gate.
At an authorized launch, verify the actual TUI identity/workspace/model/effort,
startup receipt, goal and native roster. Record one concrete tmux viewer command.
Single-instance mode uses native worker progress/completion events and normal
command handling. No recurring Muse worker-check loop is permitted, including
native timers or external timer fallbacks; its worker timer is not applicable.
Codex creates and verifies one **25-minute same-chat heartbeat**, quiet on healthy
unchanged progress and active on stalls, failures or completion. It is inactive
until verified startup. Codex coordinator selection: **GPT-6.1 Sol, High**;
record actual runtime evidence and report unavailable configuration capabilities.

Each assignment records a concrete ACK/progress deadline, handled through native
events and normal command handling without another recurring worker loop.
Long commands record owner, cwd, handle, start, expected progress and deadline;
reinspect the same command every **1–5 minutes, normally two**. Missed progress
requires a concrete decision, not duplicate builds or a reset deadline. Reuse
retained inputs and existing drivers, serialize costly native runs, and record
why each expensive check resolves a remaining uncertainty. Do not add a new
qualification framework or speculative recovery subsystem.

Originating Codex may deliver concrete scoped status/continuation/recovery
instructions when a worker or coordinator stalls. Reconcile exact identities,
active commands and writer ownership first; route native-worker recovery through
the coordinator's actual native tools. Require exact recipient ACK and actual
progress/results; injection or a send summary is not delivery. Preserve raw
rejections and return recovery evidence to the coordinator. An actual permission
or capability failure is reported explicitly. Coordinator replacement requires
command/writer reconciliation, checklist handoff and each
continuing worker's acknowledged authority transition before new dispatch.

All writes, caches, temp files and sockets stay under `/home`. Respect repository
language ownership and pinned tools. Do not mutate protected installations,
reclaim another task's artifacts, broadly prune caches, push, merge or publish.
Only after acceptance and exact resource release may Codex close owned
panes and disposable fixtures; preserve attached viewers and unique work.
