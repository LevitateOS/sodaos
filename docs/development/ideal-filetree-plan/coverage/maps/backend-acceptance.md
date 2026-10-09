# Backend acceptance

Current path navigation reconciled at `45ebf4c4` (2026-10-09); historical symbol/body selectors remain pinned to `519b76bd` unless a narrow current selector is explicitly stated.

[Responsibility map index](README.md) · [Coverage scope](../README.md).

Current responsibility accounting at `519b76bd` (2026-10-07).
One slice owns each named duty; disjoint complete symbols may share an owner.
Compound fields/clauses may share a physical line with distinct selectors.
Disposition concerns the duty, not source validity or installed qualification.

<a id="coverage-7d60accc4a2d"></a>

## Former source `internal/acceptance/command.go` (body snapshot `519b76bd`)

Historical locator and selector/body evidence are pinned to `519b76bd`; these selectors do not describe a current path or current body.

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–244; file scaffold; Command; Result; validCommandLabel; commandEvidenceWriters; closeCommandWriters; waitCommandProcess; processStillOwned; commandCaptures; Close; closeWritersWhenDone; pipeFailureAfterSuccess; commandExitOutcome; redactCommandResult; Execute; nopCloser; Quote; Remote; validSSHPort; validSSHUser; validSSHHost; regularUnwritableFile; trustedKnownHosts; Args; WaitReady | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Current scaffold duty: file scaffold; 27 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-20cae505bccf"></a>

## [internal/acceptance/developer_access.go](../../../../../internal/acceptance/developer_access.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–310; file scaffold; developerAccessSources; developerAccessFiles; developerAccessDigest; DeveloperAccessUsage; ptyProbeInput; identityProbeCommand; accessRequest; accessRequestUser; accessBrowser; accessUserResult; accessResults; requestString; checkRequestKeys; decodeAccessRequest; requestStrings; parseAccessSubnet; prefixBroadcast; loadAccessRequest; RunDeveloperAccess; runDeveloperAccess | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Current scaffold duty: file scaffold; 21 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-2e7553387d99"></a>

## [internal/acceptance/developer_access_journey_test.go](../../../../../internal/acceptance/developer_access_journey_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–144; file scaffold; stubAccessTransport; TestRunDeveloperAccessEmulatedOrchestration | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | The stubbed SSH/keygen/SCP/SFTP journey checks orchestration, semantic result fields and recorded artifact identity; it is explicitly emulated and proves no native SSH behavior. — Current source declaration/method inspected; no native qualification claim |

<a id="coverage-559d5ea5aec1"></a>

## [internal/acceptance/developer_access_request_test.go](../../../../../internal/acceptance/developer_access_request_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–112; file scaffold; writeAccessRequest; writeAccessSupport; TestLoadAccessRequest | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Current scaffold duty: file scaffold; 4 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-c55eb44e0b18"></a>

## [internal/acceptance/developer_access_session.go](../../../../../internal/acceptance/developer_access_session.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–260; file scaffold; setupDeveloperOutput; accessChecked; fetchAccessFingerprint; writeAccessPayload; accessSSHOptions; checkAccessConnection; parseAccessMemberIP; checkAccessIdentity; parseAccessUID; splitAccessSign; stripAccessUnderscores; parseAccessDigits; checkAccessUIDMap; checkAccessPTY; checkAccessSession | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Current scaffold duty: file scaffold; 16 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-f470fa6573df"></a>
<a id="internalacceptancedeveloper_access_testgo-1"></a>

## [internal/acceptance/developer_access_test.go](../../../../../internal/acceptance/developer_access_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–212; file scaffold; testFingerprint; testHostKey; TestParseAccessSubnet; writeTestKey; TestValidateAccessUser; TestQuoteAccessPath; TestParseAccessUID; TestAccessSSHOptions; accessFixture; preserveUmask; TestRunDeveloperAccessValidation | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Current scaffold duty: file scaffold; 12 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-8d7e6bf76e56"></a>

## [internal/acceptance/developer_access_transfer.go](../../../../../internal/acceptance/developer_access_transfer.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–172; file scaffold; checkAccessSudo; quoteAccessPath; checkAccessTransfer; checkAccessSFTP; probeAccessUser; checkCrossUserDenial; runAccessChecks; writeAccessResults | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Current scaffold duty: file scaffold; 9 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-0136753b3309"></a>

## [internal/acceptance/developer_access_users.go](../../../../../internal/acceptance/developer_access_users.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–169; file scaffold; validateAccessIDs; loadAccessBrowser; loadAccessHostKey; validateAccessUser; checkAccessUserKeys; checkAccessUserLogin; checkAccessUserAdmin; validateAccessUsers | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Current scaffold duty: file scaffold; 9 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-a8e5246b36b5"></a>
<a id="internalacceptanceevidencego-1"></a>

## Former source `internal/acceptance/evidence.go` (body snapshot `519b76bd`)

Historical locator and selector/body evidence are pinned to `519b76bd`; these selectors do not describe a current path or current body.

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–495; whole file; Evidence; CreateEvidence; Evidence.Close; Evidence.Path; Evidence.Writer; validEvidenceName; Evidence.mkdirEvidenceParent; Evidence.ensureEvidenceParents; Evidence.open; Evidence.Write; Evidence.scrubJSON; Evidence.encodeScrubbedJSON; Evidence.WriteJSON; Observation; hashAt; Evidence.Hashes; Evidence.PublishObservation; longestSecret; secretOverlapsBlock; retainSecretOverlap; scanEvidenceBytes; Evidence.scanRegularEvidence; Evidence.scanEvidencePath; Evidence.CheckSecrets; safeError; safeError.Error; safeError.Unwrap; Evidence.RedactString; urlPattern; redactURLs; Evidence.RedactError; redactingWriter; evidenceLimit; redactingWriter.Write; matchingSecretLen; redactPendingSecrets; urlRedactionEnd; redactingWriter.flush; redactingWriter.Close; PrivateFile | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility; 41 named units assigned here; remaining selectors preserve each duty — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |

<a id="coverage-2b8ba6d19ae5"></a>

## Former source `internal/acceptance/evidence_finalize_test.go` (body snapshot `519b76bd`)

Historical locator and selector/body evidence are pinned to `519b76bd`; these selectors do not describe a current path or current body.

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–129; file scaffold; TestStructuredEvidenceEscapesAndNumericIdentity; TestEscapedCredentialsInRawSplitWrites; TestFinalizationDoesNotPublishFailedOrOccupiedAttempts; TestEvidenceScansRetainOpenDirectoryAfterRename | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Current scaffold duty: file scaffold; 5 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-ed93c2fafab9"></a>

## Former source `internal/acceptance/evidence_test.go` (body snapshot `519b76bd`)

Historical locator and selector/body evidence are pinned to `519b76bd`; these selectors do not describe a current path or current body.

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–139; file scaffold; fixtureEvidence; TestEvidenceSplitSecretsAndRedirectQueries; TestEvidenceExclusiveAndConfined; TestRedactedErrorRetainsIdentity; TestCommandAndEvidenceFailuresAreSeparate; TestCancelledCommandIsNotDenialOrSuccess; TestEvidenceOutputBound | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Current scaffold duty: file scaffold; 8 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-ae5d28e85777"></a>

## [internal/acceptance/installed.go](../../../../../internal/acceptance/installed.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–191; file scaffold; probeFailure; Error; Unwrap; fail; failParen; failDetail; privateFile; privateDir; RestrictUmask; ownedByCaller; runOutcome; runBounded; runBoundedEnv; runBoundedDirEnv; uuidHex; machineArch | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Current scaffold duty: file scaffold; 17 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-f0c2fb4d73b2"></a>

## Former source `internal/acceptance/lifecycle_state.go` (body snapshot `519b76bd`)

Historical locator and selector/body evidence are pinned to `519b76bd`; these selectors do not describe a current path or current body.

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–501; whole file; LifecycleStateUsage; hostSodaTables; shQuote; snapReportError; snapReportError.Error; lifecycleRun; repoRoot; snapshotEntry; snapshotInputs; loadSnapshotInputs; snapshotEntries; queryHostSoda; dumpHostSoda; dumpHostTable; snapshotProject; observeProjectEndpoint; remoteSnapshotWrapper; remotePayload; execProjectSnapshot; projectSubnet; projectSnapshotIP; asStateMap; writeSnapshot; runSnapshot; runSnapshotAt; runCompare; validSnapshotLabel; RunLifecycleState; runLifecycleState | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility; 30 named units assigned here; remaining selectors preserve each duty — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |

<a id="coverage-045e205e27ac"></a>

## Former source `internal/acceptance/lifecycle_state_test.go` (body snapshot `519b76bd`)

Historical locator and selector/body evidence are pinned to `519b76bd`; these selectors do not describe a current path or current body.

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–351; file scaffold; lifecycleFixture; TestRunLifecycleCompare; TestSnapshotEntries; TestProjectSnapshotIP; TestWriteSnapshot; TestLifecycleRun; stubDashboardDB; stubTestVM; TestRunSnapshotAt; TestDumpHostSoda; TestRemotePayloadMissing; TestShQuote; TestExecProjectSnapshotRejectsBadIP; TestRunLifecycleValidation; TestRepoRoot | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Current scaffold duty: file scaffold; 16 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-cbb9678ab472"></a>
<a id="internalacceptancepersonal_gitgo-1"></a>

## [internal/acceptance/personal_git.go](../../../../../internal/acceptance/personal_git.go)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–497; whole file; PersonalGitUsage; gitRemoteTemplate; gitFetchProbe; gitTarget; loadGitTarget; parseGitTarget; gitProbe; gitProbe.gitSSHBase; gitChecked; tokenPassphrase; gitRemoteProgram; preparePassfile; loadPassphrase; gitProbe.gitKeyUser; gitProbe.fetchExportedKey; checkUnlockedKey; validateGitURL; checkGitEndpoint; checkGitUser; checkGitPath; exerciseCommand; extractCommit; gitOutcome; gitProbe.loadExerciseRepo; gitProbe.exerciseUser; writeGitOutcomes; gitProbe.phaseKey; gitProbe.phaseExercise; gitPhase; RunPersonalGit; runPersonalGit | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility; 32 named units assigned here; remaining selectors preserve each duty — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |

<a id="coverage-6cb96c2321af"></a>

## [internal/acceptance/personal_git_test.go](../../../../../internal/acceptance/personal_git_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–384; file scaffold; TestGitRemoteProgram; TestGitRemoteProgramExecutesPrepareThenUnlock; TestTokenPassphrase; TestLoadGitTarget; TestValidateGitURL; TestExtractCommit; TestGitOutcomesOrder; stubGitSSH; gitFixture; TestRunPersonalGitPrepareUnlock; TestRunPersonalGitExercise; TestRunPersonalGitValidation | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Current scaffold duty: file scaffold; 13 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-320733319b4d"></a>

## [internal/acceptance/process.go](../../../../../internal/acceptance/process.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–124; file scaffold; Process; StartProcess; StartCommand; Wait; Done; signal; Stop | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Current scaffold duty: file scaffold; 8 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-d974aa3beb79"></a>

## [internal/acceptance/process_linux_test.go](../../../../../internal/acceptance/process_linux_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–120; file scaffold; TestOwnedChildFixture; TestOwnedLeaderExitAndCancellationStopResistantDescendant | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Current scaffold duty: file scaffold; Current declaration duty: TestOwnedChildFixture; Current declaration duty: TestOwnedLeaderExitAndCancellationStopResistantDescendant — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-1fa1ffb83e8f"></a>

## [internal/acceptance/process_wait_linux.go](../../../../../internal/acceptance/process_wait_linux.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–32; file scaffold; reapOwnedChildren; ownedGroupsSupported; waitOwnedExit | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Current scaffold duty: file scaffold; 4 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-b095d6601b7f"></a>

## [internal/acceptance/process_wait_other.go](../../../../../internal/acceptance/process_wait_other.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–13; file scaffold; ownedGroupsSupported; waitOwnedExit; reapOwnedChildren | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Current scaffold duty: file scaffold; 4 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-83cabeee5635"></a>

## [internal/acceptance/service_https.go](../../../../../internal/acceptance/service_https.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–154; file scaffold; ServiceHTTPSUsage; UsageError; Error; HTTPSOrigin; checkHTTPSPort; plainHTTPSOrigin; trustedCAFile; httpsClient; CheckServiceHTTPS; RunServiceHTTPS | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Current scaffold duty: file scaffold; 11 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-86089390dd81"></a>

## [internal/acceptance/service_https_test.go](../../../../../internal/acceptance/service_https_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–172; file scaffold; TestHTTPSOrigin; writeCAFile; TestTrustedCAFile; pemCA; TestCheckServiceHTTPS; TestCheckServiceHTTPSAcceptsRedirectStatusWithoutFollowing; TestCheckServiceHTTPSRefusals; TestRunServiceHTTPS | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Current scaffold duty: file scaffold; 9 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-4284e2e1b560"></a>

## Former source `internal/acceptance/worker_linux.go` (body snapshot `519b76bd`)

Historical locator and selector/body evidence are pinned to `519b76bd`; these selectors do not describe a current path or current body.

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–176; file scaffold; Worker; validWorkerIdentity; appendBindPaths; allowedWorkerEnvKey; appendWorkerEnv; arguments; trustedPathMode; TrustedExecutable; Run | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Current scaffold duty: file scaffold; 10 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-6b94e1f1471e"></a>

## Former source `internal/acceptance/worker_linux_test.go` (body snapshot `519b76bd`)

Historical locator and selector/body evidence are pinned to `519b76bd`; these selectors do not describe a current path or current body.

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–43; file scaffold; TestWorkerUsesSeparateIdentityAndNativeServiceCustody; TestWorkerRefusesUntrustedExecutable | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Current scaffold duty: file scaffold; Current declaration duty: TestWorkerUsesSeparateIdentityAndNativeServiceCustody; Current declaration duty: TestWorkerRefusesUntrustedExecutable — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-9439df454f91"></a>

## [internal/acceptance/workload_access.go](../../../../../internal/acceptance/workload_access.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–389; file scaffold; WorkloadAccessUsage; createProbeTableSQL; insertProbeSQL; selectProbeSQL; updateProbeSQL; accessTarget; loadAccessTarget; parseAccessTarget; accessProbe; accessRemote; clientPSQL; memberPSQLPrefix; setupAccessOutput; writePassfile; fetchAccessURL; provisionMemberCredentials; verifyClientWrites; verifyMemberReads; verifyHTTPBind; workloadAccessResults; finishAccessResults; RunWorkloadAccess; runWorkloadAccess; setupAccessProbe; runAccessDatabase | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Current scaffold duty: file scaffold; 26 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-e999cd4597ee"></a>

## [internal/acceptance/workload_access_test.go](../../../../../internal/acceptance/workload_access_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–207; file scaffold; TestLoadAccessTarget; TestAccessSQL; stubAccessSSH; TestAccessRemote; TestClientPSQL; TestWritePassfile; TestAccessResultsOrder; TestRunWorkloadAccessValidation | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Current scaffold duty: file scaffold; 9 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-6446d9d0380b"></a>

## [internal/acceptance/workload_exec.go](../../../../../internal/acceptance/workload_exec.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–121; file scaffold; workloadExecProbe; WorkloadExecUsage; loadExecTarget; execBaseArgs; execDifferentUID; checkMemberDenied; RunWorkloadExec; runWorkloadExec | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Current scaffold duty: file scaffold; 9 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-3d889ea1e95c"></a>

## [internal/acceptance/workload_exec_test.go](../../../../../internal/acceptance/workload_exec_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–130; file scaffold; writeExecFixture; stubSSH; bobExitString; TestRunWorkloadExec; TestRunWorkloadExecRefusals | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Checks three current piped script roots and four fail-closed refusal cases; former retired-Python byte equality is removed. — Current complete source and selectors inspected |


## Current path reconciliation at `45ebf4c4`

This section reconciles current path ownership only. Existing numeric/body selectors above remain pinned to `519b76bd` unless a row explicitly gives a narrow current inspection.


<a id="r02-current-path-internal-acceptance-installed-test-go"></a>

### [internal/acceptance/installed_test.go](../../../../../internal/acceptance/installed_test.go)

D06: lines 11–135; fixture helper 11–19, private-input tests 21–135.
