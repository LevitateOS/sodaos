# SodaOS release engineering — single-run replacement first

## Priority and current state

**The active work is replacing the build lane, not launching the release service.**
Implement the six replacement milestones below before unattended scheduling,
production-readiness commissioning or launch. The previous complete-candidate →
trusted-delivery → native-update → automated-builder roadmap put the rewrite too
late. It is no longer the execution order. Its former milestone numbers and 28-item
checklist remain historical references in Git, not a competing active task list.

The owner rejected `d054a60`: sharing component functions retained two assemblers
and increased production orchestration. The replacement must produce and qualify
**one immutable host/application candidate used by both installation and updates**,
then remove the competing producers. A wrapper around the existing lanes does not
meet that requirement.

Existing host/app candidates, native Sigstore tooling and signed GHCR bootstrap are
reusable foundations, not completed milestones of this replacement. None of the
six replacement milestones is complete. [Implementation status](implementation-status.md)
owns current progress, retained release custody and grants; [history](implementation-history.md)
owns receipts. The predecessor repository and its Updates platform remain separate.

## Single-run build replacement implementation

**Only these six milestones define the current execution order.** B1–B6 are stable
identifiers for these milestones, not six subordinate tasks beneath another roadmap.
Finish approved work without repeated milestone handoffs; ask only for a concrete
missing target/action grant or unresolved product decision. Independent source work
can proceed while an exact native-effect gate is blocked.

| Active milestone | Deliverable | Exit |
| --- | --- | --- |
| 1 / B1 — Verify the native installation contract | Selected upstream mechanism, frozen-input/dataflow contract and deletion baseline | No assumed image-install/offline-content capability hidden in an adapter |
| 2 / B2 — Implement one Go build controller | Programs/assets/apps/host produced once with direct timing and cancellation | One complete candidate, no writable staging translation or second producer |
| 3 / B3 — Make the ISO consume that candidate | Media-only assembly and image-based first-install handoff | Actual ISO installs the same host/app digests used by updates |
| 4 / B4 — Connect native qualification | Exact ISO and baseline-to-candidate install/update/recovery tests | Protected, byte-bound native evidence without rebuilding shipping artifacts |
| 5 / B5 — Integrate protected signing and delivery | Existing trust/publisher mechanisms connected to the run | Noninteractive signing/finalization and tested channel-last delivery interface |
| 6 / B6 — Retire old lanes and prove the replacement | Competing producers deleted and callers/documentation cut over | One complete native local run, preserved verification and smaller production orchestration |

**Not prerequisites:** finishing GHCR public commissioning, publishing an ISO,
installing a timer, automatic stable promotion, aarch64 fleet readiness or migrating
retained appliances. Those follow the replacement. Required signing and native
installation/security tests are part of the replacement, not deferred verification.
Local-only or synthetic evidence never authorizes a production channel.

### Milestone 1 — verify the native installation contract

**B1 in progress: source/caller audit, native CLI/config inspection and baseline
recorded; disk/boot feasibility awaits an exact native fixture grant.** The selected
path and offline/security constraints are in the [installer contract](coreos-installer-plan.md#b1-selected-native-mechanism--source-and-cli-proof).
The [receipt](implementation-history.md#b1-native-installation-contract-and-removal-baseline)
distinguishes passed source/inspection checks from still-unrun installation.

1. Audit the actual native/ISO and host-image call graphs, outputs, trust domains and
   effects. Record production build/timing/orchestration LOC against `830ca94` (before
   the rejected extraction) and current source. Include moved/renamed helpers; count
   tests/docs separately. Do not shrink the budget by deleting verifiers.
2. Inspect the selected FCOS, bootc (currently 1.16.7), CoreOS Installer and native
   media tooling versions and actual Soda callers. Select the exact upstream host
   import/install operation and offline bound/retained-image handoff. Existing
   `coreos-installer install --offline` is not assumed to install a derived OCI host.
3. Resolve bootloader, SELinux, storage lifetime, first-machine setup, media-removal
   behavior and disk space. Check ISO level-1 per-file limits, primary filenames,
   preserved boot metadata, actual archive/blob sizes and the 256 KiB Ignition area.
   No custom boot backend, temporary registry server or insecure install flag.
4. Define the minimal additions to `appliancerelease.Payload`/candidate and
   `releasedelivery.Release`: component producing provenance, install descriptor,
   ISO hash/size/location and protected qualification. Preserve required retained
   format readers; reject unknown/incomplete inputs. Declared runtime capability and
   proved upgrade edges are different. The current payload rejects nonempty
   `UpgradeFrom`; never fill it after testing and thereby rebuild the host.
5. Specify reviewed controller/build/test/signer identities, input admission, exact
   native baseline/fixture, resource limits and public-only output boundaries. Source
   snapshots, sudo access and root-only files alone do not provide worker isolation.

**Checks/exit:** source/upstream-supported installation dataflow plus authorized local
transport/native feasibility evidence for the selected mechanism; explicit fixture
requests and file-removal/LOC baseline. If upstream cannot satisfy the desired path,
stop for that specific product decision—do not silently restore the writable-bundle
lane. No unrelated architecture or provider gate is added to x86_64 work.

#### B1 artifact and authority handoffs

The current caller graph is:

```text
build-iso.sh → build-installer.py --build-native → build-native.sh
  → soda-host-image --legacy-native → Production → native metadata/seal → ISO

soda-host-image --complete → frozen archive → Production + hostimage
  → writable-shaped stage.py output → vendor translation → candidate

soda-release → separately reviewed trust/prepare/sign/publish/fetch operations
```

Keep one future source-to-release graph; separate security identities are not separate
build recipes. Preserve existing content/verification owners rather than copying them.

| Boundary | B1 decision for implementation |
| --- | --- |
| Frozen inputs → producer | Approved commit/base/tool/package and explicit component selections; public-only source snapshot. No privileged signer credentials or implicit selected VM. |
| Producer → protected admission | Existing exact payload/candidate bytes plus OCI identities and hashed provenance. Treat outputs as untrusted until independently checked and copied into protected fresh snapshots. Never execute candidate-supplied tests as authority. |
| Component provenance | Keep the first replacement's payload/candidate v1 identity contract where sufficient. Extend versioned `app-inputs.json` to record actual producing revision or upstream reference for each selection; do not add a second independently maintained image inventory or relabel reused images. Strict reader/caller changes precede reuse. |
| Candidate → media | Existing candidate/payload bindings plus native signed directory snapshots of host and three bound apps. Embedded Project OS/Tailnet archives travel inside the signed host. Replace `mediaIdentity.BundleSHA256` with a versioned candidate-byte binding; the media record is not self-issued signing authority. |
| Media → live installer | Independently trusted media/tool bytes and verified artifact signatures before privileged image execution. The exact rootful store belongs to this installation; rootless/default builder storage is not a supported bootc install source. Native lookup/digest/cache tests remain required. |
| Live installer → installed state | Upstream filesystem installation and native bound-image copying; machine-only Ignition and first-boot import of embedded retained images. No writable bundle or client RPM transaction. |
| Qualification → final release | Extend `releasedelivery.Release` with a strict v2 media record (ISO name/hash/size and planned public URL when applicable) and exact proved starting-release references. Keep embedded `UpgradeFrom` empty rather than changing the host after tests. Retained v1 verification stays supported at its existing scope, not silently promoted to a v2 media/native claim. |
| Protected finalization → optional publisher | Exact-digest permit, final signed document and evidence; existing ledgers/observation semantics. No arbitrary commands, fixture credentials or authority from build-reported success. |

Controller, builder, native qualification and signer are distinct authorities. A
reviewed controller dispatches fixed operations; the builder can write only its work
outputs. Qualification uses an admitted driver/baseline in an isolated target. The
signer snapshots/checks exact bytes with protected keys, independent of untrusted
build code. Fixture trust cannot enter real channels; root-only files do not isolate
jobs running as the sudo-capable administrator. Installing those identities/workers
is an explicit host action, not performed by this audit.

#### B1 removal and size baseline

Measured physical source lines (including comments/blanks) at `830ca94` and
`e4f485a`; production, tests and guides are separate. The receipt retains the exact
path inventory in `loc-baseline.tsv`/JSON, not a guessed count or production benchmark.

| Selected responsibility | Before extraction | Current baseline | Production delta |
| --- | ---: | ---: | ---: |
| Build assembly/orchestration, including full media/staging/metadata helpers | 2,508 | 2,879 | +371 |
| Asset/bootstrap leaves | 335 | 335 | 0 |
| Image recipes and full workspace manifest | 160 | 160 | 0 |
| Artifact verification, including payload/import owner | 1,556 | 1,556 | 0 |
| Installer runtime and layout owner | 3,483 | 3,483 | 0 |
| Protected delivery | 1,526 | 1,526 | 0 |
| Native acceptance | 1,769 | 1,769 | 0 |
| Selected production total | 11,337 | 11,708 | +371 |

Selected colocated tests: 4,731 → 5,154; `tests/build` fixtures: 3,675 → 3,731;
seven owning guides: 3,889 → 2,958. Documentation shrinkage is not production
simplification. These are explicit selected areas, not total repository LOC.

- **Remove producer ownership:** `build-native.sh`, `build-iso.sh`, `--legacy-native`,
  partial host modes and old `tools/soda-host-image` entrypoint; replace with the one
  controller rather than retain success wrappers around them.
- **Remove overlapping supervision:** `build_progress.py`, `build-progress.sh` and
  `internal/nativebuild/progress.go`'s Python bridge. Port necessary process/timing
  checks to the native Go owner, not another helper clock.
- **Collapse assembly:** the dual-layout `Production` contract, writable staging and
  subsequent `StagePresentation`/`Complete` translation; retain canonical manifests,
  actual recipes/assets and checks. Retire legacy metadata/sealing only as production;
  keep supported historical-artifact readers and maintenance consumers.
- **Convert media to a leaf:** remove its compiler/native invocation and coordinator,
  but preserve ISO trust/readback/boot verification. Replace runtime bundle copying/
  extension continuation, preserving disk identity, secret, partial-write and setup
  guards. Report new functional installer code separately rather than hiding it.
- **Retain and connect:** ELF/OCI/bundle readers, native acceptance and protected
  delivery mechanisms. Rewire their callers; deleting verification is not savings.

At B6, classify every new/moved equivalent by responsibility, regardless of path or
language; compare both the pre-extraction and current baselines and show combined
production impact. No budget pass can come from moving orchestration into a verifier,
media leaf, test driver or another renamed package.

### Milestone 2 — implement one Go build controller

**B2; not started.** Replace execution ownership, not just command duplication.

1. Replace `tools/soda-host-image` orchestration with `tools/soda-build`, reusing
   `internal/nativebuild` and `internal/hostimage` primitives. A fixed
   `Build(ctx, request)` sequence and concrete functions suffice. Remove abstractions
   that exist only to support two assemblers; no task graph, plugin/controller
   framework, custom cache database or listener service.
2. Implement P1–P6 from the [run contract](#single-run-release-build-contract): clean
   source admission, `git archive`, frozen dependencies, shipping programs/assets,
   prepared tests, five app images, host assembly and candidate verification. One
   native Go clock/process owner replaces the Go-to-Python timing bridge. Preserve
   [timing/failure behavior](installation.md#build-timing-and-progress-implementation-plan).
3. Compile runtime programs, media console and required tools once. Existing browser
   `:prepared` suites consume the emitted assets rather than invoke another build.
   Go test instrumentation is not duplicate production compilation.
4. Replace `stage.py`'s writable-rootfs-to-vendor transformation with direct vendor
   destinations. Keep Containerfiles, locks, `forgejo-payload.json`, canonical assets,
   notices and existing source manifests authoritative. Python asset helpers may
   remain leaves, not phase owners or hidden image builders. No custom CoreOS or
   Forgejo source build; no incidental tool upgrades.
5. Produce the five app images and host, or admit explicitly selected previously
   qualified component digests through signature/provenance verification. Retain
   each reused component's real producing revision; change owning validation/callers
   deliberately, not by relaxing current-revision checks. No guessed fingerprint
   scheduler. Use ordinary Go/Bun/Podman caches and report unmeasured cache state honestly.

**Checks:** command doubles verify once-only shipping work, prepared-test inputs,
source/base/lock/platform admission, reused provenance, occupied-output refusal,
credential exclusion, process/descendant cancellation, original failure status and
no downstream effects after failure. Native ELF/OCI/content/inventory checks verify
an actual complete candidate. This intermediate unsigned artifact is not yet a
qualified end-to-end release or the command's final success outcome.

### Milestone 3 — make the ISO consume the candidate

**B3; not started.** Replace the installation handoff and backend.

1. Convert `scripts/build-installer.py` to media-only assembly; rename it to
   `assemble-installer.py` if retained. Remove native-build invocation, Go compilation,
   source selection and independent supervision/timing. Inputs are the signed
   candidate, prebuilt console/tools, public trust/bootstrap and selected live media.
2. Update `appliance/installer` / `internal/installer` using the B1-selected upstream
   image-install operation. Preserve password-only input, disk identity/in-use and
   last-moment checks, exact erase confirmation, correction/cancellation, hidden
   secret-file inputs, no replay after an attempted disk write and explicit outcomes.
3. Replace bundled `install-native.sh` and client-side package-layering continuation
   with the exact host/app handoff. Keep per-machine identity, configuration, operator
   setup and credentials outside immutable software. Required content must remain
   available after media removal, without registry access during install/first boot.
4. Keep the console/content in ordinary ISO files and preserve native boot equipment
   and readback verification. Private-network media is not public distribution media.
   The final post-test report stays outside the ISO; do not rebuild media after testing.

**Checks/exit:** assembly invokes no compiler/component builder; candidate/architecture/
path/signature substitution and oversized content refuse; metadata/ownership/modes/
boot/Ignition readback pass. Under an exact native fixture grant, install the actual
ISO, remove media and boot its selected host/apps with enforcing SELinux and working
operator access. Test partial writes and cancellation without replay. The updater
must consume those same immutable identities. The [installer owner](coreos-installer-plan.md#image-based-replacement-contract)
owns detailed disk/bootstrap constraints.

Use existing reviewed signing primitives and fixture trust to develop this handoff;
B5 connects their final protected orchestration. Implementation order is not runtime
phase order: artifact authentication still precedes privileged installation.

### Milestone 4 — connect native qualification

**B4; not started.** Make qualification consume this run's actual artifacts.

1. Connect P7–P9 to reviewed `internal/acceptance` and `tests/installed` drivers, not a
   new framework or build-supplied `passed` assertion. Baseline/test-driver/fixture
   identities are admitted inputs. Do not adopt an arbitrary retained VM or build
   an unrecorded second baseline inside the candidate run.
2. Implement the small fixed-operation Go update admission/activation caller around
   native mechanisms. Enforce exact staged digests, authority, compatibility, space,
   maintenance ownership, cancellation and honest partial-state reporting. No root
   command/image-URL interface or mutation from read-only status.
3. Exercise actual ISO first boot, same-base Soda updates, separately qualified
   base-change updates, populated databases and retained projects. Verify offline
   availability after staging, machine settings/keys/identity and declared migrations.
   A single base/update case cannot qualify all supported starting states.
4. Exercise invalid signatures and cache contents, missing content/network loss,
   disk shortage, concurrent maintenance, interrupted staging/activation/migration,
   failed apps and authorized boot failure. Test native fallback with compatible
   application/schema state and preservation of later writes; unsafe downgrade must
   refuse or require compatible forward repair, not restore an old database.
5. Verify one effective update/maintenance owner: Zincati/native root maintenance
   cannot silently bypass Soda qualification; download-only state does not activate
   unexpectedly on a generic reboot. Preserve recovery console/operator access and
   accurate status. Diagnose the Cockpit origin defect against the actual caller;
   never add a fake remote or weaken signatures to hide it.

**Checks/exit:** protected native evidence binds candidate/ISO hashes, architecture,
reviewed test-driver and baseline. Compare shipping bytes before/after tests. Missing
required tests, ambiguous fixture state or failed gates stop final admission.
Bootstrap fixtures may prove an update mechanism without promising a retained-install
migration; the `native-install-upgrade-recovery` class still requires actual update/
recovery evidence. Synthetic trust proves only its stated local scope. Wider advertised
architectures, migrations and operational policy commissioning follow the replacement.

### Milestone 5 — integrate protected signing and delivery

**B5; not started.** Connect the existing mechanisms; do not commission a release
service before replacing its builder.

1. Reuse `internal/releasedelivery` and the reviewed protected worker protocol for
   P7 and P10–P11. Keep separate build, qualification and signing/publishing authority,
   exact-digest permits and durable ledgers. Extend strict metadata only for B1's
   required provenance/media fields; test new and retained-format consumers.
2. Artifact signing after static admission authenticates the candidate for native
   tests. After qualification, sign the final release binding ISO hash/size/location
   and evidence, without changing host/app/ISO bytes. Bootstrap trust is independent
   of the artifact/channel it authenticates. Artifact signatures never approve stable.
3. Wire optional GHCR/media delivery and channel-last selection into the same command.
   Use existing native transports, local signed-directory fixtures and controlled
   publisher doubles for integration checks; do not add another build recipe or
   generic transport framework. Normal/emergency requests use the same sequence.
4. Test wrong role/repository/signer, changed bytes after tests, unknown/incomplete
   formats, expired/replayed offers, missing architecture/media, interrupted writes,
   duplicate observation, credential refusal and failed qualification. Uncertain
   publication holds the ledger; observing an effect is not permission to repeat it.

**Exit:** one noninteractive local run reaches signed final metadata; protected
admission and the optional delivery interface have native local/failure evidence.
Public registry/ISO commissioning is explicitly **not** a prerequisite for B6. Until
it is separately proved, no public/production delivery claim or channel movement
follows from local tests. Real signing-worker changes still require their exact grant;
fixture keys/credentials stay isolated and cannot become production authority.

### Milestone 6 — retire old lanes and prove the replacement

**B6; not started.** Finish source cutover before building operational automation.

1. After B3/B4 native exits pass, remove `build-native.sh`, `build-iso.sh`, old
   `tools/soda-host-image` orchestration, `--legacy-native`, host-content-only release
   modes, writable-bundle production/sealing and obsolete metadata/staging build
   paths. Remove `build-progress.sh`, `build_progress.py` and the Go/Python bridge.
   No successful compatibility wrapper may preserve a second producer indefinitely.
2. Rewire `scripts/check-native.sh`, `scripts/check-source.sh`,
   `internal/acceptance/remote_executor.py`, `package.json`, AGENTS commands and
   support recipes to consume the new candidate/ISO without rebuilding. Retain
   legacy artifact readers and authorized maintenance consumers where needed; do
   not delete retained bundles, projects, fixtures, credentials or installations.
3. Run the complete command on the authorized native worker: frozen source through
   signed candidate, ISO, install/update/recovery and signed final release metadata.
   Exercise publication/refusal interfaces under B5's local scope. No timer or
   public channel is needed for this qualified-but-unpublished replacement receipt.
4. Review the actual call graph and all production build/orchestration LOC against
   B1. Shipping work occurs once, install/update digests match, timing covers the run
   and verification remains intact. Moving code into helpers is not simplification.

**Exit:** the old competing producers are gone; one smaller production build lane
has a complete native local receipt, with real effect/qualification limits explicit.
No old artifact is deleted as a source-retirement shortcut. Only now move on to
[operational commissioning](#after-the-replacement-operational-commissioning).

## Single-run release build contract

The proposed entrypoint is `tools/soda-build`, compiled as `soda-build` (not implemented):

```sh
soda-build --arch x86_64 --out "$PWD/.artifacts/releases/UNIQUE-RUN"
soda-build --arch x86_64 --out "$PWD/.artifacts/releases/UNIQUE-RUN" --publish candidate
```

Both requests use one artifact recipe and qualification path. No `--legacy-native`,
partial-host release mode, `--skip-tests`, arbitrary resume-at-phase or false success
on partial output. Omitting publication yields a scoped qualified local result, not
an unsigned/untested result called a release. Exact signer/fixture prerequisites are
checked before expensive work. During implementation, leaf/source checks remain
independently runnable and scoped; they are not another release product.

| Phase | Work and handoff |
| --- | --- |
| P1 Admit/freeze | Clean committed canonical source via `git archive`, approved base/native architecture/tool/package inputs, identity, baseline and exact resources; no later mutable ref resolution |
| P2 Dependencies/source checks | Frozen Bun dependencies, Go verification and source-only checks |
| P3 Shipping programs/assets/prepared tests | Go runtime and media tools once; Bun/TypeScript/Lit, locale, terminal, branding and verified upstream Tea assets; tests consume those outputs |
| P4 Application images | Produce or admit the five verified immutable component images using Podman/Containerfiles and recorded provenance |
| P5 Host image | Direct vendor assembly, image-time packages, bound core apps and retained-runtime archives on pinned FCOS/bootc |
| P6 Freeze/verify candidate | Native ELF/OCI/inventory/presentation/Quadlet/bootc checks and existing payload/candidate records |
| P7 Authenticate candidate | Protected exact-digest artifact signing after static admission, before privileged installation/tests |
| P8 Assemble media | Stock live media and native customization/xorriso plus the exact candidate and prebuilt console/tools; no component build |
| P9 Native qualification | QEMU/KVM and reviewed existing drivers test the actual ISO and admitted update/recovery baselines |
| P10 Finalize/sign release | Bind candidate, ISO checksum/location, compatibility and protected evidence without rebuilding tested bytes |
| P11 Optional delivery | Native skopeo/signatures, immutable image/media uploads and anonymous checks, then channel last; available only under applicable commissioning/grants |

One reviewed Go owner handles order, elapsed time, subprocesses, cancellation,
handoffs and the final scoped result. No new workflow framework, server, cache DB
or monitoring service. Python asset/media leaves can remain; upstream native tools
own compilation, image transport, installation and verification. Versions stay in
source manifests/locks; no incidental upgrade or predecessor updater import.

One new `.artifacts/releases/RUN/` contains `inputs/`, `work/`, `artifacts/`, `evidence/`,
`release/` and `logs/`. Preserve fresh-output refusal and every failed attempt.
Existing build-relative paths in the frozen source can be intermediate work, not
another writable release payload. Existing payload/candidate/release models own
identities; do not create a parallel orchestration inventory. Credentials, protected
permits and authority ledgers stay outside public artifacts.

The controller is reviewed/protected; arbitrary build scripts and build-supplied
tests never run with its signing authority. Qualified local, published, failed,
cancelled and held are distinct outcomes. [Installation](installation.md#build-timing-and-progress-implementation-plan)
owns timing, redaction, cancellation and failure-reporting requirements.

**No qualification/signing cycle:** the ISO contains authenticated candidate content
and integrity-controlled bootstrap inputs, not its own post-install test report.
The final signed release document adds the ISO checksum and qualification afterward.
A trusted downloader verifies that binding before privileged media use. Testing
with fixture trust is not commissioning public production trust.

## Scope and ownership

One appliance release is an exact tested host/application combination; it need not
change every component. Existing Project OS roots, marketplace apps and CI job
environments do not automatically adopt new images with a host update.

| Owner | Responsibility |
| --- | --- |
| This plan | Replacement milestones, release identity/trust/qualification and later commissioning order |
| [Implementation status](implementation-status.md) | Current replacement progress, retained release state and exact grants |
| [Installation](installation.md), [installer](coreos-installer-plan.md) | Media, first-machine setup, timing and retained-install maintenance procedures |
| [Native support](native-support.md), manifests/locks | Existing tool effects, inputs and verification |
| [Native validation](native-validation.md) | Reused product journeys and architecture-specific evidence |
| [Credentials](dashboard-credentials.md) | Schema and consistent credential/database preservation |
| [Project OS](project-os.md), [Services](services-and-ai-plan.md), [OS strategy](os-product-strategy.md#update-ownership) | Runtime ownership and separately selected project/marketplace maintenance |
| [Development handoff](development-handoff.md) | Other retained targets and scoped grants, not automatic release fixtures |

The selected direction is FCOS-derived bootc with its OSTree backend, not a CoreOS
source rebuild. Existing writable installations are not migrated by selecting it.
If its native install contract fails, bring back the concrete limitation; there is
no automatic stock-CoreOS-plus-Soda-bundle fallback in the new build. General fleet
execution/telemetry, public appliance ingress, arbitrary app rollback, project-root
replacement, high availability and the predecessor Updates platform remain out of scope.

### Local candidate content and machine-state ownership

These existing complete-candidate contracts are retained when their producer changes:

- `/usr/share/soda/release.json` binds the five exact app images, source/base, schema
  and content hashes. Core Forgejo/dashboard/proxy Quadlets alone use
  `/usr/lib/bootc/bound-images.d` and caller-scoped additional storage at
  `/usr/lib/bootc/storage`. Preserve OCI manifest identities through transport;
  do not globally attach bootc storage to ordinary Podman workloads.
- Project OS/Tailnet archives under `/usr/share/soda/images` are verified/imported
  by fixed-input root-only `soda-image-import` into ordinary retained Podman storage.
  It does not start, replace or delete images/containers. Required project/companion
  backing layers remain outside bootc GC. Host updates do not select their lifecycle.
- Vendor defaults apply to new creation. `/etc/soda/host.json` remains machine-owned;
  conflicting saved image selections refuse explicit migration rather than being
  rewritten. Existing root/profile IDs remain authoritative. Companion admission
  binds run CID/CreateCommand to its configured immutable image; no hot adoption or
  companion-image upgrade is qualified. Native proof covers coordinated turnover.
- `/usr/share/soda/defaults` contains public examples, not secrets. Native OSTree
  configuration merging handles image `/etc` defaults. Machine identity, TLS,
  accounts, databases and `/var` roots never enter the generic candidate. First
  install selects actual subnet/operator state; it is not initialized in the image.
- The [Forgejo owner](forgejo-frontend-integration.md#image-owned-presentation) owns
  immutable presentation/data separation. `payload.json` is the embedded record;
  `candidate.json` adds host identity and payload byte hash without circular digest.
  Neither alone grants channel authority. Preserve canonical artwork and licenses.
- The current x86_64 RPM lock pins added NEVRAs and full inventory for its exact base.
  Missing upstream versions fail closed. Repositories/RPMs are not mirrored; byte
  reproducibility is not claimed. Aarch64 needs its own transaction/native evidence.

## Release identity and trust

A signed final release binds its unique serial/ID and normal/emergency class,
CoreOS version/digest, approved Soda revision, actual component-producing provenance,
architecture-specific host/app identities, package inventory, schema/compatible
starting releases/intermediates, activation/reboot/migration limits, media hashes,
qualification and notes. Use upstream formats where sufficient; no second app
inventory or credentials/initialized state. Keep notices and security-triage inventory.

Candidate, preview and stable select already qualified immutable artifacts. Tags
are discovery conveniences, not mutation authority. Promotion never rebuilds. Missing
architectures/unsupported starting states refuse rather than falling back to a sibling.
Public pull should require no developer token. Keep selected GHCR namespaces during
the rewrite; namespace consolidation and a stable discovery domain are not prerequisites.

### Implemented trusted-delivery contract

Reuse `internal/releasedelivery`, `tools/soda-release` and
`internal/releasedelivery/tools.json`; [native support](native-support.md#trusted-release-delivery-worker)
owns current invocations. These workers do not install/import images, change global
policy or reboot. The following security contracts survive the build replacement.

- **Native signatures:** keyed P-256 Sigstore through skopeo/containers policy, not
  a custom envelope or assumed keyless/Rekor service. Artifact/release repositories
  use the artifact role; candidate/preview/stable use separate role keys and exact
  repositories. Native policy uses `sigstoreSigned`, `exactRepository` and scoped
  `use-sigstore-attachments`. Preserve unrelated vendor trust; existing Soda overrides
  require review, not automatic replacement.
- **Signing snapshots:** private fresh copies, admitted digest rechecked, old signatures
  removed only from that new snapshot, then native signing and role verification.
  An existing signature cannot mask a wrong signer. Registry overlap appends verified
  same-role signatures; never remove retained registry evidence as routine signing.
  Restricted file/stdin inputs only; no secret in source, argv, output or public artifacts.
- **Documents:** non-executable standard OCI images contain `record.json`. Release
  records embed exact payload/candidate bytes, serial, class, provenance hashes,
  qualification and notes. Hashing/parsing alone is not signature verification.
  `local-only` evidence cannot enter preview/stable, which require protected
  `native-install-upgrade-recovery` evidence. A build's `passed` JSON grants nothing.
  Metadata changes require new serials; promotion reuses the final document.
- **Channels:** bind name, strictly increasing sequence, issued/expiry times and exact
  per-architecture release references, or explicit withdrawal. Bootstrap `NotBefore`,
  minimum sequence and freshness limits remain: offer age 60 seconds–7 days, clock
  tolerance at most 300 seconds. Credible authenticated time is required; backward/
  out-of-range time fails closed. Refresh with a higher sequence, not rebuilt images.
  Expiry/withdrawal stops new uptake, not installed workloads or database state.
- **High-water state:** preserve highest trust epoch, channel sequence/digest and
  observed release serial/digest per architecture. Reject older serials, same-serial
  forks, same-sequence substitution and trust rollback; identical fresh observation
  may repeat. Authenticated observations persist even when content later disappears.
  Missing/corrupt state never silently initializes; existing state cannot be reset.
  Native boot fallback is separate and never rewinds authority floors.
- **Authority:** one reviewed protected owner issues exact-digest permits. Build/test
  code cannot read signer/passphrase/auth inputs or select privileged trust/commands.
  Native subprocesses discard ambient credentials/config and secret-bearing error
  output. File-mode tests do not prove isolation from a sudo-capable build account.
  Successful production signing is noninteractive; exceptional reconciliation is not
  a per-release human ceremony.
- **Publication:** preserve digests, upload immutable content/signatures first, then
  copy back using native policy and anonymous access. Validate every advertised
  architecture/media object before channel selection. Bootstrap absence differs
  from auth/transport failure; previous channel identity must match permit/history.
  Readers pin, verify and admit the document before verifying selected artifacts;
  cache existence or inspect-only reads are not acceptance.
- **Uncertain writes:** GHCR lacks CAS here. One protected publisher and persistent
  locked ledger per repository records intent before effects. Observe uncertain
  writes without replay; ambiguous outcomes stay held for explicit reconciliation.
  `--observe` after permit expiry can confirm an effect, not authorize a fresh write.
  Preserve old supported intermediates, withdrawals and failed attempts; no pruning.
- **Bootstrap/rotation/recovery:** deliver independently trusted public policy through
  an authorized installation/update path, never from its own unauthenticated channel.
  Same-role rotation increments trust epoch, proves overlap and re-signs supported
  content before old-key removal. Revocation, offline clients, key/ledger loss and
  off-machine recovery need scoped drills. A local encrypted backup is not builder-loss
  recovery. Do not regenerate real keys or replay completed bootstrap for this rewrite.

Real artifact/candidate keys and eight immutable `ghcr.io/levitateos/sodaos-*` packages
already exist under owner-selected `veighnsche`. Their authenticated native round trips
passed, but visibility was last observed Internal and no mutable candidate channel was
selected. Public bootstrap trust is in `appliance/keys/release-trust.json`, not installed
into global policy. `/var/lib/soda-release` retains reviewed workers, credentials,
permits, ledgers and attempts. These are reusable custody, not proof of unattended
isolation or a reason to finish public commissioning before B1–B6.

## Native qualification, activation and preservation

The replacement must prove a real native path, not merely source/ISO packaging checks.
Reuse existing drivers and feature contracts, with exact fixture/action grants.

| Area | Evidence for claimed support |
| --- | --- |
| Build/trust | Frozen inputs, native ELF/OCI, complete inventory, signatures and rejection of tampering/wrong authority/missing content |
| Fresh install | Actual matching ISO, per-machine setup, offline first boot/media removal, protected operator access, enforcing SELinux |
| Update | Both same-base and separately qualified new-base cases; populated data/projects and explicit supported intermediates |
| Compatibility | Matching helper/API/browser graph, Forgejo/packages, units/Quadlets, configuration and required image availability |
| Maintenance | One owner, exact staged release, download-only versus activation, notices/deferral and no unqualified upstream bypass |
| Failure/recovery | Network/space/maintenance conflicts, interruption and app/boot failure; console access, compatible fallback/forward repair, later writes preserved |
| Delivery | Native local signature/metadata proof and injected incomplete/expired/withdrawn/interrupted publication; public production proof is a later commissioning gate |

Activation verifies authority/path/architecture, preflights space/config/schema and
maintenance conflicts, stages without disrupting current service where supported,
then reacquires/rechecks maintenance ownership before mutation. Quiesce writers for
consistent backups where required. Activate the paired host/app contract in tested
order, reboot when host content requires it and perform bounded product/health checks.
A successful process or bootc exit alone is not application acceptance.

Expose current/base/offered versions, verification/staging, disruption, partial failure
and recovery through native CLI/console and supported operator UI. Status must not
mutate/enroll/reboot. Preserve stock administration; resolve Cockpit's
`ostree-image-signed:docker://quay.io/fedora/fedora-coreos` remote error using exact
installed caller/source evidence, not a fake OSTree remote or weaker trust.

Preserve machine identity, credentials/Tailscale state, Forgejo/Soda databases,
repositories, roots/accounts/keys, workloads and settings. OS fallback never rewinds
shared `/var`. Classify backward-compatible, intermediate-required and unsafe-downward
migrations; unmet backup/recovery prerequisites refuse activation. Database restore
requires separate authorization and an explicit recovery point/later-write decision.
Do not blindly replay migrations or claim a lock controls arbitrary external root work.

Maintenance windows control timing, not whether terminals/CI jobs are idle. Notify
about interruption; no promise that process memory survives reboot. Define bounded
deferral and separately approved urgent-security policy before production operation.
An emergency release does not authorize unconfigured forced reboot. Provider/Tailnet
registration needs its own grant; it is not a hidden update-test prerequisite.

## After the replacement: operational commissioning

**Begin after B6, using the completed command—not before or as another build lane.**
These are downstream release-service outcomes, not the active replacement milestones.
Preserve existing acceptance requirements; separating order does not waive them.

| Later outcome | Work / acceptance boundary |
| --- | --- |
| Public delivery commissioning | Observe existing protected/GHCR state; owner changes package visibility to Public; verify anonymous signatures/digests and candidate failure/withdrawal/observation behavior. Use fresh higher-sequence offers/permits, never expired bootstrap state. Commission proposed GitHub Release ISO delivery under its separate grant. |
| Unattended operation | Agree exact CPU/RAM/disk/VM limits, identities, uptime, retained-output budget, notification destination and staging-ref admission; install one reviewed systemd timer/one-shot only under host/action approval. Complete isolation and recovery custody before enabling it. |
| Production readiness | Native receipts for every advertised architecture and upgrade path; representative writable-install migration rehearsals with consistent backups/later-write preservation; media installs the qualified release and accepts its next update. Assign release/security owner and numeric response/observation targets; execute rotation/revocation, builder-loss, emergency and interrupted-publication drills. |
| Progressive launch | Exact production release/appliance approval, protected preview/stable gates, observed maintenance activation and preservation checks. Public GHCR delivery is not installed-fleet acceptance. |
| Ongoing train | An actual new CoreOS stable event plus approved Soda changes completes the same build/test/sign/publish/promote command automatically, and an independent emergency trigger uses it too. Synthetic events prove mechanics, not the live upstream train. Reuse valid drill evidence instead of replaying effects. |

### Automated trigger and initial local builder

This development/build machine remains the proposed initial host, not an appliance
being updated. Inspect actual architecture/resources/KVM/other workloads before
commissioning; readable KVM does not establish safe fixture ownership or isolation.
The proposed native timer polls supported CoreOS stable metadata (hourly is an initial
recommendation, not an installed setting). No listener/webhook/central update server.
A CI schedule is an alternative execution host, not a second concurrent publisher.

Freeze the exact base and explicitly approved release-ready Soda revision. Do not
include an uncommitted tree or everything on a moving branch. Serialize admission
and publication using native locking and existing durable candidate/results/ledger
state. No-change/duplicate polls do not rebuild; new staged changes wait for the next
candidate. Handle malformed metadata, network failures, missed releases and downtime
catch-up without invalid direct upgrades or a tight failed-candidate retry loop.
The scheduler never switches the canonical checkout or creates a worktree.

Normal and emergency admission call the same noninteractive entrypoint and gates.
A Soda fix may use the last qualified base; an urgent base fix must not wait for
unfinished features. Upstream stable cadence is variable, not a same-day release
promise. Shorter observation needs a documented risk decision, not hidden failed
checks. Initially hold promotion explicitly; enable automatic progressive promotion
only after its evidence and production-automation grant. Successful routine signing
never requires human attendance. Uncertain effects remain held and notify the owner.

Without fleet telemetry, use explicit controlled preview evidence; downloads are
not proof of fleet health. Appliances independently verify/pull/stage and activate
under their maintenance policy. Automatic publication is not simultaneous reboot.

### Media, architectures and migration

GitHub Releases on `LevitateOS/sodaos` is the proposed ISO download destination,
separately authorized from GHCR. The final signed release binds ISO hash/size/location.
Stage artifacts before offers; candidate media is a prerelease, not GitHub Latest.
Stable metadata changes reuse tested bytes. No atomic GHCR/GitHub Release transaction
exists: retain intent and observe uncertain writes; never re-upload merely to repair
readback. Private-network/per-machine media is never a general public release.

Start the replacement on native x86_64. Later native aarch64 uses the same code with
its own package transaction, worker and receipt. Do not advertise unavailable
architectures; multi-architecture selection requires all advertised identities and
receipts. Full two-architecture production readiness requires both, but one native
architecture's implementation is not gated on its sibling.

Rehearse writable-to-image migration on representative isolated starting states,
including old units/overrides, package layering, app/companion references, schema,
config and trust/update ownership. Unknown conflicts refuse; do not replay first
install, blindly reset layered packages or remove whole directories. Actual retained
machines need fresh inventory and their own grant even after a rehearsal passes.

QCOW2 is not implemented or required for the first ISO replacement. If later selected,
it consumes the same candidate rather than assembling another OS. Offline update
import, broader cohorts/telemetry and moving to a dedicated host are follow-ups;
this does not defer the replacement ISO's offline installation/first-boot content.

## Workstream status and next action

**Next: finish B1's exact native filesystem-install/first-boot proof, then B2–B6.**
B1's caller/source/native CLI review, handoffs and deletion baseline are recorded;
no disk installation or bootc stored-image copy has run. None of the replacement
milestones is complete. The old local candidate at `45ac843`,
native trusted-delivery source and protected bootstrap, and `d054a60`/`fde23d0`
transitional tests remain evidence to reuse. They do not mark milestones of the
replacement complete. The [status](implementation-status.md) records exact artifacts,
blockers and grants; [feasibility research](release-engineering-feasibility.md) is
historical source evidence, not a selected deployment or prerequisite to rerun.

Current installer usability is preserved until B3/B4's native proof permits source
cutover. This is not permission to keep producing writable bundles in the new run.
Source retirement never deletes old builds, installations or fixture state.

This documentation rewrite grants no signing-worker installation, real publication,
trust/network change, VM/service lifecycle, migration or cleanup. Use the latest
[target/action permissions](implementation-status.md#current-permissions). In
particular, existing candidate GHCR commissioning is not ISO publication, preview/
stable promotion or unattended operation authority. Finish independent approved
source work; report any exact blocked native/effect gate without inventing broader
approval rounds or weakening its check.
