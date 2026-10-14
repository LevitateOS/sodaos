# Run Soda OS with Docker

Docker runs the complete Soda application runtime directly on the host kernel.
The runtime includes the native host helper, identity broker, systemd, Podman,
Forgejo, PostgreSQL, Caddy and the release's bundled Project images. Podman remains
the internal Project and application engine. Docker owns the outer container's
lifecycle; the operator owns the host operating system.

## Host requirements

Use a dedicated x86_64 Linux Docker Engine host with cgroup v2 and Compose 2.15 or
later. Rootless Docker and Docker Desktop are outside this deployment contract.
Allow capacity for the runtime, retained Projects, nested workloads and backups.
The complete payload occupies several gigabytes before Project data is created.

The runtime needs privileged execution for nested containers and systemd, and
host networking for private browser endpoints, Project bridges and Tailnet.
It has host administrative authority: Docker's privileged mode grants all
capabilities and host-device access ([Docker documentation](https://docs.docker.com/engine/containers/run/#runtime-privilege-and-linux-capabilities)).
The appliance's SELinux dashboard confinement is not provided by this Docker
deployment. See the [trust boundary](../architecture/trust.md#docker-deployment).
The Docker socket and host root filesystem are not mounted into the runtime.

The host must provide IPv4 forwarding for Project routing. Enable it through the
host's normal network configuration. Select a private browser address and a
Project subnet that do not conflict with the host's LAN, Docker or VPN networks.
[Networking](../architecture/networking.md) owns client routing requirements.

The selected browser HTTPS/Git ports and Cockpit port 9090 must be available on
the host. Forgejo initially uses loopback ports 3000 and 2222. NetworkManager,
DNS, kernel auditing and sysctl initialization are managed by Docker/the host,
not started inside the runtime.

## Start and configure

Use the [Compose file](../../system/docker/compose.yaml). Set `SODA_IMAGE` to the
selected Docker runtime image, preferably by immutable registry digest. Select
`SODA_PROJECT_SUBNET` before the first start; the default is `10.89.0.0/24`.

```sh
export SODA_IMAGE=YOUR_SELECTED_RUNTIME_IMAGE
export SODA_PROJECT_SUBNET=10.89.0.0/24
docker compose -f system/docker/compose.yaml up -d
docker compose -f system/docker/compose.yaml exec soda \
  systemctl is-active soda-image-import.service soda-postgres.service forgejo.service
```

Initial startup imports the bundled images and initializes database credentials
before starting Forgejo. This can take several minutes. Project configuration is
created from the release's canonical defaults only when absent. Changing the
subnet environment variable later does not rewrite an existing configuration.

Use the [deliberate operator setup](operator-setup.md#deliberate-setup-with-custom-configuration)
path. Forward port 3000 over SSH to the **Docker host** to reach Forgejo's initial
loopback installer, then create its native administrator and restricted bootstrap
token. Run native setup and activation in the runtime:

```sh
docker compose -f system/docker/compose.yaml exec soda bash
```

Inside that root terminal, use `soda-setup` and `soda-activate` as documented in
the operator guide. Put the bootstrap token in a mode-0600 file through hidden
input; never pass it in command arguments. For Cockpit access, set the runtime's
root password with `passwd`. Host SSH credentials and the runtime root password
are separate. The media-only `soda-install disk` and CoreOS-guided continuation
are not Docker setup commands.

Copy only the public CA certificate for client trust, verify browser login and a
joined Project terminal, and configure separate client routes for direct Project
SSH/services. Then complete the normal [factory configuration](../reference/factory.md).

## Persistence and maintenance

Compose retains `/etc`, `/var` and `/home` in named volumes. Together they hold
configuration, credentials, database/repository data, Podman storage and writable
Project roots, broker custody, Tailnet state, application data and backups.
`/run` and `/tmp` are ephemeral. Keep the Compose project name stable so subsequent
starts select the same volumes.

```sh
docker compose -f system/docker/compose.yaml stop
docker compose -f system/docker/compose.yaml start
```

Stopping sends systemd its shutdown signal and allows five minutes for service
shutdown. Docker can report exit code 130 after an orderly halt: Linux reports
a PID-namespace halt as SIGINT ([reboot(2)](https://man7.org/linux/man-pages/man2/reboot.2.html)).
Removing the outer container preserves named volumes; `down --volumes`
deletes them and is destructive. Do not prune Podman storage to repair a Project.

Use the existing [PostgreSQL backup command](../../cmd/soda-pg-maintenance/README.md#back-up)
inside the runtime. A complete recovery backup also needs the retained Project
roots, repositories, configuration and broker key/custody. For a consistent
volume backup, stop the runtime and back up all three volumes together. Restore
them together using the same runtime image before starting services.

The host OS is maintained outside Soda. CoreOS disk installation, rpm-ostree,
Zincati and the appliance's explicit release-maintenance command do not manage
this Docker deployment. Container recreation with the same image preserves state;
changing image versions requires a separately verified retained-state transition.
The [release contract](../architecture/release.md#docker-runtime) owns delivery
and qualification.

## Build the runtime image

This is release-builder work; users of a published runtime image do not build
components. Run the admitted controller from its clean committed checkout or
linked worktree, with its matching worker configuration. See [native build
admission](../development/native-support.md#local-host-content-image-candidate) for setup and custody.

```sh
# Docker-only production packaging: no media authority or rootfs URL.
sudo /ADMITTED/soda-build --worker-config /RESTRICTED/worker.json \
  --arch x86_64 --distribution docker --release-tag v0.1.0 \
  --out /OUTPUT_PARENT/UNIQUE --forgejo-source /CANONICAL/FORGEJO-CHECKOUT

# For development, add --development --target docker to the same command.
```

The builder freezes the Fedora base digest before compiling, produces the normal
complete candidate once, then uses its verified local host config ID as the
Dockerfile's appliance stage. Podman packages and exports the outer image without
a Docker daemon socket. Soda binaries, shared OCI payload, native units, defaults
and notices are copied unchanged; CoreOS boot/disk services are excluded.

The attempt produces `image.oci`, `docker.json`, `packages.txt`
and `compose.yaml` under `artifacts/docker/`. The verifier reads the exported
image, compares candidate-owned files and every bundled blob with the candidate,
and binds its platform, labels, sources, pinned Fedora base and observed outer
RPM inventory. A mismatch fails the build before its success receipt. The
internal `evidence/build.json` records distribution, version, completion boundary
and Docker metadata location. These are build outputs, not qualification evidence.

For a local development run, import the verified archive into Docker:

```sh
skopeo copy oci-archive:/ABS/PATH/TO/artifacts/docker/image.oci \
  docker-daemon:sodaos-docker:local
export SODA_IMAGE=sodaos-docker:local
docker compose -f /ABS/PATH/TO/artifacts/docker/compose.yaml up -d
```

Production `--distribution all` packages the same candidate for Docker and the
appliance; supply the existing media inputs and a rootfs URL for the same release
tag. The [release operator flow](../development/release.md#docker-release-operator-flow)
owns GHCR staging, public-image qualification and promotion; those publisher
commands are not yet implemented. A locally built image is not a published or
qualified Docker release.
