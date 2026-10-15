# Release architecture

Soda ships as a **minimum-deviation Fedora CoreOS** appliance and has a direct
[Docker runtime](#docker-runtime) deployment target. An exact-source candidate
carries the complete Soda payload; GitHub Releases owns appliance delivery.

This document owns the durable release model. How to run tools day-to-day lives in
[development release workflow](../development/release.md) and
[installation](../guides/installation.md).

## Candidate model

1. **Build** produces architecture-specific candidate artifacts.
2. **Package** produces the selected appliance media, Docker runtime, or both
   from that candidate.
3. **Stage** publishes the complete selected asset set as an immutable GitHub
   prerelease and, for Docker, a versioned GHCR image.
4. **Qualify and promote** tests those public bytes on native x86_64, then makes
   the unchanged release the default download.

Building or staging alone does not qualify a release. A failed build or failed
installation cannot become successful by changing its label. Experimental
host-image tooling does not replace native FCOS installation and updating.

### Distribution targets

The release request selects `appliance`, `docker`, or `all` before building.
`all` selects both distributions from one source snapshot and common candidate;
it does not rebuild Soda programs or bundled application images for Docker.
The target set and source revisions are fixed for the release tag. Adding a
distribution or changing shipping bytes requires a new release tag.

An appliance release includes its ISO and hash-bound rootfs. A Docker-only
release includes the complete runtime image and its installation metadata;
ISO generation, rootfs hosting and CoreOS installation qualification are not
prerequisites. The common candidate can still use the FCOS host archive as the
Dockerfile's payload source without producing installation media.

Both distributions retain the exact corresponding Soda and Fountain source
archives, shared payload/candidate bindings, provenance and licensing material
in the GitHub release. Docker's runtime layers are delivered through GHCR rather
than duplicated as a GitHub download. Source CI, artifact admission and deployment
qualification are distinct gates. Development outputs cannot be staged or promoted.

The implemented build and artifact boundaries are described in
[native support](../development/native-support.md#build-and-artifact-contract).
The [guided operator flow](../development/release.md#guided-release-operator-flow)
connects preparation, publication, qualification and promotion for all targets.

## Public distribution

Appliance users download one ISO. It automatically retrieves its hash-bound root filesystem
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

The public rootfs contains the host and an OSTree repository inside its EROFS
root. Application OCI bodies are acquired separately from the Payload v4 pins
into ordinary Podman storage. Explicit maintenance consumes the selected
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

The format-2 operation journal freezes both Payload v4 bindings and Fountain's
consistent source snapshot. Required packages and desired-enabled optional
packages select source and target image/package prefetch; disabled Tailnet adds
no acquisition. The updater holds the existing maintenance directory's exclusive
lock while caching both sides, checking retained rollback content, replacing
Project support and preparing exact cached packages through Fountain's stopped,
network-disabled UID1000 CLI. Fountain remains the only package installer and
requested-state writer. Recovery uses the saved snapshot and offline cache;
rollback refuses a changed installation UUID, generation or desired value.

Core finalization verifies active package bindings and unchanged intent, retires
the operation marker, and releases the exclusive lock before asking Fountain to
reconcile optional effects. Exit status 75 means the core committed but extension
readiness is failed or unknown. It preserves last-operation evidence and does
not roll back, restore the marker or overwrite a newer administrator decision.
Restarting reconciliation without a pending marker still checks current extension
readiness. Only prepared public inputs receive a container-readable SELinux label;
private maintenance journals retain their metadata-only service boundary.


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

### Docker artifact identity

The Docker binding identifies the release tag and native platform (`x86_64`,
OCI `linux/amd64`), exact Soda/Fountain source revisions and archive hashes,
shared candidate and payload hashes, and the source host manifest/archive hash.
It also binds the resolved Fedora base manifest digest, Docker recipe hash,
outer image manifest and config digests, exported OCI archive hash/size, RPM
inventory hash, and distributed Compose file hash/size. The common candidate
continues to own the bundled application image and Project-support identities;
Docker references those bindings rather than creating another application catalog.
Exact JSON field names and producer/readback interfaces belong to
[the delivery metadata model](../../lib/soda-release-deliver/README.md#docker-metadata-and-provenance).

The native build resolves the Fedora base before packaging and records the
packages actually installed. A pinned base does not make mutable RPM repositories
reproducible. Packaging reads back Soda's installed revision, payload and bundled
image identities from the resulting runtime. Missing or mismatched identities
refuse preparation and delivery. Neither a local Docker image ID nor a mutable
tag substitutes for the distribution manifest digest.

### Registry staging and recommendation

The public runtime repository is `ghcr.io/levitateos/sodaos`. Its version tag
matches the GitHub release tag; the release metadata also supplies the immutable
`ghcr.io/levitateos/sodaos@sha256:...` reference. Publication preserves and reads
back the recorded image digest. A conflicting existing version tag is refused;
the publisher treats version tags as immutable even though registry tags are
mutable. It never rebuilds or re-encodes a staged image during promotion.

GHCR package visibility must allow anonymous pulls of the staged digest.
Repository linkage, public visibility and publisher access are explicit
prerequisites, not inferred from the repository name. GitHub documents
[Container registry access and digest pulls](https://docs.github.com/en/packages/working-with-a-github-packages-registry/working-with-the-container-registry).
Registry and GitHub publication credentials stay with the release operator,
outside the build worker. Source CI stays read-only.

Staging publishes an explicitly unqualified prerelease with the Docker binding,
exact Compose file, RPM inventory, source archives and shared release metadata.
Qualification receipts are retained locally and incorporated into the editable
release notes during promotion; immutable release assets cannot be appended later.
The `latest` Docker tag is a recommendation pointer moved only to a qualified
version's unchanged digest. Installation uses that version's Compose file and
pins its recorded image digest.

GitHub's latest release remains the appliance ISO download pointer. Promoting
Docker-only releases must not change it. An `all` release can update both
recommendations only after both distributions qualify; one passing target cannot
promote a partially qualified release. Qualification records evidence without
changing either recommendation. GHCR and GitHub changes are not atomic:
partial publication or promotion must report observed remote state, retain its
artifacts and refuse blind retry, overwrite or rollback of recommendation pointers.

### Docker qualification

One fresh deployment on the supported native Docker host uses the publicly
downloaded Compose file and anonymously pulled registry digest, with new runtime
volumes. Independent readback binds the pulled manifest/config, installed Soda
revision, shared payload/application identities and host platform to the release.
Required observations cover:

- Initial image import, service readiness, operator setup and HTTPS activation.
- Native browser login, Project creation/join, terminal access and one nested
  container workload through the existing Project runtime.
- The bounded factory journey with its required provider access and finite
  funding; absent prerequisites are a blocker, not a skipped pass.
- Orderly shutdown, restart and outer-container recreation using the same image
  and the same `/etc`, `/var` and `/home` volumes. Configuration, credentials,
  repository/database records, application data and Project files remain usable;
  initialization and product mutations are not replayed.

The receipt binds the GitHub release/tag, selected target set, source revisions,
candidate/payload hashes, Docker binding and Compose hashes, observed registry
and installed identities, native host prerequisites, and hashes of finalized
observation files. Failed, skipped, missing, stale or identity-mismatched required
observations refuse qualification and promotion. Private credentials and raw
provider responses do not enter public metadata or release notes.

Local development smoke, source CI and appliance ISO qualification do not qualify
Docker. This receipt covers fresh installation and same-image persistence only.
Cross-version replacement, downgrade, backup restoration and rollback require
separate evidence before being offered as supported operations. Docker-qualified
status and maintained recommendation pointers do not automatically upgrade any
running installation.

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
