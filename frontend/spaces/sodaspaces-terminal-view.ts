import {html} from 'lit';
import type {TemplateResult} from 'lit';
import type {TerminalState} from './sodaspaces-terminal-attachment.js';

/** A render-time projection only. No transport, identity lookup or mutable state. */
export interface TerminalPresentation {
  readonly ready: boolean;
  readonly disabled: boolean;
  readonly canConnect: boolean;
  readonly canEnd: boolean;
  readonly connectLabel: string;
  readonly login: string;
  readonly project: string;
  readonly name: string;
  readonly message: string;
  readonly notice: boolean;
  readonly screenVisible: boolean;
  readonly confirmingName: string | null;
  readonly canConfirm: boolean;
}
export interface TerminalCommands {
  readonly connect: () => void;
  readonly end: () => void;
  readonly confirmEnd: () => void;
  readonly cancelEnd: () => void;
  readonly project: () => void;
  readonly rename: () => void;
  readonly hide: () => void;
  readonly menuKey: (event: KeyboardEvent) => void;
}
/** Owner state and callbacks behind the presentation/command projections. No transport here. */
export interface TerminalViewInput {
  readDisposed: () => boolean;
  readState: () => TerminalState;
  readSessionID: () => string | undefined;
  readSessionName: () => string;
  readMessage: () => string;
  readNotice: () => boolean;
  readScreenVisible: () => boolean;
  readActionBusy: () => boolean;
  readManagedEnded: () => boolean;
  readConfirming: () => string | undefined;
  setConfirming: (confirming: string | undefined) => void;
  setRetries: (retries: number) => void;
  readBindingLogin: () => string;
  readBindingProject: () => string;
  isHiddenOrInert: () => boolean;
  closeMenu: () => void;
  focusSummary: () => void;
  focusControls: () => void;
  requestCancelEndFocus: (target: string) => void;
  dispatchWorkspaceCommand: (command: 'project' | 'rename' | 'hide') => void;
  connectNow: () => void;
  endNow: () => void;
}
function viewDisabled(input: TerminalViewInput) {
  return input.readDisposed() || input.readState() === 'stale';
}
function canConnectView(input: TerminalViewInput, disabled: boolean) {
  return !(
    disabled ||
    input.readManagedEnded() ||
    input.readState() === 'opening' ||
    input.readState() === 'ready' ||
    input.readActionBusy()
  );
}
function canEndView(input: TerminalViewInput, disabled: boolean) {
  return !disabled && !!input.readSessionID() && !input.readActionBusy();
}
function contextLabels(input: TerminalViewInput) {
  return {
    name: input.readSessionName() || 'Terminal',
    login: input.readBindingLogin(),
    project: input.readBindingProject(),
  };
}
export function terminalPresentation(input: TerminalViewInput): TerminalPresentation {
  const disabled = viewDisabled(input),
    labels = contextLabels(input);
  return {
    ready: input.readState() === 'ready',
    disabled,
    canConnect: canConnectView(input, disabled),
    canEnd: canEndView(input, disabled),
    connectLabel: input.readSessionID() ? 'Reconnect terminal' : 'Open terminal',
    name: labels.name,
    login: labels.login,
    project: labels.project,
    message: input.readMessage(),
    notice: input.readNotice(),
    screenVisible: input.readScreenVisible(),
    confirmingName: input.readConfirming() ? input.readSessionName() || input.readConfirming()! : null,
    canConfirm: !disabled && !input.readActionBusy() && input.readConfirming() === input.readSessionID(),
  };
}
function menuKey(input: TerminalViewInput, event: KeyboardEvent) {
  if (event.key === 'Escape') {
    event.preventDefault();
    event.stopPropagation();
    input.closeMenu();
    input.focusSummary();
  }
}
function connectFromControls(input: TerminalViewInput) {
  input.closeMenu();
  input.setRetries(0);
  input.connectNow();
}
function endConfirmedTerminal(input: TerminalViewInput) {
  // Admission must see the exact confirmed ID before the dialog is cleared.
  if (input.readConfirming() === input.readSessionID() && !input.isHiddenOrInert()) input.endNow();
  input.setConfirming(undefined);
}
function workspaceCommand(input: TerminalViewInput, command: 'project' | 'rename' | 'hide') {
  if (input.readDisposed() || input.readState() === 'stale' || input.isHiddenOrInert()) return;
  input.closeMenu();
  input.dispatchWorkspaceCommand(command);
}
function confirmEnd(input: TerminalViewInput) {
  if (
    !input.readSessionID() ||
    input.readDisposed() ||
    input.readState() === 'stale' ||
    input.readActionBusy() ||
    input.isHiddenOrInert()
  )
    return;
  input.closeMenu();
  input.setConfirming(input.readSessionID());
  input.requestCancelEndFocus(input.readSessionID()!);
}
function cancelEnd(input: TerminalViewInput) {
  input.setConfirming(undefined);
  input.focusControls();
}
export function terminalCommands(input: TerminalViewInput): TerminalCommands {
  return {
    connect: () => connectFromControls(input),
    end: () => confirmEnd(input),
    confirmEnd: () => endConfirmedTerminal(input),
    cancelEnd: () => cancelEnd(input),
    project: () => workspaceCommand(input, 'project'),
    rename: () => workspaceCommand(input, 'rename'),
    hide: () => workspaceCommand(input, 'hide'),
    menuKey: (event) => menuKey(input, event),
  };
}
function actions(view: TerminalPresentation, commands: TerminalCommands): TemplateResult {
  return html`
    <button
      type="button"
      class="ui basic button danger"
      data-action="end"
      ?disabled=${!view.canEnd}
      @click=${commands.end}
    >
      End terminal…
    </button>
  `;
}
function confirmation(view: TerminalPresentation, commands: TerminalCommands): TemplateResult {
  return html`
    <div
      class="soda-terminal-confirm"
      role="dialog"
      aria-label="End terminal confirmation"
      @keydown=${(event: KeyboardEvent) => {
        if (event.key === 'Escape') {
          event.preventDefault();
          event.stopPropagation();
          commands.cancelEnd();
        }
      }}
    >
      <h4>End “${view.confirmingName}” in ${view.project}?</h4>
      <p>
        Original account: ${view.login}. This ends this terminal and processes in its managed session. Unsaved
        in-process work will be lost. Files and independently managed services remain.
      </p>
      <button type="button" class="ui button" data-action="cancel-end" @click=${commands.cancelEnd}>Cancel</button>
      <button type="button" class="ui button danger" ?disabled=${!view.canConfirm} @click=${commands.confirmEnd}>
        End terminal
      </button>
    </div>
  `;
}
export function renderTerminal(view: TerminalPresentation, commands: TerminalCommands): TemplateResult {
  return html`
    <section class=${'soda-terminal' + (view.ready ? ' is-connected' : '')}>
      <div class="soda-terminal-context">
        <span title=${view.project}>${view.login} @ ${view.project}</span>
        <details class="soda-menu" @keydown=${commands.menuKey}>
          <summary aria-label="Terminal actions" data-action="controls">⋯</summary>
          <div>
            <p class="soda-menu-heading" title=${view.name}><span>${view.name}</span></p>
            <button type="button" class="ui button" ?disabled=${!view.canEnd} @click=${commands.rename}>
              Rename terminal
            </button>
            <button type="button" class="ui button" ?disabled=${view.disabled} @click=${commands.hide}>
              Hide terminal
            </button>
            <button type="button" class="ui button" ?disabled=${view.disabled} @click=${commands.project}>
              Project settings
            </button>
            <div class="soda-menu-separator"></div>
            ${actions(view, commands)}
          </div>
        </details>
      </div>
      ${!view.ready && view.canConnect ? html`<button type="button" class="ui primary button" @click=${commands.connect}>${view.connectLabel}</button>` : ''}
      <p class=${'soda-terminal-status' + (!view.notice ? ' soda-visually-hidden' : '')} role="status" tabindex="-1">
        ${view.message}
      </p>
      ${view.confirmingName !== null ? confirmation(view, commands) : ''}
      <!-- Xterm exclusively owns this stable, unconditional node's descendants. -->
      <div
        class="soda-terminal-screen"
        ?hidden=${!view.screenVisible}
        aria-label=${`Terminal for ${view.login}; Ctrl+Shift+Enter focuses terminal controls`}
        @keydown=${(event: KeyboardEvent) => {
          if (event.key === 'Escape') {
            event.preventDefault();
            event.stopPropagation();
          }
        }}
      ></div>
    </section>
  `;
}
