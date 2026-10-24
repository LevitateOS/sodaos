# soda-release-tools

Operator tools for producing Soda candidates and installation media, inspecting
archives and retrieving verified upstream CoreOS inputs. This package supplies
`soda-build`, `soda-candidate`, `soda-artifacts`, `soda-candidate-check` and `soda-release`.

## Choose a command

| Command | Purpose |
| --- | --- |
| `soda-candidate` | Interactive or scripted wrapper around the admitted build controller. |
| `soda-build` | Dispatch a candidate or media build to an isolated worker. |
| `soda-candidate-check` | Verify candidate archives against architecture and exact source revisions. |
| `soda-artifacts` | Inspect OCI archives, fetch verified CoreOS inputs, convert private Butane, or check the Fountain pin against published releases. |
| `soda-release` | Stage immutable GitHub release assets and promote them after the native install smoke. |

Use `--help` on any binary to see its flags. Builds require native Linux
`x86_64`, clean committed Soda and Fountain checkouts, installed toolchains and an
operator-admitted controller with restricted worker configuration. Start in the
Soda checkout root. Follow the
[native support guide](../../docs/development/native-support.md#local-host-content-image-candidate)
for setup and input admission; a direct `cargo run` does not admit a controller.
For the source-to-download path, follow the [release workflow](../../docs/development/release.md).
The default build produces unqualified release media; staging and promotion are
explicit subsequent commands. Staging derives the media path from the candidate
and creates its exact tag when missing. Qualification consumes a protected native
collector result bound to the candidate, source, and downloaded ISO. An operator
pass flag or caller-authored observation does not establish release authority.

`soda-release collect --config /home/RELEASE-AUTHORITY/collector.json --candidate /BUILD/artifacts
--out /home/RELEASE-AUTHORITY/observations-attempt` runs the admitted collector as `soda-qualifier`
with an explicit public candidate snapshot read-only and a separate writable
attempt. The root-only configuration binds both executable hashes, completed
build/candidate/payload hashes, exact public-input hash/size inventory, purpose,
target, output parent and timeout. Every
configuration/executable/evidence ancestor must be root-controlled; a root-owned
file below a caller-writable directory is refused. The collector receives only
`--candidate`, `--out` and `--target` and emits exact-bound `collector.json` plus
the distribution's native observations. Only exit zero and confirmed native-unit
retirement produce `executed.json` and a sealed `release-evidence.json` inventory.
`soda-release finalize-evidence --out /home/RELEASE-AUTHORITY/observations-attempt` recovers that
exact executed receipt without rerunning the collector or build. Failed,
cancelled, incomplete and mismatched attempts stay unqualified. Development
evidence can finalize but cannot qualify a production release. Docker, appliance
and upgrade qualification all require the protected production receipt; appliance
uses its sealed `appliance/native.json` rather than a smoke-pass flag. Upgrade
stamping retains a private collector locator; promotion rechecks that protected
source and incorporates the typed receipt into its authenticated release summary.
Preparation and staging carry the prospective source and immutable target assets
without claiming upgrade support. Neither the local receipt nor its locator is
an immutable release attachment. The locator is never published.

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

The `soda-candidate` wrapper offers prompts and `candidate`/`media` modes,
including the required `--forgejo-source` input: it suggests the sibling
`forgejo-ext` checkout when one is present, asks otherwise, and forwards the
answer to the controller. Flags only pre-seed answers. See the
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
soda-artifacts check-fountain-pin
```

These commands read and validate existing artifacts. Use the actual archive
path and full source revisions from that candidate. They do not qualify a host.
`check-fountain-pin` answers whether a published Fountain release serves the
pinned `fountain-source.json` revision before any build input freezes; run it
from the checkout first and pass the reported URL as `SODA_FOUNTAIN_RELEASE_URL`.

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
