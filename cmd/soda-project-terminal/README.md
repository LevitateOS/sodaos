# Soda project terminal helpers

This package installs three fixed helpers inside Project OS:
`project-terminal`, `project-account` and `project-factory-roles`, under
`/usr/libexec/soda/`. Soda's host services invoke them for authorized operations;
they are not interactive login shells.

## Normal project use

Join a project through Spaces to obtain your native account, then open its
managed browser terminal or connect through ordinary SSH. The browser can
reattach a surviving managed terminal. End stops the processes supervised for
that terminal; Hide only changes presentation. Personal files remain in the
project. See [working inside a project](../../docs/guides/develop.md).

The helpers expect the packaged Project OS, managed account records, native
Linux accounts, systemd, tmux and their prepared runtime directories. Privileged
operations run through Soda's project-root path. They use fixed paths and
validated identity bindings rather than arbitrary caller-selected programs.

## Installed interfaces

| Binary | Inputs and behavior |
| --- | --- |
| `project-terminal` | Manages reservations, tmux sessions, PTY attachment, terminal metadata, subscription process hooks and managed SSH keys. |
| `project-account` | Reads one JSON request on stdin to provision or confirm a member account; `--status` selects read-only administrator observation. |
| `project-factory-roles` | Reads one JSON request on stdin to manage the bounded factory roles, approved preparation inputs and supervised work. |

`project-terminal` accepts `prepare ID`, `broker`, `keys`, or the fixed control
argument shape below. Control actions are `reserve`, `create`, `attach`,
`inspect`, `list`, `end` and `rename`:

```text
project-terminal ACTION ID LOGIN IDENTITY COLS ROWS SECONDS SOURCE_HASH NAME SCOPE
```

Soda supplies the identity, reservation, source hash and scope. Control calls
emit JSON metadata; attachment exchanges terminal frames. `broker` consumes a
bounded JSON subscription request; `keys` consumes a revision-checked key request
and returns its current revision and public keys. Key changes affect only the
Soda-managed authorized-keys file for that bound account.

`project-account` with no args accepts `login`, `identity`, `admin` and `keys`.
Provisioning creates or confirms the account, home, managed identity marker and
SSH key file; it refuses unrelated existing-account collisions. Its JSON reply
confirms `login` and `identity`. A service operator can inspect an already
provisioned account without provisioning it:

```sh
printf '%s\n' '{"login":"soda-tester","identity":42}' |
  sudo /usr/libexec/soda/project-account --status
```

Replace the example values with that account's recorded binding. The JSON reply
includes `administrator`. Unknown arguments refuse before provisioning effects.

`project-factory-roles` selects `ensure`, `approve`, `record`, `start`, `inspect`,
`stop`, `hold` or `release` through its JSON `op` field. It keeps managed state
under `/var/lib/soda/factory` and accepts bounded approved inputs. Use the
[factory operator interface](../../docs/reference/factory.md) to admit and
control factory work; creating a personal terminal does not grant that authority.

Helper failures return a nonzero status with a fixed failure response. Inspect
native account and service state when confirmation fails; a failed response is
not evidence that earlier effects did not occur.

Canonical contracts: [managed terminals](../../docs/reference/terminal.md),
[Project OS accounts](../../docs/reference/project-os.md#ownership-and-trust),
and [credentials](../../docs/reference/credentials.md).
