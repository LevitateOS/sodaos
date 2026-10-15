# Run Soda OS with Docker

Run the complete Soda runtime on a dedicated x86-64 Linux Docker Engine host, with persistent Projects and host-managed networking.

**Docker support is in preparation. No qualified Docker release is currently available.**
The existing v0.1.0 and v0.1.1 releases provide installation media, not the
Docker metadata, Compose asset and public image required below. Do not run these
installation commands until a release explicitly confirms Docker qualification
and publishes all three. For installation media, use
[Install on premises](20-install-on-premises.md) or
[Deploy in a VM or cloud](10-deploy-to-cloud.md).

Docker runs the complete Soda application runtime directly on the host kernel.
The runtime includes the native host helper, identity broker, systemd, Podman,
Forgejo, PostgreSQL, Caddy and the release's bundled Project images. Podman remains
the internal Project and application engine. Docker owns the outer container's
lifecycle; the operator owns the host operating system.

## Host requirements

Use a dedicated x86_64 Linux Docker Engine host with cgroup v2 and Compose 2.17 or
later, including the [bounded startup wait](https://github.com/docker/compose/blob/v2.17.0/cmd/compose/up.go).
Rootless Docker and Docker Desktop are outside this deployment contract.
Allow capacity for the runtime, retained Projects, nested workloads and backups.
The complete payload occupies several gigabytes before Project data is created.

The runtime needs privileged execution for nested containers and systemd, and
host networking for private browser endpoints, Project bridges and Tailnet.
It has host administrative authority: Docker's privileged mode grants all
capabilities and host-device access ([Docker documentation](https://docs.docker.com/engine/containers/run/#runtime-privilege-and-linux-capabilities)).
The appliance's SELinux dashboard confinement is not provided by this Docker
deployment. Use a dedicated host and restrict operator access accordingly.
The Docker socket and host root filesystem are not mounted into the runtime.

The host must provide IPv4 forwarding for Project routing. Enable it through the
host's normal network configuration. Select a private browser address and a
Project subnet that do not conflict with the host's LAN, Docker or VPN networks.
Configure the [client routes](#connect-clients-and-projects) separately from browser access.

Install Docker Engine with its Compose plugin, plus `curl`, `jq`, coreutils
and OpenSSL on the host. Run the commands below on that host with access to its
rootful Docker daemon. Check `docker info` for Linux, x86_64, cgroup version 2
and absence of rootless mode, and `docker compose version` for the Compose version.
Soda currently targets x86-64. Apple Silicon/ARM64 and UTM support are tracked in
[the platform backlog](https://github.com/LevitateOS/sodaos/issues/45); this guide
does not supply an ARM64 image.

The selected browser HTTPS/Git ports and Cockpit port 9090 must be available on
the host. Forgejo initially uses loopback ports 3000 and 2222. NetworkManager,
DNS, kernel auditing and sysctl initialization are managed by Docker/the host,
not started inside the runtime.

## Install a qualified released image

**The following is the installation procedure for a future qualified release;
it is not a current download.** Leave it unexecuted while the release assets or
public image are missing. A registry error is not a reason to substitute an
unqualified image or infer an unpublished tag.

Choose an exact release from [GitHub releases](https://github.com/LevitateOS/sodaos/releases)
whose notes confirm Docker qualification. It must supply `docker.json`,
`sodaos-docker-compose.yaml` and the public GHCR image identified in that metadata.
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
release_tag=REPLACE_WITH_QUALIFIED_DOCKER_TAG
if [ "$release_tag" = REPLACE_WITH_QUALIFIED_DOCKER_TAG ]; then
  printf '%s\n' 'Stop: select a published, qualified Docker release first.' >&2
  exit 1
fi
mkdir -m 0700 /home/sodaos
cd /home/sodaos
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

Select a valid, non-overlapping Project subnet before the first start. Replace
`10.89.0.0/24` in the command below if it overlaps your LAN, Docker or VPN routes.
Save the image digest and selected subnet in `.env` so restarts and recreation
use the same choices:

```sh
umask 077
printf 'SODA_IMAGE=ghcr.io/levitateos/sodaos@%s\nSODA_PROJECT_SUBNET=10.89.0.0/24\n' \
  "$(jq -r '.Image.Manifest' docker.json)" > .env
docker compose config --quiet
docker compose pull
docker compose up -d --wait --wait-timeout 1800
docker compose ps
```

Initial startup imports the bundled images and initializes database credentials;
allow several minutes. Configuration is created from canonical defaults only
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

Continue only after the qualified runtime is healthy. Docker uses the explicit
setup commands below inside the runtime; the appliance's guided disk-installation
console is not part of this path. In a terminal on your client, forward Forgejo's
loopback installer through your existing SSH account on the Docker host. Verify
that host's SSH fingerprint independently before authenticating:

```sh
ssh -N -L 33000:127.0.0.1:3000 OPERATOR@DOCKER_HOST
```

Open `http://localhost:33000`, complete Forgejo's native installation and create
its site administrator. In Forgejo **Settings > Applications**, generate that
administrator's `read:user` bootstrap token; no repository or admin scope is
needed for browser setup. On the Docker host, enter the runtime's root terminal:

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
existing configuration, inspect the retained `/etc/soda` configuration and
selected service journals before continuing. Setup refuses to overwrite existing
configuration; do not delete its state to force another bootstrap. After
confirmed setup, activate the same private address with local TLS:

```sh
soda-activate --bind-ip 192.168.50.10 --local-tls
exit
```

Alternatively, before activation, use an existing PEM certificate covering the
same browser origin and its private key in restricted files inside the runtime.
Choose this command instead of `--local-tls`:

```sh
soda-activate --bind-ip 192.168.50.10 \
  --certificate /home/operator/tls/cert.pem \
  --private-key /home/operator/tls/key.pem
exit
```

Activation records its state and refuses reactivation. If it fails after writing
configuration, inspect the services and retained activation state before deciding
how to repair it.

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

Compare the fingerprint after transfer before adding it through the client's
normal certificate trust settings. Never transfer the CA private key or disable
TLS checking. Open your configured private HTTPS origin, verify native browser
login, explicitly create or Join a Project and execute a command in its browser
terminal. The creator must Join too. Follow [First connection](30-first-connection.md)
for developer access, then complete the separate private
[factory enrollment](../30-Use-Soda/15-software-factory.md) before admitting agent work.
Browser setup does not authorize factory execution.

## Connect clients and Projects

Keep Soda, Forgejo, Cockpit and Project services on intended private networks.
Host networking exposes the runtime's listeners directly through the Docker
host's network; preserve host firewall and provider policy. Cockpit listens on
port 9090 and requires root/operator login. Host SSH, the private browser origin,
Forgejo Git SSH on the selected address at port 2222, and Project IPs are separate
endpoints.

The Docker host must forward IPv4 traffic. Select a Project subnet separate from
the LAN, Docker bridges and VPN routes; Soda also reserves `10.88.0.0/16` and
`10.90.0.0/24`. For direct Project SSH or services, configure your LAN router's
route to the actual Project subnet through the Docker host's reachable private
address, with deliberate private forwarding/firewall policy. A Project IP shown
in the browser does not prove a client route.

An approved [Tailscale subnet route](../30-Use-Soda/40-tailscale.md#route-the-project-subnet)
is an alternative. When following native Tailscale commands for this deployment,
run them inside the runtime with `docker compose exec soda tailscale …`; host
forwarding remains the Docker host operator's responsibility. Tailnet enrollment,
route advertisement, administrator approval, client route acceptance and service
access rules are separate checks. Preserve other advertised routes.

Verify direct SSH and required service ports from an intended client using its
actual Project account, IP and trusted host-key fingerprint. Browser terminals
use the private browser origin and need no direct client Project-subnet route.
Host or Tailnet access grants no Project membership or factory authority.

## Persistence and maintenance

Compose retains `/etc`, `/var` and `/home` in named volumes. Together they hold
configuration, credentials, database/repository data, Podman storage and writable
Project roots, broker custody, Tailnet state, application data and backups.
`/run` and `/tmp` are ephemeral. Keep the Compose project name stable so subsequent
starts select the same volumes. Run lifecycle commands from the protected
installation directory with the same `compose.yaml` and `.env`.

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
three named volumes and the Compose project name `sodaos`. With that default
project name, the volumes are `sodaos_config`, `sodaos_state` and `sodaos_home`;
verify the actual volume names before a backup or restore.

Removing the outer container preserves named volumes; `down --volumes`
deletes them and is destructive. Do not prune Podman storage to repair a Project.

The installed PostgreSQL timer takes nightly native backups, retaining seven
runs. For an explicit database backup on the Docker host:

```sh
cd /home/sodaos
docker compose exec soda /usr/bin/soda-pg-backup
```

The successful command names a directory under
`/var/lib/soda/backups/postgres`, containing `globals.sql`, `forgejo.dump` and
`soda.dump`. Each database dump has its own consistent snapshot; these files
alone are not a complete deployment backup. Copy required backups to protected,
independent storage rather than relying only on a backup inside a named volume.

For a consistent complete volume backup, stop factory admission, let active work
finish, quiesce writers and stop the runtime. Back up all three named volumes as
one set with the host's storage/backup tools. Include the Compose file, `.env`,
release metadata and pinned image identity. Preserve numeric ownership, modes,
links, ACLs and extended attributes. These volumes include retained Project roots,
repositories, configuration and broker key/custody, not just database files.
Treat the backup as secret material.

Restore all three volumes together with the same Compose project name and pinned
runtime image before starting services. Test the restore in isolation to avoid
duplicate machine, Tailnet or provider identities. Follow
[Backups and restoration](../50-Operate/30-backups-and-restoration.md) for the data
inventory and client-level recovery checks. Never copy live PostgreSQL data files
as an application backup or mix unrelated backup generations.

The host OS is maintained outside Soda. CoreOS disk installation, rpm-ostree,
Zincati and the appliance's explicit release-maintenance command do not manage
this Docker deployment. Container recreation with the same image preserves state;
changing image versions requires a separately verified retained-state transition.
Select a new image only when its release explicitly qualifies that transition
from your installed version. Preserve the matching metadata, Compose
asset, image identity and complete backup; a digest-pinned pull alone does not
establish upgrade compatibility. The appliance
[update procedure](../30-Use-Soda/60-updates-and-fallback.md) does not update this
Docker deployment.
