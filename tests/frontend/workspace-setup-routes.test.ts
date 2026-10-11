import {test} from 'bun:test';
import assert from 'node:assert/strict';
import {
  intentIdentity,
  newRouteFor,
  requestIdFromLocation,
  requestRouteFor,
} from '../../frontend/spaces/sodaspaces-workspace-setup';

const id = 'd0123456789abcdef01234567';

test('request id reads only from app-root request routes', () => {
  assert.equal(requestIdFromLocation('/workspace/requests/' + id), id);
  assert.equal(requestIdFromLocation('/forge/workspace/requests/' + id), id);
  for (const path of [
    '/workspace',
    '/workspace/new',
    '/workspace/requests',
    '/workspace/requests/',
    '/workspace/requests/bad',
    '/workspace/requests/d0123456789abcdef0123456',
    '/workspace/requests/d0123456789ABCDEF01234567',
    '/workspace/requests/' + id + '/extra',
    '/user2/repo1/workspace/requests/' + id,
    '/a/b/workspace/requests/' + id,
    '/issues',
    '/',
  ]) {
    assert.equal(requestIdFromLocation(path), '', path);
  }
});

test('request and new routes preserve the sub-path prefix', () => {
  assert.equal(requestRouteFor('/workspace/new', id), '/workspace/requests/' + id);
  assert.equal(requestRouteFor('/forge/workspace/new', id), '/forge/workspace/requests/' + id);
  assert.equal(requestRouteFor('/workspace/requests/' + id, id), '/workspace/requests/' + id);
  assert.equal(requestRouteFor('/', id), '/workspace/requests/' + id);
  assert.equal(newRouteFor('/workspace/requests/' + id), '/workspace/new');
  assert.equal(newRouteFor('/forge/workspace/requests/' + id), '/forge/workspace/new');
  assert.equal(newRouteFor('/'), '/workspace/new');
});

test('intent identity reads the saved/resumed detail only', () => {
  assert.equal(
    intentIdentity(new CustomEvent('intent-saved', {detail: {intent: id, revision: 'r', state: 'draft'}})),
    id
  );
  assert.equal(intentIdentity(new CustomEvent('intent-saved', {detail: null})), '');
  assert.equal(intentIdentity(new CustomEvent('intent-saved', {detail: {intent: 7}})), '');
  assert.equal(intentIdentity(new CustomEvent('intent-saved')), '');
});
