// Request place: one request's page (M11), lane B. The entry keeps this view
// alive per request id while other places show (SPEC M2), so the composer and
// its polls survive a place switch. New requests start as a Feed post.
import {html, render} from 'lit';
import type {TemplateResult} from 'lit';
import {
  check,
  id,
  object,
  readSodaErrorCode,
  readSodaJSON,
  SodaRequestError,
} from '@factory/product/frontend/factory/api/models.js';
import '@factory/product/frontend/factory/request/request.js';
import type {SodaIntent} from '@factory/product/frontend/factory/request/request.js';
import {decisionID} from '@factory/product/frontend/factory/api/request-response.js';
import {requestOverview} from '@factory/product/frontend/factory/api/progress-response.js';
import {factoryIssueDetail} from '@factory/product/frontend/factory/api/work-response.js';
import '@factory/product/frontend/factory/work/verification-allowance.js';
import {renderRepositoryPicker} from './sodaspaces-repository-picker-view.js';
import {repositoryChoices} from './sodaspaces-repository-response.js';
import type {RepositoryChoices} from './sodaspaces-repository-response.js';
import type {PreparedExtensionMount} from './soda-extension.js';
import {repositoryPlaces} from './soda-places.js';
import type {PlaceContext} from './soda-place-view.js';
import {withFactoryRepository} from './soda-place-view.js';
import {fountainCommandScope, fountainRepository, fountainRepositoryIdentity} from './soda-factory-feed-host.js';

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

function repositoryPath(context: PlaceContext): string {
  return `${context.prefix}/${context.slug.split('/').map(encodeURIComponent).join('/')}`;
}

export const mountRequestPlace = withFactoryRepository((host, context, initial) => {
  const api = intentAPI(context.transport);
  const places = context.places;
  const request = initial.place === 'request' ? initial.request : '';
  let active = true;
  let disposed = false;
  const actor = context.factoryActor;
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
  let addressRead: AbortController | undefined;

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
    const read = new AbortController();
    picker.request = read;
    try {
      const resolved = await fountainRepository(context.transport, choice.id, read.signal);
      check(resolved.host === context.factoryHost && resolved.actor === actor);
      if (disposed || !active || picker.request !== read || read.signal.aborted) return;
      if (!(await intent.acceptRepository(resolved.repository))) throw Error('Repository acceptance failed.');
      if (disposed || picker.request !== read || read.signal.aborted) return;
      notice = intent.generationNotice;
      picker.open = false;
      intent.focusReview();
    } catch {
      if (disposed || picker.request !== read || read.signal.aborted) return;
      picker.error = 'Could not save the repository for this request. Your request stays saved; try again.';
    } finally {
      if (picker.request === read) {
        picker.request = undefined;
        picker.busy = false;
        draw();
      }
    }
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
      check(overview.context.repository === context.factoryRepository);
      const issues = [...new Set(overview.mappings.filter((row) => row.effective).map((row) => row.issue_index))];
      const rows = await Promise.all(
        issues.map(async (issue): Promise<WorkRow> => {
          try {
            const detail = factoryIssueDetail(
              await api(
                `/api/repositories/${context.factoryRepository}/factory/issues/${issue}`,
                undefined,
                lifetime.signal
              ),
              context.factoryRepository,
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
    if (!accepted || bound !== context.factoryRepository) return '';
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
                  <a href=${places.issues(row.issue)}>Issue #${row.issue}</a>
                  <a href=${places.factoryStatus(row.issue)}>Factory status</a>
                  ${
                    row.pull
                      ? html`<a href=${places.pulls(row.pull)}>Pull request #${row.pull}</a>`
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
          .context=${active ? {transport: context.transport, repository: context.factoryRepository, intent: request, commandScope: (signal: AbortSignal) => fountainCommandScope(context.transport, signal)} : undefined}
        ></soda-factory-verification-allowance>
      </details>
    </section>`;
  }

  // ---- Composer events ----
  function onResumed(event: Event) {
    const detail = (event as CustomEvent).detail as Record<string, unknown> | undefined;
    if (!detail || typeof detail.intent !== 'string') return;
    if (detail.intent !== request) return;
    applyRepository(typeof detail.repository === 'string' ? detail.repository : '0', detail.state === 'accepted');
  }

  function applyRepository(repository: string, isAccepted: boolean) {
    addressRead?.abort();
    bound = repository;
    accepted = isAccepted;
    if (repository !== '0' && repository !== context.factoryRepository) {
      phase = {kind: 'elsewhere'};
      draw();
      const read = new AbortController();
      addressRead = read;
      void fountainRepositoryIdentity(context.transport, repository, AbortSignal.any([read.signal, lifetime.signal]))
        .then((resolved) => {
          if (disposed || read.signal.aborted || bound !== repository || phase.kind !== 'elsewhere') return;
          check(resolved.host === context.factoryHost && resolved.actor === actor);
          phase = {
            kind: 'elsewhere',
            slug: resolved.name,
            href: repositoryPlaces(context.prefix, resolved.name).to({place: 'request', request}),
          };
          draw();
        })
        .catch(() => {
          /* Unavailable or foreign repositories supply no native link. */
        });
      return;
    }
    phase = {kind: 'ready'};
    draw();
    if (accepted && !summary.loaded) void loadSummary();
  }

  function onOperation() {
    const revision = composer()?.revision;
    if (revision && revision.intent === request) {
      applyRepository(revision.repository, revision.state === 'accepted');
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
          <a href=${places.issues()}>Issues</a>
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
      html`<section class="soda-request-page" aria-label="Request">
        <nav class="soda-request-crumbs" aria-label="Request">
          <a href=${places.feed()}>← Feed</a>
          <span>${context.slug}</span>
        </nav>
        ${ready && unbound && !picker.open ? html`<p class="soda-request-for">This request is for ${context.slug}</p>` : ''}
        ${renderNotice()}
        ${
          ready
            ? html`<div class="soda-request-composer" ?hidden=${picker.open}>
                <soda-intent
                  .api=${api}
                  .actor=${actor}
                  .draftScope=${`fountain:${context.factoryHost}`}
                  .request=${request}
                  .project=${context.slug}
                  .blocked=${false}
                  .active=${active}
                  @soda-intent-operation=${onOperation}
                  @intent-resumed=${onResumed}
                  @intent-reviewed=${openPicker}
                  @intent-accepted=${() => {
                    const revision = composer()?.revision;
                    if (revision) applyRepository(revision.repository, true);
                  }}
                  @intent-preparation=${() => context.navigate(places.settings('environment'))}
                ></soda-intent>
                ${notice ? html`<p class="soda-request-notice" role="status">${notice}</p>` : ''}
                ${
                  bound === context.factoryRepository && (context.viewer.admin || context.viewer.operator)
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
      check(value.owner === actor);
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
      if (!value) {
        picker.request?.abort();
        picker.request = undefined;
        picker.busy = false;
      }
      draw();
    },
    dispose() {
      disposed = true;
      lifetime.abort();
      picker.request?.abort();
      render(html``, host);
    },
  };
});
