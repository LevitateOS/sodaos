// Read-only final-state diagnosis. This grants no lifecycle authority and never
// treats a completed run or charged minute as a resource to delete.
import {object} from '../sodaspaces-input';
import type {FixtureObservation} from './facts';
import type {QualificationScenario} from './scenario';
import assert from 'node:assert/strict';

export interface ResourceScope {
  readonly scenario: QualificationScenario;
  readonly observation: FixtureObservation;
}

/** Stable retained identities must not drift while this scope is in use.
 * Connection generations and lifecycle state are deliberately not frozen. */
export function assertResourceInventory(scope: ResourceScope, actual: unknown): void {
  const inventory = object(actual);
  assert(Array.isArray(inventory.projects) && Array.isArray(inventory.applications), 'Resource inventory unconfirmed');
  const sorted = (value: unknown) => rows(value).sort((a, b) => String(a.id).localeCompare(String(b.id)));
  assert.deepEqual(
    sorted(inventory.projects),
    sorted(scope.observation.inventory.projects),
    'Retained Project identity changed'
  );
  assert.deepEqual(
    sorted(inventory.applications),
    sorted(scope.observation.inventory.applications),
    'Retained App identity changed'
  );
}

interface Resource {
  readonly kind: 'project' | 'application' | 'terminal';
  readonly id: string;
  readonly project_id: string | null;
  readonly managed: boolean;
  readonly permitted: boolean;
  readonly state: 'stopped' | 'not_stopped' | 'unknown';
  readonly before_snapshot: 'preserve' | 'retire_and_confirm' | 'observe';
}
interface Blocker {
  readonly code: string;
  readonly kind: Resource['kind'] | 'custody' | 'inventory';
  readonly id: string | null;
}
export interface RetirementAction {
  readonly kind: Resource['kind'];
  readonly id: string;
  readonly project_id: string | null;
  readonly actor_id: string | null;
}
export interface ClosureDiagnostic {
  readonly scope_ready: boolean;
  readonly snapshot_ready: boolean;
  readonly selected_project_ids: readonly string[];
  readonly retained_project_ids: readonly string[];
  readonly projects_outside_factory_selection: readonly string[];
  readonly resources: readonly Resource[];
  readonly blockers: readonly Blocker[];
  /** Empty on any blocker: never retire an allowed subset of an unsafe set. */
  readonly retirement_actions: readonly RetirementAction[];
  readonly retained_history: {readonly runs: number | null; readonly charged_minutes: number | null};
  readonly actions_executed: 0;
}

function rows(value: unknown): Record<string, unknown>[] {
  return Array.isArray(value) ? value.map(object) : [];
}
function stopped(row: Record<string, unknown> | undefined): Resource['state'] {
  if (
    !row ||
    typeof row.unit_state !== 'string' ||
    !Number.isSafeInteger(row.main_pid) ||
    !Number.isSafeInteger(row.control_pid) ||
    typeof row.running !== 'boolean'
  )
    return 'unknown';
  return row.unit_state === 'inactive' && row.main_pid === 0 && row.control_pid === 0 && row.running === false
    ? 'stopped'
    : 'not_stopped';
}

export function diagnoseClosure(
  s: QualificationScenario,
  observation: FixtureObservation,
  native: Record<string, unknown>
): ClosureDiagnostic {
  const resources: Resource[] = [],
    blockers: Blocker[] = [];
  const block = (code: string, kind: Blocker['kind'], id: string | null) => blockers.push({code, kind, id});
  try {
    assertResourceInventory({scenario: s, observation}, native.resource_inventory);
  } catch {
    block('retained_inventory_changed', 'inventory', null);
  }
  const projects = rows(native.project_runtime),
    applications = rows(native.application_runtime);
  const selected = [...new Set(s.factory.length ? s.factory.map((f) => f.subject.project_id) : s.managed.project_ids)];
  const retained = observation.inventory.projects.map((p) => p.id);
  const outside = retained.filter((id) => !selected.includes(id));
  // Every retained resource participates, including resources outside the
  // selected Factory Project. An unrelated stopped resource is preserved.
  for (const id of selected.filter((id) => !retained.includes(id))) block('selected_project_missing', 'project', id);
  for (const row of projects)
    if (!retained.includes(String(row.id))) block('unregistered_runtime_project', 'inventory', String(row.id));
  for (const row of applications)
    if (!observation.inventory.applications.some((a) => a.id === row.id))
      block('unregistered_runtime_application', 'inventory', String(row.id));
  if (!Array.isArray(native.runtime_orphans)) block('runtime_inventory_unconfirmed', 'inventory', null);
  for (const orphan of rows(native.runtime_orphans))
    block('unregistered_native_resource', 'inventory', String(orphan.id));
  for (const [kind, values] of [
    ['project', observation.inventory.projects],
    ['application', observation.inventory.applications],
    ['terminal', observation.inventory.terminals ?? []],
  ] as const) {
    const ids = values.map((v) => v.id);
    if (new Set(ids).size !== ids.length) block('duplicate_inventory_resource', kind, null);
    const managed =
      kind === 'project'
        ? s.managed.project_ids
        : kind === 'application'
          ? s.managed.application_ids
          : s.managed.terminal_ids;
    const permitted =
      kind === 'project'
        ? s.permitted_effects.project_lifecycle
        : kind === 'application'
          ? s.permitted_effects.application_lifecycle
          : s.permitted_effects.terminal_lifecycle;
    for (const id of managed) if (!ids.includes(id)) block('managed_resource_unconfirmed', kind, id);
    for (const id of permitted) if (!managed.includes(id)) block('lifecycle_outside_managed_scope', kind, id);
  }
  const terminalKeys = new Set<string>();
  for (const p of observation.inventory.projects) {
    const matches = projects.filter((r) => r.id === p.id),
      row = matches.length === 1 ? matches[0] : undefined;
    const state = stopped(row),
      managed = s.managed.project_ids.includes(p.id),
      permitted = managed && s.permitted_effects.project_lifecycle.includes(p.id);
    resources.push({
      kind: 'project',
      id: p.id,
      project_id: null,
      managed,
      permitted,
      state,
      before_snapshot: state === 'stopped' ? 'preserve' : state === 'unknown' ? 'observe' : 'retire_and_confirm',
    });
    if (state === 'unknown') block('project_runtime_unconfirmed', 'project', p.id);
    else if (state !== 'stopped' && !permitted) block('project_stop_outside_scope', 'project', p.id);
    if (
      !row ||
      !Array.isArray(row.terminal_ids) ||
      !row.terminal_ids.every((t) => typeof t === 'string' && /^[a-f0-9]{32}$/.test(t))
    ) {
      block('project_terminals_unconfirmed', 'project', p.id);
      continue;
    }
    for (const id of row.terminal_ids as string[]) {
      const key = `${p.id}/${id}`;
      if (terminalKeys.has(key)) {
        block('duplicate_terminal', 'terminal', id);
        continue;
      }
      terminalKeys.add(key);
      const binding = observation.inventory.terminals?.find((t) => t.id === id && t.project_id === p.id);
      const owned = s.managed.terminal_ids.includes(id),
        allowed = owned && Boolean(binding) && s.permitted_effects.terminal_lifecycle.includes(id);
      resources.push({
        kind: 'terminal',
        id,
        project_id: p.id,
        managed: owned,
        permitted: allowed,
        state: binding ? 'not_stopped' : 'unknown',
        before_snapshot: binding ? 'retire_and_confirm' : 'observe',
      });
      if (!binding) block('terminal_actor_unconfirmed', 'terminal', id);
      if (!allowed) block('terminal_retirement_outside_scope', 'terminal', id);
      const custody = rows(native.terminals).filter((t) => t.id === id && t.project_id === p.id);
      if (binding && (custody.length !== 1 || custody[0]!.actor_id !== binding.actor_id))
        block('terminal_actor_changed', 'terminal', id);
      if (custody.length !== 1 || custody[0]!.subscription !== false)
        block(
          custody[0]?.subscription === true ? 'terminal_requires_subscription_owner' : 'terminal_custody_unconfirmed',
          'terminal',
          id
        );
    }
  }
  for (const a of observation.inventory.applications) {
    const matches = applications.filter((r) => r.id === a.id),
      state = stopped(matches.length === 1 ? matches[0] : undefined);
    const managed = s.managed.application_ids.includes(a.id),
      permitted = managed && s.permitted_effects.application_lifecycle.includes(a.id);
    resources.push({
      kind: 'application',
      id: a.id,
      project_id: null,
      managed,
      permitted,
      state,
      before_snapshot: state === 'stopped' ? 'preserve' : state === 'unknown' ? 'observe' : 'retire_and_confirm',
    });
    if (state === 'unknown') block('application_runtime_unconfirmed', 'application', a.id);
    else if (state !== 'stopped' && !permitted) block('application_stop_outside_scope', 'application', a.id);
  }
  if (observation.inventory.terminals === null && !resources.some((r) => r.kind === 'terminal'))
    block('terminal_inventory_unconfirmed', 'inventory', null);
  for (const terminal of observation.inventory.terminals ?? []) {
    if (terminalKeys.has(`${terminal.project_id}/${terminal.id}`)) continue;
    const runtime = projects.filter((p) => p.id === terminal.project_id);
    if (runtime.length !== 1 || !Array.isArray(runtime[0]!.terminal_ids))
      block('terminal_runtime_unconfirmed', 'terminal', terminal.id);
    else
      resources.push({
        kind: 'terminal',
        id: terminal.id,
        project_id: terminal.project_id,
        managed: s.managed.terminal_ids.includes(terminal.id),
        permitted:
          s.managed.terminal_ids.includes(terminal.id) && s.permitted_effects.terminal_lifecycle.includes(terminal.id),
        state: 'stopped',
        before_snapshot: 'preserve',
      });
  }
  const custody = object(native.custody);
  for (const key of [
    'leases',
    'nonterminal',
    'held',
    'unsettled',
    'preparations',
    'restores',
    'setups',
    'effective_dispatch',
  ])
    if (custody[key] !== 0) block('live_or_unconfirmed_' + key, 'custody', null);
  if (!Array.isArray(native.active_units) || native.active_units.length > 0)
    block('live_or_unconfirmed_native_units', 'custody', null);
  const history = object(native.history);
  const count = (v: unknown) => (typeof v === 'number' && Number.isSafeInteger(v) && v >= 0 ? v : null);
  const order = {terminal: 0, application: 1, project: 2};
  const actions: RetirementAction[] = blockers.length
    ? []
    : resources
        .filter((r) => r.state === 'not_stopped' && r.permitted)
        .sort((a, b) => order[a.kind] - order[b.kind] || a.id.localeCompare(b.id))
        .map((r) => ({
          kind: r.kind,
          id: r.id,
          project_id: r.project_id,
          actor_id:
            r.kind === 'terminal' ? observation.inventory.terminals!.find((t) => t.id === r.id)!.actor_id : null,
        }));
  return {
    scope_ready: blockers.length === 0,
    snapshot_ready: blockers.length === 0 && resources.every((r) => r.state === 'stopped'),
    selected_project_ids: selected,
    retained_project_ids: retained,
    projects_outside_factory_selection: outside,
    resources,
    blockers,
    retirement_actions: actions,
    retained_history: {runs: count(history.runs), charged_minutes: count(history.charged_minutes)},
    actions_executed: 0,
  };
}
