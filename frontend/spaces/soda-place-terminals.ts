// Terminals place: shells for code writers (`codex` and `muse` run in any of them) (M16, M18,
// lane D). The entry keeps this view alive while other places show (SPEC M2).
//
// Top to bottom: the environment card (status and the one next action: set
// up, start, or Join), the terminals once you have joined, and a collapsed
// "Connect from your computer (SSH)" section with the ssh command and the
// viewer's own development keys (SPEC M19, D7).
import {html, render} from 'lit';
import {object, readSodaJSON, readSodaErrorCode, SodaRequestError} from './sodaspaces-api.js';
import {detailResponse, environmentResponse} from './sodaspaces-project-response.js';
import type {Detail, Environment} from './sodaspaces-project-response.js';
import {keyPreviewResponse, profileKeysResponse, savedKeysResponse} from './sodaspaces-keys-response.js';
import type {KeyPreview, ProfileKeys, SavedKey} from './sodaspaces-keys-response.js';
import {mutate} from './sodaspaces-project-mutations.js';
import type {MutationsInput} from './sodaspaces-project-mutations.js';
import type {Connection} from './sodaspaces-project-view.js';
import type {PlaceContext, PlaceMount} from './soda-place-view.js';
import type {Place} from './soda-places.js';
import {mountSodaspaces} from './sodaspaces-workspace.js';

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
  /** Where the last result shows: beside the environment or the SSH keys. */
  private outcomeArea: 'environment' | 'keys' = 'environment';
  private sshOpen = false;
  private loadFailed = false;
  private epoch = 0;
  private disposed = false;
  private readAt = 0;
  private read: AbortController | undefined;
  /** Called after each read so the terminals below follow Join and status. */
  onChange: (() => void) | undefined;

  constructor(
    private readonly host: HTMLElement,
    /** Below the terminals: SSH is secondary once you can work in the browser. */
    private readonly sshHost: HTMLElement,
    /** The workbench header once terminals are open: status and account in one line. */
    private readonly compactHost: HTMLElement,
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

  get environmentId() {
    return this.environment?.id;
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
      this.loadFailed = false;
      this.readAt = Date.now();
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
      return {
        command: `ssh ${detail.login}@${native.ip}`,
        fingerprint: connection.fingerprint,
      };
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
    this.outcomeArea = 'environment';
    void mutate(
      this.mutations(),
      `/api/environments/${this.environment.id}/join`,
      {ssh_keys: this.useSavedKeys ? 'saved' : 'none'},
      'Joined. Open a terminal below.'
    );
  }

  private saveKey() {
    this.outcomeArea = 'keys';
    const value = this.draft.trim();
    if (!/^(ssh-|ecdsa-|sk-)/.test(value) || /PRIVATE KEY/.test(value) || /[\r\n]/.test(value)) {
      this.outcome = 'Paste one public key (it starts with ssh-, ecdsa- or sk-). Never paste a private key.';
      this.draw();
      return;
    }
    this.draft = '';
    void mutate(
      this.mutations(),
      '/api/me/development-keys',
      {public_key: value},
      'Key saved. Sync it to this environment to use it here.'
    );
  }

  private removeKey(key: SavedKey) {
    if (this.busy) return;
    if (!window.confirm('Remove this saved key? Environments keep it until you sync them.')) return;
    this.outcomeArea = 'keys';
    void mutate(
      this.mutations(),
      '/api/me/development-keys/' + key.id,
      {confirm_last: true},
      'Key removed. Sync this environment to remove it here too.',
      'DELETE'
    );
  }

  private async reviewKeys() {
    const environment = this.environment,
      login = this.detail?.login;
    if (this.busy || !environment || !login) return;
    this.outcomeArea = 'keys';
    this.outcome = '';
    this.busy = true;
    this.draw();
    try {
      this.preview = keyPreviewResponse(await this.api(`/api/environments/${environment.id}/access-keys`), login);
      this.emptyConfirmed = false;
    } catch {
      this.preview = undefined;
      this.outcome = 'Could not compare your keys with this environment. Nothing was changed.';
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
      this.outcome = 'Confirm that SSH logins with your old keys should stop working.';
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
      'This environment now uses your saved keys.'
    );
  }

  private async loadProfileKeys(page: number) {
    if (this.busy || page < 1 || page > 8) return;
    this.outcomeArea = 'keys';
    this.busy = true;
    this.draw();
    try {
      this.profileKeys = profileKeysResponse(await this.api('/api/me/forgejo-keys?page=' + page), page);
    } catch {
      this.profileKeys = undefined;
      this.outcome = 'Your Forgejo keys could not be read. Paste a public key instead.';
    } finally {
      this.busy = false;
      this.draw();
    }
  }

  private async copy() {
    if (!this.connection) return;
    this.outcomeArea = 'keys';
    try {
      await navigator.clipboard.writeText(this.connection.command);
      this.outcome = 'SSH command copied.';
    } catch {
      this.outcome = 'Copy failed. Select and copy the SSH command.';
    }
    this.draw();
  }

  private status(): {label: string; tone: string; text: string} {
    const admin = this.context.viewer.admin;
    switch (this.state) {
      case 'loading':
        return {label: 'Checking', tone: 'quiet', text: 'Checking this repository’s development environment…'};
      case 'none':
        return {
          label: 'Not set up',
          tone: 'quiet',
          text: admin
            ? 'This repository has no development environment yet. Set one up to open terminals here.'
            : 'This repository has no development environment yet. A repository administrator can set one up.',
        };
      case 'stopped':
        return {
          label: 'Stopped',
          tone: 'warn',
          text: admin
            ? 'The development environment is stopped. Start it to open terminals.'
            : 'The development environment is stopped. A repository administrator can start it.',
        };
      case 'unknown':
        return {label: 'Unavailable', tone: 'warn', text: 'The environment’s status could not be read.'};
      case 'running': {
        const detail = this.detail;
        if (detail?.login)
          return {
            label: 'Running',
            tone: 'good',
            text: `You’re joined as ${detail.login}. Your terminals run as this account.`,
          };
        return {
          label: 'Running',
          tone: 'good',
          text: 'Join to get your own account in this environment, then open terminals here.',
        };
      }
    }
  }

  private environmentAction() {
    const settings = this.context.places.settings('environment');
    const go = (event: Event) => {
      event.preventDefault();
      this.context.navigate(settings);
    };
    switch (this.state) {
      case 'none':
      case 'stopped':
        return this.context.viewer.admin
          ? html`<a class="ui primary button" href=${settings} @click=${go}
              >${this.state === 'none' ? 'Set up environment' : 'Start environment'}</a
            >`
          : '';
      case 'unknown':
        return html`<button type="button" class="ui basic button" ?disabled=${this.busy} @click=${() => this.refresh()}>
          Try again
        </button>`;
      case 'running':
        return this.renderJoin();
      default:
        return '';
    }
  }

  private renderJoin() {
    const detail = this.detail;
    if (!detail || detail.login) return '';
    if (detail.access_held)
      return html`<p class="soda-terminals-note">
        Your access is on hold. A repository administrator can restore it in Members & access.
      </p>`;
    if (!detail.execution_allowed)
      return html`<p class="soda-terminals-note">Joining needs write access to this repository.</p>`;
    return html`<div class="soda-terminals-join">
      <button
        type="button"
        class="ui primary button"
        ?disabled=${this.busy || !this.canJoin}
        @click=${() => this.join()}
      >
        Join environment
      </button>
      ${
        this.saved?.length
          ? html`<label
              ><input
                type="checkbox"
                .checked=${this.useSavedKeys}
                @change=${(event: Event) => {
                  this.useSavedKeys = (event.target as HTMLInputElement).checked;
                  this.draw();
                }}
              />
              Also install my
              ${this.saved.length === 1 ? 'saved SSH key' : `${this.saved.length} saved SSH keys`}</label
            >`
          : ''
      }
    </div>`;
  }

  private feedback(area: 'environment' | 'keys') {
    return this.outcome && this.outcomeArea === area
      ? html`<p class="soda-terminals-feedback" role="status">${this.outcome}</p>`
      : '';
  }

  private renderConnectionBlock() {
    const connection = this.connection;
    if (!connection)
      return html`<p class="soda-terminals-note">
        ${
          this.joined && this.running
            ? 'The SSH address is not available right now.'
            : 'The SSH command appears here once the environment is running and you have joined.'
        }
      </p>`;
    return html`<div class="soda-terminals-command">
        <code>${connection.command}</code>
        <button type="button" class="ui basic button" ?disabled=${this.busy} @click=${() => void this.copy()}>
          Copy
        </button>
      </div>
      <p class="soda-terminals-note">
        Works with ordinary SSH or your editor’s Remote SSH. Host key fingerprint (Ed25519):
        <code>${connection.fingerprint}</code>
      </p>`;
  }

  private renderKeyChanges(preview: KeyPreview) {
    const add = preview.saved_fingerprints.filter((key) => !preview.installed_fingerprints.includes(key));
    const remove = preview.installed_fingerprints.filter((key) => !preview.saved_fingerprints.includes(key));
    const list = (title: string, keys: string[]) =>
      keys.length
        ? html`<p>${title}</p>
            <ul class="soda-terminals-fingerprints">
              ${keys.map((key) => html`<li><code>${key}</code></li>`)}
            </ul>`
        : '';
    if (!add.length && !remove.length)
      return html`<div class="soda-terminals-sync">
        <p>This environment already uses exactly your saved keys.</p>
        <button
          type="button"
          class="ui basic button"
          @click=${() => {
            this.preview = undefined;
            this.draw();
          }}
        >
          Done
        </button>
      </div>`;
    return html`<div class="soda-terminals-sync">
      ${list('Will be added:', add)} ${list('Will be removed:', remove)}
      ${
        preview.saved_fingerprints.length
          ? ''
          : html`<label
              ><input
                type="checkbox"
                .checked=${this.emptyConfirmed}
                @change=${(event: Event) => {
                  this.emptyConfirmed = (event.target as HTMLInputElement).checked;
                  this.draw();
                }}
              />
              I understand SSH logins with these keys will stop working</label
            >`
      }
      <div class="soda-terminals-row">
        <button
          type="button"
          class="ui primary button"
          ?disabled=${this.busy || !this.detail?.execution_allowed}
          @click=${() => this.applyKeys()}
        >
          Apply changes
        </button>
        <button
          type="button"
          class="ui basic button"
          ?disabled=${this.busy}
          @click=${() => {
            this.preview = undefined;
            this.emptyConfirmed = false;
            this.draw();
          }}
        >
          Cancel
        </button>
      </div>
    </div>`;
  }

  private renderProfileKeys() {
    const keys = this.profileKeys;
    if (!keys)
      return html`<button
        type="button"
        class="soda-terminals-link"
        ?disabled=${this.busy}
        @click=${() => void this.loadProfileKeys(1)}
      >
        Or pick one of your Forgejo public keys
      </button>`;
    return html`<div class="soda-terminals-profile">
      <p class="soda-terminals-note">Pick a key you want to allow into development environments.</p>
      ${keys.items.length ? '' : html`<p class="soda-terminals-note">No public keys on this page.</p>`}
      <ul class="soda-terminals-keys">
        ${keys.items.map(
          (key) =>
            html`<li>
              <span><strong>${key.title}</strong><code>${key.fingerprint}</code></span>
              <button
                type="button"
                class="ui basic button"
                ?disabled=${this.busy}
                @click=${() => {
                  this.draft = key.public_key;
                  this.profileKeys = undefined;
                  this.outcomeArea = 'keys';
                  this.outcome = `“${key.title}” is ready to save. Check it, then Save key.`;
                  this.draw();
                }}
              >
                Use this key
              </button>
            </li>`
        )}
      </ul>
      <div class="soda-terminals-row">
        ${
          keys.page > 1
            ? html`<button
                type="button"
                class="ui basic button"
                ?disabled=${this.busy}
                @click=${() => void this.loadProfileKeys(keys.page - 1)}
              >
                Previous
              </button>`
            : ''
        }
        ${
          keys.more && keys.page < 8
            ? html`<button
                type="button"
                class="ui basic button"
                ?disabled=${this.busy}
                @click=${() => void this.loadProfileKeys(keys.page + 1)}
              >
                More keys
              </button>`
            : ''
        }
        <button
          type="button"
          class="ui basic button"
          @click=${() => {
            this.profileKeys = undefined;
            this.draw();
          }}
        >
          Close
        </button>
      </div>
    </div>`;
  }

  private renderSSH() {
    const saved = this.saved;
    const canSync = this.joined && this.running && !!this.detail?.execution_allowed;
    return html`<details
      class="soda-terminals-card soda-terminals-ssh"
      ?open=${this.sshOpen}
      @toggle=${(event: Event) => {
        this.sshOpen = (event.target as HTMLDetailsElement).open;
      }}
    >
      <summary>
        <span>
          <strong>Connect from your computer (SSH)</strong>
          <span class="soda-terminals-note">
            ${saved?.length ? `${saved.length} saved key${saved.length === 1 ? '' : 's'}` : 'Optional · no saved keys'}
          </span>
        </span>
      </summary>
      <div class="soda-terminals-ssh-body">
        <section aria-label="SSH command">
          <h3>SSH command</h3>
          ${this.renderConnectionBlock()}
        </section>
        <section aria-label="Your SSH keys">
          <h3>Your SSH keys</h3>
          <p class="soda-terminals-note">
            Keys that may log in to development environments as you. Only you see them; they are separate from your Git
            keys.
          </p>
          ${
            saved === undefined
              ? html`<p class="soda-terminals-note">Your saved keys could not be read.</p>`
              : saved.length
                ? html`<ul class="soda-terminals-keys">
                    ${saved.map(
                      (key) =>
                        html`<li>
                          <code>${key.fingerprint}</code>
                          <button
                            type="button"
                            class="ui basic button"
                            ?disabled=${this.busy}
                            @click=${() => this.removeKey(key)}
                          >
                            Remove
                          </button>
                        </li>`
                    )}
                  </ul>`
                : html`<p class="soda-terminals-note">No saved keys yet. Terminals in the browser work without one.</p>`
          }
          <label class="soda-terminals-field"
            >Add a public key
            <textarea
              rows="2"
              maxlength="16384"
              spellcheck="false"
              autocomplete="off"
              placeholder="ssh-ed25519 AAAA… you@laptop"
              .value=${this.draft}
              @input=${(event: Event) => {
                this.draft = (event.target as HTMLTextAreaElement).value;
              }}
            ></textarea>
          </label>
          <div class="soda-terminals-row">
            <button type="button" class="ui primary button" ?disabled=${this.busy} @click=${() => this.saveKey()}>
              Save key
            </button>
            ${this.renderProfileKeys()}
          </div>
        </section>
        ${
          canSync
            ? html`<section aria-label="Keys in this environment">
                <h3>Keys in this environment</h3>
                <p class="soda-terminals-note">
                  Saved keys reach this environment when you sync. Removing a key elsewhere does not log it out here
                  until you sync.
                </p>
                ${
                  this.preview
                    ? this.renderKeyChanges(this.preview)
                    : html`<button
                        type="button"
                        class="ui basic button"
                        ?disabled=${this.busy}
                        @click=${() => void this.reviewKeys()}
                      >
                        Sync my keys to this environment…
                      </button>`
                }
              </section>`
            : ''
        }
        ${this.feedback('keys')}
      </div>
    </details>`;
  }

  draw() {
    if (this.disposed) return;
    const status = this.status();
    render(
      html`<span class="soda-terminals-pill" data-tone=${status.tone} role="status">${status.label}</span>
        <strong>Development environment</strong>
        <span class="soda-terminals-note">${this.detail?.login ? `signed in as ${this.detail.login}` : ''}</span>
        ${this.feedback('environment')}`,
      this.compactHost
    );
    render(
      this.ready
        ? html``
        : html`<section class="soda-terminals-card soda-terminals-environment" aria-label="Development environment">
            <div class="soda-terminals-head">
              <h2>Development environment</h2>
              <span class="soda-terminals-pill" data-tone=${status.tone} role="status">${status.label}</span>
            </div>
            <p class="soda-terminals-lead">${status.text}</p>
            ${this.environmentAction()} ${this.feedback('environment')}
          </section>`,
      this.host
    );
    render(this.state === 'loading' ? html`` : this.renderSSH(), this.sshHost);
  }

  setActive(active: boolean) {
    if (active && Date.now() - this.readAt > 60000) void this.refresh();
  }

  dispose() {
    this.disposed = true;
    ++this.epoch;
    this.read?.abort();
    render(html``, this.host);
    render(html``, this.sshHost);
    render(html``, this.compactHost);
  }
}

const terminalsStyle = `
.soda-terminals-place { display: flex; flex-direction: column; gap: 16px; max-width: 1200px; }
.soda-terminals-card { border: 1px solid var(--soda-page-edge); background: var(--soda-page-surface); padding: 18px 20px; }
.soda-terminals-environment { display: grid; gap: 12px; justify-items: start; }
.soda-terminals-head { display: flex; align-items: center; gap: 12px; flex-wrap: wrap; }
.soda-terminals-place h2 { margin: 0; font: var(--soda-font-heading); font-size: 22px; }
.soda-terminals-place h3 { margin: 0 0 6px; font: var(--soda-font-label); }
.soda-terminals-lead { margin: 0; max-width: 70ch; }
.soda-terminals-note { margin: 0; color: var(--soda-page-muted); font: var(--soda-font-meta); max-width: 70ch; }
.soda-terminals-pill { display: inline-flex; align-items: center; gap: 6px; padding: 2px 10px; border: 1px solid var(--soda-page-edge); color: var(--soda-page-muted); font: var(--soda-font-caption); }
.soda-terminals-pill::before { content: ""; width: 7px; height: 7px; border-radius: 50%; background: currentColor; }
.soda-terminals-pill[data-tone="good"] { color: var(--soda-page-success); border-color: var(--soda-page-success); }
.soda-terminals-pill[data-tone="warn"] { color: var(--soda-page-warning); border-color: var(--soda-page-warning-edge); background: var(--soda-page-warning-bg); }
.soda-terminals-join { display: flex; align-items: center; gap: 16px; flex-wrap: wrap; }
.soda-terminals-join label, .soda-terminals-sync label { display: inline-flex; align-items: center; gap: 8px; font: var(--soda-font-meta); }
.soda-terminals-place .ui.button { margin: 0; }
.soda-terminals-feedback { margin: 0; padding: 8px 12px; border-left: 3px solid var(--soda-page-link); background: var(--soda-page-selected); font: var(--soda-font-meta); }
.soda-terminals-ssh { padding: 0; }
.soda-terminals-ssh > summary { display: flex; align-items: center; gap: 10px; padding: 14px 20px; cursor: pointer; list-style: none; }
.soda-terminals-ssh > summary::-webkit-details-marker { display: none; }
.soda-terminals-ssh > summary::before { content: "▸"; color: var(--soda-page-muted); transition: transform .15s; }
.soda-terminals-ssh[open] > summary::before { transform: rotate(90deg); }
.soda-terminals-ssh > summary > span { display: flex; align-items: baseline; gap: 12px; flex-wrap: wrap; }
.soda-terminals-ssh-body { display: grid; gap: 22px; padding: 4px 20px 20px 42px; }
.soda-terminals-ssh-body section { display: grid; gap: 10px; justify-items: start; }
.soda-terminals-command { display: flex; align-items: center; gap: 10px; flex-wrap: wrap; }
.soda-terminals-command > code { padding: 8px 12px; background: var(--soda-page-selected); border: 1px solid var(--soda-page-edge); font: var(--soda-font-mono); user-select: all; }
.soda-terminals-keys, .soda-terminals-fingerprints { list-style: none; margin: 0; padding: 0; display: grid; gap: 6px; width: 100%; max-width: 720px; }
.soda-terminals-keys > li { display: flex; align-items: center; justify-content: space-between; gap: 12px; padding: 8px 12px; border: 1px solid var(--soda-page-edge); }
.soda-terminals-keys > li > span { display: grid; gap: 2px; min-width: 0; }
.soda-terminals-keys code, .soda-terminals-fingerprints code, .soda-terminals-note code { font: var(--soda-font-mono); font-size: 13px; overflow-wrap: anywhere; }
.soda-terminals-field { display: grid; gap: 6px; width: 100%; max-width: 720px; font: var(--soda-font-label); }
.soda-terminals-field textarea { width: 100%; box-sizing: border-box; padding: 8px 10px; border: 1px solid var(--soda-page-edge); background: var(--soda-page-canvas); color: var(--soda-page-text); font: var(--soda-font-mono); font-size: 13px; resize: vertical; }
.soda-terminals-field textarea:focus { border-color: var(--soda-page-link); }
.soda-terminals-row { display: flex; align-items: center; gap: 12px; flex-wrap: wrap; }
.soda-terminals-link { padding: 0; border: 0; background: none; color: var(--soda-page-link); font: var(--soda-font-meta); text-decoration: underline; cursor: pointer; }
.soda-terminals-profile, .soda-terminals-sync { display: grid; gap: 10px; width: 100%; max-width: 720px; padding: 12px; border: 1px solid var(--soda-page-edge); background: var(--soda-page-canvas); }
.soda-terminals-sync p { margin: 0; }
.soda-terminals-bench { border: 1px solid var(--soda-page-edge); background: var(--soda-page-surface); }
.soda-terminals-bench-head { display: flex; align-items: center; justify-content: space-between; gap: 16px; flex-wrap: wrap; padding: 10px 16px; border-bottom: 1px solid var(--soda-page-edge); }
.soda-terminals-bench-status { display: flex; align-items: center; gap: 10px; flex-wrap: wrap; min-width: 0; }
.soda-terminals-bench-status .soda-terminals-feedback { flex-basis: 100%; }
.soda-terminals-workspace { height: min(75vh, 900px); min-height: 360px; display: flex; flex-direction: column; }
/* The workspace toolbar has nothing to show until a terminal is open. */
.soda-terminals-place .soda-workspace-toolbar:not(:has(> :not([hidden]))) { display: none; }
/* The bench header already names the account; the workspace's own helper line repeats it. */
.soda-terminals-place .soda-intro-helper { display: none; }
.soda-terminals-place .soda-agent-runs { display: flex; align-items: center; gap: 12px; flex-wrap: wrap; margin: 0; padding: 10px 16px; border-top: 1px solid var(--soda-page-edge); color: var(--soda-page-muted); font: var(--soda-font-meta); }
.soda-terminals-place .soda-agent-runs h2 { margin: 0; font: var(--soda-font-label); font-size: 14px; color: var(--soda-page-text); }
.soda-terminals-place .soda-agent-runs p, .soda-terminals-place .soda-agent-runs ul { margin: 0; }
.soda-terminals-workspace > .soda-spaces-page-mount { flex: 1; height: 100%; }
@media (max-width: 640px) { .soda-terminals-ssh-body { padding-left: 20px; } }
`;

export const mountTerminalsPlace: PlaceMount = (host, context, initial) => {
  host.classList.add('soda-terminals-place');
  const style = document.createElement('style');
  style.textContent = terminalsStyle;
  const environmentHost = document.createElement('div');
  // Once joined, status, session actions and terminals form one framed workbench.
  const bench = document.createElement('section');
  bench.className = 'soda-terminals-bench';
  bench.setAttribute('aria-label', 'Terminals');
  const benchHead = document.createElement('div');
  benchHead.className = 'soda-terminals-bench-head';
  const compactHost = document.createElement('div');
  compactHost.className = 'soda-terminals-bench-status';
  benchHead.append(compactHost);
  const workspaceHost = document.createElement('div');
  workspaceHost.className = 'soda-terminals-workspace';
  bench.append(benchHead, workspaceHost);
  const sshHost = document.createElement('div');
  bench.hidden = true;
  host.append(style, environmentHost, bench, sshHost);

  let run = initial.place === 'terminals' ? initial.run : undefined;
  let active = true;
  let workspace: ReturnType<typeof mountSodaspaces> | undefined;
  const environment = new EnvironmentStrip(environmentHost, sshHost, compactHost, context);

  // The terminals mount once you can use them and then stay, with their
  // connections, until this place is disposed (SPEC M2, D2).
  const ensureTerminals = () => {
    if (workspace || !environment.ready) return;
    bench.hidden = false;
    workspace = mountSodaspaces(workspaceHost, {
      kind: 'page',
      transport: context.transport,
      forgejoPrefix: context.prefix,
      repositoryId: context.repository,
      navigate: (path, options) => context.navigate(path, options),
      terminalsOnly: {runLink: () => context.places.feed()},
    });
    workspace.setVisible(active);
    void workspace.refresh();
    if (run) workspace.watchRun(run);
  };
  environment.onChange = ensureTerminals;

  return {
    update(place: Place) {
      if (place.place !== 'terminals') return;
      run = place.run;
      if (run) workspace?.watchRun(run);
    },
    setActive(value: boolean) {
      active = value;
      environment.setActive(value);
      workspace?.setVisible(value);
    },
    dispose() {
      environment.dispose();
      workspace?.dispose();
      host.replaceChildren();
      host.classList.remove('soda-terminals-place');
    },
  };
};
