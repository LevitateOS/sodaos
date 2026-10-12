# Browser spaces

[Responsibility map index](README.md) · [Coverage scope](../README.md).

Current responsibility accounting at `519b76bd` (2026-10-07).
One slice owns each named duty; disjoint complete symbols may share an owner.
Compound fields/clauses may share a physical line with distinct selectors.
Disposition concerns the duty, not source validity or installed qualification.

<a id="coverage-ca8165c83d8e"></a>

## [frontend/spaces/soda-extension.ts](../../../../../frontend/spaces/soda-extension.ts)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–2, 12–27, 84; file scaffold; nativeBase; PreparedExtensionMount | [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) | retained | Current scaffold duty: file scaffold; Current declaration duty: nativeBase; Current declaration duty: PreparedExtensionMount — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 3–11, 28–75; ExtensionMountContext; prepareExtensionMount | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | Current declaration duty: ExtensionMountContext; Current declaration duty: prepareExtensionMount — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 76–83; extensionURL | [P05](../../slices/projects.md#p05-project-startstop) | retained | Current declaration duty: extensionURL — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-f710ab93aa2b"></a>

## [frontend/spaces/soda-identity-response.ts](../../../../../frontend/spaces/soda-identity-response.ts)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–2, 36–44, 111–114; text, identityID, items | [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) | retained | Bounded representation parsing and encoding; declarations/fields: `text`, `identityID`, `items` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 3, 13–19, 45–48, 67–86; ProviderID, Enrollment, provider, enrollmentView, enrollmentOrigin | [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) | retained | Provider enrollment, owner consent and connection selection; declarations/fields: `ProviderID`, `Enrollment`, `provider`, `enrollmentView`, `enrollmentOrigin` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 4–12, 20–26, 49–66, 87–97; Connection, Grant, connectionView, availableConnectionView, grantView | [I03](../../slices/identity-brokering.md#i03-delegation-and-connection-availability) | retained | Named delegation and derived connection availability; declarations/fields: `Connection`, `Grant`, `connectionView`, `availableConnectionView`, `grantView` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 27–35, 98–110; Lease, leaseView | [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) | retained | Execution lease identity and admission projection; declarations/fields: `Lease`, `leaseView` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |

<a id="coverage-896e9d27756b"></a>

## [frontend/spaces/soda-identity.ts](../../../../../frontend/spaces/soda-identity.ts)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–17, 25–29, 36–125, 146–150, 305–344; properties, busy, message, actor, transport, project, retired, contextSerial | [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) | retained | Mounted workspace/component lifetime, navigation and focus; declarations/fields: `properties`, `busy`, `message`, `actor`, `transport`, `project`, `retired`, `contextSerial` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 18–19, 21, 23–24, 30–31, 33, 35, 126–135, 139–143, 176–193, 253–275; connections, selected, grants, available, load, loadSelected, delegate; selectedView | [I03](../../slices/identity-brokering.md#i03-delegation-and-connection-availability) | retained | Named delegation and derived connection availability; declarations/fields: `connections`, `selected`, `grants`, `available`, `load`, `loadSelected`, `delegate`; Connection/grant/lease controls compose existing child actions; declarations/fields: `selectedView` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 20, 32, 136–138, 151–164, 170–175, 237–252; enrollment, load, connect, cancel, enrollmentView | [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) | retained | Provider enrollment, owner consent and connection selection; declarations/fields: `enrollment`, `load`, `connect`, `cancel`, `enrollmentView` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 22, 34, 144–145; leases, loadSelected | [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) | retained | Execution lease identity and admission projection; declarations/fields: `leases`, `loadSelected` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 165–169, 194–205; disconnect, revokeGrant, endLease | [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) | retained | Connection/grant revocation and lease closure; declarations/fields: `disconnect`, `revokeGrant`, `endLease` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 206–236, 276–304; launch, startCodex; projectView | [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) | retained | Native provider launch integration; declarations/fields: `launch`, `startCodex`; Project availability and native Codex launch presentation; declarations/fields: `projectView` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |

<a id="coverage-28ccebb3373e"></a>

## [frontend/spaces/soda-native-paths.ts](../../../../../frontend/spaces/soda-native-paths.ts)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–4, 10–36; file scaffold; validateBase; validatePrefix | [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) | retained | Current scaffold duty: file scaffold; 5 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 5–9; forgejoPrefix | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | Current declaration duty: forgejoPrefix — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-d364231fb1e8"></a>

## [frontend/spaces/soda-spaces-entry.ts](../../../../../frontend/spaces/soda-spaces-entry.ts)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–5; file scaffold | [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) | retained | Current scaffold duty: file scaffold — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 6–25; mount | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | Current declaration duty: mount — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-6f26ed7d76cb"></a>

## [frontend/spaces/soda-workspace-panel-entry.ts](../../../../../frontend/spaces/soda-workspace-panel-entry.ts)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–5; file scaffold | [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) | retained | Current scaffold duty: file scaffold — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 6–27; mount | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | Current declaration duty: mount — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-e6523693d393"></a>
<a id="frontendspacessodaspaces-apits-1"></a>

## [frontend/spaces/sodaspaces-api.ts](../../../../../frontend/spaces/sodaspaces-api.ts)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–43, 46–55; file scaffold; sodaJSONReader; readSodaJSON; object; check; id; fingerprint; SodaRequestError; constructor | [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) | retained | Current scaffold duty: file scaffold; 9 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 44–45; projectId | [P01](../../slices/projects.md#p01-repository-association-and-creation) | retained | Current declaration duty: projectId — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-563f98f1ca43"></a>

## [frontend/spaces/sodaspaces-attention.ts](../../../../../frontend/spaces/sodaspaces-attention.ts)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–4, 40–43; file scaffold; observationReady | [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) | retained | Current scaffold duty: file scaffold; Current declaration duty: observationReady — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 5–39, 44–88; ConnectionState; TerminalObservation; states; observationShape; terminalObservation; closedAttention; transportAttention; staleAttention; attentionReason | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | Current declaration duty: ConnectionState; 9 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-e99452619350"></a>

## [frontend/spaces/sodaspaces-environment-view.ts](../../../../../frontend/spaces/sodaspaces-environment-view.ts)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–4, 70–74, 98–118, 139–151; file scaffold; osReleaseLine; EnvironmentPresentation; EnvironmentCommands; onSSHChange; joinAdmission; environmentBusy | [P01](../../slices/projects.md#p01-repository-association-and-creation) | retained | Current scaffold duty: file scaffold; 7 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 5–69, 75–97; onProjectOSChange; createdProjectOS; projectOSOptionLabel; projectOSHelp; projectOSPicker; renderProjectOS; observedOSBody; renderOSObservation | [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) | retained | Current declaration duty: onProjectOSChange; 8 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 119–138, 152–204; joinKeys; environmentActions; renderEnvironment | [P03](../../slices/projects.md#p03-human-membership-and-accounts) | retained | Current declaration duty: joinKeys; Current declaration duty: environmentActions; Current declaration duty: renderEnvironment — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-493c802d6fa9"></a>

## [frontend/spaces/sodaspaces-factory-access-response.ts](../../../../../frontend/spaces/sodaspaces-factory-access-response.ts)

Delta duty inspection at `93fb3513`. Repository-access read parsers (F06/V05): viewer standing, section editability, connection editability, required actions, per-connection usage minutes, planning allowances in real call units, and preparation state. Each parser binds its exact scope; absent records refuse, never guess.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–194; file scaffold; FactoryAccessViewer; FactoryAccessEditable; FactoryAccessConnectionEditable; FactoryAccessRequiredAction; FactoryAccessUsage; FactoryAccessPlanning; FactoryAccess; reason; editable; connectionEditable; minutes; usageEntry; requiredAction; calls; planningEntry; preparation; factoryAccessResponse | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Current scaffold duty: file scaffold; 16 named units assigned here — Current complete declaration/method or file span inspected at `93fb3513`; historical pins retain their scope. |

<a id="coverage-b1b6adb842ff"></a>

## [frontend/spaces/sodaspaces-factory-activity.ts](../../../../../frontend/spaces/sodaspaces-factory-activity.ts)

Delta duty inspection at `93fb3513`. Repository activity feed read (F05/V06): bounded kinds/items parsing, safe native links, honest unknown timestamps, settings envelope, and the Lit feed element. Consumes the F05 activity DTO with observation time and truncation flags.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–242; file scaffold; kinds; Kind; Settings; ActivityItem; Activity; Context; labels; nativePath; activityEnvelope; activitySettings; parseFactoryActivity; boundedText; timestamp; activityTarget; failureMessage; nativeID; parseItem; activityTimestamp; responseError; requestActivity; SodaFactoryActivity; renderItemTime | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Current scaffold duty: file scaffold; 22 named units assigned here — Current complete declaration/method or file span inspected at `93fb3513`; historical pins retain their scope. |

<a id="coverage-93a45d421445"></a>

## [frontend/spaces/sodaspaces-factory-display.ts](../../../../../frontend/spaces/sodaspaces-factory-display.ts)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–101; file scaffold; FactoryWatchState; DisplayInput; statusText; viewDisabled; canWatchView; watchTitle; watchPresentation; closeMenu; menuKey; watchFromControls; stopFromControls; hideFromControls; watchCommands | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Current scaffold duty: file scaffold; 17 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-36bd57d3f44e"></a>

## [frontend/spaces/sodaspaces-factory-navigation-view.ts](../../../../../frontend/spaces/sodaspaces-factory-navigation-view.ts)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–59; file scaffold; FactoryRunRow; FactoryWatchSlot; renderFactoryRuns | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Current scaffold duty: file scaffold; 4 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-302cf96ba4bb"></a>

## [frontend/spaces/sodaspaces-factory-progress-response.ts](../../../../../frontend/spaces/sodaspaces-factory-progress-response.ts)

Delta duty inspection at `93fb3513`. Strict parsers for the H-R reads (V01/V02/V05/V07): request overview, issue plan, planning allowance, lifecycle effects, repository access and revision history. Every parser binds its exact intent/issue/repository identity; absent records leave the view without a verdict.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–60; decimalID; digest; text; counter; fraction; optionalLabel; judgmentNumbers; countMap; probabilityMap | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Shared parser primitives: identifiers, digests, counters, bounded maps — Current complete declaration/method or file span inspected at `93fb3513`; historical pins retain their scope. |
| 62–217; RequestDirection; RequestContext; RequestMapping; RequestCounts; RequestAttention; RequestOverview; intentStates; attentionCodes; setupStates; decisionKinds; sourceKinds; requestDirection; requestContext; requestMapping; requestCounts; requestAttention; requestSetup; requestOverview; generationState | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Request-overview (F01) shapes and parsers — Current complete declaration/method or file span inspected at `93fb3513`; historical pins retain their scope. |
| 219–435; PlanCandidate; PlanJudgment; RecordedPlan; PlanFreshness; PriorityCandidate; PriorityCall; PriorityComparison; IssuePlan; readinessStates; freshnessStates; freshnessReasons; planCandidate; planJudgment; recordedPlan; planFreshness; priorityCandidate; candidateFlags; blockerCodes; priorityCall; priorityComparison | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Issue-plan (F03) shapes and parsers — Current complete declaration/method or file span inspected at `93fb3513`; historical pins retain their scope. |
| 437–493; PlanningAllowanceView; allowanceAmount; allowanceConsumption; allowanceBounds; allowanceEffect; planningAllowance | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Planning-allowance (H-A) shapes and parsers — Current complete declaration/method or file span inspected at `93fb3513`; historical pins retain their scope. |
| 495–667; LifecycleEffectConfig; LifecycleEffectAction; lifecycleEvents; lifecycleActions; lifecycleResults; lifecycleNote; lifecycleActor; lifecycleReviewers; lifecycleSubmitter; lifecycleEffectConfig; lifecycleResultsMap; lifecycleBoundID; lifecycleAction; lifecycleActionAttempt; lifecycleEffectActions | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Lifecycle effect config/action (L06) shapes and parsers — Current complete declaration/method or file span inspected at `93fb3513`; historical pins retain their scope. |
| 669–817; planCursor; issuePlan; AccessViewer; AccessEntry; AccessRequiredAction; AccessEffective; AccessPlanningEntry; FactoryAccess; accessViewer; accessEntry; accessRequiredAction; accessEffective; accessPlanningEntry; factoryAccess | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Plan paging cursor plus repository-access (F06) shapes and parsers — Current complete declaration/method or file span inspected at `93fb3513`; historical pins retain their scope. |
| 819–1135; RevisionTask; RevisionTaskChange; RevisionTaskDiff; RevisionIdeaBody; RevisionIdeaChange; RevisionIdeaDiff; RevisionScalarChange; RevisionComparison; RevisionWork; RevisionGeneration; RevisionEntry; RevisionHistory; revisionStates; revisionTaskFields; revisionDecisions; revisionGeneration; revisionWorkStates; revisionAuthority; revisionOutcomes; revisionIDList; revisionTask; revisionChangedTexts; revisionChangedOutcomes; revisionTaskChange; revisionTaskList; revisionTaskDiff; revisionIdeaBody; revisionIdeaBodyList; revisionIdeaChange; revisionIdeaDiff; revisionScalarChange; revisionEffectiveDiff; checkEmptyInitial; revisionComparison; revisionWork; entryGeneration; revisionEntry; revisionHistory | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Revision-history (F04) shapes and parsers — Current complete declaration/method or file span inspected at `93fb3513`; historical pins retain their scope. |
| 156–172; requestCounts sparse zeros | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Delta at `31caf48a` (V/a9bf): absent canonical by_state buckets read as zero; provided values, unknown keys, totals and IDs still validate exactly. Prior rows retain their pinned scope. |

<a id="coverage-daea7d82cb51"></a>

## [frontend/spaces/sodaspaces-factory-progress-view.ts](../../../../../frontend/spaces/sodaspaces-factory-progress-view.ts)

Delta duty inspection at `93fb3513`. Progress models plus Lit render helpers (V01/V02/V05/V07): work overview, why-next/prerequisites, direction revision, allowance, request overview, recorded plan/choice, lifecycle effects, planning reference, access standing and revision history. Models describe; renderers present; neither judges nor dispatches.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 34–249; IntentTaskBinding; IntentGeneration; generationBinding; intentGeneration; SourceState; WorkOverviewInput; OverviewCounts; AttentionItem; DependencyRow; WorkOverviewModel; deliveryBuckets; bucketDelivery; countIssues; countsText; directionText; issueForTask; dependencyRows; setupText; outcomeNote; availabilityNote; describeWorkOverview; renderWorkOverview | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Work-overview model and renderer — Current complete declaration/method or file span inspected at `93fb3513`; historical pins retain their scope. |
| 250–346; EligibleWork; WhyNextModel; ageText; describeWhyNext; PrerequisiteModel; describePrerequisites; renderWorkPlan | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Why-next and prerequisite models plus plan renderer — Current complete declaration/method or file span inspected at `93fb3513`; historical pins retain their scope. |
| 348–450; DirectionRevisionModel; describeDirectionRevision; AllowanceModel; describeAllowance; renderAllowance; renderDirectionRevision | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Direction-revision and allowance models plus renderers — Current complete declaration/method or file span inspected at `93fb3513`; historical pins retain their scope. |
| 451–570; RequestMappingRow; RequestOverviewModel; requestAttentionItem; requestSetupText; requestGenerationText; requestDirectionText; describeRequestOverview; renderRequestOverview | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Request-overview model and renderer — Current complete declaration/method or file span inspected at `93fb3513`; historical pins retain their scope. |
| 572–675; RecordedPlanModel; freshnessReasonText; freshnessBase; describeRecordedPlan; ChoiceCandidateRow; RecordedChoiceModel; durationText; estimatesText; choiceSelection; judgmentText; describeRecordedChoice | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Recorded-plan and recorded-choice models — Current complete declaration/method or file span inspected at `93fb3513`; historical pins retain their scope. |
| 676–897; LifecycleEventKind; LifecycleEffectModel; lifecycleTitles; lifecycleTargets; lifecycleResultLink; lifecycleResultLine; lifecycleSubmitterText; describeLifecycleEffect; LifecycleEffectInput; LifecycleEffectCommands; renderIssueCreatedFields; renderPrCreatedFields; renderVerifiedMergeFields; renderEffectFields; renderLifecycleEffects; renderRecordedPlan | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Lifecycle-effect model, per-event field renderers and plan renderer — Current complete declaration/method or file span inspected at `93fb3513`; historical pins retain their scope. |
| 898–954; PlanningReferenceModel; planningEntryText; describePlanningReference; renderPlanningReference | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Planning-reference model and renderer — Current complete declaration/method or file span inspected at `93fb3513`; historical pins retain their scope. |
| 955–1022; AccessStandingModel; standingPowers; standingVerdict; standingAction; describeAccessStanding; renderAccessStanding | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Access-standing model and renderer — Current complete declaration/method or file span inspected at `93fb3513`; historical pins retain their scope. |
| 1023–1175; RevisionEntryModel; RevisionHistoryModel; revisionTaskChangedLine; revisionTaskLines; revisionIdeaChangedLine; revisionIdeaLines; revisionScopeLines; revisionSubmittedNote; revisionEntryLineage; describeRevisionEntry; describeRevisionHistory; renderRevisionHistory | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Revision-entry/history models and renderer — Current complete declaration/method or file span inspected at `93fb3513`; historical pins retain their scope. |
| 552–645; OverviewScopeIssue; OverviewScope; overviewScopeFromMappings; overviewScopeFromBound; retiredOverviewScope; planContextFromScope; DetailScopeRuns; detailContextFromScope; revisionContextFromScope | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Delta at `deb79186` (V159): overview-scope feed shapes plus plan/detail/revision child-context builders. |

<a id="coverage-64f01def96fa"></a>

## [frontend/spaces/sodaspaces-factory-request.ts](../../../../../frontend/spaces/sodaspaces-factory-request.ts)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–216; file scaffold; RequestInput; blockedWatch; endedRun; inspectRun; attachPayload; onPeerOpen; acceptStatus; acceptOutput; dispatchFrame; detachClosed; onPeerMessage; onPeerClosed; bindPeer; watch; attachPeer | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Current scaffold duty: file scaffold; 17 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-9665c08139f3"></a>

## [frontend/spaces/sodaspaces-factory-response.ts](../../../../../frontend/spaces/sodaspaces-factory-response.ts)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–333; file scaffold; FactoryAuthority; FactoryControl; FactoryRun; factoryAuthorityText; factoryControlText; factoryCommandId; spaceFactoryAuthority; check; admitUnsettledRuns; admitWithdrawalCause; spaceFactoryControl; factoryRunText; factoryOutcome; factoryAttempt; admitFactoryRunIdentity; admitFactoryRunOutcome; admitFactoryRunBinding; spaceFactoryRun; spaceFactoryRuns; FactoryRunView; FactoryRunState; FactoryRunDetail; factoryPhase; factoryRunDetailView; factoryStateIdentity; factoryStateBinding; factoryStateExit; factoryStateReason; factoryStateOutput; factoryStateOutcome; factoryRunDetailState; factoryRecordIdentity; factoryRecordHarness; factoryRecordOutcome; factoryRecordProvenance; factoryRunStatusResponse; FactoryStatus; factoryFrameIdentity; factoryFrameBinding; factoryFrameOutcome; factoryStatusFrame | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Current scaffold duty: file scaffold; 79 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-ddc6cbac8535"></a>

## [frontend/spaces/sodaspaces-factory-screen.ts](../../../../../frontend/spaces/sodaspaces-factory-screen.ts)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–128; file scaffold; FactoryFit; FactoryScreenInput; clearScreen; fitReady; screenPresent; canFit; resize; requiredToken; terminalTheme; awaitScreen; openScreen; screenReady | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Current scaffold duty: file scaffold; 13 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-1c75bea449eb"></a>

## [frontend/spaces/sodaspaces-factory-settings.ts](../../../../../frontend/spaces/sodaspaces-factory-settings.ts)

Delta duty inspection at `93fb3513`. Factory settings controls: context/view/preparation state, safe text/number helpers, access reason/actor/usage text with allowance spread (delta), decision/preparation review helpers, error/input handling, and the settings element sending explicit acceptance, approval, preparation and policy actions.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 14–111; FactoryContext; View; PendingPreparation; asObject; safeText; safeDetail; safeNumber; accessReasonText; accessActorText; accessUsageStatusText; accessSpread | [F01](../../slices/factory-coordination.md#f01-repository-factory-policy) | retained | Context state, safe helpers and access text — Current complete declaration/method or file span inspected at `93fb3513`; historical pins retain their scope. |
| 112–258; factoryDecisionId; factoryPreparationId; inputChecked; hasRequiredSetupFiles; validSourcePreview; validSourceDigests; exactReturnedPaths; selectedSourceInputs; validEvidence; approvalMatches; matchingPreparations; existingPreparation; requiresReplacementReview; pendingPreparationIds; replacementId; validDecisionReview | [P08](../../slices/projects.md#p08-preparation-requirements-acceptance) | retained | Decision/preparation review helpers — Current complete declaration/method or file span inspected at `93fb3513`; historical pins retain their scope. |
| 259–314; FactoryRequestError; inputValue; inputLines; sameStrings; sameStringSet; retainPreview; retiredPreparation; matchesSourceSelection; integer | [P09](../../slices/projects.md#p09-privileged-preparation-approval) | retained | Error/input and selection-match helpers — Current complete declaration/method or file span inspected at `93fb3513`; historical pins retain their scope. |
| 315–1447; SodaFactorySettings | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Settings element; class-internal duties recorded as one unit — Current complete declaration/method or file span inspected at `93fb3513`; historical pins retain their scope. |
| 23–32; effectsRepositoryPath | [F01](../../slices/factory-coordination.md#f01-repository-factory-policy) | retained | Delta at `deb79186`: validated native repository path for effects mounts. Prior rows retain their pinned scope. |
<a id="coverage-9870e3e4fbf7"></a>

## [frontend/spaces/sodaspaces-factory-stream-response.ts](../../../../../frontend/spaces/sodaspaces-factory-stream-response.ts)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–58; file scaffold; FactoryOutput; factoryOutputBytes; check; factoryOutputFrame; factoryOutputCursor; factoryClosedReason | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Current scaffold duty: file scaffold; 15 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-1741f8be517e"></a>

## [frontend/spaces/sodaspaces-factory-view.ts](../../../../../frontend/spaces/sodaspaces-factory-view.ts)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–60; file scaffold; FactoryWatchPresentation; FactoryWatchCommands; renderFactoryWatch | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Current scaffold duty: file scaffold; 4 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty.; sodaspaces-factory.ts calls renderFactoryWatch with watchPresentation and watchCommands; the component renders factory activity and read-only output without owning transport or human-terminal lifecycle. |

<a id="coverage-d504c133fbb5"></a>

## [frontend/spaces/sodaspaces-factory-work.ts](../../../../../frontend/spaces/sodaspaces-factory-work.ts)

Delta duty inspection at `93fb3513`. Factory work mount (V01–V07): transport helpers, work-overview/plan/direction-revision/optional-effects/detail Lit elements with their reads, intervention posting with receipt parsing, and route admission plus mount entrypoints. Class-internal method duties are not split here; concern decomposition awaits the pending S review.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 97–158; WorkReadError; getJSON; Posted; writeJSON; postJSON; putJSON; readFailureText; failureInput | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Transport helpers and failure classification — Current complete declaration/method or file span inspected at `93fb3513`; historical pins retain their scope. |
| 160–357; WorkOverviewContext; overviewSignature; sameOverview; SodaFactoryWorkOverview; readOverviewRevision; readOverviewGeneration; readOverviewSummary; readOverviewIssues; readRequestOverview; readOverview | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Work-overview element and reads — Current complete declaration/method or file span inspected at `93fb3513`; historical pins retain their scope. Superseded in part at `31caf48a`: readOverviewRevision/readOverviewGeneration/readOverviewIssues were removed; see the superseding row below. |
| 359–514; WorkPlanContext; planSignature; samePlan; RecordedPlanEntry; SodaFactoryWorkPlan; readRecordedPlans; readPlanIssues | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Work-plan element and reads — Current complete declaration/method or file span inspected at `93fb3513`; historical pins retain their scope. |
| 516–788; DirectionRevisionContext; SodaFactoryDirectionRevision; allowanceFormInput; allowanceUsage; allowanceAmountError; allowanceIncreaseEffect; readDirectionRevision; readPlanningAllowance; readFactoryAccess; readRevisionAccess; readRevisionHistory | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Direction-revision element with allowance form and reads — Current complete declaration/method or file span inspected at `93fb3513`; historical pins retain their scope. |
| 790–1079; LifecycleEffectsContext; lifecycleEvents; lifecycleConfigPath; lifecycleActionsPath; EffectEntry; EffectTargets; parseEffectTargets; effectTargetsError; effectSaveBody; lifecycleSaveBody; SodaFactoryOptionalEffects; conflictSaveMessage; sameEffectsContext; effectsContextKey; emptyEntries; readEffectConfig; readEffectActions; readEffectEntries | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Optional-effects element, save bodies and reads — Current complete declaration/method or file span inspected at `93fb3513`; historical pins retain their scope. |
| 1081–1449; WorkDetailContext; RunEntry; InterventionMarks; idleMarks; detailSignature; sameDetail; runRowText; SodaFactoryWorkDetail; refreshesDetail; interventionTarget; readDetailIssue; readDetailRuns; postIntervention; parseRepositoryReceipt; parseRunReceipt; parsePostedReceipt; parsePostedOutcome | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Work-detail element, intervention posting and receipt parsing — Current complete declaration/method or file span inspected at `93fb3513`; historical pins retain their scope. |
| 1451–1527; admitWorkRoot; admitWorkRepository; admitWorkIssue; admitWorkPath; admitWorkIntent; admitWorkRuns; admitWorkProject; mountFactoryWorkOverview; mountFactoryWorkDetail; mountFactoryWorkPlan; mountFactoryDirectionRevision; mountFactoryOptionalEffects | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Route admission and mount entrypoints — Current complete declaration/method or file span inspected at `93fb3513`; historical pins retain their scope. |
| 648–661, 723; allowanceBody; increase-effect sentence | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Delta at `93fb3513`: strict H-A POST body builder; mirrors intent.ts builder (OCR-6 drift note). |
| 1225–1228, 1270–1284, 1301, 1396–1400; capabilityGate; project_stop route | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Delta at `93fb3513`: dual intervention authority (mounter context AND backend detail) plus project_stop distinct route/read-back exclusion (OCR-8). |
| 183, 395, 548, 878, 1157; light-DOM roots; 298–323, 469–514, 1366–1384, 1467, 1503; read fan-out with mount-guard caps | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Delta at `93fb3513`: light-DOM roots vs intent.ts shadow DOM (OCR-9); per-issue/run fan-out bounded by caller lists capped at 64 (OCR-10). |
| 186–254; scope; boundScope; publishScope; applyFallback | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Delta at `deb79186` (C3e23): overview-scope publication feed with fallback; retirement included. Superseded at `31caf48a`: applyFallback renamed to applyRepositoryView with repo-only scope; see the superseding row below. |
| 225–232, 233–258, 296–299, 304–318; applyRepositoryView; load H-R boundary; readRequestOverview; readOverview | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Delta at `31caf48a` (V/a9bf): intent contexts read the authoritative H-R overview with endpoint/parser failures reaching the honest unavailable boundary; repo-only contexts use the no-intent view; composed revision/generation/issues readers, fallback and their imports removed; supersedes the marked rows above, which stay as historical record. Prior rows retain their pinned scope. |

<a id="coverage-1fa80c2a3374"></a>

## [frontend/spaces/sodaspaces-factory-work-response.ts](../../../../../frontend/spaces/sodaspaces-factory-work-response.ts)

Delta duty inspection at `93fb3513`. Work-progress read parsers (V01/V03/V04): one issue's acceptance/readiness/check/merge facts, repository queue/setup/authority summary, and durable pause/resume/stop/retry/takeover outcomes. Each parser binds its exact repository/issue/run/command identity.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 11–35; commitId; commandId; digest; text; counter; reasonCode; boundedCodes; boundedLines | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Shared parser primitives — Current complete declaration/method or file span inspected at `93fb3513`; historical pins retain their scope. |
| 36–559; IssueAcceptanceRef; IssueAcceptanceStatus; IssueBlocker; IssueReadiness; IssueCheckResult; IssueChecks; IssueMerge; WorkAttempt; WorkCorrection; WorkReview; WorkPublication; WorkClosure; WorkCapability; WorkCapabilities; RequestedRetry; WorkLinks; FactoryIssueDetail; assignmentStages; publicationStages; opEffects; opCompletions; closureStates; capabilityReasons; roleName; runRef; readinessStates; checkVerdicts; mergeStages; mergeReasons; runOutcomes; closedSet; blockerDetail; endpoint; issueBlocker; issueReadiness; decimalID; checkResult; staleFlag; issueChecks; mergeReasonText; mergeCommitText; mergeFinishedUnix; issueMerge; acceptanceRef; acceptanceStatus; nonEmpty; workOutcome; opEffect; opCompletion; workAttempt; workCorrection; workReview; publicationIdentity; historyEntries; publicationHistories; workPublication; workClosure; workCapability; workCapabilities; requestedRetry; workLinks; factoryIssueDetail | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Issue-detail shapes and parsers — Current complete declaration/method or file span inspected at `93fb3513`; historical pins retain their scope. |
| 560–602; FactoryQueueSummary; FactoryPreparationSummary; FactoryEffectiveSummary; FactoryRepositorySummary; effectiveSummary; factoryRepositorySummary | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Repository queue/setup/authority summary shapes and parsers — Current complete declaration/method or file span inspected at `93fb3513`; historical pins retain their scope. |
| 603–745; PauseRunOutcome; PauseReceipt; ResumeReceipt; StopReceipt; RetryDecision; TakeoverRecord; FactoryCommandRecord; runStopOutcome; pauseReceipt; resumeReceipt; stopReceipt; retryDecision; takeoverRecord; factoryCommandRecord | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Command-outcome shapes and parsers — Current complete declaration/method or file span inspected at `93fb3513`; historical pins retain their scope. |
| 746–808; ProjectStopResult; projectStopResult; SodaErrorDetail; errorRecord; errorText; sodaErrorDetail; PendingRuns; pendingRuns | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Project-stop, error and pending-run shapes and parsers — Current complete declaration/method or file span inspected at `93fb3513`; historical pins retain their scope. |

<a id="coverage-213e7d23931f"></a>

## [frontend/spaces/sodaspaces-factory-work-view.ts](../../../../../frontend/spaces/sodaspaces-factory-work-view.ts)

Delta duty inspection at `93fb3513`. Work-detail/intervention models plus Lit render helpers (V03/V04): delivery-state projection from evidence, per-section copy, capability gating, intervention models/outcomes and command rendering. Delivery reads evidence only; controls stay gated on current capabilities.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 23–136; nativeIssueLink; nativePullLink; DeliveryState; WorkDetailModel; blockerEndpoint; describeBlocker; mergeDelivery; checksDelivery; readinessDelivery; deliveryState; deliveryCopy | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Native links and delivery-state projection — Current complete declaration/method or file span inspected at `93fb3513`; historical pins retain their scope. |
| 137–260; acceptanceCopy; readinessCopy; checksCopy; mergeCopy; attemptCopy; publicationCopy; reviewCopy; closureCopy; requestedCopy; observedCopy; detailLinks; failedCheckContexts; detailCandidate; describeIssueDetail | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Per-section copy and detail description — Current complete declaration/method or file span inspected at `93fb3513`; historical pins retain their scope. |
| 261–322; renderWorkDetail; capabilityReasonCopy; capabilityReasonText; capabilityGate | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Detail renderer and capability gating — Current complete declaration/method or file span inspected at `93fb3513`; historical pins retain their scope. |
| 323–403; InterventionKind; InterventionResult; InterventionModel; InterventionInput; interventionCopy; describeIntervention; InterventionOutcome | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Intervention models and description — Current complete declaration/method or file span inspected at `93fb3513`; historical pins retain their scope. |
| 404–519; settledRunCount; pauseOutcomeText; stopOutcomeText; projectStopOutcomeText; RunOutcome; runOutcomeText; interventionResultText; InterventionCommands; renderInterventions | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Outcome text and intervention rendering — Current complete declaration/method or file span inspected at `93fb3513`; historical pins retain their scope. |
| 325–345; upfrontCapabilities | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Delta at `deb79186`: backend-capability mapping, permissive on unknown flags. Prior rows retain their pinned scope. |

<a id="coverage-f42521383923"></a>
<a id="frontendspacessodaspaces-factoryts-1"></a>

## [frontend/spaces/sodaspaces-factory.ts](../../../../../frontend/spaces/sodaspaces-factory.ts)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–77, 121–157, 161–177, 181–278, 283–288, 299–350; file scaffold; FactoryWatchContext; renderer; SodaFactoryWatch; static properties; private lifetime; private disposed; private generation; private retries; private cursor; private viewVisible; displayInput; closeSocket; detach; json; authorityLost; watch; screenInput; requestInput; stop; started; optionalMatches; admitWatchIds; admitWatchContext; mountFactoryWatch | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Current scaffold duty: file scaffold; 27 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 78–120, 158–160, 178–180, 279–282, 289–298; constructor; createRenderRoot; configure; disconnectedCallback; render; live; invalidate; setVisible; dispose | [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) | retained | Current method duty: constructor; 9 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 339–341; mountFactoryWatch connected getter | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Delta at `8fbed0a7` (C30c/8fbe): mounted handle exposes the view's DOM-connected state so callers can drop detached views. Prior rows retain their pinned scope. |

<a id="coverage-fc90e0fc6180"></a>

## [frontend/spaces/sodaspaces-intent-response.ts](../../../../../frontend/spaces/sodaspaces-intent-response.ts)

Delta duty inspection at `93fb3513`. Intent/proposal/challenge/exploration/allowance parsers (U01–U06): departments, ideas, tasks, proposal branches/steps/alternatives/content, challenges, milestones, recommendations, exclusions, consistency reports, explorations, planning allowances, saved requests and generation status. Strict identity binding throughout.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–474; file scaffold; Department; ProductIdea; IntentTask; ProposalBranch; ProposalStep; ProposalAlternative; ProposalContent; IntentRevision; AssumptionChallenge; Milestone; ConsolidatedProposal; ConsistencyMismatch; ConsistencyReport; Exploration; PlanningAllowance; planningAllowance; SavedRequest; decisionID; text; department; departments; idea; sourceKinds; provenanceKind; provenanceDetail; provenance; effectiveOutcomes; strings; task; intentState; intentRevision; count; savedRequest; IntentGenerationStatus; intentGeneration; intentList; branch; step; alternative; proposalAlternatives; proposalBinding; proposalContent; challenge; milestone; recommendation; excludedIDs; mismatch; consistency; exploration; references; bound; array; newDecisionID | [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) | retained | Current scaffold duty: file scaffold; 53 named units assigned here — Current complete declaration/method or file span inspected at `93fb3513`; historical pins retain their scope. |
| 277–282, 298; authorizationReference; intentGeneration authorization decode | [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) | retained | Delta at `2f382509` (U07/b619): authorization_revision decoded strictly as decisionID + ':' + 64 lowercase hex, exposing the referenced revision id; bare IDs throw with no legacy reader, absent field unchanged. Prior rows retain their pinned scope. |

<a id="coverage-810da2550068"></a>

## [frontend/spaces/sodaspaces-intent-view.ts](../../../../../frontend/spaces/sodaspaces-intent-view.ts)

Delta duty inspection at `93fb3513`. Intent creation/review/resume render helpers (U01–U06): draft entry, idea cards with votes/corrections/settlement, exploration/challenge display, generation status, loss/authorization/delivery-approach sections and the department dialog.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–458; file scaffold; IntentView; input; bounds; entryActions; terminalNote; requestState; savedRequestItem; savedRequests; describeIntent; ideaBasis; ideaChoice; ideaCorrectionField; ideaSettled; ideaVote; cardBasis; settledCorrection; explorationSettled; correctionEcho; productIdea; ideaDecided; tasksDecided; challengesAnswered; reviewReady; refreshExploration; reviewLead; plural; generationText; generationStatus; lossSummary; authorizationSection; deliveryApproach; reviewIntent; departmentDialog | [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) | retained | Current scaffold duty: file scaffold; 39 named units assigned here — Current complete declaration/method or file span inspected at `93fb3513`; historical pins retain their scope. |
| 41–42, 329, 334; raiseCalls; fundingNote | [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) | retained | Delta at `39d9ad6a` (U06/06e6): review raise input with funding note. Prior rows retain their pinned scope. |

<a id="coverage-bd6b680f8057"></a>

## [frontend/spaces/sodaspaces-intent-walkthrough.ts](../../../../../frontend/spaces/sodaspaces-intent-walkthrough.ts)

Delta duty inspection at `93fb3513`. Walkthrough presentation helpers (U05/U06): flow figures, recommendation flow, compared approaches, unknowns, consistency notes, assumption challenges, change-something and exploration review. Text equivalents accompany figures.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–219; file scaffold; short; flowBranches; flowSteps; figureRows; flowFigure; recommendationFlow; comparedApproaches; unknownList; consistencyNote; challengeIdea; challengeCorrection; challengeDecision; assumptionChallenges; changeSomething; explorationReview | [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) | retained | Current scaffold duty: file scaffold; 15 named units assigned here — Current complete declaration/method or file span inspected at `93fb3513`; historical pins retain their scope. |

<a id="coverage-68f348da74db"></a>

## [frontend/spaces/sodaspaces-intent.ts](../../../../../frontend/spaces/sodaspaces-intent.ts)

Delta duty inspection at `93fb3513`. Intent creation/review/resume element (U01–U08): private draft/list/resume, attribution/correction, exploration with independent allowance consumer and generation status, plus loss guards. Top-level entrypoints only are split here; class-internal method duties await the pending S concern review.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–934; file scaffold; IntentAPI; SodaIntent; readExploration; generationMessage; generationFailure | [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) | retained | Current scaffold duty: file scaffold; 5 top-level named units assigned here; class-internal duties pending S concern review — Current complete declaration/method or file span inspected at `93fb3513`; historical pins retain their scope. |
| 29–57, 168–204; reactive/private state fields | [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) | retained | Delta at `93fb3513`: entry/describe/explore/draft/inventory/resume state per S concern review; render delegates to intent-view helpers. |
| 229–510, 523–701, 744–907; entry/describe/explore/draft/inventory/resume/generation methods | [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) | retained | Delta at `93fb3513`: creation/review/resume method duties per S concern review; no split claimed. |
| 511–522; allowanceBody | [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) | retained | Delta at `93fb3513`: strict H-A POST body builder; mirrors factory-work.ts builder (OCR-6 drift note). |
| 702–743; decidedIdea; correctProposal | [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) | retained | Delta at `93fb3513`: human-provenance stamp plus client-side edit ideas; mounter must supply the verified actor, server rejects forged attribution (OCR-11). |
| 278, 749, 771, 784, 789; shadowRoot queries | [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) | retained | Delta at `93fb3513`: shadow-DOM element vs factory-work.ts light-DOM roots (OCR-9 consistency note). |
| 202–203, 266–267, 439–440, 592–645; raiseCalls; fundingNote; refreshAllowance; allowanceUsed; postRaiseAllowance; raise | [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) | retained | Delta at `39d9ad6a` (U06/06e6): review-only allowance top-up with fresh usage read; 412 CAS retry posts fresh, unchanged raises post nothing. Prior rows retain their pinned scope. |
| 1003–1004; generationFailure intent_review_required branch | [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) | retained | Delta at `79b4a524` (U/8682): decided-refusal wording names review refusal distinctly; no retry language, no clearing. Prior rows retain their pinned scope. |
<a id="coverage-710172173a87"></a>

## [frontend/spaces/sodaspaces-inventory-response.ts](../../../../../frontend/spaces/sodaspaces-inventory-response.ts)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–8, 100–107; file scaffold; check | [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) | retained | Current scaffold duty: file scaffold; Current method duty: check — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 9–22, 91–99, 109–120; Space; spacesResponse; check | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Current declaration duty: Space; Current declaration duty: spacesResponse; Current method duty: check — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 23–27; spaceNetwork; check | [N04](../../slices/networking.md#n04-project-enrollment-policy) | retained | Current declaration duty: spaceNetwork; Current method duty: check — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 28–60; admitSpaceTerminal; check; spaceTerminals | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | Current declaration duty: admitSpaceTerminal; Current method duty: check; Current declaration duty: spaceTerminals — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 61–63, 108; spaceItem; check | [P01](../../slices/projects.md#p01-repository-association-and-creation) | retained | Current declaration duty: spaceItem; Current method duty: check — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 64–90; check | [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) | retained | Current method duty: check — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-3f914750c1f9"></a>

## [frontend/spaces/sodaspaces-keys-response.ts](../../../../../frontend/spaces/sodaspaces-keys-response.ts)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–37, 46–57; file scaffold; SavedKey; savedKeysResponse; check; ProfileKeys; profileKeysResponse | [P04](../../slices/projects.md#p04-development-ssh-access) | retained | Current scaffold duty: file scaffold; 9 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 38–45; KeyPreview; keyPreviewResponse | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Current declaration duty: KeyPreview; Current declaration duty: keyPreviewResponse — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-6feabb7c8f55"></a>

## [frontend/spaces/sodaspaces-layout.ts](../../../../../frontend/spaces/sodaspaces-layout.ts)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–5, 35–36, 99, 109–114, 165–174, 294–304, 306–307, 322–329, 345–347; file scaffold; size; layoutLimit; putEntry; check; Area; Minimum; admitSplitAxis; admitSidebar; parseLayout | [S03](../../slices/spaces-and-terminals.md#s03-views-layout-and-restoration) | retained | Current scaffold duty: file scaffold; 15 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 6–32, 38–74, 76–92, 102–108, 115–134, 136–164, 175–177, 187–250, 264–269, 275, 281, 283–293, 305, 308–310, 337–339; LayoutEntry; Pane; Split; PaneTree; WorkspaceLayout; check; panes; paneFor; focusedPane; changePane; dropPaneTab; emptiedByRemoval; withoutTab; hideTab; sameLocator; splitPane; resizeSplit; consolidate; PaneArea; Projection; paneGap; minimumSize; projectLayout; place; admitPaneTab; admitPane; readTree | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | Current declaration duty: LayoutEntry; 39 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 33–34, 37, 75, 93–98, 100–101, 135, 178–186, 251–263, 270–274, 276–280, 282, 311–321, 330–336, 340–344; localKey; emptyLayout; selectTab; check; forgetEntry; moveTab; DividerArea; TreeParse; readLayoutEntry; serializeLayout | [P04](../../slices/projects.md#p04-development-ssh-access) | retained | Current declaration duty: localKey; 19 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-9fd0db904283"></a>

## [frontend/spaces/sodaspaces-network.ts](../../../../../frontend/spaces/sodaspaces-network.ts)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–118; file scaffold; renderNetworkSelection; networkStatus; connectedDevice; onNetworkConfirm; enableBlocked; retryBlocked; networkActions; renderNetwork | [N04](../../slices/networking.md#n04-project-enrollment-policy) | retained | Current scaffold duty: file scaffold; 9 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-ff785d6b4da6"></a>

## [frontend/spaces/sodaspaces-page.ts](../../../../../frontend/spaces/sodaspaces-page.ts)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–3, 23–37; file scaffold; measure | [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) | retained | Current scaffold duty: file scaffold; Current method duty: measure — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 4–22; mountSpacesPage | [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) | retained | Current declaration duty: mountSpacesPage — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-57d98b5e3689"></a>

## [frontend/spaces/sodaspaces-project-access.ts](../../../../../frontend/spaces/sodaspaces-project-access.ts)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–6; file scaffold | [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) | retained | Current scaffold duty: file scaffold — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 7–37, 65–76; AccessInput; removeSavedKey | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Current declaration duty: AccessInput; Current declaration duty: removeSavedKey — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 38–41, 77–188; setUseSavedKeys; selectForgejoKey; saveKey; profileKeysPageAdmitted; reviewProfileKeys; profileKeysFailed; reviewKeys; applyKeys | [P04](../../slices/projects.md#p04-development-ssh-access) | retained | Current declaration duty: setUseSavedKeys; 8 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 42–49; setDraft; setConfirmEmpty | [P01](../../slices/projects.md#p01-repository-association-and-creation) | retained | Current declaration duty: setDraft; Current declaration duty: setConfirmEmpty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 50–64; joinEnvironment | [P03](../../slices/projects.md#p03-human-membership-and-accounts) | retained | Current declaration duty: joinEnvironment — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-9070ad57b5bc"></a>

## [frontend/spaces/sodaspaces-project-connection.ts](../../../../../frontend/spaces/sodaspaces-project-connection.ts)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–10; ConnectionInput | [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) | retained | Current declaration duty: ConnectionInput — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 11–21; copyBlocked; noteCopy | [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) | retained | Current declaration duty: copyBlocked; Current declaration duty: noteCopy — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 22–30; copyConnection | [P04](../../slices/projects.md#p04-development-ssh-access) | retained | Current declaration duty: copyConnection — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-4dce2c342ce8"></a>

## [frontend/spaces/sodaspaces-project-journey-view.ts](../../../../../frontend/spaces/sodaspaces-project-journey-view.ts)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–7, 42–45, 59–101, 183–216, 225–251, 268–289, 316–332; file scaffold; journeyBlocked; renderJourney; journeyUnavailableHeading; journeyUnavailableDescription; journeyUnavailableAction; renderJourneyUnavailable; executionDenied; journeyExistingKind; journeyExistingHeading; journeyExistingDescription; journeyExistingAction; journeyExistingHelper; journeyExistingFlags; renderJourneyExisting; renderStaleReloadNote; createDisabled; configureStatusHidden; pendingTone | [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) | retained | Current scaffold duty: file scaffold; 19 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 8–41, 290–295, 333–405; JourneyViewInput; configureReady; renderCreateAction; renderConfigureFeedback; renderJourneyConfigure | [P01](../../slices/projects.md#p01-repository-association-and-creation) | retained | Current declaration duty: JourneyViewInput; 5 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 46–58, 102–177, 262–267; journeyJoinReady; joinFailedHelper; joinFailedFeedback; renderJourneyJoinFailed; joinAction; joinFeedback; joinRefreshAction; renderJourneyJoin; journeyAccountReady | [P03](../../slices/projects.md#p03-human-membership-and-accounts) | retained | Current declaration duty: journeyJoinReady; 9 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 178–182, 252–256; journeyRuntimeUnavailable; journeyIncomplete | [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) | retained | Current declaration duty: journeyRuntimeUnavailable; Current declaration duty: journeyIncomplete — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 217–224, 257–261; journeyStartButton; journeyStopped | [P05](../../slices/projects.md#p05-project-startstop) | retained | Current declaration duty: journeyStartButton; Current declaration duty: journeyStopped — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 296–315; renderConfigureNetwork; renderNetworkReviewNotice | [N04](../../slices/networking.md#n04-project-enrollment-policy) | retained | Current declaration duty: renderConfigureNetwork; Current declaration duty: renderNetworkReviewNotice — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 159–170, 379–388; joinHelper; renderJoinContext; renderAcceptedRequestNote | [P01](../../slices/projects.md#p01-repository-association-and-creation) | retained | Delta at `93fb3513`: join context and accepted-request note. Prior rows retain their pinned scope. |

<a id="coverage-b2a099997de2"></a>

## [frontend/spaces/sodaspaces-project-mutations.ts](../../../../../frontend/spaces/sodaspaces-project-mutations.ts)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–7, 38–92, 175, 204–236, 242–248; file scaffold; rejected; mutate; dispatchMutation; completeMutation; checkDeleteMutation; mutateReason; mutateErrorMessage; applyMutateError; finishMutation | [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) | retained | Current scaffold duty: file scaffold; 11 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 8–12, 93–109, 160–164, 180–190; publicKeyToken; checkMutationResult; checkAccessKeysMutation; checkSaveKeyMutation; check | [P04](../../slices/projects.md#p04-development-ssh-access) | retained | Current declaration duty: publicKeyToken; 6 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 13–33; MutationsInput | [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) | retained | Current declaration duty: MutationsInput — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 34–37; mutationObject | [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) | retained | Current declaration duty: mutationObject — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 110–114, 130–133, 165–174, 176–179, 237–241; checkCreateMutation; check; noteUnconfirmedCreation | [P01](../../slices/projects.md#p01-repository-association-and-creation) | retained | Current declaration duty: checkCreateMutation; Current method duty: check; Current declaration duty: noteUnconfirmedCreation — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 115–128, 154–159; check | [N04](../../slices/networking.md#n04-project-enrollment-policy) | retained | Current method duty: check — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 129, 191–203; checkJoinMutation; joinErrorMessage | [P03](../../slices/projects.md#p03-human-membership-and-accounts) | retained | Current declaration duty: checkJoinMutation; Current declaration duty: joinErrorMessage — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 134–138; checkLifecycleMutation | [P05](../../slices/projects.md#p05-project-startstop) | retained | Current declaration duty: checkLifecycleMutation — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 139–147; check | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Current method duty: check — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 148–153; checkTailnetMutation | [N05](../../slices/networking.md#n05-project-tailnet-selection) | retained | Current declaration duty: checkTailnetMutation — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-9e0201d9e590"></a>

## [frontend/spaces/sodaspaces-project-network.ts](../../../../../frontend/spaces/sodaspaces-project-network.ts)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–5; file scaffold | [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) | retained | Current scaffold duty: file scaffold — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 6–17, 23–26, 32–45; NetworkInput; setCreateNetworkEnabled; setNetworkConfirmed; networkChangeBlocked | [N04](../../slices/networking.md#n04-project-enrollment-policy) | retained | Current declaration duty: NetworkInput; 4 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 18–22, 27–31; setJourneyNetworkEnabled; clearNetworkReview | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Current declaration duty: setJourneyNetworkEnabled; Current declaration duty: clearNetworkReview — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 46–66; networkEnableBlocked; changeNetwork | [N05](../../slices/networking.md#n05-project-tailnet-selection) | retained | Current declaration duty: networkEnableBlocked; Current declaration duty: changeNetwork — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-be0f17bba987"></a>

## [frontend/spaces/sodaspaces-project-refresh.ts](../../../../../frontend/spaces/sodaspaces-project-refresh.ts)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–15, 90–123, 140–144, 170–181, 186–202, 259–267, 278–286, 305–321, 327–339, 475–508; file scaffold; RefreshPrior; refreshBlocked; beginRefreshRead; refresh; admitCollection; loadEnvironmentCollection; refreshEmpty; tailnetOptionsFailed; environmentStatus; refreshExisting; refreshOptionalDetails; refreshErrorStatus; refreshFailed; finishRefresh | [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) | retained | Current scaffold duty: file scaffold; 16 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 16–26, 28–30, 155–169, 182–185, 208–218; admitRepositoryPart; check; applyRepositoryLabels; refreshCreateOptions | [P01](../../slices/projects.md#p01-repository-association-and-creation) | retained | Current declaration duty: admitRepositoryPart; 6 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 27, 124–139, 268–277, 413–419; admitProjectLogin; refreshAccount; applyJoinRecovery | [P03](../../slices/projects.md#p03-human-membership-and-accounts) | retained | Current declaration duty: admitProjectLogin; 4 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 31–89, 145–154, 203–207, 296–304, 322–326; RefreshInput; check; selectedCreateProfile; applyEnvironmentStatus | [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) | retained | Current declaration duty: RefreshInput; 6 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 219–258; check; networkReviewNeeded; networkEnabledAfterRefresh; refreshTailnetOptions | [N04](../../slices/networking.md#n04-project-enrollment-policy) | retained | Current method duty: check; 4 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 287–295, 358–370; environmentNeedsAdminStart; refreshLifecycleIfAdmin | [P05](../../slices/projects.md#p05-project-startstop) | retained | Current declaration duty: environmentNeedsAdminStart; Current declaration duty: refreshLifecycleIfAdmin — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 340–357, 379–389, 401–412, 420–429; refreshSavedKeysIfStandard; refreshSavedKeys; shouldLoadConnection; admitConnectionPayload; validIPv4; refreshConnectionIfJoined | [P04](../../slices/projects.md#p04-development-ssh-access) | retained | Current declaration duty: refreshSavedKeysIfStandard; 7 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 371–378, 390–400; check | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Current method duty: check — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 430–474; shouldLoadProjectNetwork; refreshProjectNetworkIfEligible; refreshProjectNetwork; projectNetworkFailed | [N05](../../slices/networking.md#n05-project-tailnet-selection) | retained | Current declaration duty: shouldLoadProjectNetwork; 4 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-b93dbb170007"></a>

## [frontend/spaces/sodaspaces-project-request.ts](../../../../../frontend/spaces/sodaspaces-project-request.ts)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–7; file scaffold | [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) | retained | Current scaffold duty: file scaffold — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 8–33, 105–118; ObservedSummary; RequestInput; announceObserved; announceChanged | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Current declaration duty: ObservedSummary; 4 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 34–64; sodaFetchInit; sodaErrorCode | [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) | retained | Current declaration duty: sodaFetchInit; Current declaration duty: sodaErrorCode — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 65–94; requestHeaders; admitHttpFailure; api | [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) | retained | Current declaration duty: requestHeaders; 4 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 95–104, 126–143; beginEpoch; beginRead; createProject | [P01](../../slices/projects.md#p01-repository-association-and-creation) | retained | Current declaration duty: beginEpoch; Current declaration duty: beginRead; Current declaration duty: createProject — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 119–125; createTailnetBody | [N04](../../slices/networking.md#n04-project-enrollment-policy) | retained | Current declaration duty: createTailnetBody — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-2faf7244069f"></a>

## [frontend/spaces/sodaspaces-project-response.ts](../../../../../frontend/spaces/sodaspaces-project-response.ts)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–2; file scaffold | [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) | retained | Current scaffold duty: file scaffold — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 3–14, 24, 34–51, 64–72, 95–96, 128–130; CreationProfile; rockyHeadless; profileImage; creationProfile; Environment; environmentResponse; check; osImage; osObservation | [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) | retained | Current declaration duty: CreationProfile; 10 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 15–23, 25–33, 61–63, 97–99, 108–110, 112–119; check; detailResponse | [P01](../../slices/projects.md#p01-repository-association-and-creation) | retained | Current method duty: check; Current declaration duty: detailResponse — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 52–60, 73–94, 105–107, 131–135; Detail; check; OSObservation | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Current declaration duty: Detail; Current method duty: check; Current declaration duty: OSObservation — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 100–104, 111, 120–127; osReleaseIdentity; osReleaseName; osRelease | [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) | retained | Current declaration duty: osReleaseIdentity; Current declaration duty: osReleaseName; Current declaration duty: osRelease — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-5cd4d6bf4948"></a>

## [frontend/spaces/sodaspaces-project-runtime.ts](../../../../../frontend/spaces/sodaspaces-project-runtime.ts)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–6; file scaffold | [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) | retained | Current scaffold duty: file scaffold — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 7–33; RuntimeInput; setStopConfirmed | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Current declaration duty: RuntimeInput; Current declaration duty: setStopConfirmed — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 34–39, 44–71; applyOSError; inspectOS | [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) | retained | Current declaration duty: applyOSError; Current declaration duty: inspectOS — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 40–43; authLost | [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) | retained | Current declaration duty: authLost — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 72–94; changeLifecycle | [P05](../../slices/projects.md#p05-project-startstop) | retained | Current declaration duty: changeLifecycle — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-1de4d50ad9b6"></a>

## [frontend/spaces/sodaspaces-project-settings-view.ts](../../../../../frontend/spaces/sodaspaces-project-settings-view.ts)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–11, 17–23, 61–65, 155–174, 200–285; file scaffold; views; View; viewTabLabel; lifecycleCaption; renderViewTabs; renderEnvironmentView; renderAccessView | [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) | retained | Current scaffold duty: file scaffold; 8 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 12–16, 24–60; Lifecycle; renderLifecycle; confirmLifecycle | [P05](../../slices/projects.md#p05-project-startstop) | retained | Current declaration duty: Lifecycle; Current declaration duty: renderLifecycle; Current declaration duty: confirmLifecycle — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 66–128, 193–199; SettingsViewInput; renderObservedOS | [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) | retained | Current declaration duty: SettingsViewInput; Current declaration duty: renderObservedOS — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 129–139; canJoin | [P03](../../slices/projects.md#p03-human-membership-and-accounts) | retained | Current declaration duty: canJoin — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 140–154; renderRepositoryContext | [P01](../../slices/projects.md#p01-repository-association-and-creation) | retained | Current declaration duty: renderRepositoryContext — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 175–192, 298–332; renderCreateNetworkSelection; networkSummaryText; renderNetworkSummary; renderTailnetSSH; renderNetworkView | [N04](../../slices/networking.md#n04-project-enrollment-policy) | retained | Current declaration duty: renderCreateNetworkSelection; 5 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 286–297; renderForgejoKeyReview | [P04](../../slices/projects.md#p04-development-ssh-access) | retained | Current declaration duty: renderForgejoKeyReview — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 219–249; accessReadiness; renderReadinessLinks | [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) | retained | Delta at `93fb3513`: readiness-links summary into Access/Network tabs. Prior rows retain their pinned scope. |

<a id="coverage-51b813d093e0"></a>

## [frontend/spaces/sodaspaces-project-view.ts](../../../../../frontend/spaces/sodaspaces-project-view.ts)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–4; file scaffold | [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) | retained | Current scaffold duty: file scaffold — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 5–216; Connection; renderConnection; KeyPresentation; KeyCommands; keyChanges; renderKeys; renderForgejoKeys | [P04](../../slices/projects.md#p04-development-ssh-access) | retained | Current declaration duty: Connection; 7 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 217–233; renderProjectStatus | [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) | retained | Current declaration duty: renderProjectStatus — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-75264836dfef"></a>
<a id="frontendspacessodaspaces-projectts-1"></a>

## [frontend/spaces/sodaspaces-project.ts](../../../../../frontend/spaces/sodaspaces-project.ts)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–67, 76, 203–267, 277–282, 293–313, 355–366, 374–392, 478–485, 590–621, 782–813; file scaffold; ProjectContext; SodaProjectControls; canRestore; constructor; createRenderRoot; configure; disconnectedCallback; blocked; running; busyAttr; active; select; shouldFocusTab; command; setPresentation; shouldRenderJourney; sectionAriaBusy; sessionCaption; boundEnvironmentId; render; onReloadPage; reloadAdmitted; reset; invalidate; refresh; dispose; mountProjectControls | [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) | retained | Current scaffold duty: file scaffold; 28 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 68–75, 283–292; viewFromTabKey; tabKey | [P04](../../slices/projects.md#p04-development-ssh-access) | retained | Current declaration duty: viewFromTabKey; Current method duty: tabKey — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 77–195, 486–557; static properties; runtimeInput; accessInput | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Current field duty: static properties; Current method duty: runtimeInput; Current method duty: accessInput — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 196–197, 199–200, 314–351, 371–373, 622–651; private lifetime; private epoch; private mutationPending; private outcomeNeedsAttention; journeyViewInput; requestRepositoryChange; boundRepositoryId; requestInput | [P01](../../slices/projects.md#p01-repository-association-and-creation) | retained | Current field duty: private lifetime; 8 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 198, 352–354, 410–477, 577–589, 652–781; private disposed; onSelectProfile; settingsViewInput; connectionInput; refreshInput; mutationsInput | [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) | retained | Current field duty: private disposed; 6 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 201–202, 268–276, 367–370; private joinFailed; private joinNeedsCheck; canJoin; projectAccountCaption | [P03](../../slices/projects.md#p03-human-membership-and-accounts) | retained | Current field duty: private joinFailed; 4 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 393–409; journeyIdentityHidden; renderIdentity; identityPresentationReady | [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) | retained | Current method duty: journeyIdentityHidden; Current method duty: renderIdentity; Current method duty: identityPresentationReady — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 558–576; networkInput | [N04](../../slices/networking.md#n04-project-enrollment-policy) | retained | Current method duty: networkInput — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-f49b4c041ca7"></a>

## [frontend/spaces/sodaspaces-repository-picker-view.ts](../../../../../frontend/spaces/sodaspaces-repository-picker-view.ts)

Current component responsibilities split by repository selection/creation behavior and the readiness meaning carried by an existing Project; current source and workspace setup consumer inspected.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–27, 36–112; Lit import and RepositoryChoices wiring; RepositoryPickerView fields; RepositoryPickerActions callbacks; search submit and query input handlers; empty eligible repository prompt; repositoryChoice selection control and repository identity rendering; repository result fieldset and empty state; pickerLocked busy/block predicate; repository page navigation; renderRepositoryPicker search, select, Create link and Continue composition | [P01](../../slices/projects.md#p01-repository-association-and-creation) | retained | Connects the repository picker view to the typed repository-choice result consumed by the explicit Project creation flow.; 10 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 28–35; repositoryBadge existing-Project readiness label; continueCaption existing-Project readiness action label | [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) | retained | Interprets the associated Project provisioned flag as either an existing ready Project or one needing inspection; it does not authorize repository creation.; Presents an associated provisioned Project as Open project and an unprovisioned reservation as Inspect project. — current source frontend/spaces/sodaspaces-repository-picker-view.ts:28-31; response parser carries Project.provisioned separately; current source frontend/spaces/sodaspaces-repository-picker-view.ts:32-35; readiness result is supplied in RepositoryChoices |
| 4–7, 88–107; RepositoryAuthorization; plural; authorizationNote; suggestedNote | [P01](../../slices/projects.md#p01-repository-association-and-creation) | retained | Delta at `93fb3513`: reviewed-authorization and suggested-repository notes. Prior rows retain their pinned scope. |

<a id="coverage-e3079694ac03"></a>

## [frontend/spaces/sodaspaces-repository-response.ts](../../../../../frontend/spaces/sodaspaces-repository-response.ts)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–7, 9–27, 30–50; file scaffold; RepositoryChoice; RepositoryChoices; repositoryPathPart; repositoryProject; check; repositoryChoice; repositoryChoices | [P01](../../slices/projects.md#p01-repository-association-and-creation) | retained | Current scaffold duty: file scaffold; 14 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 8, 28–29, 39; RepositoryChoice.project.provisioned field and validation/return value | [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) | retained | Carries the provisioned readiness state for an already associated project; repository identity, picker eligibility, and safe owner/name choices remain P01. — Current interface field and repositoryProject validation/returned value are consumed as provisioning readiness; project ID and repository selection remain separate picker duties. |

<a id="coverage-6bf6f9d85201"></a>

## [frontend/spaces/sodaspaces-session-navigation-view.ts](../../../../../frontend/spaces/sodaspaces-session-navigation-view.ts)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–4, 18–48; file scaffold; renderSessionTab | [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) | retained | Current scaffold duty: file scaffold; Current declaration duty: renderSessionTab — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 5–17; SessionTab | [P04](../../slices/projects.md#p04-development-ssh-access) | retained | Current declaration duty: SessionTab — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 49–59; NavigationSession | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | Current declaration duty: NavigationSession — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 60–123; renderProjectNavigation | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Current declaration duty: renderProjectNavigation — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-e42ed6038ad4"></a>

## [frontend/spaces/sodaspaces-terminal-actions.ts](../../../../../frontend/spaces/sodaspaces-terminal-actions.ts)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–123, 132–142; file scaffold; ActionsInput; live; publish; abortActionIfStale; closeSocket; detach; json; authorityLost; actionCurrent; finishAction | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | Current scaffold duty: file scaffold; 15 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 124–131, 143–176; confirmNativeEnd; endUnconfirmed; endBlocked; endTerminal | [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) | retained | Current declaration duty: confirmNativeEnd; 4 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-a042ed0aae28"></a>

## [frontend/spaces/sodaspaces-terminal-attachment.ts](../../../../../frontend/spaces/sodaspaces-terminal-attachment.ts)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–50, 78–124, 131–146, 177–211, 229–248, 252–382; file scaffold; TerminalState; AttachmentInput; blockedConnect; hiddenConnect; canStartConnect; detachUnready; hiddenWhileAttaching; cancelIfHidden; inspectExisting; attachPayload; onPeerOpen; acceptReady; acceptOutput; dispatchFrame; onPeerMessage; onPeerClosed; bindPeer; attachPeer; connect | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | Current scaffold duty: file scaffold; 21 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 51–77, 125–130, 147–176, 212–228, 249–251; send; observe; reserveCreate; refreshAttachedMetadata | [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) | retained | Current declaration duty: send; 6 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-57dda7c41fa5"></a>

## [frontend/spaces/sodaspaces-terminal-dialog-view.ts](../../../../../frontend/spaces/sodaspaces-terminal-dialog-view.ts)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–102; file scaffold; renderRename; CreationPresentation; renderCreation | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | Current scaffold duty: file scaffold; 4 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-cff1722ab543"></a>

## [frontend/spaces/sodaspaces-terminal-response.ts](../../../../../frontend/spaces/sodaspaces-terminal-response.ts)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–65; file scaffold; terminalID; TerminalIdentity; TerminalMetadata; admitTerminalBinding; check; admitTerminalName; admitTerminalClock; knownTerminalState; admitTerminalState; terminalMetadata; terminalResponse | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | Current scaffold duty: file scaffold; 16 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-5d55ba6553a9"></a>

## [frontend/spaces/sodaspaces-terminal-screen.ts](../../../../../frontend/spaces/sodaspaces-terminal-screen.ts)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–45, 59–74, 126–160, 179–216, 221–244; file scaffold; TerminalFit; ScreenInput; clearScreen; resize; awaitScreen; requiredToken; terminalTheme; interceptControlFocus; openTerminal; shouldFocusScreen; screenReady | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | Current scaffold duty: file scaffold; 12 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 46–58, 75–125, 217–220; canFit; geometryScreen; paneMinimum; publishMinimum; measureMinimum; measureIfCurrent | [S03](../../slices/spaces-and-terminals.md#s03-views-layout-and-restoration) | retained | Current declaration duty: canFit; 6 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 161–178; sendInput | [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) | retained | Current declaration duty: sendInput — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-3b2eeb224758"></a>

## [frontend/spaces/sodaspaces-terminal-view.ts](../../../../../frontend/spaces/sodaspaces-terminal-view.ts)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–67, 71–109, 137–148, 188–230; file scaffold; TerminalPresentation; TerminalCommands; TerminalViewInput; viewDisabled; canConnectView; contextLabels; terminalPresentation; menuKey; connectFromControls; terminalCommands; renderTerminal | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | Current scaffold duty: file scaffold; 12 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 68–70, 110–114, 120–136, 149–187; canEndView; endConfirmedTerminal; confirmEnd; cancelEnd; actions; confirmation | [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) | retained | Current declaration duty: canEndView; 6 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 115–119; workspaceCommand | [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) | retained | Current declaration duty: workspaceCommand — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-df201745844a"></a>
<a id="frontendspacessodaspaces-terminalts-1"></a>

## [frontend/spaces/sodaspaces-terminal.ts](../../../../../frontend/spaces/sodaspaces-terminal.ts)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–29, 34–41, 62–129, 166–224, 234–434, 443–451, 481–486; file scaffold; TerminalView; TerminalContext; SodaTerminal; static properties; private createName; private managedEnded; private lifetime; private disposed; private generation; private retries; private viewVisible; private lastSize; constructor; terminalViewInput; closeMenu; invalidate; actionsInput; screenInput; attachmentInput; setVisible; focus; started; restore; disconnect; mountTerminal | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | Current scaffold duty: file scaffold; 26 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 30–33, 42–61, 130–132, 163–165, 225–233, 435–442, 473–480, 488–508; Renderer; TerminalLocator; renderer; createRenderRoot; render; remember; setName; open; admitLocator | [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) | retained | Current declaration duty: Renderer; 10 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 133–162, 452–472, 487; configure; disconnectedCallback; dispose; admitMountContext | [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) | retained | Current method duty: configure; 5 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-9359a996d016"></a>

## [frontend/spaces/sodaspaces-workspace-drawer.ts](../../../../../frontend/spaces/sodaspaces-workspace-drawer.ts)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–5, 35–40, 56–75, 85–106; file scaffold; drawerOpenBlocked; drawerStillCurrent; assignDrawer; openDrawerNavigation; openInDrawer | [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) | retained | Current scaffold duty: file scaffold; 7 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 6–23, 41–55; DrawerInput; drawerEntry | [S03](../../slices/spaces-and-terminals.md#s03-views-layout-and-restoration) | retained | Current declaration duty: DrawerInput; Current declaration duty: drawerEntry — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 24–34; drawerRepositoryPart | [P01](../../slices/projects.md#p01-repository-association-and-creation) | retained | Current declaration duty: drawerRepositoryPart — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 76–84; check | [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) | retained | Current method duty: check — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-d83cbe6c88ee"></a>

## [frontend/spaces/sodaspaces-workspace-factory.ts](../../../../../frontend/spaces/sodaspaces-workspace-factory.ts)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–9; file scaffold | [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) | retained | Current scaffold duty: file scaffold — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 10–182; factoryWatchLimit; FactoryAdmission; factoryWatching; factoryRowDisabled; FactoryWatchMutation; toggleFactoryWatch; watchRun; unwatchRun; FactorySectionInput; factorySection; FactoryWatchContextInput; factoryWatchContext; FactoryCommandInput; onFactoryCommand; FactoryDisplayInput; displayFactoryWatch; displayFactoryWatches | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Current declaration duty: factoryWatchLimit; 17 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 157–162, 168; dropDetachedWatchView; displayFactoryWatch detached drop | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Delta at `8fbed0a7` (C30c/8fbe): a permanently-disconnected read view is disposed and dropped so the same run's view is recreated on next display; connected views are retained untouched. No continuous renderer or socket across filter DOM removal is claimed. Prior rows retain their pinned scope. |

<a id="coverage-e974c7ae11fb"></a>

## [frontend/spaces/sodaspaces-workspace-focus.ts](../../../../../frontend/spaces/sodaspaces-workspace-focus.ts)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–6, 67–74; WorkspaceKeyInput; RestoreFocusInput | [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) | retained | Current declaration duty: WorkspaceKeyInput; Current declaration duty: RestoreFocusInput — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 7–56, 62–66, 75–97; workspaceKey; workspaceClick; menuFocusOut; closeMenus; rememberFocus; restoreFocus | [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) | retained | Current declaration duty: workspaceKey; 7 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 57–61; RememberFocusInput | [P03](../../slices/projects.md#p03-human-membership-and-accounts) | retained | Current declaration duty: RememberFocusInput — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-ccac8cc6644c"></a>

## [frontend/spaces/sodaspaces-workspace-inventory.ts](../../../../../frontend/spaces/sodaspaces-workspace-inventory.ts)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–10, 34–42, 61–66, 203–217, 230–235, 243–250, 255–265, 277–280, 348–354; file scaffold; SpacesCollection; live; ReadSpacesInput; api; applySlotMetadata; existingSlotMetadata; restoreIfNeeded; pageJourneyActive; leaveJourneyManagement; syncPageJourney; renderMoreProjects | [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) | retained | Current scaffold duty: file scaffold; 13 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 11–27; sodaWorkspaceInit | [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) | retained | Current declaration duty: sodaWorkspaceInit — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 28–33, 52–60, 67–141; LiveReading; ApiInput; check; InventoryInput; reading; apiInput | [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) | retained | Current declaration duty: LiveReading; 6 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 43–51, 142–188, 218–229, 251–254, 266–275, 281–340; readSpacesResponse; refreshBlocked; beginRefreshRead; collectionStatus; applySpacesCollection; appendSpacesCollection; refreshSlots; refreshJourneyManagement; refreshFailed; finishRefreshSuccess; refreshLive; refresh; loadMore | [S01](../../slices/spaces-and-terminals.md#s01-authorized-inventory) | retained | Current declaration duty: readSpacesResponse; 15 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 189–202; slotLost; invalidateSlot | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | Current declaration duty: slotLost; Current declaration duty: invalidateSlot — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 236–242, 276, 341–347; clearConfigureAfterCreate; MoreProjectsInput | [P01](../../slices/projects.md#p01-repository-association-and-creation) | retained | Current declaration duty: clearConfigureAfterCreate; Current method duty: clearConfigureAfterCreate; Current declaration duty: MoreProjectsInput — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 290–294; refreshFailedStatus | [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) | retained | Delta at `93fb3513`: failed-refresh status naming. Prior rows retain their pinned scope. |

<a id="coverage-58e3f214277f"></a>

## [frontend/spaces/sodaspaces-workspace-layout.ts](../../../../../frontend/spaces/sodaspaces-workspace-layout.ts)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–17, 64–86; file scaffold; compact; projection | [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) | retained | Current scaffold duty: file scaffold; Current declaration duty: compact; Current declaration duty: projection — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 18–51, 60–63; LayoutInput; MinimumInput; minimumInput | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | Current declaration duty: LayoutInput; Current declaration duty: MinimumInput; Current declaration duty: minimumInput — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 52–59, 87–181, 190–195, 202–239, 252–299; paneMinimum; toggleSidebar; persist; loadLayout; arrange; focusPane; splitBlocked; emptyPaneMinimum; splitFits; canSplit; split; move; toggleMaximizedPane; consolidatePanes; dragDivider; adjustDivider; keyDivider; renderPaneDivider; resizeSidebar; sidebarKey; setSidebar | [S03](../../slices/spaces-and-terminals.md#s03-views-layout-and-restoration) | retained | Current declaration duty: paneMinimum; 28 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 182–189, 240–251; MaximizedInput; dividerPointerMove; dividerKey | [P04](../../slices/projects.md#p04-development-ssh-access) | retained | Current declaration duty: MaximizedInput; Current declaration duty: dividerPointerMove; Current declaration duty: dividerKey — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 196–201; ConsolidateInput | [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) | retained | Current declaration duty: ConsolidateInput — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-8916726a775c"></a>

## [frontend/spaces/sodaspaces-workspace-measurement.ts](../../../../../frontend/spaces/sodaspaces-workspace-measurement.ts)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–46; file scaffold; WorkspaceMeasurement; private retired; constructor; hostUpdated; hostDisconnected; retire | [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) | retained | Current scaffold duty: file scaffold; 7 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-dcfd0e20047d"></a>

## [frontend/spaces/sodaspaces-workspace-navigation.ts](../../../../../frontend/spaces/sodaspaces-workspace-navigation.ts)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–9, 109–181; file scaffold; unnamedRow; rowName; rowMatchesQuery; rowTerminalId; rowDescription; rowDisabled; rowNavItem; pageProjectNav; projectSubtitle | [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) | retained | Current scaffold duty: file scaffold; 10 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 10–17, 182–199; NavReading; ProjectRowsInput | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | Current declaration duty: NavReading; Current declaration duty: ProjectRowsInput — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 18–47, 58–108, 200–216; rowAttentionBlocked; rowAttention; attentionRows; filteredSpaces; rows; projectRows | [S01](../../slices/spaces-and-terminals.md#s01-authorized-inventory) | retained | Current declaration duty: rowAttentionBlocked; 6 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 48–57; FilteredSpacesInput | [S03](../../slices/spaces-and-terminals.md#s03-views-layout-and-restoration) | retained | Current declaration duty: FilteredSpacesInput — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 49–133; DiscoveryEmptyInput; discoveryEmptyMessage; recentRowKeys; splitAttentionRows; RequestGroups; byRecency; groupSavedRequests; requestRepositoryLabel; RequestGroupsInput; renderRequestGroups | [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) | retained | Delta at `93fb3513`: discovery empty/recent/attention/request-group rows. Prior rows retain their pinned scope. |

<a id="coverage-3ab1ba4c5d50"></a>

## [frontend/spaces/sodaspaces-workspace-pane-view.ts](../../../../../frontend/spaces/sodaspaces-workspace-pane-view.ts)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–12, 126–129, 199–206, 225–235, 268–303; file scaffold; sessionRow; showMoveMenu; renderMoveMenu; dropEdgeStyle; onDropEdgeOver; onDropEdgeDrop; renderDropEdges | [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) | retained | Current scaffold duty: file scaffold; 8 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 13–43; PaneChromeInput | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | Current declaration duty: PaneChromeInput — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 44–125, 130–198, 207–224, 236–267, 304–339; showPaneSwitcher; onFocusPaneChange; renderPaneSwitcher; renderPaneLayoutExtras; renderPaneSplitHelp; renderPaneMenu; paneActions; paneTabKeys; paneSession; paneAriaOwns; onTabListDragOver; onTabListDrop; onTabDragStart; onTabDragEnd; onTabDrop; sessionTabProps; renderFocusedPaneActions; filterOverflowTabs; overflowTabLabel; renderTabOverflow; moveTargetName; moveToPane; moveBeforeTab; beforeTabName; emptyPaneAttached; renderEmptyPane; paneChrome | [S03](../../slices/spaces-and-terminals.md#s03-views-layout-and-restoration) | retained | Current declaration duty: showPaneSwitcher; 27 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-fd3cb6c0a78f"></a>

## [frontend/spaces/sodaspaces-workspace-setup.ts](../../../../../frontend/spaces/sodaspaces-workspace-setup.ts)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–7, 39–46, 84, 92, 95, 100–105, 107–109, 241–272; file scaffold; searchAdmitted; SetupActions; setSetupReturn; setManagementMode; hasSelectedSpace; isStale; isActiveSurface; selectProject; rememberFocus; restoreFocus; api; searchCursor; searchRepositories | [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) | retained | Current scaffold duty: file scaffold; 14 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 8–38, 47–52, 93, 96, 106, 115–181, 189–194, 199–201, 205–240, 273–279; setupIntroKind; setupIntroHeading; setupIntroDescription; setupIntroAction; setupIntroHelper; repositorySearchPath; setProject; repositoryCursors; showManagement; renderSetupBody; onRepositoryQuery; renderSetup; beginSetup; changeRepository; cancelSetup; applyRepositorySearch; failRepositorySearch; configureProject | [P01](../../slices/projects.md#p01-repository-association-and-creation) | retained | Current declaration duty: setupIntroKind; 18 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 53–58, 94, 182–188, 195–198, 202–204; SetupReturn; setView; focusSetupPanel | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | Current declaration duty: SetupReturn; 5 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 59–83, 85–91, 97–99, 110–114; SetupReading; setSetup; setRepositoryQuery; setRepositoryChoice; setRepositoryResult; setRepositoryCursors; setRepositoryError; setRepositoryBusy; currentRepositoryRequest; admitRepositoryRequest; currentRepositoryQuery; SetupHost | [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) | retained | Current declaration duty: SetupReading; 12 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 13–75, 197–205, 303–368; requestIDPattern; requestIdFromLocation; requestRouteFor; newRouteFor; intentIdentity; inventoryIntents; reviewedAuthorization; reviewedIntent; setupOverviewContext; focusSetupIntent; beginSetup; resumeSetup; chooseIntentRepository; backToReview; inspectCommittedRequest; leaveSetup | [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) | retained | Delta at `93fb3513`: request/new routes, intent identity, reviewed summary, Back/inspect/leave semantics. Prior rows retain their pinned scope. |
| 25, 90–155; stripRoutePrefix; resumedCommittedRequest; resumedRepositoryName; readResumedRepository; resumeCommittedAdmitted; resumeCommittedOverview | [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) | retained | Delta at `deb79186` (U76/Ce703): committed-request resume reads with durable repository. Prior rows retain their pinned scope. |
| 309–412; scopeCapabilityKinds; scopeEntryCapabilities; scopeIssueEntry; scopeIssueList; overviewScopeDetail; sameScope; applyOverviewScope; WorkChildScope; childScopeFromReading; detailRunsForIssue; renderWorkDetail; renderWorkChildren | [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) | retained | Delta at `deb79186` (C3e23): scope-feed consumption with child derivation and intent-branch children render. Prior rows retain their pinned scope. |
| 293, 407–453; SetupActions.requestFactoryWatch; FactoryWatchRequest; watchRequestRecord; nonEmptyRequestPart; factoryWatchRequestDetail; resolveWatchTarget; applyWatchRequest | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Delta at `a835c02d` (C46ee): watch-request payload admission with exact live-run target resolution, malformed rejection and observe-only dispatch. Prior rows retain their pinned scope. |
| 55–58; isNewRouteLocation | [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) | retained | Delta at `59b18572` (C03/2945): exact fresh-entry route match under the admitted installation prefix. Prior rows retain their pinned scope. |

<a id="coverage-9aa49bcb876a"></a>

## [frontend/spaces/sodaspaces-workspace-shell-view.ts](../../../../../frontend/spaces/sodaspaces-workspace-shell-view.ts)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–27, 35–65, 93–118, 128–179, 183–185; file scaffold; workspaceBlocked; navigationVisible; busyAttr; workspaceClasses; hideListStatus; retryLoading; renderStatusBanners; hideWorkspaceBody; workspaceBodyClass; workspaceBodyStyle; hideNavigation; hideCanvas; inventoryHelper; renderInventoryRecovery; renderFirstTerminalIntro; hideManagement | [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) | retained | Current scaffold duty: file scaffold; 17 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 28–34, 66–92; renderSetupHeading; setupBackDisabled; renderSetupTopbar | [P01](../../slices/projects.md#p01-repository-association-and-creation) | retained | Current declaration duty: renderSetupHeading; Current declaration duty: setupBackDisabled; Current declaration duty: renderSetupTopbar — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 119–127, 180–182; hideSidebarDivider; hidePaneChrome | [S03](../../slices/spaces-and-terminals.md#s03-views-layout-and-restoration) | retained | Current declaration duty: hideSidebarDivider; Current declaration duty: hidePaneChrome — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-237ce867e7cd"></a>

## [frontend/spaces/sodaspaces-workspace-terminal-hosts.ts](../../../../../frontend/spaces/sodaspaces-workspace-terminal-hosts.ts)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–10; file scaffold | [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) | retained | Current scaffold duty: file scaffold — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 11–290; HostsInput; visibleSlot; identity; check; onTerminalLocator; commitTerminalLocator; observationAdmitted; staleObservation; noteUnread; applyTerminalObservation; onTerminalObservation; onTerminalMetadata; geometrySize; onTerminalGeometry; onTerminalFocusIn; addSlot; slotHidden; applySlotGeometry; displaySlot; displaySlots | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | Current declaration duty: HostsInput; 22 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-fa6687c92f7d"></a>

## [frontend/spaces/sodaspaces-workspace-terminals.ts](../../../../../frontend/spaces/sodaspaces-workspace-terminals.ts)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–10; file scaffold | [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) | retained | Current scaffold duty: file scaffold — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 11–527; TerminalsInput; canRestorePane; restoreSelectedPane; restoreLocators; slotName; confirmedEnd; queueMicrotask; openSavedBlocked; openSavedSpace; applyOpenLayout; shouldFocusOpened; restoreOpenedSlot; openSaved; CreationFormInput; creationEligible; creationExplanation; creationContext; creationDisabled; onCreationEnvironment; onCreationName; cancelCreation; creationForm; NewTerminalInput; defaultTerminalName; creationSpace; pickCreationSpace; pageBlocksNewTerminal; newTerminalBlocked; focusCreationDialog; newTerminal; CreateInput; createAdmitted; createSpaceReady; openCreatedSlot; createTerminal; openExistingBlocked; existingOrNewEntry; openExisting; RenameInput; renameAdmitted; applyRenamedMetadata; check; rename | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | Current declaration duty: TerminalsInput; 44 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-9649a5dd74b0"></a>

## [frontend/spaces/sodaspaces-workspace-toolbar-view.ts](../../../../../frontend/spaces/sodaspaces-workspace-toolbar-view.ts)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–102, 107–126; file scaffold; sessionsButtonHidden; sessionsButtonLabel; toolbarProjectTitle; renderToolbarProject; hideNewTerminal; disableNewTerminal; newTerminalButtonClass; newTerminalButtonLabel; renderProjectSettingsButton; hideBackButton; backButtonLabel; renderBackButton; renderOpenInDrawerOption; nativeManagementTarget; renderNativeManagementOption | [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) | retained | Current scaffold duty: file scaffold; 16 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 103–106; renderToggleSidebarOption | [S03](../../slices/spaces-and-terminals.md#s03-views-layout-and-restoration) | retained | Current declaration duty: renderToggleSidebarOption — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 127–129; renderSpacesLink | [S01](../../slices/spaces-and-terminals.md#s01-authorized-inventory) | retained | Current declaration duty: renderSpacesLink — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-dbcd6860f0bd"></a>

## [frontend/spaces/sodaspaces-workspace-types.ts](../../../../../frontend/spaces/sodaspaces-workspace-types.ts)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–18; file scaffold; WorkspaceContext | [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) | retained | Current scaffold duty: file scaffold; Current declaration duty: WorkspaceContext — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 19–32, 38–42; Slot; PaneSession | [S03](../../slices/spaces-and-terminals.md#s03-views-layout-and-restoration) | retained | Current declaration duty: Slot; Current declaration duty: PaneSession — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 33–37; Row | [S01](../../slices/spaces-and-terminals.md#s01-authorized-inventory) | retained | Current declaration duty: Row — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 43–52; FactoryWatch | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Current declaration duty: FactoryWatch — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 53–57; Creation | [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) | retained | Current declaration duty: Creation — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-1e87c626d0ae"></a>
<a id="frontendspacessodaspaces-workspace-viewts-1"></a>

## [frontend/spaces/sodaspaces-workspace-view.ts](../../../../../frontend/spaces/sodaspaces-workspace-view.ts)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–88; file scaffold; renderWelcome; renderWorkspaceIntro; renderWelcomeSteps; renderMenu | [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) | retained | Current scaffold duty: file scaffold; 5 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-b7c59695b1e9"></a>
<a id="frontendspacessodaspaces-workspacets-1"></a>

## [frontend/spaces/sodaspaces-workspace.ts](../../../../../frontend/spaces/sodaspaces-workspace.ts)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–109, 111–114, 189–190, 219–220, 223–226, 228, 243–267, 271–330, 342–381, 392–406, 410–416, 428–437, 463, 465–475, 477, 482, 486–505, 512–535, 543–558, 575–579, 634–871, 875–900, 954–975, 1002–1019, 1058–1125, 1251–1270, 1277–1299, 1345–1347, 1391–1393, 1401–1429, 1469–1478, 1555–1558, 1596–1647; file scaffold; validName; private reconnectRequired; private observedAt; private renaming; private epoch; private storageLoaded; private available; private surfaceVisible; private lifetime; private lastMinimum; constructor; createRenderRoot; activeSurface; selectedSpace; welcomeScreen; pageReadyForFirstTerminal; spaceReadyForFirstTerminal; projectRunnable; firstTerminal; inventoryRecovery; workspaceIntro; projectState; projectStatus; selected; compact; projection; configure; projectEventFromHost; pageProjectEvent; onProjectObserved; onProjectOperation; disconnectedCallback; updated; publishMinimum; measuredCell; private readonly onWorkspaceClick; private readonly onSearchInput; private readonly onThisPageChange; private readonly onShowAllAttention; private readonly onShowAttentionOnly; private readonly onRefreshClick; private readonly onBeginSetup; private readonly onShowSessions; private readonly onCancelSetup; private readonly onCapturePointer; navigationEscape; setSearchFromEvent; setThisPageFromEvent; renderNativeToolbar; renderPageToolbar; navAriaLabel; renderProjectsHeading; hideTerminalSearch; renderThisPageFilter; renderCreateAnotherProject; emptyFilterMessage; renderWelcomeFooter; renderNavigation; renderCanvas; renderWorkspaceFrame; render; connectURL; renderToolbarMenu; renderToolbar; setupReading; selectProject; markViewed; projectName; metadataRowName; draftRowName; selectRow; creationInput; closeMenus; rememberFocus; restoreFocus; showSessions; back; live; api; drawerInput; openSaved; commandAdmitted; onTerminalCommand; display; rectangle; capture; releasePointer; defaultManagementMode; setVisible; invalidate; canRestore; dispose | [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) | retained | Current scaffold duty: file scaffold; 93 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 110, 447–451; TerminalFactory; private measure | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Current declaration duty: TerminalFactory; Current field duty: private measure — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 115, 382–391, 536–542, 559–574, 976–978, 988–1001, 1271–1273, 1559–1566, 1648–1652; SodaSpaces; pollAttention; shouldRefreshAttention; hideAttentionFilters; renderEmptySpaces; rowAttentionReady; nextAttention; refresh; refreshMountedProject; mountSodaspaces | [S01](../../slices/spaces-and-terminals.md#s01-authorized-inventory) | retained | Current declaration duty: SodaSpaces; 10 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 116–188, 197–218, 221, 478–481, 901–953, 1163–1250, 1348–1390, 1503–1534, 1653–1677; static properties; private projects; private disposed; private readonly onSetupBack; setupActions; inventoryInput; hostsInput; createInput; check | [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) | retained | Current field duty: static properties; 9 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 191–196, 238–242, 331–333, 476, 592–613, 979–987, 1020–1057, 1126–1152, 1300–1344, 1430–1468; private now; private measurement; minimumInput; private readonly onNewTerminal; renameInput; navReading; paneViewInput; newTerminalInput; terminalsInput; layoutInput | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | Current field duty: private now; 10 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 222, 452–462, 580–591, 614–633, 1394–1400; private restored; private readonly onWorkspaceKey; onRenameDraft; onRenameCancel; onRenameSave; renameDisabled; renderRenameDialog; renderCreationDialog; beginRename | [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) | retained | Current field duty: private restored; 9 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 227, 464, 485; private storageKey; private readonly onNavKey; private readonly onSidebarKey | [P04](../../slices/projects.md#p04-development-ssh-access) | retained | Current field duty: private storageKey; Current field duty: private readonly onNavKey; Current field duty: private readonly onSidebarKey — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 229–237, 334–341, 417–427, 438–446, 483, 506–511, 1274–1276, 1479–1502; private canvasSize; private workspaceWidth; private cell; locator; tabHeight; geometryChanged; applyGeometry; recordCanvasGeometry; private readonly onResizeSidebar; renderDrawerProjectionTabs; persist; tabKey; hideSlot | [S03](../../slices/spaces-and-terminals.md#s03-views-layout-and-restoration) | retained | Current field duty: private canvasSize; 13 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 268–270, 407–409, 872–874, 1153–1162, 1535–1554, 1567–1595; setupScreen; onProjectChangeRepository; repositoryPrefix; moreProjectsInput; managementAdmitted; mountProject; focusConfigureIfNeeded; showManagement; presentManagement | [P01](../../slices/projects.md#p01-repository-association-and-creation) | retained | Current method duty: setupScreen; 9 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 484; private readonly onReleasePointer | [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) | retained | Current field duty: private readonly onReleasePointer — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 145, 243, 307, 936–975, 1049–1056; overviewScope; repositoryPrefix; requestIdFromLocation; committedSpace; setOverviewScope | [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) | retained | Delta at `deb79186` (C3e23): overview-scope state with reading/action plumbing plus explicit-prefix request routes. Prior rows retain their pinned scope. |
| 57–58, 69, 283, 1052, 1055–1073; requestFactoryWatch; overviewScope reset | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Delta at `a835c02d` (C46ee): reactive scope reset with detail-Watch resolution into an observe-only run watch. Prior rows retain their pinned scope. |
| 622, 715, 1110–1121; navigationSpaces | [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) | retained | Delta at `8fbed0a7`: filtered-navigation extraction shared by the empty state and row render. Prior rows retain their pinned scope. |
| 69, 1034–1064, 1122–1124; requestFactoryWatch reveal; navigationShows | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Delta at `8fbed0a7` (C30c/8fbe): detail Watch dedups via factoryWatching, unhides the exact watch, clears only obstructing filters, cancels setup and restores hidden navigation; supersedes the dispatch-only description above, which stays as historical record. Prior rows retain their pinned scope. |
| 275, 397–398, 406–407, 449, 452–456, 490, 493–511; pendingNewRoute; armNewRoute; activateNewRoute | [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) | retained | Delta at `59b18572` (C03/2945, 9a4f): one-shot fresh-entry activation on direct new-route load/traversal/forward; armed in configure/popstate/present and consumed with a current-route recheck so Back/cancel never refire. Prior rows retain their pinned scope. |
| 501, 514–535; freshEntryFromSetup | [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) | retained | Delta at `39d9ad6a` (C/7657): native New while resuming drops the loaded identity for a blank entry; fresh/busy/committed untouched, return target preserved. Prior rows retain their pinned scope. |
| 506, 511–513, 514–532; newRouteCurrent; freshEntryFromSetup guards | [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) | retained | Delta at `71f79938` (C/63d2, 71f7): accepted-overview New clears the committed flag for a blank apart-saving entry; idle Configure uses the stale/canRestore guard with the shared newRouteCurrent helper; supersedes the guard description above, which stays as historical record. Prior rows retain their pinned scope. |
