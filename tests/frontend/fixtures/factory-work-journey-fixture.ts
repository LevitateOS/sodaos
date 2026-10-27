// Full fixture work journey: every rendered section with D13 completion
// gate, proposal and adoption, delivered-task evidence and the verified
// application revision. Recorded-shaped facts only.

const head = 'a'.repeat(40);
const base = 'b'.repeat(40);
const corrected = 'c'.repeat(40);
const digestOf = (char: string): string => char.repeat(64);
const idOf = (char: string): string => char.repeat(32);

export const WORK_JOURNEY_FIXTURE = {
  repository: '7',
  issue: '12',
  observed_unix: 1720000300,
  phases: [
    {phase: 'investigate', state: 'complete'},
    {phase: 'implement', state: 'complete'},
    {phase: 'review', state: 'active'},
    {phase: 'correct', state: 'pending'},
    {phase: 'reassess', state: 'pending'},
  ],
  tasks: [
    {key: 'task-plan', lane: 'a', issue: '12', state: 'complete'},
    {key: 'task-build', lane: 'b', issue: '13', state: 'active'},
    {key: 'task-docs', lane: 'unassigned', state: 'pending'},
  ],
  commits: [{commit: head, base, pr_number: '34', pr_id: '56'}],
  findings: [
    {
      key: idOf('f'),
      severity: 'major',
      defect: 'Missing bound check before the slice write.',
      fix_kind: 'code',
    },
  ],
  corrections: [
    {candidate: corrected, effect: 'committed', completion: 'complete'},
    {effect: 'pending', completion: 'pending', reason: 'awaiting_correction_run'},
  ],
  judgments: [
    {kind: 'priority', selected: '7/12', model: 'jev-1.13.0', confidence: 0.8, wait: false},
    {
      kind: 'delivery',
      selected: 'review_again',
      model: 'jev-1.13.0',
      confidence: 0.4,
      wait: true,
      wait_reason: 'low_confidence',
    },
    {
      kind: 'reassessment-decision',
      selected: 'reassess',
      model: 'jev-1.13.0',
      confidence: 0.8,
      wait: false,
    },
    {
      kind: 'reassessment-proposal',
      selected: 'needs_attention',
      model: 'jev-1.13.0',
      confidence: 0.85,
      probability: 0.85,
      wait: true,
      wait_reason: 'provider_unknown_response',
    },
  ],
  gates: [
    {
      completion_id: idOf('c'),
      cycle_id: idOf('1'),
      state: 'attention',
      decision: 'reassess',
      proposal: 'needs_attention',
      repository: '7',
      generation: 2,
      retryable: true,
      wait_reason: 'provider_unknown_response',
      proposal_digest: digestOf('d'),
      inventory_digest: digestOf('e'),
      receipt_digests: [digestOf('a')],
    },
    {
      completion_id: idOf('c'),
      cycle_id: idOf('2'),
      state: 'released',
      decision: 'continue',
      repository: '7',
      generation: 1,
      retryable: false,
    },
  ],
  adoptions: [
    {
      intent: `d${'e'.repeat(24)}`,
      revision: `d${'f'.repeat(24)}`,
      state: 'adopted',
      task: 'task-plan',
      issue_id: '12',
      acceptance: `d${'c'.repeat(24)}`,
    },
    {
      intent: `d${'e'.repeat(24)}`,
      revision: `d${'a'.repeat(24)}`,
      state: 'proposed',
      task: 'task-docs',
      issue_id: '12',
      acceptance: `d${'b'.repeat(24)}`,
    },
  ],
  delivered_tasks: [
    {
      task: 'task-plan',
      issue: '12',
      acceptance: `d${'c'.repeat(24)}`,
      completion: idOf('9'),
      publication: idOf('8'),
      generation: 2,
    },
  ],
  verifications: [
    {
      intent: `d${'e'.repeat(24)}`,
      revision: `d${'f'.repeat(24)}`,
      source_commit: head,
      manifest: digestOf('m'),
    },
  ],
};
