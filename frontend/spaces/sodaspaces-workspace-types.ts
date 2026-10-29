import type {Space} from './sodaspaces-inventory-response.js';
import type {TerminalMetadata} from './sodaspaces-terminal-response.js';
import type {TerminalObservation} from './sodaspaces-attention.js';
import type {mountFactoryWatch} from './sodaspaces-factory.js';
import type {PreparedExtensionMount} from './soda-extension.js';
import type {LayoutEntry, Minimum} from './sodaspaces-layout.js';
import type {TerminalContext, mountTerminal} from './sodaspaces-terminal.js';
import type {MountContext} from './fountain-browser-contract.js';

export type WorkspaceView = 'feed' | 'app' | 'work' | 'plans' | 'terminal' | 'sessions' | 'project';

export type WorkspaceContext = {
  transport: PreparedExtensionMount;
  forgejoPrefix?: string;
  /** Native host programmatic navigation (H-ROUTE). */
  navigate?: MountContext['navigate'];
} & (
  | {
      kind: 'native';
      repositoryId?: string;
      pageRepositoryId?: string;
    }
  | {
      kind: 'page';
      repositoryId?: string;
      /** The SodaOS Terminals place: this repository's environment only, terminals and agent runs only. */
      terminalsOnly?: TerminalsOnly;
    }
);
/** What the Terminals place (soda-place-terminals.ts) gives a terminals-only workspace. */
export interface TerminalsOnly {
  /** Address of a run's record (its Feed card). */
  runLink: (run: string) => string;
}
/** Host-supplied presentation facts (H-ROUTE): display routing only, never authority. */
export interface WorkspacePresentation {
  mode: 'page' | 'panel';
  route: string;
  repositoryId?: string;
  visible: boolean;
}
export interface Slot {
  key: string;
  binding: TerminalContext;
  unread: boolean;
  readRequested: boolean;
  observedAt: number;
  observation: TerminalObservation | undefined;
  metadata: TerminalMetadata | undefined;
  proposedName?: string;
  minimum?: Minimum;
  unavailable: boolean;
  host: HTMLElement;
  terminal: ReturnType<typeof mountTerminal>;
}
export interface Row {
  key: string;
  entry?: LayoutEntry;
  metadata?: TerminalMetadata;
}
export interface PaneSession {
  entry: LayoutEntry;
  space: Space;
  slot: Slot | undefined;
}
export interface FactoryWatch {
  run: string;
  environmentId: string;
  repositoryId: string;
  role: string;
  issue?: string;
  attempt?: string;
  hidden: boolean;
  view?: ReturnType<typeof mountFactoryWatch>;
}
export interface Creation {
  pane: string;
  environmentId: string;
  name: string;
}
