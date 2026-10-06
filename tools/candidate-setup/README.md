# soda-candidate-setup

Rust port of `scripts/setup-soda-candidate.sh` (plus the
`scripts/candidate-storage.sh` defaults it sourced). Prepares this machine so
`sudo soda-candidate` runs without a coding agent. Development and fixture
scope only: never creates qualification or signing configs.

Messages, exit codes, installed paths, file modes, and generated file bytes
match the shell exactly. Like the script, the binary takes no arguments and
ignores any it is given; it is driven by the working directory (the
repository root) and the same environment variables.

Deliberate deltas: the tool does not `cd` to the script's location (run it
from the repository root); `worker.json` is delivered through `sudo tee`
instead of `sudo python3` (bytes identical, staging I/O failures surface as
setup errors instead of tracebacks); paths containing quotes or newlines
produce valid JSON where the shell's unquoted heredoc died with a Python
syntax error; death by signal during staging leaves the scratch staging
behind while the script's EXIT trap removed it; spawn diagnostics carry the
fixed tool prefix instead of `$0 ... line N`; and a non-EPIPE stdout
failure (ENOSPC) panics instead of aborting like the `echo` builtin.

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
