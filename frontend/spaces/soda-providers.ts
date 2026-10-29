import {LitElement, html, nothing} from 'lit';
import type {TemplateResult} from 'lit';
import type {PreparedExtensionMount} from './soda-extension.js';
import {readSodaJSON} from './sodaspaces-api.js';
import {availableConnectionView, enrollmentView, items} from './soda-identity-response.js';
import type {Connection, ProviderID} from './soda-identity-response.js';
import {providersChangedEvent} from './soda-providers-events.js';
import {
  hasClock,
  initialProvidersState,
  jevTransition,
  needsPolling,
  providerIDs,
  settled,
  transition,
} from './soda-providers-machine.js';
import type {JevEvent, JevView, ProviderView, ProvidersEvent, ProvidersState} from './soda-providers-machine.js';

const providerInfo: Record<ProviderID, {name: string; plan: string}> = {
  codex: {name: 'Codex', plan: 'OpenAI Codex with your ChatGPT plan'},
  muse: {name: 'Muse', plan: 'Muse Code subscription'},
};
const pollMs = 2000;
/** Starting a sign-in launches the provider CLI, which can take tens of seconds. */
const startTimeoutMs = 60000;
const readTimeoutMs = 15000;

function seconds(since: number, now: number): string {
  const total = Math.max(0, Math.floor((now - since) / 1000));
  return total < 60 ? `${total}s` : `${Math.floor(total / 60)}m ${String(total % 60).padStart(2, '0')}s`;
}

/** The AI accounts dialog. Device sign-in runs in the identity broker; no credential enters this element. */
export class SodaProviders extends LitElement {
  static properties = {
    machine: {state: true},
    jev: {state: true},
    copied: {state: true},
  };
  declare private machine: ProvidersState;
  declare private jev: JevView;
  declare private copied: ProviderID | '';
  private current: PreparedExtensionMount | undefined;
  /** Bumped whenever the transport changes; late replies from an older one are ignored. */
  private serial = 0;
  private reading: Promise<void> | undefined;
  private poll: ReturnType<typeof setTimeout> | undefined;
  private clock: ReturnType<typeof setInterval> | undefined;
  private copyReset: ReturnType<typeof setTimeout> | undefined;
  private lastConnections = '';

  constructor() {
    super();
    this.machine = initialProvidersState();
    this.jev = {kind: 'unknown'};
    this.copied = '';
  }

  protected firstUpdated() {
    // Stop the elapsed-time clock when the dialog closes (Escape included).
    this.dialog()?.addEventListener('close', () => this.syncTimers());
  }

  protected createRenderRoot() {
    return this;
  }

  set transport(value: PreparedExtensionMount | undefined) {
    if (value === this.current) return;
    this.serial++;
    this.stopTimers();
    this.current = value;
    this.reading = undefined;
    this.dispatch({type: 'reset'});
    this.jev = {kind: 'unknown'};
    if (!value) this.dialog()?.close();
    // Read in the background so opening the dialog shows status at once.
    else void this.refresh();
  }

  disconnectedCallback() {
    this.serial++;
    this.stopTimers();
    super.disconnectedCallback();
  }

  /** Show the dialog and refresh what it shows. */
  open = async () => {
    if (!this.current) return;
    await this.updateComplete;
    const dialog = this.dialog();
    if (dialog && !dialog.open) {
      dialog.showModal();
      // Start reading at the title rather than ringing the close button.
      dialog.querySelector<HTMLElement>('h2')?.focus();
    }
    this.syncTimers();
    await this.refresh();
  };

  private dialog() {
    return this.querySelector<HTMLDialogElement>('dialog.soda-providers-dialog');
  }

  private jevDispatch(event: JevEvent) {
    this.jev = jevTransition(this.jev, event);
  }

  private dispatch(event: ProvidersEvent) {
    this.machine = transition(this.machine, event);
    this.syncTimers();
  }

  private stopTimers() {
    clearTimeout(this.poll);
    clearInterval(this.clock);
    clearTimeout(this.copyReset);
    this.poll = this.clock = undefined;
  }

  /** Poll while a sign-in awaits the person; tick the clock while the dialog shows elapsed time. */
  private syncTimers() {
    if (needsPolling(this.machine) && !this.poll && this.current)
      this.poll = setTimeout(() => {
        this.poll = undefined;
        void this.refresh();
      }, pollMs);
    const ticking = hasClock(this.machine) && this.dialog()?.open === true;
    if (ticking && !this.clock) this.clock = setInterval(() => this.requestUpdate(), 1000);
    if (!ticking && this.clock) {
      clearInterval(this.clock);
      this.clock = undefined;
    }
  }

  private async request(path: string, body?: unknown, timeout = readTimeoutMs) {
    const serial = this.serial;
    if (!this.current) throw Error('AI accounts are unavailable on this page. Reload it.');
    const response = await this.current.request(path, {
      method: body === undefined ? 'GET' : 'POST',
      ...(body === undefined ? {} : {headers: {'Content-Type': 'application/json'}, body: JSON.stringify(body)}),
      signal: AbortSignal.timeout(timeout),
    });
    const value = await readSodaJSON(response);
    if (serial !== this.serial) throw new Superseded();
    if (!response.ok) {
      const detail = value as {message?: unknown; error?: {message?: string; code?: string}};
      const error = Error(
        detail.error?.message ??
          (typeof detail?.message === 'string' ? detail.message : `AI account request failed (${response.status}).`)
      );
      (error as Error & {status?: number}).status = response.status;
      throw error;
    }
    return value;
  }

  private reason(error: unknown, fallback: string): string {
    if (error instanceof DOMException && error.name === 'TimeoutError')
      return 'The server took too long to answer. Check your connection and try again.';
    return error instanceof Error && error.message ? error.message : fallback;
  }

  /** One status read at a time; both lists are read in parallel. */
  private refresh(): Promise<void> {
    if (!this.current) return Promise.resolve();
    if (this.reading) return this.reading;
    const serial = this.serial;
    this.dispatch({type: 'load-started'});
    this.reading = (async () => {
      try {
        // JEV's status is read alongside; its own failure stays on its card.
        const jev = this.readJev(serial);
        const [enrollments, connections] = await Promise.all([
          this.request('identity/enrollments').then((value) => items(value, enrollmentView)),
          this.request('identity/connections').then((value) => items(value, availableConnectionView)),
        ]);
        this.dispatch({type: 'loaded', enrollments, connections, now: Date.now()});
        this.announce(connections);
        await jev;
      } catch (error) {
        if (!(error instanceof Superseded) && serial === this.serial)
          this.dispatch({type: 'load-failed', reason: this.reason(error, 'Could not read your AI accounts.')});
      } finally {
        if (serial === this.serial) this.reading = undefined;
      }
    })();
    return this.reading;
  }

  private async readJev(serial: number) {
    try {
      const status = (await this.request('identity/jev')) as {connected?: unknown; can_manage?: unknown};
      if (serial === this.serial)
        this.jevDispatch({type: 'loaded', connected: status.connected === true, canManage: status.can_manage === true});
    } catch (error) {
      if (!(error instanceof Superseded) && serial === this.serial)
        this.jevDispatch({type: 'load-failed', reason: this.reason(error, 'Could not read the JEV connection.')});
    }
  }

  private async connectJev(event: SubmitEvent) {
    event.preventDefault();
    const form = event.currentTarget as HTMLFormElement;
    const input = form.elements.namedItem('key') as HTMLInputElement | null;
    const key = input?.value.trim() ?? '';
    if (!key) return;
    const serial = this.serial;
    this.jevDispatch({type: 'connect-requested', now: Date.now()});
    try {
      // Checking the key makes one small JEV call.
      await this.request('identity/jev', {key}, startTimeoutMs);
      if (input) input.value = '';
      if (serial !== this.serial) return;
      this.jev = {kind: 'connected', canManage: true};
      document.dispatchEvent(new CustomEvent(providersChangedEvent));
    } catch (error) {
      if (!(error instanceof Superseded) && serial === this.serial)
        this.jevDispatch({
          type: 'connect-failed',
          reason: this.reason(error, 'JEV could not be connected. Try again.'),
        });
    }
  }

  private async disconnectJev() {
    const serial = this.serial;
    this.jevDispatch({type: 'disconnect-requested'});
    try {
      await this.request('identity/jev/disconnect', {});
      if (serial !== this.serial) return;
      this.jev = {kind: 'disconnected', canManage: true};
      document.dispatchEvent(new CustomEvent(providersChangedEvent));
    } catch (error) {
      if (!(error instanceof Superseded) && serial === this.serial)
        this.jevDispatch({type: 'disconnect-failed', reason: this.reason(error, 'JEV could not be disconnected.')});
    }
  }

  private jevStatus(): {label: string; tone: string} {
    switch (this.jev.kind) {
      case 'unknown':
        return {label: 'Checking…', tone: 'quiet'};
      case 'disconnected':
        return {label: 'Not connected', tone: 'quiet'};
      case 'checking':
        return {label: 'Checking key…', tone: 'busy'};
      case 'connected':
        return {label: 'Connected', tone: 'good'};
      case 'disconnecting':
        return {label: 'Disconnecting…', tone: 'busy'};
      case 'failed':
        return {label: 'Needs attention', tone: 'bad'};
    }
  }

  private jevForm() {
    return html`<form class="soda-provider-key" @submit=${(event: SubmitEvent) => this.connectJev(event)}>
      <label for="soda-jev-key">TypeSafe API key</label>
      <span class="soda-provider-key-row">
        <input
          id="soda-jev-key"
          name="key"
          type="password"
          autocomplete="off"
          spellcheck="false"
          required
          placeholder="Paste your key"
        />
        <button type="submit" class="ui primary button">Connect JEV</button>
      </span>
      <p class="soda-provider-hint">
        Get a key on
        <a href="https://typesafe.ai" target="_blank" rel="noopener noreferrer">typesafe.ai</a>. SodaOS checks it with
        one small JEV question, then keeps it on this server only.
      </p>
    </form>`;
  }

  private jevBody(now: number): TemplateResult | typeof nothing {
    const view = this.jev;
    switch (view.kind) {
      case 'unknown':
        return nothing;
      case 'disconnected':
        return view.canManage
          ? this.jevForm()
          : html`<p class="soda-provider-hint">
              JEV isn’t connected on this server. Ask the installation operator to connect it.
            </p>`;
      case 'checking':
        return html`<div class="soda-provider-progress" role="status">
          <span class="soda-provider-spinner" aria-hidden="true"></span>
          <div>
            <p><strong>Checking your key with JEV</strong> · ${seconds(view.since, now)}</p>
            <p class="soda-provider-hint">This asks JEV one small question and usually takes a few seconds.</p>
          </div>
        </div>`;
      case 'connected':
      case 'disconnecting': {
        const busy = view.kind === 'disconnecting';
        return html`<p class="soda-provider-account">Connected for this whole installation</p>
          ${
            view.kind === 'disconnecting' || view.canManage
              ? html`<details class="soda-provider-manage">
                  <summary>Disconnect</summary>
                  <p>Factory stops routing posts and choosing work with JEV until it is connected again.</p>
                  <button
                    type="button"
                    class="ui red basic button"
                    ?disabled=${busy}
                    @click=${() => this.disconnectJev()}
                  >
                    ${busy ? 'Disconnecting…' : 'Disconnect JEV'}
                  </button>
                </details>`
              : nothing
          }`;
      }
      case 'failed':
        return html`<p class="soda-provider-error" role="alert">${view.reason}</p>
          ${view.canManage ? this.jevForm() : nothing}`;
    }
  }

  private jevCard(now: number) {
    const status = this.jevStatus();
    return html`<li class="soda-provider-card" data-state=${this.jev.kind} data-provider="jev">
      <div class="soda-provider-head">
        <div>
          <h3>JEV</h3>
          <p class="soda-provider-hint">TypeSafe System One · routes posts and makes Factory’s choices</p>
        </div>
        <span class="soda-provider-pill" data-tone=${status.tone}>${status.label}</span>
      </div>
      ${this.jevBody(now)}
    </li>`;
  }

  private announce(connections: Connection[]) {
    const key = JSON.stringify(connections.map((c) => [c.id, c.state]));
    if (this.lastConnections && key !== this.lastConnections)
      document.dispatchEvent(new CustomEvent(providersChangedEvent));
    this.lastConnections = key;
  }

  private async start(provider: ProviderID) {
    const serial = this.serial;
    this.dispatch({type: 'start-requested', provider, now: Date.now()});
    try {
      const attempt = enrollmentView(
        await this.request(
          'identity/enrollments',
          {provider_id: provider, label: providerInfo[provider].name, confirm_credential_exposure: true},
          startTimeoutMs
        )
      );
      if (attempt.provider_id !== provider) throw Error('Sign-in returned a different provider. Try again.');
      this.dispatch({type: 'start-succeeded', provider, attempt, now: Date.now()});
    } catch (error) {
      if (error instanceof Superseded || serial !== this.serial) return;
      // Another tab may already be signing in; show that attempt instead.
      if ((error as {status?: number}).status === 409) {
        this.dispatch({type: 'start-failed', provider, reason: 'A sign-in for this account is already in progress.'});
        this.dispatch({type: 'dismissed', provider});
        await this.refresh();
        return;
      }
      this.dispatch({
        type: 'start-failed',
        provider,
        reason: this.reason(error, 'Sign-in could not start. Try again.'),
      });
    }
  }

  private async cancel(provider: ProviderID) {
    const view = this.machine.providers[provider];
    if (view.kind !== 'awaiting') return;
    this.dispatch({type: 'cancel-requested', provider});
    try {
      await this.request(`identity/enrollments/${view.attempt.id}/cancel`, {});
      this.machine = settled(this.machine, provider);
      await this.refresh();
    } catch (error) {
      if (!(error instanceof Superseded))
        this.dispatch({type: 'cancel-failed', provider, reason: this.reason(error, 'Sign-in could not be cancelled.')});
    }
  }

  private async disconnect(provider: ProviderID) {
    const view = this.machine.providers[provider];
    if (view.kind !== 'connected') return;
    this.dispatch({type: 'disconnect-requested', provider});
    try {
      for (const connection of view.connections) await this.request(`identity/connections/${connection.id}/revoke`, {});
      this.machine = settled(this.machine, provider);
      await this.refresh();
    } catch (error) {
      if (!(error instanceof Superseded))
        this.dispatch({
          type: 'disconnect-failed',
          provider,
          reason: this.reason(error, 'The account could not be disconnected.'),
        });
    }
  }

  private async copy(provider: ProviderID, code: string) {
    try {
      await navigator.clipboard.writeText(code);
      this.copied = provider;
      clearTimeout(this.copyReset);
      this.copyReset = setTimeout(() => (this.copied = ''), 2000);
    } catch {
      /* The code stays selectable when the clipboard is blocked. */
    }
  }

  private status(view: ProviderView): {label: string; tone: string} {
    switch (view.kind) {
      case 'unknown':
        return {label: 'Checking…', tone: 'quiet'};
      case 'connected':
        return {label: 'Connected', tone: 'good'};
      case 'disconnecting':
        return {label: 'Disconnecting…', tone: 'busy'};
      case 'starting':
        return {label: 'Preparing sign-in…', tone: 'busy'};
      case 'awaiting':
        return {label: 'Waiting for you', tone: 'busy'};
      case 'cancelling':
        return {label: 'Cancelling…', tone: 'busy'};
      case 'failed':
        return {label: 'Needs attention', tone: 'bad'};
      case 'disconnected':
        return {label: view.previous ? 'Signed out' : 'Not connected', tone: 'quiet'};
    }
  }

  private body(provider: ProviderID, view: ProviderView, now: number): TemplateResult | typeof nothing {
    const name = providerInfo[provider].name;
    switch (view.kind) {
      case 'unknown':
        return nothing;
      case 'disconnected':
        return html`<button type="button" class="ui primary button" @click=${() => this.start(provider)}>
          ${view.previous ? `Reconnect ${name}` : `Connect ${name}`}
        </button>`;
      case 'starting':
        return html`<div class="soda-provider-progress" role="status">
          <span class="soda-provider-spinner" aria-hidden="true"></span>
          <div>
            <p><strong>Preparing your ${name} sign-in</strong> · ${seconds(view.since, now)}</p>
            <p class="soda-provider-hint">
              SodaOS is starting ${name} and asking it for a one-time sign-in code. This usually takes 5–20 seconds.
            </p>
          </div>
        </div>`;
      case 'awaiting':
      case 'cancelling': {
        const attempt = view.attempt;
        const cancelling = view.kind === 'cancelling';
        return html`<ol class="soda-provider-steps">
            <li>
              <span>Open the ${name} sign-in page</span>
              <a
                class="ui primary button soda-provider-external"
                href=${attempt.verification_url}
                target="_blank"
                rel="noopener noreferrer"
                >Open sign-in page
                <svg width="12" height="12" viewBox="0 0 12 12" aria-hidden="true">
                  <path d="M4 2h6v6M10 2L2 10" fill="none" stroke="currentColor" stroke-width="1.6" />
                </svg>
              </a>
            </li>
            <li>
              <span>Enter this code</span>
              <span class="soda-provider-code-row">
                <strong class="soda-provider-code">${attempt.user_code}</strong>
                <button type="button" class="ui basic button" @click=${() => this.copy(provider, attempt.user_code)}>
                  ${this.copied === provider ? 'Copied ✓' : 'Copy'}
                </button>
              </span>
            </li>
            <li>
              <span>Finish signing in there</span>
              <span class="soda-provider-progress" role="status">
                <span class="soda-provider-spinner" aria-hidden="true"></span>
                ${
                  cancelling
                    ? 'Cancelling…'
                    : view.kind === 'awaiting'
                      ? `Waiting for ${name} · ${seconds(view.since, now)}`
                      : nothing
                }
              </span>
            </li>
          </ol>
          <button type="button" class="ui basic button" ?disabled=${cancelling} @click=${() => this.cancel(provider)}>
            Cancel sign-in
          </button>`;
      }
      case 'connected':
      case 'disconnecting': {
        const busy = view.kind === 'disconnecting';
        return html`${view.connections.map(
            (connection) =>
              html`<p class="soda-provider-account">
                ${connection.email || connection.label}${connection.plan ? html` · <span>${connection.plan}</span>` : nothing}
              </p>`
          )}
          <details class="soda-provider-manage">
            <summary>Disconnect</summary>
            <p>
              Disconnecting removes the stored sign-in and ends this account’s active Factory and terminal sessions.
            </p>
            <button
              type="button"
              class="ui red basic button"
              ?disabled=${busy}
              @click=${() => this.disconnect(provider)}
            >
              ${busy ? 'Disconnecting…' : `Disconnect ${name}`}
            </button>
          </details>`;
      }
      case 'failed':
        return html`<p class="soda-provider-error" role="alert">${view.reason}</p>
          <button type="button" class="ui primary button" @click=${() => this.retry(provider)}>Try again</button>`;
    }
  }

  private retry(provider: ProviderID) {
    this.dispatch({type: 'dismissed', provider});
    void this.start(provider);
  }

  private card(provider: ProviderID, now: number) {
    const view = this.machine.providers[provider];
    const status = this.status(view);
    return html`<li class="soda-provider-card" data-state=${view.kind}>
      <div class="soda-provider-head">
        <div>
          <h3>${providerInfo[provider].name}</h3>
          <p class="soda-provider-hint">${providerInfo[provider].plan}</p>
        </div>
        <span class="soda-provider-pill" data-tone=${status.tone}>${status.label}</span>
      </div>
      ${this.body(provider, view, now)}
    </li>`;
  }

  private loadNotice() {
    const {load, loadError, refreshing} = this.machine;
    if (load === 'unavailable' || (load === 'ready' && loadError))
      return html`<div class="soda-provider-error" role="alert">
        <p>${loadError || 'Could not read your AI accounts.'}</p>
        <button type="button" class="ui basic button" @click=${() => this.refresh()}>Retry</button>
      </div>`;
    if (load === 'loading' || load === 'idle')
      return html`<p class="soda-provider-hint" role="status">Checking your accounts…</p>`;
    return html`<p class="soda-provider-hint soda-provider-refreshing" role="status">
      ${refreshing ? 'Refreshing…' : ''}
    </p>`;
  }

  protected render() {
    const now = Date.now();
    // Visual rules live in assets/branding/forgejo/components.css, which every
    // Forge page loads; only the open/closed display rule is pinned here.
    return html`<style>
        .soda-providers-dialog[open] {
          display: block !important;
        }
        .soda-providers-dialog:not([open]) {
          display: none !important;
        }
      </style>
      <dialog class="soda-providers-dialog" aria-labelledby="soda-providers-title">
        <div class="soda-providers-head">
          <div>
            <h2 id="soda-providers-title" tabindex="-1">AI accounts</h2>
            <p class="soda-provider-intro">
              Connect a subscription so Factory and your terminals can use it, and JEV so Factory can make its choices.
            </p>
          </div>
          <button type="button" class="soda-providers-close" aria-label="Close" @click=${() => this.dialog()?.close()}>
            <svg width="16" height="16" viewBox="0 0 16 16" aria-hidden="true">
              <path d="M3 3l10 10M13 3L3 13" stroke="currentColor" stroke-width="1.8" stroke-linecap="square" />
            </svg>
          </button>
        </div>
        <div class="soda-providers-content">
          ${this.loadNotice()}
          <ul class="soda-provider-cards">
            ${providerIDs.map((provider) => this.card(provider, now))} ${this.jevCard(now)}
          </ul>
        </div>
      </dialog>`;
  }
}

/** A reply that arrived after the transport changed; it is dropped silently. */
class Superseded extends Error {}

customElements.define('soda-providers', SodaProviders);
