# Recover an interrupted operation

Identify which operation was interrupted and preserve its record and observed state. Route Factory runs, host updates, Project maintenance and data restoration to their distinct recovery procedures.

First identify whether the interruption affected one Factory run, a host update, a Project maintenance action or data itself. Each has a different owner and recovery boundary; preserving the original record and current state is essential when the effect is uncertain.

## Recover a Factory run or host update

For a Factory run or provider lease, use [Factory intervention](../20-Factory/80-factory-intervention.md). Do not apply host recovery commands to a run.

For an interrupted host update, inspect the booted deployment with `sudo rpm-ostree status`. Use the native Fedora CoreOS rollback or reinstall from the release media as described in [Updates and fallback](70-updates-and-fallback.md). Neither restores database contents or newer application data; use [Backups](50-backups-and-restoration.md) for that.

## Reconcile a native Forgejo mutation

```doc-pending
{
  "id": "offline-forgejo-invocation",
  "title": "Run Forgejo in its offline recovery context",
  "message": "The complete appliance invocation will be added once recovery with the exact server image, configuration, database and repository paths is verified.",
  "guidance": "Obtain the supported offline invocation before beginning this procedure. Preserve an existing recovery fence; do not restart the web server to obtain a shell or substitute a host binary."
}
```

For a native Forgejo mutation that remains reserved after interruption, stop and inhibit the entire native writer domain before offline reconciliation. This includes HTTP/SSH write ingress, native services, workers, scheduled/admin commands and surviving external writers; verify no writer remains.

1. On an appliance, inspect and then stop the Forgejo writer unit:

   ```sh
   sudo /usr/bin/soda-forgejo-domain status
   sudo /usr/bin/soda-forgejo-domain stop
   sudo /usr/bin/soda-forgejo-domain inhibit
   sudo /usr/bin/soda-forgejo-domain status
   ```

   The status output shows the host unit, mask, container and offline marker. Confirm `unit: forgejo.service inactive`, `masked: yes`, `container: soda-forgejo absent` and a present marker before continuing. An unknown marker is a refusal condition, not confirmation of inhibition. These host checks do not show reservation rows or prove that external scheduled/admin writers have stopped.

2. Run the native commands in the stopped deployment's offline maintenance context, using its exact Forgejo server image/binary, configuration, database access and repository storage. On the appliance, the server binary belongs to the Forgejo image at `/app/gitea/gitea`; it is not a host `forgejo` command. Stopping the service removes its running container, so `podman exec soda-forgejo` cannot provide this offline context. Do not restart the server or run its normal web entrypoint to obtain a shell.

   Preserve the deployment's configuration and path mappings. Its `AppDataPath` is `/data/gitea`, mapped to `/var/lib/soda/forgejo/gitea` on the appliance; its configuration is `/data/gitea/conf/app.ini`. The offline command must see that same marker, database and repository paths, with database credentials supplied through their existing protected files. Keep PostgreSQL available. Do not point a copied binary at an empty configuration or another database, and do not put database passwords in arguments or diagnostic output.

   With that offline context established, pass the following subcommand to the deployment's server binary to inspect reservation ownership:

   ```text
   admin native-operation status
   ```

   Status reports `state: idle` or `state: held`, the native `revision`, and `inhibited: true` or `inhibited: false`. A held reservation also reports its `owner` and `generation`. Continue only with inhibition confirmed; copy the exact held owner and positive generation into this recovery subcommand:

   ```text
   admin native-operation recover --owner OWNER --generation GENERATION
   ```

   These blocks are server subcommands, not standalone appliance-shell commands. Recovery refuses unless the offline marker exists. Read the printed `verdict`: `released` establishes the known effect and releases that owner; `idle` means no owner remains; `fenced` (exit 3) stays held for intervention. A status read or an infrastructure failure is not a release verdict. Do not retry with different identifiers or replay the original mutation to discover its outcome.

3. Only after a successful reconciliation, lift inhibition and start the writer:

   ```sh
   sudo /usr/bin/soda-forgejo-domain lift
   sudo /usr/bin/soda-forgejo-domain start
   ```

   The start command refuses while the offline marker remains. Run `sudo /usr/bin/soda-forgejo-domain status` again and verify the unit is active, the runtime mask is absent and the marker is absent before admitting native writes. Retain the recovery verdict and inspect service startup if activation fails; do not treat a successful marker removal alone as a healthy restart.

There is no force-unlock or timeout-based recovery.

## Recover Project maintenance or data

Project maintenance interruptions must be reconciled against the recorded operation and exact retained container. Keep the Project stopped or otherwise in its observed state while investigating. For terminal-support, follow the check-and-same-command retry procedure in [Project maintenance](60-project-maintenance.md#recover-an-interrupted-maintenance-operation).

For Muse tool/interface interruptions and Canlang directory-rename interruptions, use the operation-specific guidance in [Project maintenance](60-project-maintenance.md#recover-an-interrupted-maintenance-operation). Inspect the retained tool directories inside the exact Project before deciding whether a retry is safe. A retained Canlang previous directory can also be the expected result of a completed installation. Do not delete records, discard retained tools to bypass a refusal or substitute a new root.

If files or database contents must be restored, follow [Backup and restoration](50-backups-and-restoration.md). Restoration is a separate data operation with an explicit target and stopped writers; software rollback does not restore data. If the last confirmed state or affected writer domain is unclear, stop further mutation and collect bounded diagnostics through [Troubleshooting](10-troubleshooting.md).
