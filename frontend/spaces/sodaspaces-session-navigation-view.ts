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
  readonly drop: (event: DragEvent) => void;
}
export function renderSessionTab(
  tab: SessionTab,
  navigation: boolean,
  draggable: boolean,
  dragover: (event: DragEvent) => void
): TemplateResult {
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
      >${tab.unread ? html`<span class="soda-unread" aria-label="Unread output"> •</span>` : ''}
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
        <div class="soda-session-list">
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
                  >${row.name}${row.unread ? html`<span class="soda-unread" aria-label="Unread output"> •</span>` : ''}</span
                >
                <small>${row.description}</small>
                ${row.attention ? html`<small class="soda-attention">${row.attention}</small>` : ''}
              </button>
            `
          )}
        </div>
        <p ?hidden=${rows.length !== 0}>No terminals observed.</p>
      </details>
      ${!selection ? html`<button class="ui button soda-project-details" ?disabled=${disabled} @click=${details}>Environment / access: ${name}</button>` : ''}
    </section>
  `;
}
