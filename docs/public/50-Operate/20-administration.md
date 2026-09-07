# Administration and troubleshooting

Inspect the right service, identity and network boundary before changing a shared Soda appliance.

Use native host root and [Cockpit](../30-Use-Soda/10-cockpit.md) for host work.
Project owners administer inside their project only. Forgejo site administration
and the configured Soda operator are separate from host root.

## Service endpoints

Use the actual installation configuration and displayed connection details:

| Service | Endpoint | Boundary |
| --- | --- | --- |
| Soda dashboard | Configured HTTPS origin on the private proxy | Forgejo-backed Soda session |
| Forgejo browser | Its own configured HTTPS origin | Forgejo identity and permissions |
| Host operator SSH | Approved private appliance endpoint, normally port 22 | Native root/operator key |
| Cockpit | Host loopback 9090 by default, through operator SSH forwarding | Native root-only browser authentication |
| Forgejo Git SSH | Configured private appliance listener, port 2222; copy the clone URL | Personal Forgejo Git key |
| Project SSH/SCP/SFTP | Displayed project IP, port 22 | Joined project-local account and installed public key |
| Project applications | Project IP and published application port, or project loopback with an SSH forward | Private route plus application authentication |

Do not reconstruct an endpoint from the Cockpit URL or an obsolete port. Browser
DNS does not create project DNS. A project bridge IP needs a route from the
actual client; host Tailnet enrollment does not supply it automatically.

Keep public cloud ingress closed. Allow only intended private service traffic
through the relevant provider, host and Tailnet policies. Preserve unrelated
rules rather than disabling the firewall or globally trusting an interface.

## Native service checks

Read state before restarting services:

```sh
rpm-ostree status
systemctl --failed
systemctl status cockpit.socket tailscaled.service forgejo.service
systemctl status soda-dashboard.service soda-proxy.service soda-host.socket
tailscale status
ss -lntp
```

Projects use `soda-project@PROJECT_ID.service`; runners use
`soda-runner@RUNNER_ID.service`. Substitute verified existing IDs. Use Cockpit
Logs or a bounded `journalctl -u UNIT -n 100 --no-pager` for the affected unit.
Logs can contain sensitive data: inspect privately and redact before sharing.
Do not dump full container inspection, process environments or credential files.

The host helper is a fixed-operation Unix-socket service. Keep its socket and
secret permissions intact; do not expose it on a network or give developers the
appliance Podman socket to bypass a failed operation.

## Capacity and persistence

Use native metrics and storage inspection to understand CPU, memory, disk and
I/O pressure. Coordinate shared workloads and runner slots against actual
capacity. Inspect caches and data with their owners before removal; project
writable roots are data, not disposable container cache.

Normal project start retains the existing container and files. Follow
[maintenance](../30-Use-Soda/60-updates-and-fallback.md) and
[backup planning](30-backups-and-restoration.md), not generic prune/recreate advice.

## Troubleshooting

| Symptom | Check first |
| --- | --- |
| Dashboard or Forgejo unavailable | Configured origin/DNS, private bind, TLS, proxy and owning service |
| Browser sign-in denied or expired | Forgejo credential/consent and actual account authority; never borrow operator credentials |
| Cockpit rejects a developer | Expected: project and Forgejo roles do not grant host administration |
| Host SSH works, Cockpit does not | Loopback tunnel, root browser password, certificate and native service/PAM diagnostic |
| Join rejected | Valid profile public key, correct environment, real provisioning result |
| Project SSH times out | Running project, client subnet route, forwarding and allowed private traffic |
| Project SSH key rejected | Displayed login, selected client key, completed join and installed project authorized keys |
| SSH or TLS identity changed | Verify through trusted operator/console access; never bypass checking blindly |
| Git clone denied | Actual clone URL, personal Git key registration, repository permissions and host identity |
| Shared tool missing | Selected mise version, `/opt/mise` installation and native permissions |
| Member cannot administer Podman | Expected: shared-engine control grants project-root power |
| Service unreachable | Readiness, bind/published port, project route and application credentials |
| Tailnet connects but Git address refresh fails | Intended port-2222 listener and separate advertisement result |
| Runner listens without jobs | Native provider registration, labels, workflow and provider result |
| Maintenance loses its response | Actual native deployment/service state before any retry |

After a partial create/join failure, preserve accounts and roots while inspecting
which step completed. An unavailable observation is not proof of absent data.
Do not rerun first-install, bootstrap or project replacement to clear an error.

## Report a useful problem

Record the release/version and architecture, approximate time, non-secret action,
expected result, actual diagnostic and affected service. Distinguish a browser
error, native failure and transport loss. Remove passwords, private keys, tokens,
authentication URLs and unrelated personal/repository data before sharing.
Use the [SodaOS issue tracker](https://github.com/LevitateOS/sodaos/issues) without
publishing private backups or raw environment dumps.
