# Understand persistent data and removal

Identify the data owner and lifetime before removing SodaOS state. Access removal, Project stop, container recreation and application unlinking have different effects and do not automatically erase retained data.

SodaOS stores state in several places with different owners and lifetimes. Stopping an operation or revoking access does not erase its retained state.

## Project and Factory state

Factory attempts and temporary run resources have bounded cleanup rules. A Project is a persistent shared environment: it can retain Linux accounts and homes, checkouts, installed tools, dirty work, SSH keys, service volumes and human data across stop/start and appliance maintenance.

The appliance's writable roots live in Podman storage, so backing up `/home` alone does not capture them. See [Backup and restoration](60-backups-and-restoration.md) before changing or removing data.

## Forgejo, access and application data

Forgejo owns native repositories, issues, pull requests and database state. Project membership or Forgejo permission removal changes access but does not automatically delete a Linux account, end an already running shell, revoke an installed key, or erase files. App-local accounts and application storage are separate from both Forgejo and Project state. Removing a Project association from an application changes navigation, not its storage or service lifetime.

## Docker volumes and host data

Docker persists Soda state in `/etc`, `/var` and `/home` volumes. Recreating or stopping the container can preserve that data; deleting volumes destroys it. Treat volume removal as an explicit data-erasure action after confirming the deployment, backup and affected teams. The outer Docker host may also contain other workloads outside Soda's ownership.

## Choose the data owner

Use the owning procedure for deliberate changes: [People and access](../30-Set-up/30-people-and-access.md) for authority changes, [Project lifecycle](../50-Spaces-and-Projects/20-projects-and-workspaces.md) for shared environments, and [Private applications](40-private-applications.md) for app lifecycle and storage. Soda does not provide an implicit archive, delete or rebuild workflow for retained human data. Never use destructive recreation as an unexplained repair.
