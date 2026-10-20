# Application verification

After issue work is delivered, the Test Pilot checks the integrated application version against the accepted outcomes and approved verification manifest. This confirms whether the requested behavior works together at that exact version; it does not deploy or install the application.

## Verify the integrated version

Verification uses the approved checks, journeys and prepared fixtures for the requested intent. The Test Pilot receives the integrated source or artifact identity, outcome coverage, fixture identity and trusted check receipts. It records what each journey observed, what evidence supports the result and which accepted outcomes remain uncovered.

The result applies only to the tested version and conditions. If source, artifact, fixture or accepted outcomes change, earlier evidence remains historical and the changed subject needs applicable fresh evidence. A finished issue, merged pull request or empty queue cannot substitute for integrated verification. Likewise, passing a subset does not establish complete coverage.

## Understand the outcome

A verified result means every required outcome has sufficient current evidence under the approved manifest. A failure records the specific observed behavior and evidence so the responsible owner can assess a bounded correction. An incomplete or inconclusive result identifies missing coverage, unavailable fixtures, failed infrastructure or evidence that cannot support a conclusion. It does not turn an untested outcome into a pass.

Missing preparation or an invalid fixture prevents a valid claim about the application behavior. A required check that fails before the Test Pilot can begin is reported as a Factory verification failure, not as an invented Test Pilot observation. Repairs and reruns remain bounded by their authorized allowances and must retain exact links to the failed outcome and tested delivery.

Confirmed application completion is recorded separately from issue completion and from an application's installation or deployment. For candidate review and merge evidence, see [Review and merge](50-review-and-merge.md). For separately authorized private application administration, see [Private applications](../70-Operate/40-private-applications.md).
