import assert from 'node:assert/strict';
import {mkdir, readFile, writeFile} from 'node:fs/promises';
import {createHash, randomBytes} from 'node:crypto';
import {execFileSync} from 'node:child_process';
import {dirname, join, resolve} from 'node:path';
import {pathToFileURL} from 'node:url';
import {createServer as createHttpsServer} from 'node:https';
import {request as requestHttp, createServer as createHttpServer} from 'node:http';
import {chromium} from 'playwright';
// Real local runtime proof from the authored fixture; no Factory execution or release.
const sodaosRoot = resolve(import.meta.dir, '../../..');
const canlangRoot = resolve(process.env.CANLANG_ROOT ?? join(sodaosRoot, '../canlang'));
const producer = (file: string) => import(pathToFileURL(join(canlangRoot, 'packages', file)).href);
const {buildSodaBundle} = await producer('cloudflare/dist/deploy/soda.js');
const {provisionSodaStorage, startSodaRuntime} = await producer('cloudflare/dist/deploy/soda-run.js');
const {CONTRACTS_VERSION} = await producer('contracts/dist/index.js');
const {PINNED_COMPATIBILITY_DATE} = await producer('cloudflare/dist/cli/artifact-defaults.js');
const compiler = process.env.CANLANG_COMPILER ?? join(canlangRoot, 'compiler/target/debug/can');

process.umask(0o077);
const state = join(
  resolve(process.env.FOUNDER_REFERENCE_DIR ?? join(sodaosRoot, '../founder-demo-runs')),
  String(Date.now())
);
assert(state.startsWith('/home/'), 'Use a private output directory beneath /home.');
await mkdir(dirname(state), {recursive: true, mode: 0o700});
await mkdir(state, {mode: 0o700});
await mkdir(state + '/data', {mode: 0o700});
await mkdir(state + '/tmp', {mode: 0o700});
process.env.TMPDIR = state + '/tmp';
console.log(JSON.stringify({stage: 'starting', state}));
const cert = state + '/cert.pem',
  key = state + '/key.pem';
execFileSync(
  'openssl',
  [
    'req',
    '-x509',
    '-newkey',
    'rsa:2048',
    '-sha256',
    '-days',
    '1',
    '-nodes',
    '-subj',
    '/CN=soda.test',
    '-addext',
    'subjectAltName=DNS:forge.soda.test,DNS:equipment.soda.test',
    '-keyout',
    key,
    '-out',
    cert,
  ],
  {stdio: 'ignore'}
);
const browserBuild = await Bun.build({
  entrypoints: [join(sodaosRoot, 'frontend/spaces/sodaspaces-applications.ts')],
  target: 'browser',
  format: 'esm',
});
assert(browserBuild.success, browserBuild.logs.join('\n'));
const js = await browserBuild.outputs[0]!.text();
let backend = 0,
  revision = 1;
const gateway = createHttpsServer({cert: await readFile(cert), key: await readFile(key)}, (request, response) => {
  if (request.headers.host?.startsWith('forge.')) {
    if (request.url === '/app.js') {
      response.writeHead(200, {'content-type': 'text/javascript'});
      response.end(js);
      return;
    }
    if (request.url === '/api/applications') {
      response.writeHead(200, {'content-type': 'application/json', 'cache-control': 'no-store'});
      response.end(
        JSON.stringify({
          applications: [],
          launchable: [
            {
              application: {id: 'a' + '1'.repeat(24), revision, name: 'NorthstarEquipment', origin: appOrigin},
              running: true,
              ready: true,
              launch_only: true,
              environment: 'company',
              project: {repository: '7', name: 'northstar/equipment', path: '/northstar/equipment'},
            },
          ],
          complete: true,
          can_setup: false,
        })
      );
      return;
    }
    response.writeHead(200, {'content-type': 'text/html'});
    response.end(
      `<!doctype html><html><meta charset="utf-8"><title>Forge application boundary check</title><style>body{font:16px system-ui;margin:0}nav{padding:1rem}main[hidden]{display:none}</style><nav><button id="feed">Feed</button><button id="app">App</button></nav><aside hidden>Project conversation</aside><main></main><script type="module">import '/app.js';const app=document.createElement('soda-applications');app.active=true;app.expanded=true;app.repository='7';app.sessionKey='browser-owner';app.request=async(path,body)=>{if(body)throw Error('No lifecycle authority');const r=await fetch(path);if(!r.ok)throw Error('Read unavailable');return r.json()};document.querySelector('main').append(app);document.querySelector('#feed').onclick=()=>{document.querySelector('main').hidden=true;document.querySelector('aside').hidden=false};document.querySelector('#app').onclick=()=>{document.querySelector('main').hidden=false;document.querySelector('aside').hidden=true};window.app=app;</script></html>`
    );
    return;
  }
  if (!backend) {
    response.writeHead(503);
    response.end();
    return;
  }
  const outgoing = requestHttp(
    {hostname: '127.0.0.1', port: backend, path: request.url, method: request.method, headers: request.headers},
    (upstream) => {
      response.writeHead(upstream.statusCode!, upstream.headers);
      upstream.pipe(response);
    }
  );
  outgoing.on('error', () => {
    if (!response.headersSent) response.writeHead(502);
    response.end();
  });
  request.pipe(outgoing);
});
await new Promise<void>((resolve, reject) => {
  gateway.once('error', reject);
  gateway.listen(0, '127.0.0.1', resolve);
});
const port = (gateway.address() as {port: number}).port;
const forgeOrigin = `https://forge.soda.test:${port}`,
  appOrigin = `https://equipment.soda.test:${port}`;
const reservation = createHttpServer();
await new Promise<void>((resolve) => reservation.listen(0, '127.0.0.1', resolve));
backend = (reservation.address() as {port: number}).port;
await new Promise<void>((resolve) => reservation.close(() => resolve()));
const original = await readFile(join(import.meta.dir, 'Northstar.can'), 'utf8');
const rule = 'do create EquipmentRequest {title,reason,amount,needs_finance=false}';
assert.equal(original.split(rule).length, 2, 'The v1 submission route must be unambiguous.');
const artifactFiles: string[] = [];
for (const [version, source] of [
  ['v1', original],
  ['v2', original.replace(rule, 'do create EquipmentRequest {title,reason,amount,needs_finance=amount.minor>100000}')],
]) {
  const directory = join(state, version!);
  await mkdir(directory, {mode: 0o700});
  const sourceFile = join(directory, 'Northstar.can');
  await writeFile(sourceFile, source!);
  const bytes = execFileSync(
    compiler,
    ['compile', '--format=json', '--catalog=' + join(canlangRoot, 'packages/values/dist/catalog.json'), sourceFile],
    {cwd: directory, encoding: 'utf8', maxBuffer: 50_000_000, env: {...process.env, TMPDIR: state + '/tmp'}}
  );
  const file = join(directory, 'artifact.json');
  await writeFile(file, bytes);
  artifactFiles.push(file);
}
const artifactFile = artifactFiles[0]!;
const artifact = JSON.parse(await readFile(artifactFile, 'utf8'));
const resources = [
  {binding: 'DB', kind: 'd1', logicalName: 'northstar-identity'},
  {binding: 'STATE_DB', kind: 'd1', logicalName: 'northstar-state'},
];
const descriptor = {
  identity: {
    appName: 'NorthstarEquipment',
    sourceRevision: 'sha256:' + artifact.sources[0].sha256,
    languageVersion: artifact.language_version,
    compilerVersion: artifact.tool_version,
    contractsVersion: CONTRACTS_VERSION,
    artifactDigest: createHash('sha256').update(JSON.stringify(artifact)).digest('hex'),
  },
  requiredCapabilities: artifact.requires.map((r: any) => r.capability),
  resourceBindings: resources,
  secrets: [],
  schedules: [],
};
const environment = {
  environment: 'northstar-browser',
  resources: resources.map((requirement) => ({requirement, resourceId: requirement.logicalName})),
  secretsPresent: [],
  vars: {CAN_AUTH_ORIGIN: appOrigin},
};
console.log(JSON.stringify({stage: 'bundle'}));
const input = buildSodaBundle(artifact, descriptor, environment, PINNED_COMPATIBILITY_DATE);
const retainBundle = async (version: string, value: any) =>
  writeFile(
    join(state, version, 'bundle.json'),
    JSON.stringify({
      ...value,
      bundle: {
        ...value.bundle,
        binaries: Object.fromEntries(
          Object.entries(value.bundle.binaries).map(([name, bytes]) => [
            name,
            Buffer.from(bytes as Uint8Array).toString('base64'),
          ])
        ),
      },
    })
  );
await retainBundle('v1', input);
const actors = Object.fromEntries(
  ['maya', 'alex', 'jordan', 'robin', 'evan'].map((name) => [
    name,
    {email: name + '@example.invalid', password: randomBytes(32).toString('base64url'), timezone: 'UTC', userId: ''},
  ])
);
const credentials = actors.maya!;
console.log(JSON.stringify({stage: 'bootstrap'}));
const provision = await provisionSodaStorage(input, state + '/data');
let owner: any;
try {
  const response = await provision.dispatchUrl('https://private.invalid/bootstrap', {
    method: 'POST',
    headers: {'content-type': 'application/json'},
    body: JSON.stringify(credentials),
  });
  assert(response.ok, 'owner bootstrap');
  owner = await response.json();
  actors.maya!.userId = owner.userId;
  for (const name of ['alex', 'jordan', 'robin', 'evan']) {
    const actor = actors[name]!;
    const added = await provision.dispatchUrl('https://private.invalid/member', {
      method: 'POST',
      headers: {'content-type': 'application/json'},
      body: JSON.stringify({...actor, teamId: owner.teamId}),
    });
    assert(added.ok, 'ordinary member ' + name);
    actor.userId = ((await added.json()) as any).userId;
  }
} finally {
  await provision.dispose();
}
console.log(JSON.stringify({stage: 'runtime'}));
let runtime = await startSodaRuntime(input, {
  dataDir: state + '/data',
  ownerTeam: owner.teamId,
  host: '127.0.0.1',
  port: backend,
});
let browser: any;
try {
  browser = await chromium.launch({
    headless: true,
    chromiumSandbox: true,
    args: ['--host-resolver-rules=MAP *.soda.test 127.0.0.1', '--no-proxy-server'],
  });
  const opened: any = {};
  async function openActor(name: string) {
    const context = await browser.newContext({ignoreHTTPSErrors: true});
    const forge = await context.newPage();
    forge.on('pageerror', (error: any) => console.log(JSON.stringify({pageError: error.message.slice(0, 200)})));
    await forge.goto(forgeOrigin);
    // Forge shows only the app's status and a link; the app runs at its own
    // origin in its own tab, never inside Forge.
    const link = forge.getByRole('link', {name: 'Open app ↗', exact: true});
    await link.waitFor();
    assert.equal(await link.getAttribute('href'), appOrigin, 'Forge links the exact app origin');
    assert.equal(await link.getAttribute('target'), '_blank');
    assert.equal(await link.getAttribute('rel'), 'noopener noreferrer');
    assert.equal(await forge.locator('iframe').count(), 0, 'Forge does not embed the app');
    const [page] = await Promise.all([context.waitForEvent('page'), link.click()]);
    page.on('pageerror', (error: any) => console.log(JSON.stringify({pageError: error.message.slice(0, 200)})));
    await page.waitForLoadState();
    assert.equal(new URL(page.url()).origin, appOrigin, 'app opened at its own origin in a new page');
    page.on('response', async (response: any) => {
      const url = new URL(response.url());
      if (response.status() >= 400 || url.pathname.endsWith('request_equipment')) {
        try {
          console.log(
            JSON.stringify({
              httpActor: name,
              path: url.pathname + url.search,
              status: response.status(),
              body: (await response.text()).slice(0, 1500),
            })
          );
        } catch {}
      }
    });
    const frame = page,
      actor = actors[name]!;
    await frame.getByLabel('Email', {exact: true}).fill(actor.email);
    await frame.getByLabel('Password', {exact: true}).fill(actor.password);
    await frame.getByRole('button', {name: 'Sign in', exact: true}).click();
    const team = frame.getByLabel('Team', {exact: true});
    await team
      .or(frame.getByRole('heading', {name: 'Northstar equipment', exact: true}))
      .first()
      .waitFor();
    if (await team.isVisible().catch(() => false)) {
      await team.selectOption({index: 0});
      await frame.getByRole('button', {name: 'Select team', exact: true}).click();
    }
    await frame.getByRole('heading', {name: 'Northstar equipment', exact: true}).waitFor();
    return (opened[name] = {context, forge, page, frame});
  }
  function operationId() {
    const bytes = randomBytes(16);
    bytes.writeUIntBE(Date.now(), 0, 6);
    bytes[6] = (bytes[6]! & 15) | 112;
    bytes[8] = (bytes[8]! & 63) | 128;
    const h = bytes.toString('hex');
    return [h.slice(0, 8), h.slice(8, 12), h.slice(12, 16), h.slice(16, 20), h.slice(20)].join('-');
  }
  async function op(actor: any, name: string, inputs: any) {
    return actor.frame.locator('body').evaluate(
      async (_body: any, arg: any) => {
        const csrf = (document.querySelector('input[name="_csrf"]') as HTMLInputElement).value;
        const response = await fetch('/api/operations/' + arg.name, {
          method: 'POST',
          headers: {'content-type': 'application/json', 'x-csrf-token': csrf},
          body: JSON.stringify({operation_id: arg.id, inputs: arg.inputs}),
        });
        return {status: response.status, body: await response.json()};
      },
      {name, inputs, id: operationId()}
    );
  }
  async function read(actor: any) {
    return actor.frame.locator('body').evaluate(async () => {
      const csrf = (document.querySelector('input[name="_csrf"]') as HTMLInputElement).value;
      if (!(window as any).__integrationMcpGrant) {
        const grant = await fetch('/mcp/grants', {
          method: 'POST',
          headers: {'content-type': 'application/json', 'x-csrf-token': csrf},
          body: JSON.stringify({client_id: 'founder-browser-integration'}),
        });
        if (!grant.ok) throw Error('MCP grant ' + grant.status);
        (window as any).__integrationMcpGrant = (await grant.json()).token;
      }
      const response = await fetch('/mcp', {
        method: 'POST',
        headers: {
          'content-type': 'application/json',
          accept: 'application/json, text/event-stream',
          authorization: 'Bearer ' + (window as any).__integrationMcpGrant,
        },
        body: JSON.stringify({
          jsonrpc: '2.0',
          id: 1,
          method: 'tools/call',
          params: {name: 'NorthstarEquipment.EquipmentRequest.read', arguments: {}},
        }),
      });
      const body = await response.json();
      if (!response.ok || body.result?.isError || !Array.isArray(body.result?.structuredContent?.records))
        throw Error('MCP read ' + response.status + ' ' + JSON.stringify(body));
      return body.result.structuredContent.records;
    });
  }
  const maya = await openActor('maya');
  for (const [name, role] of [
    ['maya', 'operations'],
    ['jordan', 'manager'],
    ['robin', 'finance'],
  ] as [string, string][]) {
    const granted = await op(maya, 'system.team.role', {
      member: {id: actors[name]!.userId},
      role: 'NorthstarEquipment.' + role,
    });
    assert.equal(granted.status, 200, 'owner grants ' + role);
  }
  const alex = await openActor('alex'),
    jordan = await openActor('jordan'),
    robin = await openActor('robin'),
    evan = await openActor('evan');
  console.log(JSON.stringify({stage: 'real-accounts-and-owner-grants'}));
  async function submit(title: string, amount: string, person = alex) {
    const form = person.frame.locator('form[action="/api/operations/NorthstarEquipment.request_equipment"]');
    await form.locator('[name="inputs[title]"]').fill(title);
    await form.locator('[name="inputs[reason]"]').fill('Equipment needed for work');
    await form.locator('[name="inputs[amount]"]').fill(amount);
    await form.locator('[name="inputs[amount__currency]"]').selectOption('EUR');
    await form.getByRole('button', {name: 'Submit', exact: true}).click();
    try {
      await person.frame.getByText(title, {exact: false}).first().waitFor({timeout: 12000});
    } catch (error) {
      await person.page.screenshot({path: state + '/submission-error.png', fullPage: true});
      console.log(
        JSON.stringify({
          formError: await form.innerText(),
          alerts: await person.frame.locator('[role=alert]').allTextContents(),
          read: await read(person),
        })
      );
      throw error;
    }
    const row = (await read(person)).find((r: any) => r.data.title === title);
    assert(row, 'saved ' + title);
    return row;
  }
  const reference = (r: any) => ({id: r.id, version: String(r.version)});
  const rowAfter = async (actor: any, id: string) => (await read(actor)).find((r: any) => r.id === id);
  let monitor = await submit('Retained monitor', '420');
  assert.equal(monitor.data.amount.minor, '42000');
  const pending = await submit('Pending pre-change workstation', '1500');
  assert.equal(pending.data.needs_finance, false);
  assert.equal((await read(evan)).length, 0, 'unrelated employee cannot read requests');
  const self = await op(alex, 'NorthstarEquipment.approve', {request: reference(monitor)});
  assert.notEqual(self.status, 200, 'requester cannot approve');
  assert.equal((await op(jordan, 'NorthstarEquipment.approve', {request: reference(monitor)})).status, 200);
  monitor = await rowAfter(maya, monitor.id);
  assert.equal(monitor.data.stage, 'fulfillment');
  assert.equal((await op(maya, 'NorthstarEquipment.deliver', {request: reference(monitor)})).status, 200);
  monitor = await rowAfter(alex, monitor.id);
  assert.equal(monitor.data.stage, 'delivered');
  await alex.forge.click('#feed');
  await alex.forge.click('#app');
  await alex.forge.getByRole('link', {name: 'Open app ↗', exact: true}).waitFor();
  await alex.frame.getByText('Retained monitor', {exact: false}).first().waitFor();
  await alex.page.reload();
  await alex.frame.getByText('Retained monitor', {exact: false}).first().waitFor();
  const before = await rowAfter(alex, monitor.id);
  assert.deepEqual(before, monitor);
  console.log(JSON.stringify({stage: 'v1-persisted-reload', monitor_id: monitor.id, pending_id: pending.id}));
  const retained = {monitor, pending};
  const v2Artifact = JSON.parse(await readFile(artifactFiles[1]!, 'utf8'));
  const v2Descriptor = {
    ...descriptor,
    identity: {
      ...descriptor.identity,
      sourceRevision: 'sha256:' + v2Artifact.sources[0].sha256,
      artifactDigest: createHash('sha256').update(JSON.stringify(v2Artifact)).digest('hex'),
    },
  };
  const v2Input = buildSodaBundle(v2Artifact, v2Descriptor, environment, PINNED_COMPATIBILITY_DATE);
  await retainBundle('v2', v2Input);
  await runtime.dispose();
  runtime = await startSodaRuntime(v2Input, {
    dataDir: state + '/data',
    ownerTeam: owner.teamId,
    host: '127.0.0.1',
    port: backend,
  });
  revision = 2;
  for (const person of Object.values(opened) as any[]) {
    await person.page.reload();
    await person.frame.getByRole('heading', {name: 'Northstar equipment', exact: true}).waitFor();
  }
  assert.deepEqual(await rowAfter(alex, monitor.id), retained.monitor, 'delivered record retained unchanged');
  assert.deepEqual(await rowAfter(alex, pending.id), retained.pending, 'pending original routing retained unchanged');
  assert.equal((await read(evan)).length, 0, 'unrelated employee remains isolated after update');
  let thousand = await submit('Threshold exactly EUR1000', '1000');
  assert.equal(thousand.data.amount.minor, '100000');
  assert.equal(thousand.data.needs_finance, false);
  let above = await submit('Threshold above EUR1000', '1001');
  assert.equal(above.data.amount.minor, '100100');
  assert.equal(above.data.needs_finance, true);
  const own = await submit('Manager own request', '250', jordan);
  assert.equal(
    (await op(jordan, 'NorthstarEquipment.approve', {request: reference(own)})).status,
    422,
    'manager cannot approve own request'
  );
  assert.equal(
    (await op(jordan, 'NorthstarEquipment.approve', {request: reference(above)})).status,
    422,
    'manager cannot skip finance route'
  );
  assert.equal(
    (await op(maya, 'NorthstarEquipment.deliver', {request: reference(above)})).status,
    422,
    'operations cannot bypass approvals'
  );
  assert.equal((await op(jordan, 'NorthstarEquipment.approve', {request: reference(thousand)})).status, 200);
  thousand = await rowAfter(maya, thousand.id);
  assert.equal(thousand.data.stage, 'fulfillment');
  assert.equal((await op(jordan, 'NorthstarEquipment.approve_finance_route', {request: reference(above)})).status, 200);
  above = await rowAfter(robin, above.id);
  assert.equal(above.data.stage, 'waiting_finance');
  assert.equal(
    (await op(maya, 'NorthstarEquipment.approve_finance', {request: reference(above)})).status,
    403,
    'operations cannot impersonate finance'
  );
  assert.equal((await op(robin, 'NorthstarEquipment.approve_finance', {request: reference(above)})).status, 200);
  above = await rowAfter(maya, above.id);
  assert.equal(above.data.stage, 'fulfillment');
  assert.equal((await op(maya, 'NorthstarEquipment.deliver', {request: reference(above)})).status, 200);
  above = await rowAfter(alex, above.id);
  assert.equal(above.data.stage, 'delivered');
  await alex.page.reload();
  await alex.frame.getByText('Threshold above EUR1000', {exact: false}).first().waitFor();
  assert.deepEqual(
    await rowAfter(alex, monitor.id),
    retained.monitor,
    'original delivered record retained after new transactions'
  );
  assert.equal((await rowAfter(alex, pending.id)).data.needs_finance, false, 'pending1500 keeps original routing');
  await maya.page.reload();
  await maya.frame.getByRole('heading', {name: 'Northstar equipment', exact: true}).waitFor();
  const histories = maya.frame
    .locator('details')
    .filter({has: maya.frame.locator('summary').getByText('History', {exact: true})});
  const historyCount = await histories.count();
  assert(historyCount >= 5, 'actual history disclosures rendered');
  const historyText: string[] = await histories.allTextContents();
  assert(
    historyText.some(
      (text) =>
        text.includes('NorthstarEquipment.request_equipment') &&
        text.includes('NorthstarEquipment.approve') &&
        text.includes('NorthstarEquipment.deliver')
    ),
    'retained create/approval/delivery audit rendered'
  );
  assert.equal(await maya.frame.locator('[data-can-history=unavailable]').count(), 0, 'operator history is available');
  console.log(
    JSON.stringify({
      stage: 'v2-retained-data-and-threshold-guards',
      monitor_id: monitor.id,
      pending_id: pending.id,
      exact1000: thousand.id,
      above1000: above.id,
      history_regions: historyCount,
    })
  );
  await alex.page.screenshot({path: state + '/app-own-tab.png', fullPage: true});
  await writeFile(
    state + '/result.json',
    JSON.stringify({
      inside_forge: false,
      own_origin_tab: true,
      real_canlang_login: true,
      reference_fixture: true,
      company_data: state + '/data',
      monitor_id: monitor.id,
      pending_id: pending.id,
      exact1000: thousand.id,
      above1000: above.id,
      retained_records: true,
      finance_guard: true,
      self_approval_denied: true,
      unrelated_employee_isolated: true,
      history_regions: historyCount,
    })
  );
} finally {
  await browser?.close();
  await runtime.dispose();
  await new Promise<void>((resolve) => gateway.close(() => resolve()));
}
