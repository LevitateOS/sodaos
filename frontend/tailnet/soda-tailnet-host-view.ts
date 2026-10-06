import {html} from 'lit';
import type {Host, Settings} from './soda-tailnet-response.js';

export type ExitChoice = {peer: Host['peers'][number]; address: string};

function advertisedExitPeer(peer: Host['peers'][number]) {
  return peer.exit_node && peer.addresses.length > 0;
}

function exitPeerChoice(exitNode: string, peer: Host['peers'][number]): ExitChoice {
  return {peer, address: peer.addresses.includes(exitNode) ? exitNode : peer.addresses[0] || ''};
}

function exitPeerOptions(host: Host, exitNode: string) {
  const options: ExitChoice[] = [];
  for (const peer of host.peers) {
    if (advertisedExitPeer(peer)) options.push(exitPeerChoice(exitNode, peer));
  }
  return options;
}

function exitNodeMissing(host: Host) {
  const selected = host.preferences.exit_node_id;
  if (selected === '') return false;
  return !host.peers.some((peer) => peer.id === selected);
}

function renderPeerItem(peer: Host['peers'][number]) {
  return html`<li>
    ${peer.dns_name || peer.id}:
    ${peer.online ? 'online' : 'offline'}${peer.expired ? ', expired' : ''}${peer.exit_node ? ', exit node' : ''} ·
    ${peer.addresses.join(', ')}
  </li>`;
}

export interface HostViewInput {
  isStale: () => boolean;
  readAuthURL: () => string;
  readExitNode: () => string;
  readHost: () => Host | null | undefined;
  controlsDisabled: () => boolean;
  readAllowLAN: () => boolean;
  readAdvertise: () => boolean;
  isHostDirty: () => boolean;
  readApplianceLabel: () => string;
  onExitNodeChange: (event: Event) => void;
  onAllowLANChange: (event: Event) => void;
  onApplyExitNode: (event: Event) => void;
  onAdvertiseChange: (event: Event) => void;
  onApplyAdvertise: (event: Event) => void;
  onDiscardHostDraft: () => void;
  onSignin: (event: Event) => void;
  onAuthentication: (event: Event) => void;
  onLogout: (event: Event) => void;
  onRefreshForgejo: (event: Event) => void;
}

function renderHostStatus(input: HostViewInput, host: Host | null | undefined) {
  if (input.isStale()) return html`<p>Appliance observations are stale. No usable endpoint is asserted.</p>`;
  if (!host)
    return html`<p>
      Host observation unavailable, not disconnected. Verify the native daemon, reviewed version and helper
      configuration through an approved private path.
    </p>`;
  return html`
    <dl>
      <dt>Native state</dt>
      <dd>${host.state}${host.expired ? ' · expired' : ''}</dd>
      <dt>Network</dt>
      <dd>${host.tailnet || 'Not observed'}</dd>
      <dt>Device</dt>
      <dd>${host.dns_name || 'Not observed'}</dd>
      <dt>Observed addresses (not verified reachable)</dt>
      <dd>${host.addresses.join(', ') || 'None observed'}</dd>
      <dt>MagicDNS</dt>
      <dd>${host.magic_dns_enabled ? 'Enabled' : 'Disabled / not observed'}</dd>
      <dt>Native health issues</dt>
      <dd>${host.health_issues} — raw diagnostics are intentionally hidden.</dd>
    </dl>
    ${host.state === 'NeedsMachineAuth' ? html`<p>Device approval is required in Tailscale. Tailnet Lock signing is not automated; do not weaken Lock or copy signing keys.</p>` : ''}
  `;
}

function renderAuthLink(input: HostViewInput) {
  if (!input.readAuthURL()) return '';
  return html`<p>
    <a
      data-authentication
      href=${input.readAuthURL()}
      target="_blank"
      rel="noopener noreferrer"
      referrerpolicy="no-referrer"
      >Continue appliance sign-in at Tailscale</a
    >. This is provider authentication, not Soda sign-in. Do not copy this link into logs or evidence.
  </p>`;
}

function renderUnavailableExitOption(input: HostViewInput, options: ExitChoice[]) {
  if (!(input.readExitNode() && !options.some((choice) => choice.address === input.readExitNode()))) return '';
  return html`<option value=${input.readExitNode()} selected>Previous selection (unavailable)</option>`;
}

function renderExitOption(input: HostViewInput, choice: ExitChoice) {
  const {peer, address} = choice;
  return html`<option
    value=${address}
    ?selected=${address === input.readExitNode()}
    ?disabled=${!peer.online || peer.expired}
  >
    ${peer.dns_name || peer.id}${!peer.online || peer.expired ? ' (unavailable)' : ''}
  </option>`;
}

function renderExitPreferences(input: HostViewInput) {
  const host = input.readHost();
  if (!host) return '';
  const options = exitPeerOptions(host, input.readExitNode());
  return html`
    <fieldset ?disabled=${input.controlsDisabled()}>
      <legend>Exit-node preferences</legend>
      ${exitNodeMissing(host) ? html`<p role="alert">The selected native exit-node ID is missing from the peer observation. It has not been cleared. Select an available replacement or explicitly choose None.</p>` : ''}
      <label
        >Exit node<select aria-label="Exit node" .value=${input.readExitNode()} @change=${input.onExitNodeChange}>
          <option value="" ?selected=${input.readExitNode() === ''}>None (clear only on Apply)</option>
          ${renderUnavailableExitOption(input, options)} ${options.map((choice) => renderExitOption(input, choice))}
        </select></label
      >
      <label
        ><input
          type="checkbox"
          .checked=${input.readAllowLAN()}
          ?disabled=${!input.readExitNode()}
          @change=${input.onAllowLANChange}
        />Allow local LAN while using exit node</label
      >
      <button type="button" @click=${input.onApplyExitNode}>Apply exit node</button>
      <label
        ><input type="checkbox" .checked=${input.readAdvertise()} @change=${input.onAdvertiseChange} />Advertise
        appliance as an exit node</label
      >
      <button type="button" @click=${input.onApplyAdvertise}>Apply advertisement</button>
      <p>Provider approval is separate from advertisement and online status. Existing subnet routes are not edited.</p>
      <button type="button" @click=${input.onDiscardHostDraft}>Discard host draft / use latest observation</button>
      ${input.isHostDirty() ? html`<p>Unsaved host draft; its original revision is retained across refresh.</p>` : ''}
    </fieldset>
  `;
}

function renderPeerList(input: HostViewInput, host: Host) {
  if (input.isStale()) return '';
  return html`<details>
    <summary>Observed peers (${host.peers.length})</summary>
    <ul>
      ${host.peers.map(renderPeerItem)}
    </ul>
  </details>`;
}

function renderHostControls(input: HostViewInput, host: Host) {
  return html`
    <fieldset ?disabled=${input.controlsDisabled()}>
      <legend>Appliance connection</legend>
      <div class="settings-actions">
        <button type="button" @click=${input.onSignin}>
          ${host.have_node_key ? 'Resume / reauthenticate' : 'Sign in appliance'}
        </button>
        <button type="button" @click=${input.onAuthentication}>Recover authentication link</button>
        <button type="button" @click=${input.onLogout}>Disconnect appliance</button>
        <button type="button" @click=${input.onRefreshForgejo}>Refresh Forgejo advertisement</button>
      </div>
    </fieldset>
    ${renderAuthLink(input)} ${renderExitPreferences(input)} ${renderPeerList(input, host)}
  `;
}

export function renderHostSection(input: HostViewInput, settings: Settings) {
  const host = settings.host;
  return html`<section aria-labelledby="tailnet-appliance">
    <h2 id="tailnet-appliance">${input.readApplianceLabel()}</h2>
    <p>Persistent native connection. Host identity and ordinary SSH authentication are not shared with projects.</p>
    ${renderHostStatus(input, host)} ${host ? renderHostControls(input, host) : ''}
  </section>`;
}
