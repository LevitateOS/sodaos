# Release workflow

How developers produce and qualify release candidates. Durable model:
[Release architecture](../architecture/release.md). Tool contracts:
[Native support](native-support.md).

Retained-Project transitions follow the
[release maintenance contract](../architecture/release.md#host-update-ownership)
and [operator procedure](../guides/operator-setup.md#explicit-release-maintenance).
Each source-to-target pair requires its own native qualification receipt.

## Pipeline map

| Symptom or task | Open first | Then |
| --- | --- | --- |
| Candidate will not build | `lib/soda-release-build`, `lib/soda-release-tools` (`soda-build`) | `lib/soda-release-image` |
| Media/assemble wrong | `lib/soda-release-image` | `cmd/soda-install` |
| Guest/fixture behavior | `tools/acceptance` | `tests/installed` |
| Publish/promote | `lib/soda-release-deliver` | `soda-release` in `lib/soda-release-tools` |

## Docker release operator flow

This section defines the **target interface to implement**, not commands already
available. Today `soda-build` has development `candidate`/`media` boundaries and
produces appliance media for production; `soda-release prepare`, `stage` and
`promote` require appliance media. Docker packaging is currently the manual
[builder procedure](../guides/docker.md#build-the-runtime-image). Adding these
interfaces does not imply that any Docker image is published or qualified.
The [release architecture](../architecture/release.md#distribution-targets)
owns distribution identities and qualification requirements.

### Select once and preflight

Extend `soda-release` with a guided `start` entrypoint. The operator selects
`appliance`, `docker`, or `all`, the version tag, release notes, admitted worker
configuration and a fresh workflow output directory. The worker configuration
identifies the Soda and Fountain sources and isolated build paths. Guided mode
shows these inputs together and validates edits before execution; supplied flags
pre-seed the same flow. Non-interactive mode requires complete explicit inputs.

Before an expensive build, check clean committed source and the Fountain pin,
the successful latest `main` push CI for that exact Soda revision, admitted
controller/worker/toolchain setup, the locked offline Canlang input, supported
native host and engine prerequisites, output freshness and available capacity.
Report publisher authentication, repository/tag conflicts and known package
visibility/access blockers. New GHCR packages default to private; first-time
publication cannot complete staging until anonymous digest pulls succeed.
Never infer publication access from a successful source CI run or registry name.

Implementation and its local builds use the task's worktree. Extend worker
admission to accept a clean, exact-revision linked Git worktree: resolve its Git
metadata/common directory read-only, archive the selected worktree revision,
and keep writable outputs, build cache and temporary state below that worktree's
`.artifacts/`. Do not redirect a build to the canonical checkout to work around
current admission. Release production still requires that exact revision to
have passed `main` push CI. Worker admission is an existing prerequisite; `start`
does not install accounts, grant sudo or admit an executable automatically.

The proposed non-interactive entrypoint is:

```sh
# Target interface; not implemented yet. Substitute admitted paths and a new tag.
soda-release start --distribution docker --tag v0.1.0 \
  --worker-config /home/BUILD/worker.json \
  --notes /home/BUILD/release-notes.md \
  --out /home/BUILD/WORKTREE/.artifacts/releases/fresh-workflow \
  --non-interactive
```

`start` delegates to the admitted `soda-build` and existing local preparation
owners, then stops with an unqualified prepared release. It performs no remote
publication or deployment. One non-secret request records selected inputs,
source/target identities and artifact locations; credentials remain separate
restricted inputs. Subsequent commands consume the prepared release instead of
asking for the version and paths again. Reuse existing progress events and
artifact/evidence files; do not introduce a publication database or resumable
build engine. A failed build remains failed and retains its diagnostics.

### Build and artifact handoff

Add `--distribution appliance|docker|all` to `soda-build`, defaulting to
`appliance` for its existing production callers. Keep `--target` as a development
completion boundary, distinct from distribution selection. Docker/`all` builds
also require `--release-tag`; `start` passes its selected tag to the builder so
the outer image and its metadata carry the same version. For `all`, that tag
must match the appliance's versioned rootfs URL.

The completion boundaries are:

| Purpose and selection | Completion boundary |
| --- | --- |
| Production `appliance` | Common candidate and existing production media |
| Production `docker` | Common candidate and Docker runtime; no ISO or rootfs URL |
| Production `all` | One common candidate, Docker runtime and production media |
| Development `--target candidate` | Common candidate only; no deployment packaging |
| Development `--target media` | Existing appliance media boundary; appliance distribution only |
| Development `--target docker --distribution docker` | Common candidate and Docker packaging, explicitly non-qualifying |

Reject media-only inputs for Docker-only/candidate requests and reject conflicting
development selections before execution. Production `all` requires the existing
versioned rootfs URL; `start` derives it from the selected tag. Packaging resolves
its Fedora base and uses the Dockerfile from the same frozen source, consuming
the verified candidate host archive without recompiling the Soda payload.
The worker sees neither GitHub nor GHCR publication credentials.

The proposed local artifact handoff is:

| Artifact | Role |
| --- | --- |
| `artifacts/docker/image.oci` | Exact outer runtime OCI archive, retained locally |
| `artifacts/docker/docker.json` | Docker identity/provenance binding defined by architecture |
| `artifacts/docker/packages.txt` | Sorted actual outer-runtime RPM inventory |
| `artifacts/docker/compose.yaml` | Exact Compose installation artifact from frozen source |

Local release preparation checks those bytes and emits the existing prepared
release directory with a manifest recording its selected distributions. Public
Docker assets are named `docker.json`, `docker-packages.txt` and
`sodaos-docker-compose.yaml`, beside shared metadata and both corresponding-source
archives. The manifest records their hashes/sizes and the intended registry
repository/version/digest. Only appliance selections require `media.json`, ISO
and rootfs assets. Existing GitHub per-asset limits remain admission gates.

### Stage, qualify and promote

These command forms are the target additions to the existing `soda-release` CLI.
Prepared input fixes the version, targets and shipping bytes; neither `stage`
nor `promote` builds an image.

| Step | Target interface and operator-visible result |
| --- | --- |
| Inspect | `soda-release status --input PREPARED`: show bound identities, completed local phases, publication observations, blockers and next action. No mutation. |
| Stage | `soda-release stage --input PREPARED`: recheck source CI and all artifact identities; create/verify the exact Git tag, publish the Docker version image preserving its manifest digest, verify anonymous registry readback, then publish the complete GitHub asset set as an immutable unqualified prerelease. No recommendation changes. |
| Exercise | Pull the staged digest without registry credentials and download that version's Compose file. Deploy to the explicitly selected fresh native host/volumes and run the required setup, browser, Project, nested-container, factory and persistence checks using the existing acceptance/installed owners. Private prerequisites stay outside public artifacts. |
| Qualify | `soda-release qualify --input PREPARED --evidence FINALIZED_EVIDENCE`: validate independently observed identities and every required target observation, then stamp the bound local qualification receipt. This command does not deploy or turn an assertion into evidence. |
| Promote | `soda-release promote --input PREPARED`: recheck exact source CI, frozen remote assets, registry/version digest and all selected qualification receipts; publish their safe summary in release notes and update only the eligible recommendation pointers. No image upload/rebuild. |

Appliance-only and `all` requests retain the existing ISO download, installation
and boot qualification gate. Docker requires its own exact-public-image receipt;
the existing appliance `--smoke-passed` confirmation cannot stand in for it.
`all` waits for both receipts. A Docker-only promotion leaves GitHub's latest ISO
release unchanged. A prerelease version tag is discoverable but unqualified;
only promotion may move `ghcr.io/levitateos/sodaos:latest`.

Every phase reports the selected target, actual identities, completed checks,
restricted diagnostics path and next command. Missing credentials, provider
funding or host prerequisites identify the blocker without a fabricated pass.
Same-image stop/start and recreation qualify persistence; changing versions or
restoring backups is outside this receipt. On ambiguous remote writes, stop and
inspect actual registry/GitHub state. Preserve partial attempts, do not clobber
version tags, and do not silently retry or undo a recommendation change.

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
