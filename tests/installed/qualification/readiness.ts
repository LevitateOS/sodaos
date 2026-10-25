import assert from 'node:assert/strict';
import {object} from '../sodaspaces-input';
import {discoverFixture, type FixtureObservation} from './facts';
import {observeNative, NativeReadError} from './native';
import {publicBytes} from './validate';
import type {QualificationScenario} from './scenario';

export interface ReadinessCheck {
  readonly name: string;
  readonly state: 'pass' | 'blocked' | 'unknown';
}
export interface Readiness {
  readonly format: 1;
  readonly checked_at: string;
  readonly ready: boolean;
  readonly checks: readonly ReadinessCheck[];
  readonly observation: FixtureObservation | null;
  readonly native: unknown;
  readonly scope: 'read-only readiness; no execution or qualification';
}

/** All known scenario requirements are evaluated before any execution path. */
export async function checkScenario(s: QualificationScenario): Promise<Readiness> {
  const checks: ReadinessCheck[] = [];
  const check = (name: string, pass: boolean, known = true) =>
    checks.push({name, state: !known ? 'unknown' : pass ? 'pass' : 'blocked'});
  let manifest: Record<string, unknown> | null = null;
  let candidateBound = false;
  try {
    manifest = object(JSON.parse((await publicBytes(s.candidate.release_manifest)).toString()));
    assert.equal(manifest.soda_revision, s.candidate.source_revision);
    const assets = manifest.assets;
    assert(Array.isArray(assets));
    assert(assets.some((a) => a.name === 'candidate.json' && a.sha256 === s.candidate.candidate_sha256));
    assert(assets.some((a) => a.name === 'payload.json' && a.sha256 === s.candidate.payload_sha256));
    candidateBound = true;
    check('candidate_source', true);
  } catch {
    check('candidate_source', false);
  }
  let trust = false;
  try {
    assert(s.trust);
    await publicBytes(s.trust.known_hosts);
    await publicBytes(s.trust.appliance_ca);
    const response = Bun.spawnSync(
      [
        '/usr/bin/curl',
        '--noproxy',
        '*',
        '--fail',
        '--silent',
        '--show-error',
        '--max-time',
        '10',
        '--cacert',
        s.trust.appliance_ca.path,
        '--output',
        '/dev/null',
        '--write-out',
        '%{http_code}',
        s.target.origin + '/user/login',
      ],
      {stdout: 'pipe', stderr: 'pipe', timeout: 15000, maxBuffer: 4096}
    );
    assert.equal(response.exitCode, 0);
    assert.equal(response.stdout.toString(), '200');
    trust = true;
  } catch {
    /* Unknown/untrusted target never gets an SSH observation. */
  }
  check('target_trust', trust);
  const effects = s.permitted_effects;
  const subset = (a: readonly string[], b: readonly string[]) => a.every((id) => b.includes(id));
  check(
    'command_scope',
    subset(effects.project_lifecycle, s.managed.project_ids) &&
      subset(effects.application_lifecycle, s.managed.application_ids) &&
      subset(effects.terminal_lifecycle, s.managed.terminal_ids) &&
      subset(
        effects.factory_repositories,
        s.factory.map((f) => f.subject.repository_id)
      ) &&
      (effects.current_context_phases.length === 0 || effects.factory_repositories.length > 0)
  );
  let native: Record<string, unknown> | null = null,
    observation: FixtureObservation | null = null;
  if (trust && candidateBound) {
    try {
      native = object(await observeNative(s));
      observation = await discoverFixture(s.factory, async () => native);
      check('native_observation', true);
    } catch (error) {
      native = {observation_error: error instanceof NativeReadError ? error.stage : 'metadata_shape'};
      check('native_observation', false);
    }
  } else check('native_observation', false, false);
  if (native && observation) {
    check(
      'installed_identity',
      native.uuid === s.target.uuid &&
        native.distribution === s.target.distribution &&
        native.source_revision === s.candidate.source_revision &&
        native.payload_sha256 === s.candidate.payload_sha256
    );
    const loaded = object(native.loaded_images),
      expected = object(native.expected_images);
    check(
      'installed_images',
      Object.keys(expected).length === 6 &&
        Object.entries(expected).every(([role, image]) => loaded[role] === object(image).Config)
    );
    if (s.target.distribution === 'docker') {
      const expectedDocker = object(object(manifest!.docker).Image),
        actualDocker = object(native.docker);
      check(
        'docker_identity',
        typeof actualDocker.container_id === 'string' &&
          /^[a-f0-9]{64}$/.test(actualDocker.container_id) &&
          actualDocker.reference === 'ghcr.io/levitateos/sodaos@' + expectedDocker.Manifest &&
          (actualDocker.engine_image === expectedDocker.Config ||
            (actualDocker.engine_image === expectedDocker.Manifest &&
              actualDocker.descriptor_digest === expectedDocker.Manifest))
      );
    }
    const custody = object(native.custody);
    check(
      'custody',
      ['leases', 'nonterminal', 'held', 'unsettled', 'preparations', 'restores', 'setups', 'effective_dispatch'].every(
        (k) => custody[k] === 0
      ) &&
        Array.isArray(native.active_units) &&
        native.active_units.length === 0
    );
    check(
      'managed_inventory',
      subset(
        s.managed.project_ids,
        observation.inventory.projects.map((p) => p.id)
      ) &&
        subset(
          s.managed.application_ids,
          observation.inventory.applications.map((a) => a.id)
        ) &&
        subset(s.managed.terminal_ids, observation.inventory.terminals?.map((t) => t.id) ?? [])
    );
    const requirements = Array.isArray(native.factory_readiness) ? native.factory_readiness.map(object) : [];
    const runtime = Array.isArray(native.project_runtime) ? native.project_runtime.map(object) : [];
    for (const [index, f] of s.factory.entries()) {
      const b = f.subject,
        actual = observation.factory[index],
        state = requirements.find((r) => r.repository_id === b.repository_id && r.issue_number === b.issue_number);
      check(
        `factory_${index}_bindings`,
        actual?.project_id === b.project_id &&
          actual.native_issue_id === b.native_issue_id &&
          observation.inventory.projects.some(
            (p) => p.id === b.project_id && p.owner_id === actual.connection?.owner_id
          )
      );
      check(
        `factory_${index}_connection`,
        actual?.connection?.state === 'ready' &&
          Number.isSafeInteger(actual.connection.generation) &&
          actual.connection.generation > 0
      );
      check(
        `factory_${index}_accepted_inputs`,
        state?.acceptance === b.acceptance &&
          state.requirements === b.requirements &&
          state.approval === b.approval &&
          runtime.some(
            (p) =>
              p.id === b.project_id && p.source === b.source_commit && p.check_script_sha256 === b.check_script_sha256
          ) &&
          JSON.stringify(state.required_checks) === JSON.stringify(b.required_checks)
      );
      const preps = Array.isArray(state?.preparations) ? state.preparations.map(object) : [];
      check(
        `factory_${index}_standing_preparations`,
        [
          [b.coder_preparation, 'soda-coder'],
          [b.reviewer_preparation, 'soda-reviewer'],
        ].every(([id, role]) =>
          preps.some(
            (p) =>
              p.id === id &&
              p.role === role &&
              p.source === b.source_commit &&
              p.requirements === b.requirements &&
              p.approval === b.approval &&
              p.ready === true &&
              p.stopped === false &&
              p.setup_exit === 0 &&
              p.check_exit === 0
          )
        )
      );
      check(
        `factory_${index}_effect_scope`,
        state?.publish_actor === b.actors.coder &&
          state.review_actor === b.actors.reviewer &&
          Array.isArray(native.ci_actors) &&
          native.ci_actors.some((a) => object(a).id === b.actors.ci && object(a).admin === false) &&
          effects.factory_repositories.includes(b.repository_id) &&
          ['implement', 'review'].every((phase) =>
            effects.current_context_phases.includes(phase as 'implement' | 'review')
          )
      );
    }
    if (s.cases.includes('factory')) check('factory_selection', s.factory.length > 0);
    if (s.cases.some((c) => c === 'persistence' || c === 'cold_recovery')) {
      check(
        'persistence_resource_scope',
        observation.inventory.terminals !== null &&
          subset(
            observation.inventory.projects.map((p) => p.id),
            effects.project_lifecycle
          ) &&
          subset(
            observation.inventory.applications.map((a) => a.id),
            effects.application_lifecycle
          ) &&
          subset(observation.inventory.terminals?.map((t) => t.id) ?? [], effects.terminal_lifecycle)
      );
      check('persistence_operation', s.target.distribution !== 'docker' || effects.same_image_recreation);
      if (s.cases.includes('cold_recovery')) check('cold_operation', effects.cold_restore);
    }
    if (s.cases.includes('app')) check('app_selection', s.managed.application_ids.length > 0);
  } else {
    for (const name of [
      'installed_identity',
      'installed_images',
      'custody',
      'managed_inventory',
      'factory_inputs',
      'persistence_resource_scope',
    ])
      check(name, false, false);
  }
  // A saved approval is not inferred from prepared setup or a non-null path.
  // Exact paid authority is admitted by the protected run path, not this read.
  check('execution_released', !s.authority.held);
  check('paid_authority', s.factory.length === 0, s.factory.length === 0);
  for (const selected of ['app', 'basic_capacity', 'two_issue_capacity', 'cold_recovery'] as const)
    if (s.cases.includes(selected)) check(`${selected}_route`, false, false);
  return {
    format: 1,
    checked_at: new Date().toISOString(),
    ready: checks.every((c) => c.state === 'pass'),
    checks,
    observation,
    native,
    scope: 'read-only readiness; no execution or qualification',
  };
}
