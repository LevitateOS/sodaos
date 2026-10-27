import assert from 'node:assert/strict';
import {check, sha, SHA, COMMIT} from './support';
import {validateFactoryPhase, snapshotScript, factoryCommandID} from './factory-native';
import {nativeExecutionScript, nativeExecutionSucceeded} from './factory-execution';
import {ciScript} from './factory-ci';
import {contextStatusScript} from './context-status';
import {issueFactoryCommand} from './factory-command';
import {mountedSpacesFacts, nativeSpacesMount} from '../../sodaspaces-journey-evidence';

let stage = 'not_started';
export function factoryDiagnostic(error: any) {
  const errorName = ['Error', 'TypeError', 'AssertionError', 'TimeoutError', 'SyntaxError', 'ProtocolError'].includes(
    error?.name
  )
    ? error.name
    : 'Error';
  const match =
    typeof error?.stack === 'string'
      ? error.stack.match(
          /\/(factory-phase\.ts|factory-native\.ts|context-status\.ts|docker-collector\.ts):(\d+):(\d+)/
        )
      : null;
  return {
    stage,
    errorName,
    sourceLocation: match ? {file: match[1], line: Number(match[2]), column: Number(match[3])} : null,
  };
}
export interface FactoryIO {
  save(relative: string, value: any): Promise<any>;
  saveBytes(relative: string, value: Uint8Array): Promise<any>;
  sshRead(fixedScript: string, timeoutMs?: number): string | Promise<string>;
  observeNative?(snapshot: unknown): Promise<void>;
}

export class FactoryPermits {
  private armed: {label: string; method: string; path: string; body: any} | undefined;
  readonly used: string[] = [];
  readonly usedRunActionPaths = new Set<string>();
  async admitRunAction(request: any) {
    const u = new URL(request.url());
    if (request.method() !== 'POST' || !/^\/.*factory\/runs\/[a-f0-9]{32}\/actions$/.test(u.pathname)) return false;
    return this.admit(request);
  }
  isRunActionResponse(response: any) {
    return response.request().method() === 'POST' && this.usedRunActionPaths.has(new URL(response.url()).pathname);
  }
  constructor(
    private origin: string,
    private repository: number,
    private save: FactoryIO['save']
  ) {}
  async arm(label: string, method: string, path: string, body: any) {
    check(!this.armed && !this.used.includes(label), 'factory_mutation_already_issued');
    const ordinary =
      (method === 'PUT' && path.endsWith('/repositories/' + this.repository + '/factory/policy')) ||
      (method === 'POST' && path.endsWith('/repositories/' + this.repository + '/factory/actions'));
    const runPath = path.match(/\/factory\/runs\/([a-f0-9]{32})\/actions$/);
    const runAction =
      method === 'POST' &&
      runPath !== null &&
      label === 'use-current-context-' + runPath[1] &&
      body.action === 'use_current_context' &&
      Object.keys(body).sort().join(',') === 'action,command_id,context' &&
      body.context &&
      Object.keys(body.context).sort().join(',') === 'fingerprint,required_digest,scope_digest' &&
      [body.context.fingerprint, body.context.scope_digest, body.context.required_digest].every((x) => SHA.test(x));
    check(ordinary || runAction, 'factory_mutation_scope');
    check(/^[a-f0-9]{32}$/.test(body.command_id), 'factory_command_identity');
    await this.save('commands/' + label + '.request.json', {method, path, body});
    this.armed = {label, method, path, body};
  }
  async admit(request: any) {
    const a = this.armed,
      u = new URL(request.url());
    if (!a || u.origin !== this.origin || u.pathname !== a.path || u.search || u.hash || request.method() !== a.method)
      return false;
    assert.deepEqual(request.postDataJSON(), a.body);
    this.armed = undefined;
    this.used.push(a.label);
    if (/\/factory\/runs\/[a-f0-9]{32}\/actions$/.test(a.path)) this.usedRunActionPaths.add(a.path);
    return true;
  }
}

export function nativeIdle(f: any) {
  return (
    f?.global?.leases === 0 &&
    f.global.nonterminal === 0 &&
    Array.isArray(f.global.native_units) &&
    f.global.native_units.length === 0
  );
}
export function fullyAccounted(f: any) {
  return nativeIdle(f) && f.global.unsettled === 0 && f.global.held === 0;
}

// ContextPending is a staging fact, not a typed judgment. Permit a normal
// asynchronous consultation to settle; a sustained unresolved stage holds this
// phase without reconstructing a choice or claiming that Attention occurred.
export function contextPendingHold(f: any, firstSeen: Map<string, number>, now: number) {
  const pending = new Set<string>(
    f.runs
      .filter(
        (r: any) =>
          r.context_pending === true &&
          r.active === true &&
          !r.reconciled &&
          !r.outcome &&
          f.assignments.some((a: any) => a.run === r.id && a.stage === 'assigned')
      )
      .map((r: any) => r.id)
  );
  for (const id of firstSeen.keys()) if (!pending.has(id)) firstSeen.delete(id);
  for (const id of pending) if (!firstSeen.has(id)) firstSeen.set(id, now);
  return nativeIdle(f) && [...firstSeen.values()].some((t) => now - t >= 120000);
}

export function validateFactoryBaseline(c: any, f: any) {
  validateFactoryPhase(c);
  check(fullyAccounted(f), 'factory_original_global_idle_required');
  for (const k of [
    'assignments',
    'runs',
    'allowances',
    'usage',
    'reservations',
    'publications',
    'merges',
    'completions',
  ])
    check(Array.isArray(f[k]) && f[k].length === 0, 'factory_issue_already_attempted');
  check(
    f.acceptance?.id === c.issue.acceptance &&
      f.acceptance.title_digest === c.issue.title_digest &&
      f.acceptance.content_digest === c.issue.content_digest,
    'factory_acceptance_changed'
  );
  check(
    f.native_issue?.id === c.issue.native_id &&
      Number(f.native_issue.index) === c.issue.number &&
      f.native_issue.repository === c.repository.id &&
      !f.native_issue.closed &&
      !f.native_issue.pull,
    'factory_native_issue_changed'
  );
  check(
    f.connection?.id === c.connection.id &&
      f.connection.owner_id === c.connection.owner_id &&
      f.connection.state === 'ready' &&
      f.connection.provider === (c.execution?.harness ?? 'codex') &&
      f.connection.generation === c.connection.generation,
    'factory_ready_connection_changed'
  );
  check(f.integrated_source === c.repository.source_commit, 'factory_integrated_source_changed');
  check(
    f.funding.sponsor_minutes === c.funding.sponsor_minutes && f.funding.rolling_minutes === c.funding.rolling_minutes,
    'factory_funding_changed'
  );
  const planned = c.limits.attempt_minutes + c.limits.intent_minutes;
  check(
    f.funding.sponsor_minutes - f.funding.repository_used_minutes >= planned &&
      f.funding.rolling_minutes * 60_000_000 - f.funding.rolling_used_micros >= planned * 60_000_000,
    'factory_unchanged_funding_insufficient'
  );
  for (const role of ['soda-coder', 'soda-reviewer']) {
    check(
      f.preparations.some(
        (p: any) =>
          p.role === role &&
          p.id === (role === 'soda-coder' ? c.coder_preparation : c.reviewer_preparation) &&
          !p.publication &&
          p.source === c.repository.source_commit &&
          p.requirements === c.requirements &&
          p.approval === c.approval &&
          p.ready === true &&
          p.stopped === false &&
          p.setup_exit === 0 &&
          p.check_exit === 0
      ),
      'factory_current_standing_unavailable'
    );
  }
}

// Build canonical NativeFactory only from the new scoped completed lifecycle.
// This function neither runs a model nor writes qualification/protected files.
export function deriveFactoryEvidence(c: any, before: any, after: any, ci: any[], observedNative: any[]) {
  validateFactoryBaseline(c, before);
  check(fullyAccounted(after), 'factory_final_accounting_pending');
  check(
    after.acceptance?.id === c.issue.acceptance &&
      after.acceptance.title_digest === c.issue.title_digest &&
      after.acceptance.content_digest === c.issue.content_digest,
    'factory_final_acceptance_changed'
  );
  check(
    after.native_issue?.id === c.issue.native_id &&
      Number(after.native_issue.index) === c.issue.number &&
      after.native_issue.repository === c.repository.id &&
      after.native_issue.closed === true &&
      !after.native_issue.pull,
    'factory_native_closure_unconfirmed'
  );
  check(
    after.funding.sponsor_minutes === before.funding.sponsor_minutes &&
      after.funding.rolling_minutes === before.funding.rolling_minutes,
    'factory_budget_terms_changed'
  );
  check(after.allowances.length === 1, 'factory_one_new_root_required');
  const allowance = after.allowances[0];
  check(
    allowance.closed === true &&
      allowance.active === false &&
      allowance.active_minutes > 0 &&
      allowance.active_minutes <= c.limits.attempt_minutes &&
      allowance.active_seconds > 0 &&
      allowance.active_seconds <= allowance.active_minutes * 60,
    'factory_finite_closed_root_required'
  );
  check(
    after.assignments.length > 0 &&
      after.assignments.length <= 64 &&
      after.assignments.every(
        (a: any) =>
          a.project === c.repository.project &&
          a.acceptance === c.issue.acceptance &&
          a.root === allowance.root &&
          a.stage === 'finished'
      ),
    'factory_assignment_scope_changed'
  );
  check(
    after.runs.length === after.assignments.length &&
      after.runs.every(
        (r: any) =>
          r.project === c.repository.project &&
          !r.active &&
          r.settled === true &&
          r.reconciled === true &&
          r.credential_returned === true
      ),
    'factory_runs_not_retired'
  );
  check(
    after.reservations.length === after.assignments.length &&
      after.reservations.every((v: any) => v.connection === c.connection.id && v.state !== 'held'),
    'factory_reservations_not_settled'
  );
  check(
    after.usage.length === after.runs.length &&
      after.usage.every(
        (u: any) =>
          u.connection === c.connection.id &&
          Number.isFinite(Date.parse(u.started_at)) &&
          Date.parse(u.ended_at) > Date.parse(u.started_at)
      ),
    'factory_actual_usage_missing'
  );
  check(after.completions.length === 1, 'factory_canonical_delivery_missing');
  const completion = after.completions[0],
    merge = after.merges.find((m: any) => m.publication === completion.publication),
    publication = after.publications.find((p: any) => p.id === completion.publication);
  check(
    completion.acceptance === c.issue.acceptance &&
      /^[a-f0-9]{32}$/.test(completion.id) &&
      completion.closing_occurrence &&
      merge &&
      publication,
    'factory_completion_scope'
  );
  check(
    /^[a-f0-9]{32}$/.test(publication.id) &&
      /^[a-f0-9]{32}$/.test(publication.assignment_id) &&
      after.assignments.some(
        (a: any) =>
          a.id === publication.assignment_id &&
          a.id === allowance.root &&
          a.root === a.id &&
          a.publication_assignment === a.id &&
          a.phase === 'investigate' &&
          a.role === 'soda-coder' &&
          a.outcome === 'succeeded' &&
          a.reported === true
      ) &&
      publication.id === merge.publication,
    'factory_publication_owner_lineage'
  );
  check(
    merge.stage === 'merged' &&
      merge.effect === 'committed' &&
      merge.completion === 'complete' &&
      COMMIT.test(merge.head) &&
      merge.merged_commit === completion.merged_commit &&
      merge.head === merge.merged_commit &&
      merge.issue_id === publication.issue_id &&
      merge.closed_unix > 0 &&
      merge.closing_occurrence === completion.closing_occurrence &&
      after.integrated_source === merge.merged_commit,
    'factory_native_merge_closure_unconfirmed'
  );
  check(
    publication.stage === 'published' &&
      publication.project === c.repository.project &&
      publication.acceptance === c.issue.acceptance &&
      publication.pr_id === merge.pr_id &&
      publication.pr_number === merge.pr_number &&
      publication.actor === c.actors.coder &&
      merge.coder_actor === c.actors.coder &&
      merge.reviewer_actor === c.actors.reviewer,
    'factory_publication_actor_binding'
  );
  const review = publication.reviews.find(
    (r: any) =>
      r.actor === c.actors.reviewer &&
      r.head === merge.head &&
      r.event === 'APPROVED' &&
      r.effect === 'committed' &&
      r.completion === 'complete'
  );
  check(review, 'factory_independent_native_review_missing');
  const selected = [];
  for (const [role, phase, tool, runID] of [
    ['soda-coder', 'implement', 'factory.finish_implementation', publication.run],
    ['soda-reviewer', 'review', 'factory.finish_review', review.run],
  ]) {
    const a = after.assignments.find(
        (a: any) =>
          a.run === runID && a.role === role && a.phase === phase && a.outcome === 'succeeded' && a.reported === true
      ),
      r = after.runs.find((r: any) => r.id === runID),
      u = after.usage.find((u: any) => u.run === runID);
    check(
      a &&
        r &&
        u &&
        r.outcome === 'succeeded' &&
        r.role === role &&
        a.connection === c.connection.id &&
        typeof r.harness === 'string' &&
        r.harness.startsWith((c.execution?.harness ?? 'codex') + '-') &&
        (!c.execution ||
          (r.model === c.execution.model &&
            r.profile?.effort === c.execution.profile.effort &&
            Boolean(r.profile?.contributor) === c.execution.profile.contributor)) &&
        r.binding?.kind === 'factory' &&
        r.binding.id === r.id &&
        r.binding.project === after.container &&
        r.binding.child_id === a.preparation &&
        r.binding.generation === r.generation &&
        /^[a-f0-9]{32}$/.test(r.binding.invocation_id) &&
        r.execution?.state === 'terminal' &&
        r.execution.binding_id === r.id &&
        r.execution.invocation_id === r.binding.invocation_id &&
        r.execution.project === after.container &&
        r.execution.child_id === a.preparation &&
        r.execution.generation === r.generation,
      'factory_actual_native_execution_missing'
    );
    check(nativeExecutionSucceeded(r.native, r, a, after.container), 'factory_native_success_retirement_unconfirmed');
    check(
      after.submissions.some((s: any) => s.assignment === a.id && s.tool === tool && s.state === 'accepted'),
      'factory_typed_finish_missing'
    );
    check(a.publication_assignment === publication.assignment_id, 'factory_execution_publication_owner_changed');
    if (role === 'soda-reviewer') check(r.source === merge.head, 'factory_reviewed_wrong_source');
    selected.push({
      assignment: a,
      run: r,
      usage: u,
      retired_native_observation: r.native,
      active_native_observation: observedNative.find((n: any) => n.run === r.id) ?? null,
    });
  }
  const assessment = after.checks.find(
    (x: any) => Number(x.pr) === merge.pr_number && x.head === merge.head && x.verdict === 'pass'
  );
  check(assessment, 'factory_current_native_checks_missing');
  assert.deepEqual(assessment.checks, c.required_checks);
  check(
    assessment.results.length === c.required_checks.length &&
      assessment.results.every(
        (r: any, i: number) => r.context === c.required_checks[i] && r.state === 'success' && r.passed === true
      ),
    'factory_check_results_not_success'
  );
  const measurement = ci.find(
    (x: any) =>
      x.head === merge.head &&
      x.state === 'success' &&
      x.actor === c.actors.ci &&
      x.context === c.required_checks[0] &&
      x.fresh_after_baseline === true &&
      SHA.test(x.measurement?.output_digest)
  );
  check(measurement, 'factory_fresh_candidate_check_measurement_missing');
  check(
    !before.preparations.some((p: any) => p.id === measurement.preparation) &&
      Date.parse(after.at) >= Date.parse(before.at),
    'factory_historical_check_reuse'
  );
  const execution = {
    format: 1,
    candidate_check: measurement,
    scope: {
      repository: c.repository.id,
      project: c.repository.project,
      issue: c.issue.number,
      native_issue: c.issue.native_id,
      acceptance: c.issue.acceptance,
    },
    closed_allowance: allowance,
    approved_executions: selected,
    publication: {
      id: publication.id,
      assignment_id: publication.assignment_id,
      pr_number: publication.pr_number,
      pr_id: publication.pr_id,
      actor: publication.actor,
    },
    review,
    merge,
    completion,
    assessment,
    accounting: {
      all_scoped_runs_retired: true,
      global: after.global,
      total_provider_elapsed_ms: after.usage.reduce(
        (n: number, u: any) => n + Date.parse(u.ended_at) - Date.parse(u.started_at),
        0
      ),
      root_active_elapsed_ms: allowance.active_seconds * 1000,
    },
    noRawRunOrProviderOutput: true,
  };
  const bytes = new TextEncoder().encode(JSON.stringify(execution, null, 2) + '\n');
  const native = {
    provider_execution_sha256: sha(bytes),
    repository_id: c.repository.id,
    issue_id: c.issue.native_id,
    pull_request_id: merge.pr_id,
    coder_actor: c.actors.coder,
    reviewer_actor: c.actors.reviewer,
    reviewed_head: merge.head,
    merged_candidate_head: merge.merged_commit,
    required_checks: Object.fromEntries(c.required_checks.map((x: string) => [x, 'success'])),
    elapsed_ms: allowance.active_seconds * 1000,
    funding_limit_ms: allowance.active_minutes * 60_000,
  };
  return {native, execution, executionBytes: bytes};
}

export async function exerciseFactoryPhase(page: any, cfg: any, _binding: any, io: FactoryIO, permits: FactoryPermits) {
  const c = validateFactoryPhase(cfg.factory_phase),
    R = c.repository.id,
    base = 'repositories/' + R + '/factory';
  stage = 'navigate_factory';
  const workspace = cfg.guest.origin + '/workspace?repository=' + R;
  if (page.url() !== workspace) await page.goto(workspace);
  stage = 'mount_factory';
  await nativeSpacesMount(page);
  stage = 'read_factory_mount';
  const mount = await mountedSpacesFacts(page);
  const request = async (path: string, method = 'GET', body?: any) => {
    check(
      !/factory\/runs(?:\/|$)/.test(path) ||
        (method === 'POST' &&
          /^factory\/runs\/[a-f0-9]{32}\/actions$/.test(path) &&
          body?.action === 'use_current_context'),
      'factory_raw_run_route_forbidden'
    );
    return page.evaluate(
      async ({path, method, body}: any) => {
        const r = await (document.querySelector('soda-spaces') as any).binding.transport.request(path, {
          method,
          ...(body === undefined ? {} : {headers: {'Content-Type': 'application/json'}, body: JSON.stringify(body)}),
          signal: AbortSignal.timeout(30000),
        });
        return {status: r.status, body: await r.json()};
      },
      {path, method, body}
    );
  };
  const get = async (path: string) => {
    const r = await request(path);
    check(r.status === 200, 'factory_normal_read_unavailable');
    return r.body;
  };
  const read = async () => {
    const facts = JSON.parse(await io.sshRead(snapshotScript(c)));
    await io.observeNative?.(facts);
    return facts;
  };
  stage = 'read_initial_baseline';
  const before = await read();
  stage = 'validate_initial_baseline';
  validateFactoryBaseline(c, before);
  stage = 'save_initial_baseline';
  await io.save('before.json', before);
  stage = 'read_owner_factory';
  const original = await get(base);
  stage = 'read_owner_access';
  const access = await get(base + '/access'),
    policy = original.policy;
  check(
    original.repository === String(R) &&
      access.viewer?.owner_or_admin &&
      access.viewer.session_current &&
      access.editable?.policy?.editable &&
      policy?.enabled &&
      policy.paused &&
      original.effective.dispatch_open === false,
    'factory_normal_owner_held_required'
  );
  check(
    original.preparation?.project === c.repository.project &&
      original.preparation.requirements === c.requirements &&
      original.preparation.approval === c.approval &&
      original.preparation.ready &&
      policy.target_branch === c.target_branch &&
      policy.merge_method === 'fast-forward-only',
    'factory_approved_preparation_changed'
  );
  check(
    policy.intent_minutes === c.limits.intent_minutes &&
      policy.attempt_limits.active_minutes === c.limits.attempt_minutes &&
      policy.max_concurrent === 1,
    'factory_existing_finite_policy_required'
  );
  assert.deepEqual(policy.required_checks, c.required_checks);
  check(
    policy.publish_actor.actor_id === String(c.actors.coder) &&
      policy.create_actor.actor_id === String(c.actors.coder) &&
      policy.review_actor.actor_id === String(c.actors.reviewer),
    'factory_enrolled_actors_changed'
  );
  for (const role of ['soda-coder', 'soda-reviewer']) {
    const actual = policy.roles[role];
    check(actual?.harness === (c.execution?.harness ?? 'codex'), 'factory_selected_harness_required');
    if (c.execution)
      check(
        actual.model === c.execution.model &&
          actual.profile?.effort === c.execution.profile.effort &&
          Boolean(actual.profile?.contributor) === c.execution.profile.contributor,
        'factory_selected_model_profile_required'
      );
  }
  const sponsor = original.sponsorships.find((s: any) => s.connection === c.connection.id);
  check(
    sponsor?.active &&
      sponsor.grant_id === c.connection.grant &&
      sponsor.generation === c.connection.generation &&
      sponsor.allowance_minutes === c.funding.sponsor_minutes &&
      sponsor.max_concurrent === 1,
    'factory_existing_sponsorship_required'
  );
  const commandID = (label: string) => factoryCommandID(c.operation_id, label);
  const policyBody = (current: any, paused: boolean) => {
    const body: any = {
      command_id: commandID(paused ? 'held-policy' : 'open-policy'),
      expected_revision: current.revision,
      enabled: current.enabled,
      paused,
    };
    for (const key of [
      'target_branch',
      'required_checks',
      'merge_method',
      'roles',
      'reassessment_minutes',
      'intent_minutes',
      'max_concurrent',
      'attempt_limits',
    ])
      body[key] = current[key];
    for (const key of ['publish_actor', 'create_actor', 'review_actor', 'merge_actor'])
      body[key] = {actor_id: current[key].actor_id, token_id: current[key].token_id};
    if (current.resolve_actor?.actor_id)
      body.resolve_actor = {actor_id: current.resolve_actor.actor_id, token_id: current.resolve_actor.token_id};
    return body;
  };
  const command = async (label: string, path: string, method: string, body: any) => {
    stage = 'journal_' + label;
    return issueFactoryCommand(
      {label, path, method, body, repository: R, owner: cfg.project_phase.actor},
      {
        prepare: () => permits.arm(label, method, mount.apiBase + '/' + path, body),
        send: () => {
          stage = 'request_' + label;
          return request(path, method, body);
        },
        lookup: (route) => {
          stage = 'reconcile_' + label;
          return request(route);
        },
        save: io.save,
      }
    );
  };
  await command('open-policy', base + '/policy', 'PUT', policyBody(policy, false));
  await command('resume', base + '/actions', 'POST', {command_id: commandID('resume'), action: 'resume'});
  const deadline = Date.now() + c.limits.maximum_seconds * 1000,
    ci: any[] = [],
    published = new Set<string>(),
    native: any[] = [];
  let after: any,
    reason = 'bounded_window_exhausted',
    resolvedContexts = new Set<string>();
  const contextSince = new Map<string, number>();
  for (let pass = 0; pass < 180 && Date.now() < deadline; pass++) {
    stage = 'read_factory_progress';
    after = await read();
    stage = 'save_factory_progress';
    await io.save('snapshots/' + String(pass).padStart(3, '0') + '.json', after);
    check(after.acceptance?.id === c.issue.acceptance, 'factory_acceptance_changed_during_run');
    for (const r of after.runs.filter((r: any) => r.native_bound))
      if (!native.some((n) => n.run === r.id)) {
        const n = {
          run: r.id,
          project: r.project,
          unit: r.unit,
          binding: r.binding,
          source: r.source,
          observed_at: after.at,
        };
        native.push(n);
        await io.save('native-' + r.id + '.json', n);
      }
    // Each exact staged implementation or published-candidate review may replay its own canonical current-context choice.
    // The canonical helper replays the recorded choice without making another judgment.
    const pending = after.assignments
      .filter(
        (a: any) =>
          ((a.phase === 'implement' && a.role === 'soda-coder') ||
            (a.phase === 'review' && a.role === 'soda-reviewer')) &&
          a.stage === 'assigned'
      )
      .map((a: any) => ({a, r: after.runs.find((r: any) => r.id === a.run)}))
      .filter((x: any) => x.r?.context_pending === true && x.r.active === true && !x.r.reconciled && !x.r.outcome);
    check(pending.length <= 1, 'one_scoped_staged_context_run');
    if (pending.length === 1 && !resolvedContexts.has(pending[0].r.id)) {
      const {a, r} = pending[0];
      stage = 'canonical_context_status';
      let status: any;
      try {
        status = JSON.parse(await io.sshRead(contextStatusScript(c, r.id, a.id, cfg.context_status)));
      } catch {
        // A normal automatic choice may admit/retire this run between the
        // snapshot and read-only helper. Re-read selected facts once; never
        // infer eligibility or send an owner action from an unavailable status.
        stage = 'context_status_transition_read';
        const current = await read();
        stage = 'context_status_transition_save';
        await io.save('context-transition-' + String(pass).padStart(3, '0') + '.json', current);
        const same = current.assignments.find(
            (x: any) =>
              x.id === a.id &&
              x.run === r.id &&
              x.root === a.root &&
              x.phase === a.phase &&
              x.role === a.role &&
              x.source === a.source &&
              x.preparation === a.preparation &&
              x.publication_assignment === a.publication_assignment &&
              x.project === c.repository.project &&
              x.acceptance === c.issue.acceptance
          ),
          now = current.runs.find((x: any) => x.id === r.id && x.project === c.repository.project);
        check(
          same &&
            now &&
            now.context_pending !== true &&
            (now.native_bound === true || (now.settled === true && now.reconciled === true && !now.active)),
          'canonical_context_status_unavailable'
        );
        after = current;
      }
      if (status) {
        stage = 'save_context_status';
        await io.save('context-status-' + String(pass).padStart(3, '0') + '.json', status);
        check(
          status.run_id === r.id &&
            status.assignment_id === a.id &&
            status.repository === R &&
            status.issue === c.issue.number &&
            status.project === c.repository.project &&
            status.attempt_root === a.root,
          'canonical_context_status_binding'
        );
        check(
          status.phase === a.phase &&
            status.role === a.role &&
            status.source_commit === a.source &&
            status.preparation === a.preparation &&
            Number.isSafeInteger(status.current_connection_generation) &&
            status.current_connection_generation > 0 &&
            after.connection?.id === c.connection.id &&
            after.connection.owner_id === c.connection.owner_id &&
            after.connection.state === 'ready' &&
            status.current_connection_generation === after.connection.generation,
          'canonical_role_context_status_binding'
        );
        if (a.phase === 'review')
          check(
            after.publications.some(
              (p: any) =>
                p.id === status.publication &&
                p.assignment_id === a.publication_assignment &&
                p.candidate === a.source &&
                p.stage === 'published'
            ) && status.candidate === a.source,
            'canonical_review_context_publication'
          );
        const view = status.context_selection;
        if (view?.can_use_current === true) {
          check(
            view.action === 'hold' &&
              view.reason === 'context_selection_attention' &&
              view.availability === 'complete' &&
              status.context_pending === true &&
              !status.reconciled &&
              !status.credential_delegated &&
              !status.native_binding_present &&
              [view.fingerprint, view.scope_digest, view.required_digest].every((x) => SHA.test(x)),
            'canonical_current_context_owner_action'
          );
          const resolutionLabel = 'use-current-context-' + r.id;
          const resolution = await command(resolutionLabel, 'factory/runs/' + r.id + '/actions', 'POST', {
            command_id: commandID(resolutionLabel),
            action: 'use_current_context',
            context: {
              fingerprint: view.fingerprint,
              scope_digest: view.scope_digest,
              required_digest: view.required_digest,
            },
          });
          check(
            resolution.command_id === commandID(resolutionLabel) &&
              resolution.run_id === r.id &&
              resolution.assignment_id === a.id &&
              resolution.bundle === 'current' &&
              resolution.fingerprint === view.fingerprint &&
              resolution.scope_digest === view.scope_digest &&
              resolution.required_digest === view.required_digest,
            'canonical_current_context_resolution_receipt'
          );
          resolvedContexts.add(r.id);
        }
      }
    }
    if (contextPendingHold(after, contextSince, Date.now())) {
      reason = 'context_pending_unresolved';
      break;
    }
    for (const publication of after.publications.filter(
      (p: any) => p.stage === 'published' && COMMIT.test(p.candidate)
    )) {
      if (
        published.has(publication.candidate) ||
        after.merges.some((m: any) => m.publication === publication.id && m.stage === 'merged')
      )
        continue;
      const candidates = after.preparations.filter(
        (p: any) =>
          p.role === 'soda-reviewer' &&
          p.publication === publication.id &&
          p.owner === publication.assignment_id &&
          p.source === publication.candidate &&
          p.ready === true &&
          !p.stopped &&
          p.setup_exit === 0 &&
          p.check_exit === 0 &&
          !before.preparations.some((old: any) => old.id === p.id)
      );
      if (candidates.length !== 1) continue;
      const prep = candidates[0],
        body = {
          preparation: prep.id,
          head: publication.candidate,
          publication: publication.id,
          owner_assignment: prep.owner,
          attempt_root: prep.attempt_root,
          pr: publication.pr_number,
          baseline_unix_ms: Date.parse(before.at),
          before_preparations: before.preparations.map((p: any) => p.id),
        };
      await io.save('ci-' + publication.candidate + '.request.json', body);
      published.add(publication.candidate);
      stage = 'candidate_native_ci';
      let receipt;
      try {
        receipt = JSON.parse(await io.sshRead(ciScript(c, cfg.guest.origin, body), 90000));
      } catch {
        receipt = JSON.parse(await io.sshRead(ciScript(c, cfg.guest.origin, body, 'inspect'), 90000));
      }
      await io.save('ci-' + publication.candidate + '.receipt.json', receipt);
      check(receipt.state === 'success', 'factory_ci_outcome_unconfirmed');
      ci.push(receipt);
    }
    if (
      after.completions.length === 1 &&
      fullyAccounted(after) &&
      after.allowances.length === 1 &&
      after.allowances[0].closed
    ) {
      reason = 'delivered';
      break;
    }
    if (
      after.assignments.length > 0 &&
      after.allowances.length === 1 &&
      after.allowances[0].closed &&
      fullyAccounted(after)
    ) {
      reason = 'attempt_finished_without_delivery';
      break;
    }
    await new Promise((resolve) => setTimeout(resolve, 10000));
  }
  stage = 'read_final_factory';
  after = await read();
  if (!nativeIdle(after)) {
    await io.save('incomplete.json', {reason, native_custody_retained: true, noAutomaticStopOrRetry: true});
    return {
      observed: false,
      reason,
      project: c.repository.project,
      paused: false,
      native_idle: false,
      scope_runs_retired: false,
    };
  }
  const current = await get(base);
  if (current.effective.dispatch_open)
    await command('pause', base + '/actions', 'POST', {command_id: commandID('pause'), action: 'pause'});
  const paused = await get(base);
  if (!paused.policy.paused) await command('held-policy', base + '/policy', 'PUT', policyBody(paused.policy, true));
  for (let pass = 0; pass < 18; pass++) {
    after = await read();
    if (fullyAccounted(after)) break;
    await new Promise((resolve) => setTimeout(resolve, 10000));
  }
  const held = await get(base);
  check(
    held.policy.paused && held.effective.dispatch_open === false && fullyAccounted(after),
    'factory_normal_pause_accounting_unconfirmed'
  );
  await io.save('after.json', after);
  const state = {
    project: c.repository.project,
    paused: true,
    dispatch_open: false,
    native_idle: true,
    scope_runs_retired: after.runs.every((r: any) => r.settled && r.reconciled && r.credential_returned),
  };
  if (reason !== 'delivered') {
    await io.save('incomplete.json', {reason, ...state, noDecisionReroll: true, noBudgetIncrease: true});
    return {observed: false, reason, ...state};
  }
  stage = 'validate_canonical_factory';
  for (const run of after.runs) {
    run.native = JSON.parse(await io.sshRead(nativeExecutionScript(c, run.id)));
    await io.save('native-retired-' + run.id + '.json', run.native);
  }
  const evidence = deriveFactoryEvidence(c, before, after, ci, native);
  await io.saveBytes('provider-execution.json', evidence.executionBytes);
  await io.save('native.json', evidence.native);
  await io.save('result.json', {
    observed: true,
    canonicalFactory: evidence.native,
    ...state,
    controls: permits.used,
    noBudgetIncrease: true,
    noRetry: true,
    ownerUseCurrent: resolvedContexts.size > 0,
    ownerUseCurrentRunCount: resolvedContexts.size,
    canonicalRecordedContextDecisionOnly: true,
    overallQualificationComplete: false,
  });
  stage = 'complete';
  return {observed: true, native: evidence.native, ...state};
}
