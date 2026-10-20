# Administer the SodaOS host

Inspect SodaOS services, logs, storage and networks in the correct host context. Appliance and Docker deployments have distinct administration boundaries; identify the affected context before changing state.

Use native host tools to inspect service health, logs, storage and networking. Keep appliance administration separate from Docker host administration: an appliance runs on Fedora CoreOS, while Docker runs Soda's services inside a privileged runtime and leaves the outer Linux host to its operator.

## Inspect an appliance

Sign in as the host root/operator and inspect failed units, recent journal entries, disk space and Podman container state with Cockpit or native command-line tools. For a focused service check, run `systemctl --failed` and `journalctl -b --no-pager -n 100 -u UNIT`, replacing `UNIT` with the exact service being investigated.

Preserve relevant diagnostics privately because logs can include configuration details. Check both the host and affected service before deciding whether the problem involves a service, disk, route or application.

## Inspect a Docker deployment

Run Soda service inspection through the Compose project from its protected installation directory. For example, use `docker compose exec soda systemctl --failed` to inspect failed units inside the runtime.

The outer host's Docker daemon, kernel, firewall, forwarding and storage remain outside the Soda runtime; administer those with the host's own tools. Keep the same Compose file, environment and named volumes when inspecting or restarting the runtime.

## Choose the owning procedure

Stock Cockpit provides host overview, logs, storage, networking, services, containers, terminal and file inspection under root/operator authentication. It listens on port 9090; keep access private and use an SSH tunnel if the operator network cannot reach it directly. In Docker, Cockpit inside Soda administers the runtime. It does not replace Cockpit or other administration of the outer host.

Use the task-specific procedure for changes: [same-root Project maintenance](30-project-maintenance.md), [private application administration](40-private-applications.md), [release updates](50-updates-and-fallback.md), [backup and restoration](60-backups-and-restoration.md), or [interrupted-operation recovery](80-recovery.md). If the affected component is unclear, start with [Troubleshooting](90-troubleshooting.md). Do not recreate a container or delete retained data to address an unexplained failure.
