# soda-rotate-lab-creds

Inspect lab credential file metadata and print rotation runbooks. This tool is
for lab operators; it defaults to a read-only inventory. Only fixture media
authority rotation is automated.

## Inspect credentials

Run from the repository root with your normal operator shell environment:

```sh
cargo run -p soda-rotate-lab-creds -- inventory
cargo run -p soda-rotate-lab-creds -- --help
cargo run -p soda-rotate-lab-creds -- --rotate fixture-authority
```

The inventory prints paths, ownership, modes, sizes and timestamps, with
`PASS`, `WARN` or `SKIP` per inspected item. It checks fixture authority,
cloudflared, runner and VM operator-key paths and available build metadata;
it does not print credential contents. Missing or unreadable files appear as
skips. No arguments has the same effect as `inventory`.

## Choose a rotation class

| Class | Behavior |
| --- | --- |
| `fixture-authority` | Print a runbook, or regenerate fixture signing material with the execution options below. |
| `cloudflared-token` | Print the owner-manual tunnel credential runbook. |
| `forgejo-runner` | Print the owner-manual runner credential runbook. |
| `lab-vm-operator` | Print the owner-manual VM operator-key runbook. |

`--execute` does not automate the three manual classes.

After explicitly selecting fixture-authority rotation, choose an existing
private temporary directory on the workspace disk:

```sh
TMPDIR=/home/soda-builder/private/tmp SODA_ROTATE_ACK=fixture-authority \
  cargo run -p soda-rotate-lab-creds -- --rotate fixture-authority --execute
```

Execution needs Skopeo, `sudo` access and the existing candidate worker setup.
It replaces fixture authority material in `/var/lib/soda-candidate-authority`
and updates its worker binding. Existing signatures stop verifying, so affected
development attempts need setup again. `SODA_REPOSITORY_PREFIX` selects the
namespace, defaulting to `ghcr.io/levitateos/sodaos`.

Private staging files use restricted modes. Keep temporary staging on the
workspace disk by setting `TMPDIR` to an existing private directory there.
If rotation is interrupted, inspect any retained staging before cleaning up
that run's files. Fixture keys are separate from release signing keys.

See [candidate setup](../candidate-setup/README.md) for the worker setup and
[native support](../../docs/development/native-support.md) for authority and
release-delivery boundaries.
