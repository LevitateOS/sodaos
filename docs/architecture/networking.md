# Networking and access

This document owns how clients reach the appliance, projects and operator tools.

## Project reachability

Project access is ordinary `user@project-ip`, SSH/PTY/SCP/SFTP and native service
ports. Soda does not invent project DNS or a custom SSH gateway.

An observed bridge address does not prove client reachability. Host Tailnet
enrollment does not imply an advertised, approved or working project subnet route.
Keep actual client and route evidence explicit.

## Origins and endpoints

Browser/OAuth origins, listeners and Forgejo Git advertisement are distinct
configured facts. Do not infer endpoints from predecessor ports or another browser
hostname.

Soda's API/OAuth service and Spaces share the configured Forgejo HTTPS origin under
`/-/soda/`. Git authentication uses native Forgejo SSH keys or HTTPS tokens,
independently of Soda development-access public keys.

## Cockpit

Cockpit listens on all interfaces and is root/operator-only. Do not silently expose host
administration or development services publicly. Stock Cockpit administration and
its private security boundary remain even when Tailnet or Runners controls move into
native Soda operator settings.

## Tailnet model

Tailnet integration places host controls in native operator settings and supports
automatically enrolled, project-scoped ephemeral nodes.

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

Providers own CI workflows, registration authority, scheduling and results. Soda
owns local runner capacity, not a scheduler. See [Runners](../reference/runners.md).
