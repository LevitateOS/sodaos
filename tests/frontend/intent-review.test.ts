import {test} from 'bun:test';
import assert from 'node:assert/strict';
import {savedReviewRead} from '../../frontend/spaces/sodaspaces-intent-response';

const current = `{
  "intent": "d000000000000000000000000",
  "head_revision": "d222222222222222222222222",
  "status": "current",
  "review": {
    "intent": "d000000000000000000000000",
    "basis_revision": "d111111111111111111111111",
    "result_revision": "d222222222222222222222222",
    "content": {
      "producer": "soda-authored-v1",
      "supported": true,
      "intent": "d000000000000000000000000",
      "revision": "d111111111111111111111111",
      "objective": "Time-off requests my manager can approve",
      "facts": ["Time-off requests my manager can approve"],
      "alternatives": [
        {
          "key": "single_person",
          "name": "Small single-person workflow",
          "audience": "One person completing their own request without a separate decider.",
          "summary": "The person records what they need and sees the recorded result directly.",
          "steps": [
            {"actor": "The person", "action": "describes what they need", "result": "The request is recorded"},
            {"actor": "The person", "action": "confirms the recorded request", "result": "The request counts as submitted"}
          ],
          "visibility": "Only the person sees their own request unless the description says otherwise.",
          "consequences": "No waiting on another person, and no second check before the request counts.",
          "evidence": "Authored pattern; no external research was performed.",
          "outcomes": ["The person completes a time-off request"]
        },
        {
          "key": "shared_review",
          "name": "Shared review workflow",
          "audience": "A person making a request and another person deciding on it.",
          "summary": "The person submits a request; a reviewer decides, and the person sees the decision.",
          "steps": [
            {"actor": "The person", "action": "describes what they need and submits the request", "result": "The reviewer can see the request"},
            {"actor": "The reviewer", "action": "makes a decision", "result": "The person sees the decision", "branches": [
              {"outcome": "Approved", "result": "The person sees confirmation"},
              {"outcome": "Declined", "result": "The person sees the reason and can revise the request"}
            ]}
          ],
          "visibility": "The person and the reviewer see the request; who else can see it is not established.",
          "consequences": "A decision takes reviewer time, and a declined request needs revision.",
          "evidence": "Authored pattern; no external research was performed.",
          "outcomes": ["The person completes a time-off request"]
        }
      ],
      "excluded": ["d666666666666666666666666"],
      "unknowns": ["Who can see a request beyond the people named in the walkthrough is not established."]
    },
    "challenges": [
      {"idea": "d444444444444444444444444", "assumption": "Everyone can see each request", "consequence": "Dates are exposed", "recommendation": "Confirm whether the visibility holds, or explain the correction."}
    ],
    "settled": ["d555555555555555555555555"],
    "recommendation": {
      "producer": "soda-authored-v1",
      "intent": "d000000000000000000000000",
      "revision": "d111111111111111111111111",
      "objective": "Time-off requests my manager can approve",
      "recommended": "shared_review",
      "alternative": {
        "key": "shared_review",
        "name": "Shared review workflow",
        "audience": "A person making a request and another person deciding on it.",
        "summary": "The person submits a request; a reviewer decides, and the person sees the decision.",
        "steps": [
          {"actor": "The person", "action": "describes what they need and submits the request", "result": "The reviewer can see the request"},
          {"actor": "The reviewer", "action": "makes a decision", "result": "The person sees the decision", "branches": [
            {"outcome": "Approved", "result": "The person sees confirmation"},
            {"outcome": "Declined", "result": "The person sees the reason and can revise the request"}
          ]}
        ],
        "visibility": "The person and the reviewer see the request; who else can see it is not established.",
        "consequences": "A decision takes reviewer time, and a declined request needs revision.",
        "evidence": "Authored pattern; no external research was performed.",
        "outcomes": ["The person completes a time-off request"]
      },
      "outcomes": ["The person completes a time-off request"],
      "constraints": [],
      "excluded": ["d666666666666666666666666"],
      "milestones": [
        {"name": "The person: describes what they need and submits the request", "result": "The reviewer can see the request"},
        {"name": "The reviewer: makes a decision", "result": "The person sees the decision"},
        {"name": "If Approved", "result": "The person sees confirmation"},
        {"name": "If Declined", "result": "The person sees the reason and can revise the request"}
      ],
      "advice": "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
    },
    "consistency": {
      "producer": "soda-authored-v1",
      "intent": "d000000000000000000000000",
      "revision": "d111111111111111111111111",
      "mismatches": [],
      "confidence": 0.9,
      "advice": "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"
    },
    "saved_unix": 1791824831
  }
}`;

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
  ];
  for (const [name, raw] of cases) assert.throws(() => savedReviewRead(raw), /Invalid or mismatched/, name);
});
