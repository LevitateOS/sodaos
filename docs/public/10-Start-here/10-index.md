# SodaOS handbook

Install SodaOS, follow Factory work from an accepted issue through verification, or develop with teammates in persistent Projects. Choose the handbook path that matches the task you need to complete.

## Find your path

If your team already has SodaOS, start with the work you came to do. Request
authors describe an outcome and follow the Factory; contributors join a Project
and use their own checkout; operators administer shared services and access.
One person can hold several of these responsibilities.

| Your next task | Go directly to |
| --- | --- |
| Find what needs your attention | [Follow the Feed evidence](../20-Factory/20-factory-feed.md#follow-work-and-inspect-evidence), then [answer a question](../20-Factory/20-factory-feed.md#comment-and-route-a-request) |
| Understand why work is waiting | [Identify the blocker and its owner](../20-Factory/80-factory-intervention.md#identify-the-actual-blocker) |
| Open or reconnect to your terminal | [Open your own Project session](../30-Projects-and-development/20-workspace-terminals.md#open-or-reconnect-to-your-terminal) |
| Start or revise a Factory request | [Describe and save the direction](../20-Factory/30-plan-new-work.md#describe-the-objective) |
| Make a human contribution | [First connection](40-first-connection.md), then [Collaboration](../30-Projects-and-development/30-collaboration.md) |
| Review a candidate or check what was delivered | [Inspect the candidate](../20-Factory/50-review-and-merge.md#inspect-the-candidate), [confirm delivery](../20-Factory/50-review-and-merge.md#confirm-delivery) and [interpret application verification](../20-Factory/60-application-verification.md#understand-the-outcome) |
| Find a past answer or decision | [Search the library and assess its evidence](../20-Factory/70-knowledge-and-decisions.md#search-and-assess-what-you-find) |
| Recover an interrupted operation | [Find the recovery procedure](../50-Operate/80-recovery.md); use [Troubleshooting](../50-Operate/10-troubleshooting.md#find-the-owning-procedure) when the cause is unclear |

These paths keep their own permissions. Reading authorized repository records
does not require Project Join. A personal terminal needs Join and current
code-write access; app administration uses separate app or operator authority.
[People and access](../40-Configure/10-people-and-access.md#choose-authority-for-the-action)
identifies the required actor for each action.

For installation, administration or a broader introduction, choose the relevant
starting point below.

| You want to… | Start here |
| --- | --- |
| Understand the product, its identities and authority boundaries | [Product model](20-product-model.md) |
| Choose an installation or Docker deployment | [Deployment options](../60-Install/10-deployment-options.md) |
| Install the appliance on hardware or a VM | [Install on premises](../60-Install/40-install-on-premises.md) |
| Set up Forgejo, operator access and private networking | [Operator setup](../60-Install/60-operator-setup.md) |
| Connect a provider account or use your own managed CLI | [Provider connections](50-provider-connections.md) |
| Plan accepted repository work and follow it to verification | [Software factory](../20-Factory/10-software-factory.md) |
| Join a Project, open a terminal and develop manually | [First connection](40-first-connection.md) |
| Develop with Git, SSH and your editor | [Connect and develop](../30-Projects-and-development/40-connect-and-develop.md) |
| Administer the appliance or diagnose a service problem | [Administration](../50-Operate/20-administration.md) |
| Start, stop or replace a private application | [Private applications](../50-Operate/40-private-applications.md) |
| Review or remove someone's access | [Review access when roles change](../40-Configure/10-people-and-access.md#review-access-when-roles-change) |
| Maintain backups or restore retained data | [Backups and restoration](../50-Operate/50-backups-and-restoration.md) |

## Follow a build from request to verified behavior

If your installation and repository are already prepared, begin with the
request in step 2. The operator and repository owners maintain the setup and
standing permissions described in steps 1 and 3.

1. **Prepare the installation and team.** Select and install SodaOS using [deployment options](../60-Install/10-deployment-options.md), then complete [operator setup](../60-Install/60-operator-setup.md), private access and team onboarding. Provider account enrollment and custody belong to [Provider connections](50-provider-connections.md).
2. **Shape the request.** Describe the outcome, review alternatives and decisions, then associate the direction with a repository in [Plan new work](../20-Factory/30-plan-new-work.md).
3. **Authorize and prepare.** The repository owner sets standing policy and required checks; the Soda operator provides bounded capacity; a provider owner sponsors a connection. Then prepare the persistent repository environment in [Prepare a Project](../40-Configure/30-prepare-project.md). The distinct grants and owners are explained in [Factory authorization](../40-Configure/20-factory-authorization.md).
4. **Let eligible issues progress.** Accepted work is assessed, planned and executed with dependencies, task checkpoints and results visible in [Issue work and dependencies](../20-Factory/40-issue-work-and-dependencies.md) and the [Factory Feed](../20-Factory/20-factory-feed.md).
5. **Review and deliver.** An independent review, required native checks and repository policy govern the exact candidate. Factory confirms native merge and original issue closure as described in [Review and merge](../20-Factory/50-review-and-merge.md). Standalone content or research completes through acceptance of its exact required result.
6. **Verify the integrated outcome.** When the request includes application behavior, the Test Pilot checks the approved outcomes against the integrated version. [Application verification](../20-Factory/60-application-verification.md) explains why issue completion and application verification are separate.

The normal eligible path does not require a manual command for every issue or a person to merge every candidate. Follow the Feed to read evidence or answer a question; use [Factory intervention](../20-Factory/80-factory-intervention.md) when an owner, decision or explicit recovery action is needed.

## Work with Soda as a person

People sign in through native Forgejo and explicitly join a repository Project when they need a development environment. [First connection](40-first-connection.md) covers joining and opening a browser terminal. You can use that path without setting up an SSH key; [Connect and develop](../30-Projects-and-development/40-connect-and-develop.md) covers direct SSH, Git and editor workflows.

Projects retain shared tools and services while keeping accounts and checkouts separate. Read [Projects and workspaces](../30-Projects-and-development/10-projects-and-workspaces.md) before managing shared work. Host administration, data protection and service recovery have their own owners under [Operate](../50-Operate/20-administration.md).
