import {html} from 'lit';
import type {TemplateResult} from 'lit';
import {renderEnvironment, renderProjectOS, renderOSObservation} from './sodaspaces-environment-view.js';
import {renderConnection, renderKeys, renderForgejoKeys} from './sodaspaces-project-view.js';
import {renderNetwork, renderNetworkSelection} from './sodaspaces-network.js';
import type {CreationProfile, Detail, Environment, OSObservation} from './sodaspaces-project-response.js';
import type {KeyPreview, ProfileKeys, SavedKey} from './sodaspaces-keys-response.js';
import type {ProjectNetwork, ProjectOptions} from '../tailnet/soda-tailnet-response.js';

export const views = ['environment', 'access', 'network'] as const;
export type View = (typeof views)[number];
export type Lifecycle = {
  running: boolean;
  boot: boolean;
};

function viewTabLabel(view: View, presentation: string): string {
  if (view === 'environment' && presentation === 'settings') return 'Overview';
  return (view[0]?.toUpperCase() || '') + view.slice(1);
}

// Stateless presentation: the concrete project owner admits every command and
// retains the original target, confirmation and asynchronous request lifetime.
function renderLifecycle(
  lifecycle: Lifecycle | undefined,
  blocked: boolean,
  confirmed: boolean,
  start: (event: MouseEvent) => void,
  stop: (event: MouseEvent) => void,
  confirm: (checked: boolean) => void
): TemplateResult {
  const stopped = !lifecycle?.running && !lifecycle?.boot;
  return html`
    <fieldset ?hidden=${!lifecycle}>
      <legend>Shared environment</legend>
      <p>${lifecycleCaption(lifecycle)}</p>
      <button
        type="button"
        class="ui primary button"
        ?hidden=${!!lifecycle?.running && lifecycle.boot}
        ?disabled=${blocked}
        @click=${start}
      >
        Start
      </button>
      <button type="button" class="ui button danger" ?hidden=${stopped} ?disabled=${blocked} @click=${stop}>
        Stop
      </button>
      <label ?hidden=${stopped}>
        <input type="checkbox" .checked=${confirmed} @change=${(event: Event) => confirmLifecycle(event, confirm)} />
        I understand Stop interrupts everyone’s SSH, terminals and workloads, and disables next-boot start.
      </label>
    </fieldset>
  `;
}

function confirmLifecycle(event: Event, confirm: (checked: boolean) => void) {
  if (event.target instanceof HTMLInputElement) confirm(event.target.checked);
}

function lifecycleCaption(lifecycle: Lifecycle | undefined): string {
  if (!lifecycle) return '';
  return `Running: ${lifecycle.running ? 'yes' : 'no'}; starts on host boot: ${lifecycle.boot ? 'yes' : 'no'}. Start restores boot start; Stop disables it.`;
}

export interface SettingsViewInput {
  readBindingSettings: () => boolean;
  readBindingPage: () => boolean;
  readBindingRepository: () => string;
  readRepository: () => string;
  readRepositoryURL: () => string;
  readSelected: () => View;
  readPresentation: () => string;
  canCreate: () => boolean;
  isBusy: () => boolean;
  isStale: () => boolean;
  isBlocked: () => boolean;
  isRunning: () => boolean;
  readEnvironment: () => Environment | undefined;
  readDetail: () => Detail | undefined;
  readDetailEnvironment: () => Environment | undefined;
  readDetailLogin: () => string | undefined;
  readExecutionAllowed: () => boolean;
  readEnvironmentAdmin: () => boolean;
  readSavedKeys: () => SavedKey[] | undefined;
  readProfileKeys: () => ProfileKeys | undefined;
  readKeyPreview: () => KeyPreview | undefined;
  readUseSavedKeys: () => boolean;
  readDraft: () => string;
  readEmptyConfirmed: () => boolean;
  readStopConfirmed: () => boolean;
  readLifecycle: () => Lifecycle | undefined;
  readConnection: () => {command: string; fingerprint: string} | undefined;
  readProfiles: () => CreationProfile[];
  readSelectedProfile: () => string;
  readNetwork: () => ProjectNetwork | undefined;
  readNetworkOptions: () => ProjectOptions | undefined;
  isNetworkEnabled: () => boolean;
  readNetworkNotice: () => string;
  readNetworkConfirmed: () => boolean;
  readObservedOS: () => OSObservation | undefined;
  readOsStatus: () => string;
  selectView: (view: View) => void;
  tabKey: (event: KeyboardEvent, view: View) => void;
  setCreateNetworkEnabled: (enabled: boolean) => void;
  selectProfile: (value: string) => void;
  setUseSavedKeys: (checked: boolean) => void;
  requestRefresh: (event: Event) => void;
  requestCreate: (event: Event) => void;
  requestJoin: (event: Event) => void;
  requestStart: (event: Event) => void;
  requestStop: (event: Event) => void;
  setStopConfirmed: (checked: boolean) => void;
  requestInspectOS: (event: Event) => void;
  reloadPage: (event: Event) => void;
  copyConnection: () => void;
  removeSavedKey: (event: Event, key: SavedKey) => void;
  setDraft: (value: string) => void;
  requestSaveKey: (event: Event) => void;
  requestReviewKeys: (event: Event) => void;
  setConfirmEmpty: (checked: boolean) => void;
  requestApplyKeys: (event: Event) => void;
  reviewProfileKeysPage: (page: number) => void;
  selectForgejoKey: (key: string) => void;
  setNetworkConfirmed: (confirmed: boolean) => void;
  changeProjectNetwork: (event: Event, action: 'enable' | 'disable' | 'retry') => void;
}

function canJoin(input: SettingsViewInput) {
  const detail = input.readDetail();
  return (
    !!detail?.environment.provisioned &&
    detail.execution_allowed &&
    !detail.login &&
    input.isRunning() &&
    (!input.readUseSavedKeys() || !!input.readSavedKeys())
  );
}

export function renderRepositoryContext(input: SettingsViewInput) {
  if (!input.readBindingSettings() || !input.readRepositoryURL()) return html``;
  return html`<div class="soda-repository-context">
    <h2>${input.readRepository()}</h2>
    <nav aria-label="Repository destinations">
      <a href=${input.readRepositoryURL() + '/settings'}>Native repository settings</a> ·
      <a href=${input.readRepositoryURL() + '#sodaspaces'}>Open repository and workspace drawer</a>
    </nav>
    <p>
      Project OS selection affects only creation. Existing roots cannot change distribution or interface here. Create,
      Join and Start remain separate explicit actions.
    </p>
  </div>`;
}

export function renderViewTabs(input: SettingsViewInput) {
  return html`<div class="soda-spaces-tabs" role="tablist" aria-label="Workspace views">
    ${views.map(
      (view) => html` <button
        type="button"
        class="ui basic button"
        data-view=${view}
        role="tab"
        aria-controls=${'soda-project-' + input.readBindingRepository() + '-' + view}
        aria-selected=${input.readSelected() === view ? 'true' : 'false'}
        tabindex=${input.readSelected() === view ? 0 : -1}
        @click=${() => input.selectView(view)}
        @keydown=${(e: KeyboardEvent) => input.tabKey(e, view)}
      >
        ${viewTabLabel(view, input.readPresentation())}
      </button>`
    )}
  </div>`;
}

function renderCreateNetworkSelection(input: SettingsViewInput) {
  if (!input.canCreate()) return html``;
  return renderNetworkSelection(input.readNetworkOptions(), input.isNetworkEnabled(), input.isBlocked(), (enabled) =>
    input.setCreateNetworkEnabled(enabled)
  );
}

function networkSummaryText(input: SettingsViewInput) {
  const network = input.readNetwork();
  if (!network) return 'not observed';
  return (network.enabled ? 'managed' : 'Off') + ' · ' + network.state;
}

function renderNetworkSummary(input: SettingsViewInput) {
  if (!input.readEnvironment()) return html``;
  return html`<p data-project-network-summary>Tailnet: ${networkSummaryText(input)}</p>`;
}

function renderObservedOS(input: SettingsViewInput) {
  if (!input.readEnvironment()) return html``;
  return renderOSObservation(input.readObservedOS(), input.readOsStatus(), input.isBlocked(), (event) =>
    input.requestInspectOS(event)
  );
}

export function renderEnvironmentView(input: SettingsViewInput) {
  return html`<section
    id=${'soda-project-' + input.readBindingRepository() + '-environment'}
    class="soda-spaces-view"
    role="tabpanel"
    aria-label="Environment"
    ?hidden=${input.readSelected() !== 'environment'}
  >
    ${renderProjectOS(
      input.readProfiles(),
      input.readSelectedProfile(),
      input.readDetailEnvironment(),
      input.isBlocked(),
      (value) => input.selectProfile(value)
    )}
    ${renderCreateNetworkSelection(input)} ${renderNetworkSummary(input)} ${renderObservedOS(input)}
    ${renderEnvironment(
      {
        busy: input.isBusy(),
        stale: input.isStale(),
        blocked: input.isBlocked(),
        canCreate: input.canCreate(),
        canJoin: canJoin(input),
        sshKeys: input.readSavedKeys()?.map((key) => key.fingerprint) || [],
        useSavedKeys: input.readUseSavedKeys(),
      },
      {
        selectSSH: (checked) => input.setUseSavedKeys(checked),
        refresh: (event) => input.requestRefresh(event),
        reload: (event) => input.reloadPage(event),
        create: (event) => input.requestCreate(event),
        join: (event) => input.requestJoin(event),
      },
      renderLifecycle(
        input.readLifecycle(),
        input.isBlocked(),
        input.readStopConfirmed(),
        (event) => input.requestStart(event),
        (event) => input.requestStop(event),
        (checked) => input.setStopConfirmed(checked)
      )
    )}
  </section>`;
}

export function renderAccessView(input: SettingsViewInput) {
  return html`<section
    id=${'soda-project-' + input.readBindingRepository() + '-access'}
    class="soda-spaces-view"
    role="tabpanel"
    aria-label="Access"
    ?hidden=${input.readSelected() !== 'access'}
  >
    ${renderConnection(
      input.readConnection(),
      input.readBindingRepository(),
      input.readBindingPage(),
      input.isBlocked(),
      () => {
        input.copyConnection();
      }
    )}
    ${renderKeys(
      {
        saved: input.readSavedKeys(),
        preview: input.readKeyPreview(),
        joined: !!input.readDetailLogin(),
        executionAllowed: input.readExecutionAllowed(),
        running: input.isRunning(),
        blocked: input.isBlocked(),
        draft: input.readDraft(),
        emptyConfirmed: input.readEmptyConfirmed(),
      },
      {
        remove: (event, key) => input.removeSavedKey(event, key),
        draft: (value) => input.setDraft(value),
        save: (event) => input.requestSaveKey(event),
        review: (event) => input.requestReviewKeys(event),
        confirmEmpty: (checked) => input.setConfirmEmpty(checked),
        apply: (event) => input.requestApplyKeys(event),
      }
    )}
    ${renderForgejoKeyReview(input)}
  </section>`;
}

function renderForgejoKeyReview(input: SettingsViewInput) {
  if (!input.readSavedKeys()) return html``;
  return renderForgejoKeys(
    input.readProfileKeys(),
    input.isBlocked(),
    (page) => {
      input.reviewProfileKeysPage(page);
    },
    (key) => input.selectForgejoKey(key)
  );
}

function renderTailnetSSH(input: SettingsViewInput) {
  const network = input.readNetwork();
  const connection = input.readConnection();
  const login = input.readDetailLogin();
  if (network?.state !== 'connected' || !connection || !login) return html``;
  return html`<fieldset>
    <legend>Own-account Tailnet SSH</legend>
    <input readonly aria-label="Tailnet SSH command" .value=${'ssh ' + login + '@' + network.addresses[0]} />
    <p>
      Ed25519 host-key fingerprint: ${connection.fingerprint}. Same project account and host key as LAN SSH; no
      Tailscale SSH or automatic authentication.
    </p>
  </fieldset>`;
}

export function renderNetworkView(input: SettingsViewInput) {
  return html`<section
    id=${'soda-project-' + input.readBindingRepository() + '-network'}
    class="soda-spaces-view"
    role="tabpanel"
    aria-label="Network"
    ?hidden=${input.readSelected() !== 'network'}
  >
    ${renderNetwork(
      input.readNetwork(),
      input.readNetworkNotice(),
      input.readEnvironmentAdmin(),
      input.isBlocked(),
      input.readNetworkConfirmed(),
      (confirmed) => input.setNetworkConfirmed(confirmed),
      (event, action) => input.changeProjectNetwork(event, action)
    )}
    ${renderTailnetSSH(input)}
  </section>`;
}
