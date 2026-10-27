// Exact normal-owner command reconciliation. An uncertain reply never causes
// another mutation: read the recorded command identity, or stop this collector.
import assert from 'node:assert/strict';
import {factoryCommandRecord} from '../../../../frontend/spaces/sodaspaces-factory-work-response';
import {check, sha} from './support';

export interface FactoryCommandIntent {
  readonly label: string;
  readonly repository: number;
  readonly owner: string;
  readonly method: string;
  readonly path: string;
  readonly body: any;
}
type Reply = {status: number; body: any};
export interface FactoryCommandRuntime {
  /** Existing permit persists the exact request before arming transport. */
  prepare(): Promise<void>;
  send(): Promise<Reply>;
  lookup(path: string): Promise<Reply>;
  save(relative: string, value: unknown): Promise<unknown>;
}

export function factoryCommandBinding(intent: FactoryCommandIntent) {
  const base = 'repositories/' + intent.repository + '/factory';
  check(
    Number.isSafeInteger(intent.repository) && intent.repository > 0 && /^[1-9][0-9]*$/.test(intent.owner),
    'factory_command_owner'
  );
  check(/^[a-f0-9]{32}$/.test(intent.body?.command_id), 'factory_command_id');
  if (intent.method === 'PUT' && intent.path === base + '/policy')
    return {type: 'policy', target: 'repository/' + intent.repository + '/policy'};
  if (intent.method === 'POST' && intent.path === base + '/actions' && ['pause', 'resume'].includes(intent.body.action))
    return {type: intent.body.action as string, target: 'repository/' + intent.repository + '/dispatch'};
  const run = /^factory\/runs\/([a-f0-9]{32})\/actions$/.exec(intent.path)?.[1];
  check(intent.method === 'POST' && run && intent.body.action === 'use_current_context', 'factory_command_route');
  return {type: 'use-current-context', target: 'run/' + run};
}

export function reconcileFactoryCommand(intent: FactoryCommandIntent, raw: unknown) {
  const expected = factoryCommandBinding(intent),
    record = factoryCommandRecord(raw, intent.body.command_id);
  const metadata = raw as {payload?: unknown; finished?: unknown};
  check(
    record.type === expected.type && record.target === expected.target && record.principal === 'native:' + intent.owner,
    'factory_command_lookup_binding'
  );
  check(typeof metadata.payload === 'string' && metadata.payload.length <= 8192, 'factory_command_lookup_payload');
  check(
    record.digest === sha(record.type + '\0' + record.target + '\0' + metadata.payload),
    'factory_command_lookup_digest'
  );
  const payload = JSON.parse(metadata.payload);
  if (record.type === 'policy') {
    check(
      payload.repository === String(intent.repository) &&
        payload.granted_by === intent.owner &&
        payload.revision === intent.body.expected_revision,
      'factory_policy_lookup_revision'
    );
    for (const [key, value] of Object.entries(intent.body))
      if (['publish_actor', 'create_actor', 'review_actor', 'resolve_actor', 'merge_actor'].includes(key))
        assert.deepEqual(
          {actor_id: payload[key]?.actor_id, token_id: payload[key]?.token_id},
          value,
          'factory_policy_lookup_actor'
        );
      else if (!['command_id', 'expected_revision'].includes(key))
        assert.deepEqual(payload[key], value, 'factory_policy_lookup_field');
  } else if (record.type === 'use-current-context') assert.deepEqual(payload, intent.body.context);
  else
    check(
      payload.cause === (record.type === 'pause' ? 'control_paused' : 'control_resume'),
      'factory_command_lookup_action'
    );
  check(
    typeof metadata.finished === 'string' && metadata.finished !== '' && typeof record.outcome === 'string',
    'factory_command_pending'
  );
  const outcome = JSON.parse(record.outcome);
  check(outcome.command_id === intent.body.command_id && outcome.error === undefined, 'factory_command_outcome');
  return {
    outcome,
    binding: {
      id: record.id,
      type: record.type,
      target: record.target,
      digest: record.digest,
      finished: metadata.finished,
    },
  };
}

export async function issueFactoryCommand(intent: FactoryCommandIntent, runtime: FactoryCommandRuntime) {
  const expected = factoryCommandBinding(intent);
  await runtime.prepare();
  try {
    const reply = await runtime.send();
    check(
      reply.status === 200 && reply.body?.command_id === intent.body.command_id,
      'factory_command_reply_unconfirmed'
    );
    await runtime.save('commands/' + intent.label + '.receipt.json', {...reply, recovered: false});
    return reply.body;
  } catch {
    // The POST/PUT was attempted once. A 404, pending record, changed payload or
    // failed lookup is uncertainty, never permission to change ID or resend.
    try {
      const reply = await runtime.lookup('factory/commands/' + intent.body.command_id);
      check(reply.status === 200, 'factory_command_lookup_unavailable');
      const found = reconcileFactoryCommand(intent, reply.body);
      await runtime.save('commands/' + intent.label + '.receipt.json', {
        status: 200,
        body: found.outcome,
        recovered: true,
        binding: found.binding,
      });
      return found.outcome;
    } catch {
      await runtime.save('commands/' + intent.label + '.uncertain.json', {
        command_id: intent.body.command_id,
        ...expected,
        automatic_retry: false,
      });
      throw Error('factory_command_requires_reconciliation');
    }
  }
}
