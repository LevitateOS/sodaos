import type {Action, Response} from './types';
import {decodeRunnerResponse} from '../../../frontend/runners/soda-runner-response';
export const coordinatorPath = '/usr/local/libexec/soda/soda-runners';
export const actions = Object.freeze(['list', 'create', 'start', 'stop', 'restart', 'remove']);
const actionSet = new Set(actions);
export function coordinatorCommand(action: string) {
  assertAction(action);
  return [coordinatorPath, action];
}
export function encodeRequest(action: string, payload: unknown) {
  assertAction(action);
  if (payload === null || Array.isArray(payload) || typeof payload !== 'object') throw new TypeError('runner request must be a JSON object');
  return `${JSON.stringify(payload)}\n`;
}
export function decodeResponse<A extends Action>(action: A, output: string): Response<A> {
  assertAction(action);
  return decodeRunnerResponse(action, JSON.parse(output));
}
function assertAction(action: string) {
  if (!actionSet.has(action)) throw new TypeError(`unsupported runner action: ${action}`);
}
