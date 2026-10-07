# Native build and installation

How to prepare builders, produce candidates, install components and activate a
software factory appliance with supporting persistent human projects.
Release model: [Release architecture](../architecture/release.md).
Tool effects: [Native support](../development/native-support.md).
Media details: [Installation media](media.md).

After installation and private operator setup, configure bounded work through the
[factory operator interface](../reference/factory.md). The public
[first-task walkthrough](../public/30-Use-Soda/15-software-factory.md) connects
admission to verified results and human merge.

Commands that write disks, mutate providers, publish artifacts or destroy fixtures
require explicit task approval. This guide is not that approval.

## Prepare the builder

Use a native `x86_64` builder under the
[platform scope](../architecture/release.md#architectures). Install the pinned Go,
Rust and Bun toolchains and native Podman. From the repository root:

```sh
bun install --frozen-lockfile
```

Produce a candidate with the admitted `soda-build` controller (see
[native support](../development/native-support.md)). Verify without rebuilding:

```sh
bash scripts/check-native.sh ARCH /ABS/PATH/TO/stage
```

Use clean exact-revision Soda and Forgejo fork checkouts and a fresh output
directory per attempt. Pass the fork checkout with `--forgejo-source`; the
candidate snapshots its exact HEAD and embeds its patched binary. The separate
`extension` image carries the Soda package installed by
`soda-extension-install.service` before an activated Forgejo starts. An install
failure blocks Forgejo startup.

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

The dashboard mounts only its config and credential files, the host and
identity helper sockets, its private data and the shared extension IPC
directory. Forgejo joins only that IPC directory through supplemental group
2100. Keep SELinux enabled so the private mounts retain separate labels.

Browser origins, Git advertisement and project routing are separate configuration
facts.

## Local CA fingerprint admission

The installer reads the public local root from one regular file without following
symlinks, with a 16 KiB input limit. It accepts one `CERTIFICATE` PEM envelope,
standard padded Base64 and surrounding whitespace. It refuses private keys,
multiple certificates and additional content.

Before displaying the SHA-256 fingerprint of the original DER, it requires
canonical DER, a positive nonzero serial using at most 20 INTEGER content octets,
matching signature identifiers, CA BasicConstraints, and certificate-signing permission
when KeyUsage is present. It refuses duplicate extensions and critical extensions
other than BasicConstraints and KeyUsage. Supported RSA, named-curve ECDSA and
Ed25519 keys must have valid algorithm parameters; the self-signature must verify
with the supported SHA-2 or Ed25519 profile.

This admission identifies the public root for explicit client trust. The operator
still decides whether to trust it; the display operation performs no validity-window,
chain or hostname verification. Invalid input prevents ready-to-open trust guidance.

## Operator services

Stock Cockpit listens on all interfaces with the operator credential. Tailscaled and project units follow
the appliance service definitions under `system/host/services/`. Local CI execution is
unavailable. Preserve credentials,
project state, backups and failed evidence unless cleanup is explicitly approved.
