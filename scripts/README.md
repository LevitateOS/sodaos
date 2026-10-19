# PostgreSQL backup integration checks

The `scripts` Go package checks PostgreSQL backup round-trip and child SQL
delivery against disposable databases. Repository developers use it while
changing backup/restore behavior. It is a test package, with no installed
executable or application API.

## Run the checks

From the repository root, with the Go toolchain selected in
[go.mod](../go.mod):

```sh
mkdir -p "$PWD/.artifacts/tmp"
TMPDIR="$PWD/.artifacts/tmp" go test -mod=readonly ./scripts
```

The tests need Podman and the repository's Rust toolchain: they build the
`soda-pg-fixture` helper through Cargo and run real PostgreSQL containers.
They skip when the container engine is unavailable. Rust outputs honor
`CARGO_TARGET_DIR` where the helper supports it.

`integration_helpers_test.go` supplies the shared Cargo build helper. Tests
use private temporary fixtures; a passing check covers its assertions rather
than an installed-host backup schedule.

See [host database services](../system/host/services/README.md#backup-and-restore)
for the appliance backup contracts, [testing](../docs/development/testing.md)
for validation layers, and [Go package conventions](../docs/development/go-packages.md)
for package ownership.
