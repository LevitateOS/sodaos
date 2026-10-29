import assert from 'node:assert/strict';
import test from 'node:test';
import {chromium} from 'playwright';

const codexURL = 'https://auth.openai.com/codex/device';
const museURL = 'https://auth.meta.com/oauth/device/';

/** A fake dashboard identity API whose start is slow, like the real CLI launch. */
function identityServer(script: string, page = '<!doctype html><html><body></body></html>') {
  const state = {completed: false, expired: false, starts: 0, cancels: 0, reads: 0, startDelay: 600};
  const attempts = new Map<string, {provider_id: string; id: string; verification_url: string; user_code: string}>();
  const live = (provider: string) => {
    const attempt = attempts.get(provider)!;
    if (provider === 'codex' && state.completed) return undefined;
    return {...attempt, state: state.expired && provider === 'muse' ? 'expired' : 'pending'};
  };
  const server = Bun.serve({
    hostname: '127.0.0.1',
    port: 0,
    async fetch(request) {
      const url = new URL(request.url);
      if (url.pathname === '/widget.js') return new Response(script, {headers: {'Content-Type': 'text/javascript'}});
      if (url.pathname === '/api/identity/connections') {
        state.reads++;
        return Response.json(
          state.completed
            ? [
                {
                  provider_id: 'codex',
                  id: 'connected',
                  owner_id: '1',
                  label: 'Codex',
                  email: 'me@example.test',
                  plan: 'pro',
                  state: 'ready',
                },
              ]
            : []
        );
      }
      if (url.pathname === '/api/identity/enrollments' && request.method === 'POST') {
        const body = (await request.json()) as {
          provider_id: string;
          label: string;
          confirm_credential_exposure: boolean;
        };
        assert.equal(body.confirm_credential_exposure, true);
        assert(body.label);
        state.starts++;
        state.expired = false;
        await Bun.sleep(state.startDelay);
        attempts.set(body.provider_id, {
          provider_id: body.provider_id,
          id: body.provider_id,
          verification_url: body.provider_id === 'codex' ? codexURL : museURL,
          user_code: 'ABCD-EFGH',
        });
        return Response.json(live(body.provider_id));
      }
      if (url.pathname === '/api/identity/enrollments')
        return Response.json([...attempts.keys()].map(live).filter((attempt) => attempt !== undefined));
      if (url.pathname.endsWith('/cancel')) {
        state.cancels++;
        attempts.delete(url.pathname.split('/').at(-2)!);
        return Response.json({});
      }
      return new Response(page, {headers: {'Content-Type': 'text/html'}});
    },
  });
  return {server, state};
}

async function launch(t: {after: (fn: () => unknown) => void}) {
  const browser = await chromium.launch({
    headless: true,
    chromiumSandbox: true,
    ...(process.env.CHROME ? {executablePath: process.env.CHROME} : {}),
  });
  t.after(() => browser.close());
  return browser.newPage();
}

test('AI accounts shows each sign-in state from preparing through connected, expiry and cancel', async (t) => {
  const build = await Bun.build({entrypoints: ['frontend/spaces/soda-providers.ts'], target: 'browser'});
  assert(build.success, build.logs.join('\n'));
  const {server, state} = identityServer(
    await build.outputs[0]!.text(),
    '<!doctype html><html><head><style>dialog { display:flex; }</style></head><body><textarea aria-label="Work draft">keep this draft</textarea></body></html>'
  );
  t.after(() => server.stop(true));
  const page = await launch(t);
  const mount = async () => {
    await page.goto(server.url.toString());
    await page.evaluate(async () => {
      const path = '/widget.js';
      const module = await import(path);
      const widget = new module.SodaProviders();
      document.body.append(widget);
      widget.transport = {request: (path: string, init: RequestInit) => fetch('/api/' + path, init)};
      (window as unknown as {widget: {open(): Promise<void>}}).widget = widget;
    });
  };
  const open = () => page.evaluate(() => (window as unknown as {widget: {open(): Promise<void>}}).widget.open());
  await mount();
  await open();
  const dialog = page.getByRole('dialog', {name: 'AI accounts', exact: true});
  assert.equal(
    await dialog.evaluate((node) => getComputedStyle(node).display),
    'block',
    'Forge styles must not squeeze the dialog'
  );
  assert(state.reads >= 1);
  const codex = dialog.locator('.soda-provider-card').filter({hasText: 'Codex'});
  const muse = dialog.locator('.soda-provider-card').filter({hasText: 'Muse'});
  await codex.getByText('Not connected', {exact: true}).waitFor();

  // Clicking Connect shows progress at once while the provider prepares a code.
  await codex.getByRole('button', {name: 'Connect Codex', exact: true}).click();
  await codex.getByText('Preparing sign-in…', {exact: true}).waitFor();
  await codex.getByText(/Preparing your Codex sign-in/).waitFor();
  await codex.getByText('ABCD-EFGH', {exact: true}).waitFor();
  assert.equal(await codex.getByRole('link', {name: 'Open sign-in page'}).getAttribute('href'), codexURL);
  await codex.getByText('Waiting for you', {exact: true}).waitFor();

  await dialog.getByRole('button', {name: 'Close', exact: true}).click();
  assert.equal(await page.getByLabel('Work draft').inputValue(), 'keep this draft');
  await open();
  await codex.getByText('ABCD-EFGH', {exact: true}).waitFor();
  assert.equal(state.starts, 1, 'reopening resumes the same sign-in');
  await mount();
  await open();
  await codex.getByText('ABCD-EFGH', {exact: true}).waitFor();
  assert.equal(state.starts, 1, 'a reload recovers the pending sign-in without starting another');

  state.completed = true;
  await codex.getByText('Connected', {exact: true}).waitFor();
  await codex.getByText(/me@example\.test/).waitFor();
  assert.equal(await codex.getByText('ABCD-EFGH', {exact: true}).count(), 0);

  await muse.getByRole('button', {name: 'Connect Muse', exact: true}).click();
  await muse.getByText('ABCD-EFGH', {exact: true}).waitFor();
  state.expired = true;
  await muse.getByText(/sign-in code expired/).waitFor();
  await muse.getByText('Needs attention', {exact: true}).waitFor();
  await muse.getByRole('button', {name: 'Try again', exact: true}).click();
  await muse.getByText('ABCD-EFGH', {exact: true}).waitFor();
  assert.equal(state.starts, 3);
  await muse.getByRole('button', {name: 'Cancel sign-in', exact: true}).click();
  await muse.getByRole('button', {name: 'Connect Muse', exact: true}).waitFor();
  assert.equal(state.cancels, 1);

  await mount();
  await open();
  await codex.getByText('Connected', {exact: true}).waitFor();
  assert.equal(state.starts, 3, 'a reload reads connection state without starting a sign-in');
});

test('the providers panel opens its single dialog from the document event on any page', async (t) => {
  const build = await Bun.build({entrypoints: ['frontend/spaces/soda-providers-entry.ts'], target: 'browser'});
  assert(build.success, build.logs.join('\n'));
  const {server, state} = identityServer(await build.outputs[0]!.text());
  t.after(() => server.stop(true));
  const page = await launch(t);
  await page.goto(server.url.toString());
  await page.evaluate(async () => {
    const path = '/widget.js';
    const module = await import(path);
    const root = document.createElement('div');
    root.hidden = true;
    document.body.append(root);
    module.mount(root, {
      contractVersion: 1,
      extensionId: 'soda',
      panelId: 'providers',
      appSubUrl: '',
      assetBase: '/assets/',
      sessionGeneration: 'generation-1',
      request: (path: string, init: RequestInit) => fetch('/api/' + path, init),
      stream: () => ({socket: undefined}),
      navigate: () => undefined,
    });
  });
  assert.equal(await page.evaluate(() => document.documentElement.hasAttribute('data-soda-providers')), true);
  await page.waitForFunction(() => document.querySelector('soda-providers') !== null);
  await page.evaluate(() => document.dispatchEvent(new CustomEvent('soda-open-providers')));
  const dialog = page.getByRole('dialog', {name: 'AI accounts', exact: true});
  await dialog.getByRole('button', {name: 'Connect Codex', exact: true}).waitFor();
  assert(await dialog.isVisible(), 'the dialog renders outside the hidden panel region');
  assert.equal(await page.locator('soda-providers').count(), 1);
  assert(state.reads >= 1);
});
