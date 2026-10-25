// Semantic admission only. Rust invokes the existing protected collector after
// verifying these exact file bindings; this module never launches product work.
import assert from 'node:assert/strict';
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

export async function admitRun(s: QualificationScenario, saved: Readiness, run: string) {
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
      .filter((c) => c.state !== 'pass' && !['paid_authority', 'execution_released'].includes(c.name))
      .map((c) => c.name)
  );
  if (s.target.distribution !== 'docker' || s.cases.join(',') !== 'installation,browser_project,factory,persistence')
    blockers.push('collector_route_unimplemented');
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
  const admission = {
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
