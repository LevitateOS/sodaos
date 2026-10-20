# Verify the release download

Install only a published SodaOS release and its matching distribution artifacts. The release's metadata binds the installer inputs and payloads to exact bytes so you can verify them before booting media or starting a Docker deployment.

For a hardware or VM installation, choose the appliance ISO from the SodaOS GitHub Releases page. The ISO retrieves its matching authenticated root filesystem from that same versioned release during installation, so the installer needs working internet access to GitHub. Do not mix media and payloads from different releases. The installer checks release identity and integrity before writing the selected disk.

For Docker, select a release whose notes explicitly identify it as Docker-qualified. Use that release's version-bound `install-docker.sh`; it verifies the release metadata and Compose file and pulls the runtime image by its immutable registry digest. A tag existing or an image being downloadable does not by itself establish Docker qualification. Follow the pinned installer and retain its deployment files as described in [Install with Docker](50-docker.md).

Download release files only from the official [SodaOS releases](https://github.com/LevitateOS/sodaos/releases) page. Check that the selected release and every artifact belong to the same tag. Keep the published metadata, installer or ISO, and any verification information together. If an artifact is missing, its identity check fails, or its release qualification is unclear, stop and select a clearly published release rather than substituting files from another version.

The download process does not require a repository checkout or a local build toolchain. You do not assemble the appliance image yourself. Once the release artifacts are verified, continue to [install on premises](30-install-on-premises.md), [prepare the VM](40-deploy-to-cloud.md), or [install the Docker distribution](50-docker.md).
