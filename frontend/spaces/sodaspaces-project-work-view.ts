import {html} from 'lit';
import './sodaspaces-factory-verification-allowance.js';
import {repeat} from 'lit/directives/repeat.js';
import type {SavedRequest, IntentPlanStatus} from './sodaspaces-intent-response.js';
import type {PreparedExtensionMount} from './soda-extension.js';

export interface ProjectWorkView {
  mode: 'project' | 'all-plans';
  repositoryName: string;
  repositoryNames: ReadonlyMap<string, string>;
  plans: readonly SavedRequest[];
  planStatuses: ReadonlyMap<string, IntentPlanStatus>;
  loaded: boolean;
  busy: boolean;
  error: string;
  more: boolean;
  disabled: boolean;
  verificationContext?: {transport: PreparedExtensionMount; repository: string} | undefined;
  newWork: () => void;
  resume: (intent: string) => void;
  refresh: () => void;
  loadMore: () => void;
}

// Work return replays the recorded plan/check state, never a saved-scope
// guess. The list entry carries only saved scope counts
// (effective_count/task_count derive from the current revision content, see
// internal/factory/intent.go Summary). The state label and action below derive
// from the authorized same-intent/revision owner GET plan status:
// completed + walkthroughs + clean check = Checked, completed + walkthroughs +
// blocking findings = Needs-correction, completed + walkthroughs without a
// recorded check = Plan-ready. Anything else stays neutral (Draft for drafts,
// Accepted/Closed for decided work). Reads are replay-only: a missing plan,
// a mismatched revision, an undecided run, or a failed read never claims a
// check and never starts planning.
export function workPlanState(plan: SavedRequest, status: IntentPlanStatus | undefined) {
  if (plan.state === 'accepted') return 'Accepted';
  if (plan.state !== 'draft') return 'Closed';
  if (!status || status.revision !== plan.revision) return 'Draft';
  if (status.state !== 'completed' || !status.walkthroughs) return 'Draft';
  if (!status.review || !status.review.consistency) return 'Plan-ready';
  return status.review.consistency.mismatches.length ? 'Needs-correction' : 'Checked';
}

function planState(plan: SavedRequest, statuses: ReadonlyMap<string, IntentPlanStatus>) {
  return workPlanState(plan, statuses.get(plan.intent));
}

function planAction(plan: SavedRequest, statuses: ReadonlyMap<string, IntentPlanStatus>) {
  const state = workPlanState(plan, statuses.get(plan.intent));
  if (state === 'Checked') return 'Review check';
  if (state === 'Needs-correction') return 'Resolve blockers';
  if (state === 'Plan-ready') return 'Check walkthrough';
  if (plan.state === 'draft') return 'Continue planning';
  if (plan.state === 'accepted') return 'View progress';
  return 'Review';
}

function repositoryName(plan: SavedRequest, names: ReadonlyMap<string, string>) {
  if (plan.repository === '0') return 'Not linked to a repository';
  return names.get(plan.repository) || 'Repository name unavailable';
}

function renderPlan(plan: SavedRequest, view: ProjectWorkView) {
  const state = planState(plan, view.planStatuses);
  return html`<article class="soda-work-item" data-intent-id=${plan.intent}>
    <div class="soda-work-item-heading">
      <span class="soda-work-state" data-state=${state.toLocaleLowerCase()}>${state}</span>
      ${
        view.mode === 'all-plans'
          ? html`<span class="soda-work-repository">${repositoryName(plan, view.repositoryNames)}</span>`
          : ''
      }
    </div>
    <h3>${plan.objective}</h3>
    <div class="soda-work-item-actions">
      <button
        type="button"
        class="ui button"
        ?disabled=${view.disabled}
        aria-label=${planAction(plan, view.planStatuses) + ': ' + plan.objective}
        @click=${() => view.resume(plan.intent)}
      >
        ${planAction(plan, view.planStatuses)} <span aria-hidden="true">→</span>
      </button>
    </div>
    ${
      plan.state === 'accepted' && view.verificationContext
        ? html`<details class="soda-verification-limits">
            <summary>Verification limits</summary>
            <soda-factory-verification-allowance
              .context=${{...view.verificationContext, intent: plan.intent}}
            ></soda-factory-verification-allowance>
          </details>`
        : ''
    }
  </article>`;
}

function renderEmpty(view: ProjectWorkView) {
  if (!view.loaded || view.plans.length || view.error) return '';
  return html`<div class="soda-work-empty">
    <h3>${view.mode === 'project' ? 'No plans linked to this project yet' : 'Your next idea starts here'}</h3>
    <p>
      ${
        view.mode === 'project'
          ? 'Describe a change to start a plan. Unfinished drafts are available in Saved plans.'
          : 'Describe what you want to change. You can save a draft and return to it here.'
      }
    </p>
  </div>`;
}

function renderListStatus(view: ProjectWorkView) {
  if (view.error) return html`<p class="soda-work-feedback" role="status">${view.error}</p>`;
  if (view.busy) return html`<p class="soda-work-feedback" role="status">Loading plans…</p>`;
  if (!view.loaded) return '';
  if (view.more)
    return html`<p class="soda-work-feedback">More plans are available. Showing ${view.plans.length} loaded plans.</p>`;
  return '';
}

export function renderProjectWork(view: ProjectWorkView) {
  const heading = view.mode === 'project' ? 'What would you like to change?' : 'Your plans';
  return html`<section class="soda-project-work" aria-label="Project work" aria-busy=${view.busy ? 'true' : 'false'}>
    <header class="soda-work-intro">
      <div>
        <p class="soda-work-eyebrow">${view.mode === 'project' ? 'Work & results' : 'Saved plans'}</p>
        <h2 tabindex="-1">${heading}</h2>
        <p>
          ${
            view.mode === 'project'
              ? `Plan the next change for ${view.repositoryName}. Follow its progress and review the results here.`
              : 'Pick up a draft, follow accepted work, or review a closed plan.'
          }
        </p>
      </div>
      <button type="button" class="ui primary button" ?disabled=${view.disabled} @click=${view.newWork}>
        <span aria-hidden="true">＋</span> New work
      </button>
    </header>
    <div class="soda-work-list-heading">
      <h3>${view.mode === 'project' ? 'Plans for this project' : 'Saved plans'}</h3>
      <button type="button" class="ui button" ?disabled=${view.disabled || view.busy} @click=${view.refresh}>
        Refresh plans
      </button>
    </div>
    <p class="soda-work-help">
      These are your saved plans. Accepted means the direction was agreed; progress shows the recorded results.
    </p>
    ${renderListStatus(view)} ${renderEmpty(view)}
    <div class="soda-work-list">
      ${repeat(
        view.plans,
        (plan) => plan.intent,
        (plan) => renderPlan(plan, view)
      )}
    </div>
    ${
      view.more
        ? html`<button
            type="button"
            class="ui button soda-work-more"
            ?disabled=${view.disabled || view.busy}
            @click=${view.loadMore}
          >
            Load more plans
          </button>`
        : ''
    }
  </section>`;
}
