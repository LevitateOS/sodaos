import type {Space, TerminalMetadata} from './sodaspaces-api.js';
import type {TerminalObservation} from './sodaspaces-attention.js';
import type {mountFactoryWatch} from './sodaspaces-factory.js';
import type {PreparedExtensionMount} from './soda-extension.js';
import type {LayoutEntry, Minimum} from './sodaspaces-layout.js';
import type {TerminalContext, mountTerminal} from './sodaspaces-terminal.js';

export type WorkspaceContext = {transport: PreparedExtensionMount; forgejoPrefix?: string} & (
  | {
      kind: 'native';
      repositoryId?: string;
      pageRepositoryId?: string;
    }
  | {
      kind: 'page';
    }
);
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
