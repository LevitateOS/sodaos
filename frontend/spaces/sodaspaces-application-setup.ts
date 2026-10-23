import {LitElement, html} from 'lit';
import type {PropertyValues} from 'lit';
import {check, object} from './sodaspaces-api.js';
import type {ApplicationRequest} from './sodaspaces-applications.js';

interface Setup {
  id: string;
  application: {id: string; name: string; origin: string};
  administrator_id: number;
  expires_at: number;
  state: 'pending' | 'complete' | 'aborted';
  observation: {container: string; running: boolean; ready: boolean} | null;
}
const appID = (value: unknown): value is string => typeof value === 'string' && /^a[0-9a-f]{24}$/.test(value);
function setup(value: unknown): Setup {
  const v = object(value), a = object(v.application);
  check(Object.keys(v).length === 6 && appID(v.id) && appID(a.id) && typeof a.name === 'string' && typeof a.origin === 'string');
  const address = new URL(a.origin);
  check(address.protocol === 'https:' && address.origin === a.origin && !address.username && !address.password);
  check(typeof v.administrator_id === 'number' && Number.isSafeInteger(v.administrator_id) && v.administrator_id > 0);
  check(typeof v.expires_at === 'number' && Number.isSafeInteger(v.expires_at) && v.expires_at > 0);
  check(v.state === 'pending' || v.state === 'complete' || v.state === 'aborted');
  let observation: Setup['observation'] = null;
  if (v.observation !== null) {
    const o = object(v.observation);
    check(Object.keys(o).length === 3 && typeof o.container === 'string' && (/^[a-f0-9]{64}$/.test(o.container) || (o.container === '' && o.running === false && o.ready === false)) && typeof o.running === 'boolean' && typeof o.ready === 'boolean' && (!o.ready || o.running));
    observation = {container: o.container, running: o.running, ready: o.ready};
  }
  return {id: v.id, application: {id: a.id, name: a.name, origin: a.origin}, administrator_id: v.administrator_id, expires_at: v.expires_at, state: v.state, observation};
}
const identity = () => 'a' + crypto.randomUUID().replaceAll('-', '').slice(0, 24);

// Operator-only provisional installation. The fragment link exists only in
// this mounted component and is discarded when its session/lifetime changes.
export class SodaApplicationSetup extends LitElement {
  static properties = {request: {attribute: false}, active: {type: Boolean}, sessionKey: {attribute: false}, records: {state: true}, busy: {state: true}, notice: {state: true}, link: {state: true}};
  declare request: ApplicationRequest | undefined;
  declare active: boolean;
  declare sessionKey: string;
  declare private records: Setup[];
  declare private busy: boolean;
  declare private notice: string;
  declare private link: {id: string; url: string} | null;
  constructor() {super(); this.active = false; this.sessionKey = ''; this.records = []; this.busy = false; this.notice = ''; this.link = null;}
  private controller: AbortController | undefined;
  private epoch = 0;
  createRenderRoot() {return this;}
  protected updated(changes: PropertyValues) {
    if (changes.has('sessionKey') || (changes.has('active') && !this.active)) this.clear();
  }
  disconnectedCallback() {super.disconnectedCallback(); this.clear();}
  private clear() {this.epoch++; this.controller?.abort(); this.controller = undefined; this.link = null; this.records = []; this.busy = false;}
  private async run(work: (request: ApplicationRequest, signal: AbortSignal) => Promise<void>) {
    if (this.busy || !this.active || !this.request) return;
    const epoch = this.epoch, controller = new AbortController();
    this.controller = controller; this.busy = true; this.notice = '';
    const request = this.request;
    try {await work(async (...args) => {const result = await request(...args); if (epoch !== this.epoch || controller.signal.aborted) throw Error('inactive'); return result;}, controller.signal);}
    catch {if (epoch === this.epoch) this.notice = 'Setup could not be confirmed. Refresh and inspect the same setup before continuing.';}
    finally {if (epoch === this.epoch) {this.busy = false; this.controller = undefined;}}
  }
  private refresh() {
    this.link = null;
    return this.run(async (request, signal) => {const v = object(await request('/api/applications/setups', undefined, signal)); check(Array.isArray(v.setups)); this.records = v.setups.map(setup);});
  }
  private apply(value: unknown) {
    const v = object(value), record = setup(v.setup);
    this.records = [...this.records.filter(row => row.id !== record.id), record].filter(row => row.state === 'pending');
    this.link = null;
    if (v.result !== undefined) {
      const result = object(v.result);
      check(result.operation === record.id);
      if (result.setup_url !== undefined) {
        check(typeof result.setup_url === 'string');
        const url = new URL(result.setup_url);
        check(url.origin === record.application.origin && url.pathname === '/setup' && !url.search && !url.username && !url.password);
        const fragment = new URLSearchParams(url.hash.slice(1));
        check(fragment.get('operation') === record.id && /^[a-f0-9]{64}$/.test(fragment.get('token') ?? '') && [...fragment.keys()].length === 2);
        this.link = {id: record.id, url: result.setup_url};
      }
    }
    if (record.state === 'complete') {this.notice = 'Owner confirmed. The application is installed and stopped. Refresh applications to start it.'; this.dispatchEvent(new CustomEvent('application-setup-complete', {bubbles: true, composed: true}));}
    else if (record.state === 'aborted') this.notice = 'Setup stopped. Its data and reserved address are retained.';
  }
  private act(record: Setup, action: string) {
    this.link = null;
    return this.run(async (request, signal) => this.apply(await request(`/api/applications/setups/${record.id}/actions`, {action, expected_container: record.observation?.container ?? '', confirm: action === 'complete' || action === 'abort'}, signal)));
  }
  private create(event: SubmitEvent) {
    event.preventDefault(); if (!(event.currentTarget instanceof HTMLFormElement)) return;
    const data = new FormData(event.currentTarget), administrator = String(data.get('administrator') ?? '');
    let application: Record<string, unknown>;
    try {application = object(JSON.parse(String(data.get('application') ?? ''))); check(!('owner_team' in application));}
    catch {this.notice = 'Enter the admitted application references without an owner team.'; return;}
    application = {...application, id: identity(), data: identity()};
    this.link = null;
    return this.run(async (request, signal) => {
      const record = setup(await request('/api/applications/setups', {application, administrator_id: administrator}, signal));
      this.records = [...this.records, record];
      this.apply(await request(`/api/applications/setups/${record.id}/actions`, {action: 'begin', expected_container: '', confirm: false}, signal));
    });
  }
  protected render() {
    return html`<details @toggle=${(event: Event) => {if ((event.currentTarget as HTMLDetailsElement).open && !this.records.length && !this.busy) void this.refresh();}}>
      <summary>Set up a new application (operator)</summary>
      <p>Create the application's first owner at its own address. SodaOS stores its administrator and confirmed team; the owner's password stays with the application.</p>
      <form @submit=${(event: SubmitEvent) => void this.create(event)}>
        <label>Admitted application references<textarea name="application" required rows="5" placeholder='{"name":"MyApp","artifact_sha256":"…","bundle_sha256":"…","tools_sha256":"…","runtime_image":"sha256:…","image":"sha256:…","origin":"https://app.example.com","port":24000}'></textarea></label>
        <label>Soda administrator user ID<input name="administrator" inputmode="numeric" pattern="[1-9][0-9]*" required></label>
        <button type="submit" class="ui button" ?disabled=${this.busy || !this.active}>Begin owner setup</button>
      </form>
      <button type="button" class="ui button" ?disabled=${this.busy || !this.active} @click=${() => void this.refresh()}>Refresh owner setups</button>
      <p role="status">${this.notice}</p>
      <ul>${this.records.map(record => html`<li><strong>${record.application.name}</strong> · ${record.application.origin}
        <p>${record.observation ? record.observation.running ? 'Owner setup is running.' : 'Owner setup is stopped.' : 'Setup state is unknown.'} Setup link expires ${new Date(record.expires_at * 1000).toLocaleString()}.</p>
        ${Date.now() >= record.expires_at * 1000 ? html`<p>The link has expired. Inspect this setup, then finish if its owner was already created. If no owner was created, stop and cancel this setup; its retained data will not be overwritten.</p>` : ''}
        ${this.link?.id === record.id ? html`<p><a class="ui primary button" href=${this.link.url} target="_blank" rel="noopener noreferrer" referrerpolicy="no-referrer">Create application owner ↗</a></p>` : ''}
        <button type="button" class="ui button" ?disabled=${this.busy || !this.active || Date.now() >= record.expires_at * 1000} @click=${() => void this.act(record, 'begin')}>Open owner setup ${record.application.name}</button>
        <button type="button" class="ui button" ?disabled=${this.busy || !this.active} @click=${() => void this.act(record, 'inspect')}>Inspect owner setup ${record.application.name}</button>
        <button type="button" class="ui button" ?disabled=${this.busy || !this.active || !record.observation?.container} @click=${() => void this.act(record, 'complete')}>Confirm owner and finish ${record.application.name}</button>
        <button type="button" class="ui button" ?disabled=${this.busy || !this.active} @click=${() => void this.act(record, 'abort')}>Stop and cancel setup ${record.application.name}</button>
      </li>`)}</ul>
    </details>`;
  }
}
if (!customElements.get('soda-application-setup')) customElements.define('soda-application-setup', SodaApplicationSetup);
