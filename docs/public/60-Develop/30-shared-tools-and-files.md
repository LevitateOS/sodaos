# Shared tools and files

A Project has persistent shared tool installations and team file locations for work across personal checkouts and Factory attempts. Project administrators maintain shared tools; members use them from their own accounts.

The Rocky headless Project includes mise and common development tools. Shared mise installations and their global configuration live in `/opt/mise` and `/etc/mise/config.toml`. Administrators install additional shared packages or mise-managed versions using native Project administration and the repository's trusted tool declarations. Members can select installed tools through the Project's mise setup and use repository-specific configuration after reviewing and trusting it. Personal clones, preferences and overrides belong in each member's home, not in the shared installation.

Shared files live under `/srv/project/shared` and are available through `~/shared` for joined users. Use these locations when a file is deliberately shared with the Project team. Keep personal work, credentials and private Git clones in your own home. Shared directories are writable collaboration space; do not use them as the trusted source for Factory setup approval or place provider tokens there.

Project-wide package and tool changes affect other people and Factory readiness. Coordinate them with the Project administrator and hold affected Factory work before changing an accepted setup input or shared prerequisite. A repository's source-controlled setup declaration does not itself authorize privileged installation. Agents use only their assigned checkouts and service access; they do not receive shared engine administration or Project root access.

Installed tools and shared files persist across terminal End, Project Stop and Start. Starting a Project reuses its existing root; it does not update old Project installations from a newer image. Use the explicit [Project maintenance](../70-Operate/30-project-maintenance.md) procedure for supported same-root tool refresh, and the [backup guide](../70-Operate/60-backups-and-restoration.md) to protect shared persistent state.

For Forgejo and GitHub command-line clients, authenticate each CLI with your own account and keep credentials out of shared files; native Git authentication is covered in [Repository collaboration](10-collaboration.md). For databases and nested application containers, use [Project services](40-project-services.md).
