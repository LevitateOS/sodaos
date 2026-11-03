import {render} from 'lit';
import {presentationLinks, renderPresentation} from '@factory/product/frontend/storybook/work-presentation-fixtures.js';
import type {
  PresentationState,
  PresentationView,
} from '@factory/product/frontend/storybook/work-presentation-fixtures.js';

function show(view: PresentationView, state: PresentationState = 'recorded', navigation = true, prefix = '/desktop') {
  render(
    renderPresentation(view, state, navigation ? presentationLinks(prefix) : undefined),
    document.querySelector('main')!
  );
}
const fixture = {show};
declare global {
  interface Window {
    factoryPresentationFixture: typeof fixture;
  }
}
window.factoryPresentationFixture = fixture;
show('detail');
