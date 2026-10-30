# Develop Soda Forge on soda-metal

From the local `sodaos` checkout, deploy the working tree to the retained
development appliance reached by `ssh soda`:

```sh
cd /home/vince/Projects/sodaos
bun run dev:deploy
bun run dev:status
```

For post layout, styling and other extension browser changes, use the fast loop:

```sh
bun run dev:ui       # build and upload browser files once
bun run dev:watch    # update automatically after browser-source edits
```

Refresh the app after the command reports the update ready. These commands need
an active development deployment. They build only the extension's JavaScript and
CSS and copy its browser assets; they do not compile Go or Forgejo, install a
package, run a database migration, or restart either service. Existing pages can
still load their older hashed chunks while a refreshed page loads the new build.

The update is an explicit development override of the installed Soda package's
browser files. Its previous assets and checksum inventory are retained beneath
`/var/home/soda-dev/ui/`, and `dev:status` reports the browser release separately.
The installed backend, manifest and package receipt stay unchanged. A full
deployment or disabling development mode restores the catalog-selected package.
Changes to the extension manifest, APIs, permissions, backend, native templates
or host styles need the normal full deployment. A manifest mismatch makes the
browser-only command stop with that explanation.

For those full changes, run the relevant [focused checks](testing.md#source-checks),
then `bun run dev:deploy` and refresh the browser. The full deploy builds
Forge presentation templates, CSS and browser modules, the Soda extension
assets/backend, and the Go dashboard. It transfers the prepared files over SSH
and restarts Forge and the dashboard to activate them.

The local machine needs the repository's Go, Rust and Bun toolchains,
and `bun install --frozen-lockfile`. The build uses the matching cached SDK at
`WORK/inputs/fountain-source-REVISION/sdk`, where `REVISION` is the first seven
characters of the source pin in `fountain-source.json`. Otherwise it uses the
sibling `forgejo-ext` SDK checkout selected by `go.mod`; `--sdk-dir PATH`
selects an explicit SDK. Successful full deployments remember the actual host
executable and SDK per SSH host in
`WORK/deploy-inputs-HOST.json`; explicit command-line options override those
selections. This keeps subsequent deployments on the working custom host rather
than silently returning to the cached release executable or SDK. These inputs
are already prepared on this machine.
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

Only full deployments restart Forge and the dashboard. The appliance keeps its retained data,
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

Repository routes use one tree rooted at `/{owner}/{repo}/feed`, with
`/requests/{request}`, `/app`, `/knowledge`, `/terminals`,
`/settings/{section}` and `/issues[/{issue}]` beneath it. Links, native
templates, manifest and host routing use this tree directly. There are no
repository `/workspace` routes or redirects and no alternate route tracks.

Deployments reuse the remembered compatible host executable. Supply an
explicit binary when testing changes to Fountain itself:

```sh
cargo run --release --locked -p soda-release-assets --bin soda-dev -- deploy \
  --fountain-bin /absolute/path/to/fountain
```

Request intake consumes the session-scoped `repository.issues` read and
request-scoped native issue-intake authorization. The extension explicitly
declares `native.repository.issues.intake`; it does not enroll a Factory bot.
Until these capabilities ship in the pinned Fountain release, deploy with both
`--fountain-bin` pointing to the updated host build and `--sdk-dir` pointing to
the matching SDK in the sibling `forgejo-ext` checkout. The cached pinned host
and SDK do not provide this callback.

Once a full deployment succeeds with these explicit paths, later deployments
reuse the remembered paths without repeating the flags. Browser-only updates do
not need a host executable or SDK at all.

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
