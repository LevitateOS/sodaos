import {object, readSodaJSON, SodaRequestError} from './sodaspaces-api.js';
import {mutate} from './sodaspaces-project-mutations.js';
import type {MutationsInput} from './sodaspaces-project-mutations.js';
import type {CreationProfile} from './sodaspaces-project-response.js';
import type {ProjectOptions} from '../tailnet/soda-tailnet-response.js';
import type {ProjectContext} from './sodaspaces-project.js';

export interface ObservedSummary {
  repositoryId: string;
  environmentId: string;
  provisioned: boolean;
  login: string;
  running: boolean;
}

export interface RequestInput {
  readBinding: () => ProjectContext | undefined;
  beginEpoch: () => number;
  abortRead: () => void;
  takeReadController: () => AbortController;
  resetState: () => void;
  invalidate: () => void;
  dispatchObserved: (detail: ObservedSummary) => void;
  dispatchChanged: (repositoryId: string) => void;
  readCanCreate: () => boolean;
  readNetworkReview: () => boolean;
  readProfiles: () => CreationProfile[];
  readSelectedProfile: () => string;
  readNetworkEnabled: () => boolean;
  readNetworkOptions: () => ProjectOptions | undefined;
  mutations: MutationsInput;
}

function sodaFetchInit(
  method: string,
  headers: Record<string, string>,
  body?: Record<string, unknown>,
  signal?: AbortSignal
): RequestInit {
  return {
    method,
    headers,
    ...(body === undefined
      ? {}
      : {
          body: JSON.stringify(body),
        }),
    ...(signal
      ? {
          signal,
        }
      : {}),
  };
}

async function sodaErrorCode(response: Response): Promise<string | undefined> {
  try {
    const error = object(object(await readSodaJSON(response)).error);
    if (typeof error.code === 'string') return error.code;
  } catch {
    /* Never display a response body. */
  }
}

function requestHeaders(method: string): Record<string, string> {
  const headers: Record<string, string> = {};
  if (method === 'GET') return headers;
  headers['Content-Type'] = 'application/json';
  return headers;
}

function admitHttpFailure(input: RequestInput, status: number) {
  if (status !== 401 && status !== 403) return;
  input.invalidate();
}

export async function api(
  input: RequestInput,
  path: string,
  method = 'GET',
  body?: Record<string, unknown>,
  signal?: AbortSignal
): Promise<unknown> {
  const binding = input.readBinding();
  if (!binding) throw Error('Missing native extension');
  const response = await binding.transport.request(
    path.slice('/api/'.length),
    sodaFetchInit(method, requestHeaders(method), body, signal)
  );
  if (response.ok) return response.status === 204 ? null : readSodaJSON(response);
  admitHttpFailure(input, response.status);
  throw new SodaRequestError(response.status, await sodaErrorCode(response));
}

export function beginEpoch(input: RequestInput) {
  return input.beginEpoch();
}

export function beginRead(input: RequestInput) {
  input.abortRead();
  input.resetState();
  return input.takeReadController();
}

export function announceObserved(input: RequestInput, summary: ObservedSummary) {
  input.dispatchObserved({
    repositoryId: summary.repositoryId,
    environmentId: summary.environmentId,
    provisioned: summary.provisioned,
    login: summary.login,
    running: summary.running,
  });
}

export function announceChanged(input: RequestInput, repositoryId: string) {
  input.dispatchChanged(repositoryId);
}

function createTailnetBody(input: RequestInput) {
  const options = input.readNetworkOptions();
  if (input.readNetworkEnabled() && options?.available)
    return {enabled: true, revision: options.revision, binding: options.binding};
  return {enabled: false};
}

export function createProject(input: RequestInput) {
  if (
    !input.readCanCreate() ||
    input.readNetworkReview() ||
    !input.readProfiles().some((p) => p.id === input.readSelectedProfile())
  )
    return;
  return mutate(
    input.mutations,
    '/api/environments',
    {
      repository_id: input.readBinding()?.repositoryId,
      profile_id: input.readSelectedProfile(),
      tailnet: createTailnetBody(input),
    },
    'Project created. Join explicitly to set up your browser-terminal account.'
  );
}
