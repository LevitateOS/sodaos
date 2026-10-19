import assert from 'node:assert/strict';
import test from 'node:test';
import {chromium} from 'playwright';

test('Providers starts device login, resumes its dialog, follows completion and retries expiry', async t => {
  const build = await Bun.build({entrypoints: ['frontend/spaces/soda-providers.ts'], target: 'browser'});
  assert(build.success, build.logs.join('\n'));
  const script = await build.outputs[0]!.text();
  let completed = false, expired = false, starts = 0, cancels = 0;
  const attempts = new Map<string, Record<string, string>>();
  const server = Bun.serve({hostname: '127.0.0.1', port: 0, async fetch(request) {
    const url = new URL(request.url);
    if (url.pathname === '/widget.js') return new Response(script, {headers: {'Content-Type': 'text/javascript'}});
    if (url.pathname === '/api/identity/connections') return Response.json(completed ? [{
      provider_id: 'codex', id: 'connected', owner_id: '1', label: 'Codex', email: '', plan: 'pro', state: 'ready',
    }] : []);
    if (url.pathname === '/api/identity/enrollments' && request.method === 'POST') {
      const body = await request.json() as {provider_id: string; label: string; confirm_credential_exposure: boolean};
      assert.equal(body.confirm_credential_exposure, true);
      assert(body.label);
      starts++;
      expired = false;
      const attempt = {provider_id: body.provider_id, id: body.provider_id, state: 'pending',
        verification_url: body.provider_id === 'codex' ? 'https://auth.openai.com/codex/device' : 'https://auth.meta.com/oauth/device/', user_code: 'ABCD-EFGH'};
      attempts.set(body.provider_id, attempt);
      return Response.json(attempt);
    }
    if (url.pathname === '/api/identity/enrollments') return Response.json([...attempts.values()]);
    if (url.pathname.endsWith('/cancel')) { cancels++; return Response.json({}); }
    if (url.pathname.startsWith('/api/identity/enrollments/')) {
      const provider = url.pathname.split('/').at(-1)!;
      return Response.json({provider_id: provider, id: provider, state: provider === 'codex' && completed ? 'completed' : expired ? 'expired' : 'pending',
        verification_url: provider === 'codex' ? 'https://auth.openai.com/codex/device' : 'https://auth.meta.com/oauth/device/', user_code: 'ABCD-EFGH'});
    }
    return new Response('<!doctype html><html><head><style>dialog { display:flex; }</style></head><body><textarea aria-label="Work draft">keep this draft</textarea></body></html>', {headers: {'Content-Type': 'text/html'}});
  }});
  t.after(() => server.stop(true));
  const browser = await chromium.launch({headless: true, chromiumSandbox: true,
    ...(process.env.CHROME ? {executablePath: process.env.CHROME} : {})});
  t.after(() => browser.close());
  const page = await browser.newPage();
  const mount = async () => {
    await page.goto(server.url.toString());
    await page.evaluate(async () => {
      const path = '/widget.js';
      const module = await import(path);
      const widget = new module.SodaProviders();
      document.body.append(widget);
      widget.transport = {request: (path: string, init: RequestInit) => fetch('/api/' + path, init)};
    });
    await page.getByRole('button', {name: 'Providers', exact: true}).click();
  };
  await mount();
  const dialog = page.getByRole('dialog', {name: 'Providers', exact: true});
  assert.equal(await dialog.evaluate(node => getComputedStyle(node).display), 'block', 'Forge dialog styles must not squeeze the content into columns');
  await dialog.getByRole('button', {name: 'Codex: Not connected', exact: true}).click();
  await dialog.getByText('ABCD-EFGH', {exact: true}).waitFor();
  assert.equal(await dialog.getByRole('link', {name: 'Open sign-in', exact: true}).getAttribute('href'), 'https://auth.openai.com/codex/device');
  await dialog.getByRole('button', {name: 'Close', exact: true}).click();
  assert.equal(await page.getByLabel('Work draft').inputValue(), 'keep this draft');
  await page.getByRole('button', {name: 'Providers', exact: true}).click();
  await dialog.getByText('ABCD-EFGH', {exact: true}).waitFor();
  assert.equal(starts, 1, 'reopening must resume the same login');
  await mount();
  await dialog.getByText('ABCD-EFGH', {exact: true}).waitFor();
  assert.equal(starts, 1, 'a full page reload must recover pending sign-in without starting another');
  completed = true;
  await dialog.getByRole('button', {name: 'Codex: Connected', exact: true}).waitFor();
  assert.equal(await dialog.getByText('ABCD-EFGH', {exact: true}).count(), 0);
  await dialog.getByRole('button', {name: 'Muse: Not connected', exact: true}).click();
  expired = true;
  await dialog.getByText('Sign-in expired.', {exact: true}).waitFor();
  await dialog.getByRole('button', {name: 'Try again', exact: true}).click();
  await dialog.getByText('ABCD-EFGH', {exact: true}).waitFor();
  assert.equal(starts, 3);
  await dialog.getByRole('button', {name: 'Cancel sign-in', exact: true}).click();
  await dialog.getByText('Sign-in cancelled.', {exact: true}).waitFor();
  assert.equal(cancels, 1);
  await mount();
  await page.getByRole('button', {name: 'Codex: Connected', exact: true}).waitFor();
  assert.equal(starts, 3, 'reload must read connection state without starting login');
});
