# Recover an interrupted operation

Identify which operation was interrupted and preserve its record and observed state. Route Factory runs, host releases, Project maintenance and data restoration to their distinct recovery procedures.

First identify whether the interruption affected one Factory run, a host/release operation, a Project maintenance action or data itself. Each has a different owner and recovery boundary; preserving the original record and current state is essential when the effect is uncertain.

## Recover a Factory run or release

For a Factory run or provider lease, use [Factory intervention](../40-Factory/90-factory-intervention.md). Do not apply host recovery commands to a run.

For an interrupted release transition, inspect the recorded operation and reconciliation service journal, then resume or roll back only through the exact recorded release procedure in [Updates and fallback](50-updates-and-fallback.md). The operation may have committed core host state while extension readiness remains unresolved; that result is not an automatic rollback.

## Reconcile a native Forgejo mutation

For a native Forgejo mutation that remains reserved after interruption, stop and inhibit the entire native writer domain before offline reconciliation. This includes HTTP/SSH write ingress, native services, workers, scheduled/admin commands and surviving external writers; verify no writer remains.

1. On an appliance, inspect and then stop the Forgejo writer unit:

   ```sh
   sudo /usr/bin/soda-forgejo-domain status
   sudo /usr/bin/soda-forgejo-domain stop
   sudo /usr/bin/soda-forgejo-domain inhibit
   sudo /usr/bin/soda-forgejo-domain status
   ```

   The status output shows the host unit, mask, container and offline marker. Confirm the unit is stopped, masked and the marker is present before continuing. It does not show reservation rows.

2. Use Forgejo's native status command to identify the exact held owner and fencing generation. Then run the offline recovery command from the Forgejo server binary, replacing `OWNER` and `GENERATION` with those exact recorded values:

   ```sh
   forgejo admin native-operation recover --owner OWNER --generation GENERATION
   ```

   The command refuses unless the offline marker exists. A `released` result establishes the known effect and releases that owner; `idle` means no owner remains; `fenced` (exit 3) stays held for intervention. Do not retry with different identifiers.

3. Only after a successful reconciliation, lift inhibition and start the writer:

   ```sh
   sudo /usr/bin/soda-forgejo-domain lift
   sudo /usr/bin/soda-forgejo-domain start
   ```

   The start command refuses while the offline marker remains. Verify the unit is active before admitting native writes.

There is no force-unlock or timeout-based recovery.

## Recover Project maintenance or data

Project maintenance interruptions must be reconciled against the recorded operation and exact retained container. Keep the Project stopped or otherwise in its observed state while investigating. For terminal-support, follow the check-and-same-command retry procedure in [Project maintenance](30-project-maintenance.md#recover-an-interrupted-maintenance-operation).

For Canlang directory-rename interruptions, inspect the retained directories and diagnostic output there before deciding whether any retry is safe. Do not delete records or substitute a new root.

If files or database contents must be restored, follow [Backup and restoration](60-backups-and-restoration.md). Restoration is a separate data operation with an explicit target and stopped writers; software rollback does not restore data. If the last confirmed state or affected writer domain is unclear, stop further mutation and collect bounded diagnostics through [Troubleshooting](90-troubleshooting.md).
