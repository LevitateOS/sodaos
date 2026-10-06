// Soda response contracts used by its two browser components. No copied Forgejo authority.

// Never accept HTML, invalid UTF-8 or an unbounded body as Soda JSON.
async function sodaJSONReader(response: Response) {
  const json = /^application\/json(?:;|$)/i.test(response.headers.get('Content-Type') || '');
  if (!json || Number(response.headers.get('Content-Length')) > 65536) {
    await response.body?.cancel();
    throw Error('Invalid Soda response');
  }
  if (!response.body) throw Error('Missing Soda response body');
  return response.body.getReader();
}

export async function readSodaJSON(response: Response): Promise<unknown> {
  const reader = await sodaJSONReader(response),
    decoder = new TextDecoder('utf-8', {fatal: true});
  let size = 0,
    text = '';
  try {
    for (;;) {
      const {done, value} = await reader.read();
      if (done) break;
      size += value.byteLength;
      if (size > 65536) throw Error('Oversized Soda response');
      text += decoder.decode(value, {stream: true});
    }
    return JSON.parse(text + decoder.decode());
  } catch (error) {
    await reader.cancel();
    throw error;
  } finally {
    reader.releaseLock();
  }
}
export function object(value: unknown): Record<string, unknown> {
  if (value === null || typeof value !== 'object' || Array.isArray(value)) throw Error('Invalid Soda object');
  return value as Record<string, unknown>; // The object shape is checked; properties remain unknown.
}
export function check(condition: unknown): asserts condition {
  if (!condition) throw Error('Invalid or mismatched Soda response');
}
export const id = (value: unknown): value is string =>
  typeof value === 'string' && /^[1-9][0-9]{0,18}$/.test(value) && BigInt(value) <= 9223372036854775807n;
export const projectId = (value: unknown): value is string =>
  typeof value === 'string' && /^p[0-9a-f]{24}$/.test(value);
export const fingerprint = (value: unknown): value is string =>
  typeof value === 'string' && /^SHA256:[A-Za-z0-9+/]{43}$/.test(value);
export interface CreationProfile {
  id: string;
  distribution: string;
  version: string;
  interface: string;
  architecture: string;
  image: string;
  revision: string;
}
function rockyHeadless(
  p: Record<string, unknown>
): Pick<CreationProfile, 'id' | 'distribution' | 'interface' | 'version'> {
  check(
    p.id === 'rocky-headless' &&
      p.distribution === 'rocky' &&
      p.interface === 'headless' &&
      typeof p.version === 'string' &&
      /^[0-9]{1,3}(\.[0-9]{1,3}){0,2}$/.test(p.version)
  );
  return {id: p.id, distribution: p.distribution, version: p.version, interface: p.interface};
}
function profileImage(p: Record<string, unknown>): Pick<CreationProfile, 'architecture' | 'image' | 'revision'> {
  check(p.architecture === 'amd64' || p.architecture === 'arm64');
  check(
    typeof p.image === 'string' &&
      /^sha256:[0-9a-f]{64}$/.test(p.image) &&
      typeof p.revision === 'string' &&
      /^[0-9a-f]{40}$/.test(p.revision)
  );
  return {architecture: p.architecture, image: p.image, revision: p.revision};
}
export function creationProfile(value: unknown): CreationProfile {
  const p = object(value);
  return {...rockyHeadless(p), ...profileImage(p)};
}
export interface Environment {
  id: string;
  repository_id: string;
  profile?: CreationProfile | null;
}
export function environmentResponse(value: unknown, repositoryId: string): Environment {
  const data = object(value);
  check(projectId(data.id) && data.repository_id === repositoryId);
  return {
    id: data.id,
    repository_id: repositoryId,
    profile: data.profile == null ? null : creationProfile(data.profile),
  };
}
export interface Detail {
  environment: Environment & {provisioned: boolean};
  login: string;
  environment_administrator: boolean;
  execution_allowed: boolean;
  native_unavailable: boolean;
  authority_unavailable: boolean;
  observed: {id: string; running: boolean} | null;
}
export function detailResponse(value: unknown, environment: Environment): Detail {
  const data = object(value),
    env = object(data.environment);
  check(
    env.id === environment.id &&
      env.repository_id === environment.repository_id &&
      typeof env.provisioned === 'boolean' &&
      typeof data.login === 'string' &&
      typeof data.environment_administrator === 'boolean' &&
      typeof data.native_unavailable === 'boolean' &&
      typeof data.authority_unavailable === 'boolean'
  );
  check(typeof data.execution_allowed === 'boolean');
  let observed: Detail['observed'] = null;
  if (data.observed !== null) {
    const state = object(data.observed);
    check(state.id === environment.id && typeof state.running === 'boolean');
    observed = {id: environment.id, running: state.running};
  }
  return {
    environment: {...environmentResponse(env, environment.repository_id), provisioned: env.provisioned},
    login: data.login,
    environment_administrator: data.environment_administrator,
    execution_allowed: data.execution_allowed,
    native_unavailable: data.native_unavailable,
    authority_unavailable: data.authority_unavailable,
    observed,
  };
}
export interface OSObservation {
  running: boolean;
  image: string | null;
  release: {id: string; version: string; name: string} | null;
}
function osImage(env: Record<string, unknown>) {
  const image = env.image === undefined ? null : env.image;
  check(image === null || (typeof image === 'string' && /^sha256:[0-9a-f]{64}$/.test(image)));
  return image as string | null;
}
function osReleaseIdentity(
  r: Record<string, unknown>,
  env: Record<string, unknown>,
  data: Record<string, unknown>
): {id: string; version: string} {
  check(
    env.running && !data.os_release_unavailable && typeof r.id === 'string' && /^[a-z0-9][a-z0-9._-]{0,63}$/.test(r.id)
  );
  check(typeof r.version === 'string' && /^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$/.test(r.version));
  return {id: r.id, version: r.version};
}
function osReleaseName(r: Record<string, unknown>): string {
  check(
    typeof r.name === 'string' &&
      r.name.length > 0 &&
      new TextEncoder().encode(r.name).length <= 256 &&
      !/[\p{Cc}\p{Cf}]/u.test(r.name)
  );
  return r.name;
}
function osRelease(data: Record<string, unknown>, env: Record<string, unknown>): OSObservation['release'] {
  if (data.os_release === null) {
    check(data.os_release_unavailable);
    return null;
  }
  const r = object(data.os_release);
  return {...osReleaseIdentity(r, env, data), name: osReleaseName(r)};
}
export function osObservation(value: unknown, environmentID: string): OSObservation {
  const data = object(value),
    env = object(data.environment);
  check(
    env.id === environmentID && typeof env.running === 'boolean' && typeof data.os_release_unavailable === 'boolean'
  );
  return {running: env.running, image: osImage(env), release: osRelease(data, env)};
}
export interface SavedKey {
  id: string;
  fingerprint: string;
  public_key?: string;
}
export function savedKeysResponse(value: unknown): SavedKey[] {
  const data = object(value);
  check(Array.isArray(data.items));
  return data.items.map((value: unknown) => {
    const key = object(value);
    check(id(key.id) && fingerprint(key.fingerprint));
    return {
      id: key.id,
      fingerprint: key.fingerprint,
      ...(typeof key.public_key === 'string' ? {public_key: key.public_key} : {}),
    };
  });
}
export interface ProfileKeys {
  items: (SavedKey & {public_key: string; title: string})[];
  page: number;
  more: boolean;
}
export function profileKeysResponse(value: unknown, page: number): ProfileKeys {
  const data = object(value);
  check(data.page === page && typeof data.more === 'boolean' && Array.isArray(data.items) && data.items.length <= 10);
  const rawItems: unknown[] = data.items,
    keys = savedKeysResponse(data);
  const items = keys.map((key, index) => {
    const row = object(rawItems[index]);
    check(typeof key.public_key === 'string' && typeof row.title === 'string');
    return {...key, public_key: key.public_key, title: row.title};
  });
  return {items, page, more: data.more};
}
export interface KeyPreview {
  login: string;
  revision: string;
  installed_fingerprints: string[];
  saved_fingerprints: string[];
}
export function keyPreviewResponse(value: unknown, login: string): KeyPreview {
  const data = object(value);
  check(
    data.login === login &&
      typeof data.revision === 'string' &&
      /^[0-9a-f]{64}$/.test(data.revision) &&
      Array.isArray(data.installed_fingerprints) &&
      Array.isArray(data.saved_fingerprints)
  );
  const installed: unknown[] = data.installed_fingerprints,
    saved: unknown[] = data.saved_fingerprints;
  check(installed.every(fingerprint) && saved.every(fingerprint));
  return {login, revision: data.revision, installed_fingerprints: installed, saved_fingerprints: saved};
}
export const terminalID = (value: unknown): value is string =>
  typeof value === 'string' && /^[0-9a-f]{32}$/.test(value);
export interface TerminalIdentity {
  expectedUserId: string;
  repositoryId: string;
  environmentId: string;
  login: string;
}
export interface TerminalMetadata {
  id: string;
  environment_id: string;
  repository_id: string;
  user_id: string;
  login: string;
  name: string;
  created_at: number;
  ready: boolean;
  attached: boolean;
  state: 'opening' | 'ready' | 'ending' | 'ended';
}
function admitTerminalBinding(data: Record<string, unknown>, binding: TerminalIdentity): string {
  check(
    terminalID(data.id) &&
      data.environment_id === binding.environmentId &&
      data.repository_id === binding.repositoryId &&
      data.user_id === binding.expectedUserId &&
      data.login === binding.login
  );
  return data.id;
}
function admitTerminalName(data: Record<string, unknown>): string {
  check(typeof data.name === 'string' && Array.from(data.name).length <= 80 && !/[\p{Cc}\p{Cf}]/u.test(data.name));
  return data.name;
}
function admitTerminalClock(data: Record<string, unknown>): number {
  check(typeof data.created_at === 'number' && Number.isSafeInteger(data.created_at) && data.created_at > 0);
  return data.created_at;
}
function knownTerminalState(state: unknown): state is TerminalMetadata['state'] {
  return state === 'opening' || state === 'ready' || state === 'ending' || state === 'ended';
}
function admitTerminalState(data: Record<string, unknown>): Pick<TerminalMetadata, 'ready' | 'attached' | 'state'> {
  check(typeof data.ready === 'boolean' && typeof data.attached === 'boolean' && knownTerminalState(data.state));
  check(data.ready === (data.state === 'ready') && (!data.attached || data.ready));
  return {ready: data.ready, attached: data.attached, state: data.state};
}
export function terminalMetadata(value: unknown, binding: TerminalIdentity): TerminalMetadata {
  const data = object(value);
  return {
    id: admitTerminalBinding(data, binding),
    environment_id: binding.environmentId,
    repository_id: binding.repositoryId,
    user_id: binding.expectedUserId,
    login: binding.login,
    name: admitTerminalName(data),
    created_at: admitTerminalClock(data),
    ...admitTerminalState(data),
  };
}
export function terminalResponse(value: unknown, binding: TerminalIdentity): TerminalMetadata | null {
  const data = object(value);
  return data.terminal === null ? null : terminalMetadata(data.terminal, binding);
}
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
export interface Space {
  tailnet_state?: string;
  factory_authority?: FactoryAuthority;
  factory_control?: FactoryControl;
  factory_runs: FactoryRun[];
  environment: Environment & {name: string; repository: string; owner_id: string; provisioned: boolean};
  login: string;
  environment_administrator: boolean;
  execution_allowed: boolean;
  authority_unavailable: boolean;
  native_unavailable: boolean;
  observed: Detail['observed'];
  terminals: TerminalMetadata[];
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
function spaceNetwork(row: Record<string, unknown>) {
  const network = row.tailnet_state;
  check(network === undefined || (typeof network === 'string' && ['unavailable', 'off', 'managed'].includes(network)));
  return typeof network === 'string' ? network : undefined;
}
function spaceFactoryAuthority(row: Record<string, unknown>) {
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
function spaceFactoryControl(row: Record<string, unknown>) {
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
function spaceFactoryRuns(row: Record<string, unknown>) {
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
export interface FactoryOutput {
  bytes: Uint8Array;
  cursor: number;
  next: number;
  gap: boolean;
  truncated: boolean;
}
function factoryOutputBytes(data: unknown): Uint8Array {
  check(typeof data === 'string' && data.length <= 65536);
  let decoded = '';
  try {
    decoded = atob(data);
  } catch {
    check(false);
  }
  check(decoded.length <= 32768);
  return Uint8Array.from(decoded, (c) => c.charCodeAt(0));
}
// factoryOutputFrame admits one output slice with server-chosen cursors. The
// cursor advance must equal the delivered bytes; a gap jumps the viewer to
// the recorded size instead of inventing bytes.
export function factoryOutputFrame(value: unknown): FactoryOutput {
  const frame = object(value);
  check(Object.keys(frame).sort().join(',') === 'cursor,data,gap,next,truncated,type');
  check(frame.type === 'output' && typeof frame.gap === 'boolean' && typeof frame.truncated === 'boolean');
  check(
    typeof frame.cursor === 'number' &&
      Number.isSafeInteger(frame.cursor) &&
      frame.cursor >= 0 &&
      typeof frame.next === 'number' &&
      Number.isSafeInteger(frame.next) &&
      frame.next >= frame.cursor
  );
  const bytes = factoryOutputBytes(frame.data);
  check(frame.next - frame.cursor === bytes.length);
  return {bytes, cursor: frame.cursor, next: frame.next, gap: frame.gap, truncated: frame.truncated};
}
// factoryOutputCursor admits one output slice against the viewer's position
// and returns the advanced cursor. A truncated first frame jumps a
// zero-cursor viewer into the trailing window; a gap jumps to the recorded
// size; otherwise the cursor must continue exactly.
export function factoryOutputCursor(viewer: number, frame: FactoryOutput): number {
  check(Number.isSafeInteger(viewer) && viewer >= 0);
  if (frame.cursor !== viewer && !frame.gap && !(frame.truncated && viewer === 0)) throw Error('cursor');
  return frame.next;
}
// factoryClosedReason admits the terminal frame of an attachment. Any reason
// ends the view; unknown reasons still end it rather than stalling.
export function factoryClosedReason(value: unknown): string {
  const frame = object(value);
  check(Object.keys(frame).sort().join(',') === 'reason,type');
  check(
    frame.type === 'closed' && typeof frame.reason === 'string' && frame.reason.length > 0 && frame.reason.length <= 64
  );
  return frame.reason;
}
function admitSpaceTerminal(
  value: unknown,
  expectedUserId: string,
  repositoryId: string,
  environmentId: string,
  login: string,
  sessions: Set<string>
) {
  const terminal = terminalMetadata(value, {expectedUserId, repositoryId, environmentId, login});
  check(!sessions.has(terminal.id));
  sessions.add(terminal.id);
  return terminal;
}
function spaceTerminals(
  row: Record<string, unknown>,
  detail: Detail,
  expectedUserId: string,
  repositoryId: string,
  environmentId: string,
  sessions: Set<string>
) {
  check(
    Array.isArray(row.terminals) &&
      row.terminals.length <= 64 &&
      (detail.execution_allowed || row.terminals.length === 0) &&
      (!detail.authority_unavailable || (!detail.environment_administrator && row.terminals.length === 0))
  );
  const terminals = row.terminals.map((value: unknown) =>
    admitSpaceTerminal(value, expectedUserId, repositoryId, environmentId, detail.login, sessions)
  );
  check(sessions.size <= 64);
  return terminals;
}
function spaceItem(value: unknown, expectedUserId: string, seen: Set<string>, sessions: Set<string>): Space {
  const row = object(value),
    env = object(row.environment);
  check(
    projectId(env.id) &&
      id(env.repository_id) &&
      id(env.owner_id) &&
      typeof env.name === 'string' &&
      typeof env.repository === 'string' &&
      !seen.has(env.id)
  );
  seen.add(env.id);
  const environmentId = env.id,
    repositoryId = env.repository_id;
  const detail = detailResponse(row, {id: environmentId, repository_id: repositoryId});
  const terminals = spaceTerminals(row, detail, expectedUserId, repositoryId, environmentId, sessions);
  const network = spaceNetwork(row);
  const authority = spaceFactoryAuthority(row);
  const control = spaceFactoryControl(row);
  const runs = spaceFactoryRuns(row);
  return {
    ...detail,
    ...(typeof network === 'string' ? {tailnet_state: network} : {}),
    ...(authority ? {factory_authority: authority} : {}),
    ...(control ? {factory_control: control} : {}),
    factory_runs: runs,
    environment: {...detail.environment, name: env.name, repository: env.repository, owner_id: env.owner_id},
    terminals,
  };
}
export function spacesResponse(value: unknown): {
  actor: {id: string; login: string};
  items: Space[];
  complete: boolean;
  nextAfter: string;
  factoryIncomplete: boolean;
} {
  const data = object(value);
  const actor = object(data.actor);
  check(
    id(actor.id) &&
      typeof actor.login === 'string' &&
      actor.login.length > 0 &&
      typeof data.complete === 'boolean' &&
      Array.isArray(data.items) &&
      data.items.length <= 32
  );
  check(data.next_after === undefined || projectId(data.next_after));
  check(data.factory_incomplete === undefined || typeof data.factory_incomplete === 'boolean');
  const seen = new Set<string>(),
    sessions = new Set<string>();
  const items = data.items.map((row: unknown) => spaceItem(row, actor.id as string, seen, sessions));
  return {
    actor: {id: actor.id as string, login: actor.login as string},
    items,
    complete: data.complete,
    nextAfter: typeof data.next_after === 'string' ? data.next_after : '',
    factoryIncomplete: data.factory_incomplete === true,
  };
}
export interface RepositoryChoice {
  id: string;
  owner: string;
  name: string;
  canCreate: boolean;
  project: {id: string; provisioned: boolean} | null;
}
export interface RepositoryChoices {
  items: RepositoryChoice[];
  page: number;
  nextCursor: string;
}
function repositoryPathPart(v: unknown): v is string {
  return (
    typeof v === 'string' &&
    v !== '' &&
    v !== '.' &&
    v !== '..' &&
    new TextEncoder().encode(v).length <= 255 &&
    !/[\/\\\\\p{Cc}\p{Cf}]/u.test(v)
  );
}
function repositoryProject(row: Record<string, unknown>): RepositoryChoice['project'] {
  if (row.project !== null) {
    const p = object(row.project);
    check(projectId(p.id) && typeof p.provisioned === 'boolean' && !row.can_create);
    return {id: p.id, provisioned: p.provisioned};
  }
  check(row.can_create);
  return null;
}
function repositoryChoice(raw: unknown, seen: Set<string>): RepositoryChoice {
  const row = object(raw);
  check(id(row.id) && !seen.has(row.id) && typeof row.can_create === 'boolean');
  seen.add(row.id);
  check(repositoryPathPart(row.owner) && repositoryPathPart(row.name));
  return {id: row.id, owner: row.owner, name: row.name, canCreate: row.can_create, project: repositoryProject(row)};
}
export function repositoryChoices(value: unknown, page: number): RepositoryChoices {
  const data = object(value);
  check(Number.isSafeInteger(page) && page >= 1);
  check(data.page === undefined && data.more === undefined && data.limited === undefined);
  check(data.next_cursor === undefined || (typeof data.next_cursor === 'string' && data.next_cursor.length <= 4096));
  check(Array.isArray(data.items) && data.items.length <= 12);
  const seen = new Set<string>();
  const items = data.items.map((raw: unknown) => repositoryChoice(raw, seen));
  return {items, page, nextCursor: typeof data.next_cursor === 'string' ? data.next_cursor : ''};
}
export class SodaRequestError extends Error {
  constructor(
    readonly status: number,
    readonly code?: string
  ) {
    super('Soda request failed');
  }
}
