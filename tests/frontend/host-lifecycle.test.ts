import test from 'node:test';
import assert from 'node:assert/strict';
import {readFile} from 'node:fs/promises';
import {setupWorkspaceDriver, fixture, openSession} from './fixtures/workspace-driver';
setupWorkspaceDriver();

// NFX-D05: components consume the host lifecycle instead of interpreting
// browser cache/visibility events as authority.
const lifecycleSources = [
  'frontend/spaces/sodaspaces-workspace.ts',
  'frontend/spaces/sodaspaces-terminal.ts',
  'frontend/spaces/sodaspaces-factory.ts',
  'frontend/spaces/sodaspaces-project.ts',
  'frontend/tailnet/soda-tailnet-page.ts',
];

test('no Soda component self-interprets pagehide, pageshow or visibilitychange', async () => {
  for (const path of lifecycleSources) {
    const source = await readFile(new URL('../../' + path, import.meta.url), 'utf8');
    assert(!/addEventListener\(\s*'(pagehide|pageshow)'/.test(source), `${path} still listens for cache events`);
    assert(!/addEventListener\(\s*'visibilitychange'/.test(source), `${path} still listens for visibilitychange`);
    assert(
      !/document\.addEventListener\(\s*'visibilitychange'/.test(source),
      `${path} still listens for visibilitychange`
    );
  }
});

test('workspace suspend, update and resume retain the exact terminal attachment', async (t) => {
  const page = await fixture(t);
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  await openSession(page, 'Build');
  const before = await page.evaluate(() => ({
    sockets: window.workspaceFixture.sockets.length,
    closed: window.workspaceFixture.sockets.at(-1)?.closed,
    writes: window.workspaceFixture.calls.filter((call) => call.method !== 'GET').length,
  }));
  assert.equal(before.closed, 0);
  const retained = await page.evaluate(async () => {
    const f = window.workspaceFixture,
      peer = f.sockets.at(-1),
      owner = document.querySelector('soda-spaces'),
      term = document.querySelector('soda-terminal');
    f.api.suspend();
    const sheltered = {closed: peer?.closed, count: f.sockets.length};
    f.api.update({mode: 'page', route: '/workspace', params: {}, visible: false});
    f.api.resume({sessionGeneration: f.generation});
    f.api.update({mode: 'page', route: '/workspace', params: {}, visible: true});
    await f.api.ready;
    return {
      ...sheltered,
      closedAfter: peer?.closed,
      sameSocket: f.sockets.at(-1) === peer,
      sameOwner: document.querySelector('soda-spaces') === owner,
      sameTerminal: document.querySelector('soda-terminal') === term,
      countAfter: f.sockets.length,
    };
  });
  assert.deepEqual(retained, {
    closed: 0,
    count: before.sockets,
    closedAfter: 0,
    sameSocket: true,
    sameOwner: true,
    sameTerminal: true,
    countAfter: before.sockets,
  });
  assert.equal(
    await page.evaluate(() => window.workspaceFixture.calls.filter((call) => call.method !== 'GET').length),
    before.writes
  );
});

test('workspace resume under a foreign generation retires without End', async (t) => {
  const page = await fixture(t);
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  await openSession(page, 'Build');
  const writes = await page.evaluate(
    () => window.workspaceFixture.calls.filter((call) => call.method !== 'GET').length
  );
  await page.evaluate(async () => {
    const f = window.workspaceFixture;
    f.api.suspend();
    f.api.resume({sessionGeneration: 'foreign-generation'});
    await f.api.ready;
  });
  assert.equal(await page.evaluate(() => window.workspaceFixture.sockets.at(-1)?.closed), 1);
  assert.equal(
    await page.evaluate(() => window.workspaceFixture.calls.filter((call) => call.method !== 'GET').length),
    writes
  );
});
