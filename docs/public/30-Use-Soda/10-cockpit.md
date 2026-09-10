# Cockpit: host administration

Use stock Cockpit and its Tailscale and Runners extensions to operate the appliance, separately from the developer dashboard.

## Connect as the operator

Cockpit is loopback-first and permits the native root/operator account. It does
not use Forgejo login. Project owners, project members and Forgejo site
administrators do not gain access merely through those roles.

From the operator client, use a verified private SSH route and a local tunnel:

```sh
ssh -N -L 127.0.0.1:9090:127.0.0.1:9090 root@APPLIANCE_ADDRESS
```

Open `https://127.0.0.1:9090` locally and sign in as root with the host credential.
A personal SSH key authenticates the tunnel; it is not Cockpit's browser password.
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
| Terminal | Privileged host administration as the operator |
| [Tailscale](40-tailscale.md) | Native Tailnet sign-in, device status and routing preferences |
| [Runners](50-ci-runners.md) | Local Forgejo runner registration and service capacity |

The operator terminal is a host shell. It is not the developer's browser workspace
terminal. Development accounts, repositories and environment joins belong in
[the Soda dashboard](05-dashboard.md); there are no Cockpit Projects, People or
Soda Updates workflows to use for those tasks. The stock Accounts navigation is
hidden; this does not remove native account tools or turn visibility into an
authorization boundary.

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
