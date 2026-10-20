# Troubleshoot SodaOS

Route a SodaOS symptom by identity, network path, service and operation before changing state. Gather bounded, relevant diagnostics, redact private data, then use the owning procedure for repair.

## Find the owning procedure

Start with the user-visible symptom and identify the affected identity, network
path, service and operation. Use the procedure for that task:

- For waiting Factory work, [identify the blocker and its owner](../20-Factory/80-factory-intervention.md#identify-the-actual-blocker).
- For a missing or disconnected human terminal, [resolve its recorded session state](../30-Projects-and-development/20-workspace-terminals.md#resolve-unavailable-terminal-states).
- For an interrupted release, native write or maintenance action, use [Recovery](80-recovery.md) with the original operation record.
- For a private app's lifecycle or readiness, use [Private applications](40-private-applications.md); app identity and administration are separate from Project membership.

If the cause is unclear, collect the bounded worksheet below and use the
relevant symptom section. An ordinary collaborator need not run host commands
to report a problem; the authorized host administrator gathers service evidence when
needed. Redact credentials, tokens, private keys and personal data before
sharing output.

## Collect a diagnostic worksheet

Record this small set before changing state. Use exact identifiers privately, and replace them with consistent labels in shared reports. Keep passwords, tokens, private keys, setup capabilities and credential-file contents out of the worksheet.

| Evidence | What to record |
| --- | --- |
| Context | Appliance or Docker; installed release/image; client and host context. |
| Identity and action | Which Forgejo, Project, operator, provider or app-local identity attempted which action, and when. |
| Target | Exact Project, run, app and generation/revision where applicable; intended browser/Git/SSH address. |
| Last confirmed state | Last successful action, acknowledged write or observed lifecycle state; change immediately before failure. |
| Observations | Failed unit name, short relevant journal excerpt, readiness state and free space on the affected filesystem. |
| Network path | Intended client, DNS name, route, listener and TLS certificate/trust result for the failing address. |
| Next owner | The linked procedure below and any operation record whose outcome remains uncertain. |

For appliance service evidence, replace `UNIT` with the actual affected service:

```sh
sudo systemctl --failed
sudo systemctl status UNIT --no-pager
sudo journalctl -b -u UNIT --no-pager -n 100
df -h
```

The first command lists failed units; an empty list does not prove an app or client path works. Status and journal output show the named unit's observed state and recent events. Disk output identifies mounted filesystem capacity, rather than which data is safe to delete. Run these in the correct context: Docker service checks belong inside the runtime through the original Compose project, while outer-host storage, daemon and routing checks belong on the Linux host. See [Administration](20-administration.md).

## Sign-in and permission failures

If a user cannot sign in or an action is denied, distinguish Forgejo account/repository permission, Project membership, the configured Soda operator, provider connection and app-local authorization. Ask which identity is attempting which action, then follow [People and access](../40-Configure/10-people-and-access.md) or [Provider connections](../10-Start-here/50-provider-connections.md).

Example worksheet entry: “Forgejo account alice can read repository team/board; Join for Project PROJECT_A is denied; no Project Linux account is confirmed.” Repository read access is useful evidence, but does not prove Project membership or account creation. Record the denied action without sharing tokens or session cookies.

A denied Join or missing Project account belongs to the [Project procedure](../30-Projects-and-development/10-projects-and-workspaces.md).

## Browser, Git and Project reachability

If a browser page fails to load, separate DNS resolution, client route, listener availability and certificate trust. Test from the intended client and verify the configured origin.

Example: “Client laptop resolves board.soda.test to the expected private address; TCP reaches the listener; browser reports certificate trust failure; the app generation is ready.” Route and readiness success narrow this to the configured origin/certificate/client trust. Reinstalling the app does not repair trust.

Git clone advertisement and Project subnet routing are separate checks. Use [Private networking and TLS](../40-Configure/50-private-networking-and-tls.md); Tailnet enrollment alone does not prove a route.

## Host, Docker and storage symptoms

For a host service or storage symptom, inspect Cockpit's service, journal, storage and networking pages or use [Administration](20-administration.md). For Docker, inspect both the Soda runtime through Compose and the outer Linux host through its own tools.

Example: “Runtime UNIT_A is inactive; outer Docker daemon is reachable; the retained /var volume is mounted; journal excerpt follows with private values removed.” An active outer container does not establish that its internal service is healthy. Capture failed unit names and recent relevant logs; do not restart or recreate a stateful component before identifying its owner.

## Factory and application symptoms

For provider waits, check connection status, subscription availability and whether return/retirement is confirmed; use [Factory intervention](../20-Factory/80-factory-intervention.md) for a run. For unavailable required checks or candidate review, follow the Factory's CI or review owner. For app readiness, inspect the recorded generation and app service before an explicit lifecycle action; see [Private applications](40-private-applications.md).

For example, record “Run RUN_A waits on provider connection CONNECTION_A; the connection is unavailable; no return is confirmed,” or “App APP_A, generation GENERATION_A, remains stopped after Install.” The first routes to provider/run intervention; the second is the expected stopped installation state and needs an authorized explicit Start. If a request lost its response, report its last confirmed operation rather than repeating a mutation.

## Disk pressure and interrupted work

Example: “The filesystem holding Project PROJECT_A has no free space; its dirty checkout and home remain present; the last terminal command failed.” Preserve those files, identify approved disposable data with its owner, and verify free space and the affected operation afterward. A failed write does not justify deleting the retained root.

For a full disk, identify the affected filesystem and preserve evidence before freeing space. Do not remove Project roots, Docker volumes or backups as a generic cleanup. If a maintenance or release operation was interrupted, preserve its record and use [Recovery](80-recovery.md). If retained data needs repair or restoration, use [Backup and restoration](50-backups-and-restoration.md) with the exact target and stopped writers.
