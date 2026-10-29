// Requests place: the viewer's requests in this repository (M8, lane B). One
// row per request with one status and a plain link to its Request page. The
// list pages through every saved request bound here, and adds the viewer's
// drafts started here that are not yet bound ("repository not yet
// confirmed"), which the old composer recorded in this browser.
import {html, render} from 'lit';
import type {TemplateResult} from 'lit';
import {readSodaJSON} from './sodaspaces-api.js';
import {intentList, intentPlanStatus, intentRevision} from './sodaspaces-intent-response.js';
import type {SavedRequest} from './sodaspaces-intent-response.js';
import {requestOverview} from './sodaspaces-factory-progress-response.js';
import type {PlaceContext, PlaceMount} from './soda-place-view.js';

export type RequestStatus =
  | 'Draft'
  | 'Needs your answers'
  | 'Planning'
  | 'Planning failed'
  | 'Ready to accept'
  | 'Building'
  | 'Built';

interface RequestRow {
  intent: string;
  objective: string;
  created: number;
  status: RequestStatus;
  /** A draft started here whose repository is not confirmed yet. */
  unconfirmed: boolean;
}

// Drafts started in a repository, per repository id, in this browser only
// (SPEC M8: a convenience; a cleared browser shows such a draft only after
// Choose repository).
const startedKey = (repository: string) => `soda-request-started:v1:${repository}`;
const startedLimit = 50;

function readStarted(repository: string): string[] {
  try {
    const value: unknown = JSON.parse(localStorage.getItem(startedKey(repository)) ?? '[]');
    return Array.isArray(value)
      ? value.filter((entry): entry is string => typeof entry === 'string' && /^d[0-9a-f]{24}$/.test(entry))
      : [];
  } catch {
    return [];
  }
}

function writeStarted(repository: string, intents: string[]) {
  try {
    localStorage.setItem(startedKey(repository), JSON.stringify(intents.slice(-startedLimit)));
  } catch {
    /* Storage is a convenience only. */
  }
}

function forgetStarted(repository: string, intents: ReadonlySet<string>) {
  if (!intents.size) return;
  writeStarted(
    repository,
    readStarted(repository).filter((entry) => !intents.has(entry))
  );
}

class ReadError extends Error {
  constructor(readonly status: number) {
    super(`Read failed (${status})`);
  }
}

async function getJSON(context: PlaceContext, path: string, signal: AbortSignal): Promise<unknown> {
  const response = await context.transport.request(path, {method: 'GET', signal});
  if (!response.ok) throw new ReadError(response.status);
  return readSodaJSON(response);
}

/** Every saved request bound to this repository, following the list cursor to the end. */
async function readBound(context: PlaceContext, signal: AbortSignal): Promise<SavedRequest[]> {
  const all: SavedRequest[] = [];
  let cursor: string | undefined;
  for (let page = 0; page < 40; page++) {
    const query = new URLSearchParams({limit: '50', repository: context.repository});
    if (cursor) query.set('cursor', cursor);
    const list = intentList(await getJSON(context, `factory/intents?${query}`, signal));
    all.push(...list.intents.filter((plan) => plan.repository === context.repository));
    cursor = list.next_cursor;
    if (!cursor) break;
  }
  return all;
}

async function draftStatus(context: PlaceContext, plan: SavedRequest, signal: AbortSignal): Promise<RequestStatus> {
  try {
    const status = intentPlanStatus(await getJSON(context, `factory/intents/${plan.intent}/plan`, signal));
    if (status.revision !== plan.revision) return 'Draft';
    if (status.state === 'pending') return 'Planning';
    if (status.state === 'attention') return 'Needs your answers';
    if (status.state === 'failed' || status.state === 'cancelled') return 'Planning failed';
    if (status.state === 'completed') return status.walkthroughs ? 'Ready to accept' : 'Planning failed';
  } catch {
    /* An unreadable plan status stays neutral. */
  }
  return 'Draft';
}

async function acceptedStatus(context: PlaceContext, plan: SavedRequest, signal: AbortSignal): Promise<RequestStatus> {
  try {
    const overview = requestOverview(
      await getJSON(context, `factory/intents/${plan.intent}/overview`, signal),
      plan.intent
    );
    const items = overview.work_items;
    return items.total > 0 && items.delivered >= items.total ? 'Built' : 'Building';
  } catch {
    return 'Building';
  }
}

async function rowFor(context: PlaceContext, plan: SavedRequest, signal: AbortSignal): Promise<RequestRow> {
  let status: RequestStatus = 'Draft';
  if (plan.state === 'accepted') status = await acceptedStatus(context, plan, signal);
  else if (plan.repository !== '0') status = await draftStatus(context, plan, signal);
  return {
    intent: plan.intent,
    objective: plan.objective,
    created: plan.created_unix,
    status,
    unconfirmed: plan.repository === '0',
  };
}

/** The viewer's drafts started here that are still unbound; forgets the ones that moved on. */
async function readStartedDrafts(
  context: PlaceContext,
  bound: ReadonlySet<string>,
  signal: AbortSignal
): Promise<SavedRequest[]> {
  const drafts: SavedRequest[] = [];
  const forget = new Set<string>();
  for (const intent of readStarted(context.repository).reverse()) {
    if (bound.has(intent)) {
      forget.add(intent);
      continue;
    }
    try {
      const revision = intentRevision(await getJSON(context, `factory/intents/${intent}`, signal));
      if (revision.repository !== '0' || revision.state !== 'draft') {
        forget.add(intent);
        continue;
      }
      drafts.push({
        intent: revision.intent,
        revision: revision.id,
        state: revision.state,
        objective: revision.objective.slice(0, 280),
        repository: '0',
        // Newest first, above the confirmed requests.
        created_unix: Number.MAX_SAFE_INTEGER - drafts.length,
        effective_count: 0,
        task_count: revision.tasks.length,
      });
    } catch (error) {
      if (signal.aborted) throw error;
      if (error instanceof ReadError && error.status === 404) forget.add(intent);
    }
  }
  forgetStarted(context.repository, forget);
  return drafts;
}

/** Map with at most `width` reads in flight. */
async function pooled<T, R>(items: readonly T[], width: number, read: (item: T) => Promise<R>): Promise<R[]> {
  const out: R[] = [];
  let next = 0;
  const lane = async () => {
    while (next < items.length) {
      const index = next++;
      out[index] = await read(items[index] as T);
    }
  };
  await Promise.all(Array.from({length: Math.min(width, items.length)}, lane));
  return out;
}

function startLine(context: PlaceContext): TemplateResult {
  const viewer = context.viewer;
  if (!viewer.loaded) return html``;
  const feed = html`<a href=${context.places.feed()}>Follow the work in Feed.</a>`;
  if (viewer.admin)
    return html`<a class="ui primary button" href=${context.places.newRequest()}
      ><span aria-hidden="true">＋</span> New request</a
    >`;
  if (viewer.codeWrite)
    return html`<p class="soda-requests-note">Only repository administrators can start Factory work here. ${feed}</p>`;
  return html`<p class="soda-requests-note">Requests are private to the person who made them. ${feed}</p>`;
}

function renderRow(context: PlaceContext, row: RequestRow): TemplateResult {
  const status = row.unconfirmed ? 'Draft · repository not yet confirmed' : row.status;
  return html`<li class="soda-request-row" data-intent-id=${row.intent}>
    <a href=${context.places.request(row.intent)}>${row.objective || 'Untitled request'}</a>
    <span class="soda-work-state" data-state=${row.status.toLowerCase().replaceAll(' ', '-')}>${status}</span>
  </li>`;
}

export const mountRequestsPlace: PlaceMount = (host, context) => {
  let rows: RequestRow[] = [];
  let loaded = false;
  let error = '';
  let controller: AbortController | undefined;
  let disposed = false;

  const draw = () => {
    if (disposed) return;
    render(
      html`<section class="soda-requests" aria-label="Requests" aria-busy=${controller ? 'true' : 'false'}>
        <header class="soda-requests-header">
          <h2 tabindex="-1">Requests</h2>
          ${startLine(context)}
        </header>
        <p class="soda-requests-help">Your requests for ${context.slug}. Only you see them.</p>
        ${error ? html`<p class="soda-work-feedback" role="status">${error}</p>` : ''}
        ${!loaded && !error ? html`<p class="soda-work-feedback" role="status">Loading requests…</p>` : ''}
        ${loaded && !rows.length ? html`<p>You have no requests in this repository yet.</p>` : ''}
        ${
          rows.length
            ? html`<ul class="soda-request-rows">
                ${rows.map((row) => renderRow(context, row))}
              </ul>`
            : ''
        }
        ${
          loaded || error
            ? html`<button type="button" class="ui button" ?disabled=${!!controller} @click=${load}>
                Refresh requests
              </button>`
            : ''
        }
      </section>`,
      host
    );
  };

  async function load() {
    controller?.abort();
    const request = new AbortController();
    controller = request;
    error = '';
    draw();
    const timeout = window.setTimeout(() => request.abort(), 30000);
    try {
      const bound = await readBound(context, request.signal);
      const drafts = await readStartedDrafts(context, new Set(bound.map((plan) => plan.intent)), request.signal);
      const listed = [...drafts, ...bound].filter((plan) => plan.state === 'draft' || plan.state === 'accepted');
      const next = await pooled(listed, 4, (plan) => rowFor(context, plan, request.signal));
      if (controller !== request) return;
      rows = next.sort((a, b) => b.created - a.created);
      loaded = true;
    } catch {
      if (controller !== request) return;
      error = rows.length
        ? 'Could not refresh your requests. Showing the ones loaded before.'
        : 'Could not load your requests. Try again.';
    } finally {
      window.clearTimeout(timeout);
      if (controller === request) controller = undefined;
      draw();
    }
  }

  void load();
  return {
    // The viewer's standing arrives after mount; redraw the start line.
    update: draw,
    dispose() {
      disposed = true;
      controller?.abort();
      controller = undefined;
      render(html``, host);
    },
  };
};
