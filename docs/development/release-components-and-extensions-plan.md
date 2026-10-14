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
