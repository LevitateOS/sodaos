import {html} from 'lit';
import {renderWorkspaceIntro} from './sodaspaces-workspace-view.js';
import {renderProjectOS} from './sodaspaces-environment-view.js';
import {renderNetworkSelection} from './sodaspaces-network.js';
import type {CreationProfile, Detail, Environment} from './sodaspaces-project-response.js';
import type {ProjectOptions} from '../tailnet/soda-tailnet-response.js';

export interface JourneyViewInput {
  isStale: () => boolean;
  isBusy: () => boolean;
  isBlocked: () => boolean;
  readBusyAttr: () => 'true' | 'false';
  readEnvironment: () => Environment | undefined;
  readDetail: () => Detail | undefined;
  canCreate: () => boolean;
  isRunning: () => boolean;
  hasJoinFailed: () => boolean;
  needsJoinCheck: () => boolean;
  isMutationPending: () => boolean;
  needsOutcomeAttention: () => boolean;
  readOutcome: () => string;
  readStatus: () => string;
  readRepositoryName: () => string;
  readBindingRepository: () => string;
  hasLifecycle: () => boolean;
  readProfiles: () => CreationProfile[];
  readSelectedProfile: () => string;
  readNetworkOptions: () => ProjectOptions | undefined;
  isNetworkEnabled: () => boolean;
  needsNetworkReview: () => boolean;
  requestRefresh: (event: Event) => void;
  refreshNow: () => void;
  requestJoin: (event: Event) => void;
  requestStart: (event: Event) => void;
  requestCreate: (event: Event) => void;
  requestRepositoryChange: (event: Event) => void;
  clearNetworkReview: (event: Event) => void;
  selectProfile: (value: string) => void;
  setJourneyNetworkEnabled: (enabled: boolean) => void;
}

function journeyBlocked(input: JourneyViewInput) {
  return input.isStale() || (!input.readEnvironment() && !input.canCreate());
}

function journeyJoinReady(input: JourneyViewInput) {
  const detail = input.readDetail();
  return (
    !input.isStale() &&
    !!detail?.environment.provisioned &&
    detail.execution_allowed &&
    input.isRunning() &&
    !detail.login &&
    !detail.authority_unavailable &&
    !detail.native_unavailable
  );
}

export function renderJourney(input: JourneyViewInput) {
  if (journeyBlocked(input)) return renderJourneyUnavailable(input);
  if (journeyJoinReady(input)) return input.hasJoinFailed() ? renderJourneyJoinFailed(input) : renderJourneyJoin(input);
  if (input.readEnvironment()) return renderJourneyExisting(input);
  return renderJourneyConfigure(input);
}

function journeyUnavailableHeading(input: JourneyViewInput) {
  if (input.isStale()) return 'Project access changed';
  if (input.isBusy()) return 'Checking project';
  return 'Project status unavailable';
}

function journeyUnavailableDescription(input: JourneyViewInput) {
  if (input.isStale()) return html`Reload Spaces to check your current access.`;
  if (input.isBusy()) return html`Checking your account and project.`;
  return html`Current project access or runtime state could not be confirmed.`;
}

function journeyUnavailableAction(input: JourneyViewInput) {
  if (input.isBusy()) return html``;
  if (input.isStale())
    return html`<button class="ui primary button" @click=${() => location.reload()}>Reload Spaces</button>`;
  return html`<button class="ui primary button" @click=${() => input.refreshNow()}>Refresh status</button>`;
}

function renderJourneyUnavailable(input: JourneyViewInput) {
  return html`<section
    data-project-controls
    class="soda-spaces-controls soda-project-journey soda-ready-project"
    aria-busy=${input.readBusyAttr()}
  >
    ${renderWorkspaceIntro({
      kind: input.isBusy() ? 'loading' : 'unavailable',
      heading: journeyUnavailableHeading(input),
      description: journeyUnavailableDescription(input),
      action: journeyUnavailableAction(input),
      helper: html`No operation is repeated when you refresh.`,
      feedback: html`<p role="status">${input.needsOutcomeAttention() ? input.readOutcome() : ''}</p>`,
    })}
  </section>`;
}

function joinFailedHelper(input: JourneyViewInput) {
  if (input.needsJoinCheck()) return html`Checking status won’t try to join again.`;
  return html`You haven’t joined this project yet.`;
}

function joinFailedFeedback(input: JourneyViewInput) {
  if (!input.needsJoinCheck() && !input.isBusy())
    return html`<button class="ui button soda-quiet-action" @click=${input.requestJoin}>Try joining again</button>`;
  return html``;
}

function renderJourneyJoinFailed(input: JourneyViewInput) {
  return html`<section
    data-project-controls
    class="soda-spaces-controls soda-project-journey soda-ready-project"
    aria-busy=${input.readBusyAttr()}
  >
    ${renderWorkspaceIntro({
      kind: 'unavailable',
      heading: 'Couldn’t join project',
      description: html`<span role="status">${input.readOutcome()}</span>`,
      action: html`<button class="ui primary button" ?disabled=${input.isBlocked()} @click=${input.requestRefresh}>
        ${input.isBusy() ? 'Checking join status…' : 'Check join status'}
      </button>`,
      helper: joinFailedHelper(input),
      feedback: joinFailedFeedback(input),
    })}
  </section>`;
}

function joinAction(input: JourneyViewInput) {
  const label = input.isMutationPending() ? 'Joining project…' : 'Join project';
  return html`<button class="ui primary button" ?disabled=${input.isBlocked()} @click=${input.requestJoin}>
    ${label}
  </button>`;
}

function joinFeedback(input: JourneyViewInput) {
  const tone = input.isMutationPending() ? 'pending' : 'warning';
  const text = input.isMutationPending()
    ? 'Setting up your project account…'
    : input.needsOutcomeAttention()
      ? input.readOutcome()
      : '';
  return html`<div class="soda-intro-feedback soda-feedback" data-tone=${tone} role="status">${text}</div>
    ${joinRefreshAction(input)}`;
}

function joinRefreshAction(input: JourneyViewInput) {
  if (!input.needsOutcomeAttention() || input.isBusy()) return html``;
  return html`<button class="ui button soda-quiet-action" ?disabled=${input.isBlocked()} @click=${input.requestRefresh}>
    Refresh status
  </button>`;
}

function renderJourneyJoin(input: JourneyViewInput) {
  return html`<section
    data-project-controls
    data-repository-id=${input.readBindingRepository()}
    data-environment-id=${input.readEnvironment()?.id || ''}
    class="soda-spaces-controls soda-project-journey soda-ready-project"
    aria-busy=${input.readBusyAttr()}
  >
    ${renderWorkspaceIntro({
      kind: 'project',
      heading: 'Project created',
      description: html`Join ${input.readRepositoryName()} to set up your personal account<span
          >in this persistent human project.</span
        >`,
      action: joinAction(input),
      helper: html`Then you can open your first browser terminal.`,
      feedback: joinFeedback(input),
    })}
  </section>`;
}

function journeyRuntimeUnavailable(input: JourneyViewInput) {
  const detail = input.readDetail();
  return !detail || detail.authority_unavailable || detail.native_unavailable || !detail.observed;
}

function executionDenied(input: JourneyViewInput) {
  const detail = input.readDetail();
  return !!detail && !detail.authority_unavailable && !detail.execution_allowed;
}

function journeyExistingKind(input: JourneyViewInput, accountReady: boolean, stopped: boolean) {
  if (input.isBusy()) return 'loading' as const;
  if (accountReady) return 'terminal' as const;
  if (stopped) return 'stopped' as const;
  return 'unavailable' as const;
}

function journeyExistingHeading(input: JourneyViewInput, accountReady: boolean, incomplete: boolean, stopped: boolean) {
  if (accountReady) return 'Your account is ready';
  if (executionDenied(input)) return 'Repository write access required';
  if (incomplete) return 'Project needs inspection';
  if (stopped) return 'Project not ready';
  return 'Project status unavailable';
}

function journeyExistingDescription(
  input: JourneyViewInput,
  accountReady: boolean,
  incomplete: boolean,
  stopped: boolean
) {
  if (accountReady) return html`Your project account is ready for a browser terminal.`;
  if (executionDenied(input))
    return html`You can inspect this project. Joining and terminal access require repository write permission.`;
  if (incomplete) return html`Project setup is incomplete. Ask the operator to inspect this project.`;
  if (stopped) return html`This project is stopped. Start it before joining or opening a terminal.`;
  return html`Current project access or runtime state could not be confirmed.`;
}

function journeyStartButton(input: JourneyViewInput, stopped: boolean) {
  if (!stopped || !input.hasLifecycle()) return html``;
  const label = input.isMutationPending() ? 'Starting project…' : 'Start project';
  return html`<button class="ui primary button" ?disabled=${input.isBlocked()} @click=${input.requestStart}>
    ${label}
  </button>`;
}

function journeyExistingAction(input: JourneyViewInput, stopped: boolean) {
  const refreshClass = stopped && input.hasLifecycle() ? 'ui button soda-quiet-action' : 'ui primary button';
  const refreshLabel = input.isBusy() && !input.isMutationPending() ? 'Checking status…' : 'Refresh status';
  return html`${journeyStartButton(input, stopped)}<button
      class=${refreshClass}
      ?disabled=${input.isBlocked()}
      @click=${input.requestRefresh}
    >
      ${refreshLabel}
    </button>`;
}

function journeyExistingHelper(input: JourneyViewInput, incomplete: boolean, stopped: boolean) {
  if (incomplete) return html`Do not recreate a reserved project.`;
  if (stopped && !input.hasLifecycle()) return html`A project administrator must start this project.`;
  return html`Refreshing checks the existing project without repeating an operation.`;
}

function journeyExistingFlags(input: JourneyViewInput) {
  const unavailable = journeyRuntimeUnavailable(input);
  return {
    incomplete: journeyIncomplete(input),
    stopped: journeyStopped(input, unavailable),
    accountReady: journeyAccountReady(input, unavailable),
  };
}

function journeyIncomplete(input: JourneyViewInput) {
  const detail = input.readDetail();
  return !!detail && !detail.environment.provisioned;
}

function journeyStopped(input: JourneyViewInput, unavailable: boolean) {
  if (unavailable) return false;
  return input.readDetail()?.observed?.running === false;
}

function journeyAccountReady(input: JourneyViewInput, unavailable: boolean) {
  if (unavailable || !input.isRunning()) return false;
  const detail = input.readDetail();
  return !!detail?.login && detail.execution_allowed;
}

function renderJourneyExisting(input: JourneyViewInput) {
  const flags = journeyExistingFlags(input);
  return html`<section
    data-project-controls
    data-repository-id=${input.readBindingRepository()}
    data-environment-id=${input.readEnvironment()?.id || ''}
    class="soda-spaces-controls soda-project-journey soda-ready-project"
    aria-busy=${input.readBusyAttr()}
  >
    ${renderWorkspaceIntro({
      kind: journeyExistingKind(input, flags.accountReady, flags.stopped),
      heading: journeyExistingHeading(input, flags.accountReady, flags.incomplete, flags.stopped),
      description: journeyExistingDescription(input, flags.accountReady, flags.incomplete, flags.stopped),
      action: journeyExistingAction(input, flags.stopped),
      helper: journeyExistingHelper(input, flags.incomplete, flags.stopped),
      feedback: html`<div class="soda-intro-feedback soda-feedback" data-tone="warning" role="status">
        ${input.needsOutcomeAttention() ? input.readOutcome() : ''}
      </div>`,
    })}
  </section>`;
}

function configureReady(input: JourneyViewInput) {
  return (
    input.canCreate() && !input.isBusy() && !input.isStale() && !input.readOutcome() && !input.needsNetworkReview()
  );
}

function renderConfigureNetwork(input: JourneyViewInput) {
  const options = input.readNetworkOptions();
  if (!options?.available) return html``;
  return renderNetworkSelection(
    options,
    input.isNetworkEnabled(),
    input.isBlocked(),
    (enabled) => input.setJourneyNetworkEnabled(enabled),
    'Enable project Tailnet'
  );
}

function renderNetworkReviewNotice(input: JourneyViewInput) {
  if (!input.needsNetworkReview()) return html``;
  return html`<p role="alert">Network availability changed. Review the option before creating.</p>
    <button class="ui button" ?disabled=${input.isBlocked()} @click=${input.clearNetworkReview}>
      Use without Tailnet
    </button>`;
}

function renderStaleReloadNote(input: JourneyViewInput) {
  if (!input.isStale()) return html``;
  return html`<p>Access changed. Reload Spaces to reconnect; no operation will be repeated.</p>`;
}

function createDisabled(input: JourneyViewInput) {
  return input.isBlocked() || !input.canCreate() || input.needsNetworkReview();
}

function configureStatusHidden(input: JourneyViewInput) {
  return configureReady(input) || input.isMutationPending();
}

function pendingTone(pending: boolean) {
  return pending ? 'pending' : 'warning';
}

function renderCreateAction(input: JourneyViewInput) {
  const label = input.isMutationPending() ? 'Creating project…' : 'Create project';
  return html`<button class="ui primary button" ?disabled=${createDisabled(input)} @click=${input.requestCreate}>
    ${label}
  </button>`;
}

function renderConfigureFeedback(input: JourneyViewInput) {
  return html`<p
      class="soda-feedback"
      data-tone=${pendingTone(input.isBusy())}
      role="status"
      ?hidden=${configureStatusHidden(input)}
    >
      ${input.readStatus()}
    </p>
    <p class="soda-feedback" data-tone=${pendingTone(input.isMutationPending())} role="status">
      ${input.isMutationPending() ? 'Creating your project…' : input.readOutcome()}
    </p>
    <button
      class="ui button soda-quiet-action"
      ?hidden=${configureReady(input) || input.isBusy()}
      ?disabled=${input.isBlocked()}
      @click=${input.requestRefresh}
    >
      Refresh status
    </button>`;
}

function renderJourneyConfigure(input: JourneyViewInput) {
  return html`<section
    data-project-controls
    data-repository-id=${input.readBindingRepository()}
    data-environment-id=""
    class="soda-spaces-controls soda-project-journey"
    aria-busy=${input.readBusyAttr()}
  >
    <header class="soda-setup-step-heading">
      <p class="soda-setup-eyebrow">New project</p>
      <h2 tabindex="-1">Configure project</h2>
      <p>Choose the system for your project.</p>
    </header>
    <div class="soda-configuration-fields">
      <div class="soda-configuration-repository">
        <p>Repository</p>
        <div class="soda-config-repository">
          <span class="soda-journey-icon soda-repository-icon" aria-hidden="true"></span
          ><span class="soda-config-repository-name"
            >${input.readRepositoryName() || 'Loading repository…'}<small>Forgejo</small></span
          ><button
            class="ui button soda-quiet-action"
            aria-label="Change repository"
            ?disabled=${input.isBlocked()}
            @click=${input.requestRepositoryChange}
          >
            Change
          </button>
        </div>
      </div>
      ${renderProjectOS(
        input.readProfiles(),
        input.readSelectedProfile(),
        undefined,
        input.isBlocked(),
        (value) => input.selectProfile(value),
        'configure'
      )}
      ${renderConfigureNetwork(input)} ${renderNetworkReviewNotice(input)}
    </div>
    <div class="soda-setup-actions">${renderCreateAction(input)}</div>
    ${renderConfigureFeedback(input)} ${renderStaleReloadNote(input)}
  </section>`;
}
