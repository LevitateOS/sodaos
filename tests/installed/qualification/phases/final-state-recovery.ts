// Final mutable-state ordering. Product lifecycle and storage effects retain
// their existing owners; supplemental recovery never invents a Docker kind.
import assert from 'node:assert/strict';
import type {Page} from 'playwright';
import type {ResourceScope} from '../closure';
import type {ApplicationHandoff} from '../application';
import type {openApplicationOwner, ApplicationRecordWitness} from '../application-owner';
import {sha, check} from './support';
import {readDocker} from './docker-native';
import type {DockerBinding, DockerIO, Witness} from './docker-binding';
import {retireOwnedResources, exercisePersistence, type PersistencePermits} from './docker-persistence';

export interface ColdCapture {
  format: 1;
  action: 'cold-capture';
  complete: true;
  directory: string;
  container: string;
  seal_sha256: string;
  writers_started: false;
  automatic_retry: false;
}
export interface ColdDestination {
  format: 1;
  action: 'isolated-cold-restore';
  complete: true;
  container: string;
  directory: string;
  image: string;
  network: 'none';
  state: 'created';
  writers_started: false;
  source_untouched: true;
  automatic_retry: false;
}
export interface FinalRecoveryRuntime {
  /** Normal administrator Start; the separate App owner is then authenticated
   * with its retained credentials. Never bootstrap, create or change a record. */
  openApplication(
    target: 'primary' | ColdDestination,
    handoff: ApplicationHandoff
  ): Promise<Awaited<ReturnType<typeof openApplicationOwner>>>;
  /** Normal administrator Stop, then exact native writer/lock confirmation. */
  stopApplication(target: 'primary' | ColdDestination, handoff: ApplicationHandoff): Promise<void>;
  /** Stop the exact outer after confirming this fresh protected snapshot, then
   * invoke the pinned Rust `journey cold capture` once. No automatic restart. */
  capture(current: unknown): Promise<ColdCapture>;
  /** Invoke `journey cold restore`; returned container is still stopped. */
  restore(capture: ColdCapture): Promise<ColdDestination>;
  /** Verify network-none/new storage, then explicitly start that container.
   * Read its native snapshot using the same installed inner reader. */
  startAndObserve(destination: ColdDestination): Promise<unknown>;
  observeIsolated(destination: ColdDestination): Promise<unknown>;
  /** Stop the exact isolated writer/outer; leave all storage/evidence intact. */
  stopIsolated(destination: ColdDestination): Promise<void>;
}

// Copying files changes inode numbers. Keep all retained business state and
// ownership/mode/byte identities; compare transient runtime IDs separately.
function portableProtected(native: any) {
  check(
    native?.protected && native.protected_sha256 === sha(JSON.stringify(native.protected)),
    'current_protected_state_required'
  );
  const value = structuredClone(native.protected);
  delete value.key_metadata?.inode;
  for (const project of value.projects ?? []) delete project.inode;
  return value;
}
function sameRetainedState(before: any, after: any, w: Witness) {
  check(
    Array.isArray(after?.active_units) &&
      after.active_units.length === 0 &&
      Array.isArray(after.runtime_orphans) &&
      after.runtime_orphans.length === 0 &&
      ['project_runtime', 'application_runtime'].every(
        (kind) =>
          Array.isArray(after[kind]) &&
          after[kind].every(
            (r: any) => r.unit_state === 'inactive' && r.main_pid === 0 && r.control_pid === 0 && r.running === false
          )
      ),
    'isolated_native_writers_retired'
  );
  assert.deepEqual(portableProtected(after), portableProtected(before), 'same_complete_retained_state');
  check(after.project?.project === w.project && after.project.sha256 === w.sha256, 'same_project_witness');
}
async function readApplication(
  runtime: FinalRecoveryRuntime,
  target: 'primary' | ColdDestination,
  app: ApplicationHandoff,
  record: ApplicationRecordWitness
) {
  check(
    record.owner_team === app.owner.team_id &&
      record.operation_id === app.record.operation_id &&
      record.label === app.record.label,
    'same_selected_app_owner_record'
  );
  try {
    const owner = await runtime.openApplication(target, app);
    await owner.readRecord(record);
    return await owner.readRendered(record);
  } finally {
    await runtime.stopApplication(target, app);
  }
}

/** Call after Factory pause/accounting and J18's finish. The concrete transport
 * supplies App owner sessions and the privileged cold executor; neither is a
 * Boolean qualification callback. The protected collector retains its four
 * existing canonical observation kinds. */
export async function exerciseFinalStateRecovery(
  input: {
    page: Page;
    apiBase: string;
    binding: DockerBinding;
    witness: Witness;
    installationNative: unknown;
    scope: ResourceScope;
    actor: string;
    permits: PersistencePermits;
    application: ApplicationHandoff;
    record: ApplicationRecordWitness;
  },
  io: DockerIO,
  runtime: FinalRecoveryRuntime
) {
  const {page, apiBase, binding: b, witness: w, scope, actor, permits, application, record} = input;
  check(
    !scope.scenario.authority.held &&
      scope.scenario.authority.scope_receipt_file !== null &&
      scope.scenario.permitted_effects.same_image_recreation &&
      scope.scenario.permitted_effects.cold_restore,
    'final_recovery_held'
  );
  check(
    scope.scenario.managed.application_ids.includes(application.application.id) &&
      scope.scenario.permitted_effects.application_lifecycle.includes(application.application.id),
    'app_recovery_scope_required'
  );
  // This owner checks global accounting, every native writer and full resource
  // inventory before issuing any stop, and captures CURRENT protected state.
  await retireOwnedResources(page, apiBase, w, b, io, permits, scope, actor);
  const persistence = await exercisePersistence(b, w, input.installationNative, io, scope);
  check(
    application.transport.distribution === 'docker' &&
      application.transport.outer_container_id === persistence.before.outer_id &&
      persistence.after.inner.application_runtime.some(
        (r: any) => r.id === application.application.id && r.container_id === application.container_id
      ),
    'retained_app_in_new_outer_required'
  );
  // Same-image recreation changes the outer ID, not App/data/owner identity.
  // Never reuse the adoption's stale Docker outer for the normal readback.
  const retainedApp: ApplicationHandoff = {
    ...application,
    transport: {...application.transport, outer_container_id: persistence.after.outer_id},
  };
  const primaryApp = await readApplication(runtime, 'primary', retainedApp, record);
  // App Start/Stop publishes observation rows. Capture a fresh final baseline
  // after that readback instead of reusing persistence's earlier whole hash.
  const final = await readDocker(io, b, 'snapshot', w, scope);
  const prior = persistence.after.inner.protected,
    current = final.inner.protected;
  assert.deepEqual(current.facts.runs, prior.facts.runs, 'same_factory_history_after_app_readback');
  assert.deepEqual(current.facts.policies, prior.facts.policies, 'same_factory_policies_after_app_readback');
  for (const table of Object.keys(prior.ledgers).filter(
    (name) => name.startsWith('factory_') || name.startsWith('identity_')
  ))
    assert.deepEqual(current.ledgers[table], prior.ledgers[table], 'same_retained_accounting_' + table);
  await io.save('supplemental/recovery/final-before-cold.json', final);
  const captured = await runtime.capture(final);
  check(
    captured.complete &&
      captured.container === final.outer_id &&
      /^[a-f0-9]{64}$/.test(captured.seal_sha256) &&
      captured.writers_started === false &&
      captured.automatic_retry === false,
    'complete_current_cold_set_required'
  );
  await io.save('supplemental/recovery/capture.json', captured);
  const destination = await runtime.restore(captured);
  check(
    destination.complete &&
      destination.network === 'none' &&
      destination.state === 'created' &&
      destination.writers_started === false &&
      destination.source_untouched &&
      destination.automatic_retry === false &&
      destination.image === b.image &&
      destination.container !== captured.container,
    'fresh_stopped_isolated_destination_required'
  );
  await io.save('supplemental/recovery/restore.json', destination);
  let supplemental;
  try {
    const restored = (await runtime.startAndObserve(destination)) as any;
    sameRetainedState(final.inner, restored, w);
    const isolatedApp = await readApplication(runtime, destination, retainedApp, record);
    const after = (await runtime.observeIsolated(destination)) as any;
    sameRetainedState(final.inner, after, w);
    supplemental = {
      format: 1,
      application: {owner_team: record.owner_team, record, primary: primaryApp, isolated: isolatedApp},
      cold: {
        seal_sha256: captured.seal_sha256,
        container: destination.container,
        same_protected_state: true,
        same_project_witness: true,
        same_factory_history: true,
      },
      capacity: null,
      canonical_kinds: ['installation', 'browser_project', 'factory', 'persistence'],
      production_qualification: false,
    };
  } finally {
    await runtime.stopIsolated(destination);
  }
  await io.save('supplemental/recovery/result.json', supplemental);
  return {persistence, supplemental};
}
