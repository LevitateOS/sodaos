# Recover an interrupted operation

First identify whether the interruption affected one Factory run, a host/release operation, a Project maintenance action or data itself. Each has a different owner and recovery boundary; preserving the original record and current state is essential when the effect is uncertain.

For a Factory run or provider lease, use [Factory intervention](../40-Factory/90-factory-intervention.md). Do not apply host recovery commands to a run. For an interrupted release transition, inspect the recorded operation and reconciliation service journal, then resume or roll back only through the exact recorded release procedure in [Updates and fallback](50-updates-and-fallback.md). The operation may have committed core host state while extension readiness remains unresolved; that result is not an automatic rollback.

For a native Forgejo mutation that remains reserved after interruption, recovery requires stopping and inhibiting the entire native writer domain before offline reconciliation. Stop HTTP/SSH write ingress, native services, workers, scheduled/admin commands and any surviving external writers; verify no writer remains. On an appliance, inspect host-side state with `sudo /usr/bin/soda-forgejo-domain status`, then stop and inhibit Forgejo:

```sh
sudo /usr/bin/soda-forgejo-domain stop
sudo /usr/bin/soda-forgejo-domain inhibit
sudo /usr/bin/soda-forgejo-domain status
```

Use Forgejo's own native status command to identify the exact held owner and fencing generation. With that evidence, run `forgejo admin native-operation recover --owner OWNER --generation GENERATION` from the Forgejo server binary. Replace both placeholders with the recorded values; the command refuses unless the offline marker exists. A `released` result establishes the known effect and releases that exact owner; `idle` means no owner remains; `fenced` (exit 3) stays held for intervention. Do not retry with different identifiers. Only after a successful reconciliation, lift inhibition and start the writer:

```sh
sudo /usr/bin/soda-forgejo-domain lift
sudo /usr/bin/soda-forgejo-domain start
```

The host status command does not reveal native reservation rows. There is no force-unlock or timeout-based recovery.

Project maintenance interruptions must be reconciled against the recorded operation and exact retained container. Keep the Project stopped or otherwise in its observed state while investigating. For terminal-support, follow the check-and-same-command retry procedure in [Project maintenance](30-project-maintenance.md#recover-an-interrupted-maintenance-operation). For Canlang directory-rename interruptions, inspect the retained directories and diagnostic output there before deciding whether any retry is safe. Do not delete records or substitute a new root.

If files or database contents must be restored, follow [Backup and restoration](60-backups-and-restoration.md). Restoration is a separate data operation with an explicit target and stopped writers; software rollback does not restore data. If the last confirmed state or affected writer domain is unclear, stop further mutation and collect bounded diagnostics through [Troubleshooting](90-troubleshooting.md).
