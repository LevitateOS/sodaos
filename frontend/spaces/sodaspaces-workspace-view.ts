import {html} from 'lit';
import type {TemplateResult} from 'lit';
import {repeat} from 'lit/directives/repeat.js';
import type {RepositoryChoices} from './sodaspaces-api.js';

/** Welcome artwork is decorative, theme-bound vector geometry, not a control. */
export function renderWelcome(blocked: boolean, create: () => void) {
  return html`
    <svg class="soda-welcome-illustration" viewBox="0 0 112 112" aria-hidden="true" focusable="false">
      <path class="soda-welcome-accent" d="M79 19 102 32 79 45 56 32Z"></path>
      <path class="soda-welcome-outline" d="M56 6 102 32V80L56 106 10 80V32ZM10 32 56 58 102 32M56 58V106"></path>
    </svg>
    <h2 tabindex="-1">Create your first project</h2>
    <p class="soda-welcome-copy"><span>A shared development system, connected to your repository.</span><span>Open terminals and work together, right in your browser.</span></p>
    <button class="ui primary button" ?disabled=${blocked} @click=${create}><span aria-hidden="true">＋</span> Create project</button>
    <p class="soda-welcome-help"><a href="https://github.com/levitateos/sodaos/blob/main/docs/public/30-Use-Soda/20-projects-and-workspaces.md" target="_blank" rel="noopener noreferrer">How Spaces works <span aria-hidden="true">↗</span><span class="soda-visually-hidden"> (opens in a new tab)</span></a></p>`;
}

/** The two post-creation prompts share one composition and illustration area. */
export function renderWorkspaceIntro(view: {kind: 'project' | 'terminal' | 'unavailable' | 'stopped' | 'loading'; heading: string; description: TemplateResult; action: TemplateResult; helper: TemplateResult; feedback?: TemplateResult}) {
  return html`<div class="soda-workspace-intro" data-state=${view.kind}><div class="soda-workspace-intro-content">
    <svg class="soda-workspace-illustration" viewBox="0 0 128 104" aria-hidden="true" focusable="false">
      <g ?hidden=${view.kind !== 'project'}><path class="soda-welcome-outline" d="M58 10 94 31V73L58 94 22 73V31ZM22 31 58 52 94 31M58 52V94"></path><circle class="soda-created-check" cx="98" cy="81" r="16"></circle><path class="soda-created-checkmark" d="m91 81 5 5 9-10"></path></g>
      <g ?hidden=${view.kind !== 'terminal'}><path class="soda-welcome-outline" d="M7 21H121V83H7ZM24 39 35 50 24 61"></path><path class="soda-terminal-cursor" d="M44 61H60"></path></g>
      <g ?hidden=${!['unavailable', 'stopped', 'loading'].includes(view.kind)}><circle class="soda-welcome-outline" cx="64" cy="52" r="34"></circle><path class="soda-welcome-outline" ?hidden=${view.kind !== 'unavailable'} d="M64 33V55M64 66V69"></path><path class="soda-welcome-outline" ?hidden=${view.kind !== 'stopped'} d="M48 52H80"></path><path class="soda-welcome-outline" ?hidden=${view.kind !== 'loading'} d="M64 30V52L79 61"></path></g>
    </svg>
    <h2 tabindex="-1">${view.heading}</h2>
    <p class="soda-intro-description">${view.description}</p>
    <div class="soda-intro-action">${view.action}</div>
    <p class="soda-intro-helper">${view.helper}</p>
    ${view.feedback || ''}
  </div></div>`;
}

export function renderWelcomeSteps(step?: 1 | 2) {
  return html`<ol class="soda-setup-footer soda-welcome-steps" aria-label=${step ? 'New project progress' : 'Getting started'} data-step=${step || 0}>
    <li aria-current=${step === 1 ? 'step' : 'false'}><span aria-hidden="true">${step === 2 ? '✓' : '01'}</span> Choose a repository</li>
    <li aria-current=${step === 2 ? 'step' : 'false'}><span aria-hidden="true">02</span> Create a project</li>
    <li><span aria-hidden="true">03</span> Open a terminal</li>
  </ol>`;
}

export function renderRepositoryPicker(view: {query: string; result: RepositoryChoices | undefined; selected: string; busy: boolean; error: string; blocked: boolean; createURL: string}, actions: {query: (value: string) => void; search: (page: number) => void; select: (id: string) => void; back: () => void; continue: () => void}) {
  const choice = view.result?.items.find(item => item.id === view.selected);
  return html`<header class="soda-setup-step-heading"><p class="soda-setup-eyebrow">New project</p><h2 tabindex="-1">Choose a repository</h2>
    <p>Choose a repository you own on this Forgejo.</p></header>
    <form class="soda-repository-search" @submit=${(e: SubmitEvent) => {e.preventDefault(); actions.search(1);}}>
      <label><span class="soda-visually-hidden">Search repositories</span><input placeholder="Search repositories…" type="search" maxlength="200" .value=${view.query} ?disabled=${view.blocked} @input=${(e: Event) => {if (e.target instanceof HTMLInputElement) actions.query(e.target.value);}}></label>
      <button class="ui button" ?disabled=${view.busy || view.blocked}>Search</button>
    </form>
    <div class="soda-repository-notice soda-feedback" data-tone=${view.error ? 'warning' : 'pending'} role="status">${view.busy ? 'Finding repositories…' : view.error}</div>
    <fieldset class="soda-repository-results" ?disabled=${view.busy || view.blocked}><legend>Available repositories</legend><div class="soda-repository-rows">
      ${view.result?.items.map(item => html`<label class="soda-repository-choice">
        <input type="radio" name="soda-repository" .checked=${view.selected === item.id} @change=${() => actions.select(item.id)}>
        <span class="soda-journey-icon soda-repository-icon" aria-hidden="true"></span><span>${item.owner}/${item.name}<small>${item.project ? item.project.provisioned ? 'Project exists' : 'Provisioning needs inspection' : ''}</small></span>
      </label>`)}
      ${view.result && !view.result.items.length ? html`<p>${view.query ? 'No matching eligible repositories. Clear your search or create a repository.' : 'No eligible repositories on this page. Only a human repository owner can create its project.'}</p>` : ''}
    </div></fieldset>
    ${view.result && (view.result.more || view.result.page > 1) ? html`<nav aria-label="Repository pages"><button class="ui button" ?disabled=${view.busy || view.blocked || view.result.page <= 1} @click=${() => actions.search((view.result?.page || 1) - 1)}>Previous repositories</button><span>Page ${view.result.page}</span><button class="ui button" ?disabled=${view.busy || view.blocked || !view.result.more} @click=${() => actions.search((view.result?.page || 1) + 1)}>Next repositories</button></nav>` : ''}
    ${view.result?.limited ? html`<p>Search limit reached. Narrow your search.</p>` : ''}
    <div class="soda-setup-actions"><a href=${view.createURL}>Create a new repository</a><button class="ui primary button" ?disabled=${view.busy || view.blocked || !choice} @click=${actions.continue}>${choice?.project ? choice.project.provisioned ? 'Open project' : 'Inspect project' : 'Continue'}</button></div>`;
}

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
      draggable=${draggable ? 'true' : 'false'} title=${tab.name + ' · ' + tab.project}
      aria-label=${tab.name + ' · ' + tab.project} aria-controls=${'soda-owner-' + tab.key}
      tabindex=${tab.selected ? '0' : '-1'} aria-selected=${tab.selected ? 'true' : 'false'}
      @dragstart=${tab.dragstart} @dragend=${tab.dragend} @dragover=${dragover} @drop=${tab.drop}
      @keydown=${tab.keydown} @click=${tab.select}>
      <span class="soda-tab-icon" aria-hidden="true"></span><span class="soda-tab-name">${tab.name}</span>${tab.unread ? html`<span class="soda-unread" aria-label="Unread output"> •</span>` : ''}
      ${tab.attention ? html`<span class="soda-attention" title=${tab.attention} aria-label=${tab.attention}> !</span>` : ''}
    </button>
  `;
}
export interface NavigationSession {
  readonly key: string;
  readonly terminalId: string;
  readonly active: boolean;
  readonly name: string;
  readonly description: string;
  readonly unread: boolean;
  readonly attention: string;
  readonly disabled: boolean;
  readonly select: () => void;
}
export function renderProjectNavigation(name: string, status: string, rows: readonly NavigationSession[], disabled: boolean, details: () => void, selection?: {active: boolean; environmentId: string; state: 'running' | 'stopped' | 'unknown'; select: () => void}): TemplateResult {
  return html`
    <section class="soda-project-group" data-environment-id=${selection?.environmentId || ''}>
      ${selection ? html`<button class="ui button soda-project-select" aria-label=${name} aria-pressed=${selection.active ? 'true' : 'false'} aria-describedby=${'soda-project-status-' + selection.environmentId} ?disabled=${disabled} @click=${selection.select}><span class="soda-journey-icon soda-repository-icon" aria-hidden="true"></span><span class="soda-project-select-body"><span>${name}</span><small class="soda-project-status" data-state=${selection.state} id=${'soda-project-status-' + selection.environmentId}>${status}</small></span></button>` : ''}
      <details ?hidden=${!!selection && rows.length === 0} ?open=${rows.length > 0}>
        <summary ?hidden=${!!selection}>${selection ? 'Terminals' : name} ${!selection ? html`<small>${status}</small>` : ''}</summary>
        <div class="soda-session-list">
          ${repeat(rows, row => row.key, row => html`
            <button class="ui button" data-terminal-id=${row.terminalId} aria-current=${row.active ? 'true' : 'false'} ?disabled=${row.disabled} @click=${row.select}>
              <span>${row.name}${row.unread ? html`<span class="soda-unread" aria-label="Unread output"> •</span>` : ''}</span>
              <small>${row.description}</small>
              ${row.attention ? html`<small class="soda-attention">${row.attention}</small>` : ''}
            </button>
          `)}
        </div>
        <p ?hidden=${rows.length !== 0}>No terminals observed.</p>
      </details>
      ${!selection ? html`<button class="ui button soda-project-details" ?disabled=${disabled} @click=${details}>Environment / access: ${name}</button>` : ''}
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
