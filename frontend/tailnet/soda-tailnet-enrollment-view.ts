import {html} from 'lit';
import type {Enrollment} from './soda-tailnet-response.js';

export interface EnrollmentViewInput {
  readNetwork: () => string;
  readTags: () => string;
  readPreauthorized: () => boolean;
  readMode: () => 'save' | 'rotate';
  readEnrollmentLabel: () => string;
  controlsDisabled: () => boolean;
  onSubmitEnrollment: (event: SubmitEvent) => void;
  onMarkEnrollmentDirty: () => void;
  onNetworkInput: (event: Event) => void;
  onTagsInput: (event: Event) => void;
  onPreauthorizedChange: (event: Event) => void;
  onDiscardEnrollmentDraft: () => void;
  onModeChange: (event: Event) => void;
  onCloseAdmission: (event: Event) => void;
  onOfferManagedDefault: (event: Event) => void;
  onKeepDefaultOff: (event: Event) => void;
}

function renderEnrollmentSummary(policy: Enrollment) {
  return html`
    <p>
      ${policy.configured ? 'Configured' : 'Not configured'} · Credential check
      ${policy.credential_checked ? 'previously passed' : 'not recorded'} · Enrollment not verified.
    </p>
    <p>
      Managed network: ${policy.tailnet || 'None'}. Future admission:
      ${policy.admission ? 'configured open' : 'closed'}. New-project default:
      ${policy.default ? 'managed, explicitly reviewed in Create' : 'Off'}.
    </p>
    <p>
      ${policy.runtime_supported ? 'Native project supervision is configured. Each managed project gets a separate ephemeral identity when it runs; token acceptance alone is not enrollment or connectivity proof.' : 'Project runtime is not configured. Automatic defaults remain unavailable.'}
      Existing projects remain Off until explicitly selected. Host login never enrolls projects.
    </p>
    <p>
      Tailscale owns network access policy. A project tag alone does not isolate host/peers. Approval-required networks
      need explicitly permitted preauthorization; automatic Tailnet Lock signing is unsupported.
    </p>
    <p>
      <a href="https://tailscale.com/kb/1215/oauth-clients" target="_blank" rel="noopener noreferrer"
        >Create a restricted Tailscale OAuth client</a
      >
      with auth_keys and only the selected tags. Token acceptance does not prove target, scope, expiry or enrollment; no
      hidden test device is created.
    </p>
  `;
}

function renderEnrollmentForm(input: EnrollmentViewInput) {
  return html`
    <form @submit=${input.onSubmitEnrollment} @input=${input.onMarkEnrollmentDirty}>
      <label
        >Managed Tailnet<input
          required
          maxlength="253"
          .value=${input.readNetwork()}
          ?readonly=${input.readMode() === 'rotate'}
          @input=${input.onNetworkInput}
      /></label>
      <label
        >Project tags (comma separated)<input
          required
          .value=${input.readTags()}
          ?readonly=${input.readMode() === 'rotate'}
          @input=${input.onTagsInput}
      /></label>
      <label
        ><input
          type="checkbox"
          .checked=${input.readPreauthorized()}
          ?disabled=${input.readMode() === 'rotate'}
          @change=${input.onPreauthorizedChange}
        />Explicitly preauthorize devices if provider policy permits</label
      >
      <label>OAuth client ID<input name="client_id" required maxlength="128" autocomplete="off" /></label>
      <label
        >OAuth client secret<input
          name="client_secret"
          type="password"
          required
          maxlength="525"
          autocomplete="new-password"
          spellcheck="false"
      /></label>
      <p>
        Secret is cleared when submitted, on failure and on departure. It is never displayed or stored in browser
        storage. Re-enter it for a subsequent Save after Check.
      </p>
      <label
        ><input type="checkbox" name="reviewed" />I reviewed private-network exposure and this binding change.
        Replacement requires explicit project reselection; rotation preserves tags and existing device
        identities.</label
      >
      <div class="settings-actions">
        <button type="submit" value="check">Check credential only</button
        ><button type="submit" value=${input.readMode()}>
          ${input.readMode() === 'rotate' ? 'Rotate credential' : 'Save binding'}
        </button>
        <button type="button" @click=${input.onDiscardEnrollmentDraft}>Discard enrollment draft</button>
      </div>
    </form>
  `;
}

function renderEnrollmentAdmission(input: EnrollmentViewInput, policy: Enrollment) {
  if (!policy.configured) return '';
  return html`<div class="settings-actions">
    <button type="button" @click=${input.onCloseAdmission}>Close future admission</button>
    ${policy.runtime_supported && policy.admission ? html`<button type="button" @click=${input.onOfferManagedDefault}>Offer managed access by default</button>` : ''}
    <button type="button" @click=${input.onKeepDefaultOff}>Keep new-project default Off</button>
  </div>`;
}

export function renderEnrollmentSection(input: EnrollmentViewInput, policy: Enrollment | undefined) {
  if (!policy) return '';
  return html`<section aria-labelledby="tailnet-enrollment">
    <h2 id="tailnet-enrollment">${input.readEnrollmentLabel()}</h2>
    ${renderEnrollmentSummary(policy)}
    <fieldset ?disabled=${input.controlsDisabled()}>
      <legend>Enrollment configuration</legend>
      <label
        >Configuration action<select
          aria-label="Configuration action"
          .value=${input.readMode()}
          @change=${input.onModeChange}
        >
          <option value="save">Save a new / replacement binding</option>
          <option value="rotate" ?disabled=${!policy.configured}>Rotate credential in this binding</option>
        </select></label
      >
      ${renderEnrollmentForm(input)} ${renderEnrollmentAdmission(input, policy)}
    </fieldset>
  </section>`;
}
