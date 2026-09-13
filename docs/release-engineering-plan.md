# SodaOS release engineering

## Status and decisions

**Milestone 1 is complete at native x86_64 local-build/inspection scope (`45ac843`).**
The complete host/app payload is exported; bootc lint passes with 13 checks and one
skip, and all 391 immutable Forgejo files plus embedded runtime archives are verified.
[Receipt](implementation-history.md#complete-local-appliance-candidate).
Six milestones carry the work through automated production operation; the 28 detailed
items are acceptance criteria, not separate execution/approval rounds. **This remains
unsigned/unpublished, not an installable or boot/upgrade-qualified release.** The
owner selected GHCR distribution and a CoreOS-aligned Soda release train with an
independent emergency lane. This guide owns the release engineering workstream and
its status: build engineering, release management, distribution and appliance updates.
Appliance updates are part of this workstream, not a separate platform. The owner
approved the first source/upstream investigation; its
[receipt and recommendation](release-engineering-feasibility.md) propose derived
FCOS with bootc's OSTree backend and logically bound core app images for proof.
This is not a validated migration or production mechanism. The work does not
authorize registry publication, automatic CI, signing-key
creation, trust changes, appliance migration, service/VM lifecycle or cleanup.
Consult the [handoff](implementation-status.md#current-permissions) before native work.
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
| [Handoff](implementation-status.md) | Retained targets, active grants and shared state |

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

Select the GHCR organization/repository names before publication; none is assumed
available. Public pull is the recommended appliance default so installation does
not require a personal GitHub token. Confirm visibility, licensing, bandwidth,
retention and anonymous-client behavior before adopting it.

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

Production signing and appliance verification are mandatory. Stage 1 selects a
native-supported signature scheme, trusted signer identity and bootstrap mechanism;
keyless signing is not assumed to work with the chosen OS client.

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

## 5. Normal and emergency release process

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
5. Sign and publish exact approved artifacts, promote to preview, then stable after
   bounded observation and explicit release-owner approval initially.
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
- Build and test those frozen inputs, then sign and publish the exact passing
  artifacts. Do not rebuild after testing. Restrict signing/promotion to a separate
  protected execution step; ordinary build scripts and test workloads must not have
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

The same entrypoints should move to a dedicated builder or scheduled GitHub Actions
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
Requirements remain in sections 1–8. Historical Stage 1/2 meant feasibility/initial
local production, not these milestone numbers.

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

**8. [ ] Implement trust and channel verification with local fixtures.**

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

- Deliverable: detect/admit → build → tests/native qualification → protected signing
  → publish → preview, with explicit initial stable promotion. The emergency entry
  uses the same pipeline and serialization without waiting for the CoreOS trigger.
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

- Deliverable: ISO delivery linked to the same signed release and trust bootstrap;
  include QCOW2 when its separately owned producer is implemented and advertised.
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

Offline import, broader fleet cohorts/telemetry and moving the builder to other
hosting are follow-ups, not prerequisites for the core online automated train.
General fleet orchestration and arbitrary application-data rollback remain out of
scope. Completion does not grant future destructive maintenance, cleanup or changes
to the predecessor's separately reserved Updates platform.

## 10. Workstream status and next action

The [six milestones above](#9-implementation-stages-and-exits) are the single task
list for this workstream. **Milestone 1 is complete for the x86_64 local candidate
at `45ac843`; milestones 2–6 remain pending.** This consolidation changes execution granularity,
not production gates or effect permissions.

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

**Next: milestone 2 — trusted delivery.** Source/synthetic verification can be built
without enabling publication; actual GHCR/signing/trust effects require their
applicable grant. No timer, native fixture or retained-appliance migration is implied.
The Cockpit error still needs installed-version/caller confirmation. No retained
target, registry, workflow, signing key or update client has been changed. Independent
Tailnet tasks remain with their own workstream; this plan does not absorb their list.
