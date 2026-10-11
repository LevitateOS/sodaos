import {LitElement, html, nothing} from 'lit';
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
  sourceSelection?: {
    source_commit: string;
    source_paths: string[];
    setup_files: string[];
    required_tools: string[];
  };
  sourcePreview?: Record<string, unknown>;
  selectedConnection: string;
  message: string;
  busy: boolean;
};

type PendingPreparation = {key: string; coder: string; reviewer: string};

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

function factoryPreparationId() {
  return `f${crypto.randomUUID().replace(/-/g, '').slice(0, 24)}`;
}

function inputChecked(form: HTMLFormElement, name: string) {
  const value = form.elements.namedItem(name);
  return value instanceof HTMLInputElement && value.checked;
}

function hasRequiredSetupFiles(sourcePaths: string[], setupFiles: string[]) {
  return ['setup.sh', 'check.sh'].every((name) => sourcePaths.includes(name) && setupFiles.includes(name));
}

function validSourcePreview(response: Record<string, unknown>, selection: NonNullable<View['sourceSelection']>) {
  const requirements = asObject(response.requirements || {});
  const inputs = Array.isArray(requirements.source_inputs) ? requirements.source_inputs.map(asObject) : [];
  const files = Array.isArray(response.files) ? response.files.map(asObject) : [];
  return (
    validSourceDigests(requirements) &&
    exactReturnedPaths(inputs, files, selection.source_paths) &&
    sameStringSet(
      Array.isArray(response.setup_files) ? response.setup_files.map(safeText) : [],
      selection.setup_files
    ) &&
    sameStringSet(
      Array.isArray(response.required_tools) ? response.required_tools.map(safeText) : [],
      selection.required_tools
    )
  );
}

function validSourceDigests(requirements: Record<string, unknown>) {
  return (
    /^[0-9a-f]{40}$/.test(safeText(requirements.source_commit)) &&
    /^[0-9a-f]{64}$/.test(safeText(requirements.setup_digest)) &&
    /^[0-9a-f]{64}$/.test(safeText(requirements.inputs_digest))
  );
}

function exactReturnedPaths(inputs: Record<string, unknown>[], files: Record<string, unknown>[], selected: string[]) {
  const paths = inputs.map((row) => safeText(row.path));
  const filePaths = files.map((row) => safeText(row.path));
  return (
    sameStringSet(paths, selected) &&
    sameStringSet(filePaths, selected) &&
    files.every((row) => typeof row.content === 'string')
  );
}

function selectedSourceInputs(requirements: Record<string, unknown>) {
  const rows = Array.isArray(requirements.source_inputs) ? requirements.source_inputs.map(asObject) : [];
  const selected = rows.reduce<Record<string, string>>((result, row) => {
    const path = safeText(row.path);
    const digest = safeText(row.digest);
    if (path && digest) result[path] = digest;
    return result;
  }, {});
  return safeText(requirements.source_commit) &&
    safeText(requirements.setup_digest) &&
    safeText(requirements.inputs_digest) &&
    Object.keys(selected).length
    ? selected
    : undefined;
}

function validEvidence(effects: string, readiness: string) {
  const byteLength = (value: string) => new TextEncoder().encode(value).length;
  return Boolean(effects && readiness && byteLength(effects) <= 8192 && byteLength(readiness) <= 8192);
}

function approvalMatches(preparation: Record<string, unknown>) {
  const requirement = safeText(preparation.requirements);
  return Boolean(
    requirement && safeText(preparation.approval) && safeText(preparation.approval_requirement) === requirement
  );
}

function matchingPreparations(
  preparation: Record<string, unknown>,
  requirement: string,
  approval: string,
  sourceCommit: string,
  setupDigest: string
) {
  const rows = Array.isArray(preparation.preparations) ? preparation.preparations.map(asObject) : [];
  return rows.filter(
    (row) =>
      row.requirements === requirement &&
      row.approval === approval &&
      row.source_commit === sourceCommit &&
      row.setup_digest === setupDigest
  );
}

function existingPreparation(rows: Record<string, unknown>[], role: string) {
  return rows.find((row) => row.role === role && !retiredPreparation(row)) || rows.find((row) => row.role === role);
}

function requiresReplacementReview(rows: Record<string, unknown>[]) {
  return ['soda-coder', 'soda-reviewer'].some((role) => {
    const row = existingPreparation(rows, role);
    return Boolean(row && retiredPreparation(row));
  });
}

function pendingPreparationIds(
  key: string,
  rows: Record<string, unknown>[],
  current: PendingPreparation | undefined,
  replaceTerminal: boolean
) {
  if (!current || current.key !== key) {
    const coder = existingPreparation(rows, 'soda-coder');
    const reviewer = existingPreparation(rows, 'soda-reviewer');
    return {
      key,
      coder: coder && !retiredPreparation(coder) ? safeText(coder.id) : factoryPreparationId(),
      reviewer: reviewer && !retiredPreparation(reviewer) ? safeText(reviewer.id) : factoryPreparationId(),
    };
  }
  if (!replaceTerminal) return current;
  return {
    ...current,
    coder: replacementId(rows, current.coder, replaceTerminal),
    reviewer: replacementId(rows, current.reviewer, replaceTerminal),
  };
}

function replacementId(rows: Record<string, unknown>[], id: string, confirmed: boolean) {
  const terminal = confirmed && rows.some((row) => safeText(row.id) === id && retiredPreparation(row));
  return terminal ? factoryPreparationId() : id;
}

function validDecisionReview(review: Record<string, unknown>) {
  return Boolean(
    review.requirement &&
    review.effects_digest &&
    review.readiness_digest &&
    Array.isArray(review.roles) &&
    review.roles.length > 0
  );
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

function inputLines(form: HTMLFormElement, name: string) {
  return [
    ...new Set(
      inputValue(form, name)
        .split(/\r?\n/)
        .map((value) => value.trim())
        .filter(Boolean)
    ),
  ];
}

function sameStrings(left: string[], right: string[]) {
  return left.length === right.length && left.every((value, index) => value === right[index]);
}

function sameStringSet(left: string[], right: string[]) {
  return sameStrings([...left].sort(), [...right].sort());
}

function retainPreview(preview: Record<string, unknown> | undefined, nextRequirement: string) {
  const requirement = asObject(asObject(preview || {}).requirements || {});
  return preview && safeText(requirement.predecessor) === nextRequirement ? preview : undefined;
}

function retiredPreparation(row: Record<string, unknown>) {
  return row.stopped === true || ['failed', 'stopped', 'interrupted'].includes(safeText(row.phase));
}

function matchesSourceSelection(form: HTMLFormElement, selection: NonNullable<View['sourceSelection']>) {
  return (
    inputValue(form, 'source_commit') === selection.source_commit &&
    sameStrings(inputLines(form, 'source_paths'), selection.source_paths) &&
    sameStrings(inputLines(form, 'setup_files'), selection.setup_files) &&
    sameStrings(inputLines(form, 'required_tools'), selection.required_tools)
  );
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
  private pendingPreparation?: PendingPreparation;

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

  private async request(path: string, method = 'GET', body?: Record<string, unknown>, timeoutMs = 15000) {
    const context = this.contextValue;
    if (!context) throw Error('Reload this Forgejo page before changing factory settings.');
    const response = await context.transport.request(path.replace(/^\/api\//, ''), {
      method,
      headers: body ? {'Content-Type': 'application/json'} : {},
      ...(body ? {body: JSON.stringify(body)} : {}),
      signal: AbortSignal.timeout(timeoutMs),
    });
    const payload = await this.responsePayload(response);
    this.requireSuccessfulResponse(response);
    return payload;
  }

  private async responsePayload(response: Response) {
    if (response.status === 204) return null;
    try {
      return await readSodaJSON(response);
    } catch {
      return null;
    }
  }

  private requireSuccessfulResponse(response: Response) {
    if (response.ok) return;
    if (response.status === 401 || response.status === 403)
      this.dispatchEvent(new CustomEvent('soda-session-invalid', {bubbles: true, composed: true}));
    if (response.status === 409 || response.status === 412)
      throw Error('Authority changed. Refresh this view and review the current revision before trying again.');
    throw new FactoryRequestError(response.status);
  }

  private async refresh() {
    const context = this.contextValue;
    if (!context) return;
    const serial = this.serial;
    const retained = {sourceSelection: this.view.sourceSelection, sourcePreview: this.view.sourcePreview};
    this.view = {...this.view, busy: true, message: ''};
    try {
      const next = await this.loadView(context, retained);
      if (serial !== this.serial) return;
      this.view = {...next, message: '', busy: false};
    } catch (error) {
      if (serial === this.serial) this.showRefreshError(error);
    }
  }

  private async loadView(
    context: FactoryContext,
    retained: {sourceSelection: View['sourceSelection']; sourcePreview: View['sourcePreview']}
  ): Promise<View> {
    const [status, preparation] = await Promise.all([
      this.request(`/api/repositories/${context.repository}/factory`),
      context.environment
        ? this.request(`/api/environments/${context.environment}/preparation`)
        : Promise.resolve(undefined),
    ]);
    const nextPreparation = asObject(preparation || {});
    const sourcePreview = retainPreview(retained.sourcePreview, safeText(nextPreparation.requirements));
    const connections = await this.loadConnections();
    const selected = connections.some((item) => item.id === this.view.selectedConnection)
      ? this.view.selectedConnection
      : safeText(connections[0]?.id);
    const related = await this.connectionFacts(selected);
    return {
      ...(status === undefined ? {} : {status: asObject(status)}),
      ...(preparation === undefined ? {} : {preparation: nextPreparation}),
      ...(retained.sourceSelection === undefined ? {} : {sourceSelection: retained.sourceSelection}),
      ...(sourcePreview === undefined ? {} : {sourcePreview}),
      connections,
      selectedConnection: selected,
      ...related,
      message: '',
      busy: false,
    };
  }

  private async loadConnections() {
    try {
      const connections = await this.request('/api/identity/connections');
      return Array.isArray(connections) ? connections.map(asObject) : [];
    } catch {
      return [];
    }
  }

  private showRefreshError(error: unknown) {
    this.view = {
      ...this.view,
      message: error instanceof Error ? error.message : 'Factory status is unavailable.',
      busy: false,
    };
  }

  private showActionError(error: unknown) {
    this.view = {
      ...this.view,
      message: error instanceof Error ? error.message : 'Change was not confirmed. Refresh before retrying.',
      busy: false,
    };
  }

  private completePreviewAction() {
    this.view = {...this.view, busy: false, message: 'Source preview returned by the current native API.'};
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
      this.view = {
        ...this.view,
        busy: false,
        message: 'Provider connection details are unavailable. Refresh before changing sponsorship or budget.',
      };
    }
  };

  private async act(form: HTMLFormElement, action: (form: HTMLFormElement) => Promise<void>, refreshAfter = true) {
    if (this.view.busy || !this.contextValue) return;
    const serial = this.serial;
    this.view = {...this.view, busy: true, message: 'Applying the explicit factory setting…'};
    try {
      await action(form);
    } catch (error) {
      if (serial === this.serial) this.showActionError(error);
      return;
    }
    if (serial !== this.serial) return;
    if (!refreshAfter) return this.completePreviewAction();
    await this.refresh();
    if (serial === this.serial) this.view = {...this.view, message: 'Change confirmed by the current API.'};
  }

  private submit = (event: SubmitEvent) => {
    event.preventDefault();
    const form = event.currentTarget;
    if (!(form instanceof HTMLFormElement) || !this.contextValue) return;
    const action = form.dataset.action;
    if (!action) return;
    const handlers: Record<string, (current: HTMLFormElement) => Promise<void>> = {
      hold: this.submitHold,
      'source-preview': this.submitSourcePreview,
      acceptance: this.submitAcceptance,
      'approval-initial': this.submitInitialApproval,
      prepare: this.submitPrepare,
      approval: this.submitApproval,
      capacity: this.submitCapacity,
      budget: this.submitBudget,
      sponsorship: this.submitSponsorship,
      'operator-grant': this.submitOperatorGrant,
      'environment-grant': this.submitEnvironmentGrant,
      policy: this.submitPolicy,
    };
    const handler = handlers[action];
    if (!handler) return;
    void this.act(form, handler, action !== 'source-preview');
  };

  private submitHold = async (form: HTMLFormElement) => {
    const context = this.contextValue;
    if (!context) throw Error('Reload this Forgejo page before changing factory settings.');
    await this.request(`/api/environments/${context.environment}/preparation`, 'POST', {
      action: inputValue(form, 'hold_action'),
    });
  };

  private submitSourcePreview = async (form: HTMLFormElement) => {
    const context = this.contextValue;
    if (!context) throw Error('Reload this Forgejo page before changing factory settings.');
    const serial = this.serial;
    const sourcePaths = inputLines(form, 'source_paths');
    const setupFiles = inputLines(form, 'setup_files');
    const requiredTools = inputLines(form, 'required_tools');
    if (!hasRequiredSetupFiles(sourcePaths, setupFiles) || setupFiles.some((path) => !sourcePaths.includes(path)))
      throw Error('Select setup.sh and check.sh as source and setup files before reviewing the native preview.');
    const selection = {
      source_commit: inputValue(form, 'source_commit'),
      source_paths: sourcePaths,
      setup_files: setupFiles,
      required_tools: requiredTools,
    };
    const response = asObject(
      await this.request(`/api/environments/${context.environment}/preparation/actions`, 'POST', {
        action: 'source-preview',
        ...selection,
      })
    );
    if (serial !== this.serial) return;
    if (!validSourcePreview(response, selection))
      throw Error('The native API did not return a complete source preview. No acceptance was recorded.');
    this.view = {...this.view, sourceSelection: selection, sourcePreview: response};
  };

  private submitAcceptance = async () => {
    const context = this.contextValue;
    if (!context) throw Error('Reload this Forgejo page before changing factory settings.');
    const serial = this.serial;
    const requirements = this.acceptanceRequirements();
    const sourceInputs = selectedSourceInputs(requirements);
    if (!sourceInputs)
      throw Error('The API has not supplied a complete, ready setup-input review. No acceptance was recorded.');
    const accepted = asObject(
      await this.request(`/api/environments/${context.environment}/preparation/acceptances`, 'POST', {
        command_id: factoryCommandId(),
        decision_id: factoryDecisionId(),
        predecessor: safeText(requirements.predecessor) || undefined,
        source_commit: safeText(requirements.source_commit),
        setup_digest: safeText(requirements.setup_digest),
        inputs_digest: safeText(requirements.inputs_digest),
        source_inputs: sourceInputs,
      })
    );
    if (serial !== this.serial) return;
    this.retainAcceptedHead(accepted);
  };

  private acceptanceRequirements() {
    const currentPreparation = asObject(this.view.preparation || {});
    const preview = asObject(currentPreparation.decision_preview || {});
    const selectionForm = this.querySelector<HTMLFormElement>('form[data-action="source-preview"]');
    const selection = this.view.sourceSelection;
    const sourcePreview = asObject(this.view.sourcePreview || {});
    const requirements = asObject(sourcePreview.requirements || preview.requirements || {});
    if (selection && selectionForm && !matchesSourceSelection(selectionForm, selection))
      throw Error('Source selection changed after preview. Review a new native preview before accepting.');
    return requirements;
  }

  private retainAcceptedHead(accepted: Record<string, unknown>) {
    const nextHead = safeText(accepted.head);
    if (!nextHead) throw Error('The accepted requirement head was not returned; refresh before continuing.');
    const currentPreview = asObject(this.view.sourcePreview || {});
    const currentRequirements = asObject(currentPreview.requirements || {});
    this.view = {
      ...this.view,
      sourcePreview: {...currentPreview, requirements: {...currentRequirements, predecessor: nextHead}},
    };
  }

  private submitInitialApproval = async (form: HTMLFormElement) => {
    const context = this.contextValue;
    if (!context) throw Error('Reload this Forgejo page before changing factory settings.');
    const currentPreparation = asObject(this.view.preparation || {});
    const effects = inputValue(form, 'effects_evidence');
    const readiness = inputValue(form, 'readiness_evidence');
    const requirement = safeText(currentPreparation.requirements);
    if (!requirement) throw Error('Accept the exact source inputs before recording initial operational approval.');
    if (!validEvidence(effects, readiness))
      throw Error('Record bounded, factual effects and readiness evidence before approval.');
    if (!inputChecked(form, 'verified'))
      throw Error('Review the installed service/resource facts and separately confirm they are ready before approval.');
    await this.request(`/api/environments/${context.environment}/preparation/actions`, 'POST', {
      command_id: factoryCommandId(),
      action: 'approve-initial',
      decision_id: factoryDecisionId(),
      predecessor: safeText(currentPreparation.approval) || undefined,
      requirement,
      verified: true,
      effects_evidence: effects,
      readiness_evidence: readiness,
    });
  };

  private submitPrepare = async (form: HTMLFormElement) => {
    const context = this.contextValue;
    if (!context) throw Error('Reload this Forgejo page before changing factory settings.');
    const input = this.prepareInput(form);
    await this.request(`/api/environments/${context.environment}/preparation/actions`, 'POST', input, 65000);
  };

  private prepareInput(form: HTMLFormElement) {
    const context = this.contextValue;
    if (!context) throw Error('Reload this Forgejo page before changing factory settings.');
    const currentPreparation = asObject(this.view.preparation || {});
    const selection = this.currentPreparationSelection();
    const {requirement, approval} = this.preparationHeads(currentPreparation);
    const identities = this.prepareIdentities(form, context, currentPreparation, selection, requirement, approval);
    return {
      action: 'prepare',
      requirement,
      approval,
      source_paths: selection.source_paths,
      setup_files: selection.setup_files,
      required_tools: selection.required_tools,
      coder_id: identities.coder,
      reviewer_id: identities.reviewer,
    };
  }

  private currentPreparationSelection() {
    const selection = this.view.sourceSelection;
    if (!selection || !Object.keys(asObject(this.view.sourcePreview || {})).length)
      throw Error('Review the exact native source preview before starting role preparation.');
    const selectionForm = this.querySelector<HTMLFormElement>('form[data-action="source-preview"]');
    if (selectionForm && !matchesSourceSelection(selectionForm, selection))
      throw Error('Source selection changed after preview. Review a new native preview before starting preparation.');
    return selection;
  }

  private prepareIdentities(
    form: HTMLFormElement,
    context: FactoryContext,
    preparation: Record<string, unknown>,
    selection: NonNullable<View['sourceSelection']>,
    requirement: string,
    approval: string
  ) {
    const sourcePreview = asObject(this.view.sourcePreview || {});
    const sourceRequirements = asObject(sourcePreview.requirements || {});
    const existingRows = matchingPreparations(
      preparation,
      requirement,
      approval,
      safeText(sourceRequirements.source_commit),
      safeText(sourceRequirements.setup_digest)
    );
    const replaceTerminal = inputChecked(form, 'replace_terminal_preparations');
    if (requiresReplacementReview(existingRows) && !replaceTerminal)
      throw Error(
        'Inspect the failed or stopped preparation identities, then confirm that new identities should replace them.'
      );
    const key = JSON.stringify({environment: context.environment, requirement, approval, source: selection});
    this.pendingPreparation = pendingPreparationIds(key, existingRows, this.pendingPreparation, replaceTerminal);
    const identities = this.pendingPreparation;
    if (!identities) throw Error('Preparation identities are unavailable; refresh before retrying.');
    return identities;
  }

  private preparationHeads(preparation: Record<string, unknown>) {
    const requirement = safeText(preparation.requirements);
    const approval = safeText(preparation.approval);
    if (!requirement || !approvalMatches(preparation))
      throw Error('Exact-input acceptance and separate initial operational approval are both required.');
    return {requirement, approval};
  }

  private submitApproval = async (form: HTMLFormElement) => {
    const context = this.contextValue;
    if (!context) throw Error('Reload this Forgejo page before changing factory settings.');
    const currentPreparation = asObject(this.view.preparation || {});
    const preview = asObject(currentPreparation.decision_preview || {});
    const review = asObject(preview.approval || {});
    if (!validDecisionReview(review))
      throw Error('The API has not supplied a complete privileged-effects review. No approval was recorded.');
    if (!inputChecked(form, 'verified'))
      throw Error(
        'Review the privileged effects and separately verify current live services and resources before approval.'
      );
    await this.request(`/api/environments/${context.environment}/preparation/actions`, 'POST', {
      command_id: factoryCommandId(),
      action: 'approve',
      decision_id: factoryDecisionId(),
      predecessor: safeText(review.predecessor) || undefined,
      requirement: safeText(review.requirement),
      effects_digest: safeText(review.effects_digest),
      readiness_digest: safeText(review.readiness_digest),
      verified:
        form.elements.namedItem('verified') instanceof HTMLInputElement &&
        (form.elements.namedItem('verified') as HTMLInputElement).checked,
    });
  };

  private submitCapacity = async (form: HTMLFormElement) => {
    const context = this.contextValue;
    if (!context) throw Error('Reload this Forgejo page before changing factory settings.');
    const status = this.view.status || {};
    const capacity = asObject(status.appliance || status.capacity || {});
    await this.request('/api/factory/capacity', 'PUT', {
      command_id: factoryCommandId(),
      expected_revision: Number(capacity.revision || 0),
      max_concurrent_runs: integer(form, 'max_concurrent_runs', 1),
      max_queued: integer(form, 'max_queued'),
    });
  };

  private submitBudget = async (form: HTMLFormElement) => {
    const context = this.contextValue;
    if (!context) throw Error('Reload this Forgejo page before changing factory settings.');
    const connection = this.view.selectedConnection;
    if (!connection) throw Error('Select a provider connection from the current account inventory.');
    const path = `/api/factory/connections/${encodeURIComponent(connection)}/usage-budget`;
    let currentBudget: Record<string, unknown> = {};
    try {
      currentBudget = asObject(this.view.usageBudget || (await this.request(path)));
    } catch (error) {
      if (!(error instanceof FactoryRequestError) || error.status !== 404) throw error;
    }
    await this.request(path, 'PUT', {
      command_id: factoryCommandId(),
      expected_revision: Number(currentBudget.revision || 0),
      rolling_minutes: integer(form, 'rolling_minutes', 1),
    });
  };

  private submitSponsorship = async (form: HTMLFormElement) => {
    const context = this.contextValue;
    if (!context) throw Error('Reload this Forgejo page before changing factory settings.');
    const status = this.view.status || {};
    const connection = this.view.selectedConnection;
    if (!connection) throw Error('Select a provider connection from the current account inventory.');
    const sponsorships = Array.isArray(status.sponsorships) ? status.sponsorships.map(asObject) : [];
    const existing = sponsorships.find((item) => item.connection === connection) || {};
    const grant = this.view.grants.find((item) => item.id === inputValue(form, 'grant_id') && item.revoked !== true);
    if (!grant) throw Error('Select a current, unrevoked broker grant from this connection.');
    const roles = ['coder', 'reviewer'].filter((role) => {
      const element = form.elements.namedItem(role);
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
        allowance_minutes: integer(form, 'allowance_minutes', 1),
        max_concurrent: integer(form, 'max_concurrent', 1),
        active: inputChecked(form, 'sponsorship_active'),
      }
    );
  };

  private submitOperatorGrant = async (form: HTMLFormElement) => {
    const context = this.contextValue;
    if (!context) throw Error('Reload this Forgejo page before changing factory settings.');
    const status = this.view.status || {};
    const grant = asObject(status.operator_grant || {});
    await this.request(`/api/repositories/${context.repository}/factory/operator-grant`, 'PUT', {
      command_id: factoryCommandId(),
      expected_revision: Number(grant.revision || 0),
      max_concurrent: integer(form, 'operator_concurrent', 1),
      active: inputChecked(form, 'operator_active'),
    });
  };

  private submitEnvironmentGrant = async (form: HTMLFormElement) => {
    const context = this.contextValue;
    if (!context) throw Error('Reload this Forgejo page before changing factory settings.');
    const status = this.view.status || {};
    const grant = asObject(status.environment_grant || {});
    if (!context.profile) throw Error('Choose and review a supported Project profile first.');
    await this.request(`/api/repositories/${context.repository}/factory/environment-grant`, 'PUT', {
      command_id: factoryCommandId(),
      expected_revision: Number(grant.revision || 0),
      active: inputChecked(form, 'environment_active'),
      project: context.environment || undefined,
      profile: context.profile,
    });
  };

  private submitPolicy = async (form: HTMLFormElement) => {
    const context = this.contextValue;
    if (!context) throw Error('Reload this Forgejo page before changing factory settings.');
    const status = this.view.status || {};
    const policy = asObject(status.policy || {});
    await this.request(`/api/repositories/${context.repository}/factory/policy`, 'PUT', this.policyInput(form, policy));
  };

  private policyInput(form: HTMLFormElement, policy: Record<string, unknown>) {
    const publish = asObject(policy.publish_actor || {});
    const create = asObject(policy.create_actor || {});
    const review = asObject(policy.review_actor || {});
    const merge = asObject(policy.merge_actor || {});
    const roles = {
      coder: {
        harness: inputValue(form, 'coder_harness'),
        harness_version: inputValue(form, 'coder_harness_version'),
        model: inputValue(form, 'coder_model'),
      },
      reviewer: {
        harness: inputValue(form, 'reviewer_harness'),
        harness_version: inputValue(form, 'reviewer_harness_version'),
        model: inputValue(form, 'reviewer_model'),
      },
    };
    const checks = inputValue(form, 'required_checks')
      .split(',')
      .map((check) => check.trim())
      .filter(Boolean);
    return {
      command_id: factoryCommandId(),
      expected_revision: Number(policy.revision || 0),
      enabled: inputChecked(form, 'policy_enabled'),
      paused: inputChecked(form, 'policy_paused'),
      target_branch: inputValue(form, 'target_branch'),
      merge_method: 'fast-forward-only',
      required_checks: checks,
      roles,
      publish_actor: publish,
      create_actor: create,
      review_actor: review,
      merge_actor: merge,
      attempt_limits: {
        active_minutes: integer(form, 'active_minutes', 1),
        correction_cycles: integer(form, 'correction_cycles'),
      },
      max_concurrent: integer(form, 'policy_concurrent', 1),
    };
  }

  private submitButton(label: string) {
    return html`<button type="submit" class="ui button" ?disabled=${this.view.busy}>${label}</button>`;
  }

  protected render() {
    return html`<section class="soda-spaces-view" aria-label="Factory authority" data-factory-settings>
      <h2>Factory authority</h2>
      <p role="status" aria-live="polite">
        ${this.view.message || (this.view.busy ? 'Refreshing current authority…' : '')}
      </p>
      <button type="button" class="ui button" ?disabled=${this.view.busy} @click=${() => this.refresh()}>
        Refresh current authority
      </button>
      ${this.renderAuthority()}
    </section>`;
  }

  private renderAuthority() {
    const context = this.contextValue;
    if (!context) return html`<p>Factory authority requires an active project binding.</p>`;
    const status = this.view.status || {};
    return html`${this.renderSummary(status)}
    ${context.environment ? this.renderPreparation() : html`<p>No Project exists for this repository, so preparation controls are unavailable.</p>`}
    ${this.renderCapacity(status)} ${this.renderProjectGrants(status)} ${this.renderPolicy(status)}
    ${this.renderProviderControls()}`;
  }

  private renderSummary(status: Record<string, unknown>) {
    const effective = asObject(status.effective || {});
    return html`${this.renderEffective(effective, status)} ${this.renderCapacitySummary(status)}
    ${this.renderSponsorshipSummary(status)}`;
  }

  private renderEffective(effective: Record<string, unknown>, status: Record<string, unknown>) {
    const queue = asObject(status.queue || {});
    return html`<p>
        Effective: ${effective.effective === true ? 'yes' : 'no'}; missing:
        ${Array.isArray(effective.missing) ? effective.missing.join(', ') || 'none' : 'not reported'}.
      </p>
      <p>
        Dispatch: ${effective.dispatch_open === true ? 'open' : 'closed'}; queue ${queue.queued ?? 'unknown'}; active
        ${queue.active ?? 'unknown'}.
      </p>`;
  }

  private renderCapacitySummary(status: Record<string, unknown>) {
    const capacity = asObject(status.appliance || status.capacity || {});
    return html`<p>
      Capacity: ${capacity.max_concurrent_runs ?? 'unset'} concurrent runs, ${capacity.max_queued ?? 'unset'} queued
      (revision ${capacity.revision ?? 0}).
    </p>`;
  }

  private renderSponsorshipSummary(status: Record<string, unknown>) {
    const rows = Array.isArray(status.sponsorships) ? status.sponsorships.map(asObject) : [];
    return html`${rows.map((row) => html`<p>Sponsorship ${safeText(row.connection)} · ${row.active === true ? 'active' : 'inactive'} · revision ${row.revision ?? 0}.</p>`)}`;
  }

  private renderPreparation() {
    const prep = this.view.preparation || {};
    return html`<fieldset>
      <legend>Project preparation</legend>
      <p>
        Hold: ${prep.hold === true ? 'active' : 'released'} (revision ${prep.hold_revision ?? 0}); requirements
        ${safeText(prep.requirements) || 'not accepted'}; approval ${safeText(prep.approval) || 'not recorded'}.
      </p>
      <form data-action="hold" @submit=${this.submit}>
        <select name="hold_action">
          <option value="hold">Hold</option>
          <option value="release">Release</option></select
        >${this.submitButton('Apply hold state')}
      </form>
      ${this.renderPreparationEvidence(prep)} ${this.renderSourceForm(prep)} ${this.renderAcceptanceForm(prep)}
      ${this.renderInitialApproval(prep)} ${this.renderPrepareForm(prep)} ${this.renderEffectsApproval(prep)}
    </fieldset>`;
  }

  private renderPreparationEvidence(prep: Record<string, unknown>) {
    if (!Array.isArray(prep.preparations)) return html`<p>The API has not returned preparation inputs.</p>`;
    return html`<h3>Preparation evidence</h3>
      ${prep.preparations.map((item) => {
        const row = asObject(item);
        return html`<p>
          ${safeText(row.role)} · ${safeText(row.phase)} · ID ${safeText(row.id)} ·
          ${row.ready === true ? 'ready' : 'not ready'} · source ${safeText(row.source_commit)} · setup
          ${safeText(row.setup_digest)}${row.missing ? html` · missing ${safeText(row.missing)}` : ''}
        </p>`;
      })}`;
  }

  private sourceReview(prep: Record<string, unknown>) {
    const decision = asObject(prep.decision_preview || {});
    const source = asObject(this.view.sourcePreview || {});
    return {
      source,
      decision,
      requirements: asObject(source.requirements || decision.requirements || {}),
      selection: this.view.sourceSelection,
    };
  }

  private renderSourceForm(prep: Record<string, unknown>) {
    const {source, decision, requirements, selection} = this.sourceReview(prep);
    const files = Array.isArray(source.files) ? source.files.map(asObject) : [];
    const inputs = Array.isArray(requirements.source_inputs) ? requirements.source_inputs.map(asObject) : [];
    const digests = new Map(inputs.map((row) => [safeText(row.path), safeText(row.digest)]));
    return html`<form data-action="source-preview" @submit=${this.submit}>
      <h3>Select and review repository source</h3>
      <p>
        Choose exact paths and required tools. Leave source commit blank to use the native approved base. The native API
        resolves the commit and computes all digests from repository bytes.
      </p>
      ${this.renderSourceFields(selection)} ${this.renderSourceDetails(requirements, decision, inputs, files, digests)}
      <button type="submit" class="ui button" ?disabled=${this.view.busy}>Preview selected source</button>
    </form>`;
  }

  private renderSourceFields(selection: View['sourceSelection']) {
    return html`<label
        >Source commit (optional)<input name="source_commit" value=${selection?.source_commit || ''} autocomplete="off"
      /></label>
      <label
        >Source paths, one per line<textarea name="source_paths" rows="5">
${selection?.source_paths.join('\n') || 'setup.sh\ncheck.sh'}</textarea>
      </label>
      <label
        >Approved setup files, one per line<textarea name="setup_files" rows="3">
${selection?.setup_files.join('\n') || 'setup.sh\ncheck.sh'}</textarea>
      </label>
      <label
        >Required tools, one per line<textarea name="required_tools" rows="3">
${selection?.required_tools.join('\n') || ''}</textarea>
      </label>`;
  }

  private renderSourceDetails(
    requirements: Record<string, unknown>,
    decision: Record<string, unknown>,
    inputs: Record<string, unknown>[],
    files: Record<string, unknown>[],
    digests: Map<string, string>
  ) {
    if (!requirements.source_commit)
      return html`<p>
        ${safeText(decision.reason) || 'Review the native source preview before accepting exact inputs.'}
      </p>`;
    return html`<p>
        Resolved source ${safeText(requirements.source_commit)} · setup digest ${safeText(requirements.setup_digest)} ·
        complete inputs ${safeText(requirements.inputs_digest)}
      </p>
      <ul>
        ${inputs.map((row) => html`<li>${safeText(row.path)} · ${safeText(row.digest)}</li>`)}
      </ul>
      <ul>
        ${files.map(
          (file) => html`<li>
            <details>
              <summary>${safeText(file.path)} · ${digests.get(safeText(file.path)) || 'source preview'}</summary>
              <pre>${typeof file.content === 'string' ? file.content : ''}</pre>
            </details>
          </li>`
        )}
      </ul>`;
  }

  private renderAcceptanceForm(prep: Record<string, unknown>) {
    const {decision, requirements} = this.sourceReview(prep);
    const sourceInputs = requirements.source_inputs;
    const canAccept = Boolean(
      requirements.source_commit &&
      requirements.setup_digest &&
      requirements.inputs_digest &&
      Array.isArray(sourceInputs) &&
      sourceInputs.length
    );
    return html`<form data-action="acceptance" @submit=${this.submit}>
      <h3>Accept exact setup inputs</h3>
      <p>
        Decision IDs are generated here. Acceptance binds the reviewed source commit, setup files, exact selected paths
        and predecessor.
      </p>
      ${this.renderAcceptedInputs(requirements, decision)}
      <button type="submit" class="ui button" ?disabled=${this.view.busy || !canAccept}>
        Record exact-input acceptance
      </button>
    </form>`;
  }

  private renderAcceptedInputs(requirements: Record<string, unknown>, decision: Record<string, unknown>) {
    if (!requirements.source_commit)
      return html`<p>
        ${safeText(decision.reason) || 'The API has not supplied a complete ready setup-input review.'}
      </p>`;
    if (!Array.isArray(requirements.source_inputs)) return html`<p>Exact input paths were not returned.</p>`;
    return html`<p>
        Source ${safeText(requirements.source_commit)} · setup ${safeText(requirements.setup_digest)} · inputs
        ${safeText(requirements.inputs_digest)} · predecessor ${safeText(requirements.predecessor)}
      </p>
      <ul>
        ${requirements.source_inputs.map((input) => {
          const row = asObject(input);
          return html`<li>${safeText(row.path)} · ${safeText(row.digest)}</li>`;
        })}
      </ul>`;
  }

  private approvalMatchesRequirement(prep: Record<string, unknown>) {
    return Boolean(safeText(prep.approval) && safeText(prep.approval_requirement) === safeText(prep.requirements));
  }

  private renderInitialApproval(prep: Record<string, unknown>) {
    if (!safeText(prep.requirements) || this.approvalMatchesRequirement(prep)) return nothing;
    return html`<form data-action="approval-initial" @submit=${this.submit}>
      <h3>Approve initial Project operations</h3>
      <p>
        Review the required privileged effects and separately record the live installed service, resource and readiness
        facts you checked. The server does not infer or verify this operational review.
      </p>
      <label>Privileged effects reviewed<textarea name="effects_evidence" rows="3" maxlength="8192"></textarea></label>
      <label
        >Installed services and resources checked<textarea
          name="readiness_evidence"
          rows="3"
          maxlength="8192"
        ></textarea>
      </label>
      <label
        ><input type="checkbox" name="verified" /> I reviewed these exact effects and checked the current live services
        and resources.</label
      >
      <button type="submit" class="ui button" ?disabled=${this.view.busy}>Record initial operational approval</button>
    </form>`;
  }

  private matchingPreparationRows(prep: Record<string, unknown>) {
    const {requirements} = this.sourceReview(prep);
    const rows = Array.isArray(prep.preparations) ? prep.preparations.map(asObject) : [];
    return rows.filter(
      (row) =>
        row.requirements === safeText(prep.requirements) &&
        row.approval === safeText(prep.approval) &&
        row.source_commit === safeText(requirements.source_commit) &&
        row.setup_digest === safeText(requirements.setup_digest)
    );
  }

  private renderPrepareForm(prep: Record<string, unknown>) {
    if (!safeText(prep.requirements) || !this.approvalMatchesRequirement(prep)) return nothing;
    const {source} = this.sourceReview(prep);
    const hasRetired = this.matchingPreparationRows(prep).some(retiredPreparation);
    return html`<form data-action="prepare" @submit=${this.submit}>
      <p>
        Exact-input acceptance and initial operational approval are recorded. Starting preparation creates separate
        coder and reviewer identities from this exact selection.
      </p>
      ${hasRetired ? html`<label><input type="checkbox" name="replace_terminal_preparations" /> I inspected the failed or stopped preparation identities and authorize new identities for those roles.</label>` : nothing}
      <button
        type="submit"
        class="ui button"
        ?disabled=${this.view.busy || !this.view.sourceSelection || !Object.keys(source).length}
      >
        Start role preparation
      </button>
    </form>`;
  }

  private renderEffectsApproval(prep: Record<string, unknown>) {
    const decision = asObject(prep.decision_preview || {});
    const review = asObject(decision.approval || {});
    return html`<form data-action="approval" @submit=${this.submit}>
      <h3>Approve reviewed effects</h3>
      ${this.renderApprovalReview(review, decision)}
    </form>`;
  }

  private renderApprovalReview(review: Record<string, unknown>, decision: Record<string, unknown>) {
    if (!validDecisionReview(review))
      return html`<p>
          ${safeText(decision.reason) || 'The API has not supplied a reviewable effects and readiness preview. Approval remains unavailable; this screen never approves automatically.'}
        </p>
        <button type="submit" class="ui button" disabled>Record separate approval</button>`;
    return html`<p>
        Requirement ${safeText(review.requirement)} · effects ${safeText(review.effects_digest)} · readiness
        ${safeText(review.readiness_digest)}${review.predecessor ? html` · predecessor ${safeText(review.predecessor)}` : ''}
      </p>
      ${this.renderApprovalRoles(review)}
      <label
        ><input type="checkbox" name="verified" /> I reviewed these effects and separately checked that the current live
        services and resources are ready. The server preview does not perform that operational check.</label
      >
      <button type="submit" class="ui button" ?disabled=${this.view.busy}>Record separate approval</button>`;
  }

  private renderApprovalRoles(review: Record<string, unknown>) {
    const roles = Array.isArray(review.roles) ? review.roles.map(asObject) : [];
    return html`<ul>
      ${roles.map(
        (row) => html`<li>
          ${safeText(row.role)} · source ${safeText(row.source_commit)} · setup ${safeText(row.setup_digest)} · tools
          ${safeText(row.tools)} · setup exit ${safeDetail(row.setup_exit)} · check exit ${safeDetail(row.check_exit)}
        </li>`
      )}
    </ul>`;
  }

  private renderCapacity(status: Record<string, unknown>) {
    const capacity = asObject(status.appliance || status.capacity || {});
    return html`<fieldset>
      <legend>Appliance capacity</legend>
      <form data-action="capacity" @submit=${this.submit}>
        ${this.numberInput('Concurrent runs', 'max_concurrent_runs', safeNumber(capacity.max_concurrent_runs, 1))}
        ${this.numberInput('Queued runs', 'max_queued', safeNumber(capacity.max_queued, 0))}
        ${this.submitButton('Save capacity')}
      </form>
    </fieldset>`;
  }

  private renderProjectGrants(status: Record<string, unknown>) {
    const operator = asObject(status.operator_grant || {});
    const environment = asObject(status.environment_grant || {});
    return html`<fieldset>
      <legend>Project grants</legend>
      <form data-action="operator-grant" @submit=${this.submit}>
        ${this.numberInput('Operator concurrent limit', 'operator_concurrent', safeNumber(operator.max_concurrent, 1))}
        <label
          ><input type="checkbox" name="operator_active" .checked=${operator.active === true} /> Active operator
          grant</label
        >
        ${this.submitButton('Save operator grant')}
      </form>
      <form data-action="environment-grant" @submit=${this.submit}>
        <p>The selected Project profile is also the profile that will be authorized.</p>
        <label
          ><input type="checkbox" name="environment_active" .checked=${environment.active === true} /> Permit factory
          Project setup</label
        >
        ${this.submitButton('Save Project grant')}
      </form>
    </fieldset>`;
  }

  private renderPolicy(status: Record<string, unknown>) {
    const policy = asObject(status.policy || {});
    const roles = asObject(policy.roles || {});
    const coder = asObject(roles.coder || {});
    const reviewer = asObject(roles.reviewer || {});
    const limits = asObject(policy.attempt_limits || {});
    return html`<fieldset>
      <legend>Repository policy</legend>
      ${this.renderPolicyBindings(policy)}
      <form data-action="policy" @submit=${this.submit}>
        ${this.textInput('Target branch ref', 'target_branch', safeText(policy.target_branch))}
        ${this.textInput('Required checks (comma separated)', 'required_checks', Array.isArray(policy.required_checks) ? policy.required_checks.join(', ') : '')}
        ${this.textInput('Coder harness', 'coder_harness', safeText(coder.harness || 'codex'))}
        ${this.textInput('Coder harness version', 'coder_harness_version', safeText(coder.harness_version))}
        ${this.textInput('Coder model', 'coder_model', safeText(coder.model))}
        ${this.textInput('Reviewer harness', 'reviewer_harness', safeText(reviewer.harness || 'codex'))}
        ${this.textInput('Reviewer harness version', 'reviewer_harness_version', safeText(reviewer.harness_version))}
        ${this.textInput('Reviewer model', 'reviewer_model', safeText(reviewer.model))}
        ${this.numberInput('Active minutes per attempt', 'active_minutes', safeNumber(limits.active_minutes, 120))}
        ${this.numberInput('Correction cycles', 'correction_cycles', safeNumber(limits.correction_cycles, 3))}
        ${this.numberInput('Repository concurrent limit', 'policy_concurrent', safeNumber(policy.max_concurrent, 1))}
        <label
          ><input type="checkbox" name="policy_enabled" .checked=${policy.enabled === true} /> Enable automatic factory
          dispatch</label
        >
        <label><input type="checkbox" name="policy_paused" .checked=${policy.paused === true} /> Pause dispatch</label>
        ${this.submitButton('Save repository policy')}
      </form>
    </fieldset>`;
  }

  private renderPolicyBindings(policy: Record<string, unknown>) {
    const publish = asObject(policy.publish_actor || {});
    const create = asObject(policy.create_actor || {});
    const review = asObject(policy.review_actor || {});
    const merge = asObject(policy.merge_actor || {});
    return html`<p>
      Current native bindings: publish ${safeText(publish.actor_id)} / token ${safeText(publish.token_id)}, create
      ${safeText(create.actor_id)} / token ${safeText(create.token_id)}, review ${safeText(review.actor_id)} / token
      ${safeText(review.token_id)}, merge ${safeText(merge.actor_id)} / token ${safeText(merge.token_id)}. Bindings come
      from current policy and cannot be changed as raw IDs here.
    </p>`;
  }

  private renderProviderControls() {
    const connection = this.view.selectedConnection;
    const generation = safeNumber(this.view.connections.find((row) => row.id === connection)?.generation, 0);
    const grants = this.view.grants.filter((grant) => grant.revoked !== true);
    return html`<fieldset>
      <legend>Provider allowance and sponsorship</legend>
      <label
        >Provider connection<select name="selected_connection" .value=${connection} @change=${this.changeConnection}>
          <option value="">Select a current connection</option>
          ${this.view.connections.map((row) => html`<option value=${safeText(row.id)}>${safeText(row.label || row.provider_id)} · ${safeText(row.state)}</option>`)}
        </select></label
      >
      ${this.renderBudgetForm()}
      <form data-action="sponsorship" @submit=${this.submit}>
        <label
          >Current broker grant<select name="grant_id">
            <option value="">Select a current grant</option>
            ${grants.map((grant) => html`<option value=${safeText(grant.id)}>${safeText(grant.project_id)} · ${safeText(grant.id)}</option>`)}
          </select></label
        >
        <p>Credential generation ${generation}.</p>
        <label><input type="checkbox" name="coder" /> Coder role</label>
        <label><input type="checkbox" name="reviewer" /> Reviewer role</label>
        ${this.numberInput('Allowance minutes', 'allowance_minutes', 60)}
        ${this.numberInput('Concurrent sessions', 'max_concurrent', 1)}
        <label><input type="checkbox" name="sponsorship_active" /> Active sponsorship</label>
        ${this.submitButton('Save sponsorship')}
      </form>
    </fieldset>`;
  }

  private renderBudgetForm() {
    const budget = this.view.usageBudget;
    return html`<form data-action="budget" @submit=${this.submit}>
      <p>
        Current rolling allowance:
        ${budget ? `${budget.rolling_minutes} minutes (revision ${budget.revision})` : 'not configured'}.
      </p>
      ${this.numberInput('Rolling minutes', 'rolling_minutes', safeNumber(budget?.rolling_minutes, 360))}
      ${this.submitButton('Save connection budget')}
    </form>`;
  }

  private textInput(label: string, name: string, value = '') {
    return html`<label>${label}<input name=${name} autocomplete="off" .value=${value} /></label>`;
  }

  private numberInput(label: string, name: string, value: number) {
    return html`<label>${label}<input type="number" name=${name} min="0" .value=${String(value)} /></label>`;
  }
}

if (!customElements.get('soda-factory-settings')) customElements.define('soda-factory-settings', SodaFactorySettings);
