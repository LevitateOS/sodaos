import {html} from 'lit';
import type {TemplateResult} from 'lit';

/** Welcome artwork is decorative, theme-bound vector geometry, not a control. */
export function renderWelcome(blocked: boolean, create: () => void) {
  return html` <svg class="soda-welcome-illustration" viewBox="0 0 112 112" aria-hidden="true" focusable="false">
      <path class="soda-welcome-accent" d="M79 19 102 32 79 45 56 32Z"></path>
      <path class="soda-welcome-outline" d="M56 6 102 32V80L56 106 10 80V32ZM10 32 56 58 102 32M56 58V106"></path>
    </svg>
    <h2 tabindex="-1">Create your first project</h2>
    <p class="soda-welcome-copy">
      <span>A persistent human project, connected to your repository.</span
      ><span>Develop, debug and intervene alongside factory work using browser terminals or SSH.</span>
    </p>
    <button class="ui primary button" ?disabled=${blocked} @click=${create}>
      <span aria-hidden="true">＋</span> Create project
    </button>
    <p class="soda-welcome-help">
      <a
        href="https://github.com/levitateos/sodaos/blob/main/docs/public/30-Use-Soda/20-projects-and-workspaces.md"
        target="_blank"
        rel="noopener noreferrer"
        >How human projects work <span aria-hidden="true">↗</span
        ><span class="soda-visually-hidden"> (opens in a new tab)</span></a
      >
    </p>`;
}

/** The two post-creation prompts share one composition and illustration area. */
export function renderWorkspaceIntro(view: {
  kind: 'project' | 'terminal' | 'unavailable' | 'stopped' | 'loading';
  heading: string;
  description: TemplateResult;
  action: TemplateResult;
  helper: TemplateResult;
  feedback?: TemplateResult;
}) {
  return html`<div class="soda-workspace-intro" data-state=${view.kind}>
    <div class="soda-workspace-intro-content">
      <svg class="soda-workspace-illustration" viewBox="0 0 128 104" aria-hidden="true" focusable="false">
        <g ?hidden=${view.kind !== 'project'}>
          <path class="soda-welcome-outline" d="M58 10 94 31V73L58 94 22 73V31ZM22 31 58 52 94 31M58 52V94"></path>
          <circle class="soda-created-check" cx="98" cy="81" r="16"></circle>
          <path class="soda-created-checkmark" d="m91 81 5 5 9-10"></path>
        </g>
        <g ?hidden=${view.kind !== 'terminal'}>
          <path class="soda-welcome-outline" d="M7 21H121V83H7ZM24 39 35 50 24 61"></path>
          <path class="soda-terminal-cursor" d="M44 61H60"></path>
        </g>
        <g ?hidden=${!['unavailable', 'stopped', 'loading'].includes(view.kind)}>
          <circle class="soda-welcome-outline" cx="64" cy="52" r="34"></circle>
          <path class="soda-welcome-outline" ?hidden=${view.kind !== 'unavailable'} d="M64 33V55M64 66V69"></path>
          <path class="soda-welcome-outline" ?hidden=${view.kind !== 'stopped'} d="M48 52H80"></path>
          <path class="soda-welcome-outline" ?hidden=${view.kind !== 'loading'} d="M64 30V52L79 61"></path>
        </g>
      </svg>
      <h2 tabindex="-1">${view.heading}</h2>
      <p class="soda-intro-description">${view.description}</p>
      <div class="soda-intro-action">${view.action}</div>
      <p class="soda-intro-helper">${view.helper}</p>
      ${view.feedback || ''}
    </div>
  </div>`;
}

export function renderWelcomeSteps(step?: 1 | 2) {
  return html`<ol
    class="soda-setup-footer soda-welcome-steps"
    aria-label=${step ? 'New project progress' : 'Getting started'}
    data-step=${step || 0}
  >
    <li aria-current=${step === 1 ? 'step' : 'false'}>
      <span aria-hidden="true">${step === 2 ? '✓' : '01'}</span> Choose a repository
    </li>
    <li aria-current=${step === 2 ? 'step' : 'false'}><span aria-hidden="true">02</span> Create a project</li>
    <li><span aria-hidden="true">03</span> Open a terminal</li>
  </ol>`;
}

/** Stateless chrome. Callbacks capture the owner's original entry/pane/project. */
export function renderMenu(label: string, glyph: string, body: TemplateResult, className = ''): TemplateResult {
  return html`
    <details class=${'soda-menu' + (className ? ' ' + className : '')}>
      <summary aria-label=${label} title=${label}>${glyph}</summary>
      <div>${body}</div>
    </details>
  `;
}
