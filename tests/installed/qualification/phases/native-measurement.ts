import {check} from './support';
import {FACTORY_SNAPSHOT} from './factory-native';

/** Read-only subset of the Factory scope. A held scenario can observe existing
 * custody without claiming policy, paid authority or execution release. */
export function measurementSnapshotScript(value: any) {
  const r = value?.repository;
  check(
    Number.isSafeInteger(r?.id) &&
      r.id > 0 &&
      [r.owner, r.name].every(
        (v) => typeof v === 'string' && /^[A-Za-z0-9._-]{1,128}$/.test(v) && !['.', '..'].includes(v)
      ) &&
      /^p[a-f0-9]{24}$/.test(r.project) &&
      [value.issue?.number, value.issue?.native_id].every((v) => Number.isSafeInteger(v) && v > 0) &&
      /^[a-f0-9]{32}$/.test(value.connection?.id) &&
      typeof value.target_branch === 'string' &&
      /^refs\/heads\/[A-Za-z0-9._/-]{1,200}$/.test(value.target_branch) &&
      !value.target_branch.includes('..') &&
      !value.target_branch.endsWith('/'),
    'native_measurement_read_scope'
  );
  return (
    'const REQUEST=' +
    JSON.stringify({
      repository: {id: r.id, owner: r.owner, name: r.name, project: r.project},
      issue: {number: value.issue.number, native_id: value.issue.native_id},
      connection: {id: value.connection.id},
      target_branch: value.target_branch,
    }) +
    ';\n' +
    FACTORY_SNAPSHOT
  );
}

export type NativeMeasurement = {
  format: 1;
  kind: 'native_execution';
  observed_at: string;
  project_container: string;
  custody: {leases: number; nonterminal: number; held: number; unsettled: number};
  native_units: {
    unit: string;
    container: string | null;
    state: string;
    main_pid: number;
    control_pid: number;
  }[];
  native_bound_runs: {
    run: string;
    project: string;
    container: string;
    unit: string;
    main_pid: number;
    invocation: string;
    generation: number;
  }[];
  capacity: null;
};

function row(value: unknown): Record<string, unknown> {
  check(value !== null && typeof value === 'object' && !Array.isArray(value), 'native_measurement_object');
  return value as Record<string, unknown>;
}
function integer(value: unknown, minimum = 0): number {
  check(typeof value === 'number' && Number.isSafeInteger(value) && value >= minimum, 'native_measurement_integer');
  return value;
}
function identity(value: unknown, pattern: RegExp): string {
  check(typeof value === 'string' && pattern.test(value), 'native_measurement_identity');
  return value;
}

/** Project the existing fixed Factory observer. This performs no additional
 * native reads and never copies run bodies, provider output or lifecycle proof.
 * Presence in native_bound_runs records an observed live binding, not success. */
export function projectNativeMeasurement(snapshot: unknown): NativeMeasurement {
  const facts = row(snapshot),
    global = row(facts.global),
    container = identity(facts.container, /^[a-f0-9]{64}$/);
  check(
    typeof facts.at === 'string' && facts.at.length <= 64 && Number.isFinite(Date.parse(facts.at)),
    'native_measurement_time'
  );
  check(
    Array.isArray(global.native_units) &&
      global.native_units.length <= 4352 &&
      Array.isArray(facts.runs) &&
      facts.runs.length <= 64,
    'native_measurement_inventory'
  );
  const native_units = global.native_units.map((value) => {
    const unit = row(value),
      state = identity(
        unit.state,
        /^(active|inactive|failed|activating|deactivating|reloading|maintenance|refreshing)$/
      );
    return {
      unit: identity(unit.unit, /^soda-(factory|prepare)-[A-Za-z0-9_.-]+\.service$/),
      container: unit.container === null ? null : identity(unit.container, /^[a-f0-9]{64}$/),
      state,
      main_pid: integer(unit.pid),
      control_pid: integer(unit.control_pid),
    };
  });
  const native_bound_runs = facts.runs.flatMap((value) => {
    const run = row(value);
    if (run.native_bound !== true) return [];
    const id = identity(run.id, /^[a-f0-9]{32}$/),
      binding = row(run.binding),
      unit = row(run.unit),
      generation = integer(run.generation, 1),
      invocation = identity(unit.InvocationID, /^[a-f0-9]{32}$/);
    check(
      run.active === true &&
        unit.LoadState === 'loaded' &&
        unit.ActiveState === 'active' &&
        unit.SubState === 'running' &&
        binding.kind === 'factory' &&
        binding.id === id &&
        binding.project === container &&
        binding.invocation_id === invocation &&
        binding.generation === generation,
      'native_measurement_live_binding'
    );
    const mainPID =
      typeof unit.MainPID === 'string' && /^[1-9][0-9]*$/.test(unit.MainPID) ? Number(unit.MainPID) : null;
    return [
      {
        run: id,
        project: identity(run.project, /^p[a-f0-9]{24}$/),
        container,
        unit: 'soda-factory-' + id + '.service',
        main_pid: integer(mainPID, 1),
        invocation,
        generation,
      },
    ];
  });
  check(
    new Set(native_units.map((unit) => String(unit.container) + ':' + unit.unit)).size === native_units.length &&
      new Set(native_bound_runs.map((run) => run.run)).size === native_bound_runs.length,
    'native_measurement_unique_inventory'
  );
  return {
    format: 1,
    kind: 'native_execution',
    observed_at: facts.at,
    project_container: container,
    custody: {
      leases: integer(global.leases),
      nonterminal: integer(global.nonterminal),
      held: integer(global.held),
      unsettled: integer(global.unsettled),
    },
    native_units,
    native_bound_runs,
    capacity: null,
  };
}
