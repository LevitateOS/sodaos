import {html} from 'lit';
import type {TemplateResult} from 'lit';
import {repeat} from 'lit/directives/repeat.js';

/** Stateless chrome. Callbacks capture the owner's original entry/pane/project. */
export function renderMenu(label: string, glyph: string, body: TemplateResult, className = ''): TemplateResult {
  return html`
    <details class=${'soda-menu' + (className ? ' ' + className : '')}>
      <summary aria-label=${label} title=${label}>${glyph}</summary>
      <div>${body}</div>
    </details>
  `;
}
export interface SessionTab {
  readonly key: string;
  readonly name: string;
  readonly project: string;
  readonly selected: boolean;
  readonly unread: boolean;
  readonly attention: string;
  readonly select: () => void;
  readonly keydown: (event: KeyboardEvent) => void;
  readonly dragstart: (event: DragEvent) => void;
  readonly dragend: () => void;
  readonly drop: (event: DragEvent) => void;
}
export function renderSessionTab(tab: SessionTab, navigation: boolean, draggable: boolean, dragover: (event: DragEvent) => void): TemplateResult {
  return html`
    <button id=${(navigation ? 'soda-navtab-' : 'soda-tab-') + tab.key} role="tab" class="ui button"
      draggable=${draggable ? 'true' : 'false'} title=${tab.project}
      aria-label=${tab.name + ' · ' + tab.project} aria-controls=${'soda-owner-' + tab.key}
      tabindex=${tab.selected ? '0' : '-1'} aria-selected=${tab.selected ? 'true' : 'false'}
      @dragstart=${tab.dragstart} @dragend=${tab.dragend} @dragover=${dragover} @drop=${tab.drop}
      @keydown=${tab.keydown} @click=${tab.select}>
      ${tab.name}${tab.unread ? html`<span class="soda-unread" aria-label="Unread output"> •</span>` : ''}
      ${tab.attention ? html`<span class="soda-attention" title=${tab.attention} aria-label=${tab.attention}> !</span>` : ''}
    </button>
  `;
}
export interface NavigationSession {
  readonly key: string;
  readonly name: string;
  readonly description: string;
  readonly unread: boolean;
  readonly attention: string;
  readonly disabled: boolean;
  readonly select: () => void;
}
export function renderProjectNavigation(name: string, status: string, rows: readonly NavigationSession[], disabled: boolean, details: () => void): TemplateResult {
  return html`
    <section class="soda-project-group">
      <details ?open=${rows.length > 0}>
        <summary>${name} <small>${status}</small></summary>
        <div class="soda-session-list">
          ${repeat(rows, row => row.key, row => html`
            <button class="ui button" ?disabled=${row.disabled} @click=${row.select}>
              <span>${row.name}${row.unread ? html`<span class="soda-unread" aria-label="Unread output"> •</span>` : ''}</span>
              <small>${row.description}</small>
              ${row.attention ? html`<small class="soda-attention">${row.attention}</small>` : ''}
            </button>
          `)}
        </div>
        <p ?hidden=${rows.length !== 0}>No sessions yet.</p>
      </details>
      <button class="ui button soda-project-details" ?disabled=${disabled} @click=${details}>Environment / access: ${name}</button>
    </section>
  `;
}
export function renderRename(name: string, project: string, disabled: boolean, change: (value: string) => void, cancel: () => void, save: () => void): TemplateResult {
  return html`
    <form class="soda-workspace-dialog" role="dialog" aria-label="Rename terminal"
      @submit=${(event: SubmitEvent) => {event.preventDefault(); save();}}>
      <label>Session name
        <input maxlength="160" .value=${name}
          @input=${(event: Event) => {if (event.target instanceof HTMLInputElement) change(event.target.value);}}>
      </label>
      <p>${project}</p>
      <button type="button" class="ui button" @click=${cancel}>Cancel</button>
      <button class="ui button" ?disabled=${disabled}>Save name</button>
    </form>
  `;
}
export interface CreationPresentation {
  readonly environmentId: string;
  readonly name: string;
  readonly projects: readonly {id: string; name: string}[];
  readonly context: string;
  readonly explanation: string;
  readonly busy: boolean;
  readonly disabled: boolean;
}
export function renderCreation(view: CreationPresentation, project: (id: string) => void, name: (value: string) => void, cancel: () => void, create: () => void): TemplateResult {
  return html`
    <form class="soda-workspace-dialog" role="dialog" aria-label="New terminal"
      @submit=${(event: SubmitEvent) => {event.preventDefault(); create();}}
      @keydown=${(event: KeyboardEvent) => {if (event.key === 'Escape') {event.stopPropagation(); cancel();}}}>
      <label>Project
        <select aria-label="Project" .value=${view.environmentId} ?disabled=${view.busy}
          @change=${(event: Event) => {if (event.target instanceof HTMLSelectElement) project(event.target.value);}}>
          ${view.projects.map(item => html`<option value=${item.id} ?selected=${item.id === view.environmentId}>${item.name}</option>`)}
        </select>
      </label>
      <p>${view.context}</p>
      <label>Terminal name
        <input maxlength="160" .value=${view.name} ?disabled=${view.busy}
          @input=${(event: Event) => {if (event.target instanceof HTMLInputElement) name(event.target.value);}}>
      </label>
      <p ?hidden=${!view.explanation}>${view.explanation} Use named Environment / access; nothing is provisioned by this chooser.</p>
      <button type="button" class="ui button" @click=${cancel}>Cancel</button>
      <button class="ui primary button" ?disabled=${view.disabled}>Create terminal</button>
    </form>
  `;
}
