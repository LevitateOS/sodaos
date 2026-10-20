# Administer the SodaOS host

Inspect SodaOS services, logs, storage and networks in the correct host context. Appliance and Docker deployments have distinct administration boundaries; identify the affected context before changing state.

Use native host tools to inspect service health, logs, storage and networking. Keep appliance administration separate from Docker host administration: an appliance runs on Fedora CoreOS, while Docker runs Soda's services inside a privileged runtime and leaves the outer Linux host to its operator.

## Inspect an appliance

Sign in as root or an authorized host administrator and inspect failed units, recent journal entries, disk space and Podman container state with Cockpit or native command-line tools. Host credentials are separate from the configured Soda operator's Forgejo account; that application role alone does not grant a host shell.

For a focused service check, run `systemctl --failed` and `journalctl -b --no-pager -n 100 -u UNIT`, replacing `UNIT` with the exact service being investigated.

For a stopped unit, compare its state with the intended lifecycle before treating it as failed: newly installed apps and deliberately stopped Projects can be correctly stopped. Record the exact unit, Project or application identity and last confirmed operation using the [diagnostic worksheet](10-troubleshooting.md#collect-a-diagnostic-worksheet).

Preserve relevant diagnostics privately because logs can include configuration details. Check both the host and affected service before deciding whether the problem involves a service, disk, route or application.

## Inspect a Docker deployment

Run Soda service inspection through the original Compose project from its protected installation directory, as the host operator who installed it. Prefix Docker commands with `sudo` when that account requires it for daemon access. For example, use `docker compose exec soda systemctl --failed` to inspect failed units inside the runtime.

For an interactive root shell inside that runtime, run `docker compose exec soda bash` from the same host directory. Run the relevant Soda administration commands there, then use `exit` to return to the outer host shell.

The outer host's Docker daemon, kernel, firewall, forwarding and storage remain outside the Soda runtime; administer those with the host's own tools. Keep the same Compose file, environment and named volumes when inspecting or restarting the runtime.

## Choose the owning procedure

Stock Cockpit provides host overview, logs, storage, networking, services, containers, terminal and file inspection under root/operator authentication. It listens on port 9090; keep access private and use the [Cockpit SSH tunnel procedure](30-cockpit.md#connect-privately) if the operator network cannot reach it directly. In Docker, Cockpit inside Soda administers the runtime. It does not replace Cockpit or other administration of the outer host.

Use the task-specific procedure for changes: [same-root Project maintenance](60-project-maintenance.md), [private application administration](40-private-applications.md), [release updates](70-updates-and-fallback.md), [backup and restoration](50-backups-and-restoration.md), or [interrupted-operation recovery](80-recovery.md). If the affected component is unclear, start with [Troubleshooting](10-troubleshooting.md). Do not recreate a container or delete retained data to address an unexplained failure.
