// Semantic admission only. Rust invokes the existing protected collector after
// verifying these exact file bindings; this module never launches product work.
import assert from 'node:assert/strict';
import {implementedRoute} from './route';
import {lstat, open, readFile} from 'node:fs/promises';
import path from 'node:path';
import {sha256, publicBytes} from './validate';
import type {PublicArtifact, QualificationScenario, ResolvedQualificationInput} from './scenario';
import type {Readiness} from './readiness';
import {checkScenario} from './readiness';

/** Approval covers effects and finite paid scope. Tool rebuilds, local file
 * locations, returned credential generations and a user hold are not revocation. */
export function approvalScope(s: QualificationScenario) {
  return {
    candidate: {
      source_revision: s.candidate.source_revision,
      candidate_sha256: s.candidate.candidate_sha256,
      payload_sha256: s.candidate.payload_sha256,
    },
    target: {distribution: s.target.distribution, name: s.target.name, uuid: s.target.uuid, origin: s.target.origin},
    cases: [...s.cases].sort(),
    managed: {
      project_ids: [...s.managed.project_ids].sort(),
      application_ids: [...s.managed.application_ids].sort(),
      terminal_ids: [...s.managed.terminal_ids].sort(),
    },
    factory: s.factory
      .map((f) => ({...f, subject: {...f.subject, required_checks: [...f.subject.required_checks].sort()}}))
      .sort((a, b) =>
        (a.subject.repository_id + '/' + a.subject.issue_number).localeCompare(
          b.subject.repository_id + '/' + b.subject.issue_number
        )
      ),
    // Omission/empty App selection preserves an unchanged Factory-only scope.
    ...(s.applications?.length
      ? {
          applications: s.applications
            .map((a) => ({
              application_id: a.application_id,
              fixture_sha256: a.fixture.sha256,
              record: a.record,
            }))
            .sort((a, b) => a.application_id.localeCompare(b.application_id)),
        }
      : {}),
    permitted_effects: {
      ...s.permitted_effects,
      project_lifecycle: [...s.permitted_effects.project_lifecycle].sort(),
      application_lifecycle: [...s.permitted_effects.application_lifecycle].sort(),
      terminal_lifecycle: [...s.permitted_effects.terminal_lifecycle].sort(),
      factory_repositories: [...s.permitted_effects.factory_repositories].sort(),
      current_context_phases: [...s.permitted_effects.current_context_phases].sort(),
    },
  };
}
export interface ScopeReceipt {
  readonly format: 1;
  readonly state: 'approved' | 'revoked';
  readonly approval_reference: string;
  readonly scope: ReturnType<typeof approvalScope>;
}
export interface RunAdmission {
  readonly format: 1;
  readonly state: 'held' | 'revoked' | 'blocked' | 'ready';
  readonly held: boolean;
  readonly approval_state: 'missing' | 'unvalidated' | 'approved' | 'revoked';
  readonly blockers: readonly string[];
  readonly approval: PublicArtifact | null;
  readonly build_receipt: PublicArtifact | null;
  readonly collector_sha256: string | null;
  readonly collection: QualificationScenario['collection'] | null;
  readonly resolved_input: PublicArtifact | null;
}
/** An admission is an immutable result, never permission to retry its effects. */
export async function retainedAdmission(run: string): Promise<RunAdmission | null> {
  const file = path.join(run, 'run-admission.json');
  try {
    const stat = await lstat(file);
    assert(stat.isFile() && stat.nlink === 1 && (stat.mode & 0o222) === 0 && stat.size <= 1 << 20);
    const value = JSON.parse(await readFile(file, 'utf8')) as RunAdmission;
    assert(
      value.format === 1 &&
        ['held', 'revoked', 'blocked', 'ready'].includes(value.state) &&
        typeof value.held === 'boolean' &&
        ['missing', 'unvalidated', 'approved', 'revoked'].includes(value.approval_state) &&
        Array.isArray(value.blockers) &&
        value.blockers.every((b) => typeof b === 'string')
    );
    return value;
  } catch (error) {
    if ((error as NodeJS.ErrnoException).code === 'ENOENT') return null;
    throw error;
  }
}
export async function interruptedAdmission(run: string): Promise<boolean> {
  for (const name of ['run-readiness.json', 'resolved-input.json']) {
    try {
      await lstat(path.join(run, name));
      return true;
    } catch (error) {
      if ((error as NodeJS.ErrnoException).code !== 'ENOENT') throw error;
    }
  }
  return false;
}

/** The only unready setup admitted is continuation of the two exact waiting
 * preparations. The protected normal-owner helper still validates its pinned
 * setup inputs and must observe ready before any Factory command. */
export function continuablePreparations(s: QualificationScenario, r: Readiness, index: number): boolean {
  if (!s.permitted_effects.fixture_setup) return false;
  const subject = s.factory[index]?.subject;
  if (!subject) return false;
  const native = r.native as {factory_readiness?: any[]} | null;
  const state = native?.factory_readiness?.find(
    (row) => row.repository_id === subject.repository_id && row.issue_number === subject.issue_number
  );
  if (!Array.isArray(state?.preparations)) return false;
  const selected = [
    [subject.coder_preparation, 'soda-coder'],
    [subject.reviewer_preparation, 'soda-reviewer'],
  ].map(([id, role]) => state.preparations.filter((p: any) => p.id === id && p.role === role));
  if (!selected.every((rows) => rows.length === 1)) return false;
  const preparations = selected.map((rows) => rows[0]);
  return (
    preparations.some((p) => p.phase === 'approved' || p.phase === 'waiting') &&
    new Set(preparations.map((p) => p.setup_digest)).size === 1 &&
    preparations.every(
      (p) =>
        p.source === subject.source_commit &&
        p.requirements === subject.requirements &&
        p.approval === subject.approval &&
        /^[a-f0-9]{64}$/.test(p.setup_digest) &&
        p.stopped === false &&
        (p.phase === 'ready'
          ? p.ready === true && p.setup_exit === 0 && p.check_exit === 0
          : ['approved', 'waiting'].includes(p.phase) && p.ready === false)
    )
  );
}
export async function publicReference(file: string): Promise<PublicArtifact> {
  const stat = await lstat(file);
  assert(stat.isFile() && stat.nlink === 1 && stat.size <= 16 << 20 && path.resolve(file) === file);
  return {path: file, sha256: sha256(await readFile(file))};
}
export async function scopeApproval(s: QualificationScenario) {
  if (s.authority.scope_receipt_file === null) return {state: 'missing' as const, receipt: null};
  try {
    const reference = await publicReference(s.authority.scope_receipt_file);
    const receipt = JSON.parse((await publicBytes(reference)).toString()) as ScopeReceipt;
    assert.deepEqual(Object.keys(receipt).sort(), ['approval_reference', 'format', 'scope', 'state']);
    assert(
      receipt.format === 1 && typeof receipt.approval_reference === 'string' && receipt.approval_reference.length > 0
    );
    assert(receipt.state === 'approved' || receipt.state === 'revoked');
    assert.deepEqual(receipt.scope, approvalScope(s));
    return {state: receipt.state, receipt: reference};
  } catch {
    return {state: 'unvalidated' as const, receipt: null};
  }
}
async function writeNew(file: string, value: unknown, mode = 0o444) {
  const handle = await open(file, 'wx', mode);
  try {
    await handle.writeFile(JSON.stringify(value, null, 2) + '\n');
    await handle.sync();
  } finally {
    await handle.close();
  }
  const directory = await open(path.dirname(file), 'r');
  try {
    await directory.sync();
  } finally {
    await directory.close();
  }
}

export async function admitRun(s: QualificationScenario, saved: Readiness, run: string): Promise<RunAdmission> {
  const retained = await retainedAdmission(run);
  if (retained) return retained;
  assert(!(await interruptedAdmission(run)), 'admission_interrupted_preserve_run');
  const approval = await scopeApproval(s);
  const blockers: string[] = [];
  if (approval.state !== 'approved') blockers.push('paid_authority_' + approval.state);
  if (s.authority.held) blockers.push('user_hold');
  if (!s.collection) blockers.push('collector_inputs_missing');
  // Holds and unavailable authority stop here, without remote reads, credential
  // reads, phase-file edits, funding or enrollment. The approval remains separate.
  let readiness = saved;
  if (!s.authority.held && approval.state === 'approved') {
    readiness = await checkScenario(s);
    await writeNew(path.join(run, 'run-readiness.json'), readiness);
  }
  blockers.push(
    ...readiness.checks
      .filter((c) => {
        if (c.state === 'pass' || ['paid_authority', 'execution_released'].includes(c.name)) return false;
        const prep = /^factory_([0-9]+)_standing_preparations$/.exec(c.name);
        return !(prep && continuablePreparations(s, readiness, Number(prep[1])));
      })
      .map((c) => c.name)
  );
  if (!implementedRoute(s)) blockers.push('collector_route_unimplemented');
  let build: PublicArtifact | null = null,
    collectorSHA: string | null = null;
  if (blockers.length === 0 && s.collection) {
    try {
      build = await publicReference(s.collection.build_receipt_file);
      const receipt = JSON.parse((await publicBytes(build)).toString());
      assert(
        receipt.format === 1 && receipt.source_clean === true && receipt.source_revision === s.driver.source_revision
      );
      assert(Array.isArray(receipt.outputs));
      const drivers = receipt.outputs.filter((o: any) => path.basename(o.path) === 'soda-qualification-journey');
      const collectors = receipt.outputs.filter((o: any) => path.basename(o.path) === 'soda-qualification-collector');
      assert(drivers.length === 1 && drivers[0].sha256 === s.driver.executable.sha256 && collectors.length === 1);
      assert(/^[a-f0-9]{64}$/.test(collectors[0].sha256));
      collectorSHA = collectors[0].sha256;
    } catch {
      blockers.push('collector_build_unbound');
    }
  }
  const state = s.authority.held
    ? 'held'
    : approval.state === 'revoked'
      ? 'revoked'
      : blockers.length
        ? 'blocked'
        : 'ready';
  let resolved: PublicArtifact | null = null;
  if (state === 'ready') {
    assert(readiness.observation && approval.receipt && build);
    const input: ResolvedQualificationInput = {
      format: 1,
      scenario: s,
      observation: readiness.observation,
      admission: {approval: approval.receipt, build_receipt: build},
      prior_attempts: s.collection?.prior_attempts ?? [],
    };
    const file = path.join(run, 'resolved-input.json');
    await writeNew(file, input, 0o600);
    resolved = await publicReference(file);
  }
  const admission: RunAdmission = {
    format: 1,
    state,
    held: s.authority.held,
    approval_state: approval.state,
    blockers: [...new Set(blockers)],
    approval: approval.receipt,
    build_receipt: build,
    collector_sha256: collectorSHA,
    collection: s.collection ?? null,
    resolved_input: resolved,
  };
  await writeNew(path.join(run, 'run-admission.json'), admission);
  return admission;
}
