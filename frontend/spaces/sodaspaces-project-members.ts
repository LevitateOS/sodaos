import {LitElement, html} from 'lit';
import type {PreparedExtensionMount} from './soda-extension.js';
import {check, id, object, projectId, readSodaJSON} from './sodaspaces-api.js';

interface Access {
  revision: number;
  held: boolean;
  requires_join: boolean;
  action: string;
  complete: boolean;
}
interface Member {
  user_id: string;
  login: string;
  access: Access;
}
interface Preview extends Member {
  installed_fingerprints: string[];
  native_unavailable: boolean;
  native:
    | {
        keys: string[];
        processes: {pid: number; command: string}[];
        login_disabled: boolean;
        keys_removed: boolean;
        processes_stopped: boolean;
      }
    | undefined;
}
function access(value: unknown): Access {
  const a = object(value);
  check(Number.isSafeInteger(a.revision) && (a.revision as number) >= 0);
  for (const key of ['held', 'requires_join', 'complete']) check(typeof a[key] === 'boolean');
  check(typeof a.action === 'string');
  return a as unknown as Access;
}
function member(value: unknown): Member {
  const m = object(value);
  check(id(m.user_id) && typeof m.login === 'string' && /^[a-z][a-z0-9_-]{0,30}$/.test(m.login));
  return {user_id: m.user_id, login: m.login, access: access(m.access)};
}

class SodaProjectMembers extends LitElement {
  static properties = {
    members: {state: true},
    preview: {state: true},
    busy: {state: true},
    message: {state: true},
    canManage: {state: true},
  };
  declare private members: Member[];
  declare private preview: Preview | undefined;
  declare private busy: boolean;
  declare private message: string;
  declare private canManage: boolean;
  private transport?: PreparedExtensionMount;
  private project = '';
  private active = false;
  private serial = 0;
  private controller?: AbortController;
  constructor() {
    super();
    this.members = [];
    this.busy = false;
    this.message = '';
    this.canManage = false;
  }
  protected createRenderRoot() {
    return this;
  }
  set context(value: {transport: PreparedExtensionMount; project: string; active: boolean}) {
    check(projectId(value.project));
    const changed = this.transport !== value.transport || this.project !== value.project;
    const opening = value.active && !this.active;
    if (changed || !value.active) {
      this.serial++;
      this.controller?.abort();
      this.busy = false;
    }
    if (changed) {
      this.members = [];
      this.preview = undefined;
      this.canManage = false;
    }
    this.transport = value.transport;
    this.project = value.project;
    this.active = value.active;
    if (this.active && (changed || opening)) void this.refresh();
  }
  disconnectedCallback() {
    this.serial++;
    this.controller?.abort();
    this.active = false;
    super.disconnectedCallback();
  }
  private async request(path: string, body?: unknown): Promise<unknown> {
    if (!this.transport || !this.active) throw Error('Open Project Access before acting.');
    this.controller = new AbortController();
    const signal = AbortSignal.any([this.controller.signal, AbortSignal.timeout(60000)]);
    const response = await this.transport.request(`environments/${this.project}/${path}`, {
      method: body === undefined ? 'GET' : 'POST',
      signal,
      ...(body === undefined ? {} : {headers: {'Content-Type': 'application/json'}, body: JSON.stringify(body)}),
    });
    const data = await readSodaJSON(response);
    if (!response.ok) {
      const error = object(object(data).error);
      throw Error(typeof error.message === 'string' ? error.message : 'Project access could not be confirmed.');
    }
    return data;
  }
  private async act(action: () => Promise<void>) {
    if (this.busy || !this.active || this.closest('[hidden], [inert]')) return;
    const serial = this.serial;
    this.busy = true;
    this.message = '';
    try {
      await action();
    } catch (error) {
      if (serial === this.serial)
        this.message =
          error instanceof Error ? error.message : 'Project access could not be confirmed. Refresh its state.';
    } finally {
      if (serial === this.serial) this.busy = false;
    }
  }
  private refresh() {
    return this.act(async () => {
      const serial = this.serial;
      const data = object(await this.request('members'));
      check(Array.isArray(data.items) && typeof data.can_manage === 'boolean');
      const members = data.items.map(member);
      if (serial !== this.serial) return;
      this.members = members;
      this.canManage = data.can_manage;
      this.preview = undefined;
    });
  }
  private review(target: Member) {
    return this.act(async () => {
      const serial = this.serial;
      const data = object(await this.request(`members/${target.user_id}/access`));
      const m = member(data);
      check(m.user_id === target.user_id && m.login === target.login && typeof data.native_unavailable === 'boolean');
      let native: Preview['native'];
      const n = object(m.access).native;
      if (n != null) {
        const value = object(n);
        check(
          Array.isArray(value.keys) && value.keys.every((k) => typeof k === 'string') && Array.isArray(value.processes)
        );
        const processes = value.processes.map((p) => {
          const item = object(p);
          check(Number.isSafeInteger(item.pid) && typeof item.command === 'string');
          return {pid: item.pid as number, command: item.command};
        });
        for (const key of ['login_disabled', 'keys_removed', 'processes_stopped'])
          check(typeof value[key] === 'boolean');
        native = {
          keys: value.keys as string[],
          processes,
          login_disabled: value.login_disabled as boolean,
          keys_removed: value.keys_removed as boolean,
          processes_stopped: value.processes_stopped as boolean,
        };
      }
      if (serial !== this.serial) return;
      const fingerprints = data.installed_fingerprints ?? [];
      check(Array.isArray(fingerprints) && fingerprints.every((f) => typeof f === 'string'));
      this.preview = {...m, native_unavailable: data.native_unavailable, native, installed_fingerprints: fingerprints};
    });
  }
  private change(action: 'revoke' | 'restore') {
    const target = this.preview;
    if (!target || !this.canManage || this.busy || !this.active) return;
    if (
      action === 'restore' &&
      (!target.access.held || (!target.access.complete && target.access.action !== 'restore'))
    )
      return;
    const question =
      action === 'revoke'
        ? `Revoke Project access for ${target.login}? This removes their Project SSH keys and ends their terminals, SSH sessions and processes, including services running as this user. Their account and files remain. Native Forgejo access is separate.`
        : `Restore Project access for ${target.login}? They must explicitly Join again with current repository write permission and select their SSH keys. Existing terminals stay ended.`;
    if (!window.confirm(question)) return;
    return this.act(async () => {
      const serial = this.serial;
      const data = object(
        await this.request(`members/${target.user_id}/access`, {
          action,
          revision: target.access.revision,
          confirm: true,
        })
      );
      const result = member(data);
      if (serial !== this.serial) return;
      this.members = this.members.map((m) => (m.user_id === result.user_id ? result : m));
      this.preview = undefined;
      this.message =
        result.access.complete && data.pending !== true
          ? action === 'revoke'
            ? 'Project access revoked. The member’s account and files are retained.'
            : 'Project access restored. The member must Join again and select keys.'
          : 'The access hold is recorded. Native effects are not confirmed. Review this member and retry the same operation to reconcile; re-entry remains blocked.';
      this.dispatchEvent(
        new CustomEvent('soda:environment-changed', {bubbles: true, detail: {environmentId: this.project}})
      );
    });
  }
  private label(a: Access) {
    if (a.held) return a.complete ? 'Revoked' : 'Revocation or restoration pending';
    if (a.requires_join) return 'Restored; Join required';
    return 'Joined';
  }
  render() {
    return html`<fieldset>
      <legend>Project members</legend>
      <p>
        Project access is separate from native Forgejo permissions. Remove repository access in Forgejo as well when
        offboarding someone.
      </p>
      <button
        type="button"
        class="ui basic button"
        ?disabled=${this.busy || !this.active}
        @click=${() => this.refresh()}
      >
        Refresh members
      </button>
      <ul>
        ${this.members.map((m) => html`<li><strong>${m.login}</strong> — ${this.label(m.access)} ${this.canManage ? html`<button type="button" class="ui basic button" ?disabled=${this.busy || !this.active} @click=${() => this.review(m)}>Review access for ${m.login}</button>` : ''}</li>`)}
      </ul>
      ${this.renderPreview()}
      <p role="status">${this.message}</p>
    </fieldset>`;
  }
  private renderPreview() {
    const p = this.preview;
    if (!p) return html``;
    return html`<section aria-label=${`Project access for ${p.login}`}>
      <h3>${p.login}</h3>
      <p>Stable member ${p.user_id} in Project ${this.project}. Account identity, home and files will be retained.</p>
      ${
        p.native_unavailable
          ? html`<p>
              Current native state is unavailable. Any recorded hold remains active. Revoking still records the hold
              before attempting native effects.
            </p>`
          : html`<p>
                ${p.native?.keys.length || 0} installed SSH keys; ${p.native?.processes.length || 0} running processes.
                Every process and service under this member’s UID is affected.
              </p>
              <ul>
                ${p.installed_fingerprints.map((fingerprint) => html`<li><code>${fingerprint}</code></li>`)}
                ${p.native?.processes.map((process) => html`<li>${process.command} (PID ${process.pid})</li>`)}
              </ul>
              <p>
                Login disabled: ${p.native?.login_disabled ? 'yes' : 'no'}; keys removed:
                ${p.native?.keys_removed ? 'yes' : 'no'}; processes stopped:
                ${p.native?.processes_stopped ? 'yes' : 'no'}.
              </p>`
      }
      <p>
        Copied credentials and native repository permissions remain separate. Restoring access requires a fresh Join
        with explicitly selected keys.
      </p>
      <button
        type="button"
        class="ui button danger"
        ?disabled=${this.busy || !this.active}
        @click=${() => this.change('revoke')}
      >
        ${p.access.held && !p.access.complete && p.access.action === 'revoke' ? 'Retry revoke Project access' : 'Revoke Project access'}</button
      >${p.access.held && p.access.complete ? html`<button type="button" class="ui button" ?disabled=${this.busy || !this.active} @click=${() => this.change('restore')}>Restore Project access</button>` : ''}${p.access.held && !p.access.complete && p.access.action === 'restore' ? html`<button type="button" class="ui button" ?disabled=${this.busy || !this.active} @click=${() => this.change('restore')}>Retry restore Project access</button>` : ''}
    </section>`;
  }
}
customElements.define('soda-project-members', SodaProjectMembers);
