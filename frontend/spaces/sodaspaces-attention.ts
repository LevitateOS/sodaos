import {object, terminalID} from './sodaspaces-api.js';
import type {TerminalMetadata} from './sodaspaces-api.js';

export type ConnectionState =
  | 'opening'
  | 'ready'
  | 'connection-lost'
  | 'attached-elsewhere'
  | 'ending'
  | 'unconfirmed'
  | 'ended'
  | 'unavailable';
export interface TerminalObservation {
  readonly kind: 'output' | 'state';
  readonly generation: number;
  readonly id: string | null;
  readonly state: ConnectionState;
}
const states: readonly ConnectionState[] = [
  'opening',
  'ready',
  'connection-lost',
  'attached-elsewhere',
  'ending',
  'unconfirmed',
  'ended',
  'unavailable',
];

function observationShape(
  v: Record<string, unknown>
): v is Record<string, unknown> & {kind: 'output' | 'state'; generation: number; id: string | null} {
  if (Object.keys(v).sort().join(',') !== 'generation,id,kind,state') return false;
  if (v.kind !== 'output' && v.kind !== 'state') return false;
  if (!Number.isSafeInteger(v.generation) || typeof v.generation !== 'number' || v.generation < 0) return false;
  return v.id === null || terminalID(v.id);
}

function observationReady(kind: unknown, state: ConnectionState, id: unknown) {
  return kind !== 'output' || (state === 'ready' && !!id);
}

export function terminalObservation(value: unknown): TerminalObservation | undefined {
  const v = object(value);
  if (!observationShape(v)) return;
  const state = states.find((item) => item === v.state);
  if (!state || !observationReady(v.kind, state, v.id)) return;
  return {kind: v.kind, generation: v.generation, id: v.id, state};
}

function closedAttention(connection: ConnectionState | undefined, metadata: TerminalMetadata | undefined) {
  if (connection === 'unconfirmed') return 'Outcome unconfirmed; inspect this exact terminal';
  if (connection === 'ending' || metadata?.state === 'ending') return 'End pending; cleanup not confirmed';
  if (connection === 'ended' || metadata?.state === 'ended') return 'Native cleanup confirmed';
  return '';
}

function transportAttention(connection: ConnectionState | undefined, metadata: TerminalMetadata | undefined) {
  if (connection === 'attached-elsewhere' && metadata?.attached !== false)
    return 'Attached elsewhere (observed writer)';
  if (connection === 'connection-lost') return 'Connection lost; cleanup not confirmed';
  if (connection === 'unavailable') return 'Connection unavailable; refresh exact status';
  return '';
}

function staleAttention(metadata: TerminalMetadata | undefined, observedAt: number, now: number) {
  if (!metadata || !observedAt || now - observedAt > 90000) return 'Status unavailable or stale; refresh';
  return '';
}

/** Observations, not semantic agent status. No inference from output or prose. */
export function attentionReason(
  metadata: TerminalMetadata | undefined,
  observedAt: number,
  now: number,
  connection?: ConnectionState
): string {
  const closed = closedAttention(connection, metadata);
  if (closed) return closed;
  const transport = transportAttention(connection, metadata);
  if (transport) return transport;
  const stale = staleAttention(metadata, observedAt, now);
  if (stale) return stale;
  if (metadata?.attached && connection !== 'ready' && connection !== 'opening')
    return 'Attached elsewhere (observed writer)';
  return '';
}
