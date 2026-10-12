import {test} from 'bun:test';
import assert from 'node:assert/strict';
import {savedReviewRead} from '../../frontend/spaces/sodaspaces-intent-response';
import {hflowCurrent as current} from './fixtures/saved-review';

function mutate(mutate: (doc: Record<string, unknown>) => void): unknown {
  const doc = JSON.parse(current) as Record<string, unknown>;
  mutate(doc);
  return doc;
}

function review(doc: Record<string, unknown>): Record<string, unknown> {
  return doc.review as Record<string, unknown>;
}

test('current read preserves basis-bound review with ordering and provenance', () => {
  const read = savedReviewRead(JSON.parse(current));
  assert.equal(read.status, 'current');
  const review = read.review;
  assert.ok(review);
  assert.equal(review.basis_revision, 'd111111111111111111111111');
  assert.equal(review.result_revision, 'd222222222222222222222222');
  assert.equal(review.content.alternatives?.length, 2);
  assert.equal(review.content.alternatives?.[0]?.key, 'single_person');
  assert.equal(review.recommendation?.milestones.length, 4);
  assert.equal(review.challenges.length, 1);
  assert.deepEqual(review.settled, ['d555555555555555555555555']);
  assert.equal(review.content.exclusions?.length, 1);
  assert.equal(review.content.exclusions?.[0]?.provenance, 'human');
});

test('stale read keeps the review past a moved head', () => {
  const read = savedReviewRead(
    mutate((doc) => {
      doc.head_revision = 'd333333333333333333333333';
      doc.status = 'stale';
    })
  );
  assert.equal(read.status, 'stale');
  assert.equal(read.review?.result_revision, 'd222222222222222222222222');
});

test('parser tolerates server null arrays', () => {
  const read = savedReviewRead(
    mutate((doc) => {
      const revision = review(doc);
      (revision.recommendation as Record<string, unknown>).constraints = null;
      (revision.consistency as Record<string, unknown>).mismatches = null;
      revision.challenges = null;
    })
  );
  assert.deepEqual(read.review?.recommendation?.constraints, []);
  assert.deepEqual(read.review?.consistency?.mismatches, []);
  assert.deepEqual(read.review?.challenges, []);
  const unsupported = savedReviewRead(
    mutate((doc) => {
      const content = review(doc).content as Record<string, unknown>;
      content.supported = false;
      content.alternatives = null;
    })
  );
  assert.equal(unsupported.review?.content.alternatives, undefined);
  const branches = savedReviewRead(
    mutate((doc) => {
      const alternatives = (review(doc).content as Record<string, unknown>).alternatives as Array<
        Record<string, unknown>
      >;
      const first = alternatives[0];
      assert.ok(first);
      const steps = first.steps as Array<Record<string, unknown>>;
      const step = steps[0];
      assert.ok(step);
      step.branches = null;
    })
  );
  const adopted = branches.review?.content.alternatives?.[0]?.steps[0];
  assert.equal(adopted?.branches, undefined);
});

test('missing read carries no review', () => {
  const read = savedReviewRead({
    intent: 'd000000000000000000000000',
    head_revision: 'd333333333333333333333333',
    status: 'missing',
  });
  assert.equal(read.status, 'missing');
  assert.equal(read.review, undefined);
});

test('parser rejects substituted identities', () => {
  const cases: Array<[string, unknown]> = [
    [
      'content bound to result',
      mutate((doc) => {
        (review(doc).content as Record<string, unknown>).revision = 'd222222222222222222222222';
      }),
    ],
    [
      'recommendation bound to result',
      mutate((doc) => {
        (review(doc).recommendation as Record<string, unknown>).revision = 'd222222222222222222222222';
      }),
    ],
    [
      'recommendation bound to another intent',
      mutate((doc) => {
        (review(doc).recommendation as Record<string, unknown>).intent = 'd999999999999999999999999';
      }),
    ],
    [
      'consistency bound to another intent',
      mutate((doc) => {
        (review(doc).consistency as Record<string, unknown>).intent = 'd999999999999999999999999';
      }),
    ],
    [
      'recommendation advice not a digest',
      mutate((doc) => {
        (review(doc).recommendation as Record<string, unknown>).advice = 'judgment call number three';
      }),
    ],
    [
      'consistency advice not a digest',
      mutate((doc) => {
        (review(doc).consistency as Record<string, unknown>).advice = 'checked and confident';
      }),
    ],
    [
      'basis equals result',
      mutate((doc) => {
        review(doc).basis_revision = 'd222222222222222222222222';
      }),
    ],
    [
      'current claim past the head',
      mutate((doc) => {
        doc.head_revision = 'd333333333333333333333333';
      }),
    ],
    [
      'stale claim at the head',
      mutate((doc) => {
        doc.status = 'stale';
      }),
    ],
    [
      'missing carrying a review',
      mutate((doc) => {
        doc.status = 'missing';
      }),
    ],
    [
      'challenge colliding with settlement',
      mutate((doc) => {
        review(doc).settled = ['d444444444444444444444444'];
      }),
    ],
    [
      'duplicate challenge ideas',
      mutate((doc) => {
        const challenges = review(doc).challenges as Array<Record<string, unknown>>;
        challenges.push({...challenges[0]});
      }),
    ],
    [
      'duplicate settled ideas',
      mutate((doc) => {
        review(doc).settled = ['d555555555555555555555555', 'd555555555555555555555555'];
      }),
    ],
    [
      'excluded idea without meanings',
      mutate((doc) => {
        delete (review(doc).content as Record<string, unknown>).exclusions;
      }),
    ],
    [
      'exclusion meaning mismatched to excluded id',
      mutate((doc) => {
        const meanings = (review(doc).content as Record<string, unknown>).exclusions as Array<Record<string, unknown>>;
        const first = meanings[0];
        assert.ok(first);
        first.idea = 'd777777777777777777777777';
      }),
    ],
    [
      'exclusion with forged provenance',
      mutate((doc) => {
        const meanings = (review(doc).content as Record<string, unknown>).exclusions as Array<Record<string, unknown>>;
        const first = meanings[0];
        assert.ok(first);
        first.provenance = 'signed';
      }),
    ],
    [
      'content from another producer',
      mutate((doc) => {
        (review(doc).content as Record<string, unknown>).producer = 'third-party-draft';
      }),
    ],
    [
      'recommendation from another producer',
      mutate((doc) => {
        (review(doc).recommendation as Record<string, unknown>).producer = 'third-party-draft';
      }),
    ],
    [
      'consistency from another producer',
      mutate((doc) => {
        (review(doc).consistency as Record<string, unknown>).producer = 'third-party-draft';
      }),
    ],
    [
      'facts not leading with the objective',
      mutate((doc) => {
        (review(doc).content as Record<string, unknown>).facts = ['A different opening claim'];
      }),
    ],
    [
      'alternative outcomes null',
      mutate((doc) => {
        const alternatives = (review(doc).content as Record<string, unknown>).alternatives as Array<
          Record<string, unknown>
        >;
        const first = alternatives[0];
        assert.ok(first);
        first.outcomes = null;
      }),
    ],
    [
      'alternative outcomes empty',
      mutate((doc) => {
        const alternatives = (review(doc).content as Record<string, unknown>).alternatives as Array<
          Record<string, unknown>
        >;
        const first = alternatives[0];
        assert.ok(first);
        first.outcomes = [];
      }),
    ],
    [
      'milestones null',
      mutate((doc) => {
        (review(doc).recommendation as Record<string, unknown>).milestones = null;
      }),
    ],
    [
      'unknowns null',
      mutate((doc) => {
        (review(doc).content as Record<string, unknown>).unknowns = null;
      }),
    ],
    [
      'exclusion meanings nulled under excluded ideas',
      mutate((doc) => {
        (review(doc).content as Record<string, unknown>).exclusions = null;
      }),
    ],
  ];
  for (const [name, raw] of cases) assert.throws(() => savedReviewRead(raw), /Invalid or mismatched/, name);
});
