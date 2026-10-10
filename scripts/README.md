# Go source and presentation checks

The `scripts` Go package checks Forgejo template composition, branding, service
wiring and cross-language wire contracts. Repository developers use it while
changing presentation assets or native helper interfaces. It is a test package,
with no installed executable or application API.

## Run focused checks

From the repository root, with the Go toolchain selected in
[go.mod](../go.mod):

```sh
mkdir -p "$PWD/.artifacts/tmp"
TMPDIR="$PWD/.artifacts/tmp" go test -mod=readonly ./scripts -run '^TestStrictjsonWire'
TMPDIR="$PWD/.artifacts/tmp" go test -mod=readonly ./scripts -run '^TestForgejoPageIntroComposition$'
```

Use `go test -mod=readonly ./scripts` for the whole package. Some tests compile
and invoke Rust or Go helpers, so the full suite needs Cargo, the repository's
Rust toolchain and writable build/cache locations on the workspace disk. The
module's local Forgejo SDK replacement must be available as configured in
`go.mod`. Rust outputs honor `CARGO_TARGET_DIR` where the helper supports it.

Tests use checked-in templates, assets and wire vectors and may create private
temporary fixtures or local sockets. Template fixtures model selected native
contexts; a passing source check covers its assertions rather than an installed
browser journey.

## Optional checks and previews

- The `branding` build tag enables native SVG renderer checks requiring
  `rsvg-convert`; use `go test -tags branding ./scripts -run '^TestForgejoBrandingMatchesSVGMaster$'`.
- `SODA_FORGEJO_TEMPLATES` points native-template comparison tests at the exact
  retained template export they require; those tests can skip when it is absent.
- `SODA_FORGEJO_GALLERY=1` enables the presentation gallery test, which also
  needs the prepared native asset server at `http://localhost:3300` and writes
  local preview output.
- `SODA_HOME_PREVIEW` selects optional front-page preview output.

Choose those opt-ins only when their fixtures and tools are prepared. Read the
individual test for its exact output location and setup. A skipped optional
check is not coverage of that integration.

See [Forgejo](../docs/reference/forgejo.md) and
[frontend presentation](../frontend/forgejo/README.md) for the owning UI guidance,
[testing](../docs/development/testing.md) for validation layers, and
[Go package conventions](../docs/development/go-packages.md) for package ownership.
