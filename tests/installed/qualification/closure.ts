// Read-only final-state diagnosis. This grants no lifecycle authority and never
// treats a completed run or charged minute as a resource to delete.
import {object} from '../sodaspaces-input';
import type {FixtureObservation} from './facts';
import type {QualificationScenario} from './scenario';

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
export interface ClosureDiagnostic {
  readonly scope_ready: boolean;
  readonly snapshot_ready: boolean;
  readonly selected_project_ids: readonly string[];
  readonly retained_project_ids: readonly string[];
  readonly projects_outside_selected_route: readonly string[];
  readonly resources: readonly Resource[];
  readonly blockers: readonly Blocker[];
  readonly retained_history: {readonly runs: number | null; readonly charged_minutes: number | null};
  readonly actions_executed: 0;
}

function rows(value: unknown): Record<string, unknown>[] {
  return Array.isArray(value) ? value.map(object) : [];
}
function stopped(row: Record<string, unknown> | undefined, project: boolean): Resource['state'] {
  if (
    !row ||
    typeof row.unit_state !== 'string' ||
    !Number.isSafeInteger(row.main_pid) ||
    !Number.isSafeInteger(row.control_pid) ||
    (project && typeof row.running !== 'boolean')
  )
    return 'unknown';
  return row.unit_state === 'inactive' &&
    row.main_pid === 0 &&
    row.control_pid === 0 &&
    (!project || row.running === false)
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
  const projects = rows(native.project_runtime),
    applications = rows(native.application_runtime);
  const selected = [...new Set(s.factory.length ? s.factory.map((f) => f.subject.project_id) : s.managed.project_ids)];
  const retained = observation.inventory.projects.map((p) => p.id);
  const outside = retained.filter((id) => !selected.includes(id));
  // The current Factory/browser route reuses its selected Factory Project.
  // Its independent stop-scope equality cannot cover another retained Project;
  // J10 must consume the shared inventory rather than relaxing that assertion.
  for (const id of outside) block('selected_route_omits_retained_project', 'project', id);
  for (const id of selected.filter((id) => !retained.includes(id))) block('selected_project_missing', 'project', id);
  for (const row of projects)
    if (!retained.includes(String(row.id))) block('unregistered_runtime_project', 'inventory', String(row.id));
  const terminalKeys = new Set<string>();
  for (const p of observation.inventory.projects) {
    const matches = projects.filter((r) => r.id === p.id),
      row = matches.length === 1 ? matches[0] : undefined;
    const state = stopped(row, true),
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
    }
  }
  for (const a of observation.inventory.applications) {
    const matches = applications.filter((r) => r.id === a.id),
      state = stopped(matches.length === 1 ? matches[0] : undefined, false);
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
  for (const terminal of observation.inventory.terminals ?? [])
    if (!terminalKeys.has(`${terminal.project_id}/${terminal.id}`))
      block('terminal_runtime_unconfirmed', 'terminal', terminal.id);
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
  return {
    scope_ready: blockers.length === 0,
    snapshot_ready: blockers.length === 0 && resources.every((r) => r.state === 'stopped'),
    selected_project_ids: selected,
    retained_project_ids: retained,
    projects_outside_selected_route: outside,
    resources,
    blockers,
    retained_history: {runs: count(history.runs), charged_minutes: count(history.charged_minutes)},
    actions_executed: 0,
  };
}
