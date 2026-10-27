// Adopt the selected normal-owner setup; never create a replacement Project,
// membership, preparation or Identity connection to get past an unresolved one.
import {check, id, object, projectId} from '../../../frontend/spaces/sodaspaces-api';
import {availableConnectionView} from '../../../frontend/spaces/soda-identity-response';
import {detailResponse} from '../../../frontend/spaces/sodaspaces-project-response';
import type {FactorySubject} from './scenario';

export interface SetupContinuation {
  readonly setup_digest: string;
  readonly source_paths: readonly string[];
  readonly setup_files: readonly string[];
  readonly required_tools: readonly string[];
}
export interface SetupSelection {
  readonly subject: FactorySubject;
  readonly owner_id: string;
  readonly connection_generation: number;
  readonly held: boolean;
  /** Non-null only after the caller admits this exact unpaid setup scope. */
  readonly continuation: SetupContinuation | null;
}
export interface SetupPreparationIntent {
  readonly method: 'POST';
  readonly path: string;
  readonly body: {
    readonly action: 'prepare';
    readonly requirement: string;
    readonly approval: string;
    readonly source_paths: readonly string[];
    readonly setup_files: readonly string[];
    readonly required_tools: readonly string[];
    readonly coder_id: string;
    readonly reviewer_id: string;
  };
}
export interface SetupRuntime {
  /** Mounted normal-owner API transport. Paths are queryless /api paths. */
  readonly get: (path: string) => Promise<unknown>;
  readonly continuePreparation: (intent: SetupPreparationIntent) => Promise<void>;
  /** Existing exclusive, durable evidence writer; an existing path must refuse. */
  readonly save: (relative: string, value: unknown) => Promise<unknown>;
}
export interface SetupPreparation {
  readonly id: string;
  readonly role: 'soda-coder' | 'soda-reviewer';
  readonly phase: string;
  readonly ready: boolean;
  readonly stopped: boolean;
  readonly source_commit: string;
  readonly setup_digest: string;
  readonly requirements: string;
  readonly approval: string;
  readonly setup_exit: number | null;
  readonly check_exit: number | null;
}
export interface SetupAdoption {
  readonly format: 1;
  readonly state: 'ready' | 'blocked' | 'pending';
  readonly blockers: readonly string[];
  readonly project: {
    readonly id: string;
    readonly repository_id: string;
    readonly owner_id: string;
    readonly login: string;
  } | null;
  readonly connection: {
    readonly id: string;
    readonly owner_id: string;
    readonly generation: number;
    readonly state: 'ready';
    readonly provider_id: 'codex';
  } | null;
  readonly preparations: readonly SetupPreparation[];
  readonly continuation_sent: boolean;
}

const phases = new Set(['approved', 'waiting', 'running', 'ready', 'failed', 'stopped', 'interrupted']);
const preparationID = (value: unknown) => typeof value === 'string' && /^f[a-f0-9]{24}$/.test(value);
function selectedPreparation(
  value: unknown,
  expectedID: string,
  role: SetupPreparation['role']
): SetupPreparation | null {
  check(Array.isArray(value));
  const rows = value.map(object).filter((row) => row.id === expectedID);
  check(rows.length <= 1);
  const row = rows[0];
  if (!row) return null;
  check(
    row.role === role &&
      phases.has(String(row.phase)) &&
      typeof row.ready === 'boolean' &&
      typeof row.stopped === 'boolean' &&
      row.ready === (row.phase === 'ready') &&
      !(row.ready && row.stopped)
  );
  for (const key of ['source_commit', 'setup_digest', 'requirements', 'approval']) check(typeof row[key] === 'string');
  const exit = (value: unknown) => {
    check(value == null || (Number.isSafeInteger(value) && (value as number) >= 0 && (value as number) <= 255));
    return value == null ? null : (value as number);
  };
  return {
    id: expectedID,
    role,
    phase: row.phase as string,
    ready: row.ready,
    stopped: row.stopped,
    source_commit: row.source_commit as string,
    setup_digest: row.setup_digest as string,
    requirements: row.requirements as string,
    approval: row.approval as string,
    setup_exit: exit(row.setup_exit),
    check_exit: exit(row.check_exit),
  };
}

function connection(value: unknown, selection: SetupSelection) {
  check(Array.isArray(value));
  const rows = value.map(object).filter((row) => row.id === selection.subject.connection_id);
  check(rows.length === 1);
  const raw = rows[0]!,
    parsed = availableConnectionView(raw);
  check(parsed.owner_id === selection.owner_id && parsed.provider_id === 'codex' && parsed.state === 'ready');
  check(Number.isSafeInteger(raw.generation) && raw.generation === selection.connection_generation);
  // Email/plan/labels and unrelated account metadata never enter the journal.
  return {
    id: parsed.id,
    owner_id: parsed.owner_id,
    generation: raw.generation as number,
    state: 'ready' as const,
    provider_id: 'codex' as const,
  };
}

async function inspect(
  selection: SetupSelection,
  runtime: SetupRuntime,
  observedPreparation?: Record<string, unknown>
): Promise<SetupAdoption> {
  const s = selection.subject,
    base = '/api/environments/' + s.project_id;
  const rawDetail = object(await runtime.get(base)),
    environment = object(rawDetail.environment);
  const detail = detailResponse(rawDetail, {id: s.project_id, repository_id: s.repository_id});
  check(environment.owner_id === selection.owner_id);
  const blockers: string[] = [];
  if (
    !detail.environment.provisioned ||
    detail.native_unavailable ||
    detail.authority_unavailable ||
    detail.observed?.running !== true ||
    !detail.execution_allowed ||
    detail.access_held ||
    detail.rejoin_required ||
    detail.login === 'root' ||
    !/^[a-z][a-z0-9_-]{0,30}$/.test(detail.login)
  )
    blockers.push('project_membership_unready');
  const members = object(await runtime.get(base + '/members'));
  check(Array.isArray(members.items) && members.authority_unavailable === false);
  const selectedMembers = members.items.map(object).filter((row) => row.user_id === selection.owner_id);
  check(selectedMembers.length === 1 && selectedMembers[0]!.login === detail.login);
  const memberAccess = object(selectedMembers[0]!.access);
  if (memberAccess.held !== false || memberAccess.requires_join !== false) blockers.push('project_access_held');
  const owned = connection(await runtime.get('/api/identity/connections'), selection);
  const available = connection(await runtime.get(base + '/identity/connections'), selection);
  check(owned.id === available.id && owned.generation === available.generation);
  const preparation = observedPreparation ?? object(await runtime.get(base + '/preparation'));
  check(preparation.project === s.project_id && typeof preparation.hold === 'boolean');
  if (preparation.hold) blockers.push('preparation_hold');
  if (
    preparation.requirements !== s.requirements ||
    preparation.approval !== s.approval ||
    preparation.approval_requirement !== s.requirements
  )
    blockers.push('preparation_heads_changed');
  const selected = [
    selectedPreparation(preparation.preparations, s.coder_preparation, 'soda-coder'),
    selectedPreparation(preparation.preparations, s.reviewer_preparation, 'soda-reviewer'),
  ];
  const preparations = selected.filter((p): p is SetupPreparation => p !== null);
  if (preparations.length !== 2) blockers.push('selected_preparation_missing');
  if (
    preparations.some(
      (p) =>
        p.source_commit !== s.source_commit ||
        p.requirements !== s.requirements ||
        p.approval !== s.approval ||
        !/^[a-f0-9]{64}$/.test(p.setup_digest)
    ) ||
    new Set(preparations.map((p) => p.setup_digest)).size !== 1
  )
    blockers.push('preparation_inputs_changed');
  if (preparations.some((p) => p.stopped || ['failed', 'stopped', 'interrupted'].includes(p.phase)))
    blockers.push('preparation_requires_intervention');
  if (
    preparations.some((p) => p.phase !== 'ready' || !p.ready || p.stopped || p.setup_exit !== 0 || p.check_exit !== 0)
  )
    blockers.push('selected_preparation_unready');
  return {
    format: 1,
    state: blockers.length ? 'blocked' : 'ready',
    blockers,
    project: {
      id: detail.environment.id,
      repository_id: detail.environment.repository_id,
      owner_id: environment.owner_id as string,
      login: detail.login,
    },
    connection: available,
    preparations,
    continuation_sent: false,
  };
}

function continuationIntent(selection: SetupSelection, adoption: SetupAdoption): SetupPreparationIntent | null {
  const input = selection.continuation;
  if (
    selection.held ||
    !input ||
    adoption.blockers.some((b) => b !== 'selected_preparation_unready') ||
    adoption.preparations.length !== 2 ||
    adoption.preparations.some(
      (p) => p.stopped || !['ready', 'approved', 'waiting'].includes(p.phase) || p.setup_digest !== input.setup_digest
    )
  )
    return null;
  const paths = (items: readonly string[], max: number, pattern: RegExp) => {
    check(items.length <= max && new Set(items).size === items.length && items.every((item) => pattern.test(item)));
    return [...items];
  };
  const source_paths = paths(input.source_paths, 64, /^[^\\\u0000-\u001f\u007f]{1,256}$/u);
  check(
    source_paths.length > 0 &&
      source_paths.every(
        (p) =>
          new TextEncoder().encode(p).length <= 256 &&
          !p.startsWith('/') &&
          p.split('/').every((part) => !['', '.', '..', '.git', '.soda-home'].includes(part))
      )
  );
  const setup_files = paths(input.setup_files, 8, /^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$/);
  check(
    setup_files.includes('setup.sh') &&
      setup_files.includes('check.sh') &&
      setup_files.every((p) => !p.includes('..') && source_paths.includes(p))
  );
  const required_tools = paths(input.required_tools, 8, /^[a-z0-9][a-z0-9._+-]{0,31}$/);
  const s = selection.subject;
  return {
    method: 'POST',
    path: '/api/environments/' + s.project_id + '/preparation/actions',
    body: {
      action: 'prepare',
      requirement: s.requirements,
      approval: s.approval,
      source_paths,
      setup_files,
      required_tools,
      coder_id: s.coder_preparation,
      reviewer_id: s.reviewer_preparation,
    },
  };
}

/** A lost response is followed by an exact retained preparation read, never
 * another POST. Ready setup is adopted entirely through normal API reads.
 * The caller owns the target lock; its durable writer refuses repeated intent.
 */
export async function adoptSetup(selection: SetupSelection, runtime: SetupRuntime): Promise<SetupAdoption> {
  check(projectId(selection.subject.project_id) && id(selection.subject.repository_id) && id(selection.owner_id));
  check(
    preparationID(selection.subject.coder_preparation) &&
      preparationID(selection.subject.reviewer_preparation) &&
      selection.subject.coder_preparation !== selection.subject.reviewer_preparation &&
      Number.isSafeInteger(selection.connection_generation) &&
      selection.connection_generation > 0
  );
  const base = 'setup/' + selection.subject.project_id;
  let adoption: SetupAdoption;
  try {
    adoption = await inspect(selection, runtime);
  } catch {
    adoption = {
      format: 1,
      state: 'blocked',
      blockers: ['setup_observation_unavailable'],
      project: null,
      connection: null,
      preparations: [],
      continuation_sent: false,
    };
  }
  await runtime.save(base + '/adoption.json', adoption);
  if (adoption.state === 'ready') return adoption;
  const intent = continuationIntent(selection, adoption);
  if (!intent) return adoption;
  // This exclusive fsync-backed write must finish before the transport sees the
  // effect. An existing intent is reconciled by the caller, never overwritten.
  await runtime.save(base + '/continuation.request.json', intent);
  let replied = true;
  try {
    await runtime.continuePreparation(intent);
  } catch {
    replied = false;
  }
  let after: SetupAdoption;
  try {
    // Read the exact operation first, even if another normal API has become
    // unavailable meanwhile. This endpoint projects retained preparation state.
    const preparation = object(await runtime.get('/api/environments/' + selection.subject.project_id + '/preparation'));
    after = {...(await inspect(selection, runtime, preparation)), continuation_sent: true};
    if (after.state !== 'ready') after = {...after, state: 'pending'};
  } catch {
    after = {...adoption, state: 'pending', blockers: ['preparation_outcome_unconfirmed'], continuation_sent: true};
  }
  await runtime.save(base + '/continuation.observed.json', {...after, response_received: replied});
  return after;
}
