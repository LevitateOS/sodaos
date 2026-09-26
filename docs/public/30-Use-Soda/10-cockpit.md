# Cockpit: host administration

Use Soda-branded stock Cockpit to administer the host. [Tailnet](40-tailscale.md)
and [Runners](50-ci-runners.md) belong to the native Forgejo dashboard, not custom
Cockpit pages.

## Connect as the operator

Cockpit listens on all interfaces and permits the native root/operator account. It does
not use Forgejo login. Project owners, project members and Forgejo site
administrators do not gain access merely through those roles.

From the operator client, open `https://APPLIANCE_ADDRESS:9090` and sign in as
root with the host credential. A loopback SSH tunnel remains available instead:

```sh
ssh -N -L 127.0.0.1:9090:127.0.0.1:9090 root@APPLIANCE_ADDRESS
```

Then open `https://127.0.0.1:9090` locally. A personal SSH key authenticates the tunnel; it is not Cockpit's browser password.
Compare the certificate with the intended server through trusted operator access.
A loopback tunnel can require handling a certificate-name mismatch deliberately;
do not disable TLS checking globally or accept an unexplained certificate change.
See [Cockpit HTTPS configuration](https://cockpit-project.org/guide/latest/https.html)
for installing the appropriate trusted certificate.

Use another private Cockpit endpoint only if the operator deliberately configured
it. Host Tailnet enrollment does not automatically expose loopback port 9090.
Do not open public administration ports as a connection shortcut.

## Choose the right page

| Page | Use it for |
| --- | --- |
| Overview and Metrics | Host identity, load and capacity investigation |
| Services and Logs | Native service state and journal diagnostics |
| Storage | Disks, filesystems, mounts and free space |
| Networking | Native interfaces and available firewall controls |
| Podman containers | Host containers, images, volumes and native container diagnostics |
| File browser | Host file inspection, transfer and deliberate maintenance |
| SELinux | Policy status and access-denial investigation |
| Diagnostic reports | Manually collect support information; review private contents before sharing |
| Accounts | Native host Linux accounts, passwords and keys |
| Software updates | Native CoreOS/rpm-ostree deployments and updates |
| Terminal | Privileged host administration as the operator |

The operator terminal is a host shell. It is not the developer's browser workspace
terminal. Development accounts, repositories and environment joins belong in
[the Soda dashboard](05-dashboard.md); there are no Cockpit Projects, People or
Soda Updates workflows to use for those tasks. Accounts manages the host, not
Forgejo identities or accounts inside projects. Podman on the host does not
automatically manage the separate engines inside projects. Native root remains
the only eligible Cockpit account, regardless of which navigation entries are visible.

## Operate deliberately

Inspect service state and diagnostics before restarting anything. A root action
can affect every developer and runner. Coordinate downtime, preserve data and
review the selected target before stopping a project or changing networking.

Follow [Cockpit's native guide](https://cockpit-project.org/guide/latest/) for
stock host-management controls. [Administration](../50-Operate/20-administration.md)
identifies Soda's services and troubleshooting boundaries.

## Reconnect and sign out

A page refresh observes state; it does not retry a failed mutation. After a lost
connection, inspect native state before repeating the action. Sign out after
operator work. Cockpit logout does not disconnect Tailscale, stop projects or
revoke developers' Forgejo/SSH access.
