import {LitElement, html} from 'lit';
import type {PreparedExtensionMount} from './soda-extension.js';
import {flag, list, num, object, readSodaJSON, record, text} from './sodaspaces-api.js';

const kinds = ['issue_created', 'pr_created', 'verified_merge'] as const;
type Kind = (typeof kinds)[number];
type Settings = Record<Kind, boolean>;
type ActivityItem = {id: string; kind: Kind; summary: string; url: string; created_unix: number};
type Activity = {
  repository: string;
  revision: number;
  observed_unix: number;
  truncated: boolean;
  settings: Settings;
  items: ActivityItem[];
};
type Context = {transport: PreparedExtensionMount; repository: string};
const labels: Record<Kind, string> = {
  issue_created: 'Issues created',
  pr_created: 'Pull requests created',
  verified_merge: 'Verified merges and original issue closure',
};

export function nativePath(value: unknown): value is string {
  return (
    typeof value === 'string' &&
    /^\/(?:[^/\\?#\s%]+\/)*[^/\\?#\s%]+$/u.test(value) &&
    !value.split('/').some((part) => part === '.' || part === '..')
  );
}

function activitySettings(value: unknown): Settings {
  const settings = record(value);
  return {
    issue_created: flag(settings.issue_created),
    pr_created: flag(settings.pr_created),
    verified_merge: flag(settings.verified_merge),
  };
}
export function parseFactoryActivity(value: unknown, repository: string): Activity {
  const data = object(value);
  const items = list(data.items).flatMap((item) => parseItem(item));
  return {
    repository,
    revision: num(data.revision),
    observed_unix: num(data.observed_unix),
    truncated: flag(data.truncated),
    settings: activitySettings(data.settings),
    items,
  };
}

function failureMessage(error: unknown) {
  return error instanceof Error ? error.message : 'Factory activity is unavailable.';
}
// parseItem reads one activity row. The URL becomes a link, so a row whose
// URL is not a plain native path, or whose kind this view cannot label, is
// left out instead of failing the whole list.
function parseItem(value: unknown): ActivityItem[] {
  const item = record(value);
  const kind = kinds.find((candidate) => candidate === item.kind);
  if (!kind || !nativePath(item.url)) return [];
  return [{id: text(item.id), kind, summary: text(item.summary), url: item.url, created_unix: num(item.created_unix)}];
}

// activityTimestamp renders a recorded event time honestly: a zero or
// unrepresentable stamp stays unknown instead of becoming an invented date.
export function activityTimestamp(createdUnix: number): {iso: string; text: string} | undefined {
  if (!Number.isSafeInteger(createdUnix) || createdUnix <= 0) return undefined;
  const date = new Date(createdUnix * 1000);
  if (!Number.isFinite(date.getTime())) return undefined;
  return {iso: date.toISOString(), text: date.toLocaleString()};
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
  private confirmedUnix = 0;
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
    this.confirmedUnix = 0;
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
        this.confirmedUnix = Math.floor(Date.now() / 1000);
        this.message = settings ? 'Activity display settings saved.' : '';
      }
    } catch (error) {
      // A failed refresh keeps the last confirmed feed and says so; stale
      // data is never presented as current or cleared into an empty result.
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
    const confirmed = activityTimestamp(this.confirmedUnix);
    return html`<section aria-label="Factory activity">
      <h3>Factory activity</h3>
      <p data-activity-message aria-live="polite">${this.message}</p>
      ${activity && confirmed ? html`<p>Last confirmed <time datetime=${confirmed.iso}>${confirmed.text}</time>.</p>` : ''}
      <button type="button" class="ui button" ?disabled=${this.busy} @click=${() => this.load()}>
        Refresh activity</button
      >${activity ? this.renderFeed(activity) : ''}
    </section>`;
  }
  private renderFeed(activity: Activity) {
    const visible = activity.items.filter((item) => activity.settings[item.kind]);
    const observed = activityTimestamp(activity.observed_unix);
    return html`<p>
        Local activity shows recorded native work. Display settings do not change execution or send notifications.
      </p>
      ${
        observed
          ? html`<p>Feed observed <time datetime=${observed.iso}>${observed.text}</time>.</p>`
          : html`<p>Feed observation time unknown.</p>`
      }
      ${activity.truncated ? html`<p>Older entries exist beyond this page.</p>` : ''}
      <ul>
        ${visible.map(
          (item) => html`<li><a href=${item.url}>${item.summary}</a> ${renderItemTime(item.created_unix)}</li>`
        )}
      </ul>
      ${activity.items.length === 0 ? html`<p>No recorded activity yet.</p>` : ''}
      ${activity.items.length > 0 && visible.length === 0 ? html`<p>No visible recorded activity.</p>` : ''}
      <form @submit=${this.save}>
        <fieldset>
          <legend>Activity visibility</legend>
          ${kinds.map(
            (kind) => html`<label
              ><input
                type="checkbox"
                name=${kind}
                .checked=${activity.settings[kind]}
                ?disabled=${this.busy}
              />${labels[kind]}</label
            >`
          )}
        </fieldset>
        <button class="ui button" type="submit" ?disabled=${this.busy}>Save activity display settings</button>
      </form>`;
  }
}
function renderItemTime(createdUnix: number) {
  const stamp = activityTimestamp(createdUnix);
  if (!stamp) return html`<span>time unknown</span>`;
  return html`<time datetime=${stamp.iso}>${stamp.text}</time>`;
}
customElements.define('soda-factory-activity', SodaFactoryActivity);
