# Shared tools and files

Install a tool once for the project and share files deliberately while keeping personal checkouts and credentials in your own home.

## Know the shared paths

| Path | Purpose | Who changes it |
| --- | --- | --- |
| `/opt/mise` | Shared installed tools, data and shims | Project administrator |
| `/etc/mise/config.toml` | Shared global tool-version configuration | Project administrator |
| `/srv/project/shared` | Project-shared files | Members under native file permissions |
| `~/shared` | Convenient link to that same shared directory | Its contents follow shared permissions |
| Your project home | Personal checkouts, editor state and credentials | You, subject to project administrator authority |

Sharing applies inside one project environment, not across every environment on
the appliance. An administrator inside the project can administer its filesystem;
a home is not a confidentiality boundary against that administrator.

## Install a shared tool

As the associated repository's owner, connect to the project—not the host—and
use project-local administration. This example installs Node 24 for the project:

```sh
sudo -i
umask 022
mise use --global node@24
exit
```

Review the selected version and installation source before running it. Native
mise settings direct the installation to `/opt/mise` and global configuration to
`/etc/mise/config.toml`. The administrator can use Rocky's native package tools
for project system packages; do not install developer packages on the appliance
host instead.

Both administrator and member can inspect the installation:

```sh
mise where node
node --version
```

They should use the same path under the shared installation. Equal version
strings alone do not establish shared files. Ordinary members use these tools
without permission to overwrite the root-owned installation.

## Repository-specific versions

A repository may carry ordinary `mise.toml` configuration. Read it before:

```sh
mise trust
mise exec -- TOOL_COMMAND
```

Replace `TOOL_COMMAND` with the intended command. A tool version selected by the
repository must exist in the installation you use. Ask the administrator to
install missing shared versions; do not repeatedly run `mise install` as a member
against an unwritable tree. Use [mise's native documentation](https://mise.jdx.dev/)
for configuration, tasks, environment variables and supported backends.

Personal installations can coexist in personal paths using native tool settings,
including explicit mise directory/configuration overrides. Keep those changes
personal and intentional; Soda does not clone toolchains, supply shared/private
selectors or promote installed binaries when Git branches merge.

## Share files intentionally

Place team-owned material in `~/shared` and agree on its names and edit ownership.
For a new file intended to be group-writable, inspect its group and permissions;
an application may create files with stricter permissions than you expect.
Native `ls -l`, `id` and narrowly scoped permission changes help diagnose this.
Do not use recursive world-writable permissions as a sharing shortcut.

Do not put private keys, tokens or personal CLI configuration in shared files.
Keep source checkouts personal unless the team intentionally wants a shared
working tree and understands concurrent edits. Git reviews coordinate source
changes; sharing a filesystem is not a merge strategy.

## Change tools without surprising teammates

Keep required versions installed while other checkouts use them. Coordinate
global-default changes, package updates and removals. Open a new shell or restart
the affected process when its environment needs refreshing; existing shells do
not receive a live configuration update from Soda.

Normal project startup preserves the installation and files. Protect important
shared material through [backup planning](../50-Operate/30-backups-and-restoration.md),
not assumptions that a container image contains your later changes.
