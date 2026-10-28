import {html, svg} from 'lit';
import {live} from 'lit/directives/live.js';
import type {TemplateResult} from 'lit';
import type {CommentPipelineState, FeedEntry} from './sodaspaces-factory-feed-response.js';
import {
  commentAttemptsExhausted,
  commentProductLaneWithoutProjection,
  feedEntryKey,
} from './sodaspaces-factory-feed-response.js';
import {nativePath} from './sodaspaces-factory-activity.js';
import {nativeIssueLink} from './sodaspaces-factory-work-view.js';
import {
  classifyFeedStatus,
  describeFeedFamily,
  renderFeedFamily,
  roleLabel,
} from './sodaspaces-factory-feed-families.js';
import type {FeedFamilyModel} from './sodaspaces-factory-feed-families.js';
import {feedRolePortraits} from './sodaspaces-factory-feed-portraits.js';
import {describePostLine, feedState, feedStateWords} from './sodaspaces-factory-feed-thread.js';
import type {
  CommentKind,
  FeedCommentGroup,
  FeedCommentLine,
  FeedState,
  FeedThreadComment,
  FeedThreadItem,
} from './sodaspaces-factory-feed-thread.js';
import type {FeedJudgment} from './sodaspaces-factory-feed-response.js';

// Pure combined-feed presentation: the shared post shell, one registered
// family section per entry (plans, issues, work, reviews, corrections,
// reassessments, verifications, integration/delivery, knowledge,
// decisions, questions, discussion) and the shared fallback/discussion
// sections. Model builders turn entry facts into display state; render
// helpers project those models to Lit templates. No transport, identity
// lookup or mutable state lives here.
//
// The wire carries kind/id/repository/seq plus optional server-derived
// actor/author/project/at/status enrichment and additive record
// content (issue/pr/ref/summary, rendered in the family sections).
// The models below render exactly those facts and say so, instead of
// inventing authors, dates, results or HTML.

export interface FeedRepositoryRef {
  readonly id: number;
  readonly path: string | undefined;
  readonly label: string;
}

export function repositoryRef(
  id: number,
  path: string | undefined,
  owner: string | undefined,
  name: string | undefined
): FeedRepositoryRef {
  const label = owner !== undefined && name !== undefined ? `${owner}/${name}` : `repository ${id}`;
  return {id, path: path !== undefined && nativePath(path) ? path : undefined, label};
}

export interface PostHeaderModel {
  readonly key: string;
  readonly avatarInitial: string;
  readonly avatarFile: string | undefined;
  readonly name: string;
  readonly role: string;
  readonly project: string;
  readonly timeText: string;
  readonly statusText: string;
  // Milestone binding line (Lane B3): merge, completion and judgment
  // posts name their milestone and the assignment they bind to.
  readonly milestoneText: string | undefined;
}

function shortFeedID(id: string): string {
  return `${id.slice(0, 12)}${id.length > 12 ? '…' : ''}`;
}

export function milestoneText(entry: FeedEntry): string | undefined {
  if (entry.activity === undefined || entry.payload?.communication) return undefined;
  const label =
    entry.activity === 'merged'
      ? 'merged'
      : entry.activity === 'completed'
        ? 'completed'
        : `${entry.payload?.judgment?.kind ?? 'recorded'} judgment`;
  const bound = entry.assignment ? ` · on assignment ${shortFeedID(entry.assignment)}` : '';
  return `Milestone: ${label}${bound}`;
}

// describePostHeader projects one entry to the agreed outside-card header:
// avatar/name/role/project/time/status, unboxed above its own card.
// Server-derived enrichment renders when the entry carries it; otherwise
// the header stays honest to the order position instead of inventing an
// author or date.
export function describePostHeader(entry: FeedEntry, repository: FeedRepositoryRef): PostHeaderModel {
  const recorded = entry.kind === 'record';
  const name = entry.author ?? (recorded ? 'Recorded work' : 'Discussion post');
  const role =
    roleLabel(entry.actor && !/^\d+$/.test(entry.actor) ? entry.actor : undefined) ??
    (recorded ? 'Factory' : 'Comment');
  return {
    key: feedEntryKey(entry),
    avatarInitial: name.slice(0, 1).toUpperCase(),
    avatarFile: feedRolePortraits[role],
    name,
    role,
    project: entry.project ?? repository.label,
    timeText: entry.at ? formatStamp(entry.at) : `recorded order #${entry.seq}, newest first`,
    statusText: readableStatus(entry),
    milestoneText: milestoneText(entry),
  };
}

// Plain chip words for the recorded system statuses shown on stage. A run
// ending is not the work being done, and an assignment only means the work
// started; anything unlisted that failed still reads as needing attention.
const systemStatusLabels: Readonly<Record<string, string>> = {
  'run:succeeded': 'Run ended',
  'run:failed': 'Failed',
  'run:cancelled': 'Stopped',
  'dispatch:assigned': 'Started',
  'dispatch:paused': 'Paused',
  'dispatch:resumed': 'Resumed',
  'intent:draft': 'Draft',
  'intent:accepted': 'Accepted',
  'intent:declined': 'Declined',
};

function systemStatusLabel(status: string | undefined): string {
  const label = status !== undefined && Object.hasOwn(systemStatusLabels, status) ? systemStatusLabels[status] : undefined;
  if (label !== undefined) return label;
  return classifyFeedStatus(status) === 'failure' ? 'Needs attention' : 'Update';
}

function readableStatus(entry: FeedEntry): string {
  const communication = entry.payload?.communication;
  if (communication) {
    if (communication.disposition === 'applied') return 'Suggestion applied';
    if (communication.disposition === 'refused') return 'Suggestion declined';
    if (communication.disposition === 'incorporated') return 'Answer incorporated';
    if (communication.kind === 'ask') {
      if (communication.questionState === 'incorporated') return 'Answer incorporated';
      if (communication.questionState === 'answered') return 'Answer recorded';
      if (communication.questionState === 'replied') return 'Reply received';
    }
    const labels = {
      ask: 'Question open',
      answer: 'Answer recorded',
      reply: 'Reply',
      flag: 'Concern raised',
      learned: 'Observation recorded',
      suggest: 'Proposed',
      partial: 'Progress reported',
      blocked: 'Blocked',
      stuck: 'Diagnosis needed',
      record_decision: 'Decision recorded',
      finish_incomplete: 'Incomplete',
      system: systemStatusLabel(entry.status),
      library_maintenance: (entry.status ?? 'Library updated').replaceAll(':', ' · ').replaceAll('_', ' '),
      artifact: 'Artifact available',
      journey: 'Journey observation recorded',
      diagnosis: 'Diagnosis recorded',
      repair_draft: 'Repair proposed',
    };
    return labels[communication.kind];
  }
  const status = entry.status ?? '';
  const systemLabel = Object.hasOwn(systemStatusLabels, status) ? systemStatusLabels[status] : undefined;
  if (systemLabel !== undefined) return systemLabel;
  if (/cancelled|canceled|stopped/.test(status)) return 'Stopped';
  if (/needs-human|request-changes|clarify/.test(status)) return 'Needs attention';
  if (classifyFeedStatus(status) === 'failure') return 'Needs attention';
  if (status === 'route:discussion') return 'Comment';
  if (status === 'route:existing_work') return 'Linked to work';
  if (status === 'feedback:delivered') return 'Feedback delivered';
  if (entry.activity === 'judgment' || status === 'judgment:applied') return 'Decision recorded';
  if (status === 'draft:valid') return 'Proposal ready';
  if (status === 'effect:created') return 'Work created';
  if (classifyFeedStatus(status) === 'settled') return 'Completed';
  if (/running/.test(status)) return 'Working';
  if (/assess/.test(status)) return 'Assessing';
  if (/pending|queued/.test(status)) return 'Waiting';
  return entry.kind === 'post' ? 'Comment' : 'Recorded';
}

function formatStamp(at: number): string {
  if (!Number.isSafeInteger(at) || at <= 0) return 'time unrecorded';
  const date = new Date(at * 1000);
  if (!Number.isFinite(date.getTime())) return 'time unrecorded';
  return date.toLocaleString(undefined, {month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit'});
}

export type PostBodyState =
  | {state: 'loading'}
  | {state: 'text'; body: string; edited: boolean}
  | {state: 'unavailable'; reason: string; commentId: string};

export interface PostBodyModel {
  readonly bodyText: string;
  readonly editedText: string;
  // Present only when the stored text failed to load: the comment identity
  // the retry control re-reads. The post keeps its place while the retry
  // runs; no draft or intake is touched.
  readonly retryKey?: string;
}

// describePostBody projects one discussion post's payload: the stored
// comment text with its edited marker. Deleted or revoked sources keep
// their tombstone; an unreadable payload says so instead of rendering
// an invented body.
export function describePostBody(body: PostBodyState): PostBodyModel {
  if (body.state === 'text')
    return {
      bodyText: body.body === '' ? 'No comment text recorded.' : body.body,
      editedText: body.edited ? ' (edited)' : '',
    };
  if (body.state === 'unavailable') return {bodyText: body.reason, editedText: '', retryKey: body.commentId};
  return {bodyText: 'Loading discussion text…', editedText: ''};
}

export function renderPostBody(model: PostBodyModel): TemplateResult {
  return html`<section aria-label="Discussion text">
    <p class="soda-feed-body-text">${model.bodyText}${model.editedText}</p>
    ${
      model.retryKey
        ? html`<button type="button" class="ui button" data-post-body-retry=${model.retryKey}>
            Retry loading discussion text
          </button>`
        : ''
    }
  </section>`;
}

export interface PlanIssueModel {
  readonly scopeText: string;
  readonly identityText: string;
  readonly repositoryLink: string | undefined;
  readonly issuesLink: string | undefined;
}

// describePlanIssue projects the plan/issue section: the recorded scope and
// identity plus narrow native links. The bound issue links from the
// family section when the entry carries it; here links land on the
// project and its issue index, never an invented issue.
export function describePlanIssue(entry: FeedEntry, repository: FeedRepositoryRef): PlanIssueModel {
  return {
    scopeText: `Scoped to ${repository.label}.`,
    identityText:
      entry.activity !== undefined
        ? `${entry.activity === 'judgment' ? 'Judgment' : entry.activity === 'merged' ? 'Merge' : 'Completion'} ${shortFeedID(entry.id)} on assignment ${entry.assignment ? shortFeedID(entry.assignment) : 'not recorded'} in global order #${entry.seq}.`
        : `Assignment ${shortFeedID(entry.id)} in global order #${entry.seq}.`,
    repositoryLink: repository.path,
    issuesLink: repository.path === undefined ? undefined : `${repository.path}/issues`,
  };
}

export interface WorkEvidenceModel {
  readonly headline: string;
  readonly provenanceText: string;
  readonly evidenceText: string;
  readonly sourceLinks: readonly {label: string; href: string}[];
}

// describeWorkEvidence projects the focal work/evidence section: the
// recorded canonical fact, its provenance behind the global order key and
// the exact source links the wire supports.
export function describeWorkEvidence(entry: FeedEntry, repository: FeedRepositoryRef): WorkEvidenceModel {
  const links: {label: string; href: string}[] = [];
  if (repository.path !== undefined) links.push({label: `Open ${repository.label}`, href: repository.path});
  return {
    headline: entry.kind === 'record' ? 'One canonical workflow record.' : 'One discussion record.',
    provenanceText: `Provenance: table-global order key #${entry.seq}, stable identity ${entry.kind}:${entry.id.slice(0, 12)}${entry.id.length > 12 ? '…' : ''}. Newest posts stay on top regardless of page arrival order.`,
    evidenceText:
      entry.kind === 'record'
        ? 'Evidence state: recorded. Detailed task, check and review evidence opens in the existing work views.'
        : 'Evidence state: the stored discussion text and its recorded outcome appear in this post.',
    sourceLinks: links,
  };
}

export type AttentionTone = 'quiet' | 'notice';

export interface DecisionAttentionModel {
  readonly tone: AttentionTone;
  readonly attentionText: string;
}

// describeDecisionAttention projects the decision/attention section: quiet
// when the post needs nothing, a notice only when it adds a fact the rest
// of the post does not already state. A recorded failure already leads in
// the family headline and the header status chip, so repeating it here as
// a banner would be a third copy of the same signal. Unknown or missing
// state never reads as success.
export function describeDecisionAttention(entry: FeedEntry, denied: boolean): DecisionAttentionModel {
  if (denied)
    return {
      tone: 'notice',
      attentionText:
        'Access changed: this entry is outside your current permission. It keeps its place without leaking its contents.',
    };
  const statusClass = classifyFeedStatus(entry.status);
  if (statusClass === 'failure')
    return {tone: 'quiet', attentionText: 'This work needs attention; the headline and status above state it.'};
  if (statusClass === 'unknown' && entry.status !== undefined)
    return {
      tone: 'notice',
      attentionText: `Recorded status ${entry.status} is not interpreted yet. Treat the outcome as unknown, not successful.`,
    };
  if (entry.kind === 'post')
    return {tone: 'quiet', attentionText: 'Discussion post with its stored text above and replies below.'};
  return {tone: 'quiet', attentionText: 'No decision or attention needed on this record.'};
}

export type CommentOutcomeTone = 'progress' | 'settled' | 'failed';

export interface ProductEvidenceModel {
  readonly text: string | undefined;
  readonly nativeLinks: readonly {label: string; href: string}[];
}

export interface CommentOutcomeModel {
  readonly productChange?: CommentPipelineState['productChange'];
  readonly productRepository?: number;
  readonly reviewBusy?: boolean;
  readonly needsClarification?: boolean;
  readonly canRetry?: boolean | undefined;
  readonly revision?: number;
  readonly detail?: string | undefined;
  readonly commentId: string;
  readonly text: string;
  readonly tone: CommentOutcomeTone;
  readonly issueLink: string | undefined;
  // Adopted lineage reads separately from the review handoff: canonical
  // completion evidence as text, native issue destinations as their own
  // links. Unknown work stays unknown, never success.
  readonly evidenceText: string | undefined;
  readonly nativeLinks: readonly {label: string; href: string}[];
  readonly retryable: boolean;
  // Server-resolved source context: the feed entry this comment was
  // submitted against, empty when the submission named none.
  readonly sourceText: string;
  // Pipeline stage list (S09): saved → assessed → routed → drafted →
  // created/linked, each with its recorded state word. Stages render
  // as an ordered list so durable continuation progress is visible
  // without resubmitting.
  readonly stages: readonly {label: string; state: string}[];
}

function commentStages(state: CommentPipelineState): {label: string; state: string}[] {
  const stages = [
    {label: 'Comment saved', state: state.intake === 'recorded' ? 'done' : state.intake},
    {
      label: 'JEV assessment',
      state:
        state.judgment === 'pending'
          ? commentAttemptsExhausted(state)
            ? 'stopped — attempts exhausted'
            : 'pending'
          : state.judgment === 'failed'
            ? 'failed'
            : 'done',
    },
  ];
  if (state.judgment === 'pending' || state.judgment === 'failed') return stages;
  if (state.productChange) {
    if (state.productChange.state === 'unavailable') {
      stages.push({label: 'Product change', state: 'unavailable — current product detail unavailable'});
      return stages;
    }
    stages.push({label: 'Product change', state: state.productChange.state.replaceAll('_', ' ')});
    stages.push(...productEvidenceStages(state.productChange));
    return stages;
  }
  if (commentProductLaneWithoutProjection(state)) {
    stages.push({label: 'Product change', state: 'unavailable — current product detail unavailable'});
    return stages;
  }
  const routeLabel =
    state.route === 'new_issue'
      ? 'new issue'
      : state.route === 'existing_work'
        ? 'existing work'
        : state.route === 'discussion'
          ? 'discussion only'
          : state.route === 'clarify'
            ? 'clarification needed'
            : 'not recorded';
  stages.push({label: `Routed: ${routeLabel}`, state: state.route === '' ? 'not recorded' : 'done'});
  if (state.route === 'new_issue') {
    stages.push({
      label: 'Product Lead issue draft',
      state:
        state.draft === 'pending'
          ? commentAttemptsExhausted(state)
            ? 'stopped — attempts exhausted'
            : 'pending'
          : state.draft === 'invalid'
            ? 'invalid'
            : 'done',
    });
    stages.push({
      label: 'Native issue creation',
      state:
        state.effect === 'pending'
          ? state.draft === 'pending' && commentAttemptsExhausted(state)
            ? 'not started'
            : 'pending'
          : state.effect === 'failed'
            ? 'failed'
            : state.effect === 'blocked'
              ? 'blocked'
              : 'done',
    });
  }
  if (state.route === 'existing_work') {
    const selected = state.existingIssue > 0 ? state.existingIssue : state.issue;
    stages.push({
      label: 'Persisted selected-issue link',
      state: selected > 0 ? `done — issue #${selected}` : state.effect === 'pending' ? 'pending' : 'not recorded',
    });
    stages.push({
      label: 'Issue discussion feedback',
      state:
        state.effect === 'applied' || state.effect === 'verified'
          ? 'delivered'
          : commentAttemptsExhausted(state)
            ? 'stopped — retry needed'
            : 'delivery not confirmed',
    });
  }
  return stages;
}

function linkedWorkStages(state: CommentPipelineState): {label: string; state: string}[] {
  const trace = state.linkedWork;
  if (!trace) return [];
  if (trace.status !== 'recorded') return [{label: 'Linked work', state: 'Current work evidence unavailable'}];
  const stages = [
    {
      label: 'Comment adoption',
      state: trace.adoption
        ? `${trace.adoption.state} · ${roleLabel(trace.adoption.actor) ?? trace.adoption.actor}${trace.adoption.reason ? ' · ' + trace.adoption.reason : ''}`
        : 'No adoption receipt recorded',
    },
  ];
  for (const work of trace.work)
    stages.push({
      label: `Linked issue #${trace.issue} · ${work.phase.replaceAll('_', ' ') || 'work'}`,
      state: `${work.outcome || work.stage} · assignment ${shortFeedID(work.assignment)}${work.sourceRevision ? ' · source ' + shortFeedID(work.sourceRevision) : ''}`,
    });
  if (!trace.work.length) stages.push({label: `Linked issue #${trace.issue} work`, state: 'No assignment recorded'});
  if (trace.completion)
    stages.push({
      label: 'Linked issue completion',
      state: `Verified completion record ${shortFeedID(trace.completion)}; comment adoption remains separate`,
    });
  return stages;
}

// productEvidenceStages projects the adopted lineage behind one same-product
// proposal: descendant acceptance, per-task canonical delivery and the
// controller-owned application check. Known adoption and verification read
// independently, even while task evidence is unavailable; unknown work reads
// as unavailable, never as native progress; no stage here claims a native
// issue creation. verified=false means no successful application check was
// recorded — not an active pending check — so the negative reads as no
// recorded success, never as work in flight.
function productEvidenceStages(change: NonNullable<CommentPipelineState['productChange']>): {
  label: string;
  state: string;
}[] {
  if (change.state === 'unavailable') return [{label: 'Change evidence', state: 'unavailable — task delivery unknown'}];
  if (change.adoptedRevision === undefined)
    return [{label: 'Change evidence', state: 'Current work evidence unavailable'}];
  const stages = [
    {
      label: 'Adopted change',
      state: `accepted ${shortFeedID(change.adoptedRevision)}${change.verified ? ' · application check passed' : ' · no successful application check recorded'}`,
    },
  ];
  if (change.evidenceStatus !== 'recorded')
    return [...stages, {label: 'Change task evidence', state: 'unavailable — task delivery unknown'}];
  for (const task of change.evidence)
    stages.push({
      label: `Change task ${shortFeedID(task.taskKey)}`,
      state: `native issue #${task.issue} · acceptance ${shortFeedID(task.acceptance)} · completion ${task.completion ? shortFeedID(task.completion) : 'pending'}`,
    });
  if (change.evidence.length === 0)
    stages.push({label: 'Change task delivery', state: 'No delivered task covers this change yet'});
  return stages;
}

// describeProductEvidence separates canonical completion evidence (text)
// from native destinations (links). Owner-only request controls stay with
// the review handoff; collaborator-visible progress is navigation only.
// Known adoption and verification read independently, even while task
// evidence is unavailable; only tasks with a canonical completion count as
// delivered. A projection miss (state unavailable) says only that nothing
// could be read — it never asserts that no adoption or delivery exists.
// verified=false means no successful application check was recorded, not an
// active pending check.
export function describeProductEvidence(
  change: NonNullable<CommentPipelineState['productChange']>,
  repository: FeedRepositoryRef
): ProductEvidenceModel {
  if (change.state === 'unavailable')
    return {
      text: 'Change detail is currently unavailable; no adopted delivery information could be read for this proposal.',
      nativeLinks: [],
    };
  if (change.adoptedRevision === undefined)
    return {
      text: 'Change work evidence is unavailable; no adopted delivery has been recorded for this proposal.',
      nativeLinks: [],
    };
  const nativeLinks =
    change.evidenceStatus === 'recorded'
      ? change.evidence.flatMap((task) => {
          const href = repository.path !== undefined ? nativeIssueLink(repository.path, String(task.issue)) : undefined;
          return href !== undefined ? [{label: `Open native issue #${task.issue}`, href}] : [];
        })
      : [];
  if (change.evidenceStatus !== 'recorded')
    return {
      text: change.verified
        ? `Adopted ${shortFeedID(change.adoptedRevision)} passed the approved application check. Change task evidence is unavailable; task delivery unknown.`
        : `Adopted ${shortFeedID(change.adoptedRevision)}; no successful approved application check recorded. Change task evidence is unavailable; task delivery unknown.`,
      nativeLinks,
    };
  const completed = change.evidence.filter((task) => task.completion !== undefined).length;
  const tasks = completed === 1 ? '1 delivered task' : `${completed} delivered tasks`;
  const text = change.verified
    ? `Adopted ${shortFeedID(change.adoptedRevision)} passed the approved application check with ${completed === 0 ? 'no delivered task covering this change yet' : `${tasks} recorded`}.`
    : `Adopted ${shortFeedID(change.adoptedRevision)}; ${completed === 0 ? 'no delivered task covers this change yet' : `${tasks} recorded`}, no successful approved application check recorded.`;
  return {text, nativeLinks};
}

export function commentSourceText(state: CommentPipelineState): string {
  const source = state.source;
  if (source === undefined) return '';
  const parts = [`About ${source.kind === 'record' ? 'recorded work' : 'a discussion post'} ${shortFeedID(source.id)}`];
  if (source.issue) parts.push(`issue #${source.issue}`);
  if (source.pr) parts.push(`pull request #${source.pr}`);
  if (source.ref) parts.push(`candidate ${source.ref}`);
  if (source.status) parts.push(`status ${source.status}`);
  // The exact accepted revision the comment was discussed against, so later
  // work can tell stale context from the current direction at a glance.
  if (source.product)
    parts.push(`accepted ${shortFeedID(source.product.intent)} @ ${shortFeedID(source.product.acceptedRevision)}`);
  return `${parts.join(' · ')}${source.summary ? ` — ${source.summary}` : ''}`;
}

// describeCommentOutcome projects one stored pipeline row to its visible
// outcome. Saved comment, assessment, draft and native effect stay separate
// observable stages; a confirmed issue links its native issue, while
// discussion-only, existing-work, clarification and failure outcomes each
// read exactly as recorded. Unknown states never read as success.
export function describeCommentOutcome(
  state: CommentPipelineState,
  repository: FeedRepositoryRef,
  reviewBusy = false
): CommentOutcomeModel {
  const selectedExisting = state.existingIssue > 0 ? state.existingIssue : state.issue;
  const linkIssue = state.route === 'existing_work' ? selectedExisting : state.issue;
  const issueLink =
    linkIssue > 0 && repository.path !== undefined ? nativeIssueLink(repository.path, String(linkIssue)) : undefined;
  const sourceText = commentSourceText(state);
  const evidence = state.productChange ? describeProductEvidence(state.productChange, repository) : undefined;
  const outcome = (text: string, tone: CommentOutcomeTone, retryable: boolean): CommentOutcomeModel => ({
    commentId: state.commentId,
    text,
    tone,
    issueLink,
    evidenceText: evidence?.text,
    nativeLinks: evidence?.nativeLinks ?? [],
    retryable,
    sourceText,
    stages: [...commentStages(state), ...linkedWorkStages(state)],
    canRetry: state.canRetry,
    revision: state.revision,
    detail: state.detail,
    reviewBusy,
    needsClarification: state.route === 'clarify',
    ...(state.productChange ? {productChange: state.productChange, productRepository: repository.id} : {}),
  });
  // A projection miss — explicit or inferred from a product lane whose
  // detail did not arrive — never falls back to detached generic progress.
  const productUnavailable = state.productChange?.state === 'unavailable' || commentProductLaneWithoutProjection(state);
  if (state.productChange) {
    switch (state.productChange.state) {
      case 'review_required':
        return outcome(
          state.productChange.canReview
            ? 'Review this proposed change before updating the accepted direction.'
            : 'Waiting for the request owner to review this proposed change.',
          'progress',
          false
        );
      case 'draft':
        return outcome(
          'The change is saved in the build request for review. Accept its direction before building.',
          'progress',
          false
        );
      case 'accepted':
        return outcome(
          'The changed direction is accepted. Follow its build and verification before applying a version.',
          'settled',
          false
        );
      case 'superseded':
        return outcome(
          'This proposal refers to an earlier direction. Refresh the request before proposing another change.',
          'failed',
          false
        );
      case 'unavailable':
        return outcome(
          'The proposed change detail is currently unavailable. Check progress to reload it.',
          'progress',
          true
        );
    }
  }
  if (productUnavailable)
    return outcome(
      'The proposed change detail is currently unavailable. Check progress to reload it.',
      'progress',
      true
    );
  if (commentAttemptsExhausted(state)) {
    if (state.judgment === 'pending' || state.judgment === 'failed')
      return outcome(
        'Assessment stopped after its bounded failed attempts. Further assessment needs an explicit retry.',
        'failed',
        false
      );
    if (state.route === 'new_issue' && state.draft === 'pending')
      return outcome(
        'Issue authoring stopped after its bounded failed attempts. No issue draft was accepted.',
        'failed',
        false
      );
    if (state.route === 'new_issue' && (state.effect === 'pending' || state.effect === 'failed'))
      return outcome(
        'Native issue creation is unconfirmed after its bounded attempts. Inspect the original operation before retrying.',
        'failed',
        false
      );
  }
  if (state.judgment === 'pending') return outcome('Saved. JEV is assessing your comment.', 'progress', true);
  if (state.judgment === 'failed')
    return outcome(
      'Assessment failed after its bounded attempts. Retrying recovers the original instead of duplicating it.',
      'failed',
      true
    );
  if (state.route === 'discussion') return outcome('Kept as discussion.', 'settled', false);
  if (state.route === 'clarify')
    return outcome('Waiting on clarification from the author before any further routing.', 'progress', false);
  if (state.route === 'existing_work') {
    if (state.effect === 'applied' || state.effect === 'verified')
      return outcome(`Feedback added to issue #${selectedExisting}.`, 'settled', false);
    if (selectedExisting <= 0 && state.effect === 'pending')
      return outcome('Finding the issue for this feedback.', 'progress', true);
    return outcome(
      selectedExisting > 0
        ? `Linked to issue #${selectedExisting}. Feedback delivery is not confirmed.`
        : 'The issue for this feedback has not been confirmed.',
      selectedExisting > 0 ? 'settled' : 'progress',
      selectedExisting <= 0
    );
  }
  if (state.draft === 'pending')
    return outcome('Routed to new work. Product Lead issue authoring pending.', 'progress', true);
  if (state.draft === 'invalid')
    return outcome('Issue draft invalid: no native effect ran and nothing was created.', 'failed', false);
  if (state.effect === 'pending' || state.effect === 'failed')
    return outcome(
      state.effect === 'pending'
        ? 'Issue drafted. Validated native creation pending.'
        : 'Native creation failed; retrying reconciles instead of duplicating.',
      state.effect === 'pending' ? 'progress' : 'failed',
      true
    );
  if (state.effect === 'blocked')
    return outcome(
      'Blocked: the source changed or access was lost before creation. The comment is preserved.',
      'failed',
      false
    );
  return outcome(`Issue #${state.issue} created.`, 'settled', false);
}

export interface DiscussionModel {
  readonly showComposer?: boolean;
  readonly compactComposer?: boolean;
  readonly label?: string;
  readonly submitLabel?: string;
  readonly key: string;
  readonly draft: string;
  readonly submitAvailable: boolean;
  readonly outcomes: readonly CommentOutcomeModel[];
  readonly intakeText: string;
  readonly discussionLink: string | undefined;
}

// describeDiscussion projects the shared fallback/discussion section.
// Submitting a human comment persists it through the Lane B intake, then
// exposes assessment, draft and native-effect progress on the same outcome.
// Retrying reuses the draft's idempotency key and recovers the original.
// Typing a draft, opening a thread or reloading never invokes intake.
export function describeDiscussion(
  entry: FeedEntry,
  repository: FeedRepositoryRef,
  draft: string,
  outcomes: readonly CommentOutcomeModel[]
): DiscussionModel {
  return {
    key: feedEntryKey(entry),
    draft,
    compactComposer: true,
    submitAvailable: draft.trim() !== '',
    outcomes,
    intakeText:
      'Submitting asks JEV to route this comment; a selected new request reaches Product Lead authoring and validated native creation. Acknowledgements stay discussion-only.',
    discussionLink:
      repository.path === undefined || !entry.issue ? undefined : `${repository.path}/issues/${entry.issue}`,
  };
}

export function renderPostHeader(model: PostHeaderModel, assetBase = ''): TemplateResult {
  return html`<div class="soda-feed-header">
    <span class="soda-feed-avatar" aria-hidden="true"
      >${
        model.avatarFile
          ? html`<img
              src=${assetBase + model.avatarFile}
              alt=""
              width="36"
              height="36"
              loading="lazy"
              decoding="async"
            />`
          : model.avatarInitial
      }</span
    >
    <div>
      <p><strong>${model.name}</strong>${model.role !== model.name ? html` · ${model.role}` : ''} · ${model.project}</p>
      <p>${model.timeText} · <span class="soda-feed-status">${model.statusText}</span></p>
      ${model.milestoneText ? html`<p>${model.milestoneText}</p>` : ''}
    </div>
  </div>`;
}

export function renderPlanIssue(model: PlanIssueModel): TemplateResult {
  return html`<section aria-label="Plan and issue scope">
    <p>${model.scopeText}</p>
    <p>${model.identityText}</p>
    <p>
      ${model.repositoryLink ? html`<a href=${model.repositoryLink}>Open project</a>` : 'Project link unavailable.'}
      ${model.issuesLink ? html`<a href=${model.issuesLink}>Open issue index</a>` : ''}
    </p>
  </section>`;
}

export function renderWorkEvidence(model: WorkEvidenceModel): TemplateResult {
  return html`<section aria-label="Work and evidence">
    <p role="status">${model.headline}</p>
    <p>${model.provenanceText}</p>
    <p>${model.evidenceText}</p>
    ${
      model.sourceLinks.length > 0
        ? html`<p>${model.sourceLinks.map((link) => html`<a href=${link.href}>${link.label}</a>`)}</p>`
        : html`<p>Source links unavailable.</p>`
    }
  </section>`;
}

export function renderDecisionAttention(model: DecisionAttentionModel): TemplateResult {
  return html`<section aria-label="Decision and attention">
    <p role=${model.tone === 'notice' ? 'alert' : 'status'}>${model.attentionText}</p>
  </section>`;
}

export function renderDiscussion(model: DiscussionModel): TemplateResult {
  return html`<section aria-label="Discussion">
    ${model.outcomes.map(
      (outcome) => html`<div class="soda-feed-outcome">
        <p role=${outcome.tone === 'failed' ? 'alert' : 'status'}>
          ${outcome.text} ${outcome.issueLink ? html`<a href=${outcome.issueLink}>Open linked issue</a>` : ''}
          ${
            outcome.retryable
              ? html`<button type="button" class="ui button" data-comment-retry=${outcome.commentId}>
                  Check progress
                </button>`
              : ''
          }
        </p>
        ${outcome.detail ? html`<p>${outcome.detail}</p>` : ''}
        ${outcome.evidenceText ? html`<p>${outcome.evidenceText}</p>` : ''}
        ${
          // Review, continuation and the accepted private request mutate or
          // open the owner's private build request, so they stay owner-only
          // (canReview). Collaborator-visible accepted progress is navigation
          // to the authorized evidence the projection exposes: it performs
          // no mutation and never targets the private intent route.
          outcome.productChange && outcome.productChange.state === 'review_required' && outcome.productChange.canReview
            ? html`<button
                type="button"
                class="ui button"
                data-comment-product-change=${outcome.commentId}
                data-product-revision=${outcome.revision ?? 0}
                data-product-repository=${outcome.productRepository ?? 0}
                ?disabled=${outcome.reviewBusy}
              >
                Review proposed change
              </button>`
            : ''
        }
        ${
          outcome.productChange && outcome.productChange.state === 'draft' && outcome.productChange.canReview
            ? html`<button
                type="button"
                class="ui button"
                data-comment-product-change=${outcome.commentId}
                data-product-revision=${outcome.revision ?? 0}
                data-product-repository=${outcome.productRepository ?? 0}
                ?disabled=${outcome.reviewBusy}
              >
                Continue change review
              </button>`
            : ''
        }
        ${
          outcome.productChange && outcome.productChange.state === 'accepted' && outcome.productChange.canReview
            ? html`<button
                type="button"
                class="ui button"
                data-comment-product-change=${outcome.commentId}
                data-product-revision=${outcome.revision ?? 0}
                data-product-repository=${outcome.productRepository ?? 0}
                ?disabled=${outcome.reviewBusy}
              >
                Open accepted change
              </button>`
            : ''
        }
        ${
          outcome.productChange &&
          outcome.productChange.state === 'accepted' &&
          !outcome.productChange.canReview &&
          outcome.nativeLinks.length > 0
            ? html`<a class="ui button" href=${outcome.nativeLinks[0]?.href ?? ''}>Open accepted evidence</a>`
            : ''
        }
        ${outcome.canRetry ? html`<button type="button" data-comment-revise=${outcome.commentId}>${outcome.needsClarification ? 'Answer clarification' : 'Revise or retry request'}</button>` : ''}
        <details>
          <summary>Processing details</summary>
          ${outcome.sourceText ? html`<p class="soda-feed-outcome-source">${outcome.sourceText}</p>` : ''}
          ${
            outcome.productChange
              ? html`<p>Build request <code>${outcome.productChange.intent}</code></p>
                  <p>Original accepted revision <code>${outcome.productChange.acceptedRevision}</code></p>
                  ${outcome.productChange.draftRevision ? html`<p>Change draft <code>${outcome.productChange.draftRevision}</code></p>` : ''}
                  ${
                    outcome.productChange.adoptedRevision
                      ? html`<p>Adopted revision <code>${outcome.productChange.adoptedRevision}</code></p>`
                      : ''
                  }
                  ${
                    outcome.nativeLinks.length > 0
                      ? html`<p>${outcome.nativeLinks.map((link) => html`<a href=${link.href}>${link.label}</a>`)}</p>`
                      : ''
                  }`
              : ''
          }
          <ol class="soda-feed-outcome-stages" aria-label="Comment pipeline stages">
            ${outcome.stages.map((stage) => html`<li><strong>${stage.label}</strong> — ${stage.state}</li>`)}
          </ol>
        </details>
      </div>`
    )}
    ${
      model.showComposer === false
        ? ''
        : html`<label
              >${model.compactComposer ? html`<span class="soda-visually-hidden">${model.label ?? 'Reply draft'}</span>` : (model.label ?? 'Reply draft')}
              <textarea
                name="reply"
                data-feed-draft=${model.key}
                rows=${model.compactComposer ? '1' : '3'}
                maxlength="4000"
                placeholder=${model.label === 'Project comment' ? 'Ask a question, share feedback, or request a change…' : 'Write a reply…'}
                .value=${live(model.draft)}
              ></textarea>
            </label>
            <p class="soda-feed-compose-actions">
              <button
                type="button"
                class="ui primary button"
                data-comment-submit=${model.key}
                .disabled=${live(!model.submitAvailable)}
                title="Submit persists the comment and starts its JEV assessment."
              >
                ${model.submitLabel ?? (model.compactComposer ? 'Post' : 'Submit reply')}
              </button>
              ${model.label === 'Project comment' ? html`<span class="soda-feed-compose-hint">JEV reviews your comment and routes the next step.</span>` : ''}
              ${model.discussionLink ? html`<a href=${model.discussionLink}>Open issue discussion</a>` : ''}
            </p>`
    }
  </section>`;
}

export interface FeedPostModel {
  readonly header: PostHeaderModel;
  readonly family: FeedFamilyModel;
  readonly plan: PlanIssueModel;
  readonly work: WorkEvidenceModel;
  readonly attention: DecisionAttentionModel;
  readonly discussion: DiscussionModel;
  readonly postBody: PostBodyModel | undefined;
  readonly comments: CommentsModel;
  // Version 2 anatomy: one of seven states, a one-sentence headline, the
  // consequence, the Next line and the work item named in the byline.
  readonly state: FeedState;
  readonly headline: string;
  readonly detail: string;
  readonly next: string;
  readonly workItem: string | undefined;
  readonly judgment: FeedJudgment | undefined;
  readonly thread: Pick<FeedThreadItem, 'comments' | 'commentCount' | 'changes' | 'from' | 'level'> | undefined;
}

// CommentsModel projects the comment drawer's pull-tab: open state plus the
// loaded reply count and first author initials. Replies load on first open,
// so before that the tab reads plainly "Comments" rather than inventing a
// count the feed page has not read yet.
export interface CommentsModel {
  readonly key: string;
  readonly open: boolean;
  readonly countText: string;
  readonly faces: readonly string[];
}

export function commentsCountText(count: number | undefined, more = false): string {
  if (count === undefined) return 'Comments';
  if (count === 1) return '1 comment';
  return `${count}${more ? '+' : ''} comments`;
}

export function describeComments(
  key: string,
  open: boolean,
  thread: {items: readonly {authorName: string}[]; next: number} | undefined
): CommentsModel {
  return {
    key,
    open,
    countText: commentsCountText(thread?.items.length, (thread?.next ?? 0) > 0),
    faces: (thread?.items ?? []).slice(0, 3).map((item) => (item.authorName || '?').slice(0, 1).toUpperCase()),
  };
}

// describeFeedPost composes one entry's full post: its registered family
// section, the shared scope/evidence sections, the discussion shell, and
// the stored discussion text for post entries. Every entry renders; no
// role or stage disappears merely because a bespoke component is
// unfinished — the fallback family preserves its facts and links.
export function describeFeedPost(
  entry: FeedEntry,
  repository: FeedRepositoryRef,
  draft: string,
  outcomes: readonly CommentOutcomeModel[],
  body: PostBodyState | undefined,
  denied: boolean,
  comments?: CommentsModel,
  thread?: FeedThreadItem
): FeedPostModel {
  const family = describeFeedFamily(entry, repository);
  const state = feedState(entry);
  const line = describePostLine(entry, state, family.family, family.headline);
  return {
    state,
    headline: line.headline,
    detail: line.detail,
    next: state === 'in_progress' && thread !== undefined && thread.level !== 'update' ? 'Nothing needed from you' : line.next,
    workItem: line.workItem,
    judgment: entry.payload?.judgment,
    thread: thread
      ? {
          comments: thread.comments,
          commentCount: thread.commentCount,
          changes: thread.changes,
          level: thread.level,
          ...(thread.from ? {from: thread.from} : {}),
        }
      : undefined,
    header: describePostHeader(entry, repository),
    family,
    plan: describePlanIssue(entry, repository),
    work: describeWorkEvidence(entry, repository),
    attention: describeDecisionAttention(entry, denied),
    discussion: describeDiscussion(entry, repository, draft, outcomes),
    postBody: body === undefined || entry.payload?.communication ? undefined : describePostBody(body),
    comments: comments ?? {key: feedEntryKey(entry), open: false, countText: 'Comments', faces: []},
  };
}

// Small state and comment icons, drawn so each reads without colour.
const stateIcons: Readonly<Record<FeedState, TemplateResult>> = {
  needs_you: svg`<circle cx="7" cy="7" r="4.2" fill="currentColor" stroke="none"/>`,
  problem: svg`<path d="M7 1.6 12.9 12.2H1.1z"/><path d="M7 5.4v3.1M7 10.3v.05"/>`,
  waiting: svg`<circle cx="7" cy="7" r="5.6"/><path d="M7 3.9V7l2.1 1.5"/>`,
  in_progress: svg`<circle cx="7" cy="7" r="5.6"/><path d="M7 1.4a5.6 5.6 0 0 1 0 11.2z" fill="currentColor" stroke="none"/>`,
  done: svg`<path d="M2.4 7.4 5.6 10.5 11.6 3.6"/>`,
  decided: svg`<path d="M7 1.4 12.6 7 7 12.6 1.4 7z" fill="currentColor" stroke="none"/>`,
  info: svg`<circle cx="7" cy="7" r="5"/>`,
};
const commentIcons: Readonly<Record<CommentKind | 'up', TemplateResult>> = {
  edit: svg`<path d="M9.4 2.2l2.4 2.4-7 7H2.4V9.2z"/>`,
  read: svg`<path d="M1.2 7s2.2-4 5.8-4 5.8 4 5.8 4-2.2 4-5.8 4S1.2 7 1.2 7z"/><circle cx="7" cy="7" r="1.6"/>`,
  pass: svg`<path d="M2.4 7.4 5.6 10.5 11.6 3.6"/>`,
  fail: svg`<path d="M3.2 3.2l7.6 7.6M10.8 3.2l-7.6 7.6"/>`,
  save: svg`<path d="M3.2 12.4V1.8M3.2 2.4h7.6L9.2 5l1.6 2.6H3.2"/>`,
  jev: svg`<path d="M7 1.4 12.6 7 7 12.6 1.4 7z" fill="currentColor" stroke="none"/>`,
  reply: svg`<path d="M5.4 3 2 6.2l3.4 3.2M2.4 6.2h5.4a4 4 0 0 1 4 4v.6"/>`,
  know: svg`<path d="M2 3h4a1.5 1.5 0 0 1 1 .5A1.5 1.5 0 0 1 8 3h4v8H8a1 1 0 0 0-1 1 1 1 0 0 0-1-1H2z"/>`,
  shot: svg`<rect x="1.6" y="2.4" width="10.8" height="9.2" rx="1.4"/><path d="M1.8 10.2 5 7l2.4 2.4L9 7.8l3.2 3.2"/>`,
  link: svg`<path d="M5.6 8.4 8.4 5.6M6.1 3.9l.9-.9a2.4 2.4 0 0 1 3.4 3.4l-.9.9M7.9 10.1l-.9.9a2.4 2.4 0 0 1-3.4-3.4l.9-.9"/>`,
  text: svg`<path d="M2 2.6h10v7H6.4L3.6 12V9.6H2z"/>`,
  up: svg`<path d="M7 12.2V2.2M3.2 6 7 2.2 10.8 6"/>`,
};
function icon(path: TemplateResult, className = ''): TemplateResult {
  return html`<svg
    class="soda-feed-ic ${className}"
    viewBox="0 0 14 14"
    aria-hidden="true"
    fill="none"
    stroke="currentColor"
    stroke-width="1.6"
    stroke-linecap="round"
    stroke-linejoin="round"
  >
    ${path}
  </svg>`;
}

function clockText(at: number | undefined): string {
  if (at === undefined || !Number.isSafeInteger(at) || at <= 0) return '';
  const date = new Date(at * 1000);
  return Number.isFinite(date.getTime()) ? date.toLocaleTimeString(undefined, {hour: '2-digit', minute: '2-digit'}) : '';
}

// One comment: icon · who · did what · to what · result · time.
function renderCommentLine(line: FeedCommentLine, inGroup = false): TemplateResult {
  const text =
    line.kind === 'text'
      ? html`<strong>${line.who}</strong><span class="soda-feed-cm-dot">·</span><span class="soda-feed-cm-quote">${line.what}</span>`
      : html`${inGroup ? '' : html`<strong>${line.who}</strong> `}${line.did}${line.what ? html` ${line.what}` : ''}${
          line.result
            ? html`<span class="soda-feed-cm-dot">·</span><span class="soda-feed-cm-result" data-tone=${line.tone}>${line.result}</span>`
            : ''
        }`;
  return html`<li class="soda-feed-cm" data-kind=${line.kind} data-tone=${line.tone}>
    ${icon(commentIcons[line.kind])}
    <span class="soda-feed-cm-text"
      >${text}${
        line.promoted
          ? html`<a class="soda-feed-cm-promoted" href=${'#feed-' + line.promoted.replace(':', '-')}
              >${icon(commentIcons.up)}Became a post</a
            >`
          : ''
      }</span
    >
    <time>${clockText(line.at)}</time>
  </li>`;
}

function renderCommentGroup(group: FeedCommentGroup): TemplateResult {
  const from = clockText(group.from);
  const to = clockText(group.to);
  return html`<li class="soda-feed-cm-group">
    <details>
      <summary class="soda-feed-cm" data-kind=${group.kind} data-tone=${group.tone}>
        ${icon(commentIcons[group.kind])}
        <span class="soda-feed-cm-text"
          ><strong>${group.who}</strong> ${group.summary}${
            group.result
              ? html`<span class="soda-feed-cm-dot">·</span><span class="soda-feed-cm-result" data-tone=${group.tone}>${group.result}</span>`
              : ''
          }</span
        >
        <time>${from && to && from !== to ? `${from}–${to}` : to}</time>
      </summary>
      <ul class="soda-feed-cms">
        ${group.items.map((item) => renderCommentLine(item, true))}
      </ul>
    </details>
  </li>`;
}

function renderThread(comments: readonly FeedThreadComment[], count: number): TemplateResult | string {
  if (comments.length === 0) return '';
  return html`<section class="soda-feed-thread" aria-label="Steps">
    <p class="soda-feed-thread-count">${icon(commentIcons.text)}${count === 1 ? '1 step' : `${count} steps`}</p>
    <ul class="soda-feed-cms">
      ${comments.map((comment) => ('group' in comment ? renderCommentGroup(comment) : renderCommentLine(comment)))}
    </ul>
  </section>`;
}

// JEV's answer as bars: the share for each option, a notch at the 80% SodaOS
// needs to act on its own, and how sure JEV was.
function renderJudgment(judgment: FeedJudgment): TemplateResult {
  const names = new Map(
    judgment.candidates.map((candidate) => [
      candidate.id,
      candidate.issue ? `Issue #${candidate.issue}${candidate.objective ? ` · ${candidate.objective}` : ''}` : (candidate.objective ?? candidate.id),
    ])
  );
  const label = (option: string) => {
    const name = names.get(option) ?? option.replaceAll('_', ' ');
    return name.charAt(0).toUpperCase() + name.slice(1);
  };
  const rows = Object.entries(judgment.probabilities)
    .sort((a, b) => b[1] - a[1])
    .slice(0, 4);
  const notch = judgment.kind === 'delivery';
  const sure = judgment.confidence;
  return html`<div class="soda-feed-jev">
    ${rows.map(([option, weight], index) => {
      const picked = option === judgment.selected;
      return html`<div class="soda-feed-jev-opt" data-top=${index === 0} data-picked=${picked}>
        <span class="soda-feed-jev-label"
          >${picked ? icon(stateIcons.done) : ''}${label(option)}</span
        ><span class="soda-feed-jev-pc">${Math.round(weight * 100)}%</span>
        <span class="soda-feed-jev-track"
          ><b style=${`width:${Math.round(weight * 100)}%`}></b>${
            notch && index === 0 ? html`<i class="soda-feed-jev-notch" style="left:calc(80% - 1px)"></i>` : ''
          }</span
        >
      </div>`;
    })}
    ${notch ? html`<p class="soda-feed-jev-key"><span class="soda-feed-jev-kn"><i></i></span>SodaOS acts on its own from 80%</p>` : ''}
    ${
      sure !== undefined
        ? html`<p class="soda-feed-jev-sure">
            JEV is <strong>${Math.round(sure * 100)}% sure</strong> of this answer${
              notch ? (sure >= 0.8 ? ', which clears the 80% SodaOS needs' : ', under the 80% SodaOS needs') : ''
            }
          </p>`
        : ''
    }
  </div>`;
}

// The recorded status word, unless it is a generic one that would hide the
// state (a failed report still reads "Progress reported" in the record).
const genericStatus = /^(Recorded|Update|Progress reported)$/;
function pillWord(model: FeedPostModel): string {
  const word = model.header.statusText;
  return genericStatus.test(word) ? feedStateWords[model.state] : word;
}

function renderByline(model: FeedPostModel, assetBase: string): TemplateResult {
  const header = model.header;
  const changed = model.thread?.changes.length ? 'updated ' : '';
  return html`<div class="soda-feed-header">
    <span class="soda-feed-avatar" aria-hidden="true"
      >${
        header.avatarFile
          ? html`<img src=${assetBase + header.avatarFile} alt="" width="28" height="28" loading="lazy" decoding="async" />`
          : header.avatarInitial
      }</span
    >
    <p>
      <strong>${header.name}</strong>${header.role !== header.name ? html`<span class="soda-feed-by-role"> · ${header.role}</span>` : ''}
      <span class="soda-feed-by-meta"
        >${model.workItem ?? header.project}<i>·</i>${changed}${header.timeText}${
          model.thread?.from
            ? html`<i>·</i><a href=${'#feed-' + model.thread.from.key.replace(':', '-')}>from ${model.thread.from.label}</a>`
            : ''
        }</span
      >
    </p>
  </div>`;
}

function renderChanges(model: FeedPostModel): TemplateResult | string {
  const changes = model.thread?.changes ?? [];
  if (changes.length === 0) return '';
  return html`<details class="soda-feed-changes">
    <summary>Show changes · ${changes.length}</summary>
    <ol>
      ${changes.map(
        (entry) =>
          html`<li>
            <time>${clockText(entry.at)}</time
            ><span>${entry.payload?.communication?.title ?? entry.summary ?? entry.status ?? 'Earlier state'}</span>
          </li>`
      )}
    </ol>
  </details>`;
}

// renderFeedPost keeps the version 2 anatomy: the byline above the card, then
// the state pill and one-sentence headline, the detail, the kind's own
// evidence, the Next line and folded details. Small steps follow as one-line
// comments, and the reply drawer stays under them.
export function renderFeedPost(
  model: FeedPostModel,
  replies: TemplateResult | string = '',
  request?: import('./sodaspaces-factory-feed-artifact.js').FeedArtifactRequest,
  assetBase = ''
): TemplateResult {
  const discussionOnly = model.family.family === 'discussion' && !model.family.communication;
  const family =
    model.family.family === 'fallback'
      ? html`${renderFeedFamily(model.family, request)} ${renderWorkEvidence(model.work)}`
      : renderFeedFamily(model.family, request);
  const focal = discussionOnly
    ? ''
    : model.judgment
      ? html`${renderJudgment(model.judgment)}
          <details class="soda-feed-more-content">
            <summary>The recorded judgment</summary>
            ${family}
          </details>`
      : family;
  return html`<article
    class="soda-feed-post"
    id=${'feed-' + model.header.key.replace(':', '-')}
    data-status=${model.family.neutral ? 'ended' : model.family.statusClass}
    data-state=${model.state}
    data-level=${model.thread?.level ?? 'post'}
    data-role=${model.header.role.toLowerCase().replace(/[\s-]+/g, '_')}
  >
    ${renderByline(model, assetBase)}
    <div class="soda-feed-stack">
      <div class="soda-feed-card" data-state=${model.state}>
        <div class="soda-feed-hgroup">
          <p class="soda-feed-hl">
            <span class="soda-feed-status soda-feed-pill" data-state=${model.state}
              >${icon(stateIcons[model.state])}${pillWord(model)}</span
            >${discussionOnly ? '' : model.headline}
          </p>
          ${model.detail && !discussionOnly ? html`<p class="soda-feed-dt">${model.detail}</p>` : ''}
        </div>
        ${model.postBody ? renderPostBody(model.postBody) : ''} ${focal}
        ${model.attention.tone === 'notice' ? renderDecisionAttention(model.attention) : ''}
        ${
          discussionOnly
            ? ''
            : html`<p class="soda-feed-next" data-state=${model.state}>
                ${model.state === 'needs_you' ? html`<b class="soda-feed-age">${icon(stateIcons.waiting)}${model.next}</b>` : model.next}
              </p>`
        }
        ${renderChanges(model)}
      </div>
      ${renderThread(model.thread?.comments ?? [], model.thread?.commentCount ?? 0)}
      <section class="soda-feed-drawer" data-open=${model.comments.open} aria-label="Comments">
        <button
          type="button"
          class="soda-feed-pull-tab"
          data-feed-drawer-toggle=${model.comments.key}
          aria-expanded=${model.comments.open}
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
            <path d="M21 11.5a8.5 8.5 0 0 1-12.4 7.5L3 21l2-5.6A8.5 8.5 0 1 1 21 11.5Z" />
          </svg>
          <strong>${model.comments.countText}</strong>
          ${model.comments.faces.length ? html`<span class="soda-feed-faces" aria-hidden="true">${model.comments.faces.map((face) => html`<i>${face}</i>`)}</span>` : ''}
          <svg
            class="soda-feed-chev"
            viewBox="0 0 24 24"
            aria-hidden="true"
            fill="none"
            stroke="currentColor"
            stroke-width="1.8"
            stroke-linecap="round"
            stroke-linejoin="round"
          >
            <path d="m6 9 6 6 6-6" />
          </svg>
        </button>
        <div class="soda-feed-drawer-body" .inert=${!model.comments.open}>
          <div class="soda-feed-drawer-inner">
            ${model.discussion.outcomes.length ? renderDiscussion({...model.discussion, showComposer: false}) : ''}
            ${replies} ${renderDiscussion({...model.discussion, outcomes: []})}
          </div>
        </div>
      </section>
    </div>
  </article>`;
}
