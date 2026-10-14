# soda-project-maintain

`soda-project-maintain` provides explicit `muse` and `terminal-support` operations.
Muse installs packaged public tools and restores the launch interface.
Terminal support repairs one admitted helper in an idle retained root.
It operates on the project's retained container and requires appliance root.

The [Project OS maintenance contract](../../docs/reference/project-os.md#same-root-maintenance)
owns admission, supported roots, effects and interrupted recovery. Terminal repair:

```sh
sudo /usr/libexec/soda/soda-project-maintain terminal-support --project p0123456789abcdef01234567 --check
sudo /usr/libexec/soda/soda-project-maintain terminal-support --project p0123456789abcdef01234567
```

`--development-installed-host` explicitly selects the already installed host helper
for an unqualified development repair. Normal repair consumes the complete installed
release's bundled input and metadata. Neither operation upgrades the host.

Terminal repair includes running and stopped Rocky 10.2 roots with empty managed
terminal/factory state. A stopped Project remains stopped: native namespace access
uses its exact retained root and existing UID/GID mappings. The compatibility probe
uses temporary private runtime mounts, without starting init or user workloads.
Container identity and runtime state are checked around the operation; the exact
root is unmounted afterward. Unknown state, active work or uncertain cleanup blocks
completion. The same container, accounts, keys, files and volumes are preserved.

## Normal use

Choose the exact project ID from Soda, then run on the appliance:

```sh
sudo /usr/libexec/soda/soda-project-maintain \
  muse --project p0123456789abcdef01234567
```

The ID has the form `p` followed by 24 lowercase hexadecimal characters.
The Muse operation checks the matching project container's identity and waits briefly
for it to be running. It does not start a stopped project.

| Option | Meaning |
| --- | --- |
| `--project ID` | Required exact project identity. |
| `--config PATH` | Absolute host configuration; defaults to `/etc/soda/host.json`. |
| `--tools PATH` | Absolute public tool directory; defaults to `/usr/share/soda/muse-tools`. |
| `--bind-only` | Restore the launch interface without installing tools or the system bus prerequisite. |

The project startup service uses `muse --bind-only --project ID` to restore the
interface after a restart. Full maintenance is an explicit operator action.

## Inputs and effects

Maintenance reads host configuration and `/usr/share/soda/release.json`, validates
the configured Muse release and public tools, and uses native Podman access.
Full maintenance installs `dbus-broker` if missing, starts `dbus.socket`, and
replaces these project files with the admitted public executables:

- `/usr/local/bin/muse`
- `/usr/local/bin/soda-identity-compose`
- `/usr/local/libexec/soda/muse`

It attaches the dedicated public launch directory at
`/run/soda-muse-interface` with restricted mount permissions. Accounts, checkouts,
volumes and unrelated installed tools remain in the same project root.
Credentials are not included in the public tool payload.

Returns `0` on completion and `1` with a prefixed stderr diagnostic on failure.
When Muse is disabled, `--bind-only` returns successfully without attaching it;
full maintenance reports that the runtime is disabled.

See [Project OS maintenance](../../docs/reference/project-os.md#same-root-maintenance),
[Muse subscription access](../../docs/reference/project-os.md#muse-subscription-access),
and [credentials](../../docs/reference/credentials.md#normal-muse-command).
