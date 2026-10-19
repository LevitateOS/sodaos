# SodaOS handbook

Install and operate SodaOS, an open-source operating system for AI software factories with persistent Projects, Forgejo collaboration and governed agent workflows.

Built on Fedora CoreOS, SodaOS integrates the infrastructure for continuing AI
development. People authorize objectives and operating policy; the factory
automatically picks up ready, authorized issues and coordinates planning, coding,
review, corrections, checks and native merge. The normal loop is fully autonomous
under standing policy; human approval gates are optional. Forgejo owns
repositories, issues, pull requests, reviews and CI records.
Humans and factory roles share persistent Podman Projects with separate accounts,
checkouts and executions. Familiar editors, terminals and SSH support development
and intervention.

## Start with your task

| You want to… | Start here |
| --- | --- |
| Understand the factory and human projects | [Product model](20-product-model.md) |
| Install on hardware from an ISO | [Install on premises](../20-Deploy/20-install-on-premises.md) |
| Install in a compatible VM | [Deploy to a cloud or VM](../20-Deploy/10-deploy-to-cloud.md) |
| Prepare a Docker deployment | [Run Soda OS with Docker](../20-Deploy/22-docker.md) |
| Configure private access and the factory | [Operator setup](../20-Deploy/25-operator-setup.md) |
| Authorize and inspect the first agent task | [Software factory walkthrough](../30-Use-Soda/15-software-factory.md) |
| Review a candidate and govern its merge | [Collaboration](../30-Use-Soda/35-collaboration.md) |
| Manage runs, capacity or interruption | [Administration](../50-Operate/20-administration.md) |
| Manage human and agent access | [People and access](../50-Operate/10-people-and-access.md) |
| Develop or intervene manually | [Projects and workspaces](../30-Use-Soda/20-projects-and-workspaces.md) |
| Connect an editor and use Git | [Connect and develop](../40-Develop/10-connect-and-develop.md) |
| Protect persistent data and retained results | [Backups and restoration](../50-Operate/30-backups-and-restoration.md) |

## Your first authorized task

1. Deploy Soda and configure private access, Forgejo and the selected agent profile.
2. Choose a private repository and write an issue with a clear expected outcome.
3. Establish standing repository policy, appliance capacity and provider sponsorship. Soda picks up issues created by authorized maintainers automatically; objectives without trusted authorization need adoption.
4. Observe the bounded attempt and inspect its pull request, CI and current review.
5. If a repair produces another commit, check its new verification.
6. Supply human approval where policy requires it, then confirm the policy-governed native merge or resolve the intervention request.
7. Confirm the run's recorded cleanup independently of its terminal outcome.

Follow the [factory walkthrough](../30-Use-Soda/15-software-factory.md) for the
workflow and operator commands. An issue, label or successful process exit alone is insufficient
authorization or verification.

## Choose the right interface

[Forgejo](../30-Use-Soda/30-forgejo.md) owns code, permissions and collaboration.
The factory coordinator owns automatic progress under the accepted requirements
and grants. The operator command inspects recorded runs, stops one run and
reconciles outstanding execution; it does not admit or launch work.
The [Soda Dashboard](../30-Use-Soda/05-dashboard.md) provides repository and human
project access. [Cockpit](../30-Use-Soda/10-cockpit.md) is for host administration.
The [CI runners](../30-Use-Soda/50-ci-runners.md) supply Forgejo verification capacity.

For manual development, follow [First connection](../20-Deploy/30-first-connection.md),
join a persistent project explicitly and use its displayed account and connection
details. A human project account is separate from a factory execution identity.

## Platforms

Choose x86-64 hardware or a matching VM. Run Soda on hardware you control or a
compatible private cloud VM with a documented boot path and private client access.
WSL2 support for x86-64 Windows gaming PCs is planned for a future release,
with no WSL2 download. Use the full hardware or VM paths at launch.
