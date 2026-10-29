// Document-level contract for the single AI accounts dialog. The providers
// panel (soda-providers-entry.ts) hosts the dialog on every Forge page; the
// navbar robot (assets/branding/forgejo/providers-button.ts) and Soda views
// open it with this event instead of hosting their own copies.
export const openProvidersEvent = 'soda-open-providers';
/** Dispatched on document after the person's provider connections change. */
export const providersChangedEvent = 'soda-provider-connections-changed';
/** Set on <html> while a providers panel can answer openProvidersEvent. */
export const providersReadyAttribute = 'data-soda-providers';

export function openProviders(): void {
  document.dispatchEvent(new CustomEvent(openProvidersEvent));
}
