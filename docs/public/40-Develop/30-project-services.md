# Project services

Run ordinary Podman workloads and native development processes inside a project, and share their endpoints without sharing engine administration.

## Who controls the runtime

The project has its own nested Podman engine and storage. The repository owner
administers shared workloads through the project-local rootful engine. Ordinary
members can use services but cannot administer that engine; access to its socket
would grant project-root power.

The project profile connects native Podman commands to
`/run/soda-podman/podman.sock`. This is not the appliance's Podman socket. Do not
replace it with an unrestricted host socket or disable permissions to let a
member manage containers. A member can still run normal user-owned development
processes within their Linux permissions.

## Build and run a repository's workload

As the project administrator, inside the project, review the repository's
Containerfile/Dockerfile, Compose definition, mounts, ports and secrets before
execution. Use its documented native commands, for example:

```sh
cd ~/repo-name
podman compose up -d --build
podman compose ps
```

`podman compose` invokes the installed Compose provider. Follow
[Podman's documentation](https://docs.podman.io/en/latest/markdown/podman-compose.1.html)
and the repository's instructions rather than inventing a Soda service format.
Image recipes define builds; Compose defines running services and storage.

Detached startup is not readiness. Inspect native service status/logs, then
verify its actual HTTP or database protocol. If startup loses its response,
inspect before repeating a build or creation action. Preserve failed containers
and data while diagnosing them.

## Secrets and data

Keep password files and tokens out of Git, shared directories, command arguments
and expanded configuration logs. Prefer native file/secret inputs supported by
the actual image and Compose provider. For an existing restricted password file,
an administrator can create a native Podman secret:

```sh
podman secret create project-db-password /absolute/private/password-file
```

The workload definition must reference that external secret with permissions
and ownership appropriate for its image. For PostgreSQL images supporting
`POSTGRES_PASSWORD_FILE`, point it at the mounted secret rather than putting
the password into the environment. Follow the selected image's documentation;
secret creation alone does not configure a database or rotate an existing one.

Use named volumes for database data and reviewed bind mounts for source files.
Both live inside the project's persistence boundary. A checkout of Compose files
is not a backup of a database volume. Use the database's own consistent backup
procedure; PostgreSQL documents [backup and restore](https://www.postgresql.org/docs/current/backup.html).

## Reach a service

Agree on ports within the project. Different projects have different IP/network
namespaces, so the same port can serve different projects without being the same
endpoint. A workload-container address is not necessarily the published project
endpoint: publish the selected port on the intended project interface.

For direct team access, use `PROJECT_IP:PORT` through the approved private route.
Check the listener's bind address, published port, firewall and application
credentials. Do not expose all development ports publicly to solve one route.

For a service bound only to project loopback, use a local SSH forward:

```sh
ssh -N -L 127.0.0.1:8080:127.0.0.1:8080 PROJECT_USER@PROJECT_IP
```

Open `http://127.0.0.1:8080` on that client. Substitute actual free ports; explicit
client loopback binding avoids offering the forward to your local network.
Members can consume HTTP/PostgreSQL normally without engine access.

## Stop and restart without replacing data

Coordinate with service users before stopping workloads. After project or host
maintenance, inspect the existing workloads and use their native start operation
when needed, such as `podman compose start` for an existing Compose application.
Do not assume automatic workload restart.

Do not use `down -v`, container deletion, project replacement or pruning as a
routine restart or readiness fix. These can destroy data. A Git merge does not
apply migrations, update running services or clean up volumes. Use deliberate
native operations and [protect data first](../50-Operate/40-data-safety-and-removal.md).
