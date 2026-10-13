# Verify downloads

Verify the release artifacts for your software factory before booting media or executing an installer.

The factory and persistent human projects use the same appliance release. Verify
the matching architecture and accompanying installation instructions before setup.

## Choose one release

Start at the [SodaOS releases](https://github.com/LevitateOS/sodaos/releases/latest)
from the official website. Read that release's notes and verification instructions.
Choose the x86-64 artifact for x86-64 hardware or a matching VM.

Use the release installation media for a PC or a VM. Keep its verification
metadata from the same release. The guided installer supplies the matching Soda
components and creates the machine's provisioning from your choices; you do not
assemble application archives or prepare a separate Ignition document.

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

If a release also publishes a compressed disk image for a particular platform,
use its matching import instructions. Verify the compressed file before
decompression and the resulting disk against its own digest when supplied;
these are different byte sequences. Preserve existing disks.

## Stop on a mismatch

A wrong architecture, missing expected metadata, unexpected signer, invalid
signature or checksum mismatch means stop. Keep the diagnostic and obtain the
correct files through the official release channel. Do not boot the image or
remove a verification step to continue.

Verification establishes artifact identity, not permission to erase a disk or
proof of a backup. Continue with [guided installation](20-install-on-premises.md)
or [cloud/VM deployment](10-deploy-to-cloud.md).
