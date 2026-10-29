// Settings → SodaOS → Environment (M19): one phase at a time from
// soda-settings-environment-machine.ts (loading, absent, creating, stopped,
// starting, running, stopping, unknown), the one next action, details, and
// Stop behind a confirmation. Joining and SSH keys live in Terminals; private
// networking in Network. The project controls own every request.
import {html, nothing} from 'lit';
import type {TemplateResult} from 'lit';
import type {SettingsViewInput} from './sodaspaces-project-settings-view.js';
import {supportMessage} from './sodaspaces-terminal-error.js';
import {environmentNotice, environmentPhase} from './soda-settings-environment-machine.js';
import type {EnvironmentPhase} from './soda-settings-environment-machine.js';

type State = EnvironmentPhase['kind'];

function phaseOf(input: SettingsViewInput): EnvironmentPhase {
  const lifecycle = input.readLifecycle();
  return environmentPhase({
    reads: input.readReads(),
    hasEnvironment: !!input.readEnvironment(),
    running: lifecycle ? lifecycle.running : input.readDetail() ? input.isRunning() : undefined,
    canCreate: input.canCreate(),
    operation: input.readOperation(),
    result: input.readOperationResult(),
  });
}

function elapsed(since: number): string {
  const total = Math.max(0, Math.floor((Date.now() - since) / 1000));
  return total < 60 ? `${total}s` : `${Math.floor(total / 60)}m ${String(total % 60).padStart(2, '0')}s`;
}

function progress(title: string, since: number, text: string): TemplateResult {
  return html`<div class="soda-env-progress" role="status">
    <span class="soda-env-spinner" aria-hidden="true"></span>
    <div>
      <p><strong>${title}</strong> · ${elapsed(since)}</p>
      <p class="soda-env-note">${text}</p>
    </div>
  </div>`;
}

function link(input: SettingsViewInput, which: 'terminals' | 'network', label: string, primary = false) {
  const links = input.readPlaceLinks();
  if (!links) return nothing;
  const href = links[which];
  return html`<a
    class=${primary ? 'ui primary button' : 'soda-env-link'}
    href=${href}
    @click=${(event: Event) => {
      event.preventDefault();
      links.navigate(href);
    }}
    >${label}</a
  >`;
}

const badges: Record<State, {label: string; tone: string}> = {
  loading: {label: 'Checking…', tone: 'quiet'},
  absent: {label: 'Not created', tone: 'quiet'},
  creating: {label: 'Creating…', tone: 'busy'},
  stopped: {label: 'Stopped', tone: 'warn'},
  starting: {label: 'Starting…', tone: 'busy'},
  running: {label: 'Running', tone: 'good'},
  stopping: {label: 'Stopping…', tone: 'busy'},
  unknown: {label: 'Status unavailable', tone: 'warn'},
};

function badge(value: State): TemplateResult {
  const {label, tone} = badges[value];
  return html`<span class="soda-env-pill" data-tone=${tone}>${label}</span>`;
}

function notice(input: SettingsViewInput): TemplateResult | typeof nothing {
  const current = environmentNotice(input.readOperationResult());
  if (!current) return nothing;
  return html`<div
    class="soda-env-notice"
    data-tone=${current.tone}
    role=${current.tone === 'bad' ? 'alert' : 'status'}
  >
    <p>${current.text}</p>
    <button type="button" class="soda-env-link" aria-label="Dismiss" @click=${() => input.dismissOperationResult()}>
      Dismiss
    </button>
  </div>`;
}

function createForm(input: SettingsViewInput): TemplateResult {
  const profiles = input.readProfiles();
  const options = input.readNetworkOptions();
  const blocked = input.isBlocked();
  return html`<div class="soda-env-form">
    ${
      profiles.length > 1
        ? html`<label class="soda-env-field"
            >System
            <select
              .value=${input.readSelectedProfile()}
              ?disabled=${blocked}
              @change=${(event: Event) => input.selectProfile((event.target as HTMLSelectElement).value)}
            >
              ${profiles.map(
                (profile) =>
                  html`<option value=${profile.id} ?selected=${profile.id === input.readSelectedProfile()}>
                    Rocky Linux ${profile.version} · ${profile.architecture}
                  </option>`
              )}
            </select></label
          >`
        : profiles[0]
          ? html`<p class="soda-env-note">System: Rocky Linux ${profiles[0].version} · ${profiles[0].architecture}</p>`
          : nothing
    }
    ${
      options?.available
        ? html`<label class="soda-env-check"
            ><input
              type="checkbox"
              .checked=${input.isNetworkEnabled()}
              ?disabled=${blocked}
              @change=${(event: Event) => input.setCreateNetworkEnabled((event.target as HTMLInputElement).checked)}
            />
            <span
              >Join the private network${options.tailnet ? ` (${options.tailnet})` : ''}
              <span class="soda-env-note"
                >Reach it from your other devices over Tailscale. You can change this later.</span
              ></span
            ></label
          >`
        : nothing
    }
    <button
      type="button"
      class="ui primary button"
      ?disabled=${blocked}
      @click=${(event: Event) => input.requestCreate(event)}
    >
      Create environment
    </button>
  </div>`;
}

function summary(input: SettingsViewInput, phase: EnvironmentPhase): TemplateResult | typeof nothing {
  const admin = input.readEnvironmentAdmin();
  const blocked = input.isBlocked();
  switch (phase.kind) {
    case 'loading':
      return html`<p class="soda-env-lead" role="status">Checking this repository’s environment…</p>`;
    case 'absent':
      if (phase.canCreate)
        return html`<p class="soda-env-lead">
            Create a Linux environment for this repository. Everyone with write access can open terminals and run agents
            in it, and its files are kept when it stops.
          </p>
          ${createForm(input)}`;
      return html`<p class="soda-env-lead">
        This repository has no development environment. Only the repository owner can create one; organization
        repositories are not supported yet.
      </p>`;
    case 'creating':
      return progress(
        'Creating the environment',
        phase.since,
        'Preparing the Linux system and its storage. This can take a minute or two; keep this page open to see the result.'
      );
    case 'starting':
      return progress(
        'Starting the environment',
        phase.since,
        'Booting the system. Terminals become available once it is running.'
      );
    case 'stopping':
      return progress(
        'Stopping the environment',
        phase.since,
        'Ending terminals and SSH sessions and shutting the system down. Files are kept.'
      );
    case 'unknown':
      return html`<p class="soda-env-lead">The environment’s status could not be read.</p>
        <button
          type="button"
          class="ui basic button"
          ?disabled=${blocked}
          @click=${(event: Event) => input.requestRefresh(event)}
        >
          Try again
        </button>`;
    case 'stopped':
      return html`<p class="soda-env-lead">Stopped. Files are kept; nobody can open terminals until it starts again.</p>
        ${
          admin
            ? html`<button
                type="button"
                class="ui primary button"
                ?disabled=${blocked}
                @click=${(event: Event) => input.requestStart(event)}
              >
                Start environment
              </button>`
            : html`<p class="soda-env-note">A repository administrator can start it.</p>`
        }`;
    case 'running': {
      const lifecycle = input.readLifecycle();
      return html`<p class="soda-env-lead">
          Running. Members with write access open terminals and agent sessions in Terminals.
          ${lifecycle && !lifecycle.boot ? html`<br /><span class="soda-env-note">It will not start again by itself after the server restarts.</span>` : nothing}
        </p>
        ${link(input, 'terminals', 'Open Terminals', true)}`;
    }
  }
}

function details(input: SettingsViewInput, value: State): TemplateResult | typeof nothing {
  const environment = input.readDetailEnvironment() || input.readEnvironment();
  if (!environment) return nothing;
  const profile = environment.profile;
  const detail = input.readDetail();
  const support = detail?.terminal_support;
  const network = input.readNetwork();
  const login = input.readDetailLogin();
  const observed = input.readObservedOS();
  return html`<section class="soda-env-card" aria-label="Environment details">
    <h3>Details</h3>
    <dl class="soda-env-details">
      <dt>System</dt>
      <dd>
        ${profile ? `Rocky Linux ${profile.version} · headless · ${profile.architecture}` : 'Older environment (system details not recorded)'}
      </dd>
      <dt>Terminals</dt>
      <dd>
        ${value !== 'running' ? 'Checked while the environment runs' : support === 'ready' ? 'Ready' : supportMessage(support || 'inspection_unavailable')}
      </dd>
      <dt>Your account</dt>
      <dd>
        ${login ? `Joined as ${login}` : input.readExecutionAllowed() ? 'Not joined yet · ' : 'Needs write access to join'}
        ${!login && input.readExecutionAllowed() ? link(input, 'terminals', 'Join in Terminals') : nothing}
      </dd>
      <dt>Private network</dt>
      <dd>
        ${network ? (network.enabled ? `On · ${network.state.replaceAll('-', ' ')}` : 'Off') : 'Not checked'} ·
        ${link(input, 'network', 'Network settings')}
      </dd>
    </dl>
    <details class="soda-env-advanced">
      <summary>Technical details</summary>
      <dl class="soda-env-details">
        ${
          profile
            ? html`<dt>Profile</dt>
                <dd><code>${profile.id}</code></dd>
                <dt>Original image</dt>
                <dd><code>${profile.image}</code></dd>
                <dt>Soda recipe</dt>
                <dd><code>${profile.revision}</code></dd>`
            : nothing
        }
        ${
          observed
            ? html`<dt>Installed OS</dt>
                <dd>
                  ${observed.release ? `${observed.release.name} (${observed.release.id} ${observed.release.version})` : observed.running ? 'Not readable' : 'Read it while the environment runs'}
                </dd>`
            : nothing
        }
      </dl>
      <button
        type="button"
        class="ui basic button"
        ?disabled=${input.isBlocked()}
        @click=${(event: Event) => input.requestInspectOS(event)}
      >
        Read installed OS
      </button>
      ${input.readOsStatus() ? html`<p class="soda-env-note" role="status">${input.readOsStatus()}</p>` : nothing}
    </details>
  </section>`;
}

function danger(input: SettingsViewInput, value: State): TemplateResult | typeof nothing {
  if (value !== 'running' || !input.readEnvironmentAdmin()) return nothing;
  const confirmed = input.readStopConfirmed();
  return html`<section class="soda-env-card soda-env-danger" aria-label="Stop environment">
    <h3>Stop environment</h3>
    <p class="soda-env-note">
      Stopping ends everyone’s terminals, SSH sessions and running work right away. Files are kept, and it stays stopped
      after server restarts until someone starts it.
    </p>
    <label class="soda-env-check"
      ><input
        type="checkbox"
        .checked=${confirmed}
        @change=${(event: Event) => input.setStopConfirmed((event.target as HTMLInputElement).checked)}
      />
      I understand this interrupts everyone’s work</label
    >
    <button
      type="button"
      class="ui red button"
      ?disabled=${input.isBlocked() || !confirmed}
      @click=${(event: Event) => input.requestStop(event)}
    >
      Stop environment
    </button>
  </section>`;
}

export function renderEnvironmentPlace(input: SettingsViewInput): TemplateResult {
  const phase = phaseOf(input);
  const value = phase.kind;
  const refreshing = input.readOperation()?.kind === 'refresh' && value !== 'loading';
  const settled = value === 'absent' || value === 'stopped' || value === 'running' || value === 'unknown';
  return html`<section
    class="soda-spaces-view soda-env"
    role="tabpanel"
    aria-label="Environment"
    ?hidden=${input.readSelected() !== 'environment'}
  >
    <style>
      .soda-env {
        display: grid;
        gap: 16px;
        max-width: 880px;
      }
      .soda-env-card {
        display: grid;
        gap: 12px;
        justify-items: start;
        padding: 18px 20px;
        border: 1px solid var(--soda-page-edge);
        background: var(--soda-page-surface);
      }
      .soda-env h3 {
        margin: 0;
        font: var(--soda-font-label);
        font-size: 17px;
      }
      .soda-env-head {
        display: flex;
        align-items: center;
        gap: 12px;
        flex-wrap: wrap;
        justify-self: stretch;
      }
      .soda-env-head h3 {
        font: var(--soda-font-heading);
        font-size: 22px;
      }
      .soda-env-head .soda-env-refresh {
        margin-left: auto;
      }
      .soda-env-lead {
        margin: 0;
        max-width: 64ch;
      }
      .soda-env-note {
        margin: 0;
        color: var(--soda-page-muted);
        font: var(--soda-font-meta);
      }
      .soda-env-pill {
        display: inline-flex;
        align-items: center;
        gap: 6px;
        padding: 2px 10px;
        border: 1px solid var(--soda-page-edge);
        color: var(--soda-page-muted);
        font: var(--soda-font-caption);
      }
      .soda-env-pill::before {
        content: '';
        width: 7px;
        height: 7px;
        border-radius: 50%;
        background: currentColor;
      }
      .soda-env-pill[data-tone='busy'] {
        color: var(--soda-page-text);
      }
      .soda-env-pill[data-tone='busy']::before {
        background: var(--soda-page-link);
        animation: soda-env-pulse 1.2s ease-in-out infinite;
      }
      @keyframes soda-env-pulse {
        50% {
          opacity: 0.25;
        }
      }
      .soda-env-progress {
        display: flex;
        gap: 12px;
        align-items: flex-start;
      }
      .soda-env-progress p {
        margin: 0;
      }
      .soda-env-spinner {
        flex: none;
        width: 18px;
        height: 18px;
        margin-top: 2px;
        border: 2px solid var(--soda-page-edge);
        border-top-color: var(--soda-page-link);
        border-radius: 50%;
        animation: soda-env-spin 0.8s linear infinite;
      }
      @keyframes soda-env-spin {
        to {
          transform: rotate(360deg);
        }
      }
      @media (prefers-reduced-motion: reduce) {
        .soda-env-spinner,
        .soda-env-pill::before {
          animation: none;
        }
      }
      .soda-env-notice {
        display: flex;
        align-items: flex-start;
        justify-content: space-between;
        gap: 16px;
        justify-self: stretch;
        padding: 10px 12px;
        border-left: 3px solid var(--soda-page-success);
        background: var(--soda-page-selected);
      }
      .soda-env-notice[data-tone='bad'] {
        border: 1px solid var(--soda-page-warning-edge);
        border-left-width: 3px;
        background: var(--soda-page-warning-bg);
      }
      .soda-env-notice p {
        margin: 0;
        overflow-wrap: anywhere;
      }
      button.soda-env-link {
        background: none;
        border: 0;
        padding: 0;
        cursor: pointer;
        font: var(--soda-font-meta);
      }
      .soda-env-pill[data-tone='good'] {
        color: var(--soda-page-success);
        border-color: var(--soda-page-success);
      }
      .soda-env-pill[data-tone='warn'] {
        color: var(--soda-page-warning);
        border-color: var(--soda-page-warning-edge);
        background: var(--soda-page-warning-bg);
      }
      .soda-env .ui.button {
        margin: 0;
      }
      .soda-env-form {
        display: grid;
        gap: 14px;
        justify-items: start;
        padding: 14px 16px;
        border: 1px solid var(--soda-page-edge);
        background: var(--soda-page-canvas);
        justify-self: stretch;
      }
      .soda-env-field {
        display: grid;
        gap: 6px;
        font: var(--soda-font-label);
      }
      .soda-env-field select {
        min-width: 260px;
      }
      .soda-env-check {
        display: flex;
        align-items: flex-start;
        gap: 10px;
      }
      .soda-env-check input {
        margin-top: 4px;
      }
      .soda-env-check > span {
        display: grid;
        gap: 2px;
      }
      .soda-env-details {
        display: grid;
        grid-template-columns: max-content 1fr;
        column-gap: 24px;
        row-gap: 8px;
        margin: 0;
      }
      .soda-env-details dt {
        color: var(--soda-page-muted);
        font: var(--soda-font-meta);
      }
      .soda-env-details dd {
        margin: 0;
        overflow-wrap: anywhere;
      }
      .soda-env-details code {
        font: var(--soda-font-mono);
        font-size: 12px;
      }
      .soda-env-link {
        color: var(--soda-page-link);
        text-decoration: underline;
      }
      .soda-env-advanced {
        justify-self: stretch;
      }
      .soda-env-advanced > summary {
        cursor: pointer;
        color: var(--soda-page-muted);
        font: var(--soda-font-meta);
      }
      .soda-env-advanced[open] {
        display: grid;
        gap: 12px;
        justify-items: start;
      }
      .soda-env-danger {
        border-color: var(--soda-page-warning-edge);
      }
      @media (max-width: 560px) {
        .soda-env-details {
          grid-template-columns: 1fr;
        }
        .soda-env-details dd {
          margin-bottom: 6px;
        }
      }
    </style>
    <section class="soda-env-card" aria-label="Development environment">
      <div class="soda-env-head">
        <h3>Development environment</h3>
        ${badge(value)}
        ${
          refreshing
            ? html`<span class="soda-env-note soda-env-refresh" role="status">Refreshing…</span>`
            : settled
              ? html`<button
                  type="button"
                  class="soda-env-link soda-env-refresh"
                  ?disabled=${input.isBusy() || input.isStale()}
                  @click=${(event: Event) => input.requestRefresh(event)}
                >
                  Refresh
                </button>`
              : nothing
        }
      </div>
      ${notice(input)}
      ${
        input.isStale()
          ? html`<p class="soda-env-note">
              This page is out of date.
              <button type="button" class="ui basic button" @click=${(event: Event) => input.reloadPage(event)}>
                Reload page
              </button>
            </p>`
          : nothing
      }
      ${summary(input, phase)}
    </section>
    ${value === 'loading' || value === 'creating' ? nothing : details(input, value)} ${danger(input, value)}
  </section>`;
}
