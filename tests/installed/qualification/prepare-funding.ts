// Normal connection-owner setup for one previously approved finite trial.
// This never opens policy/dispatch, creates an allowance or launches a model.
import assert from 'node:assert/strict';
import type {QualificationScenario} from './scenario';
import {factoryCommandID} from './phases/factory-native';

export interface PreparationAPI {
  request(route: string, method?: string, body?: unknown): Promise<{status: number; value: any}>;
  save(name: string, value: unknown): Promise<unknown>;
  readSaved(name: string): Promise<any | null>;
}

export async function prepareFunding(s: QualificationScenario, owner: string, io: PreparationAPI) {
  assert.equal(s.factory.length, 1);
  const {subject: f, limits} = s.factory[0]!;
  assert(limits.automatic_retry === false && limits.automatic_refill === false && limits.extra_inquiries === false);
  const read = async (route: string) => {
    const reply = await io.request(route);
    assert.equal(reply.status, 200, 'preparation_read_unconfirmed');
    return reply.value;
  };
  // An existing intent is never permission to send the effect again. Reads
  // below may adopt a confirmed result after an interrupted response.
  const once = async (name: string, route: string, method: string, body: unknown) => {
    assert.equal(await io.readSaved(name + '.intent.json'), null, 'preparation_effect_uncertain_no_replay');
    await io.save(name + '.intent.json', {route, method, body});
    let status: number | null = null;
    try {
      status = (await io.request(route, method, body)).status;
    } catch {
      // Only the exact normal-owner read below may establish the outcome.
    }
    await io.save(name + '.transport.json', {status, automatic_retry: false});
  };
  const connections = await read('identity/connections');
  assert(Array.isArray(connections));
  const selected = connections.filter((c: any) => c.id === f.connection_id && c.owner_id === owner);
  assert.equal(selected.length, 1);
  const connection = selected[0];
  assert(connection.state === 'ready' && Number.isSafeInteger(connection.generation) && connection.generation > 0);
  const factoryRoute = 'repositories/' + f.repository_id + '/factory';
  const status = await read(factoryRoute);
  assert(status.policy?.paused === true && status.queue?.active === 0, 'factory_must_remain_paused');
  const grantsRoute = 'identity/connections/' + f.connection_id + '/grants';
  const matchingGrants = async () => {
    const rows = await read(grantsRoute);
    assert(Array.isArray(rows));
    return rows.filter(
      (g: any) =>
        g.connection_id === f.connection_id && g.project_id === f.project_id && g.user_id === owner && !g.revoked
    );
  };
  let grants = await matchingGrants();
  assert(grants.length <= 1, 'ambiguous_project_grant');
  if (grants.length === 0) {
    await once('member-grant', 'environments/' + f.project_id + '/identity/grants', 'POST', {
      connection_id: f.connection_id,
      user_id: owner,
      confirm_subscription: true,
      confirm_credential_exposure: true,
    });
    grants = await matchingGrants();
  }
  assert(grants.length === 1 && /^[a-f0-9]{32}$/.test(grants[0].id), 'project_grant_unconfirmed');
  const grant = grants[0];
  const budgetRoute = 'factory/connections/' + f.connection_id + '/usage-budget';
  let budget = await read(budgetRoute);
  assert(budget.connection === f.connection_id && Number.isSafeInteger(budget.revision));
  if (budget.rolling_minutes !== limits.rolling_minutes) {
    assert.equal(budget.rolling_minutes, 15, 'existing_rolling_scope_changed');
    assert.equal(limits.rolling_minutes, 25, 'approved_rolling_scope_required');
    await once('rolling-budget', budgetRoute, 'PUT', {
      command_id: factoryCommandID(f.operation_id, 'prepare-rolling-budget'),
      expected_revision: budget.revision,
      rolling_minutes: limits.rolling_minutes,
    });
    budget = await read(budgetRoute);
  }
  assert.equal(budget.rolling_minutes, limits.rolling_minutes, 'rolling_budget_unconfirmed');
  const sponsorships = async () => {
    const view = await read(factoryRoute);
    assert(view.policy?.paused === true && view.queue?.active === 0, 'factory_must_remain_paused');
    assert(Array.isArray(view.sponsorships));
    return view.sponsorships.filter((x: any) => x.connection === f.connection_id);
  };
  let sponsors = await sponsorships();
  assert(sponsors.length <= 1);
  if (sponsors.length === 0) {
    await once('sponsorship', factoryRoute + '/sponsorships/' + f.connection_id, 'PUT', {
      command_id: factoryCommandID(f.operation_id, 'prepare-sponsorship'),
      expected_revision: 0,
      grant_id: grant.id,
      generation: connection.generation,
      roles: ['soda-coder', 'soda-reviewer'],
      allowance_minutes: limits.sponsor_minutes,
      max_concurrent: limits.max_concurrent,
      active: true,
    });
    sponsors = await sponsorships();
  }
  assert.equal(sponsors.length, 1, 'sponsorship_unconfirmed');
  const sponsor = sponsors[0];
  assert(
    sponsor.granted_by === owner &&
      sponsor.actor_id === owner &&
      sponsor.project_id === f.project_id &&
      sponsor.grant_id === grant.id &&
      sponsor.generation === connection.generation &&
      sponsor.active === true &&
      sponsor.allowance_minutes === limits.sponsor_minutes &&
      sponsor.max_concurrent === limits.max_concurrent,
    'sponsorship_scope_changed'
  );
  assert.deepEqual([...sponsor.roles].sort(), ['soda-coder', 'soda-reviewer']);
  const result = {
    format: 1,
    grant_id: grant.id,
    connection: f.connection_id,
    generation: connection.generation,
    repository: f.repository_id,
    project: f.project_id,
    sponsor_minutes: limits.sponsor_minutes,
    rolling_minutes: budget.rolling_minutes,
    max_concurrent: limits.max_concurrent,
    dispatch_opened: false,
    model_calls: 0,
    automatic_retry: false,
  };
  if (!(await io.readSaved('funding.json'))) await io.save('funding.json', result);
  return result;
}
