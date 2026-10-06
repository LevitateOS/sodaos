import {check, fingerprint, object, SodaRequestError} from './sodaspaces-api.js';
import {creationProfile, detailResponse, environmentResponse} from './sodaspaces-project-response.js';
import type {CreationProfile, Detail, Environment} from './sodaspaces-project-response.js';
import {savedKeysResponse} from './sodaspaces-keys-response.js';
import type {SavedKey} from './sodaspaces-keys-response.js';
import {projectOptions, projectView} from '../tailnet/soda-tailnet-response.js';
import type {ProjectNetwork, ProjectOptions} from '../tailnet/soda-tailnet-response.js';
import type {Lifecycle} from './sodaspaces-project-settings-view.js';

export type RefreshPrior = {
  profile: string;
  network: ProjectOptions | undefined;
  enabled: boolean;
};

function admitRepositoryPart(part: unknown) {
  check(
    typeof part === 'string' &&
      part !== '' &&
      part !== '.' &&
      part !== '..' &&
      part.length <= 255 &&
      !/[\/\\\x00\r\n]/.test(part)
  );
}

function admitProjectLogin(login: string) {
  check(/^[a-z][a-z0-9_-]{0,30}$/.test(login) && login !== 'root');
}

export interface RefreshInput {
  isBusy: () => boolean;
  isStale: () => boolean;
  isDisposed: () => boolean;
  hasBinding: () => boolean;
  readBinding: () => {expectedUserId: string; repositoryId: string; forgejoPrefix?: string} | undefined;
  isActive: (generation: number) => boolean;
  beginEpoch: () => number;
  beginRead: () => AbortController;
  api: (
    path: string,
    method: string,
    body: Record<string, unknown> | undefined,
    signal: AbortSignal
  ) => Promise<unknown>;
  readPresentation: () => 'standard' | 'journey' | 'settings';
  readEnvironment: () => Environment | undefined;
  readDetail: () => Detail | undefined;
  readStatus: () => string;
  readSelectedProfile: () => string;
  readNetworkOptions: () => ProjectOptions | undefined;
  isNetworkEnabled: () => boolean;
  readProfiles: () => CreationProfile[];
  isRunning: () => boolean;
  readJoinFailed: () => boolean;
  updated: () => Promise<boolean>;
  announceObserved: (summary: {
    repositoryId: string;
    environmentId: string;
    provisioned: boolean;
    login: string;
    running: boolean;
  }) => void;
  announceChanged: (repositoryId: string) => void;
  setBusy: (busy: boolean) => void;
  setStatus: (status: string) => void;
  setRepositoryName: (name: string) => void;
  setRepository: (repository: string) => void;
  setRepositoryURL: (url: string) => void;
  setProfiles: (profiles: CreationProfile[]) => void;
  setSelectedProfile: (profile: string) => void;
  setCanCreate: (canCreate: boolean) => void;
  setNetworkOptions: (options: ProjectOptions | undefined) => void;
  setNetworkReview: (review: boolean) => void;
  setNetworkEnabled: (enabled: boolean) => void;
  setEnvironment: (environment: Environment) => void;
  setDetail: (detail: Detail) => void;
  setJoinNeedsCheck: (needed: boolean) => void;
  setJoinFailed: (failed: boolean) => void;
  setOutcomeNeedsAttention: (attention: boolean) => void;
  setOutcome: (outcome: string) => void;
  setSaved: (saved: SavedKey[]) => void;
  setLifecycle: (lifecycle: Lifecycle) => void;
  setConnection: (connection: {command: string; fingerprint: string}) => void;
  setNetwork: (network: ProjectNetwork | undefined) => void;
  setNetworkNotice: (notice: string) => void;
  resetState: () => void;
}

function refreshBlocked(input: RefreshInput) {
  return input.isBusy() || input.isStale() || input.isDisposed() || !input.hasBinding();
}

function beginRefreshRead(input: RefreshInput): AbortController {
  const control = input.beginRead();
  input.setBusy(true);
  input.setStatus('Checking your account and environment…');
  return control;
}

export async function refresh(input: RefreshInput) {
  if (refreshBlocked(input)) return;
  const binding = input.readBinding();
  if (!binding) return;
  const {expectedUserId, repositoryId} = binding;
  const n = input.beginEpoch();
  const prior: RefreshPrior = {
    profile: input.readSelectedProfile(),
    network: input.readNetworkOptions(),
    enabled: input.isNetworkEnabled(),
  };
  const control = beginRefreshRead(input);
  const timeout = window.setTimeout(() => control.abort(), 15000);
  let recoveredJoin = false;
  try {
    recoveredJoin = await refreshAccount(input, n, expectedUserId, repositoryId, prior, control);
  } catch (error) {
    refreshFailed(input, n, error);
  } finally {
    await finishRefresh(input, n, timeout, repositoryId, recoveredJoin);
  }
}

async function refreshAccount(
  input: RefreshInput,
  n: number,
  expectedUserId: string | undefined,
  repositoryId: string,
  prior: RefreshPrior,
  control: AbortController
): Promise<boolean> {
  if (!input.isActive(n) || expectedUserId !== input.readBinding()?.expectedUserId) return false;
  const collection = await loadEnvironmentCollection(input, n, repositoryId, control);
  if (!collection || !input.isActive(n)) return false;
  if (collection.items.length) return refreshExisting(input, n, repositoryId, collection.items[0], control);
  await refreshEmpty(input, n, repositoryId, prior, collection.can_create, control);
  return false;
}

function admitCollection(
  collection: Record<string, unknown>,
  repository: Record<string, unknown>,
  repositoryId: string
) {
  check(
    repository.id === repositoryId &&
      Array.isArray(collection.items) &&
      collection.items.length <= 1 &&
      typeof collection.can_create === 'boolean'
  );
  check(typeof repository.owner === 'string' && typeof repository.name === 'string');
  for (const part of [repository.owner, repository.name]) admitRepositoryPart(part);
}

function applyRepositoryLabels(input: RefreshInput, repository: Record<string, unknown>, repositoryId: string) {
  const owner = repository.owner as string,
    name = repository.name as string;
  input.setRepositoryName(`${owner}/${name}`);
  input.setRepository(`Repository ${owner}/${name} · ID ${repositoryId}`);
  input.setRepositoryURL(
    location.origin +
      (input.readBinding()?.forgejoPrefix || '') +
      '/' +
      encodeURIComponent(owner) +
      '/' +
      encodeURIComponent(name)
  );
}

async function loadEnvironmentCollection(
  input: RefreshInput,
  n: number,
  repositoryId: string,
  control: AbortController
) {
  const collection = object(
    await input.api('/api/environments?repository_id=' + repositoryId, 'GET', undefined, control.signal)
  );
  const repository = object(collection.repository);
  if (!input.isActive(n)) return;
  admitCollection(collection, repository, repositoryId);
  applyRepositoryLabels(input, repository, repositoryId);
  return collection as typeof collection & {items: unknown[]; can_create: boolean};
}

async function refreshEmpty(
  input: RefreshInput,
  n: number,
  repositoryId: string,
  prior: RefreshPrior,
  canCreate: boolean,
  control: AbortController
) {
  if (canCreate && !(await refreshCreateOptions(input, n, repositoryId, prior, control))) return;
  if (!input.isActive(n)) return;
  input.setStatus(
    input.readPresentation() === 'journey'
      ? 'Creation is owner-only. You join separately after the project is ready.'
      : 'No shared environment. Creation is owner-only and does not join you.'
  );
}

function selectedCreateProfile(input: RefreshInput, prior: RefreshPrior) {
  if (input.readProfiles().some((p) => p.id === prior.profile)) return prior.profile;
  return input.readProfiles()[0]?.id || '';
}

async function refreshCreateOptions(
  input: RefreshInput,
  n: number,
  repositoryId: string,
  prior: RefreshPrior,
  control: AbortController
): Promise<boolean> {
  const available = object(
    await input.api('/api/repositories/' + repositoryId + '/profiles', 'GET', undefined, control.signal)
  );
  if (!input.isActive(n)) return false;
  check(Array.isArray(available.items) && available.items.length === 1);
  input.setProfiles(available.items.map(creationProfile));
  input.setSelectedProfile(selectedCreateProfile(input, prior));
  input.setCanCreate(!!input.readSelectedProfile());
  return refreshTailnetOptions(input, n, repositoryId, prior, control);
}

function networkReviewNeeded(prior: RefreshPrior, options: ProjectOptions) {
  if (!prior.enabled) return false;
  if (!options.available) return true;
  if (prior.network?.binding !== options.binding) return true;
  return prior.network.revision !== options.revision;
}

function networkEnabledAfterRefresh(input: RefreshInput, prior: RefreshPrior, options: ProjectOptions) {
  if (prior.enabled) return true;
  return input.readPresentation() !== 'journey' && options.available && options.default;
}

async function refreshTailnetOptions(
  input: RefreshInput,
  n: number,
  repositoryId: string,
  prior: RefreshPrior,
  control: AbortController
): Promise<boolean> {
  try {
    const options = projectOptions(
      await input.api('/api/repositories/' + repositoryId + '/tailnet-options', 'GET', undefined, control.signal)
    );
    if (!input.isActive(n)) return false;
    input.setNetworkOptions(options);
    input.setNetworkReview(networkReviewNeeded(prior, options));
    input.setNetworkEnabled(networkEnabledAfterRefresh(input, prior, options));
    return true;
  } catch (error) {
    return tailnetOptionsFailed(input, n, prior, error);
  }
}

function tailnetOptionsFailed(input: RefreshInput, n: number, prior: RefreshPrior, error: unknown): boolean {
  if (error instanceof SodaRequestError && error.status === 401) throw error;
  if (!input.isActive(n)) return false;
  input.setNetworkOptions(undefined);
  input.setNetworkEnabled(prior.enabled);
  input.setNetworkReview(prior.enabled);
  return true;
}

function applyJoinRecovery(input: RefreshInput, detail: Detail) {
  if (!input.readJoinFailed() || detail.authority_unavailable || detail.native_unavailable) return false;
  input.setJoinNeedsCheck(false);
  if (!detail.login) return false;
  input.setJoinFailed(false);
  input.setOutcomeNeedsAttention(false);
  input.setOutcome('');
  return true;
}

function environmentStatus(input: RefreshInput, detail: Detail) {
  if (!detail.environment.provisioned)
    return 'Provisioning incomplete. Ask the operator to inspect; do not recreate it.';
  if (detail.native_unavailable || !detail.observed)
    return 'Native state unavailable; refresh or ask the operator to inspect.';
  if (input.isRunning()) return 'Environment running.';
  return 'Environment stopped.';
}

function environmentNeedsAdminStart(input: RefreshInput, detail: Detail) {
  return (
    detail.environment.provisioned &&
    !input.isRunning() &&
    !detail.native_unavailable &&
    !detail.environment_administrator
  );
}

function applyEnvironmentStatus(input: RefreshInput, detail: Detail) {
  input.setStatus(environmentStatus(input, detail));
  if (environmentNeedsAdminStart(input, detail))
    input.setStatus(
      input.readStatus() +
        ' Ask the project administrator or Soda operator to Start it; nothing is started automatically.'
    );
}

async function refreshExisting(
  input: RefreshInput,
  n: number,
  repositoryId: string,
  item: unknown,
  control: AbortController
): Promise<boolean> {
  const environment = environmentResponse(item, repositoryId);
  input.setEnvironment(environment);
  const detail = detailResponse(
    await input.api(`/api/environments/${environment.id}`, 'GET', undefined, control.signal),
    environment
  );
  if (!input.isActive(n)) return false;
  input.setDetail(detail);
  const recoveredJoin = applyJoinRecovery(input, detail);
  if (detail.login) admitProjectLogin(detail.login);
  applyEnvironmentStatus(input, detail);
  if (!(await refreshOptionalDetails(input, n, environment, detail, control))) return false;
  return recoveredJoin;
}

async function refreshOptionalDetails(
  input: RefreshInput,
  n: number,
  environment: Environment,
  detail: Detail,
  control: AbortController
): Promise<boolean> {
  if (!(await refreshSavedKeysIfStandard(input, n, control))) return false;
  if (!(await refreshLifecycleIfAdmin(input, n, environment, detail, control))) return false;
  if (!(await refreshConnectionIfJoined(input, n, environment, detail, control))) return false;
  return refreshProjectNetworkIfEligible(input, n, environment, detail, control);
}

async function refreshSavedKeysIfStandard(input: RefreshInput, n: number, control: AbortController): Promise<boolean> {
  if (input.readPresentation() === 'journey') return true;
  return refreshSavedKeys(input, n, control);
}

async function refreshSavedKeys(input: RefreshInput, n: number, control: AbortController): Promise<boolean> {
  try {
    const saved = savedKeysResponse(await input.api('/api/me/development-keys', 'GET', undefined, control.signal));
    if (!input.isActive(n)) return false;
    input.setSaved(saved);
    return true;
  } catch {
    if (!input.isActive(n)) return false;
    input.setOutcome('External SSH keys unavailable. Browser-only Join remains independent.');
    return true;
  }
}

async function refreshLifecycleIfAdmin(
  input: RefreshInput,
  n: number,
  environment: Environment,
  detail: Detail,
  control: AbortController
): Promise<boolean> {
  if (!detail.environment.provisioned || !detail.environment_administrator) return true;
  const state = object(
      await input.api(`/api/environments/${environment.id}/lifecycle`, 'GET', undefined, control.signal)
    ),
    native = object(state.environment);
  if (!input.isActive(n)) return false;
  check(native.id === environment.id && typeof native.running === 'boolean' && typeof state.boot_enabled === 'boolean');
  input.setLifecycle({
    running: native.running,
    boot: state.boot_enabled,
  });
  return true;
}

function shouldLoadConnection(input: RefreshInput, detail: Detail) {
  return !!detail.login && input.isRunning() && input.readPresentation() !== 'journey';
}

function admitConnectionPayload(
  own: Record<string, unknown>,
  connection: Record<string, unknown>,
  native: Record<string, unknown>,
  environment: Environment,
  detail: Detail
) {
  check(
    own.login === detail.login &&
      native.id === environment.id &&
      native.running &&
      typeof native.ip === 'string' &&
      /^(?:[0-9]{1,3}\.){3}[0-9]{1,3}$/.test(native.ip) &&
      validIPv4(native.ip) &&
      fingerprint(connection.fingerprint)
  );
}

function validIPv4(ip: string) {
  return ip.split('.').every((octet) => Number(octet) <= 255);
}

async function refreshConnectionIfJoined(
  input: RefreshInput,
  n: number,
  environment: Environment,
  detail: Detail,
  control: AbortController
): Promise<boolean> {
  if (!shouldLoadConnection(input, detail)) return true;
  admitProjectLogin(detail.login!);
  const own = object(
    await input.api(`/api/environments/${environment.id}/connection`, 'GET', undefined, control.signal)
  );
  if (!input.isActive(n)) return false;
  const c = object(own.connection),
    native = object(c.environment);
  admitConnectionPayload(own, c, native, environment, detail);
  await input.updated();
  if (!input.isActive(n)) return false;
  input.setConnection({
    command: `ssh ${detail.login}@${native.ip}`,
    fingerprint: c.fingerprint as string,
  });
  return true;
}

function shouldLoadProjectNetwork(detail: Detail) {
  return detail.environment.provisioned && (!!detail.login || !!detail.environment_administrator);
}

async function refreshProjectNetworkIfEligible(
  input: RefreshInput,
  n: number,
  environment: Environment,
  detail: Detail,
  control: AbortController
): Promise<boolean> {
  if (!shouldLoadProjectNetwork(detail)) return true;
  return refreshProjectNetwork(input, n, environment, control);
}

async function refreshProjectNetwork(
  input: RefreshInput,
  n: number,
  environment: Environment,
  control: AbortController
): Promise<boolean> {
  try {
    const network = projectView(
      await input.api(`/api/environments/${environment.id}/tailnet`, 'GET', undefined, control.signal),
      environment.id
    );
    check(!network.saved);
    if (!input.isActive(n)) return false;
    input.setNetwork(network);
    return true;
  } catch (error) {
    return projectNetworkFailed(input, n, error);
  }
}

function projectNetworkFailed(input: RefreshInput, n: number, error: unknown): boolean {
  if (error instanceof SodaRequestError && error.status === 401) throw error;
  if (!input.isActive(n)) return false;
  input.setNetwork(undefined);
  input.setNetworkNotice(
    'Private network state unavailable. No disconnected state was inferred; ordinary project controls remain independent.'
  );
  return true;
}

function refreshErrorStatus(e: SodaRequestError) {
  if (e.code === 'profile_unavailable')
    return 'Installed Project OS unavailable or incompatible. Nothing was reserved, pulled or started; ask the operator to inspect.';
  return 'Could not confirm state. Refresh; do not infer absence or retry an uncertain action.';
}

function refreshFailed(input: RefreshInput, n: number, error: unknown) {
  if (!input.isActive(n)) return;
  const e = error instanceof SodaRequestError ? error : new SodaRequestError(0);
  input.resetState();
  input.setRepository('');
  input.setStatus(refreshErrorStatus(e));
}

async function finishRefresh(
  input: RefreshInput,
  n: number,
  timeout: number,
  repositoryId: string,
  recoveredJoin: boolean
) {
  window.clearTimeout(timeout);
  if (!input.isActive(n)) return;
  input.setBusy(false);
  input.announceObserved({
    repositoryId,
    environmentId: input.readEnvironment()?.id || '',
    provisioned: input.readDetail()?.environment.provisioned === true,
    login: input.readDetail()?.login || '',
    running: input.isRunning(),
  });
  if (recoveredJoin) input.announceChanged(repositoryId);
  await input.updated();
}
