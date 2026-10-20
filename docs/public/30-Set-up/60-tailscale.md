# Enable optional Tailscale access

Tailscale can provide a private Tailnet route to SodaOS and selected Project networks. It is optional: clients on a reachable private LAN can use SodaOS without Tailnet enrollment.

The appliance operator enables Tailnet integration through Fountain's native Soda operator settings and completes the supported sign-in there. Soda manages a pinned Tailscale container under that explicit enablement and current manager state. Do not install Tailscale on the host as a substitute or run `tailscale up` on the appliance; the host identity and Project companions have separate managed lifecycles.

Tailnet policy must allow the intended device and, when needed, the Project subnet route. A Project may use its own ephemeral companion under the selected enrollment policy; it does not inherit the host's device identity or reusable credentials. Enabling the host does not by itself advertise or approve a Project route. Address discovery or a displayed Tailnet identity also does not prove that a client can reach the service.

After enrollment and policy configuration, test from an actual Tailnet client: open the configured private browser origin and sign in. If direct Project access is needed, verify the separate Project subnet route and connect to the intended Project address. Use [private networking and TLS](20-private-networking-and-tls.md) to configure DNS, advertised Git addresses and certificate trust; these remain separate from Tailnet enrollment.

If enablement, reconciliation or cleanup is incomplete, keep the current state and inspect operator status before retrying. Disabling Tailnet closes managed admission and stops the positively identified owned host and Project containers while preserving credentials, caches and Project data. For host service diagnosis, use [Administration](../70-Operate/10-administration.md).
