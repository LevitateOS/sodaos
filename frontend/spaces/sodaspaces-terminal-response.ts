import {check, object} from './sodaspaces-api.js';

export const terminalID = (value: unknown): value is string =>
  typeof value === 'string' && /^[0-9a-f]{32}$/.test(value);
export interface TerminalIdentity {
  expectedUserId: string;
  repositoryId: string;
  environmentId: string;
  login: string;
}
export interface TerminalMetadata {
  id: string;
  environment_id: string;
  repository_id: string;
  user_id: string;
  login: string;
  name: string;
  created_at: number;
  ready: boolean;
  attached: boolean;
  state: 'opening' | 'ready' | 'ending' | 'ended';
}
function admitTerminalBinding(data: Record<string, unknown>, binding: TerminalIdentity): string {
  check(
    terminalID(data.id) &&
      data.environment_id === binding.environmentId &&
      data.repository_id === binding.repositoryId &&
      data.user_id === binding.expectedUserId &&
      data.login === binding.login
  );
  return data.id;
}
function admitTerminalName(data: Record<string, unknown>): string {
  check(typeof data.name === 'string' && Array.from(data.name).length <= 80 && !/[\p{Cc}\p{Cf}]/u.test(data.name));
  return data.name;
}
function admitTerminalClock(data: Record<string, unknown>): number {
  check(typeof data.created_at === 'number' && Number.isSafeInteger(data.created_at) && data.created_at > 0);
  return data.created_at;
}
function knownTerminalState(state: unknown): state is TerminalMetadata['state'] {
  return state === 'opening' || state === 'ready' || state === 'ending' || state === 'ended';
}
function admitTerminalState(data: Record<string, unknown>): Pick<TerminalMetadata, 'ready' | 'attached' | 'state'> {
  check(typeof data.ready === 'boolean' && typeof data.attached === 'boolean' && knownTerminalState(data.state));
  check(data.ready === (data.state === 'ready') && (!data.attached || data.ready));
  return {ready: data.ready, attached: data.attached, state: data.state};
}
export function terminalMetadata(value: unknown, binding: TerminalIdentity): TerminalMetadata {
  const data = object(value);
  return {
    id: admitTerminalBinding(data, binding),
    environment_id: binding.environmentId,
    repository_id: binding.repositoryId,
    user_id: binding.expectedUserId,
    login: binding.login,
    name: admitTerminalName(data),
    created_at: admitTerminalClock(data),
    ...admitTerminalState(data),
  };
}
export function terminalResponse(value: unknown, binding: TerminalIdentity): TerminalMetadata | null {
  const data = object(value);
  return data.terminal === null ? null : terminalMetadata(data.terminal, binding);
}
