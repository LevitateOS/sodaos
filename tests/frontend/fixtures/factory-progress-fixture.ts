import {
  SodaFactoryWorkOverview,
  SodaFactoryWorkPlan,
  SodaFactoryDirectionRevision,
} from '@factory/product/frontend/factory/work/progress.js';
import {progressStoryPeer} from '@factory/product/frontend/storybook/progress-fixtures.js';
import type {ProgressStoryState} from '@factory/product/frontend/storybook/progress-fixtures.js';
import {presentationLinks} from '@factory/product/frontend/storybook/work-presentation-fixtures.js';

let peer: ReturnType<typeof progressStoryPeer>;
let overview: SodaFactoryWorkOverview;
let plan: SodaFactoryWorkPlan;
let direction: SodaFactoryDirectionRevision;
function mount(state: ProgressStoryState, held = false) {
  peer = progressStoryPeer(state);
  if (held) peer.hold();
  overview = new SodaFactoryWorkOverview();
  plan = new SodaFactoryWorkPlan();
  direction = new SodaFactoryDirectionRevision();
  document.querySelector('main')!.replaceChildren(overview, plan, direction);
  overview.context = peer.overview;
  plan.context = peer.plan;
  direction.context = peer.direction;
}
const fixture = {
  mount,
  calls: () => peer.calls,
  scope: () => overview.boundScope,
  rebind() {
    overview.context = {...peer.overview, links: presentationLinks('/another-shell')};
    plan.context = {...peer.plan, links: presentationLinks('/another-shell')};
    direction.context = peer.direction;
  },
  fail: (status: number) => peer.fail(status),
  retire() {
    overview.context = undefined;
    plan.context = undefined;
    direction.context = undefined;
  },
  release: () => peer.release(),
  aborted: () => peer.aborted(),
  retireGeneration: () => peer.retireGeneration(),
  disconnect() {
    overview.remove();
    plan.remove();
    direction.remove();
  },
};
declare global {
  interface Window {
    factoryProgressFixture: typeof fixture;
  }
}
window.factoryProgressFixture = fixture;
mount('ready');
