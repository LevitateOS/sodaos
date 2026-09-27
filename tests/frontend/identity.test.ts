import assert from 'node:assert/strict';
import test from 'node:test';
import {chromium} from 'playwright';
import {connectionView, enrollmentView, grantView, leaseView} from '../../frontend/spaces/soda-identity-response';

test('subscription metadata remains bound to owner and excludes credential fields', () => {
  const connection = {
    provider_id: 'codex',
    id: 'connection',
    owner_id: '9007199254740993',
    label: 'personal',
    email: 'tester@example.test',
    plan: 'plus',
    state: 'ready',
    refresh_token: 'private-fixture',
    credential: 'private-fixture',
  };
  const view = connectionView(connection, connection.owner_id);
  assert.equal(view.owner_id, connection.owner_id);
  assert.equal('refresh_token' in view, false);
  assert.equal('credential' in view, false);
  assert.throws(() => connectionView(connection, '2'));
  assert.throws(() => connectionView({...connection, owner_id: Number.MAX_SAFE_INTEGER + 1}, connection.owner_id));
});

test('device enrollment only links to the selected provider authentication origin', () => {
  const enrollment = {
    provider_id: 'codex',
    id: 'enrollment',
    verification_url: 'https://auth.openai.com/codex/device',
    user_code: 'ABCD-EFGH',
    state: 'pending',
  };
  assert.equal(enrollmentView(enrollment).verification_url, enrollment.verification_url);
  const muse = {
    ...enrollment,
    provider_id: 'muse',
    verification_url: 'https://auth.meta.com/oauth/device/?code=ABCD-EFGH',
  };
  assert.equal(enrollmentView(muse).verification_url, muse.verification_url);
  assert.throws(() => enrollmentView({...muse, verification_url: enrollment.verification_url}));
  for (const url of [
    'javascript:alert(1)',
    'https://other.example.test/',
    'https://auth.openai.com@other.example.test/',
    'http://auth.openai.com/',
    'https://tester:secret@auth.openai.com/',
  ]) {
    assert.throws(() => enrollmentView({...enrollment, verification_url: url}));
  }
});

test('named grants and leases preserve decimal identities without exposing delivery', () => {
  const grant = {
    id: 'grant',
    connection_id: 'connection',
    user_id: '2',
    project_id: 'p0123456789abcdef01234567',
    revoked: false,
  };
  assert.deepEqual(grantView(grant), grant);
  assert.throws(() => grantView({...grant, user_id: '02'}));
  const lease = {
    id: 'lease',
    connection_id: 'connection',
    actor_id: '2',
    project_id: grant.project_id,
    kind: 'terminal',
    execution_id: 'execution',
    credential: 'private-fixture',
    binding: {login: 'native-user'},
  };
  const view = leaseView(lease);
  assert.equal('credential' in view, false);
  assert.equal('binding' in view, false);
  assert.throws(() => leaseView({...lease, kind: 'unknown'}));
});

test(
  'identity widget keeps stable context and starts Codex with protected metadata-only request',
  {skip: process.env.SODA_IDENTITY_BROWSER !== '1'},
  async (t) => {
    const bundle = await Bun.build({entrypoints: ['frontend/spaces/soda-identity.ts'], target: 'browser'});
    assert.equal(bundle.success, true);
    const script = await bundle.outputs[0]!.text();
    const connection = {
      provider_id: 'codex',
      id: 'connection',
      owner_id: '1',
      label: 'personal',
      email: 'tester@example.test',
      plan: 'plus',
      state: 'ready',
    };
    const requests: string[] = [],
      launches: unknown[] = [];
    const server = Bun.serve({
      hostname: '127.0.0.1',
      port: 0,
      async fetch(request) {
        const url = new URL(request.url);
        if (url.pathname === '/widget.js') return new Response(script, {headers: {'Content-Type': 'text/javascript'}});
        if (url.pathname.startsWith('/-/soda/api/')) {
          requests.push(url.pathname);
          assert.equal(request.headers.get('X-Soda-Expected-User-ID'), '1');
          if (request.method === 'POST') {
            assert.equal(request.headers.get('X-CSRF-Token'), 'csrf');
            launches.push(await request.json());
            return Response.json({terminal_id: 'a'.repeat(32)});
          }
          if (url.pathname.endsWith('/connections')) return Response.json([connection]);
          return Response.json([]);
        }
        return new Response('<!doctype html><html><head><link rel="icon" href="data:,"></head><body></body></html>', {
          headers: {'Content-Type': 'text/html'},
        });
      },
    });
    t.after(() => server.stop(true));
    const browser = await chromium.launch({
      headless: true,
      chromiumSandbox: true,
      ...(process.env.CHROME ? {executablePath: process.env.CHROME} : {}),
    });
    t.after(() => browser.close());
    const page = await browser.newPage();
    await page.goto(server.url.toString());
    await page.evaluate(async () => {
      const modulePath = '/widget.js',
        module = await import(modulePath),
        widget = new module.SodaIdentity();
      document.body.append(widget);
      widget.context = {
        actor: '1',
        session: {user: {id: '1', login: 'soda-tester'}, csrf_token: 'csrf', forgejo_url: location.origin},
        project: 'p0123456789abcdef01234567',
      };
    });
    await page.getByRole('status').filter({hasText: 'Subscription status refreshed.'}).waitFor();
    const reads = requests.length;
    await page.evaluate(() => {
      const widget = document.querySelector('soda-identity') as HTMLElement & {
        context: unknown;
        updateComplete: Promise<boolean>;
      };
      widget.context = {
        actor: '1',
        session: {user: {id: '1', login: 'soda-tester'}, csrf_token: 'csrf', forgejo_url: location.origin},
        project: 'p0123456789abcdef01234567',
      };
      return widget.updateComplete;
    });
    assert.equal(requests.length, reads);
    await page.getByRole('button', {name: "Start Codex in this project's terminal"}).click();
    await page
      .getByRole('status')
      .filter({hasText: 'Codex started. Open project terminal and select Codex.'})
      .waitFor();
    assert.deepEqual(launches, [{connection_id: 'connection', cols: 80, rows: 24}]);
    await page.evaluate(() => window.dispatchEvent(new Event('soda-session-retired')));
    assert.equal(await page.getByRole('button', {name: "Start Codex in this project's terminal"}).isEnabled(), false);
  }
);
