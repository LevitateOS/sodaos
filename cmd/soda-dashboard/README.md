# soda-dashboard

Run Soda's backend for native Forgejo extension pages, project operations and
factory coordination. This is a service process for appliance operators;
people use its features through Forgejo's installed Soda extension.

## Installed use

The [dashboard container recipe](../../system/host/services/soda-dashboard.container)
runs `/usr/local/bin/soda-dashboard` inside the dashboard image as UID/GID 2000.
It starts after activation and receives private configuration, database access
and host/identity sockets through the appliance's mounts.

Inspect an installed service with:

```sh
systemctl status soda-dashboard.service
journalctl -u soda-dashboard.service
```

Follow [Operator setup](../../docs/guides/operator-setup.md) and
[Installation](../../docs/guides/installation.md) for provisioning and activation.

## Process inputs

| Flag | Meaning |
| --- | --- |
| `--config PATH` | Dashboard JSON; default `/etc/soda/dashboard.json`. |
| `--extension-socket PATH` | Optional absolute Unix socket for the native extension bridge. |
| `--operator-socket PATH` | Optional absolute Unix socket for factory operator commands. |
| `--operator-uid UID` | Required admitted OS peer UID when the operator socket is configured. |

The installed container supplies `/ipc/soda.sock` for the extension listener and
`/run/soda/operator/operator.sock` with peer UID `0` for operator commands.
`--help` prints flags before opening service resources. Configuration fields
belong to [Service configuration](../../docs/reference/configuration.md).

## Prerequisites and effects

Startup validates embedded avatars, loads configuration and restricted secret
files, opens the encrypted PostgreSQL store, and takes exclusive factory-ledger
ownership. It reconciles outstanding work before serving. Selected background
Forgejo integrations require their configured private transport and credentials.

The HTTP listener uses the configured `listen` address. It serves backend probes,
public avatars and the configured intake handler; protected product operations
use the dedicated extension listener and verified native authority. Factory
operator commands use the separate socket and OS peer identity.

Requests can write database state and invoke host, broker and native Forgejo
operations. On SIGINT/SIGTERM the process closes admission, drains HTTP, closes
terminal peers and releases coordinator ownership. Failures are logged and exit
`1`. Route and authority details belong to the
[HTTP API](../../docs/reference/api.md), [Factory](../../docs/reference/factory.md)
and [Trust](../../docs/architecture/trust.md) guides.
