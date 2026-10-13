# Operator setup

Continue from the installed appliance to trusted private browser access using the guided setup console.

Host administration, native Forgejo accounts, Project membership and factory
execution have separate authority. Keep the root password for the operator;
create normal Forgejo accounts for developers. Browser setup does not enroll a
provider account or authorize factory work.

## Connect to the installed appliance

Keep the host on its private network and retain local or VM console access.
The guided disk installation enables root password SSH. From your client:

```sh
ssh root@APPLIANCE_ADDRESS
```

Use the actual private address displayed by the appliance and verify its SSH
host fingerprint independently before entering the installation password.
An installed operator key can be used instead; key enrollment is optional.

Run the guided setup in a root SSH terminal or local console:

```sh
/usr/libexec/soda/soda-install configure
```

Choose the listed private appliance address your client can reach. Use a stable
address or DHCP reservation: changing it later requires explicit configuration
maintenance. No domain or separately supplied certificate is needed for this
local HTTPS path.

## Complete native Forgejo setup

The console gives you an SSH tunnel command. From your client, using the selected
private appliance address:

```sh
ssh -N -L 33000:127.0.0.1:3000 root@APPLIANCE_ADDRESS
```

Open `http://localhost:33000` on that client. Complete Forgejo's native installation
and create its administrator. Keep its localhost browser URL during bootstrap;
the guided activation sets the final private HTTPS origin and Git SSH port 2222.
The unfinished installer remains behind the tunnel, rather than a public proxy.

In Forgejo **Settings > Applications**, create an administrator-owned token with
`read:user`. No admin or repository token scope is needed for browser setup.
Return to the root setup terminal, type `CONFIGURE SODA` when ready and enter the
token at the hidden prompt. Do not paste it into a command, issue or shared log.
The console supplies it privately to setup; you do not create a token file by hand.

Setup reads `/api/v1/user`, records the native operator identity and writes the
browser configuration. It creates no OAuth application and does not provision
the identity broker or its encryption key. After durable configuration it revokes
the bootstrap token server-side. Factory enrollment is a separate operator action.

## Trust and open the private browser address

Guided activation uses the selected private IP and Caddy's local HTTPS certificate.
Wait for the console to confirm the browser services and display the public CA's
SHA-256 certificate fingerprint and copy command.

Copy only the public root certificate over the verified SSH connection. Compare
its certificate fingerprint with the console, then install it through your
client/browser's normal certificate trust settings. Never copy the CA private key
or disable TLS checking to bypass a warning.

Open the exact displayed HTTPS address and sign in to Forgejo. A reported active
service or configured URL is not a successful client login; verify it from the
intended client. Cockpit remains separate host administration with root-only login.

## Check the first Project

Create or choose a repository, open Sodaspaces, explicitly create its development
environment as its owner, then Join as a code-write-authorized member. The creator
must Join too. Open a browser terminal and verify its Project login/home.

An external development SSH key is optional for browser terminals. If you need
direct Project SSH or service access, choose a non-overlapping private Project
subnet and establish its client route through your LAN router or an approved
[Tailscale subnet route](../30-Use-Soda/40-tailscale.md#route-the-project-subnet).
Soda reserves `10.88.0.0/16` and `10.90.0.0/24`; check the selected Project subnet
against other LAN/VPN routes. A displayed Project address proves no client route.

Use [People and access](../50-Operate/10-people-and-access.md) to onboard developers,
then have one complete [First connection](30-first-connection.md) with their own
account. Share only the endpoints/public identity details each person needs and
retain protected operator records and a backup procedure.

## Optional operator key

To import a laptop public key later, run `soda-install enroll-key` from the native
root console. Choose the reachable private IPv4 address, verify its displayed
host fingerprint and explicitly arm the temporary import window. It accepts one
public key through password authentication on port 22222 for at most five minutes.
Keep the private key on your client. After verifying key access, decide separately
whether to disable host password logins.

## Preserve existing setup

Rerunning configure on an activated appliance gives read-only access guidance
instead of another bootstrap. If setup or activation was not confirmed, inspect
the retained configuration and service state before retrying; do not delete its
markers, databases, encryption keys or Project roots as repair.

Installation and first setup are not upgrade commands. Follow
[maintenance](../30-Use-Soda/60-updates-and-fallback.md) for existing installations.
Configure provider connections, native factory actors and bounded execution
through the separate [factory walkthrough](../30-Use-Soda/15-software-factory.md)
after browser access is established. A login, repository or Project Join does not
grant factory execution authority.
