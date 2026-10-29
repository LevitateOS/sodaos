import {LitElement, html, nothing} from 'lit';
import type {PreparedExtensionMount} from './soda-extension.js';
import {readSodaJSON} from './sodaspaces-api.js';
import {availableConnectionView, enrollmentView, items} from './soda-identity-response.js';
import type {Connection, Enrollment, ProviderID} from './soda-identity-response.js';

const providers = [
  {id: 'codex', name: 'Codex'},
  {id: 'muse', name: 'Muse'},
] as const;

/** Device login uses the existing identity broker; no credentials enter this component. */
export class SodaProviders extends LitElement {
  static properties = {
    connections: {state: true},
    attempts: {state: true},
    busy: {state: true},
    message: {state: true},
    loaded: {state: true},
    externalTrigger: {type: Boolean, attribute: 'external-trigger'},
  };
  /** With an external trigger (the navbar robot button) the dialog is
      opened through open() and no in-page button is rendered. */
  declare externalTrigger: boolean;
  private pendingOpen = false;
  declare private connections: Connection[];
  declare private attempts: Partial<Record<ProviderID, Enrollment>>;
  declare private busy: boolean;
  declare private message: string;
  declare private loaded: boolean;
  private current: PreparedExtensionMount | undefined;
  private serial = 0;
  private poll?: ReturnType<typeof setTimeout>;

  constructor() {
    super();
    this.connections = [];
    this.attempts = {};
    this.busy = this.loaded = false;
    this.message = '';
    this.externalTrigger = false;
  }

  protected createRenderRoot() {
    return this;
  }

  set transport(value: PreparedExtensionMount | undefined) {
    if (value === this.current) return;
    this.serial++;
    clearTimeout(this.poll);
    this.current = value;
    this.connections = [];
    this.attempts = {};
    this.loaded = this.busy = false;
    this.message = '';
    this.querySelector('dialog')?.close();
    this.requestUpdate();
    void this.maybeOpen();
  }

  connectedCallback() {
    super.connectedCallback();
    // The navbar button on pages without this element navigates to the feed
    // and leaves a one-shot flag asking for the dialog on arrival.
    try {
      if (sessionStorage.getItem('soda-open-providers') === '1') {
        sessionStorage.removeItem('soda-open-providers');
        this.pendingOpen = true;
      }
    } catch {
      /* Storage can be blocked. */
    }
    void this.maybeOpen();
  }

  private async maybeOpen() {
    if (!this.pendingOpen || !this.current || !this.isConnected) return;
    this.pendingOpen = false;
    await this.updateComplete;
    await this.open();
  }

  disconnectedCallback() {
    this.serial++;
    clearTimeout(this.poll);
    super.disconnectedCallback();
  }

  open = async () => {
    if (!this.current) return;
    await this.updateComplete;
    const dialog = this.querySelector('dialog');
    if (!dialog?.open) dialog?.showModal();
    await this.run(() => this.readStatus());
  };

  private async request(path: string, body?: unknown) {
    const serial = this.serial;
    if (!this.current) throw Error('Providers are unavailable. Reopen this Project.');
    const response = await this.current.request(path, {
      method: body === undefined ? 'GET' : 'POST',
      ...(body === undefined ? {} : {headers: {'Content-Type': 'application/json'}, body: JSON.stringify(body)}),
      signal: AbortSignal.timeout(15000),
    });
    const value = await readSodaJSON(response);
    if (serial !== this.serial || !this.isConnected) throw Error('Project changed. Reopen Providers.');
    if (!response.ok) {
      const detail = value as {message?: unknown; error?: {message?: string}};
      throw Error(
        detail.error?.message ??
          (typeof detail?.message === 'string'
            ? detail.message
            : `Provider request failed (${response.status}). Try again.`)
      );
    }
    return value;
  }

  private async run(action: () => Promise<void>) {
    if (this.busy) return;
    const serial = this.serial;
    this.busy = true;
    this.message = '';
    clearTimeout(this.poll);
    try {
      await action();
    } catch (error) {
      if (serial === this.serial)
        this.message = error instanceof Error ? error.message : 'Could not read provider status.';
    } finally {
      if (serial === this.serial) {
        this.busy = false;
        this.schedulePoll();
      }
    }
  }

  private schedulePoll() {
    clearTimeout(this.poll);
    if (!this.isConnected || this.message || !Object.values(this.attempts).some((a) => a?.state === 'pending')) return;
    this.poll = setTimeout(() => void this.run(() => this.readStatus()), 1500);
  }

  private async readStatus() {
    const next = {...this.attempts};
    const recovered = items(await this.request('identity/enrollments'), enrollmentView);
    for (const attempt of recovered) next[attempt.provider_id] = attempt;
    for (const provider of providers) {
      const attempt = next[provider.id];
      if (attempt?.state === 'pending')
        next[provider.id] = enrollmentView(await this.request(`identity/enrollments/${attempt.id}`));
    }
    this.attempts = next;
    const connections = items(await this.request('identity/connections'), availableConnectionView);
    const changed = JSON.stringify(connections) !== JSON.stringify(this.connections);
    this.connections = connections;
    this.loaded = true;
    if (changed)
      this.dispatchEvent(new CustomEvent('soda-provider-connections-changed', {bubbles: true, composed: true}));
  }

  private start(provider: ProviderID) {
    return this.run(async () => {
      if (this.attempts[provider]?.state === 'pending') {
        await this.readStatus();
        return;
      }
      const attempt = enrollmentView(
        await this.request('identity/enrollments', {
          provider_id: provider,
          label: providers.find((p) => p.id === provider)!.name,
          confirm_credential_exposure: true,
        })
      );
      if (attempt.provider_id !== provider) throw Error('Sign-in returned the wrong provider. Try again.');
      this.attempts = {...this.attempts, [provider]: attempt};
    });
  }

  private cancel(provider: ProviderID) {
    return this.run(async () => {
      const attempt = this.attempts[provider];
      if (!attempt || attempt.state !== 'pending') return;
      await this.request(`identity/enrollments/${attempt.id}/cancel`, {});
      this.attempts = {...this.attempts, [provider]: {...attempt, state: 'cancelled'}};
    });
  }

  private disconnect(connection: Connection) {
    return this.run(async () => {
      await this.request(`identity/connections/${connection.id}/revoke`, {});
      await this.readStatus();
      this.message = 'Account disconnected. Its active uses have ended.';
    });
  }

  private async copy(code: string) {
    try {
      await navigator.clipboard.writeText(code);
      this.message = 'Code copied.';
    } catch {
      this.message = 'Copy the displayed code to continue.';
    }
    // Copy feedback must not stop the pending authorization check.
    clearTimeout(this.poll);
    if (Object.values(this.attempts).some((a) => a?.state === 'pending'))
      this.poll = setTimeout(() => void this.run(() => this.readStatus()), 1500);
  }

  private row(provider: (typeof providers)[number]) {
    const connected = this.connections.some((c) => c.provider_id === provider.id && c.state === 'ready');
    const attempt = this.attempts[provider.id];
    const pending = attempt?.state === 'pending';
    const previous = this.connections.some((c) => c.provider_id === provider.id);
    return html`<li class="soda-provider-row" data-connected=${connected}>
      <button
        type="button"
        class="ui basic button soda-provider-choice"
        aria-label=${`${provider.name}: ${connected ? 'Connected' : pending ? 'Awaiting sign-in' : previous ? 'Reconnect' : 'Not connected'}`}
        ?disabled=${this.busy || !this.loaded || connected || pending}
        @click=${() => this.start(provider.id)}
      >
        <strong>${provider.name}</strong>
        <span
          ><span aria-hidden="true" class="soda-provider-indicator">${connected ? '✓' : '○'}</span>
          ${connected ? 'Connected' : pending ? 'Awaiting sign-in' : previous ? 'Reconnect' : 'Connect'}</span
        >
      </button>
      ${this.connections
        .filter((c) => c.provider_id === provider.id && c.state === 'ready')
        .map(
          (connection) => html`<div class="soda-provider-detail">
            <p>${connection.email || connection.label}${connection.plan ? ` · ${connection.plan}` : ''}</p>
            <details>
              <summary>Manage account</summary>
              <p>Disconnecting ends this account’s active Factory sessions.</p>
              <button
                type="button"
                class="ui button"
                ?disabled=${this.busy}
                @click=${() => this.disconnect(connection)}
              >
                Disconnect and end active uses
              </button>
            </details>
          </div>`
        )}
      ${
        !connected && attempt
          ? html`<section aria-label=${`${provider.name} sign-in`} class="soda-provider-detail">
              <p role="status">
                ${pending ? 'Complete sign-in on the provider page. This dialog updates automatically.' : `Sign-in ${attempt.state}.`}
              </p>
              ${
                pending && attempt.verification_url
                  ? html`<a
                        class="ui primary button"
                        href=${attempt.verification_url}
                        target="_blank"
                        rel="noopener noreferrer"
                        >Open sign-in</a
                      >
                      <p>
                        Sign-in link:
                        <a href=${attempt.verification_url} target="_blank" rel="noopener noreferrer"
                          >${attempt.verification_url}</a
                        >
                      </p>`
                  : nothing
              }
              ${
                pending && attempt.user_code
                  ? html`<p>
                      Code: <strong class="soda-provider-code">${attempt.user_code}</strong>
                      <button type="button" class="ui basic button" @click=${() => this.copy(attempt.user_code)}>
                        Copy code
                      </button>
                    </p>`
                  : nothing
              }
              <button
                type="button"
                class="ui basic button"
                ?disabled=${this.busy}
                @click=${() => (pending ? this.cancel(provider.id) : this.start(provider.id))}
              >
                ${pending ? 'Cancel sign-in' : 'Try again'}
              </button>
            </section>`
          : nothing
      }
    </li>`;
  }

  protected render() {
    return html`<style>
        .soda-providers-dialog[open] {
          display: block !important;
        }
        .soda-providers-dialog:not([open]) {
          display: none !important;
        }
        .soda-providers-dialog::backdrop {
          background: rgb(0 0 0 / 35%);
        }
        .soda-providers-dialog {
          box-sizing: border-box;
          width: min(520px, calc(100vw - 32px));
          max-height: calc(100dvh - 32px);
          overflow: auto;
          padding: 24px;
          border: 1px solid var(--soda-page-text);
          border-top: 3px solid var(--soda-page-link);
          border-radius: 0;
          background: var(--soda-page-surface);
          color: var(--soda-page-text);
          font: var(--soda-font-body);
        }
        .soda-providers-dialog > header {
          display: flex;
          justify-content: space-between;
          align-items: center;
          gap: 16px;
          margin-bottom: 16px;
        }
        .soda-providers-dialog h2 {
          margin: 0;
          font: var(--soda-font-title);
        }
        .soda-providers-dialog .soda-provider-eyebrow {
          margin: 0 0 4px;
          color: var(--soda-page-muted);
          font: var(--soda-font-button);
          font-size: 11px;
          text-transform: uppercase;
          letter-spacing: 0.06em;
        }
        .soda-providers-dialog > p {
          color: var(--soda-page-muted);
          font: var(--soda-font-meta);
        }
        .soda-providers-dialog > ul {
          list-style: none;
          padding: 0;
          margin: 20px 0 0;
        }
        .soda-provider-row {
          border-top: 1px solid var(--soda-page-edge);
          padding: 12px 0;
        }
        .soda-providers-dialog .soda-provider-choice {
          display: flex;
          align-items: center;
          justify-content: space-between;
          width: 100%;
          gap: 16px;
          text-align: start;
          background: var(--soda-page-selected);
          border: 1px solid transparent;
        }
        .soda-providers-dialog .soda-provider-choice strong {
          font: var(--soda-font-label);
        }
        .soda-provider-choice > span {
          font: var(--soda-font-caption);
        }
        .soda-provider-row[data-connected='true'] .soda-provider-choice:disabled {
          opacity: 1 !important;
          color: var(--soda-page-text);
        }
        .soda-provider-row[data-connected='true'] .soda-provider-indicator {
          color: var(--soda-page-success);
        }
        .soda-provider-indicator {
          font-size: 18px;
          margin-inline-end: 4px;
        }
        .soda-provider-detail {
          padding: 12px 4px 0;
          overflow-wrap: anywhere;
          font: var(--soda-font-meta);
        }
        .soda-provider-detail summary {
          min-height: 32px;
          cursor: pointer;
          color: var(--soda-page-muted);
        }
        .soda-provider-code {
          display: inline-block;
          padding: 8px 12px;
          background: var(--soda-page-selected);
          border: 1px solid var(--soda-page-edge);
          font: var(--soda-font-mono);
          letter-spacing: 0.1em;
          user-select: all;
        }
        .soda-providers-dialog :focus-visible {
          outline: 2px solid var(--soda-page-focus);
          outline-offset: 3px;
        }
        .soda-providers-dialog [role='status']:empty {
          display: none;
        }
        @media (max-width: 480px) {
          .soda-providers-dialog {
            padding: 16px;
          }
        }</style
      >${this.externalTrigger ? nothing : html`<button type="button" class="ui basic button" @click=${this.open} ?disabled=${!this.current}>Providers</button>`}
      <dialog class="soda-providers-dialog" aria-label="Providers">
        <header>
          <div>
            <p class="soda-provider-eyebrow">Factory connections</p>
            <h2>Providers</h2>
          </div>
          <button type="button" class="ui basic button" @click=${() => this.querySelector('dialog')?.close()}>
            Close
          </button>
        </header>
        <p>
          Connect an account to use it in Factory. Connecting lets Soda securely store your account credentials and use
          them in authorized Project processes.
        </p>
        ${!this.loaded ? html`<p role="status">${this.busy ? 'Loading providers…' : 'Provider status unavailable.'}</p>` : nothing}
        <ul>
          ${providers.map((provider) => this.row(provider))}
        </ul>
        <p role="status">${this.message}</p>
        ${this.message ? html`<button type="button" class="ui basic button" ?disabled=${this.busy} @click=${() => this.run(() => this.readStatus())}>Check status</button>` : nothing}
      </dialog>`;
  }
}

customElements.define('soda-providers', SodaProviders);
