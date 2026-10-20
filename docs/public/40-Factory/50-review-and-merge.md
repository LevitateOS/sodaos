# Review and merge

Factory reviews each candidate against its exact source revision, target branch, accepted requirements and required outcomes. An independent Inspector assesses the change, and Factory merges through the native forge only after current review evidence, required CI checks and repository policy all permit delivery.

## Inspect the candidate

The Inspector receives the candidate and its relevant accepted requirements, changes, coverage obligations and existing findings. It reports concrete defects with impact and evidence, or explicitly records that the complete required review found no defects. An incomplete review does not count as clear. The Builder's own report is useful context but does not replace independent review.

When required findings remain, a Fixer addresses the full required set and provides evidence for each disposition. A proposed fix is not accepted just because code changed. The revised candidate receives a fresh independent review, or an authorized assessment of verified corrections where policy permits. Disputed findings remain visible until an independent decision resolves them. Optional advice can be declined with its rationale without disguising it as a required finding.

## Require current checks and policy

The candidate identity includes the revision being reviewed and the branch it targets. Required native Actions checks must report on that same candidate. If commits, accepted requirements, branch policy or relevant evidence change, prior review and check results may no longer apply. Factory reassesses the current subject and requests fresh evidence before delivery.

Repository protections remain effective. A Factory review cannot override a required check, current native permission, branch rule or merge restriction. The repository maintainer controls these policies; external runner setup is described in [CI runners](40-ci-runners.md).

## Confirm delivery

When the evidence and policy permit, Factory performs the native merge using its authorized actor. It then confirms both the merged result and closure of the original issue through the native completion path. A created pull request, a submitted merge request, a closed issue without the required merged outcome, or a lost response is not confirmation. If an effect is uncertain, Factory reconciles that original operation before retrying so it does not duplicate native activity.

Confirmed delivery triggers reassessment of dependent issues. Whole-application verification remains a separate step: the [Test Pilot](60-application-verification.md) checks the integrated intent version and approved outcomes after issue delivery.
