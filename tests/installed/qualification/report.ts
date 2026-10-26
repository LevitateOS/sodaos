// Receipt projection only: no transport, native reconciliation, credential read,
// finalization, qualification decision or effect dispatch belongs in this view.
import assert from 'node:assert/strict';
import {lstat, open, realpath} from 'node:fs/promises';
import {constants} from 'node:fs';
import path from 'node:path';
import {scenarioInput, sha256} from './validate';

type Row = Record<string, unknown>;
const row = (v: unknown): Row => (v !== null && typeof v === 'object' && !Array.isArray(v) ? (v as Row) : {});
const list = (v: unknown): unknown[] => (Array.isArray(v) && v.length <= 1024 ? v : []);
const text = (v: unknown, pattern = /^[A-Za-z0-9_.:/ -]{1,256}$/): string | null =>
  typeof v === 'string' && pattern.test(v) ? v : null;
const code = (v: unknown) => text(v, /^[a-z0-9_]{1,128}$/);
const digest = (v: unknown) => text(v, /^[a-f0-9]{64}$/);
const integer = (v: unknown): number | null => (typeof v === 'number' && Number.isSafeInteger(v) && v >= 0 ? v : null);
const bool = (v: unknown): boolean | null => (typeof v === 'boolean' ? v : null);
const date = (v: unknown) => {
  const value = text(v, /^\d{4}-\d\d-\d\dT\d\d:\d\d:\d\d(?:\.\d{1,6})?(?:Z|[+-]\d\d:\d\d)$/);
  return value !== null && Number.isFinite(Date.parse(value)) ? value : null;
};
const micros = (v: unknown) => text(v, /^(0|[1-9][0-9]{0,18})$/);
const kinds = ['installation', 'browser_project', 'factory', 'persistence'] as const;
interface Document {
  value: Row;
  path: string;
  sha256: string;
  bytes: number;
}

/** Fixed receipt files only. References never authorize reading an arbitrary
 * path. No directory walk or discovery of credentials/provider payloads. */
export async function journeyReport(run: string) {
  assert(run.startsWith('/home/') && path.resolve(run) === run && (await realpath(run)) === run);
  const sources: {role: string; path: string; sha256: string; bytes: number}[] = [];
  let total = 0;
  async function document(file: string, role: string, optional = false): Promise<Document | null> {
    try {
      const stat = await lstat(file);
      assert(stat.isFile() && !stat.isSymbolicLink() && stat.nlink === 1 && stat.size > 0 && stat.size <= 16 << 20);
      assert(file.startsWith('/home/') && path.resolve(file) === file && (await realpath(file)) === file);
      const handle = await open(file, constants.O_RDONLY | constants.O_NOFOLLOW);
      let bytes: Buffer;
      try {
        const actual = await handle.stat();
        assert(actual.dev === stat.dev && actual.ino === stat.ino && actual.size === stat.size);
        bytes = await handle.readFile();
        assert(bytes.length === stat.size);
      } finally {
        await handle.close();
      }
      total += bytes.length;
      assert(total <= 64 << 20 && sources.length < 1024);
      const parsed: unknown = JSON.parse(bytes.toString('utf8'));
      assert(parsed !== null && typeof parsed === 'object' && !Array.isArray(parsed));
      const result = {value: row(parsed), path: file, sha256: sha256(bytes), bytes: bytes.length};
      sources.push({role, path: file, sha256: result.sha256, bytes: result.bytes});
      return result;
    } catch (error) {
      if (optional && (error as NodeJS.ErrnoException).code === 'ENOENT') return null;
      throw error;
    }
  }
  function bound(reference: unknown, actual: Document) {
    const ref = row(reference);
    assert(ref.path === actual.path && ref.sha256 === actual.sha256, 'report_receipt_binding_changed');
  }
  const savedScenario = (await document(path.join(run, 'scenario.json'), 'scenario'))!;
  const scenario = scenarioInput(savedScenario.value);
  const journeyRequest = await document(path.join(run, 'request.json'), 'journey_request', true);
  if (journeyRequest) {
    assert(journeyRequest.value.format === 1);
    bound(journeyRequest.value.scenario, savedScenario);
    assert.deepEqual(journeyRequest.value.driver, scenario.driver.executable);
  }
  const readiness =
    (await document(path.join(run, 'run-readiness.json'), 'run_readiness', true)) ??
    (await document(path.join(run, 'readiness.json'), 'readiness'))!;
  const ready = readiness.value;
  assert(ready.format === 1 && typeof ready.ready === 'boolean' && Array.isArray(ready.checks));
  const checks = list(ready.checks).map((v) => {
    const c = row(v);
    assert(code(c.name) && ['pass', 'blocked', 'unknown'].includes(String(c.state)));
    return {name: c.name as string, state: c.state as string};
  });
  const native = row(ready.native),
    history = row(native.history),
    observation = row(ready.observation);
  const factory = scenario.factory.map((f) => {
    const state = list(native.factory_readiness)
      .map(row)
      .find((r) => r.repository_id === f.subject.repository_id && r.issue_number === f.subject.issue_number);
    const accounting = row(state?.accounting);
    const observed = list(observation.factory)
      .map(row)
      .find((r) => r.repository_id === f.subject.repository_id && r.issue_number === f.subject.issue_number);
    const connection = row(observed?.connection);
    return {
      repository_id: f.subject.repository_id,
      issue_number: f.subject.issue_number,
      connection_id: f.subject.connection_id,
      generation: connection.id === f.subject.connection_id ? integer(connection.generation) : null,
      connection_state: connection.id === f.subject.connection_id ? code(connection.state) : null,
      requested: f.limits,
      ...(f.execution
        ? {
            requested_execution: f.execution,
            observed_provider: code(state?.connection_provider),
            observed_roles: ['soda-coder', 'soda-reviewer'].map((role) => {
              const selection = row(row(state?.roles)[role]),
                profile = row(selection.profile);
              return {
                role,
                harness: code(selection.harness),
                model: text(selection.model, /^[A-Za-z0-9][A-Za-z0-9._:/-]{0,127}$/),
                profile: {
                  effort: code(profile.effort),
                  contributor: profile.contributor === undefined ? false : bool(profile.contributor),
                },
              };
            }),
          }
        : {}),
      observed: {
        sponsor_minutes: integer(accounting.sponsor_minutes),
        sponsor_active: bool(accounting.sponsor_active),
        rolling_limit_minutes: integer(accounting.rolling_limit_minutes),
        repository_charged_minutes: integer(accounting.repository_charged_minutes),
        held_reservations: integer(accounting.held_reservations),
        rolling_settled_micros: micros(accounting.rolling_settled_micros),
      },
    };
  });
  function closure(value: unknown) {
    const c = row(value);
    return {
      scope_ready: bool(c.scope_ready),
      snapshot_ready: bool(c.snapshot_ready),
      resources: list(c.resources).map((v) => {
        const r = row(v);
        return {
          kind: code(r.kind),
          id: text(r.id),
          state: code(r.state),
          managed: bool(r.managed),
          permitted: bool(r.permitted),
        };
      }),
      blockers: list(c.blockers).map((v) => {
        const b = row(v);
        return {code: code(b.code), kind: code(b.kind), id: text(b.id)};
      }),
    };
  }
  const admission = await document(path.join(run, 'run-admission.json'), 'admission', true);
  if (admission)
    assert(
      admission.value.format === 1 &&
        ['held', 'revoked', 'blocked', 'ready'].includes(String(admission.value.state)) &&
        typeof admission.value.held === 'boolean'
    );
  const intent = await document(path.join(run, 'collector-intent.json'), 'collector_intent', true);
  const binding = await document(path.join(run, 'collector-binding.json'), 'collector_binding', true);
  assert(!binding || intent, 'report_binding_without_intent');
  let collection: Row | null = null;
  const phases: Row[] = [];
  let supplemental: Row | null = null,
    resources: Row | null = null,
    failure: Row | null = null;
  if (intent) {
    assert(
      intent.value.format === 1 &&
        admission?.value.state === 'ready' &&
        admission.value.held === false &&
        admission.value.approval_state === 'approved' &&
        scenario.authority.held === false
    );
    bound(intent.value.scenario, savedScenario);
    // Rust's retained Collection serializes the optional empty history vector.
    const collectionScope = (v: unknown) => ({...row(v), prior_attempts: row(v).prior_attempts ?? []});
    assert.deepEqual(collectionScope(intent.value.collection), collectionScope(admission.value.collection));
    assert.deepEqual(collectionScope(intent.value.collection), collectionScope(scenario.collection));
    assert.deepEqual(intent.value.resolved_input, admission.value.resolved_input);
    assert.deepEqual(intent.value.approval, admission.value.approval);
    assert.deepEqual(intent.value.build_receipt, admission.value.build_receipt);
    assert(row(intent.value.approval).path === scenario.authority.scope_receipt_file);
    assert(intent.value.collector_sha256 === admission.value.collector_sha256 && digest(intent.value.collector_sha256));
    const resolved = (await document(path.join(run, 'resolved-input.json'), 'resolved_input'))!;
    bound(intent.value.resolved_input, resolved);
    assert(resolved.value.format === 1);
    assert.deepEqual(resolved.value.scenario, savedScenario.value);
    assert.deepEqual(row(resolved.value.admission).approval, intent.value.approval);
    assert.deepEqual(row(resolved.value.admission).build_receipt, intent.value.build_receipt);
    const output = row(intent.value.collection).output_directory;
    assert(typeof output === 'string' && output.startsWith('/home/') && path.resolve(output) === output);
    const request = await document(path.join(output, 'request.json'), 'collector_request', true);
    if (binding) {
      assert(binding.value.format === 1 && request);
      bound(binding.value.intent, intent);
      bound(row(binding.value.collection).request, request);
      assert(row(binding.value.collection).unit === request.value.unit);
    }
    if (!request) {
      collection = {state: 'intent_without_request', native_retirement: null, finalized: null};
    } else {
      const snapshot = (await document(path.join(output, 'resolved-input.json'), 'collector_input'))!;
      bound(request.value.resolved_input, snapshot);
      assert(snapshot.sha256 === resolved.sha256);
      const requested = row(request.value.binding);
      assert(
        requested.format === 1 &&
          request.value.attempt === path.join(output, 'attempt') &&
          /^soda-qualify-[a-f0-9]{32}\.service$/.test(String(request.value.unit)) &&
          requested.unit === request.value.unit &&
          requested.source_revision === scenario.candidate.source_revision &&
          requested.candidate_sha256 === scenario.candidate.candidate_sha256 &&
          requested.payload_sha256 === scenario.candidate.payload_sha256 &&
          requested.collector_sha256 === intent.value.collector_sha256 &&
          requested.target === scenario.target.name
      );
      const executed = await document(path.join(output, 'executed.json'), 'collector_execution', true);
      const seal = await document(path.join(output, 'observations/release-evidence.json'), 'release_evidence', true);
      assert(!seal || executed, 'report_seal_without_execution');
      const receipt = executed?.value;
      if (receipt) {
        for (const key of Object.keys(requested).filter(
          (key) => !['files', 'exit_code', 'native_retirement'].includes(key)
        ))
          assert.deepEqual(receipt[key], requested[key], 'report_execution_binding_changed');
        assert(receipt.exit_code === 0 && receipt.native_retirement === 'confirmed');
      }
      if (seal) assert.deepEqual(seal.value, receipt, 'report_seal_binding_changed');
      // Unfinished attempts may expose selected evidence, but it is explicitly
      // unsealed. A root completion receipt provides exact identities for each
      // projected file; this report does not replace the canonical validator.
      const evidenceRoot = path.join(output, executed ? 'observations' : 'attempt');
      async function evidence(relative: string, role: string) {
        const doc = await document(path.join(evidenceRoot, relative), role, true);
        if (doc && receipt) {
          const identity = row(row(receipt.files)[relative]);
          assert(identity.SHA256 === doc.sha256 && identity.Bytes === doc.bytes, 'report_evidence_binding_changed');
        }
        return doc?.value ?? null;
      }
      collection = {
        state: seal ? 'finalized_receipt' : executed ? 'executed_receipt' : 'request_retained',
        unit: request.value.unit,
        native_retirement: receipt ? 'confirmed_at_execution' : null,
        finalized: !!seal,
        evidence_binding: receipt ? 'projected_files_match_execution_receipt' : 'unsealed_saved_files',
      };
      // Controller scratch is deliberately outside the sealed product evidence.
      // Its own fixed receipts bind the exact request and original work inode;
      // absent data or a terminal worker never implies deletion.
      const preserved = await document(path.join(output, 'work-preserved.json'), 'work_preserved', true);
      const scratchIntent = await document(
        path.join(output, 'scratch-retirement-intent.json'),
        'scratch_retirement_intent',
        true
      );
      const scratchResult = await document(path.join(output, 'scratch-retirement.json'), 'scratch_retirement', true);
      if (preserved || scratchIntent || scratchResult) {
        const origin = (await document(path.join(output, 'work-origin.json'), 'work_origin'))!;
        for (const doc of [preserved, scratchIntent, scratchResult].filter((v) => v !== null))
          assert(
            doc.value.format === 1 &&
              doc.value.resource === 'collector-scratch' &&
              doc.value.request_sha256 === request.sha256 &&
              doc.value.origin_sha256 === origin.sha256 &&
              doc.value.path === path.join(output, 'consumed-work') &&
              doc.value.automatic_retry === false
          );
        if (scratchIntent)
          assert(scratchIntent.value.unit === request.value.unit && scratchIntent.value.state === 'retained');
        if (scratchResult) {
          assert(scratchIntent && ['retained', 'removed'].includes(String(scratchResult.value.state)));
          for (const key of Object.keys(scratchIntent.value).filter((key) => key !== 'state'))
            assert.deepEqual(scratchResult.value[key], scratchIntent.value[key], 'report_scratch_binding_changed');
        }
        const selected = scratchResult?.value ?? scratchIntent?.value;
        collection.scratch_retirement = {
          state: scratchResult ? code(scratchResult.value.state) : scratchIntent ? 'uncertain' : 'retained',
          intent_recorded: scratchIntent !== null,
          result_recorded: scratchResult !== null,
          outside_sealed_evidence: bool(preserved?.value.outside_sealed_evidence),
          entries: integer(selected?.entries),
          selected_bytes: integer(selected?.selected_bytes),
          selected_allocated_bytes: integer(selected?.selected_allocated_bytes),
          completed_evidence: bool(selected?.completed_evidence),
          worker_terminal: bool(selected?.worker_terminal),
          scope_permitted: bool(selected?.scope_permitted),
          filesystem_free_before_bytes: integer(selected?.filesystem_free_before_bytes),
          filesystem_free_after_bytes: integer(scratchResult?.value.filesystem_free_after_bytes),
          automatic_retry: false,
        };
      }
      for (const kind of kinds) {
        const observed = await evidence(kind + '/observation.json', kind + '_observation');
        if (observed)
          assert(
            observed.RequestedRevision === scenario.candidate.source_revision &&
              observed.Target === scenario.target.name &&
              observed.ToolRevision === scenario.driver.source_revision
          );
        phases.push({
          kind,
          saved_observation: observed !== null,
          outcome: code(observed?.Outcome),
          execution: code(observed?.Execution),
          evidence: code(observed?.Evidence),
          exit_code: integer(observed?.ExitCode),
          started: date(observed?.Started),
          finished: date(observed?.Finished),
        });
      }
      const f =
        (await evidence('factory/result.json', 'factory_result')) ??
        (await evidence('factory/incomplete.json', 'factory_incomplete'));
      if (f)
        collection.factory = {
          observed: bool(f.observed),
          reason: code(f.reason),
          paused: bool(f.paused),
          dispatch_open: bool(f.dispatch_open),
          native_idle: bool(f.native_idle),
          scope_runs_retired: bool(f.scope_runs_retired),
          no_retry: bool(f.noRetry),
          no_budget_increase: bool(f.noBudgetIncrease),
        };
      const after = await evidence('factory/after.json', 'factory_after');
      if (after) {
        const funding = row(after.funding),
          global = row(after.global);
        collection.accounting_at_factory_completion = {
          at: date(after.at),
          sponsor_minutes: integer(funding.sponsor_minutes),
          rolling_minutes: integer(funding.rolling_minutes),
          repository_used_minutes: integer(funding.repository_used_minutes),
          rolling_used_micros: integer(funding.rolling_used_micros),
          held_reservations: integer(global.held),
          unsettled_runs: integer(global.unsettled),
          leases: integer(global.leases),
          nonterminal: integer(global.nonterminal),
        };
      }
      const plan = await evidence('persistence/retirement-plan.json', 'retirement_plan');
      if (plan) {
        const outcomes: Row[] = [];
        for (const value of list(plan.retirement_actions)) {
          const action = row(value),
            kind = code(action.kind),
            id = text(action.id, /^(?:p[a-f0-9]{24}|a[a-f0-9]{24}|[a-f0-9]{32})$/);
          assert(id && kind && ['project', 'application', 'terminal'].includes(kind));
          const name = (kind === 'terminal' ? 'end-terminal-' : 'stop-' + kind + '-') + id;
          const result = await evidence('persistence/' + name + '.native.json', 'resource_retirement');
          outcomes.push({
            kind,
            id,
            saved_confirmation: result !== null,
            http_status: integer(result?.http_status),
            closure: result ? closure(result.closure) : null,
          });
        }
        resources = {plan: closure(plan), outcomes};
      }
      const measured = await evidence('supplemental/measurements/result.json', 'measurements');
      const recovery = await evidence('supplemental/recovery/result.json', 'final_recovery');
      const clonePlan = await evidence('supplemental/recovery/retirement.plan.json', 'clone_retirement_plan');
      const cloneIntent = await evidence('supplemental/recovery/retirement.intent.json', 'clone_retirement_intent');
      const cloneResult = await evidence('supplemental/recovery/retirement.json', 'clone_retirement');
      const cloneUncertain = await evidence(
        'supplemental/recovery/retirement.uncertain.json',
        'clone_retirement_uncertain'
      );
      let cloneRetirement: Row | null = null;
      if (clonePlan || cloneIntent || cloneResult || cloneUncertain) {
        const restored = await evidence('supplemental/recovery/restore.json', 'cold_restore');
        assert(recovery && restored);
        const recoverySHA = sources.find((source) => source.role === 'final_recovery')!.sha256;
        for (const doc of [clonePlan, cloneIntent, cloneResult, cloneUncertain].filter((v) => v !== null))
          assert(
            doc.container === row(recovery.cold).container &&
              doc.restored === restored.directory &&
              doc.result_sha256 === recoverySHA &&
              doc.automatic_retry === false
          );
        if (clonePlan) assert(clonePlan.action === 'isolated-cold-retirement-plan');
        if (cloneResult) assert(cloneResult.action === 'isolated-cold-retirement' && cloneResult.complete === true);
        cloneRetirement = {
          state: cloneResult ? 'removed_receipt' : cloneIntent || cloneUncertain ? 'uncertain' : 'retained',
          plan_recorded: clonePlan !== null,
          intent_recorded: cloneIntent !== null,
          result_recorded: cloneResult !== null,
          uncertainty_recorded: cloneUncertain !== null,
          selected_allocated_bytes: integer((cloneResult ?? clonePlan)?.selected_allocated_bytes),
          filesystem_available_before_bytes: integer(cloneResult?.filesystem_available_before_bytes),
          filesystem_available_after_bytes: integer(cloneResult?.filesystem_available_after_bytes),
          observed_available_delta_bytes:
            typeof cloneResult?.observed_available_delta_bytes === 'number' &&
            Number.isSafeInteger(cloneResult.observed_available_delta_bytes)
              ? cloneResult.observed_available_delta_bytes
              : null,
          source_preserved: bool(cloneResult?.source_preserved),
          private_settings_preserved: bool(cloneResult?.private_settings_preserved),
          provenance_preserved: bool(cloneResult?.provenance_preserved),
          automatic_retry: false,
        };
      }
      if (measured || recovery) {
        const cold = row(recovery?.cold),
          application = row(recovery?.application),
          record = row(application.record);
        supplemental = {
          measurements: measured
            ? {
                observed: bool(measured.observed),
                native_samples: integer(measured.native_samples),
                response_samples: integer(measured.response_samples),
                started: date(measured.started),
                finished: date(measured.finished),
              }
            : null,
          recovery: recovery
            ? {
                record_sha256: digest(record.record_sha256),
                primary_rendered_readback:
                  row(application.primary).status === 200 && row(application.primary).record_id === record.id,
                isolated_rendered_readback:
                  row(application.isolated).status === 200 && row(application.isolated).record_id === record.id,
                same_protected_state: bool(cold.same_protected_state),
                same_project_witness: bool(cold.same_project_witness),
                same_factory_history: bool(cold.same_factory_history),
                seal_sha256: digest(cold.seal_sha256),
              }
            : null,
          capacity: null,
          clone_retirement: cloneRetirement,
          production_qualification: false,
        };
      }
      const failed = await evidence('failure.json', 'collector_failure');
      if (failed) failure = {reason: code(failed.reason), phase: code(failed.phase), automatic_retry: false};
    }
  }
  return {
    format: 1,
    scope: 'saved receipt report; no execution, refresh, reconciliation or qualification decision',
    candidate: {
      source_revision: scenario.candidate.source_revision,
      candidate_sha256: scenario.candidate.candidate_sha256,
      payload_sha256: scenario.candidate.payload_sha256,
    },
    target: {distribution: scenario.target.distribution, name: scenario.target.name, uuid: scenario.target.uuid},
    cases: scenario.cases,
    readiness: {checked_at: date(ready.checked_at), ready: ready.ready, checks, closure: closure(ready.closure)},
    admission: admission
      ? {
          state: code(admission.value.state),
          held: bool(admission.value.held),
          approval_state: code(admission.value.approval_state),
          blockers: list(admission.value.blockers).map(code),
        }
      : null,
    accounting_at_readiness: {
      retained_runs: integer(history.runs),
      charged_minutes: integer(history.charged_minutes),
      factory,
      provider_token_billing: null,
    },
    collection,
    phases,
    supplemental,
    resource_retirement: resources,
    failure,
    qualification: null,
    sources,
    automatic_actions: false,
  };
}
