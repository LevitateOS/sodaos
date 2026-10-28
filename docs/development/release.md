# Release workflow

**Freshness is mandatory:** follow the [freshness policy](freshness.md) before selecting build inputs, creating a baseline or promoting. Recorded digests preserve the freshly resolved attempt; older frozen-input instructions do not authorize obsolete dependencies for new work.

How developers produce and publish releases. Durable model:
[Release architecture](../architecture/release.md). Tool contracts:
[Native support](native-support.md).

Payload v4 separates the host, application images and native extension packages.
Fountain owns package acquisition and requested enablement; Soda supplies its
packages and restricted host operations. See the
[component contract](../architecture/release.md#public-distribution) and
[Tailnet lifecycle](../architecture/networking.md#tailnet-model).

Host updates follow [host update ownership](../architecture/release.md#host-update-ownership).

## Pipeline map

| Symptom or task | Open first | Then |
| --- | --- | --- |
| Candidate will not build | `lib/soda-release-build`, `lib/soda-release-tools` (`soda-build`) | `lib/soda-release-image` |
| Media/assemble wrong | `lib/soda-release-image` | `cmd/soda-install` |
| Guest/fixture behavior | `tools/acceptance` | `tests/installed` |
| Publish/promote | `lib/soda-release-deliver` | `soda-release` in `lib/soda-release-tools` |

## Guided release operator flow

Select the release commit after its intended integration is complete. That commit,
the Fountain pin, version and distribution set define this release; later source
work does not silently change a running attempt. Run the current admitted tools'
preflight before starting production work.

An active workflow reserves the paths in its `request.json`: build output, prepared
bundle and required worker caches. Cleanup must inspect that request and preserve
these inputs while building or awaiting publication. A lack of open file descriptors does not make prepared media
disposable. Release the reservation after promotion or explicit abandonment;
failed experiments do not require indefinite retention. Keep Go, Cargo and Podman
caches separate from disposable attempt scratch, and remove them only through an
explicit cache-reset decision. Removing them makes the next build cold.

The same prepared release carries its selected `appliance`, `docker` or
`all` targets through build, stage, smoke-install and promotion. The
[release architecture](../architecture/release.md#distribution-targets) owns
artifact identities; the
[Docker installation guide](../guides/docker.md) owns the user path.

### 1. Select and preflight

Run `soda-release start` from the clean committed checkout or admitted worktree.
The guided overview shows distribution, exact version, restricted worker config,
notes and fresh workflow output together; edit a field by number and use `go`.
Flags preseed the same flow. `--non-interactive` requires every input.

The admitted worker identifies its source, Fountain checkout, controller, cache,
tools and output parent. The selected Source must be the current checkout, with
workflow and build output below its `.artifacts/`. Build home and runtime stay
inside the admitted private storage root; linked worktrees additionally require
those paths below their own `.artifacts/`. Admission
and worker setup follow [native support](native-support.md#local-host-content-image-candidate);
start does not install accounts, grant sudo or admit an executable.
Run `start` with sudo: the admitted controller and restricted worker configuration
require root. Keep GitHub authentication outside the worker. Set `GH_CONFIG_DIR`
to the existing operator GitHub configuration directory and `TMPDIR` to a private
temporary directory on the workspace disk, then preserve those two variables
through sudo; never move credentials into build inputs.
The prepared directory is private and root-owned, so use the same sudo context
for its subsequent commands and a root-owned registry auth file. The examples
below preserve that ownership consistently.

Before building, start checks committed source and Fountain pin, supported native
architecture, the locked offline Canlang archive, valid notes, fresh paths, Podman
and (for Docker) Skopeo,
exact-source main CI, immutable-release settings and existing tag/release conflicts.
It reports missing/private GHCR visibility prerequisites and workspace capacity.
Appliance/all preflight reserves at least 75 GiB of media scratch plus the native
OSTree import's 3% workspace free-space margin; cold compiler/image caches can
require additional capacity. Docker-only does not reserve appliance media scratch.
The admitted controller's `--preflight` checks its own revision, byte identity
and restricted worker configuration without dispatching a build. Missing prerequisites
fail before compilation. `start --preflight` performs these checks without building
or creating an attempt. Build failures remain failed, with retained diagnostics.

```sh
sudo --preserve-env=GH_CONFIG_DIR,TMPDIR /ADMITTED/soda-release start \
  --distribution docker --tag v0.1.2-rc.1 \
  --worker-config /home/BUILD/worker.json \
  --notes /home/BUILD/notes.md \
  --out /home/BUILD/WORKTREE/.artifacts/releases/workflow \
  --non-interactive
```

### 2. Build and prepare once

Start delegates to the admitted `soda-build`, reusing its progress display.
Docker requires an explicit release tag and skips media; all derives the same
tag's GitHub rootfs URL and packages both targets from one common candidate.
There is no partial-host, resume or skip-tests build path.

The workflow's private `request.json` records selected inputs, source identities
and actual output paths. The build lives under the worker OutputParent with a
fresh workflow-derived name; its prepared directory is `WORKFLOW/release`.
Preparation requires the successful completed production build and verifies the
source archives, exact target artifacts and generated checksums. Docker public
assets are `docker.json`, `docker-packages.txt`, `sodaos-docker-compose.yaml`
and `install-docker.sh`, beside shared provenance and both source archives.
Preparation renders the installer from this candidate's archived source with its
exact release tag, source revision and Docker metadata hash/size. The installer
enters the same immutable inventory and checksums; verification reconstructs it
from the archived template to reject a substituted script.
Appliance/all add media metadata, ISO and hash-named rootfs. All GitHub assets must
fit the existing per-asset limit. The large Docker OCI archive stays local;
private `operator.json` points to its verified candidate and is never uploaded.

Start stops with a **local prepared release**. It never publishes or
deploys. Manual `prepare --candidate ... --distribution ... --tag ... --notes ...
--out ...` uses the same preparation for an existing production candidate.
Use the prepared directory as the input to every subsequent command.

```sh
sudo --preserve-env=GH_CONFIG_DIR,TMPDIR /ADMITTED/soda-release status \
  --input /home/BUILD/WORKFLOW/release
```

Status rechecks local asset bytes, original downloads and retained observations,
and reports source CI, observed remote release, registry identity, blockers, promotion attempt state and next action. Drafts, mutable releases and
failed remote reads cannot authorize the next publication or promotion step.
It makes no registry, GitHub or installation mutation.

### 3. Stage the complete candidate

Keep publication credentials outside the worker. Supply a restricted registry auth
file for every distribution. An existing authenticated gh account with package write access
can create one without putting the token in argv or logs:

```sh
set +x
umask 077
gh auth token | sudo --preserve-env=TMPDIR skopeo login --username "$(gh api user --jq .login)" \
  --password-stdin --authfile /home/BUILD/registry-auth.json ghcr.io
sudo --preserve-env=GH_CONFIG_DIR,TMPDIR /ADMITTED/soda-release stage \
  --input /home/BUILD/WORKFLOW/release \
  --auth-file /home/BUILD/registry-auth.json
```

Stage rechecks exact-source main CI, tag, remote release conflicts and all local
hashes before publication. Every distribution uses Skopeo's digest-preserving
upload for `sodaos-dashboard`, `sodaos-forgejo`, `sodaos-project-os` and
`sodaos-canlang-runtime` under `ghcr.io/levitateos`, with the exact release tag.
Anonymous readback binds each tag and digest to Payload v4's manifest/config and
native platform. Proxy, PostgreSQL, the independent Fountain release and optional
Tailscale are checked anonymously at their upstream digests and are never republished.
The `components/public.json` receipt records successful readback; a partial attempt
requires inspection and `verify-images` rather than another upload.
Docker also uploads to
`ghcr.io/levitateos/sodaos:VERSION`, then reads the raw version manifest and
its digest-selected config anonymously. Only matching source/base/platform and
manifest/config identities produce the local `registry/registry.json` receipt.
Stage publishes the complete asset set as one immutable GitHub prerelease.
It does not move either recommendation pointer.

First-time GHCR packages default to private. If upload succeeds but public readback
fails, inspect the uploaded version and change its package visibility in GitHub.
Then finish readback without another upload:

```sh
sudo --preserve-env=GH_CONFIG_DIR,TMPDIR /ADMITTED/soda-release verify-images \
  --input /home/BUILD/WORKFLOW/release
sudo --preserve-env=GH_CONFIG_DIR,TMPDIR /ADMITTED/soda-release status \
  --input /home/BUILD/WORKFLOW/release
```

Verification can be repeated: matching public bytes complete interrupted local
verification records without uploading again. A conflicting local record is refused.
After successful registry verification, stage can publish the still-absent GitHub
release using that same public version. Existing GitHub releases are refused;
failed/uncertain draft or promotion writes require inspection, not blind replay.

### 4. Smoke-install the actual public bytes

Use a fresh disposable supported Linux host or VM and the staged prerelease's
public downloads; a cached development image or repository Compose file cannot
substitute.

- **Docker:** run the versioned release's public streamed `install-docker.sh` with
  `bash -s -- --qualification` against the immutable prerelease (see
  [Docker installation](../guides/docker.md)). The normal command accepts only a
  promoted release with the Docker marker. Log in, open a Project and a terminal,
  then restart and recreate the container with the same image.
- **Appliance:** download the staged ISO, install it on a fresh native KVM disk,
  reboot without the media, and read back the installed Soda revision and image
  identities.

Note the installed revision and the result in the release notes. A failed smoke
leaves the prerelease unrecommended; fix the cause and cut a new tag.

### 5. Promote

```sh
sudo --preserve-env=GH_CONFIG_DIR,TMPDIR /ADMITTED/soda-release status \
  --input /home/BUILD/WORKFLOW/release
sudo --preserve-env=GH_CONFIG_DIR,TMPDIR /ADMITTED/soda-release promote \
  --input /home/BUILD/WORKFLOW/release \
  --auth-file /home/BUILD/registry-auth.json
```

Promotion rechecks current CI, the immutable remote assets and the public image
digests before changing anything. It copies the existing public digest to GHCR
latest with preserved bytes, adds the line
`<!-- soda-docker-qualified: TAG SOURCE_SHA IMAGE_DIGEST -->` to the notes for
Docker (the installer requires it), and marks the release latest. Docker-only
uses `--latest=false` for GitHub and confirms the appliance latest pointer is
unchanged. GitHub and GHCR mutations are not atomic: promotion records its
attempt first, refuses replay, and leaves partial state for `status` to report.

## Development builds

- Use the smallest applicable development target while iterating.
- Do not relabel a failed release run as successful.
- Private registry mirrors and fixture package origins prove development
  behavior, not public availability.

`project-terminal` is compiled once during production asset staging. Host assembly
copies that artifact to its runtime and maintenance paths and checks its digest
against the selected built Project image. The release payload's `ProjectSupport`
binding follows the same candidate through media and GitHub delivery; delivery
also inspects the helper content in the host and Project OCI archives.

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

This handoff does not authorize a build, installation or publication. The named runtime image follows the existing image
catalog, OCI export and offline import owners described in
[release architecture](../architecture/release.md#public-distribution).
Application images remain separately assembled artifacts.

Use clean committed Soda and Fountain sources and the admitted native build worker.
The Fountain checkout must match `fountain-source.json` in that Soda revision.
Push Soda to `main` and wait for its `CI` workflow to pass before the expensive
native build. `soda-release` independently checks the latest push run for that
exact commit and the Fountain pin before staging and again before promotion.
Missing, pending, skipped, cancelled or failed CI cannot authorize delivery;
a PR's synthetic merge commit cannot qualify a different release commit.

Protected-main qualification tags use `ci/soda-main-FULL_COMMIT_SHA`, created by
`bash scripts/push.sh` as documented in [GitHub CI](testing.md#github-ci). The tag runs all source suites. Its
subsequent first main-push run reuses only the latest successful push run of that
exact tag, commit and CI workflow. The main CI gate still runs and is the receipt
checked by release admission. Missing, failed or pending prior checks execute the
suites normally. PRs, tag runs and explicit CI reruns always execute the suites.
Only the controller crate's environment/TCP fixtures serialize Rust test threads;
the rest of the workspace tests run in parallel.

GitHub Free hosts source CI. The existing local native builder owns OS builds and
the guided installation smoke; no self-hosted Actions agent or custom download
server is required. GitHub credentials remain with the operator running
`soda-release`, outside the isolated build worker. CI has read-only repository
access and receives no release keys or publication credentials.

The required `CI` result includes the Docker packaging job. It runs
`bash scripts/check-docker-packaging.sh`: checksum-pinned [Hadolint](https://github.com/hadolint/hadolint)
checks the Dockerfile, and Compose's parser verifies the supported platform,
runtime settings and persistent volumes. The Rust workspace job covers Docker
metadata/provenance, candidate reuse, development/publication refusal and worker
and CLI admission. New release components add their checks to these gates as
they land. These source/fixture checks build no OS image and do not replace
the smoke install of the published image.

Choose the release tag before building: the installer embeds the exact versioned
GitHub rootfs URL. For example, with tag `v0.1.0`:

```sh
sudo /ADMITTED/soda-build --worker-config /RESTRICTED/worker.json \
  --arch x86_64 --out /OUTPUT_PARENT/fresh-release \
  --forgejo-source /CANONICAL/FORGEJO-CHECKOUT \
  --rootfs-base-url https://github.com/LevitateOS/sodaos/releases/download/v0.1.0
```

The build emits a media candidate. It never publishes or silently
installs on the builder. Omit the development `fast` compression mode for releases.
Staging enforces the 2 GiB per-asset limit before upload.

For local release preparation without publication, supply release notes and a
new output directory:

```sh
soda-release prepare --candidate /BUILD/artifacts \
  --tag v0.1.0 \
  --notes /PRIVATE/candidate-notes.md --out /PRIVATE/new-release
```

This checks and copies the complete bound asset set using the same preparation
as `stage`, including asset limits. It does not contact GitHub, create a tag or publish.

To publish, enable immutable releases once in the repository settings. With the
installed `gh` client authenticated for this repository, stage the prepared directory:

```sh
soda-release stage --input /PRIVATE/new-release
```

Preparation binds the media and hashes the complete release asset set. Staging
rechecks those prepared bytes and source CI. It creates a missing tag at the exact built
commit, refuses a tag pointing elsewhere, then uploads a draft, checks the remote
inventory and publishes a prerelease. Notes identify it as a candidate.
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

After that smoke passes, promote:

```sh
soda-release promote --input /PRIVATE/new-release
```

Promotion rechecks the locked remote inventory, records the CI run in the
release notes and changes the same release to latest. This is the website download:
`https://github.com/LevitateOS/sodaos/releases/latest/download/sodaos-x86_64.iso`.
Use versioned URLs in the installer and latest only for the website's default CTA.
Source checks, artifact admission and the native smoke each cover a different
boundary; do not recursively rerun them after they pass without a relevant change.

<a id="fountain-upstream-maintenance"></a>

## Fountain upstream maintenance

The current Soda source pin selects [Fountain 15.0.10-fountain.3](https://github.com/veighnsche/forgejo-ext/releases/tag/v15.0.10-fountain.3),
revision `fd5d4db12daf291cdaa5cbde8cb9dee52a0e2b11`. Use its immutable
[`release.json`](https://github.com/veighnsche/forgejo-ext/releases/download/v15.0.10-fountain.3/release.json)
as `SODA_FOUNTAIN_RELEASE_URL` when preparing the builder. The existing input
resolver binds its image, corresponding sources and protocol declarations;
do not substitute the earlier local candidate image digest. This source update
does not update an admitted worker configuration or install the release on a VM.

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

Build the candidate from committed Soda source and the selected independent
Fountain release; local SDK bytes must match Soda's committed Fountain pin. Before
promotion, verify that `artifacts/forgejo-source.tar` is the archived Fountain
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
installed readback. The independent Fountain release metadata binds its image,
source and prepared source archives; Soda verifies those inputs rather than
compiling Fountain inside its own candidate build.

Local maintenance commits and development candidates do not publish anything.
Pushing a fork, publishing artifacts, signing a release or activating production
remains a separate explicitly authorized action.

## Commands

Follow the admitted interfaces in [Native support](native-support.md) for
candidate production, media, export and evidence. Carry out build, publication and
disposable-disk installation when the task authorizes those effects; a source-only
implementation task does not itself authorize remote publication.
