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

Plan version: **RCX-2 (adopted 2026-10-15; supersedes all RCX-1 interfaces below)**. Frozen contracts live in `.artifacts/release-design-refinement/` (`RCX-2.md` `7c5d8d3c`, `release-design.md` `547bfb7f`, `fountain-design.md` `68d110aa`, `tailnet-design.md` `facb7eeb`, `contract-examples.json` `83208f74`, `packets.json` `f322de0e`); those documents own their interfaces, this checklist owns execution order only. The implementation coordinator owns this checklist;
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

### I1: release component contract (RCX-2)

One canonical Payload v4 in `lib/release-inputs` (existing `soda-build-tools`
crate, no new crate): `Format=4`, six fixed required roles
(`dashboard`, `forgejo`, `proxy`, `postgres`, `project-os`, `canlang-runtime`)
as a fixed struct (no string maps), `ExtensionCatalog` version 1 with exactly
`soda` (required, no controller) then `tailnet` (optional,
`lifecycle_controller="soda"`), `OptionalExtensions.tailnet` binding the
official image, digest DAG packages-then-payload-then-host, no cycles. Native
packages are deterministic gzip-1 USTAR archives (≤512 MiB); fixed fast profile
`-zlzma,level=1 -Efragments -C1048576 --quiet`, no global dedupe; complete CPIO
≤2,013,265,920 bytes. Importer gains only
`soda-image-import ensure --payload ABS --set required|tailnet`; units run by
exact config ID with `Pull=never` after the importer. Maintenance uses the one
directory-inode flock with SH (mutations) / EX (updater), version-neutral intent
snapshot, offline `prepare`, reconcile-after-EX-drop. Full contract owns in
`release-design.md`; implement exactly it, no alternate codec, format, solver,
cache, or downloader.

### I2: Fountain package and lifecycle contract (RCX-2)

One owner for requested state: `.intents/<id>.json` (schema, UUID, decimal
generation, desired boolean, pending `activate`/`deactivate`/null) with
generation-CAS + fsync before effects; no `.disabled`/`.lifecycle` authorities.
One `PackageStore` (lock-bearing, shared CLI/manager), streamed verified
acquisition into the SHA cache, atomic receipt/intent, required bootstrap.
Real private-IPC controller dispatch with opaque lifecycle authority
(`observe|activate|deactivate`, exact tuple pin, bounded deadlines,
failed-enable cleanup exception only); nil teardown hooks removed. Boot is
required-bootstrap then async optional reconcile; maintenance uses the shared
inode guard with manager SH/EX discipline and cache/offline-prepare contract.
Native admin routes with origin/admin checks, exact error codes, 2s status
refresh. Full contract owns in `fountain-design.md` with wire fixtures in
`contract-examples.json`; implement exactly them, no second enabled-state owner,
no in-process teardown substitute, no jobs framework.

### I3: authority, host effects and maintenance (RCX-2)

Native site admin + origin protection for browser changes; short-lived manager
lifecycle authority for effect calls; no invented actor or UID-only plugin
authorization. Required `soda` controller fronts optional `tailnet` through the
exact private route even when Tailnet's backend is absent; cleanup stays callable
then. Operational grant is ephemeral (`/run/soda-tailnet/admission.json`),
closed on boot/failure, never default-open; no second durable enabled owner.
Disable drains admitted starts, stops host + all companions, verifies, preserves
credentials/cache/config/Projects; failed stop stays closed and unconfirmed.
Package IDs fixed: required `soda`, optional `tailnet` (draft `soda-tailnet`
renamed, no alias), image role `tailscale`. Full contract owns in
`tailnet-design.md`; implement exactly it, no custom distribution, no security
model substitution.

### Lanes and exclusive ownership

| Lane | Tasks and owned paths | Inputs, exclusions and join |
| --- | --- | --- |
| R — release | R00–R03; same paths (payload/producer/importer/maintenance owners in `release-design.md`) | Own Payload v4 cutover, package/image production, thin host/Docker, maintenance. Consume F01/F02/F04 and T02/T03 handoffs per packet deps. Do not edit Tailnet runtime/config readers owned by T. |
| F — Fountain | F01–F04; Fountain `modules/extensions/`, `services/extensions/`, `sdk/`, `sdk/lifecycle_client.go`, `routers/web/extensions/`, native admin routes/templates/navigation, `web_src/js/features/admin-extensions.ts`, related extension fixtures and `docs/fountain-release.md` | Use released ownership of Fountain checkout. Preserve the other chat's release work; change its smoke fixture/metadata only where the new extension contract requires it. No Soda edits. |
| T — optional Tailnet | T00–T03; `lib/host/src/tailnet/` + new `tailnet/lifecycle.rs`, `tcontrol*.rs` + `tcontrol_wire.rs`, Tailnet portions of `dbackend/`, `daemon/` and `main.rs`; `internal/tailnet/`, bridge callers, `cmd/soda-extension --package-id`, `frontend/tailnet/`, `system/containers/extension/` manifests/assets, Tailnet service/start hooks | Own required-controller/optional-package split and restricted effects. Consume F03/F04 SDK and R01 payload types. R owns release-file removals; transfer exact file reservations before deletion. |
| Coordinator | D00 (with Codex), I01, Q01, this checklist, shared root manifests/locks, final owning-doc updates and Git index operations | Reserve any newly discovered shared file before edits. Integrate native handoffs in place; no cherry-picking between shared-checkout workers. Codex alone owns D00 review and C01. |

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

Current execution is owned directly by Codex under the active RCX-2 completion
goal, following the owner's takeover instruction after Muse exhausted its quota.
Existing implementation checkpoints remain preserved. Direct repairs include
`07a95849` (producer archives, references and staged versions), `0fa7d524d4`
(bootstrap, observations and shutdown), `a26afecc15` (asynchronous optional
startup and same-generation recovery), and `d2a49627` (fixed private operator
manager bridge). Affected Fountain runtime/admin race suites pass. The native
root → UID2000 → UID1000 operator transport passes using a fixture callback;
this does not qualify Fountain's complete manager behavior. T02's operational
grant and exact-owned host runtime foundation passes its small native check:
official cached image, NeedsLogin, same-CID/PID adoption, stop-only cleanup and
preserved state hash. The retained earlier G-T container remains unchanged.
Lifecycle wiring, bounded companion inventory/drain, stop-only helpers and
image-contained CLI calls now pass affected tests. A native composition check
with fixture manager authority passes activation, real systemd supervision,
fresh-state admission, controller-instance rejection and stopped deactivation.
These are component checks, not complete product qualification. T03/R03,
producer integration and the combined native journeys remain open. Concurrent
Fountain CI/lint edits are preserved. The image smoke driver now uses the current
required-package catalog/cache contract; its focused repair is committed separately.

- [x] **D00 — Adoption/writer checkpoint.** Codex review + coordinator adoption.
  Bases `505cf545`/`c500bd92`, 10 Soda + 16 Fountain dirty files checkpointed
  (`adoption-rcx2/` tars/patches), no live commands. Prior work mapped, preserved.
- [ ] **R00 — Coherent retained-host derivative (G-R).** R; D00. Run the one
  fixed recipe in `release-design.md`: retained host OCI → ostree commit with
  regenerated closure + fresh osmet, exact assembler 1.9.4, fixed fast profile,
  complete CPIO ≤2,013,265,920 bytes, pack ≤300s. Old exact-compressor result is
  a size/codec estimate only (OSTree refs removed, osmet stale). No codec sweep,
  no production rebuild. Broad release rewrites wait for this gate.
  FAILED 2026-10-15 (`ostree container commit` refused: retained base already
  finalized, empty /sysroot, dangling /ostree; evidence build-failure.txt) —
  returned to Codex, NOT passed. Amendment 1 (`adoption-rcx2/
  g-r-recipe-amendment-1.md` + assembler cmd-import/cmdlib readbacks): OMIT
  only the repeated finalizer; derivative removes images + flips fsoptions +
  asserts empty sysroot; assembler import path owns the fresh OSTree closure;
  distinct amended outputs, no duplicate loads, all gates retained. Amended
  R00 DISPATCHED 2026-10-15 (worker 01a13d3e, T lane free + heavy slot free
  after T00g3 release; reuses existing task store after no-live-owner check;
  R01 continues disjointly).
  TIMING-016: 357.90s is aggregate live-stage (osmet×2+EROFS+initramfs/ISO
  in pack.log 861–1332), NOT demonstrated rootfs-only time — speed gate
  UNPROVEN (neither pass nor fail), open for Codex decision. CPIO size
  facts stand independently.
  TIMING-PROBE: isolated rootfs+CPIO 149.45s (mkfs 147.22 + CPIO 2.23,
  native epoch clock, timeout-300 never fired) ≤300s PASS; host-decode
  FSCK_EXIT=1 confirmed from retained log (observer/policy deviation,
  guest-reader PASS stands). R02 RELEASED on R00+R01+F02.
- [ ] **T00 — Fixed guest probe (G-T).** T; D00. v0.1.1-guest recipe in
  `tailnet-design.md` §10: actual Project userns, persistent host node, nested
  Docker, direct LocalAPI with exact caller. Retained: official digest
  `sha256:9969…`, exact-caller 200, digest-pinned execution. No workstation
  subuid/root-disk changes; no ISO build.
  STATE 2026-10-15: image-physics evidence complete (userns/socket/stop/
  persistence/nested, exact 200, full cleanup) BUT checkbox stays OPEN:
  (a) step-9 used an alias tag after the specified Host.Config FROM failed
  in the disposable legacy Docker builder ONLY (pull denial; exact failure +
  readbacks preserved) — intermediately misread as a production-caller
  failure, WITHDRAWN per REFERENCE-011: the production path builds via
  PODMAN (production_images.rs build_image line 71), and the retained-host
  R00 build already reached RUN STEP3 with Podman FROM 418ad724, proving
  config resolution in the production builder. R02 keeps normal
  identity/build acceptance, no tag fallback, no special repair;
  (b) fixture VM intentionally retained running for T02 (deviation from
  §10 clean-shutdown, reported); (c) one workstation-/tmp scratch use
  recorded (guest /tmp is /home-backed and allowed). Product qualification
  remains open; DO NOT repeat the fixture assembly merely to relabel it.
- [x] **F01 — Strict SDK freeze.** F; D00. Exact catalog/manifest/lifecycle
  types + registration + opaque authority from `fountain-design.md` §2/§5 with
  strict decode tests and one wire fixture (`contract-examples.json` tuples:
  controller/instance/UUID/generation/pin/deadline, distinct authority).
  Draft scaffold reusable; wire must match exactly.
  CLOSED 2026-10-15: frozen SDK + strict tests + one wire fixture PASS
  (`TestFrozenContractExamplesWire` vs @`83208f74`, clock 1792100000;
  independently rerun by coordinator under pinned Go 1.26.9); full
  sdk/modules/routers suites green; no tailnet.* capabilities, no second
  enabled owner, no teardown hooks. Limitation: G-F live-process proof stays
  F03; manager/admin runtime behavior unchanged (F03/F04 rebuild targets).
  Residual repaired 2026-10-15 (REVIEW-005): wire fixture now reads repo-local
  `sdk/testdata/rcx2-contract-examples.json` (byte-identical, sha
  `83208f74…`); no external-checkout path in tracked source; test PASS
  rerun pinned. F02/other writers preserved (coordinator-reserved paths).
- [x] **R01 — Canonical Payload v4.** R; F01. ACCEPTED 2026-10-15 with ONE
  residual REVIEW-010 CLOSED 2026-10-15 (concurrent drain + enforced 256KiB
  bound + reap on all paths; focused fake-podman regression PASS,
  independently rerun by coordinator). Otherwise:
  `lib/release-inputs` v4 types, all producers/readers/fixtures cut over
  together, `ensure --payload ABS --set required|tailnet` for the two trusted
  callers. Preserve ProjectSupport. No v3 compatibility branch.
- [x] **F02 — Locked store + verified acquisition.** F; F01. CLOSED 2026-10-15:
  one lock-bearing PackageStore (manager+CLI, bypass deleted), streamed
  SHA-verified acquisition, required bootstrap, sole durable .intents CAS,
  exact 5-route shell + explicit offline prepare; suites green pinned
  (modules independently rerun by coordinator). F03 owns dispatch/ordering.
  RESIDUAL OPEN 2026-10-15 (REVIEW-026): production gaps in store/intent
  owners under repair by fresh worker (checkpoint preserved). One `PackageStore`,
  streamed acquisition with redirect/tar limits, verified SHA cache + receipt,
  required bootstrap; CLI and manager share the implementation. Replace
  `.disabled`+`.lifecycle` with sole durable `.intents` generation-CAS + fsync.
- [ ] **F03 — Real controller + G-F proof.** F; F02. REOPENED 2026-10-15 (REVIEW-026):
  integrated in existing owners (parallel package removed, checkpoint
  retained); 009 strict-intent/rollback/same-generation + 012 fail-closed +
  017 decoded-key + 018 null-boolean repairs green pinned; G-F proven with 9
  live-process tests. Controller dispatch,
  distinct admission tokens, callback validation, single mutation slot, exact
  recovery ordering IN EXISTING manager/lifecycle/callback owners (no parallel
  f03dispatch package; checkpointed work integrates where it belongs). Prove
  G-F with live fixture processes over real private sockets (no nil hooks/DTOs);
  CLOSED 2026-10-15: integrated in existing owners (parallel package removed),
  009/012/017 implemented, G-F proven with 9 live-process tests green pinned.
  retain the observeLocked deadlock fix, release map mutex around external work;
  boot async. Section-7 rendering/settings framework DEFERRED to F04 (F03 only
  the authenticated manager callbacks it needs).
- [ ] **F04 — Admin UI + maintenance contract.** F; F03. REOPENED 2026-10-15 (REVIEW-026, maintenance-guard item):
  native admin routes (401/403/Origin/CAS, 4KiB forms, 303 CAS wiring),
  manager-control snapshot/reconcile/cache endpoints, actual middleware
  tests + tsc/eslint clean, all suites green pinned (routers independently
  rerun by coordinator). All four Fountain packets complete. Native admin routes
  with registered Origin/CAS/catalog guarding, rendered states/progress,
  snapshot/reconcile + same-inode SH/EX guard. Test actual middleware.
- [x] **T01 — Required controller + exact authority.** T; R01, F03, T00. CLOSED 2026-10-15:
  controller→Go-route→Rust-RPC with exact SDK tuple/callback, ForwardUnix
  seam consumed, bad authority zero effects (Lifecycle tests PASS rerun by
  coordinator). T02 needs T01 ONLY (frozen packet dependency; final
  cross-process/native product acceptance waits for repaired F path +
  combined I01 integration, which is not a reason to idle independent
  implementation). T01 checkpoint: tmp/t01-release.patch + t01-release-new
  tarball. Note: old system/containers/tailnet/Containerfile removal landed
  via T lane without recorded R reservation transfer; end state intended,
  no revert.
  Soda controller → authenticated private service → bounded Rust host RPC.
  Remove static default-open admission; ephemeral grant closed on boot/failure;
  start-drain/stop-only teardown preserving credentials. No second intent owner.
- [ ] **T02 — Boot-closed runtime integration.** T; T01. Serialized
  start/drain/stop, persistent host official-image runtime + Project companions,
  stop failures closed and unconfirmed. Preserve data. Runtime implementation
  and small native composition gate pass; final integration/reboot and failure
  journeys remain at I01/Q01, so this packet is not yet closed.
- [ ] **T03 — Tailnet package split.** T; T02, F04. Package renamed to
  `tailnet` (no alias), exact two capabilities, admin-only page, live native
  admin mapping, enabled-but-unenrolled UI, connection-loss confirmation. Package
  registration, manifests, separate asset graphs and archive staging now pass
  focused SDK/browser/producer checks. Installed enable/disable journeys remain
  unqualified until integration.
- [ ] **R02 — Thin host/Docker + dev target.** R; R00, R01, F02. Separate
  package/image production, ordered boot, obsolete bundle/installer paths
  removed; final native packages use T03 contents at I01. Fountain release
  admission, SDK source pin, real pipeline hooks and actual base attribution
  are now wired. Native probes verified official manifest/config identity and
  corrected Skopeo transport plus gzip export of a new layer. The small native
  loopback-registry roundtrip preserves the exported manifest/config/platform.
  Exact committed Fountain `56b40e8f` and its real Soda setup/presentation overlay
  pass native startup, required SDK handshake and offline persistent restart;
  the overlay preserves Fountain's executable/wrapper/entrypoint/s6 run bytes.
  Native package fixture origins now flow through the existing CLI, candidate
  wrapper and isolated worker; production package URLs remain fixed to GitHub.
  The complete-rootfs ceiling is enforced before hashing and at admission.
  Independent Fountain fixture metadata/source admission, selected integrated
  package bootstrap and combined candidate qualification remain open.
- [ ] **R03 — Maintenance integration.** R; R02, F04, T02. Snapshot with
  source/target bindings, enabled-only prefetch, cached restart, rollback
  retention, EX/prefetch/source+target/offline-prepare/drop-EX discipline.
  Implementation and 20 focused maintenance tests pass; native UID1000 offline
  store checks, confined input-label/lock probes and actual exit75 command check
  pass. The shipped Fountain CLI also prepares an exact cached required package
  offline in the Soda overlay, preserves UUID/intent/receipt, and rejects a missing
  cache before changing those identities. Complete native Source-S/Target-T
  transition and recovery remain open.
- [ ] **I01 — Integrate + joint native candidate.** Coordinator; R03, T03, G-F.
  Pin SDK/Fountain candidate, smallest combined tests + private-socket/lock
  checks; admit the distinct v4 maintenance source fixture; one joint target.
- [ ] **Q01 — Native ISO + Docker journeys.** Coordinator; I01, v4 source
  fixture. Fresh install/enable/disable/failure/restart/update + rollback with
  per-stage timings; G-P anonymous availability reported separately. Local
  fixture passes are development evidence only, never public qualification.
- [ ] **C01 — Independent acceptance and close.** Codex; Q01. Critical-boundary
  acceptance, findings resolved, owning docs verified, exact qualified revisions
  reported. No push, publishing or merge. Absorb completed design into owning
  docs and remove the plan only when its remaining work is resolved.

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
