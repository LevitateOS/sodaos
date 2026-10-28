// SodaOS settings place: Settings → SodaOS → {section} (M19, lane C). The
// entry shows it only to repository admins, environment administrators and
// the operator; the owning APIs still decide every save.
import {spacesResponse} from './sodaspaces-inventory-response.js';
import {mountProjectControls} from './sodaspaces-project.js';
import {settingsSections} from './soda-places.js';
import type {Place, SettingsSection} from './soda-places.js';
import type {PlaceMount} from './soda-place-view.js';

const section = (place: Place): SettingsSection => {
  const value = place.place === 'settings' ? place.section : '';
  return (settingsSections as readonly string[]).includes(value) ? (value as SettingsSection) : 'environment';
};

export const mountSettingsPlace: PlaceMount = (host, context, initial) => {
  let current = section(initial);
  let controls: ReturnType<typeof mountProjectControls> | undefined;
  let disposed = false;
  const status = document.createElement('p');
  status.setAttribute('role', 'status');
  status.textContent = 'Loading SodaOS settings…';
  host.append(status);
  // The project controls are bound to the viewer's account id, which the
  // spaces collection carries.
  void (async () => {
    try {
      const response = await context.transport.request('spaces', {method: 'GET'});
      if (!response.ok) throw Error('SodaOS settings could not load.');
      const {actor} = spacesResponse(await response.json());
      if (disposed) return;
      status.remove();
      controls = mountProjectControls(host, {
        expectedUserId: actor.id,
        actorLogin: actor.login,
        repositoryId: context.repository,
        page: true,
        settings: true,
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
      controls?.showSection(current);
    },
    setActive(active: boolean) {
      if (active) controls?.resume();
      else controls?.suspend();
    },
    dispose() {
      disposed = true;
      controls?.dispose();
      host.replaceChildren();
    },
  };
};
