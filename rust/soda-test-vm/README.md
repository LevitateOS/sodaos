# soda-test-vm

Rust port of `scripts/test-vm.sh`: manage the isolated x86_64 test VM
prepared under `.artifacts/test-vm`. Never installs on the builder, deletes
a disk, or changes host networking.

Messages, exit codes, the QEMU command line, ssh/tail argv, lock handling,
and pidfile checks match the shell exactly, with three deliberate deltas:
the tool operates relative to the current directory (invoke it from the
repository root) instead of cd-ing to the script's own location, whose
cd/dirname failure modes are not modeled; the usage
and started lines print the working `cargo run` invocation instead of the
deleted script path; and `exec` uses execvp PATH search, which skips a
broken shadow entry where bash would stop and fail. A non-EPIPE stdout
failure (ENOSPC) panics instead of exiting silently like `echo`.
Closed-pipe death (141, silent) matches the shell.

The operator key and known-hosts file travel as paths (`-i`,
`UserKnownHostsFile`), never content; no secret bytes are read or printed.

## Run

```sh
cargo run -p soda-test-vm -- [start|status|ssh [COMMAND...]|tunnel|web-tunnel|console]
```

## Tests

```sh
cargo test -p soda-test-vm
```

Unit tests pin pidfile semantics and argv construction; integration tests
cover usage, status shapes, refusal paths, the QEMU command line, and the
exec argv/failure shapes with fake QEMU/ssh/tail. No real VM is ever
started. Tests past the KVM gate skip cleanly where `/dev/kvm` is not
accessible.
