// Spawns the lifecycle probe in its own bun test process: the probe needs
// SODA_FORGEJO_LAYOUT_ORIGIN set and a mocked playwright, both of which must
// not leak into the shared suite process and its origin skip-guards.
import test from 'node:test';
import assert from 'node:assert/strict';

test('component-browser factory lifecycle (isolated probe)', () => {
  const proc = Bun.spawnSync([process.execPath, 'test', import.meta.dir + '/fixtures/component-browser-lifecycle.probe.ts'], {
    env: {...process.env, SODA_FORGEJO_LAYOUT_ORIGIN: 'http://localhost:3300'},
  });
  const output = `${proc.stdout}\n${proc.stderr}`;
  assert.equal(proc.exitCode, 0, `lifecycle probe failed:\n${output}`);
  assert.match(output, /3 pass/, `lifecycle probe ran no tests:\n${output}`);
});
