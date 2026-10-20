# Enable optional Tailscale access

Enable managed Tailscale when a private Tailnet route is useful for appliance or Project access. Existing clients on a reachable private LAN can continue to connect without Tailnet enrollment.

Tailscale access is optional: a reachable private LAN can provide SodaOS connectivity without Tailnet enrollment.

## Enable the managed Tailnet

Before enabling integration, have an operator account, a Tailnet you may enroll into and an intended client already authorized by that Tailnet's policy. Keep a working console or private LAN administration path available while changing access.

1. In Fountain's native Soda operator settings, enable Tailnet integration and complete the supported sign-in.
2. Confirm the managed enrollment result, then configure the Tailnet policy and any required Project routes described below.
3. Verify the actual browser, Git and Project paths you intend to use from the authorized client.

Soda manages a pinned Tailscale container under that explicit enablement and current manager state. Do not install Tailscale on the host as a substitute or run `tailscale up` on the appliance; the host identity and Project companions have separate managed lifecycles.

## Configure policy and Project routes

Tailnet policy must allow the intended device and, when needed, the Project subnet route. A Project may use its own ephemeral companion under the selected enrollment policy; it does not inherit the host's device identity or reusable credentials. Enabling the host does not by itself advertise or approve a Project route. Address discovery or a displayed Tailnet identity also does not prove that a client can reach the service.

## Verify or disable access

After enrollment and policy configuration, test from an actual Tailnet client: open the configured private browser origin and sign in. If direct Project access is needed, verify the separate Project subnet route and connect to the intended Project address. Use [private networking and TLS](50-private-networking-and-tls.md) to configure DNS, advertised Git addresses and certificate trust; these remain separate from Tailnet enrollment.

If enablement, reconciliation or cleanup is incomplete, keep the current state and inspect operator status before retrying. Disabling Tailnet closes managed admission and stops the positively identified owned host and Project containers while preserving credentials, caches and Project data. For host service diagnosis, use [Administration](../50-Operate/20-administration.md).
