import {html} from 'lit';
import type {TemplateResult} from 'lit';
import {
  feedEntryKey,
  feedPayloadCap,
  feedSummaryLimit,
  parseFeedStatusSegments,
} from './sodaspaces-factory-feed-response.js';
import type {
  FeedCoverage,
  FeedCommunication,
  FeedEntry,
  FeedFinding,
  FeedIntentChange,
  FeedPayload,
  FeedStatusSegments,
  FeedGraphItem,
} from './sodaspaces-factory-feed-response.js';
import type {FeedRepositoryRef} from './sodaspaces-factory-feed-view.js';
import {nativeIssueLink, nativePullLink} from './sodaspaces-factory-work-view.js';
import {feedComponent, renderFeedCommunication} from './sodaspaces-factory-feed-communication.js';
import {renderFeedGraph} from './sodaspaces-factory-feed-graph.js';
import type {FeedArtifactRequest} from './sodaspaces-factory-feed-artifact.js';
import {renderRecordAction} from './sodaspaces-factory-feed-record-actions.js';

// Registered feed component families for the complete factory loop. The
// server-derived actor selects the role family and the recorded status
// selects the state treatment; the frontend never invents authorship,
// results or next actions. Records also carry optional bound issue/PR,
// candidate reference and recorded summary, rendered here as the
// native links and recorded-content lines. Unknown actors and statuses
// land on the fallback family, which preserves the recorded facts,
// failures and source links instead of defaulting to success. Catalog
// references (PL/AR/BU/IN/FX/ST/TP/K/LA/F/S) name the designed
// components each family stands for; detailed records open in the
// existing work views.

export const feedFamilies = [
  'plan',
  'issue',
  'work',
  'review',
  'correction',
  'reassessment',
  'verification',
  'integration',
  'merged',
  'completed',
  'knowledge',
  'decision',
  'question',
  'answer',
  'flag',
  'proposal',
  'progress',
  'blocked',
  'stuck',
  'incomplete',
  'maintenance',
  'discussion',
  'fallback',
] as const;
export type FeedFamily = (typeof feedFamilies)[number];

export type FeedStatusClass = 'failure' | 'waiting' | 'settled' | 'unknown';

const roleFamilies: Readonly<Record<string, FeedFamily>> = {
  product_lead: 'plan',
  architect: 'issue',
  builder: 'work',
  inspector: 'review',
  fixer: 'correction',
  strategist: 'reassessment',
  test_pilot: 'verification',
  library_admin: 'knowledge',
  researcher: 'knowledge',
  jev: 'decision',
  systemone: 'decision',
  system_one: 'decision',
  factory: 'integration',
  system: 'integration',
};

const roleLabels: Readonly<Record<string, string>> = {
  product_lead: 'Product Lead',
  architect: 'Architect',
  builder: 'Builder',
  inspector: 'Inspector',
  fixer: 'Fixer',
  strategist: 'Strategist',
  test_pilot: 'Test Pilot',
  library_admin: 'Library Admin',
  researcher: 'Researcher',
  jev: 'JEV',
  systemone: 'JEV',
  system_one: 'JEV',
  factory: 'Factory',
  system: 'Factory',
};

function normalizedActor(actor: string | undefined): string {
  return (actor ?? '')
    .trim()
    .toLowerCase()
    .replace(/[\s-]+/g, '_');
}

// roleLabel renders a canonical actor id readably; free-form server
// labels pass through unchanged.
export function roleLabel(actor: string | undefined): string | undefined {
  if (actor === undefined) return undefined;
  return roleLabels[normalizedActor(actor)] ?? actor;
}

// classifyFeedStatus buckets one recorded status without upgrading it:
// only an explicit recorded completion reads as settled, and anything
// unrecognised stays unknown rather than successful. Composed segment
// statuses classify by their recorded segments: a failed outcome or
// recorded failures lead, a finished succeeded stage settles, any
// other recorded stage is still in progress.
export function classifyFeedStatus(status: string | undefined): FeedStatusClass {
  if (status === undefined) return 'unknown';
  const text = status.trim().toLowerCase();
  if (text === '') return 'unknown';
  if (
    /(?:^|:)(?:failed|invalid|blocked|stuck|unconfirmed|cancelled|canceled|needs-human|stopped|request-changes|clarify)(?=:|$)/.test(
      text
    )
  )
    return 'failure';
  if (/^route:(?:discussion|existing_work)$/.test(text)) return 'settled';
  // A recorded run end is settled (shown neutral, not as done work); an
  // assignment only says the work started and may still be running.
  if (text === 'run:succeeded') return 'settled';
  if (text === 'dispatch:assigned') return 'waiting';
  if (text === 'feedback:delivered') return 'settled';
  if (/^(?:judgment:applied|draft:valid|effect:created)$/.test(text)) return 'settled';
  if (text.includes(':')) {
    const segments = parseFeedStatusSegments(text);
    if (segments.phase !== undefined) {
      if (segments.outcome && /fail|refus|reject|error/.test(segments.outcome)) return 'failure';
      if (segments.failures !== undefined && segments.failures > 0) return 'failure';
      if (segments.verdict && /fail/.test(segments.verdict)) return 'failure';
      if (segments.stage === 'finished') return segments.outcome === 'succeeded' ? 'settled' : 'unknown';
      return 'waiting';
    }
  }
  if (
    /fail|error|refus|reject|invalid|blocked|stuck|unavail|uncertain|incomplete|stale|missing|denied|conflict|exhaust|revoked|deleted/.test(
      text
    )
  )
    return 'failure';
  if (
    /pending|waiting|queued|running|assess|verify|verifying|submitted|proposed|draft|progress|applying|working|recorded|open/.test(
      text
    )
  )
    return 'waiting';
  if (/verified|applied|accepted|complete|confirm|merged|closed|resolved|delivered|clear|passed|adopted/.test(text))
    return 'settled';
  return 'unknown';
}

// selectFeedFamily picks the registered renderer for one entry. Posts
// are discussion: their stored text and pipeline outcome are the
// content. Records follow their server-derived actor; a recorded
// question or decision status names its shared family when no role
// actor is present. Everything else keeps the honest fallback.
export function selectFeedFamily(entry: FeedEntry): FeedFamily {
  if (entry.payload?.invalid) return 'fallback';
  const communication = entry.payload?.communication;
  if (communication) {
    const bySignal: Record<FeedCommunication['kind'], FeedFamily> = {
      ask: 'question',
      answer: 'answer',
      reply: 'discussion',
      flag: 'flag',
      learned: 'knowledge',
      suggest: 'proposal',
      partial: 'progress',
      blocked: 'blocked',
      stuck: 'stuck',
      record_decision: 'decision',
      finish_incomplete: 'incomplete',
      system: 'integration',
      library_maintenance: 'maintenance',
      artifact: 'work',
      journey: 'verification',
      diagnosis: 'reassessment',
      repair_draft: 'reassessment',
    };
    const component = feedComponent(communication.componentId);
    if (communication.kind === 'partial' && component && feedFamilies.some((family) => family === component.family))
      return component.family as FeedFamily;
    return bySignal[communication.kind];
  }
  if (entry.kind === 'post') return 'discussion';
  // Milestone entries are their own steps (Lane B3): a confirmed merge
  // and a verified completion get their own posts, and a recorded Jev
  // judgment is the decision family whatever its actor spelling.
  if (entry.activity === 'merged') return 'merged';
  if (entry.activity === 'completed') return 'completed';
  if (entry.activity === 'judgment') return 'decision';
  const component = feedComponent(entry.payload?.componentId);
  if (component && feedFamilies.some((family) => family === component.family)) return component.family as FeedFamily;
  const byRole = roleFamilies[normalizedActor(entry.actor)];
  if (byRole !== undefined) return byRole;
  const status = (entry.status ?? '').toLowerCase();
  if (/question|clarif/.test(status)) return 'question';
  if (/decision|decided|judgment|judgement/.test(status)) return 'decision';
  return 'fallback';
}

export interface FeedFamilyModel {
  readonly payload?: FeedPayload;
  readonly entryKey?: string;
  readonly communication?: FeedCommunication;
  readonly componentId?: string;
  readonly family: FeedFamily;
  readonly label: string;
  readonly headline: string;
  readonly detailText: string;
  readonly stateText: string;
  // Recorded-content lines from the additive wire fields: the bounded
  // recorded result summary and the candidate reference the record
  // produced. Absent when the entry does not carry them.
  readonly summaryText: string | undefined;
  readonly refText: string | undefined;
  readonly nextText: string;
  readonly statusClass: FeedStatusClass;
  // A run that ended is settled but neutral: the run finishing says
  // nothing yet about whether the work it carried is done.
  readonly neutral?: boolean;
  readonly sourceLinks: readonly {label: string; href: string}[];
  // Designed role component blocks (catalog PL/AR/BU/IN/FX/ST/TP/K/S):
  // the focal structured section per family, built only from recorded
  // wire facts — status segments, summary, candidate ref, bound
  // issue/PR. A block with no recorded facts says so per row instead
  // of rendering invented plan steps, checks or coverage.
  readonly blocks: readonly FeedBlock[];
}

// One structured block of a family section. Steps and chain render as
// ordered lists with per-step recorded state; rows as a definition
// list; checks/coverage as a table; beforeAfter as two labelled
// panels; brief as the recorded summary text. Step/row states are
// words ('done', 'current', 'waiting', 'failed', 'not recorded'),
// never colour alone.
export type FeedBlock = (
  | {kind: 'brief'; heading: string; text: string}
  | {kind: 'alert'; heading: string; text: string}
  | {kind: 'steps'; heading: string; steps: readonly {label: string; state: string}[]}
  | {kind: 'graph'; heading: string; items: readonly FeedGraphItem[]}
  | {kind: 'rows'; heading: string; rows: readonly {term: string; value: string}[]}
  | {kind: 'checks'; heading: string; rows: readonly {name: string; result: string}[]}
  | {kind: 'details'; heading: string; text: string}
  | {
      kind: 'beforeAfter';
      heading: string;
      beforeLabel: string;
      before: string;
      afterLabel: string;
      after: string;
    }
) & {readonly note?: string | undefined};

const familyText: Readonly<Record<FeedFamily, {label: string; detail: string}>> = {
  plan: {
    label: 'Product plan',
    detail:
      'Direction, outcomes and the proposed issue graph open in the existing product views. A submitted proposal is not adopted work until the native issues are confirmed.',
  },
  issue: {
    label: 'Issue investigation and plan',
    detail:
      'Inspected source, the approach and the task plan open in the existing work views. A submitted plan is not a running Builder until work is actually admitted.',
  },
  work: {
    label: 'Implementation work',
    detail:
      'Task progress, changes, checks and checkpoints open in the existing work views. Only trusted check and checkpoint receipts establish what actually works.',
  },
  review: {
    label: 'Independent review',
    detail:
      'Coverage, findings and their evidence open in the existing review views. No findings reads as clear only when the required coverage is complete.',
  },
  correction: {
    label: 'Correction',
    detail:
      'Each finding, its disposition and regression evidence open in the existing review views. A submitted fix stays submitted until independent acceptance confirms it.',
  },
  reassessment: {
    label: 'Reassessment and repair',
    detail:
      'The diagnosed cause, proposed plan changes and protected work open in the existing work views. A proposed repair is not the current graph until adoption is confirmed.',
  },
  verification: {
    label: 'Application verification',
    detail:
      'Outcome coverage, journeys and check receipts for the exact delivery open in the existing work views. Executed checks passing does not cover outcomes never exercised.',
  },
  integration: {
    label: 'Integration and delivery',
    detail:
      'Publication, merge and issue-closure evidence opens at the native records. Each confirmed effect stays distinct: created is not merged, and verified is not deployed.',
  },
  merged: {
    label: 'Merge confirmed',
    detail:
      'One confirmed native merge, posted by Factory as its own step. The merged commit and any closing of the original issue stay distinct recorded effects.',
  },
  completed: {
    label: 'Issue completed',
    detail:
      'The bound issue was completed after its merged change was confirmed. Application verification is recorded separately for the accepted outcomes.',
  },
  knowledge: {
    label: 'Knowledge',
    detail: 'Recorded knowledge and its supporting evidence. Open the linked source to inspect the original context.',
  },
  decision: {
    label: 'Recorded decision',
    detail:
      'The chosen option, its criteria and evidence open at the recorded decision. Logging a choice does not by itself apply a plan or change an issue.',
  },
  question: {
    label: 'Question',
    detail:
      'The question, its consequence and answer history stay with the affected work. An answered question still needs its answer incorporated before dependent work relies on it.',
  },
  answer: {
    label: 'Answer',
    detail:
      'The answer is recorded with the question. Dependent work reports separately when it incorporates the answer.',
  },
  flag: {label: 'Concern raised', detail: 'A worker has reported a concern with its impact and evidence.'},
  proposal: {
    label: 'Proposal',
    detail: 'A proposed change remains a proposal until the responsible workflow adopts it.',
  },
  progress: {
    label: 'Progress update',
    detail: 'The worker reports completed and remaining work. Confirmed workflow results are recorded separately.',
  },
  blocked: {
    label: 'Blocked',
    detail: 'Progress depends on a known missing input. The release condition appears in this post.',
  },
  stuck: {
    label: 'Diagnosis needed',
    detail: 'The worker cannot find a working route. Prior attempts help the next worker continue the diagnosis.',
  },
  incomplete: {
    label: 'Incomplete handoff',
    detail: 'Useful work is preserved alongside the unresolved work and the reason it ended.',
  },
  maintenance: {
    label: 'Library maintenance',
    detail:
      'Item changes retain their evidence, scope and history. A proposal does not mean a library operation was applied.',
  },
  discussion: {
    label: 'Discussion',
    detail: 'The stored discussion text appears above with its replies and recorded outcome below.',
  },
  fallback: {
    label: 'Recorded activity',
    detail:
      'Detailed records open in the existing work views. This entry keeps its recorded facts and links until a dedicated presentation covers it.',
  },
};

function familyLinks(
  family: FeedFamily,
  repository: FeedRepositoryRef,
  entry: FeedEntry
): {label: string; href: string}[] {
  if (repository.path === undefined) return [];
  const links = [{label: `Open ${repository.label}`, href: repository.path}];
  // The bound native records, when the wire carries them, replace the
  // generic indexes; without them no issue or PR number is invented.
  const issueLink = entry.issue ? nativeIssueLink(repository.path, String(entry.issue)) : undefined;
  if (issueLink) links.push({label: `Open issue #${entry.issue}`, href: issueLink});
  else if (
    family === 'plan' ||
    family === 'issue' ||
    family === 'question' ||
    family === 'reassessment' ||
    family === 'completed'
  )
    links.push({label: 'Open issue index', href: `${repository.path}/issues`});
  const pullLink = entry.pr ? nativePullLink(repository.path, String(entry.pr)) : undefined;
  if (pullLink) links.push({label: `Open pull request #${entry.pr}`, href: pullLink});
  else if (
    family === 'work' ||
    family === 'review' ||
    family === 'correction' ||
    family === 'integration' ||
    family === 'merged'
  )
    links.push({label: 'Open pull requests', href: `${repository.path}/pulls`});
  return links;
}

function nextTextFor(statusClass: FeedStatusClass): string {
  if (statusClass === 'failure')
    return 'What follows: this needs attention before dependent work can rely on it. The failure stays visible beside any preserved work.';
  if (statusClass === 'waiting')
    return 'What follows: recorded and still in progress. Its confirmed outcome replaces this state when the owning record lands.';
  if (statusClass === 'settled')
    return 'What follows: the recorded outcome stands as shown; later stages report separately under their own authors.';
  return 'What follows: outcome unknown from this entry alone. Open the source record before treating it as done.';
}

function humanSegment(segment: string): string {
  return segment.replace(/[-_]+/g, ' ');
}

function stageState(segments: FeedStatusSegments): string {
  if (segments.outcome && /fail|refus|reject|error|cancel|needs-human|stopped/i.test(segments.outcome))
    return 'stopped';
  if (segments.stage === 'finished') return segments.outcome === 'succeeded' ? 'done' : 'not completed';
  if (segments.stage) return 'current';
  return 'not recorded';
}

function chainSteps(segments: FeedStatusSegments, labels: readonly string[]): {label: string; state: string}[] {
  const reached = new Set(segments.chain.map((step) => step.toLowerCase()));
  return labels.map((label) => ({
    label,
    state:
      reached.has(label.toLowerCase().replace(/\s+/g, '')) || reached.has(label.toLowerCase()) ? 'done' : 'waiting',
  }));
}

function checkRows(segments: FeedStatusSegments): {name: string; result: string}[] {
  const rows: {name: string; result: string}[] = [];
  if (segments.checks !== undefined)
    rows.push({
      name: 'Recorded check receipts',
      result: `${segments.checks} receipt${segments.checks === 1 ? '' : 's'}`,
    });
  if (segments.failures !== undefined)
    rows.push({
      name: 'Recorded failures',
      result: segments.failures === 0 ? 'none recorded' : `${segments.failures} failed`,
    });
  if (segments.checks !== undefined && segments.failures !== undefined) {
    const passed = segments.checks - segments.failures;
    rows.push({
      name: 'Passed (receipts minus failures)',
      result: passed >= 0 ? `${passed}` : 'counts inconsistent; see source record',
    });
  }
  if (rows.length === 0) rows.push({name: 'Checks', result: 'not recorded on this entry'});
  return rows;
}

// familyBlocks builds the designed focal component per family from the
// recorded segments and content. Each family has its own structure —
// plan brief + adoption steps, investigation rows, checkpoint/check
// table, finding rows, before/after panels, adoption chain, coverage
// table, delivery chain, lesson/decision/question rows — never one
// shared paragraph shape with swapped nouns.
function familyBlocks(family: FeedFamily, entry: FeedEntry, segments: FeedStatusSegments): FeedBlock[] {
  const summary = entry.summary;
  const brief = (heading: string): FeedBlock[] => (summary ? [{kind: 'brief', heading, text: summary}] : []);
  const phaseRows: {term: string; value: string}[] = [
    {term: 'Phase', value: segments.phase ? humanSegment(segments.phase) : 'not recorded'},
    {term: 'Stage', value: segments.stage ? humanSegment(segments.stage) : 'not recorded'},
    {term: 'Outcome', value: segments.outcome ? humanSegment(segments.outcome) : 'not recorded'},
  ];
  const issueValue = entry.issue ? `issue #${entry.issue} (bound native issue)` : 'no bound issue on this entry';
  const prValue = entry.pr
    ? `pull request #${entry.pr} (bound native pull request)`
    : 'no bound pull request on this entry';
  const refValue = entry.ref ? entry.ref : 'not recorded on this entry';
  switch (family) {
    case 'plan':
      return [
        ...brief('Proposed direction (PL01 brief)'),
        {
          kind: 'steps',
          heading: 'Proposal adoption (PL08/PL11)',
          steps: [
            {label: 'Product planning', state: stageState(segments)},
            ...(segments.outcome
              ? [{label: `Recorded outcome: ${humanSegment(segments.outcome)}`, state: stageState(segments)}]
              : []),
            {
              label: 'Native issues confirmed from the proposal',
              state: entry.issue ? 'done' : 'waiting — a submitted proposal is not adopted work',
            },
          ],
        },
        {kind: 'rows', heading: 'Resulting work', rows: [{term: 'Bound work', value: issueValue}, ...phaseRows]},
      ];
    case 'issue':
      return [
        ...brief('Investigation and approach (AR01/AR02)'),
        {
          kind: 'rows',
          heading: 'Plan binding (AR07/AR11)',
          rows: [
            {term: 'Investigated issue', value: issueValue},
            {term: 'Candidate reference', value: refValue},
            ...phaseRows,
          ],
        },
        {
          kind: 'steps',
          heading: 'Plan handoff',
          steps: [
            {label: 'Investigation', state: stageState(segments)},
            {
              label: 'Builder admitted on the accepted plan',
              state:
                segments.chain.length > 0
                  ? `recorded chain: ${segments.chain.join(', ')}`
                  : 'waiting — a submitted plan is not a running Builder',
            },
          ],
        },
      ];
    case 'work':
      return [
        ...brief('Implementation summary (BU04)'),
        {kind: 'checks', heading: 'Checks and checkpoint evidence (BU06/BU07)', rows: checkRows(segments)},
        {
          kind: 'rows',
          heading: 'Candidate',
          rows: [
            {term: 'Candidate reference', value: refValue},
            {term: 'Bound issue', value: issueValue},
            {term: 'Bound pull request', value: prValue},
            ...phaseRows,
          ],
        },
      ];
    case 'review':
      return [
        ...brief('Review summary (IN10)'),
        {
          kind: 'rows',
          heading: 'Findings and verdict (IN02/IN03)',
          rows: [
            {term: 'Verdict', value: segments.verdict ? humanSegment(segments.verdict) : 'not recorded on this entry'},
            {
              term: 'Findings',
              value:
                segments.findings !== undefined
                  ? `${segments.findings} recorded finding${segments.findings === 1 ? '' : 's'} — each opens in the native review`
                  : 'count not recorded; “nothing found” is not “nothing inspected”',
            },
            {term: 'Reviewed candidate', value: refValue},
            {term: 'Bound pull request', value: prValue},
            ...phaseRows,
          ],
        },
      ];
    case 'correction':
      return [
        ...brief('Correction summary (FX09)'),
        {
          kind: 'beforeAfter',
          heading: 'Before / after (FX03/FX04)',
          beforeLabel: 'Before — findings under correction',
          before:
            segments.findings !== undefined
              ? `${segments.findings} recorded finding${segments.findings === 1 ? '' : 's'} on the prior candidate.`
              : 'Finding count not recorded on this entry; the complete finding set opens in the native review.',
          afterLabel: 'After — submitted correction',
          after: entry.ref
            ? `New candidate ${entry.ref}. Submitted for independent acceptance; not yet an accepted resolution.`
            : 'No new candidate reference recorded on this entry yet.',
        },
        {kind: 'checks', heading: 'Regression evidence (FX04)', rows: checkRows(segments)},
        {
          kind: 'rows',
          heading: 'Disposition state (FX02/FX08)',
          rows: [{term: 'Bound issue', value: issueValue}, ...phaseRows],
        },
      ];
    case 'reassessment':
      return [
        ...brief('Diagnosis and proposed repair (ST02/ST05)'),
        {
          kind: 'beforeAfter',
          heading: 'Work graph: current vs proposed (ST06)',
          beforeLabel: 'Current graph',
          before: 'The sealed graph stands until adoption is confirmed; protected work is preserved.',
          afterLabel: 'Proposed change',
          after: summary
            ? 'See the recorded proposal above. Proposed is not adopted.'
            : 'No proposal summary recorded on this entry.',
        },
        {
          kind: 'steps',
          heading: 'Adoption progress (ST10)',
          steps: [
            {label: 'Reassessment submitted', state: stageState(segments)},
            ...chainSteps(segments, ['published', 'merged', 'completed']).filter((step) => step.state === 'done'),
            ...(segments.chain.length === 0 ? [{label: 'Native adoption effects confirmed', state: 'waiting'}] : []),
          ],
        },
      ];
    case 'verification':
      return [
        ...brief('Verification summary (TP11)'),
        {kind: 'checks', heading: 'Accepted-outcome coverage evidence (TP02/TP07)', rows: checkRows(segments)},
        {
          kind: 'rows',
          heading: 'Delivery under test (TP01)',
          rows: [
            {term: 'Delivery candidate', value: refValue},
            {
              term: 'Verification outcome',
              value: segments.verdict ? humanSegment(segments.verdict) : 'not recorded on this entry',
            },
            ...phaseRows,
          ],
        },
      ];
    case 'integration':
      return [
        ...brief('Integration summary'),
        {
          kind: 'steps',
          heading: 'Publication → merge → completion',
          steps: [
            {label: 'Candidate recorded', state: entry.ref ? 'done' : 'not recorded'},
            ...chainSteps(segments, ['published', 'merged', 'completed']),
          ],
        },
        {
          kind: 'rows',
          heading: 'Native records',
          rows: [
            {term: 'Bound issue', value: issueValue},
            {term: 'Bound pull request', value: prValue},
            {term: 'Candidate reference', value: refValue},
          ],
        },
      ];
    case 'knowledge':
      return [
        ...brief('Recorded lesson or library change (K01/LA02)'),
        {
          kind: 'rows',
          heading: 'Entry state',
          rows: [
            {
              term: 'Support / validity',
              value: segments.outcome ? humanSegment(segments.outcome) : 'keeps its recorded support label',
            },
            ...phaseRows,
          ],
        },
      ];
    case 'decision':
      return [
        ...brief('Chosen option (S16)'),
        {
          kind: 'rows',
          heading: 'Decision record',
          rows: [
            {term: 'Decided by', value: entry.author ?? roleLabel(entry.actor) ?? 'not recorded on this entry'},
            {term: 'Application', value: 'logging a choice does not by itself apply a plan'},
            ...phaseRows,
          ],
        },
      ];
    case 'question':
      return [
        ...brief('Question (S02)'),
        {
          kind: 'rows',
          heading: 'What it blocks',
          rows: [
            {
              term: 'State',
              value: segments.outcome ? humanSegment(segments.outcome) : 'needs an answer',
            },
            {term: 'Answer path', value: 'reply below to add your answer to the project conversation'},
            ...phaseRows,
          ],
        },
      ];
    case 'merged':
      return [
        ...brief('Merge'),
        {
          kind: 'steps',
          heading: 'Merge (F06)',
          steps: [
            {label: 'Native merge confirmed', state: entry.ref ? `done — merged commit ${entry.ref}` : 'done'},
            {
              label: 'Original issue closed',
              state: segments.chain.includes('closed') ? 'done' : 'not recorded on this entry',
            },
          ],
        },
        {
          kind: 'rows',
          heading: 'Native records',
          rows: [
            {term: 'Bound issue', value: issueValue},
            {term: 'Bound pull request', value: prValue},
            {term: 'Merged commit', value: refValue},
          ],
        },
      ];
    case 'completed':
      return [
        ...brief('Completion'),
        {
          kind: 'steps',
          heading: 'Issue completion (F06)',
          steps: [{label: 'Issue completion recorded', state: 'done'}],
        },
        {
          kind: 'rows',
          heading: 'Native records',
          rows: [
            {term: 'Bound issue', value: issueValue},
            {term: 'Merged commit', value: refValue},
          ],
        },
      ];
    default:
      return brief('Recorded summary');
  }
}

function percentText(value: number): string {
  return `${Math.round(value * 100)}%`;
}

function shortRef(ref: string): string {
  return ref.length > 16 ? `${ref.slice(0, 12)}…` : ref;
}

function stampText(at: number): string {
  const date = new Date(at * 1000);
  return Number.isFinite(date.getTime()) ? date.toLocaleString() : 'time unrecorded';
}

// shownNote states a capped preview honestly: fewer members shown
// than recorded, or a recorded set with no preview carried at all.
function shownNote(shown: number, total: number, unit: string): string | undefined {
  if (total > shown)
    return shown === 0
      ? `${total} recorded ${unit}; the preview is not carried on this entry. Open the source record for the full set.`
      : `Showing ${shown} of ${total} recorded ${unit}; the rest open in the source record.`;
  return undefined;
}

function cappedNote(shown: number, unit: string): string | undefined {
  return shown >= feedPayloadCap
    ? `Preview capped at ${feedPayloadCap} recorded ${unit}; the full set opens in the source record.`
    : undefined;
}

function findingText(finding: FeedFinding): string {
  const parts: string[] = [];
  if (finding.defect) parts.push(`Defect: ${finding.defect}`);
  if (finding.evidence) parts.push(`Evidence: ${finding.evidence}`);
  if (finding.requiredOutcome) parts.push(`Required outcome: ${finding.requiredOutcome}`);
  if (finding.fixKind) parts.push(`Fix kind: ${humanSegment(finding.fixKind)}`);
  if (finding.actionable !== undefined) parts.push(finding.actionable ? 'Actionable.' : 'Not actionable.');
  if (finding.location)
    parts.push(
      `At ${finding.location.path}${finding.location.line ? ` line ${finding.location.line}` : ''}${finding.location.side ? ` (${finding.location.side} side)` : ''}.`
    );
  return parts.length > 0 ? parts.join(' ') : 'Finding body not recorded on this entry.';
}

function coverageChecks(coverage: readonly FeedCoverage[], total: number, heading: string): FeedBlock[] {
  if (coverage.length === 0 && total === 0) return [];
  return [
    {
      kind: 'checks',
      heading,
      rows: coverage.map((row) => ({
        name: `${row.intentId ?? 'intent not recorded'} / ${row.outcomeId ?? 'outcome not recorded'}`,
        result: `tasks: ${row.taskKeys.length > 0 ? row.taskKeys.join(', ') : 'none recorded'}; delivered acceptance: ${
          row.deliveredAcceptanceIds.length > 0 ? row.deliveredAcceptanceIds.join(', ') : 'none recorded'
        }`,
      })),
      note: shownNote(coverage.length, total, 'coverage rows'),
    },
  ];
}

function changeTasks(changes: readonly FeedIntentChange[]): {label: string; state: string}[] {
  return changes.flatMap((change) =>
    change.tasks.map((task) => ({
      label: `${change.intentId ? `${change.intentId} / ` : ''}${task.key}: ${task.objective ?? 'objective not recorded'}`,
      state: task.dependsOn.length > 0 ? `after ${task.dependsOn.join(', ')}` : 'starts first',
    }))
  );
}

function commandPreview(argv: readonly string[]): string {
  const characters = [...argv.join(' ')];
  return characters.length <= feedSummaryLimit
    ? characters.join('')
    : characters.slice(0, feedSummaryLimit).join('') + '… (command preview)';
}

// payloadBlocks renders the typed Lane B3 content inside the existing
// family components: plan steps and dependencies, check tables with
// real outcomes, finding bodies joined to their dispositions,
// reassessment before/after, verification coverage, judgment
// selection and candidates, and the merge/completion milestone
// steps. Failures lead within their payload; capped previews state
// their totals instead of reading as the whole set.
function payloadBlocks(entry: FeedEntry, payload: FeedPayload): FeedBlock[] {
  const blocks: FeedBlock[] = [];
  const plan = payload.plan;
  if (plan) {
    if (plan.approach) blocks.push({kind: 'brief', heading: 'Approach (AR01)', text: plan.approach});
    blocks.push({
      kind: 'rows',
      heading: 'Plan binding (AR07/AR11)',
      rows: [
        {term: 'Source commit', value: plan.sourceCommit ?? 'not recorded'},
        {term: 'Generation', value: plan.generation !== undefined ? String(plan.generation) : 'not recorded'},
        {
          term: 'Acceptance coverage',
          value: plan.acceptanceCoverage.length > 0 ? plan.acceptanceCoverage.join(', ') : 'not recorded',
        },
      ],
    });
    if (plan.tasks.length > 0 || plan.tasksTotal > 0)
      blocks.push({
        kind: 'graph',
        heading: 'Plan tasks and dependencies (AR07)',
        items: plan.tasks.map((task) => ({
          id: task.id,
          label: `${task.id}: ${task.objective ?? 'objective not recorded'}`,
          dependsOn: task.dependsOn,
          ...(task.objective ? {detail: task.objective} : {}),
          state: [
            task.lane ? `lane ${task.lane}` : 'no lane recorded',
            task.dependsOn.length > 0 ? `after ${task.dependsOn.join(', ')}` : 'starts first',
          ].join('; '),
        })),
        note: shownNote(plan.tasks.length, plan.tasksTotal, 'plan tasks'),
      });
    if (plan.tasks.some((task) => task.outputs.length > 0 || task.checks.length > 0))
      blocks.push({
        kind: 'rows',
        heading: 'Task outputs and checks',
        rows: plan.tasks.map((task) => ({
          term: task.id,
          value: `outputs: ${task.outputs.length > 0 ? task.outputs.join(', ') : 'not recorded'}; checks: ${
            task.checks.length > 0 ? task.checks.join(', ') : 'not recorded'
          }`,
        })),
      });
    if (plan.lanes.length > 0)
      blocks.push({
        kind: 'rows',
        heading: 'Parallel lanes',
        rows: plan.lanes.map((lane) => ({
          term: lane.id,
          value: lane.tasks.length > 0 ? lane.tasks.join(', ') : 'no tasks recorded',
        })),
        note: cappedNote(plan.lanes.length, 'lanes'),
      });
    if (plan.checks.length > 0 || plan.checksTotal > 0)
      blocks.push({
        kind: 'checks',
        heading: 'Planned checks (AR09)',
        rows: plan.checks.map((check) => ({
          name: check.argv.length > 0 ? `${check.id} — ${commandPreview(check.argv)}` : check.id,
          result: `${
            check.required === undefined ? 'requirement not recorded' : check.required ? 'required' : 'not required'
          }${check.scope ? `; scope ${check.scope}` : ''}; planned only — not run`,
        })),
        note: shownNote(plan.checks.length, plan.checksTotal, 'planned checks'),
      });
    if (plan.findings.length > 0 || plan.findingsTotal > 0)
      blocks.push({
        kind: 'rows',
        heading: 'Investigation findings (AR04)',
        rows: plan.findings.map((text, index) => ({term: `Finding ${index + 1}`, value: text})),
        note: shownNote(plan.findings.length, plan.findingsTotal, 'investigation findings'),
      });
    if (plan.inspected.length > 0 || plan.inspectedTotal > 0)
      blocks.push({
        kind: 'rows',
        heading: 'Evidence inspected (AR04)',
        rows: plan.inspected.map((row) => ({
          term: row.path,
          value: [row.commit ? `at ${shortRef(row.commit)}` : 'commit not recorded', row.evidence]
            .filter((part) => part !== undefined)
            .join(' — '),
        })),
        note: shownNote(plan.inspected.length, plan.inspectedTotal, 'inspected paths'),
      });
    if (plan.rejected.length > 0)
      blocks.push({
        kind: 'rows',
        heading: 'Rejected approaches',
        rows: plan.rejected.map((row) => ({
          term: row.approach ?? 'approach not recorded',
          value: row.reason ?? 'reason not recorded',
        })),
        note: cappedNote(plan.rejected.length, 'rejected approaches'),
      });
    if (plan.unresolved.length > 0)
      blocks.push({
        kind: 'rows',
        heading: 'Unresolved',
        rows: plan.unresolved.map((text, index) => ({term: `Open ${index + 1}`, value: text})),
        note: cappedNote(plan.unresolved.length, 'unresolved items'),
      });
  }
  const decomposition = payload.decomposition;
  if (decomposition) {
    if (decomposition.attention)
      blocks.push({kind: 'alert', heading: 'Needs attention (PL08)', text: decomposition.attention});
    blocks.push({
      kind: 'rows',
      heading: 'Decomposition (PL06)',
      rows: [
        {term: 'Operation', value: decomposition.operation ?? 'not recorded'},
        ...decomposition.candidates.map((candidate) => ({
          term: candidate.key ?? 'candidate not named',
          value: `${candidate.rationale ?? 'rationale not recorded'}${
            candidate.tasks.length > 0 ? ` Tasks: ${candidate.tasks.map((task) => task.key).join(', ')}.` : ''
          }`,
        })),
      ],
      note: shownNote(decomposition.candidates.length, decomposition.candidatesTotal, 'candidate graphs'),
    });
    const steps = decomposition.candidates.flatMap((candidate) =>
      candidate.tasks.map((task) => ({
        label: `${candidate.key ? `${candidate.key} / ` : ''}${task.key}: ${task.objective ?? 'objective not recorded'}`,
        state: [
          task.dependsOn.length > 0 ? `after ${task.dependsOn.join(', ')}` : 'starts first',
          task.outcomes.length > 0 ? `outcomes: ${task.outcomes.join('; ')}` : 'outcomes not recorded',
        ].join('; '),
      }))
    );
    if (steps.length > 0) blocks.push({kind: 'steps', heading: 'Candidate tasks (PL06)', steps});
  }
  const implementation = payload.implementation;
  if (implementation)
    blocks.push({
      kind: 'rows',
      heading: 'Implementation receipt (BU10)',
      rows: [
        {term: 'Candidate produced', value: implementation.candidate ?? 'not recorded'},
        {
          term: 'Diagnostic references',
          value: implementation.diagnosticRefs.length > 0 ? implementation.diagnosticRefs.join(', ') : 'none recorded',
        },
      ],
      note: cappedNote(implementation.diagnosticRefs.length, 'diagnostic references'),
    });
  const tasks = payload.tasks;
  if (tasks && (tasks.tasks.length > 0 || tasks.total > 0))
    blocks.push({
      kind: 'steps',
      heading: 'Recorded task states (BU01/BU06)',
      steps: tasks.tasks.map((task) => ({label: task.id, state: task.state ?? 'not recorded'})),
      note: shownNote(tasks.tasks.length, tasks.total, 'task states'),
    });
  const review = payload.review;
  if (review) {
    if (review.newFindings.length > 0 || review.newFindingsTotal > 0)
      blocks.push({
        kind: 'rows',
        heading: 'Review findings (IN03)',
        rows: review.newFindings.map((finding) => ({
          term: finding.severity ? `${finding.key} (${finding.severity})` : finding.key,
          value: findingText(finding),
        })),
        note: shownNote(review.newFindings.length, review.newFindingsTotal, 'review findings'),
      });
    if (review.findings.length > 0 || review.findingsTotal > 0)
      blocks.push({
        kind: 'rows',
        heading: 'Recorded findings (IN03)',
        rows: review.findings.map((text, index) => ({term: `Finding ${index + 1}`, value: text})),
        note: shownNote(review.findings.length, review.findingsTotal, 'recorded findings'),
      });
    if (review.assessments.length > 0 || review.assessmentsTotal > 0)
      blocks.push({
        kind: 'rows',
        heading: 'Finding assessments (IN04)',
        rows: review.assessments.map((row) => ({
          term: row.key,
          value: `${row.status ?? 'status not recorded'}${row.evidence ? ` — ${row.evidence}` : ''}`,
        })),
        note: shownNote(review.assessments.length, review.assessmentsTotal, 'finding assessments'),
      });
    if (review.clear)
      blocks.push({
        kind: 'rows',
        heading: 'Clear review coverage (IN10)',
        rows: [
          {term: 'Reviewed head', value: review.clear.head ?? 'not recorded'},
          {term: 'Reviewed base', value: review.clear.base ?? 'not recorded'},
          {term: 'Coverage digest', value: review.clear.coverageDigest ?? 'not recorded'},
          ...review.clear.assessments.map((row) => ({
            term: `Assessment ${row.key}`,
            value: `${row.status ?? 'status not recorded'}${row.evidence ? ` — ${row.evidence}` : ''}`,
          })),
        ],
        note: shownNote(review.clear.assessments.length, review.clear.assessmentsTotal, 'coverage assessments'),
      });
    if (review.verdict || review.cycleId || (review.summary && review.summary !== entry.summary))
      blocks.push({
        kind: 'rows',
        heading: 'Verdict record (IN02)',
        rows: [
          {term: 'Recorded verdict', value: review.verdict ?? 'not recorded'},
          {term: 'Review cycle', value: review.cycleId ?? 'not recorded'},
          ...(review.summary && review.summary !== entry.summary
            ? [{term: 'Review summary', value: review.summary}]
            : []),
        ],
      });
  }
  const correction = payload.correction;
  if (correction) {
    blocks.push({
      kind: 'beforeAfter',
      heading: 'Before / after (FX03/FX04)',
      beforeLabel: 'Before — findings under correction',
      before: `${correction.findingsTotal} recorded finding${correction.findingsTotal === 1 ? '' : 's'} under correction.`,
      afterLabel: 'After — submitted correction',
      after: correction.newCandidate
        ? `New candidate ${correction.newCandidate}. Submitted for independent acceptance; not yet an accepted resolution.`
        : 'No new candidate recorded on this entry yet.',
    });
    const dispositions = new Map(correction.dispositions.map((row) => [row.key, row]));
    const joined = correction.findings.map((finding) => {
      const disposition = dispositions.get(finding.key);
      dispositions.delete(finding.key);
      return {
        term: finding.severity ? `${finding.key} (${finding.severity})` : finding.key,
        value: disposition
          ? `${findingText(finding)} Disposition: ${disposition.disposition ?? 'not recorded'}. Changed paths: ${
              disposition.changedPaths.length > 0 ? disposition.changedPaths.join(', ') : 'none recorded'
            }. Commits: ${disposition.commits.length > 0 ? disposition.commits.map(shortRef).join(', ') : 'none recorded'}.`
          : `${findingText(finding)} No disposition recorded.`,
      };
    });
    for (const disposition of dispositions.values())
      joined.push({
        term: disposition.key,
        value: `Disposition: ${disposition.disposition ?? 'not recorded'}. Changed paths: ${
          disposition.changedPaths.length > 0 ? disposition.changedPaths.join(', ') : 'none recorded'
        }. Commits: ${disposition.commits.length > 0 ? disposition.commits.map(shortRef).join(', ') : 'none recorded'}. Finding body not carried on this entry.`,
      });
    blocks.push({
      kind: 'rows',
      heading: 'Finding dispositions (FX02)',
      rows: joined,
      note:
        shownNote(correction.findings.length, correction.findingsTotal, 'findings') ??
        shownNote(correction.dispositions.length, correction.dispositionsTotal, 'dispositions'),
    });
    blocks.push({
      kind: 'rows',
      heading: 'Correction binding (FX03)',
      rows: [
        {term: 'New candidate', value: correction.newCandidate ?? 'not recorded'},
        {term: 'Finding set digest', value: correction.setDigest ?? 'not recorded'},
      ],
    });
  }
  const delta = payload.reassessment ?? payload.repair;
  if (delta) {
    const repair = payload.repair;
    if (repair && repair.causes.length > 0)
      blocks.push({
        kind: 'checks',
        heading: 'Recorded causes (ST02)',
        rows: repair.causes.map((cause) => ({
          name: cause.check ?? 'cause not named',
          result: cause.detail ?? 'detail not recorded',
        })),
        note: cappedNote(repair.causes.length, 'recorded causes'),
      });
    blocks.push({
      kind: 'beforeAfter',
      heading: 'Work graph: current vs proposed (ST06)',
      beforeLabel: 'Current graph — retained work',
      before:
        delta.retainedIssues.length > 0
          ? `Retained issues: ${delta.retainedIssues.map((issue) => `#${issue}`).join(', ')}. The sealed graph stands until adoption is confirmed.`
          : 'No retained issues recorded; the sealed graph stands until adoption is confirmed.',
      afterLabel: 'Proposed change',
      after:
        delta.changes.length > 0
          ? delta.changes
              .map((change) => `${change.intentId ?? 'intent not recorded'}: ${change.reason ?? 'reason not recorded'}`)
              .join('; ')
          : 'No graph changes recorded. Proposed is not adopted.',
      note: shownNote(delta.changes.length, delta.changesTotal, 'proposed changes'),
    });
    const steps = changeTasks(delta.changes);
    if (steps.length > 0) blocks.push({kind: 'steps', heading: 'Proposed tasks (ST06)', steps});
    if (delta.attention.length > 0)
      blocks.push({
        kind: 'rows',
        heading: 'Attention needed (ST08)',
        rows: delta.attention.map((row) => ({
          term: row.issue ? `Issue #${row.issue}` : 'issue not recorded',
          value: row.reason ?? 'reason not recorded',
        })),
        note: cappedNote(delta.attention.length, 'attention entries'),
      });
    blocks.push(...coverageChecks(delta.coverage, delta.coverageTotal, 'Outcome coverage (ST08)'));
    const binding: {term: string; value: string}[] = [
      {term: 'Inspected records', value: delta.inspectedTotal > 0 ? String(delta.inspectedTotal) : 'not recorded'},
    ];
    if (repair) binding.unshift({term: 'Repair outcome', value: repair.outcome ?? 'not recorded'});
    blocks.push({kind: 'rows', heading: 'Reassessment binding (ST05)', rows: binding});
  }
  const verification = payload.verification;
  if (verification) {
    if (verification.failures.length > 0)
      blocks.push({
        kind: 'checks',
        heading: 'Recorded failures (TP11)',
        rows: verification.failures.map((failure) => ({
          name: failure.check ?? 'check not named',
          result: failure.detail ?? 'detail not recorded',
        })),
        note: cappedNote(verification.failures.length, 'recorded failures'),
      });
    if (verification.checks.length > 0 || verification.checksTotal > 0)
      blocks.push({
        kind: 'checks',
        heading: 'Check receipts (TP07)',
        rows: verification.checks.map((receipt) => ({
          name: receipt.id,
          result: `${receipt.outcome ?? 'outcome not recorded'}${
            receipt.artifactDigest ? `; artifact ${shortRef(receipt.artifactDigest)}` : ''
          }`,
        })),
        note: shownNote(verification.checks.length, verification.checksTotal, 'check receipts'),
      });
    blocks.push(
      ...coverageChecks(verification.coverage, verification.coverageTotal, 'Accepted-outcome coverage (TP02)')
    );
    blocks.push({
      kind: 'rows',
      heading: 'Delivery under test (TP01)',
      rows: [
        {term: 'Delivery', value: verification.deliveryId ?? 'not recorded'},
        {term: 'Verification outcome', value: verification.outcome ?? 'not recorded'},
        {term: 'Artifacts recorded', value: String(verification.artifactsTotal)},
        {term: 'Evidence recorded', value: String(verification.evidenceTotal)},
      ],
    });
  }
  const judgment = payload.judgment;
  if (judgment) {
    blocks.push({
      kind: 'rows',
      heading: 'Recorded judgment (F04)',
      rows: [
        {term: 'Kind', value: `${judgment.kind} judgment`},
        {term: 'Selected', value: judgment.selected ?? 'not recorded'},
        {term: 'Outcome', value: judgment.outcome ?? 'not recorded'},
        {term: 'Effective', value: judgment.effective ?? 'not recorded'},
        {term: 'Model', value: judgment.model ?? 'not recorded'},
        {
          term: 'Probability',
          value: judgment.probability !== undefined ? percentText(judgment.probability) : 'not recorded',
        },
        {
          term: 'Confidence',
          value: judgment.confidence !== undefined ? percentText(judgment.confidence) : 'not recorded',
        },
      ],
    });
    if (judgment.candidates.length > 0 || judgment.candidatesTotal > 0)
      blocks.push({
        kind: 'checks',
        heading: 'Candidates considered (F04)',
        rows: judgment.candidates.map((candidate) => {
          const weight = judgment.probabilities[candidate.id];
          const selected = candidate.id === judgment.selected;
          return {
            name: `${candidate.id}${candidate.objective ? ` — ${candidate.objective}` : ''}${selected ? ' — selected' : ''}`,
            result: `${weight !== undefined ? percentText(weight) : 'probability not recorded'}${
              candidate.issue ? `; issue #${candidate.issue}` : ''
            }${candidate.repository ? ` in repository ${candidate.repository}` : ''}`,
          };
        }),
        note: shownNote(judgment.candidates.length, judgment.candidatesTotal, 'candidates'),
      });
    else if (Object.keys(judgment.probabilities).length > 0)
      blocks.push({
        kind: 'checks',
        heading: 'Options considered (F04)',
        rows: Object.entries(judgment.probabilities).map(([option, weight]) => ({
          name: option === judgment.selected ? `${option} — selected` : option,
          result: percentText(weight),
        })),
      });
  }
  const merge = payload.merge;
  if (merge)
    blocks.push(
      {
        kind: 'steps',
        heading: 'Merge (F06)',
        steps: [
          {
            label: 'Native merge confirmed',
            state: merge.mergedCommit ? `done — merged commit ${merge.mergedCommit}` : 'done',
          },
          {
            label: 'Original issue closed',
            state: merge.originalClosedUnix ? `done — ${stampText(merge.originalClosedUnix)}` : 'not recorded',
          },
        ],
      },
      {
        kind: 'rows',
        heading: 'Merge record (F06)',
        rows: [
          {term: 'Stage', value: merge.stage ?? 'not recorded'},
          {term: 'Publication', value: merge.publication ?? 'not recorded'},
          {term: 'Merged commit', value: merge.mergedCommit ?? 'not recorded'},
          {term: 'Merged at', value: merge.mergedUnix ? stampText(merge.mergedUnix) : 'not recorded'},
        ],
      }
    );
  const completion = payload.completion;
  if (completion)
    blocks.push(
      {
        kind: 'steps',
        heading: 'Issue completion (F06)',
        steps: [
          {
            label: 'Issue completion recorded',
            state: completion.mergedCommit ? `done — merged commit ${completion.mergedCommit}` : 'done',
          },
        ],
      },
      {
        kind: 'rows',
        heading: 'Completion record (F07/F10)',
        rows: [
          {term: 'Event', value: completion.event ?? entry.id},
          {term: 'Occurrence', value: completion.occurrence ?? 'not recorded'},
          {term: 'Publication', value: completion.publication ?? 'not recorded'},
          {term: 'Merged commit', value: completion.mergedCommit ?? 'not recorded'},
          {
            term: 'Generation',
            value: completion.generation !== undefined ? String(completion.generation) : 'not recorded',
          },
        ],
      }
    );
  return blocks;
}

// describeFeedFamily projects one entry to its focal family section:
// what changed, why it matters, its recorded state and what follows,
// with the source links the wire supports. Malformed or missing
// enrichment degrades to unknown state, never fabricated success.
export function describeFeedFamily(entry: FeedEntry, repository: FeedRepositoryRef): FeedFamilyModel {
  const family = selectFeedFamily(entry);
  const text = familyText[family];
  const communication = entry.payload?.communication;
  const resolved = communication?.disposition !== undefined || communication?.questionState === 'incorporated';
  const statusClass: FeedStatusClass =
    communication && communication.kind !== 'system' && communication.kind !== 'library_maintenance'
      ? resolved
        ? 'settled'
        : ['blocked', 'stuck', 'finish_incomplete'].includes(communication.kind) ||
            (communication.kind === 'flag' && communication.severity !== 'info')
          ? 'failure'
          : 'waiting'
      : classifyFeedStatus(entry.status);
  const recordedHeadline =
    entry.activity === 'judgment' || entry.status === 'judgment:applied'
      ? 'Decision recorded'
      : entry.status === 'draft:valid'
        ? 'Proposal ready'
        : entry.status === 'effect:created'
          ? 'Work created'
          : entry.status === 'run:succeeded'
            ? 'Run ended'
            : entry.status === 'dispatch:assigned'
              ? 'Started'
              : undefined;
  const payload = entry.payload;
  const typed: FeedBlock[] =
    payload === undefined
      ? []
      : payload.invalid
        ? [
            {
              kind: 'alert',
              heading: 'Recorded detail unavailable',
              text: 'Detailed recorded content on this entry could not be read and is not shown. The recorded status above stands; open the source record before treating it as done.',
            },
          ]
        : payloadBlocks(entry, payload);
  return {
    entryKey: feedEntryKey(entry),
    ...(payload ? {payload} : {}),
    ...(payload?.communication ? {communication: payload.communication} : {}),
    ...(payload?.componentId ? {componentId: payload.componentId} : {}),
    family,
    label: text.label,
    headline:
      recordedHeadline ??
      `${text.label}${statusClass === 'failure' ? ' needs attention' : statusClass === 'settled' ? ' completed' : statusClass === 'waiting' ? ' in progress' : ' · status unavailable'}`,
    detailText: text.detail,
    stateText:
      entry.status !== undefined
        ? `Recorded status: ${entry.status}.`
        : 'Status not recorded on this entry; its outcome is unknown, not successful.',
    summaryText: entry.summary ? `Recorded: ${entry.summary}` : undefined,
    refText: entry.ref ? `Candidate reference: ${entry.ref}.` : undefined,
    nextText: nextTextFor(statusClass),
    statusClass,
    ...(entry.status === 'run:succeeded' ? {neutral: true} : {}),
    sourceLinks: familyLinks(family, repository, entry),
    blocks: typed.length
      ? typed
      : statusClass === 'failure' && (family === 'plan' || family === 'issue')
        ? entry.summary
          ? [
              {
                kind: 'brief' as const,
                heading: 'What happened',
                text:
                  family === 'plan'
                    ? 'Planning stopped before a plan was accepted.'
                    : 'This work needs attention before it can continue.',
              },
              {kind: 'details' as const, heading: 'Technical details', text: entry.summary},
            ]
          : []
        : withoutPlaceholders(familyBlocks(family, entry, parseFeedStatusSegments(entry.status))),
  };
}

// A row that only says a fact is missing is filler, not evidence. Status-derived
// blocks keep their recorded facts and drop such rows; a block left with no
// rows is dropped too, so a post never shows an empty placeholder table.
const placeholder = /not recorded|^no bound |^no tasks recorded|count not recorded/i;
function withoutPlaceholders(blocks: readonly FeedBlock[]): FeedBlock[] {
  return blocks.flatMap((block): FeedBlock[] => {
    if (block.kind === 'rows') {
      const rows = block.rows.filter((row) => !placeholder.test(row.value));
      return rows.length > 0 ? [{...block, rows}] : [];
    }
    if (block.kind === 'checks') {
      const rows = block.rows.filter((row) => !placeholder.test(row.result));
      return rows.length > 0 ? [{...block, rows}] : [];
    }
    return [block];
  });
}

function renderFeedBlock(block: FeedBlock): TemplateResult {
  block = {...block, heading: block.heading.replace(/\s*\([A-Z]+\d+[^)]*\)/g, '')};
  const note = block.note ? html`<p class="soda-feed-block-note">${block.note}</p>` : '';
  switch (block.kind) {
    case 'alert':
      return html`${html`<div class="soda-feed-brief soda-feed-alert">
        <h4>${block.heading}</h4>
        <p role="alert">${block.text}</p>
      </div>`}${note}`;
    case 'brief':
      return html`${html`<div class="soda-feed-brief">
        <h4>${block.heading}</h4>
        <blockquote><p>${block.text}</p></blockquote>
      </div>`}${note}`;
    case 'steps':
      return html`${html`<div class="soda-feed-steps">
        <h4>${block.heading}</h4>
        <ol>
          ${block.steps.map((step) => html`<li><strong>${step.label}</strong> — ${step.state}</li>`)}
        </ol>
      </div>`}${note}`;
    case 'graph':
      return html`<div class="soda-feed-plan-graph">
        <h4>${block.heading}</h4>
        ${renderFeedGraph(block.items, block.heading)}${note}
      </div>`;
    case 'rows':
      return html`<div class="soda-feed-rows">
        <h4>${block.heading}</h4>
        <dl>
          ${block.rows.map(
            (row) =>
              html`<div>
                <dt>${row.term}</dt>
                <dd>${row.value}</dd>
              </div>`
          )}
        </dl>
        ${note}
      </div>`;
    case 'checks':
      return html`<div class="soda-feed-checks">
        <h4>${block.heading}</h4>
        <table>
          <thead>
            <tr>
              <th scope="col">Check evidence</th>
              <th scope="col">Recorded result</th>
            </tr>
          </thead>
          <tbody>
            ${block.rows.map(
              (row) =>
                html`<tr>
                  <th scope="row">${row.name}</th>
                  <td>${row.result}</td>
                </tr>`
            )}
          </tbody>
        </table>
        ${note}
      </div>`;
    case 'details':
      return html`<details class="soda-feed-technical-details">
          <summary>${block.heading}</summary>
          <p>${block.text}</p>
        </details>${note}`;
    case 'beforeAfter':
      return html`<div class="soda-feed-beforeafter">
        <h4>${block.heading}</h4>
        <div class="soda-feed-before">
          <h5>${block.beforeLabel}</h5>
          <p>${block.before}</p>
        </div>
        <div class="soda-feed-after">
          <h5>${block.afterLabel}</h5>
          <p>${block.after}</p>
        </div>
        ${note}
      </div>`;
  }
}

export function renderFeedFamily(model: FeedFamilyModel, request?: FeedArtifactRequest): TemplateResult {
  const component = feedComponent(model.componentId);
  const action = renderRecordAction(model.payload);
  return html`<section
    aria-label=${model.label}
    data-feed-family=${model.family}
    data-feed-component=${component?.id ?? ''}
  >
    ${
      model.communication
        ? renderFeedCommunication(model.communication, model.entryKey ?? '', request)
        : html`<p class="soda-feed-headline"><strong>${component?.label ?? model.headline}</strong></p>
            ${action ?? html`${model.blocks.slice(0, 4).map(renderFeedBlock)}`}
            ${
              action
                ? html`<details class="soda-feed-more-content">
                    <summary>Recorded details and evidence</summary>
                    ${model.blocks.map(renderFeedBlock)}
                  </details>`
                : model.blocks.length > 4
                  ? html`<details class="soda-feed-more-content">
                      <summary>${model.blocks.length - 4} more details</summary>
                      ${model.blocks.slice(4).map(renderFeedBlock)}
                    </details>`
                  : ''
            }`
    }
    <div class="soda-feed-card-footer">
      <details class="soda-feed-record-details">
        <summary>Details</summary>
        <p>${model.stateText}</p>
        <p>${model.detailText}</p>
        <p>${model.nextText}</p>
      </details>
      ${
        model.sourceLinks.length > 0
          ? html`<p class="soda-feed-source-links">
              ${model.sourceLinks.map((link) => html`<a href=${link.href}>${link.label}</a> `)}
            </p>`
          : ''
      }
    </div>
  </section>`;
}
