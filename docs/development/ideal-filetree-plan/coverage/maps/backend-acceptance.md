# Backend acceptance

[Responsibility map index](README.md) · [Coverage snapshot and limits](../README.md).
Page grouping is navigation; the slice IDs retain their individual review ownership.

<a id="coverage-f470fa6573df"></a>

<a id="internalacceptancedeveloper_access_testgo-1"></a>

## [internal/acceptance/developer_access_test.go](../../../../../internal/acceptance/developer_access_test.go)

Test/fixture assertions inspected as source only; no test, build, native driver or browser run performed. A defined scenario is not a passing qualification result.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 1–13 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 14–15 | Fixture/protocol support testFingerprint; declarations/fields: `testFingerprint` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 16–17 | Fixture/protocol support testHostKey; declarations/fields: `testHostKey` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 18–43 | Assertions TestParseAccessSubnet: broadcast = %s; declarations/fields: `TestParseAccessSubnet` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 44–57 | Fixture/protocol support writeTestKey; declarations/fields: `writeTestKey` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 58–109 | Assertions TestValidateAccessUser: valid user rejected: %+v %v; declarations/fields: `TestValidateAccessUser` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 110–125 | Assertions TestQuoteAccessPath: plain quote = %q %v; declarations/fields: `TestQuoteAccessPath` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 126–139 | Assertions TestParseAccessUID: uid %q = %d %v; declarations/fields: `TestParseAccessUID` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 140–163 | Assertions TestAccessSSHOptions: options differ:\n%q; declarations/fields: `TestAccessSSHOptions` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 164–173 | Fixture/protocol support accessFixture; declarations/fields: `accessFixture` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 174–180 | Fixture/protocol support preserveUmask; declarations/fields: `preserveUmask` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 181–213 | Assertions TestRunDeveloperAccessValidation: bad request accepted; declarations/fields: `TestRunDeveloperAccessValidation` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 214–239 | Fixture/protocol support writeAccessRequest; declarations/fields: `writeAccessRequest` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 240–275 | Fixture/protocol support writeAccessSupport; declarations/fields: `writeAccessSupport` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 276–318 | Assertions TestLoadAccessRequest: valid request rejected: %v; declarations/fields: `TestLoadAccessRequest` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 319–367 | Fixture/protocol support stubAccessTransport; declarations/fields: `stubAccessTransport` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 368–447 | Assertions TestRunDeveloperAccessEndToEnd: end-to-end access failed: %v; declarations/fields: `TestRunDeveloperAccessEndToEnd` |

<a id="coverage-a8e5246b36b5"></a>

<a id="internalacceptanceevidencego-1"></a>

## [internal/acceptance/evidence.go](../../../../../internal/acceptance/evidence.go)

Committed ddd8715b source map. Cohesive functions keep one owner; a dependency/reference does not confer authority over the referenced state.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 1–24 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 25–29 | Record, DTO or interface contract Evidence for Installed qualification; declarations/fields: `Evidence` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 30–61 | CreateEvidence — Installed qualification; declarations/fields: `CreateEvidence` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 62 | Evidence.Close — Installed qualification; declarations/fields: `Evidence.Close` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 63 | Evidence.Path — Installed qualification; declarations/fields: `Evidence.Path` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 64–71 | Evidence.Writer — Installed qualification; declarations/fields: `Evidence.Writer` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 72–75 | validEvidenceName — Installed qualification; declarations/fields: `validEvidenceName` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 76–89 | Evidence.mkdirEvidenceParent — Installed qualification; declarations/fields: `Evidence.mkdirEvidenceParent` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 90–103 | Evidence.ensureEvidenceParents — Installed qualification; declarations/fields: `Evidence.ensureEvidenceParents` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 104–117 | Evidence.open — Installed qualification; declarations/fields: `Evidence.open` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 118–128 | Evidence.Write — Installed qualification; declarations/fields: `Evidence.Write` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 129–158 | Evidence.scrubJSON — Installed qualification; declarations/fields: `Evidence.scrubJSON` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 159–187 | Evidence.encodeScrubbedJSON — Installed qualification; declarations/fields: `Evidence.encodeScrubbedJSON` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 188–206 | Evidence.WriteJSON — Installed qualification; declarations/fields: `Evidence.WriteJSON` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 207–218 | Record, DTO or interface contract Observation for Installed qualification; declarations/fields: `Observation` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 219–245 | hashAt — Installed qualification; declarations/fields: `hashAt` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 246–269 | Evidence.Hashes — Installed qualification; declarations/fields: `Evidence.Hashes` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 270–279 | Evidence.PublishObservation — Installed qualification; declarations/fields: `Evidence.PublishObservation` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 280–289 | longestSecret — Installed qualification; declarations/fields: `longestSecret` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 290–298 | secretOverlapsBlock — Installed qualification; declarations/fields: `secretOverlapsBlock` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 299–305 | retainSecretOverlap — Installed qualification; declarations/fields: `retainSecretOverlap` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 306–325 | scanEvidenceBytes — Installed qualification; declarations/fields: `scanEvidenceBytes` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 326–341 | Evidence.scanRegularEvidence — Installed qualification; declarations/fields: `Evidence.scanRegularEvidence` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 342–355 | Evidence.scanEvidencePath — Installed qualification; declarations/fields: `Evidence.scanEvidencePath` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 356–359 | Evidence.CheckSecrets — Installed qualification; declarations/fields: `Evidence.CheckSecrets` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 360–364 | Record, DTO or interface contract safeError for Installed qualification; declarations/fields: `safeError` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 365 | safeError.Error — Installed qualification; declarations/fields: `safeError.Error` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 366 | safeError.Unwrap — Installed qualification; declarations/fields: `safeError.Unwrap` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 367–373 | Evidence.RedactString — Installed qualification; declarations/fields: `Evidence.RedactString` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 374–375 | Declared identifiers/bounds urlPattern for Installed qualification; declarations/fields: `urlPattern` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 376–389 | redactURLs — Installed qualification; declarations/fields: `redactURLs` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 390–398 | Evidence.RedactError — Installed qualification; declarations/fields: `Evidence.RedactError` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 399–408 | Record, DTO or interface contract redactingWriter for Installed qualification; declarations/fields: `redactingWriter` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 409–410 | Declared identifiers/bounds evidenceLimit for Installed qualification; declarations/fields: `evidenceLimit` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 411–427 | redactingWriter.Write — Installed qualification; declarations/fields: `redactingWriter.Write` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 428–437 | matchingSecretLen — Installed qualification; declarations/fields: `matchingSecretLen` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 438–452 | redactPendingSecrets — Installed qualification; declarations/fields: `redactPendingSecrets` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 453–459 | urlRedactionEnd — Installed qualification; declarations/fields: `urlRedactionEnd` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 460–473 | redactingWriter.flush — Installed qualification; declarations/fields: `redactingWriter.flush` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 474–482 | redactingWriter.Close — Installed qualification; declarations/fields: `redactingWriter.Close` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 483–495 | PrivateFile — Installed qualification; declarations/fields: `PrivateFile` |

<a id="coverage-f0c2fb4d73b2"></a>

## [internal/acceptance/lifecycle_state.go](../../../../../internal/acceptance/lifecycle_state.go)

Committed ddd8715b source map. Cohesive functions keep one owner; a dependency/reference does not confer authority over the referenced state.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 1–27 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 28–31 | Declared identifiers/bounds LifecycleStateUsage for Installed qualification; declarations/fields: `LifecycleStateUsage` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 32–35 | Declared identifiers/bounds hostSodaTables for Installed qualification; declarations/fields: `hostSodaTables` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 36–41 | shQuote — Installed qualification; declarations/fields: `shQuote` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 42–43 | Record, DTO or interface contract snapReportError for Installed qualification; declarations/fields: `snapReportError` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 44–46 | snapReportError.Error — Installed qualification; declarations/fields: `snapReportError.Error` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 47–65 | lifecycleRun — Installed qualification; declarations/fields: `lifecycleRun` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 66–81 | repoRoot — Installed qualification; declarations/fields: `repoRoot` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 82–88 | Record, DTO or interface contract snapshotEntry for Installed qualification; declarations/fields: `snapshotEntry` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 89–95 | Record, DTO or interface contract snapshotInputs for Installed qualification; declarations/fields: `snapshotInputs` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 96–127 | loadSnapshotInputs — Installed qualification; declarations/fields: `loadSnapshotInputs` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 128–159 | snapshotEntries — Installed qualification; declarations/fields: `snapshotEntries` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 160–183 | queryHostSoda — Installed qualification; declarations/fields: `queryHostSoda` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 184–216 | dumpHostSoda — Installed qualification; declarations/fields: `dumpHostSoda` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 217–261 | dumpHostTable — Installed qualification; declarations/fields: `dumpHostTable` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 262–276 | snapshotProject — Installed qualification; declarations/fields: `snapshotProject` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 277–297 | observeProjectEndpoint — Installed qualification; declarations/fields: `observeProjectEndpoint` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 298–303 | Declared identifiers/bounds remoteSnapshotWrapper for Installed qualification; declarations/fields: `remoteSnapshotWrapper` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 304–312 | remotePayload — Installed qualification; declarations/fields: `remotePayload` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 313–337 | execProjectSnapshot — Installed qualification; declarations/fields: `execProjectSnapshot` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 338–340 | Declared identifiers/bounds projectSubnet for Installed qualification; declarations/fields: `projectSubnet` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 341–359 | projectSnapshotIP — Installed qualification; declarations/fields: `projectSnapshotIP` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 360–365 | asStateMap — Installed qualification; declarations/fields: `asStateMap` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 366–387 | writeSnapshot — Installed qualification; declarations/fields: `writeSnapshot` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 388–396 | runSnapshot — Installed qualification; declarations/fields: `runSnapshot` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 397–433 | runSnapshotAt — Installed qualification; declarations/fields: `runSnapshotAt` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 434–460 | runCompare — Installed qualification; declarations/fields: `runCompare` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 461–465 | validSnapshotLabel — Installed qualification; declarations/fields: `validSnapshotLabel` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 466–477 | RunLifecycleState — Installed qualification; declarations/fields: `RunLifecycleState` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 478–501 | runLifecycleState — Installed qualification; declarations/fields: `runLifecycleState` |

<a id="coverage-cbb9678ab472"></a>

<a id="internalacceptancepersonal_gitgo-1"></a>

## [internal/acceptance/personal_git.go](../../../../../internal/acceptance/personal_git.go)

Committed ddd8715b source map. Cohesive functions keep one owner; a dependency/reference does not confer authority over the referenced state.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 1–27 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 28–37 | Declared identifiers/bounds PersonalGitUsage for Installed qualification; declarations/fields: `PersonalGitUsage` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 38–77 | Declared identifiers/bounds gitRemoteTemplate for Installed qualification; declarations/fields: `gitRemoteTemplate` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 78–85 | Declared identifiers/bounds gitFetchProbe for Installed qualification; declarations/fields: `gitFetchProbe` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 86–91 | Record, DTO or interface contract gitTarget for Installed qualification; declarations/fields: `gitTarget` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 92–107 | loadGitTarget — Installed qualification; declarations/fields: `loadGitTarget` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 108–128 | parseGitTarget — Installed qualification; declarations/fields: `parseGitTarget` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 129–135 | Record, DTO or interface contract gitProbe for Installed qualification; declarations/fields: `gitProbe` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 136–150 | gitProbe.gitSSHBase — Installed qualification; declarations/fields: `gitProbe.gitSSHBase` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 151–163 | gitChecked — Installed qualification; declarations/fields: `gitChecked` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 164–174 | tokenPassphrase — Installed qualification; declarations/fields: `tokenPassphrase` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 175–185 | gitRemoteProgram — Installed qualification; declarations/fields: `gitRemoteProgram` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 186–203 | preparePassfile — Installed qualification; declarations/fields: `preparePassfile` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 204–222 | loadPassphrase — Installed qualification; declarations/fields: `loadPassphrase` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 223–247 | gitProbe.gitKeyUser — Installed qualification; declarations/fields: `gitProbe.gitKeyUser` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 248–264 | gitProbe.fetchExportedKey — Installed qualification; declarations/fields: `gitProbe.fetchExportedKey` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 265–276 | checkUnlockedKey — Installed qualification; declarations/fields: `checkUnlockedKey` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 277–291 | validateGitURL — Installed qualification; declarations/fields: `validateGitURL` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 292–303 | checkGitEndpoint — Installed qualification; declarations/fields: `checkGitEndpoint` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 304–317 | checkGitUser — Installed qualification; declarations/fields: `checkGitUser` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 318–325 | checkGitPath — Installed qualification; declarations/fields: `checkGitPath` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 326–348 | exerciseCommand — Installed qualification; declarations/fields: `exerciseCommand` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 349–363 | extractCommit — Installed qualification; declarations/fields: `extractCommit` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 364–371 | Record, DTO or interface contract gitOutcome for Installed qualification; declarations/fields: `gitOutcome` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 372–390 | gitProbe.loadExerciseRepo — Installed qualification; declarations/fields: `gitProbe.loadExerciseRepo` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 391–414 | gitProbe.exerciseUser — Installed qualification; declarations/fields: `gitProbe.exerciseUser` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 415–423 | writeGitOutcomes — Installed qualification; declarations/fields: `writeGitOutcomes` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 424–435 | gitProbe.phaseKey — Installed qualification; declarations/fields: `gitProbe.phaseKey` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 436–452 | gitProbe.phaseExercise — Installed qualification; declarations/fields: `gitProbe.phaseExercise` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 453–460 | gitPhase — Installed qualification; declarations/fields: `gitPhase` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 461–467 | RunPersonalGit — Installed qualification; declarations/fields: `RunPersonalGit` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 468–497 | runPersonalGit — Installed qualification; declarations/fields: `runPersonalGit` |

