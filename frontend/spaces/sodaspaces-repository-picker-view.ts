import {html} from 'lit';
import type {RepositoryChoices} from './sodaspaces-repository-response.js';

type RepositoryPickerView = {
  query: string;
  result: RepositoryChoices | undefined;
  selected: string;
  busy: boolean;
  error: string;
  blocked: boolean;
  createURL: string;
};
type RepositoryPickerActions = {
  query: (value: string) => void;
  search: (page: number) => void;
  select: (id: string) => void;
  back: () => void;
  continue: () => void;
};

function onRepositorySearch(event: SubmitEvent, search: (page: number) => void) {
  event.preventDefault();
  search(1);
}
function onRepositoryQuery(event: Event, query: (value: string) => void) {
  if (event.target instanceof HTMLInputElement) query(event.target.value);
}
function repositoryBadge(item: RepositoryChoices['items'][number]) {
  if (!item.project) return '';
  return item.project.provisioned ? 'Project exists' : 'Provisioning needs inspection';
}
function continueCaption(choice: RepositoryChoices['items'][number] | undefined) {
  if (!choice?.project) return 'Continue';
  return choice.project.provisioned ? 'Open project' : 'Inspect project';
}
function emptyRepositories(query: string) {
  return query
    ? 'No matching eligible repositories. Clear your search or create a repository.'
    : 'No eligible repositories on this page. Only a human repository owner can create its project.';
}
function repositoryChoice(item: RepositoryChoices['items'][number], selected: string, select: (id: string) => void) {
  return html`<label class="soda-repository-choice">
    <input type="radio" name="soda-repository" .checked=${selected === item.id} @change=${() => select(item.id)} />
    <span class="soda-journey-icon soda-repository-icon" aria-hidden="true"></span
    ><span>${item.owner}/${item.name}<small>${repositoryBadge(item)}</small></span>
  </label>`;
}
function repositoryResults(view: RepositoryPickerView, select: (id: string) => void) {
  const items = view.result?.items;
  return html`<fieldset class="soda-repository-results" ?disabled=${view.busy || view.blocked}>
    <legend>Available repositories</legend>
    <div class="soda-repository-rows">
      ${items?.map((item) => repositoryChoice(item, view.selected, select))}
      ${view.result && !view.result.items.length ? html`<p>${emptyRepositories(view.query)}</p>` : ''}
    </div>
  </fieldset>`;
}
function pickerLocked(view: RepositoryPickerView) {
  return view.busy || view.blocked;
}
function repositoryPages(view: RepositoryPickerView, search: (page: number) => void) {
  const result = view.result;
  if (!result || !(result.nextCursor || result.page > 1)) return '';
  return html`<nav aria-label="Repository pages">
    <button
      class="ui button"
      ?disabled=${pickerLocked(view) || result.page <= 1}
      @click=${() => search(result.page - 1)}
    >
      Previous repositories</button
    ><span>Page ${result.page}</span
    ><button
      class="ui button"
      ?disabled=${pickerLocked(view) || !result.nextCursor}
      @click=${() => search(result.page + 1)}
    >
      Next repositories
    </button>
  </nav>`;
}

export function renderRepositoryPicker(view: RepositoryPickerView, actions: RepositoryPickerActions) {
  const choice = view.result?.items.find((item) => item.id === view.selected);
  return html`<header class="soda-setup-step-heading">
      <p class="soda-setup-eyebrow">New project</p>
      <h2 tabindex="-1">Choose a repository</h2>
      <p>Choose a repository you own on this Forgejo.</p>
    </header>
    <form class="soda-repository-search" @submit=${(event: SubmitEvent) => onRepositorySearch(event, actions.search)}>
      <label
        ><span class="soda-visually-hidden">Search repositories</span
        ><input
          placeholder="Search repositories…"
          type="search"
          maxlength="200"
          .value=${view.query}
          ?disabled=${view.blocked}
          @input=${(event: Event) => onRepositoryQuery(event, actions.query)}
      /></label>
      <button class="ui button" ?disabled=${pickerLocked(view)}>Search</button>
    </form>
    <div class="soda-repository-notice soda-feedback" data-tone=${view.error ? 'warning' : 'pending'} role="status">
      ${view.busy ? 'Finding repositories…' : view.error}
    </div>
    ${repositoryResults(view, actions.select)} ${repositoryPages(view, actions.search)}
    <div class="soda-setup-actions">
      <a href=${view.createURL}>Create a new repository</a
      ><button class="ui primary button" ?disabled=${pickerLocked(view) || !choice} @click=${actions.continue}>
        ${continueCaption(choice)}
      </button>
    </div>`;
}
