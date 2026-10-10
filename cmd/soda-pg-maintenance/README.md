# Soda PostgreSQL maintenance

This package supplies the appliance operator's `soda-pg-backup`,
`soda-pg-restore` and `soda-pg-init-roles` commands. The shared Rust library
supports those binaries. Project developers normally use their own project
databases rather than these host-level tools.

The commands require native Podman access to the appliance PostgreSQL container
and its PostgreSQL utilities. Installed services run backup and role provisioning
as root. Authentication uses `podman exec -u postgres` and container-local peer
authentication; supply credentials through the managed files, not command args.

## Back up

The installed `soda-postgres-backup.timer` runs daily at 02:00. An explicit
operator backup uses:

```sh
sudo /usr/bin/soda-pg-backup
```

Each run writes `globals.sql`, `forgejo.dump` and `soda.dump` under
`/var/lib/soda/backups/postgres/YYYYMMDDTHHMMSSZ`. Each database dump has its own
consistent snapshot. The command checks custom archive magic and required role
presence, publishes the completed run, then retains the newest seven runs.
Successful output names the backup directory.

## Restore

Stop `forgejo.service` and `soda-host.service` before restoring. Restoration
uses `pg_restore --clean --if-exists`, replacing objects in the selected database.
`--yes` is required as the first argument.

```sh
sudo /usr/bin/soda-pg-restore --yes \
  /var/lib/soda/backups/postgres/20261010T020000Z
sudo /usr/bin/soda-pg-restore --yes \
  /var/lib/soda/backups/postgres/20261010T020000Z soda
```

With no database names, all `*.dump` files in the run are selected. Missing
databases are created with their same-name role as owner. Names must contain
only lowercase ASCII letters, digits or underscores and cannot start with `pg_`.
Restore roles only onto a fresh cluster using
`soda-pg-restore --yes --globals BACKUP_DIR`; ordinary restores assume the
standard roles already exist. A successful restore prints one confirmation per
database, or a globals confirmation.

## Provision roles

`soda-postgres-init.service` normally runs `/usr/bin/soda-pg-init-roles` after
database startup. It waits up to three minutes for readiness, reads the managed
password files, creates missing roles/databases, and sets role passwords to those
files. Existing database contents are retained. Credential-file generation is
handled by [soda-setup](../soda-setup/README.md).

## Environment inputs

Defaults suit the installed appliance; fixture operators can override them:

| Variable | Used by | Default |
| --- | --- | --- |
| `SODA_PG_CONTAINER` | All three commands | `soda-postgres` |
| `SODA_PG_DATABASES` | Backup and init | `forgejo soda` (space-separated) |
| `SODA_PG_BACKUP_DIR` | Backup | `/var/lib/soda/backups/postgres` |
| `SODA_PG_KEEP` | Backup | `7`; must be a positive integer |
| `SODA_PG_PASSWORD_DIR` | Init | `/etc/soda/postgres`; reads `NAME.passwd` |

Errors are reported on stderr with nonzero exit status. A completed backup or
restore confirmation applies only to that operation.
See [host database services](../../system/host/services/README.md#backup-and-restore)
and [operator setup](../../docs/guides/operator-setup.md).
