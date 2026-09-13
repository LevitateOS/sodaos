import {createWorkspaceModel} from '../../tests/frontend/fixtures/workspace-model.ts';
import {emptyLayout, putEntry, selectTab, splitPane} from '../../frontend/spaces/sodaspaces-layout.ts';
export const scenarios = ['welcome', 'created', 'ready', 'working', 'split', 'long-names', 'stopped', 'unavailable', 'expired', 'join-failed'] as const;
export type Scenario = typeof scenarios[number];
export type Model = ReturnType<typeof createWorkspaceModel>;
export function scenarioModel(origin: string, scenario: Scenario) {
  const seed = createWorkspaceModel(origin), model = createWorkspaceModel(origin, '1', undefined, true);
  const alpha = structuredClone(seed.spaces[0]);
  if (!alpha) throw Error('Missing shared fixture seed');
  alpha.environment.profile = model.profile;
  if (scenario !== 'welcome') model.spaces.push(alpha);
  if (scenario === 'long-names') alpha.environment.repository = 'alice/notification-service-with-a-long-repository-name';
  if (['created', 'join-failed'].includes(scenario)) alpha.login = '';
  if (['created', 'ready', 'stopped', 'join-failed', 'unavailable', 'expired'].includes(scenario)) alpha.terminals = [];
  if (scenario === 'stopped' && alpha.observed) alpha.observed.running = false;
  if (scenario === 'unavailable') {alpha.native_unavailable = true; alpha.observed = null; model.setComplete(false);}
  if (scenario === 'expired') model.setStatus(401);
  if (scenario === 'join-failed') model.setJoinFailure(true);
  let layout = emptyLayout(crypto.randomUUID());
  if (['working', 'split', 'long-names'].includes(scenario)) {
    for (const [index, terminal] of alpha.terminals.entries()) {
      terminal.name = index ? 'Tests' : 'Development';
      if (scenario === 'long-names') terminal.name += ' · notification service integration';
      const key = crypto.randomUUID();
      if (index && scenario === 'split') layout = splitPane(layout, layout.focused, 'right', crypto.randomUUID(), crypto.randomUUID());
      layout = selectTab(putEntry(layout, {key, environmentId: alpha.environment.id, locator: {kind: 'existing', id: terminal.id}}), key);
    }
  }
  return {model, layout};
}
