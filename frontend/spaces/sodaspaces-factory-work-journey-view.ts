import {html} from 'lit';
import type {TemplateResult} from 'lit';
import {nativeIssueLink, nativePullLink} from './sodaspaces-factory-work-view.js';
import type {
  JourneyAdoption,
  JourneyCommit,
  JourneyCorrection,
  JourneyFinding,
  JourneyGate,
  JourneyJudgment,
  JourneyPhase,
  JourneyTask,
  WorkJourney,
} from './sodaspaces-factory-work-journey-response.js';

// Pure fixture work-journey presentation for FW23. Model builders turn
// one issue's recorded-shaped facts into display state; render helpers
// project those models to Lit templates. No transport, identity lookup
// or mutable state lives here. Actual persisted readback binding belongs
// to FW25; fixture text must never be read as the journey.

export function describeJourneyPhase(phase: JourneyPhase): string {
  return `Phase ${phase.phase}: ${phase.state}.`;
}

export interface JourneyTaskModel {
  readonly text: string;
  readonly link: string | undefined;
}

export function describeJourneyTask(task: JourneyTask, repositoryPath: string): JourneyTaskModel {
  const lane = task.lane === 'unassigned' ? 'no lane yet' : `lane ${task.lane}`;
  const issue = task.issue === undefined ? 'no native issue mapped yet' : `issue #${task.issue}`;
  return {
    text: `Task ${task.key} (${lane}): ${task.state}, ${issue}.`,
    link: task.issue === undefined ? undefined : nativeIssueLink(repositoryPath, task.issue),
  };
}

export interface JourneyCommitModel {
  readonly text: string;
  readonly pullLink: string | undefined;
}

export function describeJourneyCommit(commit: JourneyCommit, repositoryPath: string): JourneyCommitModel {
  return {
    text: `Candidate ${commit.commit.slice(0, 12)} over base ${commit.base.slice(0, 12)}: PR #${commit.pr_number}.`,
    pullLink: nativePullLink(repositoryPath, commit.pr_number),
  };
}

export function describeJourneyFinding(finding: JourneyFinding): string {
  return `${finding.severity} finding ${finding.key.slice(0, 9)} needs ${finding.fix_kind}: ${finding.defect}`;
}

export function describeJourneyCorrection(correction: JourneyCorrection): string {
  if (correction.candidate === undefined) return 'Correction pending: no candidate published yet.';
  const effect = correction.effect === undefined ? 'effect unrecorded' : `effect ${correction.effect}`;
  const completion = correction.completion === undefined ? '' : `, ${correction.completion}`;
  return `Correction ${correction.candidate.slice(0, 12)}: ${effect}${completion}.`;
}

const judgmentKindCopy: Record<string, string> = {
  priority: 'Next-work priority',
  delivery: 'Delivery acceptance',
  'reassessment-decision': 'D13 completion decision',
  'reassessment-proposal': 'D13 reassessment proposal',
};

export function describeJourneyJudgment(judgment: JourneyJudgment): string {
  const kind = judgmentKindCopy[judgment.kind] ?? 'Judgment';
  const choice = `${kind} selected ${judgment.selected}.`;
  if (!judgment.wait) return `${choice} No wait recorded.`;
  return `${choice} Waiting: ${judgment.wait_reason}. This wait is not a confirmed new attempt.`;
}

export interface JourneyGateModel {
  readonly gateText: string;
  readonly proposalText: string;
  readonly waitText: string;
}

function gateDecisionText(gate: JourneyGate): string {
  if (gate.decision === undefined) return 'no gate decision recorded';
  return `gate decision ${gate.decision}`;
}

function gateProposalText(gate: JourneyGate): string {
  if (gate.proposal === undefined) return 'No proposal assessed yet.';
  const digest = gate.proposal_digest === undefined ? '' : ` (proposal ${gate.proposal_digest.slice(0, 12)})`;
  return `Proposal ${gate.proposal}${digest}.`;
}

function gateWaitText(gate: JourneyGate): string {
  if (gate.wait_reason === undefined) return '';
  const retry = gate.retryable ? ' An explicit retry can proceed.' : '';
  return `Waiting: ${gate.wait_reason}.${retry}`;
}

// describeJourneyGate projects one D13 completion gate to display
// state: the fenced completion cycle, its gate decision and proposal,
// and the bounded wait/retry reason. Retryable reads only on attention.
export function describeJourneyGate(gate: JourneyGate): JourneyGateModel {
  return {
    gateText:
      `Completion ${gate.completion_id.slice(0, 9)} cycle ${gate.cycle_id.slice(0, 9)} ` +
      `(generation ${gate.generation}): ${gate.state}, ${gateDecisionText(gate)}.`,
    proposalText: gateProposalText(gate),
    waitText: gateWaitText(gate),
  };
}

export function describeJourneyAdoption(adoption: JourneyAdoption): string {
  return (
    `Proposal adoption ${adoption.state}: intent ${adoption.intent.slice(0, 9)} ` +
    `at revision ${adoption.revision.slice(0, 9)}. Adoption names the bound revision only.`
  );
}

export interface WorkJourneyModel {
  readonly headline: string;
  readonly phases: readonly string[];
  readonly tasks: readonly JourneyTaskModel[];
  readonly commits: readonly JourneyCommitModel[];
  readonly findings: readonly string[];
  readonly corrections: readonly string[];
  readonly judgments: readonly string[];
  readonly gates: readonly JourneyGateModel[];
  readonly adoptions: readonly string[];
  readonly observedText: string;
}

function journeyHeadline(journey: WorkJourney): string {
  const parts: string[] = [];
  if (journey.tasks.length > 0) parts.push(`${journey.tasks.length} tasks`);
  if (journey.findings.length > 0) parts.push(`${journey.findings.length} findings`);
  if (journey.gates.length > 0) parts.push(`${journey.gates.length} completion gates`);
  return parts.length > 0 ? `Work journey: ${parts.join(' · ')}.` : 'No journey sections recorded.';
}

// describeWorkJourney projects one fixture journey to display state.
// Every section reflects only its own facts; missing sections stay
// silent instead of inventing progress.
export function describeWorkJourney(journey: WorkJourney, repositoryPath: string): WorkJourneyModel {
  return {
    headline: journeyHeadline(journey),
    phases: journey.phases.map(describeJourneyPhase),
    tasks: journey.tasks.map((task) => describeJourneyTask(task, repositoryPath)),
    commits: journey.commits.map((commit) => describeJourneyCommit(commit, repositoryPath)),
    findings: journey.findings.map(describeJourneyFinding),
    corrections: journey.corrections.map(describeJourneyCorrection),
    judgments: journey.judgments.map(describeJourneyJudgment),
    gates: journey.gates.map(describeJourneyGate),
    adoptions: journey.adoptions.map(describeJourneyAdoption),
    observedText: `Observed ${journey.observed_unix}.`,
  };
}

export function renderWorkJourney(model: WorkJourneyModel): TemplateResult {
  return html`
    <section aria-label="Work journey">
      <p role="status">${model.headline}</p>
      <ul>
        ${model.phases.map((text) => html`<li>${text}</li>`)}
      </ul>
      <ul>
        ${model.tasks.map(
          (task) => html`<li>${task.text} ${task.link ? html`<a href=${task.link}>Open issue</a>` : ''}</li>`
        )}
      </ul>
      <ul>
        ${model.commits.map(
          (commit) =>
            html`<li>
              ${commit.text} ${commit.pullLink ? html`<a href=${commit.pullLink}>Open pull request</a>` : ''}
            </li>`
        )}
      </ul>
      <ul>
        ${model.findings.map((text) => html`<li>${text}</li>`)}
      </ul>
      <ul>
        ${model.corrections.map((text) => html`<li>${text}</li>`)}
      </ul>
      <ul>
        ${model.judgments.map((text) => html`<li>${text}</li>`)}
      </ul>
      <ul>
        ${model.gates.map(
          (gate) => html`<li>${gate.gateText} ${gate.proposalText} ${gate.waitText ? html`${gate.waitText}` : ''}</li>`
        )}
      </ul>
      <ul>
        ${model.adoptions.map((text) => html`<li>${text}</li>`)}
      </ul>
      <p>${model.observedText}</p>
    </section>
  `;
}
