# Audit ownership and collaboration

These are the assignments for all 80 slices in the [catalog](slices/README.md),
using the [shared review format](review-format.md). Defining assignments does not
launch the audit, create slice review records or authorize implementation.
Review ownership is responsibility for audit work and its record; the user
retains product and engineering decision authority.

The [prepared review input baseline](review-baseline.md) pins source at
`f7e9cf9db616f93de351dd44c9bb7456feb608b9` and identifies the uncommitted
guidance by its actual byte hashes. The pre-review guidance bytes are now
retained as described there while live review records and plan updates evolve.
The user started source review; the source pin does not advance the older
catalog/structural evidence scopes or grant implementation authority.

## Guidance conflicts and controlling decisions

Read this register before assigning review questions or specifying a language
correction. These are preparation observations, not completed slice validity
findings. The conflicting repository wording was inspected at
`f7e9cf9db616f93de351dd44c9bb7456feb608b9`; the structural/coverage baselines
remain those recorded in the plan. The AGENTS refresh and this register are
uncommitted documentation changes. The current user instruction identifies
AGENTS.md as outdated and requests its refresh; the user's supplied collaboration
instructions control authorization. The language policy is the existing owner
decision recorded in the [plan](ideal-filetree-plan.md#owner-language-policy),
not a new decision made by this register.

The coordinator maintains this table and receives new conflicts from all
reviewers. Record the exact guidance revisions and claim, its affected
responsibilities, the applicable owner decision and its provenance, and either
the supported resolution or exact unresolved question. A current plan assertion
is not evidence of installed behavior. If a recorded decision's scope or
provenance is materially disputed, keep the affected correction pending and ask
the owner; do not resolve it by counting documents or reviewer agreement.

| Conflict ID | Conflicting guidance | Applicable decision or evidence | Disposition and affected review |
| --- | --- | --- | --- |
| GUIDANCE-01 | [AGENTS.md](../../../AGENTS.md) formerly required Go for backend/setup/privileged integration; the [owner policy](ideal-filetree-plan.md#owner-language-policy) records Rust for system/privileged applications. | The user identifies AGENTS as outdated. Use the recorded Go server/Rust system policy and the named retained owners in [package ownership](package-ownership.md#boundaries-that-should-remain), including Go domain/coordinator/store/client responsibilities. | Superseded blanket instruction corrected in this documentation refresh. All implementation allocations must cite their applicable owner decision; this does not port every Go file or prove a cutover landed. |
| GUIDANCE-02 | AGENTS formerly directed automatic local commits and treated task approval as broad code, fixture, provider-call and operational permission; the user's supplied collaboration instructions restrict normal-mode authorship and require matching operational instructions. | The current user instructions govern authorization independently of technical evidence or review readiness. | Superseded permission grants corrected in AGENTS. Applies to every worker and the coordinator; no audit, implementation, Git mutation or external operation is authorized by an assignment or this register. |
| GUIDANCE-03 | [Go ownership](../go.md) placement tables/directives name Go privileged host executors while its daemon paragraph and [Go package convention](../go-packages.md#package-boundaries) describe Rust cutover. Structural pending-cutover observations were historical. | Current source has Rust host binary/native adapters, its manifest and release-image `build.rs:522-553` compile selection. Superseded Go daemon/executor paths are absent; [architecture assertions](../../../internal/archcheck/arch_test.go) enforce that boundary. The living placement/port/topology descriptions now distinguish this source status from installed proof. | Retain Rust host and surviving Go client/wire. Primaries for Project/Space/Factory/Networking/D03 exchange remaining duty/caller/build/install evidence with H06. Stale owning guide tables remain a documentation correction; do not recreate a Go executor or plan deletion of already absent code. |
| GUIDANCE-04 | [Go ownership](../go.md) routes release construction to Go `release/` packages; [port assessment](port-assessment.md) records the Rust release pipeline cut over, while other structural tables retain earlier Go retirement wording. | The recorded policy retains the four Rust release crates. Historical structural observations do not override that language decision or establish which cutover duties remain at the review baseline. | Retained language is Rust; D02-D11 and H06 reconcile actual callers, manifests, build/install wiring and predecessor disposition before specifying remaining actions. Refresh stale guide wording only under its matching documentation instruction. |
| GUIDANCE-05 | [Go ownership](../go.md), [architecture overview](../../architecture/overview.md), [credentials](../../reference/credentials.md), [Factory reference](../../reference/factory.md) and the explicit [storage decision gate](../../factory/decision-gate.md) retain SQLite guidance, superseded by PostgreSQL source. The SQLite gate landed in `09332bfbceaf47d40553ff48b033cc21344c0165`; the PostgreSQL port landed in `71045ed687468f8f8c3a3727fb570d198e16d4c9`. | On 2026-10-05 the owner answered: **“Yes—retain PostgreSQL; SQLite guidance is outdated.”** This resolves intended backend choice. Current [store implementation](../../../internal/store/store.go) and [schema](../../../internal/store/schema.go) supply implementation evidence; the historical SQLite single-connection proof does not validate current PostgreSQL concurrency. | [H02-F1](reviews/H02.md#concrete-findings) owns the guidance correction and owner resolution. I03-I10, Factory persistence callers and H06 use PostgreSQL as the intended backend and assess its actual invariants. No migration, alternate schema owner or additional process follows from refreshing the plan. |
| GUIDANCE-06 | Historical [port assessment](port-assessment.md) wording reopened language for Factory CLI/Muse despite the recorded Rust allocation. The living assessment now retains that allocation. | A technical recommendation cannot override the owner's recorded language choice. Consolidation can be assessed without changing a language or process boundary. | Retain Rust targets and assess actual duties. Any proposed language change remains an owner question; do not mark it executable while unresolved. The [Factory CLI map](coverage/maps/soda-factory.md) assigns F08/H01/H03; the [native inventory](coverage/inventory/native-packages.md) assigns each Muse helper's mixed responsibilities across A/C. |

| GUIDANCE-07 | [Forgejo reference](../../reference/forgejo.md#factory-api-boundary) says Factory uses stock issue/pull/review APIs, has no merge or commit-status write and observes Actions run records. The current [Factory interfaces](../../architecture/factory-interfaces.md) and [trust contract](../../architecture/trust.md#conditional-native-mutations) require attributed conditional native writes, fresh shared snapshots and complete check/review evidence. | Apply the owning product/trust model and established target interfaces per slice. The stock-API summary cannot negate their explicit conditional merge/review requirements. Actual Go adapters and controller callers are implementation evidence, not authority to invent another native protocol. | G02/G05/G06/G07/G08 and F06/F10/F11/F12 reconcile the exact applicable caller contracts. The outdated reference summary is a documentation correction; external Fountain/SDK behavior and installed results remain separately unverified. Do not create an alternate Actions poller or status writer to satisfy stale prose. |
| GUIDANCE-08 | A review could incorrectly treat the breadth of Forgejo template overrides as architectural debt or recommend restoring stock pages. | On 2026-10-05 the owner explicitly confirmed: **“I deliberately overrid all the pages with our own design”** and said the overrides are fine as they are. Full Soda design coverage is intentional. | Preserve all-page override scope and the owner's design in G08, placement and the desired tree. Review correctness, native route/state integration, assets and test assumptions within that model. Override breadth alone does not justify removal, stock-page replacement or redesign. This decision does not certify implementation behavior or authorize code changes. |

Exact observation locations at the recorded repository revision:

- GUIDANCE-01: AGENTS.md 151-154; ideal-filetree-plan.md 43-50.
- GUIDANCE-02: AGENTS.md 73-78 and 104-137; controlling instructions are the
  owner's collaboration instructions supplied in this conversation.
- GUIDANCE-03: go.md 25-35, 42-50 and 65-68; go-packages.md 80-82 and 102-104;
  placement.md 45 and 55-59; port-assessment.md 12 and 72-78;
  internal/archcheck/arch_test.go 222-233.
- GUIDANCE-04: go.md 34 and 58-59; port-assessment.md 19;
  package-ownership.md 54 and 99-104.
- GUIDANCE-05: go.md 36, 115 and 199; architecture/overview.md 20 and 163;
  reference/credentials.md 181; internal/store/store.go 1-15 and schema.go 12-17;
  package-ownership.md 117-120; factory/decision-gate.md 1-59 and reference/factory.md 98.
- GUIDANCE-06: port-assessment.md 17; ideal-filetree-plan.md 43-50.

Every primary records applicable conflict IDs and the controlling decision in
its slice record before marking the intended model established or a dependent
language correction specified. A superseded rule with a supported controlling
decision does not need another permission request. An unresolved conflict keeps
only its dependent model conclusion/correction pending; independent review can
continue. The assigned challenger checks the decision's applicability as well
as technical evidence. Record `none identified within inspected guidance` with
scope when no conflict applies; an empty field is insufficient.

Reconcile this register after merges and new owner decisions. Replace stale
current descriptions with supported dispositions instead of accumulating a
chronology. Linked owning guides still need their own authorized corrections;
this preparation step does not silently rewrite their contracts.

## Reviewer roles and concurrency

Use three GPT-6.1 Sol reviewer workers plus this task's coordinating assistant.
This session has four concurrent agent slots including the coordinator. Primary
review and challenge work rotate within those three worker slots; the roles do
not require six simultaneously running agents.

| Role | Planned agent task name | Model | Primary responsibility groups | Slices | Assigned challenger |
| --- | --- | --- | --- | ---: | --- |
| A | `audit_projects_identity_spaces` | GPT-6.1 Sol | Projects, identity brokering, Spaces and terminals | 28 | B |
| B | `audit_factory_forgejo` | GPT-6.1 Sol | Factory coordination, Forgejo integration | 21 | C |
| C | `audit_platform_delivery` | GPT-6.1 Sol | Networking, operator administration, release/installation, shared support | 31 | A |
| Coordinator | This task's coordinating assistant | Record actual model at execution | Complete coverage, evidence conflicts and shared plan updates | All 80, coordination only | Does not substitute for the assigned challenger |

The counts describe coverage allocation, not equal effort. The user separately
authorized binding/briefing and then audit start on 2026-10-05. Their runtime
identities below now own the connected opening passes and subsequent groups.
Start from the prepared input baseline and reconcile later source/guidance
drift. At an authorized audit start, record the actual bounded pass and its
controlling contract scope. Binding or acknowledging a brief does not establish
any slice's intended model or source validity.

| Reviewer role | Runtime agent binding | Active slice / bounded pass | Source and contract baseline |
| --- | --- | --- | --- |
| A | `/root/audit_projects_identity_spaces` — GPT-6.1 Sol | I01-I10, then Projects/Spaces | Pinned source and retained guidance; actual contract scope in each record |
| B | `/root/audit_factory_forgejo` — GPT-6.1 Sol | F01-F12, then Forgejo | Pinned source and retained guidance; actual contract scope in each record |
| C | `/root/audit_platform_delivery` — GPT-6.1 Sol | H01/H02, then other H/N/O/D duties | Pinned source and retained guidance; actual contract scope in each record |

Coordinator contact is `/root`. A sends challenges to C and receives challenges
from B; B challenges A and receives challenges from C; C challenges B and
receives challenges from A. These contacts are internal reviewer messages.

## Reviewer briefing packet

Each worker receives the exact target, source/guidance pin, full assigned slice
list, primary/challenger contacts and these shared documents before auditing:
[AGENTS](../../../AGENTS.md), [documentation authority](../../README.md),
[review baseline](review-baseline.md), [format and completion criteria](review-format.md),
this file's conflict/ownership/collaboration rules,
[catalog](slices/README.md), [coverage scope](coverage/README.md),
[all tracked inventories](coverage/inventory/README.md),
[all responsibility maps](coverage/maps/README.md),
[target language policy](ideal-filetree-plan.md#owner-language-policy),
[package ownership](package-ownership.md), [ports and cutovers](port-assessment.md)
and [post-merge maintenance](maintenance.md).

Common contract inputs are the [product overview](../../product/overview.md),
[scope](../../product/scope.md), [architecture](../../architecture/overview.md),
[trust](../../architecture/trust.md) and [factory interfaces](../../architecture/factory-interfaces.md).
The links below route each role to the existing contract owners and coverage
families; their wording is subject to the conflict register and current user
decisions. Briefing confirms receipt/navigation, not contract validity. Reading
relevant requirements and establishing each intended model remain audit work.

| Role and complete primary slice set | Owning contract inputs | Initial coverage navigation; full assigned coverage remains mandatory |
| --- | --- | --- |
| A: P01-P12, I01-I10, S01-S06 | [Projects](../../product/projects.md), [Spaces](../../product/spaces.md), [Project OS](../../reference/project-os.md), [credentials](../../reference/credentials.md), [terminal](../../reference/terminal.md), [API](../../reference/api.md), [TypeScript](../typescript.md), [Lit](../lit.md) | [Project cards](slices/projects.md), [identity cards](slices/identity-brokering.md), [Space cards](slices/spaces-and-terminals.md); [host projects](coverage/maps/host-projects.md), [host terminals](coverage/maps/host-terminals.md), [broker policy](coverage/maps/identity-broker-protocol-and-policy.md), [broker state](coverage/maps/identity-broker-state-and-entrypoints.md), [providers](coverage/maps/identity-providers.md), [browser Spaces](coverage/maps/browser-spaces.md). Follow every assigned inventory/map row, including Go API/store, SQL, tests and system definitions. |
| B: F01-F12, G01-G09 | [Factory interfaces](../../architecture/factory-interfaces.md), [operator factory reference](../../reference/factory.md), [Forgejo](../../reference/forgejo.md), [API](../../reference/api.md), [credentials](../../reference/credentials.md), [testing](../testing.md) | [Factory cards](slices/factory-coordination.md), [Forgejo cards](slices/forgejo-integration.md); [control admission/dispatch](coverage/maps/factory-control-admission-and-dispatch.md), [publication/review](coverage/maps/factory-control-publication-and-review.md), [native fixtures](coverage/maps/factory-control-native-fixtures.md), [host factory](coverage/maps/host-factory-runs.md), [Go Forgejo](coverage/maps/backend-forgejo.md), [Forgejo tests](coverage/maps/tests-forgejo.md). Include native surfaces, shared records, SQL and cross-slice provider/Project boundaries. |
| C: N01-N07, O01-O07, D01-D11, H01-H06 | [Networking](../../architecture/networking.md), [operator setup](../../guides/operator-setup.md), [configuration](../../reference/configuration.md), [Cockpit](../cockpit.md), [release architecture](../../architecture/release.md), [release workflow](../release.md), [installation](../../guides/installation.md), [native support](../native-support.md), [testing](../testing.md), [Go guidance](../go.md) | [Network cards](slices/networking.md), [operator cards](slices/operator-administration.md), [delivery cards](slices/release-and-installation.md), [shared cards](slices/shared-supporting-slices.md); [Tailnet control](coverage/maps/host-tailnet-control.md), [Go store](coverage/maps/backend-store.md), [appliance definitions](coverage/maps/appliance-definitions.md), [tooling](coverage/maps/developer-tools.md), [release tools implementation](coverage/maps/soda-release-tools-implementation.md), [release tools verification](coverage/maps/soda-release-tools-verification.md). Use the full map index for all release/image/install, shared primitive, presentation/asset and verification duties. |

The subsequent audit command permits source inspection and slice/plan records.
Each primary writes only its assigned review records; the coordinator alone
writes shared plan documents and every affected primary joins boundary exchanges.
Tests/builds, Git mutations, service/fixture/provider operations and source fixes
are not authorized by this audit. Never infer implementation permission from a
review assignment or specified correction.

Required acknowledgment: runtime identity/model selection, exact slice IDs and
count, baseline result, packet/contract/coverage pointers received, challenger
and coordinator contacts, four completion dimensions and pending-only scope.
Report missing inputs to the coordinator; do not fill unknown product decisions.

On 2026-10-05, A/B/C each acknowledged its exact assignment, read the shared
packet and assigned cards, located contract/coverage navigation, verified the
source/tree plus all 11 guidance hashes, and sent internal receipts to both
other reviewers. None reported a missing input or drift. The coordinator
records receipt metadata here and reconciles its new byte identity in the
baseline. Briefing is complete. Current review scope and limits belong in the
actual records linked from the [review index](reviews/README.md).

## One primary per slice

The table below is the canonical assignment map. Each slice has exactly one
primary and one different worker responsible for challenging consequential
conclusions. Record paths are prospective; no records have been created by this
assignment setup. Actual review state and evidence belong in those records,
not a duplicate progress ledger here.

| Slice | Primary | Independent challenger | Review record when started |
| --- | --- | --- | --- |
| [P01 Repository association and creation](slices/projects.md#p01-repository-association-and-creation) | A | B | `reviews/P01.md` |
| [P02 Profile and runtime readiness](slices/projects.md#p02-profile-and-runtime-readiness) | A | B | `reviews/P02.md` |
| [P03 Human membership and accounts](slices/projects.md#p03-human-membership-and-accounts) | A | B | `reviews/P03.md` |
| [P04 Development SSH access](slices/projects.md#p04-development-ssh-access) | A | B | `reviews/P04.md` |
| [P05 Project Start/Stop](slices/projects.md#p05-project-startstop) | A | B | `reviews/P05.md` |
| [P06 Factory role accounts](slices/projects.md#p06-factory-role-accounts) | A | B | `reviews/P06.md` |
| [P07 Checkout allocation and preparation](slices/projects.md#p07-checkout-allocation-and-preparation) | A | B | `reviews/P07.md` |
| [P08 Preparation requirements acceptance](slices/projects.md#p08-preparation-requirements-acceptance) | A | B | `reviews/P08.md` |
| [P09 Privileged preparation approval](slices/projects.md#p09-privileged-preparation-approval) | A | B | `reviews/P09.md` |
| [P10 Shared tools and packages](slices/projects.md#p10-shared-tools-and-packages) | A | B | `reviews/P10.md` |
| [P11 Nested services and volumes](slices/projects.md#p11-nested-services-and-volumes) | A | B | `reviews/P11.md` |
| [P12 Maintenance holds](slices/projects.md#p12-maintenance-holds) | A | B | `reviews/P12.md` |
| [I01 Enrollment and owner consent](slices/identity-brokering.md#i01-enrollment-and-owner-consent) | A | B | `reviews/I01.md` |
| [I02 Encrypted credential custody](slices/identity-brokering.md#i02-encrypted-credential-custody) | A | B | `reviews/I02.md` |
| [I03 Delegation and connection availability](slices/identity-brokering.md#i03-delegation-and-connection-availability) | A | B | `reviews/I03.md` |
| [I04 Execution admission and lease fencing](slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) | A | B | `reviews/I04.md` |
| [I05 Native binding and private delivery](slices/identity-brokering.md#i05-native-binding-and-private-delivery) | A | B | `reviews/I05.md` |
| [I06 Completion, revocation and reconciliation](slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) | A | B | `reviews/I06.md` |
| [I07 Codex adapter](slices/identity-brokering.md#i07-codex-adapter) | A | B | `reviews/I07.md` |
| [I08 Muse adapter](slices/identity-brokering.md#i08-muse-adapter) | A | B | `reviews/I08.md` |
| [I09 Provider execution integration](slices/identity-brokering.md#i09-provider-execution-integration) | A | B | `reviews/I09.md` |
| [I10 Identity audit history](slices/identity-brokering.md#i10-identity-audit-history) | A | B | `reviews/I10.md` |
| [F01 Repository factory policy](slices/factory-coordination.md#f01-repository-factory-policy) | B | C | `reviews/F01.md` |
| [F02 Operator execution grants](slices/factory-coordination.md#f02-operator-execution-grants) | B | C | `reviews/F02.md` |
| [F03 Capacity, reservations and accounting](slices/factory-coordination.md#f03-capacity-reservations-and-accounting) | B | C | `reviews/F03.md` |
| [F04 Connection sponsorship](slices/factory-coordination.md#f04-connection-sponsorship) | B | C | `reviews/F04.md` |
| [F05 Accepted requirements and invalidation](slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) | B | C | `reviews/F05.md` |
| [F06 Issue intake and readiness](slices/factory-coordination.md#f06-issue-intake-and-readiness) | B | C | `reviews/F06.md` |
| [F07 Assignment and dispatch](slices/factory-coordination.md#f07-assignment-and-dispatch) | B | C | `reviews/F07.md` |
| [F08 Run lifecycle and intervention](slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | B | C | `reviews/F08.md` |
| [F09 Publication progression](slices/factory-coordination.md#f09-publication-progression) | B | C | `reviews/F09.md` |
| [F10 Independent review and correction](slices/factory-coordination.md#f10-independent-review-and-correction) | B | C | `reviews/F10.md` |
| [F11 Candidate verification assessment](slices/factory-coordination.md#f11-candidate-verification-assessment) | B | C | `reviews/F11.md` |
| [F12 Merge eligibility and completion](slices/factory-coordination.md#f12-merge-eligibility-and-completion) | B | C | `reviews/F12.md` |
| [S01 Authorized inventory](slices/spaces-and-terminals.md#s01-authorized-inventory) | A | B | `reviews/S01.md` |
| [S02 Workspace lifetime and navigation](slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) | A | B | `reviews/S02.md` |
| [S03 Views, layout and restoration](slices/spaces-and-terminals.md#s03-views-layout-and-restoration) | A | B | `reviews/S03.md` |
| [S04 Human terminal lifecycle](slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) | A | B | `reviews/S04.md` |
| [S05 Interactive attachment](slices/spaces-and-terminals.md#s05-interactive-attachment) | A | B | `reviews/S05.md` |
| [S06 Factory activity presentation](slices/spaces-and-terminals.md#s06-factory-activity-presentation) | A | B | `reviews/S06.md` |
| [G01 Browser authority and contributions](slices/forgejo-integration.md#g01-browser-authority-and-contributions) | B | C | `reviews/G01.md` |
| [G02 Background service admission](slices/forgejo-integration.md#g02-background-service-admission) | B | C | `reviews/G02.md` |
| [G03 Authoritative native reads](slices/forgejo-integration.md#g03-authoritative-native-reads) | B | C | `reviews/G03.md` |
| [G04 Candidate publication and PR creation](slices/forgejo-integration.md#g04-candidate-publication-and-pr-creation) | B | C | `reviews/G04.md` |
| [G05 Native review submission](slices/forgejo-integration.md#g05-native-review-submission) | B | C | `reviews/G05.md` |
| [G06 Native CI observation](slices/forgejo-integration.md#g06-native-ci-observation) | B | C | `reviews/G06.md` |
| [G07 Conditional native merge](slices/forgejo-integration.md#g07-conditional-native-merge) | B | C | `reviews/G07.md` |
| [G08 Native Forgejo presentation](slices/forgejo-integration.md#g08-native-forgejo-presentation) | B | C | `reviews/G08.md` |
| [G09 Local profile preferences](slices/forgejo-integration.md#g09-local-profile-preferences) | B | C | `reviews/G09.md` |
| [N01 Private origins, TLS and activation](slices/networking.md#n01-private-origins-tls-and-activation) | C | A | `reviews/N01.md` |
| [N02 Project LAN access](slices/networking.md#n02-project-lan-access) | C | A | `reviews/N02.md` |
| [N03 Host Tailnet control](slices/networking.md#n03-host-tailnet-control) | C | A | `reviews/N03.md` |
| [N04 Project enrollment policy](slices/networking.md#n04-project-enrollment-policy) | C | A | `reviews/N04.md` |
| [N05 Project Tailnet selection](slices/networking.md#n05-project-tailnet-selection) | C | A | `reviews/N05.md` |
| [N06 Project companion lifecycle](slices/networking.md#n06-project-companion-lifecycle) | C | A | `reviews/N06.md` |
| [N07 Git endpoint advertisement](slices/networking.md#n07-git-endpoint-advertisement) | C | A | `reviews/N07.md` |
| [O01 First-boot database provisioning](slices/operator-administration.md#o01-first-boot-database-provisioning) | C | A | `reviews/O01.md` |
| [O02 Operator identity bootstrap](slices/operator-administration.md#o02-operator-identity-bootstrap) | C | A | `reviews/O02.md` |
| [O03 Existing-install credential maintenance](slices/operator-administration.md#o03-existing-install-credential-maintenance) | C | A | `reviews/O03.md` |
| [O04 Native host administration and updates](slices/operator-administration.md#o04-native-host-administration-and-updates) | C | A | `reviews/O04.md` |
| [O05 Database backup and retention](slices/operator-administration.md#o05-database-backup-and-retention) | C | A | `reviews/O05.md` |
| [O06 Database restore](slices/operator-administration.md#o06-database-restore) | C | A | `reviews/O06.md` |
| [O07 Operator SSH enrollment](slices/operator-administration.md#o07-operator-ssh-enrollment) | C | A | `reviews/O07.md` |
| [D01 Builder admission and controllers](slices/release-and-installation.md#d01-builder-admission-and-controllers) | C | A | `reviews/D01.md` |
| [D02 Pinned input acquisition](slices/release-and-installation.md#d02-pinned-input-acquisition) | C | A | `reviews/D02.md` |
| [D03 Candidate production](slices/release-and-installation.md#d03-candidate-production) | C | A | `reviews/D03.md` |
| [D04 Authenticated installation media](slices/release-and-installation.md#d04-authenticated-installation-media) | C | A | `reviews/D04.md` |
| [D05 Artifact verification](slices/release-and-installation.md#d05-artifact-verification) | C | A | `reviews/D05.md` |
| [D06 Installed qualification](slices/release-and-installation.md#d06-installed-qualification) | C | A | `reviews/D06.md` |
| [D07 Release admission and preparation](slices/release-and-installation.md#d07-release-admission-and-preparation) | C | A | `reviews/D07.md` |
| [D08 Signing custody](slices/release-and-installation.md#d08-signing-custody) | C | A | `reviews/D08.md` |
| [D09 Publication and effect observation](slices/release-and-installation.md#d09-publication-and-effect-observation) | C | A | `reviews/D09.md` |
| [D10 Verified distribution consumption](slices/release-and-installation.md#d10-verified-distribution-consumption) | C | A | `reviews/D10.md` |
| [D11 Host installation and payload application](slices/release-and-installation.md#d11-host-installation-and-payload-application) | C | A | `reviews/D11.md` |
| [H01 Private IPC and service lifetime](slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) | C | A | `reviews/H01.md` |
| [H02 Storage mechanics](slices/shared-supporting-slices.md#h02-storage-mechanics) | C | A | `reviews/H02.md` |
| [H03 Encoding and parsing](slices/shared-supporting-slices.md#h03-encoding-and-parsing) | C | A | `reviews/H03.md` |
| [H04 Configuration and filesystem primitives](slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) | C | A | `reviews/H04.md` |
| [H05 Branding, avatars and attribution](slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) | C | A | `reviews/H05.md` |
| [H06 Developer tooling and verification infrastructure](slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | C | A | `reviews/H06.md` |

An idle worker or a worker challenging another role's conclusion remains primary
for its assigned slices. The coordinator can rebalance work when actual review
size warrants it, but must name the replacement primary and a different
challenger in this table, settle any in-progress record handoff and update
related queue/record references before the new owner writes. No implicit shared
primary or duplicate completion claim is allowed. New, split or retired slice
IDs require matching catalog, assignment and responsibility-map updates.

## Writing responsibilities

| Document/work | Writer during an authorized audit | Contributions from other reviewers |
| --- | --- | --- |
| `reviews/<slice-id>.md` | The slice's named primary | Send evidence, objections and requested changes to the primary; do not concurrently rewrite its record. |
| Assignment map, runtime bindings and shared boundary queue | Coordinator | Report scheduling needs, missing owners and boundary requests. |
| Shared catalog/indexes, coverage inventories/maps, decomposition, package/port proposals, integration and proposed tree | Coordinator | Submit exact proposed edits with source units, target paths, finding/decision IDs and evidence limits. |
| Source, manifests, runtime configuration, Git or external state | No authorization from these assignments | Review findings and reference proposals only; any later action follows the user's actual authorization rules. |

Every writer inspects current content and preserves unrelated work. The
coordinator consolidates supported conclusions without erasing dissent or
turning a recommendation into a user decision. Primaries retain responsibility
for the accuracy and completeness of their own records.

## Shared boundary exchanges

Share a material observation as soon as it affects another slice's requirement,
owned state, caller chain, language allocation, target path, lifecycle or evidence.
Send it to the other primary and the coordinator through internal agent messages.
This assignment does not authorize messages to external people or services.

An exchange identifies the exact source/contract revision, source intervals and
symbols, sending and affected slices/workers, observed behavior or question,
independent requirement authority, evidence class and limits, observed versus
inferred consequence, and related finding/action IDs. A reply addresses the
other owner's actual responsibility with evidence, or states the exact missing
fact or decision. Message receipt or agent agreement is not proof.

Each cross-slice finding has one responsible slice and stable finding ID, as
specified in the review format. Other slices link it and assess their own units;
reading a caller does not complete review of the callee. If responsibility itself
is disputed, record that boundary question without inventing domain ownership.
Every affected slice's primary participates, including all affected workers
for multi-party boundaries, even when a separate challenger is assigned.
Boundaries inside one worker's group still need a different reviewer for their
consequential conclusions.

Prioritize these exchanges from the existing catalog; they do not exhaust the
boundaries that source review may discover:

| Participants | Existing slice boundaries to trace together |
| --- | --- |
| A and B | Factory sponsorship/admission/dispatch/settlement F03/F04/F07/F08 with identity I03–I06/I09 and Project lifecycle/preparation P05–P09/P12; Spaces factory activity S06 with F08. |
| A and C | Project access/lifecycle P01–P05/P11 with networking N02/N05/N06; identity custody/closure I02/I06 with operator maintenance O03 and storage H02; delivery of Project account/role helper bytes. |
| B and C | Forgejo browser/background authority G01/G02 with private origins, host/network admission N01/N03–N05/H01; native presentation/preferences G08/G09 with storage/parsing/assets H02/H03/H05 and candidate build D03. |
| All three | H06/D06 verification drivers with domain-owned behavior; D03 shipping/build wiring for every proposed port or executable cutover. |

The coordinator owns the [shared-boundary queue](reviews/README.md#shared-boundary-queue).
Store detailed evidence and replies
in the owning review records and link them here. Keep unresolved disagreements
visible. An unanswered boundary keeps its dependent conclusion/action pending;
workers continue independent established review work instead of waiting for all
other slices or declaring the boundary complete.

The queue links canonical findings and unresolved workflows; it does not
duplicate primary records or turn acknowledgment into proof.

## Independent challenges

Apply the [completion criteria](review-format.md#completion-criteria): every
concrete finding requires an independent challenge, as do the consequential
conclusions below. The coordinator checks whole-workflow participation and
coverage; worker completion alone does not complete the audit.

The challenger map applies to consequential conclusions: authority/trust,
canonical state or shared definitions, lifecycle/termination/credential custody,
material correctness defects, language ownership, consolidation/splitting,
predecessor retirement, process topology and build/install cutover. It applies
to consequential claims that retain the current design as well as proposed
changes. Boundary-owner participation supplements the challenge; it does not
replace review by a worker other than the primary.

The primary sends the claim or finding, controlling requirement/decision,
exact source and caller chain, evidence/test subjects and limits, proposed
allocation/correction, and every affected slice to its assigned challenger.
The challenger checks underlying contracts and source, competing explanations,
test assumptions and the supported scope of the conclusion. Agreement or a
message saying reviewed cannot complete the challenge.

Record each challenge in the primary record with:

- Challenger role, runtime identity and actual model.
- Source/contract revision and exact claims, questions and responsibilities checked.
- Concrete objection with evidence, or none found within the checked scope.
- Evidence limits and remaining unexamined responsibilities.
- The primary's response and disposition of every objection.
- Accepted correction, retained disagreement or exact owner decision needed.

The assigned primary alone incorporates the response into its record. A
challenge is complete for its named questions when this evidence and the
responses are recorded; an unresolved objection can remain open. Challenge
completion does not mean the disputed claim is resolved, the whole slice was
independently verified, or implementation/installed behavior passed.
Consequential unchallenged or disputed conclusions cannot be presented as
settled when the coordinator updates the plan; record the pending challenge or
decision alongside the affected action.

## Coordinator reconciliation and upkeep

Report completion only under the [four review dimensions](review-format.md#completion-criteria).
Keep unresolved questions linked to the exact dependent implementation
instructions and mark those instructions decision pending. Distinguish completed
inspection/challenge from resolved findings, specified targets and actual
behavioral proof; do not declare the whole catalog complete while a required
dimension remains unfinished.

The coordinator checks complete coverage and the one-primary map, routes
boundary/challenge tasks, reconciles duplicate findings and incompatible target
allocations, and writes the shared plan updates. Reconcile factual disagreements
against actual evidence and established requirements. Preserve the dissent and
why it was accepted, rejected or left unresolved; a majority vote establishes
neither correctness nor product authority.

An existing user decision remains authoritative while an agent challenges its
technical fit. Record the challenge and any recommended change without silently
reopening or overriding that decision.

Bring genuinely conflicting requirements, unknown product/trust behavior and
changes to established owner decisions to the user as concrete questions with
sources, alternatives and consequences. The coordinator cannot decide those
unknowns merely to finish the audit. Independent known work continues; only the
affected model verdict and correction readiness remain pending.

After every merge, apply [maintenance](maintenance.md) and
[review freshness](review-format.md#keeping-a-review-current). Notify affected
primaries and challengers, recheck dependent workflows and source/contract
conclusions, and update ownership/records/maps together when a boundary changes.
Preserve earlier evidence scope; assigning a replacement reviewer or advancing
HEAD does not transfer or refresh completion claims. The assignment document
and queue stay current rather than become a merge diary.
