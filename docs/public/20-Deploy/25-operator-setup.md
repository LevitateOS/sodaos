# Operator setup

Establish Forgejo-backed identity, trusted private browser endpoints and project networking before inviting your team.

## Keep four authorities separate

Native host root administers the appliance and Cockpit. Forgejo has its own
administrator account. Soda records a configured operator by their stable
Forgejo identity. Project owners administer only their own environments.
Do not create developer host accounts or reuse root's password as a team login.

Complete the selected release's host/component installation first. Its initial
Forgejo and Cockpit listeners are loopback-only. Keep native console access and
use the operator's verified SSH key; do not expose the unfinished installer.

## Configure Forgejo privately

From the operator's client, open a loopback-bound tunnel:

```sh
ssh -N -L 127.0.0.1:3000:127.0.0.1:3000 root@APPLIANCE_ADDRESS
```

Replace `APPLIANCE_ADDRESS` with the approved private host endpoint and verify
its SSH host key independently. Open `http://127.0.0.1:3000` locally to complete
Forgejo's native installation and establish its administrator. That HTTP path
stays inside your SSH tunnel; it is not the team's published browser origin.

Configure the intended Forgejo HTTPS origin and Git SSH port **2222**. Preserve
Forgejo's own persistent data and follow its
[administration documentation](https://forgejo.org/docs/latest/admin/).

## Configure Soda's identity integration

Using Forgejo's native settings, create the operator token required for setup:
`read:user`, `write:user`, `write:admin` and `read:repository`. Store it through a
private input channel in a mode-0600 file on the appliance, not argv or a shared
terminal transcript. Do not lend this server credential to developers.

On the host, replace the example Forgejo origin and token-file path:

```sh
/usr/local/sbin/soda-setup \
  --forgejo-url https://git.example.test \
  --token-file /root/private/forgejo-token
```

Setup creates the actual OAuth application and records the operator identity.
Current source shares Forgejo's HTTPS origin, with Soda API/OAuth at `/-/soda/`;
the drawer is not implemented yet. That origin must resolve to the approved
endpoint and be covered by a trusted certificate. Setup refuses to overwrite existing configuration. After
an uncertain failure, inspect Forgejo's applications and Soda's existing state
before retrying; do not reset its databases.

## Activate trusted private browser access

Supply a valid certificate/key covering the configured Forgejo/Sodaspaces origin. Use a
restricted private input directory. Select the private appliance IP deliberately:

```sh
/usr/local/sbin/soda-activate \
  --bind-ip PRIVATE_APPLIANCE_IP \
  --certificate /root/private/browser-cert.pem \
  --private-key /root/private/browser-key.pem
```

Activation binds the proxy and Forgejo Git SSH to that private IP and starts the
services with their required permissions. The dashboard process is unprivileged.
Never substitute a public address or disable TLS checks to complete activation.

For Tailnet Git access, enroll the host first and choose the intended Tailnet
listener address. Later advertisement refresh is not a substitute for a reachable
listener. Browser/OAuth origins remain distinct configured values.

## Route projects to developers

Choose a private project subnet that overlaps neither the host LAN nor other
client/VPN routes. Project addresses live on the appliance's routed bridge.
Establish either a LAN-router route via the appliance or a native Tailscale subnet
route with the required administrator approval and access rules.

See [Tailscale subnet routing](../30-Use-Soda/40-tailscale.md#route-the-project-subnet).
Verify from the intended developer client, not only from the host. A dashboard
page displaying a bridge IP is not a connectivity check. Limit forwarding and
service access to the approved private networks.

## Check and hand over

1. Verify trusted Soda and Forgejo HTTPS origins from the client, and sign in as
   the configured operator through Forgejo.
2. Verify separate [root-only Cockpit access](../30-Use-Soda/10-cockpit.md).
3. Use **People** to [add a developer](../50-Operate/10-people-and-access.md).
4. Have that person complete login, development-key registration and an explicit
   join, then verify project SSH and Git independently.
5. Record private endpoints, trust material and the backup procedure in protected
   operator records. Share only the URLs/public identity details each user needs.

Setup and first installation are not normal upgrade commands. Preserve existing
configuration, grant-encryption key, databases and project roots during
[maintenance](../30-Use-Soda/60-updates-and-fallback.md).
