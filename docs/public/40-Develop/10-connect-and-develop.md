# Connect and develop

Use your preferred SSH editor, terminal or browser workspace terminal with ordinary personal Git checkouts inside a shared project.

## Connect to your workspace

Complete [joining and host-key verification](../20-Deploy/30-first-connection.md).
Use the environment's displayed account and IP:

```sh
ssh PROJECT_USER@PROJECT_IP
```

Your home belongs to you in this project. Clone repositories there, for example
`~/repo-name`. `~/shared` leads to the project's shared files, not an automatically
synchronized checkout. You can also select this project in the dashboard and
open its terminal as your existing project-local user. Opening a terminal does
not create a new account or grant host administration.

## Open an editor

For VS Code, install **Remote - SSH**, connect to the same account/IP and open
your checkout. Follow Microsoft's [Remote SSH guide](https://code.visualstudio.com/docs/remote/ssh)
for remote extensions and forwarded ports. Other SSH-capable editors use the
same connection; terminal editors run directly in your project session.

An optional client-side `~/.ssh/config` entry makes key selection explicit:

```sshconfig
Host soda-example
    HostName PROJECT_IP
    User PROJECT_USER
    IdentityFile ~/.ssh/YOUR_PERSONAL_PRIVATE_KEY
    IdentitiesOnly yes
```

Replace the values with your actual details. `soda-example` is your local client
alias, not a project hostname provided by Soda. Keep editor servers, extensions
and personal credentials in your own home.

## Use ordinary Git

Configure your own [Git authentication](../30-Use-Soda/30-forgejo.md#register-your-git-key)
and copy the repository's actual clone URL:

```sh
git clone CLONE_URL ~/repo-name
cd ~/repo-name
git status
git switch -c my-change
```

Set your author name/email, edit, commit and push through the repository's normal
workflow. Review changes with teammates in [pull requests](../30-Use-Soda/35-collaboration.md).
Joining the environment does not create a checkout or grant Git permissions.

Keep passphrase-protected Git keys and agents personal. After a restart, start
or reconnect to your own agent and unlock your key again when needed. Soda does
not forward the client's agent or copy browser credentials automatically.

## Tools and services

Use the [shared installed toolchain](20-shared-tools-and-files.md) by default.
Read a repository's configuration before trusting or executing it; ask the
project administrator to install a missing shared tool version. Do not make each
member independently download the same tool into the shared tree.

Use project databases and services at their normal addresses and ports. The
project administrator controls the shared container engine; members consume
its endpoints. [Project services](30-project-services.md) explains native
Compose, personal processes, ports and data.

## Authenticate personal CLIs and assistants

Tea and GitHub CLI are available in the project. Authenticate separately in a
private interactive terminal, using your own account:

```sh
tea logins add
tea whoami
gh auth login --git-protocol ssh
gh auth status
```

For Tea, use the configured Forgejo HTTPS origin rather than a public-service
default. Follow [Tea guidance](https://docs.codeberg.org/git/clone-commit-via-cli/)
and the [GitHub CLI manual](https://cli.github.com/manual/gh_auth_login).
Verify any proposed Git-key registration before approving it. Have the project
administrator install a needed private CA through native trust configuration;
do not disable TLS checks.

Authenticate assistants and other tools with your own credentials too. Keep
credential files out of Git and `~/shared`, with restrictive native permissions.
Browser, Git transport, Tea, GitHub CLI and assistant sessions are separate.
Use their native logout/revocation procedures when ending access.

## Finish a session without losing work

Save editor buffers and push commits you want on the Git host. Preserve untracked
files, local-only branches and database data independently. Disconnecting SSH or
signing out of the browser is not a promise that every task survives; use native
session/process tools and coordinate long-running services with the team.

A merge does not restart services, install tools, migrate a database or remove
temporary files. Perform those actions deliberately through their native tools.
