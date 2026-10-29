import {LitElement, html} from 'lit';
import type {PropertyValues} from 'lit';
import {check, object, readSodaJSON} from './sodaspaces-api.js';
import type {FeedArtifactRequest} from './sodaspaces-factory-feed-artifact.js';

interface LibraryItem {
  id: string;
  revision: number;
  kind: string;
  title: string;
  body: string;
  scope: string;
  decision: string;
  rationale: string;
  alternatives: string[];
  evidence: string[];
  role: string;
  state: string;
  updated: number;
  source: string;
  removed: boolean;
  replacements: string[];
  derived: string[];
  relations: {kind: string; target: string; evidence: string[]}[];
}
interface LibraryHistory {
  item: LibraryItem;
  reason: string;
  actor: string;
}
const strings = (value: unknown): string[] =>
  Array.isArray(value) ? value.filter((v): v is string => typeof v === 'string').slice(0, 24) : [];
const text = (value: unknown): string => (typeof value === 'string' ? value : '');
function parseItem(value: unknown): LibraryItem {
  const item = object(value);
  check(typeof item.id === 'string' && /^[a-f0-9]{32}$/.test(item.id));
  check(typeof item.revision === 'number' && item.revision > 0 && typeof item.title === 'string');
  return {
    id: item.id,
    revision: item.revision,
    kind: text(item.kind),
    title: item.title,
    body: text(item.body),
    scope: text(item.scope),
    decision: text(item.decision),
    rationale: text(item.rationale),
    alternatives: strings(item.alternatives),
    evidence: strings(item.evidence_refs),
    role: text(item.owner_role).replaceAll('_', ' '),
    state: text(item.state),
    updated: Number(item.updated_at) || 0,
    source: text(item.source_post),
    removed: item.removed === true,
    replacements: strings(item.replacements),
    derived: strings(item.derived_from),
    relations: Array.isArray(item.relations)
      ? item.relations.slice(0, 24).map((value) => {
          const relation = object(value);
          return {kind: text(relation.kind), target: text(relation.target), evidence: strings(relation.evidence)};
        })
      : [],
  };
}

export class SodaFactoryLibrary extends LitElement {
  static properties = {
    repository: {attribute: false},
    request: {attribute: false},
    itemID: {attribute: false},
    pendingSource: {attribute: false},
    view: {state: true},
  };
  declare repository: string;
  declare request: FeedArtifactRequest | undefined;
  declare itemID: string | undefined;
  declare pendingSource: string | undefined;
  declare private view: {
    busy: boolean;
    message: string;
    items: LibraryItem[];
    next: string;
    item: LibraryItem | undefined;
    history: LibraryHistory[];
  };
  private query = '';
  private kind = '';
  private history = false;
  private serial = 0;
  constructor() {
    super();
    this.view = {busy: false, message: '', items: [], next: '', item: undefined, history: []};
  }
  protected createRenderRoot() {
    return this;
  }
  protected willUpdate(changed: PropertyValues) {
    if (changed.has('repository') || changed.has('request') || changed.has('itemID')) {
      if (this.itemID) void this.openItem(this.itemID);
      else void this.search();
    }
  }
  disconnectedCallback() {
    super.disconnectedCallback();
    this.serial++;
  }
  // Every library navigation supersedes an in-flight feed source open: the
  // feed listens for library-navigate and drops the delayed response so it
  // can never open its thread.
  private notifyNavigation() {
    this.dispatchEvent(new CustomEvent('library-navigate', {bubbles: true, composed: true}));
  }
  private async read(path: string) {
    if (!this.request) throw Error('The project library is unavailable.');
    const response = await this.request(path, {method: 'GET', signal: AbortSignal.timeout(15000)});
    if (!response.ok)
      throw Error(
        response.status === 404 || response.status === 403
          ? 'This library item is not available to your account.'
          : 'The library could not be loaded. Try again.'
      );
    return object(await readSodaJSON(response));
  }
  private async search(more = false) {
    if (!this.request || !/^[1-9][0-9]*$/.test(this.repository)) return;
    this.notifyNavigation();
    const serial = ++this.serial;
    this.view = {...this.view, busy: true, message: '', item: undefined, history: []};
    const params = new URLSearchParams({repository: this.repository, query: this.query, kind: this.kind});
    if (this.history) params.set('history', 'true');
    if (more && this.view.next) params.set('cursor', this.view.next);
    try {
      const page = await this.read(`factory/library?${params}`);
      check(Array.isArray(page.items));
      const items = page.items.map(parseItem);
      if (serial !== this.serial) return;
      this.view = {
        ...this.view,
        busy: false,
        items: more ? [...this.view.items, ...items] : items,
        next: text(page.next_cursor),
      };
    } catch (error) {
      if (serial === this.serial)
        this.view = {
          ...this.view,
          busy: false,
          message: error instanceof Error ? error.message : 'The library could not be loaded.',
        };
    }
  }
  private async openItem(id: string) {
    if (!/^[a-f0-9]{32}$/.test(id)) return;
    this.notifyNavigation();
    const serial = ++this.serial;
    this.view = {...this.view, busy: true, message: ''};
    try {
      const data = await this.read(
        `factory/library/item?${new URLSearchParams({repository: this.repository, id, history: 'true'})}`
      );
      const item = parseItem(data.item);
      const history = Array.isArray(data.history)
        ? data.history.map((value) => {
            const revision = object(value);
            return {item: parseItem(revision.item), reason: text(revision.reason), actor: text(revision.actor)};
          })
        : [];
      if (serial !== this.serial) return;
      this.view = {...this.view, busy: false, item, history};
    } catch (error) {
      if (serial === this.serial)
        this.view = {
          ...this.view,
          busy: false,
          message: error instanceof Error ? error.message : 'The library item could not be loaded.',
        };
    }
  }
  private related(label: string, ids: string[]) {
    if (!ids.length) return '';
    return html`<section>
      <h4>${label}</h4>
      <div class="soda-library-relations">
        ${ids.map((id) => html`<button type="button" @click=${() => void this.openItem(id)}>Open item ${id.slice(0, 8)}</button>`)}
      </div>
    </section>`;
  }
  private content(item: LibraryItem) {
    return html`<div class="soda-library-body">
      <p>${item.body}</p>
      ${
        item.decision
          ? html`<section>
              <h4>Decision</h4>
              <p>${item.decision}</p>
            </section>`
          : ''
      }
      ${
        item.rationale
          ? html`<section>
              <h4>Reason</h4>
              <p>${item.rationale}</p>
            </section>`
          : ''
      }
      ${
        item.alternatives.length
          ? html`<section>
              <h4>Alternatives considered</h4>
              <ul>
                ${item.alternatives.map((alternative) => html`<li>${alternative}</li>`)}
              </ul>
            </section>`
          : ''
      }
      ${
        item.evidence.length
          ? html`<details>
              <summary>Evidence · ${item.evidence.length}</summary>
              <ul>
                ${item.evidence.map((ref) => html`<li><code>${ref}</code></li>`)}
              </ul>
            </details>`
          : ''
      }
    </div>`;
  }
  private sourceLink(item: LibraryItem) {
    if (!/^[a-f0-9]{32}$/.test(item.source)) return '';
    const pending = this.pendingSource === item.source;
    return html`<button
      type="button"
      ?disabled=${pending}
      @click=${() => this.dispatchEvent(new CustomEvent('library-source-open', {bubbles: true, composed: true, detail: {repository: this.repository, id: item.source, item: item.id, revision: item.revision}}))}
    >
      ${pending ? 'Opening source…' : 'Open source conversation'}
    </button>`;
  }
  protected render() {
    const item = this.view.item;
    return html`<section class="soda-library" aria-label="Project knowledge and decisions" aria-busy=${this.view.busy}>
      <p class="soda-library-intro">Shared knowledge and decisions for this project.</p>
      <form
        class="soda-library-search"
        @submit=${(event: Event) => {
          event.preventDefault();
          void this.search();
        }}
      >
        <label
          >Search the library<input
            type="search"
            maxlength="256"
            .value=${this.query}
            @input=${(event: Event) => {
              this.query = (event.target as HTMLInputElement).value;
              this.notifyNavigation();
            }}
        /></label>
        <label
          >Show<select
            aria-label="Item type"
            .value=${this.kind}
            @change=${(event: Event) => {
              this.kind = (event.target as HTMLSelectElement).value;
              void this.search();
            }}
          >
            <option value="">Knowledge and decisions</option>
            <option value="knowledge">Knowledge</option>
            <option value="decision">Decisions</option>
          </select></label
        >
        <button type="submit" ?disabled=${this.view.busy}>Search</button>
        <label
          ><input
            type="checkbox"
            .checked=${this.history}
            @change=${(event: Event) => {
              if (event.target instanceof HTMLInputElement) {
                this.history = event.target.checked;
                void this.search();
              }
            }}
          />Include removed items</label
        >
      </form>
      ${this.view.message ? html`<p role="alert">${this.view.message}</p>` : ''}
      ${this.view.busy ? html`<p role="status">Loading library…</p>` : ''}
      ${
        item
          ? html`<article class="soda-library-detail">
              <button type="button" class="soda-library-back" @click=${() => void this.search()}>
                ← Back to results
              </button>
              <div class="soda-library-item-meta">
                <span>${item.kind === 'decision' ? 'Decision' : 'Knowledge'} · revision ${item.revision}</span
                ><span>${item.removed ? 'Removed' : item.state}</span>
              </div>
              <h3>${item.title}</h3>
              <p class="soda-library-scope">${item.scope}</p>
              <p class="soda-library-item-meta">
                ${item.role}${item.updated ? ` · ${new Date(item.updated * 1000).toLocaleString()}` : ''}
              </p>
              ${this.sourceLink(item)}${this.content(item)}${this.related('Replaced by', item.replacements)}${this.related('Derived from', item.derived)}
              ${
                item.relations.length
                  ? html`<section>
                      <h4>Related items</h4>
                      <ul class="soda-library-relations">
                        ${item.relations.map((relation) => html`<li><button type="button" @click=${() => void this.openItem(relation.target)}>${relation.kind.replaceAll('_', ' ')} · ${relation.target.slice(0, 8)}</button>${relation.evidence.length ? html`<p>${relation.evidence.join(' · ')}</p>` : ''}</li>`)}
                      </ul>
                    </section>`
                  : ''
              }
              <details class="soda-library-history">
                <summary>Recent revisions · ${this.view.history.length}</summary>
                ${this.view.history.map(
                  (revision) =>
                    html`<details>
                      <summary>
                        Revision ${revision.item.revision} · ${revision.reason || 'Recorded'} · ${revision.actor}
                      </summary>
                      <h4>${revision.item.title}</h4>
                      <p>${revision.item.scope}</p>
                      ${this.sourceLink(revision.item)}${this.content(revision.item)}
                    </details>`
                )}
              </details>
            </article>`
          : html`<div class="soda-library-results">
                ${this.view.items.map(
                  (entry) =>
                    html`<article>
                      <div class="soda-library-item-meta">
                        <span>${entry.kind === 'decision' ? 'Decision' : 'Knowledge'} · revision ${entry.revision}</span
                        ><span>${entry.state}</span>
                      </div>
                      <h3>
                        <button type="button" @click=${() => void this.openItem(entry.id)}>${entry.title}</button>
                      </h3>
                      <p class="soda-library-excerpt">${entry.body}</p>
                      <p class="soda-library-item-meta">${entry.scope} · ${entry.role}</p>
                    </article>`
                )}
              </div>
              ${!this.view.busy && !this.view.message && !this.view.items.length ? html`<p class="soda-library-empty">${this.query || this.kind ? 'No matching items. Try a different search.' : 'No knowledge or decisions recorded yet. Worker observations and decisions will appear here.'}</p>` : ''}${this.view.next ? html`<button type="button" ?disabled=${this.view.busy} @click=${() => void this.search(true)}>More library items</button>` : ''}`
      }
    </section>`;
  }
}
if (typeof customElements !== 'undefined' && !customElements.get('soda-factory-library'))
  customElements.define('soda-factory-library', SodaFactoryLibrary);
declare global {
  interface HTMLElementTagNameMap {
    'soda-factory-library': SodaFactoryLibrary;
  }
}
