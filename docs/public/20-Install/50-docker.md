# Install with Docker

Deploy the released SodaOS runtime on a dedicated Linux Docker host when you can administer the host and accept the runtime's privileged access. Docker manages the outer container lifecycle; the Soda runtime uses its own internal services and Podman for Projects.

Use an x86_64 Linux host with rootful Docker Engine, Compose, cgroups v2 and host networking. Rootless Docker and Docker Desktop are outside this deployment model. The host must provide IPv4 forwarding for Project routing and persistent storage for the runtime's `/etc`, `/var` and `/home` data. Reserve the browser HTTPS, Git and Cockpit ports and choose a private browser address plus a nonoverlapping Project subnet. Allow access to the release assets and registries during initial installation.

The runtime needs privileged execution for systemd and nested containers. This gives it substantial authority over the host kernel and devices; it does not provide appliance SELinux confinement or isolate mutually untrusted teams. The operator remains responsible for Linux security, firewall policy, client routes, capacity and backups.

Use the version-bound `install-docker.sh` from an explicitly Docker-qualified release. Obtain the script from that release and verify its identity against the immutable release before running it. Select a protected install directory, then invoke the script through Bash:

```sh
script_path=/home/operator/downloads/install-docker.sh
install_dir=/home/sodaos
sudo bash "$script_path" --install-dir "$install_dir"
```

Replace `script_path` with the verified script's absolute path and `install_dir` with the protected location you choose. Keep that same directory for later Compose commands. The installer verifies the release metadata and Compose bytes, pulls the runtime by immutable image digest, checks prerequisites before creating the deployment, and guides initial setup. It does not install Docker or alter host firewall and forwarding settings. A failed start retains the deployment files and volumes for diagnosis. Re-run only the same version with the same operator and install directory; conflicting files or retained resources require inspection.

The installer stores deployment configuration under its protected install directory and keeps persistent runtime state in Docker volumes for `/etc`, `/var` and `/home`. These contain configuration and credentials, Forgejo/PostgreSQL data, Podman storage, Project roots and homes, broker custody, optional Tailnet state, application data and backups. Preserve the Compose project name and the original configuration when restarting or recreating the same deployment:

```sh
install_dir=/home/sodaos # use the same directory selected during installation
cd "$install_dir"
docker compose stop
docker compose up -d --wait --wait-timeout 1800
```

Use the same Compose file and `.env` for recreation so Docker selects the existing volumes and pinned image. Docker image replacement is not an appliance release upgrade; follow [updates and fallback](../70-Operate/50-updates-and-fallback.md) for lifecycle boundaries and retain data backups before maintenance. Do not delete volumes or prune state as a repair step.

After the runtime is ready, complete [operator setup](../30-Set-up/10-operator-setup.md) in the Docker context. The released installer can manage browser enrollment, or an operator can run the lower-level `soda-setup` and `soda-activate` inside the runtime with `docker compose exec`. Do not run the appliance-only `soda-install configure` command in Docker. Finish with [private networking and TLS](../30-Set-up/20-private-networking-and-tls.md) and verify access from an intended client.
