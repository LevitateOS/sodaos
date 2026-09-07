# Backups and restoration

Protect the appliance's mutable identities, repositories and project data using consistent native backups and tested restoration.

A persistent environment survives normal startup; it does not survive arbitrary
storage loss without a backup. Git protects only material present in another
copy, not unpushed commits, untracked files, installed tools or database volumes.
Soda does not supply a coordinated backup/restore service.

## Inventory what must survive

| Scope | Include |
| --- | --- |
| Soda | Database, configuration, grant-encryption key and integration credentials as a matching protected set |
| Forgejo | Configuration, database, repositories, LFS, attachments, packages and application secrets |
| Project environments | Container identity and writable roots, accounts/UIDs, host keys, authorized keys, homes, shared files, tools and configuration |
| Nested workloads | Engine state, named volumes, bind-mounted data, secrets and native service definitions |
| Host | Deployment information, configuration, storage/mount layout, operator access and networking |
| Tailscale | Identity/state and a deliberate restore or re-enrollment plan |
| Runners | Required native registration/work state or a deliberate provider re-registration plan |

Soda configuration lives under `/etc/soda`; its data and retained integration
state use `/var/lib/soda`. Discover actual database, Forgejo volume, project
storage and bind-mount paths from the running deployment. Copying `/home` alone
misses project accounts and data inside container storage. Exporting just a
container image misses volumes and may omit runtime identity/configuration.

Preserve numeric ownership, permissions, links, ACLs, extended attributes and
required SELinux metadata. Project user-namespace mappings and host storage
relationships matter; file copies without their identity context can assign
files to the wrong users.

Backups contain passwords/hashes, private keys and tokens. Encrypt and restrict
them, keep an independent off-machine copy, protect recovery keys separately,
and choose retention against the work you can afford to lose.

## Take a consistent backup

A coordinated, powered-off whole-machine/disk backup is a straightforward way
to preserve related state together:

1. Notify developers and let jobs finish. Save buffers, push intended commits
   and quiesce project/database writers through native controls.
2. Shut the machine down normally and confirm it is stopped.
3. Capture all boot and data volumes as one set with the platform/storage tool.
   Include separately attached project and application storage.
4. Export or copy the set to protected independent storage. Record the release,
   architecture, time, included volumes and restore instructions.
5. Restart only after capture completes and verify normal service access.

A snapshot on the same failed storage is not independent protection. A live disk
snapshot is not automatically application-consistent; use the provider's actual
guarantees and each application's quiescing procedure.

For file/application backups, follow [Forgejo backup guidance](https://forgejo.org/docs/latest/admin/upgrade/#backup)
and your databases' native methods. Use SQLite's supported backup mechanism or
stop its writer before copying; a live main database file without its WAL state
is not a reliable backup. Coordinate the Soda records, encryption key and native
project state at the same recovery point.

## Restore in isolation first

1. Verify the backup using the creating tool and choose matching architecture
   and sufficient storage.
2. Keep the original off, or isolate the restore from production networks and
   providers. Never run duplicate Tailnet, runner or machine identities together.
3. Restore the complete recorded storage/configuration set through its native
   procedure, retaining mappings, ownership, modes and required labels.
4. Boot with console access and inspect storage and services before allowing
   developer writes, provider jobs or normal network identity activation.
5. Test the checks below and deliberately resolve duplicate identities before
   handing the restored server to users.

Restoration returns data to the backup time. An older Soda database does not
contain memberships created later, even if later project roots still exist.
Do not mix unrelated backup generations or replace a production database while
writers continue. A host image fallback is not this coordinated restore.

## Test the restored work

Verify operator access and intended host/application versions, then check:

- Forgejo login, repository browse/clone and important LFS/attachment/package data;
- Soda identity/membership associations and usable authenticated connection details;
- project container identities, public SSH host keys, account ownership and homes;
- an unpushed commit and an untracked file intentionally included in the backup;
- shared executable paths, shared files and actual committed database values;
- private routing and firewall policy, without duplicate machine identity;
- runner registrations only after a deliberate provider/identity review.

Record the backup and restoration results. An archive command succeeding is not
proof of recovery. Repeat restore testing after storage, schema, credential or
backup-method changes.
