import assert from 'node:assert/strict';
import type {Page, WebSocket} from 'playwright';
import {object} from './sodaspaces-input';
import {newManagedTerminal, prepareManagedTerminal} from './sodaspaces-controls';
import {projectId} from '../../frontend/spaces/sodaspaces-api';
import {terminalID} from '../../frontend/spaces/sodaspaces-terminal-response';
import {nativeSpacesMount, spacesAPI, spacesContent} from './sodaspaces-journey-evidence';
import type {FirstUseEvidence, MatrixFacts, MatrixSession} from './sodaspaces-journey-evidence';

/** One selected repository/actor, one Create, keyless Join and terminal. Never End or cleanup. */
export async function exerciseFirstUse(
  page: Page,
  input: {actor: string; login: string; repository: string; repositoryName: string},
  permit: (path: string, body: Record<string, unknown>, terminalName?: string) => void,
  native: {
    shell: (session: MatrixSession) => Promise<MatrixFacts>;
    inspect: (session: MatrixSession, facts: MatrixFacts) => Promise<void>;
    capture?: (name: string) => Promise<void>;
  },
  evidence: FirstUseEvidence
) {
  const origin = new URL(page.url()).origin;
  const generation = await nativeSpacesMount(page);
  const spaces = await spacesContent(page);
  let expectedName = '',
    wireFailure = false,
    creates = 0,
    attaches = 0;
  const observe = (socket: WebSocket) => {
    if (!new URL(socket.url()).pathname.endsWith('/terminal')) return;
    const url = new URL(socket.url());
    if (
      url.origin !== origin.replace('https:', 'wss:') ||
      url.pathname !== `${spacesAPI}/environments/${evidence.project}/terminal` ||
      url.search ||
      url.hash
    ) {
      wireFailure = true;
      return;
    }
    socket.on('framesent', ({payload}) => {
      try {
        const v = object(JSON.parse(String(payload)));
        if (v.action !== 'create' && v.action !== 'attach') return;
        assert(
          v.repository_id === input.repository &&
            typeof v.session_generation === 'string' &&
            v.session_generation.length > 0 &&
            terminalID(v.id) &&
            v.request_id === undefined
        );
        if (v.action === 'create') {
          assert.equal(v.session_generation, generation);
          assert(expectedName && !evidence.session && ++creates === 1 && evidence.project);
          evidence.session = {id: v.id, name: expectedName, actor: input.actor, environment: evidence.project};
        } else {
          assert(v.id === evidence.session?.id);
          attaches++;
        }
      } catch {
        wireFailure = true;
      }
    });
  };
  page.on('websocket', observe);
  try {
    evidence.stage = 'confirmed empty welcome';
    await spaces.getByRole('heading', {name: 'Create your first project'}).waitFor();
    await native.capture?.('welcome');
    await spaces.getByRole('button', {name: 'Create project', exact: true}).click();
    evidence.stage = 'native repository discovery';
    await spaces
      .getByRole('radio', {name: new RegExp(input.repositoryName.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'))})
      .check();
    await native.capture?.('picker');
    await spaces.getByRole('button', {name: 'Continue', exact: true}).click();
    await spaces.getByRole('heading', {name: 'Configure project'}).waitFor();
    evidence.stage = 'installed profile selection';
    const select = spaces.getByRole('combobox', {name: /^Project OS/});
    await select.waitFor();
    const profile = await select.inputValue();
    assert(profile);
    assert.equal(await spaces.getByRole('checkbox').count(), 0, 'This selected Off fixture has no configured network');
    await native.capture?.('configure');
    evidence.stage = 'one explicit project creation';
    permit(`${spacesAPI}/environments`, {
      repository_id: input.repository,
      profile_id: profile,
      tailnet: {enabled: false},
    });
    const created = page.waitForResponse(
      (r) => new URL(r.url()).pathname === `${spacesAPI}/environments` && r.request().method() === 'POST',
      {timeout: 260000}
    );
    await spaces.getByRole('button', {name: 'Create project', exact: true}).click();
    const reply = await created,
      value = object(await reply.json());
    if (projectId(value.id)) evidence.project = value.id;
    evidence.writes.push('Create');
    evidence.create_status = reply.status();
    assert.equal(reply.status(), 201);
    assert(evidence.project && value.provisioned === true && value.repository_id === input.repository);
    evidence.stage = 'explicit browser-only Join';
    await spaces.getByRole('button', {name: 'Join project', exact: true}).waitFor();
    await native.capture?.('join');
    permit(`${spacesAPI}/environments/${evidence.project}/join`, {ssh_keys: 'none'});
    const joined = page.waitForResponse(
      (r) =>
        new URL(r.url()).pathname === `${spacesAPI}/environments/${evidence.project}/join` &&
        r.request().method() === 'POST',
      {timeout: 260000}
    );
    await spaces.getByRole('button', {name: 'Join project', exact: true}).click();
    const joinReply = await joined;
    evidence.writes.push('Join');
    evidence.join_status = joinReply.status();
    assert.equal(joinReply.status(), 200);
    await spaces.getByRole('heading', {name: 'Open your first terminal'}).waitFor();
    await native.capture?.('first-terminal');
    evidence.stage = 'one native terminal';
    expectedName = (await prepareManagedTerminal(page, input.repositoryName, evidence.project)) || '';
    assert(expectedName);
    permit(`${spacesAPI}/environments/${evidence.project}/terminal-sessions`, {}, expectedName);
    await newManagedTerminal(page, input.repositoryName, expectedName, evidence.project);
    assert(!wireFailure && evidence.session && creates === 1);
    evidence.writes.push('Create terminal');
    evidence.stage = 'native input and original account/process observation';
    const facts = await native.shell(evidence.session);
    evidence.facts = facts;
    assert(facts.login === input.login && facts.login !== 'root' && facts.tty && facts.marker === expectedName);
    await native.inspect(evidence.session, facts);
    await native.capture?.('working');
    evidence.stage = 'reload and exact native reattachment';
    await page.reload();
    await nativeSpacesMount(page);
    await (await spacesContent(page)).locator('.soda-workspace-terminal:visible .is-connected').waitFor();
    assert(!wireFailure && creates === 1 && attaches >= 1);
    await native.inspect(evidence.session, facts);
    evidence.attached = true;
    await native.capture?.('reattached');
    evidence.stage = 'complete';
  } finally {
    page.off('websocket', observe);
  }
}
