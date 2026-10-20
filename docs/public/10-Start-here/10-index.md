# SodaOS handbook

Install and operate SodaOS, follow autonomous Factory work in Forgejo and Spaces, or develop with your team in persistent Projects. Start with the task you need; the handbook links each procedure to the page that owns it.

## Find your path

| You want to… | Start here |
| --- | --- |
| Understand the product, its identities and authority boundaries | [Product model](20-product-model.md) |
| Choose an installation or Docker deployment | [Deployment options](../20-Install/10-deployment-options.md) |
| Install the appliance on hardware or a VM | [Install on premises](../20-Install/30-install-on-premises.md) |
| Set up Forgejo, operator access and private networking | [Operator setup](../30-Set-up/10-operator-setup.md) |
| Connect a provider account or use your own managed CLI | [Provider connections](../30-Set-up/40-provider-connections.md) |
| Plan accepted repository work and follow it to verification | [Software factory](../40-Factory/05-software-factory.md) |
| Join a Project, open a terminal and develop manually | [First connection](../30-Set-up/50-first-connection.md) |
| Develop with Git, SSH and your editor | [Connect and develop](../60-Develop/20-connect-and-develop.md) |
| Administer the appliance or diagnose a service problem | [Administration](../70-Operate/10-administration.md) |
| Maintain backups, restore data or recover the host | [Backups and restoration](../70-Operate/60-backups-and-restoration.md) |

## Follow a build from request to verified behavior

1. **Prepare the installation and team.** Select and install the appliance using [deployment options](../20-Install/10-deployment-options.md), then complete [operator setup](../30-Set-up/10-operator-setup.md), private access and team onboarding. Provider account enrollment and custody belong to [Provider connections](../30-Set-up/40-provider-connections.md).
2. **Shape the request.** Describe the outcome, review alternatives and decisions, then associate the direction with a repository in [Plan new work](../40-Factory/10-plan-new-work.md).
3. **Authorize and prepare.** The repository owner sets standing policy and required checks; the Soda operator provides bounded capacity; a provider owner sponsors a connection. Then prepare the persistent repository environment in [Prepare a Project](../50-Spaces-and-Projects/30-prepare-project.md). The distinct grants and owners are explained in [Factory authorization](../40-Factory/20-factory-authorization.md).
4. **Let eligible issues progress.** Accepted work is assessed, planned and executed with dependencies, task checkpoints and results visible in [Issue work and dependencies](../40-Factory/30-issue-work-and-dependencies.md) and the [Factory Feed](../40-Factory/70-factory-feed.md).
5. **Review and deliver.** An independent review, required native checks and repository policy govern the exact candidate. Factory confirms native merge and original issue closure as described in [Review and merge](../40-Factory/50-review-and-merge.md).
6. **Verify the integrated outcome.** The Test Pilot checks the approved outcomes against the integrated version. [Application verification](../40-Factory/60-application-verification.md) explains why issue completion and application verification are separate.

The normal eligible path does not require a manual command for every issue or a person to merge every candidate. Follow the Feed to read evidence or answer a question; use [Factory intervention](../40-Factory/90-factory-intervention.md) when an owner, decision or explicit recovery action is needed.

## Work with Soda as a person

People sign in through native Forgejo and explicitly join a repository Project when they need a development environment. [First connection](../30-Set-up/50-first-connection.md) covers joining and opening a browser terminal. You can use that path without setting up an SSH key; [Connect and develop](../60-Develop/20-connect-and-develop.md) covers direct SSH, Git and editor workflows.

Projects retain shared tools and services while keeping accounts and checkouts separate. Read [Projects and workspaces](../50-Spaces-and-Projects/20-projects-and-workspaces.md) before managing shared work. Host administration, data protection and service recovery have their own owners under [Operate](../70-Operate/10-administration.md).
