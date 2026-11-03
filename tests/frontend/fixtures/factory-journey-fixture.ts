import {SodaFactoryWorkJourney} from '@factory/product/frontend/factory/work/journey.js';
import type {JourneyContext} from '@factory/product/frontend/factory/work/journey.js';
import {
  journeyIssue,
  journeyLinks,
  journeyRepository,
  journeyResponse,
  journeyStoryContext,
} from '@factory/product/frontend/storybook/journey-fixtures.js';
import type {JourneyStoryState} from '@factory/product/frontend/storybook/journey-fixtures.js';

let element: SodaFactoryWorkJourney;
let context: JourneyContext;
let calls = 0;
let generation = 'declared-account-a';
let failure = 0;
let held: {signal: AbortSignal | undefined; resolve: (response: Response) => void} | undefined;

function mount(state: JourneyStoryState) {
  failure = 0;
  context = journeyStoryContext(state);
  const peer = context.transport;
  context = {
    ...context,
    transport: {
      ...peer,
      async request(path, init) {
        calls++;
        if (failure) return Response.json({error: {code: 'declared_read_failure'}}, {status: failure});
        return peer.request(path, init);
      },
    },
  };
  element = new SodaFactoryWorkJourney();
  document.querySelector('main')!.replaceChildren(element);
  element.context = context;
}

function hold() {
  generation = 'declared-account-a';
  context = {
    repository: journeyRepository,
    issue: journeyIssue,
    links: journeyLinks(),
    transport: {
      get generation() {
        return generation;
      },
      assetBase: '/',
      request(_path, init) {
        calls++;
        // This peer deliberately ignores cancellation so the element must
        // also refuse late results; the signal is retained for observation.
        return new Promise((resolve) => {
          held = {signal: init?.signal ?? undefined, resolve};
        });
      },
    },
  };
  element = new SodaFactoryWorkJourney();
  document.querySelector('main')!.replaceChildren(element);
  element.context = context;
}

const fixture = {
  mount,
  hold,
  calls: () => calls,
  failRefresh: (status: number) => {
    failure = status;
  },
  heldAborted: () => held?.signal?.aborted ?? false,
  rebindLinks() {
    context = {...context, links: journeyLinks('/another-host-route')};
    element.context = context;
  },
  retireGeneration() {
    generation = 'declared-account-b';
  },
  release() {
    held?.resolve(Response.json(journeyResponse()));
  },
};
declare global {
  interface Window {
    factoryJourneyFixture: typeof fixture;
  }
}
window.factoryJourneyFixture = fixture;
mount('recorded');
