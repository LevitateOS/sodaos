import {LitElement, html} from 'lit';
import {repeat} from 'lit/directives/repeat.js';
import type {PreparedExtensionMount} from './soda-extension.js';
import {check, id, object, readSodaJSON} from './sodaspaces-api.js';
import {activityTimestamp, nativePath} from './sodaspaces-factory-activity.js';
import {repositoryChoices} from './sodaspaces-repository-response.js';
import {
  commentPostBody,
  commentSourceForEntry,
  commentUnsettled,
  feedEntryKinds,
  feedQueryString,
  mergeFeedEntries,
  newIdempotencyKey,
  parseCommentDetail,
  parseCommentPipelineState,
  parseFeedPageView,
  parseFeedSubscriptionView,
} from './sodaspaces-factory-feed-response.js';
import type {
  CommentPipelineState,
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
  renderDiscussion,
  renderFeedPost,
} from './sodaspaces-factory-feed-view.js';
import type {CommentOutcomeModel, PostBodyState} from './sodaspaces-factory-feed-view.js';
import './sodaspaces-factory-library.js';

export interface FactoryFeedContext {
  transport: PreparedExtensionMount;
  repository: string;
  repositoryName?: string;
  repositories?: ReadonlyMap<string, string>;
  account?: string;
  global?: boolean;
}

interface RepositoryLabels {
  refs: Map<number, {path: string | undefined; label: string}>;
  listedAll: boolean;
}

const feedPageSize = 20;

function failureMessage(status: number | undefined, fallback: string): string {
  if (status === 403)
    return 'One or more subscribed projects are not visible to you. The feed shows only permitted activity.';
  return fallback;
}

// Relative reply age for comment rows ("now", "12m", "2h", "3d"); older
// replies fall back to a short date. The full timestamp stays on the
// <time> element's title in the row markup.
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

function draftKey(key: string): string {
  return `soda-feed-draft:v1:${key}`;
}

function readDraft(key: string): string {
  try {
    return window.localStorage.getItem(draftKey(key)) ?? '';
  } catch {
    return '';
  }
}

function writeDraft(key: string, draft: string): void {
  try {
    if (draft === '') window.localStorage.removeItem(draftKey(key));
    else window.localStorage.setItem(draftKey(key), draft);
  } catch {
    /* Drafts stay in memory when browser storage is unavailable. */
  }
}

// Stored intake records bind one feed post to its submitted comments:
// comment identities plus the draft's idempotency key, so a retry recovers
// the original instead of duplicating it. Bounded per post; reads re-serve
// stored pipeline states only and never drive intake.
interface StoredIntake {
  commentId: string;
  key: string;
}

function intakeKey(key: string): string {
  return `soda-feed-intake:v1:${key}`;
}

function readIntake(key: string): StoredIntake[] {
  try {
    const raw = window.localStorage.getItem(intakeKey(key));
    if (!raw) return [];
    const parsed: unknown = JSON.parse(raw);
    if (!Array.isArray(parsed)) return [];
    return parsed.slice(0, 5).flatMap((row) => {
      if (typeof row !== 'object' || row === null) return [];
      const {commentId, key: idempotency} = row as {commentId?: unknown; key?: unknown};
      return typeof commentId === 'string' && typeof idempotency === 'string' ? [{commentId, key: idempotency}] : [];
    });
  } catch {
    return [];
  }
}

function writeIntake(key: string, records: StoredIntake[]): void {
  try {
    window.localStorage.setItem(intakeKey(key), JSON.stringify(records.slice(0, 5)));
  } catch {
    /* Intake records stay in memory when browser storage is unavailable. */
  }
}

async function readJSON(transport: PreparedExtensionMount, path: string, init?: RequestInit): Promise<unknown> {
  const response = await transport.request(path, {method: 'GET', signal: AbortSignal.timeout(15000), ...init});
  if (!response.ok) {
    const error = Error(failureMessage(response.status, 'The combined feed is unavailable.'));
    (error as {status?: number}).status = response.status;
    throw error;
  }
  return readSodaJSON(response);
}

export class SodaFactoryFeed extends LitElement {
  static properties = {view: {state: true}};
  declare private view: {
    message: string;
    busy: boolean;
    subscription: FeedSubscriptionView | undefined;
    labels: RepositoryLabels;
    entries: FeedEntry[];
    next: string;
    truncated: boolean;
    pagedBack: boolean;
    confirmedUnix: number;
    filterKinds: FeedEntryKind[];
    outcomes: Map<string, CommentPipelineState[]>;
    bodies: Map<string, PostBodyState>;
    commentBusy: string[];
  };
  private current: FactoryFeedContext | undefined;
  private serial = 0;
  private drafts = new Map<string, string>();
  private pendingPosts = new Map<string, CommentPostBody>();
  private reading = false;
  private refreshTimer?: ReturnType<typeof setInterval>;
  private revising: {id: string; revision: number; body: string; original?: string; question?: string} | undefined;
  private threads = new Map<string, {items: CommentDetail[]; next: number; busy: boolean; message: string}>();
  private openDrawers = new Set<string>();
  private library: {repository: string; item?: string} | undefined;

  protected createRenderRoot() {
    return this;
  }

  constructor() {
    super();
    this.view = {
      message: '',
      busy: false,
      subscription: undefined,
      labels: {refs: new Map(), listedAll: false},
      entries: [],
      next: '',
      truncated: false,
      pagedBack: false,
      confirmedUnix: 0,
      filterKinds: [],
      outcomes: new Map(),
      bodies: new Map(),
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
    this.reading = false;
    this.current = context;
    this.serial++;
    this.drafts.clear();
    this.threads.clear();
    this.openDrawers.clear();
    this.revising = undefined;
    this.library = undefined;
    this.view = {
      ...this.view,
      busy: false,
      message: '',
      subscription: undefined,
      entries: [],
      next: '',
      truncated: false,
      pagedBack: false,
      confirmedUnix: 0,
      outcomes: new Map(),
      bodies: new Map(),
      commentBusy: [],
    };
    if (context) {
      void this.reload();
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

  private scopeRepositories(): number[] {
    const subscription = this.view.subscription;
    if (this.current?.global) return subscription?.repositories ?? [];
    const current = Number(this.current?.repository);
    return Number.isSafeInteger(current) && current > 0 ? [current] : [];
  }

  private async reload() {
    const context = this.current;
    if (!context || this.view.busy) return;
    if (!context.global && !id(context.repository)) {
      this.view = {...this.view, message: 'The combined feed needs its current project binding.'};
      return;
    }
    const serial = this.serial;
    this.view = {...this.view, busy: true};
    try {
      const subscription = context.global
        ? parseFeedSubscriptionView(await readJSON(context.transport, 'factory/feed/subscription'))
        : ({
            subscriber: context.account ?? context.transport.generation,
            revision: 0,
            repositories: [Number(context.repository)],
            kinds: [],
          } as FeedSubscriptionView);
      if (serial !== this.serial) return;
      this.view = {...this.view, subscription};
      const labels = await this.resolveLabels(context, subscription);
      if (serial !== this.serial) return;
      this.view = {...this.view, labels};
      await this.readLatest(true);
    } catch (error) {
      if (serial === this.serial)
        this.view = {
          ...this.view,
          busy: false,
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

  private async readLatest(reset: boolean) {
    const context = this.current;
    if (!context || this.reading) return;
    const serial = this.serial;
    const scope = this.scopeRepositories();
    if (scope.length === 0) {
      this.view = {
        ...this.view,
        busy: false,
        message: 'Follow at least one project below before reading the combined feed.',
      };
      return;
    }
    this.reading = true;
    this.view = {...this.view, busy: true};
    const anchor = Array.from(this.querySelectorAll<HTMLElement>('.soda-feed-post')).find(
      (el) => el.getBoundingClientRect().bottom > 0
    );
    const anchorTop = anchor?.getBoundingClientRect().top;
    try {
      const page = parseFeedPageView(
        await readJSON(
          context.transport,
          feedQueryString({repositories: scope, kinds: this.view.filterKinds, limit: feedPageSize, cursor: ''}) +
            (context.global ? '' : '&scope=project')
        )
      );
      if (serial !== this.serial) return;
      const refs = new Map(this.view.labels.refs);
      for (const entry of page.entries) {
        if (entry.project && nativePath('/' + entry.project))
          refs.set(entry.repository, {path: '/' + entry.project, label: entry.project});
      }
      const fresh = page.entries.filter(
        (entry) => !this.view.entries.some((known) => known.kind === entry.kind && known.id === entry.id)
      );
      this.view = {
        ...this.view,
        labels: {...this.view.labels, refs},
        busy: false,
        entries: mergeFeedEntries(this.view.entries, page.entries),
        next: this.view.pagedBack ? this.view.next : page.next,
        truncated: this.view.pagedBack ? this.view.truncated : page.truncated,
        confirmedUnix: Math.floor(Date.now() / 1000),
        message:
          reset && !fresh.length && this.view.entries.length > 0 ? 'Feed refreshed; no new posts.' : this.view.message,
      };
      if (reset && fresh.length > 0 && this.view.entries.length > fresh.length)
        this.view = {
          ...this.view,
          message: `${fresh.length} new post${fresh.length === 1 ? '' : 's'}.`,
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
      // Stored outcomes and post text re-serve without re-running intake.
      void this.refreshOutcomes(serial);
      void this.refreshPostBodies(serial);
    } catch (error) {
      if (serial === this.serial) {
        // A failed refresh keeps the last confirmed feed and says so; stale
        // data is never presented as current or cleared into an empty result.
        this.view = {
          ...this.view,
          busy: false,
          message: error instanceof Error ? error.message : 'The combined feed is unavailable.',
        };
      }
    } finally {
      if (serial === this.serial) this.reading = false;
    }
  }

  private async readOlder() {
    const context = this.current;
    if (!context || this.view.busy || this.view.next === '') return;
    const serial = this.serial;
    this.view = {...this.view, busy: true};
    try {
      const page = parseFeedPageView(
        await readJSON(
          context.transport,
          feedQueryString({
            repositories: this.scopeRepositories(),
            kinds: this.view.filterKinds,
            limit: feedPageSize,
            cursor: this.view.next,
          }) + (context.global ? '' : '&scope=project')
        )
      );
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
      void this.refreshOutcomes(serial);
      void this.refreshPostBodies(serial);
    } catch (error) {
      if (serial === this.serial)
        this.view = {
          ...this.view,
          busy: false,
          message: error instanceof Error ? error.message : 'Older feed entries are unavailable.',
        };
    }
  }

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
    const expected = Number(data.get('expected_revision'));
    if (!Number.isSafeInteger(expected) || expected < 0) {
      this.view = {...this.view, message: 'Feed subscription write refused: missing revision guard.'};
      return;
    }
    const body: {repositories: number[]; kinds?: FeedEntryKind[]; expected_revision: number} = {
      repositories: [...new Set(repositories)].sort((a, b) => a - b),
      expected_revision: expected,
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
        const saved = object(await readSodaJSON(response));
        check(typeof saved.revision === 'number');
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

  private onDraft = (event: Event) => {
    const area = event.target;
    if (!(area instanceof HTMLTextAreaElement)) return;
    if (area.hasAttribute('data-revision-body') && this.revising) {
      this.revising.body = area.value;
      return;
    }
    const key = area.dataset.feedDraft;
    if (!key) return;
    this.drafts.set(key, area.value.slice(0, 4000));
    writeDraft(this.storageKey(key), area.value.slice(0, 4000));
    // Draft keystrokes never re-render: reading position and focus stay
    // put. The submit button's disabled state still tracks the draft,
    // toggled directly so typing alone can enable submission.
    const submit = this.querySelector(`[data-comment-submit="${CSS.escape(key)}"]`);
    if (submit instanceof HTMLButtonElement)
      submit.disabled = area.value.trim() === '' || this.view.commentBusy.includes(key);
  };

  private draftFor(key: string): string {
    const cached = this.drafts.get(key);
    if (cached !== undefined) return cached;
    const stored = readDraft(this.storageKey(key)).slice(0, 4000);
    this.drafts.set(key, stored);
    return stored;
  }

  private backToLatest = () => {
    if (!this.current || this.view.busy) return;
    this.view = {...this.view, next: '', truncated: false, pagedBack: false};
    void this.readLatest(false).then(() => {
      this.querySelector('.soda-feed-list')?.scrollIntoView();
    });
  };

  private entryKeyFor(entry: FeedEntry): string {
    return `${entry.kind}:${entry.id}`;
  }

  private projectCommentKey(): string {
    return 'project:' + (this.current?.repository ?? '');
  }

  private storageKey(key: string) {
    return `${this.current?.account ?? this.current?.transport.generation ?? 'session'}:${key}`;
  }

  private entryForKey(key: string): FeedEntry | undefined {
    return this.view.entries.find((entry) => this.entryKeyFor(entry) === key);
  }

  private draftIdempotencyKey(key: string): string {
    // The draft key survives failed submits so a retry recovers the original
    // server-side; a confirmed submit rotates it for the next fresh reply.
    try {
      const stored = window.localStorage.getItem(`soda-feed-draftkey:v2:${this.storageKey(key)}`);
      if (stored && /^[A-Za-z0-9_-]{1,64}$/.test(stored)) return stored;
      const minted = newIdempotencyKey();
      window.localStorage.setItem(`soda-feed-draftkey:v2:${this.storageKey(key)}`, minted);
      return minted;
    } catch {
      return newIdempotencyKey();
    }
  }

  private rotateDraftKey(key: string): void {
    try {
      window.localStorage.removeItem(`soda-feed-draftkey:v2:${this.storageKey(key)}`);
    } catch {
      /* A kept key only risks recovering the confirmed comment. */
    }
  }

  private upsertOutcome(key: string, state: CommentPipelineState): boolean {
    const known = this.view.outcomes.get(key) ?? [];
    const next = [state, ...known.filter((row) => row.commentId !== state.commentId)].slice(0, 5);
    const outcomes = new Map(this.view.outcomes);
    outcomes.set(key, next);
    this.view = {...this.view, outcomes};
    return commentUnsettled(state);
  }

  // submitComment persists one draft through the Lane B intake and exposes
  // its assessment/draft/effect progress on the same outcome. The draft's
  // idempotency key is reused, so resubmitting recovers the original.
  // A failed submit keeps the draft and says so; nothing is lost.
  private async submitComment(key: string) {
    const context = this.current;
    const entry = this.entryForKey(key);
    if (!context || this.view.commentBusy.includes(key)) return;
    const repository = entry?.repository ?? (key === this.projectCommentKey() ? Number(context.repository) : 0);
    if (!Number.isSafeInteger(repository) || repository <= 0) return;
    const body = this.draftFor(key);
    // Replies bind their parent post; top-level replies omit it. The entry
    // id carries a comment only for post entries, so records never invent
    // a reply parent. Every submission also names the feed entry it
    // answers (source kind + identity): the server resolves that entry
    // inside the caller's visible repository and persists its snapshot,
    // so intake and JEV routing consume authorized recorded evidence.
    // Milestone posts name their bound assignment. The bound native
    // issue still travels as the optional scope.
    const replyTo = entry?.kind === 'post' && /^[a-f0-9]{32}$/.test(entry.id) ? entry.id : undefined;
    const sourceIssue = entry?.issue !== undefined && entry.issue > 0 ? entry.issue : undefined;
    const source = entry ? commentSourceForEntry(entry) : undefined;
    let post: CommentPostBody;
    try {
      post = commentPostBody({
        repository,
        body,
        idempotencyKey: this.draftIdempotencyKey(key),
        ...(replyTo === undefined ? {} : {replyTo}),
        ...(sourceIssue === undefined ? {} : {issue: sourceIssue}),
        ...(source === undefined ? {} : {source}),
      });
    } catch {
      this.view = {...this.view, message: 'Reply needs its text before submitting.'};
      return;
    }
    const pendingKey = `soda-feed-pending:v1:${this.storageKey(key)}`;
    try {
      const saved = this.pendingPosts.get(pendingKey) ?? JSON.parse(localStorage.getItem(pendingKey) ?? 'null');
      if (
        saved &&
        saved.repository === repository &&
        typeof saved.body === 'string' &&
        saved.idempotency_key === post.idempotency_key
      )
        post = saved;
      this.pendingPosts.set(pendingKey, post);
      localStorage.setItem(pendingKey, JSON.stringify(post));
    } catch {
      this.pendingPosts.set(pendingKey, post);
    }
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
      const state = parseCommentPipelineState(await readSodaJSON(response));
      if (serial !== this.serial) return;
      const records = [{commentId: state.commentId, key: post.idempotency_key}, ...readIntake(this.storageKey(key))];
      writeIntake(this.storageKey(key), records);
      this.rotateDraftKey(key);
      this.pendingPosts.delete(pendingKey);
      try {
        localStorage.removeItem(pendingKey);
      } catch {
        /* Memory state remains current. */
      }
      if (this.draftFor(key) === post.body) {
        this.drafts.set(key, '');
        writeDraft(this.storageKey(key), '');
      }
      this.upsertOutcome(key, state);
      const busy = this.view.commentBusy.filter((row) => row !== key);
      this.view = {
        ...this.view,
        commentBusy: busy,
        message: this.draftFor(key)
          ? 'Previous comment saved. Your newer edits are kept, ready to send.'
          : 'Comment saved.',
      };
      if (commentUnsettled(state)) this.pollOutcome(key, state.commentId, serial);
      await this.readLatest(false);
      if (entry && this.threads.has(key)) void this.loadReplies(entry);
    } catch (error) {
      if (serial === this.serial) {
        this.view = {
          ...this.view,
          commentBusy: this.view.commentBusy.filter((row) => row !== key),
          message: error instanceof Error ? error.message : 'Could not save the reply. The draft is kept.',
        };
      }
    }
  }

  // checkCommentProgress re-reads one stored pipeline row. Reads serve
  // stored state only: they never judge, draft or create. Returns true
  // once the row is settled (or unreadable), false while it advances.
  // Quiet polls skip the feed-wide message and the latest re-read.
  private async checkCommentProgress(key: string, commentId: string, quiet = false): Promise<boolean> {
    const context = this.current;
    if (!context || this.view.commentBusy.includes(commentId)) return false;
    const serial = this.serial;
    this.view = {...this.view, commentBusy: [...this.view.commentBusy, commentId]};
    try {
      const response = await context.transport.request(`factory/feed/comments/${commentId}`, {
        method: 'GET',
        signal: AbortSignal.timeout(15000),
      });
      if (!response.ok) {
        if (response.status === 404) throw Error('That comment is no longer visible. The record is kept.');
        throw Error(failureMessage(response.status, 'Could not read the reply outcome.'));
      }
      const state = parseCommentPipelineState(await readSodaJSON(response));
      if (serial !== this.serial) return true;
      const settled = !this.upsertOutcome(key, state);
      this.view = {...this.view, commentBusy: this.view.commentBusy.filter((row) => row !== commentId)};
      if (settled && state.issue > 0 && !quiet) await this.readLatest(true);
      return settled;
    } catch (error) {
      if (serial === this.serial) {
        this.view = {
          ...this.view,
          commentBusy: this.view.commentBusy.filter((row) => row !== commentId),
          ...(quiet ? {} : {message: error instanceof Error ? error.message : 'Could not read the reply outcome.'}),
        };
      }
      return true;
    }
  }

  // Bounded durable-continuation polling: while a submitted comment's
  // pipeline is unsettled, re-read its stored state a few times so the
  // server-side continuation result appears without resubmission.
  // Reads never drive intake; the poll stops on settlement, budget
  // exhaustion, disconnect or context change.
  private outcomePolls = new Set<string>();

  private pollOutcome(key: string, commentId: string, serial: number): void {
    const pollKey = `${key}:${commentId}`;
    if (this.outcomePolls.has(pollKey)) return;
    this.outcomePolls.add(pollKey);
    let attempts = 0;
    const tick = () => {
      if (serial !== this.serial || !this.current) {
        this.outcomePolls.delete(pollKey);
        return;
      }
      attempts += 1;
      void this.checkCommentProgress(key, commentId, true).then((settled) => {
        if (settled || attempts >= 10 || serial !== this.serial) {
          this.outcomePolls.delete(pollKey);
          return;
        }
        window.setTimeout(tick, 4000);
      });
    };
    window.setTimeout(tick, 4000);
  }

  // refreshOutcomes re-serves stored pipeline rows for the rendered posts,
  // so reloads and refreshes recover saved/assessing/drafting progress
  // without re-running intake. Known rows that are still pending are
  // refreshed too — durable continuation may have advanced them
  // server-side since they were first rendered. Only settled known
  // rows are kept from cache. Failures keep the last outcome.
  private async refreshOutcomes(serial: number) {
    const context = this.current;
    if (!context) return;
    for (const key of [...this.view.entries.map((entry) => this.entryKeyFor(entry)), this.projectCommentKey()]) {
      const records = readIntake(this.storageKey(key));
      if (records.length === 0) continue;
      const known = new Map((this.view.outcomes.get(key) ?? []).map((row) => [row.commentId, row]));
      let changed = false;
      for (const record of records) {
        const previous = known.get(record.commentId);
        if (previous !== undefined && !commentUnsettled(previous)) continue;
        try {
          const response = await context.transport.request(`factory/feed/comments/${record.commentId}`, {
            method: 'GET',
            signal: AbortSignal.timeout(15000),
          });
          if (!response.ok) continue;
          const state = parseCommentPipelineState(await readSodaJSON(response));
          if (serial !== this.serial) return;
          known.set(state.commentId, state);
          changed = true;
        } catch {
          /* One unreadable outcome never blocks the rest. */
        }
      }
      if (changed && serial === this.serial) {
        const outcomes = new Map(this.view.outcomes);
        outcomes.set(key, [...known.values()].slice(0, 5));
        this.view = {...this.view, outcomes};
        for (const row of known.values()) if (commentUnsettled(row)) this.pollOutcome(key, row.commentId, serial);
      }
    }
  }

  // refreshPostBodies resolves the stored text behind rendered post
  // entries. Records carry no comment, so they never issue this read.
  // An unreadable payload keeps its tombstone instead of an invented body.
  private async refreshPostBodies(serial: number) {
    const context = this.current;
    if (!context) return;
    for (const entry of this.view.entries) {
      if (entry.kind !== 'post' || entry.payload?.communication || !/^[a-f0-9]{32}$/.test(entry.id)) continue;
      try {
        const response = await context.transport.request(`factory/feed/comments/${entry.id}`, {
          method: 'GET',
          signal: AbortSignal.timeout(15000),
        });
        if (serial !== this.serial) return;
        const bodies = new Map(this.view.bodies);
        if (!response.ok) {
          bodies.set(entry.id, {
            state: 'unavailable',
            reason:
              response.status === 404
                ? 'That discussion post is no longer visible. Its pipeline record is kept where stored.'
                : 'Discussion text unavailable; the post keeps its place.',
          });
          this.view = {...this.view, bodies};
          continue;
        }
        const detail = parseCommentDetail(await readSodaJSON(response));
        if (serial !== this.serial) return;
        bodies.set(
          entry.id,
          detail.removed
            ? {
                state: 'unavailable',
                reason: 'That discussion post was removed. Its pipeline record is kept where stored.',
              }
            : {state: 'text', body: detail.body, edited: detail.edited}
        );
        this.view = {...this.view, bodies};
        if (detail.pipeline) this.upsertOutcome(this.entryKeyFor(entry), detail.pipeline);
      } catch {
        if (serial !== this.serial) return;
        const bodies = new Map(this.view.bodies);
        bodies.set(entry.id, {state: 'unavailable', reason: 'Discussion text unavailable; the post keeps its place.'});
        this.view = {...this.view, bodies};
      }
    }
  }

  // openDrawer pulls one post's comment drawer out and reads its thread on
  // first open; typing a draft or reopening never invokes intake.
  private openDrawer(key: string) {
    this.openDrawers.add(key);
    const entry = this.entryForKey(key);
    if (entry && !this.threads.has(key)) void this.loadReplies(entry);
    this.requestUpdate();
  }

  private onFeedClick = (event: Event) => {
    const target = event.target;
    if (!(target instanceof HTMLElement)) return;
    const drawerToggle = target.closest<HTMLElement>('[data-feed-drawer-toggle]');
    if (drawerToggle?.dataset.feedDrawerToggle) {
      const key = drawerToggle.dataset.feedDrawerToggle;
      if (this.openDrawers.has(key)) {
        this.openDrawers.delete(key);
        this.requestUpdate();
      } else this.openDrawer(key);
      return;
    }
    const library = target.closest<HTMLElement>('[data-library-item]');
    if (library?.dataset.libraryItem) {
      const entry = this.entryForKey(library.dataset.libraryEntry ?? '');
      if (entry) { this.library = {repository: String(entry.repository), item: library.dataset.libraryItem}; this.requestUpdate(); }
      return;
    }
    const revise = target.closest<HTMLElement>('[data-comment-revise]');
    if (revise?.dataset.commentRevise && this.current) {
      const serial = this.serial;
      void readJSON(this.current.transport, `factory/feed/comments/${revise.dataset.commentRevise}`)
        .then((value) => {
          if (serial !== this.serial) return;
          const data = object(value);
          if (data.can_retry !== true) return;
          this.revising = {id: String(data.comment_id), revision: Number(data.revision), body: String(data.body)};
          if (data.route === 'clarify')
            this.revising = {
              ...this.revising,
              original: String(data.body),
              body: '',
              question:
                typeof data.detail === 'string' ? data.detail : 'What should change, and where should it apply?',
            };
          this.requestUpdate();
          void this.updateComplete.then(() => this.querySelector<HTMLElement>('[data-revision-body]')?.focus());
        })
        .catch(() => {
          this.view = {...this.view, message: 'Could not load this request. Refresh and try again.'};
        });
      return;
    }
    if (target.closest('[data-revision-cancel]')) {
      this.revising = undefined;
      this.requestUpdate();
      return;
    }
    if (target.closest('[data-revision-submit]')) {
      void this.reviseComment();
      return;
    }
    const answer = target.closest('[data-feed-answer]');
    if (answer instanceof HTMLButtonElement && answer.dataset.feedAnswer) {
      const key = answer.dataset.feedAnswer, text = answer.dataset.feedAnswerText ?? '';
      const area = this.querySelector<HTMLTextAreaElement>(`[data-feed-draft="${CSS.escape(key)}"]`);
      if (area) {
        // Selecting a choice edits the existing draft; only Submit invokes JEV.
        area.value = area.value.trim() ? `${area.value}\n${text}`.slice(0, 4000) : text.slice(0, 4000);
        this.openDrawer(key);
        area.dispatchEvent(new Event('input', {bubbles: true}));
        void this.updateComplete.then(() =>
          this.querySelector<HTMLTextAreaElement>(`[data-feed-draft="${CSS.escape(key)}"]`)?.focus()
        );
      }
      return;
    }
    const submit = target.closest('[data-comment-submit]');
    if (submit instanceof HTMLElement && submit.dataset.commentSubmit) {
      void this.submitComment(submit.dataset.commentSubmit);
      return;
    }
    const retry = target.closest('[data-comment-retry]');
    if (retry instanceof HTMLElement && retry.dataset.commentRetry) {
      const outcome = retry.dataset.commentRetry;
      const key = [...this.view.outcomes].find(([, rows]) => rows.some((row) => row.commentId === outcome))?.[0];
      if (key) void this.checkCommentProgress(key, outcome);
    }
  };

  private async reviseComment() {
    const edit = this.revising,
      context = this.current,
      serial = this.serial;
    if (!edit || !context || this.view.busy || !edit.body.trim()) return;
    const body = edit.original === undefined ? edit.body : edit.original + '\n\nClarification: ' + edit.body;
    if ([...body].length > 4000) {
      this.view = {...this.view, message: 'Keep the original request and answer within 4,000 characters. You can shorten the original request below.'};
      return;
    }
    this.view = {...this.view, busy: true};
    try {
      const response = await context.transport.request(`factory/feed/comments/${edit.id}/retry`, {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify({
          expected_revision: edit.revision,
          body,
        }),
        signal: AbortSignal.timeout(15000),
      });
      const value = await readSodaJSON(response);
      if (!response.ok)
        throw Error((value as {error?: {message?: string}})?.error?.message ?? 'Could not update this request.');
      if (serial !== this.serial) return;
      this.revising = undefined;
      this.view = {...this.view, busy: false, message: 'Request updated. Processing will continue here.'};
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

  protected render() {
    const confirmed = activityTimestamp(this.view.confirmedUnix);
    if (this.library) return html`<section class="soda-feed" aria-label="Project library"><header class="soda-feed-toolbar"><h2>Project library</h2><button type="button" @click=${() => { this.library = undefined; this.requestUpdate(); }}>← Back to activity</button></header><soda-factory-library .repository=${this.library.repository} .itemID=${this.library.item} .request=${this.current?.transport.request}></soda-factory-library></section>`;
    return html`<section class="soda-feed" aria-label="Feed" @click=${this.onFeedClick} @input=${this.onDraft}>
      <header class="soda-feed-toolbar">
        <h2>${this.current?.global ? 'Following' : 'Activity'}</h2>
        <span class="soda-feed-toolbar-actions">
          ${!this.current?.global && this.current?.repository ? html`<button type="button" class="soda-feed-utility soda-feed-icon-button" aria-label="Knowledge & decisions" title="Knowledge & decisions" @click=${() => { if (this.current) this.library = {repository: this.current.repository}; this.requestUpdate(); }}><svg viewBox="0 0 24 24" aria-hidden="true" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"><path d="M12 6c-1.8-1.5-4.2-2-8-2v14c3.8 0 6.2.5 8 2 1.8-1.5 4.2-2 8-2V4c-3.8 0-6.2.5-8 2Zm0 0v14"/></svg></button>` : ''}
          <button type="button" class="ui button soda-feed-utility soda-feed-icon-button" aria-label="Refresh feed" title="Refresh feed" ?disabled=${this.view.busy} @click=${() => void this.readLatest(true)}><svg viewBox="0 0 24 24" aria-hidden="true" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"><path d="M20 12a8 8 0 1 1-2.34-5.66M20 3v4h-4"/></svg></button>
        </span>
      </header>
      <p data-feed-message aria-live="polite">${this.view.message}</p>
      ${
        this.revising
          ? html`<section class="soda-feed-revision" aria-label="Revise request">
              ${this.revising.original !== undefined ? html`<details><summary>Original request · edit if needed</summary><label>Original request<textarea rows="4" maxlength="4000" .value=${this.revising.original} @input=${(event: Event) => {
                const target = event.target;
                if (target instanceof HTMLTextAreaElement && this.revising) this.revising.original = target.value;
              }}></textarea></label></details>` : ''}
              <label
                >${this.revising.question ?? 'Update your request'}<textarea
                  data-revision-body
                  rows="5"
                  maxlength="4000"
                  .value=${this.revising.body}
                ></textarea>
              </label>
              <p>Submitting reassesses this request and may start work using the project’s existing allowance.</p>
              <button type="button" data-revision-submit ?disabled=${this.view.busy}>
                ${this.revising.question ? 'Submit answer' : 'Submit updated request'}</button
              ><button type="button" data-revision-cancel>Cancel</button>
            </section>`
          : ''
      }
      ${confirmed ? html`<p class="soda-feed-updated">Updated <time datetime=${confirmed.iso}>${confirmed.text}</time></p>` : ''}
      ${
        this.view.pagedBack
          ? html`<button type="button" class="ui button" ?disabled=${this.view.busy} @click=${this.backToLatest}>
              Back to latest
            </button>`
          : ''
      }
      ${this.current?.global ? this.renderSubscription() : this.renderProjectComment()} ${this.renderList()}
    </section>`;
  }

  private renderProjectComment() {
    const context = this.current;
    if (!context || !id(context.repository)) return html``;
    const key = this.projectCommentKey();
    const repository = Number(context.repository);
    const ref = this.view.labels.refs.get(repository);
    const outcomes = (this.view.outcomes.get(key) ?? []).map((state) =>
      describeCommentOutcome(state, {id: repository, path: ref?.path, label: ref?.label ?? 'This project'})
    );
    return renderDiscussion({
      key,
      draft: this.draftFor(key),
      label: 'Project comment',
      submitLabel: 'Post comment',
      submitAvailable: this.draftFor(key).trim() !== '' && !this.view.commentBusy.includes(key),
      outcomes,
      discussionLink: undefined,
      intakeText:
        'Ask a question or request a change. JEV connects your comment to existing work or asks the Product Lead to create new work.',
    });
  }

  private renderSubscription() {
    const subscription = this.view.subscription;
    if (!subscription || !this.current?.global) return html``;
    const candidates = [...this.view.labels.refs.entries()].sort((a, b) => a[0] - b[0]);
    const member = new Set(subscription.repositories);
    return html`<form @submit=${this.save}>
      <fieldset>
        <legend>Followed projects</legend>
        <input type="hidden" name="expected_revision" value=${String(subscription.revision)} />
        ${
          subscription.revision === 0
            ? html`<p>No subscription saved yet: follow at least one project to read the combined feed.</p>`
            : ''
        }
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
    </form>`;
  }

  private renderList() {
    if (this.view.entries.length === 0)
      return html`<p class="soda-feed-empty">${this.view.busy ? 'Loading activity…' : 'No activity yet.'}</p>`;
    return html`<div class="soda-feed-list">
        ${repeat(
          this.view.entries,
          (entry) => `${entry.kind}:${entry.id}`,
          (entry) => {
            const ref = this.view.labels.refs.get(entry.repository) ?? {
              path: undefined,
              label: `repository ${entry.repository}`,
            };
            const repository = {id: entry.repository, path: ref.path, label: ref.label};
            const key = `${entry.kind}:${entry.id}`;
            const outcomes: CommentOutcomeModel[] = (this.view.outcomes.get(key) ?? []).map((state) =>
              describeCommentOutcome(state, repository)
            );
            const body: PostBodyState | undefined =
              entry.kind === 'post' ? (this.view.bodies.get(entry.id) ?? {state: 'loading'}) : undefined;
            const post = describeFeedPost(
              entry,
              repository,
              this.draftFor(key),
              outcomes,
              body,
              false,
              describeComments(key, this.openDrawers.has(key), this.threads.get(key))
            );
            return renderFeedPost(
              {
                ...post,
                discussion: {
                  ...post.discussion,
                  submitAvailable: post.discussion.submitAvailable && !this.view.commentBusy.includes(key),
                },
              },
              this.renderReplies(entry),
              this.current?.transport.request,
              this.current?.transport.assetBase
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
      }
      ${this.view.truncated ? html`<p>Older entries exist beyond this page; the cursor resumes where this page ends.</p>` : ''}`;
  }

  private async loadReplies(entry: FeedEntry, older = false) {
    const context = this.current,
      source = commentSourceForEntry(entry),
      serial = this.serial;
    if (!context || !source) return;
    const key = this.entryKeyFor(entry),
      previous = this.threads.get(key);
    if (previous?.busy) return;
    this.threads.set(key, {
      ...previous,
      items: previous?.items ?? [],
      next: previous?.next ?? 0,
      busy: true,
      message: '',
    });
    this.requestUpdate();
    try {
      const query = new URLSearchParams({
        repository: String(entry.repository),
        source_kind: source.kind,
        source_id: source.id,
      });
      if (older && previous?.next) query.set('before', String(previous.next));
      const result = object(await readJSON(context.transport, 'factory/feed/replies?' + query));
      check(Array.isArray(result.items));
      const items = result.items.map(parseCommentDetail);
      check(
        typeof result.next_before === 'number' && Number.isSafeInteger(result.next_before) && result.next_before >= 0
      );
      if (serial !== this.serial) return;
      this.threads.set(key, {
        items: older ? [...(previous?.items ?? []), ...items] : items,
        next: result.next_before,
        busy: false,
        message: '',
      });
    } catch {
      if (serial !== this.serial) return;
      this.threads.set(key, {
        items: previous?.items ?? [],
        next: previous?.next ?? 0,
        busy: false,
        message: 'Replies could not be loaded. Try Refresh replies.',
      });
    }
    this.requestUpdate();
  }

  private renderReplies(entry: FeedEntry) {
    if (!commentSourceForEntry(entry)) return html``;
    const key = this.entryKeyFor(entry),
      thread = this.threads.get(key);
    const ref = this.view.labels.refs.get(entry.repository);
    if (!thread) return html`<p role="status" class="soda-feed-thread-status">Loading replies…</p>`;
    return html`<div class="soda-feed-thread">
      <p class="soda-feed-thread-head">
        <span role="status">${thread.busy ? 'Loading replies…' : thread.message || (thread.items.length ? '' : 'No replies yet.')}</span>
        <button type="button" class="soda-feed-icon-button" aria-label="Refresh replies" title="Refresh replies" ?disabled=${thread.busy} @click=${() => void this.loadReplies(entry)}><svg viewBox="0 0 24 24" aria-hidden="true" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"><path d="M20 12a8 8 0 1 1-2.34-5.66M20 3v4h-4"/></svg></button>
      </p>
      <ul class="soda-feed-comments">
        ${thread.items.map(
          (reply) => html`<li class="soda-feed-comment">
            <span class="soda-feed-avatar" aria-hidden="true">${(reply.authorName || '?').slice(0, 1).toUpperCase()}</span>
            <div class="soda-feed-comment-main">
              <p class="soda-feed-comment-who">
                <strong>${reply.authorName || 'Comment'}</strong>
                <time datetime=${new Date(reply.at * 1000).toISOString()} title=${new Date(reply.at * 1000).toLocaleString()}>· ${relativeReplyTime(reply.at)}</time>
                ${reply.edited ? html` <span class="soda-feed-comment-edited">(edited)</span>` : ''}
              </p>
              <p class="soda-feed-body-text">${reply.body}</p>
              ${reply.pipeline ? renderDiscussion({
                key: 'post:' + reply.commentId,
                draft: '',
                submitAvailable: false,
                showComposer: false,
                intakeText: '',
                discussionLink: undefined,
                outcomes: [
                  describeCommentOutcome(reply.pipeline, {
                    id: entry.repository,
                    path: ref?.path,
                    label: ref?.label ?? 'This project',
                  }),
                ],
              }) : ''}
            </div>
          </li>`
        )}
      </ul>
      ${thread.next ? html`<button type="button" class="soda-feed-more" ?disabled=${thread.busy} @click=${() => void this.loadReplies(entry, true)}>Show more</button>` : ''}
    </div>`;
  }
}

customElements.define('soda-factory-feed', SodaFactoryFeed);
