# soda-dashboard — Soda Forge backend

Run Project administration, human terminals, account controls, private
applications and its private native API. Factory coordination runs in
[`soda-factory-server`](../soda-factory-server/README.md), a separate product and
process. This command never starts a coordinator or worker MCP listener.

Build from the repository root:

```sh
go build -o .artifacts/bin/soda-dashboard ./cmd/soda-dashboard
```

| Flag | Meaning |
| --- | --- |
| `--config PATH` | Configuration; default `/etc/soda/dashboard.json`. |
| `--listen IP:PORT` | Override the configured loopback HTTP listener. |
| `--extension-socket PATH` | Private native-extension Unix listener. |
| `--operator-socket PATH` | Private Project and extension administration listener. |
| `--operator-uid UID` | Required admitted OS peer UID for operator commands. |
| `--factory-control-socket PATH` | Factory's private Project coordination endpoint. |

The Factory control socket may instead be supplied in JSON. With it omitted,
Forge runs without Factory. A Project with retained Factory policy or work still
requires Factory coordination for lifecycle changes; removing configuration does
not bypass its recorded authority.

The installed `soda-dashboard.service` runs as UID/GID 2000 and exposes health on
loopback port 8080. Its operator socket remains
`/run/soda/operator/operator.sock`; Factory commands use Factory's separate socket.
The Fountain extension sends Factory browser calls directly to Factory's own
extension socket. The public intake route also goes directly to Factory.
Native browser URLs and authority checks are preserved.

SIGINT/SIGTERM drains Forge requests and closes its human terminal attachments.
It does not stop Factory. Inspect with `systemctl status soda-dashboard.service`
and `journalctl -u soda-dashboard.service`.

See [configuration](../../docs/reference/configuration.md) and
[architecture](../../docs/architecture/overview.md).
