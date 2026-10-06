import {html} from 'lit';
import type {Confirmation} from './soda-tailnet-actions.js';

export interface ConfirmationViewInput {
  isBlocked: () => boolean;
  hasLifetime: () => boolean;
  readPending: () => Confirmation | null;
  onConfirmKeydown: (event: KeyboardEvent) => void;
  onConfirm: () => void;
  onCancel: () => void;
}

export function renderReconnect(input: ConfirmationViewInput) {
  if (!(input.isBlocked() || !input.hasLifetime())) return '';
  return html`<p>
    Refresh this Forgejo page to restore operator authorization. Native CLI/console recovery remains available.
  </p>`;
}

export function renderPending(input: ConfirmationViewInput) {
  const pending = input.readPending();
  if (!pending) return '';
  return html`<section
    class="tailnet-notice"
    role="region"
    aria-labelledby="tailnet-confirm-title"
    @keydown=${input.onConfirmKeydown}
  >
    <h2 id="tailnet-confirm-title">Confirm ${pending.label}</h2>
    <p>${pending.warning}</p>
    <div class="settings-actions">
      <button type="button" data-confirm @click=${input.onConfirm}>Confirm ${pending.label}</button
      ><button type="button" @click=${input.onCancel}>Cancel</button>
    </div>
  </section>`;
}
