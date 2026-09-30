# Soda OS handbook

Deploy Soda on infrastructure you control, run factory tasks through the operator interface, and work in persistent projects.

The operator-invoked factory workflow creates fresh environments for approved
work, enforces time and resource limits, and records outcomes and cleanup. Forgejo
keeps repositories, issues, pull requests, reviews and CI. In this workflow,
people admit objectives and merge verified results.
Persistent human projects support development, debugging and intervention through
familiar editors, browser terminals and SSH.

## Start with your task

| You want to… | Start here |
| --- | --- |
| Understand the factory and human projects | [Product model](20-product-model.md) |
| Install on hardware from an ISO | [Install on premises](../20-Deploy/20-install-on-premises.md) |
| Import a VM image or deploy on Scaleway | [Deploy to a cloud or VM](../20-Deploy/10-deploy-to-cloud.md) |
| Configure private access and the factory | [Operator setup](../20-Deploy/25-operator-setup.md) |
| Authorize and inspect the first agent task | [Software factory walkthrough](../30-Use-Soda/15-software-factory.md) |
| Review a candidate and merge it yourself | [Collaboration](../30-Use-Soda/35-collaboration.md) |
| Manage runs, capacity or interruption | [Administration](../50-Operate/20-administration.md) |
| Manage human and agent access | [People and access](../50-Operate/10-people-and-access.md) |
| Develop or intervene manually | [Projects and workspaces](../30-Use-Soda/20-projects-and-workspaces.md) |
| Connect an editor and use Git | [Connect and develop](../40-Develop/10-connect-and-develop.md) |
| Protect persistent data and retained results | [Backups and restoration](../50-Operate/30-backups-and-restoration.md) |

## Your first authorized task

1. Deploy Soda and configure private access, Forgejo and the selected agent profile.
2. Choose a private repository and write an issue with a clear expected outcome.
3. Admit the issue explicitly through the operator interface.
4. Run the bounded attempt and inspect its pull request, CI and fresh review.
5. If a repair produces another commit, check its new verification.
6. Merge the final verified commit yourself, or resolve the intervention request.
7. Confirm the run's recorded cleanup independently of its terminal outcome.

Follow the [factory walkthrough](../30-Use-Soda/15-software-factory.md) for the
actual commands. An issue, label or successful process exit alone is insufficient
authorization or verification.

## Choose the right interface

[Forgejo](../30-Use-Soda/30-forgejo.md) owns code, permissions and collaboration.
The factory operator command owns explicit admission, run status and cancellation.
The [Soda Dashboard](../30-Use-Soda/05-dashboard.md) provides repository and human
project access. [Cockpit](../30-Use-Soda/10-cockpit.md) is for host administration.
The [CI runners](../30-Use-Soda/50-ci-runners.md) supply Forgejo verification capacity.

For manual development, follow [First connection](../20-Deploy/30-first-connection.md),
join a persistent project explicitly and use its displayed account and connection
details. A human project account is separate from a factory execution identity.

## Platforms

Choose x86-64 or AArch64 to match the machine or VM. Run Soda on hardware you
control or a private cloud instance; Scaleway is the first team cloud path.
WSL2 support for x86-64 Windows gaming PCs is planned for a future release,
with no WSL2 download. Use the full hardware or VM paths at launch.
