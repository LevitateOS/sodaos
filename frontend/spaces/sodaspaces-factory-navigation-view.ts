import {html} from 'lit';
import type {TemplateResult} from 'lit';
import {repeat} from 'lit/directives/repeat.js';

export interface FactoryRunRow {
  readonly key: string;
  readonly text: string;
  readonly watching: boolean;
  readonly disabled: boolean;
  readonly toggle: () => void;
}
export interface FactoryWatchSlot {
  readonly run: string;
  readonly hidden: boolean;
  readonly close: () => void;
}
/** Factory runs beside human sessions. Statuses follow repository visibility;
 *  watching requires code-write authority, enforced again at attach. */
export function renderFactoryRuns(
  rows: readonly FactoryRunRow[],
  watches: readonly FactoryWatchSlot[],
  atCap: boolean
): TemplateResult {
  return html`
    <section class="soda-factory-group">
      <details ?open=${rows.length > 0}>
        <summary>Factory runs ${atCap ? html`<small>Watch up to 8 runs</small>` : ''}</summary>
        <div class="soda-session-list">
          ${repeat(
            rows,
            (row) => row.key,
            (row) => html`
              <button
                class="ui button"
                data-run-id=${row.key}
                aria-pressed=${row.watching ? 'true' : 'false'}
                ?disabled=${row.disabled}
                @click=${row.toggle}
              >
                <span>${row.watching ? 'Watching' : 'Watch'} · ${row.text}</span>
              </button>
            `
          )}
        </div>
        <p ?hidden=${rows.length !== 0}>No factory runs recorded.</p>
        ${repeat(
          watches,
          (slot) => slot.run,
          (slot) => html`
            <div class="soda-factory-box" ?hidden=${slot.hidden}>
              <div class="soda-factory-owner" data-factory-owner=${slot.run}></div>
              <button class="ui button" @click=${slot.close}>Close view</button>
            </div>
          `
        )}
      </details>
    </section>
  `;
}
