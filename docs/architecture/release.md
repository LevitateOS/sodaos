# Release architecture

Soda ships as a **minimum-deviation Fedora CoreOS** appliance. An exact-source
candidate carries the complete Soda payload; GitHub Releases owns public delivery.

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
registry channel, publication ledger or custom update-discovery service.

Every asset must be smaller than GitHub's 2 GiB limit. Packaging removes unused
host copies and compiler debug/symbol data, then admission checks the actual bytes;
estimated compression savings never authorize an upload. The release includes the
exact Soda and Fountain source archives, package inventory, provenance and checksums.

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

The appliance follows Fedora CoreOS update mechanics (rpm-ostree / Zincati-aligned
operation). Soda does not invent a parallel host updater for ordinary maintenance.
First-install evidence does not qualify an upgrade or recovery path. Those paths
need their own evidence before being offered as supported release operations.

## Architectures

SodaOS supports **x86_64 only**. AArch64/ARM64 is unsupported and excluded from
current implementation, build, delivery and qualification scope. Supporting it
requires a future explicit product decision; upstream platform support or existing
experimental selectors do not establish Soda support.

Qualification requires native Linux x86_64 evidence. Cross-compilation, emulation
and development checks on another host architecture do not qualify the appliance.

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
