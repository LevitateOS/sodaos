import {feedEntryKey} from './sodaspaces-factory-feed-response.js';
import type {FeedCommunication, FeedEntry, FeedJudgment} from './sodaspaces-factory-feed-response.js';
import {factoryEvent} from './sodaspaces-factory-feed-communication.js';

// The Feed's three levels. Every entry lands in exactly one: a new post
// (something worth seeing by itself), a live update (running work whose post
// changes in place) or a one-line comment under the post it belongs to.
// Comments that change the outcome or need the founder are promoted back to
// posts and link to the thread they came from. Nothing is dropped: a comment
// with no post to belong to stays a post.

const roleLabels: Readonly<Record<string, string>> = {
  product_lead: 'Product Lead',
  architect: 'Architect',
  builder: 'Builder',
  inspector: 'Inspector',
  fixer: 'Fixer',
  strategist: 'Strategist',
  test_pilot: 'Test Pilot',
  jev: 'JEV',
  systemone: 'JEV',
  system_one: 'JEV',
  factory: 'Factory',
  system: 'Factory',
};

function roleKey(actor: string | undefined): string {
  return (actor ?? '')
    .trim()
    .toLowerCase()
    .replace(/[\s-]+/g, '_');
}

// roleLabel renders a canonical actor id readably; free-form server labels
// pass through unchanged.
export function roleLabel(actor: string | undefined): string | undefined {
  if (actor === undefined) return undefined;
  return roleLabels[roleKey(actor)] ?? actor;
}

export type FeedState = 'needs_you' | 'problem' | 'waiting' | 'in_progress' | 'done' | 'decided' | 'info';
export type FeedLevel = 'post' | 'update' | 'comment';

export const feedStateWords: Readonly<Record<FeedState, string>> = {
  needs_you: 'Needs you',
  problem: 'Problem',
  waiting: 'Waiting',
  in_progress: 'In progress',
  done: 'Done',
  decided: 'Decided',
  info: 'For your information',
};

// What SodaOS did with a delivery judgment. Ready merges on its own, review
// again starts another review, everything else waits for the founder.
function deliveryState(judgment: FeedJudgment): FeedState {
  const effective = judgment.effective ?? '';
  if (effective === 'ready_to_merge') return 'done';
  if (effective === 'review_again') return 'in_progress';
  if (effective === '') return judgment.outcome ? 'decided' : 'info';
  return 'needs_you';
}

function communicationState(communication: FeedCommunication, entry: FeedEntry): FeedState {
  if (communication.disposition !== undefined) return 'decided';
  if (communication.kind !== 'ask' && (communication.severity === 'critical' || communication.severity === 'warning'))
    return 'problem';
  switch (communication.kind) {
    case 'ask':
      return communication.questionState === 'open' || communication.questionState === undefined
        ? 'needs_you'
        : 'decided';
    case 'blocked':
      return 'waiting';
    case 'stuck':
    case 'finish_incomplete':
      return 'problem';
    case 'flag':
      return communication.severity === 'info' ? 'info' : 'problem';
    case 'partial':
      return 'in_progress';
    case 'record_decision':
    case 'answer':
      return 'decided';
    case 'system':
      return statusState(entry);
    default:
      return 'info';
  }
}

// statusState reads the recorded status words of one record or event. An
// unknown status is information, never success.
function statusState(entry: FeedEntry): FeedState {
  const status = (entry.status ?? '').toLowerCase();
  if (/needs-human|clarify/.test(status)) return 'needs_you';
  if (/paused|pending|queued/.test(status)) return 'waiting';
  if (
    /fail|error|refus|reject|invalid|blocked|stuck|unconfirmed|cancel|stopped|request-changes|exhaust|denied|conflict/.test(
      status
    )
  )
    return 'problem';
  if (status === 'run:succeeded') return 'info';
  if (/judgment|decid|declined|route:/.test(status)) return 'decided';
  if (/finished:succeeded|completed|merged|verified|applied|accepted|delivered|created|valid/.test(status))
    return 'done';
  if (/running|assigned|assess|progress|working|submitted|draft|:/.test(status)) return 'in_progress';
  return 'info';
}

// feedState gives one entry exactly one of the seven states. It reads the
// recorded facts only; an unknown status is information, never success.
export function feedState(entry: FeedEntry): FeedState {
  const payload = entry.payload;
  if (payload?.communication) return communicationState(payload.communication, entry);
  if (payload?.judgment?.kind === 'delivery') return deliveryState(payload.judgment);
  if (entry.activity === 'judgment' || payload?.judgment) return 'decided';
  if (entry.activity === 'merged' || entry.activity === 'completed') return 'done';
  if (entry.kind === 'post') return 'info';
  return statusState(entry);
}

// feedLevel applies the post-or-comment rule. Stage starts and ends, results,
// questions, decisions and failures are posts; running progress is a live
// update; small steps inside work that already has a post are comments.
export function feedLevel(entry: FeedEntry): FeedLevel {
  const communication = entry.payload?.communication;
  if (entry.kind === 'post' && !communication) return 'post';
  const event = factoryEvent(communication?.componentId);
  if (event !== undefined) return event.level;
  // A priority judgment explains why work is next; a delivery assessment is a post.
  if (entry.payload?.judgment?.kind === 'priority') return 'comment';
  if (communication) {
    switch (communication.kind) {
      case 'partial':
        return 'update';
      case 'reply':
      case 'answer':
      case 'artifact':
      case 'journey':
      case 'learned':
      case 'suggest':
      case 'diagnosis':
      case 'library_maintenance':
        return 'comment';
      case 'flag':
        return communication.severity === 'info' ? 'comment' : 'post';
      default:
        return 'post';
    }
  }
  if (entry.status === 'run:succeeded') return 'comment';
  return 'post';
}

export type CommentKind =
  | 'edit'
  | 'read'
  | 'pass'
  | 'fail'
  | 'save'
  | 'jev'
  | 'reply'
  | 'know'
  | 'shot'
  | 'link'
  | 'text';
export type CommentTone = 'ok' | 'bad' | 'muted' | 'plain';

export interface FeedCommentLine {
  readonly key: string;
  readonly kind: CommentKind;
  readonly actor: string | undefined;
  readonly who: string;
  // A designed line reads who · did what · to what · result. A plain line
  // (kind text) carries only its text.
  readonly did: string;
  readonly what: string;
  readonly result: string;
  readonly tone: CommentTone;
  readonly at: number | undefined;
  // The post this comment became, when it was promoted.
  readonly promoted?: string;
}

export interface FeedCommentGroup {
  readonly key: string;
  readonly group: true;
  readonly kind: CommentKind;
  readonly actor: string | undefined;
  readonly who: string;
  readonly summary: string;
  readonly result: string;
  readonly tone: CommentTone;
  readonly from: number | undefined;
  readonly to: number | undefined;
  readonly items: readonly FeedCommentLine[];
}

export type FeedThreadComment = FeedCommentLine | FeedCommentGroup;

export interface FeedThreadItem {
  readonly entry: FeedEntry;
  readonly state: FeedState;
  readonly level: FeedLevel;
  readonly comments: readonly FeedThreadComment[];
  readonly commentCount: number;
  // Older states of a live update, newest first.
  readonly changes: readonly FeedEntry[];
  // The thread a promoted comment came from.
  readonly from?: {key: string; label: string};
}

export function whoOf(entry: FeedEntry): string {
  if (entry.author) return entry.author;
  return roleLabel(entry.actor && !/^\d+$/.test(entry.actor) ? entry.actor : undefined) ?? 'SodaOS';
}

function clip(text: string, limit = 72): string {
  const line = text.replace(/\s+/g, ' ').trim();
  return line.length > limit ? `${line.slice(0, limit - 1).trimEnd()}…` : line;
}

function plain(entry: FeedEntry, text: string): FeedCommentLine {
  return {
    key: feedEntryKey(entry),
    kind: 'text',
    actor: entry.actor,
    who: whoOf(entry),
    did: '',
    what: clip(text, 160),
    result: '',
    tone: 'plain',
    at: entry.at,
  };
}

// The icon a Factory event's comment carries, by its presentation.
const presentationKinds: Readonly<Record<string, CommentKind>> = {
  comparison: 'edit',
  artifact: 'shot',
  result: 'save',
  knowledge: 'know',
  evidence: 'read',
  diagnosis: 'read',
  graph: 'read',
  checks: 'pass',
  journey: 'pass',
  finding: 'fail',
  proposal: 'reply',
  discussion: 'reply',
  brief: 'text',
  decision: 'jev',
};

// commentLine turns one comment-level entry into its single line. Kinds
// without their own design keep the plain-text fallback: who and one line.
export function commentLine(entry: FeedEntry): FeedCommentLine {
  const who = whoOf(entry);
  const base = {key: feedEntryKey(entry), actor: entry.actor, who, at: entry.at};
  const communication = entry.payload?.communication;
  const title = communication ? clip(communication.title || communication.summary) : '';
  const presentation = factoryEvent(communication?.componentId)?.presentation;
  if (communication && presentation && presentationKinds[presentation]) {
    const flagged = communication.severity === 'critical' || communication.severity === 'warning';
    return {
      ...base,
      kind: flagged ? 'fail' : presentationKinds[presentation]!,
      did: '',
      what: title,
      result: flagged ? 'needs attention' : '',
      tone: flagged ? 'bad' : 'muted',
    };
  }
  if (communication) {
    switch (communication.kind) {
      case 'artifact': {
        const count = communication.artifacts?.length ?? 0;
        return {
          ...base,
          kind: 'shot',
          did: 'shared',
          what: title,
          result: count > 1 ? `${count} files` : '',
          tone: 'muted',
        };
      }
      case 'journey':
        return {...base, kind: 'pass', did: 'checked', what: title, result: 'observed', tone: 'muted'};
      case 'learned':
        return {...base, kind: 'know', did: 'saved a lesson', what: title, result: 'in Knowledge', tone: 'muted'};
      case 'suggest':
        return {...base, kind: 'reply', did: 'suggested', what: title, result: 'proposal', tone: 'muted'};
      case 'diagnosis':
        return {...base, kind: 'read', did: 'diagnosed', what: title, result: '', tone: 'muted'};
      case 'library_maintenance':
        return {...base, kind: 'know', did: 'updated Knowledge', what: title, result: '', tone: 'muted'};
      case 'answer':
        return {...base, kind: 'reply', did: 'answered', what: title, result: '', tone: 'muted'};
      case 'flag':
        return {...base, kind: 'read', did: 'noted', what: title, result: '', tone: 'muted'};
      default:
        return plain(entry, communication.summary || communication.title);
    }
  }
  if (entry.status === 'run:succeeded')
    return {
      ...base,
      kind: 'pass',
      did: 'finished a run',
      what: entry.issue ? `on issue #${entry.issue}` : '',
      result: 'run ended',
      tone: 'muted',
    };
  if (entry.status === 'run:failed')
    return {
      ...base,
      kind: 'fail',
      did: 'ran',
      what: entry.issue ? `issue #${entry.issue}` : 'its work',
      result: 'failed',
      tone: 'bad',
    };
  return plain(entry, entry.summary ?? entry.status ?? 'Recorded a step.');
}

// The thread a step belongs to: the same assignment first, then the same
// pull request, then the same issue.
function threadKeys(entry: FeedEntry): string[] {
  const keys: string[] = [];
  if (entry.assignment) keys.push(`a:${entry.repository}:${entry.assignment}`);
  if (entry.kind === 'record' && entry.activity === undefined && entry.payload?.communication === undefined)
    keys.push(`a:${entry.repository}:${entry.id}`);
  if (entry.pr) keys.push(`p:${entry.repository}:${entry.pr}`);
  if (entry.issue) keys.push(`i:${entry.repository}:${entry.issue}`);
  return keys;
}

function headlineLabel(entry: FeedEntry): string {
  const who = whoOf(entry);
  const work = entry.pr ? `pull request #${entry.pr}` : entry.issue ? `issue #${entry.issue}` : 'this work';
  return `${who === 'SodaOS' ? "SodaOS's" : `the ${who}'s`} post on ${work}`;
}

function liveKey(entry: FeedEntry): string | undefined {
  const scope = entry.assignment ?? (entry.pr ? `pr${entry.pr}` : entry.issue ? `issue${entry.issue}` : undefined);
  if (scope === undefined) return undefined;
  const component = entry.payload?.communication?.componentId ?? 'progress';
  return `${entry.repository}:${normalizedActor(entry.actor)}:${scope}:${component}`;
}

function normalizedActor(actor: string | undefined): string {
  return (actor ?? '').trim().toLowerCase();
}

const groupWords: Readonly<Record<CommentKind, [string, string]>> = {
  edit: ['made', 'change'],
  read: ['read', 'source'],
  pass: ['ran', 'check'],
  fail: ['hit', 'problem'],
  save: ['saved', 'checkpoint'],
  jev: ['recorded', 'decision'],
  reply: ['posted', 'reply'],
  know: ['saved', 'lesson'],
  shot: ['shared', 'preview'],
  link: ['linked', 'post'],
  text: ['posted', 'note'],
};

// "made 2 changes · ran 3 checks": one phrase per kind, in first-seen order.
function groupSummary(lines: readonly FeedCommentLine[]): string {
  const counts = new Map<CommentKind, number>();
  for (const line of lines) counts.set(line.kind, (counts.get(line.kind) ?? 0) + 1);
  return [...counts]
    .map(([kind, count]) => {
      const [verb, noun] = groupWords[kind];
      return `${verb} ${count} ${noun}${count === 1 ? '' : noun.endsWith('y') ? '' : 's'}`.replace(
        /replys$/,
        'replies'
      );
    })
    .join(' · ');
}

// Three or more steps in a row by the same author fold into one line that
// opens to show each step.
function groupLines(lines: readonly FeedCommentLine[]): FeedThreadComment[] {
  const out: FeedThreadComment[] = [];
  let run: FeedCommentLine[] = [];
  const flush = () => {
    if (run.length >= 3) {
      const first = run[0]!;
      const failed = run.filter((line) => line.tone === 'bad').length;
      out.push({
        key: `group:${first.key}`,
        group: true,
        kind: first.kind,
        actor: first.actor,
        who: first.who,
        summary: groupSummary(run),
        result: failed > 0 ? `${failed} failed` : run.every((line) => line.kind === 'pass') ? 'all passed' : '',
        tone: failed > 0 ? 'bad' : run.every((line) => line.kind === 'pass') ? 'ok' : 'muted',
        from: run[0]!.at,
        to: run[run.length - 1]!.at,
        items: run,
      });
    } else out.push(...run);
    run = [];
  };
  for (const line of lines) {
    const last = run[run.length - 1];
    if (last && (last.who !== line.who || line.promoted)) flush();
    run.push(line);
    if (line.promoted) flush();
  }
  flush();
  return out;
}

// threadFeed arranges one newest-first page into the Feed's three levels.
// Comments join the nearest post that shares their assignment, pull request
// or issue; a comment with no such post on the page stays a post.
export function threadFeed(entries: readonly FeedEntry[]): FeedThreadItem[] {
  const posts = new Map<
    string,
    {entry: FeedEntry; lines: FeedCommentLine[]; changes: FeedEntry[]; from?: {key: string; label: string}}
  >();
  const order: string[] = [];
  const owners = new Map<string, string>();
  const live = new Map<string, string>();
  // Oldest first, so a thread exists before its steps arrive and the newest
  // live state wins.
  const oldestFirst = [...entries].reverse();
  const postFor = (entry: FeedEntry) => {
    for (const key of threadKeys(entry)) {
      const owner = owners.get(key);
      if (owner !== undefined && owner !== feedEntryKey(entry)) return owner;
    }
    return undefined;
  };
  const addPost = (entry: FeedEntry, from?: {key: string; label: string}) => {
    const key = feedEntryKey(entry);
    posts.set(key, {entry, lines: [], changes: [], ...(from ? {from} : {})});
    order.push(key);
    for (const threadKey of threadKeys(entry)) if (!owners.has(threadKey)) owners.set(threadKey, key);
  };
  for (const entry of oldestFirst) {
    const level = feedLevel(entry);
    const state = feedState(entry);
    if (level === 'update') {
      const scope = liveKey(entry);
      const previous = scope !== undefined ? live.get(scope) : undefined;
      const held = previous !== undefined ? posts.get(previous) : undefined;
      if (held && previous !== undefined) {
        // The live post takes the newer state in place; the older state moves
        // to its change history and the thread stays with it.
        const index = order.indexOf(previous);
        const key = feedEntryKey(entry);
        posts.delete(previous);
        posts.set(key, {...held, entry, changes: [held.entry, ...held.changes]});
        order.splice(index, 1, key);
        live.set(scope!, key);
        for (const [threadKey, owner] of owners) if (owner === previous) owners.set(threadKey, key);
        continue;
      }
      addPost(entry);
      if (scope !== undefined) live.set(scope, feedEntryKey(entry));
      continue;
    }
    if (level === 'comment') {
      const parent = postFor(entry);
      if (parent !== undefined) {
        const line = commentLine(entry);
        if (state === 'problem' || state === 'needs_you') {
          // It changes the outcome: it becomes its own post, and its thread
          // keeps a line that links to it.
          const holder = posts.get(parent)!;
          holder.lines.push({...line, promoted: feedEntryKey(entry)});
          addPost(entry, {key: parent, label: headlineLabel(holder.entry)});
        } else posts.get(parent)!.lines.push(line);
        continue;
      }
    }
    addPost(entry);
  }
  return order
    .map((key) => posts.get(key)!)
    .reverse()
    .map((held) => ({
      entry: held.entry,
      state: feedState(held.entry),
      level: feedLevel(held.entry),
      comments: groupLines(held.lines),
      commentCount: held.lines.length,
      changes: held.changes,
      ...(held.from ? {from: held.from} : {}),
    }));
}

export interface PostLine {
  readonly headline: string;
  readonly detail: string;
  readonly next: string;
  readonly workItem: string | undefined;
}

function percent(value: number | undefined): string {
  return value === undefined ? '' : ` (${Math.round(value * 100)}%)`;
}

const roleVerbs: Readonly<Record<string, [string, string]>> = {
  product_lead: ['planning', 'planned'],
  architect: ['investigating', 'planned'],
  builder: ['building', 'built'],
  inspector: ['reviewing', 'reviewed'],
  fixer: ['fixing', 'fixed'],
  strategist: ['reassessing', 'reassessed'],
  test_pilot: ['testing', 'tested'],
  factory: ['delivering', 'delivered'],
  system: ['delivering', 'delivered'],
};

function judgmentHeadline(judgment: FeedJudgment, entry: FeedEntry): string {
  const p = percent(judgment.probability);
  if (judgment.kind === 'delivery') {
    const pr = entry.pr ? `pull request #${entry.pr}` : 'the pull request';
    switch (judgment.effective) {
      case 'ready_to_merge':
        return `JEV said ${pr} is ready to merge${p}, so SodaOS merges it`;
      case 'review_again':
        return `JEV said ${pr} needs another review${p}`;
      case 'review_allowance_exhausted':
        return `JEV says ${pr} needs another review, but the reviews are used up`;
      case 'judgment_attention':
        return `JEV wasn't sure enough about ${pr}${p}, so it waits for you`;
      default:
        return `JEV gave no usable answer on ${pr}, so it waits for you`;
    }
  }
  const top = (judgment.candidates ?? []).find((candidate) => candidate.id === judgment.selected);
  const name = top?.issue
    ? `issue #${top.issue}`
    : top?.objective
      ? clip(top.objective, 60)
      : (judgment.selected ?? 'one option');
  return `JEV put ${name} first${p}, so it runs next`;
}

// describePostLine writes the post's one-sentence headline from the
// recorded facts: who did what, and how it turned out.
export function describePostLine(entry: FeedEntry, state: FeedState): PostLine {
  const who = whoOf(entry);
  const target = entry.pr ? `pull request #${entry.pr}` : entry.issue ? `issue #${entry.issue}` : 'the project';
  const workItem = entry.pr ? `Pull request #${entry.pr}` : entry.issue ? `Issue #${entry.issue}` : undefined;
  const communication = entry.payload?.communication;
  const judgment = entry.payload?.judgment;
  const fallback = entry.kind === 'post' ? 'Comment' : `${who} recorded an update`;
  let headline = fallback;
  // A recorded summary stays in the post's own section, so it is not repeated
  // here; the detail line carries a rule or consequence only.
  let detail = '';
  if (communication) {
    // The communication's own summary follows in its section.
    headline = clip(communication.title || communication.summary || fallback, 110);
    detail = '';
  } else if (judgment) {
    headline = judgmentHeadline(judgment, entry);
    detail =
      judgment.kind === 'delivery'
        ? "SodaOS merges on its own when JEV's answer reaches 80% and JEV is at least 80% sure. Otherwise it waits for you."
        : 'SodaOS runs the top pick next. The rest wait their turn.';
  } else if (entry.activity === 'merged') {
    headline = `SodaOS merged ${entry.pr ? `pull request #${entry.pr}` : 'the pull request'}${entry.payload?.merge?.originalClosedUnix && entry.issue ? ` and closed issue #${entry.issue}` : ''}`;
  } else if (entry.activity === 'completed') {
    headline = `SodaOS completed ${entry.issue ? `issue #${entry.issue}` : 'the issue'}`;
  } else if (entry.kind === 'record') {
    const verbs = roleVerbs[roleKey(entry.actor)];
    const reviewer = roleKey(entry.actor) === 'inspector';
    const status = entry.status ?? '';
    const verdict = /verdict:([a-z_-]+)/.exec(status)?.[1] ?? entry.payload?.review?.verdict ?? '';
    if (reviewer && /request|changes|fail/.test(verdict)) headline = `${who} asked for changes on ${target}`;
    else if (reviewer && /approve|pass|clear/.test(verdict)) headline = `${who} approved ${target}`;
    else if (status === 'dispatch:assigned') headline = `${who} started on ${target}`;
    else if (status === 'run:succeeded') headline = `${who}'s run on ${target} ended`;
    else if (verbs && state === 'in_progress') headline = `${who} is ${verbs[0]} ${target}`;
    else if (verbs && state === 'done') headline = `${who} ${verbs[1]} ${target}`;
    else if (verbs && state === 'problem') headline = `${who} hit a problem ${verbs[0]} ${target}`;
    else if (verbs && state === 'waiting') headline = `${who} is waiting to start on ${target}`;
  }
  const nextFor: Readonly<Record<FeedState, string>> = {
    needs_you: 'Waits for you',
    problem: 'Needs attention before dependent work continues',
    waiting: 'Starts by itself when what it waits for is ready',
    in_progress: 'This post updates as the work moves',
    done: 'Nothing needed from you',
    decided: 'Nothing needed from you',
    info: 'Nothing needed from you',
  };
  return {headline, detail, next: nextFor[state], workItem};
}
