import test from 'node:test';
import assert from 'node:assert/strict';
import {workJourney} from '../../frontend/spaces/sodaspaces-factory-work-journey-response.js';
import {
  describeJourneyAdoption,
  describeJourneyCommit,
  describeJourneyCorrection,
  describeJourneyFinding,
  describeJourneyGate,
  describeJourneyJudgment,
  describeJourneyPhase,
  describeJourneyTask,
  describeWorkJourney,
} from '../../frontend/spaces/sodaspaces-factory-work-journey-view.js';
import {WORK_JOURNEY_FIXTURE} from './fixtures/factory-work-journey-fixture.js';

const path = '/soda-tester/demo';

test('fixture journey binds every section to its issue', () => {
  const journey = workJourney(structuredClone(WORK_JOURNEY_FIXTURE), '7', '12');
  assert.equal(journey.phases.length, 5);
  assert.equal(journey.tasks.length, 3);
  assert.equal(journey.commits.length, 1);
  assert.equal(journey.findings.length, 1);
  assert.equal(journey.corrections.length, 2);
  assert.equal(journey.judgments.length, 4);
  assert.equal(journey.gates.length, 2);
  assert.equal(journey.adoptions.length, 2);
});

test('phases, tasks and commits render lanes and native links', () => {
  const journey = workJourney(structuredClone(WORK_JOURNEY_FIXTURE), '7', '12');
  assert.match(describeJourneyPhase(journey.phases[2]!), /review.*active/);
  const task = describeJourneyTask(journey.tasks[0]!, path);
  assert.match(task.text, /task-plan.*lane a.*complete.*issue #12/);
  assert.equal(task.link, '/soda-tester/demo/issues/12');
  assert.equal(describeJourneyTask(journey.tasks[2]!, path).link, undefined);
  const commit = describeJourneyCommit(journey.commits[0]!, path);
  assert.match(commit.text, /PR #34/);
  assert.equal(commit.pullLink, '/soda-tester/demo/pulls/34');
});

test('findings and corrections render severity and effects', () => {
  const journey = workJourney(structuredClone(WORK_JOURNEY_FIXTURE), '7', '12');
  assert.match(describeJourneyFinding(journey.findings[0]!), /major.*code.*bound check/);
  assert.match(describeJourneyCorrection(journey.corrections[0]!), /committed.*complete/);
  assert.match(describeJourneyCorrection(journey.corrections[1]!), /no candidate published/);
});

test('judgments separate decided choices from effective waits', () => {
  const journey = workJourney(structuredClone(WORK_JOURNEY_FIXTURE), '7', '12');
  assert.match(describeJourneyJudgment(journey.judgments[0]!), /priority.*7\/12.*No wait/);
  assert.match(describeJourneyJudgment(journey.judgments[1]!), /Delivery.*review_again.*low_confidence/);
  assert.match(describeJourneyJudgment(journey.judgments[2]!), /D13 completion decision.*reassess/);
  assert.match(
    describeJourneyJudgment(journey.judgments[3]!),
    /D13 reassessment proposal.*needs_attention.*provider_unknown_response/
  );
});

test('D13 gates render cycles, proposals, waits and retry', () => {
  const journey = workJourney(structuredClone(WORK_JOURNEY_FIXTURE), '7', '12');
  const waiting = describeJourneyGate(journey.gates[0]!);
  assert.match(waiting.gateText, /generation 2.*attention.*reassess/);
  assert.match(waiting.proposalText, /needs_attention/);
  assert.match(waiting.waitText, /provider_unknown_response.*retry can proceed/);
  const released = describeJourneyGate(journey.gates[1]!);
  assert.match(released.gateText, /released.*continue/);
  assert.match(released.proposalText, /No proposal assessed/);
  assert.equal(released.waitText, '');
});

test('adoptions name the bound revision only', () => {
  const journey = workJourney(structuredClone(WORK_JOURNEY_FIXTURE), '7', '12');
  assert.match(describeJourneyAdoption(journey.adoptions[0]!), /adopted.*bound revision/);
  assert.match(describeJourneyAdoption(journey.adoptions[1]!), /proposed/);
});

test('journey model headlines sections and stays silent when empty', () => {
  const journey = workJourney(structuredClone(WORK_JOURNEY_FIXTURE), '7', '12');
  const model = describeWorkJourney(journey, path);
  assert.match(model.headline, /3 tasks.*1 findings.*2 completion gates/);
  assert.equal(model.phases.length, 5);
  const empty = describeWorkJourney(
    workJourney({...structuredClone(WORK_JOURNEY_FIXTURE), phases: [], tasks: [], findings: [], gates: []}, '7', '12'),
    path
  );
  assert.match(empty.headline, /No journey sections/);
});

test('journey parser refuses foreign identity and broken bounds', () => {
  assert.throws(() => workJourney(structuredClone(WORK_JOURNEY_FIXTURE), '7', '13'));
  assert.throws(() =>
    workJourney({...structuredClone(WORK_JOURNEY_FIXTURE), phases: [{phase: 'publish', state: 'active'}]}, '7', '12')
  );
  assert.throws(() =>
    workJourney({...structuredClone(WORK_JOURNEY_FIXTURE), tasks: [{key: 'k', lane: 'd', state: 'active'}]}, '7', '12')
  );
  assert.throws(() =>
    workJourney(
      {
        ...structuredClone(WORK_JOURNEY_FIXTURE),
        judgments: [{kind: 'delivery', selected: 'merge_now', wait: false}],
      },
      '7',
      '12'
    )
  );
  assert.throws(() =>
    workJourney(
      {
        ...structuredClone(WORK_JOURNEY_FIXTURE),
        judgments: [{kind: 'priority', selected: '7/12', wait: true}],
      },
      '7',
      '12'
    )
  );
  assert.throws(() =>
    workJourney(
      {
        ...structuredClone(WORK_JOURNEY_FIXTURE),
        gates: [
          {
            completion_id: 'c'.repeat(32),
            cycle_id: '1'.repeat(32),
            state: 'attention',
            repository: '7',
            generation: 1,
            retryable: true,
          },
        ],
      },
      '7',
      '12'
    )
  );
  assert.throws(() =>
    workJourney(
      {
        ...structuredClone(WORK_JOURNEY_FIXTURE),
        gates: [
          {
            completion_id: 'c'.repeat(32),
            cycle_id: '1'.repeat(32),
            state: 'released',
            repository: '7',
            generation: 1,
            retryable: true,
          },
        ],
      },
      '7',
      '12'
    )
  );
});
