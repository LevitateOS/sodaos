# soda-release-deliver

Validate candidate artifacts, prepare release documents, verify signed downloads,
sign local payloads and publish with a protected ledger. Rust imports use
`soda_release_deliver`. This package is a library with no operator CLI.

## Use

Consumers supply integrity-controlled `Trust` and the explicit inputs for each
operation. Do not use build-produced metadata as release authority. The
[trusted delivery guide](../../docs/development/native-support.md#trusted-release-delivery-worker)
owns custody, private-file handling and operation effects.

Public entry points include:

- [`check::check_candidate`](src/check.rs): bind a candidate artifacts
  directory to the requested architecture, Soda revision and Fountain revision,
  then verify its archives. Operators can use
  [`soda-candidate-check`](../soda-release-tools/README.md) for this operation.
- [`prepare::prepare`](src/prepare.rs): combine an admitted candidate,
  media binding and qualification record into a local release OCI document.
- [`model`](src/model/mod.rs): `Trust`, `Permit`, `Release`, `Channel` and
  `Highwater`, plus `admit_release` and `admit_channel`.
- [`native`](src/native/mod.rs): the `Runner` interface and restricted native
  skopeo implementation for verification and local signing.
- [`fetch::fetch`](src/fetch/mod.rs): verify a channel and its images while
  maintaining caller-selected durable highwater state.
- [`publish::publish`](src/publish/mod.rs): ledger-serialized registry
  publication or observation of an existing attempt.

For a pure digest check:

```rust
let digest = soda_release_deliver::hash_bytes(b"example payload");
assert!(soda_release_deliver::is_digest_ref(&digest));
```

This only identifies bytes; it provides no signature or qualification claim.

## Operational boundaries

Candidate checks read and verify artifacts. Preparation and signing write local
outputs; fetch downloads and updates authority state; publication uses explicit
credentials and makes real registry writes. Native operations require the locked
skopeo version in [`tools.json`](tools.json) and admitted trust/private inputs.
Keep new outputs under `.artifacts/release-delivery/` with existing real parents.

Publication failures can leave uncertain effects recorded in the ledger. Preserve
the attempt and observe it before deciding to retry; a pending publication is
held against blind replay. These APIs do not install images, change host trust,
activate releases or reboot.

See [release architecture](../../docs/architecture/release.md) for the supported
platform and release model, and [release workflow](../../docs/development/release.md)
for the pipeline stages.
