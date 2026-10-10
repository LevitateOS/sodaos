# soda-release-tools

Operator tools for producing Soda candidates and installation media, inspecting
archives and retrieving verified upstream CoreOS inputs. This package supplies
`soda-build`, `soda-candidate`, `soda-artifacts` and `soda-candidate-check`.

## Choose a command

| Command | Purpose |
| --- | --- |
| `soda-candidate` | Interactive or scripted wrapper around the admitted build controller. |
| `soda-build` | Dispatch a candidate or media build to an isolated worker. |
| `soda-candidate-check` | Verify candidate archives against architecture and exact source revisions. |
| `soda-artifacts` | Inspect OCI archives, fetch verified CoreOS inputs or convert private Butane. |

Use `--help` on any binary to see its flags. Builds require native Linux
`x86_64`, clean committed Soda and Fountain checkouts, pinned toolchains and an
operator-admitted controller with restricted worker configuration. Start in the
Soda checkout root. Follow the
[native support guide](../../docs/development/native-support.md#local-host-content-image-candidate)
for setup and input admission; a direct `cargo run` does not admit a controller.

## Produce a development candidate

Replace the paths below with the admitted executable, restricted configuration
and a fresh output below `.artifacts/releases/`:

```sh
sudo /ADMITTED/soda-build --worker-config /RESTRICTED/worker.json \
  --arch x86_64 --out /OUTPUT_PARENT/candidate-attempt --development --target candidate \
  --forgejo-source /CANONICAL/FORGEJO-CHECKOUT
```

For media, use the same `soda-build` invocation with `--target media` and
`--rootfs-base-url http://FIXTURE_ADDRESS:PORT`; optional
`--media-compression fast` applies only to development media.

The `soda-candidate` wrapper offers prompts and `candidate`/`media` modes, but
its controller invocation omits the required `--forgejo-source` input. Builds
through that wrapper fail source admission; use `soda-build` directly as shown
above. See the
[candidate operator guide](../../docs/guides/soda-candidate.md) and native support
guide for rootfs pickup and fixture authority. Builds write artifacts and logs
and execute native tools. Development output is never a qualified release.
Output leaf names use lowercase letters, digits or dashes and must be fresh.

## Check or inspect artifacts

```sh
soda-candidate-check --candidate /CANDIDATE/artifacts --arch x86_64 \
  --soda-revision FULL_SODA_COMMIT --forgejo-revision FULL_FOUNTAIN_COMMIT
soda-artifacts inspect-oci --source /CANDIDATE/artifacts/image.oci.tar \
  --arch x86_64 --revision FULL_SODA_COMMIT
```

These commands read and validate existing artifacts. Use the actual archive
path and full source revisions from that candidate. They do not qualify a host.

## Fetch and convert inputs

```sh
soda-artifacts fetch-coreos --arch x86_64 \
  --keyring /TRUSTED/fedora.gpg --signer TRUSTED_FULL_FINGERPRINT \
  --out /CACHE/NEW_ATTEMPT
soda-artifacts convert-butane --arch x86_64 \
  --source /PRIVATE/instance.bu --out /PRIVATE/instance.ign
```

`fetch-coreos-iso` accepts the same fetch flags for the upstream ISO. Fetches
download, verify hashes/signatures and retain fresh outputs. Select the keyring
and signer independently. Conversion requires installed native Butane and a real
private output parent; it writes a fresh private Ignition file. See
[verified CoreOS and provisioning](../../docs/development/native-support.md#verified-coreos-and-private-provisioning)
for prerequisites. These tools do not install, activate or publish Soda.
