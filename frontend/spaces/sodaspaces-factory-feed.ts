import {LitElement, html} from 'lit';
import {repeat} from 'lit/directives/repeat.js';
import type {PreparedExtensionMount} from './soda-extension.js';
import {allowanceHolds, allowanceHoldText, type AllowanceHold} from './sodaspaces-factory-allowance.js';
import {check, id, object, readSodaJSON} from './sodaspaces-api.js';
import {activityTimestamp, nativePath} from './sodaspaces-factory-activity.js';
import {repositoryChoices} from './sodaspaces-repository-response.js';
import {applicationsResponse} from './sodaspaces-application-response.js';
import {repositoryPlaces} from './soda-places.js';
import type {SodaPlaces} from './soda-places.js';
import type {FeedCardLinks} from './sodaspaces-factory-feed-communication.js';
import {
  commentSourceForEntry,
  feedEntryKey,
  feedEntryKinds,
  feedQueryString,
  mergeFeedEntries,
  newIdempotencyKey,
  parseCommentDetail,
  parseFeedPageView,
  parseFeedSubscriptionView,
} from './sodaspaces-factory-feed-response.js';
import type {
  CommentDetail,
  CommentPostBody,
  FeedEntry,
  FeedEntryKind,
  FeedSubscriptionView,
} from './sodaspaces-factory-feed-response.js';
import {
  describeCommentOutcome,
  describeComments,
  describeFeedPost,
  renderComposer,
  renderFeedPost,
  renderOutcome,
} from './sodaspaces-factory-feed-view.js';
import {renderFeedCommunication} from './sodaspaces-factory-feed-communication.js';
import {renderConsultation, renderContribution} from './sodaspaces-factory-consultation.js';
import {roleLabel, threadFeed} from './sodaspaces-factory-feed-thread.js';
import {feedRolePortraits} from './sodaspaces-factory-feed-portraits.js';
import {fakeFeedFilter, fakeTodoFeed} from './sodaspaces-factory-feed-fake.js';

export interface FactoryFeedContext {
  transport: PreparedExtensionMount;
  repository: string;
  repositoryName?: string;
  repositories?: ReadonlyMap<string, string>;
  account?: string;
  global?: boolean;
  /** Installation prefix (appSubUrl) for SodaOS place links. */
  prefix?: string;
}

interface RepositoryLabels {
  refs: Map<number, {path: string | undefined; label: string}>;
  listedAll: boolean;
}

interface Thread {
  items: CommentDetail[];
  next: number;
  busy: boolean;
  message: string;
  // Older pages were appended: a quiet refresh leaves the thread alone so
  // the reader keeps their place; Refresh replies starts over.
  pagedOlder: boolean;
}

interface Revising {
  id: string;
  revision: number;
  repository: string;
  body: string;
  original?: string;
  question?: string;
  // The request text the server holds after a refused stale submit.
  current?: string;
}

const feedPageSize = 20;
const commentID = /^[a-f0-9]{32}$/;

function failureMessage(status: number | undefined, fallback: string): string {
  if (status === 403)
    return 'One or more subscribed projects are not visible to you. The feed shows only permitted activity.';
  return fallback;
}

// Relative reply age for comment rows ("now", "12m", "2h", "3d"); older
// replies fall back to a short date.
export function relativeReplyTime(at: number, now = Date.now()): string {
  const seconds = Math.floor((now - at * 1000) / 1000);
  if (seconds < 60) return 'now';
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes}m`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h`;
  const days = Math.floor(hours / 24);
  if (days < 7) return `${days}d`;
  return new Date(at * 1000).toLocaleDateString(undefined, {month: 'short', day: 'numeric'});
}

// Unsent text survives reloads as plain per-key values in browser storage;
// when storage is unavailable it stays in memory only.
function readStored(key: string): string {
  try {
    return window.localStorage.getItem(key) ?? '';
  } catch {
    return '';
  }
}

function writeStored(key: string, value: string): void {
  try {
    if (value === '') window.localStorage.removeItem(key);
    else window.localStorage.setItem(key, value);
  } catch {
    /* Memory keeps the value. */
  }
}

async function readJSON(transport: PreparedExtensionMount, path: string, init?: RequestInit): Promise<unknown> {
  const response = await transport.request(path, {method: 'GET', signal: AbortSignal.timeout(15000), ...init});
  if (!response.ok) {
    const error = Error(failureMessage(response.status, 'The combined feed is unavailable.'));
    (error as {status?: number}).status = response.status;
    throw error;
  }
  // Feed pages, conversation pages and comment reads carry stored text.
  const large = path.startsWith('factory/feed?') || path.startsWith('factory/feed/');
  return readSodaJSON(response, large ? 4 * 1024 * 1024 : 65536);
}

/** Today's clock time, otherwise a short date and time. */
function shortTime(unix: number): string {
  const date = new Date(unix * 1000);
  const today = new Date().toDateString() === date.toDateString();
  return date.toLocaleString(
    undefined,
    today ? {hour: 'numeric', minute: '2-digit'} : {month: 'short', day: 'numeric', hour: 'numeric', minute: '2-digit'}
  );
}

export class SodaFactoryFeed extends LitElement {
  static properties = {view: {state: true}};
  declare private view: {
    message: string;
    busy: boolean;
    readFailed: boolean;
    subscription: FeedSubscriptionView | undefined;
    labels: RepositoryLabels;
    entries: FeedEntry[];
    next: string;
    truncated: boolean;
    pagedBack: boolean;
    confirmedUnix: number;
    commentBusy: string[];
    allowance: AllowanceHold[];
  };
  private current: FactoryFeedContext | undefined;
  // One serial: a context switch, filter change or disconnect bumps it, and
  // every response that lands after a bump is dropped.
  private serial = 0;
  // The serial whose latest-page read is in flight, if any.
  private reading: number | undefined;
  private refreshTimer?: ReturnType<typeof setInterval>;
  private drafts = new Map<string, string>();
  private revising: Revising | undefined;
  private threads = new Map<string, Thread>();
  private threadReads = new Map<string, Promise<void>>();
  private openDrawers = new Set<string>();
  // The viewer's native user id (the feed subscription's subscriber): Open
  // request shows only on the viewer's own requests.
  private viewerID: string | undefined;
  // Verified delivery id → the origin of a Ready app the viewer may open.
  private appOrigins = new Map<string, string>();
  // The repository comment box stays one line until focused.
  private commentOpen = false;
  private composeRequested = false;
  // A post to show alone (the Feed place's ?post=, e.g. a Knowledge item's source).
  private focusPost: string | undefined;
  private librarySourcePending: string | undefined;
  // A newer source open or leaving the post supersedes a read in flight.
  private librarySerial = 0;
  private focusedEntry: FeedEntry | undefined;
  private filters = {view: 'all', search: '', role: '', current: false};

  // The Fake feed tab shows a made-up todo-app feed instead of the served
  // one; nothing is read or written while it is selected.
  private get fake() {
    return this.filters.view === 'fake';
  }

  protected createRenderRoot() {
    return this;
  }

  constructor() {
    super();
    // The repository comment box opens from one line when focused.
    this.addEventListener('focusin', (event) => {
      if (this.commentOpen || !(event.target instanceof Element) || !event.target.closest('.soda-feed-comment-box'))
        return;
      this.commentOpen = true;
      this.requestUpdate();
    });
    this.view = this.emptyView();
  }

  private emptyView(): typeof this.view {
    return {
      message: '',
      busy: false,
      readFailed: false,
      subscription: undefined,
      labels: {refs: new Map(), listedAll: false},
      entries: [],
      next: '',
      truncated: false,
      pagedBack: false,
      confirmedUnix: 0,
      allowance: [],
      commentBusy: [],
    };
  }

  set context(context: FactoryFeedContext | undefined) {
    if (
      context?.transport === this.current?.transport &&
      context?.repository === this.current?.repository &&
      context?.account === this.current?.account &&
      context?.global === this.current?.global
    )
      return;
    clearInterval(this.refreshTimer);
    this.serial++;
    this.reading = undefined;
    this.current = context;
    this.drafts.clear();
    this.threads.clear();
    this.threadReads.clear();
    this.openDrawers.clear();
    this.revising = undefined;
    this.librarySerial++;
    this.librarySourcePending = undefined;
    this.viewerID = undefined;
    this.appOrigins = new Map();
    this.commentOpen = false;
    this.focusedEntry = undefined;
    this.filters = {view: 'all', search: '', role: '', current: false};
    this.view = this.emptyView();
    if (context) {
      void this.reload();
      if (this.focusPost) void this.openSourcePost(this.focusPost);
      this.refreshTimer = setInterval(() => {
        if (this.isConnected && document.visibilityState === 'visible' && !this.view.busy) void this.readLatest(false);
      }, 10000);
    }
  }

  disconnectedCallback() {
    clearInterval(this.refreshTimer);
    super.disconnectedCallback();
    this.serial++;
    this.current = undefined;
  }

  private account(): string {
    return this.current?.account ?? this.current?.transport.generation ?? 'session';
  }

  private storageKey(key: string) {
    return `${this.account()}:${key}`;
  }

  private scopeRepositories(): number[] {
    const subscription = this.view.subscription;
    // Until a subscription is saved (revision 0), /feed shows every candidate
    // repository it already loads; nothing is written.
    if (this.current?.global)
      return subscription?.revision === 0
        ? [...this.view.labels.refs.keys()].sort((a, b) => a - b).slice(0, 1024)
        : (subscription?.repositories ?? []);
    const current = Number(this.current?.repository);
    return Number.isSafeInteger(current) && current > 0 ? [current] : [];
  }

  private async reload() {
    const context = this.current;
    if (!context || this.fake || this.view.busy) return;
    if (!context.global && !id(context.repository)) {
      this.view = {...this.view, message: 'The combined feed needs its current project binding.'};
      return;
    }
    const serial = this.serial;
    this.view = {...this.view, busy: true, readFailed: false, message: this.view.readFailed ? '' : this.view.message};
    try {
      const subscription = context.global
        ? parseFeedSubscriptionView(await readJSON(context.transport, 'factory/feed/subscription'))
        : {
            subscriber: context.account ?? context.transport.generation,
            revision: 0,
            repositories: [Number(context.repository)],
            kinds: [],
          };
      if (serial !== this.serial) return;
      if (context.global) this.viewerID = subscription.subscriber;
      else void this.readViewerID(context);
      this.view = {...this.view, subscription};
      const labels = await this.resolveLabels(context, subscription);
      if (serial !== this.serial) return;
      this.view = {...this.view, labels};
      void this.readAppOrigins(context);
      await this.readLatest(true);
    } catch (error) {
      if (serial === this.serial)
        this.view = {
          ...this.view,
          busy: false,
          readFailed: true,
          message: error instanceof Error ? error.message : 'The combined feed is unavailable.',
        };
    }
  }

  private async resolveLabels(
    context: FactoryFeedContext,
    subscription: FeedSubscriptionView
  ): Promise<RepositoryLabels> {
    const refs = new Map<number, {path: string | undefined; label: string}>();
    const wanted = new Set<number>(subscription.repositories);
    const current = Number(context.repository);
    const addKnown = (numeric: number, name: string) => {
      const path = '/' + name;
      if (Number.isSafeInteger(numeric) && numeric > 0 && nativePath(path)) {
        refs.set(numeric, {path, label: name});
        wanted.delete(numeric);
      }
    };
    for (const [key, name] of context.repositories ?? []) addKnown(Number(key), name);
    if (context.repositoryName) addKnown(current, context.repositoryName);
    if (Number.isSafeInteger(current) && current > 0) wanted.add(current);
    if (context.global) {
      try {
        let after = '';
        const seen = new Set<string>();
        do {
          const page = object(
            await readJSON(context.transport, 'spaces' + (after ? '?after=' + encodeURIComponent(after) : ''))
          );
          check(Array.isArray(page.items));
          for (const item of page.items) {
            const environment = object(object(item).environment);
            if (typeof environment.repository === 'string')
              addKnown(Number(environment.repository_id), environment.repository);
          }
          const next = typeof page.next_after === 'string' ? page.next_after : '';
          if (!next || seen.has(next)) return {refs, listedAll: page.complete === true};
          seen.add(next);
          after = next;
        } while (this.current === context);
      } catch {
        return {refs, listedAll: false};
      }
    }
    try {
      const choices = repositoryChoices(await readJSON(context.transport, 'repositories?q='), 1);
      for (const item of choices.items) {
        const numeric = Number(item.id);
        if (!Number.isSafeInteger(numeric) || numeric <= 0) continue;
        const path = `/${item.owner}/${item.name}`;
        refs.set(numeric, {path: nativePath(path) ? path : undefined, label: `${item.owner}/${item.name}`});
        wanted.delete(numeric);
      }
      for (const missing of wanted) {
        if (!refs.has(missing)) refs.set(missing, {path: undefined, label: `repository ${missing}`});
      }
      return {refs, listedAll: choices.nextCursor === ''};
    } catch {
      for (const missing of wanted) refs.set(missing, {path: undefined, label: `repository ${missing}`});
      return {refs, listedAll: false};
    }
  }

  private feedPath(cursor: string): string {
    return (
      feedQueryString({
        repositories: this.scopeRepositories(),
        kinds: [],
        limit: feedPageSize,
        cursor,
        ...this.filters,
      }) + (this.current?.global ? '' : '&scope=project')
    );
  }

  // readLatest re-reads the newest page. Posts arrive with their stored text
  // and comment outcome, so this one read refreshes every card; open threads
  // re-read their newest replies alongside.
  private async readLatest(announce: boolean) {
    const context = this.current;
    if (!context || this.fake || this.reading === this.serial) return;
    const serial = this.serial;
    if (this.scopeRepositories().length === 0) {
      this.view = {
        ...this.view,
        busy: false,
        message: 'Choose at least one repository to follow before reading the Feed.',
      };
      return;
    }
    this.reading = serial;
    this.view = {...this.view, busy: true, readFailed: false, message: this.view.readFailed ? '' : this.view.message};
    const anchor = Array.from(this.querySelectorAll<HTMLElement>('.soda-feed-post')).find(
      (el) => el.getBoundingClientRect().bottom > 0
    );
    const anchorTop = anchor?.getBoundingClientRect().top;
    try {
      const page = parseFeedPageView(await readJSON(context.transport, this.feedPath('')));
      if (serial !== this.serial) return;
      const refs = new Map(this.view.labels.refs);
      for (const entry of page.entries) {
        if (entry.project && nativePath('/' + entry.project))
          refs.set(entry.repository, {path: '/' + entry.project, label: entry.project});
      }
      const known = new Set(this.view.entries.map(feedEntryKey));
      const fresh = page.entries.filter((entry) => !known.has(feedEntryKey(entry))).length;
      const older = this.view.pagedBack
        ? this.view.entries.filter((entry) => entry.seq < (page.entries.at(-1)?.seq ?? 0))
        : [];
      let message = this.view.message;
      if (announce && known.size > 0)
        message = fresh ? `${fresh} new post${fresh === 1 ? '' : 's'}.` : 'Feed refreshed; no new posts.';
      this.view = {
        ...this.view,
        labels: {...this.view.labels, refs},
        busy: false,
        entries: mergeFeedEntries(older, page.entries),
        next: this.view.pagedBack ? this.view.next : page.next,
        truncated: this.view.pagedBack ? this.view.truncated : page.truncated,
        confirmedUnix: Math.floor(Date.now() / 1000),
        message,
      };
      if (anchor && anchorTop !== undefined) {
        await this.updateComplete;
        if (serial === this.serial && anchor.isConnected) {
          const delta = anchor.getBoundingClientRect().top - anchorTop;
          const scroller = this.closest<HTMLElement>('.soda-project-content');
          if (scroller && scroller.scrollHeight > scroller.clientHeight) scroller.scrollBy({top: delta});
          else window.scrollBy({top: delta});
        }
      }
      void this.readAllowance(serial);
      for (const key of this.openDrawers) {
        const entry = this.entryForKey(key);
        if (entry && !this.threads.get(key)?.pagedOlder) void this.loadReplies(entry);
      }
    } catch (error) {
      if (serial === this.serial)
        // A failed refresh keeps the last loaded feed and says so.
        this.view = {
          ...this.view,
          busy: false,
          readFailed: true,
          message: error instanceof Error ? error.message : 'The combined feed is unavailable.',
        };
    } finally {
      if (this.reading === serial) this.reading = undefined;
    }
  }

  private async readOlder() {
    const context = this.current;
    if (!context || this.fake || this.view.busy || this.view.next === '') return;
    const serial = this.serial;
    this.view = {...this.view, busy: true};
    try {
      const page = parseFeedPageView(await readJSON(context.transport, this.feedPath(this.view.next)));
      if (serial !== this.serial) return;
      this.view = {
        ...this.view,
        busy: false,
        entries: mergeFeedEntries(this.view.entries, page.entries),
        next: page.next,
        truncated: page.truncated,
        pagedBack: true,
        confirmedUnix: Math.floor(Date.now() / 1000),
      };
    } catch (error) {
      if (serial === this.serial)
        this.view = {
          ...this.view,
          busy: false,
          message: error instanceof Error ? error.message : 'Older feed entries are unavailable.',
        };
    }
  }

  // readAllowance names a coder-time allowance hold on the followed
  // repositories above the posts. The status read is owner/admin only; any
  // failure leaves no notice.
  private async readAllowance(serial: number) {
    const context = this.current;
    if (!context) return;
    const settled = await Promise.allSettled(
      this.scopeRepositories()
        .slice(0, 8)
        .map(async (repository) =>
          allowanceHolds(await readJSON(context.transport, `repositories/${repository}/factory`), String(repository))
        )
    );
    if (serial !== this.serial) return;
    this.view = {
      ...this.view,
      allowance: settled.flatMap((result) => (result.status === 'fulfilled' ? result.value : [])),
    };
  }

  private changeFilters(next: Partial<typeof this.filters>) {
    this.focusedEntry = undefined;
    if (next.view !== undefined && (next.view === 'fake') !== this.fake) {
      // Switching between the fake and served feed drops open threads.
      this.threads.clear();
      this.openDrawers.clear();
      this.revising = undefined;
    }
    this.filters = {...this.filters, ...next};
    this.serial++;
    this.reading = undefined;
    this.view = {
      ...this.view,
      busy: false,
      readFailed: false,
      message: '',
      entries: [],
      next: '',
      truncated: false,
      pagedBack: false,
    };
    if (this.fake)
      this.view = {
        ...this.view,
        entries: fakeFeedFilter(fakeTodoFeed(Number(this.current?.repository) || 0), this.filters),
        confirmedUnix: Math.floor(Date.now() / 1000),
        message: 'Fake feed: a made-up todo app. Pick another tab for the real feed.',
      };
    else void this.readLatest(false);
  }

  private backToLatest = () => {
    if (!this.current || this.view.busy) return;
    this.view = {...this.view, next: '', truncated: false, pagedBack: false};
    void this.readLatest(false).then(() => this.querySelector('.soda-feed-list')?.scrollIntoView());
  };

  private projectCommentKey(): string {
    return 'project:' + (this.current?.repository ?? '');
  }

  private entryForKey(key: string): FeedEntry | undefined {
    const entry = [...this.view.entries, ...(this.focusedEntry ? [this.focusedEntry] : [])].find(
      (candidate) => feedEntryKey(candidate) === key
    );
    if (entry) return entry;
    // A reply row answered from inside a thread.
    for (const [root, thread] of this.threads) {
      const reply = thread.items.find((item) => 'post:' + item.commentId === key);
      const parent = reply ? this.entryForKey(root) : undefined;
      if (reply && parent)
        return {
          kind: 'post',
          id: reply.commentId,
          repository: parent.repository,
          seq: 0,
          ...(reply.communication ? {payload: {communication: reply.communication}} : {}),
        };
    }
    return undefined;
  }

  private draftFor(key: string): string {
    let draft = this.drafts.get(key);
    if (draft === undefined) {
      draft = readStored(`soda-feed-draft:v1:${this.storageKey(key)}`).slice(0, 4000);
      this.drafts.set(key, draft);
    }
    return draft;
  }

  private setDraft(key: string, draft: string) {
    this.drafts.set(key, draft);
    writeStored(`soda-feed-draft:v1:${this.storageKey(key)}`, draft);
  }

  private revisionKey(edit: {repository: string; id: string}): string {
    return `soda-feed-revision:v2:${this.account()}:${edit.repository}:${edit.id}`;
  }

  private persistRevising() {
    const edit = this.revising;
    if (edit) writeStored(this.revisionKey(edit), JSON.stringify({body: edit.body, original: edit.original}));
  }

  private onDraft = (event: Event) => {
    const area = event.target;
    if (!(area instanceof HTMLTextAreaElement)) return;
    if (this.revising && (area.hasAttribute('data-revision-body') || area.hasAttribute('data-revision-original'))) {
      if (area.hasAttribute('data-revision-body')) this.revising.body = area.value;
      else this.revising.original = area.value;
      this.persistRevising();
      return;
    }
    const key = area.dataset.feedDraft;
    if (!key) return;
    this.setDraft(key, area.value.slice(0, 4000));
    // Keystrokes never re-render: reading position and focus stay put.
    const submit = this.querySelector(`[data-comment-submit="${CSS.escape(key)}"]`);
    if (submit instanceof HTMLButtonElement)
      submit.disabled = area.value.trim() === '' || this.view.commentBusy.includes(key);
  };

  private save = (event: SubmitEvent) => {
    event.preventDefault();
    const form = event.currentTarget;
    if (!(form instanceof HTMLFormElement) || !this.current) return;
    const context = this.current;
    const serial = this.serial;
    const data = new FormData(form);
    const repositories = [...data.getAll('repository')].flatMap((value) => {
      const numeric = typeof value === 'string' ? Number(value) : NaN;
      return Number.isSafeInteger(numeric) && numeric > 0 ? [numeric] : [];
    });
    const selection = data.get('kind');
    const kinds = feedEntryKinds.filter((kind) => selection === kind);
    const body: {repositories: number[]; kinds?: FeedEntryKind[]; expected_revision: number} = {
      repositories: [...new Set(repositories)].sort((a, b) => a - b),
      expected_revision: Number(data.get('expected_revision')),
    };
    if (kinds.length > 0) body.kinds = kinds;
    this.view = {...this.view, busy: true};
    void (async () => {
      try {
        const response = await context.transport.request('factory/feed/subscription', {
          method: 'PUT',
          headers: {'Content-Type': 'application/json'},
          body: JSON.stringify(body),
          signal: AbortSignal.timeout(15000),
        });
        if (!response.ok) {
          if (response.status === 409)
            throw Error('Feed subscription changed underneath the write. Refresh before saving again.');
          throw Error(failureMessage(response.status, 'Could not save the feed subscription.'));
        }
        if (serial !== this.serial) return;
        this.view = {...this.view, busy: false, message: 'Followed projects updated.'};
        await this.reload();
      } catch (error) {
        if (serial === this.serial)
          this.view = {
            ...this.view,
            busy: false,
            message: error instanceof Error ? error.message : 'Could not save the feed subscription.',
          };
      }
    })();
  };

  /** New request (the Feed place's ?compose=): open the comment box and focus it. */
  set compose(compose: boolean) {
    if (compose === this.composeRequested) return;
    this.composeRequested = compose;
    if (!compose) return;
    this.commentOpen = true;
    this.requestUpdate();
    void this.updateComplete.then(() =>
      this.querySelector<HTMLTextAreaElement>('.soda-feed-comment-box textarea')?.focus()
    );
  }

  /** The post shown alone: the Feed place's ?post= (Knowledge's "Open source post"). */
  set post(post: string | undefined) {
    const valid = post && commentID.test(post) ? post : undefined;
    if (valid === this.focusPost) return;
    this.focusPost = valid;
    if (valid) void this.openSourcePost(valid);
    else if (this.focusedEntry) this.leaveFocusedPost();
  }

  private async openSourcePost(sourceId: string) {
    if (!this.current) return;
    const context = this.current;
    const serial = this.serial,
      token = ++this.librarySerial;
    this.librarySourcePending = sourceId;
    this.requestUpdate();
    try {
      const value = object(await readJSON(context.transport, 'factory/feed/comments/' + sourceId));
      check(context.global === true || String(value.repository) === context.repository);
      const source = parseCommentDetail(value);
      if (serial !== this.serial || token !== this.librarySerial) return;
      const entry: FeedEntry = {
        kind: 'post',
        id: source.commentId,
        repository: Number(value.repository),
        seq: 0,
        at: source.at,
        author: source.authorName,
        comment: source,
        ...(source.actor ? {actor: source.actor} : {}),
        ...(source.communication ? {payload: {communication: source.communication}} : {}),
      };
      this.focusedEntry = entry;
      this.openDrawer(feedEntryKey(entry));
    } catch (error) {
      if (serial === this.serial && token === this.librarySerial)
        this.view = {
          ...this.view,
          message: error instanceof Error ? error.message : 'Source conversation is unavailable.',
        };
    } finally {
      if (token === this.librarySerial) {
        this.librarySourcePending = undefined;
        this.requestUpdate();
      }
    }
  }

  private leaveFocusedPost() {
    this.librarySerial++;
    this.librarySourcePending = undefined;
    this.focusedEntry = undefined;
    this.requestUpdate();
  }

  // The subscription's subscriber is the viewer's own native id; the
  // repository feed reads it only to know which requests are the viewer's.
  private async readViewerID(context: FactoryFeedContext) {
    try {
      const subscriber = parseFeedSubscriptionView(
        await readJSON(context.transport, 'factory/feed/subscription')
      ).subscriber;
      if (this.current !== context) return;
      this.viewerID = subscriber;
      this.requestUpdate();
    } catch {
      // Without the viewer's id no Open request link shows.
    }
  }

  // Open app ↗ appears on a result card when a Ready launch row, or a Ready
  // administered record with an origin (operator and app administrators),
  // matches its verified delivery.
  private async readAppOrigins(context: FactoryFeedContext) {
    try {
      const inventory = applicationsResponse(await readJSON(context.transport, 'applications'));
      if (this.current !== context) return;
      const origins = new Map<string, string>();
      for (const launch of inventory.launchable)
        if (launch.ready && launch.delivery_id && launch.application.origin)
          origins.set(launch.delivery_id, launch.application.origin);
      for (const record of inventory.applications) {
        const origin = record.application.origin;
        if (record.observation?.ready && record.delivery?.delivery_id && origin)
          origins.set(record.delivery.delivery_id, origin);
      }
      this.appOrigins = origins;
      this.requestUpdate();
    } catch {
      // No app inventory: no Open app ↗; App versions still links the App place.
    }
  }

  /** SodaOS place links for a repository whose owner/repo name is known. */
  private placesFor(repository: number): SodaPlaces | undefined {
    const ref = this.view.labels.refs.get(repository);
    if (ref?.path === undefined || !/^[^/]+\/[^/]+$/.test(ref.label)) return undefined;
    try {
      return repositoryPlaces(this.current?.prefix ?? '', ref.label);
    } catch {
      return undefined;
    }
  }

  /** The repository feed's own places (undefined on /feed). */
  private ownPlaces(): SodaPlaces | undefined {
    const name = this.current?.global ? undefined : this.current?.repositoryName;
    if (!name) return undefined;
    try {
      return repositoryPlaces(this.current?.prefix ?? '', name);
    } catch {
      return undefined;
    }
  }

  private cardLinks(entry: FeedEntry): FeedCardLinks {
    const places = this.placesFor(entry.repository);
    const delivery = entry.eventIdentity;
    const appOrigin = delivery ? this.appOrigins.get(delivery) : undefined;
    return {
      ...(places ? {places} : {}),
      ...(this.viewerID ? {viewer: this.viewerID} : {}),
      ...(delivery ? {delivery} : {}),
      ...(appOrigin ? {appOrigin} : {}),
    };
  }

  /** Same-origin navigation: the host swaps SodaOS places in its shell. */
  private go(path: string) {
    const navigate = this.current?.transport.navigate;
    if (navigate) navigate(path);
    else location.assign(path);
  }

  // A draft's idempotency key survives failed submits, so a retry recovers
  // the original server-side; a confirmed submit rotates it.
  private draftIdempotencyKey(key: string): string {
    const storage = `soda-feed-draftkey:v2:${this.storageKey(key)}`;
    const stored = readStored(storage);
    if (/^[A-Za-z0-9_-]{1,64}$/.test(stored)) return stored;
    const minted = newIdempotencyKey();
    writeStored(storage, minted);
    return minted;
  }

  // submitComment saves one draft. A failed submit keeps the draft; the
  // pending payload is kept with it, so a retry after a lost answer resends
  // the original text while newer writing stays in the box.
  private async submitComment(key: string) {
    const context = this.current;
    const entry = this.entryForKey(key);
    if (!context || this.view.commentBusy.includes(key)) return;
    const repository = entry?.repository ?? (key === this.projectCommentKey() ? Number(context.repository) : 0);
    const text = this.draftFor(key);
    if (!Number.isSafeInteger(repository) || repository <= 0 || text.trim() === '') return;
    const pendingKey = `soda-feed-pending:v1:${this.storageKey(key)}`;
    const idempotency = this.draftIdempotencyKey(key);
    let post: CommentPostBody = {repository, body: text, idempotency_key: idempotency};
    if (entry?.kind === 'post' && commentID.test(entry.id)) post.reply_to = entry.id;
    if (entry?.issue) post.issue = entry.issue;
    const source = entry ? commentSourceForEntry(entry) : undefined;
    if (source) {
      post.source_kind = source.kind;
      post.source_id = source.id;
    }
    try {
      const saved = JSON.parse(readStored(pendingKey) || 'null') as CommentPostBody | null;
      if (saved?.idempotency_key === idempotency && saved.repository === repository) post = saved;
    } catch {
      /* A broken pending copy is replaced below. */
    }
    writeStored(pendingKey, JSON.stringify(post));
    const serial = this.serial;
    if (entry) this.openDrawer(key);
    this.view = {...this.view, commentBusy: [...this.view.commentBusy, key]};
    try {
      const response = await context.transport.request('factory/feed/comments', {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify(post),
        signal: AbortSignal.timeout(30000),
      });
      if (!response.ok) {
        if (response.status === 404)
          throw Error('That work is no longer visible here, or comment intake is not served yet. The draft is kept.');
        throw Error(failureMessage(response.status, 'Could not save the reply. The draft is kept.'));
      }
      if (serial !== this.serial) return;
      writeStored(`soda-feed-draftkey:v2:${this.storageKey(key)}`, '');
      writeStored(pendingKey, '');
      if (this.draftFor(key) === post.body) this.setDraft(key, '');
      this.view = {
        ...this.view,
        commentBusy: this.view.commentBusy.filter((row) => row !== key),
        message: this.draftFor(key)
          ? 'Previous comment saved. Your newer edits are kept, ready to send.'
          : 'Comment saved.',
      };
      await this.readLatest(false);
      if (entry) void this.loadReplies(entry);
    } catch (error) {
      if (serial === this.serial)
        this.view = {
          ...this.view,
          commentBusy: this.view.commentBusy.filter((row) => row !== key),
          message: error instanceof Error ? error.message : 'Could not save the reply. The draft is kept.',
        };
    }
  }

  // openRoleReply resolves a "Read the role's reply" link inside its
  // original thread: the thread opens and pages until the target comment or
  // exhaustion, then the row scrolls into view and takes focus.
  private async openRoleReply(link: HTMLElement) {
    const targetId = link.dataset.roleReplyOpen ?? '';
    const key = link.dataset.roleReplyEntry ?? '';
    const entry = key ? this.entryForKey(key) : undefined;
    const serial = this.serial;
    const say = (message: string) => {
      if (serial !== this.serial || !this.openDrawers.has(key)) return;
      const region = this.querySelector(`[data-role-reply-open="${targetId}"]`)
        ?.closest('.soda-feed-consultation')
        ?.querySelector('[data-role-reply-status]');
      if (region) region.textContent = message;
      else this.view = {...this.view, message};
    };
    if (!commentID.test(targetId)) return;
    if (!entry) {
      this.view = {
        ...this.view,
        message: 'The role’s reply is not visible here. Clear the filters to check the full conversation.',
      };
      return;
    }
    this.openDrawer(key);
    say('Looking for the role’s reply…');
    for (let pages = 0; pages < 25; pages++) {
      await this.loadReplies(entry, pages > 0);
      if (serial !== this.serial || !this.openDrawers.has(key)) return;
      const thread = this.threads.get(key);
      if (!thread || thread.message) return say(thread?.message || 'The role’s reply could not be read here.');
      if (thread.items.some((item) => item.commentId === targetId)) {
        this.requestUpdate();
        await this.updateComplete;
        const row = this.querySelector('#comment-' + targetId);
        if (row instanceof HTMLElement) {
          row.scrollIntoView({block: 'center'});
          row.setAttribute('tabindex', '-1');
          row.focus({preventScroll: true});
        }
        return say('');
      }
      if (!thread.next) break;
    }
    say('The role’s reply is not in this thread. It may have been removed, or older replies need Show more.');
  }

  private openDrawer(key: string) {
    this.openDrawers.add(key);
    const entry = this.entryForKey(key);
    if (entry && !this.threads.has(key)) void this.loadReplies(entry);
    this.requestUpdate();
  }

  // reviewProductChange opens the owner's review of a same-product change:
  // the first open saves the change as a draft, later opens open the build
  // request it lives in.
  private async reviewProductChange(comment: string, revision: number, repository: number) {
    const context = this.current,
      serial = this.serial;
    if (!context || !commentID.test(comment) || this.view.commentBusy.includes(comment)) return;
    this.view = {...this.view, commentBusy: [...this.view.commentBusy, comment]};
    try {
      const path = `factory/feed/comments/${comment}`;
      let state = parseCommentDetail(await readJSON(context.transport, path)).pipeline;
      if (state?.productChange?.state === 'review_required' && state.productChange.canReview)
        state = parseCommentDetail(
          await readJSON(context.transport, path + '/review', {
            method: 'POST',
            headers: {'Content-Type': 'application/json'},
            body: JSON.stringify({revision}),
          })
        ).pipeline;
      if (serial !== this.serial) return;
      const change = state?.productChange;
      check(change && change.state !== 'superseded');
      let message = '';
      if (change.state === 'unavailable')
        message = 'The proposed change detail is currently unavailable. Check progress to reload it.';
      else if (!change.canReview)
        message =
          change.state === 'accepted'
            ? 'The changed direction is accepted. Follow its build and verification in the linked evidence.'
            : 'Waiting for the request owner to review this proposed change.';
      else {
        message =
          change.state === 'accepted'
            ? 'Opening the accepted change.'
            : 'Change saved for review. Review and accept the direction in the same build request.';
        // The proposed change opens on its request's page (at Update proposal).
        const places = this.placesFor(repository);
        if (places) this.go(places.request(change.intent));
      }
      this.view = {...this.view, message};
      void this.readLatest(false);
    } catch {
      if (serial === this.serial)
        this.view = {
          ...this.view,
          message:
            'This change could not be opened. Refresh the conversation to check its current direction and your access; the comment is kept.',
        };
    } finally {
      if (serial === this.serial)
        this.view = {...this.view, commentBusy: this.view.commentBusy.filter((row) => row !== comment)};
    }
  }

  // openRevision loads one request for revising and restores any unsent
  // text kept for it.
  private async openRevision(comment: string) {
    const context = this.current,
      serial = this.serial;
    if (!context) return;
    try {
      const data = object(await readJSON(context.transport, `factory/feed/comments/${comment}`));
      if (serial !== this.serial || data.can_retry !== true) return;
      const edit: Revising = {
        id: String(data.comment_id),
        revision: Number(data.revision),
        repository: String(data.repository ?? context.repository),
        body: String(data.body),
      };
      if (data.route === 'clarify') {
        edit.original = edit.body;
        edit.body = '';
        edit.question =
          typeof data.detail === 'string' && data.detail
            ? data.detail
            : 'What should change, and where should it apply?';
      }
      try {
        const saved = object(JSON.parse(readStored(this.revisionKey(edit)) || 'null'));
        if (typeof saved.body === 'string') edit.body = saved.body;
        if (typeof saved.original === 'string') edit.original = saved.original;
      } catch {
        /* Nothing kept for this request. */
      }
      this.revising = edit;
      this.requestUpdate();
      await this.updateComplete;
      this.querySelector<HTMLElement>('[data-revision-body]')?.focus();
    } catch {
      this.view = {...this.view, message: 'Could not load this request. Refresh and try again.'};
    }
  }

  // reviseComment sends the revised request against the revision it was
  // opened at. A stale revision is refused; the server's current text then
  // shows beside the kept draft and the next submit uses the new revision.
  private async reviseComment() {
    const edit = this.revising,
      context = this.current,
      serial = this.serial;
    if (!edit || !context || this.view.busy || !edit.body.trim()) return;
    const body = edit.original === undefined ? edit.body : edit.original + '\n\nClarification: ' + edit.body;
    if ([...body].length > 4000) {
      this.view = {
        ...this.view,
        message:
          'Keep the original request and answer within 4,000 characters. You can shorten the original request below.',
      };
      return;
    }
    const sent = {body: edit.body, original: edit.original};
    this.view = {...this.view, busy: true};
    try {
      const response = await context.transport.request(`factory/feed/comments/${edit.id}/retry`, {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify({expected_revision: edit.revision, body}),
        signal: AbortSignal.timeout(15000),
      });
      if (serial !== this.serial) return;
      if (response.status === 409) {
        const live = object(await readJSON(context.transport, `factory/feed/comments/${edit.id}`));
        if (serial !== this.serial) return;
        if (this.revising?.id === edit.id)
          this.revising = {...this.revising, revision: Number(live.revision), current: String(live.body ?? '')};
        this.view = {
          ...this.view,
          busy: false,
          message:
            'This request changed since you opened it. Its current text is shown; your draft is kept for another submit.',
        };
        return;
      }
      const value = object(await readSodaJSON(response));
      if (!response.ok) throw Error(String(object(value.error ?? {}).message ?? 'Could not update this request.'));
      const live = this.revising;
      const newer = live?.id === edit.id && (live.body !== sent.body || live.original !== sent.original);
      if (newer && live) {
        this.revising = {...live, revision: typeof value.revision === 'number' ? value.revision : live.revision + 1};
        delete this.revising.current;
        this.persistRevising();
      } else {
        writeStored(this.revisionKey(edit), '');
        if (live?.id === edit.id) this.revising = undefined;
      }
      this.view = {
        ...this.view,
        busy: false,
        message: newer
          ? 'Request updated. Processing will continue here. Your newer edits are kept, ready to send.'
          : 'Request updated. Processing will continue here.',
      };
      await this.readLatest(false);
    } catch (error) {
      if (serial === this.serial)
        this.view = {
          ...this.view,
          busy: false,
          message: error instanceof Error ? error.message : 'Could not update this request. Your edits are kept.',
        };
    }
  }

  private onFeedClick = (event: Event) => {
    const target = event.target;
    if (!(target instanceof HTMLElement)) return;
    const closest = (selector: string) => target.closest<HTMLElement>(selector);
    if (
      this.fake &&
      closest(
        '[data-comment-submit], [data-comment-product-change], [data-comment-revise], [data-revision-submit], [data-role-reply-open]'
      )
    ) {
      // Nothing written in the fake feed reaches the project.
      this.view = {
        ...this.view,
        message: 'This is the fake feed: nothing is sent. Pick another tab for the real feed.',
      };
      return;
    }
    const productChange = closest('[data-comment-product-change]');
    if (productChange) {
      void this.reviewProductChange(
        productChange.dataset.commentProductChange ?? '',
        Number(productChange.dataset.productRevision),
        Number(productChange.dataset.productRepository)
      );
      return;
    }
    const drawer = closest('[data-feed-drawer-toggle]')?.dataset.feedDrawerToggle;
    if (drawer) {
      if (this.openDrawers.delete(drawer)) this.requestUpdate();
      else this.openDrawer(drawer);
      return;
    }
    const roleReply = closest('[data-role-reply-open]');
    if (roleReply) {
      event.preventDefault();
      void this.openRoleReply(roleReply);
      return;
    }
    const revise = closest('[data-comment-revise]')?.dataset.commentRevise;
    if (revise) {
      void this.openRevision(revise);
      return;
    }
    if (closest('[data-revision-cancel]')) {
      if (this.revising) writeStored(this.revisionKey(this.revising), '');
      this.revising = undefined;
      this.requestUpdate();
      return;
    }
    if (closest('[data-revision-submit]')) {
      void this.reviseComment();
      return;
    }
    const answer = closest('[data-feed-answer]');
    if (answer instanceof HTMLButtonElement && answer.dataset.feedAnswer) {
      const key = answer.dataset.feedAnswer,
        text = answer.dataset.feedAnswerText ?? '';
      const area = this.querySelector<HTMLTextAreaElement>(`[data-feed-draft="${CSS.escape(key)}"]`);
      if (area) {
        // Selecting a choice edits the draft; only Submit invokes JEV.
        area.value = (area.value.trim() ? `${area.value}\n${text}` : text).slice(0, 4000);
        this.openDrawer(key);
        area.dispatchEvent(new Event('input', {bubbles: true}));
        void this.updateComplete.then(() =>
          this.querySelector<HTMLTextAreaElement>(`[data-feed-draft="${CSS.escape(key)}"]`)?.focus()
        );
      }
      return;
    }
    const submit = closest('[data-comment-submit]')?.dataset.commentSubmit;
    if (submit) {
      void this.submitComment(submit);
      return;
    }
  };

  // loadReplies reads one thread's newest page, or with `older` the page
  // below what is shown. A second caller joins the read in flight.
  private loadReplies(entry: FeedEntry, older = false): Promise<void> {
    const key = feedEntryKey(entry);
    const running = this.threadReads.get(key);
    if (running) return running;
    const run = this.readThreadPage(entry, older).finally(() => this.threadReads.delete(key));
    this.threadReads.set(key, run);
    return run;
  }

  private async readThreadPage(entry: FeedEntry, older: boolean): Promise<void> {
    const context = this.current,
      source = commentSourceForEntry(entry),
      serial = this.serial;
    if (!context || !source) return;
    const key = feedEntryKey(entry);
    const previous = this.threads.get(key);
    const before = older ? (previous?.next ?? 0) : 0;
    if (older && before === 0) return;
    this.threads.set(key, {
      items: previous?.items ?? [],
      next: previous?.next ?? 0,
      pagedOlder: false,
      ...previous,
      busy: !previous || older,
      message: '',
    });
    this.requestUpdate();
    try {
      const query = new URLSearchParams({
        repository: String(entry.repository),
        source_kind: source.kind,
        source_id: source.id,
      });
      if (before > 0) query.set('before', String(before));
      const result = object(await readJSON(context.transport, 'factory/feed/replies?' + query));
      check(Array.isArray(result.items));
      // Pages arrive newest-first; threads read oldest-first.
      const page = result.items
        .map(parseCommentDetail)
        .filter((item) => !item.removed)
        .reverse();
      if (serial !== this.serial) return;
      const kept = older ? (this.threads.get(key)?.items ?? []) : [];
      const ids = new Set(page.map((item) => item.commentId));
      this.threads.set(key, {
        items: [...page, ...kept.filter((item) => !ids.has(item.commentId))],
        next: typeof result.next_before === 'number' ? result.next_before : 0,
        busy: false,
        message: '',
        pagedOlder: older,
      });
    } catch (error) {
      if (serial !== this.serial) return;
      const denied = error instanceof Error && (error as {status?: number}).status === 403;
      const failed = this.threads.get(key);
      this.threads.set(key, {
        items: failed?.items ?? [],
        next: failed?.next ?? 0,
        busy: false,
        pagedOlder: failed?.pagedOlder ?? false,
        message: denied && error instanceof Error ? error.message : 'Replies could not be loaded. Try Refresh replies.',
      });
    }
    this.requestUpdate();
  }

  protected render() {
    const confirmed = activityTimestamp(this.view.confirmedUnix);
    return html`<section class="soda-feed" aria-label="Feed" @click=${this.onFeedClick} @input=${this.onDraft}>
      ${this.current?.global ? '' : this.renderProjectComment()}
      <header class="soda-feed-toolbar">
        ${this.renderViews()}
        <span class="soda-feed-toolbar-actions">
          ${
            confirmed
              ? html`<span class="soda-feed-updated"
                  >Updated
                  <time datetime=${confirmed.iso} title=${confirmed.text}
                    >${shortTime(this.view.confirmedUnix)}</time
                  ></span
                >`
              : ''
          }
          <button
            type="button"
            class="ui button soda-feed-utility soda-feed-icon-button"
            aria-label="Refresh feed"
            title="Refresh feed"
            ?disabled=${this.view.busy}
            @click=${() => void this.readLatest(true)}
          >
            <svg
              viewBox="0 0 24 24"
              aria-hidden="true"
              fill="none"
              stroke="currentColor"
              stroke-width="1.6"
              stroke-linecap="round"
              stroke-linejoin="round"
            >
              <path d="M20 12a8 8 0 1 1-2.34-5.66M20 3v4h-4" />
            </svg>
          </button>
          ${
            this.ownPlaces()
              ? html`<a class="soda-feed-utility" href=${this.ownPlaces()?.knowledge() ?? ''}>Knowledge</a>`
              : ''
          }
        </span>
      </header>
      ${this.renderFilters()}
      ${
        this.focusedEntry
          ? html`<p>
              Source conversation
              <button
                type="button"
                @click=${() => {
                  this.leaveFocusedPost();
                  // A post opened by its address goes back to the Feed's own address.
                  const places = this.focusPost ? this.ownPlaces() : undefined;
                  if (places) this.go(places.feed());
                }}
              >
                Back to feed
              </button>
            </p>`
          : ''
      }
      <p data-feed-message aria-live="polite">${this.view.message}</p>
      ${
        this.view.allowance.length > 0
          ? html`<p class="soda-feed-brief soda-feed-alert" role="alert" data-feed-allowance>
              ${allowanceHoldText(this.view.allowance)}
            </p>`
          : ''
      }
      ${this.view.readFailed && this.view.entries.length ? html`<p class="soda-feed-updated">Showing the last loaded activity. Refresh to check for updates.</p>` : ''}
      ${this.renderRevision()}
      ${
        this.view.pagedBack
          ? html`<button type="button" class="ui button" ?disabled=${this.view.busy} @click=${this.backToLatest}>
              Back to latest
            </button>`
          : ''
      }
      ${this.current?.global ? this.renderSubscription() : ''} ${this.renderList()}
    </section>`;
  }

  private renderViews() {
    const views = [
      ['all', 'All'],
      ['attention', 'Needs me'],
      ['decisions', 'Choices & approvals'],
      ['evidence', 'Evidence'],
      ['fake', 'Fake feed'],
    ];
    return html`<nav class="soda-feed-tabs" aria-label="Filter conversation">
      ${views.map(
        ([value, label]) => html`<button
          type="button"
          aria-pressed=${this.filters.view === value ? 'true' : 'false'}
          @click=${() => this.changeFilters({view: value ?? 'all'})}
        >
          ${label}
        </button>`
      )}
    </nav>`;
  }

  private renderFilters() {
    const roles = [
      'human',
      'product_lead',
      'architect',
      'builder',
      'inspector',
      'fixer',
      'strategist',
      'test_pilot',
      'jev',
      'factory',
    ];
    return html`<div class="soda-feed-filters">
      <form
        @submit=${(event: SubmitEvent) => {
          event.preventDefault();
          if (event.currentTarget instanceof HTMLFormElement)
            this.changeFilters({search: String(new FormData(event.currentTarget).get('search') ?? '').trim()});
        }}
      >
        <span class="soda-feed-search">
          <svg
            viewBox="0 0 24 24"
            aria-hidden="true"
            fill="none"
            stroke="currentColor"
            stroke-width="1.8"
            stroke-linecap="round"
          >
            <circle cx="11" cy="11" r="6.5" />
            <path d="m16 16 4.5 4.5" />
          </svg>
          <input
            type="search"
            name="search"
            aria-label="Search conversation"
            placeholder="Search posts"
            maxlength="256"
            .value=${this.filters.search}
          />
        </span>
        <button type="submit" class="soda-visually-hidden">Search</button>
        <select
          aria-label="Filter by role"
          .value=${this.filters.role}
          @change=${(event: Event) => {
            if (event.target instanceof HTMLSelectElement) this.changeFilters({role: event.target.value});
          }}
        >
          <option value="">All authors</option>
          ${roles.map((role) => html`<option value=${role}>${role === 'human' ? 'People' : roleLabel(role)}</option>`)}
        </select>
        <label
          ><input
            type="checkbox"
            .checked=${this.filters.current}
            @change=${(event: Event) => {
              if (event.target instanceof HTMLInputElement) this.changeFilters({current: event.target.checked});
            }}
          />Current knowledge only</label
        >
      </form>
    </div>`;
  }

  private renderRevision() {
    const edit = this.revising;
    if (!edit) return '';
    return html`<section class="soda-feed-revision" aria-label="Revise request">
      ${
        edit.current !== undefined
          ? html`<p role="alert">This request changed since you opened it. Its current text:</p>
              <blockquote>${edit.current}</blockquote>`
          : ''
      }
      ${
        edit.original !== undefined
          ? html`<details>
              <summary>Original request · edit if needed</summary>
              <label
                >Original request<textarea
                  data-revision-original
                  rows="4"
                  maxlength="4000"
                  .value=${edit.original}
                ></textarea>
              </label>
            </details>`
          : ''
      }
      <label
        >${edit.question ?? 'Update your request'}<textarea
          data-revision-body
          rows="5"
          maxlength="4000"
          .value=${edit.body}
        ></textarea>
      </label>
      <p>Submitting reassesses this request and may start work using the project’s existing allowance.</p>
      <button type="button" data-revision-submit ?disabled=${this.view.busy}>
        ${edit.question ? 'Submit answer' : 'Submit updated request'}</button
      ><button type="button" data-revision-cancel>Cancel</button>
    </section>`;
  }

  private renderProjectComment() {
    const context = this.current;
    if (!context || !id(context.repository)) return html``;
    const key = this.projectCommentKey();
    const draft = this.draftFor(key);
    return html`<div class="soda-feed-comment-box">
      ${renderComposer({
        key,
        draft,
        label: 'Comment',
        submitLabel: 'Post',
        submitAvailable: draft.trim() !== '' && !this.view.commentBusy.includes(key),
        collapsed: !this.commentOpen && draft.trim() === '',
      })}
    </div>`;
  }

  private renderSubscription() {
    const subscription = this.view.subscription;
    if (!subscription || !this.current?.global) return html``;
    const candidates = [...this.view.labels.refs.entries()].sort((a, b) => a[0] - b[0]);
    // Until a subscription is saved, every candidate shows and is checked.
    const member = new Set(
      subscription.revision === 0 ? candidates.map(([repository]) => repository) : subscription.repositories
    );
    return html`<details class="soda-feed-subscription">
      <summary>Choose repositories</summary>
      <form @submit=${this.save}>
        <fieldset>
          <legend>Followed repositories</legend>
          <input type="hidden" name="expected_revision" value=${String(subscription.revision)} />
          ${subscription.revision === 0 ? html`<p>Showing every repository you can read until you choose.</p>` : ''}
          ${candidates.map(
            ([repository, ref]) => html`<label
              ><input
                type="checkbox"
                name="repository"
                value=${String(repository)}
                .checked=${member.has(repository)}
                ?disabled=${this.view.busy}
              />${ref.label}</label
            >`
          )}
        </fieldset>
        <fieldset>
          <legend>Show</legend>
          <select name="kind" aria-label="Activity type" ?disabled=${this.view.busy}>
            <option value="all" .selected=${subscription.kinds.length !== 1}>All activity</option>
            <option value="record" .selected=${subscription.kinds.length === 1 && subscription.kinds[0] === 'record'}>
              Worker activity
            </option>
            <option value="post" .selected=${subscription.kinds.length === 1 && subscription.kinds[0] === 'post'}>
              Comments
            </option>
          </select>
        </fieldset>
        <button class="ui button" type="submit" ?disabled=${this.view.busy}>Save feed subscription</button>
      </form>
    </details>`;
  }

  private repositoryRef(repository: number) {
    const ref = this.view.labels.refs.get(repository);
    const places = this.placesFor(repository);
    return {
      id: repository,
      path: ref?.path,
      label: ref?.label ?? `repository ${repository}`,
      // A repository's own feed needs no link back to itself.
      current: !this.current?.global,
      ...(places ? {places} : {}),
    };
  }

  private renderList() {
    const entries = this.focusedEntry ? [this.focusedEntry] : this.view.entries;
    if (entries.length === 0) return this.renderEmptyFeed();
    // Small steps fold under the post they belong to; running work is one
    // post that updates in place.
    return html`<div class="soda-feed-list">
        ${repeat(
          threadFeed(entries),
          (item) => feedEntryKey(item.entry),
          (item) => {
            const key = feedEntryKey(item.entry);
            const post = describeFeedPost(
              item.entry,
              this.repositoryRef(item.entry.repository),
              item,
              this.draftFor(key),
              describeComments(key, this.openDrawers.has(key), this.threads.get(key)),
              this.view.commentBusy
            );
            return renderFeedPost(
              post,
              this.renderReplies(item.entry),
              this.current?.transport.request,
              this.current?.transport.assetBase,
              this.cardLinks(item.entry)
            );
          }
        )}
      </div>
      ${
        this.view.truncated && this.view.next !== ''
          ? html`<button
              type="button"
              class="ui button"
              ?disabled=${this.view.busy}
              @click=${() => void this.readOlder()}
            >
              Load older entries
            </button>`
          : ''
      }`;
  }

  private renderEmptyFeed() {
    if (this.view.busy) return html`<section class="soda-feed-empty" role="status"><p>Loading activity…</p></section>`;
    if (this.view.readFailed)
      return html`<section class="soda-feed-empty" aria-label="Feed unavailable">
        <h3>Activity could not be loaded.</h3>
        <p>Your draft is kept. Try loading the feed again.</p>
        <button type="button" class="ui button" @click=${() => void this.reload()}>Retry loading activity</button>
      </section>`;
    const filtered =
      this.filters.view !== 'all' || this.filters.search !== '' || this.filters.role !== '' || this.filters.current;
    if (filtered)
      return html`<section class="soda-feed-empty" aria-label="No matching posts">
        <h3>No matching posts.</h3>
        <p>Try another search or clear the filters to see all activity in this feed.</p>
        <button
          type="button"
          class="ui button"
          @click=${() => this.changeFilters({view: 'all', search: '', role: '', current: false})}
        >
          Clear filters
        </button>
      </section>`;
    return html`<section class="soda-feed-empty" aria-label="No activity">
      <h3>No activity yet.</h3>
      <p>
        ${this.current?.global ? 'Posts and comments from your followed projects will appear here.' : 'Questions, updates and decisions will appear here as they are recorded. Start a conversation in the comment box above.'}
      </p>
    </section>`;
  }

  private renderReplies(entry: FeedEntry) {
    if (!commentSourceForEntry(entry)) return html``;
    const key = feedEntryKey(entry),
      thread = this.threads.get(key);
    if (!thread) return html`<p role="status" class="soda-feed-thread-status">Loading replies…</p>`;
    // Consecutive JEV calls for one decision fold into one group.
    const groups: {key: string; items: CommentDetail[]}[] = [];
    for (const reply of thread.items) {
      const previous = groups.at(-1),
        prior = previous?.items.at(-1)?.consultation;
      if (
        reply.consultation &&
        prior &&
        reply.consultation.group_id === prior.group_id &&
        reply.consultation.assignment === prior.assignment
      )
        previous?.items.push(reply);
      else groups.push({key: reply.commentId, items: [reply]});
    }
    return html`<div class="soda-feed-thread">
      <p class="soda-feed-thread-head">
        <span role="status"
          >${thread.busy ? 'Loading replies…' : thread.message || (thread.items.length ? '' : 'No replies yet.')}</span
        >
        <button
          type="button"
          class="soda-feed-icon-button"
          aria-label="Refresh replies"
          title="Refresh replies"
          ?disabled=${thread.busy}
          @click=${() => void this.loadReplies(entry)}
        >
          <svg
            viewBox="0 0 24 24"
            aria-hidden="true"
            fill="none"
            stroke="currentColor"
            stroke-width="1.6"
            stroke-linecap="round"
            stroke-linejoin="round"
          >
            <path d="M20 12a8 8 0 1 1-2.34-5.66M20 3v4h-4" />
          </svg>
        </button>
      </p>
      <ul class="soda-feed-comments">
        ${repeat(
          groups,
          (group) => group.key,
          (group) =>
            group.items.length > 1
              ? html`<li class="soda-feed-consultation-group">
                  <details>
                    <summary>
                      ${group.items[0]?.authorName || 'Worker'} · ${group.items.length} recorded JEV calls for this
                      decision
                    </summary>
                    <ul class="soda-feed-comments">
                      ${repeat(
                        group.items,
                        (reply) => reply.commentId,
                        (reply) => this.renderReply(entry, reply)
                      )}
                    </ul>
                  </details>
                </li>`
              : this.renderReply(entry, group.items[0]!)
        )}
      </ul>
      ${thread.next ? html`<button type="button" class="soda-feed-more" ?disabled=${thread.busy} @click=${() => void this.loadReplies(entry, true)}>Show more</button>` : ''}
    </div>`;
  }

  private renderReply(entry: FeedEntry, reply: CommentDetail) {
    const portrait = feedRolePortraits[roleLabel(reply.actor) ?? ''];
    return html`<li class="soda-feed-comment" id=${'comment-' + reply.commentId} data-role=${reply.actor ?? 'human'}>
      <span class="soda-feed-avatar" aria-hidden="true"
        >${
          portrait
            ? html`<img src=${(this.current?.transport.assetBase ?? '') + portrait} alt="" width="30" height="30" />`
            : (reply.authorName || '?').slice(0, 1).toUpperCase()
        }</span
      >
      <div class="soda-feed-comment-main">
        <p class="soda-feed-comment-who">
          <strong>${reply.authorName || 'Comment'}</strong>
          <time datetime=${new Date(reply.at * 1000).toISOString()} title=${new Date(reply.at * 1000).toLocaleString()}
            >· ${relativeReplyTime(reply.at)}</time
          >
          ${reply.edited ? html` <span class="soda-feed-comment-edited">(edited)</span>` : ''}
        </p>
        ${reply.body && (reply.contribution || reply.consultation) ? html`<p class="soda-feed-body-text">${reply.body}</p>` : ''}
        ${
          reply.contribution
            ? renderContribution(reply.contribution, {entryKey: feedEntryKey(entry)})
            : reply.consultation
              ? renderConsultation(reply.consultation)
              : reply.communication
                ? renderFeedCommunication(
                    reply.communication,
                    'post:' + reply.commentId,
                    this.current?.transport.request
                  )
                : html`<p class="soda-feed-body-text">${reply.body}</p>`
        }
        ${
          reply.pipeline
            ? renderOutcome(
                describeCommentOutcome(
                  reply.pipeline,
                  this.repositoryRef(entry.repository),
                  this.view.commentBusy.includes(reply.commentId)
                ),
                this.current?.transport.assetBase
              )
            : ''
        }
      </div>
    </li>`;
  }
}

customElements.define('soda-factory-feed', SodaFactoryFeed);
