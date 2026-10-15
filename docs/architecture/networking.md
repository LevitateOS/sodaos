# Networking and access

This document owns how clients reach the appliance, projects and operator tools.

## Project reachability

Project access is ordinary `user@project-ip`, SSH/PTY/SCP/SFTP and native service
ports. Soda does not invent project DNS or a custom SSH gateway.

An observed bridge address does not prove client reachability. Host Tailnet
enrollment does not imply an advertised, approved or working project subnet route.
Keep actual client and route evidence explicit.

## Origins and endpoints

Browser origins, listeners and Forgejo Git advertisement are distinct
configured facts. Do not infer endpoints from predecessor ports or another browser
hostname.

Soda pages and product operations use Forgejo's native extension service on the
configured HTTPS origin. Public `/-/soda/` routes serve the avatar provider.
Git authentication uses native Forgejo SSH keys or HTTPS tokens, independently of
Soda development-access public keys.

## Private Canlang application endpoints

Each self-hosted application uses an explicitly configured hostname and HTTPS
origin separate from Forgejo. Use the appliance's existing local/supplied TLS
policy; the operator configures DNS and client trust. Do not infer a route from
a container address or host Tailnet enrollment. App reachability requires a
real client check before qualification.

Application cookies and authorization stay with that application. Never forward
Forgejo cookies or Identity Broker credentials into an app. Private network
access does not replace Canlang user/team/session authorization. Production
application lifetime is independent of the development Project; viewing an app
does not start it or deploy a revision.

## Cockpit

Cockpit listens on all interfaces and is root/operator-only. Do not silently expose host
administration or development services publicly. Stock Cockpit administration and
its private security boundary remain even when Tailnet controls move into
native Soda operator settings.

## Tailnet model

Optional Tailnet integration supports project-scoped ephemeral nodes when
enrollment policy enables them. Fountain owns requested enablement; the required
Soda controller performs bounded native lifecycle operations. The release pins
the official Tailscale image, acquired only when enablement prepares it.

Ordinary starts require a live, generation-bound operational grant plus fresh
Fountain manager state. Admission starts closed on host-daemon startup and stays
closed during reconciliation or failure. Disable closes the grant, drains
admitted starts and stops every positively identified owned host and Project
container. Incomplete inspection or cleanup remains unconfirmed. Credentials,
configuration, package/image caches and Project data are preserved.

The optional host service supervises the exact owned container; it is not enabled
as a boot dependency. Native CLI operations execute inside that verified image,
and LocalAPI uses its private socket. Project companions retain private user
namespaces and share only their own Project network namespace.
Private LAN operation does not require Tailnet enrollment. Factory use retains
this choice; [feature disposition](../product/scope.md#feature-disposition) owns its scope.

Invariants:

- Projects inherit enrollment **policy**, never the host's device identity or
  reusable credentials.
- Observed Tailscale address or MagicDNS identity may be displayed; Soda does not
  manufacture names or install client trust.
- Marketplace apps, persistent Project OS roots, isolated AI jobs and account-owned
  desktop sessions have distinct native lifetimes and credentials; sharing UI does
  not combine privileges.

Operator procedures: [Operator setup](../guides/operator-setup.md).
API surface: [HTTP API](../reference/api.md).

## CI networking

Forgejo owns CI workflows, registration authority, scheduling and results. The
factory uses separately managed Actions capacity; Soda-provisioned local execution
is deferred and no local observation or cleanup controls ship.
