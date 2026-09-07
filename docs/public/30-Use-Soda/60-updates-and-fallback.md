# Maintenance and fallback

Maintain the CoreOS host and Soda application components deliberately while preserving persistent project state and current credentials.

## Plan the maintenance window

Review the selected release's instructions and data/schema compatibility. Record
the running host deployment and application versions, notify users, let runner
jobs finish, and take a consistent [backup](../50-Operate/30-backups-and-restoration.md).
Keep console access and the private-network configuration needed to reconnect.

Host deployment, Soda dashboard, Forgejo, proxy and project environments are
separate pieces. An OS change does not automatically update every application
or replace existing project roots. Soda's first-install/bootstrap commands are
not upgrade commands.

## Inspect the native host

As the host operator, begin with read-only observations:

```sh
rpm-ostree status
systemctl --failed
tailscale status
```

Fedora CoreOS owns the host deployment and its update policy. Review
[CoreOS update management](https://docs.fedoraproject.org/en-US/fedora-coreos/auto-updates/)
and [OS extensions](https://docs.fedoraproject.org/en-US/fedora-coreos/os-extensions/)
for scheduling, pending deployments and layered packages. Inspect the actual
policy rather than assuming automatic updates are disabled or that a downloaded
deployment is already active.

Use the native maintenance procedure appropriate to the installed release.
There is no Soda bootc image-switch or Cockpit Soda Updates workflow in this
host model. A project/application OCI image is not an OS update target.

## Update application components

Follow the target release's component-specific upgrade instructions. Preserve
Soda's configuration, database and grant-encryption key as a matching protected
set; preserve Forgejo's configuration/database/repositories through its native
upgrade procedure. Deploy matching dashboard assets and backend together.

Rehearse a schema-changing upgrade against a restricted copy before production.
Do not rerun `soda-setup`, overwrite OAuth configuration or reset a database to
bypass an upgrade error. Keep backups private and record what changed.

Existing projects retain their own writable roots, accounts, packages, installed
tools and data. Selecting a new default project image does not upgrade them.
Coordinate project-local package/tool changes with the project administrator;
do not delete/recreate a project to obtain new image contents.

## Restart an existing project

A host operator may stop/start the selected existing project during an agreed
window using its actual native identity:

```sh
systemctl stop soda-project@PROJECT_ID.service
systemctl start soda-project@PROJECT_ID.service
```

Replace `PROJECT_ID` with the verified existing environment ID, not a repository
name or an arbitrary container. These are service operations on the existing
project. Preserve its root; do not use replacement, pruning or deletion as startup.

After project or host restart, verify SSH identity, both users' access, homes,
shared tools/files and committed service data. Inspect existing workloads and
start them through their ordinary native tools when required. Persistence does
not promise automatic workload resurrection.

## Understand fallback limits

A previous host deployment or application binary is not a data backup. Native
host rollback has its own configuration semantics; an older application may not
understand the current schema. Review native compatibility and current state
before selecting an earlier deployment.

Restoring an old database discards later records and may break its relationship
with project accounts created since that backup. Do not mix an old database,
a different grant key and current native roots as an improvised rollback.
Choose a compatible forward repair or a coordinated, tested data restoration.

If an update or reboot loses its response, reconnect or use the console and
inspect booted/pending state before repeating it. Preserve diagnostics and
partial results; failure does not imply that nothing changed.
