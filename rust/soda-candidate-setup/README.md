# soda-candidate-setup

Rust port of `scripts/setup-soda-candidate.sh` (plus the
`scripts/candidate-storage.sh` defaults it sourced). Prepares this machine so
`sudo soda-candidate` runs without a coding agent. Development and fixture
scope only: never creates qualification or signing configs.

Messages, exit codes, installed paths, file modes, and generated file bytes
match the shell exactly. Like the script, the binary takes no arguments and
ignores any it is given; it is driven by the working directory (the
repository root) and the same environment variables.

## Run

From the repository root:

```sh
cargo run -p soda-candidate-setup
```

Environment (same as the script):

- `SODA_FORGEJO_SOURCE` (required): clean canonical Forgejo fork checkout.
- `SODA_REPOSITORY_PREFIX` (default `ghcr.io/levitateos/sodaos`).
- `SODA_REFRESH_AUTHORITY` (`1` regenerates the fixture media authority).
- `SODA_CANDIDATE_ROOT/HOME/RUN/SCRATCH` (development/test overrides).

## Tests

```sh
cargo test -p soda-candidate-setup
```

Unit tests pin the JSON bytes, parsers, and helpers; integration tests
exercise every failure path up to (not including) the first privileged
mutation, so the suite touches no services, credentials, or host state.
