# Back up and restore SodaOS data

A usable recovery set spans several independent writers and storage domains. Plan and verify backups for Forgejo and Soda databases, retained Project roots and homes, application data, host configuration, broker custody and Docker volumes when applicable. A PostgreSQL dump or a copy of `/home` alone is not a complete appliance backup.

On the appliance, the daily PostgreSQL backup timer writes separate `forgejo.dump`, `soda.dump` and `globals.sql` files under `/var/lib/soda/backups/postgres/`, retaining the newest seven runs. An operator can also request a run:

```sh
sudo /usr/bin/soda-pg-backup
```

Each database dump is internally consistent, but the two databases are not captured as one cross-database transaction, and the timer does not quiesce Projects or applications. For a coordinated PostgreSQL restore, first confirm Factory runs and other Soda-managed work are stopped, and stop every external/native database writer. Stop the Soda backend and Forgejo, and close the socket that can reactivate the host helper; leave PostgreSQL running so the restore command can connect:

```sh
sudo systemctl stop soda-dashboard.service forgejo.service soda-host.service soda-host.socket
sudo systemctl show --property=Id --property=ActiveState \
  soda-dashboard.service forgejo.service soda-host.service soda-host.socket
```

Confirm each stopped unit reports inactive before restoring. Do not stop `soda-postgres.service`. Schedule a broader backup with writers stopped or coordinated so database, Project, app and configuration copies describe a consistent recovery point. Keep broker encryption keys and other operator secrets protected separately from ordinary backups.

For Docker, a complete backup requires the persistent `/etc`, `/var` and `/home` volumes together. Stop the runtime before backing up those volumes; preserve the Compose file, `.env` and project name so the same deployment can be identified during recovery. Do not assume a database dump contains retained Podman Projects, broker custody or app data.

Restore is destructive to the selected target. First identify the exact database or storage domain and ensure its writers are stopped as above. The following restores every database dump in the selected backup run; `--yes` explicitly authorizes replacement:

```sh
sudo /usr/bin/soda-pg-restore --yes \
  /var/lib/soda/backups/postgres/BACKUP_TIMESTAMP
```

Replace `BACKUP_TIMESTAMP` with the exact existing backup directory. Restore roles/globals only onto a fresh cluster. Application storage uses its own stopped backup and restore procedure and writes to a new data directory. Restore a private app snapshot into a fresh data identity, then verify records and files with the matching bundle, owner credentials and origin before admitting writes.

After verifying the restored data, restore only the service states intended before maintenance. Check startup logs and native repository access before reopening ordinary work; leave deliberately stopped Projects and applications stopped. A successful restore command alone does not prove that the whole appliance is ready.

### Private application snapshots

Stop the application and confirm its writer is stopped before copying storage. Use the installed helper with a new output path:

```sh
sudo /usr/libexec/soda/soda-canlang-storage backup \
  --source /home/soda-applications/data/DATA_ID \
  --out /home/soda-applications/backups/APP-stopped-001
sudo /usr/libexec/soda/soda-canlang-storage restore \
  --source /home/soda-applications/backups/APP-stopped-001 \
  --out /home/soda-applications/data/NEW_DATA_ID
```

Replace `DATA_ID` with the exact current data directory, choose a new unused backup output and use a fresh `NEW_DATA_ID`; occupied outputs are refused. The snapshot contains the authoritative storage tree and checksum inventory. Restore verifies it and assigns runtime ownership when run as root. Reopen the restored app with its matching bundle, app-local owner credentials and public origin; verify records, authorized file bytes and work receipts before allowing writes. A fresh restore is a separate data identity, not an in-place rollback or retargeting of an existing application record.

Test restoration in an isolated, fresh target and verify database contents, repository access, retained Project roots, application records/files and credential custody. An older backup omits later acknowledged writes, and restoring it is not a software rollback. For uncertain host or native writer state, see [Recovery](80-recovery.md) before restoring data.
