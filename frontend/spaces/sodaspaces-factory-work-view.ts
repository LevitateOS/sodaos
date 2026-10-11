import {html} from 'lit';
import type {TemplateResult} from 'lit';
import {nativePath} from './sodaspaces-factory-activity.js';
import {id} from './sodaspaces-api.js';
import type {FactoryIssueDetail} from './sodaspaces-factory-work-response.js';

// Pure work-detail and intervention presentation for V03/V04. Model
// builders turn one issue's recorded facts into display state; render
// helpers project those models to Lit templates. No transport, identity
// lookup or mutable state lives here.

export function nativeIssueLink(repositoryPath: string, issue: string): string | undefined {
  if (!nativePath(repositoryPath) || !id(issue)) return undefined;
  return `${repositoryPath}/issues/${issue}`;
}

export function nativePullLink(repositoryPath: string, prNumber: string): string | undefined {
  if (!nativePath(repositoryPath) || !id(prNumber)) return undefined;
  return `${repositoryPath}/pulls/${prNumber}`;
}

export type DeliveryState =
  | 'delivered'
  | 'merge_confirmed_closure_pending'
  | 'checks_failed'
  | 'checks_pending'
  | 'evidence_refused'
  | 'cancelled'
  | 'needs_attention'
  | 'waiting'
  | 'working'
  | 'queued'
  | 'not_authorized'
  | 'recorded_complete_without_merge'
  | 'unrecorded';

export interface WorkDetailModel {
  readonly delivery: DeliveryState;
  readonly deliveryText: string;
  readonly acceptanceText: string;
  readonly acceptanceInvalid: boolean;
  readonly readinessText: string;
  readonly checksText: string;
  readonly mergeText: string;
  readonly failedChecks: readonly string[];
  readonly blockers: readonly {code: string; resolution: string; endpoint?: string}[];
  readonly candidate: string;
  readonly issueLink: string | undefined;
  readonly pullLink: string | undefined;
}

function blockerEndpoint(blocker: {endpoint_repository?: string; endpoint_issue?: string}): string | undefined {
  if (blocker.endpoint_repository === undefined || blocker.endpoint_issue === undefined) return undefined;
  return `repository ${blocker.endpoint_repository} issue #${blocker.endpoint_issue}`;
}

function describeBlocker(blocker: {
  code: string;
  resolution: string;
  endpoint_repository?: string;
  endpoint_issue?: string;
}): {code: string; resolution: string; endpoint?: string} {
  const endpoint = blockerEndpoint(blocker);
  return {code: blocker.code, resolution: blocker.resolution, ...(endpoint === undefined ? {} : {endpoint})};
}

function mergeDelivery(detail: FactoryIssueDetail): DeliveryState | undefined {
  if (detail.merge?.stage !== 'merged') return undefined;
  return detail.readiness?.readiness === 'completed' ? 'delivered' : 'merge_confirmed_closure_pending';
}

function checksDelivery(detail: FactoryIssueDetail): DeliveryState | undefined {
  if (detail.checks?.verdict === 'failed') return 'checks_failed';
  if (detail.checks?.verdict === 'pending') return 'checks_pending';
  if (detail.checks?.verdict === 'refused') return 'evidence_refused';
  return undefined;
}

function readinessDelivery(detail: FactoryIssueDetail): DeliveryState {
  switch (detail.readiness?.readiness) {
    case 'cancelled':
      return 'cancelled';
    case 'needs_intervention':
      return 'needs_attention';
    case 'blocked':
      return 'waiting';
    case 'active':
      return 'working';
    case 'queued':
      return 'queued';
    case 'not_authorized':
      return 'not_authorized';
    case 'completed':
      return 'recorded_complete_without_merge';
    default:
      return 'unrecorded';
  }
}

function deliveryState(detail: FactoryIssueDetail): DeliveryState {
  return mergeDelivery(detail) ?? checksDelivery(detail) ?? readinessDelivery(detail);
}

const deliveryCopy: Record<DeliveryState, string> = {
  delivered: 'Delivered: merge confirmed and original-issue closure confirmed.',
  merge_confirmed_closure_pending: 'Merge confirmed, issue closure pending.',
  checks_failed: 'Checks failed on the exact reviewed candidate.',
  checks_pending: 'Awaiting required checks on the exact reviewed candidate.',
  evidence_refused: 'Check evidence refused: the candidate moved or evidence is incomplete.',
  cancelled: 'Cancelled.',
  needs_attention: 'Needs attention.',
  waiting: 'Waiting on prerequisites.',
  working: 'Working.',
  queued: 'Queued.',
  not_authorized: 'Not authorized.',
  recorded_complete_without_merge: 'Recorded complete without merge evidence.',
  unrecorded: 'No recorded work yet.',
};

function acceptanceCopy(detail: FactoryIssueDetail): {text: string; invalid: boolean} {
  if (detail.status.withdrawn) return {text: 'Acceptance withdrawn.', invalid: true};
  if (!detail.status.valid)
    return {
      text: `Acceptance invalid${detail.status.reasons.length ? `: ${detail.status.reasons.slice(0, 3).join('; ')}` : '.'}`,
      invalid: true,
    };
  return {text: 'Acceptance valid.', invalid: false};
}

function readinessCopy(detail: FactoryIssueDetail): string {
  if (!detail.readiness) return 'No readiness record.';
  return `Readiness ${detail.readiness.readiness}: ${detail.readiness.reason}.`;
}

function checksCopy(detail: FactoryIssueDetail): string {
  if (!detail.checks) return 'No check assessment recorded.';
  return `Checks ${detail.checks.verdict} on ${detail.checks.head_oid.slice(0, 12)} over base ${detail.checks.base_oid.slice(0, 12)}: ${detail.checks.resolution}`;
}

function mergeCopy(detail: FactoryIssueDetail): string {
  if (!detail.merge) return 'No merge recorded.';
  const commit = detail.merge.merged_commit ? ` at ${detail.merge.merged_commit.slice(0, 12)}` : '';
  return `Merge ${detail.merge.stage}${commit}: ${detail.merge.resolution}`;
}

function failedCheckContexts(detail: FactoryIssueDetail): readonly string[] {
  return (detail.checks?.results ?? []).filter((result) => !result.passed).map((result) => result.context);
}
function detailCandidate(detail: FactoryIssueDetail): string {
  if (!detail.checks) return 'No reviewed candidate recorded.';
  return `PR #${detail.checks.pr_number} head ${detail.checks.head_oid.slice(0, 12)} over base ${detail.checks.base_oid.slice(0, 12)}`;
}
function detailPullLink(detail: FactoryIssueDetail, repositoryPath: string): string | undefined {
  const prNumber = detail.checks?.pr_number ?? detail.merge?.pr_number;
  if (prNumber === undefined) return undefined;
  return nativePullLink(repositoryPath, prNumber);
}
// describeIssueDetail projects one issue's recorded facts to display
// state. Delivery needs both merge and closure evidence; every other
// state names the recorded verdict behind it.
export function describeIssueDetail(detail: FactoryIssueDetail, repositoryPath: string): WorkDetailModel {
  const acceptance = acceptanceCopy(detail);
  const delivery = deliveryState(detail);
  return {
    delivery,
    deliveryText: deliveryCopy[delivery],
    acceptanceText: acceptance.text,
    acceptanceInvalid: acceptance.invalid,
    readinessText: readinessCopy(detail),
    checksText: checksCopy(detail),
    mergeText: mergeCopy(detail),
    failedChecks: failedCheckContexts(detail),
    blockers: (detail.readiness?.blockers ?? []).map(describeBlocker),
    candidate: detailCandidate(detail),
    issueLink: nativeIssueLink(repositoryPath, detail.issue),
    pullLink: detailPullLink(detail, repositoryPath),
  };
}

export function renderWorkDetail(model: WorkDetailModel): TemplateResult {
  return html`
    <section aria-label="Work detail">
      <p role="status">${model.deliveryText}</p>
      <p>${model.acceptanceText}</p>
      <p>${model.readinessText}</p>
      <ul>
        ${model.blockers.map(
          (blocker) =>
            html`<li>${blocker.code}: ${blocker.resolution}${blocker.endpoint ? ` (${blocker.endpoint})` : ''}</li>`
        )}
      </ul>
      <p>${model.checksText}</p>
      ${model.failedChecks.length > 0 ? html`<p>Failed checks: ${model.failedChecks.join(', ')}.</p>` : ''}
      <p>${model.mergeText}</p>
      <p>Reviewed candidate: ${model.candidate}</p>
      <p>
        ${model.issueLink ? html`<a href=${model.issueLink}>Native issue</a>` : 'Native issue link unavailable.'}
        ${model.pullLink ? html`<a href=${model.pullLink}>Native pull request</a>` : ''}
      </p>
    </section>
  `;
}

export type InterventionKind = 'pause' | 'resume' | 'stop' | 'retry' | 'takeover' | 'project_stop';
export type InterventionResult = 'idle' | 'pending' | 'confirmed' | 'uncertain' | 'failed';

export interface InterventionModel {
  readonly kind: InterventionKind;
  readonly label: string;
  readonly scope: string;
  readonly impact: string;
  readonly enabled: boolean;
  readonly result: InterventionResult;
  readonly resultText: string;
}

export interface InterventionInput {
  readonly kind: InterventionKind;
  readonly enabled: boolean;
  readonly result: InterventionResult;
  readonly resultText: string;
  readonly target: string;
}

const interventionCopy: Record<InterventionKind, {label: string; scope: string; impact: string}> = {
  pause: {
    label: 'Pause Factory work',
    scope: 'This repository',
    impact: 'Closes dispatch and stops outstanding runs. Retained work and terminals stay.',
  },
  resume: {
    label: 'Resume Factory work',
    scope: 'This repository',
    impact: 'Reopens dispatch after every recorded run settles.',
  },
  stop: {
    label: 'Stop run',
    scope: 'This run only',
    impact: 'Retires the run through the operator stop path. Other runs continue.',
  },
  retry: {
    label: 'Retry run',
    scope: 'This run only',
    impact: 'Queues one explicit retry after the prior run settles. No new attempt otherwise.',
  },
  takeover: {
    label: 'Take over run',
    scope: 'This run into your checkout',
    impact: 'Copies retained work into your own project checkout. Requires project membership.',
  },
  project_stop: {
    label: 'Stop Project for everyone',
    scope: 'The whole project',
    impact: 'Stops shared project services for everyone. Needs explicit confirmation.',
  },
};

// describeIntervention pairs one authorized control with its exact scope,
// impact and pending-versus-confirmed result. Requested actions never
// read as confirmed effects.
export function describeIntervention(input: InterventionInput): InterventionModel {
  const copy = interventionCopy[input.kind];
  return {
    kind: input.kind,
    label: copy.label,
    scope: `${copy.scope}: ${input.target}`,
    impact: copy.impact,
    enabled: input.enabled,
    result: input.result,
    resultText: input.resultText,
  };
}

export interface InterventionCommands {
  readonly act: (kind: InterventionKind) => void;
}

export function renderInterventions(
  models: readonly InterventionModel[],
  commands: InterventionCommands
): TemplateResult {
  return html`
    <section aria-label="Factory interventions">
      <ul>
        ${models.map(
          (model) => html`<li>
            <button
              type="button"
              class="ui button"
              ?disabled=${!model.enabled}
              @click=${() => commands.act(model.kind)}
            >
              ${model.label}
            </button>
            <p>${model.scope}. ${model.impact}</p>
            <p role="status">${model.resultText}</p>
          </li>`
        )}
      </ul>
    </section>
  `;
}
