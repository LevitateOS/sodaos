import {html} from 'lit';
import type {TemplateResult} from 'lit';
import components from '../../internal/factory/feed_components.json';
import type {FeedCommunication} from './sodaspaces-factory-feed-response.js';
import type {FeedArtifactRequest} from './sodaspaces-factory-feed-artifact.js';
import {actionIcon, actionList, communicationAction, commitTrail, renderFeedAction, sourceExcerpt} from './sodaspaces-factory-feed-actions.js';
import './sodaspaces-factory-feed-artifact.js';

// The catalog owns the semantic presentation; workers submit facts, never UI.
export const feedComponentCatalog = components;
export function feedComponent(id: string | undefined) {
  return components.find((component) => component.id === id);
}
const defaultPresentation: Record<FeedCommunication['kind'], string> = {
  ask: 'question', answer: 'question', reply: 'discussion', flag: 'blocker',
  learned: 'knowledge', suggest: 'proposal', partial: 'progress', blocked: 'blocker',
  stuck: 'diagnosis', record_decision: 'decision', finish_incomplete: 'incomplete',
  system: 'brief', library_maintenance: 'maintenance', artifact: 'artifact',
};

export function renderFeedCommunication(c: FeedCommunication, entryKey: string, request?: FeedArtifactRequest): TemplateResult {
  const component = feedComponent(c.componentId);
  const presentation = component?.presentation ?? defaultPresentation[c.kind];
  const action = communicationAction(c, presentation);
  const extraBalance = !['progress', 'incomplete', 'blocker', 'result', 'merge', 'brief'].includes(action) && (c.completed.length > 0 || c.remaining.length > 0);
  return html`<div class="soda-feed-communication" data-feed-component=${component?.id ?? c.kind} data-presentation=${presentation} data-communication=${c.kind}>
    <p class="soda-feed-component-caption">${actionIcon(action)}${component?.label ?? defaultPresentation[c.kind].replace(/^./, (letter) => letter.toUpperCase())}${c.scope && action !== 'knowledge' ? html`<span> · ${c.scope}</span>` : ''}</p>
    <h3>${c.title}</h3>
    ${['brief', 'knowledge', 'discussion', 'result'].includes(action) || c.summary === c.title ? '' : html`<p class="soda-feed-component-summary">${c.summary}</p>`}
    ${c.severity && action !== 'finding' ? html`<p class="soda-feed-severity" data-severity=${c.severity}>${c.severity === 'critical' ? 'Critical concern' : c.severity === 'warning' ? 'Attention needed' : 'For your information'}</p>` : ''}
    ${renderFeedAction(c, presentation, entryKey)}
    ${c.source && !['finding', 'correction', 'comparison', 'investigation'].includes(action) ? html`<div class="soda-feed-action">${sourceExcerpt(c.source)}</div>` : ''}
    ${c.commits?.length ? html`<div class="soda-feed-action" data-action="commits">${commitTrail(c.commits)}</div>` : ''}
    ${c.artifacts?.length ? html`<div class="soda-feed-artifacts">${c.artifacts.map((artifact) => html`<soda-feed-artifact .artifact=${artifact} .request=${request}></soda-feed-artifact>`)}</div>` : ''}
    ${extraBalance ? html`<details class="soda-action-more"><summary>Work completed and remaining</summary><div class="soda-action-balance">${actionList('Completed', c.completed)}${actionList('Still needed', c.remaining)}</div></details>` : ''}
    ${action !== 'diagnosis' ? actionList('Already tried', c.attempts) : ''}
    ${action !== 'question' ? actionList('Options', c.options) : ''}
    ${(action !== 'decision' || c.items.length > 0) && c.evidenceRefs.length ? html`<details class="soda-feed-evidence-refs"><summary>Evidence · ${c.evidenceRefs.length} ${c.evidenceRefs.length === 1 ? 'reference' : 'references'}</summary><ul>${c.evidenceRefs.map((ref) => html`<li><code>${ref}</code></li>`)}</ul></details>` : ''}
    ${c.libraryItemId ? html`<p class="soda-feed-library-reference"><button type="button" data-library-item=${c.libraryItemId} data-library-entry=${entryKey}>Open in project library${c.libraryRevision ? ` · revision ${c.libraryRevision}` : ''}</button></p>` : ''}
  </div>`;
}
