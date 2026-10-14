# soda-release-deliver

GitHub release preparation, immutable prerelease publication and qualification-bound
promotion. Rust imports use `soda_release_deliver`; operators use `soda-release`
from [release-tools](../soda-release-tools/README.md).

## Boundaries

- `publish::prepare` binds one selected `appliance`, `docker` or `all` candidate
  to a fresh exact public asset set; large registry archives and operator paths stay local.
- `publish::stage` checks prepared bytes, exact-source CI and tag, uploads/reads back
  the Docker version anonymously, then publishes the complete immutable GitHub prerelease.
- `publish::qualify_appliance` retains the existing downloaded-ISO/native-smoke gate.
  `publish::qualify_docker` verifies the actual public installation downloads, registry
  identity and finalized native observations with the existing acceptance reader.
- `publish::promote` rechecks all selected receipts and original evidence, preserves
  the public image digest while moving Docker latest, and updates only eligible GitHub
  recommendations. Docker-only promotion confirms appliance latest is unchanged.
- `publish::status` observes identities, remote pointers, blockers and partial attempts
  without mutation. Failed promotions retain their attempt marker and refuse replay.
- `check::check_candidate` verifies native candidate archives and identities.
- `docker::record` derives and writes Docker metadata once from the completed
  artifacts. `docker::verify` recomputes its binding; the candidate checker invokes
  it whenever a `docker/` artifact directory is present. `docker::admit_publication`
  additionally requires production purpose. These APIs do not build, publish or
  qualify a runtime; the integrated producer and GHCR transport are separate owners.
- `upgrade` admits one complete downloaded release, validates its external native
  transition receipt, and stamps a receipt only from the fixed concrete native
  observation files. Prospective source declarations alone grant no upgrade
  authority. The rootfs contains all maintenance bytes and application images.
- `native` retains the restricted Skopeo boundary for local OCI artifact signing
  and verification. These packaging keys do not replace GitHub's release attestation.

GitHub owns release immutability and publication state. Registry promotion moves
only the qualified digest recommendation; it adds no observed-authority database
or publisher ledger. Failed or uncertain
publication is inspected before any retry; uploads never use clobbering.

Staging and promotion both require the latest successful `CI` push run on `main`
for the built Soda commit. The built Fountain revision must match that commit's
`fountain-source.json`. The release notes link to the admitted CI run. CI checks
source; the separate native smoke still qualifies the actual downloaded media.

The `gh` client needs repository Contents write and Actions read permission. Staging also reads the
repository's immutable-release setting, which requires Administration read access;
enabling it is a one-time Administration write operation. Local signing uses the
Skopeo version in [tools.json](tools.json) and restricted private-file inputs.
Keys, passphrases, credentials and private VM state never enter the asset set.

These operations do not install images, activate a host or reboot. Follow the
[release workflow](../../docs/development/release.md) for commands and the
[release architecture](../../docs/architecture/release.md) for the bounded native
qualification and single-installer download contract.

## Docker metadata and provenance

`model::DockerMetadata` owns format 1 of `artifacts/docker/docker.json`.
Its strict PascalCase JSON requires every field, rejects unknown/duplicate keys,
and follows the existing hash convention: file `SHA256` values are bare lowercase
hex; OCI manifest/config digests carry `sha256:`. `FileIdentity` binds both
`SHA256` and `Bytes`. No artifact paths or registry URLs are accepted from the
document: the verifier reads fixed paths beneath the candidate directory.

| Fields | Binding |
| --- | --- |
| `Format`, `Distribution`, `Purpose`, `ReleaseTag` | Format 1, `docker`/`all`, `development`/`production`, and an exact non-`latest` version tag |
| `Architecture`, `Platform` | Native `x86_64` and OCI `linux/amd64` |
| `SodaRevision`, `ForgejoRevision`, `SodaSource`, `ForgejoSource` | Exact revisions and hashes/sizes of `source.tar` and `forgejo-source.tar`; archive commit IDs and the archived Fountain pin must match |
| `CandidateSHA256`, `PayloadSHA256`, `HostManifest`, `HostArchiveSHA256` | The admitted common candidate and payload plus its verified source host |
| `FedoraBase`, `Recipe` | `registry.fedoraproject.org/fedora@sha256:...` and the Dockerfile's hash/size from the Soda source archive |
| `Image`, `ImageArchive` | Existing canonical OCI `Image` identity, including source/base attribution, plus the hash/size of `docker/image.oci` |
| `Packages`, `Compose` | Hashes/sizes of `docker/packages.txt` and `docker/compose.yaml`; Compose must equal archived source and RPM inventory must be sorted, unique and valid |

The runtime readback verifies the embedded payload, candidate content inventory,
Project terminal-support bytes, host-bound service/Canlang files, original host
RPM inventory, corresponding-source licenses/notices, Docker init/import overrides,
and the outer RPM inventory. It checks the shared layout's index against the source
host and every embedded application manifest/config/layer blob against its
hash-bound candidate OCI archive, reusing the existing bounded OCI reader.
Copied blobs use member limits from those verified archives, including layers
larger than the ordinary 512 MiB file limit. Ordinary file limits and the
16 GiB total decoded-image budget still apply; compressed layers retain their
separate 4 GiB budget.
These are artifact checks; native service/browser/factory behavior still needs
[Docker qualification](../../docs/architecture/release.md#docker-qualification).

The producer supplies `DockerRequest` from its frozen release request and labels
the outer image before recording metadata. Existing OCI source/revision/base
labels remain mandatory. Additional labels are:

| Label | Required value |
| --- | --- |
| `io.soda.runtime` | `docker` |
| `org.opencontainers.image.version` | `ReleaseTag` |
| `io.soda.build-purpose` | `Purpose` |
| `io.soda.distribution` | `Distribution` |
| `io.soda.candidate-sha256` | `CandidateSHA256` |
| `io.soda.payload-sha256` | `PayloadSHA256` |
| `io.soda.docker.recipe-sha256` | `Recipe.SHA256` |

Changing a development metadata field to production does not qualify it: the
unchanged image's purpose label must agree, and publication admission refuses
development purpose. Record creation refuses existing metadata; verification
rejects changed files, linked artifacts, missing members and mismatched labels.

Before any registry write, the future publisher must call
`docker::admit_publication`. After an independent anonymous registry readback,
`DockerMetadata::verify_registry` compares the canonical inspected image identity.
`reference()` supplies `ghcr.io/levitateos/sodaos@sha256:...`; a Docker daemon image
ID or version tag alone is insufficient. Neither a matching registry image nor
these metadata checks grant qualification or permission to move `latest`.
