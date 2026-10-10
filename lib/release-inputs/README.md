# soda-build-tools

Read and validate inputs used by Soda release consumers without fetching,
building or publishing them. This directory's Cargo package is
`soda-build-tools`; Rust imports use `soda_build_tools`.

## Use

Depend on this workspace library for metadata validation. There is no CLI.

```rust
use soda_build_tools::reader;

assert_eq!(reader::oci_architecture("x86_64").unwrap(), "amd64");
assert!(reader::is_digest(&"a".repeat(64)));
assert!(!reader::is_digest(&"A".repeat(64)));
```

The main entry points are:

- [`reader`](src/reader.rs): architecture, SHA-256 digest and source-revision
  shape checks.
- [`reader::stream`](src/reader/stream.rs): `LiveInputs`, `ResolvedCoreOS` and
  their validators for controller-resolved input handoffs.
- [`reader::settings`](src/reader/settings.rs): recipe base, unit image and
  command discovery from caller-selected files.
- [`reader::signature`](src/reader/signature.rs): GnuPG status validation
  against a caller-supplied signer fingerprint.
- [`elf::elf64_le_header`](src/elf.rs): bounded ELF64 little-endian header
  access; consumers retain their machine and file-type allowlists.

## Optional trust-key admission

Enable the `trust-key` Cargo feature to use
`trust_key::parse_p256_public_key(&str)`. It accepts one bounded `PUBLIC KEY`
PEM block containing a strict P-256 SPKI and returns the original DER bytes
for fingerprinting. Invalid keys return `Err(())`.

The [release architecture](../../docs/architecture/release.md#trust-key-admission)
owns trust-key requirements. This parser does not grant signer authority or
verify a release signature.

## Limits and effects

Only Soda's supported `x86_64` architecture is admitted; OCI spells it `amd64`.
Digest/revision checks validate shape, not the existence or provenance of bytes.
Some settings helpers read files; the crate performs no network fetches or
writes. IO and pipeline operations belong to
[`soda-release-build`](../soda-release-build/README.md). Operators should start
with the [release tools](../soda-release-tools/README.md) and
[native support guide](../../docs/development/native-support.md).
