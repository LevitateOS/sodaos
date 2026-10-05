# Factory coordination

[Slice catalog and evidence scope](README.md). The cards below span current
source owners; they do not create packages, services or an approved intended model.

## F01 Repository factory policy

Configure which repository work the factory may perform and under which roles, checks and merge policy.

- **Entrypoints:** PUT /api/repositories/{repositoryID}/factory/policy; GET /api/repositories/{repositoryID}/factory; Coordinator.ApplyPolicy.
- **Owned data:** Revisioned factory_policies; policy command receipts in shared factory_commands.
- **Authority:** Current native repository owner or administrator; operator identity has no repository-admin bypass.
- **Dependencies:** [G01](forgejo-integration.md#g01-browser-authority-and-contributions); [F08](#f08-run-lifecycle-and-intervention); Native Forgejo repository permissions.
- **Source files:** [internal/web/api/factory_settings.go:54](../../../../internal/web/api/factory_settings.go#L54); [internal/web/api/factory_settings.go:260](../../../../internal/web/api/factory_settings.go#L260); [internal/factory/control/grants.go:47](../../../../internal/factory/control/grants.go#L47); [internal/store/factory_grants.go:55](../../../../internal/store/factory_grants.go#L55).
- **Tests:** [internal/factory/grants_test.go:29](../../../../internal/factory/grants_test.go#L29) — Source/unit: missing checks, invalid roles, actor-binding kinds and unsupported merge method refuse; [internal/factory/control/grants_test.go:77](../../../../internal/factory/control/grants_test.go#L77) — PostgreSQL integration: pausing captures dispatch, stale revision refuses; fixture grants, not native authorization proof.
- **Unclear boundaries:** Own policy decisions/revisions. F08 settles withdrawal; independent execution, capacity, sponsorship and Project grants remain necessary. Settings reference enrolled actor bindings without enrolling credentials.
- **Evidence status:** Current handler, validation and persistence; complete configured factory behavior not assessed.
- **Validity review:** [F01 record](../reviews/F01.md) — actual scope, model, findings, challenge and target-allocation status.

## F02 Operator execution grants

Record the configured appliance operator's separate permission for factory execution in a repository.

- **Entrypoints:** PUT /api/repositories/{repositoryID}/factory/operator-grant; Coordinator.ApplyOperatorGrant.
- **Owned data:** Revisioned factory_operator_grants and operation-specific command receipts.
- **Authority:** Configured Soda operator through native extension admission; active repository grant with bounded concurrency.
- **Dependencies:** [G01](forgejo-integration.md#g01-browser-authority-and-contributions); [F08](#f08-run-lifecycle-and-intervention).
- **Source files:** [internal/web/api/factory_settings.go:76](../../../../internal/web/api/factory_settings.go#L76); [internal/web/api/factory_settings.go:349](../../../../internal/web/api/factory_settings.go#L349); [internal/factory/control/grants.go:74](../../../../internal/factory/control/grants.go#L74); [internal/store/factory_grants.go:87](../../../../internal/store/factory_grants.go#L87).
- **Tests:** [internal/factory/control/grants_test.go:77](../../../../internal/factory/control/grants_test.go#L77) — PostgreSQL integration: operator-grant command replays its revision; changed payload conflicts; [internal/factory/grants_test.go:57](../../../../internal/factory/grants_test.go#L57) — Source/unit: active unbounded operator grant refuses.
- **Unclear boundaries:** Execution permission remains distinct from repository policy, appliance capacity, connection sponsorship and P01 creation authority. Withdrawal closes dispatch; confirmed native termination comes from F08.
- **Evidence status:** Current API and persistence; inspected assertions were not executed.
- **Validity review:** [F02 record](../reviews/F02.md) — actual scope, model, findings, challenge and target-allocation status.

## F03 Capacity, reservations and accounting

Bound admitted execution and account for sponsored usage without restoring allowance through retries or requirement changes.

- **Entrypoints:** PUT /api/factory/capacity; DispatchPass admission; AccountSettledRun and reservation recovery.
- **Owned data:** factory_capacity, factory_reservations and factory_usage; occupancy from held reservations and active unassigned runs.
- **Authority:** Configured operator changes capacity; admission intersects repository, operator and sponsorship limits.
- **Dependencies:** [F01](#f01-repository-factory-policy); [F02](#f02-operator-execution-grants); [F04](#f04-connection-sponsorship); [F07](#f07-assignment-and-dispatch); [F08](#f08-run-lifecycle-and-intervention); [I05](identity-brokering.md#i05-native-binding-and-private-delivery).
- **Source files:** [internal/factory/control/grants.go:63](../../../../internal/factory/control/grants.go#L63); [internal/factory/control/dispatch.go](../../../../internal/factory/control/dispatch.go); [internal/store/factory_reservations.go:11](../../../../internal/store/factory_reservations.go#L11); [internal/store/factory_assignments.go:483](../../../../internal/store/factory_assignments.go#L483).
- **Tests:** [internal/factory/control/dispatch_test.go:347](../../../../internal/factory/control/dispatch_test.go#L347) — PostgreSQL integration with fake host/broker: exhausted usage persists after acceptance edits; [internal/factory/control/dispatch_test.go:1179](../../../../internal/factory/control/dispatch_test.go#L1179) — Concurrent PostgreSQL integration with fake execution: two passes preserve capacity and budget.
- **Unclear boundaries:** Own connection usage/reservation arithmetic; F07 owns assignments and F08 settlement. Calculations and atomic admission span dispatch/store code and need joint review, not another reservation service.
- **Evidence status:** Current accounting paths; no workload or concurrency check ran.
- **Validity review:** [F03 record](../reviews/F03.md) — actual scope, model, findings, challenge and target-allocation status.

## F04 Connection sponsorship

Permit a connection owner to sponsor repository factory roles within a separately recorded allowance.

- **Entrypoints:** PUT /api/repositories/{repositoryID}/factory/sponsorships/{connection}; Coordinator.ApplySponsorship.
- **Owned data:** Revisioned factory_sponsorships referencing broker grant, credential generation, roles and allowance.
- **Authority:** Current connection owner; current matching non-revoked broker grant and credential generation.
- **Dependencies:** [G01](forgejo-integration.md#g01-browser-authority-and-contributions); [I03](identity-brokering.md#i03-delegation-and-connection-availability); [F08](#f08-run-lifecycle-and-intervention).
- **Source files:** [internal/web/api/factory_settings.go:434](../../../../internal/web/api/factory_settings.go#L434); [internal/web/api/factory_settings.go:464](../../../../internal/web/api/factory_settings.go#L464); [internal/factory/control/grants.go:85](../../../../internal/factory/control/grants.go#L85); [internal/store/factory_grants.go:103](../../../../internal/store/factory_grants.go#L103).
- **Tests:** [internal/factory/control/grants_test.go:120](../../../../internal/factory/control/grants_test.go#L120) — PostgreSQL integration: active sibling preserves dispatch; withdrawal of last sponsorship closes it; [internal/factory/grants_test.go:57](../../../../internal/factory/grants_test.go#L57) — Source/unit: foreign administrative sponsorship role refuses.
- **Unclear boundaries:** Own sponsorship authority; I03 owns the referenced grant, provider credentials stay in broker custody and F03 accounts allowance. Repository administration cannot substitute for connection ownership.
- **Evidence status:** Current ownership checks/persistence; provider behavior not exercised.
- **Validity review:** [F04 record](../reviews/F04.md) — actual scope, model, findings, challenge and target-allocation status.

## F05 Accepted requirements and invalidation

Bind execution authority to accepted native objectives, selected answers and dependency outcomes at exact versions.

- **Entrypoints:** POST /api/repositories/{repositoryID}/factory/issues/{issueID}/acceptances; POST /api/repositories/{repositoryID}/factory/issues/{issueID}/withdrawal; AdmitAcceptance and AcceptanceStatus.
- **Owned data:** Immutable issue_acceptance_decisions, current heads and withdrawals; selected native versions/digests.
- **Authority:** Current native code-write maintainer; verified eligible creation supports existing initial adoption.
- **Dependencies:** [G01](forgejo-integration.md#g01-browser-authority-and-contributions); [G03](forgejo-integration.md#g03-authoritative-native-reads); [F08](#f08-run-lifecycle-and-intervention).
- **Source files:** [internal/web/api/issue_acceptances.go:159](../../../../internal/web/api/issue_acceptances.go#L159); [internal/factory/control/acceptance.go:405](../../../../internal/factory/control/acceptance.go#L405); [internal/store/issue_acceptances.go](../../../../internal/store/issue_acceptances.go); [docs/product/overview.md:145](../../../product/overview.md#L145).
- **Tests:** [internal/factory/control/acceptance_test.go:71](../../../../internal/factory/control/acceptance_test.go#L71) — PostgreSQL integration with fake native evidence: exact inputs persist, identical command replays, stale predecessor refuses; [internal/factory/control/acceptance_test.go:329](../../../../internal/factory/control/acceptance_test.go#L329) — PostgreSQL integration: source/version/edge changes invalidate; revision change alone does not.
- **Unclear boundaries:** Own accepted meaning/history. G03 returns evidence and F06 evaluates readiness. Restored text at a new version cannot reactivate an old acceptance.
- **Evidence status:** Current acceptance source; complete native adoption/withdrawal journey not run.
- **Validity review:** [F05 record](../reviews/F05.md) — actual scope, model, findings, challenge and target-allocation status.

## F06 Issue intake and readiness

Reassess issues and dependants from authoritative evidence, distinguishing blocked, queued and unauthorized work.

- **Entrypoints:** POST /api/factory/intake; ObserveIssueEvent; ReconcileReadiness and readiness sweeps.
- **Owned data:** intake_deliveries, issue_controls, factory_readiness_sweeps and bounded traversal cursors.
- **Authority:** Verified delivery HMAC identifies hints; accepted inputs and effective authority determine readiness. Hints authorize no execution.
- **Dependencies:** [F01](#f01-repository-factory-policy); [F02](#f02-operator-execution-grants); [F03](#f03-capacity-reservations-and-accounting); [F04](#f04-connection-sponsorship); [F05](#f05-accepted-requirements-and-invalidation); [G03](forgejo-integration.md#g03-authoritative-native-reads); [P07](projects.md#p07-checkout-allocation-and-preparation).
- **Source files:** [internal/web/server.go:70](../../../../internal/web/server.go#L70); [internal/web/api/factory_intake.go:54](../../../../internal/web/api/factory_intake.go#L54); [internal/factory/control/readiness.go:67](../../../../internal/factory/control/readiness.go#L67); [internal/factory/control/readiness_sweep.go:63](../../../../internal/factory/control/readiness_sweep.go#L63).
- **Tests:** [internal/factory/control/readiness_test.go:178](../../../../internal/factory/control/readiness_test.go#L178) — PostgreSQL integration with fake snapshots: duplicate delivery skips reads; another unchanged observation still reads fresh evidence; [internal/web/api/factory_intake_test.go:110](../../../../internal/web/api/factory_intake_test.go#L110) — Source/unit HTTP handler with fake coordinator: forged deliveries refuse; [internal/web/api/factory_intake_test.go:130](../../../../internal/web/api/factory_intake_test.go#L130) — Unit HTTP handler with fake coordinator: assessment failure returns 503; new native status/body diagnostic logging is not asserted.
- **Unclear boundaries:** Owns blockers/reassessment. F05 owns acceptance; G03 owns evidence integrity; F07 owns dispatch. Assessment failure diagnostics are service-log observations, not readiness evidence; the existing 503 response remains.
- **Evidence status:** Current committed intake/readiness source inspected; automatic end-to-end operation and diagnostic output not qualified here.
- **Validity review:** [F06 record](../reviews/F06.md) — actual scope, model, findings, challenge and target-allocation status.

## F07 Assignment and dispatch

Turn eligible queued work into one bounded assignment and recorded supervised launch.

- **Entrypoints:** DispatchPass and Coordinator.Dispatch; DispatchReads; GET /api/repositories/{repositoryID}/factory/issues/{issueID}/assignment.
- **Owned data:** factory_assignments and factory_dispatch_regs; atomic packet references F03 reservations and F08 run/view records.
- **Authority:** Intersection of current policy, operator permission, sponsorship, accepted inputs and approved Project preparation.
- **Dependencies:** [F01](#f01-repository-factory-policy); [F02](#f02-operator-execution-grants); [F03](#f03-capacity-reservations-and-accounting); [F04](#f04-connection-sponsorship); [F05](#f05-accepted-requirements-and-invalidation); [F06](#f06-issue-intake-and-readiness); [G03](forgejo-integration.md#g03-authoritative-native-reads); [P05](projects.md#p05-project-startstop); [P06](projects.md#p06-factory-role-accounts); [P07](projects.md#p07-checkout-allocation-and-preparation); [P08](projects.md#p08-preparation-requirements-acceptance); [P09](projects.md#p09-privileged-preparation-approval); [P12](projects.md#p12-maintenance-holds); [I05](identity-brokering.md#i05-native-binding-and-private-delivery); [I09](identity-brokering.md#i09-provider-execution-integration).
- **Source files:** [internal/factory/control/dispatch.go:15](../../../../internal/factory/control/dispatch.go#L15); [internal/factory/control/dispatch.go:714](../../../../internal/factory/control/dispatch.go#L714); [internal/web/api/dispatch_inputs.go](../../../../internal/web/api/dispatch_inputs.go); [internal/store/factory_assignments.go](../../../../internal/store/factory_assignments.go); [internal/factory/control/dispatch_attempt.go:476](../../../../internal/factory/control/dispatch_attempt.go#L476); [internal/factory/assignment.go:248](../../../../internal/factory/assignment.go#L248).
- **Tests:** [internal/factory/control/dispatch_test.go:247](../../../../internal/factory/control/dispatch_test.go#L247) — PostgreSQL integration with fake host/broker: oldest eligible issue launches within capacity; assignment/prompt/preparation/reservation bindings match; [internal/factory/assignment_test.go:134](../../../../internal/factory/assignment_test.go#L134) — Result fence parser vectors include nested code fences in JSON summary; source-only assertion coverage, not provider output qualification.
- **Unclear boundaries:** Owns assignment admission/recovery and supplies the selected pinned harness and connection. I09 owns provider-matched lease acquisition and native effects; dispatch reads broker metadata rather than receiving credentials. Missing approved preparation waits; this slice does not own Project creation.
- **Evidence status:** Current dispatch/wiring; fake launch assertions do not establish real provider execution.
- **Validity review:** [F07 record](../reviews/F07.md) — actual scope, model, findings, challenge and target-allocation status.

## F08 Run lifecycle and intervention

Settle execution truthfully and admit scoped stop, pause, retry and member takeover controls.

- **Entrypoints:** POST /operator/factory via soda-factory status/stop/reconcile; POST /api/repositories/{repositoryID}/factory/actions; POST /api/factory/runs/{runID}/actions; GET /api/factory/runs/{runID}/output; GET /api/factory/commands/{commandID}.
- **Owned data:** factory_runs; run-action command receipts, factory_run_views, factory_takeovers and lifecycle withdrawal state.
- **Authority:** Configured OS peer for private operator commands; native maintainer for browser controls; membership separately required for takeover.
- **Dependencies:** [G01](forgejo-integration.md#g01-browser-authority-and-contributions); [F07](#f07-assignment-and-dispatch); [P05](projects.md#p05-project-startstop); [P06](projects.md#p06-factory-role-accounts); [I05](identity-brokering.md#i05-native-binding-and-private-delivery); [I06](identity-brokering.md#i06-completion-revocation-and-reconciliation).
- **Source files:** [rust/soda-factory/src/main.rs:1](../../../../rust/soda-factory/src/main.rs#L1); [internal/factory/control/operator.go:16](../../../../internal/factory/control/operator.go#L16); [internal/factory/control/settle.go:42](../../../../internal/factory/control/settle.go#L42); [internal/factory/control/lifecycle.go](../../../../internal/factory/control/lifecycle.go); [internal/web/api/factory_lifecycle.go:125](../../../../internal/web/api/factory_lifecycle.go#L125); [internal/factory/control/settle.go:259](../../../../internal/factory/control/settle.go#L259); [rust/soda-host/src/pfactory.rs:1724](../../../../rust/soda-host/src/pfactory.rs#L1724); [rust/soda-host/src/pfactory.rs:2459](../../../../rust/soda-host/src/pfactory.rs#L2459).
- **Tests:** [internal/factory/control/operator_test.go:93](../../../../internal/factory/control/operator_test.go#L93) — PostgreSQL integration with stubs: replay avoids duplicate stop, changed command conflicts, unfinished replay returns 202; [internal/factory/control/lifecycle_test.go:212](../../../../internal/factory/control/lifecycle_test.go#L212) — PostgreSQL integration with stub execution: takeover requires reconciled run and confirmed copy; repeated takeover copies once; [internal/factory/control/settle_test.go:16](../../../../internal/factory/control/settle_test.go#L16) — Source-only optional-PostgreSQL scenario with stub host/broker: second close succeeds, Cancelled outcome and stored reconciliation; [internal/factory/control/settle_test.go:55](../../../../internal/factory/control/settle_test.go#L55) — Three failed closes keep an uncertain receipt; [rust/soda-host/src/pfactory.rs:4883](../../../../rust/soda-host/src/pfactory.rs#L4883) — Scripted stop-owned uncertainty regression; [rust/soda-host/src/pfactory.rs:4941](../../../../rust/soda-host/src/pfactory.rs#L4941) — Scripted pre-delivery stop despite absent unit; [rust/soda-host/src/pfactory.rs:4959](../../../../rust/soda-host/src/pfactory.rs#L4959) — Scripted broker close succeeds on second attempt.
- **Unclear boundaries:** Own coordination, not native retirement/credential return. Retry delegates admission to F07. Spaces observes run output without acquiring execution control.
- **Evidence status:** Delta source mapped at 0d8d3b8e. Go settlement attempts broker closure up to three times, with a one-second wait that context cancellation can end. Rust stop uses three attempts with 200 ms waits and preserves stop-owned outcomes. Source assertions only; no tests, concurrent native races or installed settlement exercised.
- **Validity review:** [F08 record](../reviews/F08.md) — actual scope, model, findings, challenge and target-allocation status.

## F09 Publication progression

Advance reconciled coding output into an attributable candidate branch and PR while preserving partial or uncertain outcomes.

- **Entrypoints:** PublishPass; PublishCorrection; PublicationExecutor handoff.
- **Owned data:** factory_publications: candidate head, PR link, initial/correction intents, adopted receipts and withdrawal progression.
- **Authority:** Current accepted inputs/factory grants; publication and creation actor bindings reference separate native authority.
- **Dependencies:** [F01](#f01-repository-factory-policy); [F05](#f05-accepted-requirements-and-invalidation); [F07](#f07-assignment-and-dispatch); [F08](#f08-run-lifecycle-and-intervention); [G04](forgejo-integration.md#g04-candidate-publication-and-pr-creation); [P06](projects.md#p06-factory-role-accounts).
- **Source files:** [internal/factory/control/publication.go:16](../../../../internal/factory/control/publication.go#L16); [internal/factory/control/correction.go:55](../../../../internal/factory/control/correction.go#L55); [internal/store/factory_publications.go](../../../../internal/store/factory_publications.go).
- **Tests:** [internal/factory/control/publication_test.go:314](../../../../internal/factory/control/publication_test.go#L314) — PostgreSQL integration with fake executor: distinct branch/PR intents link one exact PR without replayed mutations; [internal/factory/control/publication_native_test.go:365](../../../../internal/factory/control/publication_native_test.go#L365) — Optional native driver: exact native tip/PR checked against receipts; seeded runs/export do not prove CLI execution.
- **Unclear boundaries:** Own local stages and candidate-head/receipt updates. G04 validates and executes native effects; F10 decides review/correction progression without a second publication ledger.
- **Evidence status:** Current progression/native-driver source; no publication executed.
- **Validity review:** [F09 record](../reviews/F09.md) — actual scope, model, findings, challenge and target-allocation status.

## F10 Independent review and correction

Run independent exact-candidate review, return findings to coding and require fresh review after bounded corrections.

- **Entrypoints:** SubmitReviewForRun; Review-cycle and retry planning; Settled reviewer output and correction result handoffs.
- **Owned data:** Role/run/result bindings, attempt allowances and review intent/outcome metadata within existing publication record.
- **Authority:** Distinct approved reviewer role/actor; current accepted requirements and bounded active-time/correction allowance.
- **Dependencies:** [F01](#f01-repository-factory-policy); [F05](#f05-accepted-requirements-and-invalidation); [F07](#f07-assignment-and-dispatch); [F08](#f08-run-lifecycle-and-intervention); [F09](#f09-publication-progression); [G05](forgejo-integration.md#g05-native-review-submission); [P06](projects.md#p06-factory-role-accounts); [P07](projects.md#p07-checkout-allocation-and-preparation); [I05](identity-brokering.md#i05-native-binding-and-private-delivery).
- **Source files:** [internal/factory/control/review_cycle.go](../../../../internal/factory/control/review_cycle.go); [internal/factory/control/dispatch_attempt.go](../../../../internal/factory/control/dispatch_attempt.go); [internal/factory/allowance.go](../../../../internal/factory/allowance.go); [docs/product/overview.md:77](../../../product/overview.md#L77); [internal/factory/review_native.go:120](../../../../internal/factory/review_native.go#L120).
- **Tests:** [internal/factory/control/review_cycle_test.go:194](../../../../internal/factory/control/review_cycle_test.go#L194) — PostgreSQL integration with fake reviewer: exact head/base, reviewer and authorization revision bound; stale review refuses; [internal/factory/allowance_test.go:57](../../../../internal/factory/allowance_test.go#L57) — Source/unit: failed corrections consume cycles; replay cannot double-charge or restore exhausted allowance; [internal/factory/review_role_test.go:21](../../../../internal/factory/review_role_test.go#L21) — Review fence parser vectors include nested diff fences inside JSON report body; source-only assertion coverage, not native review success.
- **Unclear boundaries:** Own review/correction decisions, not G05 review effects or F09 publication heads. Shared current functions/nested fields require joint review before extraction.
- **Evidence status:** Current seams; full same-Project correction/fresh-review loop not assessed.
- **Validity review:** [F10 record](../reviews/F10.md) — actual scope, model, findings, challenge and target-allocation status.

## F11 Candidate verification assessment

Apply accepted required-check policy to current candidate evidence and persist a truthful verdict.

- **Entrypoints:** AssessPublicationChecks; CheckPass; Factory issue check-assessment view.
- **Owned data:** factory_check_assessments bound to repository/PR/head/base and policy/native revisions.
- **Authority:** Repository policy defines required checks; configured bound actor supplies native observation. Agent reports cannot authorize a pass.
- **Dependencies:** [F01](#f01-repository-factory-policy); [F05](#f05-accepted-requirements-and-invalidation); [F09](#f09-publication-progression); [G06](forgejo-integration.md#g06-native-ci-observation).
- **Source files:** [internal/factory/control/checks_pass.go:61](../../../../internal/factory/control/checks_pass.go#L61); [internal/factory/control/review_cycle.go:17](../../../../internal/factory/control/review_cycle.go#L17); [internal/factory/checks.go](../../../../internal/factory/checks.go); [internal/store/factory_checks.go](../../../../internal/store/factory_checks.go).
- **Tests:** [internal/factory/control/checks_pass_test.go:55](../../../../internal/factory/control/checks_pass_test.go#L55) — PostgreSQL integration with fake observer: exact candidate/base verdict persists; absent/busy observation waits; [internal/factory/control/checks_native_test.go:283](../../../../internal/factory/control/checks_native_test.go#L283) — Optional native driver: native statuses assessed against required checks; fixture-dependent cases can skip.
- **Unclear boundaries:** Own verdict/persistence; G06 maps evidence. A changed head needs reassessment and F12 rechecks live evidence before merge. Local runner provisioning remains deferred.
- **Evidence status:** Current assessment/wiring; no Actions run or native snapshot verified.
- **Validity review:** [F11 record](../reviews/F11.md) — actual scope, model, findings, challenge and target-allocation status.

## F12 Merge eligibility and completion

Request a permitted verified merge and release dependent work only after attributable native completion.

- **Entrypoints:** MergePass; MergeExecutor observe/submit/lookup/cancel/completion handoff; Open merge row only after checkCurrent confirms passing exact current-head/base assessment.
- **Owned data:** factory_merges with immutable operation work, stage, receipt, withdrawal and completion outcome.
- **Authority:** Current policy/grants, accepted inputs, independent approval, exact checks and configured merge actor binding.
- **Dependencies:** [F01](#f01-repository-factory-policy); [F02](#f02-operator-execution-grants); [F04](#f04-connection-sponsorship); [F05](#f05-accepted-requirements-and-invalidation); [F06](#f06-issue-intake-and-readiness); [F09](#f09-publication-progression); [F10](#f10-independent-review-and-correction); [F11](#f11-candidate-verification-assessment); [G07](forgejo-integration.md#g07-conditional-native-merge).
- **Source files:** [internal/factory/control/merge.go:13](../../../../internal/factory/control/merge.go#L13); [internal/store/factory_merges.go](../../../../internal/store/factory_merges.go); [docs/product/overview.md:84](../../../product/overview.md#L84).
- **Tests:** [internal/factory/control/merge_test.go:413](../../../../internal/factory/control/merge_test.go#L413) — PostgreSQL integration with fake merger: committed effect waits for bookkeeping; unconfirmed/indeterminate outcomes stay fenced; [internal/factory/control/merge_native_test.go:347](../../../../internal/factory/control/merge_native_test.go#L347) — Optional native driver: receipt inspected, actual base tip equals bound head; prepared fixture is not whole-factory proof; [internal/factory/control/merge_test.go:261](../../../../internal/factory/control/merge_test.go#L261) — PostgreSQL/fake merger regression source: failed current check creates no merge row, then passing assessment permits progression; not executed.
- **Unclear boundaries:** Own eligibility/local progression; G07 owns native effects and receipt consistency. Native matcher also checks evidence conditions; jointly distinguish protocol validity from Soda policy before extraction. Reconciliation of an existing row and policy for opening a new row are separately asserted; native negative fixtures seed existing rows directly.
- **Evidence status:** Current coordinator/native drivers; automatic merge not exercised.
- **Validity review:** [F12 record](../reviews/F12.md) — actual scope, model, findings, challenge and target-allocation status.
