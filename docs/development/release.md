# Release workflow

How developers produce and qualify release candidates. Durable model:
[Release architecture](../architecture/release.md). Tool contracts:
[Native support](native-support.md).

The [Project runtime upgrade implementation plan](project-runtime-upgrade-plan.md)
defines the proposed work to preserve terminal support in retained Projects across
host releases. It does not add a supported upgrade path until qualified.

## Pipeline map

| Symptom or task | Open first | Then |
| --- | --- | --- |
| Candidate will not build | `lib/soda-release-build`, `lib/soda-release-tools` (`soda-build`) | `lib/soda-release-image` |
| Media/assemble wrong | `lib/soda-release-image` | `cmd/soda-install` |
| Guest/fixture behavior | `tools/acceptance` | `tests/installed` |
| Publish/promote | `lib/soda-release-deliver` | `soda-release` in `lib/soda-release-tools` |

## Development vs qualification

- Use the smallest applicable development target while iterating.
- Do not relabel a failed release run as successful.
- Retained artifacts may support explicitly non-qualifying development checks.
- Use the bounded first-install qualification defined in the
  [release architecture](../architecture/release.md#public-distribution).

`project-terminal` is compiled once during production asset staging. Host assembly
copies that artifact to its runtime and maintenance paths and checks its digest
against the selected built Project image. The release payload's `ProjectSupport`
binding follows the same candidate through media and GitHub delivery; delivery
also inspects the helper content in the host and Project OCI archives. Source CI
runs the focused artifact staging regression with the existing Rust workspace
tests. Retained-Project repair and native upgrade qualification remain separate
from fresh-install qualification. `--upgrade-from SOURCE_RELEASE_ID` records one
prospective source before building; this declaration grants no activation or
publication authority.

After the actual native journey has emitted its transition, interrupted recovery
and stopped-Project observations, stamp the unchanged candidate:

```sh
soda-release qualify-upgrade --candidate /BUILD/artifacts --evidence /PRIVATE/native-upgrade-evidence
```

The evidence directory contains `transition.json`, `interrupted_recovery.json`
and `stopped_project.json` using the shared `NativeEvidence` contract in
`lib/soda-release-deliver/src/upgrade.rs`. Stamping checks concrete source/target
deployment identities, the native finalization lock, closed admission, preserved
container and protected-state identities, observed support and stopped state. It
hashes those files and writes a new `upgrade-qualification.json` beside the candidate.
The interrupted checkpoint is `support_verified_host_locked`, after support
readback and before releasing native finalization. Matching support is a no-op;
its evidence records verification without claiming a file replacement.
Missing evidence, same-commit feasibility experiments or an unbound target cannot
produce this receipt. Retain the original observations locally.

Staging a declared upgrade source requires that receipt and includes it in the
same immutable GitHub release. Fresh-install promotion retains its separate smoke
receipt. Operator maintenance selects one complete downloaded release directory;
the existing hash-named rootfs carries the host, Project images and maintenance
support. Its `DeploymentCommit` is read back during media assembly together with
the embedded payload and support hashes. It must not be replaced with the
assembler's separate import commit.

## Build once, stage, install, promote

### Offline Canlang input

Before the candidate producer takes its Git source snapshot, select and commit
`system/project/canlang-release.json` and supply its exact public
`canlang-tools-linux-x86_64.tar.gz` beneath the canonical source checkout's
`.artifacts/canlang/`. The Canlang producer owns the compiler/platform/dependency
archive, internal inventory and release lock; Soda owns selection and checksum
admission. Archive bytes stay a generated public build input, not a second app
manifest or an untracked source/credential exception.

The snapshot owner copies that one file using the committed snapshot lock and
rejects missing, linked, oversized or checksum-mismatched input. The asset owner
then invokes `soda-stage-canlang` before the Project and runtime images consume the staged
tools, and includes the same public lock/archive in the host maintenance
payload. Exact commands and refusals belong to
[release assets](../../tools/release-assets/README.md#admit-offline-canlang-tools).
Existing retained Projects receive tools only through
[explicit Canlang maintenance](../../cmd/soda-project-maintain/README.md#canlang-tools).

This handoff does not authorize a build, installation or publication. Native
development evidence for the offline runtime is distinct from G3 appliance and
production qualification. The named runtime image follows the existing image
catalog, OCI export and offline import owners described in
[release architecture](../architecture/release.md#public-distribution).
Complete appliance delivery still requires matching source and release
qualification; application images remain separately assembled artifacts.

Use clean committed Soda and Fountain sources and the admitted native build worker.
The Fountain checkout must match `fountain-source.json` in that Soda revision.
Push Soda to `main` and wait for its `CI` workflow to pass before the expensive
native build. `soda-release` independently checks the latest push run for that
exact commit and the Fountain pin before staging and again before promotion.
Missing, pending, skipped, cancelled or failed CI cannot authorize delivery;
a PR's synthetic merge commit cannot qualify a different release commit.

GitHub Free hosts source CI. The existing local native builder owns OS builds and
the guided installation smoke; no self-hosted Actions agent or custom download
server is required. GitHub credentials remain with the operator running
`soda-release`, outside the isolated build worker. CI has read-only repository
access and receives no release keys or publication credentials.

Choose the release tag before building: the installer embeds the exact versioned
GitHub rootfs URL. For example, with tag `v0.1.0`:

```sh
sudo /ADMITTED/soda-build --worker-config /RESTRICTED/worker.json \
  --arch x86_64 --out /OUTPUT_PARENT/fresh-release \
  --forgejo-source /CANONICAL/FORGEJO-CHECKOUT \
  --rootfs-base-url https://github.com/LevitateOS/sodaos/releases/download/v0.1.0
```

The build emits an unqualified media candidate. It never publishes or silently
installs on the builder. Omit the development `fast` compression mode for releases.
Staging enforces the 2 GiB per-asset limit before upload.
The isolated worker uses a local artifact-signing key for authenticated packaging;
GitHub authenticates distribution separately.

For local release preparation without publication, supply release notes and a
new output directory:

```sh
soda-release prepare --candidate /BUILD/artifacts \
  --tag v0.1.0 \
  --notes /PRIVATE/candidate-notes.md --out /PRIVATE/new-release
```

This checks and copies the complete bound asset set using the same preparation
as `stage`, including asset limits and any required upgrade qualification receipt.
It does not contact GitHub, create a tag, publish, or qualify the release.

To publish, enable immutable releases once in the repository settings. With the
installed `gh` client authenticated for this repository, use a new output directory:

```sh
soda-release stage --candidate /BUILD/artifacts \
  --tag v0.1.0 \
  --notes /PRIVATE/candidate-notes.md --out /PRIVATE/new-release
```

Staging reads `media/media.json` from the candidate, copies and hashes the complete
release asset set, and checks source CI. It creates a missing tag at the exact built
commit, refuses a tag pointing elsewhere, then uploads a draft, checks the remote
inventory and publishes a prerelease. Notes automatically identify it as unqualified.
It refuses an existing release or changed/oversize files; it never uses upload
clobbering. Preserve a failed attempt and inspect GitHub's actual state before
retrying. Uploaded source archives include both repositories, not just GitHub's
automatically generated Soda archive. `fountain-source.tar` comes from the separate
Fountain repository selected by `fountain-source.json`, at the exact revision used
to compile Fountain. Hosting that corresponding-source archive beside the SodaOS
binary does not change Fountain's source ownership.

Download `sodaos-x86_64.iso` from that release and use those bytes for one fresh
native KVM installation into a disposable disk. Exercise the guided disk installer,
reboot without installation media, read the installed Soda revision and verify all
image identities in the candidate are locally available. Keep temporary files and
VM storage on the workspace disk. Do not substitute a retained appliance or an
injected preinstalled disk for this installation.

After that smoke passes, supply the full Soda revision actually read from the
installed guest. `--smoke-passed` confirms that this downloaded ISO installed,
the guest booted without installation media, and all candidate images were present.
The command derives the release identities and hashes and writes
`qualification.json` in the prepared release directory; no hand-written receipt
is needed.

```sh
soda-release promote --input /PRIVATE/new-release \
  --downloaded-iso /PRIVATE/sodaos-x86_64.iso \
  --observed-revision FULL_REVISION_READ_FROM_INSTALLED_GUEST --smoke-passed
```

Promotion checks the bound receipt, verifies the downloaded ISO using GitHub's
attestation and rereads the locked remote inventory. It records the CI run and receipt in
release notes and changes the same release to latest. This is the website download:
`https://github.com/LevitateOS/sodaos/releases/latest/download/sodaos-x86_64.iso`.
Use versioned URLs in the installer and latest only for the website's default CTA.
Source checks, artifact admission and the native smoke each cover a different
boundary; do not recursively rerun them after they pass without a relevant change.

<a id="fountain-upstream-maintenance"></a>

## Fountain upstream maintenance

Fountain follows Forgejo's maintained 15.0 LTS line. An update begins from an exact
reviewed upstream tag and a clean Fountain branch. Record the old upstream base, old
Fountain revision, new upstream tag and resulting Fountain revision. Review upstream
release/security notes, the complete base-to-base diff and migration implications
before replaying the attributable Fountain commits. Use `git range-diff` across the
old and new patch ranges to expose dropped, duplicated or unintentionally rewritten
changes. Do not mix a Forgejo major upgrade into an LTS maintenance update.

Refresh the Fountain source base and the matching Forgejo runtime-image base in the
same change. Reconcile dependency manifests and generated inputs from source; keep
the nested `sdk/` module independently buildable and retain the GPL root license and
all modified-vendor licenses/change notes. Run the affected upstream and Fountain
tests, the standalone SDK suite with workspace resolution disabled, and the real
package install/replace/start checks for changed session, capability, native-route
or lifecycle boundaries. A generator completing successfully is not source or
license verification.

Build the candidate only from the two clean exact-revision checkouts. Before
qualification, verify that `artifacts/forgejo-source.tar` is the archived Fountain
revision used to compile the executable and contains `LICENSE`, `sdk/go.mod`, SDK
source, `contrib/extensions/README.md`, and the vendored dependency licenses/change
notes. Verify separately that the SodaOS source archive contains the Soda extension
package/backend sources and that the host image carries `LICENSE` and
`NOTICE` under `/usr/share/licenses/soda/`. Bind both source archives, the runtime image and the separately built
package/image in the candidate inventory, then repeat the affected native installed
journeys on native `x86_64` under the [platform scope](../architecture/release.md#architectures).
The candidate also records the native architecture and hashes of the patched
Forgejo executable, separate package files and host service definitions. The host
image carries the same inventory at `/usr/share/soda/host-image/content.json` for
installed readback. Forgejo compiler provenance names the digest-pinned Go/Alpine
image and the sorted APK package versions resolved for its native SQLite toolchain.

Local maintenance commits and development candidates do not publish anything.
Pushing a fork, publishing artifacts, signing a release or activating production
remains a separate explicitly authorized action.

## Commands

Follow the admitted interfaces in [Native support](native-support.md) for
candidate production, media, export and evidence. Carry out build, publication and
disposable-disk installation when the task authorizes those effects; a source-only
implementation task does not itself authorize remote publication.
