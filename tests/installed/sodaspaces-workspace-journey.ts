// Product-owned installed scenario. Imported by the existing guarded journey;
// local fixtures supply only native observation doubles, never substitute UI writes.
import assert from 'node:assert/strict';
import type {Page, WebSocket} from 'playwright';
import {object} from './sodaspaces-input';
import type {MatrixInput, MatrixProject} from './sodaspaces-matrix-input';
import {newManagedTerminal, terminalMenu} from './sodaspaces-controls';
import {terminalID} from '../../frontend/spaces/sodaspaces-api';
export interface MatrixSession {name: string; environment: string; id: string; request: string; attachment: string; actor: string}
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
  permit: (session: MatrixSession, action: 'hide' | 'return' | 'end') => void,
  native: MatrixNative, evidence: MatrixEvidence, otherPage: () => Promise<Page>) {
  const actor = request.actors[actorIndex]; assert(actor);
  const sessions: MatrixSession[] = [], pending: {name: string; environment: string; request?: string}[] = [];
  let protocolFailure = false;
  const receive = (ws: WebSocket) => {
    const url = new URL(ws.url());
    if (!url.pathname.endsWith('/terminal')) return;
    const environment = url.pathname.split('/')[5];
    if (!environment || url.origin !== new URL(page.url()).origin.replace('https:', 'wss:') || url.search || url.hash ||
      url.pathname !== `/-/soda/api/environments/${environment}/terminal` || !request.projects.some(project => project.environment === environment)) {protocolFailure = true; return;}
    let opening: {action: string; id?: string; request?: string} | undefined;
    ws.on('framesent', ({payload}) => {
      try {
        const frame = object(JSON.parse(String(payload)));
        if (frame.action !== 'create' && frame.action !== 'attach') return;
        assert.equal(frame.expected_user_id, actor);
        const project = request.projects.find(project => project.environment === environment); assert(project);
        assert.equal(frame.repository_id, project.repository_id);
        if (frame.action === 'create') {
          const draft = pending.find(item => item.environment === environment && item.request === undefined);
          assert(draft && terminalID(frame.request_id) && frame.id === undefined);
          draft.request = frame.request_id; opening = {action: 'create', request: frame.request_id};
        } else {assert(terminalID(frame.id) && sessions.some(session => session.id === frame.id && session.environment === environment)); opening = {action: 'attach', id: frame.id};}
      } catch {protocolFailure = true;}
    });
    ws.on('framereceived', ({payload}) => {
      try {
        const frame = object(JSON.parse(String(payload))); if (frame.type !== 'session') return;
        assert(Object.keys(frame).sort().join(',') === 'attachment_id,id,request_id,type');
        assert(opening && terminalID(frame.id) && terminalID(frame.request_id) && terminalID(frame.attachment_id));
        if (opening.action === 'create') {
          const draft = pending.find(item => item.request === opening?.request); assert(draft && opening.request === frame.request_id);
          assert(!sessions.some(session => session.id === frame.id));
          const session = {name: draft.name, environment, id: frame.id, request: frame.request_id, attachment: frame.attachment_id, actor};
          sessions.push(session); evidence.sessions.push({...session}); pending.splice(pending.indexOf(draft), 1);
        } else {
          const session = sessions.find(session => session.id === opening?.id); assert(session && session.id === frame.id && session.request === frame.request_id);
          assert.notEqual(session.attachment, frame.attachment_id); session.attachment = frame.attachment_id;
        }
        opening = undefined;
      } catch {protocolFailure = true;}
    });
  };
  page.on('websocket', receive);
  const select = async (session: MatrixSession) => {
    await page.getByRole('button', {name: 'Sessions', exact: true}).click();
    await page.getByRole('button', {name: 'All', exact: true}).click();
    await page.locator('.soda-session-list button').filter({hasText: session.name}).click();
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
      const name = `matrix-${actorIndex}-${sessions.length}-${crypto.randomUUID().slice(0, 8)}`;
      pending.push({name, environment: project.environment});
      evidence.stage = 'create ' + sessions.length;
      await newManagedTerminal(page, project.repository_path.slice(1), name, project.environment);
      assert(!protocolFailure && pending.length === 0);
      const session = sessions.at(-1); assert(session);
      evidence.stage = 'native facts ' + (sessions.length - 1);
      const facts = await native.shell(page, session, true);
      assert(facts.pid > 0 && facts.start && facts.login && facts.tty && facts.marker === session.name);
      const record = evidence.sessions.find(record => record.id === session.id); assert(record); record.facts = facts;
      await native.inspect(project, actorIndex, session, facts, false);
    }
    assert.equal(sessions.length, 6); assert.equal(new Set(sessions.map(session => session.request)).size, 6);
    evidence.stage = 'contending writer';
    const contender = await otherPage();
    try {
      contender.on('websocket', receive);
      await contender.goto(new URL('/?soda-view=spaces', page.url()).href);
      await contender.locator('#sodaspaces-data[aria-busy=false]').waitFor();
      await contender.getByRole('button', {name: 'Sessions', exact: true}).click();
      const first = sessions[0]; assert(first);
      await contender.locator('.soda-session-list button').filter({hasText: first.name}).click();
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
    evidence.stage = 'hide and return';
    const first = sessions[0]; assert(first); await select(first);
    const lifetimeResponse = () => page.waitForResponse(response => new URL(response.url()).pathname === `/-/soda/api/environments/${first.environment}/terminal-sessions/${first.id}` && response.request().method() === 'POST');
    permit(first, 'hide'); const hidden = lifetimeResponse(); await terminalMenu(page, 'Hide session'); assert.equal((await hidden).status(), 200);
    await select(first); permit(first, 'return'); const returned = lifetimeResponse(); await terminalMenu(page, 'Continue working'); assert.equal((await returned).status(), 200);
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
