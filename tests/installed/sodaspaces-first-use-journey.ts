import assert from 'node:assert/strict';
import type {Page, Response, WebSocket} from 'playwright';
import {object} from './sodaspaces-input';
import {newManagedTerminal, prepareManagedTerminal} from './sodaspaces-controls';
import {projectId} from '../../frontend/spaces/sodaspaces-api';
import {terminalID} from '../../frontend/spaces/sodaspaces-terminal-response';
import {mountedSpacesFacts, nativeSpacesMount, spacesContent} from './sodaspaces-journey-evidence';
import type {FirstUseEvidence, MatrixFacts, MatrixSession} from './sodaspaces-journey-evidence';

/** Real first-use flow: declared intent, saved draft, funded design exploration,
consequential review decisions, then one selected repository/actor Create,
keyless Join and terminal. Never End or cleanup. */
export interface FirstUseDecision {
  match: string;
  decision: 'accept' | 'decline' | 'edit';
  correction?: string;
}
export async function exerciseFirstUse(
  page: Page,
  input: {
    actor: string;
    login: string;
    repository: string;
    repositoryName: string;
    objective: string;
    decisions: FirstUseDecision[];
  },
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
  const {subURL, apiBase} = await mountedSpacesFacts(page);
  const spaces = await spacesContent(page);
  evidence.factory = [];
  let expectedName = '',
    wireFailure = false,
    creates = 0,
    attaches = 0;
  const observe = (socket: WebSocket) => {
    if (!new URL(socket.url()).pathname.endsWith('/terminal')) return;
    const url = new URL(socket.url());
    if (
      url.origin !== origin.replace('https:', 'wss:') ||
      url.pathname !== `${apiBase}/environments/${evidence.project}/terminal` ||
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
  const recordFactory = (response: Response) => {
    const request = response.request(),
      pathname = new URL(request.url()).pathname;
    if (pathname.includes('/api/factory/') && (request.method() !== 'GET' || pathname.endsWith('/allowance')))
      evidence.factory?.push(`${request.method()} ...${pathname.slice(-52)} -> ${response.status()}`);
  };
  page.on('websocket', observe);
  page.on('response', recordFactory);
  try {
    evidence.stage = 'intent entry';
    assert(input.objective.trim(), 'First use needs a declared finite objective');
    await spaces.getByRole('heading', {name: 'What do you want to build?'}).waitFor();
    await native.capture?.('intent');
    assert(await spaces.getByRole('button', {name: 'Explore ideas'}).isDisabled());
    await spaces.getByPlaceholder('Describe the outcome you want…').fill(input.objective);
    assert(!(await spaces.getByRole('button', {name: 'Explore ideas'}).isDisabled()));
    evidence.stage = 'saved draft';
    const entry = new URL(page.url());
    const saving = page.waitForRequest(
      (request) => request.url().endsWith('/api/factory/intents') && request.method() === 'POST',
      {timeout: 60000}
    );
    const savedReply = page.waitForResponse(
      (r) => r.url().endsWith('/api/factory/intents') && r.request().method() === 'POST',
      {timeout: 60000}
    );
    await spaces.getByRole('button', {name: 'Save draft'}).click();
    const saveRequest = await saving,
      saveBody = object(saveRequest.postDataJSON());
    permit(new URL(saveRequest.url()).pathname, saveBody);
    assert.equal(saveBody.state, 'draft');
    assert.equal(saveBody.repository, '0');
    assert.equal(saveBody.objective, input.objective);
    assert.equal(saveBody.planning_calls, 8);
    assert(Array.isArray(saveBody.ideas) && saveBody.ideas.length === 0);
    assert(Array.isArray(saveBody.tasks) && saveBody.tasks.length === 0);
    const saveResponse = await savedReply;
    assert.equal(saveResponse.status(), 201);
    const saved = object(await saveResponse.json());
    const intent = String(saved.intent ?? '');
    assert(/^d[0-9a-f]{24}$/.test(intent), 'Saved draft must report its intent');
    let revision = String(saved.id ?? '');
    assert(revision, 'Saved draft must report its revision');
    await page.waitForURL((u) => new URL(u).pathname === `${subURL}/workspace/requests/${intent}`, {
      timeout: 60000,
    });
    const routed = new URL(page.url());
    assert.equal(routed.pathname, `${subURL}/workspace/requests/${intent}`);
    assert.equal(routed.search, entry.search, 'Native query must be preserved');
    assert.equal(routed.hash, entry.hash, 'Native fragment must be preserved');
    evidence.intent = intent;
    await native.capture?.('draft');
    evidence.stage = 'funded design exploration';
    const exploreRound = async (button: string) => {
      const roundSaving = page.waitForRequest(
        (request) => request.url().endsWith('/api/factory/intents') && request.method() === 'POST',
        {timeout: 300000}
      );
      const roundSaved = page.waitForResponse(
        (r) => r.url().endsWith('/api/factory/intents') && r.request().method() === 'POST',
        {timeout: 300000}
      );
      const funding = page.waitForResponse(
        (r) => r.url().endsWith(`/api/factory/intents/${intent}/allowance`) && r.request().method() === 'GET',
        {timeout: 300000}
      );
      const consulting = page.waitForRequest(
        (request) => request.url().endsWith(`/api/factory/intents/${intent}/explore`) && request.method() === 'POST',
        {timeout: 300000}
      );
      const consulted = page.waitForResponse(
        (r) => r.url().endsWith(`/api/factory/intents/${intent}/explore`) && r.request().method() === 'POST',
        {timeout: 300000}
      );
      await spaces.getByRole('button', {name: button}).click();
      const roundSaveRequest = await roundSaving,
        roundSaveBody = object(roundSaveRequest.postDataJSON());
      permit(new URL(roundSaveRequest.url()).pathname, roundSaveBody);
      assert.equal(roundSaveBody.state, 'draft');
      assert.equal(roundSaveBody.intent, intent);
      assert.equal(roundSaveBody.predecessor, revision);
      const roundSaveResponse = await roundSaved;
      assert.equal(roundSaveResponse.status(), 201);
      const roundSavedBody = object(await roundSaveResponse.json());
      revision = String(roundSavedBody.id ?? '');
      assert(revision, 'Explore must save its draft revision');
      const fundingReply = await funding;
      if (fundingReply.status() === 200) {
        const allowance = object(await fundingReply.json());
        assert.equal(allowance.intent, intent);
        assert(typeof allowance.amount === 'number' && allowance.amount > 0 && Number.isFinite(allowance.amount));
        evidence.factory?.push(
          `allowance funded amount=${allowance.amount} unit=${String(allowance.unit)} reserved=${String(allowance.reserved)} consumed=${String(allowance.consumed)}`
        );
      } else {
        assert.equal(fundingReply.status(), 404);
        evidence.factory?.push('allowance missing before explore; flow posts its creation');
      }
      const exploreRequest = await consulting,
        exploreBody = object(exploreRequest.postDataJSON());
      permit(new URL(exploreRequest.url()).pathname, exploreBody);
      assert.equal(exploreBody.revision, revision);
      const explored = await consulted;
      evidence.explore_status = explored.status();
      assert.equal(
        explored.status(),
        201,
        'Design exploration needs a funded planning allowance and reachable design judgments'
      );
      const proposal = object(await explored.json());
      assert.equal(proposal.intent, intent);
      assert.equal(proposal.predecessor, revision);
      assert.equal(proposal.state, 'draft');
      revision = String(proposal.id ?? '');
    };
    await exploreRound('Explore ideas');
    await spaces.getByRole('heading', {name: 'Review your product direction'}).waitFor({timeout: 60000});
    evidence.stage = 'consequential review decisions';
    const applied: string[] = [];
    const decide = (text: string | null, apply: (choice: FirstUseDecision) => Promise<void>) => {
      const found = input.decisions.filter((choice) => choice.match && (text ?? '').includes(choice.match));
      assert.equal(found.length, 1, `Exactly one recorded decision must match: ${(text ?? '').slice(0, 80)}`);
      const choice = found[0];
      assert(choice && !applied.includes(choice.match), 'Each recorded decision applies once');
      applied.push(choice.match);
      return apply(choice);
    };
    const decisionPass = async () => {
      const ideas = spaces.locator('section.intent-review > article');
      assert((await ideas.count()) > 0, 'Review must propose at least one idea');
      for (let index = 0; index < (await ideas.count()); index++) {
        const article = ideas.nth(index),
          choice = article.locator('select[aria-label^="Choice: "]');
        if (!(await choice.count()) || (await choice.inputValue()) !== 'pending') continue;
        const description = await article.locator('p').first().textContent();
        await decide(description, async (selected) => {
          await choice.selectOption(selected.decision);
          if (selected.decision === 'edit') {
            const correction = selected.correction;
            assert(typeof correction === 'string' && correction.trim(), 'An edit decision needs its correction');
            await article.getByLabel('What should change?').fill(correction);
          }
        });
      }
      const challenges = spaces.locator('section[aria-label="Questions needing your answer"] article');
      for (let index = 0; index < (await challenges.count()); index++) {
        const article = challenges.nth(index);
        if (((await article.textContent()) ?? '').includes('Answered:')) continue;
        const assumption = await article.locator('p').first().textContent();
        await decide(assumption, async (selected) => {
          if (selected.decision === 'accept')
            await article.getByRole('button', {name: /^Use recommended resolution for /}).click();
          else {
            const correction = selected.correction;
            assert(
              selected.decision === 'edit' && typeof correction === 'string' && correction.trim(),
              'Answer each question explicitly'
            );
            await article.getByLabel(/^Explain a correction to /).fill(correction);
          }
        });
      }
    };
    for (let round = 0; ; round++) {
      await decisionPass();
      const update = spaces.getByRole('button', {name: 'Update proposal'});
      if (await update.isDisabled()) break;
      assert(round < 2, 'Review must settle after recorded decisions and refresh');
      await exploreRound('Update proposal');
      await page.waitForFunction(
        () =>
          [...document.querySelectorAll('#sodaspaces-data button')].find(
            (node) => node.textContent?.trim() === 'Update proposal'
          ) instanceof HTMLButtonElement &&
          (
            [...document.querySelectorAll('#sodaspaces-data button')].find(
              (node) => node.textContent?.trim() === 'Update proposal'
            ) as HTMLButtonElement
          ).disabled,
        {timeout: 60000}
      );
    }
    assert.equal(applied.length, input.decisions.length, 'Every recorded decision must apply to review');
    evidence.decisions = applied;
    await native.capture?.('review');
    const cont = spaces.getByRole('button', {name: 'Choose repository'});
    assert(await cont.isEnabled(), 'Decided review must unlock repository choice');
    await cont.click();
    evidence.stage = 'native repository discovery';
    await spaces
      .getByRole('radio', {name: new RegExp(input.repositoryName.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'))})
      .check();
    await native.capture?.('picker');
    const accepting = page.waitForRequest(
      (request) => request.url().endsWith('/api/factory/intents') && request.method() === 'POST',
      {timeout: 120000}
    );
    const accepted = page.waitForResponse(
      (r) => r.url().endsWith('/api/factory/intents') && r.request().method() === 'POST',
      {timeout: 120000}
    );
    const generating = page.waitForRequest(
      (request) => request.url().endsWith(`/api/factory/intents/${intent}/generate`) && request.method() === 'POST',
      {timeout: 120000}
    );
    const generated = page.waitForResponse(
      (r) => r.url().endsWith(`/api/factory/intents/${intent}/generate`) && r.request().method() === 'POST',
      {timeout: 300000}
    );
    await spaces.getByRole('button', {name: 'Continue', exact: true}).click();
    const acceptRequest = await accepting,
      acceptBody = object(acceptRequest.postDataJSON());
    permit(new URL(acceptRequest.url()).pathname, acceptBody);
    assert.equal(acceptBody.state, 'accepted');
    assert.equal(acceptBody.repository, input.repository);
    assert.equal(acceptBody.intent, intent);
    const acceptResponse = await accepted;
    assert.equal(acceptResponse.status(), 201);
    const generateRequest = await generating,
      generateBody = object(generateRequest.postDataJSON());
    permit(new URL(generateRequest.url()).pathname, generateBody);
    const generateResponse = await generated;
    assert.equal(generateResponse.status(), 200);
    const generation = object(await generateResponse.json());
    evidence.generation = String(generation.state ?? '');
    assert(evidence.generation, 'Native work generation must report its state');
    await spaces.getByRole('heading', {name: 'Configure project'}).waitFor();
    evidence.stage = 'installed profile selection';
    const select = spaces.getByRole('combobox', {name: /^Project OS/});
    await select.waitFor();
    const profile = await select.inputValue();
    assert(profile);
    assert.equal(await spaces.getByRole('checkbox').count(), 0, 'This selected Off fixture has no configured network');
    await native.capture?.('configure');
    evidence.stage = 'one explicit project creation';
    permit(`${apiBase}/environments`, {
      repository_id: input.repository,
      profile_id: profile,
      tailnet: {enabled: false},
    });
    const created = page.waitForResponse(
      (r) => new URL(r.url()).pathname === `${apiBase}/environments` && r.request().method() === 'POST',
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
    permit(`${apiBase}/environments/${evidence.project}/join`, {ssh_keys: 'none'});
    const joined = page.waitForResponse(
      (r) =>
        new URL(r.url()).pathname === `${apiBase}/environments/${evidence.project}/join` &&
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
    permit(`${apiBase}/environments/${evidence.project}/terminal-sessions`, {}, expectedName);
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
    page.off('response', recordFactory);
  }
}
