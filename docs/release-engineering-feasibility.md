# Release engineering — first feasibility review

This is a source/public-upstream research receipt for Stage 1 of the
[release engineering plan](release-engineering-plan.md), not native acceptance or
an execution grant. Source was inspected at `8ffc0e1`; no retained appliance was
contacted. Public research inputs, failed URL attempts and source URLs are retained
under `.artifacts/release-engineering-research/` (`sources.tsv` and `SHA256SUMS`).
The plan owns requirements and current task status; this receipt records findings.

## Recommendation

**Prove a derived Fedora CoreOS image with bootc's existing OSTree backend and
logically bound appliance images.** Use the exact stable base, native container
build tooling and GHCR. Start with explicit activation, not an automatic timer.
Do not switch to the experimental composefs deployment backend, replace the kernel,
run a custom Cincinnati server or implement a parallel image-pull coordinator.

This is a recommended proof target, not approval to migrate the installed fleet.
Compared with rpm-ostree alone, bootc provides an existing mechanism for downloading
bound app images before host activation and retaining images for rollback deployments.
That addresses a concrete Soda need: the host helper, dashboard, Forgejo integration
and app images must be a compatible set even when the registry is unavailable after
reboot. It does not coordinate database migrations or prove application recovery.

The important costs are **moving shipped code out of writable locations** and
**replacing client-side RPM layering with image-time package installation**. A mere
registry URL change would leave both problems unresolved.

## 1. Exact base and selected upstream evidence

The repository's ISO and QEMU locks both name **CoreOS `44.20260817.3.2`**. The public
stable stream fetched during this review names the same release for x86_64; its
metadata reports last modification `2026-09-04T18:44:22Z`. This is the fetched
observation, not a promise that a moving stable URL will continue to select it.
No source lock was updated.

The [release metadata](https://builds.coreos.fedoraproject.org/prod/streams/stable/builds/44.20260817.3.2/release.json)
provides these architecture-specific base image references:

| Architecture | `quay.io/fedora/fedora-coreos` digest |
| --- | --- |
| x86_64 | `sha256:4222ad36286b40b8233e4ace5756fcdc73e22f498e4c6cc93cfe4a0e640dc27e` |
| aarch64 | `sha256:ae1aaa031ed14d9a504267d0a5033fad6eb9c29b42eedef7f1b8bb124acfbeb9` |

The [x86_64 build metadata](https://builds.coreos.fedoraproject.org/prod/streams/stable/builds/44.20260817.3.2/x86_64/meta.json)
already marks this base `containers.bootc=1` and `ostree.bootable=1`. Its
[package inventory](https://builds.coreos.fedoraproject.org/prod/streams/stable/builds/44.20260817.3.2/x86_64/commitmeta.json)
contains:

| Package | Version-release (epoch omitted) |
| --- | --- |
| bootc | `1.16.7-1.fc44` |
| rpm-ostree | `2026.2-1.fc44` |
| ostree | `2026.3-1.fc44` |
| podman | `5.8.4-1.fc44` |
| skopeo | `1.22.2-2.fc44` |
| containers-common | `0.67.0-1.fc44` |
| zincati | `0.0.32-1.fc44` |

These are upstream base metadata, not current installed-package observations. Fedora
RPM patches/build flags and native behavior still need inspection in the proof.
No OCI layers or RPMs were downloaded/built during this review.

Matching sources consulted:

- [rpm-ostree v2026.2 container model](https://github.com/coreos/rpm-ostree/blob/v2026.2/docs/container.md).
- [bootc v1.16.7 upgrades](https://github.com/bootc-dev/bootc/blob/v1.16.7/docs/src/upgrades.md),
  [rpm-ostree relationship](https://github.com/bootc-dev/bootc/blob/v1.16.7/docs/src/relationships.md),
  [security](https://github.com/bootc-dev/bootc/blob/v1.16.7/docs/src/security.md),
  [build guidance](https://github.com/bootc-dev/bootc/blob/v1.16.7/docs/src/building/guidance.md),
  [bound images](https://github.com/bootc-dev/bootc/blob/v1.16.7/docs/src/logically-bound-images.md).
  The fetched tag tree resolves to `bb8fb41e39cbb8c68b6e602307854a57b58f693a`.
- [Actual FCOS base producer](https://github.com/coreos/fedora-coreos-config/blob/682c839aabbc01564f1605bb41687a7511180031/Containerfile)
  referenced by the build, plus upstream [layering examples](https://github.com/coreos/layering-examples)
  for Tailscale and Go binaries/systemd units. Examples are illustrative, not pinned
  Soda recipes or permission to run their broad build flags.

The rpm-ostree guide still labels custom-build functionality experimental. Existing
upstream code and a bootc-marked base establish a credible proof path, not a blanket
Fedora support guarantee for Soda derivatives.

## 2. Mechanism decision table

| Question | Research result and recommendation | Remaining proof |
| --- | --- | --- |
| Derive a bootable base? | Yes in upstream mechanisms: use a digest-pinned FCOS `FROM`, add packages at image build with rpm-ostree, put non-RPM programs under `/usr`, and finalize using the matching upstream container tooling | Actual Soda package transaction, SELinux labels, boot and package inventory; pin resolved RPM inputs rather than assume today's repo contents are reproducible |
| Must we replace CoreOS with Fedora bootc? | No. The selected FCOS base already includes bootc and the bootable-image labels | Verify actual image contents and exact derived image on the existing OSTree backend |
| Can rpm-ostree consume GHCR? | It documents registry-backed OCI rebase/upgrade and `ostree-image-signed` verification through containers policy; GHCR is not a special OSTree protocol | Signed GHCR/native round trip; no retained rebase yet |
| Which activation owner? | Recommend bootc because bound app images and controlled download/application are directly useful to Soda | bootc transition from a Soda-like layered install, exact native state, rollback and application presence |
| Keep live package layering? | bootc's tagged docs say upgrades reject client-side package mutations. Put the selected host package set in the derived image | Migration must inventory and preserve operator additions; no blind `reset`/package removal |
| App image coordination? | Use bootc logically bound images for fixed appliance services; digest-pinned Quadlets in the host bind the combination | Existing Soda storage/SELinux/UID settings, local cache provenance, missing-image failure and old-deployment image availability |
| Scheduling? | Explicit activation first. Disable Zincati updates through its documented configuration in the candidate; leave bootc's fetch/apply timer inactive | Effective configs and no pre-existing staged update; later scheduling must control one activation owner |
| Signatures? | Candidate: keyed Sigstore signatures plus containers policy and registry attachments, using native clients | Exact Fedora client enforcement, cache/import behavior, signer rotation and GHCR attachment retention |

### bootc behavior that changes the design

The v1.16.7 upgrade guide documents `--download-only` and `--from-downloaded`:
download can be separated from permission to activate, and applying the downloaded
candidate need not re-resolve a moving channel. A reboot before download-only
activation discards that staged deployment while keeping cached image data. This
must be reflected accurately in status/recovery; do not invent a permanently staged
state. Verify the flags in the actual native binary before implementing the caller.

A normal upgrade stages content for the next boot; `--apply` reboots. The provided
fetch/apply timer is not a maintenance-window or migration coordinator. Do not enable
it merely because it ships with bootc. Bootc and rpm-ostree share OSTree state, but
that does not make mixing mutation owners safe, particularly for bootc-bound apps.

## 3. Actual Soda layout and required changes

The source owners inspected were `scripts/stage.py`, `scripts/install-native.sh`,
`internal/nativebuild/{bundle,installed}.go`, `appliance/provisioning/base.json`
and `appliance/services/`. Current install is intentionally first-install only.
It cannot become an updater by removing its refusal markers.

| Current owner/location | Recommended image model and implications |
| --- | --- |
| Programs in `/usr/local/libexec/soda`, setup/activation commands in `/usr/local` | Move shipped executable content to `/usr/libexec/soda` and appropriate `/usr/bin` or `/usr/sbin` paths. FCOS `/usr/local` is `/var/usrlocal`, so today's programs survive OS rollback unchanged. Update all fixed callers and inventory contracts together |
| Units in `/etc/systemd/system`, Quadlets in `/etc/containers/systemd` | Vendor units in `/usr/lib/systemd/system`, Quadlets in `/usr/share/containers/systemd`; preserve intentional `/etc` overrides. Existing `/etc` files take precedence and require exact migration, not deletion of an entire directory |
| sysusers/tmpfiles/sysctl definitions in `/etc` | Use upstream vendor directories when supported; keep runtime-created state in `/var`. Preserve UID 2000, subordinate mappings and existing identities; image build must not allocate arbitrary fixture identities |
| Forgejo templates/browser graph under `/var/lib/soda/forgejo/gitea` | Move release-owned customization into immutable host content or the pinned Forgejo app image using Forgejo's existing customization mechanism. Verify mount precedence and branding/SELinux behavior; do not replace `/data` or snapshot it into an image |
| `/etc/soda` generated config, tokens, certificates and machine settings | Keep machine-local and restricted. Separate release defaults/image selection from saved operator config; an old image ID in `host.json` must not silently override the release contract |
| Forgejo, dashboard, proxy persistent state under `/var/lib/soda` | Keep writable and preserved; handle schema/version compatibility explicitly before app startup |
| Native host RPM installation at first boot | Bake the baseline package set into the host image and preserve upstream repo/package provenance; retire the layering action only in the new image path and approved migration |
| Existing image-ID imports and `localhost/*:dev` aliases | Release-owned references must be immutable registry digests, not mutable dev aliases. Adapt exact callers and verifiers; do not retag retained images as an upgrade mechanism |

OSTree's `/etc` three-way merge preserves local edits; it does not guarantee that
new vendor defaults become effective. Explicitly surface conflicting old overrides.
Do not repeat the previous OS metadata branding error: retain vendor CoreOS identity
and use a separate Soda release record for product/version presentation.

### Bound images: use where their lifetime matches

Bootc reads `.container`/`.image` files selected by symlinks in
`/usr/lib/bootc/bound-images.d`. It pulls their `Image=` references before activation.
The tagged code (`crates/lib/src/boundimage.rs` and `podstorage.rs`) reads the bound
set and uses native Podman pulls when content is absent. Only `Image` is honored,
not arbitrary Quadlet pull flags. Existing cached content may be reused, so proving
an unsigned pull is rejected is not enough to prove all cache/import paths safe.

Soda's dashboard, Forgejo and proxy use system/rootful Quadlets (a `User=2000:2000`
container is not a rootless engine), making them candidates for this mechanism.
Configure the bootc additional image store **only for these bound services** as
upstream directs. Do not globally add it to Podman's storage configuration.

**Persistent Project OS containers must not depend on bootc garbage-collected
layers.** Their creation images need a verified import into ordinary retained
container storage. The same lifetime audit is required for Tailnet companions:
they may remain alive while releases change, and the host helper currently expects
an ordinary store image ID. Do not treat all five current bundle images as bound
simply because three appliance services fit. Test rollback and reference retention
without pruning existing project images.

## 4. Trust, GHCR and channel authority

[GitHub's registry documentation](https://docs.github.com/en/packages/working-with-a-github-packages-registry/working-with-the-container-registry)
documents OCI images, digest pulls and anonymous public pulls. No namespace was
created, authenticated or published to. GHCR availability and OCI format support do
not prove native signature attachments, retention or supported upgrade discovery.

The native rpm-ostree signed-image transport uses `/etc/containers/policy.json`.
Bootc also documents this policy; the tagged switch CLI has
`--enforce-container-sigpolicy`. The proof must verify the stored transport/policy
and refusal behavior rather than assume an image being signed forces verification.

The [policy specification](https://github.com/containers/container-libs/blob/image/v5.39.2/image/docs/containers-policy.json.5.md)
and [registry attachment configuration](https://github.com/containers/container-libs/blob/image/v5.39.2/image/docs/containers-registries.d.5.md)
match the `go.podman.io/image/v5 v5.39.2` dependency selected by skopeo v1.22.2.
They document `sigstoreSigned`, public-key policy, identity matching and
`use-sigstore-attachments`. Default registry configuration does not read/write these
attachments unless enabled. Fedora package patch/build differences remain to check.

Recommend **keyed Sigstore for the first proof**, keeping all signing inputs in
restricted local test files and trusting only test keys on an authorized disposable
fixture. Production custody/rotation needs its own decision; keyless CI identity is
not ruled out but adds issuer/identity/Rekor dependencies that are unnecessary to
prove image verification initially. Configure scoped Soda policy without weakening
or replacing unrelated Fedora/vendor trust.

Cosign-style signatures bind repository/digest, not the meaning of `stable` versus
`candidate`. A correctly signed preview image must not become stable-authorized just
because a tag moved. The smallest proposed authority is a separately signed channel
record binding channel, release sequence, approved architecture digest and freshness;
the host image can carry the application/version record without another independent
component inventory. Finalize format, bootstrap, replay protection and trusted time
handling before automatic discovery. An OCI signature by itself is not that protocol.

## 5. Zincati and the reported Cockpit error

Zincati discovers an update graph from Cincinnati; a GHCR tag is not such a graph.
Its documented `[updates] enabled = false` stops update actions while retaining an
idle service for observers. Recommend that setting in the new candidate and inspect
effective configuration during the native proof. Changing this on retained targets,
resolving an already staged deployment, or enabling another timer needs explicit
maintenance approval. A Zincati maintenance window alone does not select Soda releases.

Public Cockpit OSTree source at `0da7576ed87c25e07dcbc849d5c761f672a0162c` contains
a concrete suspicious path:

- `src/client.js:get_os_origin()` accepts `container-image-reference` as an origin.
- `get_default_origin()` splits that string at its last colon into remote and branch.
- For an image reference ending in `:stable`, its remote becomes precisely
  `ostree-image-signed:docker://quay.io/fedora/fedora-coreos`.
- `src/remotes.js:listBranches()` invokes `ostree remote refs REMOTE`.

That explains how the reported string can reach a conventional remote lookup. It
is **not yet a target-matched diagnosis**: the installed cockpit-ostree version,
complete UI path and its exact caller were not observed. The base inventory does
not contain the subsequently layered Cockpit package. Native read-only diagnostics
and matching installed source must confirm it before choosing an upstream fix or
unsupported-control presentation. No custom update page is justified by this alone.

## 6. Bounded proof proposal — not execution approval

### Local candidate preparation

Reuse the existing native build outputs and add a reviewed image recipe/inventory
adapter only after the preferred bootc proof direction is accepted. Pin the base
above and resolved host RPM artifacts; do not pull a moving `stable` in the build.
Prepare A/B/C candidates with no machine secrets: A initial image, B harmless binary/
asset change on the same base (emergency lane), C new qualified base when available.
Source/static trust fixtures may use synthetic documents; no signing-key creation
or provider resources are required for the current research scope.

### Isolated native proof

Request one **new x86_64 VM**, exact name/disk/time bound and inputs to be recorded
before execution, with console access independent of Tailnet and no public listeners.
Do not use the retained Spaces/Tailnet VM or its data. Proposed allowed effects:

1. Provision the locked base and a Soda-like initial layout with synthetic users,
   configuration, a small database and one project root; no real provider enrollment.
2. Build/load the exact candidate through approved local archive/registry transport;
   use fixture-only trust inputs. Registry service/publishing scope must be named
   explicitly; GHCR publication remains the later separate delivery stage.
3. Inspect existing client-side layering and rehearse its exact transition to the
   image-owned package set. Preserve an independent pre-transition snapshot/backup;
   prove old overrides cannot shadow new vendor content silently.
4. Perform A → B, using download-only then explicit activation/reboot. Verify exact
   helper/assets/app digests, preserved config/data/project identity, enforcing
   SELinux, app availability without registry access after reboot and idle Zincati.
5. Refuse unsigned/wrong-key/wrong-repository artifacts and incompatible schema paths.
   Exercise unavailable bound image, incomplete download and cached-image provenance.
6. Test reboot before download-only activation, one interrupted activation, a new
   deployment with a deliberately failed Soda service, and explicit compatible boot
   fallback. Preserve a later database write across fallback; never restore the old
   DB merely to make the old app pass. Report cases requiring forward repair.
7. C's base-change path is a separate case once a different exact base is qualified;
   A → B on one CoreOS base cannot stand in for it. Native aarch64 remains separate.

Retain disks, fixture records and test keys/evidence according to the exact grant;
no automatic teardown or pruning. This matrix is not permission to start a VM,
reboot, create a real project on a retained appliance or publish images.

## Outcome and remaining limits

Checks run: repository ISO/QEMU lock release matched fetched upstream metadata;
both recorded architecture digests matched the release record; four initially
fetched documents matched their selected-version copies byte-for-byte; relative
file link targets and `git diff --check` passed. Public evidence files were hashed.
These are research/document checks, not image builds, signature tests or native
upgrade/boot evidence.

The research establishes a concrete upstream mechanism worth proving: **derived
FCOS + bootc/OSTree + bound core app images + native signature policy**, with manual
activation first. rpm-ostree-only remains a possible alternative if retaining local
package layering is a product requirement, but it would need separate app-staging
coordination and must not silently replace the preferred proof path.

Stage 1 has a source recommendation and proof proposal, not exhaustive closure.
Still unresolved: installed Cockpit diagnosis; native signature/cache enforcement;
Soda package/build feasibility and reproducible RPM sourcing; exact configuration/
asset migration; channel-format/replay policy; production signing custody; per-target
migration; GHCR round trip and native upgrade/recovery evidence. No production
recipe, update client, CI workflow, registry or signing key was created.
