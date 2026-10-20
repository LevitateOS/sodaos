# Back up and restore SodaOS data

Protect the full SodaOS recovery set and coordinate its writers before restoring. Verify recovered data in an isolated target before resuming ordinary work.

## Identify the recovery set

A usable recovery set spans several independent writers and storage domains: Forgejo and Soda databases, retained Project roots and homes, application data, host configuration, broker custody and Docker volumes when applicable. A PostgreSQL dump or a copy of `/home` alone is not a complete appliance backup.

## Back up appliance databases

The daily PostgreSQL backup timer writes separate `forgejo.dump`, `soda.dump` and `globals.sql` files under `/var/lib/soda/backups/postgres/`, retaining the newest seven runs. An operator can also request a run:

```sh
sudo /usr/bin/soda-pg-backup
```

On success, the command reports the backup directory it wrote.

Each database dump is internally consistent, but the two databases are not captured as one cross-database transaction. The timer does not quiesce Projects or applications.

Schedule a broader backup with writers stopped or coordinated so database, Project, app and configuration copies describe a consistent recovery point. Keep broker encryption keys and other operator secrets protected separately from ordinary backups.

## Back up Docker volumes

For Docker, a complete backup requires the persistent `/etc`, `/var` and `/home` volumes together. Stop the runtime before backing up those volumes; preserve the Compose file, `.env` and project name so the same deployment can be identified during recovery. Do not assume a database dump contains retained Podman Projects, broker custody or app data.

## Restore PostgreSQL data

Restore is destructive to the selected target. First identify the exact database or storage domain to restore. For PostgreSQL, keep the database itself running and use these ordered steps as appliance root:

1. Confirm Factory runs and other Soda-managed work are stopped, and stop every external/native database writer. Stop the Soda backend and Forgejo, and close the socket that can reactivate the host helper:

   ```sh
   sudo systemctl stop soda-dashboard.service forgejo.service soda-host.service soda-host.socket
   sudo systemctl show --property=Id --property=ActiveState \
     soda-dashboard.service forgejo.service soda-host.service soda-host.socket
   ```

   Confirm each named unit reports `ActiveState=inactive` in the output. Do not stop `soda-postgres.service`.

2. Replace `BACKUP_TIMESTAMP` with the exact existing backup directory. This restores every database dump in the selected run; `--yes` explicitly authorizes replacement:

   ```sh
   sudo /usr/bin/soda-pg-restore --yes \
     /var/lib/soda/backups/postgres/BACKUP_TIMESTAMP
   ```

   The command reports confirmation for each database restored.

Restore roles/globals only onto a fresh cluster. Application storage uses its own stopped backup and restore procedure and writes to a new data directory. Restore a private app snapshot into a fresh data identity, then verify records and files with the matching bundle, owner credentials and origin before admitting writes.

After verifying the restored data, restore only the service states intended before maintenance. Check startup logs and native repository access before reopening ordinary work; leave deliberately stopped Projects and applications stopped. A successful restore command alone does not prove that the whole appliance is ready.

## Private application snapshots

Stop the application and confirm its writer is stopped before copying storage. Replace `DATA_ID` with the current data directory, `APP` with a short label for the application, choose a new unused backup output, and use a fresh `NEW_DATA_ID`. The helper refuses occupied output directories:

```sh
sudo /usr/libexec/soda/soda-canlang-storage backup \
  --source /home/soda-applications/data/DATA_ID \
  --out /home/soda-applications/backups/APP-stopped-001
sudo /usr/libexec/soda/soda-canlang-storage restore \
  --source /home/soda-applications/backups/APP-stopped-001 \
  --out /home/soda-applications/data/NEW_DATA_ID
```

The backup contains the authoritative storage tree and checksum inventory. Restore verifies it and assigns runtime ownership when run as root. Reopen the restored app with its matching bundle, app-local owner credentials and public origin; verify records, authorized file bytes and work receipts before allowing writes. A fresh restore is a separate data identity, not an in-place rollback or retargeting of an existing application record.

Test restoration in an isolated, fresh target and verify database contents, repository access, retained Project roots, application records/files and credential custody. An older backup omits later acknowledged writes, and restoring it is not a software rollback. For uncertain host or native writer state, see [Recovery](80-recovery.md) before restoring data.
