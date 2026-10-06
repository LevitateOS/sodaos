import {SodaRequestError} from './sodaspaces-api.js';
import {osObservation} from './sodaspaces-project-response.js';
import type {Environment, OSObservation} from './sodaspaces-project-response.js';
import {mutate} from './sodaspaces-project-mutations.js';
import type {MutationsInput} from './sodaspaces-project-mutations.js';

export interface RuntimeInput {
  readEnvironment: () => Environment | undefined;
  isBlocked: () => boolean;
  readEpoch: () => number;
  isActive: (generation: number) => boolean;
  setBusy: (busy: boolean) => void;
  setOutcome: (outcome: string) => void;
  setObservedOS: (observation: OSObservation | undefined) => void;
  setOsStatus: (status: string) => void;
  readStopConfirmed: () => boolean;
  setStopConfirmed: (confirmed: boolean) => void;
  abortRead: () => void;
  takeReadController: () => AbortController;
  invalidate: () => void;
  api: (
    path: string,
    method: string,
    body: Record<string, unknown> | undefined,
    signal: AbortSignal
  ) => Promise<unknown>;
  mutations: MutationsInput;
}

export function setStopConfirmed(input: RuntimeInput, checked: boolean) {
  input.setStopConfirmed(checked);
}

function applyOSError(input: RuntimeInput, epoch: number, error: unknown) {
  if (!input.isActive(epoch)) return;
  if (authLost(error)) input.invalidate();
  else input.setOsStatus('OS observation unavailable. Nothing was started or repaired.');
}

function authLost(error: unknown) {
  return error instanceof SodaRequestError && (error.status === 401 || error.status === 403);
}

export async function inspectOS(input: RuntimeInput) {
  const environment = input.readEnvironment();
  if (input.isBlocked() || !environment) return;
  const epoch = input.readEpoch(),
    target = environment.id;
  input.setBusy(true);
  input.setObservedOS(undefined);
  input.setOsStatus('Reading current userspace…');
  input.abortRead();
  const controller = input.takeReadController(),
    timeout = window.setTimeout(() => controller.abort(), 15000);
  try {
    const observation = osObservation(
      await input.api('/api/environments/' + target + '/os', 'GET', undefined, controller.signal),
      target
    );
    if (input.isActive(epoch)) {
      input.setObservedOS(observation);
      input.setOsStatus('Read completed. Creation metadata was not changed.');
    }
  } catch (error) {
    applyOSError(input, epoch, error);
  } finally {
    window.clearTimeout(timeout);
    if (input.isActive(epoch)) input.setBusy(false);
  }
}

export function changeLifecycle(input: RuntimeInput, stop: boolean) {
  const environment = input.readEnvironment();
  if (!environment) return;
  if (stop && !input.readStopConfirmed()) {
    input.setOutcome('Confirm the shared impact before Stop.');
    return;
  }
  return mutate(
    input.mutations,
    `/api/environments/${environment.id}/lifecycle`,
    {
      action: stop ? 'stop' : 'start',
      ...(stop
        ? {
            confirm_stop: true,
          }
        : {}),
    },
    stop
      ? 'Stop confirmed; next-boot start disabled. Existing data was not recreated or deleted.'
      : 'Start confirmed and next-boot start enabled. Refresh connection status while services initialize.'
  );
}
