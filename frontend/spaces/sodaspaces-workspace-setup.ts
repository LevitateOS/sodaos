import {html} from 'lit';

export function setupIntroKind(busy: boolean) {
  return busy ? 'loading' : 'unavailable';
}
export function setupIntroHeading(busy: boolean, reconnectRequired: boolean) {
  if (busy) return 'Loading projects…';
  if (reconnectRequired) return 'Reconnect to Forgejo';
  return 'Could not load projects';
}
export function setupIntroDescription(busy: boolean, reconnectRequired: boolean) {
  if (busy) return html`Checking the projects you can access.`;
  if (reconnectRequired) return html`Sign in again to restore your Forgejo access.`;
  return html`We couldn’t load your project list. Try again.`;
}
export function setupIntroAction(
  busy: boolean,
  reconnectRequired: boolean,
  stale: boolean,
  connectURL: string,
  blocked: boolean,
  onRefresh: () => void
) {
  if (busy) return html``;
  if (reconnectRequired) return html`<a class="ui primary button" href=${connectURL}>Reconnect to Forgejo</a>`;
  if (stale)
    return html`<button class="ui primary button" @click=${() => window.location.reload()}>Reload Spaces</button>`;
  return html`<button class="ui primary button" ?disabled=${blocked} @click=${onRefresh}>Retry projects</button>`;
}
export function setupIntroHelper(busy: boolean) {
  if (busy) return html`This will not create or start anything.`;
  return html`Your existing projects and terminals are not replaced.`;
}
export function searchAdmitted(
  stale: boolean,
  available: boolean,
  setup: 'repositories' | 'configure' | null,
  activeSurface: boolean
) {
  return !stale && available && setup === 'repositories' && activeSurface;
}
export function repositorySearchPath(query: string, cursor: string) {
  const params = new URLSearchParams({q: query});
  if (cursor) params.set('cursor', cursor);
  return '/api/repositories?' + params;
}
