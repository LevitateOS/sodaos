# Verify downloads

Check the origin, architecture and exact bytes of a SodaOS download before booting media or executing an installer.

## Choose one release

Start at the [SodaOS releases](https://github.com/LevitateOS/sodaos/releases/latest)
from the official website. Read that release's notes and verification instructions.
Choose x86-64 for x86-64 hardware, or AArch64 for ARM64 hardware and matching VMs.

Use the ISO for installation media, or QCOW2 for disk-image import. Keep the
selected artifact, its verification metadata and accompanying deployment recipe
from the same release together. A compressed image and its decompressed disk
are different byte sequences; verify each against the corresponding digest.

Do not infer filenames, signing identities or update-image references from an
older product or another repository. An application container archive is not a
bootable machine disk.

## Establish a trusted expected digest

Use the verification method and trust material published for that release.
Establish the expected signing identity/key or checksum through a trusted channel
before executing anything from the downloaded bundle. Where signed metadata is
provided, authenticate it before trusting the file digests it contains.

A SHA-256 sidecar downloaded beside an image detects corruption but does not by
itself authenticate the publisher: an attacker could replace both. Do not trust
a bundled verifier merely because the same unverified bundle contains it.

## Compare file bytes

On Linux, replace the filename with the exact downloaded artifact:

```sh
sha256sum EXACT_DOWNLOADED_FILENAME
```

On macOS:

```sh
shasum -a 256 EXACT_DOWNLOADED_FILENAME
```

On Windows PowerShell:

```powershell
Get-FileHash -Algorithm SHA256 -LiteralPath 'EXACT_DOWNLOADED_FILENAME'
```

Compare the entire digest with the authenticated expected value. If the release
provides a checksum list, inspect its filenames and use your tool's check mode
from the download directory, for example `sha256sum --check SHA256SUMS` on Linux.
Do not use broad wildcards that may select a different release.

For a `.zst` image, verify the compressed file before decompression with `zstd`.
Do not overwrite an existing disk. Verify the uncompressed image when its digest
is supplied, then use the release's import instructions.

## Stop on a mismatch

A wrong architecture, missing expected metadata, unexpected signer, invalid
signature or checksum mismatch means stop. Keep the diagnostic and obtain the
correct files through the official release channel. Do not boot the image or
remove a verification step to continue.

Verification establishes artifact identity, not permission to erase a disk or
proof of a backup. Continue with [ISO installation](20-install-on-premises.md)
or [cloud/VM deployment](10-deploy-to-cloud.md).
