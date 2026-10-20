# Verify the release download

Select a published SodaOS release and confirm its artifacts belong to that same release before installation. This preserves the release's byte identity and avoids mixing media, metadata or images from different versions.

## Select the distribution

The release's metadata binds installer inputs and payloads to exact bytes. Verify these identities before booting media or starting a Docker deployment.

For a hardware or VM installation, choose the appliance ISO from the SodaOS GitHub Releases page. The ISO retrieves its matching authenticated root filesystem from that same versioned release during installation, so the installer needs working internet access to GitHub. Do not mix media and payloads from different releases. The installer checks release identity and integrity before writing the selected disk.

For Docker, select a release whose notes explicitly identify it as Docker-qualified. Use that release's version-bound `install-docker.sh`; it verifies the release metadata and Compose file and pulls the runtime image by its immutable registry digest. A tag existing or an image being downloadable does not by itself establish Docker qualification. Follow the pinned installer and retain its deployment files as described in [Install with Docker](50-docker.md).

## Verify and keep matching artifacts

Open the official [SodaOS releases](https://github.com/LevitateOS/sodaos/releases) page and select one release entry. Confirm that the ISO or Docker installer, metadata, and any associated verification material are attached to that same entry and tag; keep them together. For Docker, continue only when the release notes explicitly mark that tag Docker-qualified. If a required artifact or its verification material is missing, an identity check fails, or qualification is unclear, stop rather than combining assets from another tag.

The ISO installation authenticates the matching root filesystem downloaded from that release. Docker's version-bound installer checks its release metadata and Compose bytes and pulls its image by immutable registry digest. Confirm the ISO itself is the official artifact attached to the selected release, and proceed only when the bytes and release identity can be established from that release's metadata and verification material.

## Continue installation

The download process does not require a repository checkout or a local build toolchain. You do not assemble the appliance image yourself. Once the release artifacts are verified, continue to [install on premises](30-install-on-premises.md), [prepare the VM](40-deploy-to-cloud.md), or [install the Docker distribution](50-docker.md).
