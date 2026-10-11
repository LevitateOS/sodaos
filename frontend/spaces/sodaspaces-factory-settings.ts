import {LitElement, html} from 'lit';
import type {PreparedExtensionMount} from './soda-extension.js';
import {object, readSodaJSON} from './sodaspaces-api.js';
import {factoryCommandId} from './sodaspaces-factory-response.js';

type FactoryContext = {
  transport: PreparedExtensionMount;
  repository: string;
  environment: string;
  profile?: unknown;
};

type View = {
  status?: Record<string, unknown>;
  preparation?: Record<string, unknown>;
  connections: Record<string, unknown>[];
  grants: Record<string, unknown>[];
  usageBudget?: Record<string, unknown>;
  selectedConnection: string;
  message: string;
  busy: boolean;
};

function asObject(value: unknown) {
  return object(value);
}

function safeText(value: unknown) {
  return typeof value === 'string' && value.length <= 256 ? value : '';
}

function safeDetail(value: unknown) {
  if (typeof value === 'string') return safeText(value);
  return typeof value === 'number' && Number.isSafeInteger(value) ? String(value) : '';
}

function safeNumber(value: unknown, fallback: number) {
  return typeof value === 'number' && Number.isSafeInteger(value) ? value : fallback;
}

function factoryDecisionId() {
  return `d${crypto.randomUUID().replace(/-/g, '').slice(0, 24)}`;
}

function inputChecked(form: HTMLFormElement, name: string) {
  const value = form.elements.namedItem(name);
  return value instanceof HTMLInputElement && value.checked;
}

class FactoryRequestError extends Error {
  constructor(readonly status: number) {
    super(`Factory request was not confirmed (${status}). Refresh before retrying.`);
  }
}

function inputValue(form: HTMLFormElement, name: string) {
  const value = new FormData(form).get(name);
  return typeof value === 'string' ? value.trim() : '';
}

function integer(form: HTMLFormElement, name: string, minimum = 0) {
  const raw = inputValue(form, name);
  if (!/^(0|[1-9][0-9]*)$/.test(raw)) throw Error('Enter a whole number in the allowed range.');
  const value = Number(raw);
  if (!Number.isSafeInteger(value) || value < minimum) throw Error('Enter a whole number in the allowed range.');
  return value;
}

export class SodaFactorySettings extends LitElement {
  static properties = {view: {state: true}};
  declare private view: View;
  private contextValue: FactoryContext | undefined;
  private serial = 0;

  constructor() {
    super();
    this.contextValue = undefined;
    this.view = {connections: [], grants: [], selectedConnection: '', message: '', busy: false};
  }

  protected createRenderRoot() {
    return this;
  }

  set context(value: FactoryContext) {
    if (
      this.contextValue?.transport === value.transport &&
      this.contextValue.repository === value.repository &&
      this.contextValue.environment === value.environment &&
      JSON.stringify(this.contextValue.profile) === JSON.stringify(value.profile)
    )
      return;
    this.contextValue = value;
    this.serial++;
    this.view = {connections: [], grants: [], selectedConnection: '', message: '', busy: false};
    void this.refresh();
  }

  disconnectedCallback() {
    this.serial++;
    this.contextValue = undefined;
    super.disconnectedCallback();
  }

  private async request(path: string, method = 'GET', body?: Record<string, unknown>) {
    const context = this.contextValue;
    if (!context) throw Error('Reload this Forgejo page before changing factory settings.');
    const response = await context.transport.request(path.replace(/^\/api\//, ''), {
      method,
      headers: body ? {'Content-Type': 'application/json'} : {},
      ...(body ? {body: JSON.stringify(body)} : {}),
      signal: AbortSignal.timeout(15000),
    });
    let payload: unknown;
    try {
      payload = response.status === 204 ? null : await readSodaJSON(response);
    } catch {
      payload = null;
    }
    if (!response.ok) {
      if (response.status === 401 || response.status === 403) {
        this.dispatchEvent(new CustomEvent('soda-session-invalid', {bubbles: true, composed: true}));
      }
      if (response.status === 409 || response.status === 412)
        throw Error('Authority changed. Refresh this view and review the current revision before trying again.');
      throw new FactoryRequestError(response.status);
    }
    return payload;
  }

  private async refresh() {
    const context = this.contextValue;
    if (!context) return;
    const serial = this.serial;
    this.view = {...this.view, busy: true, message: ''};
    try {
      const [status, preparation] = await Promise.all([
        this.request(`/api/repositories/${context.repository}/factory`),
        context.environment
          ? this.request(`/api/environments/${context.environment}/preparation`)
          : Promise.resolve(undefined),
      ]);
      if (serial !== this.serial) return;
      let connectionRows: Record<string, unknown>[] = [];
      try {
        const connections = await this.request('/api/identity/connections');
        connectionRows = Array.isArray(connections) ? connections.map(asObject) : [];
      } catch {
        // Factory authority remains readable if identity is not enrolled yet.
      }
      const selected = connectionRows.some((item) => item.id === this.view.selectedConnection)
        ? this.view.selectedConnection
        : safeText(connectionRows[0]?.id);
      const related = await this.connectionFacts(selected);
      if (serial !== this.serial) return;
      this.view = {
        ...(status === undefined ? {} : {status: asObject(status)}),
        ...(preparation === undefined ? {} : {preparation: asObject(preparation)}),
        connections: connectionRows,
        selectedConnection: selected,
        ...related,
        message: '',
        busy: false,
      };
    } catch (error) {
      if (serial === this.serial)
        this.view = {
          ...this.view,
          message: error instanceof Error ? error.message : 'Factory status is unavailable.',
          busy: false,
        };
    }
  }

  private async connectionFacts(connection: string) {
    if (!connection) return {grants: [] as Record<string, unknown>[]};
    const [grants, usageBudget] = await Promise.all([
      this.request(`/api/identity/connections/${encodeURIComponent(connection)}/grants`),
      this.request(`/api/factory/connections/${encodeURIComponent(connection)}/usage-budget`).catch((error) => {
        if (error instanceof FactoryRequestError && error.status === 404) return undefined;
        throw error;
      }),
    ]);
    return {
      grants: Array.isArray(grants) ? grants.map(asObject) : [],
      ...(usageBudget === undefined ? {} : {usageBudget: asObject(usageBudget)}),
    };
  }

  private changeConnection = async (event: Event) => {
    const select = event.currentTarget;
    if (!(select instanceof HTMLSelectElement)) return;
    const connection = select.value;
    const {usageBudget: _oldBudget, ...withoutBudget} = this.view;
    this.view = {...withoutBudget, selectedConnection: connection, grants: [], busy: true};
    try {
      const facts = await this.connectionFacts(connection);
      this.view = {...this.view, ...facts, busy: false};
    } catch {
      this.view = {...this.view, busy: false, message: 'Provider connection details are unavailable. Refresh before changing sponsorship or budget.'};
    }
  };

  private async act(form: HTMLFormElement, action: (form: HTMLFormElement) => Promise<void>) {
    if (this.view.busy || !this.contextValue) return;
    const serial = this.serial;
    this.view = {...this.view, busy: true, message: 'Applying the explicit factory setting…'};
    try {
      await action(form);
      if (serial !== this.serial) return;
      await this.refresh();
      if (serial === this.serial) this.view = {...this.view, message: 'Change confirmed by the current API.'};
    } catch (error) {
      if (serial === this.serial)
        this.view = {
          ...this.view,
          message: error instanceof Error ? error.message : 'Change was not confirmed. Refresh before retrying.',
          busy: false,
        };
    }
  }

  private submit = (event: SubmitEvent) => {
    event.preventDefault();
    const form = event.currentTarget;
    if (!(form instanceof HTMLFormElement)) return;
    const action = form.dataset.action;
    const context = this.contextValue;
    if (!context || !action) return;
    void this.act(form, async (current) => {
      const status = this.view.status || {};
      const policy = asObject(status.policy || {});
      const publish = asObject(policy.publish_actor || {});
      const create = asObject(policy.create_actor || {});
      const review = asObject(policy.review_actor || {});
      const merge = asObject(policy.merge_actor || {});
      if (action === 'hold') {
        await this.request(`/api/environments/${context.environment}/preparation`, 'POST', {
          action: inputValue(current, 'hold_action'),
        });
      } else if (action === 'acceptance') {
        const currentPreparation = asObject(this.view.preparation || {});
        const preview = asObject(currentPreparation.decision_preview || {});
        const requirements = asObject(preview.requirements || {});
        const sourceInputs = Array.isArray(requirements.source_inputs)
          ? requirements.source_inputs.map(asObject).reduce<Record<string, string>>((selected, row) => {
              const path = safeText(row.path);
              const digest = safeText(row.digest);
              if (path && digest) selected[path] = digest;
              return selected;
            }, {})
          : {};
        if (
          !requirements.source_commit ||
          !requirements.setup_digest ||
          !requirements.inputs_digest ||
          !requirements.predecessor ||
          !Object.keys(sourceInputs).length
        )
          throw Error('The API has not supplied a complete, ready setup-input review. No acceptance was recorded.');
        await this.request(`/api/environments/${context.environment}/preparation/acceptances`, 'POST', {
          command_id: factoryCommandId(),
          decision_id: factoryDecisionId(),
          predecessor: safeText(requirements.predecessor),
          source_commit: safeText(requirements.source_commit),
          setup_digest: safeText(requirements.setup_digest),
          inputs_digest: safeText(requirements.inputs_digest),
          source_inputs: sourceInputs,
        });
      } else if (action === 'approval') {
        const currentPreparation = asObject(this.view.preparation || {});
        const preview = asObject(currentPreparation.decision_preview || {});
        const review = asObject(preview.approval || {});
        if (
          !review.requirement ||
          !review.effects_digest ||
          !review.readiness_digest ||
          !Array.isArray(review.roles) ||
          review.roles.length === 0
        )
          throw Error('The API has not supplied a complete privileged-effects review. No approval was recorded.');
        if (!inputChecked(current, 'verified'))
          throw Error('Review the privileged effects and separately verify current live services and resources before approval.');
        await this.request(`/api/environments/${context.environment}/preparation/actions`, 'POST', {
          command_id: factoryCommandId(),
          action: 'approve',
          decision_id: factoryDecisionId(),
          predecessor: safeText(review.predecessor) || undefined,
          requirement: safeText(review.requirement),
          effects_digest: safeText(review.effects_digest),
          readiness_digest: safeText(review.readiness_digest),
          verified:
            current.elements.namedItem('verified') instanceof HTMLInputElement &&
            (current.elements.namedItem('verified') as HTMLInputElement).checked,
        });
      } else if (action === 'capacity') {
        const capacity = asObject(status.appliance || status.capacity || {});
        await this.request('/api/factory/capacity', 'PUT', {
          command_id: factoryCommandId(),
          expected_revision: Number(capacity.revision || 0),
          max_concurrent_runs: integer(current, 'max_concurrent_runs', 1),
          max_queued: integer(current, 'max_queued'),
        });
      } else if (action === 'budget') {
        const connection = this.view.selectedConnection;
        if (!connection) throw Error('Select a provider connection from the current account inventory.');
        const path = `/api/factory/connections/${encodeURIComponent(connection)}/usage-budget`;
        let currentBudget: Record<string, unknown> = {};
        try {
          currentBudget = asObject(this.view.usageBudget || await this.request(path));
        } catch (error) {
          if (!(error instanceof FactoryRequestError) || error.status !== 404) throw error;
        }
        await this.request(path, 'PUT', {
          command_id: factoryCommandId(),
          expected_revision: Number(currentBudget.revision || 0),
          rolling_minutes: integer(current, 'rolling_minutes', 1),
        });
      } else if (action === 'sponsorship') {
        const connection = this.view.selectedConnection;
        if (!connection) throw Error('Select a provider connection from the current account inventory.');
        const sponsorships = Array.isArray(status.sponsorships) ? status.sponsorships.map(asObject) : [];
        const existing = sponsorships.find((item) => item.connection === connection) || {};
        const grant = this.view.grants.find((item) => item.id === inputValue(current, 'grant_id') && item.revoked !== true);
        if (!grant) throw Error('Select a current, unrevoked broker grant from this connection.');
        const roles = ['coder', 'reviewer'].filter((role) => {
          const element = current.elements.namedItem(role);
          return element instanceof HTMLInputElement && element.checked;
        });
        await this.request(
          `/api/repositories/${context.repository}/factory/sponsorships/${encodeURIComponent(connection)}`,
          'PUT',
          {
            command_id: factoryCommandId(),
          grant_id: safeText(grant.id),
            expected_revision: Number(existing.revision || 0),
            generation: safeNumber(this.view.connections.find((item) => item.id === connection)?.generation, 0),
            roles,
            allowance_minutes: integer(current, 'allowance_minutes', 1),
            max_concurrent: integer(current, 'max_concurrent', 1),
            active: inputChecked(current, 'sponsorship_active'),
          }
        );
      } else if (action === 'operator-grant') {
        const grant = asObject(status.operator_grant || {});
        await this.request(`/api/repositories/${context.repository}/factory/operator-grant`, 'PUT', {
          command_id: factoryCommandId(),
          expected_revision: Number(grant.revision || 0),
          max_concurrent: integer(current, 'operator_concurrent', 1),
          active: inputChecked(current, 'operator_active'),
        });
      } else if (action === 'environment-grant') {
        const grant = asObject(status.environment_grant || {});
        if (!context.profile) throw Error('Choose and review a supported Project profile first.');
        await this.request(`/api/repositories/${context.repository}/factory/environment-grant`, 'PUT', {
          command_id: factoryCommandId(),
          expected_revision: Number(grant.revision || 0),
          active: inputChecked(current, 'environment_active'),
          project: context.environment || undefined,
          profile: context.profile,
        });
      } else if (action === 'policy') {
        const policy = asObject(status.policy || {});
        const roles = {
          coder: {
            harness: inputValue(current, 'coder_harness'),
            harness_version: inputValue(current, 'coder_harness_version'),
            model: inputValue(current, 'coder_model'),
          },
          reviewer: {
            harness: inputValue(current, 'reviewer_harness'),
            harness_version: inputValue(current, 'reviewer_harness_version'),
            model: inputValue(current, 'reviewer_model'),
          },
        };
        const checks = inputValue(current, 'required_checks')
          .split(',')
          .map((check) => check.trim())
          .filter(Boolean);
        await this.request(`/api/repositories/${context.repository}/factory/policy`, 'PUT', {
          command_id: factoryCommandId(),
          expected_revision: Number(policy.revision || 0),
          enabled:
            current.elements.namedItem('policy_enabled') instanceof HTMLInputElement &&
            (current.elements.namedItem('policy_enabled') as HTMLInputElement).checked,
          paused:
            current.elements.namedItem('policy_paused') instanceof HTMLInputElement &&
            (current.elements.namedItem('policy_paused') as HTMLInputElement).checked,
          target_branch: inputValue(current, 'target_branch'),
          merge_method: 'fast-forward-only',
          required_checks: checks,
          roles,
          publish_actor: publish,
          create_actor: create,
          review_actor: review,
          merge_actor: merge,
          attempt_limits: {
            active_minutes: integer(current, 'active_minutes', 1),
            correction_cycles: integer(current, 'correction_cycles'),
          },
          max_concurrent: integer(current, 'policy_concurrent', 1),
        });
      }
    });
  };

  private submitButton(label: string) {
    return html`<button type="submit" class="ui button" ?disabled=${this.view.busy}>${label}</button>`;
  }

  protected render() {
    const context = this.contextValue;
    const status = this.view.status || {};
    const effective = asObject(status.effective || {});
    const capacity = asObject(status.appliance || status.capacity || {});
    const policy = asObject(status.policy || {});
    const roles = asObject(policy.roles || {});
    const publish = asObject(policy.publish_actor || {});
    const create = asObject(policy.create_actor || {});
    const review = asObject(policy.review_actor || {});
    const merge = asObject(policy.merge_actor || {});
    const coder = asObject(roles.coder || {});
    const reviewer = asObject(roles.reviewer || {});
    const limits = asObject(policy.attempt_limits || {});
    const prep = this.view.preparation || {};
    const decisionPreview = asObject(prep.decision_preview || {});
    const requirementPreview = asObject(decisionPreview.requirements || {});
    const approvalPreview = asObject(decisionPreview.approval || {});
    const rows = Array.isArray(status.sponsorships) ? status.sponsorships.map(asObject) : [];
    return html`<section class="soda-spaces-view" aria-label="Factory authority" data-factory-settings>
      <h2>Factory authority</h2>
      <p role="status" aria-live="polite">
        ${this.view.message || (this.view.busy ? 'Refreshing current authority…' : '')}
      </p>
      <button type="button" class="ui button" ?disabled=${this.view.busy} @click=${() => this.refresh()}>
        Refresh current authority
      </button>
      ${
        context
          ? html`<p>
                Effective: ${effective.effective === true ? 'yes' : 'no'}; missing:
                ${Array.isArray(effective.missing) ? effective.missing.join(', ') || 'none' : 'not reported'}.
              </p>
              <p>
                Dispatch: ${effective.dispatch_open === true ? 'open' : 'closed'}; queue
                ${asObject(status.queue || {}).queued ?? 'unknown'}; active
                ${asObject(status.queue || {}).active ?? 'unknown'}.
              </p>
              <p>
                Capacity: ${capacity.max_concurrent_runs ?? 'unset'} concurrent runs, ${capacity.max_queued ?? 'unset'}
                queued (revision ${capacity.revision ?? 0}).
              </p>
              ${rows.map((row) => html`<p>Sponsorship ${safeText(row.connection)} · ${row.active === true ? 'active' : 'inactive'} · revision ${row.revision ?? 0}.</p>`)}
              ${
                context.environment
                  ? html`<fieldset>
                      <legend>Project preparation</legend>
                      <p>
                        Hold: ${prep.hold === true ? 'active' : 'released'} (revision ${prep.hold_revision ?? 0});
                        requirements ${safeText(prep.requirements) || 'not accepted'}; approval
                        ${safeText(prep.approval) || 'not recorded'}.
                      </p>
                      <form data-action="hold" @submit=${this.submit}>
                        <select name="hold_action">
                          <option value="hold">Hold</option>
                          <option value="release">Release</option></select
                        >${this.submitButton('Apply hold state')}
                      </form>
                      <h3>Preparation evidence</h3>
                      ${Array.isArray(prep.preparations)
                        ? prep.preparations.map((item) => {
                            const row = asObject(item);
                            return html`<p>${safeText(row.role)} · ${safeText(row.phase)} · ${row.ready === true ? 'ready' : 'not ready'} · source ${safeText(row.source_commit)} · setup ${safeText(row.setup_digest)}${row.missing ? html` · missing ${safeText(row.missing)}` : ''}</p>`;
                          })
                        : html`<p>The API has not returned preparation inputs.</p>`}
                      <form data-action="acceptance" @submit=${this.submit}>
                        <h3>Accept exact setup inputs</h3>
                        <p>Decision IDs are generated here. The server-provided review binds the current source, setup, exact input paths, and predecessor.</p>
                        ${requirementPreview.source_commit
                          ? html`<p>Source ${safeText(requirementPreview.source_commit)} · setup ${safeText(requirementPreview.setup_digest)} · inputs ${safeText(requirementPreview.inputs_digest)} · predecessor ${safeText(requirementPreview.predecessor)}</p>
                              ${Array.isArray(requirementPreview.source_inputs)
                                ? html`<ul>${requirementPreview.source_inputs.map((input) => {
                                    const row = asObject(input);
                                    return html`<li>${safeText(row.path)} · ${safeText(row.digest)}</li>`;
                                  })}</ul>`
                                : html`<p>Exact input paths were not returned.</p>`}`
                          : html`<p>${safeText(decisionPreview.reason) || 'The API has not supplied a complete ready setup-input review.'}</p>`}
                        <button type="submit" class="ui button" ?disabled=${this.view.busy || !requirementPreview.source_commit || !requirementPreview.setup_digest || !requirementPreview.inputs_digest || !requirementPreview.predecessor || !Array.isArray(requirementPreview.source_inputs) || requirementPreview.source_inputs.length === 0}>Record exact-input acceptance</button>
                      </form>
                      <form data-action="approval" @submit=${this.submit}>
                        <h3>Approve reviewed effects</h3>
                        ${approvalPreview.requirement && approvalPreview.effects_digest && approvalPreview.readiness_digest && Array.isArray(approvalPreview.roles) && approvalPreview.roles.length > 0
                          ? html`<p>Requirement ${safeText(approvalPreview.requirement)} · effects ${safeText(approvalPreview.effects_digest)} · readiness ${safeText(approvalPreview.readiness_digest)}${approvalPreview.predecessor ? html` · predecessor ${safeText(approvalPreview.predecessor)}` : ''}</p>
                              ${Array.isArray(approvalPreview.roles)
                                ? html`<ul>${approvalPreview.roles.map((item) => {
                                    const row = asObject(item);
                                    return html`<li>${safeText(row.role)} · source ${safeText(row.source_commit)} · setup ${safeText(row.setup_digest)} · tools ${safeText(row.tools)} · setup exit ${safeDetail(row.setup_exit)} · check exit ${safeDetail(row.check_exit)}</li>`;
                                  })}</ul>`
                                : html`<p>Role effects were not returned.</p>`}
                              <label><input type="checkbox" name="verified" /> I reviewed these effects and separately checked that the current live services and resources are ready. The server preview does not perform that operational check.</label><button type="submit" class="ui button" ?disabled=${this.view.busy}>Record separate approval</button>`
                          : html`<p>${safeText(decisionPreview.reason) || 'The API has not supplied a reviewable effects and readiness preview. Approval remains unavailable; this screen never approves automatically.'}</p><button type="submit" class="ui button" disabled>Record separate approval</button>`}
                      </form>
                    </fieldset>`
                  : html`<p>No Project exists for this repository, so preparation controls are unavailable.</p>`
              }
              <fieldset>
                <legend>Appliance capacity</legend>
                <form data-action="capacity" @submit=${this.submit}>
                  ${this.numberInput('Concurrent runs', 'max_concurrent_runs', safeNumber(capacity.max_concurrent_runs, 1))}${this.numberInput('Queued runs', 'max_queued', safeNumber(capacity.max_queued, 0))}${this.submitButton('Save capacity')}
                </form>
              </fieldset>
              <fieldset>
                <legend>Project grants</legend>
                <form data-action="operator-grant" @submit=${this.submit}>
                  ${this.numberInput('Operator concurrent limit', 'operator_concurrent', safeNumber(asObject(status.operator_grant || {}).max_concurrent, 1))}<label
                    ><input
                      type="checkbox"
                      name="operator_active"
                      .checked=${asObject(status.operator_grant || {}).active === true}
                    />
                    Active operator grant</label
                  >${this.submitButton('Save operator grant')}
                </form>
                <form data-action="environment-grant" @submit=${this.submit}>
                  <p>The selected Project profile is also the profile that will be authorized.</p>
                  <label
                    ><input
                      type="checkbox"
                      name="environment_active"
                      .checked=${asObject(status.environment_grant || {}).active === true}
                    />
                    Permit factory Project setup</label
                  >${this.submitButton('Save Project grant')}
                </form>
              </fieldset>
              <fieldset>
                <legend>Repository policy</legend>
                <p>
                  Current native bindings: publish ${safeText(publish.actor_id)} / token ${safeText(publish.token_id)}, create ${safeText(create.actor_id)} / token ${safeText(create.token_id)}, review ${safeText(review.actor_id)} / token ${safeText(review.token_id)}, merge ${safeText(merge.actor_id)} / token ${safeText(merge.token_id)}. Bindings come from current policy and cannot be changed as raw IDs here.
                </p>
                <form data-action="policy" @submit=${this.submit}>
                  ${this.textInput('Target branch ref', 'target_branch', safeText(policy.target_branch))}${this.textInput('Required checks (comma separated)', 'required_checks', Array.isArray(policy.required_checks) ? policy.required_checks.join(', ') : '')}
                  ${this.textInput('Coder harness', 'coder_harness', safeText(coder.harness || 'codex'))}${this.textInput('Coder harness version', 'coder_harness_version', safeText(coder.harness_version))}${this.textInput('Coder model', 'coder_model', safeText(coder.model))}
                  ${this.textInput('Reviewer harness', 'reviewer_harness', safeText(reviewer.harness || 'codex'))}${this.textInput('Reviewer harness version', 'reviewer_harness_version', safeText(reviewer.harness_version))}${this.textInput('Reviewer model', 'reviewer_model', safeText(reviewer.model))}
                  ${this.numberInput('Active minutes per attempt', 'active_minutes', safeNumber(limits.active_minutes, 120))}${this.numberInput('Correction cycles', 'correction_cycles', safeNumber(limits.correction_cycles, 3))}${this.numberInput('Repository concurrent limit', 'policy_concurrent', safeNumber(policy.max_concurrent, 1))}
                  <label
                    ><input type="checkbox" name="policy_enabled" .checked=${policy.enabled === true} /> Enable
                    automatic factory dispatch</label
                  ><label
                    ><input type="checkbox" name="policy_paused" .checked=${policy.paused === true} /> Pause
                    dispatch</label
                  >
                  ${this.submitButton('Save repository policy')}
                </form>
              </fieldset>
              <fieldset>
                <legend>Provider allowance and sponsorship</legend>
                <label>Provider connection<select name="selected_connection" .value=${this.view.selectedConnection} @change=${this.changeConnection}>
                  <option value="">Select a current connection</option>
                  ${this.view.connections.map((connection) => html`<option value=${safeText(connection.id)}>${safeText(connection.label || connection.provider_id)} · ${safeText(connection.state)}</option>`)}
                </select></label>
                <form data-action="budget" @submit=${this.submit}>
                  <p>Current rolling allowance: ${this.view.usageBudget ? `${this.view.usageBudget.rolling_minutes} minutes (revision ${this.view.usageBudget.revision})` : 'not configured'}.</p>
                  ${this.numberInput('Rolling minutes', 'rolling_minutes', safeNumber(this.view.usageBudget?.rolling_minutes, 360))}${this.submitButton('Save connection budget')}
                </form>
                <form data-action="sponsorship" @submit=${this.submit}>
                  <label>Current broker grant<select name="grant_id"><option value="">Select a current grant</option>${this.view.grants.filter((grant) => grant.revoked !== true).map((grant) => html`<option value=${safeText(grant.id)}>${safeText(grant.project_id)} · ${safeText(grant.id)}</option>`)}</select></label><p>Credential generation ${safeNumber(this.view.connections.find((connection) => connection.id === this.view.selectedConnection)?.generation, 0)}.</p><label
                    ><input type="checkbox" name="coder" /> Coder role</label
                  ><label><input type="checkbox" name="reviewer" /> Reviewer role</label
                  >${this.numberInput('Allowance minutes', 'allowance_minutes', 60)}${this.numberInput('Concurrent sessions', 'max_concurrent', 1)}<label
                    ><input type="checkbox" name="sponsorship_active" /> Active sponsorship</label
                  >${this.submitButton('Save sponsorship')}
                </form>
              </fieldset> `
          : html`<p>Factory authority requires an active project binding.</p>`
      }
    </section>`;
  }

  private textInput(label: string, name: string, value = '') {
    return html`<label>${label}<input name=${name} autocomplete="off" .value=${value} /></label>`;
  }

  private numberInput(label: string, name: string, value: number) {
    return html`<label>${label}<input type="number" name=${name} min="0" .value=${String(value)} /></label>`;
  }
}

if (!customElements.get('soda-factory-settings')) customElements.define('soda-factory-settings', SodaFactorySettings);
