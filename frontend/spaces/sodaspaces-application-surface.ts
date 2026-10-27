import {LitElement, css, html} from 'lit';
import {keyed} from 'lit/directives/keyed.js';
import {ref} from 'lit/directives/ref.js';

export type ApplicationState = 'ready' | 'running' | 'stopped' | 'not-installed' | 'unknown';

// The app runs at its admitted origin. Forge never receives its password,
// session cookie or document contents, and browsing performs no lifecycle action.
export class SodaApplicationSurface extends LitElement {
  static properties = {
    name: {type: String},
    origin: {type: String},
    revision: {type: Number},
    state: {type: String},
    embeddable: {type: Boolean},
    canInspect: {type: Boolean},
    active: {type: Boolean},
    sessionKey: {attribute: false},
    environment: {type: String},
    reload: {state: true},
  };
  declare name: string;
  declare origin: string;
  declare revision: number;
  declare state: ApplicationState;
  declare embeddable: boolean;
  declare canInspect: boolean;
  declare active: boolean;
  declare sessionKey: string;
  declare environment: 'company' | 'preview' | 'unknown';
  declare private reload: number;
  constructor() {
    super();
    this.name = '';
    this.origin = '';
    this.revision = 0;
    this.state = 'unknown';
    this.embeddable = false;
    this.canInspect = true;
    this.active = false;
    this.sessionKey = '';
    this.environment = 'unknown';
    this.reload = 0;
  }
  static styles = css`
    :host {
      display: block;
      margin-block: 1rem;
      color: inherit;
      font: inherit;
    }
    section {
      border: 1px solid var(--color-secondary, #737680);
      border-radius: 0.5rem;
      overflow: hidden;
    }
    header {
      display: flex;
      align-items: center;
      justify-content: space-between;
      flex-wrap: wrap;
      gap: 0.75rem;
      padding: 1rem;
    }
    h3,
    p {
      margin: 0;
    }
    h3 {
      font-size: 1.1rem;
    }
    p {
      padding: 0 1rem 1rem;
      line-height: 1.5;
    }
    .controls {
      display: flex;
      gap: 0.75rem;
      align-items: center;
      flex-wrap: wrap;
    }
    button,
    a {
      font: inherit;
    }
    button {
      padding: 0.45rem 0.75rem;
      cursor: pointer;
    }
    button:focus-visible,
    a:focus-visible {
      outline: 2px solid currentColor;
      outline-offset: 2px;
    }
    header > div {
      min-width: 0;
    }
    header a {
      overflow-wrap: anywhere;
    }
    iframe {
      display: block;
      width: 100%;
      height: var(--soda-application-height, 72vh);
      min-height: 28rem;
      border: 0;
      background: white;
    }
    @media (max-width: 600px) {
      iframe {
        min-height: 24rem;
      }
      header {
        align-items: flex-start;
      }
    }
  `;
  private configureFrame = (element?: Element) => {
    if (!(element instanceof HTMLIFrameElement)) return;
    // Configure the native token list before navigating the separate origin.
    element.sandbox.add('allow-scripts', 'allow-same-origin', 'allow-forms', 'allow-downloads', 'allow-popups');
    if (element.getAttribute('src') !== this.origin) element.src = this.origin;
  };
  protected render() {
    const usable = this.active && this.state === 'ready' && this.embeddable && this.environment !== 'unknown';
    return html`<section aria-label=${this.name + ' application'}>
      <header>
        <div>
          <h3>${this.name}</h3>
          <span
            >${this.environment === 'preview' ? 'Preview · separate data' : this.environment === 'company' ? 'Company application' : 'Application environment unconfirmed'}
            · version ${this.revision}</span
          >
        </div>
        <div class="controls">
          ${usable ? html`<button type="button" @click=${() => this.reload++}>Reload app</button>` : ''}
          <a href=${this.origin} target="_blank" rel="noopener noreferrer" referrerpolicy="no-referrer"
            >Open app separately ↗</a
          >
        </div>
      </header>
      ${
        usable
          ? html`<p>Use your application account. It is separate from your Soda Forge account.</p>
              ${keyed(
                this.sessionKey + ':' + this.origin + ':' + this.revision + ':' + this.reload,
                html`<iframe
                  ${ref(this.configureFrame)}
                  title=${this.name + ' application'}
                  referrerpolicy="no-referrer"
                ></iframe>`
              )}`
          : html`<p role="status">${this.unavailable()}</p>`
      }
    </section>`;
  }
  private unavailable() {
    if (this.environment === 'unknown')
      return 'The application environment could not be confirmed. Refresh before entering company or sample data.';
    if (!this.active) return 'Reconnect to Soda Forge, then reopen this application.';
    if (this.state === 'stopped')
      return 'This application is stopped. Its administrator can start it; your saved work is retained.';
    if (this.state === 'not-installed')
      return 'Application setup is not complete. An operator must finish setup before its administrator can start it.';
    if (this.state === 'running')
      return this.canInspect
        ? 'The application is running, but readiness has not been confirmed. Refresh applications to check its availability.'
        : 'The application is running, but readiness has not been confirmed. An operator can inspect its live status; Refresh applications only rereads the last stored status.';
    if (this.state !== 'ready')
      return this.canInspect
        ? 'Availability is unconfirmed. Refresh application status before continuing.'
        : 'Availability is unconfirmed. An operator can inspect the application; Refresh applications only rereads the last stored status.';
    return 'This address cannot use its application session inside Forge. Use a separate HTTPS application hostname on the same site as Forge, or open the app separately.';
  }
}

if (!customElements.get('soda-application-surface'))
  customElements.define('soda-application-surface', SodaApplicationSurface);
