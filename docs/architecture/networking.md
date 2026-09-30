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

## Cockpit

Cockpit listens on all interfaces and is root/operator-only. Do not silently expose host
administration or development services publicly. Stock Cockpit administration and
its private security boundary remain even when Tailnet or Runners controls move into
native Soda operator settings.

## Tailnet model

Optional Tailnet integration places host controls in native operator settings and
supports project-scoped ephemeral nodes when enrollment policy enables them.
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
is deferred. Existing local observation and cleanup do not establish usable CI
capacity. See [Runners](../reference/runners.md).
