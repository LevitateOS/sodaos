# Tailscale

Connect the appliance privately and establish approved routes to project environments without changing Soda's identity or authorization model.

Tailscale supplies network connectivity. Forgejo still authenticates browser
users, and ordinary OpenSSH authenticates project access. Soda does not replace
OpenSSH with Tailscale SSH. Tailnet administrators own device approval, access
policy and route approval.

## Initial cloud connection

Use the provider/VM console as the native host operator, not public SSH:

```sh
tailscale up
```

Open the authentication URL in your own browser and sign in to the intended
Tailnet. Keep that URL out of shared logs/screenshots. Complete device approval
in [Tailscale administration](https://login.tailscale.com/admin/machines) when
required. No developer account or project membership is created by enrollment.

Inspect the native state and private address:

```sh
tailscale status
tailscale ip -4
```

Join the permitted client to the Tailnet, then use the private operator SSH route
and [Cockpit tunnel](10-cockpit.md). Finish
[operator setup](../20-Deploy/25-operator-setup.md) for browser endpoints. Keep
provider public ingress closed and the console available.

## Use the Cockpit page

For an already reachable appliance, sign into operator Cockpit and open
**Tailscale**. Select **Sign in**, follow the native browser authentication and
check the resulting device identity and addresses. Native status is observed
while the page is active; leaving the page does not log the host out.

Visible peers and a connected device are not proof that a policy permits a
particular service. Use the advertised name when the client's DNS supports it,
or the actual Tailnet IP. Project labels are not Tailnet DNS names.

## Route the project subnet

The host's Tailnet IP and the project bridge subnet are separate networks.
Host enrollment alone does **not** make project IPs reachable.

The operator must:

1. Select the actual non-overlapping project subnet configured at installation.
2. Follow [Tailscale subnet-router setup](https://tailscale.com/kb/1019/subnets)
   for native forwarding and advertisement on the appliance.
3. Obtain the Tailnet administrator's route approval and appropriate access rules.
4. Enable route acceptance on clients where their platform requires it.
5. Check direct SSH to the displayed project IP and intended service ports from
   an actual developer client.

Preserve existing routing/firewall choices and restrict forwarding to intended
private traffic. Do not advertise a guessed example subnet or overwrite other
advertised routes. An alternative on a trusted LAN is a router route for the
project subnet via the appliance; do not install conflicting routes blindly.

## Forgejo address refresh

The page can request a Forgejo Git SSH advertisement refresh after observing a
connected host. The native Git listener must already accept the selected private
address on port 2222. A LAN-only listener is not made Tailnet-accessible by
changing its advertised address.

Unchanged advertisement needs no restart. A change can restart Forgejo, so
coordinate it with active users. Enrollment success and refresh failure are
separate results; fix the listener/configuration problem before using the page's
refresh retry. Browser/OAuth origins remain configured separately and are not
rewritten from the Tailnet name.

## Optional exit nodes

An exit node routes Internet-bound traffic through a device. It is not needed
for ordinary Tailnet access and does not replace a project-subnet route.
The page supports native exit-node selection, advertisement and the preference
for allowing local-network access while using an exit node.

Read [Tailscale's exit-node guidance](https://tailscale.com/kb/1103/exit-nodes)
before changing a shared server's routing. Advertisement requires approval;
neither advertisement nor approval alone proves routed traffic works.

## Diagnose or leave the Tailnet

Read native diagnostics and `tailscale status` before re-enrolling. Use
[Tailscale's CLI reference](https://tailscale.com/kb/1080/cli) for native actions.
Keep console access for logout or route changes that may cut off your session.

A native `tailscale logout` removes private connectivity until reauthentication;
review the device record and policy separately. Cockpit logout does not log out
Tailscale. None of these network actions revokes a Forgejo account, Git token or
project SSH key by itself.
