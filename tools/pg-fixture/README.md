# soda-pg-fixture

Start a disposable PostgreSQL 17 container for Soda and Forgejo development
tests. Each run gets random passwords, a loopback-only port and separate roles
and databases. This tool is for test authors who need a local database fixture.

## Start and stop a fixture

Run from the repository root on Linux with Podman available. The tool pulls the
pinned PostgreSQL image if it is missing. Keep fixture secrets in a private
directory on the workspace disk:

```sh
export SODA_PG_FIXTURE_DIR="$PWD/.artifacts/pg-fixture"
fixture_env=$(cargo run -q -p soda-pg-fixture -- start) && eval "$fixture_env"
# Run your tests using the returned connection values and password files.
cargo run -q -p soda-pg-fixture -- stop "$SODA_PG_CONTAINER"
```

Use a fresh fixture directory for each concurrent run. An existing directory
must belong to the current user, have mode `0700` and have no symlink components.
Proceed with tests only if `start` succeeds. Export the returned variables if
your test process reads them from its environment.

## Connection values

Successful startup prints shell assignments; it does not print passwords.

| Variable | Meaning |
| --- | --- |
| `SODA_PG_CONTAINER` | Container name to pass to `stop`. |
| `SODA_PG_HOST` / `SODA_PG_PORT` | `127.0.0.1` and the allocated host port. |
| `SODA_PG_DATABASES` | Created database names; defaults to `forgejo soda`. Each database has a matching login role. |
| `SODA_PG_DIR` | Secret directory containing `<role>.passwd` files with mode `0600`. |
| `SODA_PG_SUPER_PASSWORD_FILE` | Path to the `postgres` password file. |

Set `SODA_PG_DATABASES` before startup to choose a space-separated list. Names
accept lowercase letters, digits and underscores, but cannot start with `pg_`.
An explicitly empty list creates only the superuser cluster.

`stop` force-removes the named fixture container and its anonymous volumes.
Password files remain in the caller-owned directory; remove that run's private
files after the container stops. Exit `3` means Podman or the pinned image is
unavailable and lets a test driver report a skipped fixture.

See [testing](../../docs/development/testing.md) for validation scope and
[local testing](../../docs/guides/local-testing.md) for retained lab access.
