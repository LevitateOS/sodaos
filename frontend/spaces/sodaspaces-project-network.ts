import type {ProjectNetwork} from '../tailnet/soda-tailnet-response.js';
import type {Detail, Environment} from './sodaspaces-project-response.js';
import {mutate} from './sodaspaces-project-mutations.js';
import type {MutationsInput} from './sodaspaces-project-mutations.js';

export interface NetworkInput {
  readNetwork: () => ProjectNetwork | undefined;
  readEnvironment: () => Environment | undefined;
  readDetail: () => Detail | undefined;
  isBlocked: () => boolean;
  readNetworkConfirmed: () => boolean;
  setNetworkConfirmed: (confirmed: boolean) => void;
  setNetworkEnabled: (enabled: boolean) => void;
  setNetworkReview: (review: boolean) => void;
  mutations: MutationsInput;
}

export function setJourneyNetworkEnabled(input: NetworkInput, enabled: boolean) {
  input.setNetworkEnabled(enabled);
  input.setNetworkReview(false);
}

export function setCreateNetworkEnabled(input: NetworkInput, enabled: boolean) {
  input.setNetworkEnabled(enabled);
}

export function clearNetworkReview(input: NetworkInput) {
  input.setNetworkEnabled(false);
  input.setNetworkReview(false);
}

export function setNetworkConfirmed(input: NetworkInput, confirmed: boolean) {
  input.setNetworkConfirmed(confirmed);
}

function networkChangeBlocked(input: NetworkInput) {
  return (
    !input.readNetwork() ||
    !input.readEnvironment() ||
    !input.readDetail()?.environment_administrator ||
    !input.readNetworkConfirmed() ||
    input.isBlocked()
  );
}

function networkEnableBlocked(network: ProjectNetwork) {
  return !network.available_binding || (!!network.binding && network.binding !== network.available_binding);
}

export function changeNetwork(input: NetworkInput, action: 'enable' | 'disable' | 'retry') {
  const network = input.readNetwork();
  if (networkChangeBlocked(input)) return;
  if (action !== 'disable' && networkEnableBlocked(network!)) return;
  input.setNetworkConfirmed(false);
  return mutate(
    input.mutations,
    '/api/environments/' + input.readEnvironment()!.id + '/tailnet',
    {
      action,
      revision: network!.revision,
      confirm_id: network!.project,
      ...(action === 'disable' ? {} : {binding: network!.available_binding}),
    },
    'Network policy saved; observe the native outcome.'
  );
}
