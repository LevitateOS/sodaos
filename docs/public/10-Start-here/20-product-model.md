# Product model

Understand the shared appliance, persistent project environments and personal workspaces before choosing where to work.

## Three layers

| Layer | What lives there |
| --- | --- |
| Appliance | Compute, storage, private networking, Soda dashboard, Forgejo and operator services |
| Project environment | Persistent Rocky Linux userspace, its own IP, shared installed tools, shared files and project services |
| Your workspace in that project | Your Linux account, home, ordinary Git checkouts, personal configuration, credentials and processes |

One repository has one associated environment. A workspace is your place inside
that shared environment, not another VM or a container created for every person.
You join only the projects you need. The same person can have an account in
several independent environments without having any Linux account on the host.

Tools installed in the shared project toolchain are actual shared installations,
not merely a shared download cache. Personal checkouts are not a shared writable
working tree: collaborate with branches, commits and reviews. Shared files and
services are deliberately shared; coordinate changes with teammates.

## Identities and authority

| Identity or role | Authority |
| --- | --- |
| Forgejo account | Your browser identity and upstream repository permissions |
| Project member | Your project-local files, processes, shared resources and SSH access |
| Associated repository's human owner | Administration inside that project, including shared tools and its container engine |
| Configured Soda operator | Soda environment/access administration and initial team onboarding |
| Forgejo site administrator | Forgejo administration, under Forgejo's own permissions |
| Native host root/operator | Host services, storage, networking and Cockpit |

Owning a repository or administering Forgejo does not grant host root. A project
administrator's sudo access stays inside that project. Organizations and teams
can organize repository collaboration; they do not automatically create an
organization-owned Linux environment or map team roles to project sudo.

Soda is for a trusted team on a private network. Project namespaces and account
permissions reduce accidental interference; they are not a hostile-multitenancy
security guarantee. Do not give untrusted contributors project administration or
local runner execution merely because their work is in a container.

## Four separate kinds of access

1. **Browser:** Forgejo authenticates you; Soda has its own session for its dashboard.
2. **Project SSH:** your client proves possession of a private key whose public
   half Soda installs when you explicitly join.
3. **Git:** Forgejo or another Git host authorizes your own Git key or HTTPS credential.
4. **Network:** your LAN/Tailnet route and its access rules make an endpoint reachable.

Joining an environment does not grant repository permissions. Browser sign-in
does not authenticate a CLI or create a Git key. Network access does not create
an account. Signing out of Soda does not terminate SSH sessions or guarantee
sign-out from Forgejo. See [People and access](../50-Operate/10-people-and-access.md)
for changes and revocation.

## A lasting development environment

Normal project stop/start and host reboot preserve accounts, homes, SSH host
keys, installed tools, shared files, configuration and service data. Existing
workloads may need their ordinary native start command after a restart.
Persistence is not automatic workload restart, backup or disaster recovery.

Git carries source, migrations and tool/service definitions—not running databases
or installed binaries. Merging code does not install tools, apply migrations,
restart processes, promote live state or clean up resources. Use ordinary native
tools for those actions, with the required project permissions.

Continue with [Projects and workspaces](../30-Use-Soda/20-projects-and-workspaces.md)
or [Backups and restoration](../50-Operate/30-backups-and-restoration.md).
