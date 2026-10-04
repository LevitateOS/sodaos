# Appliance PostgreSQL runtime

Single PostgreSQL 17 container serving both Forgejo and the Soda store,
per the owner-selected default (volume-backed, nightly backup). Topology is
established by the units in this directory; this file records the contracts
other lanes build on.

## Units

| Unit | Role |
| --- | --- |
| `soda.network` | Appliance container network (`systemd-soda`) |
| `soda-postgres.container` | PostgreSQL 17.11, digest-pinned, data at `/var/lib/soda/postgres`, no published ports |
| `soda-postgres-init.service` | Idempotent role/database provisioning, runs before Forgejo |
| `soda-forgejo-migrate.service` | One-shot config migration before Forgejo (drops legacy inline `[database] PASSWD`) |
| `forgejo.container` | Forgejo on `postgres` via `soda-postgres:5432`, requires the init and migrate units |
| `soda-postgres-backup.service` + `.timer` | Consistent nightly dump at 02:00 into `/var/lib/soda/backups/postgres`, keeps 7 runs |

Startup chain: `soda-postgres` → `soda-postgres-init` → `soda-forgejo-migrate` → `forgejo`.
The Soda store (A11) orders its service after `soda-postgres-init.service`
the same way; backup runs after it so globals always hold both roles.

No data transfer is in scope: the recorded lab state is one Forgejo user
with zero repositories and a Soda store with zero projects, so appliances
initialize fresh databases. Protected-data obligations stay with the owner
if that changes.

## Secrets

Setup generates three mode-0600 root-only files (installer lane, out of
band — never in source, argv, logs or evidence):

- `/etc/soda/postgres/super.passwd` — postgres superuser, mounted into the
  DB container only (`POSTGRES_PASSWORD_FILE`)
- `/etc/soda/postgres/forgejo.passwd` — forgejo role; consumed by the init
  unit over peer auth and mounted into the Forgejo container at
  `/etc/forgejo/db_passwd` (`PASSWD_URI=file:...`)
- `/etc/soda/postgres/soda.passwd` — soda role, consumed by the init unit;
  the Soda store runtime reads it the same way (A11)

App-role passwords are never mounted into the DB container: entrypoint init
runs as uid 999 and cannot read root-only mounts, which is why role
provisioning lives in the root init unit instead of initdb.d. Backup,
restore and init authenticate over container-local peer auth
(`podman exec -u postgres`), so no password appears in process arguments.

## Backup and restore

- `soda-pg-backup` dumps globals plus `forgejo` and `soda` (each dump a
  single consistent snapshot) into a timestamped run, verifies archive
  magic and role presence, then rotates to the newest 7 runs.
- `soda-pg-restore --yes <run> [db ...]` restores dumps, recreating
  missing databases owned by their same-name role. Stop `forgejo.service`
  and soda-host first. `--globals` restores roles onto a fresh cluster;
  normal restores assume first init already created them.

## Disposable test fixture

`scripts/pg-fixture.sh` starts an ephemeral loopback PostgreSQL on the
same pinned image and role==database shape with random per-run passwords:

```sh
eval "$(scripts/pg-fixture.sh start)"
# $SODA_PG_HOST:$SODA_PG_PORT, $SODA_PG_DATABASES, $SODA_PG_DIR/*.passwd (0600)
scripts/pg-fixture.sh stop "$SODA_PG_CONTAINER"; rm -rf "$SODA_PG_DIR"
```

Passwords are exposed as file paths only; stdout carries no secrets.
`SODA_PG_DATABASES=""` starts a bare cluster to test provisioning itself.
Exit 3 means engine/image unavailable and tests must skip. Forgejo-side
tests (N-ST1) invoke the same script via the sodaos checkout.

The round-trip test is `scripts/pg_backup_test.go`; the unit-text contract
is `scripts/pg_runtime_test.go`.

## Known bounds

- The postgres image is digest-pinned with `Pull=missing`; it is not yet
  folded into the release payload import that forgejo/dashboard/proxy use,
  so first boot pulls it. C1 can promote it to payload delivery.
- Timer enablement and password generation at setup belong to the
  installer lane against the contracts above.
- Representative load, recovery targets and connection tuning remain open;
  no `MAX_OPEN_CONNS`/resource overrides are set.
