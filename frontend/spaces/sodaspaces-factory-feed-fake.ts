import type {CommentDetail, FeedEntry} from './sodaspaces-factory-feed-response.js';
import {feedState, roleLabel} from './sodaspaces-factory-feed-thread.js';

// A made-up Feed for a small todo app, shown from the Feed's right-click
// menu so the cards can be seen without a running Factory. Ids are not
// 32-hex, so no reply drawer ever asks the server for this thread.

const project = 'demo/todo-app';

function human(id: string, at: number, author: string, body: string, issue?: number): CommentDetail {
  return {
    commentId: id,
    body,
    authorName: author,
    at,
    edited: false,
    removed: false,
    pipeline: {
      commentId: id,
      revision: 0,
      intake: 'recorded',
      judgment: 'applied',
      ...(issue
        ? {route: 'new_issue', draft: 'applied', effect: 'verified', issue}
        : {route: 'discussion', draft: 'pending', effect: 'pending'}),
    },
  };
}

/** Newest first, like a served feed page. */
export function fakeTodoFeed(repository: number, now = Math.floor(Date.now() / 1000)): FeedEntry[] {
  const ago = (minutes: number) => now - minutes * 60;
  const base = {repository, project};
  const entries: Omit<FeedEntry, 'seq'>[] = [
    {
      ...base,
      kind: 'record',
      id: 'fake-ask-archive',
      actor: 'product_lead',
      author: 'Product Lead',
      at: ago(2),
      issue: 15,
      status: 'route:needs-human',
      payload: {
        communication: {
          kind: 'ask',
          title: 'Should finished todos tidy themselves away?',
          summary:
            'Ticked-off todos pile up at the bottom; some lists have 400 of them. Two people asked for a cleaner list.',
          question: 'What should happen to a todo a week after it is ticked off?',
          questionState: 'open',
          options: ['Move it to an Archive list', 'Delete it', 'Leave it where it is'],
        },
      },
    },
    {
      ...base,
      kind: 'record',
      id: 'fake-flag-bounces',
      actor: 'test_pilot',
      author: 'Test Pilot',
      at: ago(4),
      issue: 14,
      status: 'run:attention',
      payload: {
        communication: {
          kind: 'flag',
          severity: 'warning',
          title: 'Reminder emails bounce for 3 accounts',
          summary:
            'The 8:00 reminder run sent 212 emails; 3 bounced because those accounts still have the placeholder address from the signup bug in #9.',
          evidenceRefs: ['logs/reminders-2026-10-29.txt'],
        },
      },
    },
    {
      ...base,
      kind: 'record',
      id: 'fake-merge-due-dates',
      actor: 'factory',
      author: 'Factory',
      at: ago(18),
      issue: 14,
      pr: 31,
      activity: 'merged',
      status: 'merged',
      summary: 'Merged “Due dates and morning reminders” into main.',
      payload: {merge: {mergedCommit: '4f2a9c1e7b', originalClosedUnix: ago(18)}},
    },
    {
      ...base,
      kind: 'record',
      id: 'fake-delivery',
      actor: 'jev',
      author: 'JEV',
      at: ago(21),
      issue: 14,
      pr: 31,
      activity: 'judgment',
      status: 'judgment:ready_to_merge',
      summary: 'Ready to merge: every check passed and the review finding is fixed.',
      payload: {
        judgment: {
          kind: 'delivery',
          outcome: 'ready_to_merge',
          effective: 'ready_to_merge',
          confidence: 0.92,
          probabilities: {ready_to_merge: 0.92, review_again: 0.06, needs_founder: 0.02},
        },
      },
    },
    {
      ...base,
      kind: 'record',
      id: 'fake-verify',
      actor: 'test_pilot',
      author: 'Test Pilot',
      at: ago(25),
      issue: 14,
      pr: 31,
      status: 'verify:finished:succeeded',
      summary: 'Tried the feature in a browser: set a due date, waited for the reminder, ticked the todo off.',
      payload: {
        verification: {
          outcome: 'passed',
          checks: [
            {id: 'unit tests (148)', outcome: 'passed'},
            {id: 'pick a due date on a new todo', outcome: 'passed'},
            {id: 'overdue todos sort to the top', outcome: 'passed'},
            {id: 'reminder arrives at 8:00 local time', outcome: 'passed'},
          ],
        },
      },
    },
    {
      ...base,
      kind: 'record',
      id: 'fake-fix',
      actor: 'fixer',
      author: 'Fixer',
      at: ago(34),
      issue: 14,
      pr: 31,
      status: 'correct:finished:succeeded',
      summary: 'Overdue todos now sort before today’s, and todos without a date go last.',
      payload: {
        correction: {
          newCandidate: 'b81d03e',
          findings: [
            {
              key: 'sort-overdue',
              severity: 'major',
              defect: 'Overdue todos sorted below today’s.',
              location: {path: 'src/todos/sort.ts', line: 12},
            },
          ],
          dispositions: [{key: 'sort-overdue', disposition: 'fixed', changedPaths: ['src/todos/sort.ts']}],
        },
      },
    },
    {
      ...base,
      kind: 'record',
      id: 'fake-review',
      actor: 'inspector',
      author: 'Inspector',
      at: ago(46),
      issue: 14,
      pr: 31,
      status: 'review:request-changes',
      summary: 'One problem: overdue todos sink below today’s.',
      payload: {
        review: {
          verdict: 'request_changes',
          summary: 'Due dates save and display correctly, but the list order hides what is late.',
          newFindings: [
            {
              key: 'sort-overdue',
              severity: 'major',
              defect: 'The list sorts by due date descending, so yesterday’s todo appears below today’s.',
              evidence: 'Created todos due yesterday and today; yesterday’s rendered second.',
              requiredOutcome: 'Overdue first, then today, then later, then undated.',
              location: {path: 'src/todos/sort.ts', line: 12},
            },
          ],
        },
      },
    },
    {
      ...base,
      kind: 'post',
      id: 'fake-answer-tz',
      author: 'Ada',
      at: ago(70),
      issue: 14,
      status: 'route:discussion',
      comment: human('fake-answer-tz', ago(70), 'Ada', 'Local time, definitely. Nobody wants a reminder at 3am.'),
    },
    {
      ...base,
      kind: 'record',
      id: 'fake-ask-tz',
      actor: 'builder',
      author: 'Builder',
      at: ago(78),
      issue: 14,
      status: 'implement:needs-human',
      payload: {
        communication: {
          kind: 'ask',
          title: 'Which clock should the morning reminder follow?',
          summary: 'Todos store a date without a time. The reminder needs a time zone to decide when “morning” is.',
          question: 'Send the reminder at 8:00 in each person’s own time zone, or 8:00 server time?',
          questionState: 'answered',
          options: ['8:00 in each person’s time zone', '8:00 server time (UTC)'],
        },
      },
    },
    {
      ...base,
      kind: 'record',
      id: 'fake-progress',
      actor: 'builder',
      author: 'Builder',
      at: ago(90),
      issue: 14,
      status: 'implement:running',
      payload: {
        communication: {
          kind: 'partial',
          title: 'Building due dates',
          summary: 'Two of four tasks done.',
          completed: ['Add due_at column and migration', 'Accept due_at in the todos API'],
          remaining: ['Date picker on the todo row', 'Morning reminder job'],
        },
      },
    },
    {
      ...base,
      kind: 'record',
      id: 'fake-plan',
      actor: 'architect',
      author: 'Architect',
      at: ago(120),
      issue: 14,
      status: 'plan:finished:succeeded',
      summary: 'Plan for due dates and reminders: four small tasks.',
      payload: {
        plan: {
          approach:
            'Store a date-only due_at on each todo, show it on the row, and run a daily job that emails people whose todos are due today.',
          findings: [
            'Todos live in one table; adding a nullable column is safe.',
            'There is already a nightly job runner for cleanup.',
          ],
          unresolved: ['Which time zone “morning” means.'],
          tasks: [
            {id: 't1', objective: 'Add due_at column and migration', lane: 'data'},
            {id: 't2', objective: 'Accept due_at in the todos API', lane: 'api', dependsOn: ['t1']},
            {id: 't3', objective: 'Date picker on the todo row', lane: 'ui', dependsOn: ['t2']},
            {id: 't4', objective: 'Morning reminder job', lane: 'jobs', dependsOn: ['t1']},
          ],
        },
      },
    },
    {
      ...base,
      kind: 'record',
      id: 'fake-route',
      actor: 'product_lead',
      author: 'Product Lead',
      at: ago(130),
      issue: 14,
      status: 'route:new_issue',
      payload: {
        communication: {
          kind: 'record_decision',
          title: 'Ada’s request becomes issue #14 “Due dates and morning reminders”',
          summary: 'Small, clear and asked for twice before; it goes to the front of the queue.',
          decision: 'One issue covering both the due date and the reminder.',
          rationale: 'A reminder is useless without a date, and the date alone does not solve “I keep forgetting”.',
          alternatives: ['Two issues: dates first, reminders later'],
        },
      },
    },
    {
      ...base,
      kind: 'post',
      id: 'fake-request',
      author: 'Ada',
      at: ago(135),
      status: 'route:new_issue',
      comment: human(
        'fake-request',
        ago(135),
        'Ada',
        'Can todos have a due date? And ping me the morning something is due — I keep forgetting the bins.',
        14
      ),
    },
    {
      ...base,
      kind: 'record',
      id: 'fake-learned',
      actor: 'strategist',
      author: 'Strategist',
      at: ago(60 * 26),
      status: 'completed',
      payload: {
        communication: {
          kind: 'learned',
          title: 'Most todos are done within a day — or never',
          summary:
            'Last week 61% of new todos were ticked off within a day; of the rest, only 1 in 10 was ever finished. Due dates may help the long tail.',
        },
      },
    },
    {
      ...base,
      kind: 'record',
      id: 'fake-dark-mode',
      actor: 'factory',
      author: 'Factory',
      at: ago(60 * 30),
      issue: 11,
      pr: 27,
      activity: 'merged',
      status: 'merged',
      summary: 'Merged “Dark mode follows the system setting” into main.',
      payload: {merge: {mergedCommit: '9c07e2d4aa', originalClosedUnix: ago(60 * 30)}},
    },
  ];
  return entries.map((entry, index) => ({...entry, seq: 1000 - index}) as FeedEntry);
}

export interface FakeFeedFilters {
  view: string;
  search: string;
  role: string;
}

// fakeFeedFilter answers the Feed's tabs, search and author filter the way
// the server would, so the fake feed responds to them too.
export function fakeFeedFilter(entries: FeedEntry[], filters: FakeFeedFilters): FeedEntry[] {
  const search = filters.search.toLowerCase();
  return entries.filter((entry) => {
    const state = feedState(entry);
    const payload = entry.payload ?? {};
    if (filters.view === 'attention' && state !== 'needs_you' && state !== 'problem') return false;
    if (
      filters.view === 'decisions' &&
      state !== 'decided' &&
      !payload.judgment &&
      payload.communication?.kind !== 'ask'
    )
      return false;
    if (filters.view === 'evidence' && !payload.verification && !payload.review && !payload.plan && !payload.merge)
      return false;
    if (
      filters.role === 'human'
        ? entry.kind !== 'post'
        : filters.role && roleLabel(entry.actor) !== roleLabel(filters.role)
    )
      return false;
    return !search || JSON.stringify([entry.summary, entry.comment?.body, payload]).toLowerCase().includes(search);
  });
}
