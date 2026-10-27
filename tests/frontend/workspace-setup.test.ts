import test from 'node:test';
import assert from 'node:assert/strict';
import type {Page} from 'playwright';
import {
  setupWorkspaceDriver,
  chooseFirstRepository,
  reviewFirstIntent,
  completeBoundIntentPlanning,
  fixture,
  textContents,
} from './fixtures/workspace-driver';
import {captureSpacesComponent} from '../../scripts/screenshot';
setupWorkspaceDriver();

async function resumeSavedPlan(page: Page, action = 'View progress', objective = 'Build a small project tracker') {
  await page.getByRole('button', {name: 'Saved plans', exact: true}).click();
  await page.getByRole('button', {name: action + ': ' + objective, exact: true}).click();
}

async function projectPressed(page: Page) {
  return page.evaluate(() => {
    const el = document.querySelector('soda-spaces');
    const scope = el?.shadowRoot ?? el;
    if (!scope) throw Error('Missing workspace mount');
    return [...scope.querySelectorAll('.soda-project-select')]
      .map((b) => `${b.getAttribute('aria-label')}=${b.getAttribute('aria-pressed')}`)
      .join(',');
  });
}

async function focusedReviewHeading(page: Page) {
  await page.waitForFunction(() => {
    const root = document.querySelector('soda-intent')?.shadowRoot;
    const target = root?.querySelector('section.intent-review h2');
    return !!target && root?.activeElement === target;
  });
  return page.evaluate(
    () =>
      document.querySelector('soda-intent')?.shadowRoot?.querySelector('section.intent-review h2')?.textContent || ''
  );
}

async function installAuthoredPlanningPeer(page: Page) {
  await page.evaluate(() => {
    const original = window.fetch.bind(window);
    let reviewed: Record<string, unknown> | undefined;
    let content: Record<string, unknown> | undefined;
    const request = async (input: RequestInfo | URL, init?: RequestInit) => {
      const path = String(input),
        method = init?.method || 'GET';
      if (method === 'POST' && path.endsWith('/plan/review') && content) {
        const alternatives = content.alternatives as Record<string, unknown>[];
        const selected = alternatives[0]!;
        reviewed = {
          content,
          challenges: [],
          settled: [],
          recommendation: {
            producer: content.producer,
            intent: content.intent,
            revision: content.revision,
            objective: content.objective,
            recommended: selected.key,
            alternative: selected,
            outcomes: selected.outcomes,
            constraints: [],
            milestones: [{name: 'Track a project', result: 'The team sees its next action'}],
            advice: 'a'.repeat(64),
          },
          consistency: {
            producer: 'soda-authored-v1',
            intent: content.intent,
            revision: content.revision,
            mismatches: [],
            confidence: 0.9,
            advice: 'b'.repeat(64),
          },
        };
        window.workspaceFixture.calls.push({path, method, body: JSON.parse(String(init?.body))});
        return Response.json(reviewed);
      }
      const response = await original(input, init);
      if (method !== 'GET' || !path.endsWith('/plan') || !response.ok) return response;
      const status = await response.json();
      if (status.state !== 'completed') return Response.json(status);
      const {review: _modelReview, ...unreviewed} = status as Record<string, unknown>;
      void _modelReview;
      const revision = (
        document.querySelector('soda-intent') as unknown as {revision: {intent: string; objective: string}}
      ).revision;
      if (content?.revision !== status.revision) reviewed = undefined;
      content = {
        producer: 'soda-planning-agent-v1',
        supported: true,
        intent: revision.intent,
        revision: status.revision,
        objective: revision.objective,
        facts: [revision.objective],
        unknowns: ['The team can refine labels after trying the first version.'],
        alternatives: ['team_list', 'guided_steps'].map((key) => ({
          key,
          name: key === 'team_list' ? 'Team project list' : 'Guided project creation',
          audience: 'Project owners and their team',
          summary: 'Track the next action on a project.',
          steps: [
            {
              actor: 'The project owner',
              action: 'adds a project and its next action',
              result: 'The team can see who will do what next',
            },
          ],
          visibility: 'The project team sees its work',
          consequences: 'Updates remain visible in the project',
          evidence: 'The saved project tracker request',
          outcomes: [revision.objective],
        })),
      };
      return Response.json({...unreviewed, walkthroughs: content, ...(reviewed ? {review: reviewed} : {})});
    };
    Object.defineProperty(window, 'fetch', {value: request, configurable: true});
  });
}

test('setup: pending creation keeps its selection and disables navigation without duplicate writes', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await page.evaluate(() => {
    const original = window.fetch;
    Object.defineProperty(window, 'fetch', {
      configurable: true,
      value: async (input: RequestInfo | URL, init?: RequestInit) => {
        if (String(input).endsWith('/api/environments') && init?.method === 'POST')
          await new Promise<void>((resolve) =>
            window.addEventListener('release-creation', () => resolve(), {once: true})
          );
        return original(input, init);
      },
    });
    return window.workspaceFixture.api.refresh();
  });
  await chooseFirstRepository(page);
  const outcomeRegion = page.locator('.soda-configure-outcome');
  assert.equal(await outcomeRegion.count(), 1);
  assert.equal(await outcomeRegion.getAttribute('role'), 'status');
  assert.equal(await outcomeRegion.locator('.soda-feedback').count(), 0);
  const idleOutcome = await outcomeRegion.evaluate((node) => ({
    height: node.clientHeight,
    border: getComputedStyle(node).borderTopWidth,
  }));
  assert.equal(idleOutcome.height, 0);
  assert.equal(idleOutcome.border, '0px');
  const outcomeHandle = await outcomeRegion.elementHandle();
  assert.ok(outcomeHandle);
  await page.getByRole('button', {name: 'Create project', exact: true}).click();
  await page.getByRole('button', {name: 'Creating project…', exact: true}).waitFor();
  assert.equal(
    await page.evaluate((element) => document.querySelector('.soda-configure-outcome') === element, outcomeHandle),
    true
  );
  assert.equal(await outcomeRegion.locator('.soda-feedback').count(), 1);
  assert(await outcomeRegion.locator('.soda-feedback').isVisible());
  assert.equal(await outcomeRegion.locator('.soda-feedback').getAttribute('role'), null);
  assert.match(await outcomeRegion.innerText(), /Creating your project/);
  const conversation = await page.evaluate(() =>
    fetch('/assets/sodaspaces-factory-conversation.css').then(async (r) => ({
      status: r.status,
      type: r.headers.get('content-type'),
      body: await r.text(),
    }))
  );
  assert.equal(conversation.status, 200);
  assert.match(conversation.type ?? '', /text\/css/);
  assert.match(conversation.body, /\.soda-feed-filters/);
  const icon = await page.evaluate(() =>
    fetch('/assets/soda/forgejo/icons/repo-19.14.0.svg').then(async (r) => ({
      status: r.status,
      type: r.headers.get('content-type'),
      body: await r.text(),
    }))
  );
  assert.equal(icon.status, 200);
  assert.match(icon.type ?? '', /svg/);
  assert.match(icon.body, /<svg/);
  for (const name of ['Back', 'Change repository', 'Cancel setup', 'Creating project…'])
    assert(await page.getByRole('button', {name, exact: true}).isDisabled(), name);
  assert.equal(await page.locator('.soda-config-repository-name').innerText(), 'alice/Alpha\nForgejo');
  await captureSpacesComponent(page, 'pending-creation');
  await page.evaluate(() => window.dispatchEvent(new Event('release-creation')));
  await page.getByRole('heading', {name: 'Review your product direction', exact: true}).waitFor();
  await page.getByRole('button', {name: 'Accept and build', exact: true}).waitFor();
  assert.equal(
    await page.evaluate(
      () => window.workspaceFixture.calls.filter((c) => c.method !== 'GET' && !c.path.includes('/api/factory/')).length
    ),
    1
  );
});
for (const outcome of ['uncertain', 'incomplete', 'rejected'] as const)
  test(`first use: ${outcome} creation is inspected, not replayed`, async (t) => {
    const page = await fixture(t, 'page', undefined, true);
    await page.evaluate((outcome) => {
      window.workspaceFixture.setCreateOutcome(outcome);
      return window.workspaceFixture.api.refresh();
    }, outcome);
    await chooseFirstRepository(page);
    await page.getByRole('button', {name: 'Create project', exact: true}).click();
    if (outcome === 'uncertain')
      await page.getByRole('heading', {name: 'Review your product direction', exact: true}).waitFor();
    else {
      await page
        .getByText(
          outcome === 'rejected'
            ? /Installed Project OS unavailable\. No reservation/
            : /Project creation could not be confirmed/
        )
        .waitFor();
      await page.getByRole('button', {name: 'Refresh status', exact: true}).click();
      if (outcome === 'incomplete') await page.getByRole('heading', {name: 'Project needs inspection'}).waitFor();
      else await page.getByRole('button', {name: 'Create project', exact: true}).waitFor();
    }
    assert.equal(
      await page.evaluate(
        () =>
          window.workspaceFixture.calls.filter((c) => c.method !== 'GET' && !c.path.includes('/api/factory/')).length
      ),
      1
    );
    assert.equal(await page.evaluate(() => window.workspaceFixture.sockets.length), 0);
  });
test('first use: keyboard selection and post-commit Change inspects without writes', async (t) => {
  const page = await fixture(t, 'page', undefined, true, '/forgejo.test');
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  await reviewFirstIntent(page);
  await page.getByRole('radio', {name: /alice\/Alpha/}).waitFor();
  assert(await page.getByRole('button', {name: 'Continue', exact: true}).isDisabled());
  assert.equal(
    await page.getByRole('link', {name: 'Create a new repository'}).getAttribute('href'),
    '/forgejo.test/repo/create'
  );
  await page.getByRole('radio', {name: /alice\/Alpha/}).focus();
  await page.keyboard.press('Space');
  await page.getByRole('button', {name: 'Continue', exact: true}).focus();
  await page.keyboard.press('Enter');
  await page.getByRole('button', {name: 'Prepare project', exact: true}).click();
  await page.getByRole('heading', {name: 'Configure project'}).waitFor();
  const committed = await page.evaluate(() => window.location.pathname);
  await page.getByRole('button', {name: 'Change repository', exact: true}).click();
  await page.waitForFunction(() => {
    const intent = document.querySelector('soda-intent');
    const h2 = Array.from(intent?.shadowRoot?.querySelectorAll('h2') || []).find(
      (h) => h.textContent?.trim() === 'Review your product direction'
    );
    if (!h2) return false;
    const r = h2.getBoundingClientRect();
    return r.width > 0 && r.height > 0 && getComputedStyle(h2).visibility === 'visible';
  });
  assert.equal(await page.getByRole('heading', {name: 'Choose a repository'}).count(), 0);
  assert.equal(await page.evaluate(() => window.location.pathname), committed);
  assert.equal(await focusedReviewHeading(page), 'Review your product direction');
  await page.getByRole('button', {name: 'Back', exact: true}).click();
  await page.getByRole('heading', {name: 'What do you want to build?'}).waitFor();
  assert.equal(
    await page.evaluate(
      () => window.workspaceFixture.calls.filter((c) => c.method !== 'GET' && !c.path.includes('/api/factory/')).length
    ),
    0
  );
});
test('first use: committed inspection mounts the request overview with verified native links', async (t) => {
  const page = await fixture(t, 'page', undefined, true, '/forgejo.test');
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  await reviewFirstIntent(page);
  await page.getByRole('radio', {name: /alice\/Alpha/}).waitFor();
  await page.getByRole('radio', {name: /alice\/Alpha/}).focus();
  await page.keyboard.press('Space');
  await page.getByRole('button', {name: 'Continue', exact: true}).focus();
  await page.keyboard.press('Enter');
  await page.getByRole('button', {name: 'Prepare project', exact: true}).click();
  await page.getByRole('heading', {name: 'Configure project'}).waitFor();
  await completeBoundIntentPlanning(page);
  await page.getByRole('button', {name: 'Change repository', exact: true}).click();
  const overview = page.locator('soda-factory-work-overview');
  await overview.getByText('Build a small project tracker').waitFor();
  await overview.getByText('Native work generated.').waitFor();
  const link = overview.getByRole('link', {name: 'issue #5', exact: true});
  assert.equal(await link.getAttribute('href'), '/forgejo.test/alice/Alpha/issues/5');
});
test('first use: uncommitted review mounts no request overview', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  await page.getByRole('heading', {name: 'What do you want to build?', exact: true}).waitFor();
  await page.getByLabel('Describe what you want to build', {exact: true}).fill('Build a small project tracker');
  await page.getByRole('button', {name: 'Explore ideas', exact: true}).click();
  await page.getByRole('heading', {name: 'Review your product direction', exact: true}).waitFor();
  assert.equal(await page.locator('soda-factory-work-overview').count(), 0);
});
test('first use: picker Back preserves the reviewed direction', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  await reviewFirstIntent(page);
  await page.getByRole('heading', {name: 'Choose a repository'}).waitFor();
  await page.locator('.soda-setup-topbar').getByRole('button', {name: 'Back', exact: true}).click();
  await page.getByRole('heading', {name: 'Review your product direction', exact: true}).waitFor();
  assert.equal(await page.getByRole('heading', {name: 'Choose a repository'}).count(), 0);
  assert(await page.getByRole('button', {name: 'Choose repository', exact: true}).isEnabled());
  assert.equal(await focusedReviewHeading(page), 'Review your product direction');
  // A declared native planner peer supplies concrete experience prose; the
  // browser must keep it unreviewed until the explicit semantic command.
  await installAuthoredPlanningPeer(page);
  await page.getByRole('button', {name: 'Choose repository', exact: true}).click();
  await page.getByRole('radio', {name: /alice\/Alpha/}).check();
  await page.getByRole('button', {name: 'Continue', exact: true}).click();
  await page.getByRole('button', {name: 'Check walkthrough', exact: true}).click();
  await page.getByRole('button', {name: 'Accept and build', exact: true}).waitFor();
  await page.getByText('The team can see who will do what next', {exact: true}).first().waitFor();
  assert.equal(await page.getByRole('heading', {name: 'Configure project', exact: true}).count(), 0);
  assert.equal(await page.getByRole('button', {name: 'Choose repository', exact: true}).count(), 0);
  const boundRoute = await page.evaluate(() => location.pathname);
  await page
    .locator('section[aria-label="Draft planning"]')
    .getByRole('button', {name: 'Refresh status', exact: true})
    .click();
  await page.getByRole('button', {name: 'Accept and build', exact: true}).click();
  await page.locator('soda-factory-work-overview').getByText('Native work generated.').waitFor();
  assert.equal(await page.evaluate(() => location.pathname), boundRoute);
  const mutations = await page.evaluate(() => window.workspaceFixture.calls.filter((call) => call.method === 'POST'));
  assert.equal(
    mutations.filter((call) => call.path.endsWith('/plan')).length,
    1,
    'refresh must not dispatch another plan'
  );
  assert.equal(mutations.filter((call) => call.path.endsWith('/generate')).length, 1, 'acceptance creates work once');
  assert.equal(
    mutations.filter((call) => call.path.endsWith('/plan/review')).length,
    1,
    'refresh restores reviewed content without consulting again'
  );
});
test('first use: resumed draft explicitly raises work limit before planning the same request', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  await page.locator('details.bounds summary').click();
  await page.getByLabel('Maximum work items', {exact: true}).fill('1');
  await reviewFirstIntent(page);
  await page.getByRole('radio', {name: /alice\/Alpha/}).check();
  await page.getByRole('button', {name: 'Continue', exact: true}).click();
  await page.getByRole('heading', {name: 'Review your product direction', exact: true}).waitFor();
  const original = await page.evaluate(async () => {
    const calls = window.workspaceFixture.calls;
    const saved = calls.filter((call) => call.method === 'POST' && call.path.endsWith('/api/factory/intents')).at(-1)!;
    const prior = saved.body!;
    // Declared server peer supplies the canonical comment-review draft. The
    // PostgreSQL integration verifies its real accepted-generation provenance.
    const ideas = prior.ideas as Record<string, unknown>[];
    const finance = {
      id: 'd' + '9'.repeat(24),
      description: 'Require Finance approval above EUR 1000',
      consequence: 'Preserve existing requests and approval history',
      decision: 'accept',
      assumption: false,
      source: 'requested',
      provenance: 'human',
      actor: '1',
      evidence: 'Feed comment c1 revision 0 admitted by the product owner',
    };
    const draft: Record<string, unknown> = {
      ...prior,
      id: 'd' + '8'.repeat(24),
      predecessor: prior.id,
      ideas: [...ideas, finance],
      effective_outcomes: [...ideas.map((idea) => idea.id), finance.id],
    };
    const response = await fetch(saved.path, {
      method: 'POST',
      headers: {'Content-Type': 'application/json'},
      body: JSON.stringify(draft),
    });
    if (!response.ok) throw Error('Synthetic comment-review draft unavailable');
    return draft;
  });
  assert.equal(original.max_issues, 1);
  // Leave and resume through the native request route, as the comment handoff does.
  await page.evaluate(() => {
    history.pushState({}, '', '/workspace/new');
    window.dispatchEvent(new PopStateEvent('popstate'));
  });
  await page.getByRole('heading', {name: 'What do you want to build?', exact: true}).waitFor();
  await page.evaluate((intent) => {
    history.pushState({}, '', '/workspace/requests/' + intent);
    window.dispatchEvent(new PopStateEvent('popstate'));
  }, original.intent);
  await page.getByRole('heading', {name: 'Review your product direction', exact: true}).waitFor();
  await installAuthoredPlanningPeer(page);
  await page.locator('details.bounds summary').click();
  assert.equal(await page.getByLabel('Maximum work items', {exact: true}).inputValue(), '1');
  const before = await page.evaluate(() => window.workspaceFixture.calls.length);
  await page.getByLabel('Maximum work items', {exact: true}).fill('2');
  assert.equal(await page.getByRole('button', {name: 'Continue to planning', exact: true}).count(), 0);
  assert(await page.locator('soda-intent').getByRole('button', {name: 'Update proposal', exact: true}).isEnabled());
  assert.equal(await page.locator('soda-intent').locator('button.primary').innerText(), 'Update proposal');
  await page.getByText('Planning details', {exact: true}).click();
  assert(await page.getByRole('button', {name: 'Plan work', exact: true}).isDisabled());
  assert.equal(
    await page.evaluate(() => window.workspaceFixture.calls.length),
    before,
    'Editing a limit alone cannot launch work'
  );
  await page.getByRole('button', {name: 'Update proposal', exact: true}).click();
  await page.getByRole('button', {name: 'Continue to planning', exact: true}).waitFor();
  const draftOrder = await page
    .locator('soda-intent section[aria-label="Draft planning"] button')
    .evaluateAll((buttons) => buttons.map((button) => button.textContent?.trim()));
  const primaryAt = draftOrder.findIndex((label) => label === 'Continue to planning');
  const prepareAt = draftOrder.findIndex((label) => label === 'Prepare project');
  assert(
    primaryAt !== -1 && prepareAt !== -1 && primaryAt < prepareAt,
    `primary planning action precedes project preparation in Draft planning: ${JSON.stringify(draftOrder)}`
  );
  const revised = await page.evaluate(
    () =>
      window.workspaceFixture.calls
        .filter((call) => call.method === 'POST' && call.path.endsWith('/api/factory/intents'))
        .at(-1)!.body!
  );
  assert.equal(revised.intent, original.intent);
  assert.equal(revised.predecessor, original.id);
  assert.equal(revised.repository, '7');
  assert.equal(revised.max_issues, 2);
  assert.deepEqual(revised.tasks, original.tasks, 'A bounds edit does not rewrite the accepted work shape');
  assert.deepEqual(revised.ideas, original.ideas, 'A bounds edit preserves comment identity and attribution');
  assert.deepEqual(revised.effective_outcomes, original.effective_outcomes);
  assert.equal(
    await page.evaluate(
      (before) => window.workspaceFixture.calls.slice(before).some((call) => call.path.endsWith('/explore')),
      before
    ),
    false,
    'Limits alone must not remint product tasks through exploration'
  );
  assert.equal(
    await page.evaluate(
      (before) => window.workspaceFixture.calls.slice(before).some((call) => call.path.endsWith('/generate')),
      before
    ),
    false
  );
  await page.getByRole('button', {name: 'Continue to planning', exact: true}).click();
  await page.getByRole('button', {name: 'Check walkthrough', exact: true}).click();
  await page.getByRole('button', {name: 'Accept and build', exact: true}).click();
  await page.locator('soda-factory-work-overview').getByText('Native work generated.').waitFor();
  const accepted = await page.evaluate(
    () =>
      window.workspaceFixture.calls
        .filter((call) => call.method === 'POST' && call.path.endsWith('/api/factory/intents'))
        .at(-1)!.body!
  );
  assert.equal(accepted.state, 'accepted');
  assert.equal(accepted.predecessor, revised.id);
  assert.equal(accepted.intent, original.intent);
  assert.equal(accepted.max_issues, 2);
  assert.deepEqual(accepted.tasks, original.tasks);
  assert.deepEqual(accepted.ideas, original.ideas);
  assert.deepEqual(accepted.effective_outcomes, original.effective_outcomes);
});
test('review preserves the saved request after preparation and waits for an explicit planning action', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  await reviewFirstIntent(page);
  await page.evaluate(() => {
    const original = window.fetch;
    Object.defineProperty(window, 'fetch', {
      configurable: true,
      value: (input: RequestInfo | URL, init?: RequestInit) => {
        const path = String(input),
          method = init?.method || 'GET';
        if (path.includes('/api/factory/intents/') && path.endsWith('/plan')) {
          const revision = (document.querySelector('soda-intent') as unknown as {revision: {id: string}}).revision.id;
          const body = init?.body ? JSON.parse(String(init.body)) : null;
          window.workspaceFixture.calls.push({path, method, body});
          return Promise.resolve(
            Response.json(
              method === 'POST'
                ? {
                    launched: false,
                    waits: [{reason: 'preparation_not_ready', detail: 'Project preparation is required.'}],
                  }
                : {revision, state: 'none'},
              {status: method === 'POST' ? 202 : 200}
            )
          );
        }
        return original(input, init);
      },
    });
  });
  await page.getByRole('radio', {name: /alice\/Alpha/}).check();
  await page.getByRole('button', {name: 'Continue', exact: true}).click();
  await page.getByText(/Planning is waiting:/).waitFor();
  const before = await page.evaluate(() => ({
    revision: (document.querySelector('soda-intent') as unknown as {revision: unknown}).revision,
    posts: window.workspaceFixture.calls.filter((call) => call.method === 'POST' && call.path.endsWith('/plan')).length,
  }));
  await page.getByRole('button', {name: 'Prepare project', exact: true}).click();
  await page.getByRole('heading', {name: 'Configure project'}).waitFor();
  await page.getByRole('button', {name: 'Back', exact: true}).click();
  await page.getByRole('button', {name: 'Continue to planning', exact: true}).waitFor();
  await page
    .locator('soda-intent')
    .getByText('Your request and choices are saved. Continue to planning when project preparation is ready.', {
      exact: true,
    })
    .waitFor();
  assert.equal(await page.getByLabel('Desired outcome', {exact: true}).inputValue(), 'Build a small project tracker');
  const returned = await page.evaluate(() => ({
    revision: (document.querySelector('soda-intent') as unknown as {revision: unknown}).revision,
    posts: window.workspaceFixture.calls.filter((call) => call.method === 'POST' && call.path.endsWith('/plan')).length,
  }));
  assert.deepEqual(
    returned,
    before,
    'Returning from preparation must preserve the request without spending another call'
  );
  await page.getByRole('button', {name: 'Continue to planning', exact: true}).click();
  await page.getByText(/Planning is waiting:/).waitFor();
  assert.equal(
    await page.evaluate(
      () => window.workspaceFixture.calls.filter((call) => call.method === 'POST' && call.path.endsWith('/plan')).length
    ),
    before.posts + 1
  );
});

test('first use: picker summarizes reviewed authorization and the create effect', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  await reviewFirstIntent(page);
  await page.getByRole('heading', {name: 'Choose a repository'}).waitFor();
  await page.getByText('Saving 1 authorized idea and 1 work item to the chosen repository.').waitFor();
  await page
    .getByText('The draft is linked first. Factory work is created after a completed plan is accepted.')
    .waitFor();
  const create = page.getByRole('link', {name: 'Create a new repository'});
  assert.equal(await create.getAttribute('target'), '_blank');
  assert.equal(await create.getAttribute('rel'), 'noopener');
});
test('first use: resumed accepted request keeps its repository without repeating the picker', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  await reviewFirstIntent(page);
  await page.getByRole('radio', {name: /alice\/Alpha/}).check();
  await page.getByRole('button', {name: 'Continue', exact: true}).click();
  await page.getByRole('button', {name: 'Prepare project', exact: true}).click();
  await page.getByRole('heading', {name: 'Configure project'}).waitFor();
  await completeBoundIntentPlanning(page);
  const intent = await page.evaluate(() => {
    const calls = window.workspaceFixture.calls as {body?: Record<string, unknown>}[];
    const accepted = calls.find((call) => (call.body as Record<string, unknown> | undefined)?.state === 'accepted');
    return String((accepted?.body as Record<string, unknown> | undefined)?.intent || '');
  });
  assert.match(intent, /^d[0-9a-f]{24}$/);
  await page.getByRole('button', {name: 'Change repository', exact: true}).click();
  await page.waitForFunction(() => {
    const el = document.querySelector('soda-intent');
    const h2 = Array.from(el?.shadowRoot?.querySelectorAll('h2') || []).find(
      (h) => h.textContent?.trim() === 'Review your product direction'
    );
    if (!h2) return false;
    const r = h2.getBoundingClientRect();
    return r.width > 0 && r.height > 0;
  });
  await page.getByRole('button', {name: 'Back', exact: true}).click();
  await page.getByRole('heading', {name: 'What do you want to build?'}).waitFor();
  await page.evaluate((id) => {
    window.history.pushState({}, '', `/workspace/requests/${id}`);
    window.dispatchEvent(new PopStateEvent('popstate'));
  }, intent);
  await page.getByRole('heading', {name: 'Review your product direction', exact: true}).waitFor();
  assert.equal(await page.getByRole('button', {name: 'Choose repository', exact: true}).count(), 0);
  await page.locator('soda-factory-work-overview').getByText('Native work generated.').waitFor();
});
test('first use: resumed accepted request restores its overview with native links and no writes', async (t) => {
  const page = await fixture(t, 'page', undefined, true, '/forgejo.test');
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  await reviewFirstIntent(page);
  await page.getByRole('radio', {name: /alice\/Alpha/}).check();
  await page.getByRole('button', {name: 'Continue', exact: true}).click();
  await page.getByRole('button', {name: 'Prepare project', exact: true}).click();
  await page.getByRole('heading', {name: 'Configure project'}).waitFor();
  await completeBoundIntentPlanning(page);
  const intent = await page.evaluate(() => {
    const calls = window.workspaceFixture.calls as {body?: Record<string, unknown>}[];
    const accepted = calls.find((call) => (call.body as Record<string, unknown> | undefined)?.state === 'accepted');
    return String((accepted?.body as Record<string, unknown> | undefined)?.intent || '');
  });
  assert.match(intent, /^d[0-9a-f]{24}$/);
  await page.getByRole('button', {name: 'Change repository', exact: true}).click();
  await page.waitForFunction(() => {
    const el = document.querySelector('soda-intent');
    const h2 = Array.from(el?.shadowRoot?.querySelectorAll('h2') || []).find(
      (h) => h.textContent?.trim() === 'Review your product direction'
    );
    if (!h2) return false;
    const r = h2.getBoundingClientRect();
    return r.width > 0 && r.height > 0;
  });
  await page.getByRole('button', {name: 'Back', exact: true}).click();
  await page.getByRole('heading', {name: 'What do you want to build?'}).waitFor();
  assert.equal(await page.locator('soda-factory-work-overview').count(), 0);
  const settledCalls = await page.evaluate(() => window.workspaceFixture.calls.length);
  await page.evaluate((id) => {
    window.history.pushState({}, '', `/forgejo.test/workspace/requests/${id}`);
    window.dispatchEvent(new PopStateEvent('popstate'));
  }, intent);
  const overview = page.locator('soda-factory-work-overview');
  await overview.getByText('Build a small project tracker').waitFor();
  await overview.getByText('Native work generated.').waitFor();
  const link = overview.getByRole('link', {name: 'issue #5', exact: true});
  assert.equal(await link.getAttribute('href'), '/forgejo.test/alice/Alpha/issues/5');
  const writes = await page.evaluate(
    (settled) => window.workspaceFixture.calls.slice(settled).filter((call) => (call.method || 'GET') !== 'GET').length,
    settledCalls
  );
  assert.equal(writes, 0);
});
test('first use: committed inspection mounts plan, detail and revision from the overview scope', async (t) => {
  const page = await fixture(t, 'page', undefined, true, '/forgejo.test');
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  await reviewFirstIntent(page);
  await page.getByRole('radio', {name: /alice\/Alpha/}).check();
  await page.getByRole('button', {name: 'Continue', exact: true}).click();
  await page.getByRole('button', {name: 'Prepare project', exact: true}).click();
  await page.getByRole('heading', {name: 'Configure project'}).waitFor();
  await completeBoundIntentPlanning(page);
  const intent = await page.evaluate(() => {
    const calls = window.workspaceFixture.calls as {body?: Record<string, unknown>}[];
    const accepted = calls.find((call) => (call.body as Record<string, unknown> | undefined)?.state === 'accepted');
    return String((accepted?.body as Record<string, unknown> | undefined)?.intent || '');
  });
  assert.match(intent, /^d[0-9a-f]{24}$/);
  await page.getByRole('button', {name: 'Change repository', exact: true}).click();
  const overview = page.locator('soda-factory-work-overview');
  await overview.getByText('Build a small project tracker').waitFor();
  await page.locator('soda-factory-work-plan').waitFor();
  await page.locator('soda-factory-work-detail').waitFor();
  await page.locator('soda-factory-direction-revision').waitFor();
  assert.equal(await page.locator('soda-factory-work-detail').count(), 1);
  await page.waitForFunction(() =>
    window.workspaceFixture.calls.some((call) =>
      call.path.endsWith('/api/repositories/7/factory/issues/5/plan?limit=5')
    )
  );
  await page.waitForFunction(() =>
    window.workspaceFixture.calls.some((call) => call.path.endsWith('/api/repositories/7/factory/issues/5'))
  );
  await page.waitForFunction(
    (id) =>
      window.workspaceFixture.calls.some((call) => call.path.endsWith(`/api/factory/intents/${id}/revisions?limit=2`)),
    intent
  );
  await page.waitForFunction(() => {
    const el = document.querySelector('soda-intent');
    const h2 = Array.from(el?.shadowRoot?.querySelectorAll('h2') || []).find(
      (h) => h.textContent?.trim() === 'Review your product direction'
    );
    if (!h2) return false;
    const r = h2.getBoundingClientRect();
    return r.width > 0 && r.height > 0;
  });
  await page.getByRole('button', {name: 'Back', exact: true}).click();
  await page.getByRole('heading', {name: 'What do you want to build?'}).waitFor();
  assert.equal(await page.locator('soda-factory-work-overview').count(), 0);
  assert.equal(await page.locator('soda-factory-work-plan').count(), 0);
  assert.equal(await page.locator('soda-factory-work-detail').count(), 0);
  assert.equal(await page.locator('soda-factory-direction-revision').count(), 0);
});
test('setup: detail Watch binds the live run target into an observe-only watch', {timeout: 30000}, async (t) => {
  const page = await fixture(t, 'page', undefined, true, '/forgejo.test');
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  await reviewFirstIntent(page);
  await page.getByRole('radio', {name: /alice\/Alpha/}).check();
  await page.getByRole('button', {name: 'Continue', exact: true}).click();
  await page.getByRole('button', {name: 'Prepare project', exact: true}).click();
  await page.getByRole('heading', {name: 'Configure project'}).waitFor();
  await completeBoundIntentPlanning(page);
  await page.getByRole('button', {name: 'Create project', exact: true}).click();
  await page.getByRole('heading', {name: 'Review your product direction', exact: true}).waitFor();
  await page.getByRole('button', {name: 'Back', exact: true}).click();
  await page.getByRole('button', {name: 'Terminals', exact: true}).click();
  await page.getByRole('button', {name: 'Join project', exact: true}).click();
  await page.getByRole('heading', {name: 'Open your first terminal'}).waitFor();
  await page.evaluate(() => {
    window.workspaceFixture.enableFactoryRun('5');
    const [space] = window.workspaceFixture.spaces;
    space?.factory_runs.push({
      id: 'd'.repeat(32),
      role: 'soda-coder',
      issue: '5',
      attempt: 'attempt-1',
      reconciled: false,
    });
    return window.workspaceFixture.api.refresh();
  });
  await resumeSavedPlan(page);
  const resumed = page.locator('soda-factory-work-overview');
  await resumed.getByText('Build a small project tracker').waitFor();
  await resumed.getByText('Native work generated.').waitFor();
  await page.locator('soda-factory-work-detail').waitFor();
  const detail = page.locator('soda-factory-work-detail');
  await detail.getByRole('button', {name: 'Watch output', exact: true}).waitFor();
  assert.equal(await detail.getByText('no run selected').count(), 0);
  await detail.getByText('run dddddddd').first().waitFor();
  const environment = await page.evaluate(() => window.workspaceFixture.spaces[0]?.environment.id ?? '');
  assert.match(environment, /^p[0-9a-f]{24}$/);
  await detail.getByText(`project ${environment}`).waitFor();
  const calls = await page.evaluate(() => window.workspaceFixture.calls.length);
  await detail.getByRole('button', {name: 'Watch output', exact: true}).click();
  // Reveal: the Watch click leaves setup for the existing visible watch surface.
  await page.locator('.soda-factory.is-connected .xterm-rows').waitFor();
  await page.waitForFunction(() =>
    document.querySelector('.soda-factory .xterm-rows')?.textContent?.includes('Fixture factory output')
  );
  assert.match(await page.locator('.soda-factory [role="status"]').innerText(), /Watching soda-coder · issue #5/);
  const title = await page.evaluate(
    () => document.querySelector('.soda-factory .soda-terminal-context span')?.textContent ?? ''
  );
  assert.equal(title, 'soda-coder · issue #5');
  assert.ok(
    await page.evaluate(() =>
      window.workspaceFixture.sockets.some((socket) => String(socket.url).includes('/factory/runs/dddddddd'))
    )
  );
  const writes = await page.evaluate(
    (before) => window.workspaceFixture.calls.slice(before).filter((call) => call.method !== 'GET'),
    calls
  );
  assert.deepEqual(writes, []);
  const leftRequest = await page.evaluate(() => window.location.pathname);
  assert.ok(!leftRequest.includes('/workspace/requests/'));
  // Return: the saved request resumes with its committed identity and targets intact.
  await resumeSavedPlan(page);
  await page.locator('soda-factory-work-detail').waitFor();
  const returned = page.locator('soda-factory-work-detail');
  assert.equal(await returned.getByText('no run selected').count(), 0);
  await returned.getByText('run dddddddd').first().waitFor();
  const backOnRequest = await page.evaluate(() => window.location.pathname);
  assert.ok(backOnRequest.includes('/workspace/requests/'));
  // Duplicate/preservation: watching again reveals the same live watch untouched.
  const statusCalls = await page.evaluate(
    () => window.workspaceFixture.calls.filter((call) => call.path.includes('/factory/runs/dddddddd')).length
  );
  await returned.getByRole('button', {name: 'Watch output', exact: true}).click();
  await page.locator('.soda-factory.is-connected .xterm-rows').waitFor();
  assert.equal(await page.evaluate(() => document.querySelectorAll('.soda-factory').length), 1);
  assert.equal(await page.evaluate(() => window.workspaceFixture.sockets.length), 1);
  assert.equal(
    await page.evaluate(
      () => window.workspaceFixture.calls.filter((call) => call.path.includes('/factory/runs/dddddddd')).length
    ),
    statusCalls
  );
  assert.match(await page.locator('.soda-factory [role="status"]').innerText(), /Watching soda-coder · issue #5/);
});
test(
  'setup: Watch stays independent of project filters and reveals the retained view at compact widths',
  {timeout: 90000},
  async (t) => {
    const page = await fixture(t, 'page', undefined, true, '/forgejo.test');
    await page.evaluate(() => window.workspaceFixture.api.refresh());
    await reviewFirstIntent(page);
    await page.getByRole('radio', {name: /alice\/Alpha/}).check();
    await page.getByRole('button', {name: 'Continue', exact: true}).click();
    await page.getByRole('button', {name: 'Prepare project', exact: true}).click();
    await page.getByRole('heading', {name: 'Configure project'}).waitFor();
    await completeBoundIntentPlanning(page);
    await page.getByRole('button', {name: 'Create project', exact: true}).click();
    await page.getByRole('heading', {name: 'Review your product direction', exact: true}).waitFor();
    await page.getByRole('button', {name: 'Back', exact: true}).click();
    await page.getByRole('button', {name: 'Terminals', exact: true}).click();
    await page.getByRole('button', {name: 'Join project', exact: true}).click();
    await page.getByRole('heading', {name: 'Open your first terminal'}).waitFor();
    await page.evaluate(() => {
      window.workspaceFixture.enableFactoryRun('5');
      const [space] = window.workspaceFixture.spaces;
      space?.factory_runs.push({
        id: 'd'.repeat(32),
        role: 'soda-coder',
        issue: '5',
        attempt: 'attempt-1',
        reconciled: false,
      });
      return window.workspaceFixture.api.refresh();
    });
    await resumeSavedPlan(page);
    const detail = page.locator('soda-factory-work-detail');
    await detail.getByRole('button', {name: 'Watch output', exact: true}).click();
    await page.locator('.soda-factory.is-connected .xterm-rows').waitFor();
    // Phase 1: filtering the Project directory must keep the selected work and
    // its live watch connected and visible. Revealing it again keeps one owner.
    await page.locator('.soda-workspace-navigation input[type="search"]').fill('zzz-no-such-project', {force: true});
    assert(await page.locator('.soda-factory').isVisible());
    await resumeSavedPlan(page);
    await page.locator('soda-factory-work-detail').waitFor();
    await page.locator('soda-factory-work-detail').getByRole('button', {name: 'Watch output', exact: true}).click();
    await page.locator('.soda-factory.is-connected .xterm-rows').waitFor();
    assert.match(await page.locator('.soda-factory [role="status"]').innerText(), /Watching soda-coder · issue #5/);
    assert.equal(await page.locator('.soda-workspace-navigation input[type="search"]').inputValue(), '');
    assert.equal(await page.evaluate(() => document.querySelectorAll('.soda-factory').length), 1);
    assert.equal(
      await page.evaluate(
        () =>
          window.workspaceFixture.sockets.filter(
            (socket) => !socket.closed && String(socket.url).includes('/factory/runs/dddddddd')
          ).length
      ),
      1
    );
    // Phase 2: hiding the exact watch through the real factory command, then
    // watching again, reveals the same live watch without a second connection.
    await page.evaluate(() => {
      document
        .querySelector('[data-factory-owner]')
        ?.dispatchEvent(new CustomEvent('soda-factory-command', {bubbles: true, detail: 'hide'}));
    });
    await page.locator('.soda-factory').waitFor({state: 'hidden'});
    await resumeSavedPlan(page);
    await page.locator('soda-factory-work-detail').waitFor();
    await page.locator('soda-factory-work-detail').getByRole('button', {name: 'Watch output', exact: true}).click();
    await page.locator('.soda-factory.is-connected .xterm-rows').waitFor();
    assert.match(await page.locator('.soda-factory [role="status"]').innerText(), /Watching soda-coder · issue #5/);
    assert.equal(await page.evaluate(() => document.querySelectorAll('.soda-factory').length), 1);
    assert.equal(
      await page.evaluate(
        () =>
          window.workspaceFixture.sockets.filter(
            (socket) => !socket.closed && String(socket.url).includes('/factory/runs/dddddddd')
          ).length
      ),
      1
    );
    // Phase 3: compact layout shows the watch in Project work without requiring the directory.
    await resumeSavedPlan(page);
    await page.locator('soda-factory-work-detail').waitFor();
    await page.setViewportSize({width: 500, height: 800});
    await page.waitForFunction(
      () => (document.querySelector('soda-spaces') as unknown as {compact: boolean}).compact === true
    );
    await page.locator('soda-factory-work-detail').getByRole('button', {name: 'Watch output', exact: true}).click();
    await page.getByRole('button', {name: 'Work', exact: true}).waitFor();
    assert.equal(await page.locator('.soda-workspace-navigation:visible').count(), 0);
    await page.locator('.soda-factory.is-connected .xterm-rows').waitFor();
    assert.match(await page.locator('.soda-factory [role="status"]').innerText(), /Watching soda-coder · issue #5/);
  }
);
test('first use: superseded search and retired actor cannot publish stale private choices', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await page.evaluate(() => {
    const original = window.fetch;
    Object.defineProperty(window, 'fetch', {
      configurable: true,
      value: (input: RequestInfo | URL, init?: RequestInit) => {
        const url = new URL(String(input), location.origin);
        if (url.pathname.endsWith('/api/repositories') && url.searchParams.get('q') === 'Alpha')
          return new Promise<Response>((resolve) =>
            window.addEventListener(
              'release-old-search',
              () =>
                resolve(
                  Response.json({
                    items: [{id: '7', owner: 'alice', name: 'Alpha', can_create: true, project: null}],
                  })
                ),
              {once: true}
            )
          );
        return original(input, init);
      },
    });
    return window.workspaceFixture.api.refresh();
  });
  await reviewFirstIntent(page);
  await page.getByRole('radio', {name: /alice\/Alpha/}).waitFor();
  const search = page.getByRole('searchbox', {name: 'Search repositories'});
  await search.fill('Alpha');
  await search.press('Enter');
  await search.fill('Beta');
  await search.press('Enter');
  await page.getByRole('radio', {name: /alice\/Beta/}).check();
  await page.evaluate(() => window.dispatchEvent(new Event('release-old-search')));
  assert.equal(await page.getByRole('radio', {name: /alice\/Alpha/}).count(), 0);
  assert(await page.getByRole('radio', {name: /alice\/Beta/}).isChecked());
  await search.fill('Alpha');
  await search.press('Enter');
  await page.evaluate(async () => {
    window.workspaceFixture.setUser('2');
    await window.workspaceFixture.api.refresh();
    window.dispatchEvent(new Event('release-old-search'));
  });
  await page.getByRole('button', {name: 'Reload Spaces', exact: true}).waitFor();
  assert.equal(await page.getByRole('radio').count(), 0);
  assert.equal(await page.getByRole('heading', {name: 'What do you want to build?'}).count(), 0);
  assert.equal(
    await page.evaluate(
      () => window.workspaceFixture.calls.filter((c) => c.method !== 'GET' && !c.path.includes('/api/factory/')).length
    ),
    0
  );
});
for (const administrator of [true, false])
  test(`first use: stopped project Start is explicit and authorized (${administrator})`, async (t) => {
    const page = await fixture(t, 'page', undefined, true);
    await page.evaluate(() => window.workspaceFixture.api.refresh());
    await chooseFirstRepository(page);
    await page.getByRole('button', {name: 'Create project', exact: true}).click();
    await page.getByRole('heading', {name: 'Review your product direction', exact: true}).waitFor();
    await page.getByRole('button', {name: 'Back', exact: true}).click();
    await page.getByRole('button', {name: 'Terminals', exact: true}).click();
    await page.getByRole('button', {name: 'Join project', exact: true}).waitFor();
    await page.evaluate(async (administrator) => {
      const f = window.workspaceFixture,
        space = f.spaces[0];
      if (!space?.observed) throw Error('fixture');
      space.observed.running = false;
      space.environment_administrator = administrator;
      await f.api.refresh();
    }, administrator);
    await page.getByRole('heading', {name: 'Project not ready'}).waitFor();
    assert.equal(
      await page.evaluate(
        () =>
          window.workspaceFixture.calls.filter((c) => c.method !== 'GET' && !c.path.includes('/api/factory/')).length
      ),
      1
    );
    if (administrator) {
      await page.getByRole('button', {name: 'Start project', exact: true}).click();
      await page.getByRole('button', {name: 'Join project', exact: true}).waitFor();
      assert.equal(
        await page.evaluate(
          () =>
            window.workspaceFixture.calls.filter((c) => c.method !== 'GET' && !c.path.includes('/api/factory/')).length
        ),
        2
      );
    } else assert.equal(await page.getByRole('button', {name: 'Start project', exact: true}).count(), 0);
    assert.equal(await page.evaluate(() => window.workspaceFixture.sockets.length), 0);
  });
test('first use: second-project setup cancellation preserves the live renderer, input target and layout', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  await chooseFirstRepository(page);
  await page.getByRole('button', {name: 'Create project', exact: true}).click();
  await page.getByRole('heading', {name: 'Review your product direction', exact: true}).waitFor();
  await page.getByRole('button', {name: 'Back', exact: true}).click();
  await page.getByRole('button', {name: 'Terminals', exact: true}).click();
  await page.getByRole('button', {name: 'Join project', exact: true}).click();
  await page.getByRole('heading', {name: 'Open your first terminal'}).waitFor();
  await page.getByRole('button', {name: 'New terminal', exact: true}).click();
  await page.locator('.is-connected').waitFor();
  const screen = await page.locator('.xterm').elementHandle();
  const before = await page.evaluate(() => ({
    layout: sessionStorage.getItem('soda-spaces:v3:1'),
    writes: window.workspaceFixture.calls.filter((c) => c.method !== 'GET' && !c.path.includes('/api/factory/')).length,
  }));
  await reviewFirstIntent(page);
  await page.getByRole('radio', {name: /alice\/Beta/}).check();
  await page.getByRole('button', {name: 'Continue', exact: true}).click();
  await page.getByRole('button', {name: 'Prepare project', exact: true}).click();
  await page.getByRole('heading', {name: 'Configure project'}).waitFor();
  assert.equal(await page.locator('.soda-workspace-terminal:visible').count(), 0);
  await page.getByRole('button', {name: 'Cancel setup', exact: true}).click();
  await page.locator('.soda-workspace-terminal:visible .is-connected').waitFor();
  assert(await screen?.evaluate((node) => node.isConnected));
  assert.deepEqual(
    await page.evaluate(() => ({
      layout: sessionStorage.getItem('soda-spaces:v3:1'),
      writes: window.workspaceFixture.calls.filter((c) => c.method !== 'GET' && !c.path.includes('/api/factory/'))
        .length,
    })),
    before
  );
  assert.deepEqual(await page.evaluate(() => window.workspaceFixture.sockets.map((s) => s.closed)), [0]);
});
test('setup cancel restores the non-first return target instead of selecting the first project', async (t) => {
  const page = await fixture(t, 'page', undefined, false);
  await page.evaluate(() => {
    const original = window.fetch;
    Object.defineProperty(window, 'fetch', {
      configurable: true,
      value: (input: RequestInfo | URL, init?: RequestInit) => {
        const url = new URL(String(input), location.origin);
        if (url.pathname.endsWith('/api/repositories'))
          return Response.json({
            items: [
              {
                id: '7',
                owner: 'alice',
                name: 'Alpha',
                can_create: false,
                project: {id: 'p111111111111111111111111', provisioned: true},
              },
              {
                id: '8',
                owner: 'alice',
                name: 'Beta',
                can_create: false,
                project: {id: 'p222222222222222222222222', provisioned: true},
              },
              {id: '9', owner: 'alice', name: 'Gamma', can_create: true, project: null},
            ],
          });
        return original(input, init);
      },
    });
    return window.workspaceFixture.api.refresh();
  });
  await page.getByRole('heading', {name: 'Projects', exact: true}).waitFor();
  await page.getByRole('button', {name: 'alice/Beta', exact: true}).click();
  await page.getByRole('button', {name: 'Project settings'}).click();
  await page.getByRole('heading', {name: 'Project OS'}).waitFor();
  await page.getByRole('heading', {name: 'Projects', exact: true}).waitFor();
  await reviewFirstIntent(page);
  await page.getByRole('radio', {name: /alice\/Gamma/}).waitFor();
  await page.getByRole('radio', {name: /alice\/Gamma/}).check();
  await page.getByRole('button', {name: 'Continue', exact: true}).click();
  await page.getByRole('button', {name: 'Prepare project', exact: true}).click();
  await page.getByRole('button', {name: 'Cancel setup', exact: true}).waitFor();
  await page.getByRole('button', {name: 'Cancel setup', exact: true}).click();
  await page.getByRole('heading', {name: 'Project OS'}).waitFor();
  await page.getByRole('heading', {name: 'Projects', exact: true}).waitFor();
  await page.getByRole('button', {name: 'alice/Beta', exact: true}).waitFor();
  assert.equal(await projectPressed(page), 'alice/Alpha=false,alice/Beta=true');
  assert.equal(
    await page.evaluate(
      () => window.workspaceFixture.calls.filter((c) => c.method !== 'GET' && !c.path.includes('/api/factory/')).length
    ),
    0
  );
});
test('setup cancel falls back to the first project only when the return target is gone', async (t) => {
  const page = await fixture(t, 'page', undefined, false);
  await page.evaluate(() => {
    const original = window.fetch;
    Object.defineProperty(window, 'fetch', {
      configurable: true,
      value: (input: RequestInfo | URL, init?: RequestInit) => {
        const url = new URL(String(input), location.origin);
        if (url.pathname.endsWith('/api/repositories'))
          return Response.json({
            items: [
              {
                id: '7',
                owner: 'alice',
                name: 'Alpha',
                can_create: false,
                project: {id: 'p111111111111111111111111', provisioned: true},
              },
              {
                id: '8',
                owner: 'alice',
                name: 'Beta',
                can_create: false,
                project: {id: 'p222222222222222222222222', provisioned: true},
              },
              {id: '9', owner: 'alice', name: 'Gamma', can_create: true, project: null},
            ],
          });
        return original(input, init);
      },
    });
    return window.workspaceFixture.api.refresh();
  });
  await page.getByRole('heading', {name: 'Projects', exact: true}).waitFor();
  await page.getByRole('button', {name: 'alice/Beta', exact: true}).click();
  await page.getByRole('heading', {name: 'Projects', exact: true}).waitFor();
  await reviewFirstIntent(page);
  await page.getByRole('radio', {name: /alice\/Gamma/}).waitFor();
  await page.getByRole('radio', {name: /alice\/Gamma/}).check();
  await page.getByRole('button', {name: 'Continue', exact: true}).click();
  await page.getByRole('button', {name: 'Prepare project', exact: true}).click();
  await page.getByRole('button', {name: 'Cancel setup', exact: true}).waitFor();
  await page.evaluate(async () => {
    const spaces = window.workspaceFixture.spaces;
    spaces.splice(
      spaces.findIndex((s) => s.environment.repository_id === '8'),
      1
    );
    await window.workspaceFixture.api.refresh();
  });
  await page.getByRole('button', {name: 'Cancel setup', exact: true}).click();
  await page.getByRole('heading', {name: 'Projects', exact: true}).waitFor();
  await page.getByRole('button', {name: 'alice/Alpha', exact: true}).waitFor();
  assert.equal(await projectPressed(page), 'alice/Alpha=true');
  assert.equal(
    await page.evaluate(
      () => window.workspaceFixture.calls.filter((c) => c.method !== 'GET' && !c.path.includes('/api/factory/')).length
    ),
    0
  );
});
test('setup: join states its account, authority and factory independence', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  await chooseFirstRepository(page);
  await page.getByRole('button', {name: 'Create project', exact: true}).click();
  await page.getByRole('heading', {name: 'Review your product direction', exact: true}).waitFor();
  await page.getByRole('button', {name: 'Back', exact: true}).click();
  await page.getByRole('button', {name: 'Terminals', exact: true}).click();
  await page.getByRole('button', {name: 'Join project', exact: true}).waitFor();
  const journey = page.locator('.soda-project-journey');
  assert.match(await journey.innerText(), /Joining as Forgejo account/);
  assert.match(await journey.innerText(), /repository write access confirmed/);
  assert.match(await journey.innerText(), /Factory work uses its own authority/);
});
test('setup: welcome footer tracks describe, repository and create steps', async (t) => {
  const page = await fixture(t, 'page', undefined, false);
  await page.evaluate(() => {
    const original = window.fetch;
    Object.defineProperty(window, 'fetch', {
      configurable: true,
      value: (input: RequestInfo | URL, init?: RequestInit) => {
        const url = new URL(String(input), location.origin);
        if (url.pathname.endsWith('/api/repositories'))
          return Response.json({
            items: [{id: '9', owner: 'alice', name: 'Gamma', can_create: true, project: null}],
          });
        return original(input, init);
      },
    });
    return window.workspaceFixture.api.refresh();
  });
  await page.locator('.soda-create-another-project:visible').waitFor();
  await page.getByRole('button', {name: 'New project', exact: true}).click();
  const footer = page.locator('.soda-setup-footer');
  await footer.waitFor();
  const steps = async () => textContents(footer.locator('li'));
  const current = async () =>
    footer.locator('li').evaluateAll((items) => items.map((item) => item.getAttribute('aria-current')));
  await page.waitForFunction(() => document.querySelector('.soda-setup-footer')?.getAttribute('data-step') === '1');
  assert.deepEqual(await steps(), ['01 Describe the work', '02 Choose a repository', '03 Create the project']);
  assert.deepEqual(await current(), ['step', 'false', 'false']);
  await reviewFirstIntent(page);
  await page.waitForFunction(() => document.querySelector('.soda-setup-footer')?.getAttribute('data-step') === '2');
  assert.deepEqual(await current(), ['false', 'step', 'false']);
  await page.getByRole('radio', {name: /alice\/Gamma/}).check();
  await page.getByRole('button', {name: 'Continue', exact: true}).click();
  await page.getByRole('button', {name: 'Prepare project', exact: true}).click();
  await page.getByRole('button', {name: 'Create project', exact: true}).waitFor();
  await page.waitForFunction(() => document.querySelector('.soda-setup-footer')?.getAttribute('data-step') === '3');
  assert.deepEqual(await steps(), ['✓ Describe the work', '✓ Choose a repository', '03 Create the project']);
  assert.deepEqual(await current(), ['false', 'false', 'step']);
});

test('setup: failed creation preserves the accepted request across refresh without resubmission', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await page.evaluate(() => {
    window.workspaceFixture.setCreateOutcome('rejected');
    return window.workspaceFixture.api.refresh();
  });
  await chooseFirstRepository(page);
  await page.evaluate(() => {
    const el = document.querySelector('soda-spaces');
    const scope = el?.shadowRoot ?? el;
    const controls = scope?.querySelector('soda-project-controls') as unknown as {
      acceptedRequest: {intent: string; revision: string; objective: string};
    };
    if (!controls) throw Error('Missing project controls');
    controls.acceptedRequest = {
      intent: 'd222222222222222222222222',
      revision: 'd111111111111111111111111',
      objective: 'Build leave tracker',
    };
  });
  await page.getByRole('button', {name: 'Create project', exact: true}).click();
  await page.getByText(/Accepted request preserved: Build leave tracker/).waitFor();
  await page.getByRole('button', {name: 'Refresh status', exact: true}).click();
  await page.getByText(/Accepted request preserved: Build leave tracker/).waitFor();
  assert.match(await page.locator('.soda-project-journey').innerText(), /never\s+resubmit or lose it/);
  assert.equal(
    await page.evaluate(
      () => window.workspaceFixture.calls.filter((c) => c.method !== 'GET' && !c.path.includes('/api/factory/')).length
    ),
    1
  );
});

test('setup: intent entry receives the session actor and starts with an empty request', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  await page.getByRole('heading', {name: 'What do you want to build?', exact: true}).waitFor();
  await page.waitForFunction(() => {
    const root = document.querySelector('soda-spaces');
    const scope = root?.shadowRoot ?? root;
    return (scope?.querySelector('soda-intent') as unknown as {actor: string} | null)?.actor === '1';
  });
  const request = await page.evaluate(() => {
    const root = document.querySelector('soda-spaces');
    const scope = root?.shadowRoot ?? root;
    return (scope?.querySelector('soda-intent') as unknown as {request: string} | null)?.request;
  });
  assert.equal(request, '');
});

test('setup: saved draft resumes from its request route across history traversal', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  await page.getByRole('heading', {name: 'What do you want to build?', exact: true}).waitFor();
  await page.getByLabel('Describe what you want to build', {exact: true}).fill('Build a small project tracker');
  await page.getByRole('button', {name: 'Save draft', exact: true}).click();
  await page.waitForFunction(() => window.location.pathname.startsWith('/workspace/requests/d'));
  const saved = await page.evaluate(() => window.location.pathname);
  await page.getByText('Saved', {exact: true}).waitFor();
  await page.getByRole('button', {name: 'Save and leave', exact: true}).click();
  await page.waitForFunction(() => window.location.pathname === '/workspace');
  await page.getByText('Saved', {exact: true}).waitFor();
  assert.equal(
    await page.getByLabel('Describe what you want to build', {exact: true}).inputValue(),
    'Build a small project tracker'
  );
  await page.evaluate(() => {
    (document.querySelector('soda-intent') as unknown as {reset(): void}).reset();
  });
  await page.waitForFunction(() => {
    const area = document.querySelector('soda-intent')?.shadowRoot?.querySelector('textarea');
    return area instanceof HTMLTextAreaElement && area.value === '';
  });
  await page.goBack();
  await page.waitForFunction(() => window.location.pathname.startsWith('/workspace/requests/d'));
  assert.equal(await page.evaluate(() => window.location.pathname), saved);
  await page.waitForFunction(() => {
    const area = document.querySelector('soda-intent')?.shadowRoot?.querySelector('textarea');
    return area instanceof HTMLTextAreaElement && area.value === 'Build a small project tracker';
  });
  await page.getByText('Saved', {exact: true}).waitFor();
});

test('setup: direct /workspace/new load activates fresh entry in populated workspace', {timeout: 60000}, async (t) => {
  const page = await fixture(t);
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  await page.getByRole('navigation', {name: 'Projects', exact: true}).waitFor();
  const mountNewRoute = () =>
    page.evaluate(async () => {
      window.workspaceFixture = window.createWorkspaceFixture('page', false, '');
      await window.workspaceFixture.api.ready;
      return window.workspaceFixture.api.refresh();
    });
  // Direct load at the admitted new route must show fresh intent entry.
  await page.goto(new URL('/workspace/new', page.url()).href);
  await page.waitForFunction(() => !!window.createWorkspaceFixture);
  await mountNewRoute();
  await page.getByRole('heading', {name: 'What do you want to build?', exact: true}).waitFor();
  assert.deepEqual(
    await page.evaluate(() => window.workspaceFixture.calls.filter((call) => call.method !== 'GET')),
    []
  );
  // Reload keeps the fresh entry without provisioning.
  await page.reload();
  await page.waitForFunction(() => !!window.createWorkspaceFixture);
  await mountNewRoute();
  await page.getByRole('heading', {name: 'What do you want to build?', exact: true}).waitFor();
  assert.deepEqual(
    await page.evaluate(() => window.workspaceFixture.calls.filter((call) => call.method !== 'GET')),
    []
  );
  // Back exits to the ordinary workspace with terminals/navigation preserved.
  await page.getByRole('button', {name: 'Back', exact: true}).click();
  await page.getByRole('navigation', {name: 'Projects', exact: true}).waitFor();
  assert.deepEqual(
    await page.evaluate(() => window.workspaceFixture.calls.filter((call) => call.method !== 'GET')),
    []
  );
  // Traversal back into the new route reactivates the entry.
  await page.goBack();
  await page.getByRole('heading', {name: 'What do you want to build?', exact: true}).waitFor();
});

test('setup: retained owner follows native forward to /workspace/new without remount', {timeout: 60000}, async (t) => {
  const page = await fixture(t);
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  await page.getByRole('navigation', {name: 'Projects', exact: true}).waitFor();
  // Native forward: Fountain pushState then panel-entry sync calls present()
  // on the retained owner; no popstate fires and nothing remounts.
  await page.evaluate(() => {
    window.history.pushState({}, '', '/workspace/new');
    (document.querySelector('soda-spaces') as unknown as {present(kind: 'page'): void}).present('page');
  });
  await page.getByRole('heading', {name: 'What do you want to build?', exact: true}).waitFor();
  assert.deepEqual(
    await page.evaluate(() => window.workspaceFixture.calls.filter((call) => call.method !== 'GET')),
    []
  );
});

test('setup: native New while resuming starts a blank entry and saves a new identity', {timeout: 90000}, async (t) => {
  const page = await fixture(t);
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  await page.getByRole('navigation', {name: 'Projects', exact: true}).waitFor();
  await page.getByRole('button', {name: 'New project', exact: true}).click();
  await page.getByRole('heading', {name: 'What do you want to build?', exact: true}).waitFor();
  await page.getByLabel('Describe what you want to build', {exact: true}).fill('Build a small project tracker');
  await page.getByRole('button', {name: 'Save draft', exact: true}).click();
  await page.waitForFunction(() => window.location.pathname.startsWith('/workspace/requests/d'));
  const savedRoute = await page.evaluate(() => window.location.pathname);
  await page.getByRole('button', {name: 'Save and leave', exact: true}).click();
  await page.waitForFunction(() => window.location.pathname === '/workspace');
  await resumeSavedPlan(page, 'Continue planning');
  await page.waitForFunction((route) => window.location.pathname === route, savedRoute);
  const resumedIntent = await page.evaluate(
    () =>
      (document.querySelector('soda-intent') as unknown as {revision?: {intent: string}} | null)?.revision?.intent ?? ''
  );
  assert.match(resumedIntent, /^d[0-9a-f]{24}$/);
  // Same retained owner takes the native forward; the element must survive.
  await page.evaluate(() => {
    document.querySelector('soda-spaces')?.setAttribute('data-c03-probe', 'retained');
    window.history.pushState({}, '', '/workspace/new');
    (document.querySelector('soda-spaces') as unknown as {present(kind: 'page'): void}).present('page');
  });
  await page.getByRole('heading', {name: 'What do you want to build?', exact: true}).waitFor();
  assert.equal(await page.getByLabel('Describe what you want to build', {exact: true}).inputValue(), '');
  assert.equal(
    await page.evaluate(
      () => (document.querySelector('soda-intent') as unknown as {revision?: unknown} | null)?.revision
    ),
    undefined
  );
  assert.equal(await page.evaluate(() => window.location.pathname), '/workspace/new');
  assert.equal(
    await page.evaluate(() => document.querySelector('soda-spaces')?.getAttribute('data-c03-probe')),
    'retained'
  );
  // Next save mints a new identity instead of extending the resumed one.
  await page.getByLabel('Describe what you want to build', {exact: true}).fill('Build a second tracker');
  await page.getByRole('button', {name: 'Save draft', exact: true}).click();
  await page.waitForFunction(
    (route) => window.location.pathname.startsWith('/workspace/requests/d') && window.location.pathname !== route,
    savedRoute
  );
  const posts = await page.evaluate(() =>
    window.workspaceFixture.calls.filter((call) => call.method === 'POST' && call.path.endsWith('/api/factory/intents'))
  );
  assert.equal(posts.length, 3);
  const fresh = posts[posts.length - 1];
  assert.equal(fresh?.body?.intent === resumedIntent, false);
  assert.equal(fresh?.body?.predecessor, undefined);
  const foreign = await page.evaluate(() =>
    window.workspaceFixture.calls.filter(
      (call) => call.method !== 'GET' && !(call.method === 'POST' && call.path.endsWith('/api/factory/intents'))
    )
  );
  assert.deepEqual(foreign, []);
});

test('setup: native New while viewing accepted starts blank and saves apart', {timeout: 120000}, async (t) => {
  const page = await fixture(t, 'page', undefined, true, '/forgejo.test');
  const prefix = '/forgejo.test';
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  await reviewFirstIntent(page);
  await page.getByRole('radio', {name: /alice\/Alpha/}).check();
  await page.getByRole('button', {name: 'Continue', exact: true}).click();
  await page.getByRole('button', {name: 'Prepare project', exact: true}).click();
  await page.getByRole('heading', {name: 'Configure project'}).waitFor();
  await completeBoundIntentPlanning(page);
  await page.getByRole('button', {name: 'Create project', exact: true}).click();
  await page.getByRole('heading', {name: 'Review your product direction', exact: true}).waitFor();
  await page.getByRole('button', {name: 'Back', exact: true}).click();
  await page.getByRole('button', {name: 'Terminals', exact: true}).click();
  await page.getByRole('button', {name: 'Join project', exact: true}).click();
  await page.getByRole('heading', {name: 'Open your first terminal'}).waitFor();
  await page.evaluate(() => {
    window.workspaceFixture.enableFactoryRun('5');
    const [space] = window.workspaceFixture.spaces;
    space?.factory_runs.push({
      id: 'd'.repeat(32),
      role: 'soda-coder',
      issue: '5',
      attempt: 'attempt-1',
      reconciled: false,
    });
    return window.workspaceFixture.api.refresh();
  });
  await resumeSavedPlan(page);
  const resumed = page.locator('soda-factory-work-overview');
  await resumed.getByText('Build a small project tracker').waitFor();
  await resumed.getByText('Native work generated.').waitFor();
  const acceptedRoute = await page.evaluate(() => window.location.pathname);
  assert.match(acceptedRoute, /^\/forgejo\.test\/workspace\/requests\/d[0-9a-f]{24}$/);
  assert.equal(
    await page.evaluate(
      () => (document.querySelector('soda-spaces') as unknown as {setupCommitted: boolean}).setupCommitted
    ),
    true
  );
  // Mount the live watch terminal, then return to the accepted review.
  await page.locator('soda-factory-work-detail').getByRole('button', {name: 'Watch output', exact: true}).click();
  await page.locator('.soda-factory.is-connected .xterm-rows').waitFor();
  await resumeSavedPlan(page);
  await resumed.getByText('Native work generated.').waitFor();
  const watchAlive = () =>
    page.evaluate(
      () =>
        document.querySelectorAll('.soda-factory .xterm-rows').length > 0 &&
        window.workspaceFixture.sockets.some(
          (socket) => String(socket.url).includes('/factory/runs/dddddddd') && !socket.closed
        )
    );
  assert.equal(await watchAlive(), true);
  const provisionedBefore = await page.evaluate(
    () =>
      window.workspaceFixture.calls.filter((call) => call.method === 'POST' && call.path.includes('/api/environments'))
        .length
  );
  // Same retained owner takes the explicit native New.
  await page.evaluate((route) => {
    document.querySelector('soda-spaces')?.setAttribute('data-c03-probe', 'retained');
    window.history.pushState({}, '', route);
    (document.querySelector('soda-spaces') as unknown as {present(kind: 'page'): void}).present('page');
  }, `${prefix}/workspace/new`);
  await page.getByRole('heading', {name: 'What do you want to build?', exact: true}).waitFor();
  assert.equal(await page.getByLabel('Describe what you want to build', {exact: true}).inputValue(), '');
  assert.equal(
    await page.evaluate(
      () => (document.querySelector('soda-intent') as unknown as {revision?: unknown} | null)?.revision
    ),
    undefined
  );
  assert.equal(await page.locator('soda-factory-work-overview').count(), 0);
  assert.equal(
    await page.evaluate(
      () => (document.querySelector('soda-spaces') as unknown as {setupCommitted: boolean}).setupCommitted
    ),
    false
  );
  assert.equal(await page.evaluate(() => window.location.pathname), `${prefix}/workspace/new`);
  assert.equal(
    await page.evaluate(() => document.querySelector('soda-spaces')?.getAttribute('data-c03-probe')),
    'retained'
  );
  assert.equal(await watchAlive(), true);
  // Next save mints a separate identity; nothing provisions.
  await page.getByLabel('Describe what you want to build', {exact: true}).fill('Build a second tracker');
  await page.getByRole('button', {name: 'Save draft', exact: true}).click();
  await page.waitForFunction(
    (route) =>
      window.location.pathname.startsWith('/forgejo.test/workspace/requests/d') && window.location.pathname !== route,
    acceptedRoute
  );
  const acceptedIntent = acceptedRoute.slice('/forgejo.test/workspace/requests/'.length);
  const posts = await page.evaluate(() =>
    window.workspaceFixture.calls.filter((call) => call.method === 'POST' && call.path.endsWith('/api/factory/intents'))
  );
  assert.ok(posts.length > 0);
  const fresh = posts[posts.length - 1];
  assert.equal(fresh?.body?.intent === acceptedIntent, false);
  assert.equal(fresh?.body?.predecessor, undefined);
  assert.equal(
    await page.evaluate(
      () =>
        window.workspaceFixture.calls.filter(
          (call) => call.method === 'POST' && call.path.includes('/api/environments')
        ).length
    ),
    provisionedBefore
  );
  // Back returns to the pre-resume target; the accepted work is intact.
  await page.getByRole('button', {name: 'Back', exact: true}).click();
  await resumeSavedPlan(page);
  await page.waitForFunction((route) => window.location.pathname === route, acceptedRoute);
  await resumed.getByText('Native work generated.').waitFor();
});

test('setup: native New on idle Configure starts blank and saves apart', {timeout: 90000}, async (t) => {
  const page = await fixture(t, 'page', undefined, true, '/forgejo.test');
  const prefix = '/forgejo.test';
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  await reviewFirstIntent(page);
  await page.getByRole('radio', {name: /alice\/Alpha/}).check();
  await page.getByRole('button', {name: 'Continue', exact: true}).click();
  await page.getByRole('button', {name: 'Prepare project', exact: true}).click();
  await page.getByRole('heading', {name: 'Configure project'}).waitFor();
  await completeBoundIntentPlanning(page);
  const acceptedIntent = await page.evaluate(
    () =>
      (document.querySelector('soda-intent') as unknown as {revision?: {intent: string}} | null)?.revision?.intent ?? ''
  );
  assert.match(acceptedIntent, /^d[0-9a-f]{24}$/);
  assert.equal(
    await page.evaluate(() => (document.querySelector('soda-spaces') as unknown as {setup: string}).setup),
    'configure'
  );
  // Explicit native New before any Create operation takes the same owner.
  await page.evaluate((route) => {
    window.history.pushState({}, '', route);
    (document.querySelector('soda-spaces') as unknown as {present(kind: 'page'): void}).present('page');
  }, `${prefix}/workspace/new`);
  await page.getByRole('heading', {name: 'What do you want to build?', exact: true}).waitFor();
  assert.equal(await page.getByLabel('Describe what you want to build', {exact: true}).inputValue(), '');
  assert.equal(
    await page.evaluate(
      () => (document.querySelector('soda-intent') as unknown as {revision?: unknown} | null)?.revision
    ),
    undefined
  );
  assert.equal(
    await page.evaluate(() => (document.querySelector('soda-spaces') as unknown as {setup: string}).setup),
    'intent'
  );
  assert.equal(await page.evaluate(() => window.location.pathname), `${prefix}/workspace/new`);
  // Next save mints a separate identity; the accepted request is preserved.
  await page.getByLabel('Describe what you want to build', {exact: true}).fill('Build a second tracker');
  await page.getByRole('button', {name: 'Save draft', exact: true}).click();
  await page.waitForFunction(
    (intent) =>
      window.location.pathname.startsWith('/forgejo.test/workspace/requests/d') &&
      !window.location.pathname.endsWith(intent),
    acceptedIntent
  );
  const posts = await page.evaluate(() =>
    window.workspaceFixture.calls.filter((call) => call.method === 'POST' && call.path.endsWith('/api/factory/intents'))
  );
  assert.ok(posts.length > 0);
  const fresh = posts[posts.length - 1];
  assert.equal(fresh?.body?.intent === acceptedIntent, false);
  assert.equal(fresh?.body?.predecessor, undefined);
  await page.getByRole('button', {name: 'Back', exact: true}).click();
  const resumeName = /^Resume Build a small project tracker$/;
  await page.getByRole('button', {name: resumeName}).waitFor();
  await page.getByRole('button', {name: resumeName}).click();
  await page.waitForFunction(
    (intent) => window.location.pathname === `/forgejo.test/workspace/requests/${intent}`,
    acceptedIntent
  );
  await page.locator('soda-factory-work-overview').getByText('Build a small project tracker').waitFor();
});

test('setup: unknown request route reports unavailable without losing the entry', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  await page.getByRole('heading', {name: 'What do you want to build?', exact: true}).waitFor();
  await page.evaluate(
    (missing) => {
      window.history.pushState({}, '', missing);
      window.dispatchEvent(new PopStateEvent('popstate'));
    },
    '/workspace/requests/d' + '3'.repeat(24)
  );
  await page
    .getByText('This saved request is unavailable to the current account. Your current edits are kept.', {exact: true})
    .waitFor();
  await page.getByRole('heading', {name: 'What do you want to build?', exact: true}).waitFor();
});

test('setup: saving a draft syncs the request route without leaving the flow', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  await page.getByRole('heading', {name: 'What do you want to build?', exact: true}).waitFor();
  await page.getByLabel('Describe what you want to build', {exact: true}).fill('Build a small project tracker');
  await page.getByRole('button', {name: 'Save draft', exact: true}).click();
  await page.waitForFunction(() => window.location.pathname.startsWith('/workspace/requests/d'));
  assert.match(await page.evaluate(() => window.location.pathname), /^\/workspace\/requests\/d[0-9a-f]{24}$/);
  await page.getByRole('heading', {name: 'What do you want to build?', exact: true}).waitFor();
});

test('setup: new entry canonicalizes the route and leave returns', async (t) => {
  const page = await fixture(t, 'page', undefined, false);
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  await page.locator('.soda-create-another-project:visible').waitFor();
  await page.getByRole('button', {name: 'New project', exact: true}).click();
  await page.getByRole('heading', {name: 'What do you want to build?', exact: true}).waitFor();
  assert.equal(await page.evaluate(() => window.location.pathname), '/workspace/new');
  await page.getByLabel('Describe what you want to build', {exact: true}).fill('Build a small project tracker');
  await page.getByRole('button', {name: 'Save and leave', exact: true}).click();
  await page.getByRole('heading', {name: 'What do you want to build?', exact: true}).waitFor({state: 'hidden'});
});

test('setup: discovery groups saved drafts from inventory and resumes them', {timeout: 20000}, async (t) => {
  const page = await fixture(t, 'page', undefined, false);
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  const heading = page.getByRole('heading', {name: 'What do you want to build?', exact: true});
  const begin = async () => {
    await page.locator('.soda-create-another-project:visible').waitFor();
    await page.getByRole('button', {name: 'New project', exact: true}).click();
    await heading.waitFor();
  };
  const saveAndLeave = async (objective: string) => {
    await page.getByLabel('Describe what you want to build', {exact: true}).fill(objective);
    await page.getByRole('button', {name: 'Save draft', exact: true}).click();
    await page.getByText('Saved', {exact: true}).waitFor();
    const route = await page.evaluate(() => window.location.pathname);
    await page.getByRole('button', {name: 'Save and leave', exact: true}).click();
    await heading.waitFor({state: 'hidden'});
    return route;
  };
  await begin();
  await saveAndLeave('Build discovery alpha');
  await begin();
  const saved = await saveAndLeave('Build discovery beta');
  assert.match(saved, /^\/workspace\/requests\/d[0-9a-f]{24}$/);
  await begin();
  await page.locator('soda-intent').getByText('Build discovery beta', {exact: true}).waitFor();
  await page.getByRole('button', {name: 'Back', exact: true}).click();
  await heading.waitFor({state: 'hidden'});
  await page.getByRole('button', {name: 'Saved plans', exact: true}).click();
  await page.locator('.soda-work-item').first().waitFor();
  assert.equal(await page.locator('.soda-work-state[data-state="draft"]').count(), 2);
  await page.getByText('Build discovery alpha', {exact: false}).waitFor();
  await page.getByText('Not linked to a repository', {exact: true}).first().waitFor();
  await page.getByRole('button', {name: 'Continue planning: Build discovery beta', exact: true}).click();
  await page.waitForFunction(() => window.location.pathname.startsWith('/workspace/requests/d'));
  assert.equal(await page.evaluate(() => window.location.pathname), saved);
  await heading.waitFor();
  await page.waitForFunction(() => {
    const area = document.querySelector('soda-intent')?.shadowRoot?.querySelector('textarea');
    return area instanceof HTMLTextAreaElement && area.value === 'Build discovery beta';
  });
});
test('first use: picker notice keeps a stable silent wrapper and borders only real messages', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  await reviewFirstIntent(page);
  await page.getByRole('heading', {name: 'Choose a repository'}).waitFor();
  await page.getByRole('radio', {name: /alice\/Alpha/}).waitFor();
  const notice = page.locator('.soda-repository-notice');
  // Idle: the live-region wrapper persists for first announcements but takes
  // no space and shows no border.
  assert.equal(await notice.count(), 1);
  assert.equal(await notice.getAttribute('role'), 'status');
  assert.equal(await notice.locator('.soda-feedback').count(), 0);
  const idle = await notice.evaluate((node) => ({
    height: node.clientHeight,
    border: getComputedStyle(node).borderTopWidth,
    same: document.querySelector('.soda-repository-notice') === node,
  }));
  assert.equal(idle.height, 0);
  assert.equal(idle.border, '0px');
  assert.equal(idle.same, true);
  const idleHandle = await notice.elementHandle();
  assert.ok(idleHandle);
  // Gate repository search to observe the busy copy, then release it.
  await page.evaluate(() => {
    const pristine = window.fetch.bind(window);
    let release!: () => void;
    const gate = new Promise<void>((resolve) => {
      release = resolve;
    });
    (window as unknown as {releaseRepositorySearch: () => void}).releaseRepositorySearch = () => release();
    Object.defineProperty(window, 'fetch', {
      configurable: true,
      value: (input: RequestInfo | URL, init?: RequestInit) => {
        if (String(input).includes('/api/repositories?')) return gate.then(() => pristine(input, init));
        return pristine(input, init);
      },
    });
  });
  await page.getByPlaceholder('Search repositories…').fill('beta');
  await page.getByRole('button', {name: 'Search', exact: true}).click();
  await page.getByText('Finding repositories…').waitFor();
  const busyNotice = page.locator('.soda-repository-notice');
  assert.equal(await busyNotice.count(), 1);
  assert.equal(await busyNotice.locator('.soda-feedback').count(), 1);
  assert(await busyNotice.locator('.soda-feedback').isVisible());
  assert.equal(
    await page.evaluate((element) => document.querySelector('.soda-repository-notice') === element, idleHandle),
    true
  );
  await page.evaluate(() => (window as unknown as {releaseRepositorySearch: () => void}).releaseRepositorySearch());
  await page.getByRole('radio', {name: /alice\/Beta/}).waitFor();
  // Released: the same wrapper persists with no bordered child.
  assert.equal(await busyNotice.locator('.soda-feedback').count(), 0);
  assert.equal(
    await page.evaluate((element) => document.querySelector('.soda-repository-notice') === element, idleHandle),
    true
  );
  // A failed search surfaces the error copy in the same notice region.
  await page.evaluate(() => {
    const pristine = window.fetch.bind(window);
    Object.defineProperty(window, 'fetch', {
      configurable: true,
      value: (input: RequestInfo | URL, init?: RequestInit) => {
        if (String(input).includes('/api/repositories?')) return Promise.resolve(new Response(null, {status: 500}));
        return pristine(input, init);
      },
    });
  });
  await page.getByPlaceholder('Search repositories…').fill('zzz-no-match');
  await page.getByRole('button', {name: 'Search', exact: true}).click();
  await page.getByText('Could not find repositories. Search again; no project was created.').waitFor();
  const errorNotice = page.locator('.soda-repository-notice');
  assert.equal(await errorNotice.count(), 1);
  assert(await errorNotice.locator('.soda-feedback').isVisible());
  assert.equal(await errorNotice.locator('.soda-feedback').getAttribute('data-tone'), 'warning');
  assert.equal(
    await page.evaluate((element) => document.querySelector('.soda-repository-notice') === element, idleHandle),
    true
  );
});
