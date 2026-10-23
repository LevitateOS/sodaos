import {LitElement, html} from 'lit';
import {id, readSodaJSON} from './sodaspaces-api.js';
import type {PreparedExtensionMount} from './soda-extension.js';
import {newDecisionID} from './sodaspaces-intent-response.js';
import {factoryCommandId} from './sodaspaces-factory-response.js';
import {nativePath} from './sodaspaces-factory-activity.js';
import {factoryAccess} from './sodaspaces-factory-progress-response.js';
import {factoryIssueDetail, sodaErrorDetail} from './sodaspaces-factory-work-response.js';
import {
  acceptanceCommentIDs,
  acceptanceErrorMessage,
  acceptancePreview,
  acceptanceReceipt,
} from './sodaspaces-factory-acceptance-response.js';
import type {AcceptancePreview} from './sodaspaces-factory-acceptance-response.js';

export interface RequirementsAcceptanceContext {
  readonly transport: PreparedExtensionMount;
  readonly repository: string;
  readonly repositoryPath: string;
  readonly issue: string;
}
interface PrerequisiteChoice {
  readonly occurrence: string;
  readonly depends_on: string;
  readonly endpoint_repository: string;
  readonly endpoint_issue: string;
  readonly outcome: '' | 'code' | 'result';
}
interface AcceptanceCommand {
  readonly command_id: string;
  readonly decision_id: string;
  readonly predecessor?: string;
  readonly title_digest: string;
  readonly title_occurrence: string;
  readonly content_digest: string;
  readonly content_version: number;
  readonly native_revision: number;
  readonly sources: AcceptancePreview['comments'];
  readonly resolutions: AcceptancePreview['comments'];
  readonly prerequisites: readonly (PrerequisiteChoice & {prerequisite_acceptance?: string})[];
}

async function read(context: RequirementsAcceptanceContext, path: string): Promise<unknown> {
  const response = await context.transport.request(path, {method: 'GET', signal: AbortSignal.timeout(15000)});
  const value = await readSodaJSON(response);
  if (!response.ok)
    throw Error(
      acceptanceErrorMessage(value) ?? sodaErrorDetail(value).message ?? 'Current requirements are unavailable.'
    );
  return value;
}

// Native text stays in the native issue. Only the exact reviewed selections
// from acceptance-preview enter this existing maintainer decision contract.
export class SodaFactoryRequirementsAcceptance extends LitElement {
  static properties = {
    allowed: {state: true},
    message: {state: true},
    busy: {state: true},
    preview: {state: true},
    sources: {state: true},
    resolutions: {state: true},
    choices: {state: true},
    reviewed: {state: true},
    pending: {state: true},
  };
  declare private allowed: boolean;
  declare private message: string;
  declare private busy: boolean;
  declare private preview: AcceptancePreview | undefined;
  declare private sources: string;
  declare private resolutions: string;
  declare private choices: PrerequisiteChoice[];
  declare private reviewed: boolean;
  declare private pending: AcceptanceCommand | undefined;
  private current: RequirementsAcceptanceContext | undefined;
  private predecessor: string | undefined;
  private serial = 0;
  protected createRenderRoot() {
    return this;
  }
  constructor() {
    super();
    this.allowed = false;
    this.message = '';
    this.busy = false;
    this.sources = '';
    this.resolutions = '';
    this.choices = [];
    this.reviewed = false;
  }
  set context(context: RequirementsAcceptanceContext | undefined) {
    if (context && (!id(context.repository) || !id(context.issue) || !nativePath(context.repositoryPath)))
      throw Error('Invalid requirements acceptance context.');
    const prior = this.current;
    if (
      context?.transport === prior?.transport &&
      context?.repository === prior?.repository &&
      context?.repositoryPath === prior?.repositoryPath &&
      context?.issue === prior?.issue
    )
      return;
    this.current = context;
    this.serial++;
    this.allowed = false;
    this.preview = undefined;
    this.pending = undefined;
    this.predecessor = undefined;
    this.sources = '';
    this.resolutions = '';
    this.choices = [];
    this.reviewed = false;
    this.busy = false;
    this.message = '';
    if (context) void this.loadAuthority(context, this.serial);
  }
  disconnectedCallback() {
    super.disconnectedCallback();
    this.serial++;
    this.current = undefined;
  }
  private base(context: RequirementsAcceptanceContext): string {
    return `repositories/${context.repository}/factory/issues/${context.issue}`;
  }
  private async loadAuthority(context: RequirementsAcceptanceContext, serial: number) {
    this.allowed = false;
    try {
      const access = factoryAccess(
        await read(context, `repositories/${context.repository}/factory/access`),
        context.repository
      );
      if (serial !== this.serial) return;
      this.allowed = access.viewer.code_write && access.viewer.session_current;
      this.message = this.allowed
        ? ''
        : 'Accepting requirements needs current repository code-write authority and a current session.';
    } catch (error) {
      if (serial === this.serial)
        this.message = error instanceof Error ? error.message : 'Current acceptance authority is unavailable.';
    }
  }
  private changeSources(section: 'sources' | 'resolutions', event: Event) {
    this[section] = (event.target as HTMLInputElement).value;
    this.preview = undefined;
    this.reviewed = false;
    this.choices = [];
  }
  private async loadPreview() {
    const context = this.current;
    if (!context || !this.allowed || this.busy || this.pending) return;
    const serial = this.serial;
    this.busy = true;
    this.preview = undefined;
    this.reviewed = false;
    try {
      const sources = acceptanceCommentIDs(this.sources),
        resolutions = acceptanceCommentIDs(this.resolutions);
      const selected = [...sources, ...resolutions];
      if (new Set(selected).size !== selected.length || selected.length > 50)
        throw Error('Select at most 50 distinct comment IDs across requirements and resolutions.');
      const query = new URLSearchParams(selected.map((comment) => ['comment_id', comment])).toString();
      const [value, detailValue] = await Promise.all([
        read(context, `${this.base(context)}/acceptance-preview${query ? `?${query}` : ''}`),
        read(context, this.base(context)),
      ]);
      if (serial !== this.serial) return;
      const preview = acceptancePreview(value, context.repository, context.issue, selected);
      const detail = factoryIssueDetail(detailValue, context.repository, context.issue);
      this.predecessor = detail.status.acceptance?.id;
      this.choices = preview.dependencies.map((edge) => ({
        ...edge,
        endpoint_repository: '',
        endpoint_issue: '',
        outcome: '',
      }));
      this.preview = preview;
      this.message = 'Review the native issue, selected comments and every prerequisite outcome before accepting.';
    } catch (error) {
      if (serial === this.serial)
        this.message = error instanceof Error ? error.message : 'Requirements preview is unavailable.';
    } finally {
      if (serial === this.serial) this.busy = false;
    }
  }
  private changeChoice(index: number, field: 'endpoint_repository' | 'endpoint_issue' | 'outcome', event: Event) {
    const value = (event.target as HTMLInputElement | HTMLSelectElement).value;
    if (field === 'outcome' && value !== '' && value !== 'code' && value !== 'result') return;
    this.choices = this.choices.map((choice, at) =>
      at === index ? ({...choice, [field]: value} as PrerequisiteChoice) : choice
    );
    this.reviewed = false;
  }
  private async selectedPrerequisites(
    context: RequirementsAcceptanceContext
  ): Promise<AcceptanceCommand['prerequisites']> {
    return Promise.all(
      this.choices.map(async (choice) => {
        if (!id(choice.endpoint_repository) || !id(choice.endpoint_issue) || choice.outcome === '')
          throw Error('Choose the repository, issue number and required outcome for every prerequisite.');
        if (choice.outcome === 'result') return choice;
        const detail = factoryIssueDetail(
          await read(context, `repositories/${choice.endpoint_repository}/factory/issues/${choice.endpoint_issue}`),
          choice.endpoint_repository,
          choice.endpoint_issue
        );
        if (!detail.status.valid || detail.status.withdrawn || !detail.status.acceptance)
          throw Error(
            `Accept the current requirements of prerequisite #${choice.endpoint_issue} before choosing its code outcome.`
          );
        return {...choice, prerequisite_acceptance: detail.status.acceptance.id};
      })
    );
  }
  private async accept() {
    const context = this.current,
      preview = this.preview;
    if (!context || !this.allowed || this.busy || (!this.pending && (!preview || !this.reviewed))) return;
    const serial = this.serial;
    this.busy = true;
    try {
      if (!this.pending && preview) {
        const prerequisites = await this.selectedPrerequisites(context);
        if (serial !== this.serial) return;
        const sources = new Set(acceptanceCommentIDs(this.sources));
        this.pending = {
          command_id: factoryCommandId(),
          decision_id: newDecisionID(),
          ...(this.predecessor ? {predecessor: this.predecessor} : {}),
          title_digest: preview.title_digest,
          title_occurrence: preview.title_occurrence,
          content_digest: preview.body_digest,
          content_version: preview.content_version,
          native_revision: preview.native_revision,
          sources: preview.comments.filter((source) => sources.has(source.id)),
          resolutions: preview.comments.filter((source) => !sources.has(source.id)),
          prerequisites,
        };
      }
      const command = this.pending;
      if (!command) return;
      const response = await context.transport.request(`${this.base(context)}/acceptances`, {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify(command),
        signal: AbortSignal.timeout(30000),
      });
      const value = await readSodaJSON(response);
      if (serial !== this.serial) return;
      if (!response.ok) {
        if (response.status < 500) {
          this.pending = undefined;
          this.preview = undefined;
          this.reviewed = false;
        }
        throw Error(
          acceptanceErrorMessage(value) ??
            sodaErrorDetail(value).message ??
            'Acceptance was not confirmed. Refresh the preview before accepting current requirements.'
        );
      }
      acceptanceReceipt(value, command.command_id, command.decision_id);
      this.pending = undefined;
      this.preview = undefined;
      this.reviewed = false;
      this.message =
        'Requirements accepted. Factory will assess the accepted work against current grants and prerequisites.';
      this.dispatchEvent(new CustomEvent('soda-factory-requirements-accepted', {bubbles: true, composed: true}));
    } catch (error) {
      if (serial === this.serial)
        this.message = `${error instanceof Error ? error.message : 'Acceptance was not confirmed.'}${this.pending ? ' Retry the same acceptance to check its recorded result.' : ''}`;
    } finally {
      if (serial === this.serial) this.busy = false;
    }
  }
  private renderPreview(context: RequirementsAcceptanceContext) {
    const preview = this.preview;
    if (!preview) return '';
    const issueLink = `${context.repositoryPath}/issues/${context.issue}`;
    return html`<div data-requirements-preview>
      <p>
        Issue #${context.issue}, body revision ${preview.content_version}; native revision ${preview.native_revision}.
      </p>
      <p>
        ${this.predecessor ? 'This replaces the prior accepted requirements.' : 'This adopts the current issue requirements.'}
      </p>
      <ul>
        ${preview.comments.map((source) => html`<li><a href=${`${issueLink}#issuecomment-${source.id}`} target="_blank" rel="noreferrer">Review comment ${source.id}</a> (revision ${source.content_version})</li>`)}
      </ul>
      ${this.choices.length === 0 ? html`<p>No direct prerequisites.</p>` : html`<p>For every native prerequisite, confirm its address and the outcome this work needs.</p>`}
      ${this.choices.map(
        (choice, index) => html`<fieldset data-prerequisite=${choice.occurrence}>
          <legend>Native prerequisite ${choice.depends_on}</legend>
          <label
            >Prerequisite repository ID
            <input
              inputmode="numeric"
              .value=${choice.endpoint_repository}
              @input=${(event: Event) => this.changeChoice(index, 'endpoint_repository', event)}
          /></label>
          <label
            >Prerequisite issue number
            <input
              inputmode="numeric"
              .value=${choice.endpoint_issue}
              @input=${(event: Event) => this.changeChoice(index, 'endpoint_issue', event)}
          /></label>
          <label
            >Required outcome
            <select
              aria-label="Required outcome"
              .value=${choice.outcome}
              @change=${(event: Event) => this.changeChoice(index, 'outcome', event)}
            >
              <option value="">Choose an outcome</option>
              <option value="code">Code delivered under its current accepted requirements</option>
              <option value="result">An accepted result or resolution</option>
            </select></label
          >
        </fieldset>`
      )}
      <label
        ><input
          type="checkbox"
          .checked=${this.reviewed}
          @change=${(event: Event) => {
            this.reviewed = (event.target as HTMLInputElement).checked;
          }}
        />I reviewed the native issue, selected comments and prerequisite outcomes.</label
      >
      <button
        type="button"
        class="ui primary button"
        ?disabled=${this.busy || !this.reviewed}
        @click=${() => this.accept()}
      >
        ${this.predecessor ? 'Accept revised requirements' : 'Accept requirements'}
      </button>
    </div>`;
  }
  protected render() {
    const context = this.current;
    if (!context) return html``;
    return html`<section aria-label="Issue requirements acceptance">
      <h3>Issue requirements</h3>
      <p>
        <a href=${`${context.repositoryPath}/issues/${context.issue}`} target="_blank" rel="noreferrer"
          >Review issue #${context.issue} in the native forge</a
        >. Acceptance uses its exact title and body, plus the comments you select.
      </p>
      <p role="status">${this.message}</p>
      <button
        type="button"
        class="ui button"
        ?disabled=${this.busy}
        @click=${() => this.loadAuthority(context, this.serial)}
      >
        Refresh acceptance authority
      </button>
      <fieldset ?disabled=${!this.allowed || this.busy || this.pending !== undefined}>
        <legend>Optional discussion to include</legend>
        <label
          >Requirement comment IDs
          <input .value=${this.sources} @input=${(event: Event) => this.changeSources('sources', event)}
        /></label>
        <label
          >Resolution comment IDs
          <input .value=${this.resolutions} @input=${(event: Event) => this.changeSources('resolutions', event)}
        /></label>
        <p>
          Use native comment IDs separated by commas. Leave blank to accept only the issue and prerequisite outcomes.
        </p>
        <button type="button" class="ui button" @click=${() => this.loadPreview()}>Review current requirements</button>
      </fieldset>
      ${this.pending ? html`<button type="button" class="ui button" ?disabled=${this.busy || !this.allowed} @click=${() => this.accept()}>Retry same acceptance</button>` : this.renderPreview(context)}
    </section>`;
  }
}
customElements.define('soda-factory-requirements-acceptance', SodaFactoryRequirementsAcceptance);
