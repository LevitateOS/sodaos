// The AI accounts dialog as an explicit state machine. Every view the dialog
// can show is one ProviderView; every server reply or person action is one
// ProvidersEvent. `transition` is pure so each state change is testable
// without a browser, and the element only performs the effects.
import type {Connection, Enrollment, ProviderID} from './soda-identity-response.js';

export const providerIDs: readonly ProviderID[] = ['codex', 'muse'];

export type ProviderView =
  /** Not read yet. */
  | {kind: 'unknown'}
  | {kind: 'disconnected'; previous: boolean}
  /** Connect was clicked; the provider CLI is preparing a device code. */
  | {kind: 'starting'; since: number; previous: boolean}
  /** The person completes sign-in on the provider's page with this code. */
  | {kind: 'awaiting'; attempt: Enrollment; since: number}
  | {kind: 'cancelling'; attempt: Enrollment}
  | {kind: 'connected'; connections: Connection[]}
  | {kind: 'disconnecting'; connections: Connection[]}
  | {kind: 'failed'; reason: string; previous: boolean};

export type LoadState = 'idle' | 'loading' | 'ready' | 'unavailable';

export interface ProvidersState {
  load: LoadState;
  /** Why the last read failed; shown with a Retry action. */
  loadError: string;
  /** A refresh after the first read keeps showing the previous views. */
  refreshing: boolean;
  providers: Record<ProviderID, ProviderView>;
}

export type ProvidersEvent =
  | {type: 'load-started'}
  | {type: 'loaded'; connections: Connection[]; enrollments: Enrollment[]; now: number}
  | {type: 'load-failed'; reason: string}
  | {type: 'start-requested'; provider: ProviderID; now: number}
  | {type: 'start-succeeded'; provider: ProviderID; attempt: Enrollment; now: number}
  | {type: 'start-failed'; provider: ProviderID; reason: string}
  | {type: 'cancel-requested'; provider: ProviderID}
  | {type: 'cancel-failed'; provider: ProviderID; reason: string}
  | {type: 'disconnect-requested'; provider: ProviderID}
  | {type: 'disconnect-failed'; provider: ProviderID; reason: string}
  | {type: 'dismissed'; provider: ProviderID}
  /** The transport went away (page retired or session changed). */
  | {type: 'reset'};

export function initialProvidersState(): ProvidersState {
  return {
    load: 'idle',
    loadError: '',
    refreshing: false,
    providers: {codex: {kind: 'unknown'}, muse: {kind: 'unknown'}},
  };
}

/** Views whose request is still in flight; a status read must not replace them. */
function inFlight(view: ProviderView): boolean {
  return view.kind === 'starting' || view.kind === 'cancelling' || view.kind === 'disconnecting';
}

function hadConnection(view: ProviderView): boolean {
  if (view.kind === 'connected' || view.kind === 'disconnecting') return true;
  if (view.kind === 'disconnected' || view.kind === 'starting' || view.kind === 'failed') return view.previous;
  return false;
}

function attemptFailure(attempt: Enrollment): string {
  if (attempt.state === 'expired')
    return 'The sign-in code expired before sign-in finished. Start again for a new code.';
  return attempt.error || 'The provider did not complete sign-in. Start again for a new code.';
}

/** The view a status read implies for one provider. */
function observed(
  previousView: ProviderView,
  connections: Connection[],
  attempt: Enrollment | undefined,
  now: number
): ProviderView {
  const ready = connections.filter((connection) => connection.state === 'ready');
  if (ready.length) return {kind: 'connected', connections: ready};
  const previous = connections.length > 0 || hadConnection(previousView);
  if (attempt?.state === 'pending') {
    const since = previousView.kind === 'awaiting' && previousView.attempt.id === attempt.id ? previousView.since : now;
    return {kind: 'awaiting', attempt, since};
  }
  // A finished attempt only matters while the person is still watching it.
  if (attempt && previousView.kind === 'awaiting' && previousView.attempt.id === attempt.id) {
    if (attempt.state === 'canceled') return {kind: 'disconnected', previous};
    if (attempt.state !== 'completed') return {kind: 'failed', reason: attemptFailure(attempt), previous};
  }
  if (previousView.kind === 'failed') return previousView;
  return {kind: 'disconnected', previous};
}

function update(state: ProvidersState, provider: ProviderID, view: ProviderView): ProvidersState {
  return {...state, providers: {...state.providers, [provider]: view}};
}

export function transition(state: ProvidersState, event: ProvidersEvent): ProvidersState {
  switch (event.type) {
    case 'load-started':
      return state.load === 'ready' ? {...state, refreshing: true} : {...state, load: 'loading', loadError: ''};
    case 'load-failed':
      // A failed refresh keeps the views already shown; only a first read is unavailable.
      return state.load === 'ready'
        ? {...state, refreshing: false, loadError: event.reason}
        : {...state, load: 'unavailable', loadError: event.reason, refreshing: false};
    case 'loaded': {
      const providers = {...state.providers};
      for (const provider of providerIDs) {
        const current = state.providers[provider];
        if (inFlight(current)) continue;
        // The newest attempt per provider wins; the broker lists live attempts only.
        const attempt = event.enrollments.filter((item) => item.provider_id === provider).at(-1);
        providers[provider] = observed(
          current,
          event.connections.filter((item) => item.provider_id === provider),
          attempt,
          event.now
        );
      }
      return {load: 'ready', loadError: '', refreshing: false, providers};
    }
    case 'start-requested': {
      const current = state.providers[event.provider];
      if (current.kind !== 'disconnected' && current.kind !== 'failed') return state;
      return update(state, event.provider, {kind: 'starting', since: event.now, previous: hadConnection(current)});
    }
    case 'start-succeeded':
      if (state.providers[event.provider].kind !== 'starting') return state;
      return update(state, event.provider, {kind: 'awaiting', attempt: event.attempt, since: event.now});
    case 'start-failed': {
      const current = state.providers[event.provider];
      if (current.kind !== 'starting') return state;
      return update(state, event.provider, {kind: 'failed', reason: event.reason, previous: current.previous});
    }
    case 'cancel-requested': {
      const current = state.providers[event.provider];
      if (current.kind !== 'awaiting') return state;
      return update(state, event.provider, {kind: 'cancelling', attempt: current.attempt});
    }
    case 'cancel-failed': {
      const current = state.providers[event.provider];
      if (current.kind !== 'cancelling') return state;
      return update(state, event.provider, {kind: 'failed', reason: event.reason, previous: false});
    }
    case 'disconnect-requested': {
      const current = state.providers[event.provider];
      if (current.kind !== 'connected') return state;
      return update(state, event.provider, {kind: 'disconnecting', connections: current.connections});
    }
    case 'disconnect-failed': {
      const current = state.providers[event.provider];
      if (current.kind !== 'disconnecting') return state;
      return update(state, event.provider, {kind: 'failed', reason: event.reason, previous: true});
    }
    case 'dismissed': {
      const current = state.providers[event.provider];
      if (current.kind !== 'failed') return state;
      return update(state, event.provider, {kind: 'disconnected', previous: current.previous});
    }
    case 'reset':
      return initialProvidersState();
  }
}

/** A provider whose state changes on the server without a person acting. */
export function needsPolling(state: ProvidersState): boolean {
  return providerIDs.some((provider) => state.providers[provider].kind === 'awaiting');
}

/** A provider whose on-screen elapsed time advances. */
export function hasClock(state: ProvidersState): boolean {
  return providerIDs.some((provider) => {
    const kind = state.providers[provider].kind;
    return kind === 'starting' || kind === 'awaiting';
  });
}

/** Read after cancellation or disconnection settles, so the views reflect the server. */
export function settled(state: ProvidersState, provider: ProviderID): ProvidersState {
  const current = state.providers[provider];
  if (current.kind === 'cancelling') return update(state, provider, {kind: 'disconnected', previous: false});
  if (current.kind === 'disconnecting') return update(state, provider, {kind: 'disconnected', previous: true});
  return state;
}

/**
 * JEV is connected once for the whole installation (by its operator) with a
 * TypeSafe key, not by a device sign-in, so it has its own small machine.
 */
export type JevView =
  | {kind: 'unknown'}
  | {kind: 'disconnected'; canManage: boolean}
  | {kind: 'checking'; since: number}
  | {kind: 'connected'; canManage: boolean}
  | {kind: 'disconnecting'}
  | {kind: 'failed'; reason: string; canManage: boolean};

export type JevEvent =
  | {type: 'loaded'; connected: boolean; canManage: boolean}
  | {type: 'load-failed'; reason: string}
  | {type: 'connect-requested'; now: number}
  | {type: 'connect-failed'; reason: string}
  | {type: 'disconnect-requested'}
  | {type: 'disconnect-failed'; reason: string}
  | {type: 'dismissed'};

function jevManager(view: JevView): boolean {
  return 'canManage' in view ? view.canManage : true;
}

export function jevTransition(view: JevView, event: JevEvent): JevView {
  switch (event.type) {
    case 'loaded':
      // A key check or disconnect in flight owns the card until it answers.
      if (view.kind === 'checking' || view.kind === 'disconnecting') return view;
      return event.connected
        ? {kind: 'connected', canManage: event.canManage}
        : {kind: 'disconnected', canManage: event.canManage};
    case 'load-failed':
      return view.kind === 'unknown' ? {kind: 'failed', reason: event.reason, canManage: false} : view;
    case 'connect-requested':
      return {kind: 'checking', since: event.now};
    case 'connect-failed':
      return {kind: 'failed', reason: event.reason, canManage: true};
    case 'disconnect-requested':
      return view.kind === 'connected' ? {kind: 'disconnecting'} : view;
    case 'disconnect-failed':
      return {kind: 'failed', reason: event.reason, canManage: true};
    case 'dismissed':
      return view.kind === 'failed' ? {kind: 'disconnected', canManage: jevManager(view)} : view;
  }
}
