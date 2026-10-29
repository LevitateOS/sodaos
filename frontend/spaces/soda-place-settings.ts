// SodaOS settings place: Settings → SodaOS → {section} (M19, lane C). The
// entry shows it only to repository admins, environment administrators and
// the operator; the owning APIs still decide every save.
import {html, render} from 'lit';
import {spacesResponse} from './sodaspaces-inventory-response.js';
import {mountProjectControls} from './sodaspaces-project.js';
import {settingsSections} from './soda-places.js';
import type {Place, SettingsSection} from './soda-places.js';
import type {PlaceContext, PlaceMount} from './soda-place-view.js';

const section = (place: Place): SettingsSection => {
  const value = place.place === 'settings' ? place.section : '';
  return (settingsSections as readonly string[]).includes(value) ? (value as SettingsSection) : 'environment';
};

const sectionNames: Record<SettingsSection, string> = {
  environment: 'Development environment',
  access: 'Members & access',
  network: 'Network',
  factory: 'Factory',
};

// Forgejo's own repository settings pages, so the side column moves in and out.
const nativeSettings: [string, string][] = [
  ['Options', '/settings'],
  ['Collaborators', '/settings/collaboration'],
  ['Branches', '/settings/branches'],
  ['Tags', '/settings/tags'],
  ['Webhooks', '/settings/hooks'],
  ['Deploy keys', '/settings/keys'],
];

function renderSide(side: HTMLElement, heading: HTMLElement, context: PlaceContext, current: SettingsSection) {
  const repository = `${context.prefix}/${context.slug.split('/').map(encodeURIComponent).join('/')}`;
  render(
    html`<a class="soda-settings-place-back" href=${repository + '/settings'}>← Repository settings</a>
      <ul>
        ${nativeSettings.map(([label, path]) => html`<li><a href=${repository + path}>${label}</a></li>`)}
      </ul>
      <h3 id="soda-settings-place-group">SodaOS</h3>
      <ul aria-labelledby="soda-settings-place-group">
        ${settingsSections.map(
          (name) =>
            html`<li>
              <a
                class=${name === current ? 'active' : ''}
                href=${context.places.settings(name)}
                aria-current=${name === current ? 'page' : 'false'}
                >${sectionNames[name]}</a
              >
            </li>`
        )}
      </ul>`,
    side
  );
  render(html`SodaOS settings <span>· ${sectionNames[current]}</span>`, heading);
}

export const mountSettingsPlace: PlaceMount = (host, context, initial) => {
  let current = section(initial);
  let controls: ReturnType<typeof mountProjectControls> | undefined;
  let disposed = false;
  const page = document.createElement('div');
  page.className = 'soda-settings-place';
  const side = document.createElement('nav');
  side.className = 'soda-settings-place-side';
  side.setAttribute('aria-label', 'Repository settings');
  const main = document.createElement('div');
  main.className = 'soda-settings-place-main';
  const heading = document.createElement('h2');
  const status = document.createElement('p');
  status.setAttribute('role', 'status');
  status.textContent = 'Loading SodaOS settings…';
  main.append(heading, status);
  page.append(side, main);
  host.append(page);
  renderSide(side, heading, context, current);
  // The project controls are bound to the viewer's account id, which the
  // spaces collection carries.
  void (async () => {
    try {
      const response = await context.transport.request('spaces', {method: 'GET'});
      if (!response.ok) throw Error('SodaOS settings could not load.');
      const {actor} = spacesResponse(await response.json());
      if (disposed) return;
      status.remove();
      controls = mountProjectControls(main, {
        expectedUserId: actor.id,
        actorLogin: actor.login,
        repositoryId: context.repository,
        page: true,
        settings: true,
        place: true,
        transport: context.transport,
        forgejoPrefix: context.prefix,
      });
      controls.setPresentation('settings');
      controls.showSection(current);
      void controls.refresh();
    } catch (error) {
      if (!disposed) status.textContent = error instanceof Error ? error.message : 'SodaOS settings could not load.';
    }
  })();
  return {
    update(place: Place) {
      current = section(place);
      renderSide(side, heading, context, current);
      controls?.showSection(current);
    },
    setActive(active: boolean) {
      if (active) controls?.resume();
      else controls?.suspend();
    },
    dispose() {
      disposed = true;
      controls?.dispose();
      render(html``, side);
      render(html``, heading);
      host.replaceChildren();
    },
  };
};
