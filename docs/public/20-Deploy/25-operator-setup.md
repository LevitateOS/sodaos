# Operator setup

Prepare the appliance, Forgejo and private access, then configure the agent profile and authority for approved software work.

Host administration, Forgejo collaboration, factory execution and persistent human
projects have separate authority. The [factory walkthrough](../30-Use-Soda/15-software-factory.md)
describes explicit task admission after the appliance is configured.

## Keep four authorities separate

Native host root administers the appliance and Cockpit. Forgejo has its own
administrator account. Soda records a configured operator by their stable
Forgejo identity. Project owners administer only their own environments.
Do not create developer host accounts or reuse root's password as a team login.

Complete the selected release's host/component installation first. Its initial
Forgejo listener is loopback-only; Cockpit listens on all interfaces with
root-only authentication from the start, so keep the host on its private
network. Keep native console access and use the operator's verified SSH key;
keep bootstrap access private. If no operator key is installed yet, the
[optional key enrollment](../../operator/enroll-key.md) imports one laptop key
through a short password-only window.

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
`read:user`, belonging to a Forgejo site administrator. Setup only reads
`/api/v1/user` to confirm that administrator identity.
Setup does not need admin or repository token scopes. Store it through a
private input channel in a mode-0600 file on the appliance, not argv or a shared
terminal transcript. Do not lend this server credential to developers.

On the host, replace the example Forgejo origin and token-file path:

```sh
/usr/bin/soda-setup \
  --forgejo-url https://git.example.test \
  --token-file /root/private/forgejo-token
```

Setup records the operator identity and creates the grant-encryption key.
It creates no OAuth application and retains no OAuth secret; it keeps no
bootstrap-token copy/reference and leaves the supplied token file unchanged.
After durable configuration, setup revokes the bootstrap token server-side
so the token itself is unusable; failures before that point keep it usable
for retry. Other existing copies need
separately authorized maintenance;
setup does not delete or revoke them.
Soda's browser pages and product operations use Forgejo's native extension service
on the configured HTTPS origin. The public `/-/soda/` routes are limited to the
broker identity callback and avatar provider. The origin must resolve to the
approved endpoint and be covered by a trusted certificate. Setup refuses to overwrite existing configuration. After
an uncertain failure, inspect Forgejo's applications and Soda's existing state
before retrying; do not reset its databases.

## Activate trusted private browser access

Supply a valid certificate/key covering the configured Forgejo/Sodaspaces origin. Use a
restricted private input directory. Select the private appliance IP deliberately:

```sh
/usr/bin/soda-activate \
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

## Configure bounded agent execution

After private Forgejo access is established, supervise bounded agent execution
through the [factory operator reference](../../reference/factory.md): the
`soda-factory` operator command inspects recorded runs, retires one run and
settles outstanding work, as shown in the
[first-task walkthrough](../30-Use-Soda/15-software-factory.md). Automatic
intake remains unavailable: a project join, issue label or repository creation
does not admit work. Configure CPU, memory, process and writable-storage limits
against available capacity, and select permitted network destinations deliberately.

Keep the dedicated provider credential home outside human project roots. Protect
the factory ledger and its workspace/publication directories. Verify the first task,
its exact-commit CI and review, and recorded cleanup before routine operation.
