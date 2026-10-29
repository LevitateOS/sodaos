// Factory status place (M20, lane C): the issue-number lookup with the issues
// the Factory works on in this repository, and one issue's page composing the
// Factory work components. Handing an existing or imported issue to the
// Factory happens on the one-issue page (requirements acceptance).
import {html, render} from 'lit';
import {id} from './sodaspaces-api.js';
import {spacesResponse} from './sodaspaces-inventory-response.js';
import './sodaspaces-factory-work-journey.js';
import './sodaspaces-factory-acceptance.js';
import './sodaspaces-factory-context-resolution.js';
import './sodaspaces-factory-completion-retry.js';
import type {Place} from './soda-places.js';
import type {PlaceContext, PlaceMount} from './soda-place-view.js';

type Issues = {state: 'loading'} | {state: 'failed'} | {state: 'loaded'; issues: string[]};

// The issues come from this repository's spaces' factory_runs (the spaces
// collection, paged by next_after); there is no repository issue list endpoint.
async function factoryIssues(context: PlaceContext, signal: AbortSignal): Promise<string[]> {
  const issues = new Set<string>();
  const seen = new Set<string>();
  let after = '';
  for (;;) {
    const response = await context.transport.request('spaces' + (after ? '?after=' + encodeURIComponent(after) : ''), {
      method: 'GET',
      signal,
    });
    if (!response.ok) throw Error('Factory issues could not load.');
    const page = spacesResponse(await response.json());
    for (const space of page.items)
      if (space.environment.repository_id === context.repository)
        for (const run of space.factory_runs) if (id(run.issue)) issues.add(run.issue);
    if (page.complete || !page.nextAfter || seen.has(page.nextAfter)) break;
    seen.add(page.nextAfter);
    after = page.nextAfter;
  }
  return [...issues].sort((a, b) => Number(b) - Number(a));
}

export const mountFactoryStatusPlace: PlaceMount = (host, context, initial) => {
  let issue = initial.place === 'factory-status' ? initial.issue : undefined;
  let active = true;
  let input = '';
  let error = '';
  let issues: Issues = {state: 'loading'};
  let loads: AbortController | undefined;
  // Native repository path (with the installation prefix) for issue links.
  const repositoryPath = `${context.prefix}/${context.slug}`;

  const load = () => {
    if (loads) return;
    const controller = new AbortController();
    loads = controller;
    issues = {state: 'loading'};
    factoryIssues(context, controller.signal).then(
      (list) => {
        if (loads !== controller) return;
        issues = {state: 'loaded', issues: list};
        draw();
      },
      () => {
        if (loads !== controller) return;
        loads = undefined;
        issues = {state: 'failed'};
        draw();
      }
    );
  };
  const stopLoad = () => {
    if (loads && issues.state === 'loading') {
      loads.abort();
      loads = undefined;
    }
  };

  const submit = (event: Event) => {
    event.preventDefault();
    const value = input.trim();
    if (!id(value)) {
      error = 'Enter the issue number from this repository.';
      draw();
      return;
    }
    error = '';
    context.navigate(context.places.factoryStatus(value));
  };

  const lookup = () =>
    html`<section class="soda-factory-status" aria-label="Factory status">
      <h2>Factory status</h2>
      <form class="soda-factory-status-lookup" @submit=${submit} novalidate>
        <label
          >Factory status for issue #
          <input
            type="number"
            min="1"
            step="1"
            inputmode="numeric"
            .value=${input}
            @input=${(event: Event) => {
              input = (event.target as HTMLInputElement).value;
            }}
        /></label>
        <button type="submit" class="ui button">Open</button>
      </form>
      <p role="status">${error}</p>
      <h3>Issues with the Factory</h3>
      ${issues.state === 'loading'
        ? html`<p role="status">Loading the Factory's issues…</p>`
        : issues.state === 'failed'
          ? html`<p role="status">The Factory's issues could not load.</p>`
          : issues.issues.length === 0
            ? html`<p>No issues are with the Factory yet.</p>`
            : html`<ul class="soda-factory-status-issues">
                ${issues.issues.map((n) => html`<li><a href=${context.places.factoryStatus(n)}>Issue #${n}</a></li>`)}
              </ul>`}
    </section>`;

  const oneIssue = (n: string) => {
    const back = html`<p><a href=${context.places.factoryStatus()}>← Factory status</a></p>`;
    if (!id(n))
      return html`<section class="soda-factory-status" aria-label="Factory status">
        ${back}
        <h2>Factory status</h2>
        <p role="status">Enter the issue number from this repository.</p>
      </section>`;
    // While inactive the components get no context, so they stop live work.
    const live = active ? {transport: context.transport, repository: context.repository, issue: n} : undefined;
    const withPath = live && {...live, repositoryPath};
    return html`<section class="soda-factory-status" aria-label=${`Factory status for issue #${n}`}>
      ${back}
      <h2>Factory status for issue #${n}</h2>
      <p><a href=${`${repositoryPath}/issues/${n}`}>Open issue</a></p>
      <soda-factory-work-journey .context=${withPath}></soda-factory-work-journey>
      <soda-factory-requirements-acceptance .context=${withPath}></soda-factory-requirements-acceptance>
      <soda-factory-context-resolution .context=${live}></soda-factory-context-resolution>
      <soda-factory-completion-retry .context=${live}></soda-factory-completion-retry>
    </section>`;
  };

  const draw = () => render(issue ? oneIssue(issue) : lookup(), host);

  if (!issue) load();
  draw();
  return {
    update(place: Place) {
      if (place.place !== 'factory-status') return;
      issue = place.issue;
      if (!issue && active) load();
      draw();
    },
    setActive(value: boolean) {
      active = value;
      if (!active) stopLoad();
      else if (!issue) load();
      draw();
    },
    dispose() {
      loads?.abort();
      loads = undefined;
      render(html``, host);
    },
  };
};
