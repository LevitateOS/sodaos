import {LitElement, html} from 'lit';
import {repeat} from 'lit/directives/repeat.js';
import type {PreparedExtensionMount} from './soda-extension.js';
import {check, id, object, readSodaJSON} from './sodaspaces-api.js';
import {activityTimestamp, nativePath} from './sodaspaces-factory-activity.js';
import {repositoryChoices} from './sodaspaces-repository-response.js';
import {applicationsResponse} from './sodaspaces-application-response.js';
import {repositoryPlaces} from './soda-places.js';
import type {SodaPlaces} from './soda-places.js';
import type {FeedCardLinks} from './sodaspaces-factory-feed-communication.js';
import {
  commentPostBody,
  commentSourceForEntry,
  commentTombstoneOf,
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
import {renderFeedCommunication} from './sodaspaces-factory-feed-communication.js';
import {renderConsultation, renderContribution} from './sodaspaces-factory-consultation.js';
import {roleLabel} from './sodaspaces-factory-feed-families.js';
import {feedRolePortraits} from './sodaspaces-factory-feed-portraits.js';

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

const feedPageSize = 20;

// Feed-wide banner set when a change review finds the lane detail
// unavailable. A later Check progress recovery clears it only while it is
// still this same message for the same comment; any newer message stays.
const unavailableChangeMessage = 'The proposed change detail is currently unavailable. Check progress to reload it.';

function failureMessage(status: number | undefined, fallback: string): string {
  if (status === 403)
    return 'One or more subscribed projects are not visible to you. The feed shows only permitted activity.';
  return fallback;
}

// cachedChangeRefreshDue reports whether a cached thread row carrying a
// product change still advances: review_required, draft and accepted
// changes can progress server-side (reviewed, revised, adopted), so a
// closed drawer must re-read them on refresh. An unavailable change is
// recoverable, not terminal: the lane detail missed once and can arrive
// on a later read, so it re-reads with the same staleness/ordering
// guards as other refreshable states. Only superseded changes are
// terminal and never re-read (removed/tombstoned rows retire through the
// direct-GET paths).
function cachedChangeRefreshDue(state: CommentPipelineState): boolean {
  const change = state.productChange;
  return (
    change !== undefined &&
    (change.state === 'review_required' ||
      change.state === 'draft' ||
      change.state === 'accepted' ||
      change.state === 'unavailable')
  );
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

// Unsent clarification answers and revision bodies persist per
// account/repository/comment/revision. The stored payload keeps the answer
// body plus the edited original request (clarify only). Reads restore without
// invoking intake; only a confirmed submission or an explicit discard clears
// the stored text. A stale revision never auto-submits: reopening against a
// newer source revision surfaces the saved text for review. Every payload
// carries an ownership lineage token: a fresh panel-open mints one, while a
// panel that restores a stale draft inherits that draft's token, so later
// persists stay on the same line. Submit retires an older-revision key of
// its own lineage only while it still holds the exact restored-then-
// superseded snapshot (plus same-text matches) and preserves foreign
// lineages as well as later same-lineage writes.
// Persists are compare-and-swap against the panel's last-persisted text:
// a foreign write under the same key is preserved, the local text stays in
// panel memory, and an explicit conflict is surfaced — never overwritten.
interface RevisionDraft {
  body: string;
  original?: string;
  savedAt?: number;
  lineage?: string;
  ancestors?: RevisionAncestorSnapshot[];
}

// One superseded ancestor this panel's line already replaced in-panel:
// the exact stored payload (with its source revision) that a stale restore
// superseded. Persisted alongside body/lineage so the chain survives
// reloads and revision advances; submit retires each only while unchanged.
interface RevisionAncestorSnapshot {
  revision: number;
  body: string;
  original?: string;
  lineage?: string;
}

function revisionLineageOf(candidate: unknown): string | undefined {
  if (typeof candidate !== 'object' || candidate === null) return undefined;
  const lineage = (candidate as {lineage?: unknown}).lineage;
  return typeof lineage === 'string' && lineage.length > 0 ? lineage.slice(0, 128) : undefined;
}

function revisionAncestorsOf(candidate: unknown): RevisionAncestorSnapshot[] | undefined {
  if (typeof candidate !== 'object' || candidate === null) return undefined;
  const ancestors = (candidate as {ancestors?: unknown}).ancestors;
  if (ancestors === undefined) return undefined;
  if (!Array.isArray(ancestors)) return undefined;
  const snapshots: RevisionAncestorSnapshot[] = [];
  for (const row of ancestors.slice(0, 20)) {
    if (typeof row !== 'object' || row === null) continue;
    const {revision, body, original, lineage} = row as {
      revision?: unknown;
      body?: unknown;
      original?: unknown;
      lineage?: unknown;
    };
    if (typeof revision !== 'number' || !Number.isSafeInteger(revision) || revision <= 0) continue;
    if (typeof body !== 'string') continue;
    if (original !== undefined && typeof original !== 'string') continue;
    if (lineage !== undefined && typeof lineage !== 'string') continue;
    snapshots.push({
      revision,
      body: body.slice(0, 4000),
      ...(original !== undefined ? {original: original.slice(0, 4000)} : {}),
      ...(typeof lineage === 'string' && lineage.length > 0 ? {lineage: lineage.slice(0, 128)} : {}),
    });
  }
  return snapshots;
}

function revisionDraftKey(account: string, repository: string, commentId: string, revision: number): string {
  return `soda-feed-revision-draft:v1:${account}:${repository}:${commentId}:${revision}`;
}

function readRevisionDraft(
  account: string,
  repository: string,
  commentId: string,
  revision: number
): RevisionDraft | undefined {
  try {
    const raw = window.localStorage.getItem(revisionDraftKey(account, repository, commentId, revision));
    if (!raw) return undefined;
    const parsed: unknown = JSON.parse(raw);
    if (typeof parsed !== 'object' || parsed === null) return undefined;
    const {body, original, savedAt} = parsed as {body?: unknown; original?: unknown; savedAt?: unknown};
    if (typeof body !== 'string') return undefined;
    if (original !== undefined && typeof original !== 'string') return undefined;
    if (savedAt !== undefined && typeof savedAt !== 'number') return undefined;
    const lineage = revisionLineageOf(parsed);
    const ancestors = revisionAncestorsOf(parsed);
    return {
      body: body.slice(0, 4000),
      ...(original !== undefined ? {original: original.slice(0, 4000)} : {}),
      ...(typeof savedAt === 'number' ? {savedAt} : {}),
      ...(lineage !== undefined ? {lineage} : {}),
      ...(ancestors !== undefined && ancestors.length > 0 ? {ancestors} : {}),
    };
  } catch {
    return undefined;
  }
}

function writeRevisionDraft(
  account: string,
  repository: string,
  commentId: string,
  revision: number,
  draft: RevisionDraft
): void {
  try {
    if (draft.body === '' && (draft.original === undefined || draft.original === ''))
      window.localStorage.removeItem(revisionDraftKey(account, repository, commentId, revision));
    else
      window.localStorage.setItem(
        revisionDraftKey(account, repository, commentId, revision),
        JSON.stringify({
          body: draft.body.slice(0, 4000),
          ...(draft.original !== undefined ? {original: draft.original.slice(0, 4000)} : {}),
          ...(draft.lineage !== undefined ? {lineage: draft.lineage.slice(0, 128)} : {}),
          ...(draft.ancestors !== undefined && draft.ancestors.length > 0
            ? {
                ancestors: draft.ancestors.slice(0, 20).map((row) => ({
                  revision: row.revision,
                  body: row.body.slice(0, 4000),
                  ...(row.original !== undefined ? {original: row.original.slice(0, 4000)} : {}),
                  ...(row.lineage !== undefined ? {lineage: row.lineage.slice(0, 128)} : {}),
                })),
              }
            : {}),
          revision,
          savedAt: typeof draft.savedAt === 'number' ? draft.savedAt : Date.now(),
        })
      );
  } catch {
    /* Revision drafts stay in memory when browser storage is unavailable. */
  }
}

// retireSubmittedRevisionDrafts retires the submitting panel's own older
// line: each same-lineage key below the submitted revision retires only
// while it still holds the exact ancestor snapshot this panel's line
// restored and superseded (v5=A restored into v6 and revised into B must
// not resurrect after B submits — even across a reload, since the chain
// now persists in every saved draft, and across v5→v6→v7 advances, since
// each restore appends to the chain). A later write under the same
// lineage — the still-open v5 tab typing newer C into key5 after the
// restore — differs from its snapshot and is preserved. Same-text matches
// retire as ancestors. Keys holding different text under a foreign
// lineage — a divergent draft saved by another tab — are preserved;
// findStaleRevisionDraft newest-wins already prefers the newer draft, so a
// preserved older one only resurfaces for review, never as a silent
// overwrite. A draft against the adopted revision is retired only while it
// still holds the submitted text; drafts against newer sources stay
// untouched. Unreadable payloads are never silently deleted.
function retireSubmittedRevisionDrafts(
  account: string,
  repository: string,
  commentId: string,
  submittedBody: string,
  submittedOriginal: string | undefined,
  adoptedRevision: number,
  submittedRevision: number,
  submittingLineage: string | undefined,
  ancestors: RevisionAncestorSnapshot[]
): void {
  try {
    const prefix = `soda-feed-revision-draft:v1:${account}:${repository}:${commentId}:`;
    const wantedBody = submittedBody.slice(0, 4000);
    const wantedOriginal = submittedOriginal?.slice(0, 4000) ?? undefined;
    const byRevision = new Map<number, RevisionAncestorSnapshot>();
    for (const snapshot of ancestors) {
      if (snapshot.revision >= submittedRevision) continue;
      if (!byRevision.has(snapshot.revision)) byRevision.set(snapshot.revision, snapshot);
    }
    const keys: string[] = [];
    for (let index = 0; index < window.localStorage.length; index++) {
      const key = window.localStorage.key(index);
      if (key?.startsWith(prefix)) keys.push(key);
    }
    for (const key of keys) {
      const suffix = key.slice(prefix.length);
      if (!/^\d+$/.test(suffix)) continue;
      if (Number(suffix) > adoptedRevision) continue;
      let raw: string | null = null;
      try {
        raw = window.localStorage.getItem(key);
      } catch {
        continue;
      }
      if (!raw) continue;
      let parsed: unknown;
      try {
        parsed = JSON.parse(raw);
      } catch {
        continue;
      }
      if (typeof parsed !== 'object' || parsed === null) continue;
      const {body, original} = parsed as {body?: unknown; original?: unknown};
      if (typeof body !== 'string') continue;
      if (body === wantedBody && (original ?? undefined) === wantedOriginal) {
        window.localStorage.removeItem(key);
        continue;
      }
      const keyedRevision = Number(suffix);
      const snapshot = byRevision.get(keyedRevision);
      if (
        submittingLineage !== undefined &&
        snapshot !== undefined &&
        (snapshot.lineage === undefined || snapshot.lineage === submittingLineage) &&
        revisionLineageOf(parsed) === submittingLineage &&
        body === snapshot.body.slice(0, 4000) &&
        (original ?? undefined) === (snapshot.original?.slice(0, 4000) ?? undefined)
      )
        window.localStorage.removeItem(key);
    }
  } catch {
    /* Memory state remains current. */
  }
}

function revisionNumberOf(candidate: unknown): number | undefined {
  if (typeof candidate !== 'object' || candidate === null) return undefined;
  const revision = (candidate as {revision?: unknown}).revision;
  return typeof revision === 'number' && Number.isSafeInteger(revision) && revision > 0 ? revision : undefined;
}

// writeRevisionDraftUnlessDivergent stores the draft only when no
// different text already exists under the key: an absent or text-equal
// entry is written, while a divergent draft saved by another tab stays
// intact and the caller keeps its own text in the live panel.
function writeRevisionDraftUnlessDivergent(
  account: string,
  repository: string,
  commentId: string,
  revision: number,
  draft: RevisionDraft
): void {
  const body = draft.body.slice(0, 4000);
  const original = draft.original?.slice(0, 4000) ?? undefined;
  const existing = readRevisionDraft(account, repository, commentId, revision);
  if (existing !== undefined && (existing.body !== body || (existing.original ?? undefined) !== original)) return;
  writeRevisionDraft(account, repository, commentId, revision, draft);
}

function clearRevisionDraftsForComment(account: string, repository: string, commentId: string): void {
  try {
    const prefix = `soda-feed-revision-draft:v1:${account}:${repository}:${commentId}:`;
    const doomed: string[] = [];
    for (let index = 0; index < window.localStorage.length; index++) {
      const key = window.localStorage.key(index);
      if (key?.startsWith(prefix)) doomed.push(key);
    }
    for (const key of doomed) window.localStorage.removeItem(key);
  } catch {
    /* Memory state remains current. */
  }
}

function findStaleRevisionDraft(
  account: string,
  repository: string,
  commentId: string,
  revision: number
): {revision: number; draft: RevisionDraft} | undefined {
  try {
    // Every candidate carries its explicit source revision (key suffix,
    // confirmed by the stored payload) and save time; the newest applicable
    // draft wins regardless of storage enumeration order.
    const prefix = `soda-feed-revision-draft:v1:${account}:${repository}:${commentId}:`;
    const candidates: {revision: number; savedAt: number; draft: RevisionDraft}[] = [];
    for (let index = 0; index < window.localStorage.length; index++) {
      const key = window.localStorage.key(index);
      if (!key?.startsWith(prefix)) continue;
      const suffix = key.slice(prefix.length);
      if (!/^\d+$/.test(suffix) || Number(suffix) === revision) continue;
      const raw = window.localStorage.getItem(key);
      if (!raw) continue;
      const parsed: unknown = JSON.parse(raw);
      if (typeof parsed !== 'object' || parsed === null) continue;
      const {
        body,
        original,
        revision: storedRevision,
        savedAt,
      } = parsed as {
        body?: unknown;
        original?: unknown;
        revision?: unknown;
        savedAt?: unknown;
      };
      if (typeof body !== 'string') continue;
      if (original !== undefined && typeof original !== 'string') continue;
      if (body === '' && (original === undefined || original === '')) continue;
      const keyedRevision = Number(suffix);
      // Stored payloads written without an explicit revision predate the
      // versioned format; the key suffix still identifies their source.
      const explicitRevision =
        storedRevision === undefined ? keyedRevision : typeof storedRevision === 'number' ? storedRevision : NaN;
      if (!Number.isSafeInteger(explicitRevision) || explicitRevision !== keyedRevision) continue;
      if (savedAt !== undefined && typeof savedAt !== 'number') continue;
      const lineage = revisionLineageOf(parsed);
      const ancestors = revisionAncestorsOf(parsed);
      candidates.push({
        revision: keyedRevision,
        savedAt: typeof savedAt === 'number' ? savedAt : 0,
        draft: {
          body: body.slice(0, 4000),
          ...(original !== undefined ? {original: original.slice(0, 4000)} : {}),
          ...(typeof savedAt === 'number' ? {savedAt} : {}),
          ...(lineage !== undefined ? {lineage} : {}),
          ...(ancestors !== undefined && ancestors.length > 0 ? {ancestors} : {}),
        },
      });
    }
    candidates.sort((a, b) => b.revision - a.revision || b.savedAt - a.savedAt);
    const newest = candidates[0];
    return newest ? {revision: newest.revision, draft: newest.draft} : undefined;
  } catch {
    return undefined;
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

// A retired parent outcome drops its intake record too: reloads and
// refreshes re-serve stored records, so a kept record would resurrect the
// duplicate the thread already owns. The thread row stays the only surface.
function removeIntakeRecord(key: string, commentId: string): void {
  const records = readIntake(key);
  const kept = records.filter((row) => row.commentId !== commentId);
  if (kept.length !== records.length) writeIntake(key, kept);
}

async function readJSON(transport: PreparedExtensionMount, path: string, init?: RequestInit): Promise<unknown> {
  const response = await transport.request(path, {method: 'GET', signal: AbortSignal.timeout(15000), ...init});
  if (!response.ok) {
    const error = Error(failureMessage(response.status, 'The combined feed is unavailable.'));
    (error as {status?: number}).status = response.status;
    throw error;
  }
  // A conversation page contains at most 20 admitted tool records, each with
  // bounded original input and result. Other API responses keep the 64 KiB cap.
  const conversation = path.startsWith('factory/feed/replies?') || /^factory\/feed\/comments\/[a-f0-9]{32}$/.test(path);
  return readSodaJSON(response, conversation ? 4 * 1024 * 1024 : 65536);
}

// One cursor-anchored thread window: the ids one served page delivered, in
// canonical ascending order, keyed by the cursor that resumes below it.
// Windows are pairwise disjoint and oldest-first, and the flat thread items
// always read as their concatenation, so an older page splices below the
// window whose cursor fetched it — even across a same-second burst gap —
// instead of prepending in front of older kept windows or relying on
// timestamp order to interleave it.
interface ThreadPage {
  ids: string[];
  nextBefore: number;
}

// sweepThreadPage drops cached replies a fresh page proves gone. Deleted and
// revoked replies are ABSENT from served pages (the reply listing excludes
// them), never tombstoned: a kept reply inside the fresh span but omitted
// from it is gone and drops out instead of rejoining after its predecessor.
// Replies outside the span stay put: newer ones stay covered by newer
// windows, and older ones stay covered by their own window's cursor — a page
// never covers outside its span, so a truncated page never deletes cached
// entries and an older page never purges the windows below it.
function sweepThreadPage(flat: CommentDetail[], fresh: CommentDetail[]): CommentDetail[] {
  if (fresh.length === 0) return flat;
  const freshIds = new Set(fresh.map((item) => item.commentId));
  const firstFresh = flat.findIndex((item) => item.commentId === fresh[0]?.commentId);
  const lastFresh = flat.findIndex((item) => item.commentId === fresh[fresh.length - 1]?.commentId);
  if (firstFresh < 0 || lastFresh < 0) return flat;
  return flat.filter((item, position) => {
    if (freshIds.has(item.commentId)) return true;
    if (position < firstFresh || position > lastFresh) return true;
    return false;
  });
}

// coveredOlderOmissions drops cached replies an authoritative older page
// proves gone, using cursor/page coverage only — never timestamps and no
// invented seq. An older page fetched with cursor C enumerates the live
// thread below C newest-first: a cached row the page omits between two rows
// it serves is gone (deleted/revoked), as is a cached row below the served
// span once the page completes (next_before 0). Rows at/above the window
// that produced C sit outside the page's coverage and stay, as do rows
// below the served span while the page stays truncated — that tail is
// genuinely unserved. A disjoint page shares nothing cached, so it deletes
// only on a complete page with a matching anchor: the completed enumeration
// spans everything below the anchor window, superseding the older windows'
// earlier bottom coverage, so their omitted rows purge. Real pages carry
// LIMIT-sized spans, so a short complete page with a matching anchor
// genuinely exhausts what sits below it.
function coveredOlderOmissions(
  kept: CommentDetail[],
  keptPages: ThreadPage[],
  freshIds: Set<string>,
  sentBefore: number,
  complete: boolean
): Set<string> {
  const drop = new Set<string>();
  const slot = keptPages.findIndex((page) => page.nextBefore === sentBefore);
  // Rows at/above the anchor window sit outside this page's coverage. When
  // the cursor no longer matches a window (a stale race), only the newest
  // window is provably above it.
  const shield = new Set(
    (slot >= 0 ? keptPages.slice(slot) : keptPages.slice(-1)).flatMap((page) => page.ids)
  );
  const positions = new Map(kept.map((item, index) => [item.commentId, index] as const));
  let first = -1;
  let last = -1;
  for (const item of kept) {
    if (!freshIds.has(item.commentId)) continue;
    const position = positions.get(item.commentId) ?? -1;
    if (first < 0 || position < first) first = position;
    if (position > last) last = position;
  }
  if (first < 0) {
    // Disjoint page: without a shared row only a complete page with a
    // matching anchor proves coverage — the completed enumeration spans
    // everything below the anchor window, so omitted cached rows in older
    // windows are gone (deleted/revoked). Truncated pages and anchorless
    // races keep every kept row: a truncated page leaves the genuinely
    // unserved tail alone, and without a matching anchor the upper boundary
    // is unproven.
    if (!complete || slot < 0) return drop;
    const covered = new Set(keptPages.slice(0, slot).flatMap((page) => page.ids));
    for (const item of kept) {
      if (freshIds.has(item.commentId) || shield.has(item.commentId)) continue;
      if (covered.has(item.commentId)) drop.add(item.commentId);
    }
    return drop;
  }
  for (const item of kept) {
    if (freshIds.has(item.commentId) || shield.has(item.commentId)) continue;
    const position = positions.get(item.commentId) ?? -1;
    if (position > last) {
      // Newer than every shared row yet below the anchor: the page serves
      // that span top-down, so an omission is gone. Without a matching
      // anchor the upper boundary is unproven — keep.
      if (slot >= 0) drop.add(item.commentId);
    } else if (position < first) {
      // Below the served span: gone only once the bottom completes.
      if (complete) drop.add(item.commentId);
    } else {
      // Inside the served span but omitted: gone.
      drop.add(item.commentId);
    }
  }
  return drop;
}

// spliceThreadPage folds one overlapping page over the kept span it covers.
// The fresh page wins on content, so a changed payload (for example a
// returned consultation) never shows stale cached text; genuinely newer kept
// replies stay above it. The windows rebuild around the fresh span so a
// later older page still anchors to the cursor that resumes below it. The
// returned cursor continues below the merged window: the oldest touched
// window's own cursor when it still extends below the fresh span, otherwise
// the fresh page's cursor.
function spliceThreadPage(
  kept: CommentDetail[],
  keptPages: ThreadPage[],
  fresh: CommentDetail[],
  nextBefore: number
): {merged: CommentDetail[]; pages: ThreadPage[]; cursor: number} {
  const fallback = {
    merged: sweepThreadPage([...kept, ...fresh], fresh),
    pages: [...keptPages, {ids: fresh.map((item) => item.commentId), nextBefore}],
    cursor: nextBefore,
  };
  const freshIds = new Set(fresh.map((item) => item.commentId));
  if (!kept.some((item) => freshIds.has(item.commentId))) return fallback;
  const byId = new Map<string, CommentDetail>();
  for (const item of kept) byId.set(item.commentId, item);
  for (const item of fresh) byId.set(item.commentId, item);
  const keptIndex = new Map(kept.map((item, index) => [item.commentId, index] as const));
  let first = -1;
  let last = -1;
  for (let index = 0; index < fresh.length; index++) {
    const candidate = fresh[index];
    if (candidate !== undefined && keptIndex.has(candidate.commentId)) {
      if (first < 0) first = index;
      last = index;
    }
  }
  if (first < 0 || last < 0) return fallback;
  const firstId = fresh[first]?.commentId;
  const lastId = fresh[last]?.commentId;
  const prefix = firstId !== undefined ? (keptIndex.get(firstId) ?? 0) : 0;
  const suffix = lastId !== undefined ? (keptIndex.get(lastId) ?? kept.length - 1) + 1 : kept.length;
  const spliced: CommentDetail[] = [...kept.slice(0, prefix), ...fresh];
  const emitted = new Set(spliced.map((item) => item.commentId));
  for (const item of kept.slice(suffix)) {
    if (emitted.has(item.commentId)) continue;
    const row = byId.get(item.commentId);
    if (row === undefined) continue;
    spliced.push(row);
    emitted.add(item.commentId);
  }
  const merged = sweepThreadPage(spliced, fresh);
  const survivors = new Set(merged.map((item) => item.commentId));
  const positions = new Map(merged.map((item, index) => [item.commentId, index] as const));
  const firstFresh = positions.get(fresh[0]?.commentId ?? '') ?? 0;
  let firstTouched = -1;
  let lastTouched = -1;
  keptPages.forEach((page, index) => {
    if (page.ids.some((id) => freshIds.has(id))) {
      if (firstTouched < 0) firstTouched = index;
      lastTouched = index;
    }
  });
  if (firstTouched < 0 || lastTouched < 0) return {...fallback, merged};
  const prune = (page: ThreadPage): ThreadPage => ({
    ids: page.ids.filter((id) => survivors.has(id)),
    nextBefore: page.nextBefore,
  });
  const prefixPages = keptPages
    .slice(0, firstTouched)
    .map(prune)
    .filter((page) => page.ids.length > 0);
  const suffixPages = keptPages
    .slice(lastTouched + 1)
    .map(prune)
    .filter((page) => page.ids.length > 0);
  const prefixCount = prefixPages.reduce((sum, page) => sum + page.ids.length, 0);
  const suffixCount = suffixPages.reduce((sum, page) => sum + page.ids.length, 0);
  const mergedIds = merged.slice(prefixCount, merged.length - suffixCount).map((item) => item.commentId);
  const oldestTouched = keptPages[firstTouched];
  const extendsBelow =
    oldestTouched !== undefined &&
    oldestTouched.ids.some((id) => {
      const position = positions.get(id);
      return position !== undefined && position < firstFresh;
    });
  const cursor = extendsBelow && oldestTouched !== undefined ? oldestTouched.nextBefore : nextBefore;
  return {
    merged,
    pages: [...prefixPages, {ids: mergedIds, nextBefore: cursor}, ...suffixPages],
    cursor,
  };
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
  private revising:
    | {
        id: string;
        revision: number;
        repository: string;
        body: string;
        original?: string;
        question?: string;
        stale?: boolean;
        unverified?: boolean;
        conflict?: boolean;
        lineage: string;
        // Ancestor snapshot this panel restored from another revision and
        // superseded in-panel (stale v5 A revised into live v6 B). Submit
        // retires a same-lineage older key only while it still holds this
        // exact text; later writes under the same lineage are preserved.
        restoredFrom?: {revision: number; body: string; original?: string};
        // Full persisted ancestor chain this line already superseded
        // (older restores plus their chains). Every persist carries it so
        // a reload or a revision advance keeps every snapshot; submit
        // retires each only while its stored payload is unchanged.
        ancestors?: RevisionAncestorSnapshot[];
      }
    | undefined;
  // Last text this panel persisted per revision-draft key. Persists use it
  // as the compare-and-swap baseline: a stored value that no longer matches
  // is another tab's foreign write and is never overwritten.
  private revisionPersisted:
    | {account: string; repository: string; id: string; revision: number; body: string; original?: string}
    | undefined;
  // Snapshot of the revision retry POST in flight, if any. Keystrokes that
  // land while it is pending persist as newer unsent input and must survive
  // its success; a pending retry also blocks opening another revision.
  private pendingRevision:
    | {account: string; repository: string; id: string; revision: number; body: string; original?: string}
    | undefined;
  private threads = new Map<
    string,
    {items: CommentDetail[]; pages: ThreadPage[]; next: number; busy: boolean; message: string}
  >();
  private readingThreads = new Set<string>();
  // In-flight thread page reads by drawer key. A second caller awaits the
  // same promise instead of starting a duplicate GET or spinning on a
  // zero-ms timer while the read is pending.
  private threadReads = new Map<string, Promise<void>>();
  // Per-comment pipeline write order. A Refresh-replies page or a newer
  // review that lands while a Check progress GET is in flight owns the
  // thread row: the delayed response must never overwrite that newer
  // pipeline revision or permission state. Each writer stamps its claim;
  // a landing GET applies only while its stamp is still the newest. Every
  // applied pipeline write stamps again at apply time, so a slower older
  // response landing second drops instead of overwriting the just-applied
  // state — request-start claims alone cannot cover a read that started
  // first but lands second.
  private commentProgressOrder = new Map<string, number>();
  // Per-comment body write order. Body reads (refreshPostBodies,
  // reloadPostBody) stamp here — never the progress clock — so a newer
  // Check progress landing that updates only the pipeline cannot orphan
  // an in-flight body read in `loading`. A newer body read still
  // supersedes an older one, and removals stamp both clocks. Body
  // error/mismatch/catch landings apply only while their own stamp is
  // still newest, so a delayed failure never overwrites newer good text.
  // All stamps share one tick so removal marks stay comparable across
  // both surfaces.
  private commentBodyOrder = new Map<string, number>();
  private commentProgressTick = 0;
  // Authoritative removal order: every direct-GET tombstone (explicit
  // deleted/revoked flags) or access loss (403/404) stamps the shared
  // progress clock here. A replies page started before that stamp must not
  // reinsert the row when it lands late. Coverage-inferred purges never
  // record here — a missing page row alone proves nothing.
  private removalOrder = new Map<string, number>();
  // Retired-to-thread watch: intake records dropped because the thread owns
  // the row (commentId to outcome key). Retirement must not blind later
  // refreshes — watched ids keep their direct GET on every Refresh feed so
  // a later server advance still updates the cached thread row. Watched ids
  // never recreate parent outcomes; the thread owns the surface. Both maps
  // reset with the serial on context change.
  private threadWatched = new Map<string, string>();
  // Comment behind the feed-wide unavailable-change banner, if that banner
  // is still the current message. Recovery clears the banner only for this
  // same comment; an unrelated newer message is never cleared.
  private unavailableChangeComment: string | undefined;
  private roleReplyBusy = new Set<string>();
  // Newest role-reply selection queued while its own target is busy. The
  // busy guard is per target, so reselecting the same reply while its
  // lookup is in flight would otherwise be ignored and a stale reply
  // could win; the in-flight run drains this slot when it lands. The entry
  // carries the full scope token it was queued under (serial, query,
  // library, action, and context identity): a context switch bumps the
  // serial and clears the slot, and the drain revalidates every field, so
  // a queued selection can never replay — write or focus — in a new
  // project or account.
  private queuedRoleReply:
    | {
        link: HTMLElement;
        action: number;
        serial: number;
        query: number;
        library: number;
        transport: PreparedExtensionMount | undefined;
        repository: string | undefined;
        account: string | undefined;
        global: boolean | undefined;
      }
    | undefined;
  private openDrawers = new Set<string>();
  // The viewer's native user id (the feed subscription's subscriber): Open
  // request shows only on the viewer's own requests.
  private viewerID: string | undefined;
  // Verified delivery id → the origin of a Ready app the viewer may open.
  private appOrigins = new Map<string, string>();
  // The repository comment box stays one line until focused.
  private commentOpen = false;
  // A post to show alone (the Feed place's ?post=, e.g. a Knowledge item's source).
  private focusPost: string | undefined;
  // Navigation token for library source opens: each open mints a new token
  // so a superseded GET (Back navigation, newer open, context switch) can
  // never override the current view. librarySourcePending carries the source
  // id with a GET in flight for the library button pending state.
  private librarySourceSerial = 0;
  private librarySourcePending: string | undefined = undefined;
  private focusedEntry: FeedEntry | undefined;
  private filters = {view: 'all', search: '', role: '', current: false};
  private querySerial = 0;
  // Action generation: every user navigation (drawer toggle, library enter /
  // back, focused-entry back, filter change, another role-reply selection)
  // bumps it, so a superseded role-reply lookup can neither write status nor
  // steal focus after the user moves on. Filter changes already bump
  // querySerial; drawer/library/back moves bump this instead of disturbing
  // in-flight feed reads.
  private actionSerial = 0;
  private filteredViews = new Map<
    string,
    {entries: FeedEntry[]; next: string; truncated: boolean; pagedBack: boolean; scrollTop: number}
  >();

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
    this.view = {
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
    this.readingThreads.clear();
    this.threadReads.clear();
    this.openDrawers.clear();
    this.revising = undefined;
    this.librarySourceSerial++;
    this.librarySourcePending = undefined;
    this.viewerID = undefined;
    this.appOrigins = new Map();
    this.commentOpen = false;
    // A context change retires every in-flight role-reply selection: the
    // queue never replays in the new project or account, and the action
    // bump keeps even a surviving entry from validating.
    this.queuedRoleReply = undefined;
    this.roleReplyBusy.clear();
    this.actionSerial++;
    this.focusedEntry = undefined;
    this.filters = {view: 'all', search: '', role: '', current: false};
    this.filteredViews.clear();
    this.revisionPersisted = undefined;
    this.querySerial++;
    this.commentProgressOrder.clear();
    this.commentBodyOrder.clear();
    this.removalOrder.clear();
    this.threadWatched.clear();
    this.unavailableChangeComment = undefined;
    this.view = {
      ...this.view,
      busy: false,
      readFailed: false,
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
    this.librarySourceSerial++;
    this.librarySourcePending = undefined;
    this.queuedRoleReply = undefined;
    this.roleReplyBusy.clear();
    this.actionSerial++;
    this.current = undefined;
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
    if (!context || this.view.busy) return;
    if (!context.global && !id(context.repository)) {
      this.view = {...this.view, message: 'The combined feed needs its current project binding.'};
      return;
    }
    const serial = this.serial;
    this.view = {...this.view, busy: true, readFailed: false, message: this.view.readFailed ? '' : this.view.message};
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

  // The subscription's subscriber is the viewer's own native id; the
  // repository feed reads it only to know which requests are the viewer's.
  private async readViewerID(context: FactoryFeedContext) {
    try {
      const subscriber = parseFeedSubscriptionView(await readJSON(context.transport, 'factory/feed/subscription'))
        .subscriber;
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
    const querySerial = this.querySerial;
    const scope = this.scopeRepositories();
    if (scope.length === 0) {
      this.view = {
        ...this.view,
        busy: false,
        message: 'Choose at least one repository to follow before reading the Feed.',
      };
      return;
    }
    this.reading = true;
    this.view = {...this.view, busy: true, readFailed: false, message: this.view.readFailed ? '' : this.view.message};
    const anchor = Array.from(this.querySelectorAll<HTMLElement>('.soda-feed-post')).find(
      (el) => el.getBoundingClientRect().bottom > 0
    );
    const anchorTop = anchor?.getBoundingClientRect().top;
    try {
      const page = parseFeedPageView(
        await readJSON(
          context.transport,
          feedQueryString({
            repositories: scope,
            kinds: this.view.filterKinds,
            limit: feedPageSize,
            cursor: '',
            ...this.filters,
          }) + (context.global ? '' : '&scope=project')
        )
      );
      if (serial !== this.serial || querySerial !== this.querySerial) return;
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
        entries: mergeFeedEntries(
          this.view.entries.filter((entry) => page.truncated && entry.seq < (page.entries.at(-1)?.seq ?? 0)),
          page.entries
        ),
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
      for (const entry of this.focusedEntry ? [this.focusedEntry, ...this.view.entries] : this.view.entries) {
        if (this.openDrawers.has(this.entryKeyFor(entry))) void this.loadReplies(entry, false, true);
      }
    } catch (error) {
      if (serial === this.serial && querySerial === this.querySerial) {
        // A failed refresh keeps the last confirmed feed and says so; stale
        // data is never presented as current or cleared into an empty result.
        this.view = {
          ...this.view,
          busy: false,
          readFailed: true,
          message: error instanceof Error ? error.message : 'The combined feed is unavailable.',
        };
      }
    } finally {
      if (serial === this.serial && querySerial === this.querySerial) this.reading = false;
    }
  }

  private async readOlder() {
    const context = this.current;
    if (!context || this.view.busy || this.view.next === '') return;
    const serial = this.serial;
    const querySerial = this.querySerial;
    this.view = {...this.view, busy: true};
    try {
      const page = parseFeedPageView(
        await readJSON(
          context.transport,
          feedQueryString({
            repositories: this.scopeRepositories(),
            kinds: this.view.filterKinds,
            ...this.filters,
            limit: feedPageSize,
            cursor: this.view.next,
          }) + (context.global ? '' : '&scope=project')
        )
      );
      if (serial !== this.serial || querySerial !== this.querySerial) return;
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
      if (serial === this.serial && querySerial === this.querySerial)
        this.view = {
          ...this.view,
          busy: false,
          message: error instanceof Error ? error.message : 'Older feed entries are unavailable.',
        };
    }
  }

  private changeFilters(next: Partial<typeof this.filters>) {
    this.focusedEntry = undefined;
    const scroller = this.closest<HTMLElement>('.soda-project-content');
    this.filteredViews.set(JSON.stringify(this.filters), {
      entries: this.view.entries,
      next: this.view.next,
      truncated: this.view.truncated,
      pagedBack: this.view.pagedBack,
      scrollTop: scroller?.scrollTop ?? window.scrollY,
    });
    this.filters = {...this.filters, ...next};
    const cached = this.filteredViews.get(JSON.stringify(this.filters));
    this.querySerial++;
    this.actionSerial++;
    this.reading = false;
    this.view = {
      ...this.view,
      busy: true,
      readFailed: false,
      message: '',
      entries: cached?.entries ?? [],
      next: cached?.next ?? '',
      truncated: cached?.truncated ?? false,
      pagedBack: cached?.pagedBack ?? false,
    };
    const serial = this.querySerial;
    void this.updateComplete.then(() => {
      if (serial !== this.querySerial) return;
      if (scroller) scroller.scrollTop = cached?.scrollTop ?? 0;
      else window.scrollTo({top: cached?.scrollTop ?? 0});
      void this.readLatest(false);
    });
  }

  private renderFilters() {
    const views = [
      ['all', 'All'],
      ['attention', 'Needs me'],
      ['decisions', 'Choices & approvals'],
      ['evidence', 'Evidence'],
    ];
    const roles = [
      'human',
      'product_lead',
      'architect',
      'builder',
      'inspector',
      'fixer',
      'strategist',
      'test_pilot',
      'library_admin',
      'researcher',
      'jev',
      'factory',
    ];
    return html`<div class="soda-feed-filters">
      <nav aria-label="Filter conversation">
        ${views.map(
          ([value, label]) => html`<button
            type="button"
            aria-pressed=${this.filters.view === value ? 'true' : 'false'}
            @click=${() => this.changeFilters({view: value ?? 'all'})}
          >
            ${label}
          </button>`
        )}
      </nav>
      <form
        @submit=${(event: SubmitEvent) => {
          event.preventDefault();
          if (event.currentTarget instanceof HTMLFormElement)
            this.changeFilters({search: String(new FormData(event.currentTarget).get('search') ?? '').trim()});
        }}
      >
        <input
          type="search"
          name="search"
          aria-label="Search conversation"
          placeholder="Search posts"
          maxlength="256"
          .value=${this.filters.search}
        />
        <button type="submit">Search</button>
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

  private revisionAccount(): string {
    return this.current?.account ?? this.current?.transport.generation ?? 'session';
  }

  private persistRevising(): void {
    const edit = this.revising;
    if (!edit) return;
    const account = this.revisionAccount();
    const body = edit.body.slice(0, 4000);
    const original = edit.original?.slice(0, 4000) ?? undefined;
    // Compare-and-swap: overwrite only when the stored value still equals
    // this panel's last-persisted text (or the key is empty). A mismatch is
    // another tab's foreign write: preserve it, keep the local text in panel
    // memory, and surface an explicit conflict instead of overwriting.
    const tracked = this.revisionPersisted;
    const trackedHere =
      tracked !== undefined &&
      tracked.account === account &&
      tracked.repository === edit.repository &&
      tracked.id === edit.id &&
      tracked.revision === edit.revision;
    const stored = readRevisionDraft(account, edit.repository, edit.id, edit.revision);
    if (
      stored !== undefined &&
      !(
        trackedHere &&
        stored.body === tracked.body &&
        (stored.original ?? undefined) === (tracked.original ?? undefined)
      )
    ) {
      if (!edit.conflict) {
        this.revising = {...edit, conflict: true};
        this.view = {
          ...this.view,
          message:
            'Another tab saved a different draft for this revision. Your text is kept here and the saved draft was left untouched.',
        };
        this.requestUpdate();
      }
      return;
    }
    writeRevisionDraft(account, edit.repository, edit.id, edit.revision, {
      body,
      ...(original !== undefined ? {original} : {}),
      lineage: edit.lineage,
      ...(edit.ancestors !== undefined && edit.ancestors.length > 0 ? {ancestors: edit.ancestors} : {}),
    });
    this.revisionPersisted = {
      account,
      repository: edit.repository,
      id: edit.id,
      revision: edit.revision,
      body,
      ...(original !== undefined ? {original} : {}),
    };
    if (edit.conflict && this.revising) {
      const cleared = {...this.revising};
      delete cleared.conflict;
      this.revising = cleared;
      this.view = {...this.view, message: 'Draft conflict resolved; your text is now saved.'};
      this.requestUpdate();
    }
  }

  private discardRevising(): void {
    const edit = this.revising;
    if (edit) clearRevisionDraftsForComment(this.revisionAccount(), edit.repository, edit.id);
    this.revising = undefined;
    this.revisionPersisted = undefined;
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
    this.actionSerial++;
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
    const entry = [...this.view.entries, ...(this.focusedEntry ? [this.focusedEntry] : [])].find(
      (entry) => this.entryKeyFor(entry) === key
    );
    if (entry) return entry;
    for (const [root, thread] of this.threads) {
      const reply = thread.items.find((item) => 'post:' + item.commentId === key);
      const parent =
        this.view.entries.find((entry) => this.entryKeyFor(entry) === root) ??
        (this.focusedEntry && this.entryKeyFor(this.focusedEntry) === root ? this.focusedEntry : undefined);
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

  /** The post shown alone: the Feed place's ?post= (Knowledge's "Open source post"). */
  set post(post: string | undefined) {
    const valid = post && /^[a-f0-9]{32}$/.test(post) ? post : undefined;
    if (valid === this.focusPost) return;
    this.focusPost = valid;
    if (valid) void this.openSourcePost(valid);
    else if (this.focusedEntry) {
      this.librarySourceSerial++;
      this.focusedEntry = undefined;
      this.requestUpdate();
    }
  }

  private async openSourcePost(sourceId: string) {
    if (!this.current) return;
    const repository = this.current.repository;
    const serial = this.serial;
    const sourceToken = ++this.librarySourceSerial;
    const transport = this.current.transport;
    this.librarySourcePending = sourceId;
    this.requestUpdate();
    try {
      const value = object(await readJSON(transport, 'factory/feed/comments/' + sourceId));
      check(this.current?.global === true || String(value.repository) === repository);
      const source = parseCommentDetail(value);
      if (serial !== this.serial || sourceToken !== this.librarySourceSerial) return;
      const entry: FeedEntry = {
        kind: 'post',
        id: source.commentId,
        repository: Number(value.repository),
        seq: 0,
        at: source.at,
        author: source.authorName,
        ...(source.actor ? {actor: source.actor} : {}),
        ...(source.communication ? {payload: {communication: source.communication}} : {}),
      };
      this.view.bodies.set(source.commentId, {state: 'text', body: source.body, edited: source.edited});
      this.focusedEntry = entry;
      this.openDrawers.add('post:' + source.commentId);
      this.requestUpdate();
      void this.loadReplies(entry);
    } catch (error) {
      if (serial !== this.serial || sourceToken !== this.librarySourceSerial) return;
      this.view = {
        ...this.view,
        message: error instanceof Error ? error.message : 'Source conversation is unavailable.',
      };
    } finally {
      if (sourceToken === this.librarySourceSerial) {
        this.librarySourcePending = undefined;
        this.requestUpdate();
      }
    }
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

  // Newest-first claim on one comment's pipeline surface. Thread page
  // refreshes and reviews stamp the ids they apply; a landing Check
  // progress GET applies only while its own stamp is still newest.
  private stampCommentProgress(commentId: string): number {
    this.commentProgressTick += 1;
    this.commentProgressOrder.set(commentId, this.commentProgressTick);
    return this.commentProgressTick;
  }

  // stampCommentBody claims one comment's newest body read. Pipeline-only
  // writers never touch this clock, so their landings cannot orphan a
  // body read; body error/mismatch/catch landings apply only while their
  // own stamp is still newest.
  private stampCommentBody(commentId: string): number {
    this.commentProgressTick += 1;
    this.commentBodyOrder.set(commentId, this.commentProgressTick);
    return this.commentProgressTick;
  }

  // stampAuthoritativeRemoval records one direct-GET tombstone/access-loss
  // on the shared clock, in every order: later merges treat the removal
  // as the newest pipeline and body event, so a delayed live GET, a page
  // started before the stamp, or an older in-flight body read can never
  // resurrect the row.
  private stampAuthoritativeRemoval(commentId: string): void {
    this.commentProgressTick += 1;
    this.commentProgressOrder.set(commentId, this.commentProgressTick);
    this.commentBodyOrder.set(commentId, this.commentProgressTick);
    this.removalOrder.set(commentId, this.commentProgressTick);
    this.threadWatched.delete(commentId);
  }

  // A comment with a visible drawer reply row renders its pipeline from
  // the thread, never from parent outcomes.
  private threadHasComment(commentId: string): boolean {
    for (const thread of this.threads.values())
      if (thread.items.some((item) => item.commentId === commentId)) return true;
    return false;
  }

  private upsertOutcome(key: string, state: CommentPipelineState): boolean {
    const known = this.view.outcomes.get(key) ?? [];
    const next = [state, ...known.filter((row) => row.commentId !== state.commentId)].slice(0, 5);
    const outcomes = new Map(this.view.outcomes);
    outcomes.set(key, next);
    this.view = {...this.view, outcomes};
    return commentUnsettled(state);
  }

  private removeOutcome(key: string, commentId: string): void {
    const known = this.view.outcomes.get(key) ?? [];
    if (!known.some((row) => row.commentId === commentId)) return;
    const outcomes = new Map(this.view.outcomes);
    const next = known.filter((row) => row.commentId !== commentId);
    if (next.length === 0) outcomes.delete(key);
    else outcomes.set(key, next);
    this.view = {...this.view, outcomes};
  }

  // removeThreadComment drops one authoritative tombstone/access-loss id
  // from every cached thread, pruning its page windows. Explicit direct-GET
  // markers only — never a missing page row alone — so coverage evidence
  // elsewhere stays untouched.
  private removeThreadComment(commentId: string): void {
    for (const [key, thread] of this.threads) {
      if (!thread.items.some((item) => item.commentId === commentId)) continue;
      const items = thread.items.filter((item) => item.commentId !== commentId);
      const survivors = new Set(items.map((item) => item.commentId));
      const pages = thread.pages
        .map((page) => ({ids: page.ids.filter((id) => survivors.has(id)), nextBefore: page.nextBefore}))
        .filter((page) => page.ids.length > 0);
      this.threads.set(key, {...thread, items, pages});
    }
    this.requestUpdate();
  }

  // applyAuthoritativeRemoval retires one tombstoned/inaccessible comment
  // across every parent outcome and every cached thread: no stale Check
  // progress / Revise / Review controls, no resurrection through stored
  // intake, and the thread never re-renders the removed body. Fresh
  // Refresh feed / Refresh replies reads keep it absent; a fresh Check
  // progress re-GET re-confirms the same removal instead of reviving
  // pipeline state.
  private applyAuthoritativeRemoval(commentId: string, intakeKey?: string): void {
    // The removal owns the row from here on: stamp it so a delayed live
    // GET or an in-flight replies page landing later cannot resurrect it,
    // and drop any retired-to-thread watch for it.
    this.stampAuthoritativeRemoval(commentId);
    const keys = new Set(this.view.outcomes.keys());
    if (intakeKey !== undefined) keys.add(intakeKey);
    for (const key of keys) {
      if ((this.view.outcomes.get(key) ?? []).some((row) => row.commentId === commentId))
        this.removeOutcome(key, commentId);
      removeIntakeRecord(this.storageKey(key), commentId);
    }
    this.removeThreadComment(commentId);
    // A removed top-level post keeps its place with the removed tombstone:
    // without this, a removal that lands off the body paths (Check
    // progress, outcome refresh) leaves the deleted text visible after
    // retiring the controls.
    if (this.view.entries.some((entry) => entry.kind === 'post' && entry.id === commentId)) {
      const bodies = new Map(this.view.bodies);
      bodies.set(commentId, {
        state: 'unavailable',
        commentId,
        reason: 'That discussion post was removed. Its pipeline record is kept where stored.',
      });
      this.view = {...this.view, bodies};
    }
  }

  // refreshCachedThreadPipeline folds one non-superseded direct GET into
  // the cached thread row it names. Membership alone never proves
  // freshness: a closed drawer keeps its cache while the server advances,
  // so the fresh GET must update the cached row instead of dropping.
  private refreshCachedThreadPipeline(commentId: string, state: CommentPipelineState): void {
    // The applied row owns the pipeline now: a slower older read landing
    // later drops instead of overwriting it.
    this.stampCommentProgress(commentId);
    for (const [key, thread] of this.threads) {
      if (!thread.items.some((item) => item.commentId === commentId)) continue;
      this.threads.set(key, {
        ...thread,
        items: thread.items.map((item) =>
          item.commentId === commentId ? {...item, pipeline: state} : item
        ),
      });
    }
    this.requestUpdate();
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
  // An authoritative tombstone (explicit deleted/revoked flags) or access
  // loss (403/404) removes the requested comment across parent and thread:
  // no stale controls, no resurrection. A flag naming another comment is
  // not authoritative here. A bodyless payload without those flags and
  // a missing page row alone never count — only these direct-GET markers do.
  private async checkCommentProgress(key: string, commentId: string, quiet = false): Promise<boolean> {
    const context = this.current;
    if (!context || this.view.commentBusy.includes(commentId)) return false;
    const serial = this.serial;
    // The surface this check launched from: a nested reply updates its
    // thread row in place only, while top-level progress without a thread
    // row keeps the outcomes surface.
    const wasNested = this.threadHasComment(commentId);
    const order = this.stampCommentProgress(commentId);
    this.view = {...this.view, commentBusy: [...this.view.commentBusy, commentId]};
    try {
      const response = await context.transport.request(`factory/feed/comments/${commentId}`, {
        method: 'GET',
        signal: AbortSignal.timeout(15000),
      });
      if (!response.ok) {
        if (serial !== this.serial) return true;
        const supersededGone = this.commentProgressOrder.get(commentId) !== order;
        if ((response.status === 404 || response.status === 403) && !supersededGone) {
          this.applyAuthoritativeRemoval(commentId, key);
          this.view = {...this.view, commentBusy: this.view.commentBusy.filter((row) => row !== commentId)};
          return true;
        }
        if (response.status === 404) throw Error('That comment is no longer visible. The record is kept.');
        throw Error(failureMessage(response.status, 'Could not read the reply outcome.'));
      }
      const raw = await readSodaJSON(response);
      if (serial !== this.serial) return true;
      const supersededEarly = this.commentProgressOrder.get(commentId) !== order;
      const tombstone = commentTombstoneOf(raw);
      // Requested-ID binding: a terminal flag names only the requested
      // comment. A mismatched payload never purges or flags a different
      // comment; it reads as an unreadable row instead.
      if (tombstone !== undefined && tombstone.commentId !== commentId)
        throw Error('Could not read the reply outcome.');
      if (tombstone !== undefined) {
        if (!supersededEarly) {
          this.applyAuthoritativeRemoval(tombstone.commentId, key);
          this.view = {...this.view, commentBusy: this.view.commentBusy.filter((row) => row !== commentId)};
        } else {
          this.view = {...this.view, commentBusy: this.view.commentBusy.filter((row) => row !== commentId)};
        }
        return true;
      }
      const state = parseCommentPipelineState(raw);
      if (serial !== this.serial) return true;
      // Requested-ID binding for pipeline rows too: a state naming a
      // different comment never lands under this request's key or thread.
      if (state.commentId !== commentId) throw Error('Could not read the reply outcome.');
      const settled = !commentUnsettled(state);
      // A Refresh-replies page or a newer review that landed while this
      // GET was in flight owns the row now: the delayed response drops
      // without touching newer pipeline or permission state. The busy flag
      // still releases below so later checks stay available.
      const superseded = this.commentProgressOrder.get(commentId) !== order;
      const nested = this.threadHasComment(state.commentId);
      if (!superseded) {
        // A thread that authoritatively dropped the row while this GET was
        // in flight (delete/revoke) stays dropped: no resurrection into
        // parent outcomes.
        if (!wasNested) this.upsertOutcome(key, state);
        else if ((this.view.outcomes.get(key) ?? []).some((row) => row.commentId === state.commentId)) {
          // A nested recovery never creates a new parent-outcome duplicate,
          // but a row submitComment already stored must track the recovered
          // thread state instead of going stale: the thread row owns the
          // display while it carries the comment, so the redundant parent
          // row drops; when the thread no longer carries it, the stored
          // record stays as the only surface and refreshes in place. The
          // dropped intake record leaves a watch behind so later advances
          // still update the cached thread row on refresh.
          if (nested) {
            this.removeOutcome(key, state.commentId);
            removeIntakeRecord(this.storageKey(key), state.commentId);
            this.threadWatched.set(state.commentId, key);
          } else this.upsertOutcome(key, state);
        }
        // Drawer reply rows render from threads, never from outcomes: a
        // nested check refreshes the visible pipeline in place only, so the
        // parent drawer never shows the same outcome twice. A top-level
        // check keeps the original outcomes-plus-thread update.
        if (!wasNested || nested)
          for (const [threadKey, thread] of this.threads)
            if (thread.items.some((item) => item.commentId === state.commentId))
              this.threads.set(threadKey, {
                ...thread,
                items: thread.items.map((item) =>
                  item.commentId === state.commentId ? {...item, pipeline: state} : item
                ),
              });
        // A recovered change detail retires the feed-wide unavailable
        // banner, but only while it is still that same message for this
        // same comment. Any newer unrelated message stays untouched.
        if (
          state.productChange !== undefined &&
          state.productChange.state !== 'unavailable' &&
          this.unavailableChangeComment === state.commentId &&
          this.view.message === unavailableChangeMessage
        ) {
          this.view = {...this.view, message: ''};
          this.unavailableChangeComment = undefined;
        }
        // The applied state owns the pipeline now: a slower older read
        // (for example a body retry serving an older pipeline) landing
        // later drops instead of overwriting it.
        this.stampCommentProgress(commentId);
      }
      this.view = {...this.view, commentBusy: this.view.commentBusy.filter((row) => row !== commentId)};
      if (!superseded && settled && state.issue > 0 && !quiet) await this.readLatest(true);
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
  // A non-superseded GET updates the cached thread row it names: a closed
  // cached drawer keeps its cache while the server advances, so membership
  // alone never proves freshness. Per-record results merge against the
  // live cache at write time with a stamp recheck, so a newer
  // check/removal of an earlier comment landing during a delayed later
  // record never gets overwritten. Authoritative tombstones (explicit
  // deleted/revoked flags) and access loss (403/404) remove the comment
  // across parent and thread; a bodyless payload without those flags and a
  // missing page row alone never count. Retired-to-thread rows stay
  // watched: a later refresh re-reads them so subsequent advances still
  // surface on reopen.
  private async refreshOutcomes(serial: number) {
    const context = this.current;
    if (!context) return;
    for (const key of [...this.view.entries.map((entry) => this.entryKeyFor(entry)), this.projectCommentKey()]) {
      const records = readIntake(this.storageKey(key));
      if (records.length === 0) continue;
      const pending = new Map<string, {state: CommentPipelineState; order: number}>();
      for (const record of records) {
        const livePrevious = (this.view.outcomes.get(key) ?? []).find((row) => row.commentId === record.commentId);
        if (
          livePrevious !== undefined &&
          !livePrevious.productChange &&
          !commentUnsettled(livePrevious) &&
          livePrevious.issue <= 0
        )
          continue;
        // This refresh claims the row's newest read: a delayed Check
        // progress GET still in flight must drop instead of landing over
        // the fresher state this GET serves.
        const order = this.stampCommentProgress(record.commentId);
        try {
          const response = await context.transport.request(`factory/feed/comments/${record.commentId}`, {
            method: 'GET',
            signal: AbortSignal.timeout(15000),
          });
          if (serial !== this.serial) return;
          if (!response.ok) {
            const supersededGone = this.commentProgressOrder.get(record.commentId) !== order;
            if ((response.status === 404 || response.status === 403) && !supersededGone) {
              this.applyAuthoritativeRemoval(record.commentId, key);
            }
            continue;
          }
          const raw = await readSodaJSON(response);
          if (serial !== this.serial) return;
          const tombstone = commentTombstoneOf(raw);
          // Requested-ID binding: the terminal flag applies only to the
          // intake record just read. A mismatched payload never purges or
          // flags a different comment; the record stays for the next read.
          if (tombstone !== undefined && tombstone.commentId !== record.commentId) continue;
          if (tombstone !== undefined) {
            const supersededTomb = this.commentProgressOrder.get(tombstone.commentId) !== order;
            const nestedTomb = this.threadHasComment(tombstone.commentId);
            if (nestedTomb) {
              if ((this.view.outcomes.get(key) ?? []).some((row) => row.commentId === tombstone.commentId))
                this.removeOutcome(key, tombstone.commentId);
              removeIntakeRecord(this.storageKey(key), tombstone.commentId);
              if (!supersededTomb) {
                this.stampAuthoritativeRemoval(tombstone.commentId);
                this.removeThreadComment(tombstone.commentId);
              }
            } else if (!supersededTomb) {
              this.applyAuthoritativeRemoval(tombstone.commentId, key);
            }
            continue;
          }
          const state = parseCommentPipelineState(raw);
          if (serial !== this.serial) return;
          // Requested-ID binding for pipeline rows too: a state naming a
          // different comment never retires this record or lands under it.
          if (state.commentId !== record.commentId) continue;
          // A submitted nested comment lives in its thread surface only:
          // a refresh that finds the thread already rendering it retires
          // the redundant parent outcome instead of recreating it — even
          // when a thread page that landed mid-flight superseded this
          // GET's stamp, because that page owns the row now. A
          // non-superseded GET still updates the cached thread row: a
          // closed drawer keeps its cache while the server advances, so
          // reopening after Refresh must show the current state. The
          // retired intake record leaves a watch behind: retirement must
          // not blind later refreshes to this comment, so subsequent
          // advances still surface on refresh/reopen.
          if (this.threadHasComment(state.commentId)) {
            if ((this.view.outcomes.get(key) ?? []).some((row) => row.commentId === state.commentId))
              this.removeOutcome(key, state.commentId);
            removeIntakeRecord(this.storageKey(key), state.commentId);
            this.threadWatched.set(state.commentId, key);
            if (this.commentProgressOrder.get(state.commentId) === order)
              this.refreshCachedThreadPipeline(state.commentId, state);
            continue;
          }
          if (this.commentProgressOrder.get(state.commentId) !== order) continue;
          pending.set(state.commentId, {state, order});
        } catch {
          /* One unreadable outcome never blocks the rest. */
        }
      }
      if (pending.size > 0 && serial === this.serial) {
        const liveRows = new Map((this.view.outcomes.get(key) ?? []).map((row) => [row.commentId, row]));
        let changed = false;
        for (const [id, {state, order}] of pending) {
          // Recheck at write time: a newer check/removal of this comment
          // landing during a delayed later record owns the row now.
          if (this.commentProgressOrder.get(id) !== order) continue;
          if (this.threadHasComment(id)) {
            if (liveRows.has(id)) {
              liveRows.delete(id);
              this.threadWatched.set(id, key);
              removeIntakeRecord(this.storageKey(key), id);
              changed = true;
            }
            continue;
          }
          liveRows.set(id, state);
          // The merged row owns the pipeline now: a slower older read
          // landing later drops instead of overwriting it.
          this.stampCommentProgress(id);
          changed = true;
        }
        if (changed) {
          const outcomes = new Map(this.view.outcomes);
          outcomes.set(key, [...liveRows.values()].slice(0, 5));
          this.view = {...this.view, outcomes};
          for (const row of liveRows.values()) if (commentUnsettled(row)) this.pollOutcome(key, row.commentId, serial);
        }
      }
    }
    // Retired-to-thread rows keep their direct GET on every refresh: the
    // intake record is gone, but the watch is not, so a second server
    // advance still updates the cached thread row instead of staying stale
    // until a drawer reload. Watched rows never recreate parent outcomes;
    // when the thread no longer carries one, the watch drops and the next
    // drawer open re-pages it.
    if (this.threadWatched.size > 0 && serial === this.serial) {
      for (const [commentId, watchKey] of [...this.threadWatched]) {
        const order = this.stampCommentProgress(commentId);
        try {
          const response = await context.transport.request(`factory/feed/comments/${commentId}`, {
            method: 'GET',
            signal: AbortSignal.timeout(15000),
          });
          if (serial !== this.serial) return;
          if (!response.ok) {
            if (
              (response.status === 404 || response.status === 403) &&
              this.commentProgressOrder.get(commentId) === order
            ) {
              this.applyAuthoritativeRemoval(commentId, watchKey);
            }
            continue;
          }
          const raw = await readSodaJSON(response);
          if (serial !== this.serial) return;
          const tombstone = commentTombstoneOf(raw);
          // Requested-ID binding: the terminal flag applies only to the
          // watched id just read, never to a different comment it names.
          if (tombstone !== undefined && tombstone.commentId !== commentId) continue;
          if (tombstone !== undefined) {
            if (this.commentProgressOrder.get(tombstone.commentId) === order)
              this.applyAuthoritativeRemoval(tombstone.commentId, watchKey);
            continue;
          }
          const state = parseCommentPipelineState(raw);
          if (serial !== this.serial) return;
          // Requested-ID binding for pipeline rows too.
          if (state.commentId !== commentId) continue;
          if (this.commentProgressOrder.get(state.commentId) !== order) continue;
          if (this.threadHasComment(state.commentId)) {
            this.refreshCachedThreadPipeline(state.commentId, state);
            if (commentUnsettled(state)) this.pollOutcome(watchKey, state.commentId, serial);
          } else this.threadWatched.delete(state.commentId);
        } catch {
          /* One unreadable watched row never blocks the rest. */
        }
      }
    }
    // Reload survival: watches live only in memory, so an actual reload
    // drops them while retired intake records stay gone. A thread loaded
    // after the reload and then closed would otherwise reopen stale: its
    // rows have no intake record and no watch left to refresh them. Re-read
    // every cached thread row that still reads unsettled — plus change rows
    // that can still progress or recover (review_required, draft, accepted,
    // unavailable) — so the closed cache stays current. Settled rows and
    // the terminal superseded change state are skipped, watched rows were
    // covered above, and cached rows never recreate parent outcomes.
    if (serial === this.serial) {
      for (const thread of this.threads.values()) {
        for (const item of thread.items) {
          const cached = item.pipeline;
          if (!cached || !(commentUnsettled(cached) || cachedChangeRefreshDue(cached))) continue;
          const commentId = item.commentId;
          if (this.threadWatched.has(commentId)) continue;
          const order = this.stampCommentProgress(commentId);
          try {
            const response = await context.transport.request(`factory/feed/comments/${commentId}`, {
              method: 'GET',
              signal: AbortSignal.timeout(15000),
            });
            if (serial !== this.serial) return;
            if (!response.ok) {
              if (
                (response.status === 404 || response.status === 403) &&
                this.commentProgressOrder.get(commentId) === order
              ) {
                this.applyAuthoritativeRemoval(commentId);
              }
              continue;
            }
            const raw = await readSodaJSON(response);
            if (serial !== this.serial) return;
            const tombstone = commentTombstoneOf(raw);
            if (tombstone !== undefined && tombstone.commentId !== commentId) continue;
            if (tombstone !== undefined) {
              if (this.commentProgressOrder.get(tombstone.commentId) === order)
                this.applyAuthoritativeRemoval(tombstone.commentId);
              continue;
            }
            const state = parseCommentPipelineState(raw);
            if (serial !== this.serial) return;
            if (state.commentId !== commentId) continue;
            if (this.commentProgressOrder.get(state.commentId) !== order) continue;
            if (this.threadHasComment(state.commentId)) this.refreshCachedThreadPipeline(state.commentId, state);
          } catch {
            /* One unreadable cached row never blocks the rest. */
          }
        }
      }
    }
  }

  // refreshPostBodies resolves the stored text behind rendered post
  // entries. Records carry no comment, so they never issue this read.
  // An unreadable payload keeps its tombstone instead of an invented body.
  // A removed detail is terminal and never upserts pipeline progress; a
  // detail naming another comment is malformed for this entry.
  private async refreshPostBodies(serial: number) {
    const context = this.current;
    if (!context) return;
    for (const entry of this.view.entries) {
      if (entry.kind !== 'post' || entry.payload?.communication || !/^[a-f0-9]{32}$/.test(entry.id)) continue;
      // This pass claims the post's newest body read on the body clock: a
      // removal or a newer body read that lands while this GET is in flight
      // owns the text now, so a delayed payload landing over either must
      // drop instead of applying. A newer Check progress updates only the
      // pipeline and never orphans this read in `loading`.
      const order = this.stampCommentBody(entry.id);
      // Pipeline freshness at body-start: the served pipeline upserts only
      // while no newer progress write landed, so a delayed body never
      // regresses a recovered pipeline.
      const progressMark = this.commentProgressOrder.get(entry.id);
      try {
        const response = await context.transport.request(`factory/feed/comments/${entry.id}`, {
          method: 'GET',
          signal: AbortSignal.timeout(15000),
        });
        if (serial !== this.serial) return;
        if (!response.ok) {
          // A newer body read or a removal that landed while this GET was
          // in flight owns the text now: this failure is stale and drops,
          // so a delayed 503/404 never overwrites newer good text or the
          // removal tombstone with an older failure.
          const superseded = this.commentBodyOrder.get(entry.id) !== order;
          const removedWhileInFlight = (this.removalOrder.get(entry.id) ?? -1) > order;
          // Access loss retires the pipeline controls too, not just the
          // text: a missing post keeps no Check progress behind it.
          if (!superseded && !removedWhileInFlight && (response.status === 404 || response.status === 403))
            this.applyAuthoritativeRemoval(entry.id, this.entryKeyFor(entry));
          if (superseded || removedWhileInFlight) continue;
          const bodies = new Map(this.view.bodies);
          bodies.set(entry.id, {
            state: 'unavailable',
            commentId: entry.id,
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
        // Requested-ID binding: a detail naming a different comment is
        // malformed for this entry — unreadable, never a purge or a flag
        // on another comment. A newer body read or a removal that owns
        // the text now keeps it: the delayed mismatch drops instead of
        // overwriting newer good text or the tombstone.
        if (detail.commentId !== entry.id) {
          const superseded = this.commentBodyOrder.get(entry.id) !== order;
          const removedWhileInFlight = (this.removalOrder.get(entry.id) ?? -1) > order;
          if (superseded || removedWhileInFlight) continue;
          const bodies = new Map(this.view.bodies);
          bodies.set(entry.id, {
            state: 'unavailable',
            commentId: entry.id,
            reason: 'Discussion text unavailable; the post keeps its place.',
          });
          this.view = {...this.view, bodies};
          continue;
        }
        // Terminal removed state outranks pipeline progress: a removed
        // detail never upserts a pending pipeline, and its stale controls
        // retire so no Check progress survives on the tombstoned post.
        if (detail.removed) {
          this.applyAuthoritativeRemoval(entry.id, this.entryKeyFor(entry));
          const bodies = new Map(this.view.bodies);
          bodies.set(entry.id, {
            state: 'unavailable',
            commentId: entry.id,
            reason: 'That discussion post was removed. Its pipeline record is kept where stored.',
          });
          this.view = {...this.view, bodies};
          continue;
        }
        // A removal or a newer body read that landed while this GET was in
        // flight owns the row: the delayed healthy payload drops instead
        // of restoring text or pipeline over the tombstone. A newer Check
        // progress only owns the pipeline: the text still lands so the
        // read never strands in `loading`, while the served pipeline
        // upserts only while no newer progress write landed.
        const superseded = this.commentBodyOrder.get(entry.id) !== order;
        const removedWhileInFlight = (this.removalOrder.get(entry.id) ?? -1) > order;
        if (!superseded && !removedWhileInFlight) {
          const bodies = new Map(this.view.bodies);
          bodies.set(entry.id, {state: 'text', body: detail.body, edited: detail.edited});
          this.view = {...this.view, bodies};
          if (
            detail.pipeline &&
            detail.pipeline.commentId === entry.id &&
            this.commentProgressOrder.get(entry.id) === progressMark
          ) {
            this.upsertOutcome(this.entryKeyFor(entry), detail.pipeline);
            // The applied pipeline owns the row now: a slower older read
            // landing later drops instead of overwriting it.
            this.stampCommentProgress(entry.id);
          }
        }
      } catch {
        if (serial !== this.serial) return;
        // A delayed unreadable landing never overwrites a newer body read
        // or a removal that owns the text now.
        const superseded = this.commentBodyOrder.get(entry.id) !== order;
        const removedWhileInFlight = (this.removalOrder.get(entry.id) ?? -1) > order;
        if (superseded || removedWhileInFlight) continue;
        const bodies = new Map(this.view.bodies);
        bodies.set(entry.id, {
          state: 'unavailable',
          commentId: entry.id,
          reason: 'Discussion text unavailable; the post keeps its place.',
        });
        this.view = {...this.view, bodies};
      }
    }
  }

  // reloadPostBody re-reads one discussion post's stored text after a failed
  // load. The post keeps its place and drafts are untouched; an unreadable
  // payload keeps its tombstone instead of an invented body. A removed
  // detail is terminal and never upserts pipeline progress; a detail naming
  // another comment is malformed for this retry.
  private async reloadPostBody(commentId: string) {
    const context = this.current,
      serial = this.serial;
    if (!context || !/^[a-f0-9]{32}$/.test(commentId)) return;
    // This retry claims the post's newest body read on the body clock: a
    // removal or a newer body read owns the text now, while a newer Check
    // progress updates only the pipeline — the retry text still lands so
    // the post never strands in `loading`.
    const order = this.stampCommentBody(commentId);
    // Pipeline freshness at retry-start: the served pipeline upserts only
    // while no newer progress write landed.
    const progressMark = this.commentProgressOrder.get(commentId);
    const loading = new Map(this.view.bodies);
    loading.set(commentId, {state: 'loading'});
    this.view = {...this.view, bodies: loading};
    try {
      const response = await context.transport.request(`factory/feed/comments/${commentId}`, {
        method: 'GET',
        signal: AbortSignal.timeout(15000),
      });
      if (serial !== this.serial) return;
      // A newer body read or a removal that landed while this retry was in
      // flight owns the text now: an older failure landing here drops
      // instead of overwriting newer good text or the tombstone.
      const landed = this.commentBodyOrder.get(commentId) === order;
      const removedWhileInFlight = (this.removalOrder.get(commentId) ?? -1) > order;
      if (!landed || removedWhileInFlight) return;
      if (!response.ok) {
        if (response.status === 404 || response.status === 403) {
          // Access loss retires the pipeline controls too, not just the
          // text: a missing post keeps no Check progress behind it.
          const entry = this.view.entries.find(
            (candidate) => candidate.kind === 'post' && candidate.id === commentId
          );
          this.applyAuthoritativeRemoval(commentId, entry ? this.entryKeyFor(entry) : undefined);
        }
        // The map merges against the live view here: no body parse ran, so
        // nothing landed between the response and this write.
        const bodies = new Map(this.view.bodies);
        bodies.set(commentId, {
          state: 'unavailable',
          commentId,
          reason:
            response.status === 404
              ? 'That discussion post is no longer visible. Its pipeline record is kept where stored.'
              : 'Discussion text unavailable; the post keeps its place.',
        });
        this.view = {...this.view, bodies};
        return;
      }
      const detail = parseCommentDetail(await readSodaJSON(response));
      if (serial !== this.serial) return;
      // Requested-ID binding: a detail naming a different comment is
      // malformed for this retry — unreadable, never a purge or a flag
      // on another comment. A newer body read or a removal that owns
      // the text now keeps it: the delayed mismatch drops without
      // assigning, so a stale response never overwrites the live map.
      const landedNow = this.commentBodyOrder.get(commentId) === order;
      const removedNow = (this.removalOrder.get(commentId) ?? -1) > order;
      if (detail.commentId !== commentId) {
        if (!landedNow || removedNow) return;
        const bodies = new Map(this.view.bodies);
        bodies.set(commentId, {
          state: 'unavailable',
          commentId,
          reason: 'Discussion text unavailable; the post keeps its place.',
        });
        this.view = {...this.view, bodies};
        return;
      }
      if (detail.removed) {
        // Terminal removed state outranks pipeline progress: a removed
        // detail never upserts a pending pipeline, and its stale controls
        // retire so no Check progress survives on the tombstoned post. The
        // removal helper already wrote the live view; the message below
        // merges against that live map, never a pre-parse snapshot.
        const entry = this.view.entries.find((candidate) => candidate.kind === 'post' && candidate.id === commentId);
        this.applyAuthoritativeRemoval(commentId, entry ? this.entryKeyFor(entry) : undefined);
        const bodies = new Map(this.view.bodies);
        bodies.set(commentId, {
          state: 'unavailable',
          commentId,
          reason: 'That discussion post was removed. Its pipeline record is kept where stored.',
        });
        this.view = {...this.view, bodies};
        return;
      }
      // A removal or a newer body read that landed while this retry was in
      // flight owns the row: the delayed healthy payload drops without
      // assigning instead of restoring text or pipeline over the tombstone.
      // A newer Check progress only owns the pipeline: the retry text still
      // lands and clears `loading`, while the served pipeline upserts only
      // while no newer progress write landed.
      if (!landedNow || removedNow) return;
      // The map merges against the live view after the parse: a newer body
      // read or removal for another post that landed mid-parse is kept —
      // only this post's entry is set.
      const bodies = new Map(this.view.bodies);
      bodies.set(commentId, {state: 'text', body: detail.body, edited: detail.edited});
      this.view = {...this.view, bodies};
      if (
        detail.pipeline &&
        detail.pipeline.commentId === commentId &&
        this.commentProgressOrder.get(commentId) === progressMark
      ) {
        const entry = this.view.entries.find((candidate) => candidate.kind === 'post' && candidate.id === commentId);
        if (entry) {
          this.upsertOutcome(this.entryKeyFor(entry), detail.pipeline);
          // The applied pipeline owns the row now: a slower older read
          // landing later drops instead of overwriting it.
          this.stampCommentProgress(commentId);
        }
      }
    } catch {
      if (serial !== this.serial) return;
      // A delayed unreadable landing never overwrites a newer body read
      // or a removal that owns the text now.
      const superseded = this.commentBodyOrder.get(commentId) !== order;
      const removedWhileInFlight = (this.removalOrder.get(commentId) ?? -1) > order;
      if (superseded || removedWhileInFlight) return;
      const bodies = new Map(this.view.bodies);
      bodies.set(commentId, {
        state: 'unavailable',
        commentId,
        reason: 'Discussion text unavailable; the post keeps its place.',
      });
      this.view = {...this.view, bodies};
    }
  }

  // openRoleReply resolves a "Read the role's reply" link inside its
  // original thread: the thread opens, existing pages read until the
  // target comment or exhaustion, then the row scrolls into view and
  // takes focus. A missing target gets an honest inline explanation
  // (removed, not permitted, or hidden by the current view) with the
  // path to make it visible where one exists. Reads only; no intake.
  private async openRoleReply(link: HTMLElement) {
    const targetId = link.dataset.roleReplyOpen ?? '';
    if (!/^[a-f0-9]{32}$/.test(targetId) || !this.current) return;
    // Selecting a role reply is itself navigation: bump the action
    // generation FIRST — before the busy check — so a reselected reply is
    // never ignored while its own lookup is in flight and a stale reply
    // can never stay current after the user moves on.
    this.actionSerial++;
    const action = this.actionSerial;
    if (this.roleReplyBusy.has(targetId)) {
      // The same reply is already resolving: remember only the newest
      // selection. The in-flight run drains this slot when it lands, so
      // the latest selection always wins and the stale run stays silent.
      // The entry captures the full scope token so the drain can refuse
      // to replay it after a context switch.
      const queuedContext = this.current;
      this.queuedRoleReply = {
        link,
        action,
        serial: this.serial,
        query: this.querySerial,
        library: this.librarySourceSerial,
        transport: queuedContext?.transport,
        repository: queuedContext?.repository,
        account: queuedContext?.account,
        global: queuedContext?.global,
      };
      return;
    }
    await this.runRoleReply(link, targetId, action);
    // Drain a queued newest selection left while this run was in flight.
    // A newer navigation moving on again supersedes the slot instead.
    for (;;) {
      const queued = this.queuedRoleReply;
      if (!queued) return;
      this.queuedRoleReply = undefined;
      const queuedId = queued.link.dataset.roleReplyOpen ?? '';
      const current = this.current;
      const scopeIntact =
        current !== undefined &&
        queued.serial === this.serial &&
        queued.query === this.querySerial &&
        queued.library === this.librarySourceSerial &&
        queued.action === this.actionSerial &&
        queued.transport === current.transport &&
        queued.repository === current.repository &&
        queued.account === current.account &&
        queued.global === current.global;
      if (!scopeIntact || !/^[a-f0-9]{32}$/.test(queuedId) || this.roleReplyBusy.has(queuedId)) {
        // Superseded, invalid, moved to another project or account, gone,
        // or still owned by another in-flight run — that run drains it.
        // Only a still-newest pending selection under the identical scope
        // survives; never a superseded or cross-context one.
        if (scopeIntact && this.roleReplyBusy.has(queuedId)) this.queuedRoleReply = queued;
        return;
      }
      await this.runRoleReply(queued.link, queuedId, queued.action);
    }
  }

  private async runRoleReply(link: HTMLElement, targetId: string, action: number) {
    const context = this.current;
    if (!context) return;
    // Navigation token: a filter change, drawer/library/back move, another
    // role-reply selection, or context switch during the lookup must leave
    // the stale view untouched — no focus, no scroll, no message writes
    // after the user moves on. The context identity rides along so a run
    // started under one project or account can never validate under
    // another, even if a generation counter were ever to realign.
    const token = {
      serial: this.serial,
      query: this.querySerial,
      library: this.librarySourceSerial,
      action,
      transport: context.transport,
      repository: context.repository,
      account: context.account,
      global: context.global,
    };
    const live = () =>
      token.serial === this.serial &&
      token.query === this.querySerial &&
      token.library === this.librarySourceSerial &&
      token.action === this.actionSerial &&
      this.current !== undefined &&
      token.transport === this.current.transport &&
      token.repository === this.current.repository &&
      token.account === this.current.account &&
      token.global === this.current.global;
    const say = (message: string) => {
      if (!live()) return;
      const anchor = this.querySelector(`[data-role-reply-open="${targetId}"]`);
      const region = anchor?.closest('.soda-feed-consultation')?.querySelector('[data-role-reply-status]');
      if (region) region.textContent = message;
      else this.view = {...this.view, message};
    };
    this.roleReplyBusy.add(targetId);
    try {
      const key = link.dataset.roleReplyEntry ?? '',
        entry = key ? this.entryForKey(key) : undefined;
      if (!entry) {
        say('The role’s reply is not visible here. Clear the filters to check the full conversation.');
        return;
      }
      say('Looking for the role’s reply…');
      this.openDrawer(key);
      // Page the original thread until the target lands, the pages run
      // out, a page fails, or the cursor stops advancing. Each await
      // rejoins the actual in-flight read; a failed page surfaces its
      // error instead of retrying the same GET or blaming the thread,
      // and a bounded search that never reaches the target says so
      // instead of calling it out of view.
      let attempted = false;
      let exhausted = false;
      let pages = 0;
      for (;;) {
        if (!live()) return;
        const thread = this.threads.get(key);
        if (thread?.items.some((item) => item.commentId === targetId)) break;
        if (thread?.message && attempted) {
          say(thread.message);
          return;
        }
        if (thread && !thread.busy && !this.readingThreads.has(key) && !thread.next && (!thread.message || attempted)) {
          exhausted = true;
          break;
        }
        if (pages >= 25) break;
        const before = thread?.next ?? 0;
        const count = thread?.items.length ?? 0;
        await this.loadReplies(entry, before > 0);
        if (!live()) return;
        attempted = true;
        const after = this.threads.get(key);
        if (!after || after.message) {
          say(
            after?.message ??
              'The role’s reply could not be read here; it may have been removed or is not shared with you. Refresh replies checks the stored outcome again.'
          );
          return;
        }
        if (after.next === before && after.items.length === count && before !== 0) break;
        pages++;
      }
      if (!live()) return;
      this.requestUpdate();
      await this.updateComplete;
      if (!live()) return;
      const row = this.querySelector('#comment-' + targetId);
      if (row instanceof HTMLElement) {
        row.scrollIntoView({block: 'center'});
        row.setAttribute('tabindex', '-1');
        row.focus({preventScroll: true});
        say('');
        return;
      }
      try {
        if (!live()) return;
        const detail = parseCommentDetail(await readJSON(context.transport, 'factory/feed/comments/' + targetId));
        if (!live()) return;
        say(
          detail.removed
            ? 'The role’s reply is no longer available; it was removed. No other copy is kept.'
            : exhausted
              ? 'The role’s reply is not in this thread under the current view. Clear the filters to check the full conversation.'
              : 'The role’s reply was not found in the pages read so far. Use Show more to keep paging through the older replies.'
        );
      } catch {
        if (!live()) return;
        say(
          'The role’s reply could not be read here; it may have been removed or is not shared with you. Refresh replies checks the stored outcome again.'
        );
      }
    } finally {
      this.roleReplyBusy.delete(targetId);
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

  private async reviewProductChange(comment: string, revision: number, repository: number) {
    const context = this.current,
      serial = this.serial;
    if (
      !context ||
      !/^[a-f0-9]{32}$/.test(comment) ||
      !Number.isSafeInteger(revision) ||
      revision < 0 ||
      !Number.isSafeInteger(repository) ||
      repository <= 0 ||
      this.view.commentBusy.includes(comment)
    )
      return;
    this.view = {...this.view, commentBusy: [...this.view.commentBusy, comment]};
    try {
      const path = `factory/feed/comments/${comment}`;
      let value = object(await readJSON(context.transport, path));
      if (serial !== this.serial) return;
      let state = parseCommentPipelineState(value);
      check(Number(value.repository) === repository && state.commentId === comment && state.revision === revision);
      let change = state.productChange;
      check(change && change.state !== 'superseded');
      if (change.state === 'review_required') {
        check(change.canReview);
        value = object(
          await readJSON(context.transport, path + '/review', {
            method: 'POST',
            headers: {'Content-Type': 'application/json'},
            body: JSON.stringify({revision}),
          })
        );
        if (serial !== this.serial) return;
        state = parseCommentPipelineState(value);
        change = state.productChange;
        check(Number(value.repository) === repository && state.commentId === comment && state.revision === revision);
        check(change && (change.state === 'draft' || change.state === 'accepted'));
      }
      for (const [key, thread] of this.threads)
        this.threads.set(key, {
          ...thread,
          items: thread.items.map((item) => (item.commentId === comment ? {...item, pipeline: state} : item)),
        });
      const outcomes = new Map(this.view.outcomes);
      for (const [key, rows] of outcomes)
        outcomes.set(
          key,
          rows.map((row) => (row.commentId === comment ? state : row))
        );
      // This fresh read owns the comment's pipeline now: a delayed Check
      // progress GET still in flight must drop instead of overwriting it.
      this.stampCommentProgress(comment);
      // Accepted collaborator progress is the authorized visible evidence,
      // never the owner-private intent route: no intent event leaves here
      // without canReview. The private request control stays owner-only.
      if (change.state === 'accepted' && !change.canReview) {
        this.view = {
          ...this.view,
          outcomes,
          message: 'The changed direction is accepted. Follow its build and verification in the linked evidence.',
        };
        return;
      }
      // An explicit unavailable projection never reads as a save: the lane
      // detail did not arrive, so report it as unavailable and retryable
      // without opening the private intent route.
      if (change.state === 'unavailable') {
        this.view = {
          ...this.view,
          outcomes,
          message: unavailableChangeMessage,
        };
        this.unavailableChangeComment = comment;
        return;
      }
      // Navigation stays owner-only: a draft or accepted change opens only
      // with review access. Anything else keeps the truthful waiting state
      // without claiming a save or opening the intent route.
      if (!change.canReview) {
        this.view = {
          ...this.view,
          outcomes,
          message: 'Waiting for the request owner to review this proposed change.',
        };
        return;
      }
      this.view = {
        ...this.view,
        outcomes,
        message:
          change.state === 'accepted'
            ? 'Opening the accepted change.'
            : 'Change saved for review. Review and accept the direction in the same build request.',
      };
      // The proposed change opens on its request's page (at Update proposal).
      const places = this.placesFor(repository);
      if (places) this.go(places.request(change.intent));
    } catch {
      if (serial === this.serial)
        this.view = {
          ...this.view,
          message:
            'This change could not be opened. Refresh the conversation to check its current direction and your access; the comment is kept.',
        };
    } finally {
      if (serial === this.serial)
        this.view = {...this.view, commentBusy: this.view.commentBusy.filter((id) => id !== comment)};
    }
  }

  /** Same-origin navigation: the host swaps SodaOS places in its shell. */
  private go(path: string) {
    const navigate = this.current?.transport.navigate;
    if (navigate) navigate(path);
    else location.assign(path);
  }

  private onFeedClick = (event: Event) => {
    const target = event.target;
    if (!(target instanceof HTMLElement)) return;
    const productChange = target.closest<HTMLElement>('[data-comment-product-change]');
    if (productChange) {
      void this.reviewProductChange(
        productChange.dataset.commentProductChange ?? '',
        Number(productChange.dataset.productRevision),
        Number(productChange.dataset.productRepository)
      );
      return;
    }
    const drawerToggle = target.closest<HTMLElement>('[data-feed-drawer-toggle]');
    if (drawerToggle?.dataset.feedDrawerToggle) {
      const key = drawerToggle.dataset.feedDrawerToggle;
      // Drawer open/close is navigation: invalidate in-flight role-reply
      // lookups so a delayed page can neither write nor focus afterwards.
      this.actionSerial++;
      if (this.openDrawers.has(key)) {
        this.openDrawers.delete(key);
        this.requestUpdate();
      } else this.openDrawer(key);
      return;
    }
    const roleReply = target.closest<HTMLElement>('[data-role-reply-open]');
    if (roleReply?.dataset.roleReplyOpen) {
      event.preventDefault();
      void this.openRoleReply(roleReply);
      return;
    }
    const revise = target.closest<HTMLElement>('[data-comment-revise]');
    if (revise?.dataset.commentRevise && this.current && !this.pendingRevision) {
      const serial = this.serial;
      void readJSON(this.current.transport, `factory/feed/comments/${revise.dataset.commentRevise}`)
        .then((value) => {
          if (serial !== this.serial) return;
          const data = object(value);
          if (data.can_retry !== true) return;
          const account = this.revisionAccount();
          const repository = String(
            (typeof data.repository === 'number' || typeof data.repository === 'string' ? data.repository : '') ||
              this.current?.repository ||
              ''
          );
          const commentId = String(data.comment_id);
          const revision = Number(data.revision);
          // Restoring a saved draft never submits: the reopened panel only
          // re-serves the stored text for review.
          const saved = readRevisionDraft(account, repository, commentId, revision);
          const stale = saved ? undefined : findStaleRevisionDraft(account, repository, commentId, revision);
          const restored = saved ?? stale?.draft;
          // A fresh panel-open mints a lineage; a stale restore inherits the
          // restored draft's token so later persists stay on the same line.
          // The stale restore also appends the superseded snapshot (plus
          // the restored draft's own chain) so every persist carries the
          // full ancestor chain across reloads and revision advances.
          const restoredAncestors: RevisionAncestorSnapshot[] = stale
            ? [
                ...(stale.draft.ancestors ?? []).filter((row) => row.revision !== stale.revision),
                {
                  revision: stale.revision,
                  body: stale.draft.body,
                  ...(stale.draft.original !== undefined ? {original: stale.draft.original} : {}),
                  ...(stale.draft.lineage !== undefined ? {lineage: stale.draft.lineage} : {}),
                },
              ]
            : [];
          this.revising = {
            id: commentId,
            revision,
            repository,
            body: String(data.body),
            lineage: saved?.lineage ?? stale?.draft.lineage ?? newIdempotencyKey(),
            ...(stale
              ? {
                  restoredFrom: {
                    revision: stale.revision,
                    body: stale.draft.body,
                    ...(stale.draft.original !== undefined ? {original: stale.draft.original} : {}),
                  },
                }
              : {}),
            ...(stale
              ? {ancestors: restoredAncestors}
              : saved?.ancestors !== undefined && saved.ancestors.length > 0
                ? {ancestors: saved.ancestors}
                : {}),
          };
          this.revisionPersisted = saved
            ? {
                account,
                repository,
                id: commentId,
                revision,
                body: saved.body,
                ...(saved.original !== undefined ? {original: saved.original} : {}),
              }
            : undefined;
          if (data.route === 'clarify')
            this.revising = {
              ...this.revising,
              original: String(data.body),
              body: '',
              question:
                typeof data.detail === 'string' ? data.detail : 'What should change, and where should it apply?',
            };
          if (restored && (restored.body !== '' || restored.original !== undefined)) {
            this.revising.body = restored.body;
            if (this.revising.original !== undefined || restored.original !== undefined)
              this.revising.original = restored.original ?? this.revising.original ?? String(data.body);
            if (stale) {
              this.revising.stale = true;
              this.revising.repository = repository;
              this.view = {
                ...this.view,
                message:
                  'Source request changed since this draft was saved. Review the restored text before submitting.',
              };
            }
          }
          this.requestUpdate();
          void this.updateComplete.then(() => this.querySelector<HTMLElement>('[data-revision-body]')?.focus());
        })
        .catch(() => {
          this.view = {...this.view, message: 'Could not load this request. Refresh and try again.'};
        });
      return;
    }
    if (target.closest('[data-revision-cancel]')) {
      this.discardRevising();
      this.requestUpdate();
      return;
    }
    if (target.closest('[data-revision-refresh]')) {
      void this.refreshRevision();
      return;
    }
    if (target.closest('[data-revision-submit]')) {
      void this.reviseComment();
      return;
    }
    const answer = target.closest('[data-feed-answer]');
    if (answer instanceof HTMLButtonElement && answer.dataset.feedAnswer) {
      const key = answer.dataset.feedAnswer,
        text = answer.dataset.feedAnswerText ?? '';
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
    const postRetry = target.closest('[data-post-body-retry]');
    if (postRetry instanceof HTMLElement && postRetry.dataset.postBodyRetry) {
      void this.reloadPostBody(postRetry.dataset.postBodyRetry);
      return;
    }
    const retry = target.closest('[data-comment-retry]');
    if (retry instanceof HTMLElement && retry.dataset.commentRetry) {
      const outcome = retry.dataset.commentRetry;
      // Drawer reply rows render from threads, not outcomes: resolve the
      // comment in both. An absent target stays inert — no GET, no message,
      // no invented state.
      const key =
        [...this.view.outcomes].find(([, rows]) => rows.some((row) => row.commentId === outcome))?.[0] ??
        [...this.threads].find(([, thread]) => thread.items.some((item) => item.commentId === outcome))?.[0];
      if (key) void this.checkCommentProgress(key, outcome);
    }
  };

  private async reviseComment() {
    const edit = this.revising,
      context = this.current,
      serial = this.serial;
    if (!edit || !context || this.view.busy || this.pendingRevision || !edit.body.trim()) return;
    if (edit.unverified) {
      this.view = {
        ...this.view,
        message: 'The live revision could not be confirmed. Refresh the revision status before submitting.',
      };
      return;
    }
    const body = edit.original === undefined ? edit.body : edit.original + '\n\nClarification: ' + edit.body;
    if ([...body].length > 4000) {
      this.view = {
        ...this.view,
        message:
          'Keep the original request and answer within 4,000 characters. You can shorten the original request below.',
      };
      return;
    }
    // Snapshot the submitted text: keystrokes that land while the POST is
    // pending persist as newer unsent input and must survive its success.
    // A failed response keeps the drafts; restoration never auto-submits.
    const submitted: {
      account: string;
      repository: string;
      id: string;
      revision: number;
      body: string;
      original?: string;
      lineage: string;
      restoredFrom?: {revision: number; body: string; original?: string};
      ancestors?: RevisionAncestorSnapshot[];
    } = {
      account: this.revisionAccount(),
      repository: edit.repository,
      id: edit.id,
      revision: edit.revision,
      body: edit.body,
      ...(edit.original !== undefined ? {original: edit.original} : {}),
      lineage: edit.lineage,
      ...(edit.restoredFrom ? {restoredFrom: edit.restoredFrom} : {}),
      ...(edit.ancestors !== undefined && edit.ancestors.length > 0 ? {ancestors: edit.ancestors} : {}),
    };
    this.pendingRevision = submitted;
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
      this.pendingRevision = undefined;
      if (serial !== this.serial) return;
      // Adopt the real post-submit source revision so retained editing state
      // and any second Submit use a live CAS instead of the spent one. The
      // retry answer carries it; an authorized refetch covers responses that
      // do not.
      let adopted = revisionNumberOf(value) ?? submitted.revision;
      if (revisionNumberOf(value) === undefined) {
        try {
          adopted =
            revisionNumberOf(await readJSON(context.transport, `factory/feed/comments/${submitted.id}`)) ??
            submitted.revision;
        } catch {
          /* The submitted revision stays authoritative. */
        }
        if (serial !== this.serial) return;
      }
      // Re-read the live panel after every await: keystrokes may have landed
      // while the POST or refetch was pending, and the panel may have moved.
      const live = this.revising;
      const samePanel = live !== undefined && live.id === submitted.id && live.revision === submitted.revision;
      const newerUnsent =
        samePanel &&
        (live.body !== submitted.body || (live.original ?? undefined) !== (submitted.original ?? undefined));
      const verifiedFresh = adopted > submitted.revision;
      retireSubmittedRevisionDrafts(
        submitted.account,
        submitted.repository,
        submitted.id,
        submitted.body,
        submitted.original,
        adopted,
        submitted.revision,
        submitted.lineage,
        submitted.ancestors ??
          (submitted.restoredFrom
            ? [
                {
                  revision: submitted.restoredFrom.revision,
                  body: submitted.restoredFrom.body,
                  ...(submitted.restoredFrom.original !== undefined ? {original: submitted.restoredFrom.original} : {}),
                  lineage: submitted.lineage,
                },
              ]
            : [])
      );
      if (newerUnsent && live !== undefined) {
        if (verifiedFresh) {
          // Re-key the surviving newer text under the adopted revision so a
          // later reopen restores it without a stale-source warning — unless
          // another tab already stored different text there, which stays
          // intact while the live panel keeps ours sendable. The surviving
          // text keeps the submitted line's ancestor chain plus an exact
          // snapshot of the own adopted text (B at the submitted revision)
          // so a later submit still retires every unchanged superseded
          // snapshot instead of resurrecting the spent key.
          const rekeyBase =
            live.ancestors !== undefined && live.ancestors.length > 0 ? live.ancestors : (submitted.ancestors ?? []);
          const rekeyAncestors: RevisionAncestorSnapshot[] = [
            ...rekeyBase.filter((row) => row.revision !== submitted.revision),
            {
              revision: submitted.revision,
              body: live.body.slice(0, 4000),
              ...(live.original !== undefined ? {original: live.original.slice(0, 4000)} : {}),
              lineage: submitted.lineage,
            },
          ];
          writeRevisionDraftUnlessDivergent(submitted.account, submitted.repository, submitted.id, adopted, {
            body: live.body.slice(0, 4000),
            ...(live.original !== undefined ? {original: live.original.slice(0, 4000)} : {}),
            lineage: submitted.lineage,
            ancestors: rekeyAncestors,
            savedAt: Date.now(),
          });
          // The tracker follows this panel's own text under the adopted key,
          // never a foreign write the re-key preserved: the next keystroke
          // then conflicts explicitly instead of overwriting it.
          this.revisionPersisted = {
            account: submitted.account,
            repository: submitted.repository,
            id: submitted.id,
            revision: adopted,
            body: live.body.slice(0, 4000),
            ...(live.original !== undefined ? {original: live.original.slice(0, 4000)} : {}),
          };
          const adoptedPanel = {...live, revision: adopted, ancestors: rekeyAncestors};
          delete adoptedPanel.stale;
          delete adoptedPanel.conflict;
          this.revising = adoptedPanel;
          this.view = {
            ...this.view,
            busy: false,
            message: 'Request updated. Processing will continue here. Your newer edits are kept, ready to send.',
          };
        } else {
          // The submit succeeded but no live revision could be verified:
          // keep the text under its persisted key, mark the panel
          // unverified so Submit stays disabled, and require review
          // instead of claiming a live CAS.
          if (samePanel) this.revising = {...live, unverified: true};
          this.view = {
            ...this.view,
            busy: false,
            message:
              'Request updated, but the live revision could not be confirmed. Review the restored text before submitting.',
          };
        }
      } else {
        if (samePanel) {
          this.revising = undefined;
          this.revisionPersisted = undefined;
        }
        this.view = {...this.view, busy: false, message: 'Request updated. Processing will continue here.'};
      }
      await this.readLatest(false);
    } catch (error) {
      this.pendingRevision = undefined;
      if (serial === this.serial)
        this.view = {
          ...this.view,
          busy: false,
          message: error instanceof Error ? error.message : 'Could not update this request. Your edits are kept.',
        };
    }
  }

  // refreshRevision re-checks the live source revision for an unverified
  // panel through the same authorized comment GET the reopen path uses. A
  // confirmed newer revision is adopted (clearing unverified) with the kept
  // text held for review; anything else keeps unverified with an explicit
  // message. Never auto-submits and never invents a CAS.
  private async refreshRevision() {
    const edit = this.revising,
      context = this.current,
      serial = this.serial;
    if (!edit || !context || !edit.unverified) return;
    try {
      const value = object(await readJSON(context.transport, `factory/feed/comments/${edit.id}`));
      if (serial !== this.serial) return;
      const fresh = revisionNumberOf(value);
      const live = this.revising;
      if (!live || live.id !== edit.id || live.revision !== edit.revision || !live.unverified) return;
      if (fresh !== undefined && fresh > live.revision) {
        // Explicit-refresh re-key carries the same own-text snapshot as the
        // post-success re-key: the kept text superseded its old revision, so
        // a later submit retires that key while unchanged.
        const refreshAncestors: RevisionAncestorSnapshot[] = [
          ...(live.ancestors ?? []).filter((row) => row.revision !== live.revision),
          {
            revision: live.revision,
            body: live.body.slice(0, 4000),
            ...(live.original !== undefined ? {original: live.original.slice(0, 4000)} : {}),
            lineage: live.lineage,
          },
        ];
        const adopted = {...live, revision: fresh, ancestors: refreshAncestors};
        delete adopted.unverified;
        this.revising = adopted;
        writeRevisionDraftUnlessDivergent(this.revisionAccount(), live.repository, live.id, fresh, {
          body: live.body.slice(0, 4000),
          ...(live.original !== undefined ? {original: live.original.slice(0, 4000)} : {}),
          lineage: live.lineage,
          ancestors: refreshAncestors,
          savedAt: Date.now(),
        });
        this.revisionPersisted = {
          account: this.revisionAccount(),
          repository: live.repository,
          id: live.id,
          revision: fresh,
          body: live.body.slice(0, 4000),
          ...(live.original !== undefined ? {original: live.original.slice(0, 4000)} : {}),
        };
        this.view = {...this.view, message: 'Live revision confirmed. Review the kept text before submitting.'};
      } else
        this.view = {
          ...this.view,
          message: 'The live revision is still unavailable. Your text is kept; try refreshing again.',
        };
    } catch {
      if (serial !== this.serial) return;
      this.view = {
        ...this.view,
        message: 'The live revision is still unavailable. Your text is kept; try refreshing again.',
      };
    }
  }

  protected render() {
    const confirmed = activityTimestamp(this.view.confirmedUnix);
    return html`<section class="soda-feed" aria-label="Feed" @click=${this.onFeedClick} @input=${this.onDraft}>
      <header class="soda-feed-toolbar">
        <span class="soda-feed-toolbar-actions">
          ${
            this.ownPlaces()
              ? html`<a class="soda-feed-utility" href=${this.ownPlaces()?.knowledge() ?? ''}>Knowledge</a>`
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
                  this.librarySourceSerial++;
                  this.librarySourcePending = undefined;
                  this.actionSerial++;
                  this.focusedEntry = undefined;
                  this.requestUpdate();
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
      ${this.view.readFailed && this.view.entries.length ? html`<p class="soda-feed-updated">Showing the last loaded activity. Refresh to check for updates.</p>` : ''}
      ${
        this.revising
          ? html`<section class="soda-feed-revision" aria-label="Revise request">
              ${
                this.revising.stale
                  ? html`<p role="alert">
                      Source request changed since this draft was saved. Review the restored text before submitting.
                    </p>`
                  : ''
              }
              ${
                this.revising.unverified
                  ? html`<p role="alert">
                      The live revision could not be confirmed. Refresh the revision status before submitting.
                    </p>`
                  : ''
              }
              ${
                this.revising.conflict
                  ? html`<p role="alert">
                      Another tab saved a different draft for this revision. Your text is kept here and the saved draft
                      was left untouched.
                    </p>`
                  : ''
              }
              ${
                this.revising.original !== undefined
                  ? html`<details>
                      <summary>Original request · edit if needed</summary>
                      <label
                        >Original request<textarea
                          data-revision-original
                          rows="4"
                          maxlength="4000"
                          .value=${this.revising.original}
                        ></textarea>
                      </label>
                    </details>`
                  : ''
              }
              <label
                >${this.revising.question ?? 'Update your request'}<textarea
                  data-revision-body
                  rows="5"
                  maxlength="4000"
                  .value=${this.revising.body}
                ></textarea>
              </label>
              <p>Submitting reassesses this request and may start work using the project’s existing allowance.</p>
              <button
                type="button"
                data-revision-submit
                ?disabled=${this.view.busy || this.revising.unverified === true}
              >
                ${this.revising.question ? 'Submit answer' : 'Submit updated request'}</button
              >${
                this.revising.unverified
                  ? html`<button type="button" data-revision-refresh ?disabled=${this.view.busy}>
                      Refresh revision status
                    </button>`
                  : ''
              }<button type="button" data-revision-cancel>Cancel</button>
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
    const places = this.ownPlaces();
    const placesRef = places ? {places} : {};
    const outcomes = (this.view.outcomes.get(key) ?? []).map((state) =>
      describeCommentOutcome(
        state,
        {id: repository, path: ref?.path, label: ref?.label ?? 'This repository', ...placesRef},
        this.view.commentBusy.includes(state.commentId)
      )
    );
    const draft = this.draftFor(key);
    return html`<div class="soda-feed-comment-box">
      ${renderDiscussion({
        key,
        draft,
        label: 'Comment',
        submitLabel: 'Post comment',
        submitAvailable: draft.trim() !== '' && !this.view.commentBusy.includes(key),
        outcomes,
        discussionLink: undefined,
        collapsed: !this.commentOpen && draft.trim() === '',
        intakeText:
          'Ask a question or request a change. JEV connects your comment to existing work or asks the Product Lead to create new work.',
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
        ${
          subscription.revision === 0
            ? html`<p>Showing every repository you can read until you choose.</p>`
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
      </form>
    </details>`;
  }

  private renderList() {
    const entries = this.focusedEntry ? [this.focusedEntry] : this.view.entries;
    if (entries.length === 0) return this.renderEmptyFeed();
    return html`<div class="soda-feed-list">
        ${repeat(
          entries,
          (entry) => `${entry.kind}:${entry.id}`,
          (entry) => {
            const ref = this.view.labels.refs.get(entry.repository) ?? {
              path: undefined,
              label: `repository ${entry.repository}`,
            };
            const links = this.cardLinks(entry);
            const repository = {
              id: entry.repository,
              path: ref.path,
              label: ref.label,
              ...(links.places ? {places: links.places} : {}),
            };
            const key = `${entry.kind}:${entry.id}`;
            const outcomes: CommentOutcomeModel[] = (this.view.outcomes.get(key) ?? []).map((state) =>
              describeCommentOutcome(state, repository, this.view.commentBusy.includes(state.commentId))
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
              this.current?.transport.assetBase,
              links
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

  private async loadReplies(entry: FeedEntry, older = false, quiet = false): Promise<void> {
    const key = this.entryKeyFor(entry);
    const previous = this.threads.get(key);
    if (previous?.busy || this.readingThreads.has(key)) {
      // A page read is already in flight: await it instead of dropping the
      // caller or spinning until it lands.
      await this.threadReads.get(key);
      return;
    }
    const run = this.readThreadPage(entry, older, quiet);
    this.threadReads.set(key, run);
    try {
      await run;
    } finally {
      if (this.threadReads.get(key) === run) this.threadReads.delete(key);
    }
  }

  private async readThreadPage(entry: FeedEntry, older = false, quiet = false): Promise<void> {
    const context = this.current,
      source = commentSourceForEntry(entry),
      serial = this.serial;
    if (!context || !source) return;
    const key = this.entryKeyFor(entry),
      previous = this.threads.get(key);
    if (previous?.busy || this.readingThreads.has(key)) return;
    this.readingThreads.add(key);
    this.threads.set(key, {
      ...previous,
      items: previous?.items ?? [],
      pages: previous?.pages ?? [],
      next: previous?.next ?? 0,
      busy: !quiet,
      message: '',
    });
    this.requestUpdate();
    // Removal-order mark for this page: a direct-GET tombstone/access-loss
    // that lands while the page is in flight owns the row, so the merge
    // below must drop it instead of resurrecting it.
    const pageRemovalMark = this.commentProgressTick;
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
      // Explicit tombstone rows (deleted/revoked flags) never render: the
      // reply listing excludes deleted/revoked sources, so a served
      // tombstone is an authoritative removal. Missing rows alone never
      // prove removal — only these explicit markers do. Filter them from
      // the insert set and drop them from the kept cache below.
      const explicitRemovedIds = new Set(
        items.filter((item) => item.removed).map((item) => item.commentId)
      );
      const liveItems = items.filter((item) => !item.removed);
      // The thread API serves pages newest-first. Reverse each page into
      // ascending canonical order before merging: same-second comments share
      // one `at`, so server order (not a timestamp or commentId sort) is the
      // canonical tiebreak, and windows keep it structurally below.
      liveItems.reverse();
      // A page never repeats an identity; drop a duplicated row defensively
      // so the splice below keeps each position once.
      const seenFresh = new Set<string>();
      const fresh = liveItems.filter((item) => {
        if (seenFresh.has(item.commentId)) return false;
        seenFresh.add(item.commentId);
        // A direct-GET removal that landed while this page was in flight
        // owns the row: the page must not reinsert it when it lands late.
        // Removals recorded before the request stay served — the server
        // enumerated this page after they were observed.
        if ((this.removalOrder.get(item.commentId) ?? -1) > pageRemovalMark) return false;
        return true;
      });
      // The sent cursor decides what the server served: without one the
      // server serves the newest page, which replaces (foreground) or heads
      // (quiet refresh) the thread; with one the page continues below the
      // window that produced the cursor. Both stay pinned to the pre-await
      // snapshot because they describe the request, not the merge.
      const sentBefore = older && previous?.next ? previous.next : 0;
      const latest = sentBefore === 0;
      const previousNext = previous?.next;
      // Merge against the live cache, not the pre-await snapshot: a progress
      // recovery that landed while this page was in flight owns the current
      // rows now, and folding stale pre-await pipelines over them would
      // overwrite the recovered state.
      const live = this.threads.get(key);
      const kept: CommentDetail[] = live?.items ?? [];
      const keptPages: ThreadPage[] =
        live?.pages ??
        (kept.length > 0 ? [{ids: kept.map((item) => item.commentId), nextBefore: live?.next ?? 0}] : []);
      const freshIds = new Set(fresh.map((item) => item.commentId));
      const overlaps = kept.some((item) => freshIds.has(item.commentId));
      let merged: CommentDetail[];
      let pages: ThreadPage[];
      let next: number;
      if (fresh.length === 0) {
        if (latest && result.next_before === 0) {
          // An empty complete refresh is authoritative: every reply the
          // thread served is gone (deleted/revoked), so the thread clears
          // to its honest empty state instead of keeping removed bodies.
          // Truncated or older-side pages never delete: an incomplete page
          // leaves kept replies (and the older-side cursor) alone.
          merged = [];
          pages = [];
          next = 0;
        } else {
          // An empty page loads nothing: every kept reply stays. A complete
          // (cursor-zero) older-side page still closes the older side.
          merged = kept;
          pages = keptPages;
          next = result.next_before === 0 ? 0 : (previousNext ?? 0);
          // An authoritative empty older page clears the covered cached
          // tail: with nothing older than the sent cursor, kept rows below
          // the anchor window are gone while the window and newer rows stay.
          if (!latest && result.next_before === 0) {
            const slot = keptPages.findIndex((page) => page.nextBefore === sentBefore);
            if (slot >= 0) {
              const shield = new Set(keptPages.slice(slot).flatMap((page) => page.ids));
              merged = kept.filter((item) => shield.has(item.commentId));
              const survivors = new Set(merged.map((item) => item.commentId));
              pages = keptPages
                .map((page) => ({
                  ids: page.ids.filter((id) => survivors.has(id)),
                  nextBefore: page.nextBefore,
                }))
                .filter((page) => page.ids.length > 0);
              next = 0;
            }
          }
        }
      } else if (latest && (result.next_before === 0 || !quiet)) {
        // A foreground (re)read — or a complete quiet refresh, which is the
        // whole thread — replaces the thread with the newest page; older
        // pages return through Show more. Whatever the server omits is gone:
        // a deleted or revoked reply never reinserts, and its body leaves
        // with it.
        merged = fresh;
        pages = [{ids: fresh.map((item) => item.commentId), nextBefore: result.next_before}];
        next = result.next_before;
      } else if (latest) {
        // A truncated quiet refresh covers only the newest window.
        if (!overlaps) {
          // No overlap proves no direction: a disjoint page is either a
          // strictly newer burst or an older head after the kept head was
          // removed. Comment `at` is mutable on edit and proves nothing —
          // not even distinct stamps — so no timestamp comparison orders,
          // positions, or deletes here; cursors alone carry order proof.
          // A kept thread whose older side is exhausted (complete bottom
          // coverage) keeps its rows below the fresh head: the burst
          // appends above and stays pageable through the gap. A kept
          // thread with an open older side resets to the latest
          // authoritative page: the newest read is authoritative for the
          // head, so whatever it omits is gone, and the fresh cursor keeps
          // the older side pageable so the surviving tail returns through
          // Show more.
          if ((previousNext ?? 0) === 0) {
            merged = [...kept, ...fresh];
            pages = [...keptPages, {ids: fresh.map((item) => item.commentId), nextBefore: result.next_before}];
            next = result.next_before;
          } else {
            merged = fresh;
            pages = [{ids: fresh.map((item) => item.commentId), nextBefore: result.next_before}];
            next = result.next_before;
          }
        } else {
          const spliced = spliceThreadPage(kept, keptPages, fresh, result.next_before);
          // A latest page is authoritative above its head: the server
          // serves newest-first, so a cached entry newer than the newest
          // served row is gone (deleted/revoked) and drops out, while the
          // older unserved tail below the fresh span stays covered by its
          // own windows.
          const headId = fresh[fresh.length - 1]?.commentId;
          const headPos =
            headId !== undefined ? spliced.merged.findIndex((item) => item.commentId === headId) : -1;
          if (headPos >= 0 && headPos < spliced.merged.length - 1) {
            merged = spliced.merged.slice(0, headPos + 1);
            const survivors = new Set(merged.map((item) => item.commentId));
            pages = spliced.pages
              .map((page) => ({ids: page.ids.filter((id) => survivors.has(id)), nextBefore: page.nextBefore}))
              .filter((page) => page.ids.length > 0);
          } else {
            merged = spliced.merged;
            pages = spliced.pages;
          }
          // A quiet refresh keeps the older-pages cursor while the window
          // that produced it is still anchored; otherwise the fresh cursor
          // takes over and Show more pages from the rebuilt windows.
          next =
            previousNext !== undefined && pages.some((page) => page.nextBefore === previousNext)
              ? previousNext
              : result.next_before;
        }
      } else if (!overlaps) {
        // A disjoint older page continues below the window whose cursor
        // fetched it — never unconditionally in front. The anchor is the
        // window that produced the sent cursor, so a gap fill lands between
        // its neighbors even when every timestamp ties. Without a matching
        // window the oldest window keeps the previous prepend behavior.
        const anchor = keptPages.findIndex((page) => page.nextBefore === sentBefore);
        const slot = anchor >= 0 ? anchor : 0;
        const byId = new Map(kept.map((item) => [item.commentId, item] as const));
        for (const item of fresh) byId.set(item.commentId, item);
        const flat = [
          ...keptPages.slice(0, slot).flatMap((page) => page.ids),
          ...fresh.map((item) => item.commentId),
          ...keptPages.slice(slot).flatMap((page) => page.ids),
        ];
        const seen = new Set<string>();
        merged = sweepThreadPage(
          flat.flatMap((id) => {
            if (seen.has(id)) return [];
            seen.add(id);
            const row = byId.get(id);
            return row === undefined ? [] : [row];
          }),
          fresh
        );
        const survivors = new Set(merged.map((item) => item.commentId));
        pages = [
          ...keptPages.slice(0, slot),
          {ids: fresh.map((item) => item.commentId), nextBefore: result.next_before},
          ...keptPages.slice(slot),
        ]
          .map((page) => ({ids: page.ids.filter((id) => survivors.has(id)), nextBefore: page.nextBefore}))
          .filter((page) => page.ids.length > 0);
        // A complete disjoint page with a matching anchor is authoritative
        // below the anchor: the completed enumeration spans everything under
        // the sent cursor, so omitted cached rows in older windows are gone
        // (deleted/revoked) and purge. A truncated page leaves the genuinely
        // unserved tail alone, and an anchorless race proves nothing.
        const drop = coveredOlderOmissions(kept, keptPages, freshIds, sentBefore, result.next_before === 0);
        if (drop.size > 0) {
          merged = merged.filter((item) => !drop.has(item.commentId));
          const pruned = new Set(merged.map((item) => item.commentId));
          pages = pages
            .map((page) => ({ids: page.ids.filter((id) => pruned.has(id)), nextBefore: page.nextBefore}))
            .filter((page) => page.ids.length > 0);
        }
        next = result.next_before;
      } else {
        const spliced = spliceThreadPage(kept, keptPages, fresh, result.next_before);
        // An authoritative older page purges omitted cached rows its span
        // covers while the anchor window, newer rows, and the genuinely
        // unserved tail below a truncated page stay.
        const drop = coveredOlderOmissions(kept, keptPages, freshIds, sentBefore, result.next_before === 0);
        merged =
          drop.size > 0 ? spliced.merged.filter((item) => !drop.has(item.commentId)) : spliced.merged;
        if (drop.size > 0) {
          const survivors = new Set(merged.map((item) => item.commentId));
          pages = spliced.pages
            .map((page) => ({
              ids: page.ids.filter((id) => survivors.has(id)),
              nextBefore: page.nextBefore,
            }))
            .filter((page) => page.ids.length > 0);
        } else {
          pages = spliced.pages;
        }
        next = spliced.cursor;
      }
      // Explicit tombstone rows in this page are authoritative removals:
      // drop them from the merged cache even without coverage, retire any
      // parent outcome, and stamp them so a late GET or an overlapping
      // in-flight page cannot resurrect.
      if (explicitRemovedIds.size > 0) {
        merged = merged.filter((item) => !explicitRemovedIds.has(item.commentId));
        const pruned = new Set(merged.map((item) => item.commentId));
        pages = pages
          .map((page) => ({ids: page.ids.filter((id) => pruned.has(id)), nextBefore: page.nextBefore}))
          .filter((page) => page.ids.length > 0);
        for (const id of explicitRemovedIds) {
          this.stampAuthoritativeRemoval(id);
          for (const [outcomeKey, rows] of this.view.outcomes) {
            if (rows.some((row) => row.commentId === id)) this.removeOutcome(outcomeKey, id);
            removeIntakeRecord(this.storageKey(outcomeKey), id);
          }
        }
      }
      this.threads.set(key, {
        items: merged,
        pages,
        next,
        busy: false,
        message: '',
      });
      // This fresh page owns every served row's pipeline now: a delayed
      // Check progress GET still in flight must drop instead of
      // overwriting the newer revision or permission state. Only rows the
      // response actually observed are stamped — stamping merged cached
      // rows would invalidate an unrelated in-flight retry for a comment
      // this page never served.
      for (const item of fresh) this.stampCommentProgress(item.commentId);
      for (const id of explicitRemovedIds) this.stampCommentProgress(id);
      // An authoritative removal owns the dropped row too: every kept row
      // this merge purged is gone (deleted/revoked), so stamp it as well —
      // otherwise a late Check progress GET still in flight would land as
      // current and resurrect the row into parent outcomes.
      const survivors = new Set(merged.map((item) => item.commentId));
      for (const item of kept) if (!survivors.has(item.commentId)) this.stampCommentProgress(item.commentId);
    } catch (error) {
      if (serial !== this.serial) return;
      const denied = error instanceof Error && (error as {status?: number}).status === 403;
      // A failed page restores the live cache, not the pre-await snapshot,
      // so a recovery that landed mid-flight keeps its rows.
      const failed = this.threads.get(key);
      this.threads.set(key, {
        items: failed?.items ?? [],
        pages: failed?.pages ?? [],
        next: failed?.next ?? 0,
        busy: false,
        message: denied && error instanceof Error ? error.message : 'Replies could not be loaded. Try Refresh replies.',
      });
    } finally {
      if (serial === this.serial) this.readingThreads.delete(key);
    }
    this.requestUpdate();
  }

  private renderReplies(entry: FeedEntry) {
    if (!commentSourceForEntry(entry)) return html``;
    const key = this.entryKeyFor(entry),
      thread = this.threads.get(key);
    if (!thread) return html`<p role="status" class="soda-feed-thread-status">Loading replies…</p>`;
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
    const ref = this.view.labels.refs.get(entry.repository);
    return html`<li class="soda-feed-comment" id=${'comment-' + reply.commentId} data-role=${reply.actor ?? 'human'}>
      <span class="soda-feed-avatar" aria-hidden="true"
        >${
          feedRolePortraits[roleLabel(reply.actor) ?? '']
            ? html`<img
                src=${(this.current?.transport.assetBase ?? '') + feedRolePortraits[roleLabel(reply.actor) ?? '']}
                alt=""
                width="30"
                height="30"
              />`
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
        ${
          reply.body && (reply.contribution || reply.consultation)
            ? html`<p class="soda-feed-body-text">${reply.body}</p>`
            : ''
        }
        ${
          reply.contribution
            ? renderContribution(reply.contribution, {entryKey: this.entryKeyFor(entry)})
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
            ? renderDiscussion({
                key: 'post:' + reply.commentId,
                draft: '',
                submitAvailable: false,
                showComposer: false,
                intakeText: '',
                discussionLink: undefined,
                outcomes: [
                  describeCommentOutcome(
                    reply.pipeline,
                    {
                      id: entry.repository,
                      path: ref?.path,
                      label: ref?.label ?? 'This project',
                    },
                    this.view.commentBusy.includes(reply.commentId)
                  ),
                ],
              })
            : ''
        }
      </div>
    </li>`;
  }
}

customElements.define('soda-factory-feed', SodaFactoryFeed);
