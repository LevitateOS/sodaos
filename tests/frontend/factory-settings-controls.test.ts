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
        ? Response.json({
            repository: '7',
            appliance: {revision: 1, max_concurrent_runs: 1, max_queued: 0},
            effective: {effective: false, dispatch_open: false, missing: []},
          })
        : call.url.endsWith('/api/environments/p0123456789abcdef01234567/preparation')
          ? Response.json({
              project: 'test-project',
              hold: true,
              hold_revision: 6,
              requirements: '',
              approval: '',
              preparations: [
                {
                  id: 'prep-1',
                  role: 'coder',
                  phase: 'complete',
                  ready: true,
                  source_commit: 'a'.repeat(40),
                  setup_digest: 'b'.repeat(64),
                },
              ],
            })
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
  assert.match(await settings.innerText(), /coder · complete · ID prep-1 · ready/);
  assert.equal(
    await settings
      .locator('[name="decision_id"], [name="inputs_digest"], [name="effects_digest"], [name="readiness_digest"]')
      .count(),
    0
  );
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
  await page.evaluate(
    ({sourceCommit, setupDigest, inputsDigest, effectsDigest, readinessDigest}) => {
      window.drawerFixture.setReply(async (call) => {
        if (call.method === 'POST' && call.url.endsWith('/preparation/acceptances'))
          return Response.json({head: 'requirement-current'});
        return call.url.endsWith('/api/repositories/7/factory')
          ? Response.json({repository: '7', appliance: {revision: 1, max_concurrent_runs: 1, max_queued: 0}})
          : call.url.endsWith('/api/environments/p0123456789abcdef01234567/preparation')
            ? Response.json({
                project: 'test-project',
                hold: true,
                requirements: 'requirement-current',
                approval: 'approval-current',
                approval_requirement: 'requirement-current',
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
                    roles: [
                      {
                        role: 'coder',
                        source_commit: sourceCommit,
                        setup_digest: setupDigest,
                        tools: 'node',
                        setup_exit: 0,
                        check_exit: 0,
                      },
                    ],
                  },
                },
              })
            : null;
      });
    },
    {sourceCommit, setupDigest, inputsDigest, effectsDigest, readinessDigest}
  );
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
  await settings
    .getByRole('status')
    .filter({hasText: 'separately verify current live services and resources'})
    .waitFor();
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

test('initial source, exact-input acceptance, operational approval and role preparation use separate native actions', async (t) => {
  const page = await fixture(t);
  const sourceCommit = 'a'.repeat(40);
  const setupDigest = 'b'.repeat(64);
  const inputsDigest = 'c'.repeat(64);
  const inputRows = [
    {path: 'setup.sh', digest: 'e'.repeat(64)},
    {path: 'check.sh', digest: 'd'.repeat(64)},
    {path: 'config/dependencies.lock', digest: 'f'.repeat(64)},
  ];
  await page.evaluate(
    ({sourceCommit, setupDigest, inputsDigest, inputRows}) => {
      let phase = 0;
      let requirementHead = '';
      let approvalHead = '';
      window.drawerFixture.setReply(async (call) => {
        if (call.url.endsWith('/api/repositories/7/factory'))
          return Response.json({repository: '7', appliance: {revision: 1, max_concurrent_runs: 1, max_queued: 0}});
        if (call.url.endsWith('/api/environments/p0123456789abcdef01234567/preparation'))
          return Response.json({
            project: 'test-project',
            requirements: requirementHead,
            approval: approvalHead,
            approval_requirement: phase >= 2 ? requirementHead : '',
            preparations: [],
            decision_preview: {reason: 'Select repository source to begin.'},
          });
        if (call.method !== 'POST') return null;
        const body = JSON.parse(call.body || '{}') as Record<string, unknown>;
        if (call.url.endsWith('/preparation/acceptances')) {
          phase = 1;
          requirementHead = String(body.decision_id);
          return Response.json({id: body.decision_id, head: body.decision_id});
        }
        if (!call.url.endsWith('/preparation/actions')) return null;
        if (body.action === 'source-preview') {
          return Response.json({
            requirements: {
              source_commit: sourceCommit,
              setup_digest: setupDigest,
              inputs_digest: inputsDigest,
              source_inputs: inputRows,
              predecessor: '',
            },
            files: [
              {path: 'setup.sh', content: '#!/bin/sh\nprintf setup\n'},
              {path: 'check.sh', content: '#!/bin/sh\nprintf check\n'},
              {path: 'config/dependencies.lock', content: 'node 24\n'},
            ],
            required_tools: ['node'],
            setup_files: ['setup.sh', 'check.sh'],
          });
        }
        if (body.action === 'approve-initial') {
          phase = 2;
          approvalHead = String(body.decision_id);
          return Response.json({id: body.decision_id, head: body.decision_id});
        }
        if (body.action === 'prepare') {
          phase = 3;
          return Response.json({accepted: true});
        }
        return null;
      });
    },
    {sourceCommit, setupDigest, inputsDigest, inputRows}
  );
  await page.evaluate(async () => window.drawerFixture.api.setPresentation('settings'));
  await refresh(page);
  await page.getByRole('tab', {name: 'Factory'}).click();
  const settings = page.locator('soda-factory-settings');
  await settings.evaluate(async (el) => {
    await (el as HTMLElement & {updateComplete: Promise<unknown>}).updateComplete;
  });

  await settings.locator('[name="source_paths"]').fill('setup.sh\ncheck.sh\nconfig/dependencies.lock');
  await settings.locator('[name="setup_files"]').fill('setup.sh\ncheck.sh');
  await settings.locator('[name="required_tools"]').fill('node');
  await settings.getByRole('button', {name: 'Preview selected source'}).click();
  await settings.getByText(sourceCommit).first().waitFor();
  await settings.locator('details').first().locator('summary').click();
  assert.match(await settings.locator('details').first().innerText(), /printf setup/);
  let calls = await writes(page);
  const sourcePreview = calls.find((call) => JSON.parse(call.body || '{}').action === 'source-preview');
  assert(sourcePreview);
  const selected = JSON.parse(sourcePreview.body || '{}') as Record<string, unknown>;
  assert.equal(selected.source_commit, '');
  assert.deepEqual(selected.source_paths, ['setup.sh', 'check.sh', 'config/dependencies.lock']);
  assert.deepEqual(selected.setup_files, ['setup.sh', 'check.sh']);
  assert.deepEqual(selected.required_tools, ['node']);
  await settings.locator('[name="source_commit"]').fill('b'.repeat(40));
  await settings.getByRole('button', {name: 'Record exact-input acceptance'}).click();
  await settings.getByRole('status').filter({hasText: 'Source selection changed after preview.'}).waitFor();
  assert.equal((await writes(page)).filter((call) => call.url.endsWith('/preparation/acceptances')).length, 0);
  await settings.locator('[name="source_commit"]').fill('');
  await settings.getByRole('button', {name: 'Record exact-input acceptance'}).click();
  await settings.getByRole('status').filter({hasText: 'Change confirmed by the current API.'}).waitFor();
  calls = await writes(page);
  const acceptance = calls.find((call) => call.url.endsWith('/preparation/acceptances'));
  assert(acceptance);
  const accepted = JSON.parse(acceptance.body || '{}') as Record<string, unknown>;
  assert.equal(accepted.source_commit, sourceCommit);
  assert.equal(accepted.setup_digest, setupDigest);
  assert.equal(accepted.inputs_digest, inputsDigest);
  assert.deepEqual(accepted.source_inputs, {
    'check.sh': 'd'.repeat(64),
    'setup.sh': 'e'.repeat(64),
    'config/dependencies.lock': 'f'.repeat(64),
  });
  assert.equal(accepted.predecessor, undefined);

  await settings.locator('[name="effects_evidence"]').fill('Reviewed exact setup and check commands.');
  await settings
    .locator('[name="readiness_evidence"]')
    .fill('Confirmed the installed service and selected Project resources are ready.');
  await settings.getByRole('button', {name: 'Record initial operational approval'}).click();
  await settings.getByRole('status').filter({hasText: 'Review the installed service/resource facts'}).waitFor();
  assert.equal((await writes(page)).filter((call) => call.url.endsWith('/preparation/actions')).length, 1);
  await settings.locator('[name="verified"]').check();
  await settings.getByRole('button', {name: 'Record initial operational approval'}).click();
  await settings.getByRole('status').filter({hasText: 'Change confirmed by the current API.'}).waitFor();
  calls = await writes(page);
  const initialApproval = calls.find((call) => JSON.parse(call.body || '{}').action === 'approve-initial');
  assert(initialApproval);
  const approval = JSON.parse(initialApproval.body || '{}') as Record<string, unknown>;
  assert.equal(approval.requirement, accepted.decision_id);
  assert.equal(approval.verified, true);
  assert.equal(approval.effects_evidence, 'Reviewed exact setup and check commands.');
  assert.equal(
    approval.readiness_evidence,
    'Confirmed the installed service and selected Project resources are ready.'
  );

  await settings.getByRole('button', {name: 'Start role preparation'}).click();
  await settings.getByRole('status').filter({hasText: 'Change confirmed by the current API.'}).waitFor();
  calls = await writes(page);
  const prepare = calls.find((call) => JSON.parse(call.body || '{}').action === 'prepare');
  assert(prepare);
  const prepared = JSON.parse(prepare.body || '{}') as Record<string, unknown>;
  assert.match(String(prepared.coder_id), /^f[a-f0-9]{24}$/);
  assert.match(String(prepared.reviewer_id), /^f[a-f0-9]{24}$/);
  assert.notEqual(prepared.coder_id, prepared.reviewer_id);
  assert.equal(prepared.requirement, accepted.decision_id);
  assert.equal(prepared.approval, approval.decision_id);
  assert.deepEqual(prepared.source_paths, ['setup.sh', 'check.sh', 'config/dependencies.lock']);
  assert.deepEqual(prepared.setup_files, ['setup.sh', 'check.sh']);
  assert.deepEqual(prepared.required_tools, ['node']);
});

test('a preparation retry after an unconfirmed response reuses both admitted identities', async (t) => {
  const page = await fixture(t);
  const sourceCommit = 'a'.repeat(40);
  const setupDigest = 'b'.repeat(64);
  const inputsDigest = 'c'.repeat(64);
  const sourceInputs = [
    {path: 'setup.sh', digest: 'd'.repeat(64)},
    {path: 'check.sh', digest: 'e'.repeat(64)},
  ];
  await page.evaluate(
    ({sourceCommit, setupDigest, inputsDigest, sourceInputs}) => {
      let phase = 0;
      let requirementHead = '';
      let prepareCalls = 0;
      window.drawerFixture.setReply(async (call) => {
        if (call.url.endsWith('/api/repositories/7/factory'))
          return Response.json({repository: '7', appliance: {revision: 1, max_concurrent_runs: 1, max_queued: 0}});
        if (call.url.endsWith('/api/environments/p0123456789abcdef01234567/preparation'))
          return Response.json({
            project: 'test-project',
            requirements: requirementHead,
            approval: phase >= 2 ? 'approval-current' : '',
            approval_requirement: phase >= 2 ? requirementHead : '',
            preparations: [],
            decision_preview: {reason: 'Select repository source to begin.'},
          });
        if (call.method !== 'POST') return null;
        const body = JSON.parse(call.body || '{}') as Record<string, unknown>;
        if (call.url.endsWith('/preparation/acceptances')) {
          phase = 1;
          requirementHead = String(body.decision_id);
          return Response.json({id: body.decision_id, head: body.decision_id});
        }
        if (!call.url.endsWith('/preparation/actions')) return null;
        if (body.action === 'source-preview')
          return Response.json({
            requirements: {
              source_commit: sourceCommit,
              setup_digest: setupDigest,
              inputs_digest: inputsDigest,
              source_inputs: sourceInputs,
              predecessor: '',
            },
            files: [
              {path: 'setup.sh', content: '#!/bin/sh\nexit 0\n'},
              {path: 'check.sh', content: '#!/bin/sh\nexit 0\n'},
            ],
            required_tools: [],
            setup_files: ['setup.sh', 'check.sh'],
          });
        if (body.action === 'approve-initial') {
          phase = 2;
          return Response.json({id: body.decision_id});
        }
        if (body.action === 'prepare') {
          prepareCalls++;
          return prepareCalls === 1
            ? Response.json({error: {code: 'preparation_unconfirmed'}}, {status: 503})
            : Response.json({preparations: []});
        }
        return null;
      });
    },
    {sourceCommit, setupDigest, inputsDigest, sourceInputs}
  );
  await page.evaluate(async () => window.drawerFixture.api.setPresentation('settings'));
  await refresh(page);
  await page.getByRole('tab', {name: 'Factory'}).click();
  const settings = page.locator('soda-factory-settings');
  await settings.evaluate(async (el) => {
    await (el as HTMLElement & {updateComplete: Promise<unknown>}).updateComplete;
  });
  await settings.getByRole('button', {name: 'Preview selected source'}).click();
  await page.waitForTimeout(250);
  assert.match(await settings.getByRole('status').innerText(), /Source preview returned/);
  await settings.getByRole('button', {name: 'Record exact-input acceptance'}).click();
  await settings.getByRole('status').filter({hasText: 'Change confirmed'}).waitFor();
  await settings.locator('[name="effects_evidence"]').fill('Reviewed installed effects.');
  await settings.locator('[name="readiness_evidence"]').fill('Checked current services.');
  await settings.locator('[name="verified"]').check();
  await settings.getByRole('button', {name: 'Record initial operational approval'}).click();
  await settings.getByRole('status').filter({hasText: 'Change confirmed'}).waitFor();

  await settings.getByRole('button', {name: 'Start role preparation'}).click();
  await settings.getByRole('status').filter({hasText: 'not confirmed (503)'}).waitFor();
  await settings.getByRole('button', {name: 'Start role preparation'}).click();
  await settings.getByRole('status').filter({hasText: 'Change confirmed'}).waitFor();
  const preparations = (await writes(page))
    .filter((call) => call.url.endsWith('/preparation/actions'))
    .map((call) => JSON.parse(call.body || '{}') as Record<string, unknown>)
    .filter((body) => body.action === 'prepare');
  assert.equal(preparations.length, 2);
  const first = preparations[0];
  const second = preparations[1];
  assert(first);
  assert(second);
  assert.match(String(first.coder_id), /^f[a-f0-9]{24}$/);
  assert.match(String(first.reviewer_id), /^f[a-f0-9]{24}$/);
  assert.equal(second.coder_id, first.coder_id);
  assert.equal(second.reviewer_id, first.reviewer_id);
});

test('reload adopts an exact nonterminal role identity and requires review before replacing a retired one', async (t) => {
  for (const phase of ['running', 'failed']) {
    await t.test(phase, async (t) => {
      const page = await fixture(t);
      const req = 'd111111111111111111111111';
      const approval = 'd222222222222222222222222';
      const existingId = 'f111111111111111111111111';
      const sourceCommit = 'a'.repeat(40);
      const setupDigest = 'b'.repeat(64);
      await page.evaluate(
        ({req, approval, existingId, phase, sourceCommit, setupDigest}) => {
          let prepareCalls = 0;
          window.drawerFixture.setReply(async (call) => {
            if (call.url.endsWith('/api/repositories/7/factory'))
              return Response.json({repository: '7', appliance: {revision: 1, max_concurrent_runs: 1, max_queued: 0}});
            if (call.url.endsWith('/api/environments/p0123456789abcdef01234567/preparation'))
              return Response.json({
                project: 'test-project',
                requirements: req,
                approval,
                approval_requirement: req,
                preparations: [
                  {
                    id: existingId,
                    role: 'soda-coder',
                    phase,
                    ready: false,
                    stopped: phase === 'failed',
                    source_commit: sourceCommit,
                    setup_digest: setupDigest,
                    requirements: req,
                    approval,
                  },
                ],
              });
            if (call.method !== 'POST' || !call.url.endsWith('/preparation/actions')) return null;
            const body = JSON.parse(call.body || '{}') as Record<string, unknown>;
            if (body.action === 'source-preview')
              return Response.json({
                requirements: {
                  source_commit: sourceCommit,
                  setup_digest: setupDigest,
                  inputs_digest: 'c'.repeat(64),
                  source_inputs: [
                    {path: 'setup.sh', digest: 'd'.repeat(64)},
                    {path: 'check.sh', digest: 'e'.repeat(64)},
                  ],
                  predecessor: req,
                },
                files: [
                  {path: 'setup.sh', content: '#!/bin/sh\nexit 0\n'},
                  {path: 'check.sh', content: '#!/bin/sh\nexit 0\n'},
                ],
                setup_files: ['setup.sh', 'check.sh'],
                required_tools: [],
              });
            if (body.action === 'prepare') {
              prepareCalls++;
              if (phase === 'failed' && prepareCalls === 1)
                return Response.json({error: {code: 'preparation_unconfirmed'}}, {status: 503});
              return Response.json({preparations: []});
            }
            return null;
          });
        },
        {req, approval, existingId, phase, sourceCommit, setupDigest}
      );
      await page.evaluate(async () => window.drawerFixture.api.setPresentation('settings'));
      await refresh(page);
      await page.getByRole('tab', {name: 'Factory'}).click();
      const settings = page.locator('soda-factory-settings');
      await settings.evaluate(async (el) => {
        await (el as HTMLElement & {updateComplete: Promise<unknown>}).updateComplete;
      });
      await settings.getByRole('button', {name: 'Preview selected source'}).click();
      await settings.getByRole('status').filter({hasText: 'Source preview returned'}).waitFor();
      if (phase === 'failed') {
        await settings.getByRole('button', {name: 'Start role preparation'}).click();
        await settings.getByRole('status').filter({hasText: 'Inspect the failed or stopped'}).waitFor();
        assert.equal(
          (await writes(page)).filter((call) => JSON.parse(call.body || '{}').action === 'prepare').length,
          0
        );
        await settings.locator('[name="replace_terminal_preparations"]').check();
      }
      await settings.getByRole('button', {name: 'Start role preparation'}).click();
      if (phase === 'failed') {
        await settings.getByRole('status').filter({hasText: 'not confirmed (503)'}).waitFor();
        await settings.getByRole('button', {name: 'Start role preparation'}).click();
      }
      await settings.getByRole('status').filter({hasText: 'Change confirmed'}).waitFor();
      const start = (await writes(page))
        .filter((call) => call.url.endsWith('/preparation/actions'))
        .map((call) => JSON.parse(call.body || '{}') as Record<string, unknown>)
        .find((body) => body.action === 'prepare');
      assert(start);
      assert.equal(start.coder_id === existingId, phase === 'running');
      assert.match(String(start.reviewer_id), /^f[a-f0-9]{24}$/);
      if (phase === 'failed') {
        const starts = (await writes(page))
          .filter((call) => call.url.endsWith('/preparation/actions'))
          .map((call) => JSON.parse(call.body || '{}') as Record<string, unknown>)
          .filter((body) => body.action === 'prepare');
        assert.equal(starts.length, 2);
        assert.equal(starts[1]?.coder_id, starts[0]?.coder_id);
        assert.equal(starts[1]?.reviewer_id, starts[0]?.reviewer_id);
      }
    });
  }
});

test('reaccepting requirements leaves the prior approval stale and blocks preparation', async (t) => {
  const page = await fixture(t);
  const sourceCommit = 'a'.repeat(40);
  const setupDigest = 'b'.repeat(64);
  const inputsDigest = 'c'.repeat(64);
  await page.evaluate(
    ({sourceCommit, setupDigest, inputsDigest}) => {
      let requirement = 'requirement-old';
      window.drawerFixture.setReply(async (call) => {
        if (call.url.endsWith('/api/repositories/7/factory'))
          return Response.json({repository: '7', appliance: {revision: 1, max_concurrent_runs: 1, max_queued: 0}});
        if (call.url.endsWith('/api/environments/p0123456789abcdef01234567/preparation'))
          return Response.json({
            project: 'test-project',
            requirements: requirement,
            approval: 'approval-old',
            approval_requirement: 'requirement-old',
            preparations: [],
            decision_preview: {reason: 'Select repository source to begin.'},
          });
        if (call.method !== 'POST') return null;
        const body = JSON.parse(call.body || '{}') as Record<string, unknown>;
        if (call.url.endsWith('/preparation/acceptances')) {
          requirement = String(body.decision_id);
          return Response.json({id: body.decision_id, head: body.decision_id});
        }
        if (!call.url.endsWith('/preparation/actions') || body.action !== 'source-preview') return null;
        return Response.json({
          requirements: {
            source_commit: sourceCommit,
            setup_digest: setupDigest,
            inputs_digest: inputsDigest,
            source_inputs: [
              {path: 'setup.sh', digest: 'd'.repeat(64)},
              {path: 'check.sh', digest: 'e'.repeat(64)},
            ],
            predecessor: 'requirement-old',
          },
          files: [
            {path: 'setup.sh', content: '#!/bin/sh\nexit 0\n'},
            {path: 'check.sh', content: '#!/bin/sh\nexit 0\n'},
          ],
          required_tools: [],
          setup_files: ['setup.sh', 'check.sh'],
        });
      });
    },
    {sourceCommit, setupDigest, inputsDigest}
  );
  await page.evaluate(async () => window.drawerFixture.api.setPresentation('settings'));
  await refresh(page);
  await page.getByRole('tab', {name: 'Factory'}).click();
  const settings = page.locator('soda-factory-settings');
  await settings.evaluate(async (el) => {
    await (el as HTMLElement & {updateComplete: Promise<unknown>}).updateComplete;
  });
  await settings.getByRole('button', {name: 'Preview selected source'}).click();
  await settings.getByRole('status').filter({hasText: 'Source preview returned'}).waitFor();
  await settings.getByRole('button', {name: 'Record exact-input acceptance'}).click();
  await settings.getByRole('status').filter({hasText: 'Change confirmed'}).waitFor();
  await settings.getByRole('heading', {name: 'Approve initial Project operations'}).waitFor();
  assert.equal(await settings.getByRole('button', {name: 'Start role preparation'}).count(), 0);
  assert.equal((await writes(page)).filter((call) => JSON.parse(call.body || '{}').action === 'prepare').length, 0);
});

test('provider budget and sponsorship use observed connection and grant records and surface stale authority once', async (t) => {
  const page = await fixture(t);
  await page.evaluate(() =>
    window.drawerFixture.setReply(async (call) => {
      if (call.url.endsWith('/api/repositories/7/factory'))
        return Response.json({
          repository: '7',
          policy: null,
          operator_grant: null,
          appliance: {revision: 4, max_concurrent_runs: 2, max_queued: 8},
          sponsorships: [],
          environment_grant: null,
          preparation: null,
          queue: {queued: 0, active: 0},
          effective: {effective: false, dispatch_open: false, missing: []},
        });
      if (call.url.endsWith('/api/identity/connections'))
        return Response.json([
          {
            id: 'connection-observed',
            provider_id: 'codex',
            owner_id: '9',
            label: 'Primary',
            generation: 7,
            state: 'active',
          },
        ]);
      if (call.url.endsWith('/api/identity/connections/connection-observed/grants'))
        return Response.json([
          {
            id: 'grant-observed',
            connection_id: 'connection-observed',
            project_id: 'p123456789abcdef012345678',
            revision: 2,
            revoked: false,
          },
        ]);
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
        return Response.json({
          repository: '7',
          appliance: {revision: 1, max_concurrent_runs: 1, max_queued: 0},
          sponsorships: [],
          effective: {effective: false, dispatch_open: false, missing: []},
        });
      if (call.url.endsWith('/api/identity/connections'))
        return Response.json([
          {id: 'connection-observed', owner_id: '9', label: 'Primary', generation: 7, state: 'active'},
        ]);
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
  await settings.evaluate((element) =>
    element.addEventListener('soda-session-invalid', () => {
      (element as HTMLElement).dataset.invalid = 'yes';
    })
  );
  await settings.getByRole('button', {name: 'Save connection budget'}).click();
  await settings.getByRole('status').filter({hasText: 'not confirmed (403)'}).waitFor();
  sessionInvalid = (await settings.getAttribute('data-invalid')) === 'yes' ? 1 : 0;
  assert.equal(sessionInvalid, 1);
  const budgetWrites = (await writes(page)).filter((call) =>
    call.url.endsWith('/api/factory/connections/connection-observed/usage-budget')
  );
  assert.equal(budgetWrites.length, 1);
});

test('late source-preview and acceptance replies cannot write into a newer Project context', async (t) => {
  const page = await fixture(t);
  const sourceCommit = 'a'.repeat(40);
  const setupDigest = 'b'.repeat(64);
  const inputsDigest = 'c'.repeat(64);
  await page.evaluate(
    ({sourceCommit, setupDigest, inputsDigest}) => {
      let resolvePreview: ((response: Response) => void) | undefined;
      let resolveAcceptance: ((response: Response) => void) | undefined;
      window.drawerFixture.setReply(async (call) => {
        if (call.url.endsWith('/api/repositories/8/factory') || call.url.endsWith('/api/repositories/9/factory'))
          return Response.json({repository: call.url.includes('/8/') ? '8' : '9', appliance: {revision: 1}});
        if (call.url.endsWith('/api/environments/p222222222222222222222222/preparation'))
          return Response.json({
            project: 'newest-project',
            requirements: 'newest-requirement',
            approval: '',
            preparations: [],
            decision_preview: {reason: 'Newest Project source remains unreviewed.'},
          });
        if (call.url.endsWith('/api/environments/p111111111111111111111111/preparation'))
          return Response.json({
            project: 'middle-project',
            requirements: '',
            approval: '',
            preparations: [],
            decision_preview: {reason: 'Select source for the middle Project.'},
          });
        if (call.method !== 'POST') return null;
        const body = JSON.parse(call.body || '{}') as Record<string, unknown>;
        if (body.action === 'source-preview' && call.url.includes('/p0123456789abcdef01234567/'))
          return new Promise<Response>((resolve) => {
            resolvePreview = resolve;
          });
        if (call.url.endsWith('/preparation/acceptances'))
          return new Promise<Response>((resolve) => {
            resolveAcceptance = resolve;
          });
        if (body.action === 'source-preview')
          return Response.json({
            requirements: {
              source_commit: sourceCommit,
              setup_digest: setupDigest,
              inputs_digest: inputsDigest,
              source_inputs: [
                {path: 'setup.sh', digest: 'd'.repeat(64)},
                {path: 'check.sh', digest: 'e'.repeat(64)},
              ],
              predecessor: 'middle-predecessor',
            },
            files: [
              {path: 'setup.sh', content: 'MIDDLE PROJECT PREVIEW'},
              {path: 'check.sh', content: '#!/bin/sh\nexit 0\n'},
            ],
            required_tools: [],
            setup_files: ['setup.sh', 'check.sh'],
          });
        return null;
      });
      Object.assign(window, {
        resolveOldPreview: () =>
          resolvePreview?.(
            Response.json({
              requirements: {
                source_commit: sourceCommit,
                setup_digest: setupDigest,
                inputs_digest: inputsDigest,
                source_inputs: [
                  {path: 'setup.sh', digest: 'd'.repeat(64)},
                  {path: 'check.sh', digest: 'e'.repeat(64)},
                ],
                predecessor: 'old-project-predecessor',
              },
              files: [
                {path: 'setup.sh', content: 'OLD PROJECT PREVIEW MUST NOT APPEAR'},
                {path: 'check.sh', content: '#!/bin/sh\nexit 0\n'},
              ],
              required_tools: [],
              setup_files: ['setup.sh', 'check.sh'],
            })
          ),
        resolveOldAcceptance: () => resolveAcceptance?.(Response.json({head: 'old-project-accepted-head'})),
      });
    },
    {sourceCommit, setupDigest, inputsDigest}
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

  await settings.getByRole('button', {name: 'Preview selected source'}).click();
  await page.waitForFunction(() => window.drawerFixture.calls.some((call) => call.body?.includes('source-preview')));
  await settings.evaluate((el) => {
    const component = el as HTMLElement & {
      contextValue: {transport: unknown; repository: string; environment: string; profile?: unknown};
      context: {transport: unknown; repository: string; environment: string; profile?: unknown};
    };
    component.context = {...component.contextValue, repository: '8', environment: 'p111111111111111111111111'};
  });
  await settings.getByText('Select source for the middle Project.').first().waitFor();
  await page.evaluate(() => (Reflect.get(window, 'resolveOldPreview') as () => void)());
  await settings.getByText('Select source for the middle Project.').first().waitFor();
  assert.doesNotMatch(await settings.innerText(), /OLD PROJECT PREVIEW MUST NOT APPEAR|old-project-predecessor/);

  await settings.getByRole('button', {name: 'Preview selected source'}).click();
  await settings.locator('summary').filter({hasText: 'setup.sh'}).click();
  await settings.getByText('MIDDLE PROJECT PREVIEW').waitFor();
  await settings.getByRole('button', {name: 'Record exact-input acceptance'}).click();
  await page.waitForFunction(() =>
    window.drawerFixture.calls.some((call) => call.url.endsWith('/preparation/acceptances'))
  );
  await settings.evaluate((el) => {
    const component = el as HTMLElement & {
      contextValue: {transport: unknown; repository: string; environment: string; profile?: unknown};
      context: {transport: unknown; repository: string; environment: string; profile?: unknown};
    };
    component.context = {...component.contextValue, repository: '9', environment: 'p222222222222222222222222'};
  });
  await settings.getByText('Newest Project source remains unreviewed.').first().waitFor();
  await page.evaluate(() => (Reflect.get(window, 'resolveOldAcceptance') as () => void)());
  await settings.getByText('Newest Project source remains unreviewed.').first().waitFor();
  assert.doesNotMatch(await settings.innerText(), /MIDDLE PROJECT PREVIEW|old-project-accepted-head/);
  const retained = await settings.evaluate((el) => {
    const view = Reflect.get(el, 'view') as Record<string, unknown>;
    return {sourcePreview: view.sourcePreview, sourceSelection: view.sourceSelection};
  });
  assert.equal(retained.sourcePreview, undefined);
  assert.equal(retained.sourceSelection, undefined);
});
