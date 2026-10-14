# Release architecture

Soda ships as a **minimum-deviation Fedora CoreOS** appliance and has a direct
[Docker runtime](#docker-runtime) deployment target. An exact-source candidate
carries the complete Soda payload; GitHub Releases owns appliance delivery.

This document owns the durable release model. How to run tools day-to-day lives in
[development release workflow](../development/release.md) and
[installation](../guides/installation.md).

## Candidate model

1. **Build** produces architecture-specific candidate artifacts.
2. **Image/media** assembles host content and installation media that consume that
   candidate.
3. **Stage** uploads the complete asset set as an immutable GitHub prerelease.
4. **Qualify and promote** tests those public bytes on native x86_64, then makes
   the unchanged release the default download.

Building or staging alone does not qualify a release. A failed build or failed
installation cannot become successful by changing its label. Experimental
host-image tooling does not replace native FCOS installation and updating.

## Public distribution

Users download one ISO. It automatically retrieves its hash-bound root filesystem
from the same versioned GitHub release. The root filesystem contains the Soda
runtime and application images; users do not download or assemble components.
Installation requires internet access to GitHub. There is no Soda download server,
appliance registry channel, publication ledger or custom update-discovery service.
Docker delivery follows its own registry and qualification contract below.

The fixed image catalog includes `canlang-runtime`, built from the same frozen
Rocky base as Project OS and Dashboard using the admitted offline Canlang tools.
Its configuration, manifest and archive digest enter the existing payload and
shared OCI layout; installation imports it into local Podman storage with the
other release images. Candidate content checks bind its tool inventory and
license files, and the host's exact Canlang tool archive and release reference.
Applications supply an already compiled portable bundle in a separate image;
appliance activation neither compiles an application nor downloads runtime
dependencies. New installs and replacements must name the selected release's
runtime image and tool digest. Existing application generations retain their
stored references for inspection, start and stop across an appliance update.

Every asset must be smaller than GitHub's 2 GiB limit. Packaging removes unused
host copies and compiler debug/symbol data, then admission checks the actual bytes;
estimated compression savings never authorize an upload. The release includes the
exact Soda and Fountain source archives, package inventory, provenance and checksums.

The release payload binds `ProjectSupport` to the SHA-256 of `project-terminal`,
its fixed destination, its maintenance copy inside the host root filesystem and
the exact `project-os` image configuration. The producer compiles this helper once
and stages those bytes into all three uses. Candidate assembly reads back the
helper from both built images; delivery checks the corresponding OCI content.
The maintenance input ships inside the complete release and requires no separate
user download. These artifact checks establish byte identity, not a qualified
host upgrade or retained-Project transition.

Enable GitHub immutable releases before staging. Prepare a draft, upload all assets,
compare GitHub's inventory with local sizes and hashes, and publish it as a
prerelease. A network installer can then be tested against its real anonymous
download URLs. Promotion changes only notes and prerelease/latest flags; assets and
tag remain locked. The final native smoke receipt belongs in the editable release
notes because immutable releases cannot acquire new attachments after publication.
See GitHub's [immutable release contract](https://docs.github.com/en/code-security/concepts/supply-chain-security/immutable-releases).

The first-install qualification is deliberately bounded: focused checks for changed
release code, verification of the downloaded installer, and one fresh native
installation and boot with exact installed revision and bundled-image readback.
Repeat a check only after failure or a relevant change. This does not claim
hardware coverage, product-wide acceptance, or upgrade/recovery qualification.

## Trust-key admission

Local packaging trust keys use one `PUBLIC KEY` PEM block with only ASCII whitespace
around it, within the existing 1 MiB trust-document bound. Image and delivery
share the typed admission adapter in `lib/release-inputs`: it requires strict
DER SPKI, the P-256 algorithm and curve identifiers, and an uncompressed point
on the curve. Fingerprints use the original decoded DER bytes. The artifact role
authenticates local packaging inputs. GitHub's immutable-release attestation binds
the public release assets and tag; local packaging keys do not replace it.

Installer ECDSA signatures use strict DER on P-224, P-256, P-384 and P-521.
Verification hashes the certificate's original signed TBS bytes; certificate
authority, key-usage and signature-algorithm policy remain with the installer.

## Host update ownership

The following CoreOS update contract applies to appliance installations. The
[Docker runtime](#docker-runtime) has a separate deployment lifecycle.

The appliance follows Fedora CoreOS update mechanics (rpm-ostree / Zincati-aligned
operation). Soda does not invent a parallel host updater for ordinary maintenance.
First-install evidence does not qualify an upgrade or recovery path. Those paths
need their own evidence before being offered as supported release operations.

An optional `UpgradeFrom` declares one prospective source release before the
candidate is built. It is not qualification. A supported transition additionally
requires `upgrade-qualification.json`, stamped after the native transition,
interrupted recovery and stopped-Project observations exist. This external receipt
binds source release and support digest to the exact target payload, host manifest,
Project image, support digest, public rootfs and deployable OSTree commit, with
hashes of the native observation files. It changes no baked host bytes.

The existing public rootfs contains the complete host, application images and an
OSTree repository inside its EROFS root. Explicit maintenance consumes the selected
GitHub release's rootfs and metadata together; users do not assemble helpers.
Native `ostree pull-local` and locked deployment retain FCOS activation ownership.
Import includes the digest-named container blob references required by rpm-ostree
to resolve the deployment's image metadata. Mutable image pointers from the media
repository are not imported over the appliance's existing pointers.
Before finalization, the workflow rechecks the exact staged commit, serial and
lock, then binds rpm-ostree's native base identity: the container manifest digest
for an imported container commit, otherwise the base OSTree commit. This rule also
applies when that container commit came from the public rootfs repository.
Media records `DeploymentCommit` from that rootfs's actual deployment separately
from `AssemblerImportCommit`; an assembler import identity cannot select a host
deployment. Development qualification admits the exact candidate without a receipt
only through its explicit fixture path. Normal upgrade admission and release
delivery require the bound native receipt.

The explicit `release` maintenance operation closes admission before applying
support or releasing the native finalization lock. It fixes all enrolled container identities, includes
stopped roots, refuses active or uncertain work, and retains a durable operation
record across interruption and reboot. Boot reconciliation checks actual host,
Project and future-image identities before reopening admission; that additional
readback is not the pre-activation safeguard. Explicit rollback restores both the
retained native deployment and its support, requiring empty managed state again.

These controls apply to this explicit workflow. They do not intercept arbitrary
root use of rpm-ostree, upstream Zincati deployments or manual rollback. Zincati
remains enabled and native FCOS owns deployment and activation. The integrated
release-pair journey must pass before a transition is offered as supported. See
[operator maintenance](../guides/operator-setup.md#explicit-release-maintenance) and the independent
[Project repair contract](../reference/project-os.md#terminal-support).

## Architectures

SodaOS supports **x86_64 only**. AArch64/ARM64 is unsupported and excluded from
current implementation, build, delivery and qualification scope. Supporting it
requires a future explicit product decision; upstream platform support or existing
experimental selectors do not establish Soda support.

Qualification requires native Linux x86_64 evidence. Cross-compilation, emulation
and development checks on another host architecture do not qualify the appliance.

## Docker runtime

Docker is an additional direct deployment target for the complete Soda runtime.
It uses the same candidate's Soda binaries and bundled application images on
Fedora container userspace, with systemd and Podman inside the outer Docker
container. It shares the Docker host kernel and requires native x86_64 Linux,
rootful Docker Engine and cgroup v2. The [Docker guide](../guides/docker.md)
owns deployment and build commands.

The Docker image excludes CoreOS boot, disk installation and host-update
services. The operator maintains the host OS; Docker owns container lifecycle.
Persistent configuration, application data and writable Project roots survive
container recreation. Replacing the Docker image is not a qualified appliance
rpm-ostree transition or an implicitly supported retained-state upgrade.

Registry delivery must bind an immutable Docker image digest to the verified
candidate revision, bundled payload, resolved Fedora base, RPM inventory and
corresponding source/notices. Docker qualification needs a fresh deployment,
native browser/Project/factory checks, orderly shutdown and retained-state
container recreation. Appliance ISO qualification alone does not qualify Docker.

## OCI metadata admission

Consumed OCI layout, index, manifest and image-config documents must be valid
UTF-8 JSON. Unknown extension fields may be ignored, but their bytes must still
be valid UTF-8. Content identifiers use the original bytes. Layer bodies retain
their format-specific binary admission rules.

## Media

ISO is the primary installation medium. Prepared cloud disk images may follow the
same candidate. Exact artifact names, trust bootstrap and publication endpoints
belong to the delivery implementation and operator guides, not duplicated version
tables here.

## Separation from product features

Release engineering owns shipping bytes and qualification. It does not redefine
project, Spaces, Tailnet or runner product contracts. Those remain in their
canonical documents.
