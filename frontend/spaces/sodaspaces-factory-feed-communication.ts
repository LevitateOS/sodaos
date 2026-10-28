import {html} from 'lit';
import type {TemplateResult} from 'lit';
import components from '../../internal/factory/feed_components.json';
import type {FeedCommunication} from './sodaspaces-factory-feed-response.js';
import type {FeedArtifactRequest} from './sodaspaces-factory-feed-artifact.js';
import {actionIcon, actionList, communicationAction, commitTrail, renderFeedAction, sourceExcerpt} from './sodaspaces-factory-feed-actions.js';
import './sodaspaces-factory-feed-artifact.js';

// The catalog owns the semantic presentation; workers submit facts, never UI.
export function feedComponent(id: string | undefined) {
  return components.find((component) => component.id === id);
}
const defaultPresentation: Record<FeedCommunication['kind'], string> = {
  ask: 'question', answer: 'question', reply: 'discussion', flag: 'blocker',
  learned: 'knowledge', suggest: 'proposal', partial: 'progress', blocked: 'blocker',
  stuck: 'diagnosis', record_decision: 'decision', finish_incomplete: 'incomplete',
  system: 'brief', library_maintenance: 'maintenance', artifact: 'artifact',
  journey: 'journey', diagnosis: 'diagnosis', repair_draft: 'comparison',
};

// Audience captions for the factory system components: what happened, in the
// reader's words, instead of the catalog's internal component names.
const audienceCaption: Readonly<Record<string, string>> = {
  F01: 'Preparation',
  F02: 'Work started',
  F05: 'Work created',
  F06: 'Merged',
  F07: 'Ready',
  F09: 'Run problem',
  F10: 'App ready',
  // Drafts are captioned as proposals above; recorded F11 posts are decisions.
  F11: 'Decision',
};

function componentCaption(c: FeedCommunication, label: string | undefined): string | undefined {
  if (c.intentSource?.state === 'draft') return 'Proposal (draft) · not accepted';
  const id = c.componentId;
  return (id !== undefined && Object.hasOwn(audienceCaption, id) ? audienceCaption[id] : undefined) ?? label;
}

// A summary that is only a machine code (e.g. "start-unconfirmed") is the
// recorded cause, not prose: keep it, collapsed under Technical details.
const machineCode = /^[a-z0-9]+(?:[-_.:/][a-z0-9]+)+$/;
function componentSummary(summary: string): TemplateResult {
  if (!machineCode.test(summary.trim())) return html`<p class="soda-feed-component-summary">${summary}</p>`;
  return html`<details class="soda-feed-technical-details">
    <summary>Technical details</summary>
    <p>${summary}</p>
  </details>`;
}

export function renderFeedCommunication(c: FeedCommunication, entryKey: string, request?: FeedArtifactRequest): TemplateResult {
  const component = feedComponent(c.componentId);
  const presentation = component?.presentation ?? defaultPresentation[c.kind];
  const action = communicationAction(c, presentation);
  const extraBalance = !['progress', 'incomplete', 'blocker', 'result', 'merge', 'brief'].includes(action) && (c.completed.length > 0 || c.remaining.length > 0);
  return html`<div class="soda-feed-communication" data-feed-component=${component?.id ?? c.kind} data-presentation=${presentation} data-communication=${c.kind}>
    <p class="soda-feed-component-caption">${actionIcon(action)}${componentCaption(c, component?.label) ?? defaultPresentation[c.kind].replace(/^./, (letter) => letter.toUpperCase())}${c.scope && action !== 'knowledge' ? html`<span> · ${c.scope}</span>` : ''}</p>
    <h3>${c.title}</h3>
    ${['brief', 'knowledge', 'discussion', 'result'].includes(action) || c.summary === c.title ? '' : componentSummary(c.summary)}
    ${c.severity && action !== 'finding' ? html`<p class="soda-feed-severity" data-severity=${c.severity}>${c.severity === 'critical' ? 'Critical concern' : c.severity === 'warning' ? 'Attention needed' : 'For your information'}</p>` : ''}
    ${renderFeedAction(c, presentation, entryKey)}
    ${c.dispositionReason ? html`<p class="soda-feed-disposition-reason" data-post-resolution=${c.disposition ?? ''}>${c.dispositionReason}</p>` : ''}
    ${c.source && !['finding', 'correction', 'comparison', 'investigation'].includes(action) ? html`<div class="soda-feed-action">${sourceExcerpt(c.source)}</div>` : ''}
    ${c.commits?.length ? html`<div class="soda-feed-action" data-action="commits">${commitTrail(c.commits)}</div>` : ''}
    ${c.artifacts?.length ? html`<div class="soda-feed-artifacts">${c.artifacts.map((artifact) => html`<soda-feed-artifact .artifact=${artifact} .request=${request}></soda-feed-artifact>`)}</div>` : ''}
    ${extraBalance ? html`<details class="soda-action-more"><summary>Work completed and remaining</summary><div class="soda-action-balance">${actionList('Completed', c.completed)}${actionList('Still needed', c.remaining)}</div></details>` : ''}
    ${action !== 'diagnosis' ? actionList('Already tried', c.attempts) : ''}
    ${action !== 'question' ? actionList('Options', c.options) : ''}
    ${(action !== 'decision' || c.items.length > 0) && c.evidenceRefs.length ? html`<details class="soda-feed-evidence-refs"><summary>Evidence · ${c.evidenceRefs.length} ${c.evidenceRefs.length === 1 ? 'reference' : 'references'}</summary><ul>${c.evidenceRefs.map((ref) => html`<li><code>${ref}</code></li>`)}</ul></details>` : ''}
    ${c.libraryItemId ? html`<p class="soda-feed-library-reference"><button type="button" data-library-item=${c.libraryItemId} data-library-entry=${entryKey}>Open in project library${c.libraryRevision ? ` · revision ${c.libraryRevision}` : ''}</button></p>` : ''}
    ${c.intentSource ? html`<div class="soda-feed-source-reference">${c.intentSource.state === 'draft' ? 'Proposed choice' : 'Recorded choice'} · ${c.intentSource.state}<br><button type="button" data-intent-source=${c.intentSource.intent} data-intent-revision=${c.intentSource.revision} data-source-entry=${entryKey}>Open build request</button><details><summary>Source revision</summary><code>${c.intentSource.revision}</code>${c.intentSource.predecessor ? html`<p>Previous revision: <code>${c.intentSource.predecessor}</code></p>` : ''}</details></div>` : ''}
    ${c.planSource ? html`<details class="soda-feed-source-reference"><summary>Recorded plan · generation ${c.planSource.generation}</summary><p>Plan: <code>${c.planSource.plan}</code></p><p>Assignment: <code>${c.planSource.attemptRoot}</code></p><p>Source commit: <code>${c.planSource.source}</code></p><p>Plan digest: <code>${c.planSource.digest}</code></p><p>Recording this revision does not itself admit or implement the plan.</p></details>` : ''}
    ${c.planningSource ? html`<details class="soda-feed-source-reference"><summary>Recorded strategy${c.planningSource.provisional ? ' · provisional' : ''}</summary><p>Controller choice: ${c.planningSource.strategy || 'None'} · ${c.planningSource.outcome}</p><p>Snapshot: <code>${c.planningSource.snapshot}</code></p><p>Record digest: <code>${c.planningSource.recordDigest}</code></p>${c.planningSource.judgment['selected'] ? html`<details><summary>Actual JEV answer</summary><pre>${JSON.stringify(c.planningSource.judgment, null, 2)}</pre></details>` : html`<p>No JEV answer was recorded.</p>`}<p>Execution and verification remain separate records.</p></details>` : ''}
    ${c.componentId === 'F10' ? html`<button type="button" data-feed-open-app=${entryKey}>Open project app</button>` : ''}
  </div>`;
}
