# Factory control native fixtures

[Responsibility map index](README.md) · [Coverage snapshot and limits](../README.md).
Page grouping is navigation; the slice IDs retain their individual review ownership.

<a id="coverage-b068616b36c2"></a>

## [internal/factory/control/st15_demo_accept_test.go](../../../../../internal/factory/control/st15_demo_accept_test.go)

Test/fixture assertions inspected as source only; no test, build, native driver or browser run performed. A defined scenario is not a passing qualification result. Opt-in native/composed fixture may require explicit credentials, shared native services and retained evidence destinations; no current runtime/qualification claim.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) / active | 1–15 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) / active | 16–25 | Fixture/protocol support st15Fixture.readEvidence: ST15 evidence #%d: %v; declarations/fields: `st15Fixture.readEvidence` |
| [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) / active | 26–38 | Fixture/protocol support st15Fixture.admitDecision: ST15 admit #%s: %v; declarations/fields: `st15Fixture.admitDecision` |
| [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) / active | 39–124, 287–322 | Readiness and prerequisite semantics in composed demo; declarations/fields: `st15Fixture.observeControl`, `blockerCodes`, `requireBlocker`, `st15Fixture.seedIssues`, `st15Fixture.proveClosureNotOutcome` |
| [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) / active | 125–180 | Fixture/protocol support st15Fixture.admitPLeg: unaccepted P already runnable: %+v; declarations/fields: `st15Fixture.admitPLeg` |
| [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) / active | 181–241 | Fixture/protocol support st15Fixture.admitALeg: A edge to P not observed: %+v; declarations/fields: `st15Fixture.admitALeg` |
| [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) / active | 242–286 | Fixture/protocol support st15Fixture.admitBCLegs: B edge to A not observed: %+v; declarations/fields: `st15Fixture.admitBCLegs` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 323–356 | Dispatch authority and grant-withdrawal composed demo assertions; declarations/fields: `st15Fixture.proveGrantWithdrawal`, `st15Fixture.activateAuthority` |
| [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) / active | 357 | Fixture/protocol support _; declarations/fields: `_` |

<a id="coverage-4074e0c7742b"></a>

<a id="internalfactorycontrolst15_demo_journey_testgo-1"></a>

## [internal/factory/control/st15_demo_journey_test.go](../../../../../internal/factory/control/st15_demo_journey_test.go)

Test/fixture assertions inspected as source only; no test, build, native driver or browser run performed. A defined scenario is not a passing qualification result. Opt-in native/composed fixture may require explicit credentials, shared native services and retained evidence destinations; no current runtime/qualification claim. 26d420f2 starts the control driver during reviewLeg2 and awaits its result during mergeAndDependants; this is fixture scheduling, while the cancellation assertion remains F08. The browser driver lives at the explicit .artifacts/st15-demo/spaces-check.ts path; this Go source read does not verify that driver's current contents or any browser timing/outcome.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [F10](../../slices/factory-coordination.md#f10-independent-review-and-correction) / active | 1–27 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [F10](../../slices/factory-coordination.md#f10-independent-review-and-correction) / active | 28–36 | Fixture/protocol support st15Fixture.providerGate; declarations/fields: `st15Fixture.providerGate` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 37–70, 100–143 | Assignment/dispatch composed demo phase; declarations/fields: `st15Fixture.runForIssue`, `st15Fixture.pollRunForIssue`, `st15Fixture.dispatchA` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 71–99 | Composed fixture launches the external Playwright driver with explicit watch/control mode, credential-file paths and retained receipt path; return browser-process error to caller instead of failing inside a goroutine; declarations/fields: `st15Fixture.runBrowser` |
| [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) / active | 144–186, 597–608 | Factory output/activity browser fixture phase; declarations/fields: `st15Fixture.stallForBrowserDiag`, `mustViewAttempt` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 187–195, 351–378, 609–713 | Composed verification orchestration/retained fixture evidence; scenario phase assertions remain domain-owned; declarations/fields: `tailLines`, `st15Fixture.requireContentStatus`, `st15Fixture.proveRetention`, `st15Fixture.writeFinalReceipt`, `TestST15ComposedDemo` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 196–240, 379–424 | Publication and correction composed demo phase; declarations/fields: `st15Fixture.publishA`, `st15Fixture.correctA` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 241–306 | Exact independent review checkout preparation fixture; declarations/fields: `st15Fixture.prepareReviewer` |
| [F10](../../slices/factory-coordination.md#f10-independent-review-and-correction) / active | 307–328 | Fixture/protocol support st15Fixture.reviewLeg: ST15 review lacks its fenced report; declarations/fields: `st15Fixture.reviewLeg` |
| [F10](../../slices/factory-coordination.md#f10-independent-review-and-correction) / active | 329–336 | Fixture/protocol support st15Fixture.reviewLeg1; declarations/fields: `st15Fixture.reviewLeg1` |
| [F11](../../slices/factory-coordination.md#f11-candidate-verification-assessment) / active | 337–350, 459–484 | Exact-candidate verification composed demo phase; declarations/fields: `st15Fixture.ciFail`, `st15CheckLink`, `st15Fixture.ciPass` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 425–433 | Start the dependant-control browser fixture asynchronously before fresh review and later merge/dispatch, carrying its error on a buffered channel to reduce the scenario's late-attachment race; declarations/fields: `st15Fixture.reviewLeg2`, `st15Fixture.controlBrowser` |
| [F10](../../slices/factory-coordination.md#f10-independent-review-and-correction) / active | 434–458 | Source scenario rejects stale head-1 review observation, prepares an independent reviewer for corrected head-2, and requires a fresh APPROVED event; declarations/fields: `st15Fixture.reviewLeg2` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 485–512 | Drive production MergePass asynchronously until the merge settles, preserving refusal/error reporting for the composed completion-cascade scenario; declarations/fields: `st15Fixture.mergeAndDependants` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 513–544 | Observe automatic dependant assignment/run pickup after merge completion, with no synthesized intake fallback; retain exact run-to-attempt association; declarations/fields: `st15Fixture.mergeAndDependants`, `st15Fixture.runForIssue`, `mustViewAttempt` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 545–553 | Await the previously started control browser's fixture verdict and return missing-channel or driver errors to the composed scenario; declarations/fields: `st15Fixture.mergeAndDependants`, `st15Fixture.controlBrowser` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 554–571 | Require completed merge outcome and retain the merged commit/publication linkage plus the completion-cascade trigger receipt; declarations/fields: `st15Fixture.mergeAndDependants` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 572–596 | After the external browser intervention, source scenario waits for the dependant assignment to finish and requires Cancelled outcome; retain run/assignment cancellation receipt; declarations/fields: `st15Fixture.mergeAndDependants` |

<a id="coverage-10c77ae3f1f1"></a>

<a id="internalfactorycontrolst15_demo_native_testgo-1"></a>

## [internal/factory/control/st15_demo_native_test.go](../../../../../internal/factory/control/st15_demo_native_test.go)

Test/fixture assertions inspected as source only; no test, build, native driver or browser run performed. A defined scenario is not a passing qualification result. Opt-in native/composed fixture may require explicit credentials, shared native services and retained evidence destinations; no current runtime/qualification claim. Includes both the a0ff27a9 root-user-manager fixture addition and 26d420f2 controlBrowser field. H06 units here identify concrete test host/container/build/process/evidence assembly; they do not assign production Project, identity, factory or installation decisions to development tooling. No installed linger requirement is inferred from the fixture comment. a0ff27a9 adds fixture-only user@0.service setup; its production comment is a review hypothesis, not an adopted contract or installed proof.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 1–38 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 39–40 | Fixture/protocol support st15Image; declarations/fields: `st15Image` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 41–64 | Fixture/protocol support st15Config; declarations/fields: `st15Config` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 65–81 | Fixture/protocol support loadST15: ST15 fixture requires the full native/daemon/browser configuration; declarations/fields: `loadST15` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 82–91 | Fixture/protocol support st15ReceiptDir: ST15_RECEIPT_DIR must be an absolute retained directory; declarations/fields: `st15ReceiptDir` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 92–101 | Fixture/protocol support st15Receipt; declarations/fields: `st15Receipt` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 102–141, 147–149 | Fixture/protocol support st15Fixture; declarations/fields: `st15Fixture` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 142–146 | Fixture-only buffered browser error channel and comments coordinate early browser startup with dependant-stop phase; no durable product state; declarations/fields: `st15Fixture.controlBrowser` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 150–154 | Fixture/protocol support st15Check; declarations/fields: `st15Check` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 155–167 | Fixture/protocol support st15Fixture.check: ST15 %s: %v; declarations/fields: `st15Fixture.check` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 168–186 | Fixture/protocol support st15Podman: podman %s: %w: %s; declarations/fields: `st15Podman` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 187–199 | Fixture/protocol support st15Pexec; declarations/fields: `st15Pexec` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 200–211 | Fixture/protocol support st15Git: git %s: %w: %s; declarations/fields: `st15Git` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 212–216 | Fixture/protocol support st15SHA256; declarations/fields: `st15SHA256` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 217–226 | Fixture/protocol support st15RandHex; declarations/fields: `st15RandHex` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 227–247 | Fixture/protocol support st15RepoRoot: repository root not found; declarations/fields: `st15RepoRoot` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 248–263 | Fixture/protocol support st15BuildBroker: build soda-identity: %v\n%s; declarations/fields: `st15BuildBroker` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 264–279 | Fixture/protocol support st15BuildHost: build soda-host: %v\n%s; declarations/fields: `st15BuildHost` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 280–295 | Fixture/protocol support st15BuildFactoryRoles: build project-factory-roles: %v\n%s; declarations/fields: `st15BuildFactoryRoles` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 296–305 | Fixture/protocol support st15TmpfsRoot; declarations/fields: `st15TmpfsRoot` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 306–318 | Fixture/protocol support st15FakeCodex; declarations/fields: `st15FakeCodex` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 319–326 | Fixture/protocol support st15FakeMuse; declarations/fields: `st15FakeMuse` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 327–340 | Fixture/protocol support st15WaitSocket: broker socket %s never appeared; declarations/fields: `st15WaitSocket` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 341–492, 500–620 | Fixture/protocol support st15Fixture.setupHostStack: fixture userns is %q, need private; declarations/fields: `st15Fixture.setupHostStack` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 493–499 | Fixture transiently starts root user@0.service before launching the production host daemon; the comment's installed linger claim is an unqualified hypothesis, not adopted installation policy or proof; declarations/fields: `st15Fixture.setupHostStack` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 621–633 | Fixture/protocol support st15Fixture.openFactoryStore; declarations/fields: `st15Fixture.openFactoryStore` |

<a id="coverage-00cb8f9b9b02"></a>

## [internal/factory/control/st15_demo_runs_test.go](../../../../../internal/factory/control/st15_demo_runs_test.go)

Test/fixture assertions inspected as source only; no test, build, native driver or browser run performed. A defined scenario is not a passing qualification result. Opt-in native/composed fixture may require explicit credentials, shared native services and retained evidence destinations; no current runtime/qualification claim.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 1–24 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 25–72, 180–197 | Assignment/launch fixture bookkeeping; declarations/fields: `st15Fixture.launchDirect`, `st15Fixture.settleDispatched` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 73–93 | Fixture/protocol support st15Fixture.stopRunSettled; declarations/fields: `st15Fixture.stopRunSettled` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 94–152 | Fixture/protocol support st15Fixture.settleDirect: ST15 run %s retired uncertain; declarations/fields: `st15Fixture.settleDirect` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 153–179 | Fixture/protocol support st15Fixture.stopSettled: ST15 run %s retired uncertain: %s; declarations/fields: `st15Fixture.stopSettled` |
| [F10](../../slices/factory-coordination.md#f10-independent-review-and-correction) / active | 198–231 | Independent review execution/report fixture; declarations/fields: `st15Fixture.reviewPrompt`, `st15Fixture.submitReview` |
| [G06](../../slices/forgejo-integration.md#g06-native-ci-observation) / active | 232–243 | Explicit native fixture status publication; declarations/fields: `st15Fixture.seedCIStatus` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 244 | Fixture/protocol support _; declarations/fields: `_` |

<a id="coverage-d50be1d675ed"></a>

<a id="internalfactorycontrolst15_demo_seed_testgo-1"></a>

## [internal/factory/control/st15_demo_seed_test.go](../../../../../internal/factory/control/st15_demo_seed_test.go)

Test/fixture assertions inspected as source only; no test, build, native driver or browser run performed. A defined scenario is not a passing qualification result. Opt-in native/composed fixture may require explicit credentials, shared native services and retained evidence destinations; no current runtime/qualification claim.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 1–27 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 28–36 | Fixture/protocol support st15Fixture.st09; declarations/fields: `st15Fixture.st09` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 37–41 | Fixture/protocol support st15Fixture.api; declarations/fields: `st15Fixture.api` |
| [G04](../../slices/forgejo-integration.md#g04-candidate-publication-and-pr-creation) / active | 42–62, 128–168 | Publication source repository fixture; declarations/fields: `st15Fixture.git`, `st15Fixture.repoURL`, `st15Fixture.seedRepository` |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 63–127 | Existing Project creation fixture; declarations/fields: `st15Fixture.setupProject` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 169–271 | Exact-source prepared role checkout fixture; declarations/fields: `st15Fixture.prepareRoles` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 272–322 | Retained human account/workspace isolation fixture; declarations/fields: `st15Fixture.seedHumanWork` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 323–360 | Fixture/protocol support st15Fixture.wireCoordinator; declarations/fields: `st15Fixture.wireCoordinator` |
| [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) / active | 361–409, 430–454 | Intake/issues/prerequisite fixture data; declarations/fields: `st15Fixture.postIntake`, `st15Issue`, `st15Fixture.createIssue`, `st15Fixture.commentIssue`, `st15Fixture.insertEdge`, `st15Fixture.issueOpenedHint`, `st15Fixture.commentHint` |
| [G03](../../slices/forgejo-integration.md#g03-authoritative-native-reads) / active | 410–429 | Current native issue/comment read fixture; declarations/fields: `st15Fixture.readIssue`, `st15Fixture.readComment` |

