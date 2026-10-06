import {check, object, projectId, SodaRequestError} from './sodaspaces-api.js';
import {creationProfile} from './sodaspaces-project-response.js';
import {savedKeysResponse} from './sodaspaces-keys-response.js';
import {projectView} from '../tailnet/soda-tailnet-response.js';

const rejected = new Set([400, 401, 403, 404, 409, 413, 415, 422]);

function publicKeyToken(value: string | undefined) {
  if (!value) return;
  return value.trim().split(/\s+/).slice(0, 2).join(' ');
}

export interface MutationsInput {
  isBlocked: () => boolean;
  hasBinding: () => boolean;
  readEpoch: () => number;
  isActive: (generation: number) => boolean;
  readBindingRepository: () => string | undefined;
  readEnvironmentId: () => string | undefined;
  readDetailLogin: () => string | undefined;
  api: (path: string, method: string, body: Record<string, unknown>, signal: AbortSignal) => Promise<unknown>;
  refreshAfter: () => Promise<void>;
  announceOperation: () => void;
  announceChanged: (repositoryId: string | undefined) => void;
  setBusy: (busy: boolean) => void;
  setOutcomeNeedsAttention: (attention: boolean) => void;
  setJoinFailed: (failed: boolean) => void;
  setOutcome: (outcome: string) => void;
  setMutationPending: (pending: boolean) => void;
  setJoinNeedsCheck: (needed: boolean) => void;
  setCanCreate: (canCreate: boolean) => void;
}

function mutationObject(raw: unknown): Record<string, unknown> | null {
  return raw === null ? null : object(raw);
}

export async function mutate(
  input: MutationsInput,
  path: string,
  body: Record<string, unknown>,
  message: string,
  method = 'POST'
) {
  if (input.isBlocked() || !input.hasBinding()) return;
  const n = input.readEpoch();
  input.setBusy(true);
  input.setOutcomeNeedsAttention(false);
  input.setJoinFailed(false);
  input.setOutcome('Sending the explicit operation with the original page identity…');
  let dispatched = false,
    inspectCreation = false;
  const controller = new AbortController(),
    timeout = window.setTimeout(() => controller.abort(), 255000);
  try {
    dispatched = true;
    inspectCreation = await dispatchMutation(input, n, path, body, message, method, controller);
  } catch (error) {
    inspectCreation = applyMutateError(input, path, error, dispatched);
  } finally {
    await finishMutation(input, n, timeout, inspectCreation);
  }
}

async function dispatchMutation(
  input: MutationsInput,
  n: number,
  path: string,
  body: Record<string, unknown>,
  message: string,
  method: string,
  controller: AbortController
) {
  input.setMutationPending(true);
  input.announceOperation();
  input.setOutcome('Request dispatched. Closing does not cancel or undo native work.');
  const result = mutationObject(await input.api(path, method, body, controller.signal));
  if (!input.isActive(n)) return false;
  const next = checkMutationResult(input, path, method, body, result);
  completeMutation(input, next || message);
  input.setBusy(false);
  await input.refreshAfter();
  return false;
}

function completeMutation(input: MutationsInput, message: string) {
  input.setMutationPending(false);
  input.announceOperation();
  input.setOutcome(message);
  input.announceChanged(input.readBindingRepository());
}

function checkMutationResult(
  input: MutationsInput,
  path: string,
  method: string,
  body: Record<string, unknown>,
  result: Record<string, unknown> | null
): string | undefined {
  if (path === '/api/environments') return checkCreateMutation(input, body, result);
  if (path.endsWith('/join')) return checkJoinMutation(result);
  if (path.endsWith('/lifecycle')) return checkLifecycleMutation(input, body, result);
  if (path.endsWith('/tailnet')) return checkTailnetMutation(input, body, result);
  if (path.endsWith('/access-keys')) return checkAccessKeysMutation(input, body, result);
  if (method === 'DELETE') return checkDeleteMutation(result);
  if (path === '/api/me/development-keys') return checkSaveKeyMutation(body, result);
  return undefined;
}

function checkCreateMutation(
  input: MutationsInput,
  body: Record<string, unknown>,
  result: Record<string, unknown> | null
): string | undefined {
  check(
    result &&
      projectId(result.id) &&
      result.repository_id === input.readBindingRepository() &&
      result.provisioned === true &&
      creationProfile(result.profile).id === body.profile_id
  );
  if (object(body.tailnet).enabled !== true) return undefined;
  check(result.tailnet_outcome === 'queued' || result.tailnet_outcome === 'unconfirmed');
  input.setOutcomeNeedsAttention(true);
  if (result.tailnet_outcome === 'queued') return 'Project created. Network policy saved; enrollment queued.';
  return 'Project created. Network setup unconfirmed; inspect Network and explicitly retry there. Do not recreate the project.';
}

function checkJoinMutation(result: Record<string, unknown> | null): string | undefined {
  check(typeof result?.login === 'string' && /^[a-z][a-z0-9_-]{0,30}$/.test(result.login) && result.login !== 'root');
  return undefined;
}

function checkLifecycleMutation(
  input: MutationsInput,
  body: Record<string, unknown>,
  result: Record<string, unknown> | null
): string | undefined {
  check(
    result &&
      object(result.environment).id === input.readEnvironmentId() &&
      object(result.environment).running === (body.action === 'start') &&
      result.boot_enabled === (body.action === 'start')
  );
  return undefined;
}

function checkTailnetMutation(
  input: MutationsInput,
  body: Record<string, unknown>,
  result: Record<string, unknown> | null
): string | undefined {
  const network = projectView(result, input.readEnvironmentId() || '');
  check(network.saved && network.revision !== body.revision && network.enabled === (body.action !== 'disable'));
  if (network.outcome === 'queued')
    return 'Network policy saved; native work queued. Refresh observes the outcome without replay.';
  return 'Network policy saved; native outcome unconfirmed. Observe before retrying; no connection or disconnection was assumed.';
}

function checkAccessKeysMutation(
  input: MutationsInput,
  body: Record<string, unknown>,
  result: Record<string, unknown> | null
): string | undefined {
  check(
    result?.applied === true &&
      result.login === input.readDetailLogin() &&
      typeof result.revision === 'string' &&
      /^[0-9a-f]{64}$/.test(result.revision) &&
      JSON.stringify(result.installed_fingerprints) === JSON.stringify(body.saved_fingerprints)
  );
  return undefined;
}

function checkDeleteMutation(result: Record<string, unknown> | null): string | undefined {
  check(result?.removed === true && result.existing_project_access_changed === false);
  return undefined;
}

function checkSaveKeyMutation(
  body: Record<string, unknown>,
  result: Record<string, unknown> | null
): string | undefined {
  const keys = savedKeysResponse(result);
  check(typeof body.public_key === 'string');
  const publicKey = body.public_key;
  check(keys.some((k) => publicKeyToken(k.public_key) === publicKeyToken(publicKey)));
  return undefined;
}

function joinErrorMessage(input: MutationsInput, code: string | undefined, uncertain: boolean) {
  input.setJoinFailed(true);
  input.setJoinNeedsCheck(true);
  if (code === 'account_incomplete')
    return 'Soda could not finish setting up your project account. Ask your Soda administrator to check the account setup before you try again.';
  if (code === 'membership_not_saved')
    return 'Your account was set up, but Soda could not save your access to this project. Ask your Soda administrator to check your project access.';
  if (code === 'unsupported_linux_login')
    return 'Your Forgejo username cannot be used for a project account. Ask your Soda administrator for help choosing a supported username.';
  if (uncertain) return 'We couldn’t confirm whether you joined. Check join status before trying again.';
  return 'Your request to join was declined. Check your project access or ask your Soda administrator for help.';
}

function mutateReason(code: string | undefined) {
  if (code === 'profile_unavailable')
    return 'Installed Project OS unavailable. No reservation was created; refresh before another explicit action.';
  if (code === 'unsupported_linux_login')
    return 'Your Forgejo username is not supported as a Linux login. No automatic rename is performed.';
  if (code === 'invalid_public_key') return 'Provide one public SSH key without options or private key material.';
  if (code === 'saved_keys_changed') return 'Saved keys changed. Review them again before Apply.';
  if (code === 'owner_required') return 'Only the current human repository owner can create this environment.';
  if (code === 'not_provisioned') return 'Provisioning is incomplete. Ask the operator to inspect; do not recreate it.';
}

function mutateErrorMessage(input: MutationsInput, path: string, e: SodaRequestError, uncertain: boolean) {
  if (path.endsWith('/join')) return joinErrorMessage(input, e.code, uncertain);
  const reason = mutateReason(e.code);
  if (!uncertain && reason) return reason;
  if (path === '/api/environments' && uncertain)
    return 'Project creation could not be confirmed. Check the project status before trying again.';
  if (uncertain)
    return 'We couldn’t confirm that this change finished. Refresh status to check the result before trying again.';
  return 'This change could not be applied. Refresh status and review your settings before trying again.';
}

function applyMutateError(input: MutationsInput, path: string, error: unknown, dispatched: boolean) {
  const e = error instanceof SodaRequestError ? error : new SodaRequestError(0);
  const uncertain = dispatched && !rejected.has(e.status);
  const inspectCreation = uncertain && path === '/api/environments';
  if (inspectCreation) noteUnconfirmedCreation(input);
  input.setMutationPending(false);
  input.setOutcomeNeedsAttention(true);
  input.setOutcome(mutateErrorMessage(input, path, e, uncertain));
  return inspectCreation;
}

function noteUnconfirmedCreation(input: MutationsInput) {
  input.setCanCreate(false);
  input.announceChanged(input.readBindingRepository());
}

async function finishMutation(input: MutationsInput, n: number, timeout: number, inspectCreation: boolean) {
  window.clearTimeout(timeout);
  input.setMutationPending(false);
  if (!input.isActive(n)) return;
  input.setBusy(false);
  if (inspectCreation) await input.refreshAfter();
}
