# SodaOS release engineering

## Status and decisions

**Milestone 1 is complete at native x86_64 local-build/inspection scope (`45ac843`).**
The complete host/app payload is exported; bootc lint passes with 13 checks and one
skip, and all 391 immutable Forgejo files plus embedded runtime archives are verified.
[Receipt](implementation-history.md#complete-local-appliance-candidate).
Six milestones carry the work through automated production operation; the 28 detailed
items are acceptance criteria, not separate execution/approval rounds. **The original
local exports remain unsigned; signed copies were staged in internal GHCR packages.
Neither is an install/boot/upgrade-qualified release.** Milestone 2 now
has source tooling and native filesystem Sigstore proof, including synthetic-key
signing of that exact host, all five apps and its release document. Root-protected real keys and eight GHCR namespaces are now provisioned using the
owner-selected current `gh` account. Authenticated native signature/digest round trips
pass. GitHub created the packages as **internal**; public visibility, anonymous proof
and mutable candidate-channel promotion remain pending. The
owner selected GHCR distribution and a CoreOS-aligned Soda release train with an
independent emergency lane. This guide owns the release engineering workstream and
its status: build engineering, release management, distribution and appliance updates.
Appliance updates are part of this workstream, not a separate platform. The owner
approved the first source/upstream investigation; its
[receipt and recommendation](release-engineering-feasibility.md) propose derived
FCOS with bootc's OSTree backend and logically bound core app images for proof.
This is not a validated migration or production mechanism. Exact action grants,
including the bounded candidate commissioning already performed, belong to the
[handoff](implementation-status.md#current-permissions), not to a command or plan.

**Current implementation direction:** the owner rejected the `d054a60` extraction
as insufficient: it retained two assemblers and increased production orchestration.
The requested replacement is [one source-to-release run](#single-run-release-build-contract),
including installer media consuming the same host/application candidate as updates.
The [ordered implementation packages](#single-run-build-replacement-implementation)
below replace that transitional design. This pass specifies the implementation;
it does not claim the replacement exists or grant external/lifecycle effects.
The predecessor repository and its reserved Updates platform remain separate.

Selected product decisions:

- Stage qualified Soda changes while following Fedora CoreOS **stable** releases.
- A new stable base triggers a Soda candidate combining that exact base with ready
  Soda changes. Qualification precedes promotion; there is no same-day promise.
- Publish signed production releases through **GHCR**. Appliances pull approved
  updates; publishing does not immediately reboot every appliance.
- Automate the normal detect → freeze inputs → build → test → sign → publish →
  progressive promotion pipeline. Initial manual promotion is a commissioning
  safeguard, not the intended permanent release process.
- Use this machine as the proposed initial release builder, subject to host/resource
  and isolation inspection. A systemd timer and one-shot job poll upstream metadata;
  no continuously running listener or inbound webhook server is required.
- Support emergency Soda and CoreOS releases independently of the normal cadence.
- Prefer a derived bootable Soda host image, with application images pinned by the
  same release. This is the target to validate, not a selected bootc migration or
  proof that the current installation can consume such an image.
- Reuse upstream build/deployment/update mechanisms. Do not build a fleet controller
  or custom update server when registry artifacts and native clients suffice.

CoreOS stable commonly releases approximately fortnightly, but urgent fixes and
upstream delays vary that cadence. Consume actual stable release identity, not a
calendar-based assumption. Unfinished Soda features must never be a prerequisite
for qualifying a security update to the base.

## 1. Scope and ownership

One appliance release identifies a tested host/application combination. It need not
change every component. Updating marketplace applications, mutable Project OS roots
or CI job environments is not implied by updating the host.

| Owner | Responsibility |
| --- | --- |
| This plan | Release identity, qualification, channels, activation and update recovery design |
| [OS strategy](os-product-strategy.md#update-ownership) | Product-level host/application ownership |
| [Installation](installation.md) and [installer](coreos-installer-plan.md) | First install, media and retained-appliance migration procedures |
| [Native support](native-support.md), source manifests and locks | Build inputs, artifact inventory and verification |
| [Native validation](native-validation.md) | Reused product journeys and architecture-specific evidence |
| [Credentials](dashboard-credentials.md) | Schema compatibility, consistent secret/database preservation |
| [Project OS](project-os.md) | Persistent roots and separately selected project maintenance |
| [Services](services-and-ai-plan.md) | Marketplace recipe and app lifecycle |
| [Implementation status](implementation-status.md) | Single-run replacement progress, release custody and active grants |
| [Development handoff](development-handoff.md) | Other retained targets and their separate scoped grants |

Out of scope: importing the predecessor updater, a new kernel/package manager,
public appliance ingress, fleet command execution, telemetry by default, arbitrary
app rollback, project-root replacement, general disaster recovery or high
availability. Appliance updates are disruptive maintenance, not live migration.

## 2. Target architecture and feasibility decision

```text
CoreOS stable release ─┐
Ready Soda changes ────┼─> pinned candidate -> qualification -> signing -> GHCR
Emergency fix ────────┘                                      |
                                                   approved channel
                                                            |
                                               appliance pull/verify/stage
                                                            |
                                                maintenance activation
                                                            |
                                                 post-activation checks
```

Investigate the selected current CoreOS release, actual Soda files/callers and
matching upstream documentation before writing an adapter. Compare:

1. **Preferred: derived host image.** Include host packages and Soda host integration
   in a versioned bootable deployment. Determine which programs/units can live in
   immutable OS content, how defaults enter writable configuration without replacing
   operator edits, and how pinned app images are installed/started.
2. **Fallback: upstream CoreOS plus coordinated Soda bundles.** Retain today's native
   OS deployment and separately deliver Soda. Document ordering, partial completion
   and recovery costs explicitly. Do not silently adopt this if the preferred path
   fails; bring back the concrete tradeoff for a decision.

The feasibility receipt must resolve:

- Supported derivation/build path for the exact upstream base and layered packages.
- rpm-ostree versus bootc transport/activation, including the existing signed-image
  origin and whether migration is needed. OCI transport alone does not choose bootc.
- GHCR host/application artifact support, digest pinning, architecture selection,
  authentication and signature verification by the actual installed consumer.
- One owner for staging/finalization/reboot; what happens to Zincati. It must not
  independently activate unqualified upstream updates or race another owner.
- Filesystem/configuration ownership across `/usr`, `/etc` and `/var`; current
  `/usr/local` and writable Soda payload placement cannot be assumed transactional.
- Boot selection/fallback and pinned app compatibility after interrupted activation.
- Root CLI/console recovery without working Forgejo, Soda UI or Tailnet access.

### Local candidate content and machine-state ownership

`--complete` selects the image-owned layout without altering the writable installer:

- `/usr/share/soda/release.json` owns the five exact image selections, source/base,
  schema and content hashes. Three core Quadlets link into
  `/usr/lib/bootc/bound-images.d`; **only their callers** add
  `/usr/lib/bootc/storage` as an additional image store. Publication must preserve
  the exported OCI manifest digests; the GHCR round trip remains milestone 2 proof.
- Project OS and Tailnet archives live in `/usr/share/soda/images`. The root-only,
  fixed-input `soda-image-import` verifies them and imports missing image IDs into
  ordinary Podman storage before the host helper/project start. It neither starts,
  replaces nor deletes containers/images. Existing project roots and companion
  backing layers remain outside bootc GC; host updates do not select their lifecycle.
- Vendor helpers derive **new creation** image defaults from the immutable record.
  `/etc/soda/host.json` still supplies machine network/Tailnet settings. Conflicting
  saved image selections refuse with an explicit-migration error; they are never
  silently rewritten. Existing project/profile IDs remain authoritative for retained
  roots. Companion admission still binds the run-scoped CID and exact CreateCommand
  to the configured immutable image: changing that image during an existing run
  refuses, rather than adopting/replacing the companion. `/run` receipts are not a
  persistent-image migration protocol. Native update proof must cover coordinated
  run turnover; no hot companion-image upgrade is qualified.
- `/usr/share/soda/defaults` contains public examples/defaults, not credentials.
  Initial public Forgejo/proxy defaults also enter the image's `/etc` using native
  OSTree configuration merging. Machine JSON, activated identity, TLS, databases,
  accounts and `/var` roots are not built into the candidate. The empty subnet in
  the example intentionally requires first-install selection; this is not a working
  preinitialized host configuration or a retained-install migration tool.
- The [Forgejo customization owner](forgejo-frontend-integration.md#image-owned-presentation)
  defines its immutable presentation/data split. `payload.json` is the exact embedded
  record; detached `candidate.json` adds the resulting host manifest and the payload
  byte hash without a circular self-digest. Neither is signed channel authority.
- The x86_64 RPM lock pins all added NEVRAs and the full expected inventory from the
  retained build, including the selected base. Exact versions must still be available
  from the configured Fedora/Tailscale repositories. Missing versions fail closed;
  RPM bytes/repositories are not mirrored and byte reproducibility is not claimed.
  Aarch64 requires its own transaction lock and native evidence.

**Separate defect:** the reported Cockpit error treating
`ostree-image-signed:docker://quay.io/fedora/fedora-coreos` as a remote requires exact
installed Cockpit/rpm-ostree/origin diagnostics and matching upstream source review.
It is not proof the base image is missing. A GHCR URL does not itself repair it.
Do not add a fake OSTree remote, rebase the target or weaken signature policy to
make the page stop reporting an error.

## 3. Release identity and GHCR layout

The selected GHCR namespaces and observed commissioning scope are recorded under
[selected custody](#implemented-trusted-delivery-contract). Public pull is the
appliance target so installation does not require a personal GitHub token. Finish
visibility, licensing, bandwidth, retention and anonymous-client commissioning;
namespace creation alone is not public delivery proof.

Use immutable digests for host images, application images and release metadata.
Human-readable release tags are conveniences, not execution authority. Suggested
repository roles (names are placeholders): `host`, component images and `releases`.
Confirm native metadata/artifact support before selecting its wire format.

A signed release record binds at least:

- Unique Soda release ID and normal/emergency classification.
- Exact upstream CoreOS version/digest, source revision and build inputs.
- Architecture-specific host and application digests and required package inventory.
- Supported source releases, required intermediate upgrades and schema compatibility.
- Activation/reboot requirements, migration steps and recovery limitations.
- Checksums/digests for matching installer/download/offline artifacts, where produced.
- Build provenance, qualification evidence references and release notes.

Prefer upstream manifest/provenance formats where sufficient; add only Soda's actual
compatibility contract. Never put credentials, initialized databases or machine
identities in generic images. Preserve attribution/licenses and produce dependency
inventory/SBOM information suitable for security triage.

Channels: **candidate**, opt-in **preview**, and **stable**. Channel promotion selects
already-built and qualified digests, not a rebuild. Maintain explicit architecture
availability: x86_64 evidence is not aarch64 proof. Unsupported architectures or
upgrade paths receive an honest unavailable/blocked result, not another payload.

A stable Soda domain is optional indirection for future discovery/download links;
GHCR is the selected artifact host. A dynamic service is not an initial requirement.

## 4. Trust and publication

Production signing and appliance verification are mandatory. The implemented native
scheme is **keyed P-256 Sigstore using skopeo/containers policy**, not a custom
cryptographic envelope or assumed keyless-CI integration. Production identities and
bootstrap installation still require their exact provisioning grant.

Define signing-key/identity custody, protected release authority, backup where
applicable, rotation with overlap, revocation and recovery from compromise. Keep
build and promotion permissions narrowly scoped. Pin any adopted CI dependencies.
Never place long-lived signing or registry credentials in source, argv or logs.

Verify the host, app artifacts and their release binding before mutation. A valid
image signature alone does not establish that it is the latest approved channel
release. Define channel freshness, replay/downgrade protection and deliberate root
recovery overrides using existing mechanisms where available. Preserve intentional
boot fallback without silently accepting arbitrary old releases from discovery.

Publication failures leave the existing channel valid. Upload and verify immutable
artifacts first; promote the complete release reference last. Test interrupted
publication and missing per-architecture content. Define retention for supported
upgrade intermediates and recovery artifacts; do not assume mutable tags retain
otherwise unreferenced images. Never prune retained fixture or registry evidence
without exact cleanup approval.

### Implemented trusted-delivery contract

`internal/releasedelivery` and `tools/soda-release` implement these contracts;
[native support](native-support.md#trusted-release-delivery-worker) owns invocations.
`internal/releasedelivery/tools.json` owns the selected skopeo/client source versions.
The worker never installs an image, imports into Podman, changes host policy or reboots.

**Native signatures and roles.** Artifact and release repositories use an artifact
key role. Candidate, preview and stable each have a separate channel key role and
repository (`PREFIX-channel-NAME`); roles cannot share keys. The other repositories
are the M1 `PREFIX-{host,dashboard,forgejo,proxy,project-os,tailnet}` plus `PREFIX-release`.
The six images, release and candidate-channel namespaces have been staged; preview
and stable namespaces are not implicitly provisioned by this build rewrite. Each native policy requires
`sigstoreSigned`, explicit `exactRepository` identity and the configured public keys.
Repository-scoped `use-sigstore-attachments` is enabled in generated worker config.
The proposed appliance policy preserves unrelated vendor rules and refuses existing
Soda-specific overrides for review; it is never automatically installed.

The native directory transport supports local signatures as well as registry
attachments. Signers snapshot public inputs into a private attempt, remove signatures
only from that new snapshot, check the admitted manifest digest, sign it, and verify
with the required role. An existing good signature cannot mask a wrong signer. No
Rekor/keyless identity service or signing prompt is required. The selected upstream
registry writer appends nonduplicate Sigstore signatures, supporting overlap without
removing old registry signatures; actual GHCR behavior still needs commissioning.

**Release and channel documents.** Small, non-executable standard OCI images carry
one `record.json`; native skopeo owns signatures and transport. A release embeds the
exact M1 payload/candidate bytes, serial, normal/emergency classification, provenance
hashes, public qualification evidence and notes. It does not maintain a second image
inventory. Native OCI verification and M1 hashes check preparation inputs. A local
qualification assertion is not signing authority and cannot enter preview/stable;
those require the `native-install-upgrade-recovery` scope backed by actual later
milestone evidence. A protected worker must independently admit the exact document
digest: a build's self-reported “passed” JSON never grants signing credentials.
Changing frozen release metadata requires a new serial; preview-to-stable promotion
uses the same already-qualified release document and image digests, not a rebuild.

A channel binds its name, strictly increasing sequence, issued/expiry times and exact
per-architecture signed release references, or an explicit withdrawal. Trust inputs
supply a bootstrap `NotBefore`, minimum sequences and freshness limits (60 seconds
through seven days; clock tolerance at most five minutes), not an invented release
SLA. Authenticated, credible system time is a deployment prerequisite; backward or
out-of-range time fails closed. Expiry requires the pipeline to refresh offers with
a higher sequence, not rebuild their images. Expiry/withdrawal stops new uptake; it
does not roll back installed data or deployments.

**Freshness and recovery.** Private high-water state records the highest accepted
trust epoch, signed channel sequence/digest and observed release serial/digest per
architecture. Same-sequence substitutions, older serials, same-serial forks, expired
offers and trust rollback refuse; re-observation of the identical fresh offer is
allowed. Authenticated channel/release observations remain recorded even when later
content is unavailable. These are **observed authority**, not installed-state fields.
Missing/corrupt state never silently bootstraps; initialization refuses existing
state. There is no generic ignore-signature/downgrade/reset flag. Deliberate native
boot fallback remains separate M3 recovery work and must not rewind these floors.

**Publication and failure.** Signing and publishing are separate noninteractive
operations. Exact-digest permits come from the protected qualification/promotion
owner; the future scheduler supplies them automatically. Public inputs are not
executed with secrets. Signer/passphrase, permit and registry credential files are
restricted; native subprocesses discard ambient credentials/configuration and raw
error output. Public trust and worker command arguments must be integrity-controlled
by that owner, not selected by an untrusted build job. Actual UID/credential isolation
and custody are not proved merely by passing file-mode tests.

Namespace bootstrap stages only immutable signed content and checks it with the
publisher's authenticated native client. It does not move a channel tag or claim
public readiness. GitHub package visibility must then be set to Public through its
supported settings UI; repository permission inheritance does not confer public
visibility. Routine publication uses native digest preservation and copies artifacts
back with signature policy and **anonymous** access. A channel promotion verifies all
advertised architectures and every referenced artifact before uploading its immutable
OCI document/signature. Native tag listing distinguishes bootstrap absence from
transport/auth failures; an authenticated previous digest must match the permit and
preserved publisher history. Only then does the mutable channel tag move. Readers
discover a tag's digest, verify the pinned document, validate authority, then verify
the selected architecture's images through fresh native copies—not cache existence
or an inspect-only call. Unsupported architecture never falls back to its sibling.

GHCR tag writes have no CAS in this implementation. Use **one protected publisher
and persistent locked ledger per repository**, with no external competing tag writer.
The durable intent is recorded before writes. An uncertain operation is held; a
separate read-only observation can confirm completion without replay. If the desired
commit is not observable, preserve the hold/evidence for explicit reconciliation,
not a fresh ledger or blind retry. Failure handling may require operations attention;
routine successful releases require nobody to attend or sign them.

**Selected commissioning custody.** The owner confirmed control of `LevitateOS`,
selected the current `gh` account (`veighnsche`) and approved straightforward protected
signing setup. The approved namespaces are `ghcr.io/levitateos/sodaos-*`, candidate
channel only for this commissioning; the predecessor `soda-os` package is untouched.
`/var/lib/soda-release` is root-only retained builder state: reviewed worker binary,
separate encrypted artifact/candidate/preview/stable keys, private passphrase files,
current-account registry credential copy, permits and retained attempts. Public trust
is recorded in [`appliance/keys/release-trust.json`](../appliance/keys/release-trust.json),
not installed into host policy or obtained from the channel it authenticates.
No extra account, service or timer was created. Ordinary unprivileged key/credential
reads refuse; the trusted administrator still has sudo and its existing broad `gh`
credential. This is **not isolation from untrusted jobs running as that administrator**.
M4 must keep such jobs away from the signing/publishing identity, process state and
privileged command authority; no new GitHub identity is required for commissioning.
An encrypted-key backup is retained locally, with passphrases separate. Off-machine
recovery custody and its authorized drill remain outstanding; a same-machine backup
is not builder-loss recovery. Public trust bootstrap is delivered by the approved installation/update
owner, not downloaded from the channel it is meant to authenticate. Rotation adds a
new same-role key and increments the trust epoch, re-signs retained supported content,
then removes the old key only after qualified overlap. Revocation, offline clients
and key/ledger loss need the authorized M3/M5 recovery drills. Real immutable namespace
bootstrap has occurred, but no global host policy, mutable candidate/stable offer,
worker service, timer or appliance activation has been installed/enabled.

## 5. Normal and emergency release process

### Single-run release build contract

**Target, not current behavior:** one reviewed Go entrypoint, `tools/soda-build`
(compiled as `soda-build`), owns the source-to-release invocation. It replaces the
native-bundle and host-image producers; it must not wrap both. Normal and emergency
triggers provide different admitted inputs to the same fixed sequence. There is no
legacy/vendor build switch, host-content-only release mode, task graph, plugin
system or custom scheduler. Development leaf tools may remain, but do not produce
another kind of SodaOS release.

Proposed public interface (not executable today):

```sh
soda-build --arch x86_64 --out "$PWD/.artifacts/releases/UNIQUE-RUN"
soda-build --arch x86_64 --out "$PWD/.artifacts/releases/UNIQUE-RUN" --publish candidate
```

Both requests build and qualify the same complete candidate and ISO. Omitting
`--publish` prevents remote publication; it does **not** skip signing/native tests
and call the result a qualified release. Full runs require precommissioned signer
and exact qualification resources; preflight refuses missing authority/resources
before expensive work. Source/unit fixtures are still independently runnable, with
honest local-test scope, not a second release lane. No `--skip-tests`, arbitrary
resume-at-phase or success-on-partial-output switch is part of the release interface.
The invocation starts from clean canonical committed source; freeze with `git archive`,
never create a worktree or move the checkout automatically.

| Phase | Work and exact handoff | Completion evidence |
| --- | --- | --- |
| P1 Admit and freeze | Resolve approved source/base/architecture, package and tool inputs, source identity and baseline releases; reserve one release identity through the protected owner; check disk/tool/fixture/signing/publication prerequisites. | Frozen input record and source snapshot; no mutable ref resolution later in the run. |
| P2 Dependencies and source checks | Frozen Bun install, Go dependency verification, Go/unit/static/type checks that do not need generated browser assets. | Source results tied to the snapshot; missing fixture prerequisites are not silently skipped qualification. |
| P3 Programs, assets and prepared tests | Compile shipping programs, media console and necessary tools once; produce browser/locale/branding/terminal/Tea assets; run browser/product source checks against those exact generated assets. | Program/asset identities and prepared-test receipts; tests do not invoke another asset build. |
| P4 Application images | Produce or explicitly reuse the five selected application images; verify native platform, content, source provenance and immutable identities. | Five verified images plus their existing payload identity fields. |
| P5 Host image | Assemble the image-owned filesystem directly, bind core apps, include retained-runtime archives, add host packages to pinned CoreOS, build the host. | One bootable-host OCI candidate; no intermediate writable installer payload. |
| P6 Candidate verification | ELF/OCI/content/inventory/Quadlet/bootc checks; freeze host and application identities. | Existing payload/candidate record extended only for required new provenance; no qualified upgrade edges inferred. |
| P7 Candidate signing | Protected owner independently admits exact artifact digests after static checks; native signer authenticates those bytes for installation/signature testing. | Native signatures; artifact authenticity is not preview/stable approval. |
| P8 Installer assembly | A media-only leaf consumes the signed candidate, prebuilt console/tools, public trust/bootstrap inputs and selected live media. No Go/app compilation or second OS assembly occurs here. | ISO, candidate/install descriptor, readback evidence and ISO checksum. |
| P9 Native qualification | Install the actual ISO; test first boot, applicable baseline-to-candidate updates, state preservation, maintenance activation and recovery under exact fixture grants. | Protected qualification results bound to candidate digests and ISO checksum. |
| P10 Finalize release | Create final release metadata binding candidate, media checksum/location, compatibility and P9 evidence; sign it using existing authority. | Signed final release document; host, app and ISO bytes unchanged since testing. |
| P11 Deliver and select | When authorized, upload immutable artifacts/media, verify anonymous delivery and signatures/checksums, then publish the signed channel selection last. | Delivery receipts and existing durable publication ledger; local runs finish qualified-but-unpublished. |

**No signing/qualification cycle.** The ISO contains signed candidate identities and
integrity-controlled bootstrap trust, not a report that depends on booting that ISO.
The final signed release document adds the resulting ISO checksum and qualification
report afterward. Candidate artifact signing has a narrower admission gate than
channel promotion. A build-supplied `passed` file cannot issue either permission.
Trust bootstrap remains independent of the artifact/channel it verifies; the signed
final record authenticates the downloaded ISO before its first privileged use.

**Installation output.** The derived host image is both the first-install and update
artifact. The desired general ISO contains the content needed to install that exact
candidate without a registry dependency during disk installation/first boot; marketplace
and provider networking are not part of that offline claim. Prove the selected native
bootc path for host plus bound/retained images before implementing an adapter. Reuse
stock live-media boot and upstream customization tools; `coreos-installer` media
customization is not evidence that its existing offline disk installer can install a
derived OCI host. No fallback to stock CoreOS plus `install-native.sh` is permitted
inside the new run. The [installer owner](coreos-installer-plan.md#image-based-replacement-contract)
owns disk/UI/bootstrap and media constraints. A concrete upstream limitation requires
an explicit product decision, not another hidden build lane.

**One run, not one privilege domain.** A reviewed, protected controller launches
unprivileged production, confined native qualification and fixed signing/publishing
operations with separate credentials/authority. Neither candidate build code nor
build-supplied tests run as the controller/signer. Preserve existing exact-digest
permits and ledgers; no generic sudo command or build-controlled timing/helper code
runs with signing credentials. Ordinary successful release signing is noninteractive.

**Reuse without a second build system.** Reuse upstream Go/Bun/Podman caches. Explicitly
selected previously qualified component digests may be reused only after native
signature/identity verification and provenance admission. A reused component retains
its actual producing revision; do not relabel it with the current release revision.
Current Soda-built image inspections expect the current producing revision;
verified reuse needs a deliberate owning-model/caller change, not relaxed checks. Do not introduce a guessed file
fingerprint graph or treat a cache hit as qualification. Changed dependency/base
inputs invalidate affected selections; compatibility tests still cover the complete
chosen set. Report verified-digest reuse separately from unknown compiler/layer-cache
state. "Build once" concerns shipping artifacts, not necessary test-binary compilation.

**Outputs and reporting.** One new `.artifacts/releases/RUN/` contains `inputs/`,
`work/`, `artifacts/`, `evidence/`, `release/` and `logs/`. Compile in the frozen source's
expected build-relative paths where existing leaf recipes require them; these remain
intermediate work, not another public payload. Reuse existing payload/candidate and
release models instead of an independent orchestration manifest. Keys, registry auth,
protected permits and durable authority ledgers stay outside public artifacts. Emit
one final scoped outcome: qualified local, published channel, failed, cancelled or
held for uncertain effects. Timings/logging are owned by [installation](installation.md#build-timing-and-progress-implementation-plan).

**Distribution and architecture.** Keep the already selected GHCR repository/key
boundaries during this rewrite; no package deletion or naming migration is implied.
GitHub Releases on `LevitateOS/sodaos` is the proposed ordinary ISO download destination,
with ISO digest/size/location authenticated by the final signed release document.
It requires a separate commissioning/publication grant. Stage uploads before making
an offer; require a release/architecture policy before exposing stable-labelled media.
There is no atomic transaction across GHCR and GitHub Releases: uncertain results
hold selection and are observed, not blindly replayed. Preview-to-stable promotion
reuses the final release, ISO and images; a later promotion invocation is not another
build. Start with native x86_64. The same run contract applies on native aarch64 when
its package lock/worker are qualified; both-architecture publication requires matching
release identity and all advertised receipts, not emulation. QCOW2 is not in the first
replacement: no producer exists, and it must later consume this same release rather
than introduce another assembler.

### Normal train

1. Collect ready Soda changes with focused tests and explicit migrations; unfinished
   changes remain out of the release branch/candidate.
2. Detect a new upstream stable release using supported upstream metadata through
   the scheduled trigger below; no custom listener daemon is required.
3. Resolve immutable inputs and build a candidate. Duplicate triggers must not create
   competing promotions or silently change the candidate's contents.
4. Qualify it using the matrix below. If a Soda feature blocks the new base, omit it
   or use the last compatible Soda version; a base incompatibility remains a real
   blocker requiring triage, not permission to publish a known-broken combination.
5. Use protected candidate artifact signatures for native qualification as specified
   by the single-run contract; finalize/sign the release and publish/promote only
   after its qualification gate. Stable follows bounded observation and explicit
   release-owner approval initially.
6. Record supported upgrade paths, release notes, interruptions and known limitations.

### Automated trigger and initial local builder

**The intended normal process is automated end to end.** The initial implementation
must expose noninteractive build/test/release commands usable by both a local
scheduler and CI. Do not build a manual-only recipe and later duplicate it in a
workflow. Manual promotion is used while commissioning; after qualification of the
pipeline and explicit production-automation approval, passing releases progress
through the configured observation gates without a human running each command.
Failed or inconclusive gates stop promotion and notify the release owner.

The proposed initial host is **this development/build machine**, not an appliance
being updated. Before installing anything, inspect its actual architecture, available
CPU/RAM/disk, virtualization support, uptime expectations and existing workloads.
Resolve build/test isolation and exact resource limits; do not infer that the current
checkout location makes the host a dedicated or suitably privileged builder. Native
architecture proof remains separate from cross-building image artifacts.

Use a native **systemd timer plus a one-shot job**:

- Poll the supported CoreOS stable metadata periodically; an hourly interval is the
  initial recommendation, to confirm with the host setup. No public listener is
  needed. A GitHub Actions schedule is an alternative execution host, not another
  pipeline or concurrent publication owner.
- Compare upstream release identity with durable candidate results. Freeze the exact
  base digests and an explicitly approved Soda staging revision at admission. Staged
  updates mean release-ready commits, not the working tree or everything on a moving
  branch. Select the actual staging ref/approval convention before enabling the job.
- Serialize admission and publication using native locking and a small durable
  candidate/result record. Repeated polls must not rebuild or republish the same
  candidate, race an emergency release, or silently change frozen inputs. New Soda
  changes after admission wait for the next candidate or explicit emergency trigger.
- Execute the [single-run phase sequence](#single-run-release-build-contract),
  including protected artifact signing before native qualification and final release
  authority afterward. Do not rebuild after testing. Ordinary build scripts and
  test workloads must not have
  signing keys or publication credentials. A local scheduler does not make a checked
  out branch trusted release code automatically.
- Preserve failed/interrupted runs and emit sanitized status and failure notification.
  Network polling failures can be retried on a later tick, but a failed candidate
  must not enter a tight automatic build/publish loop. Resume or retry an interrupted
  pipeline only from observed state and explicit safe step semantics; never replay
  an uncertain promotion blindly.
- Use persistent scheduling/catch-up after downtime. Reconcile missed releases
  against supported upgrade paths rather than assume the newest base is always
  directly upgradeable. An offline host cannot detect, qualify or publish releases;
  its availability is part of the release-response commitment.

Keep candidate inputs, outputs and receipts under ignored `.artifacts/`; private
inputs remain restricted and untracked. Define storage capacity, evidence retention
and notification destination before unattended runs; no automatic pruning is implied.
The scheduler must not mutate this canonical checkout or include another agent's
uncommitted work. Build from the recorded committed revision in an approved isolated
source snapshot, not a new Git worktree or an automatically checked-out branch here.

The same entrypoint should move to a dedicated builder or scheduled GitHub Actions
without redesign. Initially choose one scheduler/promotion owner. A custom central
update service, fleet telemetry and inbound remote execution remain unnecessary.
Appliance-side download/activation policy is separate: automatic publication does
not authorize simultaneous reboot of all appliances or bypass maintenance windows.

This is an implementation requirement and proposed host selection, **not permission
to install/enable the timer, start unattended CI or native VM tests, create signing
keys, register GHCR credentials or publish**. Those effects require the applicable
host/action approval after inspection and proof.

### Emergency lane

- An explicit authorized emergency trigger feeds the same noninteractive pipeline
  and serialization/promotion owner, without waiting for the upstream timer.
- Critical Soda fix: use the current qualified base unless the fix needs a new one.
- Urgent CoreOS fix: qualify the corrected base without waiting for staged features.
- Use the same provenance, signature, upgrade and preservation checks. Shorten
  observation with a documented risk decision, not by hiding failed checks.
- Issue a new release identity; do not mutate an already published release.
- Withdraw a bad channel offer to stop further uptake. Already installed appliances
  need a compatible corrective release or explicit recovery, not a silent downgrade.

Before production, assign a release/security owner and numeric targets for upstream
triage, routine qualification delay, critical-response time and preview observation.
These are operational commitments still to select, not invented SLAs in this plan.

## 6. Appliance behavior and operator experience

Start with explicit root/operator activation. Add automatic check/download and
scheduled activation after the native path is proved. Read-only checks must never
trigger enrollment, migrations, service restarts or reboot.

Expose current release/base, candidate release, verification/staging state, effective
maintenance policy, expected interruption and actionable sanitized failures through
native CLI/console first. Reuse stock Cockpit where supported; add only missing
Soda-specific controls through the existing operator-authorized surface. No second
login authority, arbitrary image URL/root-command API or custom OS page by default.

Activation sequence, adapted to the selected native mechanism:

1. Verify channel/release/artifact authority, supported path and architecture.
2. Preflight space, dependencies, configuration/schema and conflicting maintenance.
3. Download/stage without disrupting the current release where upstream supports it.
4. Explain affected services/projects/sessions and acquire the native maintenance
   ownership. Recheck critical preconditions before effects.
5. Preserve required configuration and consistent database state, quiescing writers
   when required by the owning migration procedure.
6. Activate the paired host/application contract in its tested order. Reboot when
   host content changes; do not promise reboot-free fixes embedded in the host image.
7. Run bounded post-activation checks and report success, partial failure or recovery
   required. A process starting is not sufficient proof of a working appliance.

Design for power loss and later resumption using native deployment state and the
smallest necessary Soda progress record. No generic workflow engine or blind replay
of migrations. Serialization must also account for operator rpm-ostree/container
maintenance; do not claim locks control arbitrary external root commands.

Maintenance windows control timing, not proof that jobs/shells are idle. Notify about
interruptions; do not promise tmux/process memory survives host reboot. Define bounded
operator deferral and a separately agreed urgent-security policy; never indefinitely
hold patches merely because shells are open. Emergency publication does not grant
permission to forcibly reboot outside configured policy.

Roll out progressively with preview/explicit opt-in initially. Fleet cohorts and
central telemetry are optional later work. Without telemetry, report download or
local health observations honestly, not fleet-wide installation success.

## 7. Preservation and recovery

Preserve machine identity, credentials, host Tailscale state, Forgejo/Soda databases,
repositories, project roots/accounts/keys, nested workload data and operator config.
New Project OS images affect future creation only unless separate same-root
maintenance is explicitly selected. Marketplace app upgrades remain independent.

For each supported upgrade, classify migrations as backward-compatible, requiring
an intermediate release, or not safely downgradeable. Gate activation when its
backup/recovery prerequisites are unmet. Retaining a previous bootable OS is useful
but `/var` and other writable application state are not rewound by OS rollback.

Test these distinct outcomes:

- Failure before activation: keep the working deployment and report the failed stage.
- Interrupted activation: identify actual native/app state; resume only safe steps.
- New host cannot boot: tested native boot recovery, with compatible app selection.
- Host boots but Soda fails: retain console/root access and sanitized diagnostics;
  prefer compatible forward repair when data has migrated.
- Database restore: separately authorized, consistent recovery with an explicit
  recovery point and later-write handling. Never silently overwrite newer data.

Do not claim automatic rollback until both boot and application compatibility have
native failure evidence. Offline export/import should use the same signed release
and trust rules, but its implementation follows the first online upgrade proof.

## 8. Qualification matrix

Reuse current build verifiers and native product journeys, adding focused update
cases rather than duplicating an outside test framework.

| Area | Required evidence for its claimed support |
| --- | --- |
| Build/trust | Pinned inputs, correct architecture, complete inventory, valid signatures; tampered, untrusted and incomplete releases refused |
| Fresh install | Matching media/first-boot behavior and usable operator access without embedded machine state |
| Upgrade | Previous supported release to candidate with populated databases and retained projects; explicit intermediate path where needed |
| Compatibility | Helper/API/browser graph, Forgejo, native packages, systemd/Quadlet and configuration preservation |
| Lifecycle | Planned service/reboot effects, boot-enabled resources return as configured, no unexpected project creation or provider enrollment |
| Failure | Registry/network loss, disk shortage, invalid signature, concurrent maintenance, interrupted staging/activation/migration |
| Recovery | Console access, selected boot fallback, application/schema compatibility and preservation of later writes |
| Policy | Unqualified upstream update cannot bypass channel ownership; effective maintenance window and explicit activation tested |
| Distribution | Missing architecture/artifact, stale/withdrawn offer and interrupted promotion handled safely |

Real Tailnet/provider actions need their own explicit budget and grant; never cause
registration just to test an updater. Native x86_64 and native aarch64 receipts are
separate. Cross-builds/emulation are not native acceptance. Scope each receipt to
checks actually performed; do not claim exhaustive supported upgrade coverage.

## 9. Implementation stages and exits

Execute **six consolidated milestones**, not 28 separate implementation rounds.
The detailed criteria retain their numbers for traceability and evidence. Finish an
approved milestone without repeated handoffs; independent source work can overlap.
Requirements remain in sections 1–8. The single-run replacement packages below
supply implementation order across those same acceptance criteria; they are not
six additional release milestones or six new approval rounds. Historical Stage 1/2
meant feasibility/initial local production, not these milestone numbers.

| Milestone | Acceptance criteria | Execution boundary |
| --- | --- | --- |
| 1. Complete appliance candidate | 1–7 | Source and local image builds/inspection; no deployment or publication |
| 2. Trusted delivery | 8–10 | Source/synthetic trust checks; exact signing/trust/GHCR effects need approval |
| 3. Native update and recovery | 11–17 | Source plus explicitly authorized isolated native fixture/effects |
| 4. Automated release builder | 18–22 | Source first; timer installation and unattended/promotion effects need approval |
| 5. Production readiness | 23–26 | Advertised native architectures, migration/media and authorized operational drills |
| 6. Production launch | 27–28 | Exact release/target approval, progressive launch and live automated-train proof |

Reuse valid evidence rather than rerun earlier criteria for documentation changes.
A checked box supports only its stated scope; production promotion cannot bypass
applicable checks. The effect boundaries above are real grants, not automatic
permissions conferred by completing source work or reaching a numbered criterion.

### Single-run build replacement implementation

**Status: planned; no replacement code, fixture, worker or publication commissioned
by this plan.** `d054a60`/`fde23d0` remain a transitional extraction with scoped tests,
not acceptance of the requested single-run architecture. Execute B1–B6 in order;
source tasks may proceed together where their contracts do not depend on unfinished
native proof. Do not expand the existing 28-item release checklist into separate
approval handoffs. The [run contract](#single-run-release-build-contract) owns phase
semantics; feature guides own the linked installation/timing/security details.

| Package | Existing release criteria served | Deliverable / exit |
| --- | --- | --- |
| B1 Contract and native-install feasibility | 1–7, 9, 11–13, 18, 25 | Reviewed dataflow, smallest file-removal map, selected upstream install/media invocation and explicit effect/resource requests. |
| B2 Replace artifact execution | 2–7, 18–20 | One Go entrypoint/clock/process owner; image-owned artifact construction and prepared tests; no writable staging translation. |
| B3 Make media consume the candidate | 11, 13, 25 | Installer and updater consume identical host/app digests; media assembly does not compile or build components. |
| B4 Connect native qualification | 8, 11–17, 23–24 | One command tests its actual ISO and image against approved baselines, preserving state and binding evidence to bytes. |
| B5 Connect protected finalization/delivery | 8–10, 19–20, 22, 25–26 | Unattended candidate signing, final release/media authority, delivery observation and channel-last selection. |
| B6 Retire old producers and commission automation | 18–23, 26–28 | Old competing build paths removed, real full-run receipt, then authorized timer/normal/emergency operation. |

#### B1. Fix the contracts before moving code

1. Audit the current paths and record their actual runtime call graph, generated
   files, trust boundaries, external effects and code size. Use `830ca94` (before
   the rejected extraction) and current source as comparison baselines. Count all
   production build/timing/orchestration code, including helpers moved or renamed;
   report tests/docs separately. Do not satisfy the budget by deleting verifiers.
2. Confirm the selected bootc version (currently 1.16.7), CoreOS live/host inputs,
   CoreOS Installer media support and actual Soda installer calls. Specify native
   host import/install, bound-image availability, retained-image import, SELinux,
   bootloader and per-machine setup dataflow. Test local transport primitives where
   authorized; obtain a narrowly scoped native disk/VM grant for installation proof.
3. Resolve media limits up front: current ISO level-1 per-file capacity, 256 KiB
   Ignition embed area, preserved primary names/boot metadata, archive/blob sizes,
   disk-space reserve and offline first-boot content. Do not assume an OCI reference
   works as an existing `coreos-installer install --offline` input. No implicit
   image-format upgrade, custom boot backend, new registry server or insecure flag.
4. Define the smallest extensions to `appliancerelease.Payload`/candidate and
   `releasedelivery.Release`: per-component provenance for verified reuse, media
   hash/size/location, pre-install candidate identity and post-test qualification.
   Specify format/version compatibility and strict refusal of unknown or incomplete
   records; preserve validation of retained old formats where their callers need it.
   Distinguish declared runtime/schema capability from proved upgrade edges in the
   final signed release. The current payload rejects nonempty `UpgradeFrom`; do not
   fill that embedded field after testing and thereby rebuild the qualified host.
5. Record the controller/build/qualification/signer identities, protected baseline
   and test-suite selection, exact initial x86_64 resource budget and public-only
   output boundary. A source checkout or sudo-capable developer UID is not isolation.

**Exit:** one concrete native installation/media contract with no unresolved assumption
hidden in an adapter; file/LOC baseline; input/output/model decisions; exact resource
requests. If upstream cannot satisfy the required installation path, stop for that
specific product decision before implementing an alternative. Do not require unrelated
ARM build proof to complete x86_64 source/mechanism work.

#### B2. Implement one artifact execution owner

1. Replace `tools/soda-host-image`'s orchestration with `tools/soda-build`; reuse the
   existing `internal/nativebuild`/`internal/hostimage` primitives rather than add a
   parallel framework/package hierarchy. A fixed `Build(ctx, request)` sequence and
   concrete phase functions are sufficient; remove the `Production` API where it
   exists only to support two layouts. No runtime-discovered task dependencies.
2. Implement P1–P6, with one native monotonic clock and one process/cancellation
   owner. Preserve the timing owner's output/exit contract without the Go/Python
   bridge. Validate dependencies/resources before work; public artifact commands
   never inherit signing credentials. Preserve fresh-output and frozen-source checks.
3. Compile shipping/runtime programs, installer console and required tools once;
   feed generated browser assets to existing `:prepared` suites. Adapt `check-source`
   and packaging-test callers so the full run does not invoke another build. Go test
   binaries remain distinct instrumentation, not duplicated shipping artifacts.
4. Replace `stage.py`'s writable-rootfs-to-vendor transformation with direct vendor
   destinations. Keep `forgejo-payload.json`, canonical assets, Containerfiles and
   locks authoritative; do not copy their inventories into the controller. Leaf
   asset scripts may remain if they neither own phases nor initiate another build.
5. Wire all five app images and the host into one candidate. Keep selected upstream
   images/binaries upstream; no Forgejo/CoreOS source rebuild or incidental tool
   upgrade. Admit explicit reused signed digests with their own producing revision;
   build other selections normally and record cache state honestly.

**Tests:** both architectures' source admission; real native ELF inspection for the
selected worker; once-only shipping compilation/asset/image call counts; prepared
browser-test inputs; complete/incomplete candidate parsing; changed source/base/lock
and reused-component provenance; no legacy paths in the host; wrong image IDs/platforms;
command failure/cancellation descendants and original status; occupied outputs; disk
or log-write failure; no credential in commands/logs; no signing/publication on build
failure. Use process doubles first, then one authorized fresh native candidate build.
**Exit:** one complete unsigned candidate with actual phase timings; this intermediate
implementation receipt is not the normal command's final qualified-release outcome.

#### B3. Replace the installer handoff, not just its wrapper

1. Turn `scripts/build-installer.py` into media-only assembly (rename to
   `assemble-installer.py` if retained). Remove its native-build invocation, Go
   compilation, source revision selection and independent timing/supervision. Its
   only product input is the admitted signed candidate plus prebuilt media tools.
2. Update `appliance/installer` / `internal/installer` to the B1-selected native image
   installation operation. Preserve password-only input, disk identity/revalidation,
   erase confirmation, correction/cancellation, private-file handling, no replay after
   an attempted write and explicit post-write outcome. Keep first-machine setup
   separate from immutable software; no credentials/database/project state in media.
3. Replace bundled `install-native.sh`/extension-layering continuation with the exact
   host/app candidate handoff. Check signatures/identities before privileged use;
   confirm all bound/retained content survives media removal and first boot. Wire the
   native updater to those same identities, not a separate application installer.
4. Preserve upstream boot equipment and existing ISO readback verifiers. Include the
   candidate installation descriptor, console and required content as ordinary media
   files, not a large Ignition executable payload. Keep private-network media separate
   and nonpublishable as general media. Do not embed final qualification metadata and
   rebuild the ISO after testing it.

**Tests:** media assembly invokes no compiler/image builder; incoming candidate/path/
architecture/signature substitutions refuse; artifact bytes match before/after
packaging; contents/ownership/modes/boot metadata/Ignition readback; oversized inputs;
wrong/in-use/hot-changed disk; cancellation/password/secret handling; partial writes
never replay. Under the exact fixture grant, install the actual ISO and remove media,
boot the same host digest, start its bound apps and import retained images offline.
**Exit:** first-install and update artifact identities demonstrably agree; old writable
bundle is absent from new media. ISO generation alone does not close this package.

#### B4. Make qualification part of the run

1. Connect P7–P9 to reviewed protected test drivers in `internal/acceptance` and
   `tests/installed`, adapting their exact artifact/media handoffs. Baseline releases
   and fixture identities are admitted inputs; do not
   discover/reuse an arbitrary retained VM or build a hidden second baseline inside
   the candidate run. Source tests cannot manufacture protected qualification evidence.
2. Execute the [qualification matrix](#8-qualification-matrix): the actual ISO's
   first boot/operator flow, supported baseline-to-candidate updates, maintenance
   ownership, populated database/project preservation, compatibility-aware recovery,
   signature/cache enforcement, missing content and interrupted operations.
3. Record artifact/ISO/test-driver identities, native architecture, declared baseline,
   scope and result. First-release absence of supported upgrade edges must be explicit;
   never infer an upgrade path from a successful clean install. Bootstrap may use a
   separately admitted baseline fixture, but the `native-install-upgrade-recovery`
   class still requires actual upgrade/recovery tests, not just an empty advertised
   edge list. Compare immutable digests and ISO hash before/after all tests; runtime
   writes belong only to fixtures.
4. Feed only protected successful evidence into final release admission. Failures,
   unavailable required tests and ambiguous fixture state stop the run and preserve
   evidence. Signer/qualification credentials and provider operations remain separate.

**Exit:** a real single-invocation native receipt covering the advertised release
scope, plus focused negative/failure receipts. No tests silently rebuilt a shipping
artifact or weakened verification. Aarch64 uses the identical implementation on a
native worker, with its own lock/resources/receipt when separately selected.

#### B5. Finalize, publish and promote the tested bytes

1. Reuse `internal/releasedelivery` and its protected worker protocol for P7 and
   P10–P11. Extend strict document validation only for the B1-admitted provenance/
   media fields; test both retained-format handling and new-format consumers.
2. Freeze the final release document after qualification; sign it without modifying
   the ISO/host/apps. Verify media against that signed record through independently
   bootstrapped trust. An artifact signature permits testing/authentication, not
   stable selection; local-only evidence still cannot enter preview/stable.
3. Under exact grants, commission proposed GitHub Release ISO uploads alongside GHCR
   immutable image/signature delivery. A draft/staging upload must not become a stable
   download advertisement before its gate. Candidate media must be a prerelease,
   not GitHub's Latest release; stable metadata changes follow the stable gate, not
   an asset replacement. Independently verify anonymous retrieval, digest/size and
   signed binding; keep private per-machine media out of publication.
4. Move the channel only after every advertised artifact/architecture/media output
   passes. Preserve current ledger/high-water/freshness/withdrawal semantics and the
   single publisher. Record uncertain GitHub Release effects as held intent too;
   observe existing resources before any retry, never reset a ledger or re-upload to
   repair a readback failure. Later stable promotion reuses this same final release.

**Tests:** synthetic/native local signing before live effects; artifact/channel role
confusion, wrong signer/repository, tampering, expired/replayed offers, wrong media
checksum and missing architecture; interrupted image/media upload and final tag
movement; duplicate observation; denied credentials; failed qualification sends no
promotion permit; changed bytes after qualification refuse; successful local runs
perform no remote publication. **Exit:** authorized candidate-only real round trips,
including ISO delivery, with no rebuild between qualification and publication.

#### B6. Delete competing producers and commission the one entrypoint

1. Remove `build-native.sh`, `build-iso.sh`, `build-progress.sh`, `build_progress.py`,
   the Go/Python progress bridge, `--legacy-native`, host-content-only release modes
   and old `tools/soda-host-image` orchestration after their replacement exits pass.
   Remove the old native metadata/staging build path where superseded. Do not retain
   successful compatibility wrappers indefinitely: deprecated commands either migrate
   to the one supported interface during cutover or explicitly stop with guidance.
2. Rewire `scripts/check-native.sh`, `scripts/check-source.sh`,
   `internal/acceptance/remote_executor.py`, `package.json`, AGENTS commands and
   support recipes to consume the new candidate/ISO without rebuilding.
   Retain existing legacy artifact verification and authorized maintenance consumers
   that are not producers; source retirement does not delete retained bundles,
   installations, credentials, fixtures or the predecessor repository.
3. Review the actual call graph and production LOC against B1. One run must not shell
   out to an old builder, create a writable Soda payload or maintain two phase lists.
   Shipping artifact build counts, wall timings and input-to-install/update digest
   equality are acceptance receipts. Overall production build orchestration must
   shrink; moving it to helpers or dropping verification is not a reduction.
4. Only after full local/native delivery proof and exact host/action approval,
   install the one systemd timer/one-shot invocation, protected credentials/limits,
   notification destination and durable candidate admission. Normal and emergency
   triggers call this same binary; long observation holds reuse release/ledger state,
   not a generic workflow-resume engine or another artifact build.
5. Execute authorized no-change/duplicate/catch-up/failure/notification drills, then
   candidate/preview operation and separately authorized stable/initial-appliance
   deployment. Routine signing and permitted promotion require no human attendance;
   uncertain publication or unsafe recovery still stops for explicit reconciliation.

**Exit:** the full-run receipt and source graph show one producer and one installation/
update artifact identity; old release producers are gone. Then close only the existing
milestone criteria supported by actual commissioned evidence. The live upstream train,
second architecture and retained-appliance migration keep their own outstanding exits.

#### Implementation effects and remaining approvals

This plan itself changes documentation only. Source/unit/process-double work and
applicable local builds use the latest task grant. Before real effects, consult the
[current handoff](implementation-status.md#current-permissions), not this list as a grant:

- Exact disposable native VM/disk, baseline, network boundary, resource budget,
  installation/reboot/recovery actions and hold/evidence lifetime for B1/B3/B4.
- Protected controller/build/qualification identities and fixed privileged worker
  installation/update, including credential custody and off-machine recovery.
- Actual GHCR candidate commissioning within its current grant; separately approve
  GitHub Release/media visibility and writes, broader channels or trust changes.
- Timer installation/unattended execution, notification destination and promotion
  policy; actual appliance targets/activation and migration remain separate grants.
- Cleanup only of exact authorized resources. Failed attempts and uncertain writes
  are retained by default; no pruning, root recreation or fixture replacement.

These are concrete commissioning boundaries, not per-release signing prompts or
permission requests for every implementation package. No numeric VM/storage/SLA
budget is invented here. If an output cannot be qualified with available grants,
report its exact blocked gate while completing independent approved source work.

### Milestone 1 — complete appliance candidate

**Complete for the x86_64 local candidate:** criteria 2–7 below have source/emitted
artifact evidence at `45ac843`, detailed in the
[receipt](implementation-history.md#complete-local-appliance-candidate). Native
first-boot/offline upgrade, SELinux/runtime and companion-turnover proof remain
milestone 3; aarch64 qualification remains milestone 5.

**1. [x] Establish the mechanism and first host-content build.**

- Deliverable: reviewed FCOS/bootc direction, pinned base, frozen-source builder,
  image-time packages and vendor-layout binaries/units.
- Completed evidence: `f390aa6` x86_64 build, package/layout inspection and OCI verifier;
  [receipt](implementation-history.md#first-local-host-content-image-build).
- This does not complete the appliance payload, signing or native boot proof.

**2. [x] Bind the fixed appliance application images.**

- Deliverable: immutable Forgejo, dashboard and proxy image references in vendor
  Quadlets, using bootc's bound-image mechanism and service-scoped image storage.
- Check: generated host/app references agree; no mutable `:dev` selection; matching
  app content is available before activation. Native offline-after-staging proof
  follows in step 14. Do not globally attach bootc storage to all Podman workloads.

**3. [x] Package the matching Forgejo presentation.**

- Deliverable: canonical templates, native locale, browser modules, branding/fonts
  and notices in immutable release-owned content, with the actual Forgejo caller
  wired to it. Reuse the existing payload manifest/build, not a copied asset list.
- Check: emitted browser graph/epoch matches backend contracts, native customization
  paths and SELinux/mount expectations; no release content depends on copying over
  a live Forgejo data tree. Run affected frontend/packaging checks.

**4. [x] Separate image defaults from machine state and persistent project images.**

- Deliverable: explicit ownership for app selection, `/etc/soda` settings/secrets,
  runtime paths and retained image storage. Preserve project roots and keep their
  required layers independent of bootc's image garbage collection. Audit companion
  image lifetime and saved image-ID callers through their current owners.
- Check: a new host release selects its intended app versions without replacing
  credentials/configuration; project image references remain valid across host
  release changes. No real Tailnet/provider job is triggered by validation.

**5. [x] Finish the host package/build contract.**

- Deliverable: resolve the recorded Forgejo-runner/udisks2 tmpfiles warning through
  native package/service ownership; capture and pin the resolved RPM inputs and
  repository provenance needed to rebuild the candidate, including retention/access.
- Check: base/package inventory is intentional, build lint outcomes are accounted
  for, and repeating frozen inputs cannot silently resolve newer RPMs. Do not claim
  byte-reproducible image output unless that property is separately demonstrated.

**6. [x] Define and emit the complete local payload/candidate record.**

- Deliverable: version/release ID, architecture-specific host/app digests, source
  revision, provenance/inventory, schema, supported upgrade edges, migration
  requirements and release notes, following sections 3–4. Empty qualified upgrade
  edges are explicit, not permission to migrate. Intended GHCR names in a local
  candidate do not reserve/provision them. Milestone 2 establishes actual namespace/
  signing/channel authority; milestone 4 establishes the approved staging-ref policy.
- Check: complete-or-error parsing and architecture selection; unknown/incomplete
  inputs fail safely. Inventory and app references have one authority, not multiple
  independently maintained release manifests.

**7. [x] Build and inspect the complete local candidate.**

- Deliverable: a noninteractive complete host/app build from a clean frozen revision,
  with public-only artifacts and scoped local evidence.
- Check: package/binary/image identities, presentation and runtime wiring, file
  permissions, no machine secrets/state and all affected source tests. This is the
  first complete candidate, still not a native acceptance or production release.

### Milestone 2 — trusted delivery

**Source/local-native slice implemented.** Native keyed-Sigstore directory proof
passes over synthetic fixtures and the actual retained M1 host/app payload. Real
keys and immutable packages are now staged and authenticated native GHCR round trips
pass. Public visibility/anonymous and promotion commissioning (9–10), plus installed
bootc/cache enforcement, remain distinct pending evidence. See the
[receipt](implementation-history.md#trusted-delivery-source-and-native-filesystem-proof).

**8. [x] Implement trust and channel verification with local fixtures.**

- Deliverable: select the native-supported production signing scheme, signer/key
  custody and bootstrap/rotation design; implement signed channel/release binding,
  freshness and replay/downgrade behavior from section 4. Use synthetic trust inputs
  until actual key/identity creation is explicitly authorized.
- Check: unsigned, wrong-signer, wrong-repository, tampered, stale/replayed and
  preview-as-stable substitutions refuse. Test missing architecture/artifact and
  intentional recovery separately; a digest signature alone is not channel approval.

**9. [ ] Provision the exact GHCR and signing resources.**

- Deliverable, after explicit approval: selected namespaces/visibility, least-privilege
  publication identity, protected signing authority and reviewed retention/access.
  Public appliance pulls should not require a developer's personal token.
- Check: ordinary build/test execution cannot read signing/publication credentials;
  credentials do not enter source, images, command arguments or evidence. Record
  actual production trust bootstrap and recovery custody before shipping it.

**10. [ ] Implement protected publication and commissioning promotion.**

- Deliverable: publish immutable tested artifacts and signature attachments first,
  verify their availability, then publish the complete signed channel selection.
  Initially require an explicit promotion approval; do not rebuild on promotion.
- Check: signed GHCR round trip, anonymous public pull if selected, interrupted
  upload/promotion, duplicate invocation and withdrawn offer. A failure leaves the
  prior valid channel usable. Include exact per-architecture completeness checks.

### Milestone 3 — native update and recovery

**11. [ ] Implement the native update admission and activation caller.**

- Deliverable: a small fixed-operation Go caller around native bootc discovery,
  verification/staging/status/activation, with one maintenance owner. Use the
  signed approved digest rather than re-resolving a tag after confirmation.
- Check: source tests for authority, fixed targets, compatibility, disk/maintenance
  conflicts, cancellation and partial outcomes. No arbitrary image URL, executable
  or root-command interface; no migration/reboot on a read-only status request.

**12. [ ] Prepare one authorized isolated native fixture.**

- Deliverable: exact x86_64 VM/disk/hold, console access, trust and registry/test
  inputs, synthetic persistent state and declared failure scenarios. Reuse the
  existing fixture driver and product checks. Preserve all retained appliances.
- Check: fresh target is isolated, credentials are fixture-owned, recovery access
  works independently of Soda/Tailnet, and lifecycle effects fit the actual grant.
  This is not permission to reuse an expired VM or recreate an occupied root.

**13. [ ] Prove fresh installation and first boot.**

- Deliverable: connect the complete image to the owning first-install/setup flow,
  with per-machine identity/secrets and native package/service defaults.
- Check: install on the authorized disk, reach console/Cockpit/Forgejo/Soda, exercise
  existing protected product reads and a specifically authorized synthetic project
  journey; preserve private-only access and enforcing SELinux. No generic image may
  contain preinitialized operator credentials or personal application state.

**14. [ ] Prove same-base and new-base upgrades.**

- Deliverable: A → B with Soda changes on one qualified CoreOS base, and a separate
  base-change case once a second exact base is qualified. Include populated data and
  a retained project; verify declared schema/configuration migrations.
- Check: download-only does not activate prematurely; explicit activation uses the
  exact staged release; matching app images work without registry access after boot;
  settings, keys, data and project identity remain intact. One case is not evidence
  for the other, and bootc process success alone is not application health.

**15. [ ] Prove failure and compatibility-aware recovery.**

- Deliverable: interruption/recovery cases from sections 7–8 using the same fixture,
  including disk shortage, network loss, invalid artifacts, interrupted migration,
  failed app startup and a boot-failure case covered by its exact grant.
- Check: console recovery, accurate partial-state reporting, safe resumption and
  native boot fallback where app/schema compatibility permits it. Preserve a later
  database write across fallback; demonstrate refusal/forward repair where downgrade
  is unsafe. Do not simulate successful rollback by restoring an older database.

**16. [ ] Prove the single update owner and maintenance policy.**

- Deliverable: effective Zincati/bootc scheduling ownership, automatic download and
  configured activation windows, bounded deferral, interruption notice and separately
  selected urgent-security policy. Do not assume active terminals or CI jobs are idle.
- Check: upstream CoreOS cannot bypass Soda qualification; a generic reboot does not
  unexpectedly activate download-only state; concurrent native/root maintenance is
  detected or clearly reported. No second scheduler races activation.

**17. [ ] Finish operator status and resolve the Cockpit compatibility defect.**

- Deliverable: CLI/console and supported operator UI showing installed/base/offered
  versions, verification/staging state, expected disruption, failures and recovery.
  Confirm the reported OSTree remote error against the installed package/caller and
  choose a supported fix or honest unsupported-control presentation.
- Check: stock administration remains usable; update controls enforce root/operator
  authority and never offer an independent incompatible update path. Do not invent
  an OSTree remote or replace the whole Cockpit page to hide the error.

### Milestone 4 — automated release builder

**Transitional extraction, not target acceptance.** `d054a60` shared component
recipes and retained `2166333` timing behavior, but kept two assemblers and added
orchestration. The owner requested its replacement, not more adapters around it.
[The single-run implementation packages](#single-run-build-replacement-implementation)
now own execution order across M1–M5. Items 18–22 remain open; no timer or full
source-to-qualified-ISO-to-publication invocation has been proved. Native installation
and recovery remain the retirement gate, not a reason to keep producing writable
bundles in the new command. Old installed state and historical artifacts remain
preserved after source-producer retirement.

**18. [ ] Finalize this machine's builder operating contract.**

- Deliverable: complete the initial resource inspection with agreed CPU/RAM/disk
  limits, build/test isolation, native VM budget, uptime expectation, notification
  destination, staging-ref policy and retained-output capacity. Select an alternative
  builder only if the local host cannot meet the agreed requirements.
- Check: builds cannot mutate the canonical working tree, other workloads or retained
  fixtures; native test authority is confined to exact allocated resources; ordinary
  build workers cannot reach signing/promotion secrets. Readable KVM is not this proof.

**19. [ ] Implement the CoreOS trigger and candidate admission.**

- Deliverable: one-shot metadata poller, candidate identity/result record and native
  serialization, feeding the same noninteractive commands already proved locally.
  Freeze the selected base and approved Soda revision; unfinished work stays out.
- Check: no-change polls, duplicate events, changed staging head, malformed metadata,
  network failures, interrupted runs and missed releases after downtime. Do not select
  an invalid direct upgrade merely because multiple upstream releases were missed.

**20. [ ] Connect the full normal and emergency pipeline.**

- Deliverable: the [single-run phase sequence](#single-run-release-build-contract),
  including protected candidate signing before native qualification and final signed
  release metadata afterward, then authorized publication/promotion. The emergency
  entry uses the same pipeline and serialization without waiting for the CoreOS
  trigger; initial stable promotion remains explicit.
- Check: failures stop downstream effects and notify the owner; uncertain publication
  is observed rather than blindly replayed; normal/emergency races cannot publish
  conflicting channels. Candidate provenance identifies exactly what was tested.

**21. [ ] Install and commission the local timer/one-shot service.**

- Deliverable, after host/service and automatic-execution approval: deploy the reviewed
  timer/job, state/log directories and resource/security settings on this machine.
  Start with promotion held for explicit review; do not grant arbitrary future code
  unrestricted root access through the scheduler.
- Check: scheduled and catch-up ticks, real notification delivery, reboot/downtime
  recovery and preserved failure evidence. The job must not enroll providers, expand
  its fixture budget or clean retained resources automatically.

**22. [ ] Qualify unattended progressive promotion.**

- Deliverable: defined preview/observation gates and promotion policy, then enable
  automatic normal preview-to-stable promotion under explicit production approval.
- Check: a passing candidate advances with no human running release commands; a bad,
  incomplete or unobserved candidate does not advance. Confirm notification and
  withdrawal behavior. Without fleet telemetry, use explicit controlled preview
  evidence rather than interpreting downloads as fleet health.

### Milestone 5 — production readiness

**23. [ ] Qualify each supported architecture and upgrade path.**

- Deliverable: native receipts and actual image availability for x86_64 and aarch64,
  with an explicit supported-starting-release matrix and intermediate upgrades.
- Check: each advertised architecture passes its applicable install/update/recovery
  contract. x86_64 can progress independently; do not advertise missing ARM support
  or call emulation native evidence. Overall two-architecture completion remains
  pending until both have their own receipts.

**24. [ ] Implement and rehearse existing-install migration.**

- Deliverable: migration from the documented writable Soda installation to the
  image-owned layout, including exact old units/overrides, RPM layering, app/companion
  references, schema/configuration and update/trust ownership.
- Check: rehearse representative starting states in isolation with consistent
  backups and later writes preserved. Refuse unknown conflicts; do not blindly reset
  layered packages, remove whole directories or replay first-install. Actual retained
  targets still require their own migration approval and fresh inventories.

**25. [ ] Align installation media with the release train.**

- Deliverable: ISO installs the same signed host/app candidate consumed by updates,
  with qualified trust/bootstrap and no writable-bundle assembly. QCOW2, if later
  implemented and advertised, is another consumer of that candidate, not a second
  source build or separately assembled OS.
  Keep media provenance/architecture identity and first-boot credentials correct.
- Check: the advertised download installs the qualified release and can consume its
  next update without private builder transfer. A missing QCOW2 producer is an explicit
  unsupported artifact, not a reason to claim that it exists or block ISO-only delivery.

**26. [ ] Complete security and release operations drills.**

- Deliverable: assigned release/security owner, numeric response/qualification and
  observation targets, signing rotation/revocation recovery, artifact retention,
  builder failure recovery and incident/withdrawal runbooks.
- Check: execute a bounded emergency release, interrupted-publication recovery and
  trust-rotation test on authorized resources. Confirm old supported intermediates
  remain available and later writes survive the selected recovery path. A runbook
  alone is not a successful operational drill.

### Milestone 6 — production launch

**27. [ ] Publish and deploy the first production release progressively.**

- Deliverable, after exact release/target approval: promote the signed qualified
  release, offer it to the approved initial appliance group and activate according
  to their policy. Expand availability only after the declared observation gates.
- Check: intended client verifies/downloads/activates the exact release, operator
  access and product checks pass, and preservation/migration observations match the
  approved scope. Publication to GHCR is not installed-fleet acceptance.

**28. [ ] Demonstrate the ongoing automated release train and close the workstream.**

- Deliverable: one normal upstream-triggered release using approved staged Soda
  changes completes build/test/sign/publish/promotion without manual release commands,
  plus an independent emergency-triggered release through the same pipeline.
- Check: approved appliances receive and activate according to policy; failed-gate
  and withdrawal paths remain effective; operations ownership and supported scope
  are documented. A synthetic trigger can commission the mechanics, but completion
  of the live normal train requires an actual new upstream release event. Reuse the
  preceding emergency drill if it meets this exact scope; do not replay mutations
  solely to create a second receipt.

### Definition of done and follow-ups

**Done means all six milestones have evidence for their applicable criteria and advertised deployment scope:** a new
CoreOS stable release automatically produces a qualified, signed Soda release from
approved staged changes, publishes/promotes through GHCR, and reaches appliances
under explicit maintenance policy. Emergency releases work independently. Trust,
preservation, recovery and an accountable operational owner are part of completion.
Known unsupported architectures, starting states or media must stay explicit; they
cannot silently count as accepted full target support. An x86_64-only production
milestone may ship independently, but full two-architecture workstream completion
still requires the aarch64 evidence in step 23.

Offline update import, broader fleet cohorts/telemetry and moving the builder to
other hosting are follow-ups, not prerequisites for the core online automated train.
This does not defer the single-run ISO's required installation/first-boot content.
General fleet orchestration and arbitrary application-data rollback remain out of
scope. Completion does not grant future destructive maintenance, cleanup or changes
to the predecessor's separately reserved Updates platform.

## 10. Workstream status and next action

The [six milestones above](#9-implementation-stages-and-exits) are the single task
list for this workstream. **Milestone 1 is complete for the x86_64 local candidate
at `45ac843`; milestone 2 has real protected keys and authenticated immutable GHCR
round trips, with public visibility/anonymous and channel commissioning pending. Milestones 3–6 remain pending.** This consolidation changes execution granularity,
not production gates or effect permissions. The owner subsequently rejected the
shared-production extraction as the final design and requested a full single-run
replacement. Its B1–B6 implementation packages map to these existing criteria; none
is marked complete by this documentation pass.

**Recommendation:** prove derived FCOS using bootc's existing OSTree backend,
digest-pinned logically bound core appliance images and native keyed-Sigstore
verification, initially with explicit activation. The
[research receipt](release-engineering-feasibility.md) owns exact versions, source
references, the writable-layout/client-layering costs and the isolated proof proposal.
No custom update server or alternative boot backend is proposed.

The owner approved the complete appliance candidate milestone, including app binding,
Forgejo presentation, defaults/storage ownership, package fixes and release metadata.
The [host-image command](native-support.md#local-host-content-image-candidate) owns
invocation/effects. Default installer paths remain unchanged; vendor builds do not
widen native unit admission. Initial builder inspection found native x86_64, 16
logical CPUs, 62 GiB RAM, about 543 GiB free and local rootless Podman 5.8.2. That
is resource availability, not dedicated capacity or VM qualification. Builds use
pinned Go 1.26.7 rather than the shell's Go 1.27.0.

**Built:** `.artifacts/host-image/complete-45ac843/host.oci`, five app archives,
`payload.json` and detached `candidate.json`. The
[receipt](implementation-history.md#complete-local-appliance-candidate) records exact
digests, source/local tests and retained attempts. Three digest-bound Quadlets pass
the native generator; 391 Forgejo files and embedded Project OS/Tailnet archives
match their hashes. The 625-RPM inventory matches its lock and bootc lint now reports
13 passed, one skipped, no warnings. The earlier host-only candidate is preserved.

The [content ownership contract](#local-candidate-content-and-machine-state-ownership)
retains the limits: no machine settings/secrets are generated, conflicting saved
image choices refuse, and changing the companion image within an existing run is
not an admitted hot upgrade. RPM bytes are not mirrored; aarch64 has no transaction
lock/native proof. Metadata has no qualified upgrade edges.

**Next action: B1, not another build-lane extraction.** Validate the concrete native
image-install/media contract, artifact/evidence dataflow and deletion/LOC baseline,
then implement the single Go run in B2–B6. The requested [full plan](#single-run-build-replacement-implementation)
is now recorded. No replacement code or native commissioning was performed for it.
The [transitional extraction receipt](implementation-history.md#shared-build-production-and-timing-consolidation)
remains valid only for its source tests and host-context preparation, not acceptance
of the desired build architecture. No GHCR/key/appliance changes followed this plan.

**Pending milestone 2 commissioning, not part of the source refactor:** the owner
sets the eight new GHCR packages to Public.
The [real commissioning receipt](implementation-history.md#ghcr-namespace-and-signing-bootstrap)
records root-protected keys, the selected existing `gh` account, eight immutable
signed uploads and authenticated native signature/digest round trips. GitHub currently
reports **internal** visibility; an isolated anonymous probe refused, with no
credential fallback. All eight packages exist before this one-time UI handoff.
No mutable candidate tag was promoted. After visibility changes, verify anonymously
and commission candidate selection/failure handling under the existing bounded grant.
The local M1 payload remains unqualified for preview/stable or native installation.
Off-machine recovery custody and unattended worker isolation remain later readiness
work, not per-release human signing. No timer or retained-appliance migration is
implied. The Cockpit error still needs installed-version/caller confirmation. Independent
Tailnet tasks remain with their own workstream; this plan does not absorb their list.
