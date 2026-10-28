// Terminals place: shells and agent sessions for code writers (M16, M18,
// lane D). The entry keeps this view alive while other places show (SPEC M2).
//
// The environment strip is always here: "Development environment: Running",
// Join as the only primary action until you have joined, the `ssh` command,
// and Your development keys (the viewer's own keys only; SPEC M19, D7).
import {html, render} from 'lit';
import {object, readSodaJSON, readSodaErrorCode, SodaRequestError} from './sodaspaces-api.js';
import {detailResponse, environmentResponse} from './sodaspaces-project-response.js';
import type {Detail, Environment} from './sodaspaces-project-response.js';
import {keyPreviewResponse, profileKeysResponse, savedKeysResponse} from './sodaspaces-keys-response.js';
import type {KeyPreview, ProfileKeys, SavedKey} from './sodaspaces-keys-response.js';
import {mutate} from './sodaspaces-project-mutations.js';
import type {MutationsInput} from './sodaspaces-project-mutations.js';
import {renderConnection, renderForgejoKeys, renderKeys} from './sodaspaces-project-view.js';
import type {Connection} from './sodaspaces-project-view.js';
import type {PlaceContext, PlaceMount} from './soda-place-view.js';

type State = 'loading' | 'none' | 'running' | 'stopped' | 'unknown';

class EnvironmentStrip {
  private environment: Environment | undefined;
  private detail: Detail | undefined;
  private connection: Connection | undefined;
  private saved: SavedKey[] | undefined;
  private preview: KeyPreview | undefined;
  private profileKeys: ProfileKeys | undefined;
  private draft = '';
  private emptyConfirmed = false;
  private useSavedKeys = false;
  private busy = false;
  private outcome = '';
  private loadFailed = false;
  private epoch = 0;
  private disposed = false;
  private readAt = 0;
  private read: AbortController | undefined;
  /** Called after each read so the terminals below follow Join and status. */
  onChange: (() => void) | undefined;

  constructor(
    private readonly host: HTMLElement,
    private readonly context: PlaceContext
  ) {
    this.draw();
    void this.refresh();
  }

  private async api(path: string, method = 'GET', body?: Record<string, unknown>, signal?: AbortSignal) {
    const response = await this.context.transport.request(path.replace(/^\/api\//, ''), {
      method,
      ...(method === 'GET' ? {} : {headers: {'Content-Type': 'application/json'}}),
      ...(body === undefined ? {} : {body: JSON.stringify(body)}),
      ...(signal ? {signal} : {}),
    });
    if (response.ok) return response.status === 204 ? null : readSodaJSON(response);
    throw new SodaRequestError(response.status, await readSodaErrorCode(response));
  }

  private get state(): State {
    if (this.loadFailed) return 'unknown';
    if (this.readAt === 0) return 'loading';
    if (!this.environment) return 'none';
    const detail = this.detail;
    if (!detail || detail.authority_unavailable || detail.native_unavailable || !detail.observed) return 'unknown';
    return detail.observed.running && detail.environment.provisioned ? 'running' : 'stopped';
  }

  private get running() {
    return this.state === 'running';
  }

  private get joined() {
    return !!this.detail?.login;
  }

  /** Terminals can open: joined, allowed and running. */
  get ready() {
    const detail = this.detail;
    return this.running && !!detail?.login && detail.execution_allowed && !detail.access_held;
  }

  private get canJoin() {
    const detail = this.detail;
    return (
      this.running &&
      !!detail?.execution_allowed &&
      !detail.login &&
      !detail.access_held &&
      (!this.useSavedKeys || !!this.saved?.length)
    );
  }

  async refresh() {
    if (this.disposed) return;
    const n = ++this.epoch;
    this.read?.abort();
    const control = (this.read = new AbortController());
    const timeout = window.setTimeout(() => control.abort(), 15000);
    try {
      const collection = object(
        await this.api(
          '/api/environments?repository_id=' + encodeURIComponent(this.context.repository),
          'GET',
          undefined,
          control.signal
        )
      );
      if (n !== this.epoch) return;
      const items = Array.isArray(collection.items) ? collection.items : [];
      this.environment = items.length ? environmentResponse(items[0], this.context.repository) : undefined;
      this.detail = this.environment
        ? detailResponse(
            await this.api(`/api/environments/${this.environment.id}`, 'GET', undefined, control.signal),
            this.environment
          )
        : undefined;
      if (n !== this.epoch) return;
      this.connection = await this.readConnection(control.signal);
      if (n !== this.epoch) return;
      try {
        this.saved = savedKeysResponse(await this.api('/api/me/development-keys', 'GET', undefined, control.signal));
      } catch {
        this.saved = undefined;
      }
      if (n !== this.epoch) return;
      this.loadFailed = false;
      this.readAt = Date.now();
    } catch {
      if (n !== this.epoch) return;
      this.loadFailed = true;
      this.readAt = Date.now();
    } finally {
      window.clearTimeout(timeout);
      if (n === this.epoch) {
        this.read = undefined;
        this.draw();
        this.onChange?.();
      }
    }
  }

  private async readConnection(signal: AbortSignal): Promise<Connection | undefined> {
    const detail = this.detail;
    if (!this.environment || !detail?.login || !this.running) return undefined;
    try {
      const own = object(
        await this.api(`/api/environments/${this.environment.id}/connection`, 'GET', undefined, signal)
      );
      const connection = object(own.connection),
        native = object(connection.environment);
      if (own.login !== detail.login || typeof native.ip !== 'string' || typeof connection.fingerprint !== 'string')
        return undefined;
      if (!/^(?:[0-9]{1,3}\.){3}[0-9]{1,3}$/.test(native.ip)) return undefined;
      return {command: `ssh ${detail.login}@${native.ip}`, fingerprint: connection.fingerprint};
    } catch {
      return undefined;
    }
  }

  private mutations(): MutationsInput {
    return {
      isBlocked: () => this.busy || this.disposed,
      hasBinding: () => !this.disposed,
      readEpoch: () => this.epoch,
      isActive: () => !this.disposed,
      readBindingRepository: () => this.context.repository,
      readEnvironmentId: () => this.environment?.id,
      readDetailLogin: () => this.detail?.login,
      api: (path, method, body, signal) => this.api(path, method, body, signal),
      refreshAfter: () => this.refresh(),
      announceOperation: () => this.draw(),
      announceChanged: () => undefined,
      setBusy: (busy) => {
        this.busy = busy;
        this.draw();
      },
      setOutcomeNeedsAttention: () => undefined,
      setJoinFailed: () => undefined,
      setOutcome: (outcome) => {
        this.outcome = outcome;
        this.draw();
      },
      setMutationPending: () => undefined,
      setJoinNeedsCheck: () => undefined,
      setCanCreate: () => undefined,
    };
  }

  private join() {
    if (!this.environment || !this.canJoin || this.busy) return;
    void mutate(
      this.mutations(),
      `/api/environments/${this.environment.id}/join`,
      {ssh_keys: this.useSavedKeys ? 'saved' : 'none'},
      'Joined. Your terminals here use this account.'
    );
  }

  private saveKey() {
    const value = this.draft.trim();
    if (!/^(ssh-|ecdsa-|sk-)/.test(value) || /PRIVATE KEY/.test(value) || /[\r\n]/.test(value)) {
      this.outcome = 'Provide one public SSH key. Never upload a private key.';
      this.draw();
      return;
    }
    this.draft = '';
    void mutate(
      this.mutations(),
      '/api/me/development-keys',
      {public_key: value},
      'Public key saved. Review and apply to install it in this environment.'
    );
  }

  private removeKey(key: SavedKey) {
    if (this.busy) return;
    if (!window.confirm('Remove this saved public key? Keys already installed in environments stay until you apply.'))
      return;
    void mutate(
      this.mutations(),
      '/api/me/development-keys/' + key.id,
      {confirm_last: true},
      'Saved key removed. Installed keys are unchanged until you apply.',
      'DELETE'
    );
  }

  private async reviewKeys() {
    const environment = this.environment,
      login = this.detail?.login;
    if (this.busy || !environment || !login) return;
    this.busy = true;
    this.draw();
    try {
      this.preview = keyPreviewResponse(await this.api(`/api/environments/${environment.id}/access-keys`), login);
      this.emptyConfirmed = false;
    } catch {
      this.preview = undefined;
      this.outcome = 'Key preview unavailable. Nothing was changed.';
    } finally {
      this.busy = false;
      this.draw();
    }
  }

  private applyKeys() {
    const preview = this.preview,
      environment = this.environment;
    if (!preview || !environment || !this.detail?.execution_allowed) return;
    if (!preview.saved_fingerprints.length && !this.emptyConfirmed) {
      this.outcome = 'Confirm removal of the last managed key.';
      this.draw();
      return;
    }
    this.preview = undefined;
    this.emptyConfirmed = false;
    void mutate(
      this.mutations(),
      `/api/environments/${environment.id}/access-keys`,
      {
        revision: preview.revision,
        saved_fingerprints: preview.saved_fingerprints,
        confirm_empty: !preview.saved_fingerprints.length,
      },
      'Your SSH keys in this environment now match your saved keys.'
    );
  }

  private async loadProfileKeys(page: number) {
    if (this.busy || page < 1 || page > 8) return;
    this.busy = true;
    this.draw();
    try {
      this.profileKeys = profileKeysResponse(await this.api('/api/me/forgejo-keys?page=' + page), page);
    } catch {
      this.profileKeys = undefined;
      this.outcome = 'Your Forgejo keys are unavailable. Paste a public key instead.';
    } finally {
      this.busy = false;
      this.draw();
    }
  }

  private async copy() {
    if (!this.connection) return;
    try {
      await navigator.clipboard.writeText(this.connection.command);
      this.outcome = 'SSH command copied.';
    } catch {
      this.outcome = 'Copy failed. Select and copy the SSH command.';
    }
    this.draw();
  }

  private statusText() {
    switch (this.state) {
      case 'loading':
        return 'Development environment: checking…';
      case 'none':
        return 'Development environment: not set up. A repository administrator sets it up in SodaOS settings.';
      case 'running':
        return 'Development environment: Running';
      case 'stopped':
        return 'Development environment: Stopped. A repository administrator starts it in SodaOS settings.';
      case 'unknown':
        return 'Development environment: status unavailable';
    }
  }

  private renderJoin() {
    const detail = this.detail;
    if (!detail || detail.login) return '';
    if (detail.access_held)
      return html`<p>
        Your access to this environment is on hold. A repository administrator restores it in Members & access.
      </p>`;
    if (!detail.execution_allowed) return html`<p>Joining needs write access to this repository.</p>`;
    return html`<div class="soda-terminals-join">
      <button
        type="button"
        class="ui primary button"
        ?disabled=${this.busy || !this.canJoin}
        @click=${() => this.join()}
      >
        Join
      </button>
      ${this.saved?.length
        ? html`<label
            ><input
              type="checkbox"
              .checked=${this.useSavedKeys}
              @change=${(event: Event) => {
                this.useSavedKeys = (event.target as HTMLInputElement).checked;
                this.draw();
              }}
            />
            Also install my saved development keys for SSH</label
          >`
        : ''}
      <p>Join creates your account in this environment, so you can open terminals here or connect with SSH.</p>
    </div>`;
  }

  private renderKeyEditor() {
    if (!this.saved) return '';
    return html`<details class="soda-development-keys" ?open=${!this.joined}>
      <summary>Your development keys</summary>
      <p>SSH keys for development environments, shown only to you. They are not your Git keys.</p>
      ${renderKeys(
        {
          saved: this.saved,
          preview: this.preview,
          joined: this.joined,
          executionAllowed: !!this.detail?.execution_allowed,
          running: this.running,
          blocked: this.busy,
          draft: this.draft,
          emptyConfirmed: this.emptyConfirmed,
        },
        {
          remove: (_event, key) => this.removeKey(key),
          draft: (value) => {
            this.draft = value;
          },
          save: () => this.saveKey(),
          review: () => void this.reviewKeys(),
          confirmEmpty: (checked) => {
            this.emptyConfirmed = checked;
            this.draw();
          },
          apply: () => this.applyKeys(),
        }
      )}
      ${renderForgejoKeys(
        this.profileKeys,
        this.busy,
        (page) => void this.loadProfileKeys(page),
        (key) => {
          this.draft = key;
          this.outcome = 'Check the selected public key, then Save public key.';
          this.draw();
        }
      )}
    </details>`;
  }

  draw() {
    if (this.disposed) return;
    render(
      html`<section class="soda-terminals-environment" aria-label="Development environment">
        <p class="soda-terminals-status" role="status" data-state=${this.state}>${this.statusText()}</p>
        ${this.renderJoin()}
        ${renderConnection(this.connection, this.context.repository, true, this.busy, () => void this.copy())}
        ${this.renderKeyEditor()}
        ${this.outcome ? html`<p class="soda-feedback" role="status">${this.outcome}</p>` : ''}
      </section>`,
      this.host
    );
  }

  setActive(active: boolean) {
    if (active && Date.now() - this.readAt > 60000) void this.refresh();
  }

  dispose() {
    this.disposed = true;
    ++this.epoch;
    this.read?.abort();
    render(html``, this.host);
  }
}

export const mountTerminalsPlace: PlaceMount = (host, context) => {
  const environmentHost = document.createElement('div');
  host.append(environmentHost);
  const environment = new EnvironmentStrip(environmentHost, context);
  return {
    setActive: (active) => environment.setActive(active),
    dispose() {
      environment.dispose();
      environmentHost.remove();
    },
  };
};
