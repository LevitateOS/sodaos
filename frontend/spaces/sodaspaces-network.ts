import {html} from 'lit';
import type {TemplateResult} from 'lit';
import type {ProjectNetwork, ProjectOptions} from '../tailnet/soda-tailnet-response.js';

export function renderNetworkSelection(
  options: ProjectOptions | undefined,
  enabled: boolean,
  blocked: boolean,
  change: (enabled: boolean) => void,
  label = 'Use appliance-managed Tailnet'
) {
  return html`<fieldset ?disabled=${blocked}>
    <legend>Project network</legend>
    <label
      ><input
        type="checkbox"
        .checked=${enabled}
        ?disabled=${!options?.available}
        @change=${(event: Event) => {
          if (event.target instanceof HTMLInputElement) change(event.target.checked);
        }}
      />
      ${label}${options?.tailnet ? ' · ' + options.tailnet : ''}</label
    >
    <p>
      ${options?.available ? 'Creates a separate ephemeral device when this project runs. This reviewed selection is saved with Create.' : 'Managed enrollment is unavailable. Creation remains available with networking Off.'}
      Off does not change ordinary LAN access. Network failure never deletes the project.
    </p>
  </fieldset>`;
}

function networkStatus(view: ProjectNetwork | undefined, notice: string) {
  if (!view) return notice || 'Private network state has not been observed.';
  return 'Saved policy: ' + (view.enabled ? 'managed' : 'Off') + ' · Native observation: ' + view.state;
}

function connectedDevice(view: ProjectNetwork) {
  return html`<p>Device: ${view.dns_name || 'Name unavailable'} · ${view.addresses.join(', ')}</p>`;
}

function onNetworkConfirm(event: Event, confirm: (value: boolean) => void) {
  if (event.target instanceof HTMLInputElement) confirm(event.target.checked);
}

function enableBlocked(view: ProjectNetwork, confirmed: boolean) {
  return (
    !confirmed || view.enabled || !view.available_binding || (!!view.binding && view.binding !== view.available_binding)
  );
}

function retryBlocked(view: ProjectNetwork, confirmed: boolean) {
  return !confirmed || !view.enabled || view.binding !== view.available_binding;
}

function networkActions(
  view: ProjectNetwork,
  confirmed: boolean,
  confirm: (value: boolean) => void,
  act: (event: MouseEvent, action: 'enable' | 'disable' | 'retry') => void
): TemplateResult {
  return html`<label
      ><input type="checkbox" .checked=${confirmed} @change=${(event: Event) => onNetworkConfirm(event, confirm)} /> I
      confirm changing network access for ${view.project}. Off may interrupt private connections; it does not delete
      project data.</label
    >
    <div class="settings-actions">
      <button
        type="button"
        class="ui primary button"
        ?disabled=${enableBlocked(view, confirmed)}
        @click=${(event: MouseEvent) => act(event, 'enable')}
      >
        Use managed network
      </button>
      <button
        type="button"
        class="ui button"
        ?disabled=${!confirmed || !view.enabled}
        @click=${(event: MouseEvent) => act(event, 'disable')}
      >
        Turn Off
      </button>
      <button
        type="button"
        class="ui button"
        ?disabled=${retryBlocked(view, confirmed)}
        @click=${(event: MouseEvent) => act(event, 'retry')}
      >
        Retry native startup
      </button>
    </div>`;
}

export function renderNetwork(
  view: ProjectNetwork | undefined,
  notice: string,
  administrator: boolean,
  blocked: boolean,
  confirmed: boolean,
  confirm: (value: boolean) => void,
  act: (event: MouseEvent, action: 'enable' | 'disable' | 'retry') => void
) {
  return html`<fieldset ?disabled=${blocked}>
    <legend>Tailnet access</legend>
    <p role="status">${networkStatus(view, notice)}</p>
    ${view?.available_network ? html`<p>Appliance-managed network: ${view.available_network}</p>` : ''}
    ${view?.state === 'connected' ? connectedDevice(view) : ''}
    <p>
      Each project has its own ephemeral identity, separate from the appliance. Ordinary SSH and application
      authentication still apply. A connected node is not proof of client reachability.
    </p>
    ${view && administrator ? networkActions(view, confirmed, confirm, act) : ''}
    <p>
      Refresh only observes. Startup retries reuse this run's existing identity, never repeat an uncertain key request.
      A lost or logged-out identity requires a new project run; Start/Stop remain explicit Environment actions.
    </p>
  </fieldset>`;
}
