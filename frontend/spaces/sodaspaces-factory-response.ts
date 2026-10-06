import {check, id, object, projectId} from './sodaspaces-api.js';
import {terminalID} from './sodaspaces-terminal-response.js';

export interface FactoryAuthority {
  effective: boolean;
  dispatch_open: boolean;
  missing: string[];
}
export interface FactoryControl {
  dispatch_open: boolean;
  paused: boolean;
  unsettled_runs: number;
  withdrawal_cause?: string;
}
export interface FactoryRun {
  id: string;
  role: string;
  issue?: string;
  attempt?: string;
  outcome?: 'succeeded' | 'failed' | 'cancelled' | 'needs-human';
  reconciled: boolean;
}
// factoryAuthorityText renders the admitted authority verdict as a bounded
// status suffix. Reason codes are stable backend identifiers, never secrets.
export function factoryAuthorityText(authority: FactoryAuthority | undefined) {
  if (!authority) return '';
  if (authority.effective) return ' · Factory ready';
  return ' · Factory needs: ' + authority.missing.slice(0, 3).join(', ');
}
// factoryControlText renders the admitted intervention state as a bounded
// status suffix: sticky pause, closed dispatch and unsettled runs.
export function factoryControlText(control: FactoryControl | undefined) {
  if (!control) return '';
  const parts: string[] = [];
  if (control.paused) parts.push('paused');
  if (!control.dispatch_open)
    parts.push('dispatch closed' + (control.withdrawal_cause ? ': ' + control.withdrawal_cause : ''));
  if (control.unsettled_runs > 0) parts.push(control.unsettled_runs + ' unsettled');
  if (parts.length === 0) return '';
  return ' · Factory ' + parts.join(', ');
}
// factoryCommandId generates a client command identity for one idempotent
// lifecycle control. The backend replays a reused identity instead of
// executing twice.
export function factoryCommandId() {
  return crypto.randomUUID().replace(/-/g, '');
}
export function spaceFactoryAuthority(row: Record<string, unknown>) {
  const authority = row.factory_authority;
  if (authority === undefined) return undefined;
  const detail = object(authority);
  check(
    typeof detail.effective === 'boolean' &&
      typeof detail.dispatch_open === 'boolean' &&
      Array.isArray(detail.missing) &&
      detail.missing.length <= 16 &&
      detail.missing.every((reason: unknown) => typeof reason === 'string' && /^[a-z_]{1,64}$/.test(reason)) &&
      (detail.missing.length === 0) === detail.effective
  );
  const missing = detail.missing as string[];
  return {
    effective: detail.effective as boolean,
    dispatch_open: detail.dispatch_open as boolean,
    missing,
  };
}
function admitUnsettledRuns(value: unknown) {
  check(typeof value === 'number' && Number.isInteger(value) && value >= 0 && value <= 1000);
  return value as number;
}
function admitWithdrawalCause(cause: unknown, open: boolean) {
  check(open || cause !== undefined);
  check(cause === undefined || (typeof cause === 'string' && /^[a-z_]{1,256}$/.test(cause)));
  return typeof cause === 'string' ? cause : undefined;
}
export function spaceFactoryControl(row: Record<string, unknown>) {
  const control = row.factory_control;
  if (control === undefined) return undefined;
  const detail = object(control);
  check(typeof detail.dispatch_open === 'boolean' && typeof detail.paused === 'boolean');
  const open = detail.dispatch_open as boolean;
  const cause = admitWithdrawalCause(detail.withdrawal_cause, open);
  return {
    dispatch_open: open,
    paused: detail.paused as boolean,
    unsettled_runs: admitUnsettledRuns(detail.unsettled_runs),
    ...(cause ? {withdrawal_cause: cause} : {}),
  };
}
// factoryRunText renders one recorded run as a bounded status line: its role,
// issue or attempt binding and recorded outcome. Liveness stays on the run
// status route; this text never claims the process is running.
export function factoryRunText(run: FactoryRun) {
  const target = run.issue ? `issue #${run.issue}` : run.attempt ? run.attempt : `run ${run.id.slice(0, 8)}`;
  return `${run.role} · ${target} · ${run.outcome || 'active'}`;
}
const factoryOutcome = (value: unknown): value is FactoryRun['outcome'] =>
  value === 'succeeded' || value === 'failed' || value === 'cancelled' || value === 'needs-human';
const factoryAttempt = (value: unknown): value is string =>
  typeof value === 'string' && /^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$/.test(value);
function admitFactoryRunIdentity(run: Record<string, unknown>) {
  check(terminalID(run.id) && typeof run.role === 'string' && /^[a-z][a-z0-9-]{0,63}$/.test(run.role));
  return {id: run.id, role: run.role};
}
function admitFactoryRunOutcome(run: Record<string, unknown>) {
  check(typeof run.reconciled === 'boolean');
  check(run.outcome === undefined || factoryOutcome(run.outcome));
  return {
    reconciled: run.reconciled,
    ...(run.outcome === undefined ? {} : {outcome: run.outcome}),
  };
}
function admitFactoryRunBinding(run: Record<string, unknown>) {
  check(run.issue === undefined || id(run.issue));
  check(run.attempt === undefined || factoryAttempt(run.attempt));
  return {
    ...(run.issue === undefined ? {} : {issue: run.issue}),
    ...(run.attempt === undefined ? {} : {attempt: run.attempt}),
  };
}
function spaceFactoryRun(value: unknown): FactoryRun {
  const run = object(value);
  return {...admitFactoryRunIdentity(run), ...admitFactoryRunOutcome(run), ...admitFactoryRunBinding(run)};
}
export function spaceFactoryRuns(row: Record<string, unknown>) {
  check(Array.isArray(row.factory_runs) && row.factory_runs.length <= 64);
  const seen = new Set<string>();
  return row.factory_runs.map((value: unknown) => {
    const run = spaceFactoryRun(value);
    check(!seen.has(run.id));
    seen.add(run.id);
    return run;
  });
}
export interface FactoryRunView {
  repository: string;
  issue?: string;
  attempt?: string;
}
export interface FactoryRunState {
  phase: string;
  live: boolean;
  terminal: boolean;
  output_truncated: boolean;
  container?: string;
  unit?: string;
  invocation?: string;
  exit_code?: number;
  reason?: string;
  retirement?: string;
  output?: string;
}
export interface FactoryRunDetail {
  id: string;
  project_id: string;
  role: string;
  harness: string;
  model: string;
  reconciled: boolean;
  outcome?: FactoryRun['outcome'];
  summary?: string;
  view?: FactoryRunView;
  state?: FactoryRunState;
}
const factoryPhase = (value: unknown): value is string =>
  value === 'approved' ||
  value === 'running' ||
  value === 'completed' ||
  value === 'failed' ||
  value === 'stopped' ||
  value === 'uncertain';
function factoryRunDetailView(value: unknown, repositoryId: string): FactoryRunView {
  const view = object(value);
  check(view.repository === repositoryId);
  check(view.issue === undefined || id(view.issue));
  check(view.attempt === undefined || factoryAttempt(view.attempt));
  return {
    repository: repositoryId,
    ...(view.issue === undefined ? {} : {issue: view.issue}),
    ...(view.attempt === undefined ? {} : {attempt: view.attempt}),
  };
}
function factoryStateIdentity(state: Record<string, unknown>) {
  check(
    factoryPhase(state.phase) &&
      typeof state.live === 'boolean' &&
      typeof state.terminal === 'boolean' &&
      typeof state.output_truncated === 'boolean'
  );
  return {phase: state.phase, live: state.live, terminal: state.terminal, output_truncated: state.output_truncated};
}
function factoryStateBinding(state: Record<string, unknown>) {
  check(
    state.container === undefined || (typeof state.container === 'string' && /^[0-9a-f]{64}$/.test(state.container))
  );
  check(
    state.unit === undefined ||
      (typeof state.unit === 'string' && /^soda-factory-[0-9a-f]{32}\.service$/.test(state.unit))
  );
  check(state.invocation === undefined || terminalID(state.invocation));
  return {
    ...(state.container === undefined ? {} : {container: state.container}),
    ...(state.unit === undefined ? {} : {unit: state.unit}),
    ...(state.invocation === undefined ? {} : {invocation: state.invocation}),
  };
}
function factoryStateExit(state: Record<string, unknown>) {
  check(
    state.exit_code === undefined ||
      (typeof state.exit_code === 'number' &&
        Number.isInteger(state.exit_code) &&
        state.exit_code >= 0 &&
        state.exit_code <= 255)
  );
  if (state.exit_code === undefined) return {};
  return {exit_code: state.exit_code};
}
function factoryStateReason(state: Record<string, unknown>) {
  check(
    state.reason === undefined || (typeof state.reason === 'string' && /^[a-z][a-z0-9-]{0,63}$/.test(state.reason))
  );
  check(state.retirement === undefined || state.retirement === 'confirmed' || state.retirement === 'uncertain');
  return {
    ...(state.reason === undefined ? {} : {reason: state.reason}),
    ...(state.retirement === undefined ? {} : {retirement: state.retirement}),
  };
}
function factoryStateOutput(state: Record<string, unknown>) {
  check(state.output === undefined || (typeof state.output === 'string' && state.output.length <= 16384));
  if (state.output === undefined) return {};
  return {output: state.output};
}
function factoryStateOutcome(state: Record<string, unknown>) {
  return {...factoryStateReason(state), ...factoryStateOutput(state)};
}
function factoryRunDetailState(value: unknown): FactoryRunState {
  const state = object(value);
  const identity = factoryStateIdentity(state);
  check(!identity.output_truncated || state.output !== undefined);
  return {...identity, ...factoryStateBinding(state), ...factoryStateExit(state), ...factoryStateOutcome(state)};
}
function factoryRecordIdentity(record: Record<string, unknown>, runId: string) {
  check(
    record.id === runId &&
      projectId(record.project_id) &&
      typeof record.role === 'string' &&
      /^[a-z][a-z0-9-]{0,63}$/.test(record.role) &&
      typeof record.reconciled === 'boolean'
  );
  return {id: runId, project_id: record.project_id, role: record.role, reconciled: record.reconciled};
}
function factoryRecordHarness(record: Record<string, unknown>) {
  check(
    typeof record.harness === 'string' &&
      record.harness.length > 0 &&
      record.harness.length <= 128 &&
      typeof record.model === 'string' &&
      record.model.length > 0 &&
      record.model.length <= 128
  );
  check(typeof record.input_sha === 'string' && /^[0-9a-f]{40}$/.test(record.input_sha));
  return {harness: record.harness, model: record.model};
}
function factoryRecordOutcome(record: Record<string, unknown>) {
  check(record.outcome === undefined || factoryOutcome(record.outcome));
  check(record.summary === undefined || (typeof record.summary === 'string' && record.summary.length <= 16384));
  return {
    ...(record.outcome === undefined ? {} : {outcome: record.outcome}),
    ...(record.summary === undefined ? {} : {summary: record.summary}),
  };
}
function factoryRecordProvenance(record: Record<string, unknown>) {
  return {...factoryRecordHarness(record), ...factoryRecordOutcome(record)};
}
// factoryRunStatusResponse binds one run status read to its exact run and
// repository. Native IDs stay decimal strings; the phase set is closed.
export function factoryRunStatusResponse(value: unknown, runId: string, repositoryId: string): FactoryRunDetail {
  const data = object(value),
    record = object(data.run);
  check(data.view === undefined || typeof data.view === 'object');
  check(data.state === undefined || typeof data.state === 'object');
  return {
    ...factoryRecordIdentity(record, runId),
    ...factoryRecordProvenance(record),
    ...(data.view === undefined ? {} : {view: factoryRunDetailView(data.view, repositoryId)}),
    ...(data.state === undefined ? {} : {state: factoryRunDetailState(data.state)}),
  };
}
export interface FactoryStatus {
  run_id: string;
  phase: string;
  container: string;
  unit: string;
  invocation: string;
  reason: string;
  live: boolean;
  terminal: boolean;
  exit_code: number | null;
}
function factoryFrameIdentity(frame: Record<string, unknown>, runId: string) {
  check(frame.type === 'status' && frame.run_id === runId && factoryPhase(frame.phase));
  check(typeof frame.live === 'boolean' && typeof frame.terminal === 'boolean');
  return {run_id: runId, phase: frame.phase, live: frame.live, terminal: frame.terminal};
}
function factoryFrameBinding(frame: Record<string, unknown>) {
  check(typeof frame.container === 'string' && (frame.container === '' || /^[0-9a-f]{64}$/.test(frame.container)));
  check(
    typeof frame.unit === 'string' && (frame.unit === '' || /^soda-factory-[0-9a-f]{32}\.service$/.test(frame.unit))
  );
  check(typeof frame.invocation === 'string' && (frame.invocation === '' || terminalID(frame.invocation)));
  return {container: frame.container, unit: frame.unit, invocation: frame.invocation};
}
function factoryFrameOutcome(frame: Record<string, unknown>) {
  check(typeof frame.reason === 'string' && (frame.reason === '' || /^[a-z][a-z0-9-]{0,63}$/.test(frame.reason)));
  check(
    frame.exit_code === null ||
      (typeof frame.exit_code === 'number' &&
        Number.isInteger(frame.exit_code) &&
        frame.exit_code >= 0 &&
        frame.exit_code <= 255)
  );
  return {reason: frame.reason, exit_code: frame.exit_code};
}
// factoryStatusFrame admits one status frame with its exact run binding and
// fixed key set. Empty container/unit/invocation means no process binding
// was ever recorded; the viewer never guesses one.
export function factoryStatusFrame(value: unknown, runId: string): FactoryStatus {
  const frame = object(value);
  check(
    Object.keys(frame).sort().join(',') === 'container,exit_code,invocation,live,phase,reason,run_id,terminal,type,unit'
  );
  return {...factoryFrameIdentity(frame, runId), ...factoryFrameBinding(frame), ...factoryFrameOutcome(frame)};
}
