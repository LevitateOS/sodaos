# Run Soda OS with Docker

Run the complete Soda application runtime, including persistent Projects and
nested containers, on a dedicated x86-64 Linux Docker Engine host. Docker owns the
outer container; you maintain the host operating system.

**Docker support is in preparation. No qualified Docker release or tested
one-command installer URL is available yet.** Existing v0.1.0 and v0.1.1 releases
provide installation media. Use [Install on premises](20-install-on-premises.md)
or [Deploy in a VM or cloud](10-deploy-to-cloud.md) for those downloads.

## Installation requirements

- A dedicated native x86-64 Linux host or VM with rootful Docker Engine, cgroup v2
  and Docker Compose 2.17 or newer.
- `curl`, `jq`, OpenSSL, coreutils, util-linux and the `ip`/`ss` networking tools.
- Root, Docker administrator access or working sudo. Soda runs privileged with
  host networking and has administrative authority over this host.
- A private IPv4 address assigned to the host and reachable by your clients,
  persistent IPv4 forwarding, and a Project subnet that does not overlap host,
  Docker, LAN or VPN networks. Soda also reserves `10.88.0.0/16` and `10.90.0.0/24`.
- Available TCP ports 443, 2222, 3000 and 9090. Restrict access to your intended
  private clients; the installer does not modify your firewall.
- Sufficient storage for the runtime, acquired application images, Projects and
  backups. Docker data storage and the installation directory must have room to grow.

Rootless Docker, Docker Desktop, macOS, ARM64 and WSL2 are unsupported by this
installation path. Apple Silicon/UTM work remains in
[the platform backlog](https://github.com/LevitateOS/sodaos/issues/45).

## Install a qualified release

When available, this section will contain the tested command using an exact
published `install-docker.sh` URL. A qualified Docker release must supply that
script, `docker.json`, `sodaos-docker-compose.yaml`, a publicly pullable GHCR digest
and qualification results in its release notes. Select it explicitly from
[GitHub releases](https://github.com/LevitateOS/sodaos/releases); the latest ISO
release does not select Docker.

The version-bound installer checks prerequisites, release qualification,
metadata, Compose checksums and image identity. It anonymously pulls the exact
image digest, retains the deployment under `/home/sodaos`, and starts it with a
bounded service-readiness check. It prompts through your terminal, including when
streamed into Bash. It never substitutes an unverified image tag.

The interactive flow handles installation and then guides these user-owned steps:

1. Reach the initial Forgejo installation through trusted host SSH forwarding,
   complete the browser form and create your administrator account.
2. Create that administrator's `read:user` bootstrap token and enter it at the
   installer's hidden prompt. Native setup revokes it after successful binding.
3. Let the installer activate private HTTPS and export the **public** CA
   certificate. Transfer it through trusted host SSH, compare the displayed
   SHA-256 fingerprint, and add it to each intended client's trust store.
4. Open your private HTTPS address and complete browser login. Configure client
   routes for direct Project SSH/services and separately enroll any factory provider.

The installer never installs client trust or disables certificate verification.
Leaving the token empty leaves browser setup explicitly pending. Rerun the same
release interactively to continue. `--yes` accepts privileged runtime installation
without interactive browser enrollment; it does not claim that setup is complete.

## Persistence, reruns and recovery

Keep `/home/sodaos`, its private installation identity, original Compose and `.env`
files, and Compose project name `sodaos`. The named volumes retain:

| Volume | Runtime path | Persistent data |
| --- | --- | --- |
| `sodaos_config` | `/etc` | Configuration and credentials |
| `sodaos_state` | `/var` | Databases, repositories, Podman storage and application state |
| `sodaos_home` | `/home` | Writable Project roots and home directories |

Rerunning the **same release** with the same operator and directory preserves the
configuration, image identity and volumes. It refuses a changed release, subnet,
browser address or conflicting deployment. Cross-version upgrades are not yet
supported. A failed start retains state for diagnosis and a same-version retry.

From the retained installation directory, use `docker compose stop` to stop the
runtime and `docker compose up -d --wait --wait-timeout 1800` to start it. Recreating
only the outer container with the same pinned image retains the named volumes.
**Do not use `down --volumes` or prune retained Project storage.**

Back up all three volumes together while the runtime is stopped. Restore them
together using the same image and Compose identity. Restart/recreation evidence
alone does not qualify a cross-version upgrade or backup restoration.

Detailed prerequisites, installer options, diagnostics, manual inspection,
certificate handling and recovery procedures live in the
[Docker operator guide](https://github.com/LevitateOS/sodaos/blob/main/docs/guides/docker.md).
