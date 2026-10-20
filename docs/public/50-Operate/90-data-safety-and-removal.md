# Understand persistent data and removal

Identify the data owner and lifetime before removing SodaOS state. Access removal, Project stop, container recreation and application linking have different effects and do not automatically erase retained data.

SodaOS stores state in several places with different owners and lifetimes. Stopping an operation or revoking access does not erase its retained state.

## Project and Factory state

Factory attempts and temporary run resources have bounded cleanup rules. A Project is a persistent shared environment: it can retain Linux accounts and homes, checkouts, installed tools, dirty work, SSH keys, service volumes and human data across stop/start and appliance maintenance.

The appliance's writable roots live in Podman storage, so backing up `/home` alone does not capture them. See [Backup and restoration](50-backups-and-restoration.md) before changing or removing data.

## Forgejo, access and application data

Forgejo owns native repositories, issues, pull requests and database state.
Removing Forgejo permission can block managed Project access, but does not
alone revoke installed SSH keys, end an independent shell or erase files.
Explicit [Project access revocation](../40-Configure/10-people-and-access.md#review-access-when-roles-change)
removes that member's installed keys and ends their native sessions and
processes while retaining the Linux identity, home and work.

App-local accounts and application storage are separate from both Forgejo and
Project state. Linking an application to a Project, or changing that
association, changes navigation only; it does not move storage or change the
application's service lifetime.

## Docker volumes and host data

Docker persists Soda state in `/etc`, `/var` and `/home` volumes. Recreating or stopping the container can preserve that data; deleting volumes destroys it. Treat volume removal as an explicit data-erasure action after confirming the deployment, backup and affected teams. The outer Docker host may also contain other workloads outside Soda's ownership.

## Retain old data after application Restore

The explicit application Restore operation keeps the old data tree and revision while binding the same app/address to verified fresh data. That retained tree is distinct from the restored app's current storage. Do not delete either tree to resolve an unknown restore outcome; inspect the recorded operation and follow [application snapshot recovery](50-backups-and-restoration.md#restore-the-same-app-at-its-existing-address).

Before deliberately erasing retained data, identify its exact owner, confirm that no runtime or pending operation references it, and preserve a tested recovery copy. A whole-system recovery copy must include the [complete cold recovery set](50-backups-and-restoration.md#capture-a-complete-cold-recovery-point), rather than only the files convenient to copy.

## Choose the data owner

Use the owning procedure for deliberate changes: [People and access](../40-Configure/10-people-and-access.md) for authority changes, [Project lifecycle](../30-Projects-and-development/10-projects-and-workspaces.md) for shared environments, and [Private applications](40-private-applications.md) for app lifecycle and storage. Soda does not provide an implicit archive, delete or rebuild workflow for retained human data. Never use destructive recreation as an unexplained repair.
