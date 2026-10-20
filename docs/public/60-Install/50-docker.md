# Install with Docker

Deploy SodaOS on a dedicated Linux Docker host and manage the privileged runtime and its persistent state. This path gives the operator direct responsibility for the outer host as well as Soda services.

Docker manages the outer container lifecycle; the Soda runtime uses its own internal services and Podman for Projects. Use a dedicated host you can administer and accept the runtime's privileged access.

## Host prerequisites and trust

Use an x86_64 Linux host with rootful Docker Engine, Compose 2.17 or later, cgroups v2 and host networking. Rootless Docker, Docker Desktop and WSL2 are outside this deployment model. The host must provide IPv4 forwarding for Project routing and persistent storage for the runtime's `/etc`, `/var` and `/home` data.

Install the Compose plugin, Bash, `curl`, `jq`, coreutils and OpenSSL, plus the standard host tools providing `ip`, `ss`, `sysctl`, `flock`, `awk` and `grep`. Use [deployment capacity](10-deployment-options.md#select-capacity-for-your-workload) to account for the intended workload, retained data, update/backup reserve and outer-host overhead.

The installer requires all of these TCP ports to be available on the host. Availability does not mean they should all be exposed to clients:

| TCP port | Purpose | Intended access |
| --- | --- | --- |
| 443 | Private browser HTTPS | Intended private clients after activation |
| 2222 | Native Forgejo Git SSH | Authorized Git clients |
| 3000 | Forgejo bootstrap web listener | Host loopback; use the setup SSH tunnel |
| 9090 | Cockpit administration | Private operator access |

Choose a private IPv4 browser address assigned to the host. The default Project subnet is `10.89.0.0/24`; check it against LAN, Docker, VPN and other routes before accepting it. Soda reserves `10.88.0.0/16` and `10.90.0.0/24` and refuses overlap. Allow access to release assets and registries during initial installation. The host's existing SSH service supplies the bootstrap tunnel; Git SSH port 2222 is a separate service.

The runtime needs privileged execution for systemd and nested containers. This gives it substantial authority over the host kernel and devices. The deployment does not provide appliance SELinux confinement or isolate mutually untrusted teams. The operator remains responsible for Linux security, firewall policy, client routes, capacity and backups.

## Run the released installer

Use the version-bound `install-docker.sh` from an explicitly Docker-qualified release. Follow these steps:

1. Save that release's installer and complete [download verification](20-verify-downloads.md#verify-and-keep-matching-artifacts) before execution. Run the following commands on the intended Docker host with administrative access to its rootful Docker daemon.
2. As the non-root operator who will maintain this deployment, choose a protected install directory and invoke the script through Bash. The installer requests sudo when needed for Docker or directory creation and keeps deployment files owned by the invoking operator. Replace `script_path` with the verified script's absolute path and `install_dir` with the protected location you choose:

   ```sh
   script_path=/home/operator/downloads/install-docker.sh
   install_dir=/home/sodaos
   bash "$script_path" --install-dir "$install_dir"
   ```

The installer verifies release metadata and Compose bytes, pulls the runtime by immutable image digest, checks prerequisites before creating the deployment, and guides initial setup. It does not install Docker or alter host firewall and forwarding settings. A failed start retains the deployment files and volumes for diagnosis. Re-run only the same version with the same operator and install directory; conflicting files or retained resources require inspection.

### Installer options

| Option | Meaning |
| --- | --- |
| `--bind-ip PRIVATE_IPV4` | Select a reachable private IPv4 address assigned to this host; otherwise choose it interactively. |
| `--project-subnet CIDR` | Select the private Project range; the default is `10.89.0.0/24`, subject to conflict checks. |
| `--install-dir /home/PATH` | Select the protected deployment directory; the default is `/home/sodaos`. Retain the same directory for later runs. |
| `--yes` | Accept privileged execution and install the runtime without interactive browser enrollment. It leaves operator setup pending and does not waive identity, host, port or subnet validation. Supply the address and subnet explicitly for an unattended run. |

After the installer completes, inspect `docker compose ps` from its installation directory as that same operator. If the account requires sudo for the rootful Docker daemon, prefix subsequent `docker compose` commands with `sudo`; directory access remains under the operator's account.

A healthy runtime means the internal services are ready for operator setup; it does not prove browser enrollment, certificate trust or client routing. If startup fails, use `docker compose exec soda systemctl --failed` and the [Administration guide](../50-Operate/20-administration.md#inspect-a-docker-deployment) before retrying. Retain files and volumes throughout diagnosis.

## Preserve deployment state

The installer stores deployment configuration under its protected install directory and keeps persistent runtime state in Docker volumes for `/etc`, `/var` and `/home`. These contain configuration and credentials, Forgejo/PostgreSQL data, Podman storage, Project roots and homes, broker custody, optional Tailnet state, application data and backups.

Preserve the Compose project name and original configuration when restarting or recreating the same deployment. In the commands below, replace `/home/sodaos` with the same `install_dir` you selected during installation:

```sh
install_dir=/home/sodaos # use the same directory selected during installation
cd "$install_dir"
docker compose stop
docker compose up -d --wait --wait-timeout 1800
```

Use the same Compose file and `.env` for recreation so Docker selects the existing volumes and pinned image. Docker image replacement is not an appliance release upgrade; follow [updates and fallback](../50-Operate/70-updates-and-fallback.md) for lifecycle boundaries and retain data backups before maintenance. Do not delete volumes or prune state as a repair step.

## Finish operator setup

After the runtime is ready, complete [operator setup](60-operator-setup.md) in the Docker context. The released installer can manage browser enrollment, or an operator can run the lower-level `soda-setup` and `soda-activate` inside the runtime with `docker compose exec`. Do not run the appliance-only `soda-install configure` command in Docker. Finish with [private networking and TLS](../40-Configure/50-private-networking-and-tls.md) and verify access from an intended client.
