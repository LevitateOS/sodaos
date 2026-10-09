# Projects

[Slice catalog and evidence scope](README.md). The cards below span current
source owners; they do not create packages, services or an approved intended model.

## P01 Repository association and creation

- **Validity review:** [P01 record](../reviews/P01.md) — actual reviewed scope, findings and pending completion dimensions.

Bind one native repository to a reserved Project and coordinate explicit creation without treating reservation as provisioning.

- **Entrypoints:** POST /api/environments; apiCreateEnvironment; verifyRepositoryOwner; provisionAndSaveProject.
- **Owned data:** Project ID, repository association, name and creation owner; reservation identity.
- **Authority:** Current verified native repository owner; caller cannot supply another owner.
- **Dependencies:** [P02](#p02-profile-and-runtime-readiness); Native Forgejo repository callback; PostgreSQL; Existing Tailnet selection admission.
- **Source files:** [internal/web/api/environments_create.go:46](../../../../internal/web/api/environments_create.go#L46); [internal/store/schema.go:22](../../../../internal/store/schema.go#L22); [docs/product/projects.md:209 (historical line locator)](../../../product/projects.md).
- **Tests:** [tests/frontend/native-project-controls.test.ts:85](../../../../tests/frontend/native-project-controls.test.ts#L85) — Source-only browser fixture assertions: one explicit Create request, native generation, selected repository/profile; no Join or Start request.
- **Unclear boundaries:** Owns association/reservation fields. P02 owns profile and readiness fields; network policy remains a dependency. Native success precedes MarkReady; source does not establish installed creation proof.
- **Evidence status:** Representative source catalog at committed 26d420f2; assertions inspected only, no tests or native execution performed.

## P02 Profile and runtime readiness

- **Validity review:** [P02 record](../reviews/P02.md) — actual reviewed scope, findings and pending completion dimensions.

Resolve installed creation identity, inspect the exact retained container, and distinguish saved readiness from current native observations.

- **Entrypoints:** resolveNativeProfile; Host.ResolveProfile/Create/Inspect/ObserveOS; project-init boot.
- **Owned data:** Immutable creation profile; Project ready/IP observations; native container incarnation and OS observations.
- **Authority:** Privileged host inspects native Podman/systemd; API authorization controls disclosure.
- **Dependencies:** [P01](#p01-repository-association-and-creation); Podman; systemd; Pinned Project image.
- **Source files:** [internal/web/api/environments_create.go:91](../../../../internal/web/api/environments_create.go#L91); `rust/soda-host/src/project.rs:286 (historical source locator)`; `project-os/rootfs/usr/libexec/soda/project-init:11 (historical source locator)`; [docs/reference/project-os.md:103 (historical line locator)](../../../reference/project-os.md).
- **Tests:** [internal/store/project_profile_test.go:10](../../../../internal/store/project_profile_test.go#L10) — Source-only PostgreSQL fixture assertions: profile survives MarkReady and creation identity cannot be changed; [tests/build/project_runtime_test.go:91](../../../../tests/build/project_runtime_test.go#L91) — Structural source assertions: only network sysctl subtree rebound; failure emits degradation marker/warning without hard exit.
- **Unclear boundaries:** Readiness marker is not SSH, membership or application-health proof. HEAD permits degraded nested networking; review that with P11 rather than interpreting ready as full service readiness.
- **Evidence status:** Representative source catalog at committed 26d420f2; assertions inspected only, no tests or native execution performed.

## P03 Human membership and accounts

- **Validity review:** [P03 record](../reviews/P03.md) — actual reviewed scope, findings and pending completion dimensions.

Admit a human Join using current repository rights, preserve stable identity-to-login mapping, and persist membership after native account success.

- **Entrypoints:** POST /api/environments/{id}/join; persistEnvironmentJoin; project-account stdin helper.
- **Owned data:** Membership user/project/login mapping; native human UID/GID/home and association marker.
- **Authority:** Verified actor with current repository code-write; native helper provisions locked account under root.
- **Dependencies:** [P01](#p01-repository-association-and-creation); [P02](#p02-profile-and-runtime-readiness); [P04](#p04-development-ssh-access); Native Forgejo repository callback; Native passwd/group.
- **Source files:** [internal/web/api/environments_join.go:37](../../../../internal/web/api/environments_join.go#L37); `rust/soda-project-account/src/account.rs:422 (historical source locator)`; [docs/product/projects.md:253 (historical line locator)](../../../product/projects.md).
- **Tests:** [internal/web/api/environments_join_test.go:12](../../../../internal/web/api/environments_join_test.go#L12) — Source-only API assertions: invalid/reserved login rejected before provisioning; [tests/build/project_account_test.go:136](../../../../tests/build/project_account_test.go#L136) — Source-only compiled-helper test definitions with redirected roots/command doubles: locked account, home marker, shared link and empty keyfile.
- **Unclear boundaries:** Owns human accounts only; P06 owns factory roles. First provision may accept selected public keys, while later key application belongs to P04. Native failure must not become recorded membership.
- **Evidence status:** Representative source catalog at committed 26d420f2; assertions inspected only, no tests or native execution performed.

## P04 Development SSH access

- **Validity review:** [P04 record](../reviews/P04.md) — actual reviewed scope, findings and pending completion dimensions.

Manage explicit member key application and provide verified native SSH connection identity independently of browser terminal attachments.

- **Entrypoints:** GET/POST /api/environments/{id}/access-keys; GET connection; Host.AccessKeys/Connection.
- **Owned data:** Saved public keys/fingerprints; managed root-owned authorized_keys contents and revision; observed SSH endpoint/host keys.
- **Authority:** Actor controls own saved keys and installed member login; confirmation binds reviewed key set.
- **Dependencies:** [P03](#p03-human-membership-and-accounts); [P02](#p02-profile-and-runtime-readiness); OpenSSH; Native Forgejo public-key source.
- **Source files:** [internal/web/api/access_keys.go:58](../../../../internal/web/api/access_keys.go#L58); [internal/web/api/environments_api.go:198](../../../../internal/web/api/environments_api.go#L198); `rust/soda-host/src/account.rs (historical source locator)`; `rust/soda-host/src/project.rs:700 (historical source locator)`.
- **Tests:** [internal/web/lifecycle_access_keys_test.go:156](../../../../internal/web/lifecycle_access_keys_test.go#L156) — Source-only API fixture assertions: stale/caller-selected key sets refused; explicit last-key removal confirmation required.
- **Unclear boundaries:** Saved-key deletion does not itself revoke native access. Human SSH remains ordinary SSH, not a managed browser session. Account initialization overlaps P03; ongoing key content/revision ownership stays here.
- **Evidence status:** Representative source catalog at committed 26d420f2; assertions inspected only, no tests or native execution performed.

## P05 Project Start/Stop

- **Validity review:** [P05 record](../reviews/P05.md) — actual reviewed scope, findings and pending completion dimensions.

Coordinate explicit lifecycle of the same retained Project container with factory quiescence and native outcome verification.

- **Entrypoints:** POST /api/environments/{id}/lifecycle; apiLifecycleStop/Start; Host.Lifecycle.
- **Owned data:** Transient terminal stopping admission; observed lifecycle outcome; factory coordination receipts.
- **Authority:** Current Project administrator/operator authority; explicit shared-impact Stop confirmation.
- **Dependencies:** [P02](#p02-profile-and-runtime-readiness); [P12](#p12-maintenance-holds); Factory coordinator; Podman; systemd.
- **Source files:** [internal/web/api/lifecycle.go:175](../../../../internal/web/api/lifecycle.go#L175); `rust/soda-host/src/project.rs:737 (historical source locator)`; [docs/reference/project-os.md:96 (historical line locator)](../../../reference/project-os.md).
- **Tests:** [internal/web/lifecycle_access_keys_test.go:74](../../../../internal/web/lifecycle_access_keys_test.go#L74) — Source-only API fixture assertions: unauthorized/unconfirmed/unknown actions never reach native mutation.
- **Unclear boundaries:** Does not own terminal End or delete/recreate Project state. Stop-related admission and maintenance holds interact; Start's verified runtime outcome must not silently become permission to release P12.
- **Evidence status:** Representative source catalog at committed 26d420f2; assertions inspected only, no tests or native execution performed.

## P06 Factory role accounts

- **Validity review:** [P06 record](../reviews/P06.md) — actual reviewed scope, findings and pending completion dimensions.

Provision and validate fixed nonhuman native role identities and their protected home, checkout-root and credential-root layout.

- **Entrypoints:** project-factory-roles ensure; ensure_role; preparation container setup.
- **Owned data:** soda-coder/soda-reviewer native accounts; role homes/groups/shells; checkout and credential directory ownership.
- **Authority:** Privileged native helper; fixed role names; no human Join, interactive shell, wheel or development SSH authority.
- **Dependencies:** [P02](#p02-profile-and-runtime-readiness); Native passwd/group; [P07](#p07-checkout-allocation-and-preparation).
- **Source files:** `rust/soda-project-factory-roles/src/account.rs:245 (historical source locator)`; `rust/soda-project-factory-roles/src/ops_approve.rs:74 (historical source locator)`; [docs/product/projects.md:128 (historical line locator)](../../../product/projects.md).
- **Tests:** [tests/build/project_factory_roles_test.go:230](../../../../tests/build/project_factory_roles_test.go#L230) — Source-only compiled-helper tests with redirected factory root: two fixed roles, protected layout and idempotence; interactive/grouped account refusal.
- **Unclear boundaries:** Owns account/layout identity, not credentials inside those directories or lease admission. P07 owns individual checkout allocation; I05 owns authorized private credential delivery. Human account code remains P03.
- **Evidence status:** Representative source catalog at committed 26d420f2; assertions inspected only, no tests or native execution performed.

## P07 Checkout allocation and preparation

Execute approved exact-source setup in an assigned role checkout and retain preparation phase, readiness and process evidence.

- **Entrypoints:** Host.Prepare/PrepareCandidate/InspectPreparation/StopPreparation; project-factory-roles approve/record/start/inspect/stop.
- **Owned data:** Preparation identity; exact source/setup digests and decision references; protected snapshot; role checkout; supervisor and observed readiness.
- **Authority:** Requires P08 acceptance and P09 approval; native execution uses fixed P06 role; hold closes admission.
- **Dependencies:** [P06](#p06-factory-role-accounts); [P08](#p08-preparation-requirements-acceptance); [P09](#p09-privileged-preparation-approval); [P10](#p10-shared-tools-and-packages); [P12](#p12-maintenance-holds); Git bundle verification; Factory coordinator.
- **Source files:** `rust/soda-host/src/prepare.rs:444 (historical source locator)`; `rust/soda-project-factory-roles/src/ops_record.rs:167 (historical source locator)`; [internal/store/preparation.go:95](../../../../internal/store/preparation.go#L95).
- **Tests:** [internal/store/preparation_test.go:84](../../../../internal/store/preparation_test.go#L84) — Source-only PostgreSQL assertions: exact-input admission idempotent; changed setup cannot reuse identity; immutable decision references; [tests/build/project_factory_roles_test.go:347](../../../../tests/build/project_factory_roles_test.go#L347) — Source-only helper test definitions: running supervisor, repeated start and confirmed targeted stop in redirected fixture.
- **Unclear boundaries:** One owner for checkout/preparation execution state; factory consumes its evidence. Approved snapshot materialization here does not confer administrator decision authority. Native candidate preparation and general preparation share machinery.
- **Evidence status:** Representative source catalog at committed 26d420f2; assertions inspected only, no tests or native execution performed.

- **Validity review:** [P07 audit record](../reviews/P07.md).

## P08 Preparation requirements acceptance

- **Validity review:** [P08 record](../reviews/P08.md) — actual reviewed scope, findings and pending completion dimensions.

Record the code-write maintainer's acceptance of exact repository/setup inputs and the ordered requirement decision chain.

- **Entrypoints:** POST preparation-acceptances; apiPreparationAcceptances; Coordinator.AdmitRequirement.
- **Owned data:** Requirement decision ID/predecessor/head; source commit; setup and inputs digests; verified approver.
- **Authority:** Current native repository code-write maintainer; authority taken from session rather than request approver.
- **Dependencies:** [P01](#p01-repository-association-and-creation); Native Forgejo repository callback; Factory command ledger; PostgreSQL.
- **Source files:** [internal/web/api/preparation_decisions.go:22](../../../../internal/web/api/preparation_decisions.go#L22); [internal/factory/control/grants.go:203](../../../../internal/factory/control/grants.go#L203); [internal/store/project_grants.go:31](../../../../internal/store/project_grants.go#L31).
- **Tests:** [internal/store/project_grants_test.go:45](../../../../internal/store/project_grants_test.go#L45) — Source-only PostgreSQL fixture assertions: exact replay accepted, changed identity refused, predecessor chain enforced.
- **Unclear boundaries:** Acceptance owns input consent only. It does not approve privileged effects, run setup or assert observed readiness. P09 references its current decision; P07 consumes both. Shared command ledger does not merge authorities.
- **Evidence status:** Representative source catalog at committed 26d420f2; assertions inspected only, no tests or native execution performed.

## P09 Privileged preparation approval

- **Validity review:** [P09 record](../reviews/P09.md) — actual reviewed scope, findings and pending completion dimensions.

Record the native Project administrator's reviewed-effect approval against the current accepted requirement and required readiness.

- **Entrypoints:** POST preparation-actions action approve; apiPreparationActionApprove; Coordinator.AdmitApproval.
- **Owned data:** Approval decision chain/head; referenced requirement; effects/readiness digests and verified administrator.
- **Authority:** Current Project administrator distinct from repository maintainer authority; exact requirement binding.
- **Dependencies:** [P08](#p08-preparation-requirements-acceptance); [P12](#p12-maintenance-holds); Factory command ledger; PostgreSQL.
- **Source files:** [internal/web/api/preparation_decisions.go:149](../../../../internal/web/api/preparation_decisions.go#L149); [internal/factory/control/grants.go:221](../../../../internal/factory/control/grants.go#L221); [internal/store/project_grants.go:65](../../../../internal/store/project_grants.go#L65).
- **Tests:** [internal/factory/control/grants_test.go:164](../../../../internal/factory/control/grants_test.go#L164) — Source-only coordinator fixture assertions: approval of superseded requirement refused; current requirement accepted.
- **Unclear boundaries:** Owns privileged-effect decision records, not execution or filesystem snapshot state. P07 owns native materialization. Admission may interact with hold state; approval must not be described as installed proof of successful setup.
- **Evidence status:** Representative source catalog at committed 26d420f2; assertions inspected only, no tests or native execution performed.

## P10 Shared tools and packages

- **Validity review:** [P10 record](../reviews/P10.md) — actual reviewed scope, findings and pending completion dimensions.

Supply the native development foundation and persistent shared tool installations while repository-specific runtimes remain explicit mise inputs.

- **Entrypoints:** Project image build recipe; project-init shared directories; shell mise profile; approved preparation tool checks.
- **Owned data:** Installed RPM packages; /opt/mise and /etc/mise; /srv/project/shared ownership and persistent contents.
- **Authority:** Image recipe fixes shipped foundation; native Project administration controls shared installations; repository trust stays explicit.
- **Dependencies:** [P02](#p02-profile-and-runtime-readiness); [P03](#p03-human-membership-and-accounts); [P07](#p07-checkout-allocation-and-preparation); RPM/dnf; mise.
- **Source files:** `project-os/Containerfile:10 (historical source locator)`; `project-os/rootfs/usr/libexec/soda/project-init:24 (historical source locator)`; `project-os/rootfs/etc/profile.d/soda-mise.sh (historical source locator)`; [docs/reference/project-os.md:85 (historical line locator)](../../../reference/project-os.md).
- **Tests:** [tests/build/project_foundation_test.go:12](../../../../tests/build/project_foundation_test.go#L12) — Structural source assertions: required compiler/header/diagnostic packages present; signature checks retained; no blanket upgrade; [tests/build/project_runtime_test.go:105](../../../../tests/build/project_runtime_test.go#L105) — Structural probe-source assertions: shared-tools isolation uses actual second-Project address rather than static guess.
- **Unclear boundaries:** Owns installed/shared tool state, not preparation approval or repository checkout. Probe-source checks are not installed package/isolation proof. project-init spans tools, accounts and readiness and should be reviewed by responsibility.
- **Evidence status:** Representative source catalog at committed 26d420f2; assertions inspected only, no tests or native execution performed.

## P11 Nested services and volumes

- **Validity review:** [P11 record](../reviews/P11.md) — actual reviewed scope, findings and pending completion dimensions.

Expose the Project's native local Podman engine and preserve nested workload storage, volumes and secrets across Project Stop/Start.

- **Entrypoints:** soda-podman.socket/service; native podman/Compose commands inside Project; project-init network preparation.
- **Owned data:** Nested images/layers/volumes/secrets; local engine socket; runtime network-sysctl degradation marker.
- **Authority:** Project administrator access through root:wheel socket; appliance kernel/outer container remains separate authority.
- **Dependencies:** [P02](#p02-profile-and-runtime-readiness); [P05](#p05-project-startstop); Podman; netavark; systemd; Native Compose.
- **Source files:** `project-os/rootfs/etc/systemd/system/soda-podman.socket (historical source locator)`; `project-os/rootfs/etc/systemd/system/soda-podman.service (historical source locator)`; `project-os/rootfs/etc/containers/storage.conf (historical source locator)`; [docs/reference/project-os.md:93 (historical line locator)](../../../reference/project-os.md).
- **Tests:** [tests/build/project_runtime_test.go:48](../../../../tests/build/project_runtime_test.go#L48) — Structural unit assertions: local engine, activation FD and CONTAINER_HOST exclusion; [tests/build/project_runtime_test.go:67](../../../../tests/build/project_runtime_test.go#L67) — Structural assertions: root:wheel socket 0660 and managed runtime directory.
- **Unclear boundaries:** Native workloads own their application databases; Soda does not add a parallel service-state registry. HEAD's degraded networking marker is source-visible; consumer enforcement and installed volume/network behavior were not established here.
- **Evidence status:** Representative source catalog at committed 26d420f2; assertions inspected only, no tests or native execution performed.

## P12 Maintenance holds

- **Validity review:** [P12 record](../reviews/P12.md) — actual reviewed scope, findings and pending completion dimensions.

Close preparation admission through an explicit administrator hold and synchronize its native marker with durable revisioned state.

- **Entrypoints:** POST preparation hold/release; apiPreparationHold; Host.HoldPreparation; lifecycleStopHold.
- **Owned data:** Revisioned Project maintenance hold; native factory hold marker; observed synchronization state.
- **Authority:** Current Project administrator; release binds revision and native quiescence.
- **Dependencies:** [P05](#p05-project-startstop); [P07](#p07-checkout-allocation-and-preparation); Factory coordinator; PostgreSQL; Privileged factory-role helper.
- **Source files:** [internal/web/api/environments_preparation.go:121](../../../../internal/web/api/environments_preparation.go#L121); [internal/store/preparation.go:54](../../../../internal/store/preparation.go#L54); [internal/web/api/lifecycle.go:208](../../../../internal/web/api/lifecycle.go#L208).
- **Tests:** [internal/store/preparation_test.go:64](../../../../internal/store/preparation_test.go#L64) — Source-only PostgreSQL assertions: hold persisted with CAS; stale writer refused; [tests/build/project_factory_roles_test.go:396](../../../../tests/build/project_factory_roles_test.go#L396) — Source-only helper fixture assertions: hold blocks approval; release requires matching revision and quiescence.
- **Unclear boundaries:** Owns hold state, not container running state or individual run cancellation. Native marker/store reconciliation is a cross-boundary operation; P05 consumes it. Start/release wording should be reconciled before prescribing behavior.
- **Evidence status:** Representative source catalog at committed 26d420f2; assertions inspected only, no tests or native execution performed.
