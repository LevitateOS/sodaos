# soda-release-image

Assemble Soda host content, candidate artifacts and installation media through
the shared release build pipeline. Rust imports use `soda_release_image`.
This package is a library; operators use
[`soda-build`](../soda-release-tools/README.md).

## Use

For normal candidate or ISO production, follow the admitted controller workflow
in the [native support guide](../../docs/development/native-support.md#local-host-content-image-candidate).
The CLI supplies source admission, worker isolation and concrete implementations
of the library's production and progress interfaces.

Rust consumers can start from these entry points:

- [`request::Request`](src/request.rs): source, output, architecture,
  revisions, repository names and candidate/media target selection.
- [`build::build`](src/build.rs): the candidate execution entry point. It
  accepts a `Cancel`, request, `Progress` and a production factory, returning
  `request::Result` or `error::Error`.
- [`foreign::Production`](src/foreign.rs): the boundary for compilation,
  container builds, OCI inspection, input resolution and media signing.
- [`foreign::Progress`](src/foreign.rs): phase, step and build-log reporting.
- [`prepare::prepare`](src/prepare.rs): prepare a fresh host build context
  from source, revision and architecture using the supplied production adapter.
- [`build_media`](src/build_media.rs): admission and assembly of the
  authenticated inputs used for installation media.

Use the existing controller adapter in
[`soda-release-tools::pipeline`](../soda-release-tools/src/pipeline.rs) as a
reference for wiring these interfaces. Calling isolated helpers does not replace
the controller's source and worker admission.

## Inputs, outputs and effects

Builds require native Linux `x86_64`, exact committed Soda and Fountain sources,
resolved upstream inputs and the documented native tools. Each attempt uses
fresh output and produces work, artifacts, evidence, release and log directories.
Operations execute build tools, stage filesystem content and assemble archives;
media assembly can use fixture-only local signing inputs.

Candidate and media development targets stop the same producer at different
boundaries. A built candidate or locally authenticated development ISO does not
establish qualification or production signing authority. Content inventory
records exact files, modes and links; it is integrity evidence rather than a
native acceptance receipt. The library does not install or publish the result.

The [release architecture](../../docs/architecture/release.md) owns the candidate,
trust and platform contract. See [release workflow](../../docs/development/release.md)
for qualification and [installation](../../docs/guides/installation.md) for
consuming a completed candidate.
