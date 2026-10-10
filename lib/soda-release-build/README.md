# soda-release-build

Native build and input-fetch primitives used to produce Soda candidate artifacts.
This library resolves upstream CoreOS inputs, retrieves verified images and runs
production compilation, asset staging and container-image export. Rust imports
use `soda_release_build`.

## Use

Operators normally use [`soda-build` or `soda-artifacts`](../soda-release-tools/README.md),
which supply the required input admission and execution context. This package
has no binary of its own.

For Rust consumers, the main entry points are:

- [`coreos::fetch_coreos`](src/coreos.rs): fetch a new verified upstream QEMU
  image using `arch`, trusted keyring, signer fingerprint and output path;
  return a `VerifiedBase` record.
- [`coreos_iso::fetch_coreos_iso`](src/coreos_iso.rs): the equivalent verified
  upstream installation ISO retrieval.
- [`live_inputs::read_live_inputs`](src/live_inputs.rs): admit the
  controller-resolved `LiveInputs` handoff for the isolated build.
- [`production::Production`](src/production.rs): explicit source, revision,
  native-tool and output paths with execution/capture hooks. Its production
  steps compile programs, stage assets, build images and export archives.
- [`http::HttpTransport`](src/http.rs): the transport boundary used by input
  resolution and retrieval; `*_with` fetch functions accept a transport.

The input validators and records shared with other consumers live in
[`soda-build-tools`](../release-inputs/README.md).

## Inputs and effects

Use the native Linux `x86_64` builder and exact source checkouts described by
the [native support guide](../../docs/development/native-support.md#local-host-content-image-candidate).
Production requires explicit execution hooks and frozen inputs. Do not treat
constructing a default `Production` value as an executable build configuration.

CoreOS retrieval resolves the stable stream live, downloads content and checks
hashes and a GnuPG signature against an independently trusted keyring and signer.
It creates a fresh output directory, retaining failures for inspection. A
verified upstream base is a build/fixture input, not a preinstalled Soda system.

Build operations create artifacts and can execute compilers and container tools.
They do not sign, publish or install Soda releases. `Error` exposes a message
and optional exit code, signal and cancellation details for callers to report.

The [release workflow](../../docs/development/release.md) explains development
versus qualification; the [release architecture](../../docs/architecture/release.md)
owns the candidate and platform contract.
