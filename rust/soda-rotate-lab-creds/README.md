# soda-rotate-lab-creds

Rust port of `scripts/ops/rotate-lab-creds.sh`: lab credential inventory
and owner-gated rotation (B7). Defaults to read-only inventory and prints
runbooks; only `--execute` with `SODA_ROTATE_ACK=<class>` mutates, and only
for fully scriptable classes. Secrets travel via 0600 files, never argv,
environment values in logs, or stdout.

Messages, exit codes, inspected paths, file modes, and generated file bytes
match the shell exactly, with three deliberate deltas: the
fixture-authority runbook prints the working `cargo run` invocation instead
of the deleted shell path; usage and `set -u` crash lines carry the new
argv0 instead of `$0 ... line N`; and a non-EPIPE stdout failure (ENOSPC)
panics instead of exiting silently like `echo`. Closed-pipe death (141,
silent) and staging cleanup on that path match the shell. Death by another
signal (TERM/INT) during rotation leaves the 0700 staging directory behind
while the script's EXIT trap removed it; there is no sound in-process fix,
so a killed rotation needs its staging reaped by the owner. The metadata
glob assumes C-locale collation order.

## Run

```sh
cargo run -p soda-rotate-lab-creds -- [inventory|--rotate CLASS [--execute]]
```

Classes: `fixture-authority` `cloudflared-token` `forgejo-runner`
`lab-vm-operator`. Only `fixture-authority` is scriptable; the rest print
owner-manual runbooks.

## Tests

```sh
cargo test -p soda-rotate-lab-creds
```

Unit tests pin the JSON bytes, inventory formatting, and helpers;
integration tests cover help, inventory structure, runbooks, and every
refusal path. Nothing in the suite mutates credentials or services.
