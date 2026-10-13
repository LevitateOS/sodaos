# soda-release-deliver

GitHub release preparation, immutable prerelease publication and qualification-bound
promotion. Rust imports use `soda_release_deliver`; operators use `soda-release`
from [release-tools](../soda-release-tools/README.md).

## Boundaries

- `publish::prepare` binds the current candidate, media, source archives and
  provenance to a fresh local asset set and checks GitHub's per-file size limit.
- `publish::stage` creates a missing tag at the CI-qualified built revision or
  verifies the existing tag, uploads a complete draft, compares
  remote sizes and hashes, then publishes an explicitly unqualified immutable
  prerelease. It never overwrites an existing release.
- `publish::promote` binds the operator's observed installed revision and explicit
  install/boot/image smoke confirmation to the downloaded ISO, generates the
  receipt, verifies the download through GitHub and promotes unchanged assets to latest.
- `check::check_candidate` verifies native candidate archives and identities.
- `native` retains the restricted Skopeo boundary for local OCI artifact signing
  and verification. These packaging keys do not replace GitHub's release attestation.

GitHub owns release immutability and publication state. There is no registry
channel, observed-authority database or publisher ledger. Failed or uncertain
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
