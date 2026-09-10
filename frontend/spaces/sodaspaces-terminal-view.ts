import {html} from 'lit';
import type {TemplateResult} from 'lit';

/** A render-time projection only. No transport, identity lookup or mutable state. */
export interface TerminalPresentation {
  readonly ready: boolean;
  readonly disabled: boolean;
  readonly canConnect: boolean;
  readonly canEnd: boolean;
  readonly canReturn: boolean;
  readonly connectLabel: string;
  readonly login: string;
  readonly project: string;
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
  readonly return: () => void;
  readonly keep: () => void;
  readonly project: () => void;
  readonly rename: () => void;
  readonly hide: () => void;
  readonly menuKey: (event: KeyboardEvent) => void;
}
function actions(view: TerminalPresentation, commands: TerminalCommands): TemplateResult {
  return html`
    <button type="button" class="ui primary button" ?disabled=${!view.canConnect}
      @click=${commands.connect}>${view.connectLabel}</button>
    <button type="button" class="ui basic button" data-action="end" ?disabled=${!view.canEnd}
      @click=${commands.end}>End terminal…</button>
    <button type="button" class="ui basic button" ?disabled=${!view.canReturn}
      @click=${commands.return}>Continue working</button>
    <button type="button" class="ui basic button" ?disabled=${!view.canEnd}
      @click=${commands.keep}>Keep for two hours</button>
  `;
}
function confirmation(view: TerminalPresentation, commands: TerminalCommands): TemplateResult {
  return html`
    <div class="soda-terminal-confirm" role="dialog" aria-label="End terminal confirmation"
      @keydown=${(event: KeyboardEvent) => {
        if (event.key === 'Escape') {event.preventDefault(); event.stopPropagation(); commands.cancelEnd();}
      }}>
      <h4>End “${view.confirmingName}” in ${view.project}?</h4>
      <p>Original account: ${view.login}. This ends this terminal and processes in its managed session.
        Unsaved in-process work will be lost. Files and independently managed services remain.</p>
      <button type="button" class="ui button" data-action="cancel-end" @click=${commands.cancelEnd}>Cancel</button>
      <button type="button" class="ui button" ?disabled=${!view.canConfirm} @click=${commands.confirmEnd}>End terminal</button>
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
            <button type="button" class="ui button" ?disabled=${view.disabled} @click=${commands.project}>Environment / access</button>
            <button type="button" class="ui button" ?disabled=${!view.canEnd} @click=${commands.rename}>Rename terminal</button>
            <button type="button" class="ui button" ?disabled=${view.disabled} @click=${commands.hide}>Hide session</button>
            ${actions(view, commands)}
          </div>
        </details>
      </div>
      <p class=${'soda-terminal-status' + (!view.notice ? ' soda-visually-hidden' : '')}
        role="status" tabindex="-1">${view.message}</p>
      ${view.confirmingName !== null ? confirmation(view, commands) : ''}
      <!-- Xterm exclusively owns this stable, unconditional node's descendants. -->
      <div class="soda-terminal-screen" ?hidden=${!view.screenVisible}
        aria-label=${`Terminal for ${view.login}; Ctrl+Shift+Enter focuses terminal controls`}
        @keydown=${(event: KeyboardEvent) => {if (event.key === 'Escape') {event.preventDefault(); event.stopPropagation();}}}></div>
    </section>
  `;
}
