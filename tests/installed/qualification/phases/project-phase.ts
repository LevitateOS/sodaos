import assert from 'node:assert/strict';
import type {Page} from 'playwright';
import {prepareManagedTerminal, newManagedTerminal} from '../../sodaspaces-controls';
import {nativeSpacesMount, mountedSpacesFacts} from '../../sodaspaces-journey-evidence';
import {observeMatrixShell, inspectNativeTerminal} from '../../sodaspaces-matrix-native';
import {check, COMMIT, digestRef} from './support';
let stage = 'not_started';
export function projectDiagnostic(error: any) {
  const name = ['Error', 'TypeError', 'AssertionError', 'TimeoutError', 'SyntaxError', 'ProtocolError'].includes(
    error?.name
  )
    ? error.name
    : 'Error';
  const match =
    typeof error?.stack === 'string'
      ? error.stack.match(
          /\/(project-phase\.ts|docker-collector\.ts|project-persistence-witness\.ts|retained-native\.ts|coreBundle\.js):(\d+):(\d+)/
        )
      : null;
  return {
    stage,
    errorName: name,
    sourceLocation: match ? {file: match[1], line: Number(match[2]), column: Number(match[3])} : null,
  };
}
export function validateProjectPhase(value: any) {
  check(value?.execution_released === true, 'project_phase_held');
  const r = value.repository;
  check(
    Number.isSafeInteger(r?.id) &&
      r.id > 0 &&
      /^[A-Za-z0-9._-]+$/.test(r.owner) &&
      /^[A-Za-z0-9._-]+$/.test(r.name) &&
      COMMIT.test(r.source_commit),
    'project_phase_repository'
  );
  check(value.profile_id === 'rocky-headless' && /^[1-9][0-9]{0,18}$/.test(value.actor), 'project_phase_profile_actor');
  return value;
}

// This phase arms only its one new TTY reservation; no Create/Join is sent.
// The immutable request is written before route.continue sends any effect.
export class ProjectPermits {
  apiBase = '';
  project = '';
  terminalName = '';
  used: string[] = [];
  private armed: {name: string; path: string; validate: (body: any) => void} | undefined;
  constructor(private save: (name: string, value: any) => Promise<any>) {}
  arm(name: string, path: string, validate: (body: any) => void) {
    check(!this.armed && !this.used.includes(name), 'project_action_already_issued');
    this.armed = {name, path, validate};
  }
  async admit(request: any) {
    const a = this.armed;
    if (!a || request.method() !== 'POST' || new URL(request.url()).pathname !== a.path) return false;
    const body = request.postDataJSON();
    a.validate(body);
    this.armed = undefined;
    this.used.push(a.name);
    await this.save(a.name + '.request.json', {method: 'POST', path: a.path, body});
    return true;
  }
  allowsSocket(url: string, origin: string) {
    const u = new URL(url);
    return (
      this.project !== '' &&
      u.origin === origin.replace('https:', 'wss:') &&
      u.pathname === this.apiBase + '/environments/' + this.project + '/terminal' &&
      !u.search &&
      !u.hash
    );
  }
}

// Upstream owner API Create/Join and role preparation are prerequisites.
// This protected browser phase owns only a NEW real TTY and its member data.
export async function exerciseProjectPhase(
  page: Page,
  cfg: any,
  binding: any,
  permits: ProjectPermits,
  save: (name: string, value: any) => Promise<any>,
  inspectProject: (project: string, repository: any) => any,
  inspectShell: (project: string, login: string, command: string) => string,
  screenshot: (page: Page) => Promise<void>
) {
  const phase = validateProjectPhase(cfg.project_phase),
    r = phase.repository,
    origin = cfg.guest.origin;
  const factory = cfg.factory_phase?.repository;
  check(
    factory &&
      factory.id === r.id &&
      factory.owner === r.owner &&
      factory.name === r.name &&
      factory.source_commit === r.source_commit &&
      /^p[a-f0-9]{24}$/.test(factory.project),
    'same_actual_prepared_factory_project'
  );
  stage = 'navigate_workspace';
  const workspace = origin + '/workspace?repository=' + r.id;
  if (page.url() !== workspace) await page.goto(workspace);
  stage = 'mount_workspace';
  const generation = await nativeSpacesMount(page),
    mount = await mountedSpacesFacts(page);
  permits.apiBase = mount.apiBase;
  const get = async (path: string) =>
    page.evaluate(async (path) => {
      const response = await (document.querySelector('soda-spaces') as any).binding.transport.request(path, {
        method: 'GET',
        signal: AbortSignal.timeout(20000),
      });
      if (response.status !== 200) throw Error('project_read_unavailable');
      return response.json();
    }, path);
  const collection = await get('environments?repository_id=' + r.id),
    profiles = await get('repositories/' + r.id + '/profiles');
  check(
    collection.items?.length === 1 && collection.items[0].id === factory.project,
    'one_actual_prepared_factory_project'
  );
  check(Array.isArray(profiles.items), 'installed_profiles_unavailable');
  const selected = profiles.items.filter((p: any) => p.id === phase.profile_id);
  check(
    selected.length === 1 &&
      selected[0].image === binding.payload.Images['project-os'].Config &&
      selected[0].revision === binding.payload.Revision &&
      selected[0].architecture === 'amd64',
    'candidate_prepared_project_profile'
  );
  const environment = collection.items[0],
    detail = await get('environments/' + environment.id),
    spaces = await get('spaces');
  check(
    environment.repository_id === String(r.id) &&
      environment.provisioned === true &&
      environment.profile?.id === phase.profile_id &&
      environment.profile.image === selected[0].image &&
      environment.profile.revision === selected[0].revision,
    'retained_prepared_profile_binding'
  );
  check(
    spaces.actor?.id === phase.actor &&
      detail.environment?.id === environment.id &&
      detail.environment.repository_id === String(r.id) &&
      /^[a-z][a-z0-9_-]{0,30}$/.test(detail.login) &&
      detail.login !== 'root' &&
      detail.execution_allowed === true &&
      detail.observed?.running === true &&
      detail.terminal_support?.state === 'ready',
    'current_joined_owner_terminal_readiness'
  );
  const visible = spaces.items?.filter((x: any) => x.environment?.id === environment.id);
  check(
    visible?.length === 1 &&
      visible[0].login === detail.login &&
      Array.isArray(visible[0].terminals) &&
      visible[0].terminals.length === 0,
    'no_existing_owner_terminal'
  );
  const join = {login: detail.login, submitted: false, existing_membership: true};
  permits.project = environment.id;
  const native = inspectProject(environment.id, r);
  check(
    native.project === environment.id &&
      native.image === digestRef(selected[0].image) &&
      native.source === r.source_commit,
    'prepared_native_image_source'
  );
  await save('prepared-project.json', {
    environment,
    member: {actor: phase.actor, login: join.login},
    createRequested: false,
    joinRequested: false,
    attribution:
      'Project Create and Join are upstream normal owner API prerequisites; this phase does not claim those browser effects.',
  });
  await save('native-project.json', native);

  const wire = observeMatrixShell(page);
  let sessionId = '',
    creates = 0,
    attaches = 0,
    wireFailure = false,
    lastAttachGeneration = '';
  const sockets = (socket: any) => {
    if (!permits.allowsSocket(socket.url(), origin)) {
      wireFailure = true;
      return;
    }
    socket.on('framesent', ({payload}: any) => {
      try {
        const frame = JSON.parse(String(payload));
        if (frame.action !== 'create' && frame.action !== 'attach') return;
        check(
          frame.repository_id === String(r.id) &&
            typeof frame.session_generation === 'string' &&
            frame.session_generation !== '' &&
            /^[a-f0-9]{32}$/.test(frame.id) &&
            frame.request_id === undefined,
          'owned_terminal_handshake'
        );
        if (frame.action === 'create') {
          check(
            frame.session_generation === generation && ++creates === 1 && frame.name === permits.terminalName,
            'one_original_terminal'
          );
          sessionId = frame.id;
        } else {
          check(frame.id === sessionId, 'same_terminal_reattachment');
          lastAttachGeneration = frame.session_generation;
          attaches++;
        }
      } catch {
        wireFailure = true;
      }
    });
  };
  page.on('websocket', sockets);
  const selectExisting = async () => {
    const owner = page.locator('soda-spaces'),
      project = owner.locator('.soda-project-group[data-environment-id="' + environment.id + '"] .soda-project-select');
    const directory = owner.getByRole('button', {name: 'Projects', exact: true});
    stage = 'reload_await_project_directory';
    await project.or(directory).filter({visible: true}).first().waitFor();
    stage = 'reload_disclose_projects';
    if (!(await project.isVisible())) await directory.click();
    stage = 'reload_select_project';
    await project.waitFor();
    check((await project.getAttribute('aria-label')) === r.owner + '/' + r.name, 'exact_visible_project');
    await project.click();
    stage = 'reload_select_terminals';
    await owner.getByRole('button', {name: 'Terminals', exact: true}).click();
    stage = 'reload_disclose_workspace_options';
    await owner.getByLabel('Workspace options', {exact: true}).click();
    stage = 'reload_select_exact_session';
    const row = owner.locator('button[data-terminal-id="' + sessionId + '"]:visible');
    await row.waitFor();
    check((await row.count()) === 1, 'one_owned_session_row');
    await row.click();
    stage = 'reload_wait_connected';
    await owner.locator('.soda-workspace-terminal:visible .is-connected').waitFor();
  };
  try {
    stage = 'prepare_one_browser_terminal';
    const name = await prepareManagedTerminal(page, r.owner + '/' + r.name, environment.id);
    check(name && /^Terminal [1-9][0-9]*$/.test(name), 'normal_default_terminal_name');
    permits.terminalName = name;
    permits.arm('terminal', mount.apiBase + '/environments/' + environment.id + '/terminal-sessions', (body) => {
      assert.deepEqual(Object.keys(body).sort(), ['cols', 'name', 'rows']);
      check(
        body.name === name &&
          Number.isSafeInteger(body.cols) &&
          body.cols >= 2 &&
          body.cols <= 500 &&
          Number.isSafeInteger(body.rows) &&
          body.rows >= 2 &&
          body.rows <= 300,
        'owned_terminal_reservation'
      );
    });
    // Save the normal reservation reply as it arrives, before the UI connected wait.
    const reserveReply = page
      .waitForResponse(
        (response) =>
          new URL(response.url()).pathname ===
            mount.apiBase + '/environments/' + environment.id + '/terminal-sessions' &&
          response.request().method() === 'POST',
        {timeout: 60000}
      )
      .then(async (reservation) => {
        const reserved = await reservation.json();
        await save('terminal.receipt.json', {status: reservation.status(), ...reserved});
        return {status: reservation.status(), reserved};
      });
    void reserveReply.catch(() => {});
    stage = 'create_one_browser_terminal';
    await newManagedTerminal(page, r.owner + '/' + r.name, name, environment.id);
    stage = 'read_reserved_terminal';
    const {status, reserved} = await reserveReply;
    check(
      status === 201 && reserved.id === sessionId && creates === 1 && !wireFailure,
      'owned_native_terminal_created'
    );
    const session = {id: sessionId, name, environment: environment.id, actor: phase.actor};
    stage = 'read_initial_shell';
    const facts = await wire.shell(session, true);
    check(facts.login === join.login && facts.tty && facts.marker === name, 'real_browser_tty_account');
    await inspectNativeTerminal(session, facts, false, (command) => inspectShell(environment.id, join.login, command));
    await screenshot(page);
    stage = 'reload_workspace';
    await page.reload();
    stage = 'remount_workspace';
    const reloadedGeneration = await nativeSpacesMount(page);
    await selectExisting();
    check(
      creates === 1 && attaches >= 1 && lastAttachGeneration === reloadedGeneration && !wireFailure,
      'same_terminal_after_reload'
    );
    assert.deepEqual(await wire.shell(session, false), facts);
    await inspectNativeTerminal(session, facts, false, (command) => inspectShell(environment.id, join.login, command));
    const after = inspectProject(environment.id, r);
    assert.deepEqual(after, native);
    stage = 'save_result';
    await save('result.json', {
      format: 1,
      observed: true,
      repository: {id: r.id, owner: r.owner, name: r.name, source_commit: r.source_commit},
      environment,
      join,
      session,
      facts,
      creates,
      attaches,
      normalUIActions: permits.used,
      createRequested: false,
      joinRequested: false,
      preparedFactoryProjectReused: true,
      reloadedSameNativeProcess: true,
      terminalRetained: true,
      noFactoryModelActions: true,
      noEndOrStopOrCleanup: true,
      productionQualification: false,
    });
    return {project: environment.id, terminal: session.id, phaseObserved: true};
  } catch (error) {
    await save('diagnostic.json', projectDiagnostic(error));
    throw error;
  } finally {
    page.off('websocket', sockets);
    wire.dispose();
  }
}
