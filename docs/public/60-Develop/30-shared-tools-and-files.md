# Use shared Project tools and files

Projects keep shared tool installations and team files across personal checkouts and Factory attempts. Administrators maintain shared tools; members use them from their own Project accounts.

## Use shared and personal tools

The Rocky headless Project includes mise and common development tools. Shared mise installations and global configuration live in `/opt/mise` and `/etc/mise/config.toml`. Administrators install shared packages or mise-managed versions through native Project administration and trusted repository tool declarations.

Members can use installed tools through the Project's mise setup and repository-specific configuration after reviewing and trusting it. Personal clones, preferences, and overrides belong in each member's home, not in the shared installation.

## Share files with the Project

Shared files live under `/srv/project/shared` and are available through `~/shared` for joined users. Use these locations when a file is deliberately shared with the Project team. Keep personal work, credentials, and private Git clones in your own home.

Shared directories are writable collaboration space. Do not use them as the trusted source for Factory setup approval or store provider tokens there.

## Coordinate shared changes

Project-wide package and tool changes affect other people and Factory readiness. Coordinate them with the Project administrator and hold affected Factory work before changing an accepted setup input or shared prerequisite. A source-controlled repository declaration does not itself authorize privileged installation.

Agents use only their assigned checkouts and service access; they do not receive shared engine administration or Project root access. For Forgejo and GitHub CLI authentication, use your own account and keep credentials out of shared files. Native Git authentication is covered in [Repository collaboration](10-collaboration.md).

## Preserve and refresh the Project state

Installed tools and shared files persist across terminal End, Project Stop, and Start. Starting a Project reuses its existing root; it does not update old Project installations from a newer image. Use [Project maintenance](../70-Operate/30-project-maintenance.md) for supported same-root tool refresh and [Backups and restoration](../70-Operate/60-backups-and-restoration.md) to protect persistent state.

For databases and nested application containers, see [Project services](40-project-services.md).
