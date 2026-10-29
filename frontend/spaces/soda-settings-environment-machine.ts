// Settings → SodaOS → Environment as explicit states. The project controls own
// the requests and report facts (environment, lifecycle, the operation in
// flight and how the last one ended); `environmentPhase` turns them into
// exactly one phase the page shows, plus at most one result notice.
import type {EnvironmentOperation, EnvironmentResult} from './sodaspaces-project-settings-view.js';

export type EnvironmentPhase =
  /** The first status read has not finished. */
  | {kind: 'loading'}
  | {kind: 'absent'; canCreate: boolean}
  | {kind: 'creating'; since: number}
  | {kind: 'stopped'}
  | {kind: 'starting'; since: number}
  | {kind: 'running'}
  | {kind: 'stopping'; since: number}
  /** A status read failed or the server's answer was incomplete. */
  | {kind: 'unknown'};

export interface EnvironmentNotice {
  tone: 'good' | 'bad';
  text: string;
}

export interface EnvironmentFacts {
  reads: number;
  hasEnvironment: boolean;
  /** undefined when neither lifecycle nor detail says whether it runs. */
  running: boolean | undefined;
  canCreate: boolean;
  operation: EnvironmentOperation | undefined;
  result: EnvironmentResult | undefined;
}

export function environmentPhase(facts: EnvironmentFacts): EnvironmentPhase {
  const {operation} = facts;
  if (operation?.kind === 'create') return {kind: 'creating', since: operation.since};
  if (operation?.kind === 'start') return {kind: 'starting', since: operation.since};
  if (operation?.kind === 'stop') return {kind: 'stopping', since: operation.since};
  if (facts.reads === 0 && (!facts.result || facts.result.kind !== 'refresh')) return {kind: 'loading'};
  if (facts.result?.kind === 'refresh' && facts.result.failed) return {kind: 'unknown'};
  if (!facts.hasEnvironment) return {kind: 'absent', canCreate: facts.canCreate};
  if (facts.running === undefined) return {kind: 'unknown'};
  return {kind: facts.running ? 'running' : 'stopped'};
}

const done: Record<EnvironmentOperation['kind'], string> = {
  create: 'Environment created. Members with write access can now join it in Terminals.',
  start: 'Environment started.',
  stop: 'Environment stopped. Its files are kept.',
  refresh: '',
  inspect: '',
};

/** What the last finished operation means for the person; refreshes only speak on failure. */
export function environmentNotice(result: EnvironmentResult | undefined): EnvironmentNotice | undefined {
  if (!result) return undefined;
  if (result.failed)
    return {tone: 'bad', text: result.message || 'That did not finish. Refresh to see the current state.'};
  const text = done[result.kind];
  return text ? {tone: 'good', text} : undefined;
}
