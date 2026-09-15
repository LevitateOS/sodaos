# Working inside a project

Everyday developer workflows inside a Soda project environment.

Runtime contracts: [Project OS](../reference/project-os.md).
Product model: [Projects](../product/projects.md).

## Join and connect

1. Open Spaces for the repository and Join the project when prompted.
2. Install any selected development-access public keys; never share private keys.
3. Connect with ordinary SSH as `user@project-ip`, or use the managed browser
   terminal for the same project-local account.

Host Tailnet enrollment alone does not prove project reachability.

## Editors and checkouts

Keep personal Git checkouts in your project home. Shared tools and files live under
the project shared locations (`/srv/project/shared`, `~/shared`). Prefer the project's
shared mise installations for language tools unless you intentionally configure a
personal toolchain.

## Git and collaboration

Use native Forgejo authentication (SSH keys or HTTPS tokens). Soda development-access
keys are for project SSH login; they are not Git authorization. Register outbound Git
keys through Forgejo's own settings when using SSH Git.

## Shared tools and services

Project administrators install shared packages and mise versions for everyone.
Ordinary members consume those installations. Nested databases and app containers
follow [Project services](project-services.md). Packaged `tea`/`gh` follow
[Project CLIs](project-clis.md).

## Profiles and desktop

Headless profiles are terminal-first. KDE profiles add an account-owned graphical
session to the same home and tools. Opening a desktop does not create another
environment or replace managed terminals.
