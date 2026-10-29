// Request place: New request (the composer, M9) and one request's page (M11),
// lane B. The entry keeps this view alive per request id while other places
// show (SPEC M2), so the composer and its polls survive a place switch. After
// the first save, New request claims its request's address (replace), so the
// same composer continues as that request's page.
import {html, render} from 'lit';
import type {TemplateResult} from 'lit';
import {id, object, readSodaErrorCode, readSodaJSON, SodaRequestError} from './sodaspaces-api.js';
import './sodaspaces-intent.js';
import type {SodaIntent} from './sodaspaces-intent.js';
import {decisionID} from './sodaspaces-intent-response.js';
import {requestOverview} from './sodaspaces-factory-progress-response.js';
import {factoryIssueDetail} from './sodaspaces-factory-work-response.js';
import './sodaspaces-factory-verification-allowance.js';
import {renderRepositoryPicker} from './sodaspaces-repository-picker-view.js';
import {repositoryChoices} from './sodaspaces-repository-response.js';
import type {RepositoryChoices} from './sodaspaces-repository-response.js';
import type {PreparedExtensionMount} from './soda-extension.js';
import {newRequestID, repositoryPlaces} from './soda-places.js';
import type {Place} from './soda-places.js';
import {recordStartedRequest} from './soda-place-requests.js';
import type {PlaceContext, PlaceMount} from './soda-place-view.js';

type IntentAPI = (path: string, body?: Record<string, unknown>, signal?: AbortSignal) => Promise<unknown>;

/** The composer's API: Soda paths under /api/, JSON in and out, typed refusals. */
function intentAPI(transport: PreparedExtensionMount): IntentAPI {
  return async (path, body, signal) => {
    const response = await transport.request(path.replace(/^\/api\//, ''), {
      method: body ? 'POST' : 'GET',
      headers: body ? {'Content-Type': 'application/json'} : {},
      ...(body ? {body: JSON.stringify(body)} : {}),
      signal: signal ?? null,
    });
    if (response.ok) return readSodaJSON(response);
    throw new SodaRequestError(response.status, await readSodaErrorCode(response));
  };
}

// The signed-in actor, read once per session from the inventory the
// Workspace reads too. The composer keys its local drafts and choices by it.
const actors = new WeakMap<PreparedExtensionMount, Promise<string>>();
function readActor(transport: PreparedExtensionMount): Promise<string> {
  let actor = actors.get(transport);
  if (!actor) {
    actor = (async () => {
      const response = await transport.request('spaces', {method: 'GET', signal: AbortSignal.timeout(20000)});
      if (!response.ok) throw new SodaRequestError(response.status);
      const value = object(object(await readSodaJSON(response, 4 * 1024 * 1024)).actor).id;
      if (!id(value)) throw Error('Actor unavailable');
      return value;
    })();
    actor.catch(() => actors.delete(transport));
    actors.set(transport, actor);
  }
  return actor;
}

type Phase =
  | {kind: 'loading'}
  | {kind: 'ready'}
  | {kind: 'missing'}
  | {kind: 'elsewhere'; href?: string; slug?: string}
  | {kind: 'failed'; message: string};

interface WorkRow {
  issue: string;
  pull?: string;
}

interface Summary {
  loading: boolean;
  error: string;
  rows: WorkRow[];
  loaded: boolean;
}

interface Picker {
  open: boolean;
  query: string;
  result: RepositoryChoices | undefined;
  cursors: string[];
  selected: string;
  busy: boolean;
  error: string;
  authorization?: {ideas: number; tasks: number} | undefined;
  request?: AbortController | undefined;
}

/** Visible owner/name for a repository id; undefined when not visible. */
async function repositorySlug(api: IntentAPI, repository: string): Promise<string | undefined> {
  try {
    const repo = object(object(await api('/api/environments?repository_id=' + repository)).repository);
    if (repo.id !== repository || typeof repo.owner !== 'string' || typeof repo.name !== 'string') return undefined;
    if (!repo.owner || !repo.name || /[/\\]/.test(repo.owner + repo.name)) return undefined;
    return `${repo.owner}/${repo.name}`;
  } catch {
    return undefined;
  }
}

function repositoryPath(context: PlaceContext): string {
  return `${context.prefix}/${context.slug.split('/').map(encodeURIComponent).join('/')}`;
}

export const mountRequestPlace: PlaceMount = (host, context, initial) => {
  const api = intentAPI(context.transport);
  const places = context.places;
  let request = initial.place === 'request' ? initial.request : '';
  const startedNew = request === newRequestID;
  let fresh = startedNew;
  let active = true;
  let disposed = false;
  let actor = '';
  let phase: Phase = {kind: 'loading'};
  let accepted = false;
  let bound = '0';
  let notice = '';
  const summary: Summary = {loading: false, error: '', rows: [], loaded: false};
  const picker: Picker = {
    open: false,
    query: '',
    result: undefined,
    cursors: [''],
    selected: '',
    busy: false,
    error: '',
  };
  const lifetime = new AbortController();

  const composer = () => host.querySelector<SodaIntent>('soda-intent');

  // ---- Repository confirmation (Choose repository, M10) ----
  async function search(page: number) {
    if (page === 1) picker.cursors = [''];
    const cursor = picker.cursors[page - 1];
    if (cursor === undefined) return;
    picker.request?.abort();
    const controller = new AbortController();
    picker.request = controller;
    picker.busy = true;
    picker.error = '';
    draw();
    const params = new URLSearchParams({q: picker.query});
    if (cursor) params.set('cursor', cursor);
    try {
      const result = repositoryChoices(await api('/api/repositories?' + params, undefined, controller.signal), page);
      if (picker.request !== controller) return;
      picker.cursors = picker.cursors.slice(0, page);
      if (result.nextCursor) picker.cursors.push(result.nextCursor);
      picker.result = result;
      // This repository is preselected whenever it is listed.
      if (!picker.selected || !result.items.some((item) => item.id === picker.selected))
        picker.selected = result.items.some((item) => item.id === context.repository) ? context.repository : '';
    } catch {
      if (picker.request === controller)
        picker.error = 'Could not find repositories. Search again; your request stays saved.';
    } finally {
      if (picker.request === controller) {
        picker.request = undefined;
        picker.busy = false;
      }
      draw();
    }
  }

  function openPicker(event: Event) {
    const detail = (event as CustomEvent).detail as Record<string, unknown> | undefined;
    const ideas = detail?.authorizedIdeas,
      tasks = detail?.tasks;
    picker.authorization =
      Number.isInteger(ideas) && Number.isInteger(tasks) ? {ideas: ideas as number, tasks: tasks as number} : undefined;
    picker.open = true;
    picker.query = context.slug.split('/')[1] ?? '';
    picker.selected = context.repository;
    picker.result = undefined;
    draw();
    void search(1);
    focusLater('.soda-request-picker h2');
  }

  async function confirmRepository() {
    const intent = composer();
    const choice = picker.result?.items.find((item) => item.id === picker.selected);
    if (!intent || !choice || picker.busy) return;
    picker.busy = true;
    picker.error = '';
    draw();
    const ok = await intent.acceptRepository(choice.id);
    if (disposed) return;
    picker.busy = false;
    if (!ok) {
      picker.error = 'Could not save the repository for this request. Your request stays saved; try again.';
      draw();
      return;
    }
    notice = intent.generationNotice;
    picker.open = false;
    draw();
    intent.focusReview();
  }

  function closePicker() {
    picker.request?.abort();
    picker.open = false;
    picker.busy = false;
    draw();
    composer()?.focusReview();
  }

  // ---- Build summary (after acceptance) ----
  async function loadSummary() {
    if (summary.loading || !decisionID(request)) return;
    summary.loading = true;
    summary.error = '';
    draw();
    try {
      const overview = requestOverview(
        await api(`/api/factory/intents/${request}/overview`, undefined, lifetime.signal),
        request
      );
      const issues = [...new Set(overview.mappings.filter((row) => row.effective).map((row) => row.issue_index))];
      const rows = await Promise.all(
        issues.map(async (issue): Promise<WorkRow> => {
          try {
            const detail = factoryIssueDetail(
              await api(`/api/repositories/${context.repository}/factory/issues/${issue}`, undefined, lifetime.signal),
              context.repository,
              issue
            );
            const pull = detail.merge?.pr_number;
            return pull && id(pull) ? {issue, pull} : {issue};
          } catch {
            return {issue};
          }
        })
      );
      summary.rows = rows.sort((a, b) => Number(a.issue) - Number(b.issue));
      summary.loaded = true;
    } catch {
      if (!lifetime.signal.aborted) summary.error = 'Could not load the build summary. Try again.';
    } finally {
      summary.loading = false;
      draw();
    }
  }

  function renderSummary(): TemplateResult | string {
    if (!accepted || bound !== context.repository) return '';
    const repo = repositoryPath(context);
    return html`<section class="soda-build-summary" aria-label="Build summary">
      <h3>Build summary</h3>
      ${summary.error ? html`<p role="status">${summary.error}</p>` : ''}
      ${summary.loading && !summary.loaded ? html`<p role="status">Loading the work items…</p>` : ''}
      ${
        summary.loaded && !summary.rows.length
          ? html`<p>No issues yet. Soda creates one issue per work item after acceptance.</p>`
          : ''
      }
      ${
        summary.rows.length
          ? html`<ul class="soda-build-rows">
              ${summary.rows.map(
                (row) => html`<li>
                  <a href=${`${repo}/issues/${row.issue}`}>Issue #${row.issue}</a>
                  <a href=${places.factoryStatus(row.issue)}>Factory status</a>
                  ${
                    row.pull
                      ? html`<a href=${`${repo}/pulls/${row.pull}`}>Pull request #${row.pull}</a>`
                      : html`<span class="soda-build-muted">No pull request yet</span>`
                  }
                </li>`
              )}
            </ul>`
          : ''
      }
      <p class="soda-build-links">
        <a href=${places.feed()}>Follow the build in Feed</a>
        <a href=${places.app()}>App versions</a>
        <button type="button" class="ui button" ?disabled=${summary.loading} @click=${() => void loadSummary()}>
          Refresh summary
        </button>
      </p>
      <details class="soda-verification-limits">
        <summary>Verification limits</summary>
        <soda-factory-verification-allowance
          .context=${active ? {transport: context.transport, repository: context.repository, intent: request} : undefined}
        ></soda-factory-verification-allowance>
      </details>
    </section>`;
  }

  // ---- Composer events ----
  function becomeRequest(intent: string) {
    if (!fresh || !decisionID(intent)) return;
    fresh = false;
    request = intent;
    recordStartedRequest(context.repository, intent);
    draw();
    context.navigate(places.to({place: 'request', request: intent}), {replace: true});
  }

  function onResumed(event: Event) {
    const detail = (event as CustomEvent).detail as Record<string, unknown> | undefined;
    if (!detail || typeof detail.intent !== 'string') return;
    if (detail.intent !== request) return;
    applyRepository(typeof detail.repository === 'string' ? detail.repository : '0', detail.state === 'accepted');
  }

  function applyRepository(repository: string, isAccepted: boolean) {
    bound = repository;
    accepted = isAccepted;
    if (repository !== '0' && repository !== context.repository) {
      phase = {kind: 'elsewhere'};
      draw();
      void repositorySlug(api, repository).then((slug) => {
        if (disposed || !slug || phase.kind !== 'elsewhere') return;
        phase = {
          kind: 'elsewhere',
          slug,
          href: repositoryPlaces(context.prefix, slug).to({place: 'request', request}),
        };
        draw();
      });
      return;
    }
    draw();
    if (accepted && !summary.loaded) void loadSummary();
  }

  function onOperation() {
    const intent = composer();
    // Explore ideas saves the first draft without an intent-saved event.
    if (fresh && intent?.revision?.intent) becomeRequest(intent.revision.intent);
    const revision = intent?.revision;
    if (revision && revision.intent === request) {
      const wasAccepted = accepted;
      bound = revision.repository;
      accepted = revision.state === 'accepted';
      if (accepted && !wasAccepted) void loadSummary();
      draw();
    }
  }

  function focusLater(selector: string) {
    void Promise.resolve().then(() => host.querySelector<HTMLElement>(selector)?.focus({preventScroll: true}));
  }

  // ---- Page ----
  function renderNotice(): TemplateResult | string {
    if (phase.kind === 'loading') return html`<p role="status">Loading request…</p>`;
    if (phase.kind === 'failed')
      return html`<p role="status">${phase.message}</p>
        <button type="button" class="ui button" @click=${() => void start()}>Try again</button>`;
    if (phase.kind === 'missing')
      return html`<section class="soda-place-notice" aria-label="Private request">
        <h2>Request</h2>
        <p role="status">This request is private to the person who made it.</p>
        <p class="soda-build-links">
          <a href=${`${repositoryPath(context)}/issues`}>Open issues</a>
          <a href=${places.factoryStatus()}>Factory status</a>
        </p>
      </section>`;
    if (phase.kind === 'elsewhere')
      return html`<section class="soda-place-notice" aria-label="Request in another repository">
        <h2>Request</h2>
        <p role="status">
          This request belongs to ${phase.href ? html`<a href=${phase.href}>${phase.slug}</a>` : 'another repository'},
          not ${context.slug}.
        </p>
      </section>`;
    return '';
  }

  function draw() {
    if (disposed) return;
    const ready = phase.kind === 'ready';
    const unbound = bound === '0';
    render(
      html`<section class="soda-request-page" aria-label=${fresh ? 'New request' : 'Request'}>
        <nav class="soda-request-crumbs" aria-label="Request">
          <a href=${places.requests()}>← Requests</a>
          <span>${context.slug}</span>
        </nav>
        ${fresh && ready ? html`<h2 class="soda-request-title">New request</h2>` : ''}
        ${ready && unbound && !picker.open ? html`<p class="soda-request-for">This request is for ${context.slug}</p>` : ''}
        ${renderNotice()}
        ${
          ready
            ? html`<div class="soda-request-composer" ?hidden=${picker.open}>
                <soda-intent
                  .api=${api}
                  .actor=${actor}
                  .request=${fresh ? '' : request}
                  .project=${context.slug}
                  .blocked=${false}
                  .active=${active && !picker.open}
                  @intent-saved=${(event: Event) => becomeRequest(String((event as CustomEvent).detail?.intent ?? ''))}
                  @soda-intent-operation=${onOperation}
                  @intent-resumed=${onResumed}
                  @intent-reviewed=${openPicker}
                  @intent-accepted=${() => {
                    accepted = true;
                    bound = context.repository;
                    void loadSummary();
                  }}
                  @intent-preparation=${() => context.navigate(places.settings('environment'))}
                ></soda-intent>
                ${notice ? html`<p class="soda-request-notice" role="status">${notice}</p>` : ''}
                ${
                  bound === context.repository && (context.viewer.admin || context.viewer.operator)
                    ? html`<p class="soda-build-links">
                        <a href=${places.settings('environment')}>Set up the development environment</a>
                      </p>`
                    : ''
                }
              </div>`
            : ''
        }
        ${
          ready && picker.open
            ? html`<div class="soda-request-picker soda-setup-form">
                ${renderRepositoryPicker(
                  {
                    query: picker.query,
                    result: picker.result,
                    selected: picker.selected,
                    busy: picker.busy,
                    error: picker.error,
                    blocked: !active,
                    createURL: `${context.prefix}/repo/create`,
                    suggested: context.repository,
                    ...(picker.authorization ? {authorization: picker.authorization} : {}),
                  },
                  {
                    query: (value) => {
                      picker.query = value;
                      draw();
                    },
                    search: (page) => void search(page),
                    select: (value) => {
                      picker.selected = value;
                      draw();
                    },
                    back: closePicker,
                    continue: () => void confirmRepository(),
                  }
                )}
                <p><button type="button" class="ui button" @click=${closePicker}>Back to the request</button></p>
              </div>`
            : ''
        }
        ${ready ? renderSummary() : ''}
      </section>`,
      host
    );
  }

  /** Read the request (and the actor), then show the composer or the right notice. */
  async function start() {
    phase = {kind: 'loading'};
    draw();
    try {
      if (fresh) {
        actor = await readActor(context.transport);
        if (disposed) return;
        phase = {kind: 'ready'};
        draw();
        return;
      }
      if (!decisionID(request)) {
        phase = {kind: 'missing'};
        draw();
        return;
      }
      let value: Record<string, unknown>;
      try {
        value = object(await api(`/api/factory/intents/${request}`, undefined, lifetime.signal));
      } catch (error) {
        if (error instanceof SodaRequestError && (error.status === 404 || error.status === 403)) {
          phase = {kind: 'missing'};
          draw();
          return;
        }
        throw error;
      }
      if (disposed) return;
      // Only the author reads a request, so its owner is the actor.
      actor = id(value.owner) ? value.owner : await readActor(context.transport);
      if (disposed) return;
      const repository = typeof value.repository === 'string' ? value.repository : '0';
      phase = {kind: 'ready'};
      applyRepository(repository, value.state === 'accepted');
    } catch {
      if (disposed) return;
      phase = {kind: 'failed', message: 'Could not load this request. Try again.'};
      draw();
    }
  }

  void start();

  return {
    // The viewer's standing may arrive after mount.
    update: () => draw(),
    setActive(value: boolean) {
      active = value;
      draw();
    },
    claim(place: Place) {
      return place.place === 'request' && !fresh && startedNew && place.request === request;
    },
    dispose() {
      disposed = true;
      lifetime.abort();
      picker.request?.abort();
      render(html``, host);
    },
  };
};
