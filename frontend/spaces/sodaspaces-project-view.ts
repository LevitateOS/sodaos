import {html, nothing} from 'lit';
import type {TemplateResult} from 'lit';
import type {KeyPreview, SavedKey} from './sodaspaces-api.js';

export interface Connection {readonly command: string; readonly fingerprint: string}
export function renderConnection(connection: Connection | undefined, repository: string, page: boolean, blocked: boolean, copy: () => void): TemplateResult {
  return html`
    <fieldset data-control="connection" ?hidden=${!connection}>
      <legend>SSH / editor connection</legend>
      <input id=${'soda-command-' + repository} data-control="command" readonly aria-label="SSH command" .value=${connection?.command || ''}>
      <p data-control="fingerprint">${connection ? 'Ed25519 host-key fingerprint: ' + connection.fingerprint : ''}</p>
      <button data-control="copy" type="button" class="ui basic button" ?disabled=${blocked}
        data-tooltip-appendto="parent" data-clipboard-target=${connection && !page ? '#soda-command-' + repository : nothing}
        @click=${copy}>Copy SSH connection</button>
      <p>Use ordinary SSH or your editor’s Remote SSH with this account/IP. An observed IP is not proof of laptop routing.</p>
    </fieldset>
  `;
}
export interface KeyPresentation {
  readonly saved: readonly SavedKey[] | undefined;
  readonly preview: KeyPreview | undefined;
  readonly joined: boolean;
  readonly running: boolean;
  readonly blocked: boolean;
  readonly draft: string;
  readonly emptyConfirmed: boolean;
}
export interface KeyCommands {
  readonly remove: (event: MouseEvent, key: SavedKey) => void;
  readonly draft: (value: string) => void;
  readonly save: (event: MouseEvent) => void;
  readonly review: (event: MouseEvent) => void;
  readonly confirmEmpty: (checked: boolean) => void;
  readonly apply: (event: MouseEvent) => void;
}
function keyChanges(preview: KeyPreview): TemplateResult {
  return html`
    <p>The dedicated Soda-managed key file for this account will match the saved set. Review every removal;
      other accounts/files/projects and authenticated SSH sessions are unchanged.</p>
    <p>Add: ${preview.saved_fingerprints.filter(key => !preview.installed_fingerprints.includes(key)).join(', ') || 'none'}</p>
    <p>Remove: ${preview.installed_fingerprints.filter(key => !preview.saved_fingerprints.includes(key)).join(', ') || 'none'}</p>
  `;
}
export function renderKeys(view: KeyPresentation, commands: KeyCommands): TemplateResult {
  return html`
    <fieldset data-control="keys" ?hidden=${!view.saved}>
      <legend>My development SSH keys (not Forgejo Git keys)</legend>
      <ul data-control="key-list">
        ${view.saved?.map(key => html`
          <li>${key.fingerprint}
            <button type="button" class="ui basic button" ?disabled=${view.blocked}
              @click=${(event: MouseEvent) => commands.remove(event, key)}>Remove saved key</button>
          </li>
        `)}
        ${view.saved && !view.joined ? html`<li>${view.saved.length
          ? 'Start must be requested from the project administrator when stopped; then explicitly Join.'
          : 'Save your public key, then explicitly Join when the environment is running.'}</li>` : ''}
      </ul>
      <label>Public SSH key
        <textarea data-control="public-key" rows="3" maxlength="16384" spellcheck="false" autocomplete="off"
          .value=${view.draft} @input=${(event: Event) => {if (event.target instanceof HTMLTextAreaElement) commands.draft(event.target.value);}}></textarea>
      </label>
      <button data-control="save-key" type="button" class="ui primary button" ?disabled=${view.blocked} @click=${commands.save}>Save public key</button>
      <p>Removing a saved key changes future joins only. Use Review → Apply below for this project.
        Verify a replacement over SSH before revoking the old key.</p>
      <button type="button" class="ui basic button" ?hidden=${!view.joined || !view.running}
        ?disabled=${view.blocked} @click=${commands.review}>Review this project’s SSH keys</button>
      <div>${view.preview ? keyChanges(view.preview) : ''}</div>
      <label ?hidden=${!view.preview || view.preview.saved_fingerprints.length !== 0}>
        <input type="checkbox" .checked=${view.emptyConfirmed}
          @change=${(event: Event) => {if (event.target instanceof HTMLInputElement) commands.confirmEmpty(event.target.checked);}}>
        Remove all managed keys from this account: new SSH logins using them will be denied.
      </label>
      <button type="button" class="ui primary button" ?hidden=${!view.preview} ?disabled=${view.blocked}
        @click=${commands.apply}>Apply reviewed saved keys to this project</button>
    </fieldset>
  `;
}
export function renderProjectStatus(repository: string, actor: string, login: string, status: string, outcome: string): TemplateResult {
  return html`
    <div class="soda-spaces-summary">
      <p data-control="repository" class="soda-spaces-context">${repository}</p>
      <p data-control="actor">${actor}</p>
      <p data-control="login" class="soda-spaces-context">${login}</p>
      <p data-control="status" role="status">${status}</p>
      <p data-control="result" role="status">${outcome}</p>
    </div>
  `;
}
