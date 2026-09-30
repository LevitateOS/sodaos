# Factory implementation plan draft

The planning target is Soda's automatic issue → readiness → coding → PR →
review/fix → native merge → unblock loop, running in persistent Projects and
visible through Spaces. Fountain supplies generic native collaboration and
extension capabilities; Soda owns factory policy and environment coordination.

This draft establishes the planning baseline. Deliverables, implementation tasks
and dependency order will be derived from it; they are not yet specified here.
The current work is documentation only. The linked owning guides remain authoritative
for behavior, and this plan does not duplicate or replace their requirements.

## Source baseline

Recorded on September 30, 2026, before this planning edit:

| Repository | Exact commit | Working tree at capture |
| --- | --- | --- |
| SodaOS (`sodaos`) | `079edf0d017d595bfc277be77440267d11f166fc` | Clean |
| Fountain (`forgejo-ext`) | `c22b3543f6a1f88ede70ed3f046576b725430934` | Clean |

These revisions identify the source and contract inputs to planning, not installed
appliance versions or qualified release artifacts. Planning commits can extend
this document without changing that starting pair. Before assigning implementation
changes against later source, reconcile the affected differences; do not reset a
checkout to this baseline or discard intervening work. Toolchain and dependency
versions remain in the selected source manifests and locks.

## Owning contracts

| Subject | Authoritative planning input |
| --- | --- |
| Product outcome and feature disposition | [Factory workflow](../product/overview.md#software-factory-workflow) and the complete [scope guide](../product/scope.md), including supporting features. |
| Readiness and accepted requirements | [Blockers](../product/overview.md#readiness-and-blockers), [accepted native records](../product/overview.md#accepted-requirements-and-native-records), and their invalidation/reassessment rules. |
| Prompts, capacity and intervention | [Prompt assembly](../product/overview.md#prompt-assembly), [concurrency and usage limits](../product/overview.md#concurrency-and-usage-limits), [review/correction](../product/overview.md#review-and-correction), [merge conditions](../product/overview.md#automatic-merge-conditions) and [human intervention](../product/overview.md#human-intervention). |
| Environment and sessions | [Projects](../product/projects.md), including [approved preparation](../product/projects.md#preparing-the-environment), accounts/checkouts, creation and persistence; [Spaces](../product/spaces.md) owns session views and interaction boundaries. |
| Architecture and ownership | [Factory architecture decisions](../architecture/overview.md#factory-architecture-decisions), shared execution, native observations and persistence boundaries. |
| Authority and privilege | [Factory grants](../architecture/trust.md#factory-authority-boundary), [Project execution](../architecture/trust.md#project-execution-boundary), [preparation authority](../architecture/trust.md#project-preparation-authority) and [host helper](../architecture/trust.md#host-helper). |
| Fountain operations and recovery | [Generic consumption boundary](../architecture/trust.md#fountain-consumption-boundary) and [conditional mutations](../architecture/trust.md#conditional-native-mutations): publication, PR creation, review and merge; background authentication, callback binding, participating writers, cancellation and recovery. |
| Provider identity | [Identity Broker](../reference/credentials.md#identity-broker) and provider-specific subscription custody, concurrency, execution and return contracts. Native forge authentication remains separate. |
| Retained appliance support | [Networking](../architecture/networking.md), [runners](../reference/runners.md), [Project OS](../reference/project-os.md), [human terminals](../reference/terminal.md), [native services](../guides/project-services.md) and [operator setup](../guides/operator-setup.md), under the scope guide's retain/adapt decisions. |
| Acceptance and delivery | [Finished factory demonstration](testing.md#finished-product-demonstration), [acceptance cases](testing.md#acceptance-cases), [preparation acceptance](testing.md#project-preparation-acceptance), [conditional operation acceptance](testing.md#conditional-operation-acceptance), [evidence rules](testing.md#evidence-rules) and the [release architecture](../architecture/release.md). |
| Engineering and documentation | Repository [instructions](../../AGENTS.md), [Go ownership](go.md), [TypeScript](typescript.md), [Python tooling](python.md) and [documentation authority](../README.md#authority-rules). Use existing ownership and focused checks; keep one current implementation. |

Product and architecture guides describe the target. Reference guides describe
interfaces present in the baseline and do not override the target with the older
manual factory workflow. The [capability map](../research/factory-capability-map.md)
is source/evidence material for deriving changes, not an additional requirements
owner or proof that all mapped capabilities work together.

## Exclusions and first release limits

The complete [retire](../product/scope.md#retire), [deferred](../product/scope.md#deferred)
and [not pursuing](../product/scope.md#not-pursuing) lists remain in force. In
particular, the implementation plan must not reintroduce or silently require:

- Soda's removed forge OAuth adapter, parallel Soda login/sessions, login relays
  or Soda-managed Git credential mediation. Supported AI-provider custody and separately
  authorized native collaboration actors remain required.
- A second factory engine, mandatory manual issue admission or human merge,
  disposable containers for each stage, experimental compatibility paths,
  a standalone Soda frontend or superseded custom Cockpit/native forge patches.
- Soda factory rules inside Fountain, or an authentication workaround in Soda
  for a missing generic Fountain capability.
- Soda-provisioned local CI execution or Runner OS, extra Project OS/desktop
  profiles, private shared-tool/service branches or additional provider adapters.
- Post-merge production deployment, multihost/fleet orchestration, conversation
  snapshots/resume, hostile external execution or mutually untrusted tenancy,
  automatic destructive recovery, general archival/deletion programmes, automatic
  cross-system offboarding/key synchronization or silent identity remapping.
- Public application hosting, purchased domains or preprovisioned per-app public
  certificates as baseline prerequisites, predecessor host developer accounts,
  managed checkouts or the predecessor Updates platform.

The selected Project profile is [Rocky headless](../product/projects.md#selected-profiles)
on the CoreOS appliance. [Organization-owned Project creation](../product/projects.md#creation-and-ownership)
remains unsupported. Shared packages/tools and services use native Project
administration under the [preparation contract](../product/projects.md#preparing-the-environment);
the first path does not add a privileged repository installation service.
The first conditional merge interface supports only native
[`fast-forward-only`](../architecture/trust.md#initial-merge-methods), preserving
native protections and refusing unsupported methods without fallback.

These limits do not remove human Project access, supported provider connections,
optional private networking, persistent state, separately managed native Actions
capacity or required installation/update/recovery qualification. Their retained
contracts stay in the implementation scope.

## Evidence carried into planning

Reuse the [capability map's evidence](../research/factory-capability-map.md#evidence-scope)
at its recorded source and tested scope. The minimal-container experiment supports
process/filesystem feasibility; the conditional-merge prototype supports its
tested native path. Neither proves the full Project environment, complete native
writer coverage, authenticated production operations or the composed factory.

The [remaining integration boundaries](../research/factory-capability-map.md#remaining-decisive-boundaries)
belong in the implementation plan as early implementation and validation work.
They do not reopen settled product decisions or require a release build merely to
finish planning. Actual CLI/broker execution, factory Spaces views, lifecycle
coordination and full native qualification still need their stated evidence.

## Earlier extension work

The [earlier extension transition plan](forgejo-extensions-plan.md) retains useful
source references, scoped receipts and unfinished extension/appliance delivery
work. Its older baseline, task ordering and narrower completion definition do not
govern this factory plan. Reconcile that remaining work when deriving deliverables;
do not drop it, blindly rerun it or count old checked tasks as factory acceptance.

As that reconciliation proceeds, absorb relevant contracts into their existing
owners and remove superseded planning material. Remove this plan after the
implementation is completed and its final contracts are absorbed, following the
documentation authority rules.
