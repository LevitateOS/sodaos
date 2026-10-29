import {html} from 'lit';
import type {TemplateResult} from 'lit';
import {repeat} from 'lit/directives/repeat.js';

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
  readonly dragover?: (event: DragEvent) => void;
  readonly drop: (event: DragEvent) => void;
  /** Page tabs: end the terminal (asks first). */
  readonly close?: () => void;
  readonly rename?: () => void;
  readonly dragging?: boolean;
  /** Where the drop insertion line shows while another tab is dragged. */
  readonly dropSide?: 'before' | 'after' | '';
}
/**
 * A page tab: the tab button plus its own close button (a button cannot hold
 * another). Middle-click or Delete also close; double-click renames.
 */
function renderPageTab(tab: SessionTab, draggable: boolean): TemplateResult {
  return html`<div
    class="soda-tab"
    data-selected=${tab.selected ? 'true' : 'false'}
    data-dragging=${tab.dragging ? 'true' : 'false'}
    data-drop=${tab.dropSide || ''}
    @dragover=${tab.dragover}
    @drop=${tab.drop}
  >
    <button
      id=${'soda-tab-' + tab.key}
      role="tab"
      class="soda-tab-main"
      draggable=${draggable ? 'true' : 'false'}
      title=${tab.name + ' · ' + tab.project + ' — double-click to rename'}
      aria-label=${tab.name + ' · ' + tab.project}
      aria-controls=${'soda-owner-' + tab.key}
      aria-keyshortcuts="Delete Control+Shift+ArrowLeft Control+Shift+ArrowRight"
      tabindex=${tab.selected ? '0' : '-1'}
      aria-selected=${tab.selected ? 'true' : 'false'}
      @dragstart=${tab.dragstart}
      @dragend=${tab.dragend}
      @keydown=${tab.keydown}
      @click=${tab.select}
      @dblclick=${tab.rename}
      @mouseup=${(event: MouseEvent) => {
        if (event.button === 1) {
          event.preventDefault();
          tab.close?.();
        }
      }}
    >
      <span class="soda-tab-icon" aria-hidden="true"></span
      ><span class="soda-tab-name">${tab.name}</span
      >${tab.unread ? html`<span class="soda-unread" role="img" aria-label="Unread output"></span>` : ''}
      ${tab.attention ? html`<span class="soda-attention" role="img" title=${tab.attention} aria-label=${tab.attention}>!</span>` : ''}
    </button>
    <button
      type="button"
      class="soda-tab-close"
      tabindex="-1"
      aria-label=${'End ' + tab.name}
      title="End terminal (Delete)"
      @click=${(event: Event) => {
        event.stopPropagation();
        tab.close?.();
      }}
    >
      <svg width="10" height="10" viewBox="0 0 10 10" aria-hidden="true">
        <path d="M1.5 1.5l7 7M8.5 1.5l-7 7" stroke="currentColor" stroke-width="1.6" />
      </svg>
    </button>
  </div>`;
}

export function renderSessionTab(
  tab: SessionTab,
  navigation: boolean,
  draggable: boolean,
  dragover: (event: DragEvent) => void
): TemplateResult {
  if (!navigation && tab.close) return renderPageTab(tab, draggable);
  return html`
    <button
      id=${(navigation ? 'soda-navtab-' : 'soda-tab-') + tab.key}
      role="tab"
      class="ui button"
      draggable=${draggable ? 'true' : 'false'}
      title=${tab.name + ' · ' + tab.project}
      aria-label=${tab.name + ' · ' + tab.project}
      aria-controls=${'soda-owner-' + tab.key}
      tabindex=${tab.selected ? '0' : '-1'}
      aria-selected=${tab.selected ? 'true' : 'false'}
      @dragstart=${tab.dragstart}
      @dragend=${tab.dragend}
      @dragover=${dragover}
      @drop=${tab.drop}
      @keydown=${tab.keydown}
      @click=${tab.select}
    >
      <span class="soda-tab-icon" aria-hidden="true"></span
      ><span class="soda-tab-name">${tab.name}</span
      >${tab.unread ? html`<span class="soda-unread" role="img" aria-label="Unread output"> •</span>` : ''}
      ${tab.attention ? html`<span class="soda-attention" role="img" title=${tab.attention} aria-label=${tab.attention}> !</span>` : ''}
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
export interface ProjectDirectoryRow {
  readonly environmentId: string;
  readonly name: string;
  readonly state: 'running' | 'stopped' | 'unknown';
  readonly attention: number;
  readonly active: boolean;
  readonly disabled: boolean;
  readonly select: () => void;
}
export function renderProjectDirectoryRow(project: ProjectDirectoryRow): TemplateResult {
  const separator = project.name.lastIndexOf('/');
  const name = separator < 0 ? project.name : project.name.slice(separator + 1);
  const owner = separator < 0 ? '' : project.name.slice(0, separator);
  const status =
    project.state === 'unknown' ? 'Status unavailable' : project.state === 'running' ? 'Running' : 'Stopped';
  const attention =
    project.attention === 1 ? '1 terminal needs attention' : `${project.attention} terminals need attention`;
  return html`<section class="soda-project-group" data-environment-id=${project.environmentId}>
    <button
      class="ui button soda-project-select"
      aria-label=${project.name}
      aria-pressed=${project.active ? 'true' : 'false'}
      aria-describedby=${'soda-project-directory-status-' + project.environmentId}
      ?disabled=${project.disabled}
      @click=${project.select}
    >
      <span class="soda-journey-icon soda-repository-icon" aria-hidden="true"></span>
      <span class="soda-project-select-body">
        <span class="soda-project-name">${name}</span>
        <small class="soda-project-owner" ?hidden=${!owner}>${owner}</small>
        <small
          class="soda-project-status"
          data-state=${project.state}
          id=${'soda-project-directory-status-' + project.environmentId}
          >${status}</small
        >
      </span>
      ${project.attention ? html`<span class="soda-project-attention" aria-label=${attention} title=${attention}>${project.attention}</span>` : ''}
    </button>
  </section>`;
}
function renderSessionNavigationList(rows: readonly NavigationSession[]): TemplateResult {
  return html`<div class="soda-session-list">
      ${repeat(
        rows,
        (row) => row.key,
        (row) => html`
          <button
            class="ui button"
            data-terminal-id=${row.terminalId}
            aria-current=${row.active ? 'true' : 'false'}
            ?disabled=${row.disabled}
            @click=${row.select}
          >
            <span
              >${row.name}${row.unread ? html`<span class="soda-unread" role="img" aria-label="Unread output"> •</span>` : ''}</span
            >
            <small>${row.description}</small>
            ${row.attention ? html`<small class="soda-attention">${row.attention}</small>` : ''}
          </button>
        `
      )}
    </div>
    <p ?hidden=${rows.length !== 0}>No terminals observed.</p>`;
}
export function renderProjectTerminalList(rows: readonly NavigationSession[]): TemplateResult {
  return html`<section class="soda-project-terminals" aria-label="Terminals in this project">
    <p class="soda-menu-heading">Terminals in this project</p>
    ${renderSessionNavigationList(rows)}
  </section>`;
}
export function renderProjectNavigation(
  name: string,
  status: string,
  rows: readonly NavigationSession[],
  disabled: boolean,
  details: () => void,
  selection?: {active: boolean; environmentId: string; state: 'running' | 'stopped' | 'unknown'; select: () => void}
): TemplateResult {
  return html`
    <section class="soda-project-group" data-environment-id=${selection?.environmentId || ''}>
      ${
        selection
          ? html`<button
              class="ui button soda-project-select"
              aria-label=${name}
              aria-pressed=${selection.active ? 'true' : 'false'}
              aria-describedby=${'soda-project-status-' + selection.environmentId}
              ?disabled=${disabled}
              @click=${selection.select}
            >
              <span class="soda-journey-icon soda-repository-icon" aria-hidden="true"></span
              ><span class="soda-project-select-body"
                ><span>${name}</span
                ><small
                  class="soda-project-status"
                  data-state=${selection.state}
                  id=${'soda-project-status-' + selection.environmentId}
                  >${status}</small
                ></span
              >
            </button>`
          : ''
      }
      <details ?hidden=${!!selection && rows.length === 0} ?open=${rows.length > 0}>
        <summary ?hidden=${!!selection}>
          ${selection ? 'Terminals' : name} ${!selection ? html`<small>${status}</small>` : ''}
        </summary>
        ${renderSessionNavigationList(rows)}
      </details>
      ${!selection ? html`<button class="ui button soda-project-details" ?disabled=${disabled} @click=${details}>Environment / access: ${name}</button>` : ''}
    </section>
  `;
}
