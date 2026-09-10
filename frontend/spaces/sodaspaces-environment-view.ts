import {html} from 'lit';
import type {TemplateResult} from 'lit';
import type {CreationProfile, Environment} from './sodaspaces-api.js';

export function renderProjectOS(profiles: readonly CreationProfile[], selected: string, environment: Environment | undefined, blocked: boolean, select: (id: string) => void): TemplateResult {
  if (environment) {
    const p = environment.profile;
    return html`<section aria-label="Project OS"><h3>Project OS</h3>
      ${p ? html`<p>Rocky ${p.version} · headless · ${p.architecture}</p><details><summary>Immutable creation identity</summary><dl>
        <dt>Profile</dt><dd>${p.id}</dd><dt>Image</dt><dd><code>${p.image}</code></dd><dt>Soda recipe revision</dt><dd><code>${p.revision}</code></dd>
      </dl></details>` : html`<p>Legacy / unknown creation profile. The current image default is not this persistent root’s identity.</p>`}
      <p>Creation metadata describes the original image, not later installed packages. Changing distribution/interface for an existing root is not implemented.</p></section>`;
  }
  return profiles.length ? html`<label>Project OS <select .value=${selected} ?disabled=${blocked} @change=${(event: Event) => {if (event.target instanceof HTMLSelectElement && !blocked) select(event.target.value);}}>
    ${profiles.map(p => html`<option value=${p.id} ?selected=${p.id === selected}>Rocky ${p.version} headless (${p.architecture})</option>`)}
    </select></label><p>Headless provides terminal access to the shared development foundation. KDE adds graphical access, but KDE and Fedora are not available in this build. Selection alone does not pull or start anything.</p>` : html``;
}

export interface EnvironmentPresentation {
  readonly connectURL: string;
  readonly connectVisible: boolean;
  readonly busy: boolean;
  readonly stale: boolean;
  readonly signedIn: boolean;
  readonly blocked: boolean;
  readonly canCreate: boolean;
  readonly canJoin: boolean;
  readonly sshKeys: readonly string[];
  readonly useSavedKeys: boolean;
}
export interface EnvironmentCommands {
  readonly selectSSH: (checked: boolean) => void;
  readonly connect: (event: MouseEvent) => void;
  readonly refresh: (event: MouseEvent) => void;
  readonly reload: (event: MouseEvent) => void;
  readonly logout: (event: MouseEvent) => void;
  readonly create: (event: MouseEvent) => void;
  readonly join: (event: MouseEvent) => void;
}
export function renderEnvironment(view: EnvironmentPresentation, commands: EnvironmentCommands, lifecycle: TemplateResult): TemplateResult {
  return html`
    <p>Shared resources, explicit actions. Hiding does not undo work already sent.</p>
    <a data-control="sign-in" class="ui primary button" href=${view.connectURL}
      ?hidden=${!view.connectVisible || view.stale} aria-disabled=${view.busy || view.stale ? 'true' : 'false'}
      @click=${commands.connect}>Connect to Soda</a>
    ${view.canJoin ? html`<p>Join creates your real, password-locked project account. Browser terminals do not need an SSH key. External SSH and outbound Git credentials are separate.</p>
      ${view.sshKeys.length ? html`<label><input type="checkbox" .checked=${view.useSavedKeys} ?disabled=${view.blocked}
        @change=${(event: Event) => {if (event.target instanceof HTMLInputElement) commands.selectSSH(event.target.checked);}}>Also install my saved public keys for external SSH</label>
        ${view.useSavedKeys ? html`<ul>${view.sshKeys.map(key => html`<li>${key}</li>`)}</ul>` : ''}` : html`<p>No external SSH keys will be installed. You can explicitly add and apply keys later in Access.</p>`}` : ''}
    <div class="soda-spaces-actions">
      <button data-control="refresh" type="button" class="ui basic button" ?disabled=${view.busy || view.stale}
        @click=${commands.refresh}>Refresh status</button>
      <button data-control="reload" type="button" class="ui basic button" ?hidden=${!view.stale}
        @click=${commands.reload}>Reload repository page</button>
      <button data-control="sign-out" type="button" class="ui basic button" ?hidden=${!view.signedIn}
        ?disabled=${view.busy || view.stale || !view.signedIn} @click=${commands.logout}>Sign out of Soda</button>
      <button data-control="create" type="button" class="ui primary button" ?hidden=${!view.canCreate}
        ?disabled=${view.blocked} @click=${commands.create}>Create environment</button>
      <button data-control="join" type="button" class="ui primary button" ?hidden=${!view.canJoin}
        ?disabled=${view.blocked} @click=${commands.join}>Join environment</button>
    </div>
    ${lifecycle}
  `;
}
