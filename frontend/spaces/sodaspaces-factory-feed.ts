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
  CommentPostBody,
  FeedEntry,
  FeedEntryKind,
  FeedSubscriptionView,
} from './sodaspaces-factory-feed-response.js';
import {describeCommentOutcome, describeFeedPost, renderDiscussion, renderFeedPost} from './sodaspaces-factory-feed-view.js';
import type {CommentOutcomeModel, PostBodyState} from './sodaspaces-factory-feed-view.js';

export interface FactoryFeedContext {
  transport: PreparedExtensionMount;
  repository: string;
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
  private pendingScroll: number | undefined;

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
    if (context?.transport === this.current?.transport && context?.repository === this.current?.repository) return;
    this.current = context;
    this.serial++;
    this.drafts.clear();
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
    if (context) void this.reload();
  }

  disconnectedCallback() {
    super.disconnectedCallback();
    this.serial++;
    this.current = undefined;
  }

  private scopeRepositories(): number[] {
    const subscription = this.view.subscription;
    if (subscription && subscription.repositories.length > 0) return subscription.repositories;
    const current = Number(this.current?.repository);
    return Number.isSafeInteger(current) && current > 0 ? [current] : [];
  }

  private async reload() {
    const context = this.current;
    if (!context || this.view.busy) return;
    if (!id(context.repository)) {
      this.view = {...this.view, message: 'The combined feed needs its current project binding.'};
      return;
    }
    const serial = this.serial;
    this.view = {...this.view, busy: true};
    try {
      const subscription = parseFeedSubscriptionView(await readJSON(context.transport, 'factory/feed/subscription'));
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
    if (Number.isSafeInteger(current) && current > 0) wanted.add(current);
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
    if (!context) return;
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
    if (reset) this.pendingScroll = window.scrollY;
    try {
      const page = parseFeedPageView(
        await readJSON(
          context.transport,
          feedQueryString({repositories: scope, kinds: this.view.filterKinds, limit: feedPageSize, cursor: ''})
        )
      );
      if (serial !== this.serial) return;
      const fresh = page.entries.filter(
        (entry) => !this.view.entries.some((known) => known.kind === entry.kind && known.id === entry.id)
      );
      this.view = {
        ...this.view,
        busy: false,
        entries: reset ? page.entries : mergeFeedEntries(this.view.entries, page.entries),
        next: page.next,
        truncated: page.truncated,
        pagedBack: false,
        confirmedUnix: Math.floor(Date.now() / 1000),
        message:
          reset && !fresh.length && this.view.entries.length > 0 ? 'Feed refreshed; no new posts.' : this.view.message,
      };
      if (reset && fresh.length > 0 && this.view.entries.length > fresh.length)
        this.view = {
          ...this.view,
          message: `Feed refreshed; ${fresh.length} new post${fresh.length === 1 ? '' : 's'} at the top. Reading position preserved.`,
        };
      if (this.pendingScroll !== undefined) {
        const top = this.pendingScroll;
        this.pendingScroll = undefined;
        await this.updateComplete;
        if (serial === this.serial) window.scrollTo({top});
      }
      // Stored outcomes and post text re-serve without re-running intake.
      void this.refreshOutcomes(serial);
      void this.refreshPostBodies(serial);
    } catch (error) {
      if (serial === this.serial) {
        // A failed refresh keeps the last confirmed feed and says so; stale
        // data is never presented as current or cleared into an empty result.
        this.pendingScroll = undefined;
        this.view = {
          ...this.view,
          busy: false,
          message: error instanceof Error ? error.message : 'The combined feed is unavailable.',
        };
      }
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
          })
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
    const kinds = feedEntryKinds.filter((kind) => data.has(`kind-${kind}`));
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
        this.view = {...this.view, message: 'Feed subscription saved.'};
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
    const key = area.dataset.feedDraft;
    if (!key) return;
    this.drafts.set(key, area.value.slice(0, 4000));
    writeDraft(key, area.value.slice(0, 4000));
    // Draft keystrokes never re-render: reading position and focus stay
    // put. The submit button's disabled state still tracks the draft,
    // toggled directly so typing alone can enable submission.
    const submit = this.querySelector(`[data-comment-submit="${CSS.escape(key)}"]`);
    if (submit instanceof HTMLButtonElement) submit.disabled = area.value.trim() === '' || this.view.commentBusy.includes(key);
  };

  private draftFor(key: string): string {
    const cached = this.drafts.get(key);
    if (cached !== undefined) return cached;
    const stored = readDraft(key).slice(0, 4000);
    this.drafts.set(key, stored);
    return stored;
  }

  private backToLatest = () => {
    if (!this.current || this.view.busy) return;
    this.view = {...this.view, entries: [], next: '', truncated: false, pagedBack: false};
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

  private entryForKey(key: string): FeedEntry | undefined {
    return this.view.entries.find((entry) => this.entryKeyFor(entry) === key);
  }

  private draftIdempotencyKey(key: string): string {
    // The draft key survives failed submits so a retry recovers the original
    // server-side; a confirmed submit rotates it for the next fresh reply.
    try {
      const stored = window.localStorage.getItem(`soda-feed-draftkey:v1:${key}`);
      if (stored && /^[A-Za-z0-9_-]{1,64}$/.test(stored)) return stored;
      const minted = newIdempotencyKey();
      window.localStorage.setItem(`soda-feed-draftkey:v1:${key}`, minted);
      return minted;
    } catch {
      return newIdempotencyKey();
    }
  }

  private rotateDraftKey(key: string): void {
    try {
      window.localStorage.removeItem(`soda-feed-draftkey:v1:${key}`);
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
    const serial = this.serial;
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
      const records = [{commentId: state.commentId, key: post.idempotency_key}, ...readIntake(key)];
      writeIntake(key, records);
      this.rotateDraftKey(key);
      this.drafts.set(key, '');
      writeDraft(key, '');
      this.upsertOutcome(key, state);
      const busy = this.view.commentBusy.filter((row) => row !== key);
      this.view = {
        ...this.view,
        commentBusy: busy,
        message: 'Reply saved. Assessment, authoring and creation progress appear beside it.',
      };
      if (commentUnsettled(state)) this.pollOutcome(key, state.commentId, serial);
      await this.readLatest(false);
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
      const records = readIntake(key);
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
      if (entry.kind !== 'post' || !/^[a-f0-9]{32}$/.test(entry.id)) continue;
      if (this.view.bodies.has(entry.id)) continue;
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
        this.upsertOutcome(this.entryKeyFor(entry), detail.pipeline);
      } catch {
        if (serial !== this.serial) return;
        const bodies = new Map(this.view.bodies);
        bodies.set(entry.id, {state: 'unavailable', reason: 'Discussion text unavailable; the post keeps its place.'});
        this.view = {...this.view, bodies};
      }
    }
  }

  private onFeedClick = (event: Event) => {
    const target = event.target;
    if (!(target instanceof HTMLElement)) return;
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

  protected render() {
    const confirmed = activityTimestamp(this.view.confirmedUnix);
    return html`<section aria-label="Combined factory feed" @click=${this.onFeedClick} @input=${this.onDraft}>
      <h3>Combined feed</h3>
      <p data-feed-message aria-live="polite">${this.view.message}</p>
      ${this.view.subscription || confirmed ? html`<p>${confirmed ? html`Last confirmed <time datetime=${confirmed.iso}>${confirmed.text}</time>. ` : ''}Newest posts first; subscription filters inside already permitted activity and never grants access.</p>` : ''}
      <button type="button" class="ui button" ?disabled=${this.view.busy} @click=${() => void this.readLatest(true)}>
        Refresh feed
      </button>
      ${
        this.view.pagedBack
          ? html`<button type="button" class="ui button" ?disabled=${this.view.busy} @click=${this.backToLatest}>
              Back to latest
            </button>`
          : ''
      }
      ${this.renderProjectComment()} ${this.renderSubscription()} ${this.renderList()}
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
      key, draft: this.draftFor(key),
      label: 'Project comment', submitLabel: 'Post comment',
      submitAvailable: this.draftFor(key).trim() !== '' && !this.view.commentBusy.includes(key),
      outcomes, discussionLink: undefined,
      intakeText: 'Ask a question or request a change. JEV connects your comment to existing work or asks the Product Lead to create new work.',
    });
  }

  private renderSubscription() {
    const subscription = this.view.subscription;
    if (!subscription) return html``;
    const candidates = [...this.view.labels.refs.entries()].sort((a, b) => a[0] - b[0]);
    const member = new Set(subscription.repositories);
    return html`<form @submit=${this.save}>
      <fieldset>
        <legend>Followed projects</legend>
        <input type="hidden" name="expected_revision" value=${String(subscription.revision)} />
        ${
          subscription.revision === 0
            ? html`<p>No subscription saved yet: follow at least one project to read the combined feed.</p>`
            : html`<p>Revision ${subscription.revision}. A stale save refuses instead of dropping another update.</p>`
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
        ${this.view.labels.listedAll ? '' : html`<p>Project names list the first page only; followed projects outside it keep their repository number.</p>`}
      </fieldset>
      <fieldset>
        <legend>Entry kinds</legend>
        ${feedEntryKinds.map(
          (kind) => html`<label
            ><input
              type="checkbox"
              name="kind-${kind}"
              .checked=${subscription.kinds.includes(kind)}
              ?disabled=${this.view.busy}
            />${kind === 'record' ? 'Records' : 'Discussion posts'}</label
          >`
        )}
        <p>Unchecking both kinds reads everything.</p>
      </fieldset>
      <button class="ui button" type="submit" ?disabled=${this.view.busy}>Save feed subscription</button>
    </form>`;
  }

  private renderList() {
    if (this.view.entries.length === 0)
      return html`<p>${this.view.subscription ? 'No feed entries in scope yet.' : ''}</p>`;
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
            const post = describeFeedPost(entry, repository, this.draftFor(key), outcomes, body, false);
            return renderFeedPost(post);
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
}

customElements.define('soda-factory-feed', SodaFactoryFeed);
