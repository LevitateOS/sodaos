# soda-muse

`soda-muse` is Soda's project-local launcher for the native Muse CLI. In a
packaged Project OS it is installed as `/usr/local/bin/muse`, so developers use
the normal `muse` command from SSH or a managed browser terminal.

## Before using it

Join the project to obtain a provisioned account, and enroll an authorized Muse
connection through Soda. The project needs the packaged native Muse executable
and a working launch interface at `/run/soda-muse-interface/launch.sock`.

From your provisioned account and desired working directory:

```sh
muse
muse exec
```

Normal arguments are passed to the native CLI. Its commands and help describe
the available Muse operations. If several authorized connections exist, select
one using the actual connection ID:

```sh
SODA_MUSE_CONNECTION=CONNECTION_ID muse
```

The launcher forwards the current directory, terminal dimensions, `HOME`,
`XDG_CONFIG_HOME` and arguments. When `XDG_CONFIG_HOME` is unset, it uses
`$HOME/.config`; `HOME` must then be defined.

## Runtime behavior

The launch service derives your project and identity from the current account
and kernel peer evidence. It creates a private invocation view and supervised
execution. Personal settings seed that view; edits in the temporary view are
discarded at retirement. Change your personal settings for lasting preferences.
Inherited API-key overrides are removed for subscription execution.

The launcher relays terminal input/output and returns the launched command's
exit status. `muse launch service unavailable` means the project launch interface
could not be reached. A rejected upstream credential requires reconnection
through Soda.

## Service helper modes

The same binary provides fixed helper modes used by Soda: `--soda-check VERSION`,
`--soda-read-config PATH`, `--soda-copy-config SOURCE DESTINATION`,
`--soda-account IDENTITY`, and `--soda-exec ROOT CWD [ARGS...]`. These support
runtime checks, private config views, account lookup and native execution; normal
developers use `muse` without these helper switches.

See [normal Muse access](../../docs/reference/credentials.md#normal-muse-command),
[Project OS](../../docs/reference/project-os.md#muse-subscription-access), and
[working inside a project](../../docs/guides/develop.md).
