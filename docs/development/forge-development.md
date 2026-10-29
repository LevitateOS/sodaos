# Develop Soda Forge on soda-metal

From the local `sodaos` checkout, deploy the working tree to the retained
development appliance reached by `ssh soda`:

```sh
cd /home/vince/Projects/sodaos
bun run dev:deploy
bun run dev:status
```

Edit the source, run the relevant [focused checks](testing.md#source-checks),
then run `bun run dev:deploy` again and refresh the browser. The deploy builds
Forge presentation templates, CSS and browser modules, the Soda extension
assets/backend, and the Go dashboard. It transfers the prepared files over SSH
and restarts Forge and the dashboard to activate them.

The local machine needs the repository's Go, Rust and Bun toolchains,
and `bun install --frozen-lockfile`. The build uses the matching cached SDK at
`WORK/inputs/fountain-source-REVISION/sdk`, where `REVISION` is the first seven
characters of the source pin in `fountain-source.json`. Otherwise it uses the
sibling `forgejo-ext` SDK checkout selected by `go.mod`; `--sdk-dir PATH`
selects an explicit SDK. These inputs are already prepared on this machine.
The SSH account needs root access to the retained appliance.
The command explicitly reads `~/.ssh/config`; `--ssh-config PATH` selects a
different client configuration.

The default SSH host is `soda`. To select another configured development host:

```sh
cargo run --release --locked -p soda-release-assets --bin soda-dev -- deploy --host HOST
```

## What changes on the appliance

The command adds development Quadlet overrides that mount the prepared Forge
presentation and dashboard executable read-only. It packs the extension as a
normal USTAR package, seeds the SHA256-addressed package cache, and selects a
development catalog. Fountain installs the package through its existing
extension manager, preserving package identity and requested state.

The first deployment copies the retained Soda application database into
`soda_dev` and upgrades that copy for the local source. Later deployments keep
using the same development database, including changes made while testing.
The initial `soda-metal` setup copied schema 67 into a development database at
schema 69; the native database stays at 67. Forge accounts and Git repositories
are shared with the installed Forge and remain visible in either mode.

Only Forge and the dashboard restart. The appliance keeps its retained data,
configuration and Project containers. This development loop does not perform
a `bootc` update, rebuild installation media, or publish a release.

Builds and reusable inputs live beside the checkout under `.soda-dev` by
default (`/home/vince/Projects/.soda-dev` here); `--work-dir PATH` selects
another work directory. The deployed working tree may include uncommitted
changes. To build a local deployment context without uploading or restarting:

```sh
cargo run --release --locked -p soda-release-assets --bin soda-dev -- build
```

## Fountain changes and repository routes

The default asset/manifest track is `1`, matching the published Fountain
contract. Deployments reuse the compatible cached executable at
`WORK/inputs/fountain` when present. An explicit binary can be supplied when
testing changes to Fountain itself:

```sh
cargo run --release --locked -p soda-release-assets --bin soda-dev -- deploy \
  --fountain-bin /absolute/path/to/fountain
```

Use `--track 2` only with a Fountain build that supports the named repository
routes in `system/containers/extension/extension.track2.json`. Selecting a
track does not build Fountain.

## Return to the installed version

```sh
bun run dev:disable
bun run dev:status
```

Disabling removes the development overrides and restarts Forge and the
dashboard using their installed configuration and original Soda application
database. The `soda_dev` copy remains available for the next development
deployment.

The [older local VM recipe](frontend-vm-update.md) documents a previous
package-in-image installation. Use this workflow for the current native
extension-catalog appliance.
