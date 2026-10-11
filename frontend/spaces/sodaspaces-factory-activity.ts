import {LitElement, html} from 'lit';
import type {PreparedExtensionMount} from './soda-extension.js';
import {object, readSodaJSON} from './sodaspaces-api.js';

const kinds = ['issue_created', 'pr_created', 'verified_merge'] as const;
type Kind = (typeof kinds)[number];
type Settings = Record<Kind, boolean>;
type Activity = {
  repository: string;
  revision: number;
  settings: Settings;
  items: {id: string; kind: Kind; summary: string; url: string}[];
};
type Context = {transport: PreparedExtensionMount; repository: string};
const labels: Record<Kind, string> = {
  issue_created: 'Issues created',
  pr_created: 'Pull requests created',
  verified_merge: 'Verified merges and original issue closure',
};

function nativePath(value: unknown): value is string {
  return (
    typeof value === 'string' &&
    /^\/(?:[^/\\?#\s%]+\/)*[^/\\?#\s%]+$/u.test(value) &&
    !value.split('/').some((part) => part === '.' || part === '..')
  );
}

export function parseFactoryActivity(value: unknown, repository: string): Activity {
  const data = object(value);
  const settings = object(data.settings);
  if (
    data.repository !== repository ||
    !nativePath(data.repository_path) ||
    typeof data.revision !== 'number' ||
    !Number.isSafeInteger(data.revision) ||
    data.revision < 0 ||
    !Array.isArray(data.items) ||
    data.items.length > 60 ||
    kinds.some((kind) => typeof settings[kind] !== 'boolean')
  )
    throw Error('Invalid Factory activity response.');
  const items = data.items.map((value) => parseItem(value, data.repository_path));
  return {
    repository,
    revision: data.revision,
    settings: {
      issue_created: settings.issue_created === true,
      pr_created: settings.pr_created === true,
      verified_merge: settings.verified_merge === true,
    },
    items,
  };
}

function boundedText(value: unknown, limit: number): value is string {
  return typeof value === 'string' && value.length > 0 && value.length <= limit;
}
function timestamp(value: unknown): value is number {
  return typeof value === 'number' && Number.isSafeInteger(value) && value >= 0;
}
function activityTarget(kind: Kind | undefined, item: Record<string, unknown>) {
  return kind === 'pr_created' ? item.pr : item.issue;
}
function failureMessage(error: unknown) {
  return error instanceof Error ? error.message : 'Factory activity is unavailable.';
}
function nativeID(value: unknown): value is string {
  return typeof value === 'string' && /^[1-9][0-9]{0,18}$/.test(value);
}
function parseItem(value: unknown, repositoryPath: unknown) {
  const item = object(value);
  const kind = kinds.find((candidate) => candidate === item.kind);
  const target = activityTarget(kind, item);
  const url = `${repositoryPath}/${kind === 'pr_created' ? 'pulls' : 'issues'}/${target}`;
  if (!kind || !nativeID(target) || item.url !== url || !nativePath(item.url))
    throw Error('Invalid Factory activity item.');
  if (!boundedText(item.id, 256) || !boundedText(item.summary, 1024) || !timestamp(item.created_unix))
    throw Error('Invalid Factory activity item.');
  return {id: item.id, kind, summary: item.summary, url};
}
function responseError(status: number) {
  if (status === 409 || status === 412) return 'Activity settings changed. Refresh before saving again.';
  if (status === 403) return 'Only the current repository owner or administrator can save activity settings.';
  return 'Factory activity is unavailable.';
}
async function requestActivity(context: Context, revision: number | undefined, settings?: Settings) {
  const init: RequestInit = {method: 'GET', signal: AbortSignal.timeout(15000)};
  if (settings) {
    init.method = 'PUT';
    init.headers = {'Content-Type': 'application/json'};
    init.body = JSON.stringify({expected_revision: revision, settings});
  }
  const response = await context.transport.request(`repositories/${context.repository}/factory/activity`, init);
  if (!response.ok) throw Error(responseError(response.status));
  return parseFactoryActivity(await readSodaJSON(response), context.repository);
}

export class SodaFactoryActivity extends LitElement {
  static properties = {activity: {state: true}, message: {state: true}, busy: {state: true}};
  declare private activity: Activity | undefined;
  declare private message: string;
  declare private busy: boolean;
  private current: Context | undefined;
  private serial = 0;
  protected createRenderRoot() {
    return this;
  }
  constructor() {
    super();
    this.message = '';
    this.busy = false;
  }
  set context(context: Context | undefined) {
    if (context?.transport === this.current?.transport && context?.repository === this.current?.repository) return;
    this.current = context;
    this.serial++;
    this.activity = undefined;
    this.message = '';
    this.busy = false;
    if (context) void this.load();
  }
  disconnectedCallback() {
    super.disconnectedCallback();
    this.serial++;
    this.current = undefined;
  }
  private async load(settings?: Settings) {
    const context = this.current;
    if (!context || this.busy) return;
    const serial = this.serial;
    this.busy = true;
    try {
      const activity = await requestActivity(context, this.activity?.revision, settings);
      if (serial === this.serial) {
        this.activity = activity;
        this.message = settings ? 'Activity display settings saved.' : '';
      }
    } catch (error) {
      if (serial === this.serial) this.message = failureMessage(error);
    } finally {
      if (serial === this.serial) this.busy = false;
    }
  }
  private save = (event: SubmitEvent) => {
    event.preventDefault();
    if (!(event.currentTarget instanceof HTMLFormElement) || !this.activity) return;
    const data = new FormData(event.currentTarget);
    void this.load({
      issue_created: data.has('issue_created'),
      pr_created: data.has('pr_created'),
      verified_merge: data.has('verified_merge'),
    });
  };
  protected render() {
    const activity = this.activity;
    return html`<section aria-label="Factory activity">
      <h3>Factory activity</h3>
      <p data-activity-message aria-live="polite">${this.message}</p>
      <button type="button" class="ui button" ?disabled=${this.busy} @click=${() => this.load()}>
        Refresh activity</button
      >${
        activity
          ? html`<p>
                Local activity shows recorded native work. Display settings do not change execution or send
                notifications.
              </p>
              <ul>
                ${activity.items.filter((item) => activity.settings[item.kind]).map((item) => html`<li><a href=${item.url}>${item.summary}</a></li>`)}
              </ul>
              ${activity.items.some((item) => activity.settings[item.kind]) ? '' : html`<p>No visible recorded activity.</p>`}
              <form @submit=${this.save}>
                ${kinds.map((kind) => html`<label><input type="checkbox" name=${kind} .checked=${activity.settings[kind]} ?disabled=${this.busy} />${labels[kind]}</label>`)}<button
                  class="ui button"
                  type="submit"
                  ?disabled=${this.busy}
                >
                  Save activity display settings
                </button>
              </form>`
          : ''
      }
    </section>`;
  }
}
customElements.define('soda-factory-activity', SodaFactoryActivity);
