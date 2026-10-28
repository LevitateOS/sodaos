import test from 'node:test';
import assert from 'node:assert/strict';
import {setupProjectControlsDriver, fixture, refresh, writes} from './fixtures/project-controls-driver';
setupProjectControlsDriver();

test('Factory activity preserves exact native links and saves only local display preferences', async (t) => {
  const page = await fixture(t);
  await page.evaluate(() => {
    window.drawerFixture.setReply(async (call) =>
      call.url.endsWith('/api/repositories/7/factory/activity')
        ? Response.json({
            repository: '7',
            repository_path: '/fixture/repository',
            revision: 3,
            observed_unix: 4,
            settings: {issue_created: true, pr_created: true, verified_merge: true},
            items: [
              {
                id: 'issue',
                kind: 'issue_created',
                issue: '11',
                created_unix: 1,
                summary: 'Original accepted issue created',
                url: '/fixture/repository/issues/11',
              },
              {
                id: 'pr',
                kind: 'pr_created',
                issue: '11',
                pr: '12',
                created_unix: 2,
                summary: 'Pull request awaiting review',
                url: '/fixture/repository/pulls/12',
              },
              {
                id: 'merge',
                kind: 'verified_merge',
                issue: '11',
                pr: '12',
                created_unix: 3,
                summary: 'Verified merge and original issue closed',
                url: '/fixture/repository/issues/11',
              },
            ],
          })
        : null
    );
    window.drawerFixture.api.setPresentation('settings');
  });
  await refresh(page);
  await page.getByRole('tab', {name: 'Factory'}).click();
  const activity = page.locator('soda-factory-activity');
  await activity.getByRole('button', {name: 'Refresh activity', exact: true}).click();
  await activity.getByRole('link', {name: 'Original accepted issue created'}).waitFor();
  assert.equal(
    await activity.getByRole('link', {name: 'Pull request awaiting review'}).getAttribute('href'),
    '/fixture/repository/pulls/12'
  );
  await activity.locator('input[name="pr_created"]').uncheck();
  await activity.getByRole('button', {name: 'Save activity display settings'}).click();
  await activity.locator('[data-activity-message]').filter({hasText: 'Activity display settings saved.'}).waitFor();
  const calls = (await writes(page)).filter((call) => call.url.endsWith('/factory/activity'));
  assert.equal(calls.length, 1);
  assert.deepEqual(JSON.parse(calls[0]?.body || '{}'), {
    expected_revision: 3,
    settings: {issue_created: true, pr_created: false, verified_merge: true},
  });
  assert.equal((await writes(page)).filter((call) => !call.url.endsWith('/factory/activity')).length, 0);
});

test('Factory activity rejects foreign repository links instead of displaying them', async (t) => {
  const page = await fixture(t);
  await page.evaluate(() => {
    window.drawerFixture.setReply(async (call) =>
      call.url.endsWith('/factory/activity')
        ? Response.json({
            repository: '7',
            repository_path: '/fixture/repository',
            revision: 0,
            observed_unix: 2,
            settings: {issue_created: true, pr_created: true, verified_merge: true},
            items: [
              {
                id: 'bad',
                kind: 'issue_created',
                issue: '11',
                created_unix: 1,
                summary: 'Foreign issue',
                url: 'https://other.example/issues/11',
              },
            ],
          })
        : null
    );
    window.drawerFixture.api.setPresentation('settings');
  });
  await refresh(page);
  await page.getByRole('tab', {name: 'Factory'}).click();
  const activity = page.locator('soda-factory-activity');
  await activity.getByRole('button', {name: 'Refresh activity', exact: true}).click();
  await activity.locator('[data-activity-message]').filter({hasText: 'Invalid Factory activity item.'}).waitFor();
  assert.equal(await activity.getByRole('link').count(), 0);
});

test('Factory settings render current authority and submit capacity through the native session mount', async (t) => {
  const page = await fixture(t);
  await page.evaluate(async () => {
    let revision = 0;
    let grant: Record<string, unknown> | null = null;
    let loseReply = true;
    window.drawerFixture.setReply(async (call) => {
      if (!call.url.endsWith('/api/repositories/7/factory/repair')) return null;
      if (call.method === 'GET')
        return Response.json({
          revision,
          grant,
          charges: revision
            ? [{id: 'charge', kind: 'run', episode_id: 'episode', cycle: 1, active_minutes: 30, failed: true}]
            : [],
          episodes: revision
            ? [
                {
                  episode_id: 'episode',
                  reason: 'missing_producer',
                  cycles: [{id: 'cycle', number: 1, state: 'attention', reason: 'no_change', run_id: 'run'}],
                },
              ]
            : [],
        });
      const body = JSON.parse(call.body || '{}');
      revision = body.expected_revision + 1;
      grant = {...body, id: body.command_id, repository: '7'};
      if (loseReply) {
        loseReply = false;
        throw Error('Fixture lost committed reply');
      }
      return Response.json({command_id: body.command_id, revision});
    });
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

  const repair = page.locator('soda-factory-repair');
  await repair.getByText('No repair grant has been authorized.', {exact: false}).waitFor();
  assert.equal((await writes(page)).filter((call) => call.url.endsWith('/factory/repair')).length, 0);
  assert.equal(await repair.getByLabel('Authorize repository repair spending').isChecked(), false);
  await repair.getByLabel('Investigator run limit').fill('1');
  await repair.getByLabel('Proposal judgment limit').fill('1');
  await repair.getByLabel('Active minute limit').fill('30');
  await repair.getByLabel('Authorize repository repair spending').check();
  await repair.getByRole('button', {name: 'Save repair grant', exact: true}).click();
  await repair.locator('[data-repair-message]').filter({hasText: 'A repair change is unconfirmed.'}).waitFor();
  assert.equal(await repair.getByLabel('Investigator run limit').isDisabled(), true);
  await repair.getByRole('button', {name: 'Retry same repair change', exact: true}).click();
  await repair.locator('[data-repair-message]').filter({hasText: 'Repair grant confirmed.'}).waitFor();
  let repairWrites = (await writes(page)).filter((call) => call.url.endsWith('/factory/repair'));
  assert.equal(repairWrites.length, 2);
  assert.deepEqual(JSON.parse(repairWrites[0]!.body!), JSON.parse(repairWrites[1]!.body!));
  assert.equal(JSON.parse(repairWrites[0]!.body!).expected_revision, 0);
  assert.match(await repair.innerText(), /1 investigator runs, 0 proposal judgments, 30 active minutes reserved/);
  await repair.getByText('Repair episodes (1)', {exact: true}).click();
  await repair.getByText('Cycle 1: attention · no_change · Run run', {exact: true}).waitFor();
  await repair.getByLabel('Investigator run limit').fill('2');
  await repair.getByRole('button', {name: 'Save repair grant', exact: true}).click();
  await repair.locator('[data-repair-message]').filter({hasText: 'Describe the effect'}).waitFor();
  assert.equal((await writes(page)).filter((call) => call.url.endsWith('/factory/repair')).length, 2);
  await repair
    .getByLabel('Effect of this change')
    .fill('Authorize one additional investigation; preserve recorded consumption.');
  await repair.getByRole('button', {name: 'Save repair grant', exact: true}).click();
  await repair.locator('[data-repair-message]').filter({hasText: 'Repair grant confirmed.'}).waitFor();
  repairWrites = (await writes(page)).filter((call) => call.url.endsWith('/factory/repair'));
  assert.equal(repairWrites.length, 3);
  assert.equal(JSON.parse(repairWrites[2]!.body!).expected_revision, 1);
  assert.equal(JSON.parse(repairWrites[2]!.body!).investigator_runs, 2);
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

test('first repository policy selects enrolled actors and requires explicit intent funding', async (t) => {
  const page = await fixture(t);
  const bindings = [
    {actor_id: '41', token_id: '51', kind: 'git.ref.publish'},
    {actor_id: '45', token_id: '56', kind: 'git.ref.publish'},
    {actor_id: '41', token_id: '51', kind: 'pull_request.create'},
    {actor_id: '43', token_id: '53', kind: 'pull_request.review.submit'},
    {actor_id: '43', token_id: '55', kind: 'pull_request.review.thread.resolve'},
    {actor_id: '44', token_id: '54', kind: 'pull_request.merge'},
  ];
  await page.evaluate((bindings) => {
    window.drawerFixture.setReply(async (call) => {
      if (call.url.endsWith('/api/repositories/7/factory') && call.method === 'GET')
        return Response.json({repository: '7', policy: null, native_bindings: bindings});
      if (
        call.url.endsWith('/api/repositories/7/factory/policy') &&
        call.method === 'PUT' &&
        JSON.parse(call.body || '{}').intent_minutes > 0
      )
        return new Promise<Response>((resolve) =>
          window.addEventListener('release-policy-write', () => resolve(Response.json({ok: true})), {once: true})
        );
      return null;
    });
    window.drawerFixture.api.setPresentation('settings');
  }, bindings);
  await refresh(page);
  await page.getByRole('tab', {name: 'Factory'}).click();
  const form = page.locator('soda-factory-settings form[data-action="policy"]');
  const minutes = form.getByLabel('Intent active minutes', {exact: true});
  assert.equal(await minutes.inputValue(), '0', 'A new policy must not invent intent funding');
  await form.getByLabel('Publisher', {exact: true}).selectOption('45:56');
  for (const [name, value] of Object.entries({
    target_branch: 'refs/heads/main',
    required_checks: 'native-ci/check',
    coder_harness_version: 'fixture',
    coder_model: 'fixture-coder',
    reviewer_harness_version: 'fixture',
    reviewer_model: 'fixture-reviewer',
  })) {
    await form.locator(`input[name="${name}"]`).fill(value);
  }
  await form.getByRole('button', {name: 'Save repository policy', exact: true}).click();
  await page.getByRole('status').filter({hasText: 'Change confirmed by the current API.'}).waitFor();
  const sent = (await writes(page)).filter((call) => call.url.endsWith('/api/repositories/7/factory/policy'));
  assert.equal(sent.length, 1);
  const body = JSON.parse(sent[0]!.body!);
  assert.equal(body.expected_revision, 0);
  assert.equal(body.intent_minutes, 0);
  for (const [field, expected] of Object.entries({
    publish_actor: {actor_id: '45', token_id: '56'},
    create_actor: {actor_id: '41', token_id: '51'},
    review_actor: {actor_id: '43', token_id: '53'},
    resolve_actor: {actor_id: '43', token_id: '55'},
    merge_actor: {actor_id: '44', token_id: '54'},
  }))
    assert.deepEqual(body[field], expected);
  for (const invalid of ['-1', '1.5']) {
    await minutes.fill(invalid);
    await form.evaluate((element) => element.dispatchEvent(new Event('submit', {bubbles: true, cancelable: true})));
    await page.getByRole('status').filter({hasText: 'Enter a whole number in the allowed range.'}).waitFor();
    assert.equal(await minutes.inputValue(), invalid, 'A refused value remains available for correction');
    assert.equal((await writes(page)).filter((call) => call.url.endsWith('/factory/policy')).length, 1);
  }
  await minutes.fill('12');
  await form.getByRole('button', {name: 'Save repository policy', exact: true}).click();
  await page.waitForFunction(
    () =>
      window.drawerFixture.calls.filter((call) => call.url.endsWith('/factory/policy') && call.method === 'PUT')
        .length === 2
  );
  assert(await minutes.isDisabled(), 'The existing policy authority fieldset closes while a write is pending');
  assert(await form.getByRole('button', {name: 'Save repository policy', exact: true}).isDisabled());
  await page.evaluate(() => window.dispatchEvent(new Event('release-policy-write')));
  await page.getByRole('status').filter({hasText: 'Change confirmed by the current API.'}).waitFor();
  const funded = (await writes(page)).filter((call) => call.url.endsWith('/factory/policy'));
  assert.equal(funded.length, 2);
  assert.equal(JSON.parse(funded[1]!.body!).intent_minutes, 12);
});

test('repository policy preserves canonical role selections and submits only actor and token IDs', async (t) => {
  const page = await fixture(t);
  const roles = {
    'soda-coder': {harness: 'muse', harness_version: '1.4.0-R4161.1', model: 'fixture-coder'},
    'soda-reviewer': {harness: 'codex', harness_version: '0.101.0', model: 'fixture-reviewer'},
  };
  // The factory is Muse-only: saving rewrites a stored non-Muse harness.
  const submittedRoles = {
    'soda-coder': roles['soda-coder'],
    'soda-reviewer': {...roles['soda-reviewer'], harness: 'muse'},
  };
  const actors = {
    publish_actor: {actor_id: '41', token_id: '51'},
    create_actor: {actor_id: '42', token_id: '52'},
    review_actor: {actor_id: '43', token_id: '53'},
    resolve_actor: {actor_id: '43', token_id: '55'},
    merge_actor: {actor_id: '44', token_id: '54'},
  };
  await page.evaluate(
    ({roles, actors}) => {
      window.drawerFixture.setReply(async (call) =>
        call.url.endsWith('/api/repositories/7/factory') && call.method === 'GET'
          ? Response.json({
              repository: '7',
              appliance: {revision: 4, max_concurrent_runs: 2, max_queued: 8},
              policy: {
                revision: 6,
                enabled: true,
                paused: true,
                target_branch: 'refs/heads/main',
                required_checks: ['native-check', 'review-check'],
                roles,
                publish_actor: {...actors.publish_actor, kind: 'git.ref.publish'},
                create_actor: {...actors.create_actor, kind: 'pull_request.create'},
                review_actor: {...actors.review_actor, kind: 'pull_request.review.submit'},
                resolve_actor: {...actors.resolve_actor, kind: 'pull_request.review.thread.resolve'},
                merge_actor: {...actors.merge_actor, kind: 'pull_request.merge'},
                attempt_limits: {active_minutes: 10, correction_cycles: 1, delivery_calls: 2, review_rounds: 4},
                intent_minutes: 9,
                reassessment_minutes: 17,
                max_concurrent: 1,
              },
            })
          : null
      );
      window.drawerFixture.api.setPresentation('settings');
    },
    {roles, actors}
  );
  await refresh(page);
  await page.getByRole('tab', {name: 'Factory'}).click();
  const form = page.locator('soda-factory-settings form[data-action="policy"]');
  assert.equal(await form.getByLabel('Intent active minutes', {exact: true}).inputValue(), '9');
  const rendered: Record<string, string> = {};
  const selected = {
    coder_harness_version: roles['soda-coder'].harness_version,
    coder_model: roles['soda-coder'].model,
    reviewer_harness_version: roles['soda-reviewer'].harness_version,
    reviewer_model: roles['soda-reviewer'].model,
  };
  for (const [name, value] of Object.entries(selected)) {
    const input = form.locator(`input[name="${name}"]`);
    rendered[name] = await input.inputValue();
    await input.fill(value);
  }
  await form.getByRole('button', {name: 'Save repository policy', exact: true}).click();
  await page.getByRole('status').filter({hasText: 'Change confirmed by the current API.'}).waitFor();
  const sent = (await writes(page)).filter((call) => call.url.endsWith('/api/repositories/7/factory/policy'));
  assert.equal(sent.length, 1);
  const call = sent[0];
  assert(call);
  assert.equal(call.method, 'PUT');
  assert.equal(call.headers['x-extension-session-generation'], 'fixture-generation');
  const payload = JSON.parse(call.body || '{}') as Record<string, unknown>;
  assert.match(String(payload.command_id), /^[a-f0-9]{32}$/);
  assert.deepEqual(
    {rendered, submitted: {...payload, command_id: '<generated>'}},
    {
      rendered: selected,
      submitted: {
        command_id: '<generated>',
        expected_revision: 6,
        enabled: true,
        paused: true,
        target_branch: 'refs/heads/main',
        merge_method: 'fast-forward-only',
        required_checks: ['native-check', 'review-check'],
        roles: submittedRoles,
        ...actors,
        attempt_limits: {active_minutes: 10, correction_cycles: 1, delivery_calls: 2, review_rounds: 4},
        intent_minutes: 9,
        reassessment_minutes: 17,
        max_concurrent: 1,
      },
    }
  );
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
  assert.match(
    await settings.getByRole('status').filter({hasText: 'Source preview'}).innerText(),
    /Source preview returned/
  );
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
  assert.deepEqual(payload.roles, ['soda-coder']);
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
  assert.equal(sessionInvalid, 0, 'an ordinary permission refusal must not retire the signed-in session');
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
test('Factory execution usage reports exact minutes, editability and required actions', async (t) => {
  const page = await fixture(t);
  await page.evaluate(() => {
    window.drawerFixture.setReply(async (call) =>
      call.url.endsWith('/api/repositories/7/factory/access')
        ? Response.json({
            viewer: {
              code_write: true,
              owner_or_admin: false,
              environment_owner: false,
              operator: false,
              member: true,
              session_current: true,
            },
            editable: {
              policy: {editable: false, reason: 'owner_or_admin_required'},
              activity_settings: {editable: true},
              operator_grant: {editable: false, reason: 'operator_required'},
              capacity: {editable: true},
              environment_grant: {editable: false, reason: 'repository_owner_required'},
              sponsorships: [{connection: 'c1', editable: true}],
              usage_budgets: [{connection: 'c1', editable: false, reason: 'connection_owner_required'}],
            },
            effective: {missing: [], effective: true, dispatch_open: true},
            required_actions: [
              {code: 'usage_budget_missing', resolve_by: 'connection_owner', setting: 'usage_budgets'},
            ],
            preparation: null,
            usage: [
              {connection: 'c1', unit: 'minutes', budget_minutes: 360, used_minutes: 41},
              {connection: 'c2', status: 'connection_owner_only'},
            ],
            observed_unix: 1780000000,
            repository: '7',
          })
        : null
    );
    window.drawerFixture.api.setPresentation('settings');
  });
  await refresh(page);
  await page.getByRole('tab', {name: 'Factory'}).click();
  const settings = page.locator('[data-factory-settings]');
  assert(await settings.getByLabel('Intent active minutes', {exact: true}).isDisabled());
  await settings.getByText('c1: 360 minutes budget; 41 minutes used.').waitFor();
  assert.doesNotMatch(await settings.innerText(), /planning calls/i);
  assert.match(await settings.innerText(), /c2: visible to its connection owner only/);
  assert.match(await settings.innerText(), /Budget read-only: needs connection owner/);
  assert.match(await settings.innerText(), /Policy: read-only \(needs repository owner or administrator\)/);
  assert.match(await settings.innerText(), /Activity display: editable/);
  assert.match(await settings.innerText(), /usage_budget_missing: resolve as connection owner in usage_budgets/);
  assert.doesNotMatch(await settings.innerText(), /\$/);
  assert.equal((await writes(page)).filter((call) => call.method !== 'GET').length, 0);
});
test('Factory execution usage stays honestly unavailable without the access read', async (t) => {
  const page = await fixture(t);
  await page.evaluate(() => {
    window.drawerFixture.setReply(async (call) =>
      call.url.endsWith('/factory/access') ? Response.json({error: {code: 'unavailable'}}, {status: 503}) : null
    );
    window.drawerFixture.api.setPresentation('settings');
  });
  await refresh(page);
  await page.getByRole('tab', {name: 'Factory'}).click();
  const settings = page.locator('[data-factory-settings]');
  await settings.getByText('Execution usage is unavailable.').waitFor();
});
test('Factory settings reload after a tab flap during load instead of sticking busy', async (t) => {
  const page = await fixture(t);
  await page.evaluate(() => {
    const stalled = window.drawerFixture as unknown as {
      gate?: Promise<unknown>;
      release?: (body: unknown) => void;
    };
    stalled.gate = new Promise<unknown>((resolve) => {
      stalled.release = resolve;
    });
    window.drawerFixture.setReply(async (call) => {
      if (!call.url.endsWith('/api/repositories/7/factory')) return null;
      return Response.json(await stalled.gate);
    });
    window.drawerFixture.api.setPresentation('settings');
  });
  await refresh(page);
  await page.getByRole('tab', {name: 'Factory'}).click();
  await page.waitForFunction(() =>
    document
      .querySelector('soda-factory-settings p[role="status"]')
      ?.textContent?.includes('Refreshing current authority')
  );
  await page.getByRole('tab', {name: 'Overview'}).click();
  await page.getByRole('tab', {name: 'Factory'}).click();
  await page.evaluate(() => {
    const stalled = window.drawerFixture as unknown as {release?: (body: unknown) => void};
    stalled.release?.({
      repository: '7',
      policy: null,
      operator_grant: null,
      capacity: {revision: 4, max_concurrent_runs: 2, max_queued: 8},
      sponsorships: [],
      environment_grant: null,
      preparation: {project: 'p0123456789abcdef01234567', requirements: '', approval: '', ready: false},
      queue: {queued: 1, active: 0},
      effective: {effective: false, dispatch_open: false, missing: ['policy_missing'], authority: {}},
    });
  });
  await page.locator('[data-factory-settings]').filter({hasText: 'Capacity: 2 concurrent runs, 8 queued'}).waitFor();
  const refreshButton = page.locator('soda-factory-settings').getByRole('button', {name: 'Refresh current authority'});
  assert.equal(await refreshButton.isDisabled(), false);
});

test('Factory settings resume closed dispatch with no open work through the actions route', async (t) => {
  const page = await fixture(t);
  await page.evaluate(() => {
    let resumed = false;
    window.drawerFixture.setReply(async (call) => {
      if (call.url.endsWith('/api/repositories/7/factory') && call.method === 'GET')
        return Response.json({
          repository: '7',
          policy: null,
          operator_grant: null,
          capacity: {revision: 4, max_concurrent_runs: 2, max_queued: 8},
          sponsorships: [],
          environment_grant: null,
          queue: {queued: 0, active: 0},
          effective: resumed
            ? {effective: true, dispatch_open: true, missing: [], authority: {}}
            : {effective: false, dispatch_open: false, missing: ['dispatch_closed'], authority: {}},
        });
      if (call.url.endsWith('/api/repositories/7/factory/actions') && call.method === 'POST') {
        const body = JSON.parse(call.body || '{}');
        resumed = true;
        return Response.json({
          command_id: body.command_id,
          revision: 6,
          reopened: true,
          effective: {effective: true, dispatch_open: true, missing: []},
        });
      }
      return null;
    });
    window.drawerFixture.api.setPresentation('settings');
  });
  await refresh(page);
  await page.getByRole('tab', {name: 'Factory'}).click();
  const settings = page.locator('soda-factory-settings');
  await settings.getByRole('button', {name: 'Resume Factory work', exact: true}).waitFor();
  assert.match(await settings.innerText(), /Dispatch: closed/);
  await settings.getByRole('button', {name: 'Resume Factory work', exact: true}).click();
  await page.getByRole('status').filter({hasText: 'Change confirmed by the current API.'}).waitFor();
  const sent = (await writes(page)).filter((call) => call.url.endsWith('/api/repositories/7/factory/actions'));
  assert.equal(sent.length, 1);
  assert.equal(sent[0]?.method, 'POST');
  const payload = JSON.parse(sent[0]?.body || '{}') as Record<string, unknown>;
  assert.equal(payload.action, 'resume');
  assert.match(String(payload.command_id), /^[a-f0-9]{32}$/);
  assert.match(await settings.innerText(), /Dispatch: open/);
});

test('Factory settings refuse a closed-dispatch resume without replaying the command', async (t) => {
  const page = await fixture(t);
  await page.evaluate(() => {
    window.drawerFixture.setReply(async (call) => {
      if (call.url.endsWith('/api/repositories/7/factory') && call.method === 'GET')
        return Response.json({
          repository: '7',
          policy: null,
          operator_grant: null,
          capacity: {revision: 4, max_concurrent_runs: 2, max_queued: 8},
          sponsorships: [],
          environment_grant: null,
          queue: {queued: 0, active: 0},
          effective: {effective: false, dispatch_open: false, missing: ['dispatch_closed'], authority: {}},
        });
      if (call.url.endsWith('/api/repositories/7/factory/actions') && call.method === 'POST')
        return Response.json(
          {error: {code: 'authority_ineffective', message: 'Current grants do not authorize this action.'}},
          {status: 409}
        );
      return null;
    });
    window.drawerFixture.api.setPresentation('settings');
  });
  await refresh(page);
  await page.getByRole('tab', {name: 'Factory'}).click();
  const settings = page.locator('soda-factory-settings');
  await settings.getByRole('button', {name: 'Resume Factory work', exact: true}).click();
  await page.getByRole('status').filter({hasText: 'Current grants do not authorize this action.'}).waitFor();
  const sent = (await writes(page)).filter((call) => call.url.endsWith('/api/repositories/7/factory/actions'));
  assert.equal(sent.length, 1);
  assert.match(await settings.innerText(), /Dispatch: closed/);
});

test('Factory settings disable dispatch resume without repository write access', async (t) => {
  const page = await fixture(t);
  await page.evaluate(() => {
    window.drawerFixture.setReply(async (call) => {
      if (/\/api\/repositories\/\d+\/factory\/access$/.test(call.url) && call.method === 'GET')
        return Response.json({
          repository: '7',
          observed_unix: 1,
          viewer: {
            code_write: false,
            owner_or_admin: false,
            environment_owner: false,
            operator: false,
            member: true,
            session_current: true,
          },
          editable: {
            policy: {editable: false, reason: 'repository_admin_required'},
            activity_settings: {editable: true},
            operator_grant: {editable: false, reason: 'operator_required'},
            capacity: {editable: false, reason: 'operator_required'},
            environment_grant: {editable: false, reason: 'repository_admin_required'},
            sponsorships: [{connection: 'connection-observed', editable: false}],
            usage_budgets: [{connection: 'connection-observed', editable: false}],
          },
          effective: {},
          required_actions: [],
          preparation: null,
          usage: [],
          planning: [],
        });
      if (call.url.endsWith('/api/repositories/7/factory') && call.method === 'GET')
        return Response.json({
          repository: '7',
          policy: null,
          operator_grant: null,
          capacity: {revision: 4, max_concurrent_runs: 2, max_queued: 8},
          sponsorships: [],
          environment_grant: null,
          queue: {queued: 0, active: 0},
          effective: {effective: false, dispatch_open: false, missing: ['dispatch_closed'], authority: {}},
        });
      return null;
    });
    window.drawerFixture.api.setPresentation('settings');
  });
  await refresh(page);
  await page.getByRole('tab', {name: 'Factory'}).click();
  const settings = page.locator('soda-factory-settings');
  const resume = settings.getByRole('button', {name: 'Resume Factory work', exact: true});
  await resume.waitFor();
  assert.equal(await resume.isDisabled(), true);
  assert.match(await settings.innerText(), /Resume requires repository write access for your account\./);
  const sent = (await writes(page)).filter((call) => call.url.endsWith('/api/repositories/7/factory/actions'));
  assert.equal(sent.length, 0);
});

test(
  'a slow preparation preview within the backend budget renders instead of aborting',
  {timeout: 90000},
  async (t) => {
    const page = await fixture(t);
    await page.evaluate(() => {
      window.drawerFixture.setReply(async (call) => {
        if (call.url.endsWith('/api/repositories/7/factory') && call.method === 'GET')
          return Response.json({
            repository: '7',
            appliance: {revision: 1, max_concurrent_runs: 1, max_queued: 0},
          });
        if (call.url.endsWith('/api/environments/p0123456789abcdef01234567/preparation') && call.method === 'GET') {
          await new Promise((resolve) => setTimeout(resolve, 20000));
          return Response.json({
            project: 'test-project',
            hold: true,
            hold_revision: 6,
            requirements: '',
            approval: '',
            preparations: [
              {
                id: 'prep-slow',
                role: 'coder',
                phase: 'complete',
                ready: true,
                source_commit: 'a'.repeat(40),
                setup_digest: 'b'.repeat(64),
              },
            ],
          });
        }
        return null;
      });
      window.drawerFixture.api.setPresentation('settings');
    });
    const started = Date.now();
    await refresh(page);
    await page.getByRole('tab', {name: 'Factory'}).click();
    const settings = page.locator('soda-factory-settings');
    await settings
      .getByText(/ID prep-slow/)
      .first()
      .waitFor({timeout: 45000});
    assert.match(await settings.innerText(), /coder · complete · ID prep-slow · ready/);
    assert(Date.now() - started >= 19000, 'The UI waited out the slow preview instead of aborting it early');
    assert.equal(
      await settings.getByRole('button', {name: 'Refresh current authority', exact: true}).isDisabled(),
      false
    );
  }
);

test(
  'a preparation preview hung past the deadline reports honestly instead of sticking busy',
  {timeout: 150000},
  async (t) => {
    const page = await fixture(t);
    await page.evaluate(() => {
      window.drawerFixture.setReply(async (call) => {
        if (call.url.endsWith('/api/repositories/7/factory') && call.method === 'GET')
          return Response.json({
            repository: '7',
            appliance: {revision: 1, max_concurrent_runs: 1, max_queued: 0},
          });
        if (call.url.endsWith('/api/environments/p0123456789abcdef01234567/preparation') && call.method === 'GET')
          return new Promise<Response | null>(() => {});
        return null;
      });
      window.drawerFixture.api.setPresentation('settings');
    });
    const started = Date.now();
    await refresh(page);
    await page.getByRole('tab', {name: 'Factory'}).click();
    const settings = page.locator('soda-factory-settings');
    const status = settings.getByRole('status').filter({hasText: /unavailable|timed out|timeout/i});
    await status.waitFor({timeout: 120000});
    assert(
      Date.now() - started >= 40000,
      'The hung preview survived past the old 15s deadline before the new deadline aborted it'
    );
    assert.equal(await settings.getByText(/ID prep-/).count(), 0);
    assert.equal(
      await settings.getByRole('button', {name: 'Refresh current authority', exact: true}).isDisabled(),
      false
    );
  }
);
