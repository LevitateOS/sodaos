// Product-owned installed scenario. Imported by the existing guarded journey;
// local fixtures supply only native observation doubles, never substitute UI writes.
import assert from 'node:assert/strict';
import type {Page, WebSocket} from 'playwright';
import {object} from './sodaspaces-input';
import type {MatrixInput, MatrixProject} from './sodaspaces-matrix-input';
import {newManagedTerminal, prepareManagedTerminal, terminalMenu} from './sodaspaces-controls';
import {terminalID, projectId} from '../../frontend/spaces/sodaspaces-api';
export interface FirstUseEvidence {
  stage: string; project?: string; session?: MatrixSession; facts?: MatrixFacts;
  writes: string[]; create_status?: number; join_status?: number; attached?: boolean;
}
/** One selected repository/actor, one Create, keyless Join and terminal. Never End or cleanup. */
export async function exerciseFirstUse(page: Page, input: {actor: string; login: string; repository: string; repositoryName: string},
  permit: (path: string, body: Record<string, unknown>, terminalName?: string) => void,
  native: {shell: (session: MatrixSession) => Promise<MatrixFacts>; inspect: (session: MatrixSession, facts: MatrixFacts) => Promise<void>; capture?: (name: string) => Promise<void>},
  evidence: FirstUseEvidence) {
  const origin = new URL(page.url()).origin;
  let expectedName = '', wireFailure = false, creates = 0, attaches = 0;
  const observe = (socket: WebSocket) => {
    if (!new URL(socket.url()).pathname.endsWith('/terminal')) return;
    const url = new URL(socket.url());
    if (url.origin !== origin.replace('https:', 'wss:') || url.pathname !== `/-/soda/api/environments/${evidence.project}/terminal` || url.search || url.hash) {wireFailure = true; return;}
    socket.on('framesent', ({payload}) => {
      try {
        const v = object(JSON.parse(String(payload))); if (v.action !== 'create' && v.action !== 'attach') return;
        assert(v.expected_user_id === input.actor && v.repository_id === input.repository && terminalID(v.id) && v.request_id === undefined);
        if (v.action === 'create') {
          assert(expectedName && !evidence.session && ++creates === 1 && evidence.project);
          evidence.session = {id: v.id, name: expectedName, actor: input.actor, environment: evidence.project};
        } else {assert(v.id === evidence.session?.id); attaches++;}
      } catch {wireFailure = true;}
    });
  };
  page.on('websocket', observe);
  try {
    evidence.stage = 'confirmed empty welcome';
    await page.getByRole('heading', {name: 'Create your first project'}).waitFor();
    assert.equal(await page.locator('#soda-native-content').getAttribute('data-actor'), input.actor);
    await native.capture?.('welcome');
    await page.getByRole('button', {name: 'Create project', exact: true}).click();
    evidence.stage = 'native repository discovery';
    await page.getByRole('radio', {name: new RegExp(input.repositoryName.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'))}).check();
    await native.capture?.('picker');
    await page.getByRole('button', {name: 'Continue', exact: true}).click();
    await page.getByRole('heading', {name: 'Configure project'}).waitFor();
    evidence.stage = 'installed profile selection';
    const select = page.getByRole('combobox', {name: /^Project OS/}); await select.waitFor();
    const profile = await select.inputValue(); assert(profile);
    assert.equal(await page.getByRole('checkbox').count(), 0, 'This selected Off fixture has no configured network');
    await native.capture?.('configure');
    evidence.stage = 'one explicit project creation';
    permit('/-/soda/api/environments', {repository_id: input.repository, profile_id: profile, tailnet: {enabled: false}});
    const created = page.waitForResponse(r => new URL(r.url()).pathname === '/-/soda/api/environments' && r.request().method() === 'POST', {timeout: 260000});
    await page.getByRole('button', {name: 'Create project', exact: true}).click();
    const reply = await created, value = object(await reply.json());
    if (projectId(value.id)) evidence.project = value.id;
    evidence.writes.push('Create'); evidence.create_status = reply.status();
    assert.equal(reply.status(), 201); assert(evidence.project && value.provisioned === true && value.repository_id === input.repository);
    evidence.stage = 'explicit browser-only Join';
    await page.getByRole('button', {name: 'Join project', exact: true}).waitFor();
    await native.capture?.('join');
    permit(`/-/soda/api/environments/${evidence.project}/join`, {ssh_keys: 'none'});
    const joined = page.waitForResponse(r => new URL(r.url()).pathname === `/-/soda/api/environments/${evidence.project}/join` && r.request().method() === 'POST', {timeout: 260000});
    await page.getByRole('button', {name: 'Join project', exact: true}).click();
    const joinReply = await joined; evidence.writes.push('Join'); evidence.join_status = joinReply.status();
    assert.equal(joinReply.status(), 200);
    await page.getByRole('heading', {name: 'Open your first terminal'}).waitFor();
    await native.capture?.('first-terminal');
    evidence.stage = 'one native terminal';
    expectedName = await prepareManagedTerminal(page, input.repositoryName, evidence.project) || ''; assert(expectedName);
    permit(`/-/soda/api/environments/${evidence.project}/terminal-sessions`, {}, expectedName);
    await newManagedTerminal(page, input.repositoryName, expectedName, evidence.project);
    assert(!wireFailure && evidence.session && creates === 1); evidence.writes.push('Create terminal');
    evidence.stage = 'native input and original account/process observation';
    const facts = await native.shell(evidence.session); evidence.facts = facts;
    assert(facts.login === input.login && facts.login !== 'root' && facts.tty && facts.marker === expectedName);
    await native.inspect(evidence.session, facts);
    await native.capture?.('working');
    evidence.stage = 'reload and exact native reattachment';
    await page.reload();
    await page.locator('.soda-workspace-terminal:visible .is-connected').waitFor();
    assert(!wireFailure && creates === 1 && attaches >= 1);
    await native.inspect(evidence.session, facts); evidence.attached = true;
    await native.capture?.('reattached');
    evidence.stage = 'complete';
  } finally {page.off('websocket', observe);}
}

export interface MatrixSession {name: string; environment: string; id: string; actor: string}
export interface MatrixFacts {pid: number; start: string; login: string; marker: string; tty: boolean; term: string}
export interface MatrixEvidence {
  stage?: string;
  sessions: Array<MatrixSession & {facts?: MatrixFacts; end_requested?: boolean; end_http?: 'accepted' | 'transport-unconfirmed'; native_cleanup?: boolean}>;
  same_document?: boolean; exact_reload?: boolean; cli?: unknown;
}
export interface MatrixNative {
  shell(page: Page, session: MatrixSession, initialize: boolean): Promise<MatrixFacts>;
  inspect(project: MatrixProject, actor: number, session: MatrixSession, facts: MatrixFacts, ended: boolean): Promise<void>;
  cli?(page: Page, sessions: readonly (MatrixSession & {facts?: MatrixFacts})[]): Promise<unknown>;
}
export async function exerciseWorkspaceMatrix(page: Page, request: MatrixInput, actorIndex: number,
  permit: (target: Omit<MatrixSession, 'id'> & {id?: string}, action: 'reserve' | 'end') => void,
  native: MatrixNative, evidence: MatrixEvidence, otherPage: () => Promise<Page>) {
  const actor = request.actors[actorIndex]; assert(actor);
  const sessions: MatrixSession[] = [], pending: {name: string; environment: string; id?: string}[] = [];
  let protocolFailure = false;
  const receive = (ws: WebSocket) => {
    const url = new URL(ws.url());
    if (!url.pathname.endsWith('/terminal')) return;
    const environment = url.pathname.split('/')[5];
    if (!environment || url.origin !== new URL(page.url()).origin.replace('https:', 'wss:') || url.search || url.hash ||
      url.pathname !== `/-/soda/api/environments/${environment}/terminal` || !request.projects.some(project => project.environment === environment)) {protocolFailure = true; return;}
    let opening: {action: string; id: string} | undefined;
    ws.on('framesent', ({payload}) => {
      try {
        const frame = object(JSON.parse(String(payload)));
        if (frame.action !== 'create' && frame.action !== 'attach') return;
        assert.equal(frame.expected_user_id, actor);
        const project = request.projects.find(project => project.environment === environment); assert(project);
        assert.equal(frame.repository_id, project.repository_id);
        if (frame.action === 'create') {
          const draft = pending.find(item => item.environment === environment && item.id === undefined);
          assert(draft && terminalID(frame.id) && frame.request_id === undefined && !sessions.some(session => session.id === frame.id));
          draft.id = frame.id; opening = {action: 'create', id: frame.id};
          // Record the issued locator before any native acknowledgement.
          const session = {name: draft.name, environment, id: frame.id, actor}; sessions.push(session); evidence.sessions.push({...session});
        } else {assert(terminalID(frame.id) && sessions.some(session => session.id === frame.id && session.environment === environment)); opening = {action: 'attach', id: frame.id};}
      } catch {protocolFailure = true;}
    });
    ws.on('framereceived', ({payload}) => {
      try {
        const frame = object(JSON.parse(String(payload))); if (frame.type !== 'ready') return;
        assert(Object.keys(frame).join(',') === 'type' && opening && sessions.some(session => session.id === opening?.id));
        if (opening.action === 'create') {
          const draft = pending.find(item => item.id === opening?.id); assert(draft); pending.splice(pending.indexOf(draft), 1);
        }
        opening = undefined;
      } catch {protocolFailure = true;}
    });
  };
  page.on('websocket', receive);
  const select = async (session: MatrixSession) => {
    await page.getByRole('button', {name: 'Projects', exact: true}).click();
    const all = page.getByRole('button', {name: 'All', exact: true}); if (await all.isVisible()) await all.click();
    await page.locator(`.soda-session-list button[data-terminal-id="${session.id}"]`).click();
    await page.locator('.soda-workspace-terminal:visible .is-connected').waitFor();
    assert(!protocolFailure);
  };
  try {
    evidence.stage = 'full-page workspace';
    await page.bringToFront();
    await page.goto(new URL('/?soda-view=spaces', page.url()).href);
    await page.locator('#sodaspaces-data[aria-busy=false]').waitFor();
    assert.equal(await page.locator('#soda-native-content').getAttribute('data-actor'), actor);
    await page.setViewportSize({width: 1920, height: 1200});
    for (const project of request.projects) for (let number = 0; number < 3; number++) {
      const name = await prepareManagedTerminal(page, project.repository_path.slice(1), project.environment); assert(name);
      pending.push({name, environment: project.environment});
      evidence.stage = 'create ' + sessions.length;
      permit({name, environment: project.environment, actor}, 'reserve');
      await newManagedTerminal(page, project.repository_path.slice(1), name, project.environment);
      assert(!protocolFailure && pending.length === 0);
      const session = sessions.at(-1); assert(session);
      evidence.stage = 'native facts ' + (sessions.length - 1);
      const facts = await native.shell(page, session, true);
      assert(facts.pid > 0 && facts.start && facts.login && facts.tty && facts.marker === session.name);
      const record = evidence.sessions.find(record => record.id === session.id); assert(record); record.facts = facts;
      await native.inspect(project, actorIndex, session, facts, false);
    }
    assert.equal(sessions.length, 6); assert.equal(new Set(sessions.map(session => session.id)).size, 6);
    evidence.stage = 'contending writer';
    const contender = await otherPage();
    try {
      contender.on('websocket', receive);
      await contender.goto(new URL('/?soda-view=spaces', page.url()).href);
      await contender.locator('#sodaspaces-data[aria-busy=false]').waitFor();
      await contender.getByRole('button', {name: 'Projects', exact: true}).click();
      const first = sessions[0]; assert(first);
      await contender.locator(`.soda-session-list button[data-terminal-id="${first.id}"]`).click();
      await contender.getByText('An existing writer is attached.', {exact: false}).waitFor();
      assert.equal(await contender.locator('.is-connected').count(), 0); assert(!protocolFailure);
    } finally {contender.off('websocket', receive); await contender.close();}
    // Closing the contender can select the read-only journey's other native tab,
    // not this page. Use a real foreground transition, never focus emulation.
    await page.bringToFront();
    evidence.stage = 'same-document layout';
    const hosts = await page.locator('.soda-workspace-terminal').elementHandles(), screens = await page.locator('.xterm').elementHandles();
    assert.equal(hosts.length, 6); assert.equal(screens.length, 6);
    evidence.stage = 'split right';
    await page.getByLabel('Pane actions', {exact: true}).click(); await page.getByRole('button', {name: 'Split right', exact: true}).click();
    evidence.stage = 'move to pane';
    await page.getByLabel('Move terminal to pane', {exact: true}).click(); await page.getByRole('button', {name: 'Pane 2', exact: true}).click();
    evidence.stage = 'compact then wide';
    await page.setViewportSize({width: 390, height: 800}); await page.setViewportSize({width: 1920, height: 1200});
    evidence.stage = 'consolidate panes';
    await page.getByLabel('Pane actions', {exact: true}).first().click(); await page.getByRole('button', {name: 'Consolidate panes', exact: true}).click();
    for (const handle of [...hosts, ...screens]) assert(await handle.evaluate(node => node.isConnected));
    evidence.same_document = true;
    evidence.stage = 'hide and show without native actions';
    const first = sessions[0]; assert(first); await select(first);
    await terminalMenu(page, 'Hide terminal'); await select(first);
    // Document replacement is attachment loss, not End or six new shells.
    evidence.stage = 'exact reload';
    await page.reload(); await page.locator('#sodaspaces-data[aria-busy=false]').waitFor();
    for (const session of sessions) {
      await select(session);
      const record = evidence.sessions.find(record => record.id === session.id); assert(record?.facts);
      assert.deepEqual(await native.shell(page, session, false), record.facts);
    }
    assert.equal(sessions.length, 6); assert(!protocolFailure); evidence.exact_reload = true;
    evidence.stage = 'declared CLI observations';
    if (request.cli.length) {assert(native.cli); evidence.cli = await native.cli(page, evidence.sessions);}
    else evidence.cli = {outcome: 'not-run', reason: 'No declared CLI/provider scope'};
    for (const session of sessions) {
      evidence.stage = 'end ' + sessions.indexOf(session);
      await select(session); await terminalMenu(page, 'End terminal…');
      assert.equal(await page.evaluate(() => document.activeElement?.textContent), 'Cancel');
      permit(session, 'end');
      const route = `/-/soda/api/environments/${session.environment}/terminal-sessions/${session.id}`;
      const outcome = Promise.race([
        page.waitForResponse(response => new URL(response.url()).pathname === route && response.request().method() === 'POST').then(response => {assert.equal(response.status(), 200); return 'accepted' as const;}),
        page.waitForEvent('requestfailed', {predicate: request => new URL(request.url()).pathname === route && request.method() === 'POST'}).then(() => 'transport-unconfirmed' as const),
      ]);
      await page.getByRole('dialog', {name: 'End terminal confirmation', exact: true}).getByRole('button', {name: 'End terminal', exact: true}).click();
      const observed = await outcome;
      const record = evidence.sessions.find(record => record.id === session.id), project = request.projects.find(project => project.environment === session.environment);
      assert(record?.facts && project); record.end_requested = true; record.end_http = observed;
      // Independent exact PID/start, unit, cgroup and owned-record observation.
      // Never inferred from HTTP/socket closure or an absent API response.
      await native.inspect(project, actorIndex, session, record.facts, true); record.native_cleanup = true;
      for (const sibling of evidence.sessions.filter(record => !record.native_cleanup)) {
        const own = request.projects.find(project => project.environment === sibling.environment); assert(own && sibling.facts);
        await native.inspect(own, actorIndex, sibling, sibling.facts, false);
      }
    }
    assert(!protocolFailure); evidence.stage = 'complete';
  } finally {page.off('websocket', receive);}
}
