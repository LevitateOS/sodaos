import test from 'node:test';
import assert from 'node:assert/strict';
import {attentionReason, terminalObservation} from '../../frontend/spaces/sodaspaces-attention';
import type {TerminalMetadata} from '../../frontend/spaces/sodaspaces-api';
const now = 1000000;
const metadata: TerminalMetadata = {id: 'a'.repeat(32), request_id: 'b'.repeat(32), environment_id: 'p' + '1'.repeat(24), repository_id: '7', user_id: '1', login: 'alice', name: '', created_at: 1, hard_until: 10000, retain_until: 0, effective_until: 10000, ready: true, attached: true, state: 'ready'};
test('attention is bounded lifecycle observation, never output semantics or cleanup inference', () => {
  assert.equal(attentionReason(metadata, now, now, 'ready'), '');
  assert.match(attentionReason({...metadata, effective_until: 1300}, now, now), /five minutes/);
  assert.match(attentionReason({...metadata, effective_until: 999}, now, now), /deadline passed; cleanup not confirmed/);
  assert.match(attentionReason(metadata, now - 90001, now), /stale/);
  assert.match(attentionReason(undefined, now, now), /unavailable/);
  for (const state of ['connection-lost', 'attached-elsewhere', 'ending', 'unconfirmed', 'ended', 'unavailable'] as const) assert(attentionReason(metadata, now, now, state));
  assert.match(attentionReason({...metadata, state: 'unconfirmed'}, now, now, 'ready'), /unconfirmed/);
});
test('terminal observations reject bad identity, generations, unknown reasons and transcript fields', () => {
  const valid = {kind: 'output', generation: 2, id: metadata.id, requestId: metadata.request_id, state: 'ready'};
  assert.deepEqual(terminalObservation(valid), valid);
  for (const bad of [{...valid, generation: -1}, {...valid, generation: Infinity}, {...valid, id: null}, {...valid, state: 'Working'}, {...valid, data: 'private transcript'}, {...valid, requestId: 'newest'}]) assert.equal(terminalObservation(bad), undefined);
});
