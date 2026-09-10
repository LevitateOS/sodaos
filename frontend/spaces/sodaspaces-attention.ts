import {object, terminalID} from './sodaspaces-api.js';
import type {TerminalMetadata} from './sodaspaces-api.js';

export type ConnectionState = 'opening' | 'ready' | 'connection-lost' | 'attached-elsewhere' | 'ending' | 'unconfirmed' | 'ended' | 'unavailable';
export interface TerminalObservation {
  readonly kind: 'output' | 'state';
  readonly generation: number;
  readonly id: string | null;
  readonly requestId: string | null;
  readonly state: ConnectionState;
}
const states: readonly ConnectionState[] = ['opening', 'ready', 'connection-lost', 'attached-elsewhere', 'ending', 'unconfirmed', 'ended', 'unavailable'];
export function terminalObservation(value: unknown): TerminalObservation | undefined {
  const v = object(value);
  if (Object.keys(v).sort().join(',') !== 'generation,id,kind,requestId,state' ||
      (v.kind !== 'output' && v.kind !== 'state') || !Number.isSafeInteger(v.generation) || typeof v.generation !== 'number' || v.generation < 0 ||
      !(v.id === null || terminalID(v.id)) || !(v.requestId === null || terminalID(v.requestId))) return;
  const state = states.find(state => state === v.state);
  if (!state || v.kind === 'output' && (state !== 'ready' || !v.id)) return;
  return {kind: v.kind, generation: v.generation, id: v.id, requestId: v.requestId, state};
}
/** Observations, not semantic agent status. No inference from output or prose. */
export function attentionReason(metadata: TerminalMetadata | undefined, observedAt: number, now: number, connection?: ConnectionState): string {
  if (connection === 'unconfirmed' || metadata?.state === 'unconfirmed') return 'Outcome unconfirmed; exact slot retained';
  if (connection === 'ending' || metadata?.state === 'ending') return 'End pending; cleanup not confirmed';
  if (connection === 'ended' || metadata?.state === 'ended') return 'Native cleanup confirmed';
  if (connection === 'attached-elsewhere') return 'Attached elsewhere (observed writer)';
  if (connection === 'connection-lost') return 'Connection lost; cleanup not confirmed';
  if (connection === 'unavailable') return 'Connection unavailable; refresh exact status';
  if (!metadata || !observedAt || now - observedAt > 90000) return 'Status unavailable or stale; refresh';
  if (metadata.effective_until * 1000 <= now) return 'Observed deadline passed; cleanup not confirmed';
  if (metadata.effective_until * 1000 - now <= 300000) return 'Observed deadline within five minutes';
  return '';
}
