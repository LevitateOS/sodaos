# SodaOS documentation

Set up SodaOS, join a persistent project environment, and develop from your browser or preferred SSH editor.

Use a powerful computer as your remote development environment or as extra
capacity beside your everyday computer. A team can share a cloud server or
hardware it owns: builds, agents, development tools and databases run on Soda,
while each person keeps their own project-local account and ordinary checkouts.

## Start with your task

| You want to… | Start here |
| --- | --- |
| Understand people, projects and shared resources | [Product model](20-product-model.md) |
| Install on hardware or a VM from an ISO | [Install on premises](../20-Deploy/20-install-on-premises.md) |
| Import a VM image or deploy on Scaleway | [Deploy to a cloud or VM](../20-Deploy/10-deploy-to-cloud.md) |
| Configure a new server for the team | [Operator setup](../20-Deploy/25-operator-setup.md) |
| Sign in and register your development key | [First connection](../20-Deploy/30-first-connection.md) |
| Find your repository and join its environment | [Projects and workspaces](../30-Use-Soda/20-projects-and-workspaces.md) |
| Connect an editor and use Git | [Connect and develop](../40-Develop/10-connect-and-develop.md) |
| Install a tool once for the project | [Shared tools and files](../40-Develop/20-shared-tools-and-files.md) |
| Run a database or development service | [Project services](../40-Develop/30-project-services.md) |
| Add a teammate | [People and access](../50-Operate/10-people-and-access.md) |
| Diagnose or maintain the server | [Administration](../50-Operate/20-administration.md) |

## Your first project session

1. Obtain the Soda dashboard URL and private-network access from your operator.
2. Sign in through Forgejo, completing any first-password change and consent.
3. Add your **public** development-access SSH key in your Soda profile.
4. Open the project's environment. Its repository owner creates the environment
   if needed; creating a repository alone does not create one.
5. Select **Add me to this project**, even if you created it.
6. Verify the project's displayed SSH identity and connect as your own user at
   its IP address, or open your existing workspace's browser terminal.
7. Clone with your own Git credentials and use the project's shared tools and
   services alongside your personal checkout.

Soda supplies a real Linux account and usable environment when you join. You do
not need a host account, a source checkout of Soda, or Cockpit access to develop.

## Choose the right interface

The [Soda dashboard](../30-Use-Soda/05-dashboard.md) brings together repositories,
collaboration and environments. [Forgejo](../30-Use-Soda/30-forgejo.md) owns Git,
identity and repository permissions. [Cockpit](../30-Use-Soda/10-cockpit.md) is
for the host operator, including [Tailscale](../30-Use-Soda/40-tailscale.md) and
[local CI runners](../30-Use-Soda/50-ci-runners.md).

## Platforms

Choose the x86-64 or AArch64 download matching the machine or VM. Both hardware
and cloud deployments use the same project model. WSL2 support for x86-64 Windows
gaming PCs is planned for a future release, with no WSL2 download; use a full
hardware or VM installation for the release-day paths.
