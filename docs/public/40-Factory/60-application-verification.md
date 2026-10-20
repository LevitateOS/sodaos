# Application verification

The Test Pilot verifies accepted outcomes against an approved manifest on the exact integrated application version, separately from issue delivery and deployment.

## Verify the integrated version

1. **Establish the verification subject.** Use the integrated source or artifact identity, accepted outcome coverage, approved checks and journeys, prepared fixture identity and trusted check receipts for the requested intent.
2. **Record observations against that subject.** The Test Pilot records what each journey observed, which evidence supports the result and which accepted outcomes remain uncovered.

The result applies only to the tested version and conditions. If source, artifact, fixture or accepted outcomes change, earlier evidence remains historical and the changed subject needs applicable fresh evidence. A finished issue, merged pull request or empty queue cannot substitute for integrated verification. Likewise, passing a subset does not establish complete coverage.

## Understand the outcome

| Result | What it establishes |
| --- | --- |
| Verified | Every required outcome has sufficient current evidence under the approved manifest. |
| Failed | The observed behavior and evidence identify an outcome for the responsible owner to assess for bounded correction. |
| Incomplete or inconclusive | Coverage, fixtures, infrastructure or evidence cannot support a conclusion; untested outcomes do not pass. |

Missing preparation or an invalid fixture prevents a valid claim about the application behavior. A required check that fails before the Test Pilot can begin is reported as a Factory verification failure, not as an invented Test Pilot observation. Repairs and reruns remain bounded by their authorized allowances and must retain exact links to the failed outcome and tested delivery.

Confirmed application completion is recorded separately from issue completion and from an application's installation or deployment. For candidate review and merge evidence, see [Review and merge](50-review-and-merge.md). For separately authorized private application administration, see [Private applications](../70-Operate/40-private-applications.md).
