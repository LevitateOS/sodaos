# Run Soda OS with Docker

Docker runs the complete Soda application runtime directly on the host kernel.
The runtime includes the native host helper, identity broker, systemd, Podman,
the deployment metadata for Forgejo, PostgreSQL, Caddy and Project images. Podman remains
the internal Project and application engine. Docker owns the outer container's
lifecycle; the operator owns the host operating system.

Website-facing operator instructions live in the
[public Docker handbook](../public/60-Install/50-docker.md).

## Host requirements

Use a dedicated x86_64 Linux Docker Engine host with cgroup v2 and Compose 2.17 or
later, including the [bounded startup wait](https://github.com/docker/compose/blob/v2.17.0/cmd/compose/up.go).
Rootless Docker and Docker Desktop are outside this deployment contract.
Allow capacity for the runtime, retained Projects, nested workloads and backups.
The complete payload occupies several gigabytes before Project data is created.
Use the [measured deployment capacity guide](../public/60-Install/10-deployment-options.md#select-capacity-for-your-workload)
for the four-CPU, 8 GiB RAM, 96 GiB disk starting allocation, measured workloads and
update/backup reserve. Those resources cover the dedicated Linux/Docker host;
they are not a memory reservation on the outer Soda container. The measured
Docker Factory window had one active issue; application and terminal probes
ran in a separate window. These measurements do not qualify their combined
workload or two-issue concurrency for Docker.

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

Install Docker Engine with its Compose plugin, plus `curl`, `jq`, coreutils
and OpenSSL on the host. Run the commands below on that host with access to its
rootful Docker daemon. Check `docker info` for Linux, x86_64, cgroup version 2
and absence of rootless mode, and `docker compose version` for the Compose version.

The selected browser HTTPS/Git ports and Cockpit port 9090 must be available on
the host. Forgejo initially uses loopback ports 3000 and 2222. NetworkManager,
DNS, kernel auditing and sysctl initialization are managed by Docker/the host,
not started inside the runtime.

## Install a released image

The released `install-docker.sh` is the normal installation entry point. Select
an explicitly Docker-qualified release; do not use GitHub's latest ISO pointer.
No qualified installer URL is available yet. This guide records the current
release and qualification boundary for operators and maintainers. The published
handbook describes the released Docker installation path and does not carry
internal availability status.

The version-bound script embeds the exact source revision and `docker.json`
hash/size. It checks the immutable release and its Docker qualification marker,
downloads that version's metadata and Compose file over HTTPS, verifies the
platform and file identities, and pulls the image by digest with a fresh anonymous
Docker client configuration. It checks the image configuration before creating
deployment files or runtime volumes. Download scratch is private, temporary and
restricted to `/home`; an image pull may populate Docker's image cache.

Run the published script through Bash. Its interactive prompts read `/dev/tty`,
so streamed input works. Rootful Docker access requires root, existing Docker
administrator access or terminal-authenticated sudo; creating `/home/sodaos` may
also require sudo. The script explains its privileged host access before starting.
It does not install Docker, change the host firewall/forwarding policy or add client
certificate trust. Missing prerequisites are reported before deployment changes.

The installer selects a private IPv4 browser address assigned to the host and a
non-overlapping RFC1918 Project subnet. It rejects host/Docker route conflicts,
Soda's reserved `10.88.0.0/16` and `10.90.0.0/24`, and occupied required TCP ports
443, 2222, 3000 and 9090. Client routes and firewall access remain operator-owned.
It starts Compose with a bounded readiness wait. Once ready, the interactive flow
guides native Forgejo browser installation and administrator token creation, reads
the bootstrap token without echo, runs `soda-setup` and local-TLS activation, then
exports only the public CA certificate and its fingerprint for client trust.
An empty token leaves browser setup explicitly pending. Existing configuration or
activation is never silently replayed; incomplete activation requires inspection.

Options are `--bind-ip PRIVATE_IPV4`, `--project-subnet CIDR`,
`--install-dir /home/PATH` and `--yes`. `--yes` accepts privileged execution and
installs the runtime without interactive browser enrollment; the final output
states that setup remains pending. The operator can rerun interactively to finish.
`--qualification` is solely for release operators testing the exact immutable
prerelease; it changes the qualification-state gate, not integrity/host checks.

Rerun the same version's command using the same operator and installation directory.
The private `identity.json`, original metadata, Compose and `.env` bind the release,
image, browser address and subnet. Conflicting files, retained containers or volume
ownership refuse installation. Changing releases or adopting an unrecorded
deployment is unsupported. A failed start retains its files and volumes for
diagnosis and a same-version retry; no rollback, volume deletion or pruning occurs.

### Manual verification and deployment details

Choose an exact release from [GitHub releases](https://github.com/LevitateOS/sodaos/releases)
whose notes confirm Docker qualification. It must supply `docker.json`,
`sodaos-docker-compose.yaml`, `install-docker.sh` and the public GHCR image identified in that metadata.
GitHub's latest release is the appliance ISO recommendation; select the Docker
tag explicitly. Production packaging or a prerelease tag alone is not qualification.

Run the host commands as root, or prepare an operator-owned protected directory
with your host's administrative tools. The example uses a fresh `/home/sodaos`;
preserve an existing installation instead of repeating the first-install steps.
Replace the tag placeholder with that selected
qualified release. Installation consumes public release assets and requires no
repository checkout, builder toolchain or registry login.

```sh
set -euo pipefail
mkdir -m 0700 /home/sodaos
cd /home/sodaos
release_tag=REPLACE_WITH_QUALIFIED_DOCKER_TAG
release_url="https://github.com/LevitateOS/sodaos/releases/download/$release_tag"
curl --fail --show-error --location "$release_url/docker.json" -o docker.json
curl --fail --show-error --location "$release_url/sodaos-docker-compose.yaml" -o compose.yaml

jq -e --arg tag "$release_tag" '
  .Format == 1 and .Purpose == "production" and
  (.Distribution == "docker" or .Distribution == "all") and
  .ReleaseTag == $tag and .Architecture == "x86_64" and .Platform == "linux/amd64" and
  (.Image.Manifest | test("^sha256:[0-9a-f]{64}$")) and
  (.Compose.SHA256 | test("^[0-9a-f]{64}$")) and
  (.Compose.Bytes > 0 and .Compose.Bytes == (.Compose.Bytes | floor))
' docker.json > /dev/null
printf '%s  compose.yaml\n' "$(jq -r '.Compose.SHA256' docker.json)" | sha256sum --check
test "$(wc -c < compose.yaml)" -eq "$(jq -r '.Compose.Bytes' docker.json)"
```

Continue only if the downloads and every check succeed. The Compose hash binds
this file to the selected GitHub release. The pull below checks the image against
its registry manifest digest. Retain the downloaded metadata and unedited Compose
file alongside the installation's configuration.

Select a valid, non-overlapping Project subnet before the first start. Save the
image digest and subnet in `.env` so restarts and recreation use the same choices:

```sh
umask 077
printf 'SODA_IMAGE=ghcr.io/levitateos/sodaos@%s\nSODA_PROJECT_SUBNET=10.89.0.0/24\n' \
  "$(jq -r '.Image.Manifest' docker.json)" > .env
docker compose config --quiet
docker compose pull
docker compose up -d --wait --wait-timeout 1800
docker compose ps
```

Edit the subnet in `.env` before running `up` if the default overlaps your routes.
Initial startup downloads six required images by immutable digest, initializes
database credentials and lets Fountain acquire the required Soda package. Allow
several minutes and network access to the selected registries and HTTPS assets.
Exact cached dependencies support offline restarts. Tailnet remains absent until
an administrator enables it through Fountain's extension settings. Configuration is created from canonical defaults only
when absent. Changing the subnet variable later does not rewrite existing
`/etc/soda/host.json` or move retained Projects.

Healthy means the image import, host API socket, PostgreSQL, Forgejo and its
loopback health endpoint are ready for operator setup. It does not indicate
completed browser access or factory enrollment. The health check gives the
first import its own startup window and tests each required unit independently.
If startup fails, inspect selected service diagnostics before retrying:

```sh
docker compose ps
docker compose exec soda systemctl --failed
docker compose exec soda journalctl -b --no-pager -n 100 \
  -u soda-docker-init.service -u soda-image-import.service \
  -u soda-pg-provision.service -u soda-postgres.service -u forgejo.service
```

Review diagnostics privately; they can contain operator configuration. Preserve
the existing volumes while investigating.

## Configure private browser access

Use the existing [operator setup](operator-setup.md#deliberate-setup-with-custom-configuration)
commands inside the runtime. In a terminal on your client, forward Forgejo's
loopback installer through your existing SSH account on the Docker host:

```sh
ssh -N -L 33000:127.0.0.1:3000 OPERATOR@DOCKER_HOST
```

Open `http://localhost:33000`, complete Forgejo's native installation, create its
site administrator and generate that administrator's `read:user` bootstrap
token. On the Docker host, enter the runtime's root terminal:

```sh
cd /home/sodaos
docker compose exec soda bash
```

Inside that terminal, select the Docker host's assigned private address reachable
by your clients; `192.168.50.10` below is an example to replace. Read the token
through hidden input into a temporary mode-0600 file, then run setup:

```sh
set +x
umask 077
bootstrap_file=$(mktemp /run/soda-bootstrap.XXXXXX)
IFS= read -r -s -p 'Forgejo administrator read:user token: ' bootstrap_token
printf '\n'
printf '%s\n' "$bootstrap_token" > "$bootstrap_file"
unset bootstrap_token
soda-setup --forgejo-url https://192.168.50.10 --token-file "$bootstrap_file" && \
  rm -f -- "$bootstrap_file"
```

Successful setup revokes the bootstrap token. If it reports uncertainty or an
existing configuration, inspect the native state using the operator guide before
continuing. After confirmed setup, activate the same private address with local TLS:

```sh
soda-activate --bind-ip 192.168.50.10 --local-tls
exit
```

For an existing certificate/key, use the alternative in the
[activation reference](../../cmd/soda-activate/README.md#run-once).
For Cockpit access, run `docker compose exec soda passwd` to set the runtime root
password; host SSH and runtime passwords are separate. Administration uses
`docker compose exec` rather than SSH into the container.

With local TLS, copy only the public CA certificate onto the Docker host, inspect
its fingerprint there, and transfer that certificate through verified host SSH
to each intended client:

```sh
docker compose cp \
  soda:/var/lib/soda/proxy/caddy/pki/authorities/local/root.crt ./soda-root.crt
openssl x509 -in soda-root.crt -noout -fingerprint -sha256
```

Compare the fingerprint after transfer before adding it to the client's trust
store. Open your configured private HTTPS origin, verify native browser login,
and create or join a Project and execute a command in its browser terminal.
Direct Project SSH/services need their own client subnet routes; [networking](../architecture/networking.md)
owns that configuration. Complete the separate private [factory enrollment](../reference/factory.md)
before admitting agent work. CoreOS disk installation and media continuation are
not part of Docker setup.

## Persistence and maintenance

Compose retains `/etc`, `/var` and `/home` in named volumes. Together they hold
configuration, credentials, database/repository data, Podman storage and writable
Project roots, broker custody, Tailnet state, application data and backups.
`/run` and `/tmp` are ephemeral. Keep the Compose project name stable so subsequent
starts select the same volumes. Run lifecycle commands from the protected
installation directory with the same `compose.yaml` and `.env`.

Store operator-supplied public provider packages and model catalogs under
root-owned `/var/lib/soda/providers/` and configure their exact paths and pins.
Files added to `/usr/local` belong to the outer container layer and disappear
when it is recreated, while retained host configuration still references them.

```sh
cd /home/sodaos
docker compose stop
docker compose up -d --wait --wait-timeout 1800
```

Stopping sends systemd its shutdown signal and allows five minutes for service
shutdown. Docker can report exit code 130 after an orderly halt: Linux reports
a PID-namespace halt as SIGINT ([reboot(2)](https://man7.org/linux/man-pages/man2/reboot.2.html)).
To recreate the outer container with the same pinned image, use
`docker compose up -d --force-recreate --wait --wait-timeout 1800`. Preserve the
three named volumes and the Compose project name `sodaos`.

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
a Docker daemon socket. Soda binaries, Payload v4, native units, defaults
and notices are copied unchanged; CoreOS boot/disk services are excluded.

The attempt produces `image.oci`, `docker.json`, `packages.txt`
and `compose.yaml` under `artifacts/docker/`. The verifier reads the exported
image, compares candidate-owned files and deployment identities with the candidate,
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
tag. The [release operator flow](../development/release.md#guided-release-operator-flow)
owns the guided `start`, `status`, `stage`, `verify-images`, `qualify` and
`promote` commands. Qualification exercises the exact staged candidate through this
installation path on a fresh authorized host before its release can be recommended.
A locally built image is not a published or qualified Docker release.
