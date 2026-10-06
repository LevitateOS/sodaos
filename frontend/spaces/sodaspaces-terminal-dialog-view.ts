import {html} from 'lit';
import type {TemplateResult} from 'lit';

export function renderRename(
  name: string,
  project: string,
  disabled: boolean,
  change: (value: string) => void,
  cancel: () => void,
  save: () => void
): TemplateResult {
  return html`
    <form
      class="soda-workspace-dialog"
      role="dialog"
      aria-label="Rename terminal"
      @submit=${(event: SubmitEvent) => {
        event.preventDefault();
        save();
      }}
    >
      <label
        >Session name
        <input
          maxlength="160"
          .value=${name}
          @input=${(event: Event) => {
            if (event.target instanceof HTMLInputElement) change(event.target.value);
          }}
        />
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
export function renderCreation(
  view: CreationPresentation,
  project: (id: string) => void,
  name: (value: string) => void,
  cancel: () => void,
  create: () => void
): TemplateResult {
  return html`
    <form
      class="soda-workspace-dialog"
      role="dialog"
      aria-label="New terminal"
      @submit=${(event: SubmitEvent) => {
        event.preventDefault();
        create();
      }}
      @keydown=${(event: KeyboardEvent) => {
        if (event.key === 'Escape') {
          event.stopPropagation();
          cancel();
        }
      }}
    >
      <label
        >Project
        <select
          aria-label="Project"
          .value=${view.environmentId}
          ?disabled=${view.busy}
          @change=${(event: Event) => {
            if (event.target instanceof HTMLSelectElement) project(event.target.value);
          }}
        >
          ${view.projects.map((item) => html`<option value=${item.id} ?selected=${item.id === view.environmentId}>${item.name}</option>`)}
        </select>
      </label>
      <p>${view.context}</p>
      <label
        >Terminal name
        <input
          maxlength="160"
          .value=${view.name}
          ?disabled=${view.busy}
          @input=${(event: Event) => {
            if (event.target instanceof HTMLInputElement) name(event.target.value);
          }}
        />
      </label>
      <p ?hidden=${!view.explanation}>
        ${view.explanation} Use named Environment / access; nothing is provisioned by this chooser.
      </p>
      <button type="button" class="ui button" @click=${cancel}>Cancel</button>
      <button class="ui primary button" ?disabled=${view.disabled}>Create terminal</button>
    </form>
  `;
}
