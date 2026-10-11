import test from 'node:test';
import assert from 'node:assert/strict';
import {activityTimestamp, parseFactoryActivity} from '../../frontend/spaces/sodaspaces-factory-activity';

const repository = '7';
const settings = {issue_created: true, pr_created: true, verified_merge: true};
const payload = {
  repository,
  repository_path: '/soda-tester/demo',
  revision: 3,
  observed_unix: 1720000300,
  truncated: true,
  settings,
  items: [
    {
      id: 'issue:12',
      kind: 'issue_created',
      issue: '12',
      created_unix: 1720000000,
      summary: 'Issue #12 created from accepted product intent.',
      url: '/soda-tester/demo/issues/12',
    },
    {
      id: 'pr:34',
      kind: 'pr_created',
      issue: '12',
      pr: '34',
      created_unix: 1720000100,
      summary: 'Pull request #34 opened for issue #12.',
      url: '/soda-tester/demo/pulls/34',
    },
    {
      id: 'merge:12',
      kind: 'verified_merge',
      issue: '12',
      created_unix: 0,
      summary: 'Issue #12 merged and closed.',
      url: '/soda-tester/demo/issues/12',
    },
  ],
};

test('factory activity keeps repository binding, stamps and safe links', () => {
  const parsed = parseFactoryActivity(payload, repository);
  assert.equal(parsed.repository, repository);
  assert.equal(parsed.revision, 3);
  assert.equal(parsed.observed_unix, 1720000300);
  assert.equal(parsed.truncated, true);
  assert.deepEqual(parsed.settings, settings);
  assert.equal(parsed.items.length, 3);
  assert.equal(parsed.items[0]?.created_unix, 1720000000);
  assert.equal(parsed.items[1]?.url, '/soda-tester/demo/pulls/34');
  assert.equal(parsed.items[2]?.created_unix, 0);
  for (const delta of [
    {repository: '8'},
    {repository_path: '/soda-tester/demo/issues/12/..'},
    {repository_path: 'soda-tester/demo'},
    {revision: -1},
    {revision: 1.5},
    {observed_unix: -1},
    {truncated: 'yes'},
    {settings: {...settings, issue_created: 'yes'}},
    {settings: {issue_created: true, pr_created: true}},
    {items: Array.from({length: 61}, () => payload.items[0])},
  ])
    assert.throws(() => parseFactoryActivity({...payload, ...delta}, repository));
});

test('factory activity items bind kind, target and exact native link', () => {
  const [first] = payload.items;
  const good = parseFactoryActivity(payload, repository).items[0];
  assert.equal(good?.id, first?.id);
  for (const item of [
    {...first, kind: 'issue_labeled'},
    {...first, issue: '0'},
    {...first, url: '/soda-tester/demo/issues/13'},
    {...first, url: '/other/demo/issues/12'},
    {...first, id: ''},
    {...first, summary: ''},
    {...first, created_unix: -1},
    {...first, created_unix: '1720000000'},
    {id: 'pr:34', kind: 'pr_created', issue: '12', created_unix: 1, summary: 'x', url: '/soda-tester/demo/pulls/34'},
  ])
    assert.throws(() => parseFactoryActivity({...payload, items: [item]}, repository));
});

test('activity timestamps stay unknown instead of inventing a date', () => {
  const known = activityTimestamp(1720000000);
  assert.ok(known);
  assert.equal(Date.parse(known.iso), 1720000000 * 1000);
  assert.ok(known.text.length > 0);
  assert.equal(activityTimestamp(0), undefined);
  assert.equal(activityTimestamp(-5), undefined);
  assert.equal(activityTimestamp(1.5), undefined);
  assert.equal(activityTimestamp(Number.NaN), undefined);
});
