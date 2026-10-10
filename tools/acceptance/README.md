# Soda acceptance support

Run bounded native checks and retain evidence tied to a source revision and
target. This package is for operators and release developers collecting
observations from an explicitly selected builder, host or fresh VM.

## Commands

| Binary / action | Use |
| --- | --- |
| `soda-acceptance exec` | Run an existing check locally or through pinned SSH. |
| `soda-acceptance native` | Run an exact-source remote build phase with owner `P02`, `--remote` and `--request`. |
| `soda-acceptance vm` | Boot a fresh KVM fixture with owner `P03` and `--config`; optional `--hold` and `--restart` control its lifecycle. |
| `soda-acceptance probe-ssh` | Observe a pinned Git SSH endpoint with owner `P11` and `--remote`. |
| `soda-acceptance report` | Combine finalized observations into a Markdown handoff. |
| `soda-acceptance-remote` | Target-side dispatcher for `native-phase REQUEST`, `cockpit-account` and `project-state`. |
| `soda-host-probes` | Helpers used by the installed host/operator shell checks. |

From the repository root, inspect the available flags:

```sh
cargo run -p soda-acceptance --bin soda-acceptance -- --help
cargo run -p soda-acceptance --bin soda-acceptance -- exec --help
```

## Collect an observation

Execution actions require an admitted owner, full commit revision, `x86_64`
architecture, non-secret target name and a fresh private evidence directory.
For example, with an already prepared pinned SSH connection file:

```sh
cargo run -p soda-acceptance --bin soda-acceptance -- exec \
  --owner P06 --revision FULL_40_CHARACTER_COMMIT_SHA --arch x86_64 \
  --target ACTUAL_GUEST_HOSTNAME \
  --remote /home/soda-builder/private/guest.json \
  --evidence /home/soda-builder/private/new-host-observation \
  -- env SODA_NATIVE_VALIDATE=ACTUAL_GUEST_HOSTNAME bash -s < tests/installed/host.sh
```

Replace placeholders with the selected revision and actual target. The remote
file references the SSH user, host, port, private key file and trusted
`known_hosts` file. Owned process execution requires Linux. VM execution also
requires native x86_64 Linux, KVM, matching firmware and verified boot inputs.

Add `--secret-file FILE` for known private values and `--artifact-file FILE` for
public artifact hashes. `--timeout` accepts positive durations up to 24 hours
and defaults to 30 minutes. Output includes captures and a finalized
`observation.json` when evidence finalization succeeds; execution, evidence and
cleanup outcomes are recorded separately. Failed work and VM disks are retained.

## Create a handoff

```sh
cargo run -p soda-acceptance --bin soda-acceptance -- report \
  --arch x86_64 --revision FULL_40_CHARACTER_COMMIT_SHA \
  --record /home/soda-builder/private/host-run/observation.json \
  --out /home/soda-builder/private/new-support-evidence.md
```

Repeat `--record` to include more observations. The report checks revision,
architecture and retained file hashes, and preserves failed or missing scopes.

The [native support guide](../../docs/development/native-support.md) owns request
formats, owner labels, trust, VM lifecycle and evidence rules. Use the
[testing guide](../../docs/development/testing.md) for the product acceptance
journeys invoked by these support tools. Builds, remote commands and VM actions
have real effects; select an authorized target and action before running them.
