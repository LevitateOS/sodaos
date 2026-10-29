import {check, object} from './sodaspaces-api.js';

// Combined feed wire contract. The server owns identity, visibility,
// cursors and every bound on the JSON it sends; the browser reads only
// what each view needs. Field names arrive in snake_case and are mapped
// to camelCase once by `camelize`; the interfaces below mirror the Go
// JSON (internal/factory/feed_contract.go), where omitted lists, totals
// and strings stay absent.

export type FeedEntryKind = 'record' | 'post';
export const feedEntryKinds: readonly FeedEntryKind[] = ['record', 'post'];

export type FeedCommunicationKind =
  | 'ask'
  | 'answer'
  | 'reply'
  | 'flag'
  | 'learned'
  | 'suggest'
  | 'partial'
  | 'blocked'
  | 'stuck'
  | 'record_decision'
  | 'finish_incomplete'
  | 'system'
  | 'library_maintenance'
  | 'artifact'
  | 'journey'
  | 'diagnosis'
  | 'repair_draft';
export type FeedActivity = 'merged' | 'completed' | 'judgment' | FeedCommunicationKind;

export interface FeedEntry {
  kind: FeedEntryKind;
  id: string;
  repository: number;
  seq: number;
  actor?: string;
  author?: string;
  project?: string;
  at?: number;
  status?: string;
  issue?: number;
  pr?: number;
  ref?: string;
  summary?: string;
  activity?: FeedActivity;
  assignment?: string;
  eventIdentity?: string;
  payload?: FeedPayload;
  // Human posts carry their stored comment and pipeline outcome, so one
  // feed read renders the post without a per-post request.
  comment?: CommentDetail;
}

export interface FeedPlanTask {
  id: string;
  objective?: string;
  lane?: string;
  dependsOn?: string[];
}

export interface FeedInspected {
  path: string;
  commit?: string;
  evidence?: string;
}

export interface FeedPlan {
  approach?: string;
  findings?: string[];
  unresolved?: string[];
  inspected?: FeedInspected[];
  inspectedTotal?: number;
  tasks?: FeedPlanTask[];
}

export interface FeedTaskStates {
  tasks?: {id: string; state?: string}[];
}

export interface FeedImplementation {
  candidate?: string;
}

export interface FeedFinding {
  key: string;
  severity?: string;
  defect?: string;
  evidence?: string;
  requiredOutcome?: string;
  location?: {path: string; side?: string; line?: number};
}

export interface FeedAssessment {
  key: string;
  status?: string;
  evidence?: string;
}

export interface FeedReview {
  verdict?: string;
  summary?: string;
  findings?: string[];
  newFindings?: FeedFinding[];
  newFindingsTotal?: number;
  clear?: {head?: string; assessments?: FeedAssessment[]};
}

export interface FeedCorrection {
  newCandidate?: string;
  findings?: FeedFinding[];
  dispositions?: {key: string; disposition?: string; changedPaths?: string[]; commits?: string[]}[];
}

export interface FeedIntentTask {
  key: string;
  objective?: string;
  dependsOn?: string[];
}

export interface FeedGraphDelta {
  summary?: string;
  outcome?: string;
  retainedIssues?: number[];
  changes?: {reason?: string; tasks?: FeedIntentTask[]}[];
  changesTotal?: number;
  attention?: {issue?: number; reason?: string}[];
}

export interface FeedCheckFailure {
  check?: string;
  detail?: string;
}

export interface FeedVerification {
  outcome?: string;
  checks?: {id: string; outcome?: string; artifactDigest?: string}[];
  checksTotal?: number;
  failures?: FeedCheckFailure[];
}

export interface FeedDecomposition {
  attention?: string;
  candidates?: {key?: string; rationale?: string; tasks?: FeedIntentTask[]}[];
}

export interface FeedJudgmentCandidate {
  id: string;
  issue?: number;
  objective?: string;
}

export interface FeedJudgment {
  kind: 'priority' | 'delivery';
  selected?: string;
  outcome?: string;
  effective?: string;
  probability?: number;
  confidence?: number;
  probabilities?: Record<string, number>;
  candidates?: FeedJudgmentCandidate[];
}

export interface FeedMerge {
  mergedCommit?: string;
  originalClosedUnix?: number;
}

export interface FeedArtifact {
  id: string;
  name: string;
  mediaType: string;
  href: string;
}

export interface FeedGraphItem {
  id?: string;
  label: string;
  detail?: string;
  state?: string;
  dependsOn?: string[];
}

export interface FeedSourceExcerpt {
  path: string;
  revision?: string;
  startLine: number;
  code: string;
  after?: string;
  highlightLines?: number[];
}

export interface FeedCommunication {
  kind: FeedCommunicationKind;
  // A controller-written Factory event's kind; worker reports carry none.
  componentId?: string;
  title: string;
  summary: string;
  question?: string;
  questionState?: 'open' | 'answered' | 'replied' | 'incorporated';
  disposition?: 'applied' | 'refused' | 'incorporated';
  dispositionReason?: string;
  resolution?: string;
  scope?: string;
  severity?: 'info' | 'warning' | 'critical';
  evidenceRefs?: string[];
  options?: string[];
  completed?: string[];
  remaining?: string[];
  attempts?: string[];
  decision?: string;
  rationale?: string;
  alternatives?: string[];
  replyTo?: string;
  items?: FeedGraphItem[];
  artifacts?: FeedArtifact[];
  commits?: {revision: string; summary: string}[];
  source?: FeedSourceExcerpt;
  libraryItemId?: string;
  libraryRevision?: number;
  before?: string;
  after?: string;
  intentSource?: {intent: string; revision: string; predecessor?: string; state: string; owner?: string};
  planSource?: {attemptRoot: string; plan: string; digest: string; sourceCommit: string; generation?: number};
}

// FeedPayload is one record's typed result. At most one family is set;
// `tasks` may accompany an implementation or a correction.
export interface FeedPayload {
  communication?: FeedCommunication;
  plan?: FeedPlan;
  tasks?: FeedTaskStates;
  implementation?: FeedImplementation;
  review?: FeedReview;
  correction?: FeedCorrection;
  reassessment?: FeedGraphDelta;
  repair?: FeedGraphDelta;
  verification?: FeedVerification;
  decomposition?: FeedDecomposition;
  judgment?: FeedJudgment;
  merge?: FeedMerge;
}

export interface FeedPageView {
  entries: FeedEntry[];
  next: string;
  truncated: boolean;
}

export interface FeedSubscriptionView {
  subscriber: string;
  repositories: number[];
  kinds: FeedEntryKind[];
  revision: number;
}

export function feedEntryKey(entry: FeedEntry): string {
  return `${entry.kind}:${entry.id}`;
}

// Map keys that are data (option identities), not field names.
const opaqueKeys = new Set(['probabilities', 'consultation', 'contribution']);

// camelize copies one server JSON value with its snake_case field names in
// camelCase. Values are not checked: the server already bounds them.
export function camelize(value: unknown): unknown {
  if (Array.isArray(value)) return value.map(camelize);
  if (value === null || typeof value !== 'object') return value;
  const out: Record<string, unknown> = {};
  for (const [key, field] of Object.entries(value)) {
    out[key.replace(/_([a-z0-9])/g, (_, letter: string) => letter.toUpperCase())] = opaqueKeys.has(key)
      ? field
      : camelize(field);
  }
  return out;
}

// parseFeedPageView reads one served window. An entry that cannot even be
// identified is skipped; it never blanks the rest of the page.
export function parseFeedPageView(value: unknown): FeedPageView {
  const data = object(value);
  check(Array.isArray(data.entries));
  const entries = data.entries.flatMap((raw): FeedEntry[] => {
    try {
      const entry = object(raw);
      check(typeof entry.id === 'string' && typeof entry.repository === 'number' && typeof entry.seq === 'number');
      const parsed = camelize({...entry, comment: undefined}) as FeedEntry;
      if (entry.comment !== undefined) parsed.comment = parseCommentDetail(entry.comment);
      else delete parsed.comment;
      return [parsed];
    } catch {
      return [];
    }
  });
  return {entries, next: typeof data.next === 'string' ? data.next : '', truncated: data.truncated === true};
}

export function parseFeedSubscriptionView(value: unknown): FeedSubscriptionView {
  const data = object(value);
  check(typeof data.subscriber === 'string' && typeof data.revision === 'number');
  return {
    subscriber: data.subscriber,
    repositories: Array.isArray(data.repositories) ? (data.repositories as number[]) : [],
    kinds: Array.isArray(data.kinds) ? (data.kinds as FeedEntryKind[]) : [],
    revision: data.revision,
  };
}

export interface FeedQueryOptions {
  repositories: number[];
  kinds: FeedEntryKind[];
  limit: number;
  cursor: string;
  view?: string;
  search?: string;
  role?: string;
  current?: boolean;
}

// feedQueryString renders one feed query: comma-separated repository
// scope, optional kinds, a bounded window and the resume cursor. An empty
// cursor reads from the latest.
export function feedQueryString(query: FeedQueryOptions): string {
  const params = new URLSearchParams();
  params.set('repositories', [...new Set(query.repositories)].sort((a, b) => a - b).join(','));
  if (query.kinds.length > 0) params.set('kinds', query.kinds.join(','));
  params.set('limit', String(query.limit));
  if (query.cursor !== '') params.set('cursor', query.cursor);
  if (query.view && query.view !== 'all') params.set('view', query.view);
  if (query.search) params.set('search', query.search);
  if (query.role) params.set('role', query.role);
  if (query.current) params.set('current', 'true');
  return `factory/feed?${params.toString()}`;
}

// CommentSource is the server-resolved snapshot of the feed entry one
// comment was submitted against.
export interface CommentSource {
  kind: FeedEntryKind;
  id: string;
  issue?: number;
  pr?: number;
  ref?: string;
  status?: string;
  summary?: string;
  product?: {intent: string; acceptedRevision: string};
}

export interface CommentProductChange {
  intent: string;
  acceptedRevision: string;
  draftRevision?: string;
  state: 'review_required' | 'draft' | 'accepted' | 'superseded' | 'unavailable';
  canReview: boolean;
}

// CommentPipelineState is one stored comment request: its pipeline states,
// the linked native issue and the author's retry right.
export interface CommentPipelineState {
  commentId: string;
  revision: number;
  intake: string;
  judgment: string;
  route?: string;
  draft: string;
  effect: string;
  issue?: number;
  existingIssue?: number;
  detail?: string;
  canRetry?: boolean;
  attempts?: number;
  attemptLimit?: number;
  source?: CommentSource;
  productChange?: CommentProductChange;
  /** JEV's routing Choice: each option weighed, the pick and her confidence. */
  routing?: CommentRouting;
  /** The collaborator the next stage waits for: jev, product_lead or forge. */
  stalled?: string;
}

export interface CommentRouting {
  selected: string;
  confidence: number;
  options: {key: string; label: string; probability: number}[];
}

// commentSourceForEntry names the feed entry one reply answers: a post by
// its comment identity, a record by its assignment identity.
export function commentSourceForEntry(entry: FeedEntry): {kind: FeedEntryKind; id: string} | undefined {
  const id = entry.kind === 'post' ? entry.id : (entry.assignment ?? entry.id);
  return /^[a-f0-9]{32}$/.test(id) ? {kind: entry.kind, id} : undefined;
}

export interface CommentPostBody {
  repository: number;
  body: string;
  issue?: number;
  reply_to?: string;
  source_kind?: FeedEntryKind;
  source_id?: string;
  idempotency_key: string;
}

export interface FeedContribution {
  source: string;
  revision: number;
  state: string;
  profile?: string;
  reply?: string;
  body: string;
  context?: string;
  roles: unknown[];
  replies: unknown[];
  calls?: {
    assignment: string;
    question: string;
    answer: {selected: string; probabilities: Record<string, number>; confidence: number; model: string};
  }[];
  minimum_probability: number;
  minimum_confidence: number;
  probability_threshold?: number;
  confidence_threshold?: number;
  policy_version?: string;
}

export interface FeedConsultation {
  snapshot: string;
  group_id: string;
  assignment: string;
  source_revision?: string;
  tool: string;
  kind: string;
  state?: unknown;
  instructions?: unknown;
  criteria?: unknown;
  questions?: Record<string, unknown>;
  model?: string;
  status: 'pending' | 'returned' | 'failed' | 'unavailable' | 'uncertain';
  answer?: Record<string, unknown>;
  created_unix: number;
  updated_unix: number;
}

export interface CommentDetail {
  commentId: string;
  // Human comments carry their stored pipeline; worker posts do not.
  pipeline?: CommentPipelineState;
  body: string;
  authorName: string;
  at: number;
  edited: boolean;
  removed: boolean;
  actor?: string;
  communication?: FeedCommunication;
  consultation?: FeedConsultation;
  contribution?: FeedContribution;
}

// parseCommentDetail reads one comment as the server serves it (comment GET,
// replies page or an enriched feed post). Deleted or revoked sources keep
// their place without text.
export function parseCommentDetail(value: unknown): CommentDetail {
  const raw = object(value);
  check(typeof raw.comment_id === 'string');
  const data = camelize(raw) as Record<string, unknown> & CommentPipelineState;
  const removed = raw.deleted === true || raw.revoked === true;
  return {
    commentId: data.commentId,
    ...(raw.judgment !== undefined && !removed ? {pipeline: data} : {}),
    body: typeof raw.body === 'string' ? raw.body : '',
    authorName: typeof raw.author_name === 'string' ? raw.author_name : '',
    at: typeof raw.at === 'number' ? raw.at : 0,
    edited: raw.edited === true,
    removed,
    ...(typeof raw.actor === 'string' && raw.actor ? {actor: raw.actor} : {}),
    ...(raw.communication ? {communication: data.communication as FeedCommunication} : {}),
    ...(raw.consultation ? {consultation: raw.consultation as FeedConsultation} : {}),
    ...(raw.contribution ? {contribution: raw.contribution as FeedContribution} : {}),
  };
}

// newIdempotencyKey mints one caller-chosen key per draft, so a resubmit
// after a lost answer recovers the same comment.
export function newIdempotencyKey(): string {
  const bytes = new Uint8Array(16);
  crypto.getRandomValues(bytes);
  return [...bytes].map((byte) => byte.toString(16).padStart(2, '0')).join('');
}

export function commentAttemptsExhausted(state: CommentPipelineState): boolean {
  return state.attempts !== undefined && state.attemptLimit !== undefined && state.attempts >= state.attemptLimit;
}

// commentUnsettled reports whether one pipeline still advances on the
// server; the feed refresh picks up its next state.
export function commentUnsettled(state: CommentPipelineState): boolean {
  if (state.productChange || commentAttemptsExhausted(state)) return false;
  if (state.judgment === 'pending') return true;
  if (state.judgment === 'failed') return false;
  if (state.route === 'new_issue')
    return state.draft === 'pending' || (state.draft !== 'invalid' && state.effect === 'pending');
  return state.route === 'existing_work' && state.effect === 'pending';
}

// mergeFeedEntries folds one older page behind the shown entries,
// newest-first by global order key with each identity once.
export function mergeFeedEntries(confirmed: FeedEntry[], page: FeedEntry[]): FeedEntry[] {
  const byKey = new Map(confirmed.map((entry) => [feedEntryKey(entry), entry]));
  for (const entry of page) byKey.set(feedEntryKey(entry), entry);
  return [...byKey.values()].sort((a, b) => b.seq - a.seq || (a.id < b.id ? -1 : a.id > b.id ? 1 : 0));
}
