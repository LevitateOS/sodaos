import {html} from 'lit';
import {live} from 'lit/directives/live.js';
import type {TemplateResult} from 'lit';
import type {CommentPipelineState, FeedEntry} from './sodaspaces-factory-feed-response.js';
import {commentAttemptsExhausted, feedEntryKey} from './sodaspaces-factory-feed-response.js';
import {nativePath} from './sodaspaces-factory-activity.js';
import {nativeIssueLink} from './sodaspaces-factory-work-view.js';
import {
  classifyFeedStatus,
  describeFeedFamily,
  renderFeedFamily,
  roleLabel,
} from './sodaspaces-factory-feed-families.js';
import type {FeedFamilyModel} from './sodaspaces-factory-feed-families.js';

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
  if (entry.activity === undefined) return undefined;
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
  return {
    key: feedEntryKey(entry),
    avatarInitial: name.slice(0, 1).toUpperCase(),
    name,
    role: roleLabel(entry.actor && !/^\d+$/.test(entry.actor) ? entry.actor : undefined) ?? (recorded ? 'Factory' : 'Comment'),
    project: entry.project ?? repository.label,
    timeText: entry.at ? formatStamp(entry.at) : `recorded order #${entry.seq}, newest first`,
    statusText: readableStatus(entry),
    milestoneText: milestoneText(entry),
  };
}

function readableStatus(entry: FeedEntry): string {
  const status = entry.status ?? '';
  if (/cancelled|canceled|stopped/.test(status)) return 'Stopped';
  if (/needs-human|request-changes|clarify/.test(status)) return 'Needs your attention';
  if (classifyFeedStatus(status) === 'failure') return 'Needs attention';
  if (status === 'route:discussion') return 'Comment';
  if (status === 'route:existing_work') return 'Linked to work';
  if (status === 'feedback:delivered') return 'Feedback delivered';
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
  | {state: 'unavailable'; reason: string};

export interface PostBodyModel {
  readonly bodyText: string;
  readonly editedText: string;
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
  if (body.state === 'unavailable') return {bodyText: body.reason, editedText: ''};
  return {bodyText: 'Loading discussion text…', editedText: ''};
}

export function renderPostBody(model: PostBodyModel): TemplateResult {
  return html`<section aria-label="Discussion text">
    <p class="soda-feed-body-text">${model.bodyText}${model.editedText}</p>
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
// when the post needs nothing, a notice when its state is material. A
// recorded failure or wait leads here regardless of family, so no
// successful-looking section can hide it. Unknown or missing state
// never reads as success.
export function describeDecisionAttention(entry: FeedEntry, denied: boolean): DecisionAttentionModel {
  if (denied)
    return {
      tone: 'notice',
      attentionText:
        'Access changed: this entry is outside your current permission. It keeps its place without leaking its contents.',
    };
  const statusClass = classifyFeedStatus(entry.status);
  if (statusClass === 'failure')
    return {
      tone: 'notice',
      attentionText: 'This work needs attention. See the result above.',
    };
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

export interface CommentOutcomeModel {
  readonly needsClarification?: boolean;
  readonly canRetry?: boolean | undefined;
  readonly revision?: number;
  readonly detail?: string | undefined;
  readonly commentId: string;
  readonly text: string;
  readonly tone: CommentOutcomeTone;
  readonly issueLink: string | undefined;
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
      state: state.judgment === 'pending' ? (commentAttemptsExhausted(state) ? 'stopped — attempts exhausted' : 'pending') : state.judgment === 'failed' ? 'failed' : 'done',
    },
  ];
  if (state.judgment === 'pending' || state.judgment === 'failed') return stages;
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
      state: state.draft === 'pending' ? (commentAttemptsExhausted(state) ? 'stopped — attempts exhausted' : 'pending') : state.draft === 'invalid' ? 'invalid' : 'done',
    });
    stages.push({
      label: 'Native issue creation',
      state:
        state.effect === 'pending'
          ? (state.draft === 'pending' && commentAttemptsExhausted(state) ? 'not started' : 'pending')
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
    stages.push({label: 'Issue discussion feedback', state: state.effect === 'applied' || state.effect === 'verified' ? 'delivered' : commentAttemptsExhausted(state) ? 'stopped — retry needed' : 'delivery not confirmed'});
  }
  return stages;
}

export function commentSourceText(state: CommentPipelineState): string {
  const source = state.source;
  if (source === undefined) return '';
  const parts = [
    `About ${source.kind === 'record' ? 'recorded work' : 'a discussion post'} ${shortFeedID(source.id)}`,
  ];
  if (source.issue) parts.push(`issue #${source.issue}`);
  if (source.pr) parts.push(`pull request #${source.pr}`);
  if (source.ref) parts.push(`candidate ${source.ref}`);
  if (source.status) parts.push(`status ${source.status}`);
  return `${parts.join(' · ')}${source.summary ? ` — ${source.summary}` : ''}`;
}

// describeCommentOutcome projects one stored pipeline row to its visible
// outcome. Saved comment, assessment, draft and native effect stay separate
// observable stages; a confirmed issue links its native issue, while
// discussion-only, existing-work, clarification and failure outcomes each
// read exactly as recorded. Unknown states never read as success.
export function describeCommentOutcome(
  state: CommentPipelineState,
  repository: FeedRepositoryRef
): CommentOutcomeModel {
  const selectedExisting = state.existingIssue > 0 ? state.existingIssue : state.issue;
  const linkIssue = state.route === 'existing_work' ? selectedExisting : state.issue;
  const issueLink =
    linkIssue > 0 && repository.path !== undefined
      ? nativeIssueLink(repository.path, String(linkIssue))
      : undefined;
  const sourceText = commentSourceText(state);
  const outcome = (text: string, tone: CommentOutcomeTone, retryable: boolean): CommentOutcomeModel => ({
    commentId: state.commentId,
    text,
    tone,
    issueLink,
    retryable,
    sourceText,
    stages: commentStages(state),
    canRetry: state.canRetry,
    revision: state.revision,
    detail: state.detail,
    needsClarification: state.route === 'clarify',
  });
  if (commentAttemptsExhausted(state)) {
    if (state.judgment === 'pending' || state.judgment === 'failed')
      return outcome('Assessment stopped after its bounded failed attempts. Further assessment needs an explicit retry.', 'failed', false);
    if (state.route === 'new_issue' && state.draft === 'pending')
      return outcome('Issue authoring stopped after its bounded failed attempts. No issue draft was accepted.', 'failed', false);
    if (state.route === 'new_issue' && (state.effect === 'pending' || state.effect === 'failed'))
      return outcome('Native issue creation is unconfirmed after its bounded attempts. Inspect the original operation before retrying.', 'failed', false);
  }
  if (state.judgment === 'pending')
    return outcome(
      'Saved. JEV is assessing your comment.',
      'progress',
      true
    );
  if (state.judgment === 'failed')
    return outcome(
      'Assessment failed after its bounded attempts. Retrying recovers the original instead of duplicating it.',
      'failed',
      true
    );
  if (state.route === 'discussion')
    return outcome('Kept as discussion.', 'settled', false);
  if (state.route === 'clarify')
    return outcome('Waiting on clarification from the author before any further routing.', 'progress', false);
  if (state.route === 'existing_work') {
    if (state.effect === 'applied' || state.effect === 'verified')
      return outcome(`Feedback added to issue #${selectedExisting}.`, 'settled', false);
    if (selectedExisting <= 0 && state.effect === 'pending')
      return outcome(
        'Finding the issue for this feedback.',
        'progress',
        true
      );
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
  return outcome(
    `Issue #${state.issue} created.`,
    'settled',
    false
  );
}

export interface DiscussionModel {
  readonly showComposer?: boolean;
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
    submitAvailable: draft.trim() !== '',
    outcomes,
    intakeText:
      'Submitting asks JEV to route this comment; a selected new request reaches Product Lead authoring and validated native creation. Acknowledgements stay discussion-only.',
    discussionLink: repository.path === undefined || !entry.issue ? undefined : `${repository.path}/issues/${entry.issue}`,
  };
}

export function renderPostHeader(model: PostHeaderModel): TemplateResult {
  return html`<div class="soda-feed-header">
    <span class="soda-feed-avatar" aria-hidden="true">${model.avatarInitial}</span>
    <div>
      <p><strong>${model.name}</strong>${model.role !== model.name ? html` · ${model.role}` : ''} · ${model.project}</p>
      <p>${model.timeText} · ${model.statusText}</p>
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
        ${outcome.canRetry ? html`<button type="button" data-comment-revise=${outcome.commentId}>${outcome.needsClarification ? 'Answer clarification' : 'Revise or retry request'}</button>` : ''}
        <details><summary>Processing details</summary>${outcome.sourceText ? html`<p class="soda-feed-outcome-source">${outcome.sourceText}</p>` : ''}<ol class="soda-feed-outcome-stages" aria-label="Comment pipeline stages">
          ${outcome.stages.map((stage) => html`<li><strong>${stage.label}</strong> — ${stage.state}</li>`)}
        </ol></details>
      </div>`
    )}
    ${model.showComposer === false ? '' : html`<label
      >${model.label ?? 'Reply draft'}
      <textarea name="reply" data-feed-draft=${model.key} rows="3" maxlength="4000" .value=${live(model.draft)}></textarea>
    </label>
    <p>
      <button
        type="button"
        class="ui button"
        data-comment-submit=${model.key}
        ?disabled=${!model.submitAvailable}
        title="Submit persists the comment and starts its JEV assessment."
      >
        ${model.submitLabel ?? 'Submit reply'}
      </button>
      ${model.discussionLink ? html`<a href=${model.discussionLink}>Open issue discussion</a>` : ''}
    </p>`}
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
  denied: boolean
): FeedPostModel {
  return {
    header: describePostHeader(entry, repository),
    family: describeFeedFamily(entry, repository),
    plan: describePlanIssue(entry, repository),
    work: describeWorkEvidence(entry, repository),
    attention: describeDecisionAttention(entry, denied),
    discussion: describeDiscussion(entry, repository, draft, outcomes),
    postBody: body === undefined ? undefined : describePostBody(body),
  };
}

// renderFeedPost keeps the agreed anatomy: separate article boundary, the
// unboxed header immediately above its own bordered card, medium detail and
// source links inside. No timeline spine, no nested multi-role wrapper.
// A registered family section is the focal visual; discussion posts lead
// with their stored text and fallback entries keep the generic work
// section, so no entry loses its facts, failures or links.
export function renderFeedPost(model: FeedPostModel, replies: TemplateResult | string = ''): TemplateResult {
  const focal =
    model.family.family === 'discussion'
      ? ''
      : model.family.family === 'fallback'
        ? html`${renderFeedFamily(model.family)} ${renderWorkEvidence(model.work)}`
        : renderFeedFamily(model.family);
  return html`<article class="soda-feed-post" id=${'feed-' + model.header.key.replace(':', '-')} data-status=${model.family.statusClass}>
    ${renderPostHeader(model.header)}
    <div class="soda-feed-card">
      ${model.postBody ? renderPostBody(model.postBody) : ''} ${focal}
      ${model.attention.tone === 'notice' ? renderDecisionAttention(model.attention) : ''}
      ${model.discussion.outcomes.length ? renderDiscussion({...model.discussion, showComposer: false}) : ''}
      ${replies}
      <details class="soda-feed-replies"><summary>Reply</summary>${renderDiscussion({...model.discussion, outcomes: []})}</details>
    </div>
  </article>`;
}
