# Forgejo integration

[Slice catalog and evidence scope](README.md). The cards below span current
source owners; they do not create packages, services or an approved intended model.

## First-principles refresh (2026-10-13)

Current source inspection starts at Soda
`fde19669440455a77e4c81c4f08ccc60a72276eb` and the clean paired Fountain/SDK
checkout `124907625d984755bb685168d0daddf8ef1cb36e`. The table follows current
consumers, transport and receipt/observation seams; it reuses earlier cards for
inventory and historical evidence only. It does not advance their full-body
review, native/browser execution or installed qualification. `Met` is scoped
to the inspected principle; `partial` records an implementation or verification
gap. Requirements remain in [Trust](../../../architecture/trust.md),
[Factory interfaces](../../../architecture/factory-interfaces.md) and
[the Forgejo contract](../../../reference/forgejo.md). GUIDANCE-07 retains native
conditional operations; GUIDANCE-08 preserves the intentional all-page Soda design.

| Slice | Purpose | First principle | Current assessment and evidence | Desired gap | Dependency-ordered implementation/check plan |
| --- | --- | --- | --- | --- | --- |
| G01 | Admit native browser contributions and product calls. | Fountain owns sessions/navigation; client-authored context grants nothing. | **Partial pending current native composition evidence.** [extension entrypoint](../../../../cmd/soda-extension/main.go), [live authority](../../../../internal/web/auth/extension.go) and [resource admission](../../../../internal/web/auth/extension_service.go) keep the host callback and session generation authoritative. | Verification: mount/update/retire and session loss against the current host contract, including ordinary navigation with a retained terminal. | Retain Go admission and TypeScript contribution owners. Use existing admission tests, then the established browser contract driver on the paired provider. No parallel login or browser shell. |
| G02 | Share one browser-independent native admission. | Transport identity and installation admission are separate from actor credentials; readers/writers share one bootstrap owner. | **Met within inspected source wiring.** [web construction](../../../../internal/web/server.go) shares the same background; [bootstrap](../../../../internal/forgejo/background_admission.go) serializes/rebinds admission and pins the native peer. | Verification: paired native revoke/rebind and aggregate cold-start cost for F06. No second admission required. | Retain Go `ServiceBackground`. Reuse existing scripted Unix bootstrap/rebind checks before targeted native admission verification. |
| G03 | Read complete permission-checked native evidence and retained original creation provenance. | A stable revision bracket, explicit completeness and exact locators precede semantic decisions; retained creation history is explicitly limited. | **Met within inspected validation boundary.** [snapshot](../../../../internal/forgejo/snapshot.go), [pull/check/ref validation](../../../../internal/forgejo/snapshot_pull.go) and [creation recovery transport](../../../../internal/forgejo/creation_recovery.go) bind locators and bounded pages; F05/F06 own acceptance/readiness semantics. | Verification: real native hidden/completeness/revision and retained creation-page behavior; test copies cannot establish upstream truth. | After G02, reuse bracket/transport/creation recovery tests, then current native snapshot driver. Retain SDK aliases and one validation boundary; do not duplicate native history. |
| G04 | Validate candidate bytes and conditionally publish/create PR. | Validate the real bundle, bind native intent and adopt only the exact attributable receipt. | **Partial pending native effect qualification.** [publisher](../../../../internal/forgejo/publish.go), [candidate validation](../../../../internal/forgejo/publish/candidate_validation.go) and [operation receipts](../../../../internal/forgejo/publish/operation_receipts.go) are current Go owners; historical `internal/host/publish` locators below are retired. | Verification: actual export/bundle/ref/PR effects and lost-response recovery without independent retry identities. | With G02/G03 available and F09 providing saved publication intent, reuse candidate/receipt checks; then validate native publication effects and F09 adoption. Keep validation unprivileged and native Git/Forgejo as effect owners. |
| G05 | Submit the exact independent native review. | Reviewer authority is distinct; immutable intent and attributable outcome survive lost replies. | **Partial pending native workflow evidence.** [review adapter](../../../../internal/forgejo/review.go) binds actor/head/base/event; current [review caller](../../../../internal/factory/control/review_cycle.go) persists and recovers review work. | Verification: actual native review receipt and withdrawal/recovery within F10. The historical missing-persistence finding is not assumed current. | With G02/G03 available and F10 providing a saved exact review operation, reuse adapter/recovery tests; then validate the native primitive and F10 adoption. Retain body-only review and dedicated credential custody. |
| G06 | Observe native CI checks without scheduling jobs. | Both assessment and merge must interpret the same native status ordering, visibility and ref evidence. | **Met for refreshed local mapping.** [checks](../../../../internal/forgejo/checks.go) now owns shared `mapObservedChecks`; [caller tests](../../../../internal/forgejo/checks_test.go) compare assessment/merge evidence and retain their distinct unavailable reasons. | **Implemented maintenance correction:** duplicated mapper removed. Verification gap: current native Actions/snapshot production. | Shared Go reduction first, focused real-caller mapping/error checks next; then reuse native check driver when its actual upstream fixtures exist. F11 retains verdicts; no local CI scheduler. |
| G07 | Submit conditional merge and confirm native completion. | Exact intent/receipt plus fresh result reachability are required; cancellation cannot undo committed effects. | **Partial pending native completion evidence.** [merge observation](../../../../internal/forgejo/merge_observation.go) consumes G06's mapper; [merge completion](../../../../internal/forgejo/merge_completion.go) verifies same-revision ancestry and original issue closure. | Mapper duplication corrected with G06. Verify real descendant-base ancestry, protections, operation completion and issue occurrence before claiming autonomous completion. | With G02/G03 available and F12 providing admitted immutable MergeWork, run local mapping/receipt/ancestry checks; then validate native merge effects and F12 completion. Retain Go adapter and upstream native engine. |
| G08 | Present the intentional Soda design on native pages. | Native forms, handlers, permissions and lifecycle remain authoritative under every override. | **Partial.** [repository switcher](../../../../assets/branding/forgejo/repository-switcher.ts) delegates native dropdown behavior; [settings template](../../../../frontend/forgejo/templates/repo/settings/options.tmpl) composes native form concerns; current [Forgejo contract](../../../reference/forgejo.md) still identifies composition/initialization qualification work. Older full-template review is not refreshed by this pass. | Verification: current host initialization/retirement, native form/CSRF handling and geometry across retained overrides. Override breadth is not a defect. | After G01, reuse existing component/source checks and the current native browser driver. Retain TypeScript/templates/assets; fix only demonstrated contract mismatches, not the design scope. |
| G09 | Read/change the current actor's Soda display preference. | A local display label cannot become native identity or authentication authority. | **Met within inspected source.** [preferences](../../../../internal/web/auth/session.go) bound input and target the admitted user; [extension service](../../../../internal/web/auth/extension_service.go) performs live before-write authority revalidation. Trust explicitly assigns Soda preferences locally. | Verification: authenticated successful save and stale-session refusal; no profile synchronization requirement established. | After G01, reuse preference-admission tests and one current native save/read journey. Retain Go auth/store and one `users.name` preference. |

The local G06/G07 patch removes one duplicated status reducer. Focused tests
exercise both actual callers, update-time/native-ID ordering, hidden evidence and
missing/overflowing-ID error paths. These are local source tests; no native
Fountain request, browser journey, installed state or external mutation is performed.
The focused tests passed with pinned Go 1.26.7; the assigned independent reviewer
accepted the extraction and caller/error parity within this local source scope.
The same challenge accepted the bounded current review/publication/completion
and snapshot dispositions; it did not review whole new judgment implementations
or qualify native/browser behavior.

The newer native [issue-graph writer](../../../../internal/forgejo/issue_graph.go)
consumes the same G02 admission and exact-operation contracts while F05 owns
accepted-intent/task adoption. It is an additional conditional collaboration
effect within this integration group, not candidate publication or another
native ledger. Its current owner check, submit/lookup/cancel and exact receipt
consumer were inspected; generation/recovery/native qualification remains
partial with F05, and the older G04 full-body review does not cover it.

Scoped upkeep at Soda `ac22514b` / Fountain `c72f5607b2` (2026-10-13): G01
consumes the canonical a9ba2ce0ba type-only mirror and supplied owner/routes/
request/stream/lifecycle contract. Fountain retains the native shell, feature
initialization/retirement and account/repository/session authority; G08's
legitimate branding and attribution remain. Current native composition and
installed duties stay open, with dirty provider patches outside the committed
pins. This upkeep does not advance the original validity or full-body reviews.

## G01 Browser authority and contributions

- **Validity review:** [G01 record](../reviews/G01.md) — actual reviewed scope, findings and pending completion dimensions.

Connect Soda pages, panels and product calls to verified native actor/session authority.

- **Entrypoints:** soda-extension contributionAuthorizer; Private extension listener and extensionProtected product routes; Native session/resource callbacks.
- **Owned data:** Ephemeral verified actor, contribution and session-generation context; native host owns login/session authority.
- **Authority:** Live Fountain admission/callback, contribution policy and product authorization; client-authored context is insufficient.
- **Dependencies:** Native Fountain browser host and extension SDK; Native Forgejo sessions/repository permissions.
- **Source files:** [cmd/soda-extension/main.go:72](../../../../cmd/soda-extension/main.go#L72); [internal/web/auth/extension.go](../../../../internal/web/auth/extension.go); [internal/web/auth/extension_service.go:171](../../../../internal/web/auth/extension_service.go#L171); [internal/web/api/extension_native.go:180](../../../../internal/web/api/extension_native.go#L180).
- **Tests:** [cmd/soda-extension/main_test.go:51](../../../../cmd/soda-extension/main_test.go#L51) — Source/unit: Spaces/workspace allowed, Tailnet operator-only, retired Runners refused; [internal/web/auth/extension_test.go:39](../../../../internal/web/auth/extension_test.go#L39) — Source/unit: missing live callback, forged context and duplicate admissions refuse.
- **Unclear boundaries:** Own contribution/authority admission; product slices own operation semantics/data. Generic browser lifecycle remains Fountain-owned without alternate login.
- **Evidence status:** Current entrypoint/checks; browser/session-loss journeys not run.

## G02 Background service admission

- **Validity review:** [G02 record](../reviews/G02.md) — actual reviewed scope, findings and pending completion dimensions.

Provide one shared browser-independent native admission for reads and conditional operation clients.

- **Entrypoints:** NewServiceBackground bootstrap; ReadNativeRevision/ReadSnapshot; SubmitOperation/GetOperation/CancelOperation and PublishPushEnv.
- **Owned data:** Ephemeral installation admission, verified peer and bootstrap/rebinding synchronization; native host owns durable installation/operation records.
- **Authority:** Configured native Unix peer identity and host-issued installation/runtime admission; actor credential remains separately restricted.
- **Dependencies:** Native Fountain background dispatcher and extension SDK.
- **Source files:** [internal/forgejo/background.go:28](../../../../internal/forgejo/background.go#L28); [internal/web/server.go:51](../../../../internal/web/server.go#L51).
- **Tests:** [internal/forgejo/background_test.go:205](../../../../internal/forgejo/background_test.go#L205) — Source/unit with local Unix transport/scripted native host: reads, writes, lookup and push environment share bootstrap; [internal/forgejo/background_test.go:252](../../../../internal/forgejo/background_test.go#L252) — Scripted Unix transport: revoked admission rebinds; not real Fountain.
- **Unclear boundaries:** Shared transport dependency of G03–G07, not another ledger. Separate bootstraps revoke each other; preserve the existing single admission owner.
- **Evidence status:** Current shared transport/wiring; real native bootstrap not exercised.

## G03 Authoritative native reads

- **Validity review:** [G03 record](../reviews/G03.md) — actual reviewed scope, findings and pending completion dimensions.

Return bounded permission-checked collaboration evidence tied to a complete stable revision bracket.

- **Entrypoints:** BackgroundSnapshotReader; BracketedRead; ServiceObserver and acceptance/readiness/dispatch adapters.
- **Owned data:** Transient NativeSnapshot families, locators, versions, digests, visibility and completeness; native history stays upstream.
- **Authority:** Installation admission, bound native actor credential and current read permission; reads authorize no mutation.
- **Dependencies:** [G02](#g02-background-service-admission); Native Fountain snapshot contract.
- **Source files:** [internal/forgejo/snapshot_transport.go:12](../../../../internal/forgejo/snapshot_transport.go#L12); [internal/forgejo/snapshot.go:400](../../../../internal/forgejo/snapshot.go#L400); [internal/forgejo/observe.go](../../../../internal/forgejo/observe.go); [internal/web/api/issue_acceptances.go:32](../../../../internal/web/api/issue_acceptances.go#L32).
- **Tests:** [internal/forgejo/snapshot_test.go:96](../../../../internal/forgejo/snapshot_test.go#L96) — Source/unit fake reader: busy/changed brackets, missing pages and digest mismatch refuse; hidden targets redacted; [internal/forgejo/snapshot_transport_test.go:83](../../../../internal/forgejo/snapshot_transport_test.go#L83) — Source/unit fake SDK: native fields and bracket revision map into evidence.
- **Unclear boundaries:** Own evidence integrity/mapping; F05 acceptance and F06 readiness remain semantic owners. Existing adapters request native families without competing records.
- **Evidence status:** Current snapshot/adapters; live native completeness/access not verified.

## G04 Candidate publication and PR creation

- **Validity review:** [G04 record](../reviews/G04.md) — actual reviewed scope, findings and pending completion dimensions.

Validate exported bytes and perform attributable conditional branch publication and native PR creation.

- **Entrypoints:** Publisher.ObservePublication/SubmitPublish/PushBranch/SubmitPRCreate; LookupOp/CancelOp and receipt adoption.
- **Owned data:** Temporary validation repository/bundle and transient outcomes; native ref/PR records stay upstream; F09 persists progression.
- **Authority:** Restricted native publication/creation actor credential, operation binding, repository identity and fresh refs.
- **Dependencies:** [G02](#g02-background-service-admission); [G03](#g03-authoritative-native-reads); Native Git; Native Fountain conditional-operation contract.
- **Source files:** [internal/forgejo/publish.go:18](../../../../internal/forgejo/publish.go#L18); `internal/host/publish/publish.go (historical source locator)`; `internal/host/publish/operation.go (historical source locator)`.
- **Tests:** [internal/forgejo/publish_test.go:158](../../../../internal/forgejo/publish_test.go#L158) — Source/unit: exact branch/PR receipts adopt; missing/foreign receipts refuse; [internal/forgejo/publish_test.go:214](../../../../internal/forgejo/publish_test.go#L214) — Source/unit: foreign operation/kind/actor/repository/installation refuse; [internal/factory/control/publication_native_test.go:365](../../../../internal/factory/control/publication_native_test.go#L365) — Optional native driver: exact branch/PR asserted; host export/run records seeded.
- **Unclear boundaries:** Own validation/native effects including correction publication; F09 owns eligibility/stages and durable local receipt references. No second coordinator.
- **Evidence status:** Current executor/native driver; no validation or publication executed.

## G05 Native review submission

- **Validity review:** [G05 record](../reviews/G05.md) — actual reviewed scope, findings and pending completion dimensions.

Submit one body-only review bound to the exact PR/candidate and adopt its attributable outcome.

- **Entrypoints:** Reviewer.ObserveReview/SubmitReview; LookupOp/CancelOp/AdoptReview.
- **Owned data:** Transient ReviewWork/native receipt; upstream owns review/comment/operation records; F10 stores local review progression.
- **Authority:** Distinct restricted reviewer credential and review-only binding; exact actor, PR, author, head/base and event.
- **Dependencies:** [G02](#g02-background-service-admission); [G03](#g03-authoritative-native-reads); Native Fountain conditional-review contract.
- **Source files:** [internal/forgejo/review.go:19](../../../../internal/forgejo/review.go#L19); [internal/forgejo/review.go:143](../../../../internal/forgejo/review.go#L143); [internal/factory/control/review_executor.go:9](../../../../internal/factory/control/review_executor.go#L9).
- **Tests:** [internal/forgejo/review_test.go:33](../../../../internal/forgejo/review_test.go#L33) — Source/unit: APPROVED/REQUEST_CHANGES adopt only exact work; changed identity refuses; [internal/forgejo/review_test.go:70](../../../../internal/forgejo/review_test.go#L70) — Source/unit: body-only payload, no inline comments; self-review refuses; [internal/factory/control/review_native_primitive_test.go:44](../../../../internal/factory/control/review_native_primitive_test.go#L44) — Optional native primitive: REQUEST_CHANGES commits; receipt inspected, not correction-loop proof.
- **Unclear boundaries:** Own review effect/receipt consistency; F10 owns independence, findings and corrections. Cancellation cannot erase a committed review.
- **Evidence status:** Concrete Reviewer conditionally wired with configured credential; tests not run.

## G06 Native CI observation

- **Validity review:** [G06 record](../reviews/G06.md) — actual reviewed scope, findings and pending completion dimensions.

Map native check evidence for the identified PR and exact candidate into bounded observations.

- **Entrypoints:** CheckAssessor.ObserveChecks; Checks-family snapshot over shared admission.
- **Owned data:** Transient ObservedChecks, latest context states, tips, visibility/completeness; Forgejo owns Actions records.
- **Authority:** Bound read actor/credential and repository visibility; no job-dispatch or local runner authority.
- **Dependencies:** [G02](#g02-background-service-admission); [G03](#g03-authoritative-native-reads); Native Forgejo Actions and separately managed capacity.
- **Source files:** [internal/forgejo/checks.go:17](../../../../internal/forgejo/checks.go#L17); [internal/forgejo/checks.go:60](../../../../internal/forgejo/checks.go#L60); [docs/product/scope.md:62 (historical line locator)](../../../product/scope.md); [docs/public/30-Use-Soda/50-ci-runners.md:3 (historical line locator)](../../../public/30-Use-Soda/50-ci-runners.md).
- **Tests:** [internal/forgejo/checks_test.go:46](../../../../internal/forgejo/checks_test.go#L46) — Source/unit: latest statuses/refs map; moved tips pass through, hidden flagged, wrong PR refuses; [internal/factory/control/checks_native_test.go:283](../../../../internal/factory/control/checks_native_test.go#L283) — Optional native observer/assessment driver; header records upstream-dependent skips, not current live availability.
- **Unclear boundaries:** F11 owns required-check verdict/persistence. Merger also maps check snapshots, leaving a G06/G07 evidence-mapping seam for review. Soda-provisioned local CI and Runner OS remain deferred.
- **Evidence status:** Current observer wired with merge credential; native Actions/snapshots not verified.

## G07 Conditional native merge

- **Validity review:** [G07 record](../reviews/G07.md) — actual reviewed scope, findings and pending completion dimensions.

Execute one exact conditional merge and confirm its attributable primary effect and native completion.

- **Entrypoints:** Merger.ObserveMerge/SubmitMerge; LookupOp/CancelOp/AdoptMerge/ObserveCompletion.
- **Owned data:** Transient work/receipt/completion; native host owns operation/PR/ref state; F12 owns local progression.
- **Authority:** Restricted merge actor, current native protections and immutable bound repository/PR/head/base intent.
- **Dependencies:** [G02](#g02-background-service-admission); [G03](#g03-authoritative-native-reads); Native Fountain conditional-merge contract; Native Forgejo merge engine.
- **Source files:** [internal/forgejo/merge.go:52](../../../../internal/forgejo/merge.go#L52); [internal/forgejo/merge.go:244](../../../../internal/forgejo/merge.go#L244); [internal/forgejo/merge.go:402](../../../../internal/forgejo/merge.go#L402).
- **Tests:** [internal/forgejo/merge_test.go:44](../../../../internal/forgejo/merge_test.go#L44) — Source/unit: exact receipt adopts; foreign identity, missing receipt or pending effect refuses; [internal/forgejo/merge_test.go:79](../../../../internal/forgejo/merge_test.go#L79) — Source/unit: exact fast-forward-only head/base intent; no-op refuses; [internal/factory/control/merge_native_test.go:347](../../../../internal/factory/control/merge_native_test.go#L347) — Optional native primitive: actual base tip equals bound head, not full factory qualification.
- **Unclear boundaries:** Own effects and receipt/tuple consistency; F12 eligibility. Private approval/check matching overlaps policy and G06 evidence mapping, an explicit joint review seam.
- **Evidence status:** Current Merger conditionally wired; native merge/completion not exercised.

## G08 Native Forgejo presentation

- **Validity review:** [G08 record](../reviews/G08.md) — actual reviewed scope, findings and pending completion dimensions.

Maintain customized native pages and browser enhancements while preserving Forgejo handlers, forms, sessions and stored collaboration state.

- **Entrypoints:** Image-owned template overrides; repository/settings/notification browser enhancements; native locale additions.
- **Owned data:** Readonly native page projections and labels; transient disclosure/search/theme preferences; Forgejo retains repositories, issues, reviews and account state.
- **Authority:** Native Forgejo session, CSRF and handlers govern form effects; presentation code has no independent Soda execution grant.
- **Dependencies:** [G01](#g01-browser-authority-and-contributions); [H05](shared-supporting-slices.md#h05-branding-avatars-and-attribution); [D03](release-and-installation.md#d03-candidate-production); Native Forgejo template/form/browser contracts.
- **Source files:** `appliance/forgejo/templates/repo/settings/options.tmpl:8 (historical source locator)`; [assets/branding/forgejo/repository-switcher.ts:98](../../../../assets/branding/forgejo/repository-switcher.ts#L98); `appliance/forgejo/i18n/README.md:3 (historical source locator)`.
- **Tests:** [tests/forgejo/component-boundaries.test.ts:9](../../../../tests/forgejo/component-boundaries.test.ts#L9) — Opt-in browser styling fixture with native CSS and authored markup: icon clearance and component layout across themes/breakpoints; not native form/session authorization proof.
- **Unclear boundaries:** Separate native customization from G01 extension admission and H05 shared artwork. Multiple native forms inside one template remain explicit concerns with upstream authority; custom browser preferences here are native-page enhancement state, distinct from G09 Soda profile persistence.
- **Evidence status:** Candidate added after coverage exposed maintained native presentation outside the original catalog; source inspected, tests not run.

## G09 Local profile preferences

Read and change the current actor’s Soda display-name preference without treating the local profile as native authentication authority.

- **Entrypoints:** GET/POST /api/me/preferences; session.preferences; Store.RenameProfile.
- **Owned data:** Soda users.name preference; transient bounded display_name request and response.
- **Authority:** Verified current native actor and live beforeWrite authority callback; mutation targets only that actor’s local row.
- **Dependencies:** [G01](#g01-browser-authority-and-contributions); [H02](shared-supporting-slices.md#h02-storage-mechanics); [H03](shared-supporting-slices.md#h03-encoding-and-parsing).
- **Source files:** [internal/web/auth/session.go:11](../../../../internal/web/auth/session.go#L11); [internal/web/auth/extension_service.go:82](../../../../internal/web/auth/extension_service.go#L82); [internal/store/store.go:195](../../../../internal/store/store.go#L195).
- **Tests:** [internal/web/auth/extension_test.go:56](../../../../internal/web/auth/extension_test.go#L56) — Admission regression: unverified profile mutation is refused and the stored name remains unchanged; no successful native preference journey inspected.
- **Unclear boundaries:** UpsertUser maintains native login identity while preserving the existing name; RenameProfile owns preference writes on the same row. Current product authority for this local preference versus native Forgejo profile settings must be established in the intended-model review.
- **Evidence status:** Current implemented state mapped; adding a candidate review slice does not approve retaining or expanding the preference feature.

- **Validity review:** [G09 audit record](../reviews/G09.md).
