# Troubleshoot SodaOS

Route a SodaOS symptom by identity, network path, service and operation before changing state. Gather bounded, relevant diagnostics, redact private data, then use the owning procedure for repair.

Start with the user-visible symptom and identify the affected identity, network path, service and operation. Collect only diagnostics needed to route the issue, and redact credentials, tokens, private keys and personal data before sharing output.

## Sign-in and permission failures

If a user cannot sign in or an action is denied, distinguish Forgejo account/repository permission, Project membership, the configured Soda operator, provider connection and app-local authorization. Ask which identity is attempting which action, then follow [People and access](../30-Set-up/30-people-and-access.md) or [Provider connections](../30-Set-up/40-provider-connections.md).

A denied Join or missing Project account belongs to the [Project procedure](../50-Spaces-and-Projects/20-projects-and-workspaces.md).

## Browser, Git and Project reachability

If a browser page fails to load, separate DNS resolution, client route, listener availability and certificate trust. Test from the intended client and verify the configured origin.

Git clone advertisement and Project subnet routing are separate checks. Use [Private networking and TLS](../30-Set-up/20-private-networking-and-tls.md); Tailnet enrollment alone does not prove a route.

## Host, Docker and storage symptoms

For a host service or storage symptom, inspect Cockpit's service, journal, storage and networking pages or use [Administration](10-administration.md). For Docker, inspect both the Soda runtime through Compose and the outer Linux host through its own tools.

Capture failed unit names and recent relevant logs; do not restart or recreate a stateful component before identifying its owner.

## Factory and application symptoms

For provider waits, check connection status, subscription availability and whether return/retirement is confirmed; use [Factory intervention](../40-Factory/90-factory-intervention.md) for a run. For unavailable required checks or candidate review, follow the Factory's CI or review owner. For app readiness, inspect the recorded generation and app service before an explicit lifecycle action; see [Private applications](40-private-applications.md).

## Disk pressure and interrupted work

For a full disk, identify the affected filesystem and preserve evidence before freeing space. Do not remove Project roots, Docker volumes or backups as a generic cleanup. If a maintenance or release operation was interrupted, preserve its record and use [Recovery](80-recovery.md). If retained data needs repair or restoration, use [Backup and restoration](60-backups-and-restoration.md) with the exact target and stopped writers.
