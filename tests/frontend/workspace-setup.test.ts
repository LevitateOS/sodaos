import test from 'node:test';
import assert from 'node:assert/strict';
import type {Page} from 'playwright';
import {
  setupWorkspaceDriver,
  chooseFirstRepository,
  reviewFirstIntent,
  fixture,
  textContents,
} from './fixtures/workspace-driver';
import {captureSpacesComponent} from '../../scripts/screenshot';
setupWorkspaceDriver();

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
  await page.getByRole('button', {name: 'Create project', exact: true}).click();
  await page.getByRole('button', {name: 'Creating project…', exact: true}).waitFor();
  for (const name of ['Back', 'Change repository', 'Cancel setup', 'Creating project…'])
    assert(await page.getByRole('button', {name, exact: true}).isDisabled(), name);
  assert.equal(await page.locator('.soda-config-repository-name').innerText(), 'alice/Alpha\nForgejo');
  await captureSpacesComponent(page, 'pending-creation');
  await page.evaluate(() => window.dispatchEvent(new Event('release-creation')));
  await page.getByRole('button', {name: 'Join project', exact: true}).waitFor();
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
    await page
      .getByText(
        outcome === 'rejected'
          ? /Installed Project OS unavailable\. No reservation/
          : /Project creation could not be confirmed/
      )
      .waitFor();
    await page.getByRole('button', {name: 'Refresh status', exact: true}).click();
    if (outcome === 'uncertain') await page.getByRole('button', {name: 'Join project', exact: true}).waitFor();
    else if (outcome === 'incomplete') await page.getByRole('heading', {name: 'Project needs inspection'}).waitFor();
    else await page.getByRole('button', {name: 'Create project', exact: true}).waitFor();
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
  await page.getByRole('heading', {name: 'Configure project'}).waitFor();
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
});
test('first use: picker summarizes reviewed authorization and the create effect', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  await reviewFirstIntent(page);
  await page.getByRole('heading', {name: 'Choose a repository'}).waitFor();
  await page.getByText('Saving 1 authorized idea and 1 work item to the chosen repository.').waitFor();
  await page.getByText('Continuing creates factory work for that repository.').waitFor();
  const create = page.getByRole('link', {name: 'Create a new repository'});
  assert.equal(await create.getAttribute('target'), '_blank');
  assert.equal(await create.getAttribute('rel'), 'noopener');
});
test('first use: resumed accepted request suggests its repository in the picker', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  await reviewFirstIntent(page);
  await page.getByRole('radio', {name: /alice\/Alpha/}).check();
  await page.getByRole('button', {name: 'Continue', exact: true}).click();
  await page.getByRole('heading', {name: 'Configure project'}).waitFor();
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
  await page.getByRole('button', {name: 'Choose repository', exact: true}).click();
  await page.getByRole('heading', {name: 'Choose a repository'}).waitFor();
  await page.getByText('Showing the suggested repository for this request.', {exact: true}).waitFor();
});
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
  await page.getByRole('button', {name: 'Projects'}).click();
  await page.getByRole('button', {name: 'alice/Beta', exact: true}).click();
  await page.getByRole('button', {name: 'Project settings'}).click();
  await page.getByRole('heading', {name: 'Project OS'}).waitFor();
  await page.getByRole('button', {name: 'Projects'}).click();
  await reviewFirstIntent(page);
  await page.getByRole('radio', {name: /alice\/Gamma/}).waitFor();
  await page.getByRole('radio', {name: /alice\/Gamma/}).check();
  await page.getByRole('button', {name: 'Continue', exact: true}).click();
  await page.getByRole('button', {name: 'Cancel setup', exact: true}).waitFor();
  await page.getByRole('button', {name: 'Cancel setup', exact: true}).click();
  await page.getByRole('heading', {name: 'Project OS'}).waitFor();
  await page.getByRole('button', {name: 'Projects'}).click();
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
  await page.getByRole('button', {name: 'Projects'}).click();
  await page.getByRole('button', {name: 'alice/Beta', exact: true}).click();
  await page.getByRole('button', {name: 'Projects'}).click();
  await reviewFirstIntent(page);
  await page.getByRole('radio', {name: /alice\/Gamma/}).waitFor();
  await page.getByRole('radio', {name: /alice\/Gamma/}).check();
  await page.getByRole('button', {name: 'Continue', exact: true}).click();
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
  await page.getByRole('button', {name: 'Projects'}).click();
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
  await page.getByRole('button', {name: 'Create project', exact: true}).click();
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
  await page.waitForFunction(() => window.location.pathname === '/workspace/new');
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
  await page.getByRole('button', {name: 'Create project', exact: true}).click();
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
    await page.getByRole('button', {name: 'Create project', exact: true}).click();
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
  await page.getByText('Drafts (2)', {exact: true}).waitFor();
  await page.getByText('Build discovery alpha', {exact: false}).waitFor();
  await page.getByText('No repository yet', {exact: false}).first().waitFor();
  await page.getByRole('button', {name: 'Resume Build discovery beta', exact: true}).click();
  await page.waitForFunction(() => window.location.pathname.startsWith('/workspace/requests/d'));
  assert.equal(await page.evaluate(() => window.location.pathname), saved);
  await heading.waitFor();
  await page.waitForFunction(() => {
    const area = document.querySelector('soda-intent')?.shadowRoot?.querySelector('textarea');
    return area instanceof HTMLTextAreaElement && area.value === 'Build discovery beta';
  });
});
