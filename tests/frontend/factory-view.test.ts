import test from 'node:test';
import assert from 'node:assert/strict';
import {
  factoryStatusFrame,
  factoryOutputFrame,
  factoryClosedReason,
  factoryRunStatusResponse,
} from '../../frontend/spaces/sodaspaces-api';

const runId = 'a'.repeat(32);
const repositoryId = '7';
const status = {
  type: 'status',
  run_id: runId,
  phase: 'running',
  container: 'c'.repeat(64),
  unit: `soda-factory-${runId}.service`,
  invocation: 'd'.repeat(32),
  reason: '',
  live: true,
  terminal: false,
  exit_code: null,
};

test('factory status frames bind the exact run and process', () => {
  const parsed = factoryStatusFrame(status, runId);
  assert.equal(parsed.phase, 'running');
  assert.equal(parsed.live, true);
  assert.equal(parsed.exit_code, null);
  const done = factoryStatusFrame({...status, phase: 'completed', live: false, terminal: true, exit_code: 0}, runId);
  assert.equal(done.exit_code, 0);
  const unbound = factoryStatusFrame(
    {...status, phase: 'approved', container: '', unit: '', invocation: '', live: false},
    runId
  );
  assert.equal(unbound.container, '');
  for (const delta of [
    {run_id: 'b'.repeat(32)},
    {phase: 'launching'},
    {live: 'yes'},
    {container: 'short'},
    {unit: 'soda-factory-short.service'},
    {invocation: 'short'},
    {reason: 'Stale Container'},
    {exit_code: 256},
    {exit_code: '0'},
    {type: 'output'},
    {extra: 1},
  ])
    assert.throws(() => factoryStatusFrame({...status, ...delta}, runId));
});

test('factory output frames carry server cursors over exact bytes', () => {
  const parsed = factoryOutputFrame({
    type: 'output',
    data: 'aGVsbG8=',
    cursor: 0,
    next: 5,
    gap: false,
    truncated: false,
  });
  assert.deepEqual([...parsed.bytes], [104, 101, 108, 108, 111]);
  assert.equal(parsed.next, 5);
  const gap = factoryOutputFrame({type: 'output', data: '', cursor: 9, next: 9, gap: true, truncated: false});
  assert.equal(gap.gap, true);
  for (const frame of [
    {type: 'output', data: 'aGVsbG8=', cursor: 0, next: 6, gap: false, truncated: false},
    {type: 'output', data: '!!!', cursor: 0, next: 0, gap: false, truncated: false},
    {type: 'output', data: '', cursor: 0, next: -1, gap: false, truncated: false},
    {type: 'output', data: '', cursor: 2, next: 1, gap: false, truncated: false},
    {type: 'output', data: '', cursor: 0, next: 0, gap: false, truncated: 'no'},
    {type: 'status', data: '', cursor: 0, next: 0, gap: false, truncated: false},
    {type: 'output', data: '', cursor: 0, next: 0, gap: false, truncated: false, extra: 0},
  ])
    assert.throws(() => factoryOutputFrame(frame));
});

test('factory close frames end the view with a bounded reason', () => {
  assert.equal(factoryClosedReason({type: 'closed', reason: 'eof'}), 'eof');
  assert.equal(factoryClosedReason({type: 'closed', reason: 'authority_lost'}), 'authority_lost');
  for (const frame of [
    {type: 'closed', reason: ''},
    {type: 'closed', reason: 'x'.repeat(65)},
    {type: 'closed'},
    {type: 'status', reason: 'eof'},
  ])
    assert.throws(() => factoryClosedReason(frame));
});

const record = {
  id: runId,
  project_id: 'p0123456789abcdef01234567',
  role: 'coder',
  harness: 'codex-0.157.1',
  model: 'test',
  input_sha: 'e'.repeat(40),
  reconciled: false,
};

test('factory run status binds record, binding and host state', () => {
  const pending = factoryRunStatusResponse({run: record}, runId, repositoryId);
  assert.equal(pending.role, 'coder');
  assert.equal(pending.view, undefined);
  assert.equal(pending.state, undefined);
  const full = factoryRunStatusResponse(
    {
      run: {...record, outcome: 'failed', summary: 'exit 1', reconciled: true},
      view: {repository: '7', issue: '42', attempt: 'attempt-1'},
      state: {
        phase: 'failed',
        live: false,
        terminal: true,
        output_truncated: true,
        container: 'c'.repeat(64),
        unit: `soda-factory-${runId}.service`,
        invocation: 'd'.repeat(32),
        exit_code: 1,
        reason: 'execution-failed',
        retirement: 'confirmed',
        output: 'x'.repeat(16384),
      },
    },
    runId,
    repositoryId
  );
  assert.equal(full.view?.issue, '42');
  assert.equal(full.state?.exit_code, 1);
  assert.equal(full.state?.output?.length, 16384);
  for (const delta of [
    {run: {...record, id: 'b'.repeat(32)}},
    {run: {...record, project_id: 'nope'}},
    {run: {...record, harness: ''}},
    {run: {...record, input_sha: 'short'}},
    {run: {...record, outcome: 'stopped'}},
    {run: record, view: {repository: '8'}},
    {run: record, view: {repository: '7', issue: '0'}},
    {run: record, state: {phase: 'launching', live: false, terminal: false, output_truncated: false}},
    {run: record, state: {phase: 'failed', live: false, terminal: true, output_truncated: true}},
    {
      run: record,
      state: {phase: 'failed', live: false, terminal: true, output_truncated: false, output: 'x'.repeat(16385)},
    },
  ])
    assert.throws(() => factoryRunStatusResponse(delta, runId, repositoryId));
});
