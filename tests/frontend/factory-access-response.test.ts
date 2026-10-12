import {test} from 'bun:test';
import assert from 'node:assert/strict';
import {factoryAccessResponse} from '../../frontend/spaces/sodaspaces-factory-access-response';

const viewer = {
  code_write: true,
  owner_or_admin: false,
  environment_owner: false,
  operator: false,
  member: true,
  session_current: true,
};
const editable = {
  policy: {editable: false, reason: 'owner_or_admin_required'},
  activity_settings: {editable: true},
  operator_grant: {editable: false, reason: 'operator_required'},
  capacity: {editable: true},
  environment_grant: {editable: false, reason: 'repository_owner_required'},
  sponsorships: [{connection: 'c1', editable: true}],
  usage_budgets: [{connection: 'c1', editable: false, reason: 'connection_owner_required'}],
};
const dto = () => ({
  viewer: {...viewer},
  editable: JSON.parse(JSON.stringify(editable)),
  effective: {missing: [], effective: true, dispatch_open: true},
  required_actions: [{code: 'usage_budget_missing', resolve_by: 'connection_owner', setting: 'usage_budgets'}],
  preparation: null,
  usage: [{connection: 'c1', unit: 'minutes', budget_minutes: 360, used_minutes: 41}] as Record<string, unknown>[],
  planning: [],
  observed_unix: 1780000000,
  repository: '7',
});
const allowance = () => ({
  intent: 'd222222222222222222222222',
  id: 'd333333333333333333333333',
  unit: 'planning_calls',
  amount: 12,
  reserved: 2,
  consumed: 3,
  available: 7,
  ceiling: 20,
  effect: 'Explore checkout flow',
  status: 'ok',
  editable: {editable: true},
});
test('owner permission/usage DTO parses with exact units and no conversion', () => {
  const parsed = factoryAccessResponse(dto());
  assert.equal(parsed.viewer.member, true);
  assert.equal(parsed.editable.policy.reason, 'owner_or_admin_required');
  assert.equal(parsed.usage.length, 1);
  assert.equal(parsed.usage[0]?.unit, 'minutes');
  assert.equal(parsed.usage[0]?.budget_minutes, 360);
  assert.equal(parsed.usage[0]?.used_minutes, 41);
  assert.equal(parsed.required_actions[0]?.resolve_by, 'connection_owner');
  assert.equal(parsed.preparation, null);
});
test('non-owner usage stays a status; missing budget is unavailable, never zero', () => {
  const body = dto();
  body.usage = [
    {connection: 'c2', status: 'connection_owner_only'},
    {connection: 'c1', unit: 'minutes'},
  ];
  const parsed = factoryAccessResponse(body);
  assert.equal(parsed.usage[0]?.status, 'connection_owner_only');
  assert.equal(parsed.usage[0]?.budget_minutes, undefined);
  assert.equal(parsed.usage[1]?.unit, 'minutes');
  assert.equal(parsed.usage[1]?.budget_minutes, undefined);
  assert.equal(parsed.usage[1]?.used_minutes, undefined);
});
test('unknown reason and status codes stay visible instead of failing the read', () => {
  const body = dto();
  body.editable.policy = {editable: false, reason: 'future_reason'};
  body.usage = [{connection: 'c9', status: 'future_status'}];
  const parsed = factoryAccessResponse(body);
  assert.equal(parsed.editable.policy.reason, 'future_reason');
  assert.equal(parsed.usage[0]?.status, 'future_status');
});
test('planning allowances parse with real call units and no conversion', () => {
  const body = dto() as unknown as Record<string, unknown>;
  body.planning = [allowance(), {...allowance(), id: 'd444444444444444444444444', status: 'exhausted'}];
  body.planning_truncated = true;
  const parsed = factoryAccessResponse(body);
  assert.equal(parsed.planning.length, 2);
  assert.equal(parsed.planning[0]?.unit, 'planning_calls');
  assert.equal(parsed.planning[0]?.available, 7);
  assert.equal(parsed.planning[0]?.effect, 'Explore checkout flow');
  assert.equal(parsed.planning[1]?.status, 'exhausted');
  assert.equal(parsed.planning_truncated, true);
});
test('malformed permission DTO is refused without partial use', () => {
  for (const mutate of [
    (body: ReturnType<typeof dto>) => ({...body, viewer: {...viewer, member: 'yes'}}),
    (body: ReturnType<typeof dto>) => ({...body, usage: [{connection: 'c1'}]}),
    (body: ReturnType<typeof dto>) => ({
      ...body,
      usage: [{connection: 'c1', unit: 'minutes', used_minutes: -1}],
    }),
    (body: ReturnType<typeof dto>) => ({...body, repository: ''}),
    (body: ReturnType<typeof dto>) => ({...body, required_actions: [{code: 'x'}]}),
    (body: ReturnType<typeof dto>) => ({...body, planning: [{...allowance(), unit: 'minutes'}]}),
    (body: ReturnType<typeof dto>) => ({...body, planning: [{...allowance(), available: -1}]}),
    (body: ReturnType<typeof dto>) => {
      const {planning, ...rest} = body as unknown as Record<string, unknown>;
      void planning;
      return rest;
    },
  ])
    assert.throws(() => factoryAccessResponse(mutate(dto())));
});
