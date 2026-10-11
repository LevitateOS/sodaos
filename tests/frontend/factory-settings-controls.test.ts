import test from 'node:test';
import assert from 'node:assert/strict';
import {setupProjectControlsDriver, fixture, refresh, writes} from './fixtures/project-controls-driver';
setupProjectControlsDriver();

test('Factory settings render current authority and submit capacity through the native session mount', async (t) => {
  const page = await fixture(t);
  await page.evaluate(async () => {
    window.drawerFixture.api.setPresentation('settings');
  });
  await refresh(page);
  await page.getByRole('tab', {name: 'Factory'}).click();
  await page.waitForTimeout(250);
  assert.equal(await page.locator('[data-view="factory"]').getAttribute('aria-selected'), 'true');
  assert.equal(await page.locator('soda-factory-settings').count(), 1);
  await page.locator('soda-factory-settings').evaluate(async (el) => {
    await (el as HTMLElement & {updateComplete: Promise<unknown>}).updateComplete;
  });
  assert.match((await page.locator('soda-factory-settings').textContent()) || '', /Factory authority/);
  assert.match(await page.locator('[data-factory-settings]').innerText(), /Capacity: 2 concurrent runs, 8 queued/);

  await page.getByRole('button', {name: 'Save capacity'}).click();
  await page.getByRole('status').filter({hasText: 'Change confirmed by the current API.'}).waitFor();
  const sent = await writes(page);
  const capacity = sent.find((call) => call.url.endsWith('/api/factory/capacity'));
  assert(capacity);
  assert.equal(capacity.method, 'PUT');
  assert.equal(capacity.headers['x-extension-session-generation'], 'fixture-generation');
  const payload = JSON.parse(capacity.body || '{}') as Record<string, unknown>;
  assert.match(String(payload.command_id), /^[a-f0-9]{32}$/);
  assert.deepEqual(
    {...payload, command_id: '<generated>'},
    {
      command_id: '<generated>',
      expected_revision: 4,
      max_concurrent_runs: 2,
      max_queued: 8,
    }
  );
});

test('Factory settings expose stale authority without replaying a capacity command', async (t) => {
  const page = await fixture(t);
  await page.evaluate(async () => {
    window.drawerFixture.api.setPresentation('settings');
  });
  await refresh(page);
  await page.getByRole('tab', {name: 'Factory'}).click();
  await page.locator('soda-factory-settings').evaluate(async (el) => {
    await (el as HTMLElement & {updateComplete: Promise<unknown>}).updateComplete;
  });
  await page.evaluate(() =>
    window.drawerFixture.setReply(async (call) =>
      call.method === 'PUT' && call.url.endsWith('/api/factory/capacity')
        ? Response.json({error: {code: 'stale_revision'}}, {status: 412})
        : null
    )
  );
  await page.getByRole('button', {name: 'Save capacity'}).click();
  await page.getByRole('status').filter({hasText: 'Authority changed.'}).waitFor();
  const sent = await writes(page);
  assert.equal(sent.filter((call) => call.url.endsWith('/api/factory/capacity')).length, 1);
});

test('preparation controls render API evidence and stay closed without server-derived decision inputs', async (t) => {
  const page = await fixture(t);
  await page.evaluate(() =>
    window.drawerFixture.setReply(async (call) =>
      call.url.endsWith('/api/repositories/7/factory')
        ? Response.json({repository: '7', appliance: {revision: 1, max_concurrent_runs: 1, max_queued: 0}, effective: {effective: false, dispatch_open: false, missing: []}})
        : call.url.endsWith('/api/environments/p0123456789abcdef01234567/preparation')
          ? Response.json({project: 'test-project', hold: true, hold_revision: 6, requirements: '', approval: '', preparations: [{id: 'prep-1', role: 'coder', phase: 'complete', ready: true, source_commit: 'a'.repeat(40), setup_digest: 'b'.repeat(64)}]})
          : null
    )
  );
  await page.evaluate(async () => {
    window.drawerFixture.api.setPresentation('settings');
  });
  await refresh(page);
  await page.getByRole('tab', {name: 'Factory'}).click();
  const settings = page.locator('soda-factory-settings');
  await settings.evaluate(async (el) => {
    await (el as HTMLElement & {updateComplete: Promise<unknown>}).updateComplete;
  });
  assert.match(await settings.innerText(), /coder · complete · ready/);
  assert.equal(await settings.locator('[name="decision_id"], [name="inputs_digest"], [name="effects_digest"], [name="readiness_digest"]').count(), 0);
  assert.equal(await settings.getByRole('button', {name: 'Record exact-input acceptance'}).isDisabled(), true);
  assert.equal(await settings.getByRole('button', {name: 'Record separate approval'}).isDisabled(), true);
  assert.equal((await writes(page)).length, 0);
});

test('preparation decisions copy reviewed server preview facts and keep approval separate', async (t) => {
  const page = await fixture(t);
  const sourceCommit = 'a'.repeat(40);
  const setupDigest = 'b'.repeat(64);
  const inputsDigest = 'c'.repeat(64);
  const effectsDigest = 'd'.repeat(64);
  const readinessDigest = 'e'.repeat(64);
  await page.evaluate(({sourceCommit, setupDigest, inputsDigest, effectsDigest, readinessDigest}) => {
    window.drawerFixture.setReply(async (call) =>
      call.url.endsWith('/api/repositories/7/factory')
        ? Response.json({repository: '7', appliance: {revision: 1, max_concurrent_runs: 1, max_queued: 0}})
        : call.url.endsWith('/api/environments/p0123456789abcdef01234567/preparation')
          ? Response.json({
              project: 'test-project',
              hold: true,
              preparations: [],
              decision_preview: {
                requirements: {
                  source_commit: sourceCommit,
                  setup_digest: setupDigest,
                  inputs_digest: inputsDigest,
                  source_inputs: [{path: 'setup.sh', digest: 'f'.repeat(64)}],
                  predecessor: 'requirements-head',
                },
                approval: {
                  requirement: 'requirement-current',
                  effects_digest: effectsDigest,
                  readiness_digest: readinessDigest,
                  predecessor: '',
                  roles: [{role: 'coder', source_commit: sourceCommit, setup_digest: setupDigest, tools: 'node', setup_exit: 0, check_exit: 0}],
                },
              },
            })
          : null
    );
  }, {sourceCommit, setupDigest, inputsDigest, effectsDigest, readinessDigest});
  await page.evaluate(async () => window.drawerFixture.api.setPresentation('settings'));
  await refresh(page);
  await page.getByRole('tab', {name: 'Factory'}).click();
  const settings = page.locator('soda-factory-settings');
  await settings.evaluate(async (el) => {
    await (el as HTMLElement & {updateComplete: Promise<unknown>}).updateComplete;
  });
  assert.equal(await settings.getByRole('button', {name: 'Record exact-input acceptance'}).isDisabled(), false);
  assert.equal(await settings.getByRole('button', {name: 'Record separate approval'}).isDisabled(), false);

  await settings.getByRole('button', {name: 'Record exact-input acceptance'}).click();
  await settings.getByRole('status').filter({hasText: 'Change confirmed by the current API.'}).waitFor();
  let calls = await writes(page);
  const acceptance = calls.find((call) => call.url.endsWith('/preparation/acceptances'));
  assert(acceptance);
  const accepted = JSON.parse(acceptance.body || '{}') as Record<string, unknown>;
  assert.match(String(accepted.command_id), /^[a-f0-9]{32}$/);
  assert.match(String(accepted.decision_id), /^d[a-f0-9]{24}$/);
  assert.equal(accepted.predecessor, 'requirements-head');
  assert.equal(accepted.source_commit, sourceCommit);
  assert.equal(accepted.setup_digest, setupDigest);
  assert.equal(accepted.inputs_digest, inputsDigest);
  assert.deepEqual(accepted.source_inputs, {'setup.sh': 'f'.repeat(64)});
  assert.equal(calls.filter((call) => call.url.endsWith('/preparation/actions')).length, 0);

  await settings.getByRole('button', {name: 'Record separate approval'}).click();
  await settings.getByRole('status').filter({hasText: 'separately verify current live services and resources'}).waitFor();
  assert.equal((await writes(page)).filter((call) => call.url.endsWith('/preparation/actions')).length, 0);
  await settings.locator('[name="verified"]').check();
  await settings.getByRole('button', {name: 'Record separate approval'}).click();
  await settings.getByRole('status').filter({hasText: 'Change confirmed by the current API.'}).waitFor();
  calls = await writes(page);
  const approval = calls.find((call) => call.url.endsWith('/preparation/actions'));
  assert(approval);
  const approved = JSON.parse(approval.body || '{}') as Record<string, unknown>;
  assert.equal(approved.action, 'approve');
  assert.equal(approved.predecessor, undefined);
  assert.equal(approved.requirement, 'requirement-current');
  assert.equal(approved.effects_digest, effectsDigest);
  assert.equal(approved.readiness_digest, readinessDigest);
  assert.equal(approved.verified, true);
  assert.equal(calls.filter((call) => call.url.endsWith('/preparation/actions')).length, 1);
});

test('provider budget and sponsorship use observed connection and grant records and surface stale authority once', async (t) => {
  const page = await fixture(t);
  await page.evaluate(() =>
    window.drawerFixture.setReply(async (call) => {
      if (call.url.endsWith('/api/repositories/7/factory'))
        return Response.json({repository: '7', policy: null, operator_grant: null, appliance: {revision: 4, max_concurrent_runs: 2, max_queued: 8}, sponsorships: [], environment_grant: null, preparation: null, queue: {queued: 0, active: 0}, effective: {effective: false, dispatch_open: false, missing: []}});
      if (call.url.endsWith('/api/identity/connections'))
        return Response.json([{id: 'connection-observed', provider_id: 'codex', owner_id: '9', label: 'Primary', generation: 7, state: 'active'}]);
      if (call.url.endsWith('/api/identity/connections/connection-observed/grants'))
        return Response.json([{id: 'grant-observed', connection_id: 'connection-observed', project_id: 'p123456789abcdef012345678', revision: 2, revoked: false}]);
      if (call.url.endsWith('/api/factory/connections/connection-observed/usage-budget'))
        return Response.json({connection: 'connection-observed', revision: 3, rolling_minutes: 120});
      if (call.method === 'PUT' && call.url.includes('/factory/sponsorships/'))
        return Response.json({error: {code: 'stale_credential_generation'}}, {status: 409});
      return null;
    })
  );
  await page.evaluate(async () => {
    window.drawerFixture.api.setPresentation('settings');
    await window.drawerFixture.api.ready;
  });
  await refresh(page);
  await page.getByRole('tab', {name: 'Factory'}).click();
  const settings = page.locator('soda-factory-settings');
  await settings.evaluate(async (el) => {
    await (el as HTMLElement & {updateComplete: Promise<unknown>}).updateComplete;
  });
  assert.match(await settings.innerText(), /Primary · active/);
  assert.match(await settings.innerText(), /120 minutes \(revision 3\)/);
  assert.equal(await settings.locator('[name="connection"]').count(), 0);
  assert.equal(await settings.locator('[name="grant_id"]').inputValue(), '');
  await settings.locator('[name="grant_id"]').selectOption('grant-observed');
  await settings.locator('[name="coder"]').check();
  await settings.getByRole('button', {name: 'Save sponsorship'}).click();
  await settings.getByRole('status').filter({hasText: 'Authority changed.'}).waitFor();
  const writesSent = await writes(page);
  const sponsor = writesSent.find((call) => call.url.includes('/factory/sponsorships/'));
  assert(sponsor);
  assert.equal(sponsor.method, 'PUT');
  const payload = JSON.parse(sponsor.body || '{}') as Record<string, unknown>;
  assert.equal(payload.grant_id, 'grant-observed');
  assert.equal(payload.generation, 7);
  assert.deepEqual(payload.roles, ['coder']);
  assert.equal(writesSent.filter((call) => call.url.includes('/factory/sponsorships/')).length, 1);
});

test('provider budget reports a forbidden response and does not repeat the command', async (t) => {
  const page = await fixture(t);
  await page.evaluate(() =>
    window.drawerFixture.setReply(async (call) => {
      if (call.url.endsWith('/api/repositories/7/factory'))
        return Response.json({repository: '7', appliance: {revision: 1, max_concurrent_runs: 1, max_queued: 0}, sponsorships: [], effective: {effective: false, dispatch_open: false, missing: []}});
      if (call.url.endsWith('/api/identity/connections'))
        return Response.json([{id: 'connection-observed', owner_id: '9', label: 'Primary', generation: 7, state: 'active'}]);
      if (call.url.endsWith('/api/identity/connections/connection-observed/grants')) return Response.json([]);
      if (call.url.endsWith('/api/factory/connections/connection-observed/usage-budget'))
        return call.method === 'PUT'
          ? Response.json({error: {code: 'connection_owner_required'}}, {status: 403})
          : Response.json({connection: 'connection-observed', revision: 0, rolling_minutes: 120});
      return null;
    })
  );
  await page.evaluate(async () => {
    window.drawerFixture.api.setPresentation('settings');
    await window.drawerFixture.api.ready;
  });
  await refresh(page);
  await page.getByRole('tab', {name: 'Factory'}).click();
  const settings = page.locator('soda-factory-settings');
  await settings.evaluate(async (el) => {
    await (el as HTMLElement & {updateComplete: Promise<unknown>}).updateComplete;
  });
  let sessionInvalid = 0;
  await settings.evaluate((element) => element.addEventListener('soda-session-invalid', () => { (element as HTMLElement).dataset.invalid = 'yes'; }));
  await settings.getByRole('button', {name: 'Save connection budget'}).click();
  await settings.getByRole('status').filter({hasText: 'not confirmed (403)'}).waitFor();
  sessionInvalid = (await settings.getAttribute('data-invalid')) === 'yes' ? 1 : 0;
  assert.equal(sessionInvalid, 1);
  const budgetWrites = (await writes(page)).filter((call) => call.url.endsWith('/api/factory/connections/connection-observed/usage-budget'));
  assert.equal(budgetWrites.length, 1);
});
