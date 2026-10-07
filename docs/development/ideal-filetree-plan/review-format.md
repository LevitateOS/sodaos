# Slice validity review format

Use this format for every slice in the [catalog](slices/README.md). Establish
each slice's intended model during its audit, then compare implementation and
test assumptions with that model. Existing slice cards and responsibility maps
establish review scope; they do not establish correctness or approve requirements.

The [current workflow requirements](workflow-requirements.md) connect all 80
existing slice IDs to actors, authority, inputs, state and required outcomes.
Use their canonical owner links and explicit questions when establishing a model;
the map does not advance historical source review or prove implementation behavior.

Create one review record at `reviews/<slice-id>.md` when that slice's review
begins, using its exact catalog ID, such as `reviews/I06.md`. Link it from the
existing slice card and add its actual path to the proposed tree and documentation
coverage. Keep one current record per slice. This format does not create review
records or mark any slice reviewed.

Use the [assignment map and collaboration rules](review-assignments.md) for
the primary owner, consequential-conclusion challenger, boundary exchanges and
shared plan updates. Only the named primary writes its slice record; the
coordinator writes shared indexes, coverage maps and the proposed tree.

Read the shared [guidance conflicts and controlling decisions](review-assignments.md#guidance-conflicts-and-controlling-decisions)
before establishing the intended model or allocating languages. Link applicable
conflict IDs in the slice record and identify the controlling owner decision,
including its scope and provenance. Send newly discovered conflicts to the
coordinator; resolve them in the shared register rather than inventing a local
default. A corrected obsolete rule and a still-unresolved decision are different
dispositions.

The required sections and copyable record below are shared by all reviewers.
Use explicit `none`, `not performed` or `not applicable` with a reason when
appropriate. A material unknown stays unresolved; a blank field is not a decision.
Apply checks to the slice's established responsibilities rather than imposing
identity-specific operations or speculative behavior on every capability.

## Recording scope and status

Start from the [prepared review input baseline](review-baseline.md), verifying
its source commit and working-guidance hashes before recording a slice's actual
review scope. Reconcile drift explicitly; the baseline manifest identifies
inputs and does not establish an intended model or a completed validity review.

Record the exact source commit and any inspected uncommitted source changes.
Identify those changes by path and retained diff or blob identity, so the
reviewed source can be distinguished from later edits.
Identify contract revisions and controlling user decisions separately. Source
links and line ranges refer to that recorded source, not a later working tree.
State which earlier inspections are reused and why their evidence still applies.
Reuse must address the same review question and controlling requirements. A
structural inventory or line map cannot substitute for a previous validity
review, even when the source has not changed.

Keep these status fields independent:

| Field | Allowed descriptions and meaning |
| --- | --- |
| Intended model | `not established`, `established`, or `unresolved`; material unresolved requirements prevent a complete model verdict. |
| Source coverage | `not started`, `partial`, or `complete for recorded scope`; complete means all assigned files/responsibilities inspected or explicitly supported by unchanged prior inspection. |
| Findings | `not assessed`, `open`, `none found within reviewed scope`, or `resolved with linked evidence`; absence of findings is limited to the inspected scope. |
| Behavioral evidence | `not performed`, or name the checks actually performed, their evidence classes and results; a defined test is not an executed check. |
| Correction readiness | `not specified`, `decision pending`, or `specified for listed corrections`; readiness applies to named actions and does not grant implementation permission. |

Source coverage can be complete while findings remain open, behavior remains
unexecuted or a product decision is unresolved. A source-only review can complete
its stated scope. Do not use an undifferentiated `valid` or `passed` slice status.

## Establishing the intended model

Describe purpose, actors, entrypoints, authority/trust boundary, owned data and
source of truth, lifecycle, dependencies, required outcomes and exclusions. Use
the current user decision and owning product/trust contract for requirements;
cite external protocols or reproduced failures where they establish a relevant
technical constraint. Existing code and test assertions describe implementation,
not automatic product authority. An audit recommendation cannot create a
requirement merely by being recorded here.

Give each material requirement a local reference ID so findings and test
assessments can point to its independent authority. Identify conflicting or
outdated guidance and record the exact decision still needed. Review can continue
on established responsibilities while that question remains open; do not claim
agreement with an undefined requirement.

Challenge concrete assumptions in the actual caller chains: implicit provider
defaults, shared models carrying adapter-specific fields or policy, duplicated
state machines, competing sources of truth, unsuitable interfaces, resource
lifetime assumptions and ports preserving obsolete behavior. Preserve legitimate
provider differences. Mark an assumption supported, contradicted or unresolved
with its evidence; do not turn a hypothetical risk into a defect.

## Accounting for files and responsibilities

Start from the [inventory](coverage/inventory/README.md) and
[responsibility maps](coverage/maps/README.md). Account for every responsibility
assigned to the slice, including small helpers, frontend behavior, SQL fields,
scripts, configuration, manifests and tests. For mixed files, use separate exact
line ranges and field/symbol selectors; one inspected function does not establish
whole-file review. An unmapped or newly discovered responsibility keeps coverage
partial until the map is corrected.

The slice's scope is its complete assigned responsibility inventory at the
recorded baseline. A bounded pass can cover a subset, but its slice coverage
remains partial until every assigned responsibility is accounted for; do not
shrink the scope to make the completion field pass.

Identify caller orchestration separately from delegated domain authority. Link
other slices' records for their responsibilities and shared workflow findings;
do not mark those responsibilities reviewed merely because their caller was read.
A mixed file is fully reviewed only when its assigned owners' records collectively
cover every mapped responsibility.

Record the slice's established operations as caller/workflow traces, including
their relevant state and authority transitions. Link the other responsible
slices and identify boundaries still awaiting review; source coverage alone
does not establish that the complete operation implements its intended model.

Record obsolete code and generated artifacts separately. Retirement requires
current caller/build/install evidence; generated inputs retain their provenance
and consumer assessment. Neither disposition is permission to delete source.
Binary inputs use file-level responsibility instead of invented line ranges.

## Specifying language and target ownership

For every retained or successor backend/native implementation responsibility,
record its current language and target choice of **Go** or **Rust**, identifying
an existing decision or a supported recommendation. If the choice is unresolved,
record the exact pending question instead. Every correction marked specified
must have an explicit target owner and language. A slice can span both, with a
separate allocation for each responsibility.
Frontend, SQL, assets and configuration retain their explicit applicable
substrates; the Go/Rust choice does not require porting those materials.

Explain the language fit using the actual integration/dependency boundary,
resource lifetime and concurrency needs, measured runtime constraints where
available, and maintenance cost. Compare the other language and record what
supports the choice and where comparative evidence is inconclusive. Porting
effort alone does not settle the choice. Cite existing owner decisions; label
recommendations as such.
Conflicting language guidance or a proposal that changes an established owner
decision remains a decision question, rather than a silent policy override.
Do not mark the affected correction specified until its applicable decision is
identified and any material conflicting guidance is resolved. The independent
challenger checks that applicability; technical preference alone cannot settle it.

List exact target repository paths, Go packages or Rust crates/modules, their
responsibilities and actions: `retain`, `move`, `refactor`, `split`, `consolidate`,
`port` or `retire`. A directory name, wildcard, `Go/Rust`, or an unspecified
destination is insufficient for correction readiness. Several actions can be
listed when their order and effects are explicit.

Name the executable/process that hosts each owner and whether the existing
process topology changes. A package split or language choice does not imply a
sidecar. A proposed new process needs an established requirement or concrete
evidence and an explicit topology decision.

For ports and consolidation, name the retained successor, predecessor files and
smallest retired units, real callers, shared data/wire definitions, tests,
manifests, service/build/install wiring and cutover order. Keep obsolete and
pre-port implementations out of the active target; retain their retirement
instructions separately. Do not infer authorized coexistence or compatibility.
Pure deletion can have no successor; use `none — retired; no successor` for
target language, owner, files and hosting process when applicable.

## Recording findings and evidence

Give each finding a stable ID prefixed by its owning slice, such as `I06-F1`.
Record a shared finding once with the responsible slice and link the same ID from
affected reviews. Include the observed assumption/behavior, independently cited
requirement, affected caller chain, supported consequence, source/evidence,
proposed correction and exact correction targets. Distinguish an observed result
from an inference. Classify correctness defects, architectural debt,
test-contract mismatches and unresolved requirements separately; a preference is
not a correctness finding.
An unresolved-requirement finding cites the conflicting sources and missing
decision rather than inventing definitive required behavior.

Compare test assertions with the intended model before using them as evidence.
A passing test can enforce the wrong contract. Identify real implementations,
stubs/copies, fixture prerequisites and skips, and exactly what each assertion
can establish. Missing execution is an evidence limit, not automatically a defect.

Keep source inspection, authored checks, local execution, CI, native integration,
build/export, deployed revision and installed user behavior distinct. An evidence
entry records its actual result, source/artifact revision, fixture/substitutions
and limits. Planned checks belong with corrections, not performed evidence.
Require proof appropriate to the established behavior, using existing drivers
where sufficient; this format does not mandate running every proof class.

A correction is specified only when its intended behavior, language/owner,
exact affected and target paths, caller/wiring changes, prerequisites, cutover
and applicable verification are explicit. Unresolved material choices keep the
affected action `decision pending`; independent corrections can remain specified.
Proposed tests must follow established requirements or reproduced failures.
Record a finding resolved only with the correcting revision and evidence that
supports that resolution. An audit or specified correction grants no permission
to implement, run state-changing checks, commit, push or deploy.

## Keeping a review current

After each merge, follow [plan maintenance](maintenance.md) and compare changes
to both the reviewed source and its controlling contracts. Recheck affected
responsibilities, caller chains, shared boundaries, tests and target allocations.
Label affected conclusions stale until reconciled. Set source coverage partial
for outstanding changed responsibilities, intended model unresolved for material
contract questions, and affected corrections decision pending when choices are
unsettled. Record what earlier evidence still applies and why. A fresh HEAD or
passing unrelated test does not refresh the whole review. Keep the current record, rather than append a merge
diary. Advance each recorded evidence scope only to what was actually inspected
or exercised.

## Completion criteria

Completion applies to the current catalog and recorded source/contract baseline,
including all responsibilities discovered during review. Do not declare the
whole audit complete by adding up agent summaries or marking files opened.

| Completion dimension | Required evidence in the records |
| --- | --- |
| Responsibility coverage | Every established responsibility in every slice is reviewed, including mixed-file units, callers, frontend behavior, SQL, scripts, configuration and tests. Newly discovered responsibilities have owners and coverage entries. Obsolete units and generated inputs have supported, separately recorded dispositions; no unmapped file or responsibility is silently excluded. |
| Complete workflow analysis | Every established operation has a complete caller/state/authority/lifecycle trace across its participating slices, from entrypoint through its required outcomes and established failure/termination paths. Each participating primary has checked its part and shared-boundary exchanges are recorded. A missing caller, transition, owning review or boundary reply keeps the affected workflow incomplete. Source traces do not claim execution or installed proof. |
| Independent challenge | Every concrete finding and consequential conclusion has been checked by the assigned independent challenger against its underlying requirements and evidence. The record contains the challenge scope, objections, primary responses and remaining disagreement. This includes language/ownership, retirement and material claims that the existing design is adequate. Acknowledgment alone is insufficient; completion of challenge does not imply resolution of a finding. |
| Explicit target allocation | Every retained/successor implementation responsibility has a named target owner, exact target files and Go or Rust language, or the explicit applicable substrate for frontend/SQL/configuration/assets. Hosting process, caller/data/wire/test/build/install effects and retirement/cutover duties are specified. Retired-only units explicitly have no successor. No wildcard, alternative language, unnamed owner or pending target decision counts as a completed allocation. |

Record each dimension independently for each slice and cross-slice workflow.
The coordinator can report the whole audit complete only when all four are
complete across the full current catalog and there are no unreviewed assigned
responsibilities, unfinished workflow traces, outstanding required challenges
or unspecified target allocations. If any remain, name the incomplete scopes
and dimensions instead of using a blanket completion claim.

An open finding can be fully reviewed and challenged without being corrected.
An unresolved requirement or evidence limit remains visible even after its
question has been inspected and challenged. Do not recast it as satisfied or
hide it in a completion total. Unknown ownership/language/target choices keep
the affected allocation dimension incomplete.

Every unresolved question links the affected requirement, finding, workflow,
target and correction/action IDs, identifies the decision owner or missing
evidence, and states exactly what would resolve it. Mark every dependent
implementation instruction `decision pending` and state what is blocked. Do
not publish executable port, deletion, interface, schema or wiring instructions
whose correctness depends on that question; independent, fully specified
instructions can remain specified. Audit completion and correction readiness
grant no authorship or operational permission.

Use stable local IDs for workflow traces, target allocations and correction
actions, such as `I06-W1`, `I06-T1` and `I06-A1`. Link the canonical record's ID
when a question spans slices; do not create duplicate findings or allocations
merely to refer to the same blocked instruction.

Keep proof limits separate from these completion dimensions. Source review,
test assertions and source workflow traces are not a behavioral PASS; record
the exact checks performed and the established behavior still unverified.

## Copyable slice review record

```markdown
# <slice-id> <catalog name> — validity review

- Review date:
- Primary reviewer / runtime agent / model:
- Assigned independent challenger / runtime agent:
- Independent review: pending, or reviewer and exact scope checked
- Source commit / inspected uncommitted source changes:
- Controlling contract revisions / user decisions:
- Applicable guidance-conflict IDs / controlling decision, scope and provenance:
- Catalog and responsibility-map links:
- Pass scope / responsibilities inspected in this pass:
- Reused evidence and justification:
- Intended model status:
- Source coverage status:
- Findings status:
- Behavioral evidence status:
- Correction readiness:
- Completion dimensions: responsibility coverage / workflow analysis / independent challenge / target allocation

## Intended model

- Purpose and established exclusions:
- Actors, entrypoints and inputs/outputs:
- Authority and trust boundary:
- Owned state and canonical source of truth:
- Lifecycle, failure/termination behavior and required invariants:
- Dependencies and cross-slice workflows:

| Requirement ID | Required behavior or constraint | Independent authority/reference | Established or unresolved |
| --- | --- | --- | --- |

## Reviewed files and responsibilities

| Source path at recorded revision | Exact lines / symbols / responsibility | Owning slice and current package/crate | Lifecycle disposition | Inspected, reused or pending | Evidence / finding references |
| --- | --- | --- | --- | --- | --- |

- Outstanding files, intervals and shared-boundary reviews:
- Shared-boundary queue entries and exchanged evidence:

## Caller and workflow traces

| Workflow ID | Established operation / entrypoint | Actual caller chain and state/authority transitions | Related slices / review records | Requirement references | Evidence, limits and outstanding boundaries |
| --- | --- | --- | --- | --- | --- |

## Assumptions and test assessment

| Assumption or assertion | Requirement ID | Actual implementation/caller or test | Supporting/contradicting evidence and limits | Supported, contradicted or unresolved |
| --- | --- | --- | --- | --- |

- Test subjects, substitutions, fixture prerequisites/skips and missing proof:

## Language and exact target allocation

| Target allocation ID | Responsibility / source units | Current owner and language/substrate | Target package/crate/module and language/substrate | Exact target files / retirement disposition | Action and order | Hosting executable/process | Decision or recommendation reference |
| --- | --- | --- | --- | --- | --- | --- | --- |

- Go/Rust fit and alternative assessment for each implementation owner:
- Process topology and explicit justification for any proposed change:
- Conflicting guidance or undecided language/ownership/placement:

## Concrete findings

### <slice-id>-F<number> <concrete finding>

- Classification and disposition: open, decision needed, correction specified, or resolved with evidence
- Observed assumption/behavior and exact source:
- Required behavior: requirement ID and independent authority; for an unresolved requirement, conflicting sources and the exact missing decision
- Affected caller chain, responsibilities and related slices:
- Supported consequence; observed or inferred:
- Evidence and limits:
- Proposed correction and exact target-allocation references:
- Correcting revision and resolution evidence, if resolved:

Use `none found within reviewed scope` when applicable.

## Required corrections and cutover

| Action / finding ID | Exact changed and target paths | Retained successor / retired units | Caller, data/wire, test and manifest/service/build/install wiring | Prerequisites and cutover order | Planned verification and required outcome | Specified or decision pending |
| --- | --- | --- | --- | --- | --- | --- |

## Performed evidence and limits

| Evidence ID / class | Inspected source or actual command/journey | Source/artifact revision and fixture/substitutions | Actual result / retained receipt | Supported scope and unverified behavior |
| --- | --- | --- | --- | --- |

## Open decisions and remaining work

| Decision / remaining-work ID | Exact unresolved question or missing review/proof | Conflicting sources or evidence limit | Affected responsibility/action and readiness |
| --- | --- | --- | --- |

- Plan sections updated from supported conclusions:
- Independent review result and outstanding challenges:
- Unresolved questions and the exact dependent implementation instructions blocked:
```
