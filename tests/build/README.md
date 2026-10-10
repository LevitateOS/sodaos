# Build and packaging checks

This Go package checks the repository's appliance packaging, Forgejo payload,
project foundation, native helper interfaces and source-check dispatch. Its
users are repository developers validating build contracts. It has no installed
command or runtime service.

## Run selected checks

Run from the repository root with the Go toolchain in
[go.mod](../../go.mod). Keep test temporary files on the workspace disk:

```sh
mkdir -p "$PWD/.artifacts/tmp"
TMPDIR="$PWD/.artifacts/tmp" go test -mod=readonly ./tests/build \
  -run '^TestForgejoPayload'
TMPDIR="$PWD/.artifacts/tmp" go test -mod=readonly ./tests/build \
  -run '^TestTerminalAssets'
```

To run the full package, use `go test -mod=readonly ./tests/build`. Tests that
exercise native helpers build real Rust binaries through Cargo, so prepare the
pinned Rust toolchain and dependencies too. Set `CARGO_TARGET_DIR` to an existing
workspace build location if needed; the shared helper reuses builds by package,
flags and target directory. The configured local Forgejo SDK replacement must
be present for module resolution.

## What is checked

- Source manifests, pinned asset declarations, template closure and notices.
- Project image recipes, helper staging and installed-unit wiring.
- Native account, factory-role, domain, provisioning and activation helper
  behavior against test-owned files and subprocess fixtures.
- Source-check ordering, failure propagation and bounded helper diagnostics.

[helpers.go](helpers.go) supplies repository-relative file readers, assertions,
subprocess results, temporary files and the `CargoBinary` build helper. Tests
share this harness rather than an appliance installation.

Some checks execute shell fragments, fake native commands or test-owned local
services. `SODA_CADDY_BINARY` opts avatar routing checks into an actual selected
Caddy executable and local listeners; without it those cases skip. Source,
fixture and compiled-helper results do not establish installed-host or browser
acceptance. Keep optional targets and outputs scoped to the chosen check.

See [testing](../../docs/development/testing.md),
[native support](../../docs/development/native-support.md) and
[release workflow](../../docs/development/release.md) for the corresponding
native build and installed validation paths.
