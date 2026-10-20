# Verify the release download

Select a published SodaOS release and confirm its artifacts belong to that same release before installation. This preserves the release's byte identity and avoids mixing media, metadata or images from different versions.

## Select the distribution

The release's metadata binds installer inputs and payloads to exact bytes. Verify these identities before booting media or starting a Docker deployment.

For a hardware or VM installation, choose the appliance ISO from the SodaOS GitHub Releases page. The ISO retrieves its matching authenticated root filesystem from that same versioned release during installation, so the installer needs working internet access to GitHub. Do not mix media and payloads from different releases. The installer checks release identity and integrity before writing the selected disk.

For Docker, select a release whose notes explicitly identify it as Docker-qualified. Use that release's version-bound `install-docker.sh`; it verifies the release metadata and Compose file and pulls the runtime image by its immutable registry digest. A tag existing or an image being downloadable does not by itself establish Docker qualification. Follow the pinned installer and retain its deployment files as described in [Install with Docker](50-docker.md).

## Verify and keep matching artifacts

1. Open the official [SodaOS releases](https://github.com/LevitateOS/sodaos/releases) page and choose an explicit release tag. For Docker, confirm that its notes mark the release Docker-qualified.
2. On the computer holding the download, use GitHub CLI with access to `LevitateOS/sodaos`. Choose a fresh download directory. Replace `RELEASE_TAG` with the selected tag and `EXACT_ASSET_NAME` with the ISO or saved Docker installer's exact asset name from that release. Do not run these placeholders unchanged.

   ```sh
   release_tag=RELEASE_TAG
   asset_name=EXACT_ASSET_NAME
   gh release download "$release_tag" --repo LevitateOS/sodaos --pattern "$asset_name"
   gh release verify-asset "$release_tag" "$asset_name" --repo LevitateOS/sodaos
   ```

3. Continue only when verification succeeds for that file and the selected repository/tag. [GitHub's asset verification](https://cli.github.com/manual/gh_release_verify-asset) checks the saved bytes against the immutable release's attestation. Keep the verified file with its release identity; boot or execute those same bytes. If transferring it to another computer, verify the transferred file before use.

Changed bytes, an asset absent from that release, a missing attestation, or a network or verification failure do not establish trust. Stop, resolve the reported cause and repeat verification against the same selected identity. If GitHub CLI needs sign-in for access, use its normal authentication flow on your download computer; do not place access tokens in the installer command.

Checksums can detect corruption, but a checksum downloaded beside an unauthenticated file does not establish who released it. If using a checksum manifest, verify that manifest as an asset of the same immutable release before checking its entries. Keep metadata and any verification material attached to that release together; do not combine assets from different tags.

Pre-boot verification authenticates the ISO itself. During installation, the ISO separately authenticates the matching root filesystem downloaded from that release. Likewise, verifying the saved Docker installer comes before executing it; its later metadata, Compose and immutable image-digest checks protect downstream inputs. Keep all of these checks enabled.

## Continue installation

The download process does not require a repository checkout or a local build toolchain. You do not assemble the appliance image yourself. Once the release artifacts are verified, continue to [install on premises](40-install-on-premises.md), [prepare the VM](30-deploy-to-cloud.md), or [install the Docker distribution](50-docker.md).
