# Native build and installation

How to prepare builders, produce candidates, install components and activate an
appliance. Release model: [Release architecture](../architecture/release.md).
Tool effects: [Native support](../development/native-support.md).
Media details: [Installation media](media.md).

Commands that write disks, mutate providers, publish artifacts or destroy fixtures
require explicit task approval. This guide is not that approval.

## Prepare the builder

Use a matching native architecture (`x86_64` or `aarch64`). Install the pinned Go,
Bun and Python toolchains and native Podman. From the repository root:

```sh
bun install --frozen-lockfile
```

Produce a candidate with the admitted `soda-build` controller (see
[native support](../development/native-support.md)). Verify without rebuilding:

```sh
bash scripts/check-native.sh ARCH /ABS/PATH/TO/stage
```

Use a clean exact-revision checkout and a fresh output directory per attempt.

## Provision the host

Provision Fedora CoreOS with the selected installation media. Complete Forgejo's
native first-run installation on loopback as described in
[Operator setup](operator-setup.md). Do not expose an unfinished installer through
the public reverse proxy.

## Install built components

Install the candidate's application images, host content and Forgejo customization
payload through the admitted install path for that candidate. Prefer the same
immutable candidate for install and later update.

Existing-state maintenance updates affected components without replaying first
install as a service upgrade and without using `--rm` / `--replace` as repair
against preserved project roots.

## Reachability and activation

1. Establish real project reachability (addresses, routes, client trust).
2. Run setup/activate with explicit private bind address and TLS as in
   [Operator setup](operator-setup.md).
3. Trust the appliance public root certificate on intended clients when using
   local TLS.

Browser origins, Git advertisement and project routing are separate configuration
facts.

## Operator services

Stock Cockpit listens on all interfaces with the operator credential. Tailscaled, runners and project units follow
the appliance service definitions under `appliance/services/`. Preserve credentials,
project state, backups and failed evidence unless cleanup is explicitly approved.
