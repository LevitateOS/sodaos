// A view over the existing saved readiness/observation paths. No journal of its
// own, inferred execution phase, authority grant or effect/retry dispatcher.
import type {Readiness} from './readiness';
import type {QualificationScenario} from './scenario';
import {continuablePreparations} from './run';

const record = (v: unknown): Record<string, unknown> =>
  v && typeof v === 'object' && !Array.isArray(v) ? (v as Record<string, unknown>) : {};
const count = (v: unknown): number | null => (typeof v === 'number' && Number.isSafeInteger(v) && v >= 0 ? v : null);
const micros = (v: unknown): string | null =>
  typeof v === 'string' && /^(0|[1-9][0-9]{0,18})$/.test(v) && BigInt(v) <= 9223372036854775807n ? v : null;

const messages: Record<string, string> = {
  execution_released: 'Execution is held by the owner.',
  paid_authority: 'Paid authority is missing or unvalidated; wait without funding, enrollment or a model launch.',
  candidate_source: 'Check the selected source and public manifest bindings.',
  target_trust: 'Resolve pinned target trust before another native observation.',
  native_observation:
    'Selected native metadata is unconfirmed; inspect the saved observation error and refresh only the affected read.',
  installed_identity: 'Installed target, source or payload does not match the selected candidate.',
  installed_images: 'The six installed image identities are unconfirmed or mismatched.',
  docker_identity: 'The observed Docker deployment does not match the pinned release image.',
  custody: 'Native work or custody is live or unconfirmed; reconcile it through its existing owner.',
  command_scope: 'Requested effect identities extend beyond the configured managed resources.',
  managed_inventory: 'A selected managed resource is absent or unconfirmed.',
  persistence_resource_scope: 'Resolve the retained-resource closure blockers before execution.',
  persistence_operation: 'The selected persistence operation is outside configured scope.',
  cold_operation: 'The selected cold recovery operation is outside configured scope.',
  selected_route_omits_retained_project:
    'This saved check predates shared resource retirement; refresh the read-only check with the maintained driver.',
  project_stop_outside_scope:
    'This Project needs confirmed retirement before snapshot and is outside configured stop scope.',
  terminal_actor_unconfirmed: 'Resolve the retained terminal actor through its existing owner; do not infer ownership.',
  terminal_retirement_outside_scope: 'This retained terminal is outside configured retirement scope.',
  terminal_actor_changed:
    'The native terminal actor differs from its recorded binding; preserve it and reconcile ownership.',
  terminal_custody_unconfirmed: 'The terminal custody type is unconfirmed; do not issue normal End.',
  terminal_requires_subscription_owner:
    'This terminal belongs to a subscription; reconcile it through that owner before snapshot.',
  unregistered_native_resource:
    'A native resource is outside the registered inventory; preserve it and reconcile its canonical owner.',
  retained_inventory_changed:
    'Retained Project or App identities changed; refresh discovery before any lifecycle action.',
  runtime_inventory_unconfirmed: 'The complete native resource inventory has not been confirmed.',
};
function explanation(code: string): string {
  if (messages[code]) return messages[code];
  if (/^factory_[0-9]+_connection$/.test(code))
    return 'Inspect the existing selected Identity connection; no enrollment has been requested.';
  if (/^factory_[0-9]+_/.test(code))
    return 'Resolve the selected Factory binding, accepted input, preparation or effect scope through its canonical owner.';
  if (code.startsWith('live_or_unconfirmed_'))
    return 'Native work or custody is live or unconfirmed; preserve it while its existing owner reconciles.';
  if (code.endsWith('_route')) return 'This selected route is not implemented or proven by the maintained driver yet.';
  return 'Resolve the named missing or mismatched observation before execution.';
}

export function journeyStatus(s: QualificationScenario, r: Readiness) {
  const blockers = r.checks
    .filter((c) => c.state !== 'pass')
    .map((c) => ({
      code: c.name,
      state: c.state,
      kind: c.name === 'paid_authority' || c.name === 'execution_released' ? 'authority' : 'readiness',
      resource_id: null as string | null,
      message: explanation(c.name),
    }));
  for (const b of r.closure?.blockers ?? [])
    blockers.push({code: b.code, state: 'blocked', kind: b.kind, resource_id: b.id, message: explanation(b.code)});
  const native = record(r.native),
    history = record(native.history);
  const states = Array.isArray(native.factory_readiness) ? native.factory_readiness.map(record) : [];
  const factory = s.factory.map((f, index) => {
    const b = f.subject,
      observed = r.observation?.factory[index];
    const row = states.find((v) => v.repository_id === b.repository_id && v.issue_number === b.issue_number);
    const a = record(row?.accounting),
      held = count(a.held_reservations),
      settled = micros(a.rolling_settled_micros);
    return {
      repository_id: b.repository_id,
      issue_number: b.issue_number,
      connection_id: b.connection_id,
      current_generation: observed?.connection?.generation ?? null,
      requested: f.limits,
      observed: {
        sponsor_minutes: count(a.sponsor_minutes),
        sponsor_active: typeof a.sponsor_active === 'boolean' ? a.sponsor_active : null,
        sponsor_max_concurrent: count(a.sponsor_max_concurrent),
        rolling_limit_minutes: count(a.rolling_limit_minutes),
        repository_charged_minutes: count(a.repository_charged_minutes),
        held_reservations: held,
        rolling_settled_micros: settled,
        // Store.ConnectionUsageCharge owns the conservative held-reservation
        // calculation. A partial read must never stand in for that admission value.
        rolling_total_charge_micros: held === 0 ? settled : null,
      },
    };
  });
  // Advice over selected canonical fields at the saved observation time. This
  // does not refresh credentials, extend deadlines or infer native retirement.
  const supported_recovery: {code: string; resource_id: string | null; next_action: string}[] = [];
  const recovery = (code: string, resource_id: string | null, next_action: string) =>
    supported_recovery.push({code, resource_id, next_action});
  if (s.authority.held)
    recovery('user_hold', null, 'Wait for the owner to release execution; preserve this immutable run.');
  if (r.checks.some((c) => c.name === 'custody' && c.state !== 'pass'))
    recovery(
      'pending_operation',
      null,
      'Inspect the retained operation through its native owner; wait for confirmed custody/accounting.'
    );
  for (const [index, f] of s.factory.entries()) {
    const b = f.subject,
      connection = r.observation?.factory[index]?.connection,
      row = states.find((v) => v.repository_id === b.repository_id && v.issue_number === b.issue_number),
      preps = Array.isArray(row?.preparations) ? row.preparations.map(record) : [],
      selected = preps.filter((p) => [b.coder_preparation, b.reviewer_preparation].includes(String(p.id)));
    if (!connection || connection.state !== 'ready')
      recovery(
        'missing_authentication',
        b.connection_id,
        'Inspect this existing Identity connection; use its supported sign-in repair only when needed. Do not enroll a replacement automatically.'
      );
    if (selected.length !== 2 || selected.some((p) => p.ready !== true || p.stopped !== false))
      recovery(
        'partial_setup',
        b.project_id,
        continuablePreparations(s, r, index)
          ? 'The same approved/waiting preparations may continue once through the protected helper after scope admission; Factory waits for confirmed ready state.'
          : 'Inspect the exact selected preparation IDs through the Project owner; preserve partial/failed setup and create no replacements.'
      );
    const runs = Array.isArray(row?.runs) ? row.runs.map(record) : [];
    for (const run of runs) {
      const deadline = typeof run.deadline === 'string' ? Date.parse(run.deadline) : NaN;
      if (run.active === true && Number.isFinite(deadline) && deadline <= Date.parse(r.checked_at))
        recovery(
          'expired_attempt',
          typeof run.id === 'string' ? run.id : null,
          'Reconcile the same expired run through Factory/Identity; elapsed time does not prove retirement. Preserve charges and never extend or relaunch it.'
        );
    }
    const allowances = Array.isArray(row?.allowances) ? row.allowances.map(record) : [];
    for (const allowance of allowances) {
      const seconds = count(allowance.active_seconds),
        minutes = count(allowance.active_minutes);
      if (allowance.closed === true || (seconds !== null && minutes !== null && minutes > 0 && seconds >= minutes * 60))
        recovery(
          allowance.closed === true ? 'attempt_closed' : 'exhausted_allowance',
          typeof allowance.root === 'string' ? allowance.root : null,
          'Preserve the recorded allowance. A new attempt requires the canonical explicit Retry and actual finite authority; this runner does not retry or refill.'
        );
    }
    const accounting = factory[index]!.observed;
    if (
      accounting.held_reservations === 0 &&
      ((accounting.sponsor_active === true &&
        accounting.sponsor_minutes !== null &&
        accounting.repository_charged_minutes !== null &&
        accounting.repository_charged_minutes >= accounting.sponsor_minutes) ||
        (accounting.rolling_limit_minutes !== null &&
          accounting.rolling_total_charge_micros !== null &&
          BigInt(accounting.rolling_limit_minutes) * 60_000_000n - BigInt(accounting.rolling_total_charge_micros) <
            60_000_000n))
    )
      recovery(
        'exhausted_allowance',
        b.connection_id,
        'Wait for the rolling window or obtain actual authority for additional finite allowance; preserve usage and do not refill automatically.'
      );
  }
  const pass = (name: string) => r.checks.some((c) => c.name === name && c.state === 'pass');
  const authorityMissing = s.factory.length > 0 && !pass('paid_authority');
  const next_action = s.authority.held
    ? {
        kind: 'wait',
        code: 'user_hold',
        message:
          'Wait for the owner to release execution; preserve the existing connection, failed evidence and recorded usage.',
      }
    : authorityMissing
      ? {
          kind: 'wait',
          code: s.authority.scope_receipt_file === null ? 'paid_authority_missing' : 'paid_authority_unvalidated',
          message:
            'Wait for exact paid authority to be admitted through the supported run path. Do not enroll, fund or launch a model.',
        }
      : blockers.length
        ? {
            kind: 'repair',
            code: 'resolve_readiness_blockers',
            message:
              'Resolve the listed observations or configured scope, then make a fresh read-only check; preserve the prior run.',
          }
        : {
            kind: 'inspect',
            code: 'readiness_observed',
            message:
              'Read-only prerequisites are observed. Inspect existing receipts; this report does not admit execution or qualify the release.',
          };
  return {
    format: 1,
    checked_at: r.checked_at,
    ready: r.ready,
    scope: r.scope,
    checks: r.checks,
    closure: r.closure,
    blockers,
    next_action,
    local_repairs: blockers.filter((b) => b.kind !== 'authority'),
    last_confirmed_phase:
      ['candidate_source', 'target_trust', 'native_observation', 'installed_identity', 'installed_images'].every(
        pass
      ) && r.observation
        ? {name: 'readiness_observation', confirmed_at: r.observation.observed_at}
        : null,
    qualification_phase: null,
    accounting: {
      retained_runs: count(history.runs),
      charged_minutes: count(history.charged_minutes),
      factory,
      provider_token_billing: null,
      units: 'sponsorship and repository charges are model-session minutes; rolling charges are microseconds',
    },
    future_observations: {
      factory: s.factory.map((f, index) => {
        const actual = r.observation?.factory[index];
        return {
          repository_id: f.subject.repository_id,
          issue_number: f.subject.issue_number,
          run: actual?.run ?? null,
          candidate_head: actual?.candidate_head ?? null,
          pr_number: actual?.pr_number ?? null,
          publication: actual?.publication ?? null,
          model_result: null,
          ci: null,
          review: null,
          delivery: null,
        };
      }),
      cases: s.cases.filter((c) => c !== 'factory').map((c) => ({case: c, completion: null, measurements: null})),
    },
    automatic_actions: {enrollment: false, funding: false, model_launch: false, retry: false},
    supported_recovery,
  };
}
