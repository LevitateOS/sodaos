import test, {before, after} from 'node:test';
import assert from 'node:assert/strict';
import {chromium, type Browser} from 'playwright';
import type {} from './fixtures/application-fixture';
import {captureSpacesComponent} from '../../scripts/screenshot';

let browser: Browser, server: ReturnType<typeof Bun.serve>;
before(async () => {
  const result = await Bun.build({
    entrypoints: [import.meta.dir + '/fixtures/application-fixture.ts'],
    target: 'browser',
    format: 'esm',
    define: {'process.env.NODE_ENV': '"production"'},
    plugins: [
      {
        name: 'terminal-vendor',
        setup(build) {
          build.onResolve({filter: /^\.\/soda-terminal\/(xterm|addon-fit)\.mjs$/}, (args) => ({
            path: args.path,
            external: true,
          }));
        },
      },
    ],
  });
  assert(result.success, result.logs.join('\n'));
  const bundle = result.outputs[0];
  assert(bundle);
  server = Bun.serve({
    hostname: '127.0.0.1',
    port: 0,
    fetch(request) {
      const url = new URL(request.url);
      if (url.pathname === '/fixture.js') return new Response(bundle, {headers: {'Content-Type': 'text/javascript'}});
      // The same production stylesheets the workspace driver loads, so the
      // App surface renders with actual product styles.
      const local =
        url.pathname === '/assets/sodaspaces-drawer.css'
          ? 'frontend/spaces/sodaspaces-project.css'
          : url.pathname === '/assets/sodaspaces-page.css'
            ? 'frontend/spaces/sodaspaces-workspace.css'
            : url.pathname === '/assets/sodaspaces-factory-conversation.css'
              ? 'frontend/spaces/sodaspaces-factory-conversation.css'
              : url.pathname === '/assets/sodaspaces-terminal.css'
                ? 'frontend/spaces/sodaspaces-terminal.css'
                : url.pathname.startsWith('/assets/soda/forgejo/icons/')
                  ? 'assets/branding/icons/octicons/' + url.pathname.split('/').pop()
                  : url.pathname.startsWith('/assets/soda/')
                    ? 'assets/branding/' + url.pathname.slice('/assets/soda/'.length)
                    : '';
      if (local)
        return new Response(Bun.file(`${import.meta.dir}/../../${local}`), {
          headers: {
            'Content-Type': /\.m?js$/.test(local)
              ? 'text/javascript'
              : local.endsWith('.css')
                ? 'text/css'
                : local.endsWith('.svg')
                  ? 'image/svg+xml'
                  : 'application/octet-stream',
          },
        });
      return new Response(
        '<!doctype html><meta name="soda-component-fixture" content="spaces"><script>document.documentElement.dataset.theme="soda-auto"</script><link rel="stylesheet" href="/assets/sodaspaces-drawer.css"><link rel="stylesheet" href="/assets/sodaspaces-page.css"><link rel="stylesheet" href="/assets/sodaspaces-terminal.css"><link rel="stylesheet" href="/assets/soda/forgejo/components.css"><link rel="stylesheet" href="/assets/soda/forgejo/components-buttons.css"><main></main><script type="module" src="/fixture.js"></script>',
        {
          headers: {'Content-Type': 'text/html'},
        }
      );
    },
  });
  browser = await chromium.launch({headless: true, chromiumSandbox: true});
});
after(async () => {
  await browser?.close();
  server?.stop(true);
});

test('Opening Applications loads current status, fences double Start and requires confirmed Stop', async (t) => {
  const page = await browser.newPage();
  t.after(() => page.close());
  await page.goto(server.url.href);
  await page.waitForFunction(() => typeof window.createApplicationFixture === 'function');
  await page.evaluate(() => window.createApplicationFixture());
  await page.getByText('Applications', {exact: true}).click();
  await page.getByText('Last observed: Stopped', {exact: true}).waitFor();
  await page
    .getByText(
      'Accounts and saved data live with this application at its address, separate from Soda Forge. An operator manages its setup and updates.',
      {exact: true}
    )
    .waitFor();
  await page.waitForFunction(() => window.applicationFixture.calls.some((c) => c.body?.action === 'inspect'));
  assert.equal(
    await page.evaluate(
      () => window.applicationFixture.calls.filter((c) => c.body && c.body.action !== 'inspect').length
    ),
    0
  );
  assert(await page.getByRole('button', {name: 'Start Example', exact: true}).isVisible());
  await page.evaluate(() => {
    const button = Array.from(document.querySelectorAll('button')).find(
      (b) => b.textContent?.trim() === 'Start Example'
    );
    assertButton(button);
    button.click();
    button.click();
    function assertButton(value: HTMLButtonElement | undefined): asserts value is HTMLButtonElement {
      if (!value) throw Error('missing start');
    }
  });
  await page.getByText('Last observed: Ready', {exact: true}).waitFor();
  for (const theme of ['light', 'dark'])
    for (const width of [1440, 800, 390]) {
      await page.setViewportSize({width, height: 900});
      await page.evaluate((value) => {
        document.documentElement.style.colorScheme = value;
      }, theme);
      await captureSpacesComponent(page, `app-readiness-${theme}-${width}`);
    }
  await page.setViewportSize({width: 1280, height: 720});
  await page.evaluate(() => {
    document.documentElement.style.colorScheme = 'light';
  });
  const writes = await page.evaluate(() =>
    window.applicationFixture.calls.filter((c) => c.body && c.body.action !== 'inspect')
  );
  assert.equal(writes.length, 1);
  assert.deepEqual(writes[0]?.body, {
    action: 'start',
    revision: 1,
    expected_container: 'a'.repeat(64),
    confirm_stop: false,
  });
  await page.getByText('Start or stop application', {exact: true}).click();
  assert(await page.getByRole('button', {name: 'Confirm Stop Example', exact: true}).isDisabled());
  await page.getByRole('checkbox', {name: 'Stop Example for everyone', exact: true}).check();
  await page.evaluate(() => {
    window.applicationFixture.fail = true;
  });
  await page.getByRole('button', {name: 'Confirm Stop Example', exact: true}).click();
  await page.getByText('Outcome unconfirmed. Inspect the application before acting again.', {exact: true}).waitFor();
  assert.equal(await page.getByText('private diagnostic', {exact: true}).count(), 0);
  assert(await page.getByRole('button', {name: 'Start Example', exact: true}).isDisabled());
  assert.equal(await page.getByRole('link', {name: /Open application/}).count(), 1);
});

test('suspended or replaced session discards pending application data without replaying an action', async (t) => {
  const page = await browser.newPage();
  t.after(() => page.close());
  await page.goto(server.url.href);
  await page.waitForFunction(() => typeof window.createApplicationFixture === 'function');
  await page.evaluate(() => window.createApplicationFixture());
  await page.getByText('Applications', {exact: true}).click();
  await page.getByText('Last observed: Stopped', {exact: true}).waitFor();
  await page.waitForFunction(() => window.applicationFixture.calls.some((c) => c.body?.action === 'inspect'));
  await page.evaluate(() => {
    window.applicationFixture.delay = true;
  });
  await page.getByRole('button', {name: 'Refresh applications', exact: true}).click();
  await page.evaluate(async () => {
    const f = window.applicationFixture;
    f.element.active = false;
    f.element.sessionKey = 'generation-2';
    await f.element.updateComplete;
    f.release?.();
  });
  await page.waitForTimeout(50);
  assert.equal(await page.getByText('https://app.example.test', {exact: true}).count(), 0);
  assert.equal(
    await page.evaluate(
      () => window.applicationFixture.calls.filter((c) => c.body && c.body.action !== 'inspect').length
    ),
    0
  );
  assert(await page.getByRole('button', {name: 'Refresh applications', exact: true}).isDisabled());
});

test('App distinguishes unavailable, loading and empty inventory and offers read-only Feed navigation', async (t) => {
  const page = await browser.newPage();
  t.after(() => page.close());
  await page.goto(server.url.href);
  await page.waitForFunction(() => typeof window.createApplicationFixture === 'function');
  await page.evaluate(async () => {
    await window.createApplicationFixture();
    const fixture = window.applicationFixture;
    fixture.element.request = async (path, body) => {
      fixture.calls.push({path, ...(body ? {body} : {})});
      throw Error('unavailable inventory');
    };
    fixture.element.repository = '7';
    fixture.element.expanded = true;
  });
  await page
    .getByText('Applications could not be loaded. Reconnect to Soda Forge, then refresh applications.', {exact: true})
    .waitFor();
  assert.equal(await page.getByText(/No application is available to you for this project/).count(), 0);
  const statusRegion = () =>
    page.evaluate(() => {
      // SodaApplications renders into light DOM (createRenderRoot returns
      // this), so query the element/renderRoot, never shadowRoot.
      const root = window.applicationFixture.element.renderRoot ?? window.applicationFixture.element;
      const node = root?.querySelector('.soda-applications-status');
      if (!node) throw Error('Missing applications status region');
      const child = node.querySelector('p');
      const style = getComputedStyle(node);
      return {
        role: node.getAttribute('role'),
        hidden: (node as HTMLElement).hidden,
        display: style.display,
        childRoles: [...node.querySelectorAll('[role]')].map((el) => el.getAttribute('role')),
        text: child?.textContent ?? '',
      };
    });
  const failedStatus = await statusRegion();
  assert.equal(failedStatus.role, 'status');
  assert.equal(failedStatus.hidden, false);
  assert.notEqual(failedStatus.display, 'none');
  assert.deepEqual(failedStatus.childRoles, []);
  assert.match(failedStatus.text, /could not be loaded/);
  const statusHandle = await page.evaluateHandle(() =>
    (window.applicationFixture.element.renderRoot ?? window.applicationFixture.element)?.querySelector(
      '.soda-applications-status'
    )
  );
  assert.ok(statusHandle);
  const navigation = page.evaluate(
    () =>
      new Promise<unknown>((resolve) => {
        window.addEventListener('soda-project-feed-open', (event) => resolve((event as CustomEvent).detail), {
          once: true,
        });
      })
  );
  await page.getByRole('button', {name: 'View project Feed', exact: true}).click();
  assert.deepEqual(await navigation, {repository: '7'});
  await page.evaluate(() => {
    const fixture = window.applicationFixture;
    fixture.element.request = async (path, body) => {
      fixture.calls.push({path, ...(body ? {body} : {})});
      return new Promise((resolve) => {
        fixture.release = () => resolve({applications: [], complete: true, can_setup: false});
      });
    };
  });
  await page.getByText('Loading applications…', {exact: true}).waitFor();
  assert.equal(await page.getByText(/No application is available to you for this project/).count(), 0);
  const loadingStatus = await statusRegion();
  assert.equal(loadingStatus.role, 'status');
  assert.deepEqual(loadingStatus.childRoles, []);
  assert.match(loadingStatus.text, /Loading applications/);
  assert.equal(
    await page.evaluate(
      (element) =>
        (window.applicationFixture.element.renderRoot ?? window.applicationFixture.element)?.querySelector(
          '.soda-applications-status'
        ) === element,
      statusHandle
    ),
    true
  );
  await page.evaluate(() => window.applicationFixture.release?.());
  await page.getByText(/No application is available to you for this project yet/).waitFor();
  await page
    .getByText(
      'No application is available to you for this project yet. Check the project Feed for updates or ask an application administrator about access.',
      {exact: true}
    )
    .waitFor();
  assert.equal(await page.locator('soda-application-surface').count(), 0);
  assert.equal(await page.evaluate(() => window.applicationFixture.calls.some((call) => !!call.body)), false);
  const emptyStatus = await statusRegion();
  assert.equal(emptyStatus.role, 'status');
  assert.equal(emptyStatus.hidden, false);
  assert.notEqual(emptyStatus.display, 'none');
  assert.deepEqual(emptyStatus.childRoles, []);
  assert.equal(emptyStatus.text, '');
  assert.equal(
    await page.evaluate(
      (element) =>
        (window.applicationFixture.element.renderRoot ?? window.applicationFixture.element)?.querySelector(
          '.soda-applications-status'
        ) === element,
      statusHandle
    ),
    true
  );
  await statusHandle.dispose();
});

test('operator backup and Restore require explicit actions and reconcile the durable operation after a lost reply', async (t) => {
  const page = await browser.newPage();
  t.after(() => page.close());
  await page.goto(server.url.href);
  await page.waitForFunction(() => typeof window.createApplicationFixture === 'function');
  await page.evaluate(() => window.createApplicationFixture(true));
  await page.getByText('Applications', {exact: true}).click();
  await page.waitForFunction(() => window.applicationFixture.calls.some((c) => c.body?.action === 'inspect'));
  await page.getByText('Backups and recovery', {exact: true}).click();
  assert.equal(
    await page.evaluate(
      () =>
        window.applicationFixture.calls.filter((c) => c.path.endsWith('/snapshots') || c.path.endsWith('/restores'))
          .length
    ),
    0
  );
  await page.getByRole('button', {name: 'Back up Example', exact: true}).click();
  await page.getByText('Stopped backup saved. Start the application when you are ready.', {exact: true}).waitFor();
  await page.getByRole('button', {name: 'Refresh backups for Example', exact: true}).click();
  await page.getByLabel('Backup for Example', {exact: true}).selectOption('a' + '8'.repeat(24));
  assert(await page.getByRole('button', {name: 'Restore Example', exact: true}).isDisabled());
  await page.getByRole('checkbox', {name: 'Restore Example to this backup', exact: true}).check();
  await page.evaluate(() => {
    window.applicationFixture.failRestore = true;
  });
  await page.getByRole('button', {name: 'Restore Example', exact: true}).click();
  await page
    .getByText(
      'Restore outcome unconfirmed. Refresh applications and reconcile the same operation before acting again.',
      {exact: true}
    )
    .waitFor();
  await page.getByRole('button', {name: 'Refresh applications', exact: true}).click();
  await page
    .getByText('Restore is pending. Reconcile this operation before starting the application.', {exact: true})
    .waitFor();
  assert(await page.getByRole('button', {name: 'Start Example', exact: true}).isDisabled());
  assert(await page.getByRole('button', {name: 'Inspect Example', exact: true}).isDisabled());
  await page.getByRole('button', {name: 'Reconcile Restore Example', exact: true}).click();
  await page
    .getByText('Restored to fresh data at the same address. Start the application when you are ready.', {exact: true})
    .waitFor();
  const restores = await page.evaluate(() =>
    window.applicationFixture.calls.filter((c) => c.path.endsWith('/restores'))
  );
  assert.equal(restores.length, 2);
  assert.deepEqual(restores[0]?.body, restores[1]?.body);
  assert.equal(restores[0]?.body?.snapshot, 'a' + '8'.repeat(24));
  assert.equal(
    await page.getByRole('link', {name: /Open application/}).getAttribute('href'),
    'https://app.example.test'
  );
  assert.equal(await page.getByText('lost reply', {exact: true}).count(), 0);
});

test('operator first-owner setup uses an ephemeral app link and finishes the same provisional generation', async (t) => {
  const page = await browser.newPage();
  t.after(() => page.close());
  const errors: string[] = [];
  page.on('pageerror', (error) => errors.push(error.message));
  await page.goto(server.url.href);
  await page.waitForFunction(() => typeof window.createApplicationFixture === 'function');
  await page.evaluate(() => window.createApplicationFixture(true));
  await page.getByText('Applications', {exact: true}).click();
  await page.waitForFunction(() => window.applicationFixture.calls.some((c) => c.body?.action === 'inspect'));
  assert(
    await page.evaluate(() => {
      const app = document.querySelector('soda-application-surface'),
        administration = document.querySelector('.soda-application-administration');
      return !!(
        app &&
        administration instanceof HTMLDetailsElement &&
        !administration.open &&
        app.compareDocumentPosition(administration) & Node.DOCUMENT_POSITION_FOLLOWING
      );
    })
  );
  await page.getByText('Prepare or update an application', {exact: true}).click();
  await page.getByText('Set up a new application (operator)', {exact: true}).click();
  await page.getByLabel('Admitted application references', {exact: true}).fill(
    JSON.stringify({
      name: 'NewApp',
      artifact_sha256: '1'.repeat(64),
      bundle_sha256: '2'.repeat(64),
      tools_sha256: '3'.repeat(64),
      runtime_image: 'sha256:' + '4'.repeat(64),
      image: 'sha256:' + '5'.repeat(64),
      origin: 'https://new.example.test',
      port: 24000,
    })
  );
  await page.getByLabel('Soda administrator user ID', {exact: true}).fill('41');
  await page.getByRole('button', {name: 'Begin owner setup', exact: true}).click();
  const link = page.getByRole('link', {name: 'Create application owner ↗', exact: true});
  await link.waitFor();
  assert.equal(
    await link.getAttribute('href'),
    'https://new.example.test/setup#operation=a' + 'c'.repeat(24) + '&token=' + 'd'.repeat(64)
  );
  assert.equal(await link.getAttribute('rel'), 'noopener noreferrer');
  const create = await page.evaluate(() =>
    window.applicationFixture.calls.find((c) => c.path === '/api/applications/setups' && c.body)
  );
  assert(create?.body);
  assert.equal(create.body.administrator_id, '41');
  assert.equal('owner_team' in (create.body.application as Record<string, unknown>), false);
  assert.match(String((create.body.application as Record<string, unknown>).data), /^a[0-9a-f]{24}$/);
  await page.evaluate(() => {
    window.applicationFixture.setupExpiresAt = Math.floor(Date.now() / 1000) - 1;
  });
  await page.getByRole('button', {name: 'Refresh owner setups', exact: true}).click();
  await page.getByText(/The link has expired/).waitFor();
  assert.equal(await link.count(), 0);
  assert(await page.getByRole('button', {name: 'Open owner setup NewApp', exact: true}).isDisabled());
  assert(await page.getByRole('button', {name: 'Inspect owner setup NewApp', exact: true}).isEnabled());
  await page.getByRole('button', {name: 'Confirm owner and finish NewApp', exact: true}).click();
  await page
    .getByText(
      'Owner confirmed. Setup complete. Refresh applications to reconcile the completed setup before starting it.',
      {
        exact: true,
      }
    )
    .waitFor();
  const completed = await page.evaluate(() =>
    window.applicationFixture.calls.find((c) => c.path.endsWith('/actions') && c.body?.action === 'complete')
  );
  assert.deepEqual(completed?.body, {action: 'complete', expected_container: 'c'.repeat(64), confirm: true});
  assert.deepEqual(errors, []);
});

test('lost reply after Apply adopts the completed replacement without a second apply', async (t) => {
  const page = await browser.newPage();
  t.after(() => page.close());
  await page.goto(server.url.href);
  await page.waitForFunction(() => typeof window.createApplicationDeliveryFixture === 'function');
  await page.evaluate(() => window.createApplicationDeliveryFixture(true));
  await page.getByRole('button', {name: 'Prepare company version', exact: true}).click();
  const apply = page.getByRole('button', {name: 'Apply version to Example', exact: true});
  await apply.waitFor();
  await page.getByRole('checkbox', {name: 'Stop Example for everyone and apply this version', exact: true}).check();
  await page.evaluate(() => {
    const app = window.applicationFixture.element,
      delivery = document.querySelector('soda-application-delivery') as unknown as {
        request: typeof app.request;
      },
      original = app.request!;
    const losing: typeof app.request = async (path, body, signal) => {
      if (body?.action === 'replace') {
        // The server applies the replacement but the reply is lost.
        await original(path, body, signal);
        throw Error('lost reply');
      }
      return original(path, body, signal);
    };
    app.request = losing;
    delivery.request = losing;
  });
  const appReadsBefore = await page.evaluate(
    () => window.applicationFixture.calls.filter((call) => !call.body && call.path === '/api/applications').length
  );
  const buildReadsBefore = await page.evaluate(
    () =>
      window.applicationFixture.calls.filter(
        (call) => !call.body && call.path.startsWith('/api/repositories/7/application-builds')
      ).length
  );
  await apply.click();
  await page
    .getByText(
      'The apply reply was unconfirmed, so the same application was re-read: the new version is installed and stopped. Start the company application, then check its retained records.',
      {exact: true}
    )
    .waitFor();
  // The adopted replacement is shown as admitted once the parent refresh lands;
  // only then are the reconcile read counts stable.
  await page.getByText('This version is already admitted in this environment.', {exact: true}).waitFor();
  // The same-apply reconcile read runs against application state, never the
  // builds list: one reconcile read plus the parent refresh it triggers.
  assert.equal(
    await page.evaluate(
      () => window.applicationFixture.calls.filter((call) => !call.body && call.path === '/api/applications').length
    ),
    appReadsBefore + 2
  );
  assert.equal(
    await page.evaluate(
      () =>
        window.applicationFixture.calls.filter(
          (call) => !call.body && call.path.startsWith('/api/repositories/7/application-builds')
        ).length
    ),
    buildReadsBefore
  );
  assert.equal(await page.getByText(/nothing .*prepared/, {exact: false}).count(), 0);
  assert.equal(await page.getByText(/packaging/i, {exact: false}).count(), 0);
  assert.deepEqual(
    await page.evaluate(() => window.applicationFixture.calls.filter((call) => call.body?.action === 'replace')),
    [
      {
        path: `/api/applications/${'a' + '1'.repeat(24)}/actions`,
        body: {
          action: 'replace',
          revision: 1,
          expected_container: 'a'.repeat(64),
          confirm_stop: true,
          replacement_build: 'e'.repeat(32),
        },
      },
    ]
  );
  await page.waitForTimeout(300);
  assert.equal(
    await page.evaluate(() => window.applicationFixture.calls.filter((call) => call.body?.action === 'replace').length),
    1
  );
});

test('unconfirmed apply without completion keeps apply guidance without packaging words', async (t) => {
  const page = await browser.newPage();
  t.after(() => page.close());
  await page.goto(server.url.href);
  await page.waitForFunction(() => typeof window.createApplicationDeliveryFixture === 'function');
  await page.evaluate(() => window.createApplicationDeliveryFixture(true));
  await page.getByRole('button', {name: 'Prepare company version', exact: true}).click();
  const apply = page.getByRole('button', {name: 'Apply version to Example', exact: true});
  await apply.waitFor();
  await page.getByRole('checkbox', {name: 'Stop Example for everyone and apply this version', exact: true}).check();
  await page.evaluate(() => {
    const app = window.applicationFixture.element,
      delivery = document.querySelector('soda-application-delivery') as unknown as {
        request: typeof app.request;
      },
      original = app.request!;
    const failing: typeof app.request = async (path, body, signal) => {
      if (body?.action === 'replace') {
        window.applicationFixture.calls.push({path, body});
        throw Object.assign(new Error('native outcome unknown'), {status: 502, code: 'native_outcome_unconfirmed'});
      }
      return original(path, body, signal);
    };
    app.request = failing;
    delivery.request = failing;
  });
  await apply.click();
  await page
    .getByText(
      'The apply outcome for Example could not be confirmed. Refresh applications and inspect Example to reconcile the same apply before applying again.',
      {exact: true}
    )
    .waitFor();
  // The failure is about the app image/runtime, never "packaging", and the
  // builds-list refresh is not offered as its reconciliation.
  assert.equal(await page.getByText(/packaging/i, {exact: false}).count(), 0);
  assert.equal(await page.getByRole('button', {name: 'Follow this delivery in Feed', exact: true}).count(), 0);
  assert.equal(
    await page.evaluate(() => window.applicationFixture.calls.filter((call) => call.body?.action === 'replace').length),
    1
  );
  await page.waitForTimeout(300);
  assert.equal(
    await page.evaluate(() => window.applicationFixture.calls.filter((call) => call.body?.action === 'replace').length),
    1
  );
});

test('pending apply names the in-progress operation without a duplicate apply', async (t) => {
  const page = await browser.newPage();
  t.after(() => page.close());
  await page.goto(server.url.href);
  await page.waitForFunction(() => typeof window.createApplicationDeliveryFixture === 'function');
  await page.evaluate(() => window.createApplicationDeliveryFixture(true));
  await page.getByRole('button', {name: 'Prepare company version', exact: true}).click();
  const apply = page.getByRole('button', {name: 'Apply version to Example', exact: true});
  await apply.waitFor();
  await page.getByRole('checkbox', {name: 'Stop Example for everyone and apply this version', exact: true}).check();
  await page.evaluate(() => {
    const app = window.applicationFixture.element,
      delivery = document.querySelector('soda-application-delivery') as unknown as {
        request: typeof app.request;
      },
      original = app.request!;
    const failing: typeof app.request = async (path, body, signal) => {
      if (body?.action === 'replace') {
        window.applicationFixture.calls.push({path, body});
        throw Object.assign(new Error('operation in progress'), {status: 409, code: 'application_pending'});
      }
      return original(path, body, signal);
    };
    app.request = failing;
    delivery.request = failing;
  });
  const writesBefore = await page.evaluate(() => window.applicationFixture.calls.filter((call) => call.body).length);
  const inspectsBefore = await page.evaluate(
    () => window.applicationFixture.calls.filter((call) => call.body?.action === 'inspect').length
  );
  await apply.click();
  await page
    .getByText(
      'An operation on Example is already in progress. This apply was not submitted again; Refresh applications and inspect Example to reconcile the same apply before acting again.',
      {exact: true}
    )
    .waitFor();
  assert.equal(await page.getByText(/packaging/i, {exact: false}).count(), 0);
  assert.equal(await page.getByRole('button', {name: 'Follow this delivery in Feed', exact: true}).count(), 0);
  // The genuine apply POST is the only new write; pending never re-reads or retries.
  assert.equal(
    await page.evaluate(() => window.applicationFixture.calls.filter((call) => call.body).length),
    writesBefore + 1
  );
  assert.equal(
    await page.evaluate(() => window.applicationFixture.calls.filter((call) => call.body?.action === 'inspect').length),
    inspectsBefore
  );
  await page.waitForTimeout(300);
  assert.equal(
    await page.evaluate(() => window.applicationFixture.calls.filter((call) => call.body?.action === 'replace').length),
    1
  );
});

test('preview refusal keeps the company selection, build and setup', async (t) => {
  const page = await browser.newPage();
  t.after(() => page.close());
  await page.goto(server.url.href);
  await page.waitForFunction(() => typeof window.createApplicationDeliveryFixture === 'function');
  await page.evaluate(() => window.createApplicationDeliveryFixture());
  await page.getByRole('button', {name: 'Prepare company version', exact: true}).click();
  await page.getByRole('button', {name: 'Set up company application', exact: true}).waitFor();
  await page.evaluate(() => {
    const app = window.applicationFixture.element,
      delivery = document.querySelector('soda-application-delivery') as unknown as {
        request: typeof app.request;
      },
      original = app.request!;
    const failing: typeof app.request = async (path, body, signal) => {
      if (body && body.environment === 'preview') {
        window.applicationFixture.calls.push({path, body});
        throw Object.assign(new Error('preview bundle missing'), {status: 409, code: 'delivery_unavailable'});
      }
      return original(path, body, signal);
    };
    app.request = failing;
    delivery.request = failing;
  });
  await page.getByRole('button', {name: 'Prepare preview', exact: true}).click();
  await page
    .getByText(
      'This delivery has no current verified preview bundle. Follow its checks in Feed; nothing was prepared for this preview delivery and no preview setup is retained.',
      {exact: true}
    )
    .waitFor();
  // No global absence is claimed: the company build, setup and selection survive.
  assert.equal(await page.getByText(/and no setup is retained/, {exact: false}).count(), 0);
  assert.equal(await page.getByText(/no current verified application bundle/, {exact: false}).count(), 0);
  await page.getByRole('button', {name: 'Set up company application', exact: true}).waitFor();
  assert.equal(await page.evaluate(() => window.applicationFixture.element.requestedDelivery), 'b'.repeat(32));
  assert.equal(
    await page.evaluate(
      () => window.applicationFixture.calls.filter((call) => call.body?.environment === 'preview').length
    ),
    1
  );
});

test('verified delivery prepares exact bytes and resumes owner setup without deployment JSON', async (t) => {
  const page = await browser.newPage();
  t.after(() => page.close());
  const errors: string[] = [];
  page.on('pageerror', (error) => errors.push(error.message));
  await page.goto(server.url.href);
  await page.waitForFunction(() => typeof window.createApplicationDeliveryFixture === 'function');
  await page.evaluate(() => window.createApplicationDeliveryFixture());
  await page.getByRole('button', {name: 'Prepare company version', exact: true}).waitFor();
  assert.equal(await page.evaluate(() => window.applicationFixture.calls.filter((call) => call.body).length), 0);
  await page.getByRole('button', {name: 'Prepare company version', exact: true}).click();
  await page.getByText('Application administrator: Maya.', {exact: false}).waitFor();
  await page.getByRole('button', {name: 'Set up company application', exact: true}).click();
  await page.getByRole('link', {name: 'Create application owner ↗', exact: true}).waitFor();
  const writes = await page.evaluate(() => window.applicationFixture.calls.filter((call) => call.body));
  assert.deepEqual(writes[0]?.body, {delivery_id: 'b'.repeat(32), environment: 'company'});
  assert.deepEqual(writes[1]?.body, {});
  assert.equal(writes[2]?.body?.action, 'begin');
  assert.equal(
    writes.some((call) => 'application' in (call.body ?? {})),
    false
  );
  assert.deepEqual(errors, []);
});

test('prepared same-origin build updates the existing company identity with confirmed Stop', async (t) => {
  const page = await browser.newPage();
  t.after(() => page.close());
  await page.goto(server.url.href);
  await page.waitForFunction(() => typeof window.createApplicationDeliveryFixture === 'function');
  await page.evaluate(() => window.createApplicationDeliveryFixture(true));
  await page.getByRole('button', {name: 'Prepare company version', exact: true}).click();
  const apply = page.getByRole('button', {name: 'Apply version to Example', exact: true});
  await apply.waitFor();
  assert(await apply.isDisabled());
  await page.getByRole('checkbox', {name: 'Stop Example for everyone and apply this version', exact: true}).check();
  await apply.click();
  await page
    .getByText(
      'The version is installed and stopped. Start the company application, then check its retained records.',
      {exact: true}
    )
    .waitFor();
  const writes = await page.evaluate(() =>
    window.applicationFixture.calls.filter((call) => call.body?.action === 'replace')
  );
  assert.equal(writes.length, 1);
  assert.deepEqual(writes[0]?.body, {
    action: 'replace',
    revision: 1,
    expected_container: 'a'.repeat(64),
    confirm_stop: true,
    replacement_build: 'e'.repeat(32),
  });
});

test('preview preparation stays distinct from preparing the company version', async (t) => {
  const page = await browser.newPage();
  t.after(() => page.close());
  await page.goto(server.url.href);
  await page.waitForFunction(() => typeof window.createApplicationDeliveryFixture === 'function');
  await page.evaluate(() => window.createApplicationDeliveryFixture());
  await page.getByRole('button', {name: 'Prepare preview', exact: true}).click();
  await page.getByRole('button', {name: 'Set up preview', exact: true}).waitFor();
  assert.equal(await page.getByRole('button', {name: 'Set up company application', exact: true}).count(), 0);
  await page.getByRole('button', {name: 'Prepare company version', exact: true}).click();
  await page.getByRole('button', {name: 'Set up company application', exact: true}).waitFor();
  assert(await page.getByRole('button', {name: 'Set up preview', exact: true}).isVisible());
  const writes = await page.evaluate(() => window.applicationFixture.calls.filter((call) => call.body));
  assert.deepEqual(
    writes.map((call) => call.body),
    [
      {delivery_id: 'b'.repeat(32), environment: 'preview'},
      {delivery_id: 'b'.repeat(32), environment: 'company'},
    ]
  );
});

test('a selected delivery opens only its verified application and makes switching to the current app explicit', async (t) => {
  const page = await browser.newPage();
  t.after(() => page.close());
  await page.goto(server.url.href);
  await page.waitForFunction(() => typeof window.createApplicationFixture === 'function');
  await page.evaluate(async () => {
    await window.createApplicationFixture();
    const app = window.applicationFixture.element;
    const original = app.request!;
    app.request = async (...args) => {
      const result = (await original(...args)) as {
        applications?: import('../../frontend/spaces/sodaspaces-application-response').ApplicationRecord[];
        application?: unknown;
        observation?: Record<string, unknown>;
        delivery?: unknown;
      };
      const rows =
        result.applications ??
        (result.application
          ? [result as unknown as import('../../frontend/spaces/sodaspaces-application-response').ApplicationRecord]
          : []);
      for (const row of rows) {
        if (!row.observation) continue;
        row.observation.bundle_file_sha256 = 'b'.repeat(64);
        row.delivery = {
          environment: 'company',
          state: 'verified',
          intent: 'intent-current',
          revision: 'revision-current',
          delivery_id: 'delivery-current',
          run: 'run-current',
          source_commit: 'c'.repeat(40),
          bundle_file_sha256: 'b'.repeat(64),
        };
      }
      return result;
    };
    app.requestedDelivery = 'delivery-older';
    app.expanded = true;
  });
  await page
    .getByText(
      'An application matching this exact result is not available to you. The current company application may be a different version.',
      {exact: true}
    )
    .waitFor();
  assert.equal(await page.locator('soda-application-surface').count(), 0);
  assert.equal(await page.getByRole('button', {name: 'Start Example', exact: true}).count(), 0);
  await page.evaluate(() => {
    window.applicationFixture.element.requestedDelivery = 'delivery-current';
  });
  await page.locator('soda-application-surface').waitFor();
  await page.evaluate(() => {
    window.applicationFixture.element.requestedDelivery = 'delivery-older';
  });
  await page
    .getByText(
      'An application matching this exact result is not available to you. The current company application may be a different version.',
      {exact: true}
    )
    .waitFor();
  await page.getByRole('button', {name: 'Open current company application', exact: true}).click();
  await page.locator('soda-application-surface').waitFor();
  assert.equal(await page.evaluate(() => window.applicationFixture.element.requestedDelivery), '');
  assert.equal(
    await page.evaluate(
      () => window.applicationFixture.calls.filter((call) => call.body && call.body.action !== 'inspect').length
    ),
    0
  );
});

test('ordinary employee opens the exact verified result without lifecycle controls and older results stay unavailable', async (t) => {
  const page = await browser.newPage();
  t.after(() => page.close());
  await page.route('https://app.example.test/**', (route) =>
    route.fulfill({body: '<p>Employee application</p>', contentType: 'text/html'})
  );
  await page.goto(server.url.href);
  await page.waitForFunction(() => typeof window.createApplicationFixture === 'function');
  await page.evaluate(async () => {
    await window.createApplicationFixture();
    const fixture = window.applicationFixture,
      app = fixture.element;
    app.request = async (path, body) => {
      fixture.calls.push({path, ...(body ? {body} : {})});
      if (body) throw Error('ordinary account cannot administer apps');
      return {
        applications: [],
        launchable: [
          {
            application: {id: 'a' + '1'.repeat(24), revision: 2, name: 'Example', origin: 'https://app.example.test'},
            environment: 'company',
            delivery_id: 'b'.repeat(32),
            running: true,
            ready: true,
            state: 'ready',
            project: {repository: '7', name: 'northstar/equipment', path: '/northstar/equipment'},
            launch_only: true,
            can_embed: true,
          },
        ],
        complete: true,
        can_setup: false,
      };
    };
    app.repository = '7';
    app.requestedDelivery = 'b'.repeat(32);
    app.expanded = true;
  });
  await page.locator('soda-application-surface iframe').waitFor();
  assert.equal(await page.locator('soda-application-surface iframe').getAttribute('src'), 'https://app.example.test');
  await page
    .getByText(
      'Accounts and saved data live with this application at its address, separate from Soda Forge. An operator manages its setup and updates.',
      {exact: true}
    )
    .waitFor();
  await page.getByText('Application setup and updates are handled by an operator.', {exact: true}).waitFor();
  assert.equal(await page.getByText('Start or stop application', {exact: true}).count(), 0);
  assert.equal(await page.getByText('Set up a new application (operator)', {exact: true}).count(), 0);
  assert.equal(await page.locator('soda-application-delivery').count(), 0);
  await page.evaluate(() => {
    window.applicationFixture.element.requestedDelivery = 'c'.repeat(32);
  });
  await page
    .getByText(
      'An application matching this exact result is not available to you. The current company application may be a different version.',
      {exact: true}
    )
    .waitFor();
  assert.equal(await page.locator('soda-application-surface').count(), 0);
  await page.evaluate(() => {
    const app = window.applicationFixture.element;
    app.repository = '8';
    app.requestedDelivery = 'b'.repeat(32);
  });
  await page
    .getByText(
      'An application matching this exact result is not available to you. The current company application may be a different version.',
      {exact: true}
    )
    .waitFor();
  assert.equal(await page.locator('soda-application-surface').count(), 0);
  await page.evaluate(() => {
    window.applicationFixture.element.repository = '7';
  });
  await page.locator('soda-application-surface iframe').waitFor();
  assert.equal(await page.evaluate(() => window.applicationFixture.calls.some((call) => !!call.body)), false);
  await page.evaluate(() => {
    const app = window.applicationFixture.element,
      request = app.request!;
    app.request = async (...args) => {
      const value = (await request(...args)) as {launchable: Record<string, unknown>[]};
      Object.assign(value.launchable[0]!, {state: 'running', ready: false, environment: 'preview'});
      return value;
    };
  });
  await page
    .getByText(
      'The application is running, but readiness has not been confirmed. An operator can inspect its live status; Refresh applications only rereads the last stored status.',
      {exact: true}
    )
    .waitFor();
  assert.equal(await page.locator('soda-application-surface iframe').count(), 0);
  await page.getByText('Preview · separate data', {exact: false}).waitFor();
  await page.evaluate(() => {
    const app = window.applicationFixture.element,
      request = app.request!;
    app.request = async (...args) => {
      const value = (await request(...args)) as {launchable: Record<string, unknown>[]};
      Object.assign(value.launchable[0]!, {state: 'not-installed', running: false});
      return value;
    };
  });
  await page
    .getByText(
      'Application setup is not complete. An operator must finish setup before its administrator can start it.',
      {exact: true}
    )
    .waitFor();
  assert.equal(await page.locator('soda-application-surface iframe').count(), 0);
  assert.equal(await page.getByRole('button', {name: 'Start Example', exact: true}).count(), 0);
  assert.equal(await page.evaluate(() => window.applicationFixture.calls.some((call) => !!call.body)), false);
});

test('a selected delivery shows only its exact version while other verified versions stay hidden', async (t) => {
  const page = await browser.newPage();
  t.after(() => page.close());
  await page.route('https://app1.example.test/**', (route) =>
    route.fulfill({body: '<p>Version one</p>', contentType: 'text/html'})
  );
  await page.goto(server.url.href);
  await page.waitForFunction(() => typeof window.createApplicationFixture === 'function');
  await page.evaluate(async () => {
    await window.createApplicationFixture();
    const app = window.applicationFixture.element;
    const versions = [
      {suffix: '1', delivery: 'b'.repeat(32), ready: true},
      {suffix: '2', delivery: 'c'.repeat(32), ready: false},
    ];
    app.request = async (path, body) => {
      window.applicationFixture.calls.push({path, ...(body ? {body} : {})});
      const rows = versions.map(({suffix, delivery, ready}) => ({
        environment: 'company',
        application: {
          id: 'a' + suffix.repeat(24),
          revision: Number(suffix),
          name: 'Example-v' + suffix,
          artifact_sha256: '2'.repeat(64),
          bundle_sha256: '3'.repeat(64),
          tools_sha256: '4'.repeat(64),
          runtime_image: 'sha256:' + '5'.repeat(64),
          image: 'sha256:' + '6'.repeat(64),
          data: 'a' + '7'.repeat(24),
          owner_team: '12345678-1234-1234-1234-123456789abc',
          origin: `https://app${suffix}.example.test`,
          port: 20000 + Number(suffix),
        },
        administrator_id: '41',
        operator: false,
        observation: {
          id: 'a' + suffix.repeat(24),
          revision: Number(suffix),
          container: suffix.repeat(64),
          image: 'sha256:' + '6'.repeat(64),
          origin: `https://app${suffix}.example.test`,
          running: ready,
          ready,
          bundle_file_sha256: (suffix === '1' ? 'b' : 'c').repeat(64),
        },
        can_embed: true,
        project: {repository: '7', name: 'northstar/equipment', path: '/northstar/equipment'},
        delivery: {
          environment: 'company',
          state: 'verified',
          intent: 'intent-' + suffix,
          revision: 'revision-' + suffix,
          delivery_id: delivery,
          run: 'run-' + suffix,
          source_commit: suffix.repeat(40),
          bundle_file_sha256: (suffix === '1' ? 'b' : 'c').repeat(64),
        },
      }));
      if (!body) return {applications: rows, complete: true};
      // Status reads echo the addressed record; anything else would be a product write.
      if (body.action === 'inspect')
        return rows.find((row) => path === `/api/applications/${row.application.id}/actions`);
      throw Error('version filtering must not write');
    };
    app.repository = '7';
    app.requestedDelivery = 'b'.repeat(32);
    app.expanded = true;
  });
  await page.locator('.soda-application-heading').getByText('Example-v1', {exact: true}).waitFor();
  assert.equal(await page.locator('.soda-application-heading').getByText('Example-v2', {exact: true}).count(), 0);
  assert.equal(await page.locator('soda-application-surface').count(), 1);
  assert.equal(await page.locator('soda-application-surface iframe').getAttribute('src'), 'https://app1.example.test');
  assert.equal(await page.getByRole('button', {name: 'Start Example-v2', exact: true}).count(), 0);
  await page.evaluate(() => {
    window.applicationFixture.element.requestedDelivery = 'd'.repeat(32);
  });
  await page
    .getByText(
      'An application matching this exact result is not available to you. The current company application may be a different version.',
      {exact: true}
    )
    .waitFor();
  assert.equal(await page.locator('soda-application-surface').count(), 0);
  assert.equal(await page.locator('.soda-application-heading').getByText('Example-v1', {exact: true}).count(), 0);
  assert.equal(await page.locator('.soda-application-heading').getByText('Example-v2', {exact: true}).count(), 0);
  await page.getByRole('button', {name: 'Open current company application', exact: true}).click();
  await page.locator('.soda-application-heading').getByText('Example-v1', {exact: true}).waitFor();
  await page.locator('.soda-application-heading').getByText('Example-v2', {exact: true}).waitFor();
  assert.equal(await page.locator('soda-application-surface').count(), 2);
  assert.equal(await page.evaluate(() => window.applicationFixture.element.requestedDelivery), '');
  assert.equal(
    await page.evaluate(
      () => window.applicationFixture.calls.filter((call) => call.body && call.body.action !== 'inspect').length
    ),
    0
  );
});

test('browser reload retains the selected delivery for the same project and actor', async (t) => {
  const page = await browser.newPage();
  t.after(() => page.close());
  await page.goto(server.url.href);
  await page.waitForFunction(() => typeof window.createWorkspaceDeliveryFixture === 'function');
  await page.evaluate(() => window.createWorkspaceDeliveryFixture());
  const delivery = 'b'.repeat(32);
  await page.evaluate((delivery) => {
    document.querySelector('soda-spaces')?.dispatchEvent(
      new CustomEvent('soda-application-open', {
        bubbles: true,
        composed: true,
        detail: {repository: '7', deliveryId: delivery},
      })
    );
  }, delivery);
  await page.waitForFunction((delivery) => localStorage.getItem('soda-project-delivery:v1:1:7') === delivery, delivery);
  assert.equal(
    await page.evaluate(
      () =>
        (document.querySelector('soda-applications') as unknown as {requestedDelivery: string} | null)
          ?.requestedDelivery
    ),
    delivery
  );
  await page.locator('.soda-application-heading').getByText('Example', {exact: true}).waitFor();
  await page.reload();
  await page.waitForFunction(() => typeof window.createWorkspaceDeliveryFixture === 'function');
  await page.evaluate(() => window.createWorkspaceDeliveryFixture());
  assert.equal(await page.evaluate(() => localStorage.getItem('soda-project-delivery:v1:1:7')), delivery);
  await page.evaluate(() => window.workspaceDeliveryFixture.present('page', '7'));
  await page.waitForFunction(
    (delivery) =>
      (document.querySelector('soda-applications') as unknown as {requestedDelivery: string} | null)
        ?.requestedDelivery === delivery,
    delivery
  );
  await page.locator('.soda-application-heading').getByText('Example', {exact: true}).waitFor();
  // Another project never inherits the retained selection.
  await page.evaluate(() => window.workspaceDeliveryFixture.present('page', '8'));
  await page.waitForFunction(() => document.querySelector('soda-applications') === null);
  assert.equal(await page.evaluate(() => localStorage.getItem('soda-project-delivery:v1:1:8')), null);
  // A well-formed but unknown delivery survives reload as a selection and shows the truthful notice.
  await page.evaluate(() => window.workspaceDeliveryFixture.present('page', '7'));
  await page.waitForFunction(
    (delivery) =>
      (document.querySelector('soda-applications') as unknown as {requestedDelivery: string} | null)
        ?.requestedDelivery === delivery,
    delivery
  );
  const unknown = 'd'.repeat(32);
  await page.evaluate((unknown) => {
    document.querySelector('soda-spaces')?.dispatchEvent(
      new CustomEvent('soda-application-open', {
        bubbles: true,
        composed: true,
        detail: {repository: '7', deliveryId: unknown},
      })
    );
  }, unknown);
  await page.waitForFunction((unknown) => localStorage.getItem('soda-project-delivery:v1:1:7') === unknown, unknown);
  await page
    .getByText(
      'An application matching this exact result is not available to you. The current company application may be a different version.',
      {exact: true}
    )
    .waitFor();
  assert.equal(await page.locator('soda-application-surface').count(), 0);
  // The unknown selection survives a second reload with the same truthful notice.
  await page.reload();
  await page.waitForFunction(() => typeof window.createWorkspaceDeliveryFixture === 'function');
  await page.evaluate(() => window.createWorkspaceDeliveryFixture());
  assert.equal(await page.evaluate(() => localStorage.getItem('soda-project-delivery:v1:1:7')), unknown);
  await page.evaluate(() => window.workspaceDeliveryFixture.present('page', '7'));
  await page.waitForFunction(
    (unknown) =>
      (document.querySelector('soda-applications') as unknown as {requestedDelivery: string} | null)
        ?.requestedDelivery === unknown,
    unknown
  );
  await page
    .getByText(
      'An application matching this exact result is not available to you. The current company application may be a different version.',
      {exact: true}
    )
    .waitFor();
  assert.equal(await page.locator('soda-application-surface').count(), 0);
  // An invalid delivery ID is rejected and clears the retained selection.
  await page.evaluate(() => {
    document.querySelector('soda-spaces')?.dispatchEvent(
      new CustomEvent('soda-application-open', {
        bubbles: true,
        composed: true,
        detail: {repository: '7', deliveryId: 'not-a-delivery'},
      })
    );
  });
  await page.waitForFunction(() => localStorage.getItem('soda-project-delivery:v1:1:7') === null);
  assert.equal(
    await page.evaluate(
      () =>
        (document.querySelector('soda-applications') as unknown as {requestedDelivery: string} | null)
          ?.requestedDelivery
    ),
    ''
  );
  await page.locator('.soda-application-heading').getByText('Example', {exact: true}).waitFor();
  // Reselecting then opening the current application clears the retained selection.
  await page.evaluate((delivery) => {
    document.querySelector('soda-spaces')?.dispatchEvent(
      new CustomEvent('soda-application-open', {
        bubbles: true,
        composed: true,
        detail: {repository: '7', deliveryId: delivery},
      })
    );
  }, delivery);
  await page.waitForFunction((delivery) => localStorage.getItem('soda-project-delivery:v1:1:7') === delivery, delivery);
  await page.evaluate(() => {
    document
      .querySelector('soda-spaces')
      ?.dispatchEvent(new CustomEvent('application-delivery-cleared', {bubbles: true, composed: true}));
  });
  await page.waitForFunction(() => localStorage.getItem('soda-project-delivery:v1:1:7') === null);
  assert.equal(
    await page.evaluate(
      () =>
        (document.querySelector('soda-applications') as unknown as {requestedDelivery: string} | null)
          ?.requestedDelivery
    ),
    ''
  );
  assert.equal(
    await page.evaluate(
      () => window.workspaceDeliveryFixture.calls.filter((call) => call.body && call.body.action !== 'inspect').length
    ),
    0
  );
});

test('browser reload restores the opened company scope for the same project', async (t) => {
  const page = await browser.newPage();
  t.after(() => page.close());
  await page.goto(server.url.href);
  await page.waitForFunction(() => typeof window.createWorkspaceDeliveryFixture === 'function');
  await page.evaluate(() => window.createWorkspaceDeliveryFixture());
  const delivery = 'b'.repeat(32);
  await page.evaluate((delivery) => {
    document.querySelector('soda-spaces')?.dispatchEvent(
      new CustomEvent('soda-application-open', {
        bubbles: true,
        composed: true,
        detail: {repository: '7', deliveryId: delivery},
      })
    );
  }, delivery);
  await page.waitForFunction((delivery) => localStorage.getItem('soda-project-delivery:v1:1:7') === delivery, delivery);
  await page.evaluate(() => window.workspaceDeliveryFixture.present('page', '7'));
  await page.getByRole('button', {name: 'Open current company application', exact: true}).waitFor();
  await page.getByRole('button', {name: 'Open current company application', exact: true}).click();
  // Opening the company surface clears the exact selection and retains the scope.
  await page.waitForFunction(() => localStorage.getItem('soda-project-delivery:v1:1:7') === null);
  await page.waitForFunction(() => localStorage.getItem('soda-project-scope:v1:1:7') === 'company');
  await page.getByText('Showing the current company application; previews stay separate.', {exact: true}).waitFor();
  assert.equal(
    await page.evaluate(
      () =>
        (document.querySelector('soda-applications') as unknown as {requestedDelivery: string} | null)
          ?.requestedDelivery
    ),
    ''
  );
  // A reload restores the retained scope: the company surface returns without
  // another click, and the exact selection stays cleared.
  await page.reload();
  await page.waitForFunction(() => typeof window.createWorkspaceDeliveryFixture === 'function');
  await page.evaluate(() => window.createWorkspaceDeliveryFixture());
  assert.equal(await page.evaluate(() => localStorage.getItem('soda-project-scope:v1:1:7')), 'company');
  await page.evaluate(() => window.workspaceDeliveryFixture.present('page', '7'));
  await page.getByText('Showing the current company application; previews stay separate.', {exact: true}).waitFor();
  assert.equal(await page.evaluate(() => localStorage.getItem('soda-project-delivery:v1:1:7')), null);
  // Another project never inherits the retained scope.
  await page.evaluate(() => window.workspaceDeliveryFixture.present('page', '8'));
  await page.waitForFunction(() => document.querySelector('soda-applications') === null);
  assert.equal(await page.evaluate(() => localStorage.getItem('soda-project-scope:v1:1:8')), null);
  // Selecting an exact delivery leaves the company scope and clears the retention.
  await page.evaluate(() => window.workspaceDeliveryFixture.present('page', '7'));
  await page.getByText('Showing the current company application; previews stay separate.', {exact: true}).waitFor();
  await page.evaluate((delivery) => {
    document.querySelector('soda-spaces')?.dispatchEvent(
      new CustomEvent('soda-application-open', {
        bubbles: true,
        composed: true,
        detail: {repository: '7', deliveryId: delivery},
      })
    );
  }, delivery);
  await page.waitForFunction((delivery) => localStorage.getItem('soda-project-delivery:v1:1:7') === delivery, delivery);
  await page.waitForFunction(() => localStorage.getItem('soda-project-scope:v1:1:7') === null);
  await page.waitForFunction(
    (marker) =>
      [...document.querySelectorAll('soda-applications p')].filter((el) => el.textContent === marker).length === 0,
    'Showing the current company application; previews stay separate.'
  );
});

test('unavailable delivery refuses honestly with a Feed next action and no automatic retry', async (t) => {
  const page = await browser.newPage();
  t.after(() => page.close());
  await page.goto(server.url.href);
  await page.waitForFunction(() => typeof window.createApplicationDeliveryFixture === 'function');
  await page.evaluate(() => window.createApplicationDeliveryFixture());
  await page.getByRole('button', {name: 'Prepare company version', exact: true}).waitFor();
  await page.evaluate(() => {
    const app = window.applicationFixture.element,
      delivery = document.querySelector('soda-application-delivery') as unknown as {
        request: typeof app.request;
      },
      original = app.request!;
    const failing: typeof app.request = async (path, body, signal) => {
      if (body) {
        window.applicationFixture.calls.push({path, body});
        throw Object.assign(new Error('delivery refused'), {status: 409, code: 'delivery_unavailable'});
      }
      return original(path, body, signal);
    };
    app.request = failing;
    delivery.request = failing;
  });
  await page.getByRole('button', {name: 'Prepare company version', exact: true}).click();
  await page
    .getByText(
      'This delivery has no current verified company bundle. Follow its checks in Feed; nothing was prepared for this company delivery and no company setup is retained.',
      {exact: true}
    )
    .waitFor();
  // The genuine button click issues exactly its own authorized POST and
  // nothing follows it automatically.
  assert.deepEqual(await page.evaluate(() => window.applicationFixture.calls.filter((call) => call.body)), [
    {path: '/api/repositories/7/application-builds', body: {delivery_id: 'b'.repeat(32), environment: 'company'}},
  ]);
  await page.waitForTimeout(300);
  assert.equal(await page.evaluate(() => window.applicationFixture.calls.filter((call) => call.body).length), 1);
  // No setup exists, so none is claimed; the refusal never traps the user in refresh.
  assert.equal(await page.getByText(/records and setup are retained/, {exact: false}).count(), 0);
  assert.equal(await page.getByRole('button', {name: 'Set up company application', exact: true}).count(), 0);
  const navigation = page.evaluate(
    () =>
      new Promise<unknown>((resolve) => {
        window.addEventListener('soda-project-feed-open', (event) => resolve((event as CustomEvent).detail), {
          once: true,
        });
      })
  );
  await page.getByRole('button', {name: 'Follow this delivery in Feed', exact: true}).click();
  assert.deepEqual(await navigation, {repository: '7'});
  assert.equal(await page.evaluate(() => window.applicationFixture.calls.filter((call) => call.body).length), 1);
});

test('changed delivery refuses honestly without a refresh trap', async (t) => {
  const page = await browser.newPage();
  t.after(() => page.close());
  await page.goto(server.url.href);
  await page.waitForFunction(() => typeof window.createApplicationDeliveryFixture === 'function');
  await page.evaluate(() => window.createApplicationDeliveryFixture());
  await page.getByRole('button', {name: 'Prepare company version', exact: true}).waitFor();
  await page.evaluate(() => {
    const app = window.applicationFixture.element,
      delivery = document.querySelector('soda-application-delivery') as unknown as {
        request: typeof app.request;
      },
      original = app.request!;
    const failing: typeof app.request = async (path, body, signal) => {
      if (body) {
        window.applicationFixture.calls.push({path, body});
        throw Object.assign(new Error('delivery changed'), {status: 409, code: 'build_changed'});
      }
      return original(path, body, signal);
    };
    app.request = failing;
    delivery.request = failing;
  });
  await page.getByRole('button', {name: 'Prepare company version', exact: true}).click();
  await page
    .getByText(
      'This company delivery changed: the prepared version no longer matches the current verified delivery. Follow its checks in Feed; nothing was prepared for this company delivery and no company setup is retained.',
      {exact: true}
    )
    .waitFor();
  assert.equal(await page.evaluate(() => window.applicationFixture.calls.filter((call) => call.body).length), 1);
  await page.waitForTimeout(300);
  assert.equal(await page.evaluate(() => window.applicationFixture.calls.filter((call) => call.body).length), 1);
  assert.equal(await page.getByText(/records and setup are retained/, {exact: false}).count(), 0);
  await page.getByRole('button', {name: 'Follow this delivery in Feed', exact: true}).waitFor();
});

test('pending delivery keeps the refresh path without a Feed detour', async (t) => {
  const page = await browser.newPage();
  t.after(() => page.close());
  await page.goto(server.url.href);
  await page.waitForFunction(() => typeof window.createApplicationDeliveryFixture === 'function');
  await page.evaluate(() => window.createApplicationDeliveryFixture());
  await page.getByRole('button', {name: 'Prepare company version', exact: true}).waitFor();
  await page.evaluate(() => {
    const app = window.applicationFixture.element,
      delivery = document.querySelector('soda-application-delivery') as unknown as {
        request: typeof app.request;
      },
      original = app.request!;
    const failing: typeof app.request = async (path, body, signal) => {
      if (body) {
        window.applicationFixture.calls.push({path, body});
        throw Object.assign(new Error('packaging pending'), {status: 409, code: 'build_pending'});
      }
      return original(path, body, signal);
    };
    app.request = failing;
    delivery.request = failing;
  });
  await page.getByRole('button', {name: 'Prepare company version', exact: true}).click();
  await page
    .getByText(
      'Packaging this company delivery is already in progress. This preparation was not submitted again; wait for the in-progress packaging, then Refresh prepared versions to reconcile the same delivery.',
      {exact: true}
    )
    .waitFor();
  // The pending packaging is named, not denied: no absence claims.
  assert.equal(await page.getByText(/did not finish/, {exact: false}).count(), 0);
  assert.equal(await page.getByText(/nothing .*prepared/, {exact: false}).count(), 0);
  assert.equal(await page.getByRole('button', {name: 'Follow this delivery in Feed', exact: true}).count(), 0);
  await page.getByRole('button', {name: 'Refresh prepared versions', exact: true}).click();
  await page.waitForTimeout(300);
  assert.equal(await page.evaluate(() => window.applicationFixture.calls.filter((call) => call.body).length), 1);
});

test('expired session on delivery asks for reconnect instead of refresh', async (t) => {
  const page = await browser.newPage();
  t.after(() => page.close());
  await page.goto(server.url.href);
  await page.waitForFunction(() => typeof window.createApplicationDeliveryFixture === 'function');
  await page.evaluate(() => window.createApplicationDeliveryFixture());
  await page.getByRole('button', {name: 'Prepare company version', exact: true}).waitFor();
  await page.evaluate(() => {
    const app = window.applicationFixture.element,
      delivery = document.querySelector('soda-application-delivery') as unknown as {
        request: typeof app.request;
      },
      original = app.request!;
    const failing: typeof app.request = async (path, body, signal) => {
      if (body) {
        window.applicationFixture.calls.push({path, body});
        throw Object.assign(new Error('session changed'), {status: 401});
      }
      return original(path, body, signal);
    };
    app.request = failing;
    delivery.request = failing;
  });
  await page.getByRole('button', {name: 'Prepare company version', exact: true}).click();
  await page
    .getByText(
      'Your Soda session changed. Reconnect to Soda Forge, then retry this delivery; nothing was prepared or changed.',
      {exact: true}
    )
    .waitFor();
  assert.equal(await page.getByRole('button', {name: 'Follow this delivery in Feed', exact: true}).count(), 0);
  assert.equal(await page.evaluate(() => window.applicationFixture.calls.filter((call) => call.body).length), 1);
  await page.waitForTimeout(300);
  assert.equal(await page.evaluate(() => window.applicationFixture.calls.filter((call) => call.body).length), 1);
});

test('unconfirmed packaging outcome stays neutral and reconciles the same delivery without retrying', async (t) => {
  const page = await browser.newPage();
  t.after(() => page.close());
  await page.goto(server.url.href);
  await page.waitForFunction(() => typeof window.createApplicationDeliveryFixture === 'function');
  await page.evaluate(() => window.createApplicationDeliveryFixture());
  await page.getByRole('button', {name: 'Prepare company version', exact: true}).waitFor();
  await page.evaluate(() => {
    const app = window.applicationFixture.element,
      delivery = document.querySelector('soda-application-delivery') as unknown as {
        request: typeof app.request;
      },
      original = app.request!;
    const failing: typeof app.request = async (path, body, signal) => {
      if (body) {
        window.applicationFixture.calls.push({path, body});
        throw Object.assign(new Error('packaging unconfirmed'), {status: 502, code: 'build_unconfirmed'});
      }
      return original(path, body, signal);
    };
    app.request = failing;
    delivery.request = failing;
  });
  const statusRead = () =>
    page.evaluate(
      () =>
        window.applicationFixture.calls.filter(
          (call) => !call.body && call.path === '/api/repositories/7/application-builds?delivery_id=' + 'b'.repeat(32)
        ).length
    );
  const readsBefore = await statusRead();
  await page.getByRole('button', {name: 'Prepare company version', exact: true}).click();
  await page
    .getByText(
      'The packaging outcome could not be confirmed. Refresh prepared versions to reconcile this same delivery before preparing again.',
      {exact: true}
    )
    .waitFor();
  // Neutral guidance claims neither completion nor absence, and offers no
  // Feed detour for an outcome that may have completed.
  assert.equal(await page.getByText(/nothing .*prepared/, {exact: false}).count(), 0);
  assert.equal(await page.getByText(/did not finish/, {exact: false}).count(), 0);
  assert.equal(await page.getByRole('button', {name: 'Follow this delivery in Feed', exact: true}).count(), 0);
  // The genuine button click issues exactly its own POST; the same-delivery
  // status read carries no body, and nothing retries automatically.
  assert.deepEqual(await page.evaluate(() => window.applicationFixture.calls.filter((call) => call.body)), [
    {path: '/api/repositories/7/application-builds', body: {delivery_id: 'b'.repeat(32), environment: 'company'}},
  ]);
  assert.equal(await statusRead(), readsBefore + 1);
  await page.waitForTimeout(300);
  assert.equal(await page.evaluate(() => window.applicationFixture.calls.filter((call) => call.body).length), 1);
  assert.equal(await statusRead(), readsBefore + 1);
});

test('lost reply after Prepare adopts the completed build without a duplicate submission', async (t) => {
  const page = await browser.newPage();
  t.after(() => page.close());
  await page.goto(server.url.href);
  await page.waitForFunction(() => typeof window.createApplicationDeliveryFixture === 'function');
  await page.evaluate(() => window.createApplicationDeliveryFixture());
  await page.getByRole('button', {name: 'Prepare company version', exact: true}).waitFor();
  await page.evaluate(() => {
    const app = window.applicationFixture.element,
      delivery = document.querySelector('soda-application-delivery') as unknown as {
        request: typeof app.request;
      },
      original = app.request!;
    let completed = false;
    const losing: typeof app.request = async (path, body, signal) => {
      if (path.startsWith('/api/repositories/7/application-builds')) {
        window.applicationFixture.calls.push({path, ...(body ? {body} : {})});
        if (body) {
          // The server prepared the version but the reply was lost.
          completed = true;
          throw Error('lost reply');
        }
        if (completed)
          return {
            builds: [
              {
                id: 'e'.repeat(32),
                artifact: {
                  name: 'Example',
                  origin: 'https://app.example.test',
                  image: 'sha256:' + '9'.repeat(64),
                  environment: 'company',
                },
                delivery: {delivery_id: 'b'.repeat(32), source_commit: 'c'.repeat(40), environment: 'company'},
              },
            ],
            administrator: {id: '1', name: 'Maya'},
          };
      }
      return original(path, body, signal);
    };
    app.request = losing;
    delivery.request = losing;
  });
  const statusRead = () =>
    page.evaluate(
      () =>
        window.applicationFixture.calls.filter(
          (call) => !call.body && call.path === '/api/repositories/7/application-builds?delivery_id=' + 'b'.repeat(32)
        ).length
    );
  const readsBefore = await statusRead();
  await page.getByRole('button', {name: 'Prepare company version', exact: true}).click();
  await page
    .getByText(
      'The packaging reply was unconfirmed, so the same delivery was re-read: its prepared version is shown below. Owner setup and company release are separate actions.',
      {exact: true}
    )
    .waitFor();
  // The reconciled build is adopted: owner setup proceeds with no duplicate POST.
  await page.getByRole('button', {name: 'Set up company application', exact: true}).waitFor();
  assert.equal(await page.getByText(/nothing .*prepared/, {exact: false}).count(), 0);
  assert.deepEqual(await page.evaluate(() => window.applicationFixture.calls.filter((call) => call.body)), [
    {path: '/api/repositories/7/application-builds', body: {delivery_id: 'b'.repeat(32), environment: 'company'}},
  ]);
  assert.equal(await statusRead(), readsBefore + 1);
  await page.waitForTimeout(300);
  assert.equal(await page.evaluate(() => window.applicationFixture.calls.filter((call) => call.body).length), 1);
});

test('packaging service failure stays unconfirmed with a safe refresh path', async (t) => {
  const page = await browser.newPage();
  t.after(() => page.close());
  await page.goto(server.url.href);
  await page.waitForFunction(() => typeof window.createApplicationDeliveryFixture === 'function');
  await page.evaluate(() => window.createApplicationDeliveryFixture());
  await page.getByRole('button', {name: 'Prepare company version', exact: true}).waitFor();
  // Prepare one version first so the service failure lands with existing builds.
  await page.getByRole('button', {name: 'Prepare company version', exact: true}).click();
  await page.getByRole('button', {name: 'Set up company application', exact: true}).waitFor();
  await page.evaluate(() => {
    const app = window.applicationFixture.element,
      delivery = document.querySelector('soda-application-delivery') as unknown as {
        request: typeof app.request;
      },
      original = app.request!;
    const failing: typeof app.request = async (path, body, signal) => {
      if (body) {
        window.applicationFixture.calls.push({path, body});
        throw Object.assign(new Error('packaging host down'), {status: 503, code: 'native_unavailable'});
      }
      return original(path, body, signal);
    };
    app.request = failing;
    delivery.request = failing;
  });
  await page.getByRole('button', {name: 'Prepare preview', exact: true}).click();
  await page
    .getByText(
      'Application packaging is unavailable right now. The outcome was not confirmed; refresh prepared versions before acting again.',
      {exact: true}
    )
    .waitFor();
  // A host 503 never asserts absence: the prepared company version stays shown.
  assert.equal(await page.getByText(/nothing was prepared/, {exact: false}).count(), 0);
  assert.equal(await page.getByText(/no setup is retained/, {exact: false}).count(), 0);
  assert.equal(await page.getByText(/no current verified application bundle/, {exact: false}).count(), 0);
  assert.equal(await page.getByRole('button', {name: 'Follow this delivery in Feed', exact: true}).count(), 0);
  await page.getByRole('button', {name: 'Set up company application', exact: true}).waitFor();
  await page.getByRole('button', {name: 'Refresh prepared versions', exact: true}).waitFor();
  assert.equal(await page.evaluate(() => window.applicationFixture.calls.filter((call) => call.body).length), 2);
  await page.waitForTimeout(300);
  assert.equal(await page.evaluate(() => window.applicationFixture.calls.filter((call) => call.body).length), 2);
});

test('operator permission refusal names the operator instead of a session change', async (t) => {
  const page = await browser.newPage();
  t.after(() => page.close());
  await page.goto(server.url.href);
  await page.waitForFunction(() => typeof window.createApplicationDeliveryFixture === 'function');
  await page.evaluate(() => window.createApplicationDeliveryFixture());
  await page.getByRole('button', {name: 'Prepare company version', exact: true}).waitFor();
  await page.evaluate(() => {
    const app = window.applicationFixture.element,
      delivery = document.querySelector('soda-application-delivery') as unknown as {
        request: typeof app.request;
      },
      original = app.request!;
    const failing: typeof app.request = async (path, body, signal) => {
      if (body) {
        window.applicationFixture.calls.push({path, body});
        throw Object.assign(new Error('operator required'), {status: 403, code: 'operator_required'});
      }
      return original(path, body, signal);
    };
    app.request = failing;
    delivery.request = failing;
  });
  await page.getByRole('button', {name: 'Prepare company version', exact: true}).click();
  await page
    .getByText(
      'Preparing a delivery needs an operator. Ask an application operator to prepare it; ordinary accounts cannot prepare versions.',
      {exact: true}
    )
    .waitFor();
  // A permission refusal is never described as a session change.
  assert.equal(await page.getByText(/session changed/, {exact: false}).count(), 0);
  assert.equal(await page.getByRole('button', {name: 'Follow this delivery in Feed', exact: true}).count(), 0);
  assert.equal(await page.evaluate(() => window.applicationFixture.calls.filter((call) => call.body).length), 1);
  await page.waitForTimeout(300);
  assert.equal(await page.evaluate(() => window.applicationFixture.calls.filter((call) => call.body).length), 1);
});

test('unconfirmed apply with null observation stays unconfirmed without a stopped claim', async (t) => {
  const page = await browser.newPage();
  t.after(() => page.close());
  await page.goto(server.url.href);
  await page.waitForFunction(() => typeof window.createApplicationDeliveryFixture === 'function');
  await page.evaluate(() => window.createApplicationDeliveryFixture(true));
  await page.getByRole('button', {name: 'Prepare company version', exact: true}).click();
  const apply = page.getByRole('button', {name: 'Apply version to Example', exact: true});
  await apply.waitFor();
  await page.getByRole('checkbox', {name: 'Stop Example for everyone and apply this version', exact: true}).check();
  await page.evaluate(() => {
    const app = window.applicationFixture.element,
      delivery = document.querySelector('soda-application-delivery') as unknown as {
        request: typeof app.request;
      },
      original = app.request!;
    const losing: typeof app.request = async (path, body, signal) => {
      if (body?.action === 'replace') {
        // The server updates the spec but the reply is lost; the follow-up
        // observation never lands, so the re-read carries the new image with
        // observation=null.
        await original(path, body, signal);
        throw Error('lost reply');
      }
      const result = (await original(path, body, signal)) as {
        applications?: {observation: unknown}[];
      };
      if (!body && path === '/api/applications' && Array.isArray(result?.applications)) {
        const copy = JSON.parse(JSON.stringify(result)) as typeof result;
        if (copy.applications?.[0]) copy.applications[0].observation = null;
        return copy;
      }
      return result;
    };
    app.request = losing;
    delivery.request = losing;
  });
  await apply.click();
  await page
    .getByText(
      'The apply reply was unconfirmed, so the same application was re-read: the new version reference matched but runtime is unconfirmed with no observation. Refresh applications and inspect Example to reconcile the same apply before acting again.',
      {exact: true}
    )
    .waitFor();
  const notice = await page.evaluate(
    () => document.querySelector('soda-application-delivery [role="status"]')?.textContent ?? ''
  );
  assert.match(notice, /unconfirmed/);
  assert.match(notice, /reference matched but runtime is unconfirmed/);
  assert.equal(/stopp/i.test(notice), false);
  assert.equal(await page.getByText(/installed and stopped/, {exact: false}).count(), 0);
  assert.deepEqual(
    await page.evaluate(() => window.applicationFixture.calls.filter((call) => call.body?.action === 'replace')),
    [
      {
        path: `/api/applications/${'a' + '1'.repeat(24)}/actions`,
        body: {
          action: 'replace',
          revision: 1,
          expected_container: 'a'.repeat(64),
          confirm_stop: true,
          replacement_build: 'e'.repeat(32),
        },
      },
    ]
  );
  await page.waitForTimeout(300);
  assert.equal(
    await page.evaluate(() => window.applicationFixture.calls.filter((call) => call.body?.action === 'replace').length),
    1
  );
});

test('unconfirmed apply with concurrent running state shows running without a stopped claim', async (t) => {
  const page = await browser.newPage();
  t.after(() => page.close());
  await page.goto(server.url.href);
  await page.waitForFunction(() => typeof window.createApplicationDeliveryFixture === 'function');
  await page.evaluate(() => window.createApplicationDeliveryFixture(true));
  await page.getByRole('button', {name: 'Prepare company version', exact: true}).click();
  const apply = page.getByRole('button', {name: 'Apply version to Example', exact: true});
  await apply.waitFor();
  await page.getByRole('checkbox', {name: 'Stop Example for everyone and apply this version', exact: true}).check();
  await page.evaluate(() => {
    const app = window.applicationFixture.element,
      delivery = document.querySelector('soda-application-delivery') as unknown as {
        request: typeof app.request;
      },
      original = app.request!;
    const losing: typeof app.request = async (path, body, signal) => {
      if (body?.action === 'replace') {
        // The server applies the replacement but the reply is lost; a
        // concurrent Start leaves the re-read with running=true.
        await original(path, body, signal);
        throw Error('lost reply');
      }
      const result = (await original(path, body, signal)) as {
        applications?: {observation: Record<string, unknown> | null}[];
      };
      if (!body && path === '/api/applications' && Array.isArray(result?.applications)) {
        const copy = JSON.parse(JSON.stringify(result)) as typeof result;
        const row = copy.applications?.[0];
        if (row?.observation) Object.assign(row.observation, {container: 'b'.repeat(64), running: true, ready: true});
        return copy;
      }
      return result;
    };
    app.request = losing;
    delivery.request = losing;
  });
  await apply.click();
  await page
    .getByText(
      'The apply reply was unconfirmed, so the same application was re-read: the new version reference matched but the application is running. Refresh applications and inspect Example to reconcile the same apply before acting again.',
      {exact: true}
    )
    .waitFor();
  const notice = await page.evaluate(
    () => document.querySelector('soda-application-delivery [role="status"]')?.textContent ?? ''
  );
  assert.match(notice, /running/);
  assert.match(notice, /reference matched/);
  assert.equal(/stopp/i.test(notice), false);
  assert.equal(await page.getByText(/installed and stopped/, {exact: false}).count(), 0);
  assert.equal(
    await page.evaluate(() => window.applicationFixture.calls.filter((call) => call.body?.action === 'replace').length),
    1
  );
  await page.waitForTimeout(300);
  assert.equal(
    await page.evaluate(() => window.applicationFixture.calls.filter((call) => call.body?.action === 'replace').length),
    1
  );
});

test('unconfirmed apply with usable stopped observation confirms installed and stopped', async (t) => {
  const page = await browser.newPage();
  t.after(() => page.close());
  await page.goto(server.url.href);
  await page.waitForFunction(() => typeof window.createApplicationDeliveryFixture === 'function');
  await page.evaluate(() => window.createApplicationDeliveryFixture(true));
  await page.getByRole('button', {name: 'Prepare company version', exact: true}).click();
  const apply = page.getByRole('button', {name: 'Apply version to Example', exact: true});
  await apply.waitFor();
  await page.getByRole('checkbox', {name: 'Stop Example for everyone and apply this version', exact: true}).check();
  await page.evaluate(() => {
    const app = window.applicationFixture.element,
      delivery = document.querySelector('soda-application-delivery') as unknown as {
        request: typeof app.request;
      },
      original = app.request!;
    const losing: typeof app.request = async (path, body, signal) => {
      if (body?.action === 'replace') {
        // The server applies the replacement but the reply is lost; the
        // re-read carries a usable stopped observation for the new image.
        await original(path, body, signal);
        throw Error('lost reply');
      }
      return original(path, body, signal);
    };
    app.request = losing;
    delivery.request = losing;
  });
  await apply.click();
  await page
    .getByText(
      'The apply reply was unconfirmed, so the same application was re-read: the new version is installed and stopped. Start the company application, then check its retained records.',
      {exact: true}
    )
    .waitFor();
  const observed = await page.evaluate(async () => {
    const value = (await window.applicationFixture.element.request!('/api/applications', undefined)) as {
      applications: {application: {image: string}; observation: {running: boolean; ready: boolean} | null}[];
    };
    return value.applications[0];
  });
  assert.equal(observed?.application.image, 'sha256:' + '9'.repeat(64));
  assert.ok(observed?.observation);
  assert.equal(observed?.observation?.running, false);
  assert.equal(
    await page.evaluate(() => window.applicationFixture.calls.filter((call) => call.body?.action === 'replace').length),
    1
  );
  await page.waitForTimeout(300);
  assert.equal(
    await page.evaluate(() => window.applicationFixture.calls.filter((call) => call.body?.action === 'replace').length),
    1
  );
});

test('unconfirmed apply with empty-container observation stays not-installed without a stopped claim', async (t) => {
  const page = await browser.newPage();
  t.after(() => page.close());
  await page.goto(server.url.href);
  await page.waitForFunction(() => typeof window.createApplicationDeliveryFixture === 'function');
  await page.evaluate(() => window.createApplicationDeliveryFixture(true));
  await page.getByRole('button', {name: 'Prepare company version', exact: true}).click();
  const apply = page.getByRole('button', {name: 'Apply version to Example', exact: true});
  await apply.waitFor();
  await page.getByRole('checkbox', {name: 'Stop Example for everyone and apply this version', exact: true}).check();
  await page.evaluate(() => {
    const app = window.applicationFixture.element,
      delivery = document.querySelector('soda-application-delivery') as unknown as {
        request: typeof app.request;
      },
      original = app.request!;
    const losing: typeof app.request = async (path, body, signal) => {
      if (body?.action === 'replace') {
        // The server updates the spec but the reply is lost; the follow-up
        // observation never lands, so the re-read carries the new image with
        // an empty container (canonical not-installed: running=false,
        // ready=false).
        await original(path, body, signal);
        throw Error('lost reply');
      }
      const result = (await original(path, body, signal)) as {
        applications?: {observation: Record<string, unknown> | null}[];
      };
      if (!body && path === '/api/applications' && Array.isArray(result?.applications)) {
        const copy = JSON.parse(JSON.stringify(result)) as typeof result;
        const row = copy.applications?.[0];
        if (row?.observation) Object.assign(row.observation, {container: '', running: false, ready: false});
        return copy;
      }
      return result;
    };
    app.request = losing;
    delivery.request = losing;
  });
  await apply.click();
  await page
    .getByText(
      'The apply reply was unconfirmed, so the same application was re-read: the new version reference matched but the application is not installed. Refresh applications and inspect Example to reconcile the same apply before acting again.',
      {exact: true}
    )
    .waitFor();
  const notice = await page.evaluate(
    () => document.querySelector('soda-application-delivery [role="status"]')?.textContent ?? ''
  );
  assert.match(notice, /not installed/);
  assert.match(notice, /reconcile the same apply/);
  assert.equal(/stopp/i.test(notice), false);
  assert.equal(await page.getByText(/installed and stopped/, {exact: false}).count(), 0);
  assert.deepEqual(
    await page.evaluate(() => window.applicationFixture.calls.filter((call) => call.body?.action === 'replace')),
    [
      {
        path: `/api/applications/${'a' + '1'.repeat(24)}/actions`,
        body: {
          action: 'replace',
          revision: 1,
          expected_container: 'a'.repeat(64),
          confirm_stop: true,
          replacement_build: 'e'.repeat(32),
        },
      },
    ]
  );
  await page.waitForTimeout(300);
  assert.equal(
    await page.evaluate(() => window.applicationFixture.calls.filter((call) => call.body?.action === 'replace').length),
    1
  );
});

test('completed setup recovers with one inventory refresh and a Start handoff', async (t) => {
  const page = await browser.newPage();
  t.after(() => page.close());
  await page.goto(server.url.href);
  await page.waitForFunction(() => typeof window.createApplicationDeliveryFixture === 'function');
  await page.evaluate(() => window.createApplicationDeliveryFixture());
  await page.getByRole('button', {name: 'Prepare company version', exact: true}).click();
  await page.getByRole('button', {name: 'Set up company application', exact: true}).waitFor();
  await page.evaluate(() => {
    const app = window.applicationFixture.element,
      delivery = document.querySelector('soda-application-delivery') as unknown as {
        request: typeof app.request;
      },
      original = app.request!;
    const completedRecord = {
      environment: 'company',
      application: {
        id: 'a' + '1'.repeat(24),
        revision: 1,
        name: 'Example',
        artifact_sha256: '2'.repeat(64),
        bundle_sha256: '3'.repeat(64),
        tools_sha256: '4'.repeat(64),
        runtime_image: 'sha256:' + '5'.repeat(64),
        image: 'sha256:' + '6'.repeat(64),
        data: 'a' + '7'.repeat(24),
        owner_team: '12345678-1234-1234-1234-123456789abc',
        origin: 'https://app.example.test',
        port: 20001,
      },
      administrator_id: '41',
      operator: true,
      observation: {
        id: 'a' + '1'.repeat(24),
        revision: 1,
        container: 'a'.repeat(64),
        image: 'sha256:' + '6'.repeat(64),
        origin: 'https://app.example.test',
        running: false,
        ready: false,
        bundle_file_sha256: 'b'.repeat(64),
      },
      project: {repository: '7', name: 'northstar/equipment', path: '/northstar/equipment'},
      delivery: {
        environment: 'company',
        state: 'verified',
        intent: 'intent-current',
        revision: 'revision-current',
        delivery_id: 'b'.repeat(32),
        run: 'run-current',
        source_commit: 'c'.repeat(40),
        bundle_file_sha256: 'b'.repeat(64),
      },
    };
    let setupDone = false;
    const completed: typeof app.request = async (path, body, signal) => {
      if (path.endsWith('/setup') && body) {
        // Continue owner setup returns a completed record.
        setupDone = true;
        window.applicationFixture.calls.push({path, body});
        return {
          id: 'a' + 'c'.repeat(24),
          state: 'complete',
          application: {id: 'a' + '1'.repeat(24), name: 'Example', origin: 'https://app.example.test'},
        };
      }
      if (setupDone && path === '/api/applications' && !body) {
        window.applicationFixture.calls.push({path});
        return {applications: [completedRecord], launchable: [], complete: true, can_setup: true};
      }
      if (setupDone && body?.action === 'inspect') {
        // Status reads echo the addressed record; anything else would be a product write.
        window.applicationFixture.calls.push({path, body});
        return completedRecord;
      }
      return original(path, body, signal);
    };
    app.request = completed;
    delivery.request = completed;
  });
  // Replacing the request triggers one property-change refresh; let it settle
  // against the still-empty inventory before measuring the recovery refresh.
  await page.waitForTimeout(300);
  const inventoryReads = () =>
    page.evaluate(
      () => window.applicationFixture.calls.filter((call) => !call.body && call.path === '/api/applications').length
    );
  const setupPosts = () =>
    page.evaluate(
      () => window.applicationFixture.calls.filter((call) => !!call.body && call.path.endsWith('/setup')).length
    );
  const readsBefore = await inventoryReads();
  await page.getByRole('button', {name: 'Set up company application', exact: true}).click();
  await page
    .getByText(
      'Setup complete. The application is installed and stopped. Start the application, then sign in at its address.',
      {exact: true}
    )
    .waitFor();
  await page.getByRole('button', {name: 'Start Example', exact: true}).waitFor();
  // The delivery selection survives the recovery.
  assert.equal(await page.evaluate(() => window.applicationFixture.element.requestedDelivery), 'b'.repeat(32));
  // Exactly one inventory refresh follows the single setup submission: no duplicate provisioning.
  assert.equal(await inventoryReads(), readsBefore + 1);
  assert.equal(await setupPosts(), 1);
  await page.waitForTimeout(300);
  assert.equal(await inventoryReads(), readsBefore + 1);
  assert.equal(await setupPosts(), 1);
  assert.equal(
    await page.evaluate(() => window.applicationFixture.calls.filter((call) => call.body?.action === 'start').length),
    0
  );
});

test('completed setup reports a running application truthfully without a stopped claim', async (t) => {
  const page = await browser.newPage();
  t.after(() => page.close());
  await page.goto(server.url.href);
  await page.waitForFunction(() => typeof window.createApplicationDeliveryFixture === 'function');
  await page.evaluate(() => window.createApplicationDeliveryFixture());
  await page.getByRole('button', {name: 'Prepare company version', exact: true}).click();
  await page.getByRole('button', {name: 'Set up company application', exact: true}).waitFor();
  await page.evaluate(() => {
    const app = window.applicationFixture.element,
      delivery = document.querySelector('soda-application-delivery') as unknown as {
        request: typeof app.request;
      },
      original = app.request!;
    const completedRecord = {
      environment: 'company',
      application: {
        id: 'a' + '1'.repeat(24),
        revision: 1,
        name: 'Example',
        artifact_sha256: '2'.repeat(64),
        bundle_sha256: '3'.repeat(64),
        tools_sha256: '4'.repeat(64),
        runtime_image: 'sha256:' + '5'.repeat(64),
        image: 'sha256:' + '6'.repeat(64),
        data: 'a' + '7'.repeat(24),
        owner_team: '12345678-1234-1234-1234-123456789abc',
        origin: 'https://app.example.test',
        port: 20001,
      },
      administrator_id: '41',
      operator: true,
      observation: {
        id: 'a' + '1'.repeat(24),
        revision: 1,
        container: 'a'.repeat(64),
        image: 'sha256:' + '6'.repeat(64),
        origin: 'https://app.example.test',
        running: true,
        ready: true,
        bundle_file_sha256: 'b'.repeat(64),
      },
      project: {repository: '7', name: 'northstar/equipment', path: '/northstar/equipment'},
      delivery: {
        environment: 'company',
        state: 'verified',
        intent: 'intent-current',
        revision: 'revision-current',
        delivery_id: 'b'.repeat(32),
        run: 'run-current',
        source_commit: 'c'.repeat(40),
        bundle_file_sha256: 'b'.repeat(64),
      },
    };
    let setupDone = false;
    const completed: typeof app.request = async (path, body, signal) => {
      if (path.endsWith('/setup') && body) {
        // Continue owner setup returns a completed record.
        setupDone = true;
        window.applicationFixture.calls.push({path, body});
        return {
          id: 'a' + 'c'.repeat(24),
          state: 'complete',
          application: {id: 'a' + '1'.repeat(24), name: 'Example', origin: 'https://app.example.test'},
        };
      }
      if (setupDone && path === '/api/applications' && !body) {
        window.applicationFixture.calls.push({path});
        return {applications: [completedRecord], launchable: [], complete: true, can_setup: true};
      }
      if (setupDone && body?.action === 'inspect') {
        // Status reads echo the addressed record; anything else would be a product write.
        window.applicationFixture.calls.push({path, body});
        return completedRecord;
      }
      return original(path, body, signal);
    };
    app.request = completed;
    delivery.request = completed;
  });
  // Replacing the request triggers one property-change refresh; let it settle
  // against the still-empty inventory before measuring the recovery refresh.
  await page.waitForTimeout(300);
  const inventoryReads = () =>
    page.evaluate(
      () => window.applicationFixture.calls.filter((call) => !call.body && call.path === '/api/applications').length
    );
  const setupPosts = () =>
    page.evaluate(
      () => window.applicationFixture.calls.filter((call) => !!call.body && call.path.endsWith('/setup')).length
    );
  const readsBefore = await inventoryReads();
  await page.getByRole('button', {name: 'Set up company application', exact: true}).click();
  await page
    .getByText('Setup complete. The application is running and ready. Sign in at its address to use it.', {
      exact: true,
    })
    .waitFor();
  await page.getByText('Last observed: Ready', {exact: true}).waitFor();
  const notice = await page.evaluate(() => {
    const root = window.applicationFixture.element.renderRoot ?? window.applicationFixture.element;
    return root?.querySelector('.soda-applications-status')?.textContent ?? '';
  });
  assert.match(notice, /running and ready/);
  assert.equal(/stopp/i.test(notice), false);
  assert.equal(await page.getByText(/installed and stopped/, {exact: false}).count(), 0);
  // A running application cannot be started again and setup never auto-starts it.
  // The runtime section stays closed while running, so open it before asserting.
  await page.getByText('Start or stop application', {exact: true}).click();
  assert(await page.getByRole('button', {name: 'Start Example', exact: true}).isDisabled());
  // The delivery selection survives the recovery.
  assert.equal(await page.evaluate(() => window.applicationFixture.element.requestedDelivery), 'b'.repeat(32));
  // Exactly one inventory refresh follows the single setup submission: no duplicate provisioning.
  assert.equal(await inventoryReads(), readsBefore + 1);
  assert.equal(await setupPosts(), 1);
  await page.waitForTimeout(300);
  assert.equal(await inventoryReads(), readsBefore + 1);
  assert.equal(await setupPosts(), 1);
  assert.equal(
    await page.evaluate(() => window.applicationFixture.calls.filter((call) => call.body?.action === 'start').length),
    0
  );
});

test('completion during an in-flight inventory read queues one refresh before resolving', async (t) => {
  const page = await browser.newPage();
  t.after(() => page.close());
  await page.goto(server.url.href);
  await page.waitForFunction(() => typeof window.createApplicationDeliveryFixture === 'function');
  await page.evaluate(() => window.createApplicationDeliveryFixture());
  await page.getByRole('button', {name: 'Prepare company version', exact: true}).click();
  await page.getByRole('button', {name: 'Set up company application', exact: true}).waitFor();
  await page.evaluate(() => {
    const app = window.applicationFixture.element,
      delivery = document.querySelector('soda-application-delivery') as unknown as {
        request: typeof app.request;
      },
      original = app.request!;
    const completedRecord = {
      environment: 'company',
      application: {
        id: 'a' + '1'.repeat(24),
        revision: 1,
        name: 'Example',
        artifact_sha256: '2'.repeat(64),
        bundle_sha256: '3'.repeat(64),
        tools_sha256: '4'.repeat(64),
        runtime_image: 'sha256:' + '5'.repeat(64),
        image: 'sha256:' + '6'.repeat(64),
        data: 'a' + '7'.repeat(24),
        owner_team: '12345678-1234-1234-1234-123456789abc',
        origin: 'https://app.example.test',
        port: 20001,
      },
      administrator_id: '41',
      operator: true,
      observation: {
        id: 'a' + '1'.repeat(24),
        revision: 1,
        container: 'a'.repeat(64),
        image: 'sha256:' + '6'.repeat(64),
        origin: 'https://app.example.test',
        running: false,
        ready: false,
        bundle_file_sha256: 'b'.repeat(64),
      },
      project: {repository: '7', name: 'northstar/equipment', path: '/northstar/equipment'},
      delivery: {
        environment: 'company',
        state: 'verified',
        intent: 'intent-current',
        revision: 'revision-current',
        delivery_id: 'b'.repeat(32),
        run: 'run-current',
        source_commit: 'c'.repeat(40),
        bundle_file_sha256: 'b'.repeat(64),
      },
    };
    let setupDone = false;
    (
      window.applicationFixture as unknown as {completionGate: {hold: boolean; waiters: (() => void)[]}}
    ).completionGate = {hold: false, waiters: []};
    const completed: typeof app.request = async (path, body, signal) => {
      if (path.endsWith('/setup') && body) {
        // Continue owner setup returns a completed record.
        setupDone = true;
        window.applicationFixture.calls.push({path, body});
        return {
          id: 'a' + 'c'.repeat(24),
          state: 'complete',
          application: {id: 'a' + '1'.repeat(24), name: 'Example', origin: 'https://app.example.test'},
        };
      }
      if (path === '/api/applications' && !body) {
        // The read observes the inventory as of its dispatch: a read sent
        // before the setup completed still lands empty.
        const doneAtCall = setupDone;
        window.applicationFixture.calls.push({path});
        const gate = (
          window.applicationFixture as unknown as {completionGate: {hold: boolean; waiters: (() => void)[]}}
        ).completionGate;
        if (gate.hold) await new Promise<void>((resolve) => gate.waiters.push(resolve));
        return doneAtCall
          ? {applications: [completedRecord], launchable: [], complete: true, can_setup: true}
          : {applications: [], launchable: [], complete: true, can_setup: true};
      }
      if (setupDone && body?.action === 'inspect') {
        // Status reads echo the addressed record; anything else would be a product write.
        window.applicationFixture.calls.push({path, body});
        return completedRecord;
      }
      return original(path, body, signal);
    };
    app.request = completed;
    delivery.request = completed;
  });
  // Replacing the request triggers one property-change refresh; let it settle
  // against the still-empty inventory before measuring the recovery refresh.
  await page.waitForTimeout(300);
  const inventoryReads = () =>
    page.evaluate(
      () => window.applicationFixture.calls.filter((call) => !call.body && call.path === '/api/applications').length
    );
  const readsBefore = await inventoryReads();
  await page.evaluate(() => {
    (
      window.applicationFixture as unknown as {completionGate: {hold: boolean; waiters: (() => void)[]}}
    ).completionGate.hold = true;
  });
  await page.getByRole('button', {name: 'Refresh applications', exact: true}).click();
  await page.waitForFunction(
    (expected) =>
      window.applicationFixture.calls.filter((call) => !call.body && call.path === '/api/applications').length ===
      expected,
    readsBefore + 1
  );
  // The setup completes while that inventory read is still in flight.
  await page.getByRole('button', {name: 'Set up company application', exact: true}).click();
  await page.waitForFunction(() =>
    window.applicationFixture.calls.some((call) => !!call.body && call.path.endsWith('/setup'))
  );
  await page.waitForTimeout(100);
  // No success may be claimed on the stale (still empty) inventory, and the
  // queued refresh may not start until the in-flight read lands.
  assert.equal(await page.getByText(/Setup complete/, {exact: false}).count(), 0);
  assert.equal(await inventoryReads(), readsBefore + 1);
  assert.equal(await page.getByRole('button', {name: 'Start Example', exact: true}).count(), 0);
  await page.evaluate(() => {
    const gate = (window.applicationFixture as unknown as {completionGate: {hold: boolean; waiters: (() => void)[]}})
      .completionGate;
    gate.hold = false;
    gate.waiters.splice(0).forEach((resolve) => resolve());
  });
  // The in-flight read lands empty, then exactly one queued refresh resolves
  // the completion against the actual stopped observation.
  await page
    .getByText(
      'Setup complete. The application is installed and stopped. Start the application, then sign in at its address.',
      {exact: true}
    )
    .waitFor();
  await page.getByRole('button', {name: 'Start Example', exact: true}).waitFor();
  // The delivery selection survives the recovery.
  assert.equal(await page.evaluate(() => window.applicationFixture.element.requestedDelivery), 'b'.repeat(32));
  // The in-flight read plus exactly one queued refresh: no dropped read, no refresh loop.
  assert.equal(await inventoryReads(), readsBefore + 2);
  assert.equal(
    await page.evaluate(
      () => window.applicationFixture.calls.filter((call) => !!call.body && call.path.endsWith('/setup')).length
    ),
    1
  );
  await page.waitForTimeout(300);
  assert.equal(await inventoryReads(), readsBefore + 2);
  assert.equal(
    await page.evaluate(() => window.applicationFixture.calls.filter((call) => call.body?.action === 'start').length),
    0
  );
});

test('shared delivery completion reports the exact company application, never preview', async (t) => {
  const page = await browser.newPage();
  t.after(() => page.close());
  await page.goto(server.url.href);
  await page.waitForFunction(() => typeof window.createApplicationDeliveryFixture === 'function');
  await page.evaluate(() => window.createApplicationDeliveryFixture());
  await page.getByRole('button', {name: 'Prepare company version', exact: true}).click();
  await page.getByRole('button', {name: 'Set up company application', exact: true}).waitFor();
  await page.evaluate(() => {
    const app = window.applicationFixture.element,
      delivery = document.querySelector('soda-application-delivery') as unknown as {
        request: typeof app.request;
      },
      original = app.request!;
    const preview = {
      environment: 'preview',
      application: {
        id: 'a' + '2'.repeat(24),
        revision: 1,
        name: 'Example',
        artifact_sha256: '2'.repeat(64),
        bundle_sha256: '3'.repeat(64),
        tools_sha256: '4'.repeat(64),
        runtime_image: 'sha256:' + '5'.repeat(64),
        image: 'sha256:' + '6'.repeat(64),
        data: 'a' + '7'.repeat(24),
        owner_team: '12345678-1234-1234-1234-123456789abc',
        origin: 'https://preview.example.test',
        port: 20002,
      },
      administrator_id: '41',
      operator: false,
      observation: {
        id: 'a' + '2'.repeat(24),
        revision: 1,
        container: 'b'.repeat(64),
        image: 'sha256:' + '6'.repeat(64),
        origin: 'https://preview.example.test',
        running: false,
        ready: false,
        bundle_file_sha256: 'b'.repeat(64),
      },
      project: {repository: '7', name: 'northstar/equipment', path: '/northstar/equipment'},
      delivery: {
        environment: 'preview',
        state: 'verified',
        intent: 'intent-current',
        revision: 'revision-current',
        delivery_id: 'b'.repeat(32),
        run: 'run-current',
        source_commit: 'c'.repeat(40),
        bundle_file_sha256: 'b'.repeat(64),
      },
    };
    const company = {
      environment: 'company',
      application: {
        id: 'a' + '1'.repeat(24),
        revision: 1,
        name: 'Example',
        artifact_sha256: '2'.repeat(64),
        bundle_sha256: '3'.repeat(64),
        tools_sha256: '4'.repeat(64),
        runtime_image: 'sha256:' + '5'.repeat(64),
        image: 'sha256:' + '6'.repeat(64),
        data: 'a' + '7'.repeat(24),
        owner_team: '12345678-1234-1234-1234-123456789abc',
        origin: 'https://app.example.test',
        port: 20001,
      },
      administrator_id: '41',
      operator: false,
      observation: {
        id: 'a' + '1'.repeat(24),
        revision: 1,
        container: 'a'.repeat(64),
        image: 'sha256:' + '6'.repeat(64),
        origin: 'https://app.example.test',
        running: true,
        ready: true,
        bundle_file_sha256: 'b'.repeat(64),
      },
      project: {repository: '7', name: 'northstar/equipment', path: '/northstar/equipment'},
      delivery: {
        environment: 'company',
        state: 'verified',
        intent: 'intent-current',
        revision: 'revision-current',
        delivery_id: 'b'.repeat(32),
        run: 'run-current',
        source_commit: 'c'.repeat(40),
        bundle_file_sha256: 'b'.repeat(64),
      },
    };
    let setupDone = false;
    const shared: typeof app.request = async (path, body, signal) => {
      if (path.endsWith('/setup') && body) {
        // The company setup completes with the company application identity,
        // even though preview shares the same delivery.
        setupDone = true;
        window.applicationFixture.calls.push({path, body});
        return {
          id: 'a' + 'c'.repeat(24),
          state: 'complete',
          application: {id: 'a' + '1'.repeat(24), name: 'Example', origin: 'https://app.example.test'},
        };
      }
      if (!body && path === '/api/applications') {
        window.applicationFixture.calls.push({path});
        // Preview sorts first: a delivery-only guess would report preview.
        return {
          applications: setupDone ? [preview, company] : [preview],
          launchable: [],
          complete: true,
          can_setup: true,
        };
      }
      if (setupDone && body?.action === 'inspect') {
        // Status reads echo the addressed record; anything else would be a product write.
        window.applicationFixture.calls.push({path, body});
        const rows = [preview, company];
        return rows.find((row) => path === `/api/applications/${row.application.id}/actions`);
      }
      return original(path, body, signal);
    };
    app.request = shared;
    delivery.request = shared;
  });
  // Replacing the request triggers one property-change refresh; let it settle
  // against the preview-only inventory before measuring the recovery refresh.
  await page.waitForTimeout(300);
  const inventoryReads = () =>
    page.evaluate(
      () => window.applicationFixture.calls.filter((call) => !call.body && call.path === '/api/applications').length
    );
  const setupPosts = () =>
    page.evaluate(
      () => window.applicationFixture.calls.filter((call) => !!call.body && call.path.endsWith('/setup')).length
    );
  const readsBefore = await inventoryReads();
  await page.getByRole('button', {name: 'Set up company application', exact: true}).click();
  await page
    .getByText('Setup complete. The application is running and ready. Sign in at its address to use it.', {
      exact: true,
    })
    .waitFor();
  // The notice names the running company record, never the stopped preview:
  // no stopped claim appears in the completion status.
  const status = await page.evaluate(
    () =>
      (window.applicationFixture.element.renderRoot ?? window.applicationFixture.element)?.querySelector(
        '.soda-applications-status'
      )?.textContent ?? ''
  );
  assert.match(status, /running and ready/);
  assert.equal(/stopp/i.test(status), false);
  assert.equal(await setupPosts(), 1);
  assert.equal(await inventoryReads(), readsBefore + 1);
  await page.waitForTimeout(300);
  assert.equal(await inventoryReads(), readsBefore + 1);
  assert.equal(await setupPosts(), 1);
  assert.equal(
    await page.evaluate(() => window.applicationFixture.calls.filter((call) => call.body?.action === 'start').length),
    0
  );
});

test('completion without an application identity stays unconfirmed instead of guessing', async (t) => {
  const page = await browser.newPage();
  t.after(() => page.close());
  await page.goto(server.url.href);
  await page.waitForFunction(() => typeof window.createApplicationDeliveryFixture === 'function');
  await page.evaluate(() => window.createApplicationDeliveryFixture());
  await page.evaluate(() => {
    const app = window.applicationFixture.element,
      original = app.request!;
    const company = {
      environment: 'company',
      application: {
        id: 'a' + '1'.repeat(24),
        revision: 1,
        name: 'Example',
        artifact_sha256: '2'.repeat(64),
        bundle_sha256: '3'.repeat(64),
        tools_sha256: '4'.repeat(64),
        runtime_image: 'sha256:' + '5'.repeat(64),
        image: 'sha256:' + '6'.repeat(64),
        data: 'a' + '7'.repeat(24),
        owner_team: '12345678-1234-1234-1234-123456789abc',
        origin: 'https://app.example.test',
        port: 20001,
      },
      administrator_id: '41',
      operator: false,
      observation: {
        id: 'a' + '1'.repeat(24),
        revision: 1,
        container: 'a'.repeat(64),
        image: 'sha256:' + '6'.repeat(64),
        origin: 'https://app.example.test',
        running: false,
        ready: false,
        bundle_file_sha256: 'b'.repeat(64),
      },
      project: {repository: '7', name: 'northstar/equipment', path: '/northstar/equipment'},
      delivery: {
        environment: 'company',
        state: 'verified',
        intent: 'intent-current',
        revision: 'revision-current',
        delivery_id: 'b'.repeat(32),
        run: 'run-current',
        source_commit: 'c'.repeat(40),
        bundle_file_sha256: 'b'.repeat(64),
      },
    };
    const preview = {
      ...company,
      environment: 'preview',
      application: {...company.application, id: 'a' + '2'.repeat(24), origin: 'https://preview.example.test'},
      observation: {
        ...company.observation,
        id: 'a' + '2'.repeat(24),
        container: 'b'.repeat(64),
        origin: 'https://preview.example.test',
        running: true,
        ready: true,
      },
      delivery: {...company.delivery, environment: 'preview'},
    };
    const legacy: typeof app.request = async (path, body, signal) => {
      if (!body && path === '/api/applications') {
        window.applicationFixture.calls.push({path});
        return {applications: [company, preview], launchable: [], complete: true, can_setup: true};
      }
      if (body?.action === 'inspect') {
        window.applicationFixture.calls.push({path, body});
        const rows = [company, preview];
        return rows.find((row) => path === `/api/applications/${row.application.id}/actions`);
      }
      return original(path, body, signal);
    };
    app.request = legacy;
    (document.querySelector('soda-application-delivery') as unknown as {request: typeof app.request}).request = legacy;
    // No selected delivery: a delivery-only guess could report either record.
    app.requestedDelivery = '';
  });
  await page.locator('.soda-application-heading').getByText('Example', {exact: true}).first().waitFor();
  const readsBefore = await page.evaluate(
    () => window.applicationFixture.calls.filter((call) => !call.body && call.path === '/api/applications').length
  );
  // An older emitter reports completion without any application identity.
  await page.evaluate(() => {
    (document.querySelector('soda-application-delivery') as HTMLElement).dispatchEvent(
      new CustomEvent('application-setup-complete', {bubbles: true, composed: true})
    );
  });
  await page
    .getByText(
      'Setup complete. The completed application could not be confirmed from the current inventory. Refresh applications and inspect it before acting again.',
      {exact: true}
    )
    .waitFor();
  // Unconfirmed means unconfirmed: neither the stopped company state nor the
  // running preview state is claimed in the completion status.
  const status = await page.evaluate(
    () =>
      (window.applicationFixture.element.renderRoot ?? window.applicationFixture.element)?.querySelector(
        '.soda-applications-status'
      )?.textContent ?? ''
  );
  assert.match(status, /could not be confirmed/);
  assert.equal(/stopp/i.test(status), false);
  assert.equal(/running/i.test(status), false);
  assert.equal(await page.evaluate(() => window.applicationFixture.element.requestedDelivery), '');
  assert.equal(
    await page.evaluate(
      () => window.applicationFixture.calls.filter((call) => !call.body && call.path === '/api/applications').length
    ),
    readsBefore + 1
  );
  await page.waitForTimeout(300);
  assert.equal(
    await page.evaluate(() => window.applicationFixture.calls.filter((call) => call.body?.action === 'start').length),
    0
  );
  assert.equal(
    await page.evaluate(
      () => window.applicationFixture.calls.filter((call) => !!call.body && call.body.action !== 'inspect').length
    ),
    0
  );
});

test('completed owner setup reports running truth while the application runs', async (t) => {
  const page = await browser.newPage();
  t.after(() => page.close());
  const errors: string[] = [];
  page.on('pageerror', (error) => errors.push(error.message));
  await page.goto(server.url.href);
  await page.waitForFunction(() => typeof window.createApplicationFixture === 'function');
  await page.evaluate(() => window.createApplicationFixture(true));
  await page.getByText('Applications', {exact: true}).click();
  await page.waitForFunction(() => window.applicationFixture.calls.some((c) => c.body?.action === 'inspect'));
  await page.getByText('Prepare or update an application', {exact: true}).click();
  await page.getByText('Set up a new application (operator)', {exact: true}).click();
  await page.getByLabel('Admitted application references', {exact: true}).fill(
    JSON.stringify({
      name: 'NewApp',
      artifact_sha256: '1'.repeat(64),
      bundle_sha256: '2'.repeat(64),
      tools_sha256: '3'.repeat(64),
      runtime_image: 'sha256:' + '4'.repeat(64),
      image: 'sha256:' + '5'.repeat(64),
      origin: 'https://new.example.test',
      port: 24000,
    })
  );
  await page.getByLabel('Soda administrator user ID', {exact: true}).fill('41');
  await page.getByRole('button', {name: 'Begin owner setup', exact: true}).click();
  await page.getByRole('link', {name: 'Create application owner ↗', exact: true}).waitFor();
  await page.evaluate(() => {
    const app = window.applicationFixture.element,
      setup = document.querySelector('soda-application-setup') as unknown as {request: typeof app.request} | null,
      original = app.request!;
    // Canonical settlement retains a stopped provisional observation; the
    // current application may already run. The child stays neutral and the
    // exact-app parent observation supplies all runtime truth.
    const completedId = 'a' + 'd'.repeat(24);
    const runningRecord = {
      environment: 'company',
      application: {
        id: completedId,
        revision: 1,
        name: 'NewApp',
        artifact_sha256: '1'.repeat(64),
        bundle_sha256: '2'.repeat(64),
        tools_sha256: '3'.repeat(64),
        runtime_image: 'sha256:' + '4'.repeat(64),
        image: 'sha256:' + '5'.repeat(64),
        data: 'a' + '7'.repeat(24),
        owner_team: '12345678-1234-1234-1234-123456789abc',
        origin: 'https://new.example.test',
        port: 24000,
      },
      administrator_id: '41',
      operator: true,
      observation: {
        id: completedId,
        revision: 1,
        container: 'e'.repeat(64),
        image: 'sha256:' + '5'.repeat(64),
        origin: 'https://new.example.test',
        running: true,
        ready: true,
      },
    };
    let setupDone = false;
    const truthful: typeof app.request = async (path, body, signal) => {
      if (path.startsWith('/api/applications/setups/') && body && (body as {action?: string}).action === 'complete') {
        setupDone = true;
        window.applicationFixture.calls.push({path, body});
        const id = path.split('/')[4] as string;
        return {
          setup: {
            id,
            application: {id: completedId, name: 'NewApp', origin: 'https://new.example.test'},
            administrator_id: 41,
            expires_at: Math.floor(Date.now() / 1000) + 600,
            state: 'complete',
            observation: {container: 'c'.repeat(64), running: false, ready: false},
          },
          result: {operation: id},
        };
      }
      if (setupDone && !body && path === '/api/applications') {
        window.applicationFixture.calls.push({path});
        return {applications: [runningRecord], launchable: [], complete: true, can_setup: true};
      }
      if (setupDone && body && (body as {action?: string}).action === 'inspect') {
        window.applicationFixture.calls.push({path, body});
        return runningRecord;
      }
      return original(path, body, signal);
    };
    app.request = truthful;
    if (setup) setup.request = truthful;
  });
  await page.getByRole('button', {name: 'Confirm owner and finish NewApp', exact: true}).click();
  await page
    .getByText(
      'Owner confirmed. Setup complete. Refresh applications to reconcile the completed setup before starting it.',
      {
        exact: true,
      }
    )
    .waitFor();
  const childStatus = await page.evaluate(
    () => document.querySelector('soda-application-setup')?.querySelector('[role="status"]')?.textContent ?? ''
  );
  assert.match(childStatus, /Setup complete/);
  assert.equal(/stopp/i.test(childStatus), false);
  // The exact-app parent supplies runtime truth: running, never stopped.
  await page.getByText('Last observed: Ready', {exact: true}).waitFor();
  await page
    .getByText('Setup complete. The application is running and ready. Sign in at its address to use it.', {
      exact: true,
    })
    .waitFor();
  assert.equal(await page.getByText(/installed and stopped/, {exact: false}).count(), 0);
  const parentStatus = await page.evaluate(
    () =>
      (window.applicationFixture.element.renderRoot ?? window.applicationFixture.element)?.querySelector(
        '.soda-applications-status'
      )?.textContent ?? ''
  );
  assert.match(parentStatus, /running and ready/);
  assert.equal(/stopp/i.test(parentStatus), false);
  assert.equal(
    await page.evaluate(
      () => window.applicationFixture.calls.filter((call) => call.body?.action === 'complete').length
    ),
    1
  );
  assert.equal(
    await page.evaluate(() => window.applicationFixture.calls.filter((call) => call.body?.action === 'start').length),
    0
  );
  assert.deepEqual(errors, []);
});

test('a completion during the coalesced re-read resolves the latest identity without extra setup', async (t) => {
  const page = await browser.newPage();
  t.after(() => page.close());
  await page.goto(server.url.href);
  await page.waitForFunction(() => typeof window.createApplicationDeliveryFixture === 'function');
  await page.evaluate(() => window.createApplicationDeliveryFixture());
  await page.getByRole('button', {name: 'Prepare company version', exact: true}).click();
  await page.getByRole('button', {name: 'Set up company application', exact: true}).waitFor();
  await page.evaluate(() => {
    const app = window.applicationFixture.element,
      delivery = document.querySelector('soda-application-delivery') as unknown as {
        request: typeof app.request;
      },
      original = app.request!;
    const recordFor = (suffix: string, running: boolean) => ({
      environment: 'company',
      application: {
        id: 'a' + suffix.repeat(24),
        revision: 1,
        name: 'Example',
        artifact_sha256: '2'.repeat(64),
        bundle_sha256: '3'.repeat(64),
        tools_sha256: '4'.repeat(64),
        runtime_image: 'sha256:' + '5'.repeat(64),
        image: 'sha256:' + '6'.repeat(64),
        data: 'a' + '7'.repeat(24),
        owner_team: '12345678-1234-1234-1234-123456789abc',
        origin: 'https://app.example.test',
        port: 20001,
      },
      administrator_id: '41',
      operator: false,
      observation: {
        id: 'a' + suffix.repeat(24),
        revision: 1,
        container: suffix.repeat(64),
        image: 'sha256:' + '6'.repeat(64),
        origin: 'https://app.example.test',
        running,
        ready: running,
        bundle_file_sha256: 'b'.repeat(64),
      },
      project: {repository: '7', name: 'northstar/equipment', path: '/northstar/equipment'},
      delivery: {
        environment: 'company',
        state: 'verified',
        intent: 'intent-current',
        revision: 'revision-current',
        delivery_id: 'b'.repeat(32),
        run: 'run-current',
        source_commit: 'c'.repeat(40),
        bundle_file_sha256: 'b'.repeat(64),
      },
    });
    let setups = 0;
    (
      window.applicationFixture as unknown as {completionGate: {hold: boolean; waiters: (() => void)[]}}
    ).completionGate = {hold: false, waiters: []};
    const queued: typeof app.request = async (path, body, signal) => {
      if (path.endsWith('/setup') && body) {
        // Each setup completes with its own application identity.
        setups++;
        const suffix = String(setups);
        window.applicationFixture.calls.push({path, body});
        return {
          id: 'a' + 'c'.repeat(24),
          state: 'complete',
          application: {id: 'a' + suffix.repeat(24), name: 'Example', origin: 'https://app.example.test'},
        };
      }
      if (path === '/api/applications' && !body) {
        // The read observes the inventory as of its dispatch.
        const seen = setups;
        window.applicationFixture.calls.push({path});
        const gate = (
          window.applicationFixture as unknown as {completionGate: {hold: boolean; waiters: (() => void)[]}}
        ).completionGate;
        if (gate.hold) await new Promise<void>((resolve) => gate.waiters.push(resolve));
        const applications = [];
        if (seen >= 1) applications.push(recordFor('1', false));
        if (seen >= 2) applications.push(recordFor('2', false));
        if (seen >= 3) applications.push(recordFor('3', true));
        return {applications, launchable: [], complete: true, can_setup: true};
      }
      if (setups > 0 && body?.action === 'inspect') {
        // Status reads echo the addressed record; anything else would be a product write.
        window.applicationFixture.calls.push({path, body});
        const rows = [recordFor('1', false), recordFor('2', false), recordFor('3', true)];
        return rows.find((row) => path === `/api/applications/${row.application.id}/actions`);
      }
      return original(path, body, signal);
    };
    app.request = queued;
    delivery.request = queued;
  });
  // Replacing the request triggers one property-change refresh; let it settle
  // against the still-empty inventory before measuring the recovery refresh.
  await page.waitForTimeout(300);
  const inventoryReads = () =>
    page.evaluate(
      () => window.applicationFixture.calls.filter((call) => !call.body && call.path === '/api/applications').length
    );
  const setupPosts = () =>
    page.evaluate(
      () => window.applicationFixture.calls.filter((call) => !!call.body && call.path.endsWith('/setup')).length
    );
  const readsBefore = await inventoryReads();
  await page.evaluate(() => {
    (
      window.applicationFixture as unknown as {completionGate: {hold: boolean; waiters: (() => void)[]}}
    ).completionGate.hold = true;
  });
  await page.getByRole('button', {name: 'Refresh applications', exact: true}).click();
  await page.waitForFunction(
    (expected) =>
      window.applicationFixture.calls.filter((call) => !call.body && call.path === '/api/applications').length ===
      expected,
    readsBefore + 1
  );
  // The first setup completes while that inventory read is still in flight.
  await page.getByRole('button', {name: 'Set up company application', exact: true}).click();
  await page.waitForFunction(() =>
    window.applicationFixture.calls.some((call) => !!call.body && call.path.endsWith('/setup'))
  );
  // The held read cannot land until the completion event is queued.
  await page.waitForTimeout(100);
  assert.equal(await page.getByText(/Setup complete/, {exact: false}).count(), 0);
  assert.equal(await inventoryReads(), readsBefore + 1);
  await page.evaluate(() => {
    const gate = (window.applicationFixture as unknown as {completionGate: {hold: boolean; waiters: (() => void)[]}})
      .completionGate;
    gate.waiters.splice(0).forEach((resolve) => resolve());
  });
  // The in-flight read lands empty, then the queued refresh is dispatched for
  // the first completion, still held.
  await page.waitForFunction(
    (expected) =>
      window.applicationFixture.calls.filter((call) => !call.body && call.path === '/api/applications').length ===
      expected,
    readsBefore + 2
  );
  // The second setup completes while that first refresh is in flight, so the
  // coalesced re-read is owed once it lands.
  await page.getByRole('button', {name: 'Continue owner setup', exact: true}).click();
  await page.waitForFunction(
    () => window.applicationFixture.calls.filter((call) => !!call.body && call.path.endsWith('/setup')).length === 2
  );
  await page.waitForTimeout(100);
  await page.evaluate(() => {
    const gate = (window.applicationFixture as unknown as {completionGate: {hold: boolean; waiters: (() => void)[]}})
      .completionGate;
    gate.waiters.splice(0).forEach((resolve) => resolve());
  });
  // The first refresh lands one completion, then the coalesced re-read is
  // dispatched for both, still held.
  await page.waitForFunction(
    (expected) =>
      window.applicationFixture.calls.filter((call) => !call.body && call.path === '/api/applications').length ===
      expected,
    readsBefore + 3
  );
  // The third setup completes while the coalesced re-read is in flight: the
  // loop must take another pass instead of dropping its re-entry.
  await page.getByRole('button', {name: 'Continue owner setup', exact: true}).click();
  await page.waitForFunction(
    () => window.applicationFixture.calls.filter((call) => !!call.body && call.path.endsWith('/setup')).length === 3
  );
  await page.waitForTimeout(100);
  await page.evaluate(() => {
    const gate = (window.applicationFixture as unknown as {completionGate: {hold: boolean; waiters: (() => void)[]}})
      .completionGate;
    gate.hold = false;
    gate.waiters.splice(0).forEach((resolve) => resolve());
  });
  // The latest identity resolves: the third application is running and ready.
  await page
    .getByText('Setup complete. The application is running and ready. Sign in at its address to use it.', {
      exact: true,
    })
    .waitFor();
  const status = await page.evaluate(
    () =>
      (window.applicationFixture.element.renderRoot ?? window.applicationFixture.element)?.querySelector(
        '.soda-applications-status'
      )?.textContent ?? ''
  );
  assert.match(status, /running and ready/);
  assert.equal(/stopp/i.test(status), false);
  // Exactly one setup POST per completion with no duplicate provisioning and no auto Start.
  assert.equal(await setupPosts(), 3);
  assert.equal(await inventoryReads(), readsBefore + 4);
  assert.equal(
    await page.evaluate(() => window.applicationFixture.calls.filter((call) => call.body?.action === 'start').length),
    0
  );
  await page.waitForTimeout(300);
  assert.equal(await inventoryReads(), readsBefore + 4);
  assert.equal(await setupPosts(), 3);
  assert.equal(
    await page.evaluate(() => window.applicationFixture.calls.filter((call) => call.body?.action === 'start').length),
    0
  );
});

test('failed status sweep clears the stale Ready observation instead of keeping Ready controls', async (t) => {
  const page = await browser.newPage();
  t.after(() => page.close());
  await page.route('https://app.example.test/**', (route) =>
    route.fulfill({body: '<p>Example application</p>', contentType: 'text/html'})
  );
  await page.goto(server.url.href);
  await page.waitForFunction(() => typeof window.createApplicationFixture === 'function');
  await page.evaluate(async () => {
    await window.createApplicationFixture();
    const app = window.applicationFixture.element,
      original = app.request!;
    app.request = async (path, body, signal) => {
      if (body?.action === 'inspect' && (app as unknown as {inspectFailed?: boolean}).inspectFailed) {
        window.applicationFixture.calls.push({path, body});
        throw Error('native inspection failed');
      }
      const result = (await original(path, body, signal)) as {
        applications?: {observation: {running: boolean; ready: boolean} | null; can_embed?: boolean}[];
        observation?: {running: boolean; ready: boolean} | null;
        can_embed?: boolean;
      };
      // Serve a Ready record so the passing sweep enables the iframe. The
      // fixture parks non-start actions as stopped, so force both the
      // inventory rows and the single-record inspect echo back to Ready.
      for (const row of result.applications ?? []) {
        if (!row.observation) continue;
        row.observation.running = true;
        row.observation.ready = true;
        row.can_embed = true;
      }
      if (result.observation) {
        result.observation.running = true;
        result.observation.ready = true;
      }
      if (!result.applications) result.can_embed = true;
      return result;
    };
  });
  await page.getByText('Applications', {exact: true}).click();
  // Passing path: the sweep confirms Ready and enables the iframe.
  await page.getByText('Last observed: Ready', {exact: true}).waitFor();
  await page.locator('soda-application-surface iframe').waitFor();
  assert.equal(await page.locator('soda-application-surface iframe').getAttribute('src'), 'https://app.example.test');
  // The native inspection now fails: the sweep must clear the stale Ready
  // observation back to unknown instead of keeping Ready controls.
  await page.evaluate(() => {
    (window.applicationFixture.element as unknown as {inspectFailed: boolean}).inspectFailed = true;
  });
  await page.getByRole('button', {name: 'Refresh applications', exact: true}).click();
  await page
    .getByText('Some application status checks failed. Open the app or refresh to try again.', {exact: true})
    .waitFor();
  await page.getByText('Last observed: State unknown', {exact: true}).waitFor();
  assert.equal(await page.locator('soda-application-surface iframe').count(), 0);
  // Exact record identity survives: the same application row, no auto-Start.
  assert.equal(await page.locator('.soda-application-heading').getByText('Example', {exact: true}).count(), 1);
  assert.equal(
    await page.evaluate(() => window.applicationFixture.calls.filter((call) => call.body?.action === 'start').length),
    0
  );
  await page.getByText('Start or stop application', {exact: true}).click();
  assert(await page.getByRole('button', {name: 'Start Example', exact: true}).isDisabled());
  assert(await page.getByRole('checkbox', {name: 'Stop Example for everyone', exact: true}).isDisabled());
  await page.waitForTimeout(300);
  assert.equal(await page.getByText('Last observed: State unknown', {exact: true}).count(), 1);
  assert.equal(
    await page.evaluate(() => window.applicationFixture.calls.filter((call) => call.body?.action === 'start').length),
    0
  );
});

test('ordinary employee with unconfirmed availability gets an operator handoff instead of a refresh promise', async (t) => {
  const page = await browser.newPage();
  t.after(() => page.close());
  await page.goto(server.url.href);
  await page.waitForFunction(() => typeof window.createApplicationFixture === 'function');
  await page.evaluate(async () => {
    await window.createApplicationFixture();
    const fixture = window.applicationFixture,
      app = fixture.element;
    app.request = async (path, body) => {
      fixture.calls.push({path, ...(body ? {body} : {})});
      if (body) throw Error('ordinary account cannot administer apps');
      return {
        applications: [],
        launchable: [
          {
            application: {id: 'a' + '1'.repeat(24), revision: 2, name: 'Example', origin: 'https://app.example.test'},
            environment: 'company',
            running: false,
            ready: false,
            state: 'unknown',
            project: {repository: '7', name: 'northstar/equipment', path: '/northstar/equipment'},
            launch_only: true,
            can_embed: true,
          },
        ],
        complete: true,
        can_setup: false,
      };
    };
    app.repository = '7';
    app.expanded = true;
  });
  // The launch-only row holds no native inspection rights, so its refresh
  // only rereads the stored projection: the copy hands off to an operator
  // instead of promising that refresh checks availability.
  await page
    .getByText(
      'Availability is unconfirmed. An operator can inspect the application; Refresh applications only rereads the last stored status.',
      {exact: true}
    )
    .waitFor();
  assert.equal(await page.locator('soda-application-surface iframe').count(), 0);
  await page.getByRole('button', {name: 'Refresh applications', exact: true}).click();
  await page.waitForTimeout(300);
  await page
    .getByText(
      'Availability is unconfirmed. An operator can inspect the application; Refresh applications only rereads the last stored status.',
      {exact: true}
    )
    .waitFor();
  assert.equal(await page.evaluate(() => window.applicationFixture.calls.some((call) => !!call.body)), false);
  assert.equal(
    await page
      .getByText('Availability is unconfirmed. Refresh application status before continuing.', {exact: true})
      .count(),
    0
  );
});

test('inspectable unknown state keeps the refresh guidance for accounts that can re-inspect', async (t) => {
  const page = await browser.newPage();
  t.after(() => page.close());
  await page.goto(server.url.href);
  await page.waitForFunction(() => typeof window.createApplicationFixture === 'function');
  await page.evaluate(async () => {
    await window.createApplicationFixture(true);
    const app = window.applicationFixture.element,
      original = app.request!;
    app.request = async (path, body, signal) => {
      const value = (await original(path, body, signal)) as {
        applications?: {observation: unknown}[];
        observation?: unknown;
      };
      // The stored observation stays unknown on both the inventory and the
      // status reads, so the full-record surface renders its unknown branch.
      if (Array.isArray(value.applications)) for (const row of value.applications) row.observation = null;
      if (value && typeof value === 'object' && 'observation' in value && !Array.isArray(value.applications))
        value.observation = null;
      return value;
    };
    app.expanded = true;
  });
  // A full record carries an Inspect action and its refresh re-inspects, so
  // the refresh guidance stays truthful here while the handoff stays hidden.
  await page.getByText('Last observed: State unknown', {exact: true}).waitFor();
  await page
    .getByText('Availability is unconfirmed. Refresh application status before continuing.', {exact: true})
    .waitFor();
  assert.equal(
    await page
      .getByText(
        'Availability is unconfirmed. An operator can inspect the application; Refresh applications only rereads the last stored status.',
        {exact: true}
      )
      .count(),
    0
  );
});

test('continuing a setup that settled elsewhere names the retained outcome instead of stalling silently', async (t) => {
  const page = await browser.newPage();
  t.after(() => page.close());
  await page.goto(server.url.href);
  await page.waitForFunction(() => typeof window.createApplicationDeliveryFixture === 'function');
  await page.evaluate(() => window.createApplicationDeliveryFixture());
  // The wrapper below is installed before preparing so it can record the
  // retained build/setup association the served inventory carries; the stale
  // flag flips only after the first continue opens the owner link.
  await page.evaluate(() => {
    const app = window.applicationFixture.element,
      original = app.request!,
      control = window.applicationFixture as unknown as {staleSetups: boolean},
      retained: Record<string, string> = {};
    control.staleSetups = false;
    app.request = (async (path, body, signal) => {
      if (path === '/api/applications/setups' && !body && control.staleSetups) return {setups: []};
      const value = (await original(path, body, signal)) as {
        id?: unknown;
        builds?: {id: string; setup_id?: string}[];
      };
      // The prepared build permanently references its setup id, as served: a
      // delivery re-read re-attaches the retained association instead of
      // offering a fresh setup for the same build.
      const setup = path.match(/\/application-builds\/([a-f0-9]{32})\/setup$/);
      if (setup && body && typeof value.id === 'string') retained[setup[1]!] = value.id;
      if (!body && Array.isArray(value.builds))
        for (const build of value.builds) {
          const id = retained[build.id];
          if (id) build.setup_id = id;
        }
      return value;
    }) as typeof app.request;
  });
  await page.getByRole('button', {name: 'Prepare company version', exact: true}).click();
  await page.getByRole('button', {name: 'Set up company application', exact: true}).click();
  await page.getByRole('link', {name: 'Create application owner ↗', exact: true}).waitFor();
  // The same setup settles outside this page: the setups inventory no longer
  // lists it as pending while the prepared build still references its id.
  await page.evaluate(() => {
    (window.applicationFixture as unknown as {staleSetups: boolean}).staleSetups = true;
  });
  const setupActionsBefore = await page.evaluate(
    () =>
      window.applicationFixture.calls.filter((call) => !!call.body && call.path.startsWith('/api/applications/setups/'))
        .length
  );
  assert.equal(setupActionsBefore, 1);
  await page.getByRole('button', {name: 'Continue owner setup', exact: true}).click();
  await page
    .getByText(
      'This owner setup is no longer pending, so it cannot be continued. Its reserved address and data are retained; refresh applications to reconcile the current inventory before acting again.',
      {exact: true}
    )
    .waitFor();
  // No setup action was issued for the stale id: it was never begun,
  // inspected, completed or aborted, and no fresh link is shown.
  assert.equal(
    await page.evaluate(
      () =>
        window.applicationFixture.calls.filter(
          (call) => !!call.body && call.path.startsWith('/api/applications/setups/')
        ).length
    ),
    setupActionsBefore
  );
  assert.equal(await page.getByRole('link', {name: 'Create application owner ↗', exact: true}).count(), 0);
});

test('continuing an expired setup re-inspects the retained generation instead of opening a dead link', async (t) => {
  const page = await browser.newPage();
  t.after(() => page.close());
  await page.goto(server.url.href);
  await page.waitForFunction(() => typeof window.createApplicationDeliveryFixture === 'function');
  await page.evaluate(() => window.createApplicationDeliveryFixture());
  // The wrapper is installed before preparing so it can record the retained
  // build/setup association the served inventory carries; the expiry flips
  // only after the first continue opens the owner link.
  await page.evaluate(() => {
    const app = window.applicationFixture.element,
      original = app.request!,
      control = window.applicationFixture as unknown as {expiredSetups: boolean},
      retained: Record<string, string> = {};
    control.expiredSetups = false;
    app.request = (async (path, body, signal) => {
      if (body?.action === 'begin' && path.startsWith('/api/applications/setups/') && control.expiredSetups)
        throw Error('setup token expired');
      const value = (await original(path, body, signal)) as {
        id?: unknown;
        builds?: {id: string; setup_id?: string}[];
        setups?: {expires_at: number}[];
        setup?: unknown;
        result?: Record<string, unknown>;
      };
      // The prepared build permanently references its setup id, as served: a
      // delivery re-read re-attaches the retained association instead of
      // offering a fresh setup for the same build.
      const setup = path.match(/\/application-builds\/([a-f0-9]{32})\/setup$/);
      if (setup && body && typeof value.id === 'string') retained[setup[1]!] = value.id;
      if (!body && Array.isArray(value.builds))
        for (const build of value.builds) {
          const id = retained[build.id];
          if (id) build.setup_id = id;
        }
      // The 30-minute link window lapses: the host can no longer mint a fresh
      // owner link for this setup, but the retained generation stays
      // inspectable. Only begin mints links, so the inspect echo carries
      // none, as served.
      if (control.expiredSetups) {
        const past = Math.floor(Date.now() / 1000) - 1;
        if (path === '/api/applications/setups' && !body && Array.isArray(value.setups))
          return {setups: value.setups.map((row) => ({...row, expires_at: past}))};
        // The stored record keeps its lapsed expiry on every read, as served.
        if (body && path.startsWith('/api/applications/setups/') && value.setup && typeof value.setup === 'object')
          value.setup = {...(value.setup as Record<string, unknown>), expires_at: past};
        if (body?.action === 'inspect' && value.result) delete value.result.setup_url;
      }
      return value;
    }) as typeof app.request;
  });
  await page.getByRole('button', {name: 'Prepare company version', exact: true}).click();
  await page.getByRole('button', {name: 'Set up company application', exact: true}).click();
  await page.getByRole('link', {name: 'Create application owner ↗', exact: true}).waitFor();
  await page.evaluate(() => {
    (window.applicationFixture as unknown as {expiredSetups: boolean}).expiredSetups = true;
  });
  const beginsBefore = await page.evaluate(
    () => window.applicationFixture.calls.filter((call) => call.body?.action === 'begin').length
  );
  const inspectsBefore = await page.evaluate(
    () => window.applicationFixture.calls.filter((call) => call.body?.action === 'inspect').length
  );
  await page.getByRole('button', {name: 'Continue owner setup', exact: true}).click();
  await page
    .getByText(
      'This setup link has expired, so its owner link cannot be re-opened as-is. The same setup was inspected and its owner setup is running. If its owner was already created, confirm the owner and finish; otherwise stop and cancel this setup. Its owner credential, data, and reserved address stay retained.',
      {exact: true}
    )
    .waitFor();
  await page.getByText(/The link has expired/, {exact: false}).waitFor();
  // Exactly one supported read reconciled the retained setup; begin was never
  // attempted and no link — stale or fresh — is offered. No renew action is
  // offered (865 contract): no arm button, no confirm block, no fresh link.
  assert.equal(
    await page.evaluate(() => window.applicationFixture.calls.filter((call) => call.body?.action === 'begin').length),
    beginsBefore
  );
  assert.equal(
    await page.evaluate(() => window.applicationFixture.calls.filter((call) => call.body?.action === 'inspect').length),
    inspectsBefore + 1
  );
  assert.equal(await page.getByRole('link', {name: 'Create application owner ↗', exact: true}).count(), 0);
  assert.equal(await page.getByRole('button', {name: /Renew setup link/, exact: false}).count(), 0);
  assert.equal(await page.getByRole('button', {name: /Confirm renewal/, exact: false}).count(), 0);
  assert.equal(
    await page.evaluate(() => window.applicationFixture.calls.filter((call) => call.body?.action === 'renew').length),
    0
  );
  assert.equal(await page.getByText(/fresh 30-minute window/, {exact: false}).count(), 0);
  assert.equal(await page.getByText(/at most three times/, {exact: false}).count(), 0);
});

test('a failed expired-setup inspect reports unconfirmed instead of claiming inspected', async (t) => {
  const page = await browser.newPage();
  t.after(() => page.close());
  await page.goto(server.url.href);
  await page.waitForFunction(() => typeof window.createApplicationDeliveryFixture === 'function');
  await page.evaluate(() => window.createApplicationDeliveryFixture());
  await page.evaluate(() => {
    const app = window.applicationFixture.element,
      original = app.request!,
      control = window.applicationFixture as unknown as {expiredSetups: boolean},
      retained: Record<string, string> = {};
    control.expiredSetups = false;
    app.request = (async (path, body, signal) => {
      // The host reports the retained generation unconfirmed past expiry: a
      // 502 names the failure, never a usable observation.
      if (body?.action === 'inspect' && path.startsWith('/api/applications/setups/') && control.expiredSetups)
        throw Object.assign(new Error('host unconfirmed'), {status: 502, code: 'setup_unconfirmed'});
      const value = (await original(path, body, signal)) as {
        id?: unknown;
        builds?: {id: string; setup_id?: string}[];
        setups?: {expires_at: number}[];
        setup?: unknown;
        result?: Record<string, unknown>;
      };
      const setup = path.match(/\/application-builds\/([a-f0-9]{32})\/setup$/);
      if (setup && body && typeof value.id === 'string') retained[setup[1]!] = value.id;
      if (!body && Array.isArray(value.builds))
        for (const build of value.builds) {
          const id = retained[build.id];
          if (id) build.setup_id = id;
        }
      if (control.expiredSetups) {
        const past = Math.floor(Date.now() / 1000) - 1;
        if (path === '/api/applications/setups' && !body && Array.isArray(value.setups))
          return {setups: value.setups.map((row) => ({...row, expires_at: past}))};
        if (body && path.startsWith('/api/applications/setups/') && value.setup && typeof value.setup === 'object')
          value.setup = {...(value.setup as Record<string, unknown>), expires_at: past};
      }
      return value;
    }) as typeof app.request;
  });
  await page.getByRole('button', {name: 'Prepare company version', exact: true}).click();
  await page.getByRole('button', {name: 'Set up company application', exact: true}).click();
  await page.getByRole('link', {name: 'Create application owner ↗', exact: true}).waitFor();
  await page.evaluate(() => {
    (window.applicationFixture as unknown as {expiredSetups: boolean}).expiredSetups = true;
  });
  const beginsBefore = await page.evaluate(
    () => window.applicationFixture.calls.filter((call) => call.body?.action === 'begin').length
  );
  await page.getByRole('button', {name: 'Continue owner setup', exact: true}).click();
  // The failed inspect is reported with its reason; inspected is never
  // claimed and no fresh link is minted on the failure.
  await page
    .getByText(
      'This setup could not be inspected. Refresh owner setups and inspect the same setup before continuing.',
      {exact: true}
    )
    .waitFor();
  assert.equal(await page.getByText(/was inspected/, {exact: false}).count(), 0);
  assert.equal(
    await page.evaluate(() => window.applicationFixture.calls.filter((call) => call.body?.action === 'begin').length),
    beginsBefore
  );
  assert.equal(await page.getByRole('link', {name: 'Create application owner ↗', exact: true}).count(), 0);
});

test('a refused expired-setup inspect names the operator instead of a session change', async (t) => {
  const page = await browser.newPage();
  t.after(() => page.close());
  await page.goto(server.url.href);
  await page.waitForFunction(() => typeof window.createApplicationDeliveryFixture === 'function');
  await page.evaluate(() => window.createApplicationDeliveryFixture());
  await page.evaluate(() => {
    const app = window.applicationFixture.element,
      original = app.request!,
      control = window.applicationFixture as unknown as {expiredSetups: boolean},
      retained: Record<string, string> = {};
    control.expiredSetups = false;
    app.request = (async (path, body, signal) => {
      // A non-operator refusal carries the operator_required refusal, never
      // an observation and never a session change.
      if (body?.action === 'inspect' && path.startsWith('/api/applications/setups/') && control.expiredSetups)
        throw Object.assign(new Error('operator required'), {status: 403, code: 'operator_required'});
      const value = (await original(path, body, signal)) as {
        id?: unknown;
        builds?: {id: string; setup_id?: string}[];
        setups?: {expires_at: number}[];
        setup?: unknown;
        result?: Record<string, unknown>;
      };
      const setup = path.match(/\/application-builds\/([a-f0-9]{32})\/setup$/);
      if (setup && body && typeof value.id === 'string') retained[setup[1]!] = value.id;
      if (!body && Array.isArray(value.builds))
        for (const build of value.builds) {
          const id = retained[build.id];
          if (id) build.setup_id = id;
        }
      if (control.expiredSetups) {
        const past = Math.floor(Date.now() / 1000) - 1;
        if (path === '/api/applications/setups' && !body && Array.isArray(value.setups))
          return {setups: value.setups.map((row) => ({...row, expires_at: past}))};
        if (body && path.startsWith('/api/applications/setups/') && value.setup && typeof value.setup === 'object')
          value.setup = {...(value.setup as Record<string, unknown>), expires_at: past};
      }
      return value;
    }) as typeof app.request;
  });
  await page.getByRole('button', {name: 'Prepare company version', exact: true}).click();
  await page.getByRole('button', {name: 'Set up company application', exact: true}).click();
  await page.getByRole('link', {name: 'Create application owner ↗', exact: true}).waitFor();
  await page.evaluate(() => {
    (window.applicationFixture as unknown as {expiredSetups: boolean}).expiredSetups = true;
  });
  await page.getByRole('button', {name: 'Continue owner setup', exact: true}).click();
  await page.getByText('Only the configured Soda operator can inspect this setup.', {exact: true}).waitFor();
  assert.equal(await page.getByText(/session changed/, {exact: false}).count(), 0);
  assert.equal(await page.getByText(/was inspected/, {exact: false}).count(), 0);
  assert.equal(await page.getByRole('link', {name: 'Create application owner ↗', exact: true}).count(), 0);
});

test('a mismatched expired-setup inspect reply proves nothing instead of claiming inspected', async (t) => {
  const page = await browser.newPage();
  t.after(() => page.close());
  await page.goto(server.url.href);
  await page.waitForFunction(() => typeof window.createApplicationDeliveryFixture === 'function');
  await page.evaluate(() => window.createApplicationDeliveryFixture());
  await page.evaluate(() => {
    const app = window.applicationFixture.element,
      original = app.request!,
      control = window.applicationFixture as unknown as {expiredSetups: boolean},
      retained: Record<string, string> = {};
    control.expiredSetups = false;
    app.request = (async (path, body, signal) => {
      // A mismatched inspect reply carries no usable observation: the shape
      // check rejects it before any record changes.
      if (body?.action === 'inspect' && path.startsWith('/api/applications/setups/') && control.expiredSetups)
        return {setup: {id: 'a' + 'c'.repeat(24)}, result: {operation: 'a' + 'c'.repeat(24)}};
      const value = (await original(path, body, signal)) as {
        id?: unknown;
        builds?: {id: string; setup_id?: string}[];
        setups?: {expires_at: number}[];
        setup?: unknown;
        result?: Record<string, unknown>;
      };
      const setup = path.match(/\/application-builds\/([a-f0-9]{32})\/setup$/);
      if (setup && body && typeof value.id === 'string') retained[setup[1]!] = value.id;
      if (!body && Array.isArray(value.builds))
        for (const build of value.builds) {
          const id = retained[build.id];
          if (id) build.setup_id = id;
        }
      if (control.expiredSetups) {
        const past = Math.floor(Date.now() / 1000) - 1;
        if (path === '/api/applications/setups' && !body && Array.isArray(value.setups))
          return {setups: value.setups.map((row) => ({...row, expires_at: past}))};
        if (body && path.startsWith('/api/applications/setups/') && value.setup && typeof value.setup === 'object')
          value.setup = {...(value.setup as Record<string, unknown>), expires_at: past};
      }
      return value;
    }) as typeof app.request;
  });
  await page.getByRole('button', {name: 'Prepare company version', exact: true}).click();
  await page.getByRole('button', {name: 'Set up company application', exact: true}).click();
  await page.getByRole('link', {name: 'Create application owner ↗', exact: true}).waitFor();
  await page.evaluate(() => {
    (window.applicationFixture as unknown as {expiredSetups: boolean}).expiredSetups = true;
  });
  const beginsBefore = await page.evaluate(
    () => window.applicationFixture.calls.filter((call) => call.body?.action === 'begin').length
  );
  await page.getByRole('button', {name: 'Continue owner setup', exact: true}).click();
  await page
    .getByText(
      'This setup could not be inspected. Refresh owner setups and inspect the same setup before continuing.',
      {exact: true}
    )
    .waitFor();
  assert.equal(await page.getByText(/was inspected/, {exact: false}).count(), 0);
  assert.equal(
    await page.evaluate(() => window.applicationFixture.calls.filter((call) => call.body?.action === 'begin').length),
    beginsBefore
  );
  assert.equal(await page.getByRole('link', {name: 'Create application owner ↗', exact: true}).count(), 0);
});

test('a well-formed expired-setup inspect reply naming another setup proves nothing', async (t) => {
  const page = await browser.newPage();
  t.after(() => page.close());
  await page.goto(server.url.href);
  await page.waitForFunction(() => typeof window.createApplicationDeliveryFixture === 'function');
  await page.evaluate(() => window.createApplicationDeliveryFixture());
  await page.evaluate(() => {
    const app = window.applicationFixture.element,
      original = app.request!,
      control = window.applicationFixture as unknown as {expiredSetups: boolean},
      retained: Record<string, string> = {};
    control.expiredSetups = false;
    app.request = (async (path, body, signal) => {
      const value = (await original(path, body, signal)) as {
        id?: unknown;
        builds?: {id: string; setup_id?: string}[];
        setups?: {expires_at: number}[];
        setup?: unknown;
        result?: Record<string, unknown>;
      };
      const setup = path.match(/\/application-builds\/([a-f0-9]{32})\/setup$/);
      if (setup && body && typeof value.id === 'string') retained[setup[1]!] = value.id;
      if (!body && Array.isArray(value.builds))
        for (const build of value.builds) {
          const id = retained[build.id];
          if (id) build.setup_id = id;
        }
      if (control.expiredSetups) {
        const past = Math.floor(Date.now() / 1000) - 1;
        if (path === '/api/applications/setups' && !body && Array.isArray(value.setups))
          return {setups: value.setups.map((row) => ({...row, expires_at: past}))};
        if (body && path.startsWith('/api/applications/setups/') && value.setup && typeof value.setup === 'object')
          value.setup = {...(value.setup as Record<string, unknown>), expires_at: past};
        if (body?.action === 'inspect' && value.setup && typeof value.setup === 'object') {
          // A well-formed reply for a DIFFERENT setup id: every shape check
          // passes, so only the identity check can reject it before any
          // record changes. It proves nothing about the retained setup.
          const foreign = 'a' + 'd'.repeat(24);
          value.setup = {...(value.setup as Record<string, unknown>), id: foreign};
          value.result = {operation: foreign};
        }
      }
      return value;
    }) as typeof app.request;
  });
  await page.getByRole('button', {name: 'Prepare company version', exact: true}).click();
  await page.getByRole('button', {name: 'Set up company application', exact: true}).click();
  await page.getByRole('link', {name: 'Create application owner ↗', exact: true}).waitFor();
  await page.evaluate(() => {
    (window.applicationFixture as unknown as {expiredSetups: boolean}).expiredSetups = true;
  });
  const beginsBefore = await page.evaluate(
    () => window.applicationFixture.calls.filter((call) => call.body?.action === 'begin').length
  );
  await page.getByRole('button', {name: 'Continue owner setup', exact: true}).click();
  await page
    .getByText(
      'This setup could not be inspected. Refresh owner setups and inspect the same setup before continuing.',
      {exact: true}
    )
    .waitFor();
  assert.equal(await page.getByText(/was inspected/, {exact: false}).count(), 0);
  assert.equal(
    await page.evaluate(() => window.applicationFixture.calls.filter((call) => call.body?.action === 'begin').length),
    beginsBefore
  );
  assert.equal(await page.getByRole('link', {name: 'Create application owner ↗', exact: true}).count(), 0);
});

test('a failed continue refresh never retires the live setup', async (t) => {
  const page = await browser.newPage();
  t.after(() => page.close());
  await page.goto(server.url.href);
  await page.waitForFunction(() => typeof window.createApplicationDeliveryFixture === 'function');
  await page.evaluate(() => window.createApplicationDeliveryFixture());
  await page.evaluate(() => {
    const app = window.applicationFixture.element,
      original = app.request!,
      control = window.applicationFixture as unknown as {deadInventory: boolean},
      retained: Record<string, string> = {};
    control.deadInventory = false;
    app.request = (async (path, body, signal) => {
      // The inventory read fails outright: nothing about the retained setup
      // is proven, so no pending-state transition may follow.
      if (path === '/api/applications/setups' && !body && control.deadInventory) throw Error('inventory lost');
      const value = (await original(path, body, signal)) as {
        id?: unknown;
        builds?: {id: string; setup_id?: string}[];
      };
      const setup = path.match(/\/application-builds\/([a-f0-9]{32})\/setup$/);
      if (setup && body && typeof value.id === 'string') retained[setup[1]!] = value.id;
      if (!body && Array.isArray(value.builds))
        for (const build of value.builds) {
          const id = retained[build.id];
          if (id) build.setup_id = id;
        }
      return value;
    }) as typeof app.request;
  });
  await page.getByRole('button', {name: 'Prepare company version', exact: true}).click();
  await page.getByRole('button', {name: 'Set up company application', exact: true}).click();
  await page.getByRole('link', {name: 'Create application owner ↗', exact: true}).waitFor();
  await page.evaluate(() => {
    (window.applicationFixture as unknown as {deadInventory: boolean}).deadInventory = true;
  });
  const actionsBefore = await page.evaluate(
    () =>
      window.applicationFixture.calls.filter((call) => !!call.body && call.path.startsWith('/api/applications/setups/'))
        .length
  );
  await page.getByRole('button', {name: 'Continue owner setup', exact: true}).click();
  // The failed read keeps its confirmation notice; the live setup is not
  // retired on the stale read.
  await page
    .getByText('Setup could not be confirmed. Refresh and inspect the same setup before continuing.', {exact: true})
    .waitFor();
  assert.equal(await page.getByText(/no longer pending/, {exact: false}).count(), 0);
  assert.equal(
    await page.evaluate(
      () =>
        window.applicationFixture.calls.filter(
          (call) => !!call.body && call.path.startsWith('/api/applications/setups/')
        ).length
    ),
    actionsBefore
  );
});

test('a superseded continue refresh stays silent instead of retiring the setup', async (t) => {
  const page = await browser.newPage();
  t.after(() => page.close());
  await page.goto(server.url.href);
  await page.waitForFunction(() => typeof window.createApplicationDeliveryFixture === 'function');
  await page.evaluate(() => window.createApplicationDeliveryFixture());
  await page.evaluate(() => {
    const app = window.applicationFixture.element,
      original = app.request!,
      control = window.applicationFixture as unknown as {
        slowInventory: boolean;
        releaseInventory?: () => void;
      },
      retained: Record<string, string> = {};
    control.slowInventory = false;
    app.request = (async (path, body, signal) => {
      if (path === '/api/applications/setups' && !body && control.slowInventory) {
        // Record the gated read before parking it so the test can observe the
        // in-flight refresh; the wrapped transport records it again on release.
        window.applicationFixture.calls.push({path});
        await new Promise<void>((resolve) => {
          control.releaseInventory = resolve;
        });
      }
      const value = (await original(path, body, signal)) as {
        id?: unknown;
        builds?: {id: string; setup_id?: string}[];
      };
      const setup = path.match(/\/application-builds\/([a-f0-9]{32})\/setup$/);
      if (setup && body && typeof value.id === 'string') retained[setup[1]!] = value.id;
      if (!body && Array.isArray(value.builds))
        for (const build of value.builds) {
          const id = retained[build.id];
          if (id) build.setup_id = id;
        }
      return value;
    }) as typeof app.request;
  });
  await page.getByRole('button', {name: 'Prepare company version', exact: true}).click();
  await page.getByRole('button', {name: 'Set up company application', exact: true}).click();
  await page.getByRole('link', {name: 'Create application owner ↗', exact: true}).waitFor();
  const readsBefore = await page.evaluate(
    () =>
      window.applicationFixture.calls.filter((call) => call.path === '/api/applications/setups' && !call.body).length
  );
  await page.evaluate(() => {
    (window.applicationFixture as unknown as {slowInventory: boolean}).slowInventory = true;
  });
  await page.getByRole('button', {name: 'Continue owner setup', exact: true}).click();
  // The continue refresh is in flight; a context switch supersedes it before
  // the read lands.
  await page.waitForFunction(
    (before) =>
      window.applicationFixture.calls.filter((call) => call.path === '/api/applications/setups' && !call.body).length >
      before,
    readsBefore
  );
  await page.evaluate(() => {
    const control = window.applicationFixture as unknown as {
      slowInventory: boolean;
      releaseInventory?: () => void;
    };
    control.slowInventory = false;
    (document.querySelector('soda-application-setup') as unknown as {sessionKey: string}).sessionKey = 'generation-2';
    control.releaseInventory?.();
  });
  await page.waitForTimeout(300);
  // The superseded read proves nothing: no pending-state transition, no
  // setup action, and no notice from the run it no longer owns.
  assert.equal(await page.getByText(/no longer pending/, {exact: false}).count(), 0);
  assert.equal(await page.getByText(/could not be confirmed/, {exact: false}).count(), 0);
  assert.equal(await page.getByText(/was inspected/, {exact: false}).count(), 0);
  assert.equal(
    await page.evaluate(
      () =>
        window.applicationFixture.calls.filter(
          (call) => !!call.body && call.path.startsWith('/api/applications/setups/')
        ).length
    ),
    1
  );
});

test('an inspected-but-unconfirmed expired generation reports unconfirmed instead of claiming absence', async (t) => {
  const page = await browser.newPage();
  t.after(() => page.close());
  await page.goto(server.url.href);
  await page.waitForFunction(() => typeof window.createApplicationDeliveryFixture === 'function');
  await page.evaluate(() => window.createApplicationDeliveryFixture());
  await page.evaluate(() => {
    const app = window.applicationFixture.element,
      original = app.request!,
      control = window.applicationFixture as unknown as {expiredSetups: boolean},
      retained: Record<string, string> = {};
    control.expiredSetups = false;
    app.request = (async (path, body, signal) => {
      const value = (await original(path, body, signal)) as {
        id?: unknown;
        builds?: {id: string; setup_id?: string}[];
        setups?: {expires_at: number}[];
        setup?: unknown;
        result?: Record<string, unknown>;
      };
      const setup = path.match(/\/application-builds\/([a-f0-9]{32})\/setup$/);
      if (setup && body && typeof value.id === 'string') retained[setup[1]!] = value.id;
      if (!body && Array.isArray(value.builds))
        for (const build of value.builds) {
          const id = retained[build.id];
          if (id) build.setup_id = id;
        }
      if (control.expiredSetups) {
        const past = Math.floor(Date.now() / 1000) - 1;
        if (path === '/api/applications/setups' && !body && Array.isArray(value.setups))
          return {setups: value.setups.map((row) => ({...row, expires_at: past}))};
        if (body && path.startsWith('/api/applications/setups/') && value.setup && typeof value.setup === 'object')
          value.setup = {...(value.setup as Record<string, unknown>), expires_at: past};
        if (body?.action === 'inspect' && value.setup && typeof value.setup === 'object') {
          // The inspect confirmed no state: null proves nothing, so absence
          // cannot be claimed and finish is unsupported here.
          value.setup = {...(value.setup as Record<string, unknown>), observation: null};
          if (value.result) delete value.result.setup_url;
        }
      }
      return value;
    }) as typeof app.request;
  });
  await page.getByRole('button', {name: 'Prepare company version', exact: true}).click();
  await page.getByRole('button', {name: 'Set up company application', exact: true}).click();
  await page.getByRole('link', {name: 'Create application owner ↗', exact: true}).waitFor();
  await page.evaluate(() => {
    (window.applicationFixture as unknown as {expiredSetups: boolean}).expiredSetups = true;
  });
  const beginsBefore = await page.evaluate(
    () => window.applicationFixture.calls.filter((call) => call.body?.action === 'begin').length
  );
  await page.getByRole('button', {name: 'Continue owner setup', exact: true}).click();
  await page
    .getByText(
      'This setup link has expired, so its owner link cannot be re-opened as-is. The same setup was inspected but its owner state is unconfirmed, so nothing can be confirmed about it yet. Inspect the setup again or ask the Soda operator to check it before stopping and cancelling it. Its owner credential, data, and reserved address stay retained. Inspect or refresh this same setup to confirm its state.',
      {exact: true}
    )
    .waitFor();
  assert.equal(
    await page
      .getByText(
        'This setup link has expired, so its owner link cannot be re-opened as-is. The same setup was inspected and no owner setup is running. Stop and cancel this setup. Its owner credential, data, and reserved address stay retained. Refresh owner setups to confirm the inventory after cancelling.',
        {exact: true}
      )
      .count(),
    0
  );
  assert.equal(
    await page.evaluate(() => window.applicationFixture.calls.filter((call) => call.body?.action === 'begin').length),
    beginsBefore
  );
  assert.equal(await page.getByRole('link', {name: 'Create application owner ↗', exact: true}).count(), 0);
  assert.equal(await page.getByRole('button', {name: /Renew setup link/, exact: false}).count(), 0);
  assert.equal(await page.getByRole('button', {name: /Confirm renewal/, exact: false}).count(), 0);
  assert.equal(
    await page.evaluate(() => window.applicationFixture.calls.filter((call) => call.body?.action === 'renew').length),
    0
  );
  assert.equal(await page.getByText(/fresh 30-minute window/, {exact: false}).count(), 0);
});

test('an inspected-but-absent expired generation names cancel as the only path', async (t) => {
  const page = await browser.newPage();
  t.after(() => page.close());
  await page.goto(server.url.href);
  await page.waitForFunction(() => typeof window.createApplicationDeliveryFixture === 'function');
  await page.evaluate(() => window.createApplicationDeliveryFixture());
  await page.evaluate(() => {
    const app = window.applicationFixture.element,
      original = app.request!,
      control = window.applicationFixture as unknown as {expiredSetups: boolean},
      retained: Record<string, string> = {};
    control.expiredSetups = false;
    app.request = (async (path, body, signal) => {
      const value = (await original(path, body, signal)) as {
        id?: unknown;
        builds?: {id: string; setup_id?: string}[];
        setups?: {expires_at: number}[];
        setup?: unknown;
        result?: Record<string, unknown>;
      };
      const setup = path.match(/\/application-builds\/([a-f0-9]{32})\/setup$/);
      if (setup && body && typeof value.id === 'string') retained[setup[1]!] = value.id;
      if (!body && Array.isArray(value.builds))
        for (const build of value.builds) {
          const id = retained[build.id];
          if (id) build.setup_id = id;
        }
      if (control.expiredSetups) {
        const past = Math.floor(Date.now() / 1000) - 1;
        if (path === '/api/applications/setups' && !body && Array.isArray(value.setups))
          return {setups: value.setups.map((row) => ({...row, expires_at: past}))};
        if (body && path.startsWith('/api/applications/setups/') && value.setup && typeof value.setup === 'object')
          value.setup = {...(value.setup as Record<string, unknown>), expires_at: past};
        if (body?.action === 'inspect' && value.setup && typeof value.setup === 'object') {
          // Confirmed native absence: an observation object with an empty
          // container and running=false/ready=false. Only here does cancel
          // name the only path.
          value.setup = {
            ...(value.setup as Record<string, unknown>),
            observation: {container: '', running: false, ready: false},
          };
          if (value.result) delete value.result.setup_url;
        }
      }
      return value;
    }) as typeof app.request;
  });
  await page.getByRole('button', {name: 'Prepare company version', exact: true}).click();
  await page.getByRole('button', {name: 'Set up company application', exact: true}).click();
  await page.getByRole('link', {name: 'Create application owner ↗', exact: true}).waitFor();
  await page.evaluate(() => {
    (window.applicationFixture as unknown as {expiredSetups: boolean}).expiredSetups = true;
  });
  const beginsBefore = await page.evaluate(
    () => window.applicationFixture.calls.filter((call) => call.body?.action === 'begin').length
  );
  await page.getByRole('button', {name: 'Continue owner setup', exact: true}).click();
  await page
    .getByText(
      'This setup link has expired, so its owner link cannot be re-opened as-is. The same setup was inspected and no owner setup is running. Stop and cancel this setup. Its owner credential, data, and reserved address stay retained. Refresh owner setups to confirm the inventory after cancelling.',
      {exact: true}
    )
    .waitFor();
  assert.equal(
    await page
      .getByText(
        'This setup link has expired, so its owner link cannot be re-opened as-is. The same setup was inspected but its owner state is unconfirmed, so nothing can be confirmed about it yet. Inspect the setup again or ask the Soda operator to check it before stopping and cancelling it. Its owner credential, data, and reserved address stay retained. Inspect or refresh this same setup to confirm its state.',
        {exact: true}
      )
      .count(),
    0
  );
  assert.equal(
    await page.evaluate(() => window.applicationFixture.calls.filter((call) => call.body?.action === 'begin').length),
    beginsBefore
  );
  assert.equal(await page.getByRole('link', {name: 'Create application owner ↗', exact: true}).count(), 0);
  assert.equal(await page.getByRole('button', {name: /Renew setup link/, exact: false}).count(), 0);
  assert.equal(await page.getByRole('button', {name: /Confirm renewal/, exact: false}).count(), 0);
  assert.equal(
    await page.evaluate(() => window.applicationFixture.calls.filter((call) => call.body?.action === 'renew').length),
    0
  );
  assert.equal(await page.getByText(/fresh 30-minute window/, {exact: false}).count(), 0);
});

test('opening the current company application shows only the company surface', async (t) => {
  const page = await browser.newPage();
  t.after(() => page.close());
  await page.goto(server.url.href);
  await page.waitForFunction(() => typeof window.createApplicationFixture === 'function');
  await page.evaluate(async () => {
    await window.createApplicationFixture();
    const app = window.applicationFixture.element;
    const record = (
      suffix: string,
      environment: string,
      name: string,
      origin: string,
      observation: Record<string, unknown> | null,
      delivery?: Record<string, unknown>
    ) => ({
      environment,
      application: {
        id: 'a' + suffix.repeat(24),
        revision: Number(suffix),
        name,
        artifact_sha256: '2'.repeat(64),
        bundle_sha256: '3'.repeat(64),
        tools_sha256: '4'.repeat(64),
        runtime_image: 'sha256:' + '5'.repeat(64),
        image: 'sha256:' + '6'.repeat(64),
        data: 'a' + '7'.repeat(24),
        owner_team: '12345678-1234-1234-1234-123456789abc',
        origin,
        port: 20000 + Number(suffix),
      },
      administrator_id: '41',
      operator: false,
      observation,
      can_embed: true,
      project: {repository: '7', name: 'northstar/equipment', path: '/northstar/equipment'},
      ...(delivery ? {delivery} : {}),
    });
    const stopped = (suffix: string, origin: string) => ({
      id: 'a' + suffix.repeat(24),
      revision: Number(suffix),
      container: suffix.repeat(64),
      image: 'sha256:' + '6'.repeat(64),
      origin,
      running: false,
      ready: false,
      bundle_file_sha256: 'b'.repeat(64),
    });
    const rows = [
      record(
        '1',
        'company',
        'Example-company',
        'https://company.example.test',
        stopped('1', 'https://company.example.test'),
        {
          environment: 'company',
          state: 'verified',
          intent: 'intent-current',
          revision: 'revision-current',
          delivery_id: 'b'.repeat(32),
          run: 'run-current',
          source_commit: 'c'.repeat(40),
          bundle_file_sha256: 'b'.repeat(64),
        }
      ),
      // A preview runs separate data: it is never the current company application.
      record('2', 'preview', 'Example-preview', 'https://preview.example.test', {
        ...stopped('2', 'https://preview.example.test'),
        running: true,
        ready: true,
      }),
      // An unconfirmed environment belongs to neither data lane.
      record('3', 'unknown', 'Example-unconfirmed', 'https://unconfirmed.example.test', null),
    ];
    app.request = async (path, body) => {
      window.applicationFixture.calls.push({path, ...(body ? {body} : {})});
      if (!body) return {applications: rows, complete: true};
      // Status reads echo the addressed record; anything else would be a product write.
      if (body.action === 'inspect')
        return rows.find((row) => path === `/api/applications/${row.application.id}/actions`);
      throw Error('company scope must not write');
    };
    app.repository = '7';
    app.requestedDelivery = 'd'.repeat(32);
    app.expanded = true;
  });
  await page
    .getByText(
      'An application matching this exact result is not available to you. The current company application may be a different version.',
      {exact: true}
    )
    .waitFor();
  await page.getByRole('button', {name: 'Open current company application', exact: true}).click();
  // Only the confirmed company record opens: the preview and the
  // unconfirmed environment stay hidden, with the scope named.
  await page.locator('.soda-application-heading').getByText('Example-company', {exact: true}).waitFor();
  await page.getByText('Showing the current company application; previews stay separate.', {exact: true}).waitFor();
  assert.equal(await page.locator('.soda-application-heading').getByText('Example-preview', {exact: true}).count(), 0);
  assert.equal(
    await page.locator('.soda-application-heading').getByText('Example-unconfirmed', {exact: true}).count(),
    0
  );
  assert.equal(await page.locator('soda-application-surface').count(), 1);
  assert.equal(await page.evaluate(() => window.applicationFixture.element.requestedDelivery), '');
  // Selecting an exact delivery leaves the company scope for the exact view.
  await page.evaluate(() => {
    window.applicationFixture.element.requestedDelivery = 'b'.repeat(32);
  });
  await page.locator('.soda-application-heading').getByText('Example-company', {exact: true}).waitFor();
  await page.waitForFunction(
    (marker) =>
      [...document.querySelectorAll('soda-applications p')].filter((el) => el.textContent === marker).length === 0,
    'Showing the current company application; previews stay separate.'
  );
  assert.equal(
    await page.getByText('Showing the current company application; previews stay separate.', {exact: true}).count(),
    0
  );
  assert.equal(
    await page.evaluate(
      () => window.applicationFixture.calls.filter((call) => call.body && call.body.action !== 'inspect').length
    ),
    0
  );
});

test('a session switch drops the company scope instead of hiding the new previews', async (t) => {
  const page = await browser.newPage();
  t.after(() => page.close());
  await page.goto(server.url.href);
  await page.waitForFunction(() => typeof window.createApplicationFixture === 'function');
  await page.evaluate(async () => {
    await window.createApplicationFixture();
    const app = window.applicationFixture.element;
    const stopped = (suffix: string, origin: string) => ({
      id: 'a' + suffix.repeat(24),
      revision: Number(suffix),
      container: suffix.repeat(64),
      image: 'sha256:' + '6'.repeat(64),
      origin,
      running: false,
      ready: false,
      bundle_file_sha256: 'b'.repeat(64),
    });
    const record = (
      suffix: string,
      environment: string,
      name: string,
      origin: string,
      observation: Record<string, unknown> | null
    ) => ({
      environment,
      application: {
        id: 'a' + suffix.repeat(24),
        revision: Number(suffix),
        name,
        artifact_sha256: '2'.repeat(64),
        bundle_sha256: '3'.repeat(64),
        tools_sha256: '4'.repeat(64),
        runtime_image: 'sha256:' + '5'.repeat(64),
        image: 'sha256:' + '6'.repeat(64),
        data: 'a' + '7'.repeat(24),
        owner_team: '12345678-1234-1234-1234-123456789abc',
        origin,
        port: 20000 + Number(suffix),
      },
      administrator_id: '41',
      operator: false,
      observation,
      can_embed: true,
    });
    const rows = [
      record(
        '1',
        'company',
        'Example-company',
        'https://company.example.test',
        stopped('1', 'https://company.example.test')
      ),
      record('2', 'preview', 'Example-preview', 'https://preview.example.test', {
        ...stopped('2', 'https://preview.example.test'),
        running: true,
        ready: true,
      }),
    ];
    app.request = async (path, body) => {
      window.applicationFixture.calls.push({path, ...(body ? {body} : {})});
      if (!body) return {applications: rows, complete: true};
      if (body.action === 'inspect')
        return rows.find((row) => path === `/api/applications/${row.application.id}/actions`);
      throw Error('scope test must not write');
    };
    app.requestedDelivery = 'd'.repeat(32);
    app.expanded = true;
  });
  await page
    .getByText(
      'An application matching this exact result is not available to you. The current company application may be a different version.',
      {exact: true}
    )
    .waitFor();
  await page.getByRole('button', {name: 'Open current company application', exact: true}).click();
  await page.locator('.soda-application-heading').getByText('Example-company', {exact: true}).waitFor();
  await page.getByText('Showing the current company application; previews stay separate.', {exact: true}).waitFor();
  assert.equal(await page.locator('.soda-application-heading').getByText('Example-preview', {exact: true}).count(), 0);
  // A session switch invalidates the inventory the scope was selected over:
  // the scope drops with it and the new context shows its previews.
  await page.evaluate(() => {
    window.applicationFixture.element.sessionKey = 'generation-2';
  });
  await page.locator('.soda-application-heading').getByText('Example-preview', {exact: true}).waitFor();
  assert.equal(
    await page.getByText('Showing the current company application; previews stay separate.', {exact: true}).count(),
    0
  );
  assert.equal(
    await page.evaluate(
      () => window.applicationFixture.calls.filter((call) => call.body && call.body.action !== 'inspect').length
    ),
    0
  );
});

test('a reload starts outside the company scope instead of keeping a stale filter', async (t) => {
  const page = await browser.newPage();
  t.after(() => page.close());
  const scopedRows = async (delivery: string) => {
    await page.evaluate(async (deliveryId: string) => {
      await window.createApplicationFixture();
      const app = window.applicationFixture.element;
      const stopped = (suffix: string, origin: string) => ({
        id: 'a' + suffix.repeat(24),
        revision: Number(suffix),
        container: suffix.repeat(64),
        image: 'sha256:' + '6'.repeat(64),
        origin,
        running: false,
        ready: false,
        bundle_file_sha256: 'b'.repeat(64),
      });
      const record = (
        suffix: string,
        environment: string,
        name: string,
        origin: string,
        observation: Record<string, unknown> | null
      ) => ({
        environment,
        application: {
          id: 'a' + suffix.repeat(24),
          revision: Number(suffix),
          name,
          artifact_sha256: '2'.repeat(64),
          bundle_sha256: '3'.repeat(64),
          tools_sha256: '4'.repeat(64),
          runtime_image: 'sha256:' + '5'.repeat(64),
          image: 'sha256:' + '6'.repeat(64),
          data: 'a' + '7'.repeat(24),
          owner_team: '12345678-1234-1234-1234-123456789abc',
          origin,
          port: 20000 + Number(suffix),
        },
        administrator_id: '41',
        operator: false,
        observation,
        can_embed: true,
      });
      const rows = [
        record(
          '1',
          'company',
          'Example-company',
          'https://company.example.test',
          stopped('1', 'https://company.example.test')
        ),
        record('2', 'preview', 'Example-preview', 'https://preview.example.test', {
          ...stopped('2', 'https://preview.example.test'),
          running: true,
          ready: true,
        }),
      ];
      app.request = async (path, body) => {
        window.applicationFixture.calls.push({path, ...(body ? {body} : {})});
        if (!body) return {applications: rows, complete: true};
        if (body.action === 'inspect')
          return rows.find((row) => path === `/api/applications/${row.application.id}/actions`);
        throw Error('scope test must not write');
      };
      app.requestedDelivery = deliveryId;
      app.expanded = true;
    }, delivery);
  };
  await page.goto(server.url.href);
  await page.waitForFunction(() => typeof window.createApplicationFixture === 'function');
  await scopedRows('d'.repeat(32));
  await page.getByRole('button', {name: 'Open current company application', exact: true}).click();
  await page.getByText('Showing the current company application; previews stay separate.', {exact: true}).waitFor();
  // A reload is a fresh mount with no retained selection: the unfiltered
  // inventory returns and the company application is reopened explicitly,
  // never assumed.
  await page.reload();
  await page.waitForFunction(() => typeof window.createApplicationFixture === 'function');
  await scopedRows('');
  await page.locator('.soda-application-heading').getByText('Example-preview', {exact: true}).waitFor();
  await page.locator('.soda-application-heading').getByText('Example-company', {exact: true}).waitFor();
  assert.equal(
    await page.getByText('Showing the current company application; previews stay separate.', {exact: true}).count(),
    0
  );
});

test('resuming the app surface re-applies the retained company scope without another click', async (t) => {
  const page = await browser.newPage();
  t.after(() => page.close());
  await page.goto(server.url.href);
  await page.waitForFunction(() => typeof window.createApplicationFixture === 'function');
  await page.evaluate(async () => {
    await window.createApplicationFixture();
    const app = window.applicationFixture.element;
    const stopped = (suffix: string, origin: string) => ({
      id: 'a' + suffix.repeat(24),
      revision: Number(suffix),
      container: suffix.repeat(64),
      image: 'sha256:' + '6'.repeat(64),
      origin,
      running: false,
      ready: false,
      bundle_file_sha256: 'b'.repeat(64),
    });
    const record = (
      suffix: string,
      environment: string,
      name: string,
      origin: string,
      observation: Record<string, unknown> | null
    ) => ({
      environment,
      application: {
        id: 'a' + suffix.repeat(24),
        revision: Number(suffix),
        name,
        artifact_sha256: '2'.repeat(64),
        bundle_sha256: '3'.repeat(64),
        tools_sha256: '4'.repeat(64),
        runtime_image: 'sha256:' + '5'.repeat(64),
        image: 'sha256:' + '6'.repeat(64),
        data: 'a' + '7'.repeat(24),
        owner_team: '12345678-1234-1234-1234-123456789abc',
        origin,
        port: 20000 + Number(suffix),
      },
      administrator_id: '41',
      operator: false,
      observation,
      can_embed: true,
      project: {repository: '7', name: 'northstar/equipment', path: '/northstar/equipment'},
    });
    const rows = [
      record(
        '1',
        'company',
        'Example-company',
        'https://company.example.test',
        stopped('1', 'https://company.example.test')
      ),
      record('2', 'preview', 'Example-preview', 'https://preview.example.test', {
        ...stopped('2', 'https://preview.example.test'),
        running: true,
        ready: true,
      }),
    ];
    app.request = async (path, body) => {
      window.applicationFixture.calls.push({path, ...(body ? {body} : {})});
      if (!body) return {applications: rows, complete: true};
      if (body.action === 'inspect')
        return rows.find((row) => path === `/api/applications/${row.application.id}/actions`);
      throw Error('scope test must not write');
    };
    app.repository = '7';
    app.requestedDelivery = 'd'.repeat(32);
    app.expanded = true;
  });
  await page.getByRole('button', {name: 'Open current company application', exact: true}).click();
  await page.getByText('Showing the current company application; previews stay separate.', {exact: true}).waitFor();
  // The workspace retains the opened selection across a suspend and passes
  // it back unchanged.
  await page.evaluate(() => {
    window.applicationFixture.element.requestedCompanyScope = true;
  });
  await page.evaluate(() => {
    window.applicationFixture.element.active = false;
  });
  // A suspend drops the live scope with the discarded inventory.
  await page.waitForFunction(
    (marker) =>
      [...document.querySelectorAll('soda-applications p')].filter((el) => el.textContent === marker).length === 0,
    'Showing the current company application; previews stay separate.'
  );
  // Resuming re-applies the retained selection without another click: the
  // company surface returns and the preview stays hidden.
  await page.evaluate(() => {
    window.applicationFixture.element.active = true;
  });
  await page.getByText('Showing the current company application; previews stay separate.', {exact: true}).waitFor();
  await page.locator('.soda-application-heading').getByText('Example-company', {exact: true}).waitFor();
  assert.equal(await page.locator('.soda-application-heading').getByText('Example-preview', {exact: true}).count(), 0);
  assert.equal(
    await page.evaluate(
      () => window.applicationFixture.calls.filter((call) => call.body && call.body.action !== 'inspect').length
    ),
    0
  );
});

test('resuming onto an exact selection keeps the exact view unscoped', async (t) => {
  const page = await browser.newPage();
  t.after(() => page.close());
  await page.goto(server.url.href);
  await page.waitForFunction(() => typeof window.createApplicationFixture === 'function');
  await page.evaluate(async () => {
    await window.createApplicationFixture();
    const app = window.applicationFixture.element;
    const stopped = (suffix: string, origin: string) => ({
      id: 'a' + suffix.repeat(24),
      revision: Number(suffix),
      container: suffix.repeat(64),
      image: 'sha256:' + '6'.repeat(64),
      origin,
      running: false,
      ready: false,
      bundle_file_sha256: 'b'.repeat(64),
    });
    const record = (
      suffix: string,
      environment: string,
      name: string,
      origin: string,
      observation: Record<string, unknown> | null
    ) => ({
      environment,
      application: {
        id: 'a' + suffix.repeat(24),
        revision: Number(suffix),
        name,
        artifact_sha256: '2'.repeat(64),
        bundle_sha256: '3'.repeat(64),
        tools_sha256: '4'.repeat(64),
        runtime_image: 'sha256:' + '5'.repeat(64),
        image: 'sha256:' + '6'.repeat(64),
        data: 'a' + '7'.repeat(24),
        owner_team: '12345678-1234-1234-1234-123456789abc',
        origin,
        port: 20000 + Number(suffix),
      },
      administrator_id: '41',
      operator: false,
      observation,
      can_embed: true,
      project: {repository: '7', name: 'northstar/equipment', path: '/northstar/equipment'},
    });
    const rows = [
      record(
        '1',
        'company',
        'Example-company',
        'https://company.example.test',
        stopped('1', 'https://company.example.test')
      ),
      record('2', 'preview', 'Example-preview', 'https://preview.example.test', {
        ...stopped('2', 'https://preview.example.test'),
        running: true,
        ready: true,
      }),
    ];
    app.request = async (path, body) => {
      window.applicationFixture.calls.push({path, ...(body ? {body} : {})});
      if (!body) return {applications: rows, complete: true};
      if (body.action === 'inspect')
        return rows.find((row) => path === `/api/applications/${row.application.id}/actions`);
      throw Error('scope test must not write');
    };
    app.repository = '7';
    app.requestedDelivery = 'd'.repeat(32);
    app.expanded = true;
  });
  await page.getByRole('button', {name: 'Open current company application', exact: true}).click();
  await page.getByText('Showing the current company application; previews stay separate.', {exact: true}).waitFor();
  await page.evaluate(() => {
    window.applicationFixture.element.requestedCompanyScope = true;
  });
  // Selecting an exact delivery leaves the company scope for the exact view.
  await page.evaluate(() => {
    window.applicationFixture.element.requestedDelivery = 'b'.repeat(32);
  });
  await page
    .getByText(
      'An application matching this exact result is not available to you. The current company application may be a different version.',
      {exact: true}
    )
    .waitFor();
  // A suspend and resume around the exact selection never re-applies the
  // retained scope: the exact view stays unscoped.
  await page.evaluate(() => {
    window.applicationFixture.element.active = false;
  });
  await page.evaluate(() => {
    window.applicationFixture.element.active = true;
  });
  await page
    .getByText(
      'An application matching this exact result is not available to you. The current company application may be a different version.',
      {exact: true}
    )
    .waitFor();
  assert.equal(
    await page.getByText('Showing the current company application; previews stay separate.', {exact: true}).count(),
    0
  );
  assert.equal(
    await page.evaluate(
      () => window.applicationFixture.calls.filter((call) => call.body && call.body.action !== 'inspect').length
    ),
    0
  );
});

test('switching accounts drops the retained company scope without re-applying it', async (t) => {
  const page = await browser.newPage();
  t.after(() => page.close());
  await page.goto(server.url.href);
  await page.waitForFunction(() => typeof window.createApplicationFixture === 'function');
  await page.evaluate(async () => {
    await window.createApplicationFixture();
    const app = window.applicationFixture.element;
    const stopped = (suffix: string, origin: string) => ({
      id: 'a' + suffix.repeat(24),
      revision: Number(suffix),
      container: suffix.repeat(64),
      image: 'sha256:' + '6'.repeat(64),
      origin,
      running: false,
      ready: false,
      bundle_file_sha256: 'b'.repeat(64),
    });
    const record = (
      suffix: string,
      environment: string,
      name: string,
      origin: string,
      observation: Record<string, unknown> | null
    ) => ({
      environment,
      application: {
        id: 'a' + suffix.repeat(24),
        revision: Number(suffix),
        name,
        artifact_sha256: '2'.repeat(64),
        bundle_sha256: '3'.repeat(64),
        tools_sha256: '4'.repeat(64),
        runtime_image: 'sha256:' + '5'.repeat(64),
        image: 'sha256:' + '6'.repeat(64),
        data: 'a' + '7'.repeat(24),
        owner_team: '12345678-1234-1234-1234-123456789abc',
        origin,
        port: 20000 + Number(suffix),
      },
      administrator_id: '41',
      operator: false,
      observation,
      can_embed: true,
      project: {repository: '7', name: 'northstar/equipment', path: '/northstar/equipment'},
    });
    const rows = [
      record(
        '1',
        'company',
        'Example-company',
        'https://company.example.test',
        stopped('1', 'https://company.example.test')
      ),
      record('2', 'preview', 'Example-preview', 'https://preview.example.test', {
        ...stopped('2', 'https://preview.example.test'),
        running: true,
        ready: true,
      }),
    ];
    app.request = async (path, body) => {
      window.applicationFixture.calls.push({path, ...(body ? {body} : {})});
      if (!body) return {applications: rows, complete: true};
      if (body.action === 'inspect')
        return rows.find((row) => path === `/api/applications/${row.application.id}/actions`);
      throw Error('scope test must not write');
    };
    app.repository = '7';
    app.requestedDelivery = 'd'.repeat(32);
    app.expanded = true;
  });
  await page.getByRole('button', {name: 'Open current company application', exact: true}).click();
  await page.getByText('Showing the current company application; previews stay separate.', {exact: true}).waitFor();
  await page.evaluate(() => {
    window.applicationFixture.element.requestedCompanyScope = true;
  });
  // A session switch invalidates the inventory the scope was selected over:
  // the scope drops with it and the unchanged retained flag does not bring
  // it back without an active transition.
  await page.evaluate(() => {
    window.applicationFixture.element.sessionKey = 'generation-2';
  });
  await page.waitForFunction(
    (marker) =>
      [...document.querySelectorAll('.soda-application-heading')].some((el) =>
        el.textContent?.includes('Example-preview')
      ) && [...document.querySelectorAll('soda-applications p')].filter((el) => el.textContent === marker).length === 0,
    'Showing the current company application; previews stay separate.'
  );
  assert.equal(
    await page.evaluate(
      () => window.applicationFixture.calls.filter((call) => call.body && call.body.action !== 'inspect').length
    ),
    0
  );
});

test('expired owner setup offers no renew action with inspect and refresh and preserved credential copy', async (t) => {
  const page = await browser.newPage();
  t.after(() => page.close());
  const errors: string[] = [];
  page.on('pageerror', (error) => errors.push(error.message));
  await page.goto(server.url.href);
  await page.waitForFunction(() => typeof window.createApplicationFixture === 'function');
  await page.evaluate(() => window.createApplicationFixture(true));
  await page.getByText('Applications', {exact: true}).click();
  await page.getByText('Prepare or update an application', {exact: true}).click();
  await page.getByText('Set up a new application (operator)', {exact: true}).click();
  await page.getByLabel('Admitted application references', {exact: true}).fill(
    JSON.stringify({
      name: 'NewApp',
      artifact_sha256: '1'.repeat(64),
      bundle_sha256: '2'.repeat(64),
      tools_sha256: '3'.repeat(64),
      runtime_image: 'sha256:' + '4'.repeat(64),
      image: 'sha256:' + '5'.repeat(64),
      origin: 'https://new.example.test',
      port: 24000,
    })
  );
  await page.getByLabel('Soda administrator user ID', {exact: true}).fill('41');
  await page.getByRole('button', {name: 'Begin owner setup', exact: true}).click();
  await page.getByRole('link', {name: 'Create application owner ↗', exact: true}).waitFor();
  // Expire the retained generation: the host can no longer mint a fresh owner link for it.
  await page.evaluate(() => {
    window.applicationFixture.setupExpiresAt = Math.floor(Date.now() / 1000) - 1;
  });
  const postsBefore = await page.evaluate(() => window.applicationFixture.calls.filter((call) => !!call.body).length);
  await page.getByRole('button', {name: 'Refresh owner setups', exact: true}).click();
  await page
    .getByText(
      'The link has expired. Inspect this setup, then finish if its owner was already created; if no owner was created, stop and cancel this setup. Its owner credential, data, and reserved address stay retained. Inspect or refresh this same setup to confirm its state; renewal is unavailable until the owner claim can be confirmed.',
      {exact: true}
    )
    .waitFor();
  // No renew action is offered on the fresh expired visit: no arm button, no
  // confirm block, and the stale owner link is dropped with no fresh link.
  assert.equal(await page.getByRole('button', {name: /Renew setup link/, exact: false}).count(), 0);
  assert.equal(await page.getByRole('button', {name: /Confirm renewal/, exact: false}).count(), 0);
  assert.equal(await page.getByRole('link', {name: 'Create application owner ↗', exact: true}).count(), 0);
  // Inspect and refresh stay available; opening the expired link stays fenced.
  assert(await page.getByRole('button', {name: 'Inspect owner setup NewApp', exact: true}).isEnabled());
  assert(await page.getByRole('button', {name: 'Refresh owner setups', exact: true}).isEnabled());
  assert(await page.getByRole('button', {name: 'Open owner setup NewApp', exact: true}).isDisabled());
  // No old renewal promise survives anywhere on the expired row.
  assert.equal(await page.getByText(/fresh 30-minute window/, {exact: false}).count(), 0);
  assert.equal(await page.getByText(/at most three times/, {exact: false}).count(), 0);
  assert.equal(await page.getByText(/or ask the operator to renew it/, {exact: false}).count(), 0);
  assert.equal(await page.getByText(/only an operator renewal re-arms/, {exact: false}).count(), 0);
  // The refresh itself is a read: no new POSTs, and no renew POST in particular.
  assert.equal(
    await page.evaluate(() => window.applicationFixture.calls.filter((call) => !!call.body).length),
    postsBefore
  );
  assert.equal(
    await page.evaluate(() => window.applicationFixture.calls.filter((call) => call.body?.action === 'renew').length),
    0
  );
  await page.waitForTimeout(300);
  assert.equal(
    await page.evaluate(() => window.applicationFixture.calls.filter((call) => call.body?.action === 'renew').length),
    0
  );
  assert.deepEqual(errors, []);
});

test('expired owner setup still offers no renew after reload with zero writes', async (t) => {
  const page = await browser.newPage();
  t.after(() => page.close());
  const errors: string[] = [];
  page.on('pageerror', (error) => errors.push(error.message));
  await page.goto(server.url.href);
  await page.waitForFunction(() => typeof window.createApplicationFixture === 'function');
  await page.evaluate(() => window.createApplicationFixture(true));
  await page.getByText('Applications', {exact: true}).click();
  await page.getByText('Prepare or update an application', {exact: true}).click();
  await page.getByText('Set up a new application (operator)', {exact: true}).click();
  await page.getByLabel('Admitted application references', {exact: true}).fill(
    JSON.stringify({
      name: 'NewApp',
      artifact_sha256: '1'.repeat(64),
      bundle_sha256: '2'.repeat(64),
      tools_sha256: '3'.repeat(64),
      runtime_image: 'sha256:' + '4'.repeat(64),
      image: 'sha256:' + '5'.repeat(64),
      origin: 'https://new.example.test',
      port: 24000,
    })
  );
  await page.getByLabel('Soda administrator user ID', {exact: true}).fill('41');
  await page.getByRole('button', {name: 'Begin owner setup', exact: true}).click();
  await page.getByRole('link', {name: 'Create application owner ↗', exact: true}).waitFor();
  await page.evaluate(() => {
    window.applicationFixture.setupExpiresAt = Math.floor(Date.now() / 1000) - 1;
  });
  await page.getByRole('button', {name: 'Refresh owner setups', exact: true}).click();
  await page.getByText(/The link has expired/, {exact: false}).waitFor();
  assert.equal(await page.getByRole('button', {name: /Renew setup link/, exact: false}).count(), 0);
  // A reload is a fresh mount: expanding the owner setups issues only
  // inventory reads, with zero state-changing writes and no renew offer.
  // (Status inspect reads carry a body by this file's long-standing
  // convention; only non-inspect POSTs count as writes here.)
  await page.reload();
  await page.waitForFunction(() => typeof window.createApplicationFixture === 'function');
  await page.evaluate(() => window.createApplicationFixture(true));
  await page.getByText('Applications', {exact: true}).click();
  await page.getByText('Prepare or update an application', {exact: true}).click();
  await page.getByText('Set up a new application (operator)', {exact: true}).click();
  await page.waitForFunction(() =>
    window.applicationFixture.calls.some((call) => call.path === '/api/applications/setups' && !call.body)
  );
  assert.equal(
    await page.evaluate(
      () => window.applicationFixture.calls.filter((call) => call.body && call.body.action !== 'inspect').length
    ),
    0
  );
  assert.equal(
    await page.evaluate(() => window.applicationFixture.calls.filter((call) => call.body?.action === 'renew').length),
    0
  );
  assert.equal(await page.getByRole('button', {name: /Renew setup link/, exact: false}).count(), 0);
  assert.equal(await page.getByRole('button', {name: /Confirm renewal/, exact: false}).count(), 0);
  assert(await page.getByRole('button', {name: 'Refresh owner setups', exact: true}).isEnabled());
  // Re-establishing the expired generation after the reload still offers no renew.
  await page.getByLabel('Admitted application references', {exact: true}).fill(
    JSON.stringify({
      name: 'NewApp',
      artifact_sha256: '1'.repeat(64),
      bundle_sha256: '2'.repeat(64),
      tools_sha256: '3'.repeat(64),
      runtime_image: 'sha256:' + '4'.repeat(64),
      image: 'sha256:' + '5'.repeat(64),
      origin: 'https://new.example.test',
      port: 24000,
    })
  );
  await page.getByLabel('Soda administrator user ID', {exact: true}).fill('41');
  await page.getByRole('button', {name: 'Begin owner setup', exact: true}).click();
  await page.getByRole('link', {name: 'Create application owner ↗', exact: true}).waitFor();
  await page.evaluate(() => {
    window.applicationFixture.setupExpiresAt = Math.floor(Date.now() / 1000) - 1;
  });
  await page.getByRole('button', {name: 'Refresh owner setups', exact: true}).click();
  await page
    .getByText(
      'The link has expired. Inspect this setup, then finish if its owner was already created; if no owner was created, stop and cancel this setup. Its owner credential, data, and reserved address stay retained. Inspect or refresh this same setup to confirm its state; renewal is unavailable until the owner claim can be confirmed.',
      {exact: true}
    )
    .waitFor();
  assert.equal(await page.getByRole('button', {name: /Renew setup link/, exact: false}).count(), 0);
  assert.equal(await page.getByRole('button', {name: /Confirm renewal/, exact: false}).count(), 0);
  assert(await page.getByRole('button', {name: 'Inspect owner setup NewApp', exact: true}).isEnabled());
  assert.equal(
    await page.evaluate(() => window.applicationFixture.calls.filter((call) => call.body?.action === 'renew').length),
    0
  );
  assert.equal(await page.getByText(/fresh 30-minute window/, {exact: false}).count(), 0);
  assert.deepEqual(errors, []);
});

test('expired owner setup still offers no renew on re-entry with zero writes', async (t) => {
  const page = await browser.newPage();
  t.after(() => page.close());
  const errors: string[] = [];
  page.on('pageerror', (error) => errors.push(error.message));
  await page.goto(server.url.href);
  await page.waitForFunction(() => typeof window.createApplicationFixture === 'function');
  await page.evaluate(() => window.createApplicationFixture(true));
  await page.getByText('Applications', {exact: true}).click();
  await page.getByText('Prepare or update an application', {exact: true}).click();
  await page.getByText('Set up a new application (operator)', {exact: true}).click();
  await page.getByLabel('Admitted application references', {exact: true}).fill(
    JSON.stringify({
      name: 'NewApp',
      artifact_sha256: '1'.repeat(64),
      bundle_sha256: '2'.repeat(64),
      tools_sha256: '3'.repeat(64),
      runtime_image: 'sha256:' + '4'.repeat(64),
      image: 'sha256:' + '5'.repeat(64),
      origin: 'https://new.example.test',
      port: 24000,
    })
  );
  await page.getByLabel('Soda administrator user ID', {exact: true}).fill('41');
  await page.getByRole('button', {name: 'Begin owner setup', exact: true}).click();
  await page.getByRole('link', {name: 'Create application owner ↗', exact: true}).waitFor();
  await page.evaluate(() => {
    window.applicationFixture.setupExpiresAt = Math.floor(Date.now() / 1000) - 1;
  });
  await page.getByRole('button', {name: 'Refresh owner setups', exact: true}).click();
  await page.getByText(/The link has expired/, {exact: false}).waitFor();
  assert.equal(await page.getByRole('button', {name: /Renew setup link/, exact: false}).count(), 0);
  const postsBefore = await page.evaluate(() => window.applicationFixture.calls.filter((call) => !!call.body).length);
  const callsBefore = await page.evaluate(() => window.applicationFixture.calls.length);
  // Collapsing the section hides the retained generation without any request.
  await page.getByText('Set up a new application (operator)', {exact: true}).click();
  assert.equal(await page.getByRole('button', {name: 'Refresh owner setups', exact: true}).count(), 0);
  assert.equal(await page.getByRole('button', {name: /Renew setup link/, exact: false}).count(), 0);
  assert.equal(await page.getByRole('button', {name: /Confirm renewal/, exact: false}).count(), 0);
  assert.equal(await page.evaluate(() => window.applicationFixture.calls.length), callsBefore);
  // Re-entering re-renders the retained expired generation from current bytes
  // with inspect available and still no renew offer — and still no request.
  await page.getByText('Set up a new application (operator)', {exact: true}).click();
  await page
    .getByText(
      'The link has expired. Inspect this setup, then finish if its owner was already created; if no owner was created, stop and cancel this setup. Its owner credential, data, and reserved address stay retained. Inspect or refresh this same setup to confirm its state; renewal is unavailable until the owner claim can be confirmed.',
      {exact: true}
    )
    .waitFor();
  assert.equal(await page.evaluate(() => window.applicationFixture.calls.length), callsBefore);
  assert.equal(await page.getByRole('button', {name: /Renew setup link/, exact: false}).count(), 0);
  assert.equal(await page.getByRole('button', {name: /Confirm renewal/, exact: false}).count(), 0);
  assert.equal(await page.getByRole('link', {name: 'Create application owner ↗', exact: true}).count(), 0);
  assert(await page.getByRole('button', {name: 'Inspect owner setup NewApp', exact: true}).isEnabled());
  assert(await page.getByRole('button', {name: 'Refresh owner setups', exact: true}).isEnabled());
  assert.equal(await page.getByText(/fresh 30-minute window/, {exact: false}).count(), 0);
  // Collapsing and re-entering issue no request at all: no new POSTs, no renew POST.
  assert.equal(
    await page.evaluate(() => window.applicationFixture.calls.filter((call) => !!call.body).length),
    postsBefore
  );
  assert.equal(
    await page.evaluate(() => window.applicationFixture.calls.filter((call) => call.body?.action === 'renew').length),
    0
  );
  await page.waitForTimeout(300);
  assert.equal(
    await page.evaluate(() => window.applicationFixture.calls.filter((call) => call.body?.action === 'renew').length),
    0
  );
  assert.deepEqual(errors, []);
});

test('later pending generations stay visible with inspect and refresh and no renew', async (t) => {
  const page = await browser.newPage();
  t.after(() => page.close());
  const errors: string[] = [];
  page.on('pageerror', (error) => errors.push(error.message));
  await page.goto(server.url.href);
  await page.waitForFunction(() => typeof window.createApplicationFixture === 'function');
  await page.evaluate(() => window.createApplicationFixture(true));
  await page.getByText('Applications', {exact: true}).click();
  await page.getByText('Prepare or update an application', {exact: true}).click();
  await page.getByText('Set up a new application (operator)', {exact: true}).click();
  await page.getByLabel('Admitted application references', {exact: true}).fill(
    JSON.stringify({
      name: 'NewApp',
      artifact_sha256: '1'.repeat(64),
      bundle_sha256: '2'.repeat(64),
      tools_sha256: '3'.repeat(64),
      runtime_image: 'sha256:' + '4'.repeat(64),
      image: 'sha256:' + '5'.repeat(64),
      origin: 'https://new.example.test',
      port: 24000,
    })
  );
  await page.getByLabel('Soda administrator user ID', {exact: true}).fill('41');
  await page.getByRole('button', {name: 'Begin owner setup', exact: true}).click();
  await page.getByRole('link', {name: 'Create application owner ↗', exact: true}).waitFor();
  await page.evaluate(() => {
    window.applicationFixture.setupExpiresAt = Math.floor(Date.now() / 1000) - 1;
  });
  await page.getByRole('button', {name: 'Refresh owner setups', exact: true}).click();
  await page.getByText(/The link has expired/, {exact: false}).waitFor();
  // A later pending generation arrives while the first stays expired. The
  // inventory serves both; no remembered refusal may hide the later one (the
  // 865 restore rule: availability derives from current bytes per scope).
  await page.evaluate(() => {
    const app = window.applicationFixture.element,
      original = app.request!,
      setup = document.querySelector('soda-application-setup') as unknown as {
        request: typeof app.request;
      } | null;
    const second = {
      id: 'a' + 'd'.repeat(24),
      application: {id: 'a' + 'e'.repeat(24), name: 'NewApp2', origin: 'https://new2.example.test'},
      administrator_id: 41,
      expires_at: Math.floor(Date.now() / 1000) + 600,
      state: 'pending',
      observation: null,
    };
    const wrapped = (async (path, body, signal) => {
      if (path === '/api/applications/setups' && !body) {
        const value = (await original(path, body, signal)) as {setups: unknown[]};
        return {setups: [...value.setups, second]};
      }
      return original(path, body, signal);
    }) as typeof app.request;
    app.request = wrapped;
    if (setup) setup.request = wrapped;
  });
  await page.getByRole('button', {name: 'Refresh owner setups', exact: true}).click();
  await page.getByText('NewApp2', {exact: true}).waitFor();
  // Both generations stay visible: the expired first row and the later one.
  await page.getByText('NewApp', {exact: true}).waitFor();
  // Only the expired first row names expiry; the later generation is not
  // hidden behind it and carries no expiry copy.
  assert.equal(await page.getByText(/The link has expired/, {exact: false}).count(), 1);
  // Neither generation offers renew; both offer inspect and refresh stays available.
  assert.equal(await page.getByRole('button', {name: /Renew setup link/, exact: false}).count(), 0);
  assert.equal(await page.getByRole('button', {name: /Confirm renewal/, exact: false}).count(), 0);
  assert(await page.getByRole('button', {name: 'Inspect owner setup NewApp', exact: true}).isEnabled());
  assert(await page.getByRole('button', {name: 'Inspect owner setup NewApp2', exact: true}).isEnabled());
  assert(await page.getByRole('button', {name: 'Refresh owner setups', exact: true}).isEnabled());
  // Per-generation affordances follow current bytes: the expired row cannot be
  // opened while the later pending generation can.
  assert(await page.getByRole('button', {name: 'Open owner setup NewApp', exact: true}).isDisabled());
  assert(await page.getByRole('button', {name: 'Open owner setup NewApp2', exact: true}).isEnabled());
  assert.equal(await page.getByText(/fresh 30-minute window/, {exact: false}).count(), 0);
  assert.equal(
    await page.evaluate(() => window.applicationFixture.calls.filter((call) => call.body?.action === 'renew').length),
    0
  );
  await page.waitForTimeout(300);
  assert.equal(
    await page.evaluate(() => window.applicationFixture.calls.filter((call) => call.body?.action === 'renew').length),
    0
  );
  assert.deepEqual(errors, []);
});
