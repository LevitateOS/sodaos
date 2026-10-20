# Use shared Project tools and files

Projects keep shared tool installations and team files across personal checkouts and Factory attempts. Administrators maintain shared tools; members use them from their own Project accounts.

## Use shared and personal tools

The Rocky headless Project includes mise and common development tools. Shared mise installations and global configuration live in `/opt/mise` and `/etc/mise/config.toml`. Administrators install shared packages or mise-managed versions through native Project administration and trusted repository tool declarations.

Members can use installed tools through the Project's mise setup and repository-specific configuration after reviewing and trusting it. Personal clones, preferences, and overrides belong in each member's home, not in the shared installation.

## Use a repository-pinned tool

For example, a repository can select an exact Node tool in its `mise.toml`:

```toml
[tools]
node = "22.14.0"
```

This is an illustrative repository requirement, not a promise that this Node release is the Project default. Use the repository's actual reviewed declaration and locks, and ask the administrator to confirm that the selected shared tool is installed before running it. Inspect the complete mise configuration, hooks, tasks and referenced files before trusting it; configuration can execute code. In your personal checkout, explicitly trust the reviewed file and use the selected tool:

```sh
mise trust mise.toml
mise exec -- node --version
mise exec -- node -e 'console.log("Project tool ready")'
```

For this declaration, the version output is `v22.14.0` and the smoke check prints “Project tool ready”. These checks establish tool selection and execution, not that the application passes its own checks. Follow [mise configuration](https://mise.jdx.dev/configuration.html), [trust](https://mise.jdx.dev/cli/trust.html) and [exec](https://mise.jdx.dev/cli/exec.html) for their exact behavior.

If the shared tool is missing or an administrator-owned dependency is required, give the Project administrator the reviewed declaration, exact prerequisite and failing command. The administrator reviews privileged effects and installs shared prerequisites through native Project administration. A trust decision in your checkout does not grant package installation, wheel or shared service authority. Recheck the repository's command after the prerequisite is available.

## Use personal forge command-line clients

The packaged tea client works with native Forgejo collaboration; gh works with GitHub. Authenticate each as yourself inside your Project account, using its supported login flow. Use tea's installed help for the native login and issue/pull-request commands; [GitHub CLI login](https://cli.github.com/manual/gh_auth_login) explains gh authentication.

After login, select the intended host and repository and read an existing issue or pull request before making writes. Confirm the displayed repository and acting account. CLI login is separate from Project SSH access and does not sponsor Factory execution. Follow the client's own credential-storage guidance; a tea or gh login does not by itself establish the Git authentication used in the [personal HTTPS contribution guide](30-collaboration.md#clone-and-authenticate).

## Share files with the Project

Shared files live under `/srv/project/shared` and are available through `~/shared` for joined users. Use these locations when a file is deliberately shared with the Project team. Keep personal work, credentials, and private Git clones in your own home.

Project root/wheel can inspect every member’s home, files and credentials. Personal storage separates ordinary peers; it does not hide secrets from trusted Project administrators. Review [ownership-transfer access](10-projects-and-workspaces.md#review-administrative-trust-and-ownership-transfer) when responsibility changes.

Shared directories are writable collaboration space. Do not use them as the trusted source for Factory setup approval or store provider tokens there.

Shared service access also needs its own data boundary. The [PostgreSQL development example](60-project-services.md#example-a-postgresql-development-service) gives a human member, coder and reviewer separate databases and credentials. Keep its administrator-owned Compose definition and password files outside `~/shared`; ordinary users connect through their assigned endpoint without administering the engine.

## Coordinate shared changes

Project-wide package and tool changes affect other people and Factory readiness. Coordinate them with the Project administrator and hold affected Factory work before changing an accepted setup input or shared prerequisite. A source-controlled repository declaration does not itself authorize privileged installation.

Agents use only their assigned checkouts and service access; they do not receive shared engine administration or Project root access. For Forgejo and GitHub CLI authentication, use your own account and keep credentials out of shared files. The [personal Git token procedure](30-collaboration.md#create-a-personal-https-token) explains the required repository scope and password-prompt entry without storing the token in Project files.

## Preserve and refresh the Project state

Installed tools and shared files persist across terminal End, Project Stop, and Start. Starting a Project reuses its existing root; it does not update old Project installations from a newer image. Use [Project maintenance](../50-Operate/60-project-maintenance.md) for supported same-root tool refresh and [Backups and restoration](../50-Operate/50-backups-and-restoration.md) to protect persistent state.

For databases and nested application containers, see [Project services](60-project-services.md).
