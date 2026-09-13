# SodaOS release engineering

## Status and decisions

**Planning selected; implementation and native acceptance have not started.** The
owner selected GHCR distribution and a CoreOS-aligned Soda release train with an
independent emergency lane. This guide owns the release engineering workstream and
its status: build engineering, release management, distribution and appliance updates.
Appliance updates are part of this workstream, not a separate platform. It does not authorize registry publication, automatic CI, signing-key
creation, trust changes, appliance migration, service/VM lifecycle or cleanup.
Consult the [handoff](implementation-status.md#current-permissions) before native work.
The predecessor repository and its reserved Updates platform remain separate.

Selected product decisions:

- Stage qualified Soda changes while following Fedora CoreOS **stable** releases.
- A new stable base triggers a Soda candidate combining that exact base with ready
  Soda changes. Qualification precedes promotion; there is no same-day promise.
- Publish signed production releases through **GHCR**. Appliances pull approved
  updates; publishing does not immediately reboot every appliance.
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
2. Detect a new upstream stable release using supported upstream metadata. Scheduled
   polling in authorized CI is sufficient; no custom listener daemon is required.
3. Resolve immutable inputs and build a candidate. Duplicate triggers must not create
   competing promotions or silently change the candidate's contents.
4. Qualify it using the matrix below. If a Soda feature blocks the new base, omit it
   or use the last compatible Soda version; a base incompatibility remains a real
   blocker requiring triage, not permission to publish a known-broken combination.
5. Sign and publish exact approved artifacts, promote to preview, then stable after
   bounded observation and explicit release-owner approval initially.
6. Record supported upgrade paths, release notes, interruptions and known limitations.

### Emergency lane

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

Each stage is a coherent work package, not permission to execute native effects.

| Stage | Work | Exit |
| --- | --- | --- |
| 1 — feasibility and decisions | Audit actual layout, selected upstream versions, image build/consumer/signing support, Zincati ownership and Cockpit error | Concrete mechanism decision, source references, unresolved limits and exact isolated proof proposal; no speculative adapter |
| 2 — local candidate production | Derived host recipe or explicitly selected alternative, pinned apps/release record, verifier, provenance and trust fixtures | Reproducible-input local candidate and negative verification tests; no publication |
| 3 — isolated native upgrade | Authorized fixture with synthetic persistent state; initial release, next release, interruption and recovery cases | Actual activation/preservation/recovery receipts on the selected architecture |
| 4 — release delivery | Separately authorized GHCR repositories/visibility, signing authority, protected manual promotion and consumer discovery | Signed round trip, incomplete-publication refusal, retention/rotation procedure; no public automatic rollout |
| 5 — operator maintenance | Native status, explicit activation, then download/window policy and preview promotion | Effective policy, actionable failures and bounded progressive rollout proved |
| 6 — migration and production readiness | Existing-install migration, matching ISO/QCOW2 linkage where available, emergency drill and release ownership | Explicit supported starting states, architecture evidence, operational response targets and approved first production release |
| 7 — subsequent improvements | Offline import, optional broader cohorts/automation | Separate bounded acceptance; not prerequisites for the initial explicit online flow |

Migration must inspect and preserve each target's current origin, layered packages,
writable payloads, config/data and later writes. Do not replay first-install, clone a
credential-bearing appliance into an image or rebase retained fixtures as a shortcut.

## 10. Workstream status and next action

- [x] Agree normal CoreOS-aligned train and emergency lane.
- [x] Select GHCR for OCI distribution and mandatory production signing.
- [x] Record target architecture, preservation boundaries and staged implementation.
- [ ] Stage 1: inspect upstream/native feasibility and select exact mechanisms.
- [ ] Stages 2–7: not started.

**Next:** source/upstream research for Stage 1, resulting in a small decision table
with exact versions and a bounded local/native proof proposal. No current target
was inspected or changed for this plan. No registry, workflow, signing key, image,
update client or release has been created. Independent Tailnet tasks remain with
their own workstream; this plan does not absorb their task list.
